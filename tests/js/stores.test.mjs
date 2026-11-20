import test from 'node:test';
import assert from 'node:assert/strict';
import { loadCore } from './helpers.mjs';

/* stores.js 在 alpine:init 回调里注册 store：stub 收集监听器后手动触发。
   utils.js 先跑：TOAST_ICONS 用它的 iconHtml()。 */
function loadStores(extra = {}) {
  const stores = {};
  const listeners = {};
  loadCore(['utils.js', 'stores.js'], {
    document: { addEventListener: (evt, fn) => { (listeners[evt] = listeners[evt] || []).push(fn); } },
    Alpine: { store: (name, obj) => { if (obj) stores[name] = obj; return stores[name]; }, nextTick: (fn) => fn() },
    window: {},
    ...extra,
  });
  (listeners['alpine:init'] || []).forEach((fn) => fn());
  return stores;
}

/* 假计时器：按调度顺序排队，fireLast 触发最近一个（两段式退场的 stages 可逐步观察），
   fireAll 排空（含级联调度的移除计时器）。 */
function fakeTimers() {
  const queue = [];
  let seq = 1;
  return {
    stubs: {
      setTimeout: (fn, ms) => { const id = seq++; queue.push({ id, fn, ms }); return id; },
      clearTimeout: (id) => { const i = queue.findIndex((t) => t.id === id); if (i !== -1) queue.splice(i, 1); },
    },
    get pending() { return queue.length; },
    fireLast() { const t = queue.pop(); if (t) t.fn(); },
    fireAll() { while (queue.length) queue.shift().fn(); },
  };
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

/* ── Toast store：生命周期（id / 时长 / 暂停续计 / 两段式移除 / 堆叠上限） ── */

test('toast show assigns unique ids, per-type durations and defaults to info', () => {
  const ft = fakeTimers();
  const t = loadStores(ft.stubs).toast;
  t.show('成功', 'success');
  t.show('出错', 'error');
  t.show('提示');
  assert.equal(t.items.length, 3);
  const ids = t.items.map((x) => x.id);
  assert.equal(new Set(ids).size, 3);
  assert.equal(t.items[0].duration, 3000);
  assert.equal(t.items[1].duration, 5000);
  assert.equal(t.items[2].type, 'info');
  assert.equal(t.items[2].duration, 3000);
  assert.equal(ft.pending, 3);
});

test('toast auto-dismiss is two-stage: leaving flag first, removal after the exit animation', () => {
  const ft = fakeTimers();
  const t = loadStores(ft.stubs).toast;
  t.show('一条', 'success');
  ft.fireLast(); // 倒计时到点 → dismiss
  assert.equal(t.items.length, 1);
  assert.equal(t.items[0].leaving, true);
  assert.equal(ft.pending, 1); // 退场动画后的移除计时器
  ft.fireLast();
  assert.equal(t.items.length, 0);
});

test('toast pause freezes the remaining time and resume re-arms the countdown', () => {
  let now = 100000;
  const ft = fakeTimers();
  const t = loadStores({ ...ft.stubs, Date: { now: () => now } }).toast;
  t.show('悬停我', 'error');
  const id = t.items[0].id;
  now += 2000;
  t.pause(id);
  assert.equal(ft.pending, 0);
  assert.equal(t.items[0].remaining, 3000); // 5000 - 2000
  now += 60000; // 悬停期间时间任意流逝，不侵蚀剩余寿命
  t.resume(id);
  assert.equal(ft.pending, 1);
  assert.equal(t.items[0].remaining, 3000);
  t.pause(id);
  assert.equal(t.items[0].remaining, 3000);
  t.resume(id);
  ft.fireLast();
  assert.equal(t.items[0].leaving, true);
  t.resume(id); // leaving 后不再续计
  assert.equal(ft.pending, 1); // 只有移除计时器
});

test('toast dismiss clears the countdown, is idempotent and ignores unknown ids', () => {
  const ft = fakeTimers();
  const t = loadStores(ft.stubs).toast;
  t.show('手动关闭', 'info');
  const id = t.items[0].id;
  t.dismiss(id);
  assert.equal(t.items[0].leaving, true);
  assert.equal(ft.pending, 1); // 倒计时已清，只剩移除
  t.dismiss(id);
  assert.equal(ft.pending, 1);
  t.pause(99999); t.resume(99999); t.dismiss(99999);
  ft.fireLast();
  assert.equal(t.items.length, 0);
});

test('toast stack caps at five: the oldest leaves when a sixth arrives', () => {
  const ft = fakeTimers();
  const t = loadStores(ft.stubs).toast;
  for (let i = 1; i <= 6; i++) t.show('t' + i, 'info');
  assert.equal(t.items.length, 6);
  assert.equal(t.items[0].leaving, true);
  assert.ok(t.items.slice(1).every((x) => !x.leaving));
  ft.fireLast(); // 最后调度的是上限触发的移除计时器
  assert.equal(t.items.length, 5);
  assert.equal(t.items[0].msg, 't2');
});

test('toast icon falls back to info for unknown types', () => {
  const t = loadStores().toast;
  assert.match(t.icon('success'), /<svg/);
  assert.match(t.icon('error'), /<svg/);
  assert.notEqual(t.icon('success'), t.icon('error'));
  assert.equal(t.icon('不存在的类型'), t.icon('info'));
});