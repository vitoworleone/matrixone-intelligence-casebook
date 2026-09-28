<p align="center">
  <img src="assets/architecture/casebook-banner.png" alt="MatrixOne Intelligence Casebook" width="900" />
</p>

# MatrixOne Intelligence Casebook

这份案例集记录 MatrixOne Intelligence（MOI）从企业数据到业务智能体的产品设计：文件与业务表如何加工为可用知识，智能体如何调用知识与工具完成任务，以及结果如何被检查和复现。仓库收录产品需求、架构、交互原型、场景验收与评测材料；[产品全景与工作范围](docs/product-overview.md)说明各部分之间的关系。

我的工作集中在**工作流和知识库产品设计**，并参与智能体与 Astra 的产品适配梳理；Storybook 场景、产品手册和远程演示用于把设计落到可操作、可验证的任务中。

<p><strong>中文</strong> | <a href="README_EN.md">English</a></p>

<p>
  <a href="docs/architecture/canvasflow-workflow/"><img src="https://img.shields.io/badge/Workflow-Design%20Docs-5945a3?style=flat-square" alt="工作流设计原始资料" /></a>
  <a href="#工作流交互演示"><img src="https://img.shields.io/badge/Video-1080p-1266b2?style=flat-square" alt="1080p 工作流视频" /></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-MIT-b89916?style=flat-square" alt="MIT 许可证" /></a>
</p>

## 演示视频

### 工作流交互演示

1080p 视频沿一次任务展示目标输入、流程方案形成、画布节点生成与修改。对应的[工作流设计原始资料](docs/architecture/canvasflow-workflow/)完整保留交互理念、架构取舍、状态机、契约、提示词与实测记录；[MOI 工作流管理 PRD](docs/prd/02-workflow-processing/complex-workflow-management-prd.md)描述平台侧的产品规则。

https://github.com/user-attachments/assets/242fd58e-8d4c-488a-b59d-9a357e252049

### Astra 运行时演示

这段来自 [Astra 官方仓库](https://github.com/matrixorigin/Astra)的视频展示长任务在 CLI 中的执行过程，包括上下文保留、执行取舍与运行分析。案例集中的[Astra 适配讨论](docs/product-overview.md)关注 MOI 平台配置与运行能力的衔接。

https://github.com/user-attachments/assets/c008be26-4320-413c-9ad6-100aefcfa728

## 原型界面预览

下面保留原有的 MOI 原型截图，分别呈现智能体工作台和数据工作台。可交互页面与相关工程材料位于[产品原型目录](product/moi-platform-prototype/)。

<img src="assets/screenshots/moi-platform/agent-workbench-home.png" alt="MOI 智能体工作台首页" width="100%" />

*智能体工作台：首页对话入口、智能体使用入口与资源中心。*

<br />

<img src="assets/screenshots/moi-platform/data-workbench-overview.png" alt="MOI 数据工作台概览" width="100%" />

*数据工作台：数据对象、工作流、计算资源与知识库的统一入口。*

## 设计资料与验证

建议先读[产品全景](docs/product-overview.md)，再按关心的环节进入原始设计、产品规则和场景验收：

- **工作流与数据加工**：[原始设计资料](docs/architecture/canvasflow-workflow/)保留完整的设计过程、契约、提示词和观察记录；[工作流管理 PRD](docs/prd/02-workflow-processing/complex-workflow-management-prd.md)说明创建、编辑和运行规则；[安全预览与验收场景](storybook/workflow/safe-workflow-review-and-qa-package.md)给出可复现的检查路径。
- **知识库与业务问数**：[知识库架构](docs/architecture/knowledge-base/)与[管理 PRD](docs/prd/03-knowledge-search/knowledge-management-prd.md)覆盖资料和业务表的维护；[Agentic RAG](docs/architecture/knowledge-base/agentic-rag-query.md)与[NL2SQL 语义层](docs/prd/03-knowledge-search/nl2sql-semantic-layer-prd.md)展开证据检索、业务口径和查询核验。
- **智能体与运行能力**：[智能体工作台 PRD](docs/prd/04-agent-applications/agent-workbench-prd.md)记录任务澄清、资源绑定、候选配置与发布流程；[产品全景](docs/product-overview.md)说明 Astra 与平台的衔接范围，Astra 的代码与实现请参阅[官方仓库](https://github.com/matrixorigin/Astra)。
- **场景与质量验证**：[Storybook](storybook/INDEX.md)用前置条件、操作、断言和失败证据组织业务场景；[评测与质量](docs/eval/README.md)及 [PoC 方案](docs/poc/README.md)提供验证方法与交付依据。

完整目录：[产品需求](docs/prd/) · [产品架构](docs/architecture/) · [产品研究](docs/research/) · [产品原型](product/moi-platform-prototype/) · [公开边界](DISCLAIMER.md)
