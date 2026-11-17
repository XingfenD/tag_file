package bootstrap

import (
	"encoding/json"

	// 空导入只为触发 AI 配置键的注册：seedDefaultConfig 遍历注册表播种，
	// 若这里丢掉这条 import 链，AI 的 5 个键会从新库里整体消失（CLI 自身不引用 ai）。
	_ "github.com/XingfenD/tag_file/internal/service/ai"
	"github.com/XingfenD/tag_file/internal/service/config/schema"
)

// 扫描相关配置键的注册。Default 由 defaultExcludePatterns 序列化而来，
// 种子播种与设置页读取都取这里的声明，避免两处各写一份排除规则。
func init() {
	schema.RegisterGroup(schema.Group{ID: "scan", Title: "扫描设置", Order: 1})

	excludeDefault, _ := json.Marshal(defaultExcludePatterns)

	schema.Register(schema.Field{
		Key:         "scan_exclude",
		Group:       "scan",
		Label:       "扫描排除规则",
		Description: "扫描排除正则列表",
		Type:        schema.TypeList,
		ItemType:    schema.TypeRegex,
		Default:     string(excludeDefault),
	})
}

// 新模型声明（spec §10 映射）：与上面的旧 init 并存过渡。config service 的
// SetGroup 已切到块注册表，旧注册表删除（Task 5）前这里必须先有等价块，
// 否则 scan 分组的整组保存会报「未知配置分组」。
func init() {
	excludeDefault, _ := json.Marshal(defaultExcludePatterns)

	schema.RegisterBlock(schema.ConfigBlockDescriptor{
		ID:    "scan",
		Title: "扫描设置",
		Order: 1,
		InputBoxDescriptorList: []*schema.InputBoxDescriptor{
			{
				Key:         "scan_exclude",
				Label:       "扫描排除规则",
				Description: "扫描排除正则列表",
				ValueType:   schema.ValueString,
				Container:   schema.ContainerDynamicArray,
				Default:     string(excludeDefault),
				Constraint: schema.InputBoxConstraintDescriptor{
					StringConstraint: &schema.StringConstraintDescriptor{Format: schema.FormatRegex},
				},
			},
		},
	})
}
