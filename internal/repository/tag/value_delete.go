package tag

import "github.com/XingfenD/tag_file/internal/model"

func (r *Repository) DeleteValue(valueID uint) error {
	return r.db.Delete(&model.TagValue{}, valueID).Error
}
