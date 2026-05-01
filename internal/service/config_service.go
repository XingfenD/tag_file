package service

import (
	"fmt"

	"github.com/XingfenD/tag_file/internal/model"
	"github.com/XingfenD/tag_file/internal/repository"
)

type ConfigService struct {
	configRepo *repository.ConfigRepository
}

func NewConfigService() *ConfigService {
	return &ConfigService{
		configRepo: repository.NewConfigRepository(),
	}
}

func (s *ConfigService) Set(key string, value string) error {
	if key == "" {
		return fmt.Errorf("配置键不能为空")
	}
	return s.configRepo.Set(key, value)
}

func (s *ConfigService) Get(key string) (*model.Config, error) {
	cfg, err := s.configRepo.Get(key)
	if err != nil {
		return nil, fmt.Errorf("查询配置失败: %w", err)
	}
	if cfg == nil {
		return nil, fmt.Errorf("配置 '%s' 不存在", key)
	}
	return cfg, nil
}

func (s *ConfigService) List() ([]model.Config, error) {
	return s.configRepo.List()
}

func (s *ConfigService) Delete(key string) error {
	cfg, err := s.configRepo.Get(key)
	if err != nil {
		return fmt.Errorf("查询配置失败: %w", err)
	}
	if cfg == nil {
		return fmt.Errorf("配置 '%s' 不存在", key)
	}
	return s.configRepo.Delete(key)
}
