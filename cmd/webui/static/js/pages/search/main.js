/* ═══════════════════════════════════════════
   TagFile — Search Page: Query Builder + Results
   ═══════════════════════════════════════════ */

var searchState = {page: 1};
var allTags = [];
var HISTORY_KEY = 'tagfile_search_history';
var MAX_HISTORY = 10;

/* ── Init ── */
document.addEventListener('DOMContentLoaded', function() {
    initBuilder({onTagInputCreated: wireTagAutocomplete});
    initSearchButton();
    renderSearchHistory();
    renderInitialEmptyState();
    fetchAllTags();
});

function initSearchButton() {
    var btn = document.getElementById('search-btn');
    if (btn) btn.addEventListener('click', function(e) { e.preventDefault(); doSearch(1); });
}

/* ── Search ── */
function doSearch(page) {
    searchState.page = page || 1;

    var query = buildQueryFromDOM();
    if (!query || !query.conditions || query.conditions.length === 0) {
        showToast('请至少添加一个搜索条件', 'error');
        return;
    }

    var btn = document.getElementById('search-btn');
    if (btn) { btn.disabled = true; btn.textContent = '搜索中...'; }

    var params = {
        query: JSON.stringify(query),
        page: searchState.page,
        page_size: 20
    };

    API.post('/api/search', params).then(function(data) {
        if (!data.ok) { showToast(data.msg || '搜索失败', 'error'); return; }
        renderResults(data.results || [], data.pagination || {});
        saveSearchHistory(query);
        renderSearchHistory();
    }).finally(function() {
        if (btn) { btn.disabled = false; btn.textContent = '搜索'; }
    }).catch(API.onError);
}

/* ── Results ── */
function renderResults(items, pg) {
    var container = document.getElementById('search-results');
    if (!container) return;
    clearChildren(container);

    if (!items || items.length === 0) {
        container.appendChild(makeEmptyState('🔍', '未找到匹配文件', '尝试调整搜索条件'));
        return;
    }

    var card = createEl('div', {className: 'card'});
    card.appendChild(createEl('h3', {className: 'card-title'}, '搜索结果 (' + pg.total + ' 个文件)'));

    for (var i = 0; i < items.length; i++) {
        var item = items[i];
        var f = item.file;
        var tags = item.tags || [];

        var nameLink = createEl('a', {href: '/files/detail/' + encodePath(f.path), style: 'font-weight:500;font-size:14px'}, f.name);
        var meta = createEl('div', {className: 'text-muted', style: 'font-size:12px;margin-top:4px'},
            f.path + ' · ' + formatSize(f.size) + ' · ' + (f.type || '未知')
        );

        var tagRow = createEl('div', {style: 'margin-top:8px;display:flex;flex-wrap:wrap;gap:4px'});
        for (var j = 0; j < tags.length; j++) {
            tagRow.appendChild(createEl('span', {className: 'tag'}, tags[j].tag_path));
        }

        var resultItem = createEl('div', {className: 'result-item', style: 'padding:12px 0;border-bottom:1px solid var(--border)'},
            nameLink, meta, tagRow
        );
        card.appendChild(resultItem);
    }

    container.appendChild(card);

    if (pg.total_pages > 1) {
        container.appendChild(makePagination(pg, function(p) { doSearch(p); }));
    }
}

function renderInitialEmptyState() {
    var container = document.getElementById('search-results');
    if (!container) return;
    container.appendChild(makeEmptyState('🔍', '添加条件开始搜索', '支持文件名、标签、标签值、文件大小、类型、日期组合'));
}

/* ── Tag Autocomplete ── */
function fetchAllTags() {
    API.loadAllTags().then(function(tags) {
        allTags = tags;
        /* Retroactively wire autocomplete on tag inputs created before tags loaded */
        document.querySelectorAll('.qb-tag-input, .qb-val-input').forEach(function(input) {
            var row = input.closest('.qb-row');
            if (!row) return;
            var typeSelect = row.querySelector('.qb-type');
            if (!typeSelect) return;
            var type = typeSelect.value;
            if (type !== 'tag' && type !== 'tag_value') return;
            if (input.classList.contains('qb-val-input') && type === 'tag_value') return;
            if (!input.parentElement.querySelector('.dropdown')) {
                wireTagAutocomplete(input);
            }
        });
    }).catch(function(err) {
        console.error('Failed to fetch tags:', err);
        allTags = [];
    });
}

function wireTagAutocomplete(input) {
    if (!input) return;
    var dropdown = input.parentElement.querySelector('.dropdown');
    if (!dropdown) {
        dropdown = createEl('div', {className: 'dropdown'});
        input.parentElement.style.position = 'relative';
        input.parentElement.appendChild(dropdown);
    }
    var ctrl = setupTagSearch(input, dropdown);
    if (ctrl) ctrl.setTags(allTags);
}

/* ── Search History ── */
function saveSearchHistory(query) {
    var entry = JSON.stringify(query);
    var history = loadHistory();
    history = history.filter(function(h) { return h !== entry; });
    history.unshift(entry);
    if (history.length > MAX_HISTORY) history = history.slice(0, MAX_HISTORY);
    try { localStorage.setItem(HISTORY_KEY, JSON.stringify(history)); } catch(e) {}
}

function loadHistory() {
    try {
        var raw = localStorage.getItem(HISTORY_KEY);
        return raw ? JSON.parse(raw) : [];
    } catch(e) { return []; }
}

function renderSearchHistory() {
    var container = document.getElementById('search-history');
    if (!container) return;
    clearChildren(container);

    var history = loadHistory();
    if (history.length === 0) return;

    var header = createEl('div', {style: 'display:flex;justify-content:space-between;align-items:center;margin-top:12px'},
        createEl('span', {className: 'text-muted', style: 'font-size:12px'}, '最近搜索'),
        createEl('a', {href: '#', className: 'btn-ghost', style: 'font-size:11px', onclick: function(e) {
            e.preventDefault();
            try { localStorage.removeItem(HISTORY_KEY); } catch(e) {}
            renderSearchHistory();
        }}, '清除')
    );
    container.appendChild(header);

    var row = createEl('div', {style: 'display:flex;flex-wrap:wrap;gap:6px;margin-top:6px'});
    for (var i = 0; i < history.length; i++) {
        var entry = history[i];
        var query;
        try { query = JSON.parse(entry); } catch(e) { continue; }
        var summary = queryToSummary(query) || '...';
        var chip = createEl('a', {href: '#', className: 'tag', style: 'cursor:pointer;text-decoration:none;font-size:12px',
            onclick: function(q) { return function(e) { e.preventDefault(); restoreQueryToDOM(q); doSearch(1); }; }(query)
        }, summary);
        row.appendChild(chip);
    }
    container.appendChild(row);
}
