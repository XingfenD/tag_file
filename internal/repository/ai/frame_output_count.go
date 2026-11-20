package ai

import "github.com/XingfenD/tag_file/internal/model"

type rawCountRow struct {
	TaskID uint
	Cnt    int64
}

// RawCountsBySource counts the stored frames of the given first-pass tasks in
// one GROUP BY query. Tasks without stored frames are absent from the result;
// an empty source list answers an empty map without touching the database.
func (r *Repository) RawCountsBySource(sourceIDs []uint) (map[uint]int64, error) {
	counts := make(map[uint]int64, len(sourceIDs))
	if len(sourceIDs) == 0 {
		return counts, nil
	}
	var rows []rawCountRow
	err := r.db.Model(&model.AIFrameOutput{}).
		Select("task_id, COUNT(*) as cnt").
		Where("task_id IN ?", sourceIDs).
		Group("task_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		counts[row.TaskID] = row.Cnt
	}
	return counts, nil
}
