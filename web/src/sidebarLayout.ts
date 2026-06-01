import { useCallback, useState } from 'react';

export const SIDEBAR_COLLAPSED_STORAGE_KEY = 'feed-puller-sidebar-collapsed';

export function getStoredSidebarCollapsed(): boolean {
  if (typeof localStorage === 'undefined') {
    return false;
  }
  return localStorage.getItem(SIDEBAR_COLLAPSED_STORAGE_KEY) === '1';
}

export function setStoredSidebarCollapsed(collapsed: boolean): void {
  if (typeof localStorage === 'undefined') {
    return;
  }
  localStorage.setItem(SIDEBAR_COLLAPSED_STORAGE_KEY, collapsed ? '1' : '0');
}

export function useSidebarCollapsed() {
  const [collapsed, setCollapsedState] = useState(() => getStoredSidebarCollapsed());

  const setCollapsed = useCallback((next: boolean) => {
    setStoredSidebarCollapsed(next);
    setCollapsedState(next);
  }, []);

  const toggleCollapsed = useCallback(() => {
    setCollapsed(!collapsed);
  }, [collapsed, setCollapsed]);

  return { collapsed, setCollapsed, toggleCollapsed };
}
