import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { Banner } from './Banner';

describe('Banner', () => {
  it('成功提示使用 alert 与 success 样式', () => {
    render(<Banner variant="success">保存成功</Banner>);
    const banner = screen.getByRole('alert');
    expect(banner).toHaveTextContent('保存成功');
    expect(banner).toHaveClass('banner', 'banner-success');
    expect(screen.queryByRole('button', { name: '关闭错误提示' })).not.toBeInTheDocument();
  });

  it('错误提示使用 alert 与 error 样式', () => {
    render(
      <Banner variant="error" onDismiss={() => {}}>
        请求失败
      </Banner>
    );
    const banner = screen.getByRole('alert');
    expect(banner).toHaveTextContent('请求失败');
    expect(banner).toHaveClass('banner', 'banner-error');
    expect(screen.getByRole('button', { name: '关闭错误提示' })).toBeInTheDocument();
  });

  it('点击关闭按钮会触发 onDismiss', () => {
    const onDismiss = vi.fn();
    render(
      <Banner variant="error" onDismiss={onDismiss}>
        校验失败
      </Banner>
    );

    fireEvent.click(screen.getByRole('button', { name: '关闭错误提示' }));
    expect(onDismiss).toHaveBeenCalledTimes(1);
  });

  it('支持附加 className', () => {
    render(
      <Banner variant="error" className="banner-in-modal" onDismiss={() => {}}>
        校验失败
      </Banner>
    );
    expect(screen.getByRole('alert')).toHaveClass('banner-in-modal');
  });
});
