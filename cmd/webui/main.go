package main

import (
	"embed"
	"flag"
	"fmt"
	"html/template"
	"net/http"
	"os"

	"github.com/XingfenD/tag_file/cmd/webui/handlers"
	"github.com/XingfenD/tag_file/internal/bootstrap"
	"github.com/XingfenD/tag_file/internal/service"
)

//go:embed templates/*
var templateFS embed.FS

func main() {
	source := flag.String("source", "", "工作目录（默认当前目录）")
	addr := flag.String("addr", ":8088", "监听地址")
	flag.Parse()

	if *source != "" {
		if err := os.Chdir(*source); err != nil {
			fmt.Fprintf(os.Stderr, "无法切换到指定目录: %v\n", err)
			os.Exit(1)
		}
	}

	if err := bootstrap.InitDB(); err != nil {
		panic(err)
	}

	base, err := template.New("base").Funcs(handlers.FuncMap).ParseFS(templateFS, "templates/base.html")
	if err != nil {
		panic(err)
	}

	pages := make(map[string]*template.Template)
	for _, name := range []string{"home", "category", "tag", "attribute", "file", "file_detail", "search", "config"} {
		tmpl, err := template.Must(base.Clone()).ParseFS(templateFS, "templates/"+name+".html")
		if err != nil {
			panic(fmt.Sprintf("parse %s: %v", name, err))
		}
		pages[name] = tmpl
	}

	s := handlers.NewServer(pages,
		service.NewCategoryService(),
		service.NewTagService(),
		service.NewAttrService(),
		service.NewFileService(),
		service.NewSearchService(),
		service.NewConfigService(),
	)

	mux := http.NewServeMux()
	s.RegisterRoutes(mux)

	addrStr := *addr
	fmt.Printf("TagFile Web UI 已启动: http://localhost%s\n", addrStr)
	if err := http.ListenAndServe(addrStr, mux); err != nil {
		panic(err)
	}
}
