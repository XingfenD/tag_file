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

func (r *CategoryRepository) List(offset, limit int) ([]model.TagCategory, int64, error) {
	var total int64
	r.db.Model(&model.TagCategory{}).Count(&total)
	var categories []model.TagCategory
	q := r.db.Order("category_id")
	if limit > 0 {
		q = q.Offset(offset).Limit(limit)
	}
	err := q.Find(&categories).Error
	return categories, total, err
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
