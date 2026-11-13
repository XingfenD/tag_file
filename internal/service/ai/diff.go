package ai

import (
	"bytes"
	"fmt"
	"image"
	"math/bits"

	// Frames come out of ffmpeg as JPEG, and an image file that reaches the same
	// filter may be a PNG, so both decoders are registered here rather than at
	// the call sites.
	_ "image/jpeg"
	_ "image/png"
)

// The dHash grid: 9 sampled columns compared down to 8 bits per row, 8 rows,
// which is exactly the 64 bits of the hash. Sampling is nearest-neighbour by
// hand so this stays within the standard library.
const (
	hashGridWidth  = 9
	hashGridHeight = 8
)

// dHash returns the perceptual hash of one decoded image: the brightness of a
// 9x8 nearest-neighbour sample, reduced to "is this pixel brighter than the one
// to its right" for every pair of neighbours. Because only the sign of a
// difference matters, a rescaled or slightly blurred frame keeps its hash, which
// is what makes it usable for deduplicating video frames.
func dHash(img image.Image) uint64 {
	rect := img.Bounds()
	width, height := rect.Dx(), rect.Dy()

	var level [hashGridHeight][hashGridWidth]float64
	for y := 0; y < hashGridHeight; y++ {
		for x := 0; x < hashGridWidth; x++ {
			px := rect.Min.X + x*(width-1)/(hashGridWidth-1)
			py := rect.Min.Y + y*(height-1)/(hashGridHeight-1)
			// Rec.601 luma. RGBA() returns 16-bit premultiplied values, but the
			// hash only compares neighbours of the same image, so the scale and
			// the alpha division are irrelevant here.
			r, g, b, _ := img.At(px, py).RGBA()
			level[y][x] = 0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)
		}
	}

	var hash uint64
	for y := 0; y < hashGridHeight; y++ {
		for x := 0; x+1 < hashGridWidth; x++ {
			if level[y][x] > level[y][x+1] {
				hash |= 1 << uint(y*(hashGridWidth-1)+x)
			}
		}
	}
	return hash
}

// hamming counts the bits the two hashes disagree on, i.e. how different two
// frames look.
func hamming(a, b uint64) int {
	return bits.OnesCount64(a ^ b)
}

// FilterSimilarFrames walks frames in time order and drops the ones that look
// like the frame it kept last: the same scene held for ten seconds costs one
// frame, not ten model calls. The first frame is always kept, a frame within
// threshold bits of the kept one is dropped, and keeping stops once maxFrames
// are in. threshold and maxFrames come from Settings(), which already falls back
// to its defaults for non-positive values, so no extra fallback happens here.
//
// A frame that cannot be decoded is an error rather than a skip: silently
// dropping it would hide an ffmpeg failure.
func FilterSimilarFrames(frames [][]byte, threshold int, maxFrames int) ([][]byte, error) {
	kept := make([][]byte, 0, len(frames))
	var lastHash uint64
	for i, frame := range frames {
		if len(kept) >= maxFrames {
			break
		}
		img, _, err := image.Decode(bytes.NewReader(frame))
		if err != nil {
			return nil, fmt.Errorf("第 %d 帧解码失败: %w", i+1, err)
		}
		hash := dHash(img)
		// Everything is compared against the last kept frame, not the previous
		// input frame, so a slow drift over many frames is still a duplicate.
		if len(kept) > 0 && hamming(hash, lastHash) < threshold {
			continue
		}
		kept = append(kept, frame)
		lastHash = hash
	}
	return kept, nil
}
