# MOI Storybook

MOI Storybook 是核心产品能力的**场景化回归验收集**。它把原本不可控的 AI 行为，转换为固定输入、固定验收条件和可重复执行的产品 Case。

每次产品、模型、提示词、工具、工作流或数据链路更新后，所有标记为 `required` 的 Case 都必须重新通过；任何 `failed`、`blocked` 或 `not run` 的必过 Case 都不能被平均质量指标掩盖。

## Storybook 与 Eval 的区别

| 体系 | 关键问题 | 结果 |
|---|---|---|
| Storybook | 核心业务链路能否完成？ | Pass / Fail，作为迭代门禁 |
| Eval | 在链路完成时，结果质量是否提升或退化？ | 指标、趋势与基线差异 |
| 单元与接口测试 | 单一模块是否符合技术合同？ | 技术正确性 |

Storybook 可以引用 Eval 指标作为补充证据，但不会用均分替代一条场景的明确验收断言。

完整的设计、执行和治理方法见 [Storybook 测试方法论](testing-methodology.md)。新建 Case 请从 [Case 模板](case-template.md) 开始；运行产物遵循 [运行、证据与报告规范](execution-and-evidence.md)。

## Case 合同

每个 Case 必须定义：

1. 用户目标与需控制的产品风险；
2. 合成或脱敏的输入夹具，以及其版本或校验信息；
3. 前置条件、产品操作路径或 SDK/API 操作；
4. 可判定的输出与验收断言；
5. 证据产物、失败口径与清理要求；
6. 对应的运行状态与版本边界。

## 状态与门禁

Case 内容状态：`draft`、`ready`、`deprecated`。

一次运行结果：`pass`、`fail`、`blocked`、`not run`。

发布或迭代检查时，所有 `required` Case 必须为 `pass`。`not run` 只表示尚未取得证据，不表示通过；`blocked` 必须记录阻塞原因并按发布规则处理。

## 目录

- [testing-methodology.md](testing-methodology.md)：从产品承诺、风险建模、夹具和断言，到运行门禁、证据与 Case 治理的完整方法。
- [case-template.md](case-template.md)：可复制的 Case 合同模板与填写口径。
- [coverage-map.md](coverage-map.md)：产品承诺、风险与 Storybook Case 的覆盖规划矩阵。
- [execution-and-evidence.md](execution-and-evidence.md)：统一的运行阶段、JSON 报告、证据脱敏、失败分类和清理协议。
- [scripts/validate-case-contracts.mjs](scripts/validate-case-contracts.mjs)：检查每条公开 Case 是否具备完整的执行合同和隐私边界。

按产品域组织的完整导航见 [INDEX.md](INDEX.md)。

| 目录 | 回归范围 |
|---|---|
| [document-intelligence/](document-intelligence/) | 文档、图片、音视频的理解、抽取与交付。 |
| [workflow/](workflow/) | 编排、数据输入输出、触发器与受控执行。 |
| [data-knowledge/](data-knowledge/) | 数据接入、Catalog、知识库、检索与可追溯问答。 |
| [agent/](agent/) | 智能体、工具协作、结构化交付与安全边界。 |
| [evaluation/](evaluation/) | 评测数据、基准执行、结果审计与质量门槛。 |
| [operations/](operations/) | 发布、运行、诊断、模型服务与运营治理。 |

## 提交前校验

在仓库根目录运行：

```bash
node storybook/scripts/validate-case-contracts.mjs
```

该检查会拒绝缺少十段式合同、夹具管理、负向路径、资源生命周期、`required` 门禁或 `not run` 初始状态的 Case，并阻止常见的公开链接、内部任务编号和敏感配置模式进入 Case 文档。

## 公开边界

本仓库仅保留可公开的 Case 合同、合成/脱敏夹具说明和脱敏证据。真实环境地址、凭据、客户资料、人员、内部任务链接、资源 ID 与原始运行日志均不进入本目录。完整自动执行与私有证据由内部验证体系维护。
