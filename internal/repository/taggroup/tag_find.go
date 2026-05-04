package taggroup

import "github.com/XingfenD/tag_file/internal/model"

func (r *Repository) HasTagValue(groupID uint, tagID uint, tagValue string) (bool, error) {
	var count int64
	err := r.db.Model(&model.TagGroupTag{}).Where("group_id = ? AND tag_id = ? AND tag_value = ?", groupID, tagID, tagValue).Count(&count).Error
	return count > 0, err
}

func (r *Repository) GetTags(groupID uint) ([]model.Tag, error) {
	var tags []model.Tag
	err := r.db.Preload("Category").
		Joins("JOIN tag_group_tags ON tag_group_tags.tag_id = tags.tag_id").
		Where("tag_group_tags.group_id = ?", groupID).
		Group("tags.tag_id").
		Find(&tags).Error
	return tags, err
}

func (r *Repository) GetFileTags(fileID uint, offset, limit int) ([]model.TagGroupTag, int64, error) {
	var total int64
	r.db.Model(&model.TagGroupTag{}).Where("file_id = ?", fileID).Count(&total)
	var fts []model.TagGroupTag
	q := r.db.Preload("Tag.Category").Preload("Group").Where("file_id = ?", fileID).Order("id")
	if limit > 0 {
		q = q.Offset(offset).Limit(limit)
	}
	err := q.Find(&fts).Error
	return fts, total, err
}

func (r *Repository) GetFileTagsByGroup(fileID uint, groupID uint, offset, limit int) ([]model.TagGroupTag, int64, error) {
	var total int64
	r.db.Model(&model.TagGroupTag{}).Where("file_id = ? AND group_id = ?", fileID, groupID).Count(&total)
	var fts []model.TagGroupTag
	q := r.db.Preload("Tag.Category").Preload("Group").Where("file_id = ? AND group_id = ?", fileID, groupID).Order("id")
	if limit > 0 {
		q = q.Offset(offset).Limit(limit)
	}
	err := q.Find(&fts).Error
	return fts, total, err
}

func (r *Repository) GetAllFileTagValues(fileID uint) (map[uint][]string, error) {
	var fts []model.TagGroupTag
	err := r.db.Where("file_id = ? AND tag_value != ''", fileID).Find(&fts).Error
	if err != nil {
		return nil, err
	}
	m := make(map[uint][]string)
	for _, ft := range fts {
		m[ft.TagID] = append(m[ft.TagID], ft.TagValue)
	}
	return m, nil
}
