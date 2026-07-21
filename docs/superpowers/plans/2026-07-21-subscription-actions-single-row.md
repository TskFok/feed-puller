# 订阅操作按钮单行 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让订阅列表中的拉取、编辑、复制与删除按钮在所有视口宽度下保持单行显示。

**Architecture:** 修改订阅操作单元格的局部 CSS 覆盖通用操作区的自动换行规则。表格容器继续承担窄屏横向滚动，因此无需改变组件结构或按钮行为。

**Tech Stack:** React、TypeScript、Vitest、CSS、Vite。

## Global Constraints

- 仅调整订阅列表操作区，不影响通用 `.actions` 或其他列表。
- 四个操作按钮在所有宽度下保持单行；空间不足时使用既有 `.table-wrap` 横向滚动。
- 不改变按钮文案、图标、可访问名称、禁用状态或点击行为。

---

### Task 1: 订阅操作区单行布局

**Files:**
- Modify: `web/src/App.tsx:1793`
- Modify: `web/src/App.test.tsx:1043-1090`
- Modify: `web/src/styles.css:792-795`

**Interfaces:**
- Consumes: 订阅列表操作单元格的现有类名 `actions subscription-actions`。
- Produces: `.subscription-actions--single-row` 的局部单行样式；操作区所有四个按钮保持在同一 flex 行。

- [x] **Step 1: 写入失败的布局契约测试**

在 `web/src/App.test.tsx` 的“订阅列表的行内操作按钮使用非透明操作样式”测试末尾追加：

```ts
const actions = screen.getByRole('button', { name: /拉取/ }).closest<HTMLElement>('.subscription-actions');
expect(actions).toHaveClass('subscription-actions');
expect(actions).toHaveClass('subscription-actions--single-row');
```

- [x] **Step 2: 运行测试，确认新增断言在现有实现中失败**

运行：

```bash
npm test -- --run App
```

预期：测试失败，原因是订阅操作单元格尚未添加 `subscription-actions--single-row`。

- [x] **Step 3: 以最小 CSS 覆盖实现单行布局**

将 `web/src/App.tsx` 的操作单元格类名改为：

```tsx
<td className="actions subscription-actions subscription-actions--single-row">
```

将 `web/src/styles.css` 的 `.subscription-actions` 规则改为：

```css
.subscription-actions--single-row {
  align-items: center;
  gap: 10px;
  flex-wrap: nowrap;
  white-space: nowrap;
}
```

现有固定表格布局与 `.table-wrap { overflow-x: auto; }` 会承接横向溢出。

- [x] **Step 4: 运行针对性测试，确认通过**

运行：

```bash
npm test -- --run App
```

预期：`App` 测试通过。

- [x] **Step 5: 运行完整前端验证**

运行：

```bash
npm test
npm run build
```

预期：全部测试与生产构建通过。

- [x] **Step 6: 提交实现**

```bash
git add web/src/App.tsx web/src/App.test.tsx web/src/styles.css
git commit -m '修复订阅操作按钮换行'
```
