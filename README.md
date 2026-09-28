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

工作流把解析、清洗、提取等处理步骤组织成可保存、可复用的流程。流程定义包含节点、参数及输入输出绑定；每次运行则产生作业状态、节点结果和日志，帮助用户检查产物或定位失败。

### 工作流交互演示

https://github.com/user-attachments/assets/22bb82d2-bb80-4fc4-8a91-1febaeabbd1e

> [!NOTE]
> 延伸阅读：[工作流设计资料](docs/architecture/canvasflow-workflow/) · [MOI 工作流管理 PRD](docs/prd/02-workflow-processing/complex-workflow-management-prd.md)

## 知识库：资料、业务表与语义

知识库管理文件和结构化业务表的来源、处理状态与可查询范围。文档需要保留解析内容、分段、索引版本和引用位置；业务语义则维护指标定义、字段含义与表关联。Agentic RAG 围绕问题组织文档证据，NL2SQL 按业务口径查询数据，让结果能够回到资料或 SQL 核对。

<a href="assets/screenshots/moi-platform/data-workbench-overview.png"><img src="assets/screenshots/moi-platform/data-workbench-overview-preview.png" alt="MOI 数据工作台概览" width="560" /></a>

> [!NOTE]
> 延伸阅读：[知识管理 PRD](docs/prd/03-knowledge-search/knowledge-management-prd.md) · [NL2SQL 语义层方案](docs/prd/03-knowledge-search/nl2sql-semantic-layer-prd.md)

## 智能体与 Astra：配置与执行

智能体把任务目标、知识库、Skill、工具及外部连接组合为可编辑的应用配置。用户可以检查候选配置和资源授权，运行时查看进度、工具结果与产物，并按版本保存和发布。

<a href="assets/screenshots/moi-platform/agent-workbench-home.png"><img src="assets/screenshots/moi-platform/agent-workbench-home-preview.png" alt="MOI 智能体工作台首页" width="560" /></a>

### Astra 运行时演示

https://github.com/user-attachments/assets/c008be26-4320-413c-9ad6-100aefcfa728

> [!NOTE]
> 延伸阅读：[智能体工作台 PRD](docs/prd/04-agent-applications/agent-workbench-prd.md) · [Astra 运行时架构与设计](docs/architecture/astra-runtime/)

## 作品导览

| 主题 | 可查看的内容 |
| --- | --- |
| <a href="docs/prd/02-workflow-processing/complex-workflow-management-prd.md"><img src="assets/icons/workflow.svg" width="28" height="28" align="middle" alt=""> <strong>工作流设计</strong></a> | 流程创建、解析配置、节点数据绑定、运行与排障。 |
| <a href="docs/architecture/knowledge-base/"><img src="assets/icons/knowledge.svg" width="28" height="28" align="middle" alt=""> <strong>知识库设计</strong></a> | 文件与业务表管理、Agentic RAG、语义配置和引用核验。 |
| <a href="docs/product-overview.md#6-astra平台能力如何进入-agent-运行时"><img src="assets/icons/agent.svg" width="28" height="28" align="middle" alt=""> <strong>智能体与 Astra 适配</strong></a> | 知识、Skill、工具和授权进入任务，以及运行反馈回到产品界面的设计。 |
| <a href="storybook/INDEX.md"><img src="assets/icons/storybook.svg" width="28" height="28" align="middle" alt=""> <strong>Storybook 场景</strong></a> | 固定输入、操作路径、结果断言、失败证据和清理要求。 |
| <a href="docs/product-overview.md#产品手册把概念操作和异常处理连起来"><img src="assets/icons/product-manual.svg" width="28" height="28" align="middle" alt=""> <strong>产品手册</strong></a> | 工作流、知识库和智能体的概念、配置、操作与常见异常处理。 |

---

**更多资料** · [文档目录](docs/) · [公开边界](DISCLAIMER.md)
