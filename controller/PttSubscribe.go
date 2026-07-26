package controller

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sunvc/NoLets/common"
	PushToTalk2 "github.com/sunvc/NoLets/controller/PushToTalk"
)

// PttSubscribe 建立 SSE 长连接,向客户端持续推送 join/leave/update 事件。
// - 请求体沿用 JoinParams (与 /ptt/connect 相同的签名逻辑)
// - 首帧回一个 snapshot,包含订阅频道当前所有在线用户
// - 之后由 SyncChannels / BroadcastUpdate 触发的事件驱动
// - 心跳: 每 15s 发一次 comment 行, 保持中间层不断连
// - 断开:客户端关闭或 60s 无写入错误即回收
func PttSubscribe(c *gin.Context) {

	var req PushToTalk2.JoinParams
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(200, common.Failed(c, -1, err.Error(), nil))
		return
	}
	if req.ID == "" || len(req.Channels) == 0 {
		c.JSON(200, common.Failed(c, -1, "id/channels required", nil))
		return
	}

	startedAt := time.Now()
	fmt.Printf("[SSE] open id=%s channels=%v\n", req.ID, req.Channels)
	defer func() {
		fmt.Printf("[SSE] close id=%s elapsed=%s\n", req.ID, time.Since(startedAt))
	}()

	// 尝试在 handler 起点就清 deadline —— 一定要在 WriteHeader 之前
	rc := http.NewResponseController(c.Writer)
	deadlineCleared := true
	if err := rc.SetWriteDeadline(time.Time{}); err != nil {
		fmt.Printf("[SSE] SetWriteDeadline err: %v\n", err)
		deadlineCleared = false
	}
	if err := rc.SetReadDeadline(time.Time{}); err != nil {
		fmt.Printf("[SSE] SetReadDeadline err: %v\n", err)
	}
	// 若清除失败,回退到"每次写前推 1 小时"的滑动窗口方式
	pokeDeadline := func() {
		if deadlineCleared {
			return
		}
		_ = rc.SetWriteDeadline(time.Now().Add(1 * time.Hour))
	}

	// SSE 通用 header
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache, no-transform")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.WriteHeader(http.StatusOK)

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		fmt.Println("[SSE] Flusher unavailable")
		return
	}

	// 1) 注册用户到频道并把订阅通道拿到手
	PushToTalk2.SyncChannels(req.PttUser, req.Channels)

	// 每个频道单独注册一个订阅通道,合并到一个 select
	type subInfo struct {
		channel string
		ch      <-chan PushToTalk2.SubEvent
		cancel  func()
	}
	subs := make([]subInfo, 0, len(req.Channels))
	for _, ch := range req.Channels {
		evtCh, cancel := PushToTalk2.Subscribe(req.ID, ch)
		subs = append(subs, subInfo{channel: ch, ch: evtCh, cancel: cancel})
	}
	// 客户端断开或函数返回时清理
	defer func() {
		for _, s := range subs {
			s.cancel()
		}
		// 从 SyncChannels 视角移除用户
		PushToTalk2.SyncChannels(req.PttUser, []string{})
	}()

	nowMs := time.Now().UnixMilli()

	// 2) 每个订阅频道回一个 snapshot
	pokeDeadline()
	PushToTalk2.ChannelLock.RLock()
	for _, chName := range req.Channels {
		users := []PushToTalk2.PttUserResp{}
		if ch, ok := PushToTalk2.Channels[chName]; ok {
			users = ch.UserListResp()
		}
		snap := PushToTalk2.SubEvent{
			Event:   PushToTalk2.EventSnapshot,
			Channel: chName,
			Users:   users,
			Ts:      nowMs,
		}
		if !writeSSE(c.Writer, PushToTalk2.EventSnapshot, PushToTalk2.MarshalEvent(snap)) {
			PushToTalk2.ChannelLock.RUnlock()
			return
		}
	}
	PushToTalk2.ChannelLock.RUnlock()
	flusher.Flush()

	// 3) 事件循环: fan-in 所有订阅通道 + 心跳 + 断开
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	notify := c.Request.Context().Done()

	// merged fan-in 用一个 goroutine 把多个 sub chan 汇到一个
	merged := make(chan PushToTalk2.SubEvent, len(subs)*8+8)
	stopFanIn := make(chan struct{})
	for _, s := range subs {
		go func(s subInfo) {
			for {
				select {
				case <-stopFanIn:
					return
				case evt, ok := <-s.ch:
					if !ok {
						return
					}
					select {
					case merged <- evt:
					case <-stopFanIn:
						return
					}
				}
			}
		}(s)
	}
	defer close(stopFanIn)

	for {
		select {
		case <-notify:
			fmt.Printf("[SSE] client-cancel id=%s\n", req.ID)
			return
		case <-ticker.C:
			// 心跳: 一条 comment line (': keepalive'), 客户端可忽略
			pokeDeadline()
			if _, err := io.WriteString(c.Writer, ": keepalive\n\n"); err != nil {
				fmt.Printf("[SSE] keepalive write err id=%s: %v\n", req.ID, err)
				return
			}
			flusher.Flush()
		case evt := <-merged:
			pokeDeadline()
			if !writeSSE(c.Writer, evt.Event, PushToTalk2.MarshalEvent(evt)) {
				fmt.Printf("[SSE] event write err id=%s event=%s\n", req.ID, evt.Event)
				return
			}
			flusher.Flush()
		}
	}
}

// writeSSE 按 event-stream 协议写一帧。写失败返回 false 让上层退出。
func writeSSE(w http.ResponseWriter, event string, data []byte) bool {
	if event != "" {
		if _, err := fmt.Fprintf(w, "event: %s\n", event); err != nil {
			return false
		}
	}
	if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
		return false
	}
	return true
}
