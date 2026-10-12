/** Never shrink a diagram below this share of its natural size: wide charts
 * keep readable labels and the stage scrolls/pans instead of shrinking text. */
export const LEGIBLE_SCALE = 0.8;

/** Natural drawing width from Mermaid's viewBox, else a numeric width attribute. */
export function naturalDiagramWidth(svg: SVGSVGElement): number {
  const viewBox = svg.getAttribute('viewBox')?.trim().split(/[\s,]+/).map(Number);
  if (viewBox?.length === 4 && Number.isFinite(viewBox[2]) && viewBox[2] > 0) return viewBox[2];
  const width = Number.parseFloat(svg.getAttribute('width') ?? '');
  return Number.isFinite(width) && width > 0 && !/%$/.test(svg.getAttribute('width') ?? '') ? width : 0;
}

/** Size by the drawing's own width instead of the 300px replaced-element default. */
export function sizeDiagram(svg: SVGSVGElement) {
  const natural = naturalDiagramWidth(svg);
  svg.style.width = '100%';
  svg.style.height = 'auto';
  if (!natural) return;
  svg.style.maxWidth = `${Math.round(natural)}px`;
  svg.style.minWidth = `${Math.round(natural * LEGIBLE_SCALE)}px`;
}

/** Diagram controls use aria labels and CSS glyphs so they add no text to
 * review character offsets. Only vector media is transformed, never prose. */
export function mountDiagramControls(host: HTMLElement): () => void {
  const disposers: Array<() => void> = [];
  const install = () => host.querySelectorAll<SVGSVGElement>('.inline-svg > svg, .mermaid-placeholder > svg').forEach(svg => {
    const parent = svg.parentElement;
    if (!parent || parent.dataset.diagramControls) return;
    parent.dataset.diagramControls = 'true';
    const stage = document.createElement('div'); stage.className = 'diagram-stage';
    const tools = document.createElement('div'); tools.className = 'diagram-tools'; tools.setAttribute('role', 'toolbar'); tools.setAttribute('aria-label', '图示操作');
    parent.insertBefore(stage, svg); stage.append(svg); parent.prepend(tools); sizeDiagram(svg);
    let scale = 1, x = 0, y = 0, drag: { px: number; py: number; x: number; y: number } | null = null;
    const redraw = () => { svg.style.transformOrigin = 'center center'; svg.style.transform = `translate(${x}px,${y}px) scale(${scale})`; };
    const add = (label: string, glyph: string, action: () => void) => {
      const button = document.createElement('button'); button.type = 'button'; button.title = label; button.setAttribute('aria-label', label); button.dataset.glyph = glyph;
      button.onclick = action; tools.append(button);
    };
    add('放大图示', '+', () => { scale = Math.min(4, scale * 1.2); redraw(); });
    add('缩小图示', '−', () => { scale = Math.max(.25, scale / 1.2); redraw(); });
    add('复位图示', '↺', () => { scale = 1; x = y = 0; redraw(); });
    add('全屏图示', '⤢', () => parent.classList.toggle('media-fullscreen'));
    add('导出 SVG', '↓', () => {
      const clone = svg.cloneNode(true) as SVGSVGElement; clone.style.removeProperty('transform'); clone.style.removeProperty('transform-origin');
      clone.setAttribute('xmlns', 'http://www.w3.org/2000/svg');
      const url = URL.createObjectURL(new Blob([new XMLSerializer().serializeToString(clone)], { type: 'image/svg+xml' }));
      const link = document.createElement('a'); link.href = url; link.download = 'learning-diagram.svg'; link.click(); window.setTimeout(() => URL.revokeObjectURL(url), 1000);
    });
    const down = (event: PointerEvent) => { if (event.button !== 0) return; drag = { px: event.clientX, py: event.clientY, x, y }; stage.setPointerCapture?.(event.pointerId); };
    const move = (event: PointerEvent) => { if (!drag) return; x = drag.x + event.clientX - drag.px; y = drag.y + event.clientY - drag.py; redraw(); };
    const up = () => { drag = null; };
    const escape = (event: KeyboardEvent) => { if (event.key === 'Escape') parent.classList.remove('media-fullscreen'); };
    stage.addEventListener('pointerdown', down); stage.addEventListener('pointermove', move); stage.addEventListener('pointerup', up); stage.addEventListener('pointercancel', up); window.addEventListener('keydown', escape);
    disposers.push(() => {
      stage.removeEventListener('pointerdown', down); stage.removeEventListener('pointermove', move); stage.removeEventListener('pointerup', up); stage.removeEventListener('pointercancel', up); window.removeEventListener('keydown', escape);
      delete parent.dataset.diagramControls; parent.classList.remove('media-fullscreen');
      if (stage.parentElement === parent) { parent.insertBefore(svg, stage); stage.remove(); tools.remove(); }
    });
  });
  install(); const observer = new MutationObserver(install); observer.observe(host, { childList: true, subtree: true });
  return () => { observer.disconnect(); disposers.forEach(fn => fn()); };
}
