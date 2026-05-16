/* ═══════════════════════════════════════════
   TagFile — File Detail: Groups, Tags, Tag Table
   ═══════════════════════════════════════════ */

function renderGroupsAndTags(d) {
    var container = document.getElementById('groups-and-tags');
    if (!container) return;

    if (!d.groups || d.groups.length === 0) {
        container.innerHTML = '<div class="empty"><div class="empty-icon">&#127991;</div><div class="empty-text">无标签组</div></div>';
        return;
    }

    var html = '';

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

    html += '<div id="tag-table" class="mt-md"></div>';

    container.innerHTML = html;

    renderTagTable(d.tags, d.pagination);
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

/* ── Global handlers for inline onclick ── */

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
