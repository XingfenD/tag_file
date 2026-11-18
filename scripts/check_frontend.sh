#!/usr/bin/env bash
# 前端安全网（Wave 0）：JS 语法门 + JS↔模板双向引用一致性。
# 语法：node --check 逐文件（vendor 除外——压缩产物不归我们维护）。
# 引用：模板里写的每个 /static/js/*.js 必须存在；static/js 下（vendor 除外）
#       每个文件必须恰好被一个模板加载——0 个是死代码，>1 个说明加载契约漂了。
set -u
root="$(cd "$(dirname "$0")/.." && pwd)"
webui="$root/cmd/webui"
fail=0

echo '── 1/3 JS syntax (node --check) ──'
while IFS= read -r -d '' f; do
  if ! node --check "$f" 2>&1; then
    echo "SYNTAX FAIL: ${f#"$root"/}"
    fail=1
  fi
done < <(find "$webui/static/js" -name '*.js' ! -path '*/vendor/*' -print0)

echo '── 2/3 template refs resolve to a real file ──'
while IFS= read -r ref; do
  p="$webui/${ref#/}"
  if [ ! -f "$p" ]; then
    echo "MISSING: $ref (referenced by templates, no such file)"
    fail=1
  fi
done < <(grep -rho '/static/js/[^"]*\.js' "$webui/templates" | sort -u)

echo '── 3/3 every JS file is loaded by exactly one template ──'
while IFS= read -r -d '' f; do
  rel="${f#"$webui"/}"
  n=$(grep -rl "/$rel" "$webui/templates" | wc -l)
  if [ "$n" -ne 1 ]; then
    echo "REF COUNT $n (want 1): $rel"
    fail=1
  fi
done < <(find "$webui/static/js" -name '*.js' ! -path '*/vendor/*' -print0)

if [ "$fail" -ne 0 ]; then
  echo 'fe-check FAILED'
  exit 1
fi
echo 'fe-check OK'
