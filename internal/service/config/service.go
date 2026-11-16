package config

import (
	"fmt"
	"log"

	"github.com/XingfenD/tag_file/internal/model"
	repoConfig "github.com/XingfenD/tag_file/internal/repository/config"
	"github.com/XingfenD/tag_file/internal/service/config/schema"
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
	if err := validateKnown(key, value); err != nil {
		return err
	}
	return s.configRepo.Set(key, value)
}

func (s *Service) Get(key string) (*model.Config, error) {
	cfg, err := s.configRepo.Get(key)
	if err != nil {
		log.Printf("[config] Get: %v", err)
		return nil, fmt.Errorf("查询配置失败: %w", err)
	}
	if cfg == nil {
		return nil, fmt.Errorf("配置 '%s' 不存在", key)
	}
	return cfg, nil
}

func (s *Service) List(offset, limit int) ([]model.Config, int64, error) {
	list, total, err := s.configRepo.List().Offset(offset).Limit(limit).Exec()
	if err != nil {
		log.Printf("[config] List: %v", err)
		return nil, 0, fmt.Errorf("查询配置列表失败: %w", err)
	}
	return list, total, nil
}

func (s *Service) Delete(key string) error {
	if _, ok := schema.Lookup(key); ok {
		return fmt.Errorf("配置 '%s' 是系统配置，不能删除；如需还原请使用恢复默认", key)
	}
	cfg, err := s.configRepo.Get(key)
	if err != nil {
		log.Printf("[config] Delete: get: %v", err)
		return fmt.Errorf("查询配置失败: %w", err)
	}
	if cfg == nil {
		return fmt.Errorf("配置 '%s' 不存在", key)
	}
	return s.configRepo.Delete(key)
}

func (s *Service) Update(key string, value string) error {
	cfg, err := s.configRepo.Get(key)
	if err != nil {
		log.Printf("[config] Update: get: %v", err)
		return fmt.Errorf("查询配置失败: %w", err)
	}
	if cfg == nil {
		return fmt.Errorf("配置 '%s' 不存在", key)
	}
	if err := validateKnown(key, value); err != nil {
		return err
	}
	cfg.ConfigValue = value
	return s.configRepo.Update(cfg).SetValue(value).Exec()
}

// validateKnown 对注册表中的键按字段类型校验取值；未注册的键不做限制。
func validateKnown(key string, value string) error {
	field, ok := schema.Lookup(key)
	if !ok {
		return nil
	}
	if err := schema.Validate(field, value); err != nil {
		return fmt.Errorf("配置 %s 不合法: %w", key, err)
	}
	return nil
}

func (s *Service) SearchConfigs(keyword string, offset, limit int) ([]model.Config, int64, error) {
	if keyword == "" {
		return nil, 0, fmt.Errorf("搜索关键词不能为空")
	}
	return s.configRepo.Search().ByName(keyword).Offset(offset).Limit(limit).Exec()
}
