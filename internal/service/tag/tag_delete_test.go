package tag_test

import (
	"testing"

	"github.com/XingfenD/tag_file/internal/model"
	"github.com/XingfenD/tag_file/internal/service/tag"
	"github.com/XingfenD/tag_file/pkg/storage"
	"gorm.io/gorm"
)

func mustRow(t *testing.T, db *gorm.DB, v any) {
	t.Helper()
	if err := db.Create(v).Error; err != nil {
		t.Fatalf("create %+v: %v", v, err)
	}
}

// tag_group_tags and tag_values have no database-level foreign keys, so a bare
// DELETE FROM tags leaves rows that keep showing up as tags on the file and
// keep File.TagCnt above what the user can actually see.
func TestRemoveTagByID_CleansAssociationsAndCounter(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	db := storage.GetDB()

	cat := model.TagCategory{CategoryName: "动物"}
	mustRow(t, db, &cat)
	doomed := model.Tag{CategoryID: cat.CategoryID, TagName: "英短"}
	keeper := model.Tag{CategoryID: cat.CategoryID, TagName: "布偶"}
	mustRow(t, db, &doomed)
	mustRow(t, db, &keeper)
	mustRow(t, db, &model.TagValue{TagID: doomed.TagID, Value: "蓝白"})

	f := model.File{FilePath: "a.jpg", FileName: "a.jpg", TagCnt: 2}
	mustRow(t, db, &f)
	group := model.TagGroup{FileID: f.FileID, GroupName: "default", IsDefault: true}
	mustRow(t, db, &group)
	mustRow(t, db, &model.TagGroupTag{GroupID: group.GroupID, FileID: f.FileID, TagID: doomed.TagID})
	mustRow(t, db, &model.TagGroupTag{GroupID: group.GroupID, FileID: f.FileID, TagID: keeper.TagID})

	svc := tag.NewService()
	if err := svc.RemoveTagByID(doomed.TagID); err != nil {
		t.Fatalf("remove tag: %v", err)
	}

	var associations int64
	if err := db.Model(&model.TagGroupTag{}).Where("tag_id = ?", doomed.TagID).Count(&associations).Error; err != nil {
		t.Fatalf("count associations: %v", err)
	}
	if associations != 0 {
		t.Fatalf("orphan associations = %d, want 0", associations)
	}

	var values int64
	if err := db.Model(&model.TagValue{}).Where("tag_id = ?", doomed.TagID).Count(&values).Error; err != nil {
		t.Fatalf("count values: %v", err)
	}
	if values != 0 {
		t.Fatalf("orphan predefined values = %d, want 0", values)
	}

	var got model.File
	if err := db.First(&got, f.FileID).Error; err != nil {
		t.Fatalf("reload file: %v", err)
	}
	if got.TagCnt != 1 {
		t.Fatalf("TagCnt = %d, want 1", got.TagCnt)
	}

	var kept int64
	if err := db.Model(&model.TagGroupTag{}).Where("tag_id = ?", keeper.TagID).Count(&kept).Error; err != nil {
		t.Fatalf("count kept: %v", err)
	}
	if kept != 1 {
		t.Fatalf("sibling association lost: %d rows, want 1", kept)
	}
}

// Children of a deleted tag must stay reachable: ListTagTree walks down from
// the roots, so a dangling parent_id hides them from the tree while they are
// still applied to files.
func TestRemoveTagByID_ReparentsChildrenToRoot(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	db := storage.GetDB()

	cat := model.TagCategory{CategoryName: "动物"}
	mustRow(t, db, &cat)
	parent := model.Tag{CategoryID: cat.CategoryID, TagName: "猫"}
	mustRow(t, db, &parent)
	child := model.Tag{CategoryID: cat.CategoryID, TagName: "英短", ParentID: &parent.TagID}
	mustRow(t, db, &child)

	svc := tag.NewService()
	if err := svc.RemoveTagByID(parent.TagID); err != nil {
		t.Fatalf("remove parent: %v", err)
	}

	var got model.Tag
	if err := db.First(&got, child.TagID).Error; err != nil {
		t.Fatalf("reload child: %v", err)
	}
	if got.ParentID != nil {
		t.Fatalf("child parent_id = %d, want NULL", *got.ParentID)
	}

	tags, _, err := svc.ListTags("", 0, 0)
	if err != nil {
		t.Fatalf("list tags: %v", err)
	}
	if len(tags) != 1 || tags[0].TagID != child.TagID {
		t.Fatalf("remaining tags = %+v, want only the former child", tags)
	}
}

// `tag remove <name>` documents that it deletes the name in every category;
// each of those tags has to cascade the same way.
func TestRemoveTag_ByName_CascadesEveryCategory(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	db := storage.GetDB()

	a := model.TagCategory{CategoryName: "动物"}
	b := model.TagCategory{CategoryName: "颜色"}
	mustRow(t, db, &a)
	mustRow(t, db, &b)
	tagA := model.Tag{CategoryID: a.CategoryID, TagName: "urgent"}
	tagB := model.Tag{CategoryID: b.CategoryID, TagName: "urgent"}
	mustRow(t, db, &tagA)
	mustRow(t, db, &tagB)

	f := model.File{FilePath: "a.jpg", FileName: "a.jpg", TagCnt: 2}
	mustRow(t, db, &f)
	group := model.TagGroup{FileID: f.FileID, GroupName: "default", IsDefault: true}
	mustRow(t, db, &group)
	mustRow(t, db, &model.TagGroupTag{GroupID: group.GroupID, FileID: f.FileID, TagID: tagA.TagID})
	mustRow(t, db, &model.TagGroupTag{GroupID: group.GroupID, FileID: f.FileID, TagID: tagB.TagID})

	svc := tag.NewService()
	count, err := svc.RemoveTag("urgent")
	if err != nil {
		t.Fatalf("remove by name: %v", err)
	}
	if count != 2 {
		t.Fatalf("deleted %d tags, want 2", count)
	}

	var associations int64
	if err := db.Model(&model.TagGroupTag{}).Count(&associations).Error; err != nil {
		t.Fatalf("count associations: %v", err)
	}
	if associations != 0 {
		t.Fatalf("orphan associations = %d, want 0", associations)
	}

	var got model.File
	if err := db.First(&got, f.FileID).Error; err != nil {
		t.Fatalf("reload file: %v", err)
	}
	if got.TagCnt != 0 {
		t.Fatalf("TagCnt = %d, want 0", got.TagCnt)
	}
}
