package handlers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/XingfenD/tag_file/internal/model"
	configSvc "github.com/XingfenD/tag_file/internal/service/config"
)

type FileTreeNode struct {
	Name     string
	Path     string
	IsDir    bool
	Indexed  bool
	File     *model.File
	FileSize int64
	TagCnt   int
	Children []*FileTreeNode
}

func buildFileTreeFromFS(rootDir string, excludeExts map[string]bool, excludePatterns []*regexp.Regexp) ([]*FileTreeNode, error) {
	root, err := filepath.Abs(rootDir)
	if err != nil {
		return nil, err
	}

	treeRoot := &FileTreeNode{Name: root, IsDir: true}

	isExcluded := func(name string) bool {
		for _, r := range excludePatterns {
			if r.MatchString(name) {
				return true
			}
		}
		return false
	}

	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		name := d.Name()
		if name == "." || name == ".." {
			return nil
		}
		if path == root {
			return nil
		}
		if d.IsDir() && isExcluded(name) {
			return filepath.SkipDir
		}
		if !d.IsDir() && isExcluded(name) {
			return nil
		}

		relPath, _ := filepath.Rel(root, path)
		dirs, filename := splitFilePath(relPath)

		if !d.IsDir() && excludeExts != nil && len(excludeExts) > 0 {
			ext := strings.ToLower(filepath.Ext(filename))
			if excludeExts[ext] {
				return nil
			}
		}

		cur := treeRoot
		for _, dir := range dirs {
			found := false
			for _, c := range cur.Children {
				if c.Name == dir && c.IsDir {
					cur = c
					found = true
					break
				}
			}
			if !found {
				child := &FileTreeNode{Name: dir, IsDir: true}
				cur.Children = append(cur.Children, child)
				cur = child
			}
		}

		if d.IsDir() {
			cur.Children = append(cur.Children, &FileTreeNode{
				Name:  filename,
				Path:  relPath,
				IsDir: true,
			})
		} else {
			info, _ := d.Info()
			node := &FileTreeNode{
				Name:     filename,
				Path:     relPath,
				FileSize: info.Size(),
			}
			cur.Children = append(cur.Children, node)
		}
		return nil
	})

	sortTree(treeRoot)
	pruneEmptyDirs(treeRoot)
	return treeRoot.Children, err
}

func pruneEmptyDirs(node *FileTreeNode) {
	var kept []*FileTreeNode
	for _, c := range node.Children {
		if c.IsDir {
			pruneEmptyDirs(c)
			if len(c.Children) == 0 {
				continue
			}
		}
		kept = append(kept, c)
	}
	node.Children = kept
}

func markTreeIndexed(nodes []*FileTreeNode, indexed map[string]*model.File) {
	for _, n := range nodes {
		if n.IsDir {
			markTreeIndexed(n.Children, indexed)
		} else {
			if f, ok := indexed[n.Path]; ok {
				n.Indexed = true
				n.File = f
			}
		}
	}
}

func computeDirTagCnt(node *FileTreeNode) int {
	if !node.IsDir {
		if node.File != nil {
			node.TagCnt = int(node.File.TagCnt)
			return node.TagCnt
		}
		return 0
	}
	sum := 0
	for _, c := range node.Children {
		sum += computeDirTagCnt(c)
	}
	node.TagCnt = sum
	return sum
}

func splitFilePath(path string) (dirs []string, filename string) {
	path = filepath.ToSlash(path)
	if idx := strings.LastIndex(path, "/"); idx >= 0 {
		return strings.Split(path[:idx], "/"), path[idx+1:]
	}
	return nil, path
}

func sortTree(node *FileTreeNode) {
	sort.Slice(node.Children, func(i, j int) bool {
		a, b := node.Children[i], node.Children[j]
		if a.IsDir != b.IsDir {
			return a.IsDir
		}
		return a.Name < b.Name
	})
	for _, c := range node.Children {
		if c.IsDir {
			sortTree(c)
		}
	}
}

func (s *Server) HandleListFiles(w http.ResponseWriter, r *http.Request) {
	excludeExts := getHideExtensions(s.ConfigSvc)
	excludePatterns := loadExcludePatterns(s.ConfigSvc)
	patternsStr := getScanExcludeRaw(s.ConfigSvc)

	var tree []*FileTreeNode
	cached, cacheErr := loadTreeCache(patternsStr)
	if cacheErr == nil {
		tree = cached
	} else {
		var err error
		tree, err = buildFileTreeFromFS(".", excludeExts, excludePatterns)
		if err != nil {
			data := map[string]any{
				"Active":    "files",
				"Flash":     "遍历文件系统失败: " + err.Error(),
				"FlashType": "error",
			}
			s.Render(w, "file", data)
			return
		}
		saveTreeCache(tree, patternsStr)
	}

	indexed, _, _ := s.FileSvc.ListFiles(0, 0)
	indexedMap := make(map[string]*model.File, len(indexed))
	for i := range indexed {
		indexedMap[indexed[i].FilePath] = &indexed[i]
	}
	markTreeIndexed(tree, indexedMap)
	for _, n := range tree {
		computeDirTagCnt(n)
	}
	data := map[string]any{
		"Active": "files",
		"Tree":   tree,
	}
	s.Render(w, "file", data)
}

func (s *Server) HandleRefreshTree(w http.ResponseWriter, r *http.Request) {
	os.Remove(treeCacheFile)
	respond(w, "目录缓存已刷新", "success")
}

func (s *Server) HandleScanFiles(w http.ResponseWriter, r *http.Request) {
	dir := r.FormValue("dir")
	if dir == "" {
		dir = "."
	}
	result, err := s.FileSvc.ScanDirectoryOptions(dir, true)
	if err != nil {
		respond(w, err.Error(), "error")
		return
	}
	os.Remove(treeCacheFile)
	msg := fmt.Sprintf("扫描完成: 发现 %d 个文件, 新增 %d 个", result.FilesFound, result.FilesAdded)
	respond(w, msg, "success")
}

func (s *Server) HandleImportFile(w http.ResponseWriter, r *http.Request) {
	upload, header, upErr := r.FormFile("file")
	if upErr == nil {
		defer upload.Close()
		dst, err := os.Create(header.Filename)
		if err != nil {
			respond(w, "创建文件失败: "+err.Error(), "error")
			return
		}
		defer dst.Close()
		if _, err := io.Copy(dst, upload); err != nil {
			os.Remove(header.Filename)
			respond(w, "写入文件失败: "+err.Error(), "error")
			return
		}
		file, err := s.FileSvc.ImportFile(header.Filename)
		if err != nil {
			respond(w, err.Error(), "error")
			return
		}
		os.Remove(treeCacheFile)
		respond(w, "文件 «"+file.FileName+"» 已导入", "success")
		return
	}

	path := r.FormValue("path")
	if path == "" {
		respond(w, "请选择文件或输入路径", "error")
		return
	}
	file, err := s.FileSvc.ImportFile(path)
	if err != nil {
		respond(w, err.Error(), "error")
		return
	}
	os.Remove(treeCacheFile)
	respond(w, "文件 «"+file.FileName+"» 已导入", "success")
}

func (s *Server) HandleTagFile(w http.ResponseWriter, r *http.Request) {
	filePath := r.FormValue("file_path")
	tagIDStr := r.FormValue("tag_id")
	tagValue := r.FormValue("tag_value")
	groupIDStr := r.FormValue("group_id")

	if filePath == "" {
		respond(w, "无效的文件路径", "error")
		return
	}

	tagID, parseErr := ParseInt(tagIDStr)
	if parseErr != nil || tagID == 0 {
		respond(w, "无效的标签ID", "error")
		return
	}

	var err error
	if groupIDStr != "" {
		if groupID, gpErr := ParseInt(groupIDStr); gpErr == nil && groupID > 0 {
			err = s.FileSvc.TagFileToGroup(filePath, uint(tagID), tagValue, uint(groupID))
		}
	}
	if groupIDStr == "" || err != nil {
		err = s.FileSvc.TagFile(filePath, uint(tagID), tagValue)
	}
	if err != nil {
		respond(w, err.Error(), "error")
		return
	}
	respond(w, "标签已添加", "success")
}

func (s *Server) HandleUntagFile(w http.ResponseWriter, r *http.Request) {
	filePath := r.FormValue("file_path")
	tagIDStr := r.FormValue("tag_id")
	tagValue := r.FormValue("tag_value")
	groupIDStr := r.FormValue("group_id")

	if filePath == "" {
		respond(w, "无效的文件路径", "error")
		return
	}

	tagID, parseErr := ParseInt(tagIDStr)
	if parseErr != nil || tagID == 0 {
		respond(w, "无效的标签ID", "error")
		return
	}

	var err error
	if groupIDStr != "" {
		if groupID, gpErr := ParseInt(groupIDStr); gpErr == nil && groupID > 0 {
			err = s.FileSvc.UntagFileFromGroup(filePath, uint(tagID), tagValue, uint(groupID))
		}
	}
	if groupIDStr == "" || err != nil {
		err = s.FileSvc.UntagFile(filePath, uint(tagID), tagValue)
	}
	if err != nil {
		respond(w, err.Error(), "error")
		return
	}
	respond(w, "标签已移除", "success")
}

func (s *Server) HandleUpdateTagValue(w http.ResponseWriter, r *http.Request) {
	idStr := r.FormValue("id")
	ftID, err := ParseInt(idStr)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	val := r.FormValue("value")
	if err := s.FileSvc.UpdateTagValue(ftID, val); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) HandleAddGroup(w http.ResponseWriter, r *http.Request) {
	filePath := r.FormValue("file_path")
	groupName := r.FormValue("group_name")
	if filePath == "" || groupName == "" {
		respond(w, "参数不完整", "error")
		return
	}
	group, err := s.FileSvc.CreateGroup(filePath, groupName)
	if err != nil {
		respond(w, err.Error(), "error")
		return
	}
	respond(w, "标签组 «"+groupName+"» 已创建", "success", map[string]any{
		"group_id": group.GroupID, "group_name": group.GroupName, "is_default": false,
	})
}

func (s *Server) HandleRemoveGroup(w http.ResponseWriter, r *http.Request) {
	filePath := r.FormValue("file_path")
	groupIDStr := r.FormValue("group_id")
	groupID, err := ParseInt(groupIDStr)
	if err != nil || groupID <= 0 {
		respond(w, "无效的标签组ID", "error")
		return
	}
	if err := s.FileSvc.DeleteGroup(filePath, uint(groupID)); err != nil {
		respond(w, err.Error(), "error")
		return
	}
	respond(w, "标签组已删除", "success")
}

func (s *Server) HandleRenameGroup(w http.ResponseWriter, r *http.Request) {
	filePath := r.FormValue("file_path")
	groupIDStr := r.FormValue("group_id")
	newName := r.FormValue("group_name")
	groupID, err := ParseInt(groupIDStr)
	if err != nil || groupID <= 0 || newName == "" {
		respond(w, "参数不完整", "error")
		return
	}
	if err := s.FileSvc.RenameGroup(filePath, uint(groupID), newName); err != nil {
		respond(w, err.Error(), "error")
		return
	}
	respond(w, "标签组已重命名", "success", map[string]any{
		"group_id": groupID, "group_name": newName,
	})
}

func (s *Server) HandleFileGroups(w http.ResponseWriter, r *http.Request) {
	filePath, err := url.PathUnescape(r.PathValue("path"))
	if err != nil {
		filePath = r.PathValue("path")
	}
	if filePath == "" {
		s.Redirect(w, r, "/files", "无效的文件路径", "error")
		return
	}

	file, _, _, err := s.FileSvc.GetFileInfo(filePath)
	if err != nil {
		s.Redirect(w, r, "/files", err.Error(), "error")
		return
	}

	groups, _ := s.FileSvc.ListGroups(filePath)

	type groupInfo struct {
		Group    model.TagGroup
		TagCount int64
	}
	var groupList []groupInfo
	for _, g := range groups {
		_, total, _ := s.FileSvc.ListFileTagsByGroup(filePath, g.GroupID, 0, 0)
		groupList = append(groupList, groupInfo{Group: g, TagCount: total})
	}

	data := map[string]any{
		"Active": "files",
		"File":   file,
		"Groups": groupList,
	}
	s.Render(w, "file_groups", data)
}

func (s *Server) HandleRenameFile(w http.ResponseWriter, r *http.Request) {
	filePath := r.FormValue("file_path")
	newName := r.FormValue("new_name")
	if filePath == "" || newName == "" {
		respond(w, "参数不完整", "error")
		return
	}
	_, err := s.FileSvc.RenameFile(filePath, newName)
	if err != nil {
		respond(w, err.Error(), "error")
		return
	}
	respond(w, "文件已重命名为 «"+newName+"»", "success")
}

func (s *Server) HandleFileInfo(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, err := ParseInt(idStr)
	if err != nil {
		s.Redirect(w, r, "/files", "无效的文件ID", "error")
		return
	}

	file, tags, _, err := s.fileInfoByID(id)
	if err != nil {
		s.Redirect(w, r, "/files", err.Error(), "error")
		return
	}

	files, _, _ := s.FileSvc.ListFiles(0, 0)
	mediaType := IsMediaType(file.FileType)
	data := map[string]any{
		"Active":   "files",
		"Files":    files,
		"FileInfo": map[string]any{"File": file, "Tags": tags, "MediaType": mediaType},
	}
	s.Render(w, "file", data)
}

func (s *Server) HandleFileDetail(w http.ResponseWriter, r *http.Request) {
	filePath, err := url.PathUnescape(r.PathValue("path"))
	if err != nil {
		filePath = r.PathValue("path")
	}
	if filePath == "" {
		s.Redirect(w, r, "/files", "无效的文件路径", "error")
		return
	}

	file, _, _, err := s.FileSvc.GetFileInfo(filePath)
	if err != nil {
		s.Redirect(w, r, "/files", err.Error(), "error")
		return
	}

	groups, _ := s.FileSvc.ListGroups(filePath)
	var selectedGroupID uint
	if gid, parseErr := ParseInt(r.URL.Query().Get("group_id")); parseErr == nil && gid > 0 {
		selectedGroupID = uint(gid)
	}
	if selectedGroupID == 0 && len(groups) > 0 {
		selectedGroupID = groups[0].GroupID
	}

	pg := Paginate(r, 0, 10)
	var fileTags []model.TagGroupTag
	var ftTotal int64
	if selectedGroupID > 0 {
		fileTags, ftTotal, _ = s.FileSvc.ListFileTagsByGroup(filePath, selectedGroupID, pg.Start-1, pg.PageSize)
	}
	UpdatePagination(&pg, ftTotal, len(fileTags))

	previewTags, _, _ := s.FileSvc.ListFileTags(filePath, 0, 20)

	// build recursive tag path helper
	allTags, _, _ := s.TagSvc.ListTags("", 0, 0)
	idToTagFile := make(map[uint]*model.Tag, len(allTags))
	for i := range allTags {
		idToTagFile[allTags[i].TagID] = &allTags[i]
	}
	var buildTagPath func(tagID uint, depth int) string
	buildTagPath = func(tagID uint, depth int) string {
		if depth > 20 {
			return "..."
		}
		t, ok := idToTagFile[tagID]
		if !ok {
			return "?"
		}
		if t.ParentID == nil {
			return t.TagName
		}
		parent, ok := idToTagFile[*t.ParentID]
		if !ok {
			return t.TagName
		}
		return buildTagPath(parent.TagID, depth+1) + " / " + t.TagName
	}
	tagPathMap := make(map[uint]string, len(allTags))
	for _, t := range allTags {
		tagPathMap[t.TagID] = buildTagPath(t.TagID, 0)
	}

	// deduplicate preview tags by display text (with full path)
	seen := make(map[string]bool, len(previewTags))
	deduped := make([]model.TagGroupTag, 0, len(previewTags))
	previewPaths := make(map[uint]string, len(previewTags))
	for _, ft := range previewTags {
		tagPath := tagPathMap[ft.Tag.TagID]
		text := tagPath
		if ft.TagValue != "" {
			text += ":" + ft.TagValue
		}
		if !seen[text] {
			seen[text] = true
			deduped = append(deduped, ft)
			previewPaths[ft.ID] = tagPath
		}
	}
	previewTags = deduped

	// compute display paths for file tags
	tagDisplayPaths := make(map[uint]string, len(fileTags))
	for _, ft := range fileTags {
		tagDisplayPaths[ft.ID] = tagPathMap[ft.Tag.TagID]
	}

	// AJAX partial: return only tag table HTML
	if r.URL.Query().Get("partial") == "tags" {
		s.renderTagTable(w, filePath, fileTags, pg, tagDisplayPaths)
		return
	}

	allTagsWithPath := make([]map[string]any, 0, len(allTags))
	for _, t := range allTags {
		path := buildTagPath(t.TagID, 0)
		allTagsWithPath = append(allTagsWithPath, map[string]any{
			"id":   t.TagID,
			"name": t.TagName,
			"path": path,
			"cat":  t.Category.CategoryName,
		})
	}
	sort.Slice(allTagsWithPath, func(i, j int) bool {
		return allTagsWithPath[i]["path"].(string) < allTagsWithPath[j]["path"].(string)
	})
	mediaType := IsMediaType(file.FileType)
	videoPlayable := mediaType == "video" && isPlayableVideo(file.FileType)

	dir := filepath.Dir(filePath)
	indexedPaths, _, _ := s.FileSvc.ListFiles(0, 0)
	indexedSet := make(map[string]bool)
	var dirIndexed []map[string]any
	for _, f := range indexedPaths {
		indexedSet[f.FilePath] = true
		if filepath.Dir(f.FilePath) == dir {
			dirIndexed = append(dirIndexed, map[string]any{
				"Name":      f.FileName,
				"Path":      f.FilePath,
				"Size":      f.FileSize,
				"Type":      f.FileType,
				"TagCnt":    f.TagCnt,
				"IsCurrent": f.FilePath == filePath,
			})
		}
	}

	// also include unindexed files in the directory
	entries, _ := os.ReadDir(dir)
	var dirUnindexed []map[string]any
	for _, e := range entries {
		if e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		p := filepath.Join(dir, e.Name())
		if indexedSet[p] {
			continue
		}
		info, _ := e.Info()
		dirUnindexed = append(dirUnindexed, map[string]any{
			"Name":      e.Name(),
			"Path":      p,
			"Size":      info.Size(),
			"Type":      mime.TypeByExtension(filepath.Ext(e.Name())),
			"IsCurrent": p == filePath,
		})
	}

	sort.Slice(dirUnindexed, func(i, j int) bool {
		return dirUnindexed[i]["Name"].(string) < dirUnindexed[j]["Name"].(string)
	})

	data := map[string]any{
		"Active":       "files",
		"FileInfo":     map[string]any{"File": file, "MediaType": mediaType, "VideoPlayable": videoPlayable},
		"PreviewTags":  previewTags,
		"PreviewPaths": previewPaths,
		"FileTags":     fileTags,
		"TagPaths":     tagDisplayPaths,
		"Groups":       groups,
		"GroupID":      selectedGroupID,
		"PageData":     pg,
		"PageFn":       template.JS("loadPage"),
		"BaseURL":      PaginationBaseURL(r),
		"AllTags":      allTagsWithPath,
		"DirPath":      dir,
		"DirIndexed":   dirIndexed,
		"DirUnindexed": dirUnindexed,
	}
	s.Render(w, "file_detail", data)
}

func (s *Server) HandleFileRaw(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, err := ParseInt(idStr)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	files, _, _ := s.FileSvc.ListFiles(0, 0)
	var filePath string
	for _, f := range files {
		if f.FileID == id {
			filePath = f.FilePath
			break
		}
	}
	if filePath == "" {
		http.Error(w, "file not found", http.StatusNotFound)
		return
	}

	http.ServeFile(w, r, filePath)
}

func (s *Server) fileInfoByID(id uint) (file *model.File, tags []model.Tag, valMap map[uint][]string, err error) {
	files, _, _ := s.FileSvc.ListFiles(0, 0)
	var filePath string
	for _, f := range files {
		if f.FileID == id {
			filePath = f.FilePath
			break
		}
	}
	if filePath == "" {
		return nil, nil, map[uint][]string{}, fmt.Errorf("文件不存在")
	}
	return s.FileSvc.GetFileInfo(filePath)
}

func getHideExtensions(cfgSvc *configSvc.Service) map[string]bool {
	cfg, err := cfgSvc.Get("webui.hide_extensions")
	if err != nil {
		return nil
	}
	exts := make(map[string]bool)
	for _, raw := range strings.Split(cfg.ConfigValue, ",") {
		ext := strings.TrimSpace(strings.ToLower(raw))
		if ext != "" {
			if !strings.HasPrefix(ext, ".") {
				ext = "." + ext
			}
			exts[ext] = true
		}
	}
	if len(exts) == 0 {
		return nil
	}
	return exts
}

func loadExcludePatterns(cfgSvc *configSvc.Service) []*regexp.Regexp {
	raw := getScanExcludeRaw(cfgSvc)
	if raw == "" {
		return nil
	}
	var patterns []string
	if err := json.Unmarshal([]byte(raw), &patterns); err != nil {
		return nil
	}
	var res []*regexp.Regexp
	for _, p := range patterns {
		r, err := regexp.Compile(p)
		if err != nil {
			continue
		}
		res = append(res, r)
	}
	return res
}

func getScanExcludeRaw(cfgSvc *configSvc.Service) string {
	cfg, err := cfgSvc.Get("scan_exclude")
	if err != nil || cfg == nil {
		return ""
	}
	return cfg.ConfigValue
}

type treeCacheData struct {
	Patterns string          `json:"patterns"`
	Tree     []*FileTreeNode `json:"tree"`
}

const treeCacheFile = ".tagfile/webui_tree.json"

func loadTreeCache(wantPatterns string) ([]*FileTreeNode, error) {
	data, err := os.ReadFile(treeCacheFile)
	if err != nil {
		return nil, err
	}
	var cached treeCacheData
	if err := json.Unmarshal(data, &cached); err != nil {
		return nil, err
	}
	if cached.Patterns != wantPatterns {
		return nil, fmt.Errorf("cache stale: patterns changed")
	}
	return cached.Tree, nil
}

func saveTreeCache(tree []*FileTreeNode, patterns string) {
	os.MkdirAll(".tagfile", 0755)
	cached := treeCacheData{Patterns: patterns, Tree: tree}
	data, _ := json.Marshal(cached)
	os.WriteFile(treeCacheFile, data, 0644)
}

func (s *Server) renderTagTable(w http.ResponseWriter, filePath string, fileTags []model.TagGroupTag, pg PageData, tagDisplayPaths map[uint]string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if len(fileTags) == 0 {
		fmt.Fprint(w, `<div class="empty"><div class="empty-icon">&#127991;</div><div class="empty-text">暂无标签</div></div>`)
		return
	}
	fmt.Fprint(w, `<table><thead><tr><th>标签名</th><th>值</th><th>分类</th><th style="width:140px">操作</th></tr></thead><tbody>`)
	for _, ft := range fileTags {
		val := ft.TagValue
		if val == "" {
			val = "-"
		}
		tagPath := tagDisplayPaths[ft.ID]
		if tagPath == "" {
			tagPath = ft.Tag.TagName
		}
		fmt.Fprintf(w, `<tr id="ft-%d"><td data-tag-name="%s" data-tag-id="%d">%s</td><td class="ft-value">%s</td><td>%s</td><td>`,
			ft.ID, htmlEscape(ft.Tag.TagName), ft.Tag.TagID, htmlEscape(tagPath), htmlEscape(val), htmlEscape(ft.Tag.Category.CategoryName))
		fmt.Fprintf(w, `<button class="btn-default btn-sm" onclick="editFileTagValue(%d,'%s')">编辑值</button> `, ft.ID, htmlEscape(ft.TagValue))
		fmt.Fprintf(w, `<button class="btn-danger btn-sm" onclick="removeFileTag(%d,%d,'%s')">删除</button>`,
			ft.ID, ft.Tag.TagID, htmlEscape(ft.TagValue))
		fmt.Fprint(w, `</td></tr>`)
	}
	fmt.Fprint(w, `</tbody></table>`)

	// pagination
	if pg.TotalPages > 1 {
		paginationTmpl := s.pages["file_detail"].Lookup("pagination")
		if paginationTmpl != nil {
			paginationTmpl.Execute(w, map[string]any{
				"PageData": pg,
				"PageFn":   template.JS("loadPage"),
			})
		}
	}
}

func htmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&#39;")
	return s
}
