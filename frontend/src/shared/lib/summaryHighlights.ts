export interface TextAnchor {
  text: string;
  start: number;
  end: number;
}

export interface SavedHighlight {
  id: string;
  charStart: number;
  charEnd: number;
  quoteSnapshot: string;
}

export function selectionToTextAnchor(root: HTMLElement, range: Range): TextAnchor | null {
  if (!root.contains(range.commonAncestorContainer)) return null;
  const raw = range.toString();
  const leading = raw.length - raw.trimStart().length;
  const trailing = raw.length - raw.trimEnd().length;
  const text = raw.trim();
  if (!text) return null;

  const prefix = range.cloneRange();
  prefix.selectNodeContents(root);
  prefix.setEnd(range.startContainer, range.startOffset);
  const start = prefix.toString().length + leading;
  return { text, start, end: start + raw.length - leading - trailing };
}

export function resolveHighlight(
  fullText: string,
  highlight: SavedHighlight,
): { start: number; end: number } | null {
  const { charStart, charEnd, quoteSnapshot } = highlight;
  if (
    charStart >= 0 &&
    charEnd > charStart &&
    fullText.slice(charStart, charEnd) === quoteSnapshot
  ) {
    return { start: charStart, end: charEnd };
  }
  if (!quoteSnapshot) return null;

  const searchStart = Math.max(0, charStart - 600);
  const searchEnd = Math.min(fullText.length, charEnd + 600);
  const nearby = fullText.slice(searchStart, searchEnd).indexOf(quoteSnapshot);
  if (nearby >= 0) {
    const start = searchStart + nearby;
    return { start, end: start + quoteSnapshot.length };
  }
  const fallback = fullText.indexOf(quoteSnapshot);
  return fallback < 0
    ? null
    : { start: fallback, end: fallback + quoteSnapshot.length };
}

export function overlapsExisting(
  anchor: Pick<TextAnchor, 'start' | 'end'>,
  highlights: SavedHighlight[],
): boolean {
  return highlights.some((item) =>
    Math.max(anchor.start, item.charStart) < Math.min(anchor.end, item.charEnd),
  );
}

export function applyHighlights(root: HTMLElement, highlights: SavedHighlight[]) {
  root.querySelectorAll('mark[data-summary-id]').forEach((mark) => {
    mark.replaceWith(...Array.from(mark.childNodes));
  });
  root.normalize();

  const fullText = root.textContent ?? '';
  const resolved = highlights
    .map((item) => ({ item, range: resolveHighlight(fullText, item) }))
    .filter((value): value is { item: SavedHighlight; range: { start: number; end: number } } =>
      value.range !== null,
    )
    .sort((a, b) => b.range.start - a.range.start);

  for (const { item, range } of resolved) {
    const walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT);
    const nodes: Array<{ node: Text; start: number; end: number }> = [];
    let offset = 0;
    while (walker.nextNode()) {
      const node = walker.currentNode as Text;
      const end = offset + node.data.length;
      if (Math.max(offset, range.start) < Math.min(end, range.end)) {
        nodes.push({ node, start: offset, end });
      }
      offset = end;
    }

    for (const entry of nodes.reverse()) {
      const localStart = Math.max(0, range.start - entry.start);
      const localEnd = Math.min(entry.node.data.length, range.end - entry.start);
      const before = entry.node.data.slice(0, localStart);
      const selected = entry.node.data.slice(localStart, localEnd);
      const after = entry.node.data.slice(localEnd);
      const mark = document.createElement('mark');
      mark.dataset.summaryId = item.id;
      mark.textContent = selected;
      entry.node.replaceWith(
        ...(before ? [document.createTextNode(before)] : []),
        mark,
        ...(after ? [document.createTextNode(after)] : []),
      );
    }
  }
}
