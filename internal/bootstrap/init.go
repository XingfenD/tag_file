package bootstrap

import (
	"github.com/XingfenD/tag_file/internal/model"
	"github.com/XingfenD/tag_file/pkg/storage"
)

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

	return nil
}
