package main

import (
	"context"
	"fmt"
	"mime"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/XingfenD/tag_file/cmd/gui/player"
	"github.com/XingfenD/tag_file/internal/model"
	categorySvc "github.com/XingfenD/tag_file/internal/service/category"
	configSvc "github.com/XingfenD/tag_file/internal/service/config"
	exportSvc "github.com/XingfenD/tag_file/internal/service/export"
	fileSvc "github.com/XingfenD/tag_file/internal/service/file"
	searchSvc "github.com/XingfenD/tag_file/internal/service/search"
	tagSvc "github.com/XingfenD/tag_file/internal/service/tag"
)

type App struct {
	ctx         context.Context
	vlcPlayer   *player.Player
	CategorySvc *categorySvc.Service
	TagSvc      *tagSvc.Service
	FileSvc     *fileSvc.Service
	SearchSvc   *searchSvc.Service
	ConfigSvc   *configSvc.Service
	ExportSvc   *exportSvc.Service
}

func NewApp(
	catSvc *categorySvc.Service,
	tSvc *tagSvc.Service,
	fSvc *fileSvc.Service,
	srchSvc *searchSvc.Service,
	cfgSvc *configSvc.Service,
	expSvc *exportSvc.Service,
) *App {
	return &App{
		CategorySvc: catSvc,
		TagSvc:      tSvc,
		FileSvc:     fSvc,
		SearchSvc:   srchSvc,
		ConfigSvc:   cfgSvc,
		ExportSvc:   expSvc,
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) shutdown(ctx context.Context) {
	if a.vlcPlayer != nil {
		a.vlcPlayer.Close()
	}
}

// ---- Home ----

func (a *App) GetStats() map[string]any {
	files, _, _ := a.FileSvc.ListFiles(0, 0)
	categories, _, _ := a.CategorySvc.ListCategories(0, 0)
	tags, _, _ := a.TagSvc.ListTags("", 0, 0)
	configs, _, _ := a.ConfigSvc.List(0, 0)
	return map[string]any{
		"fileCount":     len(files),
		"categoryCount": len(categories),
		"tagCount":      len(tags),
		"configCount":   len(configs),
	}
}

// ---- Categories ----

func (a *App) ListCategories(q string, page, pageSize int) map[string]any {
	if pageSize <= 0 {
		pageSize = 10
	}
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * pageSize

	var categories []model.TagCategory
	var total int64
	var err error
	if q != "" {
		categories, total, err = a.CategorySvc.SearchCategories(q, offset, pageSize)
	} else {
		categories, total, err = a.CategorySvc.ListCategories(offset, pageSize)
	}
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	return map[string]any{
		"categories": categories,
		"pagination": buildPagination(page, pageSize, total, len(categories)),
		"query":      q,
	}
}

func (a *App) AddCategory(name, desc string) map[string]any {
	if _, err := a.CategorySvc.AddCategory(name, desc); err != nil {
		return map[string]any{"error": err.Error()}
	}
	return map[string]any{"success": true, "message": "分类 «" + name + "» 已添加"}
}

func (a *App) RemoveCategory(id uint) map[string]any {
	categories, _, _ := a.CategorySvc.ListCategories(0, 0)
	for _, c := range categories {
		if c.CategoryID == id {
			if err := a.CategorySvc.RemoveCategory(c.CategoryName); err != nil {
				return map[string]any{"error": err.Error()}
			}
			return map[string]any{"success": true, "message": "分类 «" + c.CategoryName + "» 已删除"}
		}
	}
	return map[string]any{"error": "分类不存在"}
}

func (a *App) UpdateCategory(id uint, name, desc string) map[string]any {
	if _, err := a.CategorySvc.UpdateCategory(id, name, desc); err != nil {
		return map[string]any{"error": err.Error()}
	}
	return map[string]any{"success": true, "message": "分类 «" + name + "» 已更新"}
}

// ---- Tags ----

func (a *App) ListTags(view, categoryID, q string, page, pageSize int) map[string]any {
	allTags, _, _ := a.TagSvc.ListTags("", 0, 0)
	idToTag := make(map[uint]*model.Tag, len(allTags))
	for i := range allTags {
		idToTag[allTags[i].TagID] = &allTags[i]
	}

	allTagsData := buildAllTagsData(allTags, idToTag)
	categories, _, _ := a.CategorySvc.ListCategories(0, 0)

	if view == "tree" {
		categoryName := ""
		if categoryID != "" {
			id := parseUint(categoryID)
			for _, c := range categories {
				if c.CategoryID == id {
					categoryName = c.CategoryName
					break
				}
			}
		}
		tree, err := a.TagSvc.ListTagTree(categoryName)
		if err != nil {
			return map[string]any{"error": err.Error()}
		}
		return map[string]any{
			"view":       "tree",
			"categories": categories,
			"tagTree":    tree,
			"allTags":    allTagsData,
			"categoryID": categoryID,
		}
	}

	// flat view
	if pageSize <= 0 {
		pageSize = 10
	}
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * pageSize

	var tags []model.Tag
	var total int64
	var err error
	if q != "" {
		tags, total, err = a.TagSvc.SearchTags(q, offset, pageSize)
	} else if categoryID != "" {
		id := parseUint(categoryID)
		for _, c := range categories {
			if c.CategoryID == id {
				tags, total, err = a.TagSvc.ListTags(c.CategoryName, offset, pageSize)
				break
			}
		}
	}
	if len(tags) == 0 && categoryID == "" && q == "" {
		tags, total, err = a.TagSvc.ListTags("", offset, pageSize)
	}

	if err != nil {
		return map[string]any{"error": err.Error()}
	}

	return map[string]any{
		"view":       "flat",
		"categories": categories,
		"allTags":    allTagsData,
		"tags":       tags,
		"pagination": buildPagination(page, pageSize, total, 0),
		"query":      q,
		"categoryID": categoryID,
	}
}

func (a *App) AddTag(categoryID uint, tagName, desc, parentTagName string) map[string]any {
	categories, _, _ := a.CategorySvc.ListCategories(0, 0)
	var categoryName string
	for _, c := range categories {
		if c.CategoryID == categoryID {
			categoryName = c.CategoryName
			break
		}
	}
	if categoryName == "" {
		return map[string]any{"error": "分类不存在"}
	}
	if _, err := a.TagSvc.AddTag(categoryName, tagName, desc, parentTagName); err != nil {
		return map[string]any{"error": err.Error()}
	}
	return map[string]any{"success": true, "message": "标签 «" + tagName + "» 已添加"}
}

func (a *App) RemoveTagByID(id uint) map[string]any {
	tags, _, _ := a.TagSvc.ListTags("", 0, 0)
	for _, t := range tags {
		if t.TagID == id {
			if _, err := a.TagSvc.RemoveTag(t.TagName); err != nil {
				return map[string]any{"error": err.Error()}
			}
			return map[string]any{"success": true, "message": "标签 «" + t.TagName + "» 已删除"}
		}
	}
	return map[string]any{"error": "标签不存在"}
}

func (a *App) UpdateTag(id uint, name, desc, parentName string) map[string]any {
	if _, err := a.TagSvc.UpdateTag(id, name, desc, parentName); err != nil {
		return map[string]any{"error": err.Error()}
	}
	return map[string]any{"success": true, "message": "标签 «" + name + "» 已更新"}
}

func (a *App) GetAPITags(categoryID uint) []map[string]any {
	if categoryID == 0 {
		return []map[string]any{}
	}
	categories, _, _ := a.CategorySvc.ListCategories(0, 0)
	var categoryName string
	for _, c := range categories {
		if c.CategoryID == categoryID {
			categoryName = c.CategoryName
			break
		}
	}
	if categoryName == "" {
		return []map[string]any{}
	}
	tags, _, _ := a.TagSvc.ListTags(categoryName, 0, 0)
	idToTag := make(map[uint]*model.Tag, len(tags))
	for i := range tags {
		idToTag[tags[i].TagID] = &tags[i]
	}
	items := make([]map[string]any, 0, len(tags))
	for i := range tags {
		items = append(items, map[string]any{
			"id":   tags[i].TagID,
			"name": tags[i].TagName,
			"path": buildTagPathMap(tags[i].TagID, idToTag),
			"cat":  categoryName,
			"cid":  tags[i].CategoryID,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i]["path"].(string) < items[j]["path"].(string)
	})
	return items
}

func (a *App) GetTagValues(tagName string) []map[string]any {
	if tagName == "" {
		return []map[string]any{}
	}
	values, err := a.TagSvc.ListTagValues(tagName)
	if err != nil {
		return []map[string]any{}
	}
	items := make([]map[string]any, 0, len(values))
	for _, v := range values {
		items = append(items, map[string]any{
			"id":    v.ValueID,
			"value": v.Value,
		})
	}
	return items
}

func (a *App) AddTagValue(tagName, value string) map[string]any {
	if tagName == "" || value == "" {
		return map[string]any{"error": "标签名和值不能为空"}
	}
	if _, err := a.TagSvc.AddTagValue(tagName, value); err != nil {
		return map[string]any{"error": err.Error()}
	}
	return map[string]any{"success": true, "message": "值 «" + value + "» 已添加"}
}

func (a *App) RemoveTagValue(tagName, value string) map[string]any {
	if tagName == "" || value == "" {
		return map[string]any{"error": "标签名和值不能为空"}
	}
	if err := a.TagSvc.RemoveTagValue(tagName, value); err != nil {
		return map[string]any{"error": err.Error()}
	}
	return map[string]any{"success": true, "message": "值 «" + value + "» 已删除"}
}

// ---- Files ----

func (a *App) GetFileTree() map[string]any {
	excludeExts := getHideExtensions(a.ConfigSvc)
	excludePatterns := loadExcludePatterns(a.ConfigSvc)
	patternsStr := getScanExcludeRaw(a.ConfigSvc)

	var tree []*FileTreeNode
	cached, cacheErr := loadTreeCache(patternsStr)
	if cacheErr == nil {
		tree = cached
	} else {
		var err error
		tree, err = buildFileTreeFromFS(".", excludeExts, excludePatterns)
		if err != nil {
			return map[string]any{"error": "遍历文件系统失败: " + err.Error()}
		}
		saveTreeCache(tree, patternsStr)
	}

	indexed, _, _ := a.FileSvc.ListFiles(0, 0)
	indexedMap := make(map[string]*model.File, len(indexed))
	for i := range indexed {
		indexedMap[indexed[i].FilePath] = &indexed[i]
	}
	markTreeIndexed(tree, indexedMap)
	for _, n := range tree {
		computeDirTagCnt(n)
	}

	return map[string]any{
		"tree": tree,
	}
}

func (a *App) ScanDirectory(dir string) map[string]any {
	if dir == "" {
		dir = "."
	}
	result, err := a.FileSvc.ScanDirectoryOptions(dir, true)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	os.Remove(treeCacheFile)
	return map[string]any{
		"success": true,
		"message": fmt.Sprintf("扫描完成: 发现 %d 个文件, 新增 %d 个", result.FilesFound, result.FilesAdded),
	}
}

func (a *App) RefreshTree() map[string]any {
	os.Remove(treeCacheFile)
	return map[string]any{"success": true, "message": "目录缓存已刷新"}
}

func (a *App) ImportFile(path string) map[string]any {
	if path == "" {
		return map[string]any{"error": "请选择文件或输入路径"}
	}
	file, err := a.FileSvc.ImportFile(path)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	os.Remove(treeCacheFile)
	return map[string]any{"success": true, "message": "文件 «" + file.FileName + "» 已导入", "file": file}
}

func (a *App) GetFileDetail(filePath string, groupID uint, page, pageSize int) map[string]any {
	if filePath == "" {
		return map[string]any{"error": "无效的文件路径"}
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * pageSize

	file, _, _, err := a.FileSvc.GetFileInfo(filePath)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}

	groups, _ := a.FileSvc.ListGroups(filePath)
	if groupID == 0 && len(groups) > 0 {
		groupID = groups[0].GroupID
	}

	var fileTags []model.TagGroupTag
	var ftTotal int64
	if groupID > 0 {
		fileTags, ftTotal, _ = a.FileSvc.ListFileTagsByGroup(filePath, groupID, offset, pageSize)
	}

	previewTags, _, _ := a.FileSvc.ListFileTags(filePath, 0, 20)

	allTags, _, _ := a.TagSvc.ListTags("", 0, 0)
	idToTag := make(map[uint]*model.Tag, len(allTags))
	for i := range allTags {
		idToTag[allTags[i].TagID] = &allTags[i]
	}
	tagPathMap := make(map[uint]string, len(allTags))
	for _, t := range allTags {
		tagPathMap[t.TagID] = buildTagPathMap(t.TagID, idToTag)
	}

	// deduplicate preview tags
	seen := make(map[string]bool)
	deduped := make([]model.TagGroupTag, 0, len(previewTags))
	for _, ft := range previewTags {
		tagPath := tagPathMap[ft.Tag.TagID]
		text := tagPath
		if ft.TagValue != "" {
			text += ":" + ft.TagValue
		}
		if !seen[text] {
			seen[text] = true
			deduped = append(deduped, ft)
		}
	}

	tagDisplayPaths := make(map[uint]string, len(fileTags))
	for _, ft := range fileTags {
		tagDisplayPaths[ft.ID] = tagPathMap[ft.Tag.TagID]
	}

	allTagsWithPath := buildAllTagsData(allTags, idToTag)
	mediaType := isMediaType(file.FileType)
	videoPlayable := mediaType == "video" && isPlayableVideo(file.FileType)

	// directory siblings
	dir := filepath.Dir(filePath)
	indexedPaths, _, _ := a.FileSvc.ListFiles(0, 0)
	indexedSet := make(map[string]bool)
	var dirIndexed []map[string]any
	for _, f := range indexedPaths {
		indexedSet[f.FilePath] = true
		if filepath.Dir(f.FilePath) == dir {
			dirIndexed = append(dirIndexed, map[string]any{
				"name":      f.FileName,
				"path":      f.FilePath,
				"size":      f.FileSize,
				"type":      f.FileType,
				"tagCnt":    f.TagCnt,
				"isCurrent": f.FilePath == filePath,
			})
		}
	}

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
			"name":      e.Name(),
			"path":      p,
			"size":      info.Size(),
			"type":      mime.TypeByExtension(filepath.Ext(e.Name())),
			"isCurrent": p == filePath,
		})
	}
	sort.Slice(dirUnindexed, func(i, j int) bool {
		return dirUnindexed[i]["name"].(string) < dirUnindexed[j]["name"].(string)
	})

	return map[string]any{
		"file":          fileToMap(file),
		"mediaType":     mediaType,
		"videoPlayable": videoPlayable,
		"previewTags":   deduped,
		"fileTags":      fileTags,
		"tagPaths":      tagDisplayPaths,
		"groups":        groups,
		"groupID":       groupID,
		"allTags":       allTagsWithPath,
		"pagination":    buildPagination(page, pageSize, ftTotal, len(fileTags)),
		"dir":           dir,
		"dirIndexed":    dirIndexed,
		"dirUnindexed":  dirUnindexed,
	}
}

func (a *App) TagFile(filePath string, tagID uint, tagValue string, groupID uint) map[string]any {
	if filePath == "" {
		return map[string]any{"error": "无效的文件路径"}
	}
	if tagID == 0 {
		return map[string]any{"error": "无效的标签ID"}
	}
	var err error
	if groupID > 0 {
		err = a.FileSvc.TagFileToGroup(filePath, tagID, tagValue, groupID)
	}
	if groupID == 0 || err != nil {
		err = a.FileSvc.TagFile(filePath, tagID, tagValue)
	}
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	return map[string]any{"success": true, "message": "标签已添加"}
}

func (a *App) UntagFile(filePath string, tagID uint, tagValue string, groupID uint) map[string]any {
	if filePath == "" {
		return map[string]any{"error": "无效的文件路径"}
	}
	if tagID == 0 {
		return map[string]any{"error": "无效的标签ID"}
	}
	var err error
	if groupID > 0 {
		err = a.FileSvc.UntagFileFromGroup(filePath, tagID, tagValue, groupID)
	}
	if groupID == 0 || err != nil {
		err = a.FileSvc.UntagFile(filePath, tagID, tagValue)
	}
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	return map[string]any{"success": true, "message": "标签已移除"}
}

func (a *App) UpdateFileTagValue(ftID uint, value string) map[string]any {
	if err := a.FileSvc.UpdateTagValue(ftID, value); err != nil {
		return map[string]any{"error": err.Error()}
	}
	return map[string]any{"success": true}
}

func (a *App) AddGroup(filePath, groupName string) map[string]any {
	if filePath == "" || groupName == "" {
		return map[string]any{"error": "参数不完整"}
	}
	group, err := a.FileSvc.CreateGroup(filePath, groupName)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	return map[string]any{
		"success":   true,
		"groupID":   group.GroupID,
		"groupName": group.GroupName,
		"isDefault": group.IsDefault,
		"message":   "标签组 «" + groupName + "» 已创建",
	}
}

func (a *App) RemoveGroup(filePath string, groupID uint) map[string]any {
	if filePath == "" || groupID == 0 {
		return map[string]any{"error": "参数不完整"}
	}
	if err := a.FileSvc.DeleteGroup(filePath, groupID); err != nil {
		return map[string]any{"error": err.Error()}
	}
	return map[string]any{"success": true, "message": "标签组已删除"}
}

func (a *App) RenameGroup(filePath string, groupID uint, newName string) map[string]any {
	if filePath == "" || groupID == 0 || newName == "" {
		return map[string]any{"error": "参数不完整"}
	}
	if err := a.FileSvc.RenameGroup(filePath, groupID, newName); err != nil {
		return map[string]any{"error": err.Error()}
	}
	return map[string]any{"success": true, "groupID": groupID, "groupName": newName, "message": "标签组已重命名"}
}

func (a *App) GetFileGroups(filePath string) map[string]any {
	if filePath == "" {
		return map[string]any{"error": "无效的文件路径"}
	}
	file, _, _, err := a.FileSvc.GetFileInfo(filePath)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	groups, _ := a.FileSvc.ListGroups(filePath)
	type groupInfo struct {
		Group    model.TagGroup `json:"group"`
		TagCount int64          `json:"tagCount"`
	}
	var groupList []groupInfo
	for _, g := range groups {
		_, total, _ := a.FileSvc.ListFileTagsByGroup(filePath, g.GroupID, 0, 0)
		groupList = append(groupList, groupInfo{Group: g, TagCount: total})
	}
	return map[string]any{
		"file":   fileToMap(file),
		"groups": groupList,
	}
}

func (a *App) RenameFile(filePath, newName string) map[string]any {
	if filePath == "" || newName == "" {
		return map[string]any{"error": "参数不完整"}
	}
	newFile, err := a.FileSvc.RenameFile(filePath, newName)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	return map[string]any{
		"success":  true,
		"message":  "文件已重命名为 «" + newName + "»",
		"filePath": newFile.FilePath,
	}
}

func (a *App) GetFileInfoByID(id uint) map[string]any {
	files, _, _ := a.FileSvc.ListFiles(0, 0)
	var filePath string
	for _, f := range files {
		if f.FileID == id {
			filePath = f.FilePath
			break
		}
	}
	if filePath == "" {
		return map[string]any{"error": "文件不存在"}
	}
	file, tags, _, err := a.FileSvc.GetFileInfo(filePath)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	return map[string]any{
		"file":      fileToMap(file),
		"tags":      tags,
		"mediaType": isMediaType(file.FileType),
	}
}

func (a *App) GetFileRawData(fileID uint) map[string]any {
	files, _, _ := a.FileSvc.ListFiles(0, 0)
	var filePath string
	for _, f := range files {
		if f.FileID == fileID {
			filePath = f.FilePath
			break
		}
	}
	if filePath == "" {
		return map[string]any{"error": "文件不存在"}
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	return map[string]any{
		"data":     data,
		"filePath": filePath,
	}
}

// ---- Search ----

func (a *App) SearchByTags(include, exclude []string, logic string) map[string]any {
	if logic == "" {
		logic = "and"
	}
	if len(include) == 0 && len(exclude) == 0 {
		return map[string]any{"error": "请至少输入一个搜索条件"}
	}
	results, err := a.SearchSvc.SearchByTags(include, exclude, logic)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	return map[string]any{
		"results":  results,
		"include":  include,
		"exclude":  exclude,
		"logic":    logic,
		"searched": true,
	}
}

// ---- Config ----

func (a *App) ListConfigs(q string, page, pageSize int) map[string]any {
	if pageSize <= 0 {
		pageSize = 10
	}
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * pageSize

	var configs []model.Config
	var total int64
	var err error
	if q != "" {
		configs, total, err = a.ConfigSvc.SearchConfigs(q, offset, pageSize)
	} else {
		configs, total, err = a.ConfigSvc.List(offset, pageSize)
	}
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	return map[string]any{
		"configs":    configs,
		"pagination": buildPagination(page, pageSize, total, len(configs)),
		"query":      q,
	}
}

func (a *App) SetConfig(key, value string) map[string]any {
	if err := a.ConfigSvc.Set(key, value); err != nil {
		return map[string]any{"error": err.Error()}
	}
	return map[string]any{"success": true, "message": "配置 «" + key + "» 已保存"}
}

func (a *App) UnsetConfig(key string) map[string]any {
	if err := a.ConfigSvc.Delete(key); err != nil {
		return map[string]any{"error": err.Error()}
	}
	return map[string]any{"success": true, "message": "配置 «" + key + "» 已删除"}
}

func (a *App) UpdateConfig(key, value string) map[string]any {
	if err := a.ConfigSvc.Update(key, value); err != nil {
		return map[string]any{"error": err.Error()}
	}
	return map[string]any{"success": true, "message": "配置 «" + key + "» 已更新"}
}

func (a *App) ExportSQL() map[string]any {
	var sb strings.Builder
	if err := a.ExportSvc.ExportToSQL(&sb); err != nil {
		return map[string]any{"error": err.Error()}
	}
	return map[string]any{"sql": sb.String()}
}

// ---- Utilities ----

func (a *App) FormatFileSize(size int64) string {
	return formatSize(size)
}

func (a *App) GetMediaType(fileType string) string {
	return isMediaType(fileType)
}

func (a *App) PathEscape(path string) string {
	return url.PathEscape(path)
}

// ---- Helpers ----

func buildPagination(page, pageSize int, total int64, count int) map[string]any {
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
	end := offset + count
	if end > int(total) {
		end = int(total)
	}
	return map[string]any{
		"page":       page,
		"pageSize":   pageSize,
		"total":      total,
		"totalPages": totalPages,
		"hasPrev":    page > 1,
		"hasNext":    page < totalPages,
		"prevPage":   page - 1,
		"nextPage":   page + 1,
		"start":      offset + 1,
		"end":        end,
	}
}

func buildTagPathMap(tagID uint, idToTag map[uint]*model.Tag) string {
	var build func(id uint, depth int) string
	build = func(id uint, depth int) string {
		if depth > 20 {
			return "..."
		}
		t, ok := idToTag[id]
		if !ok {
			return "?"
		}
		if t.ParentID == nil {
			return t.TagName
		}
		parent, ok := idToTag[*t.ParentID]
		if !ok {
			return t.TagName
		}
		return build(parent.TagID, depth+1) + " / " + t.TagName
	}
	return build(tagID, 0)
}

func buildAllTagsData(allTags []model.Tag, idToTag map[uint]*model.Tag) []map[string]any {
	result := make([]map[string]any, 0, len(allTags))
	for _, t := range allTags {
		result = append(result, map[string]any{
			"name": t.TagName,
			"path": buildTagPathMap(t.TagID, idToTag),
			"cat":  t.Category.CategoryName,
			"cid":  t.CategoryID,
			"id":   t.TagID,
		})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i]["path"].(string) < result[j]["path"].(string)
	})
	return result
}

func fileToMap(f *model.File) map[string]any {
	return map[string]any{
		"fileID":    f.FileID,
		"filePath":  f.FilePath,
		"fileName":  f.FileName,
		"fileSize":  f.FileSize,
		"fileType":  f.FileType,
		"tagCnt":    f.TagCnt,
		"createdAt": f.CreatedAt,
		"updatedAt": f.UpdatedAt,
	}
}

func parseUint(s string) uint {
	var v uint
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0
		}
		v = v*10 + uint(c-'0')
	}
	return v
}
