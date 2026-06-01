import { renderHook } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { useAppScrollElement } from './useAppScrollElement';

function mountAnchor() {
  document.body.innerHTML = `
    <main class="workspace" style="overflow-y: auto; height: 400px;">
      <div class="results-anchor"></div>
    </main>
  `;
  const anchor = document.querySelector('.results-anchor') as HTMLDivElement;
  const workspace = document.querySelector('.workspace') as HTMLElement;
  Object.defineProperty(workspace, 'scrollTop', { configurable: true, value: 0, writable: true });
  vi.spyOn(anchor, 'getBoundingClientRect').mockReturnValue({
    top: 180,
    left: 0,
    right: 800,
    bottom: 480,
    width: 800,
    height: 300,
    x: 0,
    y: 180,
    toJSON: () => ({})
  } as DOMRect);
  vi.spyOn(workspace, 'getBoundingClientRect').mockReturnValue({
    top: 0,
    left: 0,
    right: 800,
    bottom: 400,
    width: 800,
    height: 400,
    x: 0,
    y: 0,
    toJSON: () => ({})
  } as DOMRect);
  return anchor;
}

describe('useAppScrollElement', () => {
  beforeEach(() => {
    vi.stubGlobal(
      'ResizeObserver',
      vi.fn(function (this: ResizeObserver, callback: ResizeObserverCallback) {
        this.observe = vi.fn(() => {
          callback([], this);
        });
        this.unobserve = vi.fn();
        this.disconnect = vi.fn();
      })
    );
  });

  afterEach(() => {
    document.body.innerHTML = '';
    vi.unstubAllGlobals();
  });

  it('绑定 workspace 滚动容器与 scrollMargin', () => {
    const anchor = mountAnchor();
    const anchorRef = { current: anchor };
    const { result } = renderHook(() => useAppScrollElement(anchorRef));

    expect(result.current.scrollElement).toBe(document.querySelector('.workspace'));
    expect(result.current.scrollMargin).toBe(180);
  });
});
