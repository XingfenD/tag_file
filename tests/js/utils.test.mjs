import test from 'node:test';
import assert from 'node:assert/strict';
import { loadCore } from './helpers.mjs';

test('confirmDelete keeps params in URL query and refreshes only on ok', async () => {
  const calls = [];
  const s = loadCore(['utils.js'], {
    showConfirm: (msg, onOk) => onOk(),
    showToast: () => {},
    API: { del: async (url) => { calls.push(url); return { ok: true, msg: '已删' }; }, onError: () => {} },
  });
  let refreshed = 0;
  s.confirmDelete('确定删除？', '/api/files/group/3?file_path=a%2Fb', { onOk: () => refreshed++ });
  await new Promise(r => setTimeout(r, 0));
  assert.equal(calls[0], '/api/files/group/3?file_path=a%2Fb');
  assert.equal(refreshed, 1);
});

test('confirmDelete does not run onOk on a failure envelope', async () => {
  const toasts = [];
  const s = loadCore(['utils.js'], {
    API: { del: async () => ({ ok: false, msg: '被占用' }), onError: () => {} },
    showToast: (m, t) => toasts.push([m, t]),
  });
  let refreshed = 0;
  s.confirmDelete('删？', '/api/x', { okMsg: '删了', failMsg: '删除失败', onOk: () => refreshed++ });
  await new Promise(r => setTimeout(r, 0));
  assert.equal(refreshed, 0);
  assert.deepEqual(toasts[toasts.length - 1], ['被占用', 'error']);
});

test('confirmDelete routes a rejected del through API.onError without throwing', async () => {
  const errs = [];
  const s = loadCore(['utils.js'], {
    API: { del: async () => { throw new Error('network down'); }, onError: (e) => errs.push(e.message) },
  });
  s.confirmDelete('删？', '/api/x', { onOk: () => assert.fail('must not refresh') });
  await new Promise(r => setTimeout(r, 0));
  assert.deepEqual(errs, ['network down']);
});

test('reportResultOrThrow throws on failure so inlineEdit rolls back', () => {
  const s = loadCore(['utils.js'], { showToast: () => {} });
  assert.throws(() => s.reportResultOrThrow({ ok: false, msg: '重名' }, '', '更新失败'), /重名/);
  assert.doesNotThrow(() => s.reportResultOrThrow({ ok: true }, '已更新', ''));
});

test('confirmDelete success toast defaults to info type', async () => {
  const toasts = [];
  const s = loadCore(['utils.js'], {
    API: { del: async () => ({ ok: true, msg: '已删除' }), onError: () => {} },
    showToast: (m, t) => toasts.push([m, t]),
  });
  s.confirmDelete('删？', '/api/x', { onOk: () => {} });
  await new Promise(r => setTimeout(r, 0));
  assert.deepEqual(toasts, [['已删除', 'info']]);
});

test('reportResultOrThrow success toast prefers server msg over fallback', () => {
  const toasts = [];
  const s = loadCore(['utils.js'], { showToast: (m, t) => toasts.push([m, t]) });
  s.reportResultOrThrow({ ok: true, msg: '标签值已更新' }, '值已更新', '更新失败');
  s.reportResultOrThrow({ ok: true }, '值已更新', '更新失败');
  assert.deepEqual(toasts, [['标签值已更新', 'success'], ['值已更新', 'success']]);
});
