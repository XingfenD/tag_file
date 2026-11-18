/* ═══════════════════════════════════════════
   TagFile — API helper（所有页面共用）
   加载契约：本文件在 base.html 中先于 stores.js 加载；onError 运行时才引用
   stores.js 定义的 showToast（此时两者都已就位）。页面脚本只在 alpine:init
   的回调里用到 API，所以本文件的 var API 一定先于任何调用存在。
   ═══════════════════════════════════════════ */
var API = {
    /* ── 动词：POST/PUT 走 x-www-form-urlencoded，DELETE 无 body、参数在 URL ── */
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
    },

    /* ── 分页抓取：逐页取完返回扁平数组 ── */
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

    /* ── 全量标签（扁平）的标准映射 ── */
    async loadAllTags() {
        var items = await API.getAllPages('/api/tags?view=flat');
        var all = [];
        for (var i = 0; i < items.length; i++) {
            var item = items[i];
            all.push({name: item.name, path: item.path, cat: '', cid: item.cid, id: item.id});
        }
        return all;
    },

    /* 拉全量标签的优雅降级版：失败回空数组——页面只是没有补全候选，不该中断加载。 */
    async loadAllTagsSafe() {
        try {
            return await API.loadAllTags();
        } catch (e) {
            return [];
        }
    },

    async fileDetail(path, opts) {
        opts = opts || {};
        var qs = [];
        if (opts.groupId !== undefined && opts.groupId !== null) qs.push('group_id=' + opts.groupId);
        if (opts.page) qs.push('page=' + opts.page);
        if (opts.sort) qs.push('sort=' + opts.sort);
        if (opts.order) qs.push('order=' + opts.order);
        var url = '/api/files/detail/' + encodePath(path);
        return API.get(url + (qs.length ? '?' + qs.join('&') : ''));
    },

    /* ── AI 入队失败分支单源 ──
       API.post 对非 2xx 不抛（400 也照样 r.json()），成败只能看 ok：
       失败统一播报并回 null；成功侧语义（queued===0→info、乐观更新、刷新）留在各页。 */
    async enqueueAi(params) {
        var d = await API.post('/api/ai/tasks', params);
        if (!d || !d.ok) { showToast((d && d.msg) || '发起识别失败', 'error'); return null; }
        return d;
    },

    /* ── 共享错误出口 ── */
    onError: function(err) {
        console.error('API error:', err);
        if (typeof showToast === 'function') showToast('请求失败', 'error');
    }
};
