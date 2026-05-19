import { act, renderHook } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { fetchPreviewAction, isFetchPreviewSelectionLocked, useActionLoading } from './useActionLoading';

describe('useActionLoading', () => {
  it('run 期间 isActive 为 true，结束后恢复', async () => {
    const { result } = renderHook(() => useActionLoading());
    let resolve!: () => void;
    const pending = new Promise<void>((r) => {
      resolve = r;
    });

    expect(result.current.isActive('test')).toBe(false);

    let runPromise!: Promise<void>;
    act(() => {
      runPromise = result.current.run('test', async () => {
        await pending;
      });
    });

    expect(result.current.isActive('test')).toBe(true);
    expect(result.current.isBusy).toBe(true);

    await act(async () => {
      resolve();
      await runPromise;
    });

    expect(result.current.isActive('test')).toBe(false);
    expect(result.current.isBusy).toBe(false);
  });

  it('不同 action key 互不影响 isActive', async () => {
    const { result } = renderHook(() => useActionLoading());

    await act(async () => {
      await result.current.run(fetchPreviewAction.statusSubmitted, async () => undefined);
    });

    expect(result.current.isActive(fetchPreviewAction.statusSubmitted)).toBe(false);
    expect(result.current.isActive(fetchPreviewAction.statusPending)).toBe(false);
  });

  it('isAnyActive 可匹配多个 key', async () => {
    const { result } = renderHook(() => useActionLoading());
    let resolve!: () => void;
    const pending = new Promise<void>((r) => {
      resolve = r;
    });

    act(() => {
      void result.current.run(fetchPreviewAction.downloadRow(42), () => pending);
    });

    expect(
      result.current.isAnyActive(
        fetchPreviewAction.downloadRow(42),
        fetchPreviewAction.downloadRow(99)
      )
    ).toBe(true);
    expect(result.current.isAnyActive(fetchPreviewAction.statusSubmitted, fetchPreviewAction.statusPending)).toBe(
      false
    );

    await act(async () => {
      resolve();
    });
  });
});

describe('isFetchPreviewSelectionLocked', () => {
  it('批量下载时锁定勾选', () => {
    expect(isFetchPreviewSelectionLocked(fetchPreviewAction.batchDownload)).toBe(true);
  });

  it('单条下载或状态更新不锁定勾选', () => {
    expect(isFetchPreviewSelectionLocked(fetchPreviewAction.downloadRow(1))).toBe(false);
    expect(isFetchPreviewSelectionLocked(fetchPreviewAction.statusSubmitted)).toBe(false);
    expect(isFetchPreviewSelectionLocked(null)).toBe(false);
  });
});
