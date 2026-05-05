// tag.js - Tag management (flat + tree views)
var tagAllTags = [];
var tagCurrentView = 'tree';
var tagCurrentCat = '';
var tagCurrentQ = '';
var tagCurrentPage = 1;

function renderTags(data) {
    if (data.view === 'tree') {
        renderTagsTree(data);
    } else {
        renderTagsFlat(data);
    }
    tagAllTags = data.allTags || [];
    tagCurrentView = data.view || 'tree';
    tagCurrentCat = data.categoryID || '';

    // Load tag values
    setTimeout(initTagValues, 200);
}

function renderTagsCommon(data) {
    var categories = data.categories || [];
    var html = '<h2>标签管理</h2>';

    // Add form
    html += '<div class="card"><h3>添加标签</h3>' +
        '<form onsubmit="return submitAddTag(this)" class="row">' +
        '<div class="form-group"><label>所属分类</label><select name="category_id" required onchange="loadParentTagsForAdd(this.value)">' +
        '<option value="">-- 选择分类 --</option>';
    categories.forEach(function(c) {
        html += '<option value="' + c.CategoryID + '"' + (String(c.CategoryID) === tagCurrentCat ? ' selected' : '') + '>' + escapeHtml(c.CategoryName) + '</option>';
    });
    html += '</select></div>' +
        '<div class="form-group tag-search-wrap"><label>父标签（可选）</label>' +
        '<input name="parent_tag_name" id="parent_tag_select" placeholder="搜索父标签..." autocomplete="off">' +
        '<div id="parent-tag-dropdown" class="tag-search-dropdown"></div></div>' +
        '<div class="form-group"><label>标签名称</label><input name="tag_name" required placeholder="例如: 作者"></div>' +
        '<div class="form-group"><label>描述</label><input name="description" placeholder="可选描述"></div>' +
        '<button type="submit" class="btn-primary">添加</button></form></div>';

    // View toggle bar
    html += '<div class="card"><div style="display:flex;align-items:center;gap:16px;margin-bottom:16px">' +
        '<h3 style="margin:0">标签列表</h3>' +
        '<div style="display:flex;align-items:center;gap:8px;font-size:13px">' +
        '<a href="#" onclick="switchTagView(\'flat\');return false" class="btn-default btn-sm" style="text-decoration:none;' + (tagCurrentView !== 'tree' ? 'background:#e94560;color:#fff' : '') + '">平铺</a>' +
        '<a href="#" onclick="switchTagView(\'tree\');return false" class="btn-default btn-sm" style="text-decoration:none;' + (tagCurrentView === 'tree' ? 'background:#e94560;color:#fff' : '') + '">树形</a>' +
        '</div>' +
        '<div class="row" style="flex:1;justify-content:flex-end">' +
        '<select id="tag-cat-filter" style="width:130px" onchange="filterTagCat(this.value)">' +
        '<option value="">-- 所有分类 --</option>';
    categories.forEach(function(c) {
        html += '<option value="' + c.CategoryID + '"' + (String(c.CategoryID) === tagCurrentCat ? ' selected' : '') + '>' + escapeHtml(c.CategoryName) + '</option>';
    });
    html += '</select>' +
        '<input id="tag-search-q" placeholder="搜索标签..." value="' + escapeHtml(tagCurrentQ) + '" style="width:200px">' +
        '<button class="btn-default btn-sm" onclick="searchTags()">搜索</button>' +
        (tagCurrentQ ? '<button class="btn-default btn-sm" onclick="clearTagSearch()">清除</button>' : '') +
        '</div></div><div id="tag-list-wrap">';

    return html;
}

function renderTagsTree(data) {
    var html = renderTagsCommon(data);
    var tree = data.tagTree || [];

    if (tree.length > 0) {
        html += '<ul class="tag-tree-root tree-root">';
        tree.forEach(function(node) { html += renderTagTreeNode(node); });
        html += '</ul>';
    } else {
        html += '<p class="empty">暂无标签</p>';
    }
    html += '</div></div>';
    document.getElementById('page-content').innerHTML = html;

    if (tagAllTags.length > 0) {
        initParentTagSearch();
        initTagValues();
    }
}

function renderTagTreeNode(node) {
    var t = node.Tag;
    var html = '<li class="tag-tree-node">' +
        '<div class="tag-tree-row" style="display:flex;align-items:center;gap:8px;padding:4px 0;font-size:13px">' +
        '<span style="font-size:15px">🏷</span>' +
        '<span class="tag-tree-name" style="font-weight:500;color:#1a1a2e">' + escapeHtml(t.TagName) + '</span>' +
        '<span style="color:#888;font-size:12px">(' + escapeHtml(t.Category.CategoryName) + ')</span>' +
        '<span style="color:#aaa;font-size:11px">#' + t.TagID + '</span>' +
        '<span class="tag-tree-values" data-tag-name="' + escapeHtml(t.TagName) + '" style="display:flex;gap:2px;flex-wrap:wrap"></span>' +
        '<a href="#" onclick="editTagTree(this,' + t.TagID + ',\'' + escapeJs(t.TagName) + '\',' + t.Category.CategoryID + ');return false" class="btn-default btn-xs" style="text-decoration:none">编辑</a>' +
        '<a href="#" onclick="quickAddChild(this,\'' + escapeJs(t.TagName) + '\',' + t.Category.CategoryID + ');return false" class="btn-primary btn-xs" style="text-decoration:none">+</a>' +
        '<button class="btn-danger btn-xs" onclick="deleteTag(' + t.TagID + ',\'' + escapeJs(t.TagName) + '\')">删除</button>' +
        '</div>';
    if (node.Children && node.Children.length > 0) {
        html += '<ul class="tree-children" style="padding-left:24px">';
        node.Children.forEach(function(c) { html += renderTagTreeNode(c); });
        html += '</ul>';
    }
    html += '</li>';
    return html;
}

function renderTagsFlat(data) {
    var html = renderTagsCommon(data);
    var tags = data.tags || [];
    var pg = data.pagination || {};

    if (tags.length > 0) {
        html += '<table><thead><tr><th>ID</th><th>标签名</th><th>所属分类</th><th>父标签</th><th>预设值</th><th>描述</th><th>操作</th></tr></thead><tbody>';
        tags.forEach(function(t) {
            var parentPath = '-';
            if (t.ParentID) {
                for (var i = 0; i < tagAllTags.length; i++) {
                    if (tagAllTags[i].id === t.ParentID) { parentPath = tagAllTags[i].path; break; }
                }
            }
            html += '<tr id="tag-' + t.TagID + '">' +
                '<td>' + t.TagID + '</td>' +
                '<td class="tag-name">' + escapeHtml(t.TagName) + '</td>' +
                '<td data-category-id="' + t.Category.CategoryID + '">' + escapeHtml(t.Category.CategoryName) + '</td>' +
                '<td class="tag-parent">' + escapeHtml(parentPath) + '</td>' +
                '<td class="tag-values" data-tag-name="' + escapeHtml(t.TagName) + '"><span class="tag-values-loading" style="color:#aaa">...</span></td>' +
                '<td class="tag-desc">' + escapeHtml(t.Description) + '</td>' +
                '<td class="tag-actions">' +
                '<button class="btn-default btn-sm" onclick="editTag(' + t.TagID + ',' + t.Category.CategoryID + ',\'' + escapeJs(t.TagName) + '\',\'' + escapeJs(t.Description) + '\',\'' + escapeJs(parentPath) + '\')">编辑</button> ' +
                '<button class="btn-danger btn-sm" onclick="deleteTag(' + t.TagID + ',\'' + escapeJs(t.TagName) + '\')">删除</button>' +
                '</td></tr>';
        });
        html += '</tbody></table>' + renderPagination(pg, 'loadTagsPage');
    } else {
        html += '<p class="empty">暂无标签</p>';
    }
    html += '</div></div>';
    document.getElementById('page-content').innerHTML = html;

    initParentTagSearch();
    initTagValues();
}

// --- Actions ---

function switchTagView(view) {
    tagCurrentView = view;
    if (view === 'tree') tagCurrentQ = '';
    loadTags(view, tagCurrentCat, tagCurrentQ, 1);
}

function filterTagCat(catID) {
    tagCurrentCat = catID;
    loadTags(tagCurrentView, catID, tagCurrentQ, 1);
}

function searchTags() {
    tagCurrentQ = document.getElementById('tag-search-q').value;
    tagCurrentView = 'flat';
    loadTags('flat', tagCurrentCat, tagCurrentQ, 1);
}

function clearTagSearch() {
    tagCurrentQ = '';
    document.getElementById('tag-search-q').value = '';
    loadTags(tagCurrentView, tagCurrentCat, '', 1);
}

function loadTagsPage(page) {
    tagCurrentPage = page;
    loadTags(tagCurrentView, tagCurrentCat, tagCurrentQ, page);
}

function loadTags(view, catID, q, page) {
    API.listTags(view, catID, q, page || 1, 10).then(function(d) {
        if (d.error) { App.flash(d.error, 'error'); return; }
        renderTags(d);
    });
}

function submitAddTag(form) {
    var catID = parseInt(form.category_id.value) || 0;
    if (!catID) { App.flash('请选择分类', 'error'); return false; }
    var name = form.tag_name.value;
    var desc = form.description.value;
    var parent = form.parent_tag_name.value;
    API.addTag(catID, name, desc, parent).then(function(r) {
        if (r.error) { App.flash(r.error, 'error'); return; }
        App.flash(r.message);
        loadTags(tagCurrentView, tagCurrentCat, tagCurrentQ, tagCurrentPage);
    });
    return false;
}

function deleteTag(id, name) {
    if (!confirm('确定删除标签 «' + name + '»？')) return;
    API.removeTag(id).then(function(r) {
        if (r.error) { App.flash(r.error, 'error'); return; }
        App.flash(r.message);
        loadTags(tagCurrentView, tagCurrentCat, tagCurrentQ, tagCurrentPage);
    });
}

function editTag(tagId, categoryId, name, desc, parentName) {
    var row = document.getElementById('tag-' + tagId);
    var cells = row.querySelectorAll('td');
    cells[1].innerHTML = '<input id="edit_name_' + tagId + '" value="' + escapeHtml(name) + '" required style="width:100px">';
    cells[3].innerHTML = '<div class="tag-search-wrap"><input id="edit_parent_' + tagId + '" placeholder="搜索父标签..." autocomplete="off" style="width:120px" value="' + escapeHtml(parentName) + '"><div id="edit_parent_dropdown_' + tagId + '" class="tag-search-dropdown"></div></div>';
    cells[5].innerHTML = '<input id="edit_desc_' + tagId + '" value="' + escapeHtml(desc || '') + '" style="width:120px">';
    cells[6].innerHTML = '<button class="btn-primary btn-sm" onclick="saveTag(' + tagId + ')">保存</button> ' +
        '<button class="btn-default btn-sm" onclick="loadTagsPage(' + tagCurrentPage + ')">取消</button>';
    loadParentTags(categoryId, 'edit_parent_' + tagId, 'edit_parent_dropdown_' + tagId, parentName, tagId);
}

function saveTag(tagId) {
    var name = document.getElementById('edit_name_' + tagId).value;
    var parent = document.getElementById('edit_parent_' + tagId).value;
    var desc = document.getElementById('edit_desc_' + tagId).value;
    API.updateTag(tagId, name, desc, parent).then(function(r) {
        if (r.error) { App.flash(r.error, 'error'); return; }
        App.flash(r.message);
        loadTags(tagCurrentView, tagCurrentCat, tagCurrentQ, tagCurrentPage);
    });
}

// --- Tree view actions ---

function editTagTree(btn, tagID, tagName, categoryID) {
    var row = btn.closest('.tag-tree-row');
    var nameEl = row.querySelector('.tag-tree-name');
    if (document.getElementById('edit-tree-name-' + tagID)) return;
    nameEl.style.display = 'none';
    var input = document.createElement('input');
    input.id = 'edit-tree-name-' + tagID;
    input.value = tagName;
    input.style.cssText = 'width:100px;height:24px;font-size:12px;padding:2px 6px';
    btn.insertAdjacentElement('beforebegin', input);
    btn.textContent = '保存';
    btn.onclick = function() { saveTagTree(tagID); return false; };
    var cancel = document.createElement('a');
    cancel.href = '#';
    cancel.className = 'btn-default btn-xs';
    cancel.style.textDecoration = 'none';
    cancel.textContent = '取消';
    cancel.onclick = function() { loadTags('tree', tagCurrentCat, '', tagCurrentPage); return false; };
    btn.insertAdjacentElement('afterend', cancel);
    input.focus();
}

function saveTagTree(tagID) {
    var input = document.getElementById('edit-tree-name-' + tagID);
    if (!input || !input.value.trim()) return;
    API.updateTag(tagID, input.value.trim(), '', '').then(function(r) {
        if (r.error) { App.flash(r.error, 'error'); return; }
        App.flash(r.message);
        loadTags('tree', tagCurrentCat, '', tagCurrentPage);
    });
}

function quickAddChild(btn, parentName, categoryID) {
    var existing = document.getElementById('quick-add-input');
    if (existing) {
        existing.parentElement.remove();
    }
    var span = document.createElement('span');
    span.style.cssText = 'display:inline-flex;gap:4px;align-items:center';
    span.innerHTML = '<input id="quick-add-input" placeholder="子标签名" style="width:100px;height:24px;font-size:12px;padding:2px 6px">' +
        '<button onclick="saveQuickChild(\'' + escapeJs(parentName) + '\',' + categoryID + ')" class="btn-primary btn-xs">保存</button>' +
        '<button onclick="this.parentElement.remove();return false" class="btn-default btn-xs">取消</button>';
    btn.parentNode.insertBefore(span, btn.nextSibling);
    setTimeout(function() { var inp = document.getElementById('quick-add-input'); if (inp) inp.focus(); }, 50);
}

function saveQuickChild(parentName, categoryID) {
    var input = document.getElementById('quick-add-input');
    if (!input || !input.value.trim()) return;
    API.addTag(categoryID, input.value.trim(), '', parentName).then(function(r) {
        if (r.error) { App.flash(r.error, 'error'); return; }
        App.flash(r.message);
        loadTags('tree', tagCurrentCat, '', tagCurrentPage);
    });
}

// --- Tag values ---

function initTagValues() {
    document.querySelectorAll('.tag-values, .tag-tree-values').forEach(function(cell) {
        var tagName = cell.getAttribute('data-tag-name');
        if (!tagName) return;
        API.getTagValues(tagName).then(function(vals) {
            var html = '';
            (vals || []).forEach(function(v) {
                html += '<span class="tag-badge" style="margin:1px;cursor:pointer" title="点击删除" onclick="removeTagValue(\'' + escapeJs(tagName) + '\',\'' + escapeJs(v.value) + '\')">' + escapeHtml(v.value) + ' ×</span>';
            });
            if (!cell.classList.contains('tag-tree-values')) {
                html += ' <a href="#" onclick="addTagValueDialog(\'' + escapeJs(tagName) + '\');return false" class="btn-default btn-xs" style="text-decoration:none">+</a>';
            }
            cell.innerHTML = html || (!cell.classList.contains('tag-tree-values') ? '<a href="#" onclick="addTagValueDialog(\'' + escapeJs(tagName) + '\');return false" class="btn-default btn-xs" style="text-decoration:none">+</a>' : '');
        });
    });
}

function addTagValueDialog(tagName) {
    var val = prompt('输入新值:');
    if (!val) return;
    API.addTagValue(tagName, val).then(function(r) {
        if (r.error) { App.flash(r.error, 'error'); return; }
        App.flash(r.message);
        loadTags(tagCurrentView, tagCurrentCat, tagCurrentQ, tagCurrentPage);
    });
}

function removeTagValue(tagName, val) {
    if (!confirm('确定删除值 «' + val + '»？')) return;
    API.removeTagValue(tagName, val).then(function(r) {
        if (r.error) { App.flash(r.error, 'error'); return; }
        App.flash(r.message);
        loadTags(tagCurrentView, tagCurrentCat, tagCurrentQ, tagCurrentPage);
    });
}

// --- Parent tag search ---

function loadParentTags(categoryID, inputId, dropdownId, selectedName, tagId) {
    var ctrl = setupTagSearch(inputId, dropdownId);
    if (!categoryID) { ctrl.setTags(tagAllTags); return; }
    ctrl.setTags(tagAllTags.filter(function(t) { return t.cid === categoryID && t.id !== tagId; }));
}

function loadParentTagsForAdd(categoryID) {
    document.getElementById('parent_tag_select').value = '';
    if (!window._parentTagCtrl) return;
    if (!categoryID) { window._parentTagCtrl.setTags(tagAllTags); return; }
    window._parentTagCtrl.setTags(tagAllTags.filter(function(t) { return t.cid === parseInt(categoryID); }));
}

function initParentTagSearch() {
    window._parentTagCtrl = setupTagSearch('parent_tag_select', 'parent-tag-dropdown');
    window._parentTagCtrl.setTags(tagAllTags);
}
