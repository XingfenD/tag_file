/* ═══════════════════════════════════════════
   TagFile — Tag Page: Tree View Rendering
   ═══════════════════════════════════════════ */

function renderTreeView(treeItems) {
  if (!treeItems || treeItems.length === 0) {
    return makeEmptyState('&#127991;', '暂无标签');
  }
  var ul = createEl('ul', {className: 'tree-root'});
  renderTreeNodes(ul, treeItems);
  staggerChildren(ul);
  return ul;
}

function renderTreeNodes(parent, nodes) {
  for (var i = 0; i < nodes.length; i++) {
    var n = nodes[i];
    var tag = n.tag;

    var editBtn = createEl('a', {href: '#', 'data-tag-id': tag.id, 'data-tag-name': tag.name, 'data-cid': tag.cid, className: 'btn btn-default btn-xs', style: 'text-decoration:none', onclick: function(e) { e.preventDefault(); editTagTreeModal(this); }}, '编辑');
    var addChildBtn = createEl('a', {href: '#', 'data-tag-name': tag.name, 'data-cid': tag.cid, className: 'btn btn-primary btn-xs', style: 'text-decoration:none', onclick: function(e) { e.preventDefault(); quickAddChildModal(this); }}, '+');
    var delBtn = createEl('button', {className: 'btn btn-danger btn-xs', 'data-tag-id': tag.id, 'data-tag-name': tag.name, onclick: function(e) { deleteTag(this, e); }}, '删除');

    var row = createEl('div', {className: 'tag-tree-row-main'},
      createEl('span', {className: 'drag-handle', style: 'font-size:12px;color:var(--ink-muted);cursor:grab;flex-shrink:0;opacity:0;transition:opacity 0.15s ease', title: '拖拽排序', innerHTML: '&#9776;'}),
      createEl('span', {style: 'font-size:14px', innerHTML: '&#127991;'}),
      createEl('span', {className: 'tag-tree-name'}, tag.name),
      createEl('span', {className: 'text-muted', style: 'font-size:11px'}, '(' + catName(tag.cid) + ')'),
      createEl('span', {className: 'text-muted', style: 'font-size:10px;font-family:var(--font-mono)'}, '#' + tag.id),
      createEl('span', {className: 'tag-tree-values', 'data-tag-name': tag.name, style: 'display:flex;gap:2px;flex-wrap:wrap'}),
      createEl('span', {className: 'tree-actions'}, editBtn, addChildBtn, delBtn)
    );

    var li = createEl('li', {className: 'tag-tree-node', 'data-tag-id': tag.id, 'data-tag-name': tag.name}, row);
    if (n.children && n.children.length > 0) {
      var childUl = createEl('ul', {className: 'tree-children'});
      renderTreeNodes(childUl, n.children);
      li.appendChild(childUl);
    }
    parent.appendChild(li);
  }
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


