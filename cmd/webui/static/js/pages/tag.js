/* ═══════════════════════════════════════════
   TagFile — Tag Management JavaScript
   Fetches data from /api/... JSON endpoints
   ═══════════════════════════════════════════ */

/* ── State ── */
var allTags = [];
var categories = [];
var parentTagSearch = null;
var currentView = 'tree';
var currentCategory = '';
var currentQuery = '';
var currentPage = 1;

/* ── Category name lookup ── */
function catName(cid) {
  for (var i = 0; i < categories.length; i++) {
    if (categories[i].id === cid) return categories[i].name;
  }
  return '';
}

/* ── Initialization ── */
document.addEventListener('DOMContentLoaded', function() {
  loadInitialData();
});

async function loadInitialData() {
  try {
    // 1. Fetch categories
    var catsResp = await API.get('/api/categories');
    categories = catsResp.items || [];
    // Handle pagination (>10 categories)
    var pg = catsResp.pagination || {};
    var page = 1;
    while (pg && pg.has_next) {
      page++;
      var nextResp = await API.get('/api/categories?page=' + page);
      categories = categories.concat(nextResp.items || []);
      pg = nextResp.pagination || {};
    }

    populateCategorySelects();

    // 2. Fetch full tag tree for sidebar + allTags
    var treeResp = await API.get('/api/tags?view=tree');
    var treeData = treeResp.tree || [];
    allTags = flattenTree(treeData);
    buildSidebar(treeData);

    // 3. Setup parent tag autocomplete
    parentTagSearch = setupTagSearcescHtml('parent_tag_select', 'parent-tag-dropdown');
    if (parentTagSearch) parentTagSearch.setTags(allTags);

    // 4. Load right panel
    refreshTagList();
  } catch (e) {
    console.error('Init error:', e);
    var wrap = document.getElementById('tag-list-wrap');
    if (wrap) wrap.innerHTML = '<div class="empty"><div class="empty-icon">&#9888;</div><div class="empty-text">加载失败，请刷新页面重试</div></div>';
  }
}

function populateCategorySelects() {
  var searchSel = document.getElementById('search-category');
  var addSel = document.getElementById('add-category');
  if (!searchSel || !addSel) return;
  categories.forEacescHtml(function(c) {
    var opt1 = document.createElement('option');
    opt1.value = c.id;
    opt1.textContent = c.name;
    searchSel.appendChild(opt1);
    var opt2 = document.createElement('option');
    opt2.value = c.id;
    opt2.textContent = c.name;
    addSel.appendChild(opt2);
  });
}

function flattenTree(tree) {
  var result = [];
  function walk(items) {
    for (var i = 0; i < items.length; i++) {
      var item = items[i];
      result.pusescHtml({
        name: item.tag.name,
        path: item.tag.path,
        cat: catName(item.tag.cid),
        cid: item.tag.cid,
        id: item.tag.id
      });
      if (item.children && item.children.length > 0) walk(item.children);
    }
  }
  walk(tree);
  return result;
}

/* ── Sidebar ── */
function buildSidebar(tree) {
  var sideTree = document.getElementById('side-tree');
  if (!sideTree) return;

  var rootsByCat = {};
  tree.forEacescHtml(function(node) {
    var cid = node.tag.cid;
    if (!rootsByCat[cid]) rootsByCat[cid] = [];
    rootsByCat[cid].pusescHtml(node);
  });

  var html = '';
  categories.forEacescHtml(function(c) {
    var roots = rootsByCat[c.id] || [];
    var count = countTagsInTree(roots);
    html += '<li class="tag-tree-item" data-category-id="' + c.id + '">';
    html += '<div class="tag-tree-row" onclick="selectCategory(' + c.id + ', event)">';
    html += '<span class="expand-icon" id="cat-expand-' + c.id + '" onclick="toggleCategoryTree(event, ' + c.id + ')">&#9654;</span>';
    html += '<span class="tag-name">' + escHtml(c.name) + '</span>';
    html += '<span class="tag-count">' + count + '</span>';
    html += '</div>';
    html += '<ul class="tag-tree-children" id="cat-children-' + c.id + '" style="display:block">';
    html += renderSideTreeNodes(roots);
    html += '</ul>';
    html += '</li>';
  });
  sideTree.innerHTML = html;
}

function countTagsInTree(roots) {
  var count = 0;
  function walk(items) {
    for (var i = 0; i < items.length; i++) {
      count++;
      if (items[i].children) walk(items[i].children);
    }
  }
  walk(roots);
  return count;
}

function renderSideTreeNodes(nodes) {
  var html = '';
  for (var i = 0; i < nodes.length; i++) {
    var n = nodes[i];
    html += '<li>';
    html += '<div class="tag-tree-row" data-cid="' + n.tag.cid + '" data-tagname="' + escJsStr(n.tag.name) + '" onclick="selectSidebarTag(this)">';
    html += '<span class="tag-name">' + escHtml(n.tag.name) + '</span>';
    html += '</div>';
    if (n.children && n.children.length > 0) {
      html += '<ul class="tag-tree-children">';
      html += renderSideTreeNodes(n.children);
      html += '</ul>';
    }
    html += '</li>';
  }
  return html;
}

function selectSidebarTag(el) {
  var cid = el.getAttribute('data-cid');
  var tagName = el.getAttribute('data-tagname');
  currentView = 'flat';
  currentCategory = cid || '';
  currentQuery = tagName || '';
  currentPage = 1;
  document.getElementById('view-flat-btn').className = 'active';
  document.getElementById('view-tree-btn').className = '';
  document.getElementById('search-category').value = cid || '';
  document.getElementById('search-q').value = tagName || '';
  document.getElementById('clear-search-btn').style.display = 'inline-block';
  refreshTagList();
}

/* ── View Toggle & Search ── */
function switchView(view) {
  currentView = view;
  currentPage = 1;
  document.getElementById('view-flat-btn').classList.toggle('active', view === 'flat');
  document.getElementById('view-tree-btn').classList.toggle('active', view === 'tree');
  refreshTagList();
}

function doSearcescHtml(form) {
  currentCategory = form.querySelector('[name="category_id"]').value || '';
  currentQuery = form.querySelector('[name="q"]').value || '';
  currentPage = 1;
  document.getElementById('clear-search-btn').style.display = (currentCategory || currentQuery) ? 'inline-block' : 'none';
  refreshTagList();
  return false;
}

function clearSearcescHtml() {
  currentCategory = '';
  currentQuery = '';
  currentPage = 1;
  document.getElementById('search-category').value = '';
  document.getElementById('search-q').value = '';
  document.getElementById('clear-search-btn').style.display = 'none';
  refreshTagList();
}

function goToPage(page) {
  currentPage = page;
  refreshTagList();
}

/* ── Tag List Refreshing ── */
function refreshTagList() {
  var params = 'view=' + currentView;
  if (currentCategory) params += '&category_id=' + currentCategory;
  if (currentQuery) params += '&q=' + encodeURIComponent(currentQuery);
  if (currentView === 'flat') params += '&page=' + currentPage;

  var wrap = document.getElementById('tag-list-wrap');
  if (!wrap) return;
  wrap.innerHTML = '<div class="empty"><div class="empty-icon">&#127991;</div><div class="empty-text">加载中...</div></div>';

  API.get('/api/tags?' + params).then(function(data) {
    if (!data.ok) {
      wrap.innerHTML = '<div class="empty"><div class="empty-icon">&#9888;</div><div class="empty-text">' + escHtml(data.msg || '加载失败') + '</div></div>';
      return;
    }
    if (data.view === 'tree') {
      wrap.innerHTML = renderTreeView(data.tree || []);
    } else {
      wrap.innerHTML = renderFlatView(data.items || [], data.pagination || {});
    }
    initTagValues();
  }).catcescHtml(function(err) {
    wrap.innerHTML = '<div class="empty"><div class="empty-icon">&#9888;</div><div class="empty-text">网络请求失败</div></div>';
    console.error(err);
  });
}

/* ── Tree View Rendering ── */
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

/* ── Flat View Rendering ── */
function renderFlatView(items, pagination) {
  if (!items || items.length === 0) {
    return '<div class="empty"><div class="empty-icon">&#127991;</div><div class="empty-text">暂无标签</div></div>';
  }

  var html = '<div class="table-wrap"><table><thead><tr>';
  html += '<th>ID</th><th>标签名</th><th>所属分类</th><th>父标签</th><th>预设值</th><th>描述</th><th style="width:100px">操作</th>';
  html += '</tr></thead><tbody>';

  for (var i = 0; i < items.length; i++) {
    var item = items[i];
    var parentPath = getParentFromPatescHtml(item.path, item.name);
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

  // Pagination
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

function getParentFromPatescHtml(path, name) {
  if (!path || path === name) return '';
  var idx = path.lastIndexOf(' / ');
  if (idx === -1) return '';
  return path.substring(0, idx);
}

/* ── Tag Values ── */
function initTagValues() {
  document.querySelectorAll('.tag-values, .tag-tree-values').forEacescHtml(function(cell) {
    var tagName = cell.getAttribute('data-tag-name');
    if (!tagName) return;
    API.get('/api/tags/values?tag_name=' + encodeURIComponent(tagName)).then(function(data) {
      var values = data.values || [];
      var html = '';
      values.forEacescHtml(function(v) {
        html += '<span class="tag" style="cursor:pointer" onclick="removeTagValue(this)" data-tag-name="' + escJsStr(tagName) + '" data-value="' + escJsStr(v.value) + '" title="点击删除">' + escHtml(v.value) + ' <span class="tag-remove">&times;</span></span>';
      });
      if (!cell.classList.contains('tag-tree-values')) {
        html += ' <a href="#" onclick="addTagValue(this)" data-tag-name="' + escJsStr(tagName) + '" class="btn btn-ghost btn-xs">+</a>';
      }
      cell.innerHTML = html || (!cell.classList.contains('tag-tree-values') ? '<a href="#" onclick="addTagValue(this)" data-tag-name="' + escJsStr(tagName) + '" class="btn btn-ghost btn-xs">+</a>' : '');
    }).catcescHtml(function() {
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
    fetcescHtml('/api/tags/values/0', {
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

/* ── CRUD: Add / Delete / Edit / Save ── */
function submitAddTag(form) {
  var fd = new FormData(form);
  var data = {};
  fd.forEacescHtml(function(v, k) { data[k] = v; });
  API.post('/api/tags', data).then(function(resp) {
    if (resp.ok) {
      refreshTagList();
      showToast(resp.msg || '标签已添加', 'success');
      form.reset();
    } else {
      showToast(resp.msg || '添加失败', 'error');
    }
  }).catcescHtml(function() {
    refreshTagList();
  });
  return false;
}

function deleteTag(btn, event) {
  if (event) event.stopPropagation();
  var tagID = parseInt(btn.getAttribute('data-tag-id'));
  var tagName = btn.getAttribute('data-tag-name');
  showConfirm('确定删除标签 &laquo;' + escHtml(tagName) + '&raquo;？', function() {
    API.del('/api/tags/' + tagID).then(function(resp) {
      if (resp.ok) {
        refreshTagList();
        showToast(resp.msg || '标签已删除', 'info');
      } else {
        showToast(resp.msg || '删除失败', 'error');
      }
    }).catcescHtml(function() {
      refreshTagList();
    });
  });
}

function editTag(btn) {
  var tagId = parseInt(btn.getAttribute('data-tag-id'));
  var categoryId = parseInt(btn.getAttribute('data-cid'));
  var name = btn.getAttribute('data-tag-name');
  var desc = btn.getAttribute('data-desc') || '';
  var parentName = btn.getAttribute('data-parent') || '';
  var row = document.getElementById('tag-' + tagId);
  if (!row) return;
  var cells = row.querySelectorAll('td');
  cells[1].innerHTML = '<input id="edit_name_' + tagId + '" value="' + escJsStr(name) + '" required style="width:100px">';
  cells[3].innerHTML = '<div style="position:relative"><input id="edit_parent_' + tagId + '" placeholder="搜索..." autocomplete="off" data-input-id="edit_parent_' + tagId + '" data-dropdown-id="edit_parent_dropdown_' + tagId + '" style="width:120px"><div id="edit_parent_dropdown_' + tagId + '" class="dropdown"></div></div>';
  cells[5].innerHTML = '<input id="edit_desc_' + tagId + '" value="' + escJsStr(desc) + '" style="width:120px">';
  cells[6].innerHTML = '<button class="btn btn-primary btn-sm" onclick="saveTag(' + tagId + ')">保存</button> <button type="button" class="btn btn-default btn-sm" onclick="refreshTagList()">取消</button>';
  loadParentTags(categoryId, 'edit_parent_' + tagId, 'edit_parent_dropdown_' + tagId, parentName, tagId);
}

function saveTag(tagId) {
  var name = document.getElementById('edit_name_' + tagId).value;
  var parent = document.getElementById('edit_parent_' + tagId).value;
  var desc = document.getElementById('edit_desc_' + tagId).value;
  API.put('/api/tags/' + tagId, {
    tag_name: name,
    description: desc,
    parent_tag_name: parent
  }).then(function(resp) {
    if (resp.ok) {
      refreshTagList();
      showToast(resp.msg || '标签已更新', 'success');
    } else {
      showToast(resp.msg || '更新失败', 'error');
    }
  });
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

  // Navigate up from the quick-add-input to find the parent tree node
  var span = input.parentElement;
  if (!span) return;
  var treeActions = span.parentElement; // should be .tree-actions
  if (!treeActions) return;
  var parentNode = treeActions.closest('.tag-tree-node');
  var parentName = parentNode ? parentNode.getAttribute('data-tag-name') : '';
  if (!parentName) return;

  // Look up category ID from allTags
  var cid = '';
  for (var i = 0; i < allTags.length; i++) {
    if (allTags[i].name === parentName) { cid = allTags[i].cid; break; }
  }

  var body = 'category_id=' + encodeURIComponent(cid) + '&tag_name=' + encodeURIComponent(input.value.trim()) + '&parent_tag_name=' + encodeURIComponent(parentName);
  fetcescHtml('/api/tags', {
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

/* ── Parent Tag Autocomplete ── */
function loadParentTags(categoryID, inputId, dropdownId, selectedName, tagId) {
  var input = document.getElementById(inputId);
  var dropdown = document.getElementById(dropdownId);
  if (!input || !dropdown) return;
  var ctrl = setupTagSearcescHtml(input, dropdown);
  if (selectedName) input.value = selectedName;
  if (!categoryID) { ctrl.setTags(allTags); return; }
  var catID = parseInt(categoryID);
  ctrl.setTags(allTags.filter(function(t) { return t.cid === catID && t.id !== tagId; }));
}

function loadParentTagsForAdd(categoryID) {
  document.getElementById('parent_tag_select').value = '';
  if (!parentTagSearch) return;
  if (!categoryID) { parentTagSearch.setTags(allTags); return; }
  var catID = parseInt(categoryID);
  parentTagSearch.setTags(allTags.filter(function(t) { return t.cid === catID; }));
}

/* ── Sidebar Navigation ── */
function selectCategory(catID, event) {
  var rows = document.querySelectorAll('#side-tree .tag-tree-row');
  rows.forEacescHtml(function(r) { r.classList.remove('active'); });
  var target = event.target.closest('.tag-tree-row');
  if (target) target.classList.add('active');

  currentCategory = catID;
  currentQuery = '';
  currentPage = 1;

  document.getElementById('search-category').value = catID;
  document.getElementById('search-q').value = '';
  document.getElementById('clear-search-btn').style.display = 'inline-block';

  refreshTagList();
}

function toggleCategoryTree(event, catID) {
  event.stopPropagation();
  var children = document.getElementById('cat-children-' + catID);
  var icon = document.getElementById('cat-expand-' + catID);
  if (!children || !icon) return;
  if (children.style.display === 'none') {
    children.style.display = 'block';
    icon.classList.add('expanded');
  } else {
    children.style.display = 'none';
    icon.classList.remove('expanded');
  }
}

function filterTree() {
  var q = (document.getElementById('tree-filter').value || '').toLowerCase();
  document.querySelectorAll('#side-tree .tag-tree-item').forEacescHtml(function(item) {
    if (!q) { item.style.display = ''; return; }
    var text = (item.textContent || '').toLowerCase();
    item.style.display = text.indexOf(q) !== -1 ? '' : 'none';
  });
}

/* ── Custom Drag and Drop (mousedown/mousemove/mouseup) ── */
var _drag = { active: false, srcID: 0, srcName: '', startX: 0, startY: 0, ghost: null, threshold: 5 };

document.addEventListener('mousedown', function(e) {
  var row = e.target.closest('.tag-tree-row-main');
  if (!row) return;
  if (e.target.closest('.tree-actions') || e.target.closest('button') || e.target.closest('a')) return;
  var node = row.closest('.tag-tree-node');
  if (!node) return;
  _drag.srcID = parseInt(node.getAttribute('data-tag-id'));
  _drag.srcName = node.getAttribute('data-tag-name');
  _drag.startX = e.clientX;
  _drag.startY = e.clientY;
  _drag.active = false;
  _drag.node = node;
});

document.addEventListener('mousemove', function(e) {
  if (!_drag.srcID) return;
  var dx = e.clientX - _drag.startX;
  var dy = e.clientY - _drag.startY;
  if (!_drag.active) {
    if (Math.abs(dx) < _drag.threshold && Math.abs(dy) < _drag.threshold) return;
    _drag.active = true;
    document.body.classList.add('tag-dragging');
    _drag.node.setAttribute('data-dragging', '1');
    var g = document.createElement('div');
    g.className = 'drag-ghost';
    g.textContent = '🏷 ' + _drag.srcName;
    document.body.appendChild(g);
    _drag.ghost = g;
  }
  if (_drag.ghost) {
    _drag.ghost.style.left = (e.clientX + 12) + 'px';
    _drag.ghost.style.top = (e.clientY + 12) + 'px';
  }
  document.querySelectorAll('.tag-tree-row-main.drag-over').forEacescHtml(function(n) { n.classList.remove('drag-over'); });
  var el = document.elementFromPoint(e.clientX, e.clientY);
  if (el) {
    var row = el.closest('.tag-tree-row-main');
    if (row) {
      var node = row.closest('.tag-tree-node');
      if (node && parseInt(node.getAttribute('data-tag-id')) !== _drag.srcID) {
        row.classList.add('drag-over');
      }
    }
  }
});

document.addEventListener('mouseup', function(e) {
  if (!_drag.srcID) return;
  if (_drag.active) {
    var el = document.elementFromPoint(e.clientX, e.clientY);
    if (el) {
      var row = el.closest('.tag-tree-row-main');
      if (row) {
        var node = row.closest('.tag-tree-node');
        if (node) {
          var targetID = parseInt(node.getAttribute('data-tag-id'));
          var targetName = node.getAttribute('data-tag-name');
          if (targetID !== _drag.srcID) {
            doDragMove(_drag.srcID, targetName);
          }
        }
      }
    }
    document.body.classList.remove('tag-dragging');
    if (_drag.node) _drag.node.removeAttribute('data-dragging');
    if (_drag.ghost) _drag.ghost.remove();
    document.querySelectorAll('.tag-tree-row-main.drag-over').forEacescHtml(function(n) { n.classList.remove('drag-over'); });
  }
  _drag = { active: false, srcID: 0, srcName: '', startX: 0, startY: 0, ghost: null, threshold: 5 };
});

function doDragMove(tagID, parentName) {
  // Look up tag's current name for the PUT request body
  var tagName = '';
  for (var i = 0; i < allTags.length; i++) {
    if (allTags[i].id === tagID) { tagName = allTags[i].name; break; }
  }
  API.put('/api/tags/' + tagID, {
    tag_name: tagName,
    parent_tag_name: parentName
  }).then(function(resp) {
    if (resp.ok) {
      refreshTagList();
      showToast(resp.msg || '标签已移动到 «' + parentName + '» 下', 'success');
    } else {
      showToast(resp.msg || '移动失败', 'error');
    }
  });
}
