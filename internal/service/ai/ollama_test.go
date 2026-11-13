package ai_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/XingfenD/tag_file/internal/service/ai"
)

// chatRequest is the test side mirror of what the client posts. It lives here
// because ollama.go keeps its own request structs unexported, and decoding into
// a *bool for stream is what lets the test tell "stream": false apart from a
// missing stream key.
type chatRequest struct {
	Model    string `json:"model"`
	Stream   *bool  `json:"stream"`
	Format   string `json:"format"`
	Messages []struct {
		Role    string   `json:"role"`
		Content string   `json:"content"`
		Images  []string `json:"images"`
	} `json:"messages"`
}

// TestOllamaClient_SendsPromptAndBase64Images checks the whole outbound contract
// of one chat call through a real HTTP round trip: the path, the JSON body
// fields, the plain base64 images and the answer handed back untouched.
func TestOllamaClient_SendsPromptAndBase64Images(t *testing.T) {
	var gotPath, gotContentType string
	var gotReq chatRequest

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotContentType = r.Header.Get("Content-Type")
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read request body: %v", err)
		}
		if err := json.Unmarshal(body, &gotReq); err != nil {
			t.Errorf("request body is not valid JSON (%s): %v", body, err)
		}
		// A real Ollama answers with more fields than this; the client must read
		// message.content out of them and keep the text byte for byte.
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"model":"llava:13b","message":{"role":"assistant","content":"{\"tags\":[]}"}}`)
	}))
	defer srv.Close()

	// The trailing slash is part of the input: the URL is user configured, so it
	// arrives both ways and must still land on /api/chat.
	client := ai.NewOllamaClient(srv.URL+"/", "llava:13b")
	prompt := `请为这张图挑标签,分类名固定为 "动物"`
	content, err := client.Recognize(context.Background(), prompt, [][]byte{{1, 2, 3}, {4, 5, 6}})
	if err != nil {
		t.Fatalf("Recognize(%q, 2 images): %v", prompt, err)
	}
	if content != `{"tags":[]}` {
		t.Errorf("Recognize returned %q, want the model text verbatim %q", content, `{"tags":[]}`)
	}

	if gotPath != "/api/chat" {
		t.Errorf("request path %q, want %q", gotPath, "/api/chat")
	}
	if gotContentType != "application/json" {
		t.Errorf("Content-Type %q, want application/json", gotContentType)
	}
	if gotReq.Model != "llava:13b" {
		t.Errorf("body model %q, want llava:13b", gotReq.Model)
	}
	if gotReq.Stream == nil {
		t.Error(`body has no "stream" key, want "stream": false`)
	} else if *gotReq.Stream {
		t.Error("body stream is true, want false: a streamed answer cannot be read as one JSON document")
	}
	if gotReq.Format != "json" {
		t.Errorf(`body format %q, want "json"`, gotReq.Format)
	}
	if len(gotReq.Messages) != 1 {
		t.Fatalf("body has %d messages, want exactly 1: %s", len(gotReq.Messages), prompt)
	}
	msg := gotReq.Messages[0]
	if msg.Role != "user" {
		t.Errorf("messages[0].role %q, want user", msg.Role)
	}
	if msg.Content != prompt {
		t.Errorf("messages[0].content %q, want the prompt %q", msg.Content, prompt)
	}

	// Ollama wants bare base64 with no "data:image/jpeg;base64," prefix, in the
	// order the frames came in.
	wantImages := []string{
		base64.StdEncoding.EncodeToString([]byte{1, 2, 3}),
		base64.StdEncoding.EncodeToString([]byte{4, 5, 6}),
	}
	if len(msg.Images) != len(wantImages) {
		t.Fatalf("messages[0].images is %v, want %v", msg.Images, wantImages)
	}
	for i, want := range wantImages {
		if msg.Images[i] != want {
			t.Errorf("images[%d] = %q, want %q", i, msg.Images[i], want)
		}
	}
}

// TestOllamaClient_HTTPErrorAndEmptyContent covers the two failure shapes of an
// answered request: a non-2xx status, whose body has to survive into the error
// text for diagnosis, and a 200 whose content is empty, which must not look like
// a successful run that proposed no tags.
func TestOllamaClient_HTTPErrorAndEmptyContent(t *testing.T) {
	failSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "boom")
	}))
	defer failSrv.Close()

	content, err := ai.NewOllamaClient(failSrv.URL, "llava:13b").Recognize(context.Background(), "p", [][]byte{{1}})
	if err == nil {
		t.Fatalf("Recognize on a 500 answered %q, want an error", content)
	}
	if !strings.Contains(err.Error(), "500") || !strings.Contains(err.Error(), "boom") {
		t.Errorf("error %q, want it to name both the status code and the body", err)
	}
	if content != "" {
		t.Errorf("Recognize returned %q next to an error, want the empty string", content)
	}

	// Diagnosis needs the beginning of the body, not all of it: an error page can
	// be many kilobytes and would otherwise be pasted into the task log.
	longSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		fmt.Fprint(w, strings.Repeat("x", 250)+"TAIL")
	}))
	defer longSrv.Close()

	if _, err := ai.NewOllamaClient(longSrv.URL, "llava:13b").Recognize(context.Background(), "p", nil); err == nil {
		t.Fatal("Recognize on a 502 answered without an error")
	} else {
		if !strings.Contains(err.Error(), strings.Repeat("x", 200)) {
			t.Errorf("error %q, want the first 200 characters of the body", err)
		}
		if strings.Contains(err.Error(), "TAIL") {
			t.Errorf("error %q carries the tail of a long body, want it truncated at 200 characters", err)
		}
	}

	emptySrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"message":{"role":"assistant","content":""}}`)
	}))
	defer emptySrv.Close()

	content, err = ai.NewOllamaClient(emptySrv.URL, "llava:13b").Recognize(context.Background(), "p", [][]byte{{1}})
	if err == nil {
		t.Fatalf("Recognize on empty content answered %q, want an error", content)
	}
	if !strings.Contains(err.Error(), "空内容") {
		t.Errorf("error %q, want it to report 模型返回空内容", err)
	}
}

// TestOllamaClient_MissingModelAndTimeout covers the two ways a call must end
// without ever reaching a model: an unconfigured client fails fast, and a server
// that never answers is cut off by the caller's deadline instead of hanging the
// recognition worker forever.
func TestOllamaClient_MissingModelAndTimeout(t *testing.T) {
	var hits atomic.Int32
	// release stands in for a model that never answers: the handler blocks until
	// the test lets it go, so srv.Close() cannot hang on an outstanding request.
	// The time.After branch is only insurance for the case where an assertion
	// below fails and the test returns without releasing it.
	var releaseOnce sync.Once
	release := make(chan struct{})
	releaseHandler := func() { releaseOnce.Do(func() { close(release) }) }

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		select {
		case <-release:
		case <-r.Context().Done():
		case <-time.After(5 * time.Second):
		}
	}))
	defer srv.Close()
	// Registered after srv.Close so it runs first: the handler has to be let go
	// before Close waits for the outstanding request.
	defer releaseHandler()

	// The model name is what a request cannot be built without, so the check
	// comes before any byte is sent.
	if _, err := ai.NewOllamaClient(srv.URL, "").Recognize(context.Background(), "p", [][]byte{{1}}); err == nil {
		t.Fatal("Recognize with an empty model answered without an error")
	} else if !strings.Contains(err.Error(), "未配置") {
		t.Errorf("error %q, want it to report 未配置 Ollama 模型名", err)
	}
	// Settings() hands back an empty URL when the user clears the config entry,
	// which must not turn into a dial against a relative address.
	if _, err := ai.NewOllamaClient("", "llava:13b").Recognize(context.Background(), "p", [][]byte{{1}}); err == nil {
		t.Fatal("Recognize with an empty base URL answered without an error")
	} else if !strings.Contains(err.Error(), "未配置") {
		t.Errorf("error %q, want it to report 未配置 Ollama 服务地址", err)
	}
	if got := hits.Load(); got != 0 {
		t.Errorf("the server was contacted %d times by an unconfigured client, want 0", got)
	}

	client := ai.NewOllamaClient(srv.URL, "llava:13b")
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	started := time.Now()
	_, err := client.Recognize(ctx, "p", [][]byte{{1}})
	elapsed := time.Since(started)
	// The call is over, so the blocked handler can go now instead of waiting for
	// the deferred release.
	releaseHandler()
	if err == nil {
		t.Fatal("Recognize against a silent server answered without an error")
	}
	if elapsed > 5*time.Second {
		t.Errorf("Recognize took %s against a silent server, want the deadline to end it", elapsed)
	}
	// The caller's own deadline is what fired here, so the error has to stay
	// attributable to it (errors.Is on the deadline) for the worker's failure text.
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("error %v does not report the deadline that was exceeded", err)
	}
}
