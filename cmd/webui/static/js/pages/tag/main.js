/* ═══════════════════════════════════════════
   TagFile — Tag Page: State, Init, Refresh
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
    var catsResp = await API.get('/api/categories');
    categories = catsResp.items || [];
    var pg = catsResp.pagination || {};
    var page = 1;
    while (pg && pg.has_next) {
      page++;
      var nextResp = await API.get('/api/categories?page=' + page);
      categories = categories.concat(nextResp.items || []);
      pg = nextResp.pagination || {};
    }

    populateCategorySelects();

    var treeResp = await API.get('/api/tags?view=tree');
    var treeData = treeResp.tree || [];
    allTags = flattenTree(treeData);
    buildSidebar(treeData);

    parentTagSearch = setupTagSearch('parent_tag_select', 'parent-tag-dropdown');
    if (parentTagSearch) parentTagSearch.setTags(allTags);

    /* Wire up forms (replaces onsubmit attributes) */
    var searchForm = document.getElementById('tag-search-form');
    if (searchForm) {
      searchForm.addEventListener('submit', function(e) {
        e.preventDefault();
        doSearch(this);
      });
    }
    var addForm = document.getElementById('add-tag-form-inline');
    if (addForm) {
      addForm.addEventListener('submit', function(e) {
        e.preventDefault();
        submitAddTag(this);
      });
    }

    refreshTagList();
  } catch (e) {
    console.error('Init error:', e);
    var wrap = document.getElementById('tag-list-wrap');
    if (wrap) {
      while (wrap.firstChild) wrap.removeChild(wrap.firstChild);
      wrap.appendChild(makeEmptyState('&#9888;', '加载失败，请刷新页面重试'));
    }
  }
}

function populateCategorySelects() {
  var searchSel = document.getElementById('search-category');
  var addSel = document.getElementById('add-category');
  if (!searchSel || !addSel) return;
  categories.forEach(function(c) {
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
      result.push({
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

/* ── View Toggle & Search ── */
function switchView(view) {
  currentView = view;
  currentPage = 1;
  document.getElementById('view-flat-btn').classList.toggle('active', view === 'flat');
  document.getElementById('view-tree-btn').classList.toggle('active', view === 'tree');
  refreshTagList();
}

function doSearch(form) {
  currentCategory = form.querySelector('[name="category_id"]').value || '';
  currentQuery = form.querySelector('[name="q"]').value || '';
  currentPage = 1;
  document.getElementById('clear-search-btn').style.display = (currentCategory || currentQuery) ? 'inline-block' : 'none';
  refreshTagList();
}

function clearSearch() {
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
  while (wrap.firstChild) wrap.removeChild(wrap.firstChild);
  wrap.appendChild(makeEmptyState('&#127991;', '加载中...'));

  API.get('/api/tags?' + params).then(function(data) {
    while (wrap.firstChild) wrap.removeChild(wrap.firstChild);
    if (!data.ok) {
      wrap.appendChild(makeEmptyState('&#9888;', data.msg || '加载失败'));
      return;
    }
    if (data.view === 'tree') {
      wrap.appendChild(renderTreeView(data.tree || []));
    } else {
      wrap.appendChild(renderFlatView(data.items || [], data.pagination || {}));
    }
    initTagValues();
  }).catch(function(err) {
    while (wrap.firstChild) wrap.removeChild(wrap.firstChild);
    wrap.appendChild(makeEmptyState('&#9888;', '网络请求失败'));
    console.error(err);
  });
}
