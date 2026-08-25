# Matrixflow RAG 架构

## 目的与证据范围

本文描述 Matrixflow 当前与 RAG 直接相关的产品与技术链路，而不是通用 RAG 的理想架构，也不是对 [RAG 能力与平台调研](rag-research.md) 中外部产品功能的转述。对应实现已经整理到[源码快照](../../../packages/matrixflow-rag/)。

审阅基准：2026-08-25，Matrixflow 源码提交 `d4b7995fabb906cf2c492a9d27ac0680e60fbee6`。下文所说的“当前”均指该版本。

| 模块 | 在 RAG 链路中的职责 |
| --- | --- |
| [知识库生命周期](../../../packages/matrixflow-rag/source/moi-backend/pkg/session/) | 知识库、来源、来源作业、分段版本与向量绑定的生命周期。 |
| [Agent 知识工具](../../../packages/matrixflow-rag/source/moi-core/agent-tools/knowledge/) | Agent 可调用的文件定位、文本分段检索、视觉检索、检索上下文与结果呈现。 |
| [结构化表查询](../../../packages/matrixflow-rag/source/moi-core/agent-tools/knowledge/service/) | 在选定 Catalog 表范围内进行 schema 检查和只读 SQL 查询。 |
| [Agent 范围与工具装配](../../../packages/matrixflow-rag/source/moi-core/catalog/pkg/agents/) | 根据知识范围分别启用 SQL、文档 RAG、视觉检索和最终证据选择能力。 |
| [Rerank 服务](../../../packages/matrixflow-rag/source/moi-core/rerank/) | 可部署的独立 rerank 服务。 |
| [知识库产品入口](../../../packages/matrixflow-rag/source/moi-frontend/modules/moi-knowledge/) | 知识库创建、来源选择、分段与来源治理。 |
| 文档解析工作流 | Office 文档解析能力的上游服务；本文不把所有解析细节等同于 RAG 检索能力。 |

未进行端到端运行验证时，本文不推断模型实际回答质量、召回率或 UI 中每一个开关是否已开放。

## 1. 知识库接入与处理任务

### 1.1 数据来源

知识库以 semantic model 为边界，可接入：

- 本地上传的非结构化文件；
- Catalog 中已有的文件；
- Catalog 表；
- 按数据库表叶子或 Volume 文件叶子展开的 `source_selections`。

来源不是“上传即完成”。后端为来源持久化作业状态；文件需要完成绑定、解析/RAG ingest，表需要 clone 到知识库数据域。`source-jobs/reconcile` 会推进待处理作业，并且只在当前知识库的向量绑定中已经出现可发布的行时，发布新的分段版本。

两种来源在接入后进入不同的检索链路：文件进入解析、分段和文本/图片索引；Catalog 表作为可查询的结构化表，由 SQL 工具读取。表来源不会因为绑定到知识库就自动转换成文档 chunk 或进入全文+向量召回。

这一区分很重要：某次文件执行完成，不等于当前知识库已经可检索；可检索状态以来源的当前 `segment_version_id` / `index_version` 为准。

### 1.2 原始来源与知识库关系

删除知识库来源时，语义是解除“该知识库与来源”的关系：

- 文件来源不会删除原始 Catalog 文件，也不会物理删除可复用的文本/图片向量行；
- 表来源删除的是为知识库 clone 的表，不删除原始 Catalog 表；
- 已删除来源不会继续出现在检索范围内；重新添加时再创建或推进相应作业。

因此，RAG 的数据治理单位是“知识库内的来源关系”，而不是把所有底层资产当作一次性副本。

## 2. 分段、版本与多模态索引

### 2.1 分段版本是生产检索指针

对文件来源，系统支持：

1. 从已有向量行导入初始分段；
2. 编辑一个分段的文本、OCR 或图片描述；
3. 新增、启用/禁用、删除分段；
4. 用当前分段重新 embedding；
5. 把已经 committed 的历史分段版本切回生产检索。

前四类变更会基于当前版本物化新的 `segment_version_id` 与 `index_version`；切换历史版本只移动当前指针，不重新生成向量。这使修订、回滚和检索范围切换具有明确的版本语义。

### 2.2 文本与图片向量绑定

一个知识库的检索范围会携带文本向量与可选图片向量配置：

- `vector_table`、`embedding_model`：文本分段向量；
- `image_vector_table`、`image_embedding_model`：图片分段向量；
- embedding 维度、预处理版本、距离度量等元数据用于校验向量行能否复用。

文本与图片索引并非同一字段的不同展示。后端为它们维护独立表与兼容性校验，视觉检索再在结果层完成融合。

## 3. 结构化表查询链路

当 Agent 的知识范围包含 Catalog 表时，运行时会为结构化表启用两类工具：

- `describe_schema`：检查当前允许表、字段、可查询状态和语义信息；
- `query_sql`：在选定表范围内执行只读 `SELECT` / `WITH` 查询，并返回带 `artifact_id` 的结果。

这是 SQL / NL2SQL 查询链路，不是对表数据进行 embedding 后的向量 RAG。运行时会把 SQL 表范围与文档语料范围分开描述；当同一任务同时包含表和文件时，默认 Data Agent 的提示词要求分别查询 SQL 与文档证据，不能因 SQL 已返回结果就跳过文档检索。

## 4. 文本 RAG 检索链路

### 4.1 文档 RAG 主链路的两个核心工具

`moi-core/agent-tools/knowledge` 定义了两类核心工具：

- `find_rag_files`（Go 接口名 `FindRAGFiles`）：在已授权的知识库范围内按文件信息定位候选文件；
- `search_rag_chunks`（Go 接口名 `SearchRAGChunks`）：在范围、文件和版本过滤条件下搜索分段，并返回可作为上下文的证据组。

二者把知识库能力暴露给 Agent；它们不等同于一个固定的“问一句、直接生成一句”的聊天 API。Agent 可以结合任务上下文选择调用方式。

这里的“两个”只指文档 RAG 的来源定位与分段检索主链路，不是 Agent 的全部知识工具。默认 Data Agent 还绑定了 `search_visual_image`、`read_parsed_markdown`、`search_parsed_markdown`、`describe_schema`、`query_sql` 和 `select_final_sources` 等工具。

### 4.2 检索前的范围约束

检索 SQL 先约束：

- 只检索启用的行（`disabled = 0`）；
- 只使用来源当前生效的 `index_version`；
- 支持指定 Volume、文件 ID 与工作区范围；
- 将来源标签附回命中的文件和分段，供下游上下文与结果展示使用。

来源启用、有效期和强制启用规则会同步到文本/图片向量表的 `disabled` 标记，因此治理状态不是只存在于前端表单的元数据。

### 4.3 两路召回与合并

`search_rag_chunks` 对同一范围执行两条候选路由：

1. 全文路由（`fulltext`）；
2. 向量 L2 路由（`vector_l2`）。

返回的候选会合并、去重和排序。代码会另外计算查询词的字面匹配信号，用于候选之间的稳定排序；这不是调研中“可由用户配置关键词权重”的产品开关。响应会记录实际路由、全文/向量行数和 embedding model，便于调用方解释一次检索来自哪里。

### 4.4 从命中块到可读证据

主检索不止返回孤立命中块：

- 对每个候选生成上下文扩展计划，将命中块与前后分段组织为 `EvidenceGroup`；
- 如果证据是表格，会根据查询关键词优先保留相关表格行，而非无差别塞入整个表格；
- 对文本中嵌入的图片 UUID，会补齐对应图片分段；
- 还会在限定窗口内补齐邻近图片引用。

这样，回答所见的是“命中锚点 + 必要的上下文/多模态证据”，而不是只有单一向量行。

## 5. 视觉检索与视觉重排

视觉检索单独支持文本命中和图片命中。`fuseVisualResults` 在文档或视觉对象粒度聚合两路结果，使用 reciprocal-rank 形式的分数 `1 / (60 + rank)` 进行 RRF 融合。

还支持两类视觉后处理：

- `visual_object_first`：将工程约束与视觉对象文本进行匹配、加分和排序；
- `visual_text_region_first`：根据文本/表格区域的匹配数、同对象奖励及图片排名进行排序。

这是真实的“视觉候选融合 + 约束重排”。它不能直接等价为主文本 RAG 已配置 cross-encoder reranker；两个调用路径和评分语义不同。

## 6. Agent、上下文与回答

运行时可把知识工具的命中内容、来源标签、文件/图片引用、证据组和检索摘要交给 Agent。随后由 Agent/LLM 决定如何使用这些材料回答用户问题。

默认 Data Agent 绑定了 `select_final_sources`。运行时提示词要求采用 cite-then-write：完成检索后先选择最终答案实际使用的来源，再生成用户可见答案。当前支持的证据类型包括：

- `rag_chunk`：来自 `search_rag_chunks` 的文档分段；
- `visual_hit`：来自 `search_visual_image` 的视觉命中；
- `sql_result`：来自 `query_sql` 的结果制品。

因此，“Agent 具备选择并回传回答证据的机制”可以由代码确认。不过，不能据此扩展为以下尚未端到端确认的承诺：

- 每个 Agent 和终端都一定以相同样式显示规范化引文；
- 每条回答都由运行时硬性校验为“事实结论均有引用”；
- 所有 Agent 都会强制调用知识工具；
- 无命中时一定拒答，或一定走某个统一的 no-answer 策略；
- Agent 会自动反思、二次检索或进行事实核验。

这些行为需要在具体 Agent、提示词、产品配置和端到端测试中另行验证。

## 7. Rerank 的实际位置

仓库有可独立部署的 `/v1/rerank` 服务，默认配置示例使用 `BAAI/bge-reranker-base`，支持 Torch 与 ONNX 运行方式。与此同时，视觉检索中已有 RRF 融合及约束型重排。

本次审阅没有确认主 `search_rag_chunks` 链路调用该 `/v1/rerank` 服务。因此当前准确表述是：**具备可部署 rerank 能力，且视觉检索已有结果重排；主文本 RAG 的 cross-encoder 接入状态待运行链路验证。**

## 8. 与调研内容的边界

当前代码覆盖了 Native RAG 与部分 Advanced RAG：文档和结构化表接入、分段版本、向量化、文档全文+向量混合召回、结构化表 SQL 查询、元数据/来源治理、文档内表格与图片证据增强、Agent 工具调用和回答证据选择。

本次审阅未在核心生产路径中发现下列实现，不应使用“已支持”描述：

- RAPTOR 递归聚类；
- 知识图谱、实体归一化、社区报告或 GraphRAG；
- 自动关键词提取、自动问题提取、页面排名；
- RAGFlow 风格的简历/法律/书籍/手册等预设切片模板；
- 尚未确认把独立 Tavily/Web Search 工具自动编排进知识库 RAG 主检索链路；Matrixflow 存在独立的 Agent 联网搜索能力，但不能将其等同为知识库检索能力；
- 自反思、多跳重试、投票式答案生成或事实核验闭环。

详细的逐项对照见 [Matrixflow RAG 能力映射与边界](matrixflow-rag-capability-map.md)。

## 源码索引

| 主题 | 主要实现位置 |
| --- | --- |
| 知识库接口、来源与分段版本语义 | [接口与版本](../../../packages/matrixflow-rag/source/moi-backend/pkg/session/semantic_model_interface.go)、[API 说明](../../../packages/matrixflow-rag/source/moi-backend/pkg/handlers/session/semantic_model.md) |
| 来源作业及 RAG ingest | [来源作业](../../../packages/matrixflow-rag/source/moi-backend/pkg/session/semantic_model_kb_jobs.go) |
| 文本/图片向量与分段物化 | [分段与向量](../../../packages/matrixflow-rag/source/moi-backend/pkg/session/semantic_model_segments.go) |
| 知识工具契约与结果上下文 | [工具结构](../../../packages/matrixflow-rag/source/moi-core/agent-tools/knowledge/schema_core.go)、[上下文](../../../packages/matrixflow-rag/source/moi-core/agent-tools/knowledge/context.go)、[工具实现](../../../packages/matrixflow-rag/source/moi-core/agent-tools/knowledge/tools.go) |
| 结构化表 schema 与只读 SQL | [Schema 查询](../../../packages/matrixflow-rag/source/moi-core/agent-tools/knowledge/service/describe_schema.go)、[SQL 查询](../../../packages/matrixflow-rag/source/moi-core/agent-tools/knowledge/service/query_sql.go) |
| SQL、文档与视觉工具的范围装配 | [工具过滤与提示词](../../../packages/matrixflow-rag/source/moi-core/catalog/pkg/agents/platform_knowledge_tool_filter.go)、[知识工具装配](../../../packages/matrixflow-rag/source/moi-core/catalog/pkg/agents/platform_knowledge_tools.go) |
| 全文+向量检索、证据扩展、表格/图片处理 | [文本 RAG 检索](../../../packages/matrixflow-rag/source/moi-core/agent-tools/knowledge/service/rag_retrieval.go) |
| 视觉检索、融合与约束重排 | [视觉检索](../../../packages/matrixflow-rag/source/moi-core/agent-tools/knowledge/service/visual_search.go)、[视觉重排](../../../packages/matrixflow-rag/source/moi-core/agent-tools/knowledge/service/visual_search_ranking.go) |
| 回答证据类型与选择约束 | [证据 schema](../../../packages/matrixflow-rag/source/moi-core/agent-tools/knowledge/schemas.go)、[默认 Data Agent](../../../packages/matrixflow-rag/source/moi-core/catalog/pkg/agentresource/systemagents/knowledge-explore/agent.json)、[中文提示词](../../../packages/matrixflow-rag/source/moi-core/catalog/pkg/agentresource/systemagents/knowledge-explore/system_prompt.zh-CN.md) |
| 独立 rerank 服务 | [Rerank 服务](../../../packages/matrixflow-rag/source/moi-core/rerank/) |
