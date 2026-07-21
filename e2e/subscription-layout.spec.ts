import { expect, test } from '@playwright/test';
import { mockLoggedIn, resetClientState } from './helpers/mock-api';

test('订阅操作单元格保留表格布局，内部按钮单行排列', async ({ page }) => {
  await resetClientState(page);
  await mockLoggedIn(page);
  await page.unroute('**/api/subscriptions**');
  await page.route('**/api/subscriptions**', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        items: [
          {
            id: 1,
            name: '订阅名称',
            feed_url: 'https://example.test/feed.xml',
            enabled: true,
            poll_interval_minutes: 30,
            poll_cron: '',
            poll_cron_timezone: 'UTC',
            download_dir: '/data',
            include_keywords: '',
            exclude_keywords: '',
            use_proxy: false
          }
        ],
        total: 1,
        page: 1,
        page_size: 20
      })
    })
  );

  await page.goto('/');
  await expect(page.getByRole('heading', { name: '订阅' })).toBeVisible();

  const layout = await page.locator('td.subscription-actions').evaluate((cell) => {
    const actions = cell.querySelector<HTMLElement>('.subscription-actions-row');
    return {
      cellDisplay: getComputedStyle(cell).display,
      actionsDisplay: actions ? getComputedStyle(actions).display : null,
      actionsFlexWrap: actions ? getComputedStyle(actions).flexWrap : null
    };
  });

  expect(layout).toEqual({
    cellDisplay: 'table-cell',
    actionsDisplay: 'flex',
    actionsFlexWrap: 'nowrap'
  });
});
