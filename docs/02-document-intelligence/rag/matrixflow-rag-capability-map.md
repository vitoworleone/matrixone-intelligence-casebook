# Matrixflow RAG 能力映射与边界

## 使用方式

本表把 [RAG 能力与平台调研](rag-research.md) 中出现的能力与 Matrixflow 当前代码事实分开标注：

审阅基准：2026-08-25，Matrixflow 源码提交 `d4b7995fabb906cf2c492a9d27ac0680e60fbee6`。下文所说的“当前”均指该版本。

- **已实现**：在本次审阅范围内能找到生产代码路径或接口语义；
- **部分具备**：有底层服务或相邻能力，但没有确认该能力已经接入目标主链路；
- **未确认**：需要端到端运行、配置或其他模块证据后才能下结论；
- **未发现**：未在本轮审阅的核心 RAG 生产路径中找到实现，不可作为现有能力表述。

“未发现”不等同于整个仓库绝对不存在同名实验或未来计划；它表示当前 casebook 不能把该项写成 Matrixflow 已交付能力。

## 1. RAG 基础能力

| 调研能力 | Matrixflow 对应实现 | 状态 | 表述边界 |
| --- | --- | --- | --- |
| 文件与表接入 | 本地文件、Catalog 文件、Catalog 表、来源选择；来源关系和作业持久化。 | 已实现 | 文件进入解析/分段索引，Catalog 表进入结构化 SQL 查询；两者不是同一种检索模式，也不等同于支持所有外部格式或 Web 站点同步。 |
| 结构化表查询 | 选定 Catalog 表后，通过 `describe_schema` 检查范围和字段，再由 `query_sql` 执行只读 SQL。 | 已实现 | 这是 SQL / NL2SQL 链路，不是把表数据 embedding 后纳入全文+向量 RAG。 |
| 文档解析后入库 | `rag_ingest` 来源作业与解析/向量行发布后创建当前分段版本。 | 已实现 | 解析质量、格式覆盖和 OCR 效果需单独评测。 |
| 文本分段管理 | 初始导入、编辑、新增、启停、删除、重嵌入、历史版本切换。 | 已实现 | 未确认有面向所有业务场景的用户可配置切片规则。 |
| 向量化 | 文本 embedding model 与 vector table 绑定；版本物化时写入向量行。 | 已实现 | 模型可选范围取决于运行环境配置。 |
| 图像向量化 | 可选图片向量表和图片 embedding model，单独管理兼容性与分段。 | 已实现 | 不代表每一种文档都必然生成图片向量。 |
| 全文检索 | `fulltext` 路由。 | 已实现 | 不是 BM25 或关键词权重开关的同义词。 |
| 向量检索 | `vector_l2` 路由。 | 已实现 | 代码当前明确的是 L2 距离路由。 |
| 混合检索 | 同时执行全文与向量路由，候选合并、去重、稳定排序。 | 已实现 | 当前不是可在 UI 任意调权的“关键词 + 向量权重”声明。 |
| 上下文扩展 | 命中锚点与前后分段会组成 evidence group。 | 已实现 | 不是调研中的所有父子块/摘要树方案。 |
| 文档内表格块证据 | 对解析文档中的表格块依据关键词聚焦相关表格行。 | 已实现 | 不等于 Catalog 结构化表查询，也未确认 RAGFlow 的“固定 12 行一块”策略。 |
| 图片证据 | 补齐嵌入图片和邻近图片引用；视觉检索有文本/图片融合。 | 已实现 | 回答端是否显示由 Agent/UI 链路决定。 |

## 2. 来源治理、可追溯性与运行控制

| 能力 | Matrixflow 对应实现 | 状态 | 表述边界 |
| --- | --- | --- | --- |
| 来源标签 | 文件来源支持 `tags`，检索命中会携带 source tags。 | 已实现 | 这是来源级人工治理标签，不是自动标签集分类。 |
| 来源启停 | source 与 chunk 均可启停；禁用分段不会写入新向量行或被当前检索召回。 | 已实现 | 生效前提是当前版本物化/治理同步成功。 |
| 有效期 | source 可配置 `expires_at` 与 `force_enabled_after_expiry`，有效状态会同步至向量行 `disabled`。 | 已实现 | 仅描述来源治理，不代表通用权限策略。 |
| 当前索引版本 | 检索以来源的当前 `index_version` 为范围，支持切回历史 committed version。 | 已实现 | 不能省略“当前版本”条件而把历史向量混入生产召回。 |
| 来源作业可观察性 | 可列出 source jobs，显式 reconcile 推进 pending/running/终态收口。 | 已实现 | 不是完整的全平台可观测性结论。 |
| 权限边界 | 知识库接口与 source 操作有对象/资源授权语义。 | 已实现 | 本文不替代 IAM 设计与安全审计。 |

## 3. Advanced RAG、视觉检索与 Rerank

| 调研能力 | Matrixflow 对应实现 | 状态 | 表述边界 |
| --- | --- | --- | --- |
| Rerank 服务 | 仓库提供独立 `/v1/rerank` 服务。 | 部分具备 | 未确认主文本 `SearchRAGChunks` 在生产请求中调用该服务。 |
| 视觉候选融合 | 文本/图片视觉命中按 reciprocal rank fusion 聚合。 | 已实现 | 这是视觉检索的评分路径。 |
| 视觉约束重排 | 面向对象或文本/表格区域的约束匹配、加分与重排。 | 已实现 | 不应称为通用 cross-encoder rerank。 |
| 查询改写 | 本轮未找到核心链路实现。 | 未发现 | 不要以“多轮优化”或“问题优化”作为已交付功能。 |
| HyDE / CoRAG | 本轮未找到核心链路实现。 | 未发现 | 调研中的方法不自动迁移到 Matrixflow。 |
| 自动关键词提取 | 本轮未找到分段级自动抽取与可编辑工作流。 | 未发现 | 检索中使用查询关键词，不等于自动关键词抽取。 |
| 自动问题提取 / 关联问题 | 本轮未找到分段级生成与召回匹配实现。 | 未发现 | 不要与 MaxKB/RAGFlow 的关联问题混为一谈。 |
| 页面排名 | 本轮未找到页面优先级参与最终评分的实现。 | 未发现 | 不能声明为可配置页面排名。 |
| 独立联网搜索 / Tavily | Agent 平台定义了独立 Tavily Web Search 工具与工作区配置入口。 | 部分具备 | 未确认该工具会自动加入知识库 RAG 的检索、融合与回答链路，不能等同于“知识库已具备联网深度检索”。 |

## 4. GraphRAG 与 Agentic RAG

| 调研能力 | Matrixflow 对应实现 | 状态 | 表述边界 |
| --- | --- | --- | --- |
| 知识图谱 / GraphRAG | 本轮未在知识库、检索或 Agent 工具生产路径中找到实体、关系、图查询、社区报告实现。 | 未发现 | 不要以“支持知识图谱检索”对外描述。 |
| RAPTOR | 本轮未找到递归聚类、摘要节点或多层树检索实现。 | 未发现 | 不要将邻近分段扩展写成 RAPTOR。 |
| 多跳检索 | Agent 可调用检索工具，但未确认自动问题分解、循环检索或多跳编排策略。 | 未确认 | 可以说“具备供 Agent 使用的检索工具”，不能说“已实现自动多跳 RAG”。 |
| 自反思 / 自评 | 本轮未找到 Agent 对检索/回答自动反思与重试闭环。 | 未发现 | 不要按 Agentic RAG 完整闭环表述。 |
| 事实核验 / Verifier | 本轮未找到 NLI、规则或检索式事实核验链路。 | 未发现 | 证据提供不等于已验证事实正确。 |
| 回答引文 | 默认 Data Agent 绑定 `select_final_sources`，采用 cite-then-write，并支持 `rag_chunk`、`visual_hit`、`sql_result` 三类证据。 | 部分具备 | 证据选择机制已存在；各 Agent/终端的最终展示样式及事实结论的硬性引文校验仍需端到端验证。 |

## 5. RAGFlow 分块模板的对应关系

| RAGFlow 调研模板 | Matrixflow 当前可对应部分 | 结论 |
| --- | --- | --- |
| 通用文档 | 文件来源、解析后分段、全文/向量检索。 | 有基础对应，但未确认其页面并行、HTML 表格或固定块规则。 |
| Q&A | 无直接的“一行问答块 + 问答专用召回”证据。 | 未发现专用模板。 |
| 表格 | Catalog 表接入、表格行聚焦与视觉表格区域处理。 | 有相关能力，但非相同模板/切块规则。 |
| 简历、书籍、法律、手册、论文、演示文稿 | 上游可有多种文档解析；未发现相应预设切片模板与专属召回策略。 | 不应宣称已支持 RAGFlow 同名模板。 |
| One（整篇不切） | 本轮未找到以整份文档作为单块、依赖 LLM 长上下文的模式。 | 未发现。 |
| Tag（闭集自动归类） | 来源标签与标签回传。 | 只有人工治理标签的部分对应。 |

## 6. 调研文档中不应混入 Matrixflow RAG 现状的内容

以下内容属于 `rag-research.md` 对外部平台或数据流程的调研，不能写成 Matrixflow 的已实现 RAG 功能：

- MaxKB 的 Web 站点递归同步、关联问题、应用编排配置；
- RAGFlow 的图谱、RAPTOR、场景切片模板及其集成式 Tavily 深度搜索；Matrixflow 虽有独立 Agent 联网搜索工具，但尚不能据此写成知识库 RAG 主链路已集成深度搜索；
- AnythingLLM 的产品介绍；
- DataFlow 的预训练/SFT/推理数据合成、Text2SQL 和数据质量流水线。

DataFlow 中“原始文件 → 分段 → 多跳 QA”一类流程可以是未来 RAG 数据准备或评测的参考，但不是 Matrixflow 当前线上检索链路的组成部分。

## 7. 对外表述建议

适合当前证据的表述：

> Matrixflow 已具备面向文档、结构化表和图片知识范围的数据接入：文档支持版本化分段、全文与向量混合召回及证据扩展，结构化表支持受限 SQL 查询，Agent 可调用检索工具并选择回答证据；来源状态、标签、有效期和索引版本参与可检索范围控制。

不适合当前证据的表述：

> Matrixflow 已完整覆盖 RAGFlow 的 GraphRAG、RAPTOR、自动问题生成、可配置关键词权重、预设行业分块模板和深度联网检索。

## 源码索引

| 能力 | 代码/文档位置 |
| --- | --- |
| 来源、分段、版本和治理接口 | [接口与版本](../../../packages/matrixflow-rag/source/moi-backend/pkg/session/semantic_model_interface.go)、[API 说明](../../../packages/matrixflow-rag/source/moi-backend/pkg/handlers/session/semantic_model.md) |
| 来源作业与 ingest 推进 | [来源作业](../../../packages/matrixflow-rag/source/moi-backend/pkg/session/semantic_model_kb_jobs.go) |
| embedding、文本/图片向量、版本物化 | [分段与向量](../../../packages/matrixflow-rag/source/moi-backend/pkg/session/semantic_model_segments.go) |
| 检索范围、工具请求与命中 schema | [工具结构](../../../packages/matrixflow-rag/source/moi-core/agent-tools/knowledge/schema_core.go) |
| 结构化表 schema 与只读 SQL | [Schema 查询](../../../packages/matrixflow-rag/source/moi-core/agent-tools/knowledge/service/describe_schema.go)、[SQL 查询](../../../packages/matrixflow-rag/source/moi-core/agent-tools/knowledge/service/query_sql.go) |
| SQL、文档与视觉工具的范围装配 | [工具过滤与提示词](../../../packages/matrixflow-rag/source/moi-core/catalog/pkg/agents/platform_knowledge_tool_filter.go)、[知识工具装配](../../../packages/matrixflow-rag/source/moi-core/catalog/pkg/agents/platform_knowledge_tools.go) |
| 主文本混合召回、表格/图片证据增强 | [文本 RAG 检索](../../../packages/matrixflow-rag/source/moi-core/agent-tools/knowledge/service/rag_retrieval.go) |
| 视觉融合与重排 | [视觉检索](../../../packages/matrixflow-rag/source/moi-core/agent-tools/knowledge/service/visual_search.go)、[视觉重排](../../../packages/matrixflow-rag/source/moi-core/agent-tools/knowledge/service/visual_search_ranking.go) |
| 回答证据类型与选择约束 | [证据 schema](../../../packages/matrixflow-rag/source/moi-core/agent-tools/knowledge/schemas.go)、[默认 Data Agent](../../../packages/matrixflow-rag/source/moi-core/catalog/pkg/agentresource/systemagents/knowledge-explore/agent.json)、[中文提示词](../../../packages/matrixflow-rag/source/moi-core/catalog/pkg/agentresource/systemagents/knowledge-explore/system_prompt.zh-CN.md) |
| 可部署 rerank 服务 | [Rerank 服务](../../../packages/matrixflow-rag/source/moi-core/rerank/) |
