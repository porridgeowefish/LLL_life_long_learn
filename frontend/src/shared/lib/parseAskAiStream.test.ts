import { describe, it, expect } from 'vitest';
import { extractSSEData } from './parseAskAiStream';

describe('extractSSEData', () => {
  it('extracts complete data payloads and keeps the partial tail', () => {
    const buf = 'data: {"type":"text","content":"a"}\n\ndata: {"type":"done"}\n\ndata: {"type":"text","con';
    const { payloads, rest } = extractSSEData(buf);
    expect(payloads).toEqual(['{"type":"text","content":"a"}', '{"type":"done"}']);
    expect(rest).toBe('data: {"type":"text","con');
  });

  it('returns empty payloads when no terminator yet', () => {
    const { payloads, rest } = extractSSEData('data: partial only');
    expect(payloads).toEqual([]);
    expect(rest).toBe('data: partial only');
  });

  it('ignores non-data lines', () => {
    const { payloads } = extractSSEData('event: x\ndata: {"a":1}\n\n');
    expect(payloads).toEqual(['{"a":1}']);
  });
});
