package ai

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// sceneCutThreshold is the score ffmpeg's scene-detection must reach on a
// frame before the select filter treats it as a hard cut worth sampling even
// when the periodic interval has not elapsed. 0.3 is ffmpeg's own documented
// ballpark for "a real cut" versus ordinary motion.
const sceneCutThreshold = 0.3

// extractTimeout bounds one ffmpeg run, scaled to the video it decodes (see
// the function below). Decoding a long video at the smallest configured
// interval is slow work, but it must never hang the recognition worker that
// called it, hence the hard cap; the floor keeps short or unprobeable videos
// on the ceiling the old fixed constant imposed.
const (
	extractTimeoutFloor = 120 * time.Second
	extractTimeoutCap   = 30 * time.Minute
)

// probeTimeout bounds the ffprobe duration lookup. It is metadata-only work,
// so a probe that takes this long is a broken ffprobe, and the interval
// simply stays on the configured value.
const probeTimeout = 10 * time.Second

// candidateTimePattern matches the timestamp showinfo prints per selected
// frame ("pts_time:2.5", also plain "pts_time:0"). Integers and decimals
// only; showinfo never prints a negative or hex timestamp.
var candidateTimePattern = regexp.MustCompile(`pts_time:([0-9]+(?:\.[0-9]+)?)`)

// probeDuration asks ffprobe how long the video is, in seconds. Every failure
// — ffprobe missing from PATH, a non-zero exit, unparsable or non-positive
// output, the context ending first — logs and returns 0, which downstream
// callers read as "stay on the configured interval". Duration probing is an
// optimisation, never a reason for Extract to fail.
func probeDuration(ctx context.Context, videoPath string) float64 {
	if _, err := exec.LookPath("ffprobe"); err != nil {
		log.Printf("[ai] 未找到 ffprobe, 抽帧间隔不做自适应: %v", err)
		return 0
	}

	ctx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ffprobe",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		videoPath,
	)
	out, err := cmd.Output()
	if err != nil {
		log.Printf("[ai] ffprobe 探测 %s 时长失败, 抽帧间隔不做自适应: %v", videoPath, err)
		return 0
	}
	duration, err := strconv.ParseFloat(strings.TrimSpace(string(out)), 64)
	if err != nil || duration <= 0 {
		log.Printf("[ai] ffprobe 报告 %s 时长 %q 不可用, 抽帧间隔不做自适应", videoPath, strings.TrimSpace(string(out)))
		return 0
	}
	return duration
}

// adaptiveInterval stretches the configured sampling interval so that a video
// whose duration is known yields about maxFrames*10 periodic candidates spread
// over its whole runtime instead of clustering at the start. Without a
// duration (ffprobe missing or broken) it is the configured interval, and the
// candidate thinning in Extract takes over the coverage job.
func adaptiveInterval(configInterval, duration float64, maxFrames int) float64 {
	if duration <= 0 {
		return configInterval
	}
	if stretched := duration / float64(maxFrames*10); stretched > configInterval {
		return stretched
	}
	return configInterval
}

// extractTimeout is the wall-clock ceiling for one extraction of a video this
// long. Two times the duration covers the first pass (a full decode) plus the
// second pass (one fast seek per candidate) even on a busy machine, clamped
// into [extractTimeoutFloor, extractTimeoutCap]; duration 0 — everything
// ffprobe could not answer — takes the floor.
func extractTimeout(duration float64) time.Duration {
	if duration <= 0 {
		return extractTimeoutFloor
	}
	timeout := time.Duration(2 * duration * float64(time.Second))
	if timeout < extractTimeoutFloor {
		return extractTimeoutFloor
	}
	if timeout > extractTimeoutCap {
		return extractTimeoutCap
	}
	return timeout
}

// parseCandidateTimes extracts the pts_time of every frame showinfo printed,
// in the order they appear, which is ffmpeg's frame order. Lines without a
// timestamp (the banner, stream maps) are noise the regexp simply skips.
func parseCandidateTimes(showinfo []byte) []float64 {
	times := make([]float64, 0, len(showinfo)/64+1)
	for _, match := range candidateTimePattern.FindAllSubmatch(showinfo, -1) {
		// The regexp already guarantees a parsable decimal, so the ParseFloat
		// error branch is unreachable in practice; skipping is safer than
		// aborting over one odd line.
		ts, err := strconv.ParseFloat(string(match[1]), 64)
		if err != nil {
			continue
		}
		times = append(times, ts)
	}
	return times
}

// candidateTimes runs the first ffmpeg pass — a full decode with no frame
// files written — and returns the timestamps of every frame the select filter
// picked, in time order. The expression has three layers of meaning:
// isnan(prev_selected_t) forces the very first frame, gte(t-prev_selected_t,
// interval) is the periodic layer that also catches slow fades and long
// single takes, and gt(scene, threshold) adds any frame whose scene score says
// a hard cut happened before the next interval tick would have.
//
// The whole select='…',showinfo value stays ONE argument with the inner
// single quotes intact: they are ffmpeg's expression escaping, keeping the
// commas inside select() from being read as filter-chain separators.
func candidateTimes(ctx context.Context, videoPath string, interval float64) ([]float64, error) {
	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-hide_banner", "-loglevel", "info",
		"-i", videoPath,
		"-an", "-sn", "-dn",
		"-vf", fmt.Sprintf("select='isnan(prev_selected_t)+gte(t-prev_selected_t,%s)+gt(scene,%s)',showinfo",
			strconv.FormatFloat(interval, 'f', -1, 64),
			strconv.FormatFloat(sceneCutThreshold, 'f', -1, 64)),
		"-f", "null", "-",
	)
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, ffmpegError("ffmpeg 候选帧扫描失败", err, stderr.Bytes())
	}

	times := parseCandidateTimes(stderr.Bytes())
	if len(times) == 0 {
		// Exit code 0 with no selected frame happens on a video without a
		// video stream, so stderr is merged in to tell the two apart.
		return nil, ffmpegError("未能从视频识别任何候选帧", nil, stderr.Bytes())
	}
	return times, nil
}

// ffmpeg writes one JPEG per sampled frame into the temporary directory under
// this name pattern; the sequence number keeps the file names in frame order.
const (
	framePrefix  = "frame_"
	frameSuffix  = ".jpg"
	framePattern = framePrefix + "%04d" + frameSuffix
)

// ffmpegExtractor implements FrameExtractor by shelling out to the ffmpeg binary
// found in PATH. It holds no state, so the single value NewFFmpegExtractor
// returns serves every task.
type ffmpegExtractor struct{}

// NewFFmpegExtractor returns the frame extractor the recognition pipeline uses
// for video files. ffmpeg is an external dependency: when it is missing Extract
// says so instead of pretending the video had no frames.
func NewFFmpegExtractor() FrameExtractor {
	return &ffmpegExtractor{}
}

// Extract hands back candidate frame paths covering the whole video, in time
// order, inside an ExtractedFrames whose scratch directory the caller now owns
// (Cleanup deletes it). Two ffmpeg passes do the job: the first decodes end to
// end writing nothing, letting select pick candidates — the configured
// interval stretched by probeDuration when ffprobe reports the video is long
// for the maxFrames*10 candidate budget, plus a scene-change layer so hard
// cuts between interval ticks are never missed — and showinfo prints their
// timestamps; the second seeks each surviving timestamp out and writes one
// JPEG. The budget is deliberately loose: ffmpeg covers the whole film,
// FilterSimilarFrames picks the distinct frames afterwards, which is what
// actually bounds the model call.
func (e *ffmpegExtractor) Extract(ctx context.Context, videoPath string, intervalSeconds float64, maxFrames int) (extracted *ExtractedFrames, err error) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return nil, errors.New("未找到 ffmpeg,请先安装")
	}

	duration := probeDuration(ctx, videoPath)

	// The caller's own context, kept apart from the timeout one derived below: a
	// deadline shorter than the duration-scaled ceiling also surfaces as
	// DeadlineExceeded on the derived context, and that run must not be blamed
	// on the internal limit.
	callerCtx := ctx
	ctx, cancel := context.WithTimeout(ctx, extractTimeout(duration))
	defer cancel()

	times, err := candidateTimes(ctx, videoPath, adaptiveInterval(intervalSeconds, duration, maxFrames))
	if err != nil {
		// Attribution follows the old pattern. The caller deadline is checked
		// first, since the derived context reports DeadlineExceeded for its
		// parent's deadline too; the caller's error is wrapped so errors.Is
		// still names the deadline that fired. The internal branch names the
		// duration-scaled ceiling. Everything else candidateTimes produced is
		// already a user-facing ffmpegError carrying that run's stderr. The
		// deadline branches pass no stderr: candidateTimes already folded it
		// into its own error, and a killed -loglevel info run prints mostly
		// progress noise nobody should see twice.
		if callerErr := callerCtx.Err(); errors.Is(callerErr, context.DeadlineExceeded) {
			return nil, ffmpegError("ffmpeg 抽帧超过调用方设定的时限已中止", callerErr, nil)
		}
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, ffmpegError(fmt.Sprintf("ffmpeg 抽帧超过 %s 已中止", extractTimeout(duration)), ctx.Err(), nil)
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			// Cancellation, not a timeout: ctxErr is wrapped so errors.Is
			// still sees context.Canceled, which lets the worker tell a
			// cancelled task apart from a broken video.
			return nil, ffmpegError("ffmpeg 抽帧已取消", ctxErr, nil)
		}
		return nil, err
	}

	// Spend the old single-pass cap — maxFrames*10 — over the whole video:
	// when the scan produced more candidates than that, thinUniform keeps the
	// first and the last and samples the rest evenly, so thinning never drops
	// coverage at either end.
	if budget := maxFrames * 10; len(times) > budget {
		idx := thinUniform(len(times), budget)
		thinned := make([]float64, len(idx))
		for i, j := range idx {
			thinned[i] = times[j]
		}
		times = thinned
	}

	tmpDir, err := os.MkdirTemp("", "tagfile-ai-")
	if err != nil {
		return nil, fmt.Errorf("创建临时目录失败: %w", err)
	}
	// The frames live in this directory and nowhere else. On an error they are
	// worthless and the directory goes with them; on success it is handed to
	// the caller through ExtractedFrames.dir and its Cleanup does the removal.
	// The named return is what lets this defer see which of the two happened.
	defer func() {
		if err != nil {
			os.RemoveAll(tmpDir)
		}
	}()

	paths := make([]string, 0, len(times))
	for i, ts := range times {
		// -ss before -i is a fast seek: ffmpeg jumps along the container
		// index and decodes from there, landing on the frame at ts to the
		// precision the stream's own keyframes allow — cheap enough to run
		// once per candidate however many there are.
		outPath := filepath.Join(tmpDir, fmt.Sprintf(framePattern, i))
		var stderr bytes.Buffer
		cmd := exec.CommandContext(ctx, "ffmpeg",
			"-hide_banner", "-loglevel", "error", "-y",
			"-ss", strconv.FormatFloat(ts, 'f', -1, 64),
			"-i", videoPath,
			"-frames:v", "1",
			"-vf", "scale='min(768,iw)':-2",
			"-q:v", "2",
			outPath,
		)
		cmd.Stderr = &stderr
		if runErr := cmd.Run(); runErr != nil {
			return nil, ffmpegError(fmt.Sprintf("第 %d 帧(%.3fs)抽取失败", i+1, ts), runErr, stderr.Bytes())
		}
		// Exit 0 with no usable file means the seek ran off the end of the
		// stream or the encoder produced nothing; stat is the check.
		info, statErr := os.Stat(outPath)
		if statErr != nil {
			return nil, ffmpegError(fmt.Sprintf("第 %d 帧(%.3fs)没有产出图像", i+1, ts), statErr, stderr.Bytes())
		}
		if info.Size() == 0 {
			return nil, ffmpegError(fmt.Sprintf("第 %d 帧(%.3fs)没有产出图像", i+1, ts), errors.New("图像文件为空"), stderr.Bytes())
		}
		paths = append(paths, outPath)
	}

	return &ExtractedFrames{Paths: paths, dir: tmpDir}, nil
}

// ffmpegError builds the user-facing error of a failed run: the exit status (when
// there is one) first, then whatever ffmpeg complained about. -loglevel error
// means any stderr output is about this very run, so it belongs in the message
// instead of being dropped or logged.
func ffmpegError(action string, err error, stderr []byte) error {
	if err != nil {
		err = fmt.Errorf("%s: %w", action, err)
	} else {
		err = errors.New(action)
	}
	if text := strings.TrimSpace(string(stderr)); text != "" {
		return fmt.Errorf("%w, ffmpeg 报错: %s", err, text)
	}
	return err
}
