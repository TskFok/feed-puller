# 订阅弹窗勾选项命中范围 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让新增和编辑订阅弹窗中的单选、复选项仅在控件与相邻文字区域响应点击，避免行内留白造成误触。

**Architecture:** 在 `SubscriptionModal` 中为每个单选或复选控件生成稳定的 DOM `id`，并使用紧凑容器包裹原生输入控件与通过 `htmlFor` 关联的文字标签。局部 CSS 只作用于该容器，保留原生控件、键盘操作和可访问名称，且不改变通用 `.check` 样式。

**Tech Stack:** React、TypeScript、Vitest、Playwright、CSS、Vite。

## Global Constraints

- 仅调整 `SubscriptionModal` 内的固定间隔、Crontab、启用此订阅、使用代理服务器拉取和启用 AI 重命名。
- 控件及其相邻文字可切换；同行剩余空白区域不可切换。
- 不修改后端接口、数据模型、通用 `.check` 样式或其他页面控件。
- 保留现有可访问名称、Tab 焦点与空格键操作。

---

### Task 1: 用浏览器测试锁定紧凑命中区域

**Files:**
- Modify: `e2e/subscription-layout.spec.ts`

**Interfaces:**
- Consumes: 新增订阅按钮的可访问名称 `新增订阅`，以及订阅弹窗中的复选框可访问名称 `启用此订阅`。
- Produces: 对 `.modal-check-option` 的浏览器布局与点击行为回归测试。

- [x] **Step 1: 写入失败的命中范围测试**

在 `e2e/subscription-layout.spec.ts` 追加测试，并在文件中已有的登录/API mock helper 后调用它们：

```ts
test('订阅弹窗勾选项不会由行内留白切换', async ({ page }) => {
  await resetClientState(page);
  await mockLoggedIn(page);
  await page.goto('/');
  await page.getByRole('button', { name: '新增订阅' }).click();

  const checkbox = page.getByRole('checkbox', { name: '启用此订阅' });
  const option = checkbox.locator('xpath=..');
  await expect(option).toHaveClass(/modal-check-option/);
  await expect(checkbox).toBeChecked();

  const box = await option.boundingBox();
  expect(box).not.toBeNull();
  await page.mouse.click(box!.x + box!.width + 24, box!.y + box!.height / 2);
  await expect(checkbox).toBeChecked();

  await page.getByText('启用此订阅', { exact: true }).click();
  await expect(checkbox).not.toBeChecked();
});
```

- [x] **Step 2: 运行测试，确认在当前整行标签实现中失败**

运行：

```bash
npm run test:e2e -- e2e/subscription-layout.spec.ts
```

预期：失败，原因是控件父元素仍为 `label.check`，尚未具有 `modal-check-option` 紧凑容器类。

- [x] **Step 3: 校验失败原因**

确认失败发生在 `toHaveClass(/modal-check-option/)`，而非登录、API mock 或元素定位失败。

### Task 2: 以紧凑容器实现订阅弹窗控件

**Files:**
- Modify: `web/src/App.tsx:597-860`
- Modify: `web/src/styles.css:1497-1499`

**Interfaces:**
- Consumes: `useId`、`scheduleKind`、`draft.enabled`、`draft.use_proxy` 与 `draft.ai_rename_enabled` 的既有状态和更新回调。
- Produces: `.modal-check-option` 局部容器；每个输入与其文字标签通过唯一 `id`/`htmlFor` 关联。

- [x] **Step 1: 为五个输入创建标识符**

在 `SubscriptionModal` 中、`const titleId = useId();` 后新增：

```tsx
const intervalScheduleId = useId();
const cronScheduleId = useId();
const enabledId = useId();
const useProxyId = useId();
const aiRenameEnabledId = useId();
```

- [x] **Step 2: 将单选项改为输入与文字标签的紧凑容器**

将两个 `label.check.modal-check-inline` 改为以下形状（`cron` 选项同理，替换为 `cronScheduleId`、`Crontab` 及原回调）：

```tsx
<div className="modal-check-option">
  <input
    id={intervalScheduleId}
    type="radio"
    name="schedule-kind"
    checked={scheduleKind === 'interval'}
    onChange={() => {
      setScheduleKind('interval');
      setDraft((d) => ({ ...d, poll_cron: '', poll_cron_timezone: 'UTC' }));
    }}
  />
  <label htmlFor={intervalScheduleId}>固定间隔</label>
</div>
```

- [x] **Step 3: 将三个复选项改为同一形状**

把各 `label.check.modal-check-inline` 改为 `div.modal-check-option`；以启用订阅为例：

```tsx
<div className="modal-check-option">
  <input
    id={enabledId}
    type="checkbox"
    checked={draft.enabled}
    onChange={(event) => setDraft({ ...draft, enabled: event.target.checked })}
  />
  <label htmlFor={enabledId}>启用此订阅</label>
</div>
```

使用 `useProxyId` 和 `aiRenameEnabledId` 以相同方式包裹另外两个复选框，并保留其现有状态和 `onChange` 回调。

- [x] **Step 4: 添加局部紧凑布局样式**

将现有 `.modal-check-inline` 规则替换为：

```css
.modal-check-option {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  width: fit-content;
}

.modal-check-option input {
  width: 18px;
  height: 18px;
  margin: 0;
  accent-color: var(--glass-accent);
}

.modal-check-option label {
  cursor: pointer;
}
```

在窄屏/宽屏调度 fieldset 的选择器中，把 `.modal-check-inline` 改为 `.modal-check-option`，确保调度项仍处于既有 `schedule-mode-row` 布局内。

- [x] **Step 5: 运行浏览器回归测试，确认通过**

运行：

```bash
npm run test:e2e -- e2e/subscription-layout.spec.ts
```

预期：新增测试通过；点击容器右侧 24px 的留白不会改变选中状态，点击文字会切换复选框。

- [x] **Step 6: 运行完整前端验证**

运行：

```bash
npm test
npm run build
npm run test:e2e -- e2e/subscription-layout.spec.ts
```

预期：全部测试及生产构建通过。

- [x] **Step 7: 提交实现**

```bash
git add web/src/App.tsx web/src/styles.css e2e/subscription-layout.spec.ts
git commit -m '缩小订阅弹窗勾选项命中范围'
```
