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

// openAIClient implements VisionClient against one OpenAI-compatible chat
// completions deployment (openai.com, and the local servers that speak its API).
type openAIClient struct {
	baseURL string
	model   string
	apiKey  string
	// timeout is the ceiling of one call, visionTimeout by default. It is a
	// field rather than a direct read of the constant so the deadline path can be
	// exercised quickly instead of waiting three minutes for the same code.
	timeout time.Duration
	http    *http.Client
	// refineMode is the two-stage refine pipeline switch captured at
	// construction. It decides whether a request that carries images sends
	// response_format; see Recognize.
	refineMode bool
}

// NewOpenAIClient returns the vision client for one OpenAI-compatible server.
// baseURL is the configured root already carrying its version prefix
// ("https://api.openai.com/v1"), with or without a trailing slash; the client
// only appends /chat/completions. Neither value is read from config here: the
// caller passes all three in, so a task keeps using the settings that were in
// force when it was submitted even if the user edits them while it runs. An
// empty apiKey sends no Authorization header at all, which is what local servers
// that need no auth expect. refineMode (refineEnabled) gates the response_format
// rule in Recognize; with it false every body is exactly what the client sent
// before the refine pipeline existed.
func NewOpenAIClient(baseURL, model, apiKey string, refineMode bool) VisionClient {
	return &openAIClient{
		baseURL:    strings.TrimRight(baseURL, "/"),
		model:      model,
		apiKey:     apiKey,
		timeout:    visionTimeout,
		http:       &http.Client{Timeout: visionTimeout},
		refineMode: refineMode,
	}
}

// openAIMaxTokens caps one answer. Without it the server applies its own
// default (often a few hundred tokens), which truncates a long tag JSON
// mid-array and fails the parse; 4096 matches the Anthropic ceiling.
const openAIMaxTokens = 4096

// Recognize posts one chat completion carrying the prompt and every image as a
// base64 data URI whose MIME type is sniffed from the bytes, and returns the
// first choice's message content exactly as the model wrote it: no trimming, no
// parsing, no validation. Deciding what that text means is the review gate's
// job, which is why the transport and the answer schema stay testable apart.
func (c *openAIClient) Recognize(ctx context.Context, prompt string, images [][]byte) (string, error) {
	if c.model == "" {
		return "", errors.New("未配置 OpenAI 模型名")
	}
	if c.baseURL == "" {
		return "", errors.New("未配置 OpenAI 服务地址")
	}

	// content is always an array of parts, even with no frames: a nil part slice
	// would marshal as "content": null. The text part comes first and the images
	// follow in frame order.
	parts := make([]openAIContentPart, 0, len(images)+1)
	parts = append(parts, openAIContentPart{Type: "text", Text: prompt})
	for _, image := range images {
		// The MIME type is sniffed per image (http.DetectContentType reads the
		// magic bytes), because a data URI without the right prefix is what JPEG
		// and PNG frames arriving in one request would otherwise be punished for.
		parts = append(parts, openAIContentPart{
			Type: "image_url",
			ImageURL: &openAIImageURL{
				URL: "data:" + http.DetectContentType(image) + ";base64," + base64.StdEncoding.EncodeToString(image),
			},
		})
	}

	request := openAIChatRequest{
		Model: c.model,
		Messages: []openAIChatMessage{{
			Role:    "user",
			Content: parts,
		}},
		Stream:    false,
		MaxTokens: openAIMaxTokens,
	}
	// response_format rule: off mode always sends json_object (every body stays
	// byte-identical to the pre-refine client), refine mode sends it only for a
	// text-only call — a refine-mode request that carries frames leaves the key
	// out entirely.
	if !c.refineMode || len(images) == 0 {
		request.ResponseFormat = &openAIResponseFormat{Type: "json_object"}
	}

	body, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("编码 OpenAI 请求失败: %w", err)
	}

	// One deadline shared by the caller's context and the internal ceiling: the
	// shorter of the two fires, so a cancelled worker and a stuck model are both
	// reported as the deadline they hit.
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	headers := map[string]string{}
	if c.apiKey != "" {
		headers["Authorization"] = "Bearer " + c.apiKey
	}

	data, err := postVisionJSON(ctx, c.http, c.baseURL+"/chat/completions", body, headers, "OpenAI")
	if err != nil {
		return "", err
	}

	var reply openAIChatResponse
	if err := json.Unmarshal(data, &reply); err != nil {
		return "", fmt.Errorf("解析 OpenAI 响应失败: %w, 响应: %s", err, bodySnippet(data))
	}
	// A model that answers nothing is a failure, not a run with zero tags: the
	// review queue would show a finished file with nothing to review. A missing
	// choices array is the same shape seen from the envelope.
	if len(reply.Choices) == 0 || reply.Choices[0].Message.Content == "" {
		return "", errors.New("模型返回空内容")
	}
	return reply.Choices[0].Message.Content, nil
}

// The structs below are the wire format of /chat/completions and stay
// unexported: nothing outside this file speaks it. The omitempty tags keep a
// text part from carrying "image_url": null and an image part from carrying
// "text": "".
type openAIChatRequest struct {
	Model    string              `json:"model"`
	Messages []openAIChatMessage `json:"messages"`
	Stream   bool                `json:"stream"`
	// ResponseFormat is a pointer so refine mode can leave the key out of the
	// body: omitempty only fires on nil, and a value field would always send
	// "response_format". When it is set the encoding is identical to the old
	// value field, key position included.
	ResponseFormat *openAIResponseFormat `json:"response_format,omitempty"`
	MaxTokens      int                   `json:"max_tokens"`
}

type openAIResponseFormat struct {
	Type string `json:"type"`
}

type openAIChatMessage struct {
	Role    string              `json:"role"`
	Content []openAIContentPart `json:"content"`
}

type openAIContentPart struct {
	Type     string          `json:"type"`
	Text     string          `json:"text,omitempty"`
	ImageURL *openAIImageURL `json:"image_url,omitempty"`
}

type openAIImageURL struct {
	URL string `json:"url"`
}

type openAIChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}
