package taggroup

import (
	"fmt"
	"log"

	"github.com/XingfenD/tag_file/internal/model"
	repoFile "github.com/XingfenD/tag_file/internal/repository/file"
	repo "github.com/XingfenD/tag_file/internal/repository/taggroup"
	repoTag "github.com/XingfenD/tag_file/internal/repository/tag"
)

type Service struct {
	repo     *repo.Repository
	tagRepo  *repoTag.Repository
	fileRepo *repoFile.Repository
}

func NewService() *Service {
	return &Service{
		repo:     repo.NewRepository(),
		tagRepo:  repoTag.NewRepository(),
		fileRepo: repoFile.NewRepository(),
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
		return fmt.Errorf("查询标签组失败: %w", err)
	}
	if group == nil {
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

func (s *Service) TagGroup(fileID uint, groupName string, tagID uint, tagValue string) error {
	group, err := s.repo.FindGroupByName(fileID, groupName)
	if err != nil {
		return fmt.Errorf("查询标签组失败: %w", err)
	}
	if group == nil {
		return fmt.Errorf("标签组 '%s' 不存在", groupName)
	}

	tag, err := s.tagRepo.Find().ByID(tagID).Exec()
	if err != nil {
		return fmt.Errorf("查询标签失败: %w", err)
	}
	if tag == nil {
		return fmt.Errorf("标签 %d 不存在", tagID)
	}

	has, err := s.repo.HasTagValue(group.GroupID, tag.TagID, tagValue)
	if err != nil {
		return fmt.Errorf("查询关联失败: %w", err)
	}
	if has {
		return fmt.Errorf("标签组已存在标签 '%s'", tag.TagName)
	}
	if err := s.repo.AddTag(group.GroupID, fileID, tag.TagID, tagValue); err != nil {
		return err
	}
	if err := s.fileRepo.IncrementTagCnt(fileID, 1); err != nil {
		log.Printf("[taggroup] TagGroup: incr tag cnt: %v", err)
	}
	return nil
}

func (s *Service) UntagGroup(fileID uint, groupName string, tagID uint, tagValue string) error {
	group, err := s.repo.FindGroupByName(fileID, groupName)
	if err != nil {
		return fmt.Errorf("查询标签组失败: %w", err)
	}
	if group == nil {
		return fmt.Errorf("标签组 '%s' 不存在", groupName)
	}

	tag, err := s.tagRepo.Find().ByID(tagID).Exec()
	if err != nil {
		return fmt.Errorf("查询标签失败: %w", err)
	}
	if tag == nil {
		return fmt.Errorf("标签 %d 不存在", tagID)
	}

	has, err := s.repo.HasTagValue(group.GroupID, tag.TagID, tagValue)
	if err != nil {
		return fmt.Errorf("查询关联失败: %w", err)
	}
	if !has {
		return fmt.Errorf("标签组不存在标签 '%s'", tag.TagName)
	}
	if err := s.repo.RemoveTag(group.GroupID, tag.TagID, tagValue); err != nil {
		return err
	}
	if err := s.fileRepo.IncrementTagCnt(fileID, -1); err != nil {
		log.Printf("[taggroup] UntagGroup: decr tag cnt: %v", err)
	}
	return nil
}

func (s *Service) GetTags(fileID uint, groupName string) ([]model.Tag, error) {
	group, err := s.repo.FindGroupByName(fileID, groupName)
	if err != nil {
		return nil, fmt.Errorf("查询标签组失败: %w", err)
	}
	if group == nil {
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
		log.Printf("[taggroup] GetDefaultGroup: find: %v", err)
		if _, createErr := s.CreateDefaultGroup(fileID); createErr != nil {
			return nil, fmt.Errorf("创建默认标签组失败: %w", createErr)
		}
		group, err = s.repo.FindDefaultGroup(fileID)
		if err != nil {
			return nil, fmt.Errorf("获取默认标签组失败: %w", err)
		}
	}
	return group, nil
}
