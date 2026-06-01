import { render } from '@testing-library/react';
import { beforeAll, beforeEach, describe, expect, it, vi } from 'vitest';
import { PROWLARR_ROW_GAP_PX } from './prowlarrLayoutConstants';
import { PROWLARR_ROW_HEIGHT_CACHE_KEY, recordProwlarrRowHeight } from './prowlarrRowHeightCache';
import { ProwlarrVirtualResultsGrid } from './ProwlarrVirtualResultsGrid';
import type { ProwlarrRelease } from './types';

vi.mock('./useGridColumns', () => ({
  useGridColumns: () => 1
}));

function makeRelease(index: number, title?: string): ProwlarrRelease {
  return {
    guid: `guid-${index}`,
    title: title ?? `Release ${index}`,
    indexer: 'Tracker',
    indexerId: 1,
    size: 1024,
    seeders: 1,
    leechers: 0,
    protocol: 'torrent',
    infoHash: `hash-${index}`
  };
}

describe('ProwlarrVirtualResultsGrid', () => {
  const noop = () => undefined;

  beforeEach(() => {
    sessionStorage.clear();
  });

  beforeAll(() => {
    Object.defineProperty(window, 'scrollTo', { value: vi.fn(), configurable: true, writable: true });
    Element.prototype.getBoundingClientRect = vi.fn(function (this: Element) {
      return {
        x: 0,
        y: 0,
        width: 800,
        height: this.classList?.contains('prowlarr-results-grid') ? 400 : 280,
        top: this.classList?.contains('prowlarr-results-grid') ? 120 : 0,
        left: 0,
        right: 800,
        bottom: this.classList?.contains('prowlarr-results-grid') ? 520 : 280,
        toJSON: () => ({})
      } as DOMRect;
    });
  });

  it('虚拟行带 data-index、padding-bottom 与 measureElement 引用', () => {
    const results = Array.from({ length: 55 }, (_, index) => makeRelease(index));
    const { container } = render(
      <ProwlarrVirtualResultsGrid
        results={results}
        selectedGuids={new Set()}
        submittedGuids={new Set()}
        downloadingGuid={null}
        batchDownloading={false}
        formatBytes={() => '1 KB'}
        formatTime={() => '—'}
        onToggle={noop}
        onDownload={noop}
      />
    );

    expect(container.querySelector('.prowlarr-results-grid--virtual')).toBeTruthy();
    const rows = container.querySelectorAll('.prowlarr-results-virtual-row[data-index]');
    expect(rows.length).toBeGreaterThan(0);
    for (const row of rows) {
      expect(row.getAttribute('data-index')).not.toBeNull();
      expect((row as HTMLElement).style.paddingBottom).toBe(`${PROWLARR_ROW_GAP_PX}px`);
    }
  });

  it('estimateSize 使用 session 行高缓存', () => {
    recordProwlarrRowHeight(1, [10], 360);
    const results = Array.from({ length: 55 }, (_, index) => makeRelease(index));
    render(
      <ProwlarrVirtualResultsGrid
        results={results}
        selectedGuids={new Set()}
        submittedGuids={new Set()}
        downloadingGuid={null}
        batchDownloading={false}
        formatBytes={() => '1 KB'}
        formatTime={() => '—'}
        onToggle={noop}
        onDownload={noop}
      />
    );
    expect(sessionStorage.getItem(PROWLARR_ROW_HEIGHT_CACHE_KEY)).toContain('1:40');
  });
});
