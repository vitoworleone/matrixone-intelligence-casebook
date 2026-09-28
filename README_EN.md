<p align="center">
  <img src="assets/architecture/casebook-banner.png" alt="MatrixOne Intelligence Casebook" width="900" />
</p>

# MatrixOne Intelligence Casebook

This casebook documents the product design behind MatrixOne Intelligence (MOI), from enterprise data to business agents: how files and business tables become usable knowledge, how agents use knowledge and tools to complete tasks, and how their results can be checked and reproduced. It brings together product requirements, architecture, interactive prototypes, scenario acceptance, and evaluation materials. The [product overview](docs/product-overview.md) explains how these pieces fit together.

My work focused on **workflow and knowledge-base product design**. I also participated in planning the connection between agents and Astra; Storybook scenarios, product manuals, and remote demos helped turn the designs into tasks that can be operated and verified.

<p><a href="README.md">中文</a> | <strong>English</strong></p>

<p>
  <a href="docs/architecture/canvasflow-workflow/"><img src="https://img.shields.io/badge/Workflow-Design%20Docs-5945a3?style=flat-square" alt="Original workflow design documents" /></a>
  <a href="#workflow-interaction-demo"><img src="https://img.shields.io/badge/Video-1080p-1266b2?style=flat-square" alt="1080p workflow video" /></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-MIT-b89916?style=flat-square" alt="MIT license" /></a>
</p>

## Demo videos

### Workflow interaction demo

The 1080p video follows one task from goal entry and plan formation to canvas node creation and edits. The [original workflow design archive](docs/architecture/canvasflow-workflow/) retains the interaction principles, architecture decisions, state machine, contracts, prompts, and observed runs in full. The [MOI workflow PRD](docs/prd/02-workflow-processing/complex-workflow-management-prd.md) covers the platform's product rules.

https://github.com/user-attachments/assets/242fd58e-8d4c-488a-b59d-9a357e252049

### Astra runtime demo

This video from the [official Astra repository](https://github.com/matrixorigin/Astra) shows a long task running in the CLI, including retained context, execution decisions, and run analysis. The [Astra integration discussion](docs/product-overview.md) in this casebook focuses on how MOI platform configuration connects to runtime capabilities.

https://github.com/user-attachments/assets/c008be26-4320-413c-9ad6-100aefcfa728

## Prototype interface previews

These are the original MOI prototype screenshots for the agent and data workbenches. Interactive pages and related engineering materials are in the [prototype directory](product/moi-platform-prototype/).

<img src="assets/screenshots/moi-platform/agent-workbench-home.png" alt="MOI Agent workbench home" width="100%" />

*Agent workbench: the home conversation entry point, agent access, and resource center.*

<br />

<img src="assets/screenshots/moi-platform/data-workbench-overview.png" alt="MOI data workbench overview" width="100%" />

*Data workbench: a unified entry point for data objects, workflows, compute resources, and knowledge bases.*

## Design documents and validation

Start with the [product overview](docs/product-overview.md), then follow the original design, product rules, and acceptance scenarios for the area you want to inspect:

- **Workflows and data processing:** The [original design archive](docs/architecture/canvasflow-workflow/) retains the full design process, contracts, prompts, and observations. The [workflow PRD](docs/prd/02-workflow-processing/complex-workflow-management-prd.md) defines creation, editing, and run behavior; the [safe preview scenario](storybook/workflow/safe-workflow-review-and-qa-package.md) provides a reproducible acceptance path.
- **Knowledge and business queries:** The [knowledge-base architecture](docs/architecture/knowledge-base/) and [management PRD](docs/prd/03-knowledge-search/knowledge-management-prd.md) cover files and business tables. [Agentic RAG](docs/architecture/knowledge-base/agentic-rag-query.md) and the [NL2SQL semantic layer](docs/prd/03-knowledge-search/nl2sql-semantic-layer-prd.md) detail evidence retrieval, business definitions, and query verification.
- **Agents and runtime:** The [agent workbench PRD](docs/prd/04-agent-applications/agent-workbench-prd.md) covers task clarification, resource binding, candidate configurations, and publishing. The [product overview](docs/product-overview.md) describes the scope of the Astra integration discussion; see the [official Astra repository](https://github.com/matrixorigin/Astra) for its code and implementation.
- **Scenarios and quality:** [Storybook](storybook/INDEX.md) organizes business scenarios through prerequisites, actions, assertions, and failure evidence. [Evaluation](docs/eval/README.md) and [PoC plans](docs/poc/README.md) provide validation methods and delivery criteria.

Full directory: [Product requirements](docs/prd/) · [Architecture](docs/architecture/) · [Research](docs/research/) · [Prototype](product/moi-platform-prototype/) · [Publication scope](DISCLAIMER.md)
