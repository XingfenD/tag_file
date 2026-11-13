package ai

import "github.com/XingfenD/tag_file/internal/model"

const batchSize = 100

// CreateSuggestions batch-inserts suggestions and backfills each SuggestionID
// (same shape as internal/repository/file/create_batch.go).
func (r *Repository) CreateSuggestions(items []model.AISuggestion) error {
	for i := 0; i < len(items); i += batchSize {
		end := i + batchSize
		if end > len(items) {
			end = len(items)
		}
		if err := r.db.Create(items[i:end]).Error; err != nil {
			return err
		}
	}
	return nil
}
