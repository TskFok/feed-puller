import { render, screen, fireEvent, waitFor, within, act } from '@testing-library/react';
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
    expect(screen.getByRole('button', { name: '飞书登录' })).toBeInTheDocument();
  });

  it('切换到飞书登录时会请求 login-url', async () => {
    const getFeishuLoginUrl = vi.fn(async () => ({
      url: '/api/auth/feishu/login?state=login',
      goto: 'https://www.feishu.cn/suite/passport/oauth/authorize?state=login'
    }));
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: RequestInfo | URL) => {
        const path = String(input);
        if (path === '/api/auth/me') {
          return new Response(JSON.stringify({ error: '未登录' }), { status: 401 });
        }
        if (path === '/api/auth/feishu/login-url') {
          return new Response(JSON.stringify(await getFeishuLoginUrl()), {
            status: 200,
            headers: { 'Content-Type': 'application/json' }
          });
        }
        return new Response(JSON.stringify({}), { status: 200 });
      })
    );

    render(<App />);
    await waitFor(() => expect(screen.getByRole('button', { name: '飞书登录' })).toBeInTheDocument());
    fireEvent.click(screen.getByRole('button', { name: '飞书登录' }));

    await waitFor(() => expect(getFeishuLoginUrl).toHaveBeenCalled());
    expect(await screen.findByText('使用飞书 App 扫码即可登录')).toBeInTheDocument();
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
    const tzSelect = within(dialog).getByRole('combobox', { name: 'Crontab 时区（IANA）' });
    expect(tzSelect).toBeInTheDocument();
    expect(tzSelect.tagName).toBe('SELECT');
    expect(within(tzSelect).getByRole('option', { name: /上海/ })).toBeInTheDocument();
    expect(within(tzSelect).getByRole('option', { name: /东京/ })).toBeInTheDocument();
    expect(within(tzSelect).getByRole('option', { name: /纽约/ })).toBeInTheDocument();
  });

  it('新增订阅表单中 RSS 解析器为下拉选择', async () => {
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
    const parserSelect = within(dialog).getByRole('combobox', { name: 'RSS 解析器' });
    expect(parserSelect).toHaveClass('form-select');
    expect(within(parserSelect).getByRole('option', { name: /通用/ })).toBeInTheDocument();
    expect(within(parserSelect).getByRole('option', { name: /蜜柑/ })).toBeInTheDocument();
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

  it('点击复制后以新增订阅弹窗预填内容且不自动保存', async () => {
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const path = String(input);
      const method = (init?.method ?? 'GET').toUpperCase();
      if (path === '/api/auth/me') {
        return new Response(JSON.stringify({ id: 1, email: 'u@test.dev', feishu_bound: false }), {
          status: 200,
          headers: { 'Content-Type': 'application/json' }
        });
      }
      if (path === '/api/subscriptions' && method === 'GET') {
        return new Response(
          JSON.stringify([
            {
              id: 9,
              name: 'Demo',
              feed_url: 'https://example.test/feed.xml',
              enabled: true,
              poll_interval_minutes: 45,
              poll_cron: '',
              poll_cron_timezone: 'UTC',
              download_dir: '/data',
              include_keywords: 'a',
              exclude_keywords: 'b',
              use_proxy: true,
              rss_parser: 'mikan'
            }
          ]),
          { status: 200, headers: { 'Content-Type': 'application/json' } }
        );
      }
      if (path === '/api/subscriptions' && method === 'POST') {
        return new Response(JSON.stringify({ id: 99 }), { status: 201 });
      }
      return new Response(JSON.stringify({}), { status: 200 });
    });
    vi.stubGlobal('fetch', fetchMock);

    render(<App />);

    await waitFor(() => expect(screen.getByRole('heading', { name: '订阅' })).toBeInTheDocument());
    fireEvent.click(screen.getByRole('button', { name: /复制 Demo/ }));

    const dialog = await screen.findByRole('dialog', { name: '新增订阅' });
    expect(screen.getByRole('heading', { name: '新增订阅' })).toBeInTheDocument();
    expect(within(dialog).getByText(/已填入「Demo」的配置/)).toBeInTheDocument();
    expect(screen.getByDisplayValue('Demo (副本)')).toBeInTheDocument();
    expect(screen.getByDisplayValue('https://example.test/feed.xml')).toBeInTheDocument();
    expect(screen.getByDisplayValue('/data')).toBeInTheDocument();
    expect(screen.getByDisplayValue('a')).toBeInTheDocument();
    expect(screen.getByDisplayValue('b')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '创建订阅' })).toBeInTheDocument();

    const postCalls = fetchMock.mock.calls.filter(
      ([url, init]) => String(url) === '/api/subscriptions' && (init?.method ?? 'GET').toUpperCase() === 'POST'
    );
    expect(postCalls).toHaveLength(0);
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

  it('拉取完成后提示会自动消失', async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    try {
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
      expect(await screen.findByText('拉取完成')).toBeInTheDocument();

      await act(async () => {
        vi.advanceTimersByTime(4000);
      });

      expect(screen.queryByText('拉取完成')).not.toBeInTheDocument();
    } finally {
      vi.useRealTimers();
    }
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

  it('拉取预览支持勾选后批量修改状态', async () => {
    const statusCalls: { item_ids: number[]; download_status: string }[] = [];
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
                  id: 201,
                  subscription_id: 9,
                  title: 'Pending',
                  download_url: 'https://example.test/p.mp4',
                  download_status: 'pending',
                  created_at: '2026-05-19T10:00:00Z'
                },
                {
                  id: 202,
                  subscription_id: 9,
                  title: 'Done',
                  download_url: 'https://example.test/d.mp4',
                  download_status: 'submitted',
                  created_at: '2026-05-19T10:00:00Z'
                }
              ]
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } }
          );
        }
        if (path === '/api/items/batch-status' && method === 'POST') {
          const body = JSON.parse(String(init?.body ?? '{}')) as {
            item_ids: number[];
            download_status: string;
          };
          statusCalls.push(body);
          return new Response(
            JSON.stringify({
              items: body.item_ids.map((id) => ({
                id,
                subscription_id: 9,
                title: id === 201 ? 'Pending' : 'Done',
                download_url: `https://example.test/${id}.mp4`,
                download_status: body.download_status,
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
    fireEvent.click(within(dialog).getByRole('checkbox', { name: '选择 Pending' }));
    fireEvent.click(within(dialog).getByRole('button', { name: /标记已处理（1）/ }));
    await waitFor(() => expect(statusCalls.length).toBe(1));
    expect(statusCalls[0]).toEqual({ item_ids: [201], download_status: 'submitted' });
    expect(await within(dialog).findByText(/已将 1 条标记为已处理/)).toBeInTheDocument();

    fireEvent.click(within(dialog).getByRole('checkbox', { name: '选择 Pending' }));
    fireEvent.click(within(dialog).getByRole('checkbox', { name: '选择 Done' }));
    fireEvent.click(within(dialog).getByRole('button', { name: /标记未处理（1）/ }));
    await waitFor(() => expect(statusCalls.length).toBe(2));
    expect(statusCalls[1]).toEqual({ item_ids: [202], download_status: 'pending' });
    expect(await within(dialog).findByText(/已将 1 条标记为未处理/)).toBeInTheDocument();
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

  it('订阅列表可通过拖拽手柄调整顺序', async () => {
    const reorderCalls: number[][] = [];
    const subscriptionPayload = [
      {
        id: 1,
        name: 'First',
        feed_url: 'https://example.test/1.xml',
        enabled: true,
        poll_interval_minutes: 30,
        poll_cron: '',
        poll_cron_timezone: 'UTC',
        download_dir: '/data',
        include_keywords: '',
        exclude_keywords: '',
        use_proxy: false,
        sort_order: 0
      },
      {
        id: 2,
        name: 'Second',
        feed_url: 'https://example.test/2.xml',
        enabled: true,
        poll_interval_minutes: 30,
        poll_cron: '',
        poll_cron_timezone: 'UTC',
        download_dir: '/data',
        include_keywords: '',
        exclude_keywords: '',
        use_proxy: false,
        sort_order: 1
      }
    ];
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
          return new Response(JSON.stringify(subscriptionPayload), {
            status: 200,
            headers: { 'Content-Type': 'application/json' }
          });
        }
        if (path === '/api/subscriptions/reorder' && method === 'PUT') {
          const body = JSON.parse(String(init?.body ?? '{}')) as { subscription_ids: number[] };
          reorderCalls.push(body.subscription_ids);
          return new Response(JSON.stringify({ ok: true }), {
            status: 200,
            headers: { 'Content-Type': 'application/json' }
          });
        }
        return new Response(JSON.stringify({}), { status: 200 });
      })
    );

    render(<App />);

    await waitFor(() => expect(screen.getByText('First')).toBeInTheDocument());
    const handles = screen.getAllByRole('button', { name: /拖动调整/ });
    expect(handles).toHaveLength(2);

    const dataTransfer = {
      effectAllowed: 'move',
      dropEffect: 'move',
      setData: vi.fn(),
      getData: vi.fn()
    };
    const rows = screen.getAllByRole('row').slice(1);
    fireEvent.dragStart(handles[0], { dataTransfer });
    fireEvent.dragOver(rows[1], { dataTransfer });
    fireEvent.drop(rows[1], { dataTransfer });

    await waitFor(() => expect(reorderCalls.length).toBe(1));
    expect(reorderCalls[0]).toEqual([2, 1]);
    expect(await screen.findByText('订阅顺序已保存')).toBeInTheDocument();
    const names = screen.getAllByRole('row').slice(1).map((row) => within(row).getAllByRole('cell')[1].textContent);
    expect(names).toEqual(['Second', 'First']);
  });

  it('登录后可进入下载中列表并显示进度', async () => {
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
        if (path === '/api/downloads/active') {
          return new Response(
            JSON.stringify([
              {
                id: 2,
                item_id: 11,
                subscription_id: 3,
                subscription_name: '动漫',
                title: '进行中番剧',
                url: 'https://example.test/b.mp4',
                dir: '/data/anime',
                aria2_gid: 'gid-2',
                submitted_at: '2026-05-19T11:00:00Z',
                aria2_status: 'active',
                completed_length: 500,
                total_length: 1000,
                download_speed: 1024,
                progress_percent: 50
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
    fireEvent.click(screen.getByRole('button', { name: '下载中' }));
    await waitFor(() => expect(screen.getByRole('heading', { name: '下载中' })).toBeInTheDocument());
    expect(screen.getByText('进行中番剧')).toBeInTheDocument();
    expect(screen.getByText(/50\.0%/)).toBeInTheDocument();
    expect(screen.getByText('1.0 KB/s')).toBeInTheDocument();
  });

  it('登录后可进入下载完成列表', async () => {
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
        if (path === '/api/downloads/completed') {
          return new Response(
            JSON.stringify([
              {
                id: 1,
                item_id: 10,
                subscription_id: 2,
                subscription_name: '动漫',
                title: '示例番剧',
                url: 'https://example.test/a.mp4',
                dir: '/data/anime',
                completed_at: '2026-05-19T12:00:00Z'
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
    fireEvent.click(screen.getByRole('button', { name: '下载完成' }));
    await waitFor(() => expect(screen.getByRole('heading', { name: '下载完成' })).toBeInTheDocument());
    expect(screen.getByText('动漫')).toBeInTheDocument();
    expect(screen.getByText('示例番剧')).toBeInTheDocument();
    expect(screen.getByText('/data/anime')).toBeInTheDocument();
  });
});

