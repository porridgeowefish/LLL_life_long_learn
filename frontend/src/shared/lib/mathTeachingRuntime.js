// This source is inlined into the opaque teaching frame, not imported into React.
/* global document, window, HTMLElement, JXG */
(function () {
  'use strict';
  const plots = new Set();
  let sequence = 0;
  const colors = { curve: '#477fa8', gradient: '#cc7947', point: '#244b62' };

  function finite(value) {
    if (!Number.isFinite(value)) throw new Error('函数值或参数不是有限数值');
    return value;
  }
  function interval(value, fallback) {
    const range = value || fallback;
    if (!Array.isArray(range) || range.length !== 2 || !range.every(Number.isFinite) || range[0] >= range[1]) {
      throw new Error('展示范围需要两个递增的有限数值');
    }
    return range.slice();
  }
  function container(value) {
    const node = typeof value === 'string' ? document.getElementById(value) : value;
    if (!(node instanceof HTMLElement)) throw new Error('找不到图示容器');
    return node;
  }
  function boardAt(node, options) {
    if (!node.id) node.id = 'lll-math-' + (++sequence);
    node.classList.add('jxgbox');
    node.style.width = '100%';
    if (!node.style.height) node.style.height = '340px';
    return JXG.JSXGraph.initBoard(node.id, {
      renderer: 'svg', showCopyright: false, showNavigation: false,
      pan: { enabled: false }, zoom: { enabled: false }, ...options,
    });
  }
  function managed(board, values) {
    let disposed = false;
    const plot = { ...values, board, update() { if (!disposed) board.update(); }, dispose() {
      if (disposed) return;
      disposed = true; plots.delete(plot); JXG.JSXGraph.freeBoard(board);
    } };
    plots.add(plot);
    return plot;
  }
  function requireFunction(f) {
    if (typeof f !== 'function') throw new Error('请传入JavaScript函数，不能传表达式字符串');
  }
  function gradient(f, x, y) {
    requireFunction(f); finite(x); finite(y);
    const hx = 1e-5 * Math.max(1, Math.abs(x));
    const hy = 1e-5 * Math.max(1, Math.abs(y));
    return [finite((f(x + hx, y) - f(x - hx, y)) / (2 * hx)),
      finite((f(x, y + hy) - f(x, y - hy)) / (2 * hy))];
  }
  function plot2D(target, f, options = {}) {
    requireFunction(f);
    const range = interval(options.range, [-4, 4]);
    const yr = interval(options.yRange, [-4, 4]);
    const board = boardAt(container(target), { boundingbox: [range[0], yr[1], range[1], yr[0]], axis: true, keepaspectratio: false });
    try {
      const curve = board.create('functiongraph', [x => { const n = f(x); return Number.isFinite(n) ? n : NaN; }, ...range], { strokeColor: colors.curve, strokeWidth: 2 });
      return managed(board, { curve });
    } catch (error) { JXG.JSXGraph.freeBoard(board); throw error; }
  }
  function surface3D(target, f, options = {}) {
    requireFunction(f);
    const range = interval(options.range, [-2, 2]);
    const zr = interval(options.zRange, [-4, 8]);
    const board = boardAt(container(target), { boundingbox: [-6, 6, 6, -6], axis: false, keepaspectratio: false });
    try {
      // Fit the whole bounding cube under every rotation, including tall
      // surfaces whose z extent is larger than the x/y display range.
      const width = range[1] - range[0];
      const cameraRadius = Math.max(1, Math.hypot(width, width, zr[1] - zr[0]) / width);
      const view = board.create('view3d', [[-4, -4], [8, 8], [range, range, zr]], {
        projection: 'parallel', r: cameraRadius, trackball: { enabled: true },
        xPlaneRear: { visible: false }, yPlaneRear: { visible: false }, zPlaneRear: { visible: false },
      });
      const steps = Math.min(48, Math.max(8, Math.round(Number(options.steps) || 24)));
      const surface = view.create('functiongraph3d', [(x, y) => { const n = f(x, y); return Number.isFinite(n) ? n : NaN; }, range, range], {
        stepsU: steps, stepsV: steps, strokeWidth: 0.7, strokeColor: colors.curve,
      });
      return managed(board, { view, surface });
    } catch (error) { JXG.JSXGraph.freeBoard(board); throw error; }
  }
  function gradientDemo(target, options = {}) {
    const f = options.f; requireFunction(f);
    if (options.gradient !== undefined) requireFunction(options.gradient);
    const root = container(target), range = interval(options.range, [-2, 2]);
    const clamp = value => Math.max(range[0], Math.min(range[1], finite(value)));
    const initial = options.point || [1, 1];
    let x = clamp(initial[0]), y = clamp(initial[1]);
    let rate = options.learningRate === undefined ? 0.1 : finite(options.learningRate);
    if (rate <= 0 || rate > 1) throw new Error('步长需要大于0且不超过1');
    const grad = () => {
      const g = options.gradient ? options.gradient(x, y) : gradient(f, x, y);
      if (!Array.isArray(g) || g.length !== 2) throw new Error('梯度需要两个有限数值');
      return g.map(finite);
    };
    finite(f(x, y)); grad();
    root.replaceChildren(); root.classList.add('lll-math-demo');
    const style = document.createElement('style');
    style.textContent = '.lll-math-demo{color:#244b62}.lll-math-demo .jxgbox{border:1px solid #d6e5ec;border-radius:10px;background:#f7fbfd}.lll-math-controls{display:grid;gap:9px;margin-top:12px}.lll-math-controls label{display:grid;grid-template-columns:72px minmax(0,1fr) 58px;align-items:center;gap:8px}.lll-math-controls input{width:100%;min-width:0;accent-color:#477fa8}.lll-math-actions{display:flex;flex-wrap:wrap;gap:8px;margin-top:10px}.lll-math-actions button{border:1px solid #bfd5e3;border-radius:7px;background:#edf6fc;color:#244b62;padding:7px 12px;cursor:pointer}.lll-math-values{display:block;margin-top:10px;font-variant-numeric:tabular-nums;overflow-wrap:anywhere}.lll-math-note{font-size:12px;color:#637a87;margin:8px 0}';
    const heading = document.createElement('h3'); heading.textContent = options.title || '二元函数与梯度'; heading.style.margin = '0 0 8px';
    const stage = document.createElement('div'); stage.id = 'lll-math-surface-' + (++sequence);
    const controls = document.createElement('div'); controls.className = 'lll-math-controls';
    const output = document.createElement('output'); output.className = 'lll-math-values'; output.setAttribute('aria-label', '函数与梯度'); output.setAttribute('aria-live', 'polite');
    const note = document.createElement('p'); note.className = 'lll-math-note';
    note.textContent = '拖动曲面可旋转。橙色箭头表示最陡上升方向（长度缩放）；下降沿负梯度。' + (options.gradient ? '使用给定的解析梯度。' : '梯度为中心差分数值近似；不可微点附近需核对。');
    root.append(style, heading, stage, controls, output, note);
    const plot = surface3D(stage, f, options);
    const point = plot.view.create('point3d', [() => [x, y, finite(f(x, y))]], { name: 'P', size: 4, fixed: true, strokeColor: colors.point, fillColor: colors.point });
    plot.view.create('line3d', [() => [x, y, f(x, y)], () => {
      const [gx, gy] = grad(), scale = 0.45 / Math.max(1, Math.hypot(gx, gy));
      return [x + gx * scale, y + gy * scale, f(x, y) + (gx * gx + gy * gy) * scale];
    }], { strokeColor: colors.gradient, strokeWidth: 3, lastArrow: true, visible: () => Math.hypot(...grad()) > 1e-8 });
    const sliders = [];
    function slider(label, limits, value, change) {
      const row = document.createElement('label'), text = document.createElement('span'), input = document.createElement('input'), shown = document.createElement('output');
      text.textContent = label; input.type = 'range'; input.min = String(limits[0]); input.max = String(limits[1]); input.step = label === '步长' ? '0.001' : '0.01'; input.value = String(value); input.setAttribute('aria-label', label === '步长' ? label : label + ' 坐标');
      input.addEventListener('input', () => { change(Number(input.value)); render(); });
      row.append(text, input, shown); controls.append(row); sliders.push({ input, shown });
    }
    slider('x', range, x, value => { x = value; }); slider('y', range, y, value => { y = value; }); slider('步长', [0.001, 1], rate, value => { rate = value; });
    const actions = document.createElement('div'); actions.className = 'lll-math-actions';
    function button(label, handler) { const node = document.createElement('button'); node.type = 'button'; node.textContent = label; node.addEventListener('click', handler); actions.append(node); }
    button('沿负梯度走一步', () => { const [gx, gy] = grad(); setPoint(x - rate * gx, y - rate * gy); });
    button('重置', () => { rate = options.learningRate === undefined ? 0.1 : options.learningRate; setPoint(initial[0], initial[1]); }); root.append(actions);
    function render() {
      const [gx, gy] = grad();
      output.textContent = 'f(x,y) = ' + finite(f(x, y)).toFixed(3) + ' · ∇f = (' + gx.toFixed(3) + ', ' + gy.toFixed(3) + ')';
      [x, y, rate].forEach((n, i) => { sliders[i].input.value = String(n); sliders[i].shown.textContent = n.toFixed(3); });
      plot.update();
    }
    function setPoint(nextX, nextY) { const cx = clamp(nextX), cy = clamp(nextY); x = cx; y = cy; render(); }
    render();
    return { board: plot.board, view: plot.view, point, setPoint, update: render, dispose: plot.dispose };
  }
  window.LLLMath = Object.freeze({ version: '1', plot2D, surface3D, gradient, gradientDemo });
  function showError() {
    const display = () => {
      if (document.querySelector('[data-math-error]')) return;
      const note = document.createElement('p'); note.setAttribute('role', 'alert'); note.dataset.mathError = 'true';
      note.textContent = '数学图示代码运行失败，可让教师修正这段演示。';
      note.style.cssText = 'padding:12px;border:1px solid #d7b9a2;border-radius:8px;color:#8a5133;background:#fff8f3';
      document.body.append(note);
    };
    if (document.body) display(); else window.addEventListener('DOMContentLoaded', display, { once: true });
  }
  window.addEventListener('error', showError);
  window.addEventListener('unhandledrejection', showError);
  window.addEventListener('pagehide', () => [...plots].forEach(plot => plot.dispose()));
}());
