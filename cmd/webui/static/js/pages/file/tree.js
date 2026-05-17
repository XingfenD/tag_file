/* ═══════════════════════════════════════════
   TagFile — File Tree Page
   ═══════════════════════════════════════════ */

function initFileTreePage() {
    treeSortField = 'name';
    treeSortOrder = 'asc';
    initTreeSortControls();
    loadFileTree();
    setupScanForm();
    setupImportForm();
}

function initTreeSortControls() {
    var fieldSel = document.getElementById('tree-sort-field');
    var orderBtn = document.getElementById('tree-sort-order-btn');
    if (!fieldSel || !orderBtn) return;
    updateTreeOrderBtn(orderBtn);
    fieldSel.addEventListener('change', function() {
        treeSortField = fieldSel.value;
        sortAndRerenderTree();
    });
    orderBtn.addEventListener('click', function() {
        treeSortOrder = treeSortOrder === 'asc' ? 'desc' : 'asc';
        updateTreeOrderBtn(orderBtn);
        sortAndRerenderTree();
    });
}

function updateTreeOrderBtn(btn) {
    if (treeSortOrder === 'desc') {
        btn.classList.add('desc');
        btn.innerHTML = '降序 ▾';
    } else {
        btn.classList.remove('desc');
        btn.innerHTML = '升序 ▴';
    }
}

function sortAndRerenderTree() {
    var container = document.getElementById('file-tree');
    if (!container || !lastTreeData) return;
    var sorted = sortTreeNodes(structuredClone ? structuredClone(lastTreeData) : JSON.parse(JSON.stringify(lastTreeData)));
    renderFileTree(sorted, container);
}

function sortTreeNodes(nodes) {
    if (!nodes) return nodes;
    var desc = treeSortOrder === 'desc';
    nodes.sort(function(a, b) {
        var va, vb;
        if (a.is_dir && b.is_dir) return a.name.localeCompare(b.name);
        if (a.is_dir) return -1;
        if (b.is_dir) return 1;
        switch (treeSortField) {
            case 'size':  va = a.file_size || 0; vb = b.file_size || 0; break;
            case 'tagcnt': va = a.tag_cnt || 0; vb = b.tag_cnt || 0; break;
            case 'type':  va = a.name.split('.').pop() || ''; vb = b.name.split('.').pop() || ''; break;
            default:      va = a.name; vb = b.name;
        }
        if (typeof va === 'string') return desc ? vb.localeCompare(va) : va.localeCompare(vb);
        return desc ? vb - va : va - vb;
    });
    for (var i = 0; i < nodes.length; i++) {
        if (nodes[i].children) sortTreeNodes(nodes[i].children);
    }
    return nodes;
}

function loadFileTree() {
    var container = document.getElementById('file-tree');
    if (!container) return;
    API.get('/api/files').then(function(data) {
        if (data.ok && data.tree) {
            lastTreeData = data.tree;
            var sorted = sortTreeNodes(JSON.parse(JSON.stringify(data.tree)));
            renderFileTree(sorted, container);
        } else {
            while (container.firstChild) container.removeChild(container.firstChild);
            container.appendChild(makeEmptyState('&#128196;', data.msg || '加载失败'));
        }
    }).catch(function() {
        while (container.firstChild) container.removeChild(container.firstChild);
        container.appendChild(makeEmptyState('&#128196;', '加载失败'));
    });
}

function renderFileTree(nodes, container) {
    while (container.firstChild) container.removeChild(container.firstChild);

    if (!nodes || nodes.length === 0) {
        container.appendChild(makeEmptyState('&#128196;', '暂无文件', '请先扫描目录或导入文件'));
        return;
    }
    var root = createEl('div', {className: 'tree-root'});
    renderTreeNodes(root, nodes);
    container.appendChild(root);

    container.querySelectorAll('.tree-dir[data-tree-path]').forEach(function(el) {
        var key = 'tagfile_tree_open:' + el.getAttribute('data-tree-path');
        if (sessionStorage.getItem(key) === '1') el.open = true;
        el.addEventListener('toggle', function() {
            sessionStorage.setItem(key, el.open ? '1' : '0');
        });
    });
}

function renderTreeNodes(parent, nodes) {
    for (var i = 0; i < nodes.length; i++) {
        var n = nodes[i];
        if (n.is_dir) {
            var summary = createEl('summary', {className: 'tree-summary'});
            summary.appendChild(document.createTextNode(n.name + '/'));
            if (n.tag_cnt > 0) summary.appendChild(createEl('span', {className: 'badge', style: 'margin-left:4px'}, String(n.tag_cnt)));
            var details = createEl('details', {className: 'tree-dir', 'data-tree-path': n.path || ''}, summary);
            var childrenDiv = createEl('div', {className: 'tree-children'});
            if (n.children) renderTreeNodes(childrenDiv, n.children);
            details.appendChild(childrenDiv);
            parent.appendChild(details);
        } else {
            var cls = 'tree-file' + (n.indexed ? '' : ' tree-unindexed');
            var nameSpan = createEl('span', {className: 'tree-name'});
            if (n.indexed) {
                nameSpan.appendChild(createEl('a', {href: '/files/detail/' + encodePath(n.path)}, n.name));
            } else {
                nameSpan.appendChild(createEl('span', {style: 'color:var(--ink-muted)'}, n.name));
            }
            var fileDiv = createEl('div', {className: cls},
                createEl('span', {className: 'tree-icon'}, n.indexed ? iconForName(n.name) : ''),
                nameSpan
            );
            if (n.indexed && n.tag_cnt > 0) fileDiv.appendChild(createEl('span', {className: 'badge'}, String(n.tag_cnt)));
            fileDiv.appendChild(createEl('span', {className: 'tree-size'}, formatSize(n.file_size || 0)));
            if (!n.indexed) fileDiv.appendChild(createEl('span', {className: 'tree-status tree-status-no'}, '未索引'));
            parent.appendChild(fileDiv);
        }
    }
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
