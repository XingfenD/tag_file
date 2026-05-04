package category

import "github.com/XingfenD/tag_file/internal/model"

type ListOperation struct {
	repo   *Repository
	offset int
	limit  int
}

func (r *Repository) List() *ListOperation {
	return &ListOperation{repo: r}
}

func (op *ListOperation) Offset(n int) *ListOperation {
	op.offset = n
	return op
}

func (op *ListOperation) Limit(n int) *ListOperation {
	op.limit = n
	return op
}

func (op *ListOperation) Exec() ([]model.TagCategory, int64, error) {
	var total int64
	op.repo.db.Model(&model.TagCategory{}).Count(&total)
	var cats []model.TagCategory
	q := op.repo.db.Order("category_id")
	if op.limit > 0 {
		q = q.Offset(op.offset).Limit(op.limit)
	}
	err := q.Find(&cats).Error
	return cats, total, err
}
