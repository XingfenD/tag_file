package tag

import "github.com/XingfenD/tag_file/internal/model"

type ListOperation struct {
	repo   *Repository
	catID  uint
	offset int
	limit  int
}

func (r *Repository) List() *ListOperation {
	return &ListOperation{repo: r}
}

func (op *ListOperation) ByCategory(catID uint) *ListOperation {
	op.catID = catID
	return op
}

func (op *ListOperation) Offset(n int) *ListOperation {
	op.offset = n
	return op
}

func (op *ListOperation) Limit(n int) *ListOperation {
	op.limit = n
	return op
}

func (op *ListOperation) Exec() ([]model.Tag, int64, error) {
	cq := op.repo.db.Model(&model.Tag{})
	if op.catID > 0 {
		cq = cq.Where("category_id = ?", op.catID)
	}
	var total int64
	cq.Count(&total)

	q := op.repo.db.Preload("Category")
	if op.catID > 0 {
		q = q.Where("category_id = ?", op.catID)
	}
	q = q.Order("tag_id")
	if op.limit > 0 {
		q = q.Offset(op.offset).Limit(op.limit)
	}
	var tags []model.Tag
	err := q.Find(&tags).Error
	return tags, total, err
}
