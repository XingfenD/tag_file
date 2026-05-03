package handlers

import (
	"fmt"
	"strconv"
	"strings"
	"net/http"
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

type PageData struct {
	Page       int
	PageSize   int
	Total      int64
	TotalPages int
	HasPrev    bool
	HasNext    bool
	PrevPage   int
	NextPage   int
	Start      int
	End        int
}

func Paginate(r *http.Request, total int64, pageSize int) PageData {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}
	if total > 0 {
		if totalPages < 1 {
			totalPages = 1
		}
		if page > totalPages {
			page = totalPages
		}
	}
	offset := (page - 1) * pageSize
	end := offset + pageSize
	if end > int(total) {
		end = int(total)
	}
	return PageData{
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: totalPages,
		HasPrev:    page > 1,
		HasNext:    page < totalPages,
		PrevPage:   page - 1,
		NextPage:   page + 1,
		Start:      offset + 1,
		End:        end,
	}
}

func PaginationBaseURL(r *http.Request) string {
	base := r.URL.Path + "?"
	skip := map[string]bool{"page": true}
	for k, vs := range r.URL.Query() {
		if skip[k] {
			continue
		}
		for _, v := range vs {
			base += k + "=" + v + "&"
		}
	}
	return base + "page="
}

func UpdatePagination(pg *PageData, total int64, count int) {
	pg.Total = total
	if total > 0 {
		pg.TotalPages = int(total) / pg.PageSize
		if int(total)%pg.PageSize > 0 {
			pg.TotalPages++
		}
	}
	pg.HasPrev = pg.Page > 1
	pg.HasNext = pg.Page < pg.TotalPages
	pg.End = pg.Start + count - 1
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
