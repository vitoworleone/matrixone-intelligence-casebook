<p align="center">
  <img src="assets/architecture/casebook-banner.png" alt="MatrixOne Intelligence Casebook" width="900" />
</p>

# MatrixOne Intelligence

<p><strong>中文</strong> | <a href="README_EN.md">English</a></p>

<p>
  <a href="docs/architecture/canvasflow-workflow/"><img src="https://img.shields.io/badge/Workflow-Design%20Docs-5945a3?style=flat-square" alt="工作流设计原始资料" /></a>
  <a href="#工作流交互演示"><img src="https://img.shields.io/badge/Video-1080p-1266b2?style=flat-square" alt="1080p 工作流视频" /></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-MIT-b89916?style=flat-square" alt="MIT 许可证" /></a>
</p>

> 企业文件与业务数据 → 工作流加工 → 知识与业务语义 → 智能体任务 → 可核验结果。

## MOI 是什么

MatrixOne Intelligence（MOI）面向企业智能体应用，连接数据处理、知识管理、业务语义和任务执行。企业的文件、表格与业务数据先被解析和组织，再成为智能体可以查询或调用的资源；用户能够沿着数据来源、处理过程和任务产物检查结果。

MOI 还包含数据接入、API 集成与平台治理等能力。本案例集沿着工作流、知识库、智能体及 Astra 运行衔接这条产品链路展开；下方的原型、设计文档和场景材料分别展示交互方式、产品规则与验证方法。

## 工作流：数据加工与流程运行

工作流把解析、清洗、提取等处理步骤组织成可保存、可复用的流程。流程定义包含节点、参数及输入输出绑定；每次运行则产生作业状态、节点结果和日志，帮助用户检查产物或定位失败。[MOI 工作流管理 PRD](docs/prd/02-workflow-processing/complex-workflow-management-prd.md)记录创建、编辑、运行与排障的产品规则。

### 工作流交互演示

https://github.com/user-attachments/assets/22bb82d2-bb80-4fc4-8a91-1febaeabbd1e

> [!NOTE]
> 延伸阅读：[工作流设计资料](docs/architecture/canvasflow-workflow/) · [MOI 工作流管理 PRD](docs/prd/02-workflow-processing/complex-workflow-management-prd.md)

## 知识库：资料、业务表与语义

知识库管理文件和结构化业务表的来源、处理状态与可查询范围。文档需要保留解析内容、分段、索引版本和引用位置；业务语义则维护指标定义、字段含义与表关联。Agentic RAG 围绕问题寻找和组织文档证据，NL2SQL 将自然语言问题转为受业务口径约束的查询，让结果能回到原始资料或 SQL 核对。[知识管理 PRD](docs/prd/03-knowledge-search/knowledge-management-prd.md)、[Agentic RAG 架构](docs/architecture/knowledge-base/agentic-rag-query.md)和 [NL2SQL 语义层方案](docs/prd/03-knowledge-search/nl2sql-semantic-layer-prd.md)分别展开这些设计。

<a href="assets/screenshots/moi-platform/data-workbench-overview.png"><img src="assets/screenshots/moi-platform/data-workbench-overview-preview.png" alt="MOI 数据工作台概览" width="560" /></a>

- [**数据工作台**](product/moi-platform-prototype/) — 汇集数据对象、工作流、计算资源与知识库的管理入口；点击图片查看原尺寸界面。

## 智能体与 Astra：配置与执行

智能体把任务目标、知识库、Skill、工具及外部连接组合为可编辑的应用配置。用户可以检查候选配置和资源授权，运行时查看进度、工具结果与产物，并按版本保存和发布。[智能体工作台 PRD](docs/prd/04-agent-applications/agent-workbench-prd.md)说明任务交互、资源绑定和成果管理。

<a href="assets/screenshots/moi-platform/agent-workbench-home.png"><img src="assets/screenshots/moi-platform/agent-workbench-home-preview.png" alt="MOI 智能体工作台首页" width="560" /></a>

- [**智能体工作台**](product/moi-platform-prototype/) — 从对话入口进入任务，并使用智能体与资源中心；点击图片查看原尺寸界面。

### Astra 运行时演示

https://github.com/user-attachments/assets/c008be26-4320-413c-9ad6-100aefcfa728

> [!NOTE]
> 延伸阅读：[Astra 运行时架构与设计](docs/architecture/astra-runtime/) · [MOI 产品全景](docs/product-overview.md)

## 如何验证与阅读材料

[Storybook](storybook/INDEX.md)将用户任务写成可审查的场景：明确前置条件、固定输入、操作路径、结果断言、失败证据和清理要求。[评测与质量](docs/eval/README.md)关注答案与处理结果的质量口径，[PoC 方案](docs/poc/README.md)则把能力组合到具体业务问题中。PRD 与架构说明设计要求，原型展示页面和操作，Storybook 定义验收方式；具体运行结论以相应记录为准。

| 仓库入口 | 主要内容 |
| --- | --- |
| [产品需求](docs/prd/) | 数据接入、工作流、知识检索、智能体、API 与治理等产品规则 |
| [产品架构](docs/architecture/) | 能力边界、知识库架构、工作流设计和 Astra 运行时设计 |
| [交互原型](product/moi-platform-prototype/) | MOI 各工作台与管理页面的交互展示 |
| [场景验收](storybook/) | 按产品域组织的 Storybook 合同与检查路径 |
| [研究](docs/research/) · [PoC](docs/poc/) · [评测](docs/eval/) | 研究材料、业务验证方案和质量方法 |

[阅读产品全景](docs/product-overview.md) · [查看公开边界](DISCLAIMER.md)
