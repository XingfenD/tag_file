package attr

import (
	"errors"

	"github.com/XingfenD/tag_file/internal/model"
	"gorm.io/gorm"
)

func (r *Repository) CreateAttribute(attr *model.Attribute) error {
	return r.db.Create(attr).Error
}

func (r *Repository) FindAttributeByID(id uint) (*model.Attribute, error) {
	var attr model.Attribute
	err := r.db.First(&attr, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &attr, err
}

func (r *Repository) FindAttributeByName(name string, tagID uint) (*model.Attribute, error) {
	var attr model.Attribute
	err := r.db.Where("attribute_name = ? AND tag_id = ?", name, tagID).First(&attr).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &attr, err
}

func (r *Repository) ListAttributes(tagID uint, offset, limit int) ([]model.Attribute, int64, error) {
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

func (r *Repository) DeleteAttribute(id uint) error {
	return r.db.Delete(&model.Attribute{}, id).Error
}

func (r *Repository) UpdateAttribute(attr *model.Attribute) error {
	return r.db.Save(attr).Error
}

func (r *Repository) SearchAttributesByName(keyword string, tagID uint, offset, limit int) ([]model.Attribute, int64, error) {
	pattern := "%" + keyword + "%"
	countQ := r.db.Model(&model.Attribute{}).Where("attribute_name LIKE ?", pattern)
	if tagID > 0 {
		countQ = countQ.Where("tag_id = ?", tagID)
	}
	var total int64
	countQ.Count(&total)
	var attrs []model.Attribute
	q := r.db.Preload("Tag").Where("attribute_name LIKE ?", pattern)
	if tagID > 0 {
		q = q.Where("tag_id = ?", tagID)
	}
	q = q.Order("attribute_id")
	if limit > 0 {
		q = q.Offset(offset).Limit(limit)
	}
	err := q.Find(&attrs).Error
	return attrs, total, err
}

func (r *Repository) HasValues(attrID uint) (bool, error) {
	var count int64
	err := r.db.Model(&model.AttributeValue{}).Where("attribute_id = ?", attrID).Count(&count).Error
	return count > 0, err
}
