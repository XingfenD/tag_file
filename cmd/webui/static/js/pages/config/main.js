/* ═══════════════════════════════════════════
   TagFile — Config Page (Alpine.js)
   ═══════════════════════════════════════════ */

document.addEventListener('alpine:init', function() {
  Alpine.data('configPage', function() {
    return {
      q: '',
      queryInput: '',
      page: 1,
      items: [],
      pagination: {},
      cfgKey: '',
      cfgValue: '',
      cfgDesc: '',
      loading: true,

      /* ── AI 识别卡片 ──
         ai doubles as the default snapshot: a key missing from the config table
         keeps the value below, which mirrors the server-side fallbacks in
         internal/service/ai/config.go. ollama_model is intentionally optional —
         empty means "not configured" and the recognition path reports it. */
      ai: {
        ollama_url: 'http://127.0.0.1:11434',
        ollama_model: '',
        ai_frame_interval: '1',
        ai_frame_diff_threshold: '10',
        ai_max_frames: '10'
      },
      aiSaving: false,
      /* aiLoaded is the permission to save: the card writes all five keys in one go, so a
         form still holding the defaults above would replace a stored model name with an
         empty string. Only a read that has walked the whole config table turns it on; a
         key merely absent from that table is not a failure — the default is then the
         truth, matching the server-side fallback. */
      aiLoaded: false,
      aiLoadError: '',

      init: function() {
        this.refresh();
        this.loadAiConfig();
      },

      refresh: async function() {
        this.loading = true;
        try {
          var url = '/api/config?page=' + this.page + '&page_size=10';
          if (this.q) url += '&q=' + encodeURIComponent(this.q);
          var d = await API.get(url);
          if (!d.ok) { showToast(d.msg || '加载失败', 'error'); return; }
          this.items = d.items || [];
          this.pagination = d.pagination || {};
        } catch (err) {
          API.onError(err);
        } finally {
          this.loading = false;
        }
      },

      doSearch: function() {
        this.q = (this.queryInput || '').trim();
        this.page = 1;
        this.refresh();
      },

      clearSearch: function() {
        this.q = '';
        this.queryInput = '';
        this.page = 1;
        this.refresh();
      },

      goToPage: function(page) {
        this.page = page;
        this.refresh();
      },

      displayIndex: function(i) {
        return (this.pagination.page - 1) * (this.pagination.page_size || 10) + i + 1;
      },

      submitForm: async function() {
        var d = await API.post('/api/config', { key: this.cfgKey, value: this.cfgValue, description: this.cfgDesc });
        if (d.ok) {
          showToast(d.msg || '配置已保存', 'success');
          this.cfgKey = '';
          this.cfgValue = '';
          this.cfgDesc = '';
          this.refresh();
        } else {
          showToast(d.msg || '保存失败', 'error');
        }
      },

      editConfig: function(key, val) {
        var self = this;
        Modal.show({
          title: '编辑配置: ' + escHtml(key),
          fields: [
            { label: '配置键', name: 'key', value: key, readonly: true },
            { label: '配置值', name: 'value', value: val, required: true }
          ],
          onSave: async function(values) {
            var newVal = values.value.trim();
            if (!newVal) { showToast('配置值不能为空', 'error'); return false; }
            var d = await API.put('/api/config/' + encodeURIComponent(values.key), { value: newVal });
            if (!d.ok) { showToast(d.msg || '更新失败', 'error'); return false; }
            showToast(d.msg || '配置已更新', 'success');
            self.refresh();
          }
        });
      },

      removeConfig: function(key) {
        var self = this;
        showConfirm('确定删除配置 «' + key + '»？', function() {
          API.del('/api/config/' + encodeURIComponent(key)).then(function(d) {
            if (d.ok) {
              showToast(d.msg || '配置已删除', 'success');
              self.refresh();
            } else {
              showToast(d.msg || '删除失败', 'error');
            }
          }).catch(API.onError);
        });
      },

      /* ── AI 识别卡片：读取与保存 ── */
      /* The pages are walked here rather than through API.getAllPages, because that helper
         answers with a bare item array and treats a page it refuses the same as the last
         one: the caller cannot tell a truncated read from a table that really holds that
         few rows, and aiLoaded has to be sure of exactly that. */
      loadAiConfig: async function() {
        this.aiLoaded = false;
        this.aiLoadError = '';
        var stored = {};
        try {
          for (var page = 1; page <= 100; page++) {
            var d = await API.get('/api/config?page=' + page);
            if (!d.ok) {
              this.aiLoadError = d.msg || '配置列表加载失败';
              return;
            }
            var items = d.items || [];
            for (var i = 0; i < items.length; i++) stored[items[i].key] = items[i].value;
            if (!d.pagination || !d.pagination.has_next) {
              for (var key in this.ai) {
                if (stored[key] !== undefined && stored[key] !== null) this.ai[key] = String(stored[key]);
              }
              this.aiLoaded = true;
              return;
            }
          }
          this.aiLoadError = '配置条目过多，已停止读取';
        } catch (err) {
          API.onError(err);
          this.aiLoadError = '配置列表加载失败：' + (err.message || '请求出错');
        }
      },

      // API.post resolves even on HTTP 400 (only API.get throws), so the failure
      // signal is d.ok. Abandoning the loop on the first rejection leaves the keys
      // already posted saved, which the toast names together with the server msg.
      submitAiConfig: async function() {
        if (!this.aiLoaded) {
          showToast('AI 配置尚未读取成功，已取消保存', 'error');
          return;
        }
        this.aiSaving = true;
        try {
          for (var key in this.ai) {
            var d = await API.post('/api/config', { key: key, value: this.ai[key] });
            if (!d.ok) {
              showToast('保存 ' + key + ' 失败：' + (d.msg || '未知错误'), 'error');
              return;
            }
          }
          showToast('AI 配置已保存', 'success');
          this.refresh();
        } catch (err) {
          API.onError(err);
        } finally {
          this.aiSaving = false;
        }
      }
    };
  });
});
