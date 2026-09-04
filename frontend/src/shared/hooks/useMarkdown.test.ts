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

  it('renders fenced SVG while removing active content and external fetches', () => {
    const { html } = renderMarkdown(`\`\`\`svg
<svg viewBox="0 0 20 20" onload="alert(1)">
  <script>alert(1)</script>
  <image href="https://tracker.example/pixel.png" />
  <circle cx="10" cy="10" r="8" />
</svg>
\`\`\``);

    expect(html).toContain('<svg');
    expect(html).toContain('<circle');
    expect(html).not.toContain('script');
    expect(html).not.toContain('onload');
    expect(html).not.toContain('tracker.example');
  });

  it('extracts Mermaid fences across Windows newlines and language casing', () => {
    const { html, mermaid } = renderMarkdown('```Mermaid\r\nflowchart TD\r\nA["开始"] --> B["结束"]\r\n```');

    expect(mermaid).toHaveLength(1);
    expect(mermaid[0].code).toContain('flowchart TD');
    expect(html).toContain('data-mermaid-id="mermaid-0"');
    expect(html).not.toContain('```Mermaid');
  });

  it('does not fetch remote tracking images from markdown or raw HTML', () => {
    const { html } = renderMarkdown([
      '![tracker](https://tracker.example/pixel.png)',
      '<img src="//tracker.example/raw.png" alt="raw">',
    ].join('\n'));

    expect(html).not.toContain('src="https://tracker.example');
    expect(html).not.toContain('src="//tracker.example');
    expect(html).toContain('data-remote-image-blocked="true"');
  });
});
