package tag

import (
	"errors"

	"github.com/XingfenD/tag_file/internal/model"
	"github.com/XingfenD/tag_file/pkg/storage"
	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository() *Repository {
	return &Repository{db: storage.GetDB()}
}

func (r *Repository) Create(tag *model.Tag) error {
	return r.db.Create(tag).Error
}

func (r *Repository) FindByName(name string) ([]model.Tag, error) {
	var tags []model.Tag
	err := r.db.Where("tag_name = ?", name).Find(&tags).Error
	return tags, err
}

func (r *Repository) FindByNameAndCategory(name string, categoryID uint) (*model.Tag, error) {
	var tag model.Tag
	err := r.db.Where("tag_name = ? AND category_id = ?", name, categoryID).First(&tag).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &tag, err
}

func (r *Repository) FindByID(id uint) (*model.Tag, error) {
	var tag model.Tag
	err := r.db.First(&tag, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &tag, err
}

func (r *Repository) List(offset, limit int) ([]model.Tag, int64, error) {
	var total int64
	r.db.Model(&model.Tag{}).Count(&total)
	var tags []model.Tag
	q := r.db.Preload("Category").Order("tag_id")
	if limit > 0 {
		q = q.Offset(offset).Limit(limit)
	}
	err := q.Find(&tags).Error
	return tags, total, err
}

func (r *Repository) ListByCategory(categoryID uint, offset, limit int) ([]model.Tag, int64, error) {
	var total int64
	r.db.Model(&model.Tag{}).Where("category_id = ?", categoryID).Count(&total)
	var tags []model.Tag
	q := r.db.Where("category_id = ?", categoryID).Order("tag_id")
	if limit > 0 {
		q = q.Offset(offset).Limit(limit)
	}
	err := q.Find(&tags).Error
	return tags, total, err
}

func (r *Repository) Update(tag *model.Tag) error {
	return r.db.Save(tag).Error
}

func (r *Repository) SearchByName(keyword string, offset, limit int) ([]model.Tag, int64, error) {
	pattern := "%" + keyword + "%"
	var total int64
	r.db.Model(&model.Tag{}).Where("tag_name LIKE ?", pattern).Count(&total)
	var tags []model.Tag
	q := r.db.Preload("Category").Where("tag_name LIKE ?", pattern).Order("tag_id")
	if limit > 0 {
		q = q.Offset(offset).Limit(limit)
	}
	err := q.Find(&tags).Error
	return tags, total, err
}

func (r *Repository) Delete(tagID uint) error {
	return r.db.Delete(&model.Tag{}, tagID).Error
}

func (r *Repository) DeleteByName(name string) (int64, error) {
	result := r.db.Where("tag_name = ?", name).Delete(&model.Tag{})
	return result.RowsAffected, result.Error
}

func (r *Repository) FindChildren(parentID uint) ([]model.Tag, error) {
	var tags []model.Tag
	err := r.db.Where("parent_id = ?", parentID).Order("tag_id").Find(&tags).Error
	return tags, err
}

func (r *Repository) FindRootsByCategory(categoryID uint) ([]model.Tag, error) {
	var tags []model.Tag
	err := r.db.Where("category_id = ? AND parent_id IS NULL", categoryID).Order("tag_id").Find(&tags).Error
	return tags, err
}

func (r *Repository) FindDescendantIDs(tagID uint) ([]uint, error) {
	var ids []uint
	ids = append(ids, tagID)
	queue := []uint{tagID}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		children, err := r.FindChildren(current)
		if err != nil {
			return nil, err
		}
		for _, child := range children {
			ids = append(ids, child.TagID)
			queue = append(queue, child.TagID)
		}
	}
	return ids, nil
}
