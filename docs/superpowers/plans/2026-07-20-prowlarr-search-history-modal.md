# Prowlarr 搜索历史弹窗 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 Prowlarr 搜索历史改为左上角按钮触发的按需加载弹窗，并取消页面初始化时的历史请求与自动恢复。

**Architecture:** `ProwlarrSearchView` 负责历史弹窗的可见状态和按需加载；继续复用现有 API 与 `AnimatedModal` 的焦点管理。搜索执行完成后不再刷新历史列表，而是清除当前关联的历史 ID，确保仅点击“搜索历史”时才读取历史列表。

**Tech Stack:** React 18、TypeScript、Vitest、Testing Library、lucide-react、既有 `AnimatedModal`。

## Global Constraints

- 不新增后端接口、数据表、历史记录格式或依赖。
- 不在页面初次渲染时请求 `/api/prowlarr/search-history`，也不自动恢复搜索结果。
- 每次点击“搜索历史”才请求列表；列表请求失败时不打开弹窗。
- 恢复历史只读取缓存详情，不重新向 Prowlarr 发起搜索。
- 保持现有删除、清空历史时对当前关键词与结果的同步清理语义。
- 使用既有 `AnimatedModal`，关闭时焦点必须回到触发按钮。
- Git 提交信息使用简体中文。

---

## 文件结构

- `web/src/ProwlarrSearchView.tsx`：历史按钮、弹窗状态、按需列表请求、恢复后的关闭行为，以及移除初始化自动恢复。
- `web/src/ProwlarrSearchView.test.tsx`：用网络请求断言和对话框断言覆盖新的延迟加载、恢复、删除和清空流程。
- `web/src/styles.css`：历史按钮在标题左侧的布局，以及可滚动的历史弹窗内容样式。
- `design-system/pages/prowlarr.md`：把“历史 Chip”的页面规则更新为“历史按钮与弹窗”。

### Task 1: 为按需历史弹窗编写失败测试

**Files:**
- Modify: `web/src/ProwlarrSearchView.test.tsx:924-1305`

**Interfaces:**
- Consumes: `ProwlarrSearchView`、`ToastProvider`、既有 `fetch` mock 模式。
- Produces: 三个面向用户行为的测试：不自动请求、点击请求并显示弹窗、从弹窗恢复后关闭且不搜索。

- [ ] **Step 1: 在测试文件的 `submittedGuidsResponse` 后加入统一的 JSON、已配置 Prowlarr 和历史条目夹具。**

```tsx
const historyEntry = {
  id: 1,
  display_query: 'Inception',
  query: 'inception',
  media_type: 'movie' as const,
  sort_by: 'seeders' as const,
  indexer_ids: [],
  result_count: 1,
  searched_at: '2026-01-01T00:00:00Z'
};

function jsonResponse(body: unknown) {
  return new Response(JSON.stringify(body), { status: 200, headers: { 'Content-Type': 'application/json' } });
}

function configuredProwlarrResponse() {
  return jsonResponse({
    url: 'http://127.0.0.1:9696',
    api_key: 'k',
    download_dir: '/movies',
    tv_download_dir: '/tv',
    movie_rename_enabled: true,
    tmdb_api_key: '',
    indexer_ids: [],
    configured: true
  });
}
```

- [ ] **Step 2: 用以下测试替换“进入页面时自动恢复最近一次搜索历史结果”测试，先断言当前实现不符合新行为。**

```tsx
it('进入页面不请求搜索历史，也不恢复最近一次结果', async () => {
  const historyCalls: string[] = [];
  vi.mocked(fetch).mockImplementation(async (input: RequestInfo | URL) => {
    const path = String(input);
    if (path === '/api/settings/prowlarr') return configuredProwlarrResponse();
    if (path === '/api/prowlarr/indexers') return jsonResponse({ items: [] });
    if (path.startsWith('/api/prowlarr/search-history')) {
      historyCalls.push(path);
      return jsonResponse({ items: [historyEntry] });
    }
    return jsonResponse({});
  });

  render(<ToastProvider><ProwlarrSearchView /></ToastProvider>);

  await waitFor(() => expect(screen.getByRole('button', { name: '搜索历史' })).toBeInTheDocument());
  expect(historyCalls).toEqual([]);
  expect(screen.queryByText('Cached Inception')).not.toBeInTheDocument();
});
```

- [ ] **Step 3: 增加点击后加载列表并打开对话框的失败测试。**

```tsx
it('点击搜索历史后才请求列表并在弹窗展示记录', async () => {
  const historyCalls: string[] = [];
  vi.mocked(fetch).mockImplementation(async (input: RequestInfo | URL) => {
    const path = String(input);
    if (path === '/api/settings/prowlarr') return configuredProwlarrResponse();
    if (path === '/api/prowlarr/indexers') return jsonResponse({ items: [] });
    if (path.startsWith('/api/prowlarr/search-history?')) {
      historyCalls.push(path);
      return jsonResponse({ items: [historyEntry] });
    }
    return jsonResponse({});
  });

  render(<ToastProvider><ProwlarrSearchView /></ToastProvider>);
  const trigger = await screen.findByRole('button', { name: '搜索历史' });
  expect(historyCalls).toEqual([]);
  fireEvent.click(trigger);

  expect(await screen.findByRole('dialog', { name: '搜索历史' })).toBeInTheDocument();
  expect(screen.getByRole('button', { name: /Inception/ })).toBeInTheDocument();
  expect(historyCalls).toHaveLength(1);
});
```

- [ ] **Step 4: 将“点击搜索历史恢复缓存结果且不重新搜索”测试改为先点击触发按钮，并断言恢复后对话框关闭。**

```tsx
const trigger = await screen.findByRole('button', { name: '搜索历史' });
fireEvent.click(trigger);
const historyEntryButton = await screen.findByRole('button', { name: /Inception/ });
fireEvent.click(historyEntryButton);

expect(await screen.findByText('Cached Inception')).toBeInTheDocument();
expect(screen.queryByRole('dialog', { name: '搜索历史' })).not.toBeInTheDocument();
expect(searchCalls).toBe(0);
```

- [ ] **Step 5: 运行针对性测试，确认现有实现失败。**

Run: `npm test -- --run ProwlarrSearchView`

Expected: FAIL；当前实现会在初始化时请求历史且页面没有“搜索历史”按钮/对话框。

### Task 2: 实现标题按钮、延迟请求与历史弹窗

**Files:**
- Modify: `web/src/ProwlarrSearchView.tsx:1-320, 425-480`
- Modify: `web/src/styles.css:609-638, 1034-1083`
- Modify: `design-system/pages/prowlarr.md:23-25`
- Test: `web/src/ProwlarrSearchView.test.tsx:924-1305`

**Interfaces:**
- Consumes: `api.prowlarrSearchHistory(): Promise<{ items: ProwlarrSearchHistory[] }>`、`api.getProwlarrSearchHistory(id)`、`AnimatedModal`。
- Produces: `openHistoryModal(): Promise<void>` 和 `restoreHistoryEntry(entry): Promise<void>`；前者成功时打开弹窗，后者成功时关闭弹窗。

- [ ] **Step 1: 在 `ProwlarrSearchView.tsx` 导入 `AnimatedModal`，添加弹窗状态，并替换自动加载历史的 effect。**

```tsx
import { Download, History, Loader2, Search, Trash2, X } from 'lucide-react';
import { AnimatedModal } from './AnimatedModal';

const [historyModalOpen, setHistoryModalOpen] = useState(false);

const openHistoryModal = useCallback(async () => {
  try {
    const data = await api.prowlarrSearchHistory();
    setHistory(data.items ?? []);
    setHistoryModalOpen(true);
  } catch (err) {
    showToast(messageOf(err), 'error');
  }
}, [showToast]);

useEffect(() => {
  if (!config?.configured) return;
  api.prowlarrIndexers()
    .then((data) => setIndexers(data.items ?? []))
    .catch((err) => showToast(messageOf(err), 'error'));
}, [config?.configured, showToast]);
```

删除 `hasRestoredLatestHistory`、`loadHistory` 以及调用 `loadHistory({ restoreLatest: true })` 的 effect。`runSearch` 成功处理结果后改为 `setActiveHistoryId(null)`，不再调用历史列表接口。

- [ ] **Step 2: 调整恢复函数，确保详情请求成功后才更新页面，并在成功时关闭弹窗。**

```tsx
const restoreHistoryEntry = useCallback(async (entry: ProwlarrSearchHistory) => {
  try {
    const detail = await api.getProwlarrSearchHistory(entry.id);
    const items = detail.results ?? [];
    setActiveHistoryId(entry.id);
    setQuery(entry.display_query);
    setSearchType(entry.media_type);
    setSortBy(entry.sort_by);
    setSelectedIndexerIds(entry.indexer_ids ?? []);
    setSelectedGuids(new Set());
    setBatchSummary(null);
    setBatchFailuresExpanded(false);
    setFurthestSeenIndex(-1);
    setResults(items);
    setResultsSearchType(entry.media_type);
    await hydrateSubmittedGuids(items);
    setHistoryModalOpen(false);
  } catch (err) {
    showToast(messageOf(err), 'error');
  }
}, [hydrateSubmittedGuids, showToast]);
```

- [ ] **Step 3: 以标题区域左侧的按钮替换现有内联历史面板，并在页面末尾渲染标准弹窗。**

```tsx
<header className="view-header prowlarr-search-header">
  <div>
    <button type="button" className="ghost" onClick={() => void openHistoryModal()}>
      <History size={16} aria-hidden />
      搜索历史
    </button>
    <h1>Prowlarr 搜索</h1>
    <p>搜索电影或剧集 Torrent，支持搜索历史与批量下载。</p>
  </div>
</header>

{historyModalOpen && (
  <AnimatedModal onClose={() => setHistoryModalOpen(false)} ariaLabelledBy="prowlarr-search-history-title" panelClassName="prowlarr-history-modal">
    <div className="modal-header-row">
      <div className="horizontal-actions">
        <h2 id="prowlarr-search-history-title" className="modal-title">搜索历史</h2>
        {history.length > 0 && <button type="button" className="ghost" onClick={clearHistory}><Trash2 size={14} aria-hidden />清空</button>}
      </div>
      <button type="button" className="modal-close ghost" aria-label="关闭搜索历史" onClick={() => setHistoryModalOpen(false)}><X size={20} aria-hidden /></button>
    </div>
    {history.length === 0 ? <p className="prowlarr-history-empty muted">暂无搜索历史</p> : (
      <div className="history-chips">
        {history.map((entry) => (
          <div key={entry.id} className="history-chip">
            <button type="button" className="history-chip-main" onClick={() => applyHistory(entry)}>
              <span>{entry.display_query}</span>
              <span className="muted">{entry.media_type === 'tv' ? '剧集' : '电影'} · {entry.result_count} 条</span>
            </button>
            <button type="button" className="history-chip-remove" aria-label="删除" onClick={() => removeHistoryEntry(entry.id)}><X size={14} aria-hidden /></button>
          </div>
        ))}
      </div>
    )}
  </AnimatedModal>
)}
```

- [ ] **Step 4: 为标题按钮和弹窗内容写入最小样式，并更新设计系统页面说明。**

```css
.prowlarr-search-header {
  align-items: flex-start;
}

.prowlarr-search-header > div {
  display: grid;
  justify-items: start;
  gap: 8px;
}

.prowlarr-history-modal {
  width: min(720px, 96vw);
}

.prowlarr-history-empty {
  margin: 0;
  padding: 24px 0;
  text-align: center;
}
```

将 `design-system/pages/prowlarr.md` 的“历史 Chip”段改为“搜索历史弹窗”，明确按钮位于标题左上、点击时加载历史、Chip 仅在弹窗内使用。

- [ ] **Step 5: 扩展删除与清空断言，覆盖弹窗仍打开和空状态。**

```tsx
fireEvent.click(await screen.findByRole('button', { name: '搜索历史' }));
await screen.findByRole('dialog', { name: '搜索历史' });
fireEvent.click(screen.getByRole('button', { name: '清空' }));

await waitFor(() => expect(screen.getByText('搜索历史已清空')).toBeInTheDocument());
expect(screen.getByRole('dialog', { name: '搜索历史' })).toBeInTheDocument();
expect(screen.getByText('暂无搜索历史')).toBeInTheDocument();
```

- [ ] **Step 6: 运行针对性测试，确认全部通过。**

Run: `npm test -- --run ProwlarrSearchView`

Expected: PASS；历史只在点击后请求，弹窗显示与管理历史，恢复后关闭且无新的搜索请求。

- [ ] **Step 7: 执行前端类型检查与生产构建。**

Run: `npm run test:types && npm run build`

Expected: 两条命令均以退出码 0 完成。

- [ ] **Step 8: 审查变更并提交实现。**

Run: `git diff --check && git status --short`

Expected: 无空白错误，且仅包含 `ProwlarrSearchView.tsx`、其测试、`styles.css` 和 Prowlarr 页面设计说明的预期改动。

```bash
git add web/src/ProwlarrSearchView.tsx web/src/ProwlarrSearchView.test.tsx web/src/styles.css design-system/pages/prowlarr.md
git commit -m '将 Prowlarr 搜索历史改为弹窗'
```
