import { describe, expect, it } from 'vitest';

import { renderMarkdown } from './useMarkdown';

// Test the useMarkdown hook's rendering pipeline by importing the module
// and verifying the exported types/functions exist.
describe('useMarkdown module', () => {
  it('exports useMarkdown as a function', async () => {
    const mod = await import('./useMarkdown');
    expect(typeof mod.useMarkdown).toBe('function');
  });

  it('renders GitHub-flavored markdown tables', () => {
    const md = [
      '| 概念 | 作用 |',
      '| --- | --- |',
      '| NFA | 描述非确定性转移 |',
      '| DFA | 描述确定性转移 |',
    ].join('\n');

    const { html } = renderMarkdown(md);

    expect(html).toContain('<table>');
    expect(html).toContain('<thead>');
    expect(html).toContain('<tbody>');
    expect(html).toContain('<th>概念</th>');
    expect(html).toContain('<td>NFA</td>');
  });
});
