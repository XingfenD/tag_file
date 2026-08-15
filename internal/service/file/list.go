package file

import (
	"fmt"
	"log"

	"github.com/XingfenD/tag_file/internal/model"
)

type FileWithTags struct {
	File   model.File
	Tags   []model.Tag
	ValMap map[uint][]string
}

func (s *Service) FindByPath(path string) (*model.File, error) {
	return s.fileRepo.Find().ByPath(path).Exec()
}

func (s *Service) FindByID(id uint) (*model.File, error) {
	return s.fileRepo.Find().ByID(id).Exec()
}

func (s *Service) ListFilesByDir(dir string) ([]model.File, error) {
	return s.fileRepo.ListByDir(dir)
}

func (s *Service) ListFiles(offset, limit int) ([]model.File, int64, error) {
	list, total, err := s.fileRepo.List().Offset(offset).Limit(limit).Exec()
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
		log.Printf("[file] GetFileInfo: find default group: %v", err)
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

func (s *Service) SearchFilesWithTags(keyword string) ([]FileWithTags, error) {
	files, _, err := s.SearchFiles(keyword, 0, 0)
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, nil
	}

	fileIDs := make([]uint, len(files))
	for i, f := range files {
		fileIDs[i] = f.FileID
	}

	groups, err := s.tagGroupRepo.FindDefaultGroupsBatch(fileIDs)
	if err != nil {
		log.Printf("[file] SearchFilesWithTags: batch default groups: %v", err)
		result := make([]FileWithTags, len(files))
		for i, f := range files {
			result[i] = FileWithTags{File: f}
		}
		return result, nil
	}

	groupIDs := make([]uint, 0, len(groups))
	fileGroupMap := make(map[uint]uint)
	for _, g := range groups {
		groupIDs = append(groupIDs, g.GroupID)
		fileGroupMap[g.FileID] = g.GroupID
	}

	tagsByGroup := make(map[uint][]model.Tag)
	if len(groupIDs) > 0 {
		tagsByGroup, err = s.tagGroupRepo.TagsByGroupIDs(groupIDs)
		if err != nil {
			log.Printf("[file] SearchFilesWithTags: batch tags: %v", err)
		}
	}

	result := make([]FileWithTags, len(files))
	for i, f := range files {
		result[i] = FileWithTags{File: f}
		if gid, ok := fileGroupMap[f.FileID]; ok {
			result[i].Tags = tagsByGroup[gid]
		}
	}
	return result, nil
}
