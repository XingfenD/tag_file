package ai

import "context"

// FrameExtractor pulls frames out of a video file and returns them as encoded
// image bytes ordered by time.
type FrameExtractor interface {
	Extract(ctx context.Context, videoPath string, intervalSeconds float64, maxFrames int) ([][]byte, error)
}

// VisionClient sends a prompt plus images to a vision model and returns the raw
// text answer, unparsed.
type VisionClient interface {
	Recognize(ctx context.Context, prompt string, images [][]byte) (string, error)
}
