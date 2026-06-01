import { useCallback, useLayoutEffect, useRef } from 'react';
import { useVirtualizer } from '@tanstack/react-virtual';
import { useAppScrollElement } from './useAppScrollElement';
import { useGridColumns } from './useGridColumns';
import {
  PROWLARR_ROW_GAP_PX
} from './prowlarrLayoutConstants';
import {
  getCachedProwlarrRowEstimate,
  recordProwlarrRowHeight
} from './prowlarrRowHeightCache';
import type { ProwlarrVisibleRange } from './prowlarrResultsProgress';
import { ProwlarrReleaseCard, type ProwlarrReleaseCardProps } from './ProwlarrReleaseCard';
import type { ProwlarrRelease } from './types';

export type ProwlarrVirtualResultsGridProps = {
  results: ProwlarrRelease[];
  selectedGuids: Set<string>;
  submittedGuids: Set<string>;
  downloadingGuid: string | null;
  batchDownloading: boolean;
  formatBytes: ProwlarrReleaseCardProps['formatBytes'];
  formatTime: ProwlarrReleaseCardProps['formatTime'];
  onToggle: (guid: string) => void;
  onDownload: (release: ProwlarrRelease) => void;
  onVisibleRangeChange?: (range: ProwlarrVisibleRange) => void;
};

function titleLengthsForRow(results: ProwlarrRelease[], rowIndex: number, columnCount: number): number[] {
  const startIndex = rowIndex * columnCount;
  return results.slice(startIndex, startIndex + columnCount).map((release) => release.title.length);
}

function visibleRangeFromVirtualizer(
  results: ProwlarrRelease[],
  columnCount: number,
  virtualItems: Array<{ index: number }>
): ProwlarrVisibleRange | null {
  if (virtualItems.length === 0) {
    return null;
  }
  const firstRow = virtualItems[0].index;
  const lastRow = virtualItems[virtualItems.length - 1].index;
  return {
    firstItemIndex: firstRow * columnCount,
    lastItemIndex: Math.min((lastRow + 1) * columnCount - 1, results.length - 1),
    total: results.length
  };
}

export function ProwlarrVirtualResultsGrid({
  results,
  selectedGuids,
  submittedGuids,
  downloadingGuid,
  batchDownloading,
  formatBytes,
  formatTime,
  onToggle,
  onDownload,
  onVisibleRangeChange
}: ProwlarrVirtualResultsGridProps) {
  const columnCount = useGridColumns();
  const rowCount = Math.ceil(results.length / columnCount);
  const anchorRef = useRef<HTMLDivElement>(null);
  const { scrollElement, scrollMargin } = useAppScrollElement(anchorRef, results.length, rowCount, columnCount);

  const estimateRowSize = useCallback(
    (rowIndex: number) => getCachedProwlarrRowEstimate(columnCount, titleLengthsForRow(results, rowIndex, columnCount)),
    [columnCount, results]
  );

  const virtualizer = useVirtualizer({
    count: rowCount,
    getScrollElement: () => scrollElement,
    estimateSize: estimateRowSize,
    overscan: 2,
    scrollMargin
  });

  const virtualItems = virtualizer.getVirtualItems();

  const measureVirtualRow = useCallback(
    (node: HTMLDivElement | null) => {
      virtualizer.measureElement(node);
      if (!node) {
        return;
      }
      const rowIndex = Number(node.dataset.index);
      if (!Number.isFinite(rowIndex) || rowIndex < 0) {
        return;
      }
      const titleLengths = titleLengthsForRow(results, rowIndex, columnCount);
      requestAnimationFrame(() => {
        const height = node.offsetHeight;
        if (height > 0) {
          recordProwlarrRowHeight(columnCount, titleLengths, height);
        }
      });
    },
    [columnCount, results, virtualizer]
  );

  useLayoutEffect(() => {
    if (!onVisibleRangeChange) {
      return undefined;
    }
    const notify = () => {
      const range = visibleRangeFromVirtualizer(results, columnCount, virtualizer.getVirtualItems());
      if (range) {
        onVisibleRangeChange(range);
      }
    };
    notify();
    scrollElement?.addEventListener('scroll', notify, { passive: true });
    return () => {
      scrollElement?.removeEventListener('scroll', notify);
    };
  }, [columnCount, onVisibleRangeChange, results, scrollElement, virtualItems, virtualizer]);

  // 滚动容器变化时只重测 DOM 行高；勿调用 virtualizer.measure()，否则会清空
  // itemSizeCache 并回退到过小的 estimateSize，导致虚拟行 translateY 重叠。
  useLayoutEffect(() => {
    const frame = requestAnimationFrame(() => {
      const root = anchorRef.current;
      if (!root) {
        return;
      }
      root.querySelectorAll<HTMLDivElement>('.prowlarr-results-virtual-row[data-index]').forEach((node) => {
        virtualizer.measureElement(node);
      });
    });
    return () => cancelAnimationFrame(frame);
  }, [scrollElement, scrollMargin, rowCount, columnCount, results.length, virtualizer]);

  return (
    <div ref={anchorRef} className="prowlarr-results-grid prowlarr-results-grid--virtual">
      <div
        className="prowlarr-results-virtual-spacer"
        style={{ height: virtualizer.getTotalSize(), position: 'relative', width: '100%' }}
      >
        {virtualItems.map((virtualRow) => {
          const startIndex = virtualRow.index * columnCount;
          const rowItems = results.slice(startIndex, startIndex + columnCount);
          return (
            <div
              key={virtualRow.key}
              ref={measureVirtualRow}
              data-index={virtualRow.index}
              className="prowlarr-results-virtual-row"
              data-virtual-row={virtualRow.index}
              style={{
                position: 'absolute',
                top: 0,
                left: 0,
                width: '100%',
                boxSizing: 'border-box',
                paddingBottom: PROWLARR_ROW_GAP_PX,
                transform: `translateY(${virtualRow.start - scrollMargin}px)`,
                display: 'grid',
                gridTemplateColumns: `repeat(${columnCount}, minmax(0, 1fr))`,
                gap: `${PROWLARR_ROW_GAP_PX}px`
              }}
            >
              {rowItems.map((release) => (
                <ProwlarrReleaseCard
                  key={release.guid}
                  release={release}
                  selected={selectedGuids.has(release.guid)}
                  submitted={submittedGuids.has(release.guid)}
                  downloading={downloadingGuid === release.guid}
                  batchDownloading={batchDownloading}
                  formatBytes={formatBytes}
                  formatTime={formatTime}
                  onToggle={() => onToggle(release.guid)}
                  onDownload={() => onDownload(release)}
                />
              ))}
            </div>
          );
        })}
      </div>
    </div>
  );
}
