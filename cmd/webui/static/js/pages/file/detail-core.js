/* ═══════════════════════════════════════════
   TagFile — File Detail: Info, Sidebar, Media, Rename
   ═══════════════════════════════════════════ */

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
