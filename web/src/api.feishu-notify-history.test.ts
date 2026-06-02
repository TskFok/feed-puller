import { describe, expect, it, vi, beforeEach } from 'vitest';
import { api } from './api';

describe('api feishu notify history', () => {
  beforeEach(() => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: RequestInfo | URL) => {
        const path = String(input);
        if (path.startsWith('/api/feishu-notify/history?')) {
          return new Response(
            JSON.stringify({
              items: [
                {
                  id: 1,
                  event_type: 'complete',
                  source: 'prowlarr',
                  notify_type: 'webhook',
                  title: '[Prowlarr 完成]',
                  content: '正文',
                  item_count: 1,
                  status: 'sent',
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

  it('loads notify history page', async () => {
    const page = await api.feishuNotifyHistory();
    expect(page.total).toBe(1);
    expect(page.items[0]?.source).toBe('prowlarr');
  });
});
