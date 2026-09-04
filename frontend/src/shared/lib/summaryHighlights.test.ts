import { describe, expect, it } from 'vitest';

import { applyHighlights, overlapsExisting, resolveHighlight } from './summaryHighlights';

describe('summary highlights', () => {
  it('falls back to the saved quote when offsets drift', () => {
    expect(resolveHighlight('prefix target suffix', {
      id: 's1',
      charStart: 0,
      charEnd: 6,
      quoteSnapshot: 'target',
    })).toEqual({ start: 7, end: 13 });
  });

  it('detects overlapping saved summaries', () => {
    expect(overlapsExisting({ start: 4, end: 8 }, [{
      id: 's1',
      charStart: 6,
      charEnd: 10,
      quoteSnapshot: 'text',
    }])).toBe(true);
  });

  it('marks text across nested elements without changing text content', () => {
    const root = document.createElement('div');
    root.innerHTML = '<p>alpha <strong>beta</strong> gamma</p>';
    applyHighlights(root, [{
      id: 's1',
      charStart: 6,
      charEnd: 16,
      quoteSnapshot: 'beta gamma',
    }]);
    expect(root.textContent).toBe('alpha beta gamma');
    expect(root.querySelectorAll('mark[data-summary-id="s1"]')).toHaveLength(2);
  });
});
