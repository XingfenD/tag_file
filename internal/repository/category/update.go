package category

import "github.com/XingfenD/tag_file/internal/model"

type UpdateOperation struct {
	repo *Repository
	cat  *model.TagCategory
	cols []string
}

func (r *Repository) Update(cat *model.TagCategory) *UpdateOperation {
	return &UpdateOperation{repo: r, cat: cat}
}

func (op *UpdateOperation) SetName(v string) *UpdateOperation {
	op.cols = append(op.cols, "category_name")
	op.cat.CategoryName = v
	return op
}

func (op *UpdateOperation) SetDescription(v string) *UpdateOperation {
	op.cols = append(op.cols, "description")
	op.cat.Description = v
	return op
}

func (op *UpdateOperation) Exec() error {
	if len(op.cols) == 0 {
		return op.repo.db.Save(op.cat).Error
	}
	return op.repo.db.Model(op.cat).Select(op.cols).Updates(*op.cat).Error
}
