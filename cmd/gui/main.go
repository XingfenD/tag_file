package main

import (
	"embed"
	"fmt"
	"io/fs"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"

	"github.com/XingfenD/tag_file/internal/bootstrap"
	categorySvc "github.com/XingfenD/tag_file/internal/service/category"
	configSvc "github.com/XingfenD/tag_file/internal/service/config"
	exportSvc "github.com/XingfenD/tag_file/internal/service/export"
	fileSvc "github.com/XingfenD/tag_file/internal/service/file"
	searchSvc "github.com/XingfenD/tag_file/internal/service/search"
	tagSvc "github.com/XingfenD/tag_file/internal/service/tag"
)

//go:embed frontend
var frontendFS embed.FS

func main() {
	if err := bootstrap.InitDB(); err != nil {
		fmt.Fprintf(os.Stderr, "数据库初始化失败: %v\n", err)
		os.Exit(1)
	}

	app := NewApp(
		categorySvc.NewService(),
		tagSvc.NewService(),
		fileSvc.NewService(),
		searchSvc.NewService(),
		configSvc.NewService(),
		exportSvc.NewService(),
	)

	subFS, err := fs.Sub(frontendFS, "frontend")
	if err != nil {
		fmt.Fprintf(os.Stderr, "嵌入资源加载失败: %v\n", err)
		os.Exit(1)
	}

	err = wails.Run(&options.App{
		Title:     "TagFile - File Tag Manager",
		Width:     1280,
		Height:    800,
		MinWidth:  900,
		MinHeight: 600,
		AssetServer: &assetserver.Options{
			Assets: subFS,
		},
		OnStartup:  app.startup,
		OnShutdown: app.shutdown,
		Bind: []interface{}{
			app,
		},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "启动失败: %v\n", err)
		os.Exit(1)
	}
}
