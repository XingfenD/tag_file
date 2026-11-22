package schema_test

import (
	"strings"
	"testing"

	"github.com/XingfenD/tag_file/internal/model"
	repoFile "github.com/XingfenD/tag_file/internal/repository/file"
	"github.com/XingfenD/tag_file/internal/service/schema"
	"github.com/XingfenD/tag_file/pkg/storage"
)

func ensureDefaultGroup(t *testing.T, fileID uint) uint {
	t.Helper()
	var g model.TagGroup
	if err := storage.GetDB().Where("file_id = ? AND is_default = ?", fileID, true).First(&g).Error; err == nil {
		return g.GroupID
	}
	g = model.TagGroup{FileID: fileID, GroupName: "default", IsDefault: true}
	if err := storage.GetDB().Create(&g).Error; err != nil {
		t.Fatalf("create default group: %v", err)
	}
	return g.GroupID
}

func seedFile(t *testing.T, path string) uint {
	t.Helper()
	f, err := repoFile.NewRepository().Create().SetPath(path).SetName(path).Exec()
	if err != nil {
		t.Fatalf("seed file: %v", err)
	}
	return f.FileID
}

// buildRatingSchema: 评级(single/stars 绑 星级 字典1..5) + 收藏(bool) + 视角(multi 绑父)
func buildScene(t *testing.T, svc *schema.Service) (star, fav, vroot, v1, v2 uint) {
	t.Helper()
	star = seedTag(t, "星级", nil)
	for _, v := range []string{"1", "2", "3", "4", "5"} {
		if err := storage.GetDB().Create(&model.TagValue{TagID: star, Value: v}).Error; err != nil {
			t.Fatal(err)
		}
	}
	fav = seedTag(t, "收藏", nil)
	vroot = seedTag(t, "视角", nil)
	v1 = seedTag(t, "第一人称", &vroot)
	v2 = seedTag(t, "俯视", &vroot)

	s, err := svc.AddSchema("评级", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AddField(s.SchemaID, "星级", "single", "stars", star); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AddField(s.SchemaID, "收藏", "bool", "", fav); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AddField(s.SchemaID, "视角", "multi", "", vroot); err != nil {
		t.Fatal(err)
	}
	return
}

func fieldOfPanel(t *testing.T, svc *schema.Service, path string, label string) schema.FieldView {
	t.Helper()
	panels, err := svc.PanelsForFile(path)
	if err != nil {
		t.Fatalf("panels: %v", err)
	}
	if len(panels) != 1 {
		t.Fatalf("want 1 panel, got %d", len(panels))
	}
	for _, f := range panels[0].Fields {
		if f.Label == label {
			return f
		}
	}
	t.Fatalf("field %s not found", label)
	return schema.FieldView{}
}

func TestPanelsForFileShapes(t *testing.T) {
	defer setupTestDB(t)()
	svc := schema.NewService()
	star, _, _, v1, _ := buildScene(t, svc)
	seedFile(t, "a.jpg")

	p := fieldOfPanel(t, svc, "a.jpg", "星级")
	if p.Type != "single" || p.Widget != "stars" || !p.BindOK || p.Value != "" {
		t.Errorf("single panel = %+v", p)
	}
	if len(p.Options) != 5 || p.Options[4].Value != "5" {
		t.Errorf("stars options = %+v, want dict 1..5 in insertion order", p.Options)
	}
	m := fieldOfPanel(t, svc, "a.jpg", "视角")
	if len(m.Options) != 2 || m.Options[0].TagID != v1 || m.Options[0].Name != "第一人称" {
		t.Errorf("multi options = %+v, want child tags", m.Options)
	}
	if len(m.Checked) != 0 {
		t.Errorf("fresh multi checked = %v", m.Checked)
	}
	_ = star
}

func TestPanelsStaleBinding(t *testing.T) {
	defer setupTestDB(t)()
	svc := schema.NewService()
	star, _, _, _, _ := buildScene(t, svc)
	seedFile(t, "a.jpg")
	if err := storage.GetDB().Where("tag_id = ?", star).Delete(&model.Tag{}).Error; err != nil {
		t.Fatal(err)
	}

	panels, err := svc.PanelsForFile("a.jpg")
	if err != nil {
		t.Fatalf("stale binding must not 500: %v", err)
	}
	var stale, ok int
	for _, f := range panels[0].Fields {
		if f.BindOK {
			ok++
		} else {
			stale++
		}
	}
	if stale != 1 || ok != 2 {
		t.Errorf("want 1 stale + 2 healthy fields, got %+v", panels[0].Fields)
	}
}

func TestPanelsSinglePicksLatestRow(t *testing.T) {
	defer setupTestDB(t)()
	svc := schema.NewService()
	star, _, _, _, _ := buildScene(t, svc)
	fID := seedFile(t, "a.jpg")
	g := ensureDefaultGroup(t, fID)
	// 自由途径落两行（旧值 3、新值 4），面板读 id 最大行
	storage.GetDB().Create(&model.TagGroupTag{GroupID: g, FileID: fID, TagID: star, TagValue: "3"})
	storage.GetDB().Create(&model.TagGroupTag{GroupID: g, FileID: fID, TagID: star, TagValue: "4"})

	if p := fieldOfPanel(t, svc, "a.jpg", "星级"); p.Value != "4" {
		t.Errorf("panel value = %q, want 4 (latest row)", p.Value)
	}
}

func TestApplySingleTakesOverFreeRows(t *testing.T) {
	defer setupTestDB(t)()
	svc := schema.NewService()
	star, _, _, _, _ := buildScene(t, svc)
	fID := seedFile(t, "a.jpg")
	g := ensureDefaultGroup(t, fID)
	for _, v := range []string{"3", "野值"} {
		if err := storage.GetDB().Create(&model.TagGroupTag{GroupID: g, FileID: fID, TagID: star, TagValue: v}).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := repoFile.NewRepository().IncrementTagCnt(fID, 2); err != nil {
		t.Fatal(err)
	}

	f := fieldOfPanel(t, svc, "a.jpg", "星级")
	if f.Value != "野值" {
		t.Fatalf("panel should preview latest free row, got %q", f.Value)
	}
	if err := svc.ApplyField("a.jpg", f.FieldID, "2", nil); err != nil {
		t.Fatalf("apply: %v", err)
	}
	var rows []model.TagGroupTag
	storage.GetDB().Where("tag_id = ?", star).Find(&rows)
	if len(rows) != 1 || rows[0].TagValue != "2" {
		t.Fatalf("takeover left %+v", rows)
	}
	var frow model.File
	storage.GetDB().First(&frow, fID)
	if frow.TagCnt != 1 {
		t.Errorf("tag_cnt = %d, want 1", frow.TagCnt)
	}
}

func TestApplySingleRejectsUnknownValue(t *testing.T) {
	defer setupTestDB(t)()
	svc := schema.NewService()
	star, _, _, _, _ := buildScene(t, svc)
	seedFile(t, "a.jpg")
	f := fieldOfPanel(t, svc, "a.jpg", "星级")

	err := svc.ApplyField("a.jpg", f.FieldID, "9", nil)
	if err == nil || !strings.Contains(err.Error(), "字典") {
		t.Fatalf("out-of-dict value must be rejected, got %v", err)
	}
	var n int64
	storage.GetDB().Model(&model.TagGroupTag{}).Where("tag_id = ?", star).Count(&n)
	if n != 0 {
		t.Errorf("rejected apply wrote %d rows", n)
	}
}

func TestApplyBoolNoop(t *testing.T) {
	defer setupTestDB(t)()
	svc := schema.NewService()
	_, _, _, _, _ = buildScene(t, svc)
	fID := seedFile(t, "a.jpg")
	f := fieldOfPanel(t, svc, "a.jpg", "收藏")
	if f.Value != "false" {
		t.Fatalf("fresh bool = %q, want false", f.Value)
	}
	if err := svc.ApplyField("a.jpg", f.FieldID, "false", nil); err != nil {
		t.Fatalf("closing an untouched switch must be silent: %v", err)
	}
	var frow model.File
	storage.GetDB().First(&frow, fID)
	if frow.TagCnt != 0 {
		t.Errorf("noop moved tag_cnt to %d", frow.TagCnt)
	}
	// 开 → 关 → 开 全循环
	if err := svc.ApplyField("a.jpg", f.FieldID, "true", nil); err != nil {
		t.Fatal(err)
	}
	if p := fieldOfPanel(t, svc, "a.jpg", "收藏"); p.Value != "true" {
		t.Errorf("after on: %q", p.Value)
	}
	if err := svc.ApplyField("a.jpg", f.FieldID, "false", nil); err != nil {
		t.Fatal(err)
	}
	if p := fieldOfPanel(t, svc, "a.jpg", "收藏"); p.Value != "false" {
		t.Errorf("after off: %q", p.Value)
	}
}

func TestApplyMultiClearsAll(t *testing.T) {
	defer setupTestDB(t)()
	svc := schema.NewService()
	_, _, vroot, v1, v2 := buildScene(t, svc)
	fID := seedFile(t, "a.jpg")
	f := fieldOfPanel(t, svc, "a.jpg", "视角")

	if err := svc.ApplyField("a.jpg", f.FieldID, "", []uint{v1, v2}); err != nil {
		t.Fatalf("check two: %v", err)
	}
	// 父标签自由行（走计数，模拟正常打标途径）必须幸存
	g := ensureDefaultGroup(t, fID)
	storage.GetDB().Create(&model.TagGroupTag{GroupID: g, FileID: fID, TagID: vroot, TagValue: ""})
	if err := repoFile.NewRepository().IncrementTagCnt(fID, 1); err != nil {
		t.Fatal(err)
	}

	if err := svc.ApplyField("a.jpg", f.FieldID, "", nil); err != nil {
		t.Fatalf("clear all: %v", err)
	}
	var kids int64
	storage.GetDB().Model(&model.TagGroupTag{}).Where("tag_id IN ?", []uint{v1, v2}).Count(&kids)
	if kids != 0 {
		t.Errorf("children rows survive clear: %d", kids)
	}
	var frow model.File
	storage.GetDB().First(&frow, fID)
	if frow.TagCnt != 1 { // 只剩父标签那条自由行
		t.Errorf("tag_cnt = %d, want 1 (parent row survives)", frow.TagCnt)
	}
	var parentRows int64
	storage.GetDB().Model(&model.TagGroupTag{}).Where("tag_id = ?", vroot).Count(&parentRows)
	if parentRows != 1 {
		t.Errorf("parent free row touched: %d", parentRows)
	}
}

func TestApplyStaleBinding(t *testing.T) {
	defer setupTestDB(t)()
	svc := schema.NewService()
	star, _, _, _, _ := buildScene(t, svc)
	fID := seedFile(t, "a.jpg")
	g := ensureDefaultGroup(t, fID)
	f := fieldOfPanel(t, svc, "a.jpg", "星级")
	if err := storage.GetDB().Where("tag_id = ?", star).Delete(&model.Tag{}).Error; err != nil {
		t.Fatal(err)
	}
	before := rowsOfGroup(t, g)
	err := svc.ApplyField("a.jpg", f.FieldID, "3", nil)
	if err == nil || !strings.Contains(err.Error(), "绑定标签已失效") {
		t.Fatalf("stale binding apply: %v", err)
	}
	if rowsOfGroup(t, g) != before {
		t.Error("rejected apply changed rows")
	}
}

func TestPanelsWithoutSchemasTouchNoGroups(t *testing.T) {
	defer setupTestDB(t)()
	svc := schema.NewService()
	seedFile(t, "a.jpg")
	panels, err := svc.PanelsForFile("a.jpg")
	if err != nil {
		t.Fatal(err)
	}
	if len(panels) != 0 {
		t.Fatalf("panels = %+v", panels)
	}
	var n int64
	storage.GetDB().Model(&model.TagGroup{}).Count(&n)
	if n != 0 {
		t.Errorf("schema-less panels GET created %d tag_groups rows", n)
	}
}

func rowsOfGroup(t *testing.T, groupID uint) int64 {
	t.Helper()
	var n int64
	storage.GetDB().Model(&model.TagGroupTag{}).Where("group_id = ?", groupID).Count(&n)
	return n
}
