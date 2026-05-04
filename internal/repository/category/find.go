package category

import (
	"errors"

	"github.com/XingfenD/tag_file/internal/model"
	"gorm.io/gorm"
)

type FindOperation struct {
	repo *Repository
	id   *uint
	name *string
}

func (r *Repository) Find() *FindOperation {
	return &FindOperation{repo: r}
}

func (op *FindOperation) ByID(id uint) *FindOperation {
	op.id = &id
	return op
}

func (op *FindOperation) ByName(name string) *FindOperation {
	op.name = &name
	return op
}

func (op *FindOperation) Exec() (*model.TagCategory, error) {
	q := op.repo.db
	if op.id != nil {
		var cat model.TagCategory
		err := q.First(&cat, *op.id).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return &cat, err
	}
	var cat model.TagCategory
	err := q.Where("category_name = ?", *op.name).First(&cat).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &cat, err
}
