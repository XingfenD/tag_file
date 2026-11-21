package file

import (
	"mime"
	"path/filepath"
	"strings"
)

/* Go 的内置 MIME 表缺这些常见媒体扩展，而 /etc/mime.types 在 musl 容器里根本不存在、
   在 macOS 上也未必覆盖——一旦 TypeByExtension 落空、类型按字面扩展入库（".mp4"），
   WebUI 按「video/」前缀的媒体分类与 video/mp4 可播映射就双双失配，视频预览静默退化。
   这张兜底表让环境 MIME 表只做补充、不做裁决。 */
var mediaMimeFallback = map[string]string{
	".mp4":  "video/mp4",
	".m4v":  "video/mp4",
	".mov":  "video/quicktime",
	".webm": "video/webm",
	".mkv":  "video/x-matroska",
	".avi":  "video/x-msvideo",
	".ogv":  "video/ogg",
	".mp3":  "audio/mpeg",
	".m4a":  "audio/mp4",
	".aac":  "audio/aac",
	".wav":  "audio/wav",
	".flac": "audio/flac",
	".opus": "audio/opus",
	".ogg":  "audio/ogg",
}

// fileTypeByExt 是扫描与导入共用的扩展名→内容类型判定：系统/内置 MIME 表优先，
// 媒体兜底表其次，都不认识才存小写扩展名。
func fileTypeByExt(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	if fileType := mime.TypeByExtension(ext); fileType != "" {
		return fileType
	}
	if fileType, ok := mediaMimeFallback[ext]; ok {
		return fileType
	}
	return ext
}
