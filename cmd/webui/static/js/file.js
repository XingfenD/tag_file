/* ── File Management ── */
(function() {
'use strict';

var filePath = '';
var currentGroupID = 0;
var currentPage = 1;
var allTags = [];

/* =========================================
   Page Init
   ========================================= */

document.addEventListener('DOMContentLoaded', function() {
    var path = window.location.pathname;
    if (path === '/files' || path === '/files/') {
        initFileTreePage();
    } else if (path.indexOf('/files/detail/') === 0) {
        initFileDetailPage();
    } else if (path.indexOf('/files/groups/') === 0) {
        initFileGroupsPage();
    }
});

/* =========================================
   FILE TREE PAGE
   ========================================= */

function initFileTreePage() {
    loadFileTree();
    setupScanForm();
    setupImportForm();
}

function loadFileTree() {
    var container = document.getElementById('file-tree');
    if (!container) return;
    API.get('/api/files').then(function(data) {
        if (data.ok && data.tree) {
            renderFileTree(data.tree, container);
        } else {
            container.innerHTML = '<div class="empty"><div class="empty-icon">&#128196;</div><div class="empty-text">' + escHtml(data.msg || '加载失败') + '</div></div>';
        }
    }).catch(function() {
        container.innerHTML = '<div class="empty"><div class="empty-icon">&#128196;</div><div class="empty-text">加载失败</div></div>';
    });
}

function renderFileTree(nodes, container) {
    if (!nodes || nodes.length === 0) {
        container.innerHTML = '<div class="empty"><div class="empty-icon">&#128196;</div><div class="empty-text">暂无文件</div><div class="empty-hint">请先扫描目录或导入文件</div></div>';
        return;
    }
    var html = '<div class="tree-root">';
    html += renderTreeNodes(nodes);
    html += '</div>';
    container.innerHTML = html;

    // Restore sessionStorage state
    container.querySelectorAll('.tree-dir[data-tree-path]').forEach(function(el) {
        var key = 'tagfile_tree_open:' + el.getAttribute('data-tree-path');
        if (sessionStorage.getItem(key) === '1') el.open = true;
        el.addEventListener('toggle', function() {
            sessionStorage.setItem(key, el.open ? '1' : '0');
        });
    });
}

function renderTreeNodes(nodes) {
    var html = '';
    for (var i = 0; i < nodes.length; i++) {
        var n = nodes[i];
        if (n.is_dir) {
            html += '<details class="tree-dir" data-tree-path="' + escHtml(n.path || '') + '">';
            html += '<summary class="tree-summary">' + escHtml(n.name) + '/';
            if (n.tag_cnt > 0) {
                html += '<span class="badge" style="margin-left:4px">' + n.tag_cnt + '</span>';
            }
            html += '</summary>';
            html += '<div class="tree-children">';
            if (n.children) html += renderTreeNodes(n.children);
            html += '</div></details>';
        } else {
            var cls = 'tree-file' + (n.indexed ? '' : ' tree-unindexed');
            html += '<div class="' + cls + '">';
            html += '<span class="tree-icon">' + (n.indexed ? iconForName(n.name) : '') + '</span>';
            html += '<span class="tree-name">';
            if (n.indexed) {
                html += '<a href="/files/detail/' + encodePath(n.path) + '">' + escHtml(n.name) + '</a>';
            } else {
                html += '<span style="color:var(--ink-muted)">' + escHtml(n.name) + '</span>';
            }
            html += '</span>';
            if (n.indexed && n.tag_cnt > 0) {
                html += '<span class="badge">' + n.tag_cnt + '</span>';
            }
            html += '<span class="tree-size">' + formatSize(n.file_size || 0) + '</span>';
            if (n.indexed) {
                html += '<a href="/files/detail/' + encodePath(n.path) + '" class="btn btn-default btn-sm">详情</a>';
            } else {
                html += '<span class="tree-status tree-status-no">未索引</span>';
            }
            html += '</div>';
        }
    }
    return html;
}

function setupScanForm() {
    var form = document.getElementById('scan-form');
    if (!form) return;
    form.addEventListener('submit', function(e) {
        e.preventDefault();
        var btn = this.querySelector('button[type="submit"]');
        btn.disabled = true;
        btn.textContent = '扫描中...';
        API.post('/api/files/scan', new FormData(this)).then(function(data) {
            btn.disabled = false;
            btn.textContent = '扫描';
            if (data.ok) {
                showToast(data.msg, 'success');
                loadFileTree();
            } else {
                showToast(data.msg, 'error');
            }
        }).catch(function() {
            btn.disabled = false;
            btn.textContent = '扫描';
            showToast('请求失败', 'error');
        });
    });
}

function setupImportForm() {
    var form = document.getElementById('import-form');
    if (!form) return;
    form.addEventListener('submit', function(e) {
        e.preventDefault();
        var btn = this.querySelector('button[type="submit"]');
        btn.disabled = true;
        btn.textContent = '导入中...';
        fetch('/api/files/import', {method:'POST', body: new FormData(this)})
            .then(function(r) { return r.json(); })
            .then(function(data) {
                btn.disabled = false;
                btn.textContent = '导入';
                if (data.ok) {
                    showToast(data.msg, 'success');
                    loadFileTree();
                } else {
                    showToast(data.msg, 'error');
                }
            })
            .catch(function() {
                btn.disabled = false;
                btn.textContent = '导入';
                showToast('请求失败', 'error');
            });
    });
}

window.refreshTree = function() {
    API.post('/api/files/refresh', '').then(function(data) {
        if (data.ok) {
            showToast(data.msg, 'success');
            loadFileTree();
        } else {
            showToast(data.msg, 'error');
        }
    }).catch(function() {
        showToast('请求失败', 'error');
    });
};

/* =========================================
   FILE DETAIL PAGE
   ========================================= */

function initFileDetailPage() {
    var rawPath = window.location.pathname.substring('/files/detail/'.length);
    try { filePath = decodeURI(rawPath); } catch(e) { filePath = rawPath; }
    currentGroupID = 0;
    currentPage = 1;
    allTags = [];

    fetchAllTagsFromServer(function(tags) {
        allTags = tags || [];
        loadFileDetailData();
    });
}

function loadFileDetailData() {
    var url = '/api/files/detail/' + encodePath(filePath);
    url += '?group_id=' + (currentGroupID || 0) + '&page=' + (currentPage || 1);

    API.get(url).then(function(data) {
        var content = document.getElementById('file-detail-content');
        if (!content) return;
        if (!data.ok) {
            content.innerHTML = '<div class="empty"><div class="empty-icon">&#128196;</div><div class="empty-text">' + escHtml(data.msg || '加载失败') + '</div></div>';
            return;
        }
        var d = data.detail;

        // If currentGroupID is 0 (not set yet), use the first group
        if (currentGroupID === 0 && d.groups && d.groups.length > 0) {
            currentGroupID = d.groups[0].id;
        }

        renderFileInfoCard(d);
        renderDirSidebar(d);
        renderMediaPreview(d);
        renderGroupsAndTags(d);
    }).catch(function() {
        var content = document.getElementById('file-detail-content');
        if (content) content.innerHTML = '<div class="empty"><div class="empty-icon">&#128196;</div><div class="empty-text">加载失败</div></div>';
    });
}

function renderFileInfoCard(d) {
    var content = document.getElementById('file-detail-content');
    var f = d.file;

    var tagsHtml = '';
    var previewTags = (d.tags || []).slice(0, 5);
    if (previewTags.length > 0) {
        tagsHtml = '<div class="tag-preview">';
        for (var i = 0; i < previewTags.length; i++) {
            var t = previewTags[i];
            tagsHtml += '<span class="tag-chip" data-tag-name="' + escHtml(t.tag_name) + '">' + escHtml(t.tag_path) + (t.value ? ':' + escHtml(t.value) : '') + '</span>';
        }
        tagsHtml += '</div>';
    } else {
        tagsHtml = '<span class="text-muted">无标签</span>';
    }

    content.innerHTML =
        '<div class="card">' +
            '<h3 class="card-title">' + escHtml(f.name) + '</h3>' +
            '<table style="margin:14px 0">' +
                '<tr><th style="width:80px">ID</th><td class="text-mono">' + f.id + '</td></tr>' +
                '<tr><th>路径</th><td class="text-mono">' + escHtml(f.path) + '</td></tr>' +
                '<tr><th>大小</th><td>' + formatSize(f.size) + '</td></tr>' +
                '<tr><th>类型</th><td>' + escHtml(f.type) + '</td></tr>' +
                '<tr><th>标签 (' + (d.pagination ? d.pagination.total : 0) + ')</th><td>' + tagsHtml + '</td></tr>' +
            '</table>' +
            '<div style="margin-top:16px;padding-top:16px;border-top:1px solid var(--border)">' +
                '<h4 style="font-size:13px;color:var(--ink-light);margin-bottom:8px;font-weight:500">重命名</h4>' +
                '<form id="rename-form" class="form-row">' +
                    '<input type="hidden" name="file_path" value="' + escHtml(f.path) + '">' +
                    '<div class="form-group">' +
                        '<input name="new_name" required placeholder="新文件名" value="' + escHtml(f.name) + '" style="width:280px">' +
                    '</div>' +
                    '<button type="submit" class="btn btn-primary">重命名</button>' +
                '</form>' +
            '</div>' +
            '<div id="media-preview"></div>' +
            '<div style="margin-top:16px;padding-top:16px;border-top:1px solid var(--border)">' +
                '<div class="flex-between mb-md">' +
                    '<h4 style="margin:0;font-size:14px;color:var(--ink);font-weight:500;font-family:var(--font-serif)">标签列表</h4>' +
                    '<div style="display:flex;align-items:center;gap:8px">' +
                        '<a href="/files/groups/' + encodePath(f.path) + '" class="btn btn-default btn-sm">管理标签组</a>' +
                        '<button class="btn btn-ghost btn-sm" onclick="loadPage(' + currentPage + ')" style="font-size:11px">刷新</button>' +
                    '</div>' +
                '</div>' +
                '<div id="groups-and-tags"></div>' +
            '</div>' +
        '</div>';

    // Setup rename form listener
    var renameForm = document.getElementById('rename-form');
    if (renameForm) {
        renameForm.addEventListener('submit', function(e) {
            e.preventDefault();
            var data = new FormData(this);
            var newName = data.get('new_name');
            var body = new URLSearchParams(data);
            fetch('/api/files/' + encodePath(f.path), {method:'PUT', headers:{'Content-Type':'application/x-www-form-urlencoded'}, body:body})
                .then(function(r) { return r.json(); })
                .then(function(resp) {
                    if (resp.ok) {
                        showToast(resp.msg, 'success');
                        var oldPath = f.path;
                        var idx = oldPath.lastIndexOf('/');
                        var newPath = idx >= 0 ? oldPath.substring(0, idx) + '/' + newName : newName;
                        window.location.href = '/files/detail/' + encodePath(newPath);
                    } else {
                        showToast(resp.msg, 'error');
                    }
                })
                .catch(function() { showToast('请求失败', 'error'); });
        });
    }
}

function renderDirSidebar(d) {
    var dirPathEl = document.getElementById('dir-path');
    if (dirPathEl) dirPathEl.textContent = d.dir_path || '';

    // Indexed files
    var indexedList = document.getElementById('dir-indexed');
    if (indexedList) {
        if (d.dir_files && d.dir_files.length > 0) {
            var html = '';
            for (var i = 0; i < d.dir_files.length; i++) {
                var f = d.dir_files[i];
                var cls = (f.path === d.file.path) ? ' class="current"' : '';
                html += '<li' + cls + '>' +
                    '<a href="/files/detail/' + encodePath(f.path) + '">' +
                        '<span class="dir-file-icon">' + iconForMime(f.type) + '</span>' +
                        '<span class="dir-file-name">' + escHtml(f.name) + '</span>';
                if (f.tag_cnt > 0) html += '<span class="badge">' + f.tag_cnt + '</span>';
                html += '</a>' +
                    '<span class="dir-file-size">' + formatSize(f.size) + '</span>';
                if (f.path === d.file.path) html += '<span class="current-marker">&larr;</span>';
                html += '</li>';
            }
            indexedList.innerHTML = html;
            indexedList.parentElement.style.display = '';
        } else {
            indexedList.innerHTML = '';
            indexedList.parentElement.style.display = 'none';
        }
    }

    // Unindexed files
    var unindexedList = document.getElementById('dir-unindexed');
    if (unindexedList) {
        if (d.dir_unindexed && d.dir_unindexed.length > 0) {
            var html = '';
            for (var i = 0; i < d.dir_unindexed.length; i++) {
                var f = d.dir_unindexed[i];
                html += '<li>' +
                    '<span class="dir-file-icon dimmed">' + iconForMime(f.type) + '</span>' +
                    '<span class="dir-file-name dimmed">' + escHtml(f.name) + '</span>' +
                    '<span class="dir-file-size dimmed">' + formatSize(f.size) + '</span>' +
                '</li>';
            }
            unindexedList.innerHTML = html;
            unindexedList.parentElement.style.display = '';
        } else {
            unindexedList.innerHTML = '';
            unindexedList.parentElement.style.display = 'none';
        }
    }
}

function renderMediaPreview(d) {
    var container = document.getElementById('media-preview');
    if (!container) return;

    if (d.media_type === 'image') {
        container.innerHTML = '<img src="/api/files/raw/' + d.file.id + '" style="max-width:100%;max-height:600px;display:block;margin-top:16px" alt="' + escHtml(d.file.name) + '">';
    } else if (d.media_type === 'video') {
        if (d.video_playable) {
            container.innerHTML = '<video controls style="max-width:100%;max-height:600px;margin-top:16px"><source src="/api/files/raw/' + d.file.id + '" type="' + escHtml(d.file.type) + '"></video>';
        } else {
            container.innerHTML = '<div class="card" style="margin-top:16px;text-align:center"><div class="empty"><div class="empty-icon">&#127910;</div><div class="empty-text">此视频格式（' + escHtml(d.file.type) + '）浏览器不支持直接播放</div></div></div>';
        }
    } else if (d.media_type === 'audio') {
        container.innerHTML = '<audio controls style="width:100%;margin-top:16px"><source src="/api/files/raw/' + d.file.id + '" type="' + escHtml(d.file.type) + '"></audio>';
    }
}

function renderGroupsAndTags(d) {
    var container = document.getElementById('groups-and-tags');
    if (!container) return;

    if (!d.groups || d.groups.length === 0) {
        container.innerHTML = '<div class="empty"><div class="empty-icon">&#127991;</div><div class="empty-text">无标签组</div></div>';
        return;
    }

    var html = '';

    // Group tabs
    html += '<div class="tabs" id="group-tabs">';
    for (var i = 0; i < d.groups.length; i++) {
        var g = d.groups[i];
        var activeClass = (g.id === currentGroupID) ? ' active' : '';
        html += '<a href="#" onclick="switchGroup(' + g.id + ');return false" id="group-tab-' + g.id + '" class="tab' + activeClass + '">' +
            escHtml(g.name);
        if (!g.is_default) {
            html += '<span class="tab-close" onclick="event.stopPropagation();doRemoveGroup(' + g.id + ',\'' + escJsStr(g.name) + '\')" title="删除标签组">&times;</span>';
        }
        html += '</a>';
    }
    html += '<span class="tab-add" onclick="showAddGroup()" title="添加标签组">+</span>';
    html += '</div>';

    // Add group form
    html += '<div id="add-group-form" style="display:none;margin-bottom:12px">' +
        '<form onsubmit="return doAddGroup(this)" class="form-row" style="align-items:end">' +
            '<input type="hidden" name="file_path" value="' + escHtml(d.file.path) + '">' +
            '<div class="form-group">' +
                '<input name="group_name" placeholder="标签组名称" required style="width:150px">' +
            '</div>' +
            '<button type="submit" class="btn btn-primary btn-sm">创建</button>' +
            '<button type="button" class="btn btn-default btn-sm" onclick="document.getElementById(\'add-group-form\').style.display=\'none\'">取消</button>' +
        '</form>' +
    '</div>';

    // Tag form
    html += '<form onsubmit="return doTag(this)" class="form-row">' +
        '<input type="hidden" name="file_path" value="' + escHtml(d.file.path) + '">' +
        '<input type="hidden" name="group_id" value="' + currentGroupID + '">' +
        '<div class="form-group" style="position:relative">' +
            '<input id="add-tag-input" name="tag_name" required autocomplete="off" placeholder="搜索标签..." style="width:170px">' +
            '<div id="tag-dropdown" class="dropdown"></div>' +
        '</div>' +
        '<div class="form-group" id="tag_value_group" style="display:none">' +
            '<input name="tag_value" placeholder="值（可选）" style="width:110px">' +
        '</div>' +
        '<button type="submit" class="btn btn-primary btn-sm">添加</button>' +
    '</form>';

    // Tag table
    html += '<div id="tag-table" class="mt-md"></div>';

    container.innerHTML = html;

    renderTagTable(d.tags, d.pagination);

    // Initialize tag search
    setupTagSearchOnDetail();
}

function renderTagTable(tags, pg) {
    var tableContainer = document.getElementById('tag-table');
    if (!tableContainer) return;

    if (!tags || tags.length === 0) {
        tableContainer.innerHTML = '<div class="empty"><div class="empty-icon">&#127991;</div><div class="empty-text">暂无标签</div><div class="empty-hint">在上方搜索并添加标签</div></div>';
        return;
    }

    var html = '<table><thead><tr><th>标签名</th><th>值</th><th>分类</th><th style="width:140px">操作</th></tr></thead><tbody>';
    for (var i = 0; i < tags.length; i++) {
        var t = tags[i];
        var valHtml = t.value ? escHtml(t.value) : '&mdash;';
        var escVal = escJsStr(t.value || '');
        html += '<tr id="ft-' + t.id + '">' +
            '<td data-tag-name="' + escHtml(t.tag_name) + '" data-tag-id="' + t.tag_id + '" style="font-weight:500">' + escHtml(t.tag_path) + '</td>' +
            '<td class="ft-value">' + valHtml + '</td>' +
            '<td class="text-muted">' + escHtml(t.category) + '</td>' +
            '<td class="col-actions">' +
                '<button class="btn btn-default btn-sm" onclick="editFileTagValue(' + t.id + ',\'' + escVal + '\')">编辑值</button> ' +
                '<button class="btn btn-danger btn-sm" onclick="removeFileTag(' + t.id + ',' + t.tag_id + ',\'' + escVal + '\')">删除</button>' +
            '</td>' +
        '</tr>';
    }
    html += '</tbody></table>';

    // Pagination
    if (pg && pg.total_pages > 1) {
        html += '<div class="pagination">';
        if (pg.has_prev) {
            html += '<a href="#" onclick="loadPage(' + pg.prev_page + ');return false">&laquo; 前页</a>';
        } else {
            html += '<span class="disabled">&laquo; 前页</span>';
        }
        html += '<span style="font-size:12px;color:var(--ink-muted)">第 ' + pg.page + '/' + pg.total_pages + ' 页 (' + pg.total + ' 项)</span>';
        if (pg.has_next) {
            html += '<a href="#" onclick="loadPage(' + pg.next_page + ');return false">后页 &raquo;</a>';
        } else {
            html += '<span class="disabled">后页 &raquo;</span>';
        }
        html += '</div>';
    }

    tableContainer.innerHTML = html;
}

function setupTagSearchOnDetail() {
    var input = document.getElementById('add-tag-input');
    var dropdown = document.getElementById('tag-dropdown');
    if (!input || !dropdown) return;

    var ts = setupTagSearch(input, dropdown, loadTagValueInput);
    if (ts && allTags.length > 0) {
        ts.setTags(allTags);
    }
}

function loadTagValueInput(el) {
    var group = document.getElementById('tag_value_group');
    var tagName = el.value;
    if (!tagName) { group.style.display = 'none'; return; }
    fetch('/api/tags/values?tag_name=' + encodeURIComponent(tagName))
        .then(function(r) { return r.json(); })
        .then(function(data) {
            group.style.display = 'inline-block';
            var vals = data.values || [];
            if (vals.length > 0) {
                var shtml = '<select name="tag_value"><option value="">可选值</option>';
                for (var i = 0; i < vals.length; i++) {
                    shtml += '<option value="' + escHtml(vals[i].value) + '">' + escHtml(vals[i].value) + '</option>';
                }
                shtml += '</select>';
                group.innerHTML = shtml;
            } else {
                group.innerHTML = '<input name="tag_value" placeholder="值（可选）" style="width:110px">';
            }
        })
        .catch(function() {
            group.style.display = 'inline-block';
            group.innerHTML = '<input name="tag_value" placeholder="值（可选）" style="width:110px">';
        });
}

function fetchAllTagsFromServer(callback) {
    var all = [];
    function fetchPage(page) {
        API.get('/api/tags?view=flat&page=' + page).then(function(data) {
            if (data.ok && data.items) {
                for (var i = 0; i < data.items.length; i++) {
                    var item = data.items[i];
                    all.push({name: item.name, path: item.path, cat: '', cid: item.cid, id: item.id});
                }
                if (data.pagination && data.pagination.has_next) {
                    fetchPage(page + 1);
                } else {
                    callback(all);
                }
            } else {
                callback(all);
            }
        }).catch(function() { callback(all); });
    }
    fetchPage(1);
}

/* ── Detail Page: Global handlers for inline onclick ── */

window.loadPage = function(page) {
    currentPage = page;
    loadFileDetailData();
};

window.doTag = function(form) {
    var tagInput = document.getElementById('add-tag-input');
    var tagID = parseInt(tagInput.getAttribute('data-tag-id')) || 0;
    if (!tagID) { showToast('请从下拉列表中选择标签', 'error'); return false; }
    var fd = new FormData(form);
    fd.set('tag_id', tagID);
    fd.delete('tag_name');
    API.post('/api/files/tag', fd).then(function(data) {
        if (!data.ok) { showToast(data.msg, 'error'); return; }
        showToast(data.msg || '标签已添加', 'success');
        form.reset();
        var ti = document.getElementById('add-tag-input');
        if (ti) {
            ti.removeAttribute('data-tag-id');
            ti.classList.remove('tag-selected');
        }
        var valueGroup = document.getElementById('tag_value_group');
        if (valueGroup) valueGroup.style.display = 'none';
        loadFileDetailData();
    }).catch(function() { showToast('请求失败', 'error'); });
    return false;
};

window.removeFileTag = function(ftID, tagID, tagValue) {
    showConfirm('确定删除该标签？', function() {
        var body = new URLSearchParams();
        body.set('file_path', filePath);
        body.set('tag_id', tagID);
        body.set('tag_value', tagValue || '');
        body.set('group_id', currentGroupID);
        fetch('/api/files/tag', {method:'DELETE', headers:{'Content-Type':'application/x-www-form-urlencoded'}, body:body})
            .then(function(r) { return r.json(); })
            .then(function(data) {
                if (!data.ok) { showToast(data.msg, 'error'); return; }
                showToast(data.msg || '标签已删除', 'info');
                loadFileDetailData();
            })
            .catch(function() { showToast('请求失败', 'error'); });
    });
};

window.editFileTagValue = function(ftID, curValue) {
    var row = document.getElementById('ft-' + ftID);
    if (!row) return;
    var cell = row.querySelector('.ft-value');
    cell.innerHTML = '<input id="edit-val-' + ftID + '" value="' + escHtml(curValue || '') + '" style="width:100px">';
    var tdActions = row.querySelectorAll('td')[3];
    tdActions.innerHTML = '<button class="btn btn-primary btn-sm" onclick="saveFileTagValue(' + ftID + ')">保存</button> <button class="btn btn-default btn-sm" onclick="loadPage(' + currentPage + ')">取消</button>';
};

window.saveFileTagValue = function(ftID) {
    var val = document.getElementById('edit-val-' + ftID).value;
    var body = new URLSearchParams();
    body.set('id', ftID);
    body.set('value', val);
    API.put('/api/files/tag', body).then(function(data) {
        if (!data.ok) { showToast(data.msg || '更新失败', 'error'); return; }
        showToast('值已更新', 'success');
        loadFileDetailData();
    }).catch(function() { showToast('请求失败', 'error'); });
};

window.switchGroup = function(groupID) {
    currentGroupID = groupID;
    currentPage = 1;
    var groupInput = document.querySelector('input[name="group_id"]');
    if (groupInput) groupInput.value = groupID;
    loadFileDetailData();
};

window.showAddGroup = function() {
    var form = document.getElementById('add-group-form');
    if (form) form.style.display = 'block';
};

window.doAddGroup = function(form) {
    var fd = new FormData(form);
    API.post('/api/files/group', fd).then(function(data) {
        if (!data.ok) { showToast(data.msg, 'error'); return false; }
        showToast(data.msg, 'success');
        var addForm = document.getElementById('add-group-form');
        if (addForm) addForm.style.display = 'none';
        form.reset();
        loadFileDetailData();
    }).catch(function() { showToast('请求失败', 'error'); });
    return false;
};

window.doRemoveGroup = function(groupID, groupName) {
    showConfirm('确定删除标签组 ' + groupName + '？', function() {
        var body = new URLSearchParams();
        body.set('file_path', filePath);
        fetch('/api/files/group/' + groupID, {method:'DELETE', headers:{'Content-Type':'application/x-www-form-urlencoded'}, body:body})
            .then(function(r) { return r.json(); })
            .then(function(data) {
                if (!data.ok) { showToast(data.msg, 'error'); return; }
                if (currentGroupID === groupID) currentGroupID = 0;
                showToast(data.msg || '标签组已删除', 'success');
                loadFileDetailData();
            })
            .catch(function() { showToast('请求失败', 'error'); });
    });
};

/* =========================================
   FILE GROUPS PAGE
   ========================================= */

function initFileGroupsPage() {
    var rawPath = window.location.pathname.substring('/files/groups/'.length);
    try { filePath = decodeURI(rawPath); } catch(e) { filePath = rawPath; }

    var backLink = document.getElementById('groups-back-link');
    if (backLink) backLink.href = '/files/detail/' + encodePath(filePath);

    var filePathInput = document.getElementById('groups-file-path-input');
    if (filePathInput) filePathInput.value = filePath;

    loadFileGroups();
    setupGroupsAddForm();
}

function loadFileGroups() {
    var groupsContent = document.getElementById('groups-content');
    if (!groupsContent) return;

    var detailPromise = API.get('/api/files/detail/' + encodePath(filePath)).then(function(data) {
        if (data.ok && data.detail) {
            var fi = data.detail.file;
            var nameEl = document.getElementById('groups-file-name');
            if (nameEl) nameEl.textContent = fi.name;
            var pathEl = document.getElementById('groups-file-path');
            if (pathEl) pathEl.textContent = fi.path;
        }
    }).catch(function() { /* ignore */ });

    var groupsPromise = API.get('/api/files/groups/' + encodePath(filePath)).then(function(data) {
        if (data.ok && data.groups) {
            renderGroupsList(data.groups);
        } else {
            groupsContent.innerHTML = '<div class="empty"><div class="empty-icon">&#128203;</div><div class="empty-text">加载失败</div></div>';
        }
    }).catch(function() {
        groupsContent.innerHTML = '<div class="empty"><div class="empty-icon">&#128203;</div><div class="empty-text">加载失败</div></div>';
    });
}

function renderGroupsList(groups) {
    var container = document.getElementById('groups-content');
    if (!container) return;

    var countEl = document.getElementById('groups-count');
    if (countEl) countEl.textContent = groups.length;

    if (!groups || groups.length === 0) {
        container.innerHTML = '<div class="empty"><div class="empty-icon">&#128203;</div><div class="empty-text">暂无标签组</div></div>';
        return;
    }

    var html = '<table class="mt-md"><thead><tr><th>组名</th><th>标签数</th><th>类型</th><th style="width:240px">操作</th></tr></thead><tbody>';
    for (var i = 0; i < groups.length; i++) {
        var g = groups[i];
        var nameHtml = escHtml(g.name);
        if (g.is_default) nameHtml += '<span class="badge badge-red" style="margin-left:6px">默认</span>';
        var typeText = g.is_default ? '默认组' : '自定义组';
        html += '<tr id="g-' + g.id + '">' +
            '<td class="g-name">' + nameHtml + '</td>' +
            '<td>' + g.tag_count + '</td>' +
            '<td>' + typeText + '</td>' +
            '<td class="col-actions">' +
                '<button class="btn btn-default btn-sm" onclick="editGroupName(' + g.id + ',\'' + escJsStr(g.name) + '\')">重命名</button>';
        if (!g.is_default) {
            html += ' <button class="btn btn-danger btn-sm" onclick="removeGroup(' + g.id + ',\'' + escJsStr(g.name) + '\')">删除</button>';
        }
        html += ' <a href="/files/detail/' + encodePath(filePath) + '" class="btn btn-default btn-sm">查看标签</a>';
        html += '</td></tr>';
    }
    html += '</tbody></table>';
    container.innerHTML = html;
}

function setupGroupsAddForm() {
    var form = document.getElementById('add-group-form');
    if (!form) return;
    form.addEventListener('submit', function(e) {
        e.preventDefault();
        API.post('/api/files/group', new FormData(this)).then(function(data) {
            if (data.ok) {
                showToast(data.msg, 'success');
                loadFileGroups();
            } else {
                showToast(data.msg, 'error');
            }
        }).catch(function() { showToast('请求失败', 'error'); });
    });
}

window.removeGroup = function(id, name) {
    showConfirm('确定删除标签组 «' + name + '»？组内所有标签也会被移除。', function() {
        var body = new URLSearchParams();
        body.set('file_path', filePath);
        fetch('/api/files/group/' + id, {method:'DELETE', headers:{'Content-Type':'application/x-www-form-urlencoded'}, body:body})
            .then(function(r) { return r.json(); })
            .then(function(data) {
                if (data.ok) {
                    showToast(data.msg, 'success');
                    loadFileGroups();
                } else {
                    showToast(data.msg, 'error');
                }
            })
            .catch(function() { showToast('请求失败', 'error'); });
    });
};

window.editGroupName = function(groupID, curName) {
    var row = document.getElementById('g-' + groupID);
    if (!row) return;
    var cell = row.querySelector('.g-name');
    cell.innerHTML = '<input id="edit-gn-' + groupID + '" value="' + escHtml(curName) + '" style="width:140px">';
    var actions = row.querySelectorAll('td')[3];
    actions.innerHTML = '<button class="btn btn-primary btn-sm" onclick="saveGroupName(' + groupID + ')">保存</button> <button type="button" class="btn btn-default btn-sm" onclick="location.reload()">取消</button>';
};

window.saveGroupName = function(groupID) {
    var name = document.getElementById('edit-gn-' + groupID).value;
    var body = new URLSearchParams();
    body.set('file_path', filePath);
    body.set('group_name', name);
    API.put('/api/files/group/' + groupID, body).then(function(data) {
        if (data.ok) {
            showToast(data.msg, 'success');
            loadFileGroups();
        } else {
            showToast(data.msg, 'error');
        }
    }).catch(function() { showToast('请求失败', 'error'); });
};

/* =========================================
   SHARED HELPERS
   ========================================= */

function escHtml(s) {
    if (!s && s !== 0) return '';
    return String(s).replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;').replace(/'/g,'&#39;');
}

function escJsStr(s) {
    if (!s) return '';
    return String(s).replace(/\\/g,'\\\\').replace(/'/g,"\\'").replace(/"/g,'\\"').replace(/\n/g,'\\n').replace(/\r/g,'\\r');
}

function formatSize(bytes) {
    if (bytes === 0) return '0 B';
    var units = ['B', 'KB', 'MB', 'GB', 'TB'];
    var i = Math.floor(Math.log(bytes) / Math.log(1024));
    if (i >= units.length) i = units.length - 1;
    return (bytes / Math.pow(1024, i)).toFixed(i > 0 ? 1 : 0) + ' ' + units[i];
}

function iconForMime(mimeType) {
    if (!mimeType) return '📄';
    if (mimeType.indexOf('video/') === 0) return '🎬';
    if (mimeType.indexOf('image/') === 0) return '🖼';
    if (mimeType.indexOf('audio/') === 0) return '🎵';
    if (mimeType.indexOf('pdf') >= 0) return '📄';
    if (mimeType.indexOf('zip') >= 0 || mimeType.indexOf('tar') >= 0 || mimeType.indexOf('rar') >= 0 || mimeType.indexOf('gzip') >= 0) return '📦';
    return '📄';
}

function iconForName(name) {
    if (!name) return '📄';
    var parts = name.split('.');
    var ext = parts.length > 1 ? parts[parts.length - 1].toLowerCase() : '';
    if (['jpg','jpeg','png','gif','webp','bmp','svg'].indexOf(ext) >= 0) return '🖼';
    if (['mp4','webm','avi','mov','mkv','flv'].indexOf(ext) >= 0) return '🎬';
    if (['mp3','wav','flac','ogg','aac','wma'].indexOf(ext) >= 0) return '🎵';
    if (ext === 'pdf') return '📄';
    if (['zip','tar','gz','rar','7z'].indexOf(ext) >= 0) return '📦';
    return '📄';
}

function encodePath(path) {
    return path.split('/').map(function(s) { return encodeURIComponent(s); }).join('/');
}

})();
