import { describe, expect, it } from 'vitest';

import { selectionEndpointRect } from './selectionGeometry';

function rangeWithRects(rects: DOMRect[]): Range {
  return {
    getClientRects: () => rects as unknown as DOMRectList,
    getBoundingClientRect: () => new DOMRect(20, 100, 600, 100),
  } as Range;
}

describe('selectionEndpointRect', () => {
  const firstLine = new DOMRect(420, 100, 200, 24);
  const lastLine = new DOMRect(20, 140, 300, 24);

  it('anchors a downward wrapped selection to the line where the pointer was released', () => {
    expect(selectionEndpointRect(
      rangeWithRects([firstLine, lastLine]),
      { x: 180, y: 152 },
    )).toMatchObject({ top: 140, left: 20, width: 300 });
  });

  it('anchors a reverse selection to the first visual line', () => {
    expect(selectionEndpointRect(
      rangeWithRects([firstLine, lastLine]),
      { x: 500, y: 112 },
    )).toMatchObject({ top: 100, left: 420, width: 200 });
  });

  it('uses the final selected line when there is no pointer position', () => {
    expect(selectionEndpointRect(rangeWithRects([firstLine, lastLine])))
      .toMatchObject({ top: 140, left: 20 });
  });
});
