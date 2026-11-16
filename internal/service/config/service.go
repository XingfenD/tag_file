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

// ListUnknown 列出未注册的配置键（即高级区展示的数据源），分页规则同 List。
func (s *Service) ListUnknown(offset, limit int) ([]model.Config, int64, error) {
	list, _, err := s.configRepo.List().Exec()
	if err != nil {
		log.Printf("[config] ListUnknown: %v", err)
		return nil, 0, fmt.Errorf("查询配置列表失败: %w", err)
	}
	cfgs, total := pageUnknown(list, offset, limit)
	return cfgs, total, nil
}

// SearchUnknown 在未注册的配置键中按名称模糊搜索；关键词为空时等同于 ListUnknown。
func (s *Service) SearchUnknown(keyword string, offset, limit int) ([]model.Config, int64, error) {
	if keyword == "" {
		return s.ListUnknown(offset, limit)
	}
	list, _, err := s.configRepo.Search().ByName(keyword).Exec()
	if err != nil {
		log.Printf("[config] SearchUnknown: %v", err)
		return nil, 0, fmt.Errorf("搜索配置失败: %w", err)
	}
	cfgs, total := pageUnknown(list, offset, limit)
	return cfgs, total, nil
}

// pageUnknown 过滤掉注册表中的键，返回过滤后的总数与 [offset, offset+limit) 切片；limit<=0 表示不限。
func pageUnknown(cfgs []model.Config, offset, limit int) ([]model.Config, int64) {
	unknown := make([]model.Config, 0, len(cfgs))
	for _, cfg := range cfgs {
		if _, ok := schema.Lookup(cfg.ConfigName); ok {
			continue
		}
		unknown = append(unknown, cfg)
	}
	total := int64(len(unknown))
	if offset < 0 {
		offset = 0
	}
	if offset >= len(unknown) {
		return []model.Config{}, total
	}
	end := len(unknown)
	if limit > 0 && offset+limit < end {
		end = offset + limit
	}
	return unknown[offset:end], total
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
