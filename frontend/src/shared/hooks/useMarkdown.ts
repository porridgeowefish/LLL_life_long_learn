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

import { useMemo } from 'react';

import { marked } from 'marked';
import DOMPurify from 'dompurify';
import katex from 'katex';

export interface MermaidBlock {
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

function extractSVG(md: string): { stripped: string; placeholders: Map<string, string> } {
  const placeholders = new Map<string, string>();
  let index = 0;
  let stripped = md.replace(/```svg\s*\n([\s\S]*?)```/gi, (_, source: string) => {
    const id = `INLINEVECTOR${index++}X`;
    const clean = DOMPurify.sanitize(source.trim(), {
      USE_PROFILES: { svg: true, svgFilters: true, html: false },
      FORBID_TAGS: ['script', 'foreignObject', 'iframe', 'object', 'embed', 'style', 'animate', 'set'],
      FORBID_ATTR: ['onload', 'onclick', 'onerror', 'onbegin', 'style'],
      ALLOWED_URI_REGEXP: /^(?:#|data:image\/(?:png|jpeg|jpg|gif|webp);base64,)/i,
    });
    placeholders.set(id, clean);
    return `<div class="inline-svg" data-inline-svg="${id}">${id}</div>`;
  });
  // Streaming tail: an svg fence whose closing ``` has not arrived yet must
  // not fall through to marked — an unclosed fence turns the entire rest of
  // the message into one giant <pre> code wall.
  const danglingSvg = /```svg\s*\n[\s\S]*$/i.exec(stripped);
  if (danglingSvg) {
    stripped = stripped.slice(0, danglingSvg.index)
      + '<div class="mermaid-placeholder"><span class="mermaid-pending">图示正在生成…</span></div>';
  }
  return { stripped, placeholders };
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
  out = out.replace(/\$([^$\n\r][^$\n\r]*?)\$/g, (_, code: string) => {
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
  const re = /```[ \t]*mermaid[ \t]*\r?\n([\s\S]*?)```/gi;
  let i = 0;
  let mdStripped = md.replace(re, (_, code: string) => {
    const id = `mermaid-${i++}`;
    blocks.push({ id, code: code.trim() });
    return `<div data-mermaid-id="${id}" class="mermaid-placeholder"><span class="mermaid-pending">图表将在回答完成后渲染</span></div>`;
  });
  // Streaming tail: same unclosed-fence hazard as svg — stage the unfinished
  // code as a pending block instead of letting marked build a code wall.
  const dangling = /```[ \t]*mermaid[ \t]*\r?\n[\s\S]*$/i.exec(mdStripped);
  if (dangling) {
    const code = mdStripped.slice(dangling.index + dangling[0].indexOf('\n') + 1);
    mdStripped = mdStripped.slice(0, dangling.index)
      + '<div data-mermaid-id="mermaid-pending" class="mermaid-placeholder"><span class="mermaid-pending">图表正在生成…</span></div>';
    blocks.push({ id: 'mermaid-pending', code: code.trim() });
  }
  return { mdStripped, blocks };
}

// healDanglingFence — guard against an unterminated plain code fence.
// marked follows CommonMark: an unclosed fence swallows every character
// after it into one giant <pre> (raw line breaks + code background —
// the "text disappears into a box" glitch). svg/mermaid already guard
// their own fences in their extractors; this covers plain ```/~~~ ones:
//   - empty tail: drop the stray opening fence (no empty box flash).
//   - streaming: append a temporary closing fence so the code that HAS
//     arrived renders as a stable block until the real closer lands.
//   - final: if the tail still looks like code, close the fence; if it
//     looks like prose the model never meant to fence, drop the opening
//     fence and hand the text back to normal markdown rendering.
function healDanglingFence(md: string, streaming: boolean): string {
  // Tolerate \r\n line endings like the mermaid extractor does; healed
  // output is only rebuilt when a dangling fence exists, and marked is
  // fine with plain \n.
  const lines = md.split('\n').map((line) => line.replace(/\r$/, ''));
  const fence = /^( {0,3})(`{3,}|~{3,})(.*)$/;
  let openMarker = '';
  let openLength = 0;
  let openLine = -1;
  for (let i = 0; i < lines.length; i++) {
    const m = fence.exec(lines[i]);
    if (!m) continue;
    const marker = m[2][0];
    const info = m[3].trim();
    if (openMarker === '') {
      // A backtick fence's info string must not contain backticks —
      // otherwise the line is just prose/inline-code, not a fence.
      if (marker === '`' && info.includes('`')) continue;
      openMarker = marker;
      openLength = m[2].length;
      openLine = i;
    } else if (marker === openMarker && m[2].length >= openLength && info === '') {
      openMarker = '';
    }
  }
  if (openMarker === '') return md;
  const tail = lines.slice(openLine + 1);
  if (!tail.join('\n').trim()) return lines.slice(0, openLine).join('\n');
  if (streaming || looksLikeCode(tail)) return `${md}\n\`\`\``;
  return [...lines.slice(0, openLine), ...tail].join('\n');
}

// looksLikeCode — cheap heuristic over non-blank lines: code tends to
// carry assignment/brace characters, leading indentation, or keyword
// openings. Prose (especially CJK prose) rarely hits any of these.
function looksLikeCode(lines: string[]): boolean {
  const content = lines.filter((line) => line.trim());
  if (content.length === 0) return false;
  const codeish = content.filter((line) =>
    /[{};=<>]/.test(line)
    || /^ {2,}\S/.test(line)
    || /^(?:def |class |function |const |let |var |import |from |return |if |for |while |print\(|console\.)/.test(line));
  return codeish.length * 4 >= content.length;
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

DOMPurify.addHook('afterSanitizeAttributes', (node) => {
  if (node.tagName?.toLowerCase() !== 'img') return;
  const src = node.getAttribute('src') ?? '';
  if (/^(?:https?:)?\/\//i.test(src)) {
    node.removeAttribute('src');
    node.setAttribute('data-remote-image-blocked', 'true');
  }
});

export function renderMarkdown(input: string, options?: { streaming?: boolean }): MarkdownRender {
  const streaming = options?.streaming ?? false;

  // Step 1: extract math BEFORE marked touches anything.
  const { stripped, placeholders } = extractMath(input);

  // Step 2: fenced SVG is treated as display media, sanitized before it can
  // enter the general Markdown HTML pipeline. External fetches and active SVG
  // features are forbidden; local fragment references remain available.
  const svg = extractSVG(stripped);

  // Step 3: extract mermaid blocks (placeholder form).
  const { mdStripped, blocks } = extractMermaid(svg.stripped);

  // Step 3.5: heal an unterminated plain code fence before marked can
  // turn the rest of the message into one giant <pre>.
  const healed = healDanglingFence(mdStripped, streaming);

  // Step 4: parse remaining markdown.
  const rawHtml = marked.parse(healed) as string;

  // Step 4: splice KaTeX-rendered HTML back into placeholders.
  // Placeholder IDs are alphanumeric tokens (MATHBLOCK0X / MATHINLINE3X),
  // which survived marked unchanged. replaceAll handles duplicates.
  let withMath = rawHtml;
  for (const [id, html] of placeholders) {
    withMath = withMath.split(id).join(html);
  }
  for (const [id, clean] of svg.placeholders) {
    withMath = withMath.split(id).join(clean);
  }

  // Step 5: sanitize. KaTeX tags/attrs are whitelisted so the MathML
  // half of the output (used for screen readers + copy) survives.
  const safe = DOMPurify.sanitize(withMath, {
    ADD_ATTR: ['data-mermaid-id', 'data-inline-svg', 'viewBox', ...KATEX_ATTRS],
    ADD_TAGS: KATEX_TAGS,
    FORBID_TAGS: ['script', 'foreignObject', 'iframe', 'object', 'embed'],
  });

  return { html: safe, mermaid: blocks };
}

export function useMarkdown(input: string, streaming = false): MarkdownRender {
  return useMemo<MarkdownRender>(() => renderMarkdown(input, { streaming }), [input, streaming]);
}
