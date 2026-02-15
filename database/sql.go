package database

import (
	"errors"
	"time"

	"github.com/sunvc/NoLets/common"
	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

var newDB *gorm.DB
var isSqlite3 = false

type NewSQL struct{}

func (d *NewSQL) ExportOrImport(dataArr ...User) ([]User, error) {
	if len(dataArr) > 0 {
		if err := newDB.Save(&dataArr).Error; err != nil {
			return []User{}, err
		}
		return []User{}, nil
	} else {
		var users []User
		err := newDB.Model(&User{}).Find(&users).Error
		return users, err
	}

}

func (d *NewSQL) CountAll() (int, error) {
	var count int64
	result := newDB.Model(&User{}).Count(&count)
	return int(count), result.Error
}

func (d *NewSQL) DeviceTokenByKey(key string) (string, error) {

	var user *User
	if result := newDB.Where("key = ?", key).First(&user); result.Error != nil {
		return "", result.Error
	}
	return user.Token, nil
}

func (d *NewSQL) DeviceTokenByGroup(group string) ([]string, error) {
	var tokens []string
	result := newDB.Model(&User{}).Where("user_group = ?", group).Pluck("token", &tokens)
	return tokens, result.Error
}

func (d *NewSQL) SaveDeviceTokenByKey(key, token, group string) (string, error) {
	if key == "" {
		// Generate new UUID
		key = common.UserID()
	}

	var user User
	result := newDB.Where("key = ?", key).First(&user)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			// User does not exist, create new user
			user = User{
				Key:   key,
				Token: token,
				Group: group,
			}
			if err := newDB.Create(&user).Error; err != nil {
				return "", err
			}
			return key, nil
		}
		// Other database errors
		return "", result.Error
	}

	if len(token) < 64 && group == user.Group {
		newDB.Unscoped().Delete(&user)
		return key, nil
	}

	// User exists, update token
	user.Token = token
	if err := newDB.Save(&user).Error; err != nil {
		return "", err
	}

	return key, nil
}

func (d *NewSQL) Close() error {
	sqlDB, err := newDB.DB()
	if err != nil {
		return err
	}
	if isSqlite3 {
		_, err = sqlDB.Exec("PRAGMA wal_checkpoint(TRUNCATE);VACUUM;")
		_, err = sqlDB.Exec("VACUUM;")
	}

	return sqlDB.Close()
}

func (d *NewSQL) KeyExists(key string) bool {
	var user User
	// Only query primary key to improve efficiency
	err := newDB.Select("id").Where("key = ?", key).First(&user).Error
	if err != nil {
		// Return false if not exists or any error
		return false
	}
	return true
}

func NewMysql(dsn string) Database {
	var err error
	newDB, err = gorm.Open(mysql.New(mysql.Config{
		DSN:                       dsn,   // DSN data source name
		DefaultStringSize:         191,   // Default length for string type fields
		SkipInitializeWithVersion: false, // Configure automatically based on version
		DontSupportRenameColumn:   true,
	}), &gorm.Config{
		PrepareStmt: true,
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true, // Use singular table name
		},
	})

	if err != nil {
		panic("failed to connect database")
	}

	err = newDB.AutoMigrate(&User{})
	sqlDB, _ := newDB.DB()
	// MySQL connection pool
	sqlDB.SetMaxOpenConns(50)
	sqlDB.SetMaxIdleConns(10)

	if err != nil {
		panic("failed to connect database")
	}

	return &NewSQL{}
}

func NewSqlite3() Database {
	var err error

	newDB, err = gorm.Open(sqlite.Open(common.BaseDir(common.APPNAME+".sqlite")), &gorm.Config{
		PrepareStmt: true,
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true, // Use singular table name
		},
	})

	if err != nil {
		panic("failed to connect database")
	}

	err = newDB.AutoMigrate(&User{})

	if err != nil {
		panic("failed to connect database")
	}

	sqlDB, _ := newDB.DB()
	_, _ = sqlDB.Exec(`PRAGMA journal_mode = WAL;`)

	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetMaxIdleConns(2)
	sqlDB.SetConnMaxLifetime(time.Hour)

	return &NewSQL{}

}
