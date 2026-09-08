# 项目导航

进入任务前读取本文件、匹配的 `.codebuddy/rules/` 与相关 `docs/knowledge/`、`docs/function/` 导航。工作流事实仅位于 `docs/workflows/<workflow-id>/`；机器状态仅由运行时写入 `.codebuddy/workflows/<workflow-id>/state.json`。

Rules 按操作强制装配：模块或重构读取 architecture 与 engineering；API、数据或持久化读取 architecture 与 api-and-data；测试读取 testing；提交、MR 或报告读取 commit-and-mr；陌生代码只读探索读取 architecture 与对应工程导航。

## 全局行为规则（所有会话、所有工具生效）

1. 不臆测。不隐藏困惑。暴露权衡取舍。
2. 用最少的代码解决问题。不写投机性代码。
3. 只动必须动的。只清理自己留下的。
4. 明确成功标准，循环直到验证通过。

项目专属规则写入 `.codebuddy/rules/`，不写入本文件。
