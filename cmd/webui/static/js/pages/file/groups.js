/* ═══════════════════════════════════════════
   TagFile — File Groups Page (Alpine.js)
   ═══════════════════════════════════════════ */

document.addEventListener('alpine:init', function() {
  Alpine.data('fileGroupsPage', function() {
    return {
      filePath: '',
      fileName: '',
      groups: [],
      loading: true,
      error: '',
      newGroupName: '',

      init: function() {
        this.filePath = pathFromLocation('/files/groups/');
        this.load();
      },

      detailLink: function() {
        return fileDetailHref(this.filePath);
      },

      load: async function() {
        this.loading = true;
        try {
          var detailData = await API.fileDetail(this.filePath);
          if (detailData.ok && detailData.detail) {
            this.fileName = detailData.detail.file.name;
          }
          var data = await API.get('/api/files/groups/' + encodePath(this.filePath));
          if (data.ok && data.groups) {
            this.groups = data.groups;
            this.error = '';
          } else {
            this.groups = [];
            this.error = '加载失败';
          }
        } catch (e) {
          this.groups = [];
          this.error = '加载失败';
        } finally {
          this.loading = false;
        }
      },

      submitAddGroup: async function() {
        try {
          var data = await API.post('/api/files/group', { file_path: this.filePath, group_name: this.newGroupName });
          if (!reportResult(data, '', '')) return;
          this.newGroupName = '';
          this.load();
        } catch (e) {
          API.onError(e);
        }
      },

      renameGroup: async function(g, newVal) {
        var body = new URLSearchParams();
        body.set('file_path', this.filePath);
        body.set('group_name', newVal);
        var data = await API.put('/api/files/group/' + g.id, body);
        reportResultOrThrow(data, '', '');
        this.load();
      },

      removeGroup: function(g) {
        var self = this;
        var params = new URLSearchParams();
        params.set('file_path', self.filePath);
        confirmDelete('确定删除标签组 ' + g.name + '？组内所有标签也会被移除。', '/api/files/group/' + g.id + '?' + params.toString(), {
          name: g.name, onOk: function() { self.load(); }
        });
      }
    };
  });
});
