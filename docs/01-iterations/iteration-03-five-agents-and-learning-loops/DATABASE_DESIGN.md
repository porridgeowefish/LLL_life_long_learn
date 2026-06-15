# Iteration 03 Database Design

Status: draft
Owner: project maintainer
Last reviewed: 2026-06-09
Source of truth: derived from ACCEPTANCE_CRITERIA.md + README Key Decisions + DATA-01..03.
派生自验收点与拍板，每个 schema 标注关联端点 / 验收点 `[AXX]` / 拍板 `[D-XX]`。

> 文件为事实源（D-Q-04）。本文件定义 iter-03 新增的文件 schema 与持久化边界（iter-03 不引入 DB 索引）。
> 现有结构（iter-02）保持不变，仅标注 iter-03 新增项 `[NEW]`。

## 1. 持久化原则 `[D-Q-04]`

```text
文件系统是唯一事实源（source of truth）。
DB（若引入）仅作加速索引层，绝不成为事实源，可随时从文件重建。
所有 /api mutation 在后端内部先写文件，文件写成功才返回 / 才标 DB 索引完成。
```

## 2. 项目结构树（iter-02 不变 + iter-03 新增）

```text
projects/<slug>/
├── project.md                       项目主页（人类可读）[iter-02]
├── state.json                       项目状态（事实源）[iter-02，iter-03 扩字段]
│   └── + active_zone / requested_difficulty / deliverables[] / completion_contract
├── memory/
│   ├── project-memory.md            项目记忆（learner-owned，iter-03 只读）[iter-02]
│   └── project-state.json           记忆快照 [iter-02]
├── intro/
│   ├── brief.md / output.md / notes.md   [iter-02]
├── explain/
│   ├── brief.md / output.md / notes.md   [iter-02]
│   ├── confusions.json              [NEW] 困惑清单（事实源）[E-02 / D-E-04 / D-Q-03]
│   └── versions/                    [NEW] 讲解版本（重新调用保留）[E-01]
├── practice/
│   ├── tasks.json                   [NEW] 题目（学习者可见，无答案）[P-02]
│   ├── submissions/                 [NEW] 作答 [P-05]
│   │   └── <attempt>/
│   │       ├── <qid>.md             每题草稿+提交快照
│   │       ├── draft-state.json
│   │       └── submitted-snapshot.json   提交锁定快照 [P-07]
│   └── evaluations/                 [NEW] 评估（事实源）[P-08 / DATA-02]
│       ├── <attempt>.json           结构化评估
│       └── <attempt>.md             人类可读副本
├── extend/
│   ├── brief.md / prompts.md / relation-notes.md / open-questions.md  [iter-02]
│   └── output.md                    [NEW] Extend 引导性思考文本（四维度追问，无 .mmd）[D-Q-07]
├── summary/
│   ├── summary.md                   学习者总结（受保护不盲覆盖，用户完全手写）[iter-02 / S-05]
│   ├── next-steps.md                下一步建议 [iter-02]
│   ├── flashcards.json              [NEW] 闪卡数据（事实源）[S-01]
│   ├── flashcard-progress.json      [NEW] 本地复习进度 [S-02]
│   ├── review-pack.md               [NEW] 复习材料可读副本 [S-01]
│   └── deliverables-status.json     [NEW] 交付成果状态 [S-05]
├── runs/                            [iter-02]
│   ├── _index/<session-id>.json
│   └── <ts>-<agent>/
│       ├── prompt.md / stdout.log / stderr.log / result.md / run.json / package.json
│       └── transcript.ndjson        [iter-02 已有字段，iter-03 冻结 schema] [DATA-01]
├── assets/                          [iter-02]
└── subprojects/                     [iter-02]
```

## 3. Session / Turn 模型 `[iter-02 不变]`

```text
Session { ID, ProjectSlug, ZoneName, AgentID, State, RunDirRel, PromptPath,
          Turns[], CreatedAt, FinishedAt*, ExitCode*, LastMessage }
Turn    { ID, Ordinal, Type(user|assistant|system), Content, CreatedAt, RunDirRel }
State   Preparing | Launching | Running | Completed | Cancelled | Failed | AwaitingFollowup
Turns   append-only（follow-up 追加，不替换）
```

iter-03 不改 Session/Turn 结构。新增的 confusion/practice/flashcard 是独立资源，不塞进 Session。

## 4. 新增 Schema 定义

### 4.1 explain/confusions.json `[E-02]` `[D-E-04]` `[D-Q-03]`

```json
{
  "version": 1,
  "confusions": [
    {
      "id": "confusion-001",
      "sourceArtifactId": "explain/output.md",
      "sourceVersion": "v3",
      "paragraphId": "p-042",
      "charStart": 1280,
      "charEnd": 1355,
      "quoteSnapshot": "...被选中的原文片段...",
      "notes": "可选：用户补的一句原因",
      "createdAt": "2026-06-09T10:00:00Z",
      "state": "open",
      "askedAt": null,
      "resolvedAt": null
    }
  ]
}
```

字段约束 `[D-Q-03]`：charStart/charEnd/quoteSnapshot 必填（数据从第一天字符级）；
state 枚举 open | asked | resolved | deleted（deleted 为墓碑，DATA-01 软删除）。

### 4.2 practice/tasks.json `[P-01]` `[P-02]`

JSON 题库（学习者可见，无答案）。Practice Agent 按 charter 输出契约写：

```json
{
  "tasks": [
    { "id": "q1", "type": "essay", "question": "迁移题：场景+约束+自检" }
  ],
  "generatedAt": "2026-06-09T10:00:00Z"
}
```

- `type`：`short-answer` | `essay` | `code`
- 题量 1-10，默认 5（P-01）；不含答案/解析/评分（P-02）。
- `[P-03 deferred]` question → objective / confusion / source 映射待 objective 体系（讲解产出 objective id）落地后补；当前 sourceRefs 注入已覆盖"针对困惑出题"。

答案/解析存评审 artifact（学习者不可见）：`runs/<session>/private-rubric.md`。

### 4.3 practice/evaluations/<attempt>.json `[P-08]` `[DATA-02]`

```json
{
  "attempt": "attempt-2",
  "submittedAt": "2026-06-09T11:00:00Z",
  "contentHash": "sha256:...",
  "questions": [
    {
      "questionId": "q1",
      "correctness": "partial",
      "reasoningQuality": "solid",
      "transferAbility": "weak",
      "evidence": "用户答案第 2 步...",
      "note": "..."
    }
  ],
  "overall": {
    "profile": "...",
    "selfAssessmentBias": "高估 1 档",
    "nextRoundAdvice": "..."
  }
}
```

contentHash 锁定 attempt 提交快照（防漂移/防篡改；iter-03 不做 DB 索引，见 §5）。

### 4.4 summary/flashcards.json `[S-01]`

```json
{
  "version": 1,
  "cards": [
    {
      "id": "fc-001",
      "front": "SVD 的几何含义？",
      "back": "...",
      "sourceRefs": ["explain/output.md#svd", "practice/evaluations/attempt-1.json#q3"],
      "generatedReason": "练习 attempt-1 q3 答错",
      "category": "concept"
    }
  ]
}
```

每卡可追溯 sourceRefs。flashcard-progress.json 存复习状态（grade/上次复习/下次到期），
**不静默改写 flashcards.json 的知识事实** `[S-02]`。

### 4.5 runs/<session>/transcript.ndjson — trace schema 冻结 `[DATA-01]`

每行一个事件（NDJSON）。iter-03 冻结字段，部分实现（runs transcript 优先）：

```json
{ "id":"evt-001", "projectId":"...", "zone":"Explain", "type":"confusion-marked",
  "actor":"learner", "createdAt":"...", "sourceRefs":["confusion-001"],
  "artifactVersion":"explain/output.md@v3" }
```

字段冻结：id / projectId / zone / type / actor / createdAt / sourceRefs / artifactVersion。
不建独立 trace/ 目录 `[D6]`。

## 5. 评估持久化 `[DATA-02]`（文件优先，iter-03 不引入 DB 索引）

practice 评估的持久化：文件为事实源，iter-03 **不实现 DB 索引层**（见 README Deferred）。

```text
1. AI 生成评估 → 写 practice/evaluations/<attempt>.json（事实源）
2. 写 practice/evaluations/<attempt>.md（人类可读副本）
3. 任一步失败 → 保留已写文件，返回失败让前端重试
```

冲突策略：文件胜。后续若引入 SQLite（未来迭代），DB 仅作加速索引，
可从 `evaluations/*.json` 重建，绝不成为事实源（D-Q-04）。

## 6. 保护规则 `[iter-02 不变]`

```text
summary/summary.md          永不盲覆盖（artifactwriter >200B 跳过，learner-owned）
project-memory.md           同上（learner-owned）
答案/解析 private-rubric    学习者不可见（不通过 /files 读暴露）
```

## 范围外

```text
不引入 SQLite 实现（D-Q-04，仅定义边界）
不改 Session/Turn 结构
不建独立 trace/ 目录（D6）
不实现文件版本系统（explain/versions 仅占位，E-01 重调用保留策略实现期定）
不实现 extend/*.mmd 关系图（D-Q-07，改引导性思考文本 output.md）
不实现 memory/summary proposal schema（DATA-03 / S-06 defer iter-05+）
```
