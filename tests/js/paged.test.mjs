import test from 'node:test';
import assert from 'node:assert/strict';
import { loadCore } from './helpers.mjs';

test('clampToLastPage returns last page only when out of range', () => {
  const s = loadCore(['utils.js']);
  assert.equal(s.clampToLastPage({ total_pages: 2 }, 3), 2);
  assert.equal(s.clampToLastPage({ total_pages: 2 }, 1), null);
  assert.equal(s.clampToLastPage({ total_pages: 2 }, 2), null);
  assert.equal(s.clampToLastPage({ total_pages: 0 }, 3), null);
  assert.equal(s.clampToLastPage({}, 1), null);
});

test('loadPaged re-fetches the last page when the requested one fell off', async () => {
  const urls = [];
  const s = loadCore(['utils.js'], {
    API: {
      get: async (u) => {
        urls.push(u);
        if (urls.length === 1) return { ok: true, items: [], pagination: { page: 2, total_pages: 2 } };
        return { ok: true, items: ['尾巴行'], pagination: { page: 2, total_pages: 2 } };
      },
      onError: () => {},
    },
  });
  const self = { page: 3, items: [], pagination: {}, loading: false };
  await s.loadPaged(self, { url: '/api/x' }, false);
  assert.equal(urls.length, 2);
  assert.match(urls[1], /page=2/);
  assert.deepEqual(self.items, ['尾巴行']);
  assert.equal(self.page, 2);
});

test('silent loadPaged keeps loading untouched, toasts nothing, warns on reject (Review Focus 3)', async () => {
  const toasts = [], warns = [];
  const s = loadCore(['utils.js'], {
    API: { get: async () => { throw new Error('net'); }, onError: () => { warns.push('onError-ran'); } },
    showToast: (m, t) => toasts.push([m, t]),
    console: { warn: (...a) => warns.push(a.join(' ')) },
  });
  const self = { page: 1, items: [], pagination: {}, loading: false };
  await s.loadPaged(self, { url: '/api/x', warnMsg: '[paged] 静默加载失败，下一次继续:' }, true);
  assert.equal(self.loading, false);
  assert.deepEqual(toasts, []);
  assert.deepEqual(warns, ['[paged] 静默加载失败，下一次继续: Error: net']);
});

test('failed envelope toasts d.msg and keeps items on screen', async () => {
  const toasts = [];
  const s = loadCore(['utils.js'], {
    API: { get: async () => ({ ok: false, msg: '数据库锁' }), onError: () => {} },
    showToast: (m, t) => toasts.push([m, t]),
  });
  const self = { page: 1, items: ['旧行'], pagination: {}, loading: false };
  await s.loadPaged(self, { url: '/api/x', errMsg: '加载失败' }, false);
  assert.deepEqual(toasts, [['数据库锁', 'error']]);
  assert.deepEqual(self.items, ['旧行']);
  assert.equal(self.loading, false);
});

test('silent loadPaged stays quiet on a failed envelope too', async () => {
  const toasts = [];
  const s = loadCore(['utils.js'], {
    API: { get: async () => ({ ok: false, msg: '数据库锁' }), onError: () => {} },
    showToast: (m, t) => toasts.push([m, t]),
  });
  const self = { page: 1, items: ['旧行'], pagination: {}, loading: false };
  await s.loadPaged(self, { url: '/api/x' }, true);
  assert.deepEqual(toasts, []);
  assert.equal(self.loading, false);
  assert.deepEqual(self.items, ['旧行']);
});

test('pagedList clearSearch resets query/page and goToPage keeps query', async () => {
  const urls = [];
  const s = loadCore(['utils.js'], {
    API: { get: async (u) => { urls.push(u); return { ok: true, items: [], pagination: {} }; }, onError: () => {} },
  });
  const self = s.pagedList({ url: (p) => '/api/c?page=' + p.page + (p.q ? '&q=' + encodeURIComponent(p.q) : '') });
  self.q = '旧词';
  self.queryInput = '旧词';
  self.page = 4;
  await new Promise(r => { self.clearSearch(); setTimeout(r, 0); });
  assert.equal(self.q, '');
  assert.equal(self.queryInput, '');
  assert.equal(self.page, 1);
  assert.equal(urls[0], '/api/c?page=1');
  self.q = '尾巴';
  await new Promise(r => { self.goToPage(3); setTimeout(r, 0); });
  assert.equal(self.page, 3);
  assert.equal(urls[1], '/api/c?page=3&q=%E5%B0%BE%E5%B7%B4');
});

test('pagedList.doSearch trims query, resets page and refreshes', async () => {
  const urls = [];
  const s = loadCore(['utils.js'], {
    API: { get: async (u) => { urls.push(u); return { ok: true, items: [], pagination: {} }; }, onError: () => {} },
  });
  const self = Object.assign(s.pagedList({ url: (p) => '/api/c?page=' + p.page + (p.q ? '&q=' + encodeURIComponent(p.q) : '') }), {});
  self.queryInput = '  尾巴  ';
  self.page = 5;
  await new Promise(r => { self.doSearch(); setTimeout(r, 0); });
  assert.equal(self.q, '尾巴');
  assert.equal(self.page, 1);
  assert.match(urls[0], /^\/api\/c\?page=1&/);
  assert.match(urls[0], /q=%E5%B0%BE%E5%B7%B4/);
});
