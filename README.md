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

- [**数据工作台**](product/moi-platform-prototype/) — 汇集数据对象、工作流、计算资源与知识库的管理入口；点击图片查看原尺寸界面。

## 智能体与 Astra：配置与执行

智能体把任务目标、知识库、Skill、工具及外部连接组合为可编辑的应用配置。用户可以检查候选配置和资源授权，运行时查看进度、工具结果与产物，并按版本保存和发布。[智能体工作台 PRD](docs/prd/04-agent-applications/agent-workbench-prd.md)说明任务交互、资源绑定和成果管理。

<a href="assets/screenshots/moi-platform/agent-workbench-home.png"><img src="assets/screenshots/moi-platform/agent-workbench-home-preview.png" alt="MOI 智能体工作台首页" width="560" /></a>

- [**智能体工作台**](product/moi-platform-prototype/) — 从对话入口进入任务，并使用智能体与资源中心；点击图片查看原尺寸界面。

### Astra 运行时演示

https://github.com/user-attachments/assets/c008be26-4320-413c-9ad6-100aefcfa728

> [!NOTE]
> 延伸阅读：[Astra 运行时架构与设计](docs/architecture/astra-runtime/) · [MOI 产品全景](docs/product-overview.md)

## 我的工作与交付

工作流和知识库是我负责的产品设计重点；智能体与 Astra 部分侧重产品适配。场景验收、产品手册和远程 Demo 则把设计延伸到使用与验证。

| 方向 | 我做的工作与可查看的材料 |
| --- | --- |
| ![工作流](assets/icons/workflow.svg) **工作流设计** | 我梳理创建与编辑、解析能力配置、节点输入输出绑定，以及运行与异常定位；明确流程定义、发布版本和单次运行的关系。[工作流管理 PRD](docs/prd/02-workflow-processing/complex-workflow-management-prd.md) |
| ![知识库](assets/icons/knowledge.svg) **知识库设计** | 我组织文件与业务表的管理流程，展开 Agentic RAG、NL2SQL 和语义配置，并将资料范围、引用依据与答案偏差的维护入口纳入设计。[知识库架构](docs/architecture/knowledge-base/) |
| ![智能体](assets/icons/agent.svg) **智能体与 Astra 适配** | 我参与梳理知识库、Skill、工具和授权如何进入智能体任务，以及运行进度、异常和产物如何回到产品界面。[适配要求](docs/product-overview.md#6-astra平台能力如何进入-agent-运行时) |
| ![场景验收](assets/icons/storybook.svg) **Storybook 场景** | 我设计前置条件、固定输入、操作路径、结果断言、失败证据与清理要求，让关键任务有可审查的验收路径。[场景目录](storybook/INDEX.md) |
| ![手册与演示](assets/icons/delivery.svg) **产品手册与 Demo** | 我撰写使用指引并搭建远程 Demo，把配置、任务操作和排障串成完整路径；简历筛选等场景用于展示从资料输入到结果查看的过程。[工作说明](docs/product-overview.md) |

更多研究、PoC 与评测材料见[文档目录](docs/)。本页展示设计与验收材料，具体运行结论以相应记录为准。[公开边界](DISCLAIMER.md)
