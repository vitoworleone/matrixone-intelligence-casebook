# Research-to-Product Decisions

This log makes the link between research and product design explicit. It is more useful to a reviewer than a vendor inventory because each entry states a decision, its trade-off, and the evidence required to validate it.

| Research insight | Product decision | Trade-off | Evidence to add |
| --- | --- | --- | --- |
| Enterprise data is heterogeneous and noisy. | Treat parsing, metadata, quality checks, and exception review as first-class product capabilities. | More setup than a direct upload-to-chat flow. | Field-level extraction results, confidence thresholds, review queue. |
| A model answer is only useful when a user can trust it. | Require citations, permission-aware retrieval, and an explicit no-answer policy. | Some questions return a safe refusal instead of a fluent answer. | Retrieval traces, citation completeness, no-answer test cases. |
| Generic agents are hard to control in business workflows. | Constrain agents with typed tools, observable state, retries, and approval gates. | Less open-ended autonomy. | Tool contracts, state machine, recovery scenarios. |
| Data quality and product quality are coupled. | Evaluate end-to-end tasks with Golden Sets, rubrics, badcases, and regression checks. | Evaluation takes ongoing maintenance. | Evaluation report and release gate. |
| Enterprise adoption depends on more than model quality. | Make security, deployment, cost, adoption, and acceptance criteria part of the PoC design. | A narrower initial scope. | PoC plan, acceptance rubric, risk register. |

## Decision rule

Do not add a capability because it is fashionable or available in a framework. Add it only when it strengthens a measurable user outcome, a reliable delivery path, or a reusable enterprise control.
