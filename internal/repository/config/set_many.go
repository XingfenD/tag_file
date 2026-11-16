package config

import (
	"github.com/XingfenD/tag_file/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (r *Repository) SetMany(values map[string]string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		for key, value := range values {
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "config_name"}},
				DoUpdates: clause.AssignmentColumns([]string{"config_value"}),
			}).Create(&model.Config{ConfigName: key, ConfigValue: value}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
