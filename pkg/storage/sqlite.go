package storage

import (
	"log"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var db *gorm.DB

func InitDB(dbPath string) error {
	var err error
	db, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		return err
	}
	log.Println("Database initialized successfully")
	return nil
}

func GetDB() *gorm.DB {
	return db
}
