/* ═══════════════════════════════════════════
   TagFile — Tag Page: Tree View Rendering
   ═══════════════════════════════════════════ */

function renderTreeView(treeItems) {
  if (!treeItems || treeItems.length === 0) {
    return '<div class="empty"><div class="empty-icon">&#127991;</div><div class="empty-text">暂无标签</div></div>';
  }
  return '<ul class="tree-root">' + renderTreeNodes(treeItems) + '</ul>';
}

function renderTreeNodes(nodes) {
  var html = '';
  for (var i = 0; i < nodes.length; i++) {
    var n = nodes[i];
    var tag = n.tag;
    html += '<li class="tag-tree-node" data-tag-id="' + tag.id + '" data-tag-name="' + escJsStr(tag.name) + '">';
    html += '<div class="tag-tree-row-main">';
    html += '<span class="drag-handle" style="font-size:12px;color:var(--ink-muted);cursor:grab;flex-shrink:0;opacity:0;transition:opacity 0.15s ease" title="拖拽排序">&#9776;</span>';
    html += '<span style="font-size:14px">&#127991;</span>';
    html += '<span class="tag-tree-name">' + escHtml(tag.name) + '</span>';
    html += '<span class="text-muted" style="font-size:11px">(' + escHtml(catName(tag.cid)) + ')</span>';
    html += '<span class="text-muted" style="font-size:10px;font-family:var(--font-mono)">#' + tag.id + '</span>';
    html += '<span class="tag-tree-values" data-tag-name="' + escJsStr(tag.name) + '" style="display:flex;gap:2px;flex-wrap:wrap"></span>';
    html += '<span class="tree-actions">';
    html += '<a href="#" onclick="editTagTree(this)" data-tag-id="' + tag.id + '" data-tag-name="' + escJsStr(tag.name) + '" data-cid="' + tag.cid + '" class="btn btn-default btn-xs" style="text-decoration:none">编辑</a>';
    html += '<a href="#" onclick="quickAddChild(this)" data-tag-name="' + escJsStr(tag.name) + '" data-cid="' + tag.cid + '" class="btn btn-primary btn-xs" style="text-decoration:none">+</a>';
    html += '<button class="btn btn-danger btn-xs" onclick="deleteTag(this, event)" data-tag-id="' + tag.id + '" data-tag-name="' + escJsStr(tag.name) + '">删除</button>';
    html += '</span>';
    html += '</div>';
    if (n.children && n.children.length > 0) {
      html += '<ul class="tree-children">' + renderTreeNodes(n.children) + '</ul>';
    }
    html += '</li>';
  }
  return html;
}

/* ── Tree Inline Edit / Quick Add ── */
function editTagTree(btn) {
  var tagID = parseInt(btn.getAttribute('data-tag-id'));
  var tagName = btn.getAttribute('data-tag-name');
  var categoryID = btn.getAttribute('data-cid');
  var row = btn.closest('.tag-tree-row-main');
  var nameEl = row.querySelector('.tag-tree-name');
  var existing = row.querySelector('#edit-tree-name-' + tagID);
  if (existing) return;
  nameEl.style.display = 'none';
  var input = document.createElement('input');
  input.id = 'edit-tree-name-' + tagID;
  input.value = tagName;
  input.style.cssText = 'width:100px;height:24px;font-size:12px;padding:2px 6px';
  btn.insertAdjacentElement('beforebegin', input);
  btn.textContent = '保存';
  btn.onclick = function() { saveTagTree(tagID, categoryID); return false; };
  var cancel = document.createElement('a');
  cancel.href = '#';
  cancel.className = 'btn btn-default btn-xs';
  cancel.style.textDecoration = 'none';
  cancel.textContent = '取消';
  cancel.onclick = function() { refreshTagList(); return false; };
  btn.insertAdjacentElement('afterend', cancel);
  input.focus();
}

function saveTagTree(tagID, categoryID) {
  var name = document.getElementById('edit-tree-name-' + tagID).value;
  if (!name.trim()) return;
  API.put('/api/tags/' + tagID, {
    tag_name: name.trim()
  }).then(function(resp) {
    if (resp.ok) {
      refreshTagList();
      showToast(resp.msg || '标签已更新', 'success');
    } else {
      showToast(resp.msg || '更新失败', 'error');
    }
  });
}

function quickAddChild(btn) {
  var parentName = btn.getAttribute('data-tag-name');
  var categoryID = btn.getAttribute('data-cid');
  var existing = document.getElementById('quick-add-input');
  if (existing) { existing.parentElement.previousElementSibling.remove(); existing.parentElement.remove(); }
  var span = document.createElement('span');
  span.style.cssText = 'display:inline-flex;gap:4px;align-items:center';
  span.innerHTML = '<input id="quick-add-input" placeholder="子标签名" style="width:100px;height:24px;font-size:12px;padding:2px 6px">' +
    '<button onclick="saveQuickChild()" class="btn btn-primary btn-xs">保存</button>' +
    '<button onclick="this.parentElement.previousElementSibling.remove();this.parentElement.remove()" class="btn btn-default btn-xs">取消</button>';
  btn.parentNode.insertBefore(span, btn.nextSibling);
  document.getElementById('quick-add-input').focus();
}

function saveQuickChild() {
  var input = document.getElementById('quick-add-input');
  if (!input || !input.value.trim()) return;

  var span = input.parentElement;
  if (!span) return;
  var treeActions = span.parentElement;
  if (!treeActions) return;
  var parentNode = treeActions.closest('.tag-tree-node');
  var parentName = parentNode ? parentNode.getAttribute('data-tag-name') : '';
  if (!parentName) return;

  var cid = '';
  for (var i = 0; i < allTags.length; i++) {
    if (allTags[i].name === parentName) { cid = allTags[i].cid; break; }
  }

  var body = 'category_id=' + encodeURIComponent(cid) + '&tag_name=' + encodeURIComponent(input.value.trim()) + '&parent_tag_name=' + encodeURIComponent(parentName);
  fetch('/api/tags', {
    method: 'POST',
    headers: {'Content-Type': 'application/x-www-form-urlencoded'},
    body: body
  }).then(function(r) { return r.json(); }).then(function(resp) {
    if (resp.ok) {
      refreshTagList();
      showToast(resp.msg || '子标签已添加', 'success');
    } else {
      showToast(resp.msg || '添加失败', 'error');
    }
  });
}
