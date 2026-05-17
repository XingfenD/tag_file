/* ═══════════════════════════════════════════
   TagFile — Category Page: State, List, Search
   ═══════════════════════════════════════════ */

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

    var html = '<table><thead><tr><th>ID</th><th>名称</th><th>描述</th><th style="width:60px">操作</th></tr></thead><tbody>';
    for (var i = 0; i < items.length; i++) {
        var item = items[i];
        var name = escHtml(item.name);
        var desc = item.desc ? escHtml(item.desc) : '&mdash;';
        html += '<tr id="cat-' + item.id + '">' +
            '<td class="col-id">' + item.id + '</td>' +
            '<td class="cat-name" data-value="' + escHtml(item.name) + '" style="font-weight:500">' + name + '</td>' +
            '<td class="cat-desc" data-value="' + escHtml(item.desc || '') + '">' + desc + '</td>' +
            '<td class="col-actions">' +
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

    container.querySelectorAll('.cat-name').forEach(function(cell) {
        var tr = cell.closest('tr');
        if (!tr) return;
        var id = parseInt(tr.id.replace('cat-', ''));
        if (isNaN(id)) return;
        var descCell = tr.querySelector('.cat-desc');
        makeEditable(cell, {
            onSave: async function(newName) {
                var curDesc = descCell ? descCell.getAttribute('data-value') || '' : '';
                var d = await API.put('/api/categories/' + id, {name: newName, description: curDesc});
                if (!d.ok) { showToast(d.msg || '更新失败', 'error'); throw new Error(d.msg); }
                showToast(d.msg || '分类已更新', 'success');
                descCell.setAttribute('data-value', curDesc);
                fetchList(currentPage);
            }
        });
    });

    container.querySelectorAll('.cat-desc').forEach(function(cell) {
        var tr = cell.closest('tr');
        if (!tr) return;
        var id = parseInt(tr.id.replace('cat-', ''));
        if (isNaN(id)) return;
        var nameCell = tr.querySelector('.cat-name');
        makeEditable(cell, {
            emptyText: '&mdash;',
            onSave: async function(newDesc) {
                var curName = nameCell ? nameCell.getAttribute('data-value') || '' : '';
                var d = await API.put('/api/categories/' + id, {name: curName, description: newDesc});
                if (!d.ok) { showToast(d.msg || '更新失败', 'error'); throw new Error(d.msg); }
                showToast(d.msg || '分类已更新', 'success');
                nameCell.setAttribute('data-value', curName);
                fetchList(currentPage);
            }
        });
    });
}

/* ── Init: Add form + Search form ── */
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

    var searchForm = document.getElementById('cat-search-form');
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
