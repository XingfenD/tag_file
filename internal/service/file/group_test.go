package file_test

import (
	"testing"

	"github.com/XingfenD/tag_file/internal/model"
	"github.com/XingfenD/tag_file/internal/service/file"
	"github.com/XingfenD/tag_file/pkg/storage"
	"gorm.io/gorm"
)

func mustRow(t *testing.T, db *gorm.DB, v any) {
	t.Helper()
	if err := db.Create(v).Error; err != nil {
		t.Fatalf("create %+v: %v", v, err)
	}
}

// File.TagCnt is a denormalised count of tag_group_tags rows; dropping a group
// has to take its rows out of the counter as well.
func TestDeleteGroup_DecrementsFileTagCnt(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	db := storage.GetDB()

	f := model.File{FilePath: "a.jpg", FileName: "a.jpg", TagCnt: 4}
	mustRow(t, db, &f)

	cat := model.TagCategory{CategoryName: "类型"}
	mustRow(t, db, &cat)
	var tags []model.Tag
	for _, name := range []string{"t1", "t2", "t3", "t4"} {
		tag := model.Tag{CategoryID: cat.CategoryID, TagName: name}
		mustRow(t, db, &tag)
		tags = append(tags, tag)
	}

	def := model.TagGroup{FileID: f.FileID, GroupName: "default", IsDefault: true}
	extra := model.TagGroup{FileID: f.FileID, GroupName: "extra"}
	mustRow(t, db, &def)
	mustRow(t, db, &extra)

	mustRow(t, db, &model.TagGroupTag{GroupID: def.GroupID, FileID: f.FileID, TagID: tags[0].TagID})
	for _, tag := range tags[1:] {
		mustRow(t, db, &model.TagGroupTag{GroupID: extra.GroupID, FileID: f.FileID, TagID: tag.TagID})
	}

	svc := file.NewService()
	if err := svc.DeleteGroup("a.jpg", extra.GroupID); err != nil {
		t.Fatalf("delete group: %v", err)
	}

	var got model.File
	if err := db.First(&got, f.FileID).Error; err != nil {
		t.Fatalf("reload file: %v", err)
	}
	if got.TagCnt != 1 {
		t.Fatalf("TagCnt = %d, want 1 (the default group keeps one tag)", got.TagCnt)
	}

	var remaining int64
	if err := db.Model(&model.TagGroupTag{}).Where("group_id = ?", extra.GroupID).Count(&remaining).Error; err != nil {
		t.Fatalf("count rows: %v", err)
	}
	if remaining != 0 {
		t.Fatalf("group rows left = %d, want 0", remaining)
	}
}

// The group id arrives from the client, so it has to be proven to belong to the
// file being tagged; otherwise a write lands in another file's group while the
// counter is bumped on this one.
func TestTagFileToGroup_RejectsForeignGroup(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	db := storage.GetDB()

	a := model.File{FilePath: "a.jpg", FileName: "a.jpg"}
	b := model.File{FilePath: "b.jpg", FileName: "b.jpg"}
	mustRow(t, db, &a)
	mustRow(t, db, &b)

	groupA := model.TagGroup{FileID: a.FileID, GroupName: "default", IsDefault: true}
	groupB := model.TagGroup{FileID: b.FileID, GroupName: "default", IsDefault: true}
	mustRow(t, db, &groupA)
	mustRow(t, db, &groupB)

	cat := model.TagCategory{CategoryName: "类型"}
	mustRow(t, db, &cat)
	tag := model.Tag{CategoryID: cat.CategoryID, TagName: "t1"}
	other := model.Tag{CategoryID: cat.CategoryID, TagName: "t2"}
	mustRow(t, db, &tag)
	mustRow(t, db, &other)
	mustRow(t, db, &model.TagGroupTag{GroupID: groupB.GroupID, FileID: b.FileID, TagID: tag.TagID})
	if err := db.Model(&b).UpdateColumn("tag_cnt", 1).Error; err != nil {
		t.Fatalf("seed count: %v", err)
	}

	svc := file.NewService()

	if err := svc.TagFileToGroup("a.jpg", other.TagID, "", groupB.GroupID); err == nil {
		t.Fatal("expected an error tagging into another file's group, got nil")
	}
	var rows int64
	if err := db.Model(&model.TagGroupTag{}).Where("group_id = ?", groupB.GroupID).Count(&rows).Error; err != nil {
		t.Fatalf("count rows: %v", err)
	}
	if rows != 1 {
		t.Fatalf("foreign group holds %d rows, want only the owner's 1", rows)
	}

	if err := svc.UntagFileFromGroup("a.jpg", tag.TagID, "", groupB.GroupID); err == nil {
		t.Fatal("expected an error untagging from another file's group, got nil")
	}
	var kept int64
	if err := db.Model(&model.TagGroupTag{}).Where("group_id = ? AND tag_id = ?", groupB.GroupID, tag.TagID).Count(&kept).Error; err != nil {
		t.Fatalf("count rows: %v", err)
	}
	if kept != 1 {
		t.Fatalf("owner's association removed: %d rows left, want 1", kept)
	}

	var gotB model.File
	if err := db.First(&gotB, b.FileID).Error; err != nil {
		t.Fatalf("reload b: %v", err)
	}
	if gotB.TagCnt != 1 {
		t.Fatalf("owner TagCnt = %d, want 1", gotB.TagCnt)
	}
}
