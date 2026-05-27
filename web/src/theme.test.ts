import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import {
  applyTheme,
  getStoredPreference,
  getStoredTheme,
  getSystemTheme,
  initTheme,
  resolveTheme,
  setThemePreference,
  subscribeSystemTheme,
  THEME_STORAGE_KEY
} from './theme';

describe('theme', () => {
  beforeEach(() => {
    localStorage.clear();
    document.documentElement.removeAttribute('data-theme');
  });

  afterEach(() => {
    localStorage.clear();
    document.documentElement.removeAttribute('data-theme');
    vi.unstubAllGlobals();
  });

  it('无存储时偏好为 system', () => {
    expect(getStoredPreference()).toBe('system');
  });

  it('resolveTheme 在 system 下跟随 getSystemTheme', () => {
    vi.stubGlobal(
      'matchMedia',
      vi.fn(() => ({ matches: true, addEventListener: vi.fn(), removeEventListener: vi.fn() }))
    );
    expect(resolveTheme('system')).toBe('light');
  });

  it('setThemePreference 持久化并应用到 document', () => {
    setThemePreference('light');
    expect(localStorage.getItem(THEME_STORAGE_KEY)).toBe('light');
    expect(document.documentElement.dataset.theme).toBe('light');
  });

  it('initTheme 从 localStorage 恢复', () => {
    localStorage.setItem(THEME_STORAGE_KEY, 'dark');
    initTheme();
    expect(document.documentElement.dataset.theme).toBe('dark');
  });

  it('subscribeSystemTheme 在系统偏好变化时回调', () => {
    const changeListeners = new Set<() => void>();
    vi.stubGlobal('matchMedia', vi.fn(() => ({
      matches: false,
      addEventListener: (_event: string, fn: () => void) => {
        changeListeners.add(fn);
      },
      removeEventListener: (_event: string, fn: () => void) => {
        changeListeners.delete(fn);
      }
    })));

    const onChange = vi.fn();
    subscribeSystemTheme(onChange);
    for (const listener of changeListeners) {
      listener();
    }
    expect(onChange).toHaveBeenCalledWith(getSystemTheme());
  });

  it('applyTheme 在动效可用时添加 theme-switching 类', () => {
    vi.useFakeTimers();
    vi.stubGlobal(
      'matchMedia',
      vi.fn(() => ({ matches: false, addEventListener: vi.fn(), removeEventListener: vi.fn() }))
    );
    applyTheme('light');
    expect(document.documentElement.classList.contains('theme-switching')).toBe(true);
    vi.advanceTimersByTime(220);
    expect(document.documentElement.classList.contains('theme-switching')).toBe(false);
    vi.useRealTimers();
  });
});
