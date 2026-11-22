package schema

import (
	"github.com/XingfenD/tag_file/internal/model"
	"gorm.io/gorm"
)

func (r *Repository) ListSchemas() ([]model.TagSchema, error) {
	var list []model.TagSchema
	err := r.db.Preload("Fields", func(db *gorm.DB) *gorm.DB {
		return db.Order("sort_order, field_id")
	}).Order("sort_order, schema_id").Find(&list).Error
	return list, err
}

func (r *Repository) CreateSchema(s *model.TagSchema) error {
	return r.db.Create(s).Error
}

func (r *Repository) UpdateSchema(s *model.TagSchema) error {
	return r.db.Save(s).Error
}

// DeleteSchema removes the schema together with its field definitions.
// Materialized tag rows deliberately survive — a schema is a write entry,
// deleting it must not reach into the tag tree.
func (r *Repository) DeleteSchema(id uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("schema_id = ?", id).Delete(&model.TagSchemaField{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.TagSchema{}, id).Error
	})
}
