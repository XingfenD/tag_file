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
	set("ai_frame_interval", "0.5")
	seedRaw(t, "ai_frame_diff_threshold", "not-a-number")
	set("ollama_model", "qwen2.5vl:7b")

	got := ai.NewService().Settings()

	if got.MaxFrames != 25 {
		t.Fatalf("MaxFrames = %d, want 25", got.MaxFrames)
	}
	if got.FrameInterval != 0.5 {
		t.Fatalf("FrameInterval = %v, want 0.5", got.FrameInterval)
	}
	if got.DiffThreshold != 10 {
		t.Fatalf("DiffThreshold = %d, want fallback 10 on unparseable value", got.DiffThreshold)
	}
	if got.OllamaModel != "qwen2.5vl:7b" {
		t.Fatalf("OllamaModel = %q, want qwen2.5vl:7b", got.OllamaModel)
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
