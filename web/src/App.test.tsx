import { render, screen, fireEvent, waitFor, within } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { App } from './App';

describe('App', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn(async (input: RequestInfo | URL) => {
      const path = String(input);
      if (path === '/api/auth/me') {
        return new Response(JSON.stringify({ error: '未登录' }), { status: 401 });
      }
      return new Response(JSON.stringify({}), { status: 200 });
    }));
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('未登录时显示账号密码登录和飞书登录入口', async () => {
    render(<App />);

    await waitFor(() => expect(screen.getByRole('heading', { name: 'feed-puller' })).toBeInTheDocument());
    expect(screen.getByLabelText('邮箱')).toBeInTheDocument();
    expect(screen.getByLabelText('密码')).toBeInTheDocument();
    expect(screen.getByText('使用已绑定的飞书账号登录')).toBeInTheDocument();
  });

  it('登录后订阅列表为 JSON null 时不崩溃并显示空状态', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: RequestInfo | URL) => {
        const path = String(input);
        if (path === '/api/auth/me') {
          return new Response(JSON.stringify({ id: 1, email: 'u@test.dev', feishu_bound: false }), {
            status: 200,
            headers: { 'Content-Type': 'application/json' }
          });
        }
        if (path === '/api/subscriptions') {
          return new Response('null', { status: 200, headers: { 'Content-Type': 'application/json' } });
        }
        return new Response(JSON.stringify({}), { status: 200 });
      })
    );

    render(<App />);

    await waitFor(() => expect(screen.getByRole('heading', { name: '订阅' })).toBeInTheDocument());
    expect(screen.getByText('暂无订阅')).toBeInTheDocument();
  });

  it('登录后可通过新增订阅弹窗填写关键字过滤', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: RequestInfo | URL) => {
        const path = String(input);
        if (path === '/api/auth/me') {
          return new Response(JSON.stringify({ id: 1, email: 'u@test.dev', feishu_bound: false }), {
            status: 200,
            headers: { 'Content-Type': 'application/json' }
          });
        }
        if (path === '/api/subscriptions') {
          return new Response('[]', { status: 200, headers: { 'Content-Type': 'application/json' } });
        }
        return new Response(JSON.stringify({}), { status: 200 });
      })
    );

    render(<App />);

    await waitFor(() => expect(screen.getByRole('heading', { name: '订阅' })).toBeInTheDocument());
    fireEvent.click(screen.getByRole('button', { name: /新增订阅/ }));
    expect(await screen.findByRole('dialog')).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: '新增订阅' })).toBeInTheDocument();
    expect(screen.getByRole('textbox', { name: '包含关键字' })).toBeInTheDocument();
    expect(screen.getByRole('textbox', { name: '排除关键字' })).toBeInTheDocument();
    expect(screen.getByRole('dialog', { name: '新增订阅' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '关闭新建订阅' })).toBeInTheDocument();
  });

  it('新增订阅时可切换为 Crontab 调度', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: RequestInfo | URL) => {
        const path = String(input);
        if (path === '/api/auth/me') {
          return new Response(JSON.stringify({ id: 1, email: 'u@test.dev', feishu_bound: false }), {
            status: 200,
            headers: { 'Content-Type': 'application/json' }
          });
        }
        if (path === '/api/subscriptions') {
          return new Response('[]', { status: 200, headers: { 'Content-Type': 'application/json' } });
        }
        return new Response(JSON.stringify({}), { status: 200 });
      })
    );

    render(<App />);

    await waitFor(() => expect(screen.getByRole('heading', { name: '订阅' })).toBeInTheDocument());
    fireEvent.click(screen.getByRole('button', { name: /新增订阅/ }));
    const dialog = await screen.findByRole('dialog');
    fireEvent.click(within(dialog).getByRole('radio', { name: /Crontab/ }));
    expect(within(dialog).getByRole('textbox', { name: 'Crontab 表达式' })).toBeInTheDocument();
  });

  it('订阅列表展示上次与下次拉取摘要', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: RequestInfo | URL) => {
        const path = String(input);
        if (path === '/api/auth/me') {
          return new Response(JSON.stringify({ id: 1, email: 'u@test.dev', feishu_bound: false }), {
            status: 200,
            headers: { 'Content-Type': 'application/json' }
          });
        }
        if (path === '/api/subscriptions') {
          return new Response(
            JSON.stringify([
              {
                id: 9,
                name: 'Demo',
                feed_url: 'https://example.test/feed.xml',
                enabled: true,
                poll_interval_minutes: 30,
                poll_cron: '',
                poll_cron_timezone: 'UTC',
                download_dir: '/data',
                include_keywords: '',
                exclude_keywords: '',
                use_proxy: false,
                last_fetched_at: '2026-05-19T10:00:00Z',
                next_poll_at: '2026-05-19T11:00:00Z'
              }
            ]),
            { status: 200, headers: { 'Content-Type': 'application/json' } }
          );
        }
        return new Response(JSON.stringify({}), { status: 200 });
      })
    );

    render(<App />);

    await waitFor(() => expect(screen.getByRole('heading', { name: '订阅' })).toBeInTheDocument());
    expect(screen.getByText('上次')).toBeInTheDocument();
    expect(screen.getByText('下次')).toBeInTheDocument();
    expect(screen.getByText(/2026\/5\/19 18:00:00/)).toBeInTheDocument();
    expect(screen.getByText(/2026\/5\/19 19:00:00/)).toBeInTheDocument();
  });

  it('编辑订阅时展示上次与下次拉取时间及调度预览', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
        const path = String(input);
        const method = (init?.method ?? 'GET').toUpperCase();
        if (path === '/api/auth/me') {
          return new Response(JSON.stringify({ id: 1, email: 'u@test.dev', feishu_bound: false }), {
            status: 200,
            headers: { 'Content-Type': 'application/json' }
          });
        }
        if (path === '/api/subscriptions') {
          return new Response(
            JSON.stringify([
              {
                id: 9,
                name: 'Demo',
                feed_url: 'https://example.test/feed.xml',
                enabled: true,
                poll_interval_minutes: 30,
                poll_cron: '',
                poll_cron_timezone: 'UTC',
                download_dir: '/data',
                include_keywords: '',
                exclude_keywords: '',
                use_proxy: false,
                last_fetched_at: '2026-05-19T10:00:00Z',
                created_at: '2026-05-19T09:00:00Z',
                next_poll_at: '2026-05-19T10:30:00Z'
              }
            ]),
            { status: 200, headers: { 'Content-Type': 'application/json' } }
          );
        }
        if (path === '/api/subscriptions/preview-next-poll' && method === 'POST') {
          return new Response(JSON.stringify({ next_poll_at: '2026-05-19T10:30:00Z' }), {
            status: 200,
            headers: { 'Content-Type': 'application/json' }
          });
        }
        return new Response(JSON.stringify({}), { status: 200 });
      })
    );

    render(<App />);

    await waitFor(() => expect(screen.getByRole('heading', { name: '订阅' })).toBeInTheDocument());
    fireEvent.click(screen.getByRole('button', { name: /编辑/ }));
    const dialog = await screen.findByRole('dialog');
    expect(within(dialog).getByText(/上次拉取/)).toBeInTheDocument();
    expect(within(dialog).getByText(/下次预计拉取/)).toBeInTheDocument();
    await waitFor(() => expect(within(dialog).getByText(/2026/)).toBeInTheDocument());
  });

  it('点击编辑后以弹窗形式打开订阅编辑表单', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: RequestInfo | URL) => {
        const path = String(input);
        if (path === '/api/auth/me') {
          return new Response(JSON.stringify({ id: 1, email: 'u@test.dev', feishu_bound: false }), {
            status: 200,
            headers: { 'Content-Type': 'application/json' }
          });
        }
        if (path === '/api/subscriptions') {
          return new Response(
            JSON.stringify([
              {
                id: 9,
                name: 'Demo',
                feed_url: 'https://example.test/feed.xml',
                enabled: true,
                poll_interval_minutes: 30,
                poll_cron: '',
                poll_cron_timezone: 'UTC',
                download_dir: '/data',
                include_keywords: 'a',
                exclude_keywords: 'b',
                use_proxy: false
              }
            ]),
            { status: 200, headers: { 'Content-Type': 'application/json' } }
          );
        }
        return new Response(JSON.stringify({}), { status: 200 });
      })
    );

    render(<App />);

    await waitFor(() => expect(screen.getByRole('heading', { name: '订阅' })).toBeInTheDocument());
    fireEvent.click(screen.getByRole('button', { name: /编辑/ }));
    expect(await screen.findByRole('dialog')).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: '编辑订阅' })).toBeInTheDocument();
    expect(screen.getByRole('dialog', { name: '编辑订阅' })).toBeInTheDocument();
    expect(screen.getByDisplayValue('Demo')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '关闭编辑订阅' })).toBeInTheDocument();
  });

  it('点击拉取后展示条目预览弹窗', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
        const path = String(input);
        const method = (init?.method ?? 'GET').toUpperCase();
        if (path === '/api/auth/me') {
          return new Response(JSON.stringify({ id: 1, email: 'u@test.dev', feishu_bound: false }), {
            status: 200,
            headers: { 'Content-Type': 'application/json' }
          });
        }
        if (path === '/api/subscriptions') {
          return new Response(
            JSON.stringify([
              {
                id: 9,
                name: 'Demo',
                feed_url: 'https://example.test/feed.xml',
                enabled: true,
                poll_interval_minutes: 30,
                poll_cron: '',
                poll_cron_timezone: 'UTC',
                download_dir: '/data',
                include_keywords: '',
                exclude_keywords: '',
                use_proxy: false
              }
            ]),
            { status: 200, headers: { 'Content-Type': 'application/json' } }
          );
        }
        if (path === '/api/subscriptions/9/refresh' && method === 'POST') {
          return new Response(
            JSON.stringify({
              items: [
                {
                  id: 101,
                  subscription_id: 9,
                  title: 'Episode 1',
                  download_url: 'https://example.test/a.mp4',
                  download_status: 'pending',
                  created_at: '2026-05-19T10:00:00Z',
                  updated_at: '2026-05-19T10:00:00Z',
                  content_length: 1024
                }
              ]
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } }
          );
        }
        return new Response(JSON.stringify({}), { status: 200 });
      })
    );

    render(<App />);

    await waitFor(() => expect(screen.getByRole('heading', { name: '订阅' })).toBeInTheDocument());
    fireEvent.click(screen.getByRole('button', { name: /拉取/ }));
    expect(await screen.findByRole('heading', { name: /拉取结果 · Demo/ })).toBeInTheDocument();
    expect(screen.getByText('Episode 1')).toBeInTheDocument();
    expect(screen.getByText('未处理')).toBeInTheDocument();
  });

  it('拉取预览中已处理条目可重新下载', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
        const path = String(input);
        const method = (init?.method ?? 'GET').toUpperCase();
        if (path === '/api/auth/me') {
          return new Response(JSON.stringify({ id: 1, email: 'u@test.dev', feishu_bound: false }), {
            status: 200,
            headers: { 'Content-Type': 'application/json' }
          });
        }
        if (path === '/api/subscriptions') {
          return new Response(
            JSON.stringify([
              {
                id: 9,
                name: 'Demo',
                feed_url: 'https://example.test/feed.xml',
                enabled: true,
                poll_interval_minutes: 30,
                poll_cron: '',
                poll_cron_timezone: 'UTC',
                download_dir: '/data',
                include_keywords: '',
                exclude_keywords: '',
                use_proxy: false
              }
            ]),
            { status: 200, headers: { 'Content-Type': 'application/json' } }
          );
        }
        if (path === '/api/subscriptions/9/refresh' && method === 'POST') {
          return new Response(
            JSON.stringify({
              items: [
                {
                  id: 101,
                  subscription_id: 9,
                  title: 'Done',
                  download_url: 'https://example.test/a.mp4',
                  download_status: 'submitted',
                  created_at: '2026-05-19T10:00:00Z',
                  updated_at: '2026-05-19T10:00:00Z'
                }
              ]
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } }
          );
        }
        return new Response(JSON.stringify({}), { status: 200 });
      })
    );

    render(<App />);

    await waitFor(() => expect(screen.getByRole('heading', { name: '订阅' })).toBeInTheDocument());
    fireEvent.click(screen.getByRole('button', { name: /拉取/ }));
    expect(await screen.findByRole('heading', { name: /拉取结果 · Demo/ })).toBeInTheDocument();
    expect(screen.getByText('已处理')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '重新下载' })).toBeInTheDocument();
  });

  it('拉取预览支持勾选后批量下载', async () => {
    const batchCalls: number[][] = [];
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
        const path = String(input);
        const method = (init?.method ?? 'GET').toUpperCase();
        if (path === '/api/auth/me') {
          return new Response(JSON.stringify({ id: 1, email: 'u@test.dev', feishu_bound: false }), {
            status: 200,
            headers: { 'Content-Type': 'application/json' }
          });
        }
        if (path === '/api/subscriptions') {
          return new Response(
            JSON.stringify([
              {
                id: 9,
                name: 'Demo',
                feed_url: 'https://example.test/feed.xml',
                enabled: true,
                poll_interval_minutes: 30,
                poll_cron: '',
                poll_cron_timezone: 'UTC',
                download_dir: '/data',
                include_keywords: '',
                exclude_keywords: '',
                use_proxy: false
              }
            ]),
            { status: 200, headers: { 'Content-Type': 'application/json' } }
          );
        }
        if (path === '/api/subscriptions/9/refresh' && method === 'POST') {
          return new Response(
            JSON.stringify({
              items: [
                {
                  id: 101,
                  subscription_id: 9,
                  title: 'A',
                  download_url: 'https://example.test/a.mp4',
                  download_status: 'pending',
                  created_at: '2026-05-19T10:00:00Z'
                },
                {
                  id: 102,
                  subscription_id: 9,
                  title: 'B',
                  download_url: 'https://example.test/b.mp4',
                  download_status: 'pending',
                  created_at: '2026-05-19T10:00:00Z'
                }
              ]
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } }
          );
        }
        if (path === '/api/items/batch-download' && method === 'POST') {
          const body = JSON.parse(String(init?.body ?? '{}')) as { item_ids: number[] };
          batchCalls.push(body.item_ids);
          return new Response(
            JSON.stringify({
              items: body.item_ids.map((id) => ({
                id,
                subscription_id: 9,
                title: id === 101 ? 'A' : 'B',
                download_url: `https://example.test/${id}.mp4`,
                download_status: 'submitted',
                created_at: '2026-05-19T10:00:00Z'
              }))
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } }
          );
        }
        return new Response(JSON.stringify({}), { status: 200 });
      })
    );

    render(<App />);

    await waitFor(() => expect(screen.getByRole('heading', { name: '订阅' })).toBeInTheDocument());
    fireEvent.click(screen.getByRole('button', { name: /拉取/ }));
    const dialog = await screen.findByRole('dialog');
    const checkboxes = within(dialog).getAllByRole('checkbox');
    fireEvent.click(checkboxes[1]);
    fireEvent.click(checkboxes[2]);
    fireEvent.click(within(dialog).getByRole('button', { name: /批量下载（2）/ }));
    await waitFor(() => expect(batchCalls.length).toBe(1));
    expect(batchCalls[0]).toEqual(expect.arrayContaining([101, 102]));
    expect(await within(dialog).findByText(/已提交 2 条下载任务/)).toBeInTheDocument();
  });
});

