# OpenSubtitles 字幕搜索与下载

## 目标

在设置页配置 OpenSubtitles 账号，并在侧栏提供「字幕」页：按名称与语言搜索字幕，将选中文件保存到配置的服务器目录。

不经过 aria2，不写入下载任务表，不出现在「下载中 / 下载完成」。

## 非目标

- 搜索历史、结果分页、批量下载
- 按季/集、IMDB、文件哈希搜索
- 浏览器本地下载
- 将字幕提交给 aria2

## 配置存储

沿用既有 `settings` 表，四个固定键：

| 键 | 含义 |
|----|------|
| `opensubtitles_username` | 用户名 |
| `opensubtitles_password` | 密码 |
| `opensubtitles_api_key` | REST API Key |
| `opensubtitles_download_dir` | 服务器上的保存目录 |

四项去除首尾空白后均非空，才视为已配置。GET 空库时四项为空字符串且 `configured=false`。GET 返回明文，供设置页回显（与 Prowlarr API Key 一致）。PUT 先校验四项均非空，再写入；任一为空则 400，且不调用任何 `SetSetting`（禁止部分写入）。日志和错误文案不得包含密码或 API Key。

## 后端客户端

新增 `internal/opensubtitles`，只由服务端调用 `https://api.opensubtitles.com/api/v1`。HTTP 客户端不使用 RSS 全局代理（与 Prowlarr 一致）。默认超时 30 秒。

所有请求携带：

- `Api-Key: <配置的 key>`
- `User-Agent: feed-puller v1.0`
- `Accept: application/json`（JSON 接口）
- 登录与下载另加 `Content-Type: application/json`
- 下载接口另加 `Authorization: Bearer <token>`

### 搜索

`GET /api/v1/subtitles?query=<名称>&languages=<语言码>`

`languages` 为字幕页下拉选中的单个语言码。只请求第一页。将返回的 `data[]` 展平为「每个 `files[]` 条目一行」，行字段：

- `file_id`：`files[].file_id`（下载用）
- `file_name`：`files[].file_name`
- `release`：`attributes.release`，空则用 `attributes.feature_details.movie_name` 或 `title`
- `language`：`attributes.language`
- `download_count`：`attributes.download_count`
- `ratings`：`attributes.ratings`

无 `files`、或 `file_id` 缺失/≤0 的条目丢弃，不展示。

### 登录与 token

`POST /api/v1/login`，body：`{"username","password"}`。

进程内缓存 token。下载接口返回 401 时清空缓存、重新登录，并对该次下载再试一次。两次仍失败则返回错误。不把 token 写入数据库。

### 下载落盘

1. `POST /api/v1/download`，body：`{"file_id": <number>}`（`file_id` 必须 >0，否则 400）
2. 用响应中的 `link` 做 HTTP GET（跟随重定向）
3. 文件名优先用下载接口返回的 `file_name`；为空则用请求体里的 `file_name`（搜索行传入）
4. 只保留 `filepath.Base` 结果；若消毒后为空、为 `.` 或 `..`，则 400
5. 保存路径为 `filepath.Join(download_dir, sanitized_name)`，响应 `path` 即该字符串；同名覆盖
6. 将 GET `link` 的响应 body 原样写入文件（Go `http.Client` 会解码 HTTP `Content-Encoding`；不做额外解压）
7. 不创建下载任务、不调用 aria2

目录不存在或不可写时失败，错误中说明原因。不自动 `MkdirAll`。

## HTTP API

均需现有登录态。未登录返回 401。

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/settings/opensubtitles` | 返回配置；含 `configured` 布尔值 |
| PUT | `/api/settings/opensubtitles` | 保存四项；任一为空则 400 且不写库 |
| GET | `/api/subtitles/search?query=&languages=` | `query` 为空 400；未配置 503 |
| POST | `/api/subtitles/download` | body：`{"file_id": number}`，可选 `file_name` 作 fallback；未配置 503 |

JSON 字段一律 snake_case。配置体：`username`、`password`、`api_key`、`download_dir`、`configured`。

搜索响应：`{ "items": [ { file_id, file_name, release, language, download_count, ratings } ] }`。

下载成功：`{ "path": "<filepath.Join 结果>", "file_name": "<sanitized_name>" }`。前端下载请求必须带上该行的 `file_id` 和 `file_name`。

OpenSubtitles 4xx/5xx、超时、网络失败映射为 502，优先使用对方 `message`；登录失败固定文案「OpenSubtitles 登录失败」。数据库失败 500。库写入失败时已有配置不变。

## 前端

沿用现有 Glassmorphism 面板与 Lucide 图标，不新增视觉风格。

### 设置页

在 Prowlarr 面板之后增加「OpenSubtitles」表单：用户名、密码（`type=password` 且可显示/隐藏）、API Key、字幕下载目录（占位 `/data/subtitles`）、主按钮「保存 OpenSubtitles」。保存成功 toast；失败错误显示在表单附近。

### 侧栏与路由

在「Prowlarr 搜索」与「下载中」之间增加「字幕」，图标 Lucide `Subtitles`，hash `#subtitles`。

### 字幕页

未配置：提示先在设置中填写四项，按钮「前往设置」。

已配置：标题「字幕」；表单为名称输入、语言下拉、搜索按钮。语言选项：

| 展示 | 值 | 默认 |
|------|-----|------|
| 简体中文 | `zh-CN` | 是 |
| 繁体中文 | `zh-TW` | |
| 英语 | `en` | |
| 日语 | `ja` | |
| 韩语 | `ko` | |

搜索中禁用搜索按钮并显示 loading。无结果给空状态。有结果用表格，列：发行名、语言、文件名、下载次数、评分、操作（下载）。

同一时间只下一份：该行按钮 loading；成功 toast「已保存到 {path}」（`path` 用接口返回值）；失败 toast 接口错误。

不做搜索历史、分页控件、批量下载。

## 测试

TDD：先写失败测试再写实现。

- Store：空配置、`configured=false`；保存后再读回四项；缺字段拒绝且不部分写入
- Client：搜索带上 query、languages、Api-Key、User-Agent；登录拿到 token；下载拿到 link；401 时重登一次再下载
- HTTP：未登录 401；未配置搜索/下载 503；空 query 400；搜索把 OpenSubtitles 数据展平为 items；下载将文件写入临时目录且拒绝带路径的 `file_name`
- 前端：设置面板提交正确 PUT payload；字幕页搜索列出结果并可触发下载；未配置显示「前往设置」
