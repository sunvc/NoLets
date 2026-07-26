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
			channel, chOK := Channels[msg.Channel]
			if !chOK {
				ChannelLock.RUnlock()
				fmt.Printf("[PttConsumer-%d] 频道不存在: %s (sender=%s)\n", workerID, msg.Channel, msg.Sender)
				return
			}
			users := channel.UserList()
			ChannelLock.RUnlock()

			// 诊断日志：打印所有在线用户 ID 与 sender 的比对
			ids := make([]string, 0, len(users))
			for _, u := range users {
				ids = append(ids, fmt.Sprintf("%s(tok=%d)", u.ID, len(u.Token)))
			}
			fmt.Printf("[PttConsumer-%d] channel=%s sender=%q users=%v\n",
				workerID, msg.Channel, msg.Sender, ids)

			// 极速拷贝目标用户列表，排除发送者本人，并迅速释放锁
			targets := make([]PttUser, 0, len(users))
			for _, user := range users {
				if user.ID != msg.Sender && user.Token != "" {

					targets = append(targets, user)
				} else if user.ID == msg.Sender {
					fmt.Printf("[PttConsumer-%d] ⚠️ 排除自己: user.ID=%q sender=%q\n",
						workerID, user.ID, msg.Sender)
				}
			}

			if len(targets) == 0 {
				return
			}

			// 拆解成独立的原子任务，塞入第二级队列
			for _, token := range targets {
				task := PushTask{
					Token: token.Token,
					Url:   msg.Host + "/ptt/voice/" + msg.FileName,
				}

				select {
				case PushTaskQueue <- task: // 成功入队
				default:
					// 极端洪峰下，任务满载自动丢弃，保护 2 核机器不崩溃
					fmt.Printf("[Drop] 推送任务队列已满，丢弃发往用户 %s 的分片\n", token)
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
			if err := push.PttPush(task.Url, task.Token); err != nil {
				fmt.Println(fmt.Printf("[PushWorker-%d] 苹果推送失败 (Token: %s): %v\n", workerID, task.Token[:10], err))
			}
		}()
	}
}

// InitPttSystem 服务启动
func InitPttSystem() {

	for i := 0; i < 2; i++ {
		go startPttConsumer(i)
	}

	for i := 0; i < 24; i++ {
		go startPushTaskWorker(i)
	}

	StartFileCleanerService(common.BaseDir("voices"), 5*time.Minute)
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
