import { describe, expect, it } from 'vitest';

import { ALL_ZONES, ZONE_DISPLAY, ZONE_FILENAME } from './domain';

describe('ZONE_DISPLAY', () => {
  it('maps every zone to a Chinese string', () => {
    for (const zone of ALL_ZONES) {
      expect(ZONE_DISPLAY[zone]).toBeTruthy();
      expect(ZONE_DISPLAY[zone]).toMatch(/[一-鿿]+/);
    }
  });

  it('has exactly the active zone entries', () => {
    expect(Object.keys(ZONE_DISPLAY)).toHaveLength(3);
    expect(ALL_ZONES).toEqual(['Intro', 'Explain', 'Practice']);
  });

  it('values are unique', () => {
    const values = Object.values(ZONE_DISPLAY);
    expect(new Set(values).size).toBe(values.length);
  });
});

it('does not expose retired zone metadata', () => {
  expect(ZONE_DISPLAY).not.toHaveProperty('Extend');
  expect(ZONE_DISPLAY).not.toHaveProperty('Summary');
  expect(ZONE_FILENAME).not.toHaveProperty('Extend');
  expect(ZONE_FILENAME).not.toHaveProperty('Summary');
});
