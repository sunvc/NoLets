package PushToTalk

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/sunvc/NoLets/common"
)

type persistState struct {
	SavedAt      int64               `json:"saved_at"`
	Channels     map[string][]string `json:"channels"`
	UserChannels map[string][]string `json:"user_channels"`
	Users        map[string]PttUser  `json:"users"`
}

var (
	persistPath     string
	persistMu       sync.Mutex
	persistDirty    bool
	persistTicker   *time.Ticker
	persistStop     chan struct{}
	persistInterval = 30 * time.Second
	maxRestoreAge   = 10 * time.Minute
)

func InitPttSystem() {
	persistPath = common.BaseDir("ptt_state.json")
	loadState()

	for i := 0; i < 2; i++ {
		go startPttConsumer(i)
	}
	for i := 0; i < 24; i++ {
		go startPushTaskWorker(i)
	}
	StartFileCleanerService(common.BaseDir("voices"), 5*time.Minute)

	persistTicker = time.NewTicker(persistInterval)
	persistStop = make(chan struct{})
	go func() {
		for {
			select {
			case <-persistStop:
				return
			case <-persistTicker.C:
				if isDirty() {
					SaveState()
				}
			}
		}
	}()
}

func ShutdownPttSystem() {
	if persistTicker != nil {
		persistTicker.Stop()
	}
	if persistStop != nil {
		close(persistStop)
	}
	SaveState()
}

func markDirty() {
	persistMu.Lock()
	persistDirty = true
	persistMu.Unlock()
}

func isDirty() bool {
	persistMu.Lock()
	defer persistMu.Unlock()
	return persistDirty
}

func SaveState() {
	ChannelLock.RLock()
	state := persistState{
		SavedAt:      time.Now().UnixMilli(),
		Channels:     make(map[string][]string, len(Channels)),
		UserChannels: make(map[string][]string, len(UserChannels)),
		Users:        make(map[string]PttUser),
	}
	for chName, ch := range Channels {
		ids := make([]string, 0, len(ch.UserIDs))
		for uid := range ch.UserIDs {
			ids = append(ids, uid)
		}
		state.Channels[chName] = ids
	}
	for uid, chSet := range UserChannels {
		ids := make([]string, 0, len(chSet))
		for ch := range chSet {
			ids = append(ids, ch)
		}
		state.UserChannels[uid] = ids
	}
	ChannelLock.RUnlock()

	GlobalUsers.Range(func(key, value any) bool {
		if u, ok := value.(PttUser); ok {
			state.Users[u.ID] = u
		}
		return true
	})

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		log.Printf("[PTT] persist marshal error: %v", err)
		return
	}

	if err := os.MkdirAll(filepath.Dir(persistPath), 0755); err != nil {
		log.Printf("[PTT] persist mkdir error: %v", err)
		return
	}
	tmp := persistPath + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		log.Printf("[PTT] persist write error: %v", err)
		return
	}
	if err := os.Rename(tmp, persistPath); err != nil {
		log.Printf("[PTT] persist rename error: %v", err)
		return
	}

	persistMu.Lock()
	persistDirty = false
	persistMu.Unlock()
}

func loadState() {
	data, err := os.ReadFile(persistPath)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("[PTT] persist read error: %v", err)
		}
		return
	}

	var state persistState
	if err := json.Unmarshal(data, &state); err != nil {
		log.Printf("[PTT] persist unmarshal error: %v", err)
		return
	}

	if state.SavedAt > 0 {
		age := time.Since(time.UnixMilli(state.SavedAt))
		if age > maxRestoreAge {
			log.Printf("[PTT] persisted state is %v old (max %v), discarding channels/users",
				age.Round(time.Second), maxRestoreAge)
			return
		}
	}

	ChannelLock.Lock()
	for chName, ids := range state.Channels {
		ch := &Channel{UserIDs: make(map[string]struct{}, len(ids))}
		for _, uid := range ids {
			ch.UserIDs[uid] = struct{}{}
		}
		Channels[chName] = ch
	}
	for uid, chs := range state.UserChannels {
		set := make(map[string]struct{}, len(chs))
		for _, ch := range chs {
			set[ch] = struct{}{}
		}
		UserChannels[uid] = set
	}
	ChannelLock.Unlock()

	for id, u := range state.Users {
		GlobalUsers.Store(id, u)
	}

	log.Printf("[PTT] restored %d channels, %d users from %s (age %v)",
		len(state.Channels), len(state.Users), persistPath,
		time.Since(time.UnixMilli(state.SavedAt)).Round(time.Second))
}
