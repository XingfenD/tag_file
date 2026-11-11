package file_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/XingfenD/tag_file/internal/model"
	"github.com/XingfenD/tag_file/internal/service/file"
	"github.com/XingfenD/tag_file/pkg/storage"
)

// newName is joined straight onto the file's directory, so a name carrying a
// separator moves the file out of the tree and stores a file_name that no
// longer matches how it is listed.
func TestRenameFile_RejectsPathSeparators(t *testing.T) {
	ws := setupWorkspace(t)

	if err := os.WriteFile(filepath.Join(ws, "a.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(ws, "elsewhere"), 0o755); err != nil {
		t.Fatal(err)
	}

	svc := file.NewService()
	f, err := svc.ImportFile("a.txt")
	if err != nil {
		t.Fatalf("import: %v", err)
	}

	for _, bad := range []string{"../elsewhere/a.txt", "elsewhere/a.txt", "..", "."} {
		if _, err := svc.RenameFile(f.FilePath, bad); err == nil {
			t.Errorf("RenameFile(%q) succeeded, want a rejection", bad)
		}
		if _, statErr := os.Stat(filepath.Join(ws, "elsewhere", "a.txt")); statErr == nil {
			t.Fatalf("file escaped into elsewhere/ for name %q", bad)
		}
	}

	var got model.File
	if err := storage.GetDB().First(&got, f.FileID).Error; err != nil {
		t.Fatalf("reload file: %v", err)
	}
	if got.FilePath != "a.txt" || got.FileName != "a.txt" {
		t.Fatalf("record changed: path=%q name=%q, want a.txt/a.txt", got.FilePath, got.FileName)
	}
	if _, statErr := os.Stat(filepath.Join(ws, "a.txt")); statErr != nil {
		t.Fatalf("original file moved: %v", statErr)
	}

	renamed, err := svc.RenameFile("a.txt", "b.txt")
	if err != nil {
		t.Fatalf("plain rename: %v", err)
	}
	if renamed.FilePath != "b.txt" || renamed.FileName != "b.txt" {
		t.Fatalf("renamed = %q/%q, want b.txt/b.txt", renamed.FilePath, renamed.FileName)
	}
	if _, statErr := os.Stat(filepath.Join(ws, "b.txt")); statErr != nil {
		t.Fatalf("file on disk not renamed: %v", statErr)
	}
}

// Files copied in from outside the workspace used to land without a default
// group, which made them permanently un-untaggable.
func TestImportFile_OutsideCopy_GetsDefaultGroup(t *testing.T) {
	ws := setupWorkspace(t)

	outside := t.TempDir()
	src := filepath.Join(outside, "y.jpg")
	if err := os.WriteFile(src, []byte("y"), 0o644); err != nil {
		t.Fatal(err)
	}

	svc := file.NewService()
	f, err := svc.ImportFile(src)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(ws, "y.jpg")); statErr != nil {
		t.Fatalf("file was not copied into the workspace: %v", statErr)
	}

	groups, err := svc.ListGroups(f.FilePath)
	if err != nil {
		t.Fatalf("list groups: %v", err)
	}
	if len(groups) != 1 || !groups[0].IsDefault {
		t.Fatalf("groups = %+v, want exactly one default group", groups)
	}
}
