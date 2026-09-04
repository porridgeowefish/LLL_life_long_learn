type MermaidBlock = { id: string; code: string };

let apiPromise: Promise<(typeof import('mermaid'))['default']> | null = null;
let renderQueue: Promise<void> = Promise.resolve();
let renderSequence = 0;

export function normalizeMermaidSource(source: string): string {
  return source
    .replace(/^\uFEFF/, '')
    .replace(/\r\n?/g, '\n')
    .replace(/\u00a0/g, ' ')
    .replace(/^\s*mermaid\s*\n/i, '')
    .trim();
}

function mermaidApi() {
  if (!apiPromise) {
    apiPromise = import('mermaid').then((module) => {
      module.default.initialize({
        startOnLoad: false,
        theme: 'neutral',
        securityLevel: 'strict',
        deterministicIds: true,
        suppressErrorRendering: true,
        flowchart: { htmlLabels: false, useMaxWidth: true },
      });
      return module.default;
    });
  }
  return apiPromise;
}

function renderSVG(source: string, namespace: string): Promise<string> {
  const code = normalizeMermaidSource(source);
  const task = renderQueue.then(async () => {
    const api = await mermaidApi();
    const id = `lll-${namespace.replace(/[^a-z0-9-]/gi, '-')}-${++renderSequence}`;
    const result = await api.render(id, code);
    return result.svg;
  });
  renderQueue = task.then(() => undefined, () => undefined);
  return task;
}

function showFallback(target: HTMLElement, code: string) {
  target.replaceChildren();
  const details = document.createElement('details');
  details.className = 'mermaid-fallback';
  const summary = document.createElement('summary');
  summary.textContent = '图表语法暂时无法解析，查看图表源码';
  const pre = document.createElement('pre');
  pre.textContent = normalizeMermaidSource(code);
  details.append(summary, pre);
  target.append(details);
}

/**
 * Mount every Mermaid placeholder under one Markdown host. Mermaid is loaded
 * and initialized once, and render calls are serialized because its global
 * SVG id registry is not safe to drive concurrently from several views.
 */
export function mountMermaidBlocks(host: HTMLElement, blocks: MermaidBlock[], namespace: string): () => void {
  let cancelled = false;
  void (async () => {
    for (const block of blocks) {
      const target = host.querySelector<HTMLElement>(`[data-mermaid-id="${block.id}"]`);
      if (!target || cancelled) continue;
      try {
        const svg = await renderSVG(block.code, namespace);
        if (!cancelled && target.isConnected) target.innerHTML = svg;
      } catch {
        if (!cancelled && target.isConnected) showFallback(target, block.code);
      }
    }
  })();
  return () => { cancelled = true; };
}
