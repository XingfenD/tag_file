package repository

import (
	"errors"

	"github.com/XingfenD/tag_file/internal/model"
	"github.com/XingfenD/tag_file/pkg/storage"
	"gorm.io/gorm"
)

type TagRepository struct {
	db *gorm.DB
}

func NewTagRepository() *TagRepository {
	return &TagRepository{db: storage.GetDB()}
}

func (r *TagRepository) Create(tag *model.Tag) error {
	return r.db.Create(tag).Error
}

func (r *TagRepository) FindByName(name string) ([]model.Tag, error) {
	var tags []model.Tag
	err := r.db.Where("tag_name = ?", name).Find(&tags).Error
	return tags, err
}

func (r *TagRepository) FindByNameAndCategory(name string, categoryID uint) (*model.Tag, error) {
	var tag model.Tag
	err := r.db.Where("tag_name = ? AND category_id = ?", name, categoryID).First(&tag).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &tag, err
}

func (r *TagRepository) FindByID(id uint) (*model.Tag, error) {
	var tag model.Tag
	err := r.db.First(&tag, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &tag, err
}

func (r *TagRepository) List() ([]model.Tag, error) {
	var tags []model.Tag
	err := r.db.Preload("Category").Find(&tags).Error
	return tags, err
}

func (r *TagRepository) ListByCategory(categoryID uint) ([]model.Tag, error) {
	var tags []model.Tag
	err := r.db.Where("category_id = ?", categoryID).Find(&tags).Error
	return tags, err
}

func (r *TagRepository) Delete(tagID uint) error {
	return r.db.Delete(&model.Tag{}, tagID).Error
}

func (r *TagRepository) DeleteByName(name string) (int64, error) {
	result := r.db.Where("tag_name = ?", name).Delete(&model.Tag{})
	return result.RowsAffected, result.Error
}
