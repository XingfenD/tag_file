package ai

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ollamaTimeout bounds one chat request. Loading a model onto the GPU can queue
// a call behind somebody else's for minutes, and the recognition worker must
// wait that long rather than give up early — but never longer than this.
const ollamaTimeout = 180 * time.Second

// ollamaBodyLimit caps how much of a response body is read into memory. An
// answer is a few kilobytes of JSON, so anything larger means the URL points at
// something that is not Ollama (an HTML error page, a stream) and must not be
// swallowed whole.
const ollamaBodyLimit = 4 << 20

// ollamaBodySnippet is how many leading characters of a failed response are kept
// in the user facing error, enough to show Ollama's own explanation ("model
// not found", "request body too large") without pasting a whole page into it.
const ollamaBodySnippet = 200

// ollamaClient implements VisionClient against one Ollama deployment.
type ollamaClient struct {
	baseURL string
	model   string
	// timeout is the ceiling of one call, ollamaTimeout by default. It is a
	// field rather than a direct read of the constant so the deadline path can be
	// exercised quickly instead of waiting three minutes for the same code.
	timeout time.Duration
	http    *http.Client
}

// NewOllamaClient returns the vision client for one Ollama server. baseURL is the
// configured root ("http://127.0.0.1:11434"), with or without a trailing slash.
// Neither value is read from config here: the caller passes both in, so a task
// keeps using the settings that were in force when it was submitted even if the
// user edits them while it runs.
func NewOllamaClient(baseURL, model string) VisionClient {
	return &ollamaClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		model:   model,
		timeout: ollamaTimeout,
		http:    &http.Client{Timeout: ollamaTimeout},
	}
}

// Recognize posts one chat request carrying the prompt and every image
// base64-encoded, and returns message.content exactly as the model wrote it: no
// trimming, no parsing, no validation. Deciding what that text means is Task 7's
// job, which is why the transport and the answer schema stay testable apart.
func (c *ollamaClient) Recognize(ctx context.Context, prompt string, images [][]byte) (string, error) {
	if c.model == "" {
		return "", errors.New("未配置 Ollama 模型名")
	}
	if c.baseURL == "" {
		return "", errors.New("未配置 Ollama 服务地址")
	}

	encoded := make([]string, 0, len(images))
	for _, image := range images {
		// Plain base64: Ollama rejects the "data:image/jpeg;base64," prefix a
		// browser style payload would carry.
		encoded = append(encoded, base64.StdEncoding.EncodeToString(image))
	}
	body, err := json.Marshal(ollamaChatRequest{
		Model: c.model,
		Messages: []ollamaChatMessage{{
			Role:    "user",
			Content: prompt,
			Images:  encoded,
		}},
		Stream: false,
		Format: "json",
	})
	if err != nil {
		return "", fmt.Errorf("编码 Ollama 请求失败: %w", err)
	}

	// One deadline shared by the caller's context and the internal ceiling: the
	// shorter of the two fires, so a cancelled worker and a stuck model are both
	// reported as the deadline they hit.
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("创建 Ollama 请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("请求 Ollama 失败: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, ollamaBodyLimit))
	if err != nil {
		return "", fmt.Errorf("读取 Ollama 响应失败: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("Ollama 返回状态码 %d: %s", resp.StatusCode, bodySnippet(data))
	}

	var reply ollamaChatResponse
	if err := json.Unmarshal(data, &reply); err != nil {
		return "", fmt.Errorf("解析 Ollama 响应失败: %w, 响应: %s", err, bodySnippet(data))
	}
	// A model that answers nothing is a failure, not a run with zero tags: the
	// review queue would show a finished file with nothing to review.
	if reply.Message.Content == "" {
		return "", errors.New("模型返回空内容")
	}
	return reply.Message.Content, nil
}

// bodySnippet is the leading characters of a response body as it goes into an
// error message, cut on a rune boundary so a Chinese message from Ollama cannot
// end in half a character.
func bodySnippet(body []byte) string {
	text := strings.TrimSpace(string(body))
	runes := []rune(text)
	if len(runes) <= ollamaBodySnippet {
		return text
	}
	return string(runes[:ollamaBodySnippet]) + "…"
}

// The structs below are the wire format of /api/chat and stay unexported:
// nothing outside this file speaks it.
type ollamaChatRequest struct {
	Model    string              `json:"model"`
	Messages []ollamaChatMessage `json:"messages"`
	Stream   bool                `json:"stream"`
	Format   string              `json:"format"`
}

type ollamaChatMessage struct {
	Role    string   `json:"role"`
	Content string   `json:"content"`
	Images  []string `json:"images"`
}

type ollamaChatResponse struct {
	Message struct {
		Content string `json:"content"`
	} `json:"message"`
}
