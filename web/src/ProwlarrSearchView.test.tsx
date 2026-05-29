import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { ProwlarrSearchView } from './ProwlarrSearchView';
import { PROWLARR_SUBMITTED_STORAGE_KEY } from './prowlarrSubmittedGuids';
import { ToastProvider } from './Toast';

function submittedGuidsResponse(guids: string[] = []) {
  return new Response(JSON.stringify({ guids }), {
    status: 200,
    headers: { 'Content-Type': 'application/json' }
  });
}

describe('ProwlarrSearchView', () => {
  beforeEach(() => {
    sessionStorage.clear();
    vi.stubGlobal('fetch', vi.fn());
  });

  afterEach(() => {
    sessionStorage.clear();
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
      if (path === '/api/prowlarr/submitted-guids') {
        return submittedGuidsResponse();
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
      if (path === '/api/prowlarr/submitted-guids' && init?.method === 'POST') {
        return submittedGuidsResponse();
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

  it('下载成功后停留在当前搜索页', async () => {
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
        return new Response(JSON.stringify({ id: 1, title: 'Inception 2010', download_status: 'submitted' }), {
          status: 200,
          headers: { 'Content-Type': 'application/json' }
        });
      }
      if (path === '/api/prowlarr/submitted-guids' && init?.method === 'POST') {
        return submittedGuidsResponse();
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

    fireEvent.click(screen.getByRole('button', { name: '下载' }));

    await waitFor(() => expect(screen.getByText('已提交下载')).toBeInTheDocument());
    expect(screen.getByRole('heading', { name: 'Prowlarr 搜索' })).toBeInTheDocument();
    expect(screen.getByText('Inception 2010')).toBeInTheDocument();
  });

  it('下载成功后标记卡片为已提交且不可重复下载', async () => {
    let downloadCalls = 0;
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
        downloadCalls += 1;
        return new Response(JSON.stringify({ id: 1, title: 'Inception 2010', download_status: 'submitted' }), {
          status: 200,
          headers: { 'Content-Type': 'application/json' }
        });
      }
      if (path === '/api/prowlarr/submitted-guids' && init?.method === 'POST') {
        return submittedGuidsResponse();
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

    const downloadButton = screen.getByRole('button', { name: '下载' });
    fireEvent.click(downloadButton);

    await waitFor(() => expect(screen.getByRole('article')).toHaveClass('prowlarr-release-card--submitted'));
    expect(screen.getByRole('button', { name: '已提交' })).toBeDisabled();
    expect(screen.getAllByText('已提交')).toHaveLength(2);

    fireEvent.click(screen.getByRole('button', { name: '已提交' }));
    expect(downloadCalls).toBe(1);
  });

  it('下载成功 Toast 可跳转查看进度', async () => {
    const onGoActive = vi.fn();
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
        return new Response(JSON.stringify({ id: 1, title: 'Inception 2010', download_status: 'submitted' }), {
          status: 200,
          headers: { 'Content-Type': 'application/json' }
        });
      }
      if (path === '/api/prowlarr/submitted-guids' && init?.method === 'POST') {
        return submittedGuidsResponse();
      }
      return new Response(JSON.stringify({ error: 'not found' }), { status: 404 });
    });

    render(
      <ToastProvider>
        <ProwlarrSearchView onGoActive={onGoActive} />
      </ToastProvider>
    );

    fireEvent.change(screen.getByLabelText('关键词'), { target: { value: 'Inception' } });
    fireEvent.click(screen.getByRole('button', { name: '搜索' }));
    expect(await screen.findByText('Inception 2010')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: '下载' }));

    await waitFor(() => expect(screen.getByRole('button', { name: '查看进度' })).toBeInTheDocument());
    fireEvent.click(screen.getByRole('button', { name: '查看进度' }));
    expect(onGoActive).toHaveBeenCalledTimes(1);
  });

  it('批量下载后在工具栏显示成功与失败摘要', async () => {
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
        return new Response(
          JSON.stringify({
            items: [
              {
                guid: 'guid-1',
                title: 'Release One',
                indexer: 'Tracker',
                indexerId: 1,
                size: 1024,
                seeders: 5,
                leechers: 0,
                protocol: 'torrent',
                infoHash: 'abc'
              },
              {
                guid: 'guid-2',
                title: 'Release Two',
                indexer: 'Tracker',
                indexerId: 1,
                size: 2048,
                seeders: 3,
                leechers: 0,
                protocol: 'torrent',
                infoHash: 'def'
              }
            ]
          }),
          { status: 200, headers: { 'Content-Type': 'application/json' } }
        );
      }
      if (path === '/api/prowlarr/download/batch' && init?.method === 'POST') {
        return new Response(
          JSON.stringify({
            items: [{ id: 1, title: 'Release One', download_status: 'submitted', subscription_id: 0, created_at: '' }],
            failures: [{ guid: 'guid-2', error: '该资源正在下载中' }]
          }),
          { status: 200, headers: { 'Content-Type': 'application/json' } }
        );
      }
      if (path === '/api/prowlarr/submitted-guids' && init?.method === 'POST') {
        return submittedGuidsResponse();
      }
      return new Response(JSON.stringify({ error: 'not found' }), { status: 404 });
    });

    render(
      <ToastProvider>
        <ProwlarrSearchView />
      </ToastProvider>
    );

    fireEvent.change(screen.getByLabelText('关键词'), { target: { value: 'Test' } });
    fireEvent.click(screen.getByRole('button', { name: '搜索' }));
    expect(await screen.findByText('Release One')).toBeInTheDocument();

    fireEvent.click(screen.getByRole('checkbox', { name: /全选/ }));
    fireEvent.click(screen.getByRole('button', { name: '批量下载' }));

    await waitFor(() => expect(screen.getByText('本次提交：成功 1 条，失败 1 条')).toBeInTheDocument());
    expect(screen.getByRole('button', { name: '收起失败原因' })).toBeInTheDocument();
    expect(screen.getByText('该资源正在下载中')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '已提交' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '下载' })).toBeInTheDocument();
  });

  it('搜索后从后端恢复已提交状态', async () => {
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
      if (path === '/api/prowlarr/submitted-guids' && init?.method === 'POST') {
        return submittedGuidsResponse(['movie-guid']);
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

    await waitFor(() => expect(screen.getByRole('article')).toHaveClass('prowlarr-release-card--submitted'));
    expect(screen.getByRole('button', { name: '已提交' })).toBeDisabled();
  });

  it('sessionStorage 中的已提交 guid 会在搜索后合并展示', async () => {
    sessionStorage.setItem(PROWLARR_SUBMITTED_STORAGE_KEY, JSON.stringify(['movie-guid']));
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
      if (path === '/api/prowlarr/submitted-guids' && init?.method === 'POST') {
        return submittedGuidsResponse();
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

    await waitFor(() => expect(screen.getByRole('button', { name: '已提交' })).toBeDisabled());
  });
});
