import { useCallback, useState } from 'react';

export function useActionLoading() {
  const [active, setActive] = useState<string | null>(null);

  const isActive = useCallback((key: string) => active === key, [active]);

  const isAnyActive = useCallback((...keys: string[]) => keys.some((key) => active === key), [active]);

  const run = useCallback(async <T>(key: string, fn: () => Promise<T>): Promise<T> => {
    setActive(key);
    try {
      return await fn();
    } finally {
      setActive((current) => (current === key ? null : current));
    }
  }, []);

  return { active, isActive, isAnyActive, isBusy: active !== null, run };
}

export const fetchPreviewAction = {
  batchDownload: 'batch-download',
  statusSubmitted: 'status-submitted',
  statusPending: 'status-pending',
  downloadRow: (id: number) => `download-row:${id}`
} as const;

export function isFetchPreviewSelectionLocked(active: string | null): boolean {
  return active === fetchPreviewAction.batchDownload;
}
