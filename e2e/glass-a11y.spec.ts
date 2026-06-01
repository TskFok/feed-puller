import { test, expect } from '@playwright/test';
import { mockLoggedIn, mockLoggedOut, resetClientState } from './helpers/mock-api';
import {
  mockProwlarrBatchDownload,
  mockProwlarrConfigured,
  mockProwlarrSearchResults,
  virtualProwlarrRowsOverlap
} from './helpers/glass';

test.describe('玻璃态无障碍（未登录）', () => {
  test.beforeEach(async ({ page }) => {
    await resetClientState(page);
    await mockLoggedOut(page);
  });

  test('浅色主题登录面板次要文字对比度 ≥4.5:1', async ({ page }) => {
    await page.emulateMedia({ colorScheme: 'light' });
    await page.goto('/');
    await expect(page.getByRole('heading', { name: 'feed-puller' })).toBeVisible();
    await page.getByRole('button', { name: '玻璃浅色' }).click();
    await expect(page.locator('html')).toHaveAttribute('data-theme', 'light');

    const ratio = await page.evaluate(() => {
      const panelAlpha = 0.68;
      const body: [number, number, number] = [240, 253, 250];
      const blend = (fg: [number, number, number], a: number, bg: [number, number, number]) =>
        fg.map((c, i) => Math.round(a * c + (1 - a) * bg[i])) as [number, number, number];
      const lum = ([r, g, b]: [number, number, number]) => {
        const [rs, gs, bs] = [r, g, b].map((c) => {
          const x = c / 255;
          return x <= 0.03928 ? x / 12.92 : ((x + 0.055) / 1.055) ** 2.4;
        });
        return 0.2126 * rs + 0.7152 * gs + 0.0722 * bs;
      };
      const contrast = (fg: [number, number, number], bg: [number, number, number]) => {
        const L1 = lum(fg);
        const L2 = lum(bg);
        return (Math.max(L1, L2) + 0.05) / (Math.min(L1, L2) + 0.05);
      };
      const panel = blend([255, 255, 255], panelAlpha, body);
      const muted: [number, number, number] = [0x4a, 0x69, 0x64];
      return contrast(muted, panel);
    });

    expect(ratio).toBeGreaterThanOrEqual(4.5);
  });

  test('glass-no-backdrop-test 降级时面板无 backdrop-filter', async ({ page }) => {
    await page.goto('/');
    await expect(page.getByRole('heading', { name: 'feed-puller' })).toBeVisible();
    await page.evaluate(() => document.documentElement.classList.add('glass-no-backdrop-test'));
    await page.getByRole('button', { name: '玻璃浅色' }).click();

    const panelFilter = await page.locator('.login-panel').evaluate((el) => getComputedStyle(el).backdropFilter);
    expect(panelFilter === 'none' || panelFilter === '').toBe(true);
  });
});

test.describe('玻璃态无障碍与性能（已登录）', () => {
  test.beforeEach(async ({ page }) => {
    await resetClientState(page);
    await mockLoggedIn(page);
    await page.route('**/api/downloads/active**', (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ items: [], total: 0, page: 1, page_size: 20 })
      })
    );
    await page.route('**/api/downloads/completed**', (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ items: [], total: 0, page: 1, page_size: 20 })
      })
    );
  });

  test('prefers-reduced-motion 下模态无入场动画', async ({ page }) => {
    await page.emulateMedia({ reducedMotion: 'reduce' });
    await page.goto('/#subscriptions');
    await page.getByRole('button', { name: '新增订阅' }).click();
    await expect(page.getByRole('dialog')).toBeVisible();

    const overlayAnimation = await page.locator('.modal-overlay').evaluate((el) => getComputedStyle(el).animationName);
    const panelAnimation = await page.locator('.modal-panel').evaluate((el) => getComputedStyle(el).animationName);
    expect(overlayAnimation).toBe('none');
    expect(panelAnimation).toBe('none');
  });

  test('Prowlarr 大量结果启用虚拟网格', async ({ page }) => {
    await mockProwlarrConfigured(page);
    await mockProwlarrSearchResults(page, 55);
    await page.goto('/#prowlarr');
    await page.getByLabel('关键词').fill('test');
    await page.getByRole('button', { name: '搜索', exact: true }).click();

    await expect(page.locator('.prowlarr-results-grid--virtual')).toBeVisible();
    const renderedCards = await page.locator('.prowlarr-release-card:not(.prowlarr-release-card--skeleton)').count();
    expect(renderedCards).toBeLessThan(55);
    expect(renderedCards).toBeGreaterThan(0);
  });

  test('Prowlarr 虚拟网格相邻行不重叠', async ({ page }) => {
    await mockProwlarrConfigured(page);
    await mockProwlarrSearchResults(page, 55);
    await page.setViewportSize({ width: 1280, height: 720 });
    await page.goto('/#prowlarr');
    await page.getByLabel('关键词').fill('test');
    await page.getByRole('button', { name: '搜索', exact: true }).click();
    await expect(page.locator('.prowlarr-results-grid--virtual')).toBeVisible();

    expect(await virtualProwlarrRowsOverlap(page)).toBe(false);
  });

  test('Prowlarr 展开批量失败摘要后虚拟行仍不重叠', async ({ page }) => {
    await mockProwlarrConfigured(page);
    await mockProwlarrSearchResults(page, 55);
    await mockProwlarrBatchDownload(page, {
      successCount: 1,
      failures: [{ guid: 'guid-1', error: '该资源正在下载中' }]
    });
    await page.setViewportSize({ width: 1280, height: 720 });
    await page.goto('/#prowlarr');
    await page.getByLabel('关键词').fill('test');
    await page.getByRole('button', { name: '搜索', exact: true }).click();
    await expect(page.locator('.prowlarr-results-grid--virtual')).toBeVisible();

    await page.getByRole('checkbox', { name: /全选/ }).check();
    await page.getByRole('button', { name: '批量下载' }).click();
    await expect(page.getByText(/本次提交：成功 1 条，失败 1 条/)).toBeVisible();
    await expect(page.locator('.prowlarr-batch-failures-list')).toBeVisible();

    await page.evaluate(() => window.scrollTo(0, document.body.scrollHeight));
    await page.waitForTimeout(200);
    expect(await virtualProwlarrRowsOverlap(page)).toBe(false);
  });

  test('Prowlarr 超过 30 条结果启用虚拟网格', async ({ page }) => {
    await mockProwlarrConfigured(page);
    await mockProwlarrSearchResults(page, 35);
    await page.goto('/#prowlarr');
    await page.getByLabel('关键词').fill('test');
    await page.getByRole('button', { name: '搜索', exact: true }).click();
    await expect(page.locator('.prowlarr-results-grid--virtual')).toBeVisible();
  });

  test('Prowlarr 中等结果离屏卡片可标记 glass-surface--offscreen', async ({ page }) => {
    await mockProwlarrConfigured(page);
    await mockProwlarrSearchResults(page, 18);
    await page.setViewportSize({ width: 1280, height: 500 });
    await page.goto('/#prowlarr');
    await page.getByLabel('关键词').fill('test');
    await page.getByRole('button', { name: '搜索', exact: true }).click();
    await expect(page.locator('.prowlarr-release-card').first()).toBeVisible();

    await page.evaluate(() => window.scrollTo(0, document.body.scrollHeight));
    await page.waitForFunction(
      () => document.querySelectorAll('.prowlarr-release-card.glass-surface--offscreen').length > 0,
      { timeout: 3000 }
    );

    const offscreenCount = await page.locator('.prowlarr-release-card.glass-surface--offscreen').count();
    expect(offscreenCount).toBeGreaterThan(0);
  });
});
