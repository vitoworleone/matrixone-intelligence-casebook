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

Workflows organize parsing, cleaning, extraction, and other processing steps into definitions that can be saved and reused. A definition specifies nodes, parameters, and input/output bindings; each run produces job status, node results, and logs so users can inspect outputs or locate failures. The [MOI workflow PRD](docs/prd/02-workflow-processing/complex-workflow-management-prd.md) covers creation, editing, execution, and troubleshooting.

### Workflow interaction demo

https://github.com/user-attachments/assets/22bb82d2-bb80-4fc4-8a91-1febaeabbd1e

> [!NOTE]
> Read more: [Workflow design documents](docs/architecture/canvasflow-workflow/) · [MOI workflow PRD](docs/prd/02-workflow-processing/complex-workflow-management-prd.md)

## Knowledge bases: sources, tables, and semantics

Knowledge bases manage the sources, processing state, and query scope of documents and structured business tables. Documents need traceable parsed content, chunks, index versions, and citations; business semantics define metrics, field meanings, and table relationships. Agentic RAG finds and organizes document evidence for a question, while NL2SQL turns a natural-language question into a query constrained by business definitions. Results should be checkable against source material or SQL. The [knowledge management PRD](docs/prd/03-knowledge-search/knowledge-management-prd.md), [Agentic RAG architecture](docs/architecture/knowledge-base/agentic-rag-query.md), and [NL2SQL semantic layer design](docs/prd/03-knowledge-search/nl2sql-semantic-layer-prd.md) develop these areas.

<a href="assets/screenshots/moi-platform/data-workbench-overview.png"><img src="assets/screenshots/moi-platform/data-workbench-overview-preview.png" alt="MOI data workbench overview" width="560" /></a>

- [**Data workbench**](product/moi-platform-prototype/) — Entry points for data objects, workflows, compute resources, and knowledge bases. Click the image for full resolution.

## Agents and Astra: configuration and execution

Agents combine a task goal, knowledge bases, skills, tools, and external connections in an editable application configuration. Users can inspect candidate settings and resource permissions, monitor progress and tool results, and save or publish versions. The [agent workbench PRD](docs/prd/04-agent-applications/agent-workbench-prd.md) covers task interaction, resource binding, and outputs.

<a href="assets/screenshots/moi-platform/agent-workbench-home.png"><img src="assets/screenshots/moi-platform/agent-workbench-home-preview.png" alt="MOI Agent workbench home" width="560" /></a>

- [**Agent workbench**](product/moi-platform-prototype/) — Start a task from the conversation entry point and access agents and resources. Click the image for full resolution.

### Astra runtime demo

https://github.com/user-attachments/assets/c008be26-4320-413c-9ad6-100aefcfa728

> [!NOTE]
> Read more: [Astra runtime architecture and design](docs/architecture/astra-runtime/) · [MOI product overview](docs/product-overview.md)

## Validation and repository guide

[Storybook](storybook/INDEX.md) turns user tasks into reviewable scenarios with prerequisites, fixed inputs, actions, assertions, failure evidence, and cleanup requirements. [Evaluation](docs/eval/README.md) defines quality measures for answers and processing results; [PoC plans](docs/poc/README.md) combine capabilities around specific business problems. PRDs and architecture describe design requirements, prototypes show pages and interactions, and Storybook defines acceptance checks; run conclusions depend on recorded evidence.

| Repository entry | Contents |
| --- | --- |
| [Product requirements](docs/prd/) | Product rules for data ingestion, workflows, knowledge retrieval, agents, APIs, and governance |
| [Architecture](docs/architecture/) | Capability boundaries, knowledge-base architecture, workflow design, and Astra runtime design |
| [Interactive prototype](product/moi-platform-prototype/) | Interactive MOI workbench and management-page previews |
| [Scenario acceptance](storybook/) | Storybook contracts and checks organized by product area |
| [Research](docs/research/) · [PoCs](docs/poc/) · [Evaluation](docs/eval/) | Research material, business validation plans, and quality methods |

[Read the product overview](docs/product-overview.md) · [Publication scope](DISCLAIMER.md)
