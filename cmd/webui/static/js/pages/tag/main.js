/* ═══════════════════════════════════════════
   TagFile — Tag Page (Alpine.js)
   State, CRUD, Tree/Flat Views, SortableJS Drag
   ═══════════════════════════════════════════ */

document.addEventListener('alpine:init', function() {
  Alpine.data('tagPage', function() {
    return {
      view: 'tree',
      category: '',
      query: '',
      queryInput: '',
      page: 1,
      categories: [],
      tree: [],
      flatItems: [],
      pagination: {},
      valuesMap: {},
      loading: true,
      error: '',
      addCategoryID: '',
      addParent: '',
      addName: '',
      addDesc: '',

      /* ── Lookup helpers ── */
      catName: function(cid) {
        for (var i = 0; i < this.categories.length; i++) {
          if (this.categories[i].id === cid) return this.categories[i].name;
        }
        return '';
      },

      allTags: function() {
        var result = [];
        var self = this;
        (function walk(items) {
          items.forEach(function(item) {
            result.push({
              name: item.tag.name,
              path: item.tag.path,
              cat: self.catName(item.tag.cid),
              cid: item.tag.cid,
              id: item.tag.id
            });
            if (item.children && item.children.length > 0) walk(item.children);
          });
        })(this.tree);
        return result;
      },

      parentTagChoices: function() {
        var all = this.allTags();
        if (!this.addCategoryID) return all;
        var catID = parseInt(this.addCategoryID);
        return all.filter(function(t) { return t.cid === catID; });
      },

      sidebarItems: function() {
        var counts = {};
        (function walk(items) {
          items.forEach(function(item) {
            counts[item.tag.cid] = (counts[item.tag.cid] || 0) + 1;
            if (item.children) walk(item.children);
          });
        })(this.tree);
        var total = 0;
        for (var k in counts) total += counts[k];
        var list = [{ id: '', name: '全部', count: total }];
        this.categories.forEach(function(c) {
          list.push({ id: c.id, name: c.name, count: counts[c.id] || 0 });
        });
        return list;
      },

      /* ── Initialization ── */
      init: function() {
        this.$el.addEventListener('click', this.onPanelClick.bind(this));
        this.$watch('tree', () => this.afterTreeRender());
        this.$watch('valuesMap', () => this.afterTreeRender());
        this.loadInitial();
      },

      afterTreeRender: function() {
        this.$nextTick(this.initSortable.bind(this));
      },

      loadInitial: async function() {
        this.loading = true;
        this.error = '';
        try {
          this.categories = await API.getAllPages('/api/categories');
          await this.refresh();
        } catch (e) {
          console.error('Init error:', e);
          this.error = '加载失败，请刷新页面重试';
        } finally {
          this.loading = false;
        }
      },

      refresh: async function() {
        var params = 'view=' + this.view;
        if (this.category) params += '&category_id=' + this.category;
        if (this.query) params += '&q=' + encodeURIComponent(this.query);
        if (this.view === 'flat') params += '&page=' + this.page;

        try {
          var data = await API.get('/api/tags?' + params);
          if (!data.ok) { this.error = data.msg || '加载失败'; return; }
          this.error = '';
          if (data.view === 'tree') {
            this.tree = data.tree || [];
            this.flatItems = [];
          } else {
            this.flatItems = data.items || [];
            this.pagination = data.pagination || {};
          }
          await this.loadValues();
        } catch (err) {
          console.error(err);
          this.error = '网络请求失败';
        }
      },

      /* ── Tag values (batch) ── */
      loadValues: async function() {
        var names = [];
        var seen = {};
        (function walk(items) {
          items.forEach(function(item) {
            if (!seen[item.tag.name]) { seen[item.tag.name] = true; names.push(item.tag.name); }
            if (item.children) walk(item.children);
          });
        })(this.tree);
        this.flatItems.forEach(function(item) {
          if (!seen[item.name]) { seen[item.name] = true; names.push(item.name); }
        });
        if (names.length === 0) { this.valuesMap = {}; return; }
        try {
          var data = await API.get('/api/tags/values/batch?names=' + encodeURIComponent(names.join(',')));
          this.valuesMap = data.values || {};
        } catch (e) {
          this.valuesMap = {};
        }
      },

      valuesOf: function(name) {
        return this.valuesMap[name] || [];
      },

      /* ── View / filter state ── */
      switchView: function(view) {
        this.view = view;
        this.page = 1;
        this.refresh();
      },

      doSearch: function() {
        this.query = (this.queryInput || '').trim();
        this.page = 1;
        this.refresh();
      },

      clearSearch: function() {
        this.category = '';
        this.query = '';
        this.queryInput = '';
        this.page = 1;
        this.refresh();
      },

      selectCategory: function(cid) {
        this.category = String(cid);
        this.query = '';
        this.queryInput = '';
        this.page = 1;
        this.refresh();
      },

      goToPage: function(page) {
        this.page = page;
        this.refresh();
      },

      /* ── Tree HTML (Alpine has no recursive templates) ── */
      treeHtml: function() {
        var self = this;
        function nodeHtml(n) {
          var t = n.tag;
          var chips = (self.valuesOf(t.name) || []).map(function(v) {
            return '<span class="tag" style="cursor:pointer" data-tag-name="' + escHtml(t.name) + '" data-value="' + escHtml(v.value) + '" data-action="remove-value" title="点击删除">' +
              escHtml(v.value) + ' <span class="tag-remove">×</span></span>';
          }).join('');
          var children = '';
          if (n.children && n.children.length > 0) {
            children = '<ul class="tree-children">' + n.children.map(nodeHtml).join('') + '</ul>';
          }
          return '<li class="tag-tree-node" data-tag-id="' + t.id + '" data-tag-name="' + escHtml(t.name) + '" data-tag-cid="' + t.cid + '">' +
            '<div class="tag-tree-row-main">' +
            '<span class="drag-handle" style="font-size:12px;color:var(--ink-muted);cursor:grab;flex-shrink:0;opacity:0;transition:opacity 0.15s ease" title="拖拽排序">&#9776;</span>' +
            '<span style="font-size:14px">&#127991;</span>' +
            '<span class="tag-tree-name">' + escHtml(t.name) + '</span>' +
            '<span class="text-muted" style="font-size:11px">(' + escHtml(self.catName(t.cid)) + ')</span>' +
            '<span class="text-muted" style="font-size:10px;font-family:var(--font-mono)">#' + t.id + '</span>' +
            '<span class="tag-tree-values" data-tag-name="' + escHtml(t.name) + '" style="display:flex;gap:2px;flex-wrap:wrap">' + chips + '</span>' +
            '<span class="tree-actions">' +
            '<a href="#" data-action="edit" class="btn btn-default btn-xs" style="text-decoration:none">编辑</a>' +
            '<a href="#" data-action="add-child" class="btn btn-primary btn-xs" style="text-decoration:none">+</a>' +
            '<button data-action="delete" class="btn btn-danger btn-xs">删除</button>' +
            '</span></div>' + children + '</li>';
        }
        return this.tree.map(nodeHtml).join('');
      },

      /* ── Event delegation for tree actions / value chips ── */
      onPanelClick: function(e) {
        var actionEl = e.target.closest('[data-action]');
        if (!actionEl) return;
        var action = actionEl.getAttribute('data-action');
        if (action === 'remove-value') {
          e.preventDefault();
          this.removeTagValue(actionEl.getAttribute('data-tag-name'), actionEl.getAttribute('data-value'));
          return;
        }
        var node = actionEl.closest('.tag-tree-node');
        if (!node) return;
        var tagID = parseInt(node.getAttribute('data-tag-id'));
        var tagName = node.getAttribute('data-tag-name');
        var cid = parseInt(node.getAttribute('data-tag-cid'));
        e.preventDefault();
        if (action === 'edit') this.editTreeTag(tagID, tagName);
        else if (action === 'add-child') this.quickAddChild(tagName, cid);
        else if (action === 'delete') this.deleteTag(tagID, tagName);
      },

      /* ── Flat view helpers ── */
      parentOf: function(item) {
        if (!item.path || item.path === item.name) return '';
        var idx = item.path.lastIndexOf(' / ');
        if (idx === -1) return '';
        return item.path.substring(0, idx);
      },

      /* ── SortableJS nested tree drag ── */
      initSortable: function() {
        var self = this;
        var lists = this.$el.querySelectorAll('#tag-list-wrap ul');
        lists.forEach(function(ul) {
          if (ul._sortable) return;
          ul._sortable = new Sortable(ul, {
            group: 'tag-tree',
            draggable: '.tag-tree-node',
            filter: '.tree-actions, button, a',
            delay: 80,
            forceFallback: true,
            onEnd: function(evt) {
              self.onDragEnd(evt);
            }
          });
        });
      },

      onDragEnd: function(evt) {
        /* Revert DOM immediately; server refresh re-renders on success */
        var from = evt.from;
        var ref = from.children[evt.oldIndex];
        if (ref && ref !== evt.item) from.insertBefore(evt.item, ref);
        else if (!ref) from.appendChild(evt.item);

        var srcNode = evt.item;
        var srcID = parseInt(srcNode.getAttribute('data-tag-id'));
        var srcName = srcNode.getAttribute('data-tag-name');

        var targetEl = document.elementFromPoint(
          evt.originalEvent.clientX, evt.originalEvent.clientY);
        if (targetEl) {
          var targetNode = targetEl.closest('.tag-tree-node');
          if (targetNode) {
            var targetID = parseInt(targetNode.getAttribute('data-tag-id'));
            var targetName = targetNode.getAttribute('data-tag-name');
            if (targetID !== srcID) { this.doDragMove(srcID, srcName, targetName); return; }
          }
        }
      },

      doDragMove: async function(tagID, tagName, parentName) {
        var resp = await API.put('/api/tags/' + tagID, {
          tag_name: tagName,
          parent_tag_name: parentName
        });
        if (resp.ok) {
          this.refresh();
          showToast(resp.msg || '标签已移动到 «' + parentName + '» 下', 'success');
        } else {
          showToast(resp.msg || '移动失败', 'error');
        }
      },

      /* ── CRUD ── */
      submitAddTag: async function() {
        var data = {
          category_id: this.addCategoryID,
          tag_name: this.addName,
          parent_tag_name: this.addParent
        };
        if (this.addDesc) data.description = this.addDesc;
        var resp = await API.post('/api/tags', data);
        if (resp.ok) {
          this.addParent = '';
          this.addName = '';
          this.addDesc = '';
          this.refresh();
          showToast(resp.msg || '标签已添加', 'success');
        } else {
          showToast(resp.msg || '添加失败', 'error');
        }
      },

      deleteTag: function(tagID, tagName) {
        var self = this;
        showConfirm('确定删除标签 «' + tagName + '»？', function() {
          API.del('/api/tags/' + tagID).then(function(resp) {
            if (resp.ok) {
              self.refresh();
              showToast(resp.msg || '标签已删除', 'info');
            } else {
              showToast(resp.msg || '删除失败', 'error');
            }
          }).catch(function() { self.refresh(); });
        });
      },

      editTreeTag: function(tagID, tagName) {
        var self = this;
        Modal.show({
          title: '编辑标签',
          fields: [
            { label: '标签名', name: 'tag_name', value: tagName, required: true }
          ],
          onSave: async function(values) {
            var newName = values.tag_name.trim();
            if (!newName) { showToast('标签名不能为空', 'error'); return false; }
            var resp = await API.put('/api/tags/' + tagID, { tag_name: newName });
            if (!resp.ok) { showToast(resp.msg || '更新失败', 'error'); return false; }
            self.refresh();
            showToast(resp.msg || '标签已更新', 'success');
          }
        });
      },

      quickAddChild: function(parentName, categoryID) {
        var self = this;
        Modal.show({
          title: '添加子标签',
          fields: [
            { label: '父标签', name: 'parent', value: parentName, readonly: true },
            { label: '子标签名', name: 'tag_name', placeholder: '子标签名', required: true }
          ],
          saveLabel: '添加',
          saveClass: 'btn btn-primary',
          onSave: async function(values) {
            var childName = values.tag_name.trim();
            if (!childName) { showToast('标签名不能为空', 'error'); return false; }
            var resp = await API.post('/api/tags', {
              category_id: categoryID,
              tag_name: childName,
              parent_tag_name: parentName
            });
            if (!resp.ok) { showToast(resp.msg || '添加失败', 'error'); return false; }
            self.refresh();
            showToast(resp.msg || '子标签已添加', 'success');
          }
        });
      },

      editFlatTag: function(item) {
        var self = this;
        var tagsForAutocomplete = [];
        if (!item.cid) {
          tagsForAutocomplete = this.allTags();
        } else {
          var catID = parseInt(item.cid);
          tagsForAutocomplete = this.allTags().filter(function(t) { return t.cid === catID && t.id !== item.id; });
        }
        Modal.show({
          title: '编辑标签',
          fields: [
            { label: '标签名', name: 'tag_name', value: item.name, required: true },
            { label: '父标签（可选）', name: 'parent_tag_name', value: this.parentOf(item),
              placeholder: '搜索...', autocomplete: { tags: tagsForAutocomplete } },
            { label: '描述', name: 'description', value: item.description || '' }
          ],
          size: 'lg',
          onSave: async function(values) {
            var newName = values.tag_name.trim();
            if (!newName) { showToast('标签名不能为空', 'error'); return false; }
            var resp = await API.put('/api/tags/' + item.id, {
              tag_name: newName,
              description: (values.description || '').trim(),
              parent_tag_name: (values.parent_tag_name || '').trim()
            });
            if (!resp.ok) { showToast(resp.msg || '更新失败', 'error'); return false; }
            self.refresh();
            showToast(resp.msg || '标签已更新', 'success');
          }
        });
      },

      /* ── Tag values ── */
      addTagValueModal: function(tagName) {
        var self = this;
        Modal.show({
          title: '添加预设值',
          fields: [
            { label: '标签', name: 'tag', value: tagName, readonly: true },
            { label: '新预设值', name: 'value', placeholder: '输入新值', required: true }
          ],
          saveLabel: '添加',
          saveClass: 'btn btn-primary',
          onSave: async function(values) {
            var val = values.value.trim();
            if (!val) { showToast('值不能为空', 'error'); return false; }
            var resp = await API.post('/api/tags/values', { tag_name: tagName, value: val });
            if (!resp.ok) { showToast(resp.msg || '添加失败', 'error'); return false; }
            self.refresh();
            showToast(resp.msg || '预设值已添加', 'success');
          }
        });
      },

      removeTagValue: function(tagName, val) {
        var self = this;
        showConfirm('确定删除值 «' + val + '»？', function() {
          var params = 'tag_name=' + encodeURIComponent(tagName) + '&value=' + encodeURIComponent(val);
          API.del('/api/tags/values/0?' + params).then(function(resp) {
            if (resp.ok) {
              self.refresh();
              showToast(resp.msg || '预设值已删除', 'info');
            } else {
              showToast(resp.msg || '删除失败', 'error');
            }
          });
        });
      }
    };
  });
});
