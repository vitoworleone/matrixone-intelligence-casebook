# 可用工作项 (WorkItem) 说明

本文档说明**系统给用户提供了哪些可用的 WorkItem**，以及如何查询、在工作流中引用，便于用户构建自己的工作流。
工作流与 DSL 概念见 [CONCEPTS.md](../guide/CONCEPTS.md)、[DSL.md](../guide/DSL.md)。

---

## 1. 可用工作项从哪里来？

用户可见的「可用工作项」由两部分组成：

| 来源 | 说明 | 可见范围 |
|------|------|----------|
| **内建节点** | 引擎自带的节点（如 JQ、子网、回调节点等），由系统固定提供 | 所有用户 |
| **已注册工作项** | Worker 进程通过 `RegisterWorkItem` 注册、或通过服务端写入 `mowl_workitem_metadata` 的节点 | 按隔离级别：public 所有人；private 仅创建者；shared 创建者及被共享用户 |
| **自定义算子** | 用户在 Catalog 创建的 workspace 级算子，运行时以 WorkItem 形式出现在列表中 | 当前 workspace 内按创建者与隔离级别可见 |

用户通过 **列出可用工作项** 接口拿到上述合并后的列表，再在画工作流时选用合适的 **node_id**（外部工作项用 `dsl.WorkItem("节点名", node_id)`，内建节点用 DSL 提供的 `dsl.JQ` / `dsl.Subnet` / `dsl.Callback` 等）。

---

## 2. 如何查询「可用工作项」？

### 2.1 HTTP API

```http
GET /api/v1/workspaces/:workspace_id/workitems
```

- 需要认证（API Key 等），服务端根据当前用户过滤可见工作项。
- 响应体为 JSON，结构见下。

### 2.2 Go SDK

```go
workitems, err := client.WorkItems(workspaceID).List(ctx)
if err != nil { ... }
// workitems: map[string]*mowl.WorkItemMetadataList
// key = node_id，value = 该 node_id 下多个版本元数据（通常一个）
for nodeID, list := range workitems {
    for _, meta := range list.GetItems() {
        fmt.Printf("node_id=%s description=%s version=%s\n",
            nodeID, meta.GetDescription(), meta.GetVersion())
    }
}
```

### 2.3 响应结构概要

- **HTTP 200** 或 **SDK** 返回：`workitems` 为 `map<string, WorkItemMetadataList>`。
- **key**：`node_id`（工作项唯一标识，在 DSL 里用作 `dsl.WorkItem("步骤名", node_id)` 的第二个参数）。
- **value**：`WorkItemMetadataList`，内含 `repeated WorkItemMetadata items`（同一条工作项可有多个版本）。
- **WorkItemMetadata** 常用字段：`description`、`version`、`input_schema`、`output_schema`、`isolation_level`、`stream`（是否流式）等。

---

## 3. 内建节点（系统固定提供）

以下节点**对所有用户可见**，由引擎实现，无需 Worker 执行。
在 **DSL 中不通过 `WorkItem(name, node_id)` 使用**，而用对应 DSL 函数；表中仅列出 node_id 与用途，便于与「可用工作项」列表对照。

| node_id | 说明 | 在 DSL 中的用法 |
|---------|------|------------------|
| `mowl:start` | 起始节点，工作流入口，自动执行 | 无需写，由 `Workflow(...).Chain(...)` 自动生成 |
| `mowl:end` | 结束节点，工作流出口 | 无需写，自动生成 |
| `mowl:jq` | JQ 查询节点，对数据进行转换/条件判断 | `dsl.JQ("节点名", "jq 表达式")` |
| `mowl:subnet` | 子网节点，嵌套执行子工作流 | `dsl.Subnet("节点名", "子网名")`，子网需先 `DefineSubnet` |
| `mowl:worker_callback` | Worker 回调节点，向指定 Worker 发回调 | `dsl.Callback("节点名", dsl.CallbackConfig{...})` |
| `mowl:noop` | 空操作节点，直接透传数据（测试/占位） | 一般不直接写，引擎内部使用 |
| `mowl:cancellation_set` | 取消集合节点（暂未实现） | 暂不用于画图 |
| `catalog:http.request` | 内建 HTTP 请求工作项，由 Catalog 进程内 Worker 执行 | `dsl.WorkItem("步骤名", "catalog:http.request")` |

**总结**：画图时 **JQ / 子网 / 回调** 用 DSL 的 `JQ`、`Subnet`、`Callback`；**需要 Worker 执行的步骤** 才用 `dsl.WorkItem("步骤名", node_id)`，其中 `node_id` 来自「可用工作项」列表中**非内建**的项（即其他 node_id，多为业务方或平台预置的扩展工作项）。

---

## 3.1 内建工作项：`catalog:http.request`

`catalog:http.request` 是由 Catalog 进程内嵌 Worker 提供的内建 HTTP 请求工作项，无需外部 Worker 进程即可使用。

### 输入 (input)

引擎通过 `__input__` 变量将工作流数据注入节点，jq 插值上下文为 `{"data": <工作流数据>, "vars": <节点变量>}`，可用 `.data.xxx` 和 `.vars.xxx` 引用。

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `method` | string | 是 | HTTP 方法：`GET`、`POST`、`PUT`、`PATCH`、`DELETE`、`HEAD` |
| `url` | string | 是 | 请求 URL，必须以 `http://` 或 `https://` 开头 |
| `headers` | object | 否 | 请求头，key/value 均为字符串 |
| `body` | string | 否 | 请求体（字符串） |
| `timeout_seconds` | integer | 否 | 单次请求超时（秒），范围 1–300，默认 30 |

### 输出 (output)

| 字段 | 类型 | 说明 |
|------|------|------|
| `status_code` | integer | HTTP 响应状态码 |
| `headers` | object | 响应头，key/value 均为字符串 |
| `body` | string | 响应体（字符串） |

### 行为说明

- HTTP 4xx / 5xx 响应**不视为工作项失败**，`status_code` 会正常写入输出，由下游节点（如 JQ）判断处理。
- 请求超时（超过 `timeout_seconds`）或网络错误才会导致工作项失败（`FAILED` 状态）。
- 连接池与读写超时为全局配置，见 `[http_workitem]` 配置节（`catalog/pkg/config/config.go`）。

### DSL 示例（Go SDK）

```go
dsl.WorkItem("call-api", "catalog:http.request").
    Input(`{
        "method": "POST",
        "url": "https://api.example.com/data",
        "headers": {"Content-Type": "application/json"},
        "body": "{\"key\": \".data.value\"}"
    }`)
```

### YAML 示例

```yaml
work_item:
  name: call-api
  id: catalog:http.request
  input: |
    {
      "method": "POST",
      "url": "https://api.example.com/data",
      "headers": {"Content-Type": "application/json"},
      "body": "{\"key\": \".data.value\"}"
    }
```

### 带 output 变换的示例

```go
dsl.WorkItem("call-api", "catalog:http.request").
    Input(`{"method": "GET", "url": "https://api.example.com/items"}`).
    Output(`.body | fromjson | .items`)
```

上游节点的输出经 `.body | fromjson | .items` jq 表达式变换后写入工作流数据，供下游节点使用。

---

## 4. 已注册工作项（外部 WorkItem）

- **来源**：Worker 调用 `RegisterWorkItem(nodeID, metadata, handler)` 时，引擎会将元数据写入 `mowl_workitem_metadata`（若持久化开启）；或通过服务端接口注册工作项元数据。
- **可见性**：由 `isolation_level` 决定——`public` 所有人可见，`private` 仅创建者，`shared` 创建者及被共享用户。
- **多副本注册语义（同一用户）**：
  - 允许多个 Worker 副本注册同一个 `node_id`（用于负载均衡）。
  - 这些副本的元数据必须一致（`isolation_level`、`version`、`input_schema`、`output_schema`）。
  - 若同一 `node_id` 出现元数据不一致，后续副本在 `Connect()` 阶段会被拒绝，返回 `ALREADY_EXISTS`。
- **在 DSL 中的用法**：从「可用工作项」列表中取 **node_id**（即注册时的 nodeID），在流程中写为：
  ```go
  dsl.WorkItem("步骤名", node_id)
  ```
  例如列表中有一条 `"my-company:etl"`，则：
  ```go
  dsl.WorkItem("etl-step", "my-company:etl")
  ```
- **多版本**：同一 node_id 可能对应多条元数据（不同 version），列表接口返回 `WorkItemMetadataList.items`；画图时只关心 node_id，执行时引擎会按版本策略选用。
- **节点级计算资源**：所有 `work_item` 节点都可以在 YAML 的 `work_item.compute_resource_id` 中声明节点级计算资源绑定。该绑定只影响需要通过 Worker 执行的 WorkItem；`jq`、`subnet`、`callback`、控制流和 stream 节点不支持该字段。调度时节点级绑定优先于 Task/工作流默认计算资源，且不会在专属 Worker 缺失时回退到共享 Worker。

---

### 4.1 自定义算子

自定义算子通过 `POST /api/v1/workspaces/:workspace_id/custom-operators` 创建，创建后会出现在 WorkItem 列表和 Catalog 算子列表中。当前支持两种类型：

| kind | 说明 | 关键字段 |
|------|------|----------|
| `code` | Python 代码算子。源码来自请求里的 `code` 或已存在的 `source_file_id`，运行时由 `moi:custom.operator` 分发给 python-worker。 | `language=python`、`handler`、`source_file_id` |
| `builtin_binding` | 基于系统已有 WorkItem 固定部分参数后形成的自定义算子。运行时不执行自定义代码，而是映射输入后调用 `base_node_id@base_node_version`。 | `base_node_id`、`base_node_version`、`binding_config` |

`builtin_binding` 的 `binding_config` 是面向执行和 NL2DSL 的稳定元数据，格式如下：

```json
{
  "fixed_input": {
    "sql": "select * from orders where dt = :dt"
  },
  "input_mapping": [
    {"source": "params", "target": "params"}
  ]
}
```

- `fixed_input`：系统内置 WorkItem 的固定输入，适合保存 SQL、固定 catalog/database、固定开关等。
- `input_mapping`：把用户暴露给自定义算子的输入字段映射到基础 WorkItem 输入字段，`source` 和 `target` 支持点路径。
- 自定义算子支持同一 `node_id` 下的多版本；创建新版本时复用相同 `identifier` 或 `node_id` 并传入新的 `version`。已创建记录的 `version` 不可通过更新接口变更，若要升级执行契约必须创建新版本。
- 自定义算子在 DSL 中必须显式写入 `work_item.spec.version`，例如 `dsl.WorkItem("step", node_id).WithVersion("v2")`；Catalog 列表的 `preferred_version` 只用于新建节点默认值，已保存工作流按 DSL 中固定的版本执行。
- `base_node_version` 必填，避免同一个系统节点多版本共存时选错执行契约。
- 代码类自定义算子测试运行会启用 worker 级 trace。python-worker sandbox 捕获的 stdout/stderr 作为 `developer_logs` 写入 worker-call trace span，用于 Catalog 测试控制台展示；业务输出仍只来自 handler 返回的 JSON 对象，不会混入日志字段。

SQL 类内部自定义算子仍可绑定到 `moi:data.runsql@v1`。面向普通用户和 agent 的 SQL 处理节点是 `moi:data.sql.process@v1`，输入只包含 `sql`。输出使用稳定 envelope：`sql`、`rows`、`affected_rows`、`elapsed_ms`、`truncated`。`SELECT`/`SHOW`/`WITH` 查询消费 `rows`；DML/DDL 消费 `affected_rows` 作为执行元数据。如果后续逻辑需要 DML/DDL 后的业务值，应追加一个 `SELECT` SQL 节点读取 `rows`。

工作流任意 SQL 入口只接受一条业务数据语句，允许范围为查询、`INSERT`/`UPDATE`/`DELETE`/`REPLACE`，以及 Table/View/Index 的创建、修改、重命名、清空和删除。`SHOW` 只开放 Database/Table/Column/Index 等普通结构元数据；`DESCRIBE` 和 `EXPLAIN` 继续作为查询输出。用户、Role、授权、Account、Database 管理，`SET`/`USE`/事务/锁/预编译等会话控制，`KILL`/`SHOW PROCESSLIST`，备份恢复、快照、PITR、发布订阅以及其它未登记的控制语句必须在连接 MatrixOne 前拒绝。查询、DML 或 DDL 内嵌的控制函数、文件读写、显式行锁和 session 变量赋值同样不属于工作流数据面；`/*+ ... */` rewrite/optimizer hint 会在 MatrixOne 解析前改变实际执行语义，也必须拒绝。多步骤 ETL 应拆成多个用户 SQL 节点；内部系统流程可以使用 `moi:data.sql_pipeline` 的分阶段合同，但每个元素仍必须是单条允许语句，不能开启数据库驱动的 multi-statements。

当 `moi:data.runsql@v1` 的 SQL 已由 `binding_config.fixed_input.sql` 完整提供且 `input_mapping=[]` 时，自定义算子不暴露运行时输入，可使用 `{"type":"object","properties":{}}` 作为 `input_schema`。其他自定义算子输入 schema 仍至少需要一个带类型和说明的 `properties` 字段；只要存在运行时映射，就不能使用空输入 schema。

### 4.2 go-worker 已注册工作项速查

以下工作项由 `go-worker` 注册；用户可见节点会进入普通用户的能力目录和 NL2DSL 召回面，表中另保留少量内部兼容节点用于说明存量 DSL 与既有系统模板。完整 JSON Schema 与 visibility 见 `docs/workers/go-worker-workitems-catalog.json`。

| node_id | 说明 |
|---------|------|
| `moi:document.parse` | 内部 legacy 文档解析节点，固定走 V2，供仍依赖 V2 的系统模板和存量 DSL 按原 NodeId 继续执行 |
| `moi:parse` | 用户默认文档解析节点（「文档解析」），固定走 parse-v3 进程内引擎，覆盖文档/邮件/音频/视频全部文件类型，自带 V3 选项 schema。它是新建通用工作流的默认节点；旧 NodeId `parse:stage_runtime` 保留为内部 legacy alias |
| `moi:files.read_documents` | 已禁用；文件正文不得通过 workflow data 传递 |
| `moi:files.read_text` | 已禁用；文件正文不得通过 workflow data 传递 |
| `moi:files.write_documents` | 将 `documents` 写入 JSONL 文件 |
| `moi:catalog.source.read` | 从 Catalog 文件、数据资产或 volume 读取工作流输入源 |
| `moi:catalog.sink.write` | 将工作流输出写回 Catalog 文件或数据资产；用户级配置必须提供目标位置。接收带 `file_name` provenance 的 `documents` / `file_id(s)` / `trigger_context` / `source_ref` 时会复用来源文件名；无来源文件名的 `rows` / `json` / `text` 输出才需要用户提供输出文件名，`rows`/`columns` 可按 CSV 写出 |
| `moi:knowledge.index.build` | 从 `documents` 构建知识库索引 |
| `moi:data.lineage.register` | 注册源文件、解析产物、向量索引或输出文件的数据血缘关系 |
| `moi:data.sql.process` | 在 workspace 数据库执行 SQL。输入只包含 `sql`；查询输出 `rows`，DML/DDL 输出 `affected_rows` |
| `moi:data.sql_pipeline` | 批量执行 SQL 语句，支持事务包裹和 replace_by_clone 刷新 |
| `moi:llm.extract.structured` | 使用 LLM API 从文本中提取结构化 JSON |
| `moi:llm.extract.structured.advanced` | 高级结构化提取，支持 `n_to_1`/`n_to_n` 模式 |
| `moi:parser.clean.text` | 规范化文本空白字符 |
| `moi:parser.convert.audio.rich` | 音频转文本（ASR 后端） |
| `moi:parser.convert.document.docx.rich` | DOCX 富文本转换 |
| `moi:parser.convert.document.image_ocr.rich` | 图片 OCR 文档转换 |
| `moi:parser.convert.document.pdf.rich` | PDF 富文本转换 |
| `moi:parser.convert.document.pptx.rich` | PPTX 富文本转换 |
| `moi:parser.convert.document.rich` | 通用富文档转换 |
| `moi:parser.convert.html` | HTML 转纯文本 |
| `moi:parser.convert.image.rich` | 图片转文本（视觉/OCR 后端） |
| `moi:parser.convert.plain` | 纯文本源转文档 |
| `moi:parser.convert.video.rich` | 视频转文本 |
| `moi:parser.json.repair` | 修复格式错误的 JSON |
| `moi:parser.router.mime` | 按 MIME 类型路由到不同转换分支 |
| `moi:parser.split.length` | 按长度分割文本（支持重叠） |
| `moi:parser.split.level` | 按 Markdown 标题层级分割文档 |
| `moi:volumes.ensure` | 确保 volume 存在 |
| `moi:volumes.files.add` | 向 volume 添加文件引用 |
| `moi:volumes.files.list` | 列出 volume 关联文件 |
| `moi:volumes.files.move` | 在 volume 间移动文件 |
| `moi:volumes.files.remove` | 从 volume 移除文件引用 |
| `moi:volumes.resolve` | 按名称解析 volume ID |

`moi:catalog.source.read` 读取 volume 输入时会通过 Catalog VolumeFiles 分页接口遍历全部文件；显式传入 `limit` 时按 `limit` 截断返回数量。该节点只输出 `sources` / `file_ids` 等引用信息，`read_mode=documents` 会被拒绝，避免文件正文进入 workflow data。

结构化信息提取节点 `moi:llm.extract.structured` 和
`moi:llm.extract.structured.advanced` 会按 go-worker 配置
`extract.max_tokens_per_batch` 控制单批输入 token 预算，默认值为 `12000`。该参数只影响
输入文档/块的分批大小，不是 LLM 输出 `max_tokens`。

向量写入相关节点（包括组合节点 `moi:knowledge.index.build` 最终调用的
`moi:data.retrieval.vector.write`）会为**新建**目标表创建
`embedding VECF32(...)` 字段，并确保其上存在 `ivfflat` cosine 向量索引，
索引参数为 `LISTS = 256`、`OP_TYPE 'vector_cosine_ops'`。
建表使用 `CREATE TABLE IF NOT EXISTS`：已存在的 `VECF64` 等旧表结构不会被改写，
新旧类型可按知识库各自的 `vector_table` 并存；已有老表会在下一次写入时自动补齐该索引。
`moi:knowledge.index.build` 的 `volume_id` 输入使用 Catalog volume picker 的
数值型 `source_ref.volume_id`，模板应直接传递该数字值。
向量写入的 `policy=OVERWRITE` 语义为按本批次文档 `id` 覆盖：同一批次内
如果出现重复 `id` 会直接失败；写入 MatrixOne 时会在同一事务内先删除本批次
已有 `id`，再执行普通插入，保证重跑同一文件生成相同稳定 `id` 时不会因主键冲突失败。
`moi:data.retrieval.vector.write` 对带 `file_id` / `raw_file_id` 且包含
`chunk_id`、`block_uuid`、`document_index`、`parent_index`、`chunk_index`、
`chunk_start`、`chunk_end`、`page_num` 等结构化位置 metadata 的文档，会按文件与位置
metadata 生成向量行 `id`；这避免 PDF 解析块或切块复用同一上游 document `id`
时，在向量表内产生重复主键。
启用 `moi:retrieval.index.multilevel` 时，chunk 级条目的 `chunk_index` / `chunk_id`
会按同一文件内的全局 chunk 顺序生成，同时保留原始 `parent_index` 等解析定位 metadata，
并写入 `chunk_index_scope=file` 标记。Explore 表格扩展可据此按文件级
`chunk_index` 区间补全同一 section，避免多个解析块各自从 `chunk_index=0`
开始时生成重复向量行 `id`。

### 4.2 `moi:document.parse` — 内部旧版文档解析（Legacy）

`moi:document.parse` 固定使用 parser v2，保留给仍依赖 V2 的系统模板、已创建工作流的 DSL
快照和显式引用该 NodeId 的兼容场景；它不再进入普通用户的能力目录，也不用于新建的通用
模板。节点行为没有改变：输入可
直接提供 `sources`，也可提供 `file_id` / `file_ids`，节点会归一化为 `sources` 后复用
rich document converter 输出 `documents`。

当 `sources` / `file_ids` 包含多个文件时，`documents` 会保留每个 block 的
`file_id` / `raw_file_id` 来源信息。下游 `moi:catalog.sink.write` 接收
`documents: '{{ .data.documents }}'`，或历史写法 `format: json` +
`json: '{{ .data.documents }}'` 时，会按来源文件分组写回 Catalog；带 Markdown /
layout artifact 的解析结果会为每个来源文件分别生成一个 ZIP。
当 `moi:catalog.sink.write` 同时接收 `documents` 和 `extraction_result` 时，
结构化抽取结果会作为 `{stem}_extract.json` 写入同一个解析 ZIP，Catalog 不再额外保存
一个只包含抽取结果的外层 JSON 文件。

当前支持 PDF、Office/Excel、HTML 以及邮件文件 `eml` / `msg`。HTML 解析会按 DOM 顺序
输出 `text` 与 `table` documents；表格保留为 HTML table block，并在调用
`moi:catalog.sink.write` 时写入 ZIP 的 `tables/`。
邮件解析会提取邮件头、
正文、HTML 表格和图片块；内嵌图片会上传到 FileService 并以 `image_url` 写入
document metadata。外部 HTTP(S) 图片会在可访问时下载并上传到 FileService；不可访问时
保留原链接并记录 warn 日志。调用 `moi:catalog.sink.write` 时会和 PDF 解析结果一样输出 ZIP，
包含 `{stem}_parse.json`、`{stem}.md`，以及存在时的 `images/`、`tables/`。邮件图片 OCR
复用 PDF 图片解析选项：仅当 `options.vlm_ocr_model` 设置时调用该模型处理已上传图片。
ZIP 内 `{stem}_parse.json` 的 image block 会保留 `content`，并始终包含 `ocr` 与 `caption`
字段；图片解析产生的 OCR 文本会写入 `ocr`，caption 结果会写入 `caption`。OCR 仅提取图片中
实际可见的文字、公式、符号、表格文字和标签；caption 使用中文概括图片表达的主要内容。
直接解析图片文件时也会生成 Markdown artifact，因此 `moi:catalog.sink.write` 会输出 ZIP；
Markdown 固定包含独立的 `caption:`、`ocr:` 段落和图片引用，图片文件写入 ZIP 的 `images/`。

旧 rich pipeline 调用 Catalog Parser API 时，worker 侧禁用 SDK 默认重试，改为对
timeout / 504 / EOF 做有界退避重试，默认最多 2 次尝试；同时按 go-worker 进程内的
parser backend type 建独立异步处理队列，如 `PARSER_MINERU`、`PARSER_OPENXML`、
`PARSER_PADDLEOCR` 分别排队，默认每类最多 2 个并发请求，最多排队 1024 个请求。
队列满时直接返回 `RESOURCE_EXHAUSTED`（系统繁忙）并使节点失败，不继续等待。
可通过 go-worker 配置 `parser.backend_queues.default_concurrency`、
`parser.backend_queues.per_type_concurrency`、`parser.backend_queues.queue_size`
调整。`parser.backend_queues.workitem_timeout` 从文件解析 workitem 开始计时，覆盖排队、
重试和后端请求，默认 1 小时。parser API 失败、返回空文本、解析服务不可用，或输入不是
已支持的文本/文档类型时，节点直接返回 `FAILED`，错误信息保留文件名、file_id、MIME、
文档类型、数据大小和上游 parser 的 EOF、HTTP 5xx 或 timeout 细节，并附带当前 workspace
已配置 parser backend 的 `id`、`name`、`type`、`timeout_seconds` 和
`supported_mime_types`，用于判断当前文件类型是否有可路由的解析后端。节点不会把未解析的
二进制内容、空结果或替代路径结果当作成功 `documents` 输出。

当 `options.enable_parser_pipeline=true` 或 `options.workflow_parser=true` 使 PDF 进入 v2/MinerU
解析链路时，worker 会先通过 Catalog 的 parser route 为 `application/pdf` 解析到
`PARSER_MINERU` backend 和 `ONLINE` endpoint，再直连该 MinerU endpoint 获取完整
layout/Markdown ZIP。未配置 MinerU backend、endpoint 未上线、PDF route 解析到非
`PARSER_MINERU` backend，或 endpoint 地址为空时，节点直接失败，并在错误中提示需要为当前
workspace 配置 `PARSER_MINERU` backend 和 `ONLINE` endpoint。
直连 MinerU 上传前会先统计 PDF 页数；页数超过 Catalog parser route 下发的
`pdf_split_threshold_pages` 时，按 `pdf_split_pages_per_chunk`
页一段串行裁剪并上传。该分页逻辑由 parser client 内部处理，WorkItem 不感知分页配置。
分页参数由 Catalog 的 `[parser.mineru.pdf_split]` 统一管理，默认阈值为 180 页、每片 50 页。
分片响应合并时会恢复全局 `page_idx`，并将同名图片引用改写为
`images/chunkNNN_<filename>`，确保 Markdown 和 `middle_json` 中的图片引用都能在
layout image map 中找到对应内容。
分片、ZIP 响应解包、图片上传和真实 MinerU 测试留档的完整契约见
[MinerU PDF 分片与响应合并](../../workers/go-worker/pkg/workitems/parser/clients/MINERU_PDF_SPLIT.md)。
PDF v2/MinerU 与 parse-v3 都支持 `options.text_layer_backfill`：默认 `auto`，可显式设为
`off` 关闭。只接受 `auto`/`off` 两个值，传入其它值时 v2 与 v3 都直接返回错误（不会静默
回退到默认开启），避免配置笔误被吞掉后仍触发内容回填。该能力只在 MinerU 产出的普通文本类 block（`TEXT` / `TITLE` / `LIST` /
`CODE`）出现可疑乱码时触发，使用同页 `bbox` 从 PDF text layer 做 bounded extraction，
通过候选长度、坏字符分数、anchor overlap 和源页健康度仲裁后才回填；`TABLE`、`IMAGE`、
`EQUATION`、`HEADER`、`FOOTER`、`DISCARDED` 不会被改写。候选文本一旦通过仲裁，会按
PDF text layer 抽取结果原样写回，不额外折叠空白或重排换行。PDF text layer 不可用、源页
文本层本身异常、bbox 缺失或候选质量不足时，解析不会失败，只在 block metadata 中标记
degraded/skipped。
解析输出可包含 `text_repair` 摘要：`scanned` / `candidates` / `repaired` / `skipped` /
`errors` / `skipped_by_reason` 统计回填过程，并用 `original_md_file_id` 保留原 MinerU
Markdown artifact，`repaired_md_file_id` 指向上传成功后的 repaired Markdown。被处理的
block 会在 metadata 中写入 `text_repair_status`、`text_repair_source`、
`text_repair_reason`、`text_repair_badness_before`、`text_repair_badness_after`，
修复成功时还会写入 `text_repair_original_text` 与 `text_repair_repaired_text`。如果有
block 被回填且 repaired Markdown 上传成功，最终 `md_file_id` 指向 repaired Markdown。
Markdown patch 按乱码原文在原 Markdown 中定位；若某个 block 的原文定位不到（Markdown
渲染与 block 内容有出入、乱码文本在剩余内容中重复导致歧义、或 fragment merge 的 span
不连续），该 block 的回填会**就地回退**：`documents`/plain_text 恢复 MinerU 原文，
metadata 标记 `text_repair_status=reverted` 与 `text_repair_reason=markdown_unpatchable`，
摘要计入 `skipped_by_reason.markdown_unpatchable` 并置 `degraded=true`，解析继续成功；
若所有回填都被回退，`md_file_id` 保持指向原 Markdown。三类输出（documents/plain_text/
Markdown artifact）因此始终一致。只有原 Markdown 下载失败、repaired Markdown 上传失败、
或 anchors 元数据损坏（程序性错误）时节点才直接返回错误。启用回填但 PDF text layer
源不可用（上游未传 file id 或下载失败）时，摘要以 `skipped_by_reason.text_layer_unavailable`
标记并置 `degraded=true`，与显式 `off`（`skipped_by_reason.disabled`）区分。
当 PDF v2 图片块启用 `options.vlm_ocr_model` 后，最终 Markdown artifact 会在每个图片引用
下方写入 `### OCR` 与 `### Caption` 两个三级标题，分别包含该图片块的 OCR 文本和 caption
结果；Markdown 更新失败会使节点直接失败，不返回未更新 artifact 的成功结果。

#### `moi:parse` — 用户默认文档解析节点（V3）

`moi:parse`（显示名「文档解析」，旧 NodeId `parse:stage_runtime` 保留为内部 legacy
alias、复用同一 handler）是新建工作流的用户默认文档解析节点；固定
`parser_version=v3`，数据直接进入 parse-v3 进程内引擎，不依赖 RuntimeConfig 默认在
运行时落地。`options.parse_tier` 作为用户可见档位提供 `native` / `standard` /
`enhanced`，缺省为 `standard`；`options.vlm_ocr_model` 与 `options.page_selector` 也属于
V3 接受的公开选项。它按文件扩展名/MIME 由 `SourceRouter` 路由到对应解析路径——文档
（PDF 及 DOCX/PPTX 默认 `pdf` route 走 MinerU，显式 `openxml` route 走 OpenXML）、图片、
邮件（eml/msg，委托给 parser 包的 `parseEmail`）、
音频/视频（mp3/wav/mp4/mov 等，原生 `mediaBlockSource` + `asr` stage，内部调用
moi-audio-service 异步 job API，见下）——公开 options 只接受 V3 扁平 key；除上述档位、
模型和页码外，增强项例如 `complex_table` / `cross_page_table_merge` / `save_as_image` / `table_mode` /
`reading_order`(`index`/`xy_cut`) / `title_enrichment`(`off`/`vlm`) /
`image_caption` / `image_ocr` / `formula` / `docx_route`(`pdf`/`openxml`) /
`docx_openxml_strict` / `pptx_route`(`pdf`/`openxml`) / `page_selector`），
刻意不带 v2 选项 schema，避免 v2/v3 key 不匹配被 v3 严格 decoder
拒绝。`text_layer_backfill` 是默认自动开启的确定性修复策略，不在 V3 节点表单中暴露；
API 调试场景仍可显式传 `options.text_layer_backfill=off` 关闭。输入与输出契约同
`moi:document.parse`（`sources` / `file_ids` → `documents`），不在 `documents[]`
顶层新增 `table_url`。

单 source 响应会在顶层 `metadata` 发布 Standard policy 证明或可操作的 V3 参数警告；
`metadata.warnings[]` 包含被忽略参数的 `key` 与 `message`。例如 Native 显式传入
`header_footer_as_text=true` 时，有效值保持关闭，并返回该参数在 Native 档位被忽略的
warning。没有 policy 或 warning 的 Native、Enhanced 和非目标 Standard 响应继续保持
原有顶层形状；多 source 响应不合并可能属于不同文件执行的 metadata。

音频/视频是**原生源**（`mediaBlockSource`），不是 email 那样的 `ExternalSource`
委托：source 只产出一个引用文件的占位 block，真正的转写发生在 `asr` stage
（`v3/asr_stage.go`，`graphByFamily[FamilyAudio/FamilyVideo] = {"asr"}`），对齐
`imageBlockSource`「source 只引用文件、Stage 做实际工作」的范式——这也是为什么
video 将来要接视觉能力时可以单独加一个 Vision stage 挂在同一批 block 上，不用碰
ASR 部分。moi-audio-service 客户端（`clients.AudioServiceClient`）定义在
`parser/clients` 包（v3 能 import 的叶子包），通过 `Backends.Audio` 注入，和
`VLM`/`Paddle`/`File` 走同一套机制。`ParseV3Input` 只携带 `FileID`/`FileType`，
没有真实文件名/MIME/大小，因此 `asr` stage 从扩展名合成占位文件名，并从共享的
`mediaFormatByExtension` 取得 canonical MIME。该 MIME 同时用于当前 workspace 的 Catalog
route lookup 与 audio-service payload；空或不支持的扩展名不会按 kind 退化成
`audio/mpeg`/`video/mp4`，而是在路由或后端请求前 fail closed。

`clients.AudioServiceClient` 只接受 effective route 中 deployment-owned system defaultAI 的
`PARSER_MOI_AUDIO_SERVICE` backend/endpoint，并要求有效 HTTP(S) endpoint 和正数
`timeout_seconds`。endpoint、API key、backend/endpoint identity 与 sync/async 总操作预算
均来自 route；旧的 `MOI_VIDEO_BACKEND_URL`、`MOI_MEDIA_BACKEND_URL`、
`MOI_AUDIO_BACKEND_URL` 不参与 endpoint 选择。moi-audio-service 返回的
每个转写分段经 `asr` stage 的 `RemoveBlockOp`+`AddBlockOp`（同 `table_split`
拆分过合并 TABLE 的手法）写回同一个 Page 的多个 block（不是「一个时间段一个
Page」——后者仍是设计注释里悬而未决的未来优化方向）。go-worker 内部的
`moi:parser.convert.audio.rich`/`video.rich` 节点与 `moi:parse` 共用同一个
`clients.AudioServiceClient` 实现（`parser/clients/audio_service_client.go`），
不再有两套彼此不一致的 audio-service 客户端。

当音频服务在创建任务或轮询任务状态时返回 HTTP `429`，客户端会优先按
`Retry-After` 等待；未提供有效值时依次等待 2、4、8 秒，最多额外重试三次
（合计最多 4 次请求）。`Retry-After` 秒数或 HTTP-date 等待上限为 30 秒，避免
后端异常值长时间阻塞 worker。重试耗尽或收到其他非 2xx 响应时，工作流错误只包含
HTTP 状态及后端 JSON 协议中的 `code`、`message` 字段，不会回传任意响应体中的
下载 URL、令牌或其他敏感字段。

表格输出契约如下：`table_mode=html`（默认）保留 `table` document 与 HTML 内容；
`table_mode=image` 将表格作为 `image` document 输出，并通过 `metadata.level=table`
标识来源是表格。当 `save_as_image=true` 时，v3 会把 parser backend 返回的表格裁剪图
标识原样透出到 `documents[].metadata.table_image_url`；空字符串或非字符串不输出，
非空字符串不做 trim、URL 展开或 file_id 规范化。该值是上游返回的图片定位符：
可能是 FileService file_id，也可能是 URL 形态，调用方不应把它当作 ZIP 内路径。

当下游 `moi:catalog.sink.write` 生成解析 ZIP 时，`{stem}_parse.json.blocks[]` 会把
`metadata.table_image_url` 投影为顶层 `table_url`，并在 `meta.table_image_url` 保留
同一个原始值。下载 ZIP 中的 `images/<id>.*` 来自通用图片 artifact 收集
（`image_url` / `s3_image_url` / `images_file_ids` / `image_artifacts`）以及
`download_includes=images`，不会把 `table_image_url` 单独当作 ZIP artifact source
再解析一遍。

当 `download_includes` 包含 `tables`（未指定时沿用默认全量输出），且 table document 的
`readableTableHTML(content)` 结果非空时，Writer 才会把这组精确 bytes 写入 `tables/*.html`。
相同 bytes 在一个 ZIP 中只物理写入一次；内容只出现一次，且 document ID 在全部 Documents
中唯一、安全时保留 `tables/<id>.html`。这里安全 ID 必须以 Unicode 字母或数字开头，后续
仅允许字母、数字、组合标记和 `._-`，并且不超过 256 个 Unicode code point。内容重复，或
ID 为空、重复、不安全时，使用 `tables/sha256-<完整 64 位小写十六进制>.html`；显式路径与
其他内容的 hash 路径冲突时，显式路径也回退为自身的 hash 路径。SHA-256 相同仍会比较
canonical bytes，不能因理论 hash collision 误共享资源。

每个 table document 仍保留独立逻辑 block/chunk，并严格按输入 Documents 的位置关联资源：
存在 `{stem}_parse.json` 时在对应 block 顶层写 `table_html_url`，存在
`{stem}_chunks.jsonl` 时在对应 chunk metadata 写同一路径。Writer 在本地副本上投影该字段，
不会修改输入 Documents；未包含 `tables` 或 canonical bytes 为空时不输出资源和引用，任何
`table_html_url` 都保证指向 ZIP 中实际存在的资源。`table_url` 仍只表示
`metadata.table_image_url` 的表格截图语义，与 `table_html_url` 的 HTML 资源语义互不替代。
parse-v3 的 `text_layer_backfill` runtime stage 位于 table/image 写入之后、页眉页脚候选与
reading order 之前，行为和上述 v2/MinerU 回填一致：只做确定性 text-layer 回填，不调用
OCR/VLM。
v3 受 go-worker 配置 `parser.version_routing.enable_v3` 门控：缺省（未配置）即启用，
仅显式 `enable_v3: false` 作为回滚开关。PDF route 需为 workspace 配置 `PARSER_MINERU`
backend，`docx_route=openxml` / `pptx_route=openxml` 则需配置 `PARSER_OPENXML` backend。
`moi:parse` 已是新建通用工作流的用户默认文档解析节点；`moi:document.parse` 仍为 legacy
internal，供仍依赖 V2 的系统模板和存量 DSL 保持 V2 契约。

`docx_route` / `pptx_route` 的字段默认值仍是 `pdf`（先转 PDF 走通用 MinerU/PDF layout 引擎），
显式 route 始终尊重；standard/enhanced 档位缺省保持 PDF，native 档位在 route 缺省时自动选择
DOCX/PPTX OpenXML。DOCX 的 native 自动路由要求已配置 geometry aligner；`page_selector` 不再
触发 PDF fallback，而是在 OpenXML 语义与 tagged-PDF geometry join 后按原始页码投影。跨页段落
使用 DOCX 语义文本切片，跨页表格使用 canonical row 与页限定 TR/TD/TH MCID；任何片段无法完整
证明、sidecar 不可信或 selector 越界时都明确失败，不返回可能包含未选页内容的结果。无 selector
时，DOCX 几何失败仍遵循 `docx_openxml_strict`：`false` 可继续输出无 bbox 的语义块，`true` 返回
几何错误。`moi:files.write_documents` 生成的 `*_parse.json` 会保留页内投影的
`page_fragment`、`page_fragment_of` 和 `original_row_indices`，使 writer 边界仍可追溯跨页原块与
canonical 行号。OpenXML 标题层级的输出契约是：DOCX/PPTX 直解返回的 `TITLE` block 仅在
`level > 0` 时写入 canonical `metadata.title_level`（同时保留既有 `metadata.heading_level`），
`moi:files.write_documents` 再将其投影到最终 `{stem}_parse.json.blocks[].level` 顶层字段。
同一个层级用于生成 Markdown 标题，1–6 分别输出 `#`–`######`；大于 6 的值仅在 Markdown
标记处限制为六级，`*_parse.json.blocks[].level` 仍保留原始正数。OpenXML 直解路径的语义解析
不经 MinerU/PDF route，直接读取 OpenXML 结构。DOCX 正文段落自动编号由 OpenXML 语义层按
`numPr`（含段落样式继承）、`numbering.xml` 的 `numId -> abstractNum -> ilvl`、`lvlText`、
`numFmt`、start/override/restart 和 suffix 解析为结构化 `numbering_label`；wire 的 `text` /
`raw_text` 仍是原始 `w:t`，用于几何对齐和按页片段投影。Go worker 仅在 page projection 完成后
将 label 组合进最终 block content 一次；匹配的手写前缀不会重复，跨页 continuation 不会伪造
只在首片段渲染的编号。无法可靠格式化的编号保留 raw text 和 provenance，并标记
`numbering_status=unsupported_format`，不根据 `title_level` 猜编号。本次范围仅含 DOCX
顶层正文段落；`numStyleLink` / `styleLink` 间接编号、header/footer、footnote/endnote 和
table-cell 内部段落编号暂不解析。PPTX 的 bbox 直接是
author-space 原生坐标（设计即事实，无需对齐）；它生成的普通（无标签）PDF 仅用于可选视觉裁剪
和页面尺寸校验，不参与语义或几何来源。PPTX 页面尺寸与 layout 文档偏差 &gt;5% 时栅格判定不可用
（`HasPageRaster=false`），`0.5%-5%` 区间会对 bbox 做线性缩放校正并标记
`pptx_scale_corrected`。当 PPTX openxml 路由的
栅格可用时，`source_kind=pptx_picture` 的 IMAGE block 允许用栅格渲染裁剪供 `image_caption` /
`image_ocr` 使用（其余来源仍遵循默认的“无 MinerU crop 即不裁剪”规则，避免对不可信 bbox 伪造
裁剪成功）；group 内未投影旋转/翻转链的图片没有 bbox，会照常降级为 `no_crop_for_image`。

PPTX 经 PDF/MinerU 路由后，由解析器自身确认的页面标题必须在发布 Documents 前满足固定合同：
最终 `TITLE` 若带 `title_source=ooxml_sidechannel`、
`title_source=vlm_ppt_openxml_candidate` 或 `detected_by=vlm_ppt`，则
`metadata.title_level` 必须是语义上精确的整数 `1`。该合同在 `AssembleV1` 前校验，缺失、错误类型、
非整数或其他层级都会终止解析而不发布部分 Documents。它不约束 legacy `.ppt`、未被上述 producer
认领的原始 MinerU TITLE、Native OpenXML 多级标题，也不改变 `moi:files.write_documents` 对通用历史
输入的 missing-level→H1 与超出六级→H6 兼容投影。

### 4.3 `moi:connector.cdh-s3` — CDH 表数据导出至 S3（内部兼容）

`moi:connector.cdh-s3` 保留在运行时用于兼容已有 `cdh-to-mo` DSL/seed 模板，但当前 `semantic_profile.visibility = internal`，不会进入普通用户的能力目录或 NL2DSL 召回面。连接器配置页面完善前，前端不应把它作为可选节点展示。

该节点将 CDH（Cloudera Data Hub）表数据导出为 Parquet 文件并上传至 S3，返回生成的文件 ID 列表。

#### 输入 (input)

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `cdh_config_id` | integer | 是 | CDH 配置 ID |
| `database` | string | 是 | 源数据库名 |
| `table` | string | 是 | 源表名 |

#### 输出 (output)

| 字段 | 类型 | 说明 |
|------|------|------|
| `file_ids` | string[] | 导出生成的文件 ID 列表 |
| `schema_file_id` | string | 表 schema 文件 ID |
| `total_rows` | integer | 导出总行数 |
| `file_count` | integer | 导出文件数 |

#### DSL 示例

```go
dsl.WorkItem("export-cdh", "moi:connector.cdh-s3").
    Input(`{
        "cdh_config_id": 1,
        "database": "my_db",
        "table": "my_table"
    }`)
```

### 4.4 `moi:connector.s3-mo` — S3 文件导入 MatrixOne

从 S3 读取数据文件（CSV/Parquet/ORC/Excel），下载到 worker 临时目录后解析为行数据，
再批量写入 MatrixOne 数据库表。
该 workitem **自动使用 workspace 的 MatrixOne 连接信息**，无需显式传入 host/user/password。
当前节点对普通用户开放，适合把 Catalog 卷中的结构化文件或上游保存节点输出的
`file_ids` 导入到工作区 MatrixOne 表。

Excel 导入根据目标表列类型选择单元格值：MatrixOne 数值目标列读取工作簿中的原始
numeric value，避免 Excel 显示格式造成舍入、百分比或括号负数等数据损失；其他目标列
保留 Excel 格式化显示值，因此日期不会作为 Excel 序列号写入。`DECIMAL` 的 precision、
scale 和最终类型转换由目标表定义，connector 不硬编码金额小数位。所选 worksheet XML 损坏
会使导入显式失败，不会把该 worksheet 的部分解析结果作为成功数据写入。完整性校验仅针对
`sheet_name` 指定的工作表；未指定时仅校验活动工作表，未选择的工作表不参与本次解析和校验。

#### 输入 (input)

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `base_path` | string | 是 | S3 文件基础路径 |
| `table_name` | string | 是 | 目标 MatrixOne 表名 |
| `mo_database` | string | 是 | MatrixOne 目标数据库 |
| `file_ids` | string[] | 是 | 要导入的文件 ID 列表 |
| `delimiter` | string | 否 | CSV 分隔符（默认逗号） |
| `line_separator` | string | 否 | 行分隔符 |
| `start_row` | integer | 否 | CSV/Excel 导入前跳过的起始行数；导入 `catalog.sink.write` 由 `rows+columns` 生成的 CSV 时填 `1` |
| `sheet_name` | string | 否 | Excel 工作表名；不填时使用活动工作表 |
| `quote_char` | string | 否 | CSV 引号字符；不填时使用双引号 |
| `overwrite` | boolean | 否 | 导入前是否清空目标表 |

#### 输出 (output)

| 字段 | 类型 | 说明 |
|------|------|------|
| `total_rows` | integer | 导入总行数 |
| `file_count` | integer | 处理文件数 |
| `table_name` | string | 目标表名 |
| `data_lineage_analysis` | object | 本次运行的数据链路摘要：源文件数、schema 是否存在、文件扩展名分布、目标库表、列数、冲突策略、覆盖标记、起始行和导入行数 |
| `memory_usage_analysis` | object | 导入期间采集的 Go runtime 内存摘要：开始/结束/峰值 alloc、heap、sys、GC 计数，以及单文件解析阶段峰值行数和单元格数 |
| `cache_usage_analysis` | object | worker 可观测到的下载/缓存路径说明；当前 S3-to-MO 通过 `FileService.DownloadToFile` 流式写临时文件，worker 不暴露 fileservice cache counter |
| `disk_usage_analysis` | object | worker 可观测到的本地磁盘使用摘要：临时存储范围、下载字节峰值、临时文件数、XLS 转换临时目录、清理行为和磁盘压力说明 |
| `temp_file_usage_analysis` | object | 临时文件使用摘要：本次临时目录名、下载字节峰值、临时文件峰值、XLS 转换临时目录数量和清理行为 |
| `s3_usage_analysis` | object | fileservice/S3 使用摘要：metadata 查询次数、下载次数、下载字节数和下载耗时 |
| `timing_analysis` | object | 本次导入总耗时、下载、schema 校验、解析和 MatrixOne 写入耗时 |
| `oom_risk_analysis` | array | 结构化 OOM 风险条目，用本次运行的观测值解释 memory、disk、S3 和 overwrite transaction 压力点 |

> 排障提示：下载阶段是流式落盘，内存峰值主要来自解析阶段把单个文件完整物化为
> `[][]string`，以及写入阶段按 batch 展开 insert 参数。长时间或大文件导入时，
> 优先查看 `memory_usage_analysis.peak_parsed_rows`、
> `memory_usage_analysis.peak_parsed_cells`、`disk_usage_analysis.downloaded_bytes`、
> `temp_file_usage_analysis.temp_dir_peak_bytes` 和
> `timing_analysis.parse_duration_ms` / `write_duration_ms`。

#### DSL 示例

```go
dsl.WorkItem("import-to-mo", "moi:connector.s3-mo").
    Input(`{
        "base_path": "/data/exports",
        "table_name": "target_table",
        "mo_database": "my_db",
        "file_ids": [".data.file_ids"]
    }`)
```

> 提示：`file_ids` 可通过 jq 表达式引用上游节点输出，例如 `moi:connector.cdh-s3` 的 `file_ids` 输出。
> 注意：目标数据库与表需要提前创建（表结构需与导入文件匹配）。

---

## 5. 在工作流中选用工作项的建议流程

1. **列出当前可见工作项**：调用 `GET /api/v1/workspaces/:id/workitems` 或 `client.WorkItems(workspaceID).List(ctx)`。
2. **区分用途**：
   - 做**数据转换/条件判断** → 用 `dsl.JQ`，无需从列表中选。
   - 做**子工作流** → 用 `dsl.Subnet`，无需从列表中选。
   - 做**HTTP/Worker 回调** → 用 `dsl.Callback`，无需从列表中选。
   - 做**HTTP 请求**（无需外部 Worker）→ 用 `dsl.WorkItem("步骤名", "catalog:http.request")`，见 §3.1。
   - 做**由 Worker 执行业务逻辑的步骤** → 在列表中选一个 **node_id**（非内建），用 `dsl.WorkItem("步骤名", node_id)`。
3. **查看输入输出约束**：列表中每条元数据有 `input_schema`、`output_schema`（JSON Schema），可据此设计上游 JQ 或数据形状，保证与工作项约定一致。

---

## 6. 参考

- 内建节点元数据定义：`mowl/pkg/engine/builtin_metadata.go`
- 内建 node_id 常量：`mowl/pkg/model/model.go`（如 `JqNodeId`、`SubnetNodeId`）
- 列出逻辑（内建 + 按用户过滤的已注册）：`mowl/pkg/engine/server.go` 中 `ListWorkItems`
- SDK 列出接口：`go-sdk/workitem.go` 中 `WorkItems(workspaceID).List(ctx)`

---

## 7. Stream WorkItems and Routing

Some workitems emit streaming events. The engine can route these events:
- **Externally** to the dynamic service stream (default).
- **Internally** to workflow nodes for normalization, persistence, etc.
- **Both** internal + external.

Routing is controlled via node vars:

| Var | Purpose |
|-----|---------|
| `__stream_targets` | Internal routing targets (JSON array string or comma-separated list). |
| `__stream_scope` | `external` (default), `internal`, or `both`. |

### 7.1 Stream building blocks

Engine builtins for stream routing:

| node_id | Purpose |
|---------|---------|
| `mowl:stream.queue` | Serialize stream events (one-at-a-time) inside the workflow. |
| `mowl:stream.export` | Emit events to the external dynamic service stream (engine builtin). |

Legacy Data Asking stream helpers (`moi:data.event.emit` / `normalize` / `message.persist`) have been removed.

Example routing (internal-only stream for stream-capable nodes):

```yaml
vars:
  __stream_scope: internal
  __stream_targets: '["stream_sink.stream_queue"]'
```

---

## 8. Python Worker

`python-worker` no longer registers built-in static WorkItems. It only provides
Catalog custom-operator function dispatch when YAML config explicitly sets
`function_workitems.enabled=true`. Env-only startup keeps the worker process
alive but registers no WorkItems.
