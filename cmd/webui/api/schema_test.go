package api

import (
	"fmt"
	"net/url"
	"strings"
	"testing"

	"github.com/XingfenD/tag_file/internal/model"
	"github.com/XingfenD/tag_file/pkg/storage"
)

func formOf(kv ...string) string {
	v := url.Values{}
	for i := 0; i+1 < len(kv); i += 2 {
		v.Add(kv[i], kv[i+1])
	}
	return v.Encode()
}

func TestSchemaAPIRoundTrip(t *testing.T) {
	r := newTestEngine(t)
	f := seedFixture(t, "a.jpg")

	// 建场景
	code, out := doRequest(t, r, "POST", "/api/schemas", formOf("name", "评级", "description", ""))
	if code != 200 || out["ok"] != true {
		t.Fatalf("create schema: %d %v", code, out)
	}
	sid := uint(out["schema_id"].(float64))

	// 加字段（绑定 fixture 标签 tmp，字典含 v1）
	code, out = doRequest(t, r, "POST", fmt.Sprintf("/api/schemas/%d/fields", sid),
		formOf("label", "星级", "type", "single", "widget", "stars", "bind_tag_id", fmt.Sprint(f.TagID)))
	if code != 200 || out["ok"] != true {
		t.Fatalf("add field: %d %v", code, out)
	}
	fid := uint(out["field_id"].(float64))

	// 管理视图带绑定标签名
	_, out = doRequest(t, r, "GET", "/api/schemas", "")
	list, _ := out["schemas"].([]any)
	if len(list) != 1 {
		t.Fatalf("schemas list = %v", out["schemas"])
	}
	sch := list[0].(map[string]any)
	field := sch["fields"].([]any)[0].(map[string]any)
	if field["bind_tag_name"] != "tmp" {
		t.Errorf("bind_tag_name = %v, want tmp", field["bind_tag_name"])
	}

	// 面板读 → apply → 面板回显 + DB 断言接管
	_, out = doRequest(t, r, "GET", "/api/schemas/panels?file_path="+f.Path, "")
	panels := out["panels"].([]any)
	if len(panels) != 1 {
		t.Fatalf("panels = %v", out["panels"])
	}
	code, out = doRequest(t, r, "POST", "/api/schemas/apply",
		formOf("file_path", f.Path, "field_id", fmt.Sprint(fid), "value", "v1"))
	if code != 200 || out["ok"] != true {
		t.Fatalf("apply: %d %v", code, out)
	}
	_, out = doRequest(t, r, "GET", "/api/schemas/panels?file_path="+f.Path, "")
	pf := out["panels"].([]any)[0].(map[string]any)["fields"].([]any)[0].(map[string]any)
	if pf["value"] != "v1" {
		t.Errorf("panel value = %v, want v1", pf["value"])
	}
	var rows []model.TagGroupTag
	storage.GetDB().Where("group_id = ? AND tag_id = ?", f.DefaultGroup, f.TagID).Find(&rows)
	if len(rows) != 1 || rows[0].TagValue != "v1" {
		t.Errorf("materialized rows = %+v, want single v1", rows)
	}
}

func TestSchemaAPIErrorEnvelope(t *testing.T) {
	r := newTestEngine(t)
	f := seedFixture(t, "a.jpg")
	_, out := doRequest(t, r, "POST", "/api/schemas", formOf("name", "评级"))
	sid := uint(out["schema_id"].(float64))
	_, out = doRequest(t, r, "POST", fmt.Sprintf("/api/schemas/%d/fields", sid),
		formOf("label", "星级", "type", "single", "bind_tag_id", fmt.Sprint(f.TagID)))
	fid := uint(out["field_id"].(float64))

	cases := []struct{ target, body string }{
		{"/api/schemas/apply", formOf("file_path", f.Path, "field_id", "999999", "value", "v1")},
		{"/api/schemas/apply", formOf("file_path", "nope.jpg", "field_id", fmt.Sprint(fid), "value", "v1")},
		{"/api/schemas/apply", formOf("file_path", f.Path, "field_id", fmt.Sprint(fid), "value", "野值")},
	}
	for i, c := range cases {
		code, out := doRequest(t, r, "POST", c.target, c.body)
		if code != 400 || out["ok"] != false {
			t.Errorf("case %d: code=%d out=%v, want 400 ok:false", i, code, out)
		}
		if msg, _ := out["msg"].(string); msg == "" {
			t.Errorf("case %d: empty msg", i)
		}
	}

	// 重复名场景 400
	code, _ := doRequest(t, r, "POST", "/api/schemas", formOf("name", "评级"))
	if code != 400 {
		t.Errorf("duplicate schema name: code=%d, want 400", code)
	}
}

func TestSchemaFieldFormArrays(t *testing.T) {
	r := newTestEngine(t)
	f := seedFixture(t, "a.jpg")
	db := storage.GetDB()

	parent := model.Tag{CategoryID: 1, TagName: "视角"}
	if err := db.Create(&parent).Error; err != nil {
		t.Fatal(err)
	}
	var kids []model.Tag
	for _, n := range []string{"第一人称", "俯视"} {
		kt := model.Tag{CategoryID: 1, TagName: n, ParentID: &parent.TagID}
		if err := db.Create(&kt).Error; err != nil {
			t.Fatal(err)
		}
		kids = append(kids, kt)
	}

	_, out := doRequest(t, r, "POST", "/api/schemas", formOf("name", "镜头"))
	sid := uint(out["schema_id"].(float64))
	_, out = doRequest(t, r, "POST", fmt.Sprintf("/api/schemas/%d/fields", sid),
		formOf("label", "视角", "type", "multi", "bind_tag_id", fmt.Sprint(parent.TagID)))
	fid := uint(out["field_id"].(float64))

	// 重复键 tag_ids（URLSearchParams 形态）
	body := "file_path=" + url.QueryEscape(f.Path) + "&field_id=" + fmt.Sprint(fid) +
		"&tag_ids=" + fmt.Sprint(kids[0].TagID) + "&tag_ids=" + fmt.Sprint(kids[1].TagID)
	if !strings.Contains(body, "tag_ids=") {
		t.Fatal("bad body")
	}
	code, out := doRequest(t, r, "POST", "/api/schemas/apply", body)
	if code != 200 || out["ok"] != true {
		t.Fatalf("multi apply: %d %v", code, out)
	}
	var n int64
	db.Model(&model.TagGroupTag{}).Where("group_id = ? AND tag_id IN ?", f.DefaultGroup,
		[]uint{kids[0].TagID, kids[1].TagID}).Count(&n)
	if n != 2 {
		t.Errorf("multi rows = %d, want 2", n)
	}
}
