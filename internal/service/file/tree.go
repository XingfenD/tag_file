package file

import (
	"fmt"
	"log"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"github.com/XingfenD/tag_file/internal/model"
)

type TreeNode struct {
	Name      string
	Path      string
	IsDir     bool
	FileSize  int64
	TagCnt    int
	TagDone   bool
	Children  []*TreeNode
}

type DirContents struct {
	Indexed   []model.File
	Unindexed []UnindexedFile
}

type UnindexedFile struct {
	Name string
	Path string
	Size int64
	Type string
}

func (s *Service) BuildTreeFromDB() ([]*TreeNode, error) {
	files, _, err := s.ListFiles(0, 0)
	if err != nil {
		return nil, err
	}

	root := &TreeNode{Name: ".", IsDir: true}
	fileMap := make(map[string]*model.File, len(files))
	for i := range files {
		fileMap[files[i].FilePath] = &files[i]
	}

	for _, f := range files {
		parts := strings.Split(filepath.ToSlash(f.FilePath), "/")
		cur := root
		for i, part := range parts {
			isLast := i == len(parts)-1
			if isLast {
				cur.Children = append(cur.Children, &TreeNode{
					Name:     part,
					Path:     f.FilePath,
					IsDir:    false,
					FileSize: f.FileSize,
					TagCnt:   f.TagCnt,
					TagDone:  f.TagDone,
				})
			} else {
				dirPath := strings.Join(parts[:i+1], "/")
				found := false
				for _, c := range cur.Children {
					if c.IsDir && c.Name == part {
						cur = c
						found = true
						break
					}
				}
				if !found {
					dir := &TreeNode{Name: part, Path: dirPath, IsDir: true}
					cur.Children = append(cur.Children, dir)
					cur = dir
				}
			}
		}
	}

	pruneEmptyDirsTree(root)
	return root.Children, nil
}

func pruneEmptyDirsTree(node *TreeNode) {
	var kept []*TreeNode
	for _, c := range node.Children {
		if c.IsDir {
			pruneEmptyDirsTree(c)
			if len(c.Children) == 0 {
				continue
			}
		}
		kept = append(kept, c)
	}
	node.Children = kept
}

func (s *Service) ExpandDir(dir string) (*DirContents, error) {
	if dir == "" {
		dir = "."
	}

	indexed, err := s.ListFilesByDir(dir)
	if err != nil {
		log.Printf("[file] ExpandDir: list indexed: %v", err)
		return nil, fmt.Errorf("查询已索引文件失败: %w", err)
	}

	indexedSet := make(map[string]bool, len(indexed))
	for _, f := range indexed {
		indexedSet[f.FilePath] = true
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		log.Printf("[file] ExpandDir: read dir: %v", err)
		return nil, fmt.Errorf("读取目录失败: %w", err)
	}

	var unindexed []UnindexedFile
	for _, e := range entries {
		if e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		var p string
		if dir == "." || dir == "" {
			p = e.Name()
		} else {
			p = filepath.Join(dir, e.Name())
		}
		if indexedSet[p] {
			continue
		}
		info, _ := e.Info()
		var size int64
		if info != nil {
			size = info.Size()
		}
		unindexed = append(unindexed, UnindexedFile{
			Name: e.Name(),
			Path: p,
			Size: size,
			Type: mime.TypeByExtension(filepath.Ext(e.Name())),
		})
	}

	return &DirContents{
		Indexed:   indexed,
		Unindexed: unindexed,
	}, nil
}
