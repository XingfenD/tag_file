package handlers

import "net/http"

func (s *Server) HandleHome(w http.ResponseWriter, r *http.Request) {
	files, _, _ := s.FileSvc.ListFiles(0, 0)
	categories, _, _ := s.CategorySvc.ListCategories(0, 0)
	tags, _, _ := s.TagSvc.ListTags("", 0, 0)
	configs, _, _ := s.ConfigSvc.List(0, 0)

	data := map[string]any{
		"Active":        "home",
		"FileCount":     len(files),
		"CategoryCount": len(categories),
		"TagCount":      len(tags),
		"ConfigCount":   len(configs),
	}
	s.Render(w, "home", data)
}
