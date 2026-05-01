package repository

import (
	"errors"

	"github.com/XingfenD/tag_file/internal/model"
	"github.com/XingfenD/tag_file/pkg/storage"
	"gorm.io/gorm"
)

type CategoryRepository struct {
	db *gorm.DB
}

func NewCategoryRepository() *CategoryRepository {
	return &CategoryRepository{db: storage.GetDB()}
}

func (r *CategoryRepository) Create(category *model.TagCategory) error {
	return r.db.Create(category).Error
}

func (r *CategoryRepository) FindByName(name string) (*model.TagCategory, error) {
	var category model.TagCategory
	err := r.db.Where("category_name = ?", name).First(&category).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &category, err
}

func (r *CategoryRepository) FindByID(id uint) (*model.TagCategory, error) {
	var category model.TagCategory
	err := r.db.First(&category, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &category, err
}

func (r *CategoryRepository) List() ([]model.TagCategory, error) {
	var categories []model.TagCategory
	err := r.db.Find(&categories).Error
	return categories, err
}

func (r *CategoryRepository) DeleteByID(id uint) error {
	return r.db.Delete(&model.TagCategory{}, id).Error
}

func (r *CategoryRepository) DeleteByName(name string) error {
	return r.db.Where("category_name = ?", name).Delete(&model.TagCategory{}).Error
}

func (r *CategoryRepository) HasTags(categoryID uint) (bool, error) {
	var count int64
	err := r.db.Model(&model.Tag{}).Where("category_id = ?", categoryID).Count(&count).Error
	return count > 0, err
}
