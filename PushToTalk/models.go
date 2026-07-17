package PushToTalk

type PttUser struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Token     string  `json:"token"`
	Host      string  `json:"host"`
	Timestamp int64   `json:"timestamp"`
}

type Channel struct {
	UserIDs map[string]struct{}
}

func (c *Channel) UserList() []PttUser {
	userList := make([]PttUser, 0, len(c.UserIDs))

	for uid := range c.UserIDs {
		if val, ok := GlobalUsers.Load(uid); ok {
			if userPtr, ok := val.(PttUser); ok {
				userList = append(userList, userPtr)
			}
		}
	}
	return userList
}

func (c *Channel) UserListResp() []PttUserResp {
	userList := make([]PttUserResp, 0, len(c.UserIDs))

	for uid := range c.UserIDs {
		if val, ok := GlobalUsers.Load(uid); ok {
			if userPtr, ok := val.(PttUser); ok {

				userList = append(userList, PttUserResp{
					ID:        userPtr.ID,
					Name:      userPtr.Name,
					Latitude:  userPtr.Latitude,
					Longitude: userPtr.Longitude,
					Timestamp: userPtr.Timestamp,
				})
			}
		}
	}
	return userList
}

type VoiceMessage struct {
	ID        string `json:"id"`
	Host      string `json:"host"`
	Channel   string `json:"channel"`
	FileName  string `json:"fileName"`
	Sender    string `json:"sender"`
	Timestamp string `json:"timestamp"`
	CreatedAt int64  `json:"createdAt"`
}

type PushTask struct {
	Token string
	Url   string
}

type JoinParams struct {
	PttUser
	Channels []string `json:"channels"`
}

type JoinResponse struct {
	Host    string        `json:"host"`
	Channel string        `json:"channel"`
	Users   []PttUserResp `json:"users"`
}

type PttUserResp struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Timestamp int64   `json:"timestamp"`
}
