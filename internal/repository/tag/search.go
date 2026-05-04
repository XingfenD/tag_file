package tag

import "github.com/XingfenD/tag_file/internal/model"

type SearchOperation struct {
	repo    *Repository
	keyword string
	offset  int
	limit   int
}

func (r *Repository) Search() *SearchOperation {
	return &SearchOperation{repo: r}
}

func (op *SearchOperation) ByName(kw string) *SearchOperation {
	op.keyword = kw
	return op
}

func (op *SearchOperation) Offset(n int) *SearchOperation {
	op.offset = n
	return op
}

func (op *SearchOperation) Limit(n int) *SearchOperation {
	op.limit = n
	return op
}

func (op *SearchOperation) Exec() ([]model.Tag, int64, error) {
	pattern := "%" + op.keyword + "%"
	var total int64
	op.repo.db.Model(&model.Tag{}).Where("tag_name LIKE ?", pattern).Count(&total)
	var tags []model.Tag
	q := op.repo.db.Preload("Category").Where("tag_name LIKE ?", pattern).Order("tag_id")
	if op.limit > 0 {
		q = q.Offset(op.offset).Limit(op.limit)
	}
	err := q.Find(&tags).Error
	return tags, total, err
}
