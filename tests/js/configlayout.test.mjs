import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import path from 'node:path';
import vm from 'node:vm';
import { fileURLToPath } from 'node:url';

/* configPage 是页面脚本里的 Alpine.data 工厂：沙箱 stub document/Alpine
   捕获工厂后实例化，钉住条件字段分槽布局与 required 透传契约。 */
const PAGE = path.resolve(path.dirname(fileURLToPath(import.meta.url)),
  '../../cmd/webui/static/js/pages/config/main.js');

function mount() {
  const factories = {};
  const sandbox = {
    console,
    document: { addEventListener: (evt, fn) => { if (evt === 'alpine:init') fn(); } },
    Alpine: { data: (name, factory) => { factories[name] = factory; } },
  };
  vm.createContext(sandbox);
  vm.runInContext(readFileSync(PAGE, 'utf8'), sandbox, { filename: 'config/main.js' });
  return factories.configPage();
}

/* 镜像 ai 块的声明形态：provider 枚举 + 三组条件字段（地址→模型→密钥，
   ollama 无密钥位）+ 两个无条件字段。 */
function aiBoxes() {
  const box = (key, extra = {}) => ({
    key, label: key, description: '', value_type: 'String', container: 'Single',
    default: '', unit: '', show_if: null, rules: [], positional_rules: [],
    container_rules: [], value: '', has_value: false, invalid: false,
    invalid_reason: '', required: false, ...extra,
  });
  const cond = (key, equals, extra = {}) =>
    box(key, { show_if: { key: 'ai_provider', equals }, ...extra });
  return [
    box('ai_provider', { value: 'ollama', required: true }),
    cond('ollama_url', 'ollama', { required: true }),
    cond('ollama_model', 'ollama'),
    cond('ai_openai_url', 'openai', { required: true }),
    cond('ai_openai_model', 'openai'),
    cond('ai_openai_api_key', 'openai'),
    cond('ai_anthropic_url', 'anthropic', { required: true }),
    cond('ai_anthropic_model', 'anthropic'),
    cond('ai_anthropic_api_key', 'anthropic'),
    box('ai_frame_interval', { required: true }),
    box('ai_max_frames', { required: true }),
  ];
}

function buildGroup(m, provider) {
  const boxes = aiBoxes().map((x) => m.buildField(x));
  boxes[0].value = provider;
  return { id: 'ai', input_boxes: boxes, layout: m.buildLayout(boxes) };
}

test('buildLayout slots conditional fields by per-provider ordinal', () => {
  const m = mount();
  const g = buildGroup(m, 'ollama');
  // provider + 3 个槽 + 2 个无条件字段
  assert.equal(g.layout.length, 6);
  assert.equal(g.layout[0].fields.map((f) => f.key).join('|'), 'ai_provider');
  assert.equal(g.layout[0].conditional, false);
  assert.equal(g.layout[1].fields.map((f) => f.key).join('|'),
    'ollama_url|ai_openai_url|ai_anthropic_url');
  assert.equal(g.layout[2].fields.map((f) => f.key).join('|'),
    'ollama_model|ai_openai_model|ai_anthropic_model');
  // ollama 缺密钥位：第三槽只有两个候选
  assert.equal(g.layout[3].fields.map((f) => f.key).join('|'),
    'ai_openai_api_key|ai_anthropic_api_key');
  assert.equal(g.layout[3].conditional, true);
  assert.equal(g.layout[4].fields.map((f) => f.key).join('|'), 'ai_frame_interval');
  assert.equal(g.layout[5].fields.map((f) => f.key).join('|'), 'ai_max_frames');
});

test('switching provider swaps fields in place; empty slot reports invisible', () => {
  const m = mount();
  const g = buildGroup(m, 'ollama');
  const keysOf = () => g.layout.map((e) => e.fields.map((f) => f.key).join('|')).join('\n');
  const before = keysOf();

  assert.equal(m.slotVisible(g.layout[1], g), true);
  assert.equal(m.slotVisible(g.layout[3], g), false); // ollama 无密钥位 → 整槽收起

  g.input_boxes[0].value = 'openai';
  assert.equal(keysOf(), before); // 槽结构不随切换变化
  assert.equal(m.slotVisible(g.layout[3], g), true);
  assert.equal(m.fieldVisible(g.layout[1].fields[0], g), false); // ollama_url 让位
  assert.equal(m.fieldVisible(g.layout[1].fields[1], g), true);  // openai_url 原位顶上

  g.input_boxes[0].value = 'anthropic';
  assert.equal(keysOf(), before);
  assert.equal(m.fieldVisible(g.layout[2].fields[2], g), true);
});

test('buildField passes the backend required flag through', () => {
  const m = mount();
  const g = buildGroup(m, 'ollama');
  const byKey = Object.fromEntries(g.input_boxes.map((f) => [f.key, f]));
  assert.equal(byKey.ollama_url.required, true);
  assert.equal(byKey.ollama_model.required, false);
  assert.equal(byKey.ai_provider.required, true);
  assert.equal(byKey.ai_frame_interval.required, true);
});

test('unconditional-only blocks keep one entry per field', () => {
  const m = mount();
  const boxes = [
    { key: 'a', value_type: 'String', container: 'Single', rules: [], value: '' },
    { key: 'b', value_type: 'Int', container: 'Single', rules: [], value: '1' },
  ].map((x) => m.buildField(x));
  const layout = m.buildLayout(boxes);
  assert.equal(layout.length, 2);
  assert.ok(layout.every((e) => !e.conditional && e.fields.length === 1));
});