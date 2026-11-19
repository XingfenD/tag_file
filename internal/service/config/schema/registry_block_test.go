package schema_test

import (
	"sort"
	"strings"
	"testing"

	"github.com/XingfenD/tag_file/internal/service/config/schema"
)

// 块注册表是进程级的：块 ID 与键一律用 testblk_ 前缀，避免与后续任务
// init 期注册的生产块互相影响。断言只看测试键的相对位置，不依赖注册表里
// 的绝对内容。

// mustPanic 断言 fn 触发 panic，返回 panic 文案以便检查。原住旧 registry_test.go，
// 旧字段注册表删除后随本文件保留。
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

func TestRegisterBlockSelfCheck(t *testing.T) {
	t.Run("坏描述符", func(t *testing.T) {
		msg := mustPanic(t, func() {
			schema.RegisterBlock(schema.ConfigBlockDescriptor{
				ID:    "testblk_bad",
				Title: "自检用",
				Order: 900,
				InputBoxDescriptorList: []*schema.InputBoxDescriptor{{
					Key:       "testblk_bad_key",
					ValueType: schema.ValueInt,
					Container: schema.ContainerSingle,
					Constraint: schema.InputBoxConstraintDescriptor{
						StringConstraint: &schema.StringConstraintDescriptor{},
					},
				}},
			})
		})
		if !strings.Contains(msg, "testblk_bad_key") {
			t.Fatalf("panic 文案应包含键名，实际为 %q", msg)
		}
	})

	t.Run("默认值非法", func(t *testing.T) {
		msg := mustPanic(t, func() {
			schema.RegisterBlock(schema.ConfigBlockDescriptor{
				ID:    "testblk_baddefault",
				Title: "默认值自检用",
				Order: 905,
				InputBoxDescriptorList: []*schema.InputBoxDescriptor{{
					Key:       "testblk_baddefault_key",
					Default:   "10", // 越出 [1, 9]
					ValueType: schema.ValueInt,
					Container: schema.ContainerSingle,
					Constraint: schema.InputBoxConstraintDescriptor{
						IntConstraint: &schema.IntConstraintDescriptor{Min: schema.Ptr(1.0), Max: schema.Ptr(9.0)},
					},
				}},
			})
		})
		if !strings.Contains(msg, "testblk_baddefault_key") {
			t.Fatalf("panic 文案应包含键名，实际为 %q", msg)
		}
		if !strings.Contains(msg, "默认值") {
			t.Fatalf("panic 文案应说明默认值非法，实际为 %q", msg)
		}
	})

	t.Run("重复块", func(t *testing.T) {
		schema.RegisterBlock(schema.ConfigBlockDescriptor{
			ID:    "testblk_dupblk",
			Title: "重复块用",
			Order: 901,
			InputBoxDescriptorList: []*schema.InputBoxDescriptor{{
				Key:       "testblk_dupblk_key",
				ValueType: schema.ValueString,
				Container: schema.ContainerSingle,
			}},
		})
		msg := mustPanic(t, func() {
			schema.RegisterBlock(schema.ConfigBlockDescriptor{
				ID:    "testblk_dupblk",
				Title: "再次注册",
				Order: 902,
			})
		})
		if !strings.Contains(msg, "重复") {
			t.Fatalf("panic 文案应说明块重复，实际为 %q", msg)
		}
	})

	t.Run("重复键", func(t *testing.T) {
		schema.RegisterBlock(schema.ConfigBlockDescriptor{
			ID:    "testblk_dupkey_a",
			Title: "重复键用 A",
			Order: 903,
			InputBoxDescriptorList: []*schema.InputBoxDescriptor{{
				Key:       "testblk_dup_key",
				ValueType: schema.ValueString,
				Container: schema.ContainerSingle,
			}},
		})
		msg := mustPanic(t, func() {
			schema.RegisterBlock(schema.ConfigBlockDescriptor{
				ID:    "testblk_dupkey_b",
				Title: "重复键用 B",
				Order: 904,
				InputBoxDescriptorList: []*schema.InputBoxDescriptor{{
					Key:       "testblk_dup_key",
					ValueType: schema.ValueString,
					Container: schema.ContainerSingle,
				}},
			})
		})
		if !strings.Contains(msg, "testblk_dup_key") {
			t.Fatalf("panic 文案应包含键名，实际为 %q", msg)
		}
		if !strings.Contains(msg, "重复") {
			t.Fatalf("panic 文案应说明键重复，实际为 %q", msg)
		}
	})
}

func TestBlockRegistryOrdering(t *testing.T) {
	// 故意先注册 Order 大的块（块间验证真的排序）；块内先写 Key 大的描述符
	// （块内验证保持声明顺序，不按键名重排）。
	beta := &schema.InputBoxDescriptor{
		Key:       "testblk_beta",
		ValueType: schema.ValueString,
		Container: schema.ContainerSingle,
	}
	zeta := &schema.InputBoxDescriptor{
		Key:       "testblk_zeta",
		Default:   "zeta-default", // Required 声明下默认值必须非空
		ValueType: schema.ValueString,
		Container: schema.ContainerSingle,
		Constraint: schema.InputBoxConstraintDescriptor{
			StringConstraint: &schema.StringConstraintDescriptor{Required: true},
		},
	}
	alpha := &schema.InputBoxDescriptor{
		Key:       "testblk_alpha",
		Default:   "5", // 必须落在 [1, 9] 内，注册自检用
		ValueType: schema.ValueInt,
		Container: schema.ContainerSingle,
		Constraint: schema.InputBoxConstraintDescriptor{
			IntConstraint: &schema.IntConstraintDescriptor{Min: schema.Ptr(1.0), Max: schema.Ptr(9.0)},
		},
	}
	schema.RegisterBlock(schema.ConfigBlockDescriptor{
		ID:                     "testblk_b",
		Title:                  "B 块",
		Order:                  2,
		InputBoxDescriptorList: []*schema.InputBoxDescriptor{beta},
	})
	schema.RegisterBlock(schema.ConfigBlockDescriptor{
		ID:                     "testblk_a",
		Title:                  "A 块",
		Order:                  1,
		InputBoxDescriptorList: []*schema.InputBoxDescriptor{zeta, alpha},
	})

	t.Run("Blocks 按 Order 升序", func(t *testing.T) {
		blocks := schema.Blocks()
		posA, posB := -1, -1
		for i, b := range blocks {
			switch b.ID {
			case "testblk_a":
				posA = i
			case "testblk_b":
				posB = i
			}
		}
		if posA < 0 || posB < 0 {
			t.Fatalf("Blocks() 缺少测试块，实际为 %+v", blocks)
		}
		if posA > posB {
			t.Fatalf("Blocks() 未按 Order 升序：testblk_a 在 %d，testblk_b 在 %d", posA, posB)
		}
	})

	t.Run("Blocks 同 Order 按 ID 升序", func(t *testing.T) {
		// 故意先注册 ID 大的块，验证同 Order 时真的按 ID 决胜。
		schema.RegisterBlock(schema.ConfigBlockDescriptor{
			ID:    "testblk_tie_b",
			Title: "同序 B",
			Order: 500,
			InputBoxDescriptorList: []*schema.InputBoxDescriptor{{
				Key:       "testblk_tie_b_key",
				ValueType: schema.ValueString,
				Container: schema.ContainerSingle,
			}},
		})
		schema.RegisterBlock(schema.ConfigBlockDescriptor{
			ID:    "testblk_tie_a",
			Title: "同序 A",
			Order: 500,
			InputBoxDescriptorList: []*schema.InputBoxDescriptor{{
				Key:       "testblk_tie_a_key",
				ValueType: schema.ValueString,
				Container: schema.ContainerSingle,
			}},
		})
		blocks := schema.Blocks()
		posA, posB := -1, -1
		for i, b := range blocks {
			switch b.ID {
			case "testblk_tie_a":
				posA = i
			case "testblk_tie_b":
				posB = i
			}
		}
		if posA < 0 || posB < 0 {
			t.Fatalf("Blocks() 缺少同序测试块，实际为 %+v", blocks)
		}
		if posA > posB {
			t.Fatalf("Blocks() 同 Order 应按 ID 升序：testblk_tie_a 在 %d，testblk_tie_b 在 %d", posA, posB)
		}
	})

	t.Run("BlockOf 块内保持声明顺序", func(t *testing.T) {
		blk := schema.BlockOf("testblk_a")
		if blk == nil {
			t.Fatalf("BlockOf(\"testblk_a\") 不应返回 nil")
		}
		if len(blk.InputBoxDescriptorList) != 2 {
			t.Fatalf("BlockOf(\"testblk_a\") 应返回 2 个描述符，实际为 %d 个", len(blk.InputBoxDescriptorList))
		}
		// 声明顺序即设置页布局顺序：zeta 先于 alpha 声明，读取端不得按键名重排。
		if blk.InputBoxDescriptorList[0].Key != "testblk_zeta" || blk.InputBoxDescriptorList[1].Key != "testblk_alpha" {
			t.Fatalf("BlockOf 块内描述符未保持声明顺序：%v", keysOfBoxes(blk.InputBoxDescriptorList))
		}
	})

	t.Run("BlockOf 未知块返回 nil", func(t *testing.T) {
		if blk := schema.BlockOf("testblk_absent"); blk != nil {
			t.Fatalf("BlockOf(\"testblk_absent\") 应返回 nil，实际为 %+v", blk)
		}
	})

	t.Run("All 跨块按 Key 升序", func(t *testing.T) {
		boxes := schema.All()
		if len(boxes) < 3 {
			t.Fatalf("All() 至少应包含 3 个测试描述符，实际为 %d 个", len(boxes))
		}
		keys := keysOfBoxes(boxes)
		if !sort.SliceIsSorted(keys, func(i, j int) bool { return keys[i] < keys[j] }) {
			t.Fatalf("All() 未按 Key 升序：%v", keys)
		}
		posAlpha, posBeta, posZeta := -1, -1, -1
		for i, k := range keys {
			switch k {
			case "testblk_alpha":
				posAlpha = i
			case "testblk_beta":
				posBeta = i
			case "testblk_zeta":
				posZeta = i
			}
		}
		if posAlpha < 0 || posBeta < 0 || posZeta < 0 {
			t.Fatalf("All() 缺少测试键：%v", keys)
		}
		if !(posAlpha < posBeta && posBeta < posZeta) {
			t.Fatalf("All() 测试键相对顺序错误：%v", keys)
		}
	})

	t.Run("Lookup", func(t *testing.T) {
		got, ok := schema.Lookup("testblk_zeta")
		if !ok {
			t.Fatalf("Lookup(\"testblk_zeta\") 应命中")
		}
		if got.Key != zeta.Key || got.ValueType != zeta.ValueType || got.Container != zeta.Container {
			t.Fatalf("Lookup 返回的描述符与注册时不一致：%+v", got)
		}
		if got.Constraint.StringConstraint == nil || !got.Constraint.StringConstraint.Required {
			t.Fatalf("Lookup 返回的约束与注册时不一致：%+v", got.Constraint.StringConstraint)
		}
		if _, ok := schema.Lookup("no_such_key"); ok {
			t.Fatalf("Lookup(\"no_such_key\") 不应命中")
		}
	})
}

func keysOfBoxes(boxes []*schema.InputBoxDescriptor) []string {
	keys := make([]string, 0, len(boxes))
	for _, d := range boxes {
		keys = append(keys, d.Key)
	}
	return keys
}
