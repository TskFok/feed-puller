import { describe, expect, it, vi, beforeEach } from 'vitest';
import { api } from './api';

describe('api rename history', () => {
  beforeEach(() => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: RequestInfo | URL) => {
        const path = String(input);
        if (path.startsWith('/api/rename-history?')) {
          return new Response(
            JSON.stringify({
              items: [
                {
                  id: 1,
                  subscription_id: 2,
                  original_filename: '番剧 第02话.mp4',
                  original_path: '/data/anime/番剧 第02话.mp4',
                  renamed_path: '/data/anime/番剧 第02话 S01E02.mp4',
                  ai_prompt: 'prompt',
                  ai_response: '{"episode":2}',
                  status: 'success',
                  created_at: '2026-06-02T00:00:00Z'
                }
              ],
              total: 1,
              page: 1,
              page_size: 30
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } }
          );
        }
        return new Response(JSON.stringify({ error: 'unexpected' }), { status: 404 });
      })
    );
  });

  it('loads rename history page', async () => {
    const page = await api.renameHistory();
    expect(page.total).toBe(1);
    expect(page.items[0]?.original_filename).toBe('番剧 第02话.mp4');
    expect(page.items[0]?.ai_response).toBe('{"episode":2}');
  });
});
