package tag

import (
	"errors"

	"github.com/XingfenD/tag_file/internal/model"
	"gorm.io/gorm"
)

func (r *Repository) CreateValue(val *model.TagValue) error {
	return r.db.Create(val).Error
}

func (r *Repository) FindValueByName(tagID uint, value string) (*model.TagValue, error) {
	var val model.TagValue
	err := r.db.Where("tag_id = ? AND value = ?", tagID, value).First(&val).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &val, err
}

func (r *Repository) ListValues(tagID uint) ([]model.TagValue, error) {
	var vals []model.TagValue
	err := r.db.Where("tag_id = ?", tagID).Order("value_id").Find(&vals).Error
	return vals, err
}

func (r *Repository) DeleteValue(valueID uint) error {
	return r.db.Delete(&model.TagValue{}, valueID).Error
}

func (r *Repository) HasValues(tagID uint) (bool, error) {
	var count int64
	err := r.db.Model(&model.TagValue{}).Where("tag_id = ?", tagID).Count(&count).Error
	return count > 0, err
}
