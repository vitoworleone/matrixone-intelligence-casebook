# 评测 Storybook

本目录收录模型、RAG、信息提取、文档解析和结构化数据能力的评测回归合同。

| Case | 评测对象 |
|---|---|
| [SB-EV-001](rag-question-answering-regression.md) | RAG 入库、检索、回答与拒答 |
| [SB-EV-002](knowledge-benchmark-execution.md) | 知识问答基准的构建与执行 |
| [SB-EV-003](natural-language-to-sql.md) | 自然语言到 SQL 的执行正确性 |
| [SB-EV-004](document-parsing-benchmark.md) | 多类型文档解析质量 |
| [SB-EV-005](structured-extraction-benchmark.md) | Schema 驱动的信息提取质量 |
| [SB-EV-006](grounded-evaluation-dataset-generation.md) | 有来源的训练与评测数据集生成 |
| [SB-EV-007](evaluation-evidence-audit.md) | 评测证据审计与可发布结论 |
| [SB-EV-008](qa-acceptance-package.md) | 需求到可执行验收包 |

评测指标是 Storybook 的补充证据，不替代 `required` Case 的逐项 Pass/Fail 断言；所有公开 Case 当前均为 `not run`。
