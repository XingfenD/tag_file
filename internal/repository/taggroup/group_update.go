package taggroup

import "github.com/XingfenD/tag_file/internal/model"

type GroupUpdateOperation struct {
	repo  *Repository
	group *model.TagGroup
	cols  []string
}

func (r *Repository) UpdateGroup(group *model.TagGroup) *GroupUpdateOperation {
	return &GroupUpdateOperation{repo: r, group: group}
}

func (op *GroupUpdateOperation) SetName(v string) *GroupUpdateOperation {
	op.cols = append(op.cols, "group_name")
	op.group.GroupName = v
	return op
}

func (op *GroupUpdateOperation) SetDescription(v string) *GroupUpdateOperation {
	op.cols = append(op.cols, "description")
	op.group.Description = v
	return op
}

func (op *GroupUpdateOperation) Exec() error {
	if len(op.cols) == 0 {
		return op.repo.db.Save(op.group).Error
	}
	return op.repo.db.Model(op.group).Select(op.cols).Updates(*op.group).Error
}
