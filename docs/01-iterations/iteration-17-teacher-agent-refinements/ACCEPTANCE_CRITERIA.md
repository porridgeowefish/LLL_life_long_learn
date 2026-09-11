# Iteration 17 — Acceptance Criteria

Status: planned
Owner: project maintainer
Last reviewed: 2026-09-11

## AC-1 OCR

1. Given 配置了 `askAiProviders.bindings.ocr`（指向视觉模型），when 学习者上传
   png/jpg/webp 图片并选择"保存并解析"，then 不创建 CLI 助教任务，资料状态在
   OCR 完成后变为 `ready`，`derived/content.md` 含提取文本（标题/表格结构保留、
   公式为 LaTeX）。
2. Given 未配置 ocr 绑定，when 上传图片并解析，then 走既有 CLI 助教解析路径，
   行为与迭代 16 完全一致。
3. Given OCR 调用失败（网络/配额），then 资料状态回到失败态并给出原因，原文件
   保留，可重新触发解析。
4. Given OCR 成功，when 教师引用该资料，then 注入上下文的是 OCR 文本（复用
   `<selected_source_markdown>` 链路，无需教师侧改动）。

## AC-2 队列

1. Given 教师正在生成，when 学习者发送消息，then 返回 `202 {queueId}`，消息以
   队列条目出现在输入框上方，输入框保持可用。
2. Given 队列有条目，when 学习者编辑/删除，then 对应事件落盘，刷新后队列状态
   一致。
3. Given 当前响应结束（完成/失败/停止）且队列非空，then 服务端自动按序启动
   队首的下一轮，无需前端页面在场。
4. Given 无活跃响应，when 学习者通过排队端点发送，then 立即被提升为正式回合
   （等价于直接发送）。
5. Given 服务器重启，when 恢复，then 队列条目仍在事件流中、UI 恢复显示；
   启动对账不吞队列。

## AC-3 引导

1. Given 教师正在生成且队列有条目，when 学习者对该条目点"立即引导"，then
   活跃响应被中断（已生成文本落盘为 `interrupted` 消息），该条目立即成为带
   steering 标记的 learner 消息，并立即开始新一轮。
2. Given 新一轮上下文，then 包含明确的"生成中引导"说明：上一条是中断稿，
   应吸收其有效部分并按引导方向继续。
3. Given 引导时该条目不是队首，then 其余条目保持排队顺序不变，本轮结束后
   照常自动出队。
4. Given 响应恰好已结束才点引导，then 退化为普通提升发送，不产生空回合或
   重复消息。

## AC-4 复制与导出

1. Given 任意消息，when 悬浮并点复制，then 剪贴板为其 markdown 原文。
2. Given 对话头部导出按钮，when 点击，then 下载服务端渲染的完整 `.md`
   transcript（角色、时间戳、正文、附件名、排队引导标记），不受前端分页影响。

## AC-5 重新生成

1. Given 最后一条教师回复（非生成中），when 点"重新生成"，then 旧回复与旧
   teacher 消息被 `response-superseded` 事件隐藏（历史文件仍在、可审计），
   针对同一条 learner 消息立即开始新回复。
2. Given 中间历史消息，then 不提供重新生成入口（仅最后一条）。
3. Given 旧回复被取代，when 同一 learner 消息的幂等查找执行，then 返回新
   回复而非旧回复；重放/恢复路径不复活旧回复。

## AC-6 网络搜索

1. Given `webSearch` 已配置，when 教师回合涉及需要检索的问题且模型调用
   `search_web`，then 服务端调用智谱 web-search API、把结果回灌，教师继续生成
   并在回答中给出来源。
2. Given 单个回合，then 搜索执行次数 ≤ 2。
3. Given 未配置 `webSearch`，then `search_web` 工具不注册，教师行为与现在一致。
4. Given 搜索发生，then 流式消息中出现"已搜索：<query> · N 条结果"状态条，
   正文渲染不受影响。
5. Given 搜索 API 失败，then 该工具调用以失败结果回灌（教师可告知检索不可用），
   回合不崩溃。

## AC-7 仓库整理

1. Given 任意引用旧路径的脚本/文档，when 整理完成，then `grep` 旧路径
   （`agents/`（排除 `learning-agents/`）、`scripts/run-check`、
   `scripts/check-coverage`、`scripts/qa_`、根 `folders.json`）无残留引用。
2. Given 既有本地数据，when 启动新版本，then 根目录 `folders.json` 自动迁移到
   `projects/` 下并被读取（一次性兼容读取旧位置）。
3. Given `npm run check` / `archcheck` / `go test ./...`，when 整理完成，then
   全部通过。
4. Given HTTP API，then 路径与响应形状不变（契约冻结测试通过）。
