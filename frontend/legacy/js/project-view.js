// project-view.js — main project reading panel controller.

import { getProject, getZone, invokeAgent, getSession, followUp, cancelSession } from './api.js';
import { renderMarkdown } from './markdown-render.js';

let activeProject = null;
let activeZone = 'Explain';
let activeSessionId = null;
let liveSessionTurns = [];

export async function loadProject(slug, zone) {
  activeProject = slug;
  activeZone = zone || activeZone;
  const [{ project }, { predecessors }] = await Promise.all([
    getProject(slug), getZone(slug, activeZone)
  ]);
  document.getElementById('project-title').textContent = project.title;
  renderPredecessors(predecessors);
  await loadZoneOutput(slug, activeZone);
}

async function loadZoneOutput(slug, zone) {
  // We don't have a dedicated zone content endpoint yet; the run dir's
  // explain/output.md (if present) is what we want to show. For now,
  // we call getSession-by-project which isn't built. We'll wire this
  // when Phase G completes the artifact read endpoint.
  // TODO: fetch /api/projects/:id/zones/:zone/output (or equivalent)
  const out = document.getElementById('zone-output');
  out.innerHTML = `<div class="empty">${zone === 'Explain' ? '点击下方"调用讲解智能体"按钮开始学习。' : '此阶段暂无输出。'}</div>`;
}

function renderPredecessors(preds) {
  const el = document.getElementById('predecessors');
  if (!preds || !preds.length) {
    el.innerHTML = '<em>无前置阶段（这是入口阶段）</em>';
    return;
  }
  el.innerHTML = preds.map(p =>
    `<span class="pred ${p.exists ? 'ok' : 'pending'}">${p.exists ? '✓' : '○'} ${p.zoneName}</span>`
  ).join('');
}

export async function invokeExplainAgent() {
  if (!activeProject) return alert('请先选择一个项目');
  const intent = document.getElementById('invoke-intent').value.trim();
  if (!intent) return alert('请输入学习意图');
  document.getElementById('invoke-btn').disabled = true;
  document.getElementById('invoke-btn').textContent = '调用中…';
  try {
    const result = await invokeAgent('explain', {
      projectId: activeProject, zone: activeZone, intent
    });
    activeSessionId = result.session.id;
    liveSessionTurns = result.session.turns || [];
    openTerminalDrawer();
    renderTerminalOutput();
  } catch (e) {
    alert('调用失败：' + e.message);
  } finally {
    document.getElementById('invoke-btn').disabled = false;
    document.getElementById('invoke-btn').textContent = '调用讲解智能体';
  }
}

function openTerminalDrawer() {
  document.getElementById('terminal-drawer').classList.add('open');
}

function renderTerminalOutput() {
  // Pull the latest assistant turn text from the live session.
  const out = document.getElementById('terminal-stream');
  const lastAssistant = [...liveSessionTurns].reverse().find(t => t.type === 'assistant');
  if (lastAssistant) {
    const target = document.getElementById('zone-output');
    renderMarkdown(target, lastAssistant.content);
  }
}

export function bindLiveEvents(sessionEventEmitter) {
  sessionEventEmitter.on('session-state', e => {
    if (e.sessionId !== activeSessionId) return;
    document.getElementById('session-state-badge').textContent = e.state;
  });
  sessionEventEmitter.on('turn-created', async e => {
    if (e.sessionId !== activeSessionId) return;
    // Refresh the session.
    const { session } = await getSession(activeSessionId);
    liveSessionTurns = session.turns;
    renderTerminalOutput();
  });
  sessionEventEmitter.on('session-completed', async e => {
    if (e.sessionId !== activeSessionId) return;
    document.getElementById('session-state-badge').textContent = 'completed';
    const { session } = await getSession(activeSessionId);
    liveSessionTurns = session.turns;
    renderTerminalOutput();
  });
}
