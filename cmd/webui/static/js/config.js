/* ── Config management ── */
(function() {
var q = '';
var currentPage = 1;

function fetchList(page) {
    currentPage = page || 1;
    var url = '/api/config?page=' + currentPage + '&page_size=10';
    if (q) url += '&q=' + encodeURIComponent(q);
    API.get(url).then(function(d) {
        if (!d.ok) { showToast(d.msg || '加载失败', 'error'); return; }
        renderList(d.items, d.pagination);
    }).catch(function() { showToast('请求失败', 'error'); });
}

function renderList(items, pg) {
    var tbody = document.getElementById('config-tbody');
    var tableWrap = document.getElementById('config-table-wrap');
    var emptyEl = document.getElementById('config-empty');
    var paginationEl = document.getElementById('config-pagination');

    if (!tbody) return;

    /* Toggle clear button */
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

    var html = '';
    for (var i = 0; i < items.length; i++) {
        var item = items[i];
        var key = escHtml(item.key);
        var val = escHtml(item.value);
        var displayId = (pg ? (pg.page - 1) * pg.page_size + i + 1 : i + 1);
        html += '<tr id="cfg-' + displayId + '">' +
            '<td class="col-id">' + displayId + '</td>' +
            '<td><code>' + key + '</code></td>' +
            '<td class="cfg-val" style="max-width:250px;overflow:hidden;text-overflow:ellipsis">' + val + '</td>' +
            '<td class="text-muted">&mdash;</td>' +
            '<td class="col-actions">' +
                '<button class="btn btn-default btn-sm" onclick="editConfig(\'' + escJsStr(key) + '\',\'' + escJsStr(val) + '\')">编辑</button> ' +
                '<button class="btn btn-danger btn-sm" onclick="removeConfig(\'' + escJsStr(key) + '\')">删除</button>' +
            '</td>' +
        '</tr>';
    }
    tbody.innerHTML = html;

    if (paginationEl) {
        if (pg && pg.total_pages > 1) {
            var ph = '<div class="pagination">';
            if (pg.has_prev) {
                ph += '<a href="#" onclick="refreshConfigList(' + pg.prev_page + ');return false">&laquo; 上一页</a>';
            }
            ph += '<span class="page-info">第 ' + pg.page + '/' + pg.total_pages + ' 页 (共 ' + pg.total + ' 条)</span>';
            if (pg.has_next) {
                ph += '<a href="#" onclick="refreshConfigList(' + pg.next_page + ');return false">下一页 &raquo;</a>';
            }
            ph += '</div>';
            paginationEl.innerHTML = ph;
        } else {
            paginationEl.innerHTML = '';
        }
    }
}

function escHtml(s) {
    if (!s) return '';
    return String(s).replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;').replace(/'/g,'&#39;');
}

function escJsStr(s) {
    if (!s) return '';
    return String(s).replace(/\\/g,'\\\\').replace(/'/g,"\\'").replace(/"/g,'\\"').replace(/\n/g,'\\n').replace(/\r/g,'\\r');
}

/* ── Search ── */
function doSearch() {
    var input = document.querySelector('#config-search-form input[name="q"]');
    q = input ? input.value.trim() : '';
    currentPage = 1;
    fetchList(1);
}

/* ── Pagination helper (global, used by onclick) ── */
window.refreshConfigList = function(page) {
    fetchList(page);
};

/* ── Mode toggle ── */
var configMode = 'set';

/* ── Form submit ── */
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

            var opts = {method: method, headers: {'Content-Type': 'application/x-www-form-urlencoded'}, body: new URLSearchParams(data)};
            fetch(url, opts).then(function(r) { return r.json(); }).then(function(d) {
                if (d.ok) {
                    showToast(d.msg || '配置已保存', 'success');
                    resetConfigForm();
                    fetchList(1);
                } else {
                    showToast(d.msg || '保存失败', 'error');
                }
            }).catch(function() { showToast('请求失败', 'error'); });
        });
    }

    /* ── Search form ── */
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

/* ── Remove (global, used by onclick) ── */
window.removeConfig = function(key) {
    showConfirm('确定删除配置 &laquo;' + key + '&raquo;？', function() {
        API.del('/api/config/' + encodeURIComponent(key)).then(function(d) {
            if (d.ok) {
                showToast(d.msg || '配置已删除', 'success');
                fetchList(currentPage);
            } else {
                showToast(d.msg || '删除失败', 'error');
            }
        }).catch(function() { showToast('请求失败', 'error'); });
    });
};

/* ── Edit (global, used by onclick) ── */
window.editConfig = function(key, val) {
    document.getElementById('config-key').value = key;
    document.getElementById('config-value').value = val;
    document.getElementById('config-desc').value = '';
    document.getElementById('config-key').readOnly = true;
    document.getElementById('config-key').style.background = 'var(--paper)';
    configMode = 'update';
    document.getElementById('config-form-card').querySelector('.card-title').textContent = '编辑配置: ' + key;
    document.querySelector('#config-form-card .cancel-btn').style.display = '';
};

/* ── Reset form (global, used by onclick) ── */
window.resetConfigForm = function() {
    document.getElementById('config-key').readOnly = false;
    document.getElementById('config-key').style.background = '';
    document.getElementById('config-form').reset();
    configMode = 'set';
    document.getElementById('config-form-card').querySelector('.card-title').textContent = '添加 / 更新配置';
    document.querySelector('#config-form-card .cancel-btn').style.display = 'none';
};
})();
