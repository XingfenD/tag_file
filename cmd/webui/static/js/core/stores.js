/* ═══════════════════════════════════════════
   TagFile — Alpine.js Shared Stores
   Toast + Confirm Modal
   ═══════════════════════════════════════════ */

document.addEventListener('alpine:init', function() {
  /* ── Toast ── */
  Alpine.store('toast', {
    items: [],
    show: function(msg, type) {
      type = type || 'info';
      this.items.push({ msg: msg, type: type });
      var self = this;
      setTimeout(function() { self.items.shift(); }, 3000);
    }
  });

  /* ── Confirm Modal ── */
  Alpine.store('confirm', {
    active: false,
    msg: '',
    onOk: null,
    onCancel: null,
    show: function(msg, onOk, onCancel) {
      this.msg = msg;
      this.onOk = onOk || null;
      this.onCancel = onCancel || null;
      this.active = true;
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
window.showConfirm = function(msg, onOk, onCancel) { Alpine.store('confirm').show(msg, onOk, onCancel); };
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
