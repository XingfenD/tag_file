package category

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

func (op *SearchOperation) Exec() ([]model.TagCategory, int64, error) {
	pattern := "%" + op.keyword + "%"
	var total int64
	op.repo.db.Model(&model.TagCategory{}).Where("category_name LIKE ?", pattern).Count(&total)
	var cats []model.TagCategory
	q := op.repo.db.Where("category_name LIKE ?", pattern).Order("category_id")
	if op.limit > 0 {
		q = q.Offset(op.offset).Limit(op.limit)
	}
	err := q.Find(&cats).Error
	return cats, total, err
}
