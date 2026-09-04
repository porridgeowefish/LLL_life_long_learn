import { beforeEach, describe, expect, it, vi } from 'vitest';

import { applyTheme, isThemePreference, THEME_STORAGE_KEY } from './theme';

describe('theme', () => {
  beforeEach(() => {
    localStorage.clear();
    vi.stubGlobal('matchMedia', vi.fn(() => ({ matches: false })));
  });

  it('accepts only the four explicit theme preferences', () => {
    expect(isThemePreference('lychee-paper')).toBe(true);
    expect(isThemePreference('system')).toBe(false);
  });

  it('applies the resolved theme and caches the preference', () => {
    applyTheme('mountain-mist');
    expect(document.documentElement.dataset.theme).toBe('mountain-mist');
    expect(localStorage.getItem(THEME_STORAGE_KEY)).toBe('mountain-mist');
  });
});
