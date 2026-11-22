/* ═══════════════════════════════════════════
   TagFile — 目录侧栏宽度调节（纯 DOM，无 Alpine）
   把手 .dir-resize 夹在 aside.dir-sidebar 与主区之间（.detail-layout 的缝）：
   拖拽实时改宽，180–480px 夹住；宽度写进根元素 CSS 变量 --dir-sidebar-w，
   两个共用 .dir-sidebar 的页面（工作台 / 标签页）天然同一份宽度；松手落
   localStorage 的 tagfile_dir_width——这是偏好不是会话状态，关浏览器也该在。
   双击把手或 Enter 回默认；←/→ 步进 16px。窄屏堆叠布局下把手由 CSS 隐藏，
   变量留着也不生效（media query 把 width 钉回 auto）。
   另负责导航侧栏的折叠按钮 #nav-collapse：切根元素 nav-collapsed 类并落
   localStorage 的 tagfile_sidebar_collapsed（首帧由 base.html head 内联脚本恢复）。
   ═══════════════════════════════════════════ */
(function () {
  var KEY = 'tagfile_dir_width';
  var MIN = 180, MAX = 480, STEP = 16;

  function clamp(w) { return Math.min(MAX, Math.max(MIN, w)); }

  function apply(w) {
    if (w == null) document.documentElement.style.removeProperty('--dir-sidebar-w');
    else document.documentElement.style.setProperty('--dir-sidebar-w', w + 'px');
  }

  function currentW(aside) {
    if (aside) return Math.round(aside.getBoundingClientRect().width);
    var v = parseInt(document.documentElement.style.getPropertyValue('--dir-sidebar-w'), 10);
    return isNaN(v) ? MIN : v;
  }

  var stored = parseInt(localStorage.getItem(KEY), 10);
  if (!isNaN(stored)) apply(clamp(stored));

  var collapseBtn = document.getElementById('nav-collapse');
  if (collapseBtn) collapseBtn.addEventListener('click', function () {
    var collapsed = document.documentElement.classList.toggle('nav-collapsed');
    localStorage.setItem('tagfile_sidebar_collapsed', collapsed ? '1' : '0');
  });

  document.querySelectorAll('.dir-resize').forEach(function (el) {
    var aside = el.previousElementSibling;

    function save() { localStorage.setItem(KEY, String(currentW(aside))); }

    el.addEventListener('pointerdown', function (e) {
      e.preventDefault();
      var startX = e.clientX, startW = currentW(aside);
      el.classList.add('dragging');
      el.setPointerCapture(e.pointerId);
      function move(ev) { apply(clamp(startW + (ev.clientX - startX))); }
      function up() {
        el.classList.remove('dragging');
        el.removeEventListener('pointermove', move);
        el.removeEventListener('pointerup', up);
        el.removeEventListener('pointercancel', up);
        save();
      }
      el.addEventListener('pointermove', move);
      el.addEventListener('pointerup', up);
      el.addEventListener('pointercancel', up);
    });

    el.addEventListener('dblclick', function () {
      localStorage.removeItem(KEY);
      apply(null);
    });

    el.addEventListener('keydown', function (e) {
      if (e.key === 'ArrowLeft' || e.key === 'ArrowRight') {
        e.preventDefault();
        apply(clamp(currentW(aside) + (e.key === 'ArrowRight' ? STEP : -STEP)));
        save();
      } else if (e.key === 'Enter') {
        localStorage.removeItem(KEY);
        apply(null);
      }
    });
  });
})();
