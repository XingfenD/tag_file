// file.js - File tree, scan, import
function renderFiles(data) {
    if (!data.tree && !data.error) {
        API.getFileTree().then(function(d) {
            if (d.error) { App.flash(d.error, 'error'); }
            renderFiles(d);
        });
        return;
    }

    var tree = data.tree || [];
    var html = '<h2>文件管理</h2>';

    // Scan form
    html += '<div class="card"><h3>扫描目录</h3>' +
        '<form onsubmit="return doScan(this)" class="row">' +
        '<div class="form-group"><label>目录路径</label><input name="dir" placeholder="留空为当前目录" style="width:300px"></div>' +
        '<button type="submit" class="btn btn-primary">扫描</button></form></div>';

    // Import form
    html += '<div class="card"><h3>导入文件</h3>' +
        '<form onsubmit="return doImport(this)" class="row">' +
        '<div class="form-group"><label>文件路径</label>' +
        '<input name="path" required placeholder="E:\\path\\to\\file.pdf" style="width:350px"></div>' +
        '<button type="submit" class="btn-primary">导入</button></form></div>';

    // File tree
    html += '<div class="card">' +
        '<div style="display:flex;align-items:center;gap:12px;margin-bottom:16px">' +
        '<h3 style="margin:0">文件列表</h3>' +
        '<button class="btn-default btn-sm" onclick="doRefreshTree()">刷新目录缓存</button></div>';

    if (tree.length > 0) {
        html += '<div class="tree-root">';
        tree.forEach(function(node) { html += renderFileTreeNode(node, 0); });
        html += '</div>';
    } else {
        html += '<p class="empty">暂无文件，请先扫描目录</p>';
    }
    html += '</div>';

    document.getElementById('page-content').innerHTML = html;
}

function renderFileTreeNode(node, depth) {
    if (node.isDir) {
        var html = '<details class="tree-dir"' + (depth === 0 ? ' open' : '') + '>' +
            '<summary class="tree-summary">' + escapeHtml(node.name) + '/';
        if (node.tagCnt > 0) html += ' <span class="tag-cnt">' + node.tagCnt + '</span>';
        html += '</summary><div class="children">';
        (node.children || []).forEach(function(c) { html += renderFileTreeNode(c, depth + 1); });
        html += '</div></details>';
        return html;
    }

    var icon = node.indexed ? iconForFile(node.fileType || '') : '❓';
    var html = '<div class="tree-file' + (node.indexed ? '' : ' tree-unindexed') + '">' +
        '<span class="tree-icon">' + icon + '</span> ';
    if (node.indexed) {
        html += '<a href="#" onclick="openFileDetail(\'' + escapeJs(node.path) + '\');return false" class="tree-name" style="text-decoration:none;color:#1a1a2e">' + escapeHtml(node.name) + '</a>';
        if (node.tagCnt > 0) html += ' <span class="tag-cnt">' + node.tagCnt + '</span>';
    } else {
        html += '<span class="tree-name tree-unindexed">' + escapeHtml(node.name) + '</span>';
    }
    html += ' <span class="tree-size">' + formatSizeStr(node.fileSize || 0) + '</span>';
    if (node.indexed) {
        html += ' <a href="#" onclick="openFileDetail(\'' + escapeJs(node.path) + '\');return false" class="btn-default btn-sm" style="text-decoration:none;margin-left:4px">详情</a>';
    } else {
        html += ' <span class="unindexed">未索引</span>';
    }
    html += '</div>';
    return html;
}

function openFileDetail(filePath) {
    API.getFileDetail(filePath, 0, 1, 10).then(function(d) {
        if (d.error) { App.flash(d.error, 'error'); return; }
        App.navigate('fileDetail', d);
    });
}

function iconForFile(ft) {
    if (!ft) return '📄';
    if (ft.indexOf('video/') === 0) return '🎬';
    if (ft.indexOf('image/') === 0) return '🖼';
    if (ft.indexOf('audio/') === 0) return '🎵';
    if (ft.indexOf('pdf') >= 0) return '📄';
    if (ft.indexOf('zip') >= 0 || ft.indexOf('tar') >= 0 || ft.indexOf('rar') >= 0) return '📦';
    return '📄';
}

function formatSizeStr(size) {
    if (!size) return '0 B';
    var units = ['B', 'KB', 'MB', 'GB', 'TB'];
    var i = 0;
    var s = size;
    while (s >= 1024 && i < units.length - 1) { s /= 1024; i++; }
    return s.toFixed(1) + ' ' + units[i];
}

function doScan(form) {
    var dir = form.dir.value || '.';
    API.scanDirectory(dir).then(function(r) {
        if (r.error) { App.flash(r.error, 'error'); return; }
        App.flash(r.message);
        API.getFileTree().then(function(d) { renderFiles(d); });
    });
    return false;
}

function doImport(form) {
    var path = form.path.value;
    if (!path) { App.flash('请输入文件路径', 'error'); return false; }
    API.importFile(path).then(function(r) {
        if (r.error) { App.flash(r.error, 'error'); return; }
        App.flash(r.message);
        API.getFileTree().then(function(d) { renderFiles(d); });
    });
    return false;
}

function doRefreshTree() {
    API.refreshTree().then(function(r) {
        App.flash(r.message);
        API.getFileTree().then(function(d) { renderFiles(d); });
    });
}
