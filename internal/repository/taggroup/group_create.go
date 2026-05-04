package taggroup

import "github.com/XingfenD/tag_file/internal/model"

func (r *Repository) CreateGroup(group *model.TagGroup) error {
	return r.db.Create(group).Error
}
