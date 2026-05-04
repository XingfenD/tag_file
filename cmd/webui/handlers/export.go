package handlers

import (
	"fmt"
	"net/http"
	"time"
)

func (s *Server) HandleExport(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/sql")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"tagfile_%s.sql\"", time.Now().Format("20060102_150405")))

	if err := s.ExportSvc.ExportToSQL(w); err != nil {
		http.Error(w, "导出失败: "+err.Error(), http.StatusInternalServerError)
		return
	}
}
