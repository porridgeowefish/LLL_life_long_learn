# Iteration 03 Test Plan

Status: draft
Owner: project maintainer
Last reviewed: 2026-06-09
Source of truth: derived from ACCEPTANCE_CRITERIA.md（10 组验收点）+ README Carryover.
每层测试映射验收点组 `[AXX]` / 拍板 `[D-XX]` / Carryover `[C-XX]`。

> iter-02 测试基建（已核实）：后端标准 testing + withTemp* 辅助（6 包全绿，无 HTTP 集成）；
> 前端 Vitest + jsdom + @testing-library/react（14 测试全绿，MSW 未用）；
> Playwright 装了但 e2e/ 空目录无配置。iter-03 补 httptest 集成 + Playwright E2E 两层。

## 测试分层总览

| 层 | 工具 | 覆盖目标 | iter-03 状态 |
|---|------|---------|-------------|
| 后端单元 | Go testing + withTemp* | 新 store 包 + 扩展包 | 扩展 |
| 后端集成 | Go httptest（**新增**） | 端到端 API 行为 | 新增 |
| 前端单元/组件 | Vitest + testing-library | 编辑器/面板/流程组件 | 扩展 |
| E2E | Playwright（**新增配置**） | 2-3 核心学习流 | 新增 |
| Carryover 回归 | 上述各层 | C-01..04 | 新增 |

## 1. 后端单元测试（Go，withTemp* 模式）

沿用 iter-02 模式：标准 testing，无 testify，withTemp* 建临时项目结构。

### 1.1 新增 store 包

```text
confusionstore  TestConfusion_CRUD / TestConfusion_CharRangeRequired [D-Q-03]
                TestConfusion_SoftDeleteTombstone [DATA-01]
                TestConfusion_SourceRefsResolved [D-E-04]
practicestore   TestPractice_TasksGenerated / TestPractice_NoAnswersLeaked [P-02]
                TestPractice_AttemptLockAndSnapshot [P-07]
                TestPractice_EvaluationFileFirst [DATA-02]
flashcardstore  TestFlashcard_TraceableSourceRefs [S-01]
                TestFlashcard_GradeDoesNotMutateFact [S-02]
```

### 1.2 扩展现有包

```text
workspace       TestCreateProject_FourFieldContract [IN-01..04]
                TestCreateProject_DifficultyHasDescription [IN-03]
artifactwriter  TestWriteArtifact_SummaryProtected [S-05]（扩 confusion/evaluation 写）
promptassembly  TestBuild_SourceRefsInjected [D-E-04]（invoke source_refs）
```

验收映射：后端单元覆盖 IN / E（confusion）/ P（tasks+evaluation）/ S（flashcard）/ DATA 大部分字段级验收点。

## 2. 后端集成测试（Go httptest，**新增层**）

iter-02 无 HTTP 集成测试。iter-03 新增 `routes_*_test.go` 用 httptest 起真实 mux：

```text
TestInvoke_WithSourceRefs          POST /api/agents/{id}/invoke 含 sourceRefs
                                    → 断言 prompt.md 含 quote_snapshot 段 [D-E-04]
TestConfusion_CRUD_Http            GET/POST/PATCH/DELETE 全链 → 断言 confusions.json [E-02]
TestPractice_BatchOnly             断言无逐题 evaluate 路由 [D-Q-02]
                                    POST submit → POST evaluations → 读 .json+.md [P-07/08]
TestFiles_ExpandedWrite            POST /files summary/ + explain/notes.md 可写 [BTN-04]
                                    断言 summary/summary.md 仍受保护 [S-05]
TestSSE_NewEvents                  artifact-updated / evaluation-ready 推送 [§9]
```

Good Test Rule（沿用 iter-02）：断言外部行为（HTTP 状态 + 响应 + 落盘文件内容），不断言内部函数调用。

## 3. 前端单元/组件测试（Vitest）

### 3.1 组件测试（testing-library render）

```text
MarkdownEditor.test.tsx   TestLiveRender_BoldRealtime [BTN-04]
                          TestSplitPane_DraggableWidth
                          TestOuterWidth_Stays75ch
                          TestWritesBack_OriginalArtifactPath
ConfusionPanel.test.tsx   TestSelection_NoAutoInvoke [D-E-04]
                          TestUnifiedAskButton_ShowsCount [D-E-04]
                          TestFourActions_NoAITrigger
PracticeFlow.test.tsx     TestStateMachine drafting→submitted→evaluated
                          TestNoPerQuestionEvaluateButton [D-Q-02]
                          TestBatchConfirm_MissingItems [P-07]
FlashcardDeck.test.tsx    TestFaceOnlyDefault / TestFourGrade [S-02]
                          TestTabSwitch_KeepsProgress [D-Q-05]
```

### 3.2 hook / store 测试

```text
useConfusions.test.ts   增删改 + invalidate
practiceStore.test.ts   attempt 草稿/锁定状态机
flashcardStore.test.ts  进度持久化（localStorage）
```

### 3.3 lib 测试

```text
charRange.test.ts       段落级 UI 自动填 charStart/charEnd [D-Q-03]
mathExtract.test.ts     回归 latex 提取（C-03）
```

MSW：iter-03 起用 MSW mock /api，集成测试前端流程不依赖真后端（iter-02 装了没用，补上）。

## 4. E2E 测试（Playwright，**新增配置**）

iter-02 无 E2E。iter-03 新增最小配置：`frontend/playwright.config.ts` + `e2e/*.spec.ts`。

核心流 1 — 学习闭环主线：
```text
创建项目(4 字段含 difficulty) → Intro Agent → Explain 标 2 个困惑 →
"统一提问" → 进 Practice 出 5 题 → 整批作答+自评 → 整批提交 →
看 AI 评估 → Extend 引导思考（四维度追问，无图） → Summary 闪卡复习 + 切总结 tab
断言：全程无逐题按钮、Summary 不分屏、Extend 无 .mmd、磁盘落盘可复查
```

核心流 2 — 编辑器 + 笔记：
```text
进 Explain → 选段"加入笔记" → Obsidian 编辑器实时渲染 → 保存写回 notes.md
断言：**粗体** 实时渲染、宽度保持、写回原路径
```

核心流 3 — Carryover 回归：
```text
调 Intro Agent → 断言 PowerShell TUI 弹窗 [C-01]
Explain 输出 → 断言无右侧空白 [C-02] + latex 渲染 [C-03] + 无斜体 [C-04]
```

## 5. Carryover 回归矩阵

| Carryover | 测试层 | 断言 |
|---|---|---|
| C-01 launcher | E2E + 后端集成 | invoke 弹 PowerShell 窗口 + TUI 启动；stderr.log 兜底 |
| C-02 右侧空白 | E2E | Explain OutputViewer 无右侧大片空白 |
| C-03 latex | 前端 lib 单元 + E2E | \Sigma / \boldsymbol / 多行 display 正常渲染 |
| C-04 斜体 | E2E | 除 topbar logo 外无斜体 |

## 6. 测试运行命令

```text
后端单元+集成   go test ./backend-go/...
前端单元/组件   cd frontend && npm run test
前端 watch      npm run test:watch
E2E（新增）     cd frontend && npx playwright test
全量回归        go test ./backend-go/... && cd frontend && npm run test && npx playwright test
```

## 7. Good Test Rules（沿用 iter-02）

```text
测外部行为，不测内部实现细节。
文件为事实源 → 断言落盘文件内容（不只断言 HTTP 响应）。
learner-owned 文件（summary.md / project-memory.md）→ 断言不被盲覆盖。
不双轨：SSE 事件只驱动 query 失效，不流原文到 UI（LESSONS_LEARNED #3）。
```

## 验收点覆盖自检

| 验收点组 | 后端单元 | 后端集成 | 前端 | E2E |
|---|---|---|---|---|
| IN（4 字段） | ✅ workspace | ✅ | — | ✅ 流 1 |
| Intro I | — | — | — | ✅ 流 1 |
| Explain E + D-E-04 | ✅ confusionstore | ✅ | ✅ ConfusionPanel | ✅ 流 1 |
| Practice P + D-Q-02 | ✅ practicestore | ✅ BatchOnly | ✅ PracticeFlow | ✅ 流 1 |
| Extend D + D-Q-01 + D-Q-07 | — | — | — | ✅ 流 1 |
| Summary S + D-Q-05 | ✅ flashcardstore | — | ✅ FlashcardDeck | ✅ 流 1 |
| DATA | ✅ evaluation 文件优先 | ✅ Practice 评估落盘 | — | — |
| BTN-01 导航 | — | — | ✅ | ✅ |
| BTN-04 编辑器 | — | ✅ Files扩写 | ✅ MarkdownEditor | ✅ 流 2 |
| Carryover C-01..04 | — | ✅ C-01 | ✅ C-03 lib | ✅ 流 3 |

说明：Intro/Extend 以 E2E 覆盖为主（行为合同难单元化）；DATA 以单元+集成为主（无 UI）。
D-Q-03 字符级数据由 confusionstore + charRange lib 双重断言。
D-Q-07 Extend 无 .mmd，E2E 流 1 断言"无图、有四维度追问"。
DATA-03 / S-06 建议子系统 defer iter-04，本轮无 proposalstore 测试。

## 范围外

```text
不实现性能基准测试（Lighthouse 等留 iter-04）
不做覆盖率门禁（先建立测试存在，覆盖率后续提）
不 mock Claude CLI 真实输出（launcher 是真 TUI，E2E 只断言窗口+落盘）
```
