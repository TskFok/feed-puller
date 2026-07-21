# 下载完成页手动重命名全量开放 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让每条已完成下载均可手动执行重命名，而不受订阅自动 AI 重命名开关限制。

**Architecture:** 完成下载页始终提供同一个既有的重命名 API 操作。服务层仅放宽手动调用的订阅开关校验，自动下载完成时的开关判断维持不变；既有文件定位、AI 识别、重命名历史和错误传播逻辑不变。

**Tech Stack:** React、TypeScript、Vitest、Go、sqlmock。

## Global Constraints

- 默认在当前分支修改，不创建新分支。
- 自动重命名行为继续由 `ai_rename_enabled` 控制。
- 不在循环遍历中查询 SQL。
- 所有 Git 提交信息使用简体中文。

---

### Task 1: 放宽手动重命名服务限制

**Files:**

- Modify: `internal/app/service_rename_retry.go:195-216`
- Test: `internal/app/service_rename_retry_test.go:20-96`

**Interfaces:**

- Consumes: `func (s *Service) RetryCompletedDownloadRename(ctx context.Context, taskID int64) (RenameDownloadResult, error)`。
- Produces: 对 `subscriptions.ai_rename_enabled=false` 的已完成普通订阅也执行现有 `renameDownloadFileAt` 流程。

- [ ] **Step 1: 写出失败的服务层测试**

在 `TestRetryCompletedDownloadRename_Success` 后新增测试，将订阅行中的 `ai_rename_enabled` 值设为 `false`，其余下载任务、AI mock 和更新预期保持与成功用例一致：

```go
func TestRetryCompletedDownloadRename_AllowsDisabledAIRename(t *testing.T) {
	result, err := svc.RetryCompletedDownloadRename(context.Background(), 9)
	if err != nil {
		t.Fatalf("RetryCompletedDownloadRename: %v", err)
	}
	if result.Skipped || result.ToPath != target {
		t.Fatalf("unexpected rename result: %+v", result)
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("renamed file missing: %v", err)
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/app -run '^TestRetryCompletedDownloadRename_AllowsDisabledAIRename$' -count=1`

Expected: FAIL，错误信息包含 `请先在订阅设置中启用 AI 重命名`。

- [ ] **Step 3: 写最小实现**

从 `RetryCompletedDownloadRename` 删除仅用于手动调用的开关拒绝分支，保留订阅读取：

```go
sub, err := s.store.GetSubscription(ctx, task.SubscriptionID)
if err != nil {
	return RenameDownloadResult{}, fmt.Errorf("读取订阅失败: %w", err)
}
```

不要修改 `resolveDownloadFinalPath` 中的 `if !sub.AIRenameEnabled`，使自动流程仍保持原有开关控制。

- [ ] **Step 4: 运行服务层测试确认通过**

Run: `go test ./internal/app -run '^TestRetryCompletedDownloadRename' -count=1`

Expected: PASS。

- [ ] **Step 5: 提交服务层改动**

```bash
git add internal/app/service_rename_retry.go internal/app/service_rename_retry_test.go
git commit -m '允许手动重命名关闭自动开关的下载'
```

### Task 2: 始终显示完成下载的重命名操作

**Files:**

- Modify: `web/src/App.tsx:1515-1571`
- Test: `web/src/App.test.tsx:2171-2217`

**Interfaces:**

- Consumes: `api.retryCompletedDownloadRename(taskId: number)` 及 `CompletedDownload.ai_rename_enabled`。
- Produces: 所有完成下载行中的可点击「重命名」按钮；按钮继续调用 `retryRename(row)`。

- [ ] **Step 1: 写出失败的前端测试**

在「登录后可进入下载完成列表」测试附近新增用例，返回 `ai_rename_enabled: false` 的完成下载，并验证按钮可见、可点击且请求对应重命名接口：

```tsx
it('下载完成列表对关闭自动重命名的条目也提供手动重命名', async () => {
	// mock /api/auth/me、订阅列表、/api/downloads/completed 和 POST /api/downloads/1/rename。
	render(<App />);
	fireEvent.click(await screen.findByRole('button', { name: '下载完成' }));
	const renameButton = await screen.findByRole('button', { name: '重命名' });
	expect(renameButton).toBeEnabled();
	fireEvent.click(renameButton);
	await waitFor(() => expect(fetch).toHaveBeenCalledWith('/api/downloads/1/rename', expect.objectContaining({ method: 'POST' })));
});
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd web && npm test -- App.test.tsx --run`

Expected: FAIL，找不到名称为「重命名」的按钮。

- [ ] **Step 3: 写最小实现**

将完成下载表格操作列由条件渲染改为无条件渲染：

```tsx
<button
	type="button"
	className="ghost"
	disabled={renameBusyId === row.id}
	onClick={() => void retryRename(row)}
>
	{renameBusyId === row.id ? '重命名中…' : '重命名'}
</button>
```

并把页面说明改为「可手动重试刮削重命名」，不再暗示仅启用自动 AI 重命名的订阅可操作。

- [ ] **Step 4: 运行前端测试确认通过**

Run: `cd web && npm test -- App.test.tsx --run`

Expected: PASS。

- [ ] **Step 5: 提交前端改动**

```bash
git add web/src/App.tsx web/src/App.test.tsx
git commit -m '开放下载完成列表的手动重命名'
```

### Task 3: 回归验证

**Files:**

- Verify: `internal/app/service_rename_retry_test.go`
- Verify: `web/src/App.test.tsx`

**Interfaces:**

- Consumes: Tasks 1 和 2 的已提交实现。
- Produces: 后端与前端回归测试通过的验证记录。

- [ ] **Step 1: 执行后端相关测试**

Run: `go test ./internal/app -run 'TestRetryCompletedDownloadRename|TestResolveDownloadFinalPath' -count=1`

Expected: PASS。

- [ ] **Step 2: 执行前端完整测试**

Run: `cd web && npm test -- --run`

Expected: PASS。

- [ ] **Step 3: 检查工作区状态**

Run: `git status --short`

Expected: 除本计划文档外没有未预期改动；实施文件已由前两项提交。
