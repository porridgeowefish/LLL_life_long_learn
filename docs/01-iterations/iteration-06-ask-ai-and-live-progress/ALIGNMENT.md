# Iteration 06 对齐文档

状态：proposed
负责人：project maintainer
最后审阅：2026-07-06
事实源：本文是 Iteration 06 实现前的可编辑对齐文档。

## 如何编辑

直接修改这份 Markdown 文件即可。每个条目使用 `Decision` 字段表达当前结论：

```text
accepted  = 认同
revise    = 需修改
rejected  = 不采用
pending   = 待决定（多为需上机验证项）
```

如果某项是 `revise`，请在 `用户反馈` 中写清楚希望怎么改。

## AI 交接规则

用户修改本文后，AI 必须先读取这份 Markdown，再更新迭代文档和代码。

## 价值取向

Decision: accepted

结论：

```text
Iteration 06 的核心是提升"学习体验感"。
小微提问不应打断阅读；后台生成不应让用户对着静态页面干等。
```

用户反馈：可以

```text

```

## Ask-AI 独立轻量通道（不走 CLI）

Decision: accepted

结论：可以

```text
Ask-AI 是一条独立的轻量直连 HTTP 通道，使用用户自配的外源 key，
直接打 provider 流式 API，不走 claude CLI。
重型 agent 生成链路（Explain/Intro/Practice/Summary）原样不动。
这是对"模型配置由 CLI 自管"原则的有意破例，需写 ADR。
```

用户反馈：可以

```text

```

## 多厂商范围

Decision: accepted

结论：

```text
OpenAI 兼容（自定义 baseURL 覆盖 OpenAI/DeepSeek/月之暗面/通义等）
+ Anthropic 原生（一等公民 extended thinking）。
两套流式格式，后端归一化为统一帧。
```

用户反馈：

```text

```

## 流式端点：请求级、不走应用总线

Decision: accepted

结论：

```text
POST /api/projects/{id}/ask-ai/stream 直接返回 text/event-stream（请求级）。
聊天 token 不广播到全局 SSE 总线（不应发给所有客户端）。
前端用 fetch + ReadableStream 消费。
```

用户反馈：

```text

```

## 信号架构：fsnotify + hooks 混合

Decision: accepted

结论：

```text
fsnotify 管通用产物级即时刷新（运行时无关、干掉轮询）；
Claude Code hooks 管 Claude 运行时的细粒度进度与可靠完成。
非 Claude 运行时优雅退化为 fsnotify 页级进度 + 不确定进度条。
```

用户反馈：

```text

```

## 窗口形态

Decision: accepted

结论：

```text
Ask-AI 用锚定在选中文字处的浮窗（非居中 Modal），
基于已有选中坐标定位 + 视口边缘翻转。v1 不做拖拽/缩放。
```

用户反馈：

```text

```

## 入口与留存

Decision: accepted（已据用户反馈修订：由 ephemeral 改为持久化）

结论：

```text
入口："选中文字 -> 问 AI"。提问创建（或复用）一条 confusion 作为疑问载体。
Ask-AI 对话持久化在该 confusion 上（非 ephemeral），支持多轮追问。
用户关闭窗口时，后端用所配 provider 自动生成 ≤250 字中文总结；
总结输入 = 选中疑问文字 + 整段对话（结合用户疑问点），写入 confusion。
confusion 状态置为 'asked'。
侧栏（ConfusionPanel）展示疑问文字；鼠标悬停显示总结
（未生成时"生成总结中..."，失败时"总结生成失败"）。
总结完成经既有 confusion-updated SSE 事件通知前端刷新。
可从侧栏调回查看整段对话（只读；关闭后不支持继续追问）。
```

用户反馈：

```text
并非，提问后内哦让那个也要存储。而且用户可以追问AI的。用户关掉AI，后台自动生成总结，总结要结合用户的疑问点（也就是说，作为一次和AI的对话，添加到message要包含到用户的上下文），250字以内，用户可以在侧栏看到选中的疑问文字，鼠标悬空在上可以看到总结文字（如果还没有生成总结可以是生成总结中......)，可以调回查看，不支持追问。
```

## 浏览器搜索

Decision: accepted

结论：

```text
一键用系统浏览器搜索选中文字，默认 Google，设置可切 Bing。纯前端，零后端。
本迭代不做窗口内 AI 联网检索（Perplexity 式）。
```

用户反馈：

```text

```

## 交付顺序

Decision: accepted

结论：

```text
A（fsnotify 刷新）-> B（Ask-AI）-> C（hooks 细粒度进度 + 可靠完成），全做。
每片独立可验收。
```

用户反馈：

```text

```

## Windows 上 Claude Code hook 命令语法

Decision: pending

结论：

```text
hook 在 Windows 下的 shell（cmd / PowerShell）与上报命令语法需上机确认。
拟用系统自带 curl.exe（Win10+）或 PowerShell Invoke-RestMethod。
属 LESSONS_LEARNED 的 Windows+shell+UTF-8 风险区：中文 activity 文本走 stdin/文件、不走 argv。
Phase C 初实测后定稿。
```

用户反馈：

```text

```

## hook 配置注入方式

Decision: pending

结论：

```text
首选 claude --settings <run-scoped file> 指向 run 目录临时 settings（最干净）。
若该 CLI flag 不支持，退化为"一次性全局 hook 安装说明 + 文档"。
Phase C 初实测确认。
```

用户反馈：

```text

```

## 原则破例 ADR

Decision: accepted

结论：

```text
在 docs/00-product-and-architecture/ 下写一份 ADR：
声明 Ask-AI 是独立轻量直连通道、重型 agent 链路仍走 CLI。
同步改 SettingsPage 现有"平台不会要求你填写模型参数"文案。
与工作区新增的 ALIGNMENT_MODE.md 口径对齐。
```

用户反馈：

```text

```

## 开放问题

如有实现前必须澄清的问题，写在这里。

```text
hook 命令语法与 --settings 注入方式（上两项 pending）需在 Phase C 上机验证。
scope 偏大：若 A+B 已占满精力，C 可拆为紧随的小迭代。
```
