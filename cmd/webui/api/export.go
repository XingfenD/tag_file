package api

import (
	"fmt"
	"net/http"
	"time"
)

// HandleExport responds to GET /api/export.
// Streams a SQL dump of the database as an attachment.
func (rt *Router) HandleExport(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/sql")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"tagfile_%s.sql\"", time.Now().Format("20060102_150405")))

	if err := rt.ExportSvc.ExportToSQL(w); err != nil {
		// Reset content type on error so browser shows plain text
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		http.Error(w, "导出失败: "+err.Error(), http.StatusInternalServerError)
		return
	}
}
