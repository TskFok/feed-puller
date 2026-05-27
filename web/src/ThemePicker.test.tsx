import { fireEvent, render, screen } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { ThemePicker } from './ThemePicker';
import { THEME_STORAGE_KEY } from './theme';

describe('ThemePicker', () => {
  beforeEach(() => {
    localStorage.clear();
    document.documentElement.removeAttribute('data-theme');
  });

  afterEach(() => {
    localStorage.clear();
    document.documentElement.removeAttribute('data-theme');
  });

  it('panel 变体显示完整标签并可切换主题', () => {
    render(<ThemePicker variant="panel" />);

    expect(screen.getByRole('heading', { name: '外观' })).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Bubblegum 浅色' }));
    expect(document.documentElement.dataset.theme).toBe('light');
    expect(localStorage.getItem(THEME_STORAGE_KEY)).toBe('light');
  });

  it('compact 变体在登录页可用 aria-label 切换主题', () => {
    render(<ThemePicker variant="compact" />);

    expect(screen.getByText('外观')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Y2K 暗色' }));
    expect(document.documentElement.dataset.theme).toBe('dark');
  });
});
