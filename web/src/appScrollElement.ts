export type AppScrollLayout = {
  scrollElement: HTMLElement | null;
  scrollMargin: number;
};

/** 解析元素所在的应用滚动容器（workspace 或页面根节点） */
export function resolveAppScrollElement(anchor: Element | null): HTMLElement | null {
  if (typeof document === 'undefined') {
    return null;
  }

  const workspace = anchor?.closest('.workspace');
  if (workspace instanceof HTMLElement) {
    const { overflowY } = getComputedStyle(workspace);
    if (overflowY === 'auto' || overflowY === 'scroll' || overflowY === 'overlay') {
      return workspace;
    }
  }

  return document.documentElement;
}

/** 列表锚点相对滚动容器顶部的偏移，供虚拟列表 scrollMargin 使用 */
export function getAppListScrollMargin(anchor: Element | null, scrollElement: HTMLElement | null): number {
  if (!(anchor instanceof HTMLElement) || !scrollElement) {
    return 0;
  }

  const anchorRect = anchor.getBoundingClientRect();
  const scrollRect = scrollElement.getBoundingClientRect();
  return anchorRect.top - scrollRect.top + scrollElement.scrollTop;
}

export function resolveAppScrollLayout(anchor: Element | null): AppScrollLayout {
  const scrollElement = resolveAppScrollElement(anchor);
  return {
    scrollElement,
    scrollMargin: getAppListScrollMargin(anchor, scrollElement)
  };
}
