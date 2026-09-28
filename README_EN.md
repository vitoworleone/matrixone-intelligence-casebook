<p align="center">
  <img src="assets/architecture/casebook-banner.png" alt="MatrixOne Intelligence Casebook" width="900" />
</p>

# MatrixOne Intelligence

<p><a href="README.md">中文</a> | <strong>English</strong></p>

<p>
  <a href="docs/architecture/canvasflow-workflow/"><img src="https://img.shields.io/badge/Workflow-Design%20Docs-5945a3?style=flat-square" alt="Original workflow design documents" /></a>
  <a href="#workflow-interaction-demo"><img src="https://img.shields.io/badge/Video-1080p-1266b2?style=flat-square" alt="1080p workflow video" /></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-MIT-b89916?style=flat-square" alt="MIT license" /></a>
</p>

> From enterprise data to verifiable agent tasks: process source material, organize knowledge, call capabilities, and inspect results.

This product design casebook centers on **workflows and knowledge bases**. It brings together requirements, architecture, prototypes, acceptance scenarios, and evaluation materials, alongside agent and Astra integration planning. The [product overview and work scope](docs/product-overview.md) explain how the pieces fit together and the scope of participation.

## Demo videos

### Workflow interaction demo

The 1080p video shows goal entry, plan formation, and canvas node creation and edits.

https://github.com/user-attachments/assets/242fd58e-8d4c-488a-b59d-9a357e252049

> [!NOTE]
> **Explore the workflow design:** The [original archive](docs/architecture/canvasflow-workflow/) retains the interaction principles, architecture decisions, state machine, contracts, prompts, and observed runs in full. The [MOI workflow PRD](docs/prd/02-workflow-processing/complex-workflow-management-prd.md) defines the platform's creation, editing, and run behavior.

### Astra runtime demo

The video shows Astra managing context, execution, and run analysis for a long task in the CLI.

https://github.com/user-attachments/assets/c008be26-4320-413c-9ad6-100aefcfa728

> [!NOTE]
> This video comes from the [official Astra repository](https://github.com/matrixorigin/Astra). The [Astra integration discussion](docs/product-overview.md) in this casebook focuses on how MOI platform configuration connects to runtime capabilities.

## Prototype interface previews

<a href="assets/screenshots/moi-platform/agent-workbench-home.png"><img src="assets/screenshots/moi-platform/agent-workbench-home.png" alt="MOI Agent workbench home" width="560" /></a>

- [**Agent workbench**](product/moi-platform-prototype/) — Start a conversation and enter the agent and resource centers from the home page. Click the image to view it at full resolution.

<a href="assets/screenshots/moi-platform/data-workbench-overview.png"><img src="assets/screenshots/moi-platform/data-workbench-overview.png" alt="MOI data workbench overview" width="560" /></a>

- [**Data workbench**](product/moi-platform-prototype/) — Review data objects, workflows, compute resources, and knowledge bases together, then open the relevant management flow. Click the image to view it at full resolution.

## Design documents and validation

Start with the [product overview](docs/product-overview.md), then follow the original design, product rules, and acceptance scenarios for the area you want to inspect:

- **Workflows and data processing:** The [original design archive](docs/architecture/canvasflow-workflow/) retains the full design process, contracts, prompts, and observations. The [workflow PRD](docs/prd/02-workflow-processing/complex-workflow-management-prd.md) defines creation, editing, and run behavior; the [safe preview scenario](storybook/workflow/safe-workflow-review-and-qa-package.md) provides a reproducible acceptance path.
- **Knowledge and business queries:** The [knowledge-base architecture](docs/architecture/knowledge-base/) and [management PRD](docs/prd/03-knowledge-search/knowledge-management-prd.md) cover files and business tables. [Agentic RAG](docs/architecture/knowledge-base/agentic-rag-query.md) and the [NL2SQL semantic layer](docs/prd/03-knowledge-search/nl2sql-semantic-layer-prd.md) detail evidence retrieval, business definitions, and query verification.
- **Agents and runtime:** The [agent workbench PRD](docs/prd/04-agent-applications/agent-workbench-prd.md) covers task clarification, resource binding, candidate configurations, and publishing. The [product overview](docs/product-overview.md) describes the scope of the Astra integration discussion; see the [official Astra repository](https://github.com/matrixorigin/Astra) for its code and implementation.
- **Scenarios and quality:** [Storybook](storybook/INDEX.md) organizes business scenarios through prerequisites, actions, assertions, and failure evidence. [Evaluation](docs/eval/README.md) and [PoC plans](docs/poc/README.md) provide validation methods and delivery criteria.

Full directory: [Product requirements](docs/prd/) · [Architecture](docs/architecture/) · [Research](docs/research/) · [Prototype](product/moi-platform-prototype/) · [Publication scope](DISCLAIMER.md)
