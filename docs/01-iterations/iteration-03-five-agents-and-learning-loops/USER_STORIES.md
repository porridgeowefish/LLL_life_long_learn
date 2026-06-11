# Iteration 03 User Stories

Status: draft
Owner: project maintainer
Last reviewed: 2026-06-09
Source of truth: 36 条认同项 + 1 条已拍板需修改项（US-BTN-04，高优先级）= 37 条 user stories。其中 US-S-06 / US-DATA-03 标记 **defer iter-04**（iter-03 不实现）；US-D-01..06 已按 **D-Q-07** 改为引导性思考（不画图）。G-02/G-03 为流程治理项，已移至 README "Process Constraints" 节。

每条 story 编号对齐原对齐包 ID（G/IN/I/E/P/D/S/DATA/BTN），便于回溯。

---

## 全局共识（G）

### US-G-01 (from G-01)

As a learner, I want every learning project to have a stable five-stage skeleton (Intro / Explain / Practice / Extend / Summary) so that I always know whether I am hooking interest, building understanding, drilling, mapping relations, or settling review material.

Acceptance:
- 五阶段在项目导航中固定可见。
- 跳转阶段时系统展示前置材料是否齐全。
- 每个阶段有独立输入、产出、动作和持久化合同。

Data:
- `project-state.json` `active_zone`
- 每个 zone 的 output.md / confusions / tasks / evaluations / flashcards
- 跨阶段 artifact 引用通过 `source_refs[]`

---

## 项目入口（IN）

### US-IN-01 (from IN-01) 学习起点

As a learner, I want to briefly describe my current baseline so Intro and follow-up agents start from my real starting point instead of a generic textbook.

Acceptance:
- 自然语言输入，含示例占位文本。
- 允许"完全不了解"或"已有经验但概念混乱"等非等级化表达。
- 不把勾选技术经历等同于掌握。

Data:
- `project.md` `current_ability`
- `intro/brief.md`
- 经用户确认后进 `memory/project-memory.md`

### US-IN-02 (from IN-02) 学习目标

As a learner, I want to specify target ability so the five agents collaborate toward the same end.

Acceptance:
- 目标描述支持能力动词（解释 / 比较 / 实现 / 诊断 / 迁移）。
- 系统可提示把模糊目标改写为可观察目标，但不擅自改。
- 练习评审标准和总结完成度引用该目标。

Data:
- `project.md` `target_ability`
- `state.json` `completion_contract`

### US-IN-03 (from IN-03) 学习难度

As a learner, I want to declare difficulty so content is neither boring nor suddenly over-leveled.

Acceptance:
- 提供清晰的等级说明（不只是"入门 / 常规 / 挑战 / 硬核"名字）。
- 难度影响题目梯度、术语密度、反例深度、提示强度。
- 难度不写入长期记忆为固定标签。

Data:
- `project-state.json` `requested_difficulty`
- practice generation config

### US-IN-04 (from IN-04) 学习交付成果

As a learner, I want to pre-define deliverables so learning produces verifiable outcomes instead of aimless reading.

Acceptance:
- 支持常见成果选项 + 自定义补充。
- 交付成果决定练习形式与 Summary 最终结构。
- 各阶段显示当前成果完成进度。
- Summary 必须指出成果是否完成、缺什么证据。

Data:
- `project.md` `completion_standard`
- `project-state.json` `deliverables[]`
- `summary/deliverables.md` 或 `summary.md` 对应段

### US-IN-05 (from IN-05) 高级创建选项降级

As a quick-start learner, I don't want to first understand the product taxonomy; as an advanced user, I still want templates or direct entry to a specific stage.

Acceptance:
- 默认只显示四项学习合同 + 主题。
- 高级设置可展开（分类 / 模板 / 起点阶段 / 备注）。
- 直接练习或仅总结时提示缺失前置材料影响。
- 模板只预填 brief，不改五阶段固定结构。

---

## Intro（I）

### US-I-01 (from I-01) 趣味钩子

As a learner, I want to first see why this knowledge is interesting so I have intrinsic motivation to keep exploring.

Acceptance:
- 钩子与用户目标和起点相关，不是通用鸡汤。
- 长度克制，不提前展开完整讲解。
- 至少提出一个可后续验证的悬念。
- 避免伪史料、虚构名人故事、夸大结论。

Data:
- `intro/output.md` `## 兴趣入口` 段
- `intro/trace.json` `source_questions[]`

### US-I-02 (from I-02) 历史脉络

As a learner, I want to understand from a historical problem chain why knowledge grew into its current shape, so I remember design motivations instead of isolated definitions.

Acceptance:
- 历史叙事围绕问题与解决方案，不堆年份。
- 区分确定史实、常见解释、类比性叙事。
- 必要时列出可核验来源或"需要联网核验"。
- 历史段最终回扣用户当前疑问。

Data:
- `intro/output.md` `## 历史线索` 段
- source citations（若启用检索）

### US-I-03 (from I-03) 入口问题驱动

As a learner, I want Intro to remember why I came and turn my questions into a learning route, instead of overriding them with system-preset questions.

Acceptance:
- 保留用户原始表述，同时给出结构化改写。
- 入口问题数量克制（2-4 个）。
- 每个问题标注会在哪个阶段处理或验证。
- Explain 默认继承这些问题，无需重复输入。

Data:
- `intro/output.md` `## 入口问题`
- `project-state.json` `open_questions[]`
- Explain prompt assembly 自动注入

### US-I-04 (from I-04) 先验激活

As a learner, I want a lightweight prior-knowledge probe so follow-up explanation targets my real intuition.

Acceptance:
- 先验勾选可跳过，且不冒充测评结果。
- 开放回答在用户进入 Explain 时一并提交。
- 回答只用于定制讲解，不公开贴"正确 / 错误"标签。
- 用户可编辑或删除这些输入。

---

## Explain（E）

### US-E-01 (from E-01) 自动结构化讲解

As a learner, I want to receive targeted explanation without filling extra forms, so I keep focus on understanding itself.

Acceptance:
- 自动读取项目目标、起点、难度、交付成果、Intro 入口问题。
- 输出至少覆盖第一性原理、MECE 结构、概念关系、边界、常见误解。
- 内容写 `explain/output.md`，原始运行进 `runs/`。
- 重新调用不无提示覆盖旧版本（保留版本或明确替换确认）。

Data:
- `explain/output.md`
- `runs/<session>/`
- `explain/versions/` 或 `output.v1.md` 等版本策略

### US-E-02 (from E-02) 标记困惑（字符级数据 + 段落级 UI）

As a learner, I want to select confusing original text and tag it, so AI knows exactly which sentence I am stuck on.

Acceptance:
- 选中后浮层：标困惑 / 取消 / 查看清单（**不**触发 AI 追问，见 DECISIONS D-E-04）。
- 保存 source_artifact_id / source_version / paragraph_id / char_start / char_end / quote_snapshot。
- 原文更新后通过 quote_snapshot 仍可回看；定位漂移时明确提示。
- 用户可查看、编辑、解决、删除困惑。

Data:
- `explain/confusions.json` schema 见 DECISIONS D-E-04。

### US-E-03 (from E-03) 底部统一追问

As a learner, I want to ask where I read and have all Q&A appended, forming a continuous thinking trail.

Acceptance:
- 追问追加到同一学习时间线，不覆盖前文。
- 发送前展示将注入的引用、前置文件、会话状态。
- 会话不可恢复时新建 session 挂同一 Explain 线程。
- 回复可提炼到讲解、笔记、困惑解决记录，但原始 turn 不变。

Data:
- `runs/<session>/transcript.ndjson`
- `runs/<session>/turns/*`
- `explain/confusions.json` 状态变化
- `explain/notes.md`

---

## Practice 上半场（P-01..P-04）

### US-P-01 (from P-01) 题量输入

As a learner, I want to control practice scale to match available time and energy.

Acceptance:
- 题量 1-10 整数，默认 5。
- 超过 10 不静默截断：说明原因 + 二次确认。
- 显示预计完成时间或题型分布。
- 重新生成可选：替换全部 / 补充若干 / 重做某题。

Data:
- `practice/runs/<run>/requested_count`
- `practice/tasks.md` metadata

### US-P-02 (from P-02) 无答案训练题

As a learner, I want to answer independently so practice tests my ability rather than my AI-answer-recognition ability.

Acceptance:
- 题目覆盖目标能力，难度递进。
- 每题展示作答要求和不泄题的检查标准。
- 答案与解析存评审 artifact（学习者不可见）。
- 提示分层解锁，第一层不揭示关键步骤。

Data:
- `practice/tasks.md`（学习者可见）
- `runs/<session>/private-rubric.md`（学习者不可见）
- `task.answer_visibility=false` 标记

### US-P-03 (from P-03) 题型与困惑映射

As a learner, I want practice to test where I got stuck, not random questions.

Acceptance:
- 至少部分题针对未解决困惑。
- 至少一题要求迁移到新情境。
- 系统记录 question -> objective / confusion / source 映射。
- 学习者可反馈题目歧义、超纲、重复、答案不可判定。

Data:
- `question.source_refs[]`
- `question.objective_ids[]`
- `question.confusion_ids[]`

### US-P-04 (from P-04) 生成阶段按钮

As a learner, I want to modify practice without destroying answered content.

Acceptance:
- 任何重新生成不删已提交答案。
- 跳过与不会做分开记录。
- 题目反馈不计能力评分。
- 提示使用进评估上下文，但不简单按次数扣分。

---

## Practice 下半场（P-05..P-09）

### US-P-05 (from P-05) 每题作答表单

As a learner, I want to write answers per question and save drafts anytime.

Acceptance:
- 题目切换自动保存草稿。
- 未作答 / 已作答未自评 / 已完成题状态区分。
- 提交前展示缺失项清单。
- 修改保留最后提交版本；草稿历史可配置。

Data:
- `practice/submissions/<attempt>/<question>.md`
- `practice/submissions/<attempt>/draft-state.json`
- `attachments[]`

### US-P-06 (from P-06) 每题掌握自评

As a learner, I want to judge my own mastery before AI does, to train metacognition.

Acceptance:
- 每题统一量表 1-5：完全不会 / 勉强 / 基本会 / 较熟练 / 能迁移。
- 自评发生在 AI 评价之前。
- 可选反思：卡点 / 猜测成分 / 用过的提示。
- AI 比较自评与证据，但不羞辱或机械判"过度自信"。

Data:
- `submission.self_assessment.level`
- `submission.reflection`
- `submission.hints_used[]`

### US-P-07 (from P-07) 整批提交锁定快照

As a learner, I want to know which version AI evaluated so the result is reviewable.

Acceptance:
- 提交前二次确认（题数 / 空题 / 自评缺失 / 提示使用）。
- 提交产生 attempt id + 提交时间。
- 提交后修改形成新 attempt，不覆盖旧证据。
- AI 评估失败保留快照允许重试。

Data:
- `practice/attempts/<attempt>/submitted-snapshot.json`
- `practice/attempts/<attempt>/evaluation-status.json`

### US-P-08 (from P-08) AI 证据化评估

As a learner, I want AI to tell me not just score, but where I understood vs just got lucky, and what to do next.

Acceptance:
- 逐题评价引用用户答案证据。
- 区分正确性 / 推理质量 / 迁移能力 / 表达完整度。
- 总体能力画像 + 自评偏差 + 下一轮建议。
- 结构化评估落库 + 人类可读文档。
- 练习结果可被 Summary 读取；写入长期记忆需用户确认或遵循已确认策略。

Data:
- `practice/evaluations/<attempt>.json`（事实源）
- `practice/evaluations/<attempt>.md`（可读副本）
- `project-state.json` `practice_summary`
- `database/index` 加速层（非事实主源）

### US-P-09 (from P-09) 作答页按钮

As a learner, I want safe editing before submit and clear status after.

Acceptance:
- 保存草稿与提交评估是两个不同动作。
- 编辑 / 预览切换不改数据。
- 格式工具只修改选区或插入标记，支持撤销。
- 提交按钮请求中禁用并展示进度，避免重复 attempt。

---

## Extend / 拓展（D）

> **D-Q-07**：Extend 不画 Mermaid 关系图。改为用四种关系维度（因果 / 结构 / 重要性 / 同构）
> 作为**引导性思考的脚手架**——agent 在 `extend/output.md` 按四维度向学习者追问，学习者
> follow-up 回应。原 D-02..D-05 的 rigor（反事实 / 边界 / 维度 / 失效点）保留为**追问质量标准**，
> 不再是图的边标签。复用 invoke + follow-up 基建，无新端点 / 新 UI / 新文件类型。

### US-D-01 (from D-01) 四关系维度引导思考

As a learner, I want to be guided to discover different connection types between knowledge points myself, so I move from "knowing points" to "understanding a domain" through my own reasoning.

Acceptance:
- Extend output.md 按因果 / 结构 / 重要性 / 同构四个维度组织追问（不画图）。
- 每个维度先抛引导性问题，不直接给结论。
- 学习者可 follow-up 回应，agent 据回答继续深挖。
- 无 `.mmd` 文件，无 Mermaid 渲染。

Data:
- `extend/output.md`（引导追问文本）
- `extend/relation-notes.md`（学习者记录自己的关系推理）

### US-D-02 (from D-02) 因果维度追问

As a learner, I want to be pushed to distinguish cause / mechanism / result, so I don't mistake correlation for causation.

Acceptance:
- 追问引导区分直接原因 / 条件 / 机制 / 中介 / 结果。
- 至少追一个反事实或失效条件。
- 提示用户检查证据强度，不直接宣布因果。

Data:
- `extend/output.md` 因果维度段
- `extend/relation-notes.md`（用户因果陈述）

### US-D-03 (from D-03) 结构维度追问

As a learner, I want to be guided to map composition / hierarchy / boundaries, so I build a navigable structure.

Acceptance:
- 追问引导区分组成 / 包含 / 依赖 / 接口 / 边界。
- 支持从整体展开子结构，也支持从部件回到整体。
- 提醒别把时间顺序或因果关系误当结构。

Data:
- `extend/output.md` 结构维度段

### US-D-04 (from D-04) 重要性维度追问

As a learner, I want to understand importance differences so limited time goes to high-value parts.

Acceptance:
- 重要性追问必须附带维度（先修性 / 频率 / 风险 / 迁移价值）。
- 区分"基础"与"常用"，避免单一总排名。
- 允许用户改维度后重新排序讨论。

Data:
- `extend/output.md` 重要性维度段

### US-D-05 (from D-05) 同构维度追问

As a learner, I want cross-domain isomorphism to build transfer ability, while not mistaking analogy for proof.

Acceptance:
- 追问引导明确源领域 / 目标领域 / 对应映射。
- 至少指出一个映射失效点。
- 区分类比启发 / 结构同构 / 表面相似。
- 鼓励用户自补一个同构例子并接受 AI 质询。

Data:
- `extend/output.md` 同构维度段

### US-D-06 (from D-06) 领域提示与批判性提问

As a learner, I want the system to push me to think myself, not serve "advanced conclusions" on a plate.

Acceptance:
- 每维度 2-4 个提示。
- 问题与用户当前目标和困惑相关。
- 默认不给终局答案，可提供逐层提示。
- 用户的关系陈述和回答可保存到 `relation-notes.md`。

---

## Summary（S）

### US-S-01 (from S-01) 从学习痕迹生成闪卡

As a learner, I want flashcards generated from what I really learned and missed.

Acceptance:
- 每张卡可追溯 source artifact 或练习证据。
- 覆盖概念回忆 / 辨析 / 因果 / 结构 / 迁移 / 易错点。
- 答案不过长，一卡一知识点。
- AI 标注生成原因；用户可删除或改写。

Data:
- `summary/flashcards.json`
- `summary/review-pack.md`
- 每卡 `source_refs[]`

### US-S-02 (from S-02) 闪卡交互

As a learner, I want recall-first review, not passive browsing.

Acceptance:
- 默认只显示正面问题。
- 翻面前可输入或口述答案（口述后续确认）。
- 翻面后评价"忘记 / 模糊 / 掌握 / 太简单"。
- 评价更新本地复习计划，不静默改写知识事实。
- 支持上一张 / 下一张 / 随机 / 只看错卡 / 重置。

Data:
- `summary/flashcard-progress.json`（本地复习进度）

### US-S-05 (from S-05) 知识总结栏目

As a learner, I want a继续 editable summary so learning settles into my knowledge asset.

Acceptance:
- 总结引用练习证据和用户笔记，不只摘要 AI 讲解。
- 明确"已掌握"的证据门槛与不确定性。
- 交付成果有完成 / 部分完成 / 未完成状态和证据链接。
- `summary/summary.md` 始终由用户最终控制。

Data:
- `summary/summary.md`
- `summary/next-steps.md`
- `summary/deliverables-status.json`

### US-S-06 (from S-06) AI 建议必须审阅 [DEFERRED iter-04]

> **本轮 defer iter-04**（见 README Deferred）：总结 AI 建议子系统（accept/rewrite/reject）
> 不进 iter-03。iter-03 的 `summary.md` 由用户完全手写。本 story 保留供 iter-04 接续。

As a learner, I want AI to help organize but not ghost-write, so the final knowledge stays mine.

Acceptance:
- 建议显示目标位置 / 变更内容 / 理由 / 来源。
- 接受后才写编辑稿。
- 改写进入用户编辑区，不自动提交。
- 拒绝保留审计但不再反复弹同一建议。
- 保存到磁盘含成功 / 冲突 / 失败提示。

Data:
- `summary/ai-proposals/<id>.json`
- `summary/summary.md`（用户最终版）

---

## 数据层（DATA）

### US-DATA-01 (from DATA-01) 统一学习痕迹事件模型

As the system, I need traceable connection between a learning action and its context so Summary and follow-up agents don't guess.

Acceptance:
- 事件至少含 id / project_id / zone / type / actor / created_at / source_refs / artifact_version。
- 用户删除内容定义软删除或墓碑事件。
- AI 推断与用户原始输入分开存。
- 事件模型不要求 iter-03 立即全部实现，但 schema 必须冻结。

Data:
- `runs/<session>/transcript.ndjson`
- zone artifacts
- `trace-index.jsonl` 或后续 DB 索引

### US-DATA-02 (from DATA-02) 练习评估文件持久化

As a learner and maintainer, I want evaluation to persist long-term in files AND be fast-queryable.

Acceptance:
- 评估写 `evaluations/<attempt>.json`（事实源）+ `.md`（可读副本）两文件。
- 文件为事实源；DB 索引层 iter-03 不实现（见 README Deferred）。
- 写入失败明确展示并允许重试。

Data:
- `practice/evaluations/<attempt>.json`（事实源）
- `practice/evaluations/<attempt>.md`（可读副本）

### US-DATA-03 (from DATA-03) 记忆更新可见可审阅 [DEFERRED iter-04]

> **本轮 defer iter-04**（见 README Deferred）：记忆建议子系统（accept/rewrite/reject/undo）
> 不进 iter-03。iter-03 项目记忆只读。本 story 保留供 iter-04 接续。

As a learner, I want the system to remember useful patterns, but I can see and correct its judgment.

Acceptance:
- 区分项目记忆与全局学习者记忆。
- 每条建议显示依据和影响哪些 Agent。
- 用户可接受 / 改写 / 拒绝 / 撤销。
- "答错一次"不能直接形成稳定能力结论。

Data:
- `memory/proposals/<id>.json`
- `memory/project-memory.md`
- `memory/learner-profile.md`

---

## 按钮合同（BTN）

### US-BTN-01 (from BTN-01) 全局导航与项目入口

As a user, I want navigation actions to never miswrite data and to reflect current project and stage.

Acceptance:
- 当前路由和选中态一致。
- 存在未保存编辑时提示或自动保存。
- 缺失路由不显示为可用链接。
- 项目树"其他项目"必须进入对应项目，不统一跳总览。

Data:
- router state + project tree state

### US-BTN-04 (from BTN-04) Markdown 编辑器 Obsidian 风格 [HIGH PRIORITY]

As a learner, I want a live-render markdown editor (Obsidian-style) so that writing notes, answers, and summaries feels immediate and the page width stays readable.

Acceptance:
- 编辑同时渲染（live preview）：输入 `**粗体**` 实时显示粗体。
- 备选双栏模式：两栏同步、可拖拽调宽，外层容器保持学习页宽度（约 75ch）。
- 工具栏对选区操作；无选区时插入可撤销标记。
- 编辑/预览共享同一内容源。
- 自动保存、手动保存、未提交状态视觉区分。
- 链接与附件进行路径/协议安全校验。
- Mermaid 预览错误不阻止保存源码。
- 编辑器写回原 artifact 路径，不引入新文件。

Data:
- 写回原 artifact（如 `summary/summary.md`、`practice/submissions/<attempt>/*.md`、`explain/notes.md`）
- localStorage 草稿（key = project_id + 文件路径，保存后清除）

> 优先级 HIGH：编辑器是笔记 / 作答 / 总结 / 困惑草稿的统一交互面，是地基。详见 README D-BTN-04。

---

## 范围外（明确不进 iter-03）

- 字符级 UI 高亮：iter-04 或更晚
- 全文 / 跨项目搜索：iter-04 或更晚
- 移动端 / 暗色模式 / i18n / 离线 PWA：暂不规划
