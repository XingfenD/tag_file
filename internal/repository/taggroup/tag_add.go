package taggroup

import "github.com/XingfenD/tag_file/internal/model"

func (r *Repository) AddTag(groupID uint, fileID uint, tagID uint, tagValue string) error {
	t := model.TagGroupTag{GroupID: groupID, FileID: fileID, TagID: tagID, TagValue: tagValue}
	return r.db.Create(&t).Error
}
