package search

import (
	"fmt"

	"github.com/XingfenD/tag_file/internal/model"
	repoFile "github.com/XingfenD/tag_file/internal/repository/file"
	repoTag "github.com/XingfenD/tag_file/internal/repository/tag"
)

type Service struct {
	fileRepo *repoFile.Repository
	tagRepo  *repoTag.Repository
}

func NewService() *Service {
	return &Service{
		fileRepo: repoFile.NewRepository(),
		tagRepo:  repoTag.NewRepository(),
	}
}

type FileWithTags struct {
	File model.File
	Tags []model.Tag
}

func (s *Service) resolveTagIDs(tagNames []string) ([]uint, error) {
	var allIDs []uint
	for _, name := range tagNames {
		tags, err := s.tagRepo.FindByName(name)
		if err != nil {
			return nil, fmt.Errorf("查询标签 '%s' 失败: %w", name, err)
		}
		if len(tags) == 0 {
			return nil, fmt.Errorf("标签 '%s' 不存在", name)
		}
		for _, t := range tags {
			allIDs = append(allIDs, t.TagID)
		}
	}
	return allIDs, nil
}

func (s *Service) resolveTagIDsWithDescendants(tagNames []string) ([]uint, error) {
	var allIDs []uint
	for _, name := range tagNames {
		tags, err := s.tagRepo.FindByName(name)
		if err != nil {
			return nil, fmt.Errorf("查询标签 '%s' 失败: %w", name, err)
		}
		if len(tags) == 0 {
			return nil, fmt.Errorf("标签 '%s' 不存在", name)
		}
		for _, t := range tags {
			descIDs, err := s.tagRepo.FindDescendantIDs(t.TagID)
			if err != nil {
				return nil, err
			}
			allIDs = append(allIDs, descIDs...)
		}
	}
	return allIDs, nil
}

func (s *Service) resolveTagGroupsWithDescendants(tagNames []string) ([][]uint, error) {
	groups := make([][]uint, 0, len(tagNames))
	for _, name := range tagNames {
		tags, err := s.tagRepo.FindByName(name)
		if err != nil {
			return nil, fmt.Errorf("查询标签 '%s' 失败: %w", name, err)
		}
		if len(tags) == 0 {
			return nil, fmt.Errorf("标签 '%s' 不存在", name)
		}
		var group []uint
		for _, t := range tags {
			descIDs, err := s.tagRepo.FindDescendantIDs(t.TagID)
			if err != nil {
				return nil, err
			}
			group = append(group, descIDs...)
		}
		groups = append(groups, group)
	}
	return groups, nil
}

func (s *Service) SearchByTags(include []string, exclude []string, logic string) ([]FileWithTags, error) {
	if len(include) == 0 && len(exclude) == 0 {
		return nil, fmt.Errorf("请指定搜索条件")
	}

	var files []model.File

	if len(include) > 0 && len(exclude) == 0 {
		if logic == "and" {
			groups, err := s.resolveTagGroupsWithDescendants(include)
			if err != nil {
				return nil, err
			}
			files, err = s.fileRepo.SearchByTagGroups(groups)
			if err != nil {
				return nil, fmt.Errorf("搜索失败: %w", err)
			}
		} else {
			incIDs, err := s.resolveTagIDsWithDescendants(include)
			if err != nil {
				return nil, err
			}
			files, err = s.fileRepo.SearchByTags(incIDs, logic)
			if err != nil {
				return nil, fmt.Errorf("搜索失败: %w", err)
			}
		}
	} else if len(include) == 0 && len(exclude) > 0 {
		excIDs, err := s.resolveTagIDsWithDescendants(exclude)
		if err != nil {
			return nil, err
		}
		files, err = s.fileRepo.SearchByTags(excIDs, "not")
		if err != nil {
			return nil, fmt.Errorf("搜索失败: %w", err)
		}
	} else {
		excIDs, err := s.resolveTagIDsWithDescendants(exclude)
		if err != nil {
			return nil, err
		}

		if logic == "and" {
			groups, err := s.resolveTagGroupsWithDescendants(include)
			if err != nil {
				return nil, err
			}
			files, err = s.fileRepo.SearchByTagGroups(groups)
		} else {
			incIDs, err := s.resolveTagIDsWithDescendants(include)
			if err != nil {
				return nil, err
			}
			files, err = s.fileRepo.SearchByTags(incIDs, logic)
		}
		if err != nil {
			return nil, fmt.Errorf("搜索失败: %w", err)
		}

		excFiles, err := s.fileRepo.SearchByTags(excIDs, "or")
		if err != nil {
			return nil, fmt.Errorf("搜索失败: %w", err)
		}

		excMap := make(map[uint]bool)
		for _, f := range excFiles {
			excMap[f.FileID] = true
		}

		var filtered []model.File
		for _, f := range files {
			if !excMap[f.FileID] {
				filtered = append(filtered, f)
			}
		}
		files = filtered
	}

	result := make([]FileWithTags, 0, len(files))
	for _, f := range files {
		tags, err := s.fileRepo.GetTags(f.FileID)
		if err != nil {
			continue
		}
		result = append(result, FileWithTags{File: f, Tags: tags})
	}
	return result, nil
}
