package ai

import "github.com/XingfenD/tag_file/internal/model"

// CreateFrameOutput inserts one raw frame answer and backfills OutputID.
func (r *Repository) CreateFrameOutput(o *model.AIFrameOutput) error {
	return r.db.Create(o).Error
}
