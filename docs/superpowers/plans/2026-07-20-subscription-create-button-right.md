# 订阅新增按钮移至右上角 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将订阅页的“新增订阅”主按钮移至标题栏右上角，并在窄屏标题换行时继续右对齐。

**Architecture:** 订阅视图在本地使用既有 `.view-header` 结构：标题与说明放在左侧容器，新增按钮作为右侧同级元素。桌面端由通用标题栏 flex 双端对齐，窄屏沿用现有 grid 换行规则并对按钮施加 `justify-self: end`；不改变弹窗状态、表单或接口。

**Tech Stack:** React 18、TypeScript、Vitest、Testing Library、CSS。

## Global Constraints

- 保持“新增订阅”的名称、主按钮样式、加号图标、点击逻辑和现有弹窗行为不变。
- 宽屏显示在订阅标题栏右上，窄屏换行后仍右对齐。
- 移除订阅页独有的工具栏 DOM；不影响仍使用 `.subscriptions-toolbar` 的 AI 配置页。
- 不新增接口、依赖或后端改动。
- 所有提交信息必须使用简体中文。

---

## 文件结构

- `web/src/App.tsx`：在 `SubscriptionsView` 中将按钮放入订阅页标题栏，并移除其独立工具栏。
- `web/src/App.test.tsx`：覆盖按钮位于标题栏且点击仍打开新增订阅弹窗的回归行为。
- `web/src/styles.css`：为窄屏订阅标题按钮添加右对齐规则。
- `design-system/pages/subscriptions.md`：将订阅页的布局说明从工具栏修正为标题栏右上操作。

### Task 1: 将新增订阅操作迁入标题栏

**Files:**
- Modify: `web/src/App.test.tsx:656-684`
- Modify: `web/src/App.tsx:1670-1723`
- Modify: `web/src/styles.css:1529-1535`
- Modify: `design-system/pages/subscriptions.md:5-8`

**Interfaces:**
- Consumes: `setSubscriptionModal({ mode: 'create' })`；按钮的可访问名称继续为“新增订阅”。
- Produces: `.subscription-create-trigger`，作为订阅标题栏右侧按钮的稳定布局标记。

- [ ] **Step 1: 在现有“登录后可通过新增订阅弹窗填写关键字过滤”测试中写入失败断言。**

```tsx
const trigger = screen.getByRole('button', { name: '新增订阅' });
expect(trigger).toHaveClass('primary', 'subscription-create-trigger');
expect(trigger.parentElement).toHaveClass('view-header');
fireEvent.click(trigger);
```

- [ ] **Step 2: 运行组件测试，确认现有实现因按钮仍在独立工具栏中而失败。**

运行：`npm test -- --run App`

预期：失败，断言提示按钮缺少 `subscription-create-trigger` 或父元素不含 `view-header`。

- [ ] **Step 3: 在 `SubscriptionsView` 中将通用 `Header` 调用替换为局部标题栏，并移除独立工具栏。**

```tsx
<header className="view-header">
  <div>
    <h1>订阅</h1>
    <p>拖动左侧手柄可调整顺序；点「拉取」预览条目，点「编辑」配置地址与过滤规则，点「复制」基于已有订阅新建。</p>
  </div>
  <button
    type="button"
    className="primary subscription-create-trigger"
    onClick={() => setSubscriptionModal({ mode: 'create' })}
  >
    <Plus size={18} aria-hidden="true" />
    新增订阅
  </button>
</header>
```

删除原先包裹同一按钮的 `<div className="subscriptions-toolbar">…</div>`。保留 `Header` 组件及 `.subscriptions-toolbar` CSS，因为其他页面仍在使用。

- [ ] **Step 4: 在既有窄屏标题栏规则中补充订阅按钮右对齐。**

```css
.subscription-create-trigger {
  justify-self: end;
}
```

将该规则与已有的 `.prowlarr-history-trigger` 窄屏规则放在同一 `@media (max-width: 900px)` 块内，确保宽屏继续由 `.view-header` 的 `justify-content: space-between` 控制。

- [ ] **Step 5: 更新订阅页设计系统说明。**

```markdown
- 页头：`.view-header`，右上使用 `.subscription-create-trigger` 主按钮
```

替换现有“`.view-header` + 工具栏 `.subscriptions-toolbar`”布局描述；列表、弹窗和其余设计说明不变。

- [ ] **Step 6: 运行组件测试，确认按钮位置标记与新增弹窗行为均通过。**

运行：`npm test -- --run App`

预期：退出码为 0；新增按钮属于订阅标题栏，点击后仍显示“新增订阅”弹窗与关键字过滤字段。

- [ ] **Step 7: 运行完整前端验证与差异检查。**

运行：`npm test && npm run build && git diff --check`

预期：三条命令均以退出码 0 完成。

- [ ] **Step 8: 提交布局调整。**

```bash
git add web/src/App.tsx web/src/App.test.tsx web/src/styles.css design-system/pages/subscriptions.md docs/superpowers/plans/2026-07-20-subscription-create-button-right.md
git commit -m '将新增订阅按钮移至右上角'
```
