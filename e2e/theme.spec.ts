import { test, expect } from '@playwright/test';
import { mockLoggedOut, resetClientState } from './helpers/mock-api';

test.describe('登录页主题切换', () => {
  test.beforeEach(async ({ page }) => {
    await resetClientState(page);
    await mockLoggedOut(page);
  });

  test('未登录时可切换 Bubblegum 浅色主题', async ({ page }) => {
    await page.goto('/');

    await expect(page.getByRole('heading', { name: 'feed-puller' })).toBeVisible();
    await page.getByRole('button', { name: 'Bubblegum 浅色' }).click();

    await expect(page.locator('html')).toHaveAttribute('data-theme', 'light');
    await expect(page.getByRole('button', { name: 'Y2K 暗色' })).toBeVisible();
  });

  test('主题偏好写入 localStorage', async ({ page }) => {
    await page.goto('/');

    await page.getByRole('button', { name: '跟随系统' }).click();

    const stored = await page.evaluate(() => localStorage.getItem('feed-puller-theme'));
    expect(stored).toBe('system');
  });
});

test.describe('主题 crossfade 动效', () => {
  test.beforeEach(async ({ page }) => {
    await resetClientState(page);
    await mockLoggedOut(page);
  });

  test('默认动效偏好下切换主题添加并移除 theme-switching 类', async ({ page }) => {
    await page.emulateMedia({ reducedMotion: 'no-preference' });
    await page.goto('/');

    await page.getByRole('button', { name: 'Bubblegum 浅色' }).click();

    await expect(page.locator('html')).toHaveAttribute('data-theme', 'light');
    await expect(page.locator('html')).toHaveClass(/theme-switching/);
    await page.waitForFunction(() => !document.documentElement.classList.contains('theme-switching'), {
      timeout: 1000
    });
  });

  test('prefers-reduced-motion 时不添加 theme-switching 但主题仍切换', async ({ page }) => {
    await page.emulateMedia({ reducedMotion: 'reduce' });
    await page.goto('/');

    await page.getByRole('button', { name: 'Bubblegum 浅色' }).click();

    await expect(page.locator('html')).toHaveAttribute('data-theme', 'light');
    await page.waitForTimeout(100);
    const hasSwitchingClass = await page.evaluate(() =>
      document.documentElement.classList.contains('theme-switching')
    );
    expect(hasSwitchingClass).toBe(false);
  });

  test('prefers-reduced-motion 时 html 过渡被 CSS 禁用', async ({ page }) => {
    await page.emulateMedia({ reducedMotion: 'reduce' });
    await page.goto('/');

    const transitionDuration = await page.evaluate(() => getComputedStyle(document.documentElement).transitionDuration);
    expect(transitionDuration).toBe('0s');
  });

  test('设置页切换主题同样遵循 reduced-motion', async ({ page }) => {
    await page.emulateMedia({ reducedMotion: 'reduce' });
    await page.route('**/api/auth/me', (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ id: 1, email: 'u@test.dev', feishu_bound: true })
      })
    );
    await page.route('**/api/auth/options', (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ password_login_enabled: true, feishu_login_enabled: true })
      })
    );
    await page.route('**/api/settings/proxy', (route) =>
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ proxy_url: '' }) })
    );
    await page.route('**/api/settings/prowlarr', (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          url: '',
          api_key: '',
          download_dir: '',
          tv_download_dir: '',
          movie_rename_enabled: false,
          tmdb_api_key: '',
          indexer_ids: [],
          configured: false
        })
      })
    );

    await page.goto('/#settings');
    await expect(page.getByRole('heading', { name: '设置' })).toBeVisible();

    await page.getByRole('button', { name: 'Bubblegum 浅色' }).click();

    await expect(page.locator('html')).toHaveAttribute('data-theme', 'light');
    await page.waitForTimeout(100);
    expect(await page.evaluate(() => document.documentElement.classList.contains('theme-switching'))).toBe(false);
  });
});
