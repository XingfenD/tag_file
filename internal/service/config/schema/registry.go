package schema

import (
	"fmt"
	"sort"
	"sync"
)

// 注册表是进程级的：配置键的声明发生在 init 期（各消费方的 schema 文件），
// 读多写少，用一把互斥锁保证测试并发调用时也安全。
var (
	registryMu     sync.Mutex
	registryGroups = map[string]Group{}
	registryFields = map[string]Field{}
)

// RegisterGroup 注册一个设置分组。注册发生在 init 期，重复 ID 只可能是代码写错，
// 因此直接 panic，让进程在启动时暴露问题，而不是运行时静默覆盖。
func RegisterGroup(g Group) {
	registryMu.Lock()
	defer registryMu.Unlock()

	if _, ok := registryGroups[g.ID]; ok {
		panic(fmt.Sprintf("schema: 分组 %q 重复注册", g.ID))
	}
	registryGroups[g.ID] = g
}

// Register 注册一个配置字段，并用自己的默认值跑一次 Validate 自检。
// 重复键、所在分组未注册、默认值不合法都会 panic：这三类错误都是声明写错，
// 必须在启动时失败，绝不能留到用户改配置时才以「取值非法」的形式出现。
func Register(f Field) {
	registryMu.Lock()
	defer registryMu.Unlock()

	if _, ok := registryGroups[f.Group]; !ok {
		panic(fmt.Sprintf("schema: 配置键 %q 所属分组 %q 未注册", f.Key, f.Group))
	}
	if _, ok := registryFields[f.Key]; ok {
		panic(fmt.Sprintf("schema: 配置键 %q 重复注册", f.Key))
	}
	if err := Validate(f, f.Default); err != nil {
		panic(fmt.Sprintf("schema: 配置键 %q 的默认值 %q 非法: %v", f.Key, f.Default, err))
	}
	registryFields[f.Key] = f
}

// Lookup 按配置键查找字段。
func Lookup(key string) (Field, bool) {
	registryMu.Lock()
	defer registryMu.Unlock()

	f, ok := registryFields[key]
	return f, ok
}

// Groups 返回所有分组，按 Order 升序；Order 相同则按 ID 升序，保证顺序确定。
func Groups() []Group {
	registryMu.Lock()
	defer registryMu.Unlock()

	groups := make([]Group, 0, len(registryGroups))
	for _, g := range registryGroups {
		groups = append(groups, g)
	}
	sort.Slice(groups, func(i, j int) bool {
		if groups[i].Order != groups[j].Order {
			return groups[i].Order < groups[j].Order
		}
		return groups[i].ID < groups[j].ID
	})
	return groups
}

// FieldsOf 返回某个分组下的字段，按 Key 升序；未知分组返回 nil。
func FieldsOf(groupID string) []Field {
	registryMu.Lock()
	defer registryMu.Unlock()

	if _, ok := registryGroups[groupID]; !ok {
		return nil
	}
	fields := make([]Field, 0, len(registryFields))
	for _, f := range registryFields {
		if f.Group == groupID {
			fields = append(fields, f)
		}
	}
	sortFieldsByKey(fields)
	return fields
}

// All 返回所有已注册字段，按 Key 升序。
func All() []Field {
	registryMu.Lock()
	defer registryMu.Unlock()

	fields := make([]Field, 0, len(registryFields))
	for _, f := range registryFields {
		fields = append(fields, f)
	}
	sortFieldsByKey(fields)
	return fields
}

// sortFieldsByKey 就地对字段切片按 Key 升序排序。
func sortFieldsByKey(fields []Field) {
	sort.Slice(fields, func(i, j int) bool {
		return fields[i].Key < fields[j].Key
	})
}
