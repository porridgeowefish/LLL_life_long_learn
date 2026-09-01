export interface SelectionRect {
  top: number;
  right: number;
  bottom: number;
  left: number;
  width: number;
  height: number;
}

interface ViewportPoint {
  x: number;
  y: number;
}

/**
 * Return the selected line nearest the pointer release position.
 *
 * A Range bounding box covers every wrapped line, so anchoring a toolbar to
 * that box can put it in the middle of the selection. Using the nearest actual
 * client rect keeps the toolbar beside the end where the learner finished
 * selecting, regardless of drag direction.
 */
export function selectionEndpointRect(
  range: Range,
  point?: ViewportPoint,
): SelectionRect | null {
  const rects = Array.from(range.getClientRects())
    .filter(isUsableRect)
    .map(toSelectionRect);

  if (rects.length > 0) {
    if (!point || !Number.isFinite(point.x) || !Number.isFinite(point.y)) {
      return rects[rects.length - 1];
    }

    return rects.reduce((nearest, rect) =>
      distanceSquaredToRect(point, rect) < distanceSquaredToRect(point, nearest)
        ? rect
        : nearest,
    rects[0]);
  }

  const fallback = range.getBoundingClientRect();
  return isUsableRect(fallback) ? toSelectionRect(fallback) : null;
}

function isUsableRect(rect: DOMRect): boolean {
  return rect.width > 0
    && rect.height > 0
    && Number.isFinite(rect.top)
    && Number.isFinite(rect.left)
    && Number.isFinite(rect.right)
    && Number.isFinite(rect.bottom);
}

function toSelectionRect(rect: DOMRect): SelectionRect {
  return {
    top: rect.top,
    right: rect.right,
    bottom: rect.bottom,
    left: rect.left,
    width: rect.width,
    height: rect.height,
  };
}

function distanceSquaredToRect(point: ViewportPoint, rect: SelectionRect): number {
  const dx = point.x < rect.left
    ? rect.left - point.x
    : point.x > rect.right
      ? point.x - rect.right
      : 0;
  const dy = point.y < rect.top
    ? rect.top - point.y
    : point.y > rect.bottom
      ? point.y - rect.bottom
      : 0;
  return dx * dx + dy * dy;
}
