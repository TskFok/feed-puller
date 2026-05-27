import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { useRef } from 'react';
import { AnimatedModal } from './AnimatedModal';

function TestModal({ onClose }: { onClose: () => void }) {
  const inputRef = useRef<HTMLInputElement>(null);
  return (
    <AnimatedModal onClose={onClose} ariaLabelledBy="test-title" initialFocusRef={inputRef}>
      <h2 id="test-title">测试弹窗</h2>
      <input ref={inputRef} aria-label="名称" />
      <button type="button">确认</button>
    </AnimatedModal>
  );
}

describe('AnimatedModal', () => {
  afterEach(() => {
    document.body.style.overflow = '';
  });

  it('Escape 与遮罩点击会触发 onClose', () => {
    const onClose = vi.fn();
    render(<TestModal onClose={onClose} />);

    fireEvent.keyDown(document, { key: 'Escape' });
    expect(onClose).toHaveBeenCalledTimes(1);

    fireEvent.mouseDown(document.querySelector('.modal-overlay')!);
    expect(onClose).toHaveBeenCalledTimes(2);
  });

  it('打开时锁定 body 滚动并聚焦首个字段', async () => {
    render(<TestModal onClose={vi.fn()} />);
    expect(document.body.style.overflow).toBe('hidden');
    await waitFor(() => expect(screen.getByLabelText('名称')).toHaveFocus());
  });

  it('Tab 键在弹窗内循环', async () => {
    render(<TestModal onClose={vi.fn()} />);
    const input = screen.getByLabelText('名称');
    const confirm = screen.getByRole('button', { name: '确认' });
    await waitFor(() => expect(input).toHaveFocus());
    confirm.focus();
    fireEvent.keyDown(document, { key: 'Tab' });
    expect(input).toHaveFocus();
  });
});
