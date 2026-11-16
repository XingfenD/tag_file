package schema_test

import (
	"sort"
	"strings"
	"testing"

	"github.com/XingfenD/tag_file/internal/service/config/schema"
)

// mustPanic 断言 fn 触发 panic，返回 panic 文案以便检查。
func mustPanic(t *testing.T, fn func()) (msg string) {
	t.Helper()
	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatalf("期望 panic，但没有触发")
		}
		msg = panicMessage(recovered)
	}()
	fn()
	return ""
}

func panicMessage(recovered any) string {
	switch v := recovered.(type) {
	case string:
		return v
	case error:
		return v.Error()
	default:
		return ""
	}
}

func TestRegisterSelfCheck(t *testing.T) {
	// 注册表是进程级的：键与分组名一律用 testreg_ 前缀，且每个子测试用独立分组，
	// 避免与 init 期注册的生产字段以及子测试之间互相影响。

	t.Run("坏默认值", func(t *testing.T) {
		schema.RegisterGroup(schema.Group{ID: "testreg_g", Title: "自检用", Order: 900})
		msg := mustPanic(t, func() {
			schema.Register(schema.Field{
				Key:     "testreg_bad",
				Group:   "testreg_g",
				Type:    schema.TypeInt,
				Default: "abc",
			})
		})
		if !strings.Contains(msg, "testreg_bad") {
			t.Fatalf("panic 文案应包含键名，实际为 %q", msg)
		}
		if !strings.Contains(msg, "默认值") {
			t.Fatalf("panic 文案应说明默认值非法，实际为 %q", msg)
		}
	})

	t.Run("重复键", func(t *testing.T) {
		schema.RegisterGroup(schema.Group{ID: "testreg_dup_g", Title: "重复键用", Order: 901})
		schema.Register(schema.Field{
			Key:     "testreg_dup",
			Group:   "testreg_dup_g",
			Type:    schema.TypeString,
			Default: "",
		})
		msg := mustPanic(t, func() {
			schema.Register(schema.Field{
				Key:     "testreg_dup",
				Group:   "testreg_dup_g",
				Type:    schema.TypeString,
				Default: "",
			})
		})
		if !strings.Contains(msg, "testreg_dup") {
			t.Fatalf("panic 文案应包含键名，实际为 %q", msg)
		}
		if !strings.Contains(msg, "重复") {
			t.Fatalf("panic 文案应说明键重复，实际为 %q", msg)
		}
	})

	t.Run("未注册组", func(t *testing.T) {
		msg := mustPanic(t, func() {
			schema.Register(schema.Field{
				Key:     "testreg_nogroup",
				Group:   "testreg_absent",
				Type:    schema.TypeString,
				Default: "",
			})
		})
		if !strings.Contains(msg, "testreg_nogroup") {
			t.Fatalf("panic 文案应包含键名，实际为 %q", msg)
		}
		if !strings.Contains(msg, "未注册") {
			t.Fatalf("panic 文案应说明分组未注册，实际为 %q", msg)
		}
	})
}

func TestRegistryOrdering(t *testing.T) {
	// 故意先注册 Order 大的分组，验证 Groups 真的排序而不是按插入顺序返回。
	schema.RegisterGroup(schema.Group{ID: "testreg_b", Title: "B 组", Order: 2})
	schema.RegisterGroup(schema.Group{ID: "testreg_a", Title: "A 组", Order: 1})

	zeta := schema.Field{Key: "testreg_zeta", Group: "testreg_a", Type: schema.TypeString, Default: ""}
	alpha := schema.Field{Key: "testreg_alpha", Group: "testreg_a", Type: schema.TypeString, Default: ""}
	beta := schema.Field{Key: "testreg_beta", Group: "testreg_b", Type: schema.TypeString, Default: ""}
	schema.Register(zeta)
	schema.Register(alpha)
	schema.Register(beta)

	t.Run("Groups 按 Order 升序", func(t *testing.T) {
		groups := schema.Groups()
		posA, posB := -1, -1
		for i, g := range groups {
			switch g.ID {
			case "testreg_a":
				posA = i
			case "testreg_b":
				posB = i
			}
		}
		if posA < 0 || posB < 0 {
			t.Fatalf("Groups() 缺少测试分组，实际为 %+v", groups)
		}
		if posA > posB {
			t.Fatalf("Groups() 未按 Order 升序：testreg_a 在 %d，testreg_b 在 %d", posA, posB)
		}
	})

	t.Run("FieldsOf 按 Key 升序", func(t *testing.T) {
		fields := schema.FieldsOf("testreg_a")
		if len(fields) != 2 {
			t.Fatalf("FieldsOf(\"testreg_a\") 应返回 2 个字段，实际为 %d 个", len(fields))
		}
		if !sortedByKey(fields) {
			t.Fatalf("FieldsOf 未按 Key 升序：%v", keysOf(fields))
		}
		if fields[0].Key != "testreg_alpha" || fields[1].Key != "testreg_zeta" {
			t.Fatalf("FieldsOf 返回顺序错误：%v", keysOf(fields))
		}
	})

	t.Run("FieldsOf 未知组返回 nil", func(t *testing.T) {
		if fields := schema.FieldsOf("testreg_nosuchgroup"); fields != nil {
			t.Fatalf("FieldsOf 未知组应返回 nil，实际为 %+v", fields)
		}
	})

	t.Run("All 按 Key 升序", func(t *testing.T) {
		fields := schema.All()
		if len(fields) < 3 {
			t.Fatalf("All() 至少应包含 3 个测试字段，实际为 %d 个", len(fields))
		}
		if !sortedByKey(fields) {
			t.Fatalf("All() 未按 Key 升序：%v", keysOf(fields))
		}
	})

	t.Run("Lookup", func(t *testing.T) {
		got, ok := schema.Lookup("testreg_zeta")
		if !ok {
			t.Fatalf("Lookup(\"testreg_zeta\") 应命中")
		}
		if got.Key != zeta.Key || got.Group != zeta.Group || got.Type != zeta.Type {
			t.Fatalf("Lookup 返回的字段与注册时不一致：%+v", got)
		}
		if _, ok := schema.Lookup("no_such_key"); ok {
			t.Fatalf("Lookup(\"no_such_key\") 不应命中")
		}
	})
}

func keysOf(fields []schema.Field) []string {
	keys := make([]string, 0, len(fields))
	for _, f := range fields {
		keys = append(keys, f.Key)
	}
	return keys
}

func sortedByKey(fields []schema.Field) bool {
	keys := keysOf(fields)
	return sort.SliceIsSorted(keys, func(i, j int) bool { return keys[i] < keys[j] })
}
