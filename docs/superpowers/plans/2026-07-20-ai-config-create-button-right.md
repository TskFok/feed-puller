# AI 配置新增按钮移至右上角 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 AI 配置页的“新增配置”主按钮移至标题栏右上角，并在窄屏标题换行时继续右对齐。

**Architecture:** AI 配置视图本地组合既有 `.view-header`：标题和说明在左侧容器，新增按钮作为右侧同级元素。桌面端复用标题栏的 flex 双端对齐，窄屏复用已有 grid 换行并对按钮使用专用右对齐类；不修改配置弹窗、Provider 预设、接口或分页逻辑。

**Tech Stack:** React 18、TypeScript、Vitest、Testing Library、CSS。

## Global Constraints

- 保持“新增配置”的名称、主按钮样式、加号图标、点击逻辑和既有弹窗行为不变。
- 宽屏显示在 AI 配置标题栏右上，窄屏换行后仍右对齐。
- 移除 AI 配置页独有的工具栏 DOM；不影响其他页面。
- 不新增接口、依赖或后端改动。
- 所有提交信息必须使用简体中文。

---

## 文件结构

- `web/src/App.tsx`：在 `AIConfigView` 中将按钮移入本地标题栏，并移除独立工具栏。
- `web/src/App.test.tsx`：验证新增配置按钮属于标题栏且仍能打开原有弹窗。
- `web/src/styles.css`：在窄屏标题栏规则中为 AI 配置按钮添加右对齐。
- `design-system/pages/ai-config.md`：将布局说明从工具栏修正为标题栏右上操作。

### Task 1: 将新增配置操作迁入标题栏

**Files:**
- Modify: `web/src/App.test.tsx:2263-2316`
- Modify: `web/src/App.tsx:2435-2455`
- Modify: `web/src/styles.css:1529-1539`
- Modify: `design-system/pages/ai-config.md:5-9`

**Interfaces:**
- Consumes: `setModal({ mode: 'create' })`；按钮的可访问名称继续为“新增配置”。
- Produces: `.ai-config-create-trigger`，作为 AI 配置标题栏右侧按钮的稳定布局标记。

- [ ] **Step 1: 在“登录后可通过 Provider 预设快速填写 AI 配置”测试中写入失败断言。**

```tsx
const trigger = screen.getByRole('button', { name: '新增配置' });
expect(trigger).toHaveClass('primary', 'ai-config-create-trigger');
expect(trigger.parentElement).toHaveClass('view-header');
fireEvent.click(trigger);
```

- [ ] **Step 2: 运行组件测试，确认现有实现因按钮仍在独立工具栏中而失败。**

运行：`npm test -- --run App`

预期：失败，断言提示按钮缺少 `ai-config-create-trigger` 或父元素不含 `view-header`。

- [ ] **Step 3: 在 `AIConfigView` 中将通用 `Header` 调用替换为局部标题栏，并移除独立工具栏。**

```tsx
<header className="view-header">
  <div>
    <h1>AI 配置</h1>
    <p>管理 OpenAI 兼容的模型接入；列表中可检查 API 是否通畅。</p>
  </div>
  <button
    type="button"
    className="primary ai-config-create-trigger"
    onClick={() => setModal({ mode: 'create' })}
  >
    <Plus size={18} aria-hidden="true" />
    新增配置
  </button>
</header>
```

删除原先包裹同一按钮的 `<div className="subscriptions-toolbar">…</div>`。保留通用 `Header` 组件和 `.subscriptions-toolbar` 选择器，因为其他页面仍在使用。

- [ ] **Step 4: 在既有窄屏标题栏规则中补充 AI 配置按钮右对齐。**

```css
.ai-config-create-trigger {
  justify-self: end;
}
```

将该规则放在已有 `.subscription-create-trigger` 规则旁的同一 `@media (max-width: 900px)` 块中；宽屏继续由 `.view-header` 的 `justify-content: space-between` 控制。

- [ ] **Step 5: 更新 AI 配置页设计系统说明。**

```markdown
- 页头：`.view-header`，右上使用 `.ai-config-create-trigger` 主按钮
```

替换现有“`.view-header` + 工具栏 `.subscriptions-toolbar`”布局描述；列表、分页、弹窗和其余说明不变。

- [ ] **Step 6: 运行组件测试，确认布局标记与新增配置弹窗行为均通过。**

运行：`npm test -- --run App`

预期：退出码为 0；新增按钮属于 AI 配置标题栏，点击后仍可打开配置弹窗并使用 Provider 预设。

- [ ] **Step 7: 运行完整前端验证与差异检查。**

运行：`npm test && npm run build && git diff --check`

预期：三条命令均以退出码 0 完成。

- [ ] **Step 8: 提交布局调整。**

```bash
git add web/src/App.tsx web/src/App.test.tsx web/src/styles.css design-system/pages/ai-config.md docs/superpowers/plans/2026-07-20-ai-config-create-button-right.md
git commit -m '将新增配置按钮移至右上角'
```
