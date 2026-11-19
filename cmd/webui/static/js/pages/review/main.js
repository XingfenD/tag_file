/* ═══════════════════════════════════════════
   TagFile — AI Review Queue Page (Alpine.js)
   发起识别 / 任务状态（SSE 推送）/ 待审建议（多选 + 批量 + 逐条编辑）
   ═══════════════════════════════════════════ */

document.addEventListener('alpine:init', function() {
  Alpine.data('reviewPage', function() {
  /* 任务表与建议表共用同一条加载骨架：端点、状态键、推送文案都记在这张表里，
     loadList 只写一份。silent（推送触发的静默刷新）不翻 loading 标志也不弹 toast；
     越界 clamp 以 JS 状态为准退到最后一页重取。stampOf 是指纹闸（见 loadPaged）：
     必须覆盖行渲染读到的每个字段和分页位置，漏一个字段就是「变了却不重画」。 */
  var LISTS = {
    tasks: {
      url: '/api/ai/tasks',
      pageKey: 'taskPage', itemsKey: 'tasks', pagKey: 'taskPagination', loadingKey: 'taskLoading',
      errMsg: '任务加载失败', warnMsg: '[review] 任务静默刷新失败，下一次继续:',
      stampKey: 'taskStamp',
      stampOf: function(items, p) {
        return (p.page || 1) + '|' + (p.total || 0) + '|' + items.map(function(t) {
          return [t.task_id, t.status, t.error || '', t.frame_count, t.pending_count,
                  t.updated_at, t.file_path || '', t.file_name || ''].join(':');
        }).join(',');
      }
    },
    suggestions: {
      url: '/api/ai/suggestions',
      pageKey: 'sugPage', itemsKey: 'suggestions', pagKey: 'sugPagination', loadingKey: 'sugLoading',
      errMsg: '建议加载失败', warnMsg: '[review] 建议静默刷新失败，下一次继续:',
      stampKey: 'sugStamp',
      stampOf: function(items, p) {
        return (p.page || 1) + '|' + (p.total || 0) + '|' + items.map(function(s) {
          return [s.suggestion_id, s.review_status, s.final_category_name, s.category_name,
                  s.final_tag_name, s.tag_name, s.final_value, s.tag_value,
                  s.matched_tag_path || '', s.file_path || ''].join(':');
        }).join(',');
      },
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
      es: null,
      timer: null,

      /* ── 待审建议 ── */
      suggestions: [],
      sugPage: 1,
      sugPagination: {},
      sugLoading: true,
      sugStamp: '',
      selected: sel.selected,
      allTags: [],
      busy: false,

      /* ── Lifecycle ── */
      init: function() {
        var self = this;
        this.refresh();
        /* SSE 长连接：服务端在任务/建议状态真正变动时推 change 事件（外加低频保活），
           空闲页面零请求零重画。断线由 EventSource 按服务端 retry 自动重连，
           重连即收到初始事件补齐错过的变更。没有 EventSource 的环境退回轮询。 */
        if (window.EventSource) {
          this.es = new EventSource('/api/ai/events');
          this.es.addEventListener('change', function() { self.onPush(); });
        } else {
          this.timer = setInterval(function() { self.tick(); }, 3000);
        }
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

      // 离开组件时关掉长连接/定时器，别让它们对着已卸载的组件继续刷新。
      destroy: function() {
        if (this.es) this.es.close();
        this.es = null;
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

      /* 推送到达：两张表都静默重取。事件不带载荷（列表端点仍是唯一事实源），
         审核动作只动建议不动任务指纹，所以不能只看任务表；stampOf 保证数据
         没变的那张表一个 DOM 节点都不重建。 */
      onPush: async function() {
        await this.loadTasks(true);
        await this.loadSuggestions(true);
      },

      /* 无 EventSource 环境的轮询兜底：任务指纹动过才顺带刷建议。 */
      tick: async function() {
        var before = this.taskStamp;
        await this.loadTasks(true);
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
            '<td class="col-id" data-label="任务">' + escHtml(t.task_id) + '</td>' +
            '<td data-label="文件">' + fileCell + '</td>' +
            '<td data-label="状态">' + self.taskBadgeHtml(t) + '</td>' +
            '<td data-label="帧数">' + escHtml(t.frame_count) + '</td>' +
            '<td data-label="待审" title="该文件尚未审核的建议数">' + escHtml(t.pending_count) + '</td>' +
            '<td class="text-muted" data-label="更新时间" style="font-size:12px">' + escHtml(self.fmtTime(t.updated_at)) + '</td>' +
            '<td class="col-actions" data-label="操作">' +
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
            '<td data-label=""><input type="checkbox" data-action="toggle" data-id="' + escHtml(id) + '"' +
              (self.selected.indexOf(id) !== -1 ? ' checked' : '') + '></td>' +
            '<td data-label="文件">' + fileCell + '</td>' +
            '<td data-label="分类">' + self.valueCell(s.final_category_name, s.category_name) + '</td>' +
            '<td data-label="标签">' + self.valueCell(s.final_tag_name, s.tag_name) + '</td>' +
            '<td data-label="值">' + self.valueCell(s.final_value, s.tag_value) + '</td>' +
            '<td data-label="识别匹配">' + matchCell + '</td>' +
            '<td class="col-actions" data-label="操作"><button class="btn btn-default btn-sm" data-action="edit" data-id="' +
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
