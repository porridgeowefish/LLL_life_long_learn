interface InteractiveBlock { id: string; code: string }
export interface InteractiveRuntime { source: string; css: string }

export function buildInteractiveDocument(code: string, nonce: string, runtime?: InteractiveRuntime): string {
  const doc = new DOMParser().parseFromString(code, 'text/html');
  doc.querySelectorAll('base,iframe,object,embed,form,link,meta[http-equiv]').forEach(node => node.remove());
  doc.querySelectorAll('script').forEach(node => { if (node.src) node.remove(); else { node.removeAttribute('type'); node.setAttribute('nonce', nonce); } });
  // Inline event handlers would bypass the nonce boundary. Models should bind
  // listeners from their explicitly nonced local script instead.
  doc.querySelectorAll('*').forEach(node => { for (const attr of Array.from(node.attributes)) {
    if (/^on/i.test(attr.name)) node.removeAttribute(attr.name);
    if ((attr.name === 'href' || attr.name === 'action') && !attr.value.startsWith('#')) node.removeAttribute(attr.name);
  } });
  const policy = `default-src 'none'; script-src 'nonce-${nonce}'; style-src 'unsafe-inline'; img-src data: blob:; font-src data:; connect-src 'none'; frame-src 'none'; worker-src 'none'; object-src 'none'; form-action 'none'; base-uri 'none'`;
  const engine = runtime ? `<script nonce="${nonce}">${runtime.source.replace(/<\/script/gi, '<\\/script')}</script>` : '';
  return `<!doctype html><html><head><meta charset="utf-8"><meta http-equiv="Content-Security-Policy" content="${policy}"><meta name="referrer" content="no-referrer"><style>html{color-scheme:light}body{margin:0;padding:12px;font:14px/1.6 system-ui,sans-serif;overflow-wrap:anywhere}*{box-sizing:border-box}img,svg,canvas{max-width:100%}${runtime?.css ?? ''}</style>${engine}${doc.head.innerHTML}</head><body>${doc.body.innerHTML}<script nonce="${nonce}">new ResizeObserver(()=>parent.postMessage({type:'lll-gui-size',height:Math.min(900,Math.max(240,document.body.scrollHeight+24))},'*')).observe(document.body);</script></body></html>`;
}

export function mountInteractiveBlocks(host: HTMLElement, blocks: InteractiveBlock[]): () => void {
  const cleanups: Array<() => void> = [];
  let active = true;
  for (const block of blocks) {
    const target = host.querySelector<HTMLElement>(`[data-interactive-id="${block.id}"]`);
    if (!target) continue;
    const iframe = document.createElement('iframe');
    iframe.title = '交互教学图示'; iframe.setAttribute('sandbox', 'allow-scripts'); iframe.setAttribute('referrerpolicy', 'no-referrer');
    iframe.setAttribute('allow', "camera 'none'; microphone 'none'; geolocation 'none'; clipboard-read 'none'; clipboard-write 'none'");
    iframe.style.height = '320px';
    const nonce = crypto.randomUUID().replace(/-/g, '');
    // Parent index CSP permits only blob frames, preventing self-navigation to
    // remote sites or host API URLs even when an embedded script tries it.
    let url: string | null = null;
    const attach = (runtime?: InteractiveRuntime) => {
      if (!active) return;
      const documentSource = buildInteractiveDocument(block.code, nonce, runtime);
      if (typeof URL.createObjectURL === 'function') { url = URL.createObjectURL(new Blob([documentSource], { type: 'text/html' })); iframe.src = url; }
      else iframe.srcdoc = documentSource; // non-browser test environments
    };
    if (/\b(?:JXG|LLLMath)\b/.test(block.code)) {
      void import('./mathRuntime').then(module => attach(module.mathRuntime)).catch(() => {
        if (!active) return;
        const error = document.createElement('p'); error.setAttribute('role', 'alert');
        error.textContent = '数学图示未能加载，请刷新后重试。'; target.replaceChildren(error);
      });
    } else attach();
    const listener = (event: MessageEvent) => {
      if (event.source !== iframe.contentWindow || event.data?.type !== 'lll-gui-size') return;
      const height = event.data.height;
      if (typeof height === 'number' && Number.isFinite(height) && height >= 240 && height <= 900) iframe.style.height = `${Math.ceil(height)}px`;
    };
    window.addEventListener('message', listener);
    const fullscreen = document.createElement('button'); fullscreen.type = 'button'; fullscreen.className = 'media-fullscreen-toggle';
    fullscreen.setAttribute('aria-label', '全屏图示'); fullscreen.setAttribute('title', '全屏图示');
    fullscreen.onclick = () => { target.classList.toggle('media-fullscreen'); fullscreen.setAttribute('aria-pressed', String(target.classList.contains('media-fullscreen'))); };
    const escape = (event: KeyboardEvent) => { if (event.key === 'Escape') target.classList.remove('media-fullscreen'); };
    window.addEventListener('keydown', escape);
    target.replaceChildren(fullscreen, iframe);
    cleanups.push(() => { window.removeEventListener('message', listener); window.removeEventListener('keydown', escape); if (url) URL.revokeObjectURL(url); iframe.remove(); });
  }
  return () => { active = false; cleanups.forEach(fn => fn()); };
}
