import { test, expect } from '@playwright/test';
import { gotoSettings, mockFeishuUnbind, mockLoggedInBound, resetClientState } from './helpers/mock-api';

test.describe('飞书解绑', () => {
  test.beforeEach(async ({ page }) => {
    await resetClientState(page);
    await mockLoggedInBound(page);
    await mockFeishuUnbind(page);
  });

  test('已绑定用户可在设置页解绑飞书', async ({ page }) => {
    await gotoSettings(page);

    await expect(page.getByText('当前状态：Alice')).toBeVisible();
    await expect(page.getByRole('button', { name: '解绑' })).toBeVisible();

    await page.getByRole('button', { name: '解绑' }).click();

    await expect(page.getByText('飞书账号已解绑')).toBeVisible();
    await expect(page.getByText('当前状态：未绑定')).toBeVisible();
    await expect(page.getByRole('button', { name: '解绑' })).toHaveCount(0);
    await expect(page.getByRole('button', { name: '绑定飞书', exact: true })).toBeVisible();
  });

  test('解绑后迁移向导恢复为未绑定状态', async ({ page }) => {
    await gotoSettings(page);

    await page.getByRole('button', { name: '解绑' }).click();
    await expect(page.getByText('飞书账号已解绑')).toBeVisible();

    await expect(page.getByRole('heading', { name: '飞书登录迁移向导' })).toBeVisible();
    await expect(page.getByRole('button', { name: '立即绑定飞书' })).toBeVisible();
    await expect(page.getByText('请先完成飞书绑定后再关闭密码登录。')).toBeVisible();
  });
});
