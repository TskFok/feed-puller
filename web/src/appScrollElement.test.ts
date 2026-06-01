import { afterEach, describe, expect, it, vi } from 'vitest';
import {
  getAppListScrollMargin,
  resolveAppScrollElement,
  resolveAppScrollLayout
} from './appScrollElement';

describe('appScrollElement', () => {
  afterEach(() => {
    document.body.innerHTML = '';
  });

  it('workspace 可滚动时返回 workspace', () => {
    document.body.innerHTML = `
      <main class="workspace" style="overflow-y: auto; height: 400px;">
        <div class="results-anchor"></div>
      </main>
    `;
    const anchor = document.querySelector('.results-anchor');
    const workspace = document.querySelector('.workspace');
    expect(resolveAppScrollElement(anchor)).toBe(workspace);
  });

  it('workspace 不可滚动时回退到 documentElement', () => {
    document.body.innerHTML = `
      <main class="workspace" style="overflow-y: visible;">
        <div class="results-anchor"></div>
      </main>
    `;
    const anchor = document.querySelector('.results-anchor');
    expect(resolveAppScrollElement(anchor)).toBe(document.documentElement);
  });

  it('resolveAppScrollLayout 返回 scrollMargin', () => {
    document.body.innerHTML = `
      <main class="workspace" style="overflow-y: auto; height: 400px;">
        <div class="results-anchor"></div>
      </main>
    `;
    const anchor = document.querySelector('.results-anchor') as HTMLElement;
    const workspace = document.querySelector('.workspace') as HTMLElement;
    Object.defineProperty(workspace, 'scrollTop', { configurable: true, value: 80, writable: true });
    vi.spyOn(anchor, 'getBoundingClientRect').mockReturnValue({
      top: 260,
      left: 0,
      right: 800,
      bottom: 560,
      width: 800,
      height: 300,
      x: 0,
      y: 260,
      toJSON: () => ({})
    } as DOMRect);
    vi.spyOn(workspace, 'getBoundingClientRect').mockReturnValue({
      top: 100,
      left: 0,
      right: 800,
      bottom: 500,
      width: 800,
      height: 400,
      x: 0,
      y: 100,
      toJSON: () => ({})
    } as DOMRect);

    expect(resolveAppScrollLayout(anchor)).toEqual({
      scrollElement: workspace,
      scrollMargin: getAppListScrollMargin(anchor, workspace)
    });
  });

  it('getAppListScrollMargin 在缺少 anchor 时返回 0', () => {
    expect(getAppListScrollMargin(null, document.body)).toBe(0);
  });
});
