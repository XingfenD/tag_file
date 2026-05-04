package file

import "github.com/XingfenD/tag_file/internal/model"

type DeleteOperation struct {
	repo *Repository
	id   *uint
}

func (r *Repository) Delete() *DeleteOperation {
	return &DeleteOperation{repo: r}
}

func (op *DeleteOperation) ByID(id uint) *DeleteOperation {
	op.id = &id
	return op
}

func (op *DeleteOperation) Exec() error {
	return op.repo.db.Delete(&model.File{}, *op.id).Error
}
