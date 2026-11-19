package ai_test

import (
	"testing"

	"github.com/XingfenD/tag_file/internal/service/ai"
	"github.com/XingfenD/tag_file/internal/service/config/schema"
)

// lookupInputBox 取出输入框描述符并断言注册成功，失败信息带上键名便于定位。
func lookupInputBox(t *testing.T, key string) *schema.InputBoxDescriptor {
	t.Helper()

	d, ok := schema.Lookup(key)
	if !ok {
		t.Fatalf("schema.Lookup(%q) 未找到输入框，ok=false", key)
	}
	return d
}

func TestRegisterAIFields(t *testing.T) {
	cases := []struct {
		key       string
		wantType  schema.ValueType
		wantValue string
	}{
		{ai.KeyOllamaURL, schema.ValueString, ai.DefaultOllamaURL},
		{ai.KeyOllamaModel, schema.ValueString, ai.DefaultOllamaModel},
		// 定点数存储：wire 整数 10 = 显示 1.0 秒（Shift=1），Task 5 起引用 DefaultFrameInterval。
		{ai.KeyFrameInterval, schema.ValueInt, "10"},
		{ai.KeyDiffThreshold, schema.ValueInt, ai.DefaultDiffThreshold},
		{ai.KeyMaxFrames, schema.ValueInt, ai.DefaultMaxFrames},
		{ai.KeyProvider, schema.ValueString, ai.DefaultProvider},
		{ai.KeyOpenAIURL, schema.ValueString, ai.DefaultOpenAIURL},
		{ai.KeyOpenAIModel, schema.ValueString, ai.DefaultOpenAIModel},
		{ai.KeyOpenAIAPIKey, schema.ValueString, ai.DefaultOpenAIAPIKey},
		{ai.KeyAnthropicURL, schema.ValueString, ai.DefaultAnthropicURL},
		{ai.KeyAnthropicModel, schema.ValueString, ai.DefaultAnthropicModel},
		{ai.KeyAnthropicAPIKey, schema.ValueString, ai.DefaultAnthropicAPIKey},
	}

	for _, tc := range cases {
		t.Run(tc.key, func(t *testing.T) {
			d := lookupInputBox(t, tc.key)

			if d.Key != tc.key {
				t.Errorf("Key = %q, want %q", d.Key, tc.key)
			}
			if d.Container != schema.ContainerSingle {
				t.Errorf("Container = %q, want %q", d.Container, schema.ContainerSingle)
			}
			if d.ValueType != tc.wantType {
				t.Errorf("ValueType = %q, want %q", d.ValueType, tc.wantType)
			}
			if d.Default != tc.wantValue {
				t.Errorf("Default = %q, want %q", d.Default, tc.wantValue)
			}
			if d.Label == "" {
				t.Error("Label 为空")
			}
			if err := schema.Validate(*d, d.Default); err != nil {
				t.Errorf("默认值 %q 校验失败: %v", d.Default, err)
			}
		})
	}
}

// TestRegisterAIBlock 钉住块级声明：ID/标题/顺序与 12 个键的完整集合。
func TestRegisterAIBlock(t *testing.T) {
	b := schema.BlockOf("ai")
	if b == nil {
		t.Fatal(`schema.BlockOf("ai") 未找到块`)
	}
	if b.Title != "AI 识别" {
		t.Errorf("Title = %q, want %q", b.Title, "AI 识别")
	}
	if b.Order != 2 {
		t.Errorf("Order = %d, want 2", b.Order)
	}

	want := map[string]bool{
		ai.KeyProvider:        true,
		ai.KeyOllamaURL:       true,
		ai.KeyOllamaModel:     true,
		ai.KeyOpenAIURL:       true,
		ai.KeyOpenAIModel:     true,
		ai.KeyOpenAIAPIKey:    true,
		ai.KeyAnthropicURL:    true,
		ai.KeyAnthropicModel:  true,
		ai.KeyAnthropicAPIKey: true,
		ai.KeyFrameInterval:   true,
		ai.KeyDiffThreshold:   true,
		ai.KeyMaxFrames:       true,
	}
	if len(b.InputBoxDescriptorList) != len(want) {
		t.Fatalf("块内描述符数 = %d, want %d", len(b.InputBoxDescriptorList), len(want))
	}
	for _, d := range b.InputBoxDescriptorList {
		if !want[d.Key] {
			t.Errorf("块内出现意外键 %q", d.Key)
		}
	}
}

// 三个 provider URL 的指令组必须含 required 与 url（spec §10 映射）。
func TestRegisterAIProviderURLRules(t *testing.T) {
	for _, key := range []string{ai.KeyOllamaURL, ai.KeyOpenAIURL, ai.KeyAnthropicURL} {
		t.Run(key, func(t *testing.T) {
			d := lookupInputBox(t, key)

			rs := schema.RulesOf(*d)
			hasRequired, hasURL := false, false
			for _, r := range rs.Rules {
				switch r.Kind {
				case schema.RuleRequired:
					hasRequired = true
				case schema.RuleURL:
					hasURL = true
				}
			}
			if !hasRequired || !hasURL {
				t.Fatalf("%s 指令组 = %v, want 含 required+url", key, rs.Rules)
			}
		})
	}
}

// 空的模型名表示「未配置」：无任何指令，空串是合法取值，
// 否则用户无法保存未配置状态。
func TestRegisterAIModelEmptyIsValid(t *testing.T) {
	for _, key := range []string{ai.KeyOllamaModel, ai.KeyOpenAIModel, ai.KeyAnthropicModel} {
		t.Run(key, func(t *testing.T) {
			d := lookupInputBox(t, key)

			if d.Default != "" {
				t.Fatalf("%s Default = %q, want 空串", key, d.Default)
			}
			if rs := schema.RulesOf(*d); len(rs.Rules) != 0 {
				t.Errorf("%s 指令组 = %v, want 无指令", key, rs.Rules)
			}
			if err := schema.Validate(*d, ""); err != nil {
				t.Errorf("Validate(%s, \"\") = %v, want nil", key, err)
			}
		})
	}
}

func TestRegisterAIProviderEnum(t *testing.T) {
	d := lookupInputBox(t, ai.KeyProvider)

	rs := schema.RulesOf(*d)
	hasOneOf := false
	for _, r := range rs.Rules {
		if r.Kind == schema.RuleOneOf {
			hasOneOf = true
		}
	}
	if !hasOneOf {
		t.Fatal("ai_provider 指令组缺少 oneof（设置页不会渲染下拉）")
	}
	for _, v := range []string{ai.ProviderOllama, ai.ProviderOpenAI, ai.ProviderAnthropic} {
		if err := schema.Validate(*d, v); err != nil {
			t.Errorf("Validate(%q) = %v, want nil", v, err)
		}
	}
	if err := schema.Validate(*d, "bogus"); err == nil {
		t.Error(`Validate("bogus") = nil, want oneof 拒绝写入`)
	}
}

func TestRegisterAIProviderShowIf(t *testing.T) {
	cases := []struct{ key, equals string }{
		{ai.KeyOllamaURL, ai.ProviderOllama},
		{ai.KeyOllamaModel, ai.ProviderOllama},
		{ai.KeyOpenAIURL, ai.ProviderOpenAI},
		{ai.KeyOpenAIModel, ai.ProviderOpenAI},
		{ai.KeyOpenAIAPIKey, ai.ProviderOpenAI},
		{ai.KeyAnthropicURL, ai.ProviderAnthropic},
		{ai.KeyAnthropicModel, ai.ProviderAnthropic},
		{ai.KeyAnthropicAPIKey, ai.ProviderAnthropic},
	}
	for _, tc := range cases {
		d := lookupInputBox(t, tc.key)
		if d.ShowIf == nil {
			t.Fatalf("%s 缺少 ShowIf", tc.key)
		}
		if d.ShowIf.Key != ai.KeyProvider || d.ShowIf.Equals != tc.equals {
			t.Errorf("%s ShowIf = %+v, want {ai_provider, %s}", tc.key, d.ShowIf, tc.equals)
		}
	}
	for _, key := range []string{ai.KeyProvider, ai.KeyFrameInterval, ai.KeyDiffThreshold, ai.KeyMaxFrames} {
		if d := lookupInputBox(t, key); d.ShowIf != nil {
			t.Errorf("%s 不应有 ShowIf, got %+v", key, d.ShowIf)
		}
	}
}

func TestRegisterAIAPIKeysAreSecret(t *testing.T) {
	for _, key := range []string{ai.KeyOpenAIAPIKey, ai.KeyAnthropicAPIKey} {
		d := lookupInputBox(t, key)
		if !d.IsSecret() {
			t.Errorf("%s 应为 secret（掩码 + 留空不覆盖）", key)
		}
		if err := schema.Validate(*d, ""); err != nil {
			t.Errorf("%s 空串应合法（未配置）: %v", key, err)
		}
		if err := schema.Validate(*d, "sk-test"); err != nil {
			t.Errorf("%s 非空密钥应合法: %v", key, err)
		}
	}
	for _, key := range []string{ai.KeyOllamaModel, ai.KeyOpenAIModel, ai.KeyAnthropicModel} {
		if d := lookupInputBox(t, key); d.IsSecret() {
			t.Errorf("%s 是模型名, 不应是 secret", key)
		}
	}
}

func TestRegisterAIFieldBounds(t *testing.T) {
	t.Run("ai_frame_interval", func(t *testing.T) {
		d := lookupInputBox(t, ai.KeyFrameInterval)

		c := d.Constraint.IntConstraint
		if c == nil {
			t.Fatal("IntConstraint 为 nil")
		}
		if c.Min == nil || *c.Min != 0 {
			t.Fatalf("Min = %v, want 0", c.Min)
		}
		if !c.MinExclusive {
			t.Error("MinExclusive = false, want true")
		}
		if c.Max != nil {
			t.Errorf("Max = %v, want nil", *c.Max)
		}
		if c.Shift != 1 {
			t.Errorf("Shift = %d, want 1", c.Shift)
		}
		if d.Unit != "秒" {
			t.Errorf("Unit = %q, want %q", d.Unit, "秒")
		}
	})

	t.Run("ai_frame_diff_threshold", func(t *testing.T) {
		d := lookupInputBox(t, ai.KeyDiffThreshold)

		c := d.Constraint.IntConstraint
		if c == nil {
			t.Fatal("IntConstraint 为 nil")
		}
		if c.Min == nil || *c.Min != 1 {
			t.Fatalf("Min = %v, want 1", c.Min)
		}
		if c.Max == nil || *c.Max != float64(ai.MaxDiffThreshold) {
			t.Fatalf("Max = %v, want %d", c.Max, ai.MaxDiffThreshold)
		}
		if c.MinExclusive || c.MaxExclusive {
			t.Error("边界应为闭区间")
		}
	})

	t.Run("ai_max_frames", func(t *testing.T) {
		d := lookupInputBox(t, ai.KeyMaxFrames)

		c := d.Constraint.IntConstraint
		if c == nil {
			t.Fatal("IntConstraint 为 nil")
		}
		if c.Min == nil || *c.Min != 1 {
			t.Fatalf("Min = %v, want 1", c.Min)
		}
		if c.Max == nil || *c.Max != float64(ai.MaxFramesLimit) {
			t.Fatalf("Max = %v, want %d", c.Max, ai.MaxFramesLimit)
		}
		if c.MinExclusive || c.MaxExclusive {
			t.Error("边界应为闭区间")
		}
	})
}
