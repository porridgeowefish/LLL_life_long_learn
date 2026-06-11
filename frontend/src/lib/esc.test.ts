import { describe, expect, it } from 'vitest';
import { esc } from './esc';

describe('esc', () => {
  it('escapes HTML special characters', () => {
    expect(esc('<script>')).toBe('&lt;script&gt;');
    expect(esc('"hi" & \'bye\'')).toBe('&quot;hi&quot; &amp; &#39;bye&#39;');
  });

  it('returns plain text unchanged', () => {
    expect(esc('hello world')).toBe('hello world');
    expect(esc('')).toBe('');
  });

  it('handles Chinese / Unicode content', () => {
    expect(esc('项目 "测试"')).toBe('项目 &quot;测试&quot;');
  });
});
