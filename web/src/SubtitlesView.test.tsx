import { render, screen, fireEvent } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { SubtitlesView } from './SubtitlesView';
import { ToastProvider } from './Toast';

describe('SubtitlesView', () => {
  afterEach(() => {
    vi.unstubAllGlobals();
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
    await screen.findByRole('heading', { name: '字幕' });
    fireEvent.change(screen.getByLabelText('名称'), { target: { value: 'Inception' } });
    fireEvent.click(screen.getByRole('button', { name: '搜索' }));
    expect(await screen.findByText('Inception.2024')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: '下载' }));
    expect(await screen.findByText('已保存到 /data/subtitles/a.srt')).toBeInTheDocument();
  });
});
