// api.js — fetch wrappers for the LLL backend.
// All functions return Promises that resolve to parsed JSON.

const API_BASE = '';

async function jsonOrThrow(res) {
  const text = await res.text();
  let body;
  try { body = text ? JSON.parse(text) : null; } catch { body = { raw: text }; }
  if (!res.ok) {
    const msg = (body && body.error) || res.statusText;
    throw new Error(`${res.status} ${msg}`);
  }
  return body;
}

export async function getHealth() {
  const r = await fetch(`${API_BASE}/api/health`);
  return jsonOrThrow(r);
}

export async function listProjects() {
  const r = await fetch(`${API_BASE}/api/projects`);
  return jsonOrThrow(r);
}

export async function getProject(slug) {
  const r = await fetch(`${API_BASE}/api/projects/${encodeURIComponent(slug)}`);
  return jsonOrThrow(r);
}

export async function getProjectTree(slug) {
  const r = await fetch(`${API_BASE}/api/projects/${encodeURIComponent(slug)}/tree`);
  return jsonOrThrow(r);
}

export async function getZone(slug, zone) {
  const r = await fetch(`${API_BASE}/api/projects/${encodeURIComponent(slug)}/zones/${zone}`);
  return jsonOrThrow(r);
}

export async function createProject(title, slug) {
  const r = await fetch(`${API_BASE}/api/projects`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ title, slug })
  });
  return jsonOrThrow(r);
}

// createProjectRich sends the 5-field rich form payload (UX-5).
// {title, why, current, target, standard}
export async function createProjectRich(payload) {
  const r = await fetch(`${API_BASE}/api/projects`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload)
  });
  return jsonOrThrow(r);
}

export async function createSubproject(parentSlug, title) {
  const r = await fetch(`${API_BASE}/api/projects/${encodeURIComponent(parentSlug)}/subprojects`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ title })
  });
  return jsonOrThrow(r);
}

export async function listAgents() {
  const r = await fetch(`${API_BASE}/api/agents`);
  return jsonOrThrow(r);
}

export async function invokeAgent(agentId, { projectId, zone, intent, permissionMode }) {
  const r = await fetch(`${API_BASE}/api/agents/${encodeURIComponent(agentId)}/invoke`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ projectId, zone, intent, permissionMode })
  });
  return jsonOrThrow(r);
}

export async function getSession(id) {
  const r = await fetch(`${API_BASE}/api/sessions/${encodeURIComponent(id)}`);
  return jsonOrThrow(r);
}

// listSessions — supports:
//   listSessions()                    → all
//   listSessions({recent:true, limit:10}) → recent
//   listSessions({active:true})       → in-flight
export async function listSessions(opts = {}) {
  const params = new URLSearchParams();
  if (opts.recent) params.set('recent', 'true');
  if (opts.active) params.set('active', 'true');
  if (opts.limit) params.set('limit', String(opts.limit));
  const qs = params.toString();
  const r = await fetch(`${API_BASE}/api/sessions${qs ? '?' + qs : ''}`);
  return jsonOrThrow(r);
}

export async function listActiveSessions() {
  const r = await fetch(`${API_BASE}/api/sessions/active`);
  return jsonOrThrow(r);
}

export async function followUp(id, text, permissionMode) {
  const r = await fetch(`${API_BASE}/api/sessions/${encodeURIComponent(id)}/follow-up`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ text, permissionMode })
  });
  return jsonOrThrow(r);
}

export async function cancelSession(id) {
  const r = await fetch(`${API_BASE}/api/sessions/${encodeURIComponent(id)}/cancel`, {
    method: 'POST'
  });
  return jsonOrThrow(r);
}
