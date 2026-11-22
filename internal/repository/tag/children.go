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

// ChildrenMap batch-loads direct children for multiple parents in one query.
// Values keep tag_id order, same as ChildIDs.
func (r *Repository) ChildrenMap(parentIDs []uint) (map[uint][]uint, error) {
	if len(parentIDs) == 0 {
		return nil, nil
	}
	type row struct {
		TagID    uint `gorm:"column:tag_id"`
		ParentID uint `gorm:"column:parent_id"`
	}
	var rows []row
	err := r.db.Model(&model.Tag{}).Select("tag_id, parent_id").
		Where("parent_id IN ?", parentIDs).Order("parent_id, tag_id").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	m := make(map[uint][]uint)
	for _, x := range rows {
		m[x.ParentID] = append(m[x.ParentID], x.TagID)
	}
	return m, nil
}
