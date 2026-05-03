package handlers

import (
	"fmt"
	"mime"
	"net/http"
	"os"
	"path/filepath"

	"github.com/XingfenD/tag_file/internal/model"
)

func (s *Server) HandleListFiles(w http.ResponseWriter, r *http.Request) {
	pg := Paginate(r, 0, 10)
	files, total, err := s.FileSvc.ListFiles(pg.Start-1, pg.PageSize)
	if err != nil {
		s.Redirect(w, r, "/files", err.Error(), "error")
		return
	}
	UpdatePagination(&pg, total, len(files))
	flash, flashType := GetFlash(r)
	data := map[string]any{
		"Active":    "files",
		"Files":     files,
		"PageData":  pg,
		"BaseURL":   PaginationBaseURL(r),
		"Flash":     flash,
		"FlashType": flashType,
	}
	s.Render(w, "file", data)
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
	msg := fmt.Sprintf("扫描完成: 发现 %d 个文件, 新增 %d 个", result.FilesFound, result.FilesAdded)
	s.Redirect(w, r, "/files", msg, "success")
}

func (s *Server) HandleImportFile(w http.ResponseWriter, r *http.Request) {
	path := r.FormValue("path")
	if path == "" {
		s.Redirect(w, r, "/files", "请输入文件路径", "error")
		return
	}
	file, err := s.FileSvc.ImportFile(path)
	if err != nil {
		s.Redirect(w, r, "/files", err.Error(), "error")
		return
	}
	s.Redirect(w, r, "/files", "文件 «"+file.FileName+"» 已导入", "success")
}

func (s *Server) HandleTagFile(w http.ResponseWriter, r *http.Request) {
	fileIDStr := r.FormValue("file_id")
	tagName := r.FormValue("tag_name")

	fileID, err := ParseInt(fileIDStr)
	if err != nil {
		s.Redirect(w, r, "/files", "无效的文件ID", "error")
		return
	}

	files, _, _ := s.FileSvc.ListFiles(0, 0)
	for _, f := range files {
		if f.FileID == fileID {
			if err := s.FileSvc.TagFile(f.FilePath, tagName); err != nil {
				s.Redirect(w, r, "/files", err.Error(), "error")
				return
			}
			s.Redirect(w, r, "/files", "标签 «"+tagName+"» 已添加到 «"+f.FileName+"»", "success")
			return
		}
	}
	s.Redirect(w, r, "/files", "文件不存在", "error")
}

func (s *Server) HandleUntagFile(w http.ResponseWriter, r *http.Request) {
	fileIDStr := r.FormValue("file_id")
	tagName := r.FormValue("tag_name")

	fileID, err := ParseInt(fileIDStr)
	if err != nil {
		s.Redirect(w, r, "/files", "无效的文件ID", "error")
		return
	}

	files, _, _ := s.FileSvc.ListFiles(0, 0)
	for _, f := range files {
		if f.FileID == fileID {
			if err := s.FileSvc.UntagFile(f.FilePath, tagName); err != nil {
				s.Redirect(w, r, "/files", err.Error(), "error")
				return
			}
			s.Redirect(w, r, "/files", "标签 «"+tagName+"» 已从 «"+f.FileName+"» 移除", "success")
			return
		}
	}
	s.Redirect(w, r, "/files", "文件不存在", "error")
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
	idStr := r.PathValue("id")
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

	mediaType := IsMediaType(file.FileType)
	data := map[string]any{
		"Active":   "files",
		"FileInfo": map[string]any{"File": file, "Tags": tags, "MediaType": mediaType},
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
