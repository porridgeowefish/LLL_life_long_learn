# Lessons Learned — LLL 调试教训集

Status: active
Owner: project maintainer
Last reviewed: 2026-06-10
Source of truth: 把每一轮调试得到的"如果当初…就不会…"沉淀成铁律。
所有 agent 在动手前必须读这份文档，避免重复犯错。

## 0. 元原则

每一节都是用真实血泪切出来的。**没读这一份就动手 = 准备再踩一遍**。

每条教训的结构：
- **症状**：用户怎么发现问题的
- **根因**：我为什么犯错（通常是颗粒度错位 / 顶层设计错位）
- **铁律**：下次怎么避免

---

## 1. Claude CLI 运行模式（最严重）

### 症状
用户调用 Intro Agent，PowerShell 窗口弹出后显示一行行 JSON（`{"type":"message_start",...}`），用户说"我看不到对话能力，只有日志"。

### 根因
我用 `claude -p prompt.md --output-format stream-json --verbose`，**`-p` 是 print / single-shot 模式**，执行完就退出，**没有 TUI 聊天能力**。`--output-format stream-json` 让 stdout 是 JSON 行流，不是 markdown。

用户期望的是"和 PowerShell 里直接输入 `claude` 一样的效果"——真正的 TUI 聊天框，能输入、追问、用 `/help` 等命令。

### 铁律
- **`claude -p` ≠ `claude`**：前者是程序化单次执行，后者是 TUI 聊天框。
- 用户要"原生体验"就用裸 `claude`。要程序化捕获 stdout 才用 `-p`。
- **两者不可兼得**：TUI 模式 stdout 是 ANSI 控制序列，不能解析成 markdown。
- 解决方案：PowerShell 窗口直接跑 `claude`（裸命令），后端**不等退出**，前端通过 TanStack Query 自动 poll `output.md` 检测完成。

---

## 2. Headless 启动 + tail log ≠ 真实终端

### 症状
用户说"假终端"——PowerShell 窗口里 tail stdout.log，看到的是"日志回放"而非"原生 CLI 体验"。

### 根因
后端 headless spawn Claude + stdout 重定向到 stdout.log + PowerShell 窗口 Get-Content -Wait tail log。
这是双轨冗余：用户在 PowerShell 看一遍，UI 抽屉又显示一遍。
更关键：tail log 没有原生体验的颜色、流式 markdown、交互能力。

### 铁律
- **要让用户看到原生 CLI，必须在那个窗口里直接跑命令**——不能后端跑 + tail log。
- 用 `cmd /c start "title" /WAIT powershell -NoExit -Command "..."` + `CREATE_NEW_CONSOLE` 在新窗口里直接执行。
- 后端失去进程控制是接受的代价。

---

## 3. 双轨 UI 是反模式

### 症状
UI 有个"实时输出抽屉"（TerminalDrawer），用 SSE 把 stdout.log 内容流到前端模拟终端。用户在桌面 PowerShell 看一遍，UI 又显示一遍。

### 根因
我以为用户要"双视角审计"，实际用户要的是**专注**——一处看完就够了。

### 铁律
- 永远不要在 UI 里模拟一个真实终端已经提供的视图。
- SSE 流到前端的 raw text 不该被显示——只该驱动状态（如 invalidate query 让 OutputViewer 刷新）。
- 用户在 PowerShell 看 Claude 实时输出，在 UI 看 OutputViewer 渲染的最终 markdown——**分工而非重叠**。

---

## 4. React state vs URL 派生状态

### 症状
用户切到 Intro zone，调用 Intro Agent 报错 "agent explain not allowed in zone Intro"——但 UI 里 select 应该过滤了不兼容的 agent。

### 根因
`useState(compatibleAgents[0]?.id)` 只在首次 mount 设默认值，后续 zone 变化 compatibleAgents 重算了，但 React state 没重置。HTML select 在视觉上重置了，但 React state 还是旧的 'explain'，提交时 POST 旧 agent → 后端拒绝。

### 铁律
- **派生 state 必须随上游变化重算**。当某个 state 是从 props/URL/context 派生时，用 `useEffect([deps], () => setState(derived))` 同步。
- 或者直接 `useMemo` 派生（不存 state）。
- 不要假设 useState 默认值只算一次就够。

---

## 5. CSS max-width 是假响应式

### 症状
OutputViewer 渲染区右侧大片空白。

### 根因
OutputViewer 自己加了 `max-width: 880px`，父容器是 1fr 但子元素被限宽。看起来"渲染区不渲染"。

### 铁律
- **不要给内容容器加固定 max-width**——grid 父容器已经控制列宽。
- 如果一定要限宽（如长文档可读性），用 `max-width: 75ch`（字符单位）而非 px。
- 子元素的 max-width 必须**配合父容器自动居中**（`margin: 0 auto`），否则会出现"右侧空白"。

---

## 6. 多列 grid 要思考列的实际占用

### 症状
用户说"右边大片为假终端让步"——我误以为是 TerminalDrawer，实际是 invokeCol 占 280px。

### 根因
`grid-template-columns: 144px 1fr 280px` —— 中间 1fr 永远只占总宽 - 144 - 280。屏幕 1920 时主舞台 1496，看起来够，但 invokeCol 那 280px 是占用，不是浮动。

### 铁律
- 三列 grid 中如果右列是"工具栏"性质，考虑改成**底部 sticky bar**或**悬浮触发**。
- 不要让"工具栏"挤占"主舞台"——主舞台永远是 1fr 撑满。

---

## 7. markdown 扩展不可信（lexer 顺序陷阱）

### 症状
LaTeX `$A = U\Sigma V^T$` 渲染失败——`\Sigma` 被 marked 吃掉。

### 根因
`marked-katex-extension` 是 marked 的插件，但 marked lexer 在调用扩展前会预处理 backslash 转义。`\Sigma` 里的 `\S` 被 marked 当作转义符处理掉，KaTeX 实际收到的是 `S` 或别的。

### 铁律
- **不要盲信 markdown 扩展会优先处理你的语法**。
- 自定义语法的稳健做法：**预处理 + 占位符 + 后替换**。
  1. 在 marked 之前用正则提取你的语法（如 `$...$`）
  2. 渲染成 HTML，存到 Map
  3. 用唯一占位符（如 `MATHBLOCK0X`）替换原文
  4. marked 处理占位符化的 markdown（占位符是纯字母数字，marked 不动）
  5. 把占位符换回 HTML
- 这套模式适用于：数学公式、Mermaid 图、嵌入图表、自定义指令。

---

## 8. 内部 CLI 参数不要泄露到 UI

### 症状
AgentInvokePanel 暴露了 permissionMode select（default/acceptEdits/bypass），用户懵。

### 根因
我把 Claude CLI 的实现细节当成了用户选项。permissionMode 是后端控制 Claude 行为的参数，99% 用户根本不该看见。

### 铁律
- **UI 只暴露用户故事级别的选项**。CLI 参数、内部 flag、调试开关一律隐藏。
- 如果一定要让 power user 配置，藏在 env / settings 而非主表单。

---

## 9. 不要假设用户喜欢"专业感英文"

### 症状
产出里满是 "Knowledge Anchors / Relevance Hook / IN SCOPE / Learner background"。

### 根因
charter 和 primitives 全英文写，我觉得"专业"。Claude 严格遵循，产出全是英文术语 + 元对话（"自述未接触"、"以下锚点基于..."）。

### 铁律
- **目标用户说中文，所有 charter / primitive / 输出契约必须用中文**。
- 在 charter / primitive 顶部加"强制约束"段：
  - 全中文（显式列禁用英文术语清单）
  - 禁止元对话（不要"自述"、"learner..."、"以下..."等过程说明）
  - 给 ❌ 反例 + ✅ 正例

---

## 10. 全局 CSS 一刀好过散点修

### 症状
用户说"还是有斜体"——我之前只改了 OutputViewer 的 em。但 19 处 font-style: italic 散在各 module.css。

### 根因
散点修改漏改。CSS Modules 隔离了作用域，但用户的视觉偏好是**全局**的。

### 铁律
- **视觉偏好（如禁用斜体、统一字号）用 globals.css 全局声明**。
- `* { font-style: normal !important; }` + 例外白名单，比逐个 module 改可靠。
- 例外用 `:global(.foo)` 显式标注。

---

## 11. owner 意识：自己控制的才是稳定的

### 元教训
每一轮切刀，我都盲信第三方（marked 扩展、Claude CLI 文档、CSS 默认值），结果每一个都翻车。

### 铁律
- **plugins 是别人的，预处理是自己的**。
- 涉及关键渲染管线（数学公式、Mermaid、prompt 注入），自己写预处理 + 后处理。
- 涉及关键集成（Claude CLI），用 wrapper 脚本明确控制命令流，不依赖工具的默认行为。

---

## 12. Go 写 .ps1 必须带 UTF-8 BOM（中文路径乱码陷阱）

### 症状
调用 Intro Agent 弹出 PowerShell 窗口，报错"找不到路径 D:\...\projects\閲戣瀺鎶曡祫"（`金融投资` 被读成 `閲戣瀺鎶曡祫`）；banner 全乱码；Claude TUI 开了但 prompt 没注入（空 prompt）。

### 根因
Go `os.WriteFile` 写 wrapper.ps1 是 **UTF-8 无 BOM**。PowerShell 5.1 读 .ps1：**无 BOM 就按系统 ANSI 码页解码**（中文 Windows = GBK/CP936）→ 所有中文路径/文本全乱码。`金融投资` 的 UTF-8 字节被当 GBK 解成 `閲戣瀺鎶曡祫`。
- Set-Location 找不到乱码路径 → 失败。
- Get-Content prompt.md 路径乱码 → 读不到 → `$promptText` 为空 → claude 开了但没注入 prompt。
- claude 本体能开，是因为 `Get-Command claude` 路径纯英文没乱码。

### 铁律
- **Go 写给 Windows PowerShell 5.1 消费的 .ps1 / .txt，必须前置 UTF-8 BOM**（`[]byte("\ufeff"+content)` 或 `append([]byte{0xEF,0xBB,0xBF}, ...)`）。
- 读 UTF-8 无 BOM 的文件（如 prompt.md）要显式 `Get-Content -Encoding UTF8`，否则 PS 默认按 GBK 读，中文内容乱码。
- 验证：`[System.IO.File]::ReadAllBytes` 查前 3 字节是否 `EF BB BF`；对照实验（无 BOM 失败 / 有 BOM 成功）。
- 别把字面 BOM 字符写进 Go 源码——编译器报 "invalid BOM in the middle of the file"，用 `"\ufeff"` 转义或字节字面量。
- 同类风险：任何"后端 Go 写脚本/配置 + PowerShell/cmd 消费"且含非 ASCII 路径的场景。

---

## 13. PowerShell 5.1 把含 ASCII 双引号的长字符串劈成多参（argv 截断陷阱）

### 症状
claude TUI 收到的 prompt 只到"必选 Primitives / 简化的本主题包含"就断（前 642 字符），输出契约、行为规则、展开的 primitive 全丢。Agent"没写文档"——不是偷懒，是真没收到要求；它回显收到的 prompt 也是断的，停在完全相同的位置。

### 根因
PowerShell 5.1 的 legacy native-argument passing：把一个**含 ASCII 双引号（U+0022）的字符串**传给 native exe 时，无法把它当成单个 argv 元素——内嵌的 `"` 破坏了外层引号包裹，被 CRT 重新切分成多个 argv。LLL 的 prompt.md 含 62 个 ASCII 双引号（charter / primitive 描述里的"已有知识""本主题包含"等），4422 字符被劈成 51 段，claude 取 argv[1] = 前 642 字符，后面静默丢弃。
- 反引号（markdown code span）**不是** breaker——只有 U+0022 是。
- PS 7.3+ 有 `$PSNativeCommandArgumentPassing` 可解；5.1 没有，只能避开。

### 铁律
- **PS 5.1 传给 native exe 的字符串，不能含 ASCII 双引号（U+0022）。** 凡是要作为单参传过去的（如 claude 的 prompt 位置参数），先 `-replace [char]34, [char]0x201C`（换成中文引号 "）、换单引号、或删除——验证手段：写个 receiver 脚本（打印 ARGC + ARG0_LEN），对照 ARGC=1 且 ARG0_LEN = 源长度才算通过。
- **只验磁盘文件完整 ≠ 消费方收到完整。** 文件写对了，不代表传过 argv 后还对。涉及 native argv 的链路，必须验"消费方实际收到的"，不能停在文件层。
- **别信"agent 的截断报告一定是幻觉"。** 它可能在忠实回显它真收到的截断内容。先验传输路径，再下结论——本条是连犯两次的教训：先断言"编码没问题"、再断言"没截断是 agent 幻觉"，两次都被实锤打脸。
- 同类风险：任何"PowerShell 把长字符串当位置参数传给 native exe"且内容可能含 `"` 的场景。

---

## 14. 集成协议字段位置要对着规范验，不要看着像就解析（Anthropic usage 嵌套陷阱）

### 症状
用户反馈"token 记录有误，前端看不到输入 token"。usage.jsonl 里 anthropic kind 的 provider（glm-5.3）全部 `inputTokens:0`，而 openai kind（deepseek）有正常值。代码自 iter-16 后零改动，"最近才出现"。

### 根因
`gateway.go streamAnthropic` 在 `message_start` 事件里读**顶层** `usage` 字段。但 Anthropic SSE 规范中，`message_start` 的 usage 嵌套在 `message.usage` 里；只有 `message_delta` 的 usage 才在顶层。所以 input_tokens 永远读不到。openai 路径的 usage 在 chunk 顶层、解析正确——同一文件里两条路径一个对一个错，写的时候没对着协议文档逐一核对嵌套位置。

### 铁律
- **对接第三方流式协议时，每个事件类型的字段路径要对着官方规范抄，不要"看下一个样例就推广"**。SSE 事件里同一概念（usage）在不同事件类型中的嵌套位置可能不同。
- **换 provider 是隐式回归测试**。usage 解析 bug 一直存在，只是之前一直用 openai kind 没触发。"代码没变"≠"行为没变"——先查数据侧（本例：usage.jsonl 新旧记录对比直接定位）。
- 验证手段：httptest 假 SSE 服务器按规范发事件，断言 usage 两个字段都被捕获。

---

## 15. 流式 markdown 管线必须处理"未闭合围栏"（代码墙陷阱）

### 症状
用户反馈老师对话页"文字乱排"，且"偶尔出现、过会儿又没了"。事后检查渲染完成的历史消息完全正常——问题只出现在**流式生成期间**。

### 根因
`useMarkdown.ts` 的 extractMermaid/extractSVG 正则要求**闭合**围栏（```` ```mermaid...``` ````）。流式输出期间围栏还没闭合，正则不匹配 → 原文直接进 marked → marked 把未闭合围栏后的**整段剩余消息**渲染成一个巨型 `<pre>` 代码块（等宽、无样式、和正文完全不同排版）——用户看到"乱排"。围栏闭合后（生成完成）重新渲染又正常——所以"偶尔出现、事后查不到"。

### 铁律
- **任何消费"增量/半成品 markdown"的管线，都要显式处理未闭合围栏**：把 dangling 的围栏尾巴替换成"生成中"占位符，而不是让它落进通用渲染器。
- **"完成后正常、过程中异常"的渲染 bug，要在流式路径上复现**——只测最终态 HTML 永远测不到。测试要覆盖"只有开围栏没有闭围栏"的输入。
- 用户说"偶尔出现、现在没了"时，优先怀疑生命周期相关（流式中/加载中/竞态），别只查静态渲染。

下次 agent 进入项目时：

1. 读 AGENTS.md（团队原则 + 工作规则）
2. 读本文档（调试教训）
3. 再读相关 docs/00-product-and-architecture/* 文档

每个 wave 完成后，如果发现新的"如果当初…就不会…"，追加到本文档对应章节。

---

## 16. 存储根路径必须同源解析，测试隔离要覆盖每一个存储（folders.json 数据丢失）

### 症状
用户升级后：18 个分类文件夹只剩 13 个（腾讯云/Agent/编译原理等自定义文件夹消失）、已分类学习单元全部落入"未分类"。projects/folders.json 被 sync 从空台账重建（新式 ID 指纹）。

### 根因
folderstore 用 `paths.WORKSPACE` 拼路径，而全仓其他存储走 workspace 模块的 projectsRootOverride。两者在测试进程/异常启动时分叉：某个进程以空台账 + 真实项目缓存跑了一次 sync，把"仅含地图文件夹"的布局固化覆盖了 canonical 位置；自定义文件夹与 slugOrder 从此不可见。迁移复制失败还被 `_ =` 静默吞掉，空台账继续运行。

### 铁律
- **同类数据位置的解析必须收敛到同一个函数**（workspace.ProjectsRoot()），不许各自拼路径——一处 override 一处漏，就是一处数据丢失面。
- **迁移/复制失败必须显式报错**，绝不带着空状态继续——下一个写操作会把空状态固化为"事实"。
- 写"条件性迁移"代码时，必须写一个"测试进程绝不触真实磁盘"的回归测试（override 分叉场景）。
- 数据可恢复性优先：旧位置文件在迁移后保留（copy-forward），这次 18 个文件夹正是靠根目录快照救回的。

---

## 17. 定高 flex 列里的 overflow:hidden 子项会被压成边框（热力图消失）

### 症状
首页"学习节律"热力图整个消失。DOM 里 181 个格子都在、数据正常，但 section 高度只剩 2px（= 上下边框）。

### 根因
`.scroll` 是定高（视口驱动）flex 列容器。热力图 section 是唯一 `flex-shrink:1` 的子项，且自带 `overflow:hidden`（为圆角）——CSS 规范规定 overflow 非 visible 的 flex 子项 `min-height:auto` 下限为 0，于是它吸收了全部超额收缩。触发条件：项目区因另一个 bug（未分类洪泛）膨胀到 1393px。平时内容少时不触发，"偶尔出现、事后查不到"。

### 铁律
- **滚动列（overflow:auto 的 flex column）里的直接子块一律 `flex-shrink: 0`**——它们要靠滚动容纳，不是靠挤压。
- **"元素存在但看不见"先量盒子**：`getBoundingClientRect().height` 2px = 只剩边框，是 flex 塌缩/被隐藏的指纹。
- flex 子项加 `overflow:hidden` 时记住你同时交出了 `min-height:auto` 保护——这两个属性组合是塌缩高危。
- 消融法定位快：`flex:none` 一改就弹回 509px，根因当场锁定。

当前共 17 条。
