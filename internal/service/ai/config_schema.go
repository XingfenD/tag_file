package ai

import "github.com/XingfenD/tag_file/internal/service/config/schema"

// 本文件把 AI 识别用到的 5 个配置键登记进 schema 注册表，供设置页与写入校验读取。
// 默认值与边界一律引用 config.go 里的常量：注册表只做声明，取值语义仍住在消费方，
// 因此设置页展示的默认值和 Settings 的回落值不可能不一致。
func init() {
	schema.RegisterGroup(schema.Group{ID: "ai", Title: "AI 识别", Order: 2})

	// 空模型名表示「未配置」，TypeString 的空串是合法取值，识别路径会自行报错。
	schema.Register(schema.Field{
		Key:         KeyOllamaModel,
		Group:       "ai",
		Label:       "Ollama 视觉模型",
		Description: "Ollama 视觉模型名称（空表示未配置，识别时报错）",
		Type:        schema.TypeString,
		Default:     DefaultOllamaModel,
	})

	schema.Register(schema.Field{
		Key:         KeyOllamaURL,
		Group:       "ai",
		Label:       "Ollama 服务地址",
		Description: "Ollama 服务地址",
		Type:        schema.TypeURL,
		Default:     DefaultOllamaURL,
	})

	// 抽帧间隔必须大于 0（0 会让抽帧失去意义），没有上限，故 Max 留空。
	schema.Register(schema.Field{
		Key:          KeyFrameInterval,
		Group:        "ai",
		Label:        "抽帧间隔",
		Description:  "视频抽帧间隔（秒）",
		Type:         schema.TypeFloat,
		Default:      DefaultFrameInterval,
		Min:          schema.Ptr(0.0),
		MinExclusive: true,
		Unit:         "秒",
	})

	// 上限取自 MaxDiffThreshold：dHash 是 64 位，阈值超过 64 会丢弃除首帧外的所有帧。
	schema.Register(schema.Field{
		Key:         KeyDiffThreshold,
		Group:       "ai",
		Label:       "近似帧过滤阈值",
		Description: "近似帧过滤阈值（与上一保留帧的 dHash 汉明距离小于该值的帧被丢弃）",
		Type:        schema.TypeInt,
		Default:     DefaultDiffThreshold,
		Min:         schema.Ptr(1.0),
		Max:         schema.Ptr(float64(MaxDiffThreshold)),
	})

	// 上限取自 MaxFramesLimit：抽出的每一帧都会读回内存，无上限会被一次笔误打爆。
	schema.Register(schema.Field{
		Key:         KeyMaxFrames,
		Group:       "ai",
		Label:       "单视频最大帧数",
		Description: "单个视频最多抽取的帧数",
		Type:        schema.TypeInt,
		Default:     DefaultMaxFrames,
		Min:         schema.Ptr(1.0),
		Max:         schema.Ptr(float64(MaxFramesLimit)),
	})
}
