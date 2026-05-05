// home.js - Dashboard page
function renderHome(data) {
    if (!data.fileCount && data.fileCount !== 0) {
        API.getStats().then(function(stats) {
            renderHome(stats);
        });
        return;
    }
    var html = '<h2>首页</h2><div class="stats-grid">' +
        statCard(data.fileCount, '文件总数', '📁') +
        statCard(data.categoryCount, '分类数量', '📂') +
        statCard(data.tagCount, '标签数量', '🏷') +
        statCard(data.configCount, '配置项', '⚙') +
        '</div>';
    document.getElementById('page-content').innerHTML = html;
}

function statCard(value, label, icon) {
    return '<div class="stat-card"><div class="stat-icon">' + (icon || '') + '</div>' +
        '<div class="stat-value">' + (value || 0) + '</div>' +
        '<div class="stat-label">' + label + '</div></div>';
}
