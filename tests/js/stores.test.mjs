import test from 'node:test';
import assert from 'node:assert/strict';
import { loadCore } from './helpers.mjs';

/* stores.js 在 alpine:init 回调里注册 store：stub 收集监听器后手动触发。 */
function loadStores() {
  const stores = {};
  const listeners = {};
  loadCore(['stores.js'], {
    document: { addEventListener: (evt, fn) => { (listeners[evt] = listeners[evt] || []).push(fn); } },
    Alpine: { store: (name, obj) => { if (obj) stores[name] = obj; return stores[name]; }, nextTick: (fn) => fn() },
    window: {},
  });
  (listeners['alpine:init'] || []).forEach((fn) => fn());
  return stores;
}

const render = (segs) => Array.from(segs).map((x) => (x.em ? '[' + x.text + ']' : x.text)).join('');

/* ── Confirm store：显式 name 通道 ── */

test('confirm segments render the entity name as the emphasized chip segment', () => {
  const c = loadStores().confirm;
  c.show('确定把 扫描设置 的全部配置项恢复为默认值？', null, null, '扫描设置');
  assert.equal(render(c.segments()), '确定把 [扫描设置] 的全部配置项恢复为默认值？');
});

test('confirm segments without a name stay a single plain segment', () => {
  const c = loadStores().confirm;
  c.show('确定删除该标签？', null, null);
  assert.equal(render(c.segments()), '确定删除该标签？');
  c.show('确定继续？', null, null, '不在文案里的名字');
  assert.equal(render(c.segments()), '确定继续？');
});

test('confirm segments at message edges produce no empty segments', () => {
  const c = loadStores().confirm;
  c.show('风景 确定删除？', null, null, '风景');
  assert.equal(render(c.segments()), '[风景] 确定删除？');
  c.show('确定删除 风景', null, null, '风景');
  assert.equal(render(c.segments()), '确定删除 [风景]');
});

test('confirm ok/cancel still fire their callbacks after the name channel', () => {
  const c = loadStores().confirm;
  let ok = 0, cancel = 0;
  c.show('删？', () => ok++, () => cancel++, '名字');
  c.ok();
  assert.equal(ok, 1);
  assert.equal(c.active, false);
  c.show('删？', () => ok++, () => cancel++, '名字');
  c.cancel();
  assert.equal(cancel, 1);
});

test('confirmDelete forwards opts.name to showConfirm', () => {
  let captured = null;
  const s = loadCore(['utils.js'], {
    showConfirm: (msg, onOk, onCancel, name) => { captured = { msg, name }; },
  });
  s.confirmDelete('确定删除标签 风景？', '/api/tags/1', { name: '风景' });
  assert.equal(captured.msg, '确定删除标签 风景？');
  assert.equal(captured.name, '风景');
  s.confirmDelete('确定删除？', '/api/tags/1', {});
  assert.equal(captured.name, undefined);
});

/* ── Toast store：«实体名» 标记解析 ── */

test('toast segments parse a backend «name» marker into a chip', () => {
  const t = loadStores().toast;
  assert.equal(render(t.segments('标签 «风景» 已添加')), '标签 [风景] 已添加');
  assert.equal(render(t.segments('文件 «a/b.mp4» 已导入')), '文件 [a/b.mp4] 已导入');
});

test('toast segments parse the frontend tag-move marker', () => {
  const t = loadStores().toast;
  assert.equal(render(t.segments('标签已移动到 «自然» 下')), '标签已移动到 [自然] 下');
});

test('toast segments handle multiple markers and edge positions', () => {
  const t = loadStores().toast;
  assert.equal(render(t.segments('«甲» 与 «乙»')), '[甲] 与 [乙]');
  assert.equal(render(t.segments('«风景» 已删除')), '[风景] 已删除');
  assert.equal(render(t.segments('已删除 «风景»')), '已删除 [风景]');
});

test('toast segments without markers stay a single plain segment', () => {
  const t = loadStores().toast;
  assert.equal(render(t.segments('已恢复默认值，尚未保存')), '已恢复默认值，尚未保存');
  assert.equal(render(t.segments('')), '');
  assert.equal(render(t.segments(null)), '');
});

test('toast segments leave an empty «» marker as literal text', () => {
  const t = loadStores().toast;
  assert.equal(render(t.segments('空 «» 标记')), '空 «» 标记');
});