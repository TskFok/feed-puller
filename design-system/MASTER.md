# feed-puller Design System (MASTER)

Y2K Aesthetic 全局设计规范。页面级覆盖见 `design-system/pages/`。

## 风格定位

- **名称**：Y2K Aesthetic（暗色默认）/ Bubblegum Light（浅色变体）
- **关键词**：霓虹粉、.chrome 银、全息青、光泽按钮、气泡圆角、2000s 复古未来
- **字体**：标题 Orbitron · 正文 Exo 2

## 语义色令牌（CSS 变量）

组件应使用语义变量，禁止硬编码 hex。

| 令牌 | 用途 |
|------|------|
| `--y2k-pink` | 主强调、危险态边框 |
| `--y2k-cyan` | 次强调、链接、焦点环 |
| `--y2k-purple` / `--y2k-violet` | 背景渐变、装饰 |
| `--y2k-silver` | Chrome 边框、ghost 按钮 |
| `--y2k-lime` | 成功/进行中状态 |
| `--y2k-text` | 主文字 |
| `--y2k-text-muted` | 次要说明 |
| `--y2k-panel` | 卡片/面板背景 |
| `--y2k-border` / `--y2k-border-soft` | 霓虹描边 |
| `--y2k-glow-pink` / `--y2k-glow-cyan` | 发光阴影 |
| `--y2k-glossy-btn` | 主按钮渐变 |
| `--text` / `--muted` / `--surface-muted` | 兼容别名 |

## 圆角与间距

| 令牌 | 值 |
|------|-----|
| `--y2k-radius` | 14px |
| `--y2k-radius-bubble` | 22px |
| 触控最小高度 | 44px |
| 间距节奏 | 4 / 8 / 14 / 18 / 24 / 32 |

## 动效

| 令牌 | 值 |
|------|-----|
| `--motion-fast` | 180ms |
| `--motion-enter` | 220ms |
| `--motion-ease-out` | cubic-bezier(0.22, 1, 0.36, 1) |

- Tab 切换：`.view-transition` fade + translateY
- Modal：`.AnimatedModal`（`.modal-overlay` / `.modal-panel`）scale + fade + 焦点陷阱
- 主题切换：`html.theme-switching` 200ms 背景/面板 crossfade
- 尊重 `prefers-reduced-motion: reduce`

## 主题

`html[data-theme="dark"|"light"]`，持久化键 `feed-puller-theme`，取值：

| 值 | 含义 |
|----|------|
| `dark` | 固定 Y2K 暗色 |
| `light` | 固定 Bubblegum 浅色 |
| `system` | 跟随 `prefers-color-scheme`（**默认**，首次访问无存储时） |

系统偏好变化时，若当前为 `system` 则自动切换生效主题。

## 组件约定

- **主按钮**：`.primary` — 光泽粉渐变 + 胶囊形
- **次按钮**：`.ghost` — chrome 银渐变边框
- **面板**：`.panel` / `.settings-panel` / `.table-wrap` — 玻璃 + 霓虹描边
- **状态徽章**：`.status-*` — 霓虹边框 + 语义色，不仅靠颜色区分（含文字）

## 反模式（Avoid）

- Emoji 作为功能图标（使用 Lucide SVG）
- 硬编码颜色到 TSX
- 纯 hover 无 focus 态
- 动画 > 500ms 或忽略 reduced-motion

## 文件映射

| 资源 | 路径 |
|------|------|
| 全局样式 | `web/src/styles.css` |
| 主题逻辑 | `web/src/theme.ts` |
| 页面覆盖 | `design-system/pages/*.md` |

### 页面文档索引

| 页面 | 文件 |
|------|------|
| Prowlarr 搜索 | `pages/prowlarr.md` |
| 订阅 | `pages/subscriptions.md` |
| 下载中 / 完成 | `pages/downloads.md` |
| 设置 | `pages/settings.md` |
| 登录 | `pages/login.md` |
| AI 配置 | `pages/ai-config.md` |
