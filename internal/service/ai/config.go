package ai

import (
	"fmt"
	"log"
	"math"
	"strconv"
)

// Config keys of the vision tagging feature and of its optional second-stage
// refine endpoint (ai_refine_*).
const (
	KeyProvider = "ai_provider"

	KeyOllamaURL       = "ollama_url"
	KeyOllamaModel     = "ollama_model"
	KeyOpenAIURL       = "ai_openai_url"
	KeyOpenAIModel     = "ai_openai_model"
	KeyOpenAIAPIKey    = "ai_openai_api_key"
	KeyAnthropicURL    = "ai_anthropic_url"
	KeyAnthropicModel  = "ai_anthropic_model"
	KeyAnthropicAPIKey = "ai_anthropic_api_key"

	KeyRefineProvider = "ai_refine_provider"

	KeyRefineOllamaURL       = "ai_refine_ollama_url"
	KeyRefineOllamaModel     = "ai_refine_ollama_model"
	KeyRefineOpenAIURL       = "ai_refine_openai_url"
	KeyRefineOpenAIModel     = "ai_refine_openai_model"
	KeyRefineOpenAIAPIKey    = "ai_refine_openai_api_key"
	KeyRefineAnthropicURL    = "ai_refine_anthropic_url"
	KeyRefineAnthropicModel  = "ai_refine_anthropic_model"
	KeyRefineAnthropicAPIKey = "ai_refine_anthropic_api_key"

	KeyFrameInterval = "ai_frame_interval"
	KeyDiffThreshold = "ai_frame_diff_threshold"
	KeyMaxFrames     = "ai_max_frames"
)

// Providers of the ai_provider enum. The set is closed: anything else read from
// the config table takes the logged fallback to DefaultProvider.
const (
	ProviderOllama    = "ollama"
	ProviderOpenAI    = "openai"
	ProviderAnthropic = "anthropic"
)

// RefineOff is the extra member of the ai_refine_provider enum that disables
// stage two; the endpoint values themselves are the shared Provider* constants
// above, so both enums can never drift apart.
const RefineOff = "off"

// Default values as stored in the config table. bootstrap seeds a fresh
// database from these and Settings falls back to them, so the two can never
// disagree; an empty model of the active provider means "not configured" and
// the recognition path reports it.
const (
	DefaultProvider = ProviderOllama

	// The refine group shares the vision group's URL and empty model/key
	// defaults below — the two groups can never disagree by construction. Its
	// enum lands on off, the state that keeps the pipeline single-stage.
	DefaultRefineProvider = RefineOff

	DefaultOllamaURL       = "http://127.0.0.1:11434"
	DefaultOllamaModel     = ""
	DefaultOpenAIURL       = "https://api.openai.com/v1"
	DefaultOpenAIModel     = ""
	DefaultOpenAIAPIKey    = ""
	DefaultAnthropicURL    = "https://api.anthropic.com"
	DefaultAnthropicModel  = ""
	DefaultAnthropicAPIKey = ""

	DefaultFrameInterval = "10" // 定点数 wire：整数 10 = 1.0 秒（Shift=1）
	DefaultDiffThreshold = "10"
	DefaultMaxFrames     = "10"
)

// FrameIntervalShift is the fixed-point decimal shift for ai_frame_interval.
// The stored wire is an integer n meaning n / 10^FrameIntervalShift seconds, so
// Shift=1 turns the legacy seconds value "1" into the wire "10" (still 1.0s).
// It lives here next to the key and default it scales: the declaration in
// config_schema.go and the data migration in bootstrap both reference it.
const FrameIntervalShift = 1

// Bounds of the numeric keys, both enforced through intSetting's logged
// fallback and exported so the WebUI's config schema validates writes against
// the exact same limits the runtime reads with.
//
// MaxFramesLimit caps ai_max_frames because Extract hands ffmpeg
// -frames:v maxFrames*10 and then reads every JPEG it writes back into process
// memory: an unbounded setting one typo away (1000 -> ~10,000 frames) would
// OOM-kill the WebUI. 200 sits far above the default of 10, so no legitimate
// workflow reaches it.
//
// MaxDiffThreshold caps ai_frame_diff_threshold at the width of the hash: dHash
// is 64 bits (diff.go), so a Hamming distance above 64 can never occur and a
// threshold greater than 64 keeps the first frame and drops every other one —
// an unusable value that would silently tag a whole video from one still, with
// nothing in the review queue to notice it.
const (
	MaxFramesLimit   = 200
	MaxDiffThreshold = 64
)

// Typed forms of the string defaults above, derived rather than restated.
var (
	defaultFrameInterval = float64(mustParseInt(DefaultFrameInterval)) / math.Pow(10, float64(FrameIntervalShift))
	defaultDiffThreshold = mustParseInt(DefaultDiffThreshold)
	defaultMaxFrames     = mustParseInt(DefaultMaxFrames)
)

func mustParseInt(raw string) int {
	value, err := strconv.Atoi(raw)
	if err != nil {
		panic(fmt.Sprintf("ai: default config value %q is not an integer", raw))
	}
	return value
}

// Settings is the user-facing configuration snapshot of one recognition run.
type Settings struct {
	Provider string

	OllamaURL   string
	OllamaModel string

	OpenAIURL    string
	OpenAIModel  string
	OpenAIAPIKey string

	AnthropicURL    string
	AnthropicModel  string
	AnthropicAPIKey string

	RefineProvider string

	RefineOllamaURL   string
	RefineOllamaModel string

	RefineOpenAIURL    string
	RefineOpenAIModel  string
	RefineOpenAIAPIKey string

	RefineAnthropicURL    string
	RefineAnthropicModel  string
	RefineAnthropicAPIKey string

	FrameInterval float64
	DiffThreshold int
	MaxFrames     int
}

// Settings reads every key independently and never fails: a missing key, a read
// error or an unparseable/non-positive/out-of-range number falls back to its
// default. A zero diff threshold falls back too, since 0 would keep every frame
// and defeat the dedupe filter, and one above 64 for the mirror-image reason.
func (s *Service) Settings() Settings {
	return Settings{
		Provider: s.providerSetting(),

		OllamaURL:   s.stringSetting(KeyOllamaURL, DefaultOllamaURL),
		OllamaModel: s.stringSetting(KeyOllamaModel, DefaultOllamaModel),

		OpenAIURL:    s.stringSetting(KeyOpenAIURL, DefaultOpenAIURL),
		OpenAIModel:  s.stringSetting(KeyOpenAIModel, DefaultOpenAIModel),
		OpenAIAPIKey: s.stringSetting(KeyOpenAIAPIKey, DefaultOpenAIAPIKey),

		AnthropicURL:    s.stringSetting(KeyAnthropicURL, DefaultAnthropicURL),
		AnthropicModel:  s.stringSetting(KeyAnthropicModel, DefaultAnthropicModel),
		AnthropicAPIKey: s.stringSetting(KeyAnthropicAPIKey, DefaultAnthropicAPIKey),

		RefineProvider: s.refineProviderSetting(),

		RefineOllamaURL:   s.stringSetting(KeyRefineOllamaURL, DefaultOllamaURL),
		RefineOllamaModel: s.stringSetting(KeyRefineOllamaModel, DefaultOllamaModel),

		RefineOpenAIURL:    s.stringSetting(KeyRefineOpenAIURL, DefaultOpenAIURL),
		RefineOpenAIModel:  s.stringSetting(KeyRefineOpenAIModel, DefaultOpenAIModel),
		RefineOpenAIAPIKey: s.stringSetting(KeyRefineOpenAIAPIKey, DefaultOpenAIAPIKey),

		RefineAnthropicURL:    s.stringSetting(KeyRefineAnthropicURL, DefaultAnthropicURL),
		RefineAnthropicModel:  s.stringSetting(KeyRefineAnthropicModel, DefaultAnthropicModel),
		RefineAnthropicAPIKey: s.stringSetting(KeyRefineAnthropicAPIKey, DefaultAnthropicAPIKey),

		FrameInterval: s.shiftedIntSetting(KeyFrameInterval, defaultFrameInterval, FrameIntervalShift),
		DiffThreshold: s.intSetting(KeyDiffThreshold, defaultDiffThreshold, MaxDiffThreshold),
		MaxFrames:     s.intSetting(KeyMaxFrames, defaultMaxFrames, MaxFramesLimit),
	}
}

// providerSetting 读 ai_provider：封闭枚举，未知值（含空串、大小写漂移）
// 与 intSetting 的非法数同形——记一行日志并回落 ollama，绝不把非法值
// 传给分派开关。
func (s *Service) providerSetting() string {
	raw := s.stringSetting(KeyProvider, DefaultProvider)
	switch raw {
	case ProviderOllama, ProviderOpenAI, ProviderAnthropic:
		return raw
	}
	log.Printf("[ai] config %s = %q, fallback to %s", KeyProvider, raw, DefaultProvider)
	return DefaultProvider
}

// refineProviderSetting 读 ai_refine_provider：封闭枚举 = Provider* 三值加
// off，非法值与 providerSetting 同形——记一行日志并回落 off（默认即单段，
// 非法值绝不启用整理）。
func (s *Service) refineProviderSetting() string {
	raw := s.stringSetting(KeyRefineProvider, DefaultRefineProvider)
	switch raw {
	case RefineOff, ProviderOllama, ProviderOpenAI, ProviderAnthropic:
		return raw
	}
	log.Printf("[ai] config %s = %q, fallback to %s", KeyRefineProvider, raw, DefaultRefineProvider)
	return DefaultRefineProvider
}

// refineEnabled 是阶段二的总开关。读侧保证 RefineProvider 恒为封闭枚举成员，
// 所以非 off 即某个端点。
func refineEnabled(settings Settings) bool {
	return settings.RefineProvider != RefineOff
}

// RefineEnabled 是 refineEnabled 的导出包装，供 WebUI 列表把「重新整理」按钮
// 的门算在服务器侧：原文可见性不归它管，能不能重整理才归它管。
func RefineEnabled(settings Settings) bool {
	return refineEnabled(settings)
}

// checkRefineModel 镜像 checkProviderModel，管整理端点：off 无模型可要求；
// 已选端点的模型名为空是配置错误，任务必须报错而不是静默回落单段。
func checkRefineModel(settings Settings) error {
	if settings.RefineProvider == RefineOff {
		return nil
	}
	var model string
	switch settings.RefineProvider {
	case ProviderOpenAI:
		model = settings.RefineOpenAIModel
	case ProviderAnthropic:
		model = settings.RefineAnthropicModel
	default:
		model = settings.RefineOllamaModel
	}
	if model == "" {
		return fmt.Errorf("未配置整理端点的 %s 模型名, 请先在配置页填写", providerLabel(settings.RefineProvider))
	}
	return nil
}

func (s *Service) stringSetting(key string, def string) string {
	cfg, err := s.configSvc.Get(key)
	if err != nil {
		return def
	}
	return cfg.ConfigValue
}

// intSetting reads a positive int that must not exceed upperBound. Anything
// else - missing key, read error, unparseable text, non-positive or out-of-range
// number - is unusable and takes the same logged fallback to def, so no caller
// has to remember which out-of-range direction clamps and which falls back.
func (s *Service) intSetting(key string, def int, upperBound int) int {
	cfg, err := s.configSvc.Get(key)
	if err != nil {
		return def
	}
	value, err := strconv.Atoi(cfg.ConfigValue)
	if err != nil || value <= 0 || value > upperBound {
		log.Printf("[ai] config %s = %q, fallback to %d", key, cfg.ConfigValue, def)
		return def
	}
	return value
}

// shiftedIntSetting reads a fixed-point integer wire and returns its display
// value (n / 10^shift). A missing key, a read error, an unparseable wire
// (including a legacy decimal like "0.5") or a non-positive value is unusable
// and takes the same logged fallback as intSetting: no caller has to remember
// which direction clamps and which falls back.
func (s *Service) shiftedIntSetting(key string, def float64, shift int) float64 {
	cfg, err := s.configSvc.Get(key)
	if err != nil {
		return def
	}
	value, err := strconv.ParseInt(cfg.ConfigValue, 10, 64)
	if err != nil || value <= 0 {
		log.Printf("[ai] config %s = %q, fallback to %v", key, cfg.ConfigValue, def)
		return def
	}
	return float64(value) / math.Pow(10, float64(shift))
}
