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
    const scheduleCell = cell.parentElement?.querySelector<HTMLElement>('.sub-schedule-cell');
    return {
      cellDisplay: getComputedStyle(cell).display,
      actionsDisplay: actions ? getComputedStyle(actions).display : null,
      actionsFlexWrap: actions ? getComputedStyle(actions).flexWrap : null,
      scheduleWidth: scheduleCell ? getComputedStyle(scheduleCell).width : null
    };
  });

  expect(layout).toEqual({
    cellDisplay: 'table-cell',
    actionsDisplay: 'flex',
    actionsFlexWrap: 'nowrap',
    scheduleWidth: '192px'
  });
});

test('订阅弹窗勾选项不会由行内留白切换', async ({ page }) => {
  await resetClientState(page);
  await mockLoggedIn(page);
  await page.goto('/');
  await page.getByRole('button', { name: '新增订阅' }).click();

  const checkbox = page.getByRole('checkbox', { name: '启用此订阅' });
  const option = checkbox.locator('xpath=..');
  await expect(option).toHaveClass(/modal-check-option/);
  await expect(checkbox).toBeChecked();

  const box = await option.boundingBox();
  expect(box).not.toBeNull();
  await page.mouse.click(box!.x + box!.width + 24, box!.y + box!.height / 2);
  await expect(checkbox).toBeChecked();

  await page.getByText('启用此订阅', { exact: true }).click();
  await expect(checkbox).not.toBeChecked();
});
