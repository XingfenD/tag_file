package ai

import "github.com/XingfenD/tag_file/internal/service/config/schema"

// 本文件把 AI 识别用到的 21 个配置键登记进块注册表（spec §10 映射），供设置页
// 与写入校验读取。默认值与边界都引用 config.go 里的常量：注册表只做声明，
// 取值语义仍住在消费方，因此设置页展示的默认值和 Settings 的回落值不可能不一致。
// ai_provider 是封闭枚举，per-provider 的地址/模型/密钥按它条件显隐；
// 整理端点组（ai_refine_*）声明在识图组之后、抽帧字段之前，枚举 off 为首项
// 与默认值，其条件字段挂 ai_refine_provider——与识图组互不干扰；
// 两组 provider 的条件字段一律按「地址 → 模型 → 密钥」同序声明，前端按
// 序位分槽渲染，切换端点类型时其余配置项位置不动。
// ai_frame_interval 是定点数：wire 整数 10 = 显示 1.0 秒（Shift=1），
// 最小值 0 开区间（0 会让抽帧失去意义），故 Max 留空。
func init() {
	schema.RegisterBlock(schema.ConfigBlockDescriptor{
		ID:    "ai",
		Title: "AI 识别",
		Order: 2,
		InputBoxDescriptorList: []*schema.InputBoxDescriptor{
			{
				Key:         KeyProvider,
				Label:       "推理端点类型",
				ValueType:   schema.ValueString,
				Container:   schema.ContainerSingle,
				Default:     DefaultProvider,
				Constraint: schema.InputBoxConstraintDescriptor{
					StringConstraint: &schema.StringConstraintDescriptor{
						Options: []schema.Option{
							{Value: ProviderOllama, Label: "Ollama"},
							{Value: ProviderOpenAI, Label: "OpenAI 兼容"},
							{Value: ProviderAnthropic, Label: "Anthropic"},
						},
					},
				},
			},
			{
				Key:         KeyOllamaURL,
				Label:       "Ollama 服务地址",
				ShowIf:      &schema.Condition{Key: KeyProvider, Equals: ProviderOllama},
				ValueType:   schema.ValueString,
				Container:   schema.ContainerSingle,
				Default:     DefaultOllamaURL,
				Constraint: schema.InputBoxConstraintDescriptor{
					StringConstraint: &schema.StringConstraintDescriptor{
						Required: true,
						Format:   schema.FormatURL,
					},
				},
			},
			{
				Key:         KeyOllamaModel,
				Label:       "Ollama 模型ID",
				ShowIf:      &schema.Condition{Key: KeyProvider, Equals: ProviderOllama},
				ValueType:   schema.ValueString,
				Container:   schema.ContainerSingle,
				Default:     DefaultOllamaModel,
			},
			{
				Key:         KeyOpenAIURL,
				Label:       "OpenAI 服务地址",
				Description: "OpenAI 兼容服务地址，须包含 /v1 前缀（如 https://api.openai.com/v1）",
				ShowIf:      &schema.Condition{Key: KeyProvider, Equals: ProviderOpenAI},
				ValueType:   schema.ValueString,
				Container:   schema.ContainerSingle,
				Default:     DefaultOpenAIURL,
				Constraint: schema.InputBoxConstraintDescriptor{
					StringConstraint: &schema.StringConstraintDescriptor{Required: true, Format: schema.FormatURL},
				},
			},
			{
				Key:         KeyOpenAIModel,
				Label:       "OpenAI 模型ID",
				ShowIf:      &schema.Condition{Key: KeyProvider, Equals: ProviderOpenAI},
				ValueType:   schema.ValueString,
				Container:   schema.ContainerSingle,
				Default:     DefaultOpenAIModel,
			},
			{
				Key:         KeyOpenAIAPIKey,
				Label:       "OpenAI API Key",
				ShowIf:      &schema.Condition{Key: KeyProvider, Equals: ProviderOpenAI},
				ValueType:   schema.ValueString,
				Container:   schema.ContainerSingle,
				Default:     DefaultOpenAIAPIKey,
				Constraint: schema.InputBoxConstraintDescriptor{
					StringConstraint: &schema.StringConstraintDescriptor{Secret: true},
				},
			},
			{
				Key:         KeyAnthropicURL,
				Label:       "Anthropic 服务地址",
				ShowIf:      &schema.Condition{Key: KeyProvider, Equals: ProviderAnthropic},
				ValueType:   schema.ValueString,
				Container:   schema.ContainerSingle,
				Default:     DefaultAnthropicURL,
				Constraint: schema.InputBoxConstraintDescriptor{
					StringConstraint: &schema.StringConstraintDescriptor{Required: true, Format: schema.FormatURL},
				},
			},
			{
				Key:         KeyAnthropicModel,
				Label:       "Anthropic 模型ID",
				ShowIf:      &schema.Condition{Key: KeyProvider, Equals: ProviderAnthropic},
				ValueType:   schema.ValueString,
				Container:   schema.ContainerSingle,
				Default:     DefaultAnthropicModel,
			},
			{
				Key:         KeyAnthropicAPIKey,
				Label:       "Anthropic API Key",
				ShowIf:      &schema.Condition{Key: KeyProvider, Equals: ProviderAnthropic},
				ValueType:   schema.ValueString,
				Container:   schema.ContainerSingle,
				Default:     DefaultAnthropicAPIKey,
				Constraint: schema.InputBoxConstraintDescriptor{
					StringConstraint: &schema.StringConstraintDescriptor{Secret: true},
				},
			},
			// 整理端点组：URL/模型/密钥的默认值直接复用识图组常量，两组声明
			// 不可能漂移；枚举默认 off，新库保持今天的单段行为。
			{
				Key:         KeyRefineProvider,
				Label:       "整理端点类型",
				ValueType:   schema.ValueString,
				Container:   schema.ContainerSingle,
				Default:     DefaultRefineProvider,
				Constraint: schema.InputBoxConstraintDescriptor{
					StringConstraint: &schema.StringConstraintDescriptor{
						Options: []schema.Option{
							{Value: RefineOff, Label: "关闭"},
							{Value: ProviderOllama, Label: "Ollama"},
							{Value: ProviderOpenAI, Label: "OpenAI 兼容"},
							{Value: ProviderAnthropic, Label: "Anthropic"},
						},
					},
				},
			},
			{
				Key:         KeyRefineOllamaURL,
				Label:       "整理 Ollama 服务地址",
				ShowIf:      &schema.Condition{Key: KeyRefineProvider, Equals: ProviderOllama},
				ValueType:   schema.ValueString,
				Container:   schema.ContainerSingle,
				Default:     DefaultOllamaURL,
				Constraint: schema.InputBoxConstraintDescriptor{
					StringConstraint: &schema.StringConstraintDescriptor{Required: true, Format: schema.FormatURL},
				},
			},
			{
				Key:         KeyRefineOllamaModel,
				Label:       "整理 Ollama 模型ID",
				ShowIf:      &schema.Condition{Key: KeyRefineProvider, Equals: ProviderOllama},
				ValueType:   schema.ValueString,
				Container:   schema.ContainerSingle,
				Default:     DefaultOllamaModel,
			},
			{
				Key:         KeyRefineOpenAIURL,
				Label:       "整理 OpenAI 服务地址",
				Description: "OpenAI 兼容服务地址，须包含 /v1 前缀（如 https://api.openai.com/v1）",
				ShowIf:      &schema.Condition{Key: KeyRefineProvider, Equals: ProviderOpenAI},
				ValueType:   schema.ValueString,
				Container:   schema.ContainerSingle,
				Default:     DefaultOpenAIURL,
				Constraint: schema.InputBoxConstraintDescriptor{
					StringConstraint: &schema.StringConstraintDescriptor{Required: true, Format: schema.FormatURL},
				},
			},
			{
				Key:         KeyRefineOpenAIModel,
				Label:       "整理 OpenAI 模型ID",
				ShowIf:      &schema.Condition{Key: KeyRefineProvider, Equals: ProviderOpenAI},
				ValueType:   schema.ValueString,
				Container:   schema.ContainerSingle,
				Default:     DefaultOpenAIModel,
			},
			{
				Key:         KeyRefineOpenAIAPIKey,
				Label:       "整理 OpenAI API Key",
				ShowIf:      &schema.Condition{Key: KeyRefineProvider, Equals: ProviderOpenAI},
				ValueType:   schema.ValueString,
				Container:   schema.ContainerSingle,
				Default:     DefaultOpenAIAPIKey,
				Constraint: schema.InputBoxConstraintDescriptor{
					StringConstraint: &schema.StringConstraintDescriptor{Secret: true},
				},
			},
			{
				Key:         KeyRefineAnthropicURL,
				Label:       "整理 Anthropic 服务地址",
				ShowIf:      &schema.Condition{Key: KeyRefineProvider, Equals: ProviderAnthropic},
				ValueType:   schema.ValueString,
				Container:   schema.ContainerSingle,
				Default:     DefaultAnthropicURL,
				Constraint: schema.InputBoxConstraintDescriptor{
					StringConstraint: &schema.StringConstraintDescriptor{Required: true, Format: schema.FormatURL},
				},
			},
			{
				Key:         KeyRefineAnthropicModel,
				Label:       "整理 Anthropic 模型ID",
				ShowIf:      &schema.Condition{Key: KeyRefineProvider, Equals: ProviderAnthropic},
				ValueType:   schema.ValueString,
				Container:   schema.ContainerSingle,
				Default:     DefaultAnthropicModel,
			},
			{
				Key:         KeyRefineAnthropicAPIKey,
				Label:       "整理 Anthropic API Key",
				ShowIf:      &schema.Condition{Key: KeyRefineProvider, Equals: ProviderAnthropic},
				ValueType:   schema.ValueString,
				Container:   schema.ContainerSingle,
				Default:     DefaultAnthropicAPIKey,
				Constraint: schema.InputBoxConstraintDescriptor{
					StringConstraint: &schema.StringConstraintDescriptor{Secret: true},
				},
			},
			{
				Key:         KeyFrameInterval,
				Label:       "抽帧间隔",
				Description: "视频最小抽帧间隔；长视频会自动放大间隔以覆盖全片，分镜切换点始终额外采样",
				Unit:        "秒",
				ValueType:   schema.ValueInt,
				Container:   schema.ContainerSingle,
				Default:     DefaultFrameInterval,
				Constraint: schema.InputBoxConstraintDescriptor{
					IntConstraint: &schema.IntConstraintDescriptor{
						Min:          schema.Ptr(0.0),
						MinExclusive: true,
						Shift:        FrameIntervalShift,
					},
				},
			},
			{
				// 上限取自 MaxDiffThreshold：dHash 是 64 位，阈值超过 64 会丢弃
				// 除首帧外的所有帧。
				Key:         KeyDiffThreshold,
				Label:       "近似帧过滤阈值",
				ValueType:   schema.ValueInt,
				Container:   schema.ContainerSingle,
				Default:     DefaultDiffThreshold,
				Constraint: schema.InputBoxConstraintDescriptor{
					IntConstraint: &schema.IntConstraintDescriptor{
						Min: schema.Ptr(1.0),
						Max: schema.Ptr(float64(MaxDiffThreshold)),
					},
				},
			},
			{
				Key:         KeyMaxFrames,
				Label:       "单视频最大帧数",
				ValueType:   schema.ValueInt,
				Container:   schema.ContainerSingle,
				Default:     DefaultMaxFrames,
				Constraint: schema.InputBoxConstraintDescriptor{
					IntConstraint: &schema.IntConstraintDescriptor{
						Min: schema.Ptr(1.0),
						Max: schema.Ptr(float64(MaxFramesLimit)),
					},
				},
			},
		},
	})
}
