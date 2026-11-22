package schema

import (
	"errors"

	"github.com/XingfenD/tag_file/internal/model"
	"gorm.io/gorm"
)

func (r *Repository) GetSchema(id uint) (*model.TagSchema, error) {
	var s model.TagSchema
	err := r.db.Preload("Fields", func(db *gorm.DB) *gorm.DB {
		return db.Order("sort_order, field_id")
	}).First(&s, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *Repository) GetField(id uint) (*model.TagSchemaField, error) {
	var f model.TagSchemaField
	err := r.db.First(&f, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &f, nil
}

// FieldByBindTagID returns the field that occupies the tag, or nil when the
// tag is free to bind.
func (r *Repository) FieldByBindTagID(tagID uint) (*model.TagSchemaField, error) {
	var f model.TagSchemaField
	err := r.db.Where("bind_tag_id = ?", tagID).First(&f).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &f, nil
}
