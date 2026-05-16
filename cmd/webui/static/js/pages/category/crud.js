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
    var cells = document.querySelectorAll('#cat-' + id + ' td');
    cells[1].innerHTML = '<input id="edit_cat_name_' + id + '" value="' + escHtml(name) + '" required style="width:120px">';
    cells[2].innerHTML = '<input id="edit_cat_desc_' + id + '" value="' + escHtml(desc) + '" style="width:160px">';
    cells[3].innerHTML = '<button class="btn btn-primary btn-sm" onclick="saveCategory(' + id + ')">保存</button> <button type="button" class="btn btn-default btn-sm" onclick="refreshCategoryList(' + currentPage + ')">取消</button>';
};

window.saveCategory = function(id) {
    var name = document.getElementById('edit_cat_name_' + id).value;
    var desc = document.getElementById('edit_cat_desc_' + id).value;
    API.put('/api/categories/' + id, {name: name, description: desc}).then(function(d) {
        if (!d.ok) { showToast(d.msg || '更新失败', 'error'); return; }
        showToast(d.msg || '分类已更新', 'success');
        fetchList(currentPage);
    }).catch(function() { showToast('请求失败', 'error'); });
};
