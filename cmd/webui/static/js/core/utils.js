/* ═══════════════════════════════════════════
   TagFile — Shared Utilities
   Loaded after api.js, before stores.js
   ═══════════════════════════════════════════ */

/* ── HTML / JS String Escaping ── */
function escHtml(s) {
  if (!s && s !== 0) return '';
  return String(s).replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;').replace(/'/g,'&#39;');
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

/* ── AI 状态徽章（跨页共享的唯一一份映射） ──
   文件树的行内状态（treeClass）、详情页与侧栏的徽章（badgeClass + label）都从
   这一张表读，四个状态只写一次：往表里加一个状态，两处页面同时生效，不必再
   在三个 switch 之间手工配对（class 配了、label 漏了会静默渲染出空徽章）。
   状态缺失（ai_status 是 omitempty，即从未识别过）或表里没有的值一律返回
   null/''，调用方据此「不渲染徽章」——不发明第五种状态。 */
var AI_STATUS_META = {
  running:  { label: '识别中',   badgeClass: 'badge-ai-running',  treeClass: 'tree-status-ai-running' },
  pending:  { label: '待审',     badgeClass: 'badge-ai-pending',  treeClass: 'tree-status-ai-pending' },
  approved: { label: 'AI 通过',  badgeClass: 'badge-ai-approved', treeClass: 'tree-status-ai-approved' },
  failed:   { label: '识别失败', badgeClass: 'badge-ai-failed',   treeClass: 'tree-status-ai-failed' }
};

function aiStatusMeta(status) {
  if (!status) return null;
  // hasOwnProperty：'constructor'/'toString' 之类的键不能顺带命中原型上的东西。
  return Object.prototype.hasOwnProperty.call(AI_STATUS_META, status) ? AI_STATUS_META[status] : null;
}

function aiBadgeClass(status) {
  var m = aiStatusMeta(status);
  return m ? m.badgeClass : '';
}

function aiBadgeText(status) {
  var m = aiStatusMeta(status);
  return m ? m.label : '';
}

/* ── 批量识别跳过播报 ──
   文件树的「批量识别」与待审队列的「发起识别」拿到同一形状的 skipped
   （[{path, reason}]），播报文案逐字相同——这份汇总住在 core，两处页面
   只调不再抄。原因按首次出现顺序聚合，计数与去重逻辑两边原本就一致。 */
function skipSummary(skipped) {
  var counts = {}, order = [];
  skipped.forEach(function(s) {
    var reason = s.reason || '未知原因';
    if (!(reason in counts)) { counts[reason] = 0; order.push(reason); }
    counts[reason]++;
  });
  return '跳过 ' + skipped.length + ' 个: ' +
    order.map(function(r) { return r + ' ×' + counts[r]; }).join(', ');
}
