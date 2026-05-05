// fileTree.js - File tree recursive render (placeholder)
// tagTree.js - Tag tree recursive render (placeholder)
// flashMsg.js - Flash message display (adapted from shared_js.html)
function flashMsg(el, msg, duration) {
    var div = document.createElement('div');
    div.className = 'flash flash-success';
    div.textContent = msg;
    el.appendChild(div);
    setTimeout(function() { div.remove(); }, duration || 3000);
}
