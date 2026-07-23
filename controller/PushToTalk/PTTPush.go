package PushToTalk

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/sunvc/NoLets/common"
	"github.com/sunvc/NoLets/push"
)

// 第一级消费者：从 msgQueue 读取，拆解出目标人群，扇出（Fan-Out）到第二级
func startPttConsumer(workerID int) {
	for msg := range MsgQueue {
		// 匿名函数加锁隔离与异常恢复
		func() {
			defer func() {
				if r := recover(); r != nil {
					fmt.Printf("[CRITICAL] [PttConsumer-%d] 捕获异常: %v, 消息: %+v\n", workerID, r, msg)
				}
			}()

			ChannelLock.RLock()
			ch, ok := Channels[msg.Channel]
			var users []PttUser
			if ok {
				users = ch.UserList()
			}
			ChannelLock.RUnlock()
			if !ok {
				return
			}

			// 极速拷贝目标用户列表，排除发送者本人，并迅速释放锁
			targets := make([]PttUser, 0, len(users))
			for _, user := range users {
				if user.ID != msg.Sender && user.Token != "" {

					targets = append(targets, user)
				}
			}

			if len(targets) == 0 {
				return
			}

			// 拆解成独立的原子任务，塞入第二级队列
			for _, token := range targets {
				task := PushTask{
					Name:  token.Name,
					Token: token.Token,
					Url:   msg.Host + "/ptt/voice/" + msg.FileName,
				}

				select {
				case PushTaskQueue <- task: // 成功入队
				default:
					// 极端洪峰下，任务满载自动丢弃，保护 2 核机器不崩溃
					fmt.Printf("[Drop] 推送任务队列已满，丢弃发往用户 %s 的分片\n", token.ID)
				}
			}
		}()
	}
}

// 第二级消费者：长连接协程，从 pushTaskQueue 取任务，调用 APNs 真正发送
func startPushTaskWorker(workerID int) {
	for task := range PushTaskQueue {
		func() {
			defer func() {
				if r := recover(); r != nil {
					fmt.Printf("[Error] [PushWorker-%d] 推送异常: %v\n", workerID, r)
				}
			}()

			// 调用底层的推送方法
			if err := push.PttPush(task.Url, task.Name, task.Token); err != nil {
				fmt.Println(fmt.Printf("[PushWorker-%d] 苹果推送失败 (Token: %s): %v\n", workerID, task.Token[:10], err))
			}
		}()
	}
}

// startPushWakeupWorker drains PushWakeupTaskQueue. Each task is one APNs
// wake-up push (client uses the payload to build a WebSocket subscribe).
// Uses the same pool of APNs clients as the legacy path.
func startPushWakeupWorker(workerID int) {
	for task := range PushWakeupTaskQueue {
		func() {
			defer func() {
				if r := recover(); r != nil {
					fmt.Printf("[Error] [WakeupWorker-%d] 推送异常: %v\n", workerID, r)
				}
			}()

			err := push.PttPushWakeup(push.PttWakeupParams{
				Channel:   task.Channel,
				SessionID: task.SessionID,
				Host:      task.Host,
				From:      task.From,
				FromName:  task.FromName,
			}, task.Token)
			if err != nil && len(task.Token) >= 10 {
				fmt.Printf("[WakeupWorker-%d] 唤醒推送失败 (Token: %s...): %v\n", workerID, task.Token[:10], err)
			} else if err != nil {
				fmt.Printf("[WakeupWorker-%d] 唤醒推送失败: %v\n", workerID, err)
			}
		}()
	}
}

// firePushForSession is called once per session, on the first AUDIO frame.
// It enqueues one PushToTalk wake-up notification for EVERY registered member
// of the channel except the speaker — even if that member still appears to
// have a live WS connection. iOS may suspend networking before the server
// notices the stale socket, so using WS presence to suppress APNs creates a
// race where sleeping devices are never awakened. Duplicate wake-ups are safe:
// the client's wakeupSocket / SUBSCRIBE path is idempotent.
func firePushForSession(bucket *SessionBucket) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[ptt-ws] firePushForSession panic: %v", r)
		}
	}()

	// Full channel membership from the persistent PTT registry. Copy the user
	// IDs under the read lock, then release it before touching APNs.
	ChannelLock.RLock()
	ch, ok := Channels[bucket.Channel]
	var memberIDs []string
	if ok {
		memberIDs = make([]string, 0, len(ch.UserIDs))
		for uid := range ch.UserIDs {
			memberIDs = append(memberIDs, uid)
		}
	}
	ChannelLock.RUnlock()
	if !ok {
		return
	}

	enqueued := 0
	for _, uid := range memberIDs {
		if uid == bucket.From {
			continue
		}
		val, ok := GlobalUsers.Load(uid)
		if !ok {
			continue
		}
		user, ok := val.(PttUser)
		if !ok || user.Token == "" {
			continue
		}
		task := PushWakeupTask{
			Token:     user.Token,
			Name:      user.Name,
			Channel:   bucket.Channel,
			SessionID: bucket.ID,
			Host:      user.Host,
			From:      bucket.From,
			FromName:  bucket.FromName,
		}
		select {
		case PushWakeupTaskQueue <- task:
			enqueued++
		default:
			log.Printf("[ptt-ws] wakeup queue full, dropping recipient user=%s", uid)
		}
	}
	log.Printf("[ptt-ws] APNs fanout session=%s channel=%s members=%d recipients=%d",
		bucket.ID, bucket.Channel, len(memberIDs), enqueued)
}

// InitPttSystem 服务启动
func InitPttSystem() {

	for i := 0; i < 2; i++ {
		go startPttConsumer(i)
	}

	for i := 0; i < 24; i++ {
		go startPushTaskWorker(i)
	}

	// A smaller pool is enough for wake-up pushes — each session triggers a
	// single fan-out and recipients are typically < 10.
	for i := 0; i < 8; i++ {
		go startPushWakeupWorker(i)
	}

	StartSessionGCService()
	StartFileCleanerService(common.BaseDir("voices"), 5*time.Minute)
}

// StartSessionGCService reaps ended SessionBuckets whose ring-buffer replay
// window has passed, plus any still-live sessions that overshot the configured
// max hold (protection against a stuck sender who never sends END).
func StartSessionGCService() {
	interval := common.LocalConfig.System.WSSessionGCInterval
	if interval <= 0 {
		interval = 5 * time.Second
	}
	ringTTL := common.LocalConfig.System.WSRingBufferTTL
	if ringTTL <= 0 {
		ringTTL = 5 * time.Second
	}
	maxHold := common.LocalConfig.System.WSSessionMaxHold
	if maxHold <= 0 {
		maxHold = 90 * time.Second
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[ptt-ws] session GC crashed: %v — restarting in 10s", r)
				time.Sleep(10 * time.Second)
				StartSessionGCService()
			}
		}()

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			sweepSessions(ringTTL, maxHold)
		}
	}()
}

// sweepSessions is the single-tick body of the GC. Split out so future tests
// can drive it directly without waiting on a ticker.
func sweepSessions(ringTTL, maxHold time.Duration) {
	now := time.Now().UnixMilli()
	ringMs := int64(ringTTL / time.Millisecond)
	holdMs := int64(maxHold / time.Millisecond)

	ActiveSessions.Range(func(_, v any) bool {
		bucket, ok := v.(*SessionBucket)
		if !ok {
			return true
		}
		// Force-end sessions that have been running longer than max hold —
		// prevents a wedged sender from pinning the bucket forever.
		if !bucket.Ended() && now-bucket.StartedAt > holdMs {
			bucket.MarkEnded()
			log.Printf("[ptt-ws] session id=%s force-ended after %dms", bucket.ID, now-bucket.StartedAt)
		}
		// Reap buckets whose replay window has fully lapsed.
		if bucket.Ended() && now-bucket.LastActive() > ringMs {
			dropActiveSession(bucket)
		}
		return true
	})
}

// StartFileCleanerService ---
func StartFileCleanerService(dirPath string, interval time.Duration) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[后台清理服务崩溃] 正在尝试重启服务。错误原因: %v\n", r)
				time.Sleep(10 * time.Second)
				StartFileCleanerService(dirPath, interval)
			}
		}()

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		log.Printf("[后台清理服务] 已成功启动！监控目录: %s，执行间隔: %v\n", dirPath, interval)

		executeClean(dirPath)
		for range ticker.C {
			executeClean(dirPath)
		}
	}()
}

// executeClean handler
func executeClean(dirPath string) {
	expirationTime := time.Now().Add(-5 * time.Minute)
	counter := 0

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if !info.IsDir() && info.ModTime().Before(expirationTime) {
			if err := os.Remove(path); err == nil {
				counter++
			} else {
				log.Printf("[后台清理服务] 尝试删除文件失败: %s, 错误: %v\n", path, err)
			}
		}
		return nil
	})

	if err != nil {
		log.Printf("[后台清理服务] 遍历目录时发生致命错误: %v\n", err)
	} else if counter > 0 {
		log.Printf("[后台清理服务] 定时清理完成，本次共成功释放 %d 个过期文件\n", counter)
	}
}
