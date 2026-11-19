package api

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/XingfenD/tag_file/internal/model"
	"github.com/XingfenD/tag_file/pkg/storage"
)

// readSSEBlock reads one blank-line-terminated SSE block. The request context's
// deadline bounds every read, so a handler that never pushes fails the test
// instead of hanging it.
func readSSEBlock(t *testing.T, r *bufio.Reader) string {
	t.Helper()
	var lines []string
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			t.Fatalf("read SSE: %v (so far %q)", err, strings.Join(lines, "\n"))
		}
		line = strings.TrimRight(line, "\n")
		if line == "" {
			return strings.Join(lines, "\n")
		}
		lines = append(lines, line)
	}
}

// 端到端钉住推送链路：连接即收到初始 change（重连补齐语义），随后一次真实的
// enqueue 变更必须在保活间隔之外立刻再推一个 change——证明走的是 hub 广播，
// 不是 ticker。
func TestAIEvents_InitialEventAndPushOnEnqueue(t *testing.T) {
	r := newTestEngine(t)
	mustCreate(t, storage.GetDB(), &model.File{FilePath: "clip.mp4", FileName: "clip.mp4", FileType: "video/mp4"})

	srv := httptest.NewServer(r)
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", srv.URL+"/api/ai/events", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("Content-Type = %q, want text/event-stream", ct)
	}

	reader := bufio.NewReader(resp.Body)
	if block := readSSEBlock(t, reader); block != "retry: 3000" {
		t.Fatalf("first block = %q, want retry: 3000", block)
	}
	if block := readSSEBlock(t, reader); !strings.HasPrefix(block, "event: change\n") {
		t.Fatalf("initial block = %q, want event: change", block)
	}

	// 无 worker 运行, enqueue 只建行即返回; 变更信号由此进 hub。
	form := url.Values{"paths": {"clip.mp4"}}
	postResp, err := http.PostForm(srv.URL+"/api/ai/tasks", form)
	if err != nil {
		t.Fatal(err)
	}
	postResp.Body.Close()
	if postResp.StatusCode != 200 {
		t.Fatalf("enqueue status = %d, want 200", postResp.StatusCode)
	}

	if block := readSSEBlock(t, reader); !strings.HasPrefix(block, "event: change\n") {
		t.Fatalf("push block = %q, want event: change", block)
	}
}