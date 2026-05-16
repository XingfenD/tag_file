/* ═══════════════════════════════════════════
   TagFile — Home Page Stats
   ═══════════════════════════════════════════ */

document.addEventListener('DOMContentLoaded', function() {
    function setStat(id, val) {
        if (val === undefined || val === null) val = '-';
        var el = document.getElementById(id);
        if (el) el.textContent = val;
    }

    Promise.all([
        API.get('/api/categories?page=1'),
        API.get('/api/tags?view=flat&page=1'),
        API.get('/api/files?page=1'),
        API.get('/api/config?page=1')
    ]).then(function(results) {
        var cats = results[0];
        var tags = results[1];
        var files = results[2];
        var configs = results[3];

        setStat('stat-categories', cats && cats.pagination ? cats.pagination.total : (cats && cats.items ? cats.items.length : 0));
        setStat('stat-tags', tags && tags.pagination ? tags.pagination.total : (tags && tags.items ? tags.items.length : 0));
        setStat('stat-configs', configs && configs.pagination ? configs.pagination.total : (configs && configs.items ? configs.items.length : 0));

        var fileCount = 0;
        if (files && files.tree) {
            function countIndexed(nodes) {
                if (!nodes) return;
                for (var i = 0; i < nodes.length; i++) {
                    var n = nodes[i];
                    if (n.is_dir) {
                        countIndexed(n.children);
                    } else if (n.indexed) {
                        fileCount++;
                    }
                }
            }
            countIndexed(files.tree);
        }
        setStat('stat-files', fileCount);
    }).catch(function() {
        setStat('stat-files', '?');
        setStat('stat-categories', '?');
        setStat('stat-tags', '?');
        setStat('stat-configs', '?');
    });
});
