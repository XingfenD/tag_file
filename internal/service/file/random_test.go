package file_test

import (
	"path/filepath"
	"testing"

	"github.com/XingfenD/tag_file/internal/model"
	"github.com/XingfenD/tag_file/internal/service/file"
	"github.com/XingfenD/tag_file/pkg/storage"
)

// seedFile writes a file row directly so the tag counter and done flag can be
// set to exact values without going through scan.
func seedFile(t *testing.T, path string, tagCnt int, done bool) *model.File {
	t.Helper()
	f := &model.File{
		FilePath: path,
		FileName: filepath.Base(path),
		FileSize: 1,
		FileType: "text/plain",
		TagCnt:   tagCnt,
		TagDone:  done,
	}
	if err := storage.GetDB().Create(f).Error; err != nil {
		t.Fatal(err)
	}
	return f
}

func TestRandomFile_DefaultScopeSkipsDone(t *testing.T) {
	setupWorkspace(t)
	svc := file.NewService()
	seedFile(t, "done.txt", 2, true)
	want := seedFile(t, "todo.txt", 0, false)

	for i := 0; i < 20; i++ {
		pick, err := svc.RandomFile("", nil)
		if err != nil {
			t.Fatalf("RandomFile: %v", err)
		}
		if pick == nil || pick.File.FileID != want.FileID {
			t.Fatalf("draw %d: got %v, want only todo.txt", i, pick)
		}
		if pick.Total != 1 || pick.Remaining != 1 {
			t.Fatalf("pool: got total=%d remaining=%d, want 1/1", pick.Total, pick.Remaining)
		}
	}
}

func TestRandomFile_UntaggedScopeSkipsTagged(t *testing.T) {
	setupWorkspace(t)
	svc := file.NewService()
	seedFile(t, "tagged.txt", 3, false)
	want := seedFile(t, "bare.txt", 0, false)

	for i := 0; i < 10; i++ {
		pick, err := svc.RandomFile(file.ScopeUntagged, nil)
		if err != nil {
			t.Fatalf("RandomFile: %v", err)
		}
		if pick == nil || pick.File.FileID != want.FileID {
			t.Fatalf("draw %d: got %v, want only bare.txt", i, pick)
		}
	}
}

func TestRandomFile_ExcludeShrinksPool(t *testing.T) {
	setupWorkspace(t)
	svc := file.NewService()
	a := seedFile(t, "a.txt", 0, false)
	b := seedFile(t, "b.txt", 0, false)
	c := seedFile(t, "c.txt", 0, false)

	pick, err := svc.RandomFile(file.ScopeAll, []uint{a.FileID, b.FileID})
	if err != nil {
		t.Fatalf("RandomFile: %v", err)
	}
	if pick == nil || pick.File.FileID != c.FileID {
		t.Fatalf("got %v, want only c.txt", pick)
	}
	if pick.Total != 3 || pick.Remaining != 1 {
		t.Fatalf("pool: got total=%d remaining=%d, want 3/1", pick.Total, pick.Remaining)
	}
}

func TestRandomFile_ExhaustedRedrawsFromScope(t *testing.T) {
	setupWorkspace(t)
	svc := file.NewService()
	a := seedFile(t, "a.txt", 0, false)
	b := seedFile(t, "b.txt", 0, false)

	pick, err := svc.RandomFile(file.ScopeAll, []uint{a.FileID, b.FileID})
	if err != nil {
		t.Fatalf("RandomFile: %v", err)
	}
	if pick == nil {
		t.Fatal("expected a redraw, got nil pick")
	}
	if !pick.Exhausted {
		t.Error("Exhausted = false, want true when every candidate was shown")
	}
	if pick.File.FileID != a.FileID && pick.File.FileID != b.FileID {
		t.Errorf("redraw picked an unexpected file: %s", pick.File.FilePath)
	}
	if pick.Total != 2 || pick.Remaining != 0 {
		t.Errorf("pool: got total=%d remaining=%d, want 2/0", pick.Total, pick.Remaining)
	}
}

func TestRandomFile_EmptyScopeReturnsNil(t *testing.T) {
	setupWorkspace(t)
	svc := file.NewService()
	seedFile(t, "done.txt", 1, true)

	pick, err := svc.RandomFile(file.ScopeTodo, nil)
	if err != nil {
		t.Fatalf("RandomFile: %v", err)
	}
	if pick != nil {
		t.Errorf("got %v, want nil pick for a scope with no files", pick)
	}
}

func TestRandomFile_InvalidScopeErrors(t *testing.T) {
	setupWorkspace(t)
	if _, err := file.NewService().RandomFile("nonsense", nil); err == nil {
		t.Error("expected an error for an unknown scope")
	}
}

// Guards against ORDER BY silently degrading to a fixed row order: with three
// candidates and no exclusions, 40 draws must hit all of them. Missing one
// candidate has probability (2/3)^40 ≈ 9e-8, far below any plausible flake rate.
func TestRandomFile_DrawsVary(t *testing.T) {
	setupWorkspace(t)
	svc := file.NewService()
	seedFile(t, "a.txt", 0, false)
	seedFile(t, "b.txt", 0, false)
	seedFile(t, "c.txt", 0, false)

	seen := make(map[uint]bool)
	for i := 0; i < 40; i++ {
		pick, err := svc.RandomFile(file.ScopeAll, nil)
		if err != nil {
			t.Fatalf("draw %d: %v", i, err)
		}
		if pick == nil {
			t.Fatalf("draw %d: nil pick", i)
		}
		seen[pick.File.FileID] = true
	}
	if len(seen) != 3 {
		t.Errorf("40 draws saw %d of 3 files, want all 3", len(seen))
	}
}
