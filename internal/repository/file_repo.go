package repository

import (
	"errors"

	"github.com/XingfenD/tag_file/internal/model"
	"github.com/XingfenD/tag_file/pkg/storage"
	"gorm.io/gorm"
)

type FileRepository struct {
	db *gorm.DB
}

func NewFileRepository() *FileRepository {
	return &FileRepository{db: storage.GetDB()}
}

func (r *FileRepository) Create(file *model.File) error {
	return r.db.Create(file).Error
}

func (r *FileRepository) FindByPath(path string) (*model.File, error) {
	var file model.File
	err := r.db.Where("file_path = ?", path).First(&file).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &file, err
}

func (r *FileRepository) FindByID(id uint) (*model.File, error) {
	var file model.File
	err := r.db.First(&file, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &file, err
}

func (r *FileRepository) List() ([]model.File, error) {
	var files []model.File
	err := r.db.Order("file_path").Find(&files).Error
	return files, err
}

func (r *FileRepository) Delete(id uint) error {
	return r.db.Delete(&model.File{}, id).Error
}

func (r *FileRepository) AddTag(fileID uint, tagID uint) error {
	ft := model.FileTag{FileID: fileID, TagID: tagID}
	return r.db.Create(&ft).Error
}

func (r *FileRepository) RemoveTag(fileID uint, tagID uint) error {
	return r.db.Where("file_id = ? AND tag_id = ?", fileID, tagID).Delete(&model.FileTag{}).Error
}

func (r *FileRepository) HasTag(fileID uint, tagID uint) (bool, error) {
	var count int64
	err := r.db.Model(&model.FileTag{}).Where("file_id = ? AND tag_id = ?", fileID, tagID).Count(&count).Error
	return count > 0, err
}

func (r *FileRepository) GetTags(fileID uint) ([]model.Tag, error) {
	var tags []model.Tag
	err := r.db.Preload("Category").
		Joins("JOIN file_tags ON file_tags.tag_id = tags.tag_id").
		Where("file_tags.file_id = ?", fileID).
		Find(&tags).Error
	return tags, err
}

func (r *FileRepository) SearchByTags(tagIDs []uint, logic string) ([]model.File, error) {
	if len(tagIDs) == 0 {
		return nil, nil
	}

	var files []model.File
	switch logic {
	case "and":
		err := r.db.Where("file_id IN (?)",
			r.db.Table("file_tags").
				Select("file_id").
				Where("tag_id IN (?)", tagIDs).
				Group("file_id").
				Having("COUNT(DISTINCT tag_id) = ?", len(tagIDs)),
		).Order("file_path").Find(&files).Error
		return files, err
	case "not":
		err := r.db.Where("file_id NOT IN (?)",
			r.db.Table("file_tags").Select("file_id").Where("tag_id IN (?)", tagIDs),
		).Order("file_path").Find(&files).Error
		return files, err
	default: // "or"
		err := r.db.Where("file_id IN (?)",
			r.db.Table("file_tags").Select("file_id").Where("tag_id IN (?)", tagIDs),
		).Order("file_path").Find(&files).Error
		return files, err
	}
}
