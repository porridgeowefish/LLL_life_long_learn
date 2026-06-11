// tree.js — renders the project sidebar tree.

import { listProjects, getZone } from './api.js';

const ZONE_ICONS = {
  'Intro': '🌱', 'Explain': '📖', 'Practice': '✏️',
  'Extend': '🔗', 'Summary': '📝'
};
const ZONES = ['Intro', 'Explain', 'Practice', 'Extend', 'Summary'];

export async function renderTree(container, { onProjectClick, onZoneClick }) {
  const { projects } = await listProjects();
  const tree = buildTree(projects);

  container.innerHTML = '';
  if (!tree.length) {
    container.innerHTML = '<div class="tree-empty">暂无项目<br><small>点击「+ 新建项目」开始学习</small></div>';
    return;
  }

  for (const p of tree) {
    container.appendChild(projectNode(p, { onProjectClick, onZoneClick }));
  }
}

function buildTree(flat) {
  const byID = new Map(flat.map(p => [p.slug, { ...p, children: [] }]));
  const roots = [];
  for (const p of byID.values()) {
    if (p.parentProjectId && byID.has(p.parentProjectId)) {
      byID.get(p.parentProjectId).children.push(p);
    } else {
      roots.push(p);
    }
  }
  return roots;
}

function projectNode(p, { onProjectClick, onZoneClick }) {
  const wrap = document.createElement('div');
  wrap.className = 'tree-project';
  wrap.innerHTML = `
    <div class="tree-item tree-project-head" data-slug="${esc(p.slug)}">
      <span class="icon">📁</span><span>${esc(p.title)}</span>
    </div>
    <div class="tree-zones" data-slug="${esc(p.slug)}"></div>
    ${p.children.length ? '<div class="tree-subs"></div>' : ''}
  `;
  wrap.querySelector('.tree-project-head').addEventListener('click', e => {
    if (onProjectClick) onProjectClick(p.slug);
  });

  const zonesEl = wrap.querySelector('.tree-zones');
  for (const z of ZONES) {
    const zEl = document.createElement('div');
    zEl.className = 'tree-item tree-zone';
    zEl.dataset.zone = z;
    zEl.innerHTML = `<span class="icon">${ZONE_ICONS[z]}</span><span>${z}</span>`;
    zEl.addEventListener('click', () => { if (onZoneClick) onZoneClick(p.slug, z); });
    zonesEl.appendChild(zEl);
  }

  if (p.children.length) {
    const subsEl = wrap.querySelector('.tree-subs');
    for (const sub of p.children) {
      subsEl.appendChild(projectNode(sub, { onProjectClick, onZoneClick }));
    }
  }
  return wrap;
}

function esc(s) {
  const d = document.createElement('div'); d.textContent = s; return d.innerHTML;
}
