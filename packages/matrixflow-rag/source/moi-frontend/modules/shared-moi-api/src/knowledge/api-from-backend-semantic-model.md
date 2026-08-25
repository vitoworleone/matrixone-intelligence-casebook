# 语义模型管理接口

> Controller: `semantic_model.go` | Service: `pkg/session/semantic_model_service.go`

基于 moi-core v2 语义模型封装的 RESTful 接口，替代原 NL2SQL 知识管理，提供语义配置的完整 CRUD、批量导入/导出和校验功能。

**路由前缀**: `/newmoi/semantic-models/:model_id`

---

## 目录

- [统一响应格式](#统一响应格式)
- [错误码](#错误码)
- [枚举值](#枚举值)
- [数据类型](#数据类型)
- [接口列表](#接口列表)
  - [创建语义模型](#post-newmoisemantic-models)
  - [数据侧创建空知识库](#post-newmoisemantic-modelscreate-empty)
  - [创建知识库并导入数据源](#post-newmoisemantic-modelscreate-with-sources)
  - [列出语义模型](#get-newmoisemantic-models)
  - [更新语义模型](#put-newmoisemantic-modelsmodel_id)
  - [删除语义模型](#delete-newmoisemantic-modelsmodel_id)
  - [获取语义模型](#get-newmoisemantic-modelsmodel_id)
  - [预览知识库原文档](#get-newmoisemantic-modelsmodel_idsourcesfilefile_idpreview)
  - [预览知识库解析图片产物](#get-newmoisemantic-modelsmodel_idartifactsfile_idpreview)
  - [数据源列表](#get-newmoisemantic-modelsmodel_idsources)
  - [文档详情](#get-newmoisemantic-modelsmodel_idsourcesfile_iddocument)
  - [更新文档治理](#patch-newmoisemantic-modelsmodel_idsourcesfile_idgovernance)
  - [删除数据源](#delete-newmoisemantic-modelsmodel_idsourcesfile_id)
  - [追加数据源](#post-newmoisemantic-modelsmodel_idsources)
  - [查询知识库 source jobs](#get-newmoisemantic-modelsmodel_idsource-jobs)
  - [同步知识库 source jobs](#post-newmoisemantic-modelsmodel_idsource-jobsreconcile)
  - [语义条目列表](#get-newmoisemantic-modelsmodel_identries)
  - [创建语义条目](#post-newmoisemantic-modelsmodel_identries)
  - [更新语义条目](#put-newmoisemantic-modelsmodel_identriesentry_id)
  - [删除语义条目](#delete-newmoisemantic-modelsmodel_identriesentry_id)
  - [批量导入](#post-newmoisemantic-modelsmodel_idimport)
  - [导出](#get-newmoisemantic-modelsmodel_idexport)
  - [校验](#post-newmoisemantic-modelsmodel_idvalidate)

---

## 统一响应格式

```typescript
interface ApiResponse<T> {
  code: string;
  msg: string;
  data: T;
}
```

---

## 错误码

| 错误码            | HTTP | 说明                                 |
| ----------------- | ---- | ------------------------------------ |
| `OK`              | 200  | 成功                                 |
| `ErrParamInvalid` | 400  | 参数错误（kind 非法、spec 缺必填等） |
| `ErrUnauthorized` | 401  | 未登录                               |
| `ErrForbidden`    | 403  | 无权限                               |
| `ErrNotFound`     | 404  | 知识库或语义模型不存在               |
| `ErrConflict`     | 409  | entry key 重复                       |
| `ErrServer`       | 500  | 服务器内部错误                       |

---

## 枚举值

### SemanticKind（语义条目类型）

| 值                  | 说明              | 必填 spec 字段                              |
| ------------------- | ----------------- | ------------------------------------------- |
| `dimension`         | 维度列            | `column`                                    |
| `fact`              | 事实列            | `column`                                    |
| `metric`            | 业务指标          | `expr`                                      |
| `relationship`      | 表关联关系        | `left_table`, `right_table`, `join_columns` |
| `column_preference` | 列别名/废弃列替换 | `preferred`, `deprecated`                   |
| `named_filter`      | 命名过滤条件      | `expr`                                      |
| `verified_query`    | 已验证的标准问答  | `question`, `sql`                           |
| `glossary`          | 业务术语解释      | `term`, `definition`                        |
| `logic_text`        | 自然语言规则注入  | `content`, `injection_stages`               |
| `sql_resultset`     | SQL 结果集        | `sql`, `description`                        |

### InjectionStage（logic_text 注入阶段）

| 值                  | 说明           |
| ------------------- | -------------- |
| `planner_policy`    | 规划阶段策略   |
| `sql_generation`    | SQL 生成阶段   |
| `sql_followup`      | 追问 SQL 阶段  |
| `sql_regenerate`    | SQL 重生成阶段 |
| `sql_decomposition` | SQL 分解阶段   |
| `executor_rule`     | 执行器规则     |
| `renderer_rule`     | 渲染器规则     |

---

## 数据类型

```typescript
interface SemanticModelTable {
  db_name: string;
  table_names: string[];
  parents?: string[];
}

interface SemanticModelFiles {
  file_ids: string[];
  parents?: string[];
  volume_ids?: string[];
  vector_table?: string;
  embedding_model?: string;
  image_vector_table?: string;
  image_embedding_model?: string;
  image_embedding_dimension?: number;
  image_embedding_backend_id?: string;
  image_preprocess_version?: string;
  image_distance_metric?: string;
}

interface SemanticModel {
  id: number;
  name: string;
  description: string;
  tables: SemanticModelTable[]; // 结构化表集合
  files?: SemanticModelFiles; // 关联文件集合
  source_counts: {
    files: number; // source 列表中非 table 类型数量；列表/详情接口为权威值
    tables: number; // source 列表中 table 类型数量；列表/详情接口为权威值
    total: number;
  };
  table_set_hash: string;
  created_at: number;
  updated_at: number;
}

type SemanticModelSourceType = 'file' | 'volume' | 'table';

interface SemanticModelSource {
  row_id: string;
  source_id?: string;
  source_type: SemanticModelSourceType;
  model_id: number;
  resource_id: string;
  source_resource_id?: string | null;
  kb_resource_id?: string | null;
  source_file_id?: string | null;
  kb_file_id?: string | null;
  display_name: string | null;
  path: string[];
  source_path?: string | null;
  db_name: string | null;
  table_name: string | null;
  size_bytes?: number | null;
  row_count?: number | null;
  ingest_status: string | null;
  enabled: boolean | null;
  expires_at: number | null;
  expired: boolean;
  effective_enabled: boolean;
  force_enabled_after_expiry: boolean;
  tags?: string[];
  segment_version_id: string | null;
  index_version: number | null;
  updated_at?: number | null;
  error: string | null;
}

interface SemanticModelSourceListResponse {
  items: SemanticModelSource[];
  total: number;
}

interface SemanticModelSourcePreview {
  available: boolean;
  content?: string | null;
  reason?: string | null;
}

interface SemanticModelSourceFileInfo {
  tags: string[];
  expires_at: number | null;
  enabled: boolean | null;
  expired: boolean;
  effective_enabled: boolean;
  force_enabled_after_expiry: boolean;
  index_version: number | null;
  segment_version_id: string | null;
}

interface SemanticModelSegmentStatus {
  available: boolean;
  reason?: string | null;
  total: number;
}

interface SemanticModelSegmentVersion {
  version_id: string;
  current: boolean;
  index_version?: number | null;
}

interface SemanticModelSourceDocument {
  source: SemanticModelSource;
  preview: SemanticModelSourcePreview;
  file_info: SemanticModelSourceFileInfo;
  segment_status: SemanticModelSegmentStatus;
  segment_versions: SemanticModelSegmentVersion[];
}

interface UpdateSemanticModelSourceGovernanceRequest {
  tags?: string[];
  expires_at?: number | null;
  enabled?: boolean;
  force_enabled_after_expiry?: boolean;
}

interface UpdateSemanticModelSourceGovernanceResponse {
  source: SemanticModelSource;
}

type SemanticModelCreateSourceType = 'local_file' | 'catalog_file' | 'catalog_table';

interface SemanticModelCreateSource {
  source_type: SemanticModelCreateSourceType;
  file_name?: string;
  upload_kind?: 'unstructured' | 'structured';
  table_config?: string;
  file_id?: string;
  /** catalog_file 必填：文件所在的权威来源 Volume（非知识库 raw Volume）。 */
  volume_id?: number;
  table_id?: number;
}

interface SemanticModelSourceSelectionFilters {
  table_name?: string;
  file_name?: string;
  file_ext?: string[];
}

type SemanticModelSourceSelection =
  | {
      kind: 'database_tables';
      database_id: number;
      all_selected: boolean;
      selected_table_ids?: number[];
      excluded_table_ids?: number[];
      filters?: Pick<SemanticModelSourceSelectionFilters, 'table_name'>;
    }
  | {
      kind: 'volume_files';
      volume_id: number;
      all_selected: boolean;
      selected_file_ids?: string[];
      excluded_file_ids?: string[];
      filters?: Pick<SemanticModelSourceSelectionFilters, 'file_name' | 'file_ext'>;
    };

interface KnowledgeBaseSourceJobRun {
  job_id: string;
  source_id: string;
  model_id: number;
  job_type: string;
  job_status: string;
  idempotency_key: string;
  operation_id: string | null;
  workflow_execution_id: string | null;
  source_file_id: string | null;
  kb_file_id: string | null;
  source_table_id: number | null;
  kb_table_id: number | null;
  retry_count: number;
  next_retry_at: number | null;
  error: string | null;
}

interface KnowledgeBaseDataDomain {
  model_id: number;
  catalog_id: number;
  database_id: number;
  raw_volume_id: number;
  processed_volume_id: number;
  ensure_status: string;
  last_ensure_error: string | null;
  last_checked_at: number;
}

interface CreateSemanticModelWithSourcesResponse {
  model: SemanticModel;
  data_domain: KnowledgeBaseDataDomain;
  sources: SemanticModelSource[];
  jobs: KnowledgeBaseSourceJobRun[];
}

interface AppendSemanticModelSourcesRequest {
  sources: SemanticModelCreateSource[];
  source_selections?: SemanticModelSourceSelection[];
}

interface AppendSemanticModelSourcesResponse {
  data_domain: KnowledgeBaseDataDomain;
  sources: SemanticModelSource[];
  jobs: KnowledgeBaseSourceJobRun[];
}

interface SemanticEntry {
  id: number;
  kind: SemanticKind;
  key: string;
  tables: string[]; // 条目作用表（平铺表名）
  spec: object; // 按 kind 不同结构，见下方
  created_at: number;
  updated_at: number;
}

// 创建/更新语义模型请求体
interface SemanticModelUpsertRequest {
  name: string;
  description?: string;
  tables: SemanticModelTable[];
  files?: SemanticModelFiles;
}
```

### Spec 结构（按 kind）

**dimension:**

```json
{
  "column": "status",
  "data_type": "VARCHAR",
  "synonyms": ["订单状态"],
  "is_enum": true,
  "sample_values": ["PAID", "PENDING"],
  "is_time": false,
  "deprecated": false,
  "description": ""
}
```

**fact:**

```json
{
  "column": "amount",
  "data_type": "DECIMAL",
  "description": "订单金额",
  "private": false
}
```

**metric:**

```json
{
  "expr": "SUM(amount)",
  "synonyms": ["总收入"],
  "unit": "CNY",
  "description": "",
  "requires_tables": ["orders"],
  "requires_join": "orders_to_refunds",
  "semantic_pattern": "ratio"
}
```

**relationship:**

```json
{
  "left_table": "orders",
  "right_table": "customers",
  "join_columns": [{ "left": "customer_id", "right": "id" }],
  "description": "",
  "semantic_match": false
}
```

**column_preference:**

```json
{
  "preferred": "amount",
  "deprecated": "order_amount",
  "reason": "legacy column"
}
```

**named_filter:**

```json
{
  "expr": "status IN ('PAID', 'COMPLETE')",
  "synonyms": ["已完成订单"],
  "description": "",
  "applies_to": ["orders"]
}
```

**verified_query:**

```json
{
  "question": "本月总收入是多少",
  "sql": "SELECT SUM(amount) FROM orders WHERE MONTH(order_date) = MONTH(NOW())",
  "verified_by": "user_001",
  "tags": ["revenue", "monthly"]
}
```

**glossary:**

```json
{
  "term": "GMV",
  "definition": "Gross Merchandise Volume，总成交额",
  "synonyms": ["总成交额"],
  "related_metrics": ["total_revenue"],
  "formula_hint": "SUM(order_amount)"
}
```

**logic_text:**

```json
{
  "content": "查询金额时默认只统计 status = 'PAID' 的订单",
  "injection_stages": ["sql_generation", "sql_followup"],
  "priority": 1
}
```

**sql_resultset:**

```json
{
  "sql": "SELECT id, amount FROM orders WHERE status = 'PAID'",
  "description": "用于查询已支付订单的结果集",
  "expand_sql": {
    "sql": "SELECT id, amount FROM orders WHERE status = '{{status}}'",
    "params": ["status"]
  },
  "max_rows": 10000,
  "max_bytes": 2097152,
  "timeout_seconds": 10
}
```

---

## 接口列表

### POST /newmoi/semantic-models

创建语义模型。

**请求体:**

```json
{
  "name": "销售分析模型",
  "description": "用于销售数据 NL2SQL",
  "tables": [
    {
      "db_name": "sales_db",
      "table_names": ["orders", "customers"],
      "parents": []
    }
  ],
  "files": { "file_ids": ["file_001"], "parents": [] }
}
```

**响应 201:**

```json
{
  "code": "OK",
  "msg": "OK",
  "data": {
    "id": 1,
    "name": "销售分析模型",
    "description": "用于销售数据 NL2SQL",
    "tables": [
      {
        "db_name": "sales_db",
        "table_names": ["orders", "customers"],
        "parents": []
      }
    ],
    "files": { "file_ids": ["file_001"], "parents": [] },
    "table_set_hash": "abc123...",
    "created_at": 1712345678,
    "updated_at": 1712345678
  }
}
```

| 状态码 | 错误码            | 说明                |
| ------ | ----------------- | ------------------- |
| 400    | `ErrParamInvalid` | name 或 tables 为空 |
| 409    | `ErrConflict`     | 同名模型已存在      |

---

### POST /newmoi/semantic-models/create-empty

数据侧知识库页面创建不含数据源的知识库。请求只提交名称、备注和图片索引开关；后端创建语义模型、Catalog 数据库、raw/processed Volume 与固定索引配置，不创建 source、job、本地上传或 RAG 工作流。

```typescript
interface CreateEmptySemanticModelRequest {
  name: string;
  description?: string;
  image_index_enabled?: boolean;
}

interface CreateEmptySemanticModelResponse {
  model: SemanticModel;
  data_domain: KnowledgeBaseDataDomain;
}
```

```json
{
  "name": "产品文档知识库",
  "description": "产品资料",
  "image_index_enabled": true
}
```

---

### POST /newmoi/semantic-models/create-with-sources

创建知识库并同时提交初始数据源。新建知识库的 data-domain 固定使用 workspace 初始化时保留的默认 Catalog（与本地上传路径一致），不会把知识库 db/volume 挂到来源 Catalog 下。创建重试不会把已有 data domain 的正数 `catalog_id` 迁移到默认 Catalog；只有 data domain 已绑定 database 时，才按该 database 的真实父 Catalog 修复关联。请求不得再传 `target_catalog_id`（已移除；传入返回 400）。`sources` 与 `source_selections` 至少一个非空，也可以同时传入；本地上传和结构化上传继续通过 `sources[].source_type="local_file"` 提交。Catalog 大批量选择使用 `source_selections` 表达 database 表叶子或 volume 文件叶子选择，后端按页展开、校验归属、去重并写入 source/job 元数据；文件 link/copy、表 clone、RAG 后续处理由知识库资源列表轮询 `source-jobs/reconcile` 有限批次推进，不在 create 请求内全量处理。

新建知识库固定使用文本模型 `bge-m3` 和图片模型 `efficientnet-b3`（1536 维、预处理版本 `efficientnet-b3-v1-rgb-300-letterbox-imagenet`、距离度量 `cosine`）。后端从当前 workspace 的可用 embedding 能力中解析实际图片 backend id；两个固定模型任一不可用时返回 400，且不会创建语义模型。请求 `files` 可选，但其中的 `embedding_model`、`image_embedding_*`、`vector_table` 和 `image_vector_table` 均会被忽略；响应始终返回后端生成的向量表、固定模型配置和本次解析到的图片 backend id。

响应中的 `data.sources` / `data.jobs` 表示本次创建提交的数据源与作业。`data.model.source_counts` 不作为创建响应的权威来源计数；需要展示来源数量时应读取列表/详情接口返回的 `source_counts`。

完整图片 embedding 配置只表示该知识库具备图片索引能力。发布 segment version 时，如果模型已绑定后端生成的 `image_vector_table`，后端会按 `kb_file_id + index_version` 读取实际图片 rows；读到 rows 才生成图片段。当前解析文档没有 page image 或 visual object 时，segment version 可以只包含文本段。图片向量表不可见、schema 不匹配或读取失败仍是错误，不会被当作“无图片”成功。

**请求体:**

```json
{
  "name": "产品文档知识库",
  "description": "产品资料",
  "files": {
    "file_ids": [],
    "parents": []
  },
  "sources": [
    {
      "source_type": "local_file",
      "file_name": "manual.pdf",
      "file_id": "uploaded-file-1",
      "upload_kind": "unstructured"
    },
    {
      "source_type": "catalog_file",
      "file_id": "catalog-file-1",
      "volume_id": 3001
    },
    {
      "source_type": "catalog_table",
      "table_id": 1001
    }
  ],
  "source_selections": [
    {
      "kind": "database_tables",
      "database_id": 1001,
      "all_selected": true,
      "excluded_table_ids": [2002],
      "filters": { "table_name": "order" }
    },
    {
      "kind": "volume_files",
      "volume_id": 3001,
      "all_selected": true,
      "excluded_file_ids": ["file-2"],
      "filters": { "file_name": "manual", "file_ext": ["pdf", "docx"] }
    }
  ]
}
```

**响应 201:**

```json
{
  "code": "OK",
  "msg": "OK",
  "data": {
    "model": {
      "id": 7,
      "name": "产品文档知识库",
      "description": "产品资料",
      "tables": [],
      "files": {
        "file_ids": ["kb-file-1"],
        "parents": [],
        "vector_table": "kb_7_text_index",
        "embedding_model": "bge-m3",
        "image_vector_table": "kb_7_image_index",
        "image_embedding_model": "efficientnet-b3",
        "image_embedding_backend_id": "42",
        "image_embedding_dimension": 1536,
        "image_preprocess_version": "efficientnet-b3-v1-rgb-300-letterbox-imagenet",
        "image_distance_metric": "cosine"
      },
      "table_set_hash": "",
      "created_at": 1712345678,
      "updated_at": 1712345678
    },
    "data_domain": {
      "model_id": 7,
      "catalog_id": 20001,
      "database_id": 11,
      "raw_volume_id": 12,
      "processed_volume_id": 13,
      "ensure_status": "ready",
      "last_ensure_error": null,
      "last_checked_at": 1712345678
    },
    "sources": [],
    "jobs": []
  }
}
```

---

### GET /newmoi/semantic-models

分页列出语义模型。

**查询参数:**

| 参数         | 类型   | 必填 | 说明                     |
| ------------ | ------ | ---- | ------------------------ |
| `page_size`  | int    | 否   | 每页数量，1-100，默认 20 |
| `page_token` | string | 否   | 分页游标                 |

**响应 200:**

```json
{
  "code": "OK",
  "msg": "OK",
  "data": {
    "items": [
      {
        "id": 1,
        "name": "销售分析模型",
        "tables": [
          {
            "db_name": "sales_db",
            "table_names": ["orders", "customers"],
            "parents": []
          }
        ],
        "table_set_hash": "abc123...",
        "created_at": 1712345678,
        "updated_at": 1712345678
      }
    ],
    "total": 1,
    "next_page_token": ""
  }
}
```

---

### PUT /newmoi/semantic-models/:model_id

更新语义模型元信息或表/文件配置。`name` 创建后不可修改，必须与当前知识库名称完全一致（与 Catalog 物理数据库名对齐）；改名返回 `400 ErrParamInvalid`。

**请求体:**

```json
{
  "name": "销售分析模型",
  "description": "更新后的描述",
  "tables": [
    {
      "db_name": "sales_db",
      "table_names": ["orders", "customers", "refunds"],
      "parents": []
    }
  ],
  "files": { "file_ids": ["file_001", "file_002"], "parents": [] }
}
```

**响应 200:**

```json
{ "code": "OK", "msg": "OK", "data": { "updated": true } }
```

| 状态码 | 错误码            | 说明                                                      |
| ------ | ----------------- | --------------------------------------------------------- |
| 400    | `ErrParamInvalid` | name 为空、与当前名称不一致（不可改名）、或 tables 不合法 |
| 404    | `ErrNotFound`     | 模型不存在                                                |

---

### DELETE /newmoi/semantic-models/:model_id

删除语义模型及其所有条目。

**响应 200:**

```json
{ "code": "OK", "msg": "OK", "data": { "deleted": true } }
```

| 状态码 | 错误码        | 说明       |
| ------ | ------------- | ---------- |
| 404    | `ErrNotFound` | 模型不存在 |

---

### GET /newmoi/semantic-models/:model_id

获取知识库关联的语义模型信息。

**响应 200:**

```json
{
  "code": "OK",
  "msg": "OK",
  "data": {
    "id": 1,

    "name": "kb_42_semantic",
    "description": "",
    "tables": [
      {
        "db_name": "default",
        "table_names": ["customers", "orders", "refunds"],
        "parents": []
      }
    ],
    "files": null,
    "table_set_hash": "abc123...",
    "created_at": 1712345678,
    "updated_at": 1712345678
  }
}
```

**错误:**
| HTTP | code | 场景 |
|------|------|------|
| 404 | `ErrNotFound` | 知识库不存在或尚未创建语义模型 |

---

### GET /newmoi/semantic-models/:model_id/sources/file/:file_id/preview

预览当前知识库关联的原文档。来源详情调用方仅传 `source.source_file_id` 这一工作流原文档 File ID；该字段为空时必须显示不可预览，不能回退到 `source_resource_id`、`kb_file_id` 或 `kb_resource_id`。问答来源直接传其 `file_id`。后端通过目标模型向量表的 workflow data lineage 验证该原文档归属目标知识库。该接口返回原始二进制流，不使用 JSON 信封，调用方应以 `blob` 响应类型读取并在不再展示时释放对象 URL。

除该预览接口外，其他 `/sources/:source_row_id/...` 管理接口仍传 source row ID。

**权限:** 目标知识库 `semantic_model.read`。现有 PEP 对 SuperAdmin 的放行保持不变，其他用户必须有该知识库读权限。

| HTTP | code              | 场景                                              |
| ---- | ----------------- | ------------------------------------------------- |
| 400  | `ErrParamInvalid` | `model_id` 或 `file_id` 非法                      |
| 403  | `ErrForbidden`    | 无目标模型读取权限                                |
| 404  | `ErrNotFound`     | 原文档未通过 workflow data lineage 关联到目标模型 |
| 500  | `ErrServer`       | 文档关联查询或文件流不可用                        |

---

### GET /newmoi/semantic-models/:model_id/artifacts/:file_id/preview

预览知识库解析产生的图片、页图或其他文档产物。后端先通过 workflow data lineage 将产物反查到原文档，再验证原文档关联目标模型的向量表。该接口返回原始二进制流，不使用 JSON 信封；调用方应以 `blob` 响应类型读取，并在不再展示时释放对象 URL。

**权限:** 目标知识库 `semantic_model.read`。后端会验证 `file_id` 通过 workflow data lineage 关联到目标模型；未关联或跨模型的 ID 返回 404。接口不接收 `volume_id`，也不回退到通用 Catalog 文件预览。

**响应 200 headers:**

- `Content-Type`: 图片 MIME；未知时为 `application/octet-stream`
- `Content-Disposition`: 文件名可用时为 inline filename

| HTTP | code              | 场景                         |
| ---- | ----------------- | ---------------------------- |
| 400  | `ErrParamInvalid` | `model_id` 或 `file_id` 非法 |
| 403  | `ErrForbidden`    | 无目标模型读取权限           |
| 404  | `ErrNotFound`     | 图片产物未关联到目标模型     |
| 500  | `ErrServer`       | 产物关联或文件流不可用       |

---

### GET /newmoi/semantic-models/:model_id/sources

只读返回知识库关联的数据源行。阶段 4 前端数据源 tab 必须消费该接口，不直接解析 `SemanticModel.files` / `SemanticModel.tables`。

返回行分为两类：

- `governance_status=managed`：已写入 `knowledge_base_sources` 的治理 source。
- `governance_status=legacy_unbound`：只读历史候选行，来源于 semantic model 显式文件/表或 lineage register；`source_id` 为空，治理操作在前端禁用。

**响应 200:**

```json
{
  "code": "OK",
  "msg": "OK",
  "data": {
    "items": [
      {
        "row_id": "7:table:sales_db::orders",
        "source_type": "table",
        "model_id": 7,
        "resource_id": "sales_db::orders",
        "display_name": "orders",
        "path": ["sales_db", "orders"],
        "db_name": "sales_db",
        "table_name": "orders",
        "ingest_status": "unsupported",
        "enabled": true,
        "expires_at": null,
        "expired": false,
        "effective_enabled": true,
        "force_enabled_after_expiry": false,
        "tags": ["finance"],
        "segment_version_id": null,
        "index_version": null,
        "error": null
      }
    ],
    "total": 1,
    "legacy_backfill_required": false
  }
}
```

治理字段由后端按 query-time 过期状态计算返回；前端不得从 `SemanticModel.files` / `tables` 合成 source row。

---

### POST /newmoi/semantic-models/:model_id/sources/backfill-legacy

显式补齐历史 workflow/lineage 关联知识库数据的 source/job 关系。该接口只补 `knowledge_base_sources` 与 `knowledge_base_source_job_runs`，不导入 chunk，不迁移向量 rows。已有 source 保留原 `source_id` 和治理字段。

接口无请求体且幂等；重复调用或并发调用不会创建重复 source，也不会覆盖 `enabled/expires_at/tags/force_enabled_after_expiry/segment_version_id/index_version`。单次调用只处理有限批次，前端可在 `GET /sources` 返回 `legacy_backfill_required=true` 或存在 `legacy_unbound` 行后自动调用该 POST，再刷新 source list；刷新后如仍有候选行，可继续处理下一批。

该接口是状态变更操作，需要知识库对象 K3/update。只读用户打开页面时不能自动调用该 POST。

**请求体:** 无

**响应 200:**

```json
{
  "code": "OK",
  "msg": "OK",
  "data": {
    "updated": true
  }
}
```

---

### GET /newmoi/semantic-models/:model_id/source-jobs

只读返回全知识库工作信号：`items` 最多 32 个 persisted active/incomplete source job 诊断视图，`total` 是同条件下的全量 persisted source 数，顶层 `reconcile_required` 表示是否需要调用 POST reconcile。只有 semantic model 显式 file/table、legacy job、raw-volume、lineage 等 backfill 工作时，响应可以是 `items=[]、total=0、reconcile_required=true`。GET 不写数据库、不触发外部任务。

```ts
interface SemanticModelSourceJobListResponse {
  items: KnowledgeBaseSourceJobRun[];
  total: number;
  reconcile_required: boolean;
}
```

---

### POST /newmoi/semantic-models/:model_id/source-jobs/reconcile

显式刷新知识库 source jobs，并在 backend 已创建的 RAG source job 成功后发布 segment version。该接口是状态变更操作，需要知识库对象 K3/update；只读用户打开页面时不能自动调用。`GET /sources` 与 `GET /source-jobs` 保持只读，不隐式发布 chunks。

页面只能依据 `GET /source-jobs` 顶层 `reconcile_required` 调用本接口，不能从 `items`、job type 或状态重算。POST 不接收 GET 返回的 source/job ID，每轮面向全 KB 独立选批；`running` 任务按最久未检查顺序自然轮换，不使用 cursor。只读用户不得调用 POST，也不启动持续 reconcile 轮询。

**请求体:** 无

**响应 200:**

```json
{
  "code": "OK",
  "msg": "OK",
  "data": {
    "updated": true
  }
}
```

---

### GET /newmoi/semantic-models/:model_id/sources/:source_row_id/document

返回非结构化文档的详情骨架、治理状态、预览状态和只读分段状态。没有真实解析内容或分段列表时，后端必须返回明确不可用状态，不伪造正文或 chunk。

**响应 200:**

```json
{
  "code": "OK",
  "msg": "OK",
  "data": {
    "source": {
      "row_id": "7:file:file-1",
      "source_type": "file",
      "model_id": 7,
      "resource_id": "file-1",
      "display_name": "manual.pdf",
      "path": [],
      "db_name": null,
      "table_name": null,
      "ingest_status": "ready",
      "enabled": true,
      "expires_at": null,
      "expired": false,
      "effective_enabled": true,
      "force_enabled_after_expiry": false,
      "tags": ["product"],
      "segment_version_id": "segment-v1",
      "index_version": 3,
      "error": null
    },
    "preview": {
      "available": false,
      "reason": "parsed document content is unavailable"
    },
    "file_info": {
      "tags": ["product"],
      "expires_at": null,
      "enabled": true,
      "expired": false,
      "effective_enabled": true,
      "force_enabled_after_expiry": false,
      "index_version": 3,
      "segment_version_id": "segment-v1"
    },
    "segment_status": {
      "available": false,
      "reason": "segment list is not available in phase 4",
      "total": 0
    },
    "segment_versions": [
      {
        "version_id": "segment-v1",
        "current": true,
        "index_version": 3
      }
    ]
  }
}
```

**错误:**
| HTTP | code | 场景 |
|------|------|------|
| 400 | `ErrParamInvalid` | source row 不是文档类 source |
| 404 | `ErrNotFound` | 知识库或 source row 不存在 |

---

### PATCH /newmoi/semantic-models/:model_id/sources/:source_row_id/governance

更新非结构化文档治理字段。标签只作为 metadata、列表/详情筛选和 tool result metadata，不作为 RAG filter。

**请求体:**

```json
{
  "tags": ["product", "phase4"],
  "expires_at": 1782705000,
  "enabled": true,
  "force_enabled_after_expiry": false
}
```

**响应 200:**

```json
{
  "code": "OK",
  "msg": "OK",
  "data": {
    "source": {
      "row_id": "7:file:file-1",
      "source_type": "file",
      "model_id": 7,
      "resource_id": "file-1",
      "display_name": "manual.pdf",
      "path": [],
      "db_name": null,
      "table_name": null,
      "ingest_status": "ready",
      "enabled": true,
      "expires_at": 1782705000,
      "expired": false,
      "effective_enabled": true,
      "force_enabled_after_expiry": false,
      "tags": ["product", "phase4"],
      "segment_version_id": "segment-v1",
      "index_version": 3,
      "error": null
    }
  }
}
```

---

### DELETE /newmoi/semantic-models/:model_id/sources/:source_row_id

删除一个知识库数据源。该操作会删除 Catalog 中对应的 KB 内部副本文档或克隆表，更新 semantic model 的 source scope，同步禁用该文档的文本/图片向量行，并删除 `knowledge_base_sources` / source job 关系行。删除成功后，数据源列表不再返回该 source。

**响应 200:**

```json
{
  "code": "OK",
  "msg": "OK",
  "data": {
    "deleted": true
  }
}
```

**错误:**
| HTTP | code | 场景 |
|------|------|------|
| 400 | `ErrParamInvalid` | model_id 或 source row ID 非法 |
| 404 | `ErrNotFound` | 语义模型或 source row 不存在 |
| 500 | `ErrServer` | 更新 semantic model scope、删除 Catalog 源、同步 vector disabled 或删除关系行失败 |

---

### POST /newmoi/semantic-models/:model_id/sources

向已有知识库追加本地文件、Catalog 文件、Catalog 表，或追加 `source_selections` 表达的 database 表叶子 / volume 文件叶子选择。`sources` 与 `source_selections` 至少一个非空，也可以同时传入；本地上传和结构化上传继续走 `sources.local_file`。`source_selections` 只在请求内分页展开并写入 source/job 元数据，后续文件 link/copy、表 clone、RAG 处理由资源列表轮询 `source-jobs/reconcile` 有限批次推进。追加成功后前端必须重新请求 `/sources` 和 `/source-jobs`，不得用请求体伪造本地 source list。

**请求体:**

```json
{
  "sources": [
    {
      "source_type": "local_file",
      "file_name": "append.txt",
      "file_id": "uploaded-file-append",
      "upload_kind": "unstructured"
    },
    {
      "source_type": "catalog_file",
      "file_id": "catalog-file-append",
      "volume_id": 3001
    },
    {
      "source_type": "catalog_table",
      "table_id": 1002
    }
  ],
  "source_selections": [
    {
      "kind": "volume_files",
      "volume_id": 3001,
      "all_selected": true,
      "excluded_file_ids": ["file-2"],
      "filters": { "file_name": "manual" }
    }
  ]
}
```

**响应 200:**

```json
{
  "code": "OK",
  "msg": "OK",
  "data": {
    "data_domain": {
      "model_id": 7,
      "catalog_id": 3,
      "database_id": 11,
      "raw_volume_id": 12,
      "processed_volume_id": 13,
      "ensure_status": "ready",
      "last_ensure_error": null,
      "last_checked_at": 1712345678
    },
    "sources": [],
    "jobs": []
  }
}
```

---

### GET /newmoi/semantic-models/:model_id/entries

查询语义条目列表。

**Query Params:**
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| kind | string | 否 | 过滤条目类型 |
| page_size | number | 否 | 分页大小，默认 20，最大 100 |
| page_token | string | 否 | 分页游标 |

**响应 200:**

```json
{
  "code": "OK",
  "msg": "OK",
  "data": {
    "items": [
      {
        "id": 1,
        "kind": "metric",
        "key": "total_revenue",
        "tables": ["orders"],
        "spec": { "expr": "SUM(amount)" },
        "created_at": 1712345678,
        "updated_at": 1712345678
      }
    ],
    "total": 1,
    "next_page_token": ""
  }
}
```

**错误:**
| HTTP | code | 场景 |
|------|------|------|
| 400 | `ErrParamInvalid` | kind 不合法 |
| 404 | `ErrNotFound` | 知识库或语义模型不存在 |

---

### POST /newmoi/semantic-models/:model_id/entries

创建语义条目。如果知识库还没有语义模型，自动创建。

**请求体:**

```json
{
  "kind": "metric",
  "key": "total_revenue",
  "tables": ["orders"],
  "spec": { "expr": "SUM(amount)", "synonyms": ["总收入"], "unit": "CNY" }
}
```

**响应 201:**

```json
{
  "code": "OK",
  "msg": "OK",
  "data": {
    "id": 1,
    "kind": "metric",
    "key": "total_revenue",
    "tables": ["orders"],
    "spec": { "expr": "SUM(amount)", "synonyms": ["总收入"], "unit": "CNY" },
    "created_at": 1712345678,
    "updated_at": 1712345678
  }
}
```

**错误:**
| HTTP | code | 场景 |
|------|------|------|
| 400 | `ErrParamInvalid` | kind/key/spec 缺失或不合法 |
| 400 | `ErrParamInvalid` | 知识库未配置 tables |
| 404 | `ErrNotFound` | 知识库不存在 |
| 409 | `ErrConflict` | key 已存在 |

---

### PUT /newmoi/semantic-models/:model_id/entries/:entry_id

更新语义条目。kind 不可修改。

**请求体:** 同创建

**响应 200:**

```json
{ "code": "OK", "msg": "OK", "data": { "updated": true } }
```

**错误:**
| HTTP | code | 场景 |
|------|------|------|
| 400 | `ErrParamInvalid` | 校验失败或 kind 不一致 |
| 404 | `ErrNotFound` | 知识库或条目不存在 |
| 409 | `ErrConflict` | 修改 key 后冲突 |

---

### DELETE /newmoi/semantic-models/:model_id/entries/:entry_id

删除语义条目。

**响应 200:**

```json
{ "code": "OK", "msg": "OK", "data": { "deleted": true } }
```

**错误:**
| HTTP | code | 场景 |
|------|------|------|
| 404 | `ErrNotFound` | 知识库或条目不存在 |

---

### POST /newmoi/semantic-models/:model_id/import

覆盖式批量导入，替换所有现有条目。

**请求体:**

```json
{
  "entries": [
    {
      "kind": "metric",
      "key": "total_revenue",
      "tables": ["orders"],
      "spec": { "expr": "SUM(amount)" }
    },
    {
      "kind": "relationship",
      "key": "orders_to_customers",
      "tables": ["orders", "customers"],
      "spec": {
        "left_table": "orders",
        "right_table": "customers",
        "join_columns": [{ "left": "customer_id", "right": "id" }]
      }
    }
  ]
}
```

**响应 200:**

```json
{ "code": "OK", "msg": "OK", "data": { "imported": 2, "model_id": 1 } }
```

**错误:**
| HTTP | code | 场景 |
|------|------|------|
| 400 | `ErrParamInvalid` | entries 为空、校验失败、key 重复 |
| 404 | `ErrNotFound` | 知识库不存在 |

---

### GET /newmoi/semantic-models/:model_id/export

导出语义模型及所有条目。

**响应 200:**

```json
{
  "code": "OK",
  "msg": "OK",
  "data": {
    "model": {
      "name": "kb_42_semantic",
      "tables": [
        {
          "db_name": "default",
          "table_names": ["customers", "orders", "refunds"],
          "parents": []
        }
      ],
      "files": null
    },
    "entries": [
      {
        "kind": "metric",
        "key": "total_revenue",
        "tables": ["orders"],
        "spec": { "expr": "SUM(amount)" }
      }
    ]
  }
}
```

**错误:**
| HTTP | code | 场景 |
|------|------|------|
| 404 | `ErrNotFound` | 知识库或语义模型不存在 |

---

### POST /newmoi/semantic-models/:model_id/validate

校验语义模型完整性。

**响应 200（通过）:**

```json
{ "code": "OK", "msg": "OK", "data": { "valid": true } }
```

**响应 200（失败，HTTP 仍为 200）:**

```json
{
  "code": "OK",
  "msg": "OK",
  "data": {
    "valid": false,
    "errors": ["metric 'refund_rate' requires_join references missing relationship 'orders_to_refunds'"]
  }
}
```

**错误:**
| HTTP | code | 场景 |
|------|------|------|
| 404 | `ErrNotFound` | 知识库或语义模型不存在 |

---

**文档版本**: v1.0
**最后更新**: 2026-04-11
