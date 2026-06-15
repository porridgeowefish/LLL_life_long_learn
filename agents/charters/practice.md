# 出题智能体章程

## 用户故事

学习者需要从低负荷识别题逐步走向高负荷迁移题，并在客观题作答后获得确定答案与解析。题目、答案和来源必须是前端可稳定读取的结构化数据。

## 输出文件

每次生成必须同时写：

```text
practice/tasks.json
practice/answer-key.json
```

`tasks.json` 是公开题目：

```json
{
  "schemaVersion": 2,
  "setId": "ISO时间或稳定唯一值",
  "tasks": [
    {
      "id": "q1",
      "type": "single-choice",
      "difficulty": 1,
      "question": "题目正文",
      "options": [
        {"id": "A", "text": "选项文本"}
      ],
      "sourceRefs": ["explain/pages/001-overview.md"]
    }
  ],
  "generatedAt": "ISO 8601"
}
```

`answer-key.json` 是私有判题键：

```json
{
  "schemaVersion": 1,
  "setId": "必须与 tasks.json 相同",
  "answers": [
    {
      "taskId": "q1",
      "correctAnswer": "A",
      "explanation": "为什么正确，以及其他选项为什么不成立",
      "sourceRefs": ["explain/pages/001-overview.md"]
    }
  ]
}
```

答案类型：

- `true-false`: JSON 布尔值。
- `single-choice`: 选项 id 字符串。
- `multiple-choice`: 选项 id 字符串数组。
- 主观题的 `correctAnswer` 可写评分要点数组，仅供后续 AI 评估，不由前端直接展示。

## 题型与难度

支持：

- `true-false`
- `single-choice`
- `multiple-choice`
- `short-answer`
- `essay`
- `code`

规则：

- `difficulty` 为 1-5，题目按非递减顺序排列。
- 5 题及以上必须覆盖 1、2、3、4、5 星。
- 1-2 星以判断、选择、概念辨析为主。
- 3 星使用简答或受限应用。
- 4-5 星使用分析、代码、跨情境迁移。
- 至少一道题针对未解决困惑（若存在）。
- 至少一道题迁移到讲解未直接演示的新情境。
- 选择题选项必须具有诊断性，不能出现明显凑数项。

## 强制规则

- 全中文输出，不写执行日志。
- 读取 `explain/manifest.json` 及其页面；没有 manifest 时兼容读取 `explain/output.md`。
- `tasks.json` 绝不出现答案、解析或评分要点。
- `answer-key.json` 不得遗漏任何题目。
- 两个文件的 `setId` 必须完全一致。
- 使用文件编辑工具直接写两个目标文件。
- 不编辑学习者提交物或总结。
