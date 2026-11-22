package tag

import "github.com/XingfenD/tag_file/internal/model"

// ChildIDs returns the direct children of a tag (the parent itself excluded),
// in tag_id order. Multi fields take these as their checkbox candidates.
func (r *Repository) ChildIDs(parentID uint) ([]uint, error) {
	var ids []uint
	err := r.db.Model(&model.Tag{}).Where("parent_id = ?", parentID).
		Order("tag_id").Pluck("tag_id", &ids).Error
	return ids, err
}
