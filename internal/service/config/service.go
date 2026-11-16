package config

import (
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/XingfenD/tag_file/internal/model"
	repoConfig "github.com/XingfenD/tag_file/internal/repository/config"
	"github.com/XingfenD/tag_file/internal/service/config/schema"
)

type Service struct {
	configRepo *repoConfig.Repository
}

// ValidationErrors 收集整组保存时每个键的校验失败原因，键为配置键，值为原因。
type ValidationErrors map[string]string

func (v ValidationErrors) Error() string {
	keys := make([]string, 0, len(v))
	for k := range v {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s: %s", k, v[k]))
	}
	return strings.Join(parts, "; ")
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

// SetGroup 整组保存：分组内每个字段都必须出现且合法，values 中也不允许混入分组外的键。
// 任一问题都返回 ValidationErrors，且一个键都不写入。
func (s *Service) SetGroup(group string, values map[string]string) error {
	fields := schema.FieldsOf(group)
	if len(fields) == 0 {
		return fmt.Errorf("未知配置分组 '%s'", group)
	}

	verrs := ValidationErrors{}
	known := make(map[string]struct{}, len(fields))
	for _, f := range fields {
		known[f.Key] = struct{}{}
		value, ok := values[f.Key]
		if !ok {
			// secret 字段按协议（spec 6.1）从不回传值、只回 has_value，客户端从未
			// 收到过已存密钥、也就无法把它原样回传，因此缺失视为「保持不变」而非「缺少字段」。
			if f.Type != schema.TypeSecret {
				verrs[f.Key] = "缺少字段"
			}
			continue
		}
		if err := schema.Validate(f, value); err != nil {
			verrs[f.Key] = err.Error()
		}
	}
	for key := range values {
		if _, ok := known[key]; !ok {
			verrs[key] = "不属于该分组"
		}
	}
	if len(verrs) > 0 {
		return verrs
	}
	return s.configRepo.SetMany(values)
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
