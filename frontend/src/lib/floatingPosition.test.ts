import { describe, it, expect } from 'vitest';
import { flipPosition } from './floatingPosition';

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
