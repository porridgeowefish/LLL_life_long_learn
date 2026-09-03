# 渐进式讲解生成（边读边出）设计

- 状态：approved（设计已口头确认，待 spec review）
- 负责人：project maintainer
- 日期：2026-06-17
- 作用域：explain 链路（多页讲解 → 主题信息图）
- 关联实现：信息图管线已落地（`backend-go/internal/server/routes_explain.go`，未提交）；本文档管的是"生成顺序 / 渐进观测"的设计，不是信息图本身。

## 1. 背景与目标

LLL 的 agent 运行时就是 **Claude Code CLI**（`claudelauncher.Launch` 起交互式 TUI；`LaunchHeadless` 起无头单发）。agent 产出文件就是它的常规工具调用。

目标：让一个 agent **产出多个文件时，"生成"与用户的"观测/阅读"并行**——排在前面、可读性高的产物先就绪，用户边读，agent 终端在后面继续输出后面的产物。顺序是**已知、可规定**的，不是不可预测的。

具体落到 explain 链路：
- 多页讲解由 agent 终端逐页产出（文本）。
- 主题信息图由独立图像管线（`gpt-image-2`）产出（位图）。
- 二者都要"用户读着前面的，后面的在后台冒出来"。

## 2. 核心洞察：顺序是"提示词/章程"杠杆，不是后端杠杆

因为 agent 就是 Claude Code，**生成顺序与渐进性主要靠改 charter（提示词约束）实现，不需要 fsnotify / 轮询桥 / SSE 这类后端机制作为前提**。后端机制（SSE 等）只负责把"观测"从轮询延迟降到接近 0，是可选的抛光层，与"生成顺序"正交。

## 3. 现状：为什么今天不是边读边出

`agents/charters/explain.md` 第 82 行明确规定了相反的顺序：

> 先写全部页面，再原子更新 manifest，避免清单指向不存在的文件。

这条规则导致所有页一次性出现：manifest 最后才落盘，前端 `useExplainManifest`（`learningArtifacts.ts:65`，`refetchInterval: 5000`）轮到 manifest 后一次性渲染全部页。用户在 manifest 落盘前看不到任何页。

注意：同文件第 87–93 行的**追问协议已经用了增量模式**（"将新页面追加到 manifest"），证明章程层面早已支持"逐文件、自洽地更新 manifest"。Stage 1 只是把主生成路径也切到这个已被验证安全的模式。

## 4. 设计

### Stage 1（核心，改提示词；零后端改动）— 页面级边读边出

把 `agents/charters/explain.md` 的生成规则从"全部写完再更 manifest"改为**逐页增量**：

- 写完一页文件后，**立即重写 manifest，使其只包含此刻已落盘的页面**。
- 每次 manifest 落盘都自洽：清单里任何一个 `pages[].file` 都必须已存在于磁盘。
- 不停顿：写完第 1 页并更新 manifest 后，继续写第 2 页，再更新 manifest，依此类推。
- 顺序沿用现有 `001-overview → 002-prerequisites → 003-core-concepts …`，第一页本身就是最可读的"研究地图"，天然满足"可读性高的先就绪"。

效果链路：

```
agent 写 001 → 重写 manifest(只含 001) → 前端 5s 内渲染 001 → 用户开始读
        → 同时 agent 写 002 → 重写 manifest(001+002) → 前端 TOC 多出 002
        → … 用户读前面的，agent 终端继续输出后面
```

前端 ExplainReader 在 manifest 从 N 页长到 N+1 页时**保留当前 `pageID`**（仅当当前页 id 不再存在才重置，见 `ExplainReader.tsx:32-37`），所以用户读第 1 页不被打断，新页静默出现在目录里。

> 一致性等价证明：现有"写完再更"保证 manifest 永不指向缺失文件；"写完一页才把它加入 manifest"是同等级保证——任何时刻 manifest 只列已落盘文件。安全性不降级。

### Stage 2（抛光，SSE）— 信息图即时投递

信息图**不是 agent 终端能直接画的**（agent 是文本智能体，图像来自 `gpt-image-2` API）。所以信息图走独立后端管线（`routes_explain.go` 的 `runInfographicPipeline`：Stage A headless clafter 打磨提示词 → Stage B `scripts/gen_infographic.py` 生成 PNG），**排在所有页之后**触发，通过 SSE 让它在用户读页时"冒出来"：

- 后端：在 `runInfographicPipeline` 落盘 `complete` 状态后，`broadcaster.Emit("artifact-updated", {slug, artifact:"explain/infographic"})`。
- 前端：`ExplainInfographic` 订阅 `artifactUpdated`（SSE fanout 已包含此事件名，`useSSE` 单点挂载在 AppShell）；收到事件立即 `invalidate` 信息图 query，把 `<img>` 换上去。
- 兜底：保留现有 3s 轮询（`explainInfographic.ts` 的 `refetchInterval`），SSE 断连/丢失时仍能在数秒内收敛到最终图。

> 后端目前**从不** emit `artifact-updated`（readiness 全靠轮询）。Stage 2 只是在信息图这条链路上补发这一个事件。

### Stage 3（可选，未来）— 页面级 ~0s 即时

Stage 1 用现有 5s 轮询已实现页面级边读边出，但有最多 5s 延迟。若要页面也接近即时，再加一个服务端对 `explain/` 的轻量监听：poll-bridge（无新依赖）或 fsnotify（事件驱动，Windows 边角需处理），把每页落盘也推成 `artifact-updated`。**仅当 5s 延迟不可接受时才做。**

## 5. 诚实区分：两类产物的来源

| 产物 | 产生者 | 顺序/渐进机制 | 投递机制 |
|---|---|---|---|
| 多页讲解 | **agent 终端**（Claude Code TUI 写文件） | Stage 1：charter 逐页增量约束 | 现有 manifest 5s 轮询（Stage 3 可升级为 SSE） |
| 主题信息图 | **图像 API 管线**（后端，非 agent 终端） | 排在所有页之后触发（manifest 完成即所有页就绪） | Stage 2：SSE `artifact-updated` + 轮询兜底 |

要点：信息图无法由 agent 终端"边输出"（它是位图 API）。它靠"排在页之后 + SSE 投递"实现"你读着页、图在后台冒出来"的同等体感，但来源是后端管线。

> 进阶选项（先不做，YAGNI）：给 explain agent 挂一个"请求信息图"的工具（MCP / hook），让整条 explain→信息图 都由 agent 终端按既定顺序编排。

## 6. 数据流（目标态）

```
用户打开 explain
  └─ useExplainManifest 5s 轮询
        Stage1: agent 逐页写文件 + 逐次重写 manifest（自洽）
          → manifest 增长 → 前端逐页渲染（用户读第1页，agent 写第2页）
        manifest 完成（所有页就绪）
          → ExplainInfographic 自动 POST /explain/infographic（已实现）
                StageA: headless crafter 打磨提示词
                StageB: python 生成 PNG（原子写）
          → 后端 Emit("artifact-updated")
                Stage2: 前端订阅 → 即时换 <img>（轮询兜底）
```

## 7. 风险与缓解

1. **agent 是否严格遵守"逐页增量"顺序**：Claude Code 一般照办，但偶发批处理/乱序。缓解：charter 写到极明确；可在每次重写 manifest 后让 agent 自检"清单列出的是否都已存在"。
2. **manifest 一致性**：靠"写完一页才加入清单"的硬规则兜住，与现有"写完再更"同等安全。
3. **追问与新页交错**：追问协议本就是增量追加（第 87–93 行），与 Stage 1 的增量主路径一致，无冲突。
4. **信息图触发时机**：manifest 完成即所有页就绪，"页→图"顺序天然成立，不会读到半截主题就出图。
5. **SSE 丢失/断连**：保留 3s 轮询兜底，最坏几秒内收敛。
6. **agent 终端可见性**：Stage 1 依赖 agent 跑在交互式 TUI（`Launch`）才有"终端边输出"的体感；headless 模式（`LaunchHeadless`）无可见终端，但文件仍渐进落盘，前端观测效果一致。

## 8. 验证计划

- **Stage 1**：改 charter 后，跑一次 explain agent，DevTools 观察 `useExplainManifest` 轮询——manifest 应随时间**逐页增长**（先 1 页、再 2 页…），而非一次性全量；用户能在第 1 页渲染后就开始读，同时 TOC 逐个长出新页。
- **Stage 2**：信息图生成完成后，前端应在收到 SSE 事件的~1s 内换上 `<img>`（而非等到下一次 3s 轮询）；手动 kill SSE 后，轮询兜底仍能在数秒内换图。
- **回归**：现有"一次写完"的安全保证不降级——任意时刻抓取 manifest，清单中每个 `file` 都能在 `pages/` 找到对应文件。

## 9. 不在范围（本轮先不做）

- Stage 3（服务端 `explain/` 监听 / 页面级 SSE）。
- 给 agent 挂信息图工具、让 explain→信息图 全由 agent 终端编排。
- 把 Stage 1 的增量模式推广到其它 zone（仅 explain 链路）。
- 信息图管线本身的实现（已落地，待提交）。
