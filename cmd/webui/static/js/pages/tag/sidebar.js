/* ═══════════════════════════════════════════
   TagFile — Tag Page: Sidebar Tree & Navigation
   ═══════════════════════════════════════════ */

function buildSidebar(tree) {
  var sideTree = document.getElementById('side-tree');
  if (!sideTree) return;

  var rootsByCat = {};
  tree.forEach(function(node) {
    var cid = node.tag.cid;
    if (!rootsByCat[cid]) rootsByCat[cid] = [];
    rootsByCat[cid].push(node);
  });

  var html = '';
  categories.forEach(function(c) {
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

/* ── Category Navigation ── */
function selectCategory(catID, event) {
  var rows = document.querySelectorAll('#side-tree .tag-tree-row');
  rows.forEach(function(r) { r.classList.remove('active'); });
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
  document.querySelectorAll('#side-tree .tag-tree-item').forEach(function(item) {
    if (!q) { item.style.display = ''; return; }
    var text = (item.textContent || '').toLowerCase();
    item.style.display = text.indexOf(q) !== -1 ? '' : 'none';
  });
}
