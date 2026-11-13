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

	frames, err := ai.NewFFmpegExtractor().Extract(context.Background(), videoPath, 1, 5)
	if err != nil {
		t.Fatalf("Extract(%s): %v", videoPath, err)
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
