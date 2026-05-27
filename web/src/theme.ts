import { useCallback, useEffect, useState } from 'react';

export type Theme = 'dark' | 'light';
export type ThemePreference = Theme | 'system';

export const THEME_STORAGE_KEY = 'feed-puller-theme';

const THEME_COLORS: Record<Theme, string> = {
  dark: '#0a0018',
  light: '#fff0f8'
};

const THEME_TRANSITION_MS = 220;

function prefersReducedMotion(): boolean {
  if (typeof window === 'undefined' || typeof window.matchMedia !== 'function') {
    return false;
  }
  return window.matchMedia('(prefers-reduced-motion: reduce)').matches;
}

function updateThemeMeta(theme: Theme) {
  const metaTheme = document.querySelector('meta[name="theme-color"]');
  metaTheme?.setAttribute('content', THEME_COLORS[theme]);
  const metaScheme = document.querySelector('meta[name="color-scheme"]');
  metaScheme?.setAttribute('content', theme);
}

export function getSystemTheme(): Theme {
  if (typeof window === 'undefined' || typeof window.matchMedia !== 'function') {
    return 'dark';
  }
  return window.matchMedia('(prefers-color-scheme: light)').matches ? 'light' : 'dark';
}

export function getStoredPreference(): ThemePreference {
  if (typeof localStorage === 'undefined') {
    return 'system';
  }
  const value = localStorage.getItem(THEME_STORAGE_KEY);
  if (value === 'light' || value === 'dark' || value === 'system') {
    return value;
  }
  return 'system';
}

export function resolveTheme(preference: ThemePreference): Theme {
  if (preference === 'system') {
    return getSystemTheme();
  }
  return preference;
}

export function applyTheme(theme: Theme) {
  if (typeof document === 'undefined') {
    return;
  }
  const root = document.documentElement;
  root.dataset.theme = theme;
  updateThemeMeta(theme);

  if (prefersReducedMotion()) {
    return;
  }

  root.classList.add('theme-switching');
  window.setTimeout(() => root.classList.remove('theme-switching'), THEME_TRANSITION_MS);
}

export function setThemePreference(preference: ThemePreference) {
  localStorage.setItem(THEME_STORAGE_KEY, preference);
  applyTheme(resolveTheme(preference));
}

/** @deprecated 使用 setThemePreference */
export function setTheme(theme: Theme) {
  setThemePreference(theme);
}

export function initTheme() {
  applyTheme(resolveTheme(getStoredPreference()));
}

export function subscribeSystemTheme(onChange: (theme: Theme) => void) {
  if (typeof window === 'undefined' || typeof window.matchMedia !== 'function') {
    return () => undefined;
  }
  const media = window.matchMedia('(prefers-color-scheme: light)');
  const handler = () => onChange(getSystemTheme());
  media.addEventListener('change', handler);
  return () => media.removeEventListener('change', handler);
}

export function useTheme() {
  const [preference, setPreferenceState] = useState<ThemePreference>(() => getStoredPreference());
  const theme = resolveTheme(preference);

  useEffect(() => {
    applyTheme(theme);
  }, [theme]);

  useEffect(() => {
    if (preference !== 'system') {
      return undefined;
    }
    return subscribeSystemTheme((next) => applyTheme(next));
  }, [preference]);

  const setPreference = useCallback((next: ThemePreference) => {
    setThemePreference(next);
    setPreferenceState(next);
  }, []);

  const toggleTheme = useCallback(() => {
    setPreference(theme === 'dark' ? 'light' : 'dark');
  }, [setPreference, theme]);

  return { theme, preference, setTheme: setPreference, setPreference, toggleTheme };
}

/** @deprecated 使用 getStoredPreference */
export function getStoredTheme(): Theme {
  return resolveTheme(getStoredPreference());
}
