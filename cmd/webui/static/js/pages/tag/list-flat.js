/* ═══════════════════════════════════════════
   TagFile — Tag Page: Flat View Rendering
   ═══════════════════════════════════════════ */

function renderFlatView(items, pagination) {
  if (!items || items.length === 0) {
    return '<div class="empty"><div class="empty-icon">&#127991;</div><div class="empty-text">暂无标签</div></div>';
  }

  var html = '<div class="table-wrap"><table><thead><tr>';
  html += '<th>ID</th><th>标签名</th><th>所属分类</th><th>父标签</th><th>预设值</th><th>描述</th><th style="width:100px">操作</th>';
  html += '</tr></thead><tbody>';

  for (var i = 0; i < items.length; i++) {
    var item = items[i];
    var parentPath = getParentFromPath(item.path, item.name);
    html += '<tr id="tag-' + item.id + '">';
    html += '<td class="col-id">' + item.id + '</td>';
    html += '<td class="tag-name" style="font-weight:500">' + escHtml(item.name) + '</td>';
    html += '<td data-category-id="' + item.cid + '">' + escHtml(catName(item.cid)) + '</td>';
    html += '<td class="tag-parent text-muted">' + (parentPath ? escHtml(parentPath) : '&mdash;') + '</td>';
    html += '<td class="tag-values" data-tag-name="' + escJsStr(item.name) + '">';
    html += '<span class="tag-values-loading text-muted">...</span>';
    html += '</td>';
    html += '<td class="tag-desc">&mdash;</td>';
    html += '<td>';
    html += '<span class="col-actions-inner">';
    html += '<button class="btn btn-default btn-sm" onclick="editTag(this)" data-tag-id="' + item.id + '" data-cid="' + item.cid + '" data-tag-name="' + escJsStr(item.name) + '" data-desc="" data-parent="' + escJsStr(parentPath) + '">编辑</button>';
    html += '<button class="btn btn-danger btn-sm" onclick="deleteTag(this, event)" data-tag-id="' + item.id + '" data-tag-name="' + escJsStr(item.name) + '">删除</button>';
    html += '</span>';
    html += '</td>';
    html += '</tr>';
  }

  html += '</tbody></table></div>';

  if (pagination && pagination.total_pages > 1) {
    html += renderPagination(pagination);
  }

  return html;
}

function renderPagination(pg) {
  var html = '<div class="pagination">';
  if (pg.has_prev) {
    html += '<a href="#" onclick="goToPage(' + pg.prev_page + ');return false">&laquo; 前页</a>';
  } else {
    html += '<span class="disabled">&laquo; 前页</span>';
  }
  html += '<span style="font-size:12px;color:var(--ink-muted)">第 ' + pg.page + '/' + pg.total_pages + ' 页 (' + pg.total + ' 项)</span>';
  if (pg.has_next) {
    html += '<a href="#" onclick="goToPage(' + pg.next_page + ');return false">后页 &raquo;</a>';
  } else {
    html += '<span class="disabled">后页 &raquo;</span>';
  }
  html += '</div>';
  return html;
}

function getParentFromPath(path, name) {
  if (!path || path === name) return '';
  var idx = path.lastIndexOf(' / ');
  if (idx === -1) return '';
  return path.substring(0, idx);
}
