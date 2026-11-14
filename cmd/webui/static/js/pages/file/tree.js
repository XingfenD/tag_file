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
      expandedDirs: new Set(),
      /* 批量识别：勾选的是已索引媒体文件的相对路径（与 File.FilePath 同形） */
      selectedPaths: [],
      aiSubmitting: false,

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
        this.expandedDirs = new Set();
        this.selectedPaths = [];
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
          /* 每行都留勾选格，只有已索引的媒体文件才有复选框：非媒体（音频、
             文档…）给了也只会被入队闸门跳过，未索引文件压根没有路径可入队。 */
          var pick = '<span class="tree-pick">' +
            ((n.indexed && self.isAiMedia(n.name))
              ? '<input type="checkbox" data-ai-pick="1" data-path="' + escHtml(n.path || '') + '" title="选中后可批量识别">'
              : '') + '</span>';
          var done = (n.indexed && n.tag_done) ? '<span class="tree-status tree-status-yes">已完成</span>' : '';
          var ai = self.aiBadgeHtml(n.ai_status);
          var cnt = (n.indexed && n.tag_cnt > 0) ? '<span class="badge">' + n.tag_cnt + '</span>' : '';
          var unindexed = n.indexed ? '' : '<span class="tree-status tree-status-no">未索引</span>';
          return '<div class="' + cls + '">' +
            pick +
            '<span class="tree-icon">' + icon + '</span>' +
            '<span class="tree-name">' + nameHtml + '</span>' +
            done + ai + cnt +
            '<span class="tree-size">' + formatSize(n.file_size || 0) + '</span>' +
            unindexed + '</div>';
        }
        var sorted = this.sortedNodes(this.tree);
        if (!sorted || sorted.length === 0) return '';
        return sorted.map(nodeHtml).join('');
      },

      /* ── AI 徽章 ──
         ai_status 只有 running/pending/approved/failed 四个值，空（键缺失）表示
         从未识别过，不渲染任何东西——不发明第五种状态。它是「已完成」之外的独立
         维度，插在 done 之后，绝不替换或遮蔽手工状态。 */
      aiBadgeHtml: function(status) {
        var cls = '', label = '';
        switch (status) {
          case 'running':  cls = 'tree-status-ai-running';  label = '识别中'; break;
          case 'pending':  cls = 'tree-status-ai-pending';  label = '待审'; break;
          case 'approved': cls = 'tree-status-ai-approved'; label = 'AI 通过'; break;
          case 'failed':   cls = 'tree-status-ai-failed';   label = '识别失败'; break;
          default: return '';
        }
        return '<span class="tree-status ' + cls + '">' + label + '</span>';
      },

      /* 扩展名白名单与 internal/service/ai/media.go 的 IsMediaFile 一致：
         树接点不带 MIME，只能按名字判断；不在表里的文件给了复选框也只会被
         入队闸门跳过。 */
      isAiMedia: function(name) {
        if (!name) return false;
        var dot = name.lastIndexOf('.');
        if (dot < 0) return false;
        var ext = name.slice(dot).toLowerCase();
        return ['.jpg', '.jpeg', '.png', '.webp',
                '.mp4', '.mkv', '.mov', '.avi', '.webm', '.m4v'].indexOf(ext) >= 0;
      },

      afterRender: function() {
        var self = this;
        var container = this.$el.querySelector('#file-tree .tree-root');
        if (!container) return;
        this.bindPickEvents(container);
        this.syncPicks(container);
        container.querySelectorAll('.tree-dir[data-tree-path]').forEach(function(el) {
          var dirPath = el.getAttribute('data-tree-path');
          var key = 'tagfile_tree_open:' + dirPath;
          if (sessionStorage.getItem(key) === '1') el.open = true;
          el.addEventListener('toggle', function() {
            sessionStorage.setItem(key, el.open ? '1' : '0');
            if (el.open && !self.expandedDirs.has(dirPath)) {
              self.expandDir(el, dirPath);
            }
          });
        });
      },

      /* ── 批量识别勾选 ──
         x-html 生成的行里没有 Alpine 作用域，事件一律委托到 .tree-root 容器：
         容器自己不会被重渲染替换（x-html 只换 innerHTML），所以只绑一次。
         复选框落在 .tree-file 行内，既不在 <summary> 里也不是链接，但 click
         仍拦一道 stopPropagation，免得日后行上加了点击行为就顺带展开目录或跳转。 */
      bindPickEvents: function(container) {
        if (container.dataset.aiPickBound === '1') return;
        container.dataset.aiPickBound = '1';
        var self = this;
        container.addEventListener('click', function(e) {
          if (e.target.closest('input[data-ai-pick]')) e.stopPropagation();
        });
        container.addEventListener('change', function(e) {
          var el = e.target.closest('input[data-ai-pick]');
          if (!el) return;
          self.togglePick(el.getAttribute('data-path'), el.checked);
        });
      },

      // 勾选状态由 selectedPaths 单向同步到 DOM：复选框自身就是视图状态，
      // 不必为它重渲染整棵树。
      syncPicks: function(container) {
        var self = this;
        container.querySelectorAll('input[data-ai-pick]').forEach(function(el) {
          el.checked = self.selectedPaths.indexOf(el.getAttribute('data-path')) !== -1;
        });
      },

      togglePick: function(path, checked) {
        if (!path) return;
        var rest = this.selectedPaths.filter(function(p) { return p !== path; });
        this.selectedPaths = checked ? rest.concat([path]) : rest;
      },

      submitSelected: async function() {
        if (this.aiSubmitting || this.selectedPaths.length === 0) return;
        this.aiSubmitting = true;
        try {
          // paths 用重复键：URLSearchParams 实例会被 API.post 原样透传，
          // 换成对象字面量里的数组只会拼成一个逗号串。
          var p = new URLSearchParams();
          this.selectedPaths.forEach(function(path) { p.append('paths', path); });
          var d = await API.post('/api/ai/tasks', p);
          // API.post 对非 2xx 不抛（400 也照样 r.json()），成败只能看 ok。
          if (!d || !d.ok) { showToast((d && d.msg) || '发起识别失败', 'error'); return; }
          showToast(d.msg || ('已加入队列 ' + (d.queued || 0) + ' 个'), 'success');
          var skipped = d.skipped || [];
          if (skipped.length > 0) showToast(this.skipSummary(skipped), 'info');
          this.selectedPaths = [];
          this.load();
        } catch (e) {
          API.onError(e);
        } finally {
          this.aiSubmitting = false;
        }
      },

      skipSummary: function(skipped) {
        var counts = {}, order = [];
        skipped.forEach(function(s) {
          var reason = s.reason || '未知原因';
          if (!(reason in counts)) { counts[reason] = 0; order.push(reason); }
          counts[reason]++;
        });
        return '跳过 ' + skipped.length + ' 个: ' +
          order.map(function(r) { return r + ' ×' + counts[r]; }).join(', ');
      },

      expandDir: async function(detailsEl, dirPath) {
        this.expandedDirs.add(dirPath);
        var childrenEl = detailsEl.querySelector('.tree-children');
        if (!childrenEl) {
          childrenEl = document.createElement('div');
          childrenEl.className = 'tree-children';
          detailsEl.appendChild(childrenEl);
        }

        try {
          var data = await API.get('/api/files/expand?dir=' + encodeURIComponent(dirPath));
          if (!data.ok || !data.unindexed) return;

          var html = '';
          for (var i = 0; i < data.unindexed.length; i++) {
            var u = data.unindexed[i];
            html += '<div class="tree-file tree-unindexed">' +
              '<span class="tree-name"><span style="color:var(--ink-muted)">' + escHtml(u.name) + '</span></span>' +
              '<span class="tree-size">' + formatSize(u.size || 0) + '</span>' +
              '<span class="tree-status tree-status-no">未索引</span></div>';
          }
          if (html) childrenEl.insertAdjacentHTML('beforeend', html);
        } catch (e) {
          console.error('expand dir failed:', e);
        }
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
