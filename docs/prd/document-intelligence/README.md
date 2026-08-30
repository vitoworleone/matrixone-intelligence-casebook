# Case 01: Document Intelligence & RAG Design

## Case goal

Show how complex documents can become searchable, traceable knowledge through parsing, structured extraction, RAG design, citations, and evaluation.

## Supporting research

- [RAG research](../../research/rag/)：平台能力、架构模式与选型判断。
- [RAG strategy and selection](../../research/rag/rag-strategy-and-selection.md)：是否投入 RAG、如何评估价值，以及如何选择实施路径。

## Evaluation research

- [Data quality and document-parsing evaluation](../../research/document-intelligence-evaluation/data-quality-and-parsing-evaluation.md)：微调数据集质量、数据选择方法与文档解析评测框架。

## Required artifacts

1. `product-brief.md`: users, scenario, outcome, and MVP boundary.
2. `user-journey.md`: before/after workflow and human handoff.
3. `prd.md`: functional requirements and non-functional constraints.
4. `solution-architecture.md`: ingestion, parsing, extraction, retrieval, generation, and audit path.
5. `parsing-and-extraction-design.md`: schema, confidence, exception handling, and review queue.
6. `rag-design.md`: chunking, retrieval, reranking, citations, permissions, and no-answer handling.
7. `prompt-version-history.md`: versioned prompts and why each changed.
8. `evaluation-plan.md`: Golden Set, rubric, judge/manual review, and release gate.
9. `badcase-review.md`: failure taxonomy, root cause, fix, and regression case.
10. `release-and-iteration.md`: monitoring, feedback loop, and next iteration.

## Implementation boundary

The runnable prototype and its engineering documentation will belong in [`../../../product/`](../../../product/) and must use only synthetic or publicly licensed documents.
