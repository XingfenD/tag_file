package config

import (
	"errors"

	"github.com/XingfenD/tag_file/internal/model"
	"gorm.io/gorm"
)

func (r *Repository) Set(key string, value string) error {
	var cfg model.Config
	err := r.db.Where("config_name = ?", key).First(&cfg).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return r.db.Create(&model.Config{ConfigName: key, ConfigValue: value}).Error
	}
	if err != nil {
		return err
	}
	return r.db.Model(&cfg).Update("config_value", value).Error
}
