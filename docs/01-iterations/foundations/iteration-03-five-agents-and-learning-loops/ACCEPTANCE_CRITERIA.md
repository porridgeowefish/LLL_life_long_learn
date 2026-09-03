# Iteration 03 Acceptance Criteria

Status: draft
Owner: project maintainer
Last reviewed: 2026-06-09
Source of truth: derived from USER_STORIES.md (37 stories) + README Key Decisions.

每条验收点必须是二元可测（pass / fail）、可观察。分组对应 USER_STORIES 的 section，
便于回溯。拍板项（D-*）必须有显式验收，不能藏在故事里。

---

## 项目入口（IN）

```text
创建项目表单含 4 字段：current_ability / target_ability / difficulty / deliverables
difficulty 选项附带等级说明文本（非裸名字）
deliverables 支持常见选项 + 自定义补充
"完全不了解" / "已有经验但概念混乱" 等非等级化表达可用作 current_ability 输入
高级选项（分类 / 模板 / 起点阶段 / 备注）默认折叠，不挡主路径
直接选 Practice 或 Summary 起点时，系统提示缺失前置材料
勾选技术经历不被等同记录为"已掌握"
4 字段在 Intro / Explain / Practice / Summary 各阶段可追溯读取
```

## Intro（I）

```text
Intro Agent 产出含四个段：兴趣钩子 / 历史脉络 / 入口问题 / 先验激活
兴趣钩子与用户目标 + 起点相关，非通用励志文本
兴趣钩子至少提出一个可在后续阶段验证的悬念
历史叙事区分：确定史实 / 常见解释 / 类比性叙事
入口问题数量 2-4 个
每个入口问题标注在哪个阶段（Explain/Practice/Extend/Summary）被处理或验证
入口问题保留用户原始表述 + 结构化改写
Explain 调用时自动继承 Intro 入口问题，无需用户重复输入
先验激活勾选可跳过，且不冒充测评结果
无伪史料、虚构名人故事、夸大结论
```

## Explain（E + D-E-04）

```text
Explain 调用时不出现新的启动表单（零额外输入）
Explain 自动读取项目 4 字段 + Intro 输出 + 入口问题 + 已标困惑
Explain 输出覆盖：第一性原理 / MECE 结构 / 概念关系 / 边界 / 常见误解
输出写入 explain/output.md，原始运行写入 runs/
重新调用 Explain 不静默覆盖旧版本（保留版本或明确替换确认）

选中原文浮层含 4 动作：标困惑 / 追加问题 / 加入笔记 / 取消      [D-E-04]
选中浮层任何动作都不触发 AI invoke                              [D-E-04]
"追加问题"动作把选中文本放入底部提问框，不立即发送              [D-E-04]
"标困惑"动作保存到困惑清单，可用于练习生成                      [D-E-04]
困惑数据含 char_start / char_end / quote_snapshot               [E-02 / D-Q-03]
原文更新后，定位漂移时 UI 显示"定位可能漂移"警告
困惑可查看 / 编辑 / 解决 / 删除

页面级"统一提问 (N)"按钮存在，N = 困惑清单待问条数              [D-E-04]
点击"统一提问"弹出注入预览（prompt + source_refs + quote_snapshot）[D-E-04]
确认后单次 invoke 把 N 个困惑合并成单个 user turn               [D-E-04]
提交后对应 confusion 标记为 asked，但仍可继续追问               [D-E-04]

底部追问追加到同一学习时间线，不覆盖前文
发送追问前展示将注入的引用、前置文件、会话状态
会话不可恢复时系统新建 session 但挂同一 Explain 线程

"进入练习"按钮只切换页面，不自动生成题（除非用户确认题量）     [E-04]
"针对困惑生成练习"时清楚列出会被采用的困惑                     [E-04]
```

## Practice（P + D-Q-02）

```text
题量输入范围 1-10 整数，默认 5                                  [P-01]
题量超过 10 时不静默截断：说明原因 + 要求二次确认              [P-01]
显示预计完成时间或题型分布                                     [P-01]

生成的题目不含答案 / 解析 / 评分结论                           [P-02]
答案与解析存放在学习者不可见的评审 artifact                    [P-02]
提示分层解锁，第一层不揭示关键步骤                             [P-02]

至少一道题针对用户未解决困惑                                   [P-03]
至少一道题要求迁移到新情境（非复述原文）                       [P-03]
系统记录 question -> objective / confusion / source 映射        [P-03]

任何重新生成不删除已提交答案                                   [P-04]
跳过与不会做分开记录                                          [P-04]

题目切换时自动保存草稿                                         [P-05]
未作答 / 已作答未自评 / 已完成题状态可区分                    [P-05]

每题掌握自评量表 1-5（完全不会 / 勉强 / 基本会 / 较熟练 / 能迁移）[P-06]
自评发生在 AI 最终评价之前                                    [P-06]
AI 评价不羞辱或机械判定"过度自信"                              [P-06]

提交前二次确认：题数 / 空题 / 自评缺失 / 提示使用              [P-07]
提交产生 attempt id + 提交时间                                 [P-07]
提交后修改形成新 attempt，不覆盖旧评估证据                    [P-07]
AI 评估失败时保留快照允许重试                                  [P-07]

逐题评价引用用户答案中的具体证据                              [P-08]
评价区分正确性 / 推理质量 / 迁移能力 / 表达完整度              [P-08]
评价含总体能力画像 + 自评偏差 + 下一轮建议                    [P-08]

单题无 evaluate 按钮                                           [D-Q-02]
设置中无逐题评估 opt-in 开关                                   [D-Q-02]
只有整批提交 → AI 整批评估一条路径                             [D-Q-02]

保存草稿与提交评估是两个不同动作                              [P-09]
提交按钮在请求中禁用并展示进度，避免重复 attempt              [P-09]

评估落盘 practice/evaluations/<attempt>.json（事实源）         [DATA-02]
评估落盘 practice/evaluations/<attempt>.md（可读副本）         [DATA-02]
```

## Extend / 拓展（D + D-Q-01 + D-Q-07）

```text
项目侧栏该 zone 显示名为"拓展"                                  [D-Q-01]
路由 /project/:id/extend 仍然有效（zone id 不变）              [D-Q-01]
文件系统 <project>/extend/ 目录名不变                           [D-Q-01]

Extend Agent 按四关系维度引导思考，不画图（无 .mmd）            [D-Q-07]
extend/output.md 按因果/结构/重要性/同构组织追问                [D-Q-07]
每维度先抛引导问题，不直接给结论                               [D-Q-07]
学习者可 follow-up 回应，agent 据回答深挖                      [D-Q-07]
无 Mermaid 渲染 UI                                             [D-Q-07]

因果维度追问区分直接原因 / 条件 / 机制 / 中介 / 结果           [D-02]
因果至少追一个反事实或失效条件                                 [D-02]
因果提示用户检查证据强度，不直接宣布因果                       [D-02]

结构维度追问支持从整体展开子结构，也支持从部件回到整体         [D-03]

重要性维度追问必须附带维度（先修性/频率/风险/迁移价值）        [D-04]
用户可改变维度后重新讨论排序                                   [D-04]

同构维度追问明确源领域 / 目标领域 / 对应映射                   [D-05]
同构至少指出一个映射失效点                                     [D-05]
同构区分类比启发 / 结构同构 / 表面相似                         [D-05]

领域提示数量克制，每维度 2-4 个                                [D-06]
默认不给终局答案，可提供逐层提示                               [D-06]
用户的关系陈述和回答可保存到 extend/relation-notes.md          [D-06]
```

## Summary（S + D-Q-05）

```text
Summary 路由只有一个 /project/:id/summary（单路由）            [D-Q-05]
Summary 页内含两个 tab：闪卡 / 总结                             [D-Q-05]
Summary 绝无左右分屏                                           [D-Q-05]
切换 tab 不丢失闪卡复习进度（store 持久化）                    [D-Q-05]

每张闪卡可追溯到 source artifact 或练习证据                    [S-01]
闪卡覆盖：概念回忆 / 辨析 / 因果 / 结构 / 迁移 / 易错点        [S-01]
AI 标注每张闪卡生成原因，用户可删除或改写                      [S-01]

闪卡默认只显示正面问题                                        [S-02]
翻面前可输入自己的答案                                        [S-02]
翻面后可评价：忘记 / 模糊 / 掌握 / 太简单                      [S-02]
评价更新本地复习计划，不静默改写知识事实                      [S-02]
支持上一张 / 下一张 / 随机 / 只看错卡 / 重置本轮               [S-02]

summary/summary.md 始终由用户最终控制，不被盲覆盖              [S-05]
总结引用练习证据和用户笔记，非仅摘要 AI 讲解                  [S-05]
明确"已掌握"的证据门槛与不确定性                              [S-05]
交付成果有完成 / 部分完成 / 未完成状态 + 证据链接             [S-05]

[S-06 总结 AI 建议子系统 — DEFERRED iter-05+，见 README Deferred。
 iter-03 的 summary.md 由用户完全手写，无 AI 建议注入流程。]
```

## 数据层（DATA）

```text
学习痕迹事件含字段：id / project_id / zone / type / actor / created_at / source_refs / artifact_version  [DATA-01]
用户删除内容时定义软删除或墓碑事件                             [DATA-01]
AI 推断与用户原始输入分开存储                                  [DATA-01]

评估写 evaluations/<attempt>.json（事实源）+ .md（可读副本）   [DATA-02]
文件为事实源；DB 索引层 iter-03 不实现                         [DATA-02]
写入失败明确展示哪一层失败并允许重试                          [DATA-02]

[DATA-03 记忆建议子系统 — DEFERRED iter-05+，见 README Deferred。
 iter-03 项目记忆只读，无建议 accept/rewrite/reject 流程。]
```

## 导航（BTN-01）

```text
当前路由与选中态一致
存在未保存编辑时提示或自动保存
缺失路由不显示为可用链接
项目树"其他项目"必须进入对应项目，不统一跳总览
```

## Markdown 编辑器（BTN-04）[HIGH PRIORITY]

```text
编辑同时渲染：输入 **粗体** 实时显示为粗体
备选双栏模式：两栏内容同步
双栏分隔条可拖拽调整两栏宽度
外层容器宽度保持学习页设计宽度（约 75ch）
工具栏对选区操作；无选区时插入可撤销标记
编辑 / 预览共享同一内容源（切换不丢内容）
自动保存 / 手动保存 / 未提交状态视觉区分
链接与附件进行路径 / 协议安全校验
Mermaid 预览错误不阻止保存源码
编辑器写回原 artifact 路径，不引入新文件
草稿用 localStorage 临时存，保存后清除
Practice 作答 / Summary 总结 / Explain 笔记均使用同一编辑器组件
```

## Carryover 回归（C-01..C-04）

```text
调用 Intro Agent 时桌面弹出新的 PowerShell 窗口                [C-01]
窗口显示青色 banner + "初始 prompt 会自动带入"提示             [C-01]
窗口启动 Claude TUI，prompt 作为首条消息自动带入，可直接对话    [C-01]
启动/解析失败时 runs/<ts>-<agent>/stdout.log 含 PATH + 原因    [C-01]

Explain zone 调用后 OutputViewer 渲染区无右侧大片空白          [C-02]
含 \Sigma / \boldsymbol / 多行 display math 的文档正常渲染     [C-03]
除 topbar logo 外，全站无斜体渲染                              [C-04]
```

---

## 验收覆盖自检

| 来源 | story/决策数 | 验收点覆盖 |
|------|-------------|-----------|
| 项目入口 IN | 5 | ✅ |
| Intro I | 4 | ✅ |
| Explain E | 3 + D-E-04 | ✅ |
| Practice P | 9 + D-Q-02 | ✅ |
| Extend D | 6 + D-Q-01 + D-Q-07 | ✅ |
| Summary S | 4 + D-Q-05（S-06 defer iter-05+） | ✅ |
| Data DATA | 3（DATA-02 瘦文件优先；DATA-03 defer iter-05+） | ✅ |
| 导航 BTN-01 | 1 | ✅ |
| 编辑器 BTN-04 | 1（D-BTN-04 高优） | ✅ |
| Carryover | C-01..04 | ✅ |
| 拍板 D-Q-03（字符级数据） | 已并入 Explain 段 | ✅ |
| 拍板 D-Q-07（Extend 引导思考） | 已并入 Extend 段 | ✅ |
| 拍板 D-Q-04 / D-Q-06 | 认同无修改，无新验收点 | N/A |

未覆盖说明：D-Q-04（DB 选型）和 D-Q-06（不扩 scope）是治理原则，不是可测功能，
故无验收点。它们的约束已通过"文件为事实源""不重写 backbone"等条目间接覆盖。
