/* ═══════════════════════════════════════════
   TagFile — Click-to-Edit (Inline Editing)
   ═══════════════════════════════════════════ */

function makeEditable(el, opts) {
  if (!el) return;
  opts = opts || {};

  var displayHTML = el.innerHTML;
  var value = el.getAttribute('data-value') || el.textContent.trim();
  var saved = false;
  var input = null;

  el.style.cursor = 'pointer';
  el.title = opts.title || '点击编辑';
  el.classList.add('editable-cell');

  function restore() {
    el.innerHTML = displayHTML;
    el.style.cursor = 'pointer';
    el.classList.add('editable-cell');
    el.addEventListener('click', enterEdit);
  }

  function cancel() {
    if (saved) return;
    saved = true;
    removeOutsideListener();
    restore();
  }

  /* ── Click outside → save ── */
  function outsideClick(e) {
    if (el.contains(e.target)) return;
    removeOutsideListener();
    doSave();
  }

  function addOutsideListener() {
    document.addEventListener('mousedown', outsideClick, true);
  }

  function removeOutsideListener() {
    document.removeEventListener('mousedown', outsideClick, true);
  }

  async function doSave() {
    if (saved) return;
    var newVal = input ? input.value.trim() : '';
    if (newVal === value) { saved = true; removeOutsideListener(); restore(); return; }
    if (opts.validate && !opts.validate(newVal)) {
      if (input) {
        input.classList.add('inline-edit-error');
        setTimeout(function() { input.classList.remove('inline-edit-error'); }, 600);
      }
      return;
    }

    saved = true;
    removeOutsideListener();
    el.innerHTML = '<span class="inline-edit-spinner">...</span>';
    el.style.cursor = 'wait';
    el.removeEventListener('click', enterEdit);

    try {
      if (opts.onSave) await opts.onSave(newVal);
      value = newVal;
      displayHTML = newVal ? escHtml(newVal) : (opts.emptyText || '');
      el.innerHTML = displayHTML;
    } catch (e) {
      saved = false;
      el.innerHTML = '';
      el.style.cursor = 'default';
      el.classList.remove('editable-cell');
      el.appendChild(input);
      addOutsideListener();
      input.focus();
      return;
    }

    el.style.cursor = 'pointer';
    el.classList.add('editable-cell');
    el.addEventListener('click', enterEdit);
  }

  function enterEdit() {
    if (el.querySelector('input, select, textarea')) return;
    el.removeEventListener('click', enterEdit);
    saved = false;

    input = document.createElement('input');
    input.type = 'text';
    input.className = 'inline-edit-input';
    input.value = value;
    if (opts.placeholder) input.placeholder = opts.placeholder;
    input.style.cssText = 'width:' + Math.max(60, (value || opts.placeholder || '').length * 9 + 10) + 'px;' +
      'height:auto;padding:0;border:none;border-bottom:1px dashed var(--border);' +
      'background:transparent;font:inherit;font-size:inherit;color:inherit;' +
      'outline:none;border-radius:0;box-shadow:none';

    input.addEventListener('keydown', function(e) {
      if (e.key === 'Enter') { e.preventDefault(); doSave(); }
      if (e.key === 'Escape') { cancel(); }
    });

    /* blur as fallback (Tab, focus programmatic change) */
    input.addEventListener('blur', function() {
      setTimeout(function() {
        if (!saved && !el.contains(document.activeElement)) doSave();
      }, 100);
    });

    el.innerHTML = '';
    el.style.cursor = 'default';
    el.classList.remove('editable-cell');
    el.appendChild(input);
    input.focus();
    input.select();

    addOutsideListener();
  }

  el.addEventListener('click', enterEdit);
}
