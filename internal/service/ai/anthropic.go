package ai

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// anthropicMaxTokens is required by the Messages API — a request without it is
// rejected outright. 4096 comfortably covers the prompt's 1–6 tag JSON with room.
const anthropicMaxTokens = 4096

// anthropicVersion is the API version sent in the anthropic-version header on
// every request; the wire format this client speaks.
const anthropicVersion = "2023-06-01"

// anthropicClient implements VisionClient against the Anthropic Messages API.
type anthropicClient struct {
	baseURL string
	model   string
	apiKey  string
	// timeout is the ceiling of one call, visionTimeout by default. It is a
	// field rather than a direct read of the constant so the deadline path can be
	// exercised quickly instead of waiting three minutes for the same code.
	timeout time.Duration
	http    *http.Client
}

// NewAnthropicClient returns the vision client for the Anthropic Messages API.
// baseURL is the configured root ("https://api.anthropic.com"), with or without
// a trailing slash; the client appends /v1/messages. Neither value is read from
// config here: the caller passes all three in, so a task keeps using the
// settings that were in force when it was submitted even if the user edits them
// while it runs. An empty apiKey sends no x-api-key header at all, while the
// anthropic-version header is sent either way.
func NewAnthropicClient(baseURL, model, apiKey string) VisionClient {
	return &anthropicClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		model:   model,
		apiKey:  apiKey,
		timeout: visionTimeout,
		http:    &http.Client{Timeout: visionTimeout},
	}
}

// Recognize posts one message carrying every image as a base64 block followed
// by the prompt as the last text block, and returns the model's answer with
// only its type=="text" content blocks concatenated in order, no separator,
// exactly as the model wrote them: no trimming, no parsing, no validation.
// Deciding what that text means is the review gate's job, which is why the
// transport and the answer schema stay testable apart.
func (c *anthropicClient) Recognize(ctx context.Context, prompt string, images [][]byte) (string, error) {
	if c.model == "" {
		return "", errors.New("未配置 Anthropic 模型名")
	}
	if c.baseURL == "" {
		return "", errors.New("未配置 Anthropic 服务地址")
	}

	// content is always an array of blocks, even with no frames: a nil block
	// slice would marshal as "content": null. The images come first in frame
	// order and the prompt is the LAST text block.
	blocks := make([]anthropicContentBlock, 0, len(images)+1)
	for _, image := range images {
		// The MIME type is sniffed per image (http.DetectContentType reads the
		// magic bytes): labeling a PNG frame image/jpeg is what a hardcoded type
		// would do, and the API rejects the mismatch.
		blocks = append(blocks, anthropicContentBlock{
			Type: "image",
			Source: &anthropicImageSource{
				Type:      "base64",
				MediaType: http.DetectContentType(image),
				Data:      base64.StdEncoding.EncodeToString(image),
			},
		})
	}
	blocks = append(blocks, anthropicContentBlock{Type: "text", Text: prompt})

	body, err := json.Marshal(anthropicRequest{
		Model:     c.model,
		MaxTokens: anthropicMaxTokens,
		Messages: []anthropicMessage{{
			Role:    "user",
			Content: blocks,
		}},
	})
	if err != nil {
		return "", fmt.Errorf("编码 Anthropic 请求失败: %w", err)
	}

	// One deadline shared by the caller's context and the internal ceiling: the
	// shorter of the two fires, so a cancelled worker and a stuck model are both
	// reported as the deadline they hit.
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	headers := map[string]string{"anthropic-version": anthropicVersion}
	if c.apiKey != "" {
		headers["x-api-key"] = c.apiKey
	}

	// No stream field: the API defaults to non-streaming, which is all a
	// recognition worker needs.
	data, err := postVisionJSON(ctx, c.http, c.baseURL+"/v1/messages", body, headers, "Anthropic")
	if err != nil {
		return "", err
	}

	var reply anthropicResponse
	if err := json.Unmarshal(data, &reply); err != nil {
		return "", fmt.Errorf("解析 Anthropic 响应失败: %w, 响应: %s", err, bodySnippet(data))
	}
	// An answer is the concatenation of its text blocks and nothing else: a
	// tool_use block's fields must not leak into the review payload.
	var sb strings.Builder
	for _, block := range reply.Content {
		if block.Type == "text" {
			sb.WriteString(block.Text)
		}
	}
	// A model that answers nothing is a failure, not a run with zero tags: the
	// review queue would show a finished file with nothing to review. An empty
	// content array or only non-text blocks are the same shape.
	if sb.Len() == 0 {
		return "", errors.New("模型返回空内容")
	}
	return sb.String(), nil
}

// The structs below are the wire format of /v1/messages and stay unexported:
// nothing outside this file speaks it. The omitempty tags keep a text block
// from carrying "source": null and an image block from carrying "text": "".
type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	Messages  []anthropicMessage `json:"messages"`
}

type anthropicMessage struct {
	Role    string                  `json:"role"`
	Content []anthropicContentBlock `json:"content"`
}

type anthropicContentBlock struct {
	Type   string                `json:"type"`
	Source *anthropicImageSource `json:"source,omitempty"`
	Text   string                `json:"text,omitempty"`
}

type anthropicImageSource struct {
	Type      string `json:"type"`
	MediaType string `json:"media_type"`
	Data      string `json:"data"`
}

type anthropicResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
}
