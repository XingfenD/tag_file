package category

import "github.com/XingfenD/tag_file/internal/model"

type DeleteOperation struct {
	repo *Repository
	id   *uint
	name *string
}

func (r *Repository) Delete() *DeleteOperation {
	return &DeleteOperation{repo: r}
}

func (op *DeleteOperation) ByID(id uint) *DeleteOperation {
	op.id = &id
	return op
}

func (op *DeleteOperation) ByName(name string) *DeleteOperation {
	op.name = &name
	return op
}

func (op *DeleteOperation) Exec() error {
	if op.id != nil {
		return op.repo.db.Delete(&model.TagCategory{}, *op.id).Error
	}
	return op.repo.db.Where("category_name = ?", *op.name).Delete(&model.TagCategory{}).Error
}
