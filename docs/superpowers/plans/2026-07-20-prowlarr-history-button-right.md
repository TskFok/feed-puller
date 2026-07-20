# Prowlarr 搜索历史右上按钮 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 Prowlarr 搜索页的“搜索历史”入口从标题左上移动到标题右上，并在窄屏换行时保持右对齐。

**Architecture:** 将标题文本与按钮拆成标题栏的两个同级元素，复用 `.view-header` 的双端对齐。CSS 在桌面使用 flex 两端布局，在既有窄屏 grid 布局中让按钮 `justify-self: end`；不改弹窗状态、接口或历史交互。

**Tech Stack:** React 18、TypeScript、Vitest、Testing Library、CSS。

## Global Constraints

- “搜索历史”按钮位于标题区域右上；窄屏换行时仍右对齐。
- 点击按钮后获取历史、弹窗内容、错误处理和焦点管理保持不变。
- 不新增接口、依赖或后端改动。
- Git 提交信息使用简体中文。

---

## 文件结构

- `web/src/ProwlarrSearchView.tsx`：将按钮从标题文本容器中移出，成为标题栏右侧同级元素。
- `web/src/ProwlarrSearchView.test.tsx`：断言入口仍是可访问按钮，且带有用于右侧定位的稳定 CSS 类。
- `web/src/styles.css`：定义标题文本容器、右上入口及窄屏右对齐规则。
- `design-system/pages/prowlarr.md`：将布局规则从左上修正为右上。

### Task 1: 迁移入口到标题栏右侧

**Files:**
- Modify: `web/src/ProwlarrSearchView.test.tsx:955-1010`
- Modify: `web/src/ProwlarrSearchView.tsx:421-434`
- Modify: `web/src/styles.css:637-645, 1510-1515`
- Modify: `design-system/pages/prowlarr.md:23-27`

**Interfaces:**
- Consumes: `openHistoryModal(): Promise<void>`；按钮的访问名称保持为“搜索历史”。
- Produces: `.prowlarr-search-header-copy` 与 `.prowlarr-history-trigger`，分别承载标题文本和右侧入口。

- [ ] **Step 1: 在“进入页面不请求搜索历史”测试中，写入入口定位类的失败断言。**

```tsx
const trigger = screen.getByRole('button', { name: '搜索历史' });
expect(trigger).toHaveClass('prowlarr-history-trigger');
expect(trigger.parentElement).toHaveClass('prowlarr-search-header');
```

- [ ] **Step 2: 运行组件测试，确认现有实现因缺少右侧入口类而失败。**

Run: `npm test -- --run ProwlarrSearchView`

Expected: FAIL；断言提示“搜索历史”按钮缺少 `prowlarr-history-trigger`。

- [ ] **Step 3: 将标题文本和按钮改为标题栏同级元素。**

```tsx
<header className="view-header prowlarr-search-header">
  <div className="prowlarr-search-header-copy">
    <h1>Prowlarr 搜索</h1>
    <p>搜索电影或剧集 Torrent，支持搜索历史与批量下载。</p>
  </div>
  <button type="button" className="ghost prowlarr-history-trigger" onClick={() => void openHistoryModal()}>
    <History size={16} aria-hidden />
    搜索历史
  </button>
</header>
```

- [ ] **Step 4: 用以下规则替换旧的 `.prowlarr-search-header > div` 样式，并补充窄屏规则。**

```css
.prowlarr-search-header {
  align-items: flex-start;
}

.prowlarr-search-header-copy {
  display: grid;
  gap: 8px;
}

@media (max-width: 900px) {
  .prowlarr-history-trigger {
    justify-self: end;
  }
}
```

- [ ] **Step 5: 将 Prowlarr 页面说明更新为“标题区域右上”，并明确窄屏保持右对齐。**

```markdown
- 标题区域右上使用“搜索历史”按钮；窄屏换行时仍右对齐，点击时才加载历史记录。
```

- [ ] **Step 6: 运行组件测试，确认入口行为和布局标记均通过。**

Run: `npm test -- --run ProwlarrSearchView`

Expected: PASS；按钮仍可触发历史弹窗，且拥有右侧定位类。

- [ ] **Step 7: 执行类型检查、生产构建与最终差异检查。**

Run: `npm run test:types && npm run build && git diff --check`

Expected: 三条命令均以退出码 0 完成。

- [ ] **Step 8: 提交布局调整。**

```bash
git add web/src/ProwlarrSearchView.tsx web/src/ProwlarrSearchView.test.tsx web/src/styles.css design-system/pages/prowlarr.md docs/superpowers/plans/2026-07-20-prowlarr-history-button-right.md
git commit -m '将搜索历史按钮移至右上角'
```
