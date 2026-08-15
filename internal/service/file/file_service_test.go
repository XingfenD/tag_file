package file_test

import (
	"os"
	"testing"

	"github.com/XingfenD/tag_file/internal/model"
	repoFile "github.com/XingfenD/tag_file/internal/repository/file"
	"github.com/XingfenD/tag_file/internal/service/file"
	"github.com/XingfenD/tag_file/pkg/storage"
)

func setupTestDB(t *testing.T) func() {
	t.Helper()
	dbPath := ".tag_file_test.db"
	os.Remove(dbPath)
	if err := storage.InitDB(dbPath); err != nil {
		t.Fatalf("init db: %v", err)
	}
	if err := storage.GetDB().AutoMigrate(
		&model.File{}, &model.TagGroup{}, &model.TagGroupTag{}, &model.TagCategory{}, &model.Tag{}, &model.TagValue{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return func() { os.Remove(dbPath) }
}

func TestFileService_SetFileTagDone(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	repo := repoFile.NewRepository()
	f, err := repo.Create().SetPath("a/b.jpg").SetName("b.jpg").Exec()
	if err != nil {
		t.Fatalf("create file: %v", err)
	}

	svc := file.NewService()

	if _, err := svc.SetFileTagDone("missing.jpg", true); err == nil {
		t.Fatal("expected error for unregistered file")
	}

	got, err := svc.SetFileTagDone(f.FilePath, true)
	if err != nil {
		t.Fatalf("mark done: %v", err)
	}
	if !got.TagDone {
		t.Fatal("expected TagDone=true")
	}

	got, err = svc.SetFileTagDone(f.FilePath, false)
	if err != nil {
		t.Fatalf("unmark done: %v", err)
	}
	if got.TagDone {
		t.Fatal("expected TagDone=false")
	}
}
