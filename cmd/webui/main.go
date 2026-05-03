package main

import (
	"embed"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"github.com/XingfenD/tag_file/internal/bootstrap"
	"github.com/XingfenD/tag_file/internal/service"
)

//go:embed templates/*
var templateFS embed.FS

var funcMap = template.FuncMap{
	"formatSize": formatSize,
}

type server struct {
	pages       map[string]*template.Template
	categorySvc *service.CategoryService
	tagSvc      *service.TagService
	attrSvc     *service.AttrService
	fileSvc     *service.FileService
	searchSvc   *service.SearchService
	configSvc   *service.ConfigService
}

func main() {
	if err := bootstrap.InitDB(); err != nil {
		panic(err)
	}

	base, err := template.New("base").Funcs(funcMap).ParseFS(templateFS, "templates/base.html")
	if err != nil {
		panic(err)
	}

	pages := make(map[string]*template.Template)
	for _, name := range []string{"home", "category", "tag", "attribute", "file", "search", "config"} {
		tmpl, err := template.Must(base.Clone()).ParseFS(templateFS, "templates/"+name+".html")
		if err != nil {
			panic(fmt.Sprintf("parse %s: %v", name, err))
		}
		pages[name] = tmpl
	}

	s := &server{
		pages:       pages,
		categorySvc: service.NewCategoryService(),
		tagSvc:      service.NewTagService(),
		attrSvc:     service.NewAttrService(),
		fileSvc:     service.NewFileService(),
		searchSvc:   service.NewSearchService(),
		configSvc:   service.NewConfigService(),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /", s.handleHome)
	mux.HandleFunc("GET /categories", s.handleListCategories)
	mux.HandleFunc("POST /categories/add", s.handleAddCategory)
	mux.HandleFunc("POST /categories/remove", s.handleRemoveCategory)
	mux.HandleFunc("GET /tags", s.handleListTags)
	mux.HandleFunc("POST /tags/add", s.handleAddTag)
	mux.HandleFunc("POST /tags/remove", s.handleRemoveTag)
	mux.HandleFunc("GET /attributes", s.handleListAttributes)
	mux.HandleFunc("POST /attributes/add", s.handleAddAttribute)
	mux.HandleFunc("POST /attributes/remove", s.handleRemoveAttribute)
	mux.HandleFunc("POST /attributes/values/add", s.handleAddValue)
	mux.HandleFunc("POST /attributes/values/remove", s.handleRemoveValue)
	mux.HandleFunc("GET /files", s.handleListFiles)
	mux.HandleFunc("POST /files/scan", s.handleScanFiles)
	mux.HandleFunc("POST /files/import", s.handleImportFile)
	mux.HandleFunc("POST /files/tag", s.handleTagFile)
	mux.HandleFunc("POST /files/untag", s.handleUntagFile)
	mux.HandleFunc("GET /files/info", s.handleFileInfo)
	mux.HandleFunc("GET /search", s.handleSearchForm)
	mux.HandleFunc("POST /search", s.handleSearch)
	mux.HandleFunc("GET /config", s.handleListConfig)
	mux.HandleFunc("POST /config/set", s.handleSetConfig)
	mux.HandleFunc("POST /config/unset", s.handleUnsetConfig)

	addr := ":8088"
	fmt.Printf("TagFile Web UI 已启动: http://localhost%s\n", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		panic(err)
	}
}

func (s *server) render(w http.ResponseWriter, page string, data map[string]any) {
	if data == nil {
		data = map[string]any{}
	}
	data["Flash"] = ""
	data["FlashType"] = ""
	tmpl := s.pages[page]
	if tmpl == nil {
		http.Error(w, "template not found: "+page, http.StatusInternalServerError)
		return
	}
	if err := tmpl.ExecuteTemplate(w, "base", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *server) redirect(w http.ResponseWriter, r *http.Request, url, msg, msgType string) {
	if msg != "" {
		sep := "?"
		if strings.Contains(url, "?") {
			sep = "&"
		}
		url = fmt.Sprintf("%s%sflash=%s&flash_type=%s", url, sep, msg, msgType)
	}
	http.Redirect(w, r, url, http.StatusSeeOther)
}

func getFlash(r *http.Request) (string, string) {
	return r.URL.Query().Get("flash"), r.URL.Query().Get("flash_type")
}

func parseInt(s string) (uint, error) {
	v, err := strconv.ParseUint(s, 10, 64)
	return uint(v), err
}

type pageData struct {
	Page       int
	PageSize   int
	Total      int64
	TotalPages int
	HasPrev    bool
	HasNext    bool
	PrevPage   int
	NextPage   int
	Start      int
	End        int
}

func paginate(r *http.Request, total int64, pageSize int) pageData {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}
	if total > 0 {
		if totalPages < 1 {
			totalPages = 1
		}
		if page > totalPages {
			page = totalPages
		}
	}
	offset := (page - 1) * pageSize
	end := offset + pageSize
	if end > int(total) {
		end = int(total)
	}
	return pageData{
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: totalPages,
		HasPrev:    page > 1,
		HasNext:    page < totalPages,
		PrevPage:   page - 1,
		NextPage:   page + 1,
		Start:      offset + 1,
		End:        end,
	}
}

func paginationBaseURL(r *http.Request) string {
	base := r.URL.Path + "?"
	skip := map[string]bool{"page": true}
	for k, vs := range r.URL.Query() {
		if skip[k] {
			continue
		}
		for _, v := range vs {
			base += k + "=" + v + "&"
		}
	}
	return base + "page="
}

// ---- Home ----

func (s *server) handleHome(w http.ResponseWriter, r *http.Request) {
	files, _, _ := s.fileSvc.ListFiles(0, 0)
	categories, _, _ := s.categorySvc.ListCategories(0, 0)
	tags, _, _ := s.tagSvc.ListTags("", 0, 0)
	configs, _, _ := s.configSvc.List(0, 0)

	data := map[string]any{
		"Active":        "home",
		"FileCount":     len(files),
		"CategoryCount": len(categories),
		"TagCount":      len(tags),
		"ConfigCount":   len(configs),
	}
	s.render(w, "home", data)
}

// ---- Categories ----

func (s *server) handleListCategories(w http.ResponseWriter, r *http.Request) {
	pg := paginate(r, 0, 10)
	categories, total, err := s.categorySvc.ListCategories(pg.Start-1, pg.PageSize)
	pg.Total = total
	if total > 0 {
		pg.TotalPages = int(total) / pg.PageSize
		if int(total)%pg.PageSize > 0 {
			pg.TotalPages++
		}
	}
	pg.HasPrev = pg.Page > 1
	pg.HasNext = pg.Page < pg.TotalPages
	if err != nil {
		s.redirect(w, r, "/categories", err.Error(), "error")
		return
	}
	flash, flashType := getFlash(r)
	data := map[string]any{
		"Active":     "categories",
		"Categories": categories,
		"PageData":   pg,
		"BaseURL":    paginationBaseURL(r),
		"Flash":      flash,
		"FlashType":  flashType,
	}
	s.render(w, "category", data)
}

func (s *server) handleAddCategory(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	desc := r.FormValue("description")
	if _, err := s.categorySvc.AddCategory(name, desc); err != nil {
		s.redirect(w, r, "/categories", err.Error(), "error")
		return
	}
	s.redirect(w, r, "/categories", "分类 «"+name+"» 已添加", "success")
}

func (s *server) handleRemoveCategory(w http.ResponseWriter, r *http.Request) {
	idStr := r.FormValue("id")
	id, err := parseInt(idStr)
	if err != nil {
		s.redirect(w, r, "/categories", "无效的ID", "error")
		return
	}
	categories, _, err := s.categorySvc.ListCategories(0, 0)
	if err == nil {
		for _, c := range categories {
			if c.CategoryID == id {
				if err := s.categorySvc.RemoveCategory(c.CategoryName); err != nil {
					s.redirect(w, r, "/categories", err.Error(), "error")
					return
				}
				s.redirect(w, r, "/categories", "分类 «"+c.CategoryName+"» 已删除", "success")
				return
			}
		}
	}
	s.redirect(w, r, "/categories", "分类不存在", "error")
}

// ---- Tags ----

func (s *server) handleListTags(w http.ResponseWriter, r *http.Request) {
	categories, _, _ := s.categorySvc.ListCategories(0, 0)
	categoryIDStr := r.URL.Query().Get("category_id")

	pg := paginate(r, 0, 10)
	var tags any
	var total int64
	var err error
	var filterCategory uint
	if categoryIDStr != "" {
		id, parseErr := parseInt(categoryIDStr)
		if parseErr == nil {
			filterCategory = id
			for _, c := range categories {
				if c.CategoryID == id {
					tags, total, err = s.tagSvc.ListTags(c.CategoryName, pg.Start-1, pg.PageSize)
					break
				}
			}
		}
	}
	if tags == nil {
		tags, total, err = s.tagSvc.ListTags("", pg.Start-1, pg.PageSize)
	}
	if err != nil {
		s.redirect(w, r, "/tags", err.Error(), "error")
		return
	}
	pg.Total = total
	if total > 0 {
		pg.TotalPages = int(total) / pg.PageSize
		if int(total)%pg.PageSize > 0 {
			pg.TotalPages++
		}
	}
	pg.HasPrev = pg.Page > 1
	pg.HasNext = pg.Page < pg.TotalPages

	flash, flashType := getFlash(r)
	data := map[string]any{
		"Active":         "tags",
		"Categories":     categories,
		"Tags":           tags,
		"FilterCategory": filterCategory,
		"PageData":       pg,
		"BaseURL":        paginationBaseURL(r),
		"Flash":          flash,
		"FlashType":      flashType,
	}
	s.render(w, "tag", data)
}

func (s *server) handleAddTag(w http.ResponseWriter, r *http.Request) {
	categoryIDStr := r.FormValue("category_id")
	tagName := r.FormValue("tag_name")
	desc := r.FormValue("description")

	categoryID, err := parseInt(categoryIDStr)
	if err != nil {
		s.redirect(w, r, "/tags", "无效的分类ID", "error")
		return
	}

	categories, _, _ := s.categorySvc.ListCategories(0, 0)
	var categoryName string
	for _, c := range categories {
		if c.CategoryID == categoryID {
			categoryName = c.CategoryName
			break
		}
	}
	if categoryName == "" {
		s.redirect(w, r, "/tags", "分类不存在", "error")
		return
	}

	if _, err := s.tagSvc.AddTag(categoryName, tagName, desc); err != nil {
		s.redirect(w, r, "/tags", err.Error(), "error")
		return
	}
	s.redirect(w, r, "/tags", "标签 «"+tagName+"» 已添加", "success")
}

func (s *server) handleRemoveTag(w http.ResponseWriter, r *http.Request) {
	idStr := r.FormValue("id")
	id, err := parseInt(idStr)
	if err != nil {
		s.redirect(w, r, "/tags", "无效的ID", "error")
		return
	}

	tags, _, _ := s.tagSvc.ListTags("", 0, 0)
	for _, t := range tags {
		if t.TagID == id {
			if _, err := s.tagSvc.RemoveTag(t.TagName); err != nil {
				s.redirect(w, r, "/tags", err.Error(), "error")
				return
			}
			s.redirect(w, r, "/tags", "标签 «"+t.TagName+"» 已删除", "success")
			return
		}
	}
	s.redirect(w, r, "/tags", "标签不存在", "error")
}

// ---- Attributes ----

func (s *server) handleListAttributes(w http.ResponseWriter, r *http.Request) {
	tags, _, _ := s.tagSvc.ListTags("", 0, 0)
	tagIDStr := r.URL.Query().Get("tag_id")
	attrValuesForStr := r.URL.Query().Get("attr_values_for")

	pg := paginate(r, 0, 10)
	var attributes any
	var attrValues any
	var total int64
	var filterTag uint
	if tagIDStr != "" {
		id, parseErr := parseInt(tagIDStr)
		if parseErr == nil {
			filterTag = id
			for _, t := range tags {
				if t.TagID == id {
					attributes, total, _ = s.attrSvc.ListAttributes(t.TagName, pg.Start-1, pg.PageSize)
					break
				}
			}
		}
	}
	if attributes == nil {
		attributes, total, _ = s.attrSvc.ListAttributes("", pg.Start-1, pg.PageSize)
	}
	pg.Total = total
	if total > 0 {
		pg.TotalPages = int(total) / pg.PageSize
		if int(total)%pg.PageSize > 0 {
			pg.TotalPages++
		}
	}
	pg.HasPrev = pg.Page > 1
	pg.HasNext = pg.Page < pg.TotalPages

	if attrValuesForStr != "" {
		attrID, err := parseInt(attrValuesForStr)
		if err == nil {
			attrValues, _, _ = s.attrSvc.ListValues(attrID, 0, 0)
		}
	}

	flash, flashType := getFlash(r)
	data := map[string]any{
		"Active":     "attributes",
		"Tags":       tags,
		"Attributes": attributes,
		"AttrValues": attrValues,
		"FilterTag":  filterTag,
		"PageData":   pg,
		"BaseURL":    paginationBaseURL(r),
		"Flash":      flash,
		"FlashType":  flashType,
	}
	s.render(w, "attribute", data)
}

func (s *server) handleAddAttribute(w http.ResponseWriter, r *http.Request) {
	tagIDStr := r.FormValue("tag_id")
	attrName := r.FormValue("attr_name")
	desc := r.FormValue("description")

	tagID, err := parseInt(tagIDStr)
	if err != nil {
		s.redirect(w, r, "/attributes", "无效的标签ID", "error")
		return
	}

	tags, _, _ := s.tagSvc.ListTags("", 0, 0)
	var tagName string
	for _, t := range tags {
		if t.TagID == tagID {
			tagName = t.TagName
			break
		}
	}
	if tagName == "" {
		s.redirect(w, r, "/attributes", "标签不存在", "error")
		return
	}

	if _, err := s.attrSvc.AddAttribute(tagName, attrName, desc); err != nil {
		s.redirect(w, r, "/attributes", err.Error(), "error")
		return
	}
	s.redirect(w, r, "/attributes", "属性 «"+attrName+"» 已添加", "success")
}

func (s *server) handleRemoveAttribute(w http.ResponseWriter, r *http.Request) {
	idStr := r.FormValue("id")
	id, err := parseInt(idStr)
	if err != nil {
		s.redirect(w, r, "/attributes", "无效的ID", "error")
		return
	}
	if err := s.attrSvc.RemoveAttribute(id); err != nil {
		s.redirect(w, r, "/attributes", err.Error(), "error")
		return
	}
	s.redirect(w, r, "/attributes", "属性已删除", "success")
}

func (s *server) handleAddValue(w http.ResponseWriter, r *http.Request) {
	attrIDStr := r.FormValue("attr_id")
	valueName := r.FormValue("value_name")

	attrID, err := parseInt(attrIDStr)
	if err != nil {
		s.redirect(w, r, "/attributes", "无效的属性ID", "error")
		return
	}
	if _, err := s.attrSvc.AddValue(attrID, valueName); err != nil {
		s.redirect(w, r, "/attributes", err.Error(), "error")
		return
	}
	s.redirect(w, r, "/attributes", "属性值 «"+valueName+"» 已添加", "success")
}

func (s *server) handleRemoveValue(w http.ResponseWriter, r *http.Request) {
	idStr := r.FormValue("id")
	id, err := parseInt(idStr)
	if err != nil {
		s.redirect(w, r, "/attributes", "无效的ID", "error")
		return
	}
	if err := s.attrSvc.RemoveValue(id); err != nil {
		s.redirect(w, r, "/attributes", err.Error(), "error")
		return
	}
	s.redirect(w, r, "/attributes", "属性值已删除", "success")
}

// ---- Files ----

func (s *server) handleListFiles(w http.ResponseWriter, r *http.Request) {
	pg := paginate(r, 0, 10)
	files, total, err := s.fileSvc.ListFiles(pg.Start-1, pg.PageSize)
	pg.Total = total
	pg.TotalPages = int(total) / pg.PageSize
	if int(total)%pg.PageSize > 0 {
		pg.TotalPages++
	}
	pg.HasPrev = pg.Page > 1
	pg.HasNext = pg.Page < pg.TotalPages
	pg.End = pg.Start + len(files) - 1
	if err != nil {
		s.redirect(w, r, "/files", err.Error(), "error")
		return
	}
	flash, flashType := getFlash(r)
	data := map[string]any{
		"Active":    "files",
		"Files":     files,
		"PageData":  pg,
		"BaseURL":   paginationBaseURL(r),
		"Flash":     flash,
		"FlashType": flashType,
	}
	s.render(w, "file", data)
}

func (s *server) handleScanFiles(w http.ResponseWriter, r *http.Request) {
	dir := r.FormValue("dir")
	if dir == "" {
		dir = "."
	}
	result, err := s.fileSvc.ScanDirectory(dir)
	if err != nil {
		s.redirect(w, r, "/files", err.Error(), "error")
		return
	}
	msg := fmt.Sprintf("扫描完成: 发现 %d 个文件, 新增 %d 个", result.FilesFound, result.FilesAdded)
	s.redirect(w, r, "/files", msg, "success")
}

func (s *server) handleImportFile(w http.ResponseWriter, r *http.Request) {
	path := r.FormValue("path")
	if path == "" {
		s.redirect(w, r, "/files", "请输入文件路径", "error")
		return
	}
	file, err := s.fileSvc.ImportFile(path)
	if err != nil {
		s.redirect(w, r, "/files", err.Error(), "error")
		return
	}
	s.redirect(w, r, "/files", "文件 «"+file.FileName+"» 已导入", "success")
}

func (s *server) handleTagFile(w http.ResponseWriter, r *http.Request) {
	fileIDStr := r.FormValue("file_id")
	tagName := r.FormValue("tag_name")

	fileID, err := parseInt(fileIDStr)
	if err != nil {
		s.redirect(w, r, "/files", "无效的文件ID", "error")
		return
	}

	files, _, _ := s.fileSvc.ListFiles(0, 0)
	for _, f := range files {
		if f.FileID == fileID {
			if err := s.fileSvc.TagFile(f.FilePath, tagName); err != nil {
				s.redirect(w, r, "/files", err.Error(), "error")
				return
			}
			s.redirect(w, r, "/files", "标签 «"+tagName+"» 已添加到 «"+f.FileName+"»", "success")
			return
		}
	}
	s.redirect(w, r, "/files", "文件不存在", "error")
}

func (s *server) handleUntagFile(w http.ResponseWriter, r *http.Request) {
	fileIDStr := r.FormValue("file_id")
	tagName := r.FormValue("tag_name")

	fileID, err := parseInt(fileIDStr)
	if err != nil {
		s.redirect(w, r, "/files", "无效的文件ID", "error")
		return
	}

	files, _, _ := s.fileSvc.ListFiles(0, 0)
	for _, f := range files {
		if f.FileID == fileID {
			if err := s.fileSvc.UntagFile(f.FilePath, tagName); err != nil {
				s.redirect(w, r, "/files", err.Error(), "error")
				return
			}
			s.redirect(w, r, "/files", "标签 «"+tagName+"» 已从 «"+f.FileName+"» 移除", "success")
			return
		}
	}
	s.redirect(w, r, "/files", "文件不存在", "error")
}

func (s *server) handleFileInfo(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, err := parseInt(idStr)
	if err != nil {
		s.redirect(w, r, "/files", "无效的文件ID", "error")
		return
	}

	files, _, _ := s.fileSvc.ListFiles(0, 0)
	var filePath string
	for _, f := range files {
		if f.FileID == id {
			filePath = f.FilePath
			break
		}
	}
	if filePath == "" {
		s.redirect(w, r, "/files", "文件不存在", "error")
		return
	}

	file, tags, err := s.fileSvc.GetFileInfo(filePath)
	if err != nil {
		s.redirect(w, r, "/files", err.Error(), "error")
		return
	}

	flash, flashType := getFlash(r)
	data := map[string]any{
		"Active":    "files",
		"Files":     files,
		"FileInfo":  map[string]any{"File": file, "Tags": tags},
		"Flash":     flash,
		"FlashType": flashType,
	}
	s.render(w, "file", data)
}

// ---- Search ----

func (s *server) handleSearchForm(w http.ResponseWriter, r *http.Request) {
	flash, flashType := getFlash(r)
	data := map[string]any{
		"Active":    "search",
		"Flash":     flash,
		"FlashType": flashType,
	}
	s.render(w, "search", data)
}

func (s *server) handleSearch(w http.ResponseWriter, r *http.Request) {
	includeStr := r.FormValue("include")
	excludeStr := r.FormValue("exclude")
	logic := r.FormValue("logic")
	if logic == "" {
		logic = "and"
	}

	var include, exclude []string
	if includeStr != "" {
		for _, s := range strings.Split(includeStr, ",") {
			s = strings.TrimSpace(s)
			if s != "" {
				include = append(include, s)
			}
		}
	}
	if excludeStr != "" {
		for _, s := range strings.Split(excludeStr, ",") {
			s = strings.TrimSpace(s)
			if s != "" {
				exclude = append(exclude, s)
			}
		}
	}

	if len(include) == 0 && len(exclude) == 0 {
		flash, flashType := getFlash(r)
		data := map[string]any{
			"Active":    "search",
			"Include":   includeStr,
			"Exclude":   excludeStr,
			"Logic":     logic,
			"Flash":     flash,
			"FlashType": flashType,
			"Error":     "请至少输入一个搜索条件",
		}
		s.render(w, "search", data)
		return
	}

	results, err := s.searchSvc.SearchByTags(include, exclude, logic)
	if err != nil {
		flash, flashType := getFlash(r)
		data := map[string]any{
			"Active":    "search",
			"Include":   includeStr,
			"Exclude":   excludeStr,
			"Logic":     logic,
			"Flash":     flash,
			"FlashType": flashType,
		}
		s.render(w, "search", data)
		return
	}

	flash, flashType := getFlash(r)
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
	s.render(w, "search", data)
}

// ---- Config ----

func (s *server) handleListConfig(w http.ResponseWriter, r *http.Request) {
	pg := paginate(r, 0, 10)
	configs, total, err := s.configSvc.List(pg.Start-1, pg.PageSize)
	pg.Total = total
	if total > 0 {
		pg.TotalPages = int(total) / pg.PageSize
		if int(total)%pg.PageSize > 0 {
			pg.TotalPages++
		}
	}
	pg.HasPrev = pg.Page > 1
	pg.HasNext = pg.Page < pg.TotalPages
	if err != nil {
		s.redirect(w, r, "/config", err.Error(), "error")
		return
	}
	flash, flashType := getFlash(r)
	data := map[string]any{
		"Active":    "config",
		"Configs":   configs,
		"PageData":  pg,
		"BaseURL":   paginationBaseURL(r),
		"Flash":     flash,
		"FlashType": flashType,
	}
	s.render(w, "config", data)
}

func (s *server) handleSetConfig(w http.ResponseWriter, r *http.Request) {
	key := r.FormValue("key")
	value := r.FormValue("value")
	if err := s.configSvc.Set(key, value); err != nil {
		s.redirect(w, r, "/config", err.Error(), "error")
		return
	}
	s.redirect(w, r, "/config", "配置 «"+key+"» 已保存", "success")
}

func (s *server) handleUnsetConfig(w http.ResponseWriter, r *http.Request) {
	key := r.FormValue("key")
	if err := s.configSvc.Delete(key); err != nil {
		s.redirect(w, r, "/config", err.Error(), "error")
		return
	}
	s.redirect(w, r, "/config", "配置 «"+key+"» 已删除", "success")
}

func formatSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}
