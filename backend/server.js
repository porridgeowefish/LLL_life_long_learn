import http from "node:http";
import fs from "node:fs/promises";
import path from "node:path";
import { spawn } from "node:child_process";
import { fileURLToPath } from "node:url";
import crypto from "node:crypto";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const PROJECT_ROOT = path.resolve(__dirname, "..");
const FRONTEND_ROOT = path.join(PROJECT_ROOT, "frontend");
const PORT = Number(process.env.PORT || 8787);
const CLAUDE_BIN = process.env.CLAUDE_BIN || "claude";
const WORKSPACE = process.env.WORKSPACE || PROJECT_ROOT;

const tasks = new Map();
const clients = new Set();

function sendJson(res, status, payload) {
  const body = JSON.stringify(payload);
  res.writeHead(status, {
    "content-type": "application/json; charset=utf-8",
    "content-length": Buffer.byteLength(body)
  });
  res.end(body);
}

function broadcast(event, data) {
  const payload = `event: ${event}\ndata: ${JSON.stringify(data)}\n\n`;
  for (const res of clients) res.write(payload);
}

function taskSnapshot(task) {
  return {
    id: task.id,
    title: task.title,
    prompt: task.prompt,
    status: task.status,
    createdAt: task.createdAt,
    startedAt: task.startedAt,
    finishedAt: task.finishedAt,
    exitCode: task.exitCode,
    cwd: task.cwd,
    command: task.command,
    logs: task.logs.slice(-400),
    resultText: task.resultText
  };
}

function pushLog(task, stream, text, parsed = null) {
  const entry = {
    at: new Date().toISOString(),
    stream,
    text,
    parsed
  };
  task.logs.push(entry);
  if (task.logs.length > 1200) task.logs.splice(0, task.logs.length - 1200);
  broadcast("task-log", { id: task.id, entry });
}

function updateTask(task, patch = {}) {
  Object.assign(task, patch);
  broadcast("task-update", taskSnapshot(task));
}

function readBody(req) {
  return new Promise((resolve, reject) => {
    let body = "";
    req.setEncoding("utf8");
    req.on("data", chunk => {
      body += chunk;
      if (body.length > 2_000_000) {
        reject(new Error("Request body too large"));
        req.destroy();
      }
    });
    req.on("end", () => resolve(body));
    req.on("error", reject);
  });
}

function parseClaudeLine(task, line) {
  const trimmed = line.trim();
  if (!trimmed) return;

  try {
    const obj = JSON.parse(trimmed);
    const text = extractText(obj);
    if (text) task.resultText += text;
    const summary = text || summarizeClaudeEvent(obj);
    if (summary) pushLog(task, "stdout", summary, obj);
  } catch {
    task.resultText += trimmed + "\n";
    pushLog(task, "stdout", trimmed);
  }
}

function summarizeClaudeEvent(obj) {
  if (!obj || typeof obj !== "object") return "";
  if (obj.type === "system") {
    if (obj.subtype === "init") {
      return `Claude initialized · model=${obj.model || "unknown"} · permission=${obj.permissionMode || "default"}`;
    }
    if (obj.subtype === "status") {
      return `Claude status · ${obj.status || "working"}`;
    }
    if (obj.subtype === "api_retry") {
      return `Claude API retry ${obj.attempt}/${obj.max_retries} · ${obj.error || obj.error_status} · wait ${Math.round((obj.retry_delay_ms || 0) / 1000)}s`;
    }
    if (obj.subtype === "hook_response" || obj.subtype === "hook_started") {
      return "";
    }
    return `Claude system · ${obj.subtype || obj.type}`;
  }
  if (obj.type === "assistant") return "Claude assistant message";
  if (obj.type === "result") return obj.result ? String(obj.result) : "Claude result received";
  if (obj.type) return `Claude event · ${obj.type}`;
  return "";
}

function extractText(obj) {
  if (!obj || typeof obj !== "object") return "";
  if (typeof obj.result === "string") return obj.result;
  if (typeof obj.text === "string") return obj.text;
  if (typeof obj.content === "string") return obj.content;
  if (Array.isArray(obj.content)) {
    return obj.content
      .map(part => {
        if (typeof part === "string") return part;
        if (part && typeof part.text === "string") return part.text;
        return "";
      })
      .join("");
  }
  if (obj.type === "assistant" && obj.message) return extractText(obj.message);
  if (obj.type === "content_block_delta" && obj.delta?.text) return obj.delta.text;
  return "";
}

function startClaudeTask({ title, prompt, cwd, permissionMode, model, effort }) {
  const id = crypto.randomUUID();
  const safeCwd = cwd && path.isAbsolute(cwd) ? cwd : WORKSPACE;
  const args = [
    "-p",
    "--verbose",
    "--output-format",
    "stream-json",
    "--include-partial-messages",
    "--permission-mode",
    permissionMode || "default"
  ];

  if (model) args.push("--model", model);
  if (effort) args.push("--effort", effort);
  args.push(prompt);

  const task = {
    id,
    title: title || prompt.slice(0, 48) || "Claude Code 任务",
    prompt,
    status: "running",
    createdAt: new Date().toISOString(),
    startedAt: new Date().toISOString(),
    finishedAt: null,
    exitCode: null,
    cwd: safeCwd,
    command: `${CLAUDE_BIN} ${args.map(a => a.includes(" ") ? JSON.stringify(a) : a).join(" ")}`,
    logs: [],
    resultText: "",
    child: null
  };

  const child = spawn(CLAUDE_BIN, args, {
    cwd: safeCwd,
    shell: false,
    env: { ...process.env, FORCE_COLOR: "0" },
    stdio: ["ignore", "pipe", "pipe"],
    windowsHide: true
  });
  task.child = child;
  tasks.set(id, task);

  let stdoutBuffer = "";
  child.stdout.setEncoding("utf8");
  child.stdout.on("data", chunk => {
    stdoutBuffer += chunk;
    const lines = stdoutBuffer.split(/\r?\n/);
    stdoutBuffer = lines.pop() || "";
    for (const line of lines) parseClaudeLine(task, line);
  });

  child.stderr.setEncoding("utf8");
  child.stderr.on("data", chunk => {
    for (const line of chunk.split(/\r?\n/).filter(Boolean)) {
      pushLog(task, "stderr", line);
    }
  });

  child.on("error", error => {
    pushLog(task, "stderr", error.message);
    updateTask(task, {
      status: "failed",
      finishedAt: new Date().toISOString(),
      exitCode: -1
    });
  });

  child.on("close", code => {
    if (stdoutBuffer.trim()) parseClaudeLine(task, stdoutBuffer);
    updateTask(task, {
      status: code === 0 ? "completed" : "failed",
      finishedAt: new Date().toISOString(),
      exitCode: code
    });
  });

  broadcast("task-created", taskSnapshot(task));
  return task;
}

async function serveStatic(req, res) {
  const url = new URL(req.url, `http://${req.headers.host}`);
  const pathname = decodeURIComponent(url.pathname);
  const filePath = pathname === "/"
    ? path.join(FRONTEND_ROOT, "index.html")
    : path.join(FRONTEND_ROOT, pathname);

  const normalized = path.normalize(filePath);
  const publicRoot = FRONTEND_ROOT;
  if (!normalized.startsWith(publicRoot)) {
    sendJson(res, 403, { error: "Forbidden" });
    return;
  }

  try {
    const data = await fs.readFile(normalized);
    const ext = path.extname(normalized).toLowerCase();
    const contentType = {
      ".html": "text/html; charset=utf-8",
      ".css": "text/css; charset=utf-8",
      ".js": "application/javascript; charset=utf-8",
      ".json": "application/json; charset=utf-8",
      ".svg": "image/svg+xml"
    }[ext] || "application/octet-stream";
    res.writeHead(200, { "content-type": contentType });
    res.end(data);
  } catch {
    sendJson(res, 404, { error: "Not found" });
  }
}

const server = http.createServer(async (req, res) => {
  const url = new URL(req.url, `http://${req.headers.host}`);

  if (req.method === "GET" && url.pathname === "/api/events") {
    res.writeHead(200, {
      "content-type": "text/event-stream; charset=utf-8",
      "cache-control": "no-cache, no-transform",
      connection: "keep-alive",
      "x-accel-buffering": "no"
    });
    clients.add(res);
    res.write(`event: hello\ndata: ${JSON.stringify({ ok: true, tasks: [...tasks.values()].map(taskSnapshot) })}\n\n`);
    req.on("close", () => clients.delete(res));
    return;
  }

  if (req.method === "GET" && url.pathname === "/api/tasks") {
    sendJson(res, 200, { tasks: [...tasks.values()].map(taskSnapshot) });
    return;
  }

  if (req.method === "POST" && url.pathname === "/api/tasks") {
    try {
      const payload = JSON.parse(await readBody(req));
      if (!payload.prompt || typeof payload.prompt !== "string") {
        sendJson(res, 400, { error: "prompt is required" });
        return;
      }
      const task = startClaudeTask(payload);
      sendJson(res, 201, { task: taskSnapshot(task) });
    } catch (error) {
      sendJson(res, 400, { error: error.message });
    }
    return;
  }

  const cancelMatch = url.pathname.match(/^\/api\/tasks\/([^/]+)\/cancel$/);
  if (req.method === "POST" && cancelMatch) {
    const task = tasks.get(cancelMatch[1]);
    if (!task) {
      sendJson(res, 404, { error: "Task not found" });
      return;
    }
    if (task.child && task.status === "running") {
      task.child.kill("SIGTERM");
      updateTask(task, { status: "cancelled", finishedAt: new Date().toISOString() });
    }
    sendJson(res, 200, { task: taskSnapshot(task) });
    return;
  }

  if (req.method === "GET" && url.pathname === "/api/health") {
    sendJson(res, 200, { ok: true, claude: CLAUDE_BIN, workspace: WORKSPACE });
    return;
  }

  await serveStatic(req, res);
});

server.listen(PORT, () => {
  console.log(`DeepThink orchestrator running at http://localhost:${PORT}`);
  console.log(`Workspace: ${WORKSPACE}`);
});
