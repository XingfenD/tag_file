/* ═══════════════════════════════════════════
   TagFile — Theme Toggle (Glassmorphism)
   ═══════════════════════════════════════════ */

(function () {
  var STORAGE_KEY = 'tagfile-theme';

  function getSystemTheme() {
    return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
  }

  function getTheme() {
    return localStorage.getItem(STORAGE_KEY) || getSystemTheme();
  }

  function applyTheme(theme) {
    // 图标与文案都由 CSS 按 data-theme 切换，head 中执行时 body 尚未存在
    document.documentElement.setAttribute('data-theme', theme);
  }

  // Apply immediately to prevent flash
  applyTheme(getTheme());

  // Expose toggle function globally
  window.toggleTheme = function () {
    var current = getTheme();
    var next = current === 'dark' ? 'light' : 'dark';
    localStorage.setItem(STORAGE_KEY, next);
    applyTheme(next);
  };

  // Listen for system theme changes
  window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', function (e) {
    if (!localStorage.getItem(STORAGE_KEY)) {
      applyTheme(e.matches ? 'dark' : 'light');
    }
  });
})();
