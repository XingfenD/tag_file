package file

import (
	"fmt"
)

func (s *Service) TagFile(filePath string, tagName string, tagValue string) error {
	file, err := s.fileRepo.FindByPath(filePath)
	if err != nil {
		return fmt.Errorf("查询文件失败: %w", err)
	}
	if file == nil {
		return fmt.Errorf("文件 '%s' 未注册，请先 scan", filePath)
	}

	tags, err := s.tagRepo.FindByName(tagName)
	if err != nil {
		return fmt.Errorf("查询标签失败: %w", err)
	}
	if len(tags) == 0 {
		return fmt.Errorf("标签 '%s' 不存在", tagName)
	}

	for _, tag := range tags {
		if tagValue != "" {
			has, err := s.fileRepo.HasTagValue(file.FileID, tag.TagID, tagValue)
			if err != nil {
				return fmt.Errorf("查询关联失败: %w", err)
			}
			if has {
				return fmt.Errorf("文件已存在标签 '%s:%s'", tagName, tagValue)
			}
			return s.fileRepo.AddTag(file.FileID, tag.TagID, tagValue)
		}
		has, err := s.fileRepo.HasTag(file.FileID, tag.TagID)
		if err != nil {
			return fmt.Errorf("查询关联失败: %w", err)
		}
		if has {
			return fmt.Errorf("文件已存在标签 '%s'", tagName)
		}
		return s.fileRepo.AddTag(file.FileID, tag.TagID, "")
	}
	return fmt.Errorf("标签 '%s' 不存在", tagName)
}

func (s *Service) UntagFile(filePath string, tagName string, tagValue string) error {
	file, err := s.fileRepo.FindByPath(filePath)
	if err != nil {
		return fmt.Errorf("查询文件失败: %w", err)
	}
	if file == nil {
		return fmt.Errorf("文件 '%s' 未注册", filePath)
	}

	tags, err := s.tagRepo.FindByName(tagName)
	if err != nil {
		return fmt.Errorf("查询标签失败: %w", err)
	}
	if len(tags) == 0 {
		return fmt.Errorf("标签 '%s' 不存在", tagName)
	}

	for _, tag := range tags {
		has, err := s.fileRepo.HasTag(file.FileID, tag.TagID)
		if err != nil {
			return fmt.Errorf("查询关联失败: %w", err)
		}
		if has {
			return s.fileRepo.RemoveTag(file.FileID, tag.TagID, tagValue)
		}
	}
	return fmt.Errorf("文件不存在标签 '%s'", tagName)
}
