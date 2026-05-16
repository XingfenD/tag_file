/* ═══════════════════════════════════════════
   TagFile — Global Form Modal (zero innerHTML)
   ═══════════════════════════════════════════ */

var Modal = (function() {
  var overlay, container, titleEl, bodyEl, saveBtn, cancelBtn, loadEl;
  var onSaveFn = null;
  var fieldSetups = [];

  function init() {
    overlay = document.createElement('div');
    overlay.className = 'modal-overlay';
    overlay.addEventListener('click', function(e) {
      if (e.target === overlay) close();
    });

    container = document.createElement('div');
    container.className = 'modal modal-form';

    var header = document.createElement('div');
    header.className = 'modal-header';
    titleEl = document.createElement('h3');
    titleEl.className = 'modal-title';
    header.appendChild(titleEl);

    bodyEl = document.createElement('div');
    bodyEl.className = 'modal-body';

    loadEl = document.createElement('div');
    loadEl.className = 'modal-loader';
    loadEl.textContent = '...';
    loadEl.style.display = 'none';

    var footer = document.createElement('div');
    footer.className = 'modal-footer';
    saveBtn = document.createElement('button');
    saveBtn.className = 'btn btn-primary';
    saveBtn.textContent = '保存';
    saveBtn.addEventListener('click', handleSave);

    cancelBtn = document.createElement('button');
    cancelBtn.className = 'btn btn btn-default';
    cancelBtn.textContent = '取消';
    cancelBtn.addEventListener('click', close);

    footer.appendChild(saveBtn);
    footer.appendChild(cancelBtn);

    container.appendChild(header);
    container.appendChild(bodyEl);
    container.appendChild(loadEl);
    container.appendChild(footer);

    overlay.appendChild(container);
    document.body.appendChild(overlay);
  }

  function loading(on) {
    bodyEl.style.display = on ? 'none' : '';
    loadEl.style.display = on ? '' : 'none';
    saveBtn.disabled = on;
    saveBtn.textContent = on ? '保存中...' : '保存';
  }

  function readFields() {
    var values = {};
    var inputs = bodyEl.querySelectorAll('[data-field]');
    inputs.forEach(function(el) {
      var name = el.getAttribute('data-field');
      if (el.type === 'checkbox') {
        values[name] = el.checked;
      } else {
        values[name] = el.value || '';
      }
    });
    return values;
  }

  async function handleSave() {
    if (!onSaveFn) { close(); return; }
    loading(true);
    try {
      var values = readFields();
      var result = await onSaveFn(values);
      if (result !== false) close();
    } catch (e) {
      console.error('Modal save error:', e);
    } finally {
      loading(false);
    }
  }

  /* ── Form Builder (createElement, no innerHTML) ── */
  function buildField(field) {
    var group = document.createElement('div');
    group.className = 'form-group';

    if (field.label) {
      var label = document.createElement('label');
      label.textContent = field.label;
      group.appendChild(label);
    }

    var input;

    if (field.type === 'textarea') {
      input = document.createElement('textarea');
      input.rows = field.rows || 3;
    } else if (field.type === 'select') {
      input = document.createElement('select');
      if (field.options) {
        field.options.forEach(function(opt) {
          var option = document.createElement('option');
          option.value = opt.value;
          option.textContent = opt.label;
          if (opt.value === field.value) option.selected = true;
          input.appendChild(option);
        });
      }
    } else {
      input = document.createElement('input');
      input.type = field.type || 'text';
    }

    input.setAttribute('data-field', field.name);

    if (field.value != null && field.type !== 'select') {
      input.value = field.value;
    }
    if (field.placeholder) input.placeholder = field.placeholder;
    if (field.required) input.required = true;
    if (field.readonly) {
      input.readOnly = true;
      input.style.background = 'var(--paper)';
      input.style.color = 'var(--ink-muted)';
    }
    if (!field.type || field.type === 'text' || field.type === 'password') {
      input.style.width = '100%';
    }

    group.appendChild(input);

    /* Autocomplete dropdown */
    if (field.autocomplete) {
      var dropdown = document.createElement('div');
      dropdown.className = 'dropdown';
      dropdown.setAttribute('data-dropdown-for', field.name);
      group.appendChild(dropdown);
      group.style.position = 'relative';
      fieldSetups.push(function() {
        var ctrl = setupTagSearch(input, dropdown);
        if (field.autocomplete.tags) ctrl.setTags(field.autocomplete.tags);
        if (field.autocomplete.setup) field.autocomplete.setup(input, dropdown, ctrl);
      });
    }

    if (field.help) {
      var help = document.createElement('span');
      help.className = 'text-muted';
      help.style.cssText = 'font-size:11px;display:block;margin-top:2px';
      help.textContent = field.help;
      group.appendChild(help);
    }

    return group;
  }

  function buildForm(fields) {
    var form = document.createElement('form');
    fields.forEach(function(f) {
      form.appendChild(buildField(f));
    });
    return form;
  }

  function clearBody() {
    var child;
    while ((child = bodyEl.firstChild)) {
      bodyEl.removeChild(child);
    }
  }

  function show(opts) {
    titleEl.textContent = opts.title || '';
    bodyEl.style.display = '';
    loadEl.style.display = 'none';
    fieldSetups = [];

    clearBody();

    if (opts.fields) {
      var form = buildForm(opts.fields);
      bodyEl.appendChild(form);
    } else if (typeof opts.body === 'string') {
      bodyEl.innerHTML = opts.body;
    } else if (opts.body instanceof Node) {
      bodyEl.appendChild(opts.body);
    }

    onSaveFn = opts.onSave || null;
    if (opts.saveLabel) saveBtn.textContent = opts.saveLabel;
    else saveBtn.textContent = '保存';
    saveBtn.className = opts.saveClass || 'btn btn-primary';
    saveBtn.disabled = false;

    if (opts.size === 'lg') container.classList.add('modal-lg');
    else container.classList.remove('modal-lg');

    overlay.classList.add('active');
    container.classList.add('active');

    /* Run autocomplete setups after DOM attached */
    setTimeout(function() {
      fieldSetups.forEach(function(fn) { fn(); });
      fieldSetups = [];
      var firstInput = bodyEl.querySelector('input:not([readonly]), select, textarea');
      if (firstInput) firstInput.focus();
    }, 120);
  }

  function close() {
    overlay.classList.remove('active');
    container.classList.remove('active');
    clearBody();
    onSaveFn = null;
    fieldSetups = [];
  }

  document.addEventListener('keydown', function(e) {
    if (!container || !container.classList.contains('active')) return;
    if (e.key === 'Escape') {
      close();
    } else if (e.key === 'Enter') {
      var activeDropdown = container.querySelector('.dropdown.active');
      if (activeDropdown) return;
      var tagInput = document.activeElement;
      if (tagInput && tagInput.classList.contains('tag-selected')) return;
      if (document.activeElement && document.activeElement.tagName === 'TEXTAREA') return;
      e.preventDefault();
      handleSave();
    }
  });

  init();

  return {
    show: show,
    close: close,
    getField: function(name) {
      return bodyEl.querySelector('[data-field="' + name + '"]');
    }
  };
})();
