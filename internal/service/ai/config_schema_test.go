package ai_test

import (
	"testing"

	"github.com/XingfenD/tag_file/internal/service/ai"
	"github.com/XingfenD/tag_file/internal/service/config/schema"
)

// lookupField 取出字段并断言注册成功，失败信息带上键名便于定位。
func lookupField(t *testing.T, key string) schema.Field {
	t.Helper()

	f, ok := schema.Lookup(key)
	if !ok {
		t.Fatalf("schema.Lookup(%q) 未找到字段，ok=false", key)
	}
	return f
}

func TestRegisterAIFields(t *testing.T) {
	cases := []struct {
		key       string
		wantType  schema.FieldType
		wantValue string
	}{
		{ai.KeyOllamaURL, schema.TypeURL, ai.DefaultOllamaURL},
		{ai.KeyOllamaModel, schema.TypeString, ai.DefaultOllamaModel},
		{ai.KeyFrameInterval, schema.TypeFloat, ai.DefaultFrameInterval},
		{ai.KeyDiffThreshold, schema.TypeInt, ai.DefaultDiffThreshold},
		{ai.KeyMaxFrames, schema.TypeInt, ai.DefaultMaxFrames},
	}

	for _, tc := range cases {
		t.Run(tc.key, func(t *testing.T) {
			f := lookupField(t, tc.key)

			if f.Key != tc.key {
				t.Errorf("Key = %q, want %q", f.Key, tc.key)
			}
			if f.Group != "ai" {
				t.Errorf("Group = %q, want %q", f.Group, "ai")
			}
			if f.Type != tc.wantType {
				t.Errorf("Type = %q, want %q", f.Type, tc.wantType)
			}
			if f.Default != tc.wantValue {
				t.Errorf("Default = %q, want %q", f.Default, tc.wantValue)
			}
			if f.Label == "" {
				t.Error("Label 为空")
			}
			if f.Description == "" {
				t.Error("Description 为空")
			}
			if err := schema.Validate(f, f.Default); err != nil {
				t.Errorf("默认值 %q 校验失败: %v", f.Default, err)
			}
		})
	}
}

// 空的 ollama_model 表示「未配置」，校验必须放行，否则用户无法保存未配置状态。
func TestRegisterAIFieldsOllamaModelEmptyIsValid(t *testing.T) {
	f := lookupField(t, ai.KeyOllamaModel)

	if f.Default != "" {
		t.Fatalf("ollama_model Default = %q, want 空串", f.Default)
	}
	if err := schema.Validate(f, ""); err != nil {
		t.Errorf("Validate(ollama_model, \"\") = %v, want nil", err)
	}
}

func TestRegisterAIFieldBounds(t *testing.T) {
	t.Run("ai_frame_interval", func(t *testing.T) {
		f := lookupField(t, ai.KeyFrameInterval)

		if f.Min == nil || *f.Min != 0 {
			t.Fatalf("Min = %v, want 0", f.Min)
		}
		if !f.MinExclusive {
			t.Error("MinExclusive = false, want true")
		}
		if f.Max != nil {
			t.Errorf("Max = %v, want nil", *f.Max)
		}
		if f.Unit != "秒" {
			t.Errorf("Unit = %q, want %q", f.Unit, "秒")
		}
	})

	t.Run("ai_frame_diff_threshold", func(t *testing.T) {
		f := lookupField(t, ai.KeyDiffThreshold)

		if f.Min == nil || *f.Min != 1 {
			t.Fatalf("Min = %v, want 1", f.Min)
		}
		if f.Max == nil || *f.Max != float64(ai.MaxDiffThreshold) {
			t.Fatalf("Max = %v, want %d", f.Max, ai.MaxDiffThreshold)
		}
		if f.MinExclusive || f.MaxExclusive {
			t.Error("边界应为闭区间")
		}
	})

	t.Run("ai_max_frames", func(t *testing.T) {
		f := lookupField(t, ai.KeyMaxFrames)

		if f.Min == nil || *f.Min != 1 {
			t.Fatalf("Min = %v, want 1", f.Min)
		}
		if f.Max == nil || *f.Max != float64(ai.MaxFramesLimit) {
			t.Fatalf("Max = %v, want %d", f.Max, ai.MaxFramesLimit)
		}
		if f.MinExclusive || f.MaxExclusive {
			t.Error("边界应为闭区间")
		}
	})
}
