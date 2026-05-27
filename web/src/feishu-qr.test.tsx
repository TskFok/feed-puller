import { render, waitFor } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { clearFeishuQR, useFeishuQR } from './feishu-qr';

type FeishuQRHostProps = {
  authUrl: string | null;
  mode: 'login' | 'bind';
  qrContainerId: string;
  iframeContainerId: string;
  onLoginSuccess?: (user: unknown) => void;
  onBindSuccess?: () => void;
  onError?: (message: string) => void;
};

function FeishuQRHost({
  authUrl,
  mode,
  qrContainerId,
  iframeContainerId,
  onLoginSuccess,
  onBindSuccess,
  onError
}: FeishuQRHostProps) {
  useFeishuQR({
    authUrl,
    mode,
    qrContainerId,
    iframeContainerId,
    onLoginSuccess,
    onBindSuccess,
    onError
  });
  return (
    <>
      <div id={qrContainerId} />
      <div id={iframeContainerId} />
    </>
  );
}

function installQRLoginMock() {
  const QRLogin = vi.fn(function QRLoginMock(this: { matchOrigin: (origin: string) => boolean }, opt: { id: string; goto: string }) {
    const el = document.getElementById(opt.id);
    if (el) {
      el.setAttribute('data-qr-init', '1');
      el.setAttribute('data-qr-goto', opt.goto);
    }
    this.matchOrigin = (origin: string) => origin === 'https://passport.feishu.cn';
  });
  window.QRLogin = QRLogin as unknown as typeof window.QRLogin;
  return QRLogin;
}

describe('feishu-qr helpers', () => {
  it('clearFeishuQR 会清空容器并移除 message 监听', () => {
    document.body.innerHTML = `
      <div id="qr"></div>
      <div id="iframe"><iframe></iframe></div>
    `;
    const handler = vi.fn();
    window.addEventListener('message', handler);

    clearFeishuQR('qr', 'iframe', handler);

    expect(document.getElementById('qr')?.innerHTML).toBe('');
    expect(document.getElementById('iframe')?.innerHTML).toBe('');
    window.dispatchEvent(new MessageEvent('message', { data: { type: 'noop' } }));
    expect(handler).not.toHaveBeenCalled();
  });
});

describe('useFeishuQR', () => {
  afterEach(() => {
    document.body.innerHTML = '';
    delete window.QRLogin;
  });

  it('authUrl 就绪且 QRLogin 可用时初始化扫码容器', async () => {
    const QRLogin = installQRLoginMock();
    const authUrl = 'https://passport.feishu.cn/suite/passport/oauth/authorize?state=login';

    render(
      <FeishuQRHost
        authUrl={authUrl}
        mode="login"
        qrContainerId="feishuLoginQRContainer"
        iframeContainerId="feishuLoginIframeContainer"
      />
    );

    await waitFor(() => expect(QRLogin).toHaveBeenCalled());
    const qrContainer = document.getElementById('feishuLoginQRContainer');
    expect(qrContainer).toHaveAttribute('data-qr-init', '1');
    expect(qrContainer).toHaveAttribute('data-qr-goto', authUrl);
  });

  it('收到 feishu_login_success 时触发 onLoginSuccess 并清理容器', async () => {
    installQRLoginMock();
    const onLoginSuccess = vi.fn();
    const user = { id: 1, email: 'u@test.dev', feishu_bound: true };

    render(
      <FeishuQRHost
        authUrl="https://passport.feishu.cn/oauth?state=login"
        mode="login"
        qrContainerId="feishuLoginQRContainer"
        iframeContainerId="feishuLoginIframeContainer"
        onLoginSuccess={onLoginSuccess}
      />
    );

    await waitFor(() => expect(document.getElementById('feishuLoginQRContainer')).toHaveAttribute('data-qr-init', '1'));

    window.dispatchEvent(
      new MessageEvent('message', {
        data: { type: 'feishu_login_success', user }
      })
    );

    expect(onLoginSuccess).toHaveBeenCalledWith(user);
    expect(document.getElementById('feishuLoginQRContainer')?.innerHTML).toBe('');
  });

  it('收到 feishu_bind_success 时触发 onBindSuccess', async () => {
    installQRLoginMock();
    const onBindSuccess = vi.fn();

    render(
      <FeishuQRHost
        authUrl="https://passport.feishu.cn/oauth?state=bind"
        mode="bind"
        qrContainerId="feishuBindQRContainer"
        iframeContainerId="feishuBindIframeContainer"
        onBindSuccess={onBindSuccess}
      />
    );

    await waitFor(() => expect(document.getElementById('feishuBindQRContainer')).toHaveAttribute('data-qr-init', '1'));

    window.dispatchEvent(new MessageEvent('message', { data: { type: 'feishu_bind_success' } }));

    expect(onBindSuccess).toHaveBeenCalled();
    expect(document.getElementById('feishuBindQRContainer')?.innerHTML).toBe('');
  });

  it('收到 feishu_bind_error 时触发 onError', async () => {
    installQRLoginMock();
    const onError = vi.fn();

    render(
      <FeishuQRHost
        authUrl="https://passport.feishu.cn/oauth?state=bind"
        mode="bind"
        qrContainerId="feishuBindQRContainer"
        iframeContainerId="feishuBindIframeContainer"
        onError={onError}
      />
    );

    await waitFor(() => expect(document.getElementById('feishuBindQRContainer')).toHaveAttribute('data-qr-init', '1'));

    window.dispatchEvent(
      new MessageEvent('message', {
        origin: 'https://passport.feishu.cn',
        data: { type: 'feishu_bind_error', message: '绑定被拒绝' }
      })
    );

    expect(onError).toHaveBeenCalledWith('绑定被拒绝');
  });

  it('收到 tmp_code 时在 iframe 容器中创建隐藏 iframe', async () => {
    installQRLoginMock();
    const authUrl = 'https://passport.feishu.cn/oauth?state=login';

    render(
      <FeishuQRHost
        authUrl={authUrl}
        mode="login"
        qrContainerId="feishuLoginQRContainer"
        iframeContainerId="feishuLoginIframeContainer"
      />
    );

    await waitFor(() => expect(document.getElementById('feishuLoginQRContainer')).toHaveAttribute('data-qr-init', '1'));

    window.dispatchEvent(
      new MessageEvent('message', {
        origin: 'https://passport.feishu.cn',
        data: { tmp_code: 'abc123' }
      })
    );

    const iframe = document.querySelector('#feishuLoginIframeContainer iframe');
    expect(iframe).not.toBeNull();
    expect(iframe?.getAttribute('src')).toContain('tmp_code=abc123');
    expect(iframe?.getAttribute('title')).toBe('飞书登录');
  });

  it('authUrl 为空时不初始化 QRLogin', () => {
    const QRLogin = installQRLoginMock();

    render(
      <FeishuQRHost
        authUrl={null}
        mode="login"
        qrContainerId="feishuLoginQRContainer"
        iframeContainerId="feishuLoginIframeContainer"
      />
    );

    expect(QRLogin).not.toHaveBeenCalled();
  });
});
