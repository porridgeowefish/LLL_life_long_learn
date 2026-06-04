const state = {
  tasks: new Map(),
  activeTaskId: null,
  health: null,
  resultMode: "terminal",
  renderSeq: 0
};

const templates = [
  {
    title: "完整知识解构",
    body: "请使用 DeepThink 工作流，输出：学习者定位、核心界定、本质拆解、概念关系图、MECE 知识体系、边界误区、知识迁移、复习材料。"
  },
  {
    title: "面试专项学习",
    body: "请围绕面试场景整理这个主题：高频追问、2 分钟回答、5 分钟展开、案例化表达、容易暴露短板的误区。"
  },
  {
    title: "复习材料生成",
    body: "请基于学习科学生成复习材料：24h/3d/1w/2-3w/1m+ 复习表、Anki 闪卡、自测题、费曼检验、刻意练习。"
  },
  {
    title: "知识图谱任务",
    body: "请把主题拆成 MECE 结构，输出 Mermaid 图、一级维度、二级知识点、80/20 关键节点和前置依赖。"
  },
  {
    title: "资料提取",
    body: "请读取当前工作区相关资料，提取与主题相关的信息、论据、例子和可复用表达，并标注来源文件。"
  },
  {
    title: "学习计划",
    body: "请把这个主题拆成 7 天学习计划，每天包含输入材料、输出物、自测题、复盘标准和下一步动作。"
  }
];

const els = {
  topic: document.querySelector("#topic"),
  goal: document.querySelector("#goal"),
  level: document.querySelector("#level"),
  domain: document.querySelector("#domain"),
  permissionMode: document.querySelector("#permissionMode"),
  cwd: document.querySelector("#cwd"),
  notes: document.querySelector("#notes"),
  promptPreview: document.querySelector("#promptPreview"),
  runBtn: document.querySelector("#runBtn"),
  copyPromptBtn: document.querySelector("#copyPromptBtn"),
  taskList: document.querySelector("#taskList"),
  terminal: document.querySelector("#terminal"),
  renderedResult: document.querySelector("#renderedResult"),
  resultMode: document.querySelector("#resultMode"),
  activeTaskTitle: document.querySelector("#activeTaskTitle"),
  activeTaskMeta: document.querySelector("#activeTaskMeta"),
  connectionState: document.querySelector("#connectionState"),
  runningCount: document.querySelector("#runningCount"),
  totalCount: document.querySelector("#totalCount"),
  taskHint: document.querySelector("#taskHint"),
  copyResultBtn: document.querySelector("#copyResultBtn"),
  cancelBtn: document.querySelector("#cancelBtn"),
  templateGrid: document.querySelector("#templateGrid")
};

if (window.marked) {
  marked.setOptions({
    gfm: true,
    breaks: false
  });
}

if (window.mermaid) {
  mermaid.initialize({
    startOnLoad: false,
    securityLevel: "strict",
    theme: "default"
  });
}

function buildPrompt() {
  return [
    "使用 Codex/Claude 全局 skill `deepthink` 的学习科学工作流完成任务。",
    "",
    `学习主题：${els.topic.value.trim()}`,
    `已有水平：${els.level.value}`,
    `学习目标：${els.goal.value}`,
    `熟悉领域：${els.domain.value.trim()}`,
    "",
    "任务要求：",
    els.notes.value.trim(),
    "",
    "输出要求：",
    "1. 用中文 Markdown 输出。",
    "2. 先给可执行摘要，再给完整内容。",
    "3. 包含可复制到 Obsidian 的笔记结构。",
    "4. 包含 3-5 张闪卡、3-5 道主动回忆题、1 个费曼检验、1-2 个刻意练习。",
    "5. 如读取了本地资料，标注来源文件名。"
  ].join("\n");
}

function refreshPrompt() {
  els.promptPreview.value = buildPrompt();
}

async function loadHealth() {
  const res = await fetch("/api/health");
  state.health = await res.json();
  if (!els.cwd.value) els.cwd.value = state.health.workspace;
}

async function loadTasks() {
  const res = await fetch("/api/tasks");
  const data = await res.json();
  data.tasks.forEach(task => state.tasks.set(task.id, task));
  if (!state.activeTaskId && data.tasks[0]) state.activeTaskId = data.tasks[0].id;
  render();
}

function connectEvents() {
  const events = new EventSource("/api/events");
  events.addEventListener("open", () => {
    els.connectionState.textContent = "已连接";
  });
  events.addEventListener("error", () => {
    els.connectionState.textContent = "重连中";
  });
  events.addEventListener("hello", event => {
    const data = JSON.parse(event.data);
    data.tasks.forEach(task => state.tasks.set(task.id, task));
    render();
  });
  events.addEventListener("task-created", event => {
    const task = JSON.parse(event.data);
    state.tasks.set(task.id, task);
    state.activeTaskId = task.id;
    render();
  });
  events.addEventListener("task-update", event => {
    const task = JSON.parse(event.data);
    state.tasks.set(task.id, task);
    render();
  });
  events.addEventListener("task-log", event => {
    const { id, entry } = JSON.parse(event.data);
    const task = state.tasks.get(id);
    if (!task) return;
    task.logs = task.logs || [];
    task.logs.push(entry);
    if (task.logs.length > 400) task.logs.shift();
    if (id === state.activeTaskId) renderTerminal(task);
  });
}

async function runTask() {
  refreshPrompt();
  const payload = {
    title: els.topic.value.trim(),
    prompt: els.promptPreview.value,
    cwd: els.cwd.value.trim(),
    permissionMode: els.permissionMode.value
  };
  els.runBtn.disabled = true;
  els.runBtn.textContent = "启动中";
  try {
    const res = await fetch("/api/tasks", {
      method: "POST",
      headers: { "content-type": "application/json" },
      body: JSON.stringify(payload)
    });
    const data = await res.json();
    if (!res.ok) throw new Error(data.error || "启动失败");
    state.tasks.set(data.task.id, data.task);
    state.activeTaskId = data.task.id;
    showView("tasks");
    render();
  } catch (error) {
    alert(error.message);
  } finally {
    els.runBtn.disabled = false;
    els.runBtn.innerHTML = '<i data-lucide="play"></i>调用 Claude Code';
    createIcons();
  }
}

async function cancelActiveTask() {
  const task = getActiveTask();
  if (!task || task.status !== "running") return;
  await fetch(`/api/tasks/${task.id}/cancel`, { method: "POST" });
}

function getActiveTask() {
  return state.activeTaskId ? state.tasks.get(state.activeTaskId) : null;
}

function render() {
  const tasks = [...state.tasks.values()].sort((a, b) => b.createdAt.localeCompare(a.createdAt));
  els.runningCount.textContent = tasks.filter(t => t.status === "running").length;
  els.totalCount.textContent = tasks.length;
  els.taskHint.textContent = tasks.length ? `${tasks.length} 个任务，点击左侧切换` : "等待任务启动";

  els.taskList.innerHTML = tasks.length ? tasks.map(task => `
    <button class="task-item ${task.id === state.activeTaskId ? "active" : ""}" data-task-id="${task.id}">
      <strong>${escapeHtml(task.title)}</strong>
      <div class="task-meta">
        <span>${formatTime(task.createdAt)}</span>
        <span class="status ${task.status}">${task.status}</span>
      </div>
    </button>
  `).join("") : `<div class="empty">还没有任务。先在上方生成提示词，然后调用 Claude Code。</div>`;

  document.querySelectorAll(".task-item").forEach(item => {
    item.addEventListener("click", () => {
      state.activeTaskId = item.dataset.taskId;
      render();
    });
  });

  renderTerminal(getActiveTask());
  createIcons();
}

function renderTerminal(task) {
  if (!task) {
    els.activeTaskTitle.textContent = "终端输出";
    els.activeTaskMeta.textContent = "未选择任务";
    els.terminal.textContent = "";
    renderMarkdown(null);
    syncResultMode();
    return;
  }

  els.activeTaskTitle.textContent = task.title;
  els.activeTaskMeta.textContent = `${task.status} · ${task.command || ""}`;
  const lines = (task.logs || []).map(entry => {
    const prefix = entry.stream === "stderr" ? "[err]" : "[out]";
    return `${formatTime(entry.at)} ${prefix} ${entry.text}`;
  });
  if (task.resultText) {
    lines.push("", "==== RESULT TEXT ====", task.resultText);
  }
  els.terminal.textContent = lines.join("\n");
  els.terminal.scrollTop = els.terminal.scrollHeight;
  renderMarkdown(task);
  syncResultMode();
}

async function renderMarkdown(task) {
  const markdown = task?.resultText?.trim();
  if (!markdown) {
    els.renderedResult.innerHTML = `<div class="empty">还没有可渲染的 Markdown 结果。</div>`;
    return;
  }

  const seq = ++state.renderSeq;
  const mermaidBlocks = [];
  const prepared = markdown.replace(/```mermaid\s*([\s\S]*?)```/gi, (_, diagram) => {
    const index = mermaidBlocks.push(diagram.trim()) - 1;
    return `<div class="mermaid-render" data-mermaid-index="${index}"></div>`;
  });

  const html = window.marked
    ? marked.parse(prepared)
    : `<pre>${escapeHtml(prepared)}</pre>`;
  els.renderedResult.innerHTML = window.DOMPurify
    ? DOMPurify.sanitize(html, { ADD_ATTR: ["target"] })
    : html;

  if (seq !== state.renderSeq || !window.mermaid) return;

  const blocks = [...els.renderedResult.querySelectorAll("[data-mermaid-index]")];
  for (const block of blocks) {
    const index = Number(block.dataset.mermaidIndex);
    const source = mermaidBlocks[index];
    if (!source) continue;
    try {
      const id = `mermaid-${Date.now()}-${index}-${Math.random().toString(16).slice(2)}`;
      const { svg } = await mermaid.render(id, source);
      if (seq === state.renderSeq) block.innerHTML = svg;
    } catch (error) {
      block.innerHTML = `<div class="mermaid-error">Mermaid 渲染失败：${escapeHtml(error.message || error)}</div><pre><code>${escapeHtml(source)}</code></pre>`;
    }
  }
}

function syncResultMode() {
  const terminalMode = state.resultMode === "terminal";
  els.terminal.classList.toggle("hidden", !terminalMode);
  els.renderedResult.classList.toggle("hidden", terminalMode);
  els.resultMode.querySelectorAll("button").forEach(btn => {
    btn.classList.toggle("active", btn.dataset.mode === state.resultMode);
  });
}

function renderTemplates() {
  els.templateGrid.innerHTML = templates.map((template, index) => `
    <button class="template" data-template="${index}">
      <strong>${template.title}</strong>
      <span>${template.body}</span>
    </button>
  `).join("");
  document.querySelectorAll(".template").forEach(btn => {
    btn.addEventListener("click", () => {
      els.notes.value = templates[Number(btn.dataset.template)].body;
      refreshPrompt();
      showView("compose");
    });
  });
}

function showView(name) {
  document.querySelectorAll(".view").forEach(view => {
    view.classList.toggle("hidden", !view.id.endsWith(name));
  });
  document.querySelectorAll("nav button").forEach(btn => {
    btn.classList.toggle("active", btn.dataset.view === name);
  });
  if (name === "compose") {
    document.querySelector("#view-tasks").classList.remove("hidden");
  }
}

function formatTime(value) {
  if (!value) return "";
  return new Date(value).toLocaleTimeString("zh-CN", { hour12: false });
}

function escapeHtml(value) {
  return String(value ?? "")
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;");
}

async function copyText(text) {
  await navigator.clipboard.writeText(text);
}

function createIcons() {
  if (window.lucide) lucide.createIcons();
}

["topic", "goal", "level", "domain", "notes"].forEach(id => {
  document.querySelector(`#${id}`).addEventListener("input", refreshPrompt);
});

els.copyPromptBtn.addEventListener("click", async () => {
  refreshPrompt();
  await copyText(els.promptPreview.value);
});

els.copyResultBtn.addEventListener("click", async () => {
  const task = getActiveTask();
  await copyText(task?.resultText || els.terminal.textContent);
});

els.runBtn.addEventListener("click", runTask);
els.cancelBtn.addEventListener("click", cancelActiveTask);

els.resultMode.addEventListener("click", event => {
  const button = event.target.closest("button[data-mode]");
  if (!button) return;
  state.resultMode = button.dataset.mode;
  renderTerminal(getActiveTask());
});

document.querySelectorAll("nav button").forEach(btn => {
  btn.addEventListener("click", () => showView(btn.dataset.view));
});

await loadHealth();
refreshPrompt();
renderTemplates();
await loadTasks();
connectEvents();
createIcons();
