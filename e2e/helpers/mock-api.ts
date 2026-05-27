import type { Page } from '@playwright/test';
import { expect } from '@playwright/test';

const defaultAuthOptions = {
  password_login_enabled: true,
  feishu_login_enabled: true
};

const defaultProwlarrConfig = {
  url: '',
  api_key: '',
  download_dir: '',
  tv_download_dir: '',
  movie_rename_enabled: false,
  tmdb_api_key: '',
  indexer_ids: [],
  configured: false
};

export async function mockLoggedOut(page: Page) {
  await page.route('**/api/auth/me', (route) =>
    route.fulfill({
      status: 401,
      contentType: 'application/json',
      body: JSON.stringify({ error: '未登录' })
    })
  );
  await page.route('**/api/auth/options', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(defaultAuthOptions)
    })
  );
}

export async function mockLoggedIn(
  page: Page,
  user: { id: number; email: string; feishu_bound: boolean; feishu_name?: string } = {
    id: 1,
    email: 'u@test.dev',
    feishu_bound: true
  }
) {
  await page.route('**/api/auth/me', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(user)
    })
  );
  await page.route('**/api/auth/options', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(defaultAuthOptions)
    })
  );
  await page.route('**/api/subscriptions**', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ items: [], total: 0, page: 1, page_size: 20 })
    })
  );
  await page.route('**/api/settings/proxy', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ proxy_url: '' })
    })
  );
  await page.route('**/api/settings/prowlarr', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(defaultProwlarrConfig)
    })
  );
}

export async function mockFeishuBindUrl(page: Page) {
  await page.route('**/api/settings/feishu-bind-url', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        url: '/api/settings/feishu-bind?state=e2e',
        goto: 'https://passport.feishu.cn/suite/passport/oauth/authorize?state=e2e-bind'
      })
    })
  );
}

export async function mockFeishuLoginUrl(page: Page) {
  await page.route('**/api/auth/feishu/login-url', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        url: '/api/auth/feishu/login?state=e2e-login',
        goto: 'https://passport.feishu.cn/suite/passport/oauth/authorize?state=e2e-login'
      })
    })
  );
}

export async function mockFeishuOnlyLogin(page: Page) {
  await mockLoggedOut(page);
  await page.unroute('**/api/auth/options');
  await page.route('**/api/auth/options', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ password_login_enabled: false, feishu_login_enabled: true })
    })
  );
  await mockFeishuLoginUrl(page);
}

export async function mockLoggedInBound(
  page: Page,
  user: { id: number; email: string; feishu_bound: boolean; feishu_name?: string } = {
    id: 1,
    email: 'u@test.dev',
    feishu_bound: true,
    feishu_name: 'Alice'
  }
) {
  await mockLoggedIn(page, user);
}

export async function mockFeishuUnbind(page: Page) {
  await page.route('**/api/settings/feishu-binding', async (route) => {
    if (route.request().method() === 'DELETE') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ ok: true })
      });
      await page.unroute('**/api/auth/me');
      await page.route('**/api/auth/me', (meRoute) =>
        meRoute.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ id: 1, email: 'u@test.dev', feishu_bound: false })
        })
      );
      return;
    }
    await route.continue();
  });
}

export async function mockLoggedInUnbound(page: Page) {
  await mockLoggedIn(page, { id: 1, email: 'u@test.dev', feishu_bound: false });
  await mockFeishuBindUrl(page);
}

export async function gotoSettings(page: Page) {
  await page.goto('/#settings');
  await expect(page.getByRole('heading', { name: '设置' })).toBeVisible();
}

export async function resetClientState(page: Page) {
  await page.addInitScript(() => {
    localStorage.clear();
    document.documentElement.removeAttribute('data-theme');
    document.documentElement.classList.remove('theme-switching');
  });
}
