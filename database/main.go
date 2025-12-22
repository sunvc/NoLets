package database

import (
	"encoding/json"
	"log"
	"os"
	"strings"

	"github.com/sunvc/NoLets/common"
	"gorm.io/gorm"
)

var DB Database

// Database defines all the db operation
type Database interface {
	CountAll() (int, error)                      //Get db records count
	DeviceTokenByKey(key string) (string, error) //Get specified device's token
	DeviceTokenByGroup(name string) ([]string, error)
	SaveDeviceTokenByKey(key, token, group string) (string, error) //Create or update specified device's token
	ExportOrImport(users ...User) ([]User, error)
	KeyExists(key string) bool
	Close() error //Close the database
}

type User struct {
	gorm.Model
	Key   string `gorm:"type:varchar(50);uniqueIndex;not null" json:"key"`
	Token string `gorm:"type:varchar(50);" json:"token,omitempty"`
	Group string `gorm:"type:varchar(50);column:user_group;" json:"group,omitempty"`
}

func InitDatabase() {
	dsn := common.LocalConfig.System.DSN

	if len(dsn) > 5 {
		isSqlite3 = strings.Contains(dsn, "sqlite")
		if isSqlite3 {
			DB = NewSqlite3()
		} else {
			DB = NewMysql(dsn)
		}
		return
	}
	DB = NewBboltdb(common.BaseDir())

	go func() { ExportOrImport() }()
}

func ExportOrImport() {
	if filePath := common.LocalConfig.System.ExportPath; filePath != "" {
		users, err := DB.ExportOrImport()
		if err != nil {
			log.Println("Export Failed", err.Error())
			return
		}

		data, err := json.Marshal(users)
		if err != nil {
			log.Println("Export Failed", err.Error())
			return
		}

		err = os.WriteFile(common.BaseDir(filePath), data, 0644)

		if err != nil {
			log.Println("Export Failed", err.Error())
			return
		}

		log.Println("Export Success", filePath)

	} else if filePath = common.LocalConfig.System.ImportPath; filePath != "" {
		var users []User
		data, err := os.ReadFile(common.BaseDir(filePath))
		if err != nil {
			log.Println("Import Failed", err.Error())
			return
		}
		err = json.Unmarshal(data, &users)
		if err != nil {
			log.Println("Import Failed", err.Error())
			return
		}
		if len(users) > 0 {
			_, err = DB.ExportOrImport(users...)
			if err != nil {
				log.Println("Import Failed", err.Error())
				return
			}
			log.Println("File export success:", filePath)
		}
	}
}
