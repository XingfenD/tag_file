package repository

import (
	"errors"

	"github.com/XingfenD/tag_file/internal/model"
	"github.com/XingfenD/tag_file/pkg/storage"
	"gorm.io/gorm"
)

type ConfigRepository struct {
	db *gorm.DB
}

func NewConfigRepository() *ConfigRepository {
	return &ConfigRepository{db: storage.GetDB()}
}

func (r *ConfigRepository) Set(key string, value string) error {
	var cfg model.Config
	err := r.db.Where("config_name = ?", key).First(&cfg).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return r.db.Create(&model.Config{
			ConfigName:  key,
			ConfigValue: value,
		}).Error
	}
	if err != nil {
		return err
	}
	return r.db.Model(&cfg).Update("config_value", value).Error
}

func (r *ConfigRepository) Get(key string) (*model.Config, error) {
	var cfg model.Config
	err := r.db.Where("config_name = ?", key).First(&cfg).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &cfg, err
}

func (r *ConfigRepository) List() ([]model.Config, error) {
	var cfgs []model.Config
	err := r.db.Order("config_name").Find(&cfgs).Error
	return cfgs, err
}

func (r *ConfigRepository) Delete(key string) error {
	return r.db.Where("config_name = ?", key).Delete(&model.Config{}).Error
}
