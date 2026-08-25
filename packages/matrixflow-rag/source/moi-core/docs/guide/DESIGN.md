# MOI-Core 设计文档

本文档描述 moi-core 的模块划分、交互方式与数据库表结构，与当前代码实现一致。
**开发应用前请先阅读**：[核心概念](./CONCEPTS.md)、[SDK 开发指南](./SDK_GUIDE.md)。

---

## 1. 项目概述

### 1.1 定位

MOI-Core 是 Matrixflow 平台的核心服务，提供：

- **元数据管理（Catalog）**：工作区、目录、数据库、表、卷、文件、用户、角色与权限
- **工作流引擎（Mowl）**：工作流定义、版本（draft/published）、任务调度与执行
- **多租户**：基于 MatrixOne 的 ACCOUNT 与系统库/租户库分离
- **接入方式**：HTTP API（Catalog）、gRPC（Mowl），Go SDK 封装上述能力

### 1.2 模块边界

| 模块            | 职责                                                                                       | 不包含                 |
| --------------- | ------------------------------------------------------------------------------------------ | ---------------------- |
| **Catalog**     | 元数据 CRUD、工作区与租户初始化、认证、文件存储抽象、Mowl 嵌入/代理                        | 工作流 DAG 执行逻辑    |
| **Mowl**        | 工作流/版本/任务/工作项存储与查询、Case 执行、Worker 会话、动态服务、DynamicServiceManager | 元数据、认证、文件存储 |
| **go-sdk**      | HTTP/gRPC 客户端、DSL 构建、Worker 侧注册与执行入口                                        | 服务端业务实现         |
| **model**       | Proto 生成的公共类型（catalog、mowl、common 等）                                           | 业务逻辑               |
| **saga**        | 分布式事务（含补偿）、状态持久化                                                           | 具体业务步骤实现       |
| **fileservice** | 文件存储抽象（S3/本地/内存）                                                               | 元数据与权限           |
| **CDH**         | CDH 集群元数据管理：配置 CRUD、Hive Metastore 连接、数据库/表/列同步与查询                 | 工作流、认证、文件存储 |

---

## 2. 系统架构与交互

### 2.1 部署形态

- **内嵌 Mowl（本地/单进程）**：Catalog 进程内嵌 Mowl 引擎，HTTP 与 gRPC 共用同一端口（cmux）。配置 `[mowl] embedded = true`。
- **独立 Mowl（云/多实例）**：Catalog 与 Mowl 分离部署，Catalog 通过 `[mowl] proxy_endpoint` 将 gRPC 转发到 Mowl。配置 `[mowl] embedded = false`，`proxy_endpoint = "${MOWL_PROXY_ENDPOINT}"`。

### 2.2 请求路径

```
┌─────────────┐     HTTP      ┌──────────────────────────────────────────┐
│  go-sdk /   │ ────────────► │  Catalog (Gin HTTP)                       │
│  其他客户端  │               │  - /api/v1/workspaces, catalogs,          │
└─────────────┘               │    databases, volumes, files, users,      │
                              │    roles, workflows, workflow-versions,   │
                              │    tasks 等                               │
                              │  若 embedded=true：同端口 cmux 分流 gRPC   │
                              └─────────────────┬────────────────────────┘
                                                │ gRPC (同进程或 proxy)
                                                ▼
                              ┌──────────────────────────────────────────┐
                              │  Mowl 引擎                               │
                              │  - WorkflowManagementService             │
                              │  - MowlService (WorkerSession, CreateTask│
                              │    SignalWorkflow, 动态服务 Invoke 等)    │
                              └──────────────────────────────────────────┘
                                                ▲
┌─────────────┐     gRPC      │  Worker 客户端（go-sdk Worker）           │
│  Worker     │ ◄────────────  │  - 注册 WorkItem、建立 WorkerSession、    │
│  (执行节点)  │  双向流/单路   │    执行任务、收通知                      │
└─────────────┘               └──────────────────────────────────────────┘
```

- **Catalog**：对外 HTTP；对内连接 MatrixOne（系统库 + 各租户库）、对象存储（S3/本地）、可选 Runtime（Docker/K8s 动态 Worker）。
- **Mowl**：读写 mowl\_\* 表（在 Catalog 使用的同一 MatrixOne 系统库中），调度任务、维护 Case/Token/WorkItem 状态，与 Worker 通过 gRPC 通信。

### 2.3 目录结构（与实现一致）

```
moi-core/
├── catalog/                 # Catalog 服务
│   ├── cmd/main.go          # 入口，读 config 后启动 embed.Server
│   ├── pkg/
│   │   ├── api/             # HTTP 路由、中间件、Handler
│   │   ├── service/         # 业务逻辑（workspace/catalog/database/volume/file/user/role/mowl 等）
│   │   └── cdh/        # CDH 元数据管理（配置 CRUD、Hive Metastore 连接、同步）
│   │   ├── config/          # 配置结构与加载（TOML + 环境变量替换）
│   │   ├── schema/          # 表定义（SystemTables / TenantTables）
│   │   ├── embed/           # 嵌入式启动：DB、Provider、Saga、FileService、Mowl 初始化
│   │   ├── runtime/         # 动态 Worker 管理（Local/Cloud）
│   │   └── ...
│   └── etc/                 # config.toml, config-local.toml, config-cloud.toml, config-docker.toml
├── mowl/                    # Mowl 工作流引擎
│   ├── cmd/mowl-engine/     # 独立进程入口（可选）
│   ├── pkg/
│   │   ├── engine/          # 执行引擎（runner、task_manager、storage、dynamic_service）
│   │   ├── workflow/        # 工作流/版本服务（CRUD、Publish、Resolve）
│   │   └── model/           # 内部模型
│   └── ...
├── go-sdk/                  # Go 客户端
├── model/                   # Proto 生成类型
├── proto/                   # .proto 定义
├── saga/                    # Saga 组件
├── fileservice/             # 文件存储抽象
├── tests/                   # 集成测试
└── docs/                    # 本文档及 DEVELOPMENT.md、DEPLOYMENT.md
```

---

## 3. 数据库设计

### 3.1 库与租户

- **系统库**（如 `moi_test`）：单库，存放用户、工作区、API Key、Saga 状态、**全部 mowl\_\* 表**、功能权限定义。Catalog 使用同一 MatrixOne 实例的「系统连接」（如 dump/111）访问。
- **租户库**（每工作区一个，默认库名 `moi`）：由 Catalog 通过 MatrixOne ACCOUNT 机制创建；内存放该工作区下的 catalog、catalog_database、volume、file、volume_files、catalog_table、roles、role_permissions、user_roles、moi_object_permissions。Workspace 创建时 Catalog 会在该租户内执行 `SET GLOBAL protected_databases = 'moi'`，防止默认元数据库被普通 DDL 删除；已有 workspace 由 auto-upgrade offset 69 在租户升级时补齐该保护配置。

表定义以 `catalog/pkg/schema/tables.go` 为准，以下为概要。

### 3.2 系统库表（SystemTables）

| 表名                      | 说明                                                                                      |
| ------------------------- | ----------------------------------------------------------------------------------------- |
| saga_states               | Saga 实例状态                                                                             |
| saga_step_states          | Saga 步骤状态                                                                             |
| saga_idempotency          | Saga 幂等键                                                                               |
| users                     | 用户                                                                                      |
| api_keys                  | API Key                                                                                   |
| workspaces                | 工作区                                                                                    |
| workspace_users           | 工作区成员                                                                                |
| db_users                  | 工作区与 DB 用户映射                                                                      |
| mowl_workflow_definition  | 工作流定义                                                                                |
| mowl_workflow_version     | 工作流版本（draft/published）                                                             |
| mowl_task                 | 任务（含 type、input_schema、output_schema、result_mode、description 字段，支持动态服务） |
| mowl_workitem_metadata    | 工作项元数据                                                                              |
| mowl_workitem_shared      | 工作项共享关系                                                                            |
| mowl_workflow_cases       | 运行案例（含 state 字段，存储工作流执行期间的共享状态）                                   |
| mowl_case_status          | 案例状态                                                                                  |
| mowl_case_workitem        | 案例工作项                                                                                |
| mowl_case_workitem_status | 案例工作项状态                                                                            |
| mowl_case_token           | 案例令牌                                                                                  |
| mowl_log                  | 执行日志                                                                                  |
| moi_function_permissions  | 功能权限定义                                                                              |

### 3.3 租户库表（TenantTables）

| 表名                       | 说明                                                                    |
| -------------------------- | ----------------------------------------------------------------------- |
| file                       | 文件元数据                                                              |
| catalog                    | 目录                                                                    |
| catalog_database           | 数据库（source 字段区分来源：matrixone/cdh，config_id 关联 CDH 配置）   |
| volume                     | 卷                                                                      |
| volume_files               | 卷-文件关联                                                             |
| catalog_table              | 表元数据（source 字段区分来源：matrixone/cdh，config_id 关联 CDH 配置） |
| roles                      | 角色                                                                    |
| role_permissions           | 角色-权限                                                               |
| user_roles                 | 用户-角色                                                               |
| moi_object_permissions     | 对象权限                                                                |
| cdh_config                 | CDH 连接配置（每个 workspace 独立管理多个 CDH 集群配置）                |
| catalog_column             | 列元数据（CDH 同步的列级别信息，按 config_id 和 table_id 组织）         |
| semantic_models            | 语义模型（名称、描述、关联表、关联文件、table_set_hash）                |
| semantic_entries           | 语义条目（model_id、kind、key_name、tables、spec）                      |
| llm_config_version         | LLM 配置版本（多实例同步用）                                            |
| llm_backend                | LLM 后端模型配置（名称、类型、API Key、超时、支持模型列表）             |
| llm_backend_endpoint       | LLM 后端端点（地址、上下线状态）                                        |
| llm_router_config          | LLM 路由策略配置（策略、健康检查间隔、重试次数、会话亲和性）            |
| llm_session                | LLM 会话                                                                |
| llm_chat_message           | LLM 聊天消息                                                            |
| llm_tag                    | LLM 标签（会话与消息共用）                                              |
| llm_tag_relation           | 标签与会话/消息多对多关联                                               |
| embedding_config_version   | Embedding 配置版本                                                      |
| embedding_backend          | Embedding 后端模型配置                                                  |
| embedding_backend_endpoint | Embedding 后端端点                                                      |
| embedding_router_config    | Embedding 路由策略配置                                                  |

主键策略：系统库中 mowl\_\* 与业务表多用 VARCHAR(36) UUID；租户库中 catalog、volume、file 等使用 BIGINT 自增主键（与 CLAUDE 规范一致）。

### 3.4 权限角色与 MatrixOne 数据库角色

租户库 `roles` 保存 MC 角色元数据，MatrixOne 账号内的 `moi_role_{role_id}` 是实际承载数据库/表对象权限和用户角色授权的数据库角色。所有写入或使用数据库角色的入口都会幂等确保数据库角色基线：

- `CREATE ROLE IF NOT EXISTS moi_role_{role_id}`
- `GRANT CONNECT ON ACCOUNT * TO moi_role_{role_id}`
- `GRANT moi_user_role TO moi_role_{role_id}`

该规则适用于工作空间权限初始化、角色创建、对象权限 GRANT/REVOKE、用户角色分配/撤销、角色继承/撤销继承。这样可以在历史数据出现 MC 角色存在但 MatrixOne 角色缺失或基础授权缺失时，由后续写入口自动修复。

功能权限中的 `PERM_DATABASE_CREATE`、`PERM_DATABASE_DELETE` 会额外同步为 MatrixOne account 级 DDL 权限：`CREATE DATABASE ON ACCOUNT *`、`DROP DATABASE ON ACCOUNT *`。对象权限仍只同步到对应 database/table 范围，避免把表级权限扩大到整库。

### 3.5 知识库数据模型

语义模型（Semantic Model）用于管理与数据源绑定的语义知识，供查询改写、意图分解、NL2SQL 等 WorkItem 自动拉取。

**semantic_models 表**

| 字段           | 类型                  | 说明                                           |
| -------------- | --------------------- | ---------------------------------------------- |
| id             | BIGINT AUTO_INCREMENT | 主键                                           |
| name           | VARCHAR(255)          | 语义模型名称                                   |
| description    | TEXT                  | 描述                                           |
| tables         | JSON                  | 关联表集合 `[{db_name, table_names, parents}]` |
| files          | JSON                  | 关联文件集合 `{file_ids, parents}`             |
| table_set_hash | VARCHAR(64)           | 关联表集合哈希                                 |

**semantic_entries 表**

| 字段     | 类型                  | 说明                                                                                                                                     |
| -------- | --------------------- | ---------------------------------------------------------------------------------------------------------------------------------------- |
| id       | BIGINT AUTO_INCREMENT | 主键                                                                                                                                     |
| model_id | BIGINT                | 所属语义模型 ID                                                                                                                          |
| kind     | VARCHAR(64)           | 条目类型：`logic_text`、`glossary`、`verified_query`、`metric`、`dimension`、`relationship`、`column_preference`、`named_filter`、`fact` |
| key_name | VARCHAR(128)          | 条目键（模型内唯一）                                                                                                                     |
| tables   | JSON                  | 条目作用表                                                                                                                               |
| spec     | JSON                  | 条目规格（按 kind 不同结构不同）                                                                                                         |

**API 路径**

| 操作         | HTTP 方法 | 路径                                                    |
| ------------ | --------- | ------------------------------------------------------- |
| 创建知识库   | POST      | `/api/v1/workspaces/:id/knowledge-bases`                |
| 获取知识库   | GET       | `/api/v1/workspaces/:id/knowledge-bases/:kb_id`         |
| 列出知识库   | GET       | `/api/v1/workspaces/:id/knowledge-bases`                |
| 更新知识库   | PUT       | `/api/v1/workspaces/:id/knowledge-bases/:kb_id`         |
| 删除知识库   | DELETE    | `/api/v1/workspaces/:id/knowledge-bases/:kb_id`         |
| 创建语义知识 | POST      | `/api/v1/workspaces/:id/nl2sql-knowledge`               |
| 列出语义知识 | POST      | `/api/v1/workspaces/:id/nl2sql-knowledge/list`          |
| 删除语义知识 | DELETE    | `/api/v1/workspaces/:id/nl2sql-knowledge/:knowledge_id` |

### 3.6 LLM 代理与 Embedding 代理数据模型

LLM 代理和 Embedding 代理各自独立一套 Backend/Endpoint/Router 配置，存储在租户库中。代理对外暴露 OpenAI 兼容 API，内部通过 Router 策略将请求分发到可用的 Backend Endpoint。

**LLM 代理表**

| 表名                       | 说明                                                                           |
| -------------------------- | ------------------------------------------------------------------------------ |
| llm_backend                | 后端配置：名称、类型（QIANWEN/OPENAI/DEV_LLM 等）、API Key、超时、支持模型列表 |
| llm_backend_endpoint       | 端点：地址（URL）、状态（online/offline），外键关联 llm_backend                |
| llm_router_config          | 路由策略：ROUND_ROBIN 等、健康检查间隔、最大重试、会话亲和性                   |
| llm_config_version         | 配置版本号，Backend/Endpoint/Router 变更时自增，用于多实例同步                 |
| llm_session                | 会话（标题、来源、用户）                                                       |
| llm_chat_message           | 聊天消息（角色、内容、状态、token 统计）                                       |
| llm_tag / llm_tag_relation | 标签系统，会话与消息共用                                                       |

**Embedding 代理表**

| 表名                       | 说明                                  |
| -------------------------- | ------------------------------------- |
| embedding_backend          | 后端配置（与 LLM 结构相同，独立管理） |
| embedding_backend_endpoint | 端点                                  |
| embedding_router_config    | 路由策略                              |
| embedding_config_version   | 配置版本号                            |

**代理请求路径**

| 代理                   | API 路径                                           | 说明                       |
| ---------------------- | -------------------------------------------------- | -------------------------- |
| LLM Chat               | `POST /api/v1/workspaces/:id/llm/chat/completions` | OpenAI 兼容，支持 SSE 流式 |
| Embedding              | `POST /api/v1/workspaces/:id/embeddings`           | OpenAI 兼容                |
| LLM Backend 管理       | `/api/v1/workspaces/:id/llm/backends/...`          | CRUD + Endpoint 上下线     |
| Embedding Backend 管理 | `/api/v1/workspaces/:id/embeddings/backends/...`   | CRUD + Endpoint 上下线     |

WorkItem 通过 `client.LLM(workspaceID).BaseURL()` 获取代理地址，使用 WorkItem context 中的 `execution_context.user_api_key` 调用，不直连模型服务。

---

## 4. 关键交互说明

### 4.1 工作流版本与发布

- **draft**：可编辑、可删；**published**：可作为「最新已发布版本」被按名称解析；**deprecated**：可删不可发布。
- 按名称执行（如 `ExecuteByWorkflowName`）仅使用 **published** 版本；按 version_id 执行时 draft 也可执行。
- Publish 只更新状态并可选启动动态 Worker，**不会**自动执行一次工作流。

### 4.2 Catalog 与 Mowl 数据

- Mowl 的 workflow/task/case 等均落在 **Catalog 使用的同一系统库** 的 mowl\_\* 表中；Mowl 引擎通过 Catalog 注入的 DB 连接访问。
- 租户隔离：mowl\_\* 表带 `workspace_id`、`user_id`，Catalog 按请求上下文限定可见范围。

### 4.3 存储与 Runtime

- **Storage**：Catalog 配置 `[storage] type=local|s3`；local 时 `endpoint` 为本地根目录；s3 时需 endpoint、bucket、access_key、secret_key 等。
- **Runtime**：可选 `[runtime]` 的 local（Docker）/ cloud（K8s），用于动态 Worker 的拉起与回收；与「内嵌/独立 Mowl」正交。

---

## 4.4 RuntimeProvider — 动态 Worker 管理

RuntimeProvider 负责按需拉起和回收动态 Worker 进程，支持两种场景：

**场景 1：普通任务（Task 级别生命周期）**

```
CreateTask(RuntimeSpecJson) → RuntimeManager.LaunchWorkersForTask
    → Provider.Launch(WorkerSpec) → 等待 Worker 向 Mowl 注册
    → 任务执行
    → OnTaskFinished → TerminateWorkersForTask → Provider.Terminate
```

- 任务创建时，若 `Task.RuntimeSpecJson` 非空，RuntimeManager 解析 `RuntimeSpec.Workers` 并逐一 Launch。
- 等待所有 Worker 向 Mowl 引擎注册（轮询 `WorkerID` 是否在线），超时则失败。
- 任务完成（成功/失败/取消）后，TaskManager 调用 `OnTaskFinished`，触发 Worker 清理。

**场景 2：动态服务（WorkflowVersion 级别生命周期）**

```
PublishVersion(RuntimeSpecJson) → RuntimeManager.StartServiceWorkers
    → Provider.Launch(WorkerSpec) × N → 等待注册
    → 服务持续运行（多次任务复用同一批 Worker）
    → UnpublishVersion / 手动停止 → StopServiceWorkers → Provider.Terminate × N
```

- 发布版本时，若 `WorkflowVersion.RuntimeSpecJson` 非空，RuntimeManager 启动对应 Worker 并记录服务状态为 `running`。
- 多次任务执行复用同一批 Worker，无需每次重新拉起。
- 停止服务时，RuntimeManager 终止所有关联 Worker 并记录状态为 `stopped`。

**Provider 实现**

| Provider               | 类型    | 支持来源                | 说明                                               |
| ---------------------- | ------- | ----------------------- | -------------------------------------------------- |
| `LocalRuntimeProvider` | `local` | Docker 镜像、本地二进制 | 通过 Docker API 启动容器，或直接 `exec` 本地二进制 |
| `CloudRuntimeProvider` | `cloud` | Docker/OCI 镜像         | 通过 Kubernetes API 创建 Pod                       |

所有 Provider 实现 `RuntimeProvider` 接口：`Launch` / `Terminate` / `HealthCheck` / `ListManagedResources` / `Type`。

**OrphanGC（孤儿回收）**

- 后台定时任务（默认每 120 秒），调用 `ListManagedResources` 列出所有带 `mowl.runtime/managed=true` 标签的资源。
- 对比当前活跃的 task/service Worker 列表，超过 `grace_period`（默认 300 秒）仍无对应记录的资源视为孤儿，调用 `Terminate` 回收。

**StartupCleanup**

- Catalog 启动时调用 `StartupCleanup`，终止上次进程遗留的所有托管资源，再恢复动态服务 Worker。
- 防止进程重启后出现僵尸 Worker 占用资源。

**WorkerSpec 关键字段**

| 字段                      | 说明                                                            |
| ------------------------- | --------------------------------------------------------------- |
| `worker_id`               | Worker 在 Mowl 引擎中的唯一标识                                 |
| `provider_type`           | `local` 或 `cloud`；空则由 RuntimeManager 选第一个可用 Provider |
| `source.type`             | `image`（Docker/OCI）或 `binary`（本地二进制）                  |
| `source.image.repository` | 镜像地址，如 `registry.example.com/myworker:v1.0`               |
| `source.binary.path`      | 本地二进制路径                                                  |
| `resources`               | CPU / Memory / GPU 资源约束                                     |
| `env`                     | 注入 Worker 进程的环境变量                                      |
| `mowl_endpoint`           | Worker 连接的 Mowl gRPC 地址                                    |
| `api_key`                 | Worker 连接 Mowl 的认证 Key                                     |
| `startup_timeout`         | 等待 Worker 注册的超时秒数，默认 60                             |

---

## 5. 参考

- 表 SQL 与字段：`catalog/pkg/schema/tables.go`
- 配置项：`catalog/pkg/config/config.go`、`catalog/etc/config-*.toml`
- Catalog 启动与 Mowl 挂载：`catalog/pkg/embed/embed.go`
- Mowl 存储接口：`mowl/pkg/engine/storage/`、`mowl/pkg/engine/storage/sql/sql.go`
