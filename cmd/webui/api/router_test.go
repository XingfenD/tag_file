package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/XingfenD/tag_file/internal/model"
	categorySvc "github.com/XingfenD/tag_file/internal/service/category"
	configSvc "github.com/XingfenD/tag_file/internal/service/config"
	exportSvc "github.com/XingfenD/tag_file/internal/service/export"
	fileSvc "github.com/XingfenD/tag_file/internal/service/file"
	searchSvc "github.com/XingfenD/tag_file/internal/service/search"
	tagSvc "github.com/XingfenD/tag_file/internal/service/tag"
	"github.com/XingfenD/tag_file/pkg/storage"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// storage 是进程级单例，测试之间不得并行。
func newTestEngine(t *testing.T) *gin.Engine {
	t.Helper()
	if err := storage.InitDB(filepath.Join(t.TempDir(), "test.db")); err != nil {
		t.Fatalf("init db: %v", err)
	}
	if err := storage.GetDB().AutoMigrate(
		&model.File{}, &model.TagGroup{}, &model.TagGroupTag{},
		&model.TagCategory{}, &model.Tag{}, &model.TagValue{}, &model.Config{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	NewRouter(
		categorySvc.NewService(), tagSvc.NewService(), fileSvc.NewService(),
		searchSvc.NewService(), configSvc.NewService(), exportSvc.NewService(),
	).RegisterRoutes(r.Group("/api"))
	return r
}

func doRequest(t *testing.T, r *gin.Engine, method, target, body string) (int, map[string]any) {
	t.Helper()
	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, target, nil)
	} else {
		req = httptest.NewRequest(method, target, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var out map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode %s %s: %v (body=%s)", method, target, err, w.Body.String())
	}
	return w.Code, out
}

type fixture struct {
	Path         string
	FileID       uint
	TagID        uint
	DefaultGroup uint
	ExtraGroup   uint
}

// 直接写库造数据，避免走 scan 触发 .tagfile/scan_state 落盘与 cwd 依赖。
func seedFixture(t *testing.T, path string) fixture {
	t.Helper()
	db := storage.GetDB()
	cat := model.TagCategory{CategoryName: "类型"}
	mustCreate(t, db, &cat)
	tag := model.Tag{CategoryID: cat.CategoryID, TagName: "tmp"}
	mustCreate(t, db, &tag)
	mustCreate(t, db, &model.TagValue{TagID: tag.TagID, Value: "v1"})
	f := model.File{FilePath: path, FileName: filepath.Base(path), FileSize: 12, FileType: "text/plain", TagCnt: 1}
	mustCreate(t, db, &f)
	dg := model.TagGroup{FileID: f.FileID, GroupName: "default", IsDefault: true}
	eg := model.TagGroup{FileID: f.FileID, GroupName: "extra", IsDefault: false}
	mustCreate(t, db, &dg)
	mustCreate(t, db, &eg)
	mustCreate(t, db, &model.TagGroupTag{GroupID: dg.GroupID, FileID: f.FileID, TagID: tag.TagID, TagValue: ""})
	return fixture{Path: path, FileID: f.FileID, TagID: tag.TagID, DefaultGroup: dg.GroupID, ExtraGroup: eg.GroupID}
}

func mustCreate(t *testing.T, db *gorm.DB, v any) {
	t.Helper()
	if err := db.Create(v).Error; err != nil {
		t.Fatal(err)
	}
}