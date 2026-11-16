/* ═══════════════════════════════════════════
   TagFile — Config Page (Alpine.js)
   Groups and fields are rendered from /api/config/schema;
   unknown keys live in the advanced card.
   ═══════════════════════════════════════════ */

document.addEventListener('alpine:init', function() {
  var COLLAPSE_KEY = 'tagfile_config_collapsed';
  var NUMERIC = ['int', 'float', 'duration'];

  function parseRows(s) {
    try {
      var a = JSON.parse(s);
      if (Array.isArray(a)) return a.map(function(x) { return String(x); });
    } catch (e) { /* invalid stored JSON: rows stay empty, the server reason is shown */ }
    return [];
  }

  function normalizeField(f) {
    f.dirty = false;
    f.serverErr = '';
    f.rows = f.type === 'list' ? parseRows(f.value) : [];
    if (f.type === 'secret') f.value = '';
    if (f.value == null) f.value = '';
    return f;
  }

  Alpine.data('configPage', function() {
    return {
      /* ── schema / guard ──
         schemaLoaded is the permission to edit: until the schema and the stored values have
         been read, the form would show defaults that may differ from the table, and a save
         would overwrite the real values with them. */
      groups: [],
      schemaLoaded: false,
      schemaError: '',
      collapsed: {},

      /* ── advanced (unknown keys) ── */
      q: '',
      queryInput: '',
      page: 1,
      items: [],
      pagination: {},
      loading: true,

      init: function() {
        this.collapsed = this.readCollapsed();
        this.loadSchema();
        this.loadAdvanced();
      },

      readCollapsed: function() {
        try {
          var v = JSON.parse(localStorage.getItem(COLLAPSE_KEY) || '{}');
          return v && typeof v === 'object' ? v : {};
        } catch (e) {
          return {};
        }
      },

      isCollapsed: function(id) {
        return !!this.collapsed[id];
      },

      toggleGroup: function(id) {
        var next = Object.assign({}, this.collapsed);
        if (next[id]) delete next[id];
        else next[id] = true;
        this.collapsed = next;
        try {
          localStorage.setItem(COLLAPSE_KEY, JSON.stringify(next));
        } catch (e) { /* storage unavailable: the toggle still works for this page view */ }
      },

      loadSchema: async function() {
        this.schemaLoaded = false;
        this.schemaError = '';
        try {
          var d = await API.get('/api/config/schema');
          if (!d.ok) {
            this.schemaError = d.msg || '配置结构加载失败';
            return;
          }
          this.groups = (d.groups || []).map(function(g) {
            g.fields = (g.fields || []).map(normalizeField);
            g.saving = false;
            return g;
          });
          this.schemaLoaded = true;
        } catch (err) {
          API.onError(err);
          this.schemaError = '配置读取失败：' + (err.message || '请求出错');
        }
      },

      /* ── per-field validation (mirrors internal/service/config/schema, the server still decides) ── */
      rangeText: function(f) {
        if (NUMERIC.indexOf(f.type) < 0) return '';
        var lo = f.min != null ? (f.min_exclusive ? '> ' : '≥ ') + f.min : '';
        var hi = f.max != null ? (f.max_exclusive ? '< ' : '≤ ') + f.max : '';
        return [lo, hi].filter(Boolean).join(' 且 ');
      },

      hintText: function(f) {
        var parts = [];
        if (f.type !== 'secret' && f.default !== '') parts.push('默认：' + f.default);
        if (f.unit) parts.push('单位：' + f.unit);
        var r = this.rangeText(f);
        if (r) parts.push('范围：' + r);
        return parts.join('　');
      },

      numberError: function(f) {
        var raw = String(f.value == null ? '' : f.value).trim();
        if (raw === '') return '必须填写数值';
        var pattern = f.type === 'int' ? /^[+-]?\d+$/ : /^[+-]?(\d+\.?\d*|\.\d+)([eE][+-]?\d+)?$/;
        if (!pattern.test(raw)) return f.type === 'int' ? '取值必须是整数' : '取值必须是数字';
        var n = Number(raw);
        if (!isFinite(n)) return '取值必须是有限的数字';
        if (f.min != null && (f.min_exclusive ? n <= f.min : n < f.min)) return '取值超出合法范围：' + this.rangeText(f);
        if (f.max != null && (f.max_exclusive ? n >= f.max : n > f.max)) return '取值超出合法范围：' + this.rangeText(f);
        return '';
      },

      urlError: function(f) {
        var v = String(f.value == null ? '' : f.value).trim();
        if (v === '') return '取值不能为空';
        if (!/^https?:\/\/[^\/\s]+/i.test(v)) return '取值必须以 http:// 或 https:// 开头，并包含主机名';
        return '';
      },

      rowError: function(f, idx) {
        if (f.type !== 'list' || f.item_type !== 'regex') return '';
        var row = f.rows[idx];
        if (row == null || String(row).trim() === '') return '';
        try {
          new RegExp(row);
          return '';
        } catch (e) {
          return '不是合法的正则表达式：' + e.message;
        }
      },

      fieldError: function(f) {
        if (f.serverErr) return f.serverErr;
        var msg = '';
        if (NUMERIC.indexOf(f.type) >= 0) msg = this.numberError(f);
        else if (f.type === 'url') msg = this.urlError(f);
        else if (f.type === 'list') {
          for (var i = 0; i < f.rows.length && !msg; i++) {
            var rowMsg = this.rowError(f, i);
            if (rowMsg) msg = '第 ' + (i + 1) + ' 行' + rowMsg;
          }
        }
        if (msg) return msg;
        // A stored value the server flagged stays flagged until the user edits it.
        if (!f.dirty && f.invalid) return f.invalid_reason || '存量值不合法';
        return '';
      },

      invalidCount: function(g) {
        var self = this;
        return g.fields.filter(function(f) { return !!self.fieldError(f); }).length;
      },

      touch: function(f) {
        f.dirty = true;
        f.serverErr = '';
      },

      addRow: function(f) {
        f.rows.push('');
        this.touch(f);
      },

      removeRow: function(f, idx) {
        f.rows.splice(idx, 1);
        this.touch(f);
      },

      restoreDefault: function(f) {
        if (f.type === 'list') f.rows = parseRows(f.default);
        else f.value = f.default;
        this.touch(f);
      },

      /* ── group save ── */
      saveGroup: async function(g) {
        if (!this.schemaLoaded || g.saving) return;
        if (this.invalidCount(g) > 0) {
          showToast('本组有非法值，请先修正标红的字段', 'error');
          return;
        }

        var body = { group: g.id };
        g.fields.forEach(function(f) {
          if (f.type === 'secret' && f.value === '' && f.has_value) return;
          body[f.key] = f.type === 'list'
            ? JSON.stringify(f.rows.filter(function(r) { return String(r).trim() !== ''; }))
            : f.value;
        });

        g.saving = true;
        try {
          var d = await API.post('/api/config/group', body);
          if (!d.ok) {
            var failures = d.failures || [];
            failures.forEach(function(x) {
              var hit = g.fields.filter(function(f) { return f.key === x.key; })[0];
              if (hit) hit.serverErr = x.msg;
            });
            showToast(failures.length ? failures[0].msg : (d.msg || '保存失败'), 'error');
            return;
          }
          g.fields.forEach(function(f) {
            if (!(f.key in body)) return;
            if (f.type === 'secret') {
              f.has_value = body[f.key] !== '';
              f.value = '';
            } else if (f.type === 'list') {
              f.value = body[f.key];
            }
            f.dirty = false;
            f.invalid = false;
            f.serverErr = '';
          });
          showToast(d.msg || '分组已保存', 'success');
        } catch (err) {
          API.onError(err);
        } finally {
          g.saving = false;
        }
      },

      /* ── advanced: unknown keys ── */
      doSearch: function() {
        this.q = (this.queryInput || '').trim();
        this.page = 1;
        this.loadAdvanced();
      },

      clearSearch: function() {
        this.q = '';
        this.queryInput = '';
        this.page = 1;
        this.loadAdvanced();
      },

      goToPage: function(page) {
        this.page = page;
        this.loadAdvanced();
      },

      displayIndex: function(i) {
        return (this.pagination.page - 1) * (this.pagination.page_size || 10) + i + 1;
      },

      loadAdvanced: async function() {
        this.loading = true;
        try {
          var url = '/api/config?unknown=1&page=' + this.page + '&page_size=10';
          if (this.q) url += '&q=' + encodeURIComponent(this.q);
          var d = await API.get(url);
          if (!d.ok) { showToast(d.msg || '加载失败', 'error'); return; }
          this.items = d.items || [];
          this.pagination = d.pagination || {};
        } catch (err) {
          API.onError(err);
        } finally {
          this.loading = false;
        }
      },

      openAddUnknown: function() {
        var self = this;
        Modal.show({
          title: '新增自定义配置',
          fields: [
            { label: '配置键', name: 'key', value: '', required: true },
            { label: '配置值', name: 'value', value: '', required: true }
          ],
          onSave: async function(values) {
            var key = (values.key || '').trim();
            if (!key) { showToast('配置键不能为空', 'error'); return false; }
            var d = await API.post('/api/config', { key: key, value: values.value });
            if (!d.ok) { showToast(d.msg || '保存失败', 'error'); return false; }
            showToast(d.msg || '配置已保存', 'success');
            self.loadAdvanced();
          }
        });
      },

      editConfig: function(key, val) {
        var self = this;
        Modal.show({
          title: '编辑配置: ' + key,
          fields: [
            { label: '配置键', name: 'key', value: key, readonly: true },
            { label: '配置值', name: 'value', value: val, required: true }
          ],
          onSave: async function(values) {
            var newVal = values.value.trim();
            if (!newVal) { showToast('配置值不能为空', 'error'); return false; }
            var d = await API.put('/api/config/' + encodeURIComponent(values.key), { value: newVal });
            if (!d.ok) { showToast(d.msg || '更新失败', 'error'); return false; }
            showToast(d.msg || '配置已更新', 'success');
            self.loadAdvanced();
          }
        });
      },

      removeConfig: function(key) {
        var self = this;
        showConfirm('确定删除配置 «' + key + '»？', function() {
          API.del('/api/config/' + encodeURIComponent(key)).then(function(d) {
            if (d.ok) {
              showToast(d.msg || '配置已删除', 'success');
              self.loadAdvanced();
            } else {
              showToast(d.msg || '删除失败', 'error');
            }
          }).catch(API.onError);
        });
      }
    };
  });
});
