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
});

/* ── Compatibility wrappers (legacy pages until migrated) ── */
window.showToast = function(msg, type) { Alpine.store('toast').show(msg, type); };
window.showConfirm = function(msg, onOk, onCancel) { Alpine.store('confirm').show(msg, onOk, onCancel); };

/* ── Global Keyboard Shortcuts ── */
document.addEventListener('keydown', function(e) {
  if ((e.ctrlKey || e.metaKey) && e.key === 'k') {
    e.preventDefault();
    var searchInput = document.querySelector('[data-global-search]');
    if (searchInput) searchInput.focus();
  }
  if (e.key === 'Escape' && Alpine.store('confirm').active) {
    Alpine.store('confirm').cancel();
  }
});
