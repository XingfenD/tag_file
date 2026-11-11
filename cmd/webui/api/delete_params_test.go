package api

import (
	"fmt"
	"testing"

	"github.com/XingfenD/tag_file/internal/model"
	"github.com/XingfenD/tag_file/pkg/storage"
)

func TestUntagFile_QueryParams(t *testing.T) {
	r := newTestEngine(t)
	f := seedFixture(t, "a.txt")
	target := fmt.Sprintf("/api/files/tag?file_path=%s&tag_id=%d", f.Path, f.TagID)
	code, out := doRequest(t, r, "DELETE", target, "")
	if code != 200 || out["ok"] != true {
		t.Fatalf("got %d %v, want 200 ok=true", code, out)
	}
	var n int64
	storage.GetDB().Model(&model.TagGroupTag{}).Where("group_id = ?", f.DefaultGroup).Count(&n)
	if n != 0 {
		t.Fatalf("tag association still present: %d", n)
	}
}

func TestRemoveGroup_QueryParams(t *testing.T) {
	r := newTestEngine(t)
	f := seedFixture(t, "a.txt")
	target := fmt.Sprintf("/api/files/group/%d?file_path=%s", f.ExtraGroup, f.Path)
	code, out := doRequest(t, r, "DELETE", target, "")
	if code != 200 || out["ok"] != true {
		t.Fatalf("got %d %v, want 200 ok=true", code, out)
	}
	var n int64
	storage.GetDB().Model(&model.TagGroup{}).Where("group_id = ?", f.ExtraGroup).Count(&n)
	if n != 0 {
		t.Fatalf("group still present: %d", n)
	}
}

func TestRemoveTagValue_QueryParams(t *testing.T) {
	r := newTestEngine(t)
	f := seedFixture(t, "a.txt")
	code, out := doRequest(t, r, "DELETE", "/api/tags/values/0?tag_name=tmp&value=v1", "")
	if code != 200 || out["ok"] != true {
		t.Fatalf("got %d %v, want 200 ok=true", code, out)
	}
	var n int64
	storage.GetDB().Model(&model.TagValue{}).Where("tag_id = ?", f.TagID).Count(&n)
	if n != 0 {
		t.Fatalf("tag value still present: %d", n)
	}
}

func TestDeleteEndpoints_MissingParams(t *testing.T) {
	r := newTestEngine(t)
	f := seedFixture(t, "a.txt")
	cases := []struct {
		name   string
		target string
		msg    string
	}{
		{"untag", "/api/files/tag", "无效的文件路径"},
		{"group", fmt.Sprintf("/api/files/group/%d", f.ExtraGroup), "文件 '' 未注册"},
		{"value", "/api/tags/values/0", "标签名和值不能为空"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			code, out := doRequest(t, r, "DELETE", tc.target, "")
			if code != 400 || out["ok"] != false {
				t.Fatalf("got %d %v, want 400 ok=false", code, out)
			}
			if out["msg"] != tc.msg {
				t.Fatalf("msg = %q, want %q", out["msg"], tc.msg)
			}
		})
	}
}

func TestTagFile_PostStillWorks(t *testing.T) {
	r := newTestEngine(t)
	f := seedFixture(t, "a.txt")
	// TagFile 拒绝重复关联，先清掉 fixture 预置的那条，才能验证 POST 本身。
	if err := storage.GetDB().Where("group_id = ?", f.DefaultGroup).Delete(&model.TagGroupTag{}).Error; err != nil {
		t.Fatal(err)
	}
	body := fmt.Sprintf("file_path=%s&tag_id=%d", f.Path, f.TagID)
	code, out := doRequest(t, r, "POST", "/api/files/tag", body)
	if code != 200 || out["ok"] != true {
		t.Fatalf("got %d %v, want 200 ok=true", code, out)
	}
}