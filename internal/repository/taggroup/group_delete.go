package taggroup

import "github.com/XingfenD/tag_file/internal/model"

func (r *Repository) DeleteGroup(groupID uint) error {
	r.db.Where("group_id = ?", groupID).Delete(&model.TagGroupTag{})
	return r.db.Delete(&model.TagGroup{}, groupID).Error
}
