# 订阅页 — 页面覆盖

继承 `design-system/MASTER.md`。

## 布局

- 页头：`.view-header` + 工具栏 `.subscriptions-toolbar`
- 列表：`.table-wrap` 表格，拖拽列 `.sub-drag-col`
- 编辑/新建：使用 `AnimatedModal`，表单 `.subscription-edit-form`

## 交互

- 行拖拽排序：`.sub-drag-handle`，悬停/over 态 `.sub-row-drag-over`
- 拉取预览：`AnimatedModal` + `.fetch-preview-modal` 宽屏表格
- 状态徽章：`.status-*` 语义色

## 空状态

- `.empty` 居中 muted 文案 + 可选「新增订阅」主 CTA
