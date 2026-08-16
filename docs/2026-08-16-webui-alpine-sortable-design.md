# WebUI 重构设计：Alpine.js + SortableJS

日期：2026-08-16
分支：`refactor/webui-alpine-sortable`

## 目标

用 Alpine.js 3.x 和 SortableJS 全面重构 WebUI 交互层：
- 交互逻辑从 vanilla JS 命令式 DOM 操作改为 Alpine 声明式组件
- 标签树拖拽从手写 `drag-drop.js` 改为 SortableJS
- UI 外观与 CSS 完全不变，只换逻辑
- 保持无 npm、无构建步骤，`go:embed` 单二进制交付

## 现状

- Go 模板只渲染空壳（`handlers.Server` 仅传 `Active`），所有数据由页面 JS fetch `/api/*`
- 24 个 vanilla JS 文件（~3.8k 行，`var`/全局函数风格），13 个模板
- 手写交互：toast、confirm modal、标签搜索下拉（键盘导航）、内联编辑、排序控件、树拖拽
- 静态资源通过 `//go:embed static` 打包，`main.go` 提供 `/static/{path...}` 路由

## 架构

1. **依赖引入**：`cmd/webui/static/vendor/alpinejs.min.js`（3.x 固定版本）+ `sortable.min.js`（UMD 版）。从 unpkg 下载，base.html 引入，go:embed 自动打包。断网可用。
2. **共享层**（`static/js/core/`）：
   - `Alpine.store('toast')`、`Alpine.store('confirm')` 替代 `showToast`/`showConfirm`；modal 在 base.html 用 `x-show` + `x-for` 渲染，保留现有 CSS 动画
   - `Alpine.data('tagSearch')`（保留键盘导航 Up/Down/Enter/Tab/Escape）、`Alpine.data('inlineEdit')`、`Alpine.data('sortControls')`
   - `api.js` 保留全局 `API` 对象直接复用；`theme.js` 保留原样（head 中早加载防止主题闪烁）
   - 退役：`app.js`、`modal.js`、`inline-edit.js`、`sort-controls.js`
3. **页面组件**（`static/js/pages/*.js` 重写）：每页一个主组件 `Alpine.data('xxPage')`，复杂页可拆多个组件（如 tag 页的 sidebar/列表/crud），列表渲染从 `innerHTML +=` 字符串拼接改为 `<template x-for>`。
4. **SortableJS**：仅标签树视图（嵌套 Sortable，`group` 配置），`onEnd` → `PUT /api/tags/{id}` 修改 `parent_tag_name`。平铺视图与文件列表维持按钮排序，不引入拖拽。
5. **模板改造**：去内联 `onclick`/`onchange`，改 `x-data` 挂载。Alpine 表达式不使用 `{{ }}`，与 Go 模板无冲突。`{{define}}` 不能嵌套的约束在 x-for 化后部分缓解（`tagTreeNode` 递归模板逻辑迁移到 Alpine 递归组件）。

## 实施顺序（3 个提交）

1. **基建**：vendor 依赖 + base.html（Alpine 启动、toast/confirm store）+ core 共享组件（tagSearch/inlineEdit/sortControls）
2. **标签页**：tag 页全套（sidebar、树/平铺视图、crud、values、SortableJS 拖拽）
3. **其余页面**：category、config、home、file（列表/详情/groups）、search（builder）；删除退役的 vanilla 文件

## 验证

- `make build-webui` 编译通过
- `make test` Go 侧不受影响
- 起服务逐页 curl 200，浏览器 DevTools 实测：toast、confirm、标签搜索键盘导航、内联编辑、树拖拽、搜索构建器
- 更新 `docs/CHANGELOG_webui.md`（webui 变更）

## 风险与对策

- Alpine 与 Go 模板语法冲突：Alpine 用 `x-bind`/`@` 指令，不解析 `{{ }}`，无冲突
- 键盘无障碍：tagSearch 键盘导航、Escape 关闭、`ctrl/cmd+k` 全局搜索焦点必须在 Alpine 组件中完整保留
- 表单 `_return` 隐藏字段与 POST 重定向流程：全部改 fetch 后不再依赖重定向，但需保持 URL 查询状态（分类筛选、搜索词）在刷新后不丢
- 无 JS 测试框架：以浏览器实测为准，Go 侧测试确保 API 契约不变
