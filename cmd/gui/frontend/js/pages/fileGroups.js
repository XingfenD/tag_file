// fileGroups.js - Tag group management page
var _fgFilePath = '';

function renderFileGroups(data) {
    if (!data.file && !data.error) return;

    var file = data.file || {};
    var groups = data.groups || [];
    _fgFilePath = file.filePath || '';

    var html = '<h2><a href="#" onclick="App.navigate(\'files\');return false" style="text-decoration:none;color:#e94560">&larr; 返回文件管理</a></h2>';
    html += '<div class="card"><h3>标签组管理: ' + escapeHtml(file.fileName || '') + '</h3>' +
        '<p style="color:#888;font-size:13px;margin-top:4px">路径: ' + escapeHtml(file.filePath || '') + '</p></div>';

    html += '<div class="card"><h3>添加标签组</h3>' +
        '<form onsubmit="return submitAddGroup(this)" class="row">' +
        '<div class="form-group"><label>组名</label><input name="group_name" required placeholder="例如: 版本 v2、临时标记" style="width:200px"></div>' +
        '<button type="submit" class="btn-primary">添加</button></form></div>';

    html += '<div class="card"><h3>标签组列表 (' + groups.length + ' 个)</h3>';
    if (groups.length > 0) {
        html += '<table><thead><tr><th>组名</th><th>标签数</th><th>类型</th><th>操作</th></tr></thead><tbody>';
        groups.forEach(function(g) {
            html += '<tr id="g-' + g.group.groupID + '">' +
                '<td class="g-name">' + escapeHtml(g.group.groupName);
            if (g.group.isDefault) html += '<span class="tag-badge" style="background:#d4edda;color:#155724;margin-left:4px">默认</span>';
            html += '</td><td>' + (g.tagCount || 0) + '</td>' +
                '<td>' + (g.group.isDefault ? '默认组' : '自定义组') + '</td><td>' +
                '<button class="btn-default btn-sm" onclick="editGroupName(' + g.group.groupID + ',\'' + escapeJs(g.group.groupName) + '\')">重命名</button> ';
            if (!g.group.isDefault) {
                html += '<button class="btn-danger btn-sm" onclick="deleteGroup(' + g.group.groupID + ',\'' + escapeJs(g.group.groupName) + '\')">删除</button> ';
            }
            html += '</td></tr>';
        });
        html += '</tbody></table>';
    } else {
        html += '<p class="empty">暂无标签组</p>';
    }
    html += '</div>';

    document.getElementById('page-content').innerHTML = html;
}

function refreshFileGroups() {
    API.getFileGroups(_fgFilePath).then(function(d) {
        if (d.error) { App.flash(d.error, 'error'); return; }
        renderFileGroups(d);
    });
}

function submitAddGroup(form) {
    API.addGroup(_fgFilePath, form.group_name.value).then(function(r) {
        if (r.error) { App.flash(r.error, 'error'); return; }
        App.flash(r.message);
        refreshFileGroups();
    });
    return false;
}

function editGroupName(groupID, curName) {
    var row = document.getElementById('g-' + groupID);
    if (!row) return;
    row.querySelector('.g-name').innerHTML = '<input id="edit-gn-' + groupID + '" value="' + curName + '" style="width:140px">';
    row.querySelectorAll('td')[3].innerHTML = '<button class="btn-primary btn-sm" onclick="saveGroupName(' + groupID + ')">保存</button> ' +
        '<button class="btn-default btn-sm" onclick="refreshFileGroups()">取消</button>';
}

function saveGroupName(groupID) {
    var input = document.getElementById('edit-gn-' + groupID);
    if (!input) return;
    API.renameGroup(_fgFilePath, groupID, input.value).then(function(r) {
        if (r.error) { App.flash(r.error, 'error'); return; }
        App.flash(r.message);
        refreshFileGroups();
    });
}

function deleteGroup(groupID, groupName) {
    if (!confirm('确定删除标签组 «' + groupName + '»？组内所有标签也会被移除。')) return;
    API.removeGroup(_fgFilePath, groupID).then(function(r) {
        if (r.error) { App.flash(r.error, 'error'); return; }
        App.flash(r.message);
        refreshFileGroups();
    });
}
