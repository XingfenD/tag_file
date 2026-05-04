package file

import (
	"fmt"
	"log"

	"github.com/XingfenD/tag_file/internal/model"
)

func (s *Service) ListGroups(filePath string) ([]model.TagGroup, error) {
	file, err := s.fileRepo.FindByPath(filePath)
	if err != nil {
		return nil, fmt.Errorf("查询文件失败: %w", err)
	}
	if file == nil {
		return nil, fmt.Errorf("文件 '%s' 未注册", filePath)
	}
	return s.tagGroupRepo.FindGroups(file.FileID)
}

func (s *Service) CreateGroup(filePath string, groupName string) (*model.TagGroup, error) {
	file, err := s.fileRepo.FindByPath(filePath)
	if err != nil {
		return nil, fmt.Errorf("查询文件失败: %w", err)
	}
	if file == nil {
		return nil, fmt.Errorf("文件 '%s' 未注册", filePath)
	}

	existing, _ := s.tagGroupRepo.FindGroupByName(file.FileID, groupName)
	if existing != nil {
		return nil, fmt.Errorf("标签组 '%s' 已存在", groupName)
	}

	group := &model.TagGroup{
		FileID:    file.FileID,
		GroupName: groupName,
		IsDefault: false,
	}
	if err := s.tagGroupRepo.CreateGroup(group); err != nil {
		return nil, fmt.Errorf("创建标签组失败: %w", err)
	}
	return group, nil
}

func (s *Service) DeleteGroup(filePath string, groupID uint) error {
	file, err := s.fileRepo.FindByPath(filePath)
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
	file, err := s.fileRepo.FindByPath(filePath)
	if err != nil {
		return nil, 0, fmt.Errorf("查询文件失败: %w", err)
	}
	if file == nil {
		return nil, 0, fmt.Errorf("文件 '%s' 未注册", filePath)
	}
	return s.tagGroupRepo.GetFileTagsByGroup(file.FileID, groupID, offset, limit)
}

func (s *Service) TagFileToGroup(filePath string, tagName string, tagValue string, groupID uint) error {
	file, err := s.fileRepo.FindByPath(filePath)
	if err != nil {
		log.Printf("[file] TagFileToGroup: find file: %v", err)
		return fmt.Errorf("查询文件失败: %w", err)
	}
	if file == nil {
		return fmt.Errorf("文件 '%s' 未注册，请先 scan", filePath)
	}

	tags, err := s.tagRepo.FindByName(tagName)
	if err != nil {
		log.Printf("[file] TagFileToGroup: find tag: %v", err)
		return fmt.Errorf("查询标签失败: %w", err)
	}
	if len(tags) == 0 {
		return fmt.Errorf("标签 '%s' 不存在", tagName)
	}

	for _, tag := range tags {
		has, err := s.tagGroupRepo.HasTagValue(groupID, tag.TagID, tagValue)
		if err != nil {
			log.Printf("[file] TagFileToGroup: check tag: %v", err)
			return fmt.Errorf("查询关联失败: %w", err)
		}
		if has {
			if tagValue != "" {
				return fmt.Errorf("标签组中已存在标签 '%s:%s'", tagName, tagValue)
			}
			return fmt.Errorf("标签组中已存在标签 '%s'", tagName)
		}
		return s.tagGroupRepo.AddTag(groupID, file.FileID, tag.TagID, tagValue)
	}
	return fmt.Errorf("标签 '%s' 不存在", tagName)
}

func (s *Service) UntagFileFromGroup(filePath string, tagName string, tagValue string, groupID uint) error {
	file, err := s.fileRepo.FindByPath(filePath)
	if err != nil {
		log.Printf("[file] UntagFileFromGroup: find file: %v", err)
		return fmt.Errorf("查询文件失败: %w", err)
	}
	if file == nil {
		return fmt.Errorf("文件 '%s' 未注册", filePath)
	}

	tags, err := s.tagRepo.FindByName(tagName)
	if err != nil {
		log.Printf("[file] UntagFileFromGroup: find tag: %v", err)
		return fmt.Errorf("查询标签失败: %w", err)
	}
	if len(tags) == 0 {
		return fmt.Errorf("标签 '%s' 不存在", tagName)
	}

	for _, tag := range tags {
		has, err := s.tagGroupRepo.HasTagValue(groupID, tag.TagID, tagValue)
		if err != nil {
			log.Printf("[file] UntagFileFromGroup: check tag: %v", err)
			return fmt.Errorf("查询关联失败: %w", err)
		}
		if has {
			return s.tagGroupRepo.RemoveTag(groupID, tag.TagID, tagValue)
		}
	}
	return fmt.Errorf("标签组中不存在标签 '%s'", tagName)
}
