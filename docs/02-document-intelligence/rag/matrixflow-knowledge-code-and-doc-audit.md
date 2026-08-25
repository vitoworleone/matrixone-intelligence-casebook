# Matrixflow 知识库代码与文档核验

## 1. 核验结论

这次补查确认，Matrixflow 的知识库实现比原能力表列出的更完整，最重要的新增事实是：**默认 RAG ingest 已实现并启用 `doc / section / chunk` 三层索引；检索利用 doc 层定位文件、chunk 层召回，文件级表格命中还会利用 section 范围扩展证据。**

但三项容易混淆的能力需要分开表述：

| 能力 | 代码结论 | 当前状态 |
| --- | --- | --- |
| 文档/节/块多层索引 | 有 WorkItem、默认工作流、检索调用和测试。 | 已实现 |
| 独立 Cross-encoder Rerank | 有 `/v1/rerank` 服务、Go 客户端、配置项和测试。 | 已实现 |
| 主文本 RAG 使用 Cross-encoder | `search_rag_chunks` 没有装配 reranker，未找到客户端生产调用方。 | 未发现接入 |
| 视觉融合与约束重排 | 有 RRF、对象优先与文本/表格区域优先排序。 | 已实现 |
| GraphRAG | 未找到实体关系抽取、图存储/查询、社区发现或社区报告链路。 | 未发现 |
| RAPTOR | 有确定性的三层索引，但没有递归聚类、LLM 摘要节点和树形递归检索。 | 未发现 RAPTOR 实现 |

因此，原先把全部 Rerank 合并为“部分具备”不够准确；应该拆成“服务已实现、视觉重排已实现、主文本 cross-encoder 未接入”。原先遗漏的多层索引应补为“已实现”，但不能直接把它命名为 RAPTOR。

## 2. 审阅范围

- 源码快照：Matrixflow `d4b7995fabb906cf2c492a9d27ac0680e60fbee6`；
- 最新分支复核：2026-08-25 的 `upstream/dev`，提交 `3f01f9516492a9da4f911a44ab4ff7aec7744663`；
- 历史参考：包含旧 Explore/RAG 代码的 `origin/ragflow`，以及迁移提交 `d35da4cfc0b5159c0ce2b5e9453dd4d4195c8040`；
- 证据类型：生产代码、默认工作流、配置、测试、当前接口文档、运行分析和验收资料。

最新分支复核用于防止本地快照遗漏近期实现；它没有改变下文关于 Cross-encoder 主链路、GraphRAG 与 RAPTOR 的结论。

## 3. 当前知识库生产链路

| 环节 | 已确认实现 | 主要证据 |
| --- | --- | --- |
| 知识库与来源生命周期 | 知识库、文件/表来源、来源关系、source job、当前分段版本与治理状态。 | [后端 Session](../../../packages/matrixflow-rag/source/moi-backend/pkg/session/)、[API 说明](../../../packages/matrixflow-rag/source/moi-backend/pkg/handlers/session/semantic_model.md) |
| RAG ingest | 默认工作流完成读取、解析、切分、多层索引、embedding 与向量写入。 | [默认工作流](../../../packages/matrixflow-rag/source/moi-core/workflows/rag-ingest-default-v1.yaml)、[工作流说明](../../../packages/matrixflow-rag/source/moi-core/docs/workflow/RAG_INGEST.md) |
| 多层索引 | 生成 doc、section、chunk 三层条目；默认开启，支持 section 大小和文档摘要字符上限。 | [组合 WorkItem](../../../packages/matrixflow-rag/source/moi-core/workers/go-worker/pkg/workitems/user_composite.go)、[多层 WorkItem](../../../packages/matrixflow-rag/source/moi-core/workers/go-worker/pkg/workitems/retrieval_index_multilevel.go)、[索引器](../../../packages/matrixflow-rag/source/moi-core/workers/go-worker/pkg/workitems/multilevel/indexer.go) |
| 文件定位 | `find_rag_files` 只读取当前版本、启用状态的 `level = 'doc'` 条目。 | [文本检索](../../../packages/matrixflow-rag/source/moi-core/agent-tools/knowledge/service/rag_retrieval.go) |
| 分段召回 | `search_rag_chunks` 对 `level = 'chunk'` 执行全文与 `vector_l2` 两路召回。 | [文本检索](../../../packages/matrixflow-rag/source/moi-core/agent-tools/knowledge/service/rag_retrieval.go) |
| 证据扩展 | 文件级表格命中按 section 补齐 chunk；一般文本按相邻 parent 范围扩展，并聚焦表格行、嵌入图片和邻近图片。 | [文本检索](../../../packages/matrixflow-rag/source/moi-core/agent-tools/knowledge/service/rag_retrieval.go) |
| 视觉检索 | 文本/图片候选 RRF 融合，并按视觉对象或文本/表格区域约束重排。 | [视觉检索](../../../packages/matrixflow-rag/source/moi-core/agent-tools/knowledge/service/visual_search.go)、[视觉排序](../../../packages/matrixflow-rag/source/moi-core/agent-tools/knowledge/service/visual_search_ranking.go) |
| 结构化表 | `describe_schema` 与 `query_sql` 在选定表范围内完成 schema 检查和只读 SQL。 | [Schema 查询](../../../packages/matrixflow-rag/source/moi-core/agent-tools/knowledge/service/describe_schema.go)、[SQL 查询](../../../packages/matrixflow-rag/source/moi-core/agent-tools/knowledge/service/query_sql.go) |
| Agent 使用知识 | 默认 Data Agent 绑定 RAG、视觉、解析 Markdown、SQL 与最终证据选择工具。 | [Agent 配置](../../../packages/matrixflow-rag/source/moi-core/catalog/pkg/agentresource/systemagents/knowledge-explore/agent.json)、[中文提示词](../../../packages/matrixflow-rag/source/moi-core/catalog/pkg/agentresource/systemagents/knowledge-explore/system_prompt.zh-CN.md) |

## 4. Rerank、GraphRAG 与 RAPTOR

### 4.1 Rerank：服务已实现，主文本链路未接入

独立 Rerank 的实现证据完整：

- [Python 服务](../../../packages/matrixflow-rag/source/moi-core/rerank/app.py)提供 `/v1/rerank`；
- [Go 客户端](../../../packages/matrixflow-rag/source/moi-core/rerank/openai.go)负责调用兼容接口；
- [客户端测试](../../../packages/matrixflow-rag/source/moi-core/rerank/openai_test.go)覆盖请求与响应；
- [Catalog 配置](../../../packages/matrixflow-rag/source/moi-core/catalog/pkg/config/config.go)定义 endpoint、API key 和 model；
- [模型资源配置](../../../packages/matrixflow-rag/source/moi-core/catalog/pkg/agentresource/model_config.go)包含 rerank 模型类别。

主文本检索的静态调用链则是另一回事。`NewSearchRAGChunks` 当前只接收 SQL executor、embedder 和默认 embedding model；仓库中没有找到 `NewOpenAIReranker` 的生产调用，构造函数调用只出现在客户端测试。因此不能写成“文本 RAG 已使用 cross-encoder 二阶段重排”。

### 4.2 GraphRAG：没有找到生产实现

对当前源码和当日最新 `upstream/dev` 搜索 GraphRAG、知识图谱、实体/关系、社区与图查询相关实现，只找到前端 mock 资源中的能力描述，没有找到：

- 实体与关系抽取流水线；
- 图数据持久化模型；
- 图遍历或图查询工具；
- 社区发现、社区摘要或社区报告；
- 将图结果送入 Agent 回答的生产调用链与测试。

mock 描述、产品设想和调研资料都不能作为已交付证据。

### 4.3 RAPTOR：存在相邻能力，但不是 RAPTOR

当前多层索引会：

1. 拼接开头 chunk，生成有长度上限的 doc 内容；
2. 按连续 chunk 数量与字节预算形成 section；
3. 保留原始 chunk，并写入 doc/section/版本元数据；
4. 召回 chunk 后，文件级表格命中可按所属 section 扩展证据；一般文本按相邻 parent 范围扩展。

这已经是可用的层级/多粒度索引，但实现中没有递归向量聚类、模型生成摘要、摘要再聚类或递归树检索。历史 `Explore` 优化文档曾把 RAPTOR 列为长文档检索代表方案/规划项，也不是生产实现证据。

## 5. 本次补齐的源码快照

原源码包已覆盖知识库前后端、Agent 知识工具、文本/视觉检索与 Rerank，但遗漏了下列直接证据；本次已经补入：

| 类别 | 新增内容 |
| --- | --- |
| 多层索引实现 | `moi-core/workers/go-worker/pkg/workitems/multilevel/`、`retrieval_index_multilevel.go`、`user_composite.go`、注册与 schema 文件及测试。 |
| Rerank 配置 | Catalog 的 retriever/rerank 配置和模型类别定义。 |
| 当前知识库文档 | Workflow 治理、WorkItem、API/SDK、前端知识库设计和内置工具清单。 |
| 运行证据 | Semantic Model 慢查询分析、知识库流水线可靠性扫描、同库表克隆验收。 |
| 可执行验收 | `skills/kb-product-matrix/`、运行脚本和最近一次通过报告。 |
| Agent 设计边界 | A2A、Runtime trace 与原型资源路线图；均按“设计/规划资料”标注。 |

完整入口见 [Matrixflow RAG 源码快照](../../../packages/matrixflow-rag/)。

## 6. Matrixflow 知识库文档索引

### 6.1 当前契约与实现说明

- [知识库 Workflow 治理接入手册](../../../packages/matrixflow-rag/source/docs/handbooks/knowledge-base-workflow-governance.md)
- [Backend 与 Core 集成手册](../../../packages/matrixflow-rag/source/docs/handbooks/moi-backend-moi-core-integration.md)
- [知识库 API 与行为说明](../../../packages/matrixflow-rag/source/moi-backend/pkg/handlers/session/semantic_model.md)
- [内置 Skill 与工具清单](../../../packages/matrixflow-rag/source/docs/moi-built-in-skills-and-tools.md)
- [RAG ingest 工作流](../../../packages/matrixflow-rag/source/moi-core/docs/workflow/RAG_INGEST.md)
- [WorkItem 行为说明](../../../packages/matrixflow-rag/source/moi-core/docs/workflow/WORKITEMS.md)
- [多模态解析语义](../../../packages/matrixflow-rag/source/moi-core/docs/workers/parse-v3-image-semantics.md)

### 6.2 产品与前端设计

- [知识库前端 README](../../../packages/matrixflow-rag/source/moi-frontend/modules/moi-knowledge/README.md)
- [知识库产品设计](../../../packages/matrixflow-rag/source/moi-frontend/modules/moi-knowledge/docs/design-knowledge.md)
- [知识探索设计](../../../packages/matrixflow-rag/source/moi-frontend/modules/moi-knowledge/docs/design-explore.md)
- [后端契约驱动设计](../../../packages/matrixflow-rag/source/moi-frontend/modules/moi-knowledge/docs/design-by-backend.md)

### 6.3 API 与 SDK

- [Catalog API](../../../packages/matrixflow-rag/source/moi-core/docs/api/catalog-api.md)
- [Go SDK API](../../../packages/matrixflow-rag/source/moi-core/docs/api/go-sdk-api.md)
- [Python SDK API](../../../packages/matrixflow-rag/source/moi-core/docs/api/python-sdk-api.md)
- [SDK 使用指南](../../../packages/matrixflow-rag/source/moi-core/docs/guide/SDK_GUIDE.md)
- [Agent A2A API](../../../packages/matrixflow-rag/source/moi-core/docs/api/agent-a2a-api.md)

### 6.4 运行分析、验收与回归

- [Semantic Model 慢查询分析](../../../packages/matrixflow-rag/source/docs/analysis/20260706-prod-semantic-models-slow-query.md)
- [知识库流水线可靠性扫描](../../../packages/matrixflow-rag/source/docs/analysis/20260707-kb-pipeline-reliability-scan.md)
- [同库 Catalog 表克隆验收](../../../packages/matrixflow-rag/source/dev/docs/doing/2026-07-07-kb-catalog-table-clone-same-db-acceptance.md)
- [知识库产品矩阵 Skill](../../../packages/matrixflow-rag/source/skills/kb-product-matrix/SKILL.md)
- [最近一次通过的矩阵报告](../../../packages/matrixflow-rag/source/skills/kb-product-matrix/testdata/last-passed-matrix-report.json)

### 6.5 设计与演进资料

- [Agent Runtime A2A 设计](../../../packages/matrixflow-rag/source/moi-core/docs/design/agent-runtime-a2a.md)
- [Agent Runtime v2 Trace](../../../packages/matrixflow-rag/source/moi-core/docs/design/agent-runtime-v2-trace.md)
- [Agent 原型资源路线图](../../../packages/matrixflow-rag/source/moi-core/docs/design/agent-prototype-resource-roadmap.md)

本组文档中既有已落地说明，也有设计和路线图。判断实现状态时仍应回到生产代码、默认配置和测试，不能只依据标题或目标描述。

## 7. 推荐对外表述

可以表述为：

> Matrixflow 已实现版本化知识库、文档/节/块多粒度索引、全文与向量混合召回、文件级表格的 section 证据扩展、一般文本的邻近证据扩展、表格与图片证据增强、视觉 RRF/约束重排、结构化表查询、Agent 知识工具与独立 cross-encoder rerank 服务。

需要附带边界：

> 当前主文本 `search_rag_chunks` 未发现接入独立 cross-encoder；现有多粒度索引不是 RAPTOR；当前源码未发现 GraphRAG 生产链路。
