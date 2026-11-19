/* ═══════════════════════════════════════════
   TagFile — Category Page (Alpine.js)
   ═══════════════════════════════════════════ */

document.addEventListener('alpine:init', function() {
  Alpine.data('categoryPage', function() {
    return Object.assign(pagedList({
      url: function(self) {
        return '/api/categories?page=' + self.page + '&page_size=10' +
          (self.q ? '&q=' + encodeURIComponent(self.q) : '');
      }
    }), {
      addName: '',
      addDesc: '',

      init: function() {
        this.refresh();
      },

      submitAdd: async function() {
        var d = await API.post('/api/categories', { name: this.addName, description: this.addDesc });
        if (!reportResult(d, '分类已添加', '添加失败')) return;
        this.addName = '';
        this.addDesc = '';
        this.refresh();
      },

      updateCategory: async function(item, patch) {
        var d = await API.put('/api/categories/' + item.id, {
          name: patch.name !== undefined ? patch.name : item.name,
          description: patch.description !== undefined ? patch.description : (item.desc || '')
        });
        reportResultOrThrow(d, '分类已更新', '更新失败');
        this.refresh();
      },

      removeCategory: function(item) {
        var self = this;
        confirmDelete('确定删除分类 ' + item.name + '？', '/api/categories/' + item.id, {
          name: item.name, okMsg: '分类已删除', failMsg: '删除失败', onOk: function() { self.refresh(); }
        });
      }
    });
  });
});
