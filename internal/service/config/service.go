package config

import (
	"fmt"

	"github.com/XingfenD/tag_file/internal/model"
	repoConfig "github.com/XingfenD/tag_file/internal/repository/config"
)

type Service struct {
	configRepo *repoConfig.Repository
}

func NewService() *Service {
	return &Service{
		configRepo: repoConfig.NewRepository(),
	}
}

func (s *Service) Set(key string, value string) error {
	if key == "" {
		return fmt.Errorf("配置键不能为空")
	}
	return s.configRepo.Set(key, value)
}

func (s *Service) Get(key string) (*model.Config, error) {
	cfg, err := s.configRepo.Get(key)
	if err != nil {
		return nil, fmt.Errorf("查询配置失败: %w", err)
	}
	if cfg == nil {
		return nil, fmt.Errorf("配置 '%s' 不存在", key)
	}
	return cfg, nil
}

func (s *Service) List(offset, limit int) ([]model.Config, int64, error) {
	list, total, err := s.configRepo.List(offset, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("查询配置列表失败: %w", err)
	}
	return list, total, nil
}

func (s *Service) Delete(key string) error {
	cfg, err := s.configRepo.Get(key)
	if err != nil {
		return fmt.Errorf("查询配置失败: %w", err)
	}
	if cfg == nil {
		return fmt.Errorf("配置 '%s' 不存在", key)
	}
	return s.configRepo.Delete(key)
}
