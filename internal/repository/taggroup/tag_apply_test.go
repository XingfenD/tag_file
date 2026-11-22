package taggroup_test

import (
	"os"
	"testing"

	"github.com/XingfenD/tag_file/internal/model"
	repoFile "github.com/XingfenD/tag_file/internal/repository/file"
	"github.com/XingfenD/tag_file/internal/repository/taggroup"
	"github.com/XingfenD/tag_file/pkg/storage"
)

func setupTestDB(t *testing.T) func() {
	t.Helper()
	dbPath := ".tag_apply_test.db"
	os.Remove(dbPath)
	if err := storage.InitDB(dbPath); err != nil {
		t.Fatalf("init db: %v", err)
	}
	if err := storage.GetDB().AutoMigrate(
		&model.File{}, &model.TagGroup{}, &model.TagGroupTag{}, &model.TagCategory{},
		&model.Tag{}, &model.TagValue{}, &model.Config{},
		&model.AITask{}, &model.AISuggestion{}, &model.AIFrameOutput{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return func() { os.Remove(dbPath) }
}

// seedFileWithGroup creates a registered file plus its default group and
// returns their IDs.
func seedFileWithGroup(t *testing.T, repo *taggroup.Repository) (uint, uint) {
	t.Helper()
	f, err := repoFile.NewRepository().Create().SetPath("a/b.jpg").SetName("b.jpg").Exec()
	if err != nil {
		t.Fatalf("create file: %v", err)
	}
	g := &model.TagGroup{FileID: f.FileID, GroupName: "default", IsDefault: true}
	if err := repo.CreateGroup(g); err != nil {
		t.Fatalf("create group: %v", err)
	}
	return f.FileID, g.GroupID
}

func seedTag(t *testing.T, name string) uint {
	t.Helper()
	tag := model.Tag{CategoryID: 1, TagName: name}
	if err := storage.GetDB().Create(&tag).Error; err != nil {
		t.Fatalf("create tag: %v", err)
	}
	return tag.TagID
}

func rowsOf(t *testing.T, repo *taggroup.Repository, fileID uint, tagIDs []uint) []model.TagGroupTag {
	t.Helper()
	rows, err := repo.GetRowsByTags(fileID, tagIDs)
	if err != nil {
		t.Fatalf("GetRowsByTags: %v", err)
	}
	return rows
}

func tagCnt(t *testing.T, fileID uint) int64 {
	t.Helper()
	var f model.File
	if err := storage.GetDB().First(&f, fileID).Error; err != nil {
		t.Fatal(err)
	}
	return int64(f.TagCnt)
}

func TestReplaceRows(t *testing.T) {
	defer setupTestDB(t)()
	repo := taggroup.NewRepository()
	fileID, groupID := seedFileWithGroup(t, repo)
	star := seedTag(t, "星级")

	fRepo := repoFile.NewRepository()
	for _, v := range []string{"1", "2", "3"} {
		if err := repo.AddTag(groupID, fileID, star, v); err != nil {
			t.Fatalf("seed assoc: %v", err)
		}
	}
	if err := fRepo.IncrementTagCnt(fileID, 3); err != nil {
		t.Fatal(err)
	}

	// 接管式清空：删光绑定标签全部行，delta 为负，计数归零
	delta, err := repo.ReplaceRows(groupID, fileID, []uint{star}, nil)
	if err != nil {
		t.Fatalf("clear: %v", err)
	}
	if delta != -3 {
		t.Errorf("clear delta = %d, want -3", delta)
	}
	if n := len(rowsOf(t, repo, fileID, []uint{star})); n != 0 {
		t.Errorf("rows after clear = %d, want 0", n)
	}
	if c := tagCnt(t, fileID); c != 0 {
		t.Errorf("tag_cnt after clear = %d, want 0", c)
	}

	// 写入单行
	delta, err = repo.ReplaceRows(groupID, fileID, []uint{star},
		[]taggroup.ApplyRow{{TagID: star, Value: "2"}})
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	if delta != 1 {
		t.Errorf("write delta = %d, want 1", delta)
	}
	rows := rowsOf(t, repo, fileID, []uint{star})
	if len(rows) != 1 || rows[0].TagValue != "2" {
		t.Fatalf("rows after write = %+v", rows)
	}

	// no-op：空 tagIDs + 空 rows 静默返回 0，不碰既有行与计数
	delta, err = repo.ReplaceRows(groupID, fileID, nil, nil)
	if err != nil {
		t.Fatalf("noop must not error: %v", err)
	}
	if delta != 0 {
		t.Errorf("noop delta = %d, want 0", delta)
	}
	if n := len(rowsOf(t, repo, fileID, []uint{star})); n != 1 {
		t.Errorf("noop disturbed rows: %d", n)
	}
	if c := tagCnt(t, fileID); c != 1 {
		t.Errorf("noop disturbed tag_cnt: %d", c)
	}
}

func TestReplaceRowsMultiTagUnion(t *testing.T) {
	defer setupTestDB(t)()
	repo := taggroup.NewRepository()
	fileID, groupID := seedFileWithGroup(t, repo)
	c1 := seedTag(t, "第一人称")
	c2 := seedTag(t, "俯视")

	for _, id := range []uint{c1, c2} {
		if err := repo.AddTag(groupID, fileID, id, ""); err != nil {
			t.Fatalf("seed assoc: %v", err)
		}
	}
	fRepo := repoFile.NewRepository()
	if err := fRepo.IncrementTagCnt(fileID, 2); err != nil {
		t.Fatal(err)
	}

	// 子标签整组接管：旧两行删净、仅保留勾选集里的一行
	delta, err := repo.ReplaceRows(groupID, fileID, []uint{c1, c2},
		[]taggroup.ApplyRow{{TagID: c2}})
	if err != nil {
		t.Fatalf("replace: %v", err)
	}
	if delta != -1 {
		t.Errorf("delta = %d, want -1", delta)
	}
	rows := rowsOf(t, repo, fileID, []uint{c1, c2})
	if len(rows) != 1 || rows[0].TagID != c2 {
		t.Fatalf("rows = %+v, want only c2", rows)
	}
	if c := tagCnt(t, fileID); c != 1 {
		t.Errorf("tag_cnt = %d, want 1", c)
	}
}
