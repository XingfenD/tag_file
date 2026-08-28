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
			"templates/partials/pagination.html",
			"templates/partials/empty.html",
			"templates/partials/tag_badge.html",
		}

		pages := make(map[string]*template.Template)
		for _, name := range []string{"home", "category", "tag", "file", "file_detail", "file_groups", "search", "config"} {
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

		s := handlers.NewServer(pages)
		s.CoreVersion = coreConst.CoreVersion
		s.WebuiVersion = webuiConst.WebuiVersion

		rt := api.NewRouter(cs, ts, fSvc, ss, cfs, es)

		gin.SetMode(gin.ReleaseMode)
		r := gin.New()
		r.Use(gin.Logger(), gin.Recovery())

		r.HTMLRender = handlers.NewTemplateRenderer(pages)

		staticSub, _ := fs.Sub(staticFS, "static")
		r.StaticFS("/static", http.FS(staticSub))

		s.RegisterRoutes(r)

		apiGroup := r.Group("/api")
		rt.RegisterRoutes(apiGroup)

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
