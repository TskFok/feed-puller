import { useEffect, useState } from 'react';
import {
  PROWLARR_VIRTUALIZE_BASE_THRESHOLD,
  resolveProwlarrVirtualizeThreshold
} from './prowlarrVirtualizeThreshold';
import { useGridColumns } from './useGridColumns';

function readWorkspaceHeight(): number {
  if (typeof document === 'undefined') {
    return 0;
  }
  const workspace = document.querySelector('.workspace');
  return workspace instanceof HTMLElement ? workspace.clientHeight : 0;
}

export function useProwlarrVirtualizeThreshold(): number {
  const columnCount = useGridColumns();
  const [threshold, setThreshold] = useState(PROWLARR_VIRTUALIZE_BASE_THRESHOLD);

  useEffect(() => {
    const update = () => {
      setThreshold(resolveProwlarrVirtualizeThreshold(readWorkspaceHeight(), columnCount));
    };
    update();
    window.addEventListener('resize', update);
    const workspace = document.querySelector('.workspace');
    const resizeObserver =
      workspace instanceof HTMLElement && typeof ResizeObserver !== 'undefined'
        ? new ResizeObserver(update)
        : null;
    resizeObserver?.observe(workspace as HTMLElement);
    return () => {
      window.removeEventListener('resize', update);
      resizeObserver?.disconnect();
    };
  }, [columnCount]);

  return threshold;
}
