package ai

import (
	"fmt"
	"log"
	"math"
	"strconv"
)

// Config keys of the vision tagging feature.
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

// Default values as stored in the config table. bootstrap seeds a fresh
// database from these and Settings falls back to them, so the two can never
// disagree; an empty model of the active provider means "not configured" and
// the recognition path reports it.
const (
	DefaultProvider = ProviderOllama

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
