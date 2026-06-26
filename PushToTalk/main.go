package PushToTalk

import "sync"

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

	// 0. 【新增】每次同步频道时，顺手更新一下这个人的全局状态（特别是 GPS 经纬度）
	GlobalUsers.Store(user.ID, &user)

	userID := user.ID // 下面逻辑全部用这个 ID 跑
	// 将传入的新列表转为 HashSet，方便 O(1) 复杂度检索
	newChannelsMap := make(map[string]struct{}, len(newChannels))
	for _, ch := range newChannels {
		newChannelsMap[ch] = struct{}{}
	}

	ChannelLock.Lock()
	defer ChannelLock.Unlock()

	// 1. 获取该用户当前在系统里记录的旧频道列表
	oldChannelsMap, hasOld := UserChannels[userID]

	// 如果这是用户第一次连接（旧列表不存在），初始化它
	if !hasOld {
		oldChannelsMap = make(map[string]struct{})
		UserChannels[userID] = oldChannelsMap
	}

	// 2. 找出【需要退出的旧频道】：在旧列表中存在，但在新列表中消失了
	for oldCh := range oldChannelsMap {
		if _, keep := newChannelsMap[oldCh]; !keep {
			// 执行退出逻辑
			if ch, ok := Channels[oldCh]; ok {
				delete(ch.UserIDs, userID)
				if len(ch.UserIDs) == 0 {
					delete(Channels, oldCh) // 频道没人了就销毁
				}
			}
			// 从用户的副索引中移除
			delete(oldChannelsMap, oldCh)
		}
	}

	// 3. 找出【需要加入的新频道】：在新列表中存在，但在旧列表中没有
	for _, chName := range newChannels {
		if _, alreadyIn := oldChannelsMap[chName]; !alreadyIn {
			// 执行加入逻辑
			ch, ok := Channels[chName]
			if !ok {
				ch = &Channel{UserIDs: make(map[string]struct{})}
				Channels[chName] = ch
			}
			ch.UserIDs[userID] = struct{}{}

			// 同步记入用户的副索引
			oldChannelsMap[chName] = struct{}{}
		}
	}

	// 4. 【边界处理】如果用户传过来的新列表是空的，且清理后副索引也空了，直接删掉副索引项
	if len(oldChannelsMap) == 0 {
		delete(UserChannels, userID)
	}
}
