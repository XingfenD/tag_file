/* ═══════════════════════════════════════════
   TagFile — Config Page (Alpine.js, schema-driven)
   ═══════════════════════════════════════════ */

document.addEventListener('alpine:init', function() {
  Alpine.data('configPage', function() {
    return {
      schemaLoaded: false,
      schemaLoadError: '',
      groups: [],

      init: function() {
        this.loadSchema();
      },

      /* ── schema 加载（整页加载守卫）── */
      loadSchema: async function() {
        try {
          var d = await API.get('/api/config/schema');
          if (!d.ok) {
            this.schemaLoaded = false;
            this.schemaLoadError = d.msg || '配置 schema 加载失败';
            return;
          }
          this.applySchema(d.groups || []);
          this.schemaLoaded = true;
          this.schemaLoadError = '';
        } catch (err) {
          API.onError(err);
          this.schemaLoaded = false;
          this.schemaLoadError = '配置 schema 加载失败：' + (err.message || '请求出错');
        }
      },

      applySchema: function(groups) {
        var self = this;
        this.groups = groups.map(function(g) {
          return {
            id: g.id, title: g.title, order: g.order,
            fields: (g.fields || []).map(self.buildField),
            saving: false
          };
        });
      },

      buildField: function(f) {
        var field = {
          key: f.key, label: f.label, description: f.description || '',
          type: f.type, default: f.default == null ? '' : f.default,
          min: f.min, max: f.max,
          min_exclusive: !!f.min_exclusive, max_exclusive: !!f.max_exclusive,
          pattern: f.pattern || '', options: f.options || [],
          item_type: f.item_type || '', unit: f.unit || '',
          show_if: f.show_if || null,
          value: f.value == null ? '' : f.value,
          has_value: !!f.has_value,
          invalid: !!f.invalid,
          invalid_reason: f.invalid_reason || '',
          touched: false, failMsg: ''
        };
        if (f.type === 'list') {
          var rows = [];
          try {
            var arr = JSON.parse(field.value || '[]');
            if (Array.isArray(arr)) rows = arr.map(String);
          } catch (e) { rows = []; }
          field.rows = rows;
        }
        return field;
      },

      /* ── JS 侧校验（服务端兜底）── */
      touch: function(f) { f.failMsg = ''; f.touched = true; },

      fieldVisible: function(f, g) {
        if (!f.show_if) return true;
        for (var i = 0; i < g.fields.length; i++) {
          if (g.fields[i].key === f.show_if.key) return g.fields[i].value === f.show_if.equals;
        }
        return true;
      },

      rangeText: function(f) {
        var left = (f.min_exclusive || f.min == null) ? '(' : '[';
        var right = (f.max_exclusive || f.max == null) ? ')' : ']';
        if (f.min != null && f.max != null) return left + String(f.min) + ', ' + String(f.max) + right;
        if (f.min != null) return left + String(f.min) + ', +∞' + right;
        if (f.max != null) return '(-∞, ' + String(f.max) + right;
        return '任意值';
      },

      boundsError: function(f, n) {
        if (f.min != null) {
          if (f.min_exclusive ? n <= f.min : n < f.min)
            return '取值超出合法范围，应为 ' + this.rangeText(f);
        }
        if (f.max != null) {
          if (f.max_exclusive ? n >= f.max : n > f.max)
            return '取值超出合法范围，应为 ' + this.rangeText(f);
        }
        return '';
      },

      fieldError: function(f) {
        var v = f.value == null ? '' : String(f.value).trim();
        switch (f.type) {
          case 'int':
            if (v === '') return '取值必须是整数';
            if (!/^-?\d+$/.test(v)) return '取值必须是整数';
            return this.boundsError(f, Number(v));
          case 'float':
            if (v === '') return '取值必须是数字';
            var nf = Number(v);
            if (!isFinite(nf)) return '取值必须是数字';
            return this.boundsError(f, nf);
          case 'duration':
            if (v === '') return '取值必须是秒数';
            var nd = Number(v);
            if (!isFinite(nd)) return '取值必须是秒数';
            return this.boundsError(f, nd);
          case 'url':
            if (v === '') return '取值必须是 http 或 https 地址，不能为空';
            if (!/^https?:\/\//i.test(v)) return '取值必须以 http:// 或 https:// 开头';
            if (!/^https?:\/\/[^/\s]+/i.test(v)) return '取值必须包含主机名';
            return '';
          case 'string':
            if (f.pattern && v) {
              try { if (!new RegExp(f.pattern).test(v)) return '取值不符合匹配模式 ' + f.pattern; }
              catch (e) { /* 服务端兜底 */ }
            }
            return '';
          case 'list':
            if (f.item_type === 'regex') {
              var rows = f.rows || [];
              for (var i = 0; i < rows.length; i++) {
                if (!rows[i]) continue;
                try { new RegExp(rows[i]); }
                catch (e) { return '第 ' + (i + 1) + ' 项不是合法的正则表达式'; }
              }
            }
            return '';
          default:
            return '';
        }
      },

      regexRowValid: function(f, row) {
        if (!row) return true;
        try { new RegExp(row); return true; } catch (e) { return false; }
      },

      fieldInvalid: function(f) {
        if (f.failMsg) return true;
        if (f.touched) return this.fieldError(f) !== '';
        return f.invalid === true;
      },

      fieldErrorText: function(f) {
        if (f.failMsg) return f.failMsg;
        if (f.touched) return this.fieldError(f);
        return f.invalid ? f.invalid_reason : '';
      },

      groupInvalidCount: function(g) {
        var c = 0;
        for (var i = 0; i < g.fields.length; i++) if (this.fieldInvalid(g.fields[i])) c++;
        return c;
      },

      fieldHint: function(f) {
        if (f.type === 'list') return f.item_type === 'regex' ? '每行一条，正则表达式' : '每行一条';
        return '';
      },

      /* ── 字段编辑 ── */
      setBool: function(f, e) { f.value = e.target.checked ? 'true' : 'false'; this.touch(f); },
      addRow: function(f) { f.rows.push(''); this.touch(f); },
      removeRow: function(f, i) { f.rows.splice(i, 1); this.touch(f); },

      resetField: function(f) {
        if (f.type === 'list') {
          var rows = [];
          try {
            var arr = JSON.parse(f.default || '[]');
            if (Array.isArray(arr)) rows = arr.map(String);
          } catch (e) { rows = []; }
          f.rows = rows;
        } else {
          f.value = f.default == null ? '' : f.default;
        }
        this.touch(f);
      },

      resetGroup: function(g) {
        var self = this;
        showConfirm('确定把 «' + g.title + '» 的全部配置项恢复为默认值？', function() {
          for (var i = 0; i < g.fields.length; i++) self.resetField(g.fields[i]);
          showToast('已恢复默认值，尚未保存', 'info');
        });
      },

      /* ── 整组保存 ── */
      saveGroup: async function(g) {
        if (g.saving) return;
        var payload = { group: g.id };
        for (var i = 0; i < g.fields.length; i++) {
          var f = g.fields[i];
          if (f.type === 'secret') {
            // 空值（未修改）不提交该键，避免用空串覆盖已保存的密钥
            if (f.value != null && f.value !== '') payload[f.key] = f.value;
          } else if (f.type === 'list') {
            var rows = (f.rows || [])
              .map(function(r) { return r == null ? '' : String(r).trim(); })
              .filter(function(r) { return r !== ''; });
            payload[f.key] = JSON.stringify(rows);
          } else {
            payload[f.key] = f.value == null ? '' : f.value;
          }
        }
        g.saving = true;
        try {
          var d = await API.post('/api/config/group', payload);
          if (d.ok) {
            showToast(d.msg || '已保存', 'success');
            await this.refreshGroup(g);
          } else {
            for (var j = 0; j < g.fields.length; j++) g.fields[j].failMsg = '';
            var failures = d.failures || [];
            for (var k = 0; k < failures.length; k++) {
              for (var m = 0; m < g.fields.length; m++) {
                if (g.fields[m].key === failures[k].key) {
                  g.fields[m].failMsg = failures[k].msg;
                  break;
                }
              }
            }
            showToast(failures.length ? failures[0].msg : (d.msg || '保存失败'), 'error');
          }
        } catch (err) {
          API.onError(err);
        } finally {
          g.saving = false;
        }
      },

      /* 保存成功后只刷新已保存的那一组：把服务端快照（value/has_value/invalid/
         invalid_reason）合并回该组，其它组的本地未保存编辑保持不变。 */
      refreshGroup: async function(g) {
        try {
          var d = await API.get('/api/config/schema');
          if (!d.ok) return;
          var groups = d.groups || [];
          for (var i = 0; i < groups.length; i++) {
            if (groups[i].id === g.id) {
              g.fields = (groups[i].fields || []).map(this.buildField);
              break;
            }
          }
        } catch (err) { /* 保留当前状态 */ }
      }
    };
  });
});
