package database

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/sunvc/NoLets/common"
	"go.etcd.io/bbolt"
)

// BboltDB implement Database interface with ETCD's bbolt
type BboltDB struct {
}

var dbOnce sync.Once
var BBDB *bbolt.DB

func NewBboltdb(dataDir string) Database {
	bboltSetup(dataDir)
	return &BboltDB{}
}

func (d *BboltDB) ExportOrImport(dataArr ...User) ([]User, error) {
	var results []User
	var errResults error
	if len(dataArr) > 0 {
		for _, user := range dataArr {
			_, err := d.SaveDeviceTokenByKey(user.Key, user.Token, user.Group)
			if err != nil {
				results = append(results, user)
				errResults = err
			}
		}
		return results, errResults
	} else {
		var users []User
		err := BBDB.View(func(tx *bbolt.Tx) error {
			_ = tx.Bucket([]byte(common.APPNAME)).ForEach(func(k, v []byte) error {
				users = append(users, User{Key: string(k), Token: string(v)})
				return nil
			})
			return nil
		})
		return users, err
	}
}

func (d *BboltDB) CountAll() (int, error) {
	var keypairCount int
	err := BBDB.View(func(tx *bbolt.Tx) error {
		keypairCount = tx.Bucket([]byte(common.APPNAME)).Stats().KeyN
		return nil
	})

	if err != nil {
		return 0, err
	}

	return keypairCount, nil
}

func (d *BboltDB) Close() error {
	return BBDB.Close()
}

func (d *BboltDB) DeviceTokenByKey(key string) (string, error) {
	var token string
	err := BBDB.View(func(tx *bbolt.Tx) error {
		if bs := tx.Bucket([]byte(common.APPNAME)).Get([]byte(key)); bs == nil {
			return fmt.Errorf("failed to get [%s] device token from database", key)
		} else {
			token = string(bs)
			return nil
		}
	})
	if err != nil {
		return "", err
	}

	return token, nil
}

func (d *BboltDB) DeviceTokenByGroup(group string) ([]string, error) {
	return []string{}, nil
}

// SaveDeviceTokenByKey create or update device token of specified key
func (d *BboltDB) SaveDeviceTokenByKey(key, token, group string) (string, error) {
	err := BBDB.Update(func(tx *bbolt.Tx) error {

		bucket := tx.Bucket([]byte(common.APPNAME))
		// If the deviceKey is empty or the corresponding deviceToken cannot be obtained from the database,
		// it is considered as a new device registration
		if key == "" {
			// Generate a new UUID as the deviceKey when a new device register
			key = common.UserID()
		}
		// update the deviceToken
		return bucket.Put([]byte(key), []byte(token))
	})

	if err != nil {
		return "", err
	}

	return key, nil
}

// bboltSetup set up the bbolt database
func bboltSetup(dataDir string) {
	dbOnce.Do(func() {
		log.Println(fmt.Sprintf("init database [%s]...", dataDir))
		if _, err := os.Stat(dataDir); os.IsNotExist(err) {
			if err = os.MkdirAll(dataDir, 0755); err != nil {
				log.Println(fmt.Sprintf("failed to create database storage dir(%s): %v", dataDir, err))
			}
		} else if err != nil {
			log.Println(fmt.Sprintf("failed to open database storage dir(%s): %v", dataDir, err))
		}

		bboltDB, err := bbolt.Open(filepath.Join(dataDir, common.APPNAME+".db"), 0600, nil)
		if err != nil {
			log.Println(fmt.Sprintf("failed to create file (%s): %v", filepath.Join(dataDir, common.APPNAME+".db"), err))
		}

		err = bboltDB.Update(func(tx *bbolt.Tx) error {
			_, err = tx.CreateBucketIfNotExists([]byte(common.APPNAME))
			return err
		})
		if err != nil {
			log.Println(fmt.Sprintf("failed to create database bucket: %v", err))
		}

		BBDB = bboltDB
	})
}

// KeyExists checks if the specified key exists in the database, returns only bool value
func (d *BboltDB) KeyExists(key string) bool {
	err := BBDB.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(common.APPNAME))
		if bucket == nil {
			return fmt.Errorf("bucket %s not found", common.APPNAME)
		}
		// Check if key exists
		if bucket.Get([]byte(key)) != nil {
			return nil // key exists, return nil indicating no error
		}
		return fmt.Errorf("key not found") // key does not exist, return error
	})

	// If err is nil, it means key exists, otherwise key does not exist
	return err == nil
}
