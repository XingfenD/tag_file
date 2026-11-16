package bootstrap

import (
	"encoding/json"

	"github.com/XingfenD/tag_file/internal/service/config/schema"
)

// 扫描相关配置键的注册。Default 由 defaultExcludePatterns 序列化而来，
// 与 seedDefaultConfig 的种子值同源，避免两处各写一份排除规则。
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
