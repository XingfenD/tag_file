/* ═══════════════════════════════════════════
   TagFile — File Tree Page (Alpine.js)
   树的全部机制住在 core/tree.js 的 directoryTree 组件里（模板 #file-tree
   挂载）；本页只留扫描 / 导入 / 批量识别勾选这些页面职责，以及供组件闭包
   读取的排序状态。组件实例经 this.$refs.tree 取。
   ═══════════════════════════════════════════ */

document.addEventListener('alpine:init', function() {
  Alpine.data('fileTreePage', function() {
    var picks = createSelection('selectedPaths');
    return {
      /* 批量识别：勾选的是已索引媒体文件的相对路径（与 File.FilePath 同形） */
      selectedPaths: picks.selectedPaths,
      aiSubmitting: false,
      scanDir: '',
      scanning: false,
      importing: false,
      importPath: '',
      sortField: 'name',
      sortOrder: 'asc',

      init: function() {
        /* 排序控件的状态与持久化住在 core 的 sortControls 组件里（键不变：
           tagfile_tree_sort_*）；这里从同一份 sessionStorage 起初值，组件每次
           变更都经 onChange → applySort 镜像回这两个属性——directoryTree 的
           sort 闭包正是读它们，变更即触发组件重渲染。 */
        var st = readSortState('tagfile_tree', { field: 'name', order: 'asc' });
        this.sortField = st.field;
        this.sortOrder = st.order;
      },

      /* sortControls 组件的 onChange 落点：镜像回页面属性供组件闭包读取。
         存储写入与 orderLabelHtml 文案都由组件负责。 */
      applySort: function(field, order) {
        this.sortField = field;
        this.sortOrder = order;
      },

      /* ── 批量识别勾选 ──
         勾选状态归本页，组件只把 checkbox 的 change 回调进 togglePick；
         复选框自身就是视图状态，组件按需经 getPicks() 同步回 DOM。 */
      togglePick: function(path, checked) {
        if (!path) return;
        picks.toggle.call(this, path, checked);
      },

      /* 组件重取全树前清空勾选（load 是组件方法，清勾选是本页职责）。
         x-ref 注册在最近的 x-data 作用域上——嵌套作用域的 ref 页面拿不到，
         组件实例一律经 DOM 查询 + Alpine.$data 获取。 */
      reloadTree: function() {
        this.selectedPaths = [];
        var el = this.$el.querySelector('#file-tree');
        if (el) Alpine.$data(el).load();
      },

      submitSelected: async function() {
        if (this.aiSubmitting || this.selectedPaths.length === 0) return;
        this.aiSubmitting = true;
        try {
          // paths 用重复键：URLSearchParams 实例会被 API.post 原样透传，
          // 换成对象字面量里的数组只会拼成一个逗号串。
          var p = new URLSearchParams();
          this.selectedPaths.forEach(function(path) { p.append('paths', path); });
          var d = await API.enqueueAi(p);
          if (!d) return;
          // 一个都没进队列（全被闸门跳过）不算成功：服务端照样回 ok:true，
          // 但「已加入队列 0 个」用 info 播报，别把空手而归染成绿色。
          var queued = d.queued || 0;
          showToast(d.msg || ('已加入队列 ' + queued + ' 个'), queued === 0 ? 'info' : 'success');
          var skipped = d.skipped || [];
          if (skipped.length > 0) showToast(skipSummary(skipped), 'info');
          this.reloadTree();
        } catch (e) {
          API.onError(e);
        } finally {
          this.aiSubmitting = false;
        }
      },

      submitScan: async function() {
        if (this.scanning) return;
        this.scanning = true;
        try {
          // scan/refresh/import 三处服务端总带 msg，fallback 传空串保持原样逐字。
          var data = await API.post('/api/files/scan', { dir: this.scanDir });
          if (reportResult(data, '', '')) this.reloadTree();
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
          if (reportResult(data, '', '')) this.reloadTree();
        } catch (e) {
          API.onError(e);
        } finally {
          this.importing = false;
        }
      },

      refreshTree: async function() {
        try {
          var data = await API.post('/api/files/refresh', '');
          if (reportResult(data, '', '')) this.reloadTree();
        } catch (e) {
          API.onError(e);
        }
      }
    };
  });
});
