import { readFileSync } from 'node:fs';
import path from 'node:path';
import vm from 'node:vm';
import { fileURLToPath } from 'node:url';

/* 按序把 cmd/webui/static/js/core/ 下的指定脚本跑进一个 vm 沙箱并返回沙箱。
   顶层 function/var 声明自动成为沙箱属性；缺省 stub 只提供 core 脚本
   运行时才引用的浏览器全局（页面脚本不进沙箱，API/showToast 由各测试注入）。 */
const CORE = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../../cmd/webui/static/js/core');

export function loadCore(files, stubs = {}) {
  const sandbox = {
    showToast: () => {},
    showConfirm: (msg, onOk) => { if (onOk) onOk(); },
    console,
    URLSearchParams,
    setTimeout,
    ...stubs,
  };
  vm.createContext(sandbox);
  for (const f of files) {
    vm.runInContext(readFileSync(path.join(CORE, f), 'utf8'), sandbox, { filename: f });
  }
  return sandbox;
}
