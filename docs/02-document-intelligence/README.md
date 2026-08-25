# Case 01: Document Intelligence

## Case goal

Show how enterprise documents can be parsed, converted into structured information, retrieved with citations, and improved through an evaluation-driven product loop.

## RAG 调研资料

- [Matrixflow RAG 架构与能力地图](rag/README.md)：基于当前代码的架构、能力边界与调研映射。
- [RAG 能力与平台调研](rag/rag-research.md)：平台能力、分块、召回、数据合成与文档解析的原始调研稿。
- [RAG 策略与选型](rag/rag-strategy-and-selection.md)：是否投入 RAG、如何评估价值，以及如何选择实施路径。

## 质量评估调研

- [数据质量与文档解析效果评测](evaluation/data-quality-and-parsing-evaluation.md)：微调数据集质量、数据选择方法与文档解析评测框架。

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
