package api

import (
	"bytes"
	"fmt"
	"net/http"
	"time"
)

// HandleExport responds to GET /api/export.
// Streams a SQL dump of the database as an attachment.
func (rt *Router) HandleExport(w http.ResponseWriter, r *http.Request) {
	var buf bytes.Buffer
	if err := rt.ExportSvc.ExportToSQL(&buf); err != nil {
		respondStatus(w, "导出失败: "+err.Error(), msgError, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/sql")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"tagfile_%s.sql\"", time.Now().Format("20060102_150405")))
	w.Write(buf.Bytes())
}
