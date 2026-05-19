import { describe, expect, it, vi } from 'vitest';
import { clearFeishuQR } from './feishu-qr';

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
