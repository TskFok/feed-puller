import { useEffect, type RefObject } from 'react';
import { GLASS_OFFSCREEN_CLASS } from './glassConstants';

const DEFAULT_SELECTOR = '.prowlarr-release-card:not(.prowlarr-release-card--skeleton)';

type OffscreenGlassOptions = {
  selector?: string;
  rootMargin?: string;
};

/**
 * 在长列表网格上为离屏玻璃卡片关闭 backdrop-filter，减轻滚动合成开销。
 */
export function useOffscreenGlassGrid(
  containerRef: RefObject<HTMLElement | null>,
  enabled: boolean,
  deps: readonly unknown[],
  options: OffscreenGlassOptions = {}
) {
  const { selector = DEFAULT_SELECTOR, rootMargin = '120px 0px' } = options;

  useEffect(() => {
    const container = containerRef.current;
    if (!container || !enabled || typeof IntersectionObserver === 'undefined') {
      return undefined;
    }

    const targets = Array.from(container.querySelectorAll<HTMLElement>(selector));
    if (targets.length === 0) {
      return undefined;
    }

    const observer = new IntersectionObserver(
      (entries) => {
        for (const entry of entries) {
          entry.target.classList.toggle(GLASS_OFFSCREEN_CLASS, !entry.isIntersecting);
        }
      },
      { root: null, rootMargin, threshold: 0 }
    );

    for (const target of targets) {
      observer.observe(target);
    }

    return () => {
      observer.disconnect();
      for (const target of targets) {
        target.classList.remove(GLASS_OFFSCREEN_CLASS);
      }
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps -- deps 由调用方传入（如 results.length）
  }, [containerRef, enabled, rootMargin, selector, ...deps]);
}
