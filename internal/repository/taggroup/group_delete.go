package taggroup

import (
	"github.com/XingfenD/tag_file/internal/model"
	"gorm.io/gorm"
)

func (r *Repository) DeleteGroup(groupID uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("group_id = ?", groupID).Delete(&model.TagGroupTag{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.TagGroup{}, groupID).Error
	})
}
