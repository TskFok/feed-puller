import { renderHook } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import type { RefObject } from 'react';
import { GLASS_OFFSCREEN_CLASS } from './glassConstants';
import { useOffscreenGlassSurface } from './useOffscreenGlassSurface';

describe('useOffscreenGlassSurface', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('离屏时为表面添加 glass-surface--offscreen', () => {
    let callback: IntersectionObserverCallback = () => undefined;
    vi.stubGlobal(
      'IntersectionObserver',
      vi.fn((cb: IntersectionObserverCallback) => {
        callback = cb;
        return { observe: vi.fn(), disconnect: vi.fn(), unobserve: vi.fn() };
      })
    );

    const surface = document.createElement('div');
    surface.className = 'table-wrap';
    const surfaceRef: RefObject<HTMLDivElement> = { current: surface };

    renderHook(() => useOffscreenGlassSurface(surfaceRef, true, [1]));

    callback(
      [{ target: surface, isIntersecting: false } as unknown as IntersectionObserverEntry],
      {} as IntersectionObserver
    );
    expect(surface.classList.contains(GLASS_OFFSCREEN_CLASS)).toBe(true);

    callback(
      [{ target: surface, isIntersecting: true } as unknown as IntersectionObserverEntry],
      {} as IntersectionObserver
    );
    expect(surface.classList.contains(GLASS_OFFSCREEN_CLASS)).toBe(false);
  });
});
