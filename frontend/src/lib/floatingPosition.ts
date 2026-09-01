export interface ViewportBounds {
  w: number;
  h: number;
  left?: number;
  top?: number;
}

/**
 * Return the part of the layout viewport that is actually visible.
 *
 * Fixed elements and DOMRects use layout-viewport coordinates. During browser
 * zoom, viewport panning, or an overlaid browser surface, the visual viewport
 * can be smaller and offset inside it, so both the size and origin matter.
 */
export function visibleViewportBounds(): Required<ViewportBounds> {
  const visual = window.visualViewport;
  return {
    w: visual?.width ?? window.innerWidth ?? document.documentElement.clientWidth ?? 0,
    h: visual?.height ?? window.innerHeight ?? document.documentElement.clientHeight ?? 0,
    left: visual?.offsetLeft ?? 0,
    top: visual?.offsetTop ?? 0,
  };
}

/**
 * Position a fixed panel near an anchor (screen coords), flipping/clamping to
 * keep it inside the viewport. `anchor.left` is the horizontal center of the
 * selected region, so the panel opens centered on the selection when possible.
 * `anchor.top` is treated as the top of the region to open below; if it would
 * overflow the bottom, the panel opens above.
 */
export function flipPosition(
  anchor: { top: number; left: number },
  size: { width: number; height: number },
  viewport: ViewportBounds,
  margin = 8,
): { top: number; left: number } {
  const viewportLeft = viewport.left ?? 0;
  const viewportTop = viewport.top ?? 0;
  const viewportRight = viewportLeft + viewport.w;
  const viewportBottom = viewportTop + viewport.h;
  let top = anchor.top + margin;
  if (top + size.height > viewportBottom) {
    top = Math.max(viewportTop + margin, anchor.top - size.height - margin);
  }
  let left = anchor.left - size.width / 2;
  if (left + size.width > viewportRight) {
    left = Math.max(viewportLeft + margin, viewportRight - size.width - margin);
  }
  if (left < viewportLeft + margin) left = viewportLeft + margin;
  return { top, left };
}

/**
 * Position a compact selection toolbar using viewport coordinates. Prefer the
 * space below the selection, flip above it when the lower edge is crowded,
 * and clamp both axes so the toolbar always remains actionable.
 */
export function selectionToolbarPosition(
  anchor: { top: number; bottom: number; centerX: number },
  size: { width: number; height: number },
  viewport: ViewportBounds,
  margin = 8,
): { top: number; left: number } {
  const viewportLeft = viewport.left ?? 0;
  const viewportTop = viewport.top ?? 0;
  const viewportRight = viewportLeft + viewport.w;
  const viewportBottom = viewportTop + viewport.h;
  const availableBelow = viewportBottom - anchor.bottom - margin;
  const availableAbove = anchor.top - viewportTop - margin;
  const preferredTop = availableBelow >= size.height || availableBelow >= availableAbove
    ? anchor.bottom + margin
    : anchor.top - size.height - margin;

  return {
    top: Math.min(
      Math.max(preferredTop, viewportTop + margin),
      Math.max(viewportTop + margin, viewportBottom - size.height - margin),
    ),
    left: Math.min(
      Math.max(anchor.centerX - size.width / 2, viewportLeft + margin),
      Math.max(viewportLeft + margin, viewportRight - size.width - margin),
    ),
  };
}
