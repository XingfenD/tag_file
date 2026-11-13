package ai_test

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/jpeg"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/XingfenD/tag_file/internal/service/ai"
)

// frameEdge is the side length of every synthesized frame. 64 keeps the eight
// 8-row bands below aligned with the JPEG block grid, so compression cannot turn
// a smooth gradient into a wobbly one and the dHash of a frame is decided by its
// pattern alone.
const frameEdge = 64

// bandBrightness builds a brightness function whose rows ramp down (bright on
// the left) where code has the bit set and ramp up elsewhere. The dedupe filter
// samples eight rows of eight comparisons, so two codes differing in n bands
// produce frames whose hashes differ in 8n of the 64 bits.
func bandBrightness(code uint8) func(x, y int) uint8 {
	return func(x, y int) uint8 {
		level := uint8(uint32(x) * 255 / (frameEdge - 1))
		if code&(1<<uint(y*8/frameEdge)) == 0 {
			level = 255 - level
		}
		return level
	}
}

// encodeFrame JPEG-encodes a tinted grayscale ramp: every pixel is tint scaled
// by brightness(x, y). Scaling only rescales the ramp, so the perceived
// structure is the brightness pattern whatever the colour is.
func encodeFrame(t *testing.T, tint color.RGBA, brightness func(x, y int) uint8) []byte {
	t.Helper()
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
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatalf("encode frame: %v", err)
	}
	return buf.Bytes()
}

var (
	tintRed   = color.RGBA{R: 255, A: 255}
	tintBlue  = color.RGBA{B: 255, A: 255}
	tintGreen = color.RGBA{G: 255, A: 255}
	tintGray  = color.RGBA{R: 255, G: 255, B: 255, A: 255}
)

func TestFilterSimilarFrames_DropsNearDuplicates(t *testing.T) {
	// 纯红、纯红(微噪 1 像素)、纯蓝、纯蓝、纯绿 → 保留 3 帧(红/蓝/绿)。
	plain := bandBrightness(0b0000_1111)
	noisy := func(x, y int) uint8 {
		if x == 30 && y == 32 {
			return 255
		}
		return plain(x, y)
	}
	blue := bandBrightness(0b1111_0000)
	green := bandBrightness(0b0011_0011)

	frames := [][]byte{
		encodeFrame(t, tintRed, plain),
		encodeFrame(t, tintRed, noisy),
		encodeFrame(t, tintBlue, blue),
		encodeFrame(t, tintBlue, blue),
		encodeFrame(t, tintGreen, green),
	}
	if bytes.Equal(frames[0], frames[1]) {
		t.Fatal("the noisy frame must differ in bytes from the first one")
	}

	got, err := ai.FilterSimilarFrames(frames, 10, 10)
	if err != nil {
		t.Fatalf("FilterSimilarFrames: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("kept %d frames, want 3 (red/blue/green): %v", len(got), frameLens(got))
	}
	for i, want := range []int{0, 2, 4} {
		if !bytes.Equal(got[i], frames[want]) {
			t.Errorf("kept frame %d is not input frame %d", i, want)
		}
	}
}

func TestFilterSimilarFrames_AlwaysKeepsFirstAndCapsAtMax(t *testing.T) {
	// 8 帧互不相同 → threshold=10, maxFrames=3 → 恰好 3 帧, 且第 1 帧等于输入第 1 帧。
	frames := make([][]byte, 0, 8)
	for start := 0; start < 8; start++ {
		code := uint8(0b0000_1111<<uint(start)) | uint8(0b0000_1111>>(8-uint(start)))
		frames = append(frames, encodeFrame(t, tintGray, bandBrightness(code)))
	}

	// The eight codes are rotations of the same four bands, so any two of them
	// differ in at least two bands, i.e. in 16 of the 64 hash bits: the filter
	// would keep all eight and only maxFrames stops it at 3.
	got, err := ai.FilterSimilarFrames(frames, 10, 3)
	if err != nil {
		t.Fatalf("FilterSimilarFrames: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("kept %d frames, want the 3 of maxFrames: %v", len(got), frameLens(got))
	}
	if !bytes.Equal(got[0], frames[0]) {
		t.Error("the first frame must always be kept")
	}

	// 单帧输入 → 原样返回 1 帧。
	one, err := ai.FilterSimilarFrames(frames[:1], 10, 3)
	if err != nil {
		t.Fatalf("FilterSimilarFrames(single): %v", err)
	}
	if len(one) != 1 || !bytes.Equal(one[0], frames[0]) {
		t.Fatalf("kept %d frames for a single-frame input, want that frame back", len(one))
	}

	// 空输入 → 返回空、无 error。
	none, err := ai.FilterSimilarFrames([][]byte{}, 10, 3)
	if err != nil {
		t.Fatalf("FilterSimilarFrames(empty): %v", err)
	}
	if len(none) != 0 {
		t.Fatalf("kept %d frames for empty input, want 0", len(none))
	}
}

func TestFilterSimilarFrames_RejectsUndecodableFrame(t *testing.T) {
	// [][]byte{[]byte("not an image")} → 返回 error(Review Focus #4)。
	got, err := ai.FilterSimilarFrames([][]byte{[]byte("not an image")}, 10, 10)
	if err == nil {
		t.Fatalf("undecodable frame kept %d frames with no error, want an error", len(got))
	}
	if got != nil {
		t.Errorf("frames = %v, want nil next to an error", frameLens(got))
	}
}

func TestFFmpegExtractor_MissingBinaryOrZeroFrames(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("no ffmpeg")
	}

	frames, err := ai.NewFFmpegExtractor().Extract(context.Background(), filepath.Join(t.TempDir(), "missing.mp4"), 1, 5)
	if err == nil {
		t.Fatalf("extract from a missing video returned %d frames and no error", len(frames))
	}
	if frames != nil {
		t.Errorf("frames = %v, want nil next to an error", frameLens(frames))
	}
}

// frameLens summarises a result for a failure message without dumping image
// bytes into the test log.
func frameLens(frames [][]byte) []int {
	lens := make([]int, 0, len(frames))
	for _, frame := range frames {
		lens = append(lens, len(frame))
	}
	return lens
}
