package search

import (
	"fmt"
	"log"

	"github.com/XingfenD/tag_file/internal/model"
	repoTag "github.com/XingfenD/tag_file/internal/repository/tag"
	repoTagGroup "github.com/XingfenD/tag_file/internal/repository/taggroup"
)

type Service struct {
	tagRepo      *repoTag.Repository
	tagGroupRepo *repoTagGroup.Repository
}

func NewService() *Service {
	return &Service{
		tagRepo:      repoTag.NewRepository(),
		tagGroupRepo: repoTagGroup.NewRepository(),
	}
}

type FileWithTags struct {
	File model.File
	Tags []model.Tag
}

func (s *Service) resolveTagIDs(tagNames []string) ([]uint, error) {
	tags, err := s.tagRepo.FindByNames(tagNames)
	if err != nil {
		log.Printf("[search] resolveTagIDs: %v", err)
		return nil, fmt.Errorf("查询标签失败: %w", err)
	}
	found := make(map[string]bool)
	for _, t := range tags {
		found[t.TagName] = true
	}
	for _, name := range tagNames {
		if !found[name] {
			return nil, fmt.Errorf("标签 '%s' 不存在", name)
		}
	}
	allIDs := make([]uint, len(tags))
	for i, t := range tags {
		allIDs[i] = t.TagID
	}
	return allIDs, nil
}

func (s *Service) resolveTagIDsWithDescendants(tagNames []string) ([]uint, error) {
	tags, err := s.tagRepo.FindByNames(tagNames)
	if err != nil {
		return nil, fmt.Errorf("查询标签失败: %w", err)
	}
	found := make(map[string]bool)
	for _, t := range tags {
		found[t.TagName] = true
	}
	for _, name := range tagNames {
		if !found[name] {
			return nil, fmt.Errorf("标签 '%s' 不存在", name)
		}
	}

	childrenMap, err := s.buildTagChildrenMap()
	if err != nil {
		return nil, err
	}

	var allIDs []uint
	for _, t := range tags {
		allIDs = append(allIDs, collectDescendantIDs(t.TagID, childrenMap)...)
	}
	return allIDs, nil
}

func (s *Service) resolveTagGroupsWithDescendants(tagNames []string) ([][]uint, error) {
	tags, err := s.tagRepo.FindByNames(tagNames)
	if err != nil {
		return nil, fmt.Errorf("查询标签失败: %w", err)
	}
	found := make(map[string]bool)
	for _, t := range tags {
		found[t.TagName] = true
	}
	for _, name := range tagNames {
		if !found[name] {
			return nil, fmt.Errorf("标签 '%s' 不存在", name)
		}
	}

	childrenMap, err := s.buildTagChildrenMap()
	if err != nil {
		return nil, err
	}

	nameToIDs := make(map[string][]uint)
	for _, t := range tags {
		nameToIDs[t.TagName] = append(nameToIDs[t.TagName], t.TagID)
	}

	groups := make([][]uint, 0, len(tagNames))
	for _, name := range tagNames {
		var group []uint
		for _, id := range nameToIDs[name] {
			group = append(group, collectDescendantIDs(id, childrenMap)...)
		}
		groups = append(groups, group)
	}
	return groups, nil
}

func (s *Service) buildTagChildrenMap() (map[uint][]uint, error) {
	allTags, _, err := s.tagRepo.List().Offset(0).Limit(0).Exec()
	if err != nil {
		return nil, fmt.Errorf("查询标签列表失败: %w", err)
	}
	m := make(map[uint][]uint)
	for _, t := range allTags {
		if t.ParentID != nil {
			m[*t.ParentID] = append(m[*t.ParentID], t.TagID)
		}
	}
	return m, nil
}

func collectDescendantIDs(tagID uint, childrenMap map[uint][]uint) []uint {
	ids := []uint{tagID}
	queue := make([]uint, len(childrenMap[tagID]))
	copy(queue, childrenMap[tagID])
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		ids = append(ids, id)
		queue = append(queue, childrenMap[id]...)
	}
	return ids
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
			files, err = s.tagGroupRepo.SearchByTagGroups(groups)
			if err != nil {
				log.Printf("[search] SearchByTags: groups: %v", err)
				return nil, fmt.Errorf("搜索失败: %w", err)
			}
		} else {
			incIDs, err := s.resolveTagIDsWithDescendants(include)
			if err != nil {
				return nil, err
			}
			files, err = s.tagGroupRepo.SearchByTags(incIDs, logic)
			if err != nil {
				log.Printf("[search] SearchByTags: or: %v", err)
				return nil, fmt.Errorf("搜索失败: %w", err)
			}
		}
	} else if len(include) == 0 && len(exclude) > 0 {
		excIDs, err := s.resolveTagIDsWithDescendants(exclude)
		if err != nil {
			return nil, err
		}
		files, err = s.tagGroupRepo.SearchByTags(excIDs, "not")
		if err != nil {
			log.Printf("[search] SearchByTags: not: %v", err)
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
			files, err = s.tagGroupRepo.SearchByTagGroups(groups)
		} else {
			incIDs, err := s.resolveTagIDsWithDescendants(include)
			if err != nil {
				return nil, err
			}
			files, err = s.tagGroupRepo.SearchByTags(incIDs, logic)
		}
		if err != nil {
			return nil, fmt.Errorf("搜索失败: %w", err)
		}

		excFiles, err := s.tagGroupRepo.SearchByTags(excIDs, "or")
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

	// Batch load: collect all file IDs, then load default groups + tags in 2 queries
	fileIDs := make([]uint, len(files))
	for i, f := range files {
		fileIDs[i] = f.FileID
	}

	groupMap := make(map[uint]*tagPair)
	if len(fileIDs) > 0 {
		groups, err := s.tagGroupRepo.FindDefaultGroupsBatch(fileIDs)
		if err != nil {
			log.Printf("[search] batch default groups: %v", err)
		}
		groupIDs := make([]uint, 0, len(groups))
		for i := range groups {
			groupIDs = append(groupIDs, groups[i].GroupID)
			groupMap[groups[i].FileID] = &tagPair{groupID: groups[i].GroupID}
		}

		if len(groupIDs) > 0 {
			tagsByGroup, err := s.tagGroupRepo.TagsByGroupIDs(groupIDs)
			if err != nil {
				log.Printf("[search] batch tags: %v", err)
			}
			for gid, tags := range tagsByGroup {
				for _, p := range groupMap {
					if p.groupID == gid {
						p.tags = tags
					}
				}
			}
		}
	}

	result := make([]FileWithTags, 0, len(files))
	for _, f := range files {
		p := groupMap[f.FileID]
		if p == nil {
			result = append(result, FileWithTags{File: f})
		} else {
			result = append(result, FileWithTags{File: f, Tags: p.tags})
		}
	}
	return result, nil
}

type tagPair struct {
	groupID uint
	tags    []model.Tag
}
