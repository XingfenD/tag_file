package ai

import "github.com/XingfenD/tag_file/internal/model"

type pendingCountRow struct {
	FileID uint
	Cnt    int64
}

// CountPendingByFile aggregates pending suggestions per file in one query.
// Files without pending suggestions are absent from the result.
func (r *Repository) CountPendingByFile() (map[uint]int64, error) {
	var rows []pendingCountRow
	err := r.db.Model(&model.AISuggestion{}).
		Select("file_id, COUNT(*) as cnt").
		Where("review_status = ?", "pending").
		Group("file_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	counts := make(map[uint]int64, len(rows))
	for _, row := range rows {
		counts[row.FileID] = row.Cnt
	}
	return counts, nil
}
