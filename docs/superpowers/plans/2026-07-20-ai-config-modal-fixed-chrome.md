# AI 配置弹窗固定头尾 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让新增与编辑 AI 配置弹窗的标题和操作区固定，仅中间字段区域可以滚动。

**Architecture:** 在 `AIConfigModal` 中为面板、表单、字段滚动区和操作区增加 AI 配置专用类。CSS 使用局部纵向 flex 布局，面板裁剪溢出，字段区承载纵向滚动，头部和操作区禁止收缩；不改动通用模态规则。前端单测直接断言新增和编辑弹窗的 DOM 边界。

**Tech Stack:** React 18、TypeScript、CSS、Vitest、Testing Library。

## Global Constraints

- 只影响 AI 配置的新建和编辑表单弹窗，不改变 API、表单提交、模型拉取、焦点管理或其他弹窗。
- 标题、说明、关闭按钮和取消/保存操作区固定；Provider 预设和所有字段都位于中间可滚动区。
- 保持现有最大弹窗高度、可访问名称、移动端约束与主题变量。
- 不在循环遍历中查询 SQL。

---

## File Structure

- `web/src/App.tsx`：定义 AI 配置弹窗的 DOM 分区与专用类名。
- `web/src/App.test.tsx`：覆盖新增和编辑模式的固定区与滚动区结构。
- `web/src/styles.css`：为专用类提供固定头尾和中间滚动的布局规则。
- `design-system/pages/ai-config.md`：记录 AI 配置弹窗的固定头尾布局约定。

### Task 1: AI 配置弹窗结构回归测试

**Files:**
- Modify: `web/src/App.test.tsx:2310-2450`

**Interfaces:**
- Consumes: `AIConfigView` 的“新增配置”按钮、列表中的“编辑”按钮，以及 `AnimatedModal` 的 `role="dialog"`。
- Produces: 对 `.ai-config-form-modal`、`.ai-config-form-scroll`、`.ai-config-form-actions` 的回归约束。

- [ ] **Step 1: 写入失败的新增与编辑结构测试**

在 AI 配置测试区添加以下测试辅助断言，并分别通过“新增配置”和列表的“编辑”按钮打开弹窗：

```tsx
function expectAIConfigFormChrome(dialog: HTMLElement, field: HTMLElement) {
  const form = dialog.querySelector<HTMLFormElement>('form.ai-config-edit-form');
  const scrollArea = dialog.querySelector<HTMLElement>('.ai-config-form-scroll');
  const actions = dialog.querySelector<HTMLElement>('.ai-config-form-actions');

  expect(dialog).toHaveClass('ai-config-form-modal');
  expect(dialog.querySelector('.modal-header-row')?.parentElement).toBe(dialog);
  expect(form).not.toBeNull();
  expect(scrollArea?.parentElement).toBe(form);
  expect(scrollArea?.contains(field)).toBe(true);
  expect(actions?.parentElement).toBe(form);
  expect(scrollArea?.contains(actions ?? null)).toBe(false);
}
```

新增模式以 `within(dialog).getByLabelText('模型名称')` 作为 `field`；编辑模式使用含一条 `Demo` 配置的列表响应，点击 `within(screen.getByRole('row', { name: /Demo/ })).getByRole('button', { name: '编辑' })` 后，以 `within(dialog).getByDisplayValue('Demo')` 作为 `field`。

- [ ] **Step 2: 运行测试确认失败**

运行：`npm test -- --run App --testNamePattern="AI 配置弹窗"`

预期：失败，提示找不到 `.ai-config-form-modal` 或 `.ai-config-form-scroll`；这是当前弹窗尚未分区的预期失败。

- [ ] **Step 3: 保持测试仅断言可观察结构**

不模拟 CSS 计算值，也不访问组件内部状态；测试只验证面板、字段区和操作区的父子边界，以及新增与编辑两种入口。

### Task 2: 实现固定头尾与中间滚动

**Files:**
- Modify: `web/src/App.tsx:2040-2180`
- Modify: `web/src/styles.css:1140-1175`
- Modify: `design-system/pages/ai-config.md:17-20`
- Test: `web/src/App.test.tsx:2310-2450`

**Interfaces:**
- Consumes: Task 1 的 `.ai-config-form-modal`、`.ai-config-edit-form`、`.ai-config-form-scroll`、`.ai-config-form-actions` 断言。
- Produces: AI 配置弹窗固定头部和底部、可独立滚动的字段区。

- [ ] **Step 1: 将表单字段移入滚动容器**

将模态调用和表单开始标签调整为：

```tsx
<AnimatedModal
  onClose={onClose}
  ariaLabelledBy={titleId}
  initialFocusRef={firstFieldRef}
  panelClassName="ai-config-form-modal"
>
  <div className="modal-header-row">{/* 保留既有标题、说明与关闭按钮 */}</div>
  <form className="ai-config-edit-form" onSubmit={submit}>
    <div className="ai-config-form-scroll">
      {/* 保留既有 Provider fieldset 与全部字段 */}
    </div>
    <div className="modal-actions ai-config-form-actions">
      <button type="button" className="ghost" onClick={onClose}>取消</button>
      <button className="primary" disabled={saving}>{saving ? '保存中' : '保存'}</button>
    </div>
  </form>
</AnimatedModal>
```

不得改变字段的 `ref`、`value`、`onChange`、`required`、模型拉取按钮或 `submit` 函数；只移动现有 JSX 的容器边界。

- [ ] **Step 2: 添加局部 CSS 布局**

紧邻 `.subscription-form-modal` 规则添加：

```css
.ai-config-form-modal.modal-panel {
  display: flex;
  flex-direction: column;
  overflow: hidden;
  padding: 0;
}

.ai-config-form-modal .modal-header-row {
  flex-shrink: 0;
  margin: 0;
  padding: 22px 24px 14px;
}

.ai-config-edit-form {
  display: flex;
  flex: 1;
  flex-direction: column;
  min-height: 0;
}

.ai-config-form-scroll {
  display: grid;
  flex: 1;
  min-height: 0;
  gap: 18px;
  overflow-y: auto;
  padding: 0 24px 18px;
}

.ai-config-form-scroll > .modal-fieldset:first-child {
  padding-top: 0;
  border-top: 0;
}

.ai-config-form-actions.modal-actions {
  flex-shrink: 0;
  margin: 0;
  padding: 16px 24px 24px;
}
```

- [ ] **Step 3: 更新页面设计系统说明**

把 `design-system/pages/ai-config.md` 的弹窗布局描述更新为：

```markdown
- 新建/编辑：`AnimatedModal` + `.ai-config-form-modal`；标题和操作区固定，`.ai-config-form-scroll` 单独滚动
```

- [ ] **Step 4: 运行目标测试确认通过**

运行：`npm test -- --run App --testNamePattern="AI 配置弹窗"`

预期：新增与编辑结构测试通过，且无 TypeScript 或 Testing Library 错误。

- [ ] **Step 5: 运行完整前端验证**

运行：`npm test && npm run build`

预期：两个命令都以退出码 0 完成。

- [ ] **Step 6: 提交实现**

```bash
git add web/src/App.tsx web/src/App.test.tsx web/src/styles.css design-system/pages/ai-config.md
git commit -m "固定AI配置弹窗头尾"
```

## 自检

- 设计中的固定头部、固定操作区、中间字段滚动区、仅限 AI 配置范围与测试要求均由 Task 1 或 Task 2 覆盖。
- 已检查计划不存在占位符、未定义接口或相互矛盾的类名。
- 所有类名在测试、JSX、CSS 与设计系统说明中保持一致。
