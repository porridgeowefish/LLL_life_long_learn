import { describe, expect, it } from 'vitest';
import { formatBytes, truncate } from './format';

describe('formatBytes', () => {
  it('formats bytes / KB / MB', () => {
    expect(formatBytes(0)).toBe('0 B');
    expect(formatBytes(512)).toBe('512 B');
    expect(formatBytes(2048)).toBe('2.0 KB');
    expect(formatBytes(1024 * 1024)).toBe('1.00 MB');
  });
});

describe('truncate', () => {
  it('keeps short strings unchanged', () => {
    expect(truncate('short', 10)).toBe('short');
  });

  it('truncates long strings with ellipsis', () => {
    expect(truncate('a very long string', 10)).toBe('a very lo…');
  });

  it('handles exact-length boundary', () => {
    expect(truncate('1234567890', 10)).toBe('1234567890');
  });
});
