/* ═══════════════════════════════════════════
   TagFile — Home Page Stats (Alpine.js)
   ═══════════════════════════════════════════ */

document.addEventListener('alpine:init', function() {
  Alpine.data('homePage', function() {
    return {
      files: '-',
      categories: '-',
      tags: '-',
      configs: '-',

      init: async function() {
        try {
          var results = await Promise.all([
            API.get('/api/categories?page=1'),
            API.get('/api/tags?view=flat&page=1'),
            API.get('/api/files?page=1'),
            API.get('/api/config?page=1')
          ]);
          var cats = results[0];
          var tags = results[1];
          var files = results[2];
          var configs = results[3];

          this.categories = cats && cats.pagination ? cats.pagination.total : (cats && cats.items ? cats.items.length : 0);
          this.tags = tags && tags.pagination ? tags.pagination.total : (tags && tags.items ? tags.items.length : 0);
          this.configs = configs && configs.pagination ? configs.pagination.total : (configs && configs.items ? configs.items.length : 0);

          var fileCount = 0;
          if (files && files.tree) {
            (function count(nodes) {
              if (!nodes) return;
              for (var i = 0; i < nodes.length; i++) {
                var n = nodes[i];
                if (n.is_dir) count(n.children);
                else if (n.indexed) fileCount++;
              }
            })(files.tree);
          }
          this.files = fileCount;
        } catch (e) {
          this.files = '?';
          this.categories = '?';
          this.tags = '?';
          this.configs = '?';
        }
      }
    };
  });
});
