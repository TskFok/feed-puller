# 订阅弹窗固定头尾 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 使新增与编辑订阅弹窗的头部和底部操作区固定，只有表单字段区域可滚动。

**Architecture:** 订阅表单向 `AnimatedModal` 传入专用面板类名，避免影响其余弹窗。表单拆为固定的头部、拥有 `overflow-y: auto` 的字段容器，以及固定的底部操作区；专用 CSS 以 flex 约束中间区域高度。

**Tech Stack:** React 18、TypeScript、CSS、Vitest、Testing Library。

## Global Constraints

- 只修改订阅创建与编辑共用的表单弹窗；不得改变其他 `AnimatedModal` 调用方。
- 保持现有可访问名称、关闭/提交逻辑、焦点管理和接口调用不变。
- 不在循环中查询 SQL。
- 提交信息必须使用简体中文。

---

## File structure

- `web/src/App.tsx`：订阅表单弹窗的 DOM 分区与专用面板类名。
- `web/src/styles.css`：订阅表单弹窗的固定头尾和中间滚动样式。
- `web/src/App.test.tsx`：新增和编辑订阅弹窗的结构性回归测试。

### Task 1: 为新增和编辑订阅弹窗建立结构回归测试

**Files:**
- Modify: `web/src/App.test.tsx:656-687`
- Modify: `web/src/App.test.tsx:852-897`

**Interfaces:**
- Consumes: `role="dialog"` 的订阅弹窗、标题文本、现有关闭和提交按钮。
- Produces: 对 `.subscription-form-modal`、`.subscription-form-scroll` 与 `.subscription-form-actions` 的 DOM 结构断言。

- [ ] **Step 1: 在新增订阅测试中写入失败断言。**

  在现有“登录后可通过新增订阅弹窗填写关键字过滤”测试的现有 dialog 断言后，添加：

  ```tsx
  const dialog = screen.getByRole('dialog', { name: '新增订阅' });
  const form = dialog.querySelector('form.subscription-edit-form');
  const scrollArea = dialog.querySelector('.subscription-form-scroll');
  const actions = dialog.querySelector('.subscription-form-actions');

  expect(dialog).toHaveClass('subscription-form-modal');
  expect(dialog.querySelector('.modal-header-row')?.parentElement).toBe(dialog);
  expect(form).not.toBeNull();
  expect(scrollArea?.parentElement).toBe(form);
  expect(scrollArea).toContainElement(screen.getByRole('textbox', { name: '包含关键字' }));
  expect(actions?.parentElement).toBe(form);
  expect(scrollArea).not.toContainElement(actions);
  ```

- [ ] **Step 2: 在编辑订阅测试中写入失败断言。**

  在“点击编辑后以弹窗形式打开订阅编辑表单”测试的关闭按钮断言后，添加：

  ```tsx
  const dialog = screen.getByRole('dialog', { name: '编辑订阅' });
  const form = dialog.querySelector('form.subscription-edit-form');
  const scrollArea = dialog.querySelector('.subscription-form-scroll');
  const actions = dialog.querySelector('.subscription-form-actions');

  expect(dialog).toHaveClass('subscription-form-modal');
  expect(dialog.querySelector('.modal-header-row')?.parentElement).toBe(dialog);
  expect(form).not.toBeNull();
  expect(scrollArea?.parentElement).toBe(form);
  expect(scrollArea).toContainElement(screen.getByDisplayValue('Demo'));
  expect(actions?.parentElement).toBe(form);
  expect(scrollArea).not.toContainElement(actions);
  ```

- [ ] **Step 3: 运行目标测试并确认失败。**

  Run: `npx vitest run web/src/App.test.tsx -t '新增订阅弹窗填写关键字过滤|点击编辑后以弹窗形式打开订阅编辑表单'`

  Expected: FAIL，提示订阅 dialog 尚未包含 `subscription-form-modal` 或 `.subscription-form-scroll`。

### Task 2: 实现订阅表单的固定头尾布局

**Files:**
- Modify: `web/src/App.tsx:645-931`
- Modify: `web/src/styles.css:1126-1144`
- Modify: `web/src/styles.css:1255-1258`
- Modify: `web/src/styles.css:1432-1439`

**Interfaces:**
- Consumes: Task 1 的 CSS 类选择器及现有 `submit`、`onClose`、`saving` 行为。
- Produces: `.subscription-form-modal` 面板、`.subscription-form-scroll` 字段滚动区和 `.subscription-form-actions` 固定操作区。

- [ ] **Step 1: 为订阅表单弹窗添加局部面板类名。**

  将订阅 `AnimatedModal` 的开始标签改为：

  ```tsx
  <AnimatedModal
    onClose={onClose}
    ariaLabelledBy={titleId}
    initialFocusRef={firstFieldRef}
    panelClassName="subscription-form-modal"
  >
  ```

- [ ] **Step 2: 将字段与操作区分离。**

  保持 `<form className="subscription-edit-form" onSubmit={submit}>`，用以下结构包住全部五个 `fieldset`，并仅把原 `.modal-actions` 放在滚动容器之后：

  ```tsx
  <div className="subscription-form-scroll">
    {/* 保留现有基本信息、抓取与选项、保存路径、AI 刮削重命名和条目过滤五个 fieldset，字段与事件处理不变 */}
  </div>
  <div className="modal-actions subscription-form-actions">
    <button type="button" className="ghost" disabled={saving} onClick={onClose}>
      取消
    </button>
    <button type="submit" className="primary" disabled={saving}>
      {saving ? (isCreate ? '创建中…' : '保存中…') : isCreate ? '创建订阅' : '保存更改'}
    </button>
  </div>
  ```

- [ ] **Step 3: 添加仅作用于订阅表单的布局规则。**

  在 `.modal-panel` 规则后添加：

  ```css
  .subscription-form-modal.modal-panel {
    display: flex;
    flex-direction: column;
    overflow: hidden;
    padding: 0;
  }

  .subscription-form-modal .modal-header-row {
    flex-shrink: 0;
    margin: 0;
    padding: 22px 24px 14px;
  }

  .subscription-form-modal .subscription-edit-form {
    display: flex;
    flex: 1;
    flex-direction: column;
    gap: 0;
    min-height: 0;
  }

  .subscription-form-scroll {
    display: grid;
    flex: 1;
    min-height: 0;
    gap: 18px;
    overflow-y: auto;
    padding: 0 24px;
  }

  .subscription-form-actions.modal-actions {
    flex-shrink: 0;
    margin: 0;
    padding: 16px 24px 24px;
  }
  ```

  该规则通过 `.subscription-form-modal .subscription-edit-form` 局部覆盖现有 `gap: 18px` 为 `gap: 0`，使间距只由 `.subscription-form-scroll` 承担；不得修改通用 `.subscription-edit-form` 规则，从而不影响 AI 配置表单。

- [ ] **Step 4: 运行目标测试并确认通过。**

  Run: `npx vitest run web/src/App.test.tsx -t '新增订阅弹窗填写关键字过滤|点击编辑后以弹窗形式打开订阅编辑表单'`

  Expected: PASS，新增和编辑测试均验证头部、滚动字段区、底部操作区的父子关系。

- [ ] **Step 5: 提交实现。**

  ```bash
  git add web/src/App.tsx web/src/styles.css web/src/App.test.tsx
  git commit -m '固定订阅表单弹窗头尾'
  ```

### Task 3: 完整验证

**Files:**
- Verify: `web/src/App.tsx`
- Verify: `web/src/styles.css`
- Verify: `web/src/App.test.tsx`

**Interfaces:**
- Consumes: Task 2 的完整实现。
- Produces: 已通过类型检查、单元测试与生产构建的订阅弹窗布局。

- [ ] **Step 1: 运行完整前端测试。**

  Run: `npm test`

  Expected: PASS，Vitest 全部用例通过。

- [ ] **Step 2: 运行生产构建。**

  Run: `npm run build`

  Expected: PASS，TypeScript 编译和 Vite 构建完成，退出码为 0。

- [ ] **Step 3: 检查工作区状态。**

  Run: `git status --short`

  Expected: 无未提交的订阅弹窗代码改动；设计文档与实现各有独立提交。
