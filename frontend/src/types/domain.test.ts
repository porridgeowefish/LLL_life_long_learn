import { describe, expect, it, vi, beforeEach } from 'vitest';

import { ZONE_DISPLAY, ALL_ZONES } from './domain';

describe('ZONE_DISPLAY', () => {
  it('maps every zone to a Chinese string', () => {
    for (const zone of ALL_ZONES) {
      expect(ZONE_DISPLAY[zone]).toBeTruthy();
      // Chinese characters — no ASCII
      expect(ZONE_DISPLAY[zone]).toMatch(/[一-鿿]+/);
    }
  });

  it('has exactly 5 entries', () => {
    expect(Object.keys(ZONE_DISPLAY)).toHaveLength(5);
  });

  it('values are unique', () => {
    const values = Object.values(ZONE_DISPLAY);
    expect(new Set(values).size).toBe(values.length);
  });
});
