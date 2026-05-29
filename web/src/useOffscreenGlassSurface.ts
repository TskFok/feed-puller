import { useEffect, type RefObject } from 'react';
import { GLASS_OFFSCREEN_CLASS } from './glassConstants';

type OffscreenGlassSurfaceOptions = {
  rootMargin?: string;
};

/**
 * 为单个玻璃表面（如 .table-wrap）在离屏时关闭 backdrop-filter。
 */
export function useOffscreenGlassSurface(
  surfaceRef: RefObject<HTMLElement | null>,
  enabled: boolean,
  deps: readonly unknown[],
  options: OffscreenGlassSurfaceOptions = {}
) {
  const { rootMargin = '80px 0px' } = options;

  useEffect(() => {
    const surface = surfaceRef.current;
    if (!surface || !enabled || typeof IntersectionObserver === 'undefined') {
      return undefined;
    }

    const observer = new IntersectionObserver(
      ([entry]) => {
        if (entry) {
          surface.classList.toggle(GLASS_OFFSCREEN_CLASS, !entry.isIntersecting);
        }
      },
      { root: null, rootMargin, threshold: 0 }
    );

    observer.observe(surface);
    return () => {
      observer.disconnect();
      surface.classList.remove(GLASS_OFFSCREEN_CLASS);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [surfaceRef, enabled, rootMargin, ...deps]);
}
