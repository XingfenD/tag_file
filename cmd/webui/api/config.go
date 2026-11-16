package api

import (
	"errors"
	"fmt"
	"sort"

	"github.com/XingfenD/tag_file/internal/model"
	configSvc "github.com/XingfenD/tag_file/internal/service/config"
	"github.com/XingfenD/tag_file/internal/service/config/schema"
	"github.com/gin-gonic/gin"
)

// ConfigItem is the JSON representation of a config entry.
type ConfigItem struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Description string `json:"description"`
	// HasValue 只对 secret 键有意义：库中是否存在非空值。非 secret 键不设，随 omitempty 隐藏。
	HasValue bool `json:"has_value,omitempty"`
}

// HandleListConfig responds to GET /api/config.
// Supports ?q=X&page=N for search and pagination, and ?unknown=1 to list
// only unregistered keys.
func (rt *Router) HandleListConfig(c *gin.Context) {
	q := c.Query("q")
	unknown := c.Query("unknown") == "1"
	pg := parsePagination(c, 0, 10)

	var configs []model.Config
	var total int64
	var err error

	offset := (pg.Page - 1) * pg.PageSize
	switch {
	case unknown && q != "":
		configs, total, err = rt.ConfigSvc.SearchUnknown(q, offset, pg.PageSize)
	case unknown:
		configs, total, err = rt.ConfigSvc.ListUnknown(offset, pg.PageSize)
	case q != "":
		configs, total, err = rt.ConfigSvc.SearchConfigs(q, offset, pg.PageSize)
	default:
		configs, total, err = rt.ConfigSvc.List(offset, pg.PageSize)
	}
	if err != nil {
		respond(c, err.Error(), "error")
		return
	}

	pg = computePagination(total, pg.Page, pg.PageSize)

	items := make([]ConfigItem, len(configs))
	for i, cfg := range configs {
		items[i] = ConfigItem{
			Key:         cfg.ConfigName,
			Value:       cfg.ConfigValue,
			Description: cfg.Description,
		}
		// secret 键绝不回传库中的值，只回传「是否存在非空值」。
		if field, ok := schema.Lookup(cfg.ConfigName); ok && field.Type == schema.TypeSecret {
			items[i].Value = ""
			items[i].HasValue = cfg.ConfigValue != ""
		}
	}

	respond(c, "ok", "success", map[string]any{
		"items":      items,
		"pagination": pg,
	})
}

// HandleSetConfig responds to POST /api/config.
// Reads key and value from form data.
func (rt *Router) HandleSetConfig(c *gin.Context) {
	key := c.PostForm("key")
	value := c.PostForm("value")
	if err := rt.ConfigSvc.Set(key, value); err != nil {
		respond(c, err.Error(), "error")
		return
	}
	respond(c, "配置 «"+key+"» 已保存", "success")
}

// HandleUpdateConfig responds to PUT /api/config/:key.
// Reads value from form data.
func (rt *Router) HandleUpdateConfig(c *gin.Context) {
	key := c.Param("key")
	value := c.PostForm("value")
	if err := rt.ConfigSvc.Update(key, value); err != nil {
		respond(c, err.Error(), "error")
		return
	}
	respond(c, "配置 «"+key+"» 已更新", "success")
}

// HandleUnsetConfig responds to DELETE /api/config/:key.
func (rt *Router) HandleUnsetConfig(c *gin.Context) {
	key := c.Param("key")
	if err := rt.ConfigSvc.Delete(key); err != nil {
		respond(c, err.Error(), "error")
		return
	}
	respond(c, "配置 «"+key+"» 已删除", "success")
}

// HandleSetConfigGroup responds to POST /api/config/group.
// 整组保存：只收集该分组的字段（用 GetPostForm 区分缺失与空串），全有全合法才写库。
func (rt *Router) HandleSetConfigGroup(c *gin.Context) {
	group := c.PostForm("group")
	if group == "" {
		respond(c, "缺少分组参数", "error")
		return
	}
	fields := schema.FieldsOf(group)
	if len(fields) == 0 {
		respond(c, fmt.Sprintf("未知配置分组 '%s'", group), "error")
		return
	}

	values := make(map[string]string, len(fields))
	for _, f := range fields {
		if v, ok := c.GetPostForm(f.Key); ok {
			values[f.Key] = v
		}
	}

	if err := rt.ConfigSvc.SetGroup(group, values); err != nil {
		var verrs configSvc.ValidationErrors
		if errors.As(err, &verrs) {
			keys := make([]string, 0, len(verrs))
			for k := range verrs {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			failures := make([]fieldFailure, 0, len(keys))
			for _, k := range keys {
				failures = append(failures, fieldFailure{Key: k, Msg: verrs[k]})
			}
			respond(c, "配置校验失败", "error", map[string]any{"failures": failures})
			return
		}
		respond(c, err.Error(), "error")
		return
	}

	title := group
	for _, g := range schema.Groups() {
		if g.ID == group {
			title = g.Title
			break
		}
	}
	respond(c, "分组 «"+title+"» 已保存", "success")
}

// fieldFailure 是整组保存失败时单个键的校验原因，按键排序后回传。
type fieldFailure struct {
	Key string `json:"key"`
	Msg string `json:"msg"`
}

// HandleConfigSchema responds to GET /api/config/schema.
// 一次性读取整张配置表（limit 0 = 全部行），按分组拼出当前值、边界与校验状态。
func (rt *Router) HandleConfigSchema(c *gin.Context) {
	cfgs, _, err := rt.ConfigSvc.List(0, 0)
	if err != nil {
		respond(c, err.Error(), "error")
		return
	}
	stored := make(map[string]string, len(cfgs))
	for _, cfg := range cfgs {
		stored[cfg.ConfigName] = cfg.ConfigValue
	}

	groups := schema.Groups()
	out := make([]schemaGroupResponse, 0, len(groups))
	for _, g := range groups {
		fields := schema.FieldsOf(g.ID)
		fieldOut := make([]schemaFieldResponse, 0, len(fields))
		for _, f := range fields {
			fieldOut = append(fieldOut, buildSchemaField(f, stored))
		}
		out = append(out, schemaGroupResponse{
			ID:     g.ID,
			Title:  g.Title,
			Order:  g.Order,
			Fields: fieldOut,
		})
	}

	respond(c, "ok", "success", map[string]any{"groups": out})
}

// buildSchemaField 把 schema.Field 投影成前端契约：value/has_value 来自存量库，
// invalid 由 Validate 判定（secret 键恒不判），边界与选项原样透传。
func buildSchemaField(f schema.Field, stored map[string]string) schemaFieldResponse {
	value := f.Default
	hasValue := false
	if v, ok := stored[f.Key]; ok {
		value = v
		hasValue = v != ""
	}

	invalid := false
	invalidReason := ""
	if f.Type != schema.TypeSecret {
		if err := schema.Validate(f, value); err != nil {
			invalid = true
			invalidReason = err.Error()
		}
	}

	if f.Type == schema.TypeSecret {
		value = ""
	}

	options := make([]schemaOptionResponse, 0, len(f.Options))
	for _, o := range f.Options {
		options = append(options, schemaOptionResponse{Value: o.Value, Label: o.Label})
	}

	var showIf *schemaConditionResponse
	if f.ShowIf != nil {
		showIf = &schemaConditionResponse{Key: f.ShowIf.Key, Equals: f.ShowIf.Equals}
	}

	return schemaFieldResponse{
		Key:           f.Key,
		Label:         f.Label,
		Description:   f.Description,
		Type:          string(f.Type),
		Default:       f.Default,
		Min:           f.Min,
		Max:           f.Max,
		MinExclusive:  f.MinExclusive,
		MaxExclusive:  f.MaxExclusive,
		Pattern:       f.Pattern,
		Options:       options,
		ItemType:      string(f.ItemType),
		Unit:          f.Unit,
		ShowIf:        showIf,
		Value:         value,
		HasValue:      hasValue,
		Invalid:       invalid,
		InvalidReason: invalidReason,
	}
}

// schemaGroupResponse / schemaFieldResponse 是 schema 接口的局部投影结构，
// 不直接序列化 schema.Field，避免把内部字段命名风格泄漏给前端。
type schemaGroupResponse struct {
	ID     string                `json:"id"`
	Title  string                `json:"title"`
	Order  int                   `json:"order"`
	Fields []schemaFieldResponse `json:"fields"`
}

type schemaFieldResponse struct {
	Key           string                   `json:"key"`
	Label         string                   `json:"label"`
	Description   string                   `json:"description"`
	Type          string                   `json:"type"`
	Default       string                   `json:"default"`
	Min           *float64                 `json:"min"`
	Max           *float64                 `json:"max"`
	MinExclusive  bool                     `json:"min_exclusive"`
	MaxExclusive  bool                     `json:"max_exclusive"`
	Pattern       string                   `json:"pattern"`
	Options       []schemaOptionResponse   `json:"options"`
	ItemType      string                   `json:"item_type"`
	Unit          string                   `json:"unit"`
	ShowIf        *schemaConditionResponse `json:"show_if"`
	Value         string                   `json:"value"`
	HasValue      bool                     `json:"has_value"`
	Invalid       bool                     `json:"invalid"`
	InvalidReason string                   `json:"invalid_reason"`
}

type schemaOptionResponse struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

type schemaConditionResponse struct {
	Key    string `json:"key"`
	Equals string `json:"equals"`
}
