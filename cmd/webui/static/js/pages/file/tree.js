/* ═══════════════════════════════════════════
   TagFile — File Tree Page (Alpine.js)
   ═══════════════════════════════════════════ */

document.addEventListener('alpine:init', function() {
  Alpine.data('fileTreePage', function() {
    return {
      tree: [],
      sortField: 'name',
      sortOrder: 'asc',
      loading: true,
      error: '',
      scanDir: '',
      scanning: false,
      importing: false,
      importPath: '',

      init: function() {
        this.sortField = sessionStorage.getItem('tagfile_tree_sort_field') || 'name';
        this.sortOrder = sessionStorage.getItem('tagfile_tree_sort_order') || 'asc';
        this.$watch('treeHtml', () => this.$nextTick(() => this.afterRender()));
        this.load();
      },

      changeSort: function() {
        sessionStorage.setItem('tagfile_tree_sort_field', this.sortField);
        sessionStorage.setItem('tagfile_tree_sort_order', this.sortOrder);
      },

      get orderLabel() {
        return this.sortOrder === 'desc' ? '降序 ▾' : '升序 ▴';
      },

      toggleOrder: function() {
        this.sortOrder = this.sortOrder === 'asc' ? 'desc' : 'asc';
        this.changeSort();
      },

      sortedNodes: function(nodes) {
        if (!nodes) return nodes;
        var desc = this.sortOrder === 'desc';
        var self = this;
        nodes = nodes.slice().sort(function(a, b) {
          if (a.is_dir !== b.is_dir) return desc ? (a.is_dir ? -1 : 1) : (a.is_dir ? 1 : -1);
          var va, vb;
          switch (self.sortField) {
            case 'size': va = a.file_size || 0; vb = b.file_size || 0; break;
            case 'tagcnt': va = a.tag_cnt || 0; vb = b.tag_cnt || 0; break;
            case 'type': va = a.name.split('.').pop() || ''; vb = b.name.split('.').pop() || ''; break;
            default: va = a.name; vb = b.name;
          }
          var cmp;
          if (typeof va === 'string') cmp = desc ? vb.localeCompare(va) : va.localeCompare(vb);
          else cmp = desc ? vb - va : va - vb;
          if (cmp === 0) cmp = desc ? b.name.localeCompare(a.name) : a.name.localeCompare(b.name);
          return cmp;
        });
        nodes.forEach(function(n) { if (n.children) n.children = self.sortedNodes(n.children); });
        return nodes;
      },

      load: async function() {
        this.loading = true;
        try {
          var data = await API.get('/api/files');
          if (data.ok && data.tree) {
            this.tree = data.tree;
            this.error = '';
          } else {
            this.tree = [];
            this.error = data.msg || '加载失败';
          }
        } catch (e) {
          this.tree = [];
          this.error = '加载失败';
        } finally {
          this.loading = false;
        }
      },

      /* Recursive render — Alpine has no recursive templates */
      treeHtml: function() {
        var self = this;
        function nodeHtml(n) {
          if (n.is_dir) {
            var badge = n.tag_cnt > 0 ? '<span class="badge" style="margin-left:4px">' + n.tag_cnt + '</span>' : '';
            var counts = (typeof n.file_count === 'number')
              ? '<span class="tree-count"><span class="tree-count-done">✓' + (n.done_count || 0) + '</span>/' + n.file_count + '</span>'
              : '';
            var children = '';
            if (n.children) children = '<div class="tree-children">' + n.children.map(nodeHtml).join('') + '</div>';
            return '<details class="tree-dir" data-tree-path="' + escHtml(n.path || '') + '">' +
              '<summary class="tree-summary">' + escHtml(n.name) + '/' + badge + counts + '</summary>' +
              children + '</details>';
          }
          var cls = 'tree-file' + (n.indexed ? '' : ' tree-unindexed');
          var nameHtml = n.indexed
            ? '<a href="/files/detail/' + encodePath(n.path) + '">' + escHtml(n.name) + '</a>'
            : '<span style="color:var(--ink-muted)">' + escHtml(n.name) + '</span>';
          var icon = n.indexed ? iconForName(n.name) : '';
          var done = (n.indexed && n.tag_done) ? '<span class="tree-status tree-status-yes">已完成</span>' : '';
          var cnt = (n.indexed && n.tag_cnt > 0) ? '<span class="badge">' + n.tag_cnt + '</span>' : '';
          var unindexed = n.indexed ? '' : '<span class="tree-status tree-status-no">未索引</span>';
          return '<div class="' + cls + '">' +
            '<span class="tree-icon">' + icon + '</span>' +
            '<span class="tree-name">' + nameHtml + '</span>' +
            done + cnt +
            '<span class="tree-size">' + formatSize(n.file_size || 0) + '</span>' +
            unindexed + '</div>';
        }
        var sorted = this.sortedNodes(this.tree);
        if (!sorted || sorted.length === 0) return '';
        return sorted.map(nodeHtml).join('');
      },

      afterRender: function() {
        var self = this;
        var container = this.$el.querySelector('#file-tree .tree-root');
        if (!container) return;
        container.querySelectorAll('.tree-dir[data-tree-path]').forEach(function(el) {
          var key = 'tagfile_tree_open:' + el.getAttribute('data-tree-path');
          if (sessionStorage.getItem(key) === '1') el.open = true;
          el.addEventListener('toggle', function() {
            sessionStorage.setItem(key, el.open ? '1' : '0');
          });
        });
      },

      submitScan: async function() {
        if (this.scanning) return;
        this.scanning = true;
        try {
          var data = await API.post('/api/files/scan', { dir: this.scanDir });
          if (data.ok) { showToast(data.msg, 'success'); this.load(); }
          else showToast(data.msg, 'error');
        } catch (e) {
          API.onError(e);
        } finally {
          this.scanning = false;
        }
      },

      onFilePicked: function(evt) {
        var f = evt.target.files[0];
        this.importPath = f ? f.name : '';
      },

      submitImport: async function() {
        if (this.importing) return;
        var form = this.$refs.importForm;
        if (!form) return;
        this.importing = true;
        try {
          var r = await fetch('/api/files/import', { method: 'POST', body: new FormData(form) });
          var data = await r.json();
          if (data.ok) { showToast(data.msg, 'success'); this.load(); }
          else showToast(data.msg, 'error');
        } catch (e) {
          API.onError(e);
        } finally {
          this.importing = false;
        }
      },

      refreshTree: async function() {
        try {
          var data = await API.post('/api/files/refresh', '');
          if (data.ok) { showToast(data.msg, 'success'); this.load(); }
          else showToast(data.msg, 'error');
        } catch (e) {
          API.onError(e);
        }
      }
    };
  });
});
