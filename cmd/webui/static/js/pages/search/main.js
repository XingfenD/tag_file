/* ═══════════════════════════════════════════
   TagFile — Search Page (Alpine.js)
   Query Builder + Results + History
   ═══════════════════════════════════════════ */

var CONDITION_TYPES = {
    filename:  {label: '文件名',   ops: [{v:'contains',l:'包含'},{v:'not_contains',l:'不包含'},{v:'equals',l:'等于'}]},
    tag:       {label: '标签',     ops: [{v:'has',l:'包含'},{v:'not_has',l:'不包含'}]},
    tag_value: {label: '标签值',   ops: [{v:'has',l:'包含'},{v:'not_has',l:'不包含'}]},
    size:      {label: '文件大小', ops: [{v:'gt',l:'大于'},{v:'gte',l:'大于等于'},{v:'lt',l:'小于'},{v:'lte',l:'小于等于'}]},
    ext:       {label: '文件类型', ops: [{v:'in',l:'属于'},{v:'not_in',l:'不属于'}]},
    date:      {label: '修改日期', ops: [{v:'gt',l:'晚于'},{v:'gte',l:'不早于'},{v:'lt',l:'早于'},{v:'lte',l:'不晚于'}]}
};

var SIZE_UNITS = [
    {v:1, l:'B'}, {v:1024, l:'KB'}, {v:1048576, l:'MB'}, {v:1073741824, l:'GB'}
];

var HISTORY_KEY = 'tagfile_search_history';

/* 搜索页取数信封：数据键是 results、loading 翻转走 searching，
   成功后才记 searched 并落历史（两者都读 this._lastQuery）。 */
var SEARCH_SPEC = {
  fetch: function(self) {
    return API.post('/api/search', { query: JSON.stringify(self._lastQuery), page: self.page, page_size: 20 });
  },
  itemsKey: 'results',
  dataKey: 'results',
  loadingKey: 'searching',
  errMsg: '搜索失败',
  after: function(self) { self.searched = true; self.saveHistory(self._lastQuery); }
};
var MAX_HISTORY = 10;

document.addEventListener('alpine:init', function() {
  Alpine.data('searchPage', function() {
    /* Non-reactive value store keyed by row object:
       keystrokes must not trigger builder re-render, and deletions
       must not corrupt values of remaining rows. */
    var rowVals = new WeakMap();

    var self = this;

    function newRow() {
      var row = { type: 'filename', op: 'contains' };
      rowVals.set(row, {});
      return row;
    }
    function newGroup() {
      return { logic: 'and', conditions: [newRow()] };
    }

    return {
      root: newGroup(),
      allTags: [],
      searching: false,
      searched: false,
      results: [],
      pagination: {},
      history: [],

      init: function() {
        var self = this;
        /* 事件委托单源 delegate（x-html 生成的行没有 Alpine 作用域）；
           onBuilderInput 是复合选择器方言（data-path+data-field），不在此列 */
        delegate(this.$el, 'click', {
          'toggle-logic': function(el) {
            var g = self.resolveGroup(el.getAttribute('data-path') || '');
            g.logic = g.logic === 'and' ? 'or' : 'and';
          },
          'add-cond': function(el) {
            var path = el.getAttribute('data-path') || '';
            self.addRowTo(self.resolveGroup(path), path);
          },
          'add-group': function() {
            self.root.conditions.push(newGroup());
          },
          'del-row': function(el) {
            var path = el.getAttribute('data-path') || '';
            var row = self.getRow(path);
            if (!row) return;
            rowVals.delete(row);
            var parts = path.split('/');
            var parent = parts.length === 1 ? self.root : self.resolveGroup(parts.slice(0, -1).join('/'));
            parent.conditions.splice(parseInt(parts[parts.length - 1]), 1);
          },
          'del-group': function(el) {
            var path = el.getAttribute('data-path') || '';
            var parts = path.split('/');
            var parent = parts.length === 1 ? self.root : self.resolveGroup(parts.slice(0, -1).join('/'));
            var removed = parent.conditions.splice(parseInt(parts[parts.length - 1]), 1)[0];
            if (removed) rowVals.delete(removed);
          }
        });
        this.$el.addEventListener('input', this.onBuilderInput.bind(this));
        this.$el.addEventListener('change', this.onBuilderInput.bind(this));
        var self2 = this;
        this.$watch('builderHtml', function() {
          this.$nextTick(function() {
            var el = document.getElementById('query-builder');
            if (el) Alpine.initTree(el);
          });
        }.bind(this));
        this.$nextTick(function() {
          var el = document.getElementById('query-builder');
          if (el) Alpine.initTree(el);
        });
        this.loadHistory();
        this.fetchAllTags();
      },

      fetchAllTags: async function() {
        this.allTags = await API.loadAllTagsSafe();
      },

      /* ── Path helpers ── */
      resolveGroup: function(path) {
        if (!path) return this.root;
        var parts = path.split('/');
        var g = this.root;
        for (var i = 0; i < parts.length; i++) {
          var node = g.conditions[parseInt(parts[i])];
          if (!node || !node.conditions) return g;
          g = node;
        }
        return g;
      },

      getRow: function(path) {
        var parts = path.split('/');
        var g = this.root;
        for (var j = 0; j < parts.length - 1; j++) {
          var node = g.conditions[parseInt(parts[j])];
          if (node && node.conditions) g = node;
          else return null;
        }
        return g.conditions[parseInt(parts[parts.length - 1])] || null;
      },

      addRowTo: function(group, path) {
        var row = newRow();
        var idx = group.conditions.push(row) - 1;
        return { row: row, path: path ? path + '/' + idx : String(idx) };
      },

      addTopGroup: function() {
        this.root.conditions.push(newGroup());
      },

      /* ── Builder events (delegation) ── */
      onBuilderInput: function(e) {
        var el = e.target.closest('[data-path][data-field]');
        if (!el) return;
        var path = el.getAttribute('data-path');
        var field = el.getAttribute('data-field');
        var row = this.getRow(path);
        if (!row) return;
        if (field === 'op') {
          row.op = el.value;
          return;
        }
        if (field === 'type') {
          row.type = el.value;
          rowVals.set(row, {});
          return;
        }
        var vals = rowVals.get(row);
        if (!vals) { vals = {}; rowVals.set(row, vals); }
        vals[field] = el.value;
      },

      onRowTagPicked: function(path, name) {
        var row = this.getRow(path);
        if (!row) return;
        var vals = rowVals.get(row);
        if (!vals) { vals = {}; rowVals.set(row, vals); }
        vals.value = name;
        var input = this.$el.querySelector('[data-path="' + path + '"][data-field="value"]');
        if (input && input.value !== name) input.value = name;
      },

      /* ── Render builder (x-html, recursive groups) ──
         注意：下方 tagSearchHtml 的下拉结构与 cmd/webui/templates/partials/tag_search.html
         是同一形状的**两份跨系统源**（Go 静态模板 vs 运行时 JS 字符串，无法合并）——
         改任一处 dropdown/dropdown-item 结构时必须同步另一处。 */
      get builderHtml() {
        var component = this;
        function tagSearchHtml(path, field, widthCls) {
          var pk = path.replace(/\//g, '_');
          return '<div class="qb-tagwrap" x-data="tagSearch({ tags: () => allTags })" @tag-selected="onRowTagPicked(\'' + path + '\', $event.detail.name)">' +
            '<input class="' + (field === 'tag' ? 'qb-tag-input ' : 'qb-val-input ') + widthCls + '" x-model="value" placeholder="标签名..." autocomplete="off" ' +
              'data-path="' + path + '" data-field="' + field + '" :class="{\'tag-selected\': selectedId}" ' +
              '@input="render" @focus="render" @keydown="onKeydown">' +
            '<div class="dropdown" :class="{active: open}">' +
              '<template x-for="(t, i) in filtered" :key="\'st_' + pk + '_' + field + '_\' + t.id">' +
                '<div class="dropdown-item" :class="{highlighted: i === highlight}" @mouseover="highlight = i" @mousedown.prevent="select(t)">' +
                  '<span x-text="t.path || t.name"></span> <small x-text="\' (\' + (t.cat || \'\') + \')\'"></small>' +
                '</div>' +
              '</template>' +
            '</div>' +
          '</div>';
        }
        function rowHtml(row, path) {
          var type = row.type;
          var vals = rowVals.get(row) || {};
          var typeOpts = Object.keys(CONDITION_TYPES).map(function(k) {
            return '<option value="' + k + '"' + (k === type ? ' selected' : '') + '>' + CONDITION_TYPES[k].label + '</option>';
          }).join('');
          var typeDef = CONDITION_TYPES[type] || CONDITION_TYPES.filename;
          var opOpts = typeDef.ops.map(function(o) {
            return '<option value="' + o.v + '"' + (o.v === row.op ? ' selected' : '') + '>' + o.l + '</option>';
          }).join('');
          var valHtml = '';
          switch (type) {
            case 'filename':
              valHtml = '<input class="qb-val-input qb-val-name" type="text" data-path="' + path + '" data-field="value" value="' + escHtml(vals.value || '') + '" placeholder="文件名..." autocomplete="off">';
              break;
            case 'tag':
              valHtml = tagSearchHtml(path, 'value', 'qb-val-name');
              break;
            case 'tag_value':
              valHtml = tagSearchHtml(path, 'tag', 'qb-val-short') +
                '<span class="qb-sep">:</span>' +
                '<input class="qb-val-input qb-val-short" type="text" data-path="' + path + '" data-field="value" value="' + escHtml(vals.value || '') + '" placeholder="值...">';
              break;
            case 'size':
              var unitOpts = SIZE_UNITS.map(function(u) {
                return '<option value="' + u.v + '"' + (String(vals.unit || '1048576') === String(u.v) ? ' selected' : '') + '>' + u.l + '</option>';
              }).join('');
              valHtml = '<input class="qb-val-number" type="number" min="0" data-path="' + path + '" data-field="num" value="' + escHtml(vals.num !== undefined ? vals.num : '1') + '">' +
                '<select class="qb-val-unit" data-path="' + path + '" data-field="unit">' + unitOpts + '</select>';
              break;
            case 'ext':
              valHtml = '<input class="qb-val-input qb-val-ext" type="text" data-path="' + path + '" data-field="value" value="' + escHtml(vals.value || '') + '" placeholder="jpg, png, gif...">';
              break;
            case 'date':
              valHtml = '<input class="qb-val-date" type="date" data-path="' + path + '" data-field="value" value="' + escHtml(vals.value || '') + '">';
              break;
          }
          return '<div class="qb-row">' +
            '<select class="qb-type" data-path="' + path + '" data-field="type">' + typeOpts + '</select>' +
            '<select class="qb-op" data-path="' + path + '" data-field="op">' + opOpts + '</select>' +
            '<div class="qb-val-area">' + valHtml + '</div>' +
            '<button class="btn btn-xs btn-ghost" data-action="del-row" data-path="' + path + '">×</button>' +
            '</div>';
        }
        function groupHtml(group, path) {
          var logic = (group.logic || 'and').toUpperCase();
          var children = (group.conditions || []).map(function(c, i) {
            var p = path ? path + '/' + i : String(i);
            if (c.conditions) {
              return '<div class="qb-group">' + groupHtml(c, p) + '</div>';
            }
            return rowHtml(c, p);
          }).join('');
          return '<div class="qb-header">' +
            '<button class="btn btn-xs btn-ghost qb-logic-toggle" data-action="toggle-logic" data-path="' + path + '">' + logic + '</button>' +
            '<div class="qb-actions">' +
            '<button class="btn btn-xs btn-default" data-action="add-cond" data-path="' + path + '">+ 条件</button>' +
            '<button class="btn btn-xs btn-ghost" data-action="del-group" data-path="' + path + '">×</button>' +
            '</div></div>' + children;
        }
        return (this.root.conditions || []).map(function(c, i) {
          var p = String(i);
          if (c.conditions) {
            return '<div class="qb-group">' + groupHtml(c, p) + '</div>';
          }
          return rowHtml(c, p);
        }).join('');
      },

      /* ── Query building ── */
      condFromRow: function(row) {
        var vals = rowVals.get(row) || {};
        var cond = { type: row.type, op: row.op };
        switch (row.type) {
          case 'filename':
          case 'tag':
            cond.value = (vals.value || '').trim();
            break;
          case 'tag_value':
            cond.tag = (vals.tag || '').trim();
            cond.value = (vals.value || '').trim();
            break;
          case 'size':
            cond.value = (parseFloat(vals.num) || 0) * (parseInt(vals.unit) || 1);
            break;
          case 'ext':
            cond.value = (vals.value || '').split(',').map(function(s) { return s.trim(); }).filter(Boolean);
            break;
          case 'date':
            cond.value = vals.value || '';
            break;
        }
        if (!cond.value || (Array.isArray(cond.value) && cond.value.length === 0)) return null;
        return cond;
      },

      buildGroupQuery: function(group) {
        var self3 = this;
        var conditions = [];
        (group.conditions || []).forEach(function(c) {
          if (c.conditions) {
            var sub = self3.buildGroupQuery(c);
            if (sub.conditions.length > 0) conditions.push(sub);
          } else {
            var cond = self3.condFromRow(c);
            if (cond) conditions.push(cond);
          }
        });
        return { logic: group.logic || 'and', conditions: conditions };
      },

      buildQuery: function() {
        return this.buildGroupQuery(this.root);
      },

      /* ── Search ── */
      doSearch: async function(page) {
        var query = this.buildQuery();
        if (!query || !query.conditions || query.conditions.length === 0) {
          showToast('请至少添加一个搜索条件', 'error');
          return;
        }
        this.page = page || 1;
        this._lastQuery = query;
        await loadPaged(this, SEARCH_SPEC, false);
      },

      /* ── History ── */
      loadHistory: function() {
        var entries = readStoredJSON(localStorage, HISTORY_KEY, []);
        if (!Array.isArray(entries)) entries = [];
        this.history = entries.map(function(entry) {
          var q;
          try { q = JSON.parse(entry); } catch (e) { return null; }
          return { raw: entry, query: q, summary: queryToSummary(q) || '...' };
        }).filter(Boolean);
      },

      saveHistory: function(query) {
        var entry = JSON.stringify(query);
        var history = readStoredJSON(localStorage, HISTORY_KEY, []);
        if (!Array.isArray(history)) history = [];
        history = history.filter(function(h) { return h !== entry; });
        history.unshift(entry);
        if (history.length > MAX_HISTORY) history = history.slice(0, MAX_HISTORY);
        try { localStorage.setItem(HISTORY_KEY, JSON.stringify(history)); } catch (e) {}
        this.loadHistory();
      },

      clearHistory: function() {
        try { localStorage.removeItem(HISTORY_KEY); } catch (e) {}
        this.loadHistory();
      },

      useHistory: function(h) {
        var group = newGroup();
        var self4 = this;
        (function fill(container, query) {
          (query.conditions || []).forEach(function(c) {
            if (c.conditions) {
              var g = { logic: c.logic || 'and', conditions: [] };
              container.conditions.push(g);
              fill(g, c);
            } else {
              var row = { type: c.type, op: c.op };
              container.conditions.push(row);
              var vals = {};
              switch (c.type) {
                case 'filename':
                case 'tag': vals.value = c.value || ''; break;
                case 'tag_value': vals.tag = c.tag || ''; vals.value = c.value || ''; break;
                case 'size':
                  var bytes = typeof c.value === 'number' ? c.value : (parseFloat(c.value) || 0);
                  var bestUnit = 1;
                  for (var u = SIZE_UNITS.length - 1; u >= 0; u--) {
                    if (bytes >= SIZE_UNITS[u].v && bytes % SIZE_UNITS[u].v === 0) { bestUnit = SIZE_UNITS[u].v; break; }
                  }
                  vals.num = String(bytes / bestUnit);
                  vals.unit = String(bestUnit);
                  break;
                case 'ext': vals.value = Array.isArray(c.value) ? c.value.join(', ') : (c.value || ''); break;
                case 'date': vals.value = c.value || ''; break;
              }
              rowVals.set(row, vals);
            }
          });
        })(group, query);
        group.logic = query.logic || 'and';
        this.root = group;
        this.doSearch(1);
      }
    };
  });
});

/* ── Summary text for history ── */
function queryToSummary(query) {
  if (!query || !query.conditions || query.conditions.length === 0) return '';
  function conditionsToSummary(conditions) {
    var parts = [];
    for (var i = 0; i < conditions.length; i++) {
      var c = conditions[i];
      if (c.conditions) {
        parts.push('(' + conditionsToSummary(c.conditions) + ')');
      } else {
        var typeName = CONDITION_TYPES[c.type] ? CONDITION_TYPES[c.type].label : c.type;
        var opName = '';
        var typeOps = CONDITION_TYPES[c.type] ? CONDITION_TYPES[c.type].ops : [];
        for (var j = 0; j < typeOps.length; j++) {
          if (typeOps[j].v === c.op) { opName = typeOps[j].l; break; }
        }
        var val = c.value;
        if (c.type === 'size' && typeof val === 'number') val = formatSize(val);
        if (Array.isArray(val)) val = val.join(',');
        parts.push(typeName + ':' + opName + ':' + val);
      }
    }
    return parts.join(' AND ');
  }
  return conditionsToSummary(query.conditions);
}
