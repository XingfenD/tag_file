/* ── Category management ── */
(function() {
var q = '';
var currentPage = 1;

function fetchList(page) {
    currentPage = page || 1;
    var url = '/api/categories?page=' + currentPage + '&page_size=10';
    if (q) url += '&q=' + encodeURIComponent(q);
    API.get(url).then(function(d) {
        if (!d.ok) { showToast(d.msg || '加载失败', 'error'); return; }
        renderList(d.items, d.pagination);
    }).catch(function() { showToast('请求失败', 'error'); });
}

function renderList(items, pg) {
    var container = document.getElementById('category-list');
    if (!container) return;

    /* Toggle clear button */
    var clearBtn = document.getElementById('cat-clear-btn');
    if (clearBtn) clearBtn.style.display = q ? '' : 'none';

    if (!items || items.length === 0) {
        container.innerHTML =
            '<div class="empty">' +
                '<div class="empty-icon">&#128451;</div>' +
                '<div class="empty-text">暂无分类</div>' +
                '<div class="empty-hint">在上方添加第一个分类</div>' +
            '</div>';
        return;
    }

    var html = '<table><thead><tr><th>ID</th><th>名称</th><th>描述</th><th style="width:140px">操作</th></tr></thead><tbody>';
    for (var i = 0; i < items.length; i++) {
        var item = items[i];
        var name = escHtml(item.name);
        var desc = item.desc ? escHtml(item.desc) : '&mdash;';
        html += '<tr id="cat-' + item.id + '">' +
            '<td class="col-id">' + item.id + '</td>' +
            '<td class="cat-name" style="font-weight:500">' + name + '</td>' +
            '<td class="cat-desc">' + desc + '</td>' +
            '<td class="col-actions">' +
                '<button class="btn btn-default btn-sm" onclick="editCategory(' + item.id + ',\'' + escJsStr(name) + '\',\'' + escJsStr(item.desc || '') + '\')">编辑</button> ' +
                '<button class="btn btn-danger btn-sm" onclick="removeCategory(' + item.id + ',\'' + escJsStr(name) + '\')">删除</button>' +
            '</td>' +
        '</tr>';
    }
    html += '</tbody></table>';

    if (pg && pg.total_pages > 1) {
        html += '<div class="pagination">';
        if (pg.has_prev) {
            html += '<a href="#" onclick="refreshCategoryList(' + pg.prev_page + ');return false">&laquo; 上一页</a>';
        }
        html += '<span class="page-info">第 ' + pg.page + '/' + pg.total_pages + ' 页 (共 ' + pg.total + ' 条)</span>';
        if (pg.has_next) {
            html += '<a href="#" onclick="refreshCategoryList(' + pg.next_page + ');return false">下一页 &raquo;</a>';
        }
        html += '</div>';
    }

    container.innerHTML = html;
}

function escHtml(s) {
    if (!s) return '';
    return String(s).replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;').replace(/'/g,'&#39;');
}

function escJsStr(s) {
    if (!s) return '';
    return String(s).replace(/\\/g,'\\\\').replace(/'/g,"\\'").replace(/"/g,'\\"').replace(/\n/g,'\\n').replace(/\r/g,'\\r');
}

/* ── Add form ── */
document.addEventListener('DOMContentLoaded', function() {
    var form = document.getElementById('add-cat-form');
    if (form) {
        form.addEventListener('submit', function(e) {
            e.preventDefault();
            var data = new FormData(form);
            API.post('/api/categories', data).then(function(d) {
                if (!d.ok) { showToast(d.msg || '添加失败', 'error'); return; }
                showToast(d.msg || '分类已添加', 'success');
                form.reset();
                fetchList(1);
            }).catch(function() { showToast('请求失败', 'error'); });
        });
    }

    /* ── Search form ── */
    var searchForm = document.querySelector('#category-list-wrap form');
    if (searchForm) {
        searchForm.addEventListener('submit', function(e) {
            e.preventDefault();
            var input = searchForm.querySelector('input[name="q"]');
            q = input ? input.value.trim() : '';
            currentPage = 1;
            fetchList(1);
        });
        var clearLink = searchForm.querySelector('.btn-ghost');
        if (clearLink) {
            clearLink.addEventListener('click', function(e) {
                e.preventDefault();
                var input = searchForm.querySelector('input[name="q"]');
                if (input) input.value = '';
                q = '';
                currentPage = 1;
                fetchList(1);
            });
        }
    }

    fetchList(1);
});

/* ── Pagination helper (called from onclick) ── */
window.refreshCategoryList = function(page) {
    fetchList(page);
};

/* ── Remove (global, used by onclick) ── */
window.removeCategory = function(id, name) {
    showConfirm('确定删除分类 &laquo;' + name + '&raquo;？', function() {
        API.del('/api/categories/' + id).then(function(d) {
            if (!d.ok) { showToast(d.msg || '删除失败', 'error'); return; }
            showToast(d.msg || '分类已删除', 'success');
            fetchList(currentPage);
        }).catch(function() { showToast('请求失败', 'error'); });
    });
};

/* ── Edit (global, used by onclick) ── */
window.editCategory = function(id, name, desc) {
    var cells = document.querySelectorAll('#cat-' + id + ' td');
    cells[1].innerHTML = '<input id="edit_cat_name_' + id + '" value="' + escHtml(name) + '" required style="width:120px">';
    cells[2].innerHTML = '<input id="edit_cat_desc_' + id + '" value="' + escHtml(desc) + '" style="width:160px">';
    cells[3].innerHTML = '<button class="btn btn-primary btn-sm" onclick="saveCategory(' + id + ')">保存</button> <button type="button" class="btn btn-default btn-sm" onclick="refreshCategoryList(' + currentPage + ')">取消</button>';
};

/* ── Save (global, used by onclick) ── */
window.saveCategory = function(id) {
    var name = document.getElementById('edit_cat_name_' + id).value;
    var desc = document.getElementById('edit_cat_desc_' + id).value;
    API.put('/api/categories/' + id, {name: name, description: desc}).then(function(d) {
        if (!d.ok) { showToast(d.msg || '更新失败', 'error'); return; }
        showToast(d.msg || '分类已更新', 'success');
        fetchList(currentPage);
    }).catch(function() { showToast('请求失败', 'error'); });
};
})();
