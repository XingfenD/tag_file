package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/XingfenD/tag_file/internal/model"
	configSvc "github.com/XingfenD/tag_file/internal/service/config"
)

type FileTreeNode struct {
	Name     string          `json:"name"`
	Path     string          `json:"path"`
	IsDir    bool            `json:"isDir"`
	Indexed  bool            `json:"indexed"`
	File     *model.File     `json:"-"`
	FileID   uint            `json:"fileId,omitempty"`
	FileSize int64           `json:"fileSize"`
	TagCnt   int             `json:"tagCnt"`
	Children []*FileTreeNode `json:"children"`
}

const treeCacheFile = ".tagfile/webui_tree.json"

type treeCacheData struct {
	Patterns string          `json:"patterns"`
	Tree     []*FileTreeNode `json:"tree"`
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
				n.FileID = f.FileID
				n.TagCnt = int(f.TagCnt)
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

func iconForType(mimeType string) string {
	switch {
	case strings.HasPrefix(mimeType, "video/"):
		return "🎬"
	case strings.HasPrefix(mimeType, "image/"):
		return "🖼"
	case strings.HasPrefix(mimeType, "audio/"):
		return "🎵"
	case strings.Contains(mimeType, "pdf"):
		return "📄"
	case strings.Contains(mimeType, "zip") || strings.Contains(mimeType, "tar") || strings.Contains(mimeType, "rar") || strings.Contains(mimeType, "gzip"):
		return "📦"
	default:
		return "📄"
	}
}

func isMediaType(fileType string) string {
	switch {
	case strings.HasPrefix(fileType, "image/"):
		return "image"
	case strings.HasPrefix(fileType, "video/"):
		return "video"
	case strings.HasPrefix(fileType, "audio/"):
		return "audio"
	default:
		return ""
	}
}

func isPlayableVideo(fileType string) bool {
	switch fileType {
	case "video/mp4", "video/webm", "video/ogg", "video/ogv":
		return true
	default:
		return false
	}
}
