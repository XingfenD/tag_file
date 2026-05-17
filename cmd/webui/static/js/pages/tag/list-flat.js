/* ═══════════════════════════════════════════
   TagFile — Tag Page: Flat View Rendering
   ═══════════════════════════════════════════ */

function renderFlatView(items, pagination) {
  if (!items || items.length === 0) {
    return makeEmptyState('&#127991;', '暂无标签');
  }

  var tbody = createEl('tbody');
  for (var i = 0; i < items.length; i++) {
    var item = items[i];
    var parentPath = getParentFromPath(item.path, item.name);
    var editBtn = createEl('button', {className: 'btn btn-default btn-sm', 'data-tag-id': item.id, 'data-cid': item.cid, 'data-tag-name': item.name, 'data-desc': '', 'data-parent': parentPath, onclick: function() { editTag(this); }}, '编辑');
    var delBtn = createEl('button', {className: 'btn btn-danger btn-sm', 'data-tag-id': item.id, 'data-tag-name': item.name, onclick: function(e) { deleteTag(this, e); }}, '删除');
    var tr = createEl('tr', {id: 'tag-' + item.id},
      createEl('td', {className: 'col-id'}, String(item.id)),
      createEl('td', {className: 'tag-name', style: 'font-weight:500'}, item.name),
      createEl('td', {'data-category-id': item.cid}, catName(item.cid)),
      createEl('td', {className: 'tag-parent text-muted'}, parentPath || '—'),
      createEl('td', {className: 'tag-values', 'data-tag-name': item.name}, createEl('span', {className: 'tag-values-loading text-muted'}, '...')),
      createEl('td', {className: 'tag-desc'}, '—'),
      createEl('td', null, createEl('span', {className: 'col-actions-inner'}, editBtn, delBtn))
    );
    tbody.appendChild(tr);
  }

  var frag = document.createDocumentFragment();
  var tableWrap = createEl('div', {className: 'table-wrap'},
    createEl('table', null,
      createEl('thead', null, createEl('tr', null,
        createEl('th', null, 'ID'),
        createEl('th', null, '标签名'),
        createEl('th', null, '所属分类'),
        createEl('th', null, '父标签'),
        createEl('th', null, '预设值'),
        createEl('th', null, '描述'),
        createEl('th', {style: 'width:100px'}, '操作')
      )),
      tbody
    )
  );
  frag.appendChild(tableWrap);

  if (pagination && pagination.total_pages > 1) {
    frag.appendChild(renderPagination(pagination));
  }

  return frag;
}

function renderPagination(pg) {
  return makePagination(pg, goToPage, {prevText: '« 前页', nextText: '后页 »', showDisabled: true});
}

function getParentFromPath(path, name) {
  if (!path || path === name) return '';
  var idx = path.lastIndexOf(' / ');
  if (idx === -1) return '';
  return path.substring(0, idx);
}
