# 不透明操作提示 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让浅色与深色主题中的成功、失败 Toast 都使用完全不透明的背景。

**Architecture:** Toast 继续使用共享的 `.toast` 结构、状态类和交互逻辑。在专用的 `.toast` 规则中覆盖共享玻璃面板背景，复用主题已定义的 `--glass-panel-solid`，使两种状态和两套主题均获得实色背景。

**Tech Stack:** React 18、TypeScript、CSS、Vitest。

## Global Constraints

- 仅修改 Toast 及其样式回归测试，不调整其他玻璃拟态组件。
- 成功与失败提示均使用 `var(--glass-panel-solid)`；保持边框、状态色、阴影、动画和交互不变。
- 所有提交信息必须使用简体中文。

---

### Task 1: 为 Toast 不透明背景建立回归测试

**Files:**
- Modify: `web/src/modalSurfaceStyles.test.js`
- Modify: `web/src/styles.css:848-856`

**Interfaces:**
- Consumes: `web/src/styles.css` 中已定义的 `--glass-panel-solid`，浅色为 `#ffffff`、深色为 `#141c2e`。
- Produces: `.toast` 规则显式使用 `background: var(--glass-panel-solid)`，使所有 `toast-success` 与 `toast-error` 元素均不透明。

- [ ] **Step 1: 写入失败测试**

在 `web/src/modalSurfaceStyles.test.js` 的 `describe` 块中加入：

```js
  it('成功与失败 Toast 使用不透明的主题面板背景', () => {
    expect(styles.includes('.toast {\n  background: var(--glass-panel-solid);')).toBe(true);
  });
```

- [ ] **Step 2: 运行测试并确认失败**

运行：`npm test -- --run modalSurfaceStyles`

预期：失败，断言找不到 Toast 专用的 `background: var(--glass-panel-solid)` 规则；当前 `.toast` 从共享选择器继承半透明的 `var(--glass-panel)`。

- [ ] **Step 3: 写入最小实现**

在 `web/src/styles.css` 的 `.toast` 规则顶部加入：

```css
.toast {
  background: var(--glass-panel-solid);
  display: flex;
```

不改动 `.toast-success`、`.toast-error` 的边框色和左侧状态色，也不修改共享的 `.toast` 选择器。

- [ ] **Step 4: 运行测试并确认通过**

运行：`npm test -- --run modalSurfaceStyles`

预期：退出码为 0，所有 `modalSurfaceStyles` 测试通过。

- [ ] **Step 5: 运行完整前端验证**

运行：`npm test && npm run build`

预期：退出码为 0；TypeScript 测试编译、全部 Vitest 用例和 Vite 生产构建通过。

- [ ] **Step 6: 提交实现**

```bash
git add web/src/styles.css web/src/modalSurfaceStyles.test.js
git commit -m '将操作提示改为不透明背景'
```
