package taggroup

import "github.com/XingfenD/tag_file/internal/model"

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
