package storage

import (
	"log"
	"os"
	"path/filepath"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var db *gorm.DB

// busyTimeoutDSN is appended to every database path: SQLite serialises writers, and a
// second writer that arrives while the first holds the write lock is handed
// "database is locked" unless it is told to wait first. The recognition worker
// (internal/service/ai) is the first background writer in this app and writes
// concurrently with the HTTP handlers, so the wait is not a tuning knob.
//
// The driver already defaults a connection to this timeout, which is why setting it
// here changes nothing today; it is spelled out so the value this app depends on stops
// being somebody else's default. Note that the pool is deliberately left unlimited:
// one connection would deadlock any session that opens a transaction and then issues
// another query through the same *gorm.DB.
const busyTimeoutDSN = "?_pragma=busy_timeout(5000)"

func InitDB(dbPath string) error {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return err
	}
	var err error
	db, err = gorm.Open(sqlite.Open(dbPath+busyTimeoutDSN), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return err
	}
	log.Println("Database initialized successfully")
	return nil
}

func GetDB() *gorm.DB {
	return db
}
