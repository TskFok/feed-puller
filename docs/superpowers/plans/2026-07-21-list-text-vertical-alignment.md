# 列表文字垂直居中 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让所有表格型数据列表的表头与单元格内容垂直居中，同时不改变 Prowlarr 搜索结果卡片。

**Architecture:** 复用 `web/src/styles.css` 中的共享 `th, td` 规则，将当前顶端对齐替换为居中对齐。添加一项直接读取样式表的契约测试，防止共享规则意外回退为顶端对齐。

**Tech Stack:** CSS、Vitest、Vite。

## Global Constraints

- 仅影响共享表格规则所服务的数据列表。
- Prowlarr 搜索结果卡片网格不修改。
- 保留文本截断、列宽、按钮布局和现有行高。

---

### Task 1: 统一表格文字垂直对齐

**Files:**
- Create: `web/src/listTextVerticalAlignmentStyles.test.js`
- Modify: `web/src/styles.css:758-768`
- Modify: `docs/superpowers/plans/2026-07-21-list-text-vertical-alignment.md`

**Interfaces:**
- Consumes: 共享 `th, td` CSS 选择器。
- Produces: 表格表头和单元格的 `vertical-align: middle` 布局契约。

- [x] **Step 1: 写入失败的共享样式契约测试**

创建 `web/src/listTextVerticalAlignmentStyles.test.js`：

```js
import { readFileSync } from 'node:fs';
import { describe, expect, it } from 'vitest';

const root = process.cwd().endsWith('/web') ? process.cwd() : `${process.cwd()}/web`;
const styles = readFileSync(`${root}/src/styles.css`, 'utf8');

describe('list text vertical alignment styles', () => {
  it('让共享表格表头与单元格垂直居中', () => {
    expect(styles.includes('th,\ntd {\n  padding: 12px 14px;\n  border-bottom: 1px solid var(--glass-border-soft);\n  text-align: left;\n  vertical-align: middle;')).toBe(true);
  });
});
```

- [x] **Step 2: 运行测试，确认断言失败**

运行：

```bash
npm test -- --run listTextVerticalAlignmentStyles
```

预期：失败信息指出共享表格规则尚未包含 `vertical-align: middle`。

- [x] **Step 3: 实现共享垂直居中规则**

在 `web/src/styles.css` 的共享规则中替换：

```css
vertical-align: top;
```

为：

```css
vertical-align: middle;
```

- [x] **Step 4: 运行针对性测试，确认通过**

运行：

```bash
npm test -- --run listTextVerticalAlignmentStyles
```

预期：该测试通过。

- [x] **Step 5: 运行完整前端验证**

运行：

```bash
npm test
npm run build
```

预期：完整测试与生产构建通过。

- [x] **Step 6: 提交实现与计划**

```bash
git add web/src/listTextVerticalAlignmentStyles.test.js web/src/styles.css docs/superpowers/plans/2026-07-21-list-text-vertical-alignment.md
git commit -m '调整列表文字垂直居中'
```
