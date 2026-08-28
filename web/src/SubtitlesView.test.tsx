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
});
