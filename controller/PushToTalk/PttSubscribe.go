package PushToTalk

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sunvc/NoLets/common"
)

// PttSubscribe 建立 SSE 长连接,向客户端持续推送 join/leave/update 事件。
// - 请求体沿用 JoinParams (与 /ptt/connect 相同的签名逻辑)
// - 首帧回一个 snapshot,包含订阅频道当前所有在线用户
// - 之后由 SyncChannels / BroadcastUpdate 触发的事件驱动
// - 心跳: 每 15s 发一次 comment 行, 保持中间层不断连
// - 断开:客户端关闭或 60s 无写入错误即回收
func PttSubscribe(c *gin.Context) {

	var req JoinParams
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(200, common.Failed(c, -1, "%v", err.Error()))
		return
	}
	if req.ID == "" || len(req.Channels) == 0 {
		c.JSON(200, common.Failed(c, -1, "id/channels required"))
		return
	}

	rc := http.NewResponseController(c.Writer)
	deadlineCleared := true
	if err := rc.SetWriteDeadline(time.Time{}); err != nil {
		deadlineCleared = false
	}
	_ = rc.SetReadDeadline(time.Time{})
	pokeDeadline := func() {
		if deadlineCleared {
			return
		}
		_ = rc.SetWriteDeadline(time.Now().Add(1 * time.Hour))
	}

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache, no-transform")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.WriteHeader(http.StatusOK)

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		return
	}

	type subInfo struct {
		channel string
		ch      <-chan SubEvent
		cancel  func()
	}
	subs := make([]subInfo, 0, len(req.Channels))
	for _, ch := range req.Channels {
		evtCh, cancel := Subscribe(req.ID, ch)
		subs = append(subs, subInfo{channel: ch, ch: evtCh, cancel: cancel})
	}
	defer func() {
		for _, s := range subs {
			s.cancel()
		}
	}()

	nowMs := time.Now().UnixMilli()

	// 2) 每个订阅频道回一个 snapshot
	pokeDeadline()
	ChannelLock.RLock()
	for _, chName := range req.Channels {
		var users []PttUserResp
		if ch, ok := Channels[chName]; ok {
			users = ch.UserListResp()
		}
		snap := SubEvent{
			Event:   EventSnapshot,
			Channel: chName,
			Users:   users,
			Ts:      nowMs,
		}
		if !writeSSE(c.Writer, EventSnapshot, MarshalEvent(snap)) {
			ChannelLock.RUnlock()
			return
		}
	}
	ChannelLock.RUnlock()
	flusher.Flush()

	// 3) 事件循环: fan-in 所有订阅通道 + 心跳 + 断开
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	notify := c.Request.Context().Done()

	// merged fan-in 用一个 goroutine 把多个 sub chan 汇到一个
	merged := make(chan SubEvent, len(subs)*8+8)
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
			return
		case <-ticker.C:
			pokeDeadline()
			if _, err := io.WriteString(c.Writer, ": keepalive\n\n"); err != nil {
				return
			}
			flusher.Flush()
		case evt := <-merged:
			pokeDeadline()
			if !writeSSE(c.Writer, evt.Event, MarshalEvent(evt)) {
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
