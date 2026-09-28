# 工作流设计原始资料

[工作流交互演示](../../../product/moi-platform-prototype/videos/workflow-interaction-demo.mp4) · [MOI 工作流产品设计](../../product-overview.md#3-工作流从能力配置到运行结果)

这里迁入 CanvasFlow 的工作流设计文档、契约、提示词与实测记录，保留原有论证、修订过程和未完成事项。文档正文基本保持原文，仅对个人姓名与迁移后失效的文件链接做必要处理。阶段性记录可能与后来的实现不同，请先读[架构](docs/架构.md)和[取舍](docs/取舍.md)，再按时间阅读历史记录。

## 设计理念与核心机制

- [交互理念与项目边界](docs/交互理念.md)
- [架构：整条环](docs/架构.md)
- [关键取舍](docs/取舍.md)
- [状态机](docs/状态机.md)
- [执行者](docs/执行者.md)
- [PlanProposal 契约](docs/plan-契约.md)
- [词表](docs/词表.md)

## 过程、场景与证据

- [画布与对话融合：交互秩序改造计划](docs/交互秩序改造计划.md)
- [设计记录](docs/设计记录.md)
- [观察记录](docs/观察.md)
- [信息不完整时的第一次 Plan 回合](docs/scenarios/01-contract-intake-initial-plan.md)
- [节点发起整图修订的实测记录](fixtures/observed/run14-节点整图修订/README.md)
- [早期 README 存档](docs/archive/readme-2026-09-07.md)

## 配套材料

- [契约文件](contracts/)：计划、节点、步骤提交与工作流修订的 JSON Schema。
- [Agent 提示词](prompts/)：规划、首次构建与整图修订的原始提示词。
- [观察数据](fixtures/observed/)：各轮测试输入、事件、提交结果与计时。
- [开发指南](docs/开发指南.md)与[TypeScript 迁移记录](docs/TypeScript迁移.md)：保留原项目的实现与验证背景；这里迁移的是设计资料，运行命令需在原项目执行。

原始资料对应 [CanvasFlow 仓库的 `824397e` 版本](https://github.com/Bai-009/canvas-first-workflow/tree/824397ed995dc7e1293a383bf3b90d1bc1cdf63e)。本文档目录与本案例集的 [MOI 工作流管理 PRD](../../prd/02-workflow-processing/complex-workflow-management-prd.md)和[工作流安全预览验收场景](../../../storybook/workflow/safe-workflow-review-and-qa-package.md)并列，供对照阅读。
