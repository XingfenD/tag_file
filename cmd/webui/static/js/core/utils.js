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

/* ── Path Encoding ── */
function encodePath(path) {
  return path.split('/').map(function(s) { return encodeURIComponent(s); }).join('/');
}
