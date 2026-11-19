package ai

import (
	"fmt"
	"image"
	"math/bits"
	"os"

	// Frames come out of ffmpeg as JPEG, and an image file that reaches the same
	// filter may be a PNG, so both decoders are registered here rather than at
	// the call sites.
	_ "image/jpeg"
	_ "image/png"
)

// The dHash grid: 9 averaged columns compared down to 8 bits per row, 8 rows,
// which is exactly the 64 bits of the hash. Averaging is done by hand so this
// stays within the standard library.
const (
	hashGridWidth  = 9
	hashGridHeight = 8
)

// dHash returns the perceptual hash of one decoded image: the average
// brightness of each rectangle in a 9x8 grid, reduced to "is this cell brighter
// than the one to its right" for every pair of neighbours. Because only the
// sign of a difference matters, a rescaled or slightly blurred frame keeps its
// hash, which is what makes it usable for deduplicating video frames.
//
// Each cell averages its whole source rectangle instead of sampling one point:
// a single-pixel impulse or a JPEG ringing artefact is diluted to a fraction of
// the cell mean, so the comparison signs of a clean frame survive compression
// unchanged. That stability is what gives the hamming threshold its meaning.
func dHash(img image.Image) uint64 {
	rect := img.Bounds()
	width, height := rect.Dx(), rect.Dy()

	var level [hashGridHeight][hashGridWidth]float64
	for y := 0; y < hashGridHeight; y++ {
		// Integer boundaries: cell y covers rows [y*h/8, (y+1)*h/8), so the
		// last cell reaches the edge of the image.
		y0 := rect.Min.Y + y*height/hashGridHeight
		y1 := rect.Min.Y + (y+1)*height/hashGridHeight
		for x := 0; x < hashGridWidth; x++ {
			x0 := rect.Min.X + x*width/hashGridWidth
			x1 := rect.Min.X + (x+1)*width/hashGridWidth
			var sum float64
			var count int
			for py := y0; py < y1; py++ {
				for px := x0; px < x1; px++ {
					// Rec.601 luma. RGBA() returns 16-bit premultiplied
					// values, but the hash only compares neighbours of the
					// same image, so the scale and the alpha division are
					// irrelevant here.
					r, g, b, _ := img.At(px, py).RGBA()
					sum += 0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)
					count++
				}
			}
			level[y][x] = sum / float64(count)
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

// FilterSimilarFrames walks the frame files in time order and drops the ones
// that look like a frame it already kept: the same scene held for ten seconds
// costs one frame, not ten model calls. The frames stay on disk while this
// happens — each path is opened and decoded in turn and only the (path, hash)
// pair is remembered — because a frame that is dropped must never have cost
// its bytes a place in memory.
//
// Unlike a keep-then-stop scan, every frame is decoded even after maxFrames
// are in: dropping early would let the thinning below choose among only the
// first maxFrames distinct frames, and scenes that appear later in the video
// would never reach the model. Once the walk is done, if more frames survived
// than maxFrames allows, thinUniform picks a spread of them (first, last,
// evenly in between) and exactly those files are read back into memory.
// threshold and maxFrames come from Settings(), which already falls back to
// its defaults for unusable values, so no extra fallback happens here.
//
// A frame that cannot be decoded is an error rather than a skip: silently
// dropping it would hide an ffmpeg failure.
func FilterSimilarFrames(paths []string, threshold int, maxFrames int) ([][]byte, error) {
	// Pass A: stream-decode every frame, comparing each hash against *all* the
	// kept ones, not just the last: a scene that returns later (A→B→A) is a
	// duplicate of A even though it is far from B.
	type keptFrame struct {
		path string
		hash uint64
	}
	distinct := make([]keptFrame, 0, len(paths))
	for i, path := range paths {
		f, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("第 %d 帧解码失败: %w", i+1, err)
		}
		img, _, err := image.Decode(f)
		f.Close()
		if err != nil {
			return nil, fmt.Errorf("第 %d 帧解码失败: %w", i+1, err)
		}
		hash := dHash(img)
		duplicate := false
		for _, kept := range distinct {
			if hamming(hash, kept.hash) < threshold {
				duplicate = true
				break
			}
		}
		if duplicate {
			continue
		}
		distinct = append(distinct, keptFrame{path: path, hash: hash})
	}

	// Pass B: thin to maxFrames only when there are more, then read back just
	// the winners.
	selected := distinct
	if len(distinct) > maxFrames {
		idx := thinUniform(len(distinct), maxFrames)
		selected = make([]keptFrame, 0, len(idx))
		for _, i := range idx {
			selected = append(selected, distinct[i])
		}
	}
	kept := make([][]byte, 0, len(selected))
	for _, frame := range selected {
		data, err := os.ReadFile(frame.path)
		if err != nil {
			return nil, fmt.Errorf("读取帧文件 %s 失败: %w", frame.path, err)
		}
		kept = append(kept, data)
	}
	return kept, nil
}
