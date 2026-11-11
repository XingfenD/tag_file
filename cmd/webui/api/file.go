package api

import (
	"fmt"
	"io"
	"mime"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/XingfenD/tag_file/internal/model"
	fileSvc "github.com/XingfenD/tag_file/internal/service/file"
	"github.com/gin-gonic/gin"
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

// ---------------------------------------------------------------------------
// File handlers
// ---------------------------------------------------------------------------

// HandleFileTree responds to GET /api/files.
// Builds the file tree from the database (no filesystem scan).
func (rt *Router) HandleFileTree(c *gin.Context) {
	treeNodes, err := rt.FileSvc.BuildTreeFromDB()
	if err != nil {
		respond(c, "构建文件树失败: "+err.Error(), "error")
		return
	}

	tree := convertTreeNodes(treeNodes)
	for _, n := range tree {
		computeDirTagCnt(n)
		computeDirCounts(n)
	}

	respond(c, "ok", "success", map[string]any{"tree": tree})
}

func convertTreeNodes(nodes []*fileSvc.TreeNode) []*FileTreeNode {
	if nodes == nil {
		return nil
	}
	result := make([]*FileTreeNode, len(nodes))
	for i, n := range nodes {
		result[i] = &FileTreeNode{
			Name:     n.Name,
			Path:     n.Path,
			IsDir:    n.IsDir,
			Indexed:  !n.IsDir,
			TagCnt:   n.TagCnt,
			TagDone:  n.TagDone,
			FileSize: n.FileSize,
			Children: convertTreeNodes(n.Children),
		}
	}
	return result
}

// HandleExpandDir responds to GET /api/files/expand?dir=xxx.
// Returns indexed and unindexed files in a directory (single-level read).
func (rt *Router) HandleExpandDir(c *gin.Context) {
	dir := c.Query("dir")
	if dir == "" {
		dir = "."
	}

	contents, err := rt.FileSvc.ExpandDir(dir)
	if err != nil {
		respond(c, err.Error(), "error")
		return
	}

	var indexed []FileItem
	for _, f := range contents.Indexed {
		indexed = append(indexed, FileItem{
			ID:      f.FileID,
			Path:    f.FilePath,
			Name:    f.FileName,
			Size:    f.FileSize,
			Type:    f.FileType,
			TagCnt:  f.TagCnt,
			TagDone: f.TagDone,
		})
	}

	var unindexed []DirEntry
	for _, u := range contents.Unindexed {
		unindexed = append(unindexed, DirEntry{
			Name: u.Name,
			Path: u.Path,
			Size: u.Size,
			Type: u.Type,
		})
	}

	respond(c, "ok", "success", map[string]any{
		"indexed":   indexed,
		"unindexed": unindexed,
	})
}

// HandleFileDetail responds to GET /api/files/detail/*path.
// Returns full file detail including groups, tags, and directory listing.
func (rt *Router) HandleFileDetail(c *gin.Context) {
	filePath, err := url.PathUnescape(c.Param("path"))
	if err != nil {
		filePath = c.Param("path")
	}
	if filePath == "" {
		respond(c, "无效的文件路径", "error")
		return
	}

	file, _, _, err := rt.FileSvc.GetFileInfo(filePath)
	if err != nil {
		respond(c, err.Error(), "error")
		return
	}

	// Groups (single query with tag counts)
	groupsWithCounts, _ := rt.FileSvc.ListGroupsWithCounts(filePath)
	var selectedGroupID uint
	if gid, parseErr := strconv.ParseUint(c.Query("group_id"), 10, 64); parseErr == nil && gid > 0 {
		selectedGroupID = uint(gid)
	}
	if selectedGroupID == 0 && len(groupsWithCounts) > 0 {
		selectedGroupID = groupsWithCounts[0].Group.GroupID
	}

	page, _ := strconv.Atoi(c.Query("page"))
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

	fileTagIDs := make([]uint, 0, len(fileTags))
	for _, ft := range fileTags {
		fileTagIDs = append(fileTagIDs, ft.Tag.TagID)
	}
	idToTag := rt.loadTagPathMap(fileTagIDs)

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
	sortDirFiles(dirFiles, c.Query("sort"), c.Query("order"))

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

	respond(c, "ok", "success", map[string]any{"detail": detail})
}

// HandleFileGroups responds to GET /api/files/groups/*path.
func (rt *Router) HandleFileGroups(c *gin.Context) {
	filePath, err := url.PathUnescape(c.Param("path"))
	if err != nil {
		filePath = c.Param("path")
	}
	if filePath == "" {
		respond(c, "无效的文件路径", "error")
		return
	}

	groupsWithCounts, err := rt.FileSvc.ListGroupsWithCounts(filePath)
	if err != nil {
		respond(c, err.Error(), "error")
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

	respond(c, "ok", "success", map[string]any{"groups": groupItems})
}

// HandleFileRaw responds to GET /api/files/raw/:id.
// Serves the file content.
func (rt *Router) HandleFileRaw(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		respond(c, "无效的文件ID", "error")
		return
	}

	f, err := rt.FileSvc.FindByID(uint(id))
	if err != nil || f == nil {
		respond(c, "文件不存在", "error")
		return
	}
	filePath := f.FilePath

	c.File(filePath)
}

// HandleScanFiles responds to POST /api/files/scan.
// Scans the given directory for new files and removes the tree cache.
func (rt *Router) HandleScanFiles(c *gin.Context) {
	dir := c.PostForm("dir")
	if dir == "" {
		dir = "."
	}
	result, err := rt.FileSvc.ScanDirectoryOptions(dir, true)
	if err != nil {
		respond(c, err.Error(), "error")
		return
	}
	msg := fmt.Sprintf("扫描完成: 发现 %d 个文件, 新增 %d 个", result.FilesFound, result.FilesAdded)
	respond(c, msg, "success")
}

// HandleRefreshTree responds to POST /api/files/refresh.
func (rt *Router) HandleRefreshTree(c *gin.Context) {
	respond(c, "目录缓存已刷新", "success")
}

// HandleImportFile responds to POST /api/files/import.
// Accepts a file upload or a path string.
func (rt *Router) HandleImportFile(c *gin.Context) {
	upload, header, upErr := c.Request.FormFile("file")
	if upErr == nil {
		defer upload.Close()
		dst, err := os.Create(header.Filename)
		if err != nil {
			respond(c, "创建文件失败: "+err.Error(), "error")
			return
		}
		defer dst.Close()
		if _, err := io.Copy(dst, upload); err != nil {
			os.Remove(header.Filename)
			respond(c, "写入文件失败: "+err.Error(), "error")
			return
		}
		file, err := rt.FileSvc.ImportFile(header.Filename)
		if err != nil {
			respond(c, err.Error(), "error")
			return
		}
		respond(c, "文件 «"+file.FileName+"» 已导入", "success")
		return
	}

	path := c.PostForm("path")
	if path == "" {
		respond(c, "请选择文件或输入路径", "error")
		return
	}
	file, err := rt.FileSvc.ImportFile(path)
	if err != nil {
		respond(c, err.Error(), "error")
		return
	}
	respond(c, "文件 «"+file.FileName+"» 已导入", "success")
}

// HandleTagFile responds to POST /api/files/tag.
// Tags a file. If group_id is provided, tags to that group instead of default.
func (rt *Router) HandleTagFile(c *gin.Context) {
	filePath := c.PostForm("file_path")
	tagIDStr := c.PostForm("tag_id")
	tagValue := c.PostForm("tag_value")
	groupIDStr := c.PostForm("group_id")

	if filePath == "" {
		respond(c, "无效的文件路径", "error")
		return
	}

	tagID, parseErr := strconv.ParseUint(tagIDStr, 10, 64)
	if parseErr != nil || tagID == 0 {
		respond(c, "无效的标签ID", "error")
		return
	}

	var err error
	if groupIDStr != "" {
		gid, gpErr := strconv.ParseUint(groupIDStr, 10, 64)
		if gpErr != nil || gid == 0 {
			respond(c, "无效的标签组ID", "error")
			return
		}
		err = rt.FileSvc.TagFileToGroup(filePath, uint(tagID), tagValue, uint(gid))
	} else {
		err = rt.FileSvc.TagFile(filePath, uint(tagID), tagValue)
	}
	if err != nil {
		respond(c, err.Error(), "error")
		return
	}
	respond(c, "标签已添加", "success")
}

// HandleUntagFile responds to DELETE /api/files/tag.
// Removes a tag from a file. If group_id is provided, removes from that group.
func (rt *Router) HandleUntagFile(c *gin.Context) {
	filePath := c.Query("file_path")
	tagIDStr := c.Query("tag_id")
	tagValue := c.Query("tag_value")
	groupIDStr := c.Query("group_id")

	if filePath == "" {
		respond(c, "无效的文件路径", "error")
		return
	}

	tagID, parseErr := strconv.ParseUint(tagIDStr, 10, 64)
	if parseErr != nil || tagID == 0 {
		respond(c, "无效的标签ID", "error")
		return
	}

	var err error
	if groupIDStr != "" {
		gid, gpErr := strconv.ParseUint(groupIDStr, 10, 64)
		if gpErr != nil || gid == 0 {
			respond(c, "无效的标签组ID", "error")
			return
		}
		err = rt.FileSvc.UntagFileFromGroup(filePath, uint(tagID), tagValue, uint(gid))
	} else {
		err = rt.FileSvc.UntagFile(filePath, uint(tagID), tagValue)
	}
	if err != nil {
		respond(c, err.Error(), "error")
		return
	}
	respond(c, "标签已移除", "success")
}

// HandleUpdateTagValue responds to PUT /api/files/tag.
// Updates the value of a file-tag association by its ID.
func (rt *Router) HandleUpdateTagValue(c *gin.Context) {
	idStr := c.PostForm("id")
	ftID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		respond(c, "无效的ID", "error")
		return
	}
	val := c.PostForm("value")
	if err := rt.FileSvc.UpdateTagValue(uint(ftID), val); err != nil {
		respond(c, err.Error(), "error")
		return
	}
	respond(c, "ok", "success")
}

// HandleAddGroup responds to POST /api/files/group.
// Creates a new tag group for a file.
func (rt *Router) HandleAddGroup(c *gin.Context) {
	filePath := c.PostForm("file_path")
	groupName := c.PostForm("group_name")
	if filePath == "" || groupName == "" {
		respond(c, "参数不完整", "error")
		return
	}
	group, err := rt.FileSvc.CreateGroup(filePath, groupName)
	if err != nil {
		respond(c, err.Error(), "error")
		return
	}
	respond(c, "标签组 «"+groupName+"» 已创建", "success", map[string]any{
		"group": map[string]any{
			"id":         group.GroupID,
			"name":       group.GroupName,
			"is_default": false,
		},
	})
}

// HandleRemoveGroup responds to DELETE /api/files/group/:id.
// Deletes a tag group. File path comes from the URL query.
func (rt *Router) HandleRemoveGroup(c *gin.Context) {
	filePath := c.Query("file_path")
	groupIDStr := c.Param("id")
	groupID, err := strconv.ParseUint(groupIDStr, 10, 64)
	if err != nil || groupID == 0 {
		respond(c, "无效的标签组ID", "error")
		return
	}
	if err := rt.FileSvc.DeleteGroup(filePath, uint(groupID)); err != nil {
		respond(c, err.Error(), "error")
		return
	}
	respond(c, "标签组已删除", "success")
}

// HandleRenameGroup responds to PUT /api/files/group/:id.
// Renames a tag group. File path and new name come from form data.
func (rt *Router) HandleRenameGroup(c *gin.Context) {
	filePath := c.PostForm("file_path")
	groupIDStr := c.Param("id")
	newName := c.PostForm("group_name")
	groupID, err := strconv.ParseUint(groupIDStr, 10, 64)
	if err != nil || groupID == 0 || newName == "" {
		respond(c, "参数不完整", "error")
		return
	}
	if err := rt.FileSvc.RenameGroup(filePath, uint(groupID), newName); err != nil {
		respond(c, err.Error(), "error")
		return
	}
	respond(c, "标签组已重命名", "success", map[string]any{
		"group_id": groupID, "group_name": newName,
	})
}

// HandleFileDone responds to POST /api/files/done.
// Marks a file as tagging-done (or unmarks it when done=false).
func (rt *Router) HandleFileDone(c *gin.Context) {
	filePath := c.PostForm("file_path")
	if filePath == "" {
		respond(c, "无效的文件路径", "error")
		return
	}
	done := c.PostForm("done") != "false"
	if _, err := rt.FileSvc.SetFileTagDone(filePath, done); err != nil {
		respond(c, err.Error(), "error")
		return
	}
	if done {
		respond(c, "已标记为打标完成", "success")
	} else {
		respond(c, "已取消打标完成标记", "success")
	}
}

// HandleRenameFile responds to PUT /api/files/rename.
// Renames a file on disk and updates the index.
func (rt *Router) HandleRenameFile(c *gin.Context) {
	filePath := c.PostForm("file_path")
	newName := c.PostForm("new_name")
	if filePath == "" || newName == "" {
		respond(c, "参数不完整", "error")
		return
	}
	_, err := rt.FileSvc.RenameFile(filePath, newName)
	if err != nil {
		respond(c, err.Error(), "error")
		return
	}
	respond(c, "文件已重命名为 «"+newName+"»", "success")
}
