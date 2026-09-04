// DOM helpers. Replaces the legacy `$` function that was duplicated across
// 4 files. With React we rarely reach for these, but they're useful for
// imperative escape hatches (focus management, scrollIntoView, etc.).

export function $<T extends Element = HTMLElement>(id: string): T | null {
  if (typeof document === 'undefined') return null;
  return document.getElementById(id) as T | null;
}

export function scrollElementIntoView(
  el: Element | null,
  opts: ScrollIntoViewOptions = { behavior: 'smooth', block: 'start' },
): void {
  if (el && typeof el.scrollIntoView === 'function') {
    el.scrollIntoView(opts);
  }
}
