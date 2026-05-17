package taggroup

import "github.com/XingfenD/tag_file/internal/model"

func (r *Repository) CreateGroup(group *model.TagGroup) error {
	return r.db.Create(group).Error
}

func (r *Repository) CreateGroupsBatch(groups []model.TagGroup) error {
	if len(groups) == 0 {
		return nil
	}
	return r.db.Create(&groups).Error
}
