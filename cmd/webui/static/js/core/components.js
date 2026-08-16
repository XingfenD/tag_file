/* ═══════════════════════════════════════════
   TagFile — Alpine.js Shared Components
   tagSearch / inlineEdit / sortControls
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
     Usage:
     <span x-data="inlineEdit({ value: t.name, onSave: (v) => rename(v) })">
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
        var old = this.value;
        Promise.resolve(cfg.onSave ? cfg.onSave(newVal) : null)
          .then(function() { self.value = newVal; self.original = newVal; })
          .catch(function() { self.value = old; });
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

  /* ── Sort Controls (sessionStorage persisted) ──
     Usage:
     <div x-data="sortControls({ prefix: 'tagfile_tree', defaultField: 'name', defaultOrder: 'asc', onChange: (f,o) => {} })">
       <select x-model="field" @change="change">
         <option value="name">名称</option>...
       </select>
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
        this.field = sessionStorage.getItem(prefix + '_sort_field') || cfg.defaultField || 'name';
        this.order = sessionStorage.getItem(prefix + '_sort_order') || cfg.defaultOrder || 'asc';
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

});
