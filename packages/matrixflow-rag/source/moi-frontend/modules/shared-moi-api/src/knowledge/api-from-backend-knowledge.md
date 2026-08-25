# 知识库管理接口

> Controller: `knowledge.go` | Service: `pkg/session/knowledge_service.go`

基于 moi-core 封装的 RESTful 风格知识库管理接口，提供完整的 CRUD 功能。

**路由前缀**: `/newmoi/knowledge_base`

---

## 目录

- [统一响应格式](#统一响应格式)
- [错误码](#错误码)
- [数据类型](#数据类型)
- [接口列表](#接口列表)
  - [查询列表](#post-newmoiknowledge_baselist)
  - [创建知识库](#post-newmoiknowledge_base)
  - [更新知识库](#put-newmoiknowledge_baseid)
  - [删除知识库](#delete-newmoiknowledge_baseid)
  - [获取详情](#get-newmoiknowledge_baseid)
- [使用说明](#使用说明)

---

## 统一响应格式

所有接口返回统一的响应格式：

```typescript
interface ApiResponse<T> {
  code: string; // 状态码（语义常量）
  msg: string; // 描述信息
  data: T; // 业务数据
}
```

**成功响应示例**:

```json
{
  "code": "OK",
  "msg": "OK",
  "data": { "id": 1, "name": "知识库名称" }
}
```

**错误响应示例**:

```json
{
  "code": "ErrParamInvalid",
  "msg": "name is required",
  "data": null
}
```

---

## 错误码

### 成功

| 错误码 | 说明     |
| ------ | -------- |
| `OK`   | 请求成功 |

### 客户端错误

| 错误码            | HTTP 状态码 | 说明       |
| ----------------- | ----------- | ---------- |
| `ErrParamInvalid` | 400         | 参数错误   |
| `ErrUnauthorized` | 401         | 未认证     |
| `ErrForbidden`    | 403         | 无权限     |
| `ErrNotFound`     | 404         | 资源不存在 |

### 服务端错误

| 错误码      | HTTP 状态码 | 说明           |
| ----------- | ----------- | -------------- |
| `ErrServer` | 500         | 服务器内部错误 |

---

## 数据类型

### 基础类型

```typescript
// 知识库文件配置
interface KnowledgeFileSpec {
  files_id: string[]; // 文件 ID 列表
  parent: string[]; // 父级路径
}

// 知识库表配置
interface KnowledgeTableSpec {
  db_name: string; // 数据库名称
  table_name: string[]; // 表名列表（必填，WorkItem 使用）
  table_ids?: string[]; // 表 ID 列表（可选，前端回显使用）
  parent: string[]; // 父级路径
}
```

### 使用说明

> NL2SQL 语义配置已迁移至语义模型接口，详见 [semantic_model.md](./semantic_model.md)

---

## 接口列表

### POST /newmoi/knowledge_base/list

查询知识库列表，支持分页。

**权限**: `K2`

#### 请求参数

```typescript
interface QueryKnowledgeListRequest {
  page: number; // 页码（从 1 开始）
  page_size: number; // 每页数量
  offset_num?: number; // 偏移量调整（可选，默认 0）
}
```

**请求示例**:

```json
{
  "page": 1,
  "page_size": 20,
  "offset_num": 0
}
```

#### 响应参数

```typescript
interface QueryKnowledgeListResponse {
  page: number;
  page_size: number;
  total: number;
  list: KnowledgeListItem[];
}

interface KnowledgeListItem {
  id: number;
  name: string;
  usage_notes: string;
  files: KnowledgeFileSpec;
  tables: KnowledgeTableSpec[];
  first_file_name: string; // 第一个文件名（用于预览）
}
```

**响应示例**:

```json
{
  "code": "OK",
  "msg": "OK",
  "data": {
    "page": 1,
    "page_size": 20,
    "total": 5,
    "list": [
      {
        "id": 1,
        "name": "销售数据知识库",
        "usage_notes": "用于销售数据分析",
        "files": {
          "files_id": ["file_001"],
          "parent": []
        },
        "tables": [
          {
            "db_name": "sales_db",
            "table_name": ["orders", "customers"],
            "table_ids": ["101", "102"],
            "parent": []
          }
        ],
        "first_file_name": "file_001"
      }
    ]
  }
}
```

---

### POST /newmoi/knowledge_base

创建新的知识库。

**权限**: `K1`

#### 请求参数

```typescript
interface CreateKnowledgeRequest {
  name: string; // 知识库名称（必填）
  usage_notes: string; // 使用说明
  files?: KnowledgeFileSpec; // 文件配置（可选）
  tables: KnowledgeTableSpec[]; // 表配置
}
```

**请求示例**:

```json
{
  "name": "销售数据知识库",
  "usage_notes": "用于销售数据分析和报表生成",
  "tables": [
    {
      "db_name": "sales_db",
      "table_name": ["orders", "customers", "products"],
      "table_ids": ["101", "102", "103"],
      "parent": []
    }
  ],
  "files": {
    "files_id": ["file_001", "file_002"],
    "parent": []
  }
}
```

#### 响应参数

```typescript
interface CreateKnowledgeResponse {
  id: number; // 知识库 ID
  success: boolean; // 是否成功
}
```

**成功响应示例**:

```json
{
  "code": "OK",
  "msg": "OK",
  "data": {
    "id": 1,
    "success": true
  }
}
```

**错误响应示例**:

```json
{
  "code": "ErrParamInvalid",
  "msg": "name is required",
  "data": null
}
```

---

### PUT /newmoi/knowledge_base/:id

更新知识库的基本信息。

**权限**: `K3`

#### 请求参数

路径参数 `:id` — 知识库 ID（整数）

```typescript
interface UpdateKnowledgeRequest {
  name: string; // 知识库名称
  usage_notes: string; // 使用说明
  files?: KnowledgeFileSpec; // 文件配置（可选）
  tables: KnowledgeTableSpec[]; // 表配置
}
```

**请求示例**:

```
PUT /newmoi/knowledge_base/1
```

```json
{
  "name": "销售数据知识库（已更新）",
  "usage_notes": "更新后的使用说明",
  "tables": [
    {
      "db_name": "sales_db",
      "table_name": ["orders", "customers", "products", "invoices"],
      "table_ids": ["101", "102", "103", "104"],
      "parent": []
    }
  ],
  "files": {
    "files_id": ["file_001", "file_002", "file_003"],
    "parent": []
  }
}
```

#### 响应参数

```typescript
interface UpdateKnowledgeResponse {
  success: boolean;
}
```

**响应示例**:

```json
{
  "code": "OK",
  "msg": "OK",
  "data": {
    "success": true
  }
}
```

---

### DELETE /newmoi/knowledge_base/:id

删除指定的知识库及其关联的语义模型。

**权限**: `K4`

#### 请求参数

路径参数 `:id` — 知识库 ID（整数）

**请求示例**:

```
DELETE /newmoi/knowledge_base/1
```

#### 响应参数

```typescript
interface DeleteKnowledgeResponse {
  success: boolean;
}
```

**响应示例**:

```json
{
  "code": "OK",
  "msg": "OK",
  "data": {
    "success": true
  }
}
```

---

### GET /newmoi/knowledge_base/:id

根据 ID 获取知识库详细信息。

**权限**: `K5`

#### 请求参数

路径参数 `:id` — 知识库 ID（整数）

**请求示例**:

```
GET /newmoi/knowledge_base/1
```

#### 响应参数

```typescript
interface GetKnowledgeDetailResponse {
  id: number;
  name: string;
  usage_notes: string;
  files: KnowledgeFileSpec;
  tables: KnowledgeTableSpec[];
}
```

**成功响应示例**:

```json
{
  "code": "OK",
  "msg": "OK",
  "data": {
    "id": 1,
    "name": "销售数据知识库",
    "usage_notes": "用于销售数据分析和报表生成",
    "files": {
      "files_id": ["file_001", "file_002"],
      "parent": []
    },
    "tables": [
      {
        "db_name": "sales_db",
        "table_name": ["orders", "customers", "products"],
        "table_ids": ["101", "102", "103"],
        "parent": []
      }
    ]
  }
}
```

**错误响应示例**:

```json
{
  "code": "ErrNotFound",
  "msg": "knowledge base not found",
  "data": null
}
```

---

## 使用说明

### 1. table_ids 和 table_names 的关系

- **table_names**: 必填字段，WorkItem 使用此字段进行表名匹配
- **table_ids**: 可选字段，前端回显使用，避免二次查询数据库
- 两者不是强制一一对应关系，但建议同时提供以提升用户体验

**示例**:

```typescript
// 只提供 table_names（最小配置）
{
  "db_name": "sales_db",
  "table_name": ["orders", "customers"],
  "parent": []
}

// 同时提供 table_ids（推荐配置）
{
  "db_name": "sales_db",
  "table_name": ["orders", "customers"],
  "table_ids": ["101", "102"],  // 前端可直接使用，无需查询
  "parent": []
}
```

### 2. 语义模型同步

知识库创建/更新/删除时会自动同步关联的语义模型：

- ✅ 创建知识库时，如果配置了 tables，自动创建对应的 semantic model
- ✅ 更新知识库 tables 时，自动同步更新 semantic model 的 tables
- ✅ 删除知识库时，自动删除关联的 semantic model（级联删除所有 entries）

语义配置的详细管理请参考 [semantic_model.md](./semantic_model.md)。

### 3. 权限配置

所有接口需要对应的权限码：

| 权限码 | 说明           | 接口          |
| ------ | -------------- | ------------- |
| `K1`   | 创建知识库     | `POST /`      |
| `K2`   | 查询知识库列表 | `POST /list`  |
| `K3`   | 更新知识库     | `PUT /:id`    |
| `K4`   | 删除知识库     | `DELETE /:id` |
| `K5`   | 获取知识库详情 | `GET /:id`    |

---

**文档版本**: v1.0  
**最后更新**: 2026-04-04
