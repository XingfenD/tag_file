package ai

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// visionTimeout bounds one request to the vision service. Loading a model onto
// the GPU can queue a call behind somebody else's for minutes, and the
// recognition worker must wait that long rather than give up early — but never
// longer than this.
const visionTimeout = 180 * time.Second

// visionBodyLimit caps how much of a response body is read into memory. An
// answer is a few kilobytes of JSON, so anything larger means the URL points at
// something that is not the vision service (an HTML error page, a stream) and
// must not be swallowed whole.
const visionBodyLimit = 4 << 20

// visionBodySnippet is how many leading characters of a failed response are kept
// in the user facing error, enough to show the service's own explanation
// ("model not found", "request body too large") without pasting a whole page
// into it.
const visionBodySnippet = 200

// postVisionJSON sends one JSON POST and returns the body of a 2xx answer.
// provider names the service in every error ("Ollama"/"OpenAI"/"Anthropic");
// headers carries the provider's extra headers (auth), Content-Type is set here.
// The caller wraps ctx with its own deadline before calling.
func postVisionJSON(ctx context.Context, httpClient *http.Client, url string, body []byte, headers map[string]string, provider string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("创建 %s 请求失败: %w", provider, err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求 %s 失败: %w", provider, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, visionBodyLimit))
	if err != nil {
		return nil, fmt.Errorf("读取 %s 响应失败: %w", provider, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%s 返回状态码 %d: %s", provider, resp.StatusCode, bodySnippet(data))
	}
	return data, nil
}

// bodySnippet is the leading characters of a response body as it goes into an
// error message, cut on a rune boundary so a Chinese message from the vision
// service cannot end in half a character.
func bodySnippet(body []byte) string {
	text := strings.TrimSpace(string(body))
	runes := []rune(text)
	if len(runes) <= visionBodySnippet {
		return text
	}
	return string(runes[:visionBodySnippet]) + "…"
}
