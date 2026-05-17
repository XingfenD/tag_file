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
            while (groupsContent.firstChild) groupsContent.removeChild(groupsContent.firstChild);
            groupsContent.appendChild(makeEmptyState('&#128203;', '加载失败'));
        }
    }).catch(function() {
        while (groupsContent.firstChild) groupsContent.removeChild(groupsContent.firstChild);
        groupsContent.appendChild(makeEmptyState('&#128203;', '加载失败'));
    });
}

function renderGroupsList(groups) {
    var container = document.getElementById('groups-content');
    if (!container) return;

    var countEl = document.getElementById('groups-count');
    if (countEl) countEl.textContent = groups.length;

    while (container.firstChild) container.removeChild(container.firstChild);

    if (!groups || groups.length === 0) {
        container.appendChild(makeEmptyState('&#128203;', '暂无标签组'));
        return;
    }

    var tbody = createEl('tbody');
    for (var i = 0; i < groups.length; i++) {
        var g = groups[i];
        var typeText = g.is_default ? '默认组' : '自定义组';
        var nameCell = createEl('td', {className: 'g-name', 'data-value': g.name});
        nameCell.appendChild(document.createTextNode(g.name));
        if (g.is_default) nameCell.appendChild(createEl('span', {className: 'badge badge-red', style: 'margin-left:6px'}, '默认'));

        var actionsCell = createEl('td', {className: 'col-actions'});
        if (!g.is_default) {
            actionsCell.appendChild(makeActionBtn('删除', 'btn btn-danger btn-sm', function(id, n) { return function() { removeGroup(id, n); }; }(g.id, g.name)));
        }

        tbody.appendChild(createEl('tr', {id: 'g-' + g.id},
            nameCell,
            createEl('td', null, String(g.tag_count)),
            createEl('td', null, typeText),
            actionsCell
        ));
    }
    container.appendChild(createEl('table', {className: 'mt-md'},
        createEl('thead', null, createEl('tr', null,
            createEl('th', null, '组名'),
            createEl('th', null, '标签数'),
            createEl('th', null, '类型'),
            createEl('th', {style: 'width:80px'}, '操作')
        )),
        tbody
    ));

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
    showConfirm('确定删除标签组 «' + name + '»？组内所有标签也会被移除。', function() {
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


