/* ═══════════════════════════════════════════
   TagFile — File Page: State & Init
   ═══════════════════════════════════════════ */

/* ── State ── */
var filePath = '';
var currentGroupID = 0;
var currentPage = 1;
var allTags = [];

/* ── Page Dispatch ── */
document.addEventListener('DOMContentLoaded', function() {
    var path = window.location.pathname;
    if (path === '/files' || path === '/files/') {
        initFileTreePage();
    } else if (path.indexOf('/files/detail/') === 0) {
        initFileDetailPage();
    } else if (path.indexOf('/files/groups/') === 0) {
        initFileGroupsPage();
    }
});
