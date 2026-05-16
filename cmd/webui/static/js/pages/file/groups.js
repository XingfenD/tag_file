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
            '<td class="g-name" data-value="' + escHtml(g.name) + '">' + nameHtml + '</td>' +
            '<td>' + g.tag_count + '</td>' +
            '<td>' + typeText + '</td>' +
            '<td class="col-actions">' +
                '<button class="btn btn-default btn-sm" onclick="editGroupNameModal(' + g.id + ',\'' + escJsStr(g.name) + '\')">重命名</button>';
        if (!g.is_default) {
            html += ' <button class="btn btn-danger btn-sm" onclick="removeGroup(' + g.id + ',\'' + escJsStr(g.name) + '\')">删除</button>';
        }
        html += ' <a href="/files/detail/' + encodePath(filePath) + '" class="btn btn-default btn-sm">查看标签</a>';
        html += '</td></tr>';
    }
    html += '</tbody></table>';
    container.innerHTML = html;

    /* Wire up makeEditable on g-name cells (replaces editGroupName innerHTML swap) */
    var nameCells = container.querySelectorAll('.g-name');
    nameCells.forEach(function(cell) {
        var tr = cell.closest('tr');
        if (!tr) return;
        var groupID = parseInt(tr.id.replace('g-', ''));
        if (isNaN(groupID)) return;
        makeEditable(cell, {
            onSave: async function(newVal) {
                var body = new URLSearchParams();
                body.set('file_path', filePath);
                body.set('group_name', newVal);
                var data = await API.put('/api/files/group/' + groupID, body);
                if (!data.ok) { showToast(data.msg, 'error'); throw new Error(data.msg); }
                showToast(data.msg, 'success');
                loadFileGroups();
            }
        });
    });
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
    showConfirm('确定删除标签组 &laquo;' + name + '&raquo;？组内所有标签也会被移除。', function() {
        var params = new URLSearchParams();
        params.set('file_path', filePath);
        fetch('/api/files/group/' + id + '?' + params.toString(), {method:'DELETE'})
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

window.editGroupNameModal = function(groupID, curName) {
    Modal.show({
        title: '重命名标签组',
        fields: [
            { label: '标签组名', name: 'group_name', value: curName, required: true }
        ],
        onSave: async function(values) {
            var newName = values.group_name.trim();
            if (!newName) { showToast('名称不能为空', 'error'); return false; }
            var body = new URLSearchParams();
            body.set('file_path', filePath);
            body.set('group_name', newName);
            var data = await API.put('/api/files/group/' + groupID, body);
            if (!data.ok) { showToast(data.msg, 'error'); return false; }
            showToast(data.msg, 'success');
            loadFileGroups();
        }
    });
};

/* ── Legacy ── */
window.editGroupName = function(groupID, curName) {
    editGroupNameModal(groupID, curName);
};

window.saveGroupName = function(groupID) {
    // Removed — now handled by Modal
};
