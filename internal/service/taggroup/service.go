package taggroup

import (
	"fmt"
	"log"

	"github.com/XingfenD/tag_file/internal/model"
	repo "github.com/XingfenD/tag_file/internal/repository/taggroup"
	repoTag "github.com/XingfenD/tag_file/internal/repository/tag"
)

type Service struct {
	repo    *repo.Repository
	tagRepo *repoTag.Repository
}

func NewService() *Service {
	return &Service{
		repo:    repo.NewRepository(),
		tagRepo: repoTag.NewRepository(),
	}
}

func (s *Service) CreateDefaultGroup(fileID uint) (*model.TagGroup, error) {
	group := &model.TagGroup{
		FileID:    fileID,
		GroupName: "default",
		IsDefault: true,
	}
	if err := s.repo.CreateGroup(group); err != nil {
		return nil, fmt.Errorf("创建默认标签组失败: %w", err)
	}
	return group, nil
}

func (s *Service) AddGroup(fileID uint, groupName string) (*model.TagGroup, error) {
	group := &model.TagGroup{
		FileID:    fileID,
		GroupName: groupName,
		IsDefault: false,
	}
	if err := s.repo.CreateGroup(group); err != nil {
		return nil, fmt.Errorf("创建标签组失败: %w", err)
	}
	return group, nil
}

func (s *Service) RemoveGroup(fileID uint, groupName string) error {
	group, err := s.repo.FindGroupByName(fileID, groupName)
	if err != nil {
		return fmt.Errorf("标签组 '%s' 不存在", groupName)
	}
	if group.IsDefault {
		return fmt.Errorf("不能删除默认标签组")
	}
	return s.repo.DeleteGroup(group.GroupID)
}

func (s *Service) ListGroups(fileID uint) ([]model.TagGroup, error) {
	return s.repo.FindGroups(fileID)
}

func (s *Service) TagGroup(fileID uint, groupName string, tagName string, tagValue string) error {
	group, err := s.repo.FindGroupByName(fileID, groupName)
	if err != nil {
		return fmt.Errorf("标签组 '%s' 不存在", groupName)
	}

	tags, err := s.tagRepo.FindByName(tagName)
	if err != nil {
		return fmt.Errorf("查询标签失败: %w", err)
	}
	if len(tags) == 0 {
		return fmt.Errorf("标签 '%s' 不存在", tagName)
	}

	for _, tag := range tags {
		has, err := s.repo.HasTagValue(group.GroupID, tag.TagID, tagValue)
		if err != nil {
			return fmt.Errorf("查询关联失败: %w", err)
		}
		if has {
			return fmt.Errorf("标签组已存在标签 '%s'", tagName)
		}
		return s.repo.AddTag(group.GroupID, fileID, tag.TagID, tagValue)
	}
	return fmt.Errorf("标签 '%s' 不存在", tagName)
}

func (s *Service) UntagGroup(fileID uint, groupName string, tagName string, tagValue string) error {
	group, err := s.repo.FindGroupByName(fileID, groupName)
	if err != nil {
		return fmt.Errorf("标签组 '%s' 不存在", groupName)
	}

	tags, err := s.tagRepo.FindByName(tagName)
	if err != nil {
		return fmt.Errorf("查询标签失败: %w", err)
	}
	if len(tags) == 0 {
		return fmt.Errorf("标签 '%s' 不存在", tagName)
	}

	for _, tag := range tags {
		has, err := s.repo.HasTagValue(group.GroupID, tag.TagID, tagValue)
		if err != nil {
			return fmt.Errorf("查询关联失败: %w", err)
		}
		if !has {
			return fmt.Errorf("标签组不存在标签 '%s'", tagName)
		}
		return s.repo.RemoveTag(group.GroupID, tag.TagID, tagValue)
	}
	return fmt.Errorf("标签 '%s' 不存在", tagName)
}

func (s *Service) GetTags(fileID uint, groupName string) ([]model.Tag, error) {
	group, err := s.repo.FindGroupByName(fileID, groupName)
	if err != nil {
		return nil, fmt.Errorf("标签组 '%s' 不存在", groupName)
	}
	return s.repo.GetTags(group.GroupID)
}

func (s *Service) GetFileTags(fileID uint, offset, limit int) ([]model.TagGroupTag, int64, error) {
	return s.repo.GetFileTags(fileID, offset, limit)
}

func (s *Service) GetAllFileTagValues(fileID uint) (map[uint][]string, error) {
	return s.repo.GetAllFileTagValues(fileID)
}

func (s *Service) GetDefaultGroup(fileID uint) (*model.TagGroup, error) {
	group, err := s.repo.FindDefaultGroup(fileID)
	if err != nil {
		log.Printf("[taggroup] GetDefaultGroup: %v", err)
		s.CreateDefaultGroup(fileID)
		group, err = s.repo.FindDefaultGroup(fileID)
		if err != nil {
			return nil, fmt.Errorf("创建默认标签组失败: %w", err)
		}
	}
	return group, nil
}
