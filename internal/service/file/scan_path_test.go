package file_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/XingfenD/tag_file/internal/bootstrap"
	"github.com/XingfenD/tag_file/internal/model"
	"github.com/XingfenD/tag_file/internal/service/file"
	"github.com/XingfenD/tag_file/pkg/storage"
)

// setupWorkspace turns a throwaway directory into the tagging workspace and
// makes it the process working directory, which is the basis every stored
// file_path is resolved against.
func setupWorkspace(t *testing.T) string {
	t.Helper()
	ws := t.TempDir()
	t.Chdir(ws)
	// Goes through the same entry point both binaries use, so the seeded
	// scan_exclude patterns are in play.
	if err := bootstrap.InitDB(); err != nil {
		t.Fatalf("init db: %v", err)
	}
	return ws
}

func touch(t *testing.T, rel string) {
	t.Helper()
	abs := filepath.Join(".", rel)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", rel, err)
	}
	if err := os.WriteFile(abs, []byte("x"), 0o644); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}

func storedPaths(t *testing.T) []string {
	t.Helper()
	var paths []string
	if err := storage.GetDB().Model(&model.File{}).Order("file_path").Pluck("file_path", &paths).Error; err != nil {
		t.Fatalf("pluck paths: %v", err)
	}
	return paths
}

func samePaths(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

// A scan of a subdirectory must still key files by their workspace-relative
// path: the tree, detail and raw endpoints all resolve stored paths against the
// working directory, and the index-dedup map compares stored paths.
func TestScanSubdir_StoresWorkspaceRelativePaths(t *testing.T) {
	setupWorkspace(t)
	touch(t, "top.jpg")
	touch(t, filepath.Join("media", "a.jpg"))
	touch(t, filepath.Join("media", "deep", "b.jpg"))

	svc := file.NewService()
	if _, err := svc.ScanDirectoryOptions("media", true); err != nil {
		t.Fatalf("scan media: %v", err)
	}

	want := []string{
		filepath.Join("media", "a.jpg"),
		filepath.Join("media", "deep", "b.jpg"),
	}
	got := storedPaths(t)
	if !samePaths(got, want) {
		t.Fatalf("stored paths = %v, want %v", got, want)
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for _, p := range got {
		if _, statErr := os.Stat(filepath.Join(cwd, p)); statErr != nil {
			t.Errorf("stored path %q does not resolve to a file on disk: %v", p, statErr)
		}
	}
}

// Scanning the whole workspace after a subdirectory scan must recognise the
// files that scan already indexed instead of adding a second row per file.
func TestScanSubdir_ThenScanRoot_DoesNotDuplicateRows(t *testing.T) {
	setupWorkspace(t)
	touch(t, "top.jpg")
	touch(t, filepath.Join("media", "a.jpg"))

	svc := file.NewService()
	if _, err := svc.ScanDirectoryOptions("media", true); err != nil {
		t.Fatalf("scan media: %v", err)
	}
	if _, err := svc.ScanDirectoryOptions(".", true); err != nil {
		t.Fatalf("scan root: %v", err)
	}

	want := []string{filepath.Join("media", "a.jpg"), "top.jpg"}
	got := storedPaths(t)
	if !samePaths(got, want) {
		t.Fatalf("stored paths = %v, want %v", got, want)
	}
}

func TestDiffSubdir_RecognisesIndexedFiles(t *testing.T) {
	setupWorkspace(t)
	touch(t, filepath.Join("media", "a.jpg"))

	svc := file.NewService()
	if _, err := svc.ScanDirectoryOptions("media", true); err != nil {
		t.Fatalf("scan media: %v", err)
	}
	touch(t, filepath.Join("media", "late.jpg"))

	result, err := svc.DiffUnindexed("media", true)
	if err != nil {
		t.Fatalf("diff: %v", err)
	}
	// The report has to name files by the same key the index uses, otherwise
	// nothing that consumes it can act on the entry.
	want := []string{filepath.Join("media", "late.jpg")}
	if !samePaths(result.Unindexed, want) {
		t.Fatalf("unindexed = %v, want %v", result.Unindexed, want)
	}
}

func TestScanOutsideWorkspace_IsRejected(t *testing.T) {
	setupWorkspace(t)
	outside := t.TempDir()
	touchAt(t, outside, "loose.jpg")

	svc := file.NewService()
	if _, err := svc.ScanDirectoryOptions(outside, true); err == nil {
		t.Fatal("expected an error scanning outside the workspace, got nil")
	}
	if n := len(storedPaths(t)); n != 0 {
		t.Fatalf("rejected scan left %d rows behind", n)
	}
}

func touchAt(t *testing.T, dir, rel string) {
	t.Helper()
	abs := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", abs, err)
	}
	if err := os.WriteFile(abs, []byte("x"), 0o644); err != nil {
		t.Fatalf("write %s: %v", abs, err)
	}
}
