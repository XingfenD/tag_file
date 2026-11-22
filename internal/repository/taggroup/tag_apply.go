package taggroup

import (
	"github.com/XingfenD/tag_file/internal/model"
	"gorm.io/gorm"
)

// ApplyRow is one materialized tag row to keep after a replace.
type ApplyRow struct {
	TagID uint
	Value string
}

// ReplaceRows takes over the given tags in a group: every existing row of
// tagIDs is removed and rows is written in their place, in one transaction,
// keeping File.TagCnt in step with the row delta. An empty tagIDs with no
// rows is a deliberate no-op: schema controls fire on unchanged values and
// must not surface an error for "nothing was there anyway".
func (r *Repository) ReplaceRows(groupID, fileID uint, tagIDs []uint, rows []ApplyRow) (int, error) {
	if len(tagIDs) == 0 && len(rows) == 0 {
		return 0, nil
	}
	delta := len(rows)
	err := r.db.Transaction(func(tx *gorm.DB) error {
		if len(tagIDs) > 0 {
			res := tx.Where("group_id = ? AND tag_id IN ?", groupID, tagIDs).Delete(&model.TagGroupTag{})
			if res.Error != nil {
				return res.Error
			}
			delta -= int(res.RowsAffected)
		}
		if len(rows) > 0 {
			assoc := make([]model.TagGroupTag, 0, len(rows))
			for _, row := range rows {
				assoc = append(assoc, model.TagGroupTag{
					GroupID: groupID, FileID: fileID, TagID: row.TagID, TagValue: row.Value,
				})
			}
			if err := tx.Create(&assoc).Error; err != nil {
				return err
			}
		}
		if delta != 0 {
			return tx.Model(&model.File{}).Where("file_id = ?", fileID).
				UpdateColumn("tag_cnt", gorm.Expr("MAX(tag_cnt + ?, 0)", delta)).Error
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return delta, nil
}
