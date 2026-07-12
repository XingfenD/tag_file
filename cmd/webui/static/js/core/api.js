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
/* ── Paginated fetch: loads all pages and returns flat array of items ── */
    async getAllPages(url, maxPages) {
        var all = [];
        var limit = maxPages || 100;
        var page = 1;
        while (page <= limit) {
            var sep = url.indexOf('?') >= 0 ? '&' : '?';
            var data = await API.get(url + sep + 'page=' + page);
            if (data.ok && data.items) {
                for (var i = 0; i < data.items.length; i++) {
                    all.push(data.items[i]);
                }
                if (data.pagination && data.pagination.has_next) {
                    page++;
                } else {
                    break;
                }
            } else {
                break;
            }
        }
        return all;
    },

/* ── Load all tags (flat) with standard mapping ── */
    async loadAllTags() {
        var items = await API.getAllPages('/api/tags?view=flat');
        var all = [];
        for (var i = 0; i < items.length; i++) {
            var item = items[i];
            all.push({name: item.name, path: item.path, cat: '', cid: item.cid, id: item.id});
        }
        return all;
    },

/* ── Shared error handler ── */
    onError: function(err) {
        console.error('API error:', err);
        if (typeof showToast === 'function') showToast('请求失败', 'error');
    },

    async del(url) {
        var r = await fetch(url, {method:'DELETE'});
        return r.json();
    }
};
