import { describe, it, expect } from 'vitest';
import { flipPosition, selectionToolbarPosition } from './floatingPosition';

describe('flipPosition', () => {
  it('centers the panel on the selection when there is room', () => {
    const p = flipPosition({ top: 100, left: 500 }, { width: 300, height: 200 }, { w: 1000, h: 800 });
    expect(p).toEqual({ top: 108, left: 350 });
  });
  it('flips up when the panel would overflow the bottom', () => {
    const p = flipPosition({ top: 700, left: 100 }, { width: 300, height: 200 }, { w: 1000, h: 800 });
    expect(p.top).toBeLessThan(700); // moved above the anchor
  });
  it('clamps left when the centered panel would overflow the right', () => {
    const p = flipPosition({ top: 100, left: 900 }, { width: 300, height: 200 }, { w: 1000, h: 800 });
    expect(p.left).toBe(1000 - 300 - 8);
  });

  it('clamps to the viewport margin when the centered panel would overflow the left', () => {
    const p = flipPosition({ top: 100, left: 50 }, { width: 300, height: 200 }, { w: 1000, h: 800 });
    expect(p.left).toBe(8);
  });
});

describe('selectionToolbarPosition', () => {
  it('opens above a selection near the viewport bottom', () => {
    const p = selectionToolbarPosition(
      { top: 720, bottom: 744, centerX: 500 },
      { width: 280, height: 44 },
      { w: 1000, h: 760 },
    );

    expect(p.top).toBe(668);
    expect(p.top + 44).toBeLessThan(720);
  });

  it('keeps a wrapped toolbar inside a narrow viewport', () => {
    const p = selectionToolbarPosition(
      { top: 200, bottom: 224, centerX: 340 },
      { width: 260, height: 76 },
      { w: 360, h: 640 },
    );

    expect(p).toEqual({ top: 232, left: 92 });
    expect(p.left + 260).toBeLessThanOrEqual(360 - 8);
  });

  it('clamps inside an offset visual viewport instead of the larger layout viewport', () => {
    const p = selectionToolbarPosition(
      { top: 640, bottom: 664, centerX: 950 },
      { width: 280, height: 44 },
      { left: 420, top: 180, w: 600, h: 520 },
    );

    expect(p).toEqual({ top: 588, left: 732 });
    expect(p.left).toBeGreaterThanOrEqual(420 + 8);
    expect(p.left + 280).toBeLessThanOrEqual(420 + 600 - 8);
    expect(p.top + 44).toBeLessThan(640);
  });
});
