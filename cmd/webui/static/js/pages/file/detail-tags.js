/* ═══════════════════════════════════════════
   TagFile — File Detail: Groups, Tags, Tag Table
   ═══════════════════════════════════════════ */

/* ── Lightweight reload: only groups + tags, not the whole panel ── */
function reloadGroupsAndTags() {
    var url = '/api/files/detail/' + encodePath(filePath);
    url += '?group_id=' + (currentGroupID || 0) + '&page=' + (currentPage || 1);

    API.get(url).then(function(data) {
        if (!data.ok || !data.detail) return;
        var d = data.detail;
        if (currentGroupID === 0 && d.groups && d.groups.length > 0) {
            currentGroupID = d.groups[0].id;
        }
        lastDetailData = d;
        var groupInput = document.querySelector('input[name="group_id"]');
        if (groupInput) groupInput.value = currentGroupID;
        refreshFileInfoTags(d);
        renderGroupsAndTags(d);
    }).catch(function() {
        var container = document.getElementById('groups-and-tags');
        if (container) {
            while (container.firstChild) container.removeChild(container.firstChild);
            container.appendChild(makeEmptyState('&#9888;', '加载失败'));
        }
    });
}

function renderGroupsAndTags(d) {
    var container = document.getElementById('groups-and-tags');
    if (!container) return;

    if (!d.groups || d.groups.length === 0) {
        while (container.firstChild) container.removeChild(container.firstChild);
        container.appendChild(makeEmptyState('&#127991;', '无标签组'));
        return;
    }

    var frag = document.createDocumentFragment();

    /* Group tabs */
    var tabs = document.createElement('div');
    tabs.className = 'tabs';
    tabs.id = 'group-tabs';

    for (var i = 0; i < d.groups.length; i++) {
        var g = d.groups[i];
        var tab = document.createElement('a');
        tab.href = '#';
        tab.className = 'tab' + (g.id === currentGroupID ? ' active' : '');
        tab.id = 'group-tab-' + g.id;
        tab.textContent = g.name;

        tab.addEventListener('click', function(groupID) {
            return function(e) {
                e.preventDefault();
                switchGroup(groupID);
            };
        }(g.id));

        if (!g.is_default) {
            var close = document.createElement('span');
            close.className = 'tab-close';
            close.textContent = '×';
            close.title = '删除标签组';
            close.addEventListener('click', function(groupID, groupName) {
                return function(e) {
                    e.stopPropagation();
                    doRemoveGroup(groupID, groupName);
                };
            }(g.id, g.name));
            tab.appendChild(close);
        }

        tabs.appendChild(tab);
    }

    var addTab = document.createElement('span');
    addTab.className = 'tab-add';
    addTab.textContent = '+';
    addTab.title = '添加标签组';
    addTab.addEventListener('click', showAddGroup);
    tabs.appendChild(addTab);

    frag.appendChild(tabs);

    /* Add tag form */
    var tagForm = document.createElement('form');
    tagForm.className = 'form-row';

    var tfFileInput = document.createElement('input');
    tfFileInput.type = 'hidden';
    tfFileInput.name = 'file_path';
    tfFileInput.value = d.file.path;
    tagForm.appendChild(tfFileInput);

    var tfGroupInput = document.createElement('input');
    tfGroupInput.type = 'hidden';
    tfGroupInput.name = 'group_id';
    tfGroupInput.value = currentGroupID;
    tagForm.appendChild(tfGroupInput);

    var tfGroup = document.createElement('div');
    tfGroup.className = 'form-group';
    tfGroup.style.position = 'relative';
    var tfTagInput = document.createElement('input');
    tfTagInput.id = 'add-tag-input';
    tfTagInput.name = 'tag_name';
    tfTagInput.required = true;
    tfTagInput.autocomplete = 'off';
    tfTagInput.placeholder = '搜索标签...';
    tfTagInput.style.width = '170px';
    tfGroup.appendChild(tfTagInput);

    var tfDropdown = document.createElement('div');
    tfDropdown.id = 'tag-dropdown';
    tfDropdown.className = 'dropdown';
    tfGroup.appendChild(tfDropdown);
    tagForm.appendChild(tfGroup);

    var tfValueGroup = document.createElement('div');
    tfValueGroup.className = 'form-group';
    tfValueGroup.id = 'tag_value_group';
    tfValueGroup.style.display = 'none';
    var tfValueInput = document.createElement('input');
    tfValueInput.name = 'tag_value';
    tfValueInput.placeholder = '值（可选）';
    tfValueInput.style.width = '110px';
    tfValueGroup.appendChild(tfValueInput);
    tagForm.appendChild(tfValueGroup);

    var tfSubmit = document.createElement('button');
    tfSubmit.type = 'submit';
    tfSubmit.className = 'btn btn-primary';
    tfSubmit.textContent = '添加';
    tagForm.appendChild(tfSubmit);

    tagForm.addEventListener('submit', function(e) {
        e.preventDefault();
        doTag(this);
    });
    frag.appendChild(tagForm);

    /* Tag table container */
    var tagTable = document.createElement('div');
    tagTable.id = 'tag-table';
    tagTable.className = 'mt-md';
    frag.appendChild(tagTable);

    container.innerHTML = '';
    container.appendChild(frag);

    renderTagTable(d.tags, d.pagination);
    setupTagSearchOnDetail();
}

function renderTagTable(tags, pg) {
    var tableContainer = document.getElementById('tag-table');
    if (!tableContainer) return;
    while (tableContainer.firstChild) tableContainer.removeChild(tableContainer.firstChild);

    if (!tags || tags.length === 0) {
        tableContainer.appendChild(makeEmptyState('&#127991;', '暂无标签', '在上方搜索并添加标签'));
        return;
    }

    var tbody = createEl('tbody');
    for (var i = 0; i < tags.length; i++) {
        var t = tags[i];
        var delBtn = makeActionBtn('删除', 'btn btn-danger btn-sm', function(id, tid, val) { return function() { removeFileTag(id, tid, val); }; }(t.id, t.tag_id, t.value || ''));
        var valCell = createEl('td', {className: 'ft-value', 'data-value': t.value || ''}, t.value || '—');
        var tr = createEl('tr', {id: 'ft-' + t.id},
            createEl('td', {'data-tag-name': t.tag_name, 'data-tag-id': String(t.tag_id), style: 'font-weight:500'}, t.tag_path),
            valCell,
            createEl('td', {className: 'text-muted'}, t.category),
            createEl('td', {className: 'col-actions'}, delBtn)
        );
        tbody.appendChild(tr);
    }
    var table = createEl('table', null,
        createEl('thead', null, createEl('tr', null,
            createEl('th', null, '标签名'),
            createEl('th', null, '值'),
            createEl('th', null, '分类'),
            createEl('th', {style: 'width:140px'}, '操作')
        )),
        tbody
    );
    tableContainer.appendChild(table);

    if (pg && pg.total_pages > 1) {
        tableContainer.appendChild(makePagination(pg, loadPage, {prevText: '« 前页', nextText: '后页 »', showDisabled: true}));
    }

    /* Wire up makeEditable on ft-value cells */
    var valCells = tableContainer.querySelectorAll('.ft-value');
    valCells.forEach(function(cell) {
        var tr = cell.closest('tr');
        if (!tr) return;
        var ftID = parseInt(tr.id.replace('ft-', ''));
        if (isNaN(ftID)) return;
        makeEditable(cell, {
            emptyText: '—',
            onSave: async function(newVal) {
                await saveFileTagValue(ftID, newVal);
            }
        });
    });
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
            var child;
            while ((child = group.firstChild)) { group.removeChild(child); }
            if (vals.length > 0) {
                var sel = document.createElement('select');
                sel.name = 'tag_value';
                var opt0 = document.createElement('option');
                opt0.value = '';
                opt0.textContent = '可选值';
                sel.appendChild(opt0);
                for (var i = 0; i < vals.length; i++) {
                    var opt = document.createElement('option');
                    opt.value = vals[i].value;
                    opt.textContent = vals[i].value;
                    sel.appendChild(opt);
                }
                group.appendChild(sel);
            } else {
                var inp = document.createElement('input');
                inp.name = 'tag_value';
                inp.placeholder = '值（可选）';
                inp.style.width = '110px';
                group.appendChild(inp);
            }
        })
        .catch(function() {
            group.style.display = 'inline-block';
            var child;
            while ((child = group.firstChild)) { group.removeChild(child); }
            var inp = document.createElement('input');
            inp.name = 'tag_value';
            inp.placeholder = '值（可选）';
            inp.style.width = '110px';
            group.appendChild(inp);
        });
}

/* ── Global handlers ── */

window.loadPage = function(page) {
    currentPage = page;
    reloadGroupsAndTags();
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
        if (valueGroup) {
            var child;
            while ((child = valueGroup.firstChild)) { valueGroup.removeChild(child); }
            var inp = document.createElement('input');
            inp.name = 'tag_value';
            inp.placeholder = '值（可选）';
            inp.style.width = '110px';
            valueGroup.appendChild(inp);
            valueGroup.style.display = 'none';
        }
        reloadGroupsAndTags();
    }).catch(function() { showToast('请求失败', 'error'); });
    return false;
};

window.removeFileTag = function(ftID, tagID, tagValue) {
    showConfirm('确定删除该标签？', function() {
        var params = new URLSearchParams();
        params.set('file_path', filePath);
        params.set('tag_id', tagID);
        params.set('tag_value', tagValue || '');
        params.set('group_id', currentGroupID);
        fetch('/api/files/tag?' + params.toString(), {method:'DELETE'})
            .then(function(r) { return r.json(); })
            .then(function(data) {
                if (!data.ok) { showToast(data.msg, 'error'); return; }
                showToast(data.msg || '标签已删除', 'info');
                reloadGroupsAndTags();
            })
            .catch(function() { showToast('请求失败', 'error'); });
    });
};

window.editFileTagValue = function(ftID) {
    var row = document.getElementById('ft-' + ftID);
    if (!row) return;
    var cell = row.querySelector('.ft-value');
    if (cell) cell.click();
};

async function saveFileTagValue(ftID, val) {
    var body = new URLSearchParams();
    body.set('id', ftID);
    body.set('value', val);
    var data = await API.put('/api/files/tag', body);
    if (!data.ok) { showToast(data.msg || '更新失败', 'error'); throw new Error(data.msg); }
    showToast('值已更新', 'success');
    reloadGroupsAndTags();
}

window.switchGroup = function(groupID) {
    currentGroupID = groupID;
    currentPage = 1;
    reloadGroupsAndTags();
};

/* ── Add Group via Modal ── */
window.showAddGroup = function() {
    Modal.show({
        title: '创建标签组',
        fields: [
            { label: '标签组名称', name: 'group_name', required: true, placeholder: '例如: 元数据' }
        ],
        saveLabel: '创建',
        onSave: async function(values) {
            var data = await API.post('/api/files/group', {
                file_path: filePath,
                group_name: values.group_name.trim()
            });
            if (!data.ok) { showToast(data.msg, 'error'); return false; }
            showToast(data.msg, 'success');
            reloadGroupsAndTags();
        }
    });
};

window.doRemoveGroup = function(groupID, groupName) {
    showConfirm('确定删除标签组 ' + groupName + '？', function() {
        var params = new URLSearchParams();
        params.set('file_path', filePath);
        fetch('/api/files/group/' + groupID + '?' + params.toString(), {method:'DELETE'})
            .then(function(r) { return r.json(); })
            .then(function(data) {
                if (!data.ok) { showToast(data.msg, 'error'); return; }
                if (currentGroupID === groupID) currentGroupID = 0;
                showToast(data.msg || '标签组已删除', 'success');
                reloadGroupsAndTags();
            })
            .catch(function() { showToast('请求失败', 'error'); });
    });
};

/* Legacy */
window.doAddGroup = function(form) {
    // Removed — now handled by Modal via showAddGroup()
};
