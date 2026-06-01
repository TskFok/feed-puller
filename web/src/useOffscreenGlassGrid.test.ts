import { renderHook } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import type { RefObject } from 'react';
import { useOffscreenGlassGrid } from './useOffscreenGlassGrid';

function entryFor(target: HTMLElement, isIntersecting: boolean): IntersectionObserverEntry {
  return { target, isIntersecting } as unknown as IntersectionObserverEntry;
}

function mountContainer(cards = 1): RefObject<HTMLDivElement> {
  const container = document.createElement('div');
  for (let i = 0; i < cards; i += 1) {
    const card = document.createElement('article');
    card.className = 'prowlarr-release-card';
    container.appendChild(card);
  }
  return { current: container };
}

describe('useOffscreenGlassGrid', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('enabled 为 false 时不观察元素', () => {
    const observe = vi.fn();
    vi.stubGlobal(
      'IntersectionObserver',
      vi.fn(() => ({ observe, disconnect: vi.fn(), unobserve: vi.fn() }))
    );

    renderHook(() => useOffscreenGlassGrid(mountContainer(), false, [1]));

    expect(observe).not.toHaveBeenCalled();
  });

  it('离屏时为卡片添加 glass-surface--offscreen', () => {
    let callback: IntersectionObserverCallback = () => undefined;
    const observe = vi.fn();
    const disconnect = vi.fn();
    vi.stubGlobal(
      'IntersectionObserver',
      vi.fn((cb: IntersectionObserverCallback) => {
        callback = cb;
        return { observe, disconnect, unobserve: vi.fn() };
      })
    );

    const containerRef = mountContainer();
    const card = containerRef.current!.querySelector('.prowlarr-release-card')!;

    const { unmount } = renderHook(() => useOffscreenGlassGrid(containerRef, true, [1]));

    expect(observe).toHaveBeenCalledWith(card);

    callback([entryFor(card as HTMLElement, false)], {} as IntersectionObserver);
    expect(card.classList.contains('glass-surface--offscreen')).toBe(true);

    callback([entryFor(card as HTMLElement, true)], {} as IntersectionObserver);
    expect(card.classList.contains('glass-surface--offscreen')).toBe(false);

    unmount();
    expect(disconnect).toHaveBeenCalled();
    expect(card.classList.contains('glass-surface--offscreen')).toBe(false);
  });

  it('结果数不超过阈值时不启用观察', () => {
    const observe = vi.fn();
    vi.stubGlobal(
      'IntersectionObserver',
      vi.fn(() => ({ observe, disconnect: vi.fn(), unobserve: vi.fn() }))
    );

    renderHook(() => useOffscreenGlassGrid(mountContainer(), false, [6]));

    expect(observe).not.toHaveBeenCalled();
  });

  it('在 workspace 内使用 workspace 作为 IntersectionObserver root', () => {
    document.body.innerHTML = `
      <main class="workspace" style="overflow-y: auto; height: 400px;">
        <div class="results-host"></div>
      </main>
    `;
    const workspace = document.querySelector('.workspace') as HTMLElement;
    const host = document.querySelector('.results-host') as HTMLDivElement;
    const card = document.createElement('article');
    card.className = 'prowlarr-release-card';
    host.appendChild(card);

    let observerRoot: Element | null = null;
    vi.stubGlobal(
      'IntersectionObserver',
      vi.fn((_cb: IntersectionObserverCallback, init?: IntersectionObserverInit) => {
        observerRoot = (init?.root as Element | null | undefined) ?? null;
        return { observe: vi.fn(), disconnect: vi.fn(), unobserve: vi.fn() };
      })
    );

    renderHook(() => useOffscreenGlassGrid({ current: host }, true, [1]));

    expect(observerRoot).toBe(workspace);
    document.body.innerHTML = '';
  });
});
