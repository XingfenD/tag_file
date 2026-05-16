/* ═══════════════════════════════════════════
   TagFile — Tag Page: Tag Value CRUD
   ═══════════════════════════════════════════ */

function initTagValues() {
  document.querySelectorAll('.tag-values, .tag-tree-values').forEach(function(cell) {
    var tagName = cell.getAttribute('data-tag-name');
    if (!tagName) return;
    API.get('/api/tags/values?tag_name=' + encodeURIComponent(tagName)).then(function(data) {
      var values = data.values || [];
      var html = '';
      values.forEach(function(v) {
        html += '<span class="tag" style="cursor:pointer" onclick="removeTagValue(this)" data-tag-name="' + escJsStr(tagName) + '" data-value="' + escJsStr(v.value) + '" title="点击删除">' + escHtml(v.value) + ' <span class="tag-remove">&times;</span></span>';
      });
      if (!cell.classList.contains('tag-tree-values')) {
        html += ' <a href="#" onclick="addTagValueModal(this);return false" data-tag-name="' + escJsStr(tagName) + '" class="btn btn-ghost btn-xs">+</a>';
      }
      cell.innerHTML = html || (!cell.classList.contains('tag-tree-values') ? '<a href="#" onclick="addTagValueModal(this);return false" data-tag-name="' + escJsStr(tagName) + '" class="btn btn-ghost btn-xs">+</a>' : '');
    }).catch(function() {
      cell.innerHTML = '<span class="text-muted">?</span>';
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

/* ── Legacy ── */
function addTagValue(el) {
  addTagValueModal(el);
}

function removeTagValue(el) {
  var tagName = el.getAttribute('data-tag-name');
  var val = el.getAttribute('data-value');
  showConfirm('确定删除值 &laquo;' + escHtml(val) + '&raquo;？', function() {
    var params = 'tag_name=' + encodeURIComponent(tagName) + '&value=' + encodeURIComponent(val);
    fetch('/api/tags/values/0?' + params, {method:'DELETE'}).then(function(r) { return r.json(); }).then(function(resp) {
      if (resp.ok) {
        refreshTagList();
        showToast(resp.msg || '预设值已删除', 'info');
      } else {
        showToast(resp.msg || '删除失败', 'error');
      }
    });
  });
}
