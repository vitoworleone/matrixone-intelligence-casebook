# 表概览接口

> Controller: `table_overview.go` | Service: `pkg/session/knowledge_advanced_service.go`

提供数据库表和字段概览功能，用于前端展示可用的数据表结构。

**路由前缀**: `/newmoi/table`

**推荐使用**:

- 单表查询：`POST /newmoi/table/columns` - 获取指定表的列信息
- 数据库级查询：`POST /newmoi/table/database/tables` - 获取指定数据库下所有表的列信息
- 全量查询（不推荐）：`POST /newmoi/table/overview` - 获取所有表的列信息（性能较差）

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
  "data": { ... }
}
```

**错误响应示例**:

```json
{
  "code": "ErrServer",
  "msg": "failed to query database",
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

| 错误码            | HTTP 状态码 | 说明     |
| ----------------- | ----------- | -------- |
| `ErrParamInvalid` | 400         | 参数错误 |
| `ErrUnauthorized` | 401         | 未认证   |
| `ErrForbidden`    | 403         | 无权限   |

### 服务端错误

| 错误码      | HTTP 状态码 | 说明           |
| ----------- | ----------- | -------------- |
| `ErrServer` | 500         | 服务器内部错误 |

---

## 数据类型

```typescript
// 表概览信息
interface TableOverview {
  db_name: string; // 数据库名称
  table_name: string; // 表名
  col_names: string[]; // 列名列表
}

// 获取单表列信息请求
interface GetTableColumnsRequest {
  database_id: number; // 数据库 ID（必填）
  table_id?: number; // 表 ID（可选，与 table_name 二选一）
  table_name?: string; // 表名（可选，与 table_id 二选一，优先使用）
}

// 获取单表列信息响应
interface GetTableColumnsResponse {
  db_name: string; // 数据库名称
  table_name: string; // 表名
  col_names: string[]; // 列名列表
}

// 获取数据库所有表请求
interface ListDatabaseTablesRequest {
  database_id: number; // 数据库 ID
}

// 获取数据库所有表响应
interface ListDatabaseTablesResponse {
  tables: TableOverview[]; // 表列表
}
```

---

## 接口详情

### POST /newmoi/table/columns

获取指定表的列信息（推荐使用）。

#### 请求参数

```typescript
interface GetTableColumnsRequest {
  database_id: number; // 数据库 ID（必填）
  table_id?: number; // 表 ID（可选）
  table_name?: string; // 表名（可选，优先使用）
}
```

**请求示例**:

```json
{
  "database_id": 10001,
  "table_name": "users"
}
```

或

```json
{
  "database_id": 10001,
  "table_id": 20001
}
```

#### 响应参数

```typescript
interface GetTableColumnsResponse {
  db_name: string;
  table_name: string;
  col_names: string[];
}
```

**响应示例**:

```json
{
  "code": "OK",
  "msg": "OK",
  "data": {
    "db_name": "moi",
    "table_name": "users",
    "col_names": ["id", "name", "email", "created_at"]
  }
}
```

---

### POST /newmoi/table/database/tables

获取指定数据库下所有表的列信息（推荐使用）。

#### 请求参数

```typescript
interface ListDatabaseTablesRequest {
  database_id: number; // 数据库 ID
}
```

**请求示例**:

```json
{
  "database_id": 10001
}
```

#### 响应参数

```typescript
interface ListDatabaseTablesResponse {
  tables: TableOverview[];
}
```

**响应示例**:

```json
{
  "code": "OK",
  "msg": "OK",
  "data": {
    "tables": [
      {
        "db_name": "moi",
        "table_name": "users",
        "col_names": ["id", "name", "email", "created_at"]
      },
      {
        "db_name": "moi",
        "table_name": "orders",
        "col_names": ["id", "user_id", "amount", "status", "created_at"]
      }
    ]
  }
}
```

---

### POST /newmoi/table/overview

获取所有数据库的所有表和字段概览信息（不推荐使用，性能较差）。

**性能警告**: 此接口会遍历所有 catalog → database → table，对每个表执行 DESC 查询，性能开销大。建议使用 `/table/columns` 或 `/table/database/tables` 按需查询。

#### 请求参数

无请求体。

#### 响应参数

```typescript
interface TableOverviewResponse {
  tables: TableOverview[];
}
```

**响应示例**:

```json
{
  "code": "OK",
  "msg": "OK",
  "data": {
    "tables": [
      {
        "db_name": "moi",
        "table_name": "users",
        "col_names": ["id", "name", "email", "created_at"]
      },
      {
        "db_name": "moi",
        "table_name": "orders",
        "col_names": ["id", "user_id", "amount", "status", "created_at"]
      },
      {
        "db_name": "sales_db",
        "table_name": "products",
        "col_names": ["id", "name", "price", "stock"]
      }
    ]
  }
}
```

---

## 性能对比

假设有 2 个 catalog，10 个 database，100 个 table：

| 接口                     | HTTP 请求 | 数据库查询次数 | 推荐度    |
| ------------------------ | --------- | -------------- | --------- |
| `/table/columns`         | 1         | 1              | ✅ 推荐   |
| `/table/database/tables` | 1         | 10（单个 DB）  | ✅ 推荐   |
| `/table/overview`        | 1         | 100（全部）    | ❌ 不推荐 |

---

**文档版本**: v1.0  
**最后更新**: 2026-04-04
