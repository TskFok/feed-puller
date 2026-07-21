# 订阅调度列宽收紧 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将订阅列表“拉取调度”列的宽度收紧至 `9rem`。

**Architecture:** 为订阅表的“拉取调度”表头添加局部列类，并用该类与现有单元格类共同定义 `9rem` 宽度。保留表格固定布局和容器横向滚动，避免影响其他列表。

**Tech Stack:** React、TypeScript、Vitest、CSS、Vite。

## Global Constraints

- 调度列宽度为 `9rem`。
- 保持“上次 / 下次”两行摘要、其他列与 `.table-wrap` 横向滚动不变。
- 仅影响订阅列表，不修改通用表格规则。

---

### Task 1: 订阅调度列紧凑宽度

**Files:**
- Modify: `web/src/App.tsx:1743`
- Modify: `web/src/App.test.tsx:1043-1092`
- Modify: `web/src/styles.css:1267-1269`

**Interfaces:**
- Consumes: 订阅表的“拉取调度”表头与 `.sub-schedule-cell` 单元格类。
- Produces: `.sub-schedule-col` 表头类，和与单元格共享的 `9rem` 列宽规则。

- [x] **Step 1: 写入失败的调度列契约测试**

在 `web/src/App.test.tsx` 的订阅行内操作测试中追加：

```ts
expect(screen.getByRole('columnheader', { name: '拉取调度' })).toHaveClass('sub-schedule-col');
```

- [x] **Step 2: 运行测试，确认断言失败**

运行：

```bash
npm test -- --run App
```

预期：失败信息指出“拉取调度”表头缺少 `sub-schedule-col`。

- [x] **Step 3: 实现局部 9rem 列宽**

将表头改为：

```tsx
<th className="sub-schedule-col">拉取调度</th>
```

将样式改为：

```css
.sub-schedule-col,
.sub-schedule-cell {
  width: 9rem;
  min-width: 9rem;
}
```

- [x] **Step 4: 运行针对性测试，确认通过**

运行：

```bash
npm test -- --run App
```

预期：`App` 测试全部通过。

- [x] **Step 5: 运行完整前端验证**

运行：

```bash
npm test
npm run build
```

预期：完整测试与生产构建通过。

- [x] **Step 6: 提交实现**

```bash
git add web/src/App.tsx web/src/App.test.tsx web/src/styles.css docs/superpowers/plans/2026-07-21-subscription-schedule-column-width.md
git commit -m '收紧订阅调度列宽'
```
