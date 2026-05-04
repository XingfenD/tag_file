package tag

import "github.com/XingfenD/tag_file/internal/model"

type CreateOperation struct {
	repo *Repository
	tag  model.Tag
	cols []string
}

func (r *Repository) Create() *CreateOperation {
	return &CreateOperation{repo: r}
}

func (op *CreateOperation) SetCategoryID(v uint) *CreateOperation {
	op.cols = append(op.cols, "category_id")
	op.tag.CategoryID = v
	return op
}

func (op *CreateOperation) SetParentID(v *uint) *CreateOperation {
	op.cols = append(op.cols, "parent_id")
	op.tag.ParentID = v
	return op
}

func (op *CreateOperation) SetName(v string) *CreateOperation {
	op.cols = append(op.cols, "tag_name")
	op.tag.TagName = v
	return op
}

func (op *CreateOperation) SetDescription(v string) *CreateOperation {
	op.cols = append(op.cols, "description")
	op.tag.Description = v
	return op
}

func (op *CreateOperation) Exec() (*model.Tag, error) {
	if len(op.cols) > 0 {
		err := op.repo.db.Select(op.cols).Create(&op.tag).Error
		return &op.tag, err
	}
	err := op.repo.db.Create(&op.tag).Error
	return &op.tag, err
}
