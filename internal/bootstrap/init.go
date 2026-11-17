package bootstrap

import (
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/XingfenD/tag_file/internal/constant"
	"github.com/XingfenD/tag_file/internal/model"
	"github.com/XingfenD/tag_file/internal/service/ai"
	"github.com/XingfenD/tag_file/internal/service/config/schema"
	"github.com/XingfenD/tag_file/pkg/storage"
	"gorm.io/gorm"
)

var defaultExcludePatterns = []string{
	`^\$RECYCLE\.BIN$`,
	`^System Volume Information$`,
	`^\.`,
}

// seedDefaultConfig walks the config block registry and inserts only the input
// boxes that are missing, so boxes registered later still reach an existing
// database (an early return on the first key would skip them forever). Value and
// description come from the InputBoxDescriptor declaration the settings page
// already reads, so a fresh database and an upgraded one cannot disagree with
// what the code actually uses.
func seedDefaultConfig(db *gorm.DB) {
	for _, box := range schema.AllInputBoxes() {
		var count int64
		config := model.Config{
			ConfigName:  box.Key,
			ConfigValue: box.Default,
			Description: box.Description,
		}

		db.Model(&model.Config{}).Where("config_name = ?", config.ConfigName).Count(&count)
		if count > 0 {
			continue
		}
		if err := db.Create(&config).Error; err != nil {
			log.Printf("[bootstrap] seed default config %s: %v", config.ConfigName, err)
		}
	}
}

func backupDBIfNeeded(dbPath string) error {
	versionFile := filepath.Join(filepath.Dir(dbPath), "core_version")
	workspaceDir := filepath.Dir(dbPath)

	prevVersion, err := os.ReadFile(versionFile)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read version file: %w", err)
	}

	if string(prevVersion) == constant.CoreVersion {
		return nil
	}

	if len(prevVersion) > 0 {
		ts := time.Now().Format("20060102_150405")
		backupPath := fmt.Sprintf("%s.bak.%s", dbPath, ts)
		src, err := os.Open(dbPath)
		if err != nil {
			return fmt.Errorf("open db for backup: %w", err)
		}
		defer src.Close()
		dst, err := os.Create(backupPath)
		if err != nil {
			return fmt.Errorf("create backup: %w", err)
		}
		defer dst.Close()
		if _, err := io.Copy(dst, src); err != nil {
			return fmt.Errorf("copy db to backup: %w", err)
		}
		log.Printf("[bootstrap] DB backed up: %s (version %s → %s)", backupPath, string(prevVersion), constant.CoreVersion)
	} else {
		log.Printf("[bootstrap] First run, recording core version %s", constant.CoreVersion)
	}

	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		return fmt.Errorf("ensure workspace dir: %w", err)
	}
	return os.WriteFile(versionFile, []byte(constant.CoreVersion), 0644)
}

// migrationFrameIntervalFixedPoint is the ledger ID of the one data migration in
// the config refactor: ai_frame_interval flips from decimal seconds ("1"=1s) to
// a fixed-point integer wire ("10"=1s, Shift=1). Idempotence comes ONLY from the
// ledger file below, never from value inspection: even when there is nothing to
// convert, the ID must be recorded so a later startup never re-migrates the
// row the seed just wrote.
const migrationFrameIntervalFixedPoint = "ai_frame_interval_fixedpoint_v1"

// migrateConfigStorage applies one-time data migrations stored as rows in the
// config table. The ledger (.tagfile/config_migrations, one applied ID per line,
// next to core_version) is the sole idempotence guard.
//
// Failure handling is best-effort: a migration issue logs and continues so
// startup is never blocked. The one exception is the ledger write itself — the
// ID is appended BEFORE the converted value is written, so a ledger append
// failure aborts the migration for this startup (leaving the row unconverted,
// to be marked invalid by validation and retried next start once the ledger is
// writable) instead of converting the value and then failing to record it,
// which would double-convert on the next run.
func migrateConfigStorage(db *gorm.DB, dbPath string) error {
	ledgerPath := filepath.Join(filepath.Dir(dbPath), "config_migrations")

	applied, err := readMigrationLedger(ledgerPath)
	if err != nil {
		// Cannot tell what already ran; touching the row here could double-convert.
		log.Printf("[bootstrap] skip migration %s: read ledger: %v", migrationFrameIntervalFixedPoint, err)
		return nil
	}
	if applied[migrationFrameIntervalFixedPoint] {
		return nil
	}

	// Compute the converted value before touching the ledger so a ledger write
	// failure aborts cleanly with the row untouched.
	var cfg model.Config
	err = db.Where("config_name = ?", ai.KeyFrameInterval).First(&cfg).Error
	newValue := ""
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		// Fresh database: nothing to convert, but still record the ID below.
	case err != nil:
		log.Printf("[bootstrap] skip migration %s: read row: %v", migrationFrameIntervalFixedPoint, err)
		return nil
	default:
		value, perr := strconv.ParseFloat(cfg.ConfigValue, 64)
		if perr != nil {
			// Unreadable values are kept and handed to the settings page's
			// validation to flag, never silently rewritten.
			log.Printf("[bootstrap] migration %s: %q is not a legacy decimal, keeping as-is", migrationFrameIntervalFixedPoint, cfg.ConfigValue)
		} else {
			scaled := int64(math.Round(value * math.Pow(10, float64(ai.FrameIntervalShift))))
			newValue = strconv.FormatInt(scaled, 10)
		}
	}

	if err := appendMigrationLedger(ledgerPath, migrationFrameIntervalFixedPoint); err != nil {
		log.Printf("[bootstrap] abort migration %s: append ledger: %v", migrationFrameIntervalFixedPoint, err)
		return nil
	}

	if newValue == "" {
		return nil // fresh DB or kept-as-is unreadable value: nothing to write
	}
	if err := db.Model(&model.Config{}).Where("config_name = ?", ai.KeyFrameInterval).
		Update("config_value", newValue).Error; err != nil {
		log.Printf("[bootstrap] migration %s: write %q: %v", migrationFrameIntervalFixedPoint, newValue, err)
		return nil
	}
	log.Printf("[bootstrap] migration %s: %q → %q", migrationFrameIntervalFixedPoint, cfg.ConfigValue, newValue)
	return nil
}

// readMigrationLedger returns the set of applied migration IDs. A missing ledger
// means nothing has run yet; any other read error is returned so the caller can
// bail out rather than risk re-applying a migration it cannot see.
func readMigrationLedger(path string) (map[string]bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]bool{}, nil
		}
		return nil, err
	}
	applied := map[string]bool{}
	for _, line := range strings.Split(string(data), "\n") {
		if id := strings.TrimSpace(line); id != "" {
			applied[id] = true
		}
	}
	return applied, nil
}

func appendMigrationLedger(path string, id string) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.WriteString(id + "\n"); err != nil {
		return err
	}
	return nil
}

func InitDB() error {
	dbPath := ".tagfile/tagfile.db"

	if err := backupDBIfNeeded(dbPath); err != nil {
		return err
	}

	if err := storage.InitDB(dbPath); err != nil {
		return err
	}

	if err := storage.GetDB().AutoMigrate(
		&model.File{},
		&model.TagGroup{},
		&model.TagGroupTag{},
		&model.TagCategory{},
		&model.Tag{},
		&model.TagValue{},
		&model.Config{},
		&model.AITask{},
		&model.AISuggestion{},
	); err != nil {
		return err
	}

	if err := migrateConfigStorage(storage.GetDB(), dbPath); err != nil {
		return err
	}

	seedDefaultConfig(storage.GetDB())
	return nil
}
