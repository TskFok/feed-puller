# OpenSubtitles 字幕搜索与下载 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在设置页保存 OpenSubtitles 凭据与字幕目录，并在侧栏「字幕」页按名称/语言搜索后把选中字幕直接写到该目录。

**Architecture:** 配置存 `settings` 表（一次 IN 查询、一次批量 upsert）。后端 `internal/opensubtitles` 代理官方 REST API；`app.Service` 负责登录 token 缓存、落盘与路径消毒。前端只打本机 `/api/settings/opensubtitles`、`/api/subtitles/search`、`/api/subtitles/download`。

**Tech Stack:** Go、net/http、database/sql、go-sqlmock、React、TypeScript、Vitest、Lucide。

## Global Constraints

- 不经过 aria2，不写入下载任务表，不出现在「下载中 / 下载完成」。
- OpenSubtitles HTTP 不使用 RSS 全局代理；超时 30 秒。
- 请求头固定 `User-Agent: feed-puller v1.0`，JSON 接口另加 `Api-Key`；下载接口加 `Authorization: Bearer <token>`。
- 禁止循环内 SQL：配置用单次 `WHERE name IN (...)` 与单次批量 `INSERT ... VALUES (...), (...), (...), (...)`。
- PUT 先校验四项均非空再写库；缺一项则 400 且不执行任何写入。
- 日志和错误文案不得包含密码或 API Key。
- 落盘用 `filepath.Join(download_dir, SanitizeFileName)`，不 `MkdirAll`，同名覆盖。
- 搜索只拉第一页；`languages` 为下拉选中的单个语言码。
- 当前分支修改；提交信息使用中文。
- 本机 Go 测试：`GOOS=darwin GOARCH=arm64 GOCACHE=/private/tmp/feed-puller-gocache-darwin go test <pkg> -run <name> -count=1`。

---

## File Structure

- Create: `internal/store/opensubtitles_config.go` — 配置读写与校验
- Create: `internal/store/opensubtitles_config_test.go`
- Create: `internal/opensubtitles/filename.go` — `SanitizeFileName`
- Create: `internal/opensubtitles/filename_test.go`
- Create: `internal/opensubtitles/flatten.go` — 搜索 JSON 展平
- Create: `internal/opensubtitles/flatten_test.go`
- Create: `internal/opensubtitles/client.go` — Search / Login / RequestDownload / FetchFile
- Create: `internal/opensubtitles/client_test.go`
- Create: `internal/app/service_opensubtitles.go` — 未配置错误、搜索、落盘
- Create: `internal/app/service_opensubtitles_test.go`
- Create: `internal/httpapi/opensubtitles_handlers.go`
- Create: `internal/httpapi/opensubtitles_handlers_test.go`
- Modify: `internal/httpapi/server.go` — 注册路由
- Modify: `README.md` — API 概览
- Modify: `web/src/types.ts`、`web/src/api.ts`
- Create: `web/src/api.opensubtitles.test.ts`
- Create: `web/src/SubtitlesView.tsx`、`web/src/SubtitlesView.test.tsx`
- Modify: `web/src/App.tsx` — 侧栏、hash、设置面板
- Modify: `web/src/App.test.tsx`

### Task 1: 持久化 OpenSubtitles 配置

**Files:**

- Create: `internal/store/opensubtitles_config.go`
- Create: `internal/store/opensubtitles_config_test.go`

**Interfaces:**

- Produces: `store.OpenSubtitlesConfig`，JSON 字段 `username`、`password`、`api_key`、`download_dir`、`configured`。
- Produces: `ErrOpenSubtitlesConfigIncomplete`。
- Produces: `(*Store).GetOpenSubtitlesConfig(ctx) (OpenSubtitlesConfig, error)`。
- Produces: `(*Store).SaveOpenSubtitlesConfig(ctx, cfg) (OpenSubtitlesConfig, error)` — 校验失败不写库，成功后 `Get` 再返回。

- [ ] **Step 1: 写入失败测试**

```go
package store

func TestGetOpenSubtitlesConfig_Empty(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT name, value FROM settings WHERE name IN (?, ?, ?, ?)`)).
		WithArgs("opensubtitles_username", "opensubtitles_password", "opensubtitles_api_key", "opensubtitles_download_dir").
		WillReturnRows(sqlmock.NewRows([]string{"name", "value"}))
	cfg, err := New(db).GetOpenSubtitlesConfig(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Configured || cfg.Username != "" || cfg.APIKey != "" {
		t.Fatalf("got %+v", cfg)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSaveOpenSubtitlesConfig_RejectsIncompleteWithoutWrite(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = New(db).SaveOpenSubtitlesConfig(context.Background(), OpenSubtitlesConfig{
		Username: "u", Password: "p", APIKey: "", DownloadDir: "/data/subtitles",
	})
	if !errors.Is(err, ErrOpenSubtitlesConfigIncomplete) {
		t.Fatalf("err = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSaveOpenSubtitlesConfig_WritesAllKeysThenReadsBack(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO settings (name, value) VALUES (?, ?), (?, ?), (?, ?), (?, ?) ON DUPLICATE KEY UPDATE value = VALUES(value), updated_at = CURRENT_TIMESTAMP`)).
		WithArgs(
			"opensubtitles_username", "alice",
			"opensubtitles_password", "secret",
			"opensubtitles_api_key", "key-1",
			"opensubtitles_download_dir", "/data/subtitles",
		).
		WillReturnResult(sqlmock.NewResult(0, 4))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT name, value FROM settings WHERE name IN (?, ?, ?, ?)`)).
		WithArgs("opensubtitles_username", "opensubtitles_password", "opensubtitles_api_key", "opensubtitles_download_dir").
		WillReturnRows(sqlmock.NewRows([]string{"name", "value"}).
			AddRow("opensubtitles_username", "alice").
			AddRow("opensubtitles_password", "secret").
			AddRow("opensubtitles_api_key", "key-1").
			AddRow("opensubtitles_download_dir", "/data/subtitles"))
	cfg, err := New(db).SaveOpenSubtitlesConfig(context.Background(), OpenSubtitlesConfig{
		Username: " alice ", Password: " secret ", APIKey: " key-1 ", DownloadDir: " /data/subtitles ",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Configured || cfg.Username != "alice" || cfg.DownloadDir != "/data/subtitles" {
		t.Fatalf("got %+v", cfg)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
```

将 `INSERT` 的空白折叠成与实现 SQL 一致（可用 `regexp.QuoteMeta` 包住实现里那一行，或测试里用与实现完全相同的字符串）。`ExpectExec` 的 SQL 必须与实现逐字一致。

- [ ] **Step 2: 验证测试失败**

Run: `GOOS=darwin GOARCH=arm64 GOCACHE=/private/tmp/feed-puller-gocache-darwin go test ./internal/store -run 'Test(Get|Save)OpenSubtitlesConfig' -count=1`

Expected: FAIL，类型或方法不存在。

- [ ] **Step 3: 实现存储层**

```go
package store

const (
	settingOpenSubtitlesUsername    = "opensubtitles_username"
	settingOpenSubtitlesPassword    = "opensubtitles_password"
	settingOpenSubtitlesAPIKey      = "opensubtitles_api_key"
	settingOpenSubtitlesDownloadDir = "opensubtitles_download_dir"
)

var ErrOpenSubtitlesConfigIncomplete = errors.New("请填写 OpenSubtitles 用户名、密码、API Key 和下载目录")

type OpenSubtitlesConfig struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	APIKey      string `json:"api_key"`
	DownloadDir string `json:"download_dir"`
	Configured  bool   `json:"configured"`
}

func (s *Store) GetOpenSubtitlesConfig(ctx context.Context) (OpenSubtitlesConfig, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT name, value FROM settings
		WHERE name IN (?, ?, ?, ?)
	`, settingOpenSubtitlesUsername, settingOpenSubtitlesPassword, settingOpenSubtitlesAPIKey, settingOpenSubtitlesDownloadDir)
	if err != nil {
		return OpenSubtitlesConfig{}, fmt.Errorf("读取 OpenSubtitles 配置失败: %w", err)
	}
	defer rows.Close()
	values := map[string]string{}
	for rows.Next() {
		var name, value string
		if err := rows.Scan(&name, &value); err != nil {
			return OpenSubtitlesConfig{}, fmt.Errorf("读取 OpenSubtitles 配置失败: %w", err)
		}
		values[name] = value
	}
	if err := rows.Err(); err != nil {
		return OpenSubtitlesConfig{}, fmt.Errorf("读取 OpenSubtitles 配置失败: %w", err)
	}
	cfg := OpenSubtitlesConfig{
		Username:    strings.TrimSpace(values[settingOpenSubtitlesUsername]),
		Password:    strings.TrimSpace(values[settingOpenSubtitlesPassword]),
		APIKey:      strings.TrimSpace(values[settingOpenSubtitlesAPIKey]),
		DownloadDir: strings.TrimSpace(values[settingOpenSubtitlesDownloadDir]),
	}
	cfg.Configured = cfg.Username != "" && cfg.Password != "" && cfg.APIKey != "" && cfg.DownloadDir != ""
	return cfg, nil
}

func (s *Store) SaveOpenSubtitlesConfig(ctx context.Context, cfg OpenSubtitlesConfig) (OpenSubtitlesConfig, error) {
	cfg.Username = strings.TrimSpace(cfg.Username)
	cfg.Password = strings.TrimSpace(cfg.Password)
	cfg.APIKey = strings.TrimSpace(cfg.APIKey)
	cfg.DownloadDir = strings.TrimSpace(cfg.DownloadDir)
	if cfg.Username == "" || cfg.Password == "" || cfg.APIKey == "" || cfg.DownloadDir == "" {
		return OpenSubtitlesConfig{}, ErrOpenSubtitlesConfigIncomplete
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO settings (name, value) VALUES (?, ?), (?, ?), (?, ?), (?, ?)
		ON DUPLICATE KEY UPDATE value = VALUES(value), updated_at = CURRENT_TIMESTAMP`,
		settingOpenSubtitlesUsername, cfg.Username,
		settingOpenSubtitlesPassword, cfg.Password,
		settingOpenSubtitlesAPIKey, cfg.APIKey,
		settingOpenSubtitlesDownloadDir, cfg.DownloadDir,
	)
	if err != nil {
		return OpenSubtitlesConfig{}, fmt.Errorf("保存 OpenSubtitles 配置失败: %w", err)
	}
	return s.GetOpenSubtitlesConfig(ctx)
}
```

`Configured = 四项均非空`。查询失败包一层 `读取 OpenSubtitles 配置失败`。扫描 rows 时不要再查库。

- [ ] **Step 4: 验证存储层通过**

Run: `GOOS=darwin GOARCH=arm64 GOCACHE=/private/tmp/feed-puller-gocache-darwin go test ./internal/store -run 'Test(Get|Save)OpenSubtitlesConfig' -count=1`

Expected: PASS。

- [ ] **Step 5: 提交**

```bash
git add internal/store/opensubtitles_config.go internal/store/opensubtitles_config_test.go
git commit -m "$(cat <<'EOF'
支持持久化 OpenSubtitles 配置

EOF
)"
```

### Task 2: OpenSubtitles 客户端（展平、消毒、搜索、登录、下载链接、401 重登）

**Files:**

- Create: `internal/opensubtitles/filename.go`
- Create: `internal/opensubtitles/filename_test.go`
- Create: `internal/opensubtitles/flatten.go`
- Create: `internal/opensubtitles/flatten_test.go`
- Create: `internal/opensubtitles/client.go`
- Create: `internal/opensubtitles/client_test.go`

**Interfaces:**

- Produces: `var APIBaseURL = "https://api.opensubtitles.com/api/v1"`（测试可临时改，用 `t.Cleanup` 还原；这些测试不要 `t.Parallel()`）。
- Produces: `const UserAgent = "feed-puller v1.0"`。
- Produces: `type SubtitleFile struct { FileID int64; FileName, Release, Language string; DownloadCount int; Ratings float64 }`，JSON snake_case。
- Produces: `type DownloadInfo struct { Link, FileName, Message string }`。
- Produces: `SanitizeFileName(name string) (string, error)`。
- Produces: `FlattenSearchData(raw []byte) []SubtitleFile`。
- Produces: `NewClient(username, password, apiKey string) *Client`，内部 `http.Client{Timeout: 30 * time.Second}`，`Proxy` 不设置（默认环境，测试用 `httptest` 直连）。
- Produces: `(*Client).Search(ctx, query, language string) ([]SubtitleFile, error)`。
- Produces: `(*Client).RequestDownload(ctx, fileID int64) (DownloadInfo, error)` — 缓存 token；下载 401 则清 token、登录、再试一次。
- Produces: `(*Client).FetchFile(ctx, link string) ([]byte, error)`。
- Produces: 登录失败返回包含 `OpenSubtitles 登录失败` 的 error（可用 `errors.New`，上层 `errors.Is` 或 `strings.Contains` 均可；HTTP 层用 `errors.Is(err, opensubtitles.ErrLoginFailed)`）。

- [ ] **Step 1: 写入失败测试（filename + flatten + client）**

```go
func TestSanitizeFileName(t *testing.T) {
	t.Parallel()
	got, err := SanitizeFileName("/tmp/nested/foo.srt")
	if err != nil || got != "foo.srt" {
		t.Fatalf("got %q err=%v", got, err)
	}
	if _, err := SanitizeFileName(".."); err == nil {
		t.Fatal("expected error")
	}
	if _, err := SanitizeFileName(" . "); err == nil {
		t.Fatal("expected error")
	}
}

func TestFlattenSearchData_SkipsInvalidFileIDAndSplitsFiles(t *testing.T) {
	t.Parallel()
	raw := []byte(`{"data":[{"attributes":{"language":"zh-CN","download_count":10,"ratings":8.5,"release":"Show.S01","feature_details":{"movie_name":"Show","title":"Ep"},"files":[{"file_id":101,"file_name":"a.srt"},{"file_id":102,"file_name":"b.ass"}]}},{"attributes":{"language":"en","files":[{"file_id":0,"file_name":"skip.srt"}]}}]}`)
	items := FlattenSearchData(raw)
	if len(items) != 2 || items[0].FileID != 101 || items[1].FileName != "b.ass" || items[0].Release != "Show.S01" {
		t.Fatalf("items=%+v", items)
	}
}

func TestClientSearch_SendsQueryLanguageAndHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/subtitles" {
			t.Fatalf("path %s", r.URL.Path)
		}
		if r.URL.Query().Get("query") != "Inception" || r.URL.Query().Get("languages") != "zh-CN" {
			t.Fatalf("query=%s", r.URL.RawQuery)
		}
		if r.Header.Get("Api-Key") != "key-1" || r.Header.Get("User-Agent") != UserAgent {
			t.Fatalf("headers %+v", r.Header)
		}
		_, _ = w.Write([]byte(`{"data":[{"attributes":{"release":"Inception","language":"zh-CN","download_count":1,"ratings":9,"files":[{"file_id":7,"file_name":"inception.srt"}]}}]}`))
	}))
	defer server.Close()
	orig := APIBaseURL
	APIBaseURL = server.URL
	t.Cleanup(func() { APIBaseURL = orig })
	items, err := NewClient("u", "p", "key-1").Search(context.Background(), "Inception", "zh-CN")
	if err != nil || len(items) != 1 || items[0].FileID != 7 {
		t.Fatalf("items=%+v err=%v", items, err)
	}
}

func TestClientRequestDownload_RetriesLoginOn401(t *testing.T) {
	var logins, downloads int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login":
			logins++
			if r.Header.Get("Api-Key") != "key-1" || r.Header.Get("User-Agent") != UserAgent {
				t.Fatalf("login headers %+v", r.Header)
			}
			_, _ = w.Write([]byte(`{"token":"tok","status":200}`))
		case "/download":
			downloads++
			if r.Header.Get("Authorization") != "Bearer tok" {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"message":"invalid token"}`))
				return
			}
			_, _ = w.Write([]byte(`{"link":"http://example/file","file_name":"a.srt"}`))
		default:
			t.Fatalf("path %s", r.URL.Path)
		}
	}))
	defer server.Close()
	orig := APIBaseURL
	APIBaseURL = server.URL
	t.Cleanup(func() { APIBaseURL = orig })
	info, err := NewClient("u", "p", "key-1").RequestDownload(context.Background(), 7)
	if err != nil || info.FileName != "a.srt" || logins != 2 || downloads != 2 {
		t.Fatalf("info=%+v err=%v logins=%d downloads=%d", info, err, logins, downloads)
	}
}
```

`TestClientRequestDownload_RetriesLoginOn401` 的预期：第一次无 token → login（logins=1）→ download 带 Bearer tok 应成功，则 downloads=1、logins=1。若实现先 download 再 login，则 401 重试会使 logins=2、downloads=2。**按 spec「缓存 token；401 再登」**：第一次应先 login 再 download。把断言改成 `logins == 1 && downloads == 1`（先确保 token 再下载）。另写一个测试：先塞过期 token，download 401，再 login + download。

补充测试：

```go
func TestClientRequestDownload_ReloginWhenCachedTokenRejected(t *testing.T) {
	// 第一次 /download 返回 401，随后 /login 200，第二次 /download 200
}
```

实现：`Client.token` 可在测试里不导出；该测试通过「第一次 RequestDownload 成功缓存，把服务端改为拒绝旧 token」较难。改为：服务端前 N 次 download 若 Authorization 为 `Bearer stale` 则 401。Client 无法注入 stale token 除非导出 `SetTokenForTest`。

不要加测试专用 setter。实现 `ensureToken`：无 token 就 login。401 重试测试做成：login 返回 token `tok`，download 第一次无论什么都 401，第二次 200。则 logins≥1、downloads=2。

最终 401 测试断言：`downloads == 2 && logins >= 1 && info.Link != ""`。

再加 `TestClientSearch_EmptyQueryError`：query 空白应在发请求前失败。

- [ ] **Step 2: 验证测试失败**

Run: `GOOS=darwin GOARCH=arm64 GOCACHE=/private/tmp/feed-puller-gocache-darwin go test ./internal/opensubtitles -count=1`

Expected: FAIL，包不存在。

- [ ] **Step 3: 实现客户端**

`SanitizeFileName`：`filepath.Base(strings.TrimSpace(name))`，结果为 `""` / `"."` / `".."` 则 `fmt.Errorf("文件名无效")`。

`FlattenSearchData`：解析 `data[].attributes`；`release` 优先 `attributes.release`，空则 `feature_details.movie_name`，再空则 `feature_details.title`；每个 `files[]` 若 `file_id > 0` 生成一行。

`Client` 字段：`username, password, apiKey string`，`httpClient *http.Client`，`mu sync.Mutex`，`token string`。`NewClient` 里 Timeout=30s。请求 JSON 时设 `Accept: application/json`；login/download POST 设 `Content-Type: application/json`。

`Search`：trim query，空则 `fmt.Errorf("搜索关键词不能为空")`；GET `{APIBaseURL}/subtitles?query=&languages=`。非 2xx：读 body 里 `message`，没有则 `搜索字幕失败`，返回 `fmt.Errorf("%s", msg)`（不要把 api key 写进 error）。

`login`：POST `{APIBaseURL}/login` body `{"username","password"}`。非 2xx 或 token 空：返回 `ErrLoginFailed`（`var ErrLoginFailed = errors.New("OpenSubtitles 登录失败")`）。成功则加锁写入 `c.token`。

`RequestDownload`：`fileID <= 0` 返回 `fmt.Errorf("file_id 无效")`。`ensureToken` → POST `/download` `{"file_id": n}`。401 则清 token、`ensureToken`、再 POST 一次。仍失败用对方 `message` 或 `下载字幕失败`。

`FetchFile`：GET `link`，跟随默认重定向，User-Agent 照设；非 2xx 失败；返回 body bytes。

解析错误 JSON 时忽略无法解码的 body。

- [ ] **Step 4: 验证客户端通过**

Run: `GOOS=darwin GOARCH=arm64 GOCACHE=/private/tmp/feed-puller-gocache-darwin go test ./internal/opensubtitles -count=1`

Expected: PASS。

- [ ] **Step 5: 提交**

```bash
git add internal/opensubtitles
git commit -m "$(cat <<'EOF'
实现 OpenSubtitles 搜索与下载客户端

EOF
)"
```

### Task 3: 应用服务搜索与落盘

**Files:**

- Create: `internal/app/service_opensubtitles.go`
- Create: `internal/app/service_opensubtitles_test.go`

**Interfaces:**

- Consumes: `store.GetOpenSubtitlesConfig` / `SaveOpenSubtitlesConfig`；`opensubtitles.NewClient`、`Search`、`RequestDownload`、`FetchFile`、`SanitizeFileName`、`APIBaseURL`。
- Produces: `var ErrOpenSubtitlesNotConfigured = errors.New("OpenSubtitles 未配置")`。
- Produces: `(*Service).GetOpenSubtitlesConfig` / `SaveOpenSubtitlesConfig`（转调 store）。
- Produces: `(*Service).SearchSubtitles(ctx, query, language string) ([]opensubtitles.SubtitleFile, error)`。
- Produces: `(*Service).DownloadSubtitle(ctx, fileID int64, fallbackFileName string) (path string, fileName string, err error)`。

- [ ] **Step 1: 写入失败测试**

```go
func TestSearchSubtitles_NotConfigured(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil { t.Fatal(err) }
	defer db.Close()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT name, value FROM settings WHERE name IN (?, ?, ?, ?)`)).
		WithArgs("opensubtitles_username", "opensubtitles_password", "opensubtitles_api_key", "opensubtitles_download_dir").
		WillReturnRows(sqlmock.NewRows([]string{"name", "value"}))
	svc := app.NewService(store.New(db), downloader.NewAria2Client("", ""), slog.Default())
	_, err = svc.SearchSubtitles(context.Background(), "Inception", "zh-CN")
	if !errors.Is(err, app.ErrOpenSubtitlesNotConfigured) {
		t.Fatalf("err=%v", err)
	}
}

func TestDownloadSubtitle_WritesSanitizedFile(t *testing.T) {
	dir := t.TempDir()
	var sawDownload bool
	osAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login":
			_, _ = w.Write([]byte(`{"token":"tok"}`))
		case "/download":
			sawDownload = true
			fmt.Fprintf(w, `{"link":"%s/payload.srt","file_name":"/etc/../shown.srt"}`, "http://"+r.Host)
		case "/payload.srt":
			_, _ = w.Write([]byte("subtitle-bytes"))
		default:
			t.Fatalf("path %s", r.URL.Path)
		}
	}))
	defer osAPI.Close()
	orig := opensubtitles.APIBaseURL
	opensubtitles.APIBaseURL = osAPI.URL
	t.Cleanup(func() { opensubtitles.APIBaseURL = orig })

	db, mock, err := sqlmock.New()
	if err != nil { t.Fatal(err) }
	defer db.Close()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT name, value FROM settings WHERE name IN (?, ?, ?, ?)`)).
		WithArgs("opensubtitles_username", "opensubtitles_password", "opensubtitles_api_key", "opensubtitles_download_dir").
		WillReturnRows(sqlmock.NewRows([]string{"name", "value"}).
			AddRow("opensubtitles_username", "u").
			AddRow("opensubtitles_password", "p").
			AddRow("opensubtitles_api_key", "k").
			AddRow("opensubtitles_download_dir", dir))
	svc := app.NewService(store.New(db), downloader.NewAria2Client("", ""), slog.Default())
	path, name, err := svc.DownloadSubtitle(context.Background(), 9, "fallback.srt")
	if err != nil { t.Fatal(err) }
	if name != "shown.srt" || path != filepath.Join(dir, "shown.srt") || !sawDownload {
		t.Fatalf("path=%s name=%s", path, name)
	}
	body, _ := os.ReadFile(path)
	if string(body) != "subtitle-bytes" {
		t.Fatalf("body=%s", body)
	}
}

func TestDownloadSubtitle_RejectsEmptySanitizedName(t *testing.T) {
	// OS download 返回 file_name:".." 且 fallback 也是 ".."，期望 error，且 temp dir 内无文件
}
```

`TestDownloadSubtitle_WritesSanitizedFile` 里 `link` 必须是该 httptest 的 URL。用 `osAPI.URL + "/payload.srt"` 写进 JSON。

再加：`fileID==0` 在发网络前失败；目录不存在时 `WriteFile` 失败且 error 含「目录」或底层 path error（包一层 `保存字幕失败`）。

- [ ] **Step 2: 验证测试失败**

Run: `GOOS=darwin GOARCH=arm64 GOCACHE=/private/tmp/feed-puller-gocache-darwin go test ./internal/app -run 'Test(Search|Download)Subtitle' -count=1`

Expected: FAIL，方法不存在。

- [ ] **Step 3: 实现 Service**

```go
func (s *Service) SearchSubtitles(ctx context.Context, query, language string) ([]opensubtitles.SubtitleFile, error) {
	cfg, err := s.store.GetOpenSubtitlesConfig(ctx)
	if err != nil { return nil, err }
	if !cfg.Configured { return nil, ErrOpenSubtitlesNotConfigured }
	return opensubtitles.NewClient(cfg.Username, cfg.Password, cfg.APIKey).Search(ctx, query, language)
}

func (s *Service) DownloadSubtitle(ctx context.Context, fileID int64, fallbackFileName string) (string, string, error) {
	cfg, err := s.store.GetOpenSubtitlesConfig(ctx)
	if err != nil { return "", "", err }
	if !cfg.Configured { return "", "", ErrOpenSubtitlesNotConfigured }
	client := opensubtitles.NewClient(cfg.Username, cfg.Password, cfg.APIKey)
	info, err := client.RequestDownload(ctx, fileID)
	if err != nil { return "", "", err }
	name := strings.TrimSpace(info.FileName)
	if name == "" { name = fallbackFileName }
	sanitized, err := opensubtitles.SanitizeFileName(name)
	if err != nil { return "", "", err }
	dest := filepath.Join(cfg.DownloadDir, sanitized)
	body, err := client.FetchFile(ctx, info.Link)
	if err != nil { return "", "", err }
	if err := os.WriteFile(dest, body, 0o644); err != nil {
		return "", "", fmt.Errorf("保存字幕失败: %w", err)
	}
	return dest, sanitized, nil
}
```

不要 `MkdirAll`。不要调用 aria2。同一进程内 `Client` 每次 `NewClient` 会丢掉 token 缓存；spec 要求进程内缓存。因此 Service 持有：

```go
type Service struct {
	// 现有字段...
	opensubtitlesMu     sync.Mutex
	opensubtitlesClient *opensubtitles.Client
	opensubtitlesUser   string
	opensubtitlesPass   string
	opensubtitlesKey    string
}
```

`opensubtitlesClientFor(cfg)`：锁内若 username/password/apiKey 与缓存一致则复用 client，否则 `NewClient` 并记下凭据。测试 401 重登依赖同一 client 实例，下载路径会复用。

- [ ] **Step 4: 验证服务层通过**

Run: `GOOS=darwin GOARCH=arm64 GOCACHE=/private/tmp/feed-puller-gocache-darwin go test ./internal/app -run 'Test(Search|Download)Subtitle' -count=1`

Expected: PASS。随后 `go test ./internal/app -count=1` 确认无回归。

- [ ] **Step 5: 提交**

```bash
git add internal/app/service_opensubtitles.go internal/app/service_opensubtitles_test.go
git commit -m "$(cat <<'EOF'
通过服务层搜索字幕并写入配置目录

EOF
)"
```

### Task 4: HTTP API

**Files:**

- Create: `internal/httpapi/opensubtitles_handlers.go`
- Create: `internal/httpapi/opensubtitles_handlers_test.go`
- Modify: `internal/httpapi/server.go`（在 Prowlarr 路由附近注册）
- Modify: `README.md` API 概览列表

**Interfaces:**

- Consumes: Service 方法与 `store.ErrOpenSubtitlesConfigIncomplete`、`app.ErrOpenSubtitlesNotConfigured`、`opensubtitles.ErrLoginFailed`。
- Produces: `GET/PUT /api/settings/opensubtitles`；`GET /api/subtitles/search?query=&languages=`；`POST /api/subtitles/download` body `{"file_id":number,"file_name":string}`。
- 均 `requireAuth`。搜索 `items` 为 `[]opensubtitles.SubtitleFile`；下载成功 `{"path","file_name"}`。

- [ ] **Step 1: 写入失败测试**

仿 `newProwlarrServer` / `authRequest`：

```go
func TestOpenSubtitlesSettingRequiresAuth(t *testing.T) {
	srv, _, cleanup := newOpenSubtitlesServer(t)
	defer cleanup()
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/settings/opensubtitles", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("code=%d", rec.Code)
	}
}

func TestOpenSubtitlesSearch_NotConfigured(t *testing.T) {
	srv, mock, cleanup := newOpenSubtitlesServer(t)
	defer cleanup()
	expectEmptyOpenSubtitlesSettings(mock)
	req := authRequest(httptest.NewRequest(http.MethodGet, "/api/subtitles/search?query=Inception&languages=zh-CN", nil))
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestOpenSubtitlesSearch_EmptyQuery(t *testing.T) {
	// 已配置 sqlmock；query 空 → 400。不要 Expect OpenSubtitles 网络。
}

func TestOpenSubtitlesSettingPut_Incomplete(t *testing.T) {
	// PUT {"username":"u"} → 400，mock 无 Exec
}

func TestOpenSubtitlesSearch_FlattensItems(t *testing.T) {
	osAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/subtitles" {
			t.Fatalf("path %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"data":[{"attributes":{"release":"Inception","language":"zh-CN","download_count":1,"ratings":9,"files":[{"file_id":7,"file_name":"a.srt"}]}}]}`))
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
	if rec.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
	var payload struct {
		Items []opensubtitles.SubtitleFile `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Items) != 1 || payload.Items[0].FileID != 7 {
		t.Fatalf("payload=%+v", payload)
	}
}

func TestOpenSubtitlesDownload_UsesFallbackBaseName(t *testing.T) {
	dir := t.TempDir()
	var osAPI *httptest.Server
	osAPI = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login":
			_, _ = w.Write([]byte(`{"token":"tok"}`))
		case "/download":
			fmt.Fprintf(w, `{"link":"%s/payload.srt","file_name":""}`, osAPI.URL)
		case "/payload.srt":
			_, _ = w.Write([]byte("ok"))
		default:
			t.Fatalf("path %s", r.URL.Path)
		}
	}))
	defer osAPI.Close()
	orig := opensubtitles.APIBaseURL
	opensubtitles.APIBaseURL = osAPI.URL
	t.Cleanup(func() { opensubtitles.APIBaseURL = orig })
	srv, mock, cleanup := newOpenSubtitlesServer(t)
	defer cleanup()
	expectConfiguredOpenSubtitlesSettings(mock, dir)
	req := authRequest(httptest.NewRequest(http.MethodPost, "/api/subtitles/download", strings.NewReader(`{"file_id":9,"file_name":"../../etc/passwd"}`)))
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
	body, err := os.ReadFile(filepath.Join(dir, "passwd"))
	if err != nil || string(body) != "ok" {
		t.Fatalf("file err=%v body=%s", err, body)
	}
}

func TestOpenSubtitlesDownload_RejectsDotDotFileName(t *testing.T) {
	dir := t.TempDir()
	osAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/login" {
			_, _ = w.Write([]byte(`{"token":"tok"}`))
			return
		}
		if r.URL.Path == "/download" {
			_, _ = w.Write([]byte(`{"link":"http://example/x","file_name":""}`))
			return
		}
		t.Fatalf("path %s", r.URL.Path)
	}))
	defer osAPI.Close()
	orig := opensubtitles.APIBaseURL
	opensubtitles.APIBaseURL = osAPI.URL
	t.Cleanup(func() { opensubtitles.APIBaseURL = orig })
	srv, mock, cleanup := newOpenSubtitlesServer(t)
	defer cleanup()
	expectConfiguredOpenSubtitlesSettings(mock, dir)
	req := authRequest(httptest.NewRequest(http.MethodPost, "/api/subtitles/download", strings.NewReader(`{"file_id":9,"file_name":".."}`)))
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
}
```

`newOpenSubtitlesServer` 与 `newProwlarrServer` 相同构造。`expectEmptyOpenSubtitlesSettings` 匹配 Task 1 的 IN 查询。

下载 400 测试：`file_id` 缺省/0 → 400。登录失败：OS `/login` 401 → 502，body error 为 `OpenSubtitles 登录失败`。

路由必须走 `ServeHTTP`（经 `requireAuth`），不要只调未包鉴权的 handler（与 runtime-config 测试一致）。

- [ ] **Step 2: 验证测试失败**

Run: `GOOS=darwin GOARCH=arm64 GOCACHE=/private/tmp/feed-puller-gocache-darwin go test ./internal/httpapi -run OpenSubtitles -count=1`

Expected: FAIL，404 或方法不存在。

- [ ] **Step 3: 实现 handler 与路由**

`server.go` 增加：

```go
mux.HandleFunc("/api/settings/opensubtitles", server.requireAuth(server.handleOpenSubtitlesSetting))
mux.HandleFunc("/api/subtitles/search", server.requireAuth(server.handleSubtitlesSearch))
mux.HandleFunc("/api/subtitles/download", server.requireAuth(server.handleSubtitlesDownload))
```

Handler 状态映射：

- `ErrOpenSubtitlesNotConfigured` → 503
- `store.ErrOpenSubtitlesConfigIncomplete`、空 query、`file_id<=0`、`SanitizeFileName` 失败 → 400
- `opensubtitles.ErrLoginFailed` 及其余上游 API/网络错误 → 502
- 写文件/读库失败 → 500
- GET 设置 500 仅当 store 出错

搜索：`query := strings.TrimSpace(r.URL.Query().Get("query"))`，空则 400 `"query 不能为空"`。`languages` 原样 trim 传给 Service。

下载：decode `file_id`、`file_name`；成功 `writeJSON(200, map[string]string{"path": path, "file_name": fileName})`。

README「API 概览」追加上述四条。

- [ ] **Step 4: 验证 HTTP 测试通过**

Run: `GOOS=darwin GOARCH=arm64 GOCACHE=/private/tmp/feed-puller-gocache-darwin go test ./internal/httpapi -run OpenSubtitles -count=1`

Expected: PASS。再跑 `go test ./internal/httpapi -count=1`。

- [ ] **Step 5: 提交**

```bash
git add internal/httpapi/opensubtitles_handlers.go internal/httpapi/opensubtitles_handlers_test.go internal/httpapi/server.go README.md
git commit -m "$(cat <<'EOF'
暴露 OpenSubtitles 设置与字幕搜索下载接口

EOF
)"
```

### Task 5: 前端 API 客户端

**Files:**

- Modify: `web/src/types.ts`
- Modify: `web/src/api.ts`
- Create: `web/src/api.opensubtitles.test.ts`

**Interfaces:**

- Produces: `OpenSubtitlesConfig`、`SubtitleSearchItem`、`SubtitleSearchResult`、`SubtitleDownloadResult`。
- Produces: `api.openSubtitlesConfig`、`saveOpenSubtitlesConfig`、`searchSubtitles(query, languages)`、`downloadSubtitle({file_id, file_name})`。

- [ ] **Step 1: 写入失败测试**

```ts
import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { api } from './api';

describe('api opensubtitles', () => {
  afterEach(() => vi.unstubAllGlobals());
  it('searchSubtitles 带 query 与 languages', async () => {
    vi.stubGlobal('fetch', vi.fn(async (input: RequestInfo | URL) => {
      const path = String(input);
      expect(path).toContain('/api/subtitles/search?');
      expect(path).toContain('query=Inception');
      expect(path).toContain('languages=zh-CN');
      return new Response(JSON.stringify({ items: [{ file_id: 7, file_name: 'a.srt', release: 'Inception', language: 'zh-CN', download_count: 1, ratings: 9 }] }), {
        status: 200, headers: { 'Content-Type': 'application/json' }
      });
    }));
    const res = await api.searchSubtitles('Inception', 'zh-CN');
    expect(res.items[0]?.file_id).toBe(7);
  });
  it('downloadSubtitle POST file_id 与 file_name', async () => {
    vi.stubGlobal('fetch', vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      expect(String(input)).toBe('/api/subtitles/download');
      expect(init?.method).toBe('POST');
      expect(JSON.parse(String(init?.body))).toEqual({ file_id: 7, file_name: 'a.srt' });
      return new Response(JSON.stringify({ path: '/data/subtitles/a.srt', file_name: 'a.srt' }), {
        status: 200, headers: { 'Content-Type': 'application/json' }
      });
    }));
    const res = await api.downloadSubtitle({ file_id: 7, file_name: 'a.srt' });
    expect(res.path).toBe('/data/subtitles/a.srt');
  });
  it('saveOpenSubtitlesConfig PUT 四项', async () => {
    vi.stubGlobal('fetch', vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      expect(String(input)).toBe('/api/settings/opensubtitles');
      expect(init?.method).toBe('PUT');
      return new Response(JSON.stringify({ username: 'u', password: 'p', api_key: 'k', download_dir: '/d', configured: true }), {
        status: 200, headers: { 'Content-Type': 'application/json' }
      });
    }));
    const saved = await api.saveOpenSubtitlesConfig({ username: 'u', password: 'p', api_key: 'k', download_dir: '/d' });
    expect(saved.configured).toBe(true);
  });
});
```

- [ ] **Step 2: 验证测试失败**

Run: `npm test -- --run api.opensubtitles`

Expected: FAIL，方法不存在。

- [ ] **Step 3: 加类型与 api 方法**

```ts
export type OpenSubtitlesConfig = {
  username: string;
  password: string;
  api_key: string;
  download_dir: string;
  configured: boolean;
};

export type SubtitleSearchItem = {
  file_id: number;
  file_name: string;
  release: string;
  language: string;
  download_count: number;
  ratings: number;
};
```

```ts
openSubtitlesConfig: () => request<OpenSubtitlesConfig>('/api/settings/opensubtitles'),
saveOpenSubtitlesConfig: (payload: Pick<OpenSubtitlesConfig, 'username' | 'password' | 'api_key' | 'download_dir'>) =>
  request<OpenSubtitlesConfig>('/api/settings/opensubtitles', { method: 'PUT', json: payload }),
searchSubtitles: (query: string, languages: string) => {
  const params = new URLSearchParams({ query, languages });
  return request<{ items: SubtitleSearchItem[] }>(`/api/subtitles/search?${params.toString()}`);
},
downloadSubtitle: (payload: { file_id: number; file_name: string }) =>
  request<{ path: string; file_name: string }>('/api/subtitles/download', { method: 'POST', json: payload }),
```

记得在 `api.ts` 顶部 import 类型。

- [ ] **Step 4: 验证 api 测试通过**

Run: `npm test -- --run api.opensubtitles`

Expected: PASS。

- [ ] **Step 5: 提交**

```bash
git add web/src/types.ts web/src/api.ts web/src/api.opensubtitles.test.ts
git commit -m "$(cat <<'EOF'
添加 OpenSubtitles 前端 API 封装

EOF
)"
```

### Task 6: 设置页 OpenSubtitles 面板

**Files:**

- Modify: `web/src/App.tsx`（`SettingsView`）
- Modify: `web/src/App.test.tsx`

**Interfaces:**

- Consumes: `api.openSubtitlesConfig` / `saveOpenSubtitlesConfig`。
- 面板在 Prowlarr 表单之后。密码 `type=password`，按钮 `aria-label` 为「显示密码」/「隐藏密码」。保存失败把 `messageOf(err)` 放在表单内 `role="alert"`；成功 toast「OpenSubtitles 设置已保存」。

- [ ] **Step 1: 写入失败测试（追加到 App.test.tsx）**

仿「设置页加载并保存运行时服务配置」：`hash = '#settings'`，stub `/api/settings/opensubtitles` GET 回填、PUT 捕获 body。

```ts
it('设置页加载并保存 OpenSubtitles 配置', async () => {
  window.location.hash = '#settings';
  const config = { username: 'alice', password: 'secret', api_key: 'key-1', download_dir: '/data/subtitles', configured: true };
  let saved: unknown;
  // fetch stub：me、options、opensubtitles GET/PUT、其余 {}
  render(<App />);
  await screen.findByDisplayValue('alice');
  fireEvent.change(screen.getByLabelText('API Key'), { target: { value: 'key-2' } });
  // 页面上 Prowlarr 也有「API Key」。用 OpenSubtitles 面板 scope：
  const panel = screen.getByRole('heading', { name: 'OpenSubtitles' }).closest('form');
  expect(panel).toBeTruthy();
  fireEvent.change(within(panel!).getByLabelText('API Key'), { target: { value: 'key-2' } });
  fireEvent.click(within(panel!).getByRole('button', { name: '保存 OpenSubtitles' }));
  await waitFor(() => expect(saved).toEqual({ username: 'alice', password: 'secret', api_key: 'key-2', download_dir: '/data/subtitles' }));
  expect(await screen.findByText('OpenSubtitles 设置已保存')).toBeInTheDocument();
});
```

密码框 `getByLabelText('密码')` 可能与登录页冲突；设置页测试在已登录后，登录表单不在。仍用 `within(panel).getByLabelText('密码')`。

若 PUT 失败：stub 400 `{error:'请填写...'}`，期望 `getByRole('alert')` 含该文案。可与成功用例分开写。

- [ ] **Step 2: 验证测试失败**

Run: `npm test -- --run App.test`

Expected: FAIL，找不到 heading OpenSubtitles。

- [ ] **Step 3: 实现设置面板**

`SettingsView` 增加四项 state，`useEffect` 里 `api.openSubtitlesConfig()` 回填。表单：

- 用户名、密码（旁按钮切换 type）、API Key、字幕下载目录 placeholder `/data/subtitles`
- `openSubtitlesError` 字符串；submit 先 `setOpenSubtitlesError('')`，catch 设 alert
- 成功 `showToast('OpenSubtitles 设置已保存')` 并用返回值回填

密码切换用 Lucide `Eye` / `EyeOff`，按钮 `type="button"`。

label 包裹 input，保证 `getByLabelText` 可用。Prowlarr 的「API Key」label 文案保持不变；OpenSubtitles 的 API Key 也叫「API Key」，测试必须 `within(form)`。

- [ ] **Step 4: 验证设置测试通过**

Run: `npm test -- --run App.test`

Expected: PASS。

- [ ] **Step 5: 提交**

```bash
git add web/src/App.tsx web/src/App.test.tsx
git commit -m "$(cat <<'EOF'
在设置页增加 OpenSubtitles 配置

EOF
)"
```

### Task 7: 侧栏「字幕」页

**Files:**

- Create: `web/src/SubtitlesView.tsx`
- Create: `web/src/SubtitlesView.test.tsx`
- Modify: `web/src/App.tsx` — `Tab` 增加 `'subtitles'`；`APP_TABS`；侧栏在 Prowlarr 与「下载中」之间；`tab === 'subtitles' && <SubtitlesView onGoSettings={() => selectTab('settings')} />`
- Modify: `web/src/App.test.tsx` — 点击「字幕」后 `location.hash === '#subtitles'`

**Interfaces:**

- Consumes: `api.openSubtitlesConfig`、`searchSubtitles`、`downloadSubtitle`。
- 语言默认 `zh-CN`；选项：简体中文 / 繁体中文 / 英语 / 日语 / 韩语。
- 未配置：文案说明先填四项 +「前往设置」。
- 表格列：发行名、语言、文件名、下载次数、评分、操作。行 `key={file_id}`。
- 同一时间 `downloadingFileId`；成功 toast `已保存到 ${path}`。

- [ ] **Step 1: 写入失败测试**

`SubtitlesView.test.tsx`：

```tsx
it('未配置时显示前往设置', async () => {
  vi.stubGlobal('fetch', vi.fn(async (input: RequestInfo | URL) => {
    if (String(input) === '/api/settings/opensubtitles') {
      return new Response(JSON.stringify({ username: '', password: '', api_key: '', download_dir: '', configured: false }), {
        status: 200, headers: { 'Content-Type': 'application/json' }
      });
    }
    return new Response('{}', { status: 200 });
  }));
  const onGoSettings = vi.fn();
  render(<ToastProvider><SubtitlesView onGoSettings={onGoSettings} /></ToastProvider>);
  fireEvent.click(await screen.findByRole('button', { name: '前往设置' }));
  expect(onGoSettings).toHaveBeenCalled();
});

it('搜索列出结果并下载', async () => {
  vi.stubGlobal('fetch', vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const path = String(input);
    if (path === '/api/settings/opensubtitles') {
      return new Response(JSON.stringify({ username: 'u', password: 'p', api_key: 'k', download_dir: '/data/subtitles', configured: true }), {
        status: 200, headers: { 'Content-Type': 'application/json' }
      });
    }
    if (path.startsWith('/api/subtitles/search?')) {
      expect(path).toContain('query=Inception');
      expect(path).toContain('languages=zh-CN');
      return new Response(JSON.stringify({
        items: [{ file_id: 7, file_name: 'a.srt', release: 'Inception.2024', language: 'zh-CN', download_count: 12, ratings: 8.5 }]
      }), { status: 200, headers: { 'Content-Type': 'application/json' } });
    }
    if (path === '/api/subtitles/download' && init?.method === 'POST') {
      expect(JSON.parse(String(init.body))).toEqual({ file_id: 7, file_name: 'a.srt' });
      return new Response(JSON.stringify({ path: '/data/subtitles/a.srt', file_name: 'a.srt' }), {
        status: 200, headers: { 'Content-Type': 'application/json' }
      });
    }
    return new Response('{}', { status: 200 });
  }));
  render(<ToastProvider><SubtitlesView /></ToastProvider>);
  await screen.findByRole('heading', { name: '字幕' });
  fireEvent.change(screen.getByLabelText('名称'), { target: { value: 'Inception' } });
  fireEvent.click(screen.getByRole('button', { name: '搜索' }));
  expect(await screen.findByText('Inception.2024')).toBeInTheDocument();
  fireEvent.click(screen.getByRole('button', { name: '下载' }));
  expect(await screen.findByText('已保存到 /data/subtitles/a.srt')).toBeInTheDocument();
});
```

`App.test.tsx` 增加：登录后点击侧栏「字幕」，`expect(window.location.hash).toBe('#subtitles')`。fetch stub 需覆盖 `openSubtitlesConfig`（未配置即可）。

- [ ] **Step 2: 验证测试失败**

Run: `npm test -- --run SubtitlesView App.test`

Expected: FAIL，组件或 nav 不存在。

- [ ] **Step 3: 实现页面与导航**

`SubtitlesView`：加载 config；`!configured` 早退。表单 `panel`：名称 input、语言 select、搜索 button（`disabled={searching}`，loading 时 `Loader2` + `icon-spinning`）。错误 `role="alert"`。无结果：搜过显示「没有找到字幕」，未搜显示「输入名称后搜索」。结果用现有 `table-wrap` + `table`。下载按钮 `disabled={downloadingFileId != null}`，当前行可显示「下载中…」。

`App.tsx`：`import { Subtitles } from 'lucide-react'` 与 `SubtitlesView`。

```ts
type Tab = 'subscriptions' | 'prowlarr' | 'subtitles' | 'active' | ...
```

```tsx
<NavButton tab="subtitles" ... icon={<Subtitles size={18} />} label="字幕" />
```

放在 Prowlarr 下一行。workspace 内 `tab === 'subtitles' && <SubtitlesView onGoSettings={() => selectTab('settings')} />`。

视觉沿用玻璃面板，不加新风格、不用 emoji 当图标。

- [ ] **Step 4: 验证前端测试通过**

Run: `npm test -- --run SubtitlesView App.test api.opensubtitles`

Expected: PASS。再跑 `npm test` 全量。

Go：`GOOS=darwin GOARCH=arm64 GOCACHE=/private/tmp/feed-puller-gocache-darwin go test ./internal/store ./internal/opensubtitles ./internal/app ./internal/httpapi -count=1`

- [ ] **Step 5: 提交**

```bash
git add web/src/SubtitlesView.tsx web/src/SubtitlesView.test.tsx web/src/App.tsx web/src/App.test.tsx
git commit -m "$(cat <<'EOF'
新增字幕搜索页并接入侧栏

EOF
)"
```

---

## Self-Review

**Spec coverage**

- 四项 settings 键、批量读写、缺项不写库 → Task 1
- User-Agent、Api-Key、搜索 query/languages、第一页、files 展平、file_id≤0 丢弃 → Task 2
- login、token 缓存、401 重登、download link GET、SanitizeFileName、Join、不 MkdirAll、不 aria2 → Task 2–3
- HTTP 401/400/503/502、snake_case、search items、download path → Task 4
- 设置面板、密码显示隐藏、侧栏、hash、语言默认 zh-CN、表格、单任务下载 toast → Task 6–7
- 前端 API 封装测试 → Task 5
- 非目标（历史/分页/批量/季集/浏览器下载）无对应任务

**Placeholder scan:** 无 TBD；401 测试断言已定为「先 login 再 download，缓存被拒时 download 两次」。

**Type consistency:** `OpenSubtitlesConfig` 前后端字段一致；`SubtitleFile` / `SubtitleSearchItem` 均为 `file_id`、`file_name`、`release`、`language`、`download_count`、`ratings`。
