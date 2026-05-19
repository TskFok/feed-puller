import { useEffect, useRef } from 'react';

export const FEISHU_QR_VALID_ORIGINS = [
  'https://accounts.feishu.cn',
  'https://open.feishu.cn',
  'https://passport.feishu.cn',
  'https://www.feishu.cn',
  'https://login.feishu.cn',
  'https://sf3-cn.feishucdn.com'
];

declare global {
  interface Window {
    QRLogin?: new (opt: {
      id: string;
      goto: string;
      width: number;
      height: number;
      style?: string;
    }) => { matchOrigin?: (origin: string) => boolean };
  }
}

type FeishuQRMode = 'login' | 'bind';

type UseFeishuQROptions = {
  authUrl: string | null;
  mode: FeishuQRMode;
  qrContainerId: string;
  iframeContainerId: string;
  onLoginSuccess?: (user: unknown) => void;
  onBindSuccess?: () => void;
  onError?: (message: string) => void;
};

export function clearFeishuQR(qrContainerId: string, iframeContainerId: string, handler: ((e: MessageEvent) => void) | null) {
  const el = document.getElementById(qrContainerId);
  if (el) el.innerHTML = '';
  const iframeContainer = document.getElementById(iframeContainerId);
  if (iframeContainer) iframeContainer.innerHTML = '';
  if (handler) {
    window.removeEventListener('message', handler);
  }
}

export function useFeishuQR({
  authUrl,
  mode,
  qrContainerId,
  iframeContainerId,
  onLoginSuccess,
  onBindSuccess,
  onError
}: UseFeishuQROptions) {
  const qrInstanceRef = useRef<{ matchOrigin?: (origin: string) => boolean } | null>(null);
  const messageHandlerRef = useRef<((e: MessageEvent) => void) | null>(null);

  useEffect(() => {
    const QRLogin = window.QRLogin;
    if (!authUrl || !QRLogin) return;

    const container = document.getElementById(qrContainerId);
    if (!container) return;

    container.innerHTML = '';
    if (messageHandlerRef.current) {
      window.removeEventListener('message', messageHandlerRef.current);
      messageHandlerRef.current = null;
    }

    try {
      qrInstanceRef.current = new QRLogin({
        id: qrContainerId,
        goto: authUrl,
        width: 280,
        height: 280,
        style: 'width:280px;height:280px;'
      });

      const handler = (event: MessageEvent) => {
        const t = event.data?.type;
        if (t === 'feishu_login_success') {
          window.removeEventListener('message', handler);
          messageHandlerRef.current = null;
          clearFeishuQR(qrContainerId, iframeContainerId, null);
          try {
            const user = event.data?.user;
            if (user) {
              onLoginSuccess?.(user);
            } else {
              onError?.('登录结果异常');
            }
          } catch {
            onError?.('登录结果处理失败');
          }
          return;
        }
        if (t === 'feishu_bind_success') {
          window.removeEventListener('message', handler);
          messageHandlerRef.current = null;
          clearFeishuQR(qrContainerId, iframeContainerId, null);
          onBindSuccess?.();
          return;
        }
        if (t === 'feishu_bind_error' || t === 'feishu_login_error') {
          window.removeEventListener('message', handler);
          messageHandlerRef.current = null;
          onError?.(event.data?.message ?? (mode === 'bind' ? '绑定失败' : '飞书登录失败'));
          return;
        }

        const instance = qrInstanceRef.current;
        const validOrigin =
          instance && typeof instance.matchOrigin === 'function'
            ? instance.matchOrigin(event.origin)
            : FEISHU_QR_VALID_ORIGINS.some((origin) => event.origin === origin);
        if (!validOrigin) return;

        const raw = event.data;
        const tmpCode =
          typeof raw === 'string'
            ? raw
            : raw && (raw as { tmp_code?: string }).tmp_code
              ? (raw as { tmp_code: string }).tmp_code
              : null;
        if (tmpCode && /^[a-zA-Z0-9_-]+$/.test(tmpCode)) {
          const sep = authUrl.indexOf('?') >= 0 ? '&' : '?';
          const iframeSrc = authUrl + sep + 'tmp_code=' + encodeURIComponent(tmpCode);
          const iframeHost = document.getElementById(iframeContainerId);
          if (iframeHost) {
            const iframe = document.createElement('iframe');
            iframe.setAttribute('src', iframeSrc);
            iframe.setAttribute('title', mode === 'bind' ? '飞书绑定' : '飞书登录');
            iframe.style.cssText = 'position:absolute;width:0;height:0;border:0;visibility:hidden';
            iframeHost.appendChild(iframe);
          }
        }
      };

      messageHandlerRef.current = handler;
      window.addEventListener('message', handler);
    } catch {
      onError?.('初始化飞书扫码失败');
    }

    return () => {
      clearFeishuQR(qrContainerId, iframeContainerId, messageHandlerRef.current);
      messageHandlerRef.current = null;
      qrInstanceRef.current = null;
    };
  }, [authUrl, iframeContainerId, mode, onBindSuccess, onError, onLoginSuccess, qrContainerId]);

  return {
    reset: () => {
      clearFeishuQR(qrContainerId, iframeContainerId, messageHandlerRef.current);
      messageHandlerRef.current = null;
      qrInstanceRef.current = null;
    }
  };
}
