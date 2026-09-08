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

  it('replaces an unterminated mermaid fence tail with a pending placeholder instead of a code wall', () => {
    // Streaming: the closing fence has not arrived yet. The whole remaining
    // message must not collapse into one giant <pre> block.
    const md = '先说明方案。\n\n```mermaid\nflowchart TD\n    A["开始"] --> B["下一步"]';

    const { html, mermaid } = renderMarkdown(md);

    expect(html).not.toContain('<pre>');
    expect(html).toContain('data-mermaid-id="mermaid-pending"');
    // The unfinished code is staged for later rendering, not dropped.
    expect(mermaid).toHaveLength(1);
    expect(mermaid[0].code).toContain('flowchart TD');
  });

  it('replaces an unterminated svg fence tail with a pending placeholder', () => {
    const md = '正文\n\n```svg\n<svg xmlns="http://www.w3.org/2000/svg"><rect width="3" height="3"/></svg>\n后续正文';

    const { html } = renderMarkdown(md);

    expect(html).not.toContain('<pre>');
    expect(html).toContain('mermaid-placeholder');
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

describe('healDanglingFence (plain code fences)', () => {
  it('frees prose swallowed by a stray unclosed fence in a finished message', () => {
    // The model closed its real code block, then emitted a stray ``` that
    // CommonMark treats as a new opening fence — everything after it used
    // to collapse into one giant <pre>.
    const md = '第一段。\n\n```js\nconst a = 1;\n```\n\n第二段正文说明。\n```\n第三段更多正文。';

    const { html } = renderMarkdown(md);

    // The tail prose must render as a paragraph, not vanish into a <pre>.
    expect(html).toContain('第三段更多正文');
    expect(html).toMatch(/<p>[^<]*第三段更多正文/);
    expect(html).not.toContain('<pre><code>第三段');
    expect(html).not.toContain('```');
  });

  it('closes an unterminated fence whose tail is real code (final state)', () => {
    const md = '说明\n\n```python\ndef f():\n    return 1\n\n\ndef g():\n    return 2';

    const { html } = renderMarkdown(md);

    expect(html).toContain('language-python');
    expect(html).toContain('def g');
  });

  it('keeps arrived code inside a stable <pre> while streaming', () => {
    // Streaming: the closing fence has not arrived yet, but the code that
    // HAS arrived should already render as a code block instead of
    // flashing as raw prose or one unfinished wall.
    const { html } = renderMarkdown('说明\n\n```js\nconst a = 1;', { streaming: true });

    expect(html).toContain('<pre>');
    expect(html).toContain('const a');
  });

  it('drops a fence with an empty tail so no empty code box flashes', () => {
    const { html } = renderMarkdown('说明\n\n```');

    expect(html).not.toContain('<pre>');
    expect(html).toContain('说明');
  });

  it('leaves well-formed fences untouched', () => {
    const md = '前文\n\n```js\nconst x = 2;\n```\n\n后文';

    const { html } = renderMarkdown(md);

    expect(html).toContain('language-js');
    expect(html).toContain('const x');
    expect(html).toContain('<p>后文</p>');
  });
});
