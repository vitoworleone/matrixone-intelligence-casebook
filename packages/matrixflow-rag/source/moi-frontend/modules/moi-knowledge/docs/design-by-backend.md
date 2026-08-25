# Knowledge Base V2 改造设计

> 本文档面向前后端开发，包含完整 API 定义、TypeScript 类型、枚举值和错误码。

## 背景

moi-core 引入了 v2 语义模型（semantic model / semantic entries），替代原有的 nl2sql_knowledge 作为 explore SQL 的语义来源。本文档描述 moi-backend 的改造方案及前后端 API 变化。

## 核心概念

### 前端业务概念映射

| 前端概念         | 后端实体                                         | 说明                            |
| ---------------- | ------------------------------------------------ | ------------------------------- |
| 数据库（知识库） | `knowledge_base`                                 | 数据源容器，绑定 files + tables |
| 数据源配置       | `knowledge_base.tables` / `knowledge_base.files` | 不变                            |
| NL2SQL 语义配置  | `semantic_model` + `semantic_entries`            | 替代原 nl2sql_knowledge         |

### 关联机制

`knowledge_base` 和 `semantic_model` 通过 **`knowledge_base_id` 显式关联**（1:1）：

- moi-core 的 `semantic_models` 表新增 `knowledge_base_id` 列（NOT NULL，UNIQUE）
- 一个 knowledge_base 最多对应一个 semantic model
- moi-core API 支持按 `knowledge_base_id` 查询 semantic model
- moi-backend 在 knowledge_base CRUD 时同步维护对应的 semantic model
- explore 引擎在执行 SQL 时，仍然通过 `table_set_hash` 自动匹配 semantic model（运行时行为不变）

#### moi-core 侧改动（需 moi-core 配合）

`semantic_models` 表新增字段：

```sql
ALTER TABLE semantic_models ADD COLUMN knowledge_base_id BIGINT NOT NULL DEFAULT 0;
ALTER TABLE semantic_models ADD UNIQUE KEY uk_knowledge_base (knowledge_base_id);
```

moi-core API 变化：

- `POST /api/v1/workspaces/{id}/semantic-models` 请求体新增可选字段 `knowledge_base_id`
- `GET /api/v1/workspaces/{id}/semantic-models?knowledge_base_id=123` 支持按 kb_id 过滤
- `SemanticModel` 响应体新增 `knowledge_base_id` 字段
- 删除 knowledge_base 时，moi-core 不自动级联删除 semantic model（由 moi-backend 控制）

---

## API 变化

### 1. knowledge_base CRUD（保持不变）

接口路径和请求/响应结构**完全不变**，前端无需改动。

```
POST   /api/v1/knowledge_base          创建
GET    /api/v1/knowledge_base/list     列表
GET    /api/v1/knowledge_base/:id      详情
PUT    /api/v1/knowledge_base/:id      更新
DELETE /api/v1/knowledge_base/:id      删除
```

**内部变化（前端不感知）：**

- 创建时：如果 tables 非空，自动在 moi-core 创建对应的 semantic model
- 更新时：如果 tables 发生变化，自动同步更新 semantic model 的 tables
- 删除时：自动删除对应的 semantic model（级联删除所有 semantic entries）

### 2. 下掉的接口

以下接口**全部移除**，不做兼容：

```
POST   /api/v1/nl2sql_knowledge
PUT    /api/v1/nl2sql_knowledge/:id
DELETE /api/v1/nl2sql_knowledge/:id
GET    /api/v1/nl2sql_knowledge/:id
POST   /api/v1/nl2sql_knowledge/search
POST   /api/v1/nl2sql_knowledge/list
POST   /api/v1/analysis_template
PUT    /api/v1/analysis_template/:id
DELETE /api/v1/analysis_template/:id
GET    /api/v1/analysis_template/:id
POST   /api/v1/analysis_template/list
```

### 3. 新增接口

所有语义配置接口挂在 `/api/v1/knowledge_base/:id/` 下，前端通过 knowledge_base ID 操作，无需感知 semantic model ID。

#### 3.1 获取 Semantic Model 信息

```
GET /api/v1/knowledge_base/:id/semantic-model
```

返回该知识库对应的 semantic model 详情。

**Response 200:**

```json
{
  "id": 1,
  "knowledge_base_id": 42,
  "name": "sales_kb_semantic",
  "description": "",
  "tables": ["customers", "orders", "refunds"],
  "table_set_hash": "abc123...",
  "created_at": 1712345678,
  "updated_at": 1712345678
}
```

**错误情况：**

| HTTP | code              | 场景                                                         |
| ---- | ----------------- | ------------------------------------------------------------ |
| 401  | `ErrUnauthorized` | 未登录                                                       |
| 403  | `ErrForbidden`    | 无该 workspace 权限                                          |
| 404  | `ErrNotFound`     | knowledge_base 不存在                                        |
| 404  | `ErrNotFound`     | knowledge_base 存在但尚未创建 semantic model（未配置过语义） |
| 500  | `ErrServer`       | moi-core 调用失败                                            |

---

#### 3.2 Semantic Entries 列表

```
GET /api/v1/knowledge_base/:id/semantic-entries?kind=metric&page_size=20&page_token=
```

**Query Params:**

| 参数       | 类型           | 必填 | 说明                          |
| ---------- | -------------- | ---- | ----------------------------- |
| kind       | `SemanticKind` | 否   | 过滤 entry 类型，不传返回全部 |
| page_size  | number         | 否   | 分页大小，默认 20，最大 100   |
| page_token | string         | 否   | 分页游标，首页不传            |

**Response 200:**

```json
{
  "items": [
    {
      "id": 1,
      "kind": "metric",
      "key": "total_revenue",
      "tables": ["orders"],
      "spec": { "expr": "SUM(amount)", "synonyms": ["总收入"] },
      "created_at": 1712345678,
      "updated_at": 1712345678
    }
  ],
  "total": 1,
  "next_page_token": ""
}
```

**错误情况：**

| HTTP | code              | 场景                                          |
| ---- | ----------------- | --------------------------------------------- |
| 400  | `ErrParamInvalid` | `kind` 不是合法的 `SemanticKind` 枚举值       |
| 400  | `ErrParamInvalid` | `page_size` 超过 100 或为负数                 |
| 401  | `ErrUnauthorized` | 未登录                                        |
| 403  | `ErrForbidden`    | 无权限                                        |
| 404  | `ErrNotFound`     | knowledge_base 不存在                         |
| 404  | `ErrNotFound`     | semantic model 不存在（该知识库未配置过语义） |
| 500  | `ErrServer`       | moi-core 调用失败                             |

---

#### 3.3 创建 Semantic Entry

```
POST /api/v1/knowledge_base/:id/semantic-entries
```

如果该知识库还没有 semantic model，自动创建。

**Request Body:**

```json
{
  "kind": "metric",
  "key": "total_revenue",
  "tables": ["orders"],
  "spec": {
    "expr": "SUM(amount)",
    "synonyms": ["总收入", "revenue"],
    "unit": "CNY"
  }
}
```

**Response 201:**

```json
{
  "id": 1,
  "kind": "metric",
  "key": "total_revenue",
  "tables": ["orders"],
  "spec": {
    "expr": "SUM(amount)",
    "synonyms": ["总收入", "revenue"],
    "unit": "CNY"
  },
  "created_at": 1712345678,
  "updated_at": 1712345678
}
```

**错误情况：**

| HTTP | code              | 场景                                                   |
| ---- | ----------------- | ------------------------------------------------------ |
| 400  | `ErrParamInvalid` | `kind` 缺失或不合法                                    |
| 400  | `ErrParamInvalid` | `key` 缺失或为空                                       |
| 400  | `ErrParamInvalid` | `spec` 缺失或必填字段不完整（如 metric 缺 `expr`）     |
| 400  | `ErrParamInvalid` | `tables` 中包含不在 knowledge_base tables 中的表名     |
| 400  | `ErrParamInvalid` | knowledge_base 未配置 tables，无法创建 semantic entry  |
| 400  | `ErrParamInvalid` | `metric.requires_join` 引用了不存在的 relationship key |
| 400  | `ErrParamInvalid` | `verified_query.sql` 包含 DDL/DML 语句                 |
| 400  | `ErrParamInvalid` | `column_preference.preferred` 与 `deprecated` 相同     |
| 400  | `ErrParamInvalid` | `logic_text.injection_stages` 包含非法值或为空         |
| 401  | `ErrUnauthorized` | 未登录                                                 |
| 403  | `ErrForbidden`    | 无权限                                                 |
| 404  | `ErrNotFound`     | knowledge_base 不存在                                  |
| 409  | `ErrConflict`     | 同一 model 下 `key` 已存在                             |
| 500  | `ErrServer`       | moi-core 调用失败                                      |

---

#### 3.4 更新 Semantic Entry

```
PUT /api/v1/knowledge_base/:id/semantic-entries/:entry_id
```

**Request Body:** 同创建（`kind` 不可修改，传入值必须与原 entry 一致）

**Response 200:**

```json
{ "updated": true }
```

**错误情况：**

| HTTP | code              | 场景                           |
| ---- | ----------------- | ------------------------------ |
| 400  | `ErrParamInvalid` | 请求体校验失败（同创建）       |
| 400  | `ErrParamInvalid` | `kind` 与原 entry 不一致       |
| 401  | `ErrUnauthorized` | 未登录                         |
| 403  | `ErrForbidden`    | 无权限                         |
| 404  | `ErrNotFound`     | knowledge_base 或 entry 不存在 |
| 409  | `ErrConflict`     | 修改 `key` 后与其他 entry 冲突 |
| 500  | `ErrServer`       | moi-core 调用失败              |

---

#### 3.5 删除 Semantic Entry

```
DELETE /api/v1/knowledge_base/:id/semantic-entries/:entry_id
```

**Response 200:**

```json
{ "deleted": true }
```

**错误情况：**

| HTTP | code              | 场景                           |
| ---- | ----------------- | ------------------------------ |
| 401  | `ErrUnauthorized` | 未登录                         |
| 403  | `ErrForbidden`    | 无权限                         |
| 404  | `ErrNotFound`     | knowledge_base 或 entry 不存在 |
| 500  | `ErrServer`       | moi-core 调用失败              |

---

#### 3.6 批量导入

```
POST /api/v1/knowledge_base/:id/semantic-model/import
```

覆盖式导入，**替换**该 knowledge_base 对应 semantic model 的所有 entries（先删后写）。

**Request Body:**

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

**Response 200:**

```json
{
  "imported": 2,
  "model_id": 1
}
```

**错误情况：**

| HTTP | code              | 场景                              |
| ---- | ----------------- | --------------------------------- |
| 400  | `ErrParamInvalid` | `entries` 为空数组                |
| 400  | `ErrParamInvalid` | 任意 entry 校验失败（同创建规则） |
| 400  | `ErrParamInvalid` | entries 内存在重复的 `key`        |
| 400  | `ErrParamInvalid` | knowledge_base 未配置 tables      |
| 401  | `ErrUnauthorized` | 未登录                            |
| 403  | `ErrForbidden`    | 无权限                            |
| 404  | `ErrNotFound`     | knowledge_base 不存在             |
| 500  | `ErrServer`       | moi-core 调用失败                 |

---

#### 3.7 导出

```
GET /api/v1/knowledge_base/:id/semantic-model/export
```

**Response 200:**

```json
{
  "model": {
    "name": "sales_kb_semantic",
    "tables": ["customers", "orders", "refunds"]
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
```

**错误情况：**

| HTTP | code              | 场景                                          |
| ---- | ----------------- | --------------------------------------------- |
| 401  | `ErrUnauthorized` | 未登录                                        |
| 403  | `ErrForbidden`    | 无权限                                        |
| 404  | `ErrNotFound`     | knowledge_base 不存在或 semantic model 不存在 |
| 500  | `ErrServer`       | moi-core 调用失败                             |

---

#### 3.8 校验

```
POST /api/v1/knowledge_base/:id/semantic-model/validate
```

校验 semantic model 及所有 entries 的完整性（如 metric 引用的 relationship 是否存在）。

**Response 200（校验通过）:**

```json
{ "valid": true }
```

**Response 200（校验失败，注意 HTTP 仍是 200）:**

```json
{
  "valid": false,
  "errors": [
    "metric 'refund_rate' requires_join references missing relationship 'orders_to_refunds'",
    "entry 'use_paid_amount' column_preference.preferred and deprecated cannot be the same"
  ]
}
```

**错误情况（HTTP 非 200）：**

| HTTP | code              | 场景                                          |
| ---- | ----------------- | --------------------------------------------- |
| 401  | `ErrUnauthorized` | 未登录                                        |
| 403  | `ErrForbidden`    | 无权限                                        |
| 404  | `ErrNotFound`     | knowledge_base 不存在或 semantic model 不存在 |
| 500  | `ErrServer`       | moi-core 调用失败                             |

---

## TypeScript 类型定义

```typescript
// ─── 枚举 ────────────────────────────────────────────────────────────────────

/** Semantic Entry 类型 */
export type SemanticKind =
  | 'dimension' // 维度列（枚举、时间等）
  | 'fact' // 事实列（数值度量）
  | 'metric' // 业务指标（聚合公式）
  | 'relationship' // 表关联关系（JOIN）
  | 'column_preference' // 列别名/废弃列替换
  | 'named_filter' // 命名过滤条件
  | 'verified_query' // 已验证的标准问答
  | 'glossary' // 业务术语解释
  | 'logic_text' // 自然语言规则注入
  | 'sql_resultset'; // SQL 结果集

/** logic_text 注入阶段 */
export type InjectionStage =
  | 'planner_policy' // 规划阶段策略
  | 'sql_generation' // SQL 生成阶段
  | 'sql_followup' // 追问 SQL 阶段
  | 'sql_regenerate' // SQL 重生成阶段
  | 'sql_decomposition' // SQL 分解阶段
  | 'executor_rule' // 执行器规则
  | 'renderer_rule'; // 渲染器规则

/** 统一错误码 */
export type ApiErrorCode =
  | 'ErrParamInvalid' // 参数错误（400）
  | 'ErrUnauthorized' // 未认证（401）
  | 'ErrForbidden' // 无权限（403）
  | 'ErrNotFound' // 资源不存在（404）
  | 'ErrConflict' // 资源已存在（409）
  | 'ErrServer'; // 服务器内部错误（500）

// ─── 通用结构 ─────────────────────────────────────────────────────────────────

export interface ApiError {
  code: ApiErrorCode;
  msg: string;
  data: null;
}

export interface PagedResponse<T> {
  items: T[];
  total: number;
  next_page_token: string; // 空字符串表示没有下一页
}

// ─── Semantic Model ───────────────────────────────────────────────────────────

export interface SemanticModel {
  id: number;
  knowledge_base_id: number; // 关联的 knowledge_base ID
  name: string;
  description: string;
  tables: string[]; // 归一化后的表名列表（小写、排序）
  table_set_hash: string; // sha256(sorted tables)，只读
  created_at: number; // Unix timestamp（秒）
  updated_at: number;
}

// ─── Spec 结构（按 kind 区分）────────────────────────────────────────────────

export interface DimensionSpec {
  column: string; // 必填，列名
  data_type?: string; // 可选，如 VARCHAR / DATE / INT
  synonyms?: string[]; // 可选，同义词列表
  is_enum?: boolean; // 是否枚举列
  sample_values?: string[]; // 枚举列的示例值
  is_time?: boolean; // 是否时间列
  deprecated?: boolean; // 是否已废弃
  description?: string;
}

export interface FactSpec {
  column: string; // 必填，列名
  data_type?: string;
  description?: string;
  private?: boolean; // 是否对 LLM 隐藏
}

export interface MetricSpec {
  expr: string; // 必填，聚合表达式，如 SUM(amount)
  synonyms?: string[];
  unit?: string; // 单位，如 CNY / %
  description?: string;
  requires_tables?: string[]; // 计算该指标需要的表（必须是 model.tables 子集）
  requires_join?: string; // 引用的 relationship key（多表 metric 必填）
  semantic_pattern?: 'ratio' | 'semi_anti_join' | 'window'; // 语义模式提示
}

export interface JoinColumnPair {
  left: string; // 左表列名，格式 "table.column" 或 "column"
  right: string; // 右表列名
}

export interface RelationshipSpec {
  left_table: string; // 必填
  right_table: string; // 必填
  join_columns: JoinColumnPair[]; // 必填，至少一对
  description?: string;
  semantic_match?: boolean; // 是否允许语义模糊匹配
}

export interface ColumnPreferenceSpec {
  preferred: string; // 必填，推荐使用的列名
  deprecated: string; // 必填，废弃的列名（不能与 preferred 相同）
  reason?: string;
}

export interface NamedFilterSpec {
  expr: string; // 必填，SQL 表达式，如 status IN ('PAID', 'COMPLETE')
  synonyms?: string[];
  description?: string;
  applies_to?: string[]; // 适用的表名列表
}

export interface VerifiedQuerySpec {
  question: string; // 必填，自然语言问题
  sql: string; // 必填，对应的 SQL（只读 SELECT，禁止 DDL/DML）
  verified_by?: string; // 验证人 user_id
  tags?: string[];
}

export interface GlossarySpec {
  term: string; // 必填，术语名
  definition: string; // 必填，定义
  synonyms?: string[];
  related_metrics?: string[]; // 关联的 metric key 列表
  formula_hint?: string;
}

export interface LogicTextSpec {
  content: string; // 必填，自然语言规则描述
  injection_stages: InjectionStage[]; // 必填，至少一个
  priority?: number; // 注入优先级，数字越大越优先，默认 0
}

export interface SQLResultsetExpandSQLSpec {
  sql: string; // 可选整体 expand_sql 时必填
  params?: string[]; // 可选参数名列表
}

export interface SQLResultsetSpec {
  sql: string; // 必填，基础 SQL
  description: string; // 必填，结果集描述
  expand_sql?: SQLResultsetExpandSQLSpec; // 可选，扩展 SQL
  max_rows?: number; // 可选，默认 10000
  max_bytes?: number; // 可选，默认 2097152
  timeout_seconds?: number; // 可选，默认 10，最大 60
}

/** Spec 联合类型，根据 kind 对应 */
export type SemanticEntrySpec =
  | DimensionSpec
  | FactSpec
  | MetricSpec
  | RelationshipSpec
  | ColumnPreferenceSpec
  | NamedFilterSpec
  | VerifiedQuerySpec
  | GlossarySpec
  | LogicTextSpec
  | SQLResultsetSpec;

// ─── Semantic Entry ───────────────────────────────────────────────────────────

export interface SemanticEntry {
  id: number;
  kind: SemanticKind;
  key: string; // model 内唯一，建议英文小写下划线
  tables: string[]; // 必须是 model.tables 的子集，可为空数组
  spec: SemanticEntrySpec; // 按 kind 解析
  created_at: number;
  updated_at: number;
}

// ─── 请求体 ───────────────────────────────────────────────────────────────────

export interface CreateSemanticEntryRequest {
  kind: SemanticKind; // 必填
  key: string; // 必填，model 内唯一
  tables?: string[]; // 可选，必须是 knowledge_base tables 的子集
  spec: SemanticEntrySpec; // 必填，按 kind 填写对应结构
}

export type UpdateSemanticEntryRequest = CreateSemanticEntryRequest;

export interface ImportSemanticModelRequest {
  entries: CreateSemanticEntryRequest[]; // 必填，覆盖式导入
}

// ─── 响应体 ───────────────────────────────────────────────────────────────────

export interface SemanticEntryListResponse extends PagedResponse<SemanticEntry> {}

export interface MutationResponse {
  updated?: boolean;
  deleted?: boolean;
}

export interface ImportResponse {
  imported: number; // 成功导入的 entry 数量
  model_id: number;
}

export interface ExportResponse {
  model: Pick<SemanticModel, 'name' | 'tables' | 'knowledge_base_id'>;
  entries: Array<Omit<SemanticEntry, 'id' | 'created_at' | 'updated_at'>>;
}

export interface ValidateResponse {
  valid: boolean;
  errors?: string[]; // valid=false 时返回错误列表
}
```

---

## Semantic Entry Kind 说明

| Kind                | 用途                   | 必填 spec 字段                              |
| ------------------- | ---------------------- | ------------------------------------------- |
| `dimension`         | 维度列（枚举、时间等） | `column`                                    |
| `fact`              | 事实列（数值度量）     | `column`                                    |
| `metric`            | 业务指标（聚合公式）   | `expr`                                      |
| `relationship`      | 表关联关系（JOIN）     | `left_table`, `right_table`, `join_columns` |
| `column_preference` | 列别名/废弃列替换      | `preferred`, `deprecated`                   |
| `named_filter`      | 命名过滤条件           | `expr`                                      |
| `verified_query`    | 已验证的标准问答       | `question`, `sql`                           |
| `glossary`          | 业务术语解释           | `term`, `definition`                        |
| `logic_text`        | 自然语言规则注入       | `content`, `injection_stages`               |
| `sql_resultset`     | SQL 结果集             | `sql`, `description`                        |

`injection_stages` 枚举值：`planner_policy` / `sql_generation` / `sql_followup` / `sql_regenerate` / `sql_decomposition` / `executor_rule` / `renderer_rule`

---

## moi-backend 内部实现要点

### SemanticModelResolver

所有 semantic entries 操作前，先通过此逻辑获取 model ID：

```
1. 调 moi-core GET /semantic-models?knowledge_base_id=:kb_id 查找关联的 semantic model
2. 如果找到，返回 model ID
3. 如果未找到（首次配置语义）：
   a. 调 moi-core GET /knowledge-bases/:kb_id 获取 knowledge_base 详情
   b. 提取 tables.table_names，归一化（小写、去反引号、去 db 前缀、去重、排序）
   c. 如果 tables 为空，返回错误
   d. 调 moi-core POST /semantic-models 创建，传入 knowledge_base_id + tables
      （409 冲突时重新 GET 获取已有 model）
   e. 返回新建的 model ID
```

### knowledge_base 更新时的 tables 同步

```
1. 调 moi-core GET /semantic-models?knowledge_base_id=:kb_id
2. 如果找到 semantic model：
   PUT /semantic-models/:model_id  { tables: 新 tables }
   （entries 不动，tables 和 table_set_hash 自动更新）
3. 如果没找到：
   不操作（用户未配置过语义，等用户主动配置时再创建）
```

### knowledge_base 删除时的清理

```
1. 调 moi-core GET /semantic-models?knowledge_base_id=:kb_id
2. 如果找到：DELETE /semantic-models/:model_id（moi-core 级联删除所有 entries）
3. 如果没找到：不操作
```

---

## 前端改造要点

1. **知识库列表/详情页**：不变，knowledge_base CRUD 接口不变

2. **NL2SQL 语义配置 tab**（原 NL2SQL 知识 tab）：
   - 进入 tab 时调 `GET /knowledge_base/:id/semantic-model` 检查是否已有语义配置
   - 如果 knowledge_base 没有配置 tables，提示"请先配置数据表"
   - Entry 列表支持按 kind 过滤
   - 新建 entry 时根据 kind 渲染对应的动态表单
   - 支持整体导入/导出（YAML/JSON）

3. **kind 选择器**：下拉选择 10 种 kind，选中后动态渲染对应 spec 表单字段

4. **tables 字段**（entry 级别）：从 knowledge_base 的 tables 列表中多选，必须是子集

---

## 不变的部分

- explore 查询接口（`POST /explore/query`、`POST /explore/query/stream`）完全不变
- explore 请求体结构不变，`data_sources.knowledge_bases` 字段继续使用
- knowledge_base 的 files/tables 数据源配置不变
