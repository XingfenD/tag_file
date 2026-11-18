import test from 'node:test';
import assert from 'node:assert/strict';
import { loadCore } from './helpers.mjs';

test('decodePath returns the raw value for malformed URIs', () => {
  const s = loadCore(['utils.js']);
  assert.equal(s.decodePath('%E0%A4%A'), '%E0%A4%A');
  assert.equal(s.decodePath('a%20b'), 'a b');
});

test('pathFromLocation strips the route prefix and decodes', () => {
  const s = loadCore(['utils.js'], {
    window: { location: { pathname: '/files/groups/%E4%B8%AD%20a.mp4' } },
  });
  assert.equal(s.pathFromLocation('/files/groups/'), '中 a.mp4');
});

test('fileDetailHref builds the encoded detail URL', () => {
  const s = loadCore(['utils.js']);
  assert.equal(s.fileDetailHref('sub/中 a.mp4'), '/files/detail/sub/%E4%B8%AD%20a.mp4');
});

test('readStoredJSON falls back on missing key, bad JSON and null', () => {
  const s = loadCore(['utils.js']);
  const mk = (v) => ({ getItem: () => v });
  assert.deepEqual(Array.from(s.readStoredJSON(mk(null), 'k', ['fb'])), ['fb']);
  assert.deepEqual(Array.from(s.readStoredJSON(mk(''), 'k', ['fb'])), ['fb']);
  assert.deepEqual(Array.from(s.readStoredJSON(mk('not json'), 'k', ['fb'])), ['fb']);
  assert.deepEqual(Array.from(s.readStoredJSON(mk('null'), 'k', ['fb'])), ['fb']);
  assert.equal(s.readStoredJSON(mk('{"a":1}'), 'k', {}).a, 1);
});

test('readSortState mirrors the sortControls key schema', () => {
  const store = {};
  const s = loadCore(['utils.js'], {
    sessionStorage: { getItem: (k) => (k in store ? store[k] : null) },
  });
  let st = s.readSortState('tagfile_tree', { field: 'name', order: 'asc' });
  assert.deepEqual({ ...st }, { field: 'name', order: 'asc' });
  store['tagfile_tree_sort_field'] = 'size';
  store['tagfile_tree_sort_order'] = 'desc';
  st = s.readSortState('tagfile_tree', { field: 'name', order: 'asc' });
  assert.deepEqual({ ...st }, { field: 'size', order: 'desc' });
});
