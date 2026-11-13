package ai

import "github.com/XingfenD/tag_file/internal/model"

// DeletePendingByFile removes the not yet reviewed suggestions of one file
// (called before re-enqueuing so pending proposals do not pile up) and returns
// the number of deleted rows.
func (r *Repository) DeletePendingByFile(fileID uint) (int64, error) {
	res := r.db.Where("file_id = ? AND review_status = ?", fileID, "pending").
		Delete(&model.AISuggestion{})
	return res.RowsAffected, res.Error
}
