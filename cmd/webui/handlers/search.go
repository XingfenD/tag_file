package handlers

import (
	"net/http"
	"strings"
)

// HandleSearch handles POST /search with server-side rendering.
// Kept temporarily until search is overhauled to use the api package.
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
		data := map[string]any{
			"Active":  "search",
			"Include": includeStr,
			"Exclude": excludeStr,
			"Logic":   logic,
			"Error":   "请至少输入一个搜索条件",
		}
		s.Render(w, "search", data)
		return
	}

	results, err := s.SearchSvc.SearchByTags(include, exclude, logic)
	if err != nil {
		data := map[string]any{
			"Active":  "search",
			"Include": includeStr,
			"Exclude": excludeStr,
			"Logic":   logic,
		}
		s.Render(w, "search", data)
		return
	}

	data := map[string]any{
		"Active":   "search",
		"Include":  includeStr,
		"Exclude":  excludeStr,
		"Logic":    logic,
		"Results":  results,
		"Searched": true,
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
