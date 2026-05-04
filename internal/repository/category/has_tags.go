package category

import "github.com/XingfenD/tag_file/internal/model"

func (r *Repository) HasTags(categoryID uint) (bool, error) {
	var count int64
	err := r.db.Model(&model.Tag{}).Where("category_id = ?", categoryID).Count(&count).Error
	return count > 0, err
}
