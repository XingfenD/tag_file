package schema

import "github.com/XingfenD/tag_file/internal/model"

func (r *Repository) ListFields(schemaID uint) ([]model.TagSchemaField, error) {
	var fs []model.TagSchemaField
	err := r.db.Where("schema_id = ?", schemaID).Order("sort_order, field_id").Find(&fs).Error
	return fs, err
}

func (r *Repository) CreateField(f *model.TagSchemaField) error {
	return r.db.Create(f).Error
}

func (r *Repository) UpdateField(f *model.TagSchemaField) error {
	return r.db.Save(f).Error
}

func (r *Repository) DeleteField(id uint) error {
	return r.db.Delete(&model.TagSchemaField{}, id).Error
}
