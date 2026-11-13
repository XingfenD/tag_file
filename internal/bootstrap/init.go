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
	aiSvc "github.com/XingfenD/tag_file/internal/service/ai"
	"github.com/XingfenD/tag_file/pkg/storage"
	"gorm.io/gorm"
)

var defaultExcludePatterns = []string{
	`^\$RECYCLE\.BIN$`,
	`^System Volume Information$`,
	`^\.`,
}

// seedDefaultConfig checks every default key on its own and inserts only the
// missing ones, so keys added later still reach an existing database (an early
// return on the first key would skip them forever).
func seedDefaultConfig(db *gorm.DB) {
	excludeData, _ := json.Marshal(defaultExcludePatterns)
	defaults := []model.Config{
		{
			ConfigName:  "scan_exclude",
			ConfigValue: string(excludeData),
			Description: "扫描排除正则列表",
		},
		{
			ConfigName:  aiSvc.KeyOllamaURL,
			ConfigValue: "http://127.0.0.1:11434",
			Description: "Ollama 服务地址",
		},
		{
			ConfigName:  aiSvc.KeyOllamaModel,
			ConfigValue: "",
			Description: "Ollama 视觉模型名称（空表示未配置，识别时报错）",
		},
		{
			ConfigName:  aiSvc.KeyFrameInterval,
			ConfigValue: "1",
			Description: "视频抽帧间隔（秒）",
		},
		{
			ConfigName:  aiSvc.KeyDiffThreshold,
			ConfigValue: "10",
			Description: "近似帧过滤阈值（与上一保留帧的 dHash 汉明距离小于该值的帧被丢弃）",
		},
		{
			ConfigName:  aiSvc.KeyMaxFrames,
			ConfigValue: "10",
			Description: "单个视频最多抽取的帧数",
		},
	}

	for i := range defaults {
		var count int64
		db.Model(&model.Config{}).Where("config_name = ?", defaults[i].ConfigName).Count(&count)
		if count > 0 {
			continue
		}
		if err := db.Create(&defaults[i]).Error; err != nil {
			log.Printf("[bootstrap] seed default config %s: %v", defaults[i].ConfigName, err)
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
