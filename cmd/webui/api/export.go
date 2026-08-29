package api

import (
	"bytes"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// HandleExport responds to GET /api/export.
// Streams a SQL dump of the database as an attachment.
func (rt *Router) HandleExport(c *gin.Context) {
	var buf bytes.Buffer
	if err := rt.ExportSvc.ExportToSQL(&buf); err != nil {
		respondStatus(c, "导出失败: "+err.Error(), msgError, http.StatusInternalServerError)
		return
	}
	filename := fmt.Sprintf("tagfile_%s.sql", time.Now().Format("20060102_150405"))
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.Data(http.StatusOK, "application/sql", buf.Bytes())
}
