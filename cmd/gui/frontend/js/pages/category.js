// category.js - Category management page
function renderCategories(data) {
    var q = data.query || '';
    var categories = data.categories || [];
    var pg = data.pagination || {};
    var html = '<h2>分类管理</h2>';

    html += '<div class="card"><h3>添加分类</h3>' +
        '<form onsubmit="return submitAddCategory(this)" class="row">' +
        '<div class="form-group"><label>分类名称</label><input name="name" required placeholder="例如: 项目类型"></div>' +
        '<div class="form-group"><label>描述</label><input name="description" placeholder="可选描述"></div>' +
        '<button type="submit" class="btn-primary">添加</button></form></div>';

    html += '<div class="card">' +
        '<div style="display:flex;align-items:center;gap:16px;margin-bottom:16px">' +
        '<h3 style="margin:0">分类列表</h3>' +
        '<div class="row" style="flex:1;justify-content:flex-end">' +
        '<input id="cat-search" placeholder="搜索分类..." value="' + escapeHtml(q) + '" style="width:200px">' +
        '<button class="btn-default btn-sm" onclick="searchCategories()">搜索</button>' +
        (q ? '<button class="btn-default btn-sm" onclick="loadCategories()">清除</button>' : '') +
        '</div></div>';

    if (categories.length > 0) {
        html += '<table><thead><tr><th>ID</th><th>名称</th><th>描述</th><th>操作</th></tr></thead><tbody>';
        categories.forEach(function(c) {
            html += '<tr id="cat-' + c.id + '">' +
                '<td>' + c.id + '</td>' +
                '<td class="cat-name">' + escapeHtml(c.name) + '</td>' +
                '<td class="cat-desc">' + escapeHtml(c.desc) + '</td>' +
                '<td>' +
                '<button class="btn-default btn-sm" onclick="editCategory(' + c.id + ',\'' + escapeJs(c.name) + '\',\'' + escapeJs(c.desc) + '\')">编辑</button> ' +
                '<button class="btn-danger btn-sm" onclick="deleteCategory(' + c.id + ',\'' + escapeJs(c.name) + '\')">删除</button>' +
                '</td></tr>';
        });
        html += '</tbody></table>' + renderPagination(pg, 'loadCategoriesPage');
    } else {
        html += '<p class="empty">暂无分类</p>';
    }
    html += '</div>';

    document.getElementById('page-content').innerHTML = html;
    if (!data.categories && !data.error) loadCategories();
}

function loadCategories(q, page) {
    API.listCategories(q || '', page || 1, 10).then(function(d) {
        if (d.error) { App.flash(d.error, 'error'); return; }
        renderCategories(d);
    });
}

function searchCategories() {
    loadCategories(document.getElementById('cat-search').value);
}

function loadCategoriesPage(page) {
    var q = document.getElementById('cat-search') ? document.getElementById('cat-search').value : '';
    loadCategories(q, page);
}

function submitAddCategory(form) {
    API.addCategory(form.name.value, form.description.value).then(function(r) {
        if (r.error) { App.flash(r.error, 'error'); return; }
        App.flash(r.message);
        form.reset();
        loadCategories();
    });
    return false;
}

function deleteCategory(id, name) {
    if (!confirm('确定删除分类 «' + name + '»？')) return;
    API.removeCategory(id).then(function(r) {
        if (r.error) { App.flash(r.error, 'error'); return; }
        App.flash(r.message);
        loadCategories();
    });
}

function editCategory(id, name, desc) {
    var cells = document.querySelectorAll('#cat-' + id + ' td');
    cells[1].innerHTML = '<input id="edit_cat_name_' + id + '" value="' + escapeHtml(name) + '" required style="width:120px">';
    cells[2].innerHTML = '<input id="edit_cat_desc_' + id + '" value="' + escapeHtml(desc || '') + '" style="width:150px">';
    cells[3].innerHTML = '<button class="btn-primary btn-sm" onclick="saveCategory(' + id + ')">保存</button> ' +
        '<button class="btn-default btn-sm" onclick="loadCategories()">取消</button>';
}

function saveCategory(id) {
    var name = document.getElementById('edit_cat_name_' + id).value;
    var desc = document.getElementById('edit_cat_desc_' + id).value;
    API.updateCategory(id, name, desc).then(function(r) {
        if (r.error) { App.flash(r.error, 'error'); return; }
        App.flash(r.message);
        loadCategories();
    });
}

function escapeHtml(s) {
    if (!s) return '';
    return (s+'').replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
}

function escapeJs(s) {
    if (!s) return '';
    return (s+'').replace(/\\/g, '\\\\').replace(/'/g, "\\'").replace(/"/g, '\\"');
}
