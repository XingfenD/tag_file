package ai_test

import (
	"bytes"
	"context"
	"errors"
	"image"
	_ "image/jpeg" // registers the decoder for the JPEGs ffmpeg writes
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/XingfenD/tag_file/internal/service/ai"
)

// TestFFmpegExtractor_ExtractsFramesInTimeOrder drives Extract over a clip that
// ffmpeg itself synthesises, so the happy path needs no checked-in media: a box
// without ffmpeg skips, a box with it runs the real binary.
//
// The clip is a brightness ramp (each frame is one solid grey whose level grows
// with its own timestamp) rather than testsrc, because the only ordering signal
// testsrc offers is the printed frame counter, which no assertion can read
// without OCR. Here the mean brightness of a frame *is* its position in time, so
// the order Extract returns is checked arithmetically: were the frames read back
// in the wrong order, or were the bytes truncated, the sequence stops growing.
func TestFFmpegExtractor_ExtractsFramesInTimeOrder(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("no ffmpeg")
	}

	// Extract creates its frame directory under os.TempDir(), which on Unix is
	// $TMPDIR: pointed at a private directory here, so the leak check below only
	// ever sees what this run did. Against the shared /tmp a recognition worker of
	// a live tagfile instance (or another test) writes and removes tagfile-ai-*
	// directories concurrently, and the before/after comparison could fail or pass
	// by luck.
	t.Setenv("TMPDIR", t.TempDir())

	// Extract removes the directory it created again; anything left behind by a
	// future edit that drops the cleanup shows up here as a new entry.
	// Snapshotted before the call and compared on cleanup, so the check also runs
	// when an assertion below fails.
	before := tagfileTempDirs(t)
	t.Cleanup(func() {
		after := tagfileTempDirs(t)
		t.Logf("tagfile-ai-* dirs in %s: %d before, %d after the run", os.TempDir(), len(before), len(after))
		for _, dir := range after {
			if !containsString(before, dir) {
				t.Errorf("Extract left %s behind", filepath.Join(os.TempDir(), dir))
			}
		}
	})

	videoPath := filepath.Join(t.TempDir(), "ramp.mp4")
	writeRampClip(t, videoPath)

	extracted, err := ai.NewFFmpegExtractor().Extract(context.Background(), videoPath, 1, 5)
	if err != nil {
		t.Fatalf("Extract(%s): %v", videoPath, err)
	}
	// Extract hands the frame directory over to the caller on success; this
	// Cleanup runs when the test function returns, i.e. before the t.Cleanup
	// leak-check above it, so the snapshot still sees the directory gone.
	defer extracted.Cleanup()

	frames := make([][]byte, 0, len(extracted.Paths))
	for _, path := range extracted.Paths {
		frame, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read frame file %s: %v", path, err)
		}
		frames = append(frames, frame)
	}
	if len(frames) < 2 {
		t.Fatalf("Extract returned %d frames, want at least 2 for a 6 s clip sampled every 1 s: %v", len(frames), frameLens(frames))
	}

	levels := make([]float64, 0, len(frames))
	for i, frame := range frames {
		if !bytes.HasPrefix(frame, []byte{0xff, 0xd8, 0xff}) {
			t.Fatalf("frame %d does not start with the JPEG magic bytes: % x", i, min(len(frame), 3))
		}
		img, _, err := image.Decode(bytes.NewReader(frame))
		if err != nil {
			t.Fatalf("frame %d is not a decodable JPEG (%d bytes): %v", i, len(frame), err)
		}
		if got := img.Bounds().Dx(); got != rampWidth {
			t.Errorf("frame %d is %d px wide, want the %d px of the source clip", i, got, rampWidth)
		}
		levels = append(levels, meanLevel(img))
	}

	for i := 1; i < len(levels); i++ {
		if levels[i-1] >= levels[i] {
			t.Fatalf("frames are not in time order: frame %d has mean brightness %.1f, frame %d has %.1f",
				i-1, levels[i-1], i, levels[i])
		}
	}
}

// TestFFmpegExtractor_CallerDeadlineIsNotBlamedOnInternalTimeout pins the error
// attribution of a run cut short by the *caller's* deadline. One millisecond is
// never enough for ffmpeg to start, let alone decode a clip, so Extract must name
// the limit the caller set and not blame the internal 120 s ceiling: a user who
// cancelled a task would otherwise be told the extraction was too slow.
func TestFFmpegExtractor_CallerDeadlineIsNotBlamedOnInternalTimeout(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("no ffmpeg")
	}

	videoPath := filepath.Join(t.TempDir(), "ramp.mp4")
	writeRampClip(t, videoPath)

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()

	_, err := ai.NewFFmpegExtractor().Extract(ctx, videoPath, 1, 5)
	if err == nil {
		t.Fatal("Extract with a 1 ms deadline returned no error, want the run cut short")
	}
	if !strings.Contains(err.Error(), "调用方设定的时限") {
		t.Errorf("error %q, want it to report the deadline the caller set", err)
	}
	// 2m0s is how the internal extractTimeout renders: seeing it here would mean
	// the run was blamed on the ceiling it never reached.
	if strings.Contains(err.Error(), "2m0s") {
		t.Errorf("error %q blames the internal 2m0s extraction timeout", err)
	}
	// The deadline stays findable with errors.Is, which is what lets the worker
	// tell a cancelled task apart from a broken video.
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("error %v does not unwrap to context.DeadlineExceeded", err)
	}
}

// rampWidth/rampHeight/rampSeconds describe the synthesized clip; the interval of
// 1 s in Extract yields rampSeconds frames of a clip whose level grows every
// frame, which is what the ordering check above needs.
const (
	rampWidth   = 128
	rampHeight  = 96
	rampSeconds = 6
)

// writeRampClip generates a rampSeconds clip whose frames get steadily brighter
// (level = 255*timestamp/duration) with ffmpeg's own lavfi source, so no media
// file is committed. mpeg4 is libavcodec's built-in encoder, available whatever
// optional external encoders the host ffmpeg carries.
func writeRampClip(t *testing.T, path string) {
	t.Helper()
	cmd := exec.Command("ffmpeg", "-hide_banner", "-loglevel", "error", "-y",
		"-f", "lavfi",
		"-i", "nullsrc=s="+strconv.Itoa(rampWidth)+"x"+strconv.Itoa(rampHeight)+
			":d="+strconv.Itoa(rampSeconds)+":r=10,"+
			"geq=lum='trunc(255*T/"+strconv.Itoa(rampSeconds)+")':cb=128:cr=128",
		"-pix_fmt", "yuv420p", "-c:v", "mpeg4", path)
	var stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stderr, &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("generate test clip: %v, ffmpeg 报错: %s", err, strings.TrimSpace(stderr.String()))
	}
}

// tagfileTempDirs lists the Extract scratch directories currently in os.TempDir().
func tagfileTempDirs(t *testing.T) []string {
	t.Helper()
	entries, err := os.ReadDir(os.TempDir())
	if err != nil {
		t.Fatalf("read %s: %v", os.TempDir(), err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "tagfile-ai-") {
			names = append(names, entry.Name())
		}
	}
	return names
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

// meanLevel is the Rec.601 luma average of a decoded frame, i.e. how bright it
// looks, which for the ramp clip is a direct read-out of its timestamp.
func meanLevel(img image.Image) float64 {
	rect := img.Bounds()
	var sum, count uint64
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			// 16-bit premultiplied channels scaled back down; the weights sum to
			// 1000 so the division stays exact enough for a comparison.
			r, g, b, _ := img.At(x, y).RGBA()
			sum += uint64((299*r + 587*g + 114*b) / 1000 / 256)
			count++
		}
	}
	if count == 0 {
		return 0
	}
	return float64(sum) / float64(count)
}

// writeCutClip 合成红2s→蓝2s→lime2s 硬切片段(6s, 128x96, 10fps)。绿段必须用
// lime(0,255,0, 亮度≈150): CSS 的 green 是 (0,128,0), 亮度≈76 与红段不可分辨。
func writeCutClip(t *testing.T, path string) {
	t.Helper()
	cmd := exec.Command("ffmpeg", "-hide_banner", "-loglevel", "error", "-y",
		"-f", "lavfi", "-i", "color=c=red:s=128x96:d=2:r=10",
		"-f", "lavfi", "-i", "color=c=blue:s=128x96:d=2:r=10",
		"-f", "lavfi", "-i", "color=c=lime:s=128x96:d=2:r=10",
		"-filter_complex", "[0:v][1:v][2:v]concat=n=3:v=1:a=0",
		"-pix_fmt", "yuv420p", "-c:v", "mpeg4", path)
	var stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stderr, &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("generate cut clip: %v, ffmpeg 报错: %s", err, strings.TrimSpace(stderr.String()))
	}
}

func TestFFmpegExtractor_SceneCutsCoveredEvenWithCoarseInterval(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("no ffmpeg")
	}
	videoPath := filepath.Join(t.TempDir(), "cuts.mp4")
	writeCutClip(t, videoPath)
	// 配置间隔 5s: 周期采样只取 t=0,5, 蓝段(2-4s)全靠硬切检测捕获。
	extracted, err := ai.NewFFmpegExtractor().Extract(context.Background(), videoPath, 5, 10)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	defer extracted.Cleanup()
	if len(extracted.Paths) < 3 {
		t.Fatalf("候选 %d 帧, want ≥3 (红/蓝/绿分镜都要覆盖)", len(extracted.Paths))
	}
	// 平均亮度: 红≈76, 蓝≈29, 绿≈150。蓝段出现 = 硬切检测生效(周期采样漏掉它)。
	hasBlue := false
	for _, p := range extracted.Paths {
		data, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("read frame: %v", err)
		}
		img, _, err := image.Decode(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("decode frame: %v", err)
		}
		if meanLevel(img) < 45 {
			hasBlue = true
		}
	}
	if !hasBlue {
		t.Error("没有蓝段帧: 硬切分镜未被捕获, 周期采样 alone 会漏掉 2-4s 的镜头")
	}
}

func TestFFmpegExtractor_AdaptiveIntervalCoversTail(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("no ffmpeg")
	}
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("no ffprobe: 自适应间隔依赖时长探测")
	}
	videoPath := filepath.Join(t.TempDir(), "ramp.mp4")
	writeRampClip(t, videoPath)
	// maxFrames=1 → 候选预算 10; 时长 6s 把 0.1s 拉伸到 0.6s, 候选铺满全片(无需薄化)。
	// 这里不钉绝对时间戳和固定亮度窗口: select 的 gte(t-prev_selected_t,0.6) 在
	// 10fps 的 PTS 上有浮点漂移(本机实测选中帧 n=[0,6,12,19,26,32,38,44,51,58],
	// 链尾 5.8 而非纸面的 5.4), 且 mpeg4 存的是受限(limited)范围 Y, 解码拉伸成 RGB
	// 后存储亮度超过 229 就饱和到 255——纸面链尾 5.4 也读不出 229。拉伸计算本身
	// 由纯函数 TestAdaptiveInterval 钉死; 本测试(spec Review Focus #1)的职责是
	// 预算内的全片覆盖: 候选不多于预算、亮度随帧序严格递增(铺到全片且不乱序)、
	// 尾帧覆盖到后段。
	extracted, err := ai.NewFFmpegExtractor().Extract(context.Background(), videoPath, 0.1, 1)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	defer extracted.Cleanup()
	if len(extracted.Paths) < 8 || len(extracted.Paths) > 10 {
		t.Fatalf("候选 %d 帧, want 8–10 (拉伸后应铺满全片且不超过预算)", len(extracted.Paths))
	}
	levels := make([]float64, 0, len(extracted.Paths))
	for _, p := range extracted.Paths {
		data, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("read frame: %v", err)
		}
		img, _, err := image.Decode(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("decode frame: %v", err)
		}
		levels = append(levels, meanLevel(img))
	}
	for i := 1; i < len(levels); i++ {
		if levels[i-1] >= levels[i] {
			t.Fatalf("候选帧亮度未随帧序严格递增(时间序错乱?): levels=%v", levels)
		}
	}
	if tail := levels[len(levels)-1]; tail < 200 {
		t.Errorf("尾帧平均亮度 %.1f, want ≥200: 自适应间隔必须把采样铺到全片尾段", tail)
	}
}

func TestFFmpegExtractor_CandidatesThinnedToBudgetStillCoverTail(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("no ffmpeg")
	}
	videoPath := filepath.Join(t.TempDir(), "cuts.mp4")
	writeCutClip(t, videoPath)
	// 必须用硬切片段: ramp 片段的候选会被自适应间隔拉回预算内, 永远触发不了薄化。
	// 有 ffprobe: 间隔 0.6s → 10 周期候选 + 2 硬切(2s/4s) = 12 > 预算 10 → 薄化。
	// 无 ffprobe: 间隔停在 0.1s → ~60 候选, 同样薄化到 10——本测试不 skip ffprobe,
	// 顺带验证降级路径。两种情况尾帧都必须落在绿段(4-6s)。
	extracted, err := ai.NewFFmpegExtractor().Extract(context.Background(), videoPath, 0.1, 1)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	defer extracted.Cleanup()
	if len(extracted.Paths) != 10 {
		t.Fatalf("候选 %d 帧, want 恰好 10 (薄化到预算)", len(extracted.Paths))
	}
	last, err := os.ReadFile(extracted.Paths[len(extracted.Paths)-1])
	if err != nil {
		t.Fatalf("read last frame: %v", err)
	}
	img, _, err := image.Decode(bytes.NewReader(last))
	if err != nil {
		t.Fatalf("decode last frame: %v", err)
	}
	if meanLevel(img) < 140 {
		t.Errorf("尾帧平均亮度 %.1f, want >140(绿段≈150): 薄化必须覆盖全片含尾段", meanLevel(img))
	}
}
