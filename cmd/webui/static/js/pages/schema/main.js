/* ═══════════════════════════════════════════
   TagFile — Schema Admin Page (Alpine.js)
   场景管理：左列场景清单，右列字段表；字段编辑弹层自带
   tagSearch（要拿标签 id，通用弹窗只回填名字）。
   值域/候选不在这配：single 读绑定标签的字典、multi 读其子标签，
   面板只是这些既有结构的取景框。
   ═══════════════════════════════════════════ */

document.addEventListener('alpine:init', function() {
  Alpine.data('schemaPage', function() {
    return {
      schemas: [],
      selectedId: 0,
      newSchemaName: '',
      allTags: [],
      categories: [],
      typeOptions: [
        { label: '单选枚举', value: 'single' },
        { label: '开关', value: 'bool' },
        { label: '多选勾选组', value: 'multi' }
      ],
      widgetOptions: [
        { label: '下拉', value: 'select' },
        { label: '星级', value: 'stars' }
      ],
      ed: {
        show: false, schemaId: 0, fieldId: 0, sortOrder: 0,
        label: '', type: 'single', widget: 'select',
        bindId: 0, bindName: '', reset: 0,
        newCat: '', newTagName: ''
      },

      init: async function() {
        var self = this;
        var jobs = await Promise.all([
          API.get('/api/schemas'),
          API.loadAllTagsSafe(),
          API.get('/api/categories').catch(function() { return {}; })
        ]);
        if (jobs[0] && jobs[0].ok) self.schemas = jobs[0].schemas || [];
        self.allTags = jobs[1];
        self.categories = (jobs[2] && jobs[2].items) || [];
        if (self.schemas.length && !self.current()) self.select(self.schemas[0].schema_id);
      },

      load: async function() {
        var d = await API.get('/api/schemas');
        if (!d.ok) { showToast(d.msg || '场景加载失败', 'error'); return; }
        this.schemas = d.schemas || [];
      },

      current: function() {
        for (var i = 0; i < this.schemas.length; i++) {
          if (this.schemas[i].schema_id === this.selectedId) return this.schemas[i];
        }
        return null;
      },

      visibleSchemas: function() {
        var c = this.current();
        return c ? [c] : [];
      },

      select: function(id) { this.selectedId = id; },

      /* ── 场景级 ── */
      addSchema: async function() {
        var name = this.newSchemaName.trim();
        if (!name) return;
        var d = await API.post('/api/schemas', { name: name });
        if (!reportResult(d, '', '')) return;
        this.newSchemaName = '';
        await this.load();
        var s = this.schemas.filter(function(x) { return x.schema_name === name; })[0];
        if (s) this.select(s.schema_id);
      },

      /* PUT 成功后同步本地对象：不同步的话，下一次 PUT（改名/描述/上移）
         会把载回来的旧值原样写回去，静默吞掉上一次编辑。 */
      saveSchemaMeta: async function(cur, name, desc) {
        var d = await API.put('/api/schemas/' + cur.schema_id,
          { name: name, description: desc, sort_order: cur.sort_order });
        reportResultOrThrow(d, '', '保存失败');
        cur.schema_name = name;
        cur.description = desc;
      },

      renameSchema: function(cur, v) {
        return this.saveSchemaMeta(cur, v, cur.description || '');
      },

      saveDesc: function(cur, v) {
        return this.saveSchemaMeta(cur, cur.schema_name, v);
      },

      canMoveSchema: function(cur, dir) {
        var i = this.schemas.indexOf(cur);
        return i >= 0 && i + dir >= 0 && i + dir < this.schemas.length;
      },

      /* 交换相邻 sort_order 后整表重取（服务端排序是唯一顺序事实） */
      moveSchema: async function(cur, dir) {
        if (!this.canMoveSchema(cur, dir)) return;
        var i = this.schemas.indexOf(cur);
        var other = this.schemas[i + dir];
        try {
          await API.put('/api/schemas/' + cur.schema_id,
            { name: cur.schema_name, description: cur.description || '', sort_order: other.sort_order });
          await API.put('/api/schemas/' + other.schema_id,
            { name: other.schema_name, description: other.description || '', sort_order: cur.sort_order });
        } catch (e) { /* 失败靠下面的重取回到服务端真相 */ }
        await this.load();
      },

      removeSchema: function(cur) {
        var self = this;
        confirmDelete('确定删除场景「' + cur.schema_name + '」？已打过的标签不受影响。',
          '/api/schemas/' + cur.schema_id, {
            name: cur.schema_name, okMsg: '场景已删除', failMsg: '',
            onOk: async function() {
              await self.load();
              if (self.selectedId === cur.schema_id && self.schemas.length) {
                self.select(self.schemas[0].schema_id);
              }
            }
          });
      },

      /* ── 字段级 ── */
      typeName: function(f) {
        if (f.type === 'single') return f.widget === 'stars' ? '单选 · 星级' : '单选';
        if (f.type === 'bool') return '开关';
        return '多选';
      },

      fieldHint: function(f) {
        if (f.bind_tag_name === '（已删除）') return '绑定已失效，请编辑重新绑定';
        if (f.type === 'single' && f.option_count === 0) return '绑定标签字典为空，面板无候选';
        if (f.type === 'single' && f.widget === 'stars' && f.option_count !== 5) {
          return '星级共 ' + f.option_count + ' 档，建议配 5 档';
        }
        if (f.type === 'multi' && f.option_count === 0) return '绑定标签还没有子标签';
        return '';
      },

      openEditor: function(cur, f) {
        this.ed = {
          show: true, schemaId: cur.schema_id,
          fieldId: f ? f.field_id : 0, sortOrder: f ? f.sort_order : 0,
          label: f ? f.label : '', type: f ? f.type : 'single',
          widget: f ? (f.widget || 'select') : 'select',
          bindId: f ? f.bind_tag_id : 0, bindName: f ? f.bind_tag_name : '',
          reset: (this.ed.reset || 0) + 1,
          newCat: this.categories.length ? String(this.categories[0].id) : '', newTagName: ''
        };
      },

      closeEditor: function() { this.ed.show = false; },

      pickBind: function(detail) {
        this.ed.bindId = detail.id || 0;
        this.ed.bindName = detail.path || detail.name || '';
      },

      /* 就地新建：走既有 POST /api/tags（不开第二写入路径），
         建完重取平表按名字回填 id。 */
      createTagNow: async function() {
        var name = this.ed.newTagName.trim();
        if (!name) { showToast('先填新标签名', 'error'); return; }
        if (!this.ed.newCat) { showToast('先选分类', 'error'); return; }
        var d = await API.post('/api/tags', { category_id: this.ed.newCat, tag_name: name });
        if (!reportResult(d, '', '')) return;
        this.allTags = await API.loadAllTagsSafe();
        for (var i = 0; i < this.allTags.length; i++) {
          if (this.allTags[i].name === name) {
            this.ed.bindId = this.allTags[i].id;
            this.ed.bindName = this.allTags[i].path || name;
            break;
          }
        }
        this.ed.newTagName = '';
      },

      saveField: async function() {
        var ed = this.ed;
        if (!ed.label.trim()) { showToast('字段名不能为空', 'error'); return; }
        if (!ed.bindId) { showToast('请先选择绑定标签', 'error'); return; }
        var payload = {
          label: ed.label.trim(), type: ed.type,
          widget: ed.type === 'single' ? ed.widget : '',
          bind_tag_id: ed.bindId
        };
        var d;
        if (ed.fieldId) {
          payload.sort_order = ed.sortOrder;
          d = await API.put('/api/schemas/fields/' + ed.fieldId, payload);
        } else {
          d = await API.post('/api/schemas/' + ed.schemaId + '/fields', payload);
        }
        if (!reportResult(d, '', '')) return;
        this.closeEditor();
        await this.load();
      },

      removeField: function(f) {
        var self = this;
        confirmDelete('确定删除字段「' + f.label + '」？该标签已打的值不受影响。',
          '/api/schemas/fields/' + f.field_id, {
            name: f.label, okMsg: '字段已删除', failMsg: '',
            onOk: function() { self.load(); }
          });
      },

      /* 相邻交换 sort_order（两次 PUT），随后重取回服务端顺序 */
      moveField: async function(cur, i, dir) {
        var j = i + dir;
        if (j < 0 || j >= cur.fields.length) return;
        var a = cur.fields[i], b = cur.fields[j];
        var pa = { label: a.label, type: a.type, widget: a.widget, bind_tag_id: a.bind_tag_id, sort_order: b.sort_order };
        var pb = { label: b.label, type: b.type, widget: b.widget, bind_tag_id: b.bind_tag_id, sort_order: a.sort_order };
        try {
          await API.put('/api/schemas/fields/' + a.field_id, pa);
          await API.put('/api/schemas/fields/' + b.field_id, pb);
        } catch (e) { /* 以重取为准 */ }
        await this.load();
      }
    };
  });
});
