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
  viewport: { w: number; h: number },
  margin = 8,
): { top: number; left: number } {
  let top = anchor.top + margin;
  if (top + size.height > viewport.h) {
    top = Math.max(margin, anchor.top - size.height - margin);
  }
  let left = anchor.left - size.width / 2;
  if (left + size.width > viewport.w) {
    left = Math.max(margin, viewport.w - size.width - margin);
  }
  if (left < margin) left = margin;
  return { top, left };
}
