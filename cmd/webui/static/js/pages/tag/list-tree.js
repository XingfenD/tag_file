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
    html += '<a href="#" onclick="editTagTreeModal(this);return false" data-tag-id="' + tag.id + '" data-tag-name="' + escJsStr(tag.name) + '" data-cid="' + tag.cid + '" class="btn btn-default btn-xs" style="text-decoration:none">编辑</a>';
    html += '<a href="#" onclick="quickAddChildModal(this);return false" data-tag-name="' + escJsStr(tag.name) + '" data-cid="' + tag.cid + '" class="btn btn-primary btn-xs" style="text-decoration:none">+</a>';
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

/* ── Tree Edit (Modal) ── */
window.editTagTreeModal = function(btn) {
  var tagID = parseInt(btn.getAttribute('data-tag-id'));
  var tagName = btn.getAttribute('data-tag-name');

  Modal.show({
    title: '编辑标签',
    fields: [
      { label: '标签名', name: 'tag_name', value: tagName, required: true }
    ],
    onSave: async function(values) {
      var newName = values.tag_name.trim();
      if (!newName) { showToast('标签名不能为空', 'error'); return false; }
      var resp = await API.put('/api/tags/' + tagID, { tag_name: newName });
      if (!resp.ok) { showToast(resp.msg || '更新失败', 'error'); return false; }
      refreshTagList();
      showToast(resp.msg || '标签已更新', 'success');
    }
  });
};

/* ── Quick Add Child (Modal) ── */
window.quickAddChildModal = function(btn) {
  var parentName = btn.getAttribute('data-tag-name');
  var categoryID = btn.getAttribute('data-cid');

  Modal.show({
    title: '添加子标签',
    fields: [
      { label: '父标签', name: 'parent', value: parentName, readonly: true },
      { label: '子标签名', name: 'tag_name', placeholder: '子标签名', required: true }
    ],
    saveLabel: '添加',
    saveClass: 'btn btn-primary',
    onSave: async function(values) {
      var childName = values.tag_name.trim();
      if (!childName) { showToast('标签名不能为空', 'error'); return false; }
      var resp = await API.post('/api/tags', {
        category_id: categoryID,
        tag_name: childName,
        parent_tag_name: parentName
      });
      if (!resp.ok) { showToast(resp.msg || '添加失败', 'error'); return false; }
      refreshTagList();
      showToast(resp.msg || '子标签已添加', 'success');
    }
  });
};

/* ── Legacy stubs ── */
function editTagTree(btn) {
  editTagTreeModal(btn);
}

function saveTagTree(tagID, categoryID) {
  // Removed — now handled by Modal
}

function quickAddChild(btn) {
  quickAddChildModal(btn);
}

function saveQuickChild() {
  // Removed — now handled by Modal
}
