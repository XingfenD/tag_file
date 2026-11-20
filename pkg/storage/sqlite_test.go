package storage

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/XingfenD/tag_file/internal/model"
	"gorm.io/gorm"
)

// initTestDB opens a throwaway database of the same shape the app runs on and migrates
// the tables the tests below write.
func initTestDB(t *testing.T) func() {
	t.Helper()
	path := t.TempDir() + "/contention.db"
	if err := InitDB(path); err != nil {
		t.Fatalf("init db: %v", err)
	}
	if err := GetDB().AutoMigrate(
		&model.File{}, &model.TagGroup{}, &model.TagGroupTag{}, &model.TagCategory{},
		&model.Tag{}, &model.TagValue{}, &model.Config{}, &model.AITask{}, &model.AISuggestion{}, &model.AIFrameOutput{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return func() { os.Remove(path) }
}

// The recognition worker is the first background DB writer this app has: it updates
// task rows and inserts suggestions while an HTTP handler tags a file or edits the
// library. SQLite serialises writers, and a writer that loses that race is handed
// "database is locked" unless it is told to wait first — which is the fact the worker's
// claim-retry timer exists to survive.
func TestConcurrentWritersDoNotSurfaceLocked(t *testing.T) {
	cleanup := initTestDB(t)
	defer cleanup()

	const writers = 8
	const rounds = 25

	var wg sync.WaitGroup
	errs := make(chan error, writers*rounds*2)

	for w := 0; w < writers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; i < rounds; i++ {
				task := &model.AITask{FileID: uint(w*1000 + i), Status: "queued"}
				if err := GetDB().Create(task).Error; err != nil {
					errs <- fmt.Errorf("create: %w", err)
					continue
				}
				// Two statements inside one transaction, because the interesting case
				// is a write lock taken by a session that already holds the connection.
				if err := GetDB().Transaction(func(tx *gorm.DB) error {
					if err := tx.Model(task).Update("status", "running").Error; err != nil {
						return err
					}
					return tx.Model(task).Update("error", fmt.Sprintf("round %d", i)).Error
				}); err != nil {
					errs <- fmt.Errorf("transaction: %w", err)
				}
			}
		}(w)
	}

	// Readers alongside: a reader that upgrades to a write is the case a busy timeout
	// is known not to rescue, so it belongs in the test rather than behind it.
	var readers sync.WaitGroup
	for r := 0; r < 4; r++ {
		readers.Add(1)
		go func() {
			defer readers.Done()
			for i := 0; i < rounds; i++ {
				var tasks []model.AITask
				if err := GetDB().Find(&tasks).Error; err != nil {
					errs <- fmt.Errorf("read: %w", err)
				}
			}
		}()
	}

	wg.Wait()
	readers.Wait()
	close(errs)

	for err := range errs {
		if text := err.Error(); strings.Contains(text, "locked") || strings.Contains(text, "BUSY") {
			t.Errorf("并发写入撞出锁冲突: %v", err)
		} else {
			t.Errorf("并发写入报错(不是锁冲突, 但同样不该发生): %v", err)
		}
	}
}

// TestBusyTimeoutIsConfigured states the wait on the DSN instead of relying on the
// driver's own default: the timeout has to be in force for the worker's writes, and a
// connection-level pragma is invisible to every caller above this package.
func TestBusyTimeoutIsConfigured(t *testing.T) {
	cleanup := initTestDB(t)
	defer cleanup()

	var timeout int
	if err := GetDB().Raw("PRAGMA busy_timeout").Scan(&timeout).Error; err != nil {
		t.Fatalf("read busy_timeout: %v", err)
	}
	if timeout != 5000 {
		t.Errorf("busy_timeout = %d, want 5000", timeout)
	}
}
