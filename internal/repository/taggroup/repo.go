package taggroup

import (
	"github.com/XingfenD/tag_file/internal/model"
	"github.com/XingfenD/tag_file/pkg/storage"
	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository() *Repository {
	return &Repository{db: storage.GetDB()}
}

func (r *Repository) CreateGroup(group *model.TagGroup) error {
	return r.db.Create(group).Error
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

func (r *Repository) DeleteGroup(groupID uint) error {
	r.db.Where("group_id = ?", groupID).Delete(&model.TagGroupTag{})
	return r.db.Delete(&model.TagGroup{}, groupID).Error
}

func (r *Repository) AddTag(groupID uint, fileID uint, tagID uint, tagValue string) error {
	t := model.TagGroupTag{GroupID: groupID, FileID: fileID, TagID: tagID, TagValue: tagValue}
	return r.db.Create(&t).Error
}

func (r *Repository) RemoveTag(groupID uint, tagID uint, tagValue string) error {
	return r.db.Where("group_id = ? AND tag_id = ? AND tag_value = ?", groupID, tagID, tagValue).Delete(&model.TagGroupTag{}).Error
}

func (r *Repository) HasTagValue(groupID uint, tagID uint, tagValue string) (bool, error) {
	var count int64
	err := r.db.Model(&model.TagGroupTag{}).Where("group_id = ? AND tag_id = ? AND tag_value = ?", groupID, tagID, tagValue).Count(&count).Error
	return count > 0, err
}

func (r *Repository) GetTags(groupID uint) ([]model.Tag, error) {
	var tags []model.Tag
	err := r.db.Preload("Category").
		Joins("JOIN tag_group_tags ON tag_group_tags.tag_id = tags.tag_id").
		Where("tag_group_tags.group_id = ?", groupID).
		Group("tags.tag_id").
		Find(&tags).Error
	return tags, err
}

func (r *Repository) GetFileTags(fileID uint, offset, limit int) ([]model.TagGroupTag, int64, error) {
	var total int64
	r.db.Model(&model.TagGroupTag{}).Where("file_id = ?", fileID).Count(&total)
	var fts []model.TagGroupTag
	q := r.db.Preload("Tag.Category").Preload("Group").Where("file_id = ?", fileID).Order("id")
	if limit > 0 {
		q = q.Offset(offset).Limit(limit)
	}
	err := q.Find(&fts).Error
	return fts, total, err
}

func (r *Repository) GetAllFileTagValues(fileID uint) (map[uint][]string, error) {
	var fts []model.TagGroupTag
	err := r.db.Where("file_id = ? AND tag_value != ''", fileID).Find(&fts).Error
	if err != nil {
		return nil, err
	}
	m := make(map[uint][]string)
	for _, ft := range fts {
		m[ft.TagID] = append(m[ft.TagID], ft.TagValue)
	}
	return m, nil
}

func (r *Repository) SearchByTags(tagIDs []uint, logic string) ([]model.File, error) {
	if len(tagIDs) == 0 {
		return nil, nil
	}

	var files []model.File
	switch logic {
	case "and":
		err := r.db.Where("file_id IN (?)",
			r.db.Table("tag_group_tags").
				Select("file_id").
				Where("tag_id IN (?)", tagIDs).
				Group("file_id").
				Having("COUNT(DISTINCT tag_id) = ?", len(tagIDs)),
		).Order("file_path").Find(&files).Error
		return files, err
	case "not":
		err := r.db.Where("file_id NOT IN (?)",
			r.db.Table("tag_group_tags").Select("file_id").Where("tag_id IN (?)", tagIDs),
		).Order("file_path").Find(&files).Error
		return files, err
	default:
		err := r.db.Where("file_id IN (?)",
			r.db.Table("tag_group_tags").Select("file_id").Where("tag_id IN (?)", tagIDs),
		).Order("file_path").Find(&files).Error
		return files, err
	}
}

func (r *Repository) SearchByTagGroups(groups [][]uint) ([]model.File, error) {
	if len(groups) == 0 {
		return nil, nil
	}

	query := r.db
	for _, group := range groups {
		if len(group) == 0 {
			continue
		}
		query = query.Where("file_id IN (?)",
			r.db.Table("tag_group_tags").Select("file_id").Where("tag_id IN (?)", group),
		)
	}

	var files []model.File
	err := query.Order("file_path").Find(&files).Error
	return files, err
}
