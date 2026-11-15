package ai

import "github.com/XingfenD/tag_file/internal/model"

type SuggestionListOperation struct {
	repo   *Repository
	status string
	fileID uint
	taskID uint
	offset int
	limit  int
}

func (r *Repository) ListSuggestions() *SuggestionListOperation {
	return &SuggestionListOperation{repo: r}
}

func (op *SuggestionListOperation) ByStatus(status string) *SuggestionListOperation {
	op.status = status
	return op
}

func (op *SuggestionListOperation) ByFileID(fileID uint) *SuggestionListOperation {
	op.fileID = fileID
	return op
}

func (op *SuggestionListOperation) ByTaskID(taskID uint) *SuggestionListOperation {
	op.taskID = taskID
	return op
}

func (op *SuggestionListOperation) Offset(n int) *SuggestionListOperation {
	op.offset = n
	return op
}

func (op *SuggestionListOperation) Limit(n int) *SuggestionListOperation {
	op.limit = n
	return op
}

// Exec returns one page of suggestions ordered by suggestion_id plus the
// filtered total. Limit(0) means unlimited.
func (op *SuggestionListOperation) Exec() ([]model.AISuggestion, int64, error) {
	cq := op.repo.db.Model(&model.AISuggestion{})
	if op.status != "" {
		cq = cq.Where("review_status = ?", op.status)
	}
	if op.fileID > 0 {
		cq = cq.Where("file_id = ?", op.fileID)
	}
	if op.taskID > 0 {
		cq = cq.Where("task_id = ?", op.taskID)
	}
	var total int64
	cq.Count(&total)

	q := op.repo.db.Model(&model.AISuggestion{})
	if op.status != "" {
		q = q.Where("review_status = ?", op.status)
	}
	if op.fileID > 0 {
		q = q.Where("file_id = ?", op.fileID)
	}
	if op.taskID > 0 {
		q = q.Where("task_id = ?", op.taskID)
	}
	q = q.Order("suggestion_id")
	if op.limit > 0 {
		q = q.Offset(op.offset).Limit(op.limit)
	}
	var suggestions []model.AISuggestion
	err := q.Find(&suggestions).Error
	return suggestions, total, err
}
