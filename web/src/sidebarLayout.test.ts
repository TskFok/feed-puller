import { beforeEach, describe, expect, it } from 'vitest';
import {
  SIDEBAR_COLLAPSED_STORAGE_KEY,
  getStoredSidebarCollapsed,
  setStoredSidebarCollapsed
} from './sidebarLayout';

describe('sidebarLayout', () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it('默认侧栏为展开状态', () => {
    expect(getStoredSidebarCollapsed()).toBe(false);
  });

  it('收起侧栏会写入 localStorage', () => {
    setStoredSidebarCollapsed(true);
    expect(localStorage.getItem(SIDEBAR_COLLAPSED_STORAGE_KEY)).toBe('1');
    expect(getStoredSidebarCollapsed()).toBe(true);
  });

  it('展开侧栏会清除收起标记', () => {
    setStoredSidebarCollapsed(true);
    setStoredSidebarCollapsed(false);
    expect(localStorage.getItem(SIDEBAR_COLLAPSED_STORAGE_KEY)).toBe('0');
    expect(getStoredSidebarCollapsed()).toBe(false);
  });
});
