# 订阅页 — 页面覆盖

继承 `design-system/MASTER.md`。

## 布局

- 页头：`.view-header` + 工具栏 `.subscriptions-toolbar`
- 列表：`.table-wrap` 表格，拖拽列 `.sub-drag-col`
- 编辑/新建：使用 `AnimatedModal`，表单 `.subscription-edit-form`

## 玻璃与性能

- 表格容器：`.table-wrap` 使用全局玻璃 blur + `--glass-panel`
- 行数 ≥12：`.table-wrap` 绑定 `useOffscreenGlassSurface`，离屏时 `.glass-surface--offscreen`（实心 `--glass-panel-solid`）
- 表格行：`tbody tr` 使用 `content-visibility: auto` 降低长页绘制成本
- 拉取预览弹窗：`.fetch-preview-modal` + `.fetch-preview-table-wrap` 宽屏玻璃表格

## 动效

- 页面切换：`.view-transition`（`prefers-reduced-motion` 下禁用）
- 模态：`AnimatedModal` → `.modal-overlay` / `.modal-panel` 入场动画在 reduced-motion 下为 `none`
- 主题切换：设置页 `ThemePicker`，遵循 `theme-switching` / reduced-motion 规则

## 交互

- 行拖拽排序：`.sub-drag-handle`，悬停/over 态 `.sub-row-drag-over`
- 拉取预览：`AnimatedModal` + `.fetch-preview-modal` 宽屏表格
- 状态徽章：`.status-*` 语义色

## 空状态

- `.empty` 居中 muted 文案 + 可选「新增订阅」主 CTA
