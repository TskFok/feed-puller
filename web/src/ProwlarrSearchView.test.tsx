import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { ProwlarrSearchView } from './ProwlarrSearchView';
import { ToastProvider } from './Toast';

describe('ProwlarrSearchView', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn());
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('搜索中显示骨架屏卡片', async () => {
    let resolveSearch: (value: Response) => void = () => undefined;
    const searchPromise = new Promise<Response>((resolve) => {
      resolveSearch = resolve;
    });
    vi.mocked(fetch).mockImplementation(async (input: RequestInfo | URL) => {
      const path = String(input);
      if (path === '/api/settings/prowlarr') {
        return new Response(
          JSON.stringify({
            url: 'http://127.0.0.1:9696',
            api_key: 'k',
            download_dir: '/movies',
            tv_download_dir: '/tv',
            movie_rename_enabled: true,
            tmdb_api_key: '',
            indexer_ids: [],
            configured: true
          }),
          { status: 200, headers: { 'Content-Type': 'application/json' } }
        );
      }
      if (path.startsWith('/api/prowlarr/search?')) {
        return searchPromise;
      }
      if (path.startsWith('/api/prowlarr/search-history')) {
        return new Response(JSON.stringify({ items: [] }), { status: 200, headers: { 'Content-Type': 'application/json' } });
      }
      return new Response(JSON.stringify({}), { status: 200 });
    });

    render(
      <ToastProvider>
        <ProwlarrSearchView />
      </ToastProvider>
    );

    fireEvent.change(screen.getByLabelText('关键词'), { target: { value: 'Inception' } });
    fireEvent.click(screen.getByRole('button', { name: '搜索' }));

    expect(document.querySelectorAll('.prowlarr-release-card--skeleton')).toHaveLength(6);

    resolveSearch(
      new Response(JSON.stringify({ items: [] }), { status: 200, headers: { 'Content-Type': 'application/json' } })
    );
    await waitFor(() => expect(document.querySelectorAll('.prowlarr-release-card--skeleton')).toHaveLength(0));
  });

  it('下载使用产生当前结果时的搜索类型', async () => {
    const downloadBodies: unknown[] = [];
    vi.mocked(fetch).mockImplementation(async (input: RequestInfo | URL, init?: RequestInit) => {
      const path = String(input);
      if (path === '/api/settings/prowlarr') {
        return new Response(
          JSON.stringify({
            url: 'http://127.0.0.1:9696',
            api_key: 'k',
            download_dir: '/movies',
            tv_download_dir: '/tv',
            movie_rename_enabled: true,
            tmdb_api_key: '',
            indexer_ids: [],
            configured: true
          }),
          { status: 200, headers: { 'Content-Type': 'application/json' } }
        );
      }
      if (path === '/api/prowlarr/indexers') {
        return new Response(JSON.stringify({ items: [] }), {
          status: 200,
          headers: { 'Content-Type': 'application/json' }
        });
      }
      if (path.startsWith('/api/prowlarr/search-history')) {
        return new Response(JSON.stringify({ items: [] }), {
          status: 200,
          headers: { 'Content-Type': 'application/json' }
        });
      }
      if (path.startsWith('/api/prowlarr/search?')) {
        expect(path).toContain('type=movie');
        return new Response(
          JSON.stringify({
            items: [
              {
                guid: 'movie-guid',
                title: 'Inception 2010',
                indexer: 'Tracker',
                indexerId: 1,
                size: 1024,
                seeders: 5,
                leechers: 0,
                protocol: 'torrent',
                infoHash: 'abc'
              }
            ]
          }),
          { status: 200, headers: { 'Content-Type': 'application/json' } }
        );
      }
      if (path === '/api/prowlarr/download' && init?.method === 'POST') {
        downloadBodies.push(JSON.parse(String(init.body)));
        return new Response(JSON.stringify({ id: 1, title: 'Inception 2010', download_status: 'submitted' }), {
          status: 200,
          headers: { 'Content-Type': 'application/json' }
        });
      }
      return new Response(JSON.stringify({ error: 'not found' }), { status: 404 });
    });

    render(
      <ToastProvider>
        <ProwlarrSearchView />
      </ToastProvider>
    );

    fireEvent.change(screen.getByLabelText('关键词'), { target: { value: 'Inception' } });
    fireEvent.click(screen.getByRole('button', { name: '搜索' }));
    expect(await screen.findByText('Inception 2010')).toBeInTheDocument();
    expect(screen.getByRole('article')).toHaveClass('prowlarr-release-card');

    fireEvent.change(screen.getByLabelText('类型'), { target: { value: 'tv' } });
    fireEvent.click(screen.getByRole('button', { name: '下载' }));

    await waitFor(() => expect(downloadBodies).toHaveLength(1));
    expect(downloadBodies[0]).toMatchObject({
      guid: 'movie-guid',
      media_type: 'movie'
    });
  });
});
