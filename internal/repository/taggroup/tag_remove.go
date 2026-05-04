package taggroup

import "github.com/XingfenD/tag_file/internal/model"

func (r *Repository) RemoveTag(groupID uint, tagID uint, tagValue string) error {
	return r.db.Where("group_id = ? AND tag_id = ? AND tag_value = ?", groupID, tagID, tagValue).Delete(&model.TagGroupTag{}).Error
}
