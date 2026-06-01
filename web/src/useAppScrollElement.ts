import { useLayoutEffect, useState, type RefObject } from 'react';
import { resolveAppScrollLayout, type AppScrollLayout } from './appScrollElement';

export function useAppScrollElement(
  anchorRef: RefObject<HTMLElement | null>,
  ...deps: unknown[]
): AppScrollLayout {
  const [layout, setLayout] = useState<AppScrollLayout>({ scrollElement: null, scrollMargin: 0 });

  useLayoutEffect(() => {
    const update = () => {
      setLayout(resolveAppScrollLayout(anchorRef.current));
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
  }, deps);

  return layout;
}
