/* ═══════════════════════════════════════════
   TagFile — Alpine.js Shared Stores
   Toast + Confirm Modal + Form Modal
   加载契约：必须在 alpine 脚本（defer）之后、任何用到 Alpine.store 的代码
   执行前就位——本文件在 base.html 主体末尾同步加载，alpine:init 回调一定
   晚于它；window.showToast/showConfirm/Modal 是页面脚本的兼容出口，
   api.js 的 onError 也是运行时才引用 showToast。
   ═══════════════════════════════════════════ */

document.addEventListener('alpine:init', function() {
  /* ── Toast ──
     文案里的 «实体名» 是后端/前端共用的轻量标记：segments() 把它解析成
     .entity-chip 芯片（与确认弹窗同一设计语言），用户看到的不再是书名号。 */
  Alpine.store('toast', {
    items: [],
    show: function(msg, type) {
      type = type || 'info';
      this.items.push({ msg: msg, type: type });
      var self = this;
      setTimeout(function() { self.items.shift(); }, 3000);
    },
    segments: function(msg) {
      msg = msg == null ? '' : String(msg);
      var out = [], re = /«([^»]+)»/g, last = 0, m;
      while ((m = re.exec(msg)) !== null) {
        if (m.index > last) out.push({ text: msg.slice(last, m.index), em: false });
        out.push({ text: m[1], em: true });
        last = re.lastIndex;
      }
      if (last < msg.length) out.push({ text: msg.slice(last), em: false });
      return out.length ? out : [{ text: msg, em: false }];
    }
  });

  /* ── Confirm Modal ──
     name 是被操作实体的名字：文案里不再用书名号包裹，模态按 segments()
     把它渲染成 .entity-chip 芯片（与 .tag 同一套 accent 配色）。 */
  Alpine.store('confirm', {
    active: false,
    msg: '',
    name: '',
    onOk: null,
    onCancel: null,
    show: function(msg, onOk, onCancel, name) {
      this.msg = msg;
      this.name = name || '';
      this.onOk = onOk || null;
      this.onCancel = onCancel || null;
      this.active = true;
    },
    segments: function() {
      var i = this.name ? this.msg.indexOf(this.name) : -1;
      if (i < 0) return [{ text: this.msg, em: false }];
      var out = [];
      if (i > 0) out.push({ text: this.msg.slice(0, i), em: false });
      out.push({ text: this.name, em: true });
      if (i + this.name.length < this.msg.length) out.push({ text: this.msg.slice(i + this.name.length), em: false });
      return out;
    },
    ok: function() {
      this.active = false;
      var f = this.onOk;
      this.onOk = null;
      if (f) f();
    },
    cancel: function() {
      this.active = false;
      var f = this.onCancel;
      this.onCancel = null;
      if (f) f();
    }
  });

  /* ── Form Modal (replaces legacy Modal builder) ── */
  Alpine.store('modal', {
    active: false,
    title: '',
    saveLabel: '保存',
    saveClass: 'btn btn-primary',
    sizeLg: false,
    fields: [],
    loading: false,
    onSave: null,
    show: function(opts) {
      this.title = opts.title || '';
      this.fields = (opts.fields || []).map(function(f) {
        var nf = {};
        for (var k in f) nf[k] = f[k];
        if (nf.value == null) nf.value = '';
        nf.readonly = !!nf.readonly;
        nf.required = !!nf.required;
        return nf;
      });
      this.saveLabel = opts.saveLabel || '保存';
      this.saveClass = opts.saveClass || 'btn btn-primary';
      this.sizeLg = opts.size === 'lg';
      this.onSave = opts.onSave || null;
      this.loading = false;
      this.active = true;
      Alpine.nextTick(function() {
        var first = document.querySelector('#form-modal input:not([readonly]), #form-modal select:not([readonly]), #form-modal textarea');
        if (first) first.focus();
      });
    },
    close: function() {
      this.active = false;
      this.fields = [];
      this.onSave = null;
    },
    values: function() {
      var out = {};
      this.fields.forEach(function(f) { out[f.name] = f.value; });
      return out;
    },
    save: async function() {
      if (!this.onSave) { this.close(); return; }
      for (var i = 0; i < this.fields.length; i++) {
        var f = this.fields[i];
        // 弹窗页脚按钮走 save() 而非 form submit，原生 required 约束够不着——
        // required 的 trim 非空检查在这里统一做（与当年五份手工守卫逐字等价，
        // 文案默认 label+不能为空，requiredMsg 可覆盖）。
        if (f.type !== 'checkbox' && f.required && String(f.value == null ? '' : f.value).trim() === '') {
          showToast(f.requiredMsg || ((f.label || '字段') + '不能为空'), 'error');
          return;
        }
      }
      this.loading = true;
      try {
        var result = await this.onSave(this.values());
        if (result !== false) this.close();
      } catch (e) {
        console.error('Modal save error:', e);
      } finally {
        this.loading = false;
      }
    }
  });
});

/* ── Compatibility wrappers (legacy pages until migrated) ── */
window.showToast = function(msg, type) { Alpine.store('toast').show(msg, type); };
window.showConfirm = function(msg, onOk, onCancel, name) { Alpine.store('confirm').show(msg, onOk, onCancel, name); };
window.Modal = {
  show: function(opts) { Alpine.store('modal').show(opts); },
  close: function() { Alpine.store('modal').close(); }
};

/* ── Global Keyboard Shortcuts ── */
document.addEventListener('keydown', function(e) {
  if ((e.ctrlKey || e.metaKey) && e.key === 'k') {
    e.preventDefault();
    var searchInput = document.querySelector('[data-global-search]');
    if (searchInput) searchInput.focus();
  }
  var modal = Alpine.store('modal');
  if (modal.active) {
    if (e.key === 'Escape') { modal.close(); return; }
    if (e.key === 'Enter') {
      var overlay = document.getElementById('form-modal');
      if (overlay && overlay.querySelector('.dropdown.active')) return;
      var t = e.target;
      if (t && (t.classList.contains('tag-selected') || t.tagName === 'TEXTAREA')) return;
      e.preventDefault();
      modal.save();
      return;
    }
  }
  if (e.key === 'Escape') {
    var confirm = Alpine.store('confirm');
    if (confirm.active) confirm.cancel();
    var activeDropdowns = document.querySelectorAll('.dropdown.active');
    activeDropdowns.forEach(function(d) { d.classList.remove('active'); });
  }
});
