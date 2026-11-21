/* ═══════════════════════════════════════════
   TagFile — directoryTree 共享组件 (Alpine.js)
   完整折叠目录树的机制单源：全树加载（GET /api/files）、逐级排序、
   <details>/<summary> 渲染、折叠记忆（tagfile_tree_open:*，两页互通）、
   未索引子项懒加载（GET /api/files/expand?dir=）、文件行构造。
   差异全部走 config 闭包注入（列表页 = 原生导航 + 勾选列；工作台 =
   就地换文件 + 当前路径高亮），页面不复制任何树逻辑。
   ═══════════════════════════════════════════ */

document.addEventListener('alpine:init', function() {

  Alpine.data('directoryTree', function(cfg) {
    cfg = cfg || {};
    return {
      tree: [],
      loading: true,
      error: '',
      expandedDirs: new Set(),

      init: function() {
        this.load();
      },

      load: async function() {
        this.loading = true;
        this.expandedDirs = new Set();
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

      /* 排序状态归页面（各自的存储键与控件），组件每次渲染现读闭包：
         闭包在元素作用域编译，读到的是合并代理上的响应式属性。 */
      sortedNodes: function(nodes) {
        if (!nodes) return nodes;
        var st = cfg.sort ? cfg.sort() : { field: 'name', order: 'asc' };
        var desc = st.order === 'desc';
        var self = this;
        nodes = nodes.slice().sort(function(a, b) {
          if (a.is_dir !== b.is_dir) return desc ? (a.is_dir ? -1 : 1) : (a.is_dir ? 1 : -1);
          var va, vb;
          switch (st.field) {
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

      /* Recursive render — Alpine has no recursive templates。
         x-html 每次重求值后在 nextTick 重挂监听与同步勾选（页面旧
         $watch('tree'/'sort'…) 列表的等价时机，且天然覆盖排序变更）；
         afterRender 只动 DOM 不回写状态，不会形成渲染循环。 */
      treeHtml: function() {
        var self = this;
        function nodeHtml(n) {
          if (!n.is_dir) return self.fileRowHtml(n);
          var badge = n.tag_cnt > 0 ? '<span class="badge" style="margin-left:4px">' + n.tag_cnt + '</span>' : '';
          var counts = (typeof n.file_count === 'number')
            ? '<span class="tree-count"><span class="tree-count-done">' + iconHtml('check') + (n.done_count || 0) + '</span>/' + n.file_count + '</span>'
            : '';
          var children = '';
          if (n.children) children = '<div class="tree-children">' + n.children.map(nodeHtml).join('') + '</div>';
          return '<details class="tree-dir" data-tree-path="' + escHtml(n.path || '') + '">' +
            '<summary class="tree-summary"><span class="tree-caret">' + iconHtml('chevron-right') + '</span>' + escHtml(n.name) + '/' + badge + counts + '</summary>' +
            children + '</details>';
        }
        this.$nextTick(function() { self.afterRender(); });
        var sorted = this.sortedNodes(this.tree);
        if (!sorted || sorted.length === 0) return '';
        return sorted.map(nodeHtml).join('');
      },

      /* 文件行的唯一构造器：整棵树（treeHtml）和懒加载追加的行（expandDir）共用
         同一个，两边的格子数与顺序才可能一致。n 只要求 name/path/indexed，其余字段
         （tag_cnt/tag_done/ai_status/file_size）缺省即不渲染对应格子。
         data-file-path 是行身份契约：reveal 高亮、refreshFileRow 就地换行、
         onFileClick 取路径都按它定位（值经 escHtml，含引号/emoji 的路径也安全）。 */
      fileRowHtml: function(n) {
        var cls = 'tree-file' + (n.indexed ? '' : ' tree-unindexed');
        var nameHtml = n.indexed
          ? '<a href="' + fileDetailHref(n.path) + '">' + escHtml(n.name) + '</a>'
          : '<span style="color:var(--ink-muted)">' + escHtml(n.name) + '</span>';
        var icon = n.indexed ? iconHtml(iconForName(n.name)) : '';
        /* 勾选列归 cfg.picks：每行都留 .tree-pick 格，只有已索引媒体文件才有复选框——
           非媒体（音频、文档…）给了也只会被入队闸门跳过，未索引文件压根没有路径可入队。 */
        var pick = cfg.picks ? '<span class="tree-pick">' +
          ((n.indexed && this.isAiMedia(n.name))
            ? '<input type="checkbox" data-ai-pick="1" data-path="' + escHtml(n.path || '') + '" title="选中后可批量识别">'
            : '') + '</span>' : '';
        var done = (n.indexed && n.tag_done) ? '<span class="tree-status tree-status-yes">已完成</span>' : '';
        var ai = this.aiBadgeHtml(n.ai_status);
        var cnt = (n.indexed && n.tag_cnt > 0) ? '<span class="badge">' + n.tag_cnt + '</span>' : '';
        var unindexed = n.indexed ? '' : '<span class="tree-status tree-status-no">未索引</span>';
        return '<div class="' + cls + '" data-file-path="' + escHtml(n.path || '') + '">' +
          pick +
          '<span class="tree-icon">' + icon + '</span>' +
          '<span class="tree-name">' + nameHtml + '</span>' +
          done + ai + cnt +
          '<span class="tree-size">' + formatSize(n.file_size || 0) + '</span>' +
          unindexed + '</div>';
      },

      /* ── AI 徽章 ── 映射只在 core/utils.js 的 AI_STATUS_META 里有一份。 */
      aiBadgeHtml: function(status) {
        var meta = aiStatusMeta(status);
        if (!meta) return '';
        return '<span class="tree-status ' + meta.badgeClass + '">' + escHtml(meta.label) + '</span>';
      },

      /* 扩展名白名单与 internal/service/ai/media.go 的 IsMediaFile 一致。 */
      isAiMedia: function(name) {
        if (!name) return false;
        var dot = name.lastIndexOf('.');
        if (dot < 0) return false;
        var ext = name.slice(dot).toLowerCase();
        return ['.jpg', '.jpeg', '.png', '.webp',
                '.mp4', '.mkv', '.mov', '.avi', '.webm', '.m4v'].indexOf(ext) >= 0;
      },

      afterRender: function() {
        var container = this.$el.querySelector('.tree-root');
        if (!container) return;
        if (cfg.picks) {
          this.bindPickEvents(container);
          this.syncPicks(container);
        }
        this.bindNavEvents(container);
        var self = this;
        container.querySelectorAll('.tree-dir[data-tree-path]').forEach(function(el) {
          var dirPath = el.getAttribute('data-tree-path');
          var key = 'tagfile_tree_open:' + dirPath;
          if (sessionStorage.getItem(key) === '1') el.open = true;
          if (el.dataset.toggleWired === '1') return;
          el.dataset.toggleWired = '1';
          el.addEventListener('toggle', function() {
            sessionStorage.setItem(key, el.open ? '1' : '0');
            if (el.open && !self.expandedDirs.has(dirPath)) {
              self.expandDir(el, dirPath);
            }
          });
        });
        if (cfg.getCurrent) this.markCurrent(cfg.getCurrent(), container);
      },

      /* x-html 只换 innerHTML，容器本身不重渲染：事件委托绑一次。 */
      bindPickEvents: function(container) {
        if (container.dataset.aiPickBound === '1') return;
        container.dataset.aiPickBound = '1';
        container.addEventListener('click', function(e) {
          if (e.target.closest('input[data-ai-pick]')) e.stopPropagation();
        });
        container.addEventListener('change', function(e) {
          var el = e.target.closest('input[data-ai-pick]');
          if (!el) return;
          if (cfg.onTogglePick) cfg.onTogglePick(el.getAttribute('data-path'), el.checked);
        });
      },

      /* 勾选状态由页面单向同步到 DOM：复选框自身就是视图状态，不为它重渲染整棵树。 */
      syncPicks: function(container) {
        var picked = cfg.getPicks ? cfg.getPicks() : [];
        container.querySelectorAll('input[data-ai-pick]').forEach(function(el) {
          el.checked = picked.indexOf(el.getAttribute('data-path')) !== -1;
        });
      },

      /* 文件点击：cfg.onFileClick 存在则拦下 <a> 就地换文件（href 保留，
         中键/新标签页仍走原生导航）；不传则维持整页跳转。 */
      bindNavEvents: function(container) {
        if (!cfg.onFileClick) return;
        if (container.dataset.navBound === '1') return;
        container.dataset.navBound = '1';
        container.addEventListener('click', function(e) {
          var a = e.target.closest('.tree-file a');
          if (!a) return;
          e.preventDefault();
          var row = a.closest('.tree-file');
          cfg.onFileClick({ path: row ? row.getAttribute('data-file-path') : '', name: a.textContent });
        });
      },

      /* ── 工作台侧：定位当前文件 ──
         展开祖先目录逐个置 open=true——toggle 监听顺势写记忆并按需懒加载
         未索引子项；已展开的没有事件，天然幂等。定位一律遍历比对属性，
         不用 CSS 选择器拼路径（文件名里的引号/反斜杠会咬人）。 */
      reveal: function(path) {
        if (!path) return;
        var container = this.$el.querySelector('.tree-root');
        if (!container) return;
        var segs = path.split('/');
        var acc = '';
        for (var i = 0; i < segs.length - 1; i++) {
          acc = i === 0 ? segs[i] : acc + '/' + segs[i];
          var dirs = container.querySelectorAll('.tree-dir');
          for (var d = 0; d < dirs.length; d++) {
            if (dirs[d].getAttribute('data-tree-path') === acc) dirs[d].open = true;
          }
        }
        this.markCurrent(path, container);
      },

      markCurrent: function(path, container) {
        container.querySelectorAll('.tree-file.tree-current').forEach(function(el) {
          el.classList.remove('tree-current');
        });
        if (!path) return;
        var rows = container.querySelectorAll('.tree-file');
        for (var i = 0; i < rows.length; i++) {
          if (rows[i].getAttribute('data-file-path') === path) {
            rows[i].classList.add('tree-current');
            rows[i].scrollIntoView({ block: 'nearest' });
            return;
          }
        }
      },

      /* 乐观更新单行：改 this.tree 里的节点数据再重造那一行，
         整树不动、滚动位置不丢。节点找不到（未扫入/已删）静默跳过。 */
      refreshFileRow: function(fields) {
        if (!fields || !fields.path) return;
        function find(nodes) {
          for (var i = 0; i < nodes.length; i++) {
            var n = nodes[i];
            if (!n.is_dir && n.path === fields.path) return n;
            if (n.children) { var hit = find(n.children); if (hit) return hit; }
          }
          return null;
        }
        var node = find(this.tree);
        if (!node) return;
        Object.keys(fields).forEach(function(k) { node[k] = fields[k]; });
        var container = this.$el.querySelector('.tree-root');
        if (!container) return;
        var rows = container.querySelectorAll('.tree-file');
        for (var i = 0; i < rows.length; i++) {
          if (rows[i].getAttribute('data-file-path') === fields.path) {
            rows[i].outerHTML = this.fileRowHtml(node);
            if (cfg.picks) this.syncPicks(container);
            return;
          }
        }
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
            // 与整棵树同一个行构造器：格子数、顺序逐一对齐，追加进来的行不会
            // 比上面的行左移（未索引 => 没有勾选框，也没有徽章可渲染）。
            html += this.fileRowHtml({ name: u.name, path: u.path, indexed: false, file_size: u.size });
          }
          if (html) childrenEl.insertAdjacentHTML('beforeend', html);
        } catch (e) {
          console.error('expand dir failed:', e);
        }
      }
    };
  });
});
