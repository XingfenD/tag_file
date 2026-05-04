package file

import (
	"fmt"
	"log"

	"github.com/XingfenD/tag_file/internal/model"
)

func (s *Service) TagFile(filePath string, tagID uint, tagValue string) error {
	file, err := s.fileRepo.Find().ByPath(filePath).Exec()
	if err != nil {
		log.Printf("[file] TagFile: find file: %v", err)
		return fmt.Errorf("查询文件失败: %w", err)
	}
	if file == nil {
		return fmt.Errorf("文件 '%s' 未注册，请先 scan", filePath)
	}

	tag, err := s.tagRepo.Find().ByID(tagID).Exec()
	if err != nil {
		log.Printf("[file] TagFile: find tag: %v", err)
		return fmt.Errorf("查询标签失败: %w", err)
	}
	if tag == nil {
		return fmt.Errorf("标签 %d 不存在", tagID)
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

	has, err := s.tagGroupRepo.HasTagValue(group.GroupID, tag.TagID, tagValue)
	if err != nil {
		log.Printf("[file] TagFile: check tag: %v", err)
		return fmt.Errorf("查询关联失败: %w", err)
	}
	if has {
		if tagValue != "" {
			return fmt.Errorf("文件已存在标签 '%s:%s'", tag.TagName, tagValue)
		}
		return fmt.Errorf("文件已存在标签 '%s'", tag.TagName)
	}

	if err := s.tagGroupRepo.AddTag(group.GroupID, file.FileID, tag.TagID, tagValue); err != nil {
		return err
	}
	s.incrTagCnt(file.FileID)
	return nil
}

func (s *Service) incrTagCnt(fileID uint) {
	s.fileRepo.IncrementTagCnt(fileID, 1)
}

func (s *Service) decrTagCnt(fileID uint) {
	s.fileRepo.IncrementTagCnt(fileID, -1)
}

func (s *Service) UntagFile(filePath string, tagID uint, tagValue string) error {
	file, err := s.fileRepo.Find().ByPath(filePath).Exec()
	if err != nil {
		log.Printf("[file] UntagFile: find file: %v", err)
		return fmt.Errorf("查询文件失败: %w", err)
	}
	if file == nil {
		return fmt.Errorf("文件 '%s' 未注册", filePath)
	}

	tag, err := s.tagRepo.Find().ByID(tagID).Exec()
	if err != nil {
		log.Printf("[file] UntagFile: find tag: %v", err)
		return fmt.Errorf("查询标签失败: %w", err)
	}
	if tag == nil {
		return fmt.Errorf("标签 %d 不存在", tagID)
	}

	group, err := s.tagGroupRepo.FindDefaultGroup(file.FileID)
	if err != nil {
		return fmt.Errorf("默认标签组不存在")
	}

	has, err := s.tagGroupRepo.HasTagValue(group.GroupID, tag.TagID, tagValue)
	if err != nil {
		log.Printf("[file] UntagFile: check tag: %v", err)
		return fmt.Errorf("查询关联失败: %w", err)
	}
	if !has {
		return fmt.Errorf("文件不存在标签 '%s'", tag.TagName)
	}

	if err := s.tagGroupRepo.RemoveTag(group.GroupID, tag.TagID, tagValue); err != nil {
		return err
	}
	s.decrTagCnt(file.FileID)
	return nil
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
