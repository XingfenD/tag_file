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
        var rawPath = window.location.pathname.substring('/files/groups/'.length);
        try { this.filePath = decodeURI(rawPath); } catch (e) { this.filePath = rawPath; }
        this.load();
      },

      detailLink: function() {
        return '/files/detail/' + encodePath(this.filePath);
      },

      load: async function() {
        this.loading = true;
        try {
          var detailData = await API.get('/api/files/detail/' + encodePath(this.filePath));
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
          if (data.ok) {
            showToast(data.msg, 'success');
            this.newGroupName = '';
            this.load();
          } else {
            showToast(data.msg, 'error');
          }
        } catch (e) {
          API.onError(e);
        }
      },

      renameGroup: async function(g, newVal) {
        var body = new URLSearchParams();
        body.set('file_path', this.filePath);
        body.set('group_name', newVal);
        var data = await API.put('/api/files/group/' + g.id, body);
        if (!data.ok) { showToast(data.msg, 'error'); throw new Error(data.msg); }
        showToast(data.msg, 'success');
        this.load();
      },

      removeGroup: function(g) {
        var self = this;
        showConfirm('确定删除标签组 «' + g.name + '»？组内所有标签也会被移除。', function() {
          var params = new URLSearchParams();
          params.set('file_path', self.filePath);
          API.del('/api/files/group/' + g.id + '?' + params.toString()).then(function(data) {
            if (data.ok) {
              showToast(data.msg, 'success');
              self.load();
            } else {
              showToast(data.msg, 'error');
            }
          }).catch(API.onError);
        });
      }
    };
  });
});
