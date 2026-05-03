package attr

import (
	"errors"

	"github.com/XingfenD/tag_file/internal/model"
	"gorm.io/gorm"
)

func (r *Repository) CreateValue(val *model.AttributeValue) error {
	return r.db.Create(val).Error
}

func (r *Repository) FindValueByID(id uint) (*model.AttributeValue, error) {
	var val model.AttributeValue
	err := r.db.First(&val, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &val, err
}

func (r *Repository) FindValueByName(name string, attrID uint) (*model.AttributeValue, error) {
	var val model.AttributeValue
	err := r.db.Where("value_name = ? AND attribute_id = ?", name, attrID).First(&val).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &val, err
}

func (r *Repository) ListValues(attrID uint, offset, limit int) ([]model.AttributeValue, int64, error) {
	var total int64
	r.db.Model(&model.AttributeValue{}).Where("attribute_id = ?", attrID).Count(&total)
	var vals []model.AttributeValue
	q := r.db.Where("attribute_id = ?", attrID).Order("value_id")
	if limit > 0 {
		q = q.Offset(offset).Limit(limit)
	}
	err := q.Find(&vals).Error
	return vals, total, err
}

func (r *Repository) UpdateValue(val *model.AttributeValue) error {
	return r.db.Save(val).Error
}

func (r *Repository) SearchValuesByName(keyword string, attrID uint, offset, limit int) ([]model.AttributeValue, int64, error) {
	pattern := "%" + keyword + "%"
	var total int64
	r.db.Model(&model.AttributeValue{}).Where("attribute_id = ? AND value_name LIKE ?", attrID, pattern).Count(&total)
	var vals []model.AttributeValue
	q := r.db.Where("attribute_id = ? AND value_name LIKE ?", attrID, pattern).Order("value_id")
	if limit > 0 {
		q = q.Offset(offset).Limit(limit)
	}
	err := q.Find(&vals).Error
	return vals, total, err
}

func (r *Repository) DeleteValue(id uint) error {
	return r.db.Delete(&model.AttributeValue{}, id).Error
}
