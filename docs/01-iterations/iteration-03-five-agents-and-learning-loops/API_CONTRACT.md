# Iteration 03 API Contract

Status: draft
Owner: project maintainer
Last reviewed: 2026-06-09
Source of truth: derived from ACCEPTANCE_CRITERIA.md + README Key Decisions.
派生自验收点与拍板，每端点标注关联验收点 `[AXX]` / 拍板 `[D-XX]`。

> 本文件定义 iter-03 的接口合同边界。iter-02 已有端点标 `[iter-02 不变]`，
> 不重复定义完整 shape，只列扩展点。iter-03 新增端点给出完整请求/响应 shape。
>
> 字段名以 `backend-go/internal/server/routes_*.go` 真实 struct 为准（已核实），
> 不是猜测。实现时若 struct 字段与本合同冲突，以本合同为目标态，实现负责迁移。

## 设计原则

```text
D1  沿用 iter-02 混合模式：/api 管资源生命周期，/files 管文件直读直写
D2  每个 /api mutation 后端先写文件再返回（文件为事实源）
D3  invoke 扩展 source_refs[]（不新增端点）支持统一提问
D4  confusion/practice/flashcard 用新 /api 端点；笔记/总结编辑扩 /files
    所有 /api mutation 在后端内部落到文件（practice/evaluations/*.json 等）
```

## 1. Health — `[iter-02 不变]`

`GET /api/health` — 返回 ok / workspace / claude.bin / claude.available / stats。
iter-03 不改 shape，stats 可选新增 `confusions / attempts / flashcards` 计数。

## 2. Projects

### POST /api/projects — 扩展为 4 字段学习合同 `[IN-01..04]`

iter-02 现有字段（已核实 struct）：

```json
{
  "title": "Recommender Systems",
  "slug": "recommender-systems",
  "parentProjectId": null,
  "why": null,
  "current": null,
  "target": null,
  "standard": null
}
```

iter-03 目标态（4 字段合同，对齐 IN-01..04）：

```json
{
  "title": "Recommender Systems",
  "slug": "recommender-systems",
  "parentProjectId": null,
  "currentAbility": "...",      // IN-01 学习起点（现有 current，语义对齐）
  "targetAbility": "...",       // IN-02 学习目标（现有 target，语义对齐）
  "difficulty": "regular",      // IN-03 学习难度 [NEW 字段]
  "deliverables": ["..."],      // IN-04 交付成果（现有 standard 升级为数组）
  "advanced": {                 // IN-05 高级选项，默认折叠 [NEW]
    "category": null,
    "template": null,
    "startZone": null,
    "notes": null
  }
}
```

迁移注记（实现负责，非本合同范围）：
- `current` → `currentAbility`（或保留 current，加 alias）
- `target` → `targetAbility`
- `standard` → `deliverables`（string 升级为 string[]，旧值包成单元素数组）
- `difficulty` 为全新字段，枚举含等级说明（不止名字）
- `why` 保留为高级备注或并入 advanced.notes

验收 `[IN]`：
- difficulty 选项附带等级说明文本
- deliverables 支持自定义补充
- "完全不了解"等非等级化表达可用
- 高级选项默认折叠不挡主路径
- 4 字段在各阶段可追溯读取

### 其他 Projects 端点 — `[iter-02 不变]`

```text
GET    /api/projects              列表（扁平树）
GET    /api/projects/{id}         单项目（含 zones / 文件路径 / state）
GET    /api/projects/{id}/tree    侧栏树
GET    /api/projects/{id}/zones/{zone}   单 zone（含 predecessors）
POST   /api/projects/{id}/subprojects    建子项目
```

iter-03 `GET /api/projects/{id}` 响应可扩展返回 `currentAbility/targetAbility/difficulty/deliverables`，供各 agent prompt assembly 读取。

## 3. Agents

### GET /api/agents — `[iter-02 不变]`，iter-03 全量

iter-02 仅生产 Explain；iter-03 返回全部五个 agent（Intro/Explain/Practice/Extend/Summary）。
Agent 结构（id/name/icon/description/userStory/primitives/allowedZones/defaultOutputTargets）不变。

### POST /api/agents/{id}/invoke — 扩展 source_refs `[D-E-04]` `[E-04]`

iter-02 现有请求（已核实 struct）：

```json
{ "projectId": "...", "zone": "Explain", "intent": "...", "permissionMode": "default" }
```

iter-03 扩展：

```json
{
  "projectId": "...",
  "zone": "Explain",
  "intent": "...",
  "permissionMode": "default",
  "sourceRefs": ["confusion-001", "confusion-002"]   // [NEW] D-E-04 统一提问
}
```

```text
intent 为可选字段。
前端允许直接拉起 Agent，不必强制填写补充提示词。
像 Practice 这类需要结构化补充说明的页面，仍然可以主动传 intent。
```

行为链（已核实 promptassembly.Build + claudelauncher.Launch）：
```text
1. 验证 agent / projectId / zone；intent 如有值则 trim 后注入 prompt
2. 若 sourceRefs 非空：从 explain/confusions.json 解析对应 quote_snapshot
3. promptassembly.Build 组装 prompt（注入 source_refs 引用段）
4. sessionstore.Create 建 session（StatePreparing）
5. broadcaster.Emit session-created
6. claudelauncher.Launch（PowerShell TUI 窗口）
7. 返回 { session, runDir }
```

验收 `[D-E-04]`：
- sourceRefs 非空时，prompt 含对应 quote_snapshot 引用段
- N 个困惑合并成单个 user turn（不拆成 N 次 invoke）
- 进入练习只切换页面，不自动生成题

> 说明：Intro/Explain/Practice/Extend/Summary 五 agent 的产物（intro/output.md、
> extend/output.md 引导性思考文本等，**无 .mmd**，见 D-Q-07）均经 invoke 写入对应 zone，
> 前端经 GET /files 读取，无独立 /api 端点（D4：agent 产物走 invoke+/files，不走新 /api）。
> 仅 confusion/practice/flashcard 这类需要 CRUD 生命周期的资源才有独立 /api。

## 4. Sessions — `[iter-02 不变]`

```text
GET    /api/sessions                  ?active=true | ?recent=true&limit=N
GET    /api/sessions/active
GET    /api/sessions/{id}             含 turns / runDir / state
POST   /api/sessions/{id}/follow-up   { text, permissionMode }  追加 turn
POST   /api/sessions/{id}/cancel
```

follow-up 请求（已核实 struct）：`{ "text": "...", "permissionMode": "default" }`。

## 5. Confusions — `[NEW]` `[E-02]` `[D-E-04]` `[D-Q-03]`

confusion 是 Explain 困惑标记的资源生命周期，落 `explain/confusions.json`（文件为事实源）。

```text
GET    /api/projects/{id}/confusions              列表（?state=open|resolved）
POST   /api/projects/{id}/confusions              新建（标困惑动作）
PATCH  /api/projects/{id}/confusions/{cid}        更新 state/notes/resolve
DELETE /api/projects/{id}/confusions/{cid}        软删除（墓碑，DATA-01）
```

POST 请求：

```json
{
  "sourceArtifactId": "explain/output.md",
  "sourceVersion": "v3",
  "paragraphId": "p-042",
  "charStart": 1280,
  "charEnd": 1355,
  "quoteSnapshot": "...被选中的原文片段...",
  "notes": "这里搞不懂 Sigma 为什么是奇异值"
}
```

响应：完整 confusion 对象（含生成的 id / createdAt / state="open"）。

验收 `[D-Q-03]`：charStart/charEnd/quoteSnapshot 必存（数据从第一天字符级）。
验收 `[D-E-04]`：POST 不触发 invoke；困惑进清单待统一提问。

## 6. Practice — `[NEW]` `[P-01..09]` `[D-Q-02]`

```text
POST   /api/projects/{id}/practice/tasks          生成题（题量 1-10，默认 5）
GET    /api/projects/{id}/practice/tasks          读 tasks.json
POST   /api/projects/{id}/practice/submit         整批提交（锁定 attempt 快照）
POST   /api/projects/{id}/practice/evaluations    AI 整批评估
GET    /api/projects/{id}/practice/evaluations/{attempt}   读评估
```

**无逐题端点** `[D-Q-02]`：不提供 `POST .../practice/questions/{qid}/evaluate`。
单题只支持暂存（前端 localStorage / practice/submissions/<attempt>/ 草稿）。

POST /practice/tasks 请求：

```json
{ "requestedCount": 5, "confusionIds": ["confusion-001"], "regenerate": "replace" }
```

POST /practice/submit 请求（整批）：

```json
{
  "attempt": "attempt-2",
  "answers": [
    { "questionId": "q1", "content": "...", "selfAssessment": 3, "reflection": "...", "hintsUsed": 1 }
  ]
}
```

行为：写 `practice/submissions/<attempt>/<qid>.md` + submitted-snapshot.json，attempt 锁定。

POST /practice/evaluations 请求：

```json
{ "attempt": "attempt-2" }
```

行为：AI 读 attempt 快照 → 写 `practice/evaluations/<attempt>.json` + `.md`（文件为事实源，iter-03 不引入 DB 索引）→ emit `evaluation-ready`。

验收 `[D-Q-02]`：单题无 evaluate 按钮/端点；只有整批一条路径。
验收 `[P-08]`：评估引用每题答案证据 + 含自评偏差 + 下一轮建议。

## 7. Summary Flashcards — `[NEW]` `[S-01]` `[S-02]` `[D-Q-05]`

```text
POST   /api/projects/{id}/summary/flashcards                  从学习痕迹生成闪卡
GET    /api/projects/{id}/summary/flashcards                  读 flashcards.json
POST   /api/projects/{id}/summary/flashcards/{cardId}/grade   四档复习评价
```

POST /grade 请求：

```json
{ "grade": "fuzzy" }    // forgot | fuzzy | got-it | too-easy
```

行为：写 `summary/flashcard-progress.json`（本地复习计划），不静默改写知识事实。

验收 `[D-Q-05]`：单路由 /summary；闪卡/总结是 tab 切换不是分屏；切换不丢进度。

> [S-06 Summary Proposals + DATA-03 Memory Proposals — **DEFERRED iter-04**，见 README Deferred。]
> iter-03 不实现 AI 建议 / 记忆建议子系统。`summary.md` 与 `project-memory.md` 均由用户直接编辑。
> 原 §7 Summary Proposals 子节与 §8 Memory Proposals 端点族移除，iter-04 接续。

## 8. Files — 扩展写白名单 `[BTN-04]` `[E-02]`

iter-02 现状（已核实）：读允许 memory/summary/intro/explain/practice/extend；写仅 `memory/<filename>`（正则 `^[A-Za-z0-9._\-]+$`）。
代码注释已预留"iter-03 may open summary/ writes"。

iter-03 扩展写白名单：

```text
POST /files/projects/{id}/
  memory/<filename>       [iter-02 不变]
  summary/<filename>      [NEW] Obsidian 编辑器写 summary.md（learner-owned，仍受保护）
  explain/notes.md        [NEW] 笔记草稿（"加入笔记"动作）
```

保护规则不变：`summary/summary.md` 不被盲覆盖（artifactwriter >200B 跳过）。

验收 `[BTN-04]`：编辑器写回原 artifact 路径，不引入新文件。

## 9. Events (SSE) — 扩展事件 `[iter-02 基础]`

iter-02 事件：session-created / session-state / turn-created / session-failed。

iter-03 新增：

```text
artifact-updated     zone output.md 写入完成（驱动 OutputViewer 刷新）
confusion-updated    困惑清单变化（增/删/解决）
evaluation-ready     practice 整批评估完成
```

payload 含 projectId + 资源 id，前端用 queryClient.invalidateQueries 驱动刷新（不流原文，遵守 LESSONS_LEARNED #3 不双轨）。

## 范围外

```text
不新增逐题评估端点（D-Q-02）
不新增独立 trace/ 端点（D6，trace 落 runs/transcript.ndjson）
不引入 SQLite（D-Q-04，DB 索引层仅定义边界不实现）
不改 zone id（D-Q-01，仅前端显示名改"拓展"）
不新增 memory/summary proposal 端点（DATA-03 / S-06 defer iter-04，见 README Deferred）
Extend 不产 .mmd 关系图（D-Q-07，agent 产物走 invoke+/files，引导性思考文本）
```
