import { test, expect } from '@playwright/test';
import { gotoSettings, mockLoggedInUnbound, resetClientState } from './helpers/mock-api';

test.describe('飞书绑定弹窗', () => {
  test.beforeEach(async ({ page }) => {
    await resetClientState(page);
    await mockLoggedInUnbound(page);
    await page.route('**/*feishucdn.com/**', (route) => route.abort());
    await page.route('**/*feishu.cn/**', (route) => route.abort());
  });

  test('设置页点击绑定飞书打开 AnimatedModal 并渲染扫码容器', async ({ page }) => {
    await gotoSettings(page);

    await page.getByRole('button', { name: '绑定飞书', exact: true }).click();

    const dialog = page.getByRole('dialog', { name: '绑定飞书' });
    await expect(dialog).toBeVisible();
    await expect(dialog.getByText('使用飞书 App 扫码，可将飞书账号绑定到当前用户')).toBeVisible();
    await expect(dialog).toHaveClass(/bind-feishu-modal/);
    await expect(page.locator('#feishuBindQRContainer')).toBeVisible();
    await expect(page.locator('#feishuBindIframeContainer')).toBeAttached();
    await expect(page.locator('body')).toHaveCSS('overflow', 'hidden');
  });

  test('迁移向导的立即绑定飞书同样打开弹窗', async ({ page }) => {
    await gotoSettings(page);

    await page.getByRole('button', { name: '立即绑定飞书' }).click();

    await expect(page.getByRole('dialog', { name: '绑定飞书' })).toBeVisible();
    await expect(page.locator('#feishuBindQRContainer')).toBeVisible();
  });

  test('Escape 与关闭按钮可关闭弹窗', async ({ page }) => {
    await gotoSettings(page);
    await page.getByRole('button', { name: '绑定飞书', exact: true }).click();
    await expect(page.getByRole('dialog', { name: '绑定飞书' })).toBeVisible();

    await page.keyboard.press('Escape');
    await expect(page.getByRole('dialog')).toHaveCount(0);
    await expect(page.locator('body')).not.toHaveCSS('overflow', 'hidden');

    await page.getByRole('button', { name: '绑定飞书', exact: true }).click();
    await page.getByRole('dialog', { name: '绑定飞书' }).getByRole('button', { name: '关闭' }).click();
    await expect(page.getByRole('dialog')).toHaveCount(0);
  });

  test('点击遮罩关闭弹窗', async ({ page }) => {
    await gotoSettings(page);
    await page.getByRole('button', { name: '绑定飞书', exact: true }).click();
    await expect(page.getByRole('dialog', { name: '绑定飞书' })).toBeVisible();

    await page.locator('.modal-overlay').click({ position: { x: 8, y: 8 } });
    await expect(page.getByRole('dialog')).toHaveCount(0);
  });

  test('Tab 键焦点保持在弹窗内', async ({ page }) => {
    await gotoSettings(page);
    await page.getByRole('button', { name: '绑定飞书', exact: true }).click();

    const dialog = page.getByRole('dialog', { name: '绑定飞书' });
    await expect(dialog).toBeVisible();

    const closeButton = dialog.getByRole('button', { name: '关闭' });
    await expect(closeButton).toBeFocused();

    for (let i = 0; i < 6; i++) {
      const activeInDialog = await page.evaluate(() => {
        const dialogEl = document.querySelector('[role="dialog"]');
        const active = document.activeElement;
        return !!(dialogEl && active && dialogEl.contains(active));
      });
      expect(activeInDialog).toBe(true);
      await page.keyboard.press('Tab');
    }

    await expect(closeButton).toBeFocused();
  });
});
