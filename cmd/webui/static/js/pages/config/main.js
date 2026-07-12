/* ═══════════════════════════════════════════
   TagFile — Config Page: State, List, Search
   ═══════════════════════════════════════════ */

var q = '';
var currentPage = 1;

function fetchList(page) {
    currentPage = page || 1;
    var url = '/api/config?page=' + currentPage + '&page_size=10';
    if (q) url += '&q=' + encodeURIComponent(q);
    API.get(url).then(function(d) {
        if (!d.ok) { showToast(d.msg || '加载失败', 'error'); return; }
        renderList(d.items, d.pagination);
    }).catch(API.onError);
}

function renderList(items, pg) {
    var tbody = document.getElementById('config-tbody');
    var tableWrap = document.getElementById('config-table-wrap');
    var emptyEl = document.getElementById('config-empty');
    var paginationEl = document.getElementById('config-pagination');

    if (!tbody) return;

    var clearBtn = document.getElementById('config-clear-btn');
    if (clearBtn) clearBtn.style.display = q ? '' : 'none';

    if (!items || items.length === 0) {
        if (tableWrap) tableWrap.style.display = 'none';
        if (emptyEl) emptyEl.style.display = '';
        if (paginationEl) paginationEl.innerHTML = '';
        return;
    }

    if (tableWrap) tableWrap.style.display = '';
    if (emptyEl) emptyEl.style.display = 'none';

    clearChildren(tbody);
    for (var i = 0; i < items.length; i++) {
        var item = items[i];
        var key = item.key;
        var val = item.value;
        var displayId = (pg ? (pg.page - 1) * pg.page_size + i + 1 : i + 1);
        var editBtn = makeActionBtn('编辑', 'btn btn-default btn-sm', function(k, v) { return function() { editConfig(k, v); }; }(key, val));
        var delBtn = makeActionBtn('删除', 'btn btn-danger btn-sm', function(k) { return function() { removeConfig(k); }; }(key));
        var tr = createEl('tr', {id: 'cfg-' + displayId},
            createEl('td', {className: 'col-id'}, String(displayId)),
            createEl('td', null, createEl('code', null, key)),
            createEl('td', {className: 'cfg-val', style: 'max-width:250px;overflow:hidden;text-overflow:ellipsis'}, val),
            createEl('td', {className: 'text-muted'}, '—'),
            createEl('td', {className: 'col-actions'}, editBtn, document.createTextNode(' '), delBtn)
        );
        tbody.appendChild(tr);
    }

    if (paginationEl) {
        clearChildren(paginationEl);
        if (pg && pg.total_pages > 1) {
            paginationEl.appendChild(makePagination(pg, refreshConfigList, {
                infoClass: 'page-info',
                infoText: '第 ' + pg.page + '/' + pg.total_pages + ' 页 (共 ' + pg.total + ' 条)'
            }));
        }
    }
}

/* ── Search ── */
function doSearch() {
    var input = document.querySelector('#config-search-form input[name="q"]');
    q = input ? input.value.trim() : '';
    currentPage = 1;
    fetchList(1);
}

window.refreshConfigList = function(page) {
    fetchList(page);
};

/* ── Init: Form submit + Search ── */
document.addEventListener('DOMContentLoaded', function() {
    var form = document.getElementById('config-form');
    if (form) {
        form.addEventListener('submit', function(e) {
            e.preventDefault();
            var data = new FormData(this);
            var key = data.get('key');
            var value = data.get('value');

            var url, method;
            if (configMode === 'set') {
                url = '/api/config';
                method = 'POST';
            } else {
                url = '/api/config/' + encodeURIComponent(key);
                method = 'PUT';
            }

            var apiFn = method === 'POST' ? API.post : API.put;
            var apiUrl = method === 'POST' ? '/api/config' : '/api/config/' + encodeURIComponent(key);
            apiFn(apiUrl, data).then(function(d) {
                if (d.ok) {
                    showToast(d.msg || '配置已保存', 'success');
                    resetConfigForm();
                    fetchList(1);
                } else {
                    showToast(d.msg || '保存失败', 'error');
                }
            }).catch(API.onError);
        });
    }

    var searchForm = document.getElementById('config-search-form');
    if (searchForm) {
        searchForm.addEventListener('submit', function(e) {
            e.preventDefault();
            doSearch();
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
