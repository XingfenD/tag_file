package bootstrap

import (
	"encoding/json"

	"github.com/XingfenD/tag_file/internal/service/config/schema"
)

// 扫描相关配置键的块注册。Default 由 defaultExcludePatterns 序列化而来，
// 种子播种与设置页读取都取这里的声明，避免两处各写一份排除规则。
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
				Description: "输入正则表达式",
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
