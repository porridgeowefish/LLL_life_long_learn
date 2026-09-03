# Iteration 05 对齐文档

状态：active
负责人：project maintainer
最后审阅：2026-07-05
事实源：本文是 Iteration 05 实现前的可编辑对齐文档。

## 如何编辑

直接修改这份 Markdown 文件即可。每个条目使用 `Decision` 字段表达当前结论：

```text
accepted  = 认同
revise    = 需修改
rejected  = 不采用
pending   = 待决定
```

如果某项是 `revise`，请在 `用户反馈` 中写清楚希望怎么改。

## AI 交接规则

用户修改本文后，AI 必须先读取这份 Markdown，再更新迭代文档和代码。
Iteration 05 不再使用 HTML 对齐文件作为交接事实源。

## 价值取向

Decision: accepted

结论：

```text
Iteration 05 鼓励用户质疑，并把 AI 答案视为可改进的草稿。
追问对话应该持续优化内容和结构，而不是把第一版答案当成最终权威。
```

用户反馈：

```text

```

## 终端追问界面

Decision: accepted

结论：

```text
用户直接在真实 Agent 终端里追问和交互。
本轮不新增前端追问输入框，也不新增“更新当前页 / 新建模块”的前端选择控件。
```

用户反馈：

```text

```

## 默认更新行为

Decision: accepted

结论：

```text
澄清、纠错、补例子、补边界、局部深化这类追问，默认更新相关已有页面，
而不是新建一页。
```

用户反馈：

```text

```

## 独立模块边界

Decision: accepted

结论：

```text
只有当追问引入独立概念、方法、定理、案例或知识模块，
并且值得成为导航中的一级学习页面时，才新建页面。
```

用户反馈：

```text

```

## 结构重写权限

Decision: accepted

结论：

```text
如果用户的问题暴露出更好的逻辑框架，Agent 可以重构 Explain 产物：
改页面标题、重排页面顺序、拆分页面、合并页面、移动章节、新建模块页、
并重写 manifest。
```

用户反馈：

```text

```

## 保留用户思考

Decision: accepted

结论：

```text
当用户提出假设、反驳或半成型模型时，Agent 应保留其中有价值的部分，
说明它在哪里成立、边界在哪里、错误在哪里，并在有价值时整合进学习产物。
```

用户反馈：

```text

```

## 安全与可恢复

Decision: not accepted

结论：

```text
覆盖页面 Markdown 前，要保存页面快照。
进行结构级 manifest 重写前，要保存 manifest 快照。
```

用户反馈：

```text
这个不要做，目前有些浪费了。
```

## 开放问题

如果有实现前必须澄清的问题，写在这里。

```text
只需要修改intro和explain两个Agent。
```

