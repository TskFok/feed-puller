# Prowlarr 结果卡片标题与操作栏对齐 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 修复 Prowlarr 结果卡片中标题、复选框和下载操作在超长标题场景下的对齐与可见性。

**Architecture:** 保持 `ProwlarrReleaseCard` 的内容和数据流不变，只为标题行与操作栏增加明确的局部布局钩子。通过 CSS Grid 固定复选框和状态列、限制标题列收缩，并为操作栏建立宽度边界；以组件测试和 CSS 规则测试防止回归。

**Tech Stack:** React 18、TypeScript、CSS、Vitest、Testing Library。

## Global Constraints

- 仅修改 Prowlarr 结果卡片的局部结构和样式；不得改变搜索历史弹窗、接口或虚拟列表数据流。
- 标题保留单行省略和 `title` 悬停全文提示，不改为多行。
- 复选框与标题从标题行同一顶部开始；下载按钮必须位于卡片操作栏的可视范围内。
- 所有提交信息使用简体中文。

---

### Task 1: 固化卡片布局约束并修复样式

**Files:**
- Create: `web/src/ProwlarrReleaseCard.styles.test.ts`
- Modify: `web/src/ProwlarrReleaseCard.test.tsx:5-27`
- Modify: `web/src/ProwlarrReleaseCard.tsx:37-77`
- Modify: `web/src/styles.css:2417-2474`

**Interfaces:**
- Consumes: `ProwlarrReleaseCardProps`，其 `release.title` 为任意长度字符串。
- Produces: `.prowlarr-release-card-head`、`.prowlarr-release-title`、`.prowlarr-release-actions` 和 `.prowlarr-release-download` 这四个稳定的测试与样式钩子。

- [x] **Step 1: 为组件结构和样式规则写失败测试。**

在 `web/src/ProwlarrReleaseCard.test.tsx` 的现有断言后增加：

```tsx
expect(heading.parentElement).toHaveClass('prowlarr-release-card-head');
expect(screen.getByRole('button', { name: '下载' })).toHaveClass('prowlarr-release-download');
expect(screen.getByRole('button', { name: '下载' }).parentElement).toHaveClass('prowlarr-release-actions');
```

创建 `web/src/ProwlarrReleaseCard.styles.test.ts`：

```ts
import { readFileSync } from 'node:fs';
import { describe, expect, it } from 'vitest';

const root = process.cwd().endsWith('/web') ? process.cwd() : `${process.cwd()}/web`;
const styles = readFileSync(`${root}/src/styles.css`, 'utf8');

describe('ProwlarrReleaseCard styles', () => {
  it('为标题、复选框和状态建立可收缩的三列网格', () => {
    expect(styles).toContain('grid-template-columns: 18px minmax(0, 1fr) auto;');
    expect(styles).toContain('margin-top: 0;');
  });

  it('让操作栏和下载按钮保持在卡片宽度内', () => {
    expect(styles).toContain('.prowlarr-release-actions {\n  display: flex;\n  min-width: 0;');
    expect(styles).toContain('.prowlarr-release-download {\n  flex: 0 0 auto;');
  });
});
```

- [x] **Step 2: 运行测试并确认失败。**

Run: `npm test -- --run ProwlarrReleaseCard`

Expected: FAIL；下载按钮尚未具有 `prowlarr-release-download`，且样式中没有三列网格和操作栏约束。

- [x] **Step 3: 实施最小结构与样式修复。**

在 `web/src/ProwlarrReleaseCard.tsx` 的下载按钮添加局部类：

```tsx
className="primary-link prowlarr-release-download"
```

将 `web/src/styles.css` 中卡片标题与操作栏规则调整为：

```css
.prowlarr-release-card-head {
  display: grid;
  grid-template-columns: 18px minmax(0, 1fr) auto;
  align-items: start;
  gap: 10px;
  min-width: 0;
}

.prowlarr-release-card-head input[type='checkbox'] {
  width: 18px;
  height: 18px;
  margin-top: 0;
  accent-color: var(--glass-accent);
}

.prowlarr-release-title {
  min-width: 0;
}

.prowlarr-release-actions {
  display: flex;
  min-width: 0;
  justify-content: flex-end;
  padding-top: 4px;
  border-top: 1px solid var(--glass-border-soft);
}

.prowlarr-release-download {
  flex: 0 0 auto;
}
```

保留原有的 `flex-shrink: 0`、字体和颜色规则；为网格不再需要的 `.prowlarr-release-status { margin-left: auto; }` 删除该声明。

- [x] **Step 4: 运行定向测试并确认通过。**

Run: `npm test -- --run ProwlarrReleaseCard`

Expected: PASS；组件结构测试与 CSS 规则测试均通过。

- [x] **Step 5: 运行完整前端验证。**

Run: `npm test && npm run build`

Expected: 两个命令均以退出码 `0` 完成；无 TypeScript 或 Vitest 失败。

- [x] **Step 6: 提交实现。**

```bash
git add web/src/ProwlarrReleaseCard.tsx web/src/ProwlarrReleaseCard.test.tsx web/src/ProwlarrReleaseCard.styles.test.ts web/src/styles.css
git commit -m '修复 Prowlarr 结果卡片对齐'
```
