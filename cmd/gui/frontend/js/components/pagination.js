// pagination.js - Pagination component
function renderPagination(pg, pageFn) {
    if (!pg || pg.totalPages <= 1) return '';
    var html = '<div class="pagination">';
    if (pg.hasPrev) {
        html += '<button class="btn-default btn-sm" onclick="' + pageFn + '(' + pg.prevPage + ')">上一页</button> ';
    }
    html += '<span>第 ' + pg.page + '/' + pg.totalPages + ' 页 (共 ' + pg.total + ' 条)</span> ';
    if (pg.hasNext) {
        html += '<button class="btn-default btn-sm" onclick="' + pageFn + '(' + pg.nextPage + ')">下一页</button>';
    }
    html += '</div>';
    return html;
}
