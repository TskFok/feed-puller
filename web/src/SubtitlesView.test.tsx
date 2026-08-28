import { render, screen, fireEvent } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { SubtitlesView } from './SubtitlesView';
import { ToastProvider } from './Toast';

describe('SubtitlesView', () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('配置未加载完时不展示搜索表单', () => {
    vi.stubGlobal('fetch', vi.fn(() => new Promise<Response>(() => {})));
    render(
      <ToastProvider>
        <SubtitlesView />
      </ToastProvider>
    );
    expect(screen.queryByRole('button', { name: '搜索' })).not.toBeInTheDocument();
    expect(screen.queryByLabelText('名称')).not.toBeInTheDocument();
  });

  it('未配置时显示前往设置', async () => {
    vi.stubGlobal('fetch', vi.fn(async (input: RequestInfo | URL) => {
      if (String(input) === '/api/settings/opensubtitles') {
        return new Response(JSON.stringify({ username: '', password: '', api_key: '', download_dir: '', configured: false }), {
          status: 200, headers: { 'Content-Type': 'application/json' }
        });
      }
      return new Response('{}', { status: 200 });
    }));
    const onGoSettings = vi.fn();
    render(<ToastProvider><SubtitlesView onGoSettings={onGoSettings} /></ToastProvider>);
    fireEvent.click(await screen.findByRole('button', { name: '前往设置' }));
    expect(onGoSettings).toHaveBeenCalled();
  });

  it('配置加载失败时显示前往设置', async () => {
    vi.stubGlobal('fetch', vi.fn(async (input: RequestInfo | URL) => {
      if (String(input) === '/api/settings/opensubtitles') {
        return new Response(JSON.stringify({ error: 'boom' }), {
          status: 500, headers: { 'Content-Type': 'application/json' }
        });
      }
      return new Response('{}', { status: 200 });
    }));
    const onGoSettings = vi.fn();
    render(<ToastProvider><SubtitlesView onGoSettings={onGoSettings} /></ToastProvider>);
    expect(await screen.findByRole('alert')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: '搜索' })).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: '前往设置' }));
    expect(onGoSettings).toHaveBeenCalled();
  });

  it('搜索列出结果并下载', async () => {
    vi.stubGlobal('fetch', vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const path = String(input);
      if (path === '/api/settings/opensubtitles') {
        return new Response(JSON.stringify({ username: 'u', password: 'p', api_key: 'k', download_dir: '/data/subtitles', configured: true }), {
          status: 200, headers: { 'Content-Type': 'application/json' }
        });
      }
      if (path.startsWith('/api/subtitles/search?')) {
        expect(path).toContain('query=Inception');
        expect(path).toContain('languages=zh-CN');
        return new Response(JSON.stringify({
          items: [{ file_id: 7, file_name: 'a.srt', release: 'Inception.2024', language: 'zh-CN', download_count: 12, ratings: 8.5 }]
        }), { status: 200, headers: { 'Content-Type': 'application/json' } });
      }
      if (path === '/api/subtitles/download' && init?.method === 'POST') {
        expect(JSON.parse(String(init.body))).toEqual({ file_id: 7, file_name: 'a.srt' });
        return new Response(JSON.stringify({ path: '/data/subtitles/a.srt', file_name: 'a.srt' }), {
          status: 200, headers: { 'Content-Type': 'application/json' }
        });
      }
      return new Response('{}', { status: 200 });
    }));
    render(<ToastProvider><SubtitlesView /></ToastProvider>);
    await screen.findByRole('button', { name: '搜索' });
    fireEvent.change(screen.getByLabelText('名称'), { target: { value: 'Inception' } });
    fireEvent.click(screen.getByRole('button', { name: '搜索' }));
    expect(await screen.findByText('Inception.2024')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: '下载' }));
    expect(await screen.findByText('已保存到 /data/subtitles/a.srt')).toBeInTheDocument();
  });

  it('可多选语言并按固定顺序提交', async () => {
    vi.stubGlobal('fetch', vi.fn(async (input: RequestInfo | URL) => {
      const path = String(input);
      if (path === '/api/settings/opensubtitles') {
        return new Response(JSON.stringify({ username: 'u', password: 'p', api_key: 'k', download_dir: '/data/subtitles', configured: true }), {
          status: 200, headers: { 'Content-Type': 'application/json' }
        });
      }
      if (path.startsWith('/api/subtitles/search?')) {
        const params = new URL(path, 'http://local.test').searchParams;
        expect(params.get('query')).toBe('Inception');
        expect(params.get('languages')).toBe('zh-CN,zh-TW,en');
        return new Response(JSON.stringify({
          items: [{ file_id: 8, file_name: 'b.srt', release: 'Inception.multi', language: 'en', download_count: 1, ratings: 7 }]
        }), { status: 200, headers: { 'Content-Type': 'application/json' } });
      }
      return new Response('{}', { status: 200 });
    }));
    render(<ToastProvider><SubtitlesView /></ToastProvider>);
    await screen.findByRole('button', { name: '搜索' });
    expect(screen.getByLabelText('简体中文')).toBeChecked();
    fireEvent.click(screen.getByLabelText('繁体中文'));
    fireEvent.click(screen.getByLabelText('英语'));
    fireEvent.change(screen.getByLabelText('名称'), { target: { value: 'Inception' } });
    fireEvent.click(screen.getByRole('button', { name: '搜索' }));
    expect(await screen.findByText('Inception.multi')).toBeInTheDocument();
  });

  it('未选择语言时不能搜索', async () => {
    const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
      const path = String(input);
      if (path === '/api/settings/opensubtitles') {
        return new Response(JSON.stringify({ username: 'u', password: 'p', api_key: 'k', download_dir: '/data/subtitles', configured: true }), {
          status: 200, headers: { 'Content-Type': 'application/json' }
        });
      }
      return new Response('{}', { status: 200 });
    });
    vi.stubGlobal('fetch', fetchMock);
    render(<ToastProvider><SubtitlesView /></ToastProvider>);
    await screen.findByRole('button', { name: '搜索' });
    fireEvent.click(screen.getByLabelText('简体中文'));
    fireEvent.change(screen.getByLabelText('名称'), { target: { value: 'Inception' } });
    expect(screen.getByRole('button', { name: '搜索' })).toBeDisabled();
    fireEvent.click(screen.getByRole('button', { name: '搜索' }));
    expect(fetchMock.mock.calls.some(([input]) => String(input).includes('/api/subtitles/search'))).toBe(false);
  });

  it('发行名和文件名过长时截断并悬停显示全文', async () => {
    const release = 'Inception.2010.2160p.UHD.BluRay.REMUX.HEVC.DTS-HD.MA.TrueHD.7.1.Atmos-GROUP';
    const fileName = 'Inception.2010.2160p.UHD.BluRay.REMUX.HEVC.DTS-HD.MA.TrueHD.7.1.Atmos-GROUP.zh-CN.srt';
    vi.stubGlobal('fetch', vi.fn(async (input: RequestInfo | URL) => {
      const path = String(input);
      if (path === '/api/settings/opensubtitles') {
        return new Response(JSON.stringify({ username: 'u', password: 'p', api_key: 'k', download_dir: '/data/subtitles', configured: true }), {
          status: 200, headers: { 'Content-Type': 'application/json' }
        });
      }
      if (path.startsWith('/api/subtitles/search?')) {
        return new Response(JSON.stringify({
          items: [{ file_id: 9, file_name: fileName, release, language: 'zh-CN', download_count: 3, ratings: 9.1 }]
        }), { status: 200, headers: { 'Content-Type': 'application/json' } });
      }
      return new Response('{}', { status: 200 });
    }));
    render(<ToastProvider><SubtitlesView /></ToastProvider>);
    await screen.findByRole('button', { name: '搜索' });
    fireEvent.change(screen.getByLabelText('名称'), { target: { value: 'Inception' } });
    fireEvent.click(screen.getByRole('button', { name: '搜索' }));

    const releaseText = await screen.findByTitle(release);
    expect(releaseText).toHaveClass('truncated-text');
    expect(releaseText).toHaveTextContent(release);
    const fileText = screen.getByTitle(fileName);
    expect(fileText).toHaveClass('truncated-text');
    expect(fileText).toHaveTextContent(fileName);
  });

  it('结果表加宽发行名和文件名列并收窄语言次数评分', async () => {
    vi.stubGlobal('fetch', vi.fn(async (input: RequestInfo | URL) => {
      const path = String(input);
      if (path === '/api/settings/opensubtitles') {
        return new Response(JSON.stringify({ username: 'u', password: 'p', api_key: 'k', download_dir: '/data/subtitles', configured: true }), {
          status: 200, headers: { 'Content-Type': 'application/json' }
        });
      }
      if (path.startsWith('/api/subtitles/search?')) {
        return new Response(JSON.stringify({
          items: [{ file_id: 7, file_name: 'a.srt', release: 'Inception.2024', language: 'zh-CN', download_count: 12, ratings: 8.5 }]
        }), { status: 200, headers: { 'Content-Type': 'application/json' } });
      }
      return new Response('{}', { status: 200 });
    }));
    render(<ToastProvider><SubtitlesView /></ToastProvider>);
    await screen.findByRole('button', { name: '搜索' });
    fireEvent.change(screen.getByLabelText('名称'), { target: { value: 'Inception' } });
    fireEvent.click(screen.getByRole('button', { name: '搜索' }));

    const table = await screen.findByRole('table');
    expect(table).toHaveClass('subtitles-results');
    expect([...table.querySelectorAll('col')].map((col) => col.className)).toEqual([
      'subtitles-col-release',
      'subtitles-col-language',
      'subtitles-col-file',
      'subtitles-col-count',
      'subtitles-col-rating',
      'subtitles-col-actions'
    ]);
  });

  it('搜索失败时清空上次结果', async () => {
    let searchCount = 0;
    vi.stubGlobal('fetch', vi.fn(async (input: RequestInfo | URL) => {
      const path = String(input);
      if (path === '/api/settings/opensubtitles') {
        return new Response(JSON.stringify({ username: 'u', password: 'p', api_key: 'k', download_dir: '/data/subtitles', configured: true }), {
          status: 200, headers: { 'Content-Type': 'application/json' }
        });
      }
      if (path.startsWith('/api/subtitles/search?')) {
        searchCount += 1;
        if (searchCount === 1) {
          return new Response(JSON.stringify({
            items: [{ file_id: 7, file_name: 'a.srt', release: 'Inception.2024', language: 'zh-CN', download_count: 12, ratings: 8.5 }]
          }), { status: 200, headers: { 'Content-Type': 'application/json' } });
        }
        return new Response(JSON.stringify({ error: '搜索字幕失败' }), {
          status: 502, headers: { 'Content-Type': 'application/json' }
        });
      }
      return new Response('{}', { status: 200 });
    }));
    render(<ToastProvider><SubtitlesView /></ToastProvider>);
    await screen.findByRole('button', { name: '搜索' });
    fireEvent.change(screen.getByLabelText('名称'), { target: { value: 'Inception' } });
    fireEvent.click(screen.getByRole('button', { name: '搜索' }));
    expect(await screen.findByText('Inception.2024')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: '搜索' }));
    expect(await screen.findByRole('alert')).toHaveTextContent('搜索字幕失败');
    expect(screen.queryByText('Inception.2024')).not.toBeInTheDocument();
  });

  it('混合结果按语言分组，并可筛选只看某一种语言', async () => {
    vi.stubGlobal('fetch', vi.fn(async (input: RequestInfo | URL) => {
      const path = String(input);
      if (path === '/api/settings/opensubtitles') {
        return new Response(JSON.stringify({ username: 'u', password: 'p', api_key: 'k', download_dir: '/data/subtitles', configured: true }), {
          status: 200, headers: { 'Content-Type': 'application/json' }
        });
      }
      if (path.startsWith('/api/subtitles/search?')) {
        return new Response(JSON.stringify({
          items: [
            { file_id: 1, file_name: 'cn.srt', release: 'cn-rel', language: 'zh-CN', download_count: 2, ratings: 8 },
            { file_id: 2, file_name: 'en.srt', release: 'en-rel', language: 'en', download_count: 5, ratings: 7 },
            { file_id: 3, file_name: 'cn2.srt', release: 'cn-rel-2', language: 'zh-cn', download_count: 9, ratings: 6 }
          ]
        }), { status: 200, headers: { 'Content-Type': 'application/json' } });
      }
      return new Response('{}', { status: 200 });
    }));
    render(<ToastProvider><SubtitlesView /></ToastProvider>);
    await screen.findByRole('button', { name: '搜索' });
    fireEvent.change(screen.getByLabelText('名称'), { target: { value: 'Inception' } });
    fireEvent.click(screen.getByRole('button', { name: '搜索' }));

    expect(await screen.findByText('cn-rel')).toBeInTheDocument();
    expect(screen.getByText('en-rel')).toBeInTheDocument();
    const releaseCells = [...screen.getByRole('table').querySelectorAll('tbody td:first-child')].map((cell) => cell.textContent);
    expect(releaseCells).toEqual(['cn-rel-2', 'cn-rel', 'en-rel']);
    expect(screen.getByRole('button', { name: '简体中文（2）' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '英语（1）' })).toBeInTheDocument();
    expect(screen.getByRole('table').querySelector('.subtitles-language-group')).toHaveTextContent('简体中文（2）');
    const allTab = screen.getByRole('button', { name: '全部（3）' });
    expect(allTab).toHaveAttribute('aria-pressed', 'true');
    expect(screen.getByRole('group', { name: '按语言筛选' })).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: '英语（1）' }));
    expect(screen.getByText('en-rel')).toBeInTheDocument();
    expect(screen.queryByText('cn-rel')).not.toBeInTheDocument();
    expect(screen.queryByText('cn-rel-2')).not.toBeInTheDocument();
    expect(screen.getByRole('button', { name: '英语（1）' })).toHaveAttribute('aria-pressed', 'true');

    fireEvent.click(allTab);
    expect(screen.getByText('cn-rel')).toBeInTheDocument();
    expect(screen.getByText('en-rel')).toBeInTheDocument();
  });

  it('单一语言结果不展示筛选', async () => {
    vi.stubGlobal('fetch', vi.fn(async (input: RequestInfo | URL) => {
      const path = String(input);
      if (path === '/api/settings/opensubtitles') {
        return new Response(JSON.stringify({ username: 'u', password: 'p', api_key: 'k', download_dir: '/data/subtitles', configured: true }), {
          status: 200, headers: { 'Content-Type': 'application/json' }
        });
      }
      if (path.startsWith('/api/subtitles/search?')) {
        return new Response(JSON.stringify({
          items: [{ file_id: 7, file_name: 'a.srt', release: 'Inception.2024', language: 'zh-CN', download_count: 12, ratings: 8.5 }]
        }), { status: 200, headers: { 'Content-Type': 'application/json' } });
      }
      return new Response('{}', { status: 200 });
    }));
    render(<ToastProvider><SubtitlesView /></ToastProvider>);
    await screen.findByRole('button', { name: '搜索' });
    fireEvent.change(screen.getByLabelText('名称'), { target: { value: 'Inception' } });
    fireEvent.click(screen.getByRole('button', { name: '搜索' }));
    expect(await screen.findByText('Inception.2024')).toBeInTheDocument();
    expect(screen.queryByRole('group', { name: '按语言筛选' })).not.toBeInTheDocument();
  });
});
