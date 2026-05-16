/* ═══════════════════════════════════════════
   TagFile — Tag Page: Drag and Drop Reorder
   ═══════════════════════════════════════════ */

var _drag = { active: false, srcID: 0, srcName: '', startX: 0, startY: 0, ghost: null, threshold: 5 };

document.addEventListener('mousedown', function(e) {
  var row = e.target.closest('.tag-tree-row-main');
  if (!row) return;
  if (e.target.closest('.tree-actions') || e.target.closest('button') || e.target.closest('a')) return;
  var node = row.closest('.tag-tree-node');
  if (!node) return;
  _drag.srcID = parseInt(node.getAttribute('data-tag-id'));
  _drag.srcName = node.getAttribute('data-tag-name');
  _drag.startX = e.clientX;
  _drag.startY = e.clientY;
  _drag.active = false;
  _drag.node = node;
});

document.addEventListener('mousemove', function(e) {
  if (!_drag.srcID) return;
  var dx = e.clientX - _drag.startX;
  var dy = e.clientY - _drag.startY;
  if (!_drag.active) {
    if (Math.abs(dx) < _drag.threshold && Math.abs(dy) < _drag.threshold) return;
    _drag.active = true;
    document.body.classList.add('tag-dragging');
    _drag.node.setAttribute('data-dragging', '1');
    var g = document.createElement('div');
    g.className = 'drag-ghost';
    g.textContent = '🏷 ' + _drag.srcName;
    document.body.appendChild(g);
    _drag.ghost = g;
  }
  if (_drag.ghost) {
    _drag.ghost.style.left = (e.clientX + 12) + 'px';
    _drag.ghost.style.top = (e.clientY + 12) + 'px';
  }
  document.querySelectorAll('.tag-tree-row-main.drag-over').forEach(function(n) { n.classList.remove('drag-over'); });
  var el = document.elementFromPoint(e.clientX, e.clientY);
  if (el) {
    var row = el.closest('.tag-tree-row-main');
    if (row) {
      var node = row.closest('.tag-tree-node');
      if (node && parseInt(node.getAttribute('data-tag-id')) !== _drag.srcID) {
        row.classList.add('drag-over');
      }
    }
  }
});

document.addEventListener('mouseup', function(e) {
  if (!_drag.srcID) return;
  if (_drag.active) {
    var el = document.elementFromPoint(e.clientX, e.clientY);
    if (el) {
      var row = el.closest('.tag-tree-row-main');
      if (row) {
        var node = row.closest('.tag-tree-node');
        if (node) {
          var targetID = parseInt(node.getAttribute('data-tag-id'));
          var targetName = node.getAttribute('data-tag-name');
          if (targetID !== _drag.srcID) {
            doDragMove(_drag.srcID, targetName);
          }
        }
      }
    }
    document.body.classList.remove('tag-dragging');
    if (_drag.node) _drag.node.removeAttribute('data-dragging');
    if (_drag.ghost) _drag.ghost.remove();
    document.querySelectorAll('.tag-tree-row-main.drag-over').forEach(function(n) { n.classList.remove('drag-over'); });
  }
  _drag = { active: false, srcID: 0, srcName: '', startX: 0, startY: 0, ghost: null, threshold: 5 };
});

function doDragMove(tagID, parentName) {
  var tagName = '';
  for (var i = 0; i < allTags.length; i++) {
    if (allTags[i].id === tagID) { tagName = allTags[i].name; break; }
  }
  API.put('/api/tags/' + tagID, {
    tag_name: tagName,
    parent_tag_name: parentName
  }).then(function(resp) {
    if (resp.ok) {
      refreshTagList();
      showToast(resp.msg || '标签已移动到 «' + parentName + '» 下', 'success');
    } else {
      showToast(resp.msg || '移动失败', 'error');
    }
  });
}
