# Case 01: Document Intelligence

## Case goal

Show how enterprise documents can be parsed, converted into structured information, retrieved with citations, and improved through an evaluation-driven product loop.

## RAG 调研资料

[`rag/`](rag/) 收录检索增强生成相关的原始调研稿，正文保持原样，仅统一文件名与资源路径。

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

## Demo boundary

The runnable demo belongs in `apps/document-intelligence-demo/` and must use only synthetic or publicly licensed documents.
