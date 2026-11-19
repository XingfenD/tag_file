# AI 逐帧识别 + 全片分镜覆盖抽帧 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 视频识别改为逐帧喂模型（合并去重、部分失败可容忍），抽帧改为全片覆盖（自适应间隔 + ffmpeg 硬切检测），近似帧过滤升级为区域平均 dHash + 全保留帧比对 + 均匀薄化。

**Architecture:** 三条改动线汇入现有单 worker 流水线。抽帧：ffprobe 取时长 → 第一遍 ffmpeg `select`(首帧+自适应周期+scene 硬切)+`showinfo` 只产候选时间戳 → Go 均匀薄化到 `maxFrames*10` → 第二遍逐时间戳 `-ss` 定点抽帧落盘。过滤：`FilterSimilarFrames` 改收文件路径，流式解码、box 平均 dHash、与所有已保留帧比对、不早停、超预算均匀薄化，只把最终 ≤maxFrames 帧读进内存。识别：`recognize` 对每帧单独调 `VisionClient.Recognize`（接口不变），逐帧解析，`mergeRawTags` 合并去重，≥1 帧成功即 done（失败帧汇总进 Error），全失败才 failed；建议仍在末尾一次性原子写入（≤50 < 仓库 100 行批量边界）。

**Tech Stack:** Go 1.25.0、stdlib `image`/`os/exec`/`regexp`、外部 ffmpeg + ffprobe（**不预设版本**：只用 4.x 起就有的参数；运行容器需自带 ffmpeg/ffprobe，见 Task 7）。**不新增任何 go.mod 依赖。**

**Spec:** `docs/superpowers/specs/2026-11-13-ollama-vision-tagging-design.md`（2026-11-19 已修订：重设计决策并入 §5.1，抽帧/过滤/识别段 §6 与流程 §8 同步更新）；下节为决策摘要，与 spec 同源，冲突时以 spec 为准。

## 重设计决策（2026-11-19 对话确认，默认值可否决但已拍板）

1. **逐帧喂模型**：每帧一次 `Recognize(ctx, prompt, [][]byte{frame})`，各自 180s 超时；prompt 每任务构建一次复用。图片文件天然单帧，行为不变。
2. **合并去重**：(category, tag, value) 三元组精确去重保留首次出现；同 (category, tag) 存在非空值行时丢弃空值行（与出现顺序无关）；同标签不同值都保留，交审核人定。
3. **部分成功**：≥1 帧成功 → 任务 done，失败帧汇总写 Error（沿用截断提示先例）；全部帧失败 → failed，Error 记首条原因。写库仍是末尾一次性 `CreateSuggestions`，「绝不给审核队列留半个建议集」不变。
4. **dHash 升级**：9×8 网格改区域平均（box downscale，仍纯 stdlib）；与**所有**已保留帧哈希比对（治 A→B→A 切回重复）；不早停，去重后超 maxFrames 按时间均匀薄化。
5. **抽帧两遍走**：`INTERVAL = max(配置间隔, 时长/(maxFrames*10))`（配置语义变「最小间隔」）；scene 阈值 **0.3 常量**不进配置；候选超 `maxFrames*10` 均匀薄化；第二遍 `scale='min(768,iw)':-2` 压分辨率（视觉模型内部本就 resize，同时压掉候选帧内存/磁盘风险）。
6. **超时随时长伸缩**：`max(120s, 时长×2)`，封顶 30min。ffprobe 缺失/取不到时长 → log + 退回配置间隔，不炸任务。
7. **明确不做**：`AISuggestion.frame_index` 列、任务进度列、pHash、PySceneDetect/Python 依赖、scale 可配置。渐变/淡入淡出转场由周期采样层兜底，不追求完美分镜分割。
8. **ffmpeg 跨版本/容器**（需求方 2026-11-19 指令）：方案在不同机器上执行、最终都在容器里跑，**不预设任何本机 ffmpeg 版本**。只用 4.x 起就存在的参数（清单见 Global Constraints）；现有 `Dockerfile` 运行层是 `FROM scratch`，没有 ffmpeg/ffprobe 也没有 `/tmp`，AI 识别在容器里必然失败——Task 7 把运行层换成 alpine + `apk add ffmpeg`（ffprobe 随包附带）。

## Global Constraints

以下为 spec 与仓库的硬约束，每个任务的要求都隐含包含本节。

- 分支 `feat/ai-per-frame-scene-coverage`；**禁止在 `master` 提交**。每次 commit 前 `git branch --show-current`。
- 不新增 go.mod 依赖；哈希/薄化纯 stdlib。外部依赖仅 ffmpeg/ffprobe，**不预设版本**（方案在不同机器执行、最终容器运行）。允许的参数白名单（全部 4.x 起就有，禁止使用白名单外的新参数）：
  - 第一遍：`select` 滤镜及表达式函数 `isnan`/`gte`/`gt`、变量 `prev_selected_t`/`t`/`scene`，`showinfo` 滤镜（stderr 打 `pts_time:`），`-f null -`，`-an -sn -dn`。
  - 第二遍：`-ss` 放 `-i` 前（2.1 起即帧级精确）、`-frames:v 1`、`scale='min(768,iw)':-2`（`min()` 表达式 4.x 已有）、`-q:v 2`、`-y`。
  - ffprobe：`-v error -show_entries format=duration -of default=noprint_wrappers=1:nokey=1`（各版本输出稳定）。
  - **刻意不用 `-fps_mode`**（5.1+ 才有）：第一遍 `-f null` 不写帧、第二遍 `-frames:v 1` 单帧，都不需要它。测试合成片段只用 lavfi `color`/`nullsrc`/`geq`/`concat` + 内置 `mpeg4` 编码器（不依赖外部编码器库）。
- 单 worker 顺序消费不变；`VisionClient`/`ollama.go`/`prompt.go`/`worker.go`/`enqueue.go`/`review.go`/`status.go` **零改动**。
- 全量人工审核、tag:value 语义、新值不进预设字典、`File.TagDone` 纯手动——全部不变。
- 建议集 ≤ `maxTaskSuggestions`(50) < 仓库 `batchSize`(100)：写入必须是单条 INSERT，截断发生在匹配之前。
- 无 schema 变更（不加列、不迁移）；config 只改描述文案，定点 wire 与边界不动。
- 核心改动记 `docs/CHANGELOG.md` 并 bump `internal/constant.CoreVersion`（0.8.0 → 0.9.0），**同 commit**；Dockerfile 改动（webui 运行镜像）记 `docs/CHANGELOG_webui.md` 并 bump `cmd/webui/constant.WebuiVersion`（5.11.0 → 5.12.0），同 commit。同条目英文行紧接中文行（中间无空行），不同条目空行，高版本在上。
- 测试：`make test`；stdlib `testing`，外部测试包 `package ai_test`，无 testify、无 `t.Parallel()`；**禁止构建临时测试二进制**。测试共享全局 DB，worker 测试必须 cleanup `StopWorker()`。
- ffmpeg 集成测试照现有模式：`exec.LookPath` 缺失即 `t.Skip`（ffprobe 同理）；合成片段用 lavfi（`writeRampClip` 模式），不提交媒体文件。断言只依赖白名单参数的稳定语义，不依赖任何特定版本的行为差异；无 ffmpeg/ffprobe 的机器上 `make test` 必须照样全绿（skip 不算失败）。
- 容器运行层必须自带 ffmpeg/ffprobe 与可写临时目录：现 `Dockerfile` 是 `FROM scratch`（两者皆无，`os.MkdirTemp` 也会因没有 `/tmp` 失败），Task 7 换 alpine 运行层修复。
- 错误文案中文、面向审核人；`errors.Is` 可穿透 deadline 归因（caller deadline 与内部超时分开报）。

## Review Focus

spec 隐含、常规路径测试不覆盖的六类输入/失败模式，按最可能咬人排序。每条在归属任务补了测试或验证步骤。

1. **长视频抽帧超时 / ffprobe 缺失**——用户期望超时随时长伸缩而不是 120s 一刀切失败；ffprobe 不在时退回配置间隔继续识别，不是整个功能瘫痪。→ Task 4 `extractTimeout`/`adaptiveInterval` 纯函数测试 + 缺失文件集成测试。
2. **快切视频候选爆量**（几千个 cut）——用户期望薄化到预算且覆盖全片（含尾段），不是只留开头、不是 OOM。→ Task 1 `thinUniform` 大输入测试 + Task 4 候选薄化集成测试（断言尾帧）。
3. **部分帧失败但任务成功**——用户期望任务 done + Error 说明，文件级 AI 状态走 pending/approved 而非 failed（`status.go` 只认 TaskFailed）。→ Task 6 部分失败测试。
4. **单帧模型输出坏 JSON / 空内容**——用户期望只损失该帧，其余帧建议照常入队，不是整任务失败。→ Task 6 坏 JSON 单帧测试。
5. **临时目录泄漏**——用户期望 Extract 任何错误路径与成功路径都不留 `tagfile-ai-*` 目录；`Cleanup` 重复调用无害。→ Task 3 `Cleanup` nil 安全/幂等测试（`ports_internal_test.go`）+ 既有 `ExtractsFramesInTimeOrder` 的 `tagfileTempDirs` 前后快照（Task 3/4 改接口后仍生效，成功路径 `defer Cleanup` 删目录）。
6. **容器里没有 ffmpeg**——最终部署都在容器里跑，而现运行镜像是 `FROM scratch`：没有 ffmpeg/ffprobe、没有 `/tmp`，视频任务必然失败。→ Task 7 运行层换 alpine + `apk add ffmpeg`，有容器引擎时构建并运行 `ffmpeg -version`/`ffprobe -version` 验证。

---

## File Structure

| 文件 | 动作 | 职责 |
|---|---|---|
| `internal/service/ai/sample.go` | 新增 | `thinUniform` 均匀薄化（时间戳与帧共用） |
| `internal/service/ai/sample_test.go` | 新增 | 薄化纯函数测试 |
| `internal/service/ai/diff.go` | 修改 | box 平均 dHash；`FilterSimilarFrames(paths []string, …)` 流式过滤 |
| `internal/service/ai/diff_test.go` | 修改 | 噪声鲁棒、切回去重、均匀薄化、路径化既有测试 |
| `internal/service/ai/ports.go` | 修改 | `ExtractedFrames{Paths, dir}` + `Cleanup`；`FrameExtractor` 新签名 |
| `internal/service/ai/frames.go` | 重写 | ffprobe + 两遍抽帧 + 定点抽取 + 超时伸缩 |
| `internal/service/ai/frames_test.go` | 修改 | 分镜覆盖、尾段覆盖、候选薄化、时长探测集成测试 |
| `internal/service/ai/merge.go` | 新增 | `mergeRawTags` 跨帧合并去重 |
| `internal/service/ai/merge_test.go` | 新增 | 合并规则测试 |
| `internal/service/ai/recognize.go` | 修改 | 逐帧循环 + 部分失败语义 + `loadFrames` 编排 |
| `internal/service/ai/worker_test.go` | 修改 | `fakeExtractor` 落盘化、`perFrameClient`、逐帧/部分失败/合并截断测试 |
| `internal/service/ai/config_schema.go` | 修改 | 三个键的描述文案（间隔=最小间隔、阈值=全保留帧、帧数=逐帧送模型） |
| `internal/constant/version.go` | 修改 | CoreVersion 0.9.0 |
| `docs/CHANGELOG.md` | 修改 | 0.9.0 条目 |
| `Dockerfile` | 修改 | 运行层 scratch → alpine + ffmpeg（容器里 AI 识别的前提） |
| `cmd/webui/constant/version.go` | 修改 | WebuiVersion 5.12.0 |
| `docs/CHANGELOG_webui.md` | 修改 | 5.12.0 条目（镜像自带 ffmpeg/ffprobe） |

---

### Task 1: 均匀薄化纯函数 + 分支

**Files:**
- Create: `internal/service/ai/sample.go`
- Create: `internal/service/ai/sample_test.go`

**Interfaces:**
- Consumes: 无。
- Produces: `func thinUniform(n, budget int) []int`（包内私有，Task 3/4 调用）。返回从 `0..n-1` 中选出的升序索引：`n ≤ budget` 时全选；否则恰好 `budget` 个，首=0、尾=n-1、均匀分布。`budget == 1` 时返回 `[]int{0}`（避免除零）。

- [ ] **Step 1: 建分支**

```bash
git branch --show-current   # 必须是 master 且工作区干净
git checkout -b feat/ai-per-frame-scene-coverage
```

- [ ] **Step 2: 写失败测试** — 新建 `sample_test.go`。`thinUniform` 未导出，测试放**包内** `package ai`（纯函数无 DB 依赖，文件头注释说明原因；Task 4/5 的私有函数测试同此模式）：

```go
package ai

import "testing"

func TestThinUniform(t *testing.T) {
	cases := []struct{ n, budget int; want []int }{
		{0, 10, []int{}},
		{3, 10, []int{0, 1, 2}},
		{10, 10, []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}},
		{8, 3, []int{0, 3, 7}},
		{8, 1, []int{0}},
		{10000, 100, nil}, // 下面单独断言形状
	}
	for _, c := range cases {
		got := thinUniform(c.n, c.budget)
		if c.want != nil {
			if len(got) != len(c.want) { t.Fatalf("thinUniform(%d,%d)=%v, want %v", c.n, c.budget, got, c.want) }
			for i := range got { if got[i] != c.want[i] { t.Fatalf("thinUniform(%d,%d)=%v, want %v", c.n, c.budget, got, c.want) } }
			continue
		}
		if len(got) != c.budget || got[0] != 0 || got[len(got)-1] != c.n-1 {
			t.Fatalf("thinUniform(%d,%d) 形状错: len=%d first=%d last=%d", c.n, c.budget, len(got), got[0], got[len(got)-1])
		}
		for i := 1; i < len(got); i++ { if got[i] <= got[i-1] { t.Fatalf("索引非严格升序: %v", got) } }
	}
}
```

- [ ] **Step 3: 跑测试确认失败** — `go test ./internal/service/ai/ -run TestThinUniform -v`。预期 FAIL：`undefined: thinUniform`。

- [ ] **Step 4: 实现** — `sample.go`，`package ai`。公式：`budget==1 → []int{0}`；`n<=budget → 0..n-1`；否则 `idx_i = i*(n-1)/(budget-1)`（整数除法，i 从 0 到 budget-1）。n==0 返回空切片。

- [ ] **Step 5: 跑测试确认通过** — 同 Step 3 命令。预期 PASS。

- [ ] **Step 6: Commit**

```bash
git add internal/service/ai/sample.go internal/service/ai/sample_test.go
git commit -m "feat(ai): 均匀薄化采样器 thinUniform"
```

---

### Task 2: dHash 区域平均采样

**Files:**
- Modify: `internal/service/ai/diff.go:29-55`（`dHash` 内部）
- Modify: `internal/service/ai/diff_test.go`（新增测试 + `writeFrames` helper）

**Interfaces:**
- Consumes: 无（`dHash(img image.Image) uint64` 签名不变）。
- Produces: `dHash` 新语义——9×8 网格每格取源图对应矩形的**平均** Rec.601 亮度（不再单点采样），梯度符号规则不变。Task 3 沿用。测试 helper `writeFrames(t *testing.T, frames [][]byte) []string`（写入 `t.TempDir()`，返回路径，Task 3 复用）。

- [ ] **Step 1: 写失败测试** — `diff_test.go` 追加。脉冲噪声打在**旧最近邻采样点**上：旧哈希翻 ~64 位 → 误保留；box 平均把单像素稀释到 ~1/56 → 哈希稳定 → 正确丢弃。**必须用 PNG 无损编码**：JPEG 压缩本身就是低通滤波，会把单像素脉冲抹平到旧哈希也认不出，测试就无法确定性失败。先把 `encodeFrame` 的像素填充抽成 `renderFrame(tint, brightness) *image.RGBA`，`encodeFrame` 改调它（行为不变），再加 PNG 变体：

```go
func renderFrame(tint color.RGBA, brightness func(x, y int) uint8) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, frameEdge, frameEdge))
	for y := 0; y < frameEdge; y++ {
		for x := 0; x < frameEdge; x++ {
			level := uint32(brightness(x, y))
			img.SetRGBA(x, y, color.RGBA{
				R: uint8(uint32(tint.R) * level / 255),
				G: uint8(uint32(tint.G) * level / 255),
				B: uint8(uint32(tint.B) * level / 255),
				A: 255,
			})
		}
	}
	return img
}

func encodeFramePNG(t *testing.T, tint color.RGBA, brightness func(x, y int) uint8) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, renderFrame(tint, brightness)); err != nil { t.Fatalf("encode png: %v", err) }
	return buf.Bytes()
}

// writeFrames 把合成帧写进 t.TempDir(), 返回路径(按输入顺序)。
func writeFrames(t *testing.T, frames [][]byte) []string {
	t.Helper()
	dir := t.TempDir()
	paths := make([]string, 0, len(frames))
	for i, f := range frames {
		p := filepath.Join(dir, fmt.Sprintf("frame_%04d.jpg", i))
		if err := os.WriteFile(p, f, 0o644); err != nil { t.Fatalf("write frame: %v", err) }
		paths = append(paths, p)
	}
	return paths
}

func TestFilterSimilarFrames_ImpulseNoiseOnSamplePointsIsStillDuplicate(t *testing.T) {
	plain := bandBrightness(0b0000_1111)
	// 旧 dHash 的 72 个最近邻采样坐标: x*(63)/8, y*(63)/7。
	noisy := func(x, y int) uint8 {
		onGrid := false
		for gx := 0; gx < 9; gx++ {
			for gy := 0; gy < 8; gy++ {
				if x == gx*63/8 && y == gy*63/7 { onGrid = true }
			}
		}
		if onGrid {
			if plain(x, y) < 128 { return 255 }
			return 0
		}
		return plain(x, y)
	}
	// PNG 无损: 脉冲原样保留, 旧单点哈希必翻位; box 平均稀释 ~1/56, 梯度符号不变。
	frames := [][]byte{encodeFramePNG(t, tintRed, plain), encodeFramePNG(t, tintRed, noisy)}
	got, err := ai.FilterSimilarFrames(frames, 10, 10)
	if err != nil { t.Fatalf("FilterSimilarFrames: %v", err) }
	if len(got) != 1 {
		t.Fatalf("保留 %d 帧, want 1: 采样点脉冲噪声不该骗过区域平均哈希", len(got))
	}
}
```

（imports 补 `fmt`、`os`、`image/png`。writeFrames 的扩展名 .jpg 只是文件名，内容可以是 PNG——`image.Decode` 按魔数识别，diff.go 已注册 png 解码器。）

- [ ] **Step 2: 跑测试确认失败** — `go test ./internal/service/ai/ -run TestFilterSimilarFrames_ImpulseNoise -v`。预期 FAIL：保留 2 帧（旧单点哈希被翻位）。

- [ ] **Step 3: 实现 box 平均** — `diff.go` `dHash` 内部重写，签名不变。网格 cell (x,y) 覆盖源矩形 `[x*w/9, (x+1)*w/9) × [y*h/8, (y+1)*h/8)`（整数边界，最后一格取到 w/h）；对矩形内每个像素 `img.At().RGBA()` 按 Rec.601（0.299/0.587/0.114）累加求平均得 `level[y][x]`；梯度符号与位打包逻辑照旧。删掉旧 `px/py` 单点采样。注释说明：区域平均对 JPEG 伪影/脉冲噪声鲁棒，阈值语义才稳定。

- [ ] **Step 4: 跑测试确认通过** — `go test ./internal/service/ai/ -run 'TestFilterSimilarFrames|TestFFmpeg' -v`。预期：新测试 PASS，既有 `DropsNearDuplicates`/`AlwaysKeepsFirstAndCapsAtMax`/`RejectsUndecodableFrame` 全 PASS（box 平均不改变干净合成帧的判定）。

- [ ] **Step 5: Commit**

```bash
git add internal/service/ai/diff.go internal/service/ai/diff_test.go
git commit -m "feat(ai): dHash 改区域平均采样, 抗压缩伪影与脉冲噪声"
```

---

### Task 3: 过滤流式化 + 抽帧接口落盘化

**Files:**
- Modify: `internal/service/ai/ports.go`
- Create: `internal/service/ai/ports_internal_test.go`（`Cleanup` nil 安全/幂等测试，包内）
- Modify: `internal/service/ai/diff.go:63-93`（`FilterSimilarFrames`）
- Modify: `internal/service/ai/frames.go`（Extract 临时适配，Task 4 重写）
- Modify: `internal/service/ai/recognize.go:149-188`（`loadFrames` 视频分支）
- Modify: `internal/service/ai/diff_test.go`、`internal/service/ai/frames_test.go`、`internal/service/ai/worker_test.go:29-36,303`

**Interfaces:**
- Consumes: `thinUniform(n, budget int) []int`（Task 1）。
- Produces:
  - `type ExtractedFrames struct { Paths []string; dir string }`，`func (f *ExtractedFrames) Cleanup()`（nil 安全、幂等，`os.RemoveAll(f.dir)`）。`dir` 未导出：真实 extractor 内部设置；测试 fake 只填 `Paths`，Cleanup 空操作，文件由 `t.TempDir()` 回收。
  - `FrameExtractor.Extract(ctx context.Context, videoPath string, intervalSeconds float64, maxFrames int) (*ExtractedFrames, error)`。
  - `func FilterSimilarFrames(paths []string, threshold int, maxFrames int) ([][]byte, error)`：流式解码每个路径、与**所有**已保留帧哈希比对、不早停；去重后超 maxFrames 用 `thinUniform` 薄化；只读入选文件进内存。空输入 → 空切片 nil；解码失败 → `nil, fmt.Errorf("第 %d 帧解码失败: %w", i+1, err)`（i 为输入序号）。

- [ ] **Step 1: 写失败测试** — `diff_test.go`：既有三个过滤测试与 Task 2 新加的脉冲噪声测试都改用 `writeFrames` 传路径（空输入断言改 `ai.FilterSimilarFrames([]string{}, 10, 3)`）；`AlwaysKeepsFirstAndCapsAtMax` 加强断言薄化分布；新增切回场景测试。

```go
func TestFilterSimilarFrames_SceneReturnIsDedupedAgainstAllKept(t *testing.T) {
	// A→B→A: 第三个帧与「最后保留帧」B 距离大, 但与更早的 A 相同 → 必须丢弃。
	red := encodeFrame(t, tintRed, bandBrightness(0b0000_1111))
	blue := encodeFrame(t, tintBlue, bandBrightness(0b1111_0000))
	got, err := ai.FilterSimilarFrames(writeFrames(t, [][]byte{red, blue, red}), 10, 10)
	if err != nil { t.Fatalf("FilterSimilarFrames: %v", err) }
	if len(got) != 2 { t.Fatalf("保留 %d 帧, want 2 (A/B, 切回的 A 被去重)", len(got)) }
}
```

`AlwaysKeepsFirstAndCapsAtMax` 末尾追加：`if !bytes.Equal(got[2], frames[7]) { t.Error("薄化必须覆盖尾帧: 第 3 个保留帧应是输入第 8 帧") }`（thinUniform(8,3)=[0,3,7]）。`RejectsUndecodableFrame` 改：`writeFrames(t, [][]byte{[]byte("not an image")})` 传路径。

再加 `ports_internal_test.go`（`package ai`——`dir` 未导出，外部包构造不出带目录的实例；文件头注释说明原因）：

```go
package ai

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractedFramesCleanup(t *testing.T) {
	var nilFrames *ExtractedFrames
	nilFrames.Cleanup() // nil 接收者不得 panic

	scratch := filepath.Join(t.TempDir(), "tagfile-ai-test")
	if err := os.MkdirAll(scratch, 0o755); err != nil { t.Fatal(err) }
	f := &ExtractedFrames{dir: scratch}
	f.Cleanup()
	f.Cleanup() // 幂等: 第二次调用无害
	if _, err := os.Stat(scratch); !os.IsNotExist(err) {
		t.Fatalf("Cleanup 没有删掉 %s", scratch)
	}
}
```

- [ ] **Step 2: 跑测试确认失败** — `go test ./internal/service/ai/ -run TestFilterSimilarFrames -v`。预期 FAIL：编译错误（签名不符）/切回测试保留 3 帧。

- [ ] **Step 3: 改 ports.go** — 加 `ExtractedFrames` 类型与 `Cleanup` 方法（imports 补 `os`）；`FrameExtractor` 接口返回 `*ExtractedFrames`。`VisionClient` 不动。

- [ ] **Step 4: 重写 FilterSimilarFrames** — `diff.go`。两趟：A 趟按序 `os.Open`+`image.Decode` 每个路径，`dHash`，与 `keptHashes []uint64` **全部**比对（任一 `hamming < threshold` 即丢），否则记录 `(path, hash)`；B 趟 `len(distinct) > maxFrames` 时 `thinUniform` 选索引，`os.ReadFile` 入选路径返回字节。读文件失败 → `fmt.Errorf("读取帧文件 %s 失败: %w", path, err)`。注释写明：不早停是为了全片覆盖，内存只留最终帧。

- [ ] **Step 5: 临时适配 frames.go 与 loadFrames** — `ffmpegExtractor.Extract` 签名改 `*ExtractedFrames`，旧 `fps=` 命令暂不动。ffmpeg 本就把帧写进 tmpDir，**不要再经 `readFrames` 读回字节**：把 `readFrames` 的目录列举/排序逻辑改成返回路径切片（`readFramePaths(tmpDir) ([]string, error)`），返回 `&ExtractedFrames{Paths: paths, dir: tmpDir}`。原 `defer os.RemoveAll(tmpDir)` 改命名返回 `err` + `defer func(){ if err != nil { os.RemoveAll(tmpDir) } }()`——成功时目录移交调用方，由 `Cleanup` 负责删；零帧检查与错误文案照旧。`recognize.go` `loadFrames` 视频分支：

```go
extracted, err := s.Extractor.Extract(ctx, path, settings.FrameInterval, settings.MaxFrames)
if err != nil { return nil, fmt.Errorf("视频 %s 抽帧失败: %w", file.FilePath, err) }
defer extracted.Cleanup()
kept, err := FilterSimilarFrames(extracted.Paths, settings.DiffThreshold, settings.MaxFrames)
```

后续错误文案与空帧检查照旧。图片分支不动。

- [ ] **Step 6: 更新 fakeExtractor** — `worker_test.go:29-36`：

```go
type fakeExtractor struct {
	dir    string // 测试的 t.TempDir(); fake 不拥有目录, Cleanup 空操作
	frames [][]byte
	err    error
}

func (f fakeExtractor) Extract(ctx context.Context, p string, iv float64, mx int) (*ai.ExtractedFrames, error) {
	if f.err != nil { return nil, f.err }
	paths := make([]string, 0, len(f.frames))
	for i, frame := range f.frames {
		path := filepath.Join(f.dir, fmt.Sprintf("frame_%04d.jpg", i))
		if err := os.WriteFile(path, frame, 0o644); err != nil { return nil, err }
		paths = append(paths, path)
	}
	return &ai.ExtractedFrames{Paths: paths}, nil
}
```

唯一填 frames 的调用点 `worker_test.go:303` 改 `fakeExtractor{dir: t.TempDir(), frames: [][]byte{frameRed, frameRed, frameBlue}}`；其余 `fakeExtractor{}`/`{err: …}` 调用点不变（imports 补 `path/filepath`、`fmt`）。

- [ ] **Step 7: 适配既有抽帧集成测试**（改签名漏了它们就编译不过）：
  - `frames_test.go:61` `ExtractsFramesInTimeOrder`：`extracted, err := …Extract(…)` 成功后紧跟 `defer extracted.Cleanup()`（注册晚于泄漏检查的 `t.Cleanup`，LIFO 保证先删目录再拍快照），再把 `extracted.Paths` 逐个 `os.ReadFile` 读回 `frames [][]byte`，后续 JPEG 魔数/宽度/亮度递增断言原样保留。
  - `frames_test.go:108` `CallerDeadlineIsNotBlamedOnInternalTimeout`：已是 `_, err :=`，新签名直接编译通过，无需改（错误路径返回 nil，无泄漏）。
  - `diff_test.go:156` `TestFFmpegExtractor_MissingBinaryOrZeroFrames`：`frames` 改 `extracted *ai.ExtractedFrames`，`frames != nil` 改 `extracted != nil`，失败消息里的 `frameLens(frames)` 删掉（类型不再是 [][]byte）。

- [ ] **Step 8: 跑测试确认通过** — `go test ./internal/service/ai/ -v`。预期：全包 PASS（此时识别仍一次性喂全部帧，worker 测试 callCount 断言不变；视频 FrameCount=2 不变；`ExtractsFramesInTimeOrder` 泄漏检查仍绿）。

- [ ] **Step 9: Commit**

```bash
git add internal/service/ai/ports.go internal/service/ai/ports_internal_test.go internal/service/ai/diff.go internal/service/ai/diff_test.go internal/service/ai/frames.go internal/service/ai/frames_test.go internal/service/ai/recognize.go internal/service/ai/worker_test.go
git commit -m "refactor(ai): 近似帧过滤流式化, 与所有保留帧比对并均匀薄化; 抽帧接口落盘化"
```

---

### Task 4: 全片分镜覆盖抽帧

**Files:**
- Rewrite: `internal/service/ai/frames.go`
- Modify: `internal/service/ai/frames_test.go`

**Interfaces:**
- Consumes: `thinUniform`（Task 1）、`ExtractedFrames`/`FrameExtractor`（Task 3）。
- Produces（包内私有，签名钉死）：
  - `func probeDuration(ctx context.Context, videoPath string) float64` — ffprobe 失败/缺失/解析失败/≤0 一律返回 0 并 log，不返回 error。
  - `func adaptiveInterval(configInterval, duration float64, maxFrames int) float64` — `duration<=0 → configInterval`；否则 `max(configInterval, duration/float64(maxFrames*10))`。
  - `func extractTimeout(duration float64) time.Duration` — `duration<=0 → 120s`；否则 clamp(`duration×2`, 120s, 30min)。常量 `extractTimeoutFloor = 120*time.Second`、`extractTimeoutCap = 30*time.Minute`（取代旧 `extractTimeout`）。
  - `func parseCandidateTimes(showinfo []byte) []float64` — 正则 `pts_time:([0-9]+(?:\.[0-9]+)?)` 按出现顺序提取。
  - `func candidateTimes(ctx context.Context, videoPath string, interval float64) ([]float64, error)`。
  - `sceneCutThreshold = 0.3` 常量。

- [ ] **Step 1: 写失败的纯函数测试** — 新建包内文件 `frames_internal_test.go`（`package ai`——`package ai_test` 访问不到私有函数，文件头注释说明原因）：

```go
package ai

import (
	"testing"
	"time"
)

func TestAdaptiveInterval(t *testing.T) {
	cases := []struct{ config, duration float64; maxFrames int; want float64 }{
		{1, 0, 10, 1},        // ffprobe 失败 → 配置间隔
		{1, 6, 5, 1},         // 6/50=0.12 < 1 → 配置间隔
		{0.1, 6, 1, 0.6},     // 6/10=0.6 > 0.1 → 拉伸覆盖全片
		{1, 7200, 10, 72},    // 2 小时电影 / 100 候选 = 72s
	}
	for _, c := range cases {
		if got := adaptiveInterval(c.config, c.duration, c.maxFrames); got != c.want {
			t.Errorf("adaptiveInterval(%v,%v,%d)=%v, want %v", c.config, c.duration, c.maxFrames, got, c.want)
		}
	}
}

func TestExtractTimeout(t *testing.T) {
	cases := []struct{ duration float64; want time.Duration }{
		{0, 120 * time.Second},
		{30, 120 * time.Second},        // 60s < floor
		{300, 600 * time.Second},
		{100000, 30 * time.Minute},     // 封顶
	}
	for _, c := range cases {
		if got := extractTimeout(c.duration); got != c.want {
			t.Errorf("extractTimeout(%v)=%v, want %v", c.duration, got, c.want)
		}
	}
}

func TestParseCandidateTimes(t *testing.T) {
	stderr := []byte("[Parsed_showinfo_1 @ 0x1] n:0 pts:0 pts_time:0 fmt:yuv420p\n" +
		"noise line without timestamp\n" +
		"[Parsed_showinfo_1 @ 0x1] n:1 pts:20000 pts_time:2.5 fmt:yuv420p\n" +
		"[Parsed_showinfo_1 @ 0x1] n:2 pts:40000 pts_time:4 fmt:yuv420p\n")
	got := parseCandidateTimes(stderr)
	want := []float64{0, 2.5, 4}
	if len(got) != len(want) { t.Fatalf("parseCandidateTimes=%v, want %v", got, want) }
	for i := range got { if got[i] != want[i] { t.Fatalf("parseCandidateTimes=%v, want %v", got, want) } }
}
```

- [ ] **Step 2: 跑测试确认失败** — `go test ./internal/service/ai/ -run 'TestAdaptiveInterval|TestExtractTimeout|TestParseCandidateTimes' -v`。预期 FAIL：undefined。

- [ ] **Step 3: 实现纯函数** — `frames.go` 按 Interfaces 签名实现。`probeDuration`：`exec.LookPath("ffprobe")` 失败 → log 返回 0；命令 `ffprobe -v error -show_entries format=duration -of default=noprint_wrappers=1:nokey=1 <videoPath>`，10s 超时（`context.WithTimeout`），输出 `strconv.ParseFloat(strings.TrimSpace(...), 64)`，任何错误 → log 返回 0。

- [ ] **Step 4: 跑纯函数测试确认通过** — 同 Step 2。预期 PASS。

- [ ] **Step 5: 写失败的集成测试** — `frames_test.go` 追加（沿用 `t.Skip` 无 ffmpeg 模式；泄漏检查由既有 `ExtractsFramesInTimeOrder` 承担，新测试用 `defer extracted.Cleanup()` 自清）：

```go
// writeCutClip 合成红2s→蓝2s→绿2s 硬切片段(6s, 128x96, 10fps)。
func writeCutClip(t *testing.T, path string) {
	t.Helper()
	cmd := exec.Command("ffmpeg", "-hide_banner", "-loglevel", "error", "-y",
		"-f", "lavfi", "-i", "color=c=red:s=128x96:d=2:r=10",
		"-f", "lavfi", "-i", "color=c=blue:s=128x96:d=2:r=10",
		"-f", "lavfi", "-i", "color=c=green:s=128x96:d=2:r=10",
		"-filter_complex", "[0:v][1:v][2:v]concat=n=3:v=1:a=0",
		"-pix_fmt", "yuv420p", "-c:v", "mpeg4", path)
	var stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stderr, &stderr
	if err := cmd.Run(); err != nil { t.Fatalf("generate cut clip: %v, ffmpeg 报错: %s", err, strings.TrimSpace(stderr.String())) }
}

func TestFFmpegExtractor_SceneCutsCoveredEvenWithCoarseInterval(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil { t.Skip("no ffmpeg") }
	videoPath := filepath.Join(t.TempDir(), "cuts.mp4")
	writeCutClip(t, videoPath)
	// 配置间隔 5s: 周期采样只取 t=0,5, 蓝段(2-4s)全靠硬切检测捕获。
	extracted, err := ai.NewFFmpegExtractor().Extract(context.Background(), videoPath, 5, 10)
	if err != nil { t.Fatalf("Extract: %v", err) }
	defer extracted.Cleanup()
	if len(extracted.Paths) < 3 { t.Fatalf("候选 %d 帧, want ≥3 (红/蓝/绿分镜都要覆盖)", len(extracted.Paths)) }
	// 平均亮度: 红≈76, 蓝≈29, 绿≈150。蓝段出现 = 硬切检测生效(周期采样漏掉它)。
	hasBlue := false
	for _, p := range extracted.Paths {
		data, err := os.ReadFile(p)
		if err != nil { t.Fatalf("read frame: %v", err) }
		img, _, err := image.Decode(bytes.NewReader(data))
		if err != nil { t.Fatalf("decode frame: %v", err) }
		if meanLevel(img) < 45 { hasBlue = true }
	}
	if !hasBlue { t.Error("没有蓝段帧: 硬切分镜未被捕获, 周期采样 alone 会漏掉 2-4s 的镜头") }
}

func TestFFmpegExtractor_AdaptiveIntervalCoversTail(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil { t.Skip("no ffmpeg") }
	if _, err := exec.LookPath("ffprobe"); err != nil { t.Skip("no ffprobe: 自适应间隔依赖时长探测") }
	videoPath := filepath.Join(t.TempDir(), "ramp.mp4")
	writeRampClip(t, videoPath)
	// maxFrames=1 → 候选预算 10; 时长 6s 把 0.1s 拉伸到 0.6s, 候选恰好 10 个铺满全片(无需薄化)。
	extracted, err := ai.NewFFmpegExtractor().Extract(context.Background(), videoPath, 0.1, 1)
	if err != nil { t.Fatalf("Extract: %v", err) }
	defer extracted.Cleanup()
	if len(extracted.Paths) != 10 { t.Fatalf("候选 %d 帧, want 恰好 10 (间隔拉伸后无需薄化)", len(extracted.Paths)) }
	last, err := os.ReadFile(extracted.Paths[len(extracted.Paths)-1])
	if err != nil { t.Fatalf("read last frame: %v", err) }
	img, _, err := image.Decode(bytes.NewReader(last))
	if err != nil { t.Fatalf("decode last frame: %v", err) }
	// 亮度即时间戳: 拉伸后尾帧在 t≈5.4s(亮度≈229)。若间隔没被拉伸, 0.1s 会产 ~60 候选,
	// 薄化尾帧落在 t=5.9s(亮度≈251)——上限 240 把「自适应生效」与「薄化兜底」区分开。
	if level := meanLevel(img); level < 200 || level > 240 {
		t.Errorf("尾帧平均亮度 %.1f, want (200,240): 自适应间隔必须把采样铺到全片", level)
	}
}

func TestFFmpegExtractor_CandidatesThinnedToBudgetStillCoverTail(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil { t.Skip("no ffmpeg") }
	videoPath := filepath.Join(t.TempDir(), "cuts.mp4")
	writeCutClip(t, videoPath)
	// 必须用硬切片段: ramp 片段的候选会被自适应间隔拉回预算内, 永远触发不了薄化。
	// 有 ffprobe: 间隔 0.6s → 10 周期候选 + 2 硬切(2s/4s) = 12 > 预算 10 → 薄化。
	// 无 ffprobe: 间隔停在 0.1s → ~60 候选, 同样薄化到 10——本测试不 skip ffprobe,
	// 顺带验证降级路径。两种情况尾帧都必须落在绿段(4-6s)。
	extracted, err := ai.NewFFmpegExtractor().Extract(context.Background(), videoPath, 0.1, 1)
	if err != nil { t.Fatalf("Extract: %v", err) }
	defer extracted.Cleanup()
	if len(extracted.Paths) != 10 { t.Fatalf("候选 %d 帧, want 恰好 10 (薄化到预算)", len(extracted.Paths)) }
	last, err := os.ReadFile(extracted.Paths[len(extracted.Paths)-1])
	if err != nil { t.Fatalf("read last frame: %v", err) }
	img, _, err := image.Decode(bytes.NewReader(last))
	if err != nil { t.Fatalf("decode last frame: %v", err) }
	if meanLevel(img) < 140 { t.Errorf("尾帧平均亮度 %.1f, want >140(绿段≈150): 薄化必须覆盖全片含尾段", meanLevel(img)) }
}
```

（都在 `frames_test.go`，imports 补 `os`、`image`。**不写 `TestProbeDuration`**：包内测试访问不到外部包的 `writeRampClip`，且 ffprobe 正路径已由 `AdaptiveIntervalCoversTail` 间接验证——它 skip 无 ffprobe 的机器，有时长才能拉伸间隔；失败回退由纯函数 `TestAdaptiveInterval`(duration=0) 与「任何失败返回 0」的实现规则钉住。）既有 `ExtractsFramesInTimeOrder`/`CallerDeadlineIsNotBlamedOnInternalTimeout`/`MissingBinaryOrZeroFrames` 保留：ramp 片段 128px < 768 scale 不改宽度；1ms caller deadline 仍归因调用方（新流程里 probeDuration 先因 ctx 过期失败返回 0，candidateTimes 随即被杀，归因分支不变）；缺失文件仍报错。`ExtractsFramesInTimeOrder` 的泄漏检查照旧生效。

- [ ] **Step 6: 跑集成测试确认失败** — `go test ./internal/service/ai/ -run 'TestFFmpegExtractor_SceneCuts|TestFFmpegExtractor_Adaptive|TestFFmpegExtractor_CandidatesThinned' -v`。预期 FAIL（旧 Extract 固定间隔只取开头：蓝段缺失；尾帧是开头暗帧/红段，亮度断言失败）。

- [ ] **Step 7: 重写 Extract** — `frames.go`，删除 Task 3 的临时适配与旧 `fps=`/`-frames:v maxFrames*10` 逻辑。流程：
  1. `exec.LookPath("ffmpeg")` 缺失 → `errors.New("未找到 ffmpeg,请先安装")`。
  2. `duration := probeDuration(ctx, videoPath)`；`callerCtx := ctx`；`ctx, cancel := context.WithTimeout(ctx, extractTimeout(duration))`。
  3. `times, err := candidateTimes(ctx, videoPath, adaptiveInterval(intervalSeconds, duration, maxFrames))`；错误归因沿用旧模式（callerCtx deadline → 「调用方设定的时限」；内部 deadline → 报 `extractTimeout(duration)`；其余 → `ffmpegError` 包 stderr）。
  4. `budget := maxFrames*10`；`len(times) > budget` → `thinUniform` 选索引重建 times。
  5. `tmpDir, err := os.MkdirTemp("", "tagfile-ai-")`；命名返回 err + `defer func(){ if err != nil { os.RemoveAll(tmpDir) } }()`。
  6. 逐 ts：`ffmpeg -hide_banner -loglevel error -y -ss <ts> -i <videoPath> -frames:v 1 -vf scale='min(768,iw)':-2 -q:v 2 <tmpDir/frame_%04d.jpg>`（ts 用 `strconv.FormatFloat(ts,'f',-1,64)`；`-ss` 在 `-i` 前 = fast seek 帧级精确）。每个命令 `exec.CommandContext(ctx, …)`，stderr 收进 buffer，失败 → `ffmpegError(fmt.Sprintf("第 %d 帧(%.3fs)抽取失败", i+1, ts), err, stderr)`；成功后 `os.Stat` 校验存在且非空，否则报「没有产出图像」。
  7. 返回 `&ExtractedFrames{Paths: paths, dir: tmpDir}`。
  
  `candidateTimes` 的 `-vf` 参数值（一个 arg，内部单引号是 ffmpeg 表达式转义，必须保留）：
  `select='isnan(prev_selected_t)+gte(t-prev_selected_t,<INTERVAL>)+gt(scene,0.3)',showinfo`
  完整命令：`ffmpeg -hide_banner -loglevel info -i <videoPath> -an -sn -dn -vf <上式> -f null -`。从 stderr 跑 `parseCandidateTimes`；exit 0 但零候选 → `ffmpegError("未能从视频识别任何候选帧", nil, stderr)`。注释写明三层语义：首帧必选、周期层兜底渐变转场与长镜头、scene 层捕获硬切。

  **跨版本纪律**（Global Constraints 白名单）：以上每个参数/表达式函数都是 ffmpeg 4.x 起就有的；`showinfo` 的 `pts_time:` 输出格式各版本稳定。**禁止**顺手加 `-fps_mode vfr` 之类 5.1+ 才有的参数——第一遍 `-f null` 不写帧、第二遍 `-frames:v 1` 单帧，都不需要它。实现与测试都不得引用任何特定本机版本的行为。

- [ ] **Step 8: 跑测试确认通过** — `go test ./internal/service/ai/ -v`。预期：全包 PASS（含既有抽帧/worker 测试）。

- [ ] **Step 9: Commit**

```bash
git add internal/service/ai/frames.go internal/service/ai/frames_test.go internal/service/ai/frames_internal_test.go
git commit -m "feat(ai): 抽帧覆盖全片分镜 — ffprobe 自适应间隔 + ffmpeg 硬切检测两遍走"
```

---

### Task 5: 跨帧合并去重

**Files:**
- Create: `internal/service/ai/merge.go`
- Create: `internal/service/ai/merge_test.go`

**Interfaces:**
- Consumes: `RawTag{Category, Tag, Value string}`（parse.go，已导出）。
- Produces: `func mergeRawTags(raws []RawTag) []RawTag`（包内私有，Task 6 调用）。规则：(1) 三元组精确去重保留首次出现、保序；(2) 同 (Category, Tag) 存在非空 Value 行时，丢弃该 pair 的所有空 Value 行（与顺序无关）；(3) 同标签不同值都保留。空输入 → 空输出。

- [ ] **Step 1: 写失败测试** — `merge_test.go`，`package ai`（私有函数，包内测试）：

```go
package ai

import "testing"

func TestMergeRawTags(t *testing.T) {
	cases := []struct {
		name string
		in   []RawTag
		want []RawTag
	}{
		{"空输入", nil, []RawTag{}},
		{"跨帧精确重复折叠", []RawTag{{"主体", "猫", "橘色"}, {"主体", "猫", "橘色"}}, []RawTag{{"主体", "猫", "橘色"}}},
		{"同标签不同值都保留", []RawTag{{"颜色", "裤子", "黑色"}, {"颜色", "裤子", "白色"}}, []RawTag{{"颜色", "裤子", "黑色"}, {"颜色", "裤子", "白色"}}},
		{"空值让位带值_空在前", []RawTag{{"主体", "猫", ""}, {"主体", "猫", "橘色"}}, []RawTag{{"主体", "猫", "橘色"}}},
		{"空值让位带值_空在后", []RawTag{{"主体", "猫", "橘色"}, {"主体", "猫", ""}}, []RawTag{{"主体", "猫", "橘色"}}},
		{"无带值兄弟时空值保留", []RawTag{{"主体", "猫", ""}}, []RawTag{{"主体", "猫", ""}}},
		{"保序_首次出现", []RawTag{{"主体", "猫", "橘色"}, {"材质", "皮革", ""}, {"主体", "猫", "橘色"}}, []RawTag{{"主体", "猫", "橘色"}, {"材质", "皮革", ""}}},
	}
	for _, c := range cases {
		got := mergeRawTags(c.in)
		if len(got) != len(c.want) { t.Fatalf("%s: got %v, want %v", c.name, got, c.want) }
		for i := range got { if got[i] != c.want[i] { t.Fatalf("%s: got %v, want %v", c.name, got, c.want) } }
	}
}
```

- [ ] **Step 2: 跑测试确认失败** — `go test ./internal/service/ai/ -run TestMergeRawTags -v`。预期 FAIL：undefined。

- [ ] **Step 3: 实现** — `merge.go`，`package ai`。两趟：先扫一遍收 `valued map[[2]string]bool`（非空 Value 的 pair）；再按序遍历，`seen map[[3]string]bool` 跳过重复三元组，`Value=="" && valued[pair]` 跳过空值行，其余 append。注释写明规则与「不同值交审核人」的取舍。

- [ ] **Step 4: 跑测试确认通过** — 同 Step 2。预期 PASS。

- [ ] **Step 5: Commit**

```bash
git add internal/service/ai/merge.go internal/service/ai/merge_test.go
git commit -m "feat(ai): 跨帧建议合并去重 mergeRawTags"
```

---

### Task 6: 逐帧识别 + 部分失败语义

**Files:**
- Modify: `internal/service/ai/recognize.go:76-147`（`recognize`）
- Modify: `internal/service/ai/worker_test.go`

**Interfaces:**
- Consumes: `mergeRawTags`（Task 5）、`FilterSimilarFrames(paths,…)`/`ExtractedFrames.Cleanup`（Task 3）、`VisionClient.Recognize(ctx, prompt, images [][]byte)`（不变）。
- Produces: `recognize` 新语义——每帧一次 Recognize（单图切片）、逐帧 ParseSuggestions；`merged` 为空 → `fmt.Errorf("全部 %d 帧识别失败, 首条原因: %w", len(frames), firstErr)`；否则 merge → 截断（文案见下）→ buildSuggestions → 一次性 CreateSuggestions → done。部分失败 note：`fmt.Sprintf("第 %s 帧识别失败(%d/%d), 首条原因: %v", frameIndexList(failedFrames), len(failedFrames), len(frames), firstErr)`；截断 note：`"模型各帧合并后给出了 %d 条建议, 超过单任务上限 %d 条: 仅写入前 %d 条, 丢弃其余 %d 条"`；两 note 并存用 `"；"` 连接。`frameIndexList([]int) string` 私有 helper，1-based 索引「、」连接。`FrameCount` 仍 = `len(frames)`。

- [ ] **Step 1: 写失败测试** — `worker_test.go` 追加 fake 与五个测试；改既有断言。

```go
// perFrameClient 按调用序号给不同答案/错误, 模拟逐帧识别里某一帧坏掉。
type perFrameClient struct {
	mu        sync.Mutex
	outs      map[int]string // 1-based 调用序号 → 答案; 缺省用 fallback
	errs      map[int]error
	fallback  string
	calls     int
	imgCounts []int // 每次调用收到的图片数, 钉住「一次只喂一帧」
}

func (f *perFrameClient) Recognize(ctx context.Context, p string, imgs [][]byte) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	f.imgCounts = append(f.imgCounts, len(imgs))
	n := f.calls
	if err, ok := f.errs[n]; ok { return "", err }
	if out, ok := f.outs[n]; ok { return out, nil }
	return f.fallback, nil
}

func (f *perFrameClient) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

func (f *perFrameClient) imageCounts() []int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]int(nil), f.imgCounts...)
}

// tagsPayload 生成 count 条「主体/前缀i=值i」的合法答案。
func tagsPayload(prefix string, count int) string {
	var b strings.Builder
	b.WriteString(`{"tags":[`)
	for i := 0; i < count; i++ {
		if i > 0 { b.WriteString(",") }
		fmt.Fprintf(&b, `{"category":"主体","tag":"%s%d","value":"值%d"}`, prefix, i, i)
	}
	b.WriteString("]}")
	return b.String()
}

// threeFrameVideo 注册一个视频并配好三帧互不相同的 fakeExtractor。
func threeFrameVideo(t *testing.T, svc *ai.Service) *model.File {
	t.Helper()
	svc.Extractor = fakeExtractor{dir: t.TempDir(), frames: [][]byte{
		encodeFrame(t, tintRed, bandBrightness(0b0000_1111)),
		encodeFrame(t, tintBlue, bandBrightness(0b1111_0000)),
		encodeFrame(t, tintGreen, bandBrightness(0b0011_0011)),
	}}
	return registerMedia(t, "v.mp4", []byte("dummy video content"))
}
```

测试（都用 `workerFixture`+`seedTagLibrary`+`captureLogs`+`defer svc.StopWorker()` 既有模式）：
1. `TestWorker_PerFrameCallsOneImageEach`：`threeFrameVideo`，`perFrameClient{fallback: matchedJSON}`，入队启动等 done。断言 `client.callCount()==3`、`imageCounts()` 每个元素都是 1（逐帧喂图的直接证据）、`FrameCount==3`、pending 1 条（三帧同答案合并去重）。
2. `TestWorker_PartialFrameFailureStillDone`：`threeFrameVideo`，`perFrameClient{fallback: matchedJSON, errs: map[int]error{2: errors.New("模型抽风")}}`。断言 done、pending 1 条、`task.Error` 含「第 2 帧」「1/3」「模型抽风」、`callCount()==3`（失败帧之后的帧继续跑）。
3. `TestWorker_AllFramesFail`：`threeFrameVideo`，`fakeClient{err: errors.New("模型不可达")}`。断言 failed、pending 0、Error 含「全部 3 帧识别失败」与「模型不可达」。
4. `TestWorker_MergedSetTruncatedAfterDedupe`：**不用 `threeFrameVideo`**（它钉死三帧），内联两帧 extractor：`svc.Extractor = fakeExtractor{dir: t.TempDir(), frames: [][]byte{encodeFrame(t, tintRed, bandBrightness(0b0000_1111)), encodeFrame(t, tintBlue, bandBrightness(0b1111_0000))}}` + `registerMedia(t, "v.mp4", []byte("dummy video content"))`；`perFrameClient{outs: map[int]string{1: tagsPayload("A", 40), 2: tagsPayload("B", 40)}}`。断言 done、pending 恰好 50、Error 含「合并后给出了 80 条」与「丢弃其余 30 条」（证明先合并去重再截断）。
5. `TestWorker_BadJSONOnOneFrameIsolated`：`threeFrameVideo`，`perFrameClient{fallback: matchedJSON, outs: map[int]string{2: "这不是 JSON"}}`。断言 done、pending 1 条、Error 含「第 2 帧」。

既有断言更新：`TestWorker_ProcessesQueuedTaskAndStoresPendingSuggestions` 的 `callCount` 2→3（图片 1 次 + 视频 2 帧各 1 次），注释同步；`TestWorker_OversizedSuggestionSetIsTruncatedAndReported` 与 `TestWorker_CutsTheAnswerBeforeMatchingIt` 的 payload 构造可换 `tagsPayload("标签", 60/200)`（断言不变，「上限 50 条」「丢弃其余 10 条」子串在新文案里保留）。`TestWorker_RecordsFailureAndAllowsRetry` 图片半段 Error 断言「模型不可达」仍成立（新文案包裹首条原因）。

- [ ] **Step 2: 跑测试确认失败** — `go test ./internal/service/ai/ -run 'TestWorker_PerFrame|TestWorker_PartialFrame|TestWorker_AllFrames|TestWorker_MergedSet|TestWorker_BadJSON' -v`。预期 FAIL：旧 recognize 一次喂全部帧（callCount=1、无部分失败语义）。

- [ ] **Step 3: 重写 recognize 循环** — `recognize.go`。file/settings/client/prompt 构建照旧（prompt 一次）。`loadFrames` 之后：

```go
var merged []RawTag
var failedFrames []int
var firstErr error
for i, frame := range frames {
	output, err := client.Recognize(ctx, prompt, [][]byte{frame})
	if err == nil {
		var raws []RawTag
		raws, err = ParseSuggestions(output)
		if err == nil {
			merged = append(merged, raws...)
			continue
		}
	}
	failedFrames = append(failedFrames, i+1)
	if firstErr == nil { firstErr = err }
}
if len(merged) == 0 {
	return fmt.Errorf("全部 %d 帧识别失败, 首条原因: %w", len(frames), firstErr)
}
merged = mergeRawTags(merged)
```

截断块照旧但作用于 merged、文案换 Interfaces 所钉。`buildSuggestions`/`CreateSuggestions` 不动。done 更新：

```go
update := s.aiRepo.UpdateTask(task).SetStatus(TaskDone).SetFrameCount(len(frames))
var notes []string
if len(failedFrames) > 0 {
	notes = append(notes, fmt.Sprintf("第 %s 帧识别失败(%d/%d), 首条原因: %v",
		frameIndexList(failedFrames), len(failedFrames), len(frames), firstErr))
}
if truncated != "" { notes = append(notes, truncated) }
if len(notes) > 0 { update = update.SetError(strings.Join(notes, "；")) }
return update.Exec()
```

`frameIndexList` 用 `strconv.Itoa`+`strings.Join`。注释写明：部分成功沿用「done + Error 说明」先例；原子写入保证不变；每帧独立超时由 client 自带。`maxTaskSuggestions` 注释补一句「作用于跨帧合并后的集合」。

- [ ] **Step 4: 跑测试确认通过** — `go test ./internal/service/ai/ -v`。预期：全包 PASS。

- [ ] **Step 5: Commit**

```bash
git add internal/service/ai/recognize.go internal/service/ai/worker_test.go
git commit -m "feat(ai): 逐帧送模型识别, 跨帧合并去重, 部分帧失败不再炸整任务"
```

---

### Task 7: 配置文案 + 容器镜像 + CHANGELOG + 版本 + 全量验证

**Files:**
- Modify: `internal/service/ai/config_schema.go:39-85`
- Modify: `internal/constant/version.go:3`
- Modify: `docs/CHANGELOG.md`
- Modify: `Dockerfile`（运行层 scratch → alpine + ffmpeg）
- Modify: `cmd/webui/constant/version.go:3`
- Modify: `docs/CHANGELOG_webui.md`

**Interfaces:**
- Consumes: 全部前序任务。
- Produces: 设置页三个键的新描述（schema 驱动渲染，无前端代码改动）；CoreVersion 0.9.0；WebuiVersion 5.12.0；运行镜像自带 ffmpeg/ffprobe 与可写 `/tmp`。

- [ ] **Step 1: 改描述文案** — `config_schema.go` 三处 Description 精确替换：
  - `KeyFrameInterval`：`"视频最小抽帧间隔（秒）；长视频会自动放大间隔以覆盖全片，分镜切换点始终额外采样"`
  - `KeyDiffThreshold`：`"近似帧过滤阈值（与所有已保留帧的 dHash 汉明距离均小于该值的帧被丢弃）"`
  - `KeyMaxFrames`：`"单个视频最多保留的帧数（逐帧送模型识别）"`
  
  Label/Default/Constraint/Shift 全部不动。`config_schema_test.go` 只查 Description 非空，无需改。

- [ ] **Step 2: 跑 schema 测试** — `go test ./internal/service/ai/ -run TestRegisterAI -v`（config_schema_test.go 的五个测试都以此前缀开头）。预期 PASS。

- [ ] **Step 3: bump 版本 + CHANGELOG** — `internal/constant/version.go`：`const CoreVersion = "0.9.0"`。`docs/CHANGELOG.md` 在 `## [0.8.0]` 之上插入（英文行紧接中文行，条目间空行）：

```markdown
## [0.9.0] - 2026-11-19

### Changed / 变更

- Video recognition now feeds frames to the vision model one at a time instead of stuffing every frame into a single request: each frame gets its own call and parse, suggestions are merged and deduplicated across frames (exact category/tag/value triples collapse; an empty value yields to a valued sibling), and a task succeeds as long as at least one frame succeeded — failed frames are summarised in the task's Error while the suggestion set is still written in one atomic insert.
- 视频识别改为逐帧喂视觉模型，不再把全部帧塞进一个请求：每帧独立调用与解析，建议跨帧合并去重（分类/标签/值三元组精确折叠，空值让位同标签的带值行），至少一帧成功任务即完成——失败帧汇总写入任务 Error，建议集仍一次性原子入库。

- Frame extraction now covers the whole video instead of only its opening: ffprobe reads the duration, the sampling interval stretches to `max(configured interval, duration/(maxFrames*10))` (the config key is now a *minimum* interval), and ffmpeg's built-in hard-cut detection (`select scene>0.3`) adds every shot boundary as a candidate. Candidates are thinned uniformly to the frame budget, extracted at exact timestamps (`-ss`, scaled to ≤768px), and the extraction deadline scales with duration (max(120s, 2×duration), capped at 30min).
- 抽帧覆盖全片而非只取开头：ffprobe 读时长，采样间隔拉伸为 `max(配置间隔, 时长/(maxFrames*10))`（配置键语义变为「最小间隔」），ffmpeg 内置硬切检测（`select scene>0.3`）把每个分镜切换点纳入候选。候选均匀薄化到帧预算后按时间戳定点抽取（`-ss`，分辨率压到 ≤768px），抽帧超时随时长伸缩（max(120s, 2×时长)，封顶 30min）。

- The near-duplicate filter hashes frames with area-averaged 9×8 dHash (robust to JPEG artefacts and impulse noise), compares each frame against *all* kept frames (a cut back to an earlier scene is deduplicated), no longer stops early at the frame cap, and thins uniformly across time when over budget; it streams from disk so only the final ≤maxFrames frames are held in memory.
- 近似帧过滤改用区域平均 9×8 dHash（抗 JPEG 伪影与脉冲噪声），每帧与所有已保留帧比对（切回早先场景会被去重），不再凑满帧数就早停，超预算时按时间均匀薄化；过滤流式读盘，内存只留最终 ≤maxFrames 帧。
```

- [ ] **Step 4: 修 Dockerfile + webui 版本/CHANGELOG** — 现运行层 `FROM scratch` 没有 ffmpeg/ffprobe，也没有 `/tmp`（`os.MkdirTemp` 同样会失败），AI 识别在容器里必然报「未找到 ffmpeg」。改为：

```dockerfile
FROM golang:1.25-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o tagfile-webui ./cmd/webui

FROM alpine:3.22

RUN apk add --no-cache ffmpeg ca-certificates tzdata

COPY --from=builder /src/tagfile-webui /tagfile-webui

EXPOSE 8088

VOLUME ["/data"]

ENTRYPOINT ["/tagfile-webui", "--source", "/data"]
```

  要点：alpine 的 `ffmpeg` 包同时附带 ffprobe；alpine 自带 `/tmp`；证书/时区改由 apk 提供，构建层的 `apk add tzdata` 与两条 COPY（certs/zoneinfo）删除。alpine 取**实现时仍在维护的稳定 minor**（3.22 起），不钉 ffmpeg 版本——白名单参数对任何现行 alpine 的 ffmpeg（≥4.x）都成立。`cmd/webui/constant/version.go`：`const WebuiVersion = "5.12.0"`。`docs/CHANGELOG_webui.md` 在 `## [5.11.0]` 之上插入：

```markdown
## [5.12.0] - 2026-11-19

### Fixed / 修复

- Video AI recognition never worked in the published container image: the runtime stage was `FROM scratch`, which ships neither ffmpeg/ffprobe nor a writable `/tmp`, so every video task failed with 「未找到 ffmpeg」. The runtime is now Alpine with the distro `ffmpeg` package (ffprobe included) plus ca-certificates and tzdata; the build stage is unchanged and the application code is untouched.
- 已发布的容器镜像里视频 AI 识别从来跑不起来：运行层是 `FROM scratch`，既没有 ffmpeg/ffprobe 也没有可写 `/tmp`，视频任务只会报「未找到 ffmpeg」。运行层改为 Alpine 并安装发行版 `ffmpeg` 包（附带 ffprobe）及 ca-certificates、tzdata；构建层不变，应用代码零改动。
```

- [ ] **Step 5: 全量验证** — `make test`。预期：全部 PASS；有 ffmpeg/ffprobe 的环境集成测试真跑，没有的机器按既有模式 skip——**两种结果都算通过，不预设执行机器的 ffmpeg 版本**。若本机有 podman/docker：`podman build -t tagfile-webui-check .` 后 `podman run --rm tagfile-webui-check ffmpeg -version` 与 `podman run --rm tagfile-webui-check ffprobe -version` 确认两二进制在镜像里；无容器引擎则跳过此步，在交付说明里注明镜像待部署验证。

- [ ] **Step 6: Commit**

```bash
git branch --show-current   # 确认 feat/ai-per-frame-scene-coverage
git add internal/service/ai/config_schema.go internal/constant/version.go docs/CHANGELOG.md Dockerfile cmd/webui/constant/version.go docs/CHANGELOG_webui.md
git commit -m "chore(ai): 配置文案同步新语义, 运行镜像自带 ffmpeg/ffprobe, CHANGELOG 0.9.0 / webui 5.12.0"
```

---

## Self-Review 记录

- **Spec 覆盖**：重设计决策 1-6 → Task 6/5/6/2+3/4/4；决策 7（不做项）无任务，符合预期；决策 8（跨版本/容器）→ Global Constraints 白名单 + Task 7 Dockerfile。原始 spec 的审核门/单 worker/重启语义/配置机制由 Global Constraints 守住，零改动文件清单已列。
- **类型一致性**：`thinUniform(n,budget int) []int`、`FilterSimilarFrames(paths []string,threshold,maxFrames int) ([][]byte,error)`、`ExtractedFrames{Paths []string; dir string}`+`Cleanup()`、`mergeRawTags([]RawTag) []RawTag`、`adaptiveInterval/extractTimeout/parseCandidateTimes/probeDuration` 在各任务 Interfaces 与测试中拼写一致。测试引用的既有 helper（`workerFixture`/`registerMedia`/`newestTask`/`pendingOf`/`waitFor`/`captureLogs`/`matchedJSON`/`writeRampClip`/`meanLevel`/`tagfileTempDirs`）已逐一对照 worker_test.go/frames_test.go/diff_test.go 核实存在。
- **Review Focus**：六条各归 Task 4/1+4/6/6/3/7，测试或验证步骤已钉。
- **比例**：代码块集中在测试与 ffmpeg 参数（实现者无法自行决定的部分）；实现步骤只给签名与算法规则，未转录函数体。
- **自审修正（已并入正文）**：
  1. 脉冲噪声测试原用 JPEG 编码——JPEG 自身低通会抹平单像素脉冲，旧哈希也翻不了位，测试无法确定性失败；改 PNG 无损（Task 2）。
  2. 候选薄化集成测试原用 ramp 片段——自适应间隔会把候选拉回预算内，薄化永远不触发；改硬切片段（12 候选 > 预算 10），并断言尾帧绿段（Task 4）。
  3. `TestProbeDuration` 原放包内测试文件却调用外部包的 `writeRampClip`，编译不过；删除，ffprobe 正路径由 `AdaptiveIntervalCoversTail`（skip 无 ffprobe 机器）间接验证（Task 4）。
  4. Task 3 改 `Extract` 签名原漏了三个既有集成测试的适配（`ExtractsFramesInTimeOrder`/`CallerDeadline…`/`MissingBinaryOrZeroFrames`，后者在 diff_test.go）；补 Step 7 逐一说明。
  5. Task 3 临时适配原打算把 ffmpeg 已写盘的帧再读回字节二次写盘；改为直接返回路径（`readFramePaths`），错误路径命名返回 + defer 清理。
  6. `perFrameClient` 补 `imgCounts` 记录，「一次只喂一帧」有了直接断言；合并截断测试原复用钉死三帧的 `threeFrameVideo`，改内联两帧 extractor（Task 6）。
  7. 按需求方指令去除一切本机 ffmpeg 版本预设：Tech Stack/Global Constraints 改白名单制（4.x 起就有的参数，刻意不用 `-fps_mode`），集成测试断言只依赖稳定语义、无二进制即 skip（Task 4/7）。
  8. 排查发现运行镜像 `FROM scratch` 根本没有 ffmpeg/ffprobe/`/tmp`——容器是最终部署形态，新增 Task 7 Step 4 换 alpine 运行层，记 webui CHANGELOG 并 bump 5.12.0。