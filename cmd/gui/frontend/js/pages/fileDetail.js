// fileDetail.js - File detail page with media preview and tag management
var fd = {};

function renderFileDetail(data) {
    if (!data.file && !data.error) return;
    fd = data;
    fd.filePath = data.file ? data.file.filePath : '';
    fd.groupID = data.groupID || 0;
    fd.currentPage = data.pagination ? data.pagination.page : 1;
    fd.allTags = data.allTags || [];

    var file = data.file || {};
    var groups = data.groups || [];
    var pg = data.pagination || {};
    var fileTags = data.fileTags || [];
    var previewTags = data.previewTags || [];
    var dirIndexed = data.dirIndexed || [];
    var dirUnindexed = data.dirUnindexed || [];
    var dir = data.dir || '';

    var html = '<h2><a href="#" onclick="App.navigate(\'files\');return false" style="text-decoration:none;color:#e94560">&larr; 返回文件列表</a></h2>';

    // Layout
    html += '<div class="detail-layout">';

    // --- Sidebar ---
    html += '<aside class="detail-sidebar"><div class="sidebar-section"><h4>' + escapeHtml(dir) + '</h4></div>';
    if (dirIndexed.length > 0) {
        html += '<div class="sidebar-section"><h4>已索引</h4>';
        dirIndexed.forEach(function(f) {
            html += '<div class="sidebar-file' + (f.isCurrent ? ' current' : '') + '">' +
                '<a href="#" onclick="openFileDetailSibling(\'' + escapeJs(f.path) + '\');return false">' +
                iconForFile(f.type) + ' ' + escapeHtml(f.name) +
                (f.tagCnt > 0 ? ' <span class="tag-cnt">' + f.tagCnt + '</span>' : '') +
                (f.isCurrent ? ' &larr;' : '') + '</a>' +
                ' <span style="font-size:11px;color:#888">' + formatSizeStr(f.size) + '</span></div>';
        });
        html += '</div>';
    }
    if (dirUnindexed.length > 0) {
        html += '<div class="sidebar-section"><h4>未索引</h4>';
        dirUnindexed.forEach(function(f) {
            html += '<div class="sidebar-file' + (f.isCurrent ? ' current' : '') + '">' +
                '<span style="color:#aaa">' + iconForFile(f.type) + ' ' + escapeHtml(f.name);
            if (f.isCurrent) html += ' &larr;';
            html += '</span>' +
                ' <span style="font-size:11px;color:#aaa">' + formatSizeStr(f.size) + '</span></div>';
        });
        html += '</div>';
    }
    html += '</aside>';

    // --- Main content ---
    html += '<div class="detail-main">';
    html += '<div class="card"><h3>文件详情: ' + escapeHtml(file.fileName) + '</h3>';
    html += '<table style="margin-bottom:16px">' +
        '<tr><th style="width:100px">ID</th><td>' + (file.fileID || '') + '</td></tr>' +
        '<tr><th>路径</th><td>' + escapeHtml(file.filePath || '') + '</td></tr>' +
        '<tr><th>大小</th><td>' + formatSizeStr(file.fileSize || 0) + '</td></tr>' +
        '<tr><th>类型</th><td>' + escapeHtml(file.fileType || '') + '</td></tr>' +
        '<tr><th>标签 (' + (pg.total || 0) + ')</th><td id="preview-tags-area">';

    if (previewTags.length > 0) {
        previewTags.forEach(function(ft) {
            var tagPath = data.tagPaths ? (data.tagPaths[ft.ID] || ft.Tag.TagName) : ft.Tag.TagName;
            html += '<span class="tag-badge" data-tag-name="' + escapeHtml(ft.Tag.TagName) + '">' + escapeHtml(tagPath) + (ft.TagValue ? ':' + escapeHtml(ft.TagValue) : '') + '</span> ';
        });
    } else {
        html += '<span class="empty">无标签</span>';
    }
    html += '</td></tr></table>';

    // Rename
    html += '<div style="margin-top:16px;padding-top:16px;border-top:1px solid #eee">' +
        '<h4 style="margin-bottom:8px;font-size:14px;color:#555">重命名</h4>' +
        '<form onsubmit="return doRenameFile(this)" class="row">' +
        '<input type="hidden" name="file_path" value="' + escapeHtml(file.filePath) + '">' +
        '<input name="new_name" required placeholder="新文件名" value="' + escapeHtml(file.fileName) + '" style="width:250px">' +
        '<button type="submit" class="btn-primary">重命名</button></form></div>';

    // Media preview
    var mediaType = data.mediaType || '';
    if (mediaType === 'image') {
        html += '<div class="media-preview" id="media-area">' +
            '<p style="color:#888;font-size:13px">加载中...</p></div>';
    } else if (mediaType === 'video') {
        html += '<div class="media-preview" id="media-area"></div>';
    } else if (mediaType === 'audio') {
        html += '<div class="media-preview" id="media-area">' +
            '<p style="color:#888;font-size:13px">加载中...</p></div>';
    }

    // Tag management
    html += '<div style="margin-top:16px;padding-top:16px;border-top:1px solid #eee">' +
        '<div style="display:flex;align-items:center;justify-content:space-between;margin-bottom:12px">' +
        '<h4 style="margin:0;font-size:14px;color:#555">标签列表</h4>' +
        '<div style="display:flex;align-items:center;gap:8px">' +
        '<a href="#" onclick="openFileGroups();return false" class="btn-default btn-sm" style="text-decoration:none">管理标签组</a>' +
        '<span id="tag-flash" style="display:none;padding:2px 10px;font-size:12px"></span>' +
        '<button class="btn-default btn-sm" onclick="fdLoadPage(fd.currentPage)">刷新</button>' +
        '</div></div>';

    // Group tabs
    if (groups.length > 0) {
        html += '<div id="group-tabs" class="group-tabs">';
        groups.forEach(function(g) {
            html += '<button id="group-tab-' + g.GroupID + '" class="group-tab' + (g.GroupID === fd.groupID ? ' active' : '') + '" onclick="fdSwitchGroup(' + g.GroupID + ')">' +
                escapeHtml(g.GroupName);
            if (!g.IsDefault) {
                html += ' <span onclick="event.stopPropagation();fdDoRemoveGroup(' + g.GroupID + ',\'' + escapeJs(g.GroupName) + '\')" style="cursor:pointer;margin-left:4px;opacity:0.6" title="删除标签组">×</span>';
            }
            html += '</button>';
        });
        html += '<button class="btn-default btn-xs" onclick="fdShowAddGroup()">+</button></div>';

        html += '<div id="add-group-form" style="display:none;margin-bottom:12px">' +
            '<form onsubmit="return fdDoAddGroup(this)" class="row">' +
            '<input name="group_name" placeholder="标签组名称" required style="width:140px">' +
            '<button type="submit" class="btn-primary btn-sm">创建</button>' +
            '<button type="button" class="btn-default btn-sm" onclick="document.getElementById(\'add-group-form\').style.display=\'none\'">取消</button>' +
            '</form></div>';
    }

    // Add tag form
    html += '<form onsubmit="return fdDoTag(this)" class="row" style="margin-bottom:12px">' +
        '<div class="form-group tag-search-wrap">' +
        '<input id="add-tag-input" name="tag_name" required autocomplete="off" placeholder="搜索标签..." style="width:160px">' +
        '<div id="tag-dropdown" class="tag-search-dropdown"></div></div>' +
        '<div class="form-group" id="tag_value_group" style="display:none">' +
        '<input name="tag_value" placeholder="值（可选）" style="width:100px"></div>' +
        '<button type="submit" class="btn-primary btn-sm">添加</button></form>';

    // Tag table
    html += '<div id="tag-table">';
    if (fileTags.length > 0) {
        html += fdRenderTagTable(fileTags, pg, data.tagPaths || {});
    } else {
        html += '<p class="empty">暂无标签</p>';
    }
    html += '</div></div></div></div></div>';

    document.getElementById('page-content').innerHTML = html;

    // Init tag search for the add-tag form
    var tagSearch = setupTagSearch('add-tag-input', 'tag-dropdown', fdLoadTagValueInput);
    tagSearch.setTags(fd.allTags || []);

    // Load media
    if (mediaType === 'video') {
        initVideoPlayer('media-area', file.filePath, file.fileID);
    } else if (mediaType === 'image' || mediaType === 'audio') {
        fdLoadMedia(file.fileID, mediaType);
    }
}

// --- Tag table rendering ---

function fdRenderTagTable(fileTags, pg, tagPaths) {
    var html = '<table><thead><tr><th>标签名</th><th>值</th><th>分类</th><th>操作</th></tr></thead><tbody>';
    fileTags.forEach(function(ft) {
        var val = ft.TagValue || '-';
        var tagPath = tagPaths[ft.ID] || ft.Tag.TagName;
        html += '<tr id="ft-' + ft.ID + '">' +
            '<td data-tag-name="' + escapeHtml(ft.Tag.TagName) + '" data-tag-id="' + ft.Tag.TagID + '">' + escapeHtml(tagPath) + '</td>' +
            '<td class="ft-value">' + escapeHtml(val) + '</td>' +
            '<td>' + escapeHtml(ft.Tag.Category.CategoryName) + '</td>' +
            '<td><button class="btn-default btn-sm" onclick="fdEditFileTagValue(' + ft.ID + ',\'' + escapeJs(ft.TagValue) + '\')">编辑值</button> ' +
            '<button class="btn-danger btn-sm" onclick="fdRemoveFileTag(' + ft.ID + ',' + ft.Tag.TagID + ',\'' + escapeJs(ft.TagValue) + '\')">删除</button></td></tr>';
    });
    html += '</tbody></table>';
    if (pg.totalPages > 1) {
        html += renderPagination(pg, 'fdLoadPage');
    }
    return html;
}

function fdLoadPage(page) {
    fd.currentPage = page;
    API.getFileDetail(fd.filePath, fd.groupID, page, 10).then(function(d) {
        if (d.error) { App.flash(d.error, 'error'); return; }
        fd.fileTags = d.fileTags || [];
        fd.tagPaths = d.tagPaths || {};
        fd.pagination = d.pagination || {};
        document.getElementById('tag-table').innerHTML = d.fileTags.length > 0 ?
            fdRenderTagTable(d.fileTags, d.pagination, d.tagPaths) : '<p class="empty">暂无标签</p>';
        updateTagCount(d.pagination.total || 0);
    });
}

// --- Tag CRUD ---

function fdDoTag(form) {
    var tagInput = document.getElementById('add-tag-input');
    var tagID = parseInt(tagInput.getAttribute('data-tag-id')) || 0;
    if (!tagID) { flashMsg(document.getElementById('tag-flash'), '请从下拉列表中选择标签'); return false; }
    var tagValue = form.tag_value ? form.tag_value.value : '';
    var tagName = tagInput.value;
    API.tagFile(fd.filePath, tagID, tagValue, fd.groupID).then(function(r) {
        if (r.error) { App.flash(r.error, 'error'); return; }
        flashMsg(document.getElementById('tag-flash'), '标签已添加');
        form.reset();
        document.getElementById('tag_value_group').style.display = 'none';
        fdLoadPage(fd.currentPage);
        fdAddTagPreview(tagName, tagValue);
    });
    return false;
}

function fdRemoveFileTag(ftID, tagID, tagValue) {
    if (!confirm('确定删除该标签？')) return;
    API.untagFile(fd.filePath, tagID, tagValue, fd.groupID).then(function(r) {
        if (r.error) { App.flash(r.error, 'error'); return; }
        flashMsg(document.getElementById('tag-flash'), '标签已删除');
        fdLoadPage(fd.currentPage);
        var row = document.getElementById('ft-' + ftID);
        var tagName = row ? row.querySelector('[data-tag-name]').getAttribute('data-tag-name') : '';
        fdRemoveTagPreview(tagName);
    });
}

function fdEditFileTagValue(ftID, curValue) {
    var row = document.getElementById('ft-' + ftID);
    if (!row) return;
    var cell = row.querySelector('.ft-value');
    cell.innerHTML = '<input id="edit-val-' + ftID + '" value="' + escapeHtml(curValue || '') + '" style="width:100px">';
    var actions = row.querySelectorAll('td')[3];
    actions.innerHTML = '<button class="btn-primary btn-sm" onclick="fdSaveFileTagValue(' + ftID + ')">保存</button> ' +
        '<button class="btn-default btn-sm" onclick="fdLoadPage(' + fd.currentPage + ')">取消</button>';
}

function fdSaveFileTagValue(ftID) {
    var val = document.getElementById('edit-val-' + ftID).value;
    API.updateFileTagValue(ftID, val).then(function(r) {
        if (r.error) { App.flash(r.error, 'error'); return; }
        flashMsg(document.getElementById('tag-flash'), '值已更新');
        var row = document.getElementById('ft-' + ftID);
        if (row) {
            row.querySelector('.ft-value').textContent = val || '-';
            var tagName = row.querySelector('[data-tag-name]').getAttribute('data-tag-name');
            var tagID = row.querySelector('[data-tag-name]').getAttribute('data-tag-id');
            row.querySelectorAll('td')[3].innerHTML =
                '<button class="btn-default btn-sm" onclick="fdEditFileTagValue(' + ftID + ',\'' + escapeJs(val) + '\')">编辑值</button> ' +
                '<button class="btn-danger btn-sm" onclick="fdRemoveFileTag(' + ftID + ',' + tagID + ',\'' + escapeJs(val) + '\')">删除</button>';
            fdUpdateTagPreview(tagName, val);
        }
    });
}

// --- Group management ---

function fdSwitchGroup(groupID) {
    fd.groupID = groupID;
    document.querySelectorAll('#group-tabs .group-tab').forEach(function(t) {
        if (t.id === 'group-tab-' + groupID) t.classList.add('active');
        else t.classList.remove('active');
    });
    fdLoadPage(1);
}

function fdShowAddGroup() {
    document.getElementById('add-group-form').style.display = 'block';
}

function fdDoAddGroup(form) {
    API.addGroup(fd.filePath, form.group_name.value).then(function(r) {
        if (r.error) { App.flash(r.error, 'error'); return; }
        var tabs = document.getElementById('group-tabs');
        var btn = document.createElement('button');
        btn.id = 'group-tab-' + r.groupID;
        btn.className = 'group-tab';
        btn.textContent = r.groupName;
        btn.onclick = function() { fdSwitchGroup(r.groupID); };
        var del = document.createElement('span');
        del.style.cssText = 'cursor:pointer;margin-left:4px;opacity:0.6';
        del.textContent = '×';
        del.title = '删除标签组';
        del.onclick = function(e) { e.stopPropagation(); fdDoRemoveGroup(r.groupID, r.groupName); };
        btn.appendChild(del);
        tabs.insertBefore(btn, tabs.lastElementChild);
        document.getElementById('add-group-form').style.display = 'none';
        form.reset();
        fdSwitchGroup(r.groupID);
        App.flash(r.message);
    });
    return false;
}

function fdDoRemoveGroup(groupID, groupName) {
    if (!confirm('确定删除标签组 ' + groupName + '？')) return;
    API.removeGroup(fd.filePath, groupID).then(function(r) {
        if (r.error) { App.flash(r.error, 'error'); return; }
        var tab = document.getElementById('group-tab-' + groupID);
        if (tab) tab.remove();
        var firstTab = document.querySelector('#group-tabs .group-tab[id^="group-tab-"]');
        if (firstTab) {
            var newGID = parseInt(firstTab.id.replace('group-tab-', ''));
            fdSwitchGroup(newGID);
        }
        App.flash(r.message);
    });
}

// --- Tag preview ---

function fdGetTagPath(tagName) {
    for (var i = 0; i < (fd.allTags || []).length; i++) {
        if (fd.allTags[i].name === tagName) return fd.allTags[i].path;
    }
    return tagName;
}

function fdAddTagPreview(tagName, tagValue) {
    var container = document.getElementById('preview-tags-area');
    if (!container) return;
    var tagPath = fdGetTagPath(tagName);
    var label = tagPath + (tagValue ? ':' + tagValue : '');
    var span = document.createElement('span');
    span.className = 'tag-badge';
    span.setAttribute('data-tag-name', tagName);
    span.textContent = label;
    container.appendChild(span);
}

function fdRemoveTagPreview(tagName) {
    var container = document.getElementById('preview-tags-area');
    if (!container) return;
    var spans = container.querySelectorAll('.tag-badge');
    for (var i = 0; i < spans.length; i++) {
        if (spans[i].getAttribute('data-tag-name') === tagName) {
            spans[i].remove();
            return;
        }
    }
}

function fdUpdateTagPreview(tagName, newValue) {
    var container = document.getElementById('preview-tags-area');
    if (!container) return;
    var tagPath = fdGetTagPath(tagName);
    var label = tagPath + (newValue ? ':' + newValue : '');
    var spans = container.querySelectorAll('.tag-badge');
    for (var i = 0; i < spans.length; i++) {
        if (spans[i].getAttribute('data-tag-name') === tagName) {
            spans[i].textContent = label;
            return;
        }
    }
}

function updateTagCount(total) {
    var th = document.querySelector('th');
    if (!th) return;
    var m = th.textContent.match(/标签\s*\((\d+)\)/);
    if (m) th.textContent = '标签 (' + total + ')';
}

// --- Media loading ---

function fdLoadMedia(fileID, mediaType) {
    API.getFileRawData(fileID).then(function(r) {
        var area = document.getElementById('media-area');
        if (!area || r.error) return;
        if (mediaType === 'image') {
            var blob = new Blob([new Uint8Array(r.data)], {type: 'image/*'});
            area.innerHTML = '<img src="' + URL.createObjectURL(blob) + '" style="max-width:100%;max-height:600px;border-radius:6px">';
        } else if (mediaType === 'audio') {
            var blob = new Blob([new Uint8Array(r.data)], {type: 'audio/*'});
            area.innerHTML = '<audio controls style="width:100%"><source src="' + URL.createObjectURL(blob) + '"></audio>';
        }
    });
}

function fdLoadTagValueInput(el) {
    var group = document.getElementById('tag_value_group');
    var tagName = el.value;
    if (!tagName) { group.style.display = 'none'; return; }
    API.getTagValues(tagName).then(function(vals) {
        group.style.display = 'inline-block';
        if (vals && vals.length > 0) {
            group.innerHTML = '<select name="tag_value"><option value="">-- 可选值 --</option>' +
                vals.map(function(v) { return '<option value="' + escapeHtml(v.value) + '">' + escapeHtml(v.value) + '</option>'; }).join('') + '</select>';
        } else {
            group.innerHTML = '<input name="tag_value" placeholder="值（可选）" style="width:100px">';
        }
    });
}

// --- Sidebar file navigation ---

function openFileDetailSibling(filePath) {
    API.getFileDetail(filePath, 0, 1, 10).then(function(d) {
        if (d.error) { App.flash(d.error, 'error'); return; }
        fd = d;
        fd.filePath = filePath;
        fd.groupID = d.groupID || 0;
        fd.currentPage = 1;
        fd.allTags = d.allTags || [];
        renderFileDetail(d);
    });
}

function openFileGroups() {
    API.getFileGroups(fd.filePath).then(function(d) {
        if (d.error) { App.flash(d.error, 'error'); return; }
        _fgFilePath = fd.filePath;
        _fgFile = d.file;
        _fgGroups = d.groups;
        renderFileGroups(d);
    });
}

// --- Rename ---

function doRenameFile(form) {
    API.renameFile(form.file_path.value, form.new_name.value).then(function(r) {
        if (r.error) { App.flash(r.error, 'error'); return; }
        App.flash(r.message);
        fd.filePath = r.filePath;
        openFileDetailSibling(r.filePath);
    });
    return false;
}
