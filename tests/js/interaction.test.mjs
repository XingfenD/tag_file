import test from 'node:test';
import assert from 'node:assert/strict';
import { loadCore } from './helpers.mjs';

function fakeRoot() {
  const listeners = {};
  return {
    addEventListener: (evt, fn) => { listeners[evt] = fn; },
    fire: (evt, e) => listeners[evt](e),
  };
}
const elWith = (action) => ({ getAttribute: (k) => (k === 'data-action' ? action : null) });

test('delegate dispatches by data-action and ignores misses', () => {
  const s = loadCore(['utils.js']);
  const root = fakeRoot();
  const hits = [];
  s.delegate(root, 'click', {
    retry: (el, e) => hits.push(['retry', el, e]),
  });
  const e1 = { target: { closest: () => elWith('retry') }, preventDefault: () => {} };
  root.fire('click', e1);
  assert.equal(hits.length, 1);
  assert.equal(hits[0][1].getAttribute('data-action'), 'retry');
  assert.equal(hits[0][2], e1);
  root.fire('click', { target: { closest: () => null } });
  root.fire('click', { target: { closest: () => elWith('nope') }, preventDefault: () => {} });
  assert.equal(hits.length, 1);
});

test('createSelection single-sources toggle/toggleAll/prune/allSelected', () => {
  const s = loadCore(['utils.js']);
  const bag = Object.assign({ suggestions: [] }, s.createSelection('selected'));
  const sel = () => Array.from(bag.selected); // 沙箱内生成的数组跨 realm，归一后再比结构
  bag.toggle(1, true);
  bag.toggle(2, true);
  assert.deepEqual(sel(), [1, 2]);
  bag.toggle(1, false);
  assert.deepEqual(sel(), [2]);
  bag.toggleAll([1, 2, 3], true);
  assert.deepEqual(sel(), [1, 2, 3]);
  assert.equal(bag.allSelected([1, 2, 3]), true);
  assert.equal(bag.allSelected([1, 2]), false);
  assert.equal(bag.allSelected([]), false);
  bag.prune([2, 3]);
  assert.deepEqual(sel(), [2, 3]);
  bag.toggleAll([2, 3], false);
  assert.deepEqual(sel(), []);
});
