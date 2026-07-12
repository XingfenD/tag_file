package tag

import (
	"errors"

	"github.com/XingfenD/tag_file/internal/model"
	"gorm.io/gorm"
)

type FindOperation struct {
	repo    *Repository
	id      *uint
	name    *string
	catID   *uint
	pID     *uint
	preload []string
}

func (r *Repository) FindByNames(names []string) ([]model.Tag, error) {
	if len(names) == 0 {
		return nil, nil
	}
	var tags []model.Tag
	err := r.db.Where("tag_name IN (?)", names).Find(&tags).Error
	return tags, err
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

func (op *FindOperation) ByCategory(catID uint) *FindOperation {
	op.catID = &catID
	return op
}

func (op *FindOperation) ByParent(parentID *uint) *FindOperation {
	op.pID = parentID
	return op
}


func (op *FindOperation) Preload(rel string) *FindOperation {
	op.preload = append(op.preload, rel)
	return op
}

func (op *FindOperation) Exec() (*model.Tag, error) {
	var tag model.Tag
	err := op.repo.db.First(&tag, *op.id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &tag, err
}

func (op *FindOperation) All() ([]model.Tag, error) {
	q := op.repo.db
	for _, p := range op.preload {
		q = q.Preload(p)
	}
	if op.name != nil {
		q = q.Where("tag_name = ?", *op.name)
	}
	if op.catID != nil {
		q = q.Where("category_id = ?", *op.catID)
	}
	if op.pID != nil {
		q = q.Where("parent_id = ?", *op.pID)
	}
	var tags []model.Tag
	err := q.Find(&tags).Error
	return tags, err
}

func (op *FindOperation) ByParentAndName(parentID *uint, name string) *FindOperation {
	op.pID = parentID
	op.name = &name
	return op
}

func (op *FindOperation) ByNameAndCategory(name string, catID uint) *FindOperation {
	op.name = &name
	op.catID = &catID
	return op
}
