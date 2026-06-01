# Feed Puller — Glassmorphism 设计系统

全局设计规范（Source of Truth）。页面级覆盖见 `design-system/pages/`。

## 风格

- **名称**：Glassmorphism（暗色默认 / 浅色变体）
- **字体**：Inter（标题与正文，`--font-display` / `--font-body`）
- **关键词**：磨砂玻璃、半透明、背景模糊、分层景深、渐变背景

## 色彩令牌

| 令牌 | 用途 |
|------|------|
| `--glass-primary` | 主色（青绿 #0D9488）、链接、焦点 |
| `--glass-primary-light` | 主色浅变体 |
| `--glass-accent` | 强调色（橙 #F97316） |
| `--glass-success` | 成功 / 进行中 |
| `--glass-text` | 主文字 |
| `--glass-text-muted` | 次要说明 |
| `--glass-panel` | 卡片 / 面板半透明底 |
| `--glass-panel-solid` | 无模糊时的实心面板（离屏优化、降级） |
| `--glass-border` / `--glass-border-soft` | 玻璃描边 |
| `--glass-shadow` / `--glass-shadow-inset` | 外阴影与顶部高光 |
| `--glass-btn` | 主按钮渐变 |

### 浅色模式对比度（WebAIM 抽查）

在 `--glass-panel: rgba(255,255,255,0.68)` 叠于 `#f0fdfa` 背景上：

| 组合 | 近似对比度 | WCAG AA |
|------|------------|---------|
| 正文 `#134e4a` × 面板 | ≈ 9.3:1 | 通过 |
| 次要 `#4a6964` × 面板 | ≈ 5.9:1 | 通过（≥4.5:1） |

调整面板透明度时，请重新抽查次要文字对比度。

## 玻璃效果

| 令牌 | 值 |
|------|-----|
| `--glass-blur` | 16px |
| `--glass-saturate` | 180% |
| `--glass-radius` | 16px |
| `--glass-radius-lg` | 20px |

实现要点：

```css
backdrop-filter: blur(var(--glass-blur)) saturate(var(--glass-saturate));
-webkit-backdrop-filter: blur(var(--glass-blur)) saturate(var(--glass-saturate));
```

配合半透明 `--glass-panel`、1px 边框、内高光阴影。

### 无 backdrop-filter 降级

`styles.css` 使用 `@supports not ((backdrop-filter: blur(1px)) or (-webkit-backdrop-filter: blur(1px)))`：

- 浅色：`--glass-panel` 提升至约 88% 不透明白
- 暗色：面板改为近实心 `#141c2e` 系
- 相关输入/侧栏令牌同步提高不透明度

### 长列表性能（离屏模糊关闭）

当 Prowlarr 结果 **>12** 条时，对 `.prowlarr-release-card` 使用 `IntersectionObserver`：

- 视口外：添加 `.glass-surface--offscreen` → `backdrop-filter: none`，背景 `--glass-panel-solid`
- 视口内：保持完整玻璃模糊
- 辅助：`content-visibility: auto` + `contain-intrinsic-size`

## 动效令牌

| 令牌 | 默认 |
|------|------|
| `--motion-fast` | 180ms |
| `--motion-enter` | 220ms |
| `--motion-ease-out` | cubic-bezier(0.22, 1, 0.36, 1) |

| 场景 | 类名 / 行为 |
|------|-------------|
| 主题切换 | `html.theme-switching`（`theme.ts` 在 reduced-motion 下跳过） |
| 页面切换 | `.view-transition` → `view-in` |
| 模态 | `.modal-overlay` / `.modal-panel` → `overlay-in` / `modal-in` |
| Toast | `.toast` → `toast-in` |

### prefers-reduced-motion

- CSS 将 `--motion-*` 置 0，禁用关键帧与 hover 位移
- `applyTheme()` 在 `prefers-reduced-motion: reduce` 时不添加 `theme-switching`

## 主题

| 偏好 | 说明 |
|------|------|
| `dark` | 固定玻璃暗色 |
| `light` | 固定玻璃浅色 |
| `system` | 跟随系统 |

存储键：`feed-puller-theme`（localStorage）。

## 反模式（避免）

- 霓虹 `text-shadow` 作为主层次手段
- 不透明实心面板替代玻璃层（除降级/离屏优化外）
- 低对比度灰字（正文需 ≥4.5:1）
- 用 emoji 作图标
- 长列表每项始终开启 `backdrop-filter`（>12 条应启用离屏优化）

## CI 对比度门禁

```bash
npm run check:contrast
```

- `scripts/check-glass-contrast.mjs`：校验 `styles.css` 浅色令牌与 WCAG 公式
- `web/src/glassContrast.test.ts`：单元测试同一套令牌

修改浅色 `--glass-panel` / `--glass-text-muted` 时须同步 `glassContrast.ts` 与脚本中的 `EXPECTED`。

## 新页面检查清单

1. 读 `MASTER.md`，再查 `design-system/pages/[page].md` 是否有覆盖
2. 面板使用 `--glass-panel` + 标准 blur 模式
3. 浅色正文/次要字对比度 ≥4.5:1（跑 `npm run check:contrast`）
4. 模态用 `AnimatedModal`；尊重 reduced-motion
5. 长列表网格：`useOffscreenGlassGrid`；>30 条使用 `ProwlarrVirtualResultsGrid` + `prowlarrRowHeightCache`
6. 长表格：`useOffscreenGlassSurface` on `.table-wrap`（≥12 行）
7. E2E 可测：`html.glass-no-backdrop-test` 模拟无 blur 降级（见 `e2e/glass-a11y.spec.ts`）
