package config

import (
	"github.com/XingfenD/tag_file/internal/model"
	"gorm.io/gorm/clause"
)

func (r *Repository) Set(key string, value string) error {
	return r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "config_name"}},
		DoUpdates: clause.AssignmentColumns([]string{"config_value"}),
	}).Create(&model.Config{ConfigName: key, ConfigValue: value}).Error
}
