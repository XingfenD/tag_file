package repository

import (
	"errors"

	"github.com/XingfenD/tag_file/internal/model"
	"github.com/XingfenD/tag_file/pkg/storage"
	"gorm.io/gorm"
)

type AttrRepository struct {
	db *gorm.DB
}

func NewAttrRepository() *AttrRepository {
	return &AttrRepository{db: storage.GetDB()}
}

func (r *AttrRepository) CreateAttribute(attr *model.Attribute) error {
	return r.db.Create(attr).Error
}

func (r *AttrRepository) FindAttributeByID(id uint) (*model.Attribute, error) {
	var attr model.Attribute
	err := r.db.First(&attr, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &attr, err
}

func (r *AttrRepository) FindAttributeByName(name string, tagID uint) (*model.Attribute, error) {
	var attr model.Attribute
	err := r.db.Where("attribute_name = ? AND tag_id = ?", name, tagID).First(&attr).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &attr, err
}

func (r *AttrRepository) ListAttributes(tagID uint, offset, limit int) ([]model.Attribute, int64, error) {
	countQ := r.db.Model(&model.Attribute{})
	if tagID > 0 {
		countQ = countQ.Where("tag_id = ?", tagID)
	}
	var total int64
	countQ.Count(&total)

	var attrs []model.Attribute
	q := r.db.Preload("Tag").Order("attribute_id")
	if tagID > 0 {
		q = q.Where("tag_id = ?", tagID)
	}
	if limit > 0 {
		q = q.Offset(offset).Limit(limit)
	}
	err := q.Find(&attrs).Error
	return attrs, total, err
}

func (r *AttrRepository) DeleteAttribute(id uint) error {
	return r.db.Delete(&model.Attribute{}, id).Error
}

func (r *AttrRepository) HasValues(attrID uint) (bool, error) {
	var count int64
	err := r.db.Model(&model.AttributeValue{}).Where("attribute_id = ?", attrID).Count(&count).Error
	return count > 0, err
}

func (r *AttrRepository) CreateValue(val *model.AttributeValue) error {
	return r.db.Create(val).Error
}

func (r *AttrRepository) FindValueByID(id uint) (*model.AttributeValue, error) {
	var val model.AttributeValue
	err := r.db.First(&val, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &val, err
}

func (r *AttrRepository) FindValueByName(name string, attrID uint) (*model.AttributeValue, error) {
	var val model.AttributeValue
	err := r.db.Where("value_name = ? AND attribute_id = ?", name, attrID).First(&val).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &val, err
}

func (r *AttrRepository) ListValues(attrID uint, offset, limit int) ([]model.AttributeValue, int64, error) {
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

func (r *AttrRepository) DeleteValue(id uint) error {
	return r.db.Delete(&model.AttributeValue{}, id).Error
}
