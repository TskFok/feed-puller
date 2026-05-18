import { render, screen, waitFor } from '@testing-library/react';
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
});

