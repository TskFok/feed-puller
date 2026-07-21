import { expect, test } from '@playwright/test';
import { mockLoggedIn, resetClientState } from './helpers/mock-api';
import { makeProwlarrRelease, mockProwlarrConfigured } from './helpers/glass';

test.describe('Prowlarr 结果卡片窄屏布局', () => {
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

  test('超长已提交标题在窄屏中仍保持卡片布局约束', async ({ page }) => {
    const title = '一段非常长的 Prowlarr 搜索结果标题，用于验证窄屏时标题只能在单行内省略而不会挤出状态或下载按钮'.repeat(4);
    await mockProwlarrConfigured(page);
    await page.route('**/api/prowlarr/submitted-guids**', (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ guids: ['guid-0'] })
      })
    );
    await page.route('**/api/prowlarr/search?**', (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ items: [{ ...makeProwlarrRelease(0), title }] })
      })
    );
    await page.setViewportSize({ width: 360, height: 720 });
    await page.goto('/#prowlarr');
    await page.getByLabel('关键词').fill('long title');
    await page.getByRole('button', { name: '搜索', exact: true }).click();

    const heading = page.getByRole('heading', { name: title });
    const card = page.locator('.prowlarr-release-card').filter({ has: heading });
    const checkbox = card.getByRole('checkbox', { name: `选择 ${title}` });
    const truncatedTitle = card.locator('.truncated-text');
    const download = card.getByRole('button', { name: '重新下载' });

    await expect(heading).toBeVisible();
    await expect(card.getByText('已提交', { exact: true })).toBeVisible();

    const layout = await card.evaluate((cardElement) => {
      const checkboxElement = cardElement.querySelector<HTMLInputElement>("input[type='checkbox']");
      const titleElement = cardElement.querySelector<HTMLElement>('.truncated-text');
      const downloadElement = cardElement.querySelector<HTMLElement>('.prowlarr-release-download');
      if (!checkboxElement || !titleElement || !downloadElement) {
        throw new Error('缺少 Prowlarr 卡片布局节点');
      }
      const checkboxRect = checkboxElement.getBoundingClientRect();
      const titleRect = titleElement.getBoundingClientRect();
      const downloadRect = downloadElement.getBoundingClientRect();
      const cardRect = cardElement.getBoundingClientRect();
      const style = getComputedStyle(titleElement);
      return {
        checkboxTop: checkboxRect.top,
        titleTop: titleRect.top,
        titleClientWidth: titleElement.clientWidth,
        titleScrollWidth: titleElement.scrollWidth,
        titleWhiteSpace: style.whiteSpace,
        titleOverflow: style.overflow,
        titleTextOverflow: style.textOverflow,
        buttonLeft: downloadRect.left,
        buttonRight: downloadRect.right,
        buttonTop: downloadRect.top,
        buttonBottom: downloadRect.bottom,
        cardLeft: cardRect.left,
        cardRight: cardRect.right,
        cardTop: cardRect.top,
        cardBottom: cardRect.bottom
      };
    });

    expect(layout.checkboxTop).toBeCloseTo(layout.titleTop, 1);
    expect(layout.titleScrollWidth).toBeGreaterThan(layout.titleClientWidth);
    expect(layout.titleWhiteSpace).toBe('nowrap');
    expect(layout.titleOverflow).toBe('hidden');
    expect(layout.titleTextOverflow).toBe('ellipsis');
    expect(layout.buttonLeft).toBeGreaterThanOrEqual(layout.cardLeft);
    expect(layout.buttonRight).toBeLessThanOrEqual(layout.cardRight);
    expect(layout.buttonTop).toBeGreaterThanOrEqual(layout.cardTop);
    expect(layout.buttonBottom).toBeLessThanOrEqual(layout.cardBottom);
  });
});
