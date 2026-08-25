# AI-Ready Data Capability Map

## Capability layers

| Layer | Core capabilities | Evidence expected in this casebook |
| --- | --- | --- |
| Ingest and understand | Connectors, parsing, OCR/VLM handling, schema extraction, metadata enrichment | Parsing design, extraction schema, exception queue |
| Govern and prepare | Deduplication, quality checks, classification, permissions, lineage, versioning | Data contracts, governance rules, audit path |
| Retrieve and reason | Chunking, embeddings, hybrid retrieval, reranking, citations, no-answer handling | RAG design, retrieval diagnostics, cited answers |
| Execute and orchestrate | Tool contracts, workflow state, retries, approvals, human handoff | Agent and workflow design, failure recovery |
| Evaluate and improve | Golden Set, rubric, human review, badcase taxonomy, regression checks | Evaluation plan, scorecards, badcase review |

## Non-negotiable design rules

- Every transformation has a defined input, output, and owner.
- Every AI answer or action can point to evidence, a tool result, or an explicit uncertainty state.
- Every automation has a human escalation path when confidence, permission, or impact demands it.
- Quality is measured at the task level, not inferred from a model benchmark alone.

## How this maps to the cases

`Document Intelligence` proves the first three layers and the evaluation loop. `Agentic Data Workflow` proves controlled execution. `Solution PoC` proves that the capabilities can be scoped, accepted, and rolled out in an enterprise setting.
