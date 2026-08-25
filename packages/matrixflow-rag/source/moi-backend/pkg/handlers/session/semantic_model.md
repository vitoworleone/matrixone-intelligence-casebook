# 语义模型管理接口

> Controller: `semantic_model.go` | Service: `pkg/session/semantic_model_service.go`

基于 moi-core v2 语义模型封装的 RESTful 接口，替代原 NL2SQL 知识管理，提供语义配置的完整 CRUD、批量导入/导出和校验功能。

分段中的 `image_file_id` / `page_image_file_id` 是解析流程产生并维护的只读产物身份。调用方不能通过手工创建 chunk 注入或绑定这两个字段；产物预览也只接受已提交的可信解析版本关联。

**路由前缀**: `/newmoi/semantic-models/:model_id`

---

## 目录

- [统一响应格式](#统一响应格式)
- [错误码](#错误码)
- [枚举值](#枚举值)
- [数据类型](#数据类型)
- [接口列表](#接口列表)
  - [创建知识库](#post-newmoisemantic-models)
  - [数据侧创建空知识库](#post-newmoisemantic-modelscreate-empty)
  - [创建知识库并导入数据源](#post-newmoisemantic-modelscreate-with-sources)
  - [上传知识库本地文件（创建前）](#post-newmoisemantic-modelslocal-filesupload)
  - [预览创建时数据源选择计数](#post-newmoisemantic-modelssource-selectionspreview)
  - [列出语义模型](#get-newmoisemantic-models)
  - [列出知识库标签](#get-newmoisemantic-modelstags)
  - [更新语义模型](#put-newmoisemantic-modelsmodel_id)
  - [删除语义模型](#delete-newmoisemantic-modelsmodel_id)
- [获取语义模型](#get-newmoisemantic-modelsmodel_id)
- [列出知识库数据源](#get-newmoisemantic-modelsmodel_idsources)
- [查询知识库数据源存在性](#post-newmoisemantic-modelsmodel_idsourcesexistence)
- [预览追加数据源选择计数](#post-newmoisemantic-modelsmodel_idsource-selectionspreview)
- [上传知识库本地文件（追加前）](#post-newmoisemantic-modelsmodel_idlocal-filesupload)
- [追加知识库数据源](#post-newmoisemantic-modelsmodel_idsources)
- [补齐历史知识库数据源关系](#post-newmoisemantic-modelsmodel_idsourcesbackfill-legacy)
- [删除知识库数据源](#delete-newmoisemantic-modelsmodel_idsourcessource_row_id)
- [获取文档详情](#get-newmoisemantic-modelsmodel_idsourcessource_row_iddocument)
- [预览知识库原文档](#get-newmoisemantic-modelsmodel_idsourcesfilefile_idpreview)
- [预览知识库解析图片产物](#get-newmoisemantic-modelsmodel_idartifactsfile_idpreview)
  - [查询知识库 source jobs](#get-newmoisemantic-modelsmodel_idsource-jobs)
  - [同步知识库 source jobs](#post-newmoisemantic-modelsmodel_idsource-jobsreconcile)
  - [更新 source 治理](#patch-newmoisemantic-modelsmodel_idsourcessource_row_idgovernance)
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
  data: T | null;
  error?: CoreErrorInfo;
}

interface CoreErrorInfo {
  reason: string;
  domain: string;
  metadata: Record<string, string>;
}
```

当 moi-core structured error 被 moi-backend 转成非 2xx HTTP 错误时，响应会额外包含 `error` 字段，镜像上游稳定的 error reason / domain / metadata；`msg` 仍由 moi-backend 按请求 locale 渲染。backend 本地错误和 legacy core 错误不带 `error` 字段。

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

## IAM 权限

- 模型列表由 Core 按 `semantic_model.read` 在 count 和 pagination 前过滤；Backend 不读取旧知识库权限码拼接 ID 列表。
- 创建使用 workspace-scoped `semantic_model.create`；读取、修改、删除和使用分别绑定 `semantic_model.read/update/delete/use`。触发处理且修改持久内容的接口组合检查 `use + update`。
- 创建前 selection preview 使用 `semantic_model.create`，追加前 selection preview 使用目标模型的 `semantic_model.update`；两者还按请求中的 Database ID 校验 `database.read`，按请求中的 Volume ID 经 canonical root resolver 归根后校验 `volume.read`。source existence 只查询目标模型内的已添加状态，仅要求目标模型 `semantic_model.read`。
- 原文档和解析产物预览都使用目标模型的 `semantic_model.read`。Backend 按知识库工作流登记的文档关联确认原文档能到达目标模型的向量索引；产物先反查到该原文档，再使用同一关联判断。不会以 `knowledge_base_sources`、`knowledge_base_segments` 或请求方提供的 Volume 归属作为第二套预览边界。
- `/sources/file/:file_id/preview` 中的 `file_id` 是工作流原文档 File ID；`/sources/:source_row_id/...` 管理接口继续使用 source row ID。
- `create-with-sources` 和 append sources 在业务调用前检查已有依赖：`catalog_file` 必须携带权威 `volume_id`（文件所在的来源 Volume，不是知识库 raw Volume），Backend 将该 Volume 经 canonical root resolver 归根后要求 `volume.read`，不以 file-root containment 猜测多挂载文件的根；Catalog Table 只要求产品粗粒度 `table.read`，实际数据读取仍由 MatrixOne 以 Verified Effective Role 最终裁决；`source_selections` 按选择范围要求 database 的 `database.read` 或经 canonical root resolver 归根后的 volume `volume.read`；`sources` 与 `source_selections` 均为空的请求在 handler 前按参数错误拒绝。local file 在上传前不存在可授权资源，不伪造对象权限。
- Core owner service 会独立重验 action，并在创建/删除事务中维护 canonical Ownership lifecycle。任何 Core IAM、依赖解析、Ownership 或列表过滤不可用都会 fail closed。

---

## 枚举值

### SemanticKind（语义条目类型）

| 值                   | 说明              | 必填 spec 字段                              |
| -------------------- | ----------------- | ------------------------------------------- |
| `dimension`          | 维度列            | `column`                                    |
| `fact`               | 事实列            | `column`                                    |
| `metric`             | 业务指标          | `expr`                                      |
| `relationship`       | 表关联关系        | `left_table`, `right_table`, `join_columns` |
| `column_preference`  | 列别名/废弃列替换 | `preferred`, `deprecated`                   |
| `named_filter`       | 命名过滤条件      | `expr`                                      |
| `default_constraint` | 默认约束过滤      | `column`, `values`                          |
| `verified_query`     | 已验证的标准问答  | `question`, `sql`                           |
| `glossary`           | 业务术语解释      | `term`, `definition`                        |
| `logic_text`         | 自然语言规则注入  | `content`, `injection_stages`               |
| `sql_resultset`      | SQL 结果集        | `sql`, `description`                        |

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
  created_at: number;
  updated_at: number;
}

interface SemanticModelTagStat {
  tag: string;
  count: number;
}

interface ListSemanticModelTagsResponse {
  items: SemanticModelTagStat[];
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

interface CreateSemanticModelSourceRequest {
  source_type: "local_file" | "catalog_file" | "catalog_table";
  file_name?: string;
  upload_kind?: "structured" | "unstructured";
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
      kind: "database_tables";
      database_id: number;
      all_selected: boolean;
      selected_table_ids?: number[];
      excluded_table_ids?: number[];
      filters?: Pick<SemanticModelSourceSelectionFilters, "table_name">;
    }
  | {
      kind: "volume_files";
      volume_id: number;
      all_selected: boolean;
      selected_file_ids?: string[];
      excluded_file_ids?: string[];
      filters?: Pick<SemanticModelSourceSelectionFilters, "file_name" | "file_ext">;
    };

interface AppendSemanticModelSourcesRequest {
  sources: CreateSemanticModelSourceRequest[];
  source_selections?: SemanticModelSourceSelection[];
}

interface PreviewSemanticModelSourceSelectionsRequest {
  source_selections: SemanticModelSourceSelection[];
}

interface PreviewSemanticModelSourceSelectionsResponse {
  file_count: number;
  table_count: number;
  total_count: number;
}

interface KnowledgeBaseDataDomain {
  model_id: number;
  catalog_id: number;
  database_id: number;
  raw_volume_id: number;
  processed_volume_id: number;
  ensure_status: "ready" | "failed" | string;
  last_ensure_error?: string | null;
  last_checked_at: number;
}

interface SemanticModelSource {
  row_id: string;
  source_id?: string;
  source_type: "file" | "volume" | "table";
  model_id: number;
  resource_id: string;
  source_resource_id?: string;
  kb_resource_id?: string;
  display_name?: string | null;
  path: string[];
  source_path?: string | null;
  db_name?: string | null;
  table_name?: string | null;
  size_bytes?: number | null;
  row_count?: number | null;
  ingest_status?: string | null;
  enabled?: boolean | null;
  expires_at?: number | null;
  expired: boolean;
  effective_enabled: boolean;
  force_enabled_after_expiry: boolean;
  tags?: string[];
  segment_version_id?: string | null;
  index_version?: number | null;
  updated_at?: number | null;
  error?: string | null;
}

interface ListSemanticModelSourcesResponse {
  items: SemanticModelSource[];
  total: number;
  page: number;
  page_size: number;
  legacy_backfill_required?: boolean;
}

interface CheckSemanticModelSourceExistenceRequest {
  file_ids?: string[];
  table_ids?: number[];
}

interface CheckSemanticModelSourceExistenceResponse {
  file_ids: string[];
  table_ids: number[];
}

interface SemanticModelSourceDocument {
  source: SemanticModelSource;
  preview: {
    available: boolean;
    reason?: string | null;
  };
  file_info: {
    tags: string[];
    expires_at?: number | null;
    enabled?: boolean | null;
    expired: boolean;
    effective_enabled: boolean;
    force_enabled_after_expiry: boolean;
    index_version?: number | null;
    segment_version_id?: string | null;
  };
  segment_status: {
    available: boolean;
    reason?: string | null;
    total: number;
  };
  current_segment_version_id?: string | null;
  current_index_version?: number | null;
  selected_segment_version_id?: string | null;
  selected_index_version?: number | null;
  segment_versions: Array<{
    version_id: string;
    current: boolean;
    index_version?: number | null;
    base_version_id?: string | null;
    base_index_version?: number | null;
    status?: "pending" | "materializing" | "committed" | "failed" | string;
    source?: "initial_import" | "edit_chunk" | "create_chunk" | "disable_chunk" | "reembed" | string;
    chunk_count?: number;
    enabled_chunk_count?: number;
    created_at?: number;
    updated_at?: number;
  }>;
  segments: SemanticModelDocumentSegment[];
}

interface SemanticModelDocumentSegment {
  segment_id: string;
  segment_type: "text" | "image" | "table" | "transcript" | string;
  start_ms?: number;
  end_ms?: number;
  level: string;
  chunk_index?: number | null;
  chunk_id?: string | null;
  content?: string | null;
  ocr_text?: string | null;
  image_description?: string | null;
  image_file_id?: string | null; // 只读；由解析流程维护
  page_image_file_id?: string | null; // 只读；由解析流程维护
  word_count: number;
  recall_count: number;
  enabled: boolean;
  metadata?: { volume_id?: string | number };
}

interface SemanticModelSegmentMutationBase {
  base_segment_version_id: string | null;
  base_index_version: number | null;
}

interface SemanticModelSegmentMutationResult {
  document: SemanticModelSourceDocument;
}

interface UpdateSemanticModelSourceGovernanceRequest {
  tags?: string[];
  expires_at?: number | null;
  enabled?: boolean;
  force_enabled_after_expiry?: boolean;
}

interface KnowledgeBaseSourceJobRun {
  job_id: string;
  source_id: string;
  job_status: "queued" | "pending" | "running" | "succeeded" | "failed" | string;
  source_file_id?: string | null;
  kb_file_id?: string | null;
  source_table_id?: number | null;
  kb_table_id?: number | null;
  error?: string | null;
  updated_at?: number;
  reconcile_required?: boolean;
}

interface SemanticModelSourceJobListResponse {
  /** 最多 32 个 persisted active/incomplete source job 诊断视图。 */
  items: KnowledgeBaseSourceJobRun[];
  /** 全 KB persisted active/incomplete source 数量，不受 items 批次限制。 */
  total: number;
  /** 全 KB 是否仍需 POST reconcile；只有 legacy 工作时也可以为 true。 */
  reconcile_required: boolean;
}

interface AppendSemanticModelSourcesResponse {
  data_domain: KnowledgeBaseDataDomain;
  sources: SemanticModelSource[];
  jobs: KnowledgeBaseSourceJobRun[];
}
```

`local_file` 必须先通过 `/semantic-models/local-files/upload` 上传文件，再同时提供返回的 `file_id` 和原始 `file_name`。`file_name` 为空或 `file_id` 缺失、带首尾空白时，create 和 append 请求返回 `400 ErrParamInvalid`，并提示必须同时提供这两个字段；响应头 `X-Request-ID` 提供该失败的关联 ID。`content_base64` 已废弃；create 和 append 请求只要携带该字段就返回 `400 ErrParamInvalid`，调用方必须改用上传接口取得 `file_id`。首次绑定只接受尚未挂载的文件，已在当前知识库 raw Volume 的文件仅用于幂等重试，已在其他 Volume 的文件会被拒绝；这个关联条件不证明上传者身份，也不是一次性上传凭证，因此调用方只能原样使用本次上传接口返回的 `file_id`。已有 Catalog Volume 文件必须使用 `catalog_file`，并同时提供权威 `volume_id`（文件在 Catalog 中的来源 Volume）；Backend 按该 Volume 归根后校验 `volume.read`，service 再按 `(file_id, volume_id)` 校验文件归属。缺少 `volume_id` 时 create/append 在 handler 前返回 `400 ErrParamInvalid`。

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

**default_constraint:**

```json
{
  "column": "currency",
  "operator": "=",
  "values": ["CNY"],
  "gate_severity": "required"
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

创建知识库及其语义模型。成功返回前，后端会在 workspace 初始化时保留的默认 Catalog 下创建与知识库同名的物理数据库，以及 raw、processed 两个基础 Volume；`name` 必须是合法 Catalog 标识符，创建后不可修改。默认 Catalog 是 backend 保留的系统资源，后端通过 tenant reservation metadata 解析其 ID，不经过调用方的 `catalog.read` / `database.read` 列表权限；因此知识库创建准入不会因读取该系统 Catalog 而额外增加这两项权限要求。若同名物理数据库已存在且不属于该知识库已持久化的半创建 data-domain，返回 409，错误消息包含按请求语言生成的实际 Catalog/数据库路径。

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
    "created_at": 1712345678,
    "updated_at": 1712345678
  }
}
```

| 状态码 | 错误码            | 说明                                              |
| ------ | ----------------- | ------------------------------------------------- |
| 400    | `ErrParamInvalid` | name 为空或不是合法 Catalog 标识符                |
| 409    | `ErrConflict`     | 同名知识库或默认 Catalog 下同名数据库已存在 |

---

### POST /newmoi/semantic-models/create-empty

数据侧知识库页面使用的空知识库创建接口。请求只包含基本信息和可选的图片索引开关；后端创建语义模型、共享知识库 Catalog 下同名数据库、raw/processed Volume，并写入后端固定的文本向量索引配置。`image_index_enabled=true` 时还会写入固定图片索引配置。

该接口不会创建 source 元数据、source job、本地上传、Catalog 来源副本或 RAG 工作流。数据源仍应在知识库创建完成后经既有追加来源接口处理。

**权限:** workspace `semantic_model.create`。

**请求体:**

```json
{
  "name": "产品文档知识库",
  "description": "产品资料",
  "image_index_enabled": true
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
        "file_ids": [],
        "vector_table": "kb_7_text_index",
        "embedding_model": "bge-m3",
        "image_vector_table": "kb_7_image_index"
      }
    },
    "data_domain": {
      "model_id": 7,
      "catalog_id": 20001,
      "database_id": 11,
      "raw_volume_id": 12,
      "processed_volume_id": 13,
      "ensure_status": "ready"
    }
  }
}
```

---

### POST /newmoi/semantic-models/create-with-sources

创建知识库并同时提交初始数据源。新建知识库的 data-domain 固定使用 workspace 初始化时保留的默认 Catalog（与本地上传路径一致），不会把知识库 db/volume 挂到来源 Catalog 下。默认 Catalog 的解析使用 backend reservation metadata，不要求调用方拥有该系统 Catalog 的 `catalog.read` / `database.read` 权限；来源 `source_selections` 仍按其实际 database/volume 的读取权限校验。创建重试不会把已有 data domain 的正数 `catalog_id` 迁移到默认 Catalog；只有 data domain 已绑定 database 时，才按该 database 的真实父 Catalog 修复关联。请求不得再传 `target_catalog_id`（字段已从 API 移除；传入时返回 400 `ErrParamInvalid`，service 不会被调用）。`sources` 与 `source_selections` 至少一个非空，也可以同时传入；本地上传和结构化上传继续通过 `sources[].source_type="local_file"` 提交。Catalog 大批量选择使用 `source_selections` 表达 database 表叶子或 volume 文件叶子选择，后端按页展开、校验归属、去重并写入 source/job 元数据；文件 link/copy、表 clone、RAG 后续处理由知识库资源列表轮询 `source-jobs/reconcile` 有限批次推进，不在 create 请求内全量处理。

新建知识库固定使用文本模型 `bge-m3`。请求可选的 `image_index_enabled` 默认为 `false`；仅在其为 `true` 时，后端才额外固定使用图片模型 `efficientnet-b3`（1536 维、预处理版本 `efficientnet-b3-v1-rgb-300-letterbox-imagenet`、距离度量 `cosine`）并解析实际图片 backend id。`bge-m3` 不可用时始终返回 400；图片索引开启时 `efficientnet-b3` 不可用也返回 400，且不会创建语义模型。请求 `files` 可选，但其中的 `embedding_model`、`image_embedding_*`、`vector_table` 和 `image_vector_table` 均会被忽略；响应始终返回后端生成的文本向量表，仅在图片索引开启时返回图片索引配置。

`source_selections` 的 ID 必须为非空、无首尾空白且不重复的原始值；table ID 必须为正数且不重复。`filters.file_ext` 必须使用不带 `.` 的小写字母或数字扩展名且不重复。后端不会替调用方 trim、大小写转换或去重，非法值返回参数错误。

响应中的 `data.sources` / `data.jobs` 表示本次创建提交的数据源与作业。`data.model.source_counts` 不作为创建响应的权威来源计数；需要展示来源数量时应读取列表/详情接口返回的 `source_counts`。

完整图片 embedding 配置只表示该知识库具备图片索引能力。发布 segment version 时，如果模型已绑定后端生成的 `image_vector_table`，后端会按 `kb_file_id + index_version` 读取实际图片 rows；读到 rows 才生成图片段。当前解析文档没有 page image 或 visual object 时，segment version 可以只包含文本段。图片向量表不可见、schema 不匹配或读取失败仍是错误，不会被当作“无图片”成功。

`catalog_file` 初始数据源可以复用该 Catalog 文件已有的 vector lineage。文本向量以 `embedding_model` 为主要兼容边界；图片向量以 `image_embedding_model + image_embedding_dimension + image_preprocess_version + image_distance_metric` 为兼容边界，`image_embedding_backend_id` 不阻断复用。复用到当前知识库目标 vector table 时，如果 deterministic target row 已存在且 metadata 与当前绑定兼容，后端复用该 row；不兼容时返回错误，不会静默覆盖或吞掉 duplicate。

**请求体:**

```json
{
  "name": "产品文档知识库",
  "description": "产品资料",
  "image_index_enabled": false,
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
        "embedding_model": "bge-m3"
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

### POST /newmoi/semantic-models/local-files/upload

创建知识库前上传本地非结构化文件。multipart 字段名固定为 `file`，成功后返回尚未挂载的 catalog `file_id`，后续通过 `create-with-sources` 的 `sources[].source_type="local_file"` + `file_id` 绑定。调用方必须原样使用本次上传返回的 ID；绑定阶段只检查文件未关联其他 Volume，不把任意 `file_id` 视为上传者凭证。

**权限:** workspace `semantic_model.create`（与 create-with-sources 对齐）。

**请求:** `multipart/form-data`

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `file` | file | 是 | 原始文件内容；文件名取自 multipart filename |

**响应 data:**

```ts
{ file_id: string }
```

| 状态码 | 错误码 | 说明 |
| --- | --- | --- |
| 400 | `ErrParamInvalid` | 缺少 file 或文件名为空 |
| 403 | `ErrForbidden` | 当前 principal 无 workspace `semantic_model.create` |
| 500 | `ErrServer` | 上传失败 |

---

### POST /newmoi/semantic-models/source-selections/preview

在创建知识库前预览 `source_selections` 的权威去重计数。后端复用创建提交的校验、分页展开、`seenFiles` / `seenTables` 去重及 exclusions 语义；该接口只读，不创建 source 或 job。

**权限:** workspace `semantic_model.create`，并要求每个 selection 对应的 `database.read` 或 canonical root `volume.read`。

**请求体:** `PreviewSemanticModelSourceSelectionsRequest`；`source_selections` 必须非空。

**响应 200:** `ApiResponse<PreviewSemanticModelSourceSelectionsResponse>`。

**错误:** JSON 或 selection 字段非法时返回 `400 ErrParamInvalid`；筛选无匹配或 Catalog 依赖失败时沿用最终提交的 service 错误映射。

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

### GET /newmoi/semantic-models/tags

聚合列出当前用户可见知识库的标签。Core 先按 `semantic_model.read` 做 collection filter：拥有 workspace 级读取可见性时结果覆盖当前 workspace 的全部知识库；仅拥有对象级 `semantic_model.read` 时只聚合获授权知识库。没有任何获授权知识库时返回空列表，不返回未授权知识库的标签或计数。

**查询参数:**

| 参数     | 类型   | 必填 | 说明                                  |
| -------- | ------ | ---- | ------------------------------------- |
| `search` | string | 否   | 按知识库名称或描述筛选后再聚合标签。 |

**响应 200:** `ApiResponse<ListSemanticModelTagsResponse>`。`items` 按 `count` 降序排列；相同 `count` 时按 `tag` 升序排列。单个知识库内重复的同一标签只计一次。

---

### PUT /newmoi/semantic-models/:model_id

更新语义模型元信息或表/文件配置。`name` 创建后不可修改：请求体中的 `name` 必须与当前知识库名称完全一致（与 Catalog 物理数据库名对齐）；改名返回 `400 ErrParamInvalid`。仅更新描述时，请求体仍须传当前 `name` 与 `description`；省略的 `tables` 和 `files` 保留原有配置。更新包含文件并需要补齐知识库 data domain 时，已有正数 `catalog_id` 不会迁移到共享 Catalog；已绑定 database 的关联只按该 database 的真实父 Catalog 修复。

**请求体:**

```json
{
  "name": "销售分析模型",
  "description": "更新后的描述"
}
```

需要更新模型表或文件配置时，才额外传入对应字段：

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

| 状态码 | 错误码            | 说明                                                         |
| ------ | ----------------- | ------------------------------------------------------------ |
| 400    | `ErrParamInvalid` | name 为空、与当前名称不一致（不可改名）、或 tables 格式不合法 |
| 404    | `ErrNotFound`     | 模型不存在                                                   |

---

### DELETE /newmoi/semantic-models/:model_id

删除知识库本身、语义模型条目及 backend 关系数据。该接口会清理该知识库下的 `knowledge_base_segment_versions`、`knowledge_base_segments`、`knowledge_base_chunk_recall_stats`、source/job 关系、raw volume 绑定和 data domain 记录；Catalog 资源删除仍走知识库 data domain owner 边界。Catalog 删除事务内还会同步清理当前工作区普通智能体绑定与系统/共享智能体覆盖绑定中的目标知识库引用，并为引用该知识库名称的非 disabled Agent Package 版本追加 `knowledge_base_deleted` 诊断并转为 `needs_configuration`。成功响应仍为 `{deleted: true}`。单个数据源删除使用 `DELETE /newmoi/semantic-models/:model_id/sources/:source_row_id`。

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

以二进制流预览知识库关联的原文档。路径参数 `file_id` 是原文档 File ID，不是知识库 source 记录 ID。服务端按工作流登记的文档关联确认该原文档能到达目标模型的向量索引；不会接受 `volume_id`，也不以治理 source 记录作为归属边界。

**权限:** 目标知识库 `semantic_model.read`。现有 PEP 对 SuperAdmin 的放行保持不变；其他 principal 必须拥有该目标模型的读取权限。

**路径参数:**

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `model_id` | 正整数 | 是 | 目标知识库 / semantic model ID |
| `file_id` | string | 是 | 非空且无首尾空白的原文档 File ID |

**响应 200:** 二进制流，不使用 `{ code, msg, data }` 信封。

- `Content-Type`: 透传文件 MIME；缺省为 `application/octet-stream`
- `Content-Disposition`: 文件名可用时返回 `inline; filename*=UTF-8''<encoded-name>`
- Office 文档（`doc/docx/ppt/pptx/xls/xlsx`）与 Catalog `preview_stream` 对齐：先转 PDF 再返回（`Content-Type: application/pdf`）
- `.zip` 源文档与 Catalog 对齐：抽取包内 markdown 后以 `text/markdown` 返回

**错误:**

| HTTP | code | 场景 |
| --- | --- | --- |
| 400 | `ErrParamInvalid` | `model_id` 非正整数，或原文档 `file_id` 为空/带首尾空白 |
| 403 | `ErrForbidden` | 当前 principal 无目标模型的 `semantic_model.read` |
| 404 | `ErrNotFound` | `file_id` 不是通过工作流关联到目标模型向量索引的原文档 |
| 500 | `ErrServer` | 工作流关联查询、Tenant DB、Office 转换或底层文件流不可用 |

---

### GET /newmoi/semantic-models/:model_id/artifacts/:file_id/preview

以二进制流预览知识库解析产生的文档产物。`file_id` 必须通过工作流登记的解析、派生产物或工作流输出关系反查到原文档，且该原文档能到达目标模型的向量索引；仅知道一个 Catalog File ID 不足以访问该内容。图片引用支持裸 UUID 或大小写不敏感的 `.png`、`.jpg` / `.jpeg`、`.gif`、`.webp`、`.bmp`、`.tif` / `.tiff` 后缀。

**权限:** 目标知识库 `semantic_model.read`。这是 semantic model 从属产物读取，不使用通用 Catalog File preview，也不接收 `volume_id`。

**路径参数:**

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `model_id` | 正整数 | 是 | 目标知识库 / semantic model ID |
| `file_id` | string | 是 | 非空且无首尾空白的解析图片 Catalog File ID |

**响应 200:** 原始二进制流，不使用 `{ code, msg, data }` 信封。

- `Content-Type`: 透传文件 MIME；缺省为 `application/octet-stream`
- `Content-Disposition`: 文件名可用时返回 `inline; filename*=UTF-8''<encoded-name>`

**错误:**

| HTTP | code | 场景 |
| --- | --- | --- |
| 400 | `ErrParamInvalid` | `model_id` 非正整数，或 `file_id` 为空/带首尾空白 |
| 403 | `ErrForbidden` | 当前 principal 无目标模型的 `semantic_model.read` |
| 404 | `ErrNotFound` | `file_id` 不能经工作流文档关联反查到目标模型的原文档 |
| 500 | `ErrServer` | 关联查询、Tenant DB 或底层文件流不可用 |

---

### GET /newmoi/semantic-models/:model_id/sources

列出知识库数据源。未传分页参数时保持兼容并返回全部 source；显式传入 `page` 或 `page_size` 时分页。返回顺序为已登记 active source，再追加可显式 backfill 的 legacy candidate；`total` 是分页前可展示总数。单个 source 的文件或表元数据不可用时，不会让整个接口 500；该 source 会保留在 `items` 中并返回 `ingest_status="failed"` 和 `error`，前端可展示失败原因并允许用户删除该 source。仍在本地上传中的 pending 文件如果 Catalog volume metadata 暂未就绪，会继续显示为 pending，不误标 failed。

**权限:** 知识库对象 `view`

**查询参数:**

| 参数        | 类型 | 必填 | 说明                       |
| ----------- | ---- | ---- | -------------------------- |
| `page`      | int  | 否   | 页码，从 1 开始；启用分页时默认 1  |
| `page_size` | int  | 否   | 每页数量，1-100；启用分页时默认 20 |

**响应 200:**

```json
{
  "code": "OK",
  "msg": "OK",
  "data": {
    "items": [],
    "total": 0,
    "page": 1,
    "page_size": 20
  }
}
```

**错误:**
| HTTP | code | 场景 |
|------|------|------|
| 400 | `ErrParamInvalid` | model_id、page 或 page_size 非法 |
| 404 | `ErrNotFound` | 知识库不存在或尚未创建语义模型 |
| 500 | `ErrServer` | 读取 source 列表、作业列表或系统性依赖失败 |

---

### POST /newmoi/semantic-models/:model_id/sources/existence

`file_ids` / `table_ids` 必须使用非空、无首尾空白、有效且不重复的原始 ID；非法值返回参数错误，不会被静默修复。

按当前页 Catalog 文件 / 表 id 查询哪些已经存在于该知识库，用于选择器分页页内“已添加”禁用回显。该接口只匹配 active `knowledge_base_sources`：`source_type` 为 `catalog_file` 或 `catalog_table`，且 `status != "removed"`；已删除的 source 不会返回，用户可重新添加。

**权限:** 目标知识库 `semantic_model.read`。请求中的 File ID / Table ID 仅用于查询该模型内的已添加状态，不作为新的 IAM 资源。

**请求体:**

```json
{
  "file_ids": ["file-1", "file-2"],
  "table_ids": [1001, 1002]
}
```

**响应 200:**

```json
{
  "code": "OK",
  "msg": "OK",
  "data": {
    "file_ids": ["file-1"],
    "table_ids": [1002]
  }
}
```

**错误:**
| HTTP | code | 场景 |
|------|------|------|
| 400 | `ErrParamInvalid` | model_id 非法或 JSON body 非法 |
| 500 | `ErrServer` | 查询 source 关系失败 |

---

### POST /newmoi/semantic-models/:model_id/source-selections/preview

在追加数据源前预览权威去重计数。展开语义与创建预览相同，并复用追加提交的现有 source 排除逻辑，因此返回实际可新增的文件、表及总数。该接口只读。

**权限:** 目标知识库 `semantic_model.update`，并要求每个新增 selection 对应的 `database.read` 或 canonical root `volume.read`。

**请求体:** `PreviewSemanticModelSourceSelectionsRequest`；`source_selections` 必须非空。

**响应 200:** `ApiResponse<PreviewSemanticModelSourceSelectionsResponse>`。

**错误:** `model_id`、JSON 或 selection 字段非法时返回 `400 ErrParamInvalid`；筛选无匹配或依赖查询失败时沿用最终追加的 service 错误映射。

---

### POST /newmoi/semantic-models/:model_id/local-files/upload

向已有知识库追加本地非结构化文件前上传。multipart 字段名固定为 `file`，成功后返回尚未挂载的 catalog `file_id`，后续通过 `POST /semantic-models/:model_id/sources` 的 `local_file` + `file_id` 绑定。调用方必须原样使用本次上传返回的 ID；绑定阶段只检查文件未关联其他 Volume，不把任意 `file_id` 视为上传者凭证。

**权限:** 目标知识库 `semantic_model.update`（与 append sources 的 update 侧对齐）。

**请求:** `multipart/form-data`

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `file` | file | 是 | 原始文件内容；文件名取自 multipart filename |

**响应 data:**

```ts
{ file_id: string }
```

| 状态码 | 错误码 | 说明 |
| --- | --- | --- |
| 400 | `ErrParamInvalid` | model_id 非法，或缺少 file / 文件名为空 |
| 403 | `ErrForbidden` | 当前 principal 无目标模型的 `semantic_model.update` |
| 500 | `ErrServer` | 上传失败 |

---

### POST /newmoi/semantic-models/:model_id/sources

向已有知识库追加本地文件、Catalog 文件、Catalog 表数据源，或追加 `source_selections` 表达的 database 表叶子 / volume 文件叶子选择。`sources` 与 `source_selections` 至少一个非空，也可以同时传入；本地上传和结构化上传继续走 `sources.local_file`。`source_selections` 只在请求内分页展开并写入 source/job 元数据，后续文件 link/copy、表 clone、RAG 处理由资源列表轮询 `source-jobs/reconcile` 有限批次推进。

追加时不会把已有 data domain 的正数 `catalog_id` 迁移到共享 Catalog；已绑定 database 的关联只按该 database 的真实父 Catalog 修复。修复只更新现存 domain 的 `catalog_id`，不会重新插入已被并发删除的 domain。

同一知识库重复选择已有 Catalog 文件或 Catalog 表时，后端复用已存在的治理 source 与 KB 资源，不重新复制文件、不重新 clone 表、不覆盖 `enabled`、`expires_at`、`tags`、`force_enabled_after_expiry`、`segment_version_id/index_version` 等治理字段；semantic model 的 `files.file_ids` 和 `tables[].table_names` 会按原顺序去重。该复用以已绑定的 `kb_file_id` 或 `kb_table_id/db_name/table_name` 为边界，不要求 source 已完成解析；后续 workflow/RAG job 可通过作业列表和显式 reconcile/重试继续推进。历史上已经存在的重复 source 只展示和保留，不在追加时自动合并。

已删除的数据源关系会作为 `removed` source 保留在治理表中，但不出现在列表、详情或检索范围里。用户显式再次追加同一 Catalog 文件或表时，后端会重新激活该 source、重新创建 jobs，并按当前知识库绑定复用已有 Catalog 文件向量或重新触发后续处理。

Catalog 文件向量复用的兼容规则与创建接口一致：文本优先按 `embedding_model`；legacy 文本 row 没有模型 metadata 时按 `vector_table` 或空 metadata 兼容；图片按 `image_embedding_model + image_embedding_dimension + image_preprocess_version + image_distance_metric`，不因 `image_embedding_backend_id` 不同而拒绝。目标 row 已存在时必须 metadata 兼容才复用，否则请求失败。

**权限:** 知识库对象 `update`

**请求体:**

```json
{
  "sources": [
    { "source_type": "catalog_file", "file_id": "file_001", "volume_id": 3001 },
    { "source_type": "catalog_table", "table_id": 1001 }
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
      "model_id": 77,
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

**错误:**
| HTTP | code | 场景 |
|------|------|------|
| 400 | `ErrParamInvalid` | model_id 非法、sources 为空、source 字段不完整或仍携带已废弃的 `content_base64` |
| 404 | `ErrNotFound` | 语义模型或知识库 data domain 不存在 |
| 409 | `ErrConflict` | 无法安全修复 data domain 的 Catalog 关联，或修复期间 domain 已被删除 |
| 500 | `ErrServer` | data domain 未 ready、ensure failed、追加 source、运行 job 或重读持久化状态失败 |

---

### POST /newmoi/semantic-models/:model_id/sources/backfill-legacy

显式补齐历史 workflow/lineage 关联到知识库的数据源关系。该接口只写 `knowledge_base_sources` 和 `knowledge_base_source_job_runs` 缺失关系，不导入 chunk，不迁移 vector rows；`GET /sources`、`GET /source-jobs` 和文档详情 GET 不会触发该写操作。

已有 active `knowledge_base_sources` 行会保留原 `source_id` 和治理字段：`enabled`、`expires_at`、`tags`、`force_enabled_after_expiry`。已移除关联的 source 行会作为不可见阻断记录保留，自动 backfill 不会重新接入；用户显式再次添加同一 Catalog 文件或表时才会重新激活关系。缺 source 行时，后端按历史 job 的 `kb_file_id` 创建治理 source；如果只有 legacy `source_file_id`，则用该 file id 作为 KB 治理身份。对于已存在于 KB raw volume 但尚未登记 source 的历史文件，后端按 `kb_file_id` 分批接管为治理 source；只有旧行缺 `kb_file_id` 时才用 `source_file_id` 做兼容匹配。对于非知识库默认 workflow 但 lineage 指向当前知识库 `vector_table/image_vector_table` 的历史解析产物，后端按 lineage 文件资产创建 `legacy_unbound` 候选并由本接口接入治理。

该接口是幂等的：重复调用或并发调用命中已存在 source/job-run 时，不创建第二条关系、不覆盖治理字段。单次调用只处理有限批次；调用方需要重新 `GET /sources`，如果仍返回 `legacy_backfill_required=true` 或 `legacy_unbound` 候选行，再继续调用下一批。

**权限:** 知识库对象 `update`

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

**错误:**
| HTTP | code | 场景 |
|------|------|------|
| 400 | `ErrParamInvalid` | model_id 非法或历史 source/job 数据不完整 |
| 404 | `ErrNotFound` | 知识库不存在 |
| 500 | `ErrServer` | tenant DB 不可用、Catalog 文件元数据缺失或补齐写入失败 |

---

### DELETE /newmoi/semantic-models/:model_id/sources/:source_row_id

删除一个知识库数据源。该操作会更新 semantic model 的 source scope，清理该 source 的 `knowledge_base_segment_versions`、`knowledge_base_segments`、`knowledge_base_chunk_recall_stats` 和 source/job 治理关系，并把 source 标记为 `removed`；删除成功后，数据源列表不再返回该 source。

如果该 source 仍有未超时的 `running` job，接口返回 `409 ErrConflict`，且不会修改 semantic model 或 source/job 关系。前端可在当前处理结束后重试删除。

文件 source 删除只移除知识库关联，不删除 Catalog 文件，也不禁用或物理删除已写入的文本/图片向量 rows；这些 rows 会保留给后续显式重加同一 Catalog 文件时复用。表 source 删除会删除 backend 为知识库 clone 出来的 KB table；原始 Catalog 表不删除。

前端如果将该操作展示为“删除数据源”，语义仍是删除当前知识库里的 source 关系；不要向用户暗示原始 Catalog 文件、原始 Catalog 表或文件向量 rows 会被删除。

**权限:** 知识库对象 `update`

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
| 409 | `ErrConflict` | source 仍有未超时的 `running` job；本次删除不产生任何变更 |
| 404 | `ErrNotFound` | 语义模型或 source row 不存在 |
| 500 | `ErrServer` | 更新 semantic model scope、删除 KB 表或删除关系行失败 |

---

### GET /newmoi/semantic-models/:model_id/sources/:source_row_id/document

获取非结构化文档 source 的详情。默认返回当前生效分段版本；传 `segment_version_id` 时只查看该历史版本，不改变当前生效指针。预览内容由所选版本的 enabled chunks 组装，后端不会从旧 parsed artifact 兜底伪造 chunk。

**权限:** `semantic_model.read`

**Query Params:**
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| segment_version_id | string | 否 | 查看指定已存在分段版本；不发布、不更新 source 当前指针 |

**响应 200:**

```json
{
  "code": "OK",
  "msg": "OK",
  "data": {
    "source": {
      "row_id": "source-file-1",
      "source_id": "source-file-1",
      "source_type": "file",
      "model_id": 77,
      "resource_id": "kb-file-1",
      "display_name": "doc.pdf",
      "path": ["raw", "doc.pdf"],
      "ingest_status": "succeeded",
      "enabled": true,
      "expires_at": 1712345678,
      "expired": false,
      "effective_enabled": true,
      "force_enabled_after_expiry": false,
      "tags": ["finance"],
      "segment_version_id": "seg-v1",
      "index_version": 12
    },
    "preview": {
      "available": true,
      "content": "第一段内容"
    },
    "file_info": {
      "tags": ["finance"],
      "expires_at": 1712345678,
      "enabled": true,
      "expired": false,
      "effective_enabled": true,
      "force_enabled_after_expiry": false,
      "index_version": 12,
      "segment_version_id": "seg-v1"
    },
    "segment_status": {
      "available": true,
      "total": 1
    },
    "current_segment_version_id": "seg-v1",
    "current_index_version": 12,
    "selected_segment_version_id": "seg-v1",
    "selected_index_version": 12,
    "segment_versions": [
      {
        "version_id": "seg-v1",
        "current": true,
        "index_version": 12,
        "status": "committed",
        "source": "initial_import",
        "chunk_count": 1,
        "enabled_chunk_count": 1
      }
    ],
    "segments": [
      {
        "segment_id": "seg-1",
        "segment_type": "transcript",
        "start_ms": 0,
        "end_ms": 1250,
        "level": "chunk",
        "chunk_index": 0,
        "content": "第一段内容",
        "word_count": 5,
        "recall_count": 3,
        "enabled": true
      }
    ]
  }
}
```

**错误:**
| HTTP | code | 场景 |
|------|------|------|
| 400 | `ErrParamInvalid` | model_id 或 source row ID 非法，或 source 是表 |
| 404 | `ErrNotFound` | 语义模型或 source row 不存在 |
| 500 | `ErrServer` | 读取 tenant DB 或语义模型失败 |

---

### POST /newmoi/semantic-models/:model_id/sources/:source_row_id/segments/import-initial

显式从既有 vector rows 导入初始分段版本。`GET document` 不会隐式写库；找不到可导入 rows 时返回 400。

**权限:** 知识库对象 `update`

**请求体:**

```json
{
  "base_segment_version_id": null,
  "base_index_version": null
}
```

**响应 200:** `data` 为 `SemanticModelSegmentMutationResult`。

**错误:**
| HTTP | code | 场景 |
|------|------|------|
| 400 | `ErrParamInvalid` | 缺少 vector_table/embedding_model、vector schema 不兼容、无可导入 rows，或已存在初始分段版本 |
| 409 | `ErrConflict` | base 指针与当前 source 指针不一致 |

---

### PATCH /newmoi/semantic-models/:model_id/sources/:source_row_id/segments/:segment_id

编辑当前版本中的单个 chunk 文本、OCR 或图片描述。成功后复制当前版本、应用修改、重新 embedding enabled chunks，生成新的 `segment_version_id/index_version` 并更新 source 当前指针。

**权限:** 知识库对象 `update`

**请求体:**

```json
{
  "base_segment_version_id": "seg-v1",
  "base_index_version": 12,
  "content": "更新后的文本",
  "ocr_text": "更新后的 OCR",
  "image_description": "更新后的图片描述"
}
```

未出现的字段保持原值；保存失败不更新当前指针，不删除旧版本 rows。

**响应 200:** `data` 为 `SemanticModelSegmentMutationResult`。

---

### POST /newmoi/semantic-models/:model_id/sources/:source_row_id/segments

基于当前版本新建 chunk。成功后分配新的 `chunk_index`，物化新版本 rows，并更新当前指针。

**权限:** 知识库对象 `update`

`image_file_id` 和 `page_image_file_id` 由解析流程维护，手工创建 chunk 不能设置。请求中字段缺失或为 JSON `null` 表示不声明解析产物；任一字段被设置为字符串时（包括空串或纯空白）均返回 `400 ErrParamInvalid`，且在 Tenant DB、Core client、embedding 或文件服务调用前拒绝。

**请求体:**

```json
{
  "base_segment_version_id": "seg-v1",
  "base_index_version": 12,
  "level": "chunk",
  "content": "新增分段文本",
  "ocr_text": null,
  "image_description": null
}
```

**响应 200:** `data` 为 `SemanticModelSegmentMutationResult`。

**错误:**
| HTTP | code | 场景 |
|------|------|------|
| 400 | `ErrParamInvalid` | `image_file_id` 或 `page_image_file_id` 被手工设置，或其他创建参数无效 |
| 404 | `ErrNotFound` | 语义模型或 source row 不存在 |
| 409 | `ErrConflict` | base 指针与当前 source 指针不一致 |

---

### PATCH /newmoi/semantic-models/:model_id/sources/:source_row_id/segments/:segment_id/enabled

启用或禁用当前版本中的单个 chunk。成功后生成新分段版本；disabled chunk 不会写入新 vector rows，也不会被当前版本 RAG/visual/expand 路由召回。

**权限:** 知识库对象 `update`

**请求体:**

```json
{
  "base_segment_version_id": "seg-v1",
  "base_index_version": 12,
  "enabled": false
}
```

**响应 200:** `data` 为 `SemanticModelSegmentMutationResult`。

---

### DELETE /newmoi/semantic-models/:model_id/sources/:source_row_id/segments/:segment_id

删除当前版本中的单个 chunk。成功后复制当前版本、移除目标 chunk、重新 embedding enabled chunks，生成新的 `segment_version_id/index_version` 并更新 source 当前指针。历史版本 rows 保留，可继续通过 `GET document?segment_version_id=...` 查看删除前内容。

**权限:** 知识库对象 `update`

**请求体:**

```json
{
  "base_segment_version_id": "seg-v1",
  "base_index_version": 12
}
```

**响应 200:** `data` 为 `SemanticModelSegmentMutationResult`。

**错误:**
| HTTP | code | 场景 |
|------|------|------|
| 400 | `ErrParamInvalid` | segment_id 非法、base 指针缺失、删除后版本为空，或 source 是表 |
| 404 | `ErrNotFound` | 语义模型、source row 或 segment 不存在 |
| 409 | `ErrConflict` | base 指针与当前 source 指针不一致 |

---

### POST /newmoi/semantic-models/:model_id/sources/:source_row_id/segments/re-embedding

基于当前 committed segment version 重新 embedding。只使用当前版本 chunks、当前绑定 vector table 和 embedding model；不重跑原解析 workflow，不调用 go-worker 作为阶段 5 mutation owner。

**权限:** 知识库对象 `update`

**请求体:**

```json
{
  "base_segment_version_id": "seg-v1",
  "base_index_version": 12
}
```

缺少 vector table、缺少 embedding model、vector schema 缺必要列、embedding 列类型/维度不兼容、embedding 调用失败或写 vector rows 失败时，请求失败且不更新当前指针。

**响应 200:** `data` 为 `SemanticModelSegmentMutationResult`。

---

### PATCH /newmoi/semantic-models/:model_id/sources/:source_row_id/segment-versions/:version_id/current

把已 committed 的历史分段版本设为当前生效版本。该操作只更新 `knowledge_base_sources.segment_version_id/index_version`，不新建 segment version，不生成新的 `index_version`。

**权限:** 知识库对象 `update`

**请求体:**

```json
{
  "base_segment_version_id": "seg-v2",
  "base_index_version": 13
}
```

**响应 200:** `data` 为 `SemanticModelSegmentMutationResult`。

**错误:**
| HTTP | code | 场景 |
|------|------|------|
| 400 | `ErrParamInvalid` | version_id 非法或目标版本不是 committed |
| 404 | `ErrNotFound` | source row 或 segment version 不存在 |
| 409 | `ErrConflict` | base 指针与当前 source 指针不一致 |

---

### GET /newmoi/semantic-models/:model_id/source-jobs

只读返回全知识库 source job 工作信号。`items` 最多包含 32 个 persisted active/incomplete source 的诊断视图，`total` 是同一候选条件下的全量 persisted source 数；完整历史终态 job 不进入候选，也不会调用外部 workflow/import API enrichment。

顶层 `reconcile_required` 是前端调用 POST reconcile 的唯一信号。它覆盖 persisted `pending/queued/running`、终态未收口，以及 semantic model 显式 file/table、legacy job、raw-volume、lineage 等尚未持久化的 backfill，因此允许 `items=[]、total=0、reconcile_required=true`。GET 不创建或更新 source/job，不触发外部任务。

**权限:** 知识库对象 `read`

**请求体:** 无。

**响应 200:** `data` 为 `SemanticModelSourceJobListResponse`。

---

### POST /newmoi/semantic-models/:model_id/source-jobs/reconcile

显式刷新知识库 source jobs，并在 KB 文档解析作业成功后从当前 KB vector binding 中已写入的 vector rows 发布新的 chunk version。该接口处理 backend 已预先创建并持久化的 `rag_ingest` pending/queued/running/succeeded source job 绑定；`GET /sources` 和 `GET /source-jobs` 不会触发该写操作。

**运行时身份（job-frozen principal）：** source job 在创建时写入并冻结 `runtime_actor_moi_user_id`、`runtime_effective_role_id`（create-time MOI user + `VerifiedEffectiveRole`）。可选 `runtime_is_workspace_owner` 仅作审计，**不**在 deferred 路径恢复 privilege-class 旁路。`rag_ingest` 缺 actor/role 会在创建阶段失败，避免 reconcile 继承 callback/system 身份。延迟 dispatch（claim/Trigger）先 rehydrate 冻结 actor（`coreclient.WithIdentity`）+ 冻结 role，并清掉 `IsWorkspaceOwner` / `BusinessActionAuthorized` / allow facts，再按该身份 **重新进 Core** 鉴权：先 `semantic_model.use`（供内置 KB workflow / vector reuse）；`catalog_file` 优先尝试 reuse，仅在需要 workflow dispatch 时再对记录 volume 做一次 `volume.read` 门禁（allow 不跨动作复用）。deny 永久失败，Core 不可用可重试。写时 outer allow 不会被复用。历史无冻结行 fail-closed，**不做** reconcile 补写/adopt；被动 reconcile/轮询不得用本次 HTTP 调用者替换 job 身份。

刷新 RAG job 时后端可以按 source 绑定的 `kb_file_id` 查询最新 file execution，但 completed execution 只表示同一文件有完成过的解析流程，不直接表示当前 KB source 已完成。只有该 execution/lineage 产生的 rows 落在当前 KB 的 vector table 和 embedding model 绑定内，且 `index_version` 可发布时，后端才发布新的 segment version 并切换 source 当前指针。同一 `index_version` 已是当前版本时只幂等标记 job succeeded，不重复发布 segment version；没有当前 KB rows 或 binding 不匹配且没有当前 KB 正在运行的 execution 时，后端会触发默认 KB workflow。

前端只以 `GET /source-jobs` 顶层 `reconcile_required` 决定是否调用本接口，不根据 `items` 推导。每次 POST 面向全 KB 独立选批，GET 返回的 source/job ID 和数量不会限制本轮推进范围；`running` 任务也通过 POST 按最久未检查顺序观察并更新检查时间，不使用 cursor。已完整成功、已持久化失败和 removed source 不再要求 reconcile。

文件挂载的健康路径按目标 Volume 批量执行。批量 400/404 等终态错误才逐文件定位，坏文件不会阻塞同批其他文件；403、服务不可用和网络错误会把本次认领的对应 job 从 `running` 条件释放回 `queued`，本次 POST 返回错误，下一次 POST 可立即重试。该路径采用 at-least-once 语义并依赖 `AddFiles` 幂等，不提供 attempt fencing。30 分钟 lease 只恢复请求中断或释放失败后遗留的 `running` job；它不会后台定时推进，页面未触发 POST 时 job 状态不会自动变化。

**权限:** 知识库对象 `use` 和 `update`

**请求体:** 无。

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

**语义:**
- 成功解析：生成新的 `segment_version_id/index_version` 并更新 source 当前指针。
- completed file execution 不是完成态证据；source 完成必须以当前 KB `segment_version_id/index_version` 指针为准。
- 解析失败：标记 source/job failed，记录错误；不切当前指针，不修改 `enabled`。
- 无新解析版本：返回成功，不生成重复版本。
- 可继续处理：已持久化且保留为 `pending`/`queued`/`running`/`succeeded` 的 `rag_ingest` job 会在该接口中继续刷新；已明确标记 `failed` 的 job 不会被该接口自动重试。

**错误:**
| HTTP | code | 场景 |
|------|------|------|
| 400 | `ErrParamInvalid` | model_id 非法或解析产物不满足导入要求 |
| 403 | `ErrForbidden` | 当前调用身份没有知识库 raw Volume 的 `volume.write` |
| 404 | `ErrNotFound` | 语义模型或 source job 关联资源不存在 |
| 500 | `ErrServer` | 刷新 job 状态或发布 segment version 失败 |

---

### PATCH /newmoi/semantic-models/:model_id/sources/:source_row_id/governance

更新 source 的治理状态。文件 source 允许更新 `enabled/expires_at/tags/force_enabled_after_expiry`，其 effective 规则为 `enabled && (!expired || force_enabled_after_expiry)`；只有 effective 状态发生变化时才同步文本/图片向量表的 `disabled` 标记，同步失败则请求失败并回滚。仅更新 tags 或有效期不会依赖向量表。表 source 允许更新 `enabled/expires_at`；请求体出现 `tags` 或显式 `force_enabled_after_expiry` 时返回 400，且不做 vector disabled 同步。表 source 已过期后重新启用时，后端会记录强制启用状态以保持 effective 规则一致。

**权限:** 知识库对象 `update`

**请求体:**

```json
{
  "tags": ["finance", "policy"],
  "expires_at": 1712345678,
  "enabled": false,
  "force_enabled_after_expiry": true
}
```

文件和表 source 的 `expires_at: null` 表示清除有效期。未出现的字段保持原值。

**响应 200:**

```json
{
  "code": "OK",
  "msg": "OK",
  "data": {
    "source": {
      "row_id": "source-file-1",
      "source_type": "file",
      "model_id": 77,
      "resource_id": "kb-file-1",
      "enabled": false,
      "expires_at": 1712345678,
      "expired": false,
      "effective_enabled": false,
      "force_enabled_after_expiry": true,
      "tags": ["finance", "policy"],
      "index_version": 12
    }
  }
}
```

**错误:**
| HTTP | code | 场景 |
|------|------|------|
| 400 | `ErrParamInvalid` | model_id 或 source row ID 非法，JSON 字段类型错误，或表 source 请求体包含 tags/force_enabled_after_expiry |
| 404 | `ErrNotFound` | 语义模型或 source row 不存在 |
| 500 | `ErrServer` | 治理更新或 vector disabled 同步失败 |

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

仅可向空语义模型批量导入。模型已有任意条目时，接口拒绝导入；空模型按请求顺序逐条创建，
首条失败时立即返回，已创建的条目不会回滚。导入请求会将当前 `X-Request-ID` 和 `X-Trace-ID`
透传给 moi-core；当 core 返回结构化语义校验错误时，保持统一响应 envelope，在 `error` 中镜像
`reason`、`domain` 和 `metadata`，并用 backend locale 渲染 `msg`。响应头的 `X-Request-ID` 可用于
查询同一链路的 backend/core 日志。

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
| 400 | `ErrParamInvalid` | entries 为空、校验失败或模型已有条目 |
| 404 | `ErrNotFound` | 知识库不存在 |
| 409 | `ErrConflict` | key 重复 |

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
    "errors": [
      "semantic model validation failed"
    ]
  }
}
```

`errors[]` 为用户可见文案。来自 moi-core 的 structured `reason/domain/metadata` 会由 moi-backend 按请求 locale 映射到 backend i18n catalog 后返回；非结构化 legacy core 错误保持原始兼容行为。

当前 `SESSION_VALIDATION_FAILED` 是顶层 summary reason，返回“语义模型校验失败”这类摘要文案，不包含 entry 路径、字段名或上游 raw diagnostic。需要字段级错误列表时，后端需要等 moi-core 输出更具体的稳定 reason + metadata，或输出 typed validation detail 后再扩展本接口契约；前端不要从 `errors[]` 解析自然语言来推断字段路径。

**错误:**
| HTTP | code | 场景 |
|------|------|------|
| 404 | `ErrNotFound` | 知识库或语义模型不存在 |

---

**文档版本**: v1.0
**最后更新**: 2026-04-11
