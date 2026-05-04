package tag

import "github.com/XingfenD/tag_file/internal/model"

type UpdateOperation struct {
	repo *Repository
	tag  *model.Tag
	cols []string
}

func (r *Repository) Update(tag *model.Tag) *UpdateOperation {
	return &UpdateOperation{repo: r, tag: tag}
}

func (op *UpdateOperation) SetName(v string) *UpdateOperation {
	op.cols = append(op.cols, "tag_name")
	op.tag.TagName = v
	return op
}

func (op *UpdateOperation) SetDescription(v string) *UpdateOperation {
	op.cols = append(op.cols, "description")
	op.tag.Description = v
	return op
}

func (op *UpdateOperation) SetParentID(v *uint) *UpdateOperation {
	op.cols = append(op.cols, "parent_id")
	op.tag.ParentID = v
	return op
}

func (op *UpdateOperation) Exec() error {
	if len(op.cols) == 0 {
		return op.repo.db.Save(op.tag).Error
	}
	return op.repo.db.Model(op.tag).Select(op.cols).Updates(*op.tag).Error
}
