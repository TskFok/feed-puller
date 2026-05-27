# AI 配置页 — 页面覆盖

继承 `design-system/MASTER.md`。

## 布局

- 页头：`.view-header` + 工具栏 `.subscriptions-toolbar`
- 列表：`.table-wrap` 表格（模型名称 + 操作列）
- 分页：`.pagination-bar`

## 弹窗

- 新建/编辑：`AnimatedModal` + `.subscription-edit-form` 布局
- 字段：模型名称、API 地址、API Key
- 初始焦点：第一个 input（`initialFocusRef`）

## 操作

- **检查连通**：`.icon-text` + `ShieldCheck`，测试中 `.icon-spinning`
- **编辑**：打开 `AnimatedModal`
- **删除**：`.danger` 按钮，需 toast 确认反馈

## 空状态

- `.empty` / `EmptyRow`：「暂无 AI 配置」
