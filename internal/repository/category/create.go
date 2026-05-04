package category

import "github.com/XingfenD/tag_file/internal/model"

type CreateOperation struct {
	repo *Repository
	cat  model.TagCategory
	cols []string
}

func (r *Repository) Create() *CreateOperation {
	return &CreateOperation{repo: r}
}

func (op *CreateOperation) SetName(v string) *CreateOperation {
	op.cols = append(op.cols, "category_name")
	op.cat.CategoryName = v
	return op
}

func (op *CreateOperation) SetDescription(v string) *CreateOperation {
	op.cols = append(op.cols, "description")
	op.cat.Description = v
	return op
}

func (op *CreateOperation) Exec() (*model.TagCategory, error) {
	if len(op.cols) > 0 {
		err := op.repo.db.Select(op.cols).Create(&op.cat).Error
		return &op.cat, err
	}
	err := op.repo.db.Create(&op.cat).Error
	return &op.cat, err
}
