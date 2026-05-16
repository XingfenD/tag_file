/* ═══════════════════════════════════════════
   TagFile — Config Page: CRUD
   ═══════════════════════════════════════════ */

var configMode = 'set';

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

window.resetConfigForm = function() {
    document.getElementById('config-key').readOnly = false;
    document.getElementById('config-key').style.background = '';
    document.getElementById('config-form').reset();
    configMode = 'set';
    document.getElementById('config-form-card').querySelector('.card-title').textContent = '添加 / 更新配置';
    document.querySelector('#config-form-card .cancel-btn').style.display = 'none';
};
