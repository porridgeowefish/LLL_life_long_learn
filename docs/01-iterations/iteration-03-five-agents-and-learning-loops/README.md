# Iteration 03: Five Agents and Learning Loops

Status: draft
Owner: project maintainer
Last reviewed: 2026-06-09
Source of truth: this directory defines the third LLL delivery slice.

> 本文件是 iter-03 单一入口。原 DECISIONS.md / CARRYOVER.md 内容已并入下文
> "Key Decisions" 与 "Carryover" 两节（S1=B 决策）。流程治理项 G-02 / G-03
> 并入 "Process Constraints" 节（S2=A 决策）。

## Goal

Upgrade LLL from "Explain-only skeleton with five-zone folders" into a complete
five-agent learning workbench where each zone has its own agent behavior, and
the learner can traverse a full learning loop:

```text
Intro      -> 激发兴趣 + 收集入口问题
Explain    -> 结构化讲解 + 困惑标记 + 统一追问
Practice   -> 题量输入 + 整批作答 + 整批评估
Extend     -> 四关系维度引导性思考（因果/结构/重要性/同构，不画图）
Summary    -> 闪卡复习 + 用户主权总结
```

The slice must prove that:

```text
all five agents ship their own charter + primitive contracts
Explain supports "mark first, batch ask" confusion workflow
Practice ships the整批 submit + AI evidence-based evaluation loop (no 逐题)
Extend guides thinking across four relation lenses, no Mermaid graphs (D-Q-07)
Summary ships flashcards + report as switchable pages, never split-screen
project entry form collects exactly 4 learner-authored fields
learning trace schema is frozen (event-writing impl deferred)
practice evaluation persists file-first (json + md), no DB index this iter
Obsidian-style markdown editor (live-render) is the foundational interaction surface
```

## Scope

Included:

```text
HIGH PRIORITY — foundational interaction surface（编辑器是地基）
  Markdown editor: Obsidian-style live render, fallback to resizable split-pane
  笔记 / 练习作答 / 总结 / 困惑草稿 全部经过编辑器，编辑器烂则一切烂

CORE — five agents productized
  five agent charters + primitives (Intro/Explain/Practice/Extend/Summary)
  project entry contract: current ability / target ability / difficulty / deliverables
  advanced create options demoted to optional collapse
  Explain confusion marking (char-range data, paragraph-level UI first version OK)
  Explain batched "ask selected" button replacing any per-selection auto-prompt
  Practice count input (1-10, default 5) with explicit over-cap confirmation
  Practice answer draft + self-assessment + batch submit + AI evaluation (整批 only)
  Practice question/objective/confusion mapping
  Extend guided thinking across four relation lenses (causal/structural/importance/isomorphic) — NO Mermaid graphs (D-Q-07)
  Extend relation notes file learner-editable
  Summary flashcards generated from real learning traces
  Summary flashcard interaction (face/down, four-grade review)
  Summary knowledge report with deliverable status

SUPPORTING
  learning trace event model schema frozen (NO event-writing impl this iter)
  practice evaluation file-first persistence (json + md); DB index deferred
  Extend display name "拓展" (zone id stays extend)
  iteration 02 carryover: Claude launcher verification (C-01, fix applied)
```

Excluded:

```text
cloud deployment, user accounts, multi-tenant
mobile responsive
dark mode and i18n
global / cross-project search
memory agent as a sixth learning stage
database-first persistence (file remains primary)
automated long-term memory writes without user confirmation
character-level highlight UI polish (data model is char-range, UI ships paragraph first)
practice 逐题 evaluation — removed per D-Q-02 (整批 only, no per-question evaluate)
```

## Process Constraints (流程约束)

The alignment package's G-02 and G-03 are governance rules, not features:

```text
G-02 原型不是产品合同
  mock / prototype 中的每个交互都必须能在 USER_STORIES 或 Key Decisions 找到
  目的、前置、成功结果、失败结果。缺失页面或假链接标为缺口，不假装可用。

G-03 选择结果必须可被下一轮 AI 读取
  完成对齐后，用户直接修改 Markdown 对齐文档，AI 只读认同或需修改项，继续正式文档。
```

## Key Decisions (关键拍板)

Source: 4 需修改 + 4 Q-XX from 五智能体 alignment package.
Each decision records: 选项 / 拒绝方案 / 理由 / 影响 / 数据落点 / 验收.

---

### D-BTN-04: Markdown 编辑器 Obsidian 风格 [HIGH PRIORITY]

选项：默认 Obsidian 风格"编辑同时渲染"（live preview）；备选 双栏可拖拽调宽。

拒绝方案：
- 当前 v2 单栏切换"编辑 / 预览"。
- 不可调宽的双栏。

理由：
- 用户的原话："采用 obsisidian 的手段，编辑同时渲染。如果做不到，开双栏可以，但是保持现在的学习页面的宽度。而且两个页面之间大小可调整。"
- 用户的优先级补刀："高优先级，笔记体验不好那还怎么办"——编辑器是笔记、练习作答、总结、困惑草稿的统一交互面，是地基，烂了上层全部塌。
- 学习页面宽度（约 75ch）是阅读友好硬约束，不能丢。

影响：
- **优先级 HIGH**：建议在五智能体行为合同定稿后，作为底层交互面优先落地，因为它被 Practice / Summary / Explain notes 多处复用。
- 引入 `react-markdown` 或 `@uiw/react-md-editor` 等 live-preview 编辑器。
- 现有 OutputViewer 不受影响（只读，仍走 marked）。
- 编辑器封装为 `<MarkdownEditor mode="live" | "split" />`，默认 live；用户可在设置切换 split。
- split 模式下两栏比例 50/50，可拖拽分隔条，外层容器保持 75ch 居中。

数据落点：
- 编辑器写回原 artifact 路径（如 `summary/summary.md`），不引入新文件。
- 草稿状态用 localStorage 临时存（key 含 project_id + 文件路径），保存后清除。

验收：
- Obsidian 模式下输入 `**粗体**` 实时显示为粗体。
- split 模式下两栏同步，可拖拽调整宽度。
- 外层容器宽度保持学习页设计宽度。
- Practice 作答、Summary 总结、Explain 笔记均使用同一编辑器组件。

---

### D-E-04: Explain 困惑"先标记、后统一提问"

选项：选中段落或字符后只入收藏夹；用户在底部"统一提问"按钮一次性提交所有困惑。

拒绝方案：选中即触发 AI 追问（实时逐个）。

理由：
- 实时提问污染阅读流，用户失去主动权。
- 用户的原话："不能理解为实时讲解系统，而是我可以一次性选中困惑后。在按钮进行统一提问。而不是选中即提问。"
- E-02 自身 actions 字段已规定"追加问题 → 不立即发送，用户可编辑"——本决策 reinforce 该原则 + 追加统一提问入口。

影响（基于 E-02 consensus actions + E-04 feedback）：

选中浮层四动作（任何动作都**不**触发 invoke）：

| 浮层动作 | 行为 | 结果 |
|---------|------|------|
| 标困惑 | 保存选择范围 + 可补一句原因 | 进困惑清单（可用于练习生成） |
| 追加问题 | 选中文本作为引用放入底部提问框 | 不立即发送，用户可编辑 |
| 加入笔记 | 复制引用 + 来源到 explain/notes.md 草稿 | 保留来源链接 |
| 取消 | 关闭浮层 | 不保存 |

页面级新增按钮：

- "查看所有困惑"：打开按未解决/已解决筛选的列表，点击回到原文位置（E-02 actions）。
- "统一提问 (N)"：N = 困惑清单中待问条数；点击后弹注入预览（prompt + source_refs + quote_snapshot），确认后调一次 invoke，把 N 个困惑合并成单个 user turn。
- 单个困惑仍可单独提问（清单中某项 → 单独提交），是 N=1 的特例，非默认交互。

后端：

- invoke API 接受 `source_refs[]`（confusion id 列表），prompt assembly 把对应 quote_snapshot 注入。
- "进入练习只切换页面，不自动生成题，除非用户确认题量"（E-04 acceptance）。
- "针对困惑生成练习时，清楚列出会被采用的困惑"（E-04 acceptance）。

数据落点：
- `explain/confusions.json`（事实源）
  - 字段：`id` / `source_artifact_id` / `source_version` / `paragraph_id` / `char_start` / `char_end` / `quote_snapshot` / `created_at` / `state` (open/resolved/deleted) / `notes`
  - 数据模型从第一天就是字符级，UI 第一版可只展示段落级。
- `runs/<session>/prompt.md` 注入 source_refs 段落。
- `runs/<session>/transcript.ndjson` 记录每次"统一提问"作为单个 user turn。

验收：
- 选中浮层无 AI 触发动作。
- 已选清单可增删。
- "统一提问"按钮显示数量 N。
- 提交后清单中的对应 confusion 标记为 asked，但仍保留可继续追问。

---

### D-Q-01: Extend 显示名改为"拓展"

选项：UI 显示"拓展"，副标题 "Extend"；内部 zone id 保持 `extend`。

拒绝方案：
- 直接改 zone id 为 `domain` 或 `extension`。
- 只改显示名"领域"（用户原方案候选 A）。

理由：
- 用户的原话："改为拓展"。
- zone id 改动会触发：路径迁移 / API URL 变更 / session store 兼容性 / 历史数据 lookup 全部失效。
- 显示名是字符串，改动成本可控。

影响：
- 前端 i18n 资源（或 const 常量）改 `extend` → `拓展`，可在副标题或 tooltip 显示 "Extend"。
- 后端 zone registry 不动。
- 路径不动：`<project>/extend/` 目录名不变（产物改 output.md 引导文本，见 D-Q-07）。
- charters/extend.md / primitives 调用接口不动。

数据落点：
- `frontend/src/lib/constants.ts`（或类似）改显示名。
- `agents/charters/extend.md` 标题保持中英对照。

验收：
- 项目侧栏显示"拓展"。
- 路由 `/project/:id/extend` 仍有效。
- 文件系统 `<project>/extend/` 仍是这个目录名。

---

### D-Q-02: Practice 整批评估（逐题彻底删除）

选项：只支持整批提交后 AI 统一评估。**没有逐题评估**（任何形式都不提供）。

拒绝方案：
- 逐题评估作为默认交互。
- 逐题评估作为 opt-in 隐藏模式（Q2=A 决策：彻底删除，连隐藏开关也不要）。
- 评估单题后再让用户改答同题（提交后只允许新建 attempt）。

理由：
- 用户的原话："整批吧。AI 统一给出反馈。"
- 逐题评估会污染后续题目的独立性（用户根据反馈调整答案）。
- Q2 拍板：彻底删除逐题，简化 UI 与状态机，保证练习纯净。

影响：
- Practice 页面状态机：`drafting` → `self-assessing` → `submitted` → `evaluating` → `evaluated`。
- 无逐题 evaluate 按钮，无逐题 opt-in 开关，只支持"暂存草稿"。
- 提交前二次确认（题数 / 空题 / 自评缺失 / 提示使用）。
- 评估 API `POST /api/practice/{sessionId}/evaluate` 一次性接受所有 attempt，返回整体 evaluation。
- 评估失败时保留快照允许 retry。

数据落点：
- `practice/submissions/<attempt>/*.md` 每题一份草稿+提交快照。
- `practice/evaluations/<attempt>.json` 整批评估（事实源）。
- `practice/evaluations/<attempt>.md` 人类可读评估（事实源副本）。
- `database/index` 加速查询，可重建。

验收：
- 单题无 evaluate 按钮，设置中无逐题开关。
- 提交后 attempt 锁定，编辑创建新 attempt。
- 评估引用每题答案证据。
- 评估含自评偏差分析 + 下一轮建议。

---

### D-Q-03: 字符级标注（数据模型）+ 段落级 UI（首版可接受）

选项：数据模型必须含 char_start / char_end；UI 首版可只做段落级（UI 明确标注当前限制）。

拒绝方案：
- 数据模型从段落级开始，后续迁移到字符级。
- 用 xpath / CSS selector 代替字符偏移。

理由：
- 用户的原话："采用我的设计"。
- 事实源追溯：`frontend-designs/v3/DESIGN.md` D4（line 187-189）原文：
  > D4. AI 产出主舞台的"批注/写困惑"是绑定到段落还是字符级？
  > - 段落：实现简单，覆盖 90% 场景
  > - 字符：类似 Google Docs，体验好但复杂度上一个台阶
- HTML Q-03 验收三准则（用户认同）：
  1. 最终合同支持字符范围。
  2. 若首版只做段落级，UI 明确当前限制。
  3. 数据模型从第一天预留字符 range，避免后续迁移困难。
- "采用我的设计" = 采用上述三准则：合同字符级 / 首版 UI 段落级（标注限制）/ 数据从第一天预留 range。
- 数据迁移成本远高于首版 UI 实现成本；段落级 UI 是体验问题，字符级数据是合同问题，两者不耦合。

影响：
- `explain/confusions.json` 字段必含 `char_start` / `char_end` / `quote_snapshot`。
- 引用快照防止原文更新后定位漂移。
- 首版 UI 可让用户选整个段落，char_start / char_end 自动填段落起止。
- 未来 UI 升级到字符级无需迁移数据。

数据落点：
- `explain/confusions.json` schema 见 D-E-04。
- `paragraph_id` 作为辅助字段，便于 UI 高亮。

验收：
- 数据文件检查含 char_start / char_end。
- 原文更新后 UI 显示"定位可能漂移"警告，但仍可读 quote_snapshot。
- 未来 UI 升级不需要数据迁移。

---

### D-Q-05: Summary 两页可切换，绝对不允许分屏

选项：`/summary` 路由下两个 tab："闪卡" / "总结"，单页切换。

拒绝方案：
- 左右分屏（闪卡 | 总结）。
- 双路由 `/summary/flashcards` 和 `/summary/report` 并存。

理由：
- 用户的原话："可以说是两个可切换页面吧，绝对不允许分屏"。
- 分屏破坏专注；学习场景应单焦点。
- 双路由增加导航复杂度，且容易意外刷新丢状态。

影响：
- 路由保持 `/project/:id/summary`（一个路由）。
- SummaryPage 内部用 `useState<'flashcards' | 'report'>` 切换。
- 切换时保留滚动位置和闪卡进度（store 持久化）。
- 共享同一 source_refs 索引。

数据落点：
- `summary/flashcards.json`（闪卡数据）。
- `summary/summary.md`（总结文档）。
- `summary/next-steps.md`（下一步建议）。
- `summary/review-pack.md`（复习材料人类可读副本）。

验收：
- 路由只有一个 `/summary`。
- 切换 tab 不丢失闪卡复习进度。
- 屏幕宽度不限制为分屏条件。

---

### D-Q-04: 数据库选型与事实源边界（认同无修改）

首版不引入 SQLite；文件是事实源；评估必须有可读文件副本；定义重建/迁移/校验/冲突策略。
后续可能引入 SQLite 作为索引加速层，但绝不可成为事实源。

### D-Q-06: 本对齐包不扩 iter-02 scope（认同无修改）

本对齐包标记为对齐材料，不宣称已实现；用户确认后更新长期文档与 iter-03 user stories；
每个后续切片独立定义 API / 数据 / 测试 / 验收。iter-02 范围保持现状，所有新增需求进入 iter-03。

---

### D-Q-07: Extend 改为引导性思考（不画 Mermaid 关系图）

选项：Extend agent **不**产出 `.mmd` 关系图；改为用四种关系维度（因果 / 结构 / 重要性 / 同构）作为**引导性思考的脚手架**——agent 在 `extend/output.md` 中按这四个维度向学习者提出结构化追问，学习者通过 follow-up 回应。

拒绝方案：
- 画 4 张 Mermaid 关系图（causal / structure / importance / isomorphism.mmd）。
- MVP 先画 2 张。

理由：
- 用户原话："关系图不做的，这里按照这个关系进行引导性思考，而不是用mermaid画图。"
- Mermaid 图是 agent 单方面产出的静态结论；学习场景里"关系理解"的价值在学习者自己推理出关系，而非看一张图。
- 四种关系维度的 rigor（D-02 区分直接原因/机制/反事实、D-03 组成/边界、D-04 重要性维度、D-05 同构失效点）保留——但作为**追问脚手架**，不是图的边标签。
- 复用现有 invoke + follow-up 基建，Extend 不需新端点 / 新 UI / 新文件类型，成为最轻的 agent。

影响：
- 删除 `extend/{causal,structure,importance,isomorphism}.mmd` 产物与 schema。
- `extend/output.md` 成为引导性思考文本（按四维度组织追问）。
- `extend/relation-notes.md` 保留（学习者记录自己的关系推理）。
- 无 Mermaid 渲染 UI；OutputViewer 复用（只读 markdown）。

数据落点：
- `extend/output.md`（引导追问文本）
- `extend/relation-notes.md`（学习者笔记）

验收：
- Extend output.md 按因果 / 结构 / 重要性 / 同构四维度提出追问（非画图）。
- 每维度追问体现原 D-02..D-05 的 rigor（反事实 / 边界 / 维度 / 失效点）。
- 无 `.mmd` 文件产出，无 Mermaid 渲染。
- 学习者可 follow-up 回应，关系陈述可存 `relation-notes.md`。

---

### Deferred to iter-05+（本轮范围瘦身，用户确认）

下列项**不进 iter-03 实现**，defer iter-05 或更晚。iter-03 文档已同步移除其端点 / schema / 验收点：

```text
DATA-02  评估"双写合同"（file + DB 索引 / contentHash / 回滚 / 重建）
         → iter-03 只写 evaluations/<attempt>.json + .md 两文件；DB 索引层不实现
DATA-03  记忆建议子系统（memory proposals accept/rewrite/reject/undo）→ defer
S-06     总结 AI 建议（summary proposals accept/rewrite/reject）→ defer
DATA-01  trace 事件写入实现 → 保留 schema 冻结，砍事件写入
编辑器   split-pane 分栏 fallback → iter-03 只做 live-render
B 类 infra（iter-02 声明留给 iter-03 但未接，非本轮新增）
         PTY 会话连续性 / 重启 session 索引重建 / slog 结构化日志
         / zones 响应 artifact+session references / 跨平台 console → 全 defer iter-05+
```

## Carryover (遗留事项)

> iter-02 末期未闭环事项，必须在 iter-03 入口验证关闭。关闭后状态改 closed，不删除。

### C-01: Claude Code launcher 端到端调用验证

当前状态：**CLOSED — 六次修复全部 e2e 验证通过（2026-06-10）**。1–3 次（ShellExecute 开可见窗）；第 4 次（wrapper：claude 全路径解析 + prompt 位置参数自动喂入 + Stop 错误捕获 + spawn env 诊断日志）；第 5 次（编码：wrapper.ps1 加 UTF-8 BOM + Get-Content -Encoding UTF8，修中文项目路径乱码）；第 6 次（argv：PS 5.1 把含 ASCII 双引号的 prompt 劈成多参 → claude 只收前 642 字符，把 `”` → 中文 `”` 后单参全文送达）。`dist/lll.exe` 已 rebuild（16:21）。用户重启后端 e2e 复验通过：Intro Agent 实际产出完整引入文档，prompt 全文（4422 字符）到位。

根因（SO #30182508 / terraform-exec#570 + repro + 用户端到端确认）：
- Go 的 os/exec 用 CreateProcess；子控制台窗口的**可见性取决于父进程的控制台分配**。
- lll.exe 是 `go run` 起的服务进程（stdio 重定向/无正常交互控制台），其派生的子进程窗口即使 `CREATE_NEW_CONSOLE` 或 `cmd /c start` 也无法到达交互桌面——进程在跑（claude.exe 活着，数百 MB）但窗口永不出现。
- 前两次修复（CREATE_NEW_CONSOLE → cmd /c start）治标失败，因为没动到"父控制台依赖"这个根。
- 误判记录：曾以为是 stdin 重定向（IsInputRedirected）导致 claude TUI 退出——错。控制台窗口由 conhost.exe 托管，不属于 powershell.exe，故查 powershell 的 MainWindowHandle=0 不能证伪可见性。

具体变更：
- 开窗（1–3 次，ShellExecute，已用户确认可见）：spawn 改用 `ShellExecuteW("open", powershell.exe, "-NoProfile -ExecutionPolicy Bypass -NoExit -File wrapper.ps1", workDir, SW_SHOWNORMAL)`：走 Windows Shell（等同 Explorer 双击），强制交互桌面可见窗口，独立于调用方控制台。验证：headless（无控制台）Go 进程返回码 42（>32 成功）+ 用户确认绿窗口可见。重建 `console_windows.go`（ShellExecuteW 实现）+ `console_other.go`（非 Win stub）；wrapper.ps1 落盘 + -File 启动保留（规避 -Command 引号地狱）。
- 进 claude + 自动喂 prompt（第 4 次）：重写 `buildTUIWrapperScript`。根因：ShellExecute 派生的 powershell 子进程 env 可能与后端 probe env 不同，裸 `claude` 在子窗口里**可能解析不到**；而 `$ErrorActionPreference='Continue'` 让"找不到命令"成为非终止错误——不 catch、不写日志、静默跳过，窗口停在 banner（与 stderr.log 空白 + 用户"没看到 claude 页面"吻合）。修法：① claude 显式解析全路径（`Get-Command` + `~/.local/bin/claude.exe` + `AppData/.../claude.exe` fallback），脱钩 PATH；解析不到则红字报错 + 退出。② prompt 作为位置参数喂给 claude（`& $claudeExe $promptText`），仍是交互 TUI（**非 -p**，符合 LESSONS_LEARNED §1），无需手动 Ctrl+V；剪贴板保留为静默兜底。③ 启动 claude 前临时 `$ErrorActionPreference='Stop'`，让失败成为可捕获终止错误并写日志。④ spawn env 诊断（`spawn env PATH=...` + `resolved claudeExe=...`）写入 stdout.log。
- 中文路径编码（第 5 次）：根因——Go `os.WriteFile` 写 wrapper.ps1 是 UTF-8 无 BOM，PS 5.1 无 BOM 按 GBK 解码 → 中文项目路径（`金融投资`→`閲戣瀺鎶曡祫`）乱码，Set-Location / prompt.md 全失败，claude 开了但空 prompt（对照实验实锤：无 BOM 失败 / 有 BOM 成功）。修法：① 写 wrapper.ps1 前置 UTF-8 BOM（`[]byte("\ufeff"+psCmd)`）；② `Get-Content prompt.md` 加 `-Encoding UTF8`（prompt.md 也 UTF-8 无 BOM，否则内容中文乱）。详见 LESSONS_LEARNED §12。
- prompt 内容截断（第 6 次）：根因——PowerShell 5.1 legacy native-arg passing 把含 ASCII 双引号（U+0022）的字符串劈成多个 argv 元素；prompt.md 含 62 个 `"`，被劈成 51 段，claude 只取 argv[1] = 前 642 字符，输出契约 / 行为规则 / 展开 primitive 全丢（对照实验实锤：原文 ARGC=51 / 把 `"` → 中文 `“`（U+201C）后 ARGC=1、ARG0_LEN=4422 = SOURCE_LEN 完整）。修法：wrapper 在传给 claude 前一行 `$promptText = $promptText -replace [char]34, [char]0x201C`；反引号（code span）**不是** breaker，只有 U+0022 是。详见 LESSONS_LEARNED §13。

验证步骤：

```text
1. 重启 Go 后端（旧窗口 Ctrl+C → go run ./backend-go/cmd/lll）
2. 浏览器 Ctrl+F5 强刷，进入任一项目
3. 选 Intro zone，调用 Intro Agent
4. 期望：桌面弹出新 PowerShell 窗口 → 青色 banner → "初始 prompt 会自动带入"
   → "正在启动 Claude TUI..." → Claude TUI 启动，且首条消息（prompt）已自动带入，无需手动粘贴
```

失败兜底（按症状对症诊断）：

| 症状 | 可能原因 | 处理 |
|------|---------|------|
| 没弹窗 | spawn 失败 | 看 `runs/<ts>-<agent>/stderr.log`，应有 "spawn wrapper: ..." 行 |
| 窗口停在 banner，没进 claude | claude 解析不到（子进程 env 与后端不同） | 看 `runs/<ts>-<agent>/stdout.log`：`resolved claudeExe=` 是否为空；空则设 `CLAUDE_BIN` 全路径重启 |
| 窗口闪退 | powershell.exe 不在 PATH | 终端跑 `where.exe powershell` 验证 |
| 红字"⚠️ 无法读取 prompt.md" | runDir 创建失败 / 权限 | 看 stderr.log 详细堆栈 |
| 红字"⚠️ Claude 启动失败" | claude 启动即退（env/权限） | `where.exe claude` 验证；`stdout.log` 有 `claude exec failed: ...` 详细错误 |
| 红字"Set-Clipboard failed" | PowerShell restricted 模式 | 检查组策略 / ExecutionPolicy |
| UI 调用按钮报 500 | 后端日志 | 看 `go run` 输出 + run.json metadata |
| Agent 开了但 prompt 截断 / 只产出一半（第 6 刀已修） | PS argv 把含 ASCII 双引号的 prompt 劈成多参 | 让 agent 回显收到的 prompt，看是否止于"必选 Primitives"；runDir/wrapper.ps1 应含 `-replace [char]34, [char]0x201C` 行 |

> 日志落点：wrapper 侧诊断（PATH、claudeExe 解析、claude exec 错误）写在 `stdout.log`；Go 侧 spawn 失败写在 `stderr.log`。

关闭条件：① 用户报告"弹窗 + TUI 启动 + prompt 完整自动带入 + 能对话"；② Intro Agent 实际产出完整引入文档（已有知识 / 为什么学 / 入口问题 / 边界 四段），证明 prompt 全文到位（非前 642 字符截断）。**✅ 两项均已达成（2026-06-10 用户确认跑通）。**

不允许的做法：
- 没有错误证据就"再修一刀预防性优化"。
- 改回 `cmd /c start` 任何变体（arg parsing 陷阱）。
- 引入 `claude -p` 任何变体（失去 TUI）。

### C-02 / C-03 / C-04（iter-02 末期已修，iter-03 入口回归验证）

```text
C-02 前端右侧大片空白 — 已修（移除 OutputViewer max-width + invokeCol 改底部 sticky bar）
     关闭条件：iter-03 首次回归测试确认布局正常。**✅ 已达成（2026-06-11 用户回归确认）。**

C-03 latex 数学公式渲染 — 已修（手动 pre-extract + placeholder + KaTeX 直接 render）
     关闭条件：含 \Sigma / \boldsymbol / 多行 display math 文档正常渲染。**✅ 已达成（2026-06-11 用户回归确认）。**

C-04 全局斜体禁用 — 已修（globals.css 全局 em/i/h1-h6 font-style normal !important）
     关闭条件：除 topbar logo 外无斜体。**✅ 已达成（2026-06-11 用户回归确认）。**
```

## Boundary vs Iteration 02

Iteration 02 delivered:

```text
project + subproject + five zone folders
Explain Agent only
real Claude Code terminal launch
append-only session + turn model
runs/ raw trace and explain/ curated output separation
project-memory initial files
```

Iteration 03 adds:

```text
the other four agents (Intro/Practice/Extend/Summary) with full charters + primitives
closed-loop interaction on every zone (mark/ask/submit/evaluate/review)
Extend as guided thinking (four relation lenses), not graph drawing (D-Q-07)
learning trace schema frozen (event-writing impl deferred)
practice evaluation file-first persistence (DB index deferred)
summary flashcard subsystem
Markdown editor live-render upgrade (HIGH PRIORITY foundational surface)
```

Iteration 03 must not:

```text
rewrite the project / session / turn backbone
change zone ids (display names may change; ids are frozen)
abandon file-as-fact persistence
silently expand scope into cloud or auth
```

## Success Shape

By the end of this iteration, a learner should be able to:

```text
create a project with 4 contract fields (ability / target / difficulty / deliverables)
open Intro zone, invoke Intro Agent, get a hook + history + entry questions
open Explain zone, mark multiple confusing passages, hit "ask selected" once
open Practice zone, choose 5 questions, answer all, self-assess, submit, receive AI evaluation
open Extend zone, see four Mermaid relation graphs and edit relation notes
open Summary zone, switch between flashcards page and report page (never split)
review a flashcard, grade it (forgot / fuzzy / got it / too easy), see next card
accept or reject AI suggestions for summary.md
reopen the project and see all of the above persisted on disk
inspect any learning trace event and trace it back to source artifacts
write practice answers / summaries / notes in an Obsidian-style live-render editor
```

A maintainer should be able to:

```text
read this README's Key Decisions section and know why each 4+4 拍板 was made
read this README's Carryover section and verify the launcher end-to-end tested
read USER_STORIES.md and find every 认同项 from the alignment package
rebuild the practice evaluation index from practice/evaluations/*.json files
```
