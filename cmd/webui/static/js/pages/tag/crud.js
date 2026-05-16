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
  var row = document.getElementById('tag-' + tagId);
  if (!row) return;
  var cells = row.querySelectorAll('td');
  cells[1].innerHTML = '<input id="edit_name_' + tagId + '" value="' + escJsStr(name) + '" required style="width:100px">';
  cells[3].innerHTML = '<div style="position:relative"><input id="edit_parent_' + tagId + '" placeholder="搜索..." autocomplete="off" data-input-id="edit_parent_' + tagId + '" data-dropdown-id="edit_parent_dropdown_' + tagId + '" style="width:120px"><div id="edit_parent_dropdown_' + tagId + '" class="dropdown"></div></div>';
  cells[5].innerHTML = '<input id="edit_desc_' + tagId + '" value="' + escJsStr(desc) + '" style="width:120px">';
  cells[6].innerHTML = '<button class="btn btn-primary btn-sm" onclick="saveTag(' + tagId + ')">保存</button> <button type="button" class="btn btn-default btn-sm" onclick="refreshTagList()">取消</button>';
  loadParentTags(categoryId, 'edit_parent_' + tagId, 'edit_parent_dropdown_' + tagId, parentName, tagId);
}

function saveTag(tagId) {
  var name = document.getElementById('edit_name_' + tagId).value;
  var parent = document.getElementById('edit_parent_' + tagId).value;
  var desc = document.getElementById('edit_desc_' + tagId).value;
  API.put('/api/tags/' + tagId, {
    tag_name: name,
    description: desc,
    parent_tag_name: parent
  }).then(function(resp) {
    if (resp.ok) {
      refreshTagList();
      showToast(resp.msg || '标签已更新', 'success');
    } else {
      showToast(resp.msg || '更新失败', 'error');
    }
  });
}

/* ── Parent Tag Autocomplete ── */
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
