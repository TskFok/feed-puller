# 分页切换回顶 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 用户点击任一列表的上一页或下一页时，应用滚动容器自动回到顶部。

**Architecture:** 保持各个分页 Hook 与 API 调用不变。在共用的 `PaginationBar` 中处理页码按钮事件：先通知既有 `onPageChange`，再复用 `resolveAppScrollElement` 解析实际滚动容器并将其 `scrollTop` 归零。该集中式实现自动覆盖全部现有列表，且可兼容窄屏布局。

**Tech Stack:** React 18、TypeScript、Vitest、Testing Library、jsdom。

## Global Constraints

- 仅覆盖“上一页”和“下一页”按钮，不改变每页条数选择器、分页请求或筛选行为。
- 使用既有 `resolveAppScrollElement` 识别桌面 `.workspace` 或窄屏文档根节点；不在组件树外引入全局状态。
- 不增加滚动动画。
- 测试必须先失败，再添加生产代码。

---

## File Structure

- 修改 `web/src/ListPagination.tsx`：集中处理页码变更与滚动容器回顶。
- 修改 `web/src/ListPagination.test.tsx`：验证点击下一页同时通知页码和重置 `.workspace.scrollTop`。

### Task 1: 分页栏回顶行为

**Files:**
- Modify: `web/src/ListPagination.tsx`
- Test: `web/src/ListPagination.test.tsx`

**Interfaces:**
- Consumes: `PaginationBarProps.onPageChange: (page: number) => void`。
- Produces: `PaginationBar` 的上一页和下一页按钮在调用 `onPageChange` 后，将 `resolveAppScrollElement` 返回的实际滚动容器回顶。

- [x] **Step 1: 写入失败测试**

在 `web/src/ListPagination.test.tsx` 中新增测试并把组件放入可滚动容器。保留对回调参数的断言，新增滚动位置断言：

```tsx
it('切换页码时回到 workspace 顶部', () => {
  const onPageChange = vi.fn();
  const { container } = render(
    <main className="workspace" style={{ overflowY: 'auto' }}>
      <PaginationBar
        page={1}
        pageSize={30}
        totalPages={2}
        totalItems={35}
        rangeStart={1}
        rangeEnd={30}
        onPageChange={onPageChange}
        onPageSizeChange={vi.fn()}
      />
    </main>
  );
  const workspace = container.querySelector('.workspace') as HTMLElement;
  Object.defineProperty(workspace, 'scrollTop', { configurable: true, writable: true, value: 240 });

  fireEvent.click(screen.getByRole('button', { name: '下一页' }));

  expect(onPageChange).toHaveBeenCalledWith(2);
  expect(workspace.scrollTop).toBe(0);
});
```

- [x] **Step 2: 运行测试确认失败**

Run: `npm test -- --run ListPagination`

Expected: 新测试失败，`workspace.scrollTop` 仍为 `240`；其他现有分页栏测试通过。

- [x] **Step 3: 实现最小回顶逻辑**

在 `web/src/ListPagination.tsx` 中导入 `resolveAppScrollElement`，为 `<nav>` 添加 `useRef<HTMLElement>(null)`，并定义页码变更处理函数：

```tsx
const paginationRef = useRef<HTMLElement>(null);

function handlePageChange(nextPage: number) {
  onPageChange(nextPage);
  const scrollElement = resolveAppScrollElement(paginationRef.current);
  if (scrollElement) {
    scrollElement.scrollTop = 0;
  }
}
```

将两个页码按钮的 `onClick` 分别改为 `() => handlePageChange(page - 1)` 与 `() => handlePageChange(page + 1)`，并将 `ref={paginationRef}` 放到 `<nav>`。不修改每页条数选择器。

- [x] **Step 4: 运行测试确认通过**

Run: `npm test -- --run ListPagination`

Expected: `ListPagination` 全部测试通过，包括新增的 `.workspace` 回顶断言。

- [x] **Step 5: 运行完整前端验证**

Run: `npm test && npm run build`

Expected: Vitest 全绿且 TypeScript/Vite 构建成功，无新增警告。

- [x] **Step 6: 提交实现**

```bash
git add web/src/ListPagination.tsx web/src/ListPagination.test.tsx
git commit -m '修复分页切换未回顶问题'
```
