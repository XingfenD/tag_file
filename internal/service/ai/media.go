package ai

import (
	"path/filepath"
	"strings"
)

// Media kinds reported by IsMediaFile. The vision pipeline branches on them:
// images are read as a single frame, videos go through frame extraction.
const (
	kindImage = "image"
	kindVideo = "video"
)

// Accepted extensions, compared lowercased. Unknown extensions are skipped at
// enqueue time, so an unsupported file never reaches the model.
var (
	imageExts = map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".webp": true,
	}
	videoExts = map[string]bool{
		".mp4":  true,
		".mkv":  true,
		".mov":  true,
		".avi":  true,
		".webm": true,
		".m4v":  true,
	}
)

// IsMediaFile classifies a file name by its extension. It looks at the name
// only and never touches the disk, so an unindexed or renamed file is still
// classified the same way. Only the suffix after the final dot counts, so
// "archive.tar.gz" is unknown; ok is false for every extension outside the two
// tables above and for names without a dot at all.
func IsMediaFile(name string) (kind string, ok bool) {
	ext := strings.ToLower(filepath.Ext(name))
	switch {
	case imageExts[ext]:
		return kindImage, true
	case videoExts[ext]:
		return kindVideo, true
	default:
		return "", false
	}
}
