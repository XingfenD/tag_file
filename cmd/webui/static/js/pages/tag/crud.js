/* ═══════════════════════════════════════════
   TagFile — Tag Page: CRUD + Parent Autocomplete
   ═══════════════════════════════════════════ */

function submitAddTag(form) {
  var fd = new FormData(form);
  var data = {};
  fd.forEach(function(v, k) { data[k] = v; });
  API.post('/api/tags', data).then(function(resp) {
    if (resp.ok) {
      refreshTagList();
      showToast(resp.msg || '标签已添加', 'success');
      form.reset();
    } else {
      showToast(resp.msg || '添加失败', 'error');
    }
  }).catch(function() {
    refreshTagList();
  });
  return false;
}

function deleteTag(btn, event) {
  if (event) event.stopPropagation();
  var tagID = parseInt(btn.getAttribute('data-tag-id'));
  var tagName = btn.getAttribute('data-tag-name');
  showConfirm('确定删除标签 &laquo;' + escHtml(tagName) + '&raquo;？', function() {
    API.del('/api/tags/' + tagID).then(function(resp) {
      if (resp.ok) {
        refreshTagList();
        showToast(resp.msg || '标签已删除', 'info');
      } else {
        showToast(resp.msg || '删除失败', 'error');
      }
    }).catch(function() {
      refreshTagList();
    });
  });
}

function editTag(btn) {
  var tagId = parseInt(btn.getAttribute('data-tag-id'));
  var categoryId = parseInt(btn.getAttribute('data-cid'));
  var name = btn.getAttribute('data-tag-name');
  var desc = btn.getAttribute('data-desc') || '';
  var parentName = btn.getAttribute('data-parent') || '';

  var tagsForAutocomplete = [];
  if (!categoryId) {
    tagsForAutocomplete = allTags;
  } else {
    var catID = parseInt(categoryId);
    tagsForAutocomplete = allTags.filter(function(t) { return t.cid === catID && t.id !== tagId; });
  }

  Modal.show({
    title: '编辑标签',
    fields: [
      { label: '标签名', name: 'tag_name', value: name, required: true },
      { label: '父标签（可选）', name: 'parent_tag_name', value: parentName,
        placeholder: '搜索...', autocomplete: { tags: tagsForAutocomplete } },
      { label: '描述', name: 'description', value: desc }
    ],
    size: 'lg',
    onSave: async function(values) {
      var newName = values.tag_name.trim();
      if (!newName) { showToast('标签名不能为空', 'error'); return false; }
      var resp = await API.put('/api/tags/' + tagId, {
        tag_name: newName,
        description: values.description.trim(),
        parent_tag_name: values.parent_tag_name.trim()
      });
      if (!resp.ok) { showToast(resp.msg || '更新失败', 'error'); return false; }
      refreshTagList();
      showToast(resp.msg || '标签已更新', 'success');
    }
  });
}

/* ── Legacy stubs ── */
function saveTag(tagId) {
  // Removed — editing is now handled by Modal.editTag()
}

function loadParentTags(categoryID, inputId, dropdownId, selectedName, tagId) {
  var input = document.getElementById(inputId);
  var dropdown = document.getElementById(dropdownId);
  if (!input || !dropdown) return;
  var ctrl = setupTagSearch(input, dropdown);
  if (selectedName) input.value = selectedName;
  if (!categoryID) { ctrl.setTags(allTags); return; }
  var catID = parseInt(categoryID);
  ctrl.setTags(allTags.filter(function(t) { return t.cid === catID && t.id !== tagId; }));
}

function loadParentTagsForAdd(categoryID) {
  document.getElementById('parent_tag_select').value = '';
  if (!parentTagSearch) return;
  if (!categoryID) { parentTagSearch.setTags(allTags); return; }
  var catID = parseInt(categoryID);
  parentTagSearch.setTags(allTags.filter(function(t) { return t.cid === catID; }));
}
