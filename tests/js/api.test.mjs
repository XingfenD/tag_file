import test from 'node:test';
import assert from 'node:assert/strict';
import { loadCore } from './helpers.mjs';

function withFetch(status, envelope) {
  const toasts = [];
  const body = () => new Response(JSON.stringify(envelope), {
    status, headers: { 'Content-Type': 'application/json' },
  });
  const s = loadCore(['api.js', 'utils.js'], {
    fetch: async () => body(),
    Response,
    showToast: (m, t) => toasts.push([m, t]),
  });
  return { s, toasts };
}

test('enqueueAi: non-2xx failure envelope returns null and toasts server msg', async () => {
  const { s, toasts } = withFetch(400, { ok: false, msg: '目录不存在' });
  const d = await s.API.enqueueAi(new URLSearchParams([['paths', 'x.mp4']]));
  assert.equal(d, null);
  assert.deepEqual(toasts, [['目录不存在', 'error']]);
});

test('fileDetail assembles query from provided opts only', async () => {
  const urls = [];
  const s = loadCore(['api.js', 'utils.js'], {
    fetch: async (u) => { urls.push(u); return new Response(JSON.stringify({ ok: true, detail: {} }), { status: 200, headers: { 'Content-Type': 'application/json' } }); },
    Response,
  });
  await s.API.fileDetail('sub/中 a.mp4', { groupId: 2, page: 3, sort: 'size', order: 'desc' });
  assert.equal(urls[0], '/api/files/detail/sub/%E4%B8%AD%20a.mp4?group_id=2&page=3&sort=size&order=desc');
  await s.API.fileDetail('x.mp4', {});
  assert.equal(urls[1], '/api/files/detail/x.mp4');
  await s.API.fileDetail('y.mp4', { page: 2 });
  assert.equal(urls[2], '/api/files/detail/y.mp4?page=2');
  // 详情页永远带 group_id，0（默认组）是合法值，不能被真值判断丢掉
  await s.API.fileDetail('z.mp4', { groupId: 0 });
  assert.equal(urls[3], '/api/files/detail/z.mp4?group_id=0');
});

test('enqueueAi: ok envelope passes through untouched (queued/skipped intact)', async () => {
  const env = { ok: true, msg: '已加入队列 1 个', queued: 1, skipped: [{ path: 'y.mp4', reason: '已识别' }] };
  const { s, toasts } = withFetch(200, env);
  const d = await s.API.enqueueAi(new URLSearchParams([['paths', 'x.mp4'], ['force', 'true']]));
  assert.deepEqual(d, env);
  assert.deepEqual(toasts, []);
});
