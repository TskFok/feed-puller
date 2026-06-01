import { test, expect } from '@playwright/test';
import { mockLoggedIn, resetClientState } from './helpers/mock-api';
import { mockProwlarrConfigured, mockProwlarrSearchResults, scrollAppWorkspace } from './helpers/glass';

test.describe('Prowlarr 虚拟滚动', () => {
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

  test('144 条结果滚动 workspace 可加载更靠后的条目', async ({ page }) => {
    await mockProwlarrConfigured(page);
    await mockProwlarrSearchResults(page, 144);
    await page.setViewportSize({ width: 1280, height: 720 });
    await page.goto('/#prowlarr');
    await page.getByLabel('关键词').fill('bulk');
    await page.getByRole('button', { name: '搜索', exact: true }).click();

    await expect(page.locator('.prowlarr-results-grid--virtual')).toBeVisible();
    await expect(page.getByText(/已浏览 \d+ \/ 共 144 条/)).toBeVisible();

    const initialCount = await page.locator('.prowlarr-release-card:not(.prowlarr-release-card--skeleton)').count();
    expect(initialCount).toBeLessThan(144);
    expect(initialCount).toBeGreaterThan(0);

    await scrollAppWorkspace(page, 'bottom');
    await expect(page.getByRole('heading', { name: 'Release 140' })).toBeVisible();
    await expect(page.getByText('已浏览全部结果')).toBeVisible();
  });

  test('滚动 workspace 后虚拟行仍不重叠', async ({ page }) => {
    await mockProwlarrConfigured(page);
    await mockProwlarrSearchResults(page, 144);
    await page.setViewportSize({ width: 1280, height: 720 });
    await page.goto('/#prowlarr');
    await page.getByLabel('关键词').fill('bulk');
    await page.getByRole('button', { name: '搜索', exact: true }).click();
    await expect(page.locator('.prowlarr-results-grid--virtual')).toBeVisible();

    await scrollAppWorkspace(page, 'bottom');
    await page.waitForTimeout(200);

    const overlap = await page.evaluate(() => {
      const rows = Array.from(document.querySelectorAll<HTMLElement>('.prowlarr-results-virtual-row[data-index]'));
      if (rows.length < 2) {
        return false;
      }
      const sorted = rows
        .map((row) => row.getBoundingClientRect())
        .sort((a, b) => a.top - b.top);
      for (let i = 1; i < sorted.length; i += 1) {
        if (sorted[i].top < sorted[i - 1].bottom - 2) {
          return true;
        }
      }
      return false;
    });
    expect(overlap).toBe(false);
  });
});
