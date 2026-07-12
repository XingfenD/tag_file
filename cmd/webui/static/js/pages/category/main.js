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
    }).catch(API.onError);
}

function renderList(items, pg) {
    var container = document.getElementById('category-list');
    if (!container) return;

    var clearBtn = document.getElementById('cat-clear-btn');
    if (clearBtn) clearBtn.style.display = q ? '' : 'none';

    clearChildren(container);

    if (!items || items.length === 0) {
        container.appendChild(makeEmptyState('&#128451;', '暂无分类', '在上方添加第一个分类'));
        return;
    }

    var tbody = createEl('tbody');
    for (var i = 0; i < items.length; i++) {
        var item = items[i];
        var name = item.name;
        var desc = item.desc || '';
        var delBtn = makeActionBtn('删除', 'btn btn-danger btn-sm', function(id, n) { return function() { removeCategory(id, n); }; }(item.id, name));
        var tr = createEl('tr', {id: 'cat-' + item.id},
            createEl('td', {className: 'col-id'}, String(item.id)),
            createEl('td', {className: 'cat-name', 'data-value': item.name, style: 'font-weight:500'}, name),
            createEl('td', {className: 'cat-desc', 'data-value': item.desc || ''}, desc || '—'),
            createEl('td', {className: 'col-actions'}, delBtn)
        );
        tbody.appendChild(tr);
    }
    var table = createEl('table', null,
        createEl('thead', null, createEl('tr', null,
            createEl('th', null, 'ID'),
            createEl('th', null, '名称'),
            createEl('th', null, '描述'),
            createEl('th', {style: 'width:60px'}, '操作')
        )),
        tbody
    );
    container.appendChild(table);

    if (pg && pg.total_pages > 1) {
        container.appendChild(makePagination(pg, refreshCategoryList, {
            infoClass: 'page-info',
            infoText: '第 ' + pg.page + '/' + pg.total_pages + ' 页 (共 ' + pg.total + ' 条)'
        }));
    }

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
            }).catch(API.onError);
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
