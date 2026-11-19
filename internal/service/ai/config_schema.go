package ai

import "github.com/XingfenD/tag_file/internal/service/config/schema"

// 本文件把 AI 识别用到的 12 个配置键登记进块注册表（spec §10 映射），供设置页
// 与写入校验读取。默认值与边界都引用 config.go 里的常量：注册表只做声明，
// 取值语义仍住在消费方，因此设置页展示的默认值和 Settings 的回落值不可能不一致。
// ai_provider 是封闭枚举，per-provider 的地址/模型/密钥按它条件显隐；
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
				Description: "视觉模型推理服务的接口格式",
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
				Key:         KeyOllamaModel,
				Label:       "Ollama 视觉模型",
				Description: "Ollama 视觉模型名称（空表示未配置，识别时报错）",
				ShowIf:      &schema.Condition{Key: KeyProvider, Equals: ProviderOllama},
				ValueType:   schema.ValueString,
				Container:   schema.ContainerSingle,
				Default:     DefaultOllamaModel,
			},
			{
				Key:         KeyOllamaURL,
				Label:       "Ollama 服务地址",
				Description: "Ollama 服务地址",
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
				Label:       "OpenAI 视觉模型",
				Description: "OpenAI 兼容视觉模型名称（空表示未配置，识别时报错）",
				ShowIf:      &schema.Condition{Key: KeyProvider, Equals: ProviderOpenAI},
				ValueType:   schema.ValueString,
				Container:   schema.ContainerSingle,
				Default:     DefaultOpenAIModel,
			},
			{
				Key:         KeyOpenAIAPIKey,
				Label:       "OpenAI API Key",
				Description: "OpenAI 兼容服务 API 密钥（本地服务可留空）",
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
				Description: "Anthropic 服务地址（如 https://api.anthropic.com）",
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
				Label:       "Anthropic 视觉模型",
				Description: "Anthropic 视觉模型名称（空表示未配置，识别时报错）",
				ShowIf:      &schema.Condition{Key: KeyProvider, Equals: ProviderAnthropic},
				ValueType:   schema.ValueString,
				Container:   schema.ContainerSingle,
				Default:     DefaultAnthropicModel,
			},
			{
				Key:         KeyAnthropicAPIKey,
				Label:       "Anthropic API Key",
				Description: "Anthropic API 密钥（留空表示未配置）",
				ShowIf:      &schema.Condition{Key: KeyProvider, Equals: ProviderAnthropic},
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
				Description: "视频最小抽帧间隔（秒）；长视频会自动放大间隔以覆盖全片，分镜切换点始终额外采样",
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
				Description: "近似帧过滤阈值（与所有已保留帧的 dHash 汉明距离均小于该值的帧被丢弃）",
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
				// 上限取自 MaxFramesLimit：抽出的每一帧都会读回内存，无上限会被
				// 一次笔误打爆。
				Key:         KeyMaxFrames,
				Label:       "单视频最大帧数",
				Description: "单个视频最多保留的帧数（逐帧送模型识别）",
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
