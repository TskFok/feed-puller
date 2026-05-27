# 登录页 — 页面覆盖

继承 `design-system/MASTER.md`。

## 布局

- 全屏居中：`.login-screen` + `.login-panel` 气泡卡片
- 双 Tab：`.login-tabs`（密码 / 飞书，按 `/api/auth/options` 动态显示）

## 文案

- 迁移提示：`.login-migration-hint`（两种登录均可用时显示）

## 外观

- 页底 `.login-theme-picker`：`ThemePicker` compact 变体（图标按钮 + `aria-label`）

## 飞书扫码

- QR 容器：`.feishu-qr-inline` 白底以保证扫码对比度
