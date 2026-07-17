package database

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/bytedance/sonic"
	"github.com/glebarez/sqlite"
	"github.com/sunvc/NoLets/common"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

var DB Database

const (
	filePath = "database.json"
)

// Database defines all the db operation
type Database interface {
	CountAll() (int, error)                     //Get db records count
	DeviceTokenByKey(key string) (*User, error) //Get specified device's token
	DeviceTokenByGroup(name string) ([]*User, error)
	SaveDeviceTokenByKey(user User) (string, error) //Create or update specified device's token
	ExportOrImport(users ...User) ([]User, error)
	KeyExists(key string) bool
	Close() error //Close the database
}

type User struct {
	gorm.Model
	Key      string `gorm:"type:varchar(50);uniqueIndex;not null" json:"key"`
	Token    string `gorm:"type:varchar(255);" json:"token,omitempty"`
	Talk     string `gorm:"type:varchar(255);" json:"talk,omitempty"`
	Location string `gorm:"type:varchar(255);" json:"location,omitempty"`
	Group    string `gorm:"type:varchar(255);column:user_group;" json:"group,omitempty"`
}

func InitDatabase() {
	dsn := common.LocalConfig.System.DSN

	if len(dsn) > 5 {
		DB = NewMysql(dsn)
		return
	}
	DB = NewSqlite3()
	go func() { ImportData() }()
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
		fmt.Println(err.Error())
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

func ExportData() {
	path := common.BaseDir(filePath)
	users, err := DB.ExportOrImport()
	if err != nil {
		log.Println("Export Failed", err.Error())
		return
	}

	data, err := sonic.Marshal(users)
	if err != nil {
		log.Println("Export Failed", err.Error())
		return
	}

	err = os.WriteFile(path, data, 0644)

	if err != nil {
		log.Println("Export Failed", err.Error())
		return
	}

	log.Println("Export Success", path)
}

func ImportData() {
	path := common.BaseDir(filePath)
	if _, existErr := os.Stat(path); existErr != nil {
		var users []User
		data, err := os.ReadFile(path)
		if err != nil {
			log.Println("Import Failed", err.Error())
			return
		}
		err = sonic.Unmarshal(data, &users)
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
			log.Println("File export success:", path)
		}
	}
}
