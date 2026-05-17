/* ═══════════════════════════════════════════
   TagFile — File Tree Page
   ═══════════════════════════════════════════ */

function initFileTreePage() {
    loadFileTree();
    setupScanForm();
    setupImportForm();
}

function loadFileTree() {
    var container = document.getElementById('file-tree');
    if (!container) return;
    API.get('/api/files').then(function(data) {
        if (data.ok && data.tree) {
            renderFileTree(data.tree, container);
        } else {
            container.innerHTML = '<div class="empty"><div class="empty-icon">&#128196;</div><div class="empty-text">' + escHtml(data.msg || '加载失败') + '</div></div>';
        }
    }).catch(function() {
        container.innerHTML = '<div class="empty"><div class="empty-icon">&#128196;</div><div class="empty-text">加载失败</div></div>';
    });
}

function renderFileTree(nodes, container) {
    if (!nodes || nodes.length === 0) {
        container.innerHTML = '<div class="empty"><div class="empty-icon">&#128196;</div><div class="empty-text">暂无文件</div><div class="empty-hint">请先扫描目录或导入文件</div></div>';
        return;
    }
    var html = '<div class="tree-root">';
    html += renderTreeNodes(nodes);
    html += '</div>';
    container.innerHTML = html;

    container.querySelectorAll('.tree-dir[data-tree-path]').forEach(function(el) {
        var key = 'tagfile_tree_open:' + el.getAttribute('data-tree-path');
        if (sessionStorage.getItem(key) === '1') el.open = true;
        el.addEventListener('toggle', function() {
            sessionStorage.setItem(key, el.open ? '1' : '0');
        });
    });
}

function renderTreeNodes(nodes) {
    var html = '';
    for (var i = 0; i < nodes.length; i++) {
        var n = nodes[i];
        if (n.is_dir) {
            html += '<details class="tree-dir" data-tree-path="' + escHtml(n.path || '') + '">';
            html += '<summary class="tree-summary">' + escHtml(n.name) + '/';
            if (n.tag_cnt > 0) {
                html += '<span class="badge" style="margin-left:4px">' + n.tag_cnt + '</span>';
            }
            html += '</summary>';
            html += '<div class="tree-children">';
            if (n.children) html += renderTreeNodes(n.children);
            html += '</div></details>';
        } else {
            var cls = 'tree-file' + (n.indexed ? '' : ' tree-unindexed');
            html += '<div class="' + cls + '">';
            html += '<span class="tree-icon">' + (n.indexed ? iconForName(n.name) : '') + '</span>';
            html += '<span class="tree-name">';
            if (n.indexed) {
                html += '<a href="/files/detail/' + encodePath(n.path) + '">' + escHtml(n.name) + '</a>';
            } else {
                html += '<span style="color:var(--ink-muted)">' + escHtml(n.name) + '</span>';
            }
            html += '</span>';
            if (n.indexed && n.tag_cnt > 0) {
                html += '<span class="badge">' + n.tag_cnt + '</span>';
            }
            html += '<span class="tree-size">' + formatSize(n.file_size || 0) + '</span>';
            if (!n.indexed) {
                html += '<span class="tree-status tree-status-no">未索引</span>';
            }
            html += '</div>';
        }
    }
    return html;
}

function setupScanForm() {
    var form = document.getElementById('scan-form');
    if (!form) return;
    form.addEventListener('submit', function(e) {
        e.preventDefault();
        var btn = this.querySelector('button[type="submit"]');
        btn.disabled = true;
        btn.textContent = '扫描中...';
        API.post('/api/files/scan', new FormData(this)).then(function(data) {
            btn.disabled = false;
            btn.textContent = '扫描';
            if (data.ok) {
                showToast(data.msg, 'success');
                loadFileTree();
            } else {
                showToast(data.msg, 'error');
            }
        }).catch(function() {
            btn.disabled = false;
            btn.textContent = '扫描';
            showToast('请求失败', 'error');
        });
    });
}

function setupImportForm() {
    var form = document.getElementById('import-form');
    if (!form) return;
    form.addEventListener('submit', function(e) {
        e.preventDefault();
        var btn = this.querySelector('button[type="submit"]');
        btn.disabled = true;
        btn.textContent = '导入中...';
        fetch('/api/files/import', {method:'POST', body: new FormData(this)})
            .then(function(r) { return r.json(); })
            .then(function(data) {
                btn.disabled = false;
                btn.textContent = '导入';
                if (data.ok) {
                    showToast(data.msg, 'success');
                    loadFileTree();
                } else {
                    showToast(data.msg, 'error');
                }
            })
            .catch(function() {
                btn.disabled = false;
                btn.textContent = '导入';
                showToast('请求失败', 'error');
            });
    });
}

window.refreshTree = function() {
    API.post('/api/files/refresh', '').then(function(data) {
        if (data.ok) {
            showToast(data.msg, 'success');
            loadFileTree();
        } else {
            showToast(data.msg, 'error');
        }
    }).catch(function() {
        showToast('请求失败', 'error');
    });
};
