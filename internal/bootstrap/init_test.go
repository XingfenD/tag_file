package bootstrap_test

import (
	"testing"

	"github.com/XingfenD/tag_file/internal/bootstrap"
	"github.com/XingfenD/tag_file/internal/model"
	"github.com/XingfenD/tag_file/pkg/storage"
)

// Key names and values are literals on purpose: this test pins what a fresh
// database actually holds, so changing a default on one side only (the seed
// table or the reader's fallback) fails here instead of drifting silently.
const (
	keyScanExclude             = "scan_exclude"
	keyOllamaURL               = "ollama_url"
	keyOllamaModel             = "ollama_model"
	keyFrameInterval           = "ai_frame_interval"
	keyDiffThreshold           = "ai_frame_diff_threshold"
	keyMaxFrames               = "ai_max_frames"
	defaultScanExcludeValue    = `["^\\$RECYCLE\\.BIN$","^System Volume Information$","^\\."]`
	defaultOllamaURLValue      = "http://127.0.0.1:11434"
	defaultOllamaModelValue    = ""
	defaultFrameIntervalValue  = "1"
	defaultDiffThresholdValue  = "10"
	defaultMaxFramesValue      = "10"
	customizedScanExcludeValue = `["^custom$"]`
)

// useTempWorkspace points the process at a throwaway directory so InitDB writes
// .tagfile/tagfile.db there instead of into the package directory.
func useTempWorkspace(t *testing.T) {
	t.Helper()
	t.Chdir(t.TempDir())
}

// configRows returns every seeded config row keyed by name.
func configRows(t *testing.T) map[string]model.Config {
	t.Helper()
	var rows []model.Config
	if err := storage.GetDB().Find(&rows).Error; err != nil {
		t.Fatalf("load config rows: %v", err)
	}
	got := make(map[string]model.Config, len(rows))
	for _, row := range rows {
		if _, dup := got[row.ConfigName]; dup {
			t.Errorf("config %s was seeded more than once", row.ConfigName)
		}
		got[row.ConfigName] = row
	}
	return got
}

func requireRows(t *testing.T, rows map[string]model.Config, want map[string]string) {
	t.Helper()
	for name, wantValue := range want {
		row, ok := rows[name]
		if !ok {
			t.Errorf("config %s is missing from a fresh database", name)
			continue
		}
		if row.ConfigValue != wantValue {
			t.Errorf("config %s = %q, want %q", name, row.ConfigValue, wantValue)
		}
		if row.Description == "" {
			t.Errorf("config %s has an empty description", name)
		}
	}
}

func TestSeedDefaultConfig_FreshDatabaseGetsEveryKey(t *testing.T) {
	useTempWorkspace(t)

	if err := bootstrap.InitDB(); err != nil {
		t.Fatalf("InitDB: %v", err)
	}

	rows := configRows(t)
	if len(rows) != 6 {
		t.Fatalf("seeded %d config rows, want 6: %v", len(rows), rows)
	}
	requireRows(t, rows, map[string]string{
		keyScanExclude:   defaultScanExcludeValue,
		keyOllamaURL:     defaultOllamaURLValue,
		keyOllamaModel:   defaultOllamaModelValue,
		keyFrameInterval: defaultFrameIntervalValue,
		keyDiffThreshold: defaultDiffThresholdValue,
		keyMaxFrames:     defaultMaxFramesValue,
	})
}

// An existing database that already has scan_exclude (and no AI keys) must get
// the new keys without losing the user's value: the early return on the first
// key would skip every later one forever.
func TestSeedDefaultConfig_AddsNewKeysAndKeepsCustomizedValue(t *testing.T) {
	useTempWorkspace(t)

	if err := bootstrap.InitDB(); err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	db := storage.GetDB()
	if err := db.Model(&model.Config{}).Where("config_name = ?", keyScanExclude).
		Update("config_value", customizedScanExcludeValue).Error; err != nil {
		t.Fatalf("customize scan_exclude: %v", err)
	}
	if err := db.Where("config_name <> ?", keyScanExclude).Delete(&model.Config{}).Error; err != nil {
		t.Fatalf("drop AI keys: %v", err)
	}

	if err := bootstrap.InitDB(); err != nil {
		t.Fatalf("InitDB after upgrade: %v", err)
	}

	rows := configRows(t)
	if len(rows) != 6 {
		t.Fatalf("seeded %d config rows, want 6: %v", len(rows), rows)
	}
	requireRows(t, rows, map[string]string{
		keyScanExclude:   customizedScanExcludeValue,
		keyOllamaURL:     defaultOllamaURLValue,
		keyOllamaModel:   defaultOllamaModelValue,
		keyFrameInterval: defaultFrameIntervalValue,
		keyDiffThreshold: defaultDiffThresholdValue,
		keyMaxFrames:     defaultMaxFramesValue,
	})
}

func TestSeedDefaultConfig_RerunSeedsNothingNew(t *testing.T) {
	useTempWorkspace(t)

	for i := 0; i < 2; i++ {
		if err := bootstrap.InitDB(); err != nil {
			t.Fatalf("InitDB run %d: %v", i+1, err)
		}
	}

	rows := configRows(t)
	if len(rows) != 6 {
		t.Fatalf("after two runs %d config rows, want 6: %v", len(rows), rows)
	}
	var count int64
	for _, name := range []string{
		keyScanExclude, keyOllamaURL, keyOllamaModel,
		keyFrameInterval, keyDiffThreshold, keyMaxFrames,
	} {
		if err := storage.GetDB().Model(&model.Config{}).
			Where("config_name = ?", name).Count(&count).Error; err != nil {
			t.Fatalf("count %s: %v", name, err)
		}
		if count != 1 {
			t.Errorf("config %s has %d rows, want exactly 1", name, count)
		}
	}
}
