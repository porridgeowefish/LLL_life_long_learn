// markdown-render.js — Markdown + Mermaid + syntax highlighting.
// Uses CDN-loaded marked, DOMPurify, mermaid, highlight.js.

export async function renderMarkdown(targetEl, sourceMd) {
  if (!window.marked) {
    await loadScript('https://cdn.jsdelivr.net/npm/marked@12.0.2/marked.min.js');
  }
  if (!window.DOMPurify) {
    await loadScript('https://cdn.jsdelivr.net/npm/dompurify@3.1.6/dist/purify.min.js');
  }
  if (!window.mermaid) {
    await loadScript('https://cdn.jsdelivr.net/npm/mermaid@10.9.1/dist/mermaid.min.js');
    window.mermaid.initialize({ startOnLoad: false, theme: 'default' });
  }

  // Configure marked to preserve mermaid ```mermaid blocks.
  const renderer = new window.marked.Renderer();
  const origCode = renderer.code.bind(renderer);
  renderer.code = (code, lang) => {
    if (lang === 'mermaid') {
      const id = `mmd-${Math.random().toString(36).slice(2, 10)}`;
      return `<div class="mermaid" id="${id}">${escapeHtml(code)}</div>`;
    }
    return origCode(code, lang);
  };
  window.marked.setOptions({ renderer, breaks: false, gfm: true });

  const html = window.DOMPurify.sanitize(window.marked.parse(sourceMd || ''), {
    ADD_TAGS: ['div'],
    ADD_ATTR: ['class', 'id']
  });
  targetEl.innerHTML = html;

  // Render mermaid blocks after DOM is updated.
  const mermaidEls = targetEl.querySelectorAll('.mermaid');
  if (mermaidEls.length) {
    for (const el of mermaidEls) {
      try {
        const id = el.id || `mmd-${Math.random().toString(36).slice(2, 10)}`;
        const { svg } = await window.mermaid.render(id, el.textContent);
        el.innerHTML = svg;
      } catch (err) {
        el.innerHTML = `<pre style="color:#c44">Mermaid error: ${escapeHtml(err.message || err)}</pre>`;
      }
    }
  }
}

function loadScript(src) {
  return new Promise((resolve, reject) => {
    const s = document.createElement('script');
    s.src = src;
    s.onload = resolve;
    s.onerror = reject;
    document.head.appendChild(s);
  });
}

function escapeHtml(s) {
  return String(s).replace(/[&<>"']/g, c => ({
    '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;'
  }[c]));
}
