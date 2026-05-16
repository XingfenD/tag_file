/* ── API helper – all pages use this ── */
var API = {
    async get(url) {
        var r = await fetch(url);
        if (!r.ok) throw new Error('HTTP ' + r.status);
        return r.json();
    },
    async post(url, data) {
        var r = await fetch(url, {method:'POST', headers:{'Content-Type':'application/x-www-form-urlencoded'}, body: new URLSearchParams(data)});
        return r.json();
    },
    async put(url, data) {
        var r = await fetch(url, {method:'PUT', headers:{'Content-Type':'application/x-www-form-urlencoded'}, body: new URLSearchParams(data)});
        return r.json();
    },
    async del(url) {
        var r = await fetch(url, {method:'DELETE'});
        return r.json();
    }
};

/* ── Toast fallback – app.js may overlay its own later ── */
function showToast(msg, type) {
    if (typeof window.showToast === 'function') { window.showToast(msg, type); return; }
    var t = document.createElement('div');
    t.className = 'toast toast-' + (type || 'info');
    t.textContent = msg;
    t.style.cssText = 'position:fixed;bottom:20px;right:20px;z-index:9999;padding:10px 20px;border-radius:8px;color:#fff;font-size:14px;animation:fadeIn 0.3s';
    if (type === 'error') t.style.background = '#e74c3c';
    else if (type === 'success') t.style.background = '#27ae60';
    else t.style.background = '#3498db';
    document.body.appendChild(t);
    setTimeout(function(){ t.remove(); }, 3000);
}
