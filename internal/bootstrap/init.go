package bootstrap

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/XingfenD/tag_file/internal/constant"
	"github.com/XingfenD/tag_file/internal/model"
	"github.com/XingfenD/tag_file/pkg/storage"
	"gorm.io/gorm"
)

var defaultExcludePatterns = []string{
	`^\$RECYCLE\.BIN$`,
	`^System Volume Information$`,
	`^\.`,
}

func seedDefaultConfig(db *gorm.DB) {
	var count int64
	db.Model(&model.Config{}).Where("config_name = ?", "scan_exclude").Count(&count)
	if count > 0 {
		return
	}
	data, _ := json.Marshal(defaultExcludePatterns)
	db.Create(&model.Config{
		ConfigName:  "scan_exclude",
		ConfigValue: string(data),
		Description: "扫描排除正则列表",
	})
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
	); err != nil {
		return err
	}

	seedDefaultConfig(storage.GetDB())
	return nil
}
