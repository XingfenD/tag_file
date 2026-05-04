package config

import (
	"errors"

	"github.com/XingfenD/tag_file/internal/model"
	"gorm.io/gorm"
)

func (r *Repository) Get(key string) (*model.Config, error) {
	var cfg model.Config
	err := r.db.Where("config_name = ?", key).First(&cfg).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &cfg, err
}
