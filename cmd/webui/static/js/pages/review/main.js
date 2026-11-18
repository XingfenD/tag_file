/* ═══════════════════════════════════════════
   TagFile — AI Review Queue Page (Alpine.js)
   发起识别 / 任务状态（轮询）/ 待审建议（多选 + 批量 + 逐条编辑）
   ═══════════════════════════════════════════ */

document.addEventListener('alpine:init', function() {
  Alpine.data('reviewPage', function() {
  /* 任务表与建议表共用同一条加载骨架：端点、状态键、轮询文案都记在这张表里，
     loadList 只写一份。语义与两份逐字镜像的时代完全一致——silent（轮询调用）
     不翻 loading 标志（否则每 3 秒闪一次空态）也不弹 toast；越界 clamp 以 JS
     状态为准退到最后一页重取。 */
  var LISTS = {
    tasks: {
      url: '/api/ai/tasks',
      pageKey: 'taskPage', itemsKey: 'tasks', pagKey: 'taskPagination', loadingKey: 'taskLoading',
      errMsg: '任务加载失败', warnMsg: '[review] 任务轮询失败，下一次继续:',
      after: function(p) {
        p.taskStamp = p.tasks.map(function(t) { return t.task_id + ':' + t.status; }).join(',');
      }
    },
    suggestions: {
      url: '/api/ai/suggestions',
      pageKey: 'sugPage', itemsKey: 'suggestions', pagKey: 'sugPagination', loadingKey: 'sugLoading',
      errMsg: '建议加载失败', warnMsg: '[review] 建议轮询失败，下一次继续:',
      after: function(p) { p.pruneSelected(); }
    }
  };

  var sel = createSelection('selected');

  return {
      /* ── 发起识别 ── */
      dir: '.',
      recursive: false,
      force: false,
      submitting: false,

      /* ── 任务状态 ── */
      tasks: [],
      taskPage: 1,
      taskPagination: {},
      taskLoading: true,
      taskStamp: '',
      timer: null,

      /* ── 待审建议 ── */
      suggestions: [],
      sugPage: 1,
      sugPagination: {},
      sugLoading: true,
      selected: sel.selected,
      allTags: [],
      busy: false,

      /* ── Lifecycle ── */
      init: function() {
        var self = this;
        this.refresh();
        this.timer = setInterval(() => this.tick(), 3000);
        /* 事件委托（x-html 生成的行没有 Alpine 作用域） */
        delegate(this.$el, 'click', {
          retry: function(el, e) {
            e.preventDefault();
            var id = parseInt(el.getAttribute('data-id'), 10);
            if (!isNaN(id)) self.retryTask(id);
          },
          edit: function(el, e) {
            e.preventDefault();
            var item = self.suggestionById(parseInt(el.getAttribute('data-id'), 10));
            if (item) self.editSuggestion(item);
          }
        });
        delegate(this.$el, 'change', {
          'select-all': function(el) { self.toggleAll(el.checked); },
          toggle: function(el) {
            var id = parseInt(el.getAttribute('data-id'), 10);
            if (!isNaN(id)) self.toggleOne(id, el.checked);
          }
        });
        // 标签名自动补全的数据源；拉不到就只是没有候选，不影响审核本身。
        API.loadAllTagsSafe().then(function(tags) { self.allTags = tags; });
      },

      // 仓库其他页面没有 destroy 先例：本页面是唯一带定时器的，离开组件时必须清掉。
      destroy: function() {
        if (this.timer) clearInterval(this.timer);
        this.timer = null;
      },

      /* ── Loading ── */
      refresh: async function() {
        await this.loadTasks(false);
        await this.loadSuggestions(false);
      },

      loadTasks: async function(silent) {
        return this.loadList('tasks', silent);
      },

      loadSuggestions: async function(silent) {
        return this.loadList('suggestions', silent);
      },

      loadList: async function(which, silent) {
        return loadPaged(this, LISTS[which], silent);
      },

      tick: async function() {
        var before = this.taskStamp;
        await this.loadTasks(true);
        // 任务状态真的动过才顺带刷一遍建议：识别跑完时新建议会自己出现，
        // 而一次普通的网络抖动不会清掉用户刚勾好的选择。
        if (this.taskStamp !== before) await this.loadSuggestions(true);
      },

      goToTaskPage: function(page) {
        this.taskPage = Math.max(1, page);
        this.loadTasks(false);
      },

      goToSugPage: function(page) {
        this.sugPage = Math.max(1, page);
        this.loadSuggestions(false);
      },

      /* ── 发起识别 ── */
      submitEnqueue: async function() {
        var dir = (this.dir || '').trim();
        if (!dir) { showToast('请填写目录，. 表示扫描根目录', 'error'); return; }
        this.submitting = true;
        try {
          var d = await API.enqueueAi({
            dir: dir,
            recursive: this.recursive ? 'true' : 'false',
            force: this.force ? 'true' : 'false'
          });
          if (!d) return;
          showToast(d.msg || '已加入队列', 'success');
          var skipped = d.skipped || [];
          if (skipped.length > 0) showToast(skipSummary(skipped), 'info');
          this.refresh();
        } catch (err) {
          API.onError(err);
        } finally {
          this.submitting = false;
        }
      },

      /* ── 任务表 ── */
      taskRowsHtml: function() {
        var self = this;
        return this.tasks.map(function(t) {
          var fileCell = t.file_path
            ? '<a href="' + escHtml(fileDetailHref(t.file_path)) + '" title="' + escHtml(t.file_path) + '">' +
              escHtml(t.file_name || t.file_path) + '</a>'
            : '<span class="text-muted">文件已不在库里</span>';
          return '<tr>' +
            '<td class="col-id">' + escHtml(t.task_id) + '</td>' +
            '<td>' + fileCell + '</td>' +
            '<td>' + self.taskBadgeHtml(t) + '</td>' +
            '<td>' + escHtml(t.frame_count) + '</td>' +
            '<td title="该文件尚未审核的建议数">' + escHtml(t.pending_count) + '</td>' +
            '<td class="text-muted" style="font-size:12px">' + escHtml(self.fmtTime(t.updated_at)) + '</td>' +
            '<td class="col-actions">' +
              /* running gets the button too: a row left running by a dead process is
                 never claimed again, so without it that file has no way out. The
                 server decides which running rows are actually stalled — retrying a
                 live one comes back refused. */
              (t.status === 'failed' || t.status === 'running'
                ? '<button class="btn btn-default btn-sm" data-action="retry" data-id="' + escHtml(t.task_id) + '">重试</button>'
                : '<span class="text-muted">—</span>') +
            '</td></tr>';
        }).join('');
      },

      taskBadgeHtml: function(t) {
        var cls = 'badge';
        if (t.status === 'done') cls = 'badge badge-done';
        else if (t.status === 'failed') cls = 'badge badge-red';
        var html = '<span class="' + cls + '">' + escHtml(this.taskStatusLabel(t.status)) + '</span>';
        // Error 不是失败标记：done 的任务也可能带着「超出上限被丢弃」的说明。
        if (t.error) {
          html += ' <span class="text-muted" style="font-size:11px;cursor:default" title="' +
            escHtml(t.error) + '">' + escHtml(this.clip(t.error, 24)) + '</span>';
        }
        return html;
      },

      taskStatusLabel: function(status) {
        var map = { queued: '排队中', running: '识别中', done: '已完成', failed: '失败' };
        return map[status] || status || '—';
      },

      retryTask: async function(taskID) {
        try {
          var d = await API.post('/api/ai/tasks/retry', { task_id: taskID });
          if (reportResult(d, '任务已重新加入队列', '重试失败')) this.refresh();
        } catch (err) {
          API.onError(err);
        }
      },

      /* ── 建议表 ── */
      sugRowsHtml: function() {
        var self = this;
        return this.suggestions.map(function(s) {
          var id = s.suggestion_id;
          var fileCell = s.file_path
            ? '<a href="' + escHtml(fileDetailHref(s.file_path)) + '" title="' + escHtml(s.file_path) + '">' +
              escHtml(self.clip(s.file_path, 40)) + '</a>'
            : '<span class="text-muted">文件已不在库里</span>';
          var matchCell = s.matched_tag_path
            ? '<span class="badge badge-done" title="识别时匹配到已有标签；通过时按标签名复用，不会新建">复用</span> ' + escHtml(s.matched_tag_path)
            : '<span class="badge" title="识别时没有匹配到已有标签；若编辑成已有标签名，通过时同样复用">新词</span>';
          return '<tr>' +
            '<td><input type="checkbox" data-action="toggle" data-id="' + escHtml(id) + '"' +
              (self.selected.indexOf(id) !== -1 ? ' checked' : '') + '></td>' +
            '<td>' + fileCell + '</td>' +
            '<td>' + self.valueCell(s.final_category_name, s.category_name) + '</td>' +
            '<td>' + self.valueCell(s.final_tag_name, s.tag_name) + '</td>' +
            '<td>' + self.valueCell(s.final_value, s.tag_value) + '</td>' +
            '<td>' + matchCell + '</td>' +
            '<td class="col-actions"><button class="btn btn-default btn-sm" data-action="edit" data-id="' +
              escHtml(id) + '">编辑</button></td>' +
            '</tr>';
        }).join('');
      },

      // 生效值 + （被改过的）模型原建议；final_* 入库时已 trim，这里不再加工。
      valueCell: function(cur, raw) {
        if (!cur) return '<span class="text-muted">—</span>';
        var html = escHtml(cur);
        if (cur !== raw) {
          html += ' <span class="text-muted" style="font-size:11px;cursor:default" title="模型建议: ' +
            escHtml(raw) + '">（原 ' + escHtml(this.clip(raw || '(空)', 20)) + '）</span>';
        }
        return html;
      },

      sugIds: function() {
        return this.suggestions.map(function(s) { return s.suggestion_id; });
      },

      allSelected: function() { return sel.allSelected.call(this, this.sugIds()); },

      pruneSelected: function() { sel.prune.call(this, this.sugIds()); },

      toggleOne: function(id, checked) { sel.toggle.call(this, id, checked); },

      toggleAll: function(checked) { sel.toggleAll.call(this, this.sugIds(), checked); },

      suggestionById: function(id) {
        for (var i = 0; i < this.suggestions.length; i++) {
          if (this.suggestions[i].suggestion_id === id) return this.suggestions[i];
        }
        return null;
      },

      /* ── 批量审核 ── */
      reviewBatch: async function(kind) {
        if (this.selected.length === 0) { showToast('请先勾选建议', 'error'); return; }
        var approve = kind === 'approve';
        var params = new URLSearchParams();
        this.selected.forEach(function(id) { params.append('ids', id); });
        this.busy = true;
        try {
          var d = await API.post('/api/ai/suggestions/' + (approve ? 'approve' : 'reject'), params);
          if (!reportResult(d,
              approve ? '已通过 ' + (d.approved || 0) + ' 条建议' : '已拒绝 ' + (d.rejected || 0) + ' 条建议',
              approve ? '批量通过失败' : '批量拒绝失败')) return;
          this.reportFailures(d.failures || []);
          this.selected = [];
          this.refresh();
        } catch (err) {
          API.onError(err);
        } finally {
          this.busy = false;
        }
      },

      reportFailures: function(failures) {
        var shown = Math.min(failures.length, 3);
        for (var i = 0; i < shown; i++) {
          showToast('建议 #' + failures[i].id + ' 未处理: ' + (failures[i].msg || '未知原因'), 'error');
        }
        if (failures.length > shown) showToast('另有 ' + (failures.length - shown) + ' 条未处理', 'error');
      },

      /* ── 逐条编辑 ──
         标签名字段带自动补全：选中已有标签就是把建议合并到该标签（spec §6），
         因为 Approve 会按名复用已有标签而不是新建。 */
      editSuggestion: function(item) {
        var self = this;
        Modal.show({
          title: '编辑建议 #' + item.suggestion_id,
          size: 'lg',
          fields: [
            { label: '分类', name: 'category_name', value: item.final_category_name,
              placeholder: '已有分类名', help: '必须是已有分类：通过时不会自动新建分类。' },
            { label: '标签名', name: 'tag_name', value: item.final_tag_name, required: true,
              placeholder: '输入以搜索已有标签...', autocomplete: { tags: this.allTags },
              help: '上方输入一个全新标签名，通过时会在该分类下新建；在下方搜索框选中已有标签，通过时会合并到那个标签（复用，不新建）。' },
            { label: '值', name: 'tag_value', value: item.final_value, placeholder: '可选' }
          ],
          onSave: async function(values) {
            var tagName = (values.tag_name || '').trim();
            var d = await API.put('/api/ai/suggestions/' + item.suggestion_id, {
              category_name: (values.category_name || '').trim(),
              tag_name: tagName,
              tag_value: (values.tag_value || '').trim()
            });
            if (!reportResult(d, '建议已更新', '更新失败')) return false;
            self.refresh();
          }
        });
      },

      /* ── Formatting ── */
      fmtTime: function(s) {
        if (!s) return '—';
        var d = new Date(s);
        if (isNaN(d.getTime())) return s;
        return d.toLocaleString();
      },

      clip: function(s, n) {
        var str = s === undefined || s === null ? '' : String(s);
        return str.length > n ? str.slice(0, n) + '…' : str;
      }
    };
  });
});
