# V3 Frontend Design — 智能体差异化布局 + Markdown 编辑器融合

Status: draft
Owner: Claude（前端设计 Agent）
Last reviewed: 2026-06-09
Source of truth: 本文档是 V3 设计方案的评审稿，等用户确认后再落 HTML。

## 1. 设计动机（对齐 V2 现状盘点）

V2 完成了"四页骨架 + 终端抽屉"的最小可读版，但暴露了五个颗粒度问题：

```text
① 创建项目流程零页面
② 创建智能体仅占位"+ 创建"
③ 6 个智能体共用同一个 grid-3 卡片，无差异化
④ Markdown 只能渲染，没有"写+保存+预览"
⑤ AI 产出页 max-width:860px，被卡片列表稀释了视觉权重
```

V3 的目标是把 V2 的"可读"升级为"可写可学"。

## 2. 顶层布局栅格

V3 全局采用三明治栅格，把 AI 产出主舞台放大到视觉中心：

```text
┌─ Topbar 50px ──────────────────────────────────────────────┐
├─ 240 ─┬───────────────────────────────────┬──── 320 ─────┤
│ 左侧栏 │     AI 产出主舞台（≈65%）         │  右侧上下文  │
│ 项目树 │     Markdown / Mermaid / 代码     │  前置/编辑   │
│  阶段  │     学习者就地选中→批注→写困惑    │  brief 抽屉  │
├───────┴───────────────────────────────────┴──────────────┤
│ 底部终端抽屉（默认收 40 / 展开 360）                       │
└───────────────────────────────────────────────────────────┘
```

V2 vs V3 的比例对照：

| 区域              | V2           | V3           | 增益         |
|-------------------|--------------|--------------|--------------|
| 主面板宽度        | 860px 固定   | 100% - 560px | +40~60%      |
| 主字号            | 15px         | 17px         | +13%         |
| 代码块字号        | 12.8px       | 14px         | +9%          |
| 智能体卡片        | 三栏         | 双栏         | 单卡 +50%    |
| 终端抽屉高度      | 320px        | 360px        | +12%         |

## 3. 智能体差异化布局矩阵

每个智能体拿到独立的页面布局，按其职能颗粒度匹配：

| 智能体      | 输入         | 产出                | 学习者动作          | 布局栅格             |
|-------------|--------------|---------------------|---------------------|----------------------|
| 🌱 引入     | 主题一句话   | 钩子 + 入口问题     | 浏览 → 选 1 问题    | 单栏故事流 1fr       |
| 📖 讲解     | 前序 output  | 讲解 + Mermaid 图   | 浏览 + 标注困惑     | 双栏 1.6fr / 1fr     |
| ✏️ 练习     | 讲解 + 记忆  | 任务 + 约束 + 评审  | 写答案 → 自评       | 三栏 1 / 1.4 / 1     |
| 🔗 拓展     | 全部前序     | 关系矩阵 + 节点图   | 连接 / 写关系陈述   | 全宽 Canvas          |
| 📝 总结     | 全部前序     | summary.md         | 直接编辑            | 双栏编辑器 1fr / 1fr |
| 🧠 记忆     | 全部记忆     | 记忆差异建议        | 审查 → 接受/拒绝    | 1fr / 1.4fr 差异视图 |

布局示意图见 `v3/wireframes/`。

## 4. Markdown 编辑器融合

### 4.1 五类入口共用 `<MdEditor>`

| 入口              | 模式      | 保存语义        | 保存目标            | 预览策略     |
|-------------------|-----------|-----------------|---------------------|--------------|
| ① 创建项目        | guide     | 创建即写盘      | project.md          | 步骤内联预览 |
| ② 创建智能体      | guide     | 创建即注册      | agents/charters/    | 章程预览     |
| ③ 阶段 brief      | drawer    | 失焦自动保存    | {zone}/brief.md     | 左写右看     |
| ④ 总结 / 反思     | split     | Ctrl+S 显式     | summary.md          | 实时同步     |
| ⑤ 自由笔记 / 追问 | inline    | Enter 发送      | notes.md            | 短消息渲染   |

### 4.2 `<MdEditor>` 最小 API

```js
<MdEditor
  mode="guide" | "drawer" | "split" | "inline"
  value={md}
  onSave(text, meta) → Promise<{ok, path}>
  previewMode="live" | "tab" | "off"
  toolbar={['h1','h2','bold','italic','code','link','list','quote','mermaid','attach']}
  mermaid={true}
/>
```

### 4.3 视觉方案

- **split 双栏**（总结/反思）：左编辑器 / 右实时预览，工具栏置顶
- **drawer 抽屉**（项目页 brief）：折叠态收成单行 chip，展开后双栏浮层
- **inline 行内**（追问/笔记）：浮在底部，工具栏简化，Enter 发送
- **guide 向导**（创建项目/智能体）：分步表单 + 步骤内联预览

## 5. 创建项目流程（V2 缺失，V3 新增）

入口：总览页"+ 新建项目"卡片 / 顶栏"+ 新建"按钮

```text
Step 1  主题 — 一句话
Step 2  动机 — 为什么学（Markdown，可跳过）
Step 3  当前能力 / 目标能力 / 完成标准
Step 4  选择起点阶段（默认 Intro）
Step 5  确认 → 写盘 project.md + state.json + 五个 zone 目录
Step 6  自动跳转到项目页，亮起点阶段
```

每一步都用 Markdown 编辑器（mode=guide），右侧实时预览 project.md 最终样貌。

## 6. 创建智能体流程（V2 占位，V3 完整）

入口：智能体页"+ 创建智能体"按钮

```text
Step 1  身份 — 名称 / 图标 / emoji
Step 2  章程 — Markdown，定义角色边界
Step 3  行为规则 — Markdown 列表
Step 4  允许阶段 — 多选 Intro/Explain/Practice/Extend/Summary/All
Step 5  默认输出目标 — 文件路径模板
Step 6  前置解析 — 表格，从前序 zone 选文件
Step 7  确认 → 写盘 agents/charters/{name}.md + 注册表更新
```

## 7. AI 产出主舞台放大策略

- 主面板字号 root 从 15 → 17px
- `.content` 行高从 1.78 → 1.85
- 代码块字号从 12.8 → 14px，背景对比度提升
- Mermaid 图全宽渲染，最大化可视化空间
- 学习者可选中任意段落 → 弹"批注 / 写困惑 / 加入笔记"浮层
- 右侧上下文面板可折叠，让主舞台在专注模式下做到 100%

## 8. 与文档的对应关系

| V3 设计点                    | 文档来源                                                          |
|------------------------------|-------------------------------------------------------------------|
| 五阶段流                     | LEARNING_PROJECT_STRUCTURE § Learning Flow Zones                  |
| 智能体章程                   | AGENT_ARCHITECTURE § Agent Role                                   |
| 创建项目写盘 project.md      | LEARNING_PROJECT_STRUCTURE § Required Files                       |
| 总结页学习者主权             | LEARNING_PROJECT_STRUCTURE § summary/summary.md Rule              |
| 前置文件解析                 | LEARNING_PROJECT_STRUCTURE § Agent Invocation Contract            |
| 记忆差异审查                 | LEARNING_PROJECT_STRUCTURE § Memory System                        |

## 9. 交付清单（等用户对齐后再做）

```text
v3/
  DESIGN.md                         本文档
  wireframes/
    01-shell-grid.svg               全局栅格
    02-intro-agent.svg              引入智能体单栏
    03-explain-agent.svg            讲解智能体双栏
    04-practice-agent.svg           练习智能体三栏
    05-extend-agent.svg             拓展智能体画布
    06-summary-agent.svg            总结智能体编辑器
    07-memory-agent.svg             记忆差异视图
    08-create-project.svg           创建项目向导
    09-create-agent.svg             创建智能体向导
    10-md-editor-modes.svg          编辑器四模式
  html/
    index.html                      总览（V3）
    project.html                    项目页（三明治）
    create-project.html             创建项目
    agents.html                     智能体注册表（V3）
    agent-detail.html               智能体详情（按角色动态布局）
    create-agent.html               创建智能体
    memory.html                     记忆系统（差异审查）
  css/
    base.css                        共享骨架
    md-editor.css                   Markdown 编辑器
    agent-layouts.css               智能体差异化布局
  js/
    md-editor.js                    <MdEditor> 实现
```

## 10. 待用户对齐的关键决策

```text
D1. <MdEditor> 用现成库（如 Milkdown / Tiptap）还是手搓？
    - 现成：开发快，包体大，定制深时仍要写一层适配
    - 手搓：可控性高，Mermaid 集成更原生，工作量大
D2. 创建项目/智能体是"独立路由"还是"模态对话框"？
    - 独立路由：专注感强，可分享链接
    - 模态：上下文不丢，更适合从项目内创建子项目
D3. 智能体差异化布局是"每个智能体独立路由"还是"统一路由 + 动态布局切换"？
    - 独立路由：URL 清晰，但页面多
    - 动态切换：URL 简洁，但需要把 6 种布局都打包进同一个组件树
D4. AI 产出主舞台的"批注/写困惑"是绑定到段落还是字符级？
    - 段落：实现简单，覆盖 90% 场景
    - 字符：类似 Google Docs，体验好但复杂度上一个台阶
D5. 终端抽屉是否升级为"可全屏"的窗格？
    - 当前 V2 只能展开/收起
    - 全屏：调试 prompt 时方便
    - 但和"主舞台是产出"的设计冲突，可能让终端抢戏
```
