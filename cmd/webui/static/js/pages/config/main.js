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

      init: function() {
        this.refresh();
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
      }
    };
  });
});
