package database

import (
	"errors"

	"github.com/sunvc/NoLets/common"
	"gorm.io/gorm"
)

var newDB *gorm.DB

type NewSQL struct{}

func (d *NewSQL) ExportOrImport(dataArr ...common.User) ([]common.User, error) {
	if len(dataArr) > 0 {
		if err := newDB.Save(&dataArr).Error; err != nil {
			return []common.User{}, err
		}
		return []common.User{}, nil
	} else {
		var users []common.User
		err := newDB.Model(&common.User{}).Find(&users).Error
		return users, err
	}

}

func (d *NewSQL) CountAll() (int, error) {
	var count int64
	result := newDB.Model(&common.User{}).Count(&count)
	return int(count), result.Error
}

func (d *NewSQL) DeviceTokenByKey(key string) (*common.User, error) {

	var user *common.User
	if result := newDB.Where("key = ?", key).First(&user); result.Error != nil {
		return nil, result.Error
	}
	return user, nil
}

func (d *NewSQL) DeviceTokenByGroup(group string) ([]*common.User, error) {
	var users []*common.User
	if err := newDB.Where("user_group = ?", group).Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

func (d *NewSQL) SaveDeviceTokenByKey(user common.User) (string, error) {

	if user.Key == "" {
		// Generate new UUID
		user.Key = common.UserID()
	}

	var dbUser common.User
	result := newDB.Where("key = ?", user.Key).First(&dbUser)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			// User does not exist, create new user
			if err := newDB.Create(&user).Error; err != nil {
				return "", err
			}
			return user.Key, nil
		}
		// Other database errors
		return "", result.Error
	}

	if len(user.Token) < 64 && len(user.Group) > 0 && user.Group == dbUser.Group {
		newDB.Unscoped().Delete(&dbUser)
		return user.Key, nil
	}

	// User exists, update token
	dbUser.Token = user.Token
	dbUser.Talk = user.Talk
	dbUser.Location = user.Location
	if err := newDB.Save(&dbUser).Error; err != nil {
		return "", err
	}

	return dbUser.Key, nil
}

func (d *NewSQL) Close() error {
	sqlDB, err := newDB.DB()
	if err != nil {
		return err
	}
	_, err = sqlDB.Exec("PRAGMA wal_checkpoint(TRUNCATE);VACUUM;")
	_, err = sqlDB.Exec("VACUUM;")
	return sqlDB.Close()
}

func (d *NewSQL) KeyExists(key string) bool {
	var user common.User
	// Only query primary key to improve efficiency
	err := newDB.Select("id").Where("key = ?", key).First(&user).Error
	if err != nil {
		// Return false if not exists or any error
		return false
	}
	return true
}
