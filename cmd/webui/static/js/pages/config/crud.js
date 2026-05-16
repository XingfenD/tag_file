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
    Modal.show({
        title: '编辑配置: ' + escHtml(key),
        fields: [
            { label: '配置键', name: 'key', value: key, readonly: true },
            { label: '配置值', name: 'value', value: val, required: true }
        ],
        onSave: async function(values) {
            var newVal = values.value.trim();
            if (!newVal) { showToast('配置值不能为空', 'error'); return false; }
            var encodedKey = encodeURIComponent(values.key);
            var d = await API.put('/api/config/' + encodedKey, {value: newVal});
            if (!d.ok) { showToast(d.msg || '更新失败', 'error'); return false; }
            showToast(d.msg || '配置已更新', 'success');
            fetchList(1);
        }
    });
};

window.resetConfigForm = function() {
    document.getElementById('config-key').readOnly = false;
    document.getElementById('config-key').style.background = '';
    document.getElementById('config-form').reset();
    configMode = 'set';
    document.getElementById('config-form-card').querySelector('.card-title').textContent = '添加 / 更新配置';
    document.querySelector('#config-form-card .cancel-btn').style.display = 'none';
};
