package file

import (
	"github.com/XingfenD/tag_file/internal/model"
)

func (r *Repository) AddTag(fileID uint, tagID uint) error {
	ft := model.FileTag{FileID: fileID, TagID: tagID}
	return r.db.Create(&ft).Error
}

func (r *Repository) RemoveTag(fileID uint, tagID uint) error {
	return r.db.Where("file_id = ? AND tag_id = ?", fileID, tagID).Delete(&model.FileTag{}).Error
}

func (r *Repository) HasTag(fileID uint, tagID uint) (bool, error) {
	var count int64
	err := r.db.Model(&model.FileTag{}).Where("file_id = ? AND tag_id = ?", fileID, tagID).Count(&count).Error
	return count > 0, err
}

func (r *Repository) GetTags(fileID uint) ([]model.Tag, error) {
	var tags []model.Tag
	err := r.db.Preload("Category").
		Joins("JOIN file_tags ON file_tags.tag_id = tags.tag_id").
		Where("file_tags.file_id = ?", fileID).
		Find(&tags).Error
	return tags, err
}
