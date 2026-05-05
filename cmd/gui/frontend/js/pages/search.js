// search.js - Tag search page
function renderSearch(data) {
    var html = '<h2>标签搜索</h2>';

    // Search form
    html += '<div class="card"><h3>搜索条件</h3>' +
        '<form onsubmit="return doSearch(this)" class="search-box">' +
        '<div class="form-group"><label>包含标签（逗号分隔）</label>' +
        '<input name="include" placeholder="例如: Go源码, 重要" style="width:250px" value="' + escapeHtml(data.include || '') + '"></div>' +
        '<div class="form-group"><label>排除标签（逗号分隔）</label>' +
        '<input name="exclude" placeholder="例如: 废弃" style="width:250px" value="' + escapeHtml(data.exclude || '') + '"></div>' +
        '<div class="form-group"><label>逻辑</label>' +
        '<select name="logic">' +
        '<option value="and"' + (data.logic === 'and' ? ' selected' : '') + '>AND (同时包含)</option>' +
        '<option value="or"' + (data.logic === 'or' ? ' selected' : '') + '>OR (包含任一)</option>' +
        '</select></div>' +
        '<button type="submit" class="btn-primary">搜索</button></form></div>';

    // Results
    if (data.searched) {
        var results = data.results || [];
        html += '<div class="card"><h3>搜索结果 (' + results.length + ' 个文件)</h3>';
        if (results.length > 0) {
            results.forEach(function(r) {
                html += '<div class="search-result">' +
                    '<div class="file-name">' + escapeHtml(r.File.FileName) + '</div>' +
                    '<div class="file-meta">路径: ' + escapeHtml(r.File.FilePath) + ' | 大小: ' + API.formatFileSize(r.File.FileSize) + ' | 类型: ' + escapeHtml(r.File.FileType) + '</div>' +
                    '<div class="file-tags">' + (r.Tags || []).map(function(t) { return '<span class="tag-badge">' + escapeHtml(t.TagName) + '</span>'; }).join(' ') + '</div>' +
                    '</div>';
            });
        } else {
            html += '<p class="empty">未找到匹配文件</p>';
        }
        html += '</div>';
    }

    document.getElementById('page-content').innerHTML = html;
}

function doSearch(form) {
    var include = (form.include.value || '').split(',').map(function(s) { return s.trim(); }).filter(function(s) { return s; });
    var exclude = (form.exclude.value || '').split(',').map(function(s) { return s.trim(); }).filter(function(s) { return s; });
    var logic = form.logic.value;
    API.searchByTags(include, exclude, logic).then(function(r) {
        if (r.error) { App.flash(r.error, 'error'); return; }
        renderSearch(r);
    });
    return false;
}
