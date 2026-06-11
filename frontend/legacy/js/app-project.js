// app-project.js — main controller for project.html.

import { getHealth, listProjects, getProject, getZone, createProject, invokeAgent, getSession, followUp, cancelSession } from './api.js';
import { connectEvents } from './sse.js';
import { renderTree } from './tree.js';
import { renderMarkdown } from './markdown-render.js';

// === Global state ===
const state = {
  projectSlug: null,
  zoneName: 'Explain',
  sessionId: null,
  session: null,
};

// Learning phases displayed in the sidebar timeline (dot-line).
const ZONES = [
  { name: 'Intro',    label: '引入' },
  { name: 'Explain',  label: '讲解' },
  { name: 'Practice', label: '练习' },
  { name: 'Extend',   label: '拓展' },
  { name: 'Summary',  label: '总结' },
];

// === DOM helpers ===
const $ = id => document.getElementById(id);
const esc = s => { const d = document.createElement('div'); d.textContent = s; return d.innerHTML; };

// === Init ===
async function init() {
  bindUI();
  await loadHealth();
  await refreshTree();
  bindSSE();
  updateInvokeButton();
  renderZoneTimeline();

  // Pre-select project from URL query, or first available.
  const params = new URLSearchParams(location.search);
  const slug = params.get('id');
  if (slug) {
    selectProject(slug);
  } else {
    const { projects } = await listProjects();
    if (projects.length) selectProject(projects[0].slug);
  }
}

function bindUI() {
  $('new-project-btn').addEventListener('click', onCreateProject);
  $('zone-select').addEventListener('change', e => {
    state.zoneName = e.target.value;
    $('zone-name').textContent = state.zoneName;
    loadZoneOutput();
    updateInvokeButton();
    renderZoneTimeline();
  });
  $('invoke-btn').addEventListener('click', onInvokeAgent);
  $('drawer-close').addEventListener('click', () => $('terminal-drawer').classList.remove('open'));
  $('followup-send').addEventListener('click', onFollowUp);
  $('followup-input').addEventListener('keydown', e => { if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); onFollowUp(); } });
  $('cancel-session').addEventListener('click', onCancel);
}

// iter-02 only ships Explain Agent. Map zone → allowed agent id.
const ZONE_AGENTS = {
  'Explain': 'explain',
  // Intro/Practice/Extend/Summary agents come in later iterations.
};

function updateInvokeButton() {
  const btn = $('invoke-btn');
  const allowedAgent = ZONE_AGENTS[state.zoneName];
  if (allowedAgent) {
    btn.disabled = false;
    btn.textContent = `调用 ${allowedAgent === 'explain' ? '讲解' : allowedAgent}智能体`;
    btn.title = '';
  } else {
    btn.disabled = true;
    btn.textContent = '该阶段暂无智能体';
    btn.title = `iter-02 仅提供 Explain Agent，"${state.zoneName}" 阶段的智能体将在后续迭代加入`;
  }
}

async function loadHealth() {
  try {
    const h = await getHealth();
    $('connectionState').textContent = h.claude && h.claude.available ? '● 在线' : '● Claude 未就绪';
    $('connectionState').style.color = h.claude && h.claude.available ? 'var(--accent)' : 'var(--orange)';
    $('workspaceLabel').textContent = h.workspace || '';
  } catch {
    $('connectionState').textContent = '● 离线';
    $('connectionState').style.color = 'var(--pink)';
  }
}

async function refreshTree() {
  await renderTree($('project-tree'), {
    onProjectClick: slug => selectProject(slug),
    onZoneClick: (slug, zone) => { state.zoneName = zone; $('zone-select').value = zone; selectProject(slug); }
  });
}

async function selectProject(slug) {
  state.projectSlug = slug;
  try {
    const { project } = await getProject(slug);
    $('project-title').textContent = project.title;
    $('project-meta').textContent = `阶段: ${project.activeZone || '—'} · 状态: ${project.status}`;
    state.zoneName = $('zone-select').value;
    await loadZoneOutput();
    highlightTreeItem(slug);
  } catch (e) {
    console.error(e);
  }
}

function highlightTreeItem(slug) {
  document.querySelectorAll('.tree-project-head').forEach(el => {
    el.classList.toggle('active', el.dataset.slug === slug);
  });
}

async function loadZoneOutput() {
  if (!state.projectSlug) return;
  try {
    const { predecessors } = await getZone(state.projectSlug, state.zoneName);
    const preds = predecessors || [];
    $('predecessors').innerHTML = preds.length
      ? preds.map(p => `<span class="pred ${p.exists ? 'ok' : 'pending'}">${p.exists ? '✓' : '○'} ${p.zoneName}</span>`).join('')
      : '<em class="muted">无前置阶段（这是入口阶段）</em>';
    $('zone-name').textContent = state.zoneName;
    // Sidebar timeline re-rendered when zoneName changes; predecessors inform invokeBtn gating only.
  } catch (e) {
    console.error(e);
  }
  // Try to fetch the persisted zone output file.
  const zoneFile = state.zoneName === 'Summary' ? 'summary/summary.md' : `${state.zoneName.toLowerCase()}/output.md`;
  try {
    const r = await fetch(`/files/projects/${state.projectSlug}/${zoneFile}`);
    if (r.ok) {
      const text = await r.text();
      if (text.trim()) {
        renderMarkdown($('zone-output'), text);
        $('zone-status').textContent = '已生成';
        $('zone-status').className = 'badge badge-green';
        return;
      }
    }
  } catch {}
  $('zone-output').innerHTML = `<div class="empty">尚未生成 ${state.zoneName} 阶段输出。点击上方按钮开始。</div>`;
  $('zone-status').textContent = '空';
  $('zone-status').className = 'badge badge-blue';
}

function renderZoneTimeline() {
  const c = $('zone-timeline');
  if (!c) return;
  const currentIdx = ZONES.findIndex(z => z.name === state.zoneName);
  if (currentIdx < 0) {
    c.innerHTML = '';
    return;
  }
  c.innerHTML = ZONES.map((z, i) => {
    let cls = '';
    let mark = '';
    if (i < currentIdx) { cls = 'done'; mark = '已完成'; }
    else if (i === currentIdx) { cls = 'current'; mark = '进行中'; }
    return `<div class="zone-step-item ${cls}" data-zone="${z.name}">
      <span class="step-name">${z.label}</span>
      <span class="step-mark">${mark}</span>
    </div>`;
  }).join('');
  c.querySelectorAll('.zone-step-item').forEach(el => {
    el.addEventListener('click', () => {
      const zone = el.dataset.zone;
      state.zoneName = zone;
      $('zone-select').value = zone;
      $('zone-name').textContent = zone;
      loadZoneOutput();
      updateInvokeButton();
      renderZoneTimeline();
    });
  });
}

async function onCreateProject() {
  const title = prompt('项目标题:');
  if (!title) return;
  try {
    await createProject(title);
    await refreshTree();
  } catch (e) {
    alert('创建失败: ' + e.message);
  }
}

async function onInvokeAgent() {
  if (!state.projectSlug) return alert('请先选择项目');
  const agentId = ZONE_AGENTS[state.zoneName];
  if (!agentId) {
    alert(`当前阶段 "${state.zoneName}" 暂无对应智能体。\niter-02 仅提供 Explain Agent，其它阶段将在后续迭代加入。`);
    return;
  }
  const intent = $('invoke-intent').value.trim();
  if (!intent) return alert('请输入学习意图');
  $('invoke-btn').disabled = true;
  $('invoke-btn').textContent = '调用中…';
  try {
    const result = await invokeAgent(agentId, {
      projectId: state.projectSlug,
      zone: state.zoneName,
      intent,
      permissionMode: 'acceptEdits',
    });
    state.sessionId = result.session.id;
    state.session = result.session;
    openDrawer();
    appendStreamLine('success', `[${new Date().toLocaleTimeString()}] 会话 ${result.session.id.slice(0,8)} 已启动`);
    appendStreamLine('info', `Run dir: ${result.runDir}`);
    appendStreamLine('info', '[桌面已弹出 PowerShell 窗口，显示 Claude 真实执行流。请勿关闭。]');
    updateSessionInfo();
    pollUntilDone();
  } catch (e) {
    alert('调用失败: ' + e.message);
  } finally {
    updateInvokeButton();
  }
}

function openDrawer() {
  $('terminal-drawer').classList.add('open');
}

function appendStreamLine(level, text) {
  const stream = $('terminal-stream');
  if (stream.querySelector('.empty')) stream.innerHTML = '';
  const line = document.createElement('div');
  line.className = level;
  line.textContent = text;
  stream.appendChild(line);
  stream.scrollTop = stream.scrollHeight;
}

function updateSessionInfo() {
  if (!state.session) return;
  $('info-agent').textContent = state.session.agentId;
  $('info-zone').textContent = state.session.zoneName;
  $('info-state').textContent = state.session.state;
  $('info-turns').textContent = state.session.turns.length;
  $('session-state-badge').textContent = state.session.state;
  $('session-meta').textContent = state.session.id.slice(0, 12);
}

// Poll session every 2s; SSE is a "nice-to-have" overlay.
// P0-1 fix: main panel renders the persisted artifact (output.md), not Claude's
// live terminal text. Terminal text only feeds the audit drawer via appendStreamLine.
function pollUntilDone() {
  if (!state.sessionId) return;
  const timer = setInterval(async () => {
    try {
      const { session } = await getSession(state.sessionId);
      state.session = session;
      updateSessionInfo();
      if (['completed', 'failed', 'cancelled'].includes(session.state)) {
        clearInterval(timer);
        appendStreamLine(session.state === 'completed' ? 'success' : 'error',
          `[${new Date().toLocaleTimeString()}] 会话 ${session.state} (exit=${session.exitCode})`);
        if (session.state === 'completed') {
          const ok = await renderArtifactAfterCompletion(session);
          if (!ok) {
            appendStreamLine('error', '未能获取持久化的 output.md（Claude 可能未写入文件）。请查看 run 目录的 result.md');
          }
        }
      }
    } catch (e) {
      console.error(e);
    }
  }, 2000);
}

// renderArtifactAfterCompletion fetches the canonical artifact from disk and
// renders it. Tries zone output first, then per-run result.md as fallback.
// Returns true on success.
async function renderArtifactAfterCompletion(session) {
  if (!state.projectSlug) return false;
  const zone = state.zoneName === 'Summary'
    ? 'summary/summary.md'
    : `${state.zoneName.toLowerCase()}/output.md`;
  const candidates = [`/files/projects/${state.projectSlug}/${zone}`];
  // Add the latest assistant run's result.md as a fallback.
  const lastAsst = [...(session.turns || [])].reverse().find(t => t.type === 'assistant' && t.runDirRel);
  if (lastAsst) {
    candidates.push(`/files/projects/${state.projectSlug}/${lastAsst.runDirRel}/result.md`);
  }
  for (const url of candidates) {
    try {
      const r = await fetch(url);
      if (!r.ok) continue;
      const text = await r.text();
      if (text.trim()) {
        renderMarkdown($('zone-output'), text);
        $('zone-status').textContent = '已生成';
        $('zone-status').className = 'badge badge-green';
        appendStreamLine('info', `[artifact rendered from ${url}]`);
        return true;
      }
    } catch { /* try next */ }
  }
  return false;
}

async function onFollowUp() {
  if (!state.sessionId) return alert('无活跃会话');
  const text = $('followup-input').value.trim();
  if (!text) return;
  $('followup-input').value = '';
  appendStreamLine('info', `[user follow-up] ${text}`);
  try {
    await followUp(state.sessionId, text);
    pollUntilDone();
  } catch (e) {
    alert('追问失败: ' + e.message);
  }
}

async function onCancel() {
  if (!state.sessionId) return;
  if (!confirm('确认取消当前会话？')) return;
  try {
    await cancelSession(state.sessionId);
    appendStreamLine('error', '[cancelled]');
  } catch (e) { alert(e.message); }
}

function bindSSE() {
  connectEvents({
    'session-state': e => {
      if (state.sessionId && e.sessionId === state.sessionId) {
        state.session = state.session || { state: '' };
        state.session.state = e.state;
        updateSessionInfo();
      }
    },
    'terminal-output': e => {
      if (state.sessionId && e.sessionId === state.sessionId && e.text) {
        appendStreamLine('info', e.text.slice(0, 500));
      }
    },
    'session-completed': e => {
      if (state.sessionId && e.sessionId === state.sessionId) {
        appendStreamLine('success', `[session completed]`);
      }
    },
  });
}

init();
