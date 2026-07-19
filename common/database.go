package common

import (
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Key      string `gorm:"type:varchar(50);uniqueIndex;not null" json:"key"`
	Token    string `gorm:"type:varchar(255);" json:"token,omitempty"`
	Talk     string `gorm:"type:varchar(255);" json:"talk,omitempty"`
	Location string `gorm:"type:varchar(255);" json:"location,omitempty"`
	Group    string `gorm:"type:varchar(255);column:user_group;" json:"group,omitempty"`
}

func UserUnique(users []User) []User {
	tokenMap := make(map[string]struct{})
	locationMap := make(map[string]struct{})
	talkMap := make(map[string]struct{})

	response := make([]User, 0, len(users))

	for _, user := range users {

		// 判断任意一个重复
		if user.Token != "" {
			if _, ok := tokenMap[user.Token]; ok {
				continue
			}
		}

		if user.Location != "" {
			if _, ok := locationMap[user.Location]; ok {
				continue
			}
		}

		if user.Talk != "" {
			if _, ok := talkMap[user.Talk]; ok {
				continue
			}
		}

		// 记录
		if user.Token != "" {
			tokenMap[user.Token] = struct{}{}
		}

		if user.Location != "" {
			locationMap[user.Location] = struct{}{}
		}

		if user.Talk != "" {
			talkMap[user.Talk] = struct{}{}
		}

		response = append(response, user)
	}

	return response
}
