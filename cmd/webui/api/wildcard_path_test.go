package api

import (
	"fmt"
	"net/url"
	"strings"
	"testing"
)

// 与前端 utils.js 的 encodePath 等价：按 / 切段、逐段转义、再拼回。
func escapePath(path string) string {
	parts := strings.Split(path, "/")
	for i, s := range parts {
		parts[i] = url.PathEscape(s)
	}
	return strings.Join(parts, "/")
}

func TestFileDetail_PathResolution(t *testing.T) {
	for _, path := range []string{"a.txt", "sub/dir/a.txt", "含 空格.txt", "a%b.txt"} {
		t.Run(path, func(t *testing.T) {
			r := newTestEngine(t)
			seedFixture(t, path)
			code, out := doRequest(t, r, "GET", "/api/files/detail/"+escapePath(path), "")
			if code != 200 || out["ok"] != true {
				t.Fatalf("got %d %v, want 200 ok=true", code, out)
			}
			detail, _ := out["detail"].(map[string]any)
			file, _ := detail["file"].(map[string]any)
			if file["path"] != path {
				t.Fatalf("path = %v, want %q", file["path"], path)
			}
		})
	}
}

func TestFileGroups_PathResolution(t *testing.T) {
	for _, path := range []string{"a.txt", "sub/dir/a.txt", "含 空格.txt", "a%b.txt"} {
		t.Run(path, func(t *testing.T) {
			r := newTestEngine(t)
			seedFixture(t, path)
			code, out := doRequest(t, r, "GET", "/api/files/groups/"+escapePath(path), "")
			if code != 200 || out["ok"] != true {
				t.Fatalf("got %d %v, want 200 ok=true", code, out)
			}
			groups, _ := out["groups"].([]any)
			names := make(map[string]bool, len(groups))
			for _, g := range groups {
				if gm, ok := g.(map[string]any); ok {
					names[fmt.Sprint(gm["name"])] = true
				}
			}
			if !names["default"] || !names["extra"] {
				t.Fatalf("groups = %v, want both default and extra", out["groups"])
			}
		})
	}
}

func TestFileDetail_Unindexed(t *testing.T) {
	r := newTestEngine(t)
	code, out := doRequest(t, r, "GET", "/api/files/detail/nope.txt", "")
	if code != 400 || out["ok"] != false {
		t.Fatalf("got %d %v, want 400 ok=false", code, out)
	}
	msg, _ := out["msg"].(string)
	if !strings.Contains(msg, "未注册") {
		t.Fatalf("msg = %q, want it to contain 未注册", msg)
	}
}
