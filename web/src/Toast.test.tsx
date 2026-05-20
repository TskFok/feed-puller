import { act, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { TOAST_DISMISS_MS, ToastProvider, useToast } from './Toast';

function ToastProbe() {
  const { showToast } = useToast();
  return (
    <div>
      <button type="button" onClick={() => showToast('订阅已更新')}>
        显示成功
      </button>
      <button type="button" onClick={() => showToast('保存失败', 'error')}>
        显示错误
      </button>
    </div>
  );
}

describe('Toast', () => {
  afterEach(() => {
    vi.useRealTimers();
  });

  it('成功提示以浮动 Toast 展示在视口角落', () => {
    render(
      <ToastProvider>
        <ToastProbe />
      </ToastProvider>
    );

    fireEvent.click(screen.getByRole('button', { name: '显示成功' }));

    const toast = screen.getByRole('status');
    expect(toast).toHaveTextContent('订阅已更新');
    expect(toast).toHaveClass('toast', 'toast-success');
    expect(document.querySelector('.toast-viewport')).toBeInTheDocument();
  });

  it('错误提示使用 error 样式', () => {
    render(
      <ToastProvider>
        <ToastProbe />
      </ToastProvider>
    );

    fireEvent.click(screen.getByRole('button', { name: '显示错误' }));

    const toast = screen.getByRole('status');
    expect(toast).toHaveTextContent('保存失败');
    expect(toast).toHaveClass('toast', 'toast-error');
    expect(toast).toHaveTextContent('操作失败');
  });

  it('点击关闭按钮会移除 Toast', () => {
    render(
      <ToastProvider>
        <ToastProbe />
      </ToastProvider>
    );

    fireEvent.click(screen.getByRole('button', { name: '显示成功' }));
    fireEvent.click(screen.getByRole('button', { name: '关闭提示' }));
    expect(screen.queryByText('订阅已更新')).not.toBeInTheDocument();
  });

  it('超时后自动消失', () => {
    vi.useFakeTimers();
    render(
      <ToastProvider>
        <ToastProbe />
      </ToastProvider>
    );

    fireEvent.click(screen.getByRole('button', { name: '显示成功' }));
    expect(screen.getByText('订阅已更新')).toBeInTheDocument();

    act(() => {
      vi.advanceTimersByTime(TOAST_DISMISS_MS);
    });

    expect(screen.queryByText('订阅已更新')).not.toBeInTheDocument();
  });
});
