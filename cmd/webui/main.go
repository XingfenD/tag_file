package main

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"os"

	"github.com/XingfenD/tag_file/cmd/webui/api"
	webuiConst "github.com/XingfenD/tag_file/cmd/webui/constant"
	"github.com/XingfenD/tag_file/cmd/webui/handlers"
	"github.com/XingfenD/tag_file/internal/bootstrap"
	coreConst "github.com/XingfenD/tag_file/internal/constant"
	aiSvc "github.com/XingfenD/tag_file/internal/service/ai"
	categorySvc "github.com/XingfenD/tag_file/internal/service/category"
	configSvc "github.com/XingfenD/tag_file/internal/service/config"
	exportSvc "github.com/XingfenD/tag_file/internal/service/export"
	fileSvc "github.com/XingfenD/tag_file/internal/service/file"
	searchSvc "github.com/XingfenD/tag_file/internal/service/search"
	tagSvc "github.com/XingfenD/tag_file/internal/service/tag"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
)

//go:embed templates
var templateFS embed.FS

//go:embed static
var staticFS embed.FS

var (
	source string
	addr   string
)

var rootCmd = &cobra.Command{
	Use:   "tagfile-webui",
	Short: "TagFile Web UI",
	Long:  "TagFile Web UI - A web interface for managing file tags",
	RunE: func(cmd *cobra.Command, args []string) error {
		if source != "" {
			if err := os.Chdir(source); err != nil {
				return fmt.Errorf("无法切换到指定目录: %w", err)
			}
		}

		if err := bootstrap.InitDB(); err != nil {
			return err
		}

		base, err := template.New("base").Funcs(handlers.FuncMap).ParseFS(templateFS, "templates/base.html")
		if err != nil {
			return err
		}

		partials := []string{
			"templates/partials/empty.html",
			"templates/partials/config_group.html",
			"templates/partials/config_field.html",
			"templates/partials/tag_search.html",
			"templates/partials/search_form.html",
			"templates/partials/pager.html",
			"templates/partials/sort_field.html",
			"templates/partials/inline_edit.html",
		}

		pages := make(map[string]*template.Template)
		for _, name := range []string{"home", "category", "tag", "file", "file_detail", "file_groups", "search", "config", "review"} {
			files := append([]string{"templates/" + name + ".html"}, partials...)
			tmpl, err := template.Must(base.Clone()).ParseFS(templateFS, files...)
			if err != nil {
				return fmt.Errorf("parse %s: %w", name, err)
			}
			pages[name] = tmpl
		}

		cs := categorySvc.NewService()
		ts := tagSvc.NewService()
		fSvc := fileSvc.NewService()
		ss := searchSvc.NewService()
		cfs := configSvc.NewService()
		es := exportSvc.NewService()
		aiService := aiSvc.NewService()

		s := handlers.NewServer(pages)
		s.CoreVersion = coreConst.CoreVersion
		s.WebuiVersion = webuiConst.WebuiVersion

		rt := api.NewRouter(cs, ts, fSvc, ss, cfs, es, aiService)

		gin.SetMode(gin.ReleaseMode)
		r := gin.New()
		r.Use(gin.Logger(), gin.Recovery())

		r.HTMLRender = handlers.NewTemplateRenderer(pages)

		staticSub, _ := fs.Sub(staticFS, "static")
		r.StaticFS("/static", http.FS(staticSub))

		s.RegisterRoutes(r)

		apiGroup := r.Group("/api")
		rt.RegisterRoutes(apiGroup)

		// The worker starts after the database is open and before the listener
		// takes a single request: StartWorker's restart sweep is the one thing
		// that must not race with an enqueue claiming a task, and only the first
		// claim can happen over HTTP. Exactly one StartWorker per Service is the
		// contract (see internal/service/ai/worker.go).
		//
		// r.Run blocks for the life of the process and this codebase has no
		// signal handler, so the deferred StopWorker runs when the listener fails
		// rather than on Ctrl-C; a killed process leaves its running row for the
		// next start's sweep to mark 重启中断. That is the recovery Task 8 is built
		// around, and StopWorker would not cancel an in-flight recognition anyway
		// (worst case ffmpeg 120s + HTTP 180s).
		aiService.StartWorker()
		defer aiService.StopWorker()

		fmt.Printf("TagFile Web UI 已启动: http://localhost%s\n", addr)
		return r.Run(addr)
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&source, "source", "", "工作目录（默认当前目录）")
	rootCmd.PersistentFlags().StringVar(&addr, "addr", ":8088", "监听地址")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
