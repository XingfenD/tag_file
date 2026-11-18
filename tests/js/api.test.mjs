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

test('enqueueAi: ok envelope passes through untouched (queued/skipped intact)', async () => {
  const env = { ok: true, msg: '已加入队列 1 个', queued: 1, skipped: [{ path: 'y.mp4', reason: '已识别' }] };
  const { s, toasts } = withFetch(200, env);
  const d = await s.API.enqueueAi(new URLSearchParams([['paths', 'x.mp4'], ['force', 'true']]));
  assert.deepEqual(d, env);
  assert.deepEqual(toasts, []);
});
