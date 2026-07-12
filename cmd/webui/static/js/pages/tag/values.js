/* ═══════════════════════════════════════════
   TagFile — Tag Page: Tag Value CRUD
   ═══════════════════════════════════════════ */

function initTagValues() {
  document.querySelectorAll('.tag-values, .tag-tree-values').forEach(function(cell) {
    var tagName = cell.getAttribute('data-tag-name');
    if (!tagName) return;
    var isTreeCell = cell.classList.contains('tag-tree-values');
    API.get('/api/tags/values?tag_name=' + encodeURIComponent(tagName)).then(function(data) {
      var values = data.values || [];
      clearChildren(cell);
      values.forEach(function(v) {
        var chip = createEl('span', {className: 'tag', style: 'cursor:pointer', 'data-tag-name': tagName, 'data-value': v.value, title: '点击删除', onclick: function() { removeTagValue(this); }});
        chip.appendChild(document.createTextNode(v.value + ' '));
        chip.appendChild(createEl('span', {className: 'tag-remove'}, '×'));
        cell.appendChild(chip);
      });
      if (!isTreeCell) {
        var addBtn = createEl('a', {href: '#', 'data-tag-name': tagName, className: 'btn btn-ghost btn-xs', onclick: function(e) { e.preventDefault(); addTagValueModal(this); }}, '+');
        if (values.length > 0) cell.appendChild(document.createTextNode(' '));
        cell.appendChild(addBtn);
      }
    }).catch(function() {
      clearChildren(cell);
      cell.appendChild(createEl('span', {className: 'text-muted'}, '?'));
    });
  });
}

function addTagValueModal(el) {
  var tagName = el.getAttribute('data-tag-name');

  Modal.show({
    title: '添加预设值',
    fields: [
      { label: '标签', name: 'tag', value: tagName, readonly: true },
      { label: '新预设值', name: 'value', placeholder: '输入新值', required: true }
    ],
    saveLabel: '添加',
    saveClass: 'btn btn-primary',
    onSave: async function(values) {
      var val = values.value.trim();
      if (!val) { showToast('值不能为空', 'error'); return false; }
      var resp = await API.post('/api/tags/values', { tag_name: tagName, value: val });
      if (!resp.ok) { showToast(resp.msg || '添加失败', 'error'); return false; }
      refreshTagList();
      showToast(resp.msg || '预设值已添加', 'success');
    }
  });
}

function removeTagValue(el) {
  var tagName = el.getAttribute('data-tag-name');
  var val = el.getAttribute('data-value');
  showConfirm('确定删除值 «' + val + '»？', function() {
    var params = 'tag_name=' + encodeURIComponent(tagName) + '&value=' + encodeURIComponent(val);
    API.del('/api/tags/values/0?' + params).then(function(resp) {
      if (resp.ok) {
        refreshTagList();
        showToast(resp.msg || '预设值已删除', 'info');
      } else {
        showToast(resp.msg || '删除失败', 'error');
      }
    });
  });
}
