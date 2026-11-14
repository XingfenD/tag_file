/* ═══════════════════════════════════════════
   TagFile — File Detail Page (Alpine.js)
   ═══════════════════════════════════════════ */

document.addEventListener('alpine:init', function() {
  Alpine.data('fileDetailPage', function() {
    return {
      filePath: '',
      detail: null,
      currentGroupID: 0,
      currentPage: 1,
      sortField: 'name',
      sortOrder: 'asc',
      allTags: [],
      loading: true,
      error: '',
      renameName: '',
      addTagValue: '',
      tagValueChoices: [],
      addTagSelectedId: 0,
      addTagName: '',
      addTagReset: 0,
      lastLoadedTagName: '',
      aiWorking: false,

      init: function() {
        var rawPath = window.location.pathname.substring('/files/detail/'.length);
        try { this.filePath = decodeURI(rawPath); } catch (e) { this.filePath = rawPath; }

        this.sortField = sessionStorage.getItem('tagfile_detail_sort_field') || 'name';
        this.sortOrder = sessionStorage.getItem('tagfile_detail_sort_order') || 'asc';

        window.addEventListener('popstate', () => this.onPopState());
        this.$el.addEventListener('click', this.onSidebarClick.bind(this));

        this.bootstrap();
      },

      bootstrap: async function() {
        try {
          this.allTags = await API.loadAllTags();
        } catch (e) {
          this.allTags = [];
        }
        this.load(false);
      },

      onPopState: function() {
        var rawPath = window.location.pathname.substring('/files/detail/'.length);
        try { this.filePath = decodeURI(rawPath); } catch (e) { this.filePath = rawPath; }
        this.currentGroupID = 0;
        this.currentPage = 1;
        this.load(false);
      },

      onSidebarClick: function(e) {
        var link = e.target.closest('#dir-indexed a');
        if (!link) return;
        e.preventDefault();
        var href = link.getAttribute('href');
        var rawPath = href.substring('/files/detail/'.length);
        var newPath;
        try { newPath = decodeURI(rawPath); } catch (_) { newPath = rawPath; }
        if (newPath !== this.filePath) this.navigateToFile(newPath);
      },

      navigateToFile: function(newPath) {
        this.filePath = newPath;
        this.currentGroupID = 0;
        this.currentPage = 1;
        history.pushState({ filePath: newPath }, '', '/files/detail/' + encodePath(newPath));
        this.load(false, true);
      },

      changeSort: function() {
        sessionStorage.setItem('tagfile_detail_sort_field', this.sortField);
        sessionStorage.setItem('tagfile_detail_sort_order', this.sortOrder);
        this.load(true);
      },

      get orderLabel() {
        return this.sortOrder === 'desc' ? '降序 ▾' : '升序 ▴';
      },

      toggleOrder: function() {
        this.sortOrder = this.sortOrder === 'asc' ? 'desc' : 'asc';
        this.changeSort();
      },

      load: async function(sortOnly, skipSidebar) {
        var url = '/api/files/detail/' + encodePath(this.filePath);
        url += '?group_id=' + (this.currentGroupID || 0) + '&page=' + (this.currentPage || 1);
        url += '&sort=' + this.sortField + '&order=' + this.sortOrder;

        try {
          var data = await API.get(url);
          if (!data.ok) {
            this.error = data.msg || '文件不存在';
            return;
          }
          this.error = '';
          var d = data.detail;
          if (this.currentGroupID === 0 && d.groups && d.groups.length > 0) {
            this.currentGroupID = d.groups[0].id;
          }
          this.detail = d;
          if (!sortOnly) this.renameName = d.file.name;
        } catch (err) {
          var isHTTPErr = err && err.message && /^HTTP /.test(err.message);
          this.error = isHTTPErr ? '文件不存在' : '加载失败';
        } finally {
          this.loading = false;
        }
      },

      reloadGroupsAndTags: async function() {
        var url = '/api/files/detail/' + encodePath(this.filePath);
        url += '?group_id=' + (this.currentGroupID || 0) + '&page=' + (this.currentPage || 1);
        try {
          var data = await API.get(url);
          if (!data.ok || !data.detail) return;
          var d = data.detail;
          if (this.currentGroupID === 0 && d.groups && d.groups.length > 0) {
            this.currentGroupID = d.groups[0].id;
          }
          this.detail = d;
        } catch (e) {
          this.error = '加载失败';
        }
      },

      previewTags: function() {
        return (this.detail && this.detail.tags || []).slice(0, 5);
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

      /* ── Rename ── */
      submitRename: async function() {
        var f = this.detail.file;
        try {
          var resp = await API.put('/api/files/rename', { file_path: f.path, new_name: this.renameName });
          if (resp.ok) {
            showToast(resp.msg, 'success');
            var oldPath = f.path;
            var idx = oldPath.lastIndexOf('/');
            var newPath = idx >= 0 ? oldPath.substring(0, idx) + '/' + this.renameName : this.renameName;
            window.location.href = '/files/detail/' + encodePath(newPath);
          } else {
            showToast(resp.msg, 'error');
          }
        } catch (e) {
          API.onError(e);
        }
      },

      /* ── Tag done toggle ── */
      toggleTagDone: async function() {
        var f = this.detail.file;
        var next = !f.tag_done;
        try {
          var resp = await API.post('/api/files/done', { file_path: f.path, done: next ? 'true' : 'false' });
          if (resp.ok) {
            f.tag_done = next;
            var sidebarFile = (this.detail.dir_files || []).find(function(d) { return d.path === f.path; });
            if (sidebarFile) sidebarFile.tag_done = next;
            showToast(resp.msg, 'success');
          } else {
            showToast(resp.msg, 'error');
          }
        } catch (e) {
          API.onError(e);
        }
      },

      /* ── AI 识别 ──
         ai_status 是与 tag_done 完全独立的维度：徽章与按钮都只读 ai_status，
         既不写 tag_done，也不据它推断手工「已完成」。空值（键缺失，即从未识别
         过）不渲染徽章。 */
      aiBadgeClass: function(status) {
        switch (status) {
          case 'running':  return 'badge-ai-running';
          case 'pending':  return 'badge-ai-pending';
          case 'approved': return 'badge-ai-approved';
          case 'failed':   return 'badge-ai-failed';
          default: return '';
        }
      },

      aiBadgeText: function(status) {
        switch (status) {
          case 'running':  return '识别中';
          case 'pending':  return '待审';
          case 'approved': return 'AI 通过';
          case 'failed':   return '识别失败';
          default: return '';
        }
      },

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
          var d = await API.post('/api/ai/tasks', p);
          // API.post 对非 2xx 不抛（400 也照样 r.json()），成败只能看 ok。
          if (!d || !d.ok) { showToast((d && d.msg) || '发起识别失败', 'error'); return; }
          var mine = (d.skipped || []).filter(function(s) { return s.path === f.path; });
          if (mine.length > 0) { showToast('已跳过: ' + (mine[0].reason || '未知原因'), 'error'); return; }
          showToast(d.msg || '已加入队列', 'success');
          // 乐观更新，照 toggleTagDone 的写法；详情页是 pushState 导航，不能整页刷新
          f.ai_status = 'running';
          var sidebarFile = (this.detail.dir_files || []).find(function(x) { return x.path === f.path; });
          if (sidebarFile) sidebarFile.ai_status = 'running';
        } catch (e) {
          API.onError(e);
        } finally {
          this.aiWorking = false;
        }
      },

      /* ── Tag adding ── */
      onTagPicked: function(evt) {
        this.addTagSelectedId = parseInt(evt.id) || 0;
        this.addTagName = evt.name;
        this.loadTagValueChoices(evt.name);
      },

      loadTagValueChoices: async function(tagName) {
        this.lastLoadedTagName = tagName;
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
          if (!data.ok) { showToast(data.msg, 'error'); return; }
          showToast(data.msg || '标签已添加', 'success');
          this.addTagSelectedId = 0;
          this.addTagName = '';
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
        showConfirm('确定删除该标签？', function() {
          var params = new URLSearchParams();
          params.set('file_path', self.filePath);
          params.set('tag_id', t.tag_id);
          params.set('tag_value', t.value || '');
          params.set('group_id', self.currentGroupID);
          API.del('/api/files/tag?' + params.toString()).then(function(data) {
            if (!data.ok) { showToast(data.msg, 'error'); return; }
            showToast(data.msg || '标签已删除', 'info');
            self.reloadGroupsAndTags();
          }).catch(API.onError);
        });
      },

      saveFileTagValue: async function(t, val) {
        var body = new URLSearchParams();
        body.set('id', t.id);
        body.set('value', val);
        var data = await API.put('/api/files/tag', body);
        if (!data.ok) { showToast(data.msg || '更新失败', 'error'); throw new Error(data.msg); }
        showToast('值已更新', 'success');
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
            if (!data.ok) { showToast(data.msg, 'error'); return false; }
            showToast(data.msg, 'success');
            self.reloadGroupsAndTags();
          }
        });
      },

      removeGroup: function(groupID, groupName) {
        var self = this;
        showConfirm('确定删除标签组 ' + groupName + '？', function() {
          var params = new URLSearchParams();
          params.set('file_path', self.filePath);
          API.del('/api/files/group/' + groupID + '?' + params.toString()).then(function(data) {
            if (!data.ok) { showToast(data.msg, 'error'); return; }
            if (self.currentGroupID === groupID) self.currentGroupID = 0;
            showToast(data.msg || '标签组已删除', 'success');
            self.reloadGroupsAndTags();
          }).catch(API.onError);
        });
      }
    };
  });
});
