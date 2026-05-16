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
