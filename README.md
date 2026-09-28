<p align="center">
  <img src="assets/architecture/casebook-banner.png" alt="MatrixOne Intelligence Casebook" width="900" />
</p>

# MatrixOne Intelligence

<p><strong>中文</strong> | <a href="README_EN.md">English</a></p>

<p>
  <a href="docs/architecture/canvasflow-workflow/"><img src="assets/badges/workflow-design.svg" alt="工作流设计资料" height="26" /></a>
  <a href="#-工作流交互演示"><img src="assets/badges/video-1080p.svg" alt="1080p 工作流视频" height="26" /></a>
  <a href="LICENSE"><img src="assets/badges/license-mit.svg" alt="MIT 许可证" height="26" /></a>
</p>

## 🧩 MOI 是什么

MatrixOne Intelligence（MOI）连接企业数据加工、知识管理与智能体任务。本案例集展示工作流、知识库、智能体及 Astra 运行衔接的交互原型、产品规则和验证场景。

## 🔄 工作流：数据加工与流程运行

工作流把解析、清洗、提取等处理步骤组织成可保存、可复用的流程。流程定义包含节点、参数及输入输出绑定；每次运行则产生作业状态、节点结果和日志，帮助用户检查产物或定位失败。

### 🎬 工作流交互演示

https://github.com/user-attachments/assets/22bb82d2-bb80-4fc4-8a91-1febaeabbd1e

> [!NOTE]
> 延伸阅读：[工作流设计资料](docs/architecture/canvasflow-workflow/)　[MOI 工作流管理 PRD](docs/prd/02-workflow-processing/complex-workflow-management-prd.md)

## 📚 知识库：资料、业务表与语义

知识库管理文件和结构化业务表的来源、处理状态与可查询范围。文档需要保留解析内容、分段、索引版本和引用位置；业务语义则维护指标定义、字段含义与表关联。Agentic RAG 围绕问题组织文档证据，NL2SQL 按业务口径查询数据，让结果能够回到资料或 SQL 核对。

<a href="assets/screenshots/moi-platform/data-workbench-overview.png"><img src="assets/screenshots/moi-platform/data-workbench-overview-preview.png" alt="MOI 数据工作台概览" width="560" /></a>

> [!NOTE]
> 延伸阅读：[知识管理 PRD](docs/prd/03-knowledge-search/knowledge-management-prd.md)　[NL2SQL 语义层方案](docs/prd/03-knowledge-search/nl2sql-semantic-layer-prd.md)

## 🤖 智能体与 Astra：配置与执行

智能体把任务目标、知识库、Skill、工具及外部连接组合为可编辑的应用配置。用户可以检查候选配置和资源授权，运行时查看进度、工具结果与产物，并按版本保存和发布。

<a href="assets/screenshots/moi-platform/agent-workbench-home.png"><img src="assets/screenshots/moi-platform/agent-workbench-home-preview.png" alt="MOI 智能体工作台首页" width="560" /></a>

### 🎬 Astra 运行时演示

https://github.com/user-attachments/assets/c008be26-4320-413c-9ad6-100aefcfa728

> [!NOTE]
> 延伸阅读：[智能体工作台 PRD](docs/prd/04-agent-applications/agent-workbench-prd.md)　[Astra 运行时架构与设计](docs/architecture/astra-runtime/)

## 🧭 作品导览

<p align="center">
  <a href="docs/prd/02-workflow-processing/complex-workflow-management-prd.md"><img src="assets/cards/workflow-design-zh.svg" width="245" alt="工作流设计：创建、配置、运行排障"></a>
  <a href="docs/architecture/knowledge-base/"><img src="assets/cards/knowledge-base-zh.svg" width="245" alt="知识库设计：资料、检索、业务语义"></a>
  <a href="docs/product-overview.md#6-astra平台能力如何进入-agent-运行时"><img src="assets/cards/agent-astra-zh.svg" width="245" alt="智能体与 Astra 适配：资源绑定、授权、执行反馈"></a>
</p>
<p align="center">
  <a href="storybook/INDEX.md"><img src="assets/cards/storybook-zh.svg" width="245" alt="Storybook 场景：输入、断言、证据"></a>
  <a href="docs/product-overview.md#产品手册把概念操作和异常处理连起来"><img src="assets/cards/product-manual-zh.svg" width="245" alt="产品手册：配置、操作、常见问题"></a>
</p>

## <img src="assets/icons/folder.svg" width="32" height="28" align="absmiddle" alt=""> 仓库结构

> [!IMPORTANT]
> 这里按产品设计、交互原型和场景验收组织 MOI 的主要材料。下方目录树标明各类内容的位置与用途，方便按主题查阅。

```text
.
├── .github/                GitHub 模板与工作流配置预留目录
├── assets/                 封面、图标、架构图与截图
├── docs/                   产品设计与验证文档
│   ├── research/           产品、技术与场景研究
│   ├── prd/                各产品域的需求和交互规则
│   ├── architecture/       工作流、知识库与运行时架构
│   ├── poc/                业务场景验证方案
│   └── eval/               质量度量与评测口径
├── product/                MOI 可交互原型、视频与说明
├── storybook/              按产品域组织的场景验收 Case
└── scripts/                预览图生成与原型同步工具
```

## ⭐ Star History

<p align="center">
  <a href="https://www.star-history.com/?repos=vitoworleone%2Fmatrixone-intelligence-casebook&amp;type=date">
    <picture>
      <source media="(prefers-color-scheme: dark)" srcset="https://api.star-history.com/chart?repos=vitoworleone/matrixone-intelligence-casebook&amp;type=date&amp;theme=dark" />
      <source media="(prefers-color-scheme: light)" srcset="https://api.star-history.com/chart?repos=vitoworleone/matrixone-intelligence-casebook&amp;type=date" />
      <img src="https://api.star-history.com/chart?repos=vitoworleone/matrixone-intelligence-casebook&amp;type=date" alt="Star History" width="680" />
    </picture>
  </a>
</p>

## 📄 License

本项目采用 [MIT License](LICENSE)。

<p align="center"><a href="docs/">文档目录</a>　<a href="DISCLAIMER.md">公开边界</a></p>
