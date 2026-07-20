# 运行时服务配置 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在设置页编辑五项服务凭据，并让保存后的 Aria2、飞书与 Hook 配置立即生效且可跨重启保留。

**Architecture:** 使用 `settings` 表存储五项专用设置，启动时以数据库中存在的键覆盖环境变量。HTTP Server 和应用 Service 通过读写锁保护可替换的配置快照及客户端；保存先校验和批量写入，成功后一次性换入新的运行时依赖。

**Tech Stack:** Go、net/http、database/sql、go-sqlmock、React、TypeScript、Vitest。

## Global Constraints

- 仅包含 `ARIA2_RPC_URL`、`ARIA2_RPC_SECRET`、`FEISHU_APP_ID`、`FEISHU_APP_SECRET`、`ARIA2_HOOK_SECRET`；不处理 `SESSION_SECRET`。
- 接口使用既有登录鉴权，并按已确认需求返回五项明文。
- 禁止在循环中执行 SQL；五项设置使用一个集合查询和一个批量 upsert。
- 非空 `aria2_rpc_url` 必须是绝对 HTTP/HTTPS URL；其他字段允许空字符串。
- 当前分支修改，提交信息使用中文。

---

## File Structure

- `internal/store/runtime_config.go`：配置读取、批量写入和启动时覆盖。
- `internal/store/runtime_config_test.go`：集合查询与批量 upsert 契约。
- `internal/app/service.go` 及 Aria2/飞书调用文件：热替换客户端。
- `internal/httpapi/runtime_config_handlers.go`：受保护的 GET/PUT 与校验。
- `internal/httpapi/server.go`、`auth.go`、`feishu.go`、`aria2_hook.go`：动态快照读取。
- `cmd/feed-puller/main.go`：迁移后加载数据库覆盖值。
- `web/src/types.ts`、`api.ts`、`App.tsx`：类型、请求和设置面板。

### Task 1: 持久化配置并在启动时覆盖环境变量

**Files:**

- Create: `internal/store/runtime_config.go`
- Create: `internal/store/runtime_config_test.go`
- Modify: `cmd/feed-puller/main.go:48-62`

**Interfaces:**

- Produces: `store.RuntimeServiceConfig`，字段为 `aria2_rpc_url`、`aria2_rpc_secret`、`feishu_app_id`、`feishu_app_secret`、`aria2_hook_secret`。
- Produces: `GetRuntimeServiceConfig(ctx)`、`SaveRuntimeServiceConfig(ctx, cfg)`、`ApplyRuntimeServiceConfig(ctx, base)`。

- [ ] **Step 1: 写入失败测试**

```go
func TestGetRuntimeServiceConfig_UsesSingleBatchQuery(t *testing.T) {
    mock.ExpectQuery(regexp.QuoteMeta(`SELECT name, value FROM settings WHERE name IN (?, ?, ?, ?, ?)`)).
        WithArgs("aria2_rpc_url", "aria2_rpc_secret", "feishu_app_id", "feishu_app_secret", "aria2_hook_secret").
        WillReturnRows(sqlmock.NewRows([]string{"name", "value"}).AddRow("aria2_rpc_url", "https://aria2.test/jsonrpc").AddRow("feishu_app_id", "cli_test"))
    got, err := store.New(db).GetRuntimeServiceConfig(context.Background())
    if err != nil { t.Fatal(err) }
    if got.Aria2RPCURL != "https://aria2.test/jsonrpc" || got.FeishuAppID != "cli_test" { t.Fatalf("got = %+v", got) }
}
```

- [ ] **Step 2: 验证测试失败**

Run: `go test ./internal/store -run TestGetRuntimeServiceConfig_UsesSingleBatchQuery -count=1`

Expected: FAIL，因为类型和方法尚不存在。

- [ ] **Step 3: 实现最小存储层与启动覆盖**

```go
type RuntimeServiceConfig struct {
    Aria2RPCURL string `json:"aria2_rpc_url"`
    Aria2RPCSecret string `json:"aria2_rpc_secret"`
    FeishuAppID string `json:"feishu_app_id"`
    FeishuAppSecret string `json:"feishu_app_secret"`
    Aria2HookSecret string `json:"aria2_hook_secret"`
}
func (s *Store) GetRuntimeServiceConfig(ctx context.Context) (RuntimeServiceConfig, error) {
    rows, err := s.db.QueryContext(ctx, `SELECT name, value FROM settings WHERE name IN (?, ?, ?, ?, ?)`, runtimeServiceConfigKeys...)
    // 将 rows 扫描到 map；缺失键保持未覆盖状态。
}
func (s *Store) SaveRuntimeServiceConfig(ctx context.Context, cfg RuntimeServiceConfig) error {
    _, err := s.db.ExecContext(ctx, `INSERT INTO settings (name, value) VALUES (?, ?), (?, ?), (?, ?), (?, ?), (?, ?) ON DUPLICATE KEY UPDATE value = VALUES(value), updated_at = CURRENT_TIMESTAMP`, runtimeServiceConfigArgs(cfg)...)
    return err
}
```

在 `main.go` 的 `repo.Migrate(ctx)` 后调用 `cfg, err = repo.ApplyRuntimeServiceConfig(ctx, cfg)`，在创建 Aria2、Bot、Scheduler 和 HTTP Server 前处理错误。

- [ ] **Step 4: 验证存储层通过**

Run: `go test ./internal/store -run 'Test(Get|Save)RuntimeServiceConfig' -count=1`

Expected: PASS。

- [ ] **Step 5: 提交**

```bash
git add internal/store/runtime_config.go internal/store/runtime_config_test.go cmd/feed-puller/main.go
git commit -m "支持持久化运行时服务配置"
```

### Task 2: 热替换 Aria2 和飞书依赖

**Files:**

- Modify: `internal/app/service.go:20-39,130,218`
- Modify: `internal/app/service_aria2_gid.go:19,49`
- Modify: `internal/app/service_hook.go:150,168`
- Modify: `internal/app/service_rename_retry.go:259`
- Modify: `internal/app/service_notify.go:75,167-178,415,444-453,476-479`
- Create: `internal/app/service_runtime_config_test.go`

**Interfaces:**

- Produces: `(*Service).SetAria2Client(*downloader.Aria2Client)`、`Aria2Client() *downloader.Aria2Client` 和 `FeishuBot() feishuNotifySender`。

- [ ] **Step 1: 写入失败测试**

```go
func TestService_SetAria2Client_UsesReplacementForSubsequentDownload(t *testing.T) {
    oldAria2 := httptest.NewServer(jsonRPCServer("old-gid"))
    newAria2 := httptest.NewServer(jsonRPCServer("new-gid"))
    defer oldAria2.Close(); defer newAria2.Close()
    svc := app.NewService(repo, downloader.NewAria2Client(oldAria2.URL, ""), log)
    svc.SetAria2Client(downloader.NewAria2Client(newAria2.URL, "new-secret"))
    if err := svc.SubmitItemDownload(context.Background(), item.ID); err != nil { t.Fatal(err) }
    if oldCalls != 0 || newCalls != 1 { t.Fatalf("old=%d new=%d", oldCalls, newCalls) }
}
```

- [ ] **Step 2: 验证测试失败**

Run: `go test ./internal/app -run TestService_SetAria2Client_UsesReplacementForSubsequentDownload -count=1`

Expected: FAIL，因为 `SetAria2Client` 不存在。

- [ ] **Step 3: 实现用锁保护的依赖快照**

```go
type Service struct {
    dependencyMu sync.RWMutex
    aria2 *downloader.Aria2Client
    feishuBot feishuNotifySender
}
func (s *Service) SetAria2Client(client *downloader.Aria2Client) {
    s.dependencyMu.Lock(); s.aria2 = client; s.dependencyMu.Unlock()
}
func (s *Service) Aria2Client() *downloader.Aria2Client {
    s.dependencyMu.RLock(); defer s.dependencyMu.RUnlock(); return s.aria2
}
```

所有 `s.aria2` 与 `s.feishuBot` 调用点先取得局部快照再发起网络请求；`SetFeishuBot` 使用同一把锁。

- [ ] **Step 4: 验证应用层通过**

Run: `go test ./internal/app -count=1`

Expected: PASS。

- [ ] **Step 5: 提交**

```bash
git add internal/app/service.go internal/app/service_aria2_gid.go internal/app/service_hook.go internal/app/service_rename_retry.go internal/app/service_notify.go internal/app/service_runtime_config_test.go
git commit -m "支持热更新 Aria2 与飞书客户端"
```

### Task 3: 实现 HTTP 设置接口和服务端热更新

**Files:**

- Create: `internal/httpapi/runtime_config_handlers.go`
- Create: `internal/httpapi/runtime_config_handlers_test.go`
- Modify: `internal/httpapi/server.go:16-58`
- Modify: `internal/httpapi/auth.go:101-106,178-236`
- Modify: `internal/httpapi/feishu.go:39`
- Modify: `internal/httpapi/aria2_hook.go:61-76`

**Interfaces:**

- Consumes: Task 1 的 Store 方法和 Task 2 的 Service setter。
- Produces: 受 `requireAuth` 包装的 `GET/PUT /api/settings/runtime-config`。
- Produces: `runtimeConfig() config.Config` 和 `replaceRuntimeServiceConfig(store.RuntimeServiceConfig)`。

- [ ] **Step 1: 写入失败处理器测试**

```go
func TestRuntimeConfig_RequiresAuth(t *testing.T) {
    srv := newRuntimeConfigServer(t, config.Config{})
    rec := httptest.NewRecorder()
    srv.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/settings/runtime-config", nil))
    if rec.Code != http.StatusUnauthorized { t.Fatalf("code = %d", rec.Code) }
}
func TestRuntimeConfig_PutRejectsInvalidAria2URL(t *testing.T) {
    srv, mock := newRuntimeConfigServer(t, config.Config{})
    req := authRequest(httptest.NewRequest(http.MethodPut, "/api/settings/runtime-config", strings.NewReader(`{"aria2_rpc_url":"ftp://aria2.test","aria2_rpc_secret":"","feishu_app_id":"","feishu_app_secret":"","aria2_hook_secret":""}`)))
    rec := httptest.NewRecorder(); srv.handleRuntimeServiceConfig(rec, req)
    if rec.Code != http.StatusBadRequest { t.Fatalf("code = %d", rec.Code) }
    if err := mock.ExpectationsWereMet(); err != nil { t.Fatal(err) }
}
```

- [ ] **Step 2: 验证处理器测试失败**

Run: `go test ./internal/httpapi -run TestRuntimeConfig -count=1`

Expected: FAIL，因为路由和处理器尚不存在。

- [ ] **Step 3: 实现路由、校验与热替换**

```go
func (s *Server) handleRuntimeServiceConfig(w http.ResponseWriter, r *http.Request) {
    switch r.Method {
    case http.MethodGet:
        writeJSON(w, http.StatusOK, s.currentRuntimeServiceConfig())
    case http.MethodPut:
        var input store.RuntimeServiceConfig
        if err := json.NewDecoder(r.Body).Decode(&input); err != nil { writeError(w, http.StatusBadRequest, "请求体无效"); return }
        input = normalizeRuntimeServiceConfig(input)
        if err := validateAria2RPCURL(input.Aria2RPCURL); err != nil { writeError(w, http.StatusBadRequest, err.Error()); return }
        if err := s.store.SaveRuntimeServiceConfig(r.Context(), input); err != nil { writeError(w, http.StatusInternalServerError, err.Error()); return }
        s.replaceRuntimeServiceConfig(input)
        writeJSON(w, http.StatusOK, input)
    default: methodNotAllowed(w)
    }
}
```

`Server` 增加 `runtimeMu sync.RWMutex`；替换时仅更新五个字段，并创建新的 `downloader.NewAria2Client` 与 `feishu.NewBotService`。飞书登录选项、OAuth、回调和 Hook 校验均从局部快照读取，避免保存与请求并发读写冲突。

- [ ] **Step 4: 验证 HTTP API 通过**

Run: `go test ./internal/httpapi -count=1`

Expected: PASS。

- [ ] **Step 5: 提交**

```bash
git add internal/httpapi/server.go internal/httpapi/runtime_config_handlers.go internal/httpapi/runtime_config_handlers_test.go internal/httpapi/auth.go internal/httpapi/feishu.go internal/httpapi/aria2_hook.go
git commit -m "提供运行时服务配置接口"
```

### Task 4: 增加设置面板和浏览器测试

**Files:**

- Modify: `web/src/types.ts:1-30`
- Modify: `web/src/api.ts:135-150`
- Modify: `web/src/App.tsx:2521-2608,2796-2804`
- Modify: `web/src/App.test.tsx`

**Interfaces:**

- Produces: `RuntimeServiceConfig` TypeScript 类型。
- Produces: `api.runtimeServiceConfig()` 与 `api.saveRuntimeServiceConfig(payload)`。

- [ ] **Step 1: 写入失败 UI 测试**

```tsx
it('设置页加载并保存运行时服务配置', async () => {
  mockFetch((path, init) => {
    if (path === '/api/settings/runtime-config' && !init?.method) return json(runtimeConfig);
    if (path === '/api/settings/runtime-config' && init?.method === 'PUT') {
      expect(JSON.parse(String(init.body))).toEqual(runtimeConfig);
      return json(runtimeConfig);
    }
  });
  render(<App />);
  await screen.findByDisplayValue('https://aria2.test/jsonrpc');
  await userEvent.click(screen.getByRole('button', { name: '保存运行时服务配置' }));
  expect(await screen.findByText('运行时服务配置已保存并立即生效')).toBeInTheDocument();
});
```

- [ ] **Step 2: 验证 UI 测试失败**

Run: `npm test -- --run web/src/App.test.tsx`

Expected: FAIL，因为没有运行时配置请求和表单。

- [ ] **Step 3: 实现类型、API 与明文表单**

```ts
export type RuntimeServiceConfig = {
  aria2_rpc_url: string;
  aria2_rpc_secret: string;
  feishu_app_id: string;
  feishu_app_secret: string;
  aria2_hook_secret: string;
};
runtimeServiceConfig: () => request<RuntimeServiceConfig>('/api/settings/runtime-config'),
saveRuntimeServiceConfig: (payload: RuntimeServiceConfig) =>
  request<RuntimeServiceConfig>('/api/settings/runtime-config', { method: 'PUT', json: payload }),
```

在 `SettingsView` 添加状态对象和加载请求，再增加保存函数与“运行时服务配置”面板。五个输入框均为明文受控输入；成功时回填响应并提示“运行时服务配置已保存并立即生效”。

- [ ] **Step 4: 验证前端通过**

Run: `npm test -- --run web/src/App.test.tsx && npm run test:types`

Expected: PASS。

- [ ] **Step 5: 提交**

```bash
git add web/src/types.ts web/src/api.ts web/src/App.tsx web/src/App.test.tsx
git commit -m "在设置页配置运行时服务"
```

### Task 5: 完整验证与范围检查

**Files:**

- Modify: `docs/superpowers/specs/2026-07-20-runtime-service-config-design.md` only if verification finds a real discrepancy.

- [ ] **Step 1: 运行完整 Go 验证**

Run: `go test ./internal/store ./internal/app ./internal/httpapi -count=1`

Expected: PASS。

- [ ] **Step 2: 运行完整前端验证**

Run: `npm test && npm run test:types && npm run build`

Expected: PASS。

- [ ] **Step 3: 审查范围与格式**

Run: `git status --short && git diff --check && git diff --stat HEAD`

Expected: 无空白错误；只包含任务 1–4 的文件和必要规格修正，绝不暂存已有的其他 `docs/superpowers/plans/` 文件。

- [ ] **Step 4: 提交验证修正（如有）**

```bash
git add <仅本任务验证阶段修改的文件>
git commit -m "完善运行时服务配置验证"
```

