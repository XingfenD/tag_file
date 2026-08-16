/* ═══════════════════════════════════════════
   TagFile — Category Page (Alpine.js)
   ═══════════════════════════════════════════ */

document.addEventListener('alpine:init', function() {
  Alpine.data('categoryPage', function() {
    return {
      q: '',
      queryInput: '',
      page: 1,
      items: [],
      pagination: {},
      addName: '',
      addDesc: '',
      loading: true,

      init: function() {
        this.refresh();
      },

      refresh: async function() {
        this.loading = true;
        try {
          var url = '/api/categories?page=' + this.page + '&page_size=10';
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

      submitAdd: async function() {
        var d = await API.post('/api/categories', { name: this.addName, description: this.addDesc });
        if (!d.ok) { showToast(d.msg || '添加失败', 'error'); return; }
        showToast(d.msg || '分类已添加', 'success');
        this.addName = '';
        this.addDesc = '';
        this.refresh();
      },

      updateCategory: async function(item, patch) {
        var d = await API.put('/api/categories/' + item.id, {
          name: patch.name !== undefined ? patch.name : item.name,
          description: patch.description !== undefined ? patch.description : (item.desc || '')
        });
        if (!d.ok) { showToast(d.msg || '更新失败', 'error'); throw new Error(d.msg); }
        showToast(d.msg || '分类已更新', 'success');
        this.refresh();
      },

      removeCategory: function(item) {
        var self = this;
        showConfirm('确定删除分类 «' + item.name + '»？', function() {
          API.del('/api/categories/' + item.id).then(function(d) {
            if (!d.ok) { showToast(d.msg || '删除失败', 'error'); return; }
            showToast(d.msg || '分类已删除', 'success');
            self.refresh();
          }).catch(API.onError);
        });
      }
    };
  });
});
