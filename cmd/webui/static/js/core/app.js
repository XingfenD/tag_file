/* ═══════════════════════════════════════════
   TagFile — Shared JavaScript Utilities
   Toast, Modal, Tag Search, Helpers
   ═══════════════════════════════════════════ */

/* ── Toast ── */
function showToast(msg, type) {
  type = type || 'info';
  var container = document.getElementById('toast-container');
  if (!container) {
    container = document.createElement('div');
    container.id = 'toast-container';
    container.className = 'toast-container';
    document.body.appendChild(container);
  }
  var el = document.createElement('div');
  el.className = 'toast toast-' + type;
  el.textContent = msg;
  container.appendChild(el);
  el.addEventListener('animationend', function(e) {
    if (e.animationName === 'toastOut') el.remove();
  });
}

/* ── Confirm Modal ── */
function showConfirm(msg, onOk, onCancel) {
  var overlay = document.getElementById('confirm-modal');
  if (!overlay) {
    overlay = document.createElement('div');
    overlay.id = 'confirm-modal';
    overlay.className = 'modal-overlay';
    var body = createEl('div', {className: 'modal-body', id: 'confirm-body'});
    var cancelBtn = createEl('button', {className: 'btn btn-default', id: 'confirm-cancel'}, '取消');
    var okBtn = createEl('button', {className: 'btn btn-danger', id: 'confirm-ok'}, '确认');
    overlay.appendChild(createEl('div', {className: 'modal'},
      createEl('div', {className: 'modal-title'}, '确认操作'),
      body,
      createEl('div', {className: 'modal-actions'}, cancelBtn, okBtn)
    ));
    document.body.appendChild(overlay);
    overlay.addEventListener('click', function(e) {
      if (e.target === overlay) { overlay.classList.remove('active'); if (onCancel) onCancel(); }
    });
  }
  document.getElementById('confirm-body').textContent = msg;
  overlay.classList.add('active');

  var cleanup = function() { overlay.classList.remove('active'); };

  var okBtn = document.getElementById('confirm-ok');
  var cancelBtn = document.getElementById('confirm-cancel');
  var newOk = okBtn.cloneNode(true);
  var newCancel = cancelBtn.cloneNode(true);
  okBtn.parentNode.replaceChild(newOk, okBtn);
  cancelBtn.parentNode.replaceChild(newCancel, cancelBtn);

  newOk.addEventListener('click', function() { cleanup(); if (onOk) onOk(); });
  newCancel.addEventListener('click', function() { cleanup(); if (onCancel) onCancel(); });

  document.addEventListener('keydown', function escHandler(e) {
    if (e.key === 'Escape') { cleanup(); document.removeEventListener('keydown', escHandler); if (onCancel) onCancel(); }
  });
}

/* ── General Confirm ── */
function confirmAction(msg) { return confirm(msg); }

/* ── Tag Search with Dropdown ── */
function setupTagSearch(input, dropdown, onSelect) {
  input = typeof input === 'string' ? document.getElementById(input) : input;
  dropdown = typeof dropdown === 'string' ? document.getElementById(dropdown) : dropdown;
  if (!input || !dropdown) return null;

  var tags = [];
  var hideTimer;
  var highlightIdx = -1;
  var skipNextClear = false;

  function setHighlight(idx) {
    var items = dropdown.querySelectorAll('.dropdown-item');
    if (highlightIdx >= 0 && highlightIdx < items.length) {
      items[highlightIdx].classList.remove('highlighted');
    }
    highlightIdx = idx;
    if (highlightIdx >= 0 && highlightIdx < items.length) {
      items[highlightIdx].classList.add('highlighted');
      items[highlightIdx].scrollIntoView({ block: 'nearest' });
    }
  }

  function selectItem(idx, fireEvent) {
    var items = dropdown.querySelectorAll('.dropdown-item');
    if (idx >= 0 && idx < items.length) {
      var item = items[idx];
      skipNextClear = true;
      var name = item.getAttribute('data-name');
      var id = item.getAttribute('data-id');
      input.value = name;
      if (id) input.setAttribute('data-tag-id', id);
      input.classList.add('tag-selected');
      dropdown.classList.remove('active');
      highlightIdx = -1;
      if (fireEvent !== false && onSelect) onSelect(input);
    }
  }

  function renderDropdown() {
    if (onSelect) onSelect(input);
    var val = (input.value || '').trim().toLowerCase();
    if (!val) { dropdown.classList.remove('active'); highlightIdx = -1; return; }
    var filtered = tags.filter(function(t) {
      var name = (t.name || '').toLowerCase();
      var path = (t.path || t.name || '').toLowerCase();
      return name.indexOf(val) !== -1 || path.indexOf(val) !== -1;
    });
    if (filtered.length === 0) { dropdown.classList.remove('active'); highlightIdx = -1; return; }
    while (dropdown.firstChild) dropdown.removeChild(dropdown.firstChild);
    filtered.forEach(function(t) {
      var item = createEl('div', {className: 'dropdown-item', 'data-name': t.name, 'data-id': t.id || ''});
      item.appendChild(document.createTextNode((t.path || t.name) + ' '));
      item.appendChild(createEl('small', null, '(' + (t.cat || '') + ')'));
      dropdown.appendChild(item);
    });
    dropdown.classList.add('active');
    highlightIdx = -1;
    dropdown.querySelectorAll('.dropdown-item').forEach(function(item, i) {
      item.addEventListener('mouseover', function() { setHighlight(i); });
      item.addEventListener('mouseout', function() { setHighlight(-1); });
      item.addEventListener('click', function() { selectItem(i); });
    });
  }

  input.addEventListener('input', function() {
    if (skipNextClear) { skipNextClear = false; renderDropdown(); return; }
    input.removeAttribute('data-tag-id');
    input.classList.remove('tag-selected');
    renderDropdown();
  });

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

  return {
    setTags: function(t) { tags = t || []; },
    input: input,
    dropdown: dropdown
  };
}

/* ── Keyboard Shortcuts ── */
document.addEventListener('keydown', function(e) {
  if ((e.ctrlKey || e.metaKey) && e.key === 'k') {
    e.preventDefault();
    var searchInput = document.querySelector('[data-global-search]');
    if (searchInput) searchInput.focus();
  }
  if (e.key === 'Escape') {
    var activeDropdowns = document.querySelectorAll('.dropdown.active');
    activeDropdowns.forEach(function(d) { d.classList.remove('active'); });
  }
});
