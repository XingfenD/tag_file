package ai_test

import (
	"testing"

	"github.com/XingfenD/tag_file/internal/model"
	"github.com/XingfenD/tag_file/internal/service/ai"
	configSvc "github.com/XingfenD/tag_file/internal/service/config"
	"github.com/XingfenD/tag_file/pkg/storage"
	"gorm.io/gorm/clause"
)

// seedRaw writes a value straight into the config table, bypassing the
// validating service. The read side must tolerate garbage that predates or
// escapes validation, so the fallback tests need a way to plant it.
func seedRaw(t *testing.T, key, value string) {
	t.Helper()
	if err := storage.GetDB().Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "config_name"}},
		DoUpdates: clause.AssignmentColumns([]string{"config_value"}),
	}).Create(&model.Config{ConfigName: key, ConfigValue: value}).Error; err != nil {
		t.Fatalf("seed %s: %v", key, err)
	}
}

func TestAISettings_DefaultsWhenUnset(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	got := ai.NewService().Settings()

	if got.OllamaURL != "http://127.0.0.1:11434" {
		t.Fatalf("OllamaURL = %q, want default http://127.0.0.1:11434", got.OllamaURL)
	}
	if got.OllamaModel != "" {
		t.Fatalf("OllamaModel = %q, want empty (means not configured)", got.OllamaModel)
	}
	if got.FrameInterval != 1 {
		t.Fatalf("FrameInterval = %v, want default 1", got.FrameInterval)
	}
	if got.DiffThreshold != 10 {
		t.Fatalf("DiffThreshold = %d, want default 10", got.DiffThreshold)
	}
	if got.MaxFrames != 10 {
		t.Fatalf("MaxFrames = %d, want default 10", got.MaxFrames)
	}
	if got.Provider != ai.ProviderOllama {
		t.Fatalf("Provider = %q, want default ollama", got.Provider)
	}
	if got.OpenAIURL != "https://api.openai.com/v1" {
		t.Fatalf("OpenAIURL = %q, want default https://api.openai.com/v1", got.OpenAIURL)
	}
	if got.AnthropicURL != "https://api.anthropic.com" {
		t.Fatalf("AnthropicURL = %q, want default https://api.anthropic.com", got.AnthropicURL)
	}
	if got.OpenAIModel != "" || got.AnthropicModel != "" || got.OpenAIAPIKey != "" || got.AnthropicAPIKey != "" {
		t.Fatalf("未配置的模型/密钥应为空串, got %+v", got)
	}
}

// 读侧对非法 ai_provider 的容忍与 intSetting 同形：回落 ollama，绝不 panic。
// "Ollama"/"OPENAI" 大小写不匹配也算非法——枚举值是封闭集合，不做归一化。
func TestAISettings_ProviderFallsBackOnUnknown(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	for _, raw := range []string{"bogus", "", "Ollama", "OPENAI"} {
		seedRaw(t, ai.KeyProvider, raw)
		if got := ai.NewService().Settings().Provider; got != ai.ProviderOllama {
			t.Fatalf("Provider for wire %q = %q, want fallback ollama", raw, got)
		}
	}

	// 合法值原样读出；经 Set 写入证明 oneof 注册表接线成立。
	cfg := configSvc.NewService()
	for _, p := range []string{ai.ProviderOpenAI, ai.ProviderAnthropic, ai.ProviderOllama} {
		if err := cfg.Set(ai.KeyProvider, p); err != nil {
			t.Fatalf("set ai_provider=%s: %v", p, err)
		}
		if got := ai.NewService().Settings().Provider; got != p {
			t.Fatalf("Provider = %q, want %q", got, p)
		}
	}
}

func TestAISettings_ReadsProviderCredentials(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	cfg := configSvc.NewService()
	set := func(key, value string) {
		t.Helper()
		if err := cfg.Set(key, value); err != nil {
			t.Fatalf("set %s: %v", key, err)
		}
	}
	set(ai.KeyProvider, ai.ProviderOpenAI)
	set(ai.KeyOpenAIURL, "http://127.0.0.1:8000/v1")
	set(ai.KeyOpenAIModel, "qwen2.5-vl")
	set(ai.KeyOpenAIAPIKey, "sk-local")

	got := ai.NewService().Settings()
	if got.Provider != ai.ProviderOpenAI || got.OpenAIURL != "http://127.0.0.1:8000/v1" ||
		got.OpenAIModel != "qwen2.5-vl" || got.OpenAIAPIKey != "sk-local" {
		t.Fatalf("Settings 未读出 openai 凭据: %+v", got)
	}
}

func TestAISettings_ReadsConfigAndFallsBackOnGarbage(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	cfgSvc := configSvc.NewService()
	set := func(key, value string) {
		t.Helper()
		if err := cfgSvc.Set(key, value); err != nil {
			t.Fatalf("set %s: %v", key, err)
		}
	}
	set("ai_max_frames", "25")
	// 定点数 wire：整数 5 = 0.5 秒（Shift=1）。读侧只认整数，小数只存在于
	// 显示层；旧存量小数 "0.5" 已由 Task 5 迁移转换为 "5"。
	seedRaw(t, "ai_frame_interval", "5")
	seedRaw(t, "ai_frame_diff_threshold", "not-a-number")
	set("ollama_model", "qwen2.5vl:7b")

	got := ai.NewService().Settings()

	if got.MaxFrames != 25 {
		t.Fatalf("MaxFrames = %d, want 25", got.MaxFrames)
	}
	if got.FrameInterval != 0.5 {
		t.Fatalf("FrameInterval = %v, want 0.5 (wire \"5\" / 10^1)", got.FrameInterval)
	}
	if got.DiffThreshold != 10 {
		t.Fatalf("DiffThreshold = %d, want fallback 10 on unparseable value", got.DiffThreshold)
	}
	if got.OllamaModel != "qwen2.5vl:7b" {
		t.Fatalf("OllamaModel = %q, want qwen2.5vl:7b", got.OllamaModel)
	}
}

// 迁移前的旧存量小数 wire（如 "0.5"）在定点数读侧不是整数，必须当成解析
// 失败，回落默认值 1.0——绝不 panic、也绝不读出 0.5 或 5.0；"0" 非正数同样
// 回落（0 会让抽帧失去意义）。
func TestAISettings_LegacyDecimalWireFallsBackToDefault(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	for _, raw := range []string{"0.5", "abc", "0"} {
		seedRaw(t, "ai_frame_interval", raw)
		if got := ai.NewService().Settings().FrameInterval; got != 1 {
			t.Fatalf("FrameInterval for wire %q = %v, want fallback 1", raw, got)
		}
	}
}

func TestAISettings_RejectsNonPositiveNumbers(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	seedRaw(t, "ai_max_frames", "0")
	seedRaw(t, "ai_frame_interval", "-1")
	seedRaw(t, "ai_frame_diff_threshold", "0")

	got := ai.NewService().Settings()

	if got.MaxFrames != 10 {
		t.Fatalf("MaxFrames = %d, want fallback 10 on non-positive value", got.MaxFrames)
	}
	if got.FrameInterval != 1 {
		t.Fatalf("FrameInterval = %v, want fallback 1 on non-positive value", got.FrameInterval)
	}
	// 0 would keep every frame and defeat the dedupe filter, so it falls back too.
	if got.DiffThreshold != 10 {
		t.Fatalf("DiffThreshold = %d, want fallback 10 on zero threshold", got.DiffThreshold)
	}
}

// ai_max_frames is read into ffmpeg's -frames:v (multiplied by 10) and every
// extracted JPEG is held in process memory, so an unbounded value is a typo away
// from OOM-killing the WebUI. Values above maxFramesLimit fall back to the
// default exactly like any other unusable value.
func TestAISettings_RejectsMaxFramesAboveBound(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	cfgSvc := configSvc.NewService()
	set := func(key, value string) {
		t.Helper()
		if err := cfgSvc.Set(key, value); err != nil {
			t.Fatalf("set %s: %v", key, err)
		}
	}

	for _, raw := range []string{"201", "100000"} {
		seedRaw(t, "ai_max_frames", raw)
		if got := ai.NewService().Settings(); got.MaxFrames != 10 {
			t.Fatalf("MaxFrames = %d for %q, want fallback 10 above the bound", got.MaxFrames, raw)
		}
	}

	// The bound is inclusive.
	set("ai_max_frames", "200")

	if got := ai.NewService().Settings(); got.MaxFrames != 200 {
		t.Fatalf("MaxFrames = %d, want 200 (the bound is inclusive)", got.MaxFrames)
	}
}

// ai_frame_diff_threshold is bounded by the hash, not by memory: dHash is 64 bits
// (diff.go), so no Hamming distance above 64 exists and a larger threshold drops
// every frame but the first — a whole video silently tagged from one still. That
// is as unusable as a negative number, so it takes the same logged fallback.
func TestAISettings_RejectsDiffThresholdAboveBound(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	cfgSvc := configSvc.NewService()
	set := func(key, value string) {
		t.Helper()
		if err := cfgSvc.Set(key, value); err != nil {
			t.Fatalf("set %s: %v", key, err)
		}
	}

	for _, raw := range []string{"65", "1000"} {
		seedRaw(t, "ai_frame_diff_threshold", raw)
		if got := ai.NewService().Settings(); got.DiffThreshold != 10 {
			t.Fatalf("DiffThreshold = %d for %q, want fallback 10 above the 64-bit bound", got.DiffThreshold, raw)
		}
	}

	// The bound is inclusive: 64 is the largest distance two hashes can differ by.
	set("ai_frame_diff_threshold", "64")

	if got := ai.NewService().Settings(); got.DiffThreshold != 64 {
		t.Fatalf("DiffThreshold = %d, want 64 (the bound is inclusive)", got.DiffThreshold)
	}
}
