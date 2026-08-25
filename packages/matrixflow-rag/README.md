# Matrixflow RAG 源码快照

本目录保存与 RAG 架构文档对应的源码快照。文件保留原始目录层级，便于从文档直接跳转到实现与测试。

## 知识库与索引生命周期

- [知识库后端完整文件组](source/moi-backend/pkg/session/)
- [知识库 API Handler 与测试](source/moi-backend/pkg/handlers/session/)
- [知识库接口、来源与分段版本](source/moi-backend/pkg/session/semantic_model_interface.go)
- [来源作业与 RAG ingest](source/moi-backend/pkg/session/semantic_model_kb_jobs.go)
- [分段物化、文本/图片向量与版本切换](source/moi-backend/pkg/session/semantic_model_segments.go)
- [知识库 API 与行为说明](source/moi-backend/pkg/handlers/session/semantic_model.md)

## 知识库产品界面

- [知识库前端完整模块](source/moi-frontend/modules/moi-knowledge/)
- [知识库服务接口](source/moi-frontend/modules/moi-knowledge/src/service/knowledge.ts)
- [Semantic Model 服务接口](source/moi-frontend/modules/moi-knowledge/src/service/semanticModel.ts)
- [知识库探索与问答页面](source/moi-frontend/modules/moi-knowledge/src/pages/knowledge-explore/)
- [知识库创建与编辑页面](source/moi-frontend/modules/moi-knowledge/src/pages/knowledge-edit/)

## Agent 知识工具

- [知识工具目录](source/moi-core/agent-tools/knowledge/)
- [工具请求、检索范围与命中结构](source/moi-core/agent-tools/knowledge/schema_core.go)
- [检索上下文与证据聚合](source/moi-core/agent-tools/knowledge/context.go)
- [工具调用与结果呈现](source/moi-core/agent-tools/knowledge/tools.go)
- [Agent 知识工具测试](source/moi-core/agent-tools/knowledge/tools_test.go)

## 文本与视觉检索

- [全文与向量混合召回、上下文扩展、表格/图片证据](source/moi-core/agent-tools/knowledge/service/rag_retrieval.go)
- [文本 RAG 检索测试](source/moi-core/agent-tools/knowledge/service/rag_retrieval_test.go)
- [视觉检索](source/moi-core/agent-tools/knowledge/service/visual_search.go)
- [视觉融合与约束重排](source/moi-core/agent-tools/knowledge/service/visual_search_ranking.go)
- [视觉检索测试](source/moi-core/agent-tools/knowledge/service/visual_search_test.go)

## RAG ingest 与多粒度索引

- [默认 RAG ingest 工作流](source/moi-core/workflows/rag-ingest-default-v1.yaml)
- [知识库索引组合 WorkItem](source/moi-core/workers/go-worker/pkg/workitems/user_composite.go)
- [文档/节/块多层索引 WorkItem](source/moi-core/workers/go-worker/pkg/workitems/retrieval_index_multilevel.go)
- [多层索引器与测试](source/moi-core/workers/go-worker/pkg/workitems/multilevel/)
- [WorkItem 注册](source/moi-core/workers/go-worker/pkg/worker/workitems.go)
- [WorkItem 行为说明](source/moi-core/docs/workflow/WORKITEMS.md)

## Rerank 服务

- [Rerank 完整目录](source/moi-core/rerank/)
- [服务说明](source/moi-core/rerank/README.md)
- [Python 服务入口](source/moi-core/rerank/app.py)
- [Go 客户端](source/moi-core/rerank/openai.go)
- [Go 客户端测试](source/moi-core/rerank/openai_test.go)

独立服务与客户端已经实现；当前快照没有发现主文本 `search_rag_chunks` 对该客户端的生产调用。视觉检索的 RRF 与约束重排是另一条已实现路径。

## 接入工作流与资源目录

- [RAG ingest 默认工作流](source/moi-core/workflows/rag-ingest-default-v1.yaml)
- [RAG ingest 工作流说明](source/moi-core/docs/workflow/RAG_INGEST.md)
- [知识资源目录与 Knowledge Explore Agent](source/moi-core/catalog/pkg/agentresource/)
- [Agent 运行时知识工具](source/moi-core/catalog/pkg/agentruntime/knowledge_agent_tools.go)
- [平台知识工具装配与过滤](source/moi-core/catalog/pkg/agents/)
- [知识库与 Semantic Model API](source/moi-core/catalog/pkg/api/handlers/)

## SDK、前端集成与质量验证

- [Go SDK](source/moi-core/go-sdk/semantic_model.go)、[Python SDK](source/moi-core/python-sdk/moi/semantic_model.py)、[Bun SDK](source/moi-core/bun-sdk/src/services/knowledge-base.ts)
- [共享知识库 API](source/moi-frontend/modules/shared-moi-api/src/knowledge/)
- [RAG 命中证据面板](source/moi-frontend/modules/shared-moi-components/src/ai-chat-message/components/rag-chunks-panel/)
- [知识来源选择组件](source/moi-frontend/modules/shared-moi-components/src/knowledge-source-select-modal/)
- [离线 RAGAS 评测](source/moi-core/evals/ragas_offline/)
- [后端知识库 API 测试](source/moi-backend/api-tester/tests/knowledge/)
- [端到端与集成测试](source/moi-core/tests/)

## Matrixflow 知识库文档

### 当前契约与使用说明

- [知识库 Workflow 治理接入手册](source/docs/handbooks/knowledge-base-workflow-governance.md)
- [Backend 与 Core 集成手册](source/docs/handbooks/moi-backend-moi-core-integration.md)
- [知识库 API 与行为说明](source/moi-backend/pkg/handlers/session/semantic_model.md)
- [内置 Skill 与工具清单](source/docs/moi-built-in-skills-and-tools.md)
- [RAG ingest 工作流说明](source/moi-core/docs/workflow/RAG_INGEST.md)
- [WorkItem 行为说明](source/moi-core/docs/workflow/WORKITEMS.md)
- [知识库前端说明与设计](source/moi-frontend/modules/moi-knowledge/)

### API、SDK 与 Agent 运行时

- [Catalog API](source/moi-core/docs/api/catalog-api.md)
- [Go SDK API](source/moi-core/docs/api/go-sdk-api.md)
- [Python SDK API](source/moi-core/docs/api/python-sdk-api.md)
- [SDK 使用指南](source/moi-core/docs/guide/SDK_GUIDE.md)
- [Agent A2A API](source/moi-core/docs/api/agent-a2a-api.md)

该快照覆盖知识库前后端、接入工作流、多粒度索引、资源目录、Agent 知识工具、文本与视觉检索、Rerank、SDK、证据展示、评测及相关测试。
