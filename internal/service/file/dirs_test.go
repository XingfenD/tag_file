package file_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/XingfenD/tag_file/internal/model"
	"github.com/XingfenD/tag_file/internal/service/file"
	"github.com/XingfenD/tag_file/pkg/storage"
)

func mkdir(t *testing.T, rel string) {
	t.Helper()
	if err := os.MkdirAll(rel, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", rel, err)
	}
}

func setScanExclude(t *testing.T, patterns ...string) {
	t.Helper()
	data, err := json.Marshal(patterns)
	if err != nil {
		t.Fatalf("marshal patterns: %v", err)
	}
	if err := storage.GetDB().Model(&model.Config{}).
		Where("config_name = ?", "scan_exclude").
		Update("config_value", string(data)).Error; err != nil {
		t.Fatalf("update scan_exclude: %v", err)
	}
}

// The typeahead must suggest exactly the directories a scan would walk:
// configured-away and hidden subtrees are pruned whole, a parent is listed
// before its children, and the root itself is not a candidate.
func TestSuggestDirs_PrunesExcludedTrees(t *testing.T) {
	setupWorkspace(t)
	for _, rel := range []string{
		"empty",
		"photos/2026",
		"photos/.thumbs",
		"cache/old",
		".cache/deep",
	} {
		mkdir(t, rel)
	}
	setScanExclude(t, `^\.`, `^cache$`)

	got, err := file.NewService().SuggestDirs(0)
	if err != nil {
		t.Fatalf("SuggestDirs: %v", err)
	}
	want := []string{"empty", "photos", "photos/2026"}
	if !samePaths(got, want) {
		t.Fatalf("dirs = %v, want %v", got, want)
	}
}

// The cap keeps the payload bounded without reordering what it does return.
func TestSuggestDirs_RespectsLimit(t *testing.T) {
	setupWorkspace(t)
	for _, rel := range []string{"a", "b", "c"} {
		mkdir(t, rel)
	}

	got, err := file.NewService().SuggestDirs(2)
	if err != nil {
		t.Fatalf("SuggestDirs: %v", err)
	}
	want := []string{"a", "b"}
	if !samePaths(got, want) {
		t.Fatalf("dirs = %v, want %v", got, want)
	}
}
