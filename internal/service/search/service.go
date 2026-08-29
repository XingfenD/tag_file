package search

import (
	"fmt"
	"log"
	"strings"

	"github.com/XingfenD/tag_file/internal/model"
	repoTag "github.com/XingfenD/tag_file/internal/repository/tag"
	repoTagGroup "github.com/XingfenD/tag_file/internal/repository/taggroup"
	"github.com/XingfenD/tag_file/pkg/storage"
	"gorm.io/gorm"
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
	tags, err := s.requireTags(tagNames)
	if err != nil {
		return nil, err
	}
	allIDs := make([]uint, len(tags))
	for i, t := range tags {
		allIDs[i] = t.TagID
	}
	return allIDs, nil
}

func (s *Service) resolveTagIDsWithDescendants(tagNames []string) ([]uint, error) {
	tags, err := s.requireTags(tagNames)
	if err != nil {
		return nil, err
	}
	tagIDs := make([]uint, len(tags))
	for i, t := range tags {
		tagIDs[i] = t.TagID
	}
	return s.tagRepo.GetDescendantIDsMulti(tagIDs)
}

func (s *Service) resolveTagGroupsWithDescendants(tagNames []string) ([][]uint, error) {
	tags, err := s.requireTags(tagNames)
	if err != nil {
		return nil, err
	}

	nameToIDs := make(map[string][]uint)
	var allIDs []uint
	for _, t := range tags {
		nameToIDs[t.TagName] = append(nameToIDs[t.TagName], t.TagID)
		allIDs = append(allIDs, t.TagID)
	}

	descMap, err := s.tagRepo.GetDescendantIDsMultiMap(allIDs)
	if err != nil {
		return nil, fmt.Errorf("查询子孙标签失败: %w", err)
	}

	groups := make([][]uint, 0, len(tagNames))
	for _, name := range tagNames {
		var group []uint
		for _, id := range nameToIDs[name] {
			group = append(group, descMap[id]...)
		}
		groups = append(groups, group)
	}
	return groups, nil
}

// requireTags finds tags by names and validates all names exist.
func (s *Service) requireTags(tagNames []string) ([]model.Tag, error) {
	tags, err := s.tagRepo.FindByNames(tagNames)
	if err != nil {
		return nil, fmt.Errorf("查询标签失败: %w", err)
	}
	found := make(map[string]bool, len(tags))
	for _, t := range tags {
		found[t.TagName] = true
	}
	for _, name := range tagNames {
		if !found[name] {
			return nil, fmt.Errorf("标签 '%s' 不存在", name)
		}
	}
	return tags, nil
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

	// Batch load tags for all files
	result, err := s.loadTagsForFiles(files)
	if err != nil {
		log.Printf("[search] SearchByTags: load tags: %v", err)
		result = make([]FileWithTags, len(files))
		for i, f := range files {
			result[i] = FileWithTags{File: f}
		}
	}
	return result, nil
}

type tagPair struct {
	groupID uint
	tags    []model.Tag
}

// SearchByQuery executes a structured SearchQuery and returns files with tags.
func (s *Service) SearchByQuery(query SearchQuery) ([]FileWithTags, error) {
	if err := validateQuery(&query); err != nil {
		return nil, err
	}

	db := s.buildSearchQuery(query)
	var files []model.File
	if err := db.Order("file_path").Find(&files).Error; err != nil {
		log.Printf("[search] SearchByQuery: %v", err)
		return nil, fmt.Errorf("搜索失败: %w", err)
	}

	result, err := s.loadTagsForFiles(files)
	if err != nil {
		log.Printf("[search] SearchByQuery: load tags: %v", err)
		result = make([]FileWithTags, len(files))
		for i, f := range files {
			result[i] = FileWithTags{File: f}
		}
	}
	return result, nil
}

// SearchByQueryPaged executes a structured SearchQuery with SQL-level pagination.
// Returns the paginated files with tags and the total count.
func (s *Service) SearchByQueryPaged(query SearchQuery, offset, limit int) ([]FileWithTags, int64, error) {
	if err := validateQuery(&query); err != nil {
		return nil, 0, err
	}

	db := s.buildSearchQuery(query)
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("搜索计数失败: %w", err)
	}
	if total == 0 {
		return nil, 0, nil
	}

	db2 := s.buildSearchQuery(query)
	var files []model.File
	if err := db2.Order("file_path").Offset(offset).Limit(limit).Find(&files).Error; err != nil {
		return nil, 0, fmt.Errorf("搜索失败: %w", err)
	}

	result, err := s.loadTagsForFiles(files)
	if err != nil {
		log.Printf("[search] SearchByQueryPaged: load tags: %v", err)
		result = make([]FileWithTags, len(files))
		for i, f := range files {
			result[i] = FileWithTags{File: f}
		}
	}
	return result, total, nil
}

// loadTagsForFiles batch-loads default groups and tags for a list of files.
func (s *Service) loadTagsForFiles(files []model.File) ([]FileWithTags, error) {
	fileIDs := make([]uint, len(files))
	for i, f := range files {
		fileIDs[i] = f.FileID
	}

	groupMap := make(map[uint]*tagPair)
	if len(fileIDs) > 0 {
		groups, err := s.tagGroupRepo.FindDefaultGroupsBatch(fileIDs)
		if err != nil {
			return nil, fmt.Errorf("查询默认标签组失败: %w", err)
		}
		groupIDs := make([]uint, 0, len(groups))
		for i := range groups {
			groupIDs = append(groupIDs, groups[i].GroupID)
			groupMap[groups[i].FileID] = &tagPair{groupID: groups[i].GroupID}
		}

		if len(groupIDs) > 0 {
			tagsByGroup, err := s.tagGroupRepo.TagsByGroupIDs(groupIDs)
			if err != nil {
				return nil, fmt.Errorf("查询标签失败: %w", err)
			}
			for _, p := range groupMap {
				p.tags = tagsByGroup[p.groupID]
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

// buildSearchQuery recursively builds a GORM query from a SearchQuery.
func (s *Service) buildSearchQuery(q SearchQuery) *gorm.DB {
	db := storage.GetDB().Model(&model.File{})
	return s.applyConditions(db, q.Logic, q.Conditions)
}

func (s *Service) applyConditions(db *gorm.DB, logic string, conditions []SearchCondition) *gorm.DB {
	if len(conditions) == 0 {
		return db
	}

	var clauses []*gorm.DB
	for _, c := range conditions {
		if c.IsGroup() {
			sub := storage.GetDB().Model(&model.File{})
			sub = s.applyConditions(sub, c.Logic, c.Conditions)
			clauses = append(clauses, sub)
		} else {
			clause := s.buildLeafCondition(c)
			if clause != nil {
				clauses = append(clauses, clause)
			}
		}
	}

	if len(clauses) == 0 {
		return db
	}

	if logic == "or" {
		expr := clauses[0]
		for i := 1; i < len(clauses); i++ {
			expr = expr.Or(clauses[i])
		}
		return db.Where(expr)
	}

	for _, clause := range clauses {
		db = db.Where(clause)
	}
	return db
}

func (s *Service) buildLeafCondition(c SearchCondition) *gorm.DB {
	db := storage.GetDB().Model(&model.File{})

	switch c.Type {
	case "filename":
		val, _ := c.Value.(string)
		switch c.Op {
		case "contains":
			return db.Where("file_name LIKE ?", "%"+val+"%")
		case "not_contains":
			return db.Where("file_name NOT LIKE ?", "%"+val+"%")
		case "equals":
			return db.Where("file_name = ?", val)
		}

	case "tag":
		tagName, _ := c.Value.(string)
		tagIDs, err := s.resolveTagIDsWithDescendants([]string{tagName})
		if err != nil || len(tagIDs) == 0 {
			return nil
		}
		sub := storage.GetDB().Table("tag_group_tags").Select("DISTINCT file_id").Where("tag_id IN ?", tagIDs)
		if c.Op == "has" {
			return db.Where("file_id IN (?)", sub)
		}
		return db.Where("file_id NOT IN (?)", sub)

	case "tag_value":
		tagName := c.Tag
		val, _ := c.Value.(string)
		tagIDs, err := s.resolveTagIDsWithDescendants([]string{tagName})
		if err != nil || len(tagIDs) == 0 {
			return nil
		}
		sub := storage.GetDB().Table("tag_group_tags").Select("DISTINCT file_id").
			Where("tag_id IN ? AND tag_value = ?", tagIDs, val)
		if c.Op == "has" {
			return db.Where("file_id IN (?)", sub)
		}
		return db.Where("file_id NOT IN (?)", sub)

	case "size":
		val := toInt64(c.Value)
		switch c.Op {
		case "gt":
			return db.Where("file_size > ?", val)
		case "gte":
			return db.Where("file_size >= ?", val)
		case "lt":
			return db.Where("file_size < ?", val)
		case "lte":
			return db.Where("file_size <= ?", val)
		}

	case "ext":
		exts := toStringSlice(c.Value)
		if len(exts) == 0 {
			return nil
		}
		var likes []string
		var args []any
		for _, ext := range exts {
			ext = strings.ToLower(strings.TrimPrefix(ext, "."))
			likes = append(likes, "LOWER(file_name) LIKE ?")
			args = append(args, "%."+ext)
		}
		expr := strings.Join(likes, " OR ")
		if c.Op == "in" {
			return db.Where("("+expr+")", args...)
		}
		return db.Where("NOT ("+expr+")", args...)

	case "date":
		dateStr, _ := c.Value.(string)
		t, err := parseDate(dateStr)
		if err != nil {
			return nil
		}
		switch c.Op {
		case "gt":
			return db.Where("updated_at > ?", t)
		case "gte":
			return db.Where("updated_at >= ?", t)
		case "lt":
			return db.Where("updated_at < ?", t)
		case "lte":
			return db.Where("updated_at <= ?", t)
		}
	}

	return nil
}
