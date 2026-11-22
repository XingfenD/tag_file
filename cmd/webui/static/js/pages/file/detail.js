/* ═══════════════════════════════════════════
   TagFile — File Detail Workbench (Alpine.js)
   流水线：推荐 → 看内容 → 打标 → 完成并下一个；
   也支持从 URL / 目录侧栏直接打开某个文件。
   ═══════════════════════════════════════════ */

/* Same cap as maxRandomExclude on the server. */
var RECOMMEND_SEEN_CAP = 500;

document.addEventListener('alpine:init', function() {

  /* 推送刷新的防闪烁闸（loadPaged stampOf 的详情版）：指纹覆盖工作台渲染读到的
     全部字段——文件身份/状态、媒体能力、组、当前页标签、分页位置。
     SSE 保活落到空闲页面上时指纹不变，一个 DOM 节点都不动。 */
  function detailStamp(d) {
    var f = d.file || {};
    var parts = [
      f.id, f.path, f.name, f.size, f.type, f.tag_cnt, f.tag_done ? 1 : 0, f.ai_status || '',
      d.media_type || '', d.video_playable ? 1 : 0
    ];
    (d.groups || []).forEach(function(g) {
      parts.push('g' + g.id + ':' + g.name + ':' + (g.is_default ? 1 : 0) + ':' + g.tag_count);
    });
    var p = d.pagination || {};
    parts.push('p' + (p.page || 1) + ':' + (p.total || 0) + ':' + (p.total_pages || 0));
    (d.tags || []).forEach(function(t) {
      parts.push('t' + t.id + ':' + t.tag_id + ':' + t.tag_path + ':' + (t.value || '') + ':' + t.category);
    });
    return parts.join('|');
  }

  Alpine.data('fileDetailPage', function(isRecommend) {
    return {
      filePath: '',
      detail: null,
      detailStamp: '',
      currentGroupID: 0,
      currentPage: 1,
      sortField: 'name',
      sortOrder: 'asc',
      allTags: [],
      loading: true,
      error: '',
      addTagValue: '',
      tagValueChoices: [],
      addTagSelectedId: 0,
      addTagReset: 0,
      recommend: !!isRecommend,
      recScope: 'todo',
      recSeen: [],
      recTotal: 0,
      recRemaining: 0,
      recEmpty: false,
      aiWorking: false,
      pipelineBusy: false,
      es: null,
      timer: null,

      init: function() {
        if (this.recommend) {
          this.recScope = sessionStorage.getItem('tagfile_recommend_scope') || 'todo';
          this.recSeen = this.readSeen();
        } else {
          this.filePath = pathFromLocation('/files/detail/');
        }

        var st = readSortState('tagfile_detail', { field: 'name', order: 'asc' });
        this.sortField = st.field;
        this.sortOrder = st.order;

        window.addEventListener('popstate', () => this.onPopState());

        /* AI 状态与别处审核落库都走 SSE 推送：change 事件只意味着「重读详情」，
           指纹没变就不落位。没有 EventSource 的环境退回低频轮询。 */
        var self = this;
        if (window.EventSource) {
          this.es = new EventSource('/api/ai/events');
          this.es.addEventListener('change', function() { self.silentRefresh(); });
        } else {
          this.timer = setInterval(function() { self.silentRefresh(); }, 5000);
        }

        this.bootstrap();
      },

      destroy: function() {
        if (this.es) this.es.close();
        if (this.timer) clearInterval(this.timer);
      },

      get recSeenCount() {
        return this.recSeen.length;
      },

      bootstrap: async function() {
        this.allTags = await API.loadAllTagsSafe();
        if (this.recommend) {
          await this.nextFile();
          return;
        }
        this.load();
      },

      onPopState: function() {
        if (window.location.pathname.indexOf('/files/detail/') !== 0) {
          if (this.recommend) this.nextFile();
          return;
        }
        this.filePath = pathFromLocation('/files/detail/');
        this.currentGroupID = 0;
        this.currentPage = 1;
        this.loading = true;
        this.load();
      },

      navigateToFile: function(newPath) {
        this.filePath = newPath;
        this.currentGroupID = 0;
        this.currentPage = 1;
        history.pushState({ filePath: newPath }, '', '/files/detail/' + encodePath(newPath));
        /* loading 翻 true 让工作台整块重挂载：文件名的 inlineEdit 只在 init 读一次
           value，不重挂载就会顶着上一个文件的名字。 */
        this.loading = true;
        this.load();
      },

      /* 全屏查看器（GLightbox，vendor 单文件）：每次点击新建单元素实例，
         切换文件天然不留旧 src。图片走 type image（缩放/拖拽/旋转工具条）；
         本地视频走 type inline + content（3.3.1 无 html 分支，带 content 的
         slide 就是 inline；type video 会拉 CDN 的 plyr，违无 CDN 约束）。
         src 只有数字 id，无需转义；音频与非媒体没有可放大的画面，直接不响应。 */
      openViewer: function() {
        if (!this.detail) return;
        var href = '/api/files/raw/' + this.detail.file.id;
        var el = null;
        if (this.detail.media_type === 'image') {
          el = { href: href, type: 'image' };
        } else if (this.detail.media_type === 'video' && this.detail.video_playable) {
          el = { type: 'inline', content: '<video controls autoplay playsinline src="' + href + '" style="max-width:100%;max-height:100%;margin:auto;display:block"></video>' };
        }
        if (!el) return;
        new GLightbox({ elements: [el], touchNavigation: true, loop: false }).open();
      },

      /* ── 侧栏目录树（directoryTree 组件实例经 DOM 查询获取，见 Task 2 裁决：
          x-ref 不跨嵌套 x-data）──
         查询走 document：这些方法也会被树组件作用域里的闭包（onFileClick/onLoaded）
         调起，那时 this.$el 是 #wb-dir-tree 本身，从它内部再也找不到自己。
         revealSidebar 在详情落位后把当前文件展开定位；组件未加载/未就绪静默跳过
         （其 afterRender 会按 getCurrent 补上高亮）。 */
      treeApi: function() {
        var el = document.getElementById('wb-dir-tree');
        return el ? Alpine.$data(el) : null;
      },

      revealSidebar: function() {
        var t = this.treeApi();
        if (!t || t.loading) return;   // 空树上 reveal 是无害 no-op；就绪与否只看 loading
        t.reveal(this.filePath);
      },

      refreshSidebarRow: function(f) {
        var t = this.treeApi();
        if (!t) return;
        t.refreshFileRow({path: f.path, tag_cnt: f.tag_cnt, tag_done: f.tag_done, ai_status: f.ai_status});
      },

      /* ── Recommend mode ── */
      readSeen: function() {
        var raw = readStoredJSON(sessionStorage, 'tagfile_recommend_seen', []);
        if (!Array.isArray(raw)) return [];
        return raw.filter(function(n) { return typeof n === 'number' && n > 0; });
      },

      pushSeen: function(id) {
        if (this.recSeen.indexOf(id) < 0) this.recSeen.push(id);
        if (this.recSeen.length > RECOMMEND_SEEN_CAP) {
          this.recSeen = this.recSeen.slice(-RECOMMEND_SEEN_CAP);
        }
        sessionStorage.setItem('tagfile_recommend_seen', JSON.stringify(this.recSeen));
      },

      clearSeen: function() {
        this.recSeen = [];
        sessionStorage.removeItem('tagfile_recommend_seen');
      },

      nextFile: async function() {
        sessionStorage.setItem('tagfile_recommend_scope', this.recScope);
        this.loading = true;
        this.error = '';
        this.recEmpty = false;
        try {
          var url = '/api/files/random?scope=' + encodeURIComponent(this.recScope);
          if (this.recSeen.length) url += '&exclude=' + this.recSeen.join(',');
          var data = await API.get(url);
          if (!data.ok) {
            this.error = data.msg || '推荐失败';
            this.detail = null;
            return;
          }
          if (!data.file) {
            this.recEmpty = true;
            this.detail = null;
            return;
          }

          this.recTotal = data.pool_total || 0;
          this.recRemaining = data.remaining || 0;
          if (data.exhausted) {
            this.clearSeen();
            // remaining is the pre-draw count, which is 0 here; the session just
            // restarted with this one file shown.
            this.recRemaining = Math.max(this.recTotal - 1, 0);
            showToast('这个范围内都推荐过了，重新开始', 'info');
          }
          this.pushSeen(data.file.id);
          this.filePath = data.file.path;
          this.currentGroupID = 0;
          this.currentPage = 1;
          history.pushState({ filePath: data.file.path }, '', '/files/detail/' + encodePath(data.file.path));
          await this.load();
        } catch (e) {
          this.error = '加载失败';
          this.detail = null;
          API.onError(e);
        } finally {
          this.loading = false;
        }
      },

      /* 流水线主按钮：未完成 → 标记完成再抽下一个；已完成 → 直接下一个。
         标记失败就停在当前文件，不吞掉错误继续抽。 */
      doneAndNext: async function() {
        if (this.pipelineBusy || !this.detail) return;
        if (this.detail.file.tag_done) { this.nextFile(); return; }
        this.pipelineBusy = true;
        try {
          var resp = await API.post('/api/files/done', { file_path: this.detail.file.path, done: 'true' });
          if (!reportResult(resp, '', '')) return;
          this.mirrorFileField(this.detail.file, 'tag_done', true);
          await this.nextFile();
        } catch (e) {
          API.onError(e);
        } finally {
          this.pipelineBusy = false;
        }
      },

      /* sortControls 组件的 onChange 落点：镜像回页面属性供目录树组件的 sort
         闭包读取（变更即触发组件重渲染）。存储写入与文案都由控件负责。
         服务端排序参数已无消费者（dir_files 不再渲染），不再重取详情。 */
      applySort: function(field, order) {
        this.sortField = field;
        this.sortOrder = order;
      },

      /* 详情抓取与落位单源：URL 构造走 API.fileDetail（reload 从此带 sort，修复刷新丢排序）；
         applyDetail 归一「首组自选 + detail 落位 + 指纹」；越界退页由 fetchDetail 重入完成。 */
      fetchDetail: async function() {
        var data = await API.fileDetail(this.filePath, {
          groupId: this.currentGroupID || 0,
          page: this.currentPage || 1
        });
        if (!data.ok) return { err: data.msg || '文件不存在' };
        if (!data.detail) return { err: '' };
        var d = data.detail;
        var last = clampToLastPage(d.pagination, this.currentPage);
        if (last) { this.currentPage = last; return this.fetchDetail(); }
        this.applyDetail(d);
        return { detail: d };
      },

      applyDetail: function(d) {
        if (this.currentGroupID === 0 && d.groups && d.groups.length > 0) {
          this.currentGroupID = d.groups[0].id;
        }
        this.detail = d;
        this.detailStamp = detailStamp(d);
        /* 打标/删标也经这条重取路径（reloadGroupsAndTags）：先把当前行按新
           tag_cnt 重画，再展开定位，侧栏计数才会就地跟着变。 */
        var self = this;
        this.refreshSidebarRow(d.file);
        this.$nextTick(function() { self.revealSidebar(); });
      },

      /* 推送到达：静默重取当前详情，指纹没变一个节点都不动；变了才落位。
         不翻 loading、不弹 toast、不碰加标签表单——工作台正在被打字。 */
      silentRefresh: async function() {
        if (this.loading || !this.detail || !this.filePath) return;
        try {
          var data = await API.fileDetail(this.filePath, {
            groupId: this.currentGroupID || 0,
            page: this.currentPage || 1
          });
          if (!data.ok || !data.detail) return;
          var d = data.detail;
          if (detailStamp(d) === this.detailStamp) return;
          var last = clampToLastPage(d.pagination, this.currentPage);
          if (last) { this.currentPage = last; return this.silentRefresh(); }
          this.applyDetail(d);
        } catch (e) {
          console.warn('[detail] 静默刷新失败，下一次继续:', e);
        }
      },

      /* f 与侧栏树行镜像同字段单源（tag_done / ai_status 共用）：页面数据改完
         重算指纹，树里那一行就地重建；侧栏不再渲染 dir_files，镜像循环随之外移。 */
      mirrorFileField: function(f, key, val) {
        f[key] = val;
        this.detailStamp = detailStamp(this.detail);
        this.refreshSidebarRow(f);
      },

      load: async function() {
        try {
          var r = await this.fetchDetail();
          if (r.detail === undefined) { this.error = r.err || '文件不存在'; return; }
          this.error = '';
        } catch (err) {
          var isHTTPErr = err && err.message && /^HTTP /.test(err.message);
          this.error = isHTTPErr ? '文件不存在' : '加载失败';
        } finally {
          this.loading = false;
        }
      },

      reloadGroupsAndTags: async function() {
        try {
          var r = await this.fetchDetail();
          if (r.detail === undefined) return;
        } catch (e) {
          this.error = '加载失败';
        }
      },

      tagCount: function() {
        return (this.detail && this.detail.pagination && this.detail.pagination.total) || 0;
      },

      switchGroup: function(groupID) {
        this.currentGroupID = groupID;
        this.currentPage = 1;
        this.reloadGroupsAndTags();
      },

      loadPage: function(page) {
        this.currentPage = page;
        this.reloadGroupsAndTags();
      },

      /* ── Rename（文件名行内编辑）──
         inlineEdit 的 onSave 要 throw 语义触发回滚；成功后整页跳转新路径
         （重命名改变了 URL 本体，pushState 不足以重挂载工作台）。 */
      renameFile: async function(newName) {
        var f = this.detail.file;
        var resp = await API.put('/api/files/rename', { file_path: f.path, new_name: newName });
        reportResultOrThrow(resp, '', '重命名失败');
        var idx = f.path.lastIndexOf('/');
        var newPath = idx >= 0 ? f.path.substring(0, idx + 1) + newName : newName;
        window.location.href = fileDetailHref(newPath);
      },

      /* ── Tag done toggle（直开模式）── */
      toggleTagDone: async function() {
        var f = this.detail.file;
        var next = !f.tag_done;
        try {
          var resp = await API.post('/api/files/done', { file_path: f.path, done: next ? 'true' : 'false' });
          if (!reportResult(resp, '', '')) return;
          this.mirrorFileField(f, 'tag_done', next);
        } catch (e) {
          API.onError(e);
        }
      },

      /* ── AI 识别 ──
         ai_status 是与 tag_done 完全独立的维度：徽章与按钮都只读 ai_status，
         既不写 tag_done，也不据它推断手工「已完成」。状态→类名/文案的映射不在
         这里，只在 core/utils.js 的 AI_STATUS_META 里有一份，模板直接调那两个
         全局函数（与 formatSize/iconForMime 同一批跨页共享的辅助函数）；空值
         （键缺失，即从未识别过）两边都回 ''，模板的 x-show 于是什么都不画。 */
      aiActionLabel: function() {
        var s = (this.detail && this.detail.file && this.detail.file.ai_status) || '';
        return (!s || s === 'failed') ? 'AI 识别' : '重新识别';
      },

      recognizeAi: function() {
        if (this.aiWorking || !this.detail) return;
        var f = this.detail.file;
        var s = f.ai_status || '';
        // 已有结果的（running/pending/approved）重跑会丢掉未审核的建议，先确认再带 force=true
        var force = s !== '' && s !== 'failed';
        if (!force) { this.submitAiTask(false); return; }
        var self = this;
        showConfirm('该文件已有 AI 结果，重新识别会丢弃尚未审核的建议，确定继续？', function() {
          self.submitAiTask(true);
        });
      },

      submitAiTask: async function(force) {
        var f = this.detail.file;
        this.aiWorking = true;
        try {
          // paths 用重复键形式，单个文件也只走这一条路径；force 只认字面量 "true"
          var p = new URLSearchParams();
          p.append('paths', f.path);
          if (force) p.append('force', 'true');
          var d = await API.enqueueAi(p);
          if (!d) return;
          var mine = (d.skipped || []).filter(function(s) { return s.path === f.path; });
          if (mine.length > 0) { showToast('已跳过: ' + (mine[0].reason || '未知原因'), 'error'); return; }
          showToast(d.msg || '已加入队列', 'success');
          // 乐观更新，照 toggleTagDone 的写法；详情页是 pushState 导航，不能整页刷新
          this.mirrorFileField(f, 'ai_status', 'running');
        } catch (e) {
          API.onError(e);
        } finally {
          this.aiWorking = false;
        }
      },

      /* ── Tag adding ── */
      onTagPicked: function(evt) {
        this.addTagSelectedId = parseInt(evt.id) || 0;
        this.loadTagValueChoices(evt.name);
      },

      loadTagValueChoices: async function(tagName) {
        this.addTagValue = '';
        if (!tagName) { this.tagValueChoices = []; return; }
        try {
          var data = await API.get('/api/tags/values?tag_name=' + encodeURIComponent(tagName));
          this.tagValueChoices = data.values || [];
        } catch (e) {
          this.tagValueChoices = [];
        }
      },

      submitAddTag: async function() {
        if (!this.addTagSelectedId) { showToast('请从下拉列表中选择标签', 'error'); return; }
        try {
          var data = await API.post('/api/files/tag', {
            file_path: this.filePath,
            group_id: this.currentGroupID,
            tag_id: this.addTagSelectedId,
            tag_value: this.addTagValue
          });
          if (!reportResult(data, '标签已添加', '')) return;
          this.addTagSelectedId = 0;
          this.addTagValue = '';
          this.tagValueChoices = [];
          this.addTagReset = (this.addTagReset || 0) + 1;
          this.reloadGroupsAndTags();
        } catch (e) {
          API.onError(e);
        }
      },

      removeFileTag: function(t) {
        var self = this;
        var params = new URLSearchParams();
        params.set('file_path', self.filePath);
        params.set('tag_id', t.tag_id);
        params.set('tag_value', t.value || '');
        params.set('group_id', self.currentGroupID);
        confirmDelete('确定删除该标签？', '/api/files/tag?' + params.toString(), {
          okMsg: '标签已删除', failMsg: '', onOk: function() { self.reloadGroupsAndTags(); }
        });
      },

      saveFileTagValue: async function(t, val) {
        var body = new URLSearchParams();
        body.set('id', t.id);
        body.set('value', val);
        var data = await API.put('/api/files/tag', body);
        reportResultOrThrow(data, '值已更新', '更新失败');
        this.reloadGroupsAndTags();
      },

      /* ── Groups ── */
      showAddGroup: function() {
        var self = this;
        Modal.show({
          title: '创建标签组',
          fields: [
            { label: '标签组名称', name: 'group_name', required: true, placeholder: '例如: 元数据' }
          ],
          saveLabel: '创建',
          onSave: async function(values) {
            var data = await API.post('/api/files/group', {
              file_path: self.filePath,
              group_name: values.group_name.trim()
            });
            if (!reportResult(data, '', '')) return false;
            self.reloadGroupsAndTags();
          }
        });
      },

      removeGroup: function(groupID, groupName) {
        var self = this;
        var params = new URLSearchParams();
        params.set('file_path', self.filePath);
        confirmDelete('确定删除标签组 ' + groupName + '？', '/api/files/group/' + groupID + '?' + params.toString(), {
          name: groupName, okMsg: '标签组已删除', failMsg: '',
          onOk: function() {
            if (self.currentGroupID === groupID) self.currentGroupID = 0;
            self.reloadGroupsAndTags();
          }
        });
      }
    };
  });
});