package handlers

import (
	"html/template"

	"github.com/gin-gonic/gin/render"
)

type TemplateRenderer struct {
	pages map[string]*template.Template
}

func NewTemplateRenderer(pages map[string]*template.Template) *TemplateRenderer {
	return &TemplateRenderer{pages: pages}
}

func (r *TemplateRenderer) Instance(name string, data any) render.Render {
	return render.HTML{
		Template: r.pages[name],
		Data:     data,
	}
}
