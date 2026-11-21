package file_test

import (
	"testing"

	repoFile "github.com/XingfenD/tag_file/internal/repository/file"
	"github.com/XingfenD/tag_file/internal/service/file"
	"github.com/XingfenD/tag_file/pkg/storage"
)

// 存量修复门：0.14.1 之前扫入的裸扩展名类型（musl 容器没有 /etc/mime.types 时
// mime.TypeByExtension 落空的产物）必须在启动时被重算成规范 MIME 型；
// 已经是 MIME 型的、以及兜底表外合法的裸扩展名（如 .md）一律不许碰。
func TestRepairFileTypesBackfillsBareExtensions(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	repo := repoFile.NewRepository()
	seed := []struct{ path, typ string }{
		{"vid/clip.mp4", ".mp4"},
		{"vid/clip.mov", ".mov"},
		{"pic/photo.jpg", "image/jpeg"},
		{"doc/readme.md", ".md"},
	}
	for _, s := range seed {
		if _, err := repo.Create().SetPath(s.path).SetName(leaf(s.path)).SetType(s.typ).Exec(); err != nil {
			t.Fatalf("seed %s: %v", s.path, err)
		}
	}

	if err := file.RepairFileTypes(storage.GetDB()); err != nil {
		t.Fatalf("repair: %v", err)
	}

	want := map[string]string{
		"vid/clip.mp4":  "video/mp4",
		"vid/clip.mov":  "video/quicktime",
		"pic/photo.jpg": "image/jpeg",
		"doc/readme.md": ".md",
	}
	for path, typ := range want {
		f, err := repo.Find().ByPath(path).Exec()
		if err != nil || f == nil {
			t.Fatalf("find %s: %v", path, err)
		}
		if f.FileType != typ {
			t.Errorf("file_type for %s = %q, want %q", path, f.FileType, typ)
		}
	}

	// 幂等：再跑一遍不得报错，也不得再改动任何行
	if err := file.RepairFileTypes(storage.GetDB()); err != nil {
		t.Fatalf("second repair: %v", err)
	}
}

func leaf(p string) string {
	for i := len(p) - 1; i >= 0; i-- {
		if p[i] == '/' {
			return p[i+1:]
		}
	}
	return p
}
