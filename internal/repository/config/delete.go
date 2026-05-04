package config

import "github.com/XingfenD/tag_file/internal/model"

func (r *Repository) Delete(key string) error {
	return r.db.Where("config_name = ?", key).Delete(&model.Config{}).Error
}
