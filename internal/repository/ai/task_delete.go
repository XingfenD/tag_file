package ai

import (
	"github.com/XingfenD/tag_file/internal/model"
	"gorm.io/gorm"
)

// DeleteTask removes one task row, its raw frame outputs and its still-pending
// suggestions in a single transaction. The delete is conditional on the status
// the caller read: a queued row the worker claims between that read and this
// write answers 0 rows instead of vanishing under a live run, and then the
// frame outputs stay too — the worker may still be writing them. Approved/
// rejected suggestions are review history and stay. Returns the number of task
// rows deleted (0 or 1).
func (r *Repository) DeleteTask(id uint, status string) (int64, error) {
	var affected int64
	err := r.db.Transaction(func(tx *gorm.DB) error {
		res := tx.Where("task_id = ? AND status = ?", id, status).Delete(&model.AITask{})
		if res.Error != nil {
			return res.Error
		}
		affected = res.RowsAffected
		if affected == 0 {
			return nil
		}
		if err := tx.Where("task_id = ?", id).Delete(&model.AIFrameOutput{}).Error; err != nil {
			return err
		}
		return tx.Where("task_id = ? AND review_status = ?", id, "pending").
			Delete(&model.AISuggestion{}).Error
	})
	if err != nil {
		return 0, err
	}
	return affected, nil
}