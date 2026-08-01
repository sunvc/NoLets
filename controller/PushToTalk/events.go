package PushToTalk

import (
	"encoding/json"
	"sync"
)

// SSE 事件类型
const (
	EventSnapshot = "snapshot" // 订阅时下发的全量快照
	EventJoin     = "join"     // 用户加入频道
	EventLeave    = "leave"    // 用户离开频道
	EventUpdate   = "update"   // 用户位置更新
	EventPing     = "ping"     // 心跳,避免中间层判定连接死亡
)

// SubEvent 单条 SSE 事件的载体。
// - EventSnapshot 时 Users 有值,User 为 nil
// - 其它事件 User 为主体,Users 为 nil
type SubEvent struct {
	Event   string        `json:"event"`
	Channel string        `json:"channel"`
	User    *PttUserResp  `json:"user,omitempty"`
	Users   []PttUserResp `json:"users,omitempty"`
	Ts      int64         `json:"ts"`
}

// 每个订阅者一个通道(带缓冲,慢客户端不阻塞广播)
type subscriber struct {
	userID string
	ch     chan SubEvent
}

// channel -> subscribers,与 Channels/UserChannels 独立以简化锁边界
var (
	subscribers    = make(map[string]map[*subscriber]struct{}) // channelName -> set
	subscriberLock sync.RWMutex
)

const subscriberBuffer = 256

// Subscribe 为指定用户在指定频道注册订阅,返回事件通道 & 一个用于取消的 close 函数。
// 调用方 goroutine 负责从 ch 消费;若不消费,超过 subscriberBuffer 长度后的事件会被丢弃(记入日志)。
func Subscribe(userID string, channelName string) (<-chan SubEvent, func()) {
	sub := &subscriber{
		userID: userID,
		ch:     make(chan SubEvent, subscriberBuffer),
	}

	subscriberLock.Lock()
	set, ok := subscribers[channelName]
	if !ok {
		set = make(map[*subscriber]struct{})
		subscribers[channelName] = set
	}
	set[sub] = struct{}{}
	subscriberLock.Unlock()

	cancel := func() {
		subscriberLock.Lock()
		if set, ok := subscribers[channelName]; ok {
			delete(set, sub)
			if len(set) == 0 {
				delete(subscribers, channelName)
			}
		}
		subscriberLock.Unlock()
		close(sub.ch)
	}
	return sub.ch, cancel
}

// Broadcast 向频道内所有订阅者(排除 excludeUserID)推送事件。慢消费者按队列容量丢弃,不阻塞。
func Broadcast(channelName string, excludeUserID string, evt SubEvent) {
	subscriberLock.RLock()
	set, ok := subscribers[channelName]
	if !ok || len(set) == 0 {
		subscriberLock.RUnlock()
		return
	}
	targets := make([]*subscriber, 0, len(set))
	for s := range set {
		if s.userID == excludeUserID {
			continue
		}
		targets = append(targets, s)
	}
	subscriberLock.RUnlock()

	for _, s := range targets {
		select {
		case s.ch <- evt:
		default:
		}
	}
}

// MarshalEvent 便于处理:序列化事件为 JSON 字节流
func MarshalEvent(evt SubEvent) []byte {
	b, _ := json.Marshal(evt)
	return b
}
