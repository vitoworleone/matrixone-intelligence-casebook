<p align="center">
  <img src="assets/architecture/casebook-banner.png" alt="MatrixOne Intelligence Casebook" width="900" />
</p>

# MatrixOne Intelligence

<p><strong>中文</strong> | <a href="README_EN.md">English</a></p>

<p>
  <a href="docs/architecture/canvasflow-workflow/"><img src="assets/badges/workflow-design.svg" alt="工作流设计资料" height="26" /></a>
  <a href="#工作流交互演示"><img src="assets/badges/video-1080p.svg" alt="1080p 工作流视频" height="26" /></a>
  <a href="LICENSE"><img src="assets/badges/license-mit.svg" alt="MIT 许可证" height="26" /></a>
</p>

## MOI 是什么

MatrixOne Intelligence（MOI）连接企业数据加工、知识管理与智能体任务。本案例集展示工作流、知识库、智能体及 Astra 运行衔接的交互原型、产品规则和验证场景。

## 工作流：数据加工与流程运行

工作流把解析、清洗、提取等处理步骤组织成可保存、可复用的流程。流程定义包含节点、参数及输入输出绑定；每次运行则产生作业状态、节点结果和日志，帮助用户检查产物或定位失败。[MOI 工作流管理 PRD](docs/prd/02-workflow-processing/complex-workflow-management-prd.md)记录创建、编辑、运行与排障的产品规则。

### 工作流交互演示

https://github.com/user-attachments/assets/22bb82d2-bb80-4fc4-8a91-1febaeabbd1e

> [!NOTE]
> 延伸阅读：[工作流设计资料](docs/architecture/canvasflow-workflow/) · [MOI 工作流管理 PRD](docs/prd/02-workflow-processing/complex-workflow-management-prd.md)

## 知识库：资料、业务表与语义

知识库管理文件和结构化业务表的来源、处理状态与可查询范围。文档需要保留解析内容、分段、索引版本和引用位置；业务语义则维护指标定义、字段含义与表关联。Agentic RAG 围绕问题寻找和组织文档证据，NL2SQL 将自然语言问题转为受业务口径约束的查询，让结果能回到原始资料或 SQL 核对。[知识管理 PRD](docs/prd/03-knowledge-search/knowledge-management-prd.md)、[Agentic RAG 架构](docs/architecture/knowledge-base/agentic-rag-query.md)和 [NL2SQL 语义层方案](docs/prd/03-knowledge-search/nl2sql-semantic-layer-prd.md)分别展开这些设计。

<a href="assets/screenshots/moi-platform/data-workbench-overview.png"><img src="assets/screenshots/moi-platform/data-workbench-overview-preview.png" alt="MOI 数据工作台概览" width="560" /></a>

## 智能体与 Astra：配置与执行

智能体把任务目标、知识库、Skill、工具及外部连接组合为可编辑的应用配置。用户可以检查候选配置和资源授权，运行时查看进度、工具结果与产物，并按版本保存和发布。[智能体工作台 PRD](docs/prd/04-agent-applications/agent-workbench-prd.md)说明任务交互、资源绑定和成果管理。

<a href="assets/screenshots/moi-platform/agent-workbench-home.png"><img src="assets/screenshots/moi-platform/agent-workbench-home-preview.png" alt="MOI 智能体工作台首页" width="560" /></a>

### Astra 运行时演示

https://github.com/user-attachments/assets/c008be26-4320-413c-9ad6-100aefcfa728

> [!NOTE]
> 延伸阅读：[Astra 运行时架构与设计](docs/architecture/astra-runtime/) · [MOI 产品全景](docs/product-overview.md)

## 作品导览

| 主题 | 可查看的内容 |
| --- | --- |
| ![工作流](assets/icons/workflow.svg) **工作流设计** | 流程创建、解析配置、节点数据绑定、运行与排障。[设计文档](docs/prd/02-workflow-processing/complex-workflow-management-prd.md) |
| ![知识库](assets/icons/knowledge.svg) **知识库设计** | 文件与业务表管理、Agentic RAG、语义配置和引用核验。[架构专题](docs/architecture/knowledge-base/) |
| ![智能体](assets/icons/agent.svg) **智能体与 Astra 适配** | 知识、Skill、工具和授权进入任务，以及运行反馈回到产品界面的设计。[产品衔接说明](docs/product-overview.md#6-astra平台能力如何进入-agent-运行时) |
| ![场景验收](assets/icons/storybook.svg) **Storybook 场景** | 固定输入、操作路径、结果断言、失败证据和清理要求。[场景目录](storybook/INDEX.md) |
| ![产品手册](assets/icons/product-manual.svg) **产品手册** | 工作流、知识库和智能体的概念、配置、操作与常见异常处理。[公开梳理](docs/product-overview.md) |

[文档目录](docs/) · [公开边界](DISCLAIMER.md)
