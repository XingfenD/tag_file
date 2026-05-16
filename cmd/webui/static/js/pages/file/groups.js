/* ═══════════════════════════════════════════
   TagFile — File Groups Page
   ═══════════════════════════════════════════ */

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

    API.get('/api/files/detail/' + encodePath(filePath)).then(function(data) {
        if (data.ok && data.detail) {
            var fi = data.detail.file;
            var nameEl = document.getElementById('groups-file-name');
            if (nameEl) nameEl.textContent = fi.name;
            var pathEl = document.getElementById('groups-file-path');
            if (pathEl) pathEl.textContent = fi.path;
        }
    }).catch(function() { /* ignore */ });

    API.get('/api/files/groups/' + encodePath(filePath)).then(function(data) {
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
