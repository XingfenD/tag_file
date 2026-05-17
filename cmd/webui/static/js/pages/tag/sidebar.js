/* ═══════════════════════════════════════════
   TagFile — Tag Page: Category Filter Sidebar
   ═══════════════════════════════════════════ */

function buildSidebar(tree) {
    var list = document.getElementById('side-category-list');
    if (!list) return;

    /* Count tags per category from the tree */
    var counts = {};
    function walk(items) {
        for (var i = 0; i < items.length; i++) {
            var cid = items[i].tag.cid;
            counts[cid] = (counts[cid] || 0) + 1;
            if (items[i].children) walk(items[i].children);
        }
    }
    walk(tree);

    /* Total count */
    var total = 0;
    for (var k in counts) { total += counts[k]; }

    while (list.firstChild) list.removeChild(list.firstChild);

    /* "All" item */
    var allItem = createEl('li', {className: 'side-category-item active', 'data-category-id': '', onclick: function() { selectCategory(''); }},
        createEl('span', {className: 'cat-name'}, '全部'),
        createEl('span', {className: 'cat-count'}, String(total))
    );
    list.appendChild(allItem);

    /* One item per category */
    for (var i = 0; i < categories.length; i++) {
        var c = categories[i];
        var cnt = counts[c.id] || 0;
        var item = createEl('li', {className: 'side-category-item', 'data-category-id': c.id, onclick: function(cid) { return function() { selectCategory(cid); }; }(c.id)},
            createEl('span', {className: 'cat-name'}, c.name),
            createEl('span', {className: 'cat-count'}, String(cnt))
        );
        list.appendChild(item);
    }
}

function selectCategory(catID) {
    /* Highlight active item */
    document.querySelectorAll('#side-category-list .side-category-item').forEach(function(el) {
        el.classList.toggle('active', el.getAttribute('data-category-id') === String(catID));
    });

    currentCategory = catID;
    currentQuery = '';
    currentPage = 1;

    document.getElementById('search-q').value = '';
    document.getElementById('clear-search-btn').style.display = catID ? 'inline-block' : 'none';

    refreshTagList();
}

function selectSidebarTag(el) {
    /* No-op: sidebar is now category filter only */
}
