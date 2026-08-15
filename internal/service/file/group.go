package file

import (
	"fmt"
	"log"

	"github.com/XingfenD/tag_file/internal/model"
)

func (s *Service) ListGroups(filePath string) ([]model.TagGroup, error) {
	file, err := s.fileRepo.Find().ByPath(filePath).Exec()
	if err != nil {
		return nil, fmt.Errorf("查询文件失败: %w", err)
	}
	if file == nil {
		return nil, fmt.Errorf("文件 '%s' 未注册", filePath)
	}
	return s.tagGroupRepo.FindGroups(file.FileID)
}

// GroupWithCount groups a TagGroup with its tag count (single-query aggregation).
type GroupWithCount struct {
	Group    model.TagGroup
	TagCount int64
}

func (s *Service) ListGroupsWithCounts(filePath string) ([]GroupWithCount, error) {
	file, err := s.fileRepo.Find().ByPath(filePath).Exec()
	if err != nil {
		return nil, fmt.Errorf("查询文件失败: %w", err)
	}
	if file == nil {
		return nil, fmt.Errorf("文件 '%s' 未注册", filePath)
	}
	rows, err := s.tagGroupRepo.FindGroupsWithCounts(file.FileID)
	if err != nil {
		return nil, fmt.Errorf("查询标签组失败: %w", err)
	}
	result := make([]GroupWithCount, len(rows))
	for i, r := range rows {
		result[i] = GroupWithCount{Group: r.Group, TagCount: r.TagCount}
	}
	return result, nil
}

func (s *Service) CreateGroup(filePath string, groupName string) (*model.TagGroup, error) {
	file, err := s.fileRepo.Find().ByPath(filePath).Exec()
	if err != nil {
		return nil, fmt.Errorf("查询文件失败: %w", err)
	}
	if file == nil {
		return nil, fmt.Errorf("文件 '%s' 未注册", filePath)
	}

	existing, err := s.tagGroupRepo.FindGroupByName(file.FileID, groupName)
	if err != nil {
		return nil, fmt.Errorf("查询标签组失败: %w", err)
	}
	if existing != nil {
		return nil, fmt.Errorf("标签组 '%s' 已存在", groupName)
	}

	group := &model.TagGroup{
		FileID:    file.FileID,
		GroupName: groupName,
		IsDefault: false,
	}
	if err := s.tagGroupRepo.CreateGroup(group); err != nil {
		// Handle race condition: unique constraint violation
		existing, findErr := s.tagGroupRepo.FindGroupByName(file.FileID, groupName)
		if findErr == nil && existing != nil {
			return nil, fmt.Errorf("标签组 '%s' 已存在", groupName)
		}
		return nil, fmt.Errorf("创建标签组失败: %w", err)
	}
	return group, nil
}

func (s *Service) DeleteGroup(filePath string, groupID uint) error {
	file, err := s.fileRepo.Find().ByPath(filePath).Exec()
	if err != nil {
		return fmt.Errorf("查询文件失败: %w", err)
	}
	if file == nil {
		return fmt.Errorf("文件 '%s' 未注册", filePath)
	}

	group, err := s.tagGroupRepo.FindGroupByID(groupID)
	if err != nil {
		return fmt.Errorf("标签组不存在: %w", err)
	}
	if group.FileID != file.FileID {
		return fmt.Errorf("标签组不属于该文件")
	}
	if group.IsDefault {
		return fmt.Errorf("不能删除默认标签组")
	}

	return s.tagGroupRepo.DeleteGroup(groupID)
}

func (s *Service) ListFileTagsByGroup(filePath string, groupID uint, offset, limit int) ([]model.TagGroupTag, int64, error) {
	file, err := s.fileRepo.Find().ByPath(filePath).Exec()
	if err != nil {
		return nil, 0, fmt.Errorf("查询文件失败: %w", err)
	}
	if file == nil {
		return nil, 0, fmt.Errorf("文件 '%s' 未注册", filePath)
	}
	return s.tagGroupRepo.GetFileTagsByGroup(file.FileID, groupID, offset, limit)
}

func (s *Service) TagFileToGroup(filePath string, tagID uint, tagValue string, groupID uint) error {
	file, err := s.fileRepo.Find().ByPath(filePath).Exec()
	if err != nil {
		log.Printf("[file] TagFileToGroup: find file: %v", err)
		return fmt.Errorf("查询文件失败: %w", err)
	}
	if file == nil {
		return fmt.Errorf("文件 '%s' 未注册，请先 scan", filePath)
	}

	tag, err := s.tagRepo.Find().ByID(tagID).Exec()
	if err != nil {
		log.Printf("[file] TagFileToGroup: find tag: %v", err)
		return fmt.Errorf("查询标签失败: %w", err)
	}
	if tag == nil {
		return fmt.Errorf("标签 %d 不存在", tagID)
	}

	has, err := s.tagGroupRepo.HasTagValue(groupID, tag.TagID, tagValue)
	if err != nil {
		log.Printf("[file] TagFileToGroup: check tag: %v", err)
		return fmt.Errorf("查询关联失败: %w", err)
	}
	if has {
		if tagValue != "" {
			return fmt.Errorf("标签组中已存在标签 '%s:%s'", tag.TagName, tagValue)
		}
		return fmt.Errorf("标签组中已存在标签 '%s'", tag.TagName)
	}

	if err := s.tagGroupRepo.AddTag(groupID, file.FileID, tag.TagID, tagValue); err != nil {
		return err
	}
	s.incrTagCnt(file.FileID)
	return nil
}

func (s *Service) UntagFileFromGroup(filePath string, tagID uint, tagValue string, groupID uint) error {
	file, err := s.fileRepo.Find().ByPath(filePath).Exec()
	if err != nil {
		log.Printf("[file] UntagFileFromGroup: find file: %v", err)
		return fmt.Errorf("查询文件失败: %w", err)
	}
	if file == nil {
		return fmt.Errorf("文件 '%s' 未注册", filePath)
	}

	tag, err := s.tagRepo.Find().ByID(tagID).Exec()
	if err != nil {
		log.Printf("[file] UntagFileFromGroup: find tag: %v", err)
		return fmt.Errorf("查询标签失败: %w", err)
	}
	if tag == nil {
		return fmt.Errorf("标签 %d 不存在", tagID)
	}

	has, err := s.tagGroupRepo.HasTagValue(groupID, tag.TagID, tagValue)
	if err != nil {
		log.Printf("[file] UntagFileFromGroup: check tag: %v", err)
		return fmt.Errorf("查询关联失败: %w", err)
	}
	if has {
		if err := s.tagGroupRepo.RemoveTag(groupID, tag.TagID, tagValue); err != nil {
			return err
		}
		s.decrTagCnt(file.FileID)
		return nil
	}
	return fmt.Errorf("标签组中不存在标签 '%s'", tag.TagName)
}

func (s *Service) RenameGroup(filePath string, groupID uint, newName string) error {
	file, err := s.fileRepo.Find().ByPath(filePath).Exec()
	if err != nil {
		return fmt.Errorf("查询文件失败: %w", err)
	}
	if file == nil {
		return fmt.Errorf("文件 '%s' 未注册", filePath)
	}

	group, err := s.tagGroupRepo.FindGroupByID(groupID)
	if err != nil {
		return fmt.Errorf("标签组不存在: %w", err)
	}
	if group.FileID != file.FileID {
		return fmt.Errorf("标签组不属于该文件")
	}

	existing, _ := s.tagGroupRepo.FindGroupByName(file.FileID, newName)
	if existing != nil && existing.GroupID != groupID {
		return fmt.Errorf("标签组 '%s' 已存在", newName)
	}

	return s.tagGroupRepo.UpdateGroup(group).SetName(newName).Exec()
}
