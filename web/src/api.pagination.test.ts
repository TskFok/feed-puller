import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { api } from './api';

describe('api paginated list', () => {
  beforeEach(() => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: RequestInfo | URL) => {
        const path = String(input);
        if (path.startsWith('/api/subscriptions?')) {
          return new Response(
            JSON.stringify({ items: [{ id: 1, name: 'A' }], total: 1, page: 1, page_size: 30 }),
            { status: 200, headers: { 'Content-Type': 'application/json' } }
          );
        }
        if (path === '/api/subscriptions') {
          return new Response(JSON.stringify([{ id: 2, name: 'legacy' }]), {
            status: 200,
            headers: { 'Content-Type': 'application/json' }
          });
        }
        return new Response(JSON.stringify({}), { status: 404 });
      })
    );
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('解析标准分页响应', async () => {
    const res = await api.subscriptions(1, 30);
    expect(res.items).toHaveLength(1);
    expect(res.total).toBe(1);
  });

  it('兼容旧版数组响应', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async () =>
        new Response(JSON.stringify([{ id: 3, name: 'B' }]), {
          status: 200,
          headers: { 'Content-Type': 'application/json' }
        })
      )
    );
    const res = await api.subscriptions();
    expect(res.items).toHaveLength(1);
    expect(res.total).toBe(1);
  });
});
