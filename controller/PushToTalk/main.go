package PushToTalk

import (
	"sync"
	"time"
)

var (
	GlobalUsers   sync.Map
	Channels      = make(map[string]*Channel)
	UserChannels  = make(map[string]map[string]struct{})
	ChannelLock   sync.RWMutex
	MsgQueue      = make(chan VoiceMessage, 1000)
	PushTaskQueue = make(chan PushTask, 1000)
)

// SyncChannels 核心方法：对比新旧频道列表，自动增量加入和退出
func SyncChannels(user PttUser, newChannels []string) {

	GlobalUsers.Store(user.ID, user)

	userID := user.ID

	newChannelsMap := make(map[string]struct{}, len(newChannels))
	for _, ch := range newChannels {
		newChannelsMap[ch] = struct{}{}
	}

	// 收集本次变更用于广播,在锁外 fan-out
	var joined, left []string

	ChannelLock.Lock()

	oldChannelsMap, hasOld := UserChannels[userID]
	if !hasOld {
		oldChannelsMap = make(map[string]struct{})
		UserChannels[userID] = oldChannelsMap
	}

	for oldCh := range oldChannelsMap {
		if _, keep := newChannelsMap[oldCh]; !keep {
			if ch, ok := Channels[oldCh]; ok {
				delete(ch.UserIDs, userID)
				if len(ch.UserIDs) == 0 {
					delete(Channels, oldCh)
				}
			}
			delete(oldChannelsMap, oldCh)
			left = append(left, oldCh)
		}
	}

	for chName := range newChannelsMap {
		if _, alreadyIn := oldChannelsMap[chName]; !alreadyIn {
			ch, ok := Channels[chName]
			if !ok {
				ch = &Channel{UserIDs: make(map[string]struct{})}
				Channels[chName] = ch
			}
			ch.UserIDs[userID] = struct{}{}
			oldChannelsMap[chName] = struct{}{}
			joined = append(joined, chName)
		}
	}
	if len(oldChannelsMap) == 0 {
		delete(UserChannels, userID)
	}
	ChannelLock.Unlock()

	// 广播增量事件(不含用户 name,客户端从 CloudKit 按 id 拉)
	nowMs := time.Now().UnixMilli()
	userResp := PttUserResp{
		ID:        user.ID,
		Latitude:  user.Latitude,
		Longitude: user.Longitude,
		Timestamp: nowMs,
	}
	for _, chName := range joined {
		Broadcast(chName, "", SubEvent{
			Event:   EventJoin,
			Channel: chName,
			User:    &userResp,
			Ts:      nowMs,
		})
	}
	for _, chName := range left {
		Broadcast(chName, "", SubEvent{
			Event:   EventLeave,
			Channel: chName,
			User:    &PttUserResp{ID: user.ID, Timestamp: nowMs},
			Ts:      nowMs,
		})
	}
}

// BroadcastUpdate 位置心跳等场景下调用,把用户当前坐标推给它所有订阅频道的其它成员。
// 频道成员是否变化不涉及,仅广播 update 事件。
func BroadcastUpdate(user PttUser) {
	GlobalUsers.Store(user.ID, user)

	ChannelLock.RLock()
	channels, ok := UserChannels[user.ID]
	if !ok || len(channels) == 0 {
		ChannelLock.RUnlock()
		return
	}
	names := make([]string, 0, len(channels))
	for name := range channels {
		names = append(names, name)
	}
	ChannelLock.RUnlock()

	nowMs := time.Now().UnixMilli()
	userResp := PttUserResp{
		ID:        user.ID,
		Latitude:  user.Latitude,
		Longitude: user.Longitude,
		Timestamp: nowMs,
	}
	for _, chName := range names {
		Broadcast(chName, user.ID, SubEvent{
			Event:   EventUpdate,
			Channel: chName,
			User:    &userResp,
			Ts:      nowMs,
		})
	}
}
