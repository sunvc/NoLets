package PushToTalk

import (
	"sync"
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

	ChannelLock.Lock()
	defer ChannelLock.Unlock()

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
		}
	}
	if len(oldChannelsMap) == 0 {
		delete(UserChannels, userID)
	}
}
