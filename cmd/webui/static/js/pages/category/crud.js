/* ═══════════════════════════════════════════
   TagFile — Category Page: CRUD
   ═══════════════════════════════════════════ */

window.removeCategory = function(id, name) {
    showConfirm('确定删除分类 &laquo;' + name + '&raquo;？', function() {
        API.del('/api/categories/' + id).then(function(d) {
            if (!d.ok) { showToast(d.msg || '删除失败', 'error'); return; }
            showToast(d.msg || '分类已删除', 'success');
            fetchList(currentPage);
        }).catch(function() { showToast('请求失败', 'error'); });
    });
};

window.editCategory = function(id, name, desc) {
    Modal.show({
        title: '编辑分类',
        fields: [
            { label: '名称', name: 'name', value: name, required: true },
            { label: '描述', name: 'description', value: desc || '' }
        ],
        onSave: async function(values) {
            var newName = values.name.trim();
            if (!newName) { showToast('名称不能为空', 'error'); return false; }
            var d = await API.put('/api/categories/' + id, {name: newName, description: values.description.trim()});
            if (!d.ok) { showToast(d.msg || '更新失败', 'error'); return false; }
            showToast(d.msg || '分类已更新', 'success');
            fetchList(currentPage);
        }
    });
};
