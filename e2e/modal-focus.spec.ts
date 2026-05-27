import { test, expect } from '@playwright/test';
import { mockLoggedIn, resetClientState } from './helpers/mock-api';

test.describe('AnimatedModal 焦点陷阱', () => {
  test.beforeEach(async ({ page }) => {
    await resetClientState(page);
    await mockLoggedIn(page);
  });

  test('新增订阅弹窗 Tab 键在内部循环', async ({ page }) => {
    await page.goto('/');

    await page.getByRole('button', { name: '新增订阅' }).click();
    const dialog = page.getByRole('dialog');
    await expect(dialog).toBeVisible();

    const nameInput = dialog.getByRole('textbox', { name: '订阅名称' });
    await expect(nameInput).toBeFocused();

    const submitButton = dialog.getByRole('button', { name: '创建订阅' });
    await submitButton.focus();
    await page.keyboard.press('Tab');

    await expect(dialog.getByRole('button', { name: '关闭新建订阅' })).toBeFocused();

    for (let i = 0; i < 30; i++) {
      const activeInDialog = await page.evaluate(() => {
        const dialogEl = document.querySelector('[role="dialog"]');
        const active = document.activeElement;
        return !!(dialogEl && active && dialogEl.contains(active));
      });
      expect(activeInDialog).toBe(true);
      await page.keyboard.press('Tab');
    }
  });

  test('Escape 关闭弹窗并恢复页面交互', async ({ page }) => {
    await page.goto('/');

    await page.getByRole('button', { name: '新增订阅' }).click();
    await expect(page.getByRole('dialog')).toBeVisible();

    await page.keyboard.press('Escape');
    await expect(page.getByRole('dialog')).toHaveCount(0);
    await expect(page.getByRole('button', { name: '新增订阅' })).toBeEnabled();
  });
});
