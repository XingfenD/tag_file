package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"os"
	"path/filepath"
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
	Children []*FileTreeNode
}

func buildFileTreeFromFS(rootDir string, excludeExts map[string]bool) ([]*FileTreeNode, error) {
	root, err := filepath.Abs(rootDir)
	if err != nil {
		return nil, err
	}

	treeRoot := &FileTreeNode{Name: root, IsDir: true}

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
		if strings.HasPrefix(name, ".") && d.IsDir() {
			return filepath.SkipDir
		}
		if strings.HasPrefix(name, ".") && !d.IsDir() {
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
	flash, flashType := GetFlash(r)
	excludeExts := getHideExtensions(s.ConfigSvc)

	var tree []*FileTreeNode
	cached, cacheErr := loadTreeCache()
	if cacheErr == nil {
		tree = cached
	} else {
		var err error
		tree, err = buildFileTreeFromFS(".", excludeExts)
		if err != nil {
			data := map[string]any{
				"Active":    "files",
				"Flash":     "遍历文件系统失败: " + err.Error(),
				"FlashType": "error",
			}
			s.Render(w, "file", data)
			return
		}
		saveTreeCache(tree)
	}

	indexed, _, _ := s.FileSvc.ListFiles(0, 0)
	indexedMap := make(map[string]*model.File, len(indexed))
	for i := range indexed {
		indexedMap[indexed[i].FilePath] = &indexed[i]
	}
	markTreeIndexed(tree, indexedMap)
	data := map[string]any{
		"Active":    "files",
		"Tree":      tree,
		"Flash":     flash,
		"FlashType": flashType,
	}
	s.Render(w, "file", data)
}

func (s *Server) HandleRefreshTree(w http.ResponseWriter, r *http.Request) {
	os.Remove(treeCacheFile)
	s.Redirect(w, r, "/files", "目录缓存已刷新", "success")
}

func (s *Server) HandleScanFiles(w http.ResponseWriter, r *http.Request) {
	dir := r.FormValue("dir")
	if dir == "" {
		dir = "."
	}
	result, err := s.FileSvc.ScanDirectoryOptions(dir, true)
	if err != nil {
		s.Redirect(w, r, "/files", err.Error(), "error")
		return
	}
	os.Remove(treeCacheFile)
	msg := fmt.Sprintf("扫描完成: 发现 %d 个文件, 新增 %d 个", result.FilesFound, result.FilesAdded)
	s.Redirect(w, r, "/files", msg, "success")
}

func (s *Server) HandleImportFile(w http.ResponseWriter, r *http.Request) {
	upload, header, upErr := r.FormFile("file")
	if upErr == nil {
		defer upload.Close()
		dst, err := os.Create(header.Filename)
		if err != nil {
			s.Redirect(w, r, "/files", "创建文件失败: "+err.Error(), "error")
			return
		}
		defer dst.Close()
		if _, err := io.Copy(dst, upload); err != nil {
			os.Remove(header.Filename)
			s.Redirect(w, r, "/files", "写入文件失败: "+err.Error(), "error")
			return
		}
		file, err := s.FileSvc.ImportFile(header.Filename)
		if err != nil {
			s.Redirect(w, r, "/files", err.Error(), "error")
			return
		}
		os.Remove(treeCacheFile)
		s.Redirect(w, r, "/files", "文件 «"+file.FileName+"» 已导入", "success")
		return
	}

	path := r.FormValue("path")
	if path == "" {
		s.Redirect(w, r, "/files", "请选择文件或输入路径", "error")
		return
	}
	file, err := s.FileSvc.ImportFile(path)
	if err != nil {
		s.Redirect(w, r, "/files", err.Error(), "error")
		return
	}
	os.Remove(treeCacheFile)
	s.Redirect(w, r, "/files", "文件 «"+file.FileName+"» 已导入", "success")
}

func (s *Server) HandleTagFile(w http.ResponseWriter, r *http.Request) {
	filePath := r.FormValue("file_path")
	tagName := r.FormValue("tag_name")

	if filePath == "" {
		s.Redirect(w, r, "/files", "无效的文件路径", "error")
		return
	}

	if err := s.FileSvc.TagFile(filePath, tagName); err != nil {
		s.Redirect(w, r, "/files/detail/"+filePath, err.Error(), "error")
		return
	}
	s.Redirect(w, r, "/files/detail/"+filePath, "标签 «"+tagName+"» 已添加", "success")
}

func (s *Server) HandleUntagFile(w http.ResponseWriter, r *http.Request) {
	filePath := r.FormValue("file_path")
	tagName := r.FormValue("tag_name")

	if filePath == "" {
		s.Redirect(w, r, "/files", "无效的文件路径", "error")
		return
	}

	if err := s.FileSvc.UntagFile(filePath, tagName); err != nil {
		s.Redirect(w, r, "/files/detail/"+filePath, err.Error(), "error")
		return
	}
	s.Redirect(w, r, "/files/detail/"+filePath, "标签 «"+tagName+"» 已移除", "success")
}

func (s *Server) HandleRenameFile(w http.ResponseWriter, r *http.Request) {
	filePath := r.FormValue("file_path")
	newName := r.FormValue("new_name")
	if filePath == "" || newName == "" {
		s.Redirect(w, r, "/files", "参数不完整", "error")
		return
	}
	newFile, err := s.FileSvc.RenameFile(filePath, newName)
	if err != nil {
		s.Redirect(w, r, "/files/detail/"+filePath, err.Error(), "error")
		return
	}
	os.Remove(treeCacheFile)
	s.Redirect(w, r, "/files/detail/"+newFile.FilePath, "文件已重命名为 «"+newName+"»", "success")
}

func (s *Server) HandleFileInfo(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, err := ParseInt(idStr)
	if err != nil {
		s.Redirect(w, r, "/files", "无效的文件ID", "error")
		return
	}

	file, tags, err := s.fileInfoByID(id)
	if err != nil {
		s.Redirect(w, r, "/files", err.Error(), "error")
		return
	}

	files, _, _ := s.FileSvc.ListFiles(0, 0)
	mediaType := IsMediaType(file.FileType)
	flash, flashType := GetFlash(r)
	data := map[string]any{
		"Active":    "files",
		"Files":     files,
		"FileInfo":  map[string]any{"File": file, "Tags": tags, "MediaType": mediaType},
		"Flash":     flash,
		"FlashType": flashType,
	}
	s.Render(w, "file", data)
}

func (s *Server) HandleFileDetail(w http.ResponseWriter, r *http.Request) {
	filePath := r.PathValue("path")
	if filePath == "" {
		s.Redirect(w, r, "/files", "无效的文件路径", "error")
		return
	}

	file, tags, err := s.FileSvc.GetFileInfo(filePath)
	if err != nil {
		s.Redirect(w, r, "/files", err.Error(), "error")
		return
	}

	allTags, _, _ := s.TagSvc.ListTags("", 0, 0)
	flash, flashType := GetFlash(r)
	mediaType := IsMediaType(file.FileType)
	data := map[string]any{
		"Active":   "files",
		"FileInfo": map[string]any{"File": file, "Tags": tags, "MediaType": mediaType},
		"AllTags":  allTags,
		"Flash":    flash,
		"FlashType": flashType,
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

	data, err := os.ReadFile(filePath)
	if err != nil {
		http.Error(w, "read error: "+err.Error(), http.StatusNotFound)
		return
	}

	ext := filepath.Ext(filePath)
	contentType := mime.TypeByExtension(ext)
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "private, max-age=3600")
	w.Write(data)
}

func (s *Server) fileInfoByID(id uint) (file *model.File, tags []model.Tag, err error) {
	files, _, _ := s.FileSvc.ListFiles(0, 0)
	var filePath string
	for _, f := range files {
		if f.FileID == id {
			filePath = f.FilePath
			break
		}
	}
	if filePath == "" {
		return nil, nil, fmt.Errorf("文件不存在")
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

const treeCacheFile = ".tagfile/webui_tree.json"

func loadTreeCache() ([]*FileTreeNode, error) {
	data, err := os.ReadFile(treeCacheFile)
	if err != nil {
		return nil, err
	}
	var cached FileTreeNode
	if err := json.Unmarshal(data, &cached); err != nil {
		return nil, err
	}
	return cached.Children, nil
}

func saveTreeCache(tree []*FileTreeNode) {
	os.MkdirAll(".tagfile", 0755)
	cached := FileTreeNode{Name: ".", IsDir: true, Children: tree}
	data, _ := json.Marshal(cached)
	os.WriteFile(treeCacheFile, data, 0644)
}
