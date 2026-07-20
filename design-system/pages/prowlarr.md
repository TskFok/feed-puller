# Prowlarr 搜索页 — 页面覆盖

继承 `design-system/MASTER.md`。以下为 Prowlarr 专属规则。

## 布局

- 搜索表单：`.panel` 单列网格
- 结果：**响应式卡片网格** `.prowlarr-results-grid`
  - 小屏 1 列 · ≥640px 2 列 · ≥1024px 3 列
- 搜索中骨架：`.prowlarr-release-card--skeleton` × 6

## 结果卡片 `.prowlarr-release-card`

- Chrome 双层边框：`border` + `::before` 光泽顶边
- Hover：轻微上浮 + 青色光晕增强（`transform: translateY(-2px)`）
- 选中态：`.prowlarr-release-card--selected` 粉色光晕
- 元数据行：`.prowlarr-release-meta` 使用 tabular-nums
- **性能（≤30 条）**：`.prowlarr-results-grid--scrollable` + `useOffscreenGlassGrid`（视口外 `.glass-surface--offscreen`）
- **性能（>30 条）**：`ProwlarrVirtualResultsGrid`（窗口虚拟化 + `prowlarrRowHeightCache` 行高 session 缓存）
- 布局常量：`prowlarrLayoutConstants.ts`（`--prowlarr-row-gap`、`--prowlarr-card-intrinsic-height`）
- 阈值：`glassConstants.ts` 中 `GLASS_OFFSCREEN_MIN_ITEMS`（12）、`PROWLARR_VIRTUALIZE_THRESHOLD`（30）

## 搜索历史弹窗

- 标题区域右上使用“搜索历史”按钮；窄屏换行时仍右对齐，点击时才加载历史记录。
- 使用标准弹窗展示历史；无记录时显示空状态。
- 弹窗内沿用全局 `.history-chip`，hover 粉色半透明背景。

## 空状态 / 未配置

- 未配置 Prowlarr：`.panel` + 主 CTA `.primary` 跳转设置
- 无结果：居中 muted 文案，不显示空表格
