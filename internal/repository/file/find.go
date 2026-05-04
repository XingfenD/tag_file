package file

import (
	"errors"

	"github.com/XingfenD/tag_file/internal/model"
	"gorm.io/gorm"
)

type FindOperation struct {
	repo *Repository
	id   *uint
	path *string
}

func (r *Repository) Find() *FindOperation {
	return &FindOperation{repo: r}
}

func (op *FindOperation) ByID(id uint) *FindOperation {
	op.id = &id
	return op
}

func (op *FindOperation) ByPath(path string) *FindOperation {
	op.path = &path
	return op
}

func (op *FindOperation) Exec() (*model.File, error) {
	q := op.repo.db
	if op.id != nil {
		var file model.File
		err := q.First(&file, *op.id).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return &file, err
	}
	var file model.File
	err := q.Where("file_path = ?", *op.path).First(&file).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &file, err
}
