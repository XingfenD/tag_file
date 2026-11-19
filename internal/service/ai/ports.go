package ai

import (
	"context"
	"os"
)

// ExtractedFrames is what a FrameExtractor hands back: encoded frame files in
// time order, as paths on disk rather than bytes in memory. A long video at a
// short interval is far too many frames to keep decoded, so they stay in the
// scratch directory and only the frames the filter finally selects are read
// back; dir is the directory holding them, set by the extractor that created
// it.
type ExtractedFrames struct {
	Paths []string
	dir   string
}

// Cleanup removes the scratch directory the frames live in. It is nil-safe and
// idempotent: calling it on a nil result, or twice, does nothing, so a caller
// can defer it unconditionally and the error paths (which return nil) stay
// leak-free. Test fakes leave dir empty and Cleanup becomes a no-op.
func (f *ExtractedFrames) Cleanup() {
	if f == nil || f.dir == "" {
		return
	}
	os.RemoveAll(f.dir)
	f.dir = ""
}

// FrameExtractor pulls frames out of a video file and returns them as encoded
// image files ordered by time. On success the caller owns the scratch
// directory inside the result and must call Cleanup; on error the return is
// nil and nothing was left behind.
type FrameExtractor interface {
	Extract(ctx context.Context, videoPath string, intervalSeconds float64, maxFrames int) (*ExtractedFrames, error)
}

// VisionClient sends a prompt plus images to a vision model and returns the raw
// text answer, unparsed.
type VisionClient interface {
	Recognize(ctx context.Context, prompt string, images [][]byte) (string, error)
}
