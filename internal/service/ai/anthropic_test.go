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

// anthropicRequest is the test side mirror of what the Anthropic client posts.
// It lives here because anthropic.go keeps its own request structs unexported.
type anthropicRequest struct {
	Model     string `json:"model"`
	MaxTokens int    `json:"max_tokens"`
	Messages  []struct {
		Role    string `json:"role"`
		Content []struct {
			Type   string `json:"type"`
			Text   string `json:"text"`
			Source struct {
				Type      string `json:"type"`
				MediaType string `json:"media_type"`
				Data      string `json:"data"`
			} `json:"source"`
		} `json:"content"`
	} `json:"messages"`
}

// TestAnthropicClient_SendsImagesThenPrompt pins the outbound contract: path,
// the two Anthropic headers, required max_tokens, image blocks (bare base64 +
// sniffed media_type) BEFORE the text block, and the joined answer verbatim.
func TestAnthropicClient_SendsImagesThenPrompt(t *testing.T) {
	var gotMethod, gotPath, gotContentType, gotKey, gotVersion string
	var gotReq anthropicRequest

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		gotContentType = r.Header.Get("Content-Type")
		gotKey, gotVersion = r.Header.Get("x-api-key"), r.Header.Get("anthropic-version")
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read request body: %v", err)
		}
		if err := json.Unmarshal(body, &gotReq); err != nil {
			t.Errorf("request body is not valid JSON (%s): %v", body, err)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"content":[{"type":"text","text":"{\"tags\":[]}"}]}`)
	}))
	defer srv.Close()

	png := pngBytes(t)
	client := ai.NewAnthropicClient(srv.URL+"/", "claude-sonnet-4-5", "sk-ant-test")
	prompt := `请为这张图挑标签,分类名固定为 "动物"`
	content, err := client.Recognize(context.Background(), prompt, [][]byte{jpegBytes, png})
	if err != nil {
		t.Fatalf("Recognize: %v", err)
	}
	if content != `{"tags":[]}` {
		t.Errorf("Recognize returned %q, want the model text verbatim", content)
	}

	if gotMethod != http.MethodPost || gotPath != "/v1/messages" {
		t.Errorf("request %s %s, want POST /v1/messages", gotMethod, gotPath)
	}
	if gotContentType != "application/json" {
		t.Errorf("Content-Type %q, want application/json", gotContentType)
	}
	if gotKey != "sk-ant-test" {
		t.Errorf("x-api-key %q, want sk-ant-test", gotKey)
	}
	if gotVersion != "2023-06-01" {
		t.Errorf("anthropic-version %q, want 2023-06-01", gotVersion)
	}
	if gotReq.Model != "claude-sonnet-4-5" {
		t.Errorf("body model %q, want claude-sonnet-4-5", gotReq.Model)
	}
	if gotReq.MaxTokens != 4096 {
		t.Errorf("max_tokens = %d, want 4096 (Anthropic 必填)", gotReq.MaxTokens)
	}
	if len(gotReq.Messages) != 1 || gotReq.Messages[0].Role != "user" {
		t.Fatalf("messages, want exactly 1 user message: %+v", gotReq.Messages)
	}
	blocks := gotReq.Messages[0].Content
	if len(blocks) != 3 {
		t.Fatalf("content has %d blocks, want 3 (2 images + text): %+v", len(blocks), blocks)
	}
	wantMedia := []string{"image/jpeg", "image/png"}
	wantData := []string{
		base64.StdEncoding.EncodeToString(jpegBytes),
		base64.StdEncoding.EncodeToString(png),
	}
	for i := range wantMedia {
		b := blocks[i]
		if b.Type != "image" || b.Source.Type != "base64" ||
			b.Source.MediaType != wantMedia[i] || b.Source.Data != wantData[i] {
			t.Errorf("blocks[%d] = %+v, want base64 image source %s", i, b, wantMedia[i])
		}
	}
	if last := blocks[2]; last.Type != "text" || last.Text != prompt {
		t.Errorf("blocks[2] = %+v, want the prompt as the LAST text block", last)
	}
}

// TestAnthropicClient_JoinsTextBlocks pins how a multipart answer becomes one
// string: only type=="text" blocks are concatenated, in order, with no
// separator. A tool_use block sandwiched between the halves must contribute
// nothing — its fields must not leak into the answer.
func TestAnthropicClient_JoinsTextBlocks(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"content":[{"type":"text","text":"{\"tags\":"},{"type":"tool_use","text":"IGNORED"},{"type":"text","text":"[]}"}]}`)
	}))
	defer srv.Close()

	content, err := ai.NewAnthropicClient(srv.URL, "m", "k").Recognize(context.Background(), "p", [][]byte{{1}})
	if err != nil {
		t.Fatalf("Recognize: %v", err)
	}
	if content != `{"tags":[]}` {
		t.Errorf("Recognize returned %q, want the text blocks joined in order without the tool_use block", content)
	}
}

// TestAnthropicClient_NoAPIKeyHeaderWithoutKey pins the auth shape of a client
// configured without a key: no x-api-key header is sent at all, while the
// anthropic-version header — required by every request — is still there.
func TestAnthropicClient_NoAPIKeyHeaderWithoutKey(t *testing.T) {
	var gotKey, gotVersion string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey, gotVersion = r.Header.Get("x-api-key"), r.Header.Get("anthropic-version")
		fmt.Fprint(w, `{"content":[{"type":"text","text":"{}"}]}`)
	}))
	defer srv.Close()

	content, err := ai.NewAnthropicClient(srv.URL, "m", "").Recognize(context.Background(), "p", [][]byte{{1}})
	if err != nil {
		t.Fatalf("Recognize without an API key: %v", err)
	}
	if content != "{}" {
		t.Errorf("Recognize returned %q, want the model text verbatim", content)
	}
	if gotKey != "" {
		t.Errorf("x-api-key %q, want no header at all when the key is empty", gotKey)
	}
	if gotVersion != "2023-06-01" {
		t.Errorf("anthropic-version %q, want 2023-06-01 even without a key", gotVersion)
	}
}

// TestAnthropicClient_HTTPErrorAndEmptyContent covers the two failure shapes
// of an answered request, mirroring the ollama test: a non-2xx status whose
// body has to survive into the error text for diagnosis (truncated at 200
// characters), and an answer whose content array holds no usable text — empty
// or only non-text blocks — which must not look like a successful run that
// proposed no tags.
func TestAnthropicClient_HTTPErrorAndEmptyContent(t *testing.T) {
	failSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "boom")
	}))
	defer failSrv.Close()

	content, err := ai.NewAnthropicClient(failSrv.URL, "m", "k").Recognize(context.Background(), "p", [][]byte{{1}})
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

	if _, err := ai.NewAnthropicClient(longSrv.URL, "m", "k").Recognize(context.Background(), "p", nil); err == nil {
		t.Fatal("Recognize on a 502 answered without an error")
	} else {
		if !strings.Contains(err.Error(), strings.Repeat("x", 200)) {
			t.Errorf("error %q, want the first 200 characters of the body", err)
		}
		if strings.Contains(err.Error(), "TAIL") {
			t.Errorf("error %q carries the tail of a long body, want it truncated at 200 characters", err)
		}
	}

	// Both empty-answer shapes — no blocks at all and blocks that carry no
	// text — are the same failure to the review queue.
	for _, answer := range []string{
		`{"content":[]}`,
		`{"content":[{"type":"tool_use","text":"x"}]}`,
	} {
		emptySrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, answer)
		}))

		content, err := ai.NewAnthropicClient(emptySrv.URL, "m", "k").Recognize(context.Background(), "p", [][]byte{{1}})
		if err == nil {
			t.Errorf("Recognize on %s answered %q, want an error", answer, content)
		} else if !strings.Contains(err.Error(), "空内容") {
			t.Errorf("error %q on %s, want it to report 模型返回空内容", err, answer)
		}
		emptySrv.Close()
	}
}

// TestAnthropicClient_MissingModelAndTimeout covers the two ways a call must
// end without ever reaching a model: an unconfigured client fails fast, and a
// server that never answers is cut off by the caller's deadline instead of
// hanging the recognition worker forever.
func TestAnthropicClient_MissingModelAndTimeout(t *testing.T) {
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

	// Both config checks are what a request cannot be built without, so they
	// come before any byte is sent.
	if _, err := ai.NewAnthropicClient(srv.URL, "", "k").Recognize(context.Background(), "p", [][]byte{{1}}); err == nil {
		t.Fatal("Recognize with an empty model answered without an error")
	} else if !strings.Contains(err.Error(), "未配置") || !strings.Contains(err.Error(), "Anthropic 模型名") {
		t.Errorf("error %q, want it to report 未配置 Anthropic 模型名", err)
	}
	// Settings() hands back an empty URL when the user clears the config entry,
	// which must not turn into a dial against a relative address.
	if _, err := ai.NewAnthropicClient("", "m", "k").Recognize(context.Background(), "p", [][]byte{{1}}); err == nil {
		t.Fatal("Recognize with an empty base URL answered without an error")
	} else if !strings.Contains(err.Error(), "未配置") || !strings.Contains(err.Error(), "Anthropic 服务地址") {
		t.Errorf("error %q, want it to report 未配置 Anthropic 服务地址", err)
	}
	if got := hits.Load(); got != 0 {
		t.Errorf("the server was contacted %d times by an unconfigured client, want 0", got)
	}

	client := ai.NewAnthropicClient(srv.URL, "m", "k")
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

// TestAnthropicClient_UndecodableSuccessBody pins the third shape of an
// answered request: a 200 whose body is not JSON at all. The error has to name
// the parse failure, because reporting 「模型返回空内容」 would blame the model
// for what is the server's envelope.
func TestAnthropicClient_UndecodableSuccessBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, "not json at all")
	}))
	defer srv.Close()

	content, err := ai.NewAnthropicClient(srv.URL, "m", "k").Recognize(context.Background(), "p", [][]byte{{1}})
	if err == nil {
		t.Fatalf("Recognize on a non-JSON 200 answered %q, want an error", content)
	}
	if !strings.Contains(err.Error(), "解析") {
		t.Errorf("error %q, want it to report that the response could not be parsed", err)
	}
	if strings.Contains(err.Error(), "空内容") {
		t.Errorf("error %q blames 模型返回空内容, want the parse failure named instead", err)
	}
	if content != "" {
		t.Errorf("Recognize returned %q next to an error, want the empty string", content)
	}
	// The body goes into the error for diagnosis, so it has to survive here too.
	if !strings.Contains(err.Error(), "not json at all") {
		t.Errorf("error %q, want the offending response body quoted in it", err)
	}
}

// TestAnthropicClient_EmptyImageListSendsTextOnly pins the other half of the
// happy path: a request with no frames at all. content must still marshal as a
// JSON array holding just the text block — a nil block slice would send
// "content": null, and the Messages API rejects it.
func TestAnthropicClient_EmptyImageListSendsTextOnly(t *testing.T) {
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotBody, _ = io.ReadAll(r.Body)
		fmt.Fprint(w, `{"content":[{"type":"text","text":"{\"tags\":[]}"}]}`)
	}))
	defer srv.Close()

	content, err := ai.NewAnthropicClient(srv.URL, "m", "k").Recognize(context.Background(), "p", nil)
	if err != nil {
		t.Fatalf("Recognize with no images: %v", err)
	}
	if content != `{"tags":[]}` {
		t.Errorf("Recognize returned %q, want the model text verbatim %q", content, `{"tags":[]}`)
	}

	var got anthropicRequest
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("request body is not valid JSON (%s): %v", gotBody, err)
	}
	if len(got.Messages) != 1 {
		t.Fatalf("body has %d messages, want exactly 1: %s", len(got.Messages), gotBody)
	}
	// encoding/json decodes "content": null into a nil slice and an array into a
	// non-nil one, which is what tells the two shapes apart here.
	blocks := got.Messages[0].Content
	if blocks == nil {
		t.Errorf(`messages[0].content is null or absent, want an array of blocks: %s`, gotBody)
	}
	if len(blocks) != 1 {
		t.Errorf("content has %d blocks for a call with no frames, want exactly 1 (text): %s", len(blocks), gotBody)
	} else if blocks[0].Type != "text" {
		t.Errorf("blocks[0].type %q, want text", blocks[0].Type)
	}
}
