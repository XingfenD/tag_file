package ai_test

import (
	"mime"
	"os"
	"path/filepath"
	"testing"

	"github.com/XingfenD/tag_file/internal/model"
	repoFile "github.com/XingfenD/tag_file/internal/repository/file"
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
		&model.Config{}, &model.AITask{}, &model.AISuggestion{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return func() { os.Remove(dbPath) }
}

func seedFile(t *testing.T, path string) *model.File {
	t.Helper()
	f, err := repoFile.NewRepository().Create().
		SetPath(path).
		SetName(filepath.Base(path)).
		SetType(mime.TypeByExtension(filepath.Ext(path))).
		Exec()
	if err != nil {
		t.Fatalf("seed file: %v", err)
	}
	return f
}

func TestAIModels_Migrate(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	m := storage.GetDB().Migrator()
	if !m.HasTable(&model.AITask{}) || !m.HasTable(&model.AISuggestion{}) {
		t.Fatal("expected ai_tasks and ai_suggestions tables")
	}
}
