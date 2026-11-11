package api

import (
	"fmt"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/XingfenD/tag_file/internal/model"
	"github.com/XingfenD/tag_file/pkg/storage"
)

// 覆盖 RegisterRoutes 里除 scan / import / raw / export 之外的全部 30 条路由。
// 钉的是路由注册与取参链路：404 说明路由没注册上，500 说明 panic 被 Recovery 吞了。
func TestAllRoutes_RespondWithEnvelope(t *testing.T) {
	r := newTestEngine(t)
	f := seedFixture(t, "a.txt")

	var cat model.TagCategory
	if err := storage.GetDB().First(&cat).Error; err != nil {
		t.Fatal(err)
	}
	var assoc model.TagGroupTag
	if err := storage.GetDB().Where("group_id = ?", f.DefaultGroup).First(&assoc).Error; err != nil {
		t.Fatal(err)
	}

	searchQuery := `{"logic":"and","conditions":[{"type":"filename","op":"contains","value":"a"}]}`

	// 删标签/分类的两条排在最后：它们会拆掉 fixture，让后续路由测不到业务逻辑。
	routes := []struct{ method, target, body string }{
		{"GET", "/api/categories", ""},
		{"POST", "/api/categories", "name=smoke-cat&description=d"},
		{"PUT", fmt.Sprintf("/api/categories/%d", cat.CategoryID), "name=类型&description=d"},
		{"GET", fmt.Sprintf("/api/tags?category_id=%d", cat.CategoryID), ""},
		{"POST", "/api/tags", fmt.Sprintf("category_id=%d&tag_name=smoke-tag", cat.CategoryID)},
		{"PUT", fmt.Sprintf("/api/tags/%d", f.TagID), "tag_name=tmp&description=d"},
		{"GET", "/api/tags/values?tag_name=tmp", ""},
		{"GET", "/api/tags/values/batch?names=tmp", ""},
		{"POST", "/api/tags/values", "tag_name=tmp&value=v2"},
		{"GET", "/api/files", ""},
		{"GET", "/api/files/expand?dir=.", ""},
		{"POST", "/api/files/refresh", ""},
		{"GET", "/api/files/detail/" + f.Path, ""},
		{"GET", "/api/files/groups/" + f.Path, ""},
		{"POST", "/api/files/done", "file_path=" + f.Path + "&done=true"},
		{"POST", "/api/files/tag", fmt.Sprintf("file_path=%s&tag_id=%d", f.Path, f.TagID)},
		{"PUT", "/api/files/tag", fmt.Sprintf("id=%d&value=v1", assoc.ID)},
		{"POST", "/api/files/group", "file_path=" + f.Path + "&group_name=smoke-group"},
		{"PUT", fmt.Sprintf("/api/files/group/%d", f.ExtraGroup), "file_path=" + f.Path + "&group_name=renamed"},
		{"PUT", "/api/files/rename", "file_path=" + f.Path + "&new_name=b.txt"},
		{"GET", "/api/config", ""},
		{"POST", "/api/config", "key=smoke&value=1"},
		{"PUT", "/api/config/smoke", "value=2"},
		{"DELETE", "/api/config/smoke", ""},
		{"POST", "/api/search", "query=" + url.QueryEscape(searchQuery)},
		{"DELETE", fmt.Sprintf("/api/files/tag?file_path=%s&tag_id=%d", f.Path, f.TagID), ""},
		{"DELETE", fmt.Sprintf("/api/files/group/%d?file_path=%s", f.ExtraGroup, f.Path), ""},
		{"DELETE", "/api/tags/values/0?tag_name=tmp&value=v1", ""},
		{"DELETE", fmt.Sprintf("/api/tags/%d", f.TagID), ""},
		{"DELETE", fmt.Sprintf("/api/categories/%d", cat.CategoryID), ""},
	}

	for _, rt := range routes {
		t.Run(rt.method+" "+rt.target, func(t *testing.T) {
			code, out := doRequest(t, r, rt.method, rt.target, rt.body)
			if code != 200 && code != 400 {
				t.Fatalf("got %d %v, want 200 or 400", code, out)
			}
			if _, ok := out["ok"]; !ok {
				t.Fatalf("response has no ok key: %v", out)
			}
		})
	}
}

func TestExport_ReturnsSQL(t *testing.T) {
	r := newTestEngine(t)
	seedFixture(t, "a.txt")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/api/export", nil))
	if w.Code != 200 {
		t.Fatalf("got %d, want 200 (body=%s)", w.Code, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/sql" {
		t.Fatalf("Content-Type = %q, want application/sql", ct)
	}
	if !strings.Contains(w.Body.String(), "CREATE TABLE") {
		t.Fatalf("dump has no CREATE TABLE: %s", w.Body.String())
	}
}

func TestFileRaw_ReturnsBytes(t *testing.T) {
	r := newTestEngine(t)
	path := filepath.Join(t.TempDir(), "raw.txt")
	if err := os.WriteFile(path, []byte("raw-content"), 0644); err != nil {
		t.Fatal(err)
	}
	f := seedFixture(t, path)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", fmt.Sprintf("/api/files/raw/%d", f.FileID), nil))
	if w.Code != 200 {
		t.Fatalf("got %d, want 200 (body=%s)", w.Code, w.Body.String())
	}
	if w.Body.String() != "raw-content" {
		t.Fatalf("body = %q, want raw-content", w.Body.String())
	}
}

// scan 落 .tagfile/scan_state、import 往 cwd 复制文件，两者都必须隔离到临时 cwd。
func TestFilesystemRoutes(t *testing.T) {
	t.Chdir(t.TempDir())
	src := t.TempDir()
	if err := os.WriteFile(filepath.Join(src, "x.txt"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	r := newTestEngine(t)

	code, out := doRequest(t, r, "POST", "/api/files/scan", "dir="+url.QueryEscape(src))
	if code != 200 || out["ok"] != true {
		t.Fatalf("scan: got %d %v, want 200 ok=true", code, out)
	}

	code, out = doRequest(t, r, "POST", "/api/files/import", "path="+url.QueryEscape(filepath.Join(src, "x.txt")))
	if code != 200 || out["ok"] != true {
		t.Fatalf("import: got %d %v, want 200 ok=true", code, out)
	}
}