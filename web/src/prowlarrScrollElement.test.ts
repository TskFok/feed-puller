import { afterEach, describe, expect, it, vi } from 'vitest';
import { getProwlarrListScrollMargin, resolveProwlarrScrollElement } from './prowlarrScrollElement';

describe('prowlarrScrollElement re-exports', () => {
  afterEach(() => {
    document.body.innerHTML = '';
  });

  it('resolveProwlarrScrollElement 仍指向 workspace', () => {
    document.body.innerHTML = `
      <main class="workspace" style="overflow-y: auto; height: 400px;">
        <div class="prowlarr-results-grid"></div>
      </main>
    `;
    const anchor = document.querySelector('.prowlarr-results-grid');
    expect(resolveProwlarrScrollElement(anchor)).toBe(document.querySelector('.workspace'));
  });

  it('getProwlarrListScrollMargin 与 appScrollElement 一致', () => {
    document.body.innerHTML = `
      <main class="workspace">
        <div class="prowlarr-results-grid"></div>
      </main>
    `;
    const anchor = document.querySelector('.prowlarr-results-grid') as HTMLElement;
    const workspace = document.querySelector('.workspace') as HTMLElement;
    Object.defineProperty(workspace, 'scrollTop', { configurable: true, value: 120, writable: true });
    vi.spyOn(anchor, 'getBoundingClientRect').mockReturnValue({
      top: 420,
      left: 0,
      right: 800,
      bottom: 720,
      width: 800,
      height: 300,
      x: 0,
      y: 420,
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

    expect(getProwlarrListScrollMargin(anchor, workspace)).toBe(440);
  });
});
