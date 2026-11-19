package ai

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// extractTimeout bounds one ffmpeg run. Decoding a long video at the smallest
// configured interval is slow work, but it must never hang the recognition
// worker that called it.
const extractTimeout = 120 * time.Second

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

// Extract samples one frame every intervalSeconds and hands back their file
// paths in time order, at most maxFrames*10 of them, inside an
// ExtractedFrames whose scratch directory the caller now owns (Cleanup deletes
// it). The cap is deliberately loose: ffmpeg walks the whole video,
// FilterSimilarFrames picks the distinct frames afterwards, which is what
// actually bounds the model call.
func (e *ffmpegExtractor) Extract(ctx context.Context, videoPath string, intervalSeconds float64, maxFrames int) (extracted *ExtractedFrames, err error) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return nil, errors.New("未找到 ffmpeg,请先安装")
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

	// The caller's own context, kept apart from the timeout one derived below: a
	// deadline shorter than extractTimeout also surfaces as DeadlineExceeded on the
	// derived context, and that run must not be blamed on the internal limit.
	callerCtx := ctx
	ctx, cancel := context.WithTimeout(ctx, extractTimeout)
	defer cancel()

	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-hide_banner", "-loglevel", "error", "-y",
		"-i", videoPath,
		"-vf", "fps=1/"+strconv.FormatFloat(intervalSeconds, 'f', -1, 64),
		"-frames:v", strconv.Itoa(maxFrames*10),
		"-f", "image2",
		filepath.Join(tmpDir, framePattern),
	)
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		// Checked before the internal timeout below, since the derived context
		// reports DeadlineExceeded for its parent's deadline too; the caller's error
		// is wrapped so errors.Is still names the deadline that fired.
		if callerErr := callerCtx.Err(); errors.Is(callerErr, context.DeadlineExceeded) {
			return nil, ffmpegError("ffmpeg 抽帧超过调用方设定的时限已中止", callerErr, stderr.Bytes())
		}
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, ffmpegError(fmt.Sprintf("ffmpeg 抽帧超过 %s 已中止", extractTimeout), err, stderr.Bytes())
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			// Cancellation, not a timeout, and the only branch that used to drop
			// stderr: ctxErr is wrapped so errors.Is still sees context.Canceled,
			// and whatever ffmpeg had written before the kill is merged in like the
			// branches around it.
			return nil, ffmpegError("ffmpeg 抽帧已取消", ctxErr, stderr.Bytes())
		}
		return nil, ffmpegError("ffmpeg 抽帧失败", err, stderr.Bytes())
	}

	paths, err := readFramePaths(tmpDir)
	if err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		// Exit code 0 with nothing written happens on a video without a video
		// stream, so stderr is merged in to tell the two apart.
		return nil, ffmpegError("未能从视频抽取任何帧", nil, stderr.Bytes())
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

// readFramePaths lists the frame files ffmpeg wrote in file name order, which
// is the frame order. The bytes stay on disk: the filter decodes them one at
// a time and only the frames it keeps are read back into memory.
func readFramePaths(tmpDir string) ([]string, error) {
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		return nil, fmt.Errorf("读取临时目录失败: %w", err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if name := entry.Name(); !entry.IsDir() && strings.HasPrefix(name, framePrefix) && strings.HasSuffix(name, frameSuffix) {
			names = append(names, name)
		}
	}
	sort.Strings(names)

	paths := make([]string, 0, len(names))
	for _, name := range names {
		paths = append(paths, filepath.Join(tmpDir, name))
	}
	return paths, nil
}
