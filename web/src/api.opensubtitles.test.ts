import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { api } from './api';

describe('api opensubtitles', () => {
  afterEach(() => vi.unstubAllGlobals());
  it('searchSubtitles 带 query 与 languages', async () => {
    vi.stubGlobal('fetch', vi.fn(async (input: RequestInfo | URL) => {
      const path = String(input);
      expect(path).toContain('/api/subtitles/search?');
      expect(path).toContain('query=Inception');
      expect(path).toContain('languages=zh-CN');
      return new Response(JSON.stringify({ items: [{ file_id: 7, file_name: 'a.srt', release: 'Inception', language: 'zh-CN', download_count: 1, ratings: 9 }] }), {
        status: 200, headers: { 'Content-Type': 'application/json' }
      });
    }));
    const res = await api.searchSubtitles('Inception', 'zh-CN');
    expect(res.items[0]?.file_id).toBe(7);
  });
  it('searchSubtitles 提交逗号分隔的多语言', async () => {
    vi.stubGlobal('fetch', vi.fn(async (input: RequestInfo | URL) => {
      const params = new URL(String(input), 'http://local.test').searchParams;
      expect(params.get('languages')).toBe('zh-CN,zh-TW,en');
      return new Response(JSON.stringify({ items: [] }), {
        status: 200, headers: { 'Content-Type': 'application/json' }
      });
    }));
    const res = await api.searchSubtitles('Inception', 'zh-CN,zh-TW,en');
    expect(res.items).toEqual([]);
  });
  it('downloadSubtitle POST file_id 与 file_name', async () => {
    vi.stubGlobal('fetch', vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      expect(String(input)).toBe('/api/subtitles/download');
      expect(init?.method).toBe('POST');
      expect(JSON.parse(String(init?.body))).toEqual({ file_id: 7, file_name: 'a.srt' });
      return new Response(JSON.stringify({ path: '/data/subtitles/a.srt', file_name: 'a.srt' }), {
        status: 200, headers: { 'Content-Type': 'application/json' }
      });
    }));
    const res = await api.downloadSubtitle({ file_id: 7, file_name: 'a.srt' });
    expect(res.path).toBe('/data/subtitles/a.srt');
  });
  it('saveOpenSubtitlesConfig PUT 四项', async () => {
    vi.stubGlobal('fetch', vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      expect(String(input)).toBe('/api/settings/opensubtitles');
      expect(init?.method).toBe('PUT');
      return new Response(JSON.stringify({ username: 'u', password: 'p', api_key: 'k', download_dir: '/d', configured: true }), {
        status: 200, headers: { 'Content-Type': 'application/json' }
      });
    }));
    const saved = await api.saveOpenSubtitlesConfig({ username: 'u', password: 'p', api_key: 'k', download_dir: '/d' });
    expect(saved.configured).toBe(true);
  });
});
