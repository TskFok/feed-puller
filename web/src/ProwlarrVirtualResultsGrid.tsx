import { useLayoutEffect, useRef, useState } from 'react';
import { useWindowVirtualizer } from '@tanstack/react-virtual';
import { useGridColumns } from './useGridColumns';
import { ProwlarrReleaseCard, type ProwlarrReleaseCardProps } from './ProwlarrReleaseCard';
import type { ProwlarrRelease } from './types';

const ROW_ESTIMATE_PX = 248;
const ROW_GAP_PX = 14;

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
    return () => window.removeEventListener('resize', update);
  }, [results.length, rowCount, columnCount]);

  const virtualizer = useWindowVirtualizer({
    count: rowCount,
    estimateSize: () => ROW_ESTIMATE_PX + ROW_GAP_PX,
    overscan: 2,
    scrollMargin
  });

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
              className="prowlarr-results-virtual-row"
              data-virtual-row={virtualRow.index}
              style={{
                position: 'absolute',
                top: 0,
                left: 0,
                width: '100%',
                transform: `translateY(${virtualRow.start - virtualizer.options.scrollMargin}px)`,
                display: 'grid',
                gridTemplateColumns: `repeat(${columnCount}, minmax(0, 1fr))`,
                gap: `${ROW_GAP_PX}px`
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
