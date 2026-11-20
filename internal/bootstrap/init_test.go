package bootstrap_test

import (
	"os"
	"strings"
	"testing"

	"github.com/XingfenD/tag_file/internal/bootstrap"
	"github.com/XingfenD/tag_file/internal/model"
	"github.com/XingfenD/tag_file/internal/service/config/schema"
	"github.com/XingfenD/tag_file/pkg/storage"
)

// Key names and values are literals on purpose: this test pins what a fresh
// database actually holds, so changing a default on one side only (the seed
// table or the reader's fallback) fails here instead of drifting silently.
const (
	keyScanExclude                    = "scan_exclude"
	keyOllamaURL                      = "ollama_url"
	keyOllamaModel                    = "ollama_model"
	keyProvider                       = "ai_provider"
	keyOpenAIURL                      = "ai_openai_url"
	keyOpenAIModel                    = "ai_openai_model"
	keyOpenAIAPIKey                   = "ai_openai_api_key"
	keyAnthropicURL                   = "ai_anthropic_url"
	keyAnthropicModel                 = "ai_anthropic_model"
	keyAnthropicAPIKey                = "ai_anthropic_api_key"
	keyRefineProvider                 = "ai_refine_provider"
	keyRefineOllamaURL                = "ai_refine_ollama_url"
	keyRefineOllamaModel              = "ai_refine_ollama_model"
	keyRefineOpenAIURL                = "ai_refine_openai_url"
	keyRefineOpenAIModel              = "ai_refine_openai_model"
	keyRefineOpenAIAPIKey             = "ai_refine_openai_api_key"
	keyRefineAnthropicURL             = "ai_refine_anthropic_url"
	keyRefineAnthropicModel           = "ai_refine_anthropic_model"
	keyRefineAnthropicAPIKey          = "ai_refine_anthropic_api_key"
	keyFrameInterval                  = "ai_frame_interval"
	keyDiffThreshold                  = "ai_frame_diff_threshold"
	keyMaxFrames                      = "ai_max_frames"
	defaultScanExcludeValue           = `["^\\$RECYCLE\\.BIN$","^System Volume Information$","^\\."]`
	defaultOllamaURLValue             = "http://127.0.0.1:11434"
	defaultOllamaModelValue           = ""
	defaultProviderValue              = "ollama"
	defaultOpenAIURLValue             = "https://api.openai.com/v1"
	defaultOpenAIModelValue           = ""
	defaultOpenAIAPIKeyValue          = ""
	defaultAnthropicURLValue          = "https://api.anthropic.com"
	defaultAnthropicModelValue        = ""
	defaultAnthropicAPIKeyValue       = ""
	defaultRefineProviderValue        = "off"
	defaultRefineOllamaURLValue       = "http://127.0.0.1:11434"
	defaultRefineOllamaModelValue     = ""
	defaultRefineOpenAIURLValue       = "https://api.openai.com/v1"
	defaultRefineOpenAIModelValue     = ""
	defaultRefineOpenAIAPIKeyValue    = ""
	defaultRefineAnthropicURLValue    = "https://api.anthropic.com"
	defaultRefineAnthropicModelValue  = ""
	defaultRefineAnthropicAPIKeyValue = ""
	defaultFrameIntervalValue         = "10"
	defaultDiffThresholdValue         = "10"
	defaultMaxFramesValue             = "10"
	customizedScanExcludeValue        = `["^custom$"]`
)

// migrationFrameIntervalFixedPoint is the ledger ID recorded when the
// ai_frame_interval fixed-point data migration has been applied. The ledger
// lives at .tagfile/config_migrations, next to core_version.
const migrationFrameIntervalFixedPoint = "ai_frame_interval_fixedpoint_v1"

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
	}
}

// prepareConfigTable opens the database and migrates only the config table, so a
// migration test can plant a pre-upgrade row before bootstrap.InitDB runs the
// full AutoMigrate + migrate + seed sequence.
func prepareConfigTable(t *testing.T) {
	t.Helper()
	if err := storage.InitDB(".tagfile/tagfile.db"); err != nil {
		t.Fatalf("init storage: %v", err)
	}
	if err := storage.GetDB().AutoMigrate(&model.Config{}); err != nil {
		t.Fatalf("migrate config table: %v", err)
	}
}

// writeConfigRow plants a raw config row, bypassing seed and validation.
func writeConfigRow(t *testing.T, name, value string) {
	t.Helper()
	if err := storage.GetDB().Create(&model.Config{
		ConfigName: name, ConfigValue: value,
	}).Error; err != nil {
		t.Fatalf("write config %s: %v", name, err)
	}
}

// readConfigValue returns the stored wire value of a config row.
func readConfigValue(t *testing.T, name string) string {
	t.Helper()
	var cfg model.Config
	if err := storage.GetDB().Where("config_name = ?", name).First(&cfg).Error; err != nil {
		t.Fatalf("read config %s: %v", name, err)
	}
	return cfg.ConfigValue
}

// ledgerContains reports whether the migration ledger holds id. A missing ledger
// is treated as holding nothing, which is exactly what migrateConfigStorage does.
func ledgerContains(t *testing.T, id string) bool {
	t.Helper()
	data, err := os.ReadFile(".tagfile/config_migrations")
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(line) == id {
			return true
		}
	}
	return false
}

func TestSeedDefaultConfig_FreshDatabaseGetsEveryKey(t *testing.T) {
	useTempWorkspace(t)

	if err := bootstrap.InitDB(); err != nil {
		t.Fatalf("InitDB: %v", err)
	}

	rows := configRows(t)
	if len(rows) != 22 {
		t.Fatalf("seeded %d config rows, want 22: %v", len(rows), rows)
	}
	requireRows(t, rows, map[string]string{
		keyScanExclude:           defaultScanExcludeValue,
		keyProvider:              defaultProviderValue,
		keyOllamaURL:             defaultOllamaURLValue,
		keyOllamaModel:           defaultOllamaModelValue,
		keyOpenAIURL:             defaultOpenAIURLValue,
		keyOpenAIModel:           defaultOpenAIModelValue,
		keyOpenAIAPIKey:          defaultOpenAIAPIKeyValue,
		keyAnthropicURL:          defaultAnthropicURLValue,
		keyAnthropicModel:        defaultAnthropicModelValue,
		keyAnthropicAPIKey:       defaultAnthropicAPIKeyValue,
		keyRefineProvider:        defaultRefineProviderValue,
		keyRefineOllamaURL:       defaultRefineOllamaURLValue,
		keyRefineOllamaModel:     defaultRefineOllamaModelValue,
		keyRefineOpenAIURL:       defaultRefineOpenAIURLValue,
		keyRefineOpenAIModel:     defaultRefineOpenAIModelValue,
		keyRefineOpenAIAPIKey:    defaultRefineOpenAIAPIKeyValue,
		keyRefineAnthropicURL:    defaultRefineAnthropicURLValue,
		keyRefineAnthropicModel:  defaultRefineAnthropicModelValue,
		keyRefineAnthropicAPIKey: defaultRefineAnthropicAPIKeyValue,
		keyFrameInterval:         defaultFrameIntervalValue,
		keyDiffThreshold:         defaultDiffThresholdValue,
		keyMaxFrames:             defaultMaxFramesValue,
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
	if len(rows) != 22 {
		t.Fatalf("seeded %d config rows, want 22: %v", len(rows), rows)
	}
	requireRows(t, rows, map[string]string{
		keyScanExclude:           customizedScanExcludeValue,
		keyProvider:              defaultProviderValue,
		keyOllamaURL:             defaultOllamaURLValue,
		keyOllamaModel:           defaultOllamaModelValue,
		keyOpenAIURL:             defaultOpenAIURLValue,
		keyOpenAIModel:           defaultOpenAIModelValue,
		keyOpenAIAPIKey:          defaultOpenAIAPIKeyValue,
		keyAnthropicURL:          defaultAnthropicURLValue,
		keyAnthropicModel:        defaultAnthropicModelValue,
		keyAnthropicAPIKey:       defaultAnthropicAPIKeyValue,
		keyRefineProvider:        defaultRefineProviderValue,
		keyRefineOllamaURL:       defaultRefineOllamaURLValue,
		keyRefineOllamaModel:     defaultRefineOllamaModelValue,
		keyRefineOpenAIURL:       defaultRefineOpenAIURLValue,
		keyRefineOpenAIModel:     defaultRefineOpenAIModelValue,
		keyRefineOpenAIAPIKey:    defaultRefineOpenAIAPIKeyValue,
		keyRefineAnthropicURL:    defaultRefineAnthropicURLValue,
		keyRefineAnthropicModel:  defaultRefineAnthropicModelValue,
		keyRefineAnthropicAPIKey: defaultRefineAnthropicAPIKeyValue,
		keyFrameInterval:         defaultFrameIntervalValue,
		keyDiffThreshold:         defaultDiffThresholdValue,
		keyMaxFrames:             defaultMaxFramesValue,
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
	if len(rows) != 22 {
		t.Fatalf("after two runs %d config rows, want 22: %v", len(rows), rows)
	}
	var count int64
	for _, name := range []string{
		keyScanExclude, keyProvider,
		keyOllamaURL, keyOllamaModel,
		keyOpenAIURL, keyOpenAIModel, keyOpenAIAPIKey,
		keyAnthropicURL, keyAnthropicModel, keyAnthropicAPIKey,
		keyRefineProvider,
		keyRefineOllamaURL, keyRefineOllamaModel,
		keyRefineOpenAIURL, keyRefineOpenAIModel, keyRefineOpenAIAPIKey,
		keyRefineAnthropicURL, keyRefineAnthropicModel, keyRefineAnthropicAPIKey,
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

// TestSeedCoversRegistry pins the link between the block registry and the seed
// step: every registered input box must land in a fresh database with the value
// and description declared by its descriptor.
func TestSeedCoversRegistry(t *testing.T) {
	useTempWorkspace(t)

	if err := bootstrap.InitDB(); err != nil {
		t.Fatalf("InitDB: %v", err)
	}

	boxes := schema.All()
	if len(boxes) == 0 {
		t.Fatal("schema block registry is empty")
	}
	rows := configRows(t)
	for _, b := range boxes {
		row, ok := rows[b.Key]
		if !ok {
			t.Errorf("registered config %s is missing from a fresh database", b.Key)
			continue
		}
		if row.ConfigValue != b.Default {
			t.Errorf("config %s = %q, want registered default %q", b.Key, row.ConfigValue, b.Default)
		}
		if row.Description != b.Description {
			t.Errorf("config %s description = %q, want %q", b.Key, row.Description, b.Description)
		}
	}
}

// A missing row is re-seeded from the registry default, but an existing row is
// never overwritten: a user's customized value survives a restart.
func TestSeedFillsMissingKeyOnExistingDB(t *testing.T) {
	useTempWorkspace(t)

	if err := bootstrap.InitDB(); err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	// storage.InitDB replaces the package singleton, so re-read it after every run.
	if err := storage.GetDB().Where("config_name = ?", keyOllamaModel).Delete(&model.Config{}).Error; err != nil {
		t.Fatalf("drop %s: %v", keyOllamaModel, err)
	}

	if err := bootstrap.InitDB(); err != nil {
		t.Fatalf("InitDB after dropping %s: %v", keyOllamaModel, err)
	}

	field, ok := schema.Lookup(keyOllamaModel)
	if !ok {
		t.Fatalf("config %s is not registered in the schema", keyOllamaModel)
	}
	row, ok := configRows(t)[keyOllamaModel]
	if !ok {
		t.Fatalf("config %s was not re-seeded on the next run", keyOllamaModel)
	}
	if row.ConfigValue != field.Default {
		t.Errorf("re-seeded config %s = %q, want default %q", keyOllamaModel, row.ConfigValue, field.Default)
	}

	if err := storage.GetDB().Model(&model.Config{}).Where("config_name = ?", keyScanExclude).
		Update("config_value", customizedScanExcludeValue).Error; err != nil {
		t.Fatalf("customize scan_exclude: %v", err)
	}
	if err := bootstrap.InitDB(); err != nil {
		t.Fatalf("InitDB after customizing scan_exclude: %v", err)
	}

	row, ok = configRows(t)[keyScanExclude]
	if !ok {
		t.Fatalf("config %s disappeared", keyScanExclude)
	}
	if row.ConfigValue != customizedScanExcludeValue {
		t.Errorf("config %s = %q, want the customized value %q to be preserved",
			keyScanExclude, row.ConfigValue, customizedScanExcludeValue)
	}
}

// TestMigrateFrameIntervalFixedPoint is the one data migration in the refactor:
// the ai_frame_interval wire flips from decimal seconds ("1"=1s) to fixed-point
// integer ("10"=1s, Shift=1). Each subcase runs in its own throwaway workspace
// because useTempWorkspace chdirs and storage.InitDB replaces the singleton.
//
// Idempotence comes ONLY from the ledger file, never from value inspection: a
// fresh database has no ai_frame_interval row to convert, but the migration must
// still record its ledger ID so the seed's new default "10" is not re-migrated
// to "100" on the next startup.
func TestMigrateFrameIntervalFixedPoint(t *testing.T) {
	t.Run("legacy 1 becomes 10", func(t *testing.T) {
		useTempWorkspace(t)
		prepareConfigTable(t)
		writeConfigRow(t, keyFrameInterval, "1")

		if err := bootstrap.InitDB(); err != nil {
			t.Fatalf("InitDB: %v", err)
		}
		if got := readConfigValue(t, keyFrameInterval); got != "10" {
			t.Errorf("ai_frame_interval = %q, want %q", got, "10")
		}
		if !ledgerContains(t, migrationFrameIntervalFixedPoint) {
			t.Errorf("ledger .tagfile/config_migrations is missing %s", migrationFrameIntervalFixedPoint)
		}
	})

	t.Run("legacy 0.5 becomes 5", func(t *testing.T) {
		useTempWorkspace(t)
		prepareConfigTable(t)
		writeConfigRow(t, keyFrameInterval, "0.5")

		if err := bootstrap.InitDB(); err != nil {
			t.Fatalf("InitDB: %v", err)
		}
		if got := readConfigValue(t, keyFrameInterval); got != "5" {
			t.Errorf("ai_frame_interval = %q, want %q", got, "5")
		}
	})

	t.Run("idempotent across re-InitDB", func(t *testing.T) {
		useTempWorkspace(t)
		prepareConfigTable(t)
		writeConfigRow(t, keyFrameInterval, "1")

		for i := 0; i < 2; i++ {
			if err := bootstrap.InitDB(); err != nil {
				t.Fatalf("InitDB run %d: %v", i+1, err)
			}
		}
		if got := readConfigValue(t, keyFrameInterval); got != "10" {
			t.Errorf("after re-InitDB ai_frame_interval = %q, want %q (no double conversion)", got, "10")
		}
	})

	t.Run("unparseable value kept as-is and no panic", func(t *testing.T) {
		useTempWorkspace(t)
		prepareConfigTable(t)
		writeConfigRow(t, keyFrameInterval, "abc")

		if err := bootstrap.InitDB(); err != nil {
			t.Fatalf("InitDB: %v", err)
		}
		if got := readConfigValue(t, keyFrameInterval); got != "abc" {
			t.Errorf("ai_frame_interval = %q, want %q (unreadable values are kept)", got, "abc")
		}
		if !ledgerContains(t, migrationFrameIntervalFixedPoint) {
			t.Errorf("kept-as-is value must still record %s, else the startup keeps re-attempting", migrationFrameIntervalFixedPoint)
		}
	})

	t.Run("fresh db seeds 10 and never reconverts", func(t *testing.T) {
		useTempWorkspace(t)
		prepareConfigTable(t)

		if err := bootstrap.InitDB(); err != nil {
			t.Fatalf("InitDB: %v", err)
		}
		if got := readConfigValue(t, keyFrameInterval); got != "10" {
			t.Fatalf("fresh seed ai_frame_interval = %q, want %q", got, "10")
		}
		if !ledgerContains(t, migrationFrameIntervalFixedPoint) {
			t.Fatalf("fresh DB must still record %s, else the next startup re-migrates", migrationFrameIntervalFixedPoint)
		}

		if err := bootstrap.InitDB(); err != nil {
			t.Fatalf("second InitDB: %v", err)
		}
		if got := readConfigValue(t, keyFrameInterval); got != "10" {
			t.Errorf("after second InitDB ai_frame_interval = %q, want %q (must NOT be re-migrated to 100)", got, "10")
		}
	})
}

// TestSeedCoversNewlyRegisteredField is the new behaviour: a key registered by
// a package that the seed step never knew about must still reach a fresh
// database.
//
// It MUST stay the last test in this file. schema.RegisterBlock mutates a
// process-global registry, so every InitDB running after it would seed a
// extra row and break the literal 22-row assertions above. The block uses its
// own ID ("testseed_blk") because "scan" is already registered by bootstrap's
// own config_schema.go init.
func TestSeedCoversNewlyRegisteredField(t *testing.T) {
	useTempWorkspace(t)

	schema.RegisterBlock(schema.ConfigBlockDescriptor{
		ID:    "testseed_blk",
		Title: "种子覆盖测试块",
		Order: 999,
		InputBoxDescriptorList: []*schema.InputBoxDescriptor{{
			Key:         "testseed_new",
			Label:       "测试",
			Description: "测试键",
			ValueType:   schema.ValueString,
			Container:   schema.ContainerSingle,
			Default:     "v",
		}},
	})

	if err := bootstrap.InitDB(); err != nil {
		t.Fatalf("InitDB: %v", err)
	}

	row, ok := configRows(t)["testseed_new"]
	if !ok {
		t.Fatalf("config testseed_new is missing: the seed step ignored the block registry")
	}
	if row.ConfigValue != "v" {
		t.Errorf("config testseed_new = %q, want %q", row.ConfigValue, "v")
	}
}
