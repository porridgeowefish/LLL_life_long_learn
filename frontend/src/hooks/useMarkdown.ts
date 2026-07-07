// useMarkdown — render markdown to safe HTML with robust LaTeX math
// support. Wraps marked + DOMPurify + lazy-loaded mermaid.
//
// MATH PIPELINE (designed to be robust against marked's lexer eating
// backslash escapes inside math):
//
//   1. Pre-scan input for $$...$$ (display) and $...$ (inline) BEFORE
//      handing the markdown to marked.
//   2. Replace each math span with a unique plaintext placeholder
//      (e.g. MATH-BLOCK-7) — these survive marked unchanged.
//   3. Run marked.parse on the placeholder-stripped markdown.
//   4. Render each captured math span to KaTeX HTML directly.
//   5. Splice the rendered HTML back into the marked output.
//   6. DOMPurify-sanitize the final HTML (whitelist includes KaTeX tags).
//
// This bypasses marked-katex-extension (which runs INSIDE marked's lexer
// and was eating `\Sigma` etc.) and gives us full control over the math
// extraction order.

import { useEffect, useMemo, useState } from 'react';

import { marked } from 'marked';
import DOMPurify from 'dompurify';
import katex from 'katex';

interface MermaidBlock {
  id: string;
  code: string;
}

export interface MarkdownRender {
  html: string;
  mermaid: MermaidBlock[];
}

// Configure marked once. We do NOT add the KaTeX extension here — math
// extraction happens before marked runs (see extractMath).
marked.setOptions({
  gfm: true,
  breaks: false,
  async: false,
});

interface ExtractedMath {
  stripped: string;
  placeholders: Map<string, string>; // id → rendered KaTeX HTML
}

// extractMath scans markdown for $$...$$ and $...$, renders each via
// KaTeX, and substitutes a unique plaintext placeholder. Marked sees
// only the placeholders (and the surrounding text), so backslash escapes
// inside math (\Sigma, \frac, ...) are never exposed to marked's lexer.
function extractMath(md: string): ExtractedMath {
  const placeholders = new Map<string, string>();
  let i = 0;

  // 1) Display math $$...$$  (greedy, multiline).
  let out = md.replace(/\$\$([\s\S]+?)\$\$/g, (_, code: string) => {
    const id = `MATHBLOCK${i++}X`;
    let html = '';
    try {
      html = katex.renderToString(code.trim(), {
        displayMode: true,
        throwOnError: false,
        output: 'html',
        strict: 'ignore',
      });
    } catch (err) {
      // Render an inline error marker rather than crashing the page.
      html = `<code class="math-error">display math error: ${(err as Error).message}</code>`;
    }
    placeholders.set(id, html);
    return id;
  });

  // 2) Inline math $...$  (single line, no nested $).
  out = out.replace(/\$([^\$\n\r][^\$\n\r]*?)\$/g, (_, code: string) => {
    const id = `MATHINLINE${i++}X`;
    let html = '';
    try {
      html = katex.renderToString(code.trim(), {
        displayMode: false,
        throwOnError: false,
        output: 'html',
        strict: 'ignore',
      });
    } catch (err) {
      html = `<code class="math-error">inline math error: ${(err as Error).message}</code>`;
    }
    placeholders.set(id, html);
    return id;
  });

  return { stripped: out, placeholders };
}

function extractMermaid(md: string): { mdStripped: string; blocks: MermaidBlock[] } {
  const blocks: MermaidBlock[] = [];
  const re = /```mermaid\n([\s\S]*?)```/g;
  let i = 0;
  const mdStripped = md.replace(re, (_, code: string) => {
    const id = `mermaid-${i++}`;
    blocks.push({ id, code: code.trim() });
    return `<div data-mermaid-id="${id}" class="mermaid-placeholder"></div>`;
  });
  return { mdStripped, blocks };
}

// KaTeX emits MathML + nested HTML spans. DOMPurify's default allowlist
// drops MathML tags, so we extend it. KaTeX tags + attributes here are
// the minimum needed to preserve the rendering.
const KATEX_TAGS = [
  'math', 'mrow', 'mi', 'mo', 'mn', 'ms', 'msub', 'msup', 'msubsup',
  'mfrac', 'mroot', 'msqrt', 'mtext', 'mspace', 'mtable', 'mtr', 'mtd',
  'semantics', 'annotation', 'annotation-xml', 'menclose', 'merror',
  'mfenced', 'mover', 'munder', 'munderover', 'mstyle', 'mphantom',
];
const KATEX_ATTRS = [
  'class', 'style', 'mathvariant', 'encoding', 'xmlns', 'display',
  'scriptlevel', 'stretchy', 'fence', 'separator', 'lspace', 'rspace',
  'movablelimits', 'form', 'columnalign', 'rowalign', 'columnspacing',
  'rowspacing', 'columnlines', 'rowlines', 'frame', 'framespacing',
  'equalrows', 'equalcolumns', 'side', 'width', 'align', 'bevelled',
  'open', 'close', 'separators', 'accent', 'accentunder', 'dir',
  'linethickness', 'notation', 'position', 'subscriptshift',
  'superscriptshift', 'depth', 'height', 'voffset', 'lquote', 'rquote',
  'selection', 'href', 'mathbackground', 'mathcolor', 'mathsize',
];

DOMPurify.addHook('uponSanitizeAttribute', (_node, data) => {
  if (data.attrName === 'style') {
    const v = String(data.attrValue || '');
    if (/url\(|expression\(|javascript:/i.test(v)) {
      data.keepAttr = false;
      return;
    }
    data.keepAttr = true;
  }
});

export function renderMarkdown(input: string): MarkdownRender {
  // Step 1: extract math BEFORE marked touches anything.
  const { stripped, placeholders } = extractMath(input);

  // Step 2: extract mermaid blocks (placeholder form).
  const { mdStripped, blocks } = extractMermaid(stripped);

  // Step 3: parse remaining markdown.
  const rawHtml = marked.parse(mdStripped) as string;

  // Step 4: splice KaTeX-rendered HTML back into placeholders.
  // Placeholder IDs are alphanumeric tokens (MATHBLOCK0X / MATHINLINE3X),
  // which survived marked unchanged. replaceAll handles duplicates.
  let withMath = rawHtml;
  for (const [id, html] of placeholders) {
    withMath = withMath.split(id).join(html);
  }

  // Step 5: sanitize. KaTeX tags/attrs are whitelisted so the MathML
  // half of the output (used for screen readers + copy) survives.
  const safe = DOMPurify.sanitize(withMath, {
    ADD_ATTR: ['data-mermaid-id', ...KATEX_ATTRS],
    ADD_TAGS: KATEX_TAGS,
  });

  return { html: safe, mermaid: blocks };
}

export function useMarkdown(input: string): MarkdownRender {
  const [mermaidReady, setMermaidReady] = useState(false);

  useEffect(() => {
    let cancelled = false;
    if (!mermaidReady) {
      void import('mermaid').then((mod) => {
        if (cancelled) return;
        mod.default.initialize({
          startOnLoad: false,
          theme: 'neutral',
          securityLevel: 'strict',
        });
        setMermaidReady(true);
      });
    }
    return () => {
      cancelled = true;
    };
  }, [mermaidReady]);

  return useMemo<MarkdownRender>(() => renderMarkdown(input), [input]);
}
