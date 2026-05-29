# 下载列表页 — 页面覆盖

继承 `design-system/MASTER.md`。适用于「下载中」「下载完成」两个 Tab。

## 布局

- 页头：`.view-header`
- 列表：`.table-wrap` + 分页 `.pagination-bar`
- 进度：`.download-progress` / `.download-progress-fill` 粉青渐变条

## 玻璃与性能

- 列表容器：`.table-wrap` 玻璃面板（与订阅页相同令牌）
- 当前页行数 ≥12：`useOffscreenGlassSurface` 作用于 `.table-wrap`
- 离屏：`.glass-surface--offscreen` → 无 blur、`background: var(--glass-panel-solid)`
- 表格行：`content-visibility: auto`（见 `styles.css`）

## 动效

- 下载中 Tab 每 5s 刷新列表；进度条 `.download-progress-fill` 宽度过渡
- `prefers-reduced-motion`：进度条过渡禁用；页面/模态动画同 MASTER

## 数据展示

- 进度百分比：`.download-progress-percent`，tabular-nums
- 状态：`.status-submitted`、`.status-failed`、`.status-pending` 等

## 空状态

- `.empty`：无任务时的引导文案
