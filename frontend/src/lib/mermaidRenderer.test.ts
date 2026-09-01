import { describe, expect, it } from 'vitest';

import { normalizeMermaidSource } from './mermaidRenderer';

describe('normalizeMermaidSource', () => {
  it('removes transport noise without rewriting diagram semantics', () => {
    expect(normalizeMermaidSource('\uFEFFmermaid\r\nflowchart TD\r\nA["开始"]\u00a0--> B["结束"]')).toBe(
      'flowchart TD\nA["开始"] --> B["结束"]',
    );
  });
});
