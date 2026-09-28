<p align="center">
  <img src="assets/architecture/casebook-banner.png" alt="MatrixOne Intelligence Casebook" width="900" />
</p>

# MatrixOne Intelligence

<p><a href="README.md">中文</a> | <strong>English</strong></p>

<p>
  <a href="docs/architecture/canvasflow-workflow/"><img src="assets/badges/workflow-design.svg" alt="Workflow design documents" height="26" /></a>
  <a href="#workflow-interaction-demo"><img src="assets/badges/video-1080p.svg" alt="1080p workflow video" height="26" /></a>
  <a href="LICENSE"><img src="assets/badges/license-mit.svg" alt="MIT license" height="26" /></a>
</p>

## What is MOI?

MatrixOne Intelligence (MOI) connects enterprise data processing, knowledge management, and agent tasks. This casebook presents interaction prototypes, product rules, and validation scenarios for workflows, knowledge bases, agents, and Astra runtime integration.

## Workflows: processing data and running flows

Workflows organize parsing, cleaning, extraction, and other processing steps into definitions that can be saved and reused. A definition specifies nodes, parameters, and input/output bindings; each run produces job status, node results, and logs so users can inspect outputs or locate failures.

### Workflow interaction demo

https://github.com/user-attachments/assets/22bb82d2-bb80-4fc4-8a91-1febaeabbd1e

> [!NOTE]
> Read more: [Workflow design documents](docs/architecture/canvasflow-workflow/)　[MOI workflow PRD](docs/prd/02-workflow-processing/complex-workflow-management-prd.md)

## Knowledge bases: sources, tables, and semantics

Knowledge bases manage the sources, processing state, and query scope of documents and structured business tables. Documents retain parsed content, chunks, index versions, and citations; business semantics define metrics, field meanings, and table relationships. Agentic RAG organizes document evidence, while NL2SQL queries data using business definitions so results can be checked against source material or SQL.

<a href="assets/screenshots/moi-platform/data-workbench-overview.png"><img src="assets/screenshots/moi-platform/data-workbench-overview-preview.png" alt="MOI data workbench overview" width="560" /></a>

> [!NOTE]
> Read more: [Knowledge management PRD](docs/prd/03-knowledge-search/knowledge-management-prd.md)　[NL2SQL semantic layer](docs/prd/03-knowledge-search/nl2sql-semantic-layer-prd.md)

## Agents and Astra: configuration and execution

Agents combine a task goal, knowledge bases, skills, tools, and external connections in an editable application configuration. Users can inspect candidate settings and resource permissions, monitor progress and tool results, and save or publish versions.

<a href="assets/screenshots/moi-platform/agent-workbench-home.png"><img src="assets/screenshots/moi-platform/agent-workbench-home-preview.png" alt="MOI Agent workbench home" width="560" /></a>

### Astra runtime demo

https://github.com/user-attachments/assets/c008be26-4320-413c-9ad6-100aefcfa728

> [!NOTE]
> Read more: [Agent workbench PRD](docs/prd/04-agent-applications/agent-workbench-prd.md)　[Astra runtime architecture and design](docs/architecture/astra-runtime/)

## Explore the casebook

<p align="center">
  <a href="docs/prd/02-workflow-processing/complex-workflow-management-prd.md"><img src="assets/cards/workflow-design-en.svg" width="245" alt="Workflow design: create, configure, run"></a>
  <a href="docs/architecture/knowledge-base/"><img src="assets/cards/knowledge-base-en.svg" width="245" alt="Knowledge-base design: sources, retrieval, semantics"></a>
  <a href="docs/product-overview.md#6-astra平台能力如何进入-agent-运行时"><img src="assets/cards/agent-astra-en.svg" width="245" alt="Agent and Astra integration: resources, permissions, feedback"></a>
</p>
<p align="center">
  <a href="storybook/INDEX.md"><img src="assets/cards/storybook-en.svg" width="245" alt="Storybook scenarios: inputs, assertions, evidence"></a>
  <a href="docs/product-overview.md#产品手册把概念操作和异常处理连起来"><img src="assets/cards/product-manual-en.svg" width="245" alt="Product manual: configure, operate, troubleshoot"></a>
</p>

<p align="center"><a href="docs/">Documentation index</a>　<a href="DISCLAIMER.md">Publication scope</a></p>
