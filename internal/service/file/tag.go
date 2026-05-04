package file

import (
	"fmt"
	"log"

	"github.com/XingfenD/tag_file/internal/model"
)

func (s *Service) TagFile(filePath string, tagName string, tagValue string) error {
	file, err := s.fileRepo.FindByPath(filePath)
	if err != nil {
		log.Printf("[file] TagFile: find file: %v", err)
		return fmt.Errorf("查询文件失败: %w", err)
	}
	if file == nil {
		return fmt.Errorf("文件 '%s' 未注册，请先 scan", filePath)
	}

	tags, err := s.tagRepo.FindByName(tagName)
	if err != nil {
		log.Printf("[file] TagFile: find tag: %v", err)
		return fmt.Errorf("查询标签失败: %w", err)
	}
	if len(tags) == 0 {
		return fmt.Errorf("标签 '%s' 不存在", tagName)
	}

	group, err := s.tagGroupRepo.FindDefaultGroup(file.FileID)
	if err != nil {
		log.Printf("[file] TagFile: find default group: %v", err)
		s.createDefaultGroup(file.FileID)
		group, err = s.tagGroupRepo.FindDefaultGroup(file.FileID)
		if err != nil {
			return fmt.Errorf("获取默认标签组失败: %w", err)
		}
	}

	for _, tag := range tags {
		has, err := s.tagGroupRepo.HasTagValue(group.GroupID, tag.TagID, tagValue)
		if err != nil {
			log.Printf("[file] TagFile: check tag: %v", err)
			return fmt.Errorf("查询关联失败: %w", err)
		}
		if has {
			if tagValue != "" {
				return fmt.Errorf("文件已存在标签 '%s:%s'", tagName, tagValue)
			}
			return fmt.Errorf("文件已存在标签 '%s'", tagName)
		}
		return s.tagGroupRepo.AddTag(group.GroupID, file.FileID, tag.TagID, tagValue)
	}
	return fmt.Errorf("标签 '%s' 不存在", tagName)
}

func (s *Service) UntagFile(filePath string, tagName string, tagValue string) error {
	file, err := s.fileRepo.FindByPath(filePath)
	if err != nil {
		log.Printf("[file] UntagFile: find file: %v", err)
		return fmt.Errorf("查询文件失败: %w", err)
	}
	if file == nil {
		return fmt.Errorf("文件 '%s' 未注册", filePath)
	}

	tags, err := s.tagRepo.FindByName(tagName)
	if err != nil {
		log.Printf("[file] UntagFile: find tag: %v", err)
		return fmt.Errorf("查询标签失败: %w", err)
	}
	if len(tags) == 0 {
		return fmt.Errorf("标签 '%s' 不存在", tagName)
	}

	group, err := s.tagGroupRepo.FindDefaultGroup(file.FileID)
	if err != nil {
		return fmt.Errorf("默认标签组不存在")
	}

	for _, tag := range tags {
		has, err := s.tagGroupRepo.HasTagValue(group.GroupID, tag.TagID, tagValue)
		if err != nil {
			log.Printf("[file] UntagFile: check tag: %v", err)
			return fmt.Errorf("查询关联失败: %w", err)
		}
		if has {
			return s.tagGroupRepo.RemoveTag(group.GroupID, tag.TagID, tagValue)
		}
	}
	return fmt.Errorf("文件不存在标签 '%s'", tagName)
}

func (s *Service) createDefaultGroup(fileID uint) {
	s.tagGroupRepo.CreateGroup(&model.TagGroup{
		FileID:    fileID,
		GroupName: "default",
		IsDefault: true,
	})
}

func (s *Service) UpdateTagValue(ftID uint, val string) error {
	return s.tagGroupRepo.UpdateTagValue(ftID, val)
}
