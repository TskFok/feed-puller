import { useEffect, useState } from 'react';

/** 与 styles.css 中 .prowlarr-results-grid 断点一致 */
export function useGridColumns(): number {
  const [columns, setColumns] = useState(1);

  useEffect(() => {
    if (typeof window === 'undefined' || typeof window.matchMedia !== 'function') {
      return undefined;
    }
    const mqTablet = window.matchMedia('(min-width: 640px)');
    const mqDesktop = window.matchMedia('(min-width: 1024px)');
    const update = () => {
      setColumns(mqDesktop.matches ? 3 : mqTablet.matches ? 2 : 1);
    };
    update();
    mqTablet.addEventListener('change', update);
    mqDesktop.addEventListener('change', update);
    return () => {
      mqTablet.removeEventListener('change', update);
      mqDesktop.removeEventListener('change', update);
    };
  }, []);

  return columns;
}
