package search_test

import (
	"path/filepath"
	"testing"

	"github.com/XingfenD/tag_file/internal/model"
	"github.com/XingfenD/tag_file/internal/service/search"
	"github.com/XingfenD/tag_file/pkg/storage"
	"gorm.io/gorm"
)

// storage 是进程级单例，测试之间不得并行。
func setupSearchDB(t *testing.T) {
	t.Helper()
	if err := storage.InitDB(filepath.Join(t.TempDir(), "search.db")); err != nil {
		t.Fatalf("init db: %v", err)
	}
	if err := storage.GetDB().AutoMigrate(
		&model.File{}, &model.TagGroup{}, &model.TagGroupTag{},
		&model.TagCategory{}, &model.Tag{}, &model.TagValue{}, &model.Config{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	db := storage.GetDB()
	must(t, db, &model.TagCategory{CategoryName: "类型"})
	var cat model.TagCategory
	if err := db.Where("category_name = ?", "类型").First(&cat).Error; err != nil {
		t.Fatalf("load category: %v", err)
	}
	tag := model.Tag{CategoryID: cat.CategoryID, TagName: "tmp"}
	must(t, db, &tag)

	tagged := model.File{FilePath: "a.jpg", FileName: "a.jpg", FileSize: 10, FileType: "image/jpeg"}
	plain := model.File{FilePath: "b.jpg", FileName: "b.jpg", FileSize: 20, FileType: "image/jpeg"}
	must(t, db, &tagged)
	must(t, db, &plain)

	group := model.TagGroup{FileID: tagged.FileID, GroupName: "default", IsDefault: true}
	must(t, db, &group)
	must(t, db, &model.TagGroupTag{GroupID: group.GroupID, FileID: tagged.FileID, TagID: tag.TagID})
}

func must(t *testing.T, db *gorm.DB, v any) {
	t.Helper()
	if err := db.Create(v).Error; err != nil {
		t.Fatalf("create %+v: %v", v, err)
	}
}

func resultPaths(t *testing.T, files []search.FileWithTags) []string {
	t.Helper()
	paths := make([]string, 0, len(files))
	for _, f := range files {
		paths = append(paths, f.File.FilePath)
	}
	return paths
}

// An unresolvable tag has to constrain the result, not vanish from the WHERE
// clause: a dropped "has" condition silently turns into "every file matches".
func TestSearchByQuery_UnknownTagHas_ReturnsNoFiles(t *testing.T) {
	setupSearchDB(t)
	svc := search.NewService()

	got, err := svc.SearchByQuery(search.SearchQuery{
		Logic: "and",
		Conditions: []search.SearchCondition{
			{Type: "tag", Op: "has", Value: "ghost"},
		},
	})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("results = %v, want none", resultPaths(t, got))
	}

	_, total, err := svc.SearchByQueryPaged(search.SearchQuery{
		Logic: "and",
		Conditions: []search.SearchCondition{
			{Type: "tag", Op: "has", Value: "ghost"},
		},
	}, 0, 20)
	if err != nil {
		t.Fatalf("paged search: %v", err)
	}
	if total != 0 {
		t.Fatalf("paged total = %d, want 0", total)
	}
}

func TestSearchByQuery_UnknownTagNotHas_ReturnsAllFiles(t *testing.T) {
	setupSearchDB(t)
	svc := search.NewService()

	got, err := svc.SearchByQuery(search.SearchQuery{
		Logic: "and",
		Conditions: []search.SearchCondition{
			{Type: "tag", Op: "not_has", Value: "ghost"},
		},
	})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("results = %v, want both files", resultPaths(t, got))
	}
}

// An unknown tag inside a group must not take its sibling conditions with it.
func TestSearchByQuery_UnknownTagInGroup_StillConstrainsSiblings(t *testing.T) {
	setupSearchDB(t)
	svc := search.NewService()

	got, err := svc.SearchByQuery(search.SearchQuery{
		Logic: "and",
		Conditions: []search.SearchCondition{
			{
				Logic: "or",
				Conditions: []search.SearchCondition{
					{Type: "tag", Op: "has", Value: "ghost"},
					{Type: "filename", Op: "contains", Value: "b."},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(got) != 1 || got[0].File.FilePath != "b.jpg" {
		t.Fatalf("results = %v, want [b.jpg]", resultPaths(t, got))
	}
}

func TestSearchByQuery_UnknownTagValue_ReturnsNoFiles(t *testing.T) {
	setupSearchDB(t)
	svc := search.NewService()

	got, err := svc.SearchByQuery(search.SearchQuery{
		Logic: "and",
		Conditions: []search.SearchCondition{
			{Type: "tag_value", Op: "has", Tag: "ghost", Value: "x"},
		},
	})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("results = %v, want none", resultPaths(t, got))
	}
}

// A malformed operand is the caller's mistake and has to come back as an
// error rather than an unconstrained query.
func TestSearchByQuery_BadDate_IsAnError(t *testing.T) {
	setupSearchDB(t)
	svc := search.NewService()

	_, err := svc.SearchByQuery(search.SearchQuery{
		Logic: "and",
		Conditions: []search.SearchCondition{
			{Type: "date", Op: "gt", Value: "not-a-date"},
		},
	})
	if err == nil {
		t.Fatal("expected an error for an unparseable date, got nil")
	}
}

func TestSearchByQuery_EmptyExtList_IsAnError(t *testing.T) {
	setupSearchDB(t)
	svc := search.NewService()

	_, err := svc.SearchByQuery(search.SearchQuery{
		Logic: "and",
		Conditions: []search.SearchCondition{
			{Type: "ext", Op: "in", Value: []any{}},
		},
	})
	if err == nil {
		t.Fatal("expected an error for an empty extension list, got nil")
	}
}

// Guards the happy paths the rejections above must not break.
func TestSearchByQuery_KnownConditions_StillFilter(t *testing.T) {
	setupSearchDB(t)
	svc := search.NewService()

	got, err := svc.SearchByQuery(search.SearchQuery{
		Logic: "and",
		Conditions: []search.SearchCondition{
			{Type: "tag", Op: "has", Value: "tmp"},
		},
	})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(got) != 1 || got[0].File.FilePath != "a.jpg" {
		t.Fatalf("tagged results = %v, want [a.jpg]", resultPaths(t, got))
	}

	got, err = svc.SearchByQuery(search.SearchQuery{
		Logic: "and",
		Conditions: []search.SearchCondition{
			{Type: "ext", Op: "in", Value: []any{"jpg"}},
			{Type: "size", Op: "gt", Value: float64(15)},
		},
	})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(got) != 1 || got[0].File.FilePath != "b.jpg" {
		t.Fatalf("size results = %v, want [b.jpg]", resultPaths(t, got))
	}
}
