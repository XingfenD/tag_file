import test from 'node:test';
import assert from 'node:assert/strict';
import { loadCore } from './helpers.mjs';

/* selectMenu 是 Alpine.data 工厂：沙箱里 stub document/Alpine 捕获工厂与 effect，
   再手工挂 $refs/$dispatch 两个 magic 后实例化，逐条钉键盘/ARIA/双向镜像契约。 */
function mount(cfg) {
  const factories = {};
  const effects = [];
  const events = [];
  let focusCount = 0;
  const s = loadCore(['components.js'], {
    document: { addEventListener: (evt, fn) => { if (evt === 'alpine:init') fn(); } },
    Alpine: {
      data: (name, factory) => { factories[name] = factory; },
      effect: (fn) => { effects.push(fn); fn(); },
    },
  });
  const m = factories.selectMenu(cfg);
  m.$refs = { trigger: { focus: () => { focusCount++; } } };
  m.$dispatch = (name, detail) => events.push([name, detail]);
  m.init();
  return {
    m,
    events,
    effects,
    focusCount: () => focusCount,
    key: (k, extra = {}) => m.onKeydown({ key: k, preventDefault: () => {}, ...extra }),
    rerunEffects: () => effects.forEach((fn) => fn()),
  };
}

const OPS = [
  { value: 'contains', label: '包含' },
  { value: 'not_contains', label: '不包含' },
  { value: 'equals', label: '等于' },
];

test('label follows the live value getter and falls back to placeholder', () => {
  const host = { op: 'equals' };
  const t = mount({ options: OPS, value: () => host.op, placeholder: '选操作符' });
  assert.equal(t.m.label, '等于');
  host.op = 'nope';
  t.rerunEffects();
  assert.equal(t.m.label, '选操作符');
  assert.equal(t.m.selected, null);
});

test('openMenu highlights the current value; toggle and disabled gate it', () => {
  const t = mount({ options: OPS, value: 'not_contains' });
  t.m.openMenu();
  assert.equal(t.m.open, true);
  assert.equal(t.m.highlight, 1);
  t.m.toggle();
  assert.equal(t.m.open, false);
  const d = mount({ options: OPS, value: 'contains', disabled: true });
  d.m.openMenu();
  assert.equal(d.m.open, false);
});

test('pick writes back through onChange, dispatches change, closes and refocuses', () => {
  const host = { op: 'contains' };
  const writes = [];
  const t = mount({
    options: OPS,
    value: () => host.op,
    onChange: (v) => { host.op = v; writes.push(v); },
  });
  t.m.openMenu();
  t.m.pick(2);
  assert.deepEqual(writes, ['equals']);
  assert.equal(host.op, 'equals');
  assert.deepEqual(JSON.parse(JSON.stringify(t.events)), [['change', { value: 'equals' }]]); // 跨 realm 归一
  assert.equal(t.m.open, false);
  assert.equal(t.focusCount(), 1);
  t.m.pick(2); // 同值不重复派发（与原生 select 的 change 语义一致）
  assert.equal(t.events.length, 1);
});

test('arrow keys open then wrap; Enter picks the highlighted option', () => {
  const host = { op: 'contains' };
  const t = mount({ options: OPS, value: () => host.op, onChange: (v) => { host.op = v; } });
  t.key('ArrowDown');
  assert.equal(t.m.open, true);
  assert.equal(t.m.highlight, 0);
  t.key('ArrowUp');
  assert.equal(t.m.highlight, 2); // 回绕到末项
  t.key('Enter');
  assert.equal(host.op, 'equals');
  assert.equal(t.m.open, false);
});

test('Home/End jump, Escape and Tab close, Space opens', () => {
  const t = mount({ options: OPS, value: 'contains' });
  t.key(' ');
  assert.equal(t.m.open, true);
  t.key('End');
  assert.equal(t.m.highlight, 2);
  t.key('Home');
  assert.equal(t.m.highlight, 0);
  t.key('Escape');
  assert.equal(t.m.open, false);
  t.m.openMenu();
  t.key('Tab');
  assert.equal(t.m.open, false);
});

test('type-ahead jumps to the next label starting with the char', () => {
  const t = mount({ options: OPS, value: 'contains' });
  t.key('不');
  assert.equal(t.m.open, true);
  assert.equal(t.m.highlight, 1);
  t.key('b'); // 无匹配：保持原位
  assert.equal(t.m.highlight, 1);
});

test('options getter stays live so async-loaded choices appear', () => {
  const host = { cats: [] };
  const t = mount({
    options: () => [{ value: '', label: '选择分类' }].concat(host.cats.map((c) => ({ value: String(c.id), label: c.name }))),
    value: '',
  });
  assert.equal(t.m.options.length, 1);
  host.cats = [{ id: 4, name: 'probe-cat' }];
  assert.equal(t.m.options.length, 2);
  assert.equal(t.m.options[1].label, 'probe-cat');
});

test('optionId and aria-selected track value for the activedescendant pattern', () => {
  const t = mount({ options: OPS, value: 'equals' });
  assert.equal(t.m.optionId(1), 'sm-opt-1-1');
  assert.equal(t.m.listId, 'sm-list-1');
  assert.notEqual(t.m.listId, t.m.optionId(1));
  t.m.openMenu();
  assert.equal(t.m.highlight, 2);
  assert.equal(String(t.m.options[2].value) === t.m.value, true);
});

test('ids stay stable across calls and differ per instance', () => {
  const t = mount({ options: OPS, value: 'contains' });
  assert.equal(t.m.optionId(0), t.m.optionId(0));
  assert.equal(t.m.listId, t.m.listId);
  const factories = {};
  loadCore(['components.js'], {
    document: { addEventListener: (evt, fn) => { if (evt === 'alpine:init') fn(); } },
    Alpine: { data: (name, factory) => { factories[name] = factory; }, effect: (fn) => fn() },
  });
  const a = factories.selectMenu({ options: OPS, value: 'contains' });
  const b = factories.selectMenu({ options: OPS, value: 'contains' });
  assert.notEqual(a.listId, b.listId);
  assert.notEqual(a.optionId(0), b.optionId(0));
});