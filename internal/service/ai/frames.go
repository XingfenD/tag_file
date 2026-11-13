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

// Extract samples one frame every intervalSeconds and returns them as encoded
// JPEGs ordered by time, at most maxFrames*10 of them. The cap is deliberately
// loose: ffmpeg walks the whole video, FilterSimilarFrames picks the distinct
// frames afterwards, which is what actually bounds the model call.
func (e *ffmpegExtractor) Extract(ctx context.Context, videoPath string, intervalSeconds float64, maxFrames int) ([][]byte, error) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return nil, errors.New("未找到 ffmpeg,请先安装")
	}

	tmpDir, err := os.MkdirTemp("", "tagfile-ai-")
	if err != nil {
		return nil, fmt.Errorf("创建临时目录失败: %w", err)
	}
	// The frames live in this directory and nowhere else, so one defer cleans up
	// every return path below, error paths included.
	defer os.RemoveAll(tmpDir)

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
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, ffmpegError(fmt.Sprintf("ffmpeg 抽帧超过 %s 已中止", extractTimeout), err, stderr.Bytes())
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, fmt.Errorf("ffmpeg 抽帧已取消: %w", ctxErr)
		}
		return nil, ffmpegError("ffmpeg 抽帧失败", err, stderr.Bytes())
	}

	frames, err := readFrames(tmpDir)
	if err != nil {
		return nil, err
	}
	if len(frames) == 0 {
		// Exit code 0 with nothing written happens on a video without a video
		// stream, so stderr is merged in to tell the two apart.
		return nil, ffmpegError("未能从视频抽取任何帧", nil, stderr.Bytes())
	}
	return frames, nil
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

// readFrames loads the extracted JPEGs back in file name order, which is the
// frame order ffmpeg wrote them in.
func readFrames(tmpDir string) ([][]byte, error) {
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

	frames := make([][]byte, 0, len(names))
	for _, name := range names {
		frame, err := os.ReadFile(filepath.Join(tmpDir, name))
		if err != nil {
			return nil, fmt.Errorf("读取抽帧文件 %s 失败: %w", name, err)
		}
		frames = append(frames, frame)
	}
	return frames, nil
}
