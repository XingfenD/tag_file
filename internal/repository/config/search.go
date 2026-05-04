package config

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

func (op *SearchOperation) Exec() ([]model.Config, int64, error) {
	pattern := "%" + op.keyword + "%"
	var total int64
	op.repo.db.Model(&model.Config{}).Where("config_name LIKE ?", pattern).Count(&total)
	var cfgs []model.Config
	q := op.repo.db.Where("config_name LIKE ?", pattern).Order("config_name")
	if op.limit > 0 {
		q = q.Offset(op.offset).Limit(op.limit)
	}
	err := q.Find(&cfgs).Error
	return cfgs, total, err
}
