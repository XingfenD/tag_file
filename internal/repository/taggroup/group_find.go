package taggroup

import "github.com/XingfenD/tag_file/internal/model"

// DefaultGroupPair maps file_id -> default group.
type DefaultGroupPair struct {
	FileID    uint
	GroupID   uint
	GroupName string
}

// FindDefaultGroupsBatch loads default groups for multiple files in a single query.
func (r *Repository) FindDefaultGroupsBatch(fileIDs []uint) ([]DefaultGroupPair, error) {
	if len(fileIDs) == 0 {
		return nil, nil
	}
	var rows []DefaultGroupPair
	err := r.db.Table("tag_groups").
		Select("file_id, group_id, group_name").
		Where("file_id IN (?) AND is_default = ?", fileIDs, true).
		Find(&rows).Error
	return rows, err
}

// TagsByGroupIDs loads tags for multiple groups in a single query.
// Returns map[groupID] -> []Tag.
func (r *Repository) TagsByGroupIDs(groupIDs []uint) (map[uint][]model.Tag, error) {
	if len(groupIDs) == 0 {
		return nil, nil
	}
	type row struct {
		GroupID uint
		model.Tag
	}
	var rows []row
	err := r.db.Table("tags").
		Select("tag_group_tags.group_id, tags.*").
		Joins("JOIN tag_group_tags ON tag_group_tags.tag_id = tags.tag_id").
		Where("tag_group_tags.group_id IN (?)", groupIDs).
		Preload("Category").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	m := make(map[uint][]model.Tag)
	for _, r := range rows {
		m[r.GroupID] = append(m[r.GroupID], r.Tag)
	}
	return m, nil
}

func (r *Repository) FindDefaultGroup(fileID uint) (*model.TagGroup, error) {
	var group model.TagGroup
	err := r.db.Where("file_id = ? AND is_default = ?", fileID, true).First(&group).Error
	if err != nil {
		return nil, err
	}
	return &group, nil
}

func (r *Repository) FindGroupByID(groupID uint) (*model.TagGroup, error) {
	var group model.TagGroup
	err := r.db.First(&group, groupID).Error
	if err != nil {
		return nil, err
	}
	return &group, nil
}

func (r *Repository) FindGroupByName(fileID uint, name string) (*model.TagGroup, error) {
	var group model.TagGroup
	err := r.db.Where("file_id = ? AND group_name = ?", fileID, name).First(&group).Error
	if err != nil {
		return nil, err
	}
	return &group, nil
}

func (r *Repository) FindGroups(fileID uint) ([]model.TagGroup, error) {
	var groups []model.TagGroup
	err := r.db.Where("file_id = ?", fileID).Order("group_id").Find(&groups).Error
	return groups, err
}

// FindGroupsWithCounts returns all groups for a file with tag counts in a single query.
func (r *Repository) FindGroupsWithCounts(fileID uint) ([]struct {
	Group    model.TagGroup
	TagCount int64
}, error) {
	type result struct {
		model.TagGroup
		TagCount int64
	}
	var rows []result
	err := r.db.Table("tag_groups").
		Select("tag_groups.*, COALESCE(COUNT(tag_group_tags.id), 0) AS tag_count").
		Joins("LEFT JOIN tag_group_tags ON tag_group_tags.group_id = tag_groups.group_id").
		Where("tag_groups.file_id = ?", fileID).
		Group("tag_groups.group_id").
		Order("tag_groups.group_id").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]struct {
		Group    model.TagGroup
		TagCount int64
	}, len(rows))
	for i, r := range rows {
		out[i].Group = r.TagGroup
		out[i].TagCount = r.TagCount
	}
	return out, nil
}
