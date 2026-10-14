/** Recover model output that omitted the interactive fence. Scripts still
 * never enter host HTML: only complete math islands become opaque frames. */
export function extractMathHTML(input: string, register: (code: string) => string): string {
  function islands(source: string) {
    // Balance div tags rather than stopping at the first nested closing div.
    // The script must immediately follow one or more complete containers.
    const starts = /<div\b[^>]*>/gi;
    let out = '', copied = 0, start: RegExpExecArray | null;
    while ((start = starts.exec(source))) {
      if (/\bdata-interactive-id\s*=/i.test(start[0])) continue;
      const tags = /<\/?div\b[^>]*>/gi;
      tags.lastIndex = start.index;
      let depth = 0, end = start.index, tag: RegExpExecArray | null;
      while ((tag = tags.exec(source))) {
        depth += /^<\//.test(tag[0]) ? -1 : 1;
        if (depth !== 0) continue;
        end = tags.lastIndex;
        const next = /^\s*(<div\b[^>]*>)/i.exec(source.slice(end));
        if (next && /\bdata-interactive-id\s*=/i.test(next[1])) break;
        if (next) { tags.lastIndex = end + next[0].length - next[1].length; continue; }
        break;
      }
      if (depth !== 0) break;
      if (end === start.index) continue;
      starts.lastIndex = end;
      const script = /^\s*<script\b([^>]*)>([\s\S]*?)<\/script\s*>/i.exec(source.slice(end));
      if (!script || /\bsrc\s*=/i.test(script[1]) || !/\b(?:JXG|LLLMath)\s*(?:\.|\[)/.test(script[2])) continue;
      const through = end + script[0].length;
      out += source.slice(copied, start.index) + register(source.slice(start.index, through));
      copied = through; starts.lastIndex = through;
    }
    return out + source.slice(copied);
  }
  // Protect code examples, including an incomplete fence. Only an html fence
  // containing math is a compatibility spelling of an interactive widget.
  const fences = /(^|\n)[ \t]*(`{3,}|~{3,})[ \t]*([^\r\n]*)\r?\n([\s\S]*?)(?:\r?\n[ \t]*\2[ \t]*(?=\r?\n|$)|$)/g;
  let out = '', copied = 0, fence: RegExpExecArray | null;
  while ((fence = fences.exec(input))) {
    out += islands(input.slice(copied, fence.index));
    const recovered = fence[3].trim().toLowerCase() === 'html' ? islands(fence[4]) : fence[4];
    out += recovered === fence[4] ? fence[0] : fence[1] + recovered;
    copied = fences.lastIndex;
  }
  return out + islands(input.slice(copied));
}
