package handlers

import (
	"net/http"
	"strings"
)

func (s *Server) HandleSearchForm(w http.ResponseWriter, r *http.Request) {
	flash, flashType := GetFlash(r)
	data := map[string]any{
		"Active":    "search",
		"Flash":     flash,
		"FlashType": flashType,
	}
	s.Render(w, "search", data)
}

func (s *Server) HandleSearch(w http.ResponseWriter, r *http.Request) {
	includeStr := r.FormValue("include")
	excludeStr := r.FormValue("exclude")
	logic := r.FormValue("logic")
	if logic == "" {
		logic = "and"
	}

	include := splitTags(includeStr)
	exclude := splitTags(excludeStr)

	if len(include) == 0 && len(exclude) == 0 {
		flash, flashType := GetFlash(r)
		data := map[string]any{
			"Active":    "search",
			"Include":   includeStr,
			"Exclude":   excludeStr,
			"Logic":     logic,
			"Flash":     flash,
			"FlashType": flashType,
			"Error":     "请至少输入一个搜索条件",
		}
		s.Render(w, "search", data)
		return
	}

	results, err := s.SearchSvc.SearchByTags(include, exclude, logic)
	if err != nil {
		flash, flashType := GetFlash(r)
		data := map[string]any{
			"Active":    "search",
			"Include":   includeStr,
			"Exclude":   excludeStr,
			"Logic":     logic,
			"Flash":     flash,
			"FlashType": flashType,
		}
		s.Render(w, "search", data)
		return
	}

	flash, flashType := GetFlash(r)
	data := map[string]any{
		"Active":    "search",
		"Include":   includeStr,
		"Exclude":   excludeStr,
		"Logic":     logic,
		"Results":   results,
		"Searched":  true,
		"Flash":     flash,
		"FlashType": flashType,
	}
	s.Render(w, "search", data)
}

func splitTags(s string) []string {
	if s == "" {
		return nil
	}
	var result []string
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}
