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
// Supports ?q=X&page=N for search and pagination.
func (rt *Router) HandleListConfig(c *gin.Context) {
	q := c.Query("q")
	pg := parsePagination(c, 0, 10)

	var configs []model.Config
	var total int64
	var err error

	offset := (pg.Page - 1) * pg.PageSize
	if q != "" {
		configs, total, err = rt.ConfigSvc.SearchConfigs(q, offset, pg.PageSize)
	} else {
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
		if d, ok := schema.Lookup(cfg.ConfigName); ok && d.IsSecret() {
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
	block := schema.BlockOf(group)
	if block == nil {
		respond(c, fmt.Sprintf("未知配置分组 '%s'", group), "error")
		return
	}

	values := make(map[string]string, len(block.InputBoxDescriptorList))
	for _, d := range block.InputBoxDescriptorList {
		if v, ok := c.GetPostForm(d.Key); ok {
			values[d.Key] = v
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

	respond(c, "分组 «"+block.Title+"» 已保存", "success")
}

// fieldFailure 是整组保存失败时单个键的校验原因，按键排序后回传。
type fieldFailure struct {
	Key string `json:"key"`
	Msg string `json:"msg"`
}

// HandleConfigSchema responds to GET /api/config/schema.
// 一次性读取整张配置表（limit 0 = 全部行），按块拼出当前值、规则指令与校验状态。
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

	blocks := schema.Blocks()
	out := make([]schemaBlockResponse, 0, len(blocks))
	for _, b := range blocks {
		boxes := make([]schemaInputBoxResponse, 0, len(b.InputBoxDescriptorList))
		for _, d := range b.InputBoxDescriptorList {
			boxes = append(boxes, buildSchemaInputBox(d, stored))
		}
		out = append(out, schemaBlockResponse{
			ID:         b.ID,
			Title:      b.Title,
			Order:      b.Order,
			InputBoxes: boxes,
		})
	}

	respond(c, "ok", "success", map[string]any{"blocks": out})
}

// buildSchemaInputBox 把 InputBoxDescriptor 投影成前端契约：value/has_value 来自
// 存量库，invalid 由 Validate 判定（secret 键恒不判），约束与行为经
// RulesOf 分解为指令（hint/message/widget 已在分解期渲染，params 原样透传）。
func buildSchemaInputBox(d *schema.InputBoxDescriptor, stored map[string]string) schemaInputBoxResponse {
	value := d.Default
	hasValue := false
	if v, ok := stored[d.Key]; ok {
		value = v
		hasValue = v != ""
	}

	invalid := false
	invalidReason := ""
	if !d.IsSecret() {
		if err := schema.Validate(*d, value); err != nil {
			invalid = true
			invalidReason = err.Error()
		}
	}

	// secret 键的默认值同样必须掩码：Default 本身就可能带明文密钥。
	defaultValue := d.Default
	if d.IsSecret() {
		value = ""
		defaultValue = ""
	}

	var showIf *schemaConditionResponse
	if d.ShowIf != nil {
		showIf = &schemaConditionResponse{Key: d.ShowIf.Key, Equals: d.ShowIf.Equals}
	}

	rs := schema.RulesOf(*d)

	return schemaInputBoxResponse{
		Key:             d.Key,
		Label:           d.Label,
		Description:     d.Description,
		ValueType:       string(d.ValueType),
		Container:       string(d.Container),
		Default:         defaultValue,
		Unit:            d.Unit,
		ShowIf:          showIf,
		Rules:           projectRules(rs.Rules),
		PositionalRules: projectRuleGroups(rs.PositionalRules),
		ContainerRules:  projectRules(rs.ContainerRules),
		Value:           value,
		HasValue:        hasValue,
		Invalid:         invalid,
		InvalidReason:   invalidReason,
	}
}

// projectRules 把一组指令投影为前端契约；空组回传空切片而非 nil，避免 JSON null。
func projectRules(rules []schema.Rule) []schemaRuleResponse {
	out := make([]schemaRuleResponse, 0, len(rules))
	for _, r := range rules {
		out = append(out, schemaRuleResponse{
			Kind:    string(r.Kind),
			Role:    string(r.Role),
			Params:  nonNilParams(r.Params),
			Hint:    r.Hint,
			Message: r.Message,
			Widget:  r.Widget,
		})
	}
	return out
}

// nonNilParams 把 nil 参数归一化为空 map：无参数指令的 params 序列化为 {}
// 而非 null（spec §9 钉 params 恒为对象）。
func nonNilParams(params map[string]string) map[string]string {
	if params == nil {
		return map[string]string{}
	}
	return params
}

// projectRuleGroups 投影逐位指令组（仅 Array 有）；空组回传空切片而非 nil。
func projectRuleGroups(groups [][]schema.Rule) [][]schemaRuleResponse {
	out := make([][]schemaRuleResponse, 0, len(groups))
	for _, g := range groups {
		out = append(out, projectRules(g))
	}
	return out
}

// schemaBlockResponse / schemaInputBoxResponse / schemaRuleResponse 是 schema 接口
// 的局部投影结构，不直接序列化描述符，避免把内部字段命名风格泄漏给前端。
type schemaBlockResponse struct {
	ID         string                   `json:"id"`
	Title      string                   `json:"title"`
	Order      int                      `json:"order"`
	InputBoxes []schemaInputBoxResponse `json:"input_boxes"`
}

type schemaInputBoxResponse struct {
	Key             string                   `json:"key"`
	Label           string                   `json:"label"`
	Description     string                   `json:"description"`
	ValueType       string                   `json:"value_type"`
	Container       string                   `json:"container"`
	Default         string                   `json:"default"`
	Unit            string                   `json:"unit"`
	ShowIf          *schemaConditionResponse `json:"show_if"`
	Rules           []schemaRuleResponse     `json:"rules"`
	PositionalRules [][]schemaRuleResponse   `json:"positional_rules"`
	ContainerRules  []schemaRuleResponse     `json:"container_rules"`
	Value           string                   `json:"value"`
	HasValue        bool                     `json:"has_value"`
	Invalid         bool                     `json:"invalid"`
	InvalidReason   string                   `json:"invalid_reason"`
}

type schemaRuleResponse struct {
	Kind    string            `json:"kind"`
	Role    string            `json:"role"`
	Params  map[string]string `json:"params"`
	Hint    string            `json:"hint"`
	Message string            `json:"message"`
	Widget  string            `json:"widget"`
}

type schemaConditionResponse struct {
	Key    string `json:"key"`
	Equals string `json:"equals"`
}
