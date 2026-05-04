package taggroup

import "github.com/XingfenD/tag_file/internal/model"

func (r *Repository) UpdateTagValue(ftID uint, newValue string) error {
	return r.db.Model(&model.TagGroupTag{}).Where("id = ?", ftID).Update("tag_value", newValue).Error
}
