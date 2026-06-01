import type { Page } from '@playwright/test';

/** 检测可见虚拟行是否发生纵向重叠（2px 容差） */
export async function virtualProwlarrRowsOverlap(page: Page): Promise<boolean> {
  return page.evaluate(() => {
    const rows = Array.from(document.querySelectorAll<HTMLElement>('.prowlarr-results-virtual-row[data-index]'));
    if (rows.length < 2) {
      return false;
    }
    const sorted = rows
      .map((row) => {
        const rect = row.getBoundingClientRect();
        return { top: rect.top, bottom: rect.bottom };
      })
      .sort((a, b) => a.top - b.top);
    for (let i = 1; i < sorted.length; i += 1) {
      if (sorted[i].top < sorted[i - 1].bottom - 2) {
        return true;
      }
    }
    return false;
  });
}

export function makeProwlarrRelease(index: number) {
  return {
    guid: `guid-${index}`,
    title: `Release ${index}`,
    indexer: 'Tracker',
    indexerId: 1,
    size: 1024 * (index + 1),
    seeders: index % 20,
    leechers: 0,
    protocol: 'torrent',
    downloadUrl: `https://example.test/d/${index}`,
    infoHash: `hash${index}`,
    publishDate: '2026-01-01T00:00:00Z'
  };
}

export async function mockProwlarrConfigured(page: Page) {
  await page.route('**/api/settings/prowlarr', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        url: 'http://127.0.0.1:9696',
        api_key: 'k',
        download_dir: '/movies',
        tv_download_dir: '/tv',
        movie_rename_enabled: true,
        tmdb_api_key: '',
        indexer_ids: [],
        configured: true
      })
    })
  );
  await page.route('**/api/prowlarr/indexers', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ items: [] })
    })
  );
  await page.route('**/api/prowlarr/search-history**', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ items: [] })
    })
  );
  await page.route('**/api/prowlarr/submitted-guids', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ guids: [] })
    });
  });
}

export async function mockProwlarrSearchResults(page: Page, count: number) {
  const items = Array.from({ length: count }, (_, i) => makeProwlarrRelease(i));
  await page.route('**/api/prowlarr/search?**', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ items })
    })
  );
}

export async function mockProwlarrBatchDownload(
  page: Page,
  options: { successCount?: number; failures?: { guid: string; error: string }[] } = {}
) {
  const { successCount = 1, failures = [{ guid: 'guid-1', error: '该资源正在下载中' }] } = options;
  await page.route('**/api/prowlarr/download/batch', async (route) => {
    if (route.request().method() !== 'POST') {
      await route.continue();
      return;
    }
    const body = route.request().postDataJSON() as { releases?: { guid: string; title: string }[] };
    const items = (body.releases ?? []).slice(0, successCount).map((item, index) => ({
      id: index + 1,
      title: item.title,
      download_status: 'submitted',
      subscription_id: 0,
      created_at: '2026-01-01T00:00:00Z'
    }));
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ items, failures })
    });
  });
}
