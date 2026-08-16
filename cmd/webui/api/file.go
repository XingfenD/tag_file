package api

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/XingfenD/tag_file/internal/model"
	configSvc "github.com/XingfenD/tag_file/internal/service/config"
)

// FileItem is the JSON representation of an indexed file.
type FileItem struct {
	ID      uint   `json:"id"`
	Path    string `json:"path"`
	Name    string `json:"name"`
	Size    int64  `json:"size"`
	Type    string `json:"type"`
	TagCnt  int    `json:"tag_cnt"`
	TagDone bool   `json:"tag_done"`
}

// FileTreeNode represents a node in the filesystem tree.
type FileTreeNode struct {
	Name      string          `json:"name"`
	Path      string          `json:"path"`
	IsDir     bool            `json:"is_dir"`
	Indexed   bool            `json:"indexed"`
	TagCnt    int             `json:"tag_cnt"`
	TagDone   bool            `json:"tag_done"`
	FileSize  int64           `json:"file_size,omitempty"`
	FileCount int             `json:"file_count,omitempty"`
	DoneCount int             `json:"done_count,omitempty"`
	Children  []*FileTreeNode `json:"children,omitempty"`
}

// GroupItem is the JSON representation of a tag group.
type GroupItem struct {
	ID        uint   `json:"id"`
	Name      string `json:"name"`
	IsDefault bool   `json:"is_default"`
	TagCount  int64  `json:"tag_count"`
}

// FileTagItem is the JSON representation of a tag applied to a file.
type FileTagItem struct {
	ID       uint   `json:"id"`
	TagID    uint   `json:"tag_id"`
	TagName  string `json:"tag_name"`
	TagPath  string `json:"tag_path"`
	Value    string `json:"value"`
	Category string `json:"category"`
}

// DirEntry represents a file entry in a directory listing.
type DirEntry struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Size int64  `json:"size"`
	Type string `json:"type"`
}

// FileDetailData is the full detail response for a single file.
type FileDetailData struct {
	File          FileItem      `json:"file"`
	MediaType     string        `json:"media_type"`
	VideoPlayable bool          `json:"video_playable"`
	Groups        []GroupItem   `json:"groups"`
	Tags          []FileTagItem `json:"tags"`
	Pagination    Pagination    `json:"pagination,omitempty"`
	DirFiles      []FileItem    `json:"dir_files"`
	DirUnindexed  []DirEntry    `json:"dir_unindexed"`
	DirPath       string        `json:"dir_path"`
}

// ---------------------------------------------------------------------------
// Tree cache
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// Config helpers
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// Media type helpers
// ---------------------------------------------------------------------------

func isMediaType(fileType string) string {
	if strings.HasPrefix(fileType, "image/") {
		return "image"
	}
	if strings.HasPrefix(fileType, "video/") {
		return "video"
	}
	if strings.HasPrefix(fileType, "audio/") {
		return "audio"
	}
	return ""
}

var playableVideoTypes = map[string]bool{
	"video/mp4":  true,
	"video/webm": true,
	"video/ogg":  true,
	"video/ogv":  true,
}

func isPlayableVideo(fileType string) bool {
	return playableVideoTypes[fileType]
}

// ---------------------------------------------------------------------------
// File tree building
// ---------------------------------------------------------------------------

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
		} else if f, ok := indexed[n.Path]; ok {
			n.Indexed = true
			n.TagCnt = f.TagCnt
			n.TagDone = f.TagDone
		}
	}
}

func computeDirTagCnt(node *FileTreeNode) int {
	if !node.IsDir {
		return node.TagCnt
	}
	sum := 0
	for _, c := range node.Children {
		sum += computeDirTagCnt(c)
	}
	node.TagCnt = sum
	return sum
}

// computeDirCounts fills FileCount (all files in subtree, indexed or not)
// and DoneCount (files with TagDone) for every directory node.
func computeDirCounts(node *FileTreeNode) (files, done int) {
	if !node.IsDir {
		if node.TagDone {
			return 1, 1
		}
		return 1, 0
	}
	for _, c := range node.Children {
		f, d := computeDirCounts(c)
		files += f
		done += d
	}
	node.FileCount = files
	node.DoneCount = done
	return files, done
}

func sortDirFiles(files []FileItem, sortBy, order string) {
	if len(files) <= 1 {
		return
	}
	desc := order == "desc"
	sort.Slice(files, func(i, j int) bool {
		switch sortBy {
		case "size":
			if desc {
				return files[i].Size > files[j].Size
			}
			return files[i].Size < files[j].Size
		case "type":
			if desc {
				return files[i].Type > files[j].Type
			}
			return files[i].Type < files[j].Type
		case "tagcnt":
			if desc {
				return files[i].TagCnt > files[j].TagCnt
			}
			return files[i].TagCnt < files[j].TagCnt
		default:
			if desc {
				return files[i].Name > files[j].Name
			}
			return files[i].Name < files[j].Name
		}
	})
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

// ---------------------------------------------------------------------------
// File handlers
// ---------------------------------------------------------------------------

// HandleFileTree responds to GET /api/files.
// Builds the filesystem tree, marks indexed files, and returns JSON.
func (rt *Router) HandleFileTree(w http.ResponseWriter, r *http.Request) {
	excludeExts := getHideExtensions(rt.ConfigSvc)
	excludePatterns := loadExcludePatterns(rt.ConfigSvc)
	patternsStr := getScanExcludeRaw(rt.ConfigSvc)

	var tree []*FileTreeNode
	cached, cacheErr := loadTreeCache(patternsStr)
	if cacheErr == nil {
		tree = cached
	} else {
		var err error
		tree, err = buildFileTreeFromFS(".", excludeExts, excludePatterns)
		if err != nil {
			respond(w, "遍历文件系统失败: "+err.Error(), "error")
			return
		}
		saveTreeCache(tree, patternsStr)
	}

	indexed, _, _ := rt.FileSvc.ListFiles(0, 0)
	indexedMap := make(map[string]*model.File, len(indexed))
	for i := range indexed {
		indexedMap[indexed[i].FilePath] = &indexed[i]
	}
	markTreeIndexed(tree, indexedMap)
	for _, n := range tree {
		computeDirTagCnt(n)
		computeDirCounts(n)
	}

	respond(w, "ok", "success", map[string]any{"tree": tree})
}

// HandleFileDetail responds to GET /api/files/detail/{path...}.
// Returns full file detail including groups, tags, and directory listing.
func (rt *Router) HandleFileDetail(w http.ResponseWriter, r *http.Request) {
	filePath, err := url.PathUnescape(r.PathValue("path"))
	if err != nil {
		filePath = r.PathValue("path")
	}
	if filePath == "" {
		respond(w, "无效的文件路径", "error")
		return
	}

	file, _, _, err := rt.FileSvc.GetFileInfo(filePath)
	if err != nil {
		respond(w, err.Error(), "error")
		return
	}

	// Groups (single query with tag counts)
	groupsWithCounts, _ := rt.FileSvc.ListGroupsWithCounts(filePath)
	var selectedGroupID uint
	if gid, parseErr := strconv.ParseUint(r.URL.Query().Get("group_id"), 10, 64); parseErr == nil && gid > 0 {
		selectedGroupID = uint(gid)
	}
	if selectedGroupID == 0 && len(groupsWithCounts) > 0 {
		selectedGroupID = groupsWithCounts[0].Group.GroupID
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize := 10

	// File tags for selected group
	var fileTags []model.TagGroupTag
	var ftTotal int64
	if selectedGroupID > 0 {
		fileTags, ftTotal, _ = rt.FileSvc.ListFileTagsByGroup(filePath, selectedGroupID, (page-1)*pageSize, pageSize)
	}
	pg := computePagination(ftTotal, page, pageSize)

	// Build tag paths
	allTags, _, _ := rt.TagSvc.ListTags("", 0, 0)
	idToTag := make(map[uint]*model.Tag, len(allTags))
	for i := range allTags {
		idToTag[allTags[i].TagID] = &allTags[i]
	}

	tagItems := make([]FileTagItem, 0, len(fileTags))
	for _, ft := range fileTags {
		tagPath := buildTagPath(ft.Tag.TagID, idToTag, 0)
		tagItems = append(tagItems, FileTagItem{
			ID:       ft.ID,
			TagID:    ft.Tag.TagID,
			TagName:  ft.Tag.TagName,
			TagPath:  tagPath,
			Value:    ft.TagValue,
			Category: ft.Tag.Category.CategoryName,
		})
	}

	// Group items with tag counts
	groupItems := make([]GroupItem, 0, len(groupsWithCounts))
	for _, gc := range groupsWithCounts {
		groupItems = append(groupItems, GroupItem{
			ID:        gc.Group.GroupID,
			Name:      gc.Group.GroupName,
			IsDefault: gc.Group.IsDefault,
			TagCount:  gc.TagCount,
		})
	}

	mediaType := isMediaType(file.FileType)
	videoPlayable := isPlayableVideo(file.FileType)

	// Indexed files in the same directory
	dir := filepath.Dir(filePath)
	dirFilesRaw, _ := rt.FileSvc.ListFilesByDir(dir)
	indexedSet := make(map[string]bool)
	var dirFiles []FileItem
	for _, f := range dirFilesRaw {
		indexedSet[f.FilePath] = true
		dirFiles = append(dirFiles, FileItem{
			ID:      f.FileID,
			Path:    f.FilePath,
			Name:    f.FileName,
			Size:    f.FileSize,
			Type:    f.FileType,
			TagCnt:  f.TagCnt,
			TagDone: f.TagDone,
		})
	}
	sortDirFiles(dirFiles, r.URL.Query().Get("sort"), r.URL.Query().Get("order"))

	// Unindexed files in the directory
	entries, readErr := os.ReadDir(dir)
	var dirUnindexed []DirEntry
	if readErr == nil {
		for _, e := range entries {
			if e.IsDir() || strings.HasPrefix(e.Name(), ".") {
				continue
			}
			p := filepath.Join(dir, e.Name())
			if indexedSet[p] {
				continue
			}
			info, _ := e.Info()
			var size int64
			if info != nil {
				size = info.Size()
			}
			dirUnindexed = append(dirUnindexed, DirEntry{
				Name: e.Name(),
				Path: p,
				Size: size,
				Type: mime.TypeByExtension(filepath.Ext(e.Name())),
			})
		}
	}
	sort.Slice(dirUnindexed, func(i, j int) bool {
		return dirUnindexed[i].Name < dirUnindexed[j].Name
	})

	detail := FileDetailData{
		File: FileItem{
			ID:      file.FileID,
			Path:    file.FilePath,
			Name:    file.FileName,
			Size:    file.FileSize,
			Type:    file.FileType,
			TagCnt:  file.TagCnt,
			TagDone: file.TagDone,
		},
		MediaType:     mediaType,
		VideoPlayable: videoPlayable,
		Groups:        groupItems,
		Tags:          tagItems,
		Pagination:    pg,
		DirFiles:      dirFiles,
		DirUnindexed:  dirUnindexed,
		DirPath:       dir,
	}

	respond(w, "ok", "success", map[string]any{"detail": detail})
}

// HandleFileGroups responds to GET /api/files/groups/{path...}.
func (rt *Router) HandleFileGroups(w http.ResponseWriter, r *http.Request) {
	filePath, err := url.PathUnescape(r.PathValue("path"))
	if err != nil {
		filePath = r.PathValue("path")
	}
	if filePath == "" {
		respond(w, "无效的文件路径", "error")
		return
	}

	groupsWithCounts, err := rt.FileSvc.ListGroupsWithCounts(filePath)
	if err != nil {
		respond(w, err.Error(), "error")
		return
	}

	groupItems := make([]GroupItem, 0, len(groupsWithCounts))
	for _, gc := range groupsWithCounts {
		groupItems = append(groupItems, GroupItem{
			ID:        gc.Group.GroupID,
			Name:      gc.Group.GroupName,
			IsDefault: gc.Group.IsDefault,
			TagCount:  gc.TagCount,
		})
	}

	respond(w, "ok", "success", map[string]any{"groups": groupItems})
}

// HandleFileRaw responds to GET /api/files/raw/{id}.
// Serves the file content using http.ServeFile.
func (rt *Router) HandleFileRaw(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		respond(w, "无效的文件ID", "error")
		return
	}

	f, err := rt.FileSvc.FindByID(uint(id))
	if err != nil || f == nil {
		respond(w, "文件不存在", "error")
		return
	}
	filePath := f.FilePath

	http.ServeFile(w, r, filePath)
}

// HandleScanFiles responds to POST /api/files/scan.
// Scans the given directory for new files and removes the tree cache.
func (rt *Router) HandleScanFiles(w http.ResponseWriter, r *http.Request) {
	dir := r.FormValue("dir")
	if dir == "" {
		dir = "."
	}
	result, err := rt.FileSvc.ScanDirectoryOptions(dir, true)
	if err != nil {
		respond(w, err.Error(), "error")
		return
	}
	os.Remove(treeCacheFile)
	msg := fmt.Sprintf("扫描完成: 发现 %d 个文件, 新增 %d 个", result.FilesFound, result.FilesAdded)
	respond(w, msg, "success")
}

// HandleRefreshTree responds to POST /api/files/refresh.
// Removes the tree cache file so the tree is rebuilt on next request.
func (rt *Router) HandleRefreshTree(w http.ResponseWriter, r *http.Request) {
	os.Remove(treeCacheFile)
	respond(w, "目录缓存已刷新", "success")
}

// HandleImportFile responds to POST /api/files/import.
// Accepts a file upload or a path string. Removes tree cache on success.
func (rt *Router) HandleImportFile(w http.ResponseWriter, r *http.Request) {
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
		file, err := rt.FileSvc.ImportFile(header.Filename)
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
	file, err := rt.FileSvc.ImportFile(path)
	if err != nil {
		respond(w, err.Error(), "error")
		return
	}
	os.Remove(treeCacheFile)
	respond(w, "文件 «"+file.FileName+"» 已导入", "success")
}

// HandleTagFile responds to POST /api/files/tag.
// Tags a file. If group_id is provided, tags to that group instead of default.
func (rt *Router) HandleTagFile(w http.ResponseWriter, r *http.Request) {
	filePath := r.FormValue("file_path")
	tagIDStr := r.FormValue("tag_id")
	tagValue := r.FormValue("tag_value")
	groupIDStr := r.FormValue("group_id")

	if filePath == "" {
		respond(w, "无效的文件路径", "error")
		return
	}

	tagID, parseErr := strconv.ParseUint(tagIDStr, 10, 64)
	if parseErr != nil || tagID == 0 {
		respond(w, "无效的标签ID", "error")
		return
	}

	var err error
	if groupIDStr != "" {
		if gid, gpErr := strconv.ParseUint(groupIDStr, 10, 64); gpErr == nil && gid > 0 {
			err = rt.FileSvc.TagFileToGroup(filePath, uint(tagID), tagValue, uint(gid))
		}
	}
	if groupIDStr == "" || err != nil {
		err = rt.FileSvc.TagFile(filePath, uint(tagID), tagValue)
	}
	if err != nil {
		respond(w, err.Error(), "error")
		return
	}
	respond(w, "标签已添加", "success")
}

// HandleUntagFile responds to DELETE /api/files/tag.
// Removes a tag from a file. If group_id is provided, removes from that group.
func (rt *Router) HandleUntagFile(w http.ResponseWriter, r *http.Request) {
	filePath := r.FormValue("file_path")
	tagIDStr := r.FormValue("tag_id")
	tagValue := r.FormValue("tag_value")
	groupIDStr := r.FormValue("group_id")

	if filePath == "" {
		respond(w, "无效的文件路径", "error")
		return
	}

	tagID, parseErr := strconv.ParseUint(tagIDStr, 10, 64)
	if parseErr != nil || tagID == 0 {
		respond(w, "无效的标签ID", "error")
		return
	}

	var err error
	if groupIDStr != "" {
		if gid, gpErr := strconv.ParseUint(groupIDStr, 10, 64); gpErr == nil && gid > 0 {
			err = rt.FileSvc.UntagFileFromGroup(filePath, uint(tagID), tagValue, uint(gid))
		}
	}
	if groupIDStr == "" || err != nil {
		err = rt.FileSvc.UntagFile(filePath, uint(tagID), tagValue)
	}
	if err != nil {
		respond(w, err.Error(), "error")
		return
	}
	respond(w, "标签已移除", "success")
}

// HandleUpdateTagValue responds to PUT /api/files/tag.
// Updates the value of a file-tag association by its ID.
func (rt *Router) HandleUpdateTagValue(w http.ResponseWriter, r *http.Request) {
	idStr := r.FormValue("id")
	ftID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		respond(w, "无效的ID", "error")
		return
	}
	val := r.FormValue("value")
	if err := rt.FileSvc.UpdateTagValue(uint(ftID), val); err != nil {
		respond(w, err.Error(), "error")
		return
	}
	respond(w, "ok", "success")
}

// HandleAddGroup responds to POST /api/files/group.
// Creates a new tag group for a file.
func (rt *Router) HandleAddGroup(w http.ResponseWriter, r *http.Request) {
	filePath := r.FormValue("file_path")
	groupName := r.FormValue("group_name")
	if filePath == "" || groupName == "" {
		respond(w, "参数不完整", "error")
		return
	}
	group, err := rt.FileSvc.CreateGroup(filePath, groupName)
	if err != nil {
		respond(w, err.Error(), "error")
		return
	}
	respond(w, "标签组 «"+groupName+"» 已创建", "success", map[string]any{
		"group": map[string]any{
			"id":         group.GroupID,
			"name":       group.GroupName,
			"is_default": false,
		},
	})
}

// HandleRemoveGroup responds to DELETE /api/files/group/{id}.
// Deletes a tag group. File path comes from form data.
func (rt *Router) HandleRemoveGroup(w http.ResponseWriter, r *http.Request) {
	filePath := r.FormValue("file_path")
	groupIDStr := r.PathValue("id")
	groupID, err := strconv.ParseUint(groupIDStr, 10, 64)
	if err != nil || groupID == 0 {
		respond(w, "无效的标签组ID", "error")
		return
	}
	if err := rt.FileSvc.DeleteGroup(filePath, uint(groupID)); err != nil {
		respond(w, err.Error(), "error")
		return
	}
	respond(w, "标签组已删除", "success")
}

// HandleRenameGroup responds to PUT /api/files/group/{id}.
// Renames a tag group. File path and new name come from form data.
func (rt *Router) HandleRenameGroup(w http.ResponseWriter, r *http.Request) {
	filePath := r.FormValue("file_path")
	groupIDStr := r.PathValue("id")
	newName := r.FormValue("group_name")
	groupID, err := strconv.ParseUint(groupIDStr, 10, 64)
	if err != nil || groupID == 0 || newName == "" {
		respond(w, "参数不完整", "error")
		return
	}
	if err := rt.FileSvc.RenameGroup(filePath, uint(groupID), newName); err != nil {
		respond(w, err.Error(), "error")
		return
	}
	respond(w, "标签组已重命名", "success", map[string]any{
		"group_id": groupID, "group_name": newName,
	})
}

// HandleFileDone responds to POST /api/files/done.
// Marks a file as tagging-done (or unmarks it when done=false).
func (rt *Router) HandleFileDone(w http.ResponseWriter, r *http.Request) {
	filePath := r.FormValue("file_path")
	if filePath == "" {
		respond(w, "无效的文件路径", "error")
		return
	}
	done := r.FormValue("done") != "false"
	if _, err := rt.FileSvc.SetFileTagDone(filePath, done); err != nil {
		respond(w, err.Error(), "error")
		return
	}
	if done {
		respond(w, "已标记为打标完成", "success")
	} else {
		respond(w, "已取消打标完成标记", "success")
	}
}

// HandleRenameFile responds to PUT /api/files/{path...}.
// Renames a file on disk and updates the index.
func (rt *Router) HandleRenameFile(w http.ResponseWriter, r *http.Request) {
	filePath, err := url.PathUnescape(r.PathValue("path"))
	if err != nil {
		filePath = r.PathValue("path")
	}
	newName := r.FormValue("new_name")
	if filePath == "" || newName == "" {
		respond(w, "参数不完整", "error")
		return
	}
	_, err = rt.FileSvc.RenameFile(filePath, newName)
	if err != nil {
		respond(w, err.Error(), "error")
		return
	}
	respond(w, "文件已重命名为 «"+newName+"»", "success")
}
