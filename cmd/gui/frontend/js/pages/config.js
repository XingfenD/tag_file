// config.js - Config management + export
function renderConfig(data) {
    var q = data.query || '';
    var configs = data.configs || [];
    var pg = data.pagination || {};

    var html = '<h2>系统配置</h2>';

    html += '<div class="card"><h3 id="cfg-form-title">添加 / 更新配置</h3>' +
        '<form id="cfg-form" onsubmit="return submitConfig(this)" class="row">' +
        '<input type="hidden" name="mode" value="set">' +
        '<div class="form-group"><label>配置键</label><input name="key" id="cfg-key" required placeholder="例如: default_editor"></div>' +
        '<div class="form-group"><label>配置值</label><input name="value" id="cfg-value" required placeholder="配置值" style="width:300px"></div>' +
        '<button type="submit" class="btn-primary" id="cfg-submit-btn">保存</button>' +
        '<button type="button" class="btn-default btn-sm" id="cfg-cancel-btn" style="display:none;margin-left:8px" onclick="resetConfigForm()">取消</button>' +
        '</form></div>';

    html += '<div class="card">' +
        '<div style="display:flex;align-items:center;gap:16px;margin-bottom:16px">' +
        '<h3 style="margin:0">配置列表</h3>' +
        '<div class="row" style="flex:1;justify-content:flex-end">' +
        '<input id="cfg-search" placeholder="搜索配置..." value="' + escapeHtml(q) + '" style="width:200px">' +
        '<button class="btn-default btn-sm" onclick="searchConfigs()">搜索</button>' +
        (q ? '<button class="btn-default btn-sm" onclick="loadConfigs()">清除</button>' : '') +
        '</div></div>';

    if (configs.length > 0) {
        html += '<table><thead><tr><th>ID</th><th>键</th><th>值</th><th>描述</th><th>操作</th></tr></thead><tbody>';
        configs.forEach(function(c) {
            html += '<tr id="cfg-' + c.id + '">' +
                '<td>' + c.id + '</td>' +
                '<td><code>' + escapeHtml(c.key) + '</code></td>' +
                '<td class="cfg-val" style="max-width:250px;overflow:hidden;text-overflow:ellipsis">' + escapeHtml(c.value) + '</td>' +
                '<td>' + escapeHtml(c.desc) + '</td>' +
                '<td>' +
                '<button class="btn-default btn-sm" onclick="editConfig(\'' + escapeJs(c.key) + '\',\'' + escapeJs(c.value) + '\')">编辑</button> ' +
                '<button class="btn-danger btn-sm" onclick="deleteConfig(\'' + escapeJs(c.key) + '\')">删除</button>' +
                '</td></tr>';
        });
        html += '</tbody></table>' + renderPagination(pg, 'loadConfigsPage');
    } else {
        html += '<p class="empty">暂无配置</p>';
    }
    html += '</div>';

    html += '<div class="card"><h3>数据导出</h3>' +
        '<p style="font-size:13px;color:#666;margin-bottom:12px">将数据库导出为 SQL 文件，可用于备份或迁移。</p>' +
        '<button class="btn-default" onclick="doExport()">📥 导出 SQL 文件</button></div>';

    document.getElementById('page-content').innerHTML = html;
    if (!data.configs && !data.error) loadConfigs();
}

function loadConfigs(q, page) {
    API.listConfigs(q || '', page || 1, 10).then(function(d) {
        if (d.error) { App.flash(d.error, 'error'); return; }
        renderConfig(d);
    });
}

function searchConfigs() {
    loadConfigs(document.getElementById('cfg-search').value);
}

function loadConfigsPage(page) {
    var q = document.getElementById('cfg-search') ? document.getElementById('cfg-search').value : '';
    loadConfigs(q, page);
}

function submitConfig(form) {
    var key = form.key.value;
    var value = form.value.value;
    var mode = form.mode.value;
    var promise = mode === 'update' ? API.updateConfig(key, value) : API.setConfig(key, value);
    promise.then(function(r) {
        if (r.error) { App.flash(r.error, 'error'); return; }
        App.flash(r.message);
        resetConfigForm();
        loadConfigs();
    });
    return false;
}

function editConfig(key, val) {
    document.getElementById('cfg-key').value = key;
    document.getElementById('cfg-key').readOnly = true;
    document.getElementById('cfg-key').style.background = '#f0f0f0';
    document.getElementById('cfg-value').value = val;
    document.getElementById('cfg-form-title').textContent = '编辑配置: ' + key;
    document.getElementById('cfg-form').mode.value = 'update';
    document.getElementById('cfg-cancel-btn').style.display = 'inline';
    document.getElementById('cfg-submit-btn').textContent = '更新';
}

function resetConfigForm() {
    var f = document.getElementById('cfg-form');
    f.reset();
    f.mode.value = 'set';
    document.getElementById('cfg-key').readOnly = false;
    document.getElementById('cfg-key').style.background = '';
    document.getElementById('cfg-form-title').textContent = '添加 / 更新配置';
    document.getElementById('cfg-cancel-btn').style.display = 'none';
    document.getElementById('cfg-submit-btn').textContent = '保存';
}

function deleteConfig(key) {
    if (!confirm('确定删除配置 «' + key + '»？')) return;
    API.unsetConfig(key).then(function(r) {
        if (r.error) { App.flash(r.error, 'error'); return; }
        App.flash(r.message);
        loadConfigs();
    });
}

function doExport() {
    API.exportSQL().then(function(r) {
        if (r.error) { App.flash(r.error, 'error'); return; }
        var blob = new Blob([r.sql], {type: 'application/sql'});
        var a = document.createElement('a');
        a.href = URL.createObjectURL(blob);
        a.download = 'tagfile_' + new Date().toISOString().replace(/[:.]/g, '').slice(0, 15) + '.sql';
        a.click();
        App.flash('SQL 文件已导出');
    });
}
