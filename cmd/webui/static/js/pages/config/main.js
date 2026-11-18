/* ═══════════════════════════════════════════
   TagFile — Config Page (Alpine.js, 通用指令执行器)
   ═══════════════════════════════════════════

   页面只是 /api/config/schema 下发指令（rules）的通用执行器：
   - 控件取指令的 widget（无则按结构默认），「哪种约束用哪种控件」
     的后端语义映射只住在后端 ruleDefs，JS 零副本；
   - 提示/占位/校验文案一律用下发的 hint/message，无中文文案副本；
   - 本地 ruleCheckers 只认识指令原语 kind，逐条镜像后端 validate
     语义；未知 kind 优雅降级（本地放行，信任服务端 invalid_reason
     与整组保存的 failures）；
   - render/behavior 指令只驱动 UX（textarea/password/留空不提交），
     校验语义永远只在服务端独立成立。 */

/* 本地可执行的 validate 指令注册表（spec §4.4）。值级指令收到存储
   形态串（Int 已换算回整数，边界在显示单位判定）；容器级指令
   （item_count / fixed_arity）收到行数组。失败文案不在此处——显示
   下发的 rule.message。 */
var ruleCheckers = {
  required:     function (v, p) { return v !== ''; },
  int:          function (v, p) { return /^-?\d+$/.test(v); },
  int_range:    function (v, p) {
    if (!/^-?\d+$/.test(v)) return false;
    var shift = p.shift === undefined ? 0 : Number(p.shift);
    if (!isFinite(shift) || shift < 0) shift = 0;
    var display = Number(v) / Math.pow(10, shift);
    if (p.min !== undefined) {
      var min = Number(p.min);
      if (p.min_exclusive === 'true' ? display <= min : display < min) return false;
    }
    if (p.max !== undefined) {
      var max = Number(p.max);
      if (p.max_exclusive === 'true' ? display >= max : display > max) return false;
    }
    return true;
  },
  url:          function (v, p) {
    if (v === '') return false;
    if (!/^https?:\/\//i.test(v)) return false;
    return /^https?:\/\/[^/\s]+/i.test(v);
  },
  regex:        function (v, p) {
    try { new RegExp(v); return true; } catch (e) { return false; }
  },
  pattern:      function (v, p) {
    if (v === '') return true;
    try { return new RegExp(p.pattern).test(v); } catch (e) { return true; }
  },
  oneof:        function (v, p) {
    var opts = [];
    try { opts = JSON.parse(p.options || '[]'); } catch (e) { return true; }
    for (var i = 0; i < opts.length; i++) if (opts[i] && opts[i].value === v) return true;
    return false;
  },
  bool:         function (v, p) { return v === 'true' || v === 'false'; },
  item_count:   function (rows, p) {
    if (p.min_items !== undefined && rows.length < Number(p.min_items)) return false;
    if (p.max_items !== undefined && rows.length > Number(p.max_items)) return false;
    return true;
  },
  fixed_arity:  function (rows, p) { return rows.length === Number(p.n); }
};

/* 指令组读取侧单源：findRule 回首个匹配规则或 null，六个扫描器共用一次遍历。 */
function findRule(rules, match) {
  var test = typeof match === 'function' ? match : function(r) { return r.kind === match; };
  for (var i = 0; i < rules.length; i++) { if (test(rules[i])) return rules[i]; }
  return null;
}

/* DynamicArray 的保存形态：trim+去空行。校验（fieldError）与提交
   （containerPayload）必须共用此单源——漂移即本地放行、服务端拒收。 */
function keptRows(f) {
  return (f.rows || [])
    .map(function(r) { return r == null ? '' : String(r).trim(); })
    .filter(function(r) { return r !== ''; });
}

document.addEventListener('alpine:init', function() {
  Alpine.data('configPage', function() {
    return {
      schemaLoaded: false,
      schemaLoadError: '',
      blocks: [],

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
          this.applySchema(d.blocks || []);
          this.schemaLoaded = true;
          this.schemaLoadError = '';
        } catch (err) {
          API.onError(err);
          this.schemaLoaded = false;
          this.schemaLoadError = '配置 schema 加载失败：' + (err.message || '请求出错');
        }
      },

      applySchema: function(blocks) {
        var self = this;
        this.blocks = blocks.map(function(b) {
          return {
            id: b.id, title: b.title, order: b.order,
            input_boxes: (b.input_boxes || []).map(function(x) { return self.buildField(x); }),
            saving: false
          };
        });
      },

      /* ── input box → 字段模型 ──
         widget：Single 取 rules 里第一个非空 widget（后端已按
         secret>oneof>multiline 优先级排序），无则按结构默认——这些是
         HTML 原语级映射，属前端结构知识。shift/options 直接从指令
         Params 读，不另存一份。Int 显示值 = 存储整数 / 10^shift。 */
      buildField: function(f) {
        var field = {
          key: f.key, label: f.label, description: f.description || '',
          unit: f.unit || '', default: f.default == null ? '' : f.default,
          show_if: f.show_if || null,
          value_type: f.value_type, container: f.container,
          rules: f.rules || [],
          positional_rules: f.positional_rules || [],
          container_rules: f.container_rules || [],
          value: f.value == null ? '' : f.value,
          has_value: !!f.has_value,
          invalid: !!f.invalid,
          invalid_reason: f.invalid_reason || '',
          touched: false, failMsg: ''
        };
        field.shift = this.shiftOfRules(field.rules);
        field.options = this.optionsOfRules(field.rules);
        field.widget = this.widgetOf(field);
        if (field.container !== 'Single') {
          field.rows = this.parseRows(field.value, field);
        } else if (field.value_type === 'Int') {
          field.value = this.storedToDisplay(field.value, field.shift);
        }
        return field;
      },

      widgetOf: function(f) {
        if (f.container === 'Single') {
          var w = this.widgetFromRules(f.rules);
          if (w) return w;
        }
        if (f.container === 'Array') return 'rows-fixed';
        if (f.container === 'DynamicArray') return 'rows';
        if (f.value_type === 'Int') return 'number';
        if (f.value_type === 'Bool') return 'checkbox';
        return 'input';
      },

      widgetFromRules: function(rules) {
        var r = findRule(rules, function(x) { return x.widget; });
        return r ? r.widget : '';
      },

      /* 行控件：去掉容器后按该位指令组推导（渲染/行为指令按后端约束
         只出现在 Single，此处结构默认兜底），Int 行→number、Bool 行→
         checkbox、String 行→input。 */
      rowWidget: function(f, i) {
        var w = this.widgetFromRules(this.rowRules(f, i));
        if (w) return w;
        if (f.value_type === 'Int') return 'number';
        if (f.value_type === 'Bool') return 'checkbox';
        return 'input';
      },

      /* 第 i 行适用的指令组：Array 逐位，DynamicArray 统一 */
      rowRules: function(f, i) {
        if (f.container === 'Array') return f.positional_rules[i] || [];
        return f.rules;
      },

      shiftOfRules: function(rules) {
        var r = findRule(rules, 'int_range');
        if (!r) return 0;
        var s = Number(r.params && r.params.shift);
        return (isFinite(s) && s > 0) ? s : 0;
      },

      /* 不用 findRule（首命中即停）：第一条 oneof 的 options 损坏时要继续
         尝试后面的 oneof，逐条解析直到拿到合法数组。 */
      optionsOfRules: function(rules) {
        for (var i = 0; i < rules.length; i++) {
          var r = rules[i];
          if (r.kind === 'oneof' && r.params && r.params.options) {
            try {
              var opts = JSON.parse(r.params.options);
              if (Array.isArray(opts)) return opts;
            } catch (e) { /* 损坏的 options 参数交给服务端兜底 */ }
          }
        }
        return [];
      },

      isSecret: function(f) {
        return !!findRule(f.rules, 'secret');
      },

      /* secret 占位文案：该 behavior 指令下发的 hint（has_value 时显示） */
      secretHint: function(f) {
        var r = findRule(f.rules, 'secret');
        return r ? (r.hint || '') : '';
      },

      ruleMessage: function(rules, kind) {
        var r = findRule(rules, kind);
        return r ? (r.message || '') : '';
      },

      /* ── 定点数换算（spec §6）── */

      /* 存储整数 → 显示值：非整数存量（legacy/损坏 wire）原样展示，
         交服务端 invalid/invalid_reason 标记，前端不静默改写。 */
      storedToDisplay: function(stored, shift) {
        var s = stored == null ? '' : String(stored);
        if (s === '' || !/^-?\d+$/.test(s)) return s;
        if (!shift) return s;
        return (Number(s) / Math.pow(10, shift)).toFixed(shift);
      },

      /* 显示值 → 存储整数串：先拒 NaN/Inf（返回 null），再
         Math.round 取整——后端永远收整数串。 */
      intToWire: function(display, shift) {
        var s = display == null ? '' : String(display).trim();
        if (s === '') return '';
        var n = Number(s);
        if (!isFinite(n)) return null;
        return String(Math.round(n * Math.pow(10, shift)));
      },

      /* Single 字段的存储形态串（保存与 show_if 比较共用） */
      wireValue: function(f) {
        if (f.container !== 'Single') return f.value == null ? '' : f.value;
        if (f.value_type === 'Int') return this.intToWire(f.value, f.shift);
        return f.value == null ? '' : f.value;
      },

      /* 容器值（JSON 字符串数组）→ 行编辑器行；Int 行换算为显示值 */
      parseRows: function(value, f) {
        var rows = [];
        try {
          var arr = JSON.parse(value || '[]');
          if (Array.isArray(arr)) rows = arr.map(String);
        } catch (e) { rows = []; }
        if (f.value_type === 'Int') {
          for (var i = 0; i < rows.length; i++) {
            rows[i] = this.storedToDisplay(rows[i], this.shiftOfRules(this.rowRules(f, i)));
          }
        }
        return rows;
      },

      /* ── JS 侧校验（服务端兜底）：跑下发的 validate 指令 ── */
      touch: function(f) { f.failMsg = ''; f.touched = true; },

      fieldVisible: function(f, g) {
        if (!f.show_if) return true;
        for (var i = 0; i < g.input_boxes.length; i++) {
          if (g.input_boxes[i].key === f.show_if.key) {
            return this.wireValue(g.input_boxes[i]) === f.show_if.equals;
          }
        }
        return true;
      },

      /* 值级/项级校验：显示值先换算成服务端收到的存储形态（Int 按该
         指令组的 shift 取整；NaN/Inf 借 int 指令下发的文案提示），
         逐条跑已知 checker，失败返回该指令下发的 message。未知 kind
         跳过本地预检，信任服务端。 */
      rulesError: function(f, rules, display) {
        var v = display == null ? '' : String(display).trim();
        if (f.value_type === 'Int') {
          var wire = this.intToWire(v, this.shiftOfRules(rules));
          if (wire === null) return this.ruleMessage(rules, 'int');
          v = wire;
        }
        for (var i = 0; i < rules.length; i++) {
          var r = rules[i];
          if (r.role !== 'validate') continue;
          var checker = ruleCheckers[r.kind];
          if (!checker) continue;
          if (!checker(v, r.params || {})) return r.message || '';
        }
        return '';
      },

      /* 容器级校验（item_count / fixed_arity）：作用于保存形态的行集 */
      containerError: function(f, rows) {
        for (var i = 0; i < f.container_rules.length; i++) {
          var r = f.container_rules[i];
          if (r.role !== 'validate') continue;
          var checker = ruleCheckers[r.kind];
          if (!checker) continue;
          if (!checker(rows, r.params || {})) return r.message || '';
        }
        return '';
      },

      /* 镜像服务端管线：DynamicArray 过滤空行（保存形态）后跑容器级 +
         逐项（统一指令组）；Array 恰好 N 行跑容器级 + 逐位指令组。
         项级失败拼「第 N 项」（1-based，与服务端逐字一致）。 */
      fieldError: function(f) {
        if (f.container === 'Single') return this.rulesError(f, f.rules, f.value);
        var rows = f.rows || [];
        var kept = f.container === 'DynamicArray' ? keptRows(f) : rows;
        var err = this.containerError(f, kept);
        if (err) return err;
        for (var i = 0; i < kept.length; i++) {
          if (f.container === 'Array' && i >= f.positional_rules.length) break;
          err = this.rulesError(f, this.rowRules(f, i), kept[i]);
          if (err) return '第 ' + (i + 1) + ' 项' + err;
        }
        return '';
      },

      /* 行级标红（通用项校验）：DynamicArray 空行跳过（保存时被过滤） */
      rowInvalid: function(f, i) {
        var rows = f.rows || [];
        if (i >= rows.length) return false;
        if (f.container === 'DynamicArray' &&
            String(rows[i] == null ? '' : rows[i]).trim() === '') return false;
        return this.rulesError(f, this.rowRules(f, i), rows[i]) !== '';
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
        for (var i = 0; i < g.input_boxes.length; i++) {
          if (this.fieldInvalid(g.input_boxes[i])) c++;
        }
        return c;
      },

      /* 提示行：validate/render 指令下发 hint 的拼装。behavior 指令
         （如 secret）的 hint 由其专属 UX 消费（password 占位文案），
         不在提示行重复出现。 */
      fieldHint: function(f) {
        var hints = [];
        for (var i = 0; i < f.rules.length; i++) {
          if (f.rules[i].hint && f.rules[i].role !== 'behavior') hints.push(f.rules[i].hint);
        }
        return hints.join('；');
      },

      /* ── 字段编辑 ── */
      setBool: function(f, e) { f.value = e.target.checked ? 'true' : 'false'; this.touch(f); },
      setBoolRow: function(f, i, e) {
        f.rows[i] = e.target.checked ? 'true' : 'false';
        this.touch(f);
      },
      addRow: function(f) { f.rows.push(''); this.touch(f); },
      removeRow: function(f, i) { f.rows.splice(i, 1); this.touch(f); },

      resetField: function(f) {
        if (f.container !== 'Single') {
          f.rows = this.parseRows(f.default, f);
        } else if (f.value_type === 'Int') {
          f.value = this.storedToDisplay(f.default, f.shift);
        } else {
          f.value = f.default == null ? '' : f.default;
        }
        this.touch(f);
      },

      resetGroup: function(g) {
        var self = this;
        showConfirm('确定把 «' + g.title + '» 的全部配置项恢复为默认值？', function() {
          for (var i = 0; i < g.input_boxes.length; i++) self.resetField(g.input_boxes[i]);
          showToast('已恢复默认值，尚未保存', 'info');
        });
      },

      /* ── 整组保存 ── */
      saveGroup: async function(g) {
        if (g.saving) return;
        var payload = { group: g.id };
        for (var i = 0; i < g.input_boxes.length; i++) {
          var f = g.input_boxes[i];
          if (f.container !== 'Single') {
            payload[f.key] = this.containerPayload(f);
          } else if (this.isSecret(f)) {
            // secret（behavior 指令）：空值不提交该键，服务端视为「保持不变」
            if (f.value != null && f.value !== '') payload[f.key] = f.value;
          } else {
            var wire = this.wireValue(f);
            if (wire !== null) payload[f.key] = wire;
          }
        }
        g.saving = true;
        try {
          var d = await API.post('/api/config/group', payload);
          if (d.ok) {
            showToast(d.msg || '已保存', 'success');
            await this.refreshGroup(g);
          } else {
            for (var j = 0; j < g.input_boxes.length; j++) g.input_boxes[j].failMsg = '';
            var failures = d.failures || [];
            for (var k = 0; k < failures.length; k++) {
              for (var m = 0; m < g.input_boxes.length; m++) {
                if (g.input_boxes[m].key === failures[k].key) {
                  g.input_boxes[m].failMsg = failures[k].msg;
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

      /* 容器字段的存储形态：DynamicArray 行 trim + 过滤空行 → JSON
         字符串数组；Array 恰好 N 行原样（不过滤，空串合法）；Int 行
         先按该位 shift 换算回整数串。 */
      containerPayload: function(f) {
        var self = this;
        var rows = f.rows || [];
        var toWire = function(display, rules) {
          if (f.value_type !== 'Int') return display == null ? '' : String(display);
          var w = self.intToWire(display, self.shiftOfRules(rules));
          // NaN/Inf：本地保存门已拦（int 指令文案）；兜底发原串，服务端拒存并回填原因
          return w === null ? String(display == null ? '' : display) : w;
        };
        if (f.container === 'DynamicArray') {
          var kept = keptRows(f).map(function(r) { return toWire(r, f.rules); });
          return JSON.stringify(kept);
        }
        var out = [];
        for (var i = 0; i < rows.length; i++) {
          out.push(toWire(rows[i], f.positional_rules[i] || []));
        }
        return JSON.stringify(out);
      },

      /* 保存成功后只刷新已保存的那一组：把服务端快照（value/has_value/
         invalid/invalid_reason）合并回该组，其它组的本地未保存编辑保持
         不变。 */
      refreshGroup: async function(g) {
        var self = this;
        try {
          var d = await API.get('/api/config/schema');
          if (!d.ok) return;
          var blocks = d.blocks || [];
          for (var i = 0; i < blocks.length; i++) {
            if (blocks[i].id === g.id) {
              g.input_boxes = (blocks[i].input_boxes || [])
                .map(function(x) { return self.buildField(x); });
              break;
            }
          }
        } catch (err) { /* 保留当前状态 */ }
      }
    };
  });
});
