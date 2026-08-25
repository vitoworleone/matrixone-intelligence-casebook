# MOI-Core 文档索引

## 指南 (guide/)

| 文档 | 说明 |
|------|------|
| [CONCEPTS.md](./guide/CONCEPTS.md) | 核心概念：工作流、任务、WorkItem、Worker、动态服务 |
| [DESIGN.md](./guide/DESIGN.md) | 架构设计：模块划分、交互方式、数据库表结构 |
| [DEVELOPMENT.md](./guide/DEVELOPMENT.md) | 开发指南：环境搭建、构建、运行、测试 |
| [DEPLOYMENT.md](./guide/DEPLOYMENT.md) | 部署指南：本地开发与 Kubernetes 部署 |
| [SDK_GUIDE.md](./guide/SDK_GUIDE.md) | SDK 开发指南：Go/Python SDK 使用教程 |
| [DSL.md](./guide/DSL.md) | DSL 工作流构建：Chain/Parallel/Xor/Or/Loop 等 |
| [AGENT_PACKAGE.md](./guide/AGENT_PACKAGE.md) | Agent package 构造、校验、构建、加载与自定义工具开发指南 |
| [moi-cli README](../cli/README.md) | CLI API/DIRECT DB 能力矩阵、Kubernetes 配置初始化、写操作保护与审计 runbook |

## 架构 (architecture/)

| 文档 | 说明 |
|------|------|
| [README.md](./architecture/README.md) | 架构文档入口索引：推荐阅读顺序与文档概览 |
| [SYSTEM_OVERVIEW.md](./architecture/SYSTEM_OVERVIEW.md) | 系统总览：所有组件的职责、交互关系、通信协议，内嵌模式与独立部署模式差异 |
| [DATA_FLOW.md](./architecture/DATA_FLOW.md) | 数据流架构：元数据管理、工作流执行、文件存储、动态服务调用、Worker 会话等数据流转路径 |
| [DEPLOYMENT_ARCH.md](./architecture/DEPLOYMENT_ARCH.md) | 部署架构：本地开发与生产环境部署拓扑、基础设施依赖、RuntimeProvider 模式、配置管理 |
| [SECURITY_ARCH.md](./architecture/SECURITY_ARCH.md) | 安全架构：API Key 认证、角色权限模型、多租户隔离、工作区级别数据隔离 |
| [COMPONENT_DESIGN.md](./architecture/COMPONENT_DESIGN.md) | 子系统详细设计：Catalog、Mowl、Saga、FileService、LLM/Embedding 代理的内部架构与设计 |

## 升级 (upgrade/)

| 文档 | 说明 |
|------|------|
| [auto_upgrade.md](./auto_upgrade.md) | 自动升级框架设计与实施计划 |

## 发布状态

| 文档 | 说明 |
|------|------|
| [pricing-snapshot-v2.md](./pricing-snapshot-v2.md) | Pricing Snapshot v2 当前延后发布，Catalog 不暴露任何可用入口 |

## 设计 (design/)

| 文档 | 说明 |
|------|------|
| [agentloop-memory.md](./design/agentloop-memory.md) | AgentLoop 分层记忆与压缩设计 |
| [agent-runtime-a2a.md](./design/agent-runtime-a2a.md) | Agent Runtime 门面、A2A 运行时协议、provider 抽象与 agent-runtime-v2 默认 backend 设计 |
| [agent-prototype-resource-roadmap.md](./design/agent-prototype-resource-roadmap.md) | 原型需要但不属于 Agent Runtime 的资源管理能力设计与落地计划 |
| [pluggable-agent-capability-architecture.md](./design/pluggable-agent-capability-architecture.md) | Pluggable Agent capability、Astra binding、Agent package 与运行时网关设计 |
| [dataconn-workerization-contract.md](./design/dataconn-workerization-contract.md) | Dataconn workerization 首阶段 validate/metadata 契约边界 |
| [java-sdk-worker-runtime.md](./design/java-sdk-worker-runtime.md) | Java SDK 通用 Mowl worker runtime 状态与范围 |
| [compute-resource-node-placement.md](./design/compute-resource-node-placement.md) | ComputeResource worker Kubernetes 节点调度配置 |
| [compute-resource-iam-create-saga.md](./design/compute-resource-iam-create-saga.md) | ComputeResource 创建与 IAM Ownership 的持久化 Saga 合同 |

## Explore

| 文档 | 说明 |
|------|------|
| [explore_design.md](./explore_design.md) | Explore 引擎设计概览与数据流 |
| [explore_optimizations.md](./explore_optimizations.md) | Explore 召回优化与后续能力总览 |
| [explore_architecture_convergence.md](./explore_architecture_convergence.md) | Explore 架构收敛方案（统一控制面与统一排序目标） |
| [explore_capability_contract_v1.md](./explore_capability_contract_v1.md) | Explore 通用能力对齐契约（Run/Event/Artifact/Control） |
| [explore_api_sse_zh.md](./explore_api_sse_zh.md) | Explore 接口与 SSE 协议说明（中文，run.v1） |
| [explore_api_sse_en.md](./explore_api_sse_en.md) | Explore API and SSE guide (English, run.v1) |

## 工作流

| 文档 | 说明 |
|------|------|
| [VOLUME_SOURCE_EXECUTION.md](./workflow/VOLUME_SOURCE_EXECUTION.md) | 整卷与指定文件选择的逐文件分发契约 |

## API 参考 (api/)

| 文档 | 说明 | 生成方式 |
|------|------|----------|
| [catalog-api.md](./api/catalog-api.md) | Catalog HTTP REST API（172 个端点） | 自动生成（swagger） |
| [agent-a2a-api.md](./api/agent-a2a-api.md) | 通用 Agent A2A HTTP API（Explore/Workflow，含 workflow ask-user 提交） | 手写 |
| [mowl-api.md](./api/mowl-api.md) | Mowl Engine gRPC API（3 服务，40 RPC） | 自动生成（proto） |
| [go-sdk-api.md](./api/go-sdk-api.md) | Go SDK API 参考 | 自动生成（源码） |
| [python-sdk-api.md](./api/python-sdk-api.md) | Python SDK API 参考 | 自动生成（源码） |
| [fileservice-api.md](./api/fileservice-api.md) | FileService API 参考 | 手写 |
| [model-api.md](./api/model-api.md) | Model（Protobuf）API 参考 | 手写 |
| [saga-api.md](./api/saga-api.md) | Saga 组件 API 参考 | 手写 |
| [llm-api.md](./api/llm-api.md) | LLM 模块 API 参考 | 手写 |

## 示例 (examples/)

| 文档 | 说明 |
|------|------|
| [catalog-examples.md](./examples/catalog-examples.md) | Catalog 服务使用示例 |
| [cli-examples.md](./examples/cli-examples.md) | core-cli 命令行示例 |
| [go-sdk-examples.md](./examples/go-sdk-examples.md) | Go SDK 按能力分类的示例代码 |
| [fileservice-examples.md](./examples/fileservice-examples.md) | FileService 使用示例 |
| [model-examples.md](./examples/model-examples.md) | Model 使用示例 |
| [saga-examples.md](./examples/saga-examples.md) | Saga 组件使用示例 |
| [llm-examples.md](./examples/llm-examples.md) | LLM 模块使用示例 |
| [go-worker-e2e-dsl.md](./examples/go-worker-e2e-dsl.md) | Go Worker E2E YAML 工作流示例 |

## 工作流 (workflow/)

| 文档 | 说明 |
|------|------|
| [WORKITEMS.md](./workflow/WORKITEMS.md) | 可用 WorkItem 列表与使用说明 |
| [RAG_INGEST.md](./workflow/RAG_INGEST.md) | RAG Ingest 工作流 |
| [LINEAGE.md](./workflow/LINEAGE.md) | Workflow Lineage 的 moi-core 原子能力、数据模型与 provenance 约束 |

### 第三方接入

| 文档 | 说明 |
|------|------|
| [MINERU_OPENXML_CATALOG_INTERFACE.md](./parser/MINERU_OPENXML_CATALOG_INTERFACE.md) | 参考 LLM 接口实现 MinerU、OpenXML 的 Catalog 统一接口需求 |

## Worker (workers/)

| 文档 | 说明 | 生成方式 |
|------|------|----------|
| [node-capability-matrix.md](./workers/node-capability-matrix.md) | Worker 节点能力清单（Go + Python） | 校验（make doc） |
| [go-worker-workitems-catalog.json](./workers/go-worker-workitems-catalog.json) | Go Worker WorkItems 注册清单 | 自动生成（make doc-update） |
| [go-worker-nodes.json](./workers/go-worker-nodes.json) | Go Worker 节点 ID 列表 | 自动生成（make doc-update） |
| [java-worker-workitems-catalog.json](./workers/java-worker-workitems-catalog.json) | Java Worker WorkItems 注册清单 | 自动生成（make doc-update） |
| [java-worker-nodes.json](./workers/java-worker-nodes.json) | Java Worker 节点 ID 列表 | 自动生成（make doc-update） |
| [python-worker-workitems-catalog.json](./workers/python-worker-workitems-catalog.json) | Python Worker WorkItems 注册清单 | 自动生成（make doc-update） |
| [python-worker-nodes.json](./workers/python-worker-nodes.json) | Python Worker 节点 ID 列表 | 自动生成（make doc-update） |
| [go-worker-component-mapping.md](./workers/go-worker-component-mapping.md) | Go Worker 组件迁移映射 | 手写 |
| [java-worker-component-mapping.md](./workers/java-worker-component-mapping.md) | Java Worker 组件映射 | 手写 |
| [python-worker-component-mapping.md](./workers/python-worker-component-mapping.md) | Python Worker 组件迁移映射 | 手写 |
| [workers-migration.md](./workers/workers-migration.md) | Worker 迁移指南 | 手写 |
| [workers-migration-progress.md](./workers/workers-migration-progress.md) | Worker 迁移进度跟踪 | 手写 |
