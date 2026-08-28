# 字幕搜索结果分页

## 目标

「字幕」页搜索结果使用现有 `PaginationBar` 分页浏览；当前已加载结果翻尽后，再点「下一页」会请求 OpenSubtitles 的下一页并追加，从而看到超过对方首页的结果。

本 spec 覆盖原字幕设计中明确不做的「结果分页」。

## 非目标

- 按每页条数预取多页 OpenSubtitles 结果
- 两套翻页控件（前端页 + OpenSubtitles 页）
- 向 OpenSubtitles 传递 `page_size`（对方不支持，每页条数由其 `per_page` 决定，通常为 50）
- 搜索历史、批量下载、虚拟滚动
- 用 `useServerPagination` 把前端页码 1:1 映射成 OpenSubtitles 页码

## 背景约束

- OpenSubtitles `GET /api/v1/subtitles` 只接受 `page`（从 1 起），响应含 `page`、`total_pages`、`total_count`、`per_page`、`data`。
- `total_count` 是对方字幕条目数，展平 `files[]` 后行数可以更多，不能当作表格总行数。
- 语言筛选、每页 10/30/50/100 都是前端行为，不重打搜索。

## 架构

后端搜索补上 OpenSubtitles 的 `page`，并把该页元数据原样转给前端。前端累积已加载行，用 `usePagination` 对「语言筛选后的列表」切片。只有在切片末页且对方还有后续页时，才再请求 `page=N+1` 并按 `file_id` 去重追加。

不把本次搜索的前端 `page_size` 传给后端。

## HTTP API

`GET /api/subtitles/search?query=&languages=&page=`

| 参数 | 规则 |
|------|------|
| `query` / `languages` | 不变：空则 400 |
| `page` | 可选，缺省或 `<1` 时为 1；只转发给 OpenSubtitles，不做 `page_size` 规范化 |

成功响应：

```json
{
  "items": [{ "file_id": 1, "file_name": "a.srt", "release": "", "language": "zh-CN", "download_count": 0, "ratings": 0 }],
  "page": 1,
  "total_pages": 5,
  "total_count": 234
}
```

- `items`：该 OpenSubtitles 页展平后的行，规则与现有搜索一致（含 `order_by=download_count`）。
- `page` / `total_pages` / `total_count`：来自对方响应；字段缺失或不可解析时，`page` 用请求页码，`total_pages` 与 `total_count` 为 0。
- `total_pages < 1` 时前端视为没有后续页（`hasMore` 为 false）。
- 不返回 `page_size`，避免与前端每页条数混淆。

未配置 503、对方失败 502 等错误映射不变。

## 后端组件

| 单位 | 职责 |
|------|------|
| `ParseSearchResponse`（`internal/opensubtitles`） | 从原始 JSON 解析分页元数据，并调用现有 `FlattenSearchData` 得到 `items` |
| `Client.Search` | 增加 `page int`（`<1` 则发 `1`），query 增加 `page`；返回 `SearchPage{Items, Page, TotalPages, TotalCount}` |
| `Service.SearchSubtitles` / `handleSubtitlesSearch` | 把 `page` 传下去，按上面的 JSON 写出 |

`FlattenSearchData` 行为不变。

## 前端数据流

字幕页自己管累积，不用 `useServerPagination`。

状态：

- `items`：已加载行（跨 OpenSubtitles 页累积）
- `osPage`：最后一次成功返回的对方 `page`（无成功结果时为 0）
- `osTotalPages`：最后一次成功返回的 `total_pages`（无成功结果时为 0）
- 其余：现有 `query` / `languages` / `resultLanguage` / `searching`

`hasMore = osPage >= 1 && osPage < osTotalPages`。

1. **新搜索**：清空 `items`、`osPage=0`、`osTotalPages=0`，请求 `page=1`，用响应覆盖 `items` 与分页元数据，前端页码回到 1。
2. **语言筛选 / 每页条数**：只重置前端页码为 1，不请求。
3. **上一页 / 未到切片末页的下一页**：只改前端页码；已加载数据保留，不请求。
4. **切片末页再点下一页** 且 `hasMore`：请求 `page=osPage+1`，按 `file_id` 追加（已存在的跳过），更新 `osPage` / `osTotalPages`。每次点击最多请求 1 个 OpenSubtitles 页，不自动连拉。

追加后的前端页码：

- 若筛选后的条数已经够「当前页 + 1」，则进入下一前端页。
- 否则留在当前页（例如每页 100、对方一页约 50 行，新行补在同一页）。

语言筛选后本次追加没有新的可见行：toast 提示「没有更多该语言结果」，若仍 `hasMore` 可再点下一页。不自动连拉。

加载更多失败：保留已有结果，toast 错误；前端页码不变。

## UI

结果表下方使用现有 `PaginationBar`。

- `totalItems` = 语言筛选后的已加载条数（不是 `total_count`）。
- 范围文案随已加载条数变化，例如先「显示 1–30，共 50 条」，追加后变为「共 100 条」。
- 语言筛选按钮上的「全部（N）」与分组人数合同上，随累积更新。
- 先按现有规则分组排序，再展平，再对展平列表切片；当前页重新分组渲染。某一语言的组标题只在该页有该语言行时出现。
- 无结果时仍不渲染分页条。

`PaginationBar` 增加可选属性，默认不影响其它列表：

- `hasMore`：为 true 时，即使已在最后一页，「下一页」仍可点。
- `busy`：为 true 时禁用上一页 / 下一页 / 每页条数（加载更多期间）。

父组件在 `onPageChange(next)` 里区分：`next <= totalPages` 则只改页码；否则走「加载更多」。不得先把前端页码设到超出 `totalPages` 再拉数（`usePagination` 会把越界页码夹回去）。

每页条数选项、回顶行为保持 `PaginationBar` 现有实现。

## 测试

TDD：先写失败测试再写实现。

- `ParseSearchResponse`：读出 `page` / `total_pages` / `total_count`，items 仍走展平规则；缺字段时的缺省；非法 JSON。
- `Client.Search`：请求带上 `page`；`page<1` 时发 `page=1`；其它 query/headers 不变。
- HTTP：`/api/subtitles/search?page=2` 把对方该页元数据写入响应；缺 `page` 时请求对方 `page=1`。
- `PaginationBar`：`hasMore` 时末页下一页可点；`busy` 时按钮禁用；默认行为回归。
- `SubtitlesView`：结果超过每页条数时出现分页条且只渲染当前页；改每页条数 / 语言筛选不发新搜索；末页下一页请求 `page=2` 并追加；`file_id` 重复不双计；加载更多 0 条可见结果时 toast；新搜索重置为对方第 1 页。
