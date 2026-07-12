/* ═══════════════════════════════════════════
   TagFile — Shared Utilities
   Loaded after api.js, before app.js
   ═══════════════════════════════════════════ */

/* ── HTML / JS String Escaping ── */
function escHtml(s) {
  if (!s && s !== 0) return '';
  return String(s).replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;').replace(/'/g,'&#39;');
}

function escJsStr(s) {
  if (!s) return '';
  return String(s).replace(/\\/g,'\\\\').replace(/'/g,"\\'").replace(/"/g,'\\"').replace(/\n/g,'\\n').replace(/\r/g,'\\r');
}

/* ── File Size Formatting ── */
function formatSize(bytes) {
  if (bytes === 0) return '0 B';
  var units = ['B', 'KB', 'MB', 'GB', 'TB'];
  var i = Math.floor(Math.log(bytes) / Math.log(1024));
  if (i >= units.length) i = units.length - 1;
  return (bytes / Math.pow(1024, i)).toFixed(i > 0 ? 1 : 0) + ' ' + units[i];
}

/* ── File Type Icons ── */
function iconForMime(mimeType) {
  if (!mimeType) return '📄';
  if (mimeType.indexOf('video/') === 0) return '🎬';
  if (mimeType.indexOf('image/') === 0) return '🖼';
  if (mimeType.indexOf('audio/') === 0) return '🎵';
  if (mimeType.indexOf('pdf') >= 0) return '📄';
  if (mimeType.indexOf('zip') >= 0 || mimeType.indexOf('tar') >= 0 || mimeType.indexOf('rar') >= 0 || mimeType.indexOf('gzip') >= 0) return '📦';
  return '📄';
}

function iconForName(name) {
  if (!name) return '📄';
  var parts = name.split('.');
  var ext = parts.length > 1 ? parts[parts.length - 1].toLowerCase() : '';
  if (['jpg','jpeg','png','gif','webp','bmp','svg'].indexOf(ext) >= 0) return '🖼';
  if (['mp4','webm','avi','mov','mkv','flv'].indexOf(ext) >= 0) return '🎬';
  if (['mp3','wav','flac','ogg','aac','wma'].indexOf(ext) >= 0) return '🎵';
  if (ext === 'pdf') return '📄';
  if (['zip','tar','gz','rar','7z'].indexOf(ext) >= 0) return '📦';
  return '📄';
}

/* ── DOM Helper ── */
function clearChildren(el) {
  el.replaceChildren();
}

function createEl(tag, attrs /*, ...children */) {
  var el = document.createElement(tag);
  if (attrs) {
    for (var k in attrs) {
      if (!attrs.hasOwnProperty(k)) continue;
      if (k === 'className') el.className = attrs[k];
      else if (k === 'textContent') el.textContent = attrs[k];
      else if (k === 'style' && typeof attrs[k] === 'string') el.style.cssText = attrs[k];
      else if (k.indexOf('on') === 0) el.addEventListener(k.substring(2).toLowerCase(), attrs[k]);
      else el.setAttribute(k, attrs[k]);
    }
  }
  for (var i = 2; i < arguments.length; i++) {
    var child = arguments[i];
    if (child == null) continue;
    if (typeof child === 'string' || typeof child === 'number') el.appendChild(document.createTextNode(String(child)));
    else if (child instanceof Node) el.appendChild(child);
  }
  return el;
}

/* ── Path Encoding ── */
function encodePath(path) {
  return path.split('/').map(function(s) { return encodeURIComponent(s); }).join('/');
}

/* ── Reusable DOM Builders ── */
function makeEmptyState(icon, text, hint) {
  var div = createEl('div', {className: 'empty'},
    createEl('div', {className: 'empty-icon', innerHTML: icon}),
    createEl('div', {className: 'empty-text'}, text)
  );
  if (hint) div.appendChild(createEl('div', {className: 'empty-hint'}, hint));
  return div;
}

function makePagination(pg, onPageFn, opts) {
  opts = opts || {};
  var prevText = opts.prevText || '« 上一页';
  var nextText = opts.nextText || '下一页 »';
  var div = createEl('div', {className: 'pagination'});
  if (pg.has_prev) {
    div.appendChild(createEl('a', {href: '#', onclick: function(e) { e.preventDefault(); onPageFn(pg.prev_page); }}, prevText));
  } else if (opts.showDisabled) {
    div.appendChild(createEl('span', {className: 'disabled'}, prevText));
  }
  var info = opts.infoText || ('第 ' + pg.page + '/' + pg.total_pages + ' 页 (' + pg.total + ' 项)');
  var infoAttrs = opts.infoClass ? {className: opts.infoClass} : {style: 'font-size:12px;color:var(--ink-muted)'};
  div.appendChild(createEl('span', infoAttrs, info));
  if (pg.has_next) {
    div.appendChild(createEl('a', {href: '#', onclick: function(e) { e.preventDefault(); onPageFn(pg.next_page); }}, nextText));
  } else if (opts.showDisabled) {
    div.appendChild(createEl('span', {className: 'disabled'}, nextText));
  }
  return div;
}

function makeActionBtn(text, className, onClick) {
  return createEl('button', {className: className, onclick: onClick}, text);
}

/* ── Stagger Animation Helper ── */
function staggerChildren(parent) {
  parent.classList.add('stagger');
  var children = parent.children;
  for (var i = 0; i < children.length; i++) {
    children[i].style.setProperty('--stagger-i', i);
  }
}
