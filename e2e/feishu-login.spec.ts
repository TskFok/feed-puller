import { test, expect } from '@playwright/test';
import { mockFeishuOnlyLogin, mockLoggedOut, resetClientState } from './helpers/mock-api';

test.describe('登录页飞书扫码', () => {
  test.beforeEach(async ({ page }) => {
    await resetClientState(page);
    await page.route('**/*feishucdn.com/**', (route) => route.abort());
    await page.route('**/*feishu.cn/**', (route) => route.abort());
  });

  test('仅启用飞书登录时自动加载并渲染扫码容器', async ({ page }) => {
    await mockFeishuOnlyLogin(page);

    await page.goto('/');

    await expect(page.getByRole('heading', { name: 'feed-puller' })).toBeVisible();
    await expect(page.getByText('使用飞书 App 扫码即可登录')).toBeVisible();
    await expect(page.locator('#feishuLoginQRContainer')).toBeVisible();
    await expect(page.locator('#feishuLoginIframeContainer')).toBeAttached();
    await expect(page.getByLabel('邮箱')).toHaveCount(0);
    await expect(page.getByRole('button', { name: '飞书登录' })).toHaveCount(0);
  });

  test('双登录模式下切换到飞书后渲染扫码容器', async ({ page }) => {
    await mockLoggedOut(page);
    await page.route('**/api/auth/feishu/login-url', (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          url: '/api/auth/feishu/login?state=e2e',
          goto: 'https://passport.feishu.cn/suite/passport/oauth/authorize?state=e2e'
        })
      })
    );

    await page.goto('/');

    await page.getByRole('button', { name: '飞书登录' }).click();
    await expect(page.getByText('使用飞书 App 扫码即可登录')).toBeVisible();
    await expect(page.locator('#feishuLoginQRContainer')).toBeVisible();
    await expect(page.locator('#feishuLoginIframeContainer')).toBeAttached();
  });

  test('请求 login-url 期间显示加载提示', async ({ page }) => {
    await mockLoggedOut(page);
    await page.route('**/api/auth/feishu/login-url', async (route) => {
      await new Promise((resolve) => setTimeout(resolve, 300));
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          url: '/api/auth/feishu/login?state=e2e',
          goto: 'https://passport.feishu.cn/suite/passport/oauth/authorize?state=e2e'
        })
      });
    });

    await page.goto('/');
    await page.getByRole('button', { name: '飞书登录' }).click();

    await expect(page.getByText('正在加载飞书扫码...')).toBeVisible();
    await expect(page.locator('#feishuLoginQRContainer')).toHaveCount(0);

    await expect(page.getByText('使用飞书 App 扫码即可登录')).toBeVisible({ timeout: 5000 });
    await expect(page.locator('#feishuLoginQRContainer')).toBeVisible();
  });
});
