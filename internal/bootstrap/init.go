package bootstrap

import (
	"encoding/json"

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

func InitDB() error {
	if err := storage.InitDB(".tagfile/tagfile.db"); err != nil {
		return err
	}

	if err := storage.GetDB().AutoMigrate(
		&model.File{},
		&model.FileTag{},
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
