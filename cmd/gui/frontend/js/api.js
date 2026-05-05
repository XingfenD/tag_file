// api.js - Thin wrappers around Wails Go bindings
var API = {
    // Home
    getStats: function() { return window.go.main.App.GetStats(); },

    // Categories
    listCategories: function(q, page, pageSize) { return window.go.main.App.ListCategories(q, page, pageSize); },
    addCategory: function(name, desc) { return window.go.main.App.AddCategory(name, desc); },
    removeCategory: function(id) { return window.go.main.App.RemoveCategory(id); },
    updateCategory: function(id, name, desc) { return window.go.main.App.UpdateCategory(id, name, desc); },

    // Tags
    listTags: function(view, categoryID, q, page, pageSize) { return window.go.main.App.ListTags(view, categoryID, q, page, pageSize); },
    addTag: function(categoryID, name, desc, parentName) { return window.go.main.App.AddTag(categoryID, name, desc, parentName); },
    removeTag: function(id) { return window.go.main.App.RemoveTagByID(id); },
    updateTag: function(id, name, desc, parentName) { return window.go.main.App.UpdateTag(id, name, desc, parentName); },
    getAPITags: function(categoryID) { return window.go.main.App.GetAPITags(categoryID); },
    getTagValues: function(tagName) { return window.go.main.App.GetTagValues(tagName); },
    addTagValue: function(tagName, value) { return window.go.main.App.AddTagValue(tagName, value); },
    removeTagValue: function(tagName, value) { return window.go.main.App.RemoveTagValue(tagName, value); },

    // Files
    getFileTree: function() { return window.go.main.App.GetFileTree(); },
    scanDirectory: function(dir) { return window.go.main.App.ScanDirectory(dir); },
    refreshTree: function() { return window.go.main.App.RefreshTree(); },
    importFile: function(path) { return window.go.main.App.ImportFile(path); },
    getFileDetail: function(filePath, groupID, page, pageSize) { return window.go.main.App.GetFileDetail(filePath, groupID, page, pageSize); },
    tagFile: function(filePath, tagID, tagValue, groupID) { return window.go.main.App.TagFile(filePath, tagID, tagValue, groupID); },
    untagFile: function(filePath, tagID, tagValue, groupID) { return window.go.main.App.UntagFile(filePath, tagID, tagValue, groupID); },
    updateFileTagValue: function(ftID, value) { return window.go.main.App.UpdateFileTagValue(ftID, value); },
    addGroup: function(filePath, groupName) { return window.go.main.App.AddGroup(filePath, groupName); },
    removeGroup: function(filePath, groupID) { return window.go.main.App.RemoveGroup(filePath, groupID); },
    renameGroup: function(filePath, groupID, newName) { return window.go.main.App.RenameGroup(filePath, groupID, newName); },
    getFileGroups: function(filePath) { return window.go.main.App.GetFileGroups(filePath); },
    renameFile: function(filePath, newName) { return window.go.main.App.RenameFile(filePath, newName); },
    getFileInfoByID: function(id) { return window.go.main.App.GetFileInfoByID(id); },
    getFileRawData: function(fileID) { return window.go.main.App.GetFileRawData(fileID); },

    // Search
    searchByTags: function(include, exclude, logic) { return window.go.main.App.SearchByTags(include, exclude, logic); },

    // Config
    listConfigs: function(q, page, pageSize) { return window.go.main.App.ListConfigs(q, page, pageSize); },
    setConfig: function(key, value) { return window.go.main.App.SetConfig(key, value); },
    unsetConfig: function(key) { return window.go.main.App.UnsetConfig(key); },
    updateConfig: function(key, value) { return window.go.main.App.UpdateConfig(key, value); },
    exportSQL: function() { return window.go.main.App.ExportSQL(); },

    // Utility
    formatFileSize: function(size) { return window.go.main.App.FormatFileSize(size); },
    getMediaType: function(fileType) { return window.go.main.App.GetMediaType(fileType); },
    pathEscape: function(path) { return window.go.main.App.PathEscape(path); }
};
