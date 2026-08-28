package handlers

import (
	"fmt"
	"strconv"
	"strings"
)

func ParseInt(s string) (uint, error) {
	v, err := strconv.ParseUint(s, 10, 64)
	return uint(v), err
}

func formatSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}

func IsMediaType(fileType string) string {
	if strings.HasPrefix(fileType, "image/") {
		return "image"
	}
	if strings.HasPrefix(fileType, "video/") {
		return "video"
	}
	if strings.HasPrefix(fileType, "audio/") {
		return "audio"
	}
	return ""
}

var playableVideoTypes = map[string]bool{
	"video/mp4":  true,
	"video/webm": true,
	"video/ogg":  true,
	"video/ogv":  true,
}

func isPlayableVideo(fileType string) bool {
	return playableVideoTypes[fileType]
}
