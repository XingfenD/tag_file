package file

import (
	"fmt"
	"log"

	"github.com/XingfenD/tag_file/internal/model"
)

func (s *Service) ListFiles(offset, limit int) ([]model.File, int64, error) {
	list, total, err := s.fileRepo.List(offset, limit)
	if err != nil {
		log.Printf("[file] ListFiles: %v", err)
		return nil, 0, fmt.Errorf("查询文件列表失败: %w", err)
	}
	return list, total, nil
}

func (s *Service) IsIndexed(path string) (bool, error) {
	file, err := s.fileRepo.FindByPath(path)
	if err != nil {
		log.Printf("[file] IsIndexed: %v", err)
		return false, fmt.Errorf("查询文件失败: %w", err)
	}
	return file != nil, nil
}

func (s *Service) GetFileInfo(path string) (*model.File, []model.Tag, map[uint][]string, error) {
	file, err := s.fileRepo.FindByPath(path)
	if err != nil {
		log.Printf("[file] GetFileInfo: find file: %v", err)
		return nil, nil, nil, fmt.Errorf("查询文件失败: %w", err)
	}
	if file == nil {
		return nil, nil, nil, fmt.Errorf("文件 '%s' 未注册", path)
	}
	tags, err := s.fileRepo.GetTags(file.FileID)
	if err != nil {
		log.Printf("[file] GetFileInfo: get tags: %v", err)
		return nil, nil, nil, fmt.Errorf("查询标签失败: %w", err)
	}
	valMap, err := s.fileRepo.GetAllTagValues(file.FileID)
	if err != nil {
		valMap = make(map[uint][]string)
	}
	return file, tags, valMap, nil
}

func (s *Service) ListFileTags(path string, offset, limit int) ([]model.FileTag, int64, error) {
	file, err := s.fileRepo.FindByPath(path)
	if err != nil {
		return nil, 0, fmt.Errorf("查询文件失败: %w", err)
	}
	if file == nil {
		return nil, 0, fmt.Errorf("文件 '%s' 未注册", path)
	}
	return s.fileRepo.GetFileTags(file.FileID, offset, limit)
}
