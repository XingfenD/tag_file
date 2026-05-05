package file

import (
	"fmt"
	"log"
	"time"

	"github.com/XingfenD/tag_file/internal/model"
)

func (s *Service) ListFiles(offset, limit int) ([]model.File, int64, error) {
	start := time.Now()
	list, total, err := s.fileRepo.List().Offset(offset).Limit(limit).Exec()
	if elapsed := time.Since(start); elapsed > 50*time.Millisecond {
		log.Printf("[perf] file.ListFiles(offset=%d,limit=%d): %v, total=%d", offset, limit, elapsed, total)
	}
	if err != nil {
		log.Printf("[file] ListFiles: %v", err)
		return nil, 0, fmt.Errorf("查询文件列表失败: %w", err)
	}
	return list, total, nil
}

func (s *Service) IsIndexed(path string) (bool, error) {
	file, err := s.fileRepo.Find().ByPath(path).Exec()
	if err != nil {
		log.Printf("[file] IsIndexed: %v", err)
		return false, fmt.Errorf("查询文件失败: %w", err)
	}
	return file != nil, nil
}

func (s *Service) GetFileInfo(path string) (*model.File, []model.Tag, map[uint][]string, error) {
	file, err := s.fileRepo.Find().ByPath(path).Exec()
	if err != nil {
		log.Printf("[file] GetFileInfo: find file: %v", err)
		return nil, nil, nil, fmt.Errorf("查询文件失败: %w", err)
	}
	if file == nil {
		return nil, nil, nil, fmt.Errorf("文件 '%s' 未注册", path)
	}
	group, err := s.tagGroupRepo.FindDefaultGroup(file.FileID)
	if err != nil {
		return file, nil, nil, nil
	}
	tags, err := s.tagGroupRepo.GetTags(group.GroupID)
	if err != nil {
		log.Printf("[file] GetFileInfo: get tags: %v", err)
		return file, nil, nil, nil
	}
	valMap, err := s.tagGroupRepo.GetAllFileTagValues(file.FileID)
	if err != nil {
		valMap = make(map[uint][]string)
	}
	return file, tags, valMap, nil
}

func (s *Service) ListFileTags(path string, offset, limit int) ([]model.TagGroupTag, int64, error) {
	file, err := s.fileRepo.Find().ByPath(path).Exec()
	if err != nil {
		return nil, 0, fmt.Errorf("查询文件失败: %w", err)
	}
	if file == nil {
		return nil, 0, fmt.Errorf("文件 '%s' 未注册", path)
	}
	return s.tagGroupRepo.GetFileTags(file.FileID, offset, limit)
}
