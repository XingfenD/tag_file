# AI 多端点推理（Ollama / OpenAI 兼容 / Anthropic）— 设计文档

日期：2026-11-19
状态：已与需求方对话确认（「加可选多种端点类型」「手写客户端」）；实施计划见 `docs/superpowers/plans/2026-11-19-ai-multi-provider-endpoints.md`，排在逐帧计划（`2026-11-19-ai-per-frame-scene-coverage.md`）合入之后执行。本文是 `2026-11-13-ollama-vision-tagging-design.md` 的端点层扩展：识别管线（抽帧、近似帧过滤、逐帧识别、合并去重、审核闸门、任务状态机）一律不变。

## 1. 意图与背景

现有视觉识别只接 Ollama 原生端点（`POST /api/chat`，images 为裸 base64 数组，`format:"json"`）——这是 Ollama 独有协议，不是行业通用格式。用户手里可能已经有别的推理服务：

- **OpenAI 兼容** chat completions（`/v1/chat/completions`）：OpenAI 官方、vLLM、LM Studio、llama.cpp server、one-api 等聚合网关都讲这个格式；
- **Anthropic** messages（`/v1/messages`）：Claude 视觉模型。

需求：把端点格式做成可选项，同一套识别管线按配置接入不同服务，而不是为每家服务各造一条管线。

社区有官方 Go SDK（`github.com/openai/openai-go`、`github.com/anthropics/anthropic-sdk-go`，均支持自定义 base URL 与视觉输入），但需求方拍板**手写薄客户端**：仓库目前只有 3 个直接依赖，三种格式各是一个 POST + 一层 JSON 解析，SDK 的抽象面超过需要。

配置侧的路已经铺好：config-schema 协议里枚举（`oneof`→select 控件）、secret（掩码 + 留空不覆盖）、`ShowIf`（条件显隐）均已端到端实现，且 spec 明文预期 `ai_provider` 是 ShowIf 的第一个使用者（`2026-11-16-config-schema-design.md:77,115,176`）。因此本功能**前端零代码改动**。

## 2. 目标

- 设置页新增「推理端点类型」选择：`ollama`（默认）/ `openai` / `anthropic`。
- 每种端点有自己的地址、模型名、API 密钥（Ollama 无密钥，保持现状）；字段随选择显隐。
- 识别按任务读取当前配置构建客户端：切换端点或修改凭据，下一个文件即生效，无需重启。
- 三个客户端共用同一个 `VisionClient` 接口、同一套传输预算与错误语义；上游逐帧识别、合并、审核闸门完全不变。
- 现有 Ollama 用户无感：键名不改、默认值不改、行为不改、无数据迁移。

## 3. 非目标

- 不引入任何 SDK 或新依赖（手写标准库 HTTP）。
- 不做端点连通性测试按钮、模型列表拉取、模型自动发现。
- 不支持流式响应（一律非流式）。
- 不做多端点并行、自动故障转移链——一个任务只用一个端点，失败就是失败，可重试。
- 不改提示词、输出解析、审核流程、任务状态机、抽帧与过滤。
- 不开放 temperature / top_p 等逐端点参数面（Anthropic 的 `max_tokens` 是协议必填字段，取固定常量，不进配置）。
- 不做 CLI 入口（AI 识别仅 WebUI，沿用原 spec）。

## 4. 反目标（做成了也算失败）

- 图片 `media_type` 写死 `image/jpeg`——PNG 上传被 Anthropic/OpenAI 拒收。
- 给免鉴权的本地服务发空 `Authorization: Bearer ` / 空 `x-api-key`——被服务端拒绝。
- 空答案（`choices:[]`、`content:[]`、空串、只有非文本块）被当成「识别成功但没标签」——审核队列看到一个完成却无东西可审的文件。
- 非法 `ai_provider` 值让识别崩溃，或拨到错误端点。
- 密钥明文回传前端，或整组保存时把空提交当成「清空」抹掉已存密钥。
- 重命名现有 `ollama_url` / `ollama_model` 导致存量配置丢失。
- 改配置要重启进程才生效。

## 5. 已确认的决策（需求方拍板）

| 决策点 | 结论 |
|---|---|
| 端点格式 | 可选多种：Ollama 原生（默认）、OpenAI 兼容、Anthropic |
| 实现方式 | 手写薄客户端，不引入 SDK（「你要是喜欢手写就手写吧」） |
| Ollama 现状 | 默认端点，键名与行为不变 |

## 6. 默认选择（设计者决定，需求方未反对，可否决）

### 6.1 配置模型

块 `ai` 由 5 键增至 12 键。新增 7 键 + 现有 2 键加显隐条件：

| 键 | 约束 | 默认值（存储形态） | 显示条件 |
|---|---|---|---|
| `ai_provider` | 枚举 ollama / openai / anthropic | `ollama` | 恒显 |
| `ollama_url`（现有） | required + url | `http://127.0.0.1:11434` | provider = ollama |
| `ollama_model`（现有） | 无（空 = 未配置） | `""` | provider = ollama |
| `ai_openai_url` | required + url | `https://api.openai.com/v1` | provider = openai |
| `ai_openai_model` | 无（空 = 未配置） | `""` | provider = openai |
| `ai_openai_api_key` | secret | `""` | provider = openai |
| `ai_anthropic_url` | required + url | `https://api.anthropic.com` | provider = anthropic |
| `ai_anthropic_model` | 无（空 = 未配置） | `""` | provider = anthropic |
| `ai_anthropic_api_key` | secret | `""` | provider = anthropic |

三个抽帧键（`ai_frame_interval` / `ai_frame_diff_threshold` / `ai_max_frames`）声明不变、恒显。

- **读侧回落**：`ai_provider` 未知值（含空串、大小写漂移如 `Ollama`）→ 记日志并回落 `ollama`，与 `intSetting` 对非法数的处理同形；其余键沿用 `Settings()` 逐键独立回落（缺键/读错/非法值 → 默认值，永不失败）。写侧由 `oneof` / `required` / `url` 校验拒收非法值。
- **URL 语义**：OpenAI base URL **含 `/v1`**（客户端追加 `/chat/completions`；vLLM/LM Studio/one-api 同形）；Anthropic base URL 是根地址（客户端追加 `/v1/messages`）。两者只 trim 尾部斜杠，不自动补前缀——补前缀会让带路径的网关地址（如 `https://gw.example.com/openai/v1`）无法配置。
- **密钥可选**：本地 OpenAI 兼容服务普遍免鉴权，空密钥是合法存储值，此时不发任何鉴权头。secret 行为沿用 config-schema spec §8：服务端投影掩码（值从未下发）、`SetGroup` 缺失/空串 = 保持不变，行为强制在服务端独立成立。
- **种子与升级**：bootstrap 按注册表 walk 自动补种新键（`internal/bootstrap/init.go:35-52`），存量库下次启动即完成升级，无手工迁移、无账本条目。
- **设置页零改动**：`oneof`→select、`secret`→password、`ShowIf` 显隐均为已实现协议（config-schema spec §4 / §8 / §9）。`saveGroup` 提交整组**含隐藏字段**（`config/main.js:387-401`），隐藏字段保持合法的种子默认值，整组保存不会死锁；不改 `SetGroup`。

### 6.2 线上契约（三种格式）

**Ollama（现状，逐字不变）**：`POST {base}/api/chat`，`{model, messages:[{role:"user", content:提示词, images:[裸base64…]}], stream:false, format:"json"}`；读 `message.content`。

**OpenAI 兼容**：`POST {base}/chat/completions`

```json
{
  "model": "…",
  "stream": false,
  "response_format": {"type": "json_object"},
  "messages": [{"role": "user", "content": [
    {"type": "text", "text": "提示词"},
    {"type": "image_url", "image_url": {"url": "data:<嗅探MIME>;base64,<图1>"}}
  ]}]
}
```

- 鉴权：密钥非空时 `Authorization: Bearer <key>`，否则不发该头。
- 答案：`choices[0].message.content` 原样返回；`choices` 为空或 content 为空 → 失败（§6.3）。
- 部分兼容实现不支持 `response_format`：服务端错误原样进任务 Error（带 body 片段），**不做逐端点降级开关**（YAGNI，错误可见即可诊断）。

**Anthropic**：`POST {base}/v1/messages`

```json
{
  "model": "…",
  "max_tokens": 4096,
  "messages": [{"role": "user", "content": [
    {"type": "image", "source": {"type": "base64", "media_type": "<嗅探MIME>", "data": "<裸base64>"}},
    {"type": "text", "text": "提示词"}
  ]}]
}
```

- 头：`anthropic-version: 2023-06-01` 恒发；`x-api-key: <key>` 仅密钥非空时发。
- `max_tokens` 为协议必填，固定常量 4096（提示词约定 1–6 条标签的 JSON，绰绰有余）。
- 块序：**图片块在前、文本块在最后**。
- 答案：按序拼接 `content[]` 中所有 `type == "text"` 块的 `text`（无分隔符）；非文本块（如 `tool_use`）不参与；拼完为空 → 失败。
- 无 JSON 模式：提示词已要求 JSON，`ParseSuggestions` 容忍代码围栏/散文包裹（`parse.go:47-58`），不需要协议层强制。

**图片 MIME**：OpenAI data-URI 前缀与 Anthropic `source.media_type` 一律按字节嗅探（`http.DetectContentType`，标准库）——视频帧是 JPEG，但上传的图片文件可能是 PNG/GIF/WebP。Ollama 保持裸 base64（无 MIME 概念）。

### 6.3 传输与错误语义（三客户端共用）

- 预算：180s 超时、4 MiB 响应体上限、错误消息取 body 前 200 rune（rune 边界截断，中文错误页不切半个字）。
- 空模型名 / 空地址 → `未配置 X 模型名` / `未配置 X 服务地址`，**任何字节发出之前**拦下。
- 非 2xx → 状态码 + body 片段；200 但非 JSON → 解析错误并引用 body（绝不报「模型返回空内容」，那是模型的锅不是网关的）；空答案 → `模型返回空内容`。
- 调用方 deadline 可归因：`errors.Is(err, context.DeadlineExceeded)` 成立。
- 共享 HTTP 管道从 `ollama.go` 抽出（常量改名 `vision*`），Ollama 错误文案逐字不变——现有五个 ollama 测试是重构安全网。

### 6.4 分派

- `recognize` 每任务：先查**当前端点**的模型名（先于 `Client` 注入 seam——缺配置不能被注入的假客户端掩盖），再按 `ai_provider` 建客户端；default 分支 = ollama，也是非法值回落的落点。
- 缺模型名 → 任务失败「未配置 X 模型名, 请先在配置页填写」，不发请求、不产建议、记一行日志。
- 上游逐帧管线（计划一）经同一个 `VisionClient` 接口穿过，抽帧/过滤/合并/审核一律不改。

### 6.5 版本

- core `0.9.0` → `0.10.0`，只记 `docs/CHANGELOG.md`；不触碰 `cmd/webui/` 下任何文件，不 bump `WebuiVersion`（AGENTS.md 按文件归属记日志）。

## 7. 系统现状与约束（实现必须遵守）

- `VisionClient` 接口不变（`internal/service/ai/ports.go:13`）：`Recognize(ctx, prompt, images [][]byte) (string, error)`。
- 客户端按任务构建（`NewService` 留 `Client` nil）；worker、审核、状态机、抽帧不变。
- `Settings()` 永不失败：逐键独立回落默认值并记日志。
- config schema 是叶子包：新键声明住 `ai/config_schema.go`，默认值/边界常量住消费方 `ai/config.go`，注册表只做声明。
- Go 1.25 标准库，`go.mod` 直接依赖不变；测试 stdlib `testing` + `httptest`，外部 `package ai_test`（私有符号用 in-package，先例 `parse_test.go`）；`make test`。
- 不在 master 提交；CHANGELOG 同条目英文行 + 中文行相邻、高版本在上；版本号与常量同 commit。
- bootstrap 种子测试对行数有字面量断言（现 6 行），新增键需要把 `internal/bootstrap/init_test.go` 同步改为 13 行——这是唯一受影响的非 ai 包。

## 8. 方案轮廓（方向性，细节留给实现者）

- `config.go` / `config_schema.go`：7 新键 + provider 常量 + `Settings` 扩字段 + 12 键声明。
- `client.go`（新）：共享传输管道（超时/体上限/片段截断/统一 POST）。
- `openai.go` / `anthropic.go`（新）：两个薄客户端，各自 wire struct 私有。
- `ollama.go`：重构到共享管道，错误文案不变。
- `provider.go`（新）+ `recognize.go`：模型名检查 + 按 provider 建客户端，替换现有 ollama-only 块。
- 签名、wire struct、错误文案逐字、测试用例均由实施计划钉死。

## 9. 留给实现者的决定

实施计划已钉死签名、wire struct、错误文案与测试；剩余仅非契约细节：内部辅助函数的进一步拆分、日志措辞、测试脚手架的组织方式。

## 10. 验收口径

- 设置页出现端点类型选择；切换后对应字段显隐正确；密钥掩码、留空不覆盖、绝不明文回传。
- 配置 OpenAI 兼容 / Anthropic 端点与模型名后，图片/视频识别产出建议并进待审（管线与 ollama 一致）。
- 免鉴权本地服务（空密钥）可用；不发空鉴权头。
- PNG 图片文件在 openai / anthropic 端点正确识别（media_type 与字节一致）。
- 默认 ollama 行为不变：现有 ollama / worker 测试逐字通过。
- 改端点/凭据下一个任务生效，无需重启；非法 `ai_provider` 回落 ollama 且有日志。
- 缺模型名、空答案、非 JSON 响应都成为可见的任务失败，无静默零建议。
- `make test` 全绿；`go.mod` 直接依赖不变。