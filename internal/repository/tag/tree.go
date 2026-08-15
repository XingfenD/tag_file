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
