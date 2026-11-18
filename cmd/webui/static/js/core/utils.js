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

/* ── API 包络播报 ──
   respond 信封的 ok/msg 两分支 + fallback 文案是全站最高频的样板：d.msg 优先、
   无则用本波调用点的 fallback，成功 okType 默认 success（删除类传 'info'）。
   返回 d.ok 布尔，调用方决定刷新或（弹窗 onSave 里）return false。
   d 允许 undefined：与 API.post「非 2xx 也回 json」的现实对齐，空对象按失败播报。 */
function reportResult(d, okFallback, failFallback, okType) {
  if (d && d.ok) { showToast(d.msg || okFallback, okType || 'success'); return true; }
  showToast((d && d.msg) || failFallback, 'error');
  return false;
}

/* ── 删除确认统一骨架 ──
   showConfirm+API.del+reportResult 的全站单源：参数永远在 URL query
   （Go 只解析 POST/PUT/PATCH body，API.del 无 body）；成功播报默认 'info'
   （删除类语义归一）；reject 走 API.onError，失败不执行任何刷新。 */
function confirmDelete(msg, url, opts) {
  opts = opts || {};
  showConfirm(msg, function() {
    API.del(url).then(function(d) {
      if (reportResult(d, opts.okMsg || '', opts.failMsg || '', opts.okType || 'info') && opts.onOk) opts.onOk();
    }).catch(API.onError);
  });
}

/* 行内编辑的 onSave 需要 throw 语义触发回滚，reportResult 只回布尔。 */
function reportResultOrThrow(d, okFallback, failFallback, okType) {
  if (!reportResult(d, okFallback, failFallback, okType)) throw new Error((d && d.msg) || failFallback);
}

/* ── 事件委托单源（data-action 方言）──
   root 上监听 evtName，命中 [data-action] 且表里有对应 handler 才调用。
   只覆盖单 data-action 场景；复合选择器与 once-bound 特例保持各页自写。 */
function delegate(root, evtName, handlers) {
  root.addEventListener(evtName, function(e) {
    var el = e.target.closest('[data-action]');
    if (!el) return;
    var fn = handlers[el.getAttribute('data-action')];
    if (fn) fn(el, e);
  });
}

/* 多选集行为包（filter/concat 惯用式单源）：方法都按 this[key] 读写，
   宿主（Alpine 代理对象）用 .call(this,...) 转调保证响应性；初始键值供宿主直接铺进 state。 */
function createSelection(key) {
  var bag = {};
  bag[key] = [];
  bag.toggle = function(id, checked) {
    var rest = this[key].filter(function(s) { return s !== id; });
    this[key] = checked ? rest.concat([id]) : rest;
  };
  bag.toggleAll = function(ids, checked) {
    this[key] = checked ? ids.slice() : [];
  };
  bag.allSelected = function(ids) {
    return ids.length > 0 && this[key].length === ids.length;
  };
  bag.prune = function(aliveIds) {
    var alive = {};
    aliveIds.forEach(function(id) { alive[id] = true; });
    this[key] = this[key].filter(function(id) { return alive[id]; });
  };
  return bag;
}

/* ── 分页加载单源 ── */
/* 越界判据：page>1 且服务端算出的总页数非零且小于当前页。回退值即最后一页。 */
function clampToLastPage(pagination, page) {
  var pages = (pagination && pagination.total_pages) || 0;
  return page > 1 && pages > 0 && page > pages ? pages : null;
}

/* review loadList 的泛化：spec {url|fetch, pageKey, itemsKey(状态键), dataKey(信封键,缺省 items),
   pagKey, loadingKey, errMsg, warnMsg, after, clamp}。silent 轮询不翻 loading、不弹 toast、
   reject 只 console.warn；越界时退到最后一页重取（items 与 pagination 必须同源）。 */
async function loadPaged(self, spec, silent) {
  var pageKey = spec.pageKey || 'page';
  var itemsKey = spec.itemsKey || 'items';
  var dataKey = spec.dataKey || 'items';
  var pagKey = spec.pagKey || 'pagination';
  var loadingKey = spec.loadingKey || 'loading';
  var errMsg = spec.errMsg || '加载失败';
  var fetch = spec.fetch || function(s) { return API.get(spec.url + '?page=' + s[pageKey]); };
  if (!silent) self[loadingKey] = true;
  try {
    var d = await fetch(self);
    if (!d.ok) {
      if (!silent) showToast(d.msg || errMsg, 'error');
      return;
    }
    if (spec.clamp !== false) {
      var last = clampToLastPage(d.pagination, self[pageKey]);
      if (last) {
        // pagination.page 是夹回来的值，items 却是按越界 offset 取的：
        // 以 JS 状态为准，退到最后一页重取，别让翻页停在空白页上。
        self[pageKey] = last;
        return await loadPaged(self, spec, silent);
      }
    }
    self[itemsKey] = d[dataKey] || [];
    self[pagKey] = d.pagination || {};
    if (spec.after) spec.after(self);
  } catch (err) {
    if (silent) { console.warn(spec.warnMsg || '[paged] 静默加载失败，下一次继续:', err); return; }
    API.onError(err);
  } finally {
    if (!silent) self[loadingKey] = false;
  }
}

/* 标准分页页状态（searchForm/pager partial 方言：queryInput/doSearch/goToPage/pagination）。
   页面用 Object.assign(pagedList({...}), {页面私有}) 组合。 */
function pagedList(cfg) {
  var spec = {
    fetch: function(s) { return API.get(cfg.url(s)); },
    errMsg: cfg.errMsg || '加载失败',
    after: cfg.after,
  };
  return {
    q: '',
    queryInput: '',
    page: 1,
    items: [],
    pagination: {},
    loading: true,
    refresh: async function(silent) { return loadPaged(this, spec, silent); },
    doSearch: function() {
      this.q = (this.queryInput || '').trim();
      this.page = 1;
      this.refresh();
    },
    clearSearch: function() {
      this.q = '';
      this.queryInput = '';
      this.page = 1;
      this.refresh();
    },
    goToPage: function(page) {
      this.page = page;
      this.refresh();
    },
  };
}
