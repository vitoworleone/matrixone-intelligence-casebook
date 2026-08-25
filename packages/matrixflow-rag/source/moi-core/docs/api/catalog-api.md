# Catalog Service API

moi-core Catalog Service RESTful API，提供元数据管理、用户认证、工作空间管理等功能。

版本: 1.0

## 目录

- [API Key 管理](#api-key-管理)
- [Agent - Connection](#agent---connection)
- [Astra External Auth](#astra-external-auth)
- [CDH 元数据管理](#cdh-元数据管理)
- [CDH 配置管理](#cdh-配置管理)
- [Case 管理](#case-管理)
- [Catalog 管理](#catalog-管理)
- [CatalogTrace（内部）](#catalogtrace（内部）)
- [ComputeResource](#computeresource)
- [Custom Operator 管理](#custom-operator-管理)
- [Data Asset 管理](#data-asset-管理)
- [Data Share](#data-share)
- [Database 管理](#database-管理)
- [Dataphin 元数据管理](#dataphin-元数据管理)
- [Dataphin 配置管理](#dataphin-配置管理)
- [Dynamic Service 管理](#dynamic-service-管理)
- [Embedding 管理](#embedding-管理)
- [File 管理](#file-管理)
- [Garbage Collection](#garbage-collection)
- [IAM](#iam)
- [Internal](#internal)
- [Internal Billing](#internal-billing)
- [LLM 管理](#llm-管理)
- [MaxCompute 元数据管理](#maxcompute-元数据管理)
- [MaxCompute 配置管理](#maxcompute-配置管理)
- [Mowl Lineage](#mowl-lineage)
- [Parse Result 管理](#parse-result-管理)
- [Semantic Model 管理](#semantic-model-管理)
- [System](#system)
- [System - Data Share Migration](#system---data-share-migration)
- [System - Default AI](#system---default-ai)
- [System - Runtime Dynamic Config](#system---runtime-dynamic-config)
- [System Builtin File](#system-builtin-file)
- [System Resource Display](#system-resource-display)
- [Task 管理](#task-管理)
- [User 管理](#user-管理)
- [Volume Content 管理](#volume-content-管理)
- [Volume File 管理](#volume-file-管理)
- [Volume 管理](#volume-管理)
- [WorkItem 管理](#workitem-管理)
- [Workbook 管理](#workbook-管理)
- [Workflow App](#workflow-app)
- [Workflow Deployment](#workflow-deployment)
- [Workflow Execution](#workflow-execution)
- [Workflow Lineage](#workflow-lineage)
- [Workflow 管理](#workflow-管理)
- [Workspace - Runtime Dynamic Config](#workspace---runtime-dynamic-config)
- [Workspace 管理](#workspace-管理)
- [Workspace 管理 (Internal)](#workspace-管理-(internal))
- [agents](#agents)
- [mowl](#mowl)
- [健康检查](#健康检查)
- [其他](#其他)

---

## API Key 管理

### GET /api/v1/apikeys

**列出 API Key**

列出当前用户的所有 API Key，不返回完整密钥，仅返回前缀

认证: 需要 API Key

#### 查询参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page_size | integer | 否 | 每页数量 |
| page_token | string | 否 | 分页令牌 |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | API Key 列表 | `auth.ListAPIKeysResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`auth.ListAPIKeysResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| api_keys | []auth.APIKey |  | - |

响应示例:

```json
{
  "api_keys": [{
    "created_at": 0,
    "expires_at": 0,
    "id": "string",
    "idempotency_key": "string",
    "key_prefix": "string",
    "last_used_at": 0,
    "name": "string",
    "scopes": ["string"],
    "uid": "string"
  }]
}
```

---

### POST /api/v1/apikeys

**创建 API Key**

创建新的 API Key，完整密钥仅在创建时返回一次。系统用户可通过 user_id 为其他用户创建 API Key

认证: 需要 API Key

#### 请求体

类型: `auth.CreateAPIKeyRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| expires_in_days | integer |  | - |
| idempotency_key | string |  | 可选；相同用户和幂等键重试返回同一密钥 |
| name | string |  | - |
| scopes | []string |  | - |
| user_id | string |  | 可选，仅系统用户可用 |

示例:

```json
{
  "expires_in_days": 0,
  "idempotency_key": "string",
  "name": "string",
  "scopes": ["string"],
  "user_id": "string"
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 201 | 创建成功 | `auth.CreateAPIKeyResponse` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`auth.CreateAPIKeyResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| api_key | auth.APIKey |  | - |
| key | string |  | 完整的密钥，仅创建时返回一次 |

响应示例:

```json
{
  "api_key": {
    "created_at": 0,
    "expires_at": 0,
    "id": "string",
    "idempotency_key": "string",
    "key_prefix": "string",
    "last_used_at": 0,
    "name": "string",
    "scopes": ["string"],
    "uid": "string"
  },
  "key": "string"
}
```

---

### DELETE /api/v1/apikeys/{id}

**删除 API Key**

删除指定的 API Key，仅允许删除自己的 Key

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | API Key ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 204 | 删除成功 |  |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 403 | 无权限 | `gin.ErrorResponse` |
| 404 | API Key 不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

---

## Agent - Connection

### POST /api/v1/workspaces/{id}/connections/actions/batch-create-mcp-tools

**批量创建 MCP 工具（连接 + 多工具，逐项结果）**

复用或创建 MCP Connection 后按 (connection_id, tool_name) 幂等创建多个工具；重复请求返回已有工具，单项失败不回滚其他成功工具

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 请求体

类型: `agentresource.MCPToolsBatchCreateInput`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| api_key_header | string |  | - |
| auth_type | string |  | - |
| connection_id | string |  | - |
| credential | agentresource.MCPConnectionCredentialInput |  | - |
| endpoint_uri | string |  | - |
| items | []agentresource.MCPToolBatchItemInput |  | - |
| name | string |  | - |
| transport | string |  | - |
| user_id | string |  | - |
| visibility | string |  | - |
| workspace_id | string |  | - |

示例:

```json
{
  "api_key_header": "string",
  "auth_type": "string",
  "connection_id": "string",
  "credential": {
    "api_key": "string",
    "basic_password": "string",
    "basic_username": "string",
    "bearer_token": "string",
    "custom_headers": {}
  },
  "endpoint_uri": "string",
  "items": [{
    "category": "string",
    "description": "string",
    "input_schema": {},
    "name": "string",
    "output_schema": {},
    "tags": ["string"],
    "tool_name": "string"
  }],
  "name": "string",
  "transport": "string",
  "user_id": "string",
  "visibility": "string",
  "workspace_id": "string"
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 连接与逐项创建结果 | `agentresource.MCPToolsBatchCreateResult` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 503 | connection service 未配置 | `gin.ErrorResponse` |

响应字段 (`agentresource.MCPToolsBatchCreateResult`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| connection | agentresource.Connection |  | - |
| connection_status | string |  | - |
| items | []agentresource.MCPToolBatchResultItem |  | - |

响应示例:

```json
{
  "connection": {
    "annotations": {},
    "auth_type": "string",
    "capabilities": ["string"],
    "config": {},
    "created_at": "string",
    "created_by": "string",
    "credential_ref": "string",
    "description": "string",
    "endpoint_uri": "string",
    "id": "string",
    "kind": "string",
    "labels": {},
    "last_test_error": "string",
    "last_test_status": "string",
    "last_tested_at": "string",
    "metadata": {},
    "name": "string",
    "owner_user_id": "string",
    "status": "string",
    "updated_at": "string",
    "updated_by": "string",
    "version": 0,
    "visibility": "string",
    "workspace_id": "string"
  },
  "connection_status": "string",
  "items": [{
    "error": {
      "code": "string",
      "message": "string"
    },
    "status": "string",
    "tool": {
      "annotations": {},
      "approval_policy_ref": "string",
      "bindability_reason": "string",
      "bindable": false,
      "category": "string",
      "created_at": "string",
      "created_by": "string",
      "credential_ref": "string",
      "description": "string",
      "icon_ref": "string",
      "id": "string",
      "input_schema": {},
      "kind": "string",
      "labels": {},
      "market_metadata": {},
      "metadata": {},
      "name": "string",
      "output_schema": {},
      "phase": "string",
      "redaction_policy_ref": "string",
      "side_effect_class": "string",
      "source_ref": {
        "config": {},
        "id": "string",
        "type": "string",
        "uri": "string",
        "version": "string"
      },
      "status": "string",
      "supported_runtimes": ["string"],
      "sync": {
        "last_sync_at": "string",
        "last_sync_error": "string",
        "status": "string"
      },
      "tags": ["string"],
      "updated_at": "string",
      "updated_by": "string",
      "version": 0,
      "workspace_id": "string"
    },
    "tool_name": "string"
  }]
}
```

---

### POST /api/v1/workspaces/{id}/connections/actions/probe-mcp

**探测 MCP 工具服务可用工具列表**

执行 MCP initialize 握手后调用 tools/list，验证连通性并返回可用工具数量与列表

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 请求体

类型: `handlers.ProbeMCPConnectionRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| api_key_header | string |  | - |
| auth_type | string |  | - |
| connection_id | string |  | - |
| credential | agentresource.MCPConnectionCredentialInput |  | - |
| endpoint_uri | string |  | - |
| transport | string |  | Transport is the UI transport selection (sse | http-streaming).
Controls which MCP transport handshake the probe executes. |

示例:

```json
{
  "api_key_header": "string",
  "auth_type": "string",
  "connection_id": "string",
  "credential": {
    "api_key": "string",
    "basic_password": "string",
    "basic_username": "string",
    "bearer_token": "string",
    "custom_headers": {}
  },
  "endpoint_uri": "string",
  "transport": "string"
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 工具列表与数量 | `handlers.ProbeMCPConnectionResponse` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 502 | MCP Server 无法连接或返回错误 | `gin.ErrorResponse` |
| 503 | probe 未启用 | `gin.ErrorResponse` |

响应字段 (`handlers.ProbeMCPConnectionResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| protocol_version | string |  | - |
| tool_count | integer |  | - |
| tools | []agentruntimev2.MCPTool |  | - |

响应示例:

```json
{
  "protocol_version": "string",
  "tool_count": 0,
  "tools": [{
    "_meta": ["string"],
    "annotations": ["string"],
    "description": "string",
    "icons": ["string"],
    "inputSchema": ["string"],
    "name": "string",
    "outputSchema": ["string"],
    "title": "string"
  }]
}
```

---

## Astra External Auth

### POST /api/v1/astra/edge-tokens/check

**Check an edge-registration token (revocation)**

Verifies an edge-registration token's signature, expiry, and jti revocation status. Called by Astra at edge WebSocket connect time.

#### 请求体

类型: `handlers.checkEdgeTokenRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| token | string |  | Token is the edge-registration token (with or without "Bearer " prefix). |

示例:

```json
{
  "token": "string"
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | Token is valid and not revoked | `object` |
| 400 | Invalid request | `object` |
| 401 | Invalid, expired, or revoked token | `object` |
| 503 | Edge token service unavailable | `object` |

---

### POST /api/v1/astra/external-catalog

**Astra external model catalog callbacks (service-to-service)**

Serves list_catalog_by_scope and issue_runtime_context_by_scope for edge-registration principals. Called by astra-server; authenticated with the external-catalog service key.

#### 请求体

类型: `handlers.externalCatalogActionRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| action | string |  | - |
| external_subject | string |  | - |
| provider_id | string |  | - |
| provider_scope_id | string |  | - |
| requested_model_id | string |  | - |

示例:

```json
{
  "action": "string",
  "external_subject": "string",
  "provider_id": "string",
  "provider_scope_id": "string",
  "requested_model_id": "string"
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | Action response (shape depends on action) | `object` |
| 400 | Invalid request or unknown action | `object` |
| 401 | Invalid service credential | `object` |
| 403 | Workspace model runtime permission denied | `object` |
| 404 | Requested model unavailable | `object` |
| 503 | Core authorization unavailable | `object` |

---

### POST /api/v1/astra/runner-tokens/revoke

**Revoke a runner (edge-registration) token**

Records a runner token jti as revoked. Internal service endpoint authenticated by a shared service key.

#### 请求体

类型: `handlers.revokeRunnerTokenRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| edge_agent_id | string |  | - |
| expires_at | string |  | ExpiresAt is the RFC3339 timestamp when the revoked token would naturally
expire; used to purge the revocation row afterwards. |
| jti | string |  | - |
| workspace_id | string |  | - |

示例:

```json
{
  "edge_agent_id": "string",
  "expires_at": "string",
  "jti": "string",
  "workspace_id": "string"
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | Revoked | `object` |
| 400 | Invalid request | `object` |
| 401 | Invalid service key | `object` |
| 503 | Revocation not configured | `object` |

---

## CDH 元数据管理

### GET /api/v1/workspaces/{id}/cdh/configs/{config_id}/databases

**列出 CDH 数据库**

列出指定 CDH 配置下的所有数据库，支持分页

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| config_id | integer | 是 | CDH 配置 ID |

#### 查询参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page_size | integer | 否 | 每页数量（默认 20，最大 100） |
| page_token | string | 否 | 分页令牌 |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 数据库列表 | `catalog.ListCDHDatabasesResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.ListCDHDatabasesResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| items | []catalog.CDHDatabase |  | - |
| next_page_token | string |  | - |
| total | integer |  | - |

响应示例:

```json
{
  "items": [{
    "cdh_version": "string",
    "comment": "string",
    "config_id": 0,
    "id": 0,
    "name": "string",
    "source": "string",
    "synced_at": 0
  }],
  "next_page_token": "string",
  "total": 0
}
```

---

### GET /api/v1/workspaces/{id}/cdh/configs/{config_id}/databases/{database_id}

**获取 CDH 数据库**

获取指定 CDH 数据库的详细信息

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| config_id | integer | 是 | CDH 配置 ID |
| database_id | integer | 是 | Database ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 数据库详情 | `catalog.CDHDatabase` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 404 | 不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.CDHDatabase`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| cdh_version | string |  | 同步时的 CDH 版本 |
| comment | string |  | 数据库描述 |
| config_id | integer |  | 关联的 CDH 配置 ID |
| id | integer |  | - |
| name | string |  | 数据库名称 |
| source | string |  | 数据源标识，固定为 "cdh" |
| synced_at | integer |  | 最后同步时间（Unix 时间戳，秒） |

响应示例:

```json
{
  "cdh_version": "string",
  "comment": "string",
  "config_id": 0,
  "id": 0,
  "name": "string",
  "source": "string",
  "synced_at": 0
}
```

---

### GET /api/v1/workspaces/{id}/cdh/configs/{config_id}/databases/{database_id}/tables

**列出 CDH 表**

列出指定 CDH 数据库下的所有表，支持分页

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| config_id | integer | 是 | CDH 配置 ID |
| database_id | integer | 是 | Database ID |

#### 查询参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page_size | integer | 否 | 每页数量（默认 20，最大 100） |
| page_token | string | 否 | 分页令牌 |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 表列表 | `catalog.ListCDHTablesResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.ListCDHTablesResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| items | []catalog.CDHTable |  | - |
| next_page_token | string |  | - |
| total | integer |  | - |

响应示例:

```json
{
  "items": [{
    "cdh_version": "string",
    "columns": [{
      "comment": "string",
      "data_type": "string",
      "id": 0,
      "name": "string",
      "ordinal": 0
    }],
    "comment": "string",
    "config_id": 0,
    "database_id": 0,
    "hdfs_path": "string",
    "id": 0,
    "name": "string",
    "storage_format": "string",
    "table_type": "string"
  }],
  "next_page_token": "string",
  "total": 0
}
```

---

### GET /api/v1/workspaces/{id}/cdh/configs/{config_id}/databases/{database_id}/tables/{table_id}

**获取 CDH 表**

获取指定 CDH 表的详细信息

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| config_id | integer | 是 | CDH 配置 ID |
| database_id | integer | 是 | Database ID |
| table_id | integer | 是 | Table ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 表详情 | `catalog.CDHTable` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 404 | 不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.CDHTable`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| cdh_version | string |  | 同步时的 CDH 版本 |
| columns | []catalog.CDHColumn |  | 列信息（GetTable 时返回） |
| comment | string |  | 表描述 |
| config_id | integer |  | 关联的 CDH 配置 ID |
| database_id | integer |  | 所属数据库 ID |
| hdfs_path | string |  | HDFS 路径 |
| id | integer |  | - |
| name | string |  | 表名称 |
| storage_format | string |  | 存储格式 |
| table_type | string |  | 表类型：MANAGED_TABLE, EXTERNAL_TABLE 等 |

响应示例:

```json
{
  "cdh_version": "string",
  "columns": [{
    "comment": "string",
    "data_type": "string",
    "id": 0,
    "name": "string",
    "ordinal": 0
  }],
  "comment": "string",
  "config_id": 0,
  "database_id": 0,
  "hdfs_path": "string",
  "id": 0,
  "name": "string",
  "storage_format": "string",
  "table_type": "string"
}
```

---

### GET /api/v1/workspaces/{id}/cdh/configs/{config_id}/health

**CDH 连接健康检查**

检查指定 CDH 配置的连接是否正常

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| config_id | integer | 是 | CDH 配置 ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 健康状态 | `object` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

---

### POST /api/v1/workspaces/{id}/cdh/configs/{config_id}/stop-sync

**停止 CDH 同步**

取消指定 CDH 配置的周期性同步工作流

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| config_id | integer | 是 | CDH 配置 ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 停止结果 | `object` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

---

### POST /api/v1/workspaces/{id}/cdh/configs/{config_id}/sync

**同步 CDH 元数据**

创建周期性同步 CDH Hive Metastore 元数据的工作流

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| config_id | integer | 是 | CDH 配置 ID |

#### 请求体

类型: `catalog.SyncCDHMetadataRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| cron_expression | string |  | cron 表达式，如 "0 */30 * * * *"（每30分钟） |
| database_name | string |  | 要同步的数据库名称 |

示例:

```json
{
  "cron_expression": "string",
  "database_name": "string"
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 同步结果 | `catalog.SyncCDHMetadataResponse` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.SyncCDHMetadataResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| case_id | string |  | 本次主动触发的执行ID |
| database |  |  | 同步后的数据库元数据 |
| tables_deleted | integer |  | 删除的表数量（CDH 中已不存在） |
| tables_synced | integer |  | 新同步的表数量 |
| tables_updated | integer |  | 更新的表数量 |
| task_id | string |  | 触发同步的任务ID（定时任务） |

响应示例:

```json
{
  "case_id": "string",
  "database": "",
  "tables_deleted": 0,
  "tables_synced": 0,
  "tables_updated": 0,
  "task_id": "string"
}
```

---

## CDH 配置管理

### GET /api/v1/workspaces/{id}/cdh/configs

**列出 CDH 配置**

列出指定 workspace 中的所有 CDH 配置，支持分页

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 查询参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page_size | integer | 否 | 每页数量（默认 20，最大 100） |
| page_token | string | 否 | 分页令牌 |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 配置列表 | `catalog.ListCDHConfigsResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.ListCDHConfigsResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| items | []catalog.CDHConfig |  | - |
| next_page_token | string |  | - |
| total | integer |  | - |

响应示例:

```json
{
  "items": [{
    "connect_timeout": 0,
    "created_at": 0,
    "created_by": "string",
    "hive_address": "string",
    "id": 0,
    "kerberos_principal": "string",
    "metastore_address": "string",
    "name": "string",
    "sync_cron_expression": "string",
    "sync_database_name": "string",
    "sync_task_id": "string",
    "updated_at": 0,
    "updated_by": "string",
    "version": "string"
  }],
  "next_page_token": "string",
  "total": 0
}
```

---

### POST /api/v1/workspaces/{id}/cdh/configs

**创建 CDH 配置**

在指定 workspace 中创建新的 CDH Hive Metastore 连接配置

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 请求体

类型: `catalog.CreateCDHConfigRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| connect_timeout | integer |  | 连接超时（秒），可选，默认 10 |
| hive_address | string |  | HiveServer2 地址（host:port），必填 |
| kerberos_keytab | string |  | Kerberos keytab 路径（可选） |
| kerberos_principal | string |  | Kerberos 主体（可选） |
| metastore_address | string |  | Hive Metastore 地址（host:port），必填 |
| name | string |  | 配置名称，必填 |
| version | string |  | CDH 版本，必填，如 "6.3.2" |

示例:

```json
{
  "connect_timeout": 0,
  "hive_address": "string",
  "kerberos_keytab": "string",
  "kerberos_principal": "string",
  "metastore_address": "string",
  "name": "string",
  "version": "string"
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 201 | 创建成功 | `catalog.CDHConfig` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 409 | 配置名称已存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.CDHConfig`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| connect_timeout | integer |  | 连接超时（秒），默认 10 |
| created_at | integer |  | Unix 时间戳（秒） |
| created_by | string |  | 创建者用户 ID |
| hive_address | string |  | HiveServer2 地址（host:port），如 "hiveserver2:10000" |
| id | integer |  | - |
| kerberos_principal | string |  | Kerberos 主体（可选） |
| metastore_address | string |  | Hive Metastore 地址（host:port），如 "hive-metastore:9083" |
| name | string |  | 配置名称，workspace 内唯一 |
| sync_cron_expression | string |  | 同步 cron 表达式 |
| sync_database_name | string |  | 正在同步的数据库名称 |
| sync_task_id | string |  | 周期性同步任务 ID（空表示未启动同步） |
| updated_at | integer |  | Unix 时间戳（秒） |
| updated_by | string |  | 更新者用户 ID |
| version | string |  | CDH 版本，如 "6.3.2" |

响应示例:

```json
{
  "connect_timeout": 0,
  "created_at": 0,
  "created_by": "string",
  "hive_address": "string",
  "id": 0,
  "kerberos_principal": "string",
  "metastore_address": "string",
  "name": "string",
  "sync_cron_expression": "string",
  "sync_database_name": "string",
  "sync_task_id": "string",
  "updated_at": 0,
  "updated_by": "string",
  "version": "string"
}
```

---

### GET /api/v1/workspaces/{id}/cdh/configs/{config_id}

**获取 CDH 配置**

根据 ID 获取指定 CDH 配置的详细信息

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| config_id | integer | 是 | CDH 配置 ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 配置详情 | `catalog.CDHConfig` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 404 | 配置不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.CDHConfig`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| connect_timeout | integer |  | 连接超时（秒），默认 10 |
| created_at | integer |  | Unix 时间戳（秒） |
| created_by | string |  | 创建者用户 ID |
| hive_address | string |  | HiveServer2 地址（host:port），如 "hiveserver2:10000" |
| id | integer |  | - |
| kerberos_principal | string |  | Kerberos 主体（可选） |
| metastore_address | string |  | Hive Metastore 地址（host:port），如 "hive-metastore:9083" |
| name | string |  | 配置名称，workspace 内唯一 |
| sync_cron_expression | string |  | 同步 cron 表达式 |
| sync_database_name | string |  | 正在同步的数据库名称 |
| sync_task_id | string |  | 周期性同步任务 ID（空表示未启动同步） |
| updated_at | integer |  | Unix 时间戳（秒） |
| updated_by | string |  | 更新者用户 ID |
| version | string |  | CDH 版本，如 "6.3.2" |

响应示例:

```json
{
  "connect_timeout": 0,
  "created_at": 0,
  "created_by": "string",
  "hive_address": "string",
  "id": 0,
  "kerberos_principal": "string",
  "metastore_address": "string",
  "name": "string",
  "sync_cron_expression": "string",
  "sync_database_name": "string",
  "sync_task_id": "string",
  "updated_at": 0,
  "updated_by": "string",
  "version": "string"
}
```

---

### PUT /api/v1/workspaces/{id}/cdh/configs/{config_id}

**更新 CDH 配置**

部分更新指定 CDH 配置，所有字段均为可选

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| config_id | integer | 是 | CDH 配置 ID |

#### 请求体

类型: `catalog.UpdateCDHConfigRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| connect_timeout | integer |  | - |
| hive_address | string |  | - |
| kerberos_keytab | string |  | - |
| kerberos_principal | string |  | - |
| metastore_address | string |  | - |
| name | string |  | - |
| version | string |  | - |

示例:

```json
{
  "connect_timeout": 0,
  "hive_address": "string",
  "kerberos_keytab": "string",
  "kerberos_principal": "string",
  "metastore_address": "string",
  "name": "string",
  "version": "string"
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 更新后的配置 | `catalog.CDHConfig` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 404 | 配置不存在 | `gin.ErrorResponse` |
| 409 | 配置名称已存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.CDHConfig`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| connect_timeout | integer |  | 连接超时（秒），默认 10 |
| created_at | integer |  | Unix 时间戳（秒） |
| created_by | string |  | 创建者用户 ID |
| hive_address | string |  | HiveServer2 地址（host:port），如 "hiveserver2:10000" |
| id | integer |  | - |
| kerberos_principal | string |  | Kerberos 主体（可选） |
| metastore_address | string |  | Hive Metastore 地址（host:port），如 "hive-metastore:9083" |
| name | string |  | 配置名称，workspace 内唯一 |
| sync_cron_expression | string |  | 同步 cron 表达式 |
| sync_database_name | string |  | 正在同步的数据库名称 |
| sync_task_id | string |  | 周期性同步任务 ID（空表示未启动同步） |
| updated_at | integer |  | Unix 时间戳（秒） |
| updated_by | string |  | 更新者用户 ID |
| version | string |  | CDH 版本，如 "6.3.2" |

响应示例:

```json
{
  "connect_timeout": 0,
  "created_at": 0,
  "created_by": "string",
  "hive_address": "string",
  "id": 0,
  "kerberos_principal": "string",
  "metastore_address": "string",
  "name": "string",
  "sync_cron_expression": "string",
  "sync_database_name": "string",
  "sync_task_id": "string",
  "updated_at": 0,
  "updated_by": "string",
  "version": "string"
}
```

---

### DELETE /api/v1/workspaces/{id}/cdh/configs/{config_id}

**删除 CDH 配置**

删除指定 CDH 配置，如果配置正在被使用则无法删除

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| config_id | integer | 是 | CDH 配置 ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 204 | 删除成功 |  |
| 401 | 未认证 | `gin.ErrorResponse` |
| 404 | 配置不存在 | `gin.ErrorResponse` |
| 409 | 配置正在使用中 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

---

## Case 管理

### GET /api/v1/workspaces/{id}/cases

**列出工作流 Case（执行日志）**

跨任务分页返回当前 workspace + 用户的工作流执行记录。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 查询参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| workflow_version_id | array | 否 | 过滤指定 workflow_version_id，可重复 |
| limit | integer | 否 | 每页条数，默认 20，最大 200 |
| offset | integer | 否 | 分页偏移量，默认 0 |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | Case 列表 | `object` |
| 400 | 参数错误 | `object` |
| 401 | 未认证 | `object` |
| 500 | 内部错误 | `object` |

---

### GET /api/v1/workspaces/{id}/cases/{case_id}/workitems

**列出 Case 的 workitems**

返回指定 case 的全部 workitem 行 + 状态。默认不返回 workitem.data / workitem.result 大 payload；需要原始 payload 时必须显式设置 include_payload=true。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| case_id | string | 是 | Case ID |

#### 查询参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| include_payload | boolean | 否 | 是否返回 workitem.data / workitem.result 原始 payload，默认 false |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | Workitem 列表 | `object` |
| 400 | 参数错误 | `object` |
| 401 | 未认证 | `object` |
| 500 | 内部错误 | `object` |

---

## Catalog 管理

### GET /api/v1/workspaces/{id}/catalogs/{catalog_id}/stats

**获取 Catalog 统计信息**

返回指定 Catalog 下的 database、table、volume、file 数量汇总

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| catalog_id | integer | 是 | Catalog ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 统计信息 | `handlers.CatalogStatsResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 404 | 不存在 | `gin.ErrorResponse` |
| 503 | 数据库连接失败 | `gin.ErrorResponse` |

响应字段 (`handlers.CatalogStatsResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| database_count | integer |  | - |
| file_count | integer |  | - |
| table_count | integer |  | - |
| volume_count | integer |  | - |

响应示例:

```json
{
  "database_count": 0,
  "file_count": 0,
  "table_count": 0,
  "volume_count": 0
}
```

---

### GET /api/v1/workspaces/{workspace_id}/catalog-tree

**获取 Catalog 树**

返回经过权限过滤的 Catalog、Database、Table 和 Volume 层级

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| workspace_id | string | 是 | Workspace ID |

#### 查询参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| include_table_leaves | boolean | 否 | 是否包含 Table 叶子节点（默认 true） |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | Catalog 树 | `catalog.GetCatalogTreeResponse` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.GetCatalogTreeResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| catalogs | []catalog.Catalog |  | - |
| databases | []catalog.Database |  | - |
| tables | []catalog.Table |  | - |
| volumes | []catalog.Volume |  | - |

响应示例:

```json
{
  "catalogs": [{
    "comment": "string",
    "created_at": 0,
    "created_by": "string",
    "display_bindings": [{
      "default_text": "string",
      "display_key": "string",
      "display_owner": "string",
      "field": "string"
    }],
    "id": 0,
    "name": "string",
    "updated_at": 0,
    "updated_by": "string"
  }],
  "databases": [{
    "catalog_id": 0,
    "comment": "string",
    "created_at": 0,
    "created_by": "string",
    "details_visible": false,
    "display_bindings": [{
      "default_text": "string",
      "display_key": "string",
      "display_owner": "string",
      "field": "string"
    }],
    "id": 0,
    "is_pub": false,
    "is_sub": false,
    "name": "string",
    "updated_at": 0,
    "updated_by": "string"
  }],
  "tables": [{
    "catalog_id": 0,
    "comment": "string",
    "created_at": 0,
    "created_by": "string",
    "database_id": 0,
    "extensions": {},
    "id": 0,
    "name": "string",
    "updated_at": 0,
    "updated_by": "string"
  }],
  "volumes": [{
    "catalog_id": 0,
    "comment": "string",
    "created_at": 0,
    "created_by": "string",
    "database_id": 0,
    "deleted": false,
    "deleted_at": 0,
    "display_bindings": [{
      "default_text": "string",
      "display_key": "string",
      "display_owner": "string",
      "field": "string"
    }],
    "id": 0,
    "name": "string",
    "parent_id": 0,
    "save_path": "string",
    "trigger_binding": "",
    "updated_at": 0,
    "updated_by": "string"
  }]
}
```

---

### GET /api/v1/workspaces/{workspace_id}/catalogs

**列出 Catalog**

列出指定 workspace 中的所有 Catalog，支持分页

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| workspace_id | string | 是 | Workspace ID |

#### 查询参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page_size | integer | 否 | 每页数量（默认 20，最大 100） |
| page_token | string | 否 | 分页令牌 |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | Catalog 列表 | `github_com_matrixflow_moi-core_model_catalog.ListCatalogsResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`github_com_matrixflow_moi-core_model_catalog.ListCatalogsResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| items | []catalog.Catalog |  | - |
| next_page_token | string |  | - |
| total | integer |  | - |

响应示例:

```json
{
  "items": [{
    "comment": "string",
    "created_at": 0,
    "created_by": "string",
    "display_bindings": [{
      "default_text": "string",
      "display_key": "string",
      "display_owner": "string",
      "field": "string"
    }],
    "id": 0,
    "name": "string",
    "updated_at": 0,
    "updated_by": "string"
  }],
  "next_page_token": "string",
  "total": 0
}
```

---

### POST /api/v1/workspaces/{workspace_id}/catalogs

**创建 Catalog**

在指定 workspace 中创建新的 Catalog。名称必须为 1–255 个字符，首字符只能是小写英文字母、中文汉字、数字或下划线，后续还可使用连字符和点号；不自动修正输入

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| workspace_id | string | 是 | Workspace ID |

#### 请求体

类型: `github_com_matrixflow_moi-core_model_catalog.CreateCatalogRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| comment | string |  | - |
| name | string |  | - |

示例:

```json
{
  "comment": "string",
  "name": "string"
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 201 | 创建成功 | `catalog.Catalog` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 409 | 名称已存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.Catalog`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| comment | string |  | - |
| created_at | integer |  | Unix 时间戳（秒） |
| created_by | string |  | 创建者用户 ID |
| display_bindings | []catalog.ResourceDisplayBinding |  | - |
| id | integer |  | - |
| name | string |  | - |
| updated_at | integer |  | Unix 时间戳（秒） |
| updated_by | string |  | 更新者用户 ID |

响应示例:

```json
{
  "comment": "string",
  "created_at": 0,
  "created_by": "string",
  "display_bindings": [{
    "default_text": "string",
    "display_key": "string",
    "display_owner": "string",
    "field": "string"
  }],
  "id": 0,
  "name": "string",
  "updated_at": 0,
  "updated_by": "string"
}
```

---

### GET /api/v1/workspaces/{workspace_id}/catalogs/summaries

**列出 Catalog 首页摘要**

返回当前调用者可见的 Catalog 及可见 Database 数量，支持分页

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| workspace_id | string | 是 | Workspace ID |

#### 查询参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page_size | integer | 否 | 每页数量（默认 20，最大 100） |
| page_token | string | 否 | 分页令牌 |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | Catalog 首页摘要 | `catalog.ListCatalogSummariesResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.ListCatalogSummariesResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| items | []catalog.CatalogSummary |  | - |
| next_page_token | string |  | - |
| total | integer |  | - |

响应示例:

```json
{
  "items": [{
    "catalog": {
      "comment": "string",
      "created_at": 0,
      "created_by": "string",
      "display_bindings": [{
        "default_text": "string",
        "display_key": "string",
        "display_owner": "string",
        "field": "string"
      }],
      "id": 0,
      "name": "string",
      "updated_at": 0,
      "updated_by": "string"
    },
    "database_count": 0
  }],
  "next_page_token": "string",
  "total": 0
}
```

---

### GET /api/v1/workspaces/{workspace_id}/catalogs/{id}

**获取 Catalog**

根据 ID 获取指定 Catalog 的详细信息

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| workspace_id | string | 是 | Workspace ID |
| id | integer | 是 | Catalog ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | Catalog 详情 | `catalog.Catalog` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 404 | 不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.Catalog`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| comment | string |  | - |
| created_at | integer |  | Unix 时间戳（秒） |
| created_by | string |  | 创建者用户 ID |
| display_bindings | []catalog.ResourceDisplayBinding |  | - |
| id | integer |  | - |
| name | string |  | - |
| updated_at | integer |  | Unix 时间戳（秒） |
| updated_by | string |  | 更新者用户 ID |

响应示例:

```json
{
  "comment": "string",
  "created_at": 0,
  "created_by": "string",
  "display_bindings": [{
    "default_text": "string",
    "display_key": "string",
    "display_owner": "string",
    "field": "string"
  }],
  "id": 0,
  "name": "string",
  "updated_at": 0,
  "updated_by": "string"
}
```

---

### PUT /api/v1/workspaces/{workspace_id}/catalogs/{id}

**更新 Catalog**

更新指定 Catalog 的信息。显式提交新名称时使用与创建相同的名称规则；仅修改其他字段不会重新校验或改写历史名称

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| workspace_id | string | 是 | Workspace ID |
| id | integer | 是 | Catalog ID |

#### 请求体

类型: `github_com_matrixflow_moi-core_model_catalog.UpdateCatalogRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| comment | string |  | - |
| name | string |  | - |

示例:

```json
{
  "comment": "string",
  "name": "string"
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 更新后的 Catalog | `catalog.Catalog` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 404 | 不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.Catalog`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| comment | string |  | - |
| created_at | integer |  | Unix 时间戳（秒） |
| created_by | string |  | 创建者用户 ID |
| display_bindings | []catalog.ResourceDisplayBinding |  | - |
| id | integer |  | - |
| name | string |  | - |
| updated_at | integer |  | Unix 时间戳（秒） |
| updated_by | string |  | 更新者用户 ID |

响应示例:

```json
{
  "comment": "string",
  "created_at": 0,
  "created_by": "string",
  "display_bindings": [{
    "default_text": "string",
    "display_key": "string",
    "display_owner": "string",
    "field": "string"
  }],
  "id": 0,
  "name": "string",
  "updated_at": 0,
  "updated_by": "string"
}
```

---

### DELETE /api/v1/workspaces/{workspace_id}/catalogs/{id}

**删除 Catalog**

删除指定 Catalog

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| workspace_id | string | 是 | Workspace ID |
| id | integer | 是 | Catalog ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 204 | 删除成功 |  |
| 401 | 未认证 | `gin.ErrorResponse` |
| 404 | 不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

---

### GET /api/v1/workspaces/{workspace_id}/catalogs/{id}/databases

**列出 Catalog 下的数据库**

列出指定 Catalog 下的所有数据库，支持分页

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| workspace_id | string | 是 | Workspace ID |
| id | integer | 是 | Catalog ID |

#### 查询参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page_size | integer | 否 | 每页数量（默认 20，最大 100） |
| page_token | string | 否 | 分页令牌 |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 数据库列表 | `github_com_matrixflow_moi-core_model_catalog.ListDatabasesResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 404 | Catalog 不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`github_com_matrixflow_moi-core_model_catalog.ListDatabasesResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| items | []catalog.Database |  | - |
| next_page_token | string |  | - |
| total | integer |  | - |

响应示例:

```json
{
  "items": [{
    "catalog_id": 0,
    "comment": "string",
    "created_at": 0,
    "created_by": "string",
    "details_visible": false,
    "display_bindings": [{
      "default_text": "string",
      "display_key": "string",
      "display_owner": "string",
      "field": "string"
    }],
    "id": 0,
    "is_pub": false,
    "is_sub": false,
    "name": "string",
    "updated_at": 0,
    "updated_by": "string"
  }],
  "next_page_token": "string",
  "total": 0
}
```

---

## CatalogTrace（内部）

### POST /api/v1/system/workspaces/{id}/catalog-traces

**创建 Catalog Trace 节点（内部）**

在指定 workspace 下为 Langfuse connector 创建 CatalogTrace 资源节点（系统账号调用）

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 请求体

类型: `catalog.CreateCatalogTraceRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| catalog_id | integer |  | - |
| connector_id | string |  | - |
| langfuse_host | string |  | - |
| public_key | string |  | 原始 pk；写入前由 catalog 内部脱敏存储 |
| storage_ref | string |  | langfuse_observations 所在库名 |

示例:

```json
{
  "catalog_id": 0,
  "connector_id": "string",
  "langfuse_host": "string",
  "public_key": "string",
  "storage_ref": "string"
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 201 | 创建成功 | `catalog.CatalogTrace` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 409 | connector_id 已存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.CatalogTrace`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| catalog_id | integer |  | - |
| connector_id | string |  | dataconn connector.id |
| created_at | integer |  | Unix 时间戳（秒） |
| deleted | boolean |  | - |
| deleted_at | integer |  | Unix 时间戳（秒）；0 = 未删除 |
| id | integer |  | - |
| langfuse_host | string |  | 展示用；不存 secret |
| last_error | string |  | sync_status=error 时的错误摘要 |
| last_synced_at | integer |  | Unix 时间戳（秒）；0 = 从未同步 |
| observation_count | integer |  | - |
| public_key_masked | string |  | pk 脱敏后展示（保留前4位 + "****"） |
| storage_ref | string |  | 指向 langfuse_observations 所在库名 |
| sync_status | catalog.SyncStatus |  | - |
| updated_at | integer |  | Unix 时间戳（秒） |

响应示例:

```json
{
  "catalog_id": 0,
  "connector_id": "string",
  "created_at": 0,
  "deleted": false,
  "deleted_at": 0,
  "id": 0,
  "langfuse_host": "string",
  "last_error": "string",
  "last_synced_at": 0,
  "observation_count": 0,
  "public_key_masked": "string",
  "storage_ref": "string",
  "sync_status": {},
  "updated_at": 0
}
```

---

### GET /api/v1/system/workspaces/{id}/catalog-traces/{connector_id}

**获取 Catalog Trace 详情（内部）**

返回指定 connector 的 CatalogTrace 资源节点（系统账号调用；用户侧 DC2 权限由 moi-backend 校验）

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| connector_id | string | 是 | Connector ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | CatalogTrace 详情 | `catalog.CatalogTrace` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 404 | 不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.CatalogTrace`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| catalog_id | integer |  | - |
| connector_id | string |  | dataconn connector.id |
| created_at | integer |  | Unix 时间戳（秒） |
| deleted | boolean |  | - |
| deleted_at | integer |  | Unix 时间戳（秒）；0 = 未删除 |
| id | integer |  | - |
| langfuse_host | string |  | 展示用；不存 secret |
| last_error | string |  | sync_status=error 时的错误摘要 |
| last_synced_at | integer |  | Unix 时间戳（秒）；0 = 从未同步 |
| observation_count | integer |  | - |
| public_key_masked | string |  | pk 脱敏后展示（保留前4位 + "****"） |
| storage_ref | string |  | 指向 langfuse_observations 所在库名 |
| sync_status | catalog.SyncStatus |  | - |
| updated_at | integer |  | Unix 时间戳（秒） |

响应示例:

```json
{
  "catalog_id": 0,
  "connector_id": "string",
  "created_at": 0,
  "deleted": false,
  "deleted_at": 0,
  "id": 0,
  "langfuse_host": "string",
  "last_error": "string",
  "last_synced_at": 0,
  "observation_count": 0,
  "public_key_masked": "string",
  "storage_ref": "string",
  "sync_status": {},
  "updated_at": 0
}
```

---

### DELETE /api/v1/system/workspaces/{id}/catalog-traces/{connector_id}

**软删除 Catalog Trace（内部）**

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| connector_id | string | 是 | Connector ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 204 | 删除成功 |  |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

---

### PUT /api/v1/system/workspaces/{id}/catalog-traces/{connector_id}/stats

**更新 Catalog Trace 同步统计（内部）**

Mirror Worker 每轮 sync 后调用，更新 sync_status / observation_count / last_synced_at / last_error

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| connector_id | string | 是 | Connector ID |

#### 请求体

类型: `catalog.UpdateCatalogTraceStatsRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| connector_id | string |  | - |
| last_error | string |  | 仅 sync_status=error 时有效；否则传空字符串 |
| last_synced_at | integer |  | Unix 时间戳（秒） |
| observation_count | integer |  | - |
| sync_status | catalog.SyncStatus |  | - |

示例:

```json
{
  "connector_id": "string",
  "last_error": "string",
  "last_synced_at": 0,
  "observation_count": 0,
  "sync_status": {}
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 更新成功 | `catalog.CatalogTrace` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 404 | connector_id 不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.CatalogTrace`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| catalog_id | integer |  | - |
| connector_id | string |  | dataconn connector.id |
| created_at | integer |  | Unix 时间戳（秒） |
| deleted | boolean |  | - |
| deleted_at | integer |  | Unix 时间戳（秒）；0 = 未删除 |
| id | integer |  | - |
| langfuse_host | string |  | 展示用；不存 secret |
| last_error | string |  | sync_status=error 时的错误摘要 |
| last_synced_at | integer |  | Unix 时间戳（秒）；0 = 从未同步 |
| observation_count | integer |  | - |
| public_key_masked | string |  | pk 脱敏后展示（保留前4位 + "****"） |
| storage_ref | string |  | 指向 langfuse_observations 所在库名 |
| sync_status | catalog.SyncStatus |  | - |
| updated_at | integer |  | Unix 时间戳（秒） |

响应示例:

```json
{
  "catalog_id": 0,
  "connector_id": "string",
  "created_at": 0,
  "deleted": false,
  "deleted_at": 0,
  "id": 0,
  "langfuse_host": "string",
  "last_error": "string",
  "last_synced_at": 0,
  "observation_count": 0,
  "public_key_masked": "string",
  "storage_ref": "string",
  "sync_status": {},
  "updated_at": 0
}
```

---

## ComputeResource

### POST /api/v1/system/workspaces/{id}/compute-resources/{resource_id}/activate

**激活计算资源**

Internal system API used by standalone Mowl to wake or touch a serverless ComputeResource during CR-bound scheduling.

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| resource_id | string | 是 | ComputeResource ID |

#### 请求体

类型: `handlers.activateComputeResourceRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| reason | string |  | - |
| required_worker_type | string |  | - |

示例:

```json
{
  "reason": "string",
  "required_worker_type": "string"
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | OK | `gin.H` |
| 400 | Bad Request | `gin.ErrorResponse` |
| 401 | Unauthorized | `gin.ErrorResponse` |
| 403 | Forbidden | `gin.ErrorResponse` |
| 503 | Service Unavailable | `gin.ErrorResponse` |

---

### GET /api/v1/workspaces/{id}/admin/compute-resources

**管理员列出全部计算资源**

分页列出 catalog 中未删除的 ComputeResource，并返回所属 Workspace 信息。支持按 workspace_id、status、name 精确筛选，以及按资源、工作区和账号归属字段统一包含匹配。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Authorization Workspace ID |

#### 查询参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| limit | integer | 否 | Page size, default 50, max 200 |
| offset | integer | 否 | Page offset, default 0 |
| workspace_id | string | 否 | Filter by Workspace ID |
| status | string | 否 | Filter by non-deleted ComputeResource status |
| name | string | 否 | Filter by exact ComputeResource name |
| keyword | string | 否 | Case-insensitive contains search across resource, workspace and account ownership fields |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | OK | `compute.AdminComputeResourceListResult` |
| 400 | Bad Request | `gin.ErrorResponse` |

响应字段 (`compute.AdminComputeResourceListResult`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| items | []compute.AdminComputeResource |  | - |
| limit | integer |  | - |
| offset | integer |  | - |
| total | integer |  | - |

响应示例:

```json
{
  "items": [{
    "auto_suspend_minutes": 0,
    "cpu": 0,
    "cpu_milli": 0,
    "created_at": "string",
    "created_by": "string",
    "current_replicas": 0,
    "description": "string",
    "desired_replicas": 0,
    "go_worker_image_id": "string",
    "gpu": 0,
    "gpu_cores": 0,
    "gpu_count": 0,
    "gpu_memory_mib": 0,
    "id": "string",
    "is_default": false,
    "kind": "string",
    "last_activation_at": "string",
    "last_active_at": "string",
    "max_replicas": 0,
    "memory_gib": 0,
    "memory_mib": 0,
    "min_replicas": 0,
    "name": "string",
    "platform": "string",
    "python_worker_image_id": "string",
    "scale_reason": "string",
    "spec_id": "string",
    "status": "string",
    "status_message": "string",
    "updated_at": "string",
    "worker_images": [{
      "image_id": "string",
      "kind": "string",
      "platform": "string",
      "worker_type": "string"
    }],
    "workspace": {
      "account_name": "string",
      "id": "string",
      "name": "string",
      "owner_id": "string"
    },
    "workspace_id": "string"
  }],
  "limit": 0,
  "offset": 0,
  "total": 0
}
```

---

### PUT /api/v1/workspaces/{id}/admin/compute-resources/{resource_id}

**管理员更新计算资源**

根据 ComputeResource ID 解析所属 Workspace 后更新资源配置。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Authorization Workspace ID |
| resource_id | string | 是 | ComputeResource ID |

#### 请求体

类型: `compute.UpdateRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| auto_suspend_minutes | integer |  | - |
| cpu | integer |  | - |
| description | string |  | - |
| go_worker_image_id | string |  | - |
| gpu | integer |  | - |
| kind | string |  | - |
| max_replicas | integer |  | - |
| memory_gib | integer |  | - |
| min_replicas | integer |  | - |
| name | string |  | - |
| platform | string |  | - |
| python_worker_image_id | string |  | - |
| spec_id | string |  | - |
| worker_images | []compute.WorkerImageSelection |  | - |

示例:

```json
{
  "auto_suspend_minutes": 0,
  "cpu": 0,
  "description": "string",
  "go_worker_image_id": "string",
  "gpu": 0,
  "kind": "string",
  "max_replicas": 0,
  "memory_gib": 0,
  "min_replicas": 0,
  "name": "string",
  "platform": "string",
  "python_worker_image_id": "string",
  "spec_id": "string",
  "worker_images": [{
    "image_id": "string",
    "kind": "string",
    "platform": "string",
    "worker_type": "string"
  }]
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | OK | `compute.ComputeResource` |
| 400 | Bad Request | `gin.ErrorResponse` |
| 403 | Forbidden | `gin.ErrorResponse` |
| 404 | Not Found | `gin.ErrorResponse` |
| 503 | Service Unavailable | `gin.ErrorResponse` |

响应字段 (`compute.ComputeResource`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| auto_suspend_minutes | integer |  | - |
| cpu | integer |  | - |
| cpu_milli | integer |  | - |
| created_at | string |  | - |
| created_by | string |  | - |
| current_replicas | integer |  | - |
| description | string |  | - |
| desired_replicas | integer |  | - |
| go_worker_image_id | string |  | - |
| gpu | integer |  | - |
| gpu_cores | integer |  | - |
| gpu_count | integer |  | - |
| gpu_memory_mib | integer |  | - |
| id | string |  | - |
| is_default | boolean |  | - |
| kind | string |  | - |
| last_activation_at | string |  | - |
| last_active_at | string |  | - |
| max_replicas | integer |  | - |
| memory_gib | integer |  | - |
| memory_mib | integer |  | - |
| min_replicas | integer |  | - |
| name | string |  | - |
| platform | string |  | - |
| python_worker_image_id | string |  | - |
| scale_reason | string |  | - |
| spec_id | string |  | - |
| status | string |  | - |
| status_message | string |  | - |
| updated_at | string |  | - |
| worker_images | []compute.WorkerImageSelection |  | - |
| workspace_id | string |  | - |

响应示例:

```json
{
  "auto_suspend_minutes": 0,
  "cpu": 0,
  "cpu_milli": 0,
  "created_at": "string",
  "created_by": "string",
  "current_replicas": 0,
  "description": "string",
  "desired_replicas": 0,
  "go_worker_image_id": "string",
  "gpu": 0,
  "gpu_cores": 0,
  "gpu_count": 0,
  "gpu_memory_mib": 0,
  "id": "string",
  "is_default": false,
  "kind": "string",
  "last_activation_at": "string",
  "last_active_at": "string",
  "max_replicas": 0,
  "memory_gib": 0,
  "memory_mib": 0,
  "min_replicas": 0,
  "name": "string",
  "platform": "string",
  "python_worker_image_id": "string",
  "scale_reason": "string",
  "spec_id": "string",
  "status": "string",
  "status_message": "string",
  "updated_at": "string",
  "worker_images": [{
    "image_id": "string",
    "kind": "string",
    "platform": "string",
    "worker_type": "string"
  }],
  "workspace_id": "string"
}
```

---

### DELETE /api/v1/workspaces/{id}/admin/compute-resources/{resource_id}

**管理员删除计算资源**

根据 ComputeResource ID 解析所属 Workspace 后删除对应 worker Deployment。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Authorization Workspace ID |
| resource_id | string | 是 | ComputeResource ID |

#### 查询参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| force | boolean | 否 | 强制删除：取消活跃任务并解绑工作流 |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | OK | `gin.H` |
| 404 | Not Found | `gin.ErrorResponse` |
| 409 | Conflict | `gin.H` |
| 503 | Service Unavailable | `gin.ErrorResponse` |

---

### GET /api/v1/workspaces/{id}/admin/compute-resources/{resource_id}/preflight-delete

**管理员删除预检**

根据 ComputeResource ID 解析所属 Workspace 后执行删除预检。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Authorization Workspace ID |
| resource_id | string | 是 | ComputeResource ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | OK | `compute.DeletePreflightResult` |
| 404 | Not Found | `gin.ErrorResponse` |

响应字段 (`compute.DeletePreflightResult`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| active_task_ids | []string |  | - |
| active_tasks | integer |  | - |
| bound_workflow_details | []compute.BoundWorkflowInfo |  | - |
| bound_workflows | integer |  | - |

响应示例:

```json
{
  "active_task_ids": ["string"],
  "active_tasks": 0,
  "bound_workflow_details": [{
    "id": "string",
    "name": "string",
    "node_names": ["string"]
  }],
  "bound_workflows": 0
}
```

---

### POST /api/v1/workspaces/{id}/admin/compute-resources/{resource_id}/resume

**管理员恢复计算资源到空闲**

根据 ComputeResource ID 解析所属 Workspace 后将已暂停的 ComputeResource 重置为 IDLE，不部署 Worker。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Authorization Workspace ID |
| resource_id | string | 是 | ComputeResource ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | OK | `compute.ComputeResource` |
| 400 | Bad Request | `gin.ErrorResponse` |
| 404 | Not Found | `gin.ErrorResponse` |
| 503 | Service Unavailable | `gin.ErrorResponse` |

响应字段 (`compute.ComputeResource`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| auto_suspend_minutes | integer |  | - |
| cpu | integer |  | - |
| cpu_milli | integer |  | - |
| created_at | string |  | - |
| created_by | string |  | - |
| current_replicas | integer |  | - |
| description | string |  | - |
| desired_replicas | integer |  | - |
| go_worker_image_id | string |  | - |
| gpu | integer |  | - |
| gpu_cores | integer |  | - |
| gpu_count | integer |  | - |
| gpu_memory_mib | integer |  | - |
| id | string |  | - |
| is_default | boolean |  | - |
| kind | string |  | - |
| last_activation_at | string |  | - |
| last_active_at | string |  | - |
| max_replicas | integer |  | - |
| memory_gib | integer |  | - |
| memory_mib | integer |  | - |
| min_replicas | integer |  | - |
| name | string |  | - |
| platform | string |  | - |
| python_worker_image_id | string |  | - |
| scale_reason | string |  | - |
| spec_id | string |  | - |
| status | string |  | - |
| status_message | string |  | - |
| updated_at | string |  | - |
| worker_images | []compute.WorkerImageSelection |  | - |
| workspace_id | string |  | - |

响应示例:

```json
{
  "auto_suspend_minutes": 0,
  "cpu": 0,
  "cpu_milli": 0,
  "created_at": "string",
  "created_by": "string",
  "current_replicas": 0,
  "description": "string",
  "desired_replicas": 0,
  "go_worker_image_id": "string",
  "gpu": 0,
  "gpu_cores": 0,
  "gpu_count": 0,
  "gpu_memory_mib": 0,
  "id": "string",
  "is_default": false,
  "kind": "string",
  "last_activation_at": "string",
  "last_active_at": "string",
  "max_replicas": 0,
  "memory_gib": 0,
  "memory_mib": 0,
  "min_replicas": 0,
  "name": "string",
  "platform": "string",
  "python_worker_image_id": "string",
  "scale_reason": "string",
  "spec_id": "string",
  "status": "string",
  "status_message": "string",
  "updated_at": "string",
  "worker_images": [{
    "image_id": "string",
    "kind": "string",
    "platform": "string",
    "worker_type": "string"
  }],
  "workspace_id": "string"
}
```

---

### POST /api/v1/workspaces/{id}/admin/compute-resources/{resource_id}/retry

**管理员重试计算资源到空闲**

根据 ComputeResource ID 解析所属 Workspace 后将 ERROR 状态的 ComputeResource 重置为 IDLE，不部署 Worker。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Authorization Workspace ID |
| resource_id | string | 是 | ComputeResource ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | OK | `compute.ComputeResource` |
| 400 | Bad Request | `gin.ErrorResponse` |
| 404 | Not Found | `gin.ErrorResponse` |
| 503 | Service Unavailable | `gin.ErrorResponse` |

响应字段 (`compute.ComputeResource`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| auto_suspend_minutes | integer |  | - |
| cpu | integer |  | - |
| cpu_milli | integer |  | - |
| created_at | string |  | - |
| created_by | string |  | - |
| current_replicas | integer |  | - |
| description | string |  | - |
| desired_replicas | integer |  | - |
| go_worker_image_id | string |  | - |
| gpu | integer |  | - |
| gpu_cores | integer |  | - |
| gpu_count | integer |  | - |
| gpu_memory_mib | integer |  | - |
| id | string |  | - |
| is_default | boolean |  | - |
| kind | string |  | - |
| last_activation_at | string |  | - |
| last_active_at | string |  | - |
| max_replicas | integer |  | - |
| memory_gib | integer |  | - |
| memory_mib | integer |  | - |
| min_replicas | integer |  | - |
| name | string |  | - |
| platform | string |  | - |
| python_worker_image_id | string |  | - |
| scale_reason | string |  | - |
| spec_id | string |  | - |
| status | string |  | - |
| status_message | string |  | - |
| updated_at | string |  | - |
| worker_images | []compute.WorkerImageSelection |  | - |
| workspace_id | string |  | - |

响应示例:

```json
{
  "auto_suspend_minutes": 0,
  "cpu": 0,
  "cpu_milli": 0,
  "created_at": "string",
  "created_by": "string",
  "current_replicas": 0,
  "description": "string",
  "desired_replicas": 0,
  "go_worker_image_id": "string",
  "gpu": 0,
  "gpu_cores": 0,
  "gpu_count": 0,
  "gpu_memory_mib": 0,
  "id": "string",
  "is_default": false,
  "kind": "string",
  "last_activation_at": "string",
  "last_active_at": "string",
  "max_replicas": 0,
  "memory_gib": 0,
  "memory_mib": 0,
  "min_replicas": 0,
  "name": "string",
  "platform": "string",
  "python_worker_image_id": "string",
  "scale_reason": "string",
  "spec_id": "string",
  "status": "string",
  "status_message": "string",
  "updated_at": "string",
  "worker_images": [{
    "image_id": "string",
    "kind": "string",
    "platform": "string",
    "worker_type": "string"
  }],
  "workspace_id": "string"
}
```

---

### GET /api/v1/workspaces/{id}/admin/compute-resources/{resource_id}/runtime

**管理员获取计算资源 Worker Runtime**

根据 ComputeResource ID 解析所属 Workspace 后返回该资源的按 Worker 类型运行状态。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Authorization Workspace ID |
| resource_id | string | 是 | ComputeResource ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | OK | `[]compute.ComputeResourceWorkerRuntime` |
| 400 | Bad Request | `gin.ErrorResponse` |
| 403 | Forbidden | `gin.ErrorResponse` |
| 404 | Not Found | `gin.ErrorResponse` |

---

### POST /api/v1/workspaces/{id}/admin/compute-resources/{resource_id}/suspend

**管理员暂停计算资源**

根据 ComputeResource ID 解析所属 Workspace 后暂停运行中的 ComputeResource。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Authorization Workspace ID |
| resource_id | string | 是 | ComputeResource ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | OK | `compute.ComputeResource` |
| 400 | Bad Request | `gin.ErrorResponse` |
| 404 | Not Found | `gin.ErrorResponse` |
| 503 | Service Unavailable | `gin.ErrorResponse` |

响应字段 (`compute.ComputeResource`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| auto_suspend_minutes | integer |  | - |
| cpu | integer |  | - |
| cpu_milli | integer |  | - |
| created_at | string |  | - |
| created_by | string |  | - |
| current_replicas | integer |  | - |
| description | string |  | - |
| desired_replicas | integer |  | - |
| go_worker_image_id | string |  | - |
| gpu | integer |  | - |
| gpu_cores | integer |  | - |
| gpu_count | integer |  | - |
| gpu_memory_mib | integer |  | - |
| id | string |  | - |
| is_default | boolean |  | - |
| kind | string |  | - |
| last_activation_at | string |  | - |
| last_active_at | string |  | - |
| max_replicas | integer |  | - |
| memory_gib | integer |  | - |
| memory_mib | integer |  | - |
| min_replicas | integer |  | - |
| name | string |  | - |
| platform | string |  | - |
| python_worker_image_id | string |  | - |
| scale_reason | string |  | - |
| spec_id | string |  | - |
| status | string |  | - |
| status_message | string |  | - |
| updated_at | string |  | - |
| worker_images | []compute.WorkerImageSelection |  | - |
| workspace_id | string |  | - |

响应示例:

```json
{
  "auto_suspend_minutes": 0,
  "cpu": 0,
  "cpu_milli": 0,
  "created_at": "string",
  "created_by": "string",
  "current_replicas": 0,
  "description": "string",
  "desired_replicas": 0,
  "go_worker_image_id": "string",
  "gpu": 0,
  "gpu_cores": 0,
  "gpu_count": 0,
  "gpu_memory_mib": 0,
  "id": "string",
  "is_default": false,
  "kind": "string",
  "last_activation_at": "string",
  "last_active_at": "string",
  "max_replicas": 0,
  "memory_gib": 0,
  "memory_mib": 0,
  "min_replicas": 0,
  "name": "string",
  "platform": "string",
  "python_worker_image_id": "string",
  "scale_reason": "string",
  "spec_id": "string",
  "status": "string",
  "status_message": "string",
  "updated_at": "string",
  "worker_images": [{
    "image_id": "string",
    "kind": "string",
    "platform": "string",
    "worker_type": "string"
  }],
  "workspace_id": "string"
}
```

---

### GET /api/v1/workspaces/{id}/compute-resource-specs

**列出计算资源规格**

列出 catalog 中管理的 ComputeResource 规格，包含启用和禁用规格。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | OK | `[]compute.ComputeResourceSpec` |
| 400 | Bad Request | `gin.ErrorResponse` |

---

### POST /api/v1/workspaces/{id}/compute-resource-specs

**创建计算资源规格**

创建新的 ComputeResource 规格。只有系统 admin 或配置 allowlist 中的账号可以操作。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 请求体

类型: `compute.SpecRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| cpu_milli | integer |  | - |
| credit_per_hour | number |  | - |
| description | string |  | - |
| enabled | boolean |  | - |
| family | string |  | - |
| family_name | string |  | - |
| family_name_en | string |  | - |
| gpu_cores | integer |  | - |
| gpu_count | integer |  | - |
| gpu_memory_mib | integer |  | - |
| id | string |  | - |
| kind | string |  | - |
| memory_mib | integer |  | - |
| node_placement | nodeplacement.K8sNodePlacement |  | - |

示例:

```json
{
  "cpu_milli": 0,
  "credit_per_hour": 0,
  "description": "string",
  "enabled": false,
  "family": "string",
  "family_name": "string",
  "family_name_en": "string",
  "gpu_cores": 0,
  "gpu_count": 0,
  "gpu_memory_mib": 0,
  "id": "string",
  "kind": "string",
  "memory_mib": 0,
  "node_placement": {
    "affinity": {
      "nodeAffinity": {
        "requiredDuringSchedulingIgnoredDuringExecution": {
          "nodeSelectorTerms": [{
            "matchExpressions": [{
              "key": "matrixorigin.io/worker-pool",
              "operator": "In",
              "values": ["gpu"]
            }]
          }]
        }
      }
    },
    "node_selector": {
      "kubernetes.io/os": "linux"
    },
    "priority_class_name": "moi-critical",
    "runtime_class_name": "nvidia",
    "scheduler_name": "default-scheduler",
    "tolerations": [{
      "effect": "NoSchedule",
      "key": "nvidia.com/gpu",
      "operator": "Exists"
    }],
    "topology_spread_constraints": [{
      "labelSelector": {
        "matchLabels": {
          "app": "moi-worker"
        }
      },
      "maxSkew": 1,
      "topologyKey": "topology.kubernetes.io/zone",
      "whenUnsatisfiable": "DoNotSchedule"
    }]
  }
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | OK | `compute.ComputeResourceSpec` |
| 400 | Bad Request | `gin.ErrorResponse` |
| 403 | Forbidden | `gin.ErrorResponse` |

响应字段 (`compute.ComputeResourceSpec`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| cpu_milli | integer |  | - |
| created_at | string |  | - |
| created_by | string |  | - |
| credit_per_hour | number |  | - |
| description | string |  | - |
| description_en | string |  | - |
| enabled | boolean |  | - |
| family | string |  | - |
| family_name | string |  | - |
| family_name_en | string |  | - |
| gpu_cores | integer |  | - |
| gpu_count | integer |  | - |
| gpu_memory_mib | integer |  | - |
| id | string |  | - |
| is_system | boolean |  | - |
| kind | string |  | - |
| memory_mib | integer |  | - |
| node_placement | nodeplacement.K8sNodePlacement |  | - |
| updated_at | string |  | - |
| updated_by | string |  | - |

响应示例:

```json
{
  "cpu_milli": 0,
  "created_at": "string",
  "created_by": "string",
  "credit_per_hour": 0,
  "description": "string",
  "description_en": "string",
  "enabled": false,
  "family": "string",
  "family_name": "string",
  "family_name_en": "string",
  "gpu_cores": 0,
  "gpu_count": 0,
  "gpu_memory_mib": 0,
  "id": "string",
  "is_system": false,
  "kind": "string",
  "memory_mib": 0,
  "node_placement": {
    "affinity": {
      "nodeAffinity": {
        "requiredDuringSchedulingIgnoredDuringExecution": {
          "nodeSelectorTerms": [{
            "matchExpressions": [{
              "key": "matrixorigin.io/worker-pool",
              "operator": "In",
              "values": ["gpu"]
            }]
          }]
        }
      }
    },
    "node_selector": {
      "kubernetes.io/os": "linux"
    },
    "priority_class_name": "moi-critical",
    "runtime_class_name": "nvidia",
    "scheduler_name": "default-scheduler",
    "tolerations": [{
      "effect": "NoSchedule",
      "key": "nvidia.com/gpu",
      "operator": "Exists"
    }],
    "topology_spread_constraints": [{
      "labelSelector": {
        "matchLabels": {
          "app": "moi-worker"
        }
      },
      "maxSkew": 1,
      "topologyKey": "topology.kubernetes.io/zone",
      "whenUnsatisfiable": "DoNotSchedule"
    }]
  },
  "updated_at": "string",
  "updated_by": "string"
}
```

---

### GET /api/v1/workspaces/{id}/compute-resource-specs/management-permission

**获取计算资源规格管理权限**

返回当前用户是否满足 catalog 侧规格管理 allowlist/admin 判定。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | OK | `compute.SpecManagementPermission` |
| 400 | Bad Request | `gin.ErrorResponse` |

响应字段 (`compute.SpecManagementPermission`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| can_manage_specs | boolean |  | - |

响应示例:

```json
{
  "can_manage_specs": false
}
```

---

### PUT /api/v1/workspaces/{id}/compute-resource-specs/{spec_id}

**更新计算资源规格**

更新已有 ComputeResource 规格。只有系统 admin 或配置 allowlist 中的账号可以操作。更新 node_placement 只影响后续新建、资源更新或下一次按需激活，不会立即重建已运行 worker。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| spec_id | string | 是 | Spec ID |

#### 请求体

类型: `compute.SpecRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| cpu_milli | integer |  | - |
| credit_per_hour | number |  | - |
| description | string |  | - |
| enabled | boolean |  | - |
| family | string |  | - |
| family_name | string |  | - |
| family_name_en | string |  | - |
| gpu_cores | integer |  | - |
| gpu_count | integer |  | - |
| gpu_memory_mib | integer |  | - |
| id | string |  | - |
| kind | string |  | - |
| memory_mib | integer |  | - |
| node_placement | nodeplacement.K8sNodePlacement |  | - |

示例:

```json
{
  "cpu_milli": 0,
  "credit_per_hour": 0,
  "description": "string",
  "enabled": false,
  "family": "string",
  "family_name": "string",
  "family_name_en": "string",
  "gpu_cores": 0,
  "gpu_count": 0,
  "gpu_memory_mib": 0,
  "id": "string",
  "kind": "string",
  "memory_mib": 0,
  "node_placement": {
    "affinity": {
      "nodeAffinity": {
        "requiredDuringSchedulingIgnoredDuringExecution": {
          "nodeSelectorTerms": [{
            "matchExpressions": [{
              "key": "matrixorigin.io/worker-pool",
              "operator": "In",
              "values": ["gpu"]
            }]
          }]
        }
      }
    },
    "node_selector": {
      "kubernetes.io/os": "linux"
    },
    "priority_class_name": "moi-critical",
    "runtime_class_name": "nvidia",
    "scheduler_name": "default-scheduler",
    "tolerations": [{
      "effect": "NoSchedule",
      "key": "nvidia.com/gpu",
      "operator": "Exists"
    }],
    "topology_spread_constraints": [{
      "labelSelector": {
        "matchLabels": {
          "app": "moi-worker"
        }
      },
      "maxSkew": 1,
      "topologyKey": "topology.kubernetes.io/zone",
      "whenUnsatisfiable": "DoNotSchedule"
    }]
  }
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | OK | `compute.ComputeResourceSpec` |
| 400 | Bad Request | `gin.ErrorResponse` |
| 403 | Forbidden | `gin.ErrorResponse` |
| 404 | Not Found | `gin.ErrorResponse` |

响应字段 (`compute.ComputeResourceSpec`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| cpu_milli | integer |  | - |
| created_at | string |  | - |
| created_by | string |  | - |
| credit_per_hour | number |  | - |
| description | string |  | - |
| description_en | string |  | - |
| enabled | boolean |  | - |
| family | string |  | - |
| family_name | string |  | - |
| family_name_en | string |  | - |
| gpu_cores | integer |  | - |
| gpu_count | integer |  | - |
| gpu_memory_mib | integer |  | - |
| id | string |  | - |
| is_system | boolean |  | - |
| kind | string |  | - |
| memory_mib | integer |  | - |
| node_placement | nodeplacement.K8sNodePlacement |  | - |
| updated_at | string |  | - |
| updated_by | string |  | - |

响应示例:

```json
{
  "cpu_milli": 0,
  "created_at": "string",
  "created_by": "string",
  "credit_per_hour": 0,
  "description": "string",
  "description_en": "string",
  "enabled": false,
  "family": "string",
  "family_name": "string",
  "family_name_en": "string",
  "gpu_cores": 0,
  "gpu_count": 0,
  "gpu_memory_mib": 0,
  "id": "string",
  "is_system": false,
  "kind": "string",
  "memory_mib": 0,
  "node_placement": {
    "affinity": {
      "nodeAffinity": {
        "requiredDuringSchedulingIgnoredDuringExecution": {
          "nodeSelectorTerms": [{
            "matchExpressions": [{
              "key": "matrixorigin.io/worker-pool",
              "operator": "In",
              "values": ["gpu"]
            }]
          }]
        }
      }
    },
    "node_selector": {
      "kubernetes.io/os": "linux"
    },
    "priority_class_name": "moi-critical",
    "runtime_class_name": "nvidia",
    "scheduler_name": "default-scheduler",
    "tolerations": [{
      "effect": "NoSchedule",
      "key": "nvidia.com/gpu",
      "operator": "Exists"
    }],
    "topology_spread_constraints": [{
      "labelSelector": {
        "matchLabels": {
          "app": "moi-worker"
        }
      },
      "maxSkew": 1,
      "topologyKey": "topology.kubernetes.io/zone",
      "whenUnsatisfiable": "DoNotSchedule"
    }]
  },
  "updated_at": "string",
  "updated_by": "string"
}
```

---

### PATCH /api/v1/workspaces/{id}/compute-resource-specs/{spec_id}/enabled

**启用或禁用计算资源规格**

设置 ComputeResource 规格的 enabled 状态。禁用只影响新建资源，不删除历史引用。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| spec_id | string | 是 | Spec ID |

#### 请求体

类型: `compute.SetSpecEnabledRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| enabled | boolean |  | - |

示例:

```json
{
  "enabled": false
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | OK | `compute.ComputeResourceSpec` |
| 400 | Bad Request | `gin.ErrorResponse` |
| 403 | Forbidden | `gin.ErrorResponse` |
| 404 | Not Found | `gin.ErrorResponse` |

响应字段 (`compute.ComputeResourceSpec`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| cpu_milli | integer |  | - |
| created_at | string |  | - |
| created_by | string |  | - |
| credit_per_hour | number |  | - |
| description | string |  | - |
| description_en | string |  | - |
| enabled | boolean |  | - |
| family | string |  | - |
| family_name | string |  | - |
| family_name_en | string |  | - |
| gpu_cores | integer |  | - |
| gpu_count | integer |  | - |
| gpu_memory_mib | integer |  | - |
| id | string |  | - |
| is_system | boolean |  | - |
| kind | string |  | - |
| memory_mib | integer |  | - |
| node_placement | nodeplacement.K8sNodePlacement |  | - |
| updated_at | string |  | - |
| updated_by | string |  | - |

响应示例:

```json
{
  "cpu_milli": 0,
  "created_at": "string",
  "created_by": "string",
  "credit_per_hour": 0,
  "description": "string",
  "description_en": "string",
  "enabled": false,
  "family": "string",
  "family_name": "string",
  "family_name_en": "string",
  "gpu_cores": 0,
  "gpu_count": 0,
  "gpu_memory_mib": 0,
  "id": "string",
  "is_system": false,
  "kind": "string",
  "memory_mib": 0,
  "node_placement": {
    "affinity": {
      "nodeAffinity": {
        "requiredDuringSchedulingIgnoredDuringExecution": {
          "nodeSelectorTerms": [{
            "matchExpressions": [{
              "key": "matrixorigin.io/worker-pool",
              "operator": "In",
              "values": ["gpu"]
            }]
          }]
        }
      }
    },
    "node_selector": {
      "kubernetes.io/os": "linux"
    },
    "priority_class_name": "moi-critical",
    "runtime_class_name": "nvidia",
    "scheduler_name": "default-scheduler",
    "tolerations": [{
      "effect": "NoSchedule",
      "key": "nvidia.com/gpu",
      "operator": "Exists"
    }],
    "topology_spread_constraints": [{
      "labelSelector": {
        "matchLabels": {
          "app": "moi-worker"
        }
      },
      "maxSkew": 1,
      "topologyKey": "topology.kubernetes.io/zone",
      "whenUnsatisfiable": "DoNotSchedule"
    }]
  },
  "updated_at": "string",
  "updated_by": "string"
}
```

---

### GET /api/v1/workspaces/{id}/compute-resources

**列出计算资源**

列出指定 Workspace 下未删除的 ComputeResource。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | OK | `[]handlers.ComputeResourceResponse` |
| 400 | Bad Request | `gin.ErrorResponse` |

---

### POST /api/v1/workspaces/{id}/compute-resources

**创建计算资源**

在指定 Workspace 下创建 ComputeResource 元数据。资源初始为 IDLE，首次绑定任务调度时再触发 serverless 激活。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 请求体

类型: `compute.CreateRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| auto_suspend_minutes | integer |  | - |
| cpu | integer |  | - |
| description | string |  | - |
| go_worker_image_id | string |  | - |
| gpu | integer |  | - |
| kind | string |  | Kind is an optional compatibility hint. The enabled spec is canonical. |
| max_replicas | integer |  | - |
| memory_gib | integer |  | - |
| min_replicas | integer |  | - |
| name | string |  | - |
| platform | string |  | - |
| python_worker_image_id | string |  | - |
| spec_id | string |  | - |
| worker_images | []compute.WorkerImageSelection |  | - |

示例:

```json
{
  "auto_suspend_minutes": 0,
  "cpu": 0,
  "description": "string",
  "go_worker_image_id": "string",
  "gpu": 0,
  "kind": "string",
  "max_replicas": 0,
  "memory_gib": 0,
  "min_replicas": 0,
  "name": "string",
  "platform": "string",
  "python_worker_image_id": "string",
  "spec_id": "string",
  "worker_images": [{
    "image_id": "string",
    "kind": "string",
    "platform": "string",
    "worker_type": "string"
  }]
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | OK | `compute.ComputeResource` |
| 400 | Bad Request | `gin.ErrorResponse` |
| 401 | Unauthorized | `gin.ErrorResponse` |
| 409 | Conflict | `gin.ErrorResponse` |
| 503 | Service Unavailable | `gin.ErrorResponse` |

响应字段 (`compute.ComputeResource`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| auto_suspend_minutes | integer |  | - |
| cpu | integer |  | - |
| cpu_milli | integer |  | - |
| created_at | string |  | - |
| created_by | string |  | - |
| current_replicas | integer |  | - |
| description | string |  | - |
| desired_replicas | integer |  | - |
| go_worker_image_id | string |  | - |
| gpu | integer |  | - |
| gpu_cores | integer |  | - |
| gpu_count | integer |  | - |
| gpu_memory_mib | integer |  | - |
| id | string |  | - |
| is_default | boolean |  | - |
| kind | string |  | - |
| last_activation_at | string |  | - |
| last_active_at | string |  | - |
| max_replicas | integer |  | - |
| memory_gib | integer |  | - |
| memory_mib | integer |  | - |
| min_replicas | integer |  | - |
| name | string |  | - |
| platform | string |  | - |
| python_worker_image_id | string |  | - |
| scale_reason | string |  | - |
| spec_id | string |  | - |
| status | string |  | - |
| status_message | string |  | - |
| updated_at | string |  | - |
| worker_images | []compute.WorkerImageSelection |  | - |
| workspace_id | string |  | - |

响应示例:

```json
{
  "auto_suspend_minutes": 0,
  "cpu": 0,
  "cpu_milli": 0,
  "created_at": "string",
  "created_by": "string",
  "current_replicas": 0,
  "description": "string",
  "desired_replicas": 0,
  "go_worker_image_id": "string",
  "gpu": 0,
  "gpu_cores": 0,
  "gpu_count": 0,
  "gpu_memory_mib": 0,
  "id": "string",
  "is_default": false,
  "kind": "string",
  "last_activation_at": "string",
  "last_active_at": "string",
  "max_replicas": 0,
  "memory_gib": 0,
  "memory_mib": 0,
  "min_replicas": 0,
  "name": "string",
  "platform": "string",
  "python_worker_image_id": "string",
  "scale_reason": "string",
  "spec_id": "string",
  "status": "string",
  "status_message": "string",
  "updated_at": "string",
  "worker_images": [{
    "image_id": "string",
    "kind": "string",
    "platform": "string",
    "worker_type": "string"
  }],
  "workspace_id": "string"
}
```

---

### GET /api/v1/workspaces/{id}/compute-resources/worker-images

**列出 Worker 镜像**

列出 catalog 配置同步到 DB 的 active Worker 镜像。worker_type 按可调度 Worker 类型过滤；type 保持为 image_key 兼容过滤，两个参数取交集。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 查询参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| worker_type | string | 否 | Worker 类型，例如 go-worker 或 python-worker |
| type | string | 否 | 镜像变体 image_key（兼容参数） |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | OK | `[]compute.WorkerImage` |
| 400 | Bad Request | `gin.ErrorResponse` |

---

### GET /api/v1/workspaces/{id}/compute-resources/{resource_id}

**获取计算资源**

获取指定 Workspace 下的 ComputeResource。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| resource_id | string | 是 | ComputeResource ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | OK | `handlers.ComputeResourceResponse` |
| 404 | Not Found | `gin.ErrorResponse` |

响应字段 (`handlers.ComputeResourceResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| allowed_actions | []string |  | - |
| auto_suspend_minutes | integer |  | - |
| cpu | integer |  | - |
| cpu_milli | integer |  | - |
| created_at | string |  | - |
| created_by | string |  | - |
| current_replicas | integer |  | - |
| description | string |  | - |
| desired_replicas | integer |  | - |
| go_worker_image_id | string |  | - |
| gpu | integer |  | - |
| gpu_cores | integer |  | - |
| gpu_count | integer |  | - |
| gpu_memory_mib | integer |  | - |
| id | string |  | - |
| is_default | boolean |  | - |
| kind | string |  | - |
| last_activation_at | string |  | - |
| last_active_at | string |  | - |
| max_replicas | integer |  | - |
| memory_gib | integer |  | - |
| memory_mib | integer |  | - |
| min_replicas | integer |  | - |
| name | string |  | - |
| platform | string |  | - |
| python_worker_image_id | string |  | - |
| scale_reason | string |  | - |
| spec_id | string |  | - |
| status | string |  | - |
| status_message | string |  | - |
| updated_at | string |  | - |
| worker_images | []compute.WorkerImageSelection |  | - |
| workspace_id | string |  | - |

响应示例:

```json
{
  "allowed_actions": ["string"],
  "auto_suspend_minutes": 0,
  "cpu": 0,
  "cpu_milli": 0,
  "created_at": "string",
  "created_by": "string",
  "current_replicas": 0,
  "description": "string",
  "desired_replicas": 0,
  "go_worker_image_id": "string",
  "gpu": 0,
  "gpu_cores": 0,
  "gpu_count": 0,
  "gpu_memory_mib": 0,
  "id": "string",
  "is_default": false,
  "kind": "string",
  "last_activation_at": "string",
  "last_active_at": "string",
  "max_replicas": 0,
  "memory_gib": 0,
  "memory_mib": 0,
  "min_replicas": 0,
  "name": "string",
  "platform": "string",
  "python_worker_image_id": "string",
  "scale_reason": "string",
  "spec_id": "string",
  "status": "string",
  "status_message": "string",
  "updated_at": "string",
  "worker_images": [{
    "image_id": "string",
    "kind": "string",
    "platform": "string",
    "worker_type": "string"
  }],
  "workspace_id": "string"
}
```

---

### PUT /api/v1/workspaces/{id}/compute-resources/{resource_id}

**更新计算资源**

更新指定 Workspace 下的 ComputeResource 配置；不会直接启动、停止或重建 worker workload。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| resource_id | string | 是 | ComputeResource ID |

#### 请求体

类型: `compute.UpdateRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| auto_suspend_minutes | integer |  | - |
| cpu | integer |  | - |
| description | string |  | - |
| go_worker_image_id | string |  | - |
| gpu | integer |  | - |
| kind | string |  | - |
| max_replicas | integer |  | - |
| memory_gib | integer |  | - |
| min_replicas | integer |  | - |
| name | string |  | - |
| platform | string |  | - |
| python_worker_image_id | string |  | - |
| spec_id | string |  | - |
| worker_images | []compute.WorkerImageSelection |  | - |

示例:

```json
{
  "auto_suspend_minutes": 0,
  "cpu": 0,
  "description": "string",
  "go_worker_image_id": "string",
  "gpu": 0,
  "kind": "string",
  "max_replicas": 0,
  "memory_gib": 0,
  "min_replicas": 0,
  "name": "string",
  "platform": "string",
  "python_worker_image_id": "string",
  "spec_id": "string",
  "worker_images": [{
    "image_id": "string",
    "kind": "string",
    "platform": "string",
    "worker_type": "string"
  }]
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | OK | `compute.ComputeResource` |
| 400 | Bad Request | `gin.ErrorResponse` |
| 404 | Not Found | `gin.ErrorResponse` |
| 409 | Conflict | `gin.ErrorResponse` |
| 503 | Service Unavailable | `gin.ErrorResponse` |

响应字段 (`compute.ComputeResource`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| auto_suspend_minutes | integer |  | - |
| cpu | integer |  | - |
| cpu_milli | integer |  | - |
| created_at | string |  | - |
| created_by | string |  | - |
| current_replicas | integer |  | - |
| description | string |  | - |
| desired_replicas | integer |  | - |
| go_worker_image_id | string |  | - |
| gpu | integer |  | - |
| gpu_cores | integer |  | - |
| gpu_count | integer |  | - |
| gpu_memory_mib | integer |  | - |
| id | string |  | - |
| is_default | boolean |  | - |
| kind | string |  | - |
| last_activation_at | string |  | - |
| last_active_at | string |  | - |
| max_replicas | integer |  | - |
| memory_gib | integer |  | - |
| memory_mib | integer |  | - |
| min_replicas | integer |  | - |
| name | string |  | - |
| platform | string |  | - |
| python_worker_image_id | string |  | - |
| scale_reason | string |  | - |
| spec_id | string |  | - |
| status | string |  | - |
| status_message | string |  | - |
| updated_at | string |  | - |
| worker_images | []compute.WorkerImageSelection |  | - |
| workspace_id | string |  | - |

响应示例:

```json
{
  "auto_suspend_minutes": 0,
  "cpu": 0,
  "cpu_milli": 0,
  "created_at": "string",
  "created_by": "string",
  "current_replicas": 0,
  "description": "string",
  "desired_replicas": 0,
  "go_worker_image_id": "string",
  "gpu": 0,
  "gpu_cores": 0,
  "gpu_count": 0,
  "gpu_memory_mib": 0,
  "id": "string",
  "is_default": false,
  "kind": "string",
  "last_activation_at": "string",
  "last_active_at": "string",
  "max_replicas": 0,
  "memory_gib": 0,
  "memory_mib": 0,
  "min_replicas": 0,
  "name": "string",
  "platform": "string",
  "python_worker_image_id": "string",
  "scale_reason": "string",
  "spec_id": "string",
  "status": "string",
  "status_message": "string",
  "updated_at": "string",
  "worker_images": [{
    "image_id": "string",
    "kind": "string",
    "platform": "string",
    "worker_type": "string"
  }],
  "workspace_id": "string"
}
```

---

### DELETE /api/v1/workspaces/{id}/compute-resources/{resource_id}

**删除计算资源**

删除指定 ComputeResource 对应的 worker Deployment。若有活跃任务或绑定工作流，
默认返回 409 Conflict。传 force=true 强制删除（取消任务、解绑工作流）。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| resource_id | string | 是 | ComputeResource ID |

#### 查询参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| force | boolean | 否 | 强制删除：取消活跃任务并解绑工作流 |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | OK | `gin.H` |
| 404 | Not Found | `gin.ErrorResponse` |
| 409 | 有活跃任务或绑定工作流 | `gin.H` |
| 503 | Service Unavailable | `gin.ErrorResponse` |

---

### POST /api/v1/workspaces/{id}/compute-resources/{resource_id}/adopt-legacy-ownership

**认领历史计算资源 Ownership**

仅已验证的 workspace SuperAdmin 可为缺少 canonical Ownership 的历史 ComputeResource 执行一次显式认领。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| resource_id | string | 是 | ComputeResource ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | OK | `handlers.ComputeResourceOwnershipResponse` |
| 403 | Forbidden | `gin.ErrorResponse` |
| 404 | Not Found | `gin.ErrorResponse` |
| 503 | Service Unavailable | `gin.ErrorResponse` |

响应字段 (`handlers.ComputeResourceOwnershipResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| authorized_role_id | string |  | - |
| idempotent_replay | boolean |  | - |
| owner_role_id | string |  | - |
| ownership_version | integer |  | - |
| resource_id | string |  | - |
| resource_type | string |  | - |
| status | string |  | - |
| workspace_id | string |  | - |

响应示例:

```json
{
  "authorized_role_id": "string",
  "idempotent_replay": false,
  "owner_role_id": "string",
  "ownership_version": 0,
  "resource_id": "string",
  "resource_type": "string",
  "status": "string",
  "workspace_id": "string"
}
```

---

### GET /api/v1/workspaces/{id}/compute-resources/{resource_id}/metrics

**获取计算资源指标**

获取指定 ComputeResource 的使用指标。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| resource_id | string | 是 | ComputeResource ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | OK | `gin.H` |

---

### GET /api/v1/workspaces/{id}/compute-resources/{resource_id}/preflight-delete

**删除预检**

检查指定 ComputeResource 是否有活跃任务或绑定的工作流，返回阻塞信息。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| resource_id | string | 是 | ComputeResource ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | OK | `compute.DeletePreflightResult` |
| 404 | Not Found | `gin.ErrorResponse` |

响应字段 (`compute.DeletePreflightResult`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| active_task_ids | []string |  | - |
| active_tasks | integer |  | - |
| bound_workflow_details | []compute.BoundWorkflowInfo |  | - |
| bound_workflows | integer |  | - |

响应示例:

```json
{
  "active_task_ids": ["string"],
  "active_tasks": 0,
  "bound_workflow_details": [{
    "id": "string",
    "name": "string",
    "node_names": ["string"]
  }],
  "bound_workflows": 0
}
```

---

### POST /api/v1/workspaces/{id}/compute-resources/{resource_id}/resume

**恢复计算资源到空闲**

将已暂停的 ComputeResource 重置为 IDLE，不部署 Worker；后续 WorkItem demand 才会激活所需 worker type。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| resource_id | string | 是 | ComputeResource ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | OK | `compute.ComputeResource` |
| 400 | Bad Request | `gin.ErrorResponse` |
| 404 | Not Found | `gin.ErrorResponse` |
| 503 | Service Unavailable | `gin.ErrorResponse` |

响应字段 (`compute.ComputeResource`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| auto_suspend_minutes | integer |  | - |
| cpu | integer |  | - |
| cpu_milli | integer |  | - |
| created_at | string |  | - |
| created_by | string |  | - |
| current_replicas | integer |  | - |
| description | string |  | - |
| desired_replicas | integer |  | - |
| go_worker_image_id | string |  | - |
| gpu | integer |  | - |
| gpu_cores | integer |  | - |
| gpu_count | integer |  | - |
| gpu_memory_mib | integer |  | - |
| id | string |  | - |
| is_default | boolean |  | - |
| kind | string |  | - |
| last_activation_at | string |  | - |
| last_active_at | string |  | - |
| max_replicas | integer |  | - |
| memory_gib | integer |  | - |
| memory_mib | integer |  | - |
| min_replicas | integer |  | - |
| name | string |  | - |
| platform | string |  | - |
| python_worker_image_id | string |  | - |
| scale_reason | string |  | - |
| spec_id | string |  | - |
| status | string |  | - |
| status_message | string |  | - |
| updated_at | string |  | - |
| worker_images | []compute.WorkerImageSelection |  | - |
| workspace_id | string |  | - |

响应示例:

```json
{
  "auto_suspend_minutes": 0,
  "cpu": 0,
  "cpu_milli": 0,
  "created_at": "string",
  "created_by": "string",
  "current_replicas": 0,
  "description": "string",
  "desired_replicas": 0,
  "go_worker_image_id": "string",
  "gpu": 0,
  "gpu_cores": 0,
  "gpu_count": 0,
  "gpu_memory_mib": 0,
  "id": "string",
  "is_default": false,
  "kind": "string",
  "last_activation_at": "string",
  "last_active_at": "string",
  "max_replicas": 0,
  "memory_gib": 0,
  "memory_mib": 0,
  "min_replicas": 0,
  "name": "string",
  "platform": "string",
  "python_worker_image_id": "string",
  "scale_reason": "string",
  "spec_id": "string",
  "status": "string",
  "status_message": "string",
  "updated_at": "string",
  "worker_images": [{
    "image_id": "string",
    "kind": "string",
    "platform": "string",
    "worker_type": "string"
  }],
  "workspace_id": "string"
}
```

---

### POST /api/v1/workspaces/{id}/compute-resources/{resource_id}/retry

**重试计算资源到空闲**

将 ERROR 状态的 ComputeResource 重置为 IDLE；不会直接重新部署 worker workload。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| resource_id | string | 是 | ComputeResource ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | OK | `compute.ComputeResource` |
| 400 | Bad Request | `gin.ErrorResponse` |
| 404 | Not Found | `gin.ErrorResponse` |
| 503 | Service Unavailable | `gin.ErrorResponse` |

响应字段 (`compute.ComputeResource`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| auto_suspend_minutes | integer |  | - |
| cpu | integer |  | - |
| cpu_milli | integer |  | - |
| created_at | string |  | - |
| created_by | string |  | - |
| current_replicas | integer |  | - |
| description | string |  | - |
| desired_replicas | integer |  | - |
| go_worker_image_id | string |  | - |
| gpu | integer |  | - |
| gpu_cores | integer |  | - |
| gpu_count | integer |  | - |
| gpu_memory_mib | integer |  | - |
| id | string |  | - |
| is_default | boolean |  | - |
| kind | string |  | - |
| last_activation_at | string |  | - |
| last_active_at | string |  | - |
| max_replicas | integer |  | - |
| memory_gib | integer |  | - |
| memory_mib | integer |  | - |
| min_replicas | integer |  | - |
| name | string |  | - |
| platform | string |  | - |
| python_worker_image_id | string |  | - |
| scale_reason | string |  | - |
| spec_id | string |  | - |
| status | string |  | - |
| status_message | string |  | - |
| updated_at | string |  | - |
| worker_images | []compute.WorkerImageSelection |  | - |
| workspace_id | string |  | - |

响应示例:

```json
{
  "auto_suspend_minutes": 0,
  "cpu": 0,
  "cpu_milli": 0,
  "created_at": "string",
  "created_by": "string",
  "current_replicas": 0,
  "description": "string",
  "desired_replicas": 0,
  "go_worker_image_id": "string",
  "gpu": 0,
  "gpu_cores": 0,
  "gpu_count": 0,
  "gpu_memory_mib": 0,
  "id": "string",
  "is_default": false,
  "kind": "string",
  "last_activation_at": "string",
  "last_active_at": "string",
  "max_replicas": 0,
  "memory_gib": 0,
  "memory_mib": 0,
  "min_replicas": 0,
  "name": "string",
  "platform": "string",
  "python_worker_image_id": "string",
  "scale_reason": "string",
  "spec_id": "string",
  "status": "string",
  "status_message": "string",
  "updated_at": "string",
  "worker_images": [{
    "image_id": "string",
    "kind": "string",
    "platform": "string",
    "worker_type": "string"
  }],
  "workspace_id": "string"
}
```

---

### GET /api/v1/workspaces/{id}/compute-resources/{resource_id}/runtime

**获取计算资源 Worker Runtime**

返回 ComputeResource 下每个已激活 worker type 的持久化运行状态和副本分配。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| resource_id | string | 是 | ComputeResource ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | OK | `[]compute.ComputeResourceWorkerRuntime` |
| 404 | Not Found | `gin.ErrorResponse` |

---

### POST /api/v1/workspaces/{id}/compute-resources/{resource_id}/suspend

**暂停计算资源**

暂停运行中的 ComputeResource，停止 worker 节点，不再消耗算力。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| resource_id | string | 是 | ComputeResource ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | OK | `compute.ComputeResource` |
| 400 | Bad Request | `gin.ErrorResponse` |
| 404 | Not Found | `gin.ErrorResponse` |
| 503 | Service Unavailable | `gin.ErrorResponse` |

响应字段 (`compute.ComputeResource`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| auto_suspend_minutes | integer |  | - |
| cpu | integer |  | - |
| cpu_milli | integer |  | - |
| created_at | string |  | - |
| created_by | string |  | - |
| current_replicas | integer |  | - |
| description | string |  | - |
| desired_replicas | integer |  | - |
| go_worker_image_id | string |  | - |
| gpu | integer |  | - |
| gpu_cores | integer |  | - |
| gpu_count | integer |  | - |
| gpu_memory_mib | integer |  | - |
| id | string |  | - |
| is_default | boolean |  | - |
| kind | string |  | - |
| last_activation_at | string |  | - |
| last_active_at | string |  | - |
| max_replicas | integer |  | - |
| memory_gib | integer |  | - |
| memory_mib | integer |  | - |
| min_replicas | integer |  | - |
| name | string |  | - |
| platform | string |  | - |
| python_worker_image_id | string |  | - |
| scale_reason | string |  | - |
| spec_id | string |  | - |
| status | string |  | - |
| status_message | string |  | - |
| updated_at | string |  | - |
| worker_images | []compute.WorkerImageSelection |  | - |
| workspace_id | string |  | - |

响应示例:

```json
{
  "auto_suspend_minutes": 0,
  "cpu": 0,
  "cpu_milli": 0,
  "created_at": "string",
  "created_by": "string",
  "current_replicas": 0,
  "description": "string",
  "desired_replicas": 0,
  "go_worker_image_id": "string",
  "gpu": 0,
  "gpu_cores": 0,
  "gpu_count": 0,
  "gpu_memory_mib": 0,
  "id": "string",
  "is_default": false,
  "kind": "string",
  "last_activation_at": "string",
  "last_active_at": "string",
  "max_replicas": 0,
  "memory_gib": 0,
  "memory_mib": 0,
  "min_replicas": 0,
  "name": "string",
  "platform": "string",
  "python_worker_image_id": "string",
  "scale_reason": "string",
  "spec_id": "string",
  "status": "string",
  "status_message": "string",
  "updated_at": "string",
  "worker_images": [{
    "image_id": "string",
    "kind": "string",
    "platform": "string",
    "worker_type": "string"
  }],
  "workspace_id": "string"
}
```

---

## Custom Operator 管理

### GET /api/v1/workspaces/{id}/custom-operators

**列出自定义算子**

列出 workspace 内的自定义算子，支持 enabled/language/kind/node_id/identifier/version/base_node_id 过滤

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 查询参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page_size | integer | 否 | 分页大小 |
| page_token | string | 否 | 分页 token |
| enabled | boolean | 否 | 是否启用 |
| language | string | 否 | 实现语言 |
| kind | string | 否 | 算子类型：code 或 builtin_binding |
| node_id | string | 否 | WorkItem Node ID |
| identifier | string | 否 | 算子标识 |
| version | string | 否 | 算子版本 |
| base_node_id | string | 否 | builtin_binding 基础 WorkItem Node ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 查询成功 | `catalog.ListCustomOperatorsResponse` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.ListCustomOperatorsResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| items | []catalog.CustomOperator |  | - |
| next_page_token | string |  | - |
| total | integer |  | - |

响应示例:

```json
{
  "items": [{
    "base_node_id": "string",
    "base_node_version": "string",
    "binding_config": "string",
    "catalog_id": 0,
    "created_at": 0,
    "created_by": "string",
    "database_id": 0,
    "description": "string",
    "enabled": false,
    "handler": "string",
    "id": 0,
    "identifier": "string",
    "input_schema": "string",
    "isolation_level": "string",
    "kind": {},
    "language": {},
    "name": "string",
    "node_id": "string",
    "output_schema": "string",
    "source_file_id": "string",
    "updated_at": 0,
    "updated_by": "string",
    "version": "string"
  }],
  "next_page_token": "string",
  "total": 0
}
```

---

### POST /api/v1/workspaces/{id}/custom-operators

**创建自定义算子**

创建 workspace 内的自定义算子。kind=code 时源码来自 code 或 source_file_id；kind=builtin_binding 时必须提供 base_node_id、base_node_version、binding_config，不能提供源码、语言或 handler。
kind=code 的 Python 源码必须在模块级绑定 handler；handler 必须能接收 (workspace_id, sdk, input)。Catalog 在写入前校验语法及可静态解析的签名，动态构造的 callable 由 worker 在加载时校验。
同一自定义算子可通过相同 identifier 或 node_id 创建多个 version。node_id + version 与 identifier + version 必须唯一；执行工作流时自定义算子节点必须在 DSL 中显式指定 work_item.spec.version。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 请求体

类型: `handlers.createCustomOperatorRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| base_node_id | string |  | - |
| base_node_version | string |  | - |
| binding_config |  |  | - |
| catalog_id | integer |  | - |
| code | string |  | - |
| database_id | integer |  | - |
| description | string |  | - |
| enabled | boolean |  | - |
| handler | string |  | - |
| identifier | string |  | - |
| input_schema |  |  | - |
| isolation_level | string |  | - |
| kind | string |  | - |
| language | string |  | - |
| name | string |  | - |
| node_id | string |  | - |
| output_schema |  |  | - |
| source_file_id | string |  | - |
| version | string |  | - |

示例:

```json
{
  "base_node_id": "string",
  "base_node_version": "string",
  "binding_config": "",
  "catalog_id": 0,
  "code": "string",
  "database_id": 0,
  "description": "string",
  "enabled": false,
  "handler": "string",
  "identifier": "string",
  "input_schema": "",
  "isolation_level": "string",
  "kind": "string",
  "language": "string",
  "name": "string",
  "node_id": "string",
  "output_schema": "",
  "source_file_id": "string",
  "version": "string"
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 201 | 创建成功 | `catalog.CustomOperator` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 409 | 算子已存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.CustomOperator`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| base_node_id | string |  | - |
| base_node_version | string |  | - |
| binding_config | string |  | - |
| catalog_id | integer |  | - |
| created_at | integer |  | - |
| created_by | string |  | - |
| database_id | integer |  | - |
| description | string |  | - |
| enabled | boolean |  | - |
| handler | string |  | - |
| id | integer |  | - |
| identifier | string |  | - |
| input_schema | string |  | - |
| isolation_level | string |  | - |
| kind | catalog.CustomOperatorKind |  | - |
| language | catalog.CustomOperatorLanguage |  | - |
| name | string |  | - |
| node_id | string |  | - |
| output_schema | string |  | - |
| source_file_id | string |  | - |
| updated_at | integer |  | - |
| updated_by | string |  | - |
| version | string |  | - |

响应示例:

```json
{
  "base_node_id": "string",
  "base_node_version": "string",
  "binding_config": "string",
  "catalog_id": 0,
  "created_at": 0,
  "created_by": "string",
  "database_id": 0,
  "description": "string",
  "enabled": false,
  "handler": "string",
  "id": 0,
  "identifier": "string",
  "input_schema": "string",
  "isolation_level": "string",
  "kind": {},
  "language": {},
  "name": "string",
  "node_id": "string",
  "output_schema": "string",
  "source_file_id": "string",
  "updated_at": 0,
  "updated_by": "string",
  "version": "string"
}
```

---

### GET /api/v1/workspaces/{id}/custom-operators/{operator_id}

**获取自定义算子**

获取 workspace 内指定自定义算子元数据

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| operator_id | integer | 是 | 自定义算子 ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 查询成功 | `catalog.CustomOperator` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 404 | 算子不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.CustomOperator`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| base_node_id | string |  | - |
| base_node_version | string |  | - |
| binding_config | string |  | - |
| catalog_id | integer |  | - |
| created_at | integer |  | - |
| created_by | string |  | - |
| database_id | integer |  | - |
| description | string |  | - |
| enabled | boolean |  | - |
| handler | string |  | - |
| id | integer |  | - |
| identifier | string |  | - |
| input_schema | string |  | - |
| isolation_level | string |  | - |
| kind | catalog.CustomOperatorKind |  | - |
| language | catalog.CustomOperatorLanguage |  | - |
| name | string |  | - |
| node_id | string |  | - |
| output_schema | string |  | - |
| source_file_id | string |  | - |
| updated_at | integer |  | - |
| updated_by | string |  | - |
| version | string |  | - |

响应示例:

```json
{
  "base_node_id": "string",
  "base_node_version": "string",
  "binding_config": "string",
  "catalog_id": 0,
  "created_at": 0,
  "created_by": "string",
  "database_id": 0,
  "description": "string",
  "enabled": false,
  "handler": "string",
  "id": 0,
  "identifier": "string",
  "input_schema": "string",
  "isolation_level": "string",
  "kind": {},
  "language": {},
  "name": "string",
  "node_id": "string",
  "output_schema": "string",
  "source_file_id": "string",
  "updated_at": 0,
  "updated_by": "string",
  "version": "string"
}
```

---

### PUT /api/v1/workspaces/{id}/custom-operators/{operator_id}

**更新自定义算子**

更新自定义算子元数据或源码。已创建算子的 kind 不能变更；kind=builtin_binding 不能设置源码、语言或 handler。
kind=code 的 Python 源码必须在模块级绑定 handler；handler 必须能接收 (workspace_id, sdk, input)。Catalog 在写入前校验语法及可静态解析的签名，动态构造的 callable 由 worker 在加载时校验。
已创建算子的 version 不能通过更新接口变更；需要新版本时应调用创建接口新增一条相同 identifier 或 node_id、不同 version 的记录。
启停状态不能通过更新接口变更；需要启用或停用时调用 enable/disable 动作接口。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| operator_id | integer | 是 | 自定义算子 ID |

#### 请求体

类型: `handlers.updateCustomOperatorRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| base_node_id | string |  | - |
| base_node_version | string |  | - |
| binding_config |  |  | - |
| catalog_id | integer |  | - |
| code | string |  | - |
| database_id | integer |  | - |
| description | string |  | - |
| enabled | boolean |  | - |
| handler | string |  | - |
| input_schema |  |  | - |
| isolation_level | string |  | - |
| kind | string |  | - |
| language | string |  | - |
| name | string |  | - |
| output_schema |  |  | - |
| source_file_id | string |  | - |
| version | string |  | - |

示例:

```json
{
  "base_node_id": "string",
  "base_node_version": "string",
  "binding_config": "",
  "catalog_id": 0,
  "code": "string",
  "database_id": 0,
  "description": "string",
  "enabled": false,
  "handler": "string",
  "input_schema": "",
  "isolation_level": "string",
  "kind": "string",
  "language": "string",
  "name": "string",
  "output_schema": "",
  "source_file_id": "string",
  "version": "string"
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 更新成功 | `catalog.CustomOperator` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 404 | 算子不存在 | `gin.ErrorResponse` |
| 409 | 冲突 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.CustomOperator`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| base_node_id | string |  | - |
| base_node_version | string |  | - |
| binding_config | string |  | - |
| catalog_id | integer |  | - |
| created_at | integer |  | - |
| created_by | string |  | - |
| database_id | integer |  | - |
| description | string |  | - |
| enabled | boolean |  | - |
| handler | string |  | - |
| id | integer |  | - |
| identifier | string |  | - |
| input_schema | string |  | - |
| isolation_level | string |  | - |
| kind | catalog.CustomOperatorKind |  | - |
| language | catalog.CustomOperatorLanguage |  | - |
| name | string |  | - |
| node_id | string |  | - |
| output_schema | string |  | - |
| source_file_id | string |  | - |
| updated_at | integer |  | - |
| updated_by | string |  | - |
| version | string |  | - |

响应示例:

```json
{
  "base_node_id": "string",
  "base_node_version": "string",
  "binding_config": "string",
  "catalog_id": 0,
  "created_at": 0,
  "created_by": "string",
  "database_id": 0,
  "description": "string",
  "enabled": false,
  "handler": "string",
  "id": 0,
  "identifier": "string",
  "input_schema": "string",
  "isolation_level": "string",
  "kind": {},
  "language": {},
  "name": "string",
  "node_id": "string",
  "output_schema": "string",
  "source_file_id": "string",
  "updated_at": 0,
  "updated_by": "string",
  "version": "string"
}
```

---

### DELETE /api/v1/workspaces/{id}/custom-operators/{operator_id}

**删除自定义算子**

删除自定义算子并减少源码文件引用计数

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| operator_id | integer | 是 | 自定义算子 ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 删除成功 | `object` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 404 | 算子不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

---

### GET /api/v1/workspaces/{id}/custom-operators/{operator_id}/code

**获取自定义算子源码**

下载自定义算子的 Python 源码内容

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| operator_id | integer | 是 | 自定义算子 ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 源码内容 | `file` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 404 | 算子不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

---

### POST /api/v1/workspaces/{id}/custom-operators/{operator_id}/disable

**停用自定义算子**

只更新自定义算子的启用状态为 false。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| operator_id | integer | 是 | 自定义算子 ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 停用成功 | `catalog.CustomOperator` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 404 | 算子不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.CustomOperator`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| base_node_id | string |  | - |
| base_node_version | string |  | - |
| binding_config | string |  | - |
| catalog_id | integer |  | - |
| created_at | integer |  | - |
| created_by | string |  | - |
| database_id | integer |  | - |
| description | string |  | - |
| enabled | boolean |  | - |
| handler | string |  | - |
| id | integer |  | - |
| identifier | string |  | - |
| input_schema | string |  | - |
| isolation_level | string |  | - |
| kind | catalog.CustomOperatorKind |  | - |
| language | catalog.CustomOperatorLanguage |  | - |
| name | string |  | - |
| node_id | string |  | - |
| output_schema | string |  | - |
| source_file_id | string |  | - |
| updated_at | integer |  | - |
| updated_by | string |  | - |
| version | string |  | - |

响应示例:

```json
{
  "base_node_id": "string",
  "base_node_version": "string",
  "binding_config": "string",
  "catalog_id": 0,
  "created_at": 0,
  "created_by": "string",
  "database_id": 0,
  "description": "string",
  "enabled": false,
  "handler": "string",
  "id": 0,
  "identifier": "string",
  "input_schema": "string",
  "isolation_level": "string",
  "kind": {},
  "language": {},
  "name": "string",
  "node_id": "string",
  "output_schema": "string",
  "source_file_id": "string",
  "updated_at": 0,
  "updated_by": "string",
  "version": "string"
}
```

---

### POST /api/v1/workspaces/{id}/custom-operators/{operator_id}/enable

**启用自定义算子**

只更新自定义算子的启用状态为 true。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| operator_id | integer | 是 | 自定义算子 ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 启用成功 | `catalog.CustomOperator` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 404 | 算子不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.CustomOperator`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| base_node_id | string |  | - |
| base_node_version | string |  | - |
| binding_config | string |  | - |
| catalog_id | integer |  | - |
| created_at | integer |  | - |
| created_by | string |  | - |
| database_id | integer |  | - |
| description | string |  | - |
| enabled | boolean |  | - |
| handler | string |  | - |
| id | integer |  | - |
| identifier | string |  | - |
| input_schema | string |  | - |
| isolation_level | string |  | - |
| kind | catalog.CustomOperatorKind |  | - |
| language | catalog.CustomOperatorLanguage |  | - |
| name | string |  | - |
| node_id | string |  | - |
| output_schema | string |  | - |
| source_file_id | string |  | - |
| updated_at | integer |  | - |
| updated_by | string |  | - |
| version | string |  | - |

响应示例:

```json
{
  "base_node_id": "string",
  "base_node_version": "string",
  "binding_config": "string",
  "catalog_id": 0,
  "created_at": 0,
  "created_by": "string",
  "database_id": 0,
  "description": "string",
  "enabled": false,
  "handler": "string",
  "id": 0,
  "identifier": "string",
  "input_schema": "string",
  "isolation_level": "string",
  "kind": {},
  "language": {},
  "name": "string",
  "node_id": "string",
  "output_schema": "string",
  "source_file_id": "string",
  "updated_at": 0,
  "updated_by": "string",
  "version": "string"
}
```

---

## Data Asset 管理

### GET /api/v1/workspaces/{id}/catalog-files/{file_id}/asset

**解析目录文件与产物桥接关系**

根据 catalog file id 返回其对应的 lineage 入口桥接信息

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| file_id | string | 是 | Catalog File ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 解析成功 | `catalog.CatalogFileAssetResolveItem` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.CatalogFileAssetResolveItem`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| entry_case_id | string |  | - |
| entry_recorded_by_workitem_id | string |  | - |
| file_exists | boolean |  | - |
| file_id | string |  | - |
| has_artifact | boolean |  | - |
| origin_file_ext | string |  | - |
| origin_file_name | string |  | - |
| parsed_asset_id | string |  | - |
| parsed_file_id | string |  | - |
| root_asset_id | string |  | - |
| source_file_id | string |  | - |
| stage_asset_id | string |  | - |
| volume_id | integer |  | - |

响应示例:

```json
{
  "entry_case_id": "string",
  "entry_recorded_by_workitem_id": "string",
  "file_exists": false,
  "file_id": "string",
  "has_artifact": false,
  "origin_file_ext": "string",
  "origin_file_name": "string",
  "parsed_asset_id": "string",
  "parsed_file_id": "string",
  "root_asset_id": "string",
  "source_file_id": "string",
  "stage_asset_id": "string",
  "volume_id": 0
}
```

---

### POST /api/v1/workspaces/{id}/catalog-files:batchResolve

**批量解析目录文件与产物桥接关系**

批量返回多个 catalog file id 对应的 lineage 入口桥接信息

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 请求体

类型: `catalog.BatchResolveCatalogFilesRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| file_ids | []string |  | - |

示例:

```json
{
  "file_ids": ["string"]
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 解析成功 | `catalog.BatchResolveCatalogFilesResponse` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.BatchResolveCatalogFilesResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| items | []catalog.CatalogFileAssetResolveItem |  | - |

响应示例:

```json
{
  "items": [{
    "entry_case_id": "string",
    "entry_recorded_by_workitem_id": "string",
    "file_exists": false,
    "file_id": "string",
    "has_artifact": false,
    "origin_file_ext": "string",
    "origin_file_name": "string",
    "parsed_asset_id": "string",
    "parsed_file_id": "string",
    "root_asset_id": "string",
    "source_file_id": "string",
    "stage_asset_id": "string",
    "volume_id": 0
  }]
}
```

---

### POST /api/v1/workspaces/{id}/data-assets

**创建数据资产**

在指定 workspace 中创建新的数据资产

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 请求体

类型: `catalog.CreateDataAssetRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| asset_id | string |  | - |
| asset_ref | string |  | - |
| asset_type | string |  | - |
| meta | structpb.Struct |  | - |
| name | string |  | - |
| raw_file_id | string |  | Deprecated: Marked as deprecated in catalog/catalog.proto. |
| replace_meta | boolean |  | - |
| source | string |  | - |
| volume_id | integer |  | - |

示例:

```json
{
  "asset_id": "string",
  "asset_ref": "string",
  "asset_type": "string",
  "meta": {
    "fields": {}
  },
  "name": "string",
  "raw_file_id": "string",
  "replace_meta": false,
  "source": "string",
  "volume_id": 0
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 201 | 创建成功 | `catalog.DataAsset` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.DataAsset`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| asset_id | string |  | - |
| asset_ref | string |  | - |
| asset_type | string |  | - |
| created_at | integer |  | - |
| id | integer |  | - |
| meta | structpb.Struct |  | - |
| name | string |  | - |
| raw_file_id | string |  | Deprecated: Marked as deprecated in catalog/catalog.proto. |
| source | string |  | - |
| updated_at | integer |  | - |
| volume_id | integer |  | - |

响应示例:

```json
{
  "asset_id": "string",
  "asset_ref": "string",
  "asset_type": "string",
  "created_at": 0,
  "id": 0,
  "meta": {
    "fields": {}
  },
  "name": "string",
  "raw_file_id": "string",
  "source": "string",
  "updated_at": 0,
  "volume_id": 0
}
```

---

### POST /api/v1/workspaces/{id}/data-assets/derivations

**创建数据衍生**

在指定 workspace 中为数据资产创建衍生记录

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 请求体

类型: `catalog.CreateDataDerivationRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| asset_id | string |  | Deprecated: Marked as deprecated in catalog/catalog.proto. |
| case_id | string |  | - |
| file_id | string |  | Deprecated: Marked as deprecated in catalog/catalog.proto. |
| idempotency_key | string |  | - |
| kind | string |  | - |
| logical_slot | string |  | - |
| meta | structpb.Struct |  | - |
| parallel_index | integer |  | - |
| producer_workitem_id | string |  | - |
| recorded_by_workitem_id | string |  | - |
| root_asset_id | string |  | - |
| source_asset_id | string |  | - |
| target_asset_id | string |  | - |

示例:

```json
{
  "asset_id": "string",
  "case_id": "string",
  "file_id": "string",
  "idempotency_key": "string",
  "kind": "string",
  "logical_slot": "string",
  "meta": {
    "fields": {}
  },
  "parallel_index": 0,
  "producer_workitem_id": "string",
  "recorded_by_workitem_id": "string",
  "root_asset_id": "string",
  "source_asset_id": "string",
  "target_asset_id": "string"
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 201 | 创建成功 | `catalog.DataDerivation` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.DataDerivation`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| asset_id | string |  | Deprecated: Marked as deprecated in catalog/catalog.proto. |
| case_id | string |  | - |
| created_at | integer |  | - |
| file_id | string |  | Deprecated: Marked as deprecated in catalog/catalog.proto. |
| id | integer |  | - |
| idempotency_key | string |  | - |
| kind | string |  | - |
| logical_slot | string |  | - |
| meta | structpb.Struct |  | - |
| parallel_index | integer |  | - |
| producer_workitem_id | string |  | - |
| recorded_by_workitem_id | string |  | - |
| root_asset_id | string |  | - |
| source_asset_id | string |  | - |
| target_asset_id | string |  | - |
| updated_at | integer |  | - |

响应示例:

```json
{
  "asset_id": "string",
  "case_id": "string",
  "created_at": 0,
  "file_id": "string",
  "id": 0,
  "idempotency_key": "string",
  "kind": "string",
  "logical_slot": "string",
  "meta": {
    "fields": {}
  },
  "parallel_index": 0,
  "producer_workitem_id": "string",
  "recorded_by_workitem_id": "string",
  "root_asset_id": "string",
  "source_asset_id": "string",
  "target_asset_id": "string",
  "updated_at": 0
}
```

---

### PUT /api/v1/workspaces/{id}/data-assets/manifest

**更新或创建解析清单**

在指定 workspace 中更新或创建数据资产的解析清单

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 请求体

类型: `catalog.UpsertParsedManifestRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| asset_id | string |  | Deprecated: Marked as deprecated in catalog/catalog.proto. |
| manifest | structpb.Struct |  | - |
| parsed_asset_id | string |  | - |
| parsed_file_id | string |  | - |
| raw_file_id | string |  | Deprecated: Marked as deprecated in catalog/catalog.proto. |
| root_asset_id | string |  | - |
| source_file_id | string |  | - |

示例:

```json
{
  "asset_id": "string",
  "manifest": {
    "fields": {}
  },
  "parsed_asset_id": "string",
  "parsed_file_id": "string",
  "raw_file_id": "string",
  "root_asset_id": "string",
  "source_file_id": "string"
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 操作成功 | `catalog.ParsedManifest` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.ParsedManifest`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| asset_id | string |  | Deprecated: Marked as deprecated in catalog/catalog.proto. |
| created_at | integer |  | - |
| id | integer |  | - |
| manifest | structpb.Struct |  | - |
| parsed_asset_id | string |  | - |
| parsed_file_id | string |  | - |
| raw_file_id | string |  | Deprecated: Marked as deprecated in catalog/catalog.proto. |
| root_asset_id | string |  | - |
| source_file_id | string |  | - |
| updated_at | integer |  | - |

响应示例:

```json
{
  "asset_id": "string",
  "created_at": 0,
  "id": 0,
  "manifest": {
    "fields": {}
  },
  "parsed_asset_id": "string",
  "parsed_file_id": "string",
  "raw_file_id": "string",
  "root_asset_id": "string",
  "source_file_id": "string",
  "updated_at": 0
}
```

---

### GET /api/v1/workspaces/{id}/data-assets/resolve

**解析数据资产**

根据 asset_id 或 asset_type+asset_ref 解析数据资产及其衍生和清单信息

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 查询参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| asset_id | string | 否 | 数据资产 ID |
| asset_type | string | 否 | 资产类型 |
| asset_ref | string | 否 | 类型内引用 |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 解析结果 | `catalog.DataAssetResolveResponse` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 404 | 资产不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.DataAssetResolveResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| asset | catalog.DataAsset |  | - |
| assets_by_id | object |  | - |
| derivations | []catalog.DataDerivation |  | - |
| manifest | catalog.ParsedManifest |  | - |

响应示例:

```json
{
  "asset": {
    "asset_id": "string",
    "asset_ref": "string",
    "asset_type": "string",
    "created_at": 0,
    "id": 0,
    "meta": {
      "fields": {}
    },
    "name": "string",
    "raw_file_id": "string",
    "source": "string",
    "updated_at": 0,
    "volume_id": 0
  },
  "assets_by_id": {},
  "derivations": [{
    "asset_id": "string",
    "case_id": "string",
    "created_at": 0,
    "file_id": "string",
    "id": 0,
    "idempotency_key": "string",
    "kind": "string",
    "logical_slot": "string",
    "meta": {
      "fields": {}
    },
    "parallel_index": 0,
    "producer_workitem_id": "string",
    "recorded_by_workitem_id": "string",
    "root_asset_id": "string",
    "source_asset_id": "string",
    "target_asset_id": "string",
    "updated_at": 0
  }],
  "manifest": {
    "asset_id": "string",
    "created_at": 0,
    "id": 0,
    "manifest": {
      "fields": {}
    },
    "parsed_asset_id": "string",
    "parsed_file_id": "string",
    "raw_file_id": "string",
    "root_asset_id": "string",
    "source_file_id": "string",
    "updated_at": 0
  }
}
```

---

### POST /api/v1/workspaces/{id}/data-assets:registerLineage

**注册数据血缘**

在单个事务内注册 source、parsed、vector/output 数据资产，并建立 typed derivation 与 parsed manifest；未显式指定 asset_id 时按 asset_type+asset_ref 复用既有数据资产

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 请求体

类型: `handlers.registerLineageRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| derived_file_ids | []string |  | - |
| edge_provenance | object |  | - |
| embedding_model | string |  | - |
| image_distance_metric | string |  | - |
| image_embedding_backend_id | string |  | - |
| image_embedding_dimension | integer |  | - |
| image_embedding_model | string |  | - |
| image_index_file_status | handlers.registerLineageImageIndexFileStatus |  | - |
| image_preprocess_version | string |  | - |
| image_vector_table | string |  | - |
| output_file_id | string |  | - |
| parsed_file_id | string |  | - |
| runtime | handlers.registerLineageRuntime |  | - |
| semantic_model_ref_vector_table | string |  | - |
| source | string |  | - |
| source_file_id | string |  | - |
| source_file_name | string |  | - |
| vector_table | string |  | - |
| volume_id | integer |  | - |

示例:

```json
{
  "derived_file_ids": ["string"],
  "edge_provenance": {},
  "embedding_model": "string",
  "image_distance_metric": "string",
  "image_embedding_backend_id": "string",
  "image_embedding_dimension": 0,
  "image_embedding_model": "string",
  "image_index_file_status": {
    "indexed_images": 0,
    "source_file_id": "string",
    "status": "string"
  },
  "image_preprocess_version": "string",
  "image_vector_table": "string",
  "output_file_id": "string",
  "parsed_file_id": "string",
  "runtime": {
    "case_id": "string",
    "parallel_index": 0,
    "recorded_by_workitem_id": "string"
  },
  "semantic_model_ref_vector_table": "string",
  "source": "string",
  "source_file_id": "string",
  "source_file_name": "string",
  "vector_table": "string",
  "volume_id": 0
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 注册成功 | `handlers.registerLineageResponse` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`handlers.registerLineageResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| derivations | []catalog.DataDerivation |  | - |
| image_vector_asset | catalog.DataAsset |  | - |
| manifest | catalog.ParsedManifest |  | - |
| output_asset | catalog.DataAsset |  | - |
| parsed_asset | catalog.DataAsset |  | - |
| source_asset | catalog.DataAsset |  | - |
| vector_asset | catalog.DataAsset |  | - |

响应示例:

```json
{
  "derivations": [{
    "asset_id": "string",
    "case_id": "string",
    "created_at": 0,
    "file_id": "string",
    "id": 0,
    "idempotency_key": "string",
    "kind": "string",
    "logical_slot": "string",
    "meta": {
      "fields": {}
    },
    "parallel_index": 0,
    "producer_workitem_id": "string",
    "recorded_by_workitem_id": "string",
    "root_asset_id": "string",
    "source_asset_id": "string",
    "target_asset_id": "string",
    "updated_at": 0
  }],
  "image_vector_asset": {
    "asset_id": "string",
    "asset_ref": "string",
    "asset_type": "string",
    "created_at": 0,
    "id": 0,
    "meta": {
      "fields": {}
    },
    "name": "string",
    "raw_file_id": "string",
    "source": "string",
    "updated_at": 0,
    "volume_id": 0
  },
  "manifest": {
    "asset_id": "string",
    "created_at": 0,
    "id": 0,
    "manifest": {
      "fields": {}
    },
    "parsed_asset_id": "string",
    "parsed_file_id": "string",
    "raw_file_id": "string",
    "root_asset_id": "string",
    "source_file_id": "string",
    "updated_at": 0
  },
  "output_asset": {
    "asset_id": "string",
    "asset_ref": "string",
    "asset_type": "string",
    "created_at": 0,
    "id": 0,
    "meta": {
      "fields": {}
    },
    "name": "string",
    "raw_file_id": "string",
    "source": "string",
    "updated_at": 0,
    "volume_id": 0
  },
  "parsed_asset": {
    "asset_id": "string",
    "asset_ref": "string",
    "asset_type": "string",
    "created_at": 0,
    "id": 0,
    "meta": {
      "fields": {}
    },
    "name": "string",
    "raw_file_id": "string",
    "source": "string",
    "updated_at": 0,
    "volume_id": 0
  },
  "source_asset": {
    "asset_id": "string",
    "asset_ref": "string",
    "asset_type": "string",
    "created_at": 0,
    "id": 0,
    "meta": {
      "fields": {}
    },
    "name": "string",
    "raw_file_id": "string",
    "source": "string",
    "updated_at": 0,
    "volume_id": 0
  },
  "vector_asset": {
    "asset_id": "string",
    "asset_ref": "string",
    "asset_type": "string",
    "created_at": 0,
    "id": 0,
    "meta": {
      "fields": {}
    },
    "name": "string",
    "raw_file_id": "string",
    "source": "string",
    "updated_at": 0,
    "volume_id": 0
  }
}
```

---

## Data Share

### GET /api/v1/workspaces/{id}/data-share/publishes

**List Data Publish resources**

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 查询参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| keyword | string | 否 | Name or object keyword |
| page | integer | 否 | Page number |
| page_size | integer | 否 | Page size |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | OK | `catalog.ListDataSharePublicationsResponse` |
| 400 | Bad Request | `gin.ErrorResponse` |
| 403 | Forbidden | `gin.ErrorResponse` |

响应字段 (`catalog.ListDataSharePublicationsResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| list | []catalog.DataSharePublication |  | - |
| page | integer |  | - |
| page_size | integer |  | - |
| total | integer |  | - |

响应示例:

```json
{
  "list": [{
    "created_at": "string",
    "created_by": "string",
    "id": "string",
    "mo_database_name": "string",
    "name": "string",
    "object_display": {
      "catalog": {
        "comment": "string",
        "created_at": 0,
        "created_by": "string",
        "display_bindings": [{}],
        "id": 0,
        "name": "string",
        "updated_at": 0,
        "updated_by": "string"
      },
      "database": {
        "catalog_id": 0,
        "comment": "string",
        "created_at": 0,
        "created_by": "string",
        "details_visible": false,
        "display_bindings": [{}],
        "id": 0,
        "is_pub": false,
        "is_sub": false,
        "name": "string",
        "updated_at": 0,
        "updated_by": "string"
      }
    },
    "permission": "string",
    "remark": "string",
    "source_database_id": "string",
    "table_scope": {
      "mode": "string",
      "object_ids": ["string"]
    },
    "targets": [{
      "workspace_id": "string",
      "workspace_name": "string"
    }],
    "updated_at": "string"
  }],
  "page": 0,
  "page_size": 0,
  "total": 0
}
```

---

### POST /api/v1/workspaces/{id}/data-share/publishes

**Create a Data Publish resource**

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 请求体

类型: `catalog.CreateDataSharePublicationRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string |  | - |
| remark | string |  | - |
| source_database_id | string |  | - |
| table_scope |  |  | MatrixOne currently interprets an omitted TABLE list as all Tables.
Therefore selected requires at least one Table ID. |
| target_workspace_ids | []string |  | - |

示例:

```json
{
  "name": "string",
  "remark": "string",
  "source_database_id": "string",
  "table_scope": "",
  "target_workspace_ids": ["string"]
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | OK | `catalog.DataSharePublication` |
| 400 | Bad Request | `gin.ErrorResponse` |
| 403 | Forbidden | `gin.ErrorResponse` |
| 409 | Conflict | `gin.ErrorResponse` |

响应字段 (`catalog.DataSharePublication`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| created_at | string |  | - |
| created_by | string |  | - |
| id | string |  | - |
| mo_database_name | string |  | - |
| name | string |  | - |
| object_display | catalog.DataShareObjectDisplay |  | - |
| permission | string |  | - |
| remark | string |  | - |
| source_database_id | string |  | - |
| table_scope | catalog.DataShareObjectScope |  | - |
| targets | []catalog.DataSharePublicationTarget |  | - |
| updated_at | string |  | - |

响应示例:

```json
{
  "created_at": "string",
  "created_by": "string",
  "id": "string",
  "mo_database_name": "string",
  "name": "string",
  "object_display": {
    "catalog": {
      "comment": "string",
      "created_at": 0,
      "created_by": "string",
      "display_bindings": [{
        "default_text": "string",
        "display_key": "string",
        "display_owner": "string",
        "field": "string"
      }],
      "id": 0,
      "name": "string",
      "updated_at": 0,
      "updated_by": "string"
    },
    "database": {
      "catalog_id": 0,
      "comment": "string",
      "created_at": 0,
      "created_by": "string",
      "details_visible": false,
      "display_bindings": [{
        "default_text": "string",
        "display_key": "string",
        "display_owner": "string",
        "field": "string"
      }],
      "id": 0,
      "is_pub": false,
      "is_sub": false,
      "name": "string",
      "updated_at": 0,
      "updated_by": "string"
    }
  },
  "permission": "string",
  "remark": "string",
  "source_database_id": "string",
  "table_scope": {
    "mode": "string",
    "object_ids": ["string"]
  },
  "targets": [{
    "workspace_id": "string",
    "workspace_name": "string"
  }],
  "updated_at": "string"
}
```

---

### GET /api/v1/workspaces/{id}/data-share/publishes/check-name

**Check Data Publish name availability**

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 查询参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 是 | Publication name |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | OK | `catalog.DataShareNameAvailability` |
| 400 | Bad Request | `gin.ErrorResponse` |
| 403 | Forbidden | `gin.ErrorResponse` |

响应字段 (`catalog.DataShareNameAvailability`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| available | boolean |  | - |

响应示例:

```json
{
  "available": false
}
```

---

### GET /api/v1/workspaces/{id}/data-share/publishes/summary

**Summarize Data Publish resources**

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | OK | `catalog.DataSharePublicationSummary` |
| 403 | Forbidden | `gin.ErrorResponse` |

响应字段 (`catalog.DataSharePublicationSummary`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| target_total | integer |  | - |
| total | integer |  | - |

响应示例:

```json
{
  "target_total": 0,
  "total": 0
}
```

---

### PUT /api/v1/workspaces/{id}/data-share/publishes/{data_share_id}

**Update a Data Publish resource**

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| data_share_id | integer | 是 | Publication ID |

#### 请求体

类型: `catalog.UpdateDataSharePublicationRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| remark | catalog.DataShareStringValue |  | - |
| table_scope |  |  | When present, selected requires at least one Table ID. |
| target_workspace_ids | catalog.DataShareStringList |  | - |

示例:

```json
{
  "remark": {
    "value": "string"
  },
  "table_scope": "",
  "target_workspace_ids": {
    "values": ["string"]
  }
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | OK | `catalog.DataSharePublication` |
| 400 | Bad Request | `gin.ErrorResponse` |
| 403 | Forbidden | `gin.ErrorResponse` |
| 409 | Conflict | `gin.ErrorResponse` |

响应字段 (`catalog.DataSharePublication`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| created_at | string |  | - |
| created_by | string |  | - |
| id | string |  | - |
| mo_database_name | string |  | - |
| name | string |  | - |
| object_display | catalog.DataShareObjectDisplay |  | - |
| permission | string |  | - |
| remark | string |  | - |
| source_database_id | string |  | - |
| table_scope | catalog.DataShareObjectScope |  | - |
| targets | []catalog.DataSharePublicationTarget |  | - |
| updated_at | string |  | - |

响应示例:

```json
{
  "created_at": "string",
  "created_by": "string",
  "id": "string",
  "mo_database_name": "string",
  "name": "string",
  "object_display": {
    "catalog": {
      "comment": "string",
      "created_at": 0,
      "created_by": "string",
      "display_bindings": [{
        "default_text": "string",
        "display_key": "string",
        "display_owner": "string",
        "field": "string"
      }],
      "id": 0,
      "name": "string",
      "updated_at": 0,
      "updated_by": "string"
    },
    "database": {
      "catalog_id": 0,
      "comment": "string",
      "created_at": 0,
      "created_by": "string",
      "details_visible": false,
      "display_bindings": [{
        "default_text": "string",
        "display_key": "string",
        "display_owner": "string",
        "field": "string"
      }],
      "id": 0,
      "is_pub": false,
      "is_sub": false,
      "name": "string",
      "updated_at": 0,
      "updated_by": "string"
    }
  },
  "permission": "string",
  "remark": "string",
  "source_database_id": "string",
  "table_scope": {
    "mode": "string",
    "object_ids": ["string"]
  },
  "targets": [{
    "workspace_id": "string",
    "workspace_name": "string"
  }],
  "updated_at": "string"
}
```

---

### DELETE /api/v1/workspaces/{id}/data-share/publishes/{data_share_id}

**Delete a Data Publish resource**

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| data_share_id | integer | 是 | Publication ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 204 | No Content |  |
| 400 | Bad Request | `gin.ErrorResponse` |
| 403 | Forbidden | `gin.ErrorResponse` |
| 409 | Conflict | `gin.ErrorResponse` |

---

### GET /api/v1/workspaces/{id}/data-share/source-databases/{source_database_id}/tables

**List publishable Data Share source Tables**

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| source_database_id | integer | 是 | Source Database ID |

#### 查询参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | integer | 否 | Page number |
| page_size | integer | 否 | Page size |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | OK | `catalog.ListDataShareSourceTablesResponse` |
| 400 | Bad Request | `gin.ErrorResponse` |
| 403 | Forbidden | `gin.ErrorResponse` |
| 404 | Not Found | `gin.ErrorResponse` |

响应字段 (`catalog.ListDataShareSourceTablesResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| list | []catalog.DataShareSourceTable |  | - |
| page | integer |  | - |
| page_size | integer |  | - |
| total | integer |  | - |

响应示例:

```json
{
  "list": [{
    "id": "string",
    "name": "string"
  }],
  "page": 0,
  "page_size": 0,
  "total": 0
}
```

---

### GET /api/v1/workspaces/{id}/data-share/subscriptions

**List Data Subscriptions**

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 查询参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| keyword | string | 否 | Publication or subscription keyword |
| status | string | 否 | Subscription status |
| page | integer | 否 | Page number |
| page_size | integer | 否 | Page size |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | OK | `catalog.ListDataShareSubscriptionsResponse` |
| 400 | Bad Request | `gin.ErrorResponse` |
| 403 | Forbidden | `gin.ErrorResponse` |

响应字段 (`catalog.ListDataShareSubscriptionsResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| list | []catalog.DataShareSubscription |  | - |
| page | integer |  | - |
| page_size | integer |  | - |
| total | integer |  | - |

响应示例:

```json
{
  "list": [{
    "created_at": "string",
    "id": "string",
    "mo_database_name": "string",
    "object_display": {
      "catalog": {
        "comment": "string",
        "created_at": 0,
        "created_by": "string",
        "display_bindings": [{}],
        "id": 0,
        "name": "string",
        "updated_at": 0,
        "updated_by": "string"
      },
      "database": {
        "catalog_id": 0,
        "comment": "string",
        "created_at": 0,
        "created_by": "string",
        "details_visible": false,
        "display_bindings": [{}],
        "id": 0,
        "is_pub": false,
        "is_sub": false,
        "name": "string",
        "updated_at": 0,
        "updated_by": "string"
      }
    },
    "pub_name": "string",
    "published_at": "string",
    "publisher": "string",
    "source_database_id": "string",
    "source_workspace_id": "string",
    "source_workspace_name": "string",
    "status": "string",
    "sub_name": "string",
    "subscribed_by": "string",
    "table_scope": {
      "mode": "string",
      "object_ids": ["string"]
    },
    "target_database_id": "string"
  }],
  "page": 0,
  "page_size": 0,
  "total": 0
}
```

---

### POST /api/v1/workspaces/{id}/data-share/subscriptions/{data_share_id}/subscribe

**Accept a Data Subscription invitation**

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| data_share_id | integer | 是 | Subscription ID |

#### 请求体

类型: `catalog.AcceptDataShareSubscriptionRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| sub_name | string |  | - |

示例:

```json
{
  "sub_name": "string"
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | OK | `catalog.DataShareSubscription` |
| 400 | Bad Request | `gin.ErrorResponse` |
| 403 | Forbidden | `gin.ErrorResponse` |
| 409 | Conflict | `gin.ErrorResponse` |

响应字段 (`catalog.DataShareSubscription`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| created_at | string |  | - |
| id | string |  | - |
| mo_database_name | string |  | - |
| object_display | catalog.DataShareObjectDisplay |  | - |
| pub_name | string |  | - |
| published_at | string |  | - |
| publisher | string |  | - |
| source_database_id | string |  | - |
| source_workspace_id | string |  | - |
| source_workspace_name | string |  | - |
| status | string |  | - |
| sub_name | string |  | - |
| subscribed_by | string |  | - |
| table_scope | catalog.DataShareObjectScope |  | - |
| target_database_id | string |  | - |

响应示例:

```json
{
  "created_at": "string",
  "id": "string",
  "mo_database_name": "string",
  "object_display": {
    "catalog": {
      "comment": "string",
      "created_at": 0,
      "created_by": "string",
      "display_bindings": [{
        "default_text": "string",
        "display_key": "string",
        "display_owner": "string",
        "field": "string"
      }],
      "id": 0,
      "name": "string",
      "updated_at": 0,
      "updated_by": "string"
    },
    "database": {
      "catalog_id": 0,
      "comment": "string",
      "created_at": 0,
      "created_by": "string",
      "details_visible": false,
      "display_bindings": [{
        "default_text": "string",
        "display_key": "string",
        "display_owner": "string",
        "field": "string"
      }],
      "id": 0,
      "is_pub": false,
      "is_sub": false,
      "name": "string",
      "updated_at": 0,
      "updated_by": "string"
    }
  },
  "pub_name": "string",
  "published_at": "string",
  "publisher": "string",
  "source_database_id": "string",
  "source_workspace_id": "string",
  "source_workspace_name": "string",
  "status": "string",
  "sub_name": "string",
  "subscribed_by": "string",
  "table_scope": {
    "mode": "string",
    "object_ids": ["string"]
  },
  "target_database_id": "string"
}
```

---

### POST /api/v1/workspaces/{id}/data-share/subscriptions/{data_share_id}/unsubscribe

**Delete an active Data Subscription**

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| data_share_id | integer | 是 | Subscription ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | OK | `catalog.DataShareSubscription` |
| 400 | Bad Request | `gin.ErrorResponse` |
| 403 | Forbidden | `gin.ErrorResponse` |
| 409 | Conflict | `gin.ErrorResponse` |

响应字段 (`catalog.DataShareSubscription`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| created_at | string |  | - |
| id | string |  | - |
| mo_database_name | string |  | - |
| object_display | catalog.DataShareObjectDisplay |  | - |
| pub_name | string |  | - |
| published_at | string |  | - |
| publisher | string |  | - |
| source_database_id | string |  | - |
| source_workspace_id | string |  | - |
| source_workspace_name | string |  | - |
| status | string |  | - |
| sub_name | string |  | - |
| subscribed_by | string |  | - |
| table_scope | catalog.DataShareObjectScope |  | - |
| target_database_id | string |  | - |

响应示例:

```json
{
  "created_at": "string",
  "id": "string",
  "mo_database_name": "string",
  "object_display": {
    "catalog": {
      "comment": "string",
      "created_at": 0,
      "created_by": "string",
      "display_bindings": [{
        "default_text": "string",
        "display_key": "string",
        "display_owner": "string",
        "field": "string"
      }],
      "id": 0,
      "name": "string",
      "updated_at": 0,
      "updated_by": "string"
    },
    "database": {
      "catalog_id": 0,
      "comment": "string",
      "created_at": 0,
      "created_by": "string",
      "details_visible": false,
      "display_bindings": [{
        "default_text": "string",
        "display_key": "string",
        "display_owner": "string",
        "field": "string"
      }],
      "id": 0,
      "is_pub": false,
      "is_sub": false,
      "name": "string",
      "updated_at": 0,
      "updated_by": "string"
    }
  },
  "pub_name": "string",
  "published_at": "string",
  "publisher": "string",
  "source_database_id": "string",
  "source_workspace_id": "string",
  "source_workspace_name": "string",
  "status": "string",
  "sub_name": "string",
  "subscribed_by": "string",
  "table_scope": {
    "mode": "string",
    "object_ids": ["string"]
  },
  "target_database_id": "string"
}
```

---

## Database 管理

### GET /api/v1/workspaces/{id}/catalogs/{catalog_id}/databases

**列出 Catalog 下的数据库**

列出指定 Catalog 下的所有数据库，支持分页

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| catalog_id | integer | 是 | Catalog ID |

#### 查询参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page_size | integer | 否 | 每页数量（默认 20，最大 100） |
| page_token | string | 否 | 分页令牌 |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 数据库列表 | `github_com_matrixflow_moi-core_model_catalog.ListDatabasesResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 404 | Catalog 不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`github_com_matrixflow_moi-core_model_catalog.ListDatabasesResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| items | []catalog.Database |  | - |
| next_page_token | string |  | - |
| total | integer |  | - |

响应示例:

```json
{
  "items": [{
    "catalog_id": 0,
    "comment": "string",
    "created_at": 0,
    "created_by": "string",
    "details_visible": false,
    "display_bindings": [{
      "default_text": "string",
      "display_key": "string",
      "display_owner": "string",
      "field": "string"
    }],
    "id": 0,
    "is_pub": false,
    "is_sub": false,
    "name": "string",
    "updated_at": 0,
    "updated_by": "string"
  }],
  "next_page_token": "string",
  "total": 0
}
```

---

### GET /api/v1/workspaces/{id}/databases/{database_id}

**获取数据库**

根据 ID 获取数据库元数据详情

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| database_id | integer | 是 | Database ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 数据库详情 | `catalog.Database` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 404 | 数据库不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.Database`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| catalog_id | integer |  | - |
| comment | string |  | - |
| created_at | integer |  | Unix 时间戳（秒） |
| created_by | string |  | 创建者用户 ID |
| details_visible | boolean |  | True only when the caller has direct database visibility, rather than
receiving this database as a minimal ancestor for an authorized child. |
| display_bindings | []catalog.ResourceDisplayBinding |  | - |
| id | integer |  | - |
| is_pub | boolean |  | Set by Catalog display projections when this database has an active Data
Share publication. |
| is_sub | boolean |  | Set by Catalog display projections when this database is an active Data
Share subscription projection. |
| name | string |  | - |
| updated_at | integer |  | Unix 时间戳（秒） |
| updated_by | string |  | 更新者用户 ID |

响应示例:

```json
{
  "catalog_id": 0,
  "comment": "string",
  "created_at": 0,
  "created_by": "string",
  "details_visible": false,
  "display_bindings": [{
    "default_text": "string",
    "display_key": "string",
    "display_owner": "string",
    "field": "string"
  }],
  "id": 0,
  "is_pub": false,
  "is_sub": false,
  "name": "string",
  "updated_at": 0,
  "updated_by": "string"
}
```

---

### GET /api/v1/workspaces/{id}/databases/{database_id}/children

**列出 Database 直接子节点**

返回当前调用者可见的 Table 和根 Volume 轻量列表，不含详情或统计

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| database_id | integer | 是 | Database ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | Database 直接子节点 | `catalog.ListDatabaseChildrenResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.ListDatabaseChildrenResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| items | []catalog.DatabaseChild |  | - |

响应示例:

```json
{
  "items": [{
    "comment": "string",
    "display_bindings": [{
      "default_text": "string",
      "display_key": "string",
      "display_owner": "string",
      "field": "string"
    }],
    "id": 0,
    "name": "string",
    "type": "string"
  }]
}
```

---

### GET /api/v1/workspaces/{id}/databases/{database_id}/stats

**获取 Database 统计信息**

返回指定 Database 下的 table、volume、file 数量汇总

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| database_id | integer | 是 | Database ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 统计信息 | `handlers.DatabaseStatsResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 404 | 不存在 | `gin.ErrorResponse` |
| 503 | 数据库连接失败 | `gin.ErrorResponse` |

响应字段 (`handlers.DatabaseStatsResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| file_count | integer |  | - |
| table_count | integer |  | - |
| volume_count | integer |  | - |

响应示例:

```json
{
  "file_count": 0,
  "table_count": 0,
  "volume_count": 0
}
```

---

### GET /api/v1/workspaces/{id}/databases/{database_id}/tables

**列出数据库下的表**

列出指定数据库下的所有表，支持分页

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| database_id | integer | 是 | Database ID |

#### 查询参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page_size | integer | 否 | 每页数量（默认 20，最大 100） |
| page_token | string | 否 | 分页令牌 |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 表列表 | `catalog.ListTablesResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 404 | 数据库不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.ListTablesResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| items | []catalog.Table |  | - |
| next_page_token | string |  | - |
| total | integer |  | - |

响应示例:

```json
{
  "items": [{
    "catalog_id": 0,
    "comment": "string",
    "created_at": 0,
    "created_by": "string",
    "database_id": 0,
    "extensions": {},
    "id": 0,
    "name": "string",
    "updated_at": 0,
    "updated_by": "string"
  }],
  "next_page_token": "string",
  "total": 0
}
```

---

### POST /api/v1/workspaces/{id}/metadata/sync

**同步数据库元数据**

从 MatrixOne 同步数据库和表的元数据到 moi-core。用户创建 Database 或 Table 的同步意图会校验新名称；普通后台发现保留 MatrixOne 中已有的历史名称

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 请求体

类型: `catalog.SyncMetadataRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| catalog_id | integer |  | 要关联的 Catalog ID |
| comment | string |  | 数据库描述（可选，nil=不修改，""=清空） |
| database_name | string |  | MatrixOne 中的数据库名称 |
| table_create_database_id | integer |  | 待 Core 按 name/catalog 复核的父 Database ID |
| table_create_name | string |  | 用户本次刚创建的表；仅用于校验并登记粗粒度 Table owner |

示例:

```json
{
  "catalog_id": 0,
  "comment": "string",
  "database_name": "string",
  "table_create_database_id": 0,
  "table_create_name": "string"
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 同步结果 | `catalog.SyncMetadataResponse` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 403 | 权限不足 | `gin.ErrorResponse` |
| 404 | 数据库或 Catalog 不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.SyncMetadataResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| database |  |  | 同步后的数据库元数据 |
| tables | []catalog.Table |  | 同步后的表元数据列表 |
| tables_synced | integer |  | 同步的表数量 |
| tables_updated | integer |  | 更新的表数量（已存在的表） |

响应示例:

```json
{
  "database": "",
  "tables": [{
    "catalog_id": 0,
    "comment": "string",
    "created_at": 0,
    "created_by": "string",
    "database_id": 0,
    "extensions": {},
    "id": 0,
    "name": "string",
    "updated_at": 0,
    "updated_by": "string"
  }],
  "tables_synced": 0,
  "tables_updated": 0
}
```

---

### GET /api/v1/workspaces/{id}/tables/{table_id}

**获取表**

根据 ID 获取单表元数据详情。该接口支持表对象权限校验，不要求调用方具备数据库/目录列表权限。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| table_id | integer | 是 | Table ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 表详情 | `catalog.GetTableResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 403 | 无权限 | `gin.ErrorResponse` |
| 404 | 表不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.GetTableResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| catalog | catalog.Catalog |  | - |
| database | catalog.Database |  | - |
| table | catalog.Table |  | - |

响应示例:

```json
{
  "catalog": {
    "comment": "string",
    "created_at": 0,
    "created_by": "string",
    "display_bindings": [{
      "default_text": "string",
      "display_key": "string",
      "display_owner": "string",
      "field": "string"
    }],
    "id": 0,
    "name": "string",
    "updated_at": 0,
    "updated_by": "string"
  },
  "database": {
    "catalog_id": 0,
    "comment": "string",
    "created_at": 0,
    "created_by": "string",
    "details_visible": false,
    "display_bindings": [{
      "default_text": "string",
      "display_key": "string",
      "display_owner": "string",
      "field": "string"
    }],
    "id": 0,
    "is_pub": false,
    "is_sub": false,
    "name": "string",
    "updated_at": 0,
    "updated_by": "string"
  },
  "table": {
    "catalog_id": 0,
    "comment": "string",
    "created_at": 0,
    "created_by": "string",
    "database_id": 0,
    "extensions": {},
    "id": 0,
    "name": "string",
    "updated_at": 0,
    "updated_by": "string"
  }
}
```

---

## Dataphin 元数据管理

### GET /api/v1/workspaces/{id}/dataphin/configs/{config_id}/databases

**列出 Dataphin 数据库**

列出指定 Dataphin 配置下的所有数据库（项目），支持分页

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| config_id | integer | 是 | Dataphin 配置 ID |

#### 查询参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page_size | integer | 否 | 每页数量（默认 20，最大 100） |
| page_token | string | 否 | 分页令牌 |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 数据库列表 | `catalog.ListDPDatabasesResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.ListDPDatabasesResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| items | []catalog.DPDatabase |  | - |
| next_page_token | string |  | - |
| total | integer |  | - |

响应示例:

```json
{
  "items": [{
    "comment": "string",
    "config_id": 0,
    "id": 0,
    "name": "string",
    "source": "string",
    "synced_at": 0
  }],
  "next_page_token": "string",
  "total": 0
}
```

---

### GET /api/v1/workspaces/{id}/dataphin/configs/{config_id}/databases/{database_id}

**获取 Dataphin 数据库**

获取指定 Dataphin 数据库（项目）的详细信息

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| config_id | integer | 是 | Dataphin 配置 ID |
| database_id | integer | 是 | Database ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 数据库详情 | `catalog.DPDatabase` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 404 | 不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.DPDatabase`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| comment | string |  | 项目描述 |
| config_id | integer |  | 关联的 Dataphin 配置 ID |
| id | integer |  | - |
| name | string |  | 项目名称 |
| source | string |  | 数据源标识，固定为 "dataphin" |
| synced_at | integer |  | 最后同步时间（Unix 时间戳，秒） |

响应示例:

```json
{
  "comment": "string",
  "config_id": 0,
  "id": 0,
  "name": "string",
  "source": "string",
  "synced_at": 0
}
```

---

### GET /api/v1/workspaces/{id}/dataphin/configs/{config_id}/databases/{database_id}/tables

**列出 Dataphin 表**

列出指定 Dataphin 数据库下的所有表，支持分页

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| config_id | integer | 是 | Dataphin 配置 ID |
| database_id | integer | 是 | Database ID |

#### 查询参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page_size | integer | 否 | 每页数量（默认 20，最大 100） |
| page_token | string | 否 | 分页令牌 |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 表列表 | `catalog.ListDPTablesResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.ListDPTablesResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| items | []catalog.DPTable |  | - |
| next_page_token | string |  | - |
| total | integer |  | - |

响应示例:

```json
{
  "items": [{
    "columns": [{
      "comment": "string",
      "data_type": "string",
      "id": 0,
      "name": "string",
      "ordinal": 0
    }],
    "comment": "string",
    "config_id": 0,
    "database_id": 0,
    "description": "string",
    "id": 0,
    "name": "string",
    "table_type": "string"
  }],
  "next_page_token": "string",
  "total": 0
}
```

---

### GET /api/v1/workspaces/{id}/dataphin/configs/{config_id}/databases/{database_id}/tables/{table_id}

**获取 Dataphin 表**

获取指定 Dataphin 表的详细信息（包含列信息）

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| config_id | integer | 是 | Dataphin 配置 ID |
| database_id | integer | 是 | Database ID |
| table_id | integer | 是 | Table ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 表详情 | `catalog.DPTable` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 404 | 不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.DPTable`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| columns | []catalog.DPColumn |  | 列信息（GetTable 时返回） |
| comment | string |  | 表描述 |
| config_id | integer |  | 关联的 Dataphin 配置 ID |
| database_id | integer |  | 所属数据库（项目）ID |
| description | string |  | 表描述信息 |
| id | integer |  | - |
| name | string |  | 表名称 |
| table_type | string |  | 表类型：维度表、事实表、汇总表、拉链表等 |

响应示例:

```json
{
  "columns": [{
    "comment": "string",
    "data_type": "string",
    "id": 0,
    "name": "string",
    "ordinal": 0
  }],
  "comment": "string",
  "config_id": 0,
  "database_id": 0,
  "description": "string",
  "id": 0,
  "name": "string",
  "table_type": "string"
}
```

---

### GET /api/v1/workspaces/{id}/dataphin/configs/{config_id}/health

**Dataphin 连接健康检查**

检查指定 Dataphin 配置的连接是否正常

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| config_id | integer | 是 | Dataphin 配置 ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 健康状态 | `object` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

---

### POST /api/v1/workspaces/{id}/dataphin/configs/{config_id}/stop-sync

**停止 Dataphin 同步**

取消指定 Dataphin 配置的周期性同步工作流

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| config_id | integer | 是 | Dataphin 配置 ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 204 | 停止成功 |  |
| 401 | 未认证 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

---

### POST /api/v1/workspaces/{id}/dataphin/configs/{config_id}/sync

**同步 Dataphin 元数据**

创建周期性同步 Dataphin 元数据的工作流

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| config_id | integer | 是 | Dataphin 配置 ID |

#### 请求体

类型: `catalog.SyncDPMetadataRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| cron_expression | string |  | cron 表达式，如 "0 */30 * * * *"（每30分钟） |
| project_name | string |  | 要同步的 Dataphin 项目名称 |

示例:

```json
{
  "cron_expression": "string",
  "project_name": "string"
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 同步结果 | `catalog.SyncDPMetadataResponse` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.SyncDPMetadataResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| case_id | string |  | 本次主动触发的执行ID |
| database |  |  | 同步后的数据库（项目）元数据 |
| tables_deleted | integer |  | 删除的表数量（Dataphin 中已不存在） |
| tables_synced | integer |  | 新同步的表数量 |
| tables_updated | integer |  | 更新的表数量 |
| task_id | string |  | 触发同步的任务ID（定时任务） |

响应示例:

```json
{
  "case_id": "string",
  "database": "",
  "tables_deleted": 0,
  "tables_synced": 0,
  "tables_updated": 0,
  "task_id": "string"
}
```

---

## Dataphin 配置管理

### GET /api/v1/workspaces/{id}/dataphin/configs

**列出 Dataphin 配置**

分页列出工作区内的 Dataphin 配置

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 查询参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page_size | integer | 否 | 每页数量 |
| page_token | string | 否 | 分页令牌 |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 配置列表 | `catalog.ListDPConfigsResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.ListDPConfigsResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| items | []catalog.DPConfig |  | - |
| next_page_token | string |  | - |
| total | integer |  | - |

响应示例:

```json
{
  "items": [{
    "access_key_id": "string",
    "created_at": 0,
    "created_by": "string",
    "endpoint": "string",
    "id": 0,
    "name": "string",
    "project_name": "string",
    "region": "string",
    "sync_cron_expression": "string",
    "sync_database_name": "string",
    "sync_task_id": "string",
    "updated_at": 0,
    "updated_by": "string"
  }],
  "next_page_token": "string",
  "total": 0
}
```

---

### POST /api/v1/workspaces/{id}/dataphin/configs

**创建 Dataphin 配置**

创建新的 Dataphin 连接配置

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 请求体

类型: `dataphin.CreateDPConfigRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| access_key_id | string |  | 阿里云 AccessKeyId，必填 |
| access_key_secret | string |  | 阿里云 AccessKeySecret，必填 |
| endpoint | string |  | Dataphin Endpoint，必填 |
| name | string |  | 配置名称，必填 |
| project_name | string |  | Dataphin 项目名称（可选） |
| region | string |  | 阿里云 Region（可选） |

示例:

```json
{
  "access_key_id": "string",
  "access_key_secret": "string",
  "endpoint": "string",
  "name": "string",
  "project_name": "string",
  "region": "string"
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 201 | 创建的配置 | `catalog.DPConfig` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 409 | 名称重复 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.DPConfig`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| access_key_id | string |  | 阿里云 AccessKeyId |
| created_at | integer |  | Unix 时间戳（秒） |
| created_by | string |  | 创建者用户 ID |
| endpoint | string |  | 注意：不包含 access_key_secret，避免在 API 响应中泄露敏感凭证 |
| id | integer |  | - |
| name | string |  | 配置名称，workspace 内唯一 |
| project_name | string |  | Dataphin 项目名称（可选） |
| region | string |  | 阿里云 Region（可选） |
| sync_cron_expression | string |  | 同步 cron 表达式 |
| sync_database_name | string |  | 正在同步的数据库名称 |
| sync_task_id | string |  | 周期性同步任务 ID（空表示未启动同步） |
| updated_at | integer |  | Unix 时间戳（秒） |
| updated_by | string |  | 更新者用户 ID |

响应示例:

```json
{
  "access_key_id": "string",
  "created_at": 0,
  "created_by": "string",
  "endpoint": "string",
  "id": 0,
  "name": "string",
  "project_name": "string",
  "region": "string",
  "sync_cron_expression": "string",
  "sync_database_name": "string",
  "sync_task_id": "string",
  "updated_at": 0,
  "updated_by": "string"
}
```

---

### GET /api/v1/workspaces/{id}/dataphin/configs/{config_id}

**获取 Dataphin 配置**

根据 ID 获取 Dataphin 配置详情

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| config_id | integer | 是 | Dataphin 配置 ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 配置详情 | `catalog.DPConfig` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 404 | 不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.DPConfig`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| access_key_id | string |  | 阿里云 AccessKeyId |
| created_at | integer |  | Unix 时间戳（秒） |
| created_by | string |  | 创建者用户 ID |
| endpoint | string |  | 注意：不包含 access_key_secret，避免在 API 响应中泄露敏感凭证 |
| id | integer |  | - |
| name | string |  | 配置名称，workspace 内唯一 |
| project_name | string |  | Dataphin 项目名称（可选） |
| region | string |  | 阿里云 Region（可选） |
| sync_cron_expression | string |  | 同步 cron 表达式 |
| sync_database_name | string |  | 正在同步的数据库名称 |
| sync_task_id | string |  | 周期性同步任务 ID（空表示未启动同步） |
| updated_at | integer |  | Unix 时间戳（秒） |
| updated_by | string |  | 更新者用户 ID |

响应示例:

```json
{
  "access_key_id": "string",
  "created_at": 0,
  "created_by": "string",
  "endpoint": "string",
  "id": 0,
  "name": "string",
  "project_name": "string",
  "region": "string",
  "sync_cron_expression": "string",
  "sync_database_name": "string",
  "sync_task_id": "string",
  "updated_at": 0,
  "updated_by": "string"
}
```

---

### PUT /api/v1/workspaces/{id}/dataphin/configs/{config_id}

**更新 Dataphin 配置**

更新指定 Dataphin 配置的属性

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| config_id | integer | 是 | Dataphin 配置 ID |

#### 请求体

类型: `dataphin.UpdateDPConfigRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| access_key_id | string |  | - |
| access_key_secret | string |  | - |
| endpoint | string |  | - |
| name | string |  | - |
| project_name | string |  | - |
| region | string |  | - |

示例:

```json
{
  "access_key_id": "string",
  "access_key_secret": "string",
  "endpoint": "string",
  "name": "string",
  "project_name": "string",
  "region": "string"
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 更新后的配置 | `catalog.DPConfig` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 404 | 不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.DPConfig`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| access_key_id | string |  | 阿里云 AccessKeyId |
| created_at | integer |  | Unix 时间戳（秒） |
| created_by | string |  | 创建者用户 ID |
| endpoint | string |  | 注意：不包含 access_key_secret，避免在 API 响应中泄露敏感凭证 |
| id | integer |  | - |
| name | string |  | 配置名称，workspace 内唯一 |
| project_name | string |  | Dataphin 项目名称（可选） |
| region | string |  | 阿里云 Region（可选） |
| sync_cron_expression | string |  | 同步 cron 表达式 |
| sync_database_name | string |  | 正在同步的数据库名称 |
| sync_task_id | string |  | 周期性同步任务 ID（空表示未启动同步） |
| updated_at | integer |  | Unix 时间戳（秒） |
| updated_by | string |  | 更新者用户 ID |

响应示例:

```json
{
  "access_key_id": "string",
  "created_at": 0,
  "created_by": "string",
  "endpoint": "string",
  "id": 0,
  "name": "string",
  "project_name": "string",
  "region": "string",
  "sync_cron_expression": "string",
  "sync_database_name": "string",
  "sync_task_id": "string",
  "updated_at": 0,
  "updated_by": "string"
}
```

---

### DELETE /api/v1/workspaces/{id}/dataphin/configs/{config_id}

**删除 Dataphin 配置**

删除指定 Dataphin 配置，级联删除所有关联元数据

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| config_id | integer | 是 | Dataphin 配置 ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 204 | 删除成功 |  |
| 401 | 未认证 | `gin.ErrorResponse` |
| 404 | 不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

---

## Dynamic Service 管理

### GET /api/v1/workspaces/{id}/dynamic-services

**列出 Dynamic Service**

列出指定 workspace 中的所有 Dynamic Service（当前由 Task API 处理，此接口返回 501）

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 服务列表 | `object` |
| 501 | 未实现 | `gin.ErrorResponse` |

---

### POST /api/v1/workspaces/{id}/dynamic-services

**创建 Dynamic Service**

创建新的 Dynamic Service（当前由 Task API 处理，此接口返回 501）

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 201 | 创建成功 | `object` |
| 501 | 未实现 | `gin.ErrorResponse` |

---

### POST /api/v1/workspaces/{id}/dynamic-services/invoke

**调用 Dynamic Service**

调用指定的 Dynamic Service；命中 Agent Automation service name 时返回任务运行记录

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 请求体

类型: JSON 对象

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 普通 Dynamic Service 调用成功 | `object` |
| 202 | Agent Automation 任务已接收 | `object` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 404 | 服务不存在 | `gin.ErrorResponse` |
| 409 | 服务未就绪 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

---

### GET /api/v1/workspaces/{id}/dynamic-services/{service_id}

**获取 Dynamic Service**

根据 ID 获取指定 Dynamic Service 的详细信息（当前由 Task API 处理，此接口返回 501）

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| service_id | string | 是 | Dynamic Service ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 服务详情 | `object` |
| 501 | 未实现 | `gin.ErrorResponse` |

---

### DELETE /api/v1/workspaces/{id}/dynamic-services/{service_id}

**删除 Dynamic Service**

删除指定的 Dynamic Service（当前由 Task API 处理，此接口返回 501）

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| service_id | string | 是 | Dynamic Service ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 204 | 删除成功 |  |
| 501 | 未实现 | `gin.ErrorResponse` |

---

### POST /api/v1/workspaces/{id}/dynamic-services/{service_id}/start

**启动 Dynamic Service**

启动指定的 Dynamic Service（当前未实现，返回 501）

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| service_id | string | 是 | Dynamic Service ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 启动成功 | `object` |
| 501 | 未实现 | `gin.ErrorResponse` |

---

### POST /api/v1/workspaces/{id}/dynamic-services/{service_id}/stop

**停止 Dynamic Service**

停止指定的 Dynamic Service（当前未实现，返回 501）

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| service_id | string | 是 | Dynamic Service ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 停止成功 | `object` |
| 501 | 未实现 | `gin.ErrorResponse` |

---

## Embedding 管理

### POST /api/v1/workspaces/{id}/embeddings

**执行 Embedding 请求**

发送 OpenAI 兼容的 Embedding 请求，将文本转换为向量表示。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 请求体

类型: JSON 对象

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | Embedding 响应 | `object` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |
| 503 | 无可用后端 | `gin.ErrorResponse` |

---

### GET /api/v1/workspaces/{id}/embeddings/backends

**获取 Embedding 后端列表**

获取指定 workspace 中所有 Embedding 后端配置。需要 PERM_MODEL_RESOURCE_READ（兼容旧 PERM_LLM_INVOKE）或 workspace 管理员权限；响应会脱敏 api_key_encrypted。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 获取成功 | `catalog.ListBackendsResponse` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.ListBackendsResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| backends | []catalog.Backend |  | - |
| total | integer |  | - |

响应示例:

```json
{
  "backends": [{
    "api_key_encrypted": "string",
    "created_at": 0,
    "id": 0,
    "models": ["string"],
    "name": "string",
    "origin_ref": "string",
    "origin_version": "string",
    "provider_origin": "string",
    "reasoning_control_protocol": {},
    "timeout_seconds": 0,
    "type": {},
    "updated_at": 0
  }],
  "total": 0
}
```

---

### POST /api/v1/workspaces/{id}/embeddings/backends

**创建 Embedding 后端**

在指定 workspace 中创建新的 Embedding 后端配置

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 请求体

类型: `catalog.CreateBackendRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| api_key_encrypted | string |  | - |
| models | []string |  | - |
| name | string |  | - |
| origin_ref | string |  | - |
| origin_version | string |  | - |
| provider_origin | string |  | - |
| reasoning_control_protocol | catalog.ReasoningControlProtocol |  | - |
| timeout_seconds | integer |  | - |
| type | catalog.BackendType |  | - |

示例:

```json
{
  "api_key_encrypted": "string",
  "models": ["string"],
  "name": "string",
  "origin_ref": "string",
  "origin_version": "string",
  "provider_origin": "string",
  "reasoning_control_protocol": {},
  "timeout_seconds": 0,
  "type": {}
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 201 | 创建成功 | `catalog.Backend` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.Backend`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| api_key_encrypted | string |  | 可选，加密存储 |
| created_at | integer |  | - |
| id | integer |  | - |
| models | []string |  | - |
| name | string |  | - |
| origin_ref | string |  | - |
| origin_version | string |  | - |
| provider_origin | string |  | provider_origin is a trusted billing classification. Tenant-created
backends are external_provider; Genesis is assigned only by a trusted
integration or migration, never inferred from a provider/model string. |
| reasoning_control_protocol | catalog.ReasoningControlProtocol |  | - |
| timeout_seconds | integer |  | - |
| type | catalog.BackendType |  | - |
| updated_at | integer |  | - |

响应示例:

```json
{
  "api_key_encrypted": "string",
  "created_at": 0,
  "id": 0,
  "models": ["string"],
  "name": "string",
  "origin_ref": "string",
  "origin_version": "string",
  "provider_origin": "string",
  "reasoning_control_protocol": {},
  "timeout_seconds": 0,
  "type": {},
  "updated_at": 0
}
```

---

### GET /api/v1/workspaces/{id}/embeddings/backends/{backend_id}

**获取 Embedding 后端详情**

根据后端 ID 获取指定 Embedding 后端的详细信息。需要 PERM_MODEL_RESOURCE_READ（兼容旧 PERM_LLM_INVOKE）或 workspace 管理员权限；响应会脱敏 api_key_encrypted。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| backend_id | integer | 是 | 后端 ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 获取成功 | `catalog.Backend` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 404 | 后端不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.Backend`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| api_key_encrypted | string |  | 可选，加密存储 |
| created_at | integer |  | - |
| id | integer |  | - |
| models | []string |  | - |
| name | string |  | - |
| origin_ref | string |  | - |
| origin_version | string |  | - |
| provider_origin | string |  | provider_origin is a trusted billing classification. Tenant-created
backends are external_provider; Genesis is assigned only by a trusted
integration or migration, never inferred from a provider/model string. |
| reasoning_control_protocol | catalog.ReasoningControlProtocol |  | - |
| timeout_seconds | integer |  | - |
| type | catalog.BackendType |  | - |
| updated_at | integer |  | - |

响应示例:

```json
{
  "api_key_encrypted": "string",
  "created_at": 0,
  "id": 0,
  "models": ["string"],
  "name": "string",
  "origin_ref": "string",
  "origin_version": "string",
  "provider_origin": "string",
  "reasoning_control_protocol": {},
  "timeout_seconds": 0,
  "type": {},
  "updated_at": 0
}
```

---

### PUT /api/v1/workspaces/{id}/embeddings/backends/{backend_id}

**更新 Embedding 后端**

更新指定 Embedding 后端的配置信息

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| backend_id | integer | 是 | 后端 ID |

#### 请求体

类型: `catalog.UpdateBackendRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| api_key_encrypted | string |  | - |
| models | []string |  | - |
| name | string |  | - |
| origin_ref | string |  | - |
| origin_version | string |  | - |
| provider_origin | string |  | - |
| reasoning_control_protocol | catalog.ReasoningControlProtocol |  | - |
| timeout_seconds | integer |  | - |

示例:

```json
{
  "api_key_encrypted": "string",
  "models": ["string"],
  "name": "string",
  "origin_ref": "string",
  "origin_version": "string",
  "provider_origin": "string",
  "reasoning_control_protocol": {},
  "timeout_seconds": 0
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 更新成功 | `catalog.Backend` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.Backend`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| api_key_encrypted | string |  | 可选，加密存储 |
| created_at | integer |  | - |
| id | integer |  | - |
| models | []string |  | - |
| name | string |  | - |
| origin_ref | string |  | - |
| origin_version | string |  | - |
| provider_origin | string |  | provider_origin is a trusted billing classification. Tenant-created
backends are external_provider; Genesis is assigned only by a trusted
integration or migration, never inferred from a provider/model string. |
| reasoning_control_protocol | catalog.ReasoningControlProtocol |  | - |
| timeout_seconds | integer |  | - |
| type | catalog.BackendType |  | - |
| updated_at | integer |  | - |

响应示例:

```json
{
  "api_key_encrypted": "string",
  "created_at": 0,
  "id": 0,
  "models": ["string"],
  "name": "string",
  "origin_ref": "string",
  "origin_version": "string",
  "provider_origin": "string",
  "reasoning_control_protocol": {},
  "timeout_seconds": 0,
  "type": {},
  "updated_at": 0
}
```

---

### DELETE /api/v1/workspaces/{id}/embeddings/backends/{backend_id}

**删除 Embedding 后端**

删除指定的 Embedding 后端配置

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| backend_id | integer | 是 | 后端 ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 删除成功 | `object` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

---

### GET /api/v1/workspaces/{id}/embeddings/backends/{backend_id}/endpoints

**列出 Embedding 后端端点**

列出指定后端下的所有 Embedding 服务端点。需要 PERM_MODEL_RESOURCE_READ（兼容旧 PERM_LLM_INVOKE）或 workspace 管理员权限。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| backend_id | integer | 是 | 后端 ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 端点列表 | `[]catalog.BackendEndpoint` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

---

### POST /api/v1/workspaces/{id}/embeddings/backends/{backend_id}/endpoints

**创建 Embedding 端点**

为指定后端创建新的 Embedding 服务端点

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| backend_id | integer | 是 | 后端 ID |

#### 请求体

类型: JSON 对象

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 201 | 创建成功 | `catalog.BackendEndpoint` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.BackendEndpoint`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| address | string |  | - |
| backend_id | integer |  | - |
| created_at | integer |  | - |
| id | integer |  | - |
| status | catalog.EndpointStatus |  | - |
| updated_at | integer |  | - |

响应示例:

```json
{
  "address": "string",
  "backend_id": 0,
  "created_at": 0,
  "id": 0,
  "status": {},
  "updated_at": 0
}
```

---

### PUT /api/v1/workspaces/{id}/embeddings/backends/{backend_id}/endpoints/{endpoint_id}/status

**设置 Embedding 端点状态**

更新指定 Embedding 端点的启用/禁用状态

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| backend_id | integer | 是 | 后端 ID |
| endpoint_id | integer | 是 | 端点 ID |

#### 请求体

类型: `catalog.SetEndpointStatusRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| backend_id | integer |  | - |
| endpoint_id | integer |  | - |
| status | catalog.EndpointStatus |  | - |

示例:

```json
{
  "backend_id": 0,
  "endpoint_id": 0,
  "status": {}
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 设置成功 | `object` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

---

### GET /api/v1/workspaces/{id}/embeddings/models

**获取可用 Embedding 模型列表**

扁平化当前 workspace 的所有 Embedding 后端模型，供工作流表单下拉使用。可选返回 TaaS Embedding 子类型字段 type（embedding_text / embedding_multimodal）。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 获取成功 | `handlers.EmbeddingListModelsResponse` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`handlers.EmbeddingListModelsResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| models | []handlers.EmbeddingListModelsResponseItem |  | - |

响应示例:

```json
{
  "models": [{
    "backend_id": 0,
    "backend_name": "string",
    "dim": 0,
    "model": "string",
    "type": "string"
  }]
}
```

---

### GET /api/v1/workspaces/{id}/embeddings/router-config

**获取 Embedding 路由配置**

获取指定 workspace 的 Embedding 路由配置

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 获取成功 | `catalog.GetRouterConfigResponse` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.GetRouterConfigResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| config | catalog.RouterConfig |  | - |

响应示例:

```json
{
  "config": {
    "created_at": 0,
    "enable_session_affinity": false,
    "health_check_interval_seconds": 0,
    "id": 0,
    "max_retries": 0,
    "strategy": {},
    "updated_at": 0
  }
}
```

---

### PUT /api/v1/workspaces/{id}/embeddings/router-config

**更新 Embedding 路由配置**

更新指定 workspace 的 Embedding 路由配置

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 请求体

类型: `catalog.PutRouterConfigRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| enable_session_affinity | boolean |  | - |
| health_check_interval_seconds | integer |  | - |
| max_retries | integer |  | - |
| strategy | catalog.RouterStrategy |  | - |

示例:

```json
{
  "enable_session_affinity": false,
  "health_check_interval_seconds": 0,
  "max_retries": 0,
  "strategy": {}
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 更新成功 | `catalog.GetRouterConfigResponse` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.GetRouterConfigResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| config | catalog.RouterConfig |  | - |

响应示例:

```json
{
  "config": {
    "created_at": 0,
    "enable_session_affinity": false,
    "health_check_interval_seconds": 0,
    "id": 0,
    "max_retries": 0,
    "strategy": {},
    "updated_at": 0
  }
}
```

---

## File 管理

### POST /api/v1/workspaces/{id}/catalog-files/uploads

**上传私有 Catalog 文件**

上传文件并持久化到当前用户的私有 Catalog Volume

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 201 | 上传成功 | `handlers.UploadFileResponse` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`handlers.UploadFileResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| catalog_file | workspacefileupload.CatalogFile |  | - |
| file_id | string |  | - |
| md5 | string |  | - |
| original_name | string |  | - |
| size | integer |  | - |

响应示例:

```json
{
  "catalog_file": {
    "file_id": "string",
    "md5": "string",
    "name": "string",
    "path": "string",
    "size": 0,
    "volume_id": 0,
    "workspace_id": "string"
  },
  "file_id": "string",
  "md5": "string",
  "original_name": "string",
  "size": 0
}
```

---

### POST /api/v1/workspaces/{id}/files

**上传文件**

上传新文件到指定 workspace。普通调用者需要 PERM_FILE_UPLOAD；system user 入口仅供受信任内部服务使用

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 201 | 上传成功 | `handlers.UploadFileResponse` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 403 | 无权限：普通调用者需要 PERM_FILE_UPLOAD；system-user bypass 仅供受信任内部服务 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`handlers.UploadFileResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| catalog_file | workspacefileupload.CatalogFile |  | - |
| file_id | string |  | - |
| md5 | string |  | - |
| original_name | string |  | - |
| size | integer |  | - |

响应示例:

```json
{
  "catalog_file": {
    "file_id": "string",
    "md5": "string",
    "name": "string",
    "path": "string",
    "size": 0,
    "volume_id": 0,
    "workspace_id": "string"
  },
  "file_id": "string",
  "md5": "string",
  "original_name": "string",
  "size": 0
}
```

---

### GET /api/v1/workspaces/{id}/files/{file_id}

**获取文件元数据**

获取指定文件的元数据信息

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| file_id | string | 是 | File ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 文件元数据 | `catalog.File` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 404 | 文件不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.File`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| created_at | integer |  | Unix 时间戳（秒） |
| created_by | string |  | 创建者用户 ID |
| extension | string |  | - |
| hash | string |  | - |
| id | string |  | - |
| name | string |  | - |
| parent_id | string |  | - |
| path | string |  | - |
| size | integer |  | - |
| type | catalog.FileType |  | - |
| updated_at | integer |  | Unix 时间戳（秒） |
| updated_by | string |  | 更新者用户 ID |
| volume_id | string |  | - |

响应示例:

```json
{
  "created_at": 0,
  "created_by": "string",
  "extension": "string",
  "hash": "string",
  "id": "string",
  "name": "string",
  "parent_id": "string",
  "path": "string",
  "size": 0,
  "type": {},
  "updated_at": 0,
  "updated_by": "string",
  "volume_id": "string"
}
```

---

### DELETE /api/v1/workspaces/{id}/files/{file_id}

**删除文件**

删除指定文件

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| file_id | string | 是 | File ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 204 | 删除成功 |  |
| 401 | 未认证 | `gin.ErrorResponse` |
| 404 | 文件不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

---

### GET /api/v1/workspaces/{id}/files/{file_id}/download

**下载文件**

下载指定文件的内容

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| file_id | string | 是 | File ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 文件内容 | `file` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 404 | 文件不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

---

### GET /api/v1/workspaces/{id}/files/{file_id}/preview

**预览文件**

预览指定 Runtime File；HTML、SVG 等可执行格式以附件响应，Office 文档转换为 PDF

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| file_id | string | 是 | File ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 预览内容 | `file` |
| 400 | 不支持的文件类型 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 404 | 文件不存在或无权访问 | `gin.ErrorResponse` |
| 413 | 文件超过预览大小限制 | `gin.ErrorResponse` |
| 502 | Office 转换失败 | `gin.ErrorResponse` |
| 503 | 预览服务不可用 | `gin.ErrorResponse` |
| 504 | Office 转换超时 | `gin.ErrorResponse` |

---

## Garbage Collection

### POST /api/v1/workspaces/{id}/garbage-collection

**触发垃圾回收**

触发指定 workspace 的垃圾回收，清理孤立文件和已删除的 Volume

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 请求体

类型: `handlers.TriggerGarbageCollectionRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| batch_size | integer |  | BatchSize is the maximum number of items to process.
Default: 100 |
| orphan_file_threshold_hours | integer |  | OrphanFileThresholdHours is the minimum age of orphan files before they can be cleaned (in hours).
Default: 24 hours |

示例:

```json
{
  "batch_size": 0,
  "orphan_file_threshold_hours": 0
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 回收结果 | `handlers.TriggerGarbageCollectionResponse` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 404 | Workspace 不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`handlers.TriggerGarbageCollectionResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| deleted_volumes_cleaned | integer |  | - |
| message | string |  | - |
| orphan_files_cleaned | integer |  | - |

响应示例:

```json
{
  "deleted_volumes_cleaned": 0,
  "message": "string",
  "orphan_files_cleaned": 0
}
```

---

## IAM

### POST /api/v1/system/iam/resources/authorized/delete

**Begin authorized Backend-owned IAM resource deletion**

认证: 需要 API Key

#### 请求体

类型: `iam.ResourceLifecycleMutationRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| operation_id | string |  | - |
| principal | iam.Principal |  | - |
| request_id | string |  | - |
| resource | iam.ResourceDescriptor |  | - |
| role_id | string |  | - |
| trace_id | string |  | - |
| workspace_id | string |  | - |

示例:

```json
{
  "operation_id": "string",
  "principal": {
    "principal_id": "string",
    "principal_type": {}
  },
  "request_id": "string",
  "resource": {
    "parent_resource": {
      "parent_resource": {
        "parent_resource": {},
        "resource_id": "string",
        "resource_path": [{}],
        "resource_type": "string"
      },
      "resource_id": "string",
      "resource_path": [{
        "parent_resource": {},
        "resource_id": "string",
        "resource_path": [{}],
        "resource_type": "string"
      }],
      "resource_type": "string"
    },
    "resource_id": "string",
    "resource_path": [{
      "parent_resource": {
        "parent_resource": {},
        "resource_id": "string",
        "resource_path": [{}],
        "resource_type": "string"
      },
      "resource_id": "string",
      "resource_path": [{
        "parent_resource": {},
        "resource_id": "string",
        "resource_path": [{}],
        "resource_type": "string"
      }],
      "resource_type": "string"
    }],
    "resource_type": "string"
  },
  "role_id": "string",
  "trace_id": "string",
  "workspace_id": "string"
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | OK | `iam.ResourceLifecycleMutationResponse` |

响应字段 (`iam.ResourceLifecycleMutationResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| idempotent_replay | boolean |  | - |
| owner_role_id | string |  | - |
| ownership_version | integer |  | - |
| request_id | string |  | - |
| resource | iam.ResourceDescriptor |  | - |
| trace_id | string |  | - |
| workspace_id | string |  | - |

响应示例:

```json
{
  "idempotent_replay": false,
  "owner_role_id": "string",
  "ownership_version": 0,
  "request_id": "string",
  "resource": {
    "parent_resource": {
      "parent_resource": {
        "parent_resource": {},
        "resource_id": "string",
        "resource_path": [{}],
        "resource_type": "string"
      },
      "resource_id": "string",
      "resource_path": [{
        "parent_resource": {},
        "resource_id": "string",
        "resource_path": [{}],
        "resource_type": "string"
      }],
      "resource_type": "string"
    },
    "resource_id": "string",
    "resource_path": [{
      "parent_resource": {
        "parent_resource": {},
        "resource_id": "string",
        "resource_path": [{}],
        "resource_type": "string"
      },
      "resource_id": "string",
      "resource_path": [{
        "parent_resource": {},
        "resource_id": "string",
        "resource_path": [{}],
        "resource_type": "string"
      }],
      "resource_type": "string"
    }],
    "resource_type": "string"
  },
  "trace_id": "string",
  "workspace_id": "string"
}
```

---

### POST /api/v1/system/iam/resources/authorized/delete/finalize

**Finalize authorized Backend-owned IAM resource deletion**

认证: 需要 API Key

#### 请求体

类型: `iam.ResourceLifecycleMutationRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| operation_id | string |  | - |
| principal | iam.Principal |  | - |
| request_id | string |  | - |
| resource | iam.ResourceDescriptor |  | - |
| role_id | string |  | - |
| trace_id | string |  | - |
| workspace_id | string |  | - |

示例:

```json
{
  "operation_id": "string",
  "principal": {
    "principal_id": "string",
    "principal_type": {}
  },
  "request_id": "string",
  "resource": {
    "parent_resource": {
      "parent_resource": {
        "parent_resource": {},
        "resource_id": "string",
        "resource_path": [{}],
        "resource_type": "string"
      },
      "resource_id": "string",
      "resource_path": [{
        "parent_resource": {},
        "resource_id": "string",
        "resource_path": [{}],
        "resource_type": "string"
      }],
      "resource_type": "string"
    },
    "resource_id": "string",
    "resource_path": [{
      "parent_resource": {
        "parent_resource": {},
        "resource_id": "string",
        "resource_path": [{}],
        "resource_type": "string"
      },
      "resource_id": "string",
      "resource_path": [{
        "parent_resource": {},
        "resource_id": "string",
        "resource_path": [{}],
        "resource_type": "string"
      }],
      "resource_type": "string"
    }],
    "resource_type": "string"
  },
  "role_id": "string",
  "trace_id": "string",
  "workspace_id": "string"
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | OK | `iam.ResourceLifecycleMutationResponse` |

响应字段 (`iam.ResourceLifecycleMutationResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| idempotent_replay | boolean |  | - |
| owner_role_id | string |  | - |
| ownership_version | integer |  | - |
| request_id | string |  | - |
| resource | iam.ResourceDescriptor |  | - |
| trace_id | string |  | - |
| workspace_id | string |  | - |

响应示例:

```json
{
  "idempotent_replay": false,
  "owner_role_id": "string",
  "ownership_version": 0,
  "request_id": "string",
  "resource": {
    "parent_resource": {
      "parent_resource": {
        "parent_resource": {},
        "resource_id": "string",
        "resource_path": [{}],
        "resource_type": "string"
      },
      "resource_id": "string",
      "resource_path": [{
        "parent_resource": {},
        "resource_id": "string",
        "resource_path": [{}],
        "resource_type": "string"
      }],
      "resource_type": "string"
    },
    "resource_id": "string",
    "resource_path": [{
      "parent_resource": {
        "parent_resource": {},
        "resource_id": "string",
        "resource_path": [{}],
        "resource_type": "string"
      },
      "resource_id": "string",
      "resource_path": [{
        "parent_resource": {},
        "resource_id": "string",
        "resource_path": [{}],
        "resource_type": "string"
      }],
      "resource_type": "string"
    }],
    "resource_type": "string"
  },
  "trace_id": "string",
  "workspace_id": "string"
}
```

---

### POST /api/v1/system/iam/resources/authorized/register

**Register Backend-owned IAM resource with a frozen Effective Role**

认证: 需要 API Key

#### 请求体

类型: `iam.ResourceLifecycleMutationRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| operation_id | string |  | - |
| principal | iam.Principal |  | - |
| request_id | string |  | - |
| resource | iam.ResourceDescriptor |  | - |
| role_id | string |  | - |
| trace_id | string |  | - |
| workspace_id | string |  | - |

示例:

```json
{
  "operation_id": "string",
  "principal": {
    "principal_id": "string",
    "principal_type": {}
  },
  "request_id": "string",
  "resource": {
    "parent_resource": {
      "parent_resource": {
        "parent_resource": {},
        "resource_id": "string",
        "resource_path": [{}],
        "resource_type": "string"
      },
      "resource_id": "string",
      "resource_path": [{
        "parent_resource": {},
        "resource_id": "string",
        "resource_path": [{}],
        "resource_type": "string"
      }],
      "resource_type": "string"
    },
    "resource_id": "string",
    "resource_path": [{
      "parent_resource": {
        "parent_resource": {},
        "resource_id": "string",
        "resource_path": [{}],
        "resource_type": "string"
      },
      "resource_id": "string",
      "resource_path": [{
        "parent_resource": {},
        "resource_id": "string",
        "resource_path": [{}],
        "resource_type": "string"
      }],
      "resource_type": "string"
    }],
    "resource_type": "string"
  },
  "role_id": "string",
  "trace_id": "string",
  "workspace_id": "string"
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | OK | `iam.ResourceLifecycleMutationResponse` |

响应字段 (`iam.ResourceLifecycleMutationResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| idempotent_replay | boolean |  | - |
| owner_role_id | string |  | - |
| ownership_version | integer |  | - |
| request_id | string |  | - |
| resource | iam.ResourceDescriptor |  | - |
| trace_id | string |  | - |
| workspace_id | string |  | - |

响应示例:

```json
{
  "idempotent_replay": false,
  "owner_role_id": "string",
  "ownership_version": 0,
  "request_id": "string",
  "resource": {
    "parent_resource": {
      "parent_resource": {
        "parent_resource": {},
        "resource_id": "string",
        "resource_path": [{}],
        "resource_type": "string"
      },
      "resource_id": "string",
      "resource_path": [{
        "parent_resource": {},
        "resource_id": "string",
        "resource_path": [{}],
        "resource_type": "string"
      }],
      "resource_type": "string"
    },
    "resource_id": "string",
    "resource_path": [{
      "parent_resource": {
        "parent_resource": {},
        "resource_id": "string",
        "resource_path": [{}],
        "resource_type": "string"
      },
      "resource_id": "string",
      "resource_path": [{
        "parent_resource": {},
        "resource_id": "string",
        "resource_path": [{}],
        "resource_type": "string"
      }],
      "resource_type": "string"
    }],
    "resource_type": "string"
  },
  "trace_id": "string",
  "workspace_id": "string"
}
```

---

## Internal

### GET /api/v1/system/workspaces/{id}/semantic-model-artifacts/{file_id}/download

**Download a semantic model artifact**

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| file_id | string | 是 | Semantic model artifact file ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | Artifact content | `file` |
| 400 | Invalid workspace or file ID | `gin.ErrorResponse` |
| 404 | Artifact not found in workspace | `gin.ErrorResponse` |
| 500 | Internal error | `gin.ErrorResponse` |

---

## Internal Billing

### POST /api/v1/internal/billing/workspace-subjects/sync

**Synchronize billing workspace subject**

Applies a moi-backend authenticated billing-subject transition with a durable idempotency key.

认证: 需要 API Key

#### 请求体

类型: `subjectsync.Request`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| billing_account_id | string |  | - |
| effective_from | string |  | - |
| effective_to | string |  | - |
| idempotency_key | string |  | - |
| local_account_uid | string |  | - |
| moi_user_id | string |  | - |
| reason | string |  | - |
| request_hash | string |  | - |
| request_id | string |  | - |
| source | string |  | - |
| source_version | string |  | - |
| source_version_type | string |  | - |
| status | string |  | - |
| subject_id | string |  | - |
| subject_type | string |  | - |
| subject_uc_account_id | string |  | - |
| workspace_id | string |  | - |

示例:

```json
{
  "billing_account_id": "string",
  "effective_from": "string",
  "effective_to": "string",
  "idempotency_key": "string",
  "local_account_uid": "string",
  "moi_user_id": "string",
  "reason": "string",
  "request_hash": "string",
  "request_id": "string",
  "source": "string",
  "source_version": "string",
  "source_version_type": "string",
  "status": "string",
  "subject_id": "string",
  "subject_type": "string",
  "subject_uc_account_id": "string",
  "workspace_id": "string"
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | Applied or idempotent transition | `subjectsync.Response` |
| 400 | Invalid request or rejected transition | `gin.ErrorResponse` |
| 403 | Caller is not the authenticated moi-backend service | `gin.ErrorResponse` |
| 503 | Billing subject sync is unavailable | `gin.ErrorResponse` |

响应字段 (`subjectsync.Response`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| applied | boolean |  | - |
| current_source_version | string |  | - |
| current_source_version_type | string |  | - |
| current_subject | domain.SubjectSnapshot |  | - |
| error_code | string |  | - |
| result | string |  | - |
| subject_history_id | integer |  | - |
| workspace_id | string |  | - |

响应示例:

```json
{
  "applied": false,
  "current_source_version": "string",
  "current_source_version_type": "string",
  "current_subject": {
    "billing_account_id": "string",
    "local_account_uid": "string",
    "moi_user_id": "string",
    "subject_history_id": 0,
    "subject_id": "string",
    "subject_type": {},
    "subject_uc_account_id": "string"
  },
  "error_code": "string",
  "result": "string",
  "subject_history_id": 0,
  "workspace_id": "string"
}
```

---

## LLM 管理

### GET /api/v1/workspaces/{id}/llm/backends

**获取 LLM 后端列表**

获取指定 workspace 中所有 LLM 后端配置。需要 PERM_MODEL_RESOURCE_READ（兼容旧 PERM_LLM_INVOKE）或 workspace 管理员权限；响应会脱敏 api_key_encrypted。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 获取成功 | `catalog.ListBackendsResponse` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.ListBackendsResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| backends | []catalog.Backend |  | - |
| total | integer |  | - |

响应示例:

```json
{
  "backends": [{
    "api_key_encrypted": "string",
    "created_at": 0,
    "id": 0,
    "models": ["string"],
    "name": "string",
    "origin_ref": "string",
    "origin_version": "string",
    "provider_origin": "string",
    "reasoning_control_protocol": {},
    "timeout_seconds": 0,
    "type": {},
    "updated_at": 0
  }],
  "total": 0
}
```

---

### POST /api/v1/workspaces/{id}/llm/backends

**创建 LLM 后端**

在指定 workspace 中创建新的 LLM 后端配置

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 请求体

类型: `catalog.CreateBackendRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| api_key_encrypted | string |  | - |
| models | []string |  | - |
| name | string |  | - |
| origin_ref | string |  | - |
| origin_version | string |  | - |
| provider_origin | string |  | - |
| reasoning_control_protocol | catalog.ReasoningControlProtocol |  | - |
| timeout_seconds | integer |  | - |
| type | catalog.BackendType |  | - |

示例:

```json
{
  "api_key_encrypted": "string",
  "models": ["string"],
  "name": "string",
  "origin_ref": "string",
  "origin_version": "string",
  "provider_origin": "string",
  "reasoning_control_protocol": {},
  "timeout_seconds": 0,
  "type": {}
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 201 | 创建成功 | `catalog.Backend` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.Backend`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| api_key_encrypted | string |  | 可选，加密存储 |
| created_at | integer |  | - |
| id | integer |  | - |
| models | []string |  | - |
| name | string |  | - |
| origin_ref | string |  | - |
| origin_version | string |  | - |
| provider_origin | string |  | provider_origin is a trusted billing classification. Tenant-created
backends are external_provider; Genesis is assigned only by a trusted
integration or migration, never inferred from a provider/model string. |
| reasoning_control_protocol | catalog.ReasoningControlProtocol |  | - |
| timeout_seconds | integer |  | - |
| type | catalog.BackendType |  | - |
| updated_at | integer |  | - |

响应示例:

```json
{
  "api_key_encrypted": "string",
  "created_at": 0,
  "id": 0,
  "models": ["string"],
  "name": "string",
  "origin_ref": "string",
  "origin_version": "string",
  "provider_origin": "string",
  "reasoning_control_protocol": {},
  "timeout_seconds": 0,
  "type": {},
  "updated_at": 0
}
```

---

### GET /api/v1/workspaces/{id}/llm/backends/{backend_id}

**获取 LLM 后端详情**

根据后端 ID 获取指定 LLM 后端的详细信息。需要 PERM_MODEL_RESOURCE_READ（兼容旧 PERM_LLM_INVOKE）或 workspace 管理员权限；响应会脱敏 api_key_encrypted。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| backend_id | integer | 是 | 后端 ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 获取成功 | `catalog.Backend` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 404 | 后端不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.Backend`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| api_key_encrypted | string |  | 可选，加密存储 |
| created_at | integer |  | - |
| id | integer |  | - |
| models | []string |  | - |
| name | string |  | - |
| origin_ref | string |  | - |
| origin_version | string |  | - |
| provider_origin | string |  | provider_origin is a trusted billing classification. Tenant-created
backends are external_provider; Genesis is assigned only by a trusted
integration or migration, never inferred from a provider/model string. |
| reasoning_control_protocol | catalog.ReasoningControlProtocol |  | - |
| timeout_seconds | integer |  | - |
| type | catalog.BackendType |  | - |
| updated_at | integer |  | - |

响应示例:

```json
{
  "api_key_encrypted": "string",
  "created_at": 0,
  "id": 0,
  "models": ["string"],
  "name": "string",
  "origin_ref": "string",
  "origin_version": "string",
  "provider_origin": "string",
  "reasoning_control_protocol": {},
  "timeout_seconds": 0,
  "type": {},
  "updated_at": 0
}
```

---

### PUT /api/v1/workspaces/{id}/llm/backends/{backend_id}

**更新 LLM 后端**

更新指定 LLM 后端的配置信息

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| backend_id | integer | 是 | 后端 ID |

#### 请求体

类型: `catalog.UpdateBackendRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| api_key_encrypted | string |  | - |
| models | []string |  | - |
| name | string |  | - |
| origin_ref | string |  | - |
| origin_version | string |  | - |
| provider_origin | string |  | - |
| reasoning_control_protocol | catalog.ReasoningControlProtocol |  | - |
| timeout_seconds | integer |  | - |

示例:

```json
{
  "api_key_encrypted": "string",
  "models": ["string"],
  "name": "string",
  "origin_ref": "string",
  "origin_version": "string",
  "provider_origin": "string",
  "reasoning_control_protocol": {},
  "timeout_seconds": 0
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 更新成功 | `catalog.Backend` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 404 | 后端不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.Backend`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| api_key_encrypted | string |  | 可选，加密存储 |
| created_at | integer |  | - |
| id | integer |  | - |
| models | []string |  | - |
| name | string |  | - |
| origin_ref | string |  | - |
| origin_version | string |  | - |
| provider_origin | string |  | provider_origin is a trusted billing classification. Tenant-created
backends are external_provider; Genesis is assigned only by a trusted
integration or migration, never inferred from a provider/model string. |
| reasoning_control_protocol | catalog.ReasoningControlProtocol |  | - |
| timeout_seconds | integer |  | - |
| type | catalog.BackendType |  | - |
| updated_at | integer |  | - |

响应示例:

```json
{
  "api_key_encrypted": "string",
  "created_at": 0,
  "id": 0,
  "models": ["string"],
  "name": "string",
  "origin_ref": "string",
  "origin_version": "string",
  "provider_origin": "string",
  "reasoning_control_protocol": {},
  "timeout_seconds": 0,
  "type": {},
  "updated_at": 0
}
```

---

### DELETE /api/v1/workspaces/{id}/llm/backends/{backend_id}

**删除 LLM 后端**

删除指定的 LLM 后端配置

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| backend_id | integer | 是 | 后端 ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 204 | 删除成功 |  |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 404 | 后端不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

---

### GET /api/v1/workspaces/{id}/llm/backends/{backend_id}/endpoints

**列出 LLM 后端端点**

列出指定后端下的所有端点。需要 PERM_MODEL_RESOURCE_READ（兼容旧 PERM_LLM_INVOKE）或 workspace 管理员权限。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| backend_id | integer | 是 | 后端 ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 端点列表 | `[]catalog.BackendEndpoint` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

---

### POST /api/v1/workspaces/{id}/llm/backends/{backend_id}/endpoints

**创建 LLM 端点**

为指定后端创建新的 LLM 服务端点

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| backend_id | integer | 是 | 后端 ID |

#### 请求体

类型: JSON 对象

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 201 | 创建成功 | `catalog.BackendEndpoint` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.BackendEndpoint`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| address | string |  | - |
| backend_id | integer |  | - |
| created_at | integer |  | - |
| id | integer |  | - |
| status | catalog.EndpointStatus |  | - |
| updated_at | integer |  | - |

响应示例:

```json
{
  "address": "string",
  "backend_id": 0,
  "created_at": 0,
  "id": 0,
  "status": {},
  "updated_at": 0
}
```

---

### PUT /api/v1/workspaces/{id}/llm/backends/{backend_id}/endpoints/{endpoint_id}/status

**设置端点状态**

更新指定 LLM 端点的启用/禁用状态

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| backend_id | integer | 是 | 后端 ID |
| endpoint_id | integer | 是 | 端点 ID |

#### 请求体

类型: `catalog.SetEndpointStatusRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| backend_id | integer |  | - |
| endpoint_id | integer |  | - |
| status | catalog.EndpointStatus |  | - |

示例:

```json
{
  "backend_id": 0,
  "endpoint_id": 0,
  "status": {}
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 204 | 设置成功 |  |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 404 | 端点不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

---

### POST /api/v1/workspaces/{id}/llm/chat/completions

**LLM 聊天补全**

发送 OpenAI 兼容的聊天补全请求，支持流式和非流式响应；请求体必须包含非空字符串 model，缺失时返回 REQUIRED_PARAMETER_MISSING

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 请求体

类型: JSON 对象

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 聊天补全响应（流式返回 text/event-stream） | `object` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |
| 503 | 无可用后端 | `gin.ErrorResponse` |

---

### GET /api/v1/workspaces/{id}/llm/messages/{message_id}

**获取 LLM 消息详情**

根据消息 ID 获取指定聊天消息的详细信息

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| message_id | integer | 是 | 消息 ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 获取成功 | `catalog.ChatMessage` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 404 | 消息不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.ChatMessage`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| config | string |  | - |
| content | string |  | - |
| created_at | integer |  | - |
| id | integer |  | - |
| model | string |  | - |
| modified_response | string |  | - |
| original_content | string |  | - |
| response | string |  | - |
| role | catalog.MessageRole |  | - |
| session_id | integer |  | - |
| source | string |  | - |
| status | string |  | - |
| updated_at | integer |  | - |
| user_id | string |  | - |

响应示例:

```json
{
  "config": "string",
  "content": "string",
  "created_at": 0,
  "id": 0,
  "model": "string",
  "modified_response": "string",
  "original_content": "string",
  "response": "string",
  "role": {},
  "session_id": 0,
  "source": "string",
  "status": "string",
  "updated_at": 0,
  "user_id": "string"
}
```

---

### GET /api/v1/workspaces/{id}/llm/messages/{message_id}/tags

**获取消息标签列表**

获取指定消息关联的所有标签

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| message_id | integer | 是 | 消息 ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 获取成功 | `object` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

---

### POST /api/v1/workspaces/{id}/llm/messages/{message_id}/tags

**添加消息标签关联**

为指定消息添加标签关联关系

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| message_id | integer | 是 | 消息 ID |

#### 请求体

类型: JSON 对象

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 204 | 添加成功 |  |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

---

### DELETE /api/v1/workspaces/{id}/llm/messages/{message_id}/tags/{tag_source}/{tag_name}

**移除消息标签关联**

移除指定消息与标签的关联关系

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| message_id | integer | 是 | 消息 ID |
| tag_source | string | 是 | 标签来源 |
| tag_name | string | 是 | 标签名称 |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 204 | 移除成功 |  |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

---

### GET /api/v1/workspaces/{id}/llm/models

**获取可用模型列表**

扁平化当前 workspace 的可用模型，供工作流表单下拉使用。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 获取成功 | `handlers.ListModelsResponse` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`handlers.ListModelsResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| models | []handlers.ListModelsResponseItem |  | - |

响应示例:

```json
{
  "models": [{
    "backend_id": 0,
    "backend_name": "string",
    "model": "string",
    "model_type": "string"
  }]
}
```

---

### GET /api/v1/workspaces/{id}/llm/router-config

**获取路由配置**

获取指定 workspace 的 LLM 路由配置

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 获取成功 | `catalog.GetRouterConfigResponse` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.GetRouterConfigResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| config | catalog.RouterConfig |  | - |

响应示例:

```json
{
  "config": {
    "created_at": 0,
    "enable_session_affinity": false,
    "health_check_interval_seconds": 0,
    "id": 0,
    "max_retries": 0,
    "strategy": {},
    "updated_at": 0
  }
}
```

---

### PUT /api/v1/workspaces/{id}/llm/router-config

**更新路由配置**

更新指定 workspace 的 LLM 路由配置

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 请求体

类型: `catalog.PutRouterConfigRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| enable_session_affinity | boolean |  | - |
| health_check_interval_seconds | integer |  | - |
| max_retries | integer |  | - |
| strategy | catalog.RouterStrategy |  | - |

示例:

```json
{
  "enable_session_affinity": false,
  "health_check_interval_seconds": 0,
  "max_retries": 0,
  "strategy": {}
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 更新成功 | `catalog.GetRouterConfigResponse` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.GetRouterConfigResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| config | catalog.RouterConfig |  | - |

响应示例:

```json
{
  "config": {
    "created_at": 0,
    "enable_session_affinity": false,
    "health_check_interval_seconds": 0,
    "id": 0,
    "max_retries": 0,
    "strategy": {},
    "updated_at": 0
  }
}
```

---

### GET /api/v1/workspaces/{id}/llm/sessions

**获取 LLM 会话列表**

分页获取指定 workspace 中的 LLM 会话列表

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 查询参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| source | string | 否 | 会话来源过滤 |
| keyword | string | 否 | 关键词搜索（匹配会话标题） |
| tag | string | 否 | 按标签名过滤 |
| page | integer | 否 | 页码（默认 1） |
| page_size | integer | 否 | 每页数量（默认 20，最大 100） |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 获取成功 | `catalog.ListSessionsResponse` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.ListSessionsResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| sessions | []catalog.Session |  | - |
| total | integer |  | - |

响应示例:

```json
{
  "sessions": [{
    "config": "string",
    "created_at": 0,
    "id": 0,
    "source": "string",
    "title": "string",
    "updated_at": 0,
    "user_id": "string"
  }],
  "total": 0
}
```

---

### POST /api/v1/workspaces/{id}/llm/sessions

**创建 LLM 会话**

在指定 workspace 中创建新的 LLM 对话会话

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 请求体

类型: `catalog.CreateSessionRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| config | string |  | - |
| source | string |  | - |
| title | string |  | - |

示例:

```json
{
  "config": "string",
  "source": "string",
  "title": "string"
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 201 | 创建成功 | `catalog.Session` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.Session`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| config | string |  | - |
| created_at | integer |  | - |
| id | integer |  | - |
| source | string |  | - |
| title | string |  | - |
| updated_at | integer |  | - |
| user_id | string |  | - |

响应示例:

```json
{
  "config": "string",
  "created_at": 0,
  "id": 0,
  "source": "string",
  "title": "string",
  "updated_at": 0,
  "user_id": "string"
}
```

---

### GET /api/v1/workspaces/{id}/llm/sessions/{session_id}

**获取 LLM 会话详情**

根据会话 ID 获取指定 LLM 会话的详细信息

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| session_id | integer | 是 | 会话 ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 获取成功 | `catalog.Session` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 404 | 会话不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.Session`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| config | string |  | - |
| created_at | integer |  | - |
| id | integer |  | - |
| source | string |  | - |
| title | string |  | - |
| updated_at | integer |  | - |
| user_id | string |  | - |

响应示例:

```json
{
  "config": "string",
  "created_at": 0,
  "id": 0,
  "source": "string",
  "title": "string",
  "updated_at": 0,
  "user_id": "string"
}
```

---

### PUT /api/v1/workspaces/{id}/llm/sessions/{session_id}

**更新 LLM 会话**

更新指定 LLM 会话的信息

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| session_id | integer | 是 | 会话 ID |

#### 请求体

类型: `catalog.UpdateSessionRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| config | string |  | - |
| expected_title | string |  | Optional CAS guard; a mismatch returns the current unchanged session without an error. |
| title | string |  | - |

示例:

```json
{
  "config": "string",
  "expected_title": "string",
  "title": "string"
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 返回结果会话；CAS 条件不匹配时返回当前未修改会话 | `catalog.Session` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 404 | 会话不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.Session`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| config | string |  | - |
| created_at | integer |  | - |
| id | integer |  | - |
| source | string |  | - |
| title | string |  | - |
| updated_at | integer |  | - |
| user_id | string |  | - |

响应示例:

```json
{
  "config": "string",
  "created_at": 0,
  "id": 0,
  "source": "string",
  "title": "string",
  "updated_at": 0,
  "user_id": "string"
}
```

---

### DELETE /api/v1/workspaces/{id}/llm/sessions/{session_id}

**删除 LLM 会话**

删除指定的 LLM 会话及其关联数据

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| session_id | integer | 是 | 会话 ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 204 | 删除成功 |  |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

---

### GET /api/v1/workspaces/{id}/llm/sessions/{session_id}/messages

**获取 LLM 消息列表**

分页获取指定会话中的聊天消息列表

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| session_id | integer | 是 | 会话 ID |

#### 查询参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| after | integer | 否 | 起始消息 ID（用于游标分页） |
| limit | integer | 否 | 返回数量限制（默认 50，最大 100） |
| role | integer | 否 | 消息角色过滤 |
| status | string | 否 | 消息状态过滤 |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 获取成功 | `catalog.ListMessagesResponse` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.ListMessagesResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| messages | []catalog.ChatMessage |  | - |
| total | integer |  | - |

响应示例:

```json
{
  "messages": [{
    "config": "string",
    "content": "string",
    "created_at": 0,
    "id": 0,
    "model": "string",
    "modified_response": "string",
    "original_content": "string",
    "response": "string",
    "role": {},
    "session_id": 0,
    "source": "string",
    "status": "string",
    "updated_at": 0,
    "user_id": "string"
  }],
  "total": 0
}
```

---

### POST /api/v1/workspaces/{id}/llm/sessions/{session_id}/messages

**创建 LLM 消息**

在指定会话中创建新的聊天消息

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| session_id | integer | 是 | 会话 ID |

#### 请求体

类型: `catalog.ChatMessage`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| config | string |  | - |
| content | string |  | - |
| created_at | integer |  | - |
| id | integer |  | - |
| model | string |  | - |
| modified_response | string |  | - |
| original_content | string |  | - |
| response | string |  | - |
| role | catalog.MessageRole |  | - |
| session_id | integer |  | - |
| source | string |  | - |
| status | string |  | - |
| updated_at | integer |  | - |
| user_id | string |  | - |

示例:

```json
{
  "config": "string",
  "content": "string",
  "created_at": 0,
  "id": 0,
  "model": "string",
  "modified_response": "string",
  "original_content": "string",
  "response": "string",
  "role": {},
  "session_id": 0,
  "source": "string",
  "status": "string",
  "updated_at": 0,
  "user_id": "string"
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 201 | 创建成功 | `catalog.ChatMessage` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.ChatMessage`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| config | string |  | - |
| content | string |  | - |
| created_at | integer |  | - |
| id | integer |  | - |
| model | string |  | - |
| modified_response | string |  | - |
| original_content | string |  | - |
| response | string |  | - |
| role | catalog.MessageRole |  | - |
| session_id | integer |  | - |
| source | string |  | - |
| status | string |  | - |
| updated_at | integer |  | - |
| user_id | string |  | - |

响应示例:

```json
{
  "config": "string",
  "content": "string",
  "created_at": 0,
  "id": 0,
  "model": "string",
  "modified_response": "string",
  "original_content": "string",
  "response": "string",
  "role": {},
  "session_id": 0,
  "source": "string",
  "status": "string",
  "updated_at": 0,
  "user_id": "string"
}
```

---

### GET /api/v1/workspaces/{id}/llm/sessions/{session_id}/messages/latest

**获取最新消息 ID**

获取指定会话中最新一条消息的 ID

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| session_id | integer | 是 | 会话 ID |

#### 查询参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| completed_only | string | 否 | 是否仅返回已完成的消息（1 或 true） |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 获取成功 | `catalog.LatestMessageResponse` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.LatestMessageResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| message_id | integer |  | - |

响应示例:

```json
{
  "message_id": 0
}
```

---

### POST /api/v1/workspaces/{id}/llm/sessions/{session_id}/messages/{message_id}/append-modified-response

**追加修改后的回复**

向指定消息追加修改后的 LLM 回复内容

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| session_id | integer | 是 | 会话 ID |
| message_id | integer | 是 | 消息 ID |

#### 请求体

类型: `catalog.AppendModifiedResponseRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| append_content | string |  | - |

示例:

```json
{
  "append_content": "string"
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 204 | 追加成功 |  |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

---

### PUT /api/v1/workspaces/{id}/llm/sessions/{session_id}/messages/{message_id}/modify-response

**修改消息回复**

修改指定消息的 LLM 回复内容

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| session_id | integer | 是 | 会话 ID |
| message_id | integer | 是 | 消息 ID |

#### 请求体

类型: `catalog.ModifyResponseRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| modified_response | string |  | - |

示例:

```json
{
  "modified_response": "string"
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 204 | 修改成功 |  |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

---

### GET /api/v1/workspaces/{id}/llm/sessions/{session_id}/tags

**获取会话标签列表**

获取指定会话关联的所有标签

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| session_id | integer | 是 | 会话 ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 获取成功 | `object` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

---

### POST /api/v1/workspaces/{id}/llm/sessions/{session_id}/tags

**添加会话标签关联**

为指定会话添加标签关联关系

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| session_id | integer | 是 | 会话 ID |

#### 请求体

类型: JSON 对象

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 204 | 添加成功 |  |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

---

### DELETE /api/v1/workspaces/{id}/llm/sessions/{session_id}/tags/{tag_source}/{tag_name}

**移除会话标签关联**

移除指定会话与标签的关联关系

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| session_id | integer | 是 | 会话 ID |
| tag_source | string | 是 | 标签来源 |
| tag_name | string | 是 | 标签名称 |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 204 | 移除成功 |  |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

---

### GET /api/v1/workspaces/{id}/llm/tags

**获取 LLM 标签列表**

分页获取指定 workspace 中的 LLM 标签列表

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 查询参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| source | string | 否 | 标签来源过滤 |
| keyword | string | 否 | 关键词搜索 |
| page | integer | 否 | 页码（默认 1） |
| page_size | integer | 否 | 每页数量（默认 20，最大 100） |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 获取成功 | `catalog.ListTagsResponse` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.ListTagsResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| tags | []catalog.Tag |  | - |
| total | integer |  | - |

响应示例:

```json
{
  "tags": [{
    "created_at": 0,
    "name": "string",
    "source": "string",
    "updated_at": 0
  }],
  "total": 0
}
```

---

### POST /api/v1/workspaces/{id}/llm/tags

**创建 LLM 标签**

在指定 workspace 中创建新的 LLM 标签

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 请求体

类型: `catalog.CreateTagRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string |  | - |
| source | string |  | - |

示例:

```json
{
  "name": "string",
  "source": "string"
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 201 | 创建成功 | `catalog.Tag` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.Tag`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| created_at | integer |  | - |
| name | string |  | - |
| source | string |  | - |
| updated_at | integer |  | - |

响应示例:

```json
{
  "created_at": 0,
  "name": "string",
  "source": "string",
  "updated_at": 0
}
```

---

### DELETE /api/v1/workspaces/{id}/llm/tags/{source}/{name}

**删除 LLM 标签**

根据来源和名称删除指定的 LLM 标签

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| source | string | 是 | 标签来源 |
| name | string | 是 | 标签名称 |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 204 | 删除成功 |  |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

---

## MaxCompute 元数据管理

### GET /api/v1/workspaces/{id}/maxcompute/configs/{config_id}/databases

**列出 MaxCompute 数据库**

列出指定 MaxCompute 配置下的所有数据库，支持分页

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| config_id | integer | 是 | MaxCompute 配置 ID |

#### 查询参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page_size | integer | 否 | 每页数量（默认 20，最大 100） |
| page_token | string | 否 | 分页令牌 |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 数据库列表 | `maxcompute.ListMCDatabasesResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 404 | 配置不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`maxcompute.ListMCDatabasesResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| items | []catalog.MCDatabase |  | - |
| next_page_token | string |  | - |
| total | integer |  | - |

响应示例:

```json
{
  "items": [{
    "comment": "string",
    "config_id": 0,
    "id": 0,
    "name": "string",
    "source": "string",
    "synced_at": 0
  }],
  "next_page_token": "string",
  "total": 0
}
```

---

### GET /api/v1/workspaces/{id}/maxcompute/configs/{config_id}/databases/{database_id}

**获取 MaxCompute 数据库详情**

根据 ID 获取指定 MaxCompute 数据库的详细信息

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| config_id | integer | 是 | MaxCompute 配置 ID |
| database_id | integer | 是 | 数据库 ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 数据库详情 | `maxcompute.MCDatabase` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 404 | 数据库不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`maxcompute.MCDatabase`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| comment | string |  | 项目描述 |
| config_id | integer |  | 关联的 MaxCompute 配置 ID |
| id | integer |  | - |
| name | string |  | 项目名称 |
| source | string |  | 数据源标识，固定为 "maxcompute" |
| synced_at | integer |  | 最后同步时间（Unix 时间戳，秒） |

响应示例:

```json
{
  "comment": "string",
  "config_id": 0,
  "id": 0,
  "name": "string",
  "source": "string",
  "synced_at": 0
}
```

---

### GET /api/v1/workspaces/{id}/maxcompute/configs/{config_id}/databases/{database_id}/tables

**列出 MaxCompute 数据表**

列出指定数据库下的所有数据表，支持分页

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| config_id | integer | 是 | MaxCompute 配置 ID |
| database_id | integer | 是 | 数据库 ID |

#### 查询参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page_size | integer | 否 | 每页数量（默认 20，最大 100） |
| page_token | string | 否 | 分页令牌 |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 数据表列表 | `maxcompute.ListMCTablesResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 404 | 数据库不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`maxcompute.ListMCTablesResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| items | []catalog.MCTable |  | - |
| next_page_token | string |  | - |
| total | integer |  | - |

响应示例:

```json
{
  "items": [{
    "columns": [{
      "comment": "string",
      "data_type": "string",
      "id": 0,
      "name": "string",
      "ordinal": 0
    }],
    "comment": "string",
    "config_id": 0,
    "database_id": 0,
    "id": 0,
    "name": "string",
    "table_type": "string",
    "view_text": "string"
  }],
  "next_page_token": "string",
  "total": 0
}
```

---

### GET /api/v1/workspaces/{id}/maxcompute/configs/{config_id}/databases/{database_id}/tables/{table_id}

**获取 MaxCompute 数据表详情**

根据 ID 获取指定数据表的详细信息，包括列定义和分区信息

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| config_id | integer | 是 | MaxCompute 配置 ID |
| database_id | integer | 是 | 数据库 ID |
| table_id | integer | 是 | 数据表 ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 数据表详情 | `maxcompute.MCTable` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 404 | 数据表不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`maxcompute.MCTable`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| columns | []catalog.MCColumn |  | 列信息（GetTable 时返回） |
| comment | string |  | 表描述 |
| config_id | integer |  | 关联的 MaxCompute 配置 ID |
| database_id | integer |  | 所属数据库（项目）ID |
| id | integer |  | - |
| name | string |  | 表名称 |
| table_type | string |  | 表类型：MANAGED_TABLE, EXTERNAL_TABLE, VIRTUAL_VIEW 等 |
| view_text | string |  | 视图定义文本（仅视图类型有值） |

响应示例:

```json
{
  "columns": [{
    "comment": "string",
    "data_type": "string",
    "id": 0,
    "name": "string",
    "ordinal": 0
  }],
  "comment": "string",
  "config_id": 0,
  "database_id": 0,
  "id": 0,
  "name": "string",
  "table_type": "string",
  "view_text": "string"
}
```

---

### GET /api/v1/workspaces/{id}/maxcompute/configs/{config_id}/health

**检查 MaxCompute 配置健康状态**

检查指定 MaxCompute 配置的连接健康状态

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| config_id | integer | 是 | MaxCompute 配置 ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 健康状态 | `maxcompute.MCHealthCheckResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 404 | 配置不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`maxcompute.MCHealthCheckResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| message | string |  | 状态消息 |
| status | string |  | 状态：healthy, unhealthy |

响应示例:

```json
{
  "message": "string",
  "status": "string"
}
```

---

### POST /api/v1/workspaces/{id}/maxcompute/configs/{config_id}/stop-sync

**停止 MaxCompute 元数据同步**

停止指定 MaxCompute 配置正在进行的元数据同步任务

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| config_id | integer | 是 | MaxCompute 配置 ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 204 | 停止成功 |  |
| 401 | 未认证 | `gin.ErrorResponse` |
| 404 | 配置不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

---

### POST /api/v1/workspaces/{id}/maxcompute/configs/{config_id}/sync

**同步 MaxCompute 元数据**

触发指定 MaxCompute 配置的元数据同步任务

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| config_id | integer | 是 | MaxCompute 配置 ID |

#### 请求体

类型: `maxcompute.SyncMCMetadataRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| cron_expression | string |  | cron 表达式，如 "0 */30 * * * *"（每30分钟） |
| project_name | string |  | 要同步的 MaxCompute 项目名称 |

示例:

```json
{
  "cron_expression": "string",
  "project_name": "string"
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 同步成功 | `maxcompute.SyncMCMetadataResponse` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 404 | 配置不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`maxcompute.SyncMCMetadataResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| case_id | string |  | 本次主动触发的执行ID |
| database |  |  | 同步后的数据库（项目）元数据 |
| tables_deleted | integer |  | 删除的表数量（MaxCompute 中已不存在） |
| tables_synced | integer |  | 新同步的表数量 |
| tables_updated | integer |  | 更新的表数量 |
| task_id | string |  | 触发同步的任务ID（定时任务） |

响应示例:

```json
{
  "case_id": "string",
  "database": "",
  "tables_deleted": 0,
  "tables_synced": 0,
  "tables_updated": 0,
  "task_id": "string"
}
```

---

## MaxCompute 配置管理

### GET /api/v1/workspaces/{id}/maxcompute/configs

**列出 MaxCompute 配置**

列出指定 workspace 中的所有 MaxCompute 配置，支持分页

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 查询参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page_size | integer | 否 | 每页数量（默认 20，最大 100） |
| page_token | string | 否 | 分页令牌 |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 配置列表 | `maxcompute.ListMCConfigsResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`maxcompute.ListMCConfigsResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| items | []catalog.MCConfig |  | - |
| next_page_token | string |  | - |
| total | integer |  | - |

响应示例:

```json
{
  "items": [{
    "access_key_id": "string",
    "access_key_secret": "string",
    "created_at": 0,
    "created_by": "string",
    "endpoint": "string",
    "id": 0,
    "name": "string",
    "project_name": "string",
    "region": "string",
    "sync_cron_expression": "string",
    "sync_database_name": "string",
    "sync_task_id": "string",
    "updated_at": 0,
    "updated_by": "string"
  }],
  "next_page_token": "string",
  "total": 0
}
```

---

### POST /api/v1/workspaces/{id}/maxcompute/configs

**创建 MaxCompute 配置**

在指定 workspace 中创建新的 MaxCompute 连接配置

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 请求体

类型: `maxcompute.CreateMCConfigRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| access_key_id | string |  | 阿里云 AccessKeyId，必填 |
| access_key_secret | string |  | 阿里云 AccessKeySecret，必填 |
| endpoint | string |  | MaxCompute Endpoint，必填 |
| name | string |  | 配置名称，必填 |
| project_name | string |  | MaxCompute 项目名称（可选） |
| region | string |  | 阿里云 Region（可选） |

示例:

```json
{
  "access_key_id": "string",
  "access_key_secret": "string",
  "endpoint": "string",
  "name": "string",
  "project_name": "string",
  "region": "string"
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 201 | 创建成功 | `maxcompute.MCConfig` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 409 | 配置名称已存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`maxcompute.MCConfig`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| access_key_id | string |  | 阿里云 AccessKeyId |
| access_key_secret | string |  | 阿里云 AccessKeySecret（供内部服务如 workitem 使用） |
| created_at | integer |  | Unix 时间戳（秒） |
| created_by | string |  | 创建者用户 ID |
| endpoint | string |  | MaxCompute Endpoint |
| id | integer |  | - |
| name | string |  | 配置名称，workspace 内唯一 |
| project_name | string |  | MaxCompute 项目名称（可选） |
| region | string |  | 阿里云 Region（可选） |
| sync_cron_expression | string |  | 同步 cron 表达式 |
| sync_database_name | string |  | 正在同步的数据库名称 |
| sync_task_id | string |  | 周期性同步任务 ID（空表示未启动同步） |
| updated_at | integer |  | Unix 时间戳（秒） |
| updated_by | string |  | 更新者用户 ID |

响应示例:

```json
{
  "access_key_id": "string",
  "access_key_secret": "string",
  "created_at": 0,
  "created_by": "string",
  "endpoint": "string",
  "id": 0,
  "name": "string",
  "project_name": "string",
  "region": "string",
  "sync_cron_expression": "string",
  "sync_database_name": "string",
  "sync_task_id": "string",
  "updated_at": 0,
  "updated_by": "string"
}
```

---

### GET /api/v1/workspaces/{id}/maxcompute/configs/{config_id}

**获取 MaxCompute 配置**

根据 ID 获取指定 MaxCompute 配置的详细信息

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| config_id | integer | 是 | MaxCompute 配置 ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 配置详情 | `maxcompute.MCConfig` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 404 | 配置不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`maxcompute.MCConfig`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| access_key_id | string |  | 阿里云 AccessKeyId |
| access_key_secret | string |  | 阿里云 AccessKeySecret（供内部服务如 workitem 使用） |
| created_at | integer |  | Unix 时间戳（秒） |
| created_by | string |  | 创建者用户 ID |
| endpoint | string |  | MaxCompute Endpoint |
| id | integer |  | - |
| name | string |  | 配置名称，workspace 内唯一 |
| project_name | string |  | MaxCompute 项目名称（可选） |
| region | string |  | 阿里云 Region（可选） |
| sync_cron_expression | string |  | 同步 cron 表达式 |
| sync_database_name | string |  | 正在同步的数据库名称 |
| sync_task_id | string |  | 周期性同步任务 ID（空表示未启动同步） |
| updated_at | integer |  | Unix 时间戳（秒） |
| updated_by | string |  | 更新者用户 ID |

响应示例:

```json
{
  "access_key_id": "string",
  "access_key_secret": "string",
  "created_at": 0,
  "created_by": "string",
  "endpoint": "string",
  "id": 0,
  "name": "string",
  "project_name": "string",
  "region": "string",
  "sync_cron_expression": "string",
  "sync_database_name": "string",
  "sync_task_id": "string",
  "updated_at": 0,
  "updated_by": "string"
}
```

---

### PUT /api/v1/workspaces/{id}/maxcompute/configs/{config_id}

**更新 MaxCompute 配置**

部分更新指定 MaxCompute 配置，所有字段均为可选

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| config_id | integer | 是 | MaxCompute 配置 ID |

#### 请求体

类型: `maxcompute.UpdateMCConfigRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| access_key_id | string |  | - |
| access_key_secret | string |  | - |
| endpoint | string |  | - |
| name | string |  | - |
| project_name | string |  | - |
| region | string |  | - |

示例:

```json
{
  "access_key_id": "string",
  "access_key_secret": "string",
  "endpoint": "string",
  "name": "string",
  "project_name": "string",
  "region": "string"
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 更新后的配置 | `maxcompute.MCConfig` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 404 | 配置不存在 | `gin.ErrorResponse` |
| 409 | 配置名称已存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`maxcompute.MCConfig`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| access_key_id | string |  | 阿里云 AccessKeyId |
| access_key_secret | string |  | 阿里云 AccessKeySecret（供内部服务如 workitem 使用） |
| created_at | integer |  | Unix 时间戳（秒） |
| created_by | string |  | 创建者用户 ID |
| endpoint | string |  | MaxCompute Endpoint |
| id | integer |  | - |
| name | string |  | 配置名称，workspace 内唯一 |
| project_name | string |  | MaxCompute 项目名称（可选） |
| region | string |  | 阿里云 Region（可选） |
| sync_cron_expression | string |  | 同步 cron 表达式 |
| sync_database_name | string |  | 正在同步的数据库名称 |
| sync_task_id | string |  | 周期性同步任务 ID（空表示未启动同步） |
| updated_at | integer |  | Unix 时间戳（秒） |
| updated_by | string |  | 更新者用户 ID |

响应示例:

```json
{
  "access_key_id": "string",
  "access_key_secret": "string",
  "created_at": 0,
  "created_by": "string",
  "endpoint": "string",
  "id": 0,
  "name": "string",
  "project_name": "string",
  "region": "string",
  "sync_cron_expression": "string",
  "sync_database_name": "string",
  "sync_task_id": "string",
  "updated_at": 0,
  "updated_by": "string"
}
```

---

### DELETE /api/v1/workspaces/{id}/maxcompute/configs/{config_id}

**删除 MaxCompute 配置**

删除指定 MaxCompute 配置，如果配置正在被使用则无法删除

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| config_id | integer | 是 | MaxCompute 配置 ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 204 | 删除成功 |  |
| 401 | 未认证 | `gin.ErrorResponse` |
| 404 | 配置不存在 | `gin.ErrorResponse` |
| 409 | 配置正在使用中 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

---

## Mowl Lineage

### GET /api/v1/workspaces/{id}/mowl/cases/{case_id}/node-executions/{node_execution_id}/block-map

**获取节点 block lineage 映射**

返回节点输出 block refs 与可证明的 BlockMapping edges；缺少证据时返回 explicit unavailable reason。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| case_id | string | 是 | Case ID |
| node_execution_id | string | 是 | Node execution ID |

#### 查询参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| root_asset_id | string | 否 | Root asset ID; when present filters block refs and mapping edges |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | Block map | `object` |

---

### POST /api/v1/workspaces/{id}/mowl/cases/{case_id}/node-executions/{node_execution_id}/rerun

**创建基础 rerun 分支**

从不可变 NodeExecutionSnapshot 解析输入激活边界，并创建 dormant rerun branch。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| case_id | string | 是 | Source case ID |
| node_execution_id | string | 是 | Node execution ID |

#### 查询参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| expected_rerun_contract_hash | string | 是 | Contract hash returned by rerun-plan |

#### 请求体

类型: `mowl.RerunRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| config_override | []integer |  | - |
| input_payload_override | []integer |  | - |
| mode |  |  | Mode is always rerun_current_and_downstream for basic rerun. |
| vars_payload_override | []integer |  | - |

示例:

```json
{
  "config_override": ["string"],
  "input_payload_override": ["string"],
  "mode": "",
  "vars_payload_override": ["string"]
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 201 | Created rerun branch | `object` |
| 400 | Missing expected rerun contract hash | `gin.ErrorResponse` |

---

### GET /api/v1/workspaces/{id}/mowl/cases/{case_id}/node-executions/{node_execution_id}/rerun-plan

**预检基础 rerun 契约**

校验 rerun 输入并返回创建阶段必须匹配的不可变契约指纹，不创建生命周期记录。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| case_id | string | 是 | Source case ID |
| node_execution_id | string | 是 | Node execution ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | Validated rerun contract | `object` |

---

### GET /api/v1/workspaces/{id}/mowl/cases/{case_id}/node-executions/{node_execution_id}/snapshot

**获取节点执行快照**

返回指定 node_execution_id 的 immutable runtime boundary snapshot。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| case_id | string | 是 | Case ID |
| node_execution_id | string | 是 | Node execution ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | Node execution snapshot | `mowl.NodeExecutionSnapshot` |

响应字段 (`mowl.NodeExecutionSnapshot`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| activation_id | string |  | - |
| activation_sequence | integer |  | - |
| activation_token_snapshot_ids | string |  | - |
| branch_id | string |  | - |
| case_id | string |  | - |
| case_state_after_ref | string |  | - |
| case_state_before_ref | string |  | - |
| completed_at | string |  | - |
| config_snapshot_ref | string |  | - |
| consumer_name | string |  | - |
| error_snapshot_ref | string |  | - |
| input_snapshot_ref | string |  | - |
| node_execution_id | string |  | - |
| node_name | string |  | - |
| output_snapshot_ref | string |  | - |
| runtime_workitem_task_id | string |  | - |
| started_at | string |  | - |
| status | mowl.NodeExecutionStatus |  | - |
| vars_snapshot_ref | string |  | - |
| workflow_execution_id | string |  | - |
| workflow_node_instance_id | string |  | - |
| workflow_version_id | string |  | - |
| workitem_semantic_profile | string |  | - |
| workitem_type_id | string |  | - |
| workspace_id | string |  | - |

响应示例:

```json
{
  "activation_id": "string",
  "activation_sequence": 0,
  "activation_token_snapshot_ids": "string",
  "branch_id": "string",
  "case_id": "string",
  "case_state_after_ref": "string",
  "case_state_before_ref": "string",
  "completed_at": "string",
  "config_snapshot_ref": "string",
  "consumer_name": "string",
  "error_snapshot_ref": "string",
  "input_snapshot_ref": "string",
  "node_execution_id": "string",
  "node_name": "string",
  "output_snapshot_ref": "string",
  "runtime_workitem_task_id": "string",
  "started_at": "string",
  "status": {},
  "vars_snapshot_ref": "string",
  "workflow_execution_id": "string",
  "workflow_node_instance_id": "string",
  "workflow_version_id": "string",
  "workitem_semantic_profile": "string",
  "workitem_type_id": "string",
  "workspace_id": "string"
}
```

---

### POST /api/v1/workspaces/{id}/mowl/cases/{case_id}/root-assets/{root_asset_id}/node-executions/{node_execution_id}/effective-revisions

**切换节点输出 effective revisions**

CAS 更新 node-output target 的 effective revision pointers，并返回新的 effective_set_version。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| case_id | string | 是 | Case ID |
| root_asset_id | string | 是 | Root asset ID |
| node_execution_id | string | 是 | Node execution ID |

#### 请求体

类型: `mowl.SwitchEffectiveRevisionsRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| expected_effective_set_version | integer |  | - |
| switches | []mowl.SwitchEffectiveRevisionEntry |  | - |

示例:

```json
{
  "expected_effective_set_version": 0,
  "switches": [{
    "block_id": "string",
    "effective_revision_id": "string",
    "use_original": false
  }]
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | Effective set version | `object` |

---

### GET /api/v1/workspaces/{id}/mowl/cases/{case_id}/root-assets/{root_asset_id}/node-executions/{node_execution_id}/output-blocks

**列出节点输出 blocks**

按 root asset 与 node_execution_id 返回 node-output target 下的 output block refs。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| case_id | string | 是 | Case ID |
| root_asset_id | string | 是 | Root asset ID |
| node_execution_id | string | 是 | Node execution ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | Node output blocks | `object` |

---

### GET /api/v1/workspaces/{id}/mowl/cases/{case_id}/root-assets/{root_asset_id}/node-executions/{node_execution_id}/output-blocks/{block_id}/revisions

**列出节点输出 block revisions**

返回指定 node-output block 的 OutputBlockRevision 列表。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| case_id | string | 是 | Case ID |
| root_asset_id | string | 是 | Root asset ID |
| node_execution_id | string | 是 | Node execution ID |
| block_id | string | 是 | Block ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | Block revisions | `object` |

---

### POST /api/v1/workspaces/{id}/mowl/cases/{case_id}/root-assets/{root_asset_id}/node-executions/{node_execution_id}/output-blocks/{block_id}/revisions

**创建节点输出 block revision**

为 node-output block 创建新的 OutputBlockRevision，并注册 immutable content ref owner。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| case_id | string | 是 | Case ID |
| root_asset_id | string | 是 | Root asset ID |
| node_execution_id | string | 是 | Node execution ID |
| block_id | string | 是 | Block ID |

#### 请求体

类型: `mowl.CreateRevisionRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| base_revision_id | string |  | Exactly one base selector must be provided. |
| expected_base_content_hash | string |  | Optional CAS guard for the resolved base content hash. |
| expected_effective_set_version | integer |  | - |
| patch_json | string |  | - |
| revision_content_hash | string |  | - |
| revision_content_payload | string |  | - |
| revision_content_ref | string |  | Revised content: either a full ref+hash or a base+patch pair. |
| use_current_effective | boolean |  | - |
| use_original | boolean |  | - |

示例:

```json
{
  "base_revision_id": "string",
  "expected_base_content_hash": "string",
  "expected_effective_set_version": 0,
  "patch_json": "string",
  "revision_content_hash": "string",
  "revision_content_payload": "string",
  "revision_content_ref": "string",
  "use_current_effective": false,
  "use_original": false
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 201 | Created revision | `mowl.OutputBlockRevision` |

响应字段 (`mowl.OutputBlockRevision`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| base_content_hash | string |  | - |
| base_content_ref | string |  | - |
| base_effective_set_version | integer |  | - |
| base_revision_id | string |  | - |
| block_id | string |  | - |
| branch_id | string |  | - |
| case_id | string |  | - |
| created_at | string |  | - |
| created_by | string |  | - |
| node_execution_id | string |  | - |
| output_artifact_id | string |  | - |
| patch_json | string |  | - |
| revision_content_hash | string |  | - |
| revision_content_ref | string |  | - |
| revision_id | string |  | - |
| root_asset_id | string |  | - |
| status | mowl.RevisionStatus |  | - |
| target_scope | mowl.TargetScope |  | - |
| workspace_id | string |  | - |

响应示例:

```json
{
  "base_content_hash": "string",
  "base_content_ref": "string",
  "base_effective_set_version": 0,
  "base_revision_id": "string",
  "block_id": "string",
  "branch_id": "string",
  "case_id": "string",
  "created_at": "string",
  "created_by": "string",
  "node_execution_id": "string",
  "output_artifact_id": "string",
  "patch_json": "string",
  "revision_content_hash": "string",
  "revision_content_ref": "string",
  "revision_id": "string",
  "root_asset_id": "string",
  "status": {},
  "target_scope": {},
  "workspace_id": "string"
}
```

---

### POST /api/v1/workspaces/{id}/mowl/cases/{case_id}/root-assets/{root_asset_id}/node-executions/{node_execution_id}/output-blocks/{block_id}/revisions/{revision_id}/status

**更新节点输出 revision 状态**

将 node-output active revision 标记为 superseded 或 discarded，并返回新的 effective_set_version。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| case_id | string | 是 | Case ID |
| root_asset_id | string | 是 | Root asset ID |
| node_execution_id | string | 是 | Node execution ID |
| block_id | string | 是 | Block ID |
| revision_id | string | 是 | Revision ID |

#### 请求体

类型: `mowl.UpdateRevisionStatusRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| expected_effective_set_version | integer |  | - |
| status | mowl.RevisionStatus |  | - |

示例:

```json
{
  "expected_effective_set_version": 0,
  "status": {}
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | Updated revision | `object` |

---

### POST /api/v1/workspaces/{id}/mowl/cases/{case_id}/root-assets/{root_asset_id}/node-executions/{node_execution_id}/output-emissions/{target_output_emission_id}/rerun-with-revisions

**创建 revision-rerun 分支**

根据 explicit target_output_emission_id 和 revision selection 创建 downstream-with-revisions rerun branch。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| case_id | string | 是 | Source case ID |
| root_asset_id | string | 是 | Root asset ID |
| node_execution_id | string | 是 | Node execution ID |
| target_output_emission_id | string | 是 | Target output emission ID |

#### 请求体

类型: `mowl.RevisionRerunRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| mode |  |  | rerun_downstream_with_revisions |
| selected_revisions_json | []integer |  | - |
| source_effective_set_version | integer |  | - |

示例:

```json
{
  "mode": "",
  "selected_revisions_json": ["string"],
  "source_effective_set_version": 0
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 201 | Created rerun branch | `object` |

---

### GET /api/v1/workspaces/{id}/mowl/cases/{case_id}/root-assets/{root_asset_id}/output-artifacts

**列出最终输出产物**

返回指定 case_id + root_asset_id 下已提交的 FinalOutputArtifact rows。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| case_id | string | 是 | Case ID |
| root_asset_id | string | 是 | Root asset ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | Output artifacts | `object` |

---

### POST /api/v1/workspaces/{id}/mowl/cases/{case_id}/root-assets/{root_asset_id}/output-artifacts/{output_artifact_id}/effective-revisions

**切换最终产物 effective revisions**

CAS 更新 final artifact target 的 effective revision pointers，并返回新的 effective_set_version。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| case_id | string | 是 | Case ID |
| root_asset_id | string | 是 | Root asset ID |
| output_artifact_id | string | 是 | Output artifact ID |

#### 请求体

类型: `mowl.SwitchEffectiveRevisionsRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| expected_effective_set_version | integer |  | - |
| switches | []mowl.SwitchEffectiveRevisionEntry |  | - |

示例:

```json
{
  "expected_effective_set_version": 0,
  "switches": [{
    "block_id": "string",
    "effective_revision_id": "string",
    "use_original": false
  }]
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | Effective set version | `object` |

---

### GET /api/v1/workspaces/{id}/mowl/cases/{case_id}/root-assets/{root_asset_id}/output-artifacts/{output_artifact_id}/output-blocks

**列出最终产物 output blocks**

按 root asset 与 output artifact 返回最终产物作用域下的 output block refs。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| case_id | string | 是 | Case ID |
| root_asset_id | string | 是 | Root asset ID |
| output_artifact_id | string | 是 | Output artifact ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | Output blocks | `object` |

---

### GET /api/v1/workspaces/{id}/mowl/cases/{case_id}/root-assets/{root_asset_id}/output-artifacts/{output_artifact_id}/output-blocks/{block_id}/revisions

**列出最终产物 block revisions**

返回指定 final artifact block 的 OutputBlockRevision 列表。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| case_id | string | 是 | Case ID |
| root_asset_id | string | 是 | Root asset ID |
| output_artifact_id | string | 是 | Output artifact ID |
| block_id | string | 是 | Block ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | Block revisions | `object` |

---

### POST /api/v1/workspaces/{id}/mowl/cases/{case_id}/root-assets/{root_asset_id}/output-artifacts/{output_artifact_id}/output-blocks/{block_id}/revisions

**创建最终产物 block revision**

为最终产物 block 创建新的 OutputBlockRevision，并注册 immutable content ref owner。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| case_id | string | 是 | Case ID |
| root_asset_id | string | 是 | Root asset ID |
| output_artifact_id | string | 是 | Output artifact ID |
| block_id | string | 是 | Block ID |

#### 请求体

类型: `mowl.CreateRevisionRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| base_revision_id | string |  | Exactly one base selector must be provided. |
| expected_base_content_hash | string |  | Optional CAS guard for the resolved base content hash. |
| expected_effective_set_version | integer |  | - |
| patch_json | string |  | - |
| revision_content_hash | string |  | - |
| revision_content_payload | string |  | - |
| revision_content_ref | string |  | Revised content: either a full ref+hash or a base+patch pair. |
| use_current_effective | boolean |  | - |
| use_original | boolean |  | - |

示例:

```json
{
  "base_revision_id": "string",
  "expected_base_content_hash": "string",
  "expected_effective_set_version": 0,
  "patch_json": "string",
  "revision_content_hash": "string",
  "revision_content_payload": "string",
  "revision_content_ref": "string",
  "use_current_effective": false,
  "use_original": false
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 201 | Created revision | `mowl.OutputBlockRevision` |

响应字段 (`mowl.OutputBlockRevision`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| base_content_hash | string |  | - |
| base_content_ref | string |  | - |
| base_effective_set_version | integer |  | - |
| base_revision_id | string |  | - |
| block_id | string |  | - |
| branch_id | string |  | - |
| case_id | string |  | - |
| created_at | string |  | - |
| created_by | string |  | - |
| node_execution_id | string |  | - |
| output_artifact_id | string |  | - |
| patch_json | string |  | - |
| revision_content_hash | string |  | - |
| revision_content_ref | string |  | - |
| revision_id | string |  | - |
| root_asset_id | string |  | - |
| status | mowl.RevisionStatus |  | - |
| target_scope | mowl.TargetScope |  | - |
| workspace_id | string |  | - |

响应示例:

```json
{
  "base_content_hash": "string",
  "base_content_ref": "string",
  "base_effective_set_version": 0,
  "base_revision_id": "string",
  "block_id": "string",
  "branch_id": "string",
  "case_id": "string",
  "created_at": "string",
  "created_by": "string",
  "node_execution_id": "string",
  "output_artifact_id": "string",
  "patch_json": "string",
  "revision_content_hash": "string",
  "revision_content_ref": "string",
  "revision_id": "string",
  "root_asset_id": "string",
  "status": {},
  "target_scope": {},
  "workspace_id": "string"
}
```

---

### POST /api/v1/workspaces/{id}/mowl/cases/{case_id}/root-assets/{root_asset_id}/output-artifacts/{output_artifact_id}/output-blocks/{block_id}/revisions/{revision_id}/status

**更新最终产物 revision 状态**

将 active revision 标记为 superseded 或 discarded，并返回新的 effective_set_version。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| case_id | string | 是 | Case ID |
| root_asset_id | string | 是 | Root asset ID |
| output_artifact_id | string | 是 | Output artifact ID |
| block_id | string | 是 | Block ID |
| revision_id | string | 是 | Revision ID |

#### 请求体

类型: `mowl.UpdateRevisionStatusRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| expected_effective_set_version | integer |  | - |
| status | mowl.RevisionStatus |  | - |

示例:

```json
{
  "expected_effective_set_version": 0,
  "status": {}
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | Updated revision | `object` |

---

### GET /api/v1/workspaces/{id}/mowl/reruns/by-workflow-execution/{workflow_execution_id}

**获取 workflow execution 对应的 rerun 计划**

若 execution 是 runtime rerun branch，返回创建它的 rerun plan。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| workflow_execution_id | string | 是 | Workflow execution ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | Rerun plan | `mowl.RerunPlan` |

响应字段 (`mowl.RerunPlan`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| bootstrap_attempt | integer |  | - |
| bootstrap_boundary_mode | mowl.BootstrapBoundaryMode |  | - |
| bootstrap_lease_expires_at | string |  | - |
| bootstrap_lease_owner | string |  | - |
| bootstrap_spec_json | string |  | - |
| bootstrap_token_mapping_json | string |  | - |
| branch_id | string |  | - |
| contract_drift_override_json | string |  | - |
| created_at | string |  | - |
| created_by | string |  | - |
| error_message | string |  | - |
| metadata_drift_override_json | string |  | - |
| mode | mowl.RerunMode |  | - |
| new_case_id | string |  | - |
| new_workflow_execution_id | string |  | - |
| observability_parity | boolean |  | - |
| planned_execution_set_json | string |  | - |
| planning_contract_snapshot_hash | string |  | - |
| planning_contract_snapshot_json | string |  | - |
| rerun_id | string |  | - |
| revision_set_id | string |  | - |
| root_asset_id | string |  | - |
| runtime_control_model_version | string |  | - |
| side_effect_policy_decisions_json | string |  | - |
| source_boundary_token_snapshot_ids | string |  | - |
| source_branch_id | string |  | - |
| source_case_id | string |  | - |
| source_workflow_execution_id | string |  | - |
| status | mowl.RerunPlanStatus |  | - |
| target_config_override_json | string |  | - |
| target_config_override_schema_hash | string |  | - |
| target_input_activation_id | string |  | - |
| target_node_execution_id | string |  | - |
| target_output_emission_id | string |  | - |
| updated_at | string |  | - |
| workflow_version_id | string |  | - |
| workspace_id | string |  | - |

响应示例:

```json
{
  "bootstrap_attempt": 0,
  "bootstrap_boundary_mode": {},
  "bootstrap_lease_expires_at": "string",
  "bootstrap_lease_owner": "string",
  "bootstrap_spec_json": "string",
  "bootstrap_token_mapping_json": "string",
  "branch_id": "string",
  "contract_drift_override_json": "string",
  "created_at": "string",
  "created_by": "string",
  "error_message": "string",
  "metadata_drift_override_json": "string",
  "mode": {},
  "new_case_id": "string",
  "new_workflow_execution_id": "string",
  "observability_parity": false,
  "planned_execution_set_json": "string",
  "planning_contract_snapshot_hash": "string",
  "planning_contract_snapshot_json": "string",
  "rerun_id": "string",
  "revision_set_id": "string",
  "root_asset_id": "string",
  "runtime_control_model_version": "string",
  "side_effect_policy_decisions_json": "string",
  "source_boundary_token_snapshot_ids": "string",
  "source_branch_id": "string",
  "source_case_id": "string",
  "source_workflow_execution_id": "string",
  "status": {},
  "target_config_override_json": "string",
  "target_config_override_schema_hash": "string",
  "target_input_activation_id": "string",
  "target_node_execution_id": "string",
  "target_output_emission_id": "string",
  "updated_at": "string",
  "workflow_version_id": "string",
  "workspace_id": "string"
}
```

---

### GET /api/v1/workspaces/{id}/mowl/reruns/{rerun_id}

**获取 rerun 计划**

返回指定 rerun plan 及其 branch/runtime control 状态。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| rerun_id | string | 是 | Rerun ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | Rerun plan | `mowl.RerunPlan` |

响应字段 (`mowl.RerunPlan`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| bootstrap_attempt | integer |  | - |
| bootstrap_boundary_mode | mowl.BootstrapBoundaryMode |  | - |
| bootstrap_lease_expires_at | string |  | - |
| bootstrap_lease_owner | string |  | - |
| bootstrap_spec_json | string |  | - |
| bootstrap_token_mapping_json | string |  | - |
| branch_id | string |  | - |
| contract_drift_override_json | string |  | - |
| created_at | string |  | - |
| created_by | string |  | - |
| error_message | string |  | - |
| metadata_drift_override_json | string |  | - |
| mode | mowl.RerunMode |  | - |
| new_case_id | string |  | - |
| new_workflow_execution_id | string |  | - |
| observability_parity | boolean |  | - |
| planned_execution_set_json | string |  | - |
| planning_contract_snapshot_hash | string |  | - |
| planning_contract_snapshot_json | string |  | - |
| rerun_id | string |  | - |
| revision_set_id | string |  | - |
| root_asset_id | string |  | - |
| runtime_control_model_version | string |  | - |
| side_effect_policy_decisions_json | string |  | - |
| source_boundary_token_snapshot_ids | string |  | - |
| source_branch_id | string |  | - |
| source_case_id | string |  | - |
| source_workflow_execution_id | string |  | - |
| status | mowl.RerunPlanStatus |  | - |
| target_config_override_json | string |  | - |
| target_config_override_schema_hash | string |  | - |
| target_input_activation_id | string |  | - |
| target_node_execution_id | string |  | - |
| target_output_emission_id | string |  | - |
| updated_at | string |  | - |
| workflow_version_id | string |  | - |
| workspace_id | string |  | - |

响应示例:

```json
{
  "bootstrap_attempt": 0,
  "bootstrap_boundary_mode": {},
  "bootstrap_lease_expires_at": "string",
  "bootstrap_lease_owner": "string",
  "bootstrap_spec_json": "string",
  "bootstrap_token_mapping_json": "string",
  "branch_id": "string",
  "contract_drift_override_json": "string",
  "created_at": "string",
  "created_by": "string",
  "error_message": "string",
  "metadata_drift_override_json": "string",
  "mode": {},
  "new_case_id": "string",
  "new_workflow_execution_id": "string",
  "observability_parity": false,
  "planned_execution_set_json": "string",
  "planning_contract_snapshot_hash": "string",
  "planning_contract_snapshot_json": "string",
  "rerun_id": "string",
  "revision_set_id": "string",
  "root_asset_id": "string",
  "runtime_control_model_version": "string",
  "side_effect_policy_decisions_json": "string",
  "source_boundary_token_snapshot_ids": "string",
  "source_branch_id": "string",
  "source_case_id": "string",
  "source_workflow_execution_id": "string",
  "status": {},
  "target_config_override_json": "string",
  "target_config_override_schema_hash": "string",
  "target_input_activation_id": "string",
  "target_node_execution_id": "string",
  "target_output_emission_id": "string",
  "updated_at": "string",
  "workflow_version_id": "string",
  "workspace_id": "string"
}
```

---

### POST /api/v1/workspaces/{id}/mowl/reruns/{rerun_id}/cancel

**取消 rerun 分支**

提交 Catalog control-plane 终态，并 best-effort 通知 embedded 或 standalone Mowl runtime。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| rerun_id | string | 是 | Rerun ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | Cancel result | `object` |

---

### POST /api/v1/workspaces/{id}/mowl/reruns/{rerun_id}/start

**启动 rerun 分支**

物化 frozen bootstrap spec，随后让 embedded 或 standalone Mowl attach 既有 case 并调度 bootstrap tokens。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| rerun_id | string | 是 | Rerun ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | Start result | `object` |

---

### GET /api/v1/workspaces/{id}/mowl/workflow-executions/{source_workflow_execution_id}/reruns

**列出源执行的 rerun 分支**

返回 source_workflow_execution_id 派生出来的所有 runtime rerun branch。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| source_workflow_execution_id | string | 是 | Source workflow execution ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | Rerun branch list | `mowl.ListRerunsResponse` |

响应字段 (`mowl.ListRerunsResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| reruns | []mowl.RerunPlan |  | - |

响应示例:

```json
{
  "reruns": [{
    "bootstrap_attempt": 0,
    "bootstrap_boundary_mode": {},
    "bootstrap_lease_expires_at": "string",
    "bootstrap_lease_owner": "string",
    "bootstrap_spec_json": "string",
    "bootstrap_token_mapping_json": "string",
    "branch_id": "string",
    "contract_drift_override_json": "string",
    "created_at": "string",
    "created_by": "string",
    "error_message": "string",
    "metadata_drift_override_json": "string",
    "mode": {},
    "new_case_id": "string",
    "new_workflow_execution_id": "string",
    "observability_parity": false,
    "planned_execution_set_json": "string",
    "planning_contract_snapshot_hash": "string",
    "planning_contract_snapshot_json": "string",
    "rerun_id": "string",
    "revision_set_id": "string",
    "root_asset_id": "string",
    "runtime_control_model_version": "string",
    "side_effect_policy_decisions_json": "string",
    "source_boundary_token_snapshot_ids": "string",
    "source_branch_id": "string",
    "source_case_id": "string",
    "source_workflow_execution_id": "string",
    "status": {},
    "target_config_override_json": "string",
    "target_config_override_schema_hash": "string",
    "target_input_activation_id": "string",
    "target_node_execution_id": "string",
    "target_output_emission_id": "string",
    "updated_at": "string",
    "workflow_version_id": "string",
    "workspace_id": "string"
  }]
}
```

---

## Parse Result 管理

### POST /api/v1/workspaces/{id}/parse-results/export

**导出解析结果**

将解析结果导出为文件

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 请求体

类型: `catalog.ParseResultsExportRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| file_name | string |  | - |
| parser | catalog.ParseResultParser |  | - |
| results | []catalog.ParseResult |  | - |

示例:

```json
{
  "file_name": "string",
  "parser": {},
  "results": [{
    "block_type": "string",
    "content": "string",
    "disabled": false,
    "id": "string",
    "index": 0,
    "level": "string",
    "meta": {
      "fields": {}
    },
    "result_type": "string",
    "source_files": {
      "fields": {}
    },
    "underlying_result_type": "string",
    "upstream_blocks": {
      "fields": {}
    }
  }]
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 导出结果 | `catalog.ParseResultsExportResponse` |
| 400 | 参数错误 | `gin.H` |
| 500 | 内部错误 | `gin.H` |

响应字段 (`catalog.ParseResultsExportResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| files | []catalog.ParseResultsExportFile |  | - |

响应示例:

```json
{
  "files": [{
    "content": "string",
    "name": "string"
  }]
}
```

---

### POST /api/v1/workspaces/{id}/parse-results/modify

**修改解析结果**

修改指定解析结果的内容

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 请求体

类型: `catalog.ParseResultsModifyRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| content | string |  | - |
| fields | structpb.Struct |  | - |
| parser | catalog.ParseResultParser |  | - |
| result | catalog.ParseResult |  | - |

示例:

```json
{
  "content": "string",
  "fields": {
    "fields": {}
  },
  "parser": {},
  "result": {
    "block_type": "string",
    "content": "string",
    "disabled": false,
    "id": "string",
    "index": 0,
    "level": "string",
    "meta": {
      "fields": {}
    },
    "result_type": "string",
    "source_files": {
      "fields": {}
    },
    "underlying_result_type": "string",
    "upstream_blocks": {
      "fields": {}
    }
  }
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 修改后的结果 | `catalog.ParseResultsModifyResponse` |
| 400 | 参数错误 | `gin.H` |
| 500 | 内部错误 | `gin.H` |

响应字段 (`catalog.ParseResultsModifyResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| result | catalog.ParseResult |  | - |

响应示例:

```json
{
  "result": {
    "block_type": "string",
    "content": "string",
    "disabled": false,
    "id": "string",
    "index": 0,
    "level": "string",
    "meta": {
      "fields": {}
    },
    "result_type": "string",
    "source_files": {
      "fields": {}
    },
    "underlying_result_type": "string",
    "upstream_blocks": {
      "fields": {}
    }
  }
}
```

---

### POST /api/v1/workspaces/{id}/parse-results/view

**查看解析结果**

查看指定解析结果的详细视图

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 请求体

类型: `catalog.ParseResultsViewRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| parser | catalog.ParseResultParser |  | - |
| results | []catalog.ParseResult |  | - |

示例:

```json
{
  "parser": {},
  "results": [{
    "block_type": "string",
    "content": "string",
    "disabled": false,
    "id": "string",
    "index": 0,
    "level": "string",
    "meta": {
      "fields": {}
    },
    "result_type": "string",
    "source_files": {
      "fields": {}
    },
    "underlying_result_type": "string",
    "upstream_blocks": {
      "fields": {}
    }
  }]
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 解析结果视图 | `catalog.ParseResultsViewResponse` |
| 400 | 参数错误 | `gin.H` |
| 500 | 内部错误 | `gin.H` |

响应字段 (`catalog.ParseResultsViewResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| results | []catalog.ParseResultView |  | - |

响应示例:

```json
{
  "results": [{
    "chars_count": 0,
    "content": "string",
    "content_object": {
      "fields": {}
    },
    "content_type": "string",
    "end_time": "string",
    "extraction": {
      "fields": {}
    },
    "index": 0,
    "level": "string",
    "source": {
      "fields": {}
    },
    "speaker": "string",
    "start_time": "string"
  }]
}
```

---

## Semantic Model 管理

### GET /api/v1/workspaces/{id}/semantic-models

**列出语义模型**

列出指定 workspace 下的语义模型

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 查询参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page_size | integer | 否 | 分页大小 |
| page_token | string | 否 | 分页游标 |
| search | string | 否 | 模糊搜索关键词（匹配名称或描述） |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 查询成功 | `handlers.semanticModelListResponse` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`handlers.semanticModelListResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| items | []handlers.semanticModelResponse |  | - |
| next_page_token | string |  | - |
| total | integer |  | - |

响应示例:

```json
{
  "items": [{
    "created_at": 0,
    "created_by": "string",
    "description": "string",
    "files": ["string"],
    "id": 0,
    "name": "string",
    "table_set_hash": "string",
    "tables": ["string"],
    "updated_at": 0,
    "updated_by": "string"
  }],
  "next_page_token": "string",
  "total": 0
}
```

---

### POST /api/v1/workspaces/{id}/semantic-models

**创建语义模型**

在指定 workspace 下创建语义模型

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 请求体

类型: `handlers.semanticModelCreateRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| description | string |  | - |
| files | []integer |  | - |
| name | string |  | - |
| tables | []integer |  | - |

示例:

```json
{
  "description": "string",
  "files": ["string"],
  "name": "string",
  "tables": ["string"]
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 201 | 创建成功 | `handlers.semanticModelResponse` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 409 | 模型已存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`handlers.semanticModelResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| created_at | integer |  | - |
| created_by | string |  | - |
| description | string |  | - |
| files | []integer |  | - |
| id | integer |  | - |
| name | string |  | - |
| table_set_hash | string |  | - |
| tables | []integer |  | - |
| updated_at | integer |  | - |
| updated_by | string |  | - |

响应示例:

```json
{
  "created_at": 0,
  "created_by": "string",
  "description": "string",
  "files": ["string"],
  "id": 0,
  "name": "string",
  "table_set_hash": "string",
  "tables": ["string"],
  "updated_at": 0,
  "updated_by": "string"
}
```

---

### POST /api/v1/workspaces/{id}/semantic-models/import

**导入语义模型**

导入语义模型，支持单个对象或对象数组（批量导入）

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 请求体

类型: `handlers.semanticModelImportRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| description | string |  | - |
| entries | []handlers.semanticEntryUpsertRequest |  | - |
| files | []integer |  | - |
| name | string |  | - |
| tables | []integer |  | - |

示例:

```json
{
  "description": "string",
  "entries": [{
    "key": "string",
    "kind": "string",
    "spec": ["string"],
    "tables": ["string"]
  }],
  "files": ["string"],
  "name": "string",
  "tables": ["string"]
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 201 | 导入成功 | `object` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 409 | 模型或条目冲突 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

---

### GET /api/v1/workspaces/{id}/semantic-models/tags

**列出语义模型标签聚合**

列出指定 workspace 下语义模型的 KB 级标签聚合，可按 search 缩小范围

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 查询参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| search | string | 否 | 模糊搜索关键词（匹配名称或描述） |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 查询成功 | `handlers.semanticModelTagListResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`handlers.semanticModelTagListResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| items | []handlers.semanticModelTagStatResponse |  | - |

响应示例:

```json
{
  "items": [{
    "count": 0,
    "tag": "string"
  }]
}
```

---

### GET /api/v1/workspaces/{id}/semantic-models/{model_id}

**获取语义模型**

获取指定语义模型详情

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| model_id | integer | 是 | 模型 ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 查询成功 | `handlers.semanticModelResponse` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 404 | 模型不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`handlers.semanticModelResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| created_at | integer |  | - |
| created_by | string |  | - |
| description | string |  | - |
| files | []integer |  | - |
| id | integer |  | - |
| name | string |  | - |
| table_set_hash | string |  | - |
| tables | []integer |  | - |
| updated_at | integer |  | - |
| updated_by | string |  | - |

响应示例:

```json
{
  "created_at": 0,
  "created_by": "string",
  "description": "string",
  "files": ["string"],
  "id": 0,
  "name": "string",
  "table_set_hash": "string",
  "tables": ["string"],
  "updated_at": 0,
  "updated_by": "string"
}
```

---

### PUT /api/v1/workspaces/{id}/semantic-models/{model_id}

**更新语义模型**

更新指定语义模型。若名称变更，同一事务内将引用旧名称的非 disabled Agent Package 版本标记为 needs_configuration。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| model_id | integer | 是 | 模型 ID |

#### 请求体

类型: `handlers.semanticModelUpdateRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| description | string |  | - |
| files | []integer |  | - |
| knowledge_base_database_display_name | handlers.semanticModelDatabaseDisplayNameInternalRequest |  | - |
| name | string |  | - |
| tables | []integer |  | - |

示例:

```json
{
  "description": "string",
  "files": ["string"],
  "knowledge_base_database_display_name": {
    "database_id": 0,
    "display_name": "string"
  },
  "name": "string",
  "tables": ["string"]
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 更新成功 | `object` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 404 | 模型不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

---

### DELETE /api/v1/workspaces/{id}/semantic-models/{model_id}

**删除语义模型**

删除指定语义模型。同一租户事务内同步清理当前工作区普通智能体绑定与系统/共享智能体覆盖绑定中的目标知识库引用，并为引用该知识库名称的非 disabled Agent Package 版本追加 knowledge_base_deleted 诊断并转为 needs_configuration。成功响应仍为 {deleted:true}。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| model_id | integer | 是 | 模型 ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 删除成功 | `object` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 404 | 模型不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

---

### GET /api/v1/workspaces/{id}/semantic-models/{model_id}/entries

**列出语义条目**

列出指定语义模型下的语义条目

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| model_id | integer | 是 | 模型 ID |

#### 查询参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| kind | string | 否 | 条目类型过滤 |
| page_size | integer | 否 | 分页大小 |
| page_token | string | 否 | 分页游标 |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 查询成功 | `handlers.semanticEntryListResponse` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 404 | 模型不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`handlers.semanticEntryListResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| items | []handlers.semanticEntryResponse |  | - |
| next_page_token | string |  | - |
| total | integer |  | - |

响应示例:

```json
{
  "items": [{
    "created_at": 0,
    "created_by": "string",
    "id": 0,
    "key": "string",
    "kind": "string",
    "model_id": 0,
    "spec": ["string"],
    "tables": ["string"],
    "updated_at": 0,
    "updated_by": "string"
  }],
  "next_page_token": "string",
  "total": 0
}
```

---

### POST /api/v1/workspaces/{id}/semantic-models/{model_id}/entries

**创建语义条目**

在指定语义模型下创建语义条目

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| model_id | integer | 是 | 模型 ID |

#### 请求体

类型: `handlers.semanticEntryUpsertRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| key | string |  | - |
| kind | string |  | - |
| spec | []integer |  | - |
| tables | []string |  | - |

示例:

```json
{
  "key": "string",
  "kind": "string",
  "spec": ["string"],
  "tables": ["string"]
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 201 | 创建成功 | `handlers.semanticEntryResponse` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 404 | 模型不存在 | `gin.ErrorResponse` |
| 409 | 条目已存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`handlers.semanticEntryResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| created_at | integer |  | - |
| created_by | string |  | - |
| id | integer |  | - |
| key | string |  | - |
| kind | string |  | - |
| model_id | integer |  | - |
| spec | []integer |  | - |
| tables | []string |  | - |
| updated_at | integer |  | - |
| updated_by | string |  | - |

响应示例:

```json
{
  "created_at": 0,
  "created_by": "string",
  "id": 0,
  "key": "string",
  "kind": "string",
  "model_id": 0,
  "spec": ["string"],
  "tables": ["string"],
  "updated_at": 0,
  "updated_by": "string"
}
```

---

### PUT /api/v1/workspaces/{id}/semantic-models/{model_id}/entries/{entry_id}

**更新语义条目**

更新指定语义模型下的语义条目

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| model_id | integer | 是 | 模型 ID |
| entry_id | integer | 是 | 条目 ID |

#### 请求体

类型: `handlers.semanticEntryUpsertRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| key | string |  | - |
| kind | string |  | - |
| spec | []integer |  | - |
| tables | []string |  | - |

示例:

```json
{
  "key": "string",
  "kind": "string",
  "spec": ["string"],
  "tables": ["string"]
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 更新成功 | `object` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 404 | 模型或条目不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

---

### DELETE /api/v1/workspaces/{id}/semantic-models/{model_id}/entries/{entry_id}

**删除语义条目**

删除指定语义模型下的语义条目

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| model_id | integer | 是 | 模型 ID |
| entry_id | integer | 是 | 条目 ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 删除成功 | `object` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 404 | 模型或条目不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

---

### GET /api/v1/workspaces/{id}/semantic-models/{model_id}/export

**导出语义模型**

导出语义模型及其全部语义条目

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| model_id | integer | 是 | 模型 ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 导出成功 | `object` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 404 | 模型不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

---

### POST /api/v1/workspaces/{id}/semantic-models/{model_id}/validate

**校验语义模型**

对语义模型及全部语义条目执行一致性校验

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| model_id | integer | 是 | 模型 ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 校验通过 | `object` |
| 400 | 参数错误或校验失败 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 404 | 模型不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

---

## System

### GET /api/v1/system/callbacks/{provider}/{workspace_id}/{credential_ref}

**Handle provider callback**

Handles provider URL verification and event callbacks proxied by moi-backend. Supports channel callback providers such as wecom, github, feishu, and slack. Requires moi-core system API key.

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| provider | string | 是 | Callback provider |
| workspace_id | string | 是 | Workspace ID |
| credential_ref | string | 是 | Channel credential ref |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | Provider protocol response | `string` |
| 400 | Invalid callback request | `common.APIResponse` |
| 401 | Invalid callback signature | `common.APIResponse` |
| 404 | Callback credential not found | `common.APIResponse` |
| 424 | Callback credential is not ready | `common.APIResponse` |
| 502 | Callback event dispatch failed | `common.APIResponse` |
| 503 | Callback processing unavailable | `common.APIResponse` |

---

### POST /api/v1/system/callbacks/{provider}/{workspace_id}/{credential_ref}

**Handle provider callback**

Handles provider URL verification and event callbacks proxied by moi-backend. Supports channel callback providers such as wecom, github, feishu, and slack. Requires moi-core system API key.

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| provider | string | 是 | Callback provider |
| workspace_id | string | 是 | Workspace ID |
| credential_ref | string | 是 | Channel credential ref |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | Provider protocol response | `string` |
| 400 | Invalid callback request | `common.APIResponse` |
| 401 | Invalid callback signature | `common.APIResponse` |
| 404 | Callback credential not found | `common.APIResponse` |
| 424 | Callback credential is not ready | `common.APIResponse` |
| 502 | Callback event dispatch failed | `common.APIResponse` |
| 503 | Callback processing unavailable | `common.APIResponse` |

---

### POST /api/v1/system/dataconn/connector-secrets:resolve

**解析 dataconn connector secret material**

仅允许 moi-core system API key 调用 Catalog-owned connector secret resolver。请求必须携带 runtime config attestation，Catalog 负责验证后返回 secret material。

认证: 需要 API Key

#### 请求体

类型: `dataconn.ResolveConnectorSecretsRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| attestation_expires_at | integer |  | - |
| attestation_key_id | string |  | - |
| config_attestation | string |  | - |
| config_hash | string |  | - |
| connector_id | string |  | - |
| purpose | string |  | - |
| referenced_secret_refs | []string |  | - |
| request_id | string |  | - |
| secret_refs | []string |  | - |
| user_id | string |  | - |
| workspace_id | string |  | - |

示例:

```json
{
  "attestation_expires_at": 0,
  "attestation_key_id": "string",
  "config_attestation": "string",
  "config_hash": "string",
  "connector_id": "string",
  "purpose": "string",
  "referenced_secret_refs": ["string"],
  "request_id": "string",
  "secret_refs": ["string"],
  "user_id": "string",
  "workspace_id": "string"
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 解析成功 | `dataconn.ResolveConnectorSecretsResponse` |
| 400 | 请求参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 403 | 无权限 | `gin.ErrorResponse` |
| 404 | secret 不存在 | `gin.ErrorResponse` |
| 412 | secret 状态或 attestation 不匹配 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`dataconn.ResolveConnectorSecretsResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| secrets | []dataconn.ResolvedConnectorSecret |  | - |

响应示例:

```json
{
  "secrets": [{
    "content_type": "string",
    "file_name": "string",
    "secret_path": "string",
    "secret_ref": "string",
    "value": ["string"]
  }]
}
```

---

### POST /api/v1/system/dataconn/worker-execution:invoke

**调用 dataconn worker 执行**

通过 Catalog 调用 dataconn worker node。请求 payload 只透传给 worker，不在 Catalog 侧解析 connector config 或 secret refs。

认证: 需要 API Key

#### 请求体

类型: `mowl.InvokeRegisteredWorkItemRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| data | string |  | - |
| node_id | string |  | - |
| payload_bytes | []integer |  | - |
| payload_type | string |  | - |
| request_id | string |  | - |
| runtime_context | data.ExecutionContext |  | - |
| target_worker_generation | string |  | - |
| target_worker_id | string |  | target_worker_id and target_worker_generation form an immutable
worker-session binding. Both must be supplied together. Mowl fails the
request when that exact worker generation is no longer attached; it
never silently reschedules a targeted invocation. |
| timeout_ms | integer |  | - |

示例:

```json
{
  "data": "string",
  "node_id": "string",
  "payload_bytes": ["string"],
  "payload_type": "string",
  "request_id": "string",
  "runtime_context": {
    "activation_id": "string",
    "activation_sequence": 0,
    "branch_id": "string",
    "business_action_authorized": false,
    "case_id": "string",
    "effective_role_id": "string",
    "execution_contract_version": "string",
    "idempotency_key": "string",
    "is_workspace_owner": false,
    "node_execution_id": "string",
    "parallel_index": 0,
    "parallel_total": 0,
    "rerun_id": "string",
    "runtime_workitem_task_id": "string",
    "task_id": "string",
    "user_api_key": "string",
    "user_id": "string",
    "workflow_execution_id": "string",
    "workflow_node_instance_id": "string",
    "workflow_version_id": "string",
    "workitem_type_id": "string",
    "workspace_access_verified": false,
    "workspace_id": "string"
  },
  "target_worker_generation": "string",
  "target_worker_id": "string",
  "timeout_ms": 0
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 调用成功 | `mowl.InvokeRegisteredWorkItemResponse` |
| 400 | 请求参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 403 | 无权限或 node 不允许 | `gin.ErrorResponse` |
| 503 | worker 未注册或 dispatch 失败 | `gin.ErrorResponse` |
| 504 | worker 执行超时 | `gin.ErrorResponse` |

响应字段 (`mowl.InvokeRegisteredWorkItemResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| data | string |  | - |
| error | string |  | - |
| payload_bytes | []integer |  | - |
| payload_type | string |  | - |
| request_id | string |  | - |
| status | string |  | - |
| worker_generation | string |  | Opaque identity of the attached WorkerSession generation that handled
this invocation. Callers use it with target_worker_id for sticky
follow-up requests. |
| worker_id | string |  | - |

响应示例:

```json
{
  "data": "string",
  "error": "string",
  "payload_bytes": ["string"],
  "payload_type": "string",
  "request_id": "string",
  "status": "string",
  "worker_generation": "string",
  "worker_id": "string"
}
```

---

### POST /api/v1/system/runtime-actor-credentials:resolve

**Resolve a workflow runtime actor credential**

Resolves the current credential for a durable MOI actor. This endpoint is restricted to the Catalog system API key.

认证: 需要 API Key

#### 请求体

类型: `handlers.RuntimeActorCredentialResolveRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| moi_user_id | string | 是 | - |
| workspace_id | string | 是 | - |

示例:

```json
{
  "moi_user_id": "string",
  "workspace_id": "string"
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | Credential resolved | `handlers.RuntimeActorCredentialResolveResponse` |
| 400 | Invalid request | `gin.ErrorResponse` |
| 401 | Unauthenticated | `gin.ErrorResponse` |
| 403 | System API key required | `gin.ErrorResponse` |
| 503 | Credential resolution unavailable | `gin.ErrorResponse` |

响应字段 (`handlers.RuntimeActorCredentialResolveResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| api_key | string |  | - |

响应示例:

```json
{
  "api_key": "string"
}
```

---

### POST /api/v1/system/structured-load/runtime:operation

**Dispatch structured-load runtime operation**

Dispatches system structured-load runtime operations by action suffix, including :gate, :resolve, checkpoint, commit, reconcile, progress, error, and finalize operations.

认证: 需要 API Key

#### 请求体

类型: JSON 对象

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | operation response | `object` |
| 400 | invalid request | `gin.ErrorResponse` |
| 404 | operation not found | `gin.ErrorResponse` |
| 503 | structured-load service unavailable | `gin.ErrorResponse` |

---

### POST /api/v1/system/structured-load/tasks:operation

**Dispatch structured-load task operation**

Dispatches system structured-load task operations by action suffix, including :create, :get, :set-status, and :list-runs.

认证: 需要 API Key

#### 请求体

类型: JSON 对象

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | operation response | `object` |
| 400 | invalid request | `gin.ErrorResponse` |
| 404 | operation not found | `gin.ErrorResponse` |
| 503 | structured-load service unavailable | `gin.ErrorResponse` |

---

### GET /api/v1/system/upgrade/status

**获取自动升级状态**

返回当前 binary final version、final readiness 以及每个 upgrade step 的 tenant ready/failed/blocked 计数。仅 moi-core system API key 可访问。

认证: 需要 API Key

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 获取成功 | `catalog.UpgradeStatus` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 403 | 无权限 | `gin.ErrorResponse` |
| 500 | 服务内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.UpgradeStatus`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| final_ready | boolean |  | - |
| final_version | string |  | - |
| final_version_offset | integer |  | - |
| steps | []catalog.UpgradeStepStatus |  | - |

响应示例:

```json
{
  "final_ready": false,
  "final_version": "string",
  "final_version_offset": 0,
  "steps": [{
    "blocked_tenant": 0,
    "failed_tenant": 0,
    "final_version": "string",
    "final_version_offset": 0,
    "from_version": "string",
    "id": 0,
    "ready_tenant": 0,
    "state": 0,
    "to_version": "string",
    "to_version_offset": 0,
    "total_tenant": 0,
    "upgrade_order": 0,
    "upgrade_system": 0,
    "upgrade_tenant": 0
  }]
}
```

---

### GET /api/v1/system/upgrade/tenant-tasks

**查询 tenant 升级任务**

查询 tenant 升级任务，默认可用于定位 failed/blocked workspace。state 支持 created/running/ready/failed/blocked/retry_requested 或对应数字，多个值用逗号分隔。仅 moi-core system API key 可访问。

认证: 需要 API Key

#### 查询参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| state | string | 否 | 任务状态过滤，逗号分隔，默认不过滤 |
| workspace_id | string | 否 | Workspace ID 过滤 |
| upgrade_id | integer | 否 | Upgrade step ID 过滤 |
| limit | integer | 否 | 返回条数，默认 50 |
| offset | integer | 否 | 分页偏移，默认 0 |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 获取成功 | `catalog.ListUpgradeTenantTasksResponse` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 403 | 无权限 | `gin.ErrorResponse` |
| 500 | 服务内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.ListUpgradeTenantTasksResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| tasks | []catalog.UpgradeTenantTask |  | - |
| total | integer |  | - |

响应示例:

```json
{
  "tasks": [{
    "account_name": "string",
    "attempts": 0,
    "claimed_at": 0,
    "claimed_by": "string",
    "created_at": 0,
    "error_class": "string",
    "heartbeat_at": 0,
    "id": 0,
    "last_error": "string",
    "next_retry_at": 0,
    "state": 0,
    "target_version": "string",
    "target_version_offset": 0,
    "updated_at": 0,
    "upgrade_id": 0,
    "workspace_id": "string"
  }],
  "total": 0
}
```

---

### GET /api/v1/system/upgrade/tenant-tasks/{task_id}

**获取 tenant 升级任务详情**

按 task_id 获取单个 tenant 升级任务，包含 workspace、target version、attempt、claim 与错误字段。仅 moi-core system API key 可访问。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| task_id | integer | 是 | Tenant upgrade task ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 获取成功 | `catalog.UpgradeTenantTask` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 403 | 无权限 | `gin.ErrorResponse` |
| 404 | 任务不存在 | `gin.ErrorResponse` |
| 500 | 服务内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.UpgradeTenantTask`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| account_name | string |  | - |
| attempts | integer |  | - |
| claimed_at | integer |  | - |
| claimed_by | string |  | - |
| created_at | integer |  | - |
| error_class | string |  | - |
| heartbeat_at | integer |  | - |
| id | integer |  | - |
| last_error | string |  | - |
| next_retry_at | integer |  | - |
| state | integer |  | - |
| target_version | string |  | - |
| target_version_offset | integer |  | - |
| updated_at | integer |  | - |
| upgrade_id | integer |  | - |
| workspace_id | string |  | - |

响应示例:

```json
{
  "account_name": "string",
  "attempts": 0,
  "claimed_at": 0,
  "claimed_by": "string",
  "created_at": 0,
  "error_class": "string",
  "heartbeat_at": 0,
  "id": 0,
  "last_error": "string",
  "next_retry_at": 0,
  "state": 0,
  "target_version": "string",
  "target_version_offset": 0,
  "updated_at": 0,
  "upgrade_id": 0,
  "workspace_id": "string"
}
```

---

### GET /api/v1/system/upgrade/tenant-tasks/{task_id}/events

**查询 tenant 升级任务事件**

查询 moi_upgrade_tenant_events 中记录的 created/claimed/failed/blocked/retry_requested/ready 事件，用于排查失败、阻塞与 lease 竞态。仅 moi-core system API key 可访问。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| task_id | integer | 是 | Tenant upgrade task ID |

#### 查询参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| limit | integer | 否 | 返回条数，默认 100 |
| offset | integer | 否 | 分页偏移，默认 0 |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 获取成功 | `catalog.ListUpgradeTenantTaskEventsResponse` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 403 | 无权限 | `gin.ErrorResponse` |
| 500 | 服务内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.ListUpgradeTenantTaskEventsResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| events | []catalog.UpgradeTenantTaskEvent |  | - |
| total | integer |  | - |

响应示例:

```json
{
  "events": [{
    "account_name": "string",
    "attempt": 0,
    "created_at": 0,
    "error_class": "string",
    "error_code": "string",
    "error_message": "string",
    "event_type": "string",
    "id": 0,
    "operator_id": "string",
    "request_id": "string",
    "state_after": 0,
    "state_before": 0,
    "target_version": "string",
    "target_version_offset": 0,
    "tenant_task_id": 0,
    "upgrade_id": 0,
    "worker_id": "string",
    "workspace_id": "string"
  }],
  "total": 0
}
```

---

### POST /api/v1/system/upgrade/tenant-tasks/{task_id}/retry

**重试 tenant 升级任务**

默认将 failed/blocked tenant task 标记为 retry_requested。force=true 可重放 ready/failed/blocked 历史 task，但所有 retry 在 detached operation 仍持有 claim 时均 fail-closed；只有其 monitor 确认退出后才可重试。force 必须携带非空 operator_id，所有过渡都写入 moi_upgrade_tenant_events。仅 moi-core system API key 可访问。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| task_id | integer | 是 | Tenant upgrade task ID |

#### 查询参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| force | boolean | 否 | Replay a ready/failed/blocked historical task; requires operator_id and no detached operation |

#### 请求体

类型: `catalog.RetryUpgradeTenantTaskRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| operator_id | string |  | - |

示例:

```json
{
  "operator_id": "string"
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 已请求重试 | `catalog.UpgradeTenantTask` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 403 | 无权限 | `gin.ErrorResponse` |
| 404 | 任务不存在 | `gin.ErrorResponse` |
| 409 | 当前状态不允许重试 | `gin.ErrorResponse` |

响应字段 (`catalog.UpgradeTenantTask`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| account_name | string |  | - |
| attempts | integer |  | - |
| claimed_at | integer |  | - |
| claimed_by | string |  | - |
| created_at | integer |  | - |
| error_class | string |  | - |
| heartbeat_at | integer |  | - |
| id | integer |  | - |
| last_error | string |  | - |
| next_retry_at | integer |  | - |
| state | integer |  | - |
| target_version | string |  | - |
| target_version_offset | integer |  | - |
| updated_at | integer |  | - |
| upgrade_id | integer |  | - |
| workspace_id | string |  | - |

响应示例:

```json
{
  "account_name": "string",
  "attempts": 0,
  "claimed_at": 0,
  "claimed_by": "string",
  "created_at": 0,
  "error_class": "string",
  "heartbeat_at": 0,
  "id": 0,
  "last_error": "string",
  "next_retry_at": 0,
  "state": 0,
  "target_version": "string",
  "target_version_offset": 0,
  "updated_at": 0,
  "upgrade_id": 0,
  "workspace_id": "string"
}
```

---

### GET /api/v1/system/version

**获取 catalog 版本信息**

返回当前 catalog 进程的构建版本信息。仅 moi-core system API key 可访问。

认证: 需要 API Key

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 获取成功 | `handlers.SystemVersionResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 403 | 无权限 | `gin.ErrorResponse` |

响应字段 (`handlers.SystemVersionResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| branch_name | string |  | - |
| build_time | string |  | - |
| components | []handlers.ComponentVersion |  | - |
| git_commit | string |  | - |
| go_version | string |  | - |
| service | string |  | - |
| version | string |  | - |

响应示例:

```json
{
  "branch_name": "string",
  "build_time": "string",
  "components": [{
    "branch_name": "string",
    "build_time": "string",
    "component": "string",
    "error": "string",
    "git_commit": "string",
    "go_version": "string",
    "status": "string",
    "version": "string"
  }],
  "git_commit": "string",
  "go_version": "string",
  "service": "string",
  "version": "string"
}
```

---

### GET /api/v1/system/workspaces/{id}/catalogs/{catalog_id}/resolve

**解析可信 Catalog 元数据**

内部接口：返回 workspace-scoped Catalog 事实，结果不表示权限放行

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| catalog_id | integer | 是 | Catalog ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | Catalog 详情 | `catalog.Catalog` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 403 | 权限不足 | `gin.ErrorResponse` |
| 404 | Catalog 不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.Catalog`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| comment | string |  | - |
| created_at | integer |  | Unix 时间戳（秒） |
| created_by | string |  | 创建者用户 ID |
| display_bindings | []catalog.ResourceDisplayBinding |  | - |
| id | integer |  | - |
| name | string |  | - |
| updated_at | integer |  | Unix 时间戳（秒） |
| updated_by | string |  | 更新者用户 ID |

响应示例:

```json
{
  "comment": "string",
  "created_at": 0,
  "created_by": "string",
  "display_bindings": [{
    "default_text": "string",
    "display_key": "string",
    "display_owner": "string",
    "field": "string"
  }],
  "id": 0,
  "name": "string",
  "updated_at": 0,
  "updated_by": "string"
}
```

---

### GET /api/v1/system/workspaces/{id}/custom-operators/resolve

**解析可信 Custom Operator identity**

内部接口：按 workspace 内 node_id + version 唯一返回 Custom Operator；结果不表示权限放行

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 查询参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| node_id | string | 是 | Custom Operator Node ID |
| version | string | 是 | Custom Operator Version |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | Custom Operator | `catalog.CustomOperator` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 403 | 权限不足 | `gin.ErrorResponse` |
| 404 | Custom Operator 不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.CustomOperator`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| base_node_id | string |  | - |
| base_node_version | string |  | - |
| binding_config | string |  | - |
| catalog_id | integer |  | - |
| created_at | integer |  | - |
| created_by | string |  | - |
| database_id | integer |  | - |
| description | string |  | - |
| enabled | boolean |  | - |
| handler | string |  | - |
| id | integer |  | - |
| identifier | string |  | - |
| input_schema | string |  | - |
| isolation_level | string |  | - |
| kind | catalog.CustomOperatorKind |  | - |
| language | catalog.CustomOperatorLanguage |  | - |
| name | string |  | - |
| node_id | string |  | - |
| output_schema | string |  | - |
| source_file_id | string |  | - |
| updated_at | integer |  | - |
| updated_by | string |  | - |
| version | string |  | - |

响应示例:

```json
{
  "base_node_id": "string",
  "base_node_version": "string",
  "binding_config": "string",
  "catalog_id": 0,
  "created_at": 0,
  "created_by": "string",
  "database_id": 0,
  "description": "string",
  "enabled": false,
  "handler": "string",
  "id": 0,
  "identifier": "string",
  "input_schema": "string",
  "isolation_level": "string",
  "kind": {},
  "language": {},
  "name": "string",
  "node_id": "string",
  "output_schema": "string",
  "source_file_id": "string",
  "updated_at": 0,
  "updated_by": "string",
  "version": "string"
}
```

---

### GET /api/v1/system/workspaces/{id}/custom-operators/{operator_id}

**按 ID 解析可信 Custom Operator identity**

内部接口：需 system API key；按 workspace 内算子 ID 唯一返回 Custom Operator；结果不表示权限放行

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| operator_id | integer | 是 | 自定义算子 ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | Custom Operator | `catalog.CustomOperator` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 404 | Custom Operator 不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.CustomOperator`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| base_node_id | string |  | - |
| base_node_version | string |  | - |
| binding_config | string |  | - |
| catalog_id | integer |  | - |
| created_at | integer |  | - |
| created_by | string |  | - |
| database_id | integer |  | - |
| description | string |  | - |
| enabled | boolean |  | - |
| handler | string |  | - |
| id | integer |  | - |
| identifier | string |  | - |
| input_schema | string |  | - |
| isolation_level | string |  | - |
| kind | catalog.CustomOperatorKind |  | - |
| language | catalog.CustomOperatorLanguage |  | - |
| name | string |  | - |
| node_id | string |  | - |
| output_schema | string |  | - |
| source_file_id | string |  | - |
| updated_at | integer |  | - |
| updated_by | string |  | - |
| version | string |  | - |

响应示例:

```json
{
  "base_node_id": "string",
  "base_node_version": "string",
  "binding_config": "string",
  "catalog_id": 0,
  "created_at": 0,
  "created_by": "string",
  "database_id": 0,
  "description": "string",
  "enabled": false,
  "handler": "string",
  "id": 0,
  "identifier": "string",
  "input_schema": "string",
  "isolation_level": "string",
  "kind": {},
  "language": {},
  "name": "string",
  "node_id": "string",
  "output_schema": "string",
  "source_file_id": "string",
  "updated_at": 0,
  "updated_by": "string",
  "version": "string"
}
```

---

### GET /api/v1/system/workspaces/{id}/data-assets/{asset_id}/resolve-root

**解析可信 DataAsset 根 Volume**

内部接口：按 workspace 内 DataAsset ID 返回 canonical root Volume ID，结果不表示权限放行

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| asset_id | string | 是 | DataAsset ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 根 Volume ID | `handlers.dataAssetRootResponse` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 404 | DataAsset 或根 Volume 不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |
| 503 | 解析服务不可用 | `gin.ErrorResponse` |

响应字段 (`handlers.dataAssetRootResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| root_volume_id | string |  | - |

响应示例:

```json
{
  "root_volume_id": "string"
}
```

---

### GET /api/v1/system/workspaces/{id}/databases/{database_id}/resolve

**解析可信数据库元数据**

内部接口：返回 workspace-scoped Database 事实，结果不表示权限放行

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| database_id | integer | 是 | Database ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 数据库详情 | `catalog.Database` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 403 | 权限不足 | `gin.ErrorResponse` |
| 404 | 数据库不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.Database`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| catalog_id | integer |  | - |
| comment | string |  | - |
| created_at | integer |  | Unix 时间戳（秒） |
| created_by | string |  | 创建者用户 ID |
| details_visible | boolean |  | True only when the caller has direct database visibility, rather than
receiving this database as a minimal ancestor for an authorized child. |
| display_bindings | []catalog.ResourceDisplayBinding |  | - |
| id | integer |  | - |
| is_pub | boolean |  | Set by Catalog display projections when this database has an active Data
Share publication. |
| is_sub | boolean |  | Set by Catalog display projections when this database is an active Data
Share subscription projection. |
| name | string |  | - |
| updated_at | integer |  | Unix 时间戳（秒） |
| updated_by | string |  | 更新者用户 ID |

响应示例:

```json
{
  "catalog_id": 0,
  "comment": "string",
  "created_at": 0,
  "created_by": "string",
  "details_visible": false,
  "display_bindings": [{
    "default_text": "string",
    "display_key": "string",
    "display_owner": "string",
    "field": "string"
  }],
  "id": 0,
  "is_pub": false,
  "is_sub": false,
  "name": "string",
  "updated_at": 0,
  "updated_by": "string"
}
```

---

### GET /api/v1/system/workspaces/{id}/databases/{database_id}/tables/resolve

**解析数据库下的可信表集合**

内部接口：返回指定 workspace 和 Database 下的完整 Table identity 集合，结果不表示权限放行

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| database_id | integer | 是 | Database ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 表集合 | `catalog.ListTablesResponse` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 403 | 权限不足 | `gin.ErrorResponse` |
| 404 | 数据库不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.ListTablesResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| items | []catalog.Table |  | - |
| next_page_token | string |  | - |
| total | integer |  | - |

响应示例:

```json
{
  "items": [{
    "catalog_id": 0,
    "comment": "string",
    "created_at": 0,
    "created_by": "string",
    "database_id": 0,
    "extensions": {},
    "id": 0,
    "name": "string",
    "updated_at": 0,
    "updated_by": "string"
  }],
  "next_page_token": "string",
  "total": 0
}
```

---

### POST /api/v1/system/workspaces/{id}/databases:compensate-create-iam

**补偿数据库创建 IAM**

内部接口：仅回滚指定创建请求登记的 Database IAM Ownership，并校验父 Catalog 与无子资源条件

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 请求体

类型: JSON 对象

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 补偿成功 | `gin.H` |
| 400 | 参数或父资源不匹配 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 403 | 权限不足 | `gin.ErrorResponse` |
| 409 | 数据库仍有子资源 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

---

### POST /api/v1/system/workspaces/{id}/files/resolve-roots

**解析可信文件根 Volume**

内部接口：通过 Catalog containment 解析文件对应的 canonical root Volumes，结果不表示权限放行

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 请求体

类型: `handlers.resolveFileRootsRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| file_ids | []string |  | - |

示例:

```json
{
  "file_ids": ["string"]
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 根 Volume ID 列表 | `handlers.resolveFileRootsResponse` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 403 | 权限不足 | `gin.ErrorResponse` |
| 404 | 文件或 Volume 不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`handlers.resolveFileRootsResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| root_volume_ids | []string |  | - |

响应示例:

```json
{
  "root_volume_ids": ["string"]
}
```

---

### GET /api/v1/system/workspaces/{id}/structured-load/target-databases/{database_id}/resolve

**解析结构化载入目标库**

内部接口：通过 system API key 获取结构化载入目标库元数据，不经过用户对象权限校验

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| database_id | integer | 是 | Database ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 数据库详情 | `catalog.Database` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 403 | 权限不足 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.Database`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| catalog_id | integer |  | - |
| comment | string |  | - |
| created_at | integer |  | Unix 时间戳（秒） |
| created_by | string |  | 创建者用户 ID |
| details_visible | boolean |  | True only when the caller has direct database visibility, rather than
receiving this database as a minimal ancestor for an authorized child. |
| display_bindings | []catalog.ResourceDisplayBinding |  | - |
| id | integer |  | - |
| is_pub | boolean |  | Set by Catalog display projections when this database has an active Data
Share publication. |
| is_sub | boolean |  | Set by Catalog display projections when this database is an active Data
Share subscription projection. |
| name | string |  | - |
| updated_at | integer |  | Unix 时间戳（秒） |
| updated_by | string |  | 更新者用户 ID |

响应示例:

```json
{
  "catalog_id": 0,
  "comment": "string",
  "created_at": 0,
  "created_by": "string",
  "details_visible": false,
  "display_bindings": [{
    "default_text": "string",
    "display_key": "string",
    "display_owner": "string",
    "field": "string"
  }],
  "id": 0,
  "is_pub": false,
  "is_sub": false,
  "name": "string",
  "updated_at": 0,
  "updated_by": "string"
}
```

---

### GET /api/v1/system/workspaces/{id}/structured-load/target-databases/{database_id}/runtime/resolve

**解析结构化载入目标库运行时**

内部接口：通过 system API key 获取结构化载入目标库元数据和已就绪 runtime 引用，不经过用户对象权限校验

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| database_id | integer | 是 | Database ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 数据库详情与结构化载入 runtime 引用 | `dataconn.StructuredLoadTargetDatabaseResolution` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 403 | 权限不足 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`dataconn.StructuredLoadTargetDatabaseResolution`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| database | catalog.Database |  | - |
| runtime_ref | dataconn.StructuredLoadRuntimeRef |  | - |

响应示例:

```json
{
  "database": {
    "catalog_id": 0,
    "comment": "string",
    "created_at": 0,
    "created_by": "string",
    "details_visible": false,
    "display_bindings": [{
      "default_text": "string",
      "display_key": "string",
      "display_owner": "string",
      "field": "string"
    }],
    "id": 0,
    "is_pub": false,
    "is_sub": false,
    "name": "string",
    "updated_at": 0,
    "updated_by": "string"
  },
  "runtime_ref": {
    "account_name": "string",
    "commit_evidence_table": "string",
    "reconcile_attestation_table": "string",
    "runtime_checkpoint_table": "string",
    "runtime_database": "string",
    "runtime_ref_id": "string",
    "schema_version": 0,
    "target_fence_table": "string",
    "workspace_id": "string"
  }
}
```

---

### GET /api/v1/system/workspaces/{id}/structured-load/target-tables/{table_id}/resolve

**解析结构化载入目标表**

内部接口：通过 system API key 获取结构化载入目标表、数据库和目录元数据，不经过用户对象权限校验

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| table_id | integer | 是 | Table ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 表详情 | `catalog.GetTableResponse` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 403 | 权限不足 | `gin.ErrorResponse` |
| 404 | 表、数据库或 Catalog 不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.GetTableResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| catalog | catalog.Catalog |  | - |
| database | catalog.Database |  | - |
| table | catalog.Table |  | - |

响应示例:

```json
{
  "catalog": {
    "comment": "string",
    "created_at": 0,
    "created_by": "string",
    "display_bindings": [{
      "default_text": "string",
      "display_key": "string",
      "display_owner": "string",
      "field": "string"
    }],
    "id": 0,
    "name": "string",
    "updated_at": 0,
    "updated_by": "string"
  },
  "database": {
    "catalog_id": 0,
    "comment": "string",
    "created_at": 0,
    "created_by": "string",
    "details_visible": false,
    "display_bindings": [{
      "default_text": "string",
      "display_key": "string",
      "display_owner": "string",
      "field": "string"
    }],
    "id": 0,
    "is_pub": false,
    "is_sub": false,
    "name": "string",
    "updated_at": 0,
    "updated_by": "string"
  },
  "table": {
    "catalog_id": 0,
    "comment": "string",
    "created_at": 0,
    "created_by": "string",
    "database_id": 0,
    "extensions": {},
    "id": 0,
    "name": "string",
    "updated_at": 0,
    "updated_by": "string"
  }
}
```

---

### GET /api/v1/system/workspaces/{id}/tables/{table_id}/resolve

**解析可信表元数据**

内部接口：返回 workspace-scoped Table、Database 与 Catalog 事实，结果不表示权限放行

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| table_id | integer | 是 | Table ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 表详情 | `catalog.GetTableResponse` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 403 | 权限不足 | `gin.ErrorResponse` |
| 404 | 表、数据库或 Catalog 不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.GetTableResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| catalog | catalog.Catalog |  | - |
| database | catalog.Database |  | - |
| table | catalog.Table |  | - |

响应示例:

```json
{
  "catalog": {
    "comment": "string",
    "created_at": 0,
    "created_by": "string",
    "display_bindings": [{
      "default_text": "string",
      "display_key": "string",
      "display_owner": "string",
      "field": "string"
    }],
    "id": 0,
    "name": "string",
    "updated_at": 0,
    "updated_by": "string"
  },
  "database": {
    "catalog_id": 0,
    "comment": "string",
    "created_at": 0,
    "created_by": "string",
    "details_visible": false,
    "display_bindings": [{
      "default_text": "string",
      "display_key": "string",
      "display_owner": "string",
      "field": "string"
    }],
    "id": 0,
    "is_pub": false,
    "is_sub": false,
    "name": "string",
    "updated_at": 0,
    "updated_by": "string"
  },
  "table": {
    "catalog_id": 0,
    "comment": "string",
    "created_at": 0,
    "created_by": "string",
    "database_id": 0,
    "extensions": {},
    "id": 0,
    "name": "string",
    "updated_at": 0,
    "updated_by": "string"
  }
}
```

---

### POST /api/v1/system/workspaces/{id}/tables:compensate-create-iam

**补偿表创建 IAM**

内部接口：仅回滚指定创建请求登记的 Table IAM Ownership，并校验父 Database

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 请求体

类型: JSON 对象

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 补偿成功 | `gin.H` |
| 400 | 参数或父资源不匹配 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 403 | 权限不足 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

---

### GET /api/v1/system/workspaces/{id}/volumes/{volume_id}/resolve-root

**解析可信根 Volume**

内部接口：沿 workspace 内父链返回 canonical root Volume，结果不表示权限放行

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| volume_id | integer | 是 | Volume ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 根 Volume | `catalog.Volume` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 403 | 权限不足 | `gin.ErrorResponse` |
| 404 | Volume 不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.Volume`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| catalog_id | integer |  | - |
| comment | string |  | - |
| created_at | integer |  | Unix 时间戳（秒） |
| created_by | string |  | 创建者用户 ID |
| database_id | integer |  | - |
| deleted | boolean |  | 软删除支持 - Requirements: 10.1 |
| deleted_at | integer |  | 删除时间（Unix时间戳，秒），仅当 deleted=true 时有值 |
| display_bindings | []catalog.ResourceDisplayBinding |  | - |
| id | integer |  | - |
| name | string |  | - |
| parent_id | integer |  | 层级结构支持 - Requirements: 9.1 |
| save_path | string |  | - |
| trigger_binding |  |  | 作为工作流触发器的绑定状态（只读） |
| updated_at | integer |  | Unix 时间戳（秒） |
| updated_by | string |  | 更新者用户 ID |

响应示例:

```json
{
  "catalog_id": 0,
  "comment": "string",
  "created_at": 0,
  "created_by": "string",
  "database_id": 0,
  "deleted": false,
  "deleted_at": 0,
  "display_bindings": [{
    "default_text": "string",
    "display_key": "string",
    "display_owner": "string",
    "field": "string"
  }],
  "id": 0,
  "name": "string",
  "parent_id": 0,
  "save_path": "string",
  "trigger_binding": "",
  "updated_at": 0,
  "updated_by": "string"
}
```

---

## System - Data Share Migration

### GET /api/v1/system/data-share/drift

**查询 Data Share 迁移漂移**

认证: 需要 API Key

#### 查询参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page_size | integer | 否 | 分页大小，最大 100 |
| page_token | string | 否 | 下一页 token |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | OK | `datashare.DataShareMigrationDriftPage` |
| 400 | Bad Request | `gin.ErrorResponse` |
| 403 | Forbidden | `gin.ErrorResponse` |
| 500 | Internal Server Error | `gin.ErrorResponse` |

响应字段 (`datashare.DataShareMigrationDriftPage`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| classifications | []datashare.DataShareMigrationClassificationCount |  | - |
| next_page_token | string |  | - |
| publication_projection_equal | boolean |  | - |
| results | []datashare.DataShareMigrationDriftResult |  | - |
| skipped_reasons | []datashare.DataShareMigrationSkipCount |  | - |
| subscription_projection_equal | boolean |  | - |
| total_results | integer |  | - |
| unresolved_objects | integer |  | - |

响应示例:

```json
{
  "classifications": [{
    "count": 0,
    "kind": "string"
  }],
  "next_page_token": "string",
  "publication_projection_equal": false,
  "results": [{
    "identity": "string",
    "input_digest": "string",
    "kind": "string",
    "proof_kind": "string",
    "reason": "string",
    "resolution_status": "string"
  }],
  "skipped_reasons": [{
    "count": 0,
    "reason": "string"
  }],
  "subscription_projection_equal": false,
  "total_results": 0,
  "unresolved_objects": 0
}
```

---

### POST /api/v1/system/data-share/drift/cleanup

**清理 provenance 明确的 Data Share 历史遗留发布**

认证: 需要 API Key

#### 请求体

类型: `datashare.DataShareOrphanCleanupRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| apply | boolean |  | - |
| expected_mo_comment | string |  | - |
| publication_id | integer |  | - |
| publication_name | string |  | - |
| request_id | string |  | - |
| revoke_publication_only | boolean |  | - |
| source_workspace_id | string |  | - |
| targets | []datashare.DataShareOrphanCleanupTarget |  | - |
| trace_id | string |  | - |

示例:

```json
{
  "apply": false,
  "expected_mo_comment": "string",
  "publication_id": 0,
  "publication_name": "string",
  "request_id": "string",
  "revoke_publication_only": false,
  "source_workspace_id": "string",
  "targets": [{
    "catalog_id": 0,
    "database_id": 0,
    "database_name": "string",
    "subscription_id": 0,
    "target_workspace_id": "string"
  }],
  "trace_id": "string"
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | OK | `datashare.DataShareOrphanCleanupResult` |
| 400 | Bad Request | `gin.ErrorResponse` |
| 403 | Forbidden | `gin.ErrorResponse` |
| 409 | Conflict | `gin.ErrorResponse` |
| 500 | Internal Server Error | `gin.ErrorResponse` |

响应字段 (`datashare.DataShareOrphanCleanupResult`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| active_targets | []datashare.DataShareOrphanTargetObservation |  | - |
| applied | boolean |  | - |
| residual_targets | []datashare.DataShareOrphanTargetObservation |  | - |
| saga_id | string |  | - |

响应示例:

```json
{
  "active_targets": [{
    "database_name": "string",
    "target_account_name": "string",
    "target_workspace_id": "string"
  }],
  "applied": false,
  "residual_targets": [{
    "database_name": "string",
    "target_account_name": "string",
    "target_workspace_id": "string"
  }],
  "saga_id": "string"
}
```

---

### POST /api/v1/system/data-share/drift:reconcile

**修复 global_paused 下的 Data Share 投影漂移**

认证: 需要 API Key

#### 请求体

类型: `handlers.ReconcileDataShareDriftRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| migration_id | string | 是 | - |

示例:

```json
{
  "migration_id": "string"
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | OK | `datashare.DataShareMigrationResult` |
| 400 | Bad Request | `gin.ErrorResponse` |
| 403 | Forbidden | `gin.ErrorResponse` |
| 409 | Conflict | `gin.ErrorResponse` |
| 503 | Service Unavailable | `gin.ErrorResponse` |

响应字段 (`datashare.DataShareMigrationResult`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| eligible_imported | integer |  | - |
| examples | []datashare.MigrationResultExample |  | - |
| record_skipped | integer |  | - |
| scanned_publications | integer |  | - |
| scanned_relations | integer |  | - |
| skipped_reasons | object |  | - |
| system_errors | integer |  | - |
| unresolved_objects | integer |  | - |

响应示例:

```json
{
  "eligible_imported": 0,
  "examples": [{
    "identity": "string",
    "input_digest": "string",
    "kind": "string",
    "proof_kind": "string",
    "reason": "string",
    "resolution_status": "string"
  }],
  "record_skipped": 0,
  "scanned_publications": 0,
  "scanned_relations": 0,
  "skipped_reasons": {},
  "system_errors": 0,
  "unresolved_objects": 0
}
```

---

### GET /api/v1/system/data-share/migration

**获取 Data Share 全局迁移状态**

认证: 需要 API Key

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | OK | `datashare.DataShareMigrationAdminStatus` |
| 403 | Forbidden | `gin.ErrorResponse` |
| 500 | Internal Server Error | `gin.ErrorResponse` |

响应字段 (`datashare.DataShareMigrationAdminStatus`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| migration_id | string |  | - |
| mode | string |  | - |
| revision | integer |  | - |

响应示例:

```json
{
  "migration_id": "string",
  "mode": "string",
  "revision": 0
}
```

---

### POST /api/v1/system/data-share/migration:run

**执行 Data Share 全局迁移动作**

认证: 需要 API Key

#### 请求体

类型: `handlers.RunDataShareMigrationRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| action | string | 是 | - |
| approval | datashare.DataShareCutoverFinalizationApproval |  | - |
| expected_revision | integer |  | - |
| migration_id | string | 是 | - |

示例:

```json
{
  "action": "string",
  "approval": {
    "all_cn_data_share_ddl_clear": false,
    "catalog_workload_restarted": false,
    "compatibility_preflight_complete": false,
    "skipped_live_mo_facts_reviewed": false
  },
  "expected_revision": 0,
  "migration_id": "string"
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | OK | `object` |
| 400 | Bad Request | `gin.ErrorResponse` |
| 403 | Forbidden | `gin.ErrorResponse` |
| 409 | Conflict | `gin.ErrorResponse` |
| 503 | Service Unavailable | `gin.ErrorResponse` |

---

### POST /api/v1/system/data-share/recovery/terminate

**明确终止未产生副作用的 Data Share Failed Saga**

认证: 需要 API Key

#### 请求体

类型: `datashare.DataShareSagaTerminationRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| apply | boolean |  | - |
| approval | datashare.DataShareDDLRecoveryApproval |  | - |
| saga_id | string |  | - |

示例:

```json
{
  "apply": false,
  "approval": {
    "all_cn_data_share_ddl_drained": false,
    "catalog_workload_stopped": false,
    "exact_readback_confirmed": false
  },
  "saga_id": "string"
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | OK | `datashare.DataShareSagaTerminationResult` |
| 400 | Bad Request | `gin.ErrorResponse` |
| 403 | Forbidden | `gin.ErrorResponse` |
| 409 | Conflict | `gin.ErrorResponse` |
| 500 | Internal Server Error | `gin.ErrorResponse` |

响应字段 (`datashare.DataShareSagaTerminationResult`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| saga_id | string |  | - |
| saga_name | string |  | - |
| status | string |  | - |

响应示例:

```json
{
  "saga_id": "string",
  "saga_name": "string",
  "status": "string"
}
```

---

### POST /api/v1/system/data-share/recovery:resume

**严格恢复 Data Share 外部 DDL Saga**

认证: 需要 API Key

#### 请求体

类型: `datashare.DataShareDDLRecoveryRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| apply | boolean |  | - |
| approval | datashare.DataShareDDLRecoveryApproval |  | - |
| saga_id | string |  | - |

示例:

```json
{
  "apply": false,
  "approval": {
    "all_cn_data_share_ddl_drained": false,
    "catalog_workload_stopped": false,
    "exact_readback_confirmed": false
  },
  "saga_id": "string"
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | OK | `datashare.DataShareDDLRecoveryResult` |
| 400 | Bad Request | `gin.ErrorResponse` |
| 403 | Forbidden | `gin.ErrorResponse` |
| 409 | Conflict | `gin.ErrorResponse` |
| 500 | Internal Server Error | `gin.ErrorResponse` |

响应字段 (`datashare.DataShareDDLRecoveryResult`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| resolution | datashare.DataShareDDLRecoveryResolution |  | - |
| saga_id | string |  | - |
| saga_name | string |  | - |
| status | string |  | - |
| step_name | string |  | - |

响应示例:

```json
{
  "resolution": {},
  "saga_id": "string",
  "saga_name": "string",
  "status": "string",
  "step_name": "string"
}
```

---

## System - Default AI

### GET /api/v1/system/default-ai-services/config

**获取系统默认 AI 服务配置**

获取 system DB 中的 LLM、Embedding、File Parser 默认后端配置。仅 moi-core system API key 可访问。

认证: 需要 API Key

#### 查询参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| service_type | array | 否 | 服务类型，可重复传入：llm、embedding、file_parser |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 获取成功 | `systemdefaultai.Config` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 403 | 无权限 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`systemdefaultai.Config`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| services | []systemdefaultai.ServiceConfig |  | - |

响应示例:

```json
{
  "services": [{
    "backends": [{
      "api_key_encrypted": "string",
      "api_keys_encrypted": ["string"],
      "created_at": 0,
      "endpoints": [{
        "address": "string",
        "backend_id": 0,
        "created_at": 0,
        "id": 0,
        "status": "string",
        "updated_at": 0
      }],
      "id": 0,
      "models": ["string"],
      "name": "string",
      "priority": 0,
      "reasoning_control_protocol": "string",
      "status": "string",
      "supported_mime_types": ["string"],
      "timeout_seconds": 0,
      "type": "string",
      "updated_at": 0
    }],
    "router_config": {
      "enable_session_affinity": false,
      "health_check_interval_seconds": 0,
      "max_retries": 0,
      "strategy": "string"
    },
    "service_type": "string",
    "version": 0
  }]
}
```

---

### PUT /api/v1/system/default-ai-services/config

**写入系统默认 AI 服务配置**

替换请求中指定服务类型的系统默认 LLM、Embedding、File Parser 后端配置。仅 moi-core system API key 可访问。

认证: 需要 API Key

#### 请求体

类型: `systemdefaultai.Config`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| services | []systemdefaultai.ServiceConfig |  | - |

示例:

```json
{
  "services": [{
    "backends": [{
      "api_key_encrypted": "string",
      "api_keys_encrypted": ["string"],
      "created_at": 0,
      "endpoints": [{
        "address": "string",
        "backend_id": 0,
        "created_at": 0,
        "id": 0,
        "status": "string",
        "updated_at": 0
      }],
      "id": 0,
      "models": ["string"],
      "name": "string",
      "priority": 0,
      "reasoning_control_protocol": "string",
      "status": "string",
      "supported_mime_types": ["string"],
      "timeout_seconds": 0,
      "type": "string",
      "updated_at": 0
    }],
    "router_config": {
      "enable_session_affinity": false,
      "health_check_interval_seconds": 0,
      "max_retries": 0,
      "strategy": "string"
    },
    "service_type": "string",
    "version": 0
  }]
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 写入成功 | `systemdefaultai.Config` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 403 | 无权限 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`systemdefaultai.Config`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| services | []systemdefaultai.ServiceConfig |  | - |

响应示例:

```json
{
  "services": [{
    "backends": [{
      "api_key_encrypted": "string",
      "api_keys_encrypted": ["string"],
      "created_at": 0,
      "endpoints": [{
        "address": "string",
        "backend_id": 0,
        "created_at": 0,
        "id": 0,
        "status": "string",
        "updated_at": 0
      }],
      "id": 0,
      "models": ["string"],
      "name": "string",
      "priority": 0,
      "reasoning_control_protocol": "string",
      "status": "string",
      "supported_mime_types": ["string"],
      "timeout_seconds": 0,
      "type": "string",
      "updated_at": 0
    }],
    "router_config": {
      "enable_session_affinity": false,
      "health_check_interval_seconds": 0,
      "max_retries": 0,
      "strategy": "string"
    },
    "service_type": "string",
    "version": 0
  }]
}
```

---

## System - Runtime Dynamic Config

### GET /api/v1/system/runtime-configs/{namespace}/{config_key}

**获取全局 Runtime Dynamic Config**

读取已注册的全局动态配置。仅原始 moi-core system API key 可访问。响应 ETag 是 Global Revision。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| namespace | string | 是 | 配置命名空间 |
| config_key | string | 是 | 配置键 |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | OK | `handlers.RuntimeDynamicConfigResponse` |
| 400 | Bad Request | `gin.ErrorResponse` |
| 401 | Unauthorized | `gin.ErrorResponse` |
| 403 | Forbidden | `gin.ErrorResponse` |
| 404 | Not Found | `gin.ErrorResponse` |
| 500 | Internal Server Error | `gin.ErrorResponse` |

响应字段 (`handlers.RuntimeDynamicConfigResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| config_key | string |  | - |
| created_at | string |  | - |
| namespace | string |  | - |
| revision | integer |  | - |
| schema_version | integer |  | - |
| scope_id | string |  | - |
| scope_type | string |  | - |
| updated_at | string |  | - |
| updated_by | string |  | - |
| value | object |  | - |

响应示例:

```json
{
  "config_key": "string",
  "created_at": "string",
  "namespace": "string",
  "revision": 0,
  "schema_version": 0,
  "scope_id": "string",
  "scope_type": "string",
  "updated_at": "string",
  "updated_by": "string",
  "value": {}
}
```

---

### PUT /api/v1/system/runtime-configs/{namespace}/{config_key}

**写入全局 Runtime Dynamic Config**

expected_revision=0 创建；大于 0 时执行 Revision CAS 更新。可用 If-Match 代替 body expected_revision。仅原始 moi-core system API key 可访问。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| namespace | string | 是 | 配置命名空间 |
| config_key | string | 是 | 配置键 |

#### 请求体

类型: `handlers.PutRuntimeDynamicConfigRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| expected_revision | integer |  | - |
| schema_version | integer | 是 | - |
| value | object | 是 | - |

示例:

```json
{
  "expected_revision": 0,
  "schema_version": 0,
  "value": {}
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | OK | `handlers.RuntimeDynamicConfigResponse` |
| 400 | Bad Request | `gin.ErrorResponse` |
| 401 | Unauthorized | `gin.ErrorResponse` |
| 403 | Forbidden | `gin.ErrorResponse` |
| 409 | Conflict | `gin.ErrorResponse` |
| 412 | Precondition Failed | `gin.ErrorResponse` |
| 500 | Internal Server Error | `gin.ErrorResponse` |

响应字段 (`handlers.RuntimeDynamicConfigResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| config_key | string |  | - |
| created_at | string |  | - |
| namespace | string |  | - |
| revision | integer |  | - |
| schema_version | integer |  | - |
| scope_id | string |  | - |
| scope_type | string |  | - |
| updated_at | string |  | - |
| updated_by | string |  | - |
| value | object |  | - |

响应示例:

```json
{
  "config_key": "string",
  "created_at": "string",
  "namespace": "string",
  "revision": 0,
  "schema_version": 0,
  "scope_id": "string",
  "scope_type": "string",
  "updated_at": "string",
  "updated_by": "string",
  "value": {}
}
```

---

## System Builtin File

### GET /api/v1/system/builtin-files/{file_id}

**查询内置共享文件**

查询内置共享文件的大小和内容摘要。仅 system API key 可调用

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| file_id | string | 是 | Builtin file ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | OK | `handlers.BuiltinFileMetadata` |
| 400 | Bad Request | `gin.ErrorResponse` |
| 404 | Not Found | `gin.ErrorResponse` |
| 500 | Internal Server Error | `gin.ErrorResponse` |

响应字段 (`handlers.BuiltinFileMetadata`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| file_id | string |  | - |
| md5 | string |  | - |
| size | integer |  | - |

响应示例:

```json
{
  "file_id": "string",
  "md5": "string",
  "size": 0
}
```

---

### POST /api/v1/system/builtin-files/{file_id}

**发布内置共享文件**

以保留 ID 发布不可变的共享文件；相同 ID 和内容幂等成功，不同内容返回冲突。仅 system API key 可调用

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| file_id | string | 是 | Builtin file ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | OK | `handlers.BuiltinFileMetadata` |
| 201 | Created | `handlers.BuiltinFileMetadata` |
| 400 | Bad Request | `gin.ErrorResponse` |
| 409 | Conflict | `gin.ErrorResponse` |
| 500 | Internal Server Error | `gin.ErrorResponse` |

响应字段 (`handlers.BuiltinFileMetadata`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| file_id | string |  | - |
| md5 | string |  | - |
| size | integer |  | - |

响应示例:

```json
{
  "file_id": "string",
  "md5": "string",
  "size": 0
}
```

响应字段 (`handlers.BuiltinFileMetadata`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| file_id | string |  | - |
| md5 | string |  | - |
| size | integer |  | - |

响应示例:

```json
{
  "file_id": "string",
  "md5": "string",
  "size": 0
}
```

---

### POST /api/v1/system/workspaces/{id}/volumes/{volume_id}/builtin-files:attach

**挂载内置共享文件**

在一个租户事务中写入内置文件元数据和 Volume 引用，不复制共享对象。仅 system API key 可调用

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| volume_id | integer | 是 | Volume ID |

#### 请求体

类型: `handlers.AttachBuiltinFilesRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| items | []handlers.AttachBuiltinFileItem | 是 | - |

示例:

```json
{
  "items": [{
    "content_type": "string",
    "file_id": "string",
    "md5": "string",
    "original_name": "string",
    "size": 0
  }]
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | OK | `handlers.AttachBuiltinFilesResponse` |
| 400 | Bad Request | `gin.ErrorResponse` |
| 404 | Not Found | `gin.ErrorResponse` |
| 409 | Conflict | `gin.ErrorResponse` |
| 500 | Internal Server Error | `gin.ErrorResponse` |

响应字段 (`handlers.AttachBuiltinFilesResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| file_ids | []string |  | - |

响应示例:

```json
{
  "file_ids": ["string"]
}
```

---

## System Resource Display

### POST /api/v1/system/workspaces/{id}/system-resource-display/mappings

**绑定资源展示映射**

为 producer 显式登记的资源字段绑定 display owner/key metadata

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 请求体

类型: `handlers.EnsureResourceDisplayMappingsRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| bindings | []handlers.ResourceDisplayBindingRequest |  | - |

示例:

```json
{
  "bindings": [{
    "default_text": "string",
    "display_key": "string",
    "display_owner": "string",
    "field": "string",
    "resource_id": "string",
    "resource_type": "string"
  }]
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 绑定成功 | `gin.H` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 403 | 非系统 API key | `gin.ErrorResponse` |
| 404 | 资源不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

---

## Task 管理

### GET /api/v1/workspaces/{id}/tasks

**列出 Task**

列出指定 workspace 中的所有 Task

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 查询参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| status | integer | 否 | 按状态过滤 |
| periodic_only | string | 否 | 仅显示周期性 Task（true/false） |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | Task 列表 | `object` |
| 400 | 参数错误 | `object` |
| 401 | 未认证 | `object` |
| 500 | 内部错误 | `object` |

---

### POST /api/v1/workspaces/{id}/tasks

**创建 Task**

在指定 workspace 中创建新的 Task

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 请求体

类型: JSON 对象

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 201 | 创建成功 | `object` |
| 400 | 参数错误 | `object` |
| 401 | 未认证 | `object` |
| 500 | 内部错误 | `object` |

---

### GET /api/v1/workspaces/{id}/tasks/{task_id}

**获取 Task**

根据 ID 获取指定 Task 的详细信息

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| task_id | string | 是 | Task ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | Task 详情 | `object` |
| 400 | 参数错误 | `object` |
| 404 | Task 不存在 | `object` |
| 500 | 内部错误 | `object` |

---

### POST /api/v1/workspaces/{id}/tasks/{task_id}/cancel

**取消 Task**

取消指定 Task 的执行

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| task_id | string | 是 | Task ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 取消成功 | `object` |
| 400 | 参数错误 | `object` |
| 401 | 未认证 | `object` |
| 500 | 内部错误 | `object` |

---

### GET /api/v1/workspaces/{id}/tasks/{task_id}/cases

**获取 Task Cases**

获取指定 Task 的所有 Case 列表

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| task_id | string | 是 | Task ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | Case 列表 | `object` |
| 400 | 参数错误 | `object` |
| 500 | 内部错误 | `object` |

---

### GET /api/v1/workspaces/{id}/tasks/{task_id}/cases/{case_id}/status

**获取 Case 状态**

获取指定 Task Case 的执行状态

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| task_id | string | 是 | Task ID |
| case_id | string | 是 | Case ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | Case 状态 | `object` |
| 400 | 参数错误 | `object` |
| 404 | Case 不存在 | `object` |
| 500 | 内部错误 | `object` |

---

### POST /api/v1/workspaces/{id}/tasks/{task_id}/trigger

**触发 Task**

立即触发指定 Task 执行

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| task_id | string | 是 | Task ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 触发成功，返回 task_id 和 case_id | `object` |
| 400 | 参数错误 | `object` |
| 401 | 未认证 | `object` |
| 403 | 无权限 | `object` |
| 404 | Task 不存在 | `object` |
| 500 | 内部错误 | `object` |

---

## User 管理

### GET /api/v1/users

**列出用户**

列出所有用户

认证: 需要 API Key

#### 查询参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page_size | integer | 否 | 每页数量 |
| page_token | string | 否 | 分页令牌 |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 用户列表 | `[]github_com_matrixflow_moi-core_model_user.User` |
| 500 | 内部错误 | `gin.ErrorResponse` |

---

### POST /api/v1/users

**创建用户**

创建新用户，仅系统用户可操作

认证: 需要 API Key

#### 请求体

类型: `user.CreateUserRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| email | string |  | - |
| nickname | string |  | - |
| password | string |  | - |
| passwordless | boolean |  | Passwordless users are provisioned by a trusted identity provider. Core
creates an internal random credential, but does not expose it as a
user-supplied password. |
| phone | string |  | - |
| username | string |  | 用户名，用于数据库登录，必须唯一 |

示例:

```json
{
  "email": "string",
  "nickname": "string",
  "password": "string",
  "passwordless": false,
  "phone": "string",
  "username": "string"
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 201 | 创建成功 | `github_com_matrixflow_moi-core_model_user.User` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 403 | 无权限 | `gin.ErrorResponse` |
| 409 | 邮箱已存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`github_com_matrixflow_moi-core_model_user.User`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| created_at | integer |  | Unix 时间戳（秒） |
| email | string |  | - |
| id | string |  | - |
| nickname | string |  | - |
| phone | string |  | - |
| status | user.UserStatus |  | - |
| updated_at | integer |  | Unix 时间戳（秒） |
| username | string |  | 用户名，用于数据库登录，必须唯一 |

响应示例:

```json
{
  "created_at": 0,
  "email": "string",
  "id": "string",
  "nickname": "string",
  "phone": "string",
  "status": {},
  "updated_at": 0,
  "username": "string"
}
```

---

### GET /api/v1/users/email/{email}

**根据邮箱获取用户**

根据邮箱地址获取用户信息

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| email | string | 是 | 用户邮箱（URL 编码） |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 用户详情 | `github_com_matrixflow_moi-core_model_user.User` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 404 | 用户不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`github_com_matrixflow_moi-core_model_user.User`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| created_at | integer |  | Unix 时间戳（秒） |
| email | string |  | - |
| id | string |  | - |
| nickname | string |  | - |
| phone | string |  | - |
| status | user.UserStatus |  | - |
| updated_at | integer |  | Unix 时间戳（秒） |
| username | string |  | 用户名，用于数据库登录，必须唯一 |

响应示例:

```json
{
  "created_at": 0,
  "email": "string",
  "id": "string",
  "nickname": "string",
  "phone": "string",
  "status": {},
  "updated_at": 0,
  "username": "string"
}
```

---

### GET /api/v1/users/phone/{phone}

**根据手机号获取用户**

根据完整手机号精确获取用户信息

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| phone | string | 是 | 用户手机号（URL 编码） |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 用户详情 | `github_com_matrixflow_moi-core_model_user.User` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 404 | 用户不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`github_com_matrixflow_moi-core_model_user.User`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| created_at | integer |  | Unix 时间戳（秒） |
| email | string |  | - |
| id | string |  | - |
| nickname | string |  | - |
| phone | string |  | - |
| status | user.UserStatus |  | - |
| updated_at | integer |  | Unix 时间戳（秒） |
| username | string |  | 用户名，用于数据库登录，必须唯一 |

响应示例:

```json
{
  "created_at": 0,
  "email": "string",
  "id": "string",
  "nickname": "string",
  "phone": "string",
  "status": {},
  "updated_at": 0,
  "username": "string"
}
```

---

### GET /api/v1/users/{id}

**获取用户**

根据 ID 获取指定用户的详细信息

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | 用户 ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 用户详情 | `github_com_matrixflow_moi-core_model_user.User` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 404 | 用户不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`github_com_matrixflow_moi-core_model_user.User`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| created_at | integer |  | Unix 时间戳（秒） |
| email | string |  | - |
| id | string |  | - |
| nickname | string |  | - |
| phone | string |  | - |
| status | user.UserStatus |  | - |
| updated_at | integer |  | Unix 时间戳（秒） |
| username | string |  | 用户名，用于数据库登录，必须唯一 |

响应示例:

```json
{
  "created_at": 0,
  "email": "string",
  "id": "string",
  "nickname": "string",
  "phone": "string",
  "status": {},
  "updated_at": 0,
  "username": "string"
}
```

---

### PUT /api/v1/users/{id}

**Update user**

Update a user's nickname, phone, or status. Requires a system user.

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | User ID |

#### 请求体

类型: `user.UpdateUserRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string |  | - |
| nickname | string |  | - |
| phone | string |  | - |
| status | user.UserStatus |  | - |

示例:

```json
{
  "id": "string",
  "nickname": "string",
  "phone": "string",
  "status": {}
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | Updated user | `github_com_matrixflow_moi-core_model_user.User` |
| 400 | Invalid request | `gin.ErrorResponse` |
| 403 | Forbidden | `gin.ErrorResponse` |
| 404 | User not found | `gin.ErrorResponse` |
| 500 | Internal error | `gin.ErrorResponse` |

响应字段 (`github_com_matrixflow_moi-core_model_user.User`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| created_at | integer |  | Unix 时间戳（秒） |
| email | string |  | - |
| id | string |  | - |
| nickname | string |  | - |
| phone | string |  | - |
| status | user.UserStatus |  | - |
| updated_at | integer |  | Unix 时间戳（秒） |
| username | string |  | 用户名，用于数据库登录，必须唯一 |

响应示例:

```json
{
  "created_at": 0,
  "email": "string",
  "id": "string",
  "nickname": "string",
  "phone": "string",
  "status": {},
  "updated_at": 0,
  "username": "string"
}
```

---

### DELETE /api/v1/users/{id}

**删除用户**

删除指定用户及其关联资源；若用户仍拥有工作区则拒绝删除

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | 用户 ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 204 | 删除成功 |  |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 403 | 无权限（系统用户不可删除） | `gin.ErrorResponse` |
| 404 | 用户不存在 | `gin.ErrorResponse` |
| 409 | 用户仍拥有工作区 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

---

## Volume Content 管理

### GET /api/v1/workspaces/{id}/volumes/{volume_id}/contents

**列出 Volume 内容**

列出指定 Volume 下的子 Volume 和文件，支持过滤和分页

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| volume_id | integer | 是 | Volume ID |

#### 查询参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| include | string | 否 | 包含类型：volumes/files/all（默认 all） |
| filter_name | string | 否 | 名称过滤 |
| filter_type | string | 否 | 类型过滤 |
| order_by | string | 否 | 排序字段 |
| order | string | 否 | 排序方向：asc/desc |
| page_size | integer | 否 | 每页数量 |
| page_token | string | 否 | 分页令牌 |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 内容列表 | `handlers.ListContentsResponse` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 403 | 无权限：需要根 Volume 的 volume.read 权限 | `gin.ErrorResponse` |
| 404 | Volume 不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |
| 503 | IAM 授权服务或可信资源解析暂不可用 | `gin.ErrorResponse` |

响应字段 (`handlers.ListContentsResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| items | []tenant.VolumeContentItem |  | - |
| next_page_token | string |  | - |
| stats | tenant.VolumeContentStats |  | - |
| total | integer |  | - |
| volume |  |  | - |

响应示例:

```json
{
  "items": [{
    "created_at": 0,
    "created_by": "string",
    "file": {
      "created_at": 0,
      "created_by": "string",
      "file_id": "string",
      "file_name": "string",
      "file_path": "string",
      "id": 0,
      "revision": 0,
      "sandbox_path": "string",
      "updated_at": 0,
      "updated_by": "string",
      "volume_id": 0
    },
    "id": "string",
    "name": "string",
    "type": {},
    "updated_at": 0,
    "volume": {
      "catalog_id": 0,
      "comment": "string",
      "created_at": 0,
      "created_by": "string",
      "database_id": 0,
      "deleted": false,
      "deleted_at": 0,
      "display_bindings": [{
        "default_text": "string",
        "display_key": "string",
        "display_owner": "string",
        "field": "string"
      }],
      "id": 0,
      "name": "string",
      "parent_id": 0,
      "save_path": "string",
      "trigger_binding": "",
      "updated_at": 0,
      "updated_by": "string"
    }
  }],
  "next_page_token": "string",
  "stats": {
    "file_count": 0,
    "total_count": 0,
    "volume_count": 0
  },
  "total": 0,
  "volume": ""
}
```

---

## Volume File 管理

### GET /api/v1/workspaces/{id}/volumes/{volume_id}/files

**列出 Volume 中的文件**

列出指定 Volume 中的所有文件，支持分页

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| volume_id | integer | 是 | Volume ID |

#### 查询参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page_size | integer | 否 | 每页数量（默认 20，最大 100） |
| page_token | string | 否 | 分页令牌 |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 文件列表 | `handlers.ListVolumeFilesResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 404 | Volume 不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |
| 503 | IAM 授权服务未配置或暂不可用 | `gin.ErrorResponse` |

响应字段 (`handlers.ListVolumeFilesResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| items | []catalog.VolumeFile |  | - |
| next_page_token | string |  | - |
| total | integer |  | - |

响应示例:

```json
{
  "items": [{
    "created_at": 0,
    "created_by": "string",
    "file_id": "string",
    "file_name": "string",
    "file_path": "string",
    "id": 0,
    "revision": 0,
    "sandbox_path": "string",
    "updated_at": 0,
    "updated_by": "string",
    "volume_id": 0
  }],
  "next_page_token": "string",
  "total": 0
}
```

---

### POST /api/v1/workspaces/{id}/volumes/{volume_id}/files

**添加文件到 Volume**

将文件添加到指定 Volume 中，需要目标 Volume 的 volume.write 权限。require_unlinked 默认为 false；设为 true 时，文件若已属于其他 Volume 则返回 400，已属于当前目标 Volume 仍按幂等成功处理。该字段只约束文件关联状态，不校验上传者身份，也不是一次性上传凭证

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| volume_id | integer | 是 | Volume ID |

#### 请求体

类型: `handlers.AddFilesToVolumeRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| file_ids | []string |  | - |
| items | []handlers.AddFilesToVolumeItem |  | - |
| require_unlinked | boolean |  | RequireUnlinked rejects files associated with a Volume other than the target.
It constrains association state only; it does not prove uploader identity or possession. |

示例:

```json
{
  "file_ids": ["string"],
  "items": [{
    "file_id": "string",
    "file_name": "string",
    "file_path": "string"
  }],
  "require_unlinked": false
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 201 | 添加成功 |  |
| 400 | 参数错误，或 require_unlinked 文件已属于其他 Volume | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 403 | 无权限：需要目标 Volume 的 volume.write 权限 | `gin.ErrorResponse` |
| 404 | Volume 或文件不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |
| 503 | IAM 授权服务未配置或暂不可用 | `gin.ErrorResponse` |

---

### DELETE /api/v1/workspaces/{id}/volumes/{volume_id}/files

**从 Volume 移除文件**

从指定 Volume 中移除文件

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| volume_id | integer | 是 | Volume ID |

#### 请求体

类型: `handlers.RemoveFilesRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| file_ids | []string | 是 | - |

示例:

```json
{
  "file_ids": ["string"]
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 204 | 移除成功 |  |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 404 | Volume 或文件不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |
| 503 | IAM 授权服务未配置或暂不可用 | `gin.ErrorResponse` |

---

### GET /api/v1/workspaces/{id}/volumes/{volume_id}/files/detail

**列出 Volume 中的文件（含文件详情）**

列出指定 Volume 中的文件，连表查询 file 表返回完整文件元数据

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| volume_id | integer | 是 | Volume ID |

#### 查询参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page_size | integer | 否 | 每页数量（默认 20，最大 100） |
| page_token | string | 否 | 分页令牌 |
| order_by | string | 否 | 排序字段 |
| order | string | 否 | 排序方向 (asc/desc) |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 文件详情列表 | `handlers.ListVolumeFilesDetailResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 404 | Volume 不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |
| 503 | IAM 授权服务未配置或暂不可用 | `gin.ErrorResponse` |

响应字段 (`handlers.ListVolumeFilesDetailResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| items | []catalog.VolumeFileDetail |  | - |
| next_page_token | string |  | - |
| total | integer |  | - |

响应示例:

```json
{
  "items": [{
    "content_type": "string",
    "created_at": 0,
    "created_by": "string",
    "file_id": "string",
    "file_name": "string",
    "file_path": "string",
    "id": 0,
    "md5": "string",
    "original_name": "string",
    "ref_count": 0,
    "revision": 0,
    "sandbox_path": "string",
    "size": 0,
    "source": "string",
    "source_call_id": "string",
    "source_session_id": "string",
    "source_task_id": "string",
    "source_tool": "string",
    "updated_at": 0,
    "updated_by": "string",
    "volume_id": 0
  }],
  "next_page_token": "string",
  "total": 0
}
```

---

### POST /api/v1/workspaces/{id}/volumes/{volume_id}/files/move

**移动文件**

将文件从当前 Volume 移动到目标 Volume

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| volume_id | integer | 是 | 源 Volume ID |

#### 请求体

类型: `handlers.MoveFilesRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| file_ids | []string | 是 | - |
| target_volume_id | integer | 是 | - |

示例:

```json
{
  "file_ids": ["string"],
  "target_volume_id": 0
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 移动成功 |  |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 404 | Volume 或文件不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |
| 503 | IAM 授权服务未配置或暂不可用 | `gin.ErrorResponse` |

---

### POST /api/v1/workspaces/{id}/volumes/{volume_id}/files/trigger

**重新触发 Volume 文件工作流**

对已经在 Volume 中的文件重新创建 volume trigger delivery，并按 trigger 并发限制调度。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| volume_id | integer | 是 | Volume ID |

#### 请求体

类型: `handlers.TriggerVolumeFilesRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| file_ids | []string | 是 | - |

示例:

```json
{
  "file_ids": ["string"]
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 触发结果 | `handlers.TriggerVolumeFilesResponse` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 404 | Volume 或文件不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`handlers.TriggerVolumeFilesResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| triggered | integer |  | - |

响应示例:

```json
{
  "triggered": 0
}
```

---

## Volume 管理

### GET /api/v1/workspaces/{workspace_id}/databases/{database_id}/volumes

**列出 Volume**

列出指定 database 下的所有 Volume，支持分页

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| workspace_id | string | 是 | Workspace ID |
| database_id | integer | 是 | Database ID |

#### 查询参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page_size | integer | 否 | 每页数量（默认 20，最大 100） |
| page_token | string | 否 | 分页令牌 |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | Volume 列表 | `catalog.ListVolumesResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.ListVolumesResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| items | []catalog.Volume |  | - |
| next_page_token | string |  | - |
| total | integer |  | - |

响应示例:

```json
{
  "items": [{
    "catalog_id": 0,
    "comment": "string",
    "created_at": 0,
    "created_by": "string",
    "database_id": 0,
    "deleted": false,
    "deleted_at": 0,
    "display_bindings": [{
      "default_text": "string",
      "display_key": "string",
      "display_owner": "string",
      "field": "string"
    }],
    "id": 0,
    "name": "string",
    "parent_id": 0,
    "save_path": "string",
    "trigger_binding": "",
    "updated_at": 0,
    "updated_by": "string"
  }],
  "next_page_token": "string",
  "total": 0
}
```

---

### POST /api/v1/workspaces/{workspace_id}/databases/{database_id}/volumes

**创建 Volume**

在指定 database 下创建新的 Volume，支持通过 parent_id 创建子 Volume。名称必须为 1–255 个字符，首字符只能是小写英文字母、中文汉字、数字或下划线，后续还可使用连字符和点号；不自动修正输入

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| workspace_id | string | 是 | Workspace ID |
| database_id | integer | 是 | Database ID |

#### 请求体

类型: `catalog.CreateVolumeRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| catalog_id | integer |  | - |
| comment | string |  | - |
| database_id | integer |  | - |
| name | string |  | - |
| parent_id | integer |  | 父 Volume ID，为空表示创建根 Volume |
| save_path | string |  | - |

示例:

```json
{
  "catalog_id": 0,
  "comment": "string",
  "database_id": 0,
  "name": "string",
  "parent_id": 0,
  "save_path": "string"
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 201 | 创建成功 | `catalog.Volume` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 409 | Volume 名称已存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.Volume`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| catalog_id | integer |  | - |
| comment | string |  | - |
| created_at | integer |  | Unix 时间戳（秒） |
| created_by | string |  | 创建者用户 ID |
| database_id | integer |  | - |
| deleted | boolean |  | 软删除支持 - Requirements: 10.1 |
| deleted_at | integer |  | 删除时间（Unix时间戳，秒），仅当 deleted=true 时有值 |
| display_bindings | []catalog.ResourceDisplayBinding |  | - |
| id | integer |  | - |
| name | string |  | - |
| parent_id | integer |  | 层级结构支持 - Requirements: 9.1 |
| save_path | string |  | - |
| trigger_binding |  |  | 作为工作流触发器的绑定状态（只读） |
| updated_at | integer |  | Unix 时间戳（秒） |
| updated_by | string |  | 更新者用户 ID |

响应示例:

```json
{
  "catalog_id": 0,
  "comment": "string",
  "created_at": 0,
  "created_by": "string",
  "database_id": 0,
  "deleted": false,
  "deleted_at": 0,
  "display_bindings": [{
    "default_text": "string",
    "display_key": "string",
    "display_owner": "string",
    "field": "string"
  }],
  "id": 0,
  "name": "string",
  "parent_id": 0,
  "save_path": "string",
  "trigger_binding": "",
  "updated_at": 0,
  "updated_by": "string"
}
```

---

### GET /api/v1/workspaces/{workspace_id}/volumes/{id}

**获取 Volume**

根据 ID 获取指定 Volume 的详细信息

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| workspace_id | string | 是 | Workspace ID |
| id | integer | 是 | Volume ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | Volume 详情 | `catalog.Volume` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 404 | Volume 不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.Volume`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| catalog_id | integer |  | - |
| comment | string |  | - |
| created_at | integer |  | Unix 时间戳（秒） |
| created_by | string |  | 创建者用户 ID |
| database_id | integer |  | - |
| deleted | boolean |  | 软删除支持 - Requirements: 10.1 |
| deleted_at | integer |  | 删除时间（Unix时间戳，秒），仅当 deleted=true 时有值 |
| display_bindings | []catalog.ResourceDisplayBinding |  | - |
| id | integer |  | - |
| name | string |  | - |
| parent_id | integer |  | 层级结构支持 - Requirements: 9.1 |
| save_path | string |  | - |
| trigger_binding |  |  | 作为工作流触发器的绑定状态（只读） |
| updated_at | integer |  | Unix 时间戳（秒） |
| updated_by | string |  | 更新者用户 ID |

响应示例:

```json
{
  "catalog_id": 0,
  "comment": "string",
  "created_at": 0,
  "created_by": "string",
  "database_id": 0,
  "deleted": false,
  "deleted_at": 0,
  "display_bindings": [{
    "default_text": "string",
    "display_key": "string",
    "display_owner": "string",
    "field": "string"
  }],
  "id": 0,
  "name": "string",
  "parent_id": 0,
  "save_path": "string",
  "trigger_binding": "",
  "updated_at": 0,
  "updated_by": "string"
}
```

---

### PUT /api/v1/workspaces/{workspace_id}/volumes/{id}

**更新 Volume**

更新指定 Volume 的信息。显式提交新名称时使用与创建相同的名称规则；仅修改其他字段不会重新校验或改写历史名称

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| workspace_id | string | 是 | Workspace ID |
| id | integer | 是 | Volume ID |

#### 请求体

类型: `catalog.UpdateVolumeRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| comment | string |  | - |
| name | string |  | - |
| save_path | string |  | - |

示例:

```json
{
  "comment": "string",
  "name": "string",
  "save_path": "string"
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 更新后的 Volume | `catalog.Volume` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 404 | Volume 不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.Volume`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| catalog_id | integer |  | - |
| comment | string |  | - |
| created_at | integer |  | Unix 时间戳（秒） |
| created_by | string |  | 创建者用户 ID |
| database_id | integer |  | - |
| deleted | boolean |  | 软删除支持 - Requirements: 10.1 |
| deleted_at | integer |  | 删除时间（Unix时间戳，秒），仅当 deleted=true 时有值 |
| display_bindings | []catalog.ResourceDisplayBinding |  | - |
| id | integer |  | - |
| name | string |  | - |
| parent_id | integer |  | 层级结构支持 - Requirements: 9.1 |
| save_path | string |  | - |
| trigger_binding |  |  | 作为工作流触发器的绑定状态（只读） |
| updated_at | integer |  | Unix 时间戳（秒） |
| updated_by | string |  | 更新者用户 ID |

响应示例:

```json
{
  "catalog_id": 0,
  "comment": "string",
  "created_at": 0,
  "created_by": "string",
  "database_id": 0,
  "deleted": false,
  "deleted_at": 0,
  "display_bindings": [{
    "default_text": "string",
    "display_key": "string",
    "display_owner": "string",
    "field": "string"
  }],
  "id": 0,
  "name": "string",
  "parent_id": 0,
  "save_path": "string",
  "trigger_binding": "",
  "updated_at": 0,
  "updated_by": "string"
}
```

---

### DELETE /api/v1/workspaces/{workspace_id}/volumes/{id}

**删除 Volume**

删除指定 Volume

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| workspace_id | string | 是 | Workspace ID |
| id | integer | 是 | Volume ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 204 | 删除成功 |  |
| 401 | 未认证 | `gin.ErrorResponse` |
| 404 | Volume 不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

---

### GET /api/v1/workspaces/{workspace_id}/volumes/{volume_id}/children

**获取子 Volume**

获取指定 Volume 下的子 Volume 列表，支持分页

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| workspace_id | string | 是 | Workspace ID |
| volume_id | integer | 是 | 父 Volume ID |

#### 查询参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page_size | integer | 否 | 每页数量（默认 20，最大 100） |
| page_token | string | 否 | 分页令牌 |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 子 Volume 列表 | `catalog.ListVolumesResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 404 | Volume 不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`catalog.ListVolumesResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| items | []catalog.Volume |  | - |
| next_page_token | string |  | - |
| total | integer |  | - |

响应示例:

```json
{
  "items": [{
    "catalog_id": 0,
    "comment": "string",
    "created_at": 0,
    "created_by": "string",
    "database_id": 0,
    "deleted": false,
    "deleted_at": 0,
    "display_bindings": [{
      "default_text": "string",
      "display_key": "string",
      "display_owner": "string",
      "field": "string"
    }],
    "id": 0,
    "name": "string",
    "parent_id": 0,
    "save_path": "string",
    "trigger_binding": "",
    "updated_at": 0,
    "updated_by": "string"
  }],
  "next_page_token": "string",
  "total": 0
}
```

---

### GET /api/v1/workspaces/{workspace_id}/volumes/{volume_id}/path

**获取 Volume 路径**

获取从根到指定 Volume 的完整路径

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| workspace_id | string | 是 | Workspace ID |
| volume_id | integer | 是 | Volume ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | Volume 路径列表 | `[]catalog.Volume` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 404 | Volume 不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

---

## WorkItem 管理

### GET /api/v1/workspaces/{id}/workitems

**列出 WorkItem**

列出指定 workspace 中的所有可用 WorkItem

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | WorkItem 列表 | `object` |
| 400 | 参数错误 | `object` |
| 401 | 未认证 | `object` |
| 500 | 内部错误 | `object` |

---

### GET /api/v1/workspaces/{id}/workitems/catalog

**列出 WorkItem 目录**

列出指定 workspace 中的所有可用 WorkItem（前端友好格式）

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 查询参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| locale | string | 否 | Language locale (e.g. LANGUAGE_ZH) |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | WorkItem 目录列表 | `object` |
| 400 | 参数错误 | `object` |
| 401 | 未认证 | `object` |
| 500 | 内部错误 | `object` |

---

### GET /api/v1/workspaces/{id}/workitems/catalog/{node_id}

**获取单个 WorkItem 目录条目**

获取指定 workspace 中指定 node_id 的 WorkItem 详情

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| node_id | string | 是 | WorkItem Node ID |

#### 查询参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| locale | string | 否 | Language locale (e.g. LANGUAGE_ZH) |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | WorkItem 详情 | `object` |
| 400 | 参数错误 | `object` |
| 401 | 未认证 | `object` |
| 404 | 未找到 | `object` |
| 500 | 内部错误 | `object` |

---

## Workbook 管理

### GET /api/v1/workspaces/{id}/workbooks

**列出工作簿**

按 IAM 权限过滤后分页返回 workspace 下的工作簿列表。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 查询参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page_size | integer | 否 | 分页大小 |
| page_token | string | 否 | 分页 token |
| search | string | 否 | 名称模糊搜索 |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 查询成功 | `handlers.workbookListResponse` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`handlers.workbookListResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| items | []handlers.workbookResponse |  | - |
| next_page_token | string |  | - |
| total | integer |  | - |

响应示例:

```json
{
  "items": [{
    "created_at": "string",
    "id": "string",
    "name": "string",
    "uid": "string",
    "updated_at": "string"
  }],
  "next_page_token": "string",
  "total": 0
}
```

---

### POST /api/v1/workspaces/{id}/workbooks

**创建工作簿**

在 workspace 内创建新的 SQL 编辑器工作簿，同时注册 IAM ownership 记录。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 请求体

类型: `handlers.workbookCreateRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string |  | - |

示例:

```json
{
  "name": "string"
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 201 | 创建成功 | `handlers.workbookResponse` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`handlers.workbookResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| created_at | string |  | - |
| id | string |  | - |
| name | string |  | - |
| uid | string |  | - |
| updated_at | string |  | - |

响应示例:

```json
{
  "created_at": "string",
  "id": "string",
  "name": "string",
  "uid": "string",
  "updated_at": "string"
}
```

---

### GET /api/v1/workspaces/{id}/workbooks/{workbook_id}

**获取工作簿**

获取指定工作簿元数据（需有读权限）。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| workbook_id | string | 是 | Workbook ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 查询成功 | `handlers.workbookResponse` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 404 | 未找到 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`handlers.workbookResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| created_at | string |  | - |
| id | string |  | - |
| name | string |  | - |
| uid | string |  | - |
| updated_at | string |  | - |

响应示例:

```json
{
  "created_at": "string",
  "id": "string",
  "name": "string",
  "uid": "string",
  "updated_at": "string"
}
```

---

### PUT /api/v1/workspaces/{id}/workbooks/{workbook_id}

**重命名工作簿**

更新工作簿名称。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| workbook_id | string | 是 | Workbook ID |

#### 请求体

类型: `handlers.workbookUpdateRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string |  | - |

示例:

```json
{
  "name": "string"
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 更新成功 | `object` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 404 | 未找到 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

---

### DELETE /api/v1/workspaces/{id}/workbooks/{workbook_id}

**删除工作簿**

删除工作簿及其版本，同时完成 IAM ownership 清理。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| workbook_id | string | 是 | Workbook ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 删除成功 | `object` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 404 | 未找到 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

---

### GET /api/v1/workspaces/{id}/workbooks/{workbook_id}/versions

**列出工作簿版本**

分页列出指定工作簿的所有版本。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| workbook_id | string | 是 | Workbook ID |

#### 查询参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page_size | integer | 否 | 分页大小 |
| page_token | string | 否 | 分页 token |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 查询成功 | `handlers.workbookVersionListResponse` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 404 | 未找到 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`handlers.workbookVersionListResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| items | []handlers.workbookVersionResponse |  | - |
| next_page_token | string |  | - |
| total | integer |  | - |

响应示例:

```json
{
  "items": [{
    "created_at": "string",
    "id": "string",
    "sql_content": "string",
    "status": 0,
    "updated_at": "string",
    "version": "string",
    "workbook_id": "string"
  }],
  "next_page_token": "string",
  "total": 0
}
```

---

### POST /api/v1/workspaces/{id}/workbooks/{workbook_id}/versions

**创建工作簿版本**

为工作簿创建新的 draft 版本。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| workbook_id | string | 是 | Workbook ID |

#### 请求体

类型: `handlers.workbookVersionCreateRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| sql_content | string |  | - |
| version | string |  | - |

示例:

```json
{
  "sql_content": "string",
  "version": "string"
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 201 | 创建成功 | `handlers.workbookVersionResponse` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 404 | 未找到 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`handlers.workbookVersionResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| created_at | string |  | - |
| id | string |  | - |
| sql_content | string |  | - |
| status | integer |  | 1=draft, 2=finalize |
| updated_at | string |  | - |
| version | string |  | - |
| workbook_id | string |  | - |

响应示例:

```json
{
  "created_at": "string",
  "id": "string",
  "sql_content": "string",
  "status": 0,
  "updated_at": "string",
  "version": "string",
  "workbook_id": "string"
}
```

---

### GET /api/v1/workspaces/{id}/workbooks/{workbook_id}/versions/{version_id}

**获取工作簿版本详情**

获取指定版本（或通过 ?version= 指定 draft）的内容与元数据。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| workbook_id | string | 是 | Workbook ID |
| version_id | string | 是 | Version ID（或使用 ?version= 查 draft） |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 查询成功 | `handlers.workbookDetailResponse` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 404 | 未找到 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`handlers.workbookDetailResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| version | handlers.workbookVersionResponse |  | - |
| workbook | handlers.workbookResponse |  | - |

响应示例:

```json
{
  "version": {
    "created_at": "string",
    "id": "string",
    "sql_content": "string",
    "status": 0,
    "updated_at": "string",
    "version": "string",
    "workbook_id": "string"
  },
  "workbook": {
    "created_at": "string",
    "id": "string",
    "name": "string",
    "uid": "string",
    "updated_at": "string"
  }
}
```

---

### PUT /api/v1/workspaces/{id}/workbooks/{workbook_id}/versions/{version_id}

**更新工作簿版本内容**

更新 draft 版本的 SQL 内容。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| workbook_id | string | 是 | Workbook ID |
| version_id | string | 是 | Version ID |

#### 请求体

类型: `handlers.workbookVersionUpdateRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| sql_content | string |  | - |

示例:

```json
{
  "sql_content": "string"
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 更新成功 | `object` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 404 | 未找到 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

---

### POST /api/v1/workspaces/{id}/workbooks/{workbook_id}/versions/{version_id}/finalize

**终结工作簿版本**

将 draft 版本终结为正式版本（可选更新内容后终结）。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| workbook_id | string | 是 | Workbook ID |
| version_id | string | 是 | Version ID |

#### 请求体

类型: `handlers.workbookVersionUpdateRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| sql_content | string |  | - |

示例:

```json
{
  "sql_content": "string"
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 终结成功 | `object` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 404 | 未找到 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

---

## Workflow App

### POST /api/v1/workspaces/{id}/system-workflow-apps/{kind}

**确保系统 Workflow App**

创建或更新内置系统 workflow app，并返回已发布版本

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| kind | string | 是 | System workflow kind |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | System workflow ref | `workflowapp.SystemWorkflowRef` |
| 400 | 参数错误 | `object` |
| 401 | 未认证 | `object` |

响应字段 (`workflowapp.SystemWorkflowRef`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| mowl_workflow_def_id | string |  | - |
| mowl_workflow_version_id | string |  | - |
| pipeline_graph_json | string |  | - |
| workflow_app_id | string |  | - |

响应示例:

```json
{
  "mowl_workflow_def_id": "string",
  "mowl_workflow_version_id": "string",
  "pipeline_graph_json": "string",
  "workflow_app_id": "string"
}
```

---

### GET /api/v1/workspaces/{id}/workflow-apps

**列出 Workflow Apps**

查询系统库中的产品层 workflow 列表

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 查询参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| offset | integer | 否 | Offset |
| limit | integer | 否 | Limit |
| source_type | string | 否 | Source type |
| status | string | 否 | Workflow status |
| execution_mode | string | 否 | Execution mode |
| name_search | string | 否 | Name search |
| include_dynamic_service | boolean | 否 | Include dynamic service workflows |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | Workflow list | `workflowapp.WorkflowListResponse` |
| 401 | 未认证 | `object` |

响应字段 (`workflowapp.WorkflowListResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| total | integer |  | - |
| workflows | []workflowapp.WorkflowSummary |  | - |

响应示例:

```json
{
  "total": 0,
  "workflows": [{
    "candidate_id": "string",
    "compute_resource_bindings": [{
      "id": "string",
      "name": "string",
      "node_names": ["string"],
      "workflow_level": false
    }],
    "compute_resource_id": "string",
    "created_at": "string",
    "cron_expression": "string",
    "description": "string",
    "draft_id": "string",
    "execution_mode": "string",
    "execution_summary": {
      "active_execution_id": "string",
      "active_execution_status": "string",
      "active_executions": 0,
      "latest_execution_at": "string",
      "latest_execution_id": "string",
      "latest_execution_status": "string",
      "total_executions": 0
    },
    "id": "string",
    "latest_version": 0,
    "latest_version_status": "string",
    "latest_workflow_version_id": "string",
    "moi_workflow_def_id": "string",
    "name": "string",
    "parameter_summary": {
      "filled_required_fields": 0,
      "missing_required_field_ids": ["string"],
      "missing_required_field_labels": ["string"],
      "missing_required_fields": 0,
      "required_fields": 0,
      "status": "string",
      "total_fields": 0
    },
    "source_type": "string",
    "status": "string",
    "trigger_summary": {
      "configured": false,
      "cron_expression": "string",
      "enabled": false,
      "mode": "string",
      "service_name": "string",
      "volume_id": 0
    },
    "updated_at": "string"
  }]
}
```

---

### GET /api/v1/workspaces/{id}/workflow-apps/{workflow_id}

**获取 Workflow App**

查询一个产品层 workflow 详情

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| workflow_id | string | 是 | Workflow ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | Workflow detail | `workflowapp.WorkflowEnvelope` |
| 401 | 未认证 | `object` |
| 404 | 资源不存在 | `object` |

响应字段 (`workflowapp.WorkflowEnvelope`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| warnings | []string |  | - |
| workflow | workflowapp.WorkflowDetail |  | - |

响应示例:

```json
{
  "warnings": ["string"],
  "workflow": {
    "candidate_id": "string",
    "compute_resource_bindings": [{
      "id": "string",
      "name": "string",
      "node_names": ["string"],
      "workflow_level": false
    }],
    "compute_resource_id": "string",
    "created_at": "string",
    "cron_expression": "string",
    "default_values": {},
    "default_values_json": "string",
    "deployment_config_json": "string",
    "description": "string",
    "design_graph_json": "string",
    "draft_id": "string",
    "dsl_yaml": "string",
    "execution_mode": "string",
    "execution_summary": {
      "active_execution_id": "string",
      "active_execution_status": "string",
      "active_executions": 0,
      "latest_execution_at": "string",
      "latest_execution_id": "string",
      "latest_execution_status": "string",
      "total_executions": 0
    },
    "goal": "string",
    "id": "string",
    "latest_version": 0,
    "latest_version_status": "string",
    "latest_workflow_version_id": "string",
    "moi_workflow_def_id": "string",
    "name": "string",
    "parameter_summary": {
      "filled_required_fields": 0,
      "missing_required_field_ids": ["string"],
      "missing_required_field_labels": ["string"],
      "missing_required_fields": 0,
      "required_fields": 0,
      "status": "string",
      "total_fields": 0
    },
    "planner_model": "string",
    "run_context_json": "string",
    "runtime_fields_json": "string",
    "runtime_layout_json": "string",
    "session_id": "string",
    "source_type": "string",
    "status": "string",
    "trigger_summary": {
      "configured": false,
      "cron_expression": "string",
      "enabled": false,
      "mode": "string",
      "service_name": "string",
      "volume_id": 0
    },
    "updated_at": "string"
  }
}
```

---

### PATCH /api/v1/workspaces/{id}/workflow-apps/{workflow_id}

**更新 Workflow App**

更新 workflow app 元数据

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| workflow_id | string | 是 | Workflow ID |

#### 请求体

类型: `workflowapp.UpdateWorkflowRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| compute_resource_id | string |  | - |
| default_values_json | string |  | - |
| description | string |  | - |
| name | string |  | - |
| session_id | string |  | SessionID binds the workflow Copilot conversation (A2A context id). |
| status | string |  | - |

示例:

```json
{
  "compute_resource_id": "string",
  "default_values_json": "string",
  "description": "string",
  "name": "string",
  "session_id": "string",
  "status": "string"
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | Workflow detail | `workflowapp.WorkflowEnvelope` |
| 400 | 参数错误 | `object` |
| 401 | 未认证 | `object` |
| 404 | 资源不存在 | `object` |

响应字段 (`workflowapp.WorkflowEnvelope`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| warnings | []string |  | - |
| workflow | workflowapp.WorkflowDetail |  | - |

响应示例:

```json
{
  "warnings": ["string"],
  "workflow": {
    "candidate_id": "string",
    "compute_resource_bindings": [{
      "id": "string",
      "name": "string",
      "node_names": ["string"],
      "workflow_level": false
    }],
    "compute_resource_id": "string",
    "created_at": "string",
    "cron_expression": "string",
    "default_values": {},
    "default_values_json": "string",
    "deployment_config_json": "string",
    "description": "string",
    "design_graph_json": "string",
    "draft_id": "string",
    "dsl_yaml": "string",
    "execution_mode": "string",
    "execution_summary": {
      "active_execution_id": "string",
      "active_execution_status": "string",
      "active_executions": 0,
      "latest_execution_at": "string",
      "latest_execution_id": "string",
      "latest_execution_status": "string",
      "total_executions": 0
    },
    "goal": "string",
    "id": "string",
    "latest_version": 0,
    "latest_version_status": "string",
    "latest_workflow_version_id": "string",
    "moi_workflow_def_id": "string",
    "name": "string",
    "parameter_summary": {
      "filled_required_fields": 0,
      "missing_required_field_ids": ["string"],
      "missing_required_field_labels": ["string"],
      "missing_required_fields": 0,
      "required_fields": 0,
      "status": "string",
      "total_fields": 0
    },
    "planner_model": "string",
    "run_context_json": "string",
    "runtime_fields_json": "string",
    "runtime_layout_json": "string",
    "session_id": "string",
    "source_type": "string",
    "status": "string",
    "trigger_summary": {
      "configured": false,
      "cron_expression": "string",
      "enabled": false,
      "mode": "string",
      "service_name": "string",
      "volume_id": 0
    },
    "updated_at": "string"
  }
}
```

---

### DELETE /api/v1/workspaces/{id}/workflow-apps/{workflow_id}

**删除 Workflow App**

当 workflow 没有未完成 execution 时，删除 workflow app 及其触发器、cron、执行记录和 mowl workflow 版本；存在 preparing/submitted/triggered/scheduled/running/paused execution 时返回 409

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| workflow_id | string | 是 | Workflow ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | Delete result | `workflowapp.DeleteResponse` |
| 401 | 未认证 | `object` |
| 404 | 资源不存在 | `object` |
| 409 | 存在未完成 execution | `object` |

响应字段 (`workflowapp.DeleteResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| deleted | boolean |  | - |
| disabled_cron_task_ids | []string |  | - |
| warnings | []string |  | - |

响应示例:

```json
{
  "deleted": false,
  "disabled_cron_task_ids": ["string"],
  "warnings": ["string"]
}
```

---

### POST /api/v1/workspaces/{id}/workflow-apps/{workflow_id}/pause

**暂停 Workflow App**

暂停 workflow app，并在安全调度边界暂停该 workflow 下正在运行的 executions

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| workflow_id | string | 是 | Workflow ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | Workflow detail | `workflowapp.WorkflowEnvelope` |
| 400 | 参数错误 | `object` |
| 401 | 未认证 | `object` |
| 404 | 资源不存在 | `object` |

响应字段 (`workflowapp.WorkflowEnvelope`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| warnings | []string |  | - |
| workflow | workflowapp.WorkflowDetail |  | - |

响应示例:

```json
{
  "warnings": ["string"],
  "workflow": {
    "candidate_id": "string",
    "compute_resource_bindings": [{
      "id": "string",
      "name": "string",
      "node_names": ["string"],
      "workflow_level": false
    }],
    "compute_resource_id": "string",
    "created_at": "string",
    "cron_expression": "string",
    "default_values": {},
    "default_values_json": "string",
    "deployment_config_json": "string",
    "description": "string",
    "design_graph_json": "string",
    "draft_id": "string",
    "dsl_yaml": "string",
    "execution_mode": "string",
    "execution_summary": {
      "active_execution_id": "string",
      "active_execution_status": "string",
      "active_executions": 0,
      "latest_execution_at": "string",
      "latest_execution_id": "string",
      "latest_execution_status": "string",
      "total_executions": 0
    },
    "goal": "string",
    "id": "string",
    "latest_version": 0,
    "latest_version_status": "string",
    "latest_workflow_version_id": "string",
    "moi_workflow_def_id": "string",
    "name": "string",
    "parameter_summary": {
      "filled_required_fields": 0,
      "missing_required_field_ids": ["string"],
      "missing_required_field_labels": ["string"],
      "missing_required_fields": 0,
      "required_fields": 0,
      "status": "string",
      "total_fields": 0
    },
    "planner_model": "string",
    "run_context_json": "string",
    "runtime_fields_json": "string",
    "runtime_layout_json": "string",
    "session_id": "string",
    "source_type": "string",
    "status": "string",
    "trigger_summary": {
      "configured": false,
      "cron_expression": "string",
      "enabled": false,
      "mode": "string",
      "service_name": "string",
      "volume_id": 0
    },
    "updated_at": "string"
  }
}
```

---

### POST /api/v1/workspaces/{id}/workflow-apps/{workflow_id}/resume

**恢复 Workflow App**

恢复 workflow app，并恢复由 workflow 级暂停影响的 executions

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| workflow_id | string | 是 | Workflow ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | Workflow detail | `workflowapp.WorkflowEnvelope` |
| 400 | 参数错误 | `object` |
| 401 | 未认证 | `object` |
| 404 | 资源不存在 | `object` |

响应字段 (`workflowapp.WorkflowEnvelope`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| warnings | []string |  | - |
| workflow | workflowapp.WorkflowDetail |  | - |

响应示例:

```json
{
  "warnings": ["string"],
  "workflow": {
    "candidate_id": "string",
    "compute_resource_bindings": [{
      "id": "string",
      "name": "string",
      "node_names": ["string"],
      "workflow_level": false
    }],
    "compute_resource_id": "string",
    "created_at": "string",
    "cron_expression": "string",
    "default_values": {},
    "default_values_json": "string",
    "deployment_config_json": "string",
    "description": "string",
    "design_graph_json": "string",
    "draft_id": "string",
    "dsl_yaml": "string",
    "execution_mode": "string",
    "execution_summary": {
      "active_execution_id": "string",
      "active_execution_status": "string",
      "active_executions": 0,
      "latest_execution_at": "string",
      "latest_execution_id": "string",
      "latest_execution_status": "string",
      "total_executions": 0
    },
    "goal": "string",
    "id": "string",
    "latest_version": 0,
    "latest_version_status": "string",
    "latest_workflow_version_id": "string",
    "moi_workflow_def_id": "string",
    "name": "string",
    "parameter_summary": {
      "filled_required_fields": 0,
      "missing_required_field_ids": ["string"],
      "missing_required_field_labels": ["string"],
      "missing_required_fields": 0,
      "required_fields": 0,
      "status": "string",
      "total_fields": 0
    },
    "planner_model": "string",
    "run_context_json": "string",
    "runtime_fields_json": "string",
    "runtime_layout_json": "string",
    "session_id": "string",
    "source_type": "string",
    "status": "string",
    "trigger_summary": {
      "configured": false,
      "cron_expression": "string",
      "enabled": false,
      "mode": "string",
      "service_name": "string",
      "volume_id": 0
    },
    "updated_at": "string"
  }
}
```

---

### POST /api/v1/workspaces/{id}/workflow-apps/{workflow_id}/validate-delete

**校验 Workflow App 是否可删除**

使用与删除 Workflow App 相同的阻塞条件做非破坏性校验；存在未完成 execution、open volume delivery 或 open volume dispatch job 时返回 409

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| workflow_id | string | 是 | Workflow ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | Validate delete result | `workflowapp.ValidateDeleteResponse` |
| 401 | 未认证 | `object` |
| 404 | 资源不存在 | `object` |
| 409 | 存在未完成执行或未完成 Volume 触发任务 | `object` |

响应字段 (`workflowapp.ValidateDeleteResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| deletable | boolean |  | - |

响应示例:

```json
{
  "deletable": false
}
```

---

## Workflow Deployment

### POST /api/v1/workspaces/{id}/workflow-deployments

**发布 Workflow**

在系统库事务内发布 workflow app、mowl version 以及对应执行资源

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 请求体

类型: `workflowdeployment.DeployRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| candidate_id | string |  | - |
| compute_resource_id | string |  | - |
| cron | workflowdeployment.CronSpec |  | - |
| data_json | string |  | - |
| default_values_json | string |  | - |
| deployment_config_json | string |  | - |
| description | string |  | - |
| design_graph_json | string |  | - |
| draft_id | string |  | - |
| dsl_yaml | string |  | - |
| dynamic_service | workflowdeployment.DynamicServiceSpec |  | - |
| execution_mode | string |  | - |
| goal | string |  | - |
| name | string |  | - |
| planner_model | string |  | - |
| run_context_json | string |  | - |
| runtime_fields_json | string |  | - |
| runtime_layout_json | string |  | - |
| session_id | string |  | - |
| source_type | string |  | - |
| status | string |  | - |
| vars_json | string |  | - |
| volume_trigger | workflowdeployment.VolumeTriggerSpec |  | - |
| workflow_id | string |  | Product-layer workflow metadata. When WorkflowID is provided the deployment updates
that workflow_app row; otherwise Deploy creates a new product workflow row. |

示例:

```json
{
  "candidate_id": "string",
  "compute_resource_id": "string",
  "cron": {
    "cron_expression": "string",
    "data_json": "string",
    "task_name": "string",
    "vars_json": "string"
  },
  "data_json": "string",
  "default_values_json": "string",
  "deployment_config_json": "string",
  "description": "string",
  "design_graph_json": "string",
  "draft_id": "string",
  "dsl_yaml": "string",
  "dynamic_service": {
    "input_schema": "string",
    "output_schema": "string",
    "result_mode": "string",
    "runtime_spec_json": "string",
    "service_name": "string"
  },
  "execution_mode": "string",
  "goal": "string",
  "name": "string",
  "planner_model": "string",
  "run_context_json": "string",
  "runtime_fields_json": "string",
  "runtime_layout_json": "string",
  "session_id": "string",
  "source_type": "string",
  "status": "string",
  "vars_json": "string",
  "volume_trigger": {
    "auto_dispatch_enabled": false,
    "enabled": false,
    "max_concurrency": 0,
    "vars_json": "string",
    "volume_id": 0
  },
  "workflow_id": "string"
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 201 | Deployment result | `object` |
| 400 | 参数错误 | `object` |
| 401 | 未认证 | `object` |
| 503 | 服务不可用 | `object` |

---

## Workflow Execution

### POST /api/v1/workspaces/{id}/system-workflow-apps/{kind}/executions

**执行系统 Workflow App**

通过 workflow-app 统一入口执行内置系统 workflow

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| kind | string | 是 | System workflow kind |

#### 请求体

类型: `workflowapp.SystemWorkflowExecutionRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| compute_resource_id | string |  | - |
| cron_expression | string |  | - |
| data_json | string |  | - |
| execution_mode | string |  | - |
| idempotency_key | string |  | - |
| task_id | string |  | - |
| task_name | string |  | - |
| transient | boolean |  | - |
| trigger_now | boolean |  | - |
| vars_json | string |  | - |

示例:

```json
{
  "compute_resource_id": "string",
  "cron_expression": "string",
  "data_json": "string",
  "execution_mode": "string",
  "idempotency_key": "string",
  "task_id": "string",
  "task_name": "string",
  "transient": false,
  "trigger_now": false,
  "vars_json": "string"
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | System execution result | `workflowapp.SystemWorkflowExecutionEnvelope` |
| 400 | 参数错误 | `object` |
| 401 | 未认证 | `object` |

响应字段 (`workflowapp.SystemWorkflowExecutionEnvelope`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| error | string |  | - |
| moi_case_id | string |  | - |
| moi_task_id | string |  | - |
| mowl_workflow_def_id | string |  | - |
| mowl_workflow_version_id | string |  | - |
| status | string |  | - |
| workflow_app_id | string |  | - |
| workflow_execution_id | string |  | - |

响应示例:

```json
{
  "error": "string",
  "moi_case_id": "string",
  "moi_task_id": "string",
  "mowl_workflow_def_id": "string",
  "mowl_workflow_version_id": "string",
  "status": "string",
  "workflow_app_id": "string",
  "workflow_execution_id": "string"
}
```

---

### POST /api/v1/workspaces/{id}/system-workflow-apps/{kind}/executions/ensure

**Ensure one system Workflow App execution**

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| kind | string | 是 | System workflow kind |

#### 请求体

类型: `workflowapp.SystemWorkflowExecutionRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| compute_resource_id | string |  | - |
| cron_expression | string |  | - |
| data_json | string |  | - |
| execution_mode | string |  | - |
| idempotency_key | string |  | - |
| task_id | string |  | - |
| task_name | string |  | - |
| transient | boolean |  | - |
| trigger_now | boolean |  | - |
| vars_json | string |  | - |

示例:

```json
{
  "compute_resource_id": "string",
  "cron_expression": "string",
  "data_json": "string",
  "execution_mode": "string",
  "idempotency_key": "string",
  "task_id": "string",
  "task_name": "string",
  "transient": false,
  "trigger_now": false,
  "vars_json": "string"
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | System execution result | `workflowapp.SystemWorkflowExecutionEnvelope` |
| 400 | Invalid request | `object` |
| 401 | Unauthenticated | `object` |

响应字段 (`workflowapp.SystemWorkflowExecutionEnvelope`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| error | string |  | - |
| moi_case_id | string |  | - |
| moi_task_id | string |  | - |
| mowl_workflow_def_id | string |  | - |
| mowl_workflow_version_id | string |  | - |
| status | string |  | - |
| workflow_app_id | string |  | - |
| workflow_execution_id | string |  | - |

响应示例:

```json
{
  "error": "string",
  "moi_case_id": "string",
  "moi_task_id": "string",
  "mowl_workflow_def_id": "string",
  "mowl_workflow_version_id": "string",
  "status": "string",
  "workflow_app_id": "string",
  "workflow_execution_id": "string"
}
```

---

### GET /api/v1/workspaces/{id}/workflow-apps/executions

**列出 Workflow Executions**

查询 workflow app 的执行记录

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 查询参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| offset | integer | 否 | Offset |
| limit | integer | 否 | Limit |
| status | string | 否 | Execution status |
| execution_mode | string | 否 | Execution mode |
| workflow_name | string | 否 | Workflow name |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | Execution list | `workflowapp.ExecutionListResponse` |
| 401 | 未认证 | `object` |

响应字段 (`workflowapp.ExecutionListResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| executions | []workflowapp.ExecutionDetail |  | - |
| total | integer |  | - |

响应示例:

```json
{
  "executions": [{
    "case_error": "string",
    "case_result": "string",
    "created_at": "string",
    "cron_expression": "string",
    "data_name": "string",
    "dispatch_job_id": "string",
    "ended_at": "string",
    "error": "string",
    "execution_id": "string",
    "execution_mode": "string",
    "failure": {
      "case_id": "string",
      "code": "string",
      "details": {},
      "message": "string",
      "node_id": "string",
      "node_name": "string",
      "parallel_index": 0,
      "raw_error": "string",
      "span_id": "string",
      "span_kind": "string",
      "stage": "string",
      "task_id": "string",
      "trace_id": "string",
      "worker_id": "string",
      "workitem_id": "string"
    },
    "input_payload": {},
    "input_payload_json": "string",
    "moi_case_id": "string",
    "moi_task_id": "string",
    "moi_workflow_def_id": "string",
    "moi_workflow_version_id": "string",
    "pause_scope": "string",
    "started_at": "string",
    "status": "string",
    "updated_at": "string",
    "vars_payload": {},
    "vars_payload_json": "string",
    "workflow_id": "string",
    "workflow_name": "string"
  }],
  "total": 0
}
```

---

### GET /api/v1/workspaces/{id}/workflow-apps/executions/{execution_id}

**获取 Workflow Execution**

查询一次 workflow 执行详情

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| execution_id | string | 是 | Execution ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | Execution detail | `workflowapp.ExecutionEnvelope` |
| 401 | 未认证 | `object` |
| 404 | 资源不存在 | `object` |

响应字段 (`workflowapp.ExecutionEnvelope`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| execution | workflowapp.ExecutionDetail |  | - |

响应示例:

```json
{
  "execution": {
    "case_error": "string",
    "case_result": "string",
    "created_at": "string",
    "cron_expression": "string",
    "data_name": "string",
    "dispatch_job_id": "string",
    "ended_at": "string",
    "error": "string",
    "execution_id": "string",
    "execution_mode": "string",
    "failure": {
      "case_id": "string",
      "code": "string",
      "details": {},
      "message": "string",
      "node_id": "string",
      "node_name": "string",
      "parallel_index": 0,
      "raw_error": "string",
      "span_id": "string",
      "span_kind": "string",
      "stage": "string",
      "task_id": "string",
      "trace_id": "string",
      "worker_id": "string",
      "workitem_id": "string"
    },
    "input_payload": {},
    "input_payload_json": "string",
    "moi_case_id": "string",
    "moi_task_id": "string",
    "moi_workflow_def_id": "string",
    "moi_workflow_version_id": "string",
    "pause_scope": "string",
    "started_at": "string",
    "status": "string",
    "updated_at": "string",
    "vars_payload": {},
    "vars_payload_json": "string",
    "workflow_id": "string",
    "workflow_name": "string"
  }
}
```

---

### GET /api/v1/workspaces/{id}/workflow-apps/executions/{execution_id}/result

**获取 Workflow Execution 结果**

刷新并返回一次 workflow 执行结果

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| execution_id | string | 是 | Execution ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | Execution result | `workflowapp.ExecutionEnvelope` |
| 401 | 未认证 | `object` |
| 404 | 资源不存在 | `object` |

响应字段 (`workflowapp.ExecutionEnvelope`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| execution | workflowapp.ExecutionDetail |  | - |

响应示例:

```json
{
  "execution": {
    "case_error": "string",
    "case_result": "string",
    "created_at": "string",
    "cron_expression": "string",
    "data_name": "string",
    "dispatch_job_id": "string",
    "ended_at": "string",
    "error": "string",
    "execution_id": "string",
    "execution_mode": "string",
    "failure": {
      "case_id": "string",
      "code": "string",
      "details": {},
      "message": "string",
      "node_id": "string",
      "node_name": "string",
      "parallel_index": 0,
      "raw_error": "string",
      "span_id": "string",
      "span_kind": "string",
      "stage": "string",
      "task_id": "string",
      "trace_id": "string",
      "worker_id": "string",
      "workitem_id": "string"
    },
    "input_payload": {},
    "input_payload_json": "string",
    "moi_case_id": "string",
    "moi_task_id": "string",
    "moi_workflow_def_id": "string",
    "moi_workflow_version_id": "string",
    "pause_scope": "string",
    "started_at": "string",
    "status": "string",
    "updated_at": "string",
    "vars_payload": {},
    "vars_payload_json": "string",
    "workflow_id": "string",
    "workflow_name": "string"
  }
}
```

---

### GET /api/v1/workspaces/{id}/workflow-apps/file-executions

**列出文件相关 Workflow Executions**

查询与指定文件关联的 workflow 执行记录

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 查询参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| file_id | string | 是 | File ID |
| semantic_model_id | integer | 否 | Semantic model ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | File execution list | `workflowapp.FileExecutionsResponse` |
| 401 | 未认证 | `object` |

响应字段 (`workflowapp.FileExecutionsResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| executions | []workflowapp.FileExecutionSummary |  | - |
| total | integer |  | - |

响应示例:

```json
{
  "executions": [{
    "case_id": "string",
    "case_start_state": "string",
    "created_at": "string",
    "ended_at": "string",
    "error": "string",
    "execution_id": "string",
    "execution_mode": "string",
    "scheduler_visible": false,
    "started_at": "string",
    "status": "string",
    "updated_at": "string",
    "workflow_id": "string"
  }],
  "total": 0
}
```

---

### GET /api/v1/workspaces/{id}/workflow-apps/{workflow_id}/executions

**列出 Workflow Executions**

查询 workflow app 的执行记录

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| workflow_id | string | 是 | Workflow ID |

#### 查询参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| offset | integer | 否 | Offset |
| limit | integer | 否 | Limit |
| status | string | 否 | Execution status |
| execution_mode | string | 否 | Execution mode |
| workflow_name | string | 否 | Workflow name |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | Execution list | `workflowapp.ExecutionListResponse` |
| 401 | 未认证 | `object` |

响应字段 (`workflowapp.ExecutionListResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| executions | []workflowapp.ExecutionDetail |  | - |
| total | integer |  | - |

响应示例:

```json
{
  "executions": [{
    "case_error": "string",
    "case_result": "string",
    "created_at": "string",
    "cron_expression": "string",
    "data_name": "string",
    "dispatch_job_id": "string",
    "ended_at": "string",
    "error": "string",
    "execution_id": "string",
    "execution_mode": "string",
    "failure": {
      "case_id": "string",
      "code": "string",
      "details": {},
      "message": "string",
      "node_id": "string",
      "node_name": "string",
      "parallel_index": 0,
      "raw_error": "string",
      "span_id": "string",
      "span_kind": "string",
      "stage": "string",
      "task_id": "string",
      "trace_id": "string",
      "worker_id": "string",
      "workitem_id": "string"
    },
    "input_payload": {},
    "input_payload_json": "string",
    "moi_case_id": "string",
    "moi_task_id": "string",
    "moi_workflow_def_id": "string",
    "moi_workflow_version_id": "string",
    "pause_scope": "string",
    "started_at": "string",
    "status": "string",
    "updated_at": "string",
    "vars_payload": {},
    "vars_payload_json": "string",
    "workflow_id": "string",
    "workflow_name": "string"
  }],
  "total": 0
}
```

---

### POST /api/v1/workspaces/{id}/workflow-apps/{workflow_id}/executions

**创建 Workflow Execution**

为 workflow app 创建一次执行；当 vars_payload_json.source_ref 指向 Volume 数据源时，会扫描该 Volume 当前关联的全部文件，并为每个文件创建独立的文件级 workflow execution；文件级 execution 保留原始触发来源，手动派发记录为 one_shot，定时派发记录为 cron，自动卷触发记录为 volume_trigger。source_type=memory_governance 的工作流只能由同步会话触发，此接口拒绝手动执行。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| workflow_id | string | 是 | Workflow ID |

#### 请求体

类型: `handlers.WorkflowAppCreateExecutionRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| compute_resource_id | string |  | - |
| input_payload_json | string |  | - |
| run_once | boolean |  | - |
| trigger_now | boolean |  | - |
| vars_payload_json | string |  | - |

示例:

```json
{
  "compute_resource_id": "string",
  "input_payload_json": "string",
  "run_once": false,
  "trigger_now": false,
  "vars_payload_json": "string"
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | Execution detail | `workflowapp.ExecutionEnvelope` |
| 400 | 参数错误 | `object` |
| 401 | 未认证 | `object` |
| 409 | 记忆治理工作流不支持手动执行 | `object` |

响应字段 (`workflowapp.ExecutionEnvelope`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| execution | workflowapp.ExecutionDetail |  | - |

响应示例:

```json
{
  "execution": {
    "case_error": "string",
    "case_result": "string",
    "created_at": "string",
    "cron_expression": "string",
    "data_name": "string",
    "dispatch_job_id": "string",
    "ended_at": "string",
    "error": "string",
    "execution_id": "string",
    "execution_mode": "string",
    "failure": {
      "case_id": "string",
      "code": "string",
      "details": {},
      "message": "string",
      "node_id": "string",
      "node_name": "string",
      "parallel_index": 0,
      "raw_error": "string",
      "span_id": "string",
      "span_kind": "string",
      "stage": "string",
      "task_id": "string",
      "trace_id": "string",
      "worker_id": "string",
      "workitem_id": "string"
    },
    "input_payload": {},
    "input_payload_json": "string",
    "moi_case_id": "string",
    "moi_task_id": "string",
    "moi_workflow_def_id": "string",
    "moi_workflow_version_id": "string",
    "pause_scope": "string",
    "started_at": "string",
    "status": "string",
    "updated_at": "string",
    "vars_payload": {},
    "vars_payload_json": "string",
    "workflow_id": "string",
    "workflow_name": "string"
  }
}
```

---

### POST /api/v1/workspaces/{id}/workflow-apps/{workflow_id}/executions/batch-retry

**批量重试失败的 Volume 文件执行**

在一个来源 dispatch job 内创建单个 file_multi 重试批次

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| workflow_id | string | 是 | Workflow ID |

#### 请求体

类型: `workflowapp.BatchRetryExecutionsRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| all_failed | boolean |  | - |
| execution_ids | []string |  | - |
| request_id | string |  | - |
| source_dispatch_job_id | string |  | - |

示例:

```json
{
  "all_failed": false,
  "execution_ids": ["string"],
  "request_id": "string",
  "source_dispatch_job_id": "string"
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | Dispatch job | `workflowapp.BatchRetryExecutionsResponse` |
| 400 | 参数错误 | `object` |
| 401 | 未认证 | `object` |

响应字段 (`workflowapp.BatchRetryExecutionsResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| accepted_count | integer |  | - |
| dispatch_job_id | string |  | - |
| replayed | boolean |  | - |
| source_dispatch_job_id | string |  | - |
| status | string |  | - |

响应示例:

```json
{
  "accepted_count": 0,
  "dispatch_job_id": "string",
  "replayed": false,
  "source_dispatch_job_id": "string",
  "status": "string"
}
```

---

### GET /api/v1/workspaces/{id}/workflow-apps/{workflow_id}/executions/{execution_id}

**获取 Workflow Execution**

查询一次 workflow 执行详情

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| workflow_id | string | 是 | Workflow ID |
| execution_id | string | 是 | Execution ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | Execution detail | `workflowapp.ExecutionEnvelope` |
| 401 | 未认证 | `object` |
| 404 | 资源不存在 | `object` |

响应字段 (`workflowapp.ExecutionEnvelope`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| execution | workflowapp.ExecutionDetail |  | - |

响应示例:

```json
{
  "execution": {
    "case_error": "string",
    "case_result": "string",
    "created_at": "string",
    "cron_expression": "string",
    "data_name": "string",
    "dispatch_job_id": "string",
    "ended_at": "string",
    "error": "string",
    "execution_id": "string",
    "execution_mode": "string",
    "failure": {
      "case_id": "string",
      "code": "string",
      "details": {},
      "message": "string",
      "node_id": "string",
      "node_name": "string",
      "parallel_index": 0,
      "raw_error": "string",
      "span_id": "string",
      "span_kind": "string",
      "stage": "string",
      "task_id": "string",
      "trace_id": "string",
      "worker_id": "string",
      "workitem_id": "string"
    },
    "input_payload": {},
    "input_payload_json": "string",
    "moi_case_id": "string",
    "moi_task_id": "string",
    "moi_workflow_def_id": "string",
    "moi_workflow_version_id": "string",
    "pause_scope": "string",
    "started_at": "string",
    "status": "string",
    "updated_at": "string",
    "vars_payload": {},
    "vars_payload_json": "string",
    "workflow_id": "string",
    "workflow_name": "string"
  }
}
```

---

### DELETE /api/v1/workspaces/{id}/workflow-apps/{workflow_id}/executions/{execution_id}

**删除 Workflow Execution**

删除一次 workflow 执行记录

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| workflow_id | string | 是 | Workflow ID |
| execution_id | string | 是 | Execution ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | Delete result | `workflowapp.DeleteResponse` |
| 401 | 未认证 | `object` |
| 404 | 资源不存在 | `object` |

响应字段 (`workflowapp.DeleteResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| deleted | boolean |  | - |
| disabled_cron_task_ids | []string |  | - |
| warnings | []string |  | - |

响应示例:

```json
{
  "deleted": false,
  "disabled_cron_task_ids": ["string"],
  "warnings": ["string"]
}
```

---

### POST /api/v1/workspaces/{id}/workflow-apps/{workflow_id}/executions/{execution_id}/cancel

**取消 Workflow Execution**

取消正在运行的 workflow 执行

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| workflow_id | string | 是 | Workflow ID |
| execution_id | string | 是 | Execution ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | Execution detail | `workflowapp.ExecutionEnvelope` |
| 400 | 参数错误 | `object` |
| 401 | 未认证 | `object` |

响应字段 (`workflowapp.ExecutionEnvelope`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| execution | workflowapp.ExecutionDetail |  | - |

响应示例:

```json
{
  "execution": {
    "case_error": "string",
    "case_result": "string",
    "created_at": "string",
    "cron_expression": "string",
    "data_name": "string",
    "dispatch_job_id": "string",
    "ended_at": "string",
    "error": "string",
    "execution_id": "string",
    "execution_mode": "string",
    "failure": {
      "case_id": "string",
      "code": "string",
      "details": {},
      "message": "string",
      "node_id": "string",
      "node_name": "string",
      "parallel_index": 0,
      "raw_error": "string",
      "span_id": "string",
      "span_kind": "string",
      "stage": "string",
      "task_id": "string",
      "trace_id": "string",
      "worker_id": "string",
      "workitem_id": "string"
    },
    "input_payload": {},
    "input_payload_json": "string",
    "moi_case_id": "string",
    "moi_task_id": "string",
    "moi_workflow_def_id": "string",
    "moi_workflow_version_id": "string",
    "pause_scope": "string",
    "started_at": "string",
    "status": "string",
    "updated_at": "string",
    "vars_payload": {},
    "vars_payload_json": "string",
    "workflow_id": "string",
    "workflow_name": "string"
  }
}
```

---

### POST /api/v1/workspaces/{id}/workflow-apps/{workflow_id}/executions/{execution_id}/pause

**暂停 Workflow Execution**

在安全调度边界暂停正在运行的 workflow 执行

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| workflow_id | string | 是 | Workflow ID |
| execution_id | string | 是 | Execution ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | Execution detail | `workflowapp.ExecutionEnvelope` |
| 400 | 参数错误 | `object` |
| 401 | 未认证 | `object` |

响应字段 (`workflowapp.ExecutionEnvelope`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| execution | workflowapp.ExecutionDetail |  | - |

响应示例:

```json
{
  "execution": {
    "case_error": "string",
    "case_result": "string",
    "created_at": "string",
    "cron_expression": "string",
    "data_name": "string",
    "dispatch_job_id": "string",
    "ended_at": "string",
    "error": "string",
    "execution_id": "string",
    "execution_mode": "string",
    "failure": {
      "case_id": "string",
      "code": "string",
      "details": {},
      "message": "string",
      "node_id": "string",
      "node_name": "string",
      "parallel_index": 0,
      "raw_error": "string",
      "span_id": "string",
      "span_kind": "string",
      "stage": "string",
      "task_id": "string",
      "trace_id": "string",
      "worker_id": "string",
      "workitem_id": "string"
    },
    "input_payload": {},
    "input_payload_json": "string",
    "moi_case_id": "string",
    "moi_task_id": "string",
    "moi_workflow_def_id": "string",
    "moi_workflow_version_id": "string",
    "pause_scope": "string",
    "started_at": "string",
    "status": "string",
    "updated_at": "string",
    "vars_payload": {},
    "vars_payload_json": "string",
    "workflow_id": "string",
    "workflow_name": "string"
  }
}
```

---

### GET /api/v1/workspaces/{id}/workflow-apps/{workflow_id}/executions/{execution_id}/result

**获取 Workflow Execution 结果**

刷新并返回一次 workflow 执行结果

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| workflow_id | string | 是 | Workflow ID |
| execution_id | string | 是 | Execution ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | Execution result | `workflowapp.ExecutionEnvelope` |
| 401 | 未认证 | `object` |
| 404 | 资源不存在 | `object` |

响应字段 (`workflowapp.ExecutionEnvelope`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| execution | workflowapp.ExecutionDetail |  | - |

响应示例:

```json
{
  "execution": {
    "case_error": "string",
    "case_result": "string",
    "created_at": "string",
    "cron_expression": "string",
    "data_name": "string",
    "dispatch_job_id": "string",
    "ended_at": "string",
    "error": "string",
    "execution_id": "string",
    "execution_mode": "string",
    "failure": {
      "case_id": "string",
      "code": "string",
      "details": {},
      "message": "string",
      "node_id": "string",
      "node_name": "string",
      "parallel_index": 0,
      "raw_error": "string",
      "span_id": "string",
      "span_kind": "string",
      "stage": "string",
      "task_id": "string",
      "trace_id": "string",
      "worker_id": "string",
      "workitem_id": "string"
    },
    "input_payload": {},
    "input_payload_json": "string",
    "moi_case_id": "string",
    "moi_task_id": "string",
    "moi_workflow_def_id": "string",
    "moi_workflow_version_id": "string",
    "pause_scope": "string",
    "started_at": "string",
    "status": "string",
    "updated_at": "string",
    "vars_payload": {},
    "vars_payload_json": "string",
    "workflow_id": "string",
    "workflow_name": "string"
  }
}
```

---

### POST /api/v1/workspaces/{id}/workflow-apps/{workflow_id}/executions/{execution_id}/resume

**恢复 Workflow Execution**

恢复已暂停的 workflow 执行

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| workflow_id | string | 是 | Workflow ID |
| execution_id | string | 是 | Execution ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | Execution detail | `workflowapp.ExecutionEnvelope` |
| 400 | 参数错误 | `object` |
| 401 | 未认证 | `object` |

响应字段 (`workflowapp.ExecutionEnvelope`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| execution | workflowapp.ExecutionDetail |  | - |

响应示例:

```json
{
  "execution": {
    "case_error": "string",
    "case_result": "string",
    "created_at": "string",
    "cron_expression": "string",
    "data_name": "string",
    "dispatch_job_id": "string",
    "ended_at": "string",
    "error": "string",
    "execution_id": "string",
    "execution_mode": "string",
    "failure": {
      "case_id": "string",
      "code": "string",
      "details": {},
      "message": "string",
      "node_id": "string",
      "node_name": "string",
      "parallel_index": 0,
      "raw_error": "string",
      "span_id": "string",
      "span_kind": "string",
      "stage": "string",
      "task_id": "string",
      "trace_id": "string",
      "worker_id": "string",
      "workitem_id": "string"
    },
    "input_payload": {},
    "input_payload_json": "string",
    "moi_case_id": "string",
    "moi_task_id": "string",
    "moi_workflow_def_id": "string",
    "moi_workflow_version_id": "string",
    "pause_scope": "string",
    "started_at": "string",
    "status": "string",
    "updated_at": "string",
    "vars_payload": {},
    "vars_payload_json": "string",
    "workflow_id": "string",
    "workflow_name": "string"
  }
}
```

---

### POST /api/v1/workspaces/{id}/workflow-apps/{workflow_id}/executions/{execution_id}/retry

**重试 Workflow Execution**

使用同一输入创建新的 workflow 执行

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| workflow_id | string | 是 | Workflow ID |
| execution_id | string | 是 | Execution ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | Execution detail | `workflowapp.ExecutionEnvelope` |
| 400 | 参数错误 | `object` |
| 401 | 未认证 | `object` |

响应字段 (`workflowapp.ExecutionEnvelope`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| execution | workflowapp.ExecutionDetail |  | - |

响应示例:

```json
{
  "execution": {
    "case_error": "string",
    "case_result": "string",
    "created_at": "string",
    "cron_expression": "string",
    "data_name": "string",
    "dispatch_job_id": "string",
    "ended_at": "string",
    "error": "string",
    "execution_id": "string",
    "execution_mode": "string",
    "failure": {
      "case_id": "string",
      "code": "string",
      "details": {},
      "message": "string",
      "node_id": "string",
      "node_name": "string",
      "parallel_index": 0,
      "raw_error": "string",
      "span_id": "string",
      "span_kind": "string",
      "stage": "string",
      "task_id": "string",
      "trace_id": "string",
      "worker_id": "string",
      "workitem_id": "string"
    },
    "input_payload": {},
    "input_payload_json": "string",
    "moi_case_id": "string",
    "moi_task_id": "string",
    "moi_workflow_def_id": "string",
    "moi_workflow_version_id": "string",
    "pause_scope": "string",
    "started_at": "string",
    "status": "string",
    "updated_at": "string",
    "vars_payload": {},
    "vars_payload_json": "string",
    "workflow_id": "string",
    "workflow_name": "string"
  }
}
```

---

### POST /api/v1/workspaces/{id}/workflow-apps/{workflow_id}/memory-session-executions

**运行会话触发的记忆治理工作流**

由 Backend 在运行期重新校验 workflow.run、connector.use 和自定义算子依赖后调用；Run ID 同时作为幂等执行标识。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| workflow_id | string | 是 | Workflow ID |

#### 请求体

类型: `handlers.memoryGovernanceExecutionRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| connector_id | string |  | - |
| data | []integer |  | - |
| platform_managed | boolean |  | - |
| run_id | string |  | - |
| workflow_version_id | string |  | - |

示例:

```json
{
  "connector_id": "string",
  "data": ["string"],
  "platform_managed": false,
  "run_id": "string",
  "workflow_version_id": "string"
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | Execution identity | `workflowapp.ImmediateExecutionResult` |
| 400 | 参数错误 | `object` |
| 401 | 未认证 | `object` |
| 403 | 缺少精确 workflow.run 授权事实 | `object` |
| 409 | 不可重试的版本或生命周期冲突 | `object` |

响应字段 (`workflowapp.ImmediateExecutionResult`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| case_id | string |  | - |
| execution_id | string |  | - |
| task_id | string |  | - |

响应示例:

```json
{
  "case_id": "string",
  "execution_id": "string",
  "task_id": "string"
}
```

---

## Workflow Lineage

### GET /api/v1/workspaces/{id}/mowl/assets/{root_asset_id}/cases

**列出 root asset 对应的 lineage cases**

返回一个 root asset 对应的 case 候选列表，用于 lineage overview 切换 case

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| root_asset_id | string | 是 | Root Asset ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 查询成功 | `mowl.ListAssetCasesResponse` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`mowl.ListAssetCasesResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| cases | []mowl.AssetCaseCandidate |  | - |
| root_asset_id | string |  | - |

响应示例:

```json
{
  "cases": [{
    "case_id": "string",
    "created_at": "string",
    "duration_ms": 0,
    "role": "string",
    "selected_by_entry": false,
    "source": "string",
    "status": "string",
    "task_id": "string",
    "workflow_version_id": "string"
  }],
  "root_asset_id": "string"
}
```

---

### GET /api/v1/workspaces/{id}/mowl/assets/{root_asset_id}/cases/{case_id}/artifact

**获取一个 asset case 的聚合产物详情**

返回 root asset 在指定 case 下的聚合 lineage 产物详情

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| root_asset_id | string | 是 | Root Asset ID |
| case_id | string | 是 | Case ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 查询成功 | `mowl.AssetArtifactDetailResponse` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 404 | artifact 不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`mowl.AssetArtifactDetailResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| case_id | string |  | - |
| created_at | string |  | - |
| derivation_files | []mowl.ArtifactDerivationFile |  | - |
| parsed_file_available | boolean |  | - |
| parsed_file_id | string |  | - |
| producer_workitem_id | string |  | - |
| raw_file_id | string |  | - |
| recorded_by_workitem_id | string |  | - |
| root_asset_id | string |  | - |
| stage_asset_id | string |  | - |
| status | string |  | - |
| task_id | string |  | - |
| workflow_version_id | string |  | - |

响应示例:

```json
{
  "case_id": "string",
  "created_at": "string",
  "derivation_files": [{
    "file_id": "string",
    "kind": "string",
    "produced_at": "string",
    "producer_workitem_id": "string",
    "recorded_by_workitem_id": "string",
    "stage_asset_id": "string"
  }],
  "parsed_file_available": false,
  "parsed_file_id": "string",
  "producer_workitem_id": "string",
  "raw_file_id": "string",
  "recorded_by_workitem_id": "string",
  "root_asset_id": "string",
  "stage_asset_id": "string",
  "status": "string",
  "task_id": "string",
  "workflow_version_id": "string"
}
```

---

### GET /api/v1/workspaces/{id}/mowl/cases/{case_id}/invocation

**获取 case 级调用快照**

返回 case 的 invocation input 与 vars 快照，vars 中会清理服务端注入元数据

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| case_id | string | 是 | Case ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 查询成功 | `mowl.CaseInvocationResponse` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 404 | case 不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`mowl.CaseInvocationResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| case_id | string |  | - |
| input |  |  | - |
| vars |  |  | - |

响应示例:

```json
{
  "case_id": "string",
  "input": "",
  "vars": ""
}
```

---

### GET /api/v1/workspaces/{id}/mowl/cases/{case_id}/workitems

**列出 case 的 lineage workitem 快照**

返回一个 case 下的 workitem lineage 摘要列表，可按 root_asset_id 附带当前产物匹配信息

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| case_id | string | 是 | Case ID |

#### 查询参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| root_asset_id | string | 否 | Root Asset ID |
| include_runtime_input_snapshot | boolean | 否 | 是否在摘要中附带 runtime_input 快照 |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 查询成功 | `mowl.ListCaseWorkItemsResponse` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 404 | case 不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`mowl.ListCaseWorkItemsResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| case_id | string |  | - |
| items | []mowl.WorkItemSnapshotSummary |  | - |
| root_asset_id | string |  | - |
| workflow_version_id | string |  | - |

响应示例:

```json
{
  "case_id": "string",
  "items": [{
    "anchor_source": "string",
    "case_id": "string",
    "config_preview": "string",
    "duration_ms": 0,
    "error": "string",
    "input_content_ref": "string",
    "input_preview": "string",
    "last_updated_at": "string",
    "matched_current_artifact": false,
    "matched_current_artifact_fields": ["string"],
    "matched_current_artifact_ids": ["string"],
    "matched_stage_asset_ids": ["string"],
    "node_execution_id": "string",
    "node_id": "string",
    "output_content_ref": "string",
    "output_preview": "string",
    "parallel_index": 0,
    "runtime_input_snapshot": "",
    "status": "string",
    "workflow_version_id": "string",
    "workitem_id": "string",
    "workitem_type_id": "string"
  }],
  "root_asset_id": "string",
  "workflow_version_id": "string"
}
```

---

### GET /api/v1/workspaces/{id}/mowl/cases/{case_id}/workitems/{workitem_id}

**获取单个 workitem 的 lineage 快照详情**

返回一个 case workitem 的配置、输入、变量与输出快照详情

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| case_id | string | 是 | Case ID |
| workitem_id | string | 是 | Workitem ID |

#### 查询参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| parallel_index | integer | 否 | Parallel Index |
| root_asset_id | string | 否 | Root Asset ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 查询成功 | `mowl.GetCaseWorkItemResponse` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 404 | workitem 不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`mowl.GetCaseWorkItemResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| case_id | string |  | - |
| item | mowl.WorkItemSnapshotDetail |  | - |
| workflow_version_id | string |  | - |

响应示例:

```json
{
  "case_id": "string",
  "item": {
    "case_id": "string",
    "config_snapshot": "",
    "duration_ms": 0,
    "error": "string",
    "extensions": {},
    "last_updated_at": "string",
    "node_execution_id": "string",
    "node_id": "string",
    "parallel_index": 0,
    "root_asset_id": "string",
    "runtime_input_snapshot": "",
    "runtime_output_snapshot": "",
    "runtime_vars_snapshot": "",
    "status": "string",
    "workflow_version_id": "string",
    "workitem_id": "string",
    "workitem_type_id": "string"
  },
  "workflow_version_id": "string"
}
```

---

### GET /api/v1/workspaces/{id}/mowl/cases/{case_id}/workitems/{workitem_id}/artifact-scope

**获取单个 workitem 的 artifact-scope 视图**

将一个 workitem 的运行时负载收敛到指定 root_asset_id 的产物作用域

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| case_id | string | 是 | Case ID |
| workitem_id | string | 是 | Workitem ID |

#### 查询参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| root_asset_id | string | 是 | Root Asset ID |
| parallel_index | integer | 否 | Parallel Index |
| preview_limit | integer | 否 | Preview Limit |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 查询成功 | `mowl.ArtifactScopeResponse` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 404 | workitem 不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`mowl.ArtifactScopeResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| artifact_id | string |  | - |
| artifact_refs | []mowl.LineageArtifactRef |  | - |
| case_id | string |  | - |
| input_projection | [] |  | - |
| matched_stage_asset_ids | []string |  | - |
| node_id | string |  | - |
| output_projection | [] |  | - |
| parallel_index | integer |  | - |
| preview | mowl.LineageFileScopePreview |  | - |
| resolution_note | string |  | - |
| root_asset_id | string |  | - |
| scope | mowl.LineageFileScopeSummary |  | - |
| scope_source | string |  | - |
| workflow_version_id | string |  | - |
| workitem_id | string |  | - |
| workitem_type_id | string |  | - |

响应示例:

```json
{
  "artifact_id": "string",
  "artifact_refs": [{
    "id": "string",
    "matched": false,
    "name": "string",
    "source": "string",
    "type": "string"
  }],
  "case_id": "string",
  "input_projection": ["string"],
  "matched_stage_asset_ids": ["string"],
  "node_id": "string",
  "output_projection": ["string"],
  "parallel_index": 0,
  "preview": {
    "content_format": "string",
    "full_content_ref": "string",
    "items": ["string"],
    "overflow_hint": "string"
  },
  "resolution_note": "string",
  "root_asset_id": "string",
  "scope": {
    "collection_type": "string",
    "dimensions": ["string"],
    "matched_count": 0,
    "summary": "string",
    "total_count": 0
  },
  "scope_source": "string",
  "workflow_version_id": "string",
  "workitem_id": "string",
  "workitem_type_id": "string"
}
```

---

## Workflow 管理

### POST /api/v1/workspaces/{id}/workflow-versions

**创建 Workflow 版本**

为指定 Workflow 创建新版本

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 请求体

类型: JSON 对象

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 201 | 创建成功 | `object` |
| 400 | 参数错误 | `object` |
| 401 | 未认证 | `object` |
| 500 | 内部错误 | `object` |

---

### GET /api/v1/workspaces/{id}/workflow-versions/{version_id}

**获取 Workflow 版本**

根据版本 ID 获取指定 Workflow 版本的详细信息

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| version_id | string | 是 | 版本 ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 版本详情 | `object` |
| 400 | 参数错误 | `object` |
| 401 | 未认证 | `object` |
| 404 | 版本不存在 | `object` |
| 500 | 内部错误 | `object` |

---

### DELETE /api/v1/workspaces/{id}/workflow-versions/{version_id}

**删除 Workflow 版本**

删除指定 Workflow 版本

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| version_id | string | 是 | 版本 ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 204 | 删除成功 |  |
| 400 | 参数错误 | `object` |
| 401 | 未认证 | `object` |
| 500 | 内部错误 | `object` |

---

### POST /api/v1/workspaces/{id}/workflow-versions/{version_id}/deprecate

**弃用 Workflow 版本**

弃用指定 Workflow 版本，使其不再可用

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| version_id | string | 是 | 版本 ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 弃用成功 | `object` |
| 400 | 参数错误 | `object` |
| 401 | 未认证 | `object` |
| 500 | 内部错误 | `object` |

---

### POST /api/v1/workspaces/{id}/workflow-versions/{version_id}/publish

**发布 Workflow 版本**

发布指定 Workflow 版本，使其可用于执行

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| version_id | string | 是 | 版本 ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 发布成功 | `object` |
| 400 | 参数错误 | `object` |
| 401 | 未认证 | `object` |
| 500 | 内部错误 | `object` |

---

### GET /api/v1/workspaces/{id}/workflows

**列出 Workflow**

列出指定 workspace 中的所有 Workflow 定义

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 查询参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 否 | 按名称过滤 |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | Workflow 列表 | `object` |
| 400 | 参数错误 | `object` |
| 401 | 未认证 | `object` |
| 500 | 内部错误 | `object` |

---

### POST /api/v1/workspaces/{id}/workflows

**创建 Workflow**

在指定 workspace 中创建新的 Workflow 定义

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 请求体

类型: JSON 对象

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 201 | 创建成功 | `object` |
| 400 | 参数错误 | `object` |
| 401 | 未认证 | `object` |
| 500 | 内部错误 | `object` |

---

### GET /api/v1/workspaces/{id}/workflows/by-name/{name}

**按名称获取 Workflow**

根据名称获取指定 Workflow 定义的详细信息

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| name | string | 是 | Workflow 名称 |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | Workflow 详情 | `object` |
| 400 | 参数错误 | `object` |
| 401 | 未认证 | `object` |
| 404 | Workflow 不存在 | `object` |
| 500 | 内部错误 | `object` |

---

### GET /api/v1/workspaces/{id}/workflows/{workflow_id}

**获取 Workflow**

根据 ID 获取指定 Workflow 定义的详细信息

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| workflow_id | string | 是 | Workflow ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | Workflow 详情 | `object` |
| 400 | 参数错误 | `object` |
| 401 | 未认证 | `object` |
| 404 | Workflow 不存在 | `object` |
| 500 | 内部错误 | `object` |

---

### PUT /api/v1/workspaces/{id}/workflows/{workflow_id}

**更新 Workflow**

更新指定 Workflow 定义的名称或描述

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| workflow_id | string | 是 | Workflow ID |

#### 请求体

类型: JSON 对象

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 更新成功 | `object` |
| 400 | 参数错误 | `object` |
| 401 | 未认证 | `object` |
| 500 | 内部错误 | `object` |

---

### DELETE /api/v1/workspaces/{id}/workflows/{workflow_id}

**删除 Workflow**

删除指定 Workflow 定义

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| workflow_id | string | 是 | Workflow ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 204 | 删除成功 |  |
| 400 | 参数错误 | `object` |
| 401 | 未认证 | `object` |
| 500 | 内部错误 | `object` |

---

### GET /api/v1/workspaces/{id}/workflows/{workflow_id}/versions

**列出 Workflow 版本**

列出指定 Workflow 的所有版本

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| workflow_id | string | 是 | Workflow ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 版本列表 | `object` |
| 400 | 参数错误 | `object` |
| 401 | 未认证 | `object` |
| 500 | 内部错误 | `object` |

---

### GET /api/v1/workspaces/{id}/workflows/{workflow_id}/versions/latest

**获取最新已发布版本**

获取指定 Workflow 的最新已发布版本

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| workflow_id | string | 是 | Workflow ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 最新已发布版本详情 | `object` |
| 400 | 参数错误 | `object` |
| 401 | 未认证 | `object` |
| 404 | 版本不存在 | `object` |
| 500 | 内部错误 | `object` |

---

### GET /api/v1/workspaces/{id}/workflows/{workflow_id}/versions/{version}

**按版本号获取 Workflow 版本**

根据 Workflow ID 和版本号获取指定版本的详细信息

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| workflow_id | string | 是 | Workflow ID |
| version | integer | 是 | 版本号 |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 版本详情 | `object` |
| 400 | 参数错误 | `object` |
| 401 | 未认证 | `object` |
| 404 | 版本不存在 | `object` |
| 500 | 内部错误 | `object` |

---

## Workspace - Runtime Dynamic Config

### GET /api/v1/workspaces/{id}/runtime-configs/{namespace}/{config_key}

**获取 Workspace Effective Runtime Dynamic Config**

Workspace Override 优先，否则显式继承 Global。仅 Workspace Admin 或 System User 可访问。ETag 为 Override Revision；无 Override 时为 0。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| namespace | string | 是 | 配置命名空间 |
| config_key | string | 是 | 配置键 |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | OK | `handlers.WorkspaceRuntimeDynamicConfigResponse` |
| 400 | Bad Request | `gin.ErrorResponse` |
| 401 | Unauthorized | `gin.ErrorResponse` |
| 403 | Forbidden | `gin.ErrorResponse` |
| 404 | Not Found | `gin.ErrorResponse` |
| 500 | Internal Server Error | `gin.ErrorResponse` |

响应字段 (`handlers.WorkspaceRuntimeDynamicConfigResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| config_key | string |  | - |
| created_at | string |  | - |
| effective_revision | integer |  | - |
| namespace | string |  | - |
| override_present | boolean |  | - |
| override_revision | integer |  | - |
| resolved_scope | string |  | - |
| resolved_scope_id | string |  | - |
| schema_version | integer |  | - |
| updated_at | string |  | - |
| updated_by | string |  | - |
| value | object |  | - |

响应示例:

```json
{
  "config_key": "string",
  "created_at": "string",
  "effective_revision": 0,
  "namespace": "string",
  "override_present": false,
  "override_revision": 0,
  "resolved_scope": "string",
  "resolved_scope_id": "string",
  "schema_version": 0,
  "updated_at": "string",
  "updated_by": "string",
  "value": {}
}
```

---

### PUT /api/v1/workspaces/{id}/runtime-configs/{namespace}/{config_key}

**写入 Workspace Runtime Dynamic Config Override**

expected_revision=0 创建 Override；大于 0 时使用 Override Revision 执行 CAS 更新，不能使用 Global Revision。仅 Workspace Admin 或 System User 可访问。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| namespace | string | 是 | 配置命名空间 |
| config_key | string | 是 | 配置键 |

#### 请求体

类型: `handlers.PutRuntimeDynamicConfigRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| expected_revision | integer |  | - |
| schema_version | integer | 是 | - |
| value | object | 是 | - |

示例:

```json
{
  "expected_revision": 0,
  "schema_version": 0,
  "value": {}
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | OK | `handlers.WorkspaceRuntimeDynamicConfigResponse` |
| 400 | Bad Request | `gin.ErrorResponse` |
| 401 | Unauthorized | `gin.ErrorResponse` |
| 403 | Forbidden | `gin.ErrorResponse` |
| 409 | Conflict | `gin.ErrorResponse` |
| 412 | Precondition Failed | `gin.ErrorResponse` |
| 500 | Internal Server Error | `gin.ErrorResponse` |

响应字段 (`handlers.WorkspaceRuntimeDynamicConfigResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| config_key | string |  | - |
| created_at | string |  | - |
| effective_revision | integer |  | - |
| namespace | string |  | - |
| override_present | boolean |  | - |
| override_revision | integer |  | - |
| resolved_scope | string |  | - |
| resolved_scope_id | string |  | - |
| schema_version | integer |  | - |
| updated_at | string |  | - |
| updated_by | string |  | - |
| value | object |  | - |

响应示例:

```json
{
  "config_key": "string",
  "created_at": "string",
  "effective_revision": 0,
  "namespace": "string",
  "override_present": false,
  "override_revision": 0,
  "resolved_scope": "string",
  "resolved_scope_id": "string",
  "schema_version": 0,
  "updated_at": "string",
  "updated_by": "string",
  "value": {}
}
```

---

### DELETE /api/v1/workspaces/{id}/runtime-configs/{namespace}/{config_key}

**删除 Workspace Runtime Dynamic Config Override**

删除必须匹配当前 Override Revision。删除后返回 Global Effective Config；Global 不存在时明确报错。仅 Workspace Admin 或 System User 可访问。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| namespace | string | 是 | 配置命名空间 |
| config_key | string | 是 | 配置键 |

#### 查询参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| expected_revision | integer | 否 | 期望 Override Revision |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | OK | `handlers.WorkspaceRuntimeDynamicConfigResponse` |
| 400 | Bad Request | `gin.ErrorResponse` |
| 401 | Unauthorized | `gin.ErrorResponse` |
| 403 | Forbidden | `gin.ErrorResponse` |
| 404 | Not Found | `gin.ErrorResponse` |
| 409 | Conflict | `gin.ErrorResponse` |
| 500 | Internal Server Error | `gin.ErrorResponse` |

响应字段 (`handlers.WorkspaceRuntimeDynamicConfigResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| config_key | string |  | - |
| created_at | string |  | - |
| effective_revision | integer |  | - |
| namespace | string |  | - |
| override_present | boolean |  | - |
| override_revision | integer |  | - |
| resolved_scope | string |  | - |
| resolved_scope_id | string |  | - |
| schema_version | integer |  | - |
| updated_at | string |  | - |
| updated_by | string |  | - |
| value | object |  | - |

响应示例:

```json
{
  "config_key": "string",
  "created_at": "string",
  "effective_revision": 0,
  "namespace": "string",
  "override_present": false,
  "override_revision": 0,
  "resolved_scope": "string",
  "resolved_scope_id": "string",
  "schema_version": 0,
  "updated_at": "string",
  "updated_by": "string",
  "value": {}
}
```

---

## Workspace 管理

### POST /api/v1/system/workspaces/{id}/invitations

**Create a workspace invitation**

Creates a Core-owned invitation for an existing principal identified by canonical Core user ID. complete_immediately=false (default) leaves the invitation pending; complete_immediately=true runs the acceptance Saga as the inviter's frozen Effective Role. Empty initial roles are allowed only when completing immediately. This system endpoint is intended for the Backend BFF after canonical IAM authorization.

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 请求体

类型: `handlers.createWorkspaceInvitationRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| complete_immediately | boolean |  | - |
| default_role_id | string |  | - |
| effective_role_id | string | 是 | - |
| initial_role_ids | []string |  | - |
| invited_by_user_id | string | 是 | - |
| member_alias | string | 是 | - |
| member_description | string |  | - |
| request_id | string | 是 | - |
| subject_attributes | []workspace.InvitationSubjectAttribute |  | - |
| target_user_id | string | 是 | Canonical Core users.id resolved by the BFF. |
| trace_id | string | 是 | - |

示例:

```json
{
  "complete_immediately": false,
  "default_role_id": "string",
  "effective_role_id": "string",
  "initial_role_ids": ["string"],
  "invited_by_user_id": "string",
  "member_alias": "string",
  "member_description": "string",
  "request_id": "string",
  "subject_attributes": [{
    "attribute_id": 0,
    "value": "string"
  }],
  "target_user_id": "string",
  "trace_id": "string"
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | Invitation created | `handlers.workspaceInvitationResponse` |
| 400 | Invalid request | `gin.ErrorResponse` |
| 403 | Permission denied | `gin.ErrorResponse` |
| 404 | Invitation target not found | `gin.ErrorResponse` |
| 409 | Invitation or member already exists | `gin.ErrorResponse` |
| 500 | Internal error | `gin.ErrorResponse` |
| 503 | Invitation service or tenant transaction manager unavailable | `gin.ErrorResponse` |

响应字段 (`handlers.workspaceInvitationResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| created_at | integer |  | - |
| default_role_id | string |  | - |
| id | string |  | - |
| initial_role_ids | []string |  | - |
| invited_by_user_id | string |  | - |
| member_alias | string |  | - |
| member_description | string |  | - |
| owner_id | string |  | - |
| status | string |  | - |
| target_user_id | string |  | - |
| workspace_id | string |  | - |
| workspace_name | string |  | - |
| workspace_status | integer |  | - |

响应示例:

```json
{
  "created_at": 0,
  "default_role_id": "string",
  "id": "string",
  "initial_role_ids": ["string"],
  "invited_by_user_id": "string",
  "member_alias": "string",
  "member_description": "string",
  "owner_id": "string",
  "status": "string",
  "target_user_id": "string",
  "workspace_id": "string",
  "workspace_name": "string",
  "workspace_status": 0
}
```

---

### GET /api/v1/system/workspaces/{id}/invitations/pending

**List pending invitations for a workspace**

Lists pending invitations targeting the given workspace. This system endpoint is intended for the Backend BFF after canonical IAM authorization.

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | Pending invitations | `[]handlers.workspaceInvitationResponse` |
| 403 | Permission denied | `gin.ErrorResponse` |
| 404 | Workspace or invitation not found | `gin.ErrorResponse` |
| 500 | Internal error | `gin.ErrorResponse` |
| 503 | Invitation service unavailable | `gin.ErrorResponse` |

---

### GET /api/v1/workspaces

**列出 Workspace**

列出当前用户可访问的所有 Workspace（仅返回状态为 NORMAL 的工作区，INITIALIZING 和 FAILED 状态不返回）

认证: 需要 API Key

#### 查询参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page_size | integer | 否 | 每页数量 |
| page_token | string | 否 | 分页令牌 |
| owner_id | string | 否 | 精确 workspace owner 过滤条件 |
| shard_count | integer | 否 | 稳定 workspace-ID shard 总数；与 shard_index 成对提供 |
| shard_index | integer | 否 | 稳定 workspace-ID shard 索引，从 0 开始 |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | Workspace 列表 | `workspace.ListWorkspacesResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`workspace.ListWorkspacesResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| next_page_token | string |  | - |
| total | integer |  | - |
| workspaces | []workspace.Workspace |  | - |

响应示例:

```json
{
  "next_page_token": "string",
  "total": 0,
  "workspaces": [{
    "account_name": "string",
    "created_at": 0,
    "description": "string",
    "id": "string",
    "name": "string",
    "owner_changed_at": 0,
    "owner_id": "string",
    "owner_revision": 0,
    "status": {},
    "updated_at": 0
  }]
}
```

---

### POST /api/v1/workspaces

**创建 Workspace**

创建新的 Workspace。创建成功后 workspace 经历 INITIALIZING → NORMAL 状态转换。如果权限初始化失败，返回 500 且 workspace 状态为 FAILED。

认证: 需要 API Key

#### 请求体

类型: `workspace.CreateWorkspaceRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| description | string |  | - |
| idempotency_key | string |  | - |
| name | string |  | - |

示例:

```json
{
  "description": "string",
  "idempotency_key": "string",
  "name": "string"
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 201 | 创建成功 | `workspace.Workspace` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 409 | 名称已存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`workspace.Workspace`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| account_name | string |  | - |
| created_at | integer |  | Unix 时间戳（秒） |
| description | string |  | - |
| id | string |  | - |
| name | string |  | - |
| owner_changed_at | integer |  | Unix 时间戳（微秒） |
| owner_id | string |  | - |
| owner_revision | integer |  | 单调 owner revision |
| status | workspace.WorkspaceStatus |  | - |
| updated_at | integer |  | Unix 时间戳（秒） |

响应示例:

```json
{
  "account_name": "string",
  "created_at": 0,
  "description": "string",
  "id": "string",
  "name": "string",
  "owner_changed_at": 0,
  "owner_id": "string",
  "owner_revision": 0,
  "status": {},
  "updated_at": 0
}
```

---

### GET /api/v1/workspaces/invitations/pending

**List pending workspace invitations**

Lists pending invitations whose target is the authenticated Core principal.

认证: 需要 API Key

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | Pending invitations | `[]handlers.workspaceInvitationResponse` |
| 401 | Unauthenticated | `gin.ErrorResponse` |
| 500 | Internal error | `gin.ErrorResponse` |
| 503 | Invitation service unavailable | `gin.ErrorResponse` |

---

### POST /api/v1/workspaces/invitations/{invitation_id}/accept

**Accept a workspace invitation**

Atomically accepts a pending invitation targeted to the authenticated Core principal and creates the canonical workspace membership and initial role bindings.

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| invitation_id | string | 是 | Invitation ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | Invitation accepted | `handlers.workspaceInvitationResponse` |
| 401 | Unauthenticated | `gin.ErrorResponse` |
| 403 | Invitation target mismatch or inactive account | `gin.ErrorResponse` |
| 404 | Invitation not found | `gin.ErrorResponse` |
| 409 | Invitation cannot be accepted in its current state | `gin.ErrorResponse` |
| 500 | Internal error | `gin.ErrorResponse` |
| 503 | Target lifecycle unavailable | `gin.ErrorResponse` |

响应字段 (`handlers.workspaceInvitationResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| created_at | integer |  | - |
| default_role_id | string |  | - |
| id | string |  | - |
| initial_role_ids | []string |  | - |
| invited_by_user_id | string |  | - |
| member_alias | string |  | - |
| member_description | string |  | - |
| owner_id | string |  | - |
| status | string |  | - |
| target_user_id | string |  | - |
| workspace_id | string |  | - |
| workspace_name | string |  | - |
| workspace_status | integer |  | - |

响应示例:

```json
{
  "created_at": 0,
  "default_role_id": "string",
  "id": "string",
  "initial_role_ids": ["string"],
  "invited_by_user_id": "string",
  "member_alias": "string",
  "member_description": "string",
  "owner_id": "string",
  "status": "string",
  "target_user_id": "string",
  "workspace_id": "string",
  "workspace_name": "string",
  "workspace_status": 0
}
```

---

### GET /api/v1/workspaces/{id}

**获取 Workspace**

根据 ID 获取指定 Workspace 的详细信息

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | Workspace 详情 | `workspace.Workspace` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 404 | Workspace 不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`workspace.Workspace`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| account_name | string |  | - |
| created_at | integer |  | Unix 时间戳（秒） |
| description | string |  | - |
| id | string |  | - |
| name | string |  | - |
| owner_changed_at | integer |  | Unix 时间戳（微秒） |
| owner_id | string |  | - |
| owner_revision | integer |  | 单调 owner revision |
| status | workspace.WorkspaceStatus |  | - |
| updated_at | integer |  | Unix 时间戳（秒） |

响应示例:

```json
{
  "account_name": "string",
  "created_at": 0,
  "description": "string",
  "id": "string",
  "name": "string",
  "owner_changed_at": 0,
  "owner_id": "string",
  "owner_revision": 0,
  "status": {},
  "updated_at": 0
}
```

---

### PUT /api/v1/workspaces/{id}

**更新 Workspace**

仅 Workspace 唯一创建者可以更新 Workspace

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 请求体

类型: `workspace.UpdateWorkspaceRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| description | string |  | - |
| id | string |  | - |
| name | string |  | - |

示例:

```json
{
  "description": "string",
  "id": "string",
  "name": "string"
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 更新成功 | `workspace.Workspace` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 403 | 非 Workspace 创建者 | `gin.ErrorResponse` |
| 404 | Workspace 不存在 | `gin.ErrorResponse` |
| 409 | 仍有分享数据发布、订阅或未完成操作 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |
| 503 | 无法确认分享数据删除前置条件 | `gin.ErrorResponse` |

响应字段 (`workspace.Workspace`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| account_name | string |  | - |
| created_at | integer |  | Unix 时间戳（秒） |
| description | string |  | - |
| id | string |  | - |
| name | string |  | - |
| owner_changed_at | integer |  | Unix 时间戳（微秒） |
| owner_id | string |  | - |
| owner_revision | integer |  | 单调 owner revision |
| status | workspace.WorkspaceStatus |  | - |
| updated_at | integer |  | Unix 时间戳（秒） |

响应示例:

```json
{
  "account_name": "string",
  "created_at": 0,
  "description": "string",
  "id": "string",
  "name": "string",
  "owner_changed_at": 0,
  "owner_id": "string",
  "owner_revision": 0,
  "status": {},
  "updated_at": 0
}
```

---

### DELETE /api/v1/workspaces/{id}

**删除 Workspace**

仅 Workspace 唯一创建者可以删除指定 Workspace

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 204 | 删除成功 |  |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 403 | 非 Workspace 创建者 | `gin.ErrorResponse` |
| 404 | Workspace 不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

---

### GET /api/v1/workspaces/{id}/db-connection

**获取数据库连接信息**

获取指定 Workspace 的数据库连接信息。Workspace 状态必须为 NORMAL，INITIALIZING 或 FAILED 状态返回 503。

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 数据库连接信息 | `workspace.DBConnection` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 403 | 无权限 | `gin.ErrorResponse` |
| 404 | Workspace 不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |
| 503 | Workspace 正在初始化 | `gin.ErrorResponse` |

响应字段 (`workspace.DBConnection`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| account_name | string |  | - |
| database | string |  | - |
| effective_role_db_role | string |  | - |
| effective_role_id | string |  | - |
| effective_role_source | string |  | - |
| effective_role_version | string |  | - |
| host | string |  | - |
| password | string |  | - |
| port | integer |  | - |
| session_init_sqls | []string |  | - |
| username | string |  | - |

响应示例:

```json
{
  "account_name": "string",
  "database": "string",
  "effective_role_db_role": "string",
  "effective_role_id": "string",
  "effective_role_source": "string",
  "effective_role_version": "string",
  "host": "string",
  "password": "string",
  "port": 0,
  "session_init_sqls": ["string"],
  "username": "string"
}
```

---

### POST /api/v1/workspaces/{id}/workspace-names:resolve

**批量解析 Workspace 名称**

在当前 Workspace 有效成员上下文下返回 NORMAL Workspace 的最小 id/name 投影

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | 当前 Workspace ID |

#### 请求体

类型: `handlers.resolveWorkspaceNamesRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| ids | []string |  | - |

示例:

```json
{
  "ids": ["string"]
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | Workspace 名称投影 | `handlers.resolveWorkspaceNamesResponse` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 403 | 当前 Workspace 成员身份或 Effective Role 无效 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`handlers.resolveWorkspaceNamesResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| workspaces | []handlers.workspaceNameProjectionResponse |  | - |

响应示例:

```json
{
  "workspaces": [{
    "id": "string",
    "name": "string"
  }]
}
```

---

## Workspace 管理 (Internal)

### GET /api/v1/system/workspaces

**列出可运行 Workspace（仅系统用户）**

返回全部 NORMAL Workspace，供受信后台控制面 Worker 发现租户；不按调用者成员关系过滤

认证: 需要 API Key

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 可运行 Workspace 列表 | `workspace.ListWorkspacesResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 403 | 非系统用户 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`workspace.ListWorkspacesResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| next_page_token | string |  | - |
| total | integer |  | - |
| workspaces | []workspace.Workspace |  | - |

响应示例:

```json
{
  "next_page_token": "string",
  "total": 0,
  "workspaces": [{
    "account_name": "string",
    "created_at": 0,
    "description": "string",
    "id": "string",
    "name": "string",
    "owner_changed_at": 0,
    "owner_id": "string",
    "owner_revision": 0,
    "status": {},
    "updated_at": 0
  }]
}
```

---

### GET /api/v1/workspaces/{id}/owner-db-connection

**获取 Owner 数据库连接信息 (仅系统用户)**

获取指定 Workspace 的 Owner 数据库连接信息，仅限系统用户调用

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | Owner 数据库连接信息 | `workspace.DBConnection` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 403 | 非系统用户 | `gin.ErrorResponse` |
| 404 | Workspace 不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`workspace.DBConnection`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| account_name | string |  | - |
| database | string |  | - |
| effective_role_db_role | string |  | - |
| effective_role_id | string |  | - |
| effective_role_source | string |  | - |
| effective_role_version | string |  | - |
| host | string |  | - |
| password | string |  | - |
| port | integer |  | - |
| session_init_sqls | []string |  | - |
| username | string |  | - |

响应示例:

```json
{
  "account_name": "string",
  "database": "string",
  "effective_role_db_role": "string",
  "effective_role_id": "string",
  "effective_role_source": "string",
  "effective_role_version": "string",
  "host": "string",
  "password": "string",
  "port": 0,
  "session_init_sqls": ["string"],
  "username": "string"
}
```

---

### GET /api/v1/workspaces/{id}/system-roles

**获取 Workspace 系统角色引用 (仅系统用户)**

获取指定 Workspace 中 moi-core 初始化的系统角色引用，仅限系统用户调用

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 系统角色引用 | `workspace.WorkspaceSystemRoles` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 403 | 非系统用户 | `gin.ErrorResponse` |
| 404 | Workspace 不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |

响应字段 (`workspace.WorkspaceSystemRoles`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| admin_role | workspace.SystemRoleRef |  | - |
| superadmin_role | workspace.SystemRoleRef |  | - |

响应示例:

```json
{
  "admin_role": {
    "db_role_name": "string",
    "id": 0,
    "name": "string"
  },
  "superadmin_role": {
    "db_role_name": "string",
    "id": 0,
    "name": "string"
  }
}
```

---

### GET /api/v1/workspaces/{id}/users/{user_id}/db-connection

**获取指定用户数据库连接信息 (仅系统用户)**

获取指定 Workspace 中指定用户的数据库连接信息，仅限系统用户调用

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| user_id | string | 是 | 用户 ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 用户数据库连接信息 | `workspace.DBConnection` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 403 | 非系统用户或用户无权限 | `gin.ErrorResponse` |
| 404 | Workspace 不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |
| 503 | Workspace 正在初始化 | `gin.ErrorResponse` |

响应字段 (`workspace.DBConnection`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| account_name | string |  | - |
| database | string |  | - |
| effective_role_db_role | string |  | - |
| effective_role_id | string |  | - |
| effective_role_source | string |  | - |
| effective_role_version | string |  | - |
| host | string |  | - |
| password | string |  | - |
| port | integer |  | - |
| session_init_sqls | []string |  | - |
| username | string |  | - |

响应示例:

```json
{
  "account_name": "string",
  "database": "string",
  "effective_role_db_role": "string",
  "effective_role_id": "string",
  "effective_role_source": "string",
  "effective_role_version": "string",
  "host": "string",
  "password": "string",
  "port": 0,
  "session_init_sqls": ["string"],
  "username": "string"
}
```

---

### GET /api/v1/workspaces/{id}/users/{user_id}/owner-credential/api-key

**获取 Workspace OWNER API Key 元数据 (仅系统用户)**

获取指定 Workspace OWNER 的独立 admin API Key 元数据，仅限系统用户调用

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| user_id | string | 是 | Workspace OWNER 用户 ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | OWNER API Key 元数据 | `auth.APIKey` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 403 | 非系统用户或目标用户不是 OWNER | `gin.ErrorResponse` |
| 404 | Workspace 或 OWNER 凭据不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |
| 503 | OWNER 凭据正在初始化或初始化失败 | `gin.ErrorResponse` |

响应字段 (`auth.APIKey`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| created_at | integer |  | Unix 时间戳（秒） |
| expires_at | integer |  | 过期时间（Unix 时间戳，0 表示永不过期） |
| id | string |  | - |
| idempotency_key | string |  | 系统自动创建资源的稳定幂等键 |
| key_prefix | string |  | 用于显示，如 "moi_xxx..." |
| last_used_at | integer |  | 最后使用时间（Unix 时间戳） |
| name | string |  | - |
| scopes | []string |  | 权限范围 |
| uid | string |  | 所属用户 ID |

响应示例:

```json
{
  "created_at": 0,
  "expires_at": 0,
  "id": "string",
  "idempotency_key": "string",
  "key_prefix": "string",
  "last_used_at": 0,
  "name": "string",
  "scopes": ["string"],
  "uid": "string"
}
```

---

### POST /api/v1/workspaces/{id}/users/{user_id}/owner-credential/api-key

**显示 Workspace OWNER API Key (仅系统用户)**

显示指定 Workspace OWNER 的独立 admin API Key，仅限系统用户调用

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| user_id | string | 是 | Workspace OWNER 用户 ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | OWNER API Key 及密钥 | `auth.APIKeyWithSecret` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 403 | 非系统用户或目标用户不是 OWNER | `gin.ErrorResponse` |
| 404 | Workspace 或 OWNER 凭据不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |
| 503 | OWNER 凭据正在初始化或初始化失败 | `gin.ErrorResponse` |

响应字段 (`auth.APIKeyWithSecret`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| api_key | auth.APIKey |  | - |
| key | string |  | 完整的密钥，仅创建时返回一次 |

响应示例:

```json
{
  "api_key": {
    "created_at": 0,
    "expires_at": 0,
    "id": "string",
    "idempotency_key": "string",
    "key_prefix": "string",
    "last_used_at": 0,
    "name": "string",
    "scopes": ["string"],
    "uid": "string"
  },
  "key": "string"
}
```

---

### PUT /api/v1/workspaces/{id}/users/{user_id}/owner-credential/api-key

**轮换 Workspace OWNER API Key (仅系统用户)**

轮换指定 Workspace OWNER 的独立 admin API Key，仅限系统用户调用

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| user_id | string | 是 | Workspace OWNER 用户 ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 轮换后的 OWNER API Key 及密钥 | `auth.APIKeyWithSecret` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 403 | 非系统用户或目标用户不是 OWNER | `gin.ErrorResponse` |
| 404 | Workspace 或 OWNER 凭据不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |
| 503 | OWNER 凭据正在初始化或初始化失败 | `gin.ErrorResponse` |

响应字段 (`auth.APIKeyWithSecret`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| api_key | auth.APIKey |  | - |
| key | string |  | 完整的密钥，仅创建时返回一次 |

响应示例:

```json
{
  "api_key": {
    "created_at": 0,
    "expires_at": 0,
    "id": "string",
    "idempotency_key": "string",
    "key_prefix": "string",
    "last_used_at": 0,
    "name": "string",
    "scopes": ["string"],
    "uid": "string"
  },
  "key": "string"
}
```

---

### GET /api/v1/workspaces/{id}/users/{user_id}/owner-credential/db-connection

**获取 Workspace OWNER 数据库连接 (仅系统用户)**

获取指定 Workspace OWNER 的独立 admin 数据库连接，仅限系统用户调用

认证: 需要 API Key

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| user_id | string | 是 | Workspace OWNER 用户 ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | OWNER admin 数据库连接 | `workspace.DBConnection` |
| 400 | 参数错误 | `gin.ErrorResponse` |
| 401 | 未认证 | `gin.ErrorResponse` |
| 403 | 非系统用户或目标用户不是 OWNER | `gin.ErrorResponse` |
| 404 | Workspace 或 OWNER 凭据不存在 | `gin.ErrorResponse` |
| 500 | 内部错误 | `gin.ErrorResponse` |
| 503 | OWNER 凭据正在初始化或初始化失败 | `gin.ErrorResponse` |

响应字段 (`workspace.DBConnection`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| account_name | string |  | - |
| database | string |  | - |
| effective_role_db_role | string |  | - |
| effective_role_id | string |  | - |
| effective_role_source | string |  | - |
| effective_role_version | string |  | - |
| host | string |  | - |
| password | string |  | - |
| port | integer |  | - |
| session_init_sqls | []string |  | - |
| username | string |  | - |

响应示例:

```json
{
  "account_name": "string",
  "database": "string",
  "effective_role_db_role": "string",
  "effective_role_id": "string",
  "effective_role_source": "string",
  "effective_role_version": "string",
  "host": "string",
  "password": "string",
  "port": 0,
  "session_init_sqls": ["string"],
  "username": "string"
}
```

---

## agents

### POST /api/v1/agents/a2a

**Invoke an A2A agent method**

Invoke A2A JSON-RPC methods for a registered agent by agent_code or agent_id in request body.

#### 请求体

类型: `gin.agentJSONRPCRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | []integer |  | - |
| jsonrpc | string |  | - |
| method | string |  | - |
| params | []integer |  | - |

示例:

```json
{
  "id": ["string"],
  "jsonrpc": "string",
  "method": "string",
  "params": ["string"]
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | OK | `object` |
| 400 | Bad Request | `gin.ErrorResponse` |
| 500 | Internal Server Error | `gin.ErrorResponse` |

---

### GET /api/v1/agents/card

**Get agent card**

Get the A2A agent card by agent_code or agent_id query parameter.

#### 查询参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| agent_code | string | 否 | Agent code |
| agent_id | string | 否 | Agent ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | OK | `object` |
| 400 | Bad Request | `gin.ErrorResponse` |
| 500 | Internal Server Error | `gin.ErrorResponse` |

---

### POST /api/v1/mcp/http

**Invoke the MOI runtime MCP gateway**

Invoke runtime MCP JSON-RPC methods with a MOI RuntimeGrant bearer.

#### 请求体

类型: JSON 对象

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | OK | `object` |
| 400 | Bad Request | `object` |
| 401 | Unauthorized | `object` |
| 422 | Unprocessable Entity | `object` |
| 500 | Internal Server Error | `object` |
| 503 | Service Unavailable | `object` |

---

### POST /api/v1/models/openai/chat/completions

**Invoke runtime OpenAI-compatible chat completions**

Invoke the selected runtime model through the internal MOI OpenAI-compatible model gateway with a task-scoped RuntimeGrant bearer. The gateway verifies grant scope without a live IAM PEP.

#### 请求体

类型: JSON 对象

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | OK | `object` |
| 400 | Bad Request | `object` |
| 401 | Unauthorized | `object` |
| 500 | Internal Server Error | `object` |
| 503 | Service Unavailable | `object` |

---

### POST /api/v1/models/resolve

**Resolve a runtime model invocation descriptor**

Resolve the selected runtime model through the internal MOI model gateway with a task-scoped RuntimeGrant bearer. The gateway verifies grant scope without a live IAM PEP.

#### 请求体

类型: JSON 对象

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | OK | `object` |
| 400 | Bad Request | `object` |
| 401 | Unauthorized | `object` |
| 500 | Internal Server Error | `object` |
| 503 | Service Unavailable | `object` |

---

### GET /api/v1/query-visuals/{file_id}/content

**Read a short-lived query visual capability**

Serve a query visual to an external visual model with a short-lived signed URL.

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| file_id | string | 是 | Query visual file ID |

#### 查询参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| capability | string | 是 | Signed query visual capability |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | Image | `file` |
| 404 | Not Found | `object` |

---

### POST /api/v1/runtime-executors/authorize

**Reauthorize a runtime executor dispatch**

Revalidate the task actor, Effective Role, Agent use, and selected Runner or managed Sandbox use immediately before Astra dispatches a tool call.

#### 请求体

类型: `agentruntime.RuntimeExecutorDispatchAuthorizationRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| executor_id | string |  | - |
| run_id | string |  | - |
| task_id | string |  | - |
| tool_call_id | string |  | - |
| turn_chain_id | string |  | - |

示例:

```json
{
  "executor_id": "string",
  "run_id": "string",
  "task_id": "string",
  "tool_call_id": "string",
  "turn_chain_id": "string"
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 204 | No Content |  |
| 400 | Bad Request | `object` |
| 401 | Unauthorized | `object` |
| 403 | Forbidden | `object` |
| 503 | Service Unavailable | `object` |

---

### POST /api/v1/runtime-files

**Upload a runtime Catalog file**

Accept a bounded raw body from a managed Edge and publish it to the current user's private Catalog.

#### 查询参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| call_id | string | 是 | Runtime tool call ID |
| filename | string | 是 | File name |

#### 请求体

类型: `string`

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | OK | `object` |
| 400 | Bad Request | `object` |
| 401 | Unauthorized | `object` |
| 403 | Forbidden | `object` |
| 503 | Service Unavailable | `object` |

---

### POST /api/v1/skills/http

**Invoke the MOI runtime skill gateway**

Invoke runtime skill JSON-RPC discovery methods with a MOI RuntimeGrant bearer.

#### 请求体

类型: JSON 对象

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | OK | `object` |
| 400 | Bad Request | `object` |
| 401 | Unauthorized | `object` |
| 500 | Internal Server Error | `object` |
| 503 | Service Unavailable | `object` |

---

### GET /api/v1/workspaces/{id}/agents/{agent_id}/.well-known/agent-card.json

**Get agent card**

Get the A2A agent card for a workspace agent.

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| agent_id | string | 是 | Agent ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | OK | `object` |
| 400 | Bad Request | `github_com_matrixflow_moi-core_catalog_pkg_agentruntime_a2a.JSONRPCResponse` |
| 404 | Not Found | `github_com_matrixflow_moi-core_catalog_pkg_agentruntime_a2a.JSONRPCResponse` |
| 422 | Unprocessable Entity | `github_com_matrixflow_moi-core_catalog_pkg_agentruntime_a2a.JSONRPCResponse` |
| 500 | Internal Server Error | `github_com_matrixflow_moi-core_catalog_pkg_agentruntime_a2a.JSONRPCResponse` |

---

### POST /api/v1/workspaces/{id}/agents/{agent_id}/a2a

**Invoke an A2A agent method**

Invoke A2A JSON-RPC methods for a workspace agent, including streaming message calls.

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| agent_id | string | 是 | Agent ID |

#### 请求体

类型: JSON 对象

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | OK | `object` |
| 400 | Bad Request | `github_com_matrixflow_moi-core_catalog_pkg_agentruntime_a2a.JSONRPCResponse` |
| 404 | Not Found | `github_com_matrixflow_moi-core_catalog_pkg_agentruntime_a2a.JSONRPCResponse` |
| 422 | Unprocessable Entity | `github_com_matrixflow_moi-core_catalog_pkg_agentruntime_a2a.JSONRPCResponse` |
| 500 | Internal Server Error | `github_com_matrixflow_moi-core_catalog_pkg_agentruntime_a2a.JSONRPCResponse` |

---

### POST /api/v1/workspaces/{id}/agents/{agent_id}/mcp/http

**Invoke the MOI runtime MCP gateway**

Invoke runtime MCP JSON-RPC methods with a MOI RuntimeGrant bearer.

#### 请求体

类型: JSON 对象

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | OK | `object` |
| 400 | Bad Request | `object` |
| 401 | Unauthorized | `object` |
| 422 | Unprocessable Entity | `object` |
| 500 | Internal Server Error | `object` |
| 503 | Service Unavailable | `object` |

---

### GET /api/v1/workspaces/{id}/agents/{agent_id}/query-visuals/{file_id}/preview

**Preview an uploaded query visual**

Preview a query visual through the current agent.use admission. This endpoint does not require a Volume.

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| agent_id | string | 是 | Agent ID |
| file_id | string | 是 | Query visual file ID |

#### 查询参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| agent_workspace_id | string | 否 | Agent workspace ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | Image | `file` |
| 400 | Bad Request | `github_com_matrixflow_moi-core_catalog_pkg_agentruntime_a2a.JSONRPCResponse` |
| 403 | Forbidden | `github_com_matrixflow_moi-core_catalog_pkg_agentruntime_a2a.JSONRPCResponse` |
| 404 | Not Found | `object` |
| 503 | Service Unavailable | `object` |

---

## mowl

### GET /api/v1/workspaces/{id}/mowl/cases/{case_id}/trace

**Get workflow case trace**

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| case_id | string | 是 | Case ID |

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | OK | `object` |
| 400 | Bad Request | `object` |
| 401 | Unauthorized | `object` |
| 404 | Not Found | `object` |
| 500 | Internal Server Error | `object` |

---

## 健康检查

### GET /health

**健康检查**

返回服务整体健康状态，包括所有依赖项的检查结果

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 服务健康 | `health.HealthResponse` |
| 503 | 服务不健康 | `health.HealthResponse` |

响应字段 (`health.HealthResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| checks | object |  | Individual check results |
| status | string |  | Overall status |
| timestamp | string |  | ISO8601 timestamp |

响应示例:

```json
{
  "checks": {},
  "status": "string",
  "timestamp": "string"
}
```

---

### GET /health/live

**存活检查**

检查服务进程是否存活，只要进程能响应就返回 200

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 服务存活 | `health.HealthResponse` |

响应字段 (`health.HealthResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| checks | object |  | Individual check results |
| status | string |  | Overall status |
| timestamp | string |  | ISO8601 timestamp |

响应示例:

```json
{
  "checks": {},
  "status": "string",
  "timestamp": "string"
}
```

---

### GET /health/ready

**就绪检查**

检查服务是否准备好接受流量（数据库连接等依赖就绪）

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | 服务就绪 | `health.HealthResponse` |
| 503 | 服务未就绪 | `health.HealthResponse` |

响应字段 (`health.HealthResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| checks | object |  | Individual check results |
| status | string |  | Overall status |
| timestamp | string |  | ISO8601 timestamp |

响应示例:

```json
{
  "checks": {},
  "status": "string",
  "timestamp": "string"
}
```

---

## 其他

### POST /api/v1/system/catalogs/databases

**List catalog databases for a trusted Backend**

---

### POST /api/v1/system/catalogs/summaries

**List catalog summaries for a trusted Backend**

---

### POST /api/v1/system/databases/children

**List database children for a trusted Backend**

---

### POST /api/v1/system/iam/application-policy/apply

**Apply IAM application role policy**

---

### POST /api/v1/system/iam/application-policy/validate

**Validate IAM application role policy**

---

### POST /api/v1/system/iam/authorize

**Authorize an IAM action**

---

### POST /api/v1/system/iam/current-principal-access-projection

**Project current principal IAM access**

---

### POST /api/v1/system/iam/current-principal-roles

**List current principal IAM roles**

---

### POST /api/v1/system/iam/data-publishes/resolve-active

**Resolve active data publish IAM ownership**

---

### POST /api/v1/system/iam/data-subscriptions/revoke

**Revoke data subscription IAM ownership**

---

### POST /api/v1/system/iam/default-role/clear

**Clear a principal default IAM role**

---

### POST /api/v1/system/iam/default-role/set

**Set a principal default IAM role**

---

### POST /api/v1/system/iam/object-grants/mutate

**Mutate IAM object grants for one role**

---

### POST /api/v1/system/iam/object-permission-view

**Get IAM object permission view**

---

### POST /api/v1/system/iam/ownership-impact

**Get IAM role ownership impact**

---

### POST /api/v1/system/iam/permission-schema

**Get IAM permission schema**

---

### POST /api/v1/system/iam/principal-roles/assign

**Assign an IAM role to a principal**

---

### POST /api/v1/system/iam/principal-roles/unassign

**Unassign an IAM role from a principal**

---

### POST /api/v1/system/iam/resource-access-filter

**Filter accessible IAM resources**

---

### POST /api/v1/system/iam/resource-batch-describe

**Batch describe IAM permission resources**

---

### POST /api/v1/system/iam/resource-picker

**List IAM permission resources**

---

### POST /api/v1/system/iam/resources/delete

**Begin Backend-owned IAM resource deletion**

---

### POST /api/v1/system/iam/resources/delete/cancel

**Cancel Backend-owned IAM resource deletion**

---

### POST /api/v1/system/iam/resources/delete:finalize

**Finalize Backend-owned IAM resource deletion**

---

### POST /api/v1/system/iam/resources:register

**Register Backend-owned IAM resource**

---

### POST /api/v1/system/iam/role-audit-events

**List IAM role audit events**

---

### POST /api/v1/system/iam/role-create

**Create IAM application role**

---

### POST /api/v1/system/iam/role-members

**List IAM role members**

---

### POST /api/v1/system/iam/role-permission-view

**Get IAM role permission view**

---

### POST /api/v1/system/iam/role-provisioning

**Get IAM role provisioning state**

---

### POST /api/v1/system/iam/role-provisioning-recoveries

**List IAM role provisioning recoveries**

---

### POST /api/v1/system/iam/role-provisioning/retry

**Retry IAM role provisioning**

---

### POST /api/v1/system/iam/roles

**List IAM roles**

---

### POST /api/v1/system/iam/roles/create

**Create an IAM role**

---

### POST /api/v1/system/iam/roles/delete

**Delete an IAM role**

---

### POST /api/v1/system/iam/roles/delete-impact

**Inspect IAM role deletion impact**

---

### POST /api/v1/system/iam/roles/get

**Get an IAM role**

---

### POST /api/v1/system/iam/roles/inheritance/create

**Create IAM role inheritance**

---

### POST /api/v1/system/iam/roles/inheritance/delete

**Delete IAM role inheritance**

---

### POST /api/v1/system/iam/roles/inheritance/graph

**Get IAM role inheritance graph**

---

### POST /api/v1/system/iam/roles/inheritance/list

**List IAM role inheritance**

---

### POST /api/v1/system/iam/roles/lifecycle

**Update an IAM role lifecycle**

---

### POST /api/v1/system/iam/roles/policy/apply

**Apply an IAM role application policy**

---

### POST /api/v1/system/iam/roles/policy/get

**Get an IAM role application policy**

---

### POST /api/v1/system/iam/roles/update

**Update IAM role metadata**

---

### POST /api/v1/system/iam/set-default-role

**Set current principal default IAM role**

---

### POST /api/v1/system/iam/subject-attributes

**List IAM subject attribute definitions**

---

### POST /api/v1/system/iam/subject-attributes/create

**Create IAM subject attribute definition**

---

### POST /api/v1/system/iam/subject-attributes/delete

**Delete IAM subject attribute definition**

---

### POST /api/v1/system/iam/subject-attributes/update

**Update IAM subject attribute definition**

---

### POST /api/v1/system/iam/trusted-principal-access-projection

**Project trusted Backend principal IAM access**

---

### POST /api/v1/system/iam/user-role-bindings

**Get IAM user role bindings**

---

### POST /api/v1/system/iam/user-role-bindings/put

**Replace IAM user role bindings**

---

### POST /api/v1/system/iam/workspace-member-audit-events

**List IAM workspace member audit events**

---

### POST /api/v1/system/iam/workspace-members

**List IAM workspace members**

---

### POST /api/v1/system/iam/workspace-members/lifecycle

**Update IAM workspace member lifecycle**

---

### POST /api/v1/system/iam/workspace-members/profile

**Update IAM workspace member profile**

---

### POST /api/v1/system/iam/workspace-members/subject-attributes

**Replace IAM workspace member subject attributes**

---

### GET /api/v1/system/workspaces/{id}/llm/route

**解析 LLM 路由**

---

### GET /api/v1/system/workspaces/{id}/parsers/convert-route

**解析 Office Converter 路由**

---

### GET /api/v1/system/workspaces/{id}/parsers/route

**解析 Parser 路由**

---

### GET /api/v1/workspaces/{id}/agent-automation-runs

**List workspace agent automation runs**

---

### GET /api/v1/workspaces/{id}/agent-automation-runs/{run_id}

**Get agent automation run**

---

### GET /api/v1/workspaces/{id}/agent-automation-runs/{run_id}/events

**List agent automation run events**

---

### GET /api/v1/workspaces/{id}/agent-automation-runs/{run_id}/result

**Get agent automation run result**

---

### GET /api/v1/workspaces/{id}/agent-automation-tasks

**List agent automation tasks**

---

### POST /api/v1/workspaces/{id}/agent-automation-tasks

**Create agent automation task**

---

### GET /api/v1/workspaces/{id}/agent-automation-tasks/{automation_task_id}

**Get agent automation task**

---

### PATCH /api/v1/workspaces/{id}/agent-automation-tasks/{automation_task_id}

**Update agent automation task**

---

### DELETE /api/v1/workspaces/{id}/agent-automation-tasks/{automation_task_id}

**Archive agent automation task**

---

### GET /api/v1/workspaces/{id}/agent-automation-tasks/{automation_task_id}/runs

**List agent automation runs**

---

### POST /api/v1/workspaces/{id}/agent-automation-tasks/{automation_task_id}/runs

**Run agent automation task now**

---

### GET /api/v1/workspaces/{id}/agent-builder/agents/{agent_id}/versions/{version}/current-agent

**Get current Agent Builder editable config**

---

### POST /api/v1/workspaces/{id}/agent-builder/candidates

**Propose Agent Builder candidate**

---

### POST /api/v1/workspaces/{id}/agent-builder/candidates/{agent_id}/candidate-versions/{candidate_version}/cancel

**Cancel Agent Builder candidate**

---

### POST /api/v1/workspaces/{id}/agent-builder/candidates/{agent_id}/candidate-versions/{candidate_version}/commit

**Commit Agent Builder candidate**

---

### POST /api/v1/workspaces/{id}/agent-builder/candidates/{agent_id}/candidate-versions/{candidate_version}/repropose

**Repropose Agent Builder candidate**

---

### GET /api/v1/workspaces/{id}/agent-builder/resources

**List Agent Builder authoring resources**

---

### POST /api/v1/workspaces/{id}/agent-packages/load

**Load agent package**

---

### GET /api/v1/workspaces/{id}/agent-packages/{agent_id}/versions/{version}/export

**Export agent package**

---

### GET /api/v1/workspaces/{id}/agent-runtime-manifests/{manifest_id}

**Get agent runtime manifest**

---

### GET /api/v1/workspaces/{id}/agent-runtime-providers

**List agent runtime providers**

---

### GET /api/v1/workspaces/{id}/agent-runtime-providers/{provider_id}/profiles/{profile_id}

**Get agent runtime provider**

---

### GET /api/v1/workspaces/{id}/agent-runtime-tasks

**List agent runtime tasks**

---

### GET /api/v1/workspaces/{id}/agent-runtime-tasks/{task_id}

**Get agent runtime task**

---

### GET /api/v1/workspaces/{id}/agent-runtime-tasks/{task_id}/events

**List agent runtime task events**

---

### GET /api/v1/workspaces/{id}/agent-runtime-turn-snapshots/{snapshot_id}

**Get agent runtime turn snapshot**

---

### GET /api/v1/workspaces/{id}/agent-runtime/data-parts

**List agent runtime data parts**

---

### GET /api/v1/workspaces/{id}/agent-task-templates

**List agent task templates**

---

### POST /api/v1/workspaces/{id}/agent-task-templates

**Create agent task template**

---

### GET /api/v1/workspaces/{id}/agent-task-templates/{template_id}

**Get agent task template**

---

### PATCH /api/v1/workspaces/{id}/agent-task-templates/{template_id}

**Update agent task template**

---

### GET /api/v1/workspaces/{id}/agent-workflow-bindings

**List agent workflow bindings**

---

### POST /api/v1/workspaces/{id}/agent-workflow-bindings

**Create agent workflow binding**

---

### GET /api/v1/workspaces/{id}/agent-workflow-bindings/{binding_id}

**Get agent workflow binding**

---

### PATCH /api/v1/workspaces/{id}/agent-workflow-bindings/{binding_id}

**Update agent workflow binding**

---

### GET /api/v1/workspaces/{id}/agents

**List agent resources**

#### 查询参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| catalog | string | 否 | Definition catalog |

---

### POST /api/v1/workspaces/{id}/agents

**Create agent resource**

---

### GET /api/v1/workspaces/{id}/agents/{agent_id}

**Get agent resource**

---

### PATCH /api/v1/workspaces/{id}/agents/{agent_id}

**Update agent resource**

---

### DELETE /api/v1/workspaces/{id}/agents/{agent_id}

**Delete agent resource**

---

### GET /api/v1/workspaces/{id}/agents/{agent_id}/bindings

**Get agent bindings**

---

### PATCH /api/v1/workspaces/{id}/agents/{agent_id}/bindings

**Update agent bindings**

---

### PATCH /api/v1/workspaces/{id}/agents/{agent_id}/bindings/model

**Update agent model binding**

---

### GET /api/v1/workspaces/{id}/agents/{agent_id}/policies

**Get agent policies**

---

### PUT /api/v1/workspaces/{id}/agents/{agent_id}/policies

**Update agent policies**

---

### GET /api/v1/workspaces/{id}/agents/{agent_id}/versions

**List agent versions**

---

### DELETE /api/v1/workspaces/{id}/agents/{agent_id}/versions/{version}

**Delete agent version**

---

### POST /api/v1/workspaces/{id}/agents/{agent_id}/versions/{version}/default

**Set default agent version**

---

### POST /api/v1/workspaces/{id}/agents/{agent_id}/versions/{version}/disable

**Disable agent version**

---

### PUT /api/v1/workspaces/{id}/agents/{agent_id}/versions/{version}/runtime-bindings/{provider}/{profile}

**Upsert agent runtime binding**

---

### POST /api/v1/workspaces/{id}/agents/{agent_id}/versions/{version}/runtime-bindings/{provider}/{profile}/reconcile

**Reconcile agent runtime binding**

---

### GET /api/v1/workspaces/{id}/callback-rules

**List callback routing rules**

Lists dynamic callback forwarding rules for a workspace, optionally filtered by provider, credential_ref, and enabled status.

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 查询参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| provider | string | 否 | Callback provider, e.g. wecom |
| credential_ref | string | 否 | Provider credential ref |
| enabled | boolean | 否 | Filter by enabled status |
| limit | integer | 否 | Page size |
| offset | integer | 否 | Page offset |

---

### POST /api/v1/workspaces/{id}/callback-rules

**Create callback routing rule**

Creates a workspace/provider/credential scoped callback forwarding rule. Public callback URLs are not agent-scoped; target dispatch is configured here.

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |

#### 请求体

类型: `callbackrouting.RuleCreateInput`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| credential_ref | string |  | - |
| description | string |  | - |
| enabled | boolean |  | - |
| id | string |  | - |
| match | callbackrouting.RuleMatch |  | - |
| name | string |  | - |
| priority | integer |  | - |
| provider | string |  | - |
| stop_after_match | boolean |  | - |
| target_config | object |  | - |
| target_type | string |  | - |
| workspace_id | string |  | - |

示例:

```json
{
  "credential_ref": "string",
  "description": "string",
  "enabled": false,
  "id": "string",
  "match": {
    "event_actions": ["string"],
    "event_types": ["string"],
    "field_equals": {},
    "resource_kinds": ["string"]
  },
  "name": "string",
  "priority": 0,
  "provider": "string",
  "stop_after_match": false,
  "target_config": {},
  "target_type": "string",
  "workspace_id": "string"
}
```

---

### GET /api/v1/workspaces/{id}/callback-rules/{callback_rule_id}

**Get callback routing rule**

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| callback_rule_id | string | 是 | Callback rule ID |

---

### PATCH /api/v1/workspaces/{id}/callback-rules/{callback_rule_id}

**Update callback routing rule**

Updates a callback routing rule using optimistic locking through the version field.

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| callback_rule_id | string | 是 | Callback rule ID |

#### 请求体

类型: `callbackrouting.RuleUpdateInput`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| description | string |  | - |
| enabled | boolean |  | - |
| match | callbackrouting.RuleMatch |  | - |
| name | string |  | - |
| priority | integer |  | - |
| stop_after_match | boolean |  | - |
| target_config | object |  | - |
| target_type | string |  | - |
| version | integer |  | - |

示例:

```json
{
  "description": "string",
  "enabled": false,
  "match": {
    "event_actions": ["string"],
    "event_types": ["string"],
    "field_equals": {},
    "resource_kinds": ["string"]
  },
  "name": "string",
  "priority": 0,
  "stop_after_match": false,
  "target_config": {},
  "target_type": "string",
  "version": 0
}
```

---

### DELETE /api/v1/workspaces/{id}/callback-rules/{callback_rule_id}

**Delete callback routing rule**

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| callback_rule_id | string | 是 | Callback rule ID |

---

### GET /api/v1/workspaces/{id}/channels/{provider}/instances

**List channel instances**

---

### POST /api/v1/workspaces/{id}/channels/{provider}/instances

**Create channel instance**

#### 请求体

类型: `handlers.channelInstanceCreateRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| channel_type | string |  | - |
| config | object |  | - |
| description | string |  | - |
| labels | object |  | - |
| name | string |  | - |
| secrets | object |  | - |
| visibility | string |  | - |

示例:

```json
{
  "channel_type": "string",
  "config": {},
  "description": "string",
  "labels": {},
  "name": "string",
  "secrets": {},
  "visibility": "string"
}
```

---

### POST /api/v1/workspaces/{id}/channels/{provider}/instances/test

**Test channel configuration**

#### 请求体

类型: `handlers.channelInstanceCreateRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| channel_type | string |  | - |
| config | object |  | - |
| description | string |  | - |
| labels | object |  | - |
| name | string |  | - |
| secrets | object |  | - |
| visibility | string |  | - |

示例:

```json
{
  "channel_type": "string",
  "config": {},
  "description": "string",
  "labels": {},
  "name": "string",
  "secrets": {},
  "visibility": "string"
}
```

---

### GET /api/v1/workspaces/{id}/channels/{provider}/instances/{instance_id}

**Get channel instance**

---

### PATCH /api/v1/workspaces/{id}/channels/{provider}/instances/{instance_id}

**Update channel instance**

#### 请求体

类型: `handlers.channelInstanceUpdateRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| channel_type | string |  | - |
| config | object |  | - |
| description | string |  | - |
| expected_version | integer |  | - |
| labels | object |  | - |
| name | string |  | - |
| secrets | object |  | - |
| visibility | string |  | - |

示例:

```json
{
  "channel_type": "string",
  "config": {},
  "description": "string",
  "expected_version": 0,
  "labels": {},
  "name": "string",
  "secrets": {},
  "visibility": "string"
}
```

---

### DELETE /api/v1/workspaces/{id}/channels/{provider}/instances/{instance_id}

**Delete channel instance**

---

### POST /api/v1/workspaces/{id}/channels/{provider}/instances/{instance_id}/test

**Test channel instance**

---

### GET /api/v1/workspaces/{id}/connections

**List agent connections**

---

### POST /api/v1/workspaces/{id}/connections

**Create agent connection**

---

### GET /api/v1/workspaces/{id}/connections/{connection_id}

**Get agent connection**

---

### PATCH /api/v1/workspaces/{id}/connections/{connection_id}

**Update agent connection**

---

### GET /api/v1/workspaces/{id}/conversations

**List agent conversations**

---

### POST /api/v1/workspaces/{id}/conversations

**Create agent conversation**

---

### GET /api/v1/workspaces/{id}/conversations/{conversation_id}

**Get agent conversation**

---

### PATCH /api/v1/workspaces/{id}/conversations/{conversation_id}

**Update agent conversation**

---

### GET /api/v1/workspaces/{id}/conversations/{conversation_id}/messages

**List agent conversation messages**

---

### GET /api/v1/workspaces/{id}/data-dashboards

**查询数据仪表盘列表**

---

### POST /api/v1/workspaces/{id}/data-dashboards

**创建数据仪表盘**

---

### GET /api/v1/workspaces/{id}/data-dashboards/{dashboard_id}

**查询数据仪表盘详情**

---

### PATCH /api/v1/workspaces/{id}/data-dashboards/{dashboard_id}

**更新数据仪表盘**

---

### DELETE /api/v1/workspaces/{id}/data-dashboards/{dashboard_id}

**完成删除数据仪表盘**

---

### POST /api/v1/workspaces/{id}/data-dashboards/{dashboard_id}/charts

**创建数据仪表盘图表**

---

### PATCH /api/v1/workspaces/{id}/data-dashboards/{dashboard_id}/charts/{chart_id}

**更新数据仪表盘图表**

---

### DELETE /api/v1/workspaces/{id}/data-dashboards/{dashboard_id}/charts/{chart_id}

**完成删除数据仪表盘图表**

---

### POST /api/v1/workspaces/{id}/data-dashboards/{dashboard_id}/charts/{chart_id}/delete-operation

**开始删除数据仪表盘图表**

---

### POST /api/v1/workspaces/{id}/data-dashboards/{dashboard_id}/charts/{chart_id}/evaluate-alert

**评估数据仪表盘图表告警**

---

### PATCH /api/v1/workspaces/{id}/data-dashboards/{dashboard_id}/charts/{chart_id}/schedule-state

**完成图表调度状态**

---

### POST /api/v1/workspaces/{id}/data-dashboards/{dashboard_id}/delete-operation

**开始删除数据仪表盘**

---

### GET /api/v1/workspaces/{id}/data-dashboards/{dashboard_id}/execution-spec

**查询数据仪表盘执行配置**

---

### GET /api/v1/workspaces/{id}/data-dashboards/{dashboard_id}/refresh-plan

**查询仪表盘刷新计划**

---

### POST /api/v1/workspaces/{id}/data-dashboards/{dashboard_id}/sql-draft

**生成数据仪表盘 SQL 草稿**

#### 请求体

类型: `handlers.DataDashboardSQLDraftRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| previous_sql | string |  | - |
| question | string |  | - |
| request_id | string |  | - |
| schema | []handlers.DataDashboardTableSchema |  | - |
| title | string |  | - |
| validation_error | string |  | - |

示例:

```json
{
  "previous_sql": "string",
  "question": "string",
  "request_id": "string",
  "schema": [{
    "columns": [{
      "comment": "string",
      "distinct_ratio": 0,
      "is_primary_key": false,
      "name": "string",
      "nullable": false,
      "population_score": 0,
      "primary_key": false,
      "type": "string"
    }],
    "ddl": "string",
    "name": "string"
  }],
  "title": "string",
  "validation_error": "string"
}
```

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | OK | `handlers.DataDashboardSQLDraftResult` |

响应字段 (`handlers.DataDashboardSQLDraftResult`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| chart_type | string |  | - |
| dashboard_id | string |  | - |
| sql_text | string |  | - |

响应示例:

```json
{
  "chart_type": "string",
  "dashboard_id": "string",
  "sql_text": "string"
}
```

---

### GET /api/v1/workspaces/{id}/feedback

**List agent runtime feedback**

---

### GET /api/v1/workspaces/{id}/feedback/stats

**Get agent runtime feedback stats**

---

### GET /api/v1/workspaces/{id}/knowledge-bases

**List agent knowledge bases**

---

### POST /api/v1/workspaces/{id}/knowledge-bases

**Create agent knowledge base**

---

### GET /api/v1/workspaces/{id}/knowledge-bases/{knowledge_base_id}

**Get agent knowledge base**

---

### PATCH /api/v1/workspaces/{id}/knowledge-bases/{knowledge_base_id}

**Update agent knowledge base**

---

### GET /api/v1/workspaces/{id}/model-configs

**List agent model configs**

---

### POST /api/v1/workspaces/{id}/model-configs

**Create agent model config**

---

### GET /api/v1/workspaces/{id}/model-configs/{model_config_id}

**Get agent model config**

---

### PATCH /api/v1/workspaces/{id}/model-configs/{model_config_id}

**Update agent model config**

---

### GET /api/v1/workspaces/{id}/operations

**List agent resource operations**

---

### GET /api/v1/workspaces/{id}/operations/{operation_id}

**Get agent resource operation**

---

### POST /api/v1/workspaces/{id}/operations/{operation_id}/cancel

**Cancel agent resource operation**

---

### GET /api/v1/workspaces/{id}/parsers/backends

**获取 Parser 后端列表**

获取指定 workspace 中所有 Parser 后端配置。需要 PERM_MODEL_RESOURCE_READ（兼容旧 PERM_PARSER_INVOKE）或 workspace 管理员权限；响应会脱敏 api_key_encrypted。

---

### POST /api/v1/workspaces/{id}/parsers/backends

**创建 Parser 后端**

---

### GET /api/v1/workspaces/{id}/parsers/backends/{backend_id}

**获取 Parser 后端详情**

根据后端 ID 获取指定 Parser 后端的详细信息。需要 PERM_MODEL_RESOURCE_READ（兼容旧 PERM_PARSER_INVOKE）或 workspace 管理员权限；响应会脱敏 api_key_encrypted。

---

### PUT /api/v1/workspaces/{id}/parsers/backends/{backend_id}

**更新 Parser 后端**

---

### DELETE /api/v1/workspaces/{id}/parsers/backends/{backend_id}

**删除 Parser 后端**

---

### GET /api/v1/workspaces/{id}/parsers/backends/{backend_id}/endpoints

**列出 Parser 后端端点**

列出指定后端下的所有 Parser 服务端点。需要 PERM_MODEL_RESOURCE_READ（兼容旧 PERM_PARSER_INVOKE）或 workspace 管理员权限。

---

### POST /api/v1/workspaces/{id}/parsers/backends/{backend_id}/endpoints

**创建 Parser 端点**

---

### PUT /api/v1/workspaces/{id}/parsers/backends/{backend_id}/endpoints/{endpoint_id}/status

**设置 Parser 端点状态**

---

### POST /api/v1/workspaces/{id}/parsers/convert

**文档格式转换**

---

### POST /api/v1/workspaces/{id}/parsers/parse

**执行文件解析**

---

### GET /api/v1/workspaces/{id}/parsers/router-config

**获取 Parser 路由配置**

---

### PUT /api/v1/workspaces/{id}/parsers/router-config

**更新 Parser 路由配置**

---

### GET /api/v1/workspaces/{id}/runtime-policy-profiles

**List agent runtime policy profiles**

---

### POST /api/v1/workspaces/{id}/runtime-policy-profiles

**Create agent runtime policy profile**

---

### GET /api/v1/workspaces/{id}/runtime-policy-profiles/{policy_id}

**Get agent runtime policy profile**

---

### PATCH /api/v1/workspaces/{id}/runtime-policy-profiles/{policy_id}

**Update agent runtime policy profile**

---

### GET /api/v1/workspaces/{id}/skills

**List agent skills**

Lists workspace skills and read-only system default skills. System default skills keep workspace_id=system and source_type=system so clients can keep them non-editable.

---

### POST /api/v1/workspaces/{id}/skills

**Create agent skill**

---

### POST /api/v1/workspaces/{id}/skills/import

**Import agent skill package**

---

### POST /api/v1/workspaces/{id}/skills/import/inspect

**Inspect agent skill package**

---

### POST /api/v1/workspaces/{id}/skills/polish/stream

**Stream agent skill draft polishing**

Accepts visible draft fields name, description, instruction, input_requirements, output_format, and tags; protected resource metadata is not sent to the model. Emits started {request_id,prompt_version}, delta {request_id,content}, ping {request_id,timestamp}, result {request_id,result} or error {request_id,error:{code,message,retryable}}, then done {request_id}. Only result contains the validated skill; delta is preview text. Error codes: INVALID_DRAFT, NO_AVAILABLE_MODEL, MODEL_REQUEST_FAILED, MODEL_OUTPUT_EMPTY, MODEL_OUTPUT_INVALID, MODEL_OUTPUT_UNSAFE, STREAM_INTERRUPTED, INTERNAL_ERROR.

---

### GET /api/v1/workspaces/{id}/skills/tags

**List agent skill tags**

---

### GET /api/v1/workspaces/{id}/skills/{skill_id}

**Get agent skill**

---

### PATCH /api/v1/workspaces/{id}/skills/{skill_id}

**Update agent skill**

---

### POST /api/v1/workspaces/{id}/skills/{skill_id}/execute

**Execute agent skill**

---

### GET /api/v1/workspaces/{id}/skills/{skill_id}/files

**List agent skill files**

---

### GET /api/v1/workspaces/{id}/skills/{skill_id}/files/content

**Download agent skill file**

---

### GET /api/v1/workspaces/{id}/skills/{skill_id}/referencing-agents

**List agents referencing an agent skill**

---

### GET /api/v1/workspaces/{id}/skills/{skill_id}/versions

**List agent skill versions**

---

### POST /api/v1/workspaces/{id}/skills/{skill_id}/versions/{version}/current

**Set current agent skill version**

---

### GET /api/v1/workspaces/{id}/system-agent-setups/{agent_id}

**Get system agent setup**

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| agent_id | string | 是 | System agent ID |

---

### PUT /api/v1/workspaces/{id}/system-agent-setups/{agent_id}

**Update system agent setup**

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| agent_id | string | 是 | System agent ID |

#### 请求体

类型: `systemagentsetup.UpdateInput`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| channels | object |  | - |
| enabled | boolean |  | - |
| knowledge_bases | object |  | - |
| model | agentresource.AgentModelConfig |  | - |
| project_scope | githubprojectscope.Scope |  | - |
| reconciliation_task | systemagentsetup.ReconciliationTaskConfig |  | - |
| suites | []opssuites.Suite |  | Suites is required when AgentDefinition.SuiteScope is set (Ops Agent).
Each suite must declare a Kubernetes namespace (and optional log_namespace). |

示例:

```json
{
  "channels": {},
  "enabled": false,
  "knowledge_bases": {},
  "model": {
    "default_model": "string",
    "model_config_ref": "string",
    "params_override": {}
  },
  "project_scope": {
    "host": "string",
    "number": 0,
    "owner": "string",
    "owner_type": "string",
    "priority_field_id": "string",
    "project_id": "string",
    "status_field_id": "string",
    "title": "string",
    "type_field_id": "string",
    "url": "string"
  },
  "reconciliation_task": {
    "cron_expression": "string"
  },
  "suites": [{
    "containers": ["string"],
    "display_name": "string",
    "enabled": false,
    "log_namespace": "string",
    "namespace": "string",
    "suite_id": "string"
  }]
}
```

---

### GET /api/v1/workspaces/{id}/system-agent-setups/{agent_id}/github-projects

**List system agent GitHub Projects**

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| agent_id | string | 是 | System agent ID |

#### 查询参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| channel_instance_id | string | 是 | GitHub Channel instance ID |

---

### POST /api/v1/workspaces/{id}/system-agent-setups/{agent_id}/github-projects

**List system agent GitHub Projects from channel form**

#### 路径参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | Workspace ID |
| agent_id | string | 是 | System agent ID |

#### 请求体

类型: `systemagentsetup.ChannelCreateInput`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| config | object |  | - |
| description | string |  | - |
| name | string |  | - |
| secrets | object |  | - |

示例:

```json
{
  "config": {},
  "description": "string",
  "name": "string",
  "secrets": {}
}
```

---

### GET /api/v1/workspaces/{id}/tools

**List agent tools**

Lists tools with real limit/offset pagination. By default only tools owned by the path workspace are returned. Pass catalog=system to list read-only system default tools (workspace_id=system) while still authorizing against the path workspace. Catalogs are intentionally not merged; clients that need both should issue two requests. Tools with i18n metadata include localized display_name and display_description. The locale is resolved from Accept-Language first, then Content-Language; the first supported zh/en language range is used, otherwise en-US is used. The stable name and description fields remain unchanged. List responses omit input_schema/output_schema by default so the tool library first paint stays small; pass include_schema=true or view=full when callers need schemas in the list payload. GET tool detail always returns full schemas.

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | Agent tool list | `handlers.agentToolListResponse` |

响应字段 (`handlers.agentToolListResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| code | integer |  | - |
| data | handlers.agentToolListDataResponse |  | - |
| msg | string |  | - |

响应示例:

```json
{
  "code": 0,
  "data": {
    "items": [{
      "annotations": {},
      "approval_policy_ref": "string",
      "bindability_reason": "string",
      "bindable": false,
      "category": "string",
      "created_at": "string",
      "created_by": "string",
      "credential_ref": "string",
      "description": "string",
      "display_description": "string",
      "display_name": "string",
      "icon_ref": "string",
      "id": "string",
      "input_schema": {},
      "kind": "string",
      "labels": {},
      "market_metadata": {},
      "metadata": {},
      "name": "string",
      "output_schema": {},
      "phase": "string",
      "redaction_policy_ref": "string",
      "side_effect_class": "string",
      "source_ref": {
        "config": {},
        "id": "string",
        "type": "string",
        "uri": "string",
        "version": "string"
      },
      "status": "string",
      "supported_runtimes": ["string"],
      "sync": {
        "last_sync_at": "string",
        "last_sync_error": "string",
        "status": "string"
      },
      "tags": ["string"],
      "updated_at": "string",
      "updated_by": "string",
      "version": 0,
      "workspace_id": "string"
    }],
    "limit": 0,
    "offset": 0,
    "total": 0
  },
  "msg": "string"
}
```

---

### POST /api/v1/workspaces/{id}/tools

**Create agent tool**

---

### GET /api/v1/workspaces/{id}/tools/github/connect

**Get built-in GitHub tool connection status**

---

### POST /api/v1/workspaces/{id}/tools/github/connect

**Connect built-in GitHub tool**

---

### DELETE /api/v1/workspaces/{id}/tools/github/connect

**Disconnect built-in GitHub tool**

---

### GET /api/v1/workspaces/{id}/tools/grafana/connect

**Get built-in Grafana tool connection status**

---

### POST /api/v1/workspaces/{id}/tools/grafana/connect

**Connect built-in Grafana tool**

---

### DELETE /api/v1/workspaces/{id}/tools/grafana/connect

**Disconnect built-in Grafana tool**

---

### GET /api/v1/workspaces/{id}/tools/mail/{provider}/connect

**Get built-in mail tool connection status**

---

### POST /api/v1/workspaces/{id}/tools/mail/{provider}/connect

**Connect built-in mail tool**

---

### DELETE /api/v1/workspaces/{id}/tools/mail/{provider}/connect

**Disconnect built-in mail tool**

---

### GET /api/v1/workspaces/{id}/tools/tags

**List agent tool tags**

---

### POST /api/v1/workspaces/{id}/tools/wecom/callback-secrets/generate

**Generate WeCom callback secrets**

---

### GET /api/v1/workspaces/{id}/tools/{tool_id}

**Get agent tool**

Gets an agent tool. Tools with i18n metadata include localized display_name and display_description. The locale is resolved from Accept-Language first, then Content-Language; the first supported zh/en language range is used, otherwise en-US is used. The stable name and description fields remain unchanged.

#### 响应

| 状态码 | 说明 | 类型 |
|--------|------|------|
| 200 | Agent tool detail | `handlers.agentToolGetResponse` |

响应字段 (`handlers.agentToolGetResponse`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| code | integer |  | - |
| data | handlers.agentToolResponse |  | - |
| msg | string |  | - |

响应示例:

```json
{
  "code": 0,
  "data": {
    "annotations": {},
    "approval_policy_ref": "string",
    "bindability_reason": "string",
    "bindable": false,
    "category": "string",
    "created_at": "string",
    "created_by": "string",
    "credential_ref": "string",
    "description": "string",
    "display_description": "string",
    "display_name": "string",
    "icon_ref": "string",
    "id": "string",
    "input_schema": {},
    "kind": "string",
    "labels": {},
    "market_metadata": {},
    "metadata": {},
    "name": "string",
    "output_schema": {},
    "phase": "string",
    "redaction_policy_ref": "string",
    "side_effect_class": "string",
    "source_ref": {
      "config": {},
      "id": "string",
      "type": "string",
      "uri": "string",
      "version": "string"
    },
    "status": "string",
    "supported_runtimes": ["string"],
    "sync": {
      "last_sync_at": "string",
      "last_sync_error": "string",
      "status": "string"
    },
    "tags": ["string"],
    "updated_at": "string",
    "updated_by": "string",
    "version": 0,
    "workspace_id": "string"
  },
  "msg": "string"
}
```

---

### PATCH /api/v1/workspaces/{id}/tools/{tool_id}

**Update agent tool**

---

### GET /api/v1/workspaces/{id}/tools/{tool_id}/referencing-agents

**List agents referencing an agent tool**

---
