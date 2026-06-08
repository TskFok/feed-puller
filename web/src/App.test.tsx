import { render, screen, fireEvent, waitFor, within, act } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { App } from './App';
import type { User } from './types';

function isSubscriptionsListPath(path: string) {
  return path === '/api/subscriptions' || path.startsWith('/api/subscriptions?');
}

function isActiveDownloadsPath(path: string) {
  return path === '/api/downloads/active' || path.startsWith('/api/downloads/active?');
}

function isCompletedDownloadsPath(path: string) {
  return path === '/api/downloads/completed' || path.startsWith('/api/downloads/completed?');
}

function isAIConfigsListPath(path: string) {
  return path === '/api/ai-configs' || path.startsWith('/api/ai-configs?');
}

describe('App', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn(async (input: RequestInfo | URL) => {
      const path = String(input);
      if (path === '/api/auth/me') {
        return new Response(JSON.stringify({ error: '未登录' }), { status: 401 });
      }
      if (path === '/api/auth/options') {
        return new Response(JSON.stringify({ password_login_enabled: true, feishu_login_enabled: true }), {
          status: 200,
          headers: { 'Content-Type': 'application/json' }
        });
      }
      return new Response(JSON.stringify({}), { status: 200 });
    }));
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.unstubAllGlobals();
    window.location.hash = '';
  });

  it('未登录时显示账号密码登录和飞书登录入口', async () => {
    render(<App />);

    await waitFor(() => expect(screen.getByRole('heading', { name: 'feed-puller' })).toBeInTheDocument());
    expect(await screen.findByLabelText('邮箱')).toBeInTheDocument();
    expect(screen.getByLabelText('密码')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '飞书登录' })).toBeInTheDocument();
  });

  it('禁用账号密码登录时仅显示飞书登录', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: RequestInfo | URL) => {
        const path = String(input);
        if (path === '/api/auth/me') {
          return new Response(JSON.stringify({ error: '未登录' }), { status: 401 });
        }
        if (path === '/api/auth/options') {
          return new Response(JSON.stringify({ password_login_enabled: false, feishu_login_enabled: true }), {
            status: 200,
            headers: { 'Content-Type': 'application/json' }
          });
        }
        if (path === '/api/auth/feishu/login-url') {
          return new Response(
            JSON.stringify({
              url: '/api/auth/feishu/login?state=login',
              goto: 'https://www.feishu.cn/suite/passport/oauth/authorize?state=login'
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } }
          );
        }
        return new Response(JSON.stringify({}), { status: 200 });
      })
    );

    render(<App />);

    await waitFor(() => expect(screen.getByRole('heading', { name: 'feed-puller' })).toBeInTheDocument());
    expect(screen.queryByLabelText('邮箱')).not.toBeInTheDocument();
    expect(screen.queryByLabelText('密码')).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: '账号密码登录' })).not.toBeInTheDocument();
    expect(await screen.findByText('使用飞书 App 扫码即可登录')).toBeInTheDocument();
  });

  it('未登录且两种登录均可用时显示迁移提示', async () => {
    render(<App />);

    await waitFor(() =>
      expect(
        screen.getByText(/首次使用？请先用账号密码登录，在「设置 → 飞书登录迁移向导」中绑定飞书后再关闭密码登录。/)
      ).toBeInTheDocument()
    );
  });

  it('未登录时可在登录页切换主题', async () => {
    localStorage.clear();
    render(<App />);

    await waitFor(() => expect(screen.getByRole('heading', { name: 'feed-puller' })).toBeInTheDocument());
    fireEvent.click(screen.getByRole('button', { name: '玻璃浅色' }));
    expect(document.documentElement.dataset.theme).toBe('light');
    expect(localStorage.getItem('feed-puller-theme')).toBe('light');
  });

  it('登录后未绑定飞书时显示迁移横幅并可前往设置', async () => {
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
        if (path === '/api/auth/options') {
          return new Response(JSON.stringify({ password_login_enabled: true, feishu_login_enabled: true }), {
            status: 200,
            headers: { 'Content-Type': 'application/json' }
          });
        }
        if (path === '/api/subscriptions' || path.startsWith('/api/subscriptions?')) {
          return new Response(JSON.stringify({ items: [], total: 0, page: 1, page_size: 20 }), {
            status: 200,
            headers: { 'Content-Type': 'application/json' }
          });
        }
        if (path === '/api/settings/proxy') {
          return new Response(JSON.stringify({ proxy_url: '' }), {
            status: 200,
            headers: { 'Content-Type': 'application/json' }
          });
        }
        if (path === '/api/settings/prowlarr') {
          return new Response(
            JSON.stringify({
              url: '',
              api_key: '',
              download_dir: '',
              tv_download_dir: '',
              movie_rename_enabled: false,
              tmdb_api_key: '',
              indexer_ids: [],
              configured: false
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } }
          );
        }
        return new Response(JSON.stringify({}), { status: 200 });
      })
    );

    render(<App />);

    await waitFor(() => expect(screen.getByText(/建议完成飞书登录迁移/)).toBeInTheDocument());
    fireEvent.click(screen.getByRole('button', { name: '前往设置' }));
    expect(await screen.findByRole('heading', { name: '飞书登录迁移向导' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '立即绑定飞书' })).toBeInTheDocument();
  });

  it('点击不再提示后隐藏飞书迁移横幅', async () => {
    localStorage.clear();
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
        if (path === '/api/auth/options') {
          return new Response(JSON.stringify({ password_login_enabled: true, feishu_login_enabled: true }), {
            status: 200,
            headers: { 'Content-Type': 'application/json' }
          });
        }
        if (path === '/api/subscriptions' || path.startsWith('/api/subscriptions?')) {
          return new Response(JSON.stringify({ items: [], total: 0, page: 1, page_size: 20 }), {
            status: 200,
            headers: { 'Content-Type': 'application/json' }
          });
        }
        if (path === '/api/settings/proxy') {
          return new Response(JSON.stringify({ proxy_url: '' }), {
            status: 200,
            headers: { 'Content-Type': 'application/json' }
          });
        }
        if (path === '/api/settings/prowlarr') {
          return new Response(
            JSON.stringify({
              url: '',
              api_key: '',
              download_dir: '',
              tv_download_dir: '',
              movie_rename_enabled: false,
              tmdb_api_key: '',
              indexer_ids: [],
              configured: false
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } }
          );
        }
        return new Response(JSON.stringify({}), { status: 200 });
      })
    );

    render(<App />);

    await waitFor(() => expect(screen.getByText(/建议完成飞书登录迁移/)).toBeInTheDocument());
    fireEvent.click(screen.getByRole('button', { name: '不再提示' }));
    expect(screen.queryByText(/建议完成飞书登录迁移/)).not.toBeInTheDocument();
    expect(localStorage.getItem('feed-puller-feishu-banner-dismissed')).toBe('1');
  });

  it('设置页可切换玻璃浅色主题', async () => {
    localStorage.clear();
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: RequestInfo | URL) => {
        const path = String(input);
        if (path === '/api/auth/me') {
          return new Response(JSON.stringify({ id: 1, email: 'u@test.dev', feishu_bound: true }), {
            status: 200,
            headers: { 'Content-Type': 'application/json' }
          });
        }
        if (path === '/api/auth/options') {
          return new Response(JSON.stringify({ password_login_enabled: true, feishu_login_enabled: true }), {
            status: 200,
            headers: { 'Content-Type': 'application/json' }
          });
        }
        if (path === '/api/settings/proxy') {
          return new Response(JSON.stringify({ proxy_url: '' }), { status: 200, headers: { 'Content-Type': 'application/json' } });
        }
        if (path === '/api/settings/prowlarr') {
          return new Response(
            JSON.stringify({
              url: '',
              api_key: '',
              download_dir: '',
              tv_download_dir: '',
              movie_rename_enabled: false,
              tmdb_api_key: '',
              indexer_ids: [],
              configured: false
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } }
          );
        }
        return new Response(JSON.stringify({}), { status: 200 });
      })
    );

    window.location.hash = '#settings';
    render(<App />);

    await waitFor(() => expect(screen.getByRole('heading', { name: '设置' })).toBeInTheDocument());
    fireEvent.click(screen.getByRole('button', { name: '玻璃浅色' }));
    expect(document.documentElement.dataset.theme).toBe('light');
    expect(localStorage.getItem('feed-puller-theme')).toBe('light');
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
        if (path === '/api/auth/options') {
          return new Response(JSON.stringify({ password_login_enabled: true, feishu_login_enabled: true }), {
            status: 200,
            headers: { 'Content-Type': 'application/json' }
          });
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

  it('飞书登录加载完成后渲染扫码容器并初始化 QRLogin', async () => {
    const QRLogin = vi.fn(function QRLoginMock(
      this: { matchOrigin: (origin: string) => boolean },
      opt: { id: string; goto: string }
    ) {
      const el = document.getElementById(opt.id);
      if (el) el.setAttribute('data-qr-init', '1');
      this.matchOrigin = () => true;
    });
    window.QRLogin = QRLogin as unknown as typeof window.QRLogin;

    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: RequestInfo | URL) => {
        const path = String(input);
        if (path === '/api/auth/me') {
          return new Response(JSON.stringify({ error: '未登录' }), { status: 401 });
        }
        if (path === '/api/auth/options') {
          return new Response(JSON.stringify({ password_login_enabled: true, feishu_login_enabled: true }), {
            status: 200,
            headers: { 'Content-Type': 'application/json' }
          });
        }
        if (path === '/api/auth/feishu/login-url') {
          return new Response(
            JSON.stringify({
              url: '/api/auth/feishu/login?state=login',
              goto: 'https://passport.feishu.cn/suite/passport/oauth/authorize?state=login'
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } }
          );
        }
        return new Response(JSON.stringify({}), { status: 200 });
      })
    );

    render(<App />);
    await waitFor(() => expect(screen.getByRole('button', { name: '飞书登录' })).toBeInTheDocument());
    fireEvent.click(screen.getByRole('button', { name: '飞书登录' }));

    expect(await screen.findByText('使用飞书 App 扫码即可登录')).toBeInTheDocument();
    await waitFor(() => expect(QRLogin).toHaveBeenCalled());
    expect(document.getElementById('feishuLoginQRContainer')).toBeInTheDocument();
    expect(document.getElementById('feishuLoginIframeContainer')).toBeInTheDocument();
    expect(document.getElementById('feishuLoginQRContainer')).toHaveAttribute('data-qr-init', '1');
    delete window.QRLogin;
  });

  it('切换到飞书登录时先显示加载提示', async () => {
    let resolveLoginUrl: ((value: Response) => void) | undefined;
    const loginUrlPromise = new Promise<Response>((resolve) => {
      resolveLoginUrl = resolve;
    });

    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: RequestInfo | URL) => {
        const path = String(input);
        if (path === '/api/auth/me') {
          return new Response(JSON.stringify({ error: '未登录' }), { status: 401 });
        }
        if (path === '/api/auth/options') {
          return new Response(JSON.stringify({ password_login_enabled: true, feishu_login_enabled: true }), {
            status: 200,
            headers: { 'Content-Type': 'application/json' }
          });
        }
        if (path === '/api/auth/feishu/login-url') {
          return loginUrlPromise;
        }
        return new Response(JSON.stringify({}), { status: 200 });
      })
    );

    render(<App />);
    await waitFor(() => expect(screen.getByRole('button', { name: '飞书登录' })).toBeInTheDocument());
    fireEvent.click(screen.getByRole('button', { name: '飞书登录' }));

    expect(screen.getByText('正在加载飞书扫码...')).toBeInTheDocument();
    expect(document.getElementById('feishuLoginQRContainer')).not.toBeInTheDocument();

    await act(async () => {
      resolveLoginUrl!(
        new Response(
          JSON.stringify({
            url: '/api/auth/feishu/login?state=login',
            goto: 'https://passport.feishu.cn/suite/passport/oauth/authorize?state=login'
          }),
          { status: 200, headers: { 'Content-Type': 'application/json' } }
        )
      );
    });

    expect(await screen.findByText('使用飞书 App 扫码即可登录')).toBeInTheDocument();
    expect(document.getElementById('feishuLoginQRContainer')).toBeInTheDocument();
  });

  it('飞书扫码登录成功后会进入主界面', async () => {
    window.QRLogin = vi.fn(function QRLoginMock(this: { matchOrigin: () => boolean }) {
      this.matchOrigin = () => true;
    }) as unknown as typeof window.QRLogin;

    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: RequestInfo | URL) => {
        const path = String(input);
        if (path === '/api/auth/me') {
          return new Response(JSON.stringify({ error: '未登录' }), { status: 401 });
        }
        if (path === '/api/auth/options') {
          return new Response(JSON.stringify({ password_login_enabled: false, feishu_login_enabled: true }), {
            status: 200,
            headers: { 'Content-Type': 'application/json' }
          });
        }
        if (path === '/api/auth/feishu/login-url') {
          return new Response(
            JSON.stringify({
              url: '/api/auth/feishu/login?state=login',
              goto: 'https://passport.feishu.cn/suite/passport/oauth/authorize?state=login'
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } }
          );
        }
        if (isSubscriptionsListPath(path)) {
          return new Response(JSON.stringify({ items: [], total: 0, page: 1, page_size: 20 }), {
            status: 200,
            headers: { 'Content-Type': 'application/json' }
          });
        }
        return new Response(JSON.stringify({}), { status: 200 });
      })
    );

    render(<App />);
    await waitFor(() => expect(document.getElementById('feishuLoginQRContainer')).toBeInTheDocument());

    const loggedInUser = { id: 1, email: 'u@test.dev', feishu_bound: true, feishu_name: 'Alice' };
    window.dispatchEvent(
      new MessageEvent('message', {
        data: { type: 'feishu_login_success', user: loggedInUser }
      })
    );

    expect(await screen.findByRole('heading', { name: '订阅' })).toBeInTheDocument();
    delete window.QRLogin;
  });

  it('已绑定飞书时设置页显示解绑按钮并可解绑', async () => {
    let currentUser: User = { id: 1, email: 'u@test.dev', feishu_bound: true, feishu_name: 'Alice' };
    const unbindFeishu = vi.fn(async () => ({ ok: true }));

    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
        const path = String(input);
        const method = (init?.method ?? 'GET').toUpperCase();
        if (path === '/api/auth/me') {
          return new Response(JSON.stringify(currentUser), {
            status: 200,
            headers: { 'Content-Type': 'application/json' }
          });
        }
        if (path === '/api/auth/options') {
          return new Response(JSON.stringify({ password_login_enabled: true, feishu_login_enabled: true }), {
            status: 200,
            headers: { 'Content-Type': 'application/json' }
          });
        }
        if (path === '/api/settings/proxy') {
          return new Response(JSON.stringify({ proxy_url: '' }), { status: 200, headers: { 'Content-Type': 'application/json' } });
        }
        if (path === '/api/settings/prowlarr') {
          return new Response(
            JSON.stringify({
              url: '',
              api_key: '',
              download_dir: '',
              tv_download_dir: '',
              movie_rename_enabled: false,
              tmdb_api_key: '',
              indexer_ids: [],
              configured: false
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } }
          );
        }
        if (path === '/api/settings/feishu-binding' && method === 'DELETE') {
          await unbindFeishu();
          currentUser = { id: 1, email: 'u@test.dev', feishu_bound: false };
          return new Response(JSON.stringify({ ok: true }), { status: 200, headers: { 'Content-Type': 'application/json' } });
        }
        return new Response(JSON.stringify({}), { status: 200 });
      })
    );

    window.location.hash = '#settings';
    render(<App />);

    await waitFor(() => expect(screen.getByRole('heading', { name: '设置' })).toBeInTheDocument());
    expect(screen.getByText('当前状态：Alice')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '解绑' })).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: '解绑' }));

    await waitFor(() => expect(unbindFeishu).toHaveBeenCalled());
    expect(await screen.findByText('飞书账号已解绑')).toBeInTheDocument();
    expect(screen.getByText('当前状态：未绑定')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: '解绑' })).not.toBeInTheDocument();
  });

  it('未绑定飞书时设置页不显示解绑按钮', async () => {
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
        if (path === '/api/auth/options') {
          return new Response(JSON.stringify({ password_login_enabled: true, feishu_login_enabled: true }), {
            status: 200,
            headers: { 'Content-Type': 'application/json' }
          });
        }
        if (path === '/api/settings/proxy') {
          return new Response(JSON.stringify({ proxy_url: '' }), { status: 200, headers: { 'Content-Type': 'application/json' } });
        }
        if (path === '/api/settings/prowlarr') {
          return new Response(
            JSON.stringify({
              url: '',
              api_key: '',
              download_dir: '',
              tv_download_dir: '',
              movie_rename_enabled: false,
              tmdb_api_key: '',
              indexer_ids: [],
              configured: false
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } }
          );
        }
        return new Response(JSON.stringify({}), { status: 200 });
      })
    );

    window.location.hash = '#settings';
    render(<App />);

    await waitFor(() => expect(screen.getByRole('heading', { name: '设置' })).toBeInTheDocument());
    expect(screen.getByText('当前状态：未绑定')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: '解绑' })).not.toBeInTheDocument();
  });

  it('打开绑定弹窗时渲染扫码容器', async () => {
    window.QRLogin = vi.fn(function QRLoginMock(this: { matchOrigin: () => boolean }) {
      this.matchOrigin = () => true;
    }) as unknown as typeof window.QRLogin;

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
        if (path === '/api/auth/options') {
          return new Response(JSON.stringify({ password_login_enabled: true, feishu_login_enabled: true }), {
            status: 200,
            headers: { 'Content-Type': 'application/json' }
          });
        }
        if (path === '/api/settings/proxy') {
          return new Response(JSON.stringify({ proxy_url: '' }), { status: 200, headers: { 'Content-Type': 'application/json' } });
        }
        if (path === '/api/settings/prowlarr') {
          return new Response(
            JSON.stringify({
              url: '',
              api_key: '',
              download_dir: '',
              tv_download_dir: '',
              movie_rename_enabled: false,
              tmdb_api_key: '',
              indexer_ids: [],
              configured: false
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } }
          );
        }
        if (path === '/api/settings/feishu-bind-url') {
          return new Response(
            JSON.stringify({
              url: '/api/settings/feishu-bind?state=bind',
              goto: 'https://passport.feishu.cn/suite/passport/oauth/authorize?state=bind'
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } }
          );
        }
        return new Response(JSON.stringify({}), { status: 200 });
      })
    );

    window.location.hash = '#settings';
    render(<App />);

    await waitFor(() => expect(screen.getByRole('heading', { name: '设置' })).toBeInTheDocument());
    fireEvent.click(screen.getByRole('button', { name: /^绑定飞书$/ }));

    const dialog = await screen.findByRole('dialog', { name: '绑定飞书' });
    expect(within(dialog).getByText('使用飞书 App 扫码，可将飞书账号绑定到当前用户')).toBeInTheDocument();
    expect(document.getElementById('feishuBindQRContainer')).toBeInTheDocument();
    expect(document.getElementById('feishuBindIframeContainer')).toBeInTheDocument();
    delete window.QRLogin;
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
        if (isSubscriptionsListPath(path)) {
          return new Response('null', { status: 200, headers: { 'Content-Type': 'application/json' } });
        }
        return new Response(JSON.stringify({}), { status: 200 });
      })
    );

    render(<App />);

    await waitFor(() => expect(screen.getByRole('heading', { name: '订阅' })).toBeInTheDocument());
    await waitFor(() => expect(screen.getByText('暂无订阅')).toBeInTheDocument());
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
        if (isSubscriptionsListPath(path)) {
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
        if (isSubscriptionsListPath(path)) {
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
        if (isSubscriptionsListPath(path)) {
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
        if (isSubscriptionsListPath(path)) {
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
                next_poll_at: '2030-05-19T11:00:00Z'
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
    expect(screen.getByText(/2030\/5\/19 19:00:00/)).toBeInTheDocument();
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
        if (isSubscriptionsListPath(path)) {
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
        if (isSubscriptionsListPath(path)) {
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

  it('创建订阅成功后不会自动调用拉取接口', async () => {
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const path = String(input);
      const method = (init?.method ?? 'GET').toUpperCase();
      if (path === '/api/auth/me') {
        return new Response(JSON.stringify({ id: 1, email: 'u@test.dev', feishu_bound: false }), {
          status: 200,
          headers: { 'Content-Type': 'application/json' }
        });
      }
      if (isSubscriptionsListPath(path) && method === 'GET') {
        return new Response(JSON.stringify([]), {
          status: 200,
          headers: { 'Content-Type': 'application/json' }
        });
      }
      if (path === '/api/subscriptions/preview-next-poll' && method === 'POST') {
        return new Response(JSON.stringify({ next_poll_at: '2026-05-19T13:00:00Z' }), {
          status: 200,
          headers: { 'Content-Type': 'application/json' }
        });
      }
      if (path === '/api/subscriptions' && method === 'POST') {
        return new Response(
          JSON.stringify({
            id: 42,
            name: 'New',
            feed_url: 'https://example.test/new.xml',
            enabled: true,
            poll_interval_minutes: 30,
            poll_cron: '',
            poll_cron_timezone: 'UTC',
            download_dir: '/data/new',
            include_keywords: '',
            exclude_keywords: '',
            use_proxy: false,
            rss_parser: 'generic',
            created_at: '2026-05-19T12:00:00Z'
          }),
          { status: 201, headers: { 'Content-Type': 'application/json' } }
        );
      }
      return new Response(JSON.stringify({}), { status: 200 });
    });
    vi.stubGlobal('fetch', fetchMock);

    render(<App />);

    await waitFor(() => expect(screen.getByRole('heading', { name: '订阅' })).toBeInTheDocument());
    fireEvent.click(screen.getByRole('button', { name: /新增订阅/ }));

    const dialog = await screen.findByRole('dialog', { name: '新增订阅' });
    fireEvent.change(within(dialog).getByRole('textbox', { name: '订阅名称' }), { target: { value: 'New' } });
    fireEvent.change(within(dialog).getByRole('textbox', { name: '订阅地址' }), {
      target: { value: 'https://example.test/new.xml' }
    });
    fireEvent.change(within(dialog).getByRole('textbox', { name: '下载目录' }), { target: { value: '/data/new' } });
    fireEvent.click(within(dialog).getByRole('button', { name: '创建订阅' }));

    await waitFor(() => expect(screen.getByText(/订阅已创建/)).toBeInTheDocument());

    const refreshCalls = fetchMock.mock.calls.filter(
      ([url, init]) => /\/api\/subscriptions\/\d+\/refresh$/.test(String(url)) && (init?.method ?? 'GET').toUpperCase() === 'POST'
    );
    expect(refreshCalls).toHaveLength(0);
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
      if (isSubscriptionsListPath(path) && method === 'GET') {
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

  it('订阅列表的行内操作按钮使用非透明操作样式', async () => {
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
        if (isSubscriptionsListPath(path) && method === 'GET') {
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
        return new Response(JSON.stringify({}), { status: 200 });
      })
    );

    render(<App />);

    await waitFor(() => expect(screen.getByRole('heading', { name: '订阅' })).toBeInTheDocument());
    expect(screen.getByRole('button', { name: /拉取/ })).toHaveClass('subscription-action');
    expect(screen.getByRole('button', { name: /编辑/ })).toHaveClass('subscription-action');
    expect(screen.getByRole('button', { name: /复制 Demo/ })).toHaveClass('subscription-action');
  });

  it('删除订阅前需要二次确认', async () => {
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const path = String(input);
      const method = (init?.method ?? 'GET').toUpperCase();
      if (path === '/api/auth/me') {
        return new Response(JSON.stringify({ id: 1, email: 'u@test.dev', feishu_bound: false }), {
          status: 200,
          headers: { 'Content-Type': 'application/json' }
        });
      }
      if (isSubscriptionsListPath(path) && method === 'GET') {
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
      if (path === '/api/subscriptions/9' && method === 'DELETE') {
        return new Response(JSON.stringify({ ok: true }), {
          status: 200,
          headers: { 'Content-Type': 'application/json' }
        });
      }
      return new Response(JSON.stringify({}), { status: 200 });
    });
    vi.stubGlobal('fetch', fetchMock);
    const deleteCalls = () =>
      fetchMock.mock.calls.filter(
        ([url, init]) => String(url) === '/api/subscriptions/9' && (init?.method ?? 'GET').toUpperCase() === 'DELETE'
      );

    render(<App />);

    await waitFor(() => expect(screen.getByRole('heading', { name: '订阅' })).toBeInTheDocument());
    fireEvent.click(screen.getByRole('button', { name: '删除' }));

    const dialog = await screen.findByRole('dialog', { name: '删除订阅' });
    expect(within(dialog).getByText(/Demo/)).toBeInTheDocument();
    expect(deleteCalls()).toHaveLength(0);

    fireEvent.click(within(dialog).getByRole('button', { name: '取消' }));
    await waitFor(() => expect(screen.queryByRole('dialog', { name: '删除订阅' })).not.toBeInTheDocument());
    expect(deleteCalls()).toHaveLength(0);

    fireEvent.click(screen.getByRole('button', { name: '删除' }));
    const confirmDialog = await screen.findByRole('dialog', { name: '删除订阅' });
    fireEvent.click(within(confirmDialog).getByRole('button', { name: '确认删除' }));

    await waitFor(() => expect(deleteCalls()).toHaveLength(1));
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
        if (isSubscriptionsListPath(path)) {
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
    const dialog = await screen.findByRole('dialog');
    expect(within(dialog).getByRole('heading', { name: /拉取结果 · Demo/ })).toBeInTheDocument();
    expect(within(dialog).getByText('Episode 1')).toBeInTheDocument();
    expect(within(dialog).getByRole('cell', { name: '未处理' })).toBeInTheDocument();
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
          if (isSubscriptionsListPath(path)) {
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
        if (isSubscriptionsListPath(path)) {
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
    const dialog = await screen.findByRole('dialog');
    expect(within(dialog).getByRole('heading', { name: /拉取结果 · Demo/ })).toBeInTheDocument();
    expect(within(dialog).getByRole('cell', { name: '已处理' })).toBeInTheDocument();
    expect(within(dialog).getByRole('button', { name: '重新下载' })).toBeInTheDocument();
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
        if (isSubscriptionsListPath(path)) {
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
    expect(await screen.findByText(/已将 1 条标记为已处理/)).toBeInTheDocument();

    fireEvent.click(within(dialog).getByRole('checkbox', { name: '选择 Pending' }));
    fireEvent.click(within(dialog).getByRole('checkbox', { name: '选择 Done' }));
    fireEvent.click(within(dialog).getByRole('button', { name: /标记未处理（1）/ }));
    await waitFor(() => expect(statusCalls.length).toBe(2));
    expect(statusCalls[1]).toEqual({ item_ids: [202], download_status: 'pending' });
    expect(await screen.findByText(/已将 1 条标记为未处理/)).toBeInTheDocument();
  });

  it('拉取预览批量修改状态时仅被点击的按钮显示更新中', async () => {
    let resolveStatus: ((value: Response) => void) | undefined;
    const statusPromise = new Promise<Response>((resolve) => {
      resolveStatus = resolve;
    });

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
        if (isSubscriptionsListPath(path)) {
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
                }
              ]
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } }
          );
        }
        if (path === '/api/items/batch-status' && method === 'POST') {
          return statusPromise;
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

    await waitFor(() => {
      expect(within(dialog).getByRole('button', { name: '更新中…' })).toBeInTheDocument();
    });
    expect(within(dialog).getByRole('button', { name: /标记未处理（1）/ })).toBeInTheDocument();
    expect(within(dialog).getByRole('button', { name: /标记未处理（1）/ })).not.toBeDisabled();

    await act(async () => {
      resolveStatus!(
        new Response(
          JSON.stringify({
            items: [
              {
                id: 201,
                subscription_id: 9,
                title: 'Pending',
                download_url: 'https://example.test/p.mp4',
                download_status: 'submitted',
                created_at: '2026-05-19T10:00:00Z'
              }
            ]
          }),
          { status: 200, headers: { 'Content-Type': 'application/json' } }
        )
      );
    });

    await waitFor(() =>
      expect(within(dialog).getByRole('button', { name: /标记已处理（1）/ })).toBeInTheDocument()
    );
  });

  it('拉取预览单条下载时仅当前行显示提交中', async () => {
    let resolveDownload: ((value: Response) => void) | undefined;
    const downloadPromise = new Promise<Response>((resolve) => {
      resolveDownload = resolve;
    });

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
        if (isSubscriptionsListPath(path)) {
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
                  id: 401,
                  subscription_id: 9,
                  title: 'Item A',
                  download_url: 'https://example.test/a.mp4',
                  download_status: 'pending',
                  created_at: '2026-05-19T10:00:00Z'
                },
                {
                  id: 402,
                  subscription_id: 9,
                  title: 'Item B',
                  download_url: 'https://example.test/b.mp4',
                  download_status: 'pending',
                  created_at: '2026-05-19T10:00:00Z'
                }
              ]
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } }
          );
        }
        if (path === '/api/items/401/download' && method === 'POST') {
          return downloadPromise;
        }
        return new Response(JSON.stringify({}), { status: 200 });
      })
    );

    render(<App />);

    await waitFor(() => expect(screen.getByRole('heading', { name: '订阅' })).toBeInTheDocument());
    fireEvent.click(screen.getByRole('button', { name: /拉取/ }));
    const dialog = await screen.findByRole('dialog');
    const rowA = within(dialog).getByText('Item A').closest('tr')!;
    const rowB = within(dialog).getByText('Item B').closest('tr')!;
    fireEvent.click(within(rowA).getByRole('button', { name: '下载' }));

    await waitFor(() => {
      expect(within(rowA).getByRole('button', { name: '提交中…' })).toBeInTheDocument();
    });
    expect(within(rowB).getByRole('button', { name: '下载' })).not.toBeDisabled();
    expect(within(rowB).getByRole('checkbox', { name: '选择 Item B' })).not.toBeDisabled();
    expect(within(rowA).getByRole('checkbox', { name: '选择 Item A' })).toBeDisabled();

    await act(async () => {
      resolveDownload!(
        new Response(
          JSON.stringify({
            id: 401,
            subscription_id: 9,
            title: 'Item A',
            download_url: 'https://example.test/a.mp4',
            download_status: 'submitting',
            created_at: '2026-05-19T10:00:00Z'
          }),
          { status: 200, headers: { 'Content-Type': 'application/json' } }
        )
      );
    });

    await waitFor(() => expect(within(dialog).queryByRole('button', { name: '提交中…' })).not.toBeInTheDocument());
  });

  it('拉取预览批量修改状态时不阻塞单条下载', async () => {
    let resolveStatus: ((value: Response) => void) | undefined;
    const statusPromise = new Promise<Response>((resolve) => {
      resolveStatus = resolve;
    });

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
        if (isSubscriptionsListPath(path)) {
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
                  id: 501,
                  subscription_id: 9,
                  title: 'Pending',
                  download_url: 'https://example.test/p.mp4',
                  download_status: 'pending',
                  created_at: '2026-05-19T10:00:00Z'
                }
              ]
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } }
          );
        }
        if (path === '/api/items/batch-status' && method === 'POST') {
          return statusPromise;
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

    await waitFor(() => {
      expect(within(dialog).getByRole('button', { name: '更新中…' })).toBeInTheDocument();
    });
    expect(within(dialog).getByRole('button', { name: '下载' })).not.toBeDisabled();
    expect(within(dialog).getByRole('checkbox', { name: '选择 Pending' })).not.toBeDisabled();

    await act(async () => {
      resolveStatus!(
        new Response(
          JSON.stringify({
            items: [
              {
                id: 501,
                subscription_id: 9,
                title: 'Pending',
                download_url: 'https://example.test/p.mp4',
                download_status: 'submitted',
                created_at: '2026-05-19T10:00:00Z'
              }
            ]
          }),
          { status: 200, headers: { 'Content-Type': 'application/json' } }
        )
      );
    });
  });

  it('拉取预览支持按状态筛选', async () => {
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
        if (isSubscriptionsListPath(path)) {
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
                  id: 301,
                  subscription_id: 9,
                  title: 'Pending Item',
                  download_url: 'https://example.test/p.mp4',
                  download_status: 'pending',
                  created_at: '2026-05-19T10:00:00Z'
                },
                {
                  id: 302,
                  subscription_id: 9,
                  title: 'Done Item',
                  download_url: 'https://example.test/d.mp4',
                  download_status: 'submitted',
                  created_at: '2026-05-19T10:00:00Z'
                },
                {
                  id: 303,
                  subscription_id: 9,
                  title: 'No URL Item',
                  download_url: '',
                  download_status: 'pending',
                  created_at: '2026-05-19T10:00:00Z'
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
    const dialog = await screen.findByRole('dialog');

    expect(within(dialog).getByText('Pending Item')).toBeInTheDocument();
    expect(within(dialog).getByText('Done Item')).toBeInTheDocument();
    expect(within(dialog).getByText('No URL Item')).toBeInTheDocument();
    expect(within(dialog).getByText('显示 3 / 3 条')).toBeInTheDocument();

    fireEvent.change(within(dialog).getByLabelText('状态筛选'), { target: { value: 'pending' } });
    expect(within(dialog).getByText('Pending Item')).toBeInTheDocument();
    expect(within(dialog).queryByText('Done Item')).not.toBeInTheDocument();
    expect(within(dialog).queryByText('No URL Item')).not.toBeInTheDocument();
    expect(within(dialog).getByText('显示 1 / 3 条')).toBeInTheDocument();

    fireEvent.change(within(dialog).getByLabelText('状态筛选'), { target: { value: 'submitted' } });
    expect(within(dialog).queryByText('Pending Item')).not.toBeInTheDocument();
    expect(within(dialog).getByText('Done Item')).toBeInTheDocument();
    expect(within(dialog).queryByText('No URL Item')).not.toBeInTheDocument();

    fireEvent.change(within(dialog).getByLabelText('状态筛选'), { target: { value: 'no-download' } });
    expect(within(dialog).queryByText('Pending Item')).not.toBeInTheDocument();
    expect(within(dialog).queryByText('Done Item')).not.toBeInTheDocument();
    expect(within(dialog).getByText('No URL Item')).toBeInTheDocument();
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
        if (isSubscriptionsListPath(path)) {
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
    expect(await screen.findByText(/已提交 2 条下载任务/)).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: '查看进度' }));
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
    expect(await screen.findByRole('heading', { name: '下载中' })).toBeInTheDocument();
  });

  it('订阅列表可通过拖拽手柄调整顺序', async () => {
    const reorderCalls: number[][] = [];
    let subscriptionOrder = [1, 2];
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
        if (isSubscriptionsListPath(path)) {
          const byId = new Map(subscriptionPayload.map((sub) => [sub.id, sub]));
          const items = subscriptionOrder.map((id) => byId.get(id)!);
          return new Response(
            JSON.stringify({ items, total: items.length, page: 1, page_size: 30 }),
            { status: 200, headers: { 'Content-Type': 'application/json' } }
          );
        }
        if (path === '/api/subscriptions/ids') {
          return new Response(JSON.stringify({ ids: subscriptionOrder }), {
            status: 200,
            headers: { 'Content-Type': 'application/json' }
          });
        }
        if (path === '/api/subscriptions/reorder' && method === 'PUT') {
          const body = JSON.parse(String(init?.body ?? '{}')) as { subscription_ids: number[] };
          reorderCalls.push(body.subscription_ids);
          subscriptionOrder = body.subscription_ids;
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
        if (isSubscriptionsListPath(path)) {
          return new Response('[]', { status: 200, headers: { 'Content-Type': 'application/json' } });
        }
        if (isActiveDownloadsPath(path)) {
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

  it('下载中列表在上一次 active 请求未完成时不会重复请求', async () => {
    let activeCallCount = 0;
    let resolveActive: (() => void) | undefined;

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
        if (isSubscriptionsListPath(path)) {
          return new Response('[]', { status: 200, headers: { 'Content-Type': 'application/json' } });
        }
        if (isActiveDownloadsPath(path)) {
          activeCallCount += 1;
          return new Promise<Response>((resolve) => {
            resolveActive = () => {
              resolve(
                new Response(
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
                )
              );
            };
          });
        }
        return new Response(JSON.stringify({}), { status: 200 });
      })
    );

    render(<App />);
    await waitFor(() => expect(screen.getByRole('heading', { name: '订阅' })).toBeInTheDocument());
    fireEvent.click(screen.getByRole('button', { name: '下载中' }));
    await waitFor(() => expect(screen.getByRole('heading', { name: '下载中' })).toBeInTheDocument());
    expect(activeCallCount).toBe(1);

    vi.useFakeTimers({ shouldAdvanceTime: true });
    try {
      await act(async () => {
        await vi.advanceTimersByTimeAsync(5000);
      });
      expect(activeCallCount).toBe(1);

      await act(async () => {
        resolveActive?.();
      });
      await waitFor(() => expect(screen.getByText('进行中番剧')).toBeInTheDocument());
      expect(activeCallCount).toBe(1);
    } finally {
      vi.useRealTimers();
    }
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
        if (isSubscriptionsListPath(path)) {
          return new Response('[]', { status: 200, headers: { 'Content-Type': 'application/json' } });
        }
        if (isCompletedDownloadsPath(path)) {
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
                final_path: '/data/anime/番剧 S01E01.mp4',
                ai_rename_enabled: true,
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
    expect(screen.getByRole('button', { name: '重命名' })).toBeInTheDocument();
    expect(screen.getByText('动漫')).toBeInTheDocument();
    expect(screen.getByText('示例番剧')).toBeInTheDocument();
    expect(screen.getByText('/data/anime')).toBeInTheDocument();
    expect(screen.getByText('/data/anime/番剧 S01E01.mp4')).toBeInTheDocument();
  });

  it('下载完成列表在上一次 completed 请求未完成时不会重复请求', async () => {
    let completedCallCount = 0;
    let resolveCompleted: (() => void) | undefined;

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
        if (isSubscriptionsListPath(path)) {
          return new Response('[]', { status: 200, headers: { 'Content-Type': 'application/json' } });
        }
        if (isCompletedDownloadsPath(path)) {
          completedCallCount += 1;
          return new Promise<Response>((resolve) => {
            resolveCompleted = () => {
              resolve(
                new Response(
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
                )
              );
            };
          });
        }
        return new Response(JSON.stringify({}), { status: 200 });
      })
    );

    render(<App />);
    await waitFor(() => expect(screen.getByRole('heading', { name: '订阅' })).toBeInTheDocument());
    fireEvent.click(screen.getByRole('button', { name: '下载完成' }));
    await waitFor(() => expect(screen.getByRole('heading', { name: '下载完成' })).toBeInTheDocument());
    expect(completedCallCount).toBe(1);

    vi.useFakeTimers({ shouldAdvanceTime: true });
    try {
      await act(async () => {
        await vi.advanceTimersByTimeAsync(30000);
      });
      expect(completedCallCount).toBe(1);

      await act(async () => {
        resolveCompleted?.();
      });
      await waitFor(() => expect(screen.getByText('示例番剧')).toBeInTheDocument());
      expect(completedCallCount).toBe(1);
    } finally {
      vi.useRealTimers();
    }
  });

  it('登录后可通过 Provider 预设快速填写 AI 配置', async () => {
    const createAIConfig = vi.fn(async () => ({
      id: 1,
      name: 'DeepSeek',
      url: 'https://api.deepseek.com/v1',
      model: 'deepseek-chat',
      api_key: 'sk-test',
      request_options: ''
    }));
    const savedAIConfigs: Awaited<ReturnType<typeof createAIConfig>>[] = [];
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
        const path = String(input);
        if (path === '/api/auth/me') {
          return new Response(JSON.stringify({ id: 1, email: 'u@test.dev', feishu_bound: false }), {
            status: 200,
            headers: { 'Content-Type': 'application/json' }
          });
        }
        if (isSubscriptionsListPath(path)) {
          return new Response('[]', { status: 200, headers: { 'Content-Type': 'application/json' } });
        }
        if (isAIConfigsListPath(path) && (!init || init.method === undefined)) {
          return new Response(JSON.stringify({ items: savedAIConfigs, total: savedAIConfigs.length, page: 1, page_size: 30 }), {
            status: 200,
            headers: { 'Content-Type': 'application/json' }
          });
        }
        if (path === '/api/ai-configs' && init?.method === 'POST') {
          const created = await createAIConfig();
          savedAIConfigs.push(created);
          return new Response(JSON.stringify(created), {
            status: 201,
            headers: { 'Content-Type': 'application/json' }
          });
        }
        return new Response(JSON.stringify({}), { status: 200 });
      })
    );

    render(<App />);
    await waitFor(() => expect(screen.getByRole('heading', { name: '订阅' })).toBeInTheDocument());
    fireEvent.click(screen.getByRole('button', { name: 'AI 配置' }));
    await waitFor(() => expect(screen.getByRole('heading', { name: 'AI 配置' })).toBeInTheDocument());
    fireEvent.click(screen.getByRole('button', { name: '新增配置' }));

    const dialog = await screen.findByRole('dialog');
    fireEvent.click(within(dialog).getByRole('button', { name: 'DeepSeek' }));
    expect(within(dialog).getByLabelText('API 地址')).toHaveValue('https://api.deepseek.com/v1');
    expect(within(dialog).getByRole('textbox', { name: '模型' })).toHaveValue('deepseek-chat');
    fireEvent.change(within(dialog).getByLabelText('API Key'), { target: { value: 'sk-test' } });
    fireEvent.click(within(dialog).getByRole('button', { name: '保存' }));

    await waitFor(() => expect(createAIConfig).toHaveBeenCalled());
    expect(await screen.findByText('DeepSeek')).toBeInTheDocument();
  });

  it('登录后可刷新 AI 模型列表并选择模型', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
        const path = String(input);
        if (path === '/api/auth/me') {
          return new Response(JSON.stringify({ id: 1, email: 'u@test.dev', feishu_bound: false }), {
            status: 200,
            headers: { 'Content-Type': 'application/json' }
          });
        }
        if (isSubscriptionsListPath(path)) {
          return new Response('[]', { status: 200, headers: { 'Content-Type': 'application/json' } });
        }
        if (isAIConfigsListPath(path)) {
          return new Response(JSON.stringify({ items: [], total: 0, page: 1, page_size: 30 }), {
            status: 200,
            headers: { 'Content-Type': 'application/json' }
          });
        }
        if (path === '/api/ai-configs/models' && init?.method === 'POST') {
          return new Response(JSON.stringify({ models: ['gpt-4o-mini', 'gpt-4o'] }), {
            status: 200,
            headers: { 'Content-Type': 'application/json' }
          });
        }
        return new Response(JSON.stringify({}), { status: 200 });
      })
    );

    render(<App />);
    await waitFor(() => expect(screen.getByRole('heading', { name: '订阅' })).toBeInTheDocument());
    fireEvent.click(screen.getByRole('button', { name: 'AI 配置' }));
    fireEvent.click(screen.getByRole('button', { name: '新增配置' }));

    const dialog = await screen.findByRole('dialog');
    fireEvent.change(within(dialog).getByLabelText('API Key'), { target: { value: 'sk-test' } });
    fireEvent.click(within(dialog).getByRole('button', { name: '刷新模型' }));

    await waitFor(() =>
      expect(within(dialog).getByRole('combobox', { name: '模型' })).toBeInTheDocument()
    );
    expect(within(dialog).getByRole('combobox', { name: '模型' })).toHaveValue('gpt-4o-mini');
  });

  it('登录后可进入 AI 配置并新增一条配置', async () => {
    const createAIConfig = vi.fn(async (payload: Record<string, unknown>) => ({
      id: 1,
      name: 'DeepSeek',
      url: 'https://api.deepseek.com/v1',
      model: 'deepseek-chat',
      api_key: 'sk-test',
      request_options: payload.request_options
    }));
    const savedAIConfigs: Awaited<ReturnType<typeof createAIConfig>>[] = [];
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
        const path = String(input);
        if (path === '/api/auth/me') {
          return new Response(JSON.stringify({ id: 1, email: 'u@test.dev', feishu_bound: false }), {
            status: 200,
            headers: { 'Content-Type': 'application/json' }
          });
        }
        if (isSubscriptionsListPath(path)) {
          return new Response('[]', { status: 200, headers: { 'Content-Type': 'application/json' } });
        }
        if (isAIConfigsListPath(path) && (!init || init.method === undefined)) {
          return new Response(JSON.stringify({ items: savedAIConfigs, total: savedAIConfigs.length, page: 1, page_size: 30 }), {
            status: 200,
            headers: { 'Content-Type': 'application/json' }
          });
        }
        if (path === '/api/ai-configs' && init?.method === 'POST') {
          const created = await createAIConfig(JSON.parse(String(init.body)));
          savedAIConfigs.push(created);
          return new Response(JSON.stringify(created), {
            status: 201,
            headers: { 'Content-Type': 'application/json' }
          });
        }
        return new Response(JSON.stringify({}), { status: 200 });
      })
    );

    render(<App />);
    await waitFor(() => expect(screen.getByRole('heading', { name: '订阅' })).toBeInTheDocument());
    fireEvent.click(screen.getByRole('button', { name: 'AI 配置' }));
    await waitFor(() => expect(screen.getByRole('heading', { name: 'AI 配置' })).toBeInTheDocument());
    fireEvent.click(screen.getByRole('button', { name: '新增配置' }));

    const dialog = await screen.findByRole('dialog');
    fireEvent.change(within(dialog).getByLabelText('模型名称'), { target: { value: 'DeepSeek' } });
    fireEvent.change(within(dialog).getByLabelText('API 地址'), {
      target: { value: 'https://api.deepseek.com/v1' }
    });
    fireEvent.change(within(dialog).getByRole('textbox', { name: '模型' }), { target: { value: 'deepseek-chat' } });
    fireEvent.change(within(dialog).getByLabelText('高级请求参数（JSON，可选）'), {
      target: { value: '{"temperature":0.8}' }
    });
    fireEvent.change(within(dialog).getByLabelText('API Key'), { target: { value: 'sk-test' } });
    fireEvent.click(within(dialog).getByRole('button', { name: '保存' }));

    await waitFor(() => expect(createAIConfig).toHaveBeenCalled());
    expect(createAIConfig).toHaveBeenCalledWith(expect.objectContaining({ request_options: '{"temperature":0.8}' }));
    expect(await screen.findByText('DeepSeek')).toBeInTheDocument();
  });

  it('刷新后根据 URL hash 恢复当前标签页', async () => {
    window.location.hash = '#active';
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
        if (isSubscriptionsListPath(path)) {
          return new Response('[]', { status: 200, headers: { 'Content-Type': 'application/json' } });
        }
        if (isActiveDownloadsPath(path)) {
          return new Response('[]', { status: 200, headers: { 'Content-Type': 'application/json' } });
        }
        return new Response(JSON.stringify({}), { status: 200 });
      })
    );

    render(<App />);
    await waitFor(() => expect(screen.getByRole('heading', { name: '下载中' })).toBeInTheDocument());
    expect(screen.queryByRole('heading', { name: '订阅' })).not.toBeInTheDocument();
  });

  it('登录后可折叠侧栏并为工作区留出更多空间', async () => {
    localStorage.clear();
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
        if (isSubscriptionsListPath(path)) {
          return new Response('[]', { status: 200, headers: { 'Content-Type': 'application/json' } });
        }
        return new Response(JSON.stringify({}), { status: 200 });
      })
    );

    render(<App />);

    await waitFor(() => expect(screen.getByRole('heading', { name: '订阅' })).toBeInTheDocument());
    expect(document.querySelector('.app-shell--sidebar-collapsed')).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: '收起侧栏' }));
    expect(document.querySelector('.app-shell--sidebar-collapsed')).toBeInTheDocument();
    expect(localStorage.getItem('feed-puller-sidebar-collapsed')).toBe('1');

    fireEvent.click(screen.getByRole('button', { name: '展开侧栏' }));
    expect(document.querySelector('.app-shell--sidebar-collapsed')).not.toBeInTheDocument();
    expect(localStorage.getItem('feed-puller-sidebar-collapsed')).toBe('0');
  });

  it('切换标签页时同步 URL hash', async () => {
    window.location.hash = '';
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
        if (isSubscriptionsListPath(path)) {
          return new Response('[]', { status: 200, headers: { 'Content-Type': 'application/json' } });
        }
        if (isCompletedDownloadsPath(path)) {
          return new Response('[]', { status: 200, headers: { 'Content-Type': 'application/json' } });
        }
        return new Response(JSON.stringify({}), { status: 200 });
      })
    );

    render(<App />);
    await waitFor(() => expect(screen.getByRole('heading', { name: '订阅' })).toBeInTheDocument());
    fireEvent.click(screen.getByRole('button', { name: '下载完成' }));
    await waitFor(() => expect(window.location.hash).toBe('#completed'));
  });
});
