import { useCallback, useLayoutEffect, useRef, useState } from 'react';
import { useWindowVirtualizer } from '@tanstack/react-virtual';
import { useGridColumns } from './useGridColumns';
import {
  PROWLARR_ROW_GAP_PX,
  PROWLARR_ROW_TOTAL_ESTIMATE_PX
} from './prowlarrLayoutConstants';
import {
  getCachedProwlarrRowEstimate,
  recordProwlarrRowHeight
} from './prowlarrRowHeightCache';
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
};

function titleLengthsForRow(results: ProwlarrRelease[], rowIndex: number, columnCount: number): number[] {
  const startIndex = rowIndex * columnCount;
  return results.slice(startIndex, startIndex + columnCount).map((release) => release.title.length);
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
  onDownload
}: ProwlarrVirtualResultsGridProps) {
  const columnCount = useGridColumns();
  const rowCount = Math.ceil(results.length / columnCount);
  const anchorRef = useRef<HTMLDivElement>(null);
  const [scrollMargin, setScrollMargin] = useState(0);

  const estimateRowSize = useCallback(
    (rowIndex: number) => getCachedProwlarrRowEstimate(columnCount, titleLengthsForRow(results, rowIndex, columnCount)),
    [columnCount, results]
  );

  const virtualizer = useWindowVirtualizer({
    count: rowCount,
    estimateSize: estimateRowSize,
    overscan: 2,
    scrollMargin
  });

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
    const update = () => {
      const el = anchorRef.current;
      if (!el) {
        return;
      }
      setScrollMargin(el.getBoundingClientRect().top + window.scrollY);
    };
    update();
    window.addEventListener('resize', update);
    const anchor = anchorRef.current;
    const resizeObserver =
      anchor && typeof ResizeObserver !== 'undefined' ? new ResizeObserver(update) : null;
    if (anchor) {
      resizeObserver?.observe(anchor);
    }
    return () => {
      window.removeEventListener('resize', update);
      resizeObserver?.disconnect();
    };
  }, [results.length, rowCount, columnCount]);

  useLayoutEffect(() => {
    virtualizer.measure();
  }, [scrollMargin, rowCount, columnCount, results.length]);

  return (
    <div ref={anchorRef} className="prowlarr-results-grid prowlarr-results-grid--virtual">
      <div
        className="prowlarr-results-virtual-spacer"
        style={{ height: virtualizer.getTotalSize(), position: 'relative', width: '100%' }}
      >
        {virtualizer.getVirtualItems().map((virtualRow) => {
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
                transform: `translateY(${virtualRow.start - virtualizer.options.scrollMargin}px)`,
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
