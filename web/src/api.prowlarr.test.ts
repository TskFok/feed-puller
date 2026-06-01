import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { api } from './api';

describe('api prowlarr', () => {
  beforeEach(() => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
        const path = String(input);
        if (path.startsWith('/api/prowlarr/search?')) {
          expect(path).toContain('query=inception');
          expect(path).toContain('type=movie');
          expect(path).toContain('sort=seeders');
          return new Response(JSON.stringify({ items: [{ guid: 'g1', title: 'Inception', protocol: 'torrent' }] }), {
            status: 200,
            headers: { 'Content-Type': 'application/json' }
          });
        }
        if (path.startsWith('/api/prowlarr/search-history')) {
          if (init?.method === 'DELETE' && path === '/api/prowlarr/search-history') {
            return new Response(JSON.stringify({ ok: true }), { status: 200, headers: { 'Content-Type': 'application/json' } });
          }
          return new Response(JSON.stringify({ items: [{ id: 1, display_query: 'Inception', query: 'inception', media_type: 'movie', sort_by: 'seeders', indexer_ids: [], result_count: 1, searched_at: '2026-01-01T00:00:00Z' }] }), {
            status: 200,
            headers: { 'Content-Type': 'application/json' }
          });
        }
        if (path === '/api/prowlarr/indexers') {
          return new Response(JSON.stringify({ items: [{ id: 1, name: 'Tracker', enable: true, protocol: 'torrent' }] }), {
            status: 200,
            headers: { 'Content-Type': 'application/json' }
          });
        }
        if (path === '/api/prowlarr/download/batch' && init?.method === 'POST') {
          const body = JSON.parse(String(init.body));
          expect(body.releases).toHaveLength(2);
          return new Response(JSON.stringify({ items: [{ id: 1, download_status: 'submitted' }], failures: [{ guid: 'g2', error: '该资源正在下载中' }] }), {
            status: 200,
            headers: { 'Content-Type': 'application/json' }
          });
        }
        if (path === '/api/prowlarr/download' && init?.method === 'POST') {
          const body = JSON.parse(String(init.body));
          expect(body.guid).toBe('g1');
          return new Response(JSON.stringify({ id: 1, title: 'Inception', download_status: 'submitted' }), {
            status: 200,
            headers: { 'Content-Type': 'application/json' }
          });
        }
        if (path === '/api/settings/prowlarr' && init?.method === 'PUT') {
          return new Response(
            JSON.stringify({
              url: 'http://127.0.0.1:9696',
              api_key: 'k',
              download_dir: '/movies',
              tv_download_dir: '/tv',
              movie_rename_enabled: true,
              tmdb_api_key: 'tmdb',
              indexer_ids: [1],
              configured: true
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } }
          );
        }
        return new Response(JSON.stringify({ error: 'not found' }), { status: 404 });
      })
    );
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('searchProwlarr 请求搜索接口', async () => {
    const res = await api.searchProwlarr('inception', { type: 'movie', sort: 'seeders' });
    expect(res.items).toHaveLength(1);
    expect(res.items[0].guid).toBe('g1');
  });

  it('searchProwlarr 保留空索引器选择', async () => {
    await api.searchProwlarr('inception', { type: 'movie', sort: 'seeders', indexerIds: [] });
    const path = String(vi.mocked(fetch).mock.calls[0][0]);
    expect(path).toContain('indexer_ids=');
  });

  it('prowlarrSearchHistory 请求搜索历史', async () => {
    const res = await api.prowlarrSearchHistory();
    expect(res.items[0].display_query).toBe('Inception');
  });

  it('getProwlarrSearchHistory 请求单条历史及缓存结果', async () => {
    vi.mocked(fetch).mockImplementation(async (input: RequestInfo | URL) => {
      const path = String(input);
      if (path === '/api/prowlarr/search-history/1') {
        return new Response(
          JSON.stringify({
            id: 1,
            display_query: 'Inception',
            query: 'inception',
            media_type: 'movie',
            sort_by: 'seeders',
            indexer_ids: [],
            result_count: 1,
            searched_at: '2026-01-01T00:00:00Z',
            results: [{ guid: 'g1', title: 'Inception', protocol: 'torrent' }]
          }),
          { status: 200, headers: { 'Content-Type': 'application/json' } }
        );
      }
      return new Response(JSON.stringify({ error: 'not found' }), { status: 404 });
    });
    const res = await api.getProwlarrSearchHistory(1);
    expect(res.results).toHaveLength(1);
    expect(res.results[0].guid).toBe('g1');
  });

  it('batchDownloadProwlarrReleases 批量提交下载', async () => {
    const res = await api.batchDownloadProwlarrReleases([
      { guid: 'g1', title: 'A' },
      { guid: 'g2', title: 'B' }
    ]);
    expect(res.items).toHaveLength(1);
    expect(res.failures).toHaveLength(1);
  });

  it('clearProwlarrSearchHistory 清空历史', async () => {
    const res = await api.clearProwlarrSearchHistory();
    expect(res.ok).toBe(true);
  });

  it('prowlarrSubmittedGuids 查询已提交 guid', async () => {
    vi.mocked(fetch).mockImplementation(async (input: RequestInfo | URL, init?: RequestInit) => {
      const path = String(input);
      if (path === '/api/prowlarr/submitted-guids' && init?.method === 'POST') {
        const body = JSON.parse(String(init.body));
        expect(body.guids).toEqual(['g1', 'g2']);
        return new Response(JSON.stringify({ guids: ['g1'] }), {
          status: 200,
          headers: { 'Content-Type': 'application/json' }
        });
      }
      return new Response(JSON.stringify({ error: 'not found' }), { status: 404 });
    });
    const res = await api.prowlarrSubmittedGuids(['g1', 'g2']);
    expect(res.guids).toEqual(['g1']);
  });
});
