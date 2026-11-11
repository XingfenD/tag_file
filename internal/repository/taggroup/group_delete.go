package taggroup

import (
	"github.com/XingfenD/tag_file/internal/model"
	"gorm.io/gorm"
)

// DeleteGroup removes the group together with its tag associations, and takes
// the removed rows out of File.TagCnt — that counter is denormalised, so
// skipping it leaves files advertising tags they no longer carry.
func (r *Repository) DeleteGroup(groupID uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		type fileDelta struct {
			FileID uint `gorm:"column:file_id"`
			Rows   int  `gorm:"column:n"`
		}
		var deltas []fileDelta
		if err := tx.Model(&model.TagGroupTag{}).
			Select("file_id, COUNT(*) AS n").
			Where("group_id = ?", groupID).
			Group("file_id").
			Scan(&deltas).Error; err != nil {
			return err
		}

		if err := tx.Where("group_id = ?", groupID).Delete(&model.TagGroupTag{}).Error; err != nil {
			return err
		}
		if err := tx.Delete(&model.TagGroup{}, groupID).Error; err != nil {
			return err
		}

		for _, d := range deltas {
			if err := tx.Model(&model.File{}).Where("file_id = ?", d.FileID).
				UpdateColumn("tag_cnt", gorm.Expr("MAX(tag_cnt - ?, 0)", d.Rows)).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
