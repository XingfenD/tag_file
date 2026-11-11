package api

import (
	"fmt"
	"testing"

	"github.com/XingfenD/tag_file/internal/model"
	"github.com/XingfenD/tag_file/pkg/storage"
)

func TestRandomRoute_PrefersUndoneFiles(t *testing.T) {
	r := newTestEngine(t)
	f := seedFixture(t, "todo.txt")
	mustCreate(t, storage.GetDB(), &model.File{
		FilePath: "done.txt", FileName: "done.txt", FileSize: 3, FileType: "text/plain", TagDone: true,
	})

	code, out := doRequest(t, r, "GET", "/api/files/random", "")
	if code != 200 || out["ok"] != true {
		t.Fatalf("got %d %v, want 200 ok=true", code, out)
	}
	file, _ := out["file"].(map[string]any)
	if file == nil || file["path"] != f.Path {
		t.Fatalf("file = %v, want todo.txt", file)
	}
	if out["pool_total"] != float64(1) || out["remaining"] != float64(1) {
		t.Fatalf("pool = total %v remaining %v, want 1/1", out["pool_total"], out["remaining"])
	}
}

func TestRandomRoute_ExclusionExhaustsPool(t *testing.T) {
	r := newTestEngine(t)
	f := seedFixture(t, "todo.txt")

	code, out := doRequest(t, r, "GET", fmt.Sprintf("/api/files/random?exclude=%d", f.FileID), "")
	if code != 200 || out["ok"] != true {
		t.Fatalf("got %d %v, want 200 ok=true", code, out)
	}
	if out["exhausted"] != true {
		t.Errorf("exhausted = %v, want true", out["exhausted"])
	}
	if out["remaining"] != float64(0) {
		t.Errorf("remaining = %v, want 0", out["remaining"])
	}
	file, _ := out["file"].(map[string]any)
	if file == nil || file["path"] != f.Path {
		t.Fatalf("expected a redraw of todo.txt, got %v", file)
	}
}

func TestRandomRoute_ScopeReachesService(t *testing.T) {
	r := newTestEngine(t)
	seedFixture(t, "tagged.txt")

	code, out := doRequest(t, r, "GET", "/api/files/random?scope=untagged", "")
	if code != 200 || out["ok"] != true {
		t.Fatalf("got %d %v, want 200 ok=true", code, out)
	}
	if out["file"] != nil {
		t.Errorf("file = %v, want nil: the only file already carries a tag", out["file"])
	}
}

func TestRandomRoute_EmptyPoolIsNotAnError(t *testing.T) {
	r := newTestEngine(t)

	code, out := doRequest(t, r, "GET", "/api/files/random", "")
	if code != 200 || out["ok"] != true {
		t.Fatalf("got %d %v, want 200 ok=true", code, out)
	}
	if out["file"] != nil {
		t.Errorf("file = %v, want nil with no indexed files", out["file"])
	}
}

func TestRandomRoute_RejectsUnknownScope(t *testing.T) {
	r := newTestEngine(t)
	seedFixture(t, "todo.txt")

	code, out := doRequest(t, r, "GET", "/api/files/random?scope=nonsense", "")
	if code != 400 || out["ok"] != false {
		t.Fatalf("got %d %v, want 400 ok=false", code, out)
	}
}
