package schema_test

import (
	"testing"

	"github.com/XingfenD/tag_file/internal/service/config/schema"
)

func TestValidate(t *testing.T) {
	cases := []struct {
		name  string
		f     schema.Field
		value string
		ok    bool
	}{
		{"string 空串合法", schema.Field{Type: schema.TypeString}, "", true},
		{"string pattern 命中", schema.Field{Type: schema.TypeString, Pattern: `^[a-z]+$`}, "abc", true},
		{"string pattern 不命中", schema.Field{Type: schema.TypeString, Pattern: `^[a-z]+$`}, "A1", false},
		{"string 空串跳过 pattern", schema.Field{Type: schema.TypeString, Pattern: `^[a-z]+$`}, "", true},
		{"url 合法", schema.Field{Type: schema.TypeURL}, "http://127.0.0.1:11434", true},
		{"url 空串非法", schema.Field{Type: schema.TypeURL}, "", false},
		{"url 无 scheme 非法", schema.Field{Type: schema.TypeURL}, "127.0.0.1:11434", false},
		{"url ftp 非法", schema.Field{Type: schema.TypeURL}, "ftp://x/y", false},
		{"int 边界闭区间", schema.Field{Type: schema.TypeInt, Min: schema.Ptr(1.0), Max: schema.Ptr(64.0)}, "64", true},
		{"int 越上界", schema.Field{Type: schema.TypeInt, Min: schema.Ptr(1.0), Max: schema.Ptr(64.0)}, "65", false},
		{"int 非数字", schema.Field{Type: schema.TypeInt}, "not-a-number", false},
		{"float 排他下界拒 0", schema.Field{Type: schema.TypeFloat, Min: schema.Ptr(0.0), MinExclusive: true}, "0", false},
		{"float 排他下界收小正数", schema.Field{Type: schema.TypeFloat, Min: schema.Ptr(0.0), MinExclusive: true}, "0.05", true},
		{"float 拒负数", schema.Field{Type: schema.TypeFloat, Min: schema.Ptr(0.0), MinExclusive: true}, "-1", false},
		{"float 拒 NaN", schema.Field{Type: schema.TypeFloat, Min: schema.Ptr(0.0), MinExclusive: true}, "NaN", false},
		{"float 拒 +Inf", schema.Field{Type: schema.TypeFloat, Min: schema.Ptr(0.0), MinExclusive: true}, "+Inf", false},
		{"bool 只认 true/false", schema.Field{Type: schema.TypeBool}, "TRUE", false},
		{"enum 命中选项", schema.Field{Type: schema.TypeEnum, Options: []schema.Option{{Value: "a"}, {Value: "b"}}}, "b", true},
		{"enum 未命中", schema.Field{Type: schema.TypeEnum, Options: []schema.Option{{Value: "a"}}}, "c", false},
		{"list 合法 JSON 正则数组", schema.Field{Type: schema.TypeList, ItemType: schema.TypeRegex}, `["^\\.","^x$"]`, true},
		{"list 空数组合法", schema.Field{Type: schema.TypeList, ItemType: schema.TypeRegex}, `[]`, true},
		{"list 非法 JSON", schema.Field{Type: schema.TypeList, ItemType: schema.TypeRegex}, `not json`, false},
		{"list 非法正则项", schema.Field{Type: schema.TypeList, ItemType: schema.TypeRegex}, `["("]`, false},
		{"list 非数组 JSON", schema.Field{Type: schema.TypeList, ItemType: schema.TypeRegex}, `{"a":1}`, false},
		{"secret 任意串合法", schema.Field{Type: schema.TypeSecret}, "sk-xxx", true},
		{"duration 正秒数", schema.Field{Type: schema.TypeDuration, Min: schema.Ptr(0.0), MinExclusive: true}, "180", true},
		{"duration 拒 NaN", schema.Field{Type: schema.TypeDuration, Min: schema.Ptr(0.0), MinExclusive: true}, "NaN", false},
		{"duration 拒 +Inf", schema.Field{Type: schema.TypeDuration, Min: schema.Ptr(0.0), MinExclusive: true}, "+Inf", false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := schema.Validate(c.f, c.value)
			if c.ok {
				if err != nil {
					t.Fatalf("期望校验通过，实际报错: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("期望校验失败，实际通过")
			}
			if err.Error() == "" {
				t.Fatalf("校验错误文案不能为空")
			}
		})
	}
}
