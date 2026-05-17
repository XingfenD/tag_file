/* ═══════════════════════════════════════════
   TagFile — Click-to-Edit (Inline Editing)
   Replace cell with a styled <input>, readonly by default.
   Click to enter edit mode, blur to save, Esc to cancel.
   ═══════════════════════════════════════════ */

function makeEditable(el, opts) {
  if (!el) return;
  opts = opts || {};

  var value = el.getAttribute('data-value') || el.textContent.trim();
  var displayHTML = el.innerHTML;
  var placeholder = opts.placeholder || '';
  var emptyText = opts.emptyText || '';
  var editing = false;
  var changed = false;

  var input = document.createElement('input');
  input.type = 'text';
  input.className = 'inline-edit-input';
  input.value = value;
  input.readOnly = true;
  if (placeholder) input.placeholder = placeholder;

  el.innerHTML = '';
  el.appendChild(input);

  function enterEdit() {
    if (editing) return;
    editing = true;
    changed = false;
    input.readOnly = false;
    input.classList.add('editing');
    input.focus();
    input.select();
    document.addEventListener('mousedown', outsideClick, true);
  }

  function exitEdit() {
    editing = false;
    input.readOnly = true;
    input.classList.remove('editing');
    document.removeEventListener('mousedown', outsideClick, true);
  }

  function outsideClick(e) {
    if (el.contains(e.target)) return;
    document.removeEventListener('mousedown', outsideClick, true);
    doSave();
  }

  async function doSave() {
    var newVal = input.value.trim();
    if (newVal === value) { exitEdit(); return; }
    if (opts.validate && !opts.validate(newVal)) {
      input.classList.add('inline-edit-error');
      setTimeout(function() { input.classList.remove('inline-edit-error'); }, 600);
      return;
    }

    exitEdit();
    changed = true;

    try {
      if (opts.onSave) await opts.onSave(newVal);
      value = newVal;
      input.value = newVal || emptyText;
    } catch (e) {
      input.value = value;
    }
  }

  input.addEventListener('click', function() {
    if (!editing) enterEdit();
  });

  input.addEventListener('blur', function() {
    setTimeout(function() {
      if (editing && !el.contains(document.activeElement)) doSave();
    }, 100);
  });

  input.addEventListener('keydown', function(e) {
    if (!editing) return;
    if (e.key === 'Enter') { e.preventDefault(); doSave(); }
    if (e.key === 'Escape') {
      input.value = value;
      exitEdit();
    }
  });
}
