# 列表长文本截断与悬停查看 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让全部数据列表中的长业务文本以单行省略展示，并在悬停时显示完整值，避免文本撑坏列表布局。

**Architecture:** 新增无状态 `TruncatedText` 展示组件，集中提供文本、原生 `title` 和截断 CSS 类。列表页只负责将已有的最终显示文本传入组件；全局表格通过固定布局稳定列宽，Prowlarr 卡片复用同一组件而不改变其选择、状态或操作逻辑。

**Tech Stack:** React 18、TypeScript、Vitest、Testing Library、Vite、CSS。

## Global Constraints

- 仅处理名称、标题、路径、通知正文、错误、AI 提示词和 AI 返回内容等可变长业务文本。
- 保持状态、时间、数值、进度、操作按钮和订阅调度摘要的现有展示方式。
- 使用浏览器原生 `title` 提示，不引入新的浮层依赖或自定义 Tooltip。
- 所有列表文本显示为单行省略；窄屏仍保留当前表格横向滚动兜底。
- 默认在当前分支修改；提交信息使用简体中文。

---

## 文件结构

- 新建 `web/src/TruncatedText.tsx`：复用的文本截断展示组件。
- 新建 `web/src/TruncatedText.test.tsx`：验证完整文本、原生提示与截断类。
- 新建 `web/src/ProwlarrReleaseCard.test.tsx`：验证卡片标题通过该组件保留完整提示。
- 修改 `web/src/App.tsx`：将所有表格列表中的可变长业务文本替换为 `TruncatedText`。
- 修改 `web/src/ProwlarrReleaseCard.tsx`：将搜索结果卡片标题替换为 `TruncatedText`。
- 修改 `web/src/styles.css`：添加统一截断规则，固定表格布局，并移除不再适用的 `.break` 换行规则。

### Task 1: `TruncatedText` 组件

**Files:**
- Create: `web/src/TruncatedText.tsx`
- Create: `web/src/TruncatedText.test.tsx`

**Interfaces:**
- Produces: `TruncatedText({ children, className? }: { children: string; className?: string }): JSX.Element`。
- Consumes: 无运行时依赖。

- [x] **Step 1: 写入失败的组件测试**

```tsx
import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { TruncatedText } from './TruncatedText';

describe('TruncatedText', () => {
  it('保留完整文本作为原生提示并应用单行省略类', () => {
    const value = '一个非常长的列表文本，用于确认不会换行且可在悬停时查看全文';
    render(<TruncatedText>{value}</TruncatedText>);

    const text = screen.getByText(value);
    expect(text).toHaveAttribute('title', value);
    expect(text).toHaveClass('truncated-text');
  });

  it('合并调用方传入的样式类', () => {
    render(<TruncatedText className="muted">完整路径</TruncatedText>);

    expect(screen.getByText('完整路径')).toHaveClass('truncated-text', 'muted');
  });
});
```

- [x] **Step 2: 运行测试，确认因模块不存在而失败**

Run: `npm test -- --run TruncatedText`

Expected: FAIL，报错无法解析 `./TruncatedText` 模块。

- [x] **Step 3: 添加最小实现**

```tsx
type TruncatedTextProps = {
  children: string;
  className?: string;
};

export function TruncatedText({ children, className }: TruncatedTextProps) {
  const classes = ['truncated-text', className].filter(Boolean).join(' ');
  return (
    <span className={classes} title={children}>
      {children}
    </span>
  );
}
```

- [x] **Step 4: 运行组件测试，确认通过**

Run: `npm test -- --run TruncatedText`

Expected: PASS，2 个测试通过。

- [x] **Step 5: 提交组件与测试**

```bash
git add web/src/TruncatedText.tsx web/src/TruncatedText.test.tsx
git commit -m "新增列表文本省略组件"
```

### Task 2: 表格列表接入统一截断

**Files:**
- Modify: `web/src/App.tsx:1-20`（导入 `TruncatedText`）
- Modify: `web/src/App.tsx:1266,1404-1405,1554-1557,1790,2308-2310,2386-2390,2481`（替换长文本单元格）
- Modify: `web/src/App.test.tsx`（订阅列表回归测试）
- Modify: `web/src/styles.css:748-777`（表格固定布局与截断样式）

**Interfaces:**
- Consumes: Task 1 的 `TruncatedText`。
- Produces: 所有表格列表的业务文本均具有单行省略及完整 `title` 提示。

- [x] **Step 1: 写入失败的列表截断回归测试**

在 `web/src/App.test.tsx` 的现有订阅列表测试组中加入以下测试，并沿用该测试组既有的订阅 API mock：

```tsx
it('订阅名称以单行省略展示并保留完整悬停提示', async () => {
  const longName = '这个订阅名称很长，用于验证列表不会因为文本过长而换行并破坏布局';
  // 现有 mock 的 items[0].name 设置为 longName，随后渲染 <App />。
  expect(await screen.findByTitle(longName)).toHaveClass('truncated-text');
  expect(screen.getByTitle(longName)).toHaveTextContent(longName);
});
```

- [x] **Step 2: 运行测试，确认断言失败**

Run: `npm test -- --run "订阅名称以单行省略展示并保留完整悬停提示"`

Expected: FAIL，找不到 `title` 为 `longName` 的元素。

- [x] **Step 3: 将业务文本改为统一组件，并添加 CSS 约束**

在 `App.tsx` 中保留当前空值回退表达式，并把得到的文本传给 `TruncatedText`。例如：

```tsx
<td><TruncatedText>{row.subscription_name}</TruncatedText></td>
<td><TruncatedText>{row.title || row.url || '（无标题）'}</TruncatedText></td>
<td className="muted"><TruncatedText>{row.final_path?.trim() || '—'}</TruncatedText></td>
```

按相同方式替换拉取预览的标题；下载中和下载完成的订阅、标题、目录、文件路径；订阅名称；通知历史的标题、内容、错误；重命名记录的原始文件名、结果路径、提示词、返回值、错误；AI 配置名称。

在 `styles.css` 的表格规则之后添加：

```css
.table-wrap table {
  table-layout: fixed;
}

.truncated-text {
  display: block;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
```

删除未再使用的 `.break` 规则，并从相应 `td` 移除 `break` 类；保留 `muted` 等现有语义类。

- [x] **Step 4: 运行回归测试和类型检查**

Run: `npm test -- --run "订阅名称以单行省略展示并保留完整悬停提示"`

Expected: PASS，新增订阅列表断言通过。

Run: `npm run test:types`

Expected: PASS，TypeScript 无错误。

- [x] **Step 5: 提交表格接入**

```bash
git add web/src/App.tsx web/src/App.test.tsx web/src/styles.css
git commit -m "统一列表长文本省略展示"
```

### Task 3: Prowlarr 搜索结果卡片接入

**Files:**
- Create: `web/src/ProwlarrReleaseCard.test.tsx`
- Modify: `web/src/ProwlarrReleaseCard.tsx:1,44`
- Modify: `web/src/styles.css:2410-2420`（移除标题换行规则）

**Interfaces:**
- Consumes: Task 1 的 `TruncatedText` 与既有 `ProwlarrReleaseCardProps`。
- Produces: Prowlarr 结果卡片标题单行省略，并保留完整 `title` 提示。

- [x] **Step 1: 写入失败的卡片标题测试**

```tsx
import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { ProwlarrReleaseCard } from './ProwlarrReleaseCard';

describe('ProwlarrReleaseCard', () => {
  it('将长标题作为可截断文本并保留完整悬停提示', () => {
    const title = '一个非常长的 Prowlarr 搜索结果标题，用于验证卡片标题不会换行';
    render(
      <ProwlarrReleaseCard
        release={{ guid: 'guid-1', title, indexer: 'Tracker', indexerId: 1, size: 1024, seeders: 1, leechers: 0, protocol: 'torrent' }}
        selected={false}
        submitted={false}
        downloading={false}
        batchDownloading={false}
        formatBytes={() => '1 KB'}
        formatTime={() => '—'}
        onToggle={vi.fn()}
        onDownload={vi.fn()}
      />
    );

    const heading = screen.getByRole('heading', { name: title });
    const text = screen.getByTitle(title);
    expect(heading).toContainElement(text);
    expect(text).toHaveClass('truncated-text');
  });
});
```

- [x] **Step 2: 运行测试，确认断言失败**

Run: `npm test -- --run ProwlarrReleaseCard`

Expected: FAIL，找不到带完整标题 `title` 属性的元素。

- [x] **Step 3: 接入组件并移除标题换行行为**

在 `ProwlarrReleaseCard.tsx` 导入 `TruncatedText`，并将标题替换为：

```tsx
<h3 className="prowlarr-release-title">
  <TruncatedText>{release.title}</TruncatedText>
</h3>
```

从 `.prowlarr-release-title` 删除 `overflow-wrap: anywhere;`；保留现有 `flex: 1` 与 `min-width: 0`，以保证状态徽标仍能占据固定空间。

- [x] **Step 4: 运行卡片测试和相关虚拟列表回归测试**

Run: `npm test -- --run ProwlarrReleaseCard ProwlarrVirtualResultsGrid`

Expected: PASS，卡片提示测试与虚拟列表测试均通过。

- [x] **Step 5: 提交卡片接入**

```bash
git add web/src/ProwlarrReleaseCard.tsx web/src/ProwlarrReleaseCard.test.tsx web/src/styles.css
git commit -m "优化搜索结果长标题展示"
```

### Task 4: 全量验证

**Files:**
- Modify: `docs/superpowers/plans/2026-07-21-list-text-truncation.md`（勾选已完成步骤）

**Interfaces:**
- Consumes: Tasks 1-3 的已提交实现。
- Produces: 经测试和生产构建验证的完整改动。

- [x] **Step 1: 运行完整前端测试**

Run: `npm test`

Expected: PASS，Vitest 全部测试通过且 TypeScript 测试配置无错误。

- [x] **Step 2: 运行生产构建**

Run: `npm run build`

Expected: PASS，`tsc -b` 与 `vite build` 均以退出码 0 完成。

- [x] **Step 3: 检查变更范围和空白错误**

Run: `git diff --check HEAD~3..HEAD && git status --short`

Expected: 空白检查无输出；工作区只包含计划勾选或无未提交代码变更。

- [x] **Step 4: 提交执行记录**

```bash
git add docs/superpowers/plans/2026-07-21-list-text-truncation.md
git commit -m "记录列表文本省略实施"
```
