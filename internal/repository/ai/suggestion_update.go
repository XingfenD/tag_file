package ai

import (
	"time"

	"github.com/XingfenD/tag_file/internal/model"
)

type SuggestionUpdateOperation struct {
	repo       *Repository
	suggestion *model.AISuggestion
	cols       []string
}

func (r *Repository) UpdateSuggestion(s *model.AISuggestion) *SuggestionUpdateOperation {
	return &SuggestionUpdateOperation{repo: r, suggestion: s}
}

// SetFinal rewrites only the reviewed final category/tag/value; the original
// model output fields stay untouched for comparison during review.
func (op *SuggestionUpdateOperation) SetFinal(category, tag, value string) *SuggestionUpdateOperation {
	op.cols = append(op.cols, "final_category_name", "final_tag_name", "final_value")
	op.suggestion.FinalCategoryName = category
	op.suggestion.FinalTagName = tag
	op.suggestion.FinalValue = value
	return op
}

// SetReview writes the review status, the approved tag id (may be nil) and
// reviewed_at.
func (op *SuggestionUpdateOperation) SetReview(status string, approvedTagID *uint) *SuggestionUpdateOperation {
	op.cols = append(op.cols, "review_status", "approved_tag_id", "reviewed_at")
	op.suggestion.ReviewStatus = status
	op.suggestion.ApprovedTagID = approvedTagID
	now := time.Now()
	op.suggestion.ReviewedAt = &now
	return op
}

// Exec writes only the columns whose setters were called (cols whitelist,
// same shape as internal/repository/tag/update.go).
func (op *SuggestionUpdateOperation) Exec() error {
	if len(op.cols) == 0 {
		return op.repo.db.Save(op.suggestion).Error
	}
	return op.repo.db.Model(op.suggestion).Select(op.cols).Updates(*op.suggestion).Error
}
