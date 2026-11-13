package ai_test

import (
	"testing"

	"github.com/XingfenD/tag_file/internal/service/ai"
	configSvc "github.com/XingfenD/tag_file/internal/service/config"
)

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
	set("ai_frame_diff_threshold", "not-a-number")
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

	cfgSvc := configSvc.NewService()
	set := func(key, value string) {
		t.Helper()
		if err := cfgSvc.Set(key, value); err != nil {
			t.Fatalf("set %s: %v", key, err)
		}
	}
	set("ai_max_frames", "0")
	set("ai_frame_interval", "-1")
	set("ai_frame_diff_threshold", "0")

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
