# 字幕搜索结果分页 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 「字幕」页用现有 `PaginationBar` 切片已加载结果；翻过末页时再请求 OpenSubtitles 下一页并按 `file_id` 追加。

**Architecture:** 后端 `Search` 增加 `page`，解析对方 `page` / `total_pages` / `total_count` 后原样返回。前端累积已加载行，用 `usePagination` 做语言筛选后的切片；仅在切片末页且 `hasMore` 时请求 `page=osPage+1`。

**Tech Stack:** Go、net/http、React、TypeScript、Vitest、Testing Library。

## Global Constraints

- 不向 OpenSubtitles 传递 `page_size`；前端每页 10/30/50/100 只切已加载结果。
- 不用 `useServerPagination` 把前端页码 1:1 映射成对方页码。
- 每次「加载更多」最多请求 1 个 OpenSubtitles 页，不自动连拉。
- `totalItems` 用语言筛选后的已加载条数，不用 `total_count`。
- JSON 字段 snake_case：`page`、`total_pages`、`total_count`。
- 日志和错误文案不得包含密码或 API Key。
- 禁止循环内 SQL（本功能不新增 SQL）。
- 当前分支修改；提交信息使用中文。
- 本机 Go 测试：`GOOS=darwin GOARCH=arm64 GOCACHE=/private/tmp/feed-puller-gocache-darwin go test <pkg> -run <name> -count=1`。
- 本机前端测试：在仓库根目录 `npx vitest run src/<file>.test.ts`。

---

## File Structure

- Modify: `internal/opensubtitles/flatten.go` — 新增 `SearchPage`、`ParseSearchResponse`
- Modify: `internal/opensubtitles/flatten_test.go`
- Modify: `internal/opensubtitles/client.go` — `Search` 增加 `page`，返回 `SearchPage`
- Modify: `internal/opensubtitles/client_test.go`
- Modify: `internal/app/service_opensubtitles.go` — `SearchSubtitles` 增加 `page`，返回 `SearchPage`
- Modify: `internal/app/service_opensubtitles_test.go`
- Modify: `internal/httpapi/opensubtitles_handlers.go` — 解析 `page`，写出分页元数据
- Modify: `internal/httpapi/opensubtitles_handlers_test.go`
- Modify: `web/src/ListPagination.tsx` — 可选 `hasMore`、`busy`
- Modify: `web/src/ListPagination.test.tsx`
- Modify: `web/src/types.ts` — `SubtitleSearchResult` 增加分页字段
- Modify: `web/src/api.ts` — `searchSubtitles` 增加 `page`
- Modify: `web/src/api.opensubtitles.test.ts`
- Modify: `web/src/SubtitlesView.tsx`
- Modify: `web/src/SubtitlesView.test.tsx`

### Task 1: 解析 OpenSubtitles 搜索分页元数据

**Files:**

- Modify: `internal/opensubtitles/flatten.go`
- Modify: `internal/opensubtitles/flatten_test.go`

**Interfaces:**

- Produces: `type SearchPage struct { Items []SubtitleFile; Page int; TotalPages int; TotalCount int }`，JSON 标签 `items`、`page`、`total_pages`、`total_count`。
- Produces: `func ParseSearchResponse(raw []byte, requestPage int) SearchPage`。
- Consumes: 现有 `FlattenSearchData`。

- [ ] **Step 1: 写入失败测试**

在 `flatten_test.go` 追加：

```go
func TestParseSearchResponse_ReadsPaginationAndFlattens(t *testing.T) {
	t.Parallel()
	raw := []byte(`{"page":2,"total_pages":5,"total_count":234,"data":[{"attributes":{"release":"Show","language":"zh-CN","download_count":1,"files":[{"file_id":101,"file_name":"a.srt"}]}}]}`)
	got := ParseSearchResponse(raw, 1)
	if got.Page != 2 || got.TotalPages != 5 || got.TotalCount != 234 {
		t.Fatalf("meta=%+v", got)
	}
	if len(got.Items) != 1 || got.Items[0].FileID != 101 || got.Items[0].FileName != "a.srt" {
		t.Fatalf("items=%+v", got.Items)
	}
}

func TestParseSearchResponse_MissingMetaUsesRequestPageAndZeroTotals(t *testing.T) {
	t.Parallel()
	raw := []byte(`{"data":[{"attributes":{"files":[{"file_id":1,"file_name":"a.srt"}]}}]}`)
	got := ParseSearchResponse(raw, 3)
	if got.Page != 3 || got.TotalPages != 0 || got.TotalCount != 0 {
		t.Fatalf("meta=%+v", got)
	}
	if len(got.Items) != 1 || got.Items[0].FileID != 1 {
		t.Fatalf("items=%+v", got.Items)
	}
}

func TestParseSearchResponse_InvalidJSON(t *testing.T) {
	t.Parallel()
	got := ParseSearchResponse([]byte(`not json`), 2)
	if got.Page != 2 || got.TotalPages != 0 || got.TotalCount != 0 || len(got.Items) != 0 {
		t.Fatalf("got=%+v", got)
	}
}
```

- [ ] **Step 2: 跑测试确认失败**

Run: `GOOS=darwin GOARCH=arm64 GOCACHE=/private/tmp/feed-puller-gocache-darwin go test ./internal/opensubtitles -run TestParseSearchResponse -count=1`

Expected: FAIL，`ParseSearchResponse` 未定义。

- [ ] **Step 3: 最小实现**

在 `flatten.go` 增加：

```go
type SearchPage struct {
	Items      []SubtitleFile `json:"items"`
	Page       int            `json:"page"`
	TotalPages int            `json:"total_pages"`
	TotalCount int            `json:"total_count"`
}

func ParseSearchResponse(raw []byte, requestPage int) SearchPage {
	page := SearchPage{Page: requestPage, Items: []SubtitleFile{}}
	var meta struct {
		Page       int `json:"page"`
		TotalPages int `json:"total_pages"`
		TotalCount int `json:"total_count"`
	}
	if json.Unmarshal(raw, &meta) == nil {
		if meta.Page > 0 {
			page.Page = meta.Page
		}
		page.TotalPages = meta.TotalPages
		page.TotalCount = meta.TotalCount
	}
	if items := FlattenSearchData(raw); len(items) > 0 {
		page.Items = items
	}
	return page
}
```

`FlattenSearchData` 保持不变。`json` 已在该文件 import。

- [ ] **Step 4: 跑测试确认通过**

Run: `GOOS=darwin GOARCH=arm64 GOCACHE=/private/tmp/feed-puller-gocache-darwin go test ./internal/opensubtitles -run 'Test(ParseSearchResponse|FlattenSearchData)' -count=1`

Expected: PASS。

- [ ] **Step 5: 提交**

```bash
git add internal/opensubtitles/flatten.go internal/opensubtitles/flatten_test.go
git commit -m "$(cat <<'EOF'
解析 OpenSubtitles 搜索响应中的分页元数据。

EOF
)"
```

### Task 2: 搜索客户端转发 page 并返回 SearchPage

**Files:**

- Modify: `internal/opensubtitles/client.go`
- Modify: `internal/opensubtitles/client_test.go`
- Modify: `internal/app/service_opensubtitles.go`
- Modify: `internal/app/service_opensubtitles_test.go`

**Interfaces:**

- Consumes: `ParseSearchResponse`、`SearchPage`。
- Produces: `func (c *Client) Search(ctx context.Context, query, language string, page int) (SearchPage, error)`。`page < 1` 时请求 `page=1`。
- Produces: `func (s *Service) SearchSubtitles(ctx context.Context, query, language string, page int) (opensubtitles.SearchPage, error)`。

- [ ] **Step 1: 写入失败测试**

在 `client_test.go` 的 `TestClientSearch_SendsQueryLanguageAndHeaders` 中断言 `r.URL.Query().Get("page") == "1"`，并把返回值改为 `SearchPage`。追加：

```go
func TestClientSearch_ForwardsPageAndClampsBelowOne(t *testing.T) {
	var gotPage string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPage = r.URL.Query().Get("page")
		_, _ = w.Write([]byte(`{"page":2,"total_pages":4,"total_count":80,"data":[{"attributes":{"files":[{"file_id":7,"file_name":"a.srt"}]}}]}`))
	}))
	defer server.Close()
	orig := APIBaseURL
	APIBaseURL = server.URL
	t.Cleanup(func() { APIBaseURL = orig })
	got, err := NewClient("u", "p", "key-1").Search(context.Background(), "Inception", "zh-CN", 0)
	if err != nil || gotPage != "1" || got.Page != 2 || got.TotalPages != 4 || got.TotalCount != 80 || len(got.Items) != 1 {
		t.Fatalf("got=%+v page=%s err=%v", got, gotPage, err)
	}
}

func TestClientSearch_SendsRequestedPage(t *testing.T) {
	var gotPage string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPage = r.URL.Query().Get("page")
		_, _ = w.Write([]byte(`{"page":3,"total_pages":3,"data":[]}`))
	}))
	defer server.Close()
	orig := APIBaseURL
	APIBaseURL = server.URL
	t.Cleanup(func() { APIBaseURL = orig })
	got, err := NewClient("u", "p", "key-1").Search(context.Background(), "Inception", "zh-CN", 3)
	if err != nil || gotPage != "3" || got.Page != 3 || got.TotalPages != 3 || len(got.Items) != 0 {
		t.Fatalf("got=%+v page=%s err=%v", got, gotPage, err)
	}
}
```

把现有 `TestClientSearch_SendsQueryLanguageAndHeaders` / `EmptyQuery` / `Non2xx*` 的 `Search(..., query, lang)` 改为传入 `1`，成功路径解包 `SearchPage`。

在 `service_opensubtitles_test.go` 把 `SearchSubtitles(ctx, query, lang)` 改为传入 `1`，成功路径改为读 `SearchPage`：

```go
got, err := svc.SearchSubtitles(context.Background(), "Inception", "zh-CN", 1)
if err != nil || len(got.Items) != 1 || got.Items[0].FileID != 7 {
	t.Fatalf("got=%+v err=%v", got, err)
}
```

`TestSearchSubtitles_NotConfigured` 同样补 `page` 参数（返回值可忽略）。

- [ ] **Step 2: 跑测试确认失败**

Run: `GOOS=darwin GOARCH=arm64 GOCACHE=/private/tmp/feed-puller-gocache-darwin go test ./internal/opensubtitles -run TestClientSearch -count=1`

Expected: FAIL，`Search` 参数个数不匹配。

- [ ] **Step 3: 最小实现**

`client.go` 增加 `"strconv"`。将 `Search` 改为：

```go
func (c *Client) Search(ctx context.Context, query, language string, page int) (SearchPage, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return SearchPage{}, ErrEmptyQuery
	}
	if page < 1 {
		page = 1
	}
	endpoint, err := url.Parse(strings.TrimRight(APIBaseURL, "/") + "/subtitles")
	if err != nil {
		return SearchPage{}, fmt.Errorf("搜索字幕失败: %w", err)
	}
	q := endpoint.Query()
	q.Set("query", query)
	q.Set("languages", language)
	q.Set("order_by", "download_count")
	q.Set("order_direction", "desc")
	q.Set("page", strconv.Itoa(page))
	endpoint.RawQuery = q.Encode()
	// 其余 HTTP 逻辑与现在相同，成功时：
	return ParseSearchResponse(body, page), nil
}
```

`service_opensubtitles.go`：

```go
func (s *Service) SearchSubtitles(ctx context.Context, query, language string, page int) (opensubtitles.SearchPage, error) {
	cfg, err := s.store.GetOpenSubtitlesConfig(ctx)
	if err != nil {
		return opensubtitles.SearchPage{}, err
	}
	if !cfg.Configured {
		return opensubtitles.SearchPage{}, ErrOpenSubtitlesNotConfigured
	}
	return s.opensubtitlesClientFor(cfg).Search(ctx, query, language, page)
}
```

handler 仍按旧签名调用时会编译失败，下一步立刻改 handler。若本步要单独编译，可暂时让 handler 传入 `1` 并只取 `Items`（Step 3b，见下）。

在本步同时把 handler 调用改成 `SearchSubtitles(ctx, query, language, 1)` 并继续只写 `{"items": result.Items}`，这样 `go test ./internal/httpapi` 仍通过。分页 query 留给 Task 3。

- [ ] **Step 4: 跑测试确认通过**

Run:

```
GOOS=darwin GOARCH=arm64 GOCACHE=/private/tmp/feed-puller-gocache-darwin go test ./internal/opensubtitles ./internal/app ./internal/httpapi -count=1
```

Expected: PASS。

- [ ] **Step 5: 提交**

```bash
git add internal/opensubtitles/client.go internal/opensubtitles/client_test.go internal/app/service_opensubtitles.go internal/app/service_opensubtitles_test.go internal/httpapi/opensubtitles_handlers.go
git commit -m "$(cat <<'EOF'
搜索字幕时向 OpenSubtitles 转发页码。

EOF
)"
```

### Task 3: HTTP 搜索接口返回分页元数据

**Files:**

- Modify: `internal/httpapi/opensubtitles_handlers.go`
- Modify: `internal/httpapi/opensubtitles_handlers_test.go`

**Interfaces:**

- Consumes: `parse page` 从 query；`Service.SearchSubtitles(..., page)`。
- Produces: `200` JSON `{items, page, total_pages, total_count}`。缺 `page` 或 `<1` 时下游发对方 `page=1`。

- [ ] **Step 1: 写入失败测试**

```go
func TestOpenSubtitlesSearch_ForwardsPageAndReturnsMeta(t *testing.T) {
	osAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/subtitles" {
			t.Fatalf("path %s", r.URL.Path)
		}
		if r.URL.Query().Get("page") != "2" {
			t.Fatalf("query=%s", r.URL.RawQuery)
		}
		_, _ = w.Write([]byte(`{"page":2,"total_pages":5,"total_count":120,"data":[{"attributes":{"release":"Inception","language":"zh-CN","files":[{"file_id":7,"file_name":"a.srt"}]}}]}`))
	}))
	defer osAPI.Close()
	orig := opensubtitles.APIBaseURL
	opensubtitles.APIBaseURL = osAPI.URL
	t.Cleanup(func() { opensubtitles.APIBaseURL = orig })
	srv, mock, cleanup := newOpenSubtitlesServer(t)
	defer cleanup()
	expectConfiguredOpenSubtitlesSettings(mock)
	req := authRequest(httptest.NewRequest(http.MethodGet, "/api/subtitles/search?query=Inception&languages=zh-CN&page=2", nil))
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
	var payload opensubtitles.SearchPage
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Page != 2 || payload.TotalPages != 5 || payload.TotalCount != 120 || len(payload.Items) != 1 || payload.Items[0].FileID != 7 {
		t.Fatalf("payload=%+v", payload)
	}
}

func TestOpenSubtitlesSearch_DefaultPageIsOne(t *testing.T) {
	var gotPage string
	osAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPage = r.URL.Query().Get("page")
		_, _ = w.Write([]byte(`{"page":1,"total_pages":1,"data":[]}`))
	}))
	defer osAPI.Close()
	orig := opensubtitles.APIBaseURL
	opensubtitles.APIBaseURL = osAPI.URL
	t.Cleanup(func() { opensubtitles.APIBaseURL = orig })
	srv, mock, cleanup := newOpenSubtitlesServer(t)
	defer cleanup()
	expectConfiguredOpenSubtitlesSettings(mock)
	req := authRequest(httptest.NewRequest(http.MethodGet, "/api/subtitles/search?query=Inception&languages=zh-CN", nil))
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || gotPage != "1" {
		t.Fatalf("code=%d page=%s body=%s", rec.Code, gotPage, rec.Body.String())
	}
}
```

- [ ] **Step 2: 跑测试确认失败**

Run: `GOOS=darwin GOARCH=arm64 GOCACHE=/private/tmp/feed-puller-gocache-darwin go test ./internal/httpapi -run 'TestOpenSubtitlesSearch_ForwardsPage|TestOpenSubtitlesSearch_DefaultPage' -count=1`

Expected: FAIL，响应无 `total_pages` 或上游未收到 `page=2`。

- [ ] **Step 3: 最小实现**

`opensubtitles_handlers.go` 增加 `"strconv"`。`handleSubtitlesSearch` 在校验 query/languages 之后：

```go
page, _ := strconv.Atoi(r.URL.Query().Get("page"))
result, err := s.service.SearchSubtitles(r.Context(), query, language, page)
if err != nil {
	writeOpenSubtitlesError(w, err)
	return
}
if result.Items == nil {
	result.Items = []opensubtitles.SubtitleFile{}
}
writeJSON(w, http.StatusOK, result)
```

不要解析或转发 `page_size`。

- [ ] **Step 4: 跑测试确认通过**

Run: `GOOS=darwin GOARCH=arm64 GOCACHE=/private/tmp/feed-puller-gocache-darwin go test ./internal/httpapi -run OpenSubtitles -count=1`

Expected: PASS。

- [ ] **Step 5: 提交**

```bash
git add internal/httpapi/opensubtitles_handlers.go internal/httpapi/opensubtitles_handlers_test.go
git commit -m "$(cat <<'EOF'
字幕搜索接口返回 OpenSubtitles 分页元数据。

EOF
)"
```

### Task 4: PaginationBar 支持 hasMore 与 busy

**Files:**

- Modify: `web/src/ListPagination.tsx`
- Modify: `web/src/ListPagination.test.tsx`

**Interfaces:**

- Produces: `PaginationBarProps.hasMore?: boolean`（默认 false）；`busy?: boolean`（默认 false）。
- 其它列表不传这两个属性时行为与现在完全一致。

- [ ] **Step 1: 写入失败测试**

在 `ListPagination.test.tsx` 追加：

```tsx
it('hasMore 时末页仍可点下一页', () => {
  const onPageChange = vi.fn();
  render(
    <PaginationBar
      page={2}
      pageSize={30}
      totalPages={2}
      totalItems={35}
      rangeStart={31}
      rangeEnd={35}
      hasMore
      onPageChange={onPageChange}
      onPageSizeChange={vi.fn()}
    />
  );
  const next = screen.getByRole('button', { name: '下一页' });
  expect(next).toBeEnabled();
  fireEvent.click(next);
  expect(onPageChange).toHaveBeenCalledWith(3);
});

it('busy 时禁用翻页和每页条数', () => {
  render(
    <PaginationBar
      page={1}
      pageSize={30}
      totalPages={2}
      totalItems={35}
      rangeStart={1}
      rangeEnd={30}
      hasMore
      busy
      onPageChange={vi.fn()}
      onPageSizeChange={vi.fn()}
    />
  );
  expect(screen.getByRole('button', { name: '上一页' })).toBeDisabled();
  expect(screen.getByRole('button', { name: '下一页' })).toBeDisabled();
  expect(screen.getByRole('combobox')).toBeDisabled();
});
```

- [ ] **Step 2: 跑测试确认失败**

Run: `npx vitest run src/ListPagination.test.tsx`

Expected: FAIL，末页下一页仍 disabled，或 `busy` 未禁用控件。

- [ ] **Step 3: 最小实现**

`PaginationBarProps` 增加 `hasMore?: boolean`、`busy?: boolean`。解构默认 `hasMore = false`、`busy = false`。

```tsx
<button type="button" className="ghost" disabled={page <= 1 || busy} aria-label="上一页" onClick={() => handlePageChange(page - 1)}>上一页</button>
<button type="button" className="ghost" disabled={(page >= totalPages && !hasMore) || busy} aria-label="下一页" onClick={() => handlePageChange(page + 1)}>下一页</button>
<select ... disabled={busy} ...>
```

- [ ] **Step 4: 跑测试确认通过**

Run: `npx vitest run src/ListPagination.test.tsx`

Expected: PASS。

- [ ] **Step 5: 提交**

```bash
git add web/src/ListPagination.tsx web/src/ListPagination.test.tsx
git commit -m "$(cat <<'EOF'
分页条支持还有后续页与加载中禁用。

EOF
)"
```

### Task 5: 前端 API 传递 page 并读取分页元数据

**Files:**

- Modify: `web/src/types.ts`
- Modify: `web/src/api.ts`
- Modify: `web/src/api.opensubtitles.test.ts`

**Interfaces:**

- Produces: `SubtitleSearchResult = { items: SubtitleSearchItem[]; page: number; total_pages: number; total_count: number }`。
- Produces: `api.searchSubtitles(query, languages, page = 1)`，query 含 `page`。缺省字段时 `page` 用入参，`total_pages` / `total_count` 为 0。

- [ ] **Step 1: 写入失败测试**

在 `api.opensubtitles.test.ts` 把现有搜索断言改为可读 `page`。追加：

```ts
it('searchSubtitles 带 page 并回传分页元数据', async () => {
  vi.stubGlobal('fetch', vi.fn(async (input: RequestInfo | URL) => {
    const params = new URL(String(input), 'http://local.test').searchParams;
    expect(params.get('query')).toBe('Inception');
    expect(params.get('languages')).toBe('zh-CN');
    expect(params.get('page')).toBe('2');
    return new Response(JSON.stringify({
      items: [{ file_id: 7, file_name: 'a.srt', release: 'Inception', language: 'zh-CN', download_count: 1, ratings: 9 }],
      page: 2,
      total_pages: 5,
      total_count: 120
    }), { status: 200, headers: { 'Content-Type': 'application/json' } });
  }));
  const res = await api.searchSubtitles('Inception', 'zh-CN', 2);
  expect(res.page).toBe(2);
  expect(res.total_pages).toBe(5);
  expect(res.total_count).toBe(120);
  expect(res.items[0]?.file_id).toBe(7);
});

it('searchSubtitles 缺省 page 为 1，缺元数据时补零', async () => {
  vi.stubGlobal('fetch', vi.fn(async (input: RequestInfo | URL) => {
    const params = new URL(String(input), 'http://local.test').searchParams;
    expect(params.get('page')).toBe('1');
    return new Response(JSON.stringify({ items: [] }), {
      status: 200, headers: { 'Content-Type': 'application/json' }
    });
  }));
  const res = await api.searchSubtitles('Inception', 'zh-CN');
  expect(res.items).toEqual([]);
  expect(res.page).toBe(1);
  expect(res.total_pages).toBe(0);
  expect(res.total_count).toBe(0);
});
```

现有「带 query 与 languages」用例改为同时 `expect(params.get('page')).toBe('1')`。

- [ ] **Step 2: 跑测试确认失败**

Run: `npx vitest run src/api.opensubtitles.test.ts`

Expected: FAIL，URL 无 `page` 或返回值无 `total_pages`。

- [ ] **Step 3: 最小实现**

`types.ts`：

```ts
export type SubtitleSearchResult = {
  items: SubtitleSearchItem[];
  page: number;
  total_pages: number;
  total_count: number;
};
```

`api.ts`：

```ts
searchSubtitles: async (query: string, languages: string, page = 1) => {
  const params = new URLSearchParams({ query, languages, page: String(page) });
  const data = await request<Partial<SubtitleSearchResult>>(`/api/subtitles/search?${params.toString()}`);
  return {
    items: Array.isArray(data.items) ? data.items : [],
    page: data.page ?? page,
    total_pages: data.total_pages ?? 0,
    total_count: data.total_count ?? 0
  };
},
```

在 `api.ts` 顶部 type import 增加 `SubtitleSearchResult`。不要用 `normalizePaginated`（会注入 `page_size`）。

- [ ] **Step 4: 跑测试确认通过**

Run: `npx vitest run src/api.opensubtitles.test.ts`

Expected: PASS。

- [ ] **Step 5: 提交**

```bash
git add web/src/types.ts web/src/api.ts web/src/api.opensubtitles.test.ts
git commit -m "$(cat <<'EOF'
字幕搜索客户端传递页码并读取分页元数据。

EOF
)"
```

### Task 6: 字幕页切片分页与加载更多

**Files:**

- Modify: `web/src/SubtitlesView.tsx`
- Modify: `web/src/SubtitlesView.test.tsx`

**Interfaces:**

- Consumes: `api.searchSubtitles(query, languages, page)`、`usePagination`、`PaginationBar` 的 `hasMore` / `busy`。
- `hasMore = osPage >= 1 && osPage < osTotalPages`。
- 新搜索请求 `page=1` 并覆盖 `items`。
- 切片末页且 `hasMore` 时请求 `osPage+1`，按 `file_id` 追加。
- 语言筛选 / 每页条数不发请求。
- 追加后若筛选条数已够下一前端页则 `setPage(page+1)`，否则留在当前页。
- 追加后当前语言可见行未增加：`showToast('没有更多该语言结果')`。
- 加载更多失败：`showToast(message, 'error')`，保留已有结果。

- [ ] **Step 1: 写入失败测试**

在 `SubtitlesView.test.tsx` 增加 `beforeEach(() => { localStorage.clear(); localStorage.setItem('feed-puller.page-size', '10'); })`。

辅助函数：

```ts
function subtitleItem(id: number, language = 'zh-CN'): SubtitleSearchItem {
  return {
    file_id: id,
    file_name: `f${id}.srt`,
    release: `rel-${id}`,
    language,
    download_count: id,
    ratings: 1
  };
}

function configuredFetch(handler: (path: string, init?: RequestInit) => Response | Promise<Response>) {
  return vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const path = String(input);
    if (path === '/api/settings/opensubtitles') {
      return new Response(JSON.stringify({
        username: 'u', password: 'p', api_key: 'k', download_dir: '/data/subtitles', configured: true
      }), { status: 200, headers: { 'Content-Type': 'application/json' } });
    }
    return handler(path, init);
  });
}
```

用例（均需先搜到结果）：

1. **超过每页条数时只渲染当前页**  
   返回 12 条 zh-CN、`total_pages: 1`。断言有 `rel-12`（download_count 最大，排第一），没有 `rel-1`；有「显示 1–10，共 12 条」；点下一页后出现更小 id 的行。

2. **改每页条数 / 语言筛选不发新搜索**  
   混合 2 条 zh-CN + 1 条 en，`total_pages: 1`。记录 search 调用次数。点「英语」后再改每页条数为 30。`search` 仍为 1 次。

3. **末页下一页请求 page=2 并追加**  
   第一次 10 条 zh-CN、`page:1,total_pages:2`；点下一页（此时已是末页）发 `page=2`，返回 `rel-100`。断言新旧行都在累积后的列表里（可能要再翻页才能看见新行：page_size=10 时 10+1=11，应进入第 2 前端页并看到 `rel-100`）。

4. **file_id 重复不双计**  
   第二页返回已存在的 `file_id`。断言「共 10 条」不变，且该 `file_id` 只出现一行。

5. **加载更多 0 条可见结果时 toast**  
   第 1 页 2 条 zh-CN + 1 条 en，`total_pages:2`。点「英语」，再点下一页。第 2 页只有 zh-CN。断言 toast「没有更多该语言结果」，英语行仍在。

6. **新搜索重置为对方第 1 页**  
   先加载到 page=2，再改名称搜索。第二次请求 `page=1`，第一次的追加行消失。

现有搜索测试继续有效：它们的结果少于 10 条时不出现分页条。

- [ ] **Step 2: 跑测试确认失败**

Run: `npx vitest run src/SubtitlesView.test.tsx`

Expected: FAIL，无分页条或未请求 `page=2`。

- [ ] **Step 3: 最小实现**

`SubtitlesView.tsx` 要点：

```tsx
const [osPage, setOsPage] = useState(0);
const [osTotalPages, setOsTotalPages] = useState(0);
const [loadingMore, setLoadingMore] = useState(false);

const allGroups = groupSubtitleItems(items);
const visibleGroups = resultLanguage === '' ? allGroups : allGroups.filter((g) => g.language === resultLanguage);
const flatVisible = visibleGroups.flatMap((g) => g.items);
const pagination = usePagination(flatVisible.length, [resultLanguage]);
const pagedItems = pagination.slice(flatVisible);
const pagedGroups = groupSubtitleItems(pagedItems);
const hasMore = osPage >= 1 && osPage < osTotalPages;

async function handleSearch(...) {
  setItems([]);
  setOsPage(0);
  setOsTotalPages(0);
  pagination.setPage(1);
  const data = await api.searchSubtitles(trimmed, languageParam, 1);
  setItems(data.items ?? []);
  setOsPage(data.page);
  setOsTotalPages(data.total_pages);
}

async function loadMore() {
  if (loadingMore || !hasMore) return;
  const visibleBefore = flatVisible.length;
  setLoadingMore(true);
  try {
    const data = await api.searchSubtitles(query.trim(), joinSubtitleLanguages(languages), osPage + 1);
    const seen = new Set(items.map((item) => item.file_id));
    const appended = (data.items ?? []).filter((item) => !seen.has(item.file_id));
    const nextItems = [...items, ...appended];
    setItems(nextItems);
    setOsPage(data.page);
    setOsTotalPages(data.total_pages);
    const nextVisible = /* 用 nextItems 按当前 resultLanguage 重算 flatVisible.length */;
    if (nextVisible > pagination.page * pagination.pageSize) {
      pagination.setPage(pagination.page + 1);
    } else if (nextVisible === visibleBefore) {
      showToast('没有更多该语言结果');
    }
  } catch (err) {
    showToast(messageOf(err), 'error');
  } finally {
    setLoadingMore(false);
  }
}

function handlePageChange(next: number) {
  if (next <= pagination.totalPages) {
    pagination.setPage(next);
    return;
  }
  void loadMore();
}
```

表格改为渲染 `pagedGroups`。`PaginationBar` 放在表格下方，传入 `hasMore`、`busy={loadingMore}`。

`usePagination` 的 `setPage` 在搜索开始时调用；`resetDeps` 含 `resultLanguage` 即可，新搜索在 `handleSearch` 里显式 `setPage(1)`。

重算 `nextVisible` 时复用 `groupSubtitleItems` + 语言过滤，不要在循环里请求接口。

注意：`handleSearch` / `loadMore` 不要把 `pagination.setPage` 放进会过期的闭包而不更新；需要时用函数式更新或把 `page`/`pageSize` 列入依赖并在点击时读取最新值。

- [ ] **Step 4: 跑测试确认通过**

Run: `npx vitest run src/SubtitlesView.test.tsx src/ListPagination.test.tsx src/api.opensubtitles.test.ts`

Expected: PASS。再跑：

```
GOOS=darwin GOARCH=arm64 GOCACHE=/private/tmp/feed-puller-gocache-darwin go test ./internal/opensubtitles ./internal/app ./internal/httpapi -count=1
```

Expected: PASS。

- [ ] **Step 5: 提交**

```bash
git add web/src/SubtitlesView.tsx web/src/SubtitlesView.test.tsx
git commit -m "$(cat <<'EOF'
字幕搜索结果支持分页浏览并按需加载后续页。

EOF
)"
```

---

## Spec coverage

| Spec 要求 | 任务 |
|-----------|------|
| `ParseSearchResponse` 读 page/total_pages/total_count，非法 JSON 缺省 | Task 1 |
| `Client.Search` 发 `page`，`<1` 当 1 | Task 2 |
| HTTP `page=2` 转发并返回元数据；缺 page 请求对方 page=1 | Task 3 |
| 不返回 / 不转发 `page_size` | Task 3、5 |
| `PaginationBar.hasMore` / `busy` | Task 4 |
| 前端 API `page` 与缺省补零 | Task 5 |
| 切片分页、语言筛选不重搜、末页加载更多、去重、0 可见 toast、新搜索重置 | Task 6 |
| `totalItems` 为已加载筛选条数 | Task 6 |
| 每次加载更多只请求 1 页 | Task 6 |
| 不用 `useServerPagination` | Task 6 |

## Type consistency

- Go：`SearchPage`、`ParseSearchResponse(raw, requestPage)`、`Client.Search(..., page int) (SearchPage, error)`、`Service.SearchSubtitles(..., page int) (SearchPage, error)`。
- JSON：`items`、`page`、`total_pages`、`total_count`。
- TS：`SubtitleSearchResult` 同上；`api.searchSubtitles(query, languages, page = 1)`。
- UI：`hasMore`、`busy`；toast 文案「没有更多该语言结果」。
