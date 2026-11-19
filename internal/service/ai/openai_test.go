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

// openAIChatRequest is the test side mirror of what the OpenAI client posts.
// It lives here because openai.go keeps its own request structs unexported,
// and decoding into a *bool for stream is what lets the test tell
// "stream": false apart from a missing stream key, exactly like chatRequest
// in ollama_test.go.
type openAIChatRequest struct {
	Model          string `json:"model"`
	Stream         *bool  `json:"stream"`
	ResponseFormat struct {
		Type string `json:"type"`
	} `json:"response_format"`
	Messages []struct {
		Role    string `json:"role"`
		Content []struct {
			Type     string `json:"type"`
			Text     string `json:"text"`
			ImageURL struct {
				URL string `json:"url"`
			} `json:"image_url"`
		} `json:"content"`
	} `json:"messages"`
}

// 测试用图片字节：JPEG 魔数（http.DetectContentType 只读前 512 字节即可判型）
// 与一张真实的 1×1 透明 PNG。两种 MIME 都必须按字节嗅探，不能写死 image/jpeg。
var jpegBytes = []byte{0xFF, 0xD8, 0xFF, 0xE0}

func pngBytes(t *testing.T) []byte {
	t.Helper()
	data, err := base64.StdEncoding.DecodeString(
		"iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg==")
	if err != nil {
		t.Fatalf("decode test png: %v", err)
	}
	return data
}

// TestOpenAIClient_SendsPromptAndDataURIImages pins the whole outbound contract
// through a real HTTP round trip: path, JSON fields, data-URI images with
// sniffed MIME types in frame order, the bearer header, and the answer verbatim.
func TestOpenAIClient_SendsPromptAndDataURIImages(t *testing.T) {
	var gotMethod, gotPath, gotContentType, gotAuth string
	var gotReq openAIChatRequest

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		gotContentType, gotAuth = r.Header.Get("Content-Type"), r.Header.Get("Authorization")
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read request body: %v", err)
		}
		if err := json.Unmarshal(body, &gotReq); err != nil {
			t.Errorf("request body is not valid JSON (%s): %v", body, err)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"choices":[{"message":{"role":"assistant","content":"{\"tags\":[]}"}}]}`)
	}))
	defer srv.Close()

	png := pngBytes(t)
	// 尾斜杠是输入的一部分：地址由用户配置，两种写法都必须落到 /chat/completions。
	// base 已含 /v1（Pinned Decision 3），客户端不再追加前缀。
	client := ai.NewOpenAIClient(srv.URL+"/", "gpt-4o", "sk-test")
	prompt := `请为这张图挑标签,分类名固定为 "动物"`
	content, err := client.Recognize(context.Background(), prompt, [][]byte{jpegBytes, png})
	if err != nil {
		t.Fatalf("Recognize: %v", err)
	}
	if content != `{"tags":[]}` {
		t.Errorf("Recognize returned %q, want the model text verbatim", content)
	}

	if gotMethod != http.MethodPost || gotPath != "/chat/completions" {
		t.Errorf("request %s %s, want POST /chat/completions", gotMethod, gotPath)
	}
	if gotContentType != "application/json" {
		t.Errorf("Content-Type %q, want application/json", gotContentType)
	}
	if gotAuth != "Bearer sk-test" {
		t.Errorf("Authorization %q, want Bearer sk-test", gotAuth)
	}
	if gotReq.Model != "gpt-4o" {
		t.Errorf("body model %q, want gpt-4o", gotReq.Model)
	}
	if gotReq.Stream == nil || *gotReq.Stream {
		t.Error(`body stream, want explicit false`)
	}
	if gotReq.ResponseFormat.Type != "json_object" {
		t.Errorf("response_format.type %q, want json_object", gotReq.ResponseFormat.Type)
	}
	if len(gotReq.Messages) != 1 || gotReq.Messages[0].Role != "user" {
		t.Fatalf("messages, want exactly 1 user message: %+v", gotReq.Messages)
	}
	parts := gotReq.Messages[0].Content
	if len(parts) != 3 {
		t.Fatalf("content has %d parts, want 3 (text + 2 images): %+v", len(parts), parts)
	}
	if parts[0].Type != "text" || parts[0].Text != prompt {
		t.Errorf("parts[0] = %+v, want the prompt as a text part", parts[0])
	}
	wantURLs := []string{
		"data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(jpegBytes),
		"data:image/png;base64," + base64.StdEncoding.EncodeToString(png),
	}
	for i, want := range wantURLs {
		p := parts[i+1]
		if p.Type != "image_url" || p.ImageURL.URL != want {
			t.Errorf("parts[%d] = %+v, want image_url %q", i+1, p, want)
		}
	}
}

// TestOpenAIClient_NoAuthHeaderWithoutKey pins the auth shape of a client
// configured without a key: no Authorization header is sent at all. An empty
// "Bearer " would be rejected by local OpenAI-compatible servers that need no
// auth.
func TestOpenAIClient_NoAuthHeaderWithoutKey(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		fmt.Fprint(w, `{"choices":[{"message":{"content":"{}"}}]}`)
	}))
	defer srv.Close()

	content, err := ai.NewOpenAIClient(srv.URL, "m", "").Recognize(context.Background(), "p", [][]byte{{1}})
	if err != nil {
		t.Fatalf("Recognize without an API key: %v", err)
	}
	if content != "{}" {
		t.Errorf("Recognize returned %q, want the model text verbatim", content)
	}
	if gotAuth != "" {
		t.Errorf("Authorization %q, want no header at all when the key is empty", gotAuth)
	}
}

// TestOpenAIClient_HTTPErrorAndEmptyContent covers the two failure shapes of an
// answered request, mirroring the ollama test: a non-2xx status whose body has
// to survive into the error text for diagnosis (truncated at 200 characters),
// and an answer that carries no text — an empty content or no choices at all —
// which must not look like a successful run that proposed no tags.
func TestOpenAIClient_HTTPErrorAndEmptyContent(t *testing.T) {
	failSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "boom")
	}))
	defer failSrv.Close()

	content, err := ai.NewOpenAIClient(failSrv.URL, "m", "k").Recognize(context.Background(), "p", [][]byte{{1}})
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

	if _, err := ai.NewOpenAIClient(longSrv.URL, "m", "k").Recognize(context.Background(), "p", nil); err == nil {
		t.Fatal("Recognize on a 502 answered without an error")
	} else {
		if !strings.Contains(err.Error(), strings.Repeat("x", 200)) {
			t.Errorf("error %q, want the first 200 characters of the body", err)
		}
		if strings.Contains(err.Error(), "TAIL") {
			t.Errorf("error %q carries the tail of a long body, want it truncated at 200 characters", err)
		}
	}

	// Both empty-answer shapes — a blank content and no choices at all — are the
	// same failure to the review queue.
	for _, answer := range []string{
		`{"choices":[{"message":{"content":""}}]}`,
		`{"choices":[]}`,
	} {
		emptySrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, answer)
		}))

		content, err := ai.NewOpenAIClient(emptySrv.URL, "m", "k").Recognize(context.Background(), "p", [][]byte{{1}})
		if err == nil {
			t.Errorf("Recognize on %s answered %q, want an error", answer, content)
		} else if !strings.Contains(err.Error(), "空内容") {
			t.Errorf("error %q on %s, want it to report 模型返回空内容", err, answer)
		}
		emptySrv.Close()
	}
}

// TestOpenAIClient_MissingModelAndTimeout covers the two ways a call must end
// without ever reaching a model: an unconfigured client fails fast, and a server
// that never answers is cut off by the caller's deadline instead of hanging the
// recognition worker forever.
func TestOpenAIClient_MissingModelAndTimeout(t *testing.T) {
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
	if _, err := ai.NewOpenAIClient(srv.URL, "", "k").Recognize(context.Background(), "p", [][]byte{{1}}); err == nil {
		t.Fatal("Recognize with an empty model answered without an error")
	} else if !strings.Contains(err.Error(), "未配置") || !strings.Contains(err.Error(), "OpenAI 模型名") {
		t.Errorf("error %q, want it to report 未配置 OpenAI 模型名", err)
	}
	// Settings() hands back an empty URL when the user clears the config entry,
	// which must not turn into a dial against a relative address.
	if _, err := ai.NewOpenAIClient("", "m", "k").Recognize(context.Background(), "p", [][]byte{{1}}); err == nil {
		t.Fatal("Recognize with an empty base URL answered without an error")
	} else if !strings.Contains(err.Error(), "未配置") || !strings.Contains(err.Error(), "OpenAI 服务地址") {
		t.Errorf("error %q, want it to report 未配置 OpenAI 服务地址", err)
	}
	if got := hits.Load(); got != 0 {
		t.Errorf("the server was contacted %d times by an unconfigured client, want 0", got)
	}

	client := ai.NewOpenAIClient(srv.URL, "m", "k")
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

// TestOpenAIClient_UndecodableSuccessBody pins the third shape of an answered
// request: a 200 whose body is not JSON at all. The error has to name the parse
// failure, because reporting 「模型返回空内容」 would blame the model for what is
// the server's envelope.
func TestOpenAIClient_UndecodableSuccessBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, "not json at all")
	}))
	defer srv.Close()

	content, err := ai.NewOpenAIClient(srv.URL, "m", "k").Recognize(context.Background(), "p", [][]byte{{1}})
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

// TestOpenAIClient_EmptyImageListSendsTextOnly pins the other half of the happy
// path: a request with no frames at all. content must still marshal as a JSON
// array holding just the text part — a nil part slice would send "content":
// null, and some servers reject the message outright.
func TestOpenAIClient_EmptyImageListSendsTextOnly(t *testing.T) {
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotBody, _ = io.ReadAll(r.Body)
		fmt.Fprint(w, `{"choices":[{"message":{"content":"{\"tags\":[]}"}}]}`)
	}))
	defer srv.Close()

	content, err := ai.NewOpenAIClient(srv.URL, "m", "k").Recognize(context.Background(), "p", nil)
	if err != nil {
		t.Fatalf("Recognize with no images: %v", err)
	}
	if content != `{"tags":[]}` {
		t.Errorf("Recognize returned %q, want the model text verbatim %q", content, `{"tags":[]}`)
	}

	var got openAIChatRequest
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("request body is not valid JSON (%s): %v", gotBody, err)
	}
	if len(got.Messages) != 1 {
		t.Fatalf("body has %d messages, want exactly 1: %s", len(got.Messages), gotBody)
	}
	// encoding/json decodes "content": null into a nil slice and an array into a
	// non-nil one, which is what tells the two shapes apart here.
	parts := got.Messages[0].Content
	if parts == nil {
		t.Errorf(`messages[0].content is null or absent, want an array of parts: %s`, gotBody)
	}
	if len(parts) != 1 {
		t.Errorf("content has %d parts for a call with no frames, want exactly 1 (text): %s", len(parts), gotBody)
	} else if parts[0].Type != "text" {
		t.Errorf("parts[0].type %q, want text", parts[0].Type)
	}
}
