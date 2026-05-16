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
        html += ' <a href="#" onclick="addTagValue(this)" data-tag-name="' + escJsStr(tagName) + '" class="btn btn-ghost btn-xs">+</a>';
      }
      cell.innerHTML = html || (!cell.classList.contains('tag-tree-values') ? '<a href="#" onclick="addTagValue(this)" data-tag-name="' + escJsStr(tagName) + '" class="btn btn-ghost btn-xs">+</a>' : '');
    }).catch(function() {
      cell.innerHTML = '<span class="text-muted">?</span>';
    });
  });
}

function addTagValue(el) {
  var tagName = el.getAttribute('data-tag-name');
  var val = prompt('输入新值:');
  if (!val) return;
  API.post('/api/tags/values', {
    tag_name: tagName,
    value: val
  }).then(function(resp) {
    if (resp.ok) {
      refreshTagList();
      showToast(resp.msg || '预设值已添加', 'success');
    } else {
      showToast(resp.msg || '添加失败', 'error');
    }
  });
}

function removeTagValue(el) {
  var tagName = el.getAttribute('data-tag-name');
  var val = el.getAttribute('data-value');
  showConfirm('确定删除值 &laquo;' + escHtml(val) + '&raquo;？', function() {
    var body = 'tag_name=' + encodeURIComponent(tagName) + '&value=' + encodeURIComponent(val);
    fetch('/api/tags/values/0', {
      method: 'DELETE',
      headers: {'Content-Type': 'application/x-www-form-urlencoded'},
      body: body
    }).then(function(r) { return r.json(); }).then(function(resp) {
      if (resp.ok) {
        refreshTagList();
        showToast(resp.msg || '预设值已删除', 'info');
      } else {
        showToast(resp.msg || '删除失败', 'error');
      }
    });
  });
}
