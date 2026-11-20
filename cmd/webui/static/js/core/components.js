/* ═══════════════════════════════════════════
   TagFile — Alpine.js Shared Components
   tagSearch / inlineEdit / infoTip / sortControls / configGroup / selectMenu
   加载契约：只注册 alpine:init 监听，不碰其它 core 文件的全局；模板按名字
   （x-data="tagSearch(...)" 等）引用这些组件，改名要同步改全部模板。
   ═══════════════════════════════════════════ */

document.addEventListener('alpine:init', function() {

  /* ── Tag Search Autocomplete ──
     Usage:
     <div x-data="tagSearch({ tags: allTags })" @tag-selected="onPicked($event.detail)">
       <input x-model="value" @input="render" @focus="render" @keydown="onKeydown" ...>
       <div class="dropdown" :class="{active: open}">
         <template x-for="(t,i) in filtered" :key="t.id">
           <div class="dropdown-item" :class="{highlighted: i===highlight}"
                @mouseover="highlight=i" @mousedown.prevent="select(t)">
             <span x-text="t.path || t.name"></span>
             <small x-text="' (' + (t.cat || '') + ')'"></small>
           </div>
         </template>
       </div>
     </div>
  ── */
  Alpine.data('tagSearch', function(cfg) {
    cfg = cfg || {};
    return {
      value: '',
      tags: [],
      open: false,
      highlight: -1,
      selectedId: null,
      init: function() {
        this.tags = cfg.tags || [];
        this.value = (typeof cfg.value === 'function' ? cfg.value() : (cfg.value || ''));
        if (typeof cfg.reset === 'function') {
          var self = this;
          var last = cfg.reset();
          Alpine.effect(function() {
            var cur = cfg.reset();
            if (cur !== last) {
              last = cur;
              self.value = '';
              self.selectedId = null;
              self.open = false;
              self.highlight = -1;
            }
          });
        }
      },
      tagsLive: function() {
        return typeof cfg.tags === 'function' ? cfg.tags() : (cfg.tags || this.tags);
      },
      get filtered() {
        var v = (this.value || '').trim().toLowerCase();
        if (!v) return [];
        return this.tagsLive().filter(function(t) {
          var name = (t.name || '').toLowerCase();
          var path = (t.path || t.name || '').toLowerCase();
          return name.indexOf(v) !== -1 || path.indexOf(v) !== -1;
        });
      },
      render: function() {
        this.selectedId = null;
        this.open = this.filtered.length > 0;
        this.highlight = -1;
      },
      select: function(t) {
        this.value = t.name;
        this.selectedId = t.id || '';
        this.open = false;
        this.highlight = -1;
        this.$dispatch('tag-selected', { id: this.selectedId, name: this.value, tag: t });
      },
      onKeydown: function(e) {
        var items = this.filtered;
        if (e.key === 'ArrowDown') {
          e.preventDefault();
          if (!this.open) this.render();
          this.highlight = this.highlight + 1 >= items.length ? 0 : this.highlight + 1;
        } else if (e.key === 'ArrowUp') {
          e.preventDefault();
          if (!this.open) this.render();
          this.highlight = this.highlight - 1 < 0 ? items.length - 1 : this.highlight - 1;
        } else if (e.key === 'Enter' || e.key === 'Tab') {
          if (this.open && items.length > 0) {
            e.preventDefault();
            this.select(items[this.highlight >= 0 ? this.highlight : 0]);
          }
        } else if (e.key === 'Escape') {
          this.open = false;
          this.highlight = -1;
        }
      }
    };
  });

  /* ── Inline Edit ──
     用法见 templates/partials/inline_edit.html（inlineEditCell partial，四处行内编辑单元格单源）。
     裸写骨架：<span x-data="inlineEdit({ value: t.name, onSave: (v) => rename(v) })">
       <input x-ref="input" x-model="value" class="inline-edit-input" :class="{editing: editing}"
              readonly="readonly" :readonly="!editing"
              @click="enter" @blur="editing && save()"
              @click.outside="editing && save()" @keydown="onKeydown">
     </span>
  ── */
  Alpine.data('inlineEdit', function(cfg) {
    cfg = cfg || {};
    return {
      value: '',
      original: '',
      editing: false,
      error: false,
      init: function() {
        this.value = cfg.value || '';
        this.original = this.value;
      },
      get display() {
        return this.value || (cfg.emptyText || '');
      },
      enter: function() {
        if (this.editing) return;
        this.editing = true;
        var self = this;
        this.$nextTick(function() {
          if (self.$refs.input) { self.$refs.input.focus(); self.$refs.input.select(); }
        });
      },
      save: function() {
        if (!this.editing) return;
        var newVal = (this.$refs.input ? this.$refs.input.value : this.value).trim();
        if (newVal === this.original) { this.editing = false; return; }
        if (cfg.validate && !cfg.validate(newVal)) {
          this.error = true;
          var self = this;
          setTimeout(function() { self.error = false; }, 600);
          return;
        }
        this.editing = false;
        var self = this;
        // 编辑期间 x-model 持续把输入写进 value，回滚必须用 original（编辑前快照），不能用 save 时刻的 value
        Promise.resolve(cfg.onSave ? cfg.onSave(newVal) : null)
          .then(function() { self.value = newVal; self.original = newVal; })
          .catch(function() { self.value = self.original; });
      },
      cancel: function() {
        if (this.$refs.input) this.$refs.input.value = this.original;
        this.value = this.original;
        this.editing = false;
      },
      onKeydown: function(e) {
        if (e.key === 'Enter') { e.preventDefault(); this.save(); }
        else if (e.key === 'Escape') this.cancel();
      }
    };
  });

  /* ── Info Tip (hover / focus / touch toggle) ──
     Usage:
     <span class="info-tip" x-data="infoTip" tabindex="0"
           @pointerenter="enter($event)" @pointerleave="leave($event)"
           @pointerdown="down($event)"
           @focusin="focusIn()" @focusout="focusOut()" @click.prevent="tap()">
       <span class="info-tip-mark">ⓘ</span>
       <span class="info-tip-pop" x-show="open" x-cloak x-text="description"></span>
     </span>
     Mouse/pen hover and keyboard focus show the tip; a touch tap toggles it.
     The tap's synthetic focus/mouseenter are suppressed so the first tap
     leaves the tip open. Description text lives at the usage site (x-text,
     never x-html); this component only owns the open/hide state.
   ── */
  Alpine.data('infoTip', function() {
    return {
      open: false,
      touch: false,
      enter: function(e) { if (e.pointerType !== 'touch') this.open = true; },
      leave: function(e) { if (e.pointerType !== 'touch') this.open = false; },
      down: function(e) { this.touch = e.pointerType === 'touch'; },
      focusIn: function() { if (!this.touch) this.open = true; },
      focusOut: function() { if (!this.touch) this.open = false; },
      tap: function() {
        if (this.touch) this.open = !this.open;
        this.touch = false;
      }
    };
  });

  /* ── Sort Controls (sessionStorage persisted) ──
     Usage:
     <div x-data="sortControls({ prefix: 'tagfile_tree', defaultField: 'name', defaultOrder: 'asc', onChange: (f,o) => {} })">
       {{template "selectMenuField" …}}（value: () => field, onChange: v => field = v, @change="change()"）
       <button :class="{desc: order==='desc'}" @click="toggle">...</button>
     </div>
  ── */
  Alpine.data('sortControls', function(cfg) {
    cfg = cfg || {};
    return {
      field: 'name',
      order: 'asc',
      init: function() {
        var prefix = cfg.prefix || 'tagfile';
        var st = readSortState(prefix, { field: cfg.defaultField || 'name', order: cfg.defaultOrder || 'asc' });
        this.field = st.field;
        this.order = st.order;
      },
      get orderLabel() {
        return this.order === 'desc' ? '降序 ▾' : '升序 ▴';
      },
      change: function() {
        var prefix = cfg.prefix || 'tagfile';
        sessionStorage.setItem(prefix + '_sort_field', this.field);
        sessionStorage.setItem(prefix + '_sort_order', this.order);
        if (cfg.onChange) cfg.onChange(this.field, this.order);
      },
      toggle: function() {
        this.order = this.order === 'asc' ? 'desc' : 'asc';
        this.change();
      }
    };
  });

  /* ── Config Group (collapsible settings block frame) ──
     Usage:
     <div class="card config-group" x-data="configGroup({ id: g.id })">
       <div class="config-group-head" @click="toggle()">
         <span class="config-group-arrow" x-text="collapsed ? '▸' : '▾'"></span>
         ...
       </div>
       <div class="config-group-body" x-show="!collapsed"> ... </div>
     </div>
     The frame owns only its own collapsed state (default expanded, persisted in
     the shared cfg.key map). Field rendering and group save/reset stay at the
     usage site, so the block can be reused by any page that supplies them.
  ── */
  Alpine.data('configGroup', function(cfg) {
    cfg = cfg || {};
    var storageKey = cfg.key || 'tagfile_config_collapsed';
    return {
      collapsed: false,
      readMap: function() {
        return readStoredJSON(localStorage, storageKey, {});
      },
      init: function() {
        this.collapsed = !!this.readMap()[cfg.id];
      },
      toggle: function() {
        this.collapsed = !this.collapsed;
        var map = this.readMap();
        map[cfg.id] = this.collapsed;
        try { localStorage.setItem(storageKey, JSON.stringify(map)); } catch (e) {}
      },
      // 高度过渡期间裁剪组体（clipping → overflow:hidden），结束即撤——
      // 常驻裁剪会把组内下拉列表剪断（见 config.css 的注释）。
      // 只认组体自身上的 grid-template-rows 事件：字段 input 的
      // border-color 过渡会冒泡到这里，必须过滤。
      clipBody: false,
      clipBodyRun: function(e, el) { if (e.target === el) this.clipBody = true; },
      clipBodyEnd: function(e, el) {
        if (e.target === el && e.propertyName === 'grid-template-rows') this.clipBody = false;
      }
    };
  });

  /* ── Select Menu (app-styled listbox; native <select> 的展开态是系统菜单，CSS 够不着) ──
     用法见 templates/partials/select_menu.html（selectMenuField partial，模板宿主单源）：
     <div class="select-menu input-w-sm" x-data="selectMenu({ options: () => categories,
            value: () => addCategoryID, onChange: v => addCategoryID = v, placeholder: '选择分类' })"
          @change="addParent = ''"> …trigger + .dropdown 列表骨架… </div>
     value 用活 getter：init 里挂 Alpine.effect 把外部改动镜像进来（重置/联动），
     onChange 把用户选择写回宿主；$dispatch('change') 供 @change 副作用（touch/nextFile）。
     键盘与 ARIA 走 combobox+listbox 的 activedescendant 模式：焦点始终在 trigger，
     ↑/↓/Home/End 移动 highlight，Enter/Space 选中，Esc/外点/Tab 关闭，单字符 type-ahead。
     id 不能用 Alpine 的 $id：它每次绑定求值都换后缀，aria-controls/activedescendant
     会指向不存在的节点；这里在工厂闭包里发一次序号，trigger 与选项共用同一套稳定 id。
     search 构建器的 JS 字符串版（pages/search/main.js selectMenuHtml）是跨系统第二源，
     改骨架必须同步。 */
  var selectMenuSeq = 0;
  Alpine.data('selectMenu', function(cfg) {
    cfg = cfg || {};
    var uid = ++selectMenuSeq;
    return {
      value: '',
      open: false,
      highlight: -1,
      init: function() {
        this.value = this.readValue();
        if (typeof cfg.value === 'function') {
          var self = this;
          Alpine.effect(function() { self.value = self.readValue(); });
        }
      },
      readValue: function() {
        var v = typeof cfg.value === 'function' ? cfg.value() : cfg.value;
        return v == null ? '' : String(v);
      },
      get options() {
        var o = typeof cfg.options === 'function' ? cfg.options() : cfg.options;
        return o || [];
      },
      get disabled() {
        return !!(typeof cfg.disabled === 'function' ? cfg.disabled() : cfg.disabled);
      },
      get selected() {
        var opts = this.options;
        for (var i = 0; i < opts.length; i++) {
          if (String(opts[i].value) === this.value) return opts[i];
        }
        return null;
      },
      get label() {
        var s = this.selected;
        return s ? s.label : (cfg.placeholder || '');
      },
      optionId: function(i) { return 'sm-opt-' + uid + '-' + i; },
      listId: 'sm-list-' + uid,
      openMenu: function() {
        if (this.disabled || this.open) return;
        this.open = true;
        var opts = this.options;
        this.highlight = 0;
        for (var i = 0; i < opts.length; i++) {
          if (String(opts[i].value) === this.value) { this.highlight = i; break; }
        }
      },
      close: function() {
        this.open = false;
        this.highlight = -1;
      },
      toggle: function() {
        if (this.open) this.close(); else this.openMenu();
      },
      pick: function(i) {
        var o = this.options[i];
        if (!o) return;
        this.setValue(String(o.value));
        this.close();
        if (this.$refs.trigger) this.$refs.trigger.focus();
      },
      setValue: function(v) {
        if (v === this.value) return;
        this.value = v;
        if (cfg.onChange) cfg.onChange(v);
        this.$dispatch('change', { value: v });
      },
      move: function(delta) {
        var n = this.options.length;
        if (!n) return;
        this.highlight = (this.highlight + delta + n) % n;
      },
      typeAhead: function(ch) {
        var opts = this.options;
        var start = this.open ? this.highlight + 1 : 0;
        for (var k = 0; k < opts.length; k++) {
          var i = (start + k) % opts.length;
          if ((opts[i].label || '').charAt(0).toLowerCase() === ch) {
            if (!this.open) this.openMenu();
            this.highlight = i;
            return;
          }
        }
      },
      onKeydown: function(e) {
        if (this.disabled) return;
        var k = e.key;
        if (k === 'ArrowDown' || k === 'ArrowUp') {
          e.preventDefault();
          if (!this.open) this.openMenu(); else this.move(k === 'ArrowDown' ? 1 : -1);
        } else if (k === 'Home' || k === 'End') {
          if (!this.open) return;
          e.preventDefault();
          this.highlight = k === 'Home' ? 0 : this.options.length - 1;
        } else if (k === 'Enter' || k === ' ') {
          e.preventDefault();
          if (this.open) this.pick(this.highlight); else this.openMenu();
        } else if (k === 'Escape' || k === 'Tab') {
          this.close();
        } else if (k.length === 1 && !e.metaKey && !e.ctrlKey && !e.altKey) {
          this.typeAhead(k.toLowerCase());
        }
      }
    };
  });

});
