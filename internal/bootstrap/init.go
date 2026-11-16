package bootstrap

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/XingfenD/tag_file/internal/constant"
	"github.com/XingfenD/tag_file/internal/model"
	"github.com/XingfenD/tag_file/internal/service/config/schema"
	"github.com/XingfenD/tag_file/pkg/storage"
	"gorm.io/gorm"
)

var defaultExcludePatterns = []string{
	`^\$RECYCLE\.BIN$`,
	`^System Volume Information$`,
	`^\.`,
}

// seedDefaultConfig walks the config schema registry and inserts only the keys
// that are missing, so keys registered later still reach an existing database
// (an early return on the first key would skip them forever). Value and
// description come from the Field declaration the settings page already reads,
// so a fresh database and an upgraded one cannot disagree with what the code
// actually uses.
func seedDefaultConfig(db *gorm.DB) {
	for _, field := range schema.All() {
		var count int64
		config := model.Config{
			ConfigName:  field.Key,
			ConfigValue: field.Default,
			Description: field.Description,
		}

		db.Model(&model.Config{}).Where("config_name = ?", config.ConfigName).Count(&count)
		if count > 0 {
			continue
		}
		if err := db.Create(&config).Error; err != nil {
			log.Printf("[bootstrap] seed default config %s: %v", config.ConfigName, err)
		}
	}
}

func backupDBIfNeeded(dbPath string) error {
	versionFile := filepath.Join(filepath.Dir(dbPath), "core_version")
	workspaceDir := filepath.Dir(dbPath)

	prevVersion, err := os.ReadFile(versionFile)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read version file: %w", err)
	}

	if string(prevVersion) == constant.CoreVersion {
		return nil
	}

	if len(prevVersion) > 0 {
		ts := time.Now().Format("20060102_150405")
		backupPath := fmt.Sprintf("%s.bak.%s", dbPath, ts)
		src, err := os.Open(dbPath)
		if err != nil {
			return fmt.Errorf("open db for backup: %w", err)
		}
		defer src.Close()
		dst, err := os.Create(backupPath)
		if err != nil {
			return fmt.Errorf("create backup: %w", err)
		}
		defer dst.Close()
		if _, err := io.Copy(dst, src); err != nil {
			return fmt.Errorf("copy db to backup: %w", err)
		}
		log.Printf("[bootstrap] DB backed up: %s (version %s → %s)", backupPath, string(prevVersion), constant.CoreVersion)
	} else {
		log.Printf("[bootstrap] First run, recording core version %s", constant.CoreVersion)
	}

	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		return fmt.Errorf("ensure workspace dir: %w", err)
	}
	return os.WriteFile(versionFile, []byte(constant.CoreVersion), 0644)
}

func InitDB() error {
	dbPath := ".tagfile/tagfile.db"

	if err := backupDBIfNeeded(dbPath); err != nil {
		return err
	}

	if err := storage.InitDB(dbPath); err != nil {
		return err
	}

	if err := storage.GetDB().AutoMigrate(
		&model.File{},
		&model.TagGroup{},
		&model.TagGroupTag{},
		&model.TagCategory{},
		&model.Tag{},
		&model.TagValue{},
		&model.Config{},
		&model.AITask{},
		&model.AISuggestion{},
	); err != nil {
		return err
	}

	seedDefaultConfig(storage.GetDB())
	return nil
}
