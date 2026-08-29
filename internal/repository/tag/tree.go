package tag

// GetDescendantIDs uses a recursive CTE to find all descendant tag IDs (including the root).
func (r *Repository) GetDescendantIDs(tagID uint) ([]uint, error) {
	var ids []uint
	err := r.db.Raw(`
		WITH RECURSIVE descendants(id) AS (
			SELECT tag_id FROM tags WHERE tag_id = ?
			UNION ALL
			SELECT t.tag_id FROM tags t INNER JOIN descendants d ON t.parent_id = d.id
		)
		SELECT id FROM descendants
	`, tagID).Scan(&ids).Error
	return ids, err
}

// GetDescendantIDsMultiMap returns a map of tagID -> []descendantIDs for each input tag ID.
func (r *Repository) GetDescendantIDsMultiMap(tagIDs []uint) (map[uint][]uint, error) {
	if len(tagIDs) == 0 {
		return nil, nil
	}
	type row struct {
		RootID uint `gorm:"column:root_id"`
		TagID  uint `gorm:"column:tag_id"`
	}
	var rows []row
	err := r.db.Raw(`
		WITH RECURSIVE descendants(root_id, tag_id) AS (
			SELECT tag_id, tag_id FROM tags WHERE tag_id IN ?
			UNION ALL
			SELECT d.root_id, t.tag_id FROM tags t INNER JOIN descendants d ON t.parent_id = d.tag_id
		)
		SELECT root_id, tag_id FROM descendants
	`, tagIDs).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	m := make(map[uint][]uint)
	for _, r := range rows {
		m[r.RootID] = append(m[r.RootID], r.TagID)
	}
	return m, nil
}

// GetDescendantIDsMulti uses a recursive CTE for multiple root tag IDs.
func (r *Repository) GetDescendantIDsMulti(tagIDs []uint) ([]uint, error) {
	if len(tagIDs) == 0 {
		return nil, nil
	}
	var ids []uint
	err := r.db.Raw(`
		WITH RECURSIVE descendants(id) AS (
			SELECT tag_id FROM tags WHERE tag_id IN ?
			UNION ALL
			SELECT t.tag_id FROM tags t INNER JOIN descendants d ON t.parent_id = d.id
		)
		SELECT id FROM descendants
	`, tagIDs).Scan(&ids).Error
	return ids, err
}
