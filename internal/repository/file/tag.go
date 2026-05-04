package file

import (
	"github.com/XingfenD/tag_file/internal/model"
)

func (r *Repository) AddTag(fileID uint, tagID uint, tagValue string) error {
	ft := model.FileTag{FileID: fileID, TagID: tagID, TagValue: tagValue}
	return r.db.Create(&ft).Error
}

func (r *Repository) RemoveTag(fileID uint, tagID uint, tagValue string) error {
	return r.db.Where("file_id = ? AND tag_id = ? AND tag_value = ?", fileID, tagID, tagValue).Delete(&model.FileTag{}).Error
}

func (r *Repository) HasTag(fileID uint, tagID uint) (bool, error) {
	var count int64
	err := r.db.Model(&model.FileTag{}).Where("file_id = ? AND tag_id = ?", fileID, tagID).Count(&count).Error
	return count > 0, err
}

func (r *Repository) HasTagValue(fileID uint, tagID uint, tagValue string) (bool, error) {
	var count int64
	err := r.db.Model(&model.FileTag{}).Where("file_id = ? AND tag_id = ? AND tag_value = ?", fileID, tagID, tagValue).Count(&count).Error
	return count > 0, err
}

func (r *Repository) GetTags(fileID uint) ([]model.Tag, error) {
	var tags []model.Tag
	err := r.db.Preload("Category").
		Joins("JOIN file_tags ON file_tags.tag_id = tags.tag_id").
		Where("file_tags.file_id = ?", fileID).
		Group("tags.tag_id").
		Find(&tags).Error
	return tags, err
}

func (r *Repository) GetTagValues(fileID uint, tagID uint) ([]string, error) {
	var vals []string
	err := r.db.Model(&model.FileTag{}).
		Where("file_id = ? AND tag_id = ? AND tag_value != ''", fileID, tagID).
		Pluck("tag_value", &vals).Error
	return vals, err
}

func (r *Repository) GetAllTagValues(fileID uint) (map[uint][]string, error) {
	var fts []model.FileTag
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

func (r *Repository) GetFileTags(fileID uint, offset, limit int) ([]model.FileTag, int64, error) {
	var total int64
	r.db.Model(&model.FileTag{}).Where("file_id = ?", fileID).Count(&total)
	var fts []model.FileTag
	q := r.db.Preload("Tag.Category").Where("file_id = ?", fileID).Order("id")
	if limit > 0 {
		q = q.Offset(offset).Limit(limit)
	}
	err := q.Find(&fts).Error
	return fts, total, err
}
