/* ═══════════════════════════════════════════
   TagFile — Search Page: Form, Results, Pagination, History
   ═══════════════════════════════════════════ */

var searchState = {
    include: '',
    exclude: '',
    logic: 'and',
    page: 1
};

var allTags = [];
var HISTORY_KEY = 'tagfile_search_history';
var MAX_HISTORY = 10;

/* ── Init ── */
document.addEventListener('DOMContentLoaded', function() {
    initSearchForm();
    renderSearchHistory();
    renderInitialEmptyState();
    fetchAllTags();
});

/* ── Form ── */
function initSearchForm() {
    var form = document.getElementById('search-form');
    if (!form) return;
    form.addEventListener('submit', function(e) {
        e.preventDefault();
        doSearch(1);
    });
}

function doSearch(page) {
    var includeInput = document.getElementById('include-input');
    var excludeInput = document.getElementById('exclude-input');
    var logicSelect = document.getElementById('logic-select');

    searchState.include = (includeInput ? includeInput.value : '').trim();
    searchState.exclude = (excludeInput ? excludeInput.value : '').trim();
    searchState.logic = logicSelect ? logicSelect.value : 'and';
    searchState.page = page || 1;

    if (!searchState.include && !searchState.exclude) {
        showToast('请至少输入一个搜索条件', 'error');
        return;
    }

    var btn = document.querySelector('#search-form .btn-primary');
    if (btn) { btn.disabled = true; btn.textContent = '搜索中...'; }

    var params = {
        include: searchState.include,
        exclude: searchState.exclude,
        logic: searchState.logic,
        page: searchState.page,
        page_size: 20
    };

    API.post('/api/search', params).then(function(data) {
        if (btn) { btn.disabled = false; btn.textContent = '搜索'; }
        if (!data.ok) { showToast(data.msg || '搜索失败', 'error'); return; }
        renderResults(data.results || [], data.pagination || {});
        saveSearchHistory(searchState.include, searchState.exclude, searchState.logic);
        renderSearchHistory();
    }).catch(function() {
        if (btn) { btn.disabled = false; btn.textContent = '搜索'; }
        showToast('请求失败', 'error');
    });
}

/* ── Results ── */
function renderResults(items, pg) {
    var container = document.getElementById('search-results');
    if (!container) return;
    while (container.firstChild) container.removeChild(container.firstChild);

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
    container.appendChild(makeEmptyState('🔍', '输入标签开始搜索', '支持 AND/OR 逻辑组合'));
}

/* ── Pagination helper ── */
window.doSearchPage = function(page) {
    doSearch(page);
};

/* ── Tag Autocomplete ── */
function fetchAllTags() {
    var all = [];
    function fetchPage(page) {
        API.get('/api/tags?view=flat&page=' + page + '&page_size=9999').then(function(data) {
            if (data.ok && data.items) {
                for (var i = 0; i < data.items.length; i++) {
                    var item = data.items[i];
                    all.push({name: item.name, path: item.path, cat: '', cid: item.cid, id: item.id});
                }
                if (data.pagination && data.pagination.has_next) {
                    fetchPage(page + 1);
                } else {
                    allTags = all;
                    setupAutocompletes();
                }
            } else {
                allTags = all;
                setupAutocompletes();
            }
        }).catch(function() { allTags = all; });
    }
    fetchPage(1);
}

function setupAutocompletes() {
    setupCommaAutocomplete('include-input', 'include-dropdown');
    setupCommaAutocomplete('exclude-input', 'exclude-dropdown');
}

function setupCommaAutocomplete(inputId, dropdownId) {
    var input = document.getElementById(inputId);
    if (!input) return;

    var dropdown = document.getElementById(dropdownId);
    if (!dropdown) {
        dropdown = createEl('div', {id: dropdownId, className: 'dropdown'});
        input.parentElement.appendChild(dropdown);
    }

    var highlightIdx = -1;
    var hideTimer;

    function setHighlight(idx) {
        var items = dropdown.querySelectorAll('.dropdown-item');
        if (highlightIdx >= 0 && highlightIdx < items.length) {
            items[highlightIdx].classList.remove('highlighted');
        }
        highlightIdx = idx;
        if (highlightIdx >= 0 && highlightIdx < items.length) {
            items[highlightIdx].classList.add('highlighted');
            items[highlightIdx].scrollIntoView({block: 'nearest'});
        }
    }

    function getLastSegment(val) {
        var parts = val.split(',');
        return (parts[parts.length - 1] || '').trim();
    }

    function selectItem(idx) {
        var items = dropdown.querySelectorAll('.dropdown-item');
        if (idx >= 0 && idx < items.length) {
            var item = items[idx];
            var name = item.getAttribute('data-name');
            var val = input.value;
            var parts = val.split(',');
            parts[parts.length - 1] = ' ' + name;
            input.value = parts.join(',').replace(/^,/, '').trim();
            dropdown.classList.remove('active');
            highlightIdx = -1;
            input.focus();
        }
    }

    function renderDropdown() {
        var segment = getLastSegment(input.value).toLowerCase();
        if (!segment) { dropdown.classList.remove('active'); highlightIdx = -1; return; }

        var existingTags = {};
        var parts = input.value.split(',');
        for (var i = 0; i < parts.length - 1; i++) {
            var t = parts[i].trim().toLowerCase();
            if (t) existingTags[t] = true;
        }

        var filtered = [];
        for (var i = 0; i < allTags.length; i++) {
            var t = allTags[i];
            if (existingTags[t.name.toLowerCase()]) continue;
            var name = (t.name || '').toLowerCase();
            var path = (t.path || t.name || '').toLowerCase();
            if (name.indexOf(segment) !== -1 || path.indexOf(segment) !== -1) {
                filtered.push(t);
            }
            if (filtered.length >= 10) break;
        }

        if (filtered.length === 0) { dropdown.classList.remove('active'); highlightIdx = -1; return; }

        while (dropdown.firstChild) dropdown.removeChild(dropdown.firstChild);
        for (var i = 0; i < filtered.length; i++) {
            var t = filtered[i];
            var item = createEl('div', {className: 'dropdown-item', 'data-name': t.name});
            item.appendChild(document.createTextNode((t.path || t.name) + ' '));
            item.appendChild(createEl('small', null, '(' + (t.cat || '') + ')'));
            dropdown.appendChild(item);
        }
        dropdown.classList.add('active');
        highlightIdx = -1;

        dropdown.querySelectorAll('.dropdown-item').forEach(function(el, i) {
            el.addEventListener('mouseover', function() { setHighlight(i); });
            el.addEventListener('mouseout', function() { setHighlight(-1); });
            el.addEventListener('click', function() { selectItem(i); });
        });
    }

    input.addEventListener('input', renderDropdown);
    input.addEventListener('focus', renderDropdown);
    input.addEventListener('blur', function() {
        clearTimeout(hideTimer);
        hideTimer = setTimeout(function() { dropdown.classList.remove('active'); }, 160);
    });

    input.addEventListener('keydown', function(e) {
        var items = dropdown.querySelectorAll('.dropdown-item');
        if (e.key === 'ArrowDown') {
            e.preventDefault();
            if (!dropdown.classList.contains('active')) renderDropdown();
            setHighlight(highlightIdx + 1 >= items.length ? 0 : highlightIdx + 1);
        } else if (e.key === 'ArrowUp') {
            e.preventDefault();
            if (!dropdown.classList.contains('active')) renderDropdown();
            setHighlight(highlightIdx - 1 < 0 ? items.length - 1 : highlightIdx - 1);
        } else if (e.key === 'Enter') {
            if (dropdown.classList.contains('active') && items.length > 0) {
                e.preventDefault();
                selectItem(highlightIdx >= 0 ? highlightIdx : 0);
            }
        } else if (e.key === 'Tab') {
            if (dropdown.classList.contains('active') && items.length > 0) {
                e.preventDefault();
                selectItem(highlightIdx >= 0 ? highlightIdx : 0);
            }
        } else if (e.key === 'Escape') {
            dropdown.classList.remove('active');
            highlightIdx = -1;
        }
    });
}
/* ── Search History ── */
function saveSearchHistory(include, exclude, logic) {
    if (!include && !exclude) return;
    var entry = {include: include, exclude: exclude, logic: logic};
    var history = loadHistory();

    // Deduplicate
    history = history.filter(function(h) {
        return h.include !== entry.include || h.exclude !== entry.exclude || h.logic !== entry.logic;
    });

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
    while (container.firstChild) container.removeChild(container.firstChild);

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
        var h = history[i];
        var label = buildHistoryLabel(h);
        var chip = createEl('a', {href: '#', className: 'tag', style: 'cursor:pointer;text-decoration:none;font-size:12px', onclick: function(entry) {
            return function(e) {
                e.preventDefault();
                applyHistory(entry);
            };
        }(h)}, label);
        row.appendChild(chip);
    }
    container.appendChild(row);
}

function buildHistoryLabel(entry) {
    var parts = [];
    if (entry.include) parts.push(entry.include);
    if (entry.exclude) parts.push('NOT ' + entry.exclude);
    return parts.join(' ' + (entry.logic || 'and').toUpperCase() + ' ');
}

function applyHistory(entry) {
    var includeInput = document.getElementById('include-input');
    var excludeInput = document.getElementById('exclude-input');
    var logicSelect = document.getElementById('logic-select');
    if (includeInput) includeInput.value = entry.include || '';
    if (excludeInput) excludeInput.value = entry.exclude || '';
    if (logicSelect) logicSelect.value = entry.logic || 'and';
    doSearch(1);
}
