package config

import (
	"errors"

	"github.com/XingfenD/tag_file/internal/model"
	"github.com/XingfenD/tag_file/pkg/storage"
	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository() *Repository {
	return &Repository{db: storage.GetDB()}
}

func (r *Repository) Set(key string, value string) error {
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

func (r *Repository) Get(key string) (*model.Config, error) {
	var cfg model.Config
	err := r.db.Where("config_name = ?", key).First(&cfg).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &cfg, err
}

func (r *Repository) List(offset, limit int) ([]model.Config, int64, error) {
	var total int64
	r.db.Model(&model.Config{}).Count(&total)
	var cfgs []model.Config
	q := r.db.Order("config_name")
	if limit > 0 {
		q = q.Offset(offset).Limit(limit)
	}
	err := q.Find(&cfgs).Error
	return cfgs, total, err
}

func (r *Repository) Delete(key string) error {
	return r.db.Where("config_name = ?", key).Delete(&model.Config{}).Error
}
