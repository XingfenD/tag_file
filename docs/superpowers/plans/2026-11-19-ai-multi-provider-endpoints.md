# AI Multi-Provider Inference Endpoints Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let the vision pipeline talk to three selectable endpoint formats — Ollama native (default, unchanged), OpenAI-compatible chat completions, and Anthropic messages — configured on the settings page with per-provider URL / model / API-key fields.

**Architecture:** Hand-written stdlib HTTP clients (no SDK, no new dependency) implement the unchanged `VisionClient` interface and share one HTTP plumbing helper extracted from `ollama.go`. A per-task factory dispatches on `Settings.Provider`; the config block registry drives the settings page entirely through the already-wired `ShowIf` / `oneof` / `secret` protocol, so **zero frontend code changes**.

**Tech Stack:** Go 1.25 stdlib (`net/http`, `encoding/json`, `httptest`), existing `internal/service/config/schema` block registry, GORM/SQLite.

**Spec:** `docs/superpowers/specs/2026-11-19-ai-multi-provider-endpoints-design.md` (feature design, extends `2026-11-13-ollama-vision-tagging-design.md`; «Pinned Decisions» below is its implementation-side projection). The config protocol it builds on: `docs/superpowers/specs/2026-11-17-config-inputbox-descriptor-design.md` (ShowIf §3, secret §8, oneof/规则协议 §4, API 契约 §9), anticipated by `docs/superpowers/specs/2026-11-16-config-schema-design.md:77,115,176` (`ai_provider` as the first enum + ShowIf user).

**Prerequisite:** Execute AFTER `docs/superpowers/plans/2026-11-19-ai-per-frame-scene-coverage.md` is merged. Baseline: core `0.9.0`, webui `5.12.0`. Branch: `feat/ai-multi-provider-endpoints` (create from the merged per-frame branch).

## Global Constraints

- Module `github.com/XingfenD/tag_file`, Go 1.25.0. **No new dependencies** — clients are hand-written stdlib HTTP.
- `VisionClient` interface unchanged: `Recognize(ctx context.Context, prompt string, images [][]byte) (string, error)` (`internal/service/ai/ports.go:13`).
- Clients are built **per task** (`NewService` leaves `Client` nil); editing provider/URL/model/key takes effect on the next file without a restart.
- `Settings()` never fails: missing key / read error / unusable value → logged default, per key independently.
- Existing stored key names `ollama_url` / `ollama_model` are **kept** (no rename, no data migration); they only gain `ShowIf`.
- Client error conventions (mirror `ollama.go`): Chinese text naming the provider; empty model/URL → `未配置 X 模型名` / `未配置 X 服务地址` **before any network contact**; empty answer → `模型返回空内容`; non-2xx → status + first 200 runes of body; undecodable 200 → parse error quoting the body, never `空内容`; caller deadline stays attributable via `errors.Is(err, context.DeadlineExceeded)`.
- Transport budgets shared by all three clients: 180s timeout, 4 MiB response body limit, 200-rune error snippet cut on a rune boundary.
- Tests: stdlib `testing`, external `package ai_test` (in-package `package ai` only for unexported symbols, precedent `parse_test.go`), `httptest` for HTTP, no testify, no `t.Parallel`, no ad-hoc test binaries. Run via `go test` / `make test`.
- Never commit on `master` (AGENTS.md). Branch `feat/ai-multi-provider-endpoints`.
- CHANGELOG: this plan touches **no file under `cmd/webui/`** → `docs/CHANGELOG.md` only, `CoreVersion` `0.9.0` → `0.10.0` in the same commit. No `WebuiVersion` bump. Entry format: English line immediately followed by Chinese line (no blank line between them), blank line between entries, higher versions on top.
- Settings page: schema-driven. `saveGroup` submits **all** block fields including hidden ones (`cmd/webui/static/js/pages/config/main.js:387-401`); hidden provider fields keep their seeded defaults, which are all valid, so group save never deadlocks. No `SetGroup` change.

## Pinned Decisions

1. **Provider enum:** `ai_provider` ∈ {`ollama` (default), `openai`, `anthropic`}. Write side: `oneof` validation rejects anything else. Read side: unknown/empty stored value → logged fallback to `ollama` (garbage in the DB must never crash or dial a wrong endpoint).
2. **New keys (7):** `ai_provider`, `ai_openai_url`, `ai_openai_model`, `ai_openai_api_key`, `ai_anthropic_url`, `ai_anthropic_model`, `ai_anthropic_api_key`. Block `ai` grows 5 → 12 keys; bootstrap seeds the new keys automatically (registry walk, `internal/bootstrap/init.go:35-52`).
3. **URL semantics:** OpenAI base URL **includes `/v1`** (default `https://api.openai.com/v1`; vLLM/LM Studio/one-api follow the same shape), client appends `/chat/completions`. Anthropic base URL is the root (default `https://api.anthropic.com`), client appends `/v1/messages`. Both `strings.TrimRight(baseURL, "/")` only — no auto-appending of path prefixes.
4. **API keys are optional** (local OpenAI-compatible servers need none): `Authorization: Bearer <key>` sent **only when key non-empty**; `x-api-key: <key>` likewise; `anthropic-version: 2023-06-01` always. Keys are `Secret` descriptors → masked in the API, 留空不覆盖 in `SetGroup` (`internal/service/config/service.go:66-70`). Empty key is a legal stored value.
5. **Image MIME is sniffed per image** with `http.DetectContentType(image)`: OpenAI data-URI prefix `data:<mime>;base64,`, Anthropic `source.media_type`. Video frames are JPEG but uploaded image files may be PNG — a hardcoded `image/jpeg` would make Anthropic reject them. Ollama stays bare base64 (no MIME).
6. **OpenAI wire:** `stream: false`, `response_format: {"type":"json_object"}`, one user message whose `content` is an array of parts — text part first, then one `image_url` part per image in order. Answer: `choices[0].message.content` verbatim; `choices` empty or content empty → `模型返回空内容`.
7. **Anthropic wire:** `max_tokens: 4096` (constant; required by the API), one user message whose `content` is an array of blocks — **image blocks first, text block last**. No JSON mode exists; the prompt already demands JSON and `ParseSuggestions` tolerates fences/prose (`parse.go:47-58`). Answer: concatenate all `type == "text"` blocks in order, no separator; nothing left → `模型返回空内容`.
8. **Shared plumbing:** new `client.go` holds `visionTimeout` / `visionBodyLimit` / `visionBodySnippet` (renamed from `ollama*`), `bodySnippet` (moved verbatim), and `postVisionJSON`. `ollama.go` is refactored onto it with **byte-identical error texts** — the five existing ollama tests are the safety net and must pass unchanged.
9. **Dispatch order in `recognize.go`:** per-provider empty-model check **before** the `s.Client` seam (the existing `TestWorker_EmptyModelConfigFails` pins that a configured fake client is never called when the model is missing), then the factory switch. Default branch = ollama, which is also the garbage-provider fallback path.
10. **Versioning:** core `0.9.0` → `0.10.0`, `docs/CHANGELOG.md` only (see Global Constraints).
11. **Constructor signatures:** `NewOllamaClient(baseURL, model string) VisionClient` unchanged; `NewOpenAIClient(baseURL, model, apiKey string) VisionClient`; `NewAnthropicClient(baseURL, model, apiKey string) VisionClient`.

## Review Focus

Input classes the design implies but a naive implementation would break, most likely first — each is pinned by a test in the owning task:

1. **A PNG (or GIF/WebP) image file, not a JPEG** — OpenAI/Anthropic must receive the sniffed media type, or the API rejects the request. → Task 2 & 3 wire tests send one JPEG-magic image and one real 1×1 PNG and assert both MIME types.
2. **An OpenAI-compatible server without auth (vLLM, LM Studio, llama.cpp)** — an empty key must not send `Authorization: Bearer ` (empty bearer) or `x-api-key: `. → Task 2 `NoAuthHeaderWithoutKey`, Task 3 `NoAPIKeyHeaderWithoutKey`.
3. **Garbage `ai_provider` in the DB** (manual edit, future format) — recognition must fall back to ollama with a log line, not crash or dial the wrong endpoint. → Task 1 `TestAISettings_ProviderFallsBackOnUnknown`, Task 4 dispatch test leg 4.
4. **An Anthropic answer split across several content blocks, with non-text blocks interleaved** — only text blocks join, in order; a `tool_use` block's fields must not leak into the answer. → Task 3 `JoinsTextBlocks`.
5. **A 200 with no answer** (`choices: []`, `content: []`, only non-text blocks, empty string) — must fail as `模型返回空内容`, never look like a finished run with zero tags. → Task 2 & 3 error-shape tests.

## File Structure

| File | Responsibility |
|---|---|
| `internal/service/ai/config.go` (modify) | 7 new keys + defaults + provider constants; `Settings` gains 7 fields; `providerSetting` logged-fallback reader |
| `internal/service/ai/config_schema.go` (modify) | Block `ai` registers 12 descriptors: provider enum first, per-provider trios with `ShowIf`, frame keys last |
| `internal/service/ai/client.go` (create) | Shared transport: `visionTimeout`/`visionBodyLimit`/`visionBodySnippet`, `bodySnippet`, `postVisionJSON` |
| `internal/service/ai/openai.go` (create) | OpenAI-compatible client: wire structs, content parts, `choices[0]` extraction |
| `internal/service/ai/anthropic.go` (create) | Anthropic client: wire structs, image-first blocks, text-block join |
| `internal/service/ai/ollama.go` (modify) | Refactor onto `postVisionJSON`; constants/`bodySnippet` move out; error texts unchanged |
| `internal/service/ai/provider.go` (create) | `providerLabel`, `checkProviderModel`, `newVisionClient` factory |
| `internal/service/ai/recognize.go` (modify) | Replace the ollama-only check+build block (:88-100) with provider check + factory |
| `internal/service/ai/parse.go` (modify) | One comment reference `ollamaBodySnippet` → `visionBodySnippet` (:22) |
| Tests | `config_schema_test.go`, `config_test.go` (extend); `openai_test.go`, `anthropic_test.go`, `client_internal_test.go` (create); `worker_test.go` (extend); `internal/bootstrap/init_test.go` (adapt literal counts) |
| `internal/constant/version.go`, `docs/CHANGELOG.md` (modify) | Release 0.10.0 |

---

### Task 1: Config Keys, Settings and Schema Registration

**Files:**
- Modify: `internal/service/ai/config.go:11-29,74-94`
- Modify: `internal/service/ai/config_schema.go` (whole `init`)
- Test: `internal/service/ai/config_schema_test.go`, `internal/service/ai/config_test.go`
- Adapt: `internal/bootstrap/init_test.go:17-31,136-146,172-182,195-202,389`

**Interfaces:**
- Consumes: `schema.RegisterBlock`, `schema.InputBoxDescriptor`, `schema.StringConstraintDescriptor{Required, Format, Options, Secret}`, `schema.Condition{Key, Equals}`, `schema.Option{Value, Label}` (all existing, `internal/service/config/schema/inputbox.go`).
- Produces (Tasks 2–4 rely on these exact names):
  - Keys: `KeyProvider = "ai_provider"`, `KeyOpenAIURL = "ai_openai_url"`, `KeyOpenAIModel = "ai_openai_model"`, `KeyOpenAIAPIKey = "ai_openai_api_key"`, `KeyAnthropicURL = "ai_anthropic_url"`, `KeyAnthropicModel = "ai_anthropic_model"`, `KeyAnthropicAPIKey = "ai_anthropic_api_key"`.
  - Providers: `ProviderOllama = "ollama"`, `ProviderOpenAI = "openai"`, `ProviderAnthropic = "anthropic"`.
  - Defaults: `DefaultProvider = ProviderOllama`, `DefaultOpenAIURL = "https://api.openai.com/v1"`, `DefaultOpenAIModel = ""`, `DefaultOpenAIAPIKey = ""`, `DefaultAnthropicURL = "https://api.anthropic.com"`, `DefaultAnthropicModel = ""`, `DefaultAnthropicAPIKey = ""`.
  - `Settings` struct gains `Provider, OpenAIURL, OpenAIModel, OpenAIAPIKey, AnthropicURL, AnthropicModel, AnthropicAPIKey string` (existing five fields unchanged).

- [ ] **Step 1: Write the failing schema tests**

In `config_schema_test.go`:

a) Extend the `TestRegisterAIFields` table with 7 rows (all `schema.ValueString`):

```go
{ai.KeyProvider, schema.ValueString, ai.DefaultProvider},
{ai.KeyOpenAIURL, schema.ValueString, ai.DefaultOpenAIURL},
{ai.KeyOpenAIModel, schema.ValueString, ai.DefaultOpenAIModel},
{ai.KeyOpenAIAPIKey, schema.ValueString, ai.DefaultOpenAIAPIKey},
{ai.KeyAnthropicURL, schema.ValueString, ai.DefaultAnthropicURL},
{ai.KeyAnthropicModel, schema.ValueString, ai.DefaultAnthropicModel},
{ai.KeyAnthropicAPIKey, schema.ValueString, ai.DefaultAnthropicAPIKey},
```

b) In `TestRegisterAIBlock`, grow the `want` map to the 12-key set (the 7 above plus the existing 5). The `len` assertion adapts automatically.

c) Replace `TestRegisterAIOllamaURLRules` with a table over all three URL keys (same `required`+`url` assertion body, renamed `TestRegisterAIProviderURLRules`):

```go
for _, key := range []string{ai.KeyOllamaURL, ai.KeyOpenAIURL, ai.KeyAnthropicURL} {
	t.Run(key, func(t *testing.T) { /* existing required+url body on lookupInputBox(t, key) */ })
}
```

d) Replace `TestRegisterAIFieldsOllamaModelEmptyIsValid` with `TestRegisterAIModelEmptyIsValid`, same body (Default `""`, zero rules, `Validate(*d, "")` nil) looped over `ai.KeyOllamaModel, ai.KeyOpenAIModel, ai.KeyAnthropicModel`.

e) New tests:

```go
func TestRegisterAIProviderEnum(t *testing.T) {
	d := lookupInputBox(t, ai.KeyProvider)

	rs := schema.RulesOf(*d)
	hasOneOf := false
	for _, r := range rs.Rules {
		if r.Kind == schema.RuleOneOf {
			hasOneOf = true
		}
	}
	if !hasOneOf {
		t.Fatal("ai_provider 指令组缺少 oneof（设置页不会渲染下拉）")
	}
	for _, v := range []string{ai.ProviderOllama, ai.ProviderOpenAI, ai.ProviderAnthropic} {
		if err := schema.Validate(*d, v); err != nil {
			t.Errorf("Validate(%q) = %v, want nil", v, err)
		}
	}
	if err := schema.Validate(*d, "bogus"); err == nil {
		t.Error(`Validate("bogus") = nil, want oneof 拒绝写入`)
	}
}

func TestRegisterAIProviderShowIf(t *testing.T) {
	cases := []struct{ key, equals string }{
		{ai.KeyOllamaURL, ai.ProviderOllama},
		{ai.KeyOllamaModel, ai.ProviderOllama},
		{ai.KeyOpenAIURL, ai.ProviderOpenAI},
		{ai.KeyOpenAIModel, ai.ProviderOpenAI},
		{ai.KeyOpenAIAPIKey, ai.ProviderOpenAI},
		{ai.KeyAnthropicURL, ai.ProviderAnthropic},
		{ai.KeyAnthropicModel, ai.ProviderAnthropic},
		{ai.KeyAnthropicAPIKey, ai.ProviderAnthropic},
	}
	for _, tc := range cases {
		d := lookupInputBox(t, tc.key)
		if d.ShowIf == nil {
			t.Fatalf("%s 缺少 ShowIf", tc.key)
		}
		if d.ShowIf.Key != ai.KeyProvider || d.ShowIf.Equals != tc.equals {
			t.Errorf("%s ShowIf = %+v, want {ai_provider, %s}", tc.key, d.ShowIf, tc.equals)
		}
	}
	for _, key := range []string{ai.KeyProvider, ai.KeyFrameInterval, ai.KeyDiffThreshold, ai.KeyMaxFrames} {
		if d := lookupInputBox(t, key); d.ShowIf != nil {
			t.Errorf("%s 不应有 ShowIf, got %+v", key, d.ShowIf)
		}
	}
}

func TestRegisterAIAPIKeysAreSecret(t *testing.T) {
	for _, key := range []string{ai.KeyOpenAIAPIKey, ai.KeyAnthropicAPIKey} {
		d := lookupInputBox(t, key)
		if !d.IsSecret() {
			t.Errorf("%s 应为 secret（掩码 + 留空不覆盖）", key)
		}
		if err := schema.Validate(*d, ""); err != nil {
			t.Errorf("%s 空串应合法（未配置）: %v", key, err)
		}
		if err := schema.Validate(*d, "sk-test"); err != nil {
			t.Errorf("%s 非空密钥应合法: %v", key, err)
		}
	}
	for _, key := range []string{ai.KeyOllamaModel, ai.KeyOpenAIModel, ai.KeyAnthropicModel} {
		if d := lookupInputBox(t, key); d.IsSecret() {
			t.Errorf("%s 是模型名, 不应是 secret", key)
		}
	}
}
```

- [ ] **Step 2: Write the failing Settings tests**

In `config_test.go`:

a) Extend `TestAISettings_DefaultsWhenUnset` with:

```go
if got.Provider != ai.ProviderOllama {
	t.Fatalf("Provider = %q, want default ollama", got.Provider)
}
if got.OpenAIURL != "https://api.openai.com/v1" {
	t.Fatalf("OpenAIURL = %q, want default https://api.openai.com/v1", got.OpenAIURL)
}
if got.AnthropicURL != "https://api.anthropic.com" {
	t.Fatalf("AnthropicURL = %q, want default https://api.anthropic.com", got.AnthropicURL)
}
if got.OpenAIModel != "" || got.AnthropicModel != "" || got.OpenAIAPIKey != "" || got.AnthropicAPIKey != "" {
	t.Fatalf("未配置的模型/密钥应为空串, got %+v", got)
}
```

b) New tests (`seedRaw` already lives in this file):

```go
// 读侧对非法 ai_provider 的容忍与 intSetting 同形：回落 ollama，绝不 panic。
// "Ollama"/"OPENAI" 大小写不匹配也算非法——枚举值是封闭集合，不做归一化。
func TestAISettings_ProviderFallsBackOnUnknown(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	for _, raw := range []string{"bogus", "", "Ollama", "OPENAI"} {
		seedRaw(t, ai.KeyProvider, raw)
		if got := ai.NewService().Settings().Provider; got != ai.ProviderOllama {
			t.Fatalf("Provider for wire %q = %q, want fallback ollama", raw, got)
		}
	}

	// 合法值原样读出；经 Set 写入证明 oneof 注册表接线成立。
	cfg := configSvc.NewService()
	for _, p := range []string{ai.ProviderOpenAI, ai.ProviderAnthropic, ai.ProviderOllama} {
		if err := cfg.Set(ai.KeyProvider, p); err != nil {
			t.Fatalf("set ai_provider=%s: %v", p, err)
		}
		if got := ai.NewService().Settings().Provider; got != p {
			t.Fatalf("Provider = %q, want %q", got, p)
		}
	}
}

func TestAISettings_ReadsProviderCredentials(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	cfg := configSvc.NewService()
	set := func(key, value string) {
		t.Helper()
		if err := cfg.Set(key, value); err != nil {
			t.Fatalf("set %s: %v", key, err)
		}
	}
	set(ai.KeyProvider, ai.ProviderOpenAI)
	set(ai.KeyOpenAIURL, "http://127.0.0.1:8000/v1")
	set(ai.KeyOpenAIModel, "qwen2.5-vl")
	set(ai.KeyOpenAIAPIKey, "sk-local")

	got := ai.NewService().Settings()
	if got.Provider != ai.ProviderOpenAI || got.OpenAIURL != "http://127.0.0.1:8000/v1" ||
		got.OpenAIModel != "qwen2.5-vl" || got.OpenAIAPIKey != "sk-local" {
		t.Fatalf("Settings 未读出 openai 凭据: %+v", got)
	}
}
```

- [ ] **Step 3: Run the tests to verify they fail**

Run: `go test ./internal/service/ai/ -run 'TestRegisterAI|TestAISettings' -v`
Expected: compile failure — `undefined: ai.KeyProvider`, `ai.Settings` has no field `Provider`.

- [ ] **Step 4: Implement the config declarations**

In `config.go`: add the 7 key constants to the key block, the provider constants and 7 defaults to the default block (values exactly as in Interfaces; update the block comment — an empty model of the **active provider** means "not configured"). Extend `Settings` with the 7 string fields. In `Settings()` read them via `stringSetting` (keys/defaults from Interfaces), and read the provider through a new helper:

```go
// providerSetting 读 ai_provider：封闭枚举，未知值（含空串、大小写漂移）
// 与 intSetting 的非法数同形——记一行日志并回落 ollama，绝不把非法值
// 传给分派开关。
func (s *Service) providerSetting() string {
	raw := s.stringSetting(KeyProvider, DefaultProvider)
	switch raw {
	case ProviderOllama, ProviderOpenAI, ProviderAnthropic:
		return raw
	}
	log.Printf("[ai] config %s = %q, fallback to %s", KeyProvider, raw, DefaultProvider)
	return DefaultProvider
}
```

In `config_schema.go`: rewrite the `init` descriptor list to 12 entries in this display order — `ai_provider`, `ollama_model`, `ollama_url`, `ai_openai_url`, `ai_openai_model`, `ai_openai_api_key`, `ai_anthropic_url`, `ai_anthropic_model`, `ai_anthropic_api_key`, `ai_frame_interval`, `ai_frame_diff_threshold`, `ai_max_frames`. The three frame keys keep their current descriptors byte-for-byte. The new/changed ones (declaration data is the spec — copy verbatim):

```go
{
	Key:         KeyProvider,
	Label:       "推理端点类型",
	Description: "视觉模型推理服务的接口格式",
	ValueType:   schema.ValueString,
	Container:   schema.ContainerSingle,
	Default:     DefaultProvider,
	Constraint: schema.InputBoxConstraintDescriptor{
		StringConstraint: &schema.StringConstraintDescriptor{
			Options: []schema.Option{
				{Value: ProviderOllama, Label: "Ollama"},
				{Value: ProviderOpenAI, Label: "OpenAI 兼容"},
				{Value: ProviderAnthropic, Label: "Anthropic"},
			},
		},
	},
},
```

The two existing ollama descriptors keep every field and gain `ShowIf: &schema.Condition{Key: KeyProvider, Equals: ProviderOllama}`. The six new provider fields follow this shape (substitute key/label/description/default/equals per row):

```go
{
	Key:         KeyOpenAIURL,
	Label:       "OpenAI 服务地址",
	Description: "OpenAI 兼容服务地址，须包含 /v1 前缀（如 https://api.openai.com/v1）",
	ShowIf:      &schema.Condition{Key: KeyProvider, Equals: ProviderOpenAI},
	ValueType:   schema.ValueString,
	Container:   schema.ContainerSingle,
	Default:     DefaultOpenAIURL,
	Constraint: schema.InputBoxConstraintDescriptor{
		StringConstraint: &schema.StringConstraintDescriptor{Required: true, Format: schema.FormatURL},
	},
},
```

| Key | Label | Description | Constraint |
|---|---|---|---|
| `ai_openai_url` | OpenAI 服务地址 | OpenAI 兼容服务地址，须包含 /v1 前缀（如 https://api.openai.com/v1） | Required + FormatURL |
| `ai_openai_model` | OpenAI 视觉模型 | OpenAI 兼容视觉模型名称（空表示未配置，识别时报错） | none |
| `ai_openai_api_key` | OpenAI API Key | OpenAI 兼容服务 API 密钥（本地服务可留空） | Secret |
| `ai_anthropic_url` | Anthropic 服务地址 | Anthropic 服务地址（如 https://api.anthropic.com） | Required + FormatURL |
| `ai_anthropic_model` | Anthropic 视觉模型 | Anthropic 视觉模型名称（空表示未配置，识别时报错） | none |
| `ai_anthropic_api_key` | Anthropic API Key | Anthropic API 密钥（留空表示未配置） | Secret |

All six carry `ShowIf` on `KeyProvider` with their provider's value; models/keys default `""`.

- [ ] **Step 5: Run the ai tests — pass; bootstrap tests — expected fail**

Run: `go test ./internal/service/ai/ -run 'TestRegisterAI|TestAISettings' -v` → PASS.
Run: `go test ./internal/bootstrap/ -v` → FAIL: the literal `6` row counts and key maps in `init_test.go` predate the 7 new keys. This failure is the adaptation signal, not a bug.

- [ ] **Step 6: Adapt the bootstrap seed tests**

In `internal/bootstrap/init_test.go`:
- Add to the literal const block (values pinned, same drift-guard rationale as the existing ones):

```go
keyProvider        = "ai_provider"
keyOpenAIURL       = "ai_openai_url"
keyOpenAIModel     = "ai_openai_model"
keyOpenAIAPIKey    = "ai_openai_api_key"
keyAnthropicURL    = "ai_anthropic_url"
keyAnthropicModel  = "ai_anthropic_model"
keyAnthropicAPIKey = "ai_anthropic_api_key"

defaultProviderValue        = "ollama"
defaultOpenAIURLValue       = "https://api.openai.com/v1"
defaultOpenAIModelValue     = ""
defaultOpenAIAPIKeyValue    = ""
defaultAnthropicURLValue    = "https://api.anthropic.com"
defaultAnthropicModelValue  = ""
defaultAnthropicAPIKeyValue = ""
```

- Change all three `len(rows) != 6` assertions (lines 136, 172, 195) to `13`, message text `want 13`.
- Extend both `requireRows` maps (fresh-database and upgrade tests) with the 7 new key→default pairs.
- Extend the per-key count list in `TestSeedDefaultConfig_RerunSeedsNothingNew` (lines 199-202) with the 7 new key constants.
- In the `TestSeedCoversNewlyRegisteredField` comment (line 389), replace "would seed a seventh row" with "would seed an extra row" (the literal count is now 13).

Run: `go test ./internal/bootstrap/ ./internal/service/ai/ -v`
Expected: PASS (the whole ai package, including the unchanged ollama/worker tests).

- [ ] **Step 7: Commit**

```bash
git add internal/service/ai/config.go internal/service/ai/config_schema.go \
  internal/service/ai/config_schema_test.go internal/service/ai/config_test.go \
  internal/bootstrap/init_test.go
git commit -m "feat(ai): provider enum + per-provider endpoint config keys (ollama/openai/anthropic)"
```

---

### Task 2: OpenAI-Compatible Client + Shared HTTP Plumbing

**Files:**
- Create: `internal/service/ai/client.go`, `internal/service/ai/openai.go`
- Create: `internal/service/ai/openai_test.go`, `internal/service/ai/client_internal_test.go`
- Modify: `internal/service/ai/ollama.go` (constants + `bodySnippet` move out; HTTP block → `postVisionJSON`)
- Modify: `internal/service/ai/parse.go:22` (comment reference rename only)

**Interfaces:**
- Consumes: `VisionClient` (`ports.go:13`); nothing from Task 1 (constructors take plain strings).
- Produces:
  - `NewOpenAIClient(baseURL, model, apiKey string) VisionClient`
  - `postVisionJSON(ctx context.Context, httpClient *http.Client, url string, body []byte, headers map[string]string, provider string) ([]byte, error)` — POST with `Content-Type: application/json` plus `headers`, 2xx body (≤ `visionBodyLimit`) or a provider-named error.
  - `visionTimeout = 180 * time.Second`, `visionBodyLimit = 4 << 20`, `visionBodySnippet = 200`, `bodySnippet(body []byte) string` (all in `client.go`; Task 3 uses them).

- [ ] **Step 1: Write the failing tests**

Create `openai_test.go` (`package ai_test`). The request mirror — `Stream *bool` distinguishes `"stream": false` from a missing key, exactly like `chatRequest` in `ollama_test.go:22-33`:

```go
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
```

The contract test (full code — this is the new wire contract):

```go
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
```

Remaining tests — each mirrors the named ollama test (`ollama_test.go`) with the same scaffolding (httptest capture, the `release chan` + `sync.Once` + deferred-release pattern for the silent server, `closedLocalPort` is NOT needed here); only the deltas are pinned:

- `TestOpenAIClient_NoAuthHeaderWithoutKey` — `ai.NewOpenAIClient(srv.URL, "m", "")` against a server answering `{"choices":[{"message":{"content":"{}"}}]}`; assert `r.Header.Get("Authorization") == ""` (an empty bearer would be rejected by local servers that need no auth).
- `TestOpenAIClient_HTTPErrorAndEmptyContent` — mirror `TestOllamaClient_HTTPErrorAndEmptyContent` (500+"boom" both in error; 502 with `strings.Repeat("x",250)+"TAIL"` → first 200 chars present, `TAIL` absent). Empty-answer servers: `{"choices":[{"message":{"content":""}}]}` **and** `{"choices":[]}` → both errors contain `空内容`.
- `TestOpenAIClient_MissingModelAndTimeout` — mirror `TestOllamaClient_MissingModelAndTimeout`: empty model → error contains `未配置` and `OpenAI 模型名`; empty URL → `未配置` and `OpenAI 服务地址`; `hits == 0`; 20ms caller deadline against the silent server → `errors.Is(err, context.DeadlineExceeded)`.
- `TestOpenAIClient_UndecodableSuccessBody` — mirror `TestOllamaClient_UndecodableSuccessBody`: non-JSON 200 → error contains `解析`, NOT `空内容`, and quotes `not json at all`.
- `TestOpenAIClient_EmptyImageListSendsTextOnly` — mirror `TestOllamaClient_EmptyImageListSendsAnArray` with `nil` images: `content` decodes as a non-nil array of exactly 1 part, `parts[0].Type == "text"` (a nil part slice would marshal as `null`).

Create `client_internal_test.go` (`package ai`, precedent `parse_test.go` — `bodySnippet` is unexported):

```go
package ai

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// bodySnippet 从 ollama.go 移入 client.go 时唯一未被现有测试钉住的行为：
// 中文错误页按 rune 截断，不能切在字节中间。
func TestBodySnippetCutsOnRuneBoundary(t *testing.T) {
	got := bodySnippet([]byte(strings.Repeat("模", visionBodySnippet+50)))
	runes := []rune(got)
	if len(runes) != visionBodySnippet+1 {
		t.Fatalf("snippet = %d runes, want %d (200 + 省略号)", len(runes), visionBodySnippet+1)
	}
	if !utf8.ValidString(got) {
		t.Fatal("snippet 不是合法 UTF-8: 切在了字节中间")
	}
	if !strings.HasSuffix(got, "…") {
		t.Errorf("snippet 应以 … 结尾, got %q", got)
	}
	if got := bodySnippet([]byte("  hi  ")); got != "hi" {
		t.Errorf("bodySnippet 短 body = %q, want trimmed %q", got, "hi")
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/service/ai/ -run 'TestOpenAIClient|TestBodySnippet' -v`
Expected: compile failure — `undefined: ai.NewOpenAIClient`, `undefined: visionBodySnippet`.

- [ ] **Step 3: Implement `client.go`, `openai.go`, and refactor `ollama.go`**

`client.go` — move the three constants out of `ollama.go` renamed (`visionTimeout`, `visionBodyLimit`, `visionBodySnippet`, same values/comments generalized to "the vision service"), move `bodySnippet` verbatim (it reads `visionBodySnippet`), and add the shared round trip. Error texts are the contract — the ollama tests assert substrings of them:

```go
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
```

`openai.go` — struct `openAIClient{baseURL, model, apiKey string; timeout time.Duration; http *http.Client}` mirroring `ollamaClient` (the `timeout` field exists for the same reason as ollama's). Constructor:

```go
func NewOpenAIClient(baseURL, model, apiKey string) VisionClient {
	return &openAIClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		model:   model,
		apiKey:  apiKey,
		timeout: visionTimeout,
		http:    &http.Client{Timeout: visionTimeout},
	}
}
```

Wire structs (unexported, `omitempty` keeps text parts from carrying `"image_url":null` and vice versa):

```go
type openAIChatRequest struct {
	Model          string               `json:"model"`
	Messages       []openAIChatMessage  `json:"messages"`
	Stream         bool                 `json:"stream"`
	ResponseFormat openAIResponseFormat `json:"response_format"`
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
```

`Recognize` body, in order (exact error strings pinned):
1. `c.model == ""` → `errors.New("未配置 OpenAI 模型名")`; `c.baseURL == ""` → `errors.New("未配置 OpenAI 服务地址")`.
2. Build parts: `make([]openAIContentPart, 0, len(images)+1)`, text part `{Type: "text", Text: prompt}` first; per image append `{Type: "image_url", ImageURL: &openAIImageURL{URL: "data:" + http.DetectContentType(image) + ";base64," + base64.StdEncoding.EncodeToString(image)}}`.
3. Marshal `openAIChatRequest{Model, Messages: one user message, Stream: false, ResponseFormat: {Type: "json_object"}}`; failure → `fmt.Errorf("编码 OpenAI 请求失败: %w", err)`.
4. `ctx, cancel := context.WithTimeout(ctx, c.timeout); defer cancel()`.
5. `headers := map[string]string{}`; if `c.apiKey != ""` set `headers["Authorization"] = "Bearer " + c.apiKey`.
6. `data, err := postVisionJSON(ctx, c.http, c.baseURL+"/chat/completions", body, headers, "OpenAI")`; error → return it unchanged.
7. Unmarshal into `openAIChatResponse`; failure → `fmt.Errorf("解析 OpenAI 响应失败: %w, 响应: %s", err, bodySnippet(data))`.
8. `len(reply.Choices) == 0 || reply.Choices[0].Message.Content == ""` → `errors.New("模型返回空内容")`; else return the content verbatim.

`ollama.go` refactor — delete the three constants and `bodySnippet` (now in `client.go`); in `Recognize` replace everything from `http.NewRequestWithContext` through the status-code check (current lines 95-113) with:

```go
data, err := postVisionJSON(ctx, c.http, c.baseURL+"/api/chat", body, nil, "Ollama")
if err != nil {
	return "", err
}
```

Keep the empty-config checks, the marshal (`编码 Ollama 请求失败`), the `context.WithTimeout` wrap, the unmarshal (`解析 Ollama 响应失败: %w, 响应: %s`) and the empty-content check exactly as they are. In `parse.go:22` update the comment reference `ollamaBodySnippet` → `visionBodySnippet`.

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./internal/service/ai/ -v`
Expected: PASS — the new OpenAI tests, the rune-boundary test, **and all five existing ollama tests unchanged** (the refactor's safety net), plus the rest of the package.

- [ ] **Step 5: Commit**

```bash
git add internal/service/ai/client.go internal/service/ai/openai.go \
  internal/service/ai/openai_test.go internal/service/ai/client_internal_test.go \
  internal/service/ai/ollama.go internal/service/ai/parse.go
git commit -m "feat(ai): hand-written OpenAI-compatible vision client on shared HTTP plumbing"
```

---

### Task 3: Anthropic Client

**Files:**
- Create: `internal/service/ai/anthropic.go`, `internal/service/ai/anthropic_test.go`

**Interfaces:**
- Consumes: `postVisionJSON`, `visionTimeout`, `bodySnippet` (Task 2, `client.go`); `VisionClient`.
- Produces: `NewAnthropicClient(baseURL, model, apiKey string) VisionClient`; constants `anthropicMaxTokens = 4096`, `anthropicVersion = "2023-06-01"` (file-local).

- [ ] **Step 1: Write the failing tests**

Create `anthropic_test.go` (`package ai_test`; `jpegBytes`/`pngBytes` from `openai_test.go` are same-package and reusable). Request mirror:

```go
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
```

The contract test (full code):

```go
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
```

Remaining tests (mirror the named ollama tests for scaffolding; deltas pinned):

- `TestAnthropicClient_JoinsTextBlocks` — server answers `{"content":[{"type":"text","text":"{\"tags\":"},{"type":"tool_use","text":"IGNORED"},{"type":"text","text":"[]}"}]}`; `Recognize` returns exactly `{"tags":[]}` (text blocks joined in order, non-text block's fields never leak).
- `TestAnthropicClient_NoAPIKeyHeaderWithoutKey` — `NewAnthropicClient(srv.URL, "m", "")`; assert `x-api-key` header **absent** (`== ""`) while `anthropic-version` is still `2023-06-01`.
- `TestAnthropicClient_HTTPErrorAndEmptyContent` — mirror the ollama test (500+"boom"; 502 long-body 200-rune truncation, no `TAIL`). Empty-answer servers: `{"content":[]}` **and** `{"content":[{"type":"tool_use","text":"x"}]}` → both errors contain `空内容`.
- `TestAnthropicClient_MissingModelAndTimeout` — mirror: empty model → `未配置` + `Anthropic 模型名`; empty URL → `未配置` + `Anthropic 服务地址`; `hits == 0`; 20ms deadline → `errors.Is(err, context.DeadlineExceeded)`.
- `TestAnthropicClient_UndecodableSuccessBody` — mirror: non-JSON 200 → `解析`, not `空内容`, body quoted.
- `TestAnthropicClient_EmptyImageListSendsTextOnly` — `nil` images → exactly 1 content block, `type == "text"`, non-nil array.

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/service/ai/ -run TestAnthropicClient -v`
Expected: compile failure — `undefined: ai.NewAnthropicClient`.

- [ ] **Step 3: Implement `anthropic.go`**

Constants `anthropicMaxTokens = 4096` (one comment line: Anthropic requires `max_tokens`; 4096 covers the prompt's 1–6 tag JSON with room) and `anthropicVersion = "2023-06-01"`. Struct `anthropicClient{baseURL, model, apiKey string; timeout time.Duration; http *http.Client}`; constructor mirrors Task 2's (TrimRight `/`, `visionTimeout`).

Wire structs:

```go
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
```

`Recognize` body, in order:
1. Empty model/URL → `errors.New("未配置 Anthropic 模型名")` / `errors.New("未配置 Anthropic 服务地址")`.
2. Blocks: `make([]anthropicContentBlock, 0, len(images)+1)`; per image append `{Type: "image", Source: &anthropicImageSource{Type: "base64", MediaType: http.DetectContentType(image), Data: base64.StdEncoding.EncodeToString(image)}}`; **then** append `{Type: "text", Text: prompt}` last.
3. Marshal `anthropicRequest{Model, MaxTokens: anthropicMaxTokens, Messages: one user message}`; failure → `fmt.Errorf("编码 Anthropic 请求失败: %w", err)`.
4. `context.WithTimeout(ctx, c.timeout)` + defer cancel.
5. `headers := map[string]string{"anthropic-version": anthropicVersion}`; if `c.apiKey != ""` set `headers["x-api-key"] = c.apiKey`.
6. `postVisionJSON(ctx, c.http, c.baseURL+"/v1/messages", body, headers, "Anthropic")`; error → unchanged.
7. Unmarshal; failure → `fmt.Errorf("解析 Anthropic 响应失败: %w, 响应: %s", err, bodySnippet(data))`.
8. Join: `var sb strings.Builder`; for each block with `Type == "text"` write its `Text`; `sb.Len() == 0` → `errors.New("模型返回空内容")`; else return `sb.String()`.

No `stream` field (Anthropic defaults to non-streaming) and no JSON mode — `ParseSuggestions` already repairs fences/prose.

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./internal/service/ai/ -v`
Expected: PASS (whole package).

- [ ] **Step 5: Commit**

```bash
git add internal/service/ai/anthropic.go internal/service/ai/anthropic_test.go
git commit -m "feat(ai): hand-written Anthropic vision client"
```

---

### Task 4: Provider Dispatch in the Recognition Pipeline

**Files:**
- Create: `internal/service/ai/provider.go`
- Modify: `internal/service/ai/recognize.go:88-100`
- Test: `internal/service/ai/worker_test.go` (append two tests)

**Interfaces:**
- Consumes: `Settings` with `Provider`/`OpenAI*`/`Anthropic*` fields (Task 1); `NewOpenAIClient` (Task 2); `NewAnthropicClient` (Task 3); `NewOllamaClient` (existing). Test helpers already in `worker_test.go`/`config_test.go`: `workerFixture`, `seedTagLibrary`, `registerMedia`, `encodeFrame`, `tintRed`, `bandBrightness`, `statusOf`, `newestTask`, `waitFor`, `captureLogs` (`.containing`), `countSuggestions`, `closedLocalPort`, `fakeClient`, `fakeExtractor`, `matchedJSON`, `seedRaw`.
- Produces: `checkProviderModel(settings Settings) error`, `newVisionClient(settings Settings) VisionClient`, `providerLabel(provider string) string` (all unexported, `provider.go`).

- [ ] **Step 1: Write the failing worker tests**

Append to `worker_test.go`:

```go
// 与 TestWorker_EmptyModelConfigFails 同形，但逐个端点类型验证：缺的是
// 「当前端点」的模型名——fixture 种下的 ollama_model 不能满足 openai/anthropic
// 的检查，配置缺失必须在建客户端（以及 fake seam）之前拦下。
func TestWorker_EmptyModelConfigFailsPerProvider(t *testing.T) {
	for _, tc := range []struct{ provider, label string }{
		{ai.ProviderOpenAI, "OpenAI"},
		{ai.ProviderAnthropic, "Anthropic"},
	} {
		t.Run(tc.provider, func(t *testing.T) {
			workerFixture(t)
			logs := captureLogs(t)
			seedTagLibrary(t)
			if err := configSvc.NewService().Set(ai.KeyProvider, tc.provider); err != nil {
				t.Fatalf("set ai_provider: %v", err)
			}
			// 该端点的模型名默认 ""（未配置），无需再清空。

			svc := ai.NewService()
			defer svc.StopWorker()
			svc.Extractor = fakeExtractor{}
			client := &fakeClient{out: matchedJSON}
			svc.Client = client

			picture := registerMedia(t, "a.jpg", encodeFrame(t, tintRed, bandBrightness(0b0000_1111)))
			if _, err := svc.EnqueuePaths([]string{"a.jpg"}, false); err != nil {
				t.Fatalf("EnqueuePaths: %v", err)
			}
			svc.StartWorker()
			waitFor(t, "缺模型名的任务变成 failed", func() bool { return statusOf(t, picture.FileID) == ai.TaskFailed })

			task := newestTask(t, picture.FileID)
			if !strings.Contains(task.Error, "未配置") || !strings.Contains(task.Error, tc.label) {
				t.Errorf("Error = %q, want 含「未配置」与 %s", task.Error, tc.label)
			}
			if got := countSuggestions(t, picture.FileID, ai.ReviewPending); got != 0 {
				t.Errorf("失败任务留下 %d 条待审建议, want 0", got)
			}
			svc.StopWorker()
			if got := client.callCount(); got != 0 {
				t.Errorf("模型被调用 %d 次, want 0 (配置缺失要在建客户端之前就拦下)", got)
			}
			if got := logs.containing("识别任务"); len(got) != 1 || !strings.Contains(got[0], "未配置") {
				t.Errorf("缺配置该有一行带原因的日志, got %v", got)
			}
		})
	}
}

// TestWorker_ProviderDispatchFollowsConfig pins dispatch + per-task rebuild in
// one pipeline run: every leg edits config while no worker runs, retries the
// same file, and the failure must name the port of the endpoint the config now
// selects — never the previous one. Leg 4 plants garbage ai_provider and must
// land back on the ollama URL (read-side fallback reaching dispatch).
func TestWorker_ProviderDispatchFollowsConfig(t *testing.T) {
	workerFixture(t)
	seedTagLibrary(t)

	svc := ai.NewService()
	defer svc.StopWorker()
	svc.Extractor = fakeExtractor{}
	config := configSvc.NewService()
	set := func(key, value string) {
		t.Helper()
		if err := config.Set(key, value); err != nil {
			t.Fatalf("set %s: %v", key, err)
		}
	}

	picture := registerMedia(t, "a.jpg", encodeFrame(t, tintRed, bandBrightness(0b0000_1111)))
	result, err := svc.EnqueuePaths([]string{"a.jpg"}, false)
	if err != nil {
		t.Fatalf("EnqueuePaths: %v", err)
	}
	if len(result.Queued) != 1 {
		t.Fatalf("queued %d tasks, want 1 (skipped: %+v)", len(result.Queued), result.Skipped)
	}

	failOnce := func(why string) *model.AITask {
		t.Helper()
		svc.StartWorker()
		waitFor(t, why, func() bool { return statusOf(t, picture.FileID) == ai.TaskFailed })
		svc.StopWorker()
		return newestTask(t, picture.FileID)
	}
	retry := func() {
		t.Helper()
		if err := svc.RetryTask(result.Queued[0]); err != nil {
			t.Fatalf("RetryTask: %v", err)
		}
	}

	ollamaPort := closedLocalPort(t)
	openaiPort := closedLocalPort(t)
	anthropicPort := closedLocalPort(t)
	if ollamaPort == openaiPort || openaiPort == anthropicPort || ollamaPort == anthropicPort {
		t.Fatalf("保留到重复端口 (%s/%s/%s), 无法区分端点", ollamaPort, openaiPort, anthropicPort)
	}

	set(ai.KeyOllamaURL, "http://127.0.0.1:"+ollamaPort)
	task := failOnce("默认 ollama 端点在死地址上失败")
	if !strings.Contains(task.Error, ollamaPort) {
		t.Errorf("Error = %q, want 点名 ollama 端口 %s", task.Error, ollamaPort)
	}

	set(ai.KeyProvider, ai.ProviderOpenAI)
	set(ai.KeyOpenAIURL, "http://127.0.0.1:"+openaiPort)
	set(ai.KeyOpenAIModel, "gpt-4o")
	retry()
	task = failOnce("切到 openai 后在另一个死地址上失败")
	if !strings.Contains(task.Error, openaiPort) || strings.Contains(task.Error, ollamaPort) {
		t.Errorf("Error = %q, want 只点名 openai 端口 %s (分派没有跟着 ai_provider 走)", task.Error, openaiPort)
	}

	set(ai.KeyProvider, ai.ProviderAnthropic)
	set(ai.KeyAnthropicURL, "http://127.0.0.1:"+anthropicPort)
	set(ai.KeyAnthropicModel, "claude-sonnet-4-5")
	retry()
	task = failOnce("切到 anthropic 后在第三个死地址上失败")
	if !strings.Contains(task.Error, anthropicPort) || strings.Contains(task.Error, openaiPort) {
		t.Errorf("Error = %q, want 只点名 anthropic 端口 %s", task.Error, anthropicPort)
	}

	// 非法端点类型回落 ollama：回到第一条腿配置的 ollama 地址。
	seedRaw(t, ai.KeyProvider, "bogus")
	retry()
	task = failOnce("非法端点类型回落到 ollama")
	if !strings.Contains(task.Error, ollamaPort) || strings.Contains(task.Error, anthropicPort) {
		t.Errorf("Error = %q, want 回落 ollama 端口 %s", task.Error, ollamaPort)
	}

	if got := countSuggestions(t, picture.FileID, ai.ReviewPending); got != 0 {
		t.Errorf("四个失败任务留下 %d 条待审建议, want 0", got)
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/service/ai/ -run 'TestWorker_EmptyModelConfigFailsPerProvider|TestWorker_ProviderDispatchFollowsConfig' -v`
Expected: FAIL behaviorally (not compile) — dispatch ignores `ai_provider`: the openai leg's error names the ollama port; the per-provider empty-model test sees the fixture's ollama model satisfy the check and the fake client gets called (`callCount 1`).

- [ ] **Step 3: Implement `provider.go` and the dispatch edit**

`provider.go` (full code — the error text is pinned by the existing ollama test, which must keep passing):

```go
package ai

import "fmt"

// providerLabel names the provider in user-facing config errors.
func providerLabel(provider string) string {
	switch provider {
	case ProviderOpenAI:
		return "OpenAI"
	case ProviderAnthropic:
		return "Anthropic"
	default:
		return "Ollama"
	}
}

// checkProviderModel reports the active provider's missing model name as the
// config error it is. It runs BEFORE the Client seam so a missing model is
// never papered over by an injected client (TestWorker_EmptyModelConfigFails).
func checkProviderModel(settings Settings) error {
	var model string
	switch settings.Provider {
	case ProviderOpenAI:
		model = settings.OpenAIModel
	case ProviderAnthropic:
		model = settings.AnthropicModel
	default:
		model = settings.OllamaModel
	}
	if model == "" {
		return fmt.Errorf("未配置 %s 模型名, 请先在配置页填写", providerLabel(settings.Provider))
	}
	return nil
}

// newVisionClient builds the client for the configured endpoint. Called per
// task, so config edits take effect on the next file without a restart. The
// default branch is ollama — also the landing spot of the read-side fallback
// for a garbage ai_provider value.
func newVisionClient(settings Settings) VisionClient {
	switch settings.Provider {
	case ProviderOpenAI:
		return NewOpenAIClient(settings.OpenAIURL, settings.OpenAIModel, settings.OpenAIAPIKey)
	case ProviderAnthropic:
		return NewAnthropicClient(settings.AnthropicURL, settings.AnthropicModel, settings.AnthropicAPIKey)
	default:
		return NewOllamaClient(settings.OllamaURL, settings.OllamaModel)
	}
}
```

In `recognize.go` replace lines 88-100 (the `settings := s.Settings()` line stays; the ollama-only check and `NewOllamaClient` call go) with:

```go
	settings := s.Settings()
	if err := checkProviderModel(settings); err != nil {
		return err
	}
	client := s.Client
	if client == nil {
		// Built per task (NewService deliberately leaves Client nil), so editing the
		// provider, address or model takes effect on the next file without a restart.
		client = newVisionClient(settings)
	}
```

Everything below (`ctx := context.Background()`, frames, prompt, single `Recognize`, parse, cap, store) is untouched — the per-frame pipeline from plan 1 keeps working through the same interface.

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./internal/service/ai/ -v`
Expected: PASS — including the unchanged `TestWorker_EmptyModelConfigFails` (ollama error text is byte-identical) and `TestWorker_RebuildsClientFromSettingsPerTask`.

- [ ] **Step 5: Commit**

```bash
git add internal/service/ai/provider.go internal/service/ai/recognize.go internal/service/ai/worker_test.go
git commit -m "feat(ai): dispatch recognition to the configured provider per task"
```

---

### Task 5: Release 0.10.0

**Files:**
- Modify: `docs/CHANGELOG.md` (new top entry)
- Modify: `internal/constant/version.go:3`

**Interfaces:**
- Consumes: everything above; `constant.CoreVersion` read by `backupDBIfNeeded` (a version bump triggers the automatic DB backup on next start — intended).
- Produces: core `0.10.0`.

- [ ] **Step 1: Add the CHANGELOG entry**

Insert directly under the header block (above `## [0.9.0]`), EN line then ZH line with no blank line between them, blank line after the pair:

```markdown
## [0.10.0] - 2026-11-19

### Added / 新增

- Vision recognition supports three selectable endpoint formats: Ollama native (the default, unchanged), OpenAI-compatible chat completions (vLLM, LM Studio, one-api, …) and Anthropic messages. The settings page gains an `ai_provider` selector whose per-provider URL, model and API-key fields follow the selection via the schema's `ShowIf`; keys are masked and left-empty-to-keep. Clients are hand-written stdlib HTTP built per task, so switching provider or editing credentials takes effect on the next file without a restart. Image media types are sniffed from the bytes, so PNG uploads reach OpenAI/Anthropic correctly.
- 视觉识别支持三种可选端点格式：Ollama 原生（默认，行为不变）、OpenAI 兼容 chat completions（vLLM、LM Studio、one-api 等）与 Anthropic messages。设置页新增 `ai_provider` 选择器，各端点的地址、模型、API 密钥字段经 schema 的 `ShowIf` 随选择显隐；密钥掩码、留空不覆盖。客户端为手写标准库 HTTP 实现、按任务构建，切换端点或修改凭据后下一个文件即生效，无需重启。图片媒体类型按字节嗅探，PNG 图片也能正确送达 OpenAI/Anthropic。
```

(If the execution date differs, use the actual commit date.)

- [ ] **Step 2: Bump the version constant**

`internal/constant/version.go`: `const CoreVersion = "0.10.0"`. No `WebuiVersion` change — no file under `cmd/webui/` was touched (Global Constraints).

- [ ] **Step 3: Run the full suite**

Run: `make test`
Expected: PASS across every package (bootstrap seed counts, ai config/clients/worker, schema, webui handlers).

- [ ] **Step 4: Commit**

```bash
git add docs/CHANGELOG.md internal/constant/version.go
git commit -m "chore: bump CoreVersion to 0.10.0 with multi-provider endpoints changelog"
```

---

## Self-Review Record

1. **Spec coverage:** approved design points → provider enum + fallback (Task 1/4), 7 keys + ShowIf + secret (Task 1), OpenAI client (Task 2), Anthropic client (Task 3), per-task dispatch + empty-model gate (Task 4), release (Task 5). Settings page needs no code: `fieldVisible` (config/main.js:269-277), `oneof`→select widget (rules.go:92), secret masking (api/config.go:54-57,200-205) and 留空不覆盖 (service.go:66-70) all verified present.
2. **Step scan:** every step names one action with a checkable result; descriptor/wire-struct/test code is declaration data or the new contract, not bodies the tests already determine. Recognize bodies are ordered steps with exact error strings.
3. **Type consistency:** `Settings` field names (Task 1) match `checkProviderModel`/`newVisionClient` reads (Task 4); constructor signatures (Pinned Decision 11) match factory calls; `postVisionJSON` signature (Task 2 Interfaces) matches both new clients' calls; `visionBodySnippet` used by the in-package test exists after Task 2's move.
4. **Review Focus:** all five lines have owning tests (listed inline). Provider-switch-mid-flight is additionally pinned by Task 4's four-leg dispatch test.
5. **Proportion:** mirrored test scaffolding is referenced (`ollama_test.go` patterns) rather than transcribed; full code appears only for new contracts, declarations and the two integration tests.
6. **Cross-file traps checked:** bootstrap `init_test.go` literal 6-row counts adapted in Task 1 (the only non-ai package affected); `parse.go:22` comment reference renamed in Task 2; existing ollama/worker error-text assertions preserved byte-for-byte (verified against `ollama_test.go` and `worker_test.go:598-635,792-853`); `seedRaw`/`closedLocalPort`/`captureLogs` helpers confirmed same-package reachable; `saveGroup` submits hidden fields but their seeded defaults are all valid (no SetGroup change needed).