# 知识库 Workflow 治理接入手册

本文档说明 workflow 解析或生成的文件、数据表如何进入 New MOI
知识库治理体系，以及后续维护时应使用的唯一写入边界。

## Owner 边界

知识库治理写入由 `moi-backend` 负责，包含：

- `knowledge_base_data_domains`
- `knowledge_base_raw_volumes`
- `knowledge_base_sources`
- `knowledge_base_source_job_runs`
- `knowledge_base_segment_versions`
- `knowledge_base_segments`
- source 当前 `segment_version_id/index_version` 指针

`moi-core/workers/go-worker` 不直接写这些治理表，也不调用
`moi-backend`。文件 RAG workflow 负责 parse、text/image index、parsed
docset 和 lineage/register 等数据产物；结构化 import workflow 负责解析文件
并写目标表。两者都不直接写 KB 治理表；治理发布由 backend 基于预先创建并
持久化的 source/job 绑定执行。backend import worker 的 completion hook 可以
在同一进程内调用 session service 同步结构化表绑定，这不是
`moi-core/workers/go-worker -> moi-backend` 回调。

## 写入路径

新增文件的 workflow 产物接入知识库只走 backend 已持久化的 source/job
绑定和显式同步路径：

```text
POST create-with-sources 或 POST /newmoi/semantic-models/:model_id/sources
  -> backend 确保 KB Catalog data domain 和 RAG workflow 定义
  -> backend 创建 source/job 绑定
POST /newmoi/semantic-models/:model_id/source-jobs/reconcile
  -> backend 关联 raw file，并在需要时显式触发 KB workflow
  -> workflow parse/index/lineage/register
POST /newmoi/semantic-models/:model_id/source-jobs/reconcile
  -> backend 观察 file execution，刷新已持久化 source/job 状态
  -> 成功时从 vector rows 发布 segment version
```

历史数据只走 backfill 路径：

```text
GET /newmoi/semantic-models/:model_id/sources
  -> 只读返回 managed rows 和 legacy_unbound candidates
POST /newmoi/semantic-models/:model_id/sources/backfill-legacy
  -> 只创建或补齐缺失的治理 source/job 关系
POST /newmoi/semantic-models/:model_id/sources/:source_row_id/segments/import-initial
  -> 用户显式传 source row ID 时，把历史 vector rows 导入为 v1 chunks
```

`GET` 接口必须保持只读。不要把 source/job 创建、vector row 迁移或
segment version 发布藏在 `GET /sources`、`GET /source-jobs` 或文档详情
`GET` 后面。

### 资源选择、添加与同步总览

create/append 接口不接收文件内容。本地文件在调用接口前已经上传到 Catalog
并取得 `file_id`；结构化文件还会先取得 `conn_file_id` 并完成预览配置。
Catalog 批量选择以 `source_selections` 表达，具体叶子由 backend 在请求内
分页展开和校验。

```text
用户打开资源选择器
    |
    +-- 本地非结构化文件
    |     POST /catalog/file/upload
    |     -> 得到 file_id
    |
    +-- 本地结构化文件（CSV/XLS/XLSX）
    |     POST /connectors/file/upload
    |     -> 得到 conn_file_id
    |     POST /connectors/file/preview
    |     -> 配置 sheet、列、目标表和导入规则
    |
    +-- Catalog 文件或表
          查询 Catalog tree / database table leaves / volume file leaves
          -> 生成 source_selections
    |
    v
sources / source_selections
    |
    v
POST /newmoi/semantic-models/create-with-sources
或 POST /newmoi/semantic-models/:model_id/sources
    |
    | [1] backend 展开并校验 Catalog 选择
    | [2] 确保 KB database、raw/processed volume
    | [3] 有文档资源时部署 KB 专属 RAG workflow 定义
    | [4] 写 source.status=pending 和 source job intent
    | [5] 结构化本地文件在 metadata 提交后立即创建 import_task
    v
请求返回
    |
    +-- 结构化 import task 完成
    |     backend worker completion hook 可直接同步表 final binding
    |
    v
资源页 GET /sources + GET /source-jobs
    |
    v
reconcile_required ?
    | 否                              | 是
    v                                 v
继续只读轮询或展示终态          POST /source-jobs/reconcile
                                      |
                                      | 有限批次：
                                      | - 原子领取 copy/load/table_clone/rag_ingest
                                      | - 观察 workflow/import task 终态
                                      | - 补齐 lineage source/job
                                      | - 同步表绑定或发布 segment version
                                      v
                              final binding 完整 ?
                                  | 否              | 是
                                  v                 v
                       source.status 保持 pending  source.status=succeeded
                       外部任务 running 时继续 GET 顶层 reconcile_required=true
```

四层状态各自只回答一个问题：

```text
workflow/file execution/import task = 真实长任务是否完成
KB source job                       = 当前资源动作及 KB 同步推进到哪一步
顶层 reconcile_required             = 全 KB 是否仍需 POST 推进或观察有限批次
source.status + final binding       = 资源是否已经在当前 KB 可用
```

页面同步不承担长任务存活性。用户关闭页面后，Catalog delivery、Mowl
task/case 和 import task 仍由各自 owner 继续执行。文件 RAG 的 final binding
可以在下次打开资源页时发布；结构化 import task 正常完成时还会由 backend
worker completion hook 主动调用同一套同步逻辑，页面 reconcile 是恢复和补偿
入口。

## 参与模块与使用边界

| 模块 | 负责的功能和状态 | 知识库添加资源时的用法 |
| --- | --- | --- |
| `moi-frontend/modules/shared-moi-components` | 本地文件预上传、结构化预览、Catalog 选择表达式 | 只生成 `sources/source_selections`，不创建 KB job，不执行 copy/clone/RAG |
| `moi-backend/pkg/session` | KB data domain、source/job、segment version、final binding | create/append 登记意图；reconcile 领取 KB 步骤、观察外部任务并发布治理结果 |
| `moi-backend/pkg/catalog` | MO database/table DDL、Catalog metadata/permission 同步适配 | 创建 KB database；执行 Catalog table 的 `CREATE TABLE ... CLONE`；调用 moi-core `SyncMetadata` |
| `moi-backend/pkg/dataconn` 与 `moi-backend/pkg/worker` | `import_task/import_task_run` 业务状态和结构化载入编排 | 创建结构化 import task；以 `external:import-file` WorkItem 编排实际载入并回写 `structured_table_results` |
| `moi-core/catalog` | Catalog 元数据、file-volume 关联、volume trigger delivery、workflow execution 投影、data asset/lineage | 提供 `AddFiles/TriggerFiles`、workflow deployment/execution、file execution 查询和 lineage 登记边界 |
| `moi-core/mowl` | workflow definition/version 对应的 task、case、WorkItem 真实运行状态 | 调度 KB RAG、`import_file` 和 `s3_to_mo` workflow；不写 KB source/job 表 |
| `moi-core/workers/go-worker` | 单个 WorkItem 的解析、embedding、向量/文件/lineage 产物和 `s3_to_mo` 写表 | 执行标准 RAG 节点和结构化文件到 MO 的实际数据处理；不判断 KB source 是否完成 |

每个 KB data domain 记录当前 `model_id` 对应的 Catalog、KB database、
`raw_document` 和 `processed` volume；本地图片、音视频或结构化文件按需建立
`raw_image`、`raw_audio_video` 或 `raw_structured` volume。跨模块治理绑定和
source 身份使用 Catalog 资源 ID；名称只允许由 Catalog owner 在资源创建重试
时解析已知 database/volume，不能据此反推 source。

当前默认 KB RAG workflow 的 volume trigger 只绑定 data domain 的
`raw_document` volume。local file intent 虽会为图片和音视频建立
`raw_image/raw_audio_video`，但当前没有为这两个 volume 部署对应 trigger 的
代码路径，因此本文确认闭环的普通文件路径仅指文档类文件。下文“图片索引”是
文档解析后对 page image/visual object 建索引，不代表原始图片或音视频文件已
接入同一条默认 RAG workflow。

同名 `load` 不代表统一的载入框架。四类 KB job 的当前语义是：

```text
copy        = catalog_file 写路径占位 job：确认 source/file 绑定后直接 succeeded，
              不把用户 Catalog 文件 AddFiles 进 KB raw volume
load        = 本地非结构化文件的 raw volume 关联（AddFiles）
              或结构化 import task 的状态观察与结果表同步
table_clone = backend 执行 MO table clone 并同步 Catalog metadata
rag_ingest  = 复用向量，或触发/观察文件 workflow，最后发布 segment version
```

`load` 中的 `AddFiles` 只新增 `volume_files` 关联和引用计数，不复制文件二进制。
`catalog_file` 的 `copy` **不**调用 `AddFiles`：文件仍归用户 volume；请求里的
`volume_id` 只用于写时 IAM/成员校验，并作为 source 行上的定位字段。
`table_clone`、结构化 import task 和 RAG workflow 是三条不同执行路径，不能根据
其中一条的完成状态推导另外两条完成。

### 普通文件 RAG 的跨模块调用

本地非结构化文件在 `load` 关联到 KB raw volume 后进入 RAG；Catalog 文件保持在
用户原 volume，由 `source_ref.volume_id`（source 上记录的用户 volume）定位读取，
与 local 文件共用后续 Trigger/workflow 路径：

```text
KB source/job (moi-backend)
  |
  | reconcile 原子领取 copy（catalog_file）或 load（local_file）
  v
catalog_file copy: 不 AddFiles，直接 finish
local_file load:   VolumeFiles.AddFiles -> KB raw volume
  | KB workflow 的 auto_dispatch=false，因此这里不自动触发 RAG
  v
copy/load succeeded，source 仍 pending
  |
  | reconcile 原子领取 rag_ingest
  | 显式调用 VolumeFiles.TriggerFiles（按 source 记录的 volume/file）
  v
Catalog volume_trigger_delivery
  | 持久化派发、重试和 execution/task/case ID
  v
Catalog workflow_execution（产品层状态投影）
  |
  v
Mowl task -> case -> WorkItems（真实 workflow 运行状态）
  |
  v
go-worker
  moi:catalog.source.read
  -> moi:document.parse
  -> moi:parser.split.documents.length
  -> moi:knowledge.index.build
  -> moi:files.write_documents
  -> moi:catalog.sink.write
  -> moi:data.lineage.register
  |
  +--> Catalog file/volume、data_asset/data_derivation/parsed_manifest
  +--> MatrixOne text/image vector rows
  |
  v
Catalog workflow reconciler 把 Mowl case 终态同步到 workflow_execution
  |
  v
KB 页面后续 reconcile
  -> ListFileExecutions
  -> 校验当前 KB vector table/embedding binding
  -> 发布 segment version
  -> source final binding succeeded
```

Mowl task/case 和 Catalog delivery 的执行、重试、终态收敛都不依赖 KB 页面。
页面只在 `reconcile_required=true` 时推进 KB 侧的触发或 final binding 同步。
启用图片索引时，workflow 还会执行 `moi:document_visual.parse` 和
`moi:document_visual.index.image`；文本和图片 rows 都必须通过当前 KB binding
校验后才能发布。

### 本地结构化文件载入任务

结构化本地文件不是标准 RAG workflow。它使用 backend 载入任务作为业务
owner，再通过两层 system workflow 执行实际写表：

```text
共享资源选择器
  -> connector upload + preview
  -> file_id + conn_file_id + table_config
  |
  v
KB create/append metadata 提交
  |
  v
backend dataconn: import_task + import_task_run
  | 业务载入任务状态 owner
  v
Catalog system workflow: import_file
  -> Mowl task/case
  -> backend worker: external:import-file
       | 导入编排，必要时创建目标表并 SyncMetadata
       v
       Catalog system workflow: s3_to_mo
       -> Mowl task/case
       -> go-worker: moi:connector.s3-mo
       -> 解析 CSV/XLS/XLSX 并写入 MO target table
  |
  v
backend worker
  -> 回写 structured_table_results
  -> import_task/import_task_run 终态
  -> ImportCompletionHook
  -> RunPendingKnowledgeBaseSourceJobs
  -> kb_table_id/db_name/table_name + semantic model table scope
  -> source/load succeeded
  |
  +-- completion hook 未完成 KB 同步时
        页面 GET 投影 reconcile_required=true
        -> POST reconcile 观察同一个 import task 并补齐 final binding
```

KB load job 不复制 import task 的内部状态机，也不直接执行文件解析；它只保存
`operation_id=import_task:<task_id>`，观察 task/run 终态，并把
`structured_table_results` 转成当前 KB 的最终表绑定。

结构化 metadata 阶段的 source 已使用 `source_type=catalog_table`，但会暂存
上传文件的 `source_file_id/kb_file_id`；final binding 成功后清除临时文件身份，
改写为 `kb_table_id/db_name/table_name`。一个多 sheet import task 返回多张表时，
第一张表复用原 source/load job，其余结果各自新增 table source/load job。

### Catalog 表 clone

Catalog 表不创建 workflow 或 import task：

```text
KB table_clone job
  |
  | reconcile CAS 领取
  v
moi-backend catalog dataCenterService
  -> MatrixOne: CREATE TABLE target CLONE source
  -> moi-core Databases.SyncMetadata
  -> permission sync；任一步失败时回滚已创建目标表/metadata
  |
  v
moi-backend session
  -> kb_table_id/db_name/table_name
  -> semantic model table scope
  -> source/table_clone succeeded
```

当前没有 moi-core `CloneTableForKnowledgeBase` API；这个名字是 backend
session 对 `moi-backend/pkg/catalog` owner 能力的适配接口。

### 前端如何使用

数据侧和智能体侧共用资源选择器及 `shared-moi-api/knowledge` 契约：

- create/append 成功后重新读取 `/sources` 和 `/source-jobs`，不使用请求体在
  本地伪造 source/job 状态。
- 页面只依据 backend 返回的 `reconcile_required` 决定是否 POST reconcile；
  `pending/queued/running` 或 `reconcile_required=true` 时每 5 秒轮询 job。
- 数据侧以知识库更新权限控制追加和自动 reconcile；智能体侧调用同一 backend
  API，最终权限仍由 backend 强制校验。
- Catalog 的“全选当前范围”保持为带 filters、selected/excluded IDs 的
  `source_selections`；前端不展开成大量 source，backend 在请求内分页展开。

## Source 复用与资源边界

向已有知识库追加 Catalog 文件或 Catalog 表时，`moi-backend` 先按同一
`model_id + source_file_id/source_table_id` 查找已有治理 source。命中已有
绑定后的文件或表 source 时，复用已有 `kb_file_id` 或 `kb_table_id/db_name/
table_name`，不重新复制文件、不重新 clone 表，也不覆盖 `enabled`、
`expires_at`、`tags`、`force_enabled_after_expiry`、当前
`segment_version_id/index_version` 等治理字段。semantic model scope 只
补缺失的 file/table 引用，并保持去重。

复用不是“完成态”判断。Catalog source 的重试边界是已持久化的
source/job 关系：文件 copy 完成后即有 `kb_file_id`，表 clone 完成后即有
`kb_table_id/db_name/table_name`。后续 workflow/RAG 解析或 job 状态通过
作业列表和显式 reconcile 继续推进，因此重复追加同一 Catalog 资源应进入
已有 source/job，而不是创建并行 source。

历史上已经存在的重复 source 只按原数据展示和保留；追加路径不自动合并、
不改写历史 owner 字段。新建知识库时不做旧 source 复用，因为新
`model_id` 本身就是新的治理边界。

已移除的 source 是显式的“解除知识库关联”记录，不是可见 active
source。用户再次追加同一 Catalog 文件或 Catalog 表时，backend 会重新
激活同一 `source_id`，重新创建 source jobs，并按当前知识库配置继续
copy/reuse/clone；自动 backfill 不会重新激活 removed source。

Catalog 文件可以复用已有解析产物和向量 lineage。复用只改变当前知识库
的治理关系和 segment version，不改变原 Catalog 文件所有权。复用过程中
如果目标 KB vector table 已经存在 deterministic row id，backend 必须先
显式读取并校验 metadata；metadata 与当前 KB 绑定兼容时复用该 row，不能
再次插入同一 row id，也不能吞掉 duplicate 错误伪装成功。

向量复用兼容边界：

- 文本向量优先以 `embedding_model` 作为语义空间边界；历史 metadata 没有
  `embedding_model` 时，才按 `vector_table` 或空 metadata 兼容旧数据。
- 图片向量以 `image_embedding_model`、`image_embedding_dimension`、
  `image_preprocess_version` 和 `image_distance_metric` 作为语义空间边界。
  `image_embedding_backend_id` 是服务实例/路由元数据，不作为复用阻断条件。
- 表结构一致只说明 row 可以读写，不说明向量空间可比；不同 embedding
  model、维度、图片预处理版本或距离度量不能混用。

## 资源复用与 Chunk 版本状态模型

资源复用和 chunk 版本是一条单向状态链。前一层成功只能作为后一层的
前置条件，不能直接投影成当前 KB 可检索：

```text
Catalog/local resource
  |
  v
[source 关系]
  knowledge_base_sources.source_id
  |
  v
[raw 绑定]
  file: kb_file_id
  table: kb_table_id/db_name/table_name
  |
  v
[chunk 候选]
  current KB text/image vector rows
  |
  v
[committed chunk version]
  knowledge_base_segment_versions
  knowledge_base_segments
  |
  v
[active chunk version]
  knowledge_base_sources.segment_version_id/index_version
  |
  v
GET sources/source-jobs, LLM/Agent retrieval
```

状态层含义：

| 层级 | 关键字段/数据 | 证明什么 | 不证明什么 | 下一步 |
|---|---|---|---|---|
| source 关系 | `model_id + source_file_id/source_table_id`、`source_id` | 当前 KB 有治理关系，可以承接 job 和 source 状态 | raw 文件已进入 KB、chunks 已可检索 | 建立或复用 raw 绑定 |
| raw 绑定 | 文件 `kb_file_id`；表 `kb_table_id/db_name/table_name` | 资源已进入当前 KB 的存储边界 | RAG chunks 已生成或可检索 | 文件进入 RAG 复用/解析；表可完成 |
| chunk 候选 | 当前 KB text/image vector rows，按 `kb_file_id + index_version + chunk_index` 成批 | 有可发布的 chunk 候选数据 | 当前 source 指针已切换 | 发布或复用 segment version |
| committed chunk version | `knowledge_base_segment_versions`、`knowledge_base_segments` | 某个 `index_version` 已被 backend 提交为 chunk 版本 | 当前资源正在使用该版本 | 切换 source 当前指针 |
| active chunk version | `knowledge_base_sources.segment_version_id/index_version` | 当前 KB 资源完成；资源列表和 LLM/Agent 检索可使用 | 新 workflow 执行结果一定已覆盖旧版本 | 作为当前检索版本，直到新版本提交并切换 |

证据边界：

| 证据 | 能证明 | 不能证明 |
|---|---|---|
| `workflow_execution_id` 或按 `file_id/kb_file_id` 查询到的最新 file execution | 同一文件的最新 workflow 存在、运行中、成功或失败 | 当前 KB source 已有可检索 chunks |
| 旧 KB 的 completed execution | 同一个文件曾经被其他流程处理过 | 当前 KB 已完成 |
| 当前 KB vector rows | 当前 KB 有 chunk 候选数据，且 rows 必须落在目标 KB 的 vector table 和 embedding model 绑定内 | source 当前指针已切换 |
| `knowledge_base_segment_versions` | chunk 版本已提交 | source 当前正在使用该版本 |
| `source.segment_version_id/index_version` | 当前 KB 资源 active chunk version | 新解析结果一定已经发布 |

完成态判定顺序：

| 步骤 | 判断 | 结果 |
|---|---|---|
| 1 | 按 `file_id/kb_file_id` 查询最新 file execution | 只得到过程状态，不直接完成 source |
| 2 | execution completed 后，校验 execution/lineage/vector rows 是否对应目标 KB 的 vector binding | 匹配才进入发布；不匹配不能复用 |
| 3 | 目标 KB text/image vector table 中存在兼容 rows，且 `index_version` 可发布 | 创建或复用 segment version，并切换 source 当前指针 |
| 4 | 没有目标 KB rows，但目标 KB workflow execution 正在 running/queued | 保持处理中，继续轮询/reconcile |
| 5 | 没有目标 KB rows，也没有目标 KB 正在跑的 execution | 触发默认 KB workflow |

目标 KB 的 vector binding 包含文本向量表、图片向量表、文本 embedding
model、图片 embedding model、图片维度、图片预处理版本和距离度量等当前
KB 配置。资源已有向量表但 vector table 或 model 与目标 KB 不一致时，
不能当成复用完成态。

同一个资源被再次解析到同一个 KB 时，必须新增或复用一个已提交
segment version，不覆盖旧版本：

- 当前 `index_version` 已经提交且 source 指针已指向该版本：保持幂等，
  不创建重复版本。
- 当前 KB 出现更新的兼容 `index_version`：创建新的
  `knowledge_base_segment_versions` 和 `knowledge_base_segments`，再切换
  source 当前指针。
- workflow 失败、vector rows 缺失、metadata 不兼容或图片表读取失败：
  不切换 source 当前指针；已有已提交版本继续保留。

Catalog 文件跨 KB 复用只在当前 KB 绑定完整兼容时成立：

| 场景 | 判定 | chunk version 动作 | source/job 投影 |
|---|---|---|---|
| 旧 KB 或外部 workflow 有 completed execution，但 rows 不在当前 KB vector 绑定，或 vector table/model 不匹配 | 不可复用为完成态 | 不发布当前 KB segment version | 继续 pending/running 或 `reconcile_required=true` |
| 外部 workflow 选择同一资源并指定当前 KB，rows 写入当前 KB text/image vector 绑定 | 可作为当前 KB 候选结果 | 发布新的当前 KB segment version，切换 source 指针 | `rag_ingest` 可 succeeded，source 可 succeeded |
| 外部 workflow 指定其他 KB，rows 只在其他 KB vector table | 不属于当前 KB 结果 | 不发布、不切换 | 当前 KB 不得 succeeded |
| 当前 KB text-only，历史文本向量兼容且可接入当前 KB 绑定 | 可复用文本 | 发布只包含文本 chunks 的 segment version | 可 succeeded |
| 资源已有向量 rows，但 vector table 或 embedding model 与目标 KB 不一致 | 不可复用 | 不发布新 active version | 保留/建立 raw 绑定并重跑目标 KB workflow |
| 当前 KB 需要图片向量，历史结果缺少图片 rows 或图片配置不兼容 | 不可复用为完成态 | 不发布新 active version；保留旧版本 | 保留/建立 raw 绑定并重跑当前 KB workflow |
| 当前 KB 比历史结果多一个必需 vector 绑定 | 不可复用为完成态 | 不发布新 active version | 重跑当前 KB workflow |
| 历史资源所在 KB 比当前 KB 多出当前 KB 不需要的 vector 绑定 | 额外 rows 不参与当前 KB | 只发布当前 KB 需要且兼容的 chunks | 不阻断 text-only 复用 |

`source-jobs` 的投影必须跟随上述状态模型：有旧 completed file
execution 但没有当前 KB active segment 指针时，job 可以展示为需要
reconcile 或进行中，不能因为“同 file_id 有完成执行”而投影成当前 KB
已完成。

### 页面同步信号

`GET /sources` 和 `GET /source-jobs` 只观察状态。`GET /source-jobs` 顶层
`reconcile_required` 是后端计算的唯一页面写入信号；它不表示 source 已成功或
已失败，前端不得根据 item、job type、外部执行状态或 binding 自行推导。

| 当前观察 | `reconcile_required` |
| --- | --- |
| job 为 `pending` 或 `queued`，且 source 未 `removed/failed` | `true` |
| 外部任务或本地已领取步骤为 `running` | `true`，POST 按最久未检查顺序观察 |
| job/外部任务为 `succeeded`，但 final binding 未完整 | `true` |
| job/外部任务为 `failed/cancelled`，但 source 尚未持久化为 `failed` | `true` |
| source 已 `failed`、`removed`，或 `succeeded` 且 final binding 完整 | `false` |

因此 `job_status=succeeded + reconcile_required=true` 是正常组合：动作已经
完成，但 source 尚待发布 segment 或写入最终表绑定。`items` 只保留最多 32 个
persisted 候选诊断视图；显式 file/table、lineage-only 等 legacy 工作只影响顶层信号，因此允许
`items=[]、total=0、reconcile_required=true`。GET 不创建 source/job，后续 POST
reconcile 才执行 backfill 和发布。

## Reconcile Contract

`source-jobs/reconcile` 只处理 backend 已创建的明确绑定：

- `knowledge_base_sources.source_id`
- `knowledge_base_source_job_runs.job_id`
- `knowledge_base_source_job_runs.kb_file_id` / `kb_table_id`
- `knowledge_base_source_job_runs.operation_id`
- `knowledge_base_source_job_runs.workflow_execution_id`

RAG workflow job 由 backend 预创建为 `rag_ingest`，状态可以是
`pending`、`queued`、`running` 或 `succeeded`。reconcile 可以按
`file_id/kb_file_id` 查最新 file execution 并刷新 workflow 状态，但
workflow completed 只是一类过程证据；只有这个 execution、lineage 或
vector rows 能证明结果落在当前 KB 的 vector binding 内，并且 source
当前指针已切换到可发布的 segment version 时，文件 source 才算 RAG
完成。没有当前 KB rows、也没有当前 KB 正在跑的 execution 时，reconcile
应触发默认 KB workflow。

`workflow_trigger:<workflow_id>` 记录的是触发来源，不作为 chunk 发布的
过滤边界；同一个 KB 文件后续可能由替换 workflow 或外部指定当前 KB 的
workflow 重新解析，当前 segment version 必须跟随该 `kb_file_id` 在当前
KB vector 绑定下最新可发布的解析结果。

禁止按文件名、路径、表名、vector table 名或 lineage 单独反推 KB source。
缺少明确 source/job 绑定时必须失败或保持未处理状态，不能伪造成成功。

## Source/Job 状态流转图

知识库资源流转分两层：

```text
knowledge_base_sources
  表示资源当前状态和最终绑定结果：
    source_type
    status
    kb_file_id
    kb_table_id
    db_name
    table_name
    segment_version_id
    index_version

knowledge_base_source_job_runs
  表示这个资源下一步要执行的动作：
    source_id
    job_type
    job_status
    operation_id
    workflow_execution_id
```

核心规则：

```text
source_type/status/final binding 决定“资源是否完成”
job_type/job_status 决定“reconcile 下一步做什么”
operation_id 只记录动作来源或外部任务标识，不能单独证明动作已完成
workflow_execution_id 或 file execution 只能证明 workflow 真实存在
active segment 指针才能证明当前 KB chunks 已完成
```

`source-jobs/reconcile` 每次独立面向全 KB 选批，不接收或复用 GET `items`。
处理顺序固定按阶段推进，不按 `job_id` 或创建时间猜测依赖：

```text
POST /newmoi/semantic-models/:model_id/source-jobs/reconcile
  |
  v
[1] backfill legacy sources/jobs
    把已经关联到知识库但缺少 source/job 投影的历史资源补齐
  |
  v
[2] 按阶段和固定批次执行
    fast-bind (copy/load) -> table_clone -> structured finalize/wait
      -> RAG finalize -> RAG dispatch -> RAG running observation
  |
  v
[3] job 执行后回写 source final binding
  |
  v
[4] final binding 完整
      source.status = succeeded
    否则
      source.status 继续 pending
```

running observation 按 `updated_at ASC, job_id ASC` 选择最久未检查的任务，并在
本轮检查后显式更新 `updated_at`。超过单批上限的 running job 会在后续轮次自然
轮换，不需要 cursor、客户端累计集合或轮次状态。

### Catalog File

Direct `sources[]` catalog_file items and `source_selections.volume_files` must
include an authoritative `volume_id`. Backend does not guess volume from
`volume_files` when `volume_id` is missing.

Identity and permission model:

- **Business identity** = `file_id` only (`source_id` is stable on model + file).
- **`volume_id`** = write-time gate (caller may add this file from this volume) and
  storage location pointer on the source row; not a second identity axis.
- Same `file_id` claimed twice in one request (any volumes) is rejected.
- Same `file_id` already active on the model under another `volume_id` is rejected
  (fail-closed); same volume reuses the existing source.
- Selection expand must not silently drop cross-volume collisions.
- Files remain user-owned; remove source only unlinks the KB relation (no delete
  of the Catalog file, no reverse Unlink from user volume).
- **Job-frozen principal** (offset121 columns on `knowledge_base_source_job_runs`):
  create freezes `runtime_actor_moi_user_id` + `runtime_effective_role_id`
  (create-time MOI user + `VerifiedEffectiveRole`). Optional
  `runtime_is_workspace_owner` is stored for **audit only** and is **not** used
  as a live privilege-class bypass on deferred dispatch. `rag_ingest` refuses
  create if actor/role is missing so reconcile never inherits a callback/system
  principal. Deferred claim/Trigger **rehydrates** actor via
  `coreclient.WithIdentity` and restores the frozen role as
  `VerifiedEffectiveRoleID` only — it clears `IsWorkspaceOwner` /
  `BusinessActionAuthorized` / allow facts and never calls
  `principaltrust.MarkPrivilegedCoreIAM`. Reconcile HTTP caller identity is never
  used for workflow dispatch and is never adopted onto historical rows.
- **Deferred `rag_ingest` auth order** (under rehydrated job principal):
  1. `semantic_model.use` on the model → action-scoped allow ctx for built-in KB
     workflow / vector reuse hops
  2. catalog_file only: try vector reuse **before** `volume.read` (reuse does not
     need volume)
  3. if workflow dispatch is still required: `volume.read` on the recorded user
     volume once as a **gate only** (allow context discarded; never cross into
     workflow/run hops)
  Both steps always re-enter Core PDP under the frozen role (binding/lifecycle
  and current policy checked live). Deny fails closed (job failed); Core
  unavailable is retryable (job stays pending/queued). Write-time outer allow is
  never reused. offset121 adds freeze columns only — it does **not** fabricate
  admin role or privilege-class for historical rows. Jobs without a create-time
  freeze stay fail-closed and undispatchable.

```text
source_type = catalog_file

初始：
  source.status = pending
  jobs:
    copy        pending/queued
    rag_ingest  pending/queued
  raw_volume_id = 请求 volume_id（用户源 volume，不是 KB domain raw）

copy:
  |
  |-- 不调用 VolumeFiles.AddFiles
  |-- 确认 source_file_id/kb_file_id 与记录 volume
  |-- 直接 finish copy job
  v
copy succeeded
  operation_id = catalog_file_link:<file_id>  （历史 wire/DB 前缀，语义是 bind 非 AddFiles）
  source_file_id/kb_file_id 确认

rag_ingest:
  |
  |-- 前置条件：copy succeeded
  |
  |-- 先尝试复用已有解析产物和向量结果
  |     |
  |     |-- 复用成功
  |     |     rag_ingest succeeded
  |     |     写入 source.segment_version_id/index_version
  |     |     不触发 workflow
  |     |
  |     |-- 复用失败
  |           触发 KB RAG workflow（按用户 volume + file 定位）
  |           operation_id = workflow_trigger:<workflow_id>
  |
  |-- workflow 已存在
  |     通过 workflow_execution_id
  |     或当前 KB kb_file_id 关联的 file execution
  |     跟踪 workflow 状态
  |
  |-- workflow succeeded
        校验当前 KB vector rows 是否可发布
        发布 segment version
        写入 source.segment_version_id/index_version
        source.status = succeeded
```

`copy` 只推进 job/source 状态，不把文件收编进 KB raw volume，也不复制向量表。
Catalog file 的 `rag_ingest` 表示“确保当前 KB 下有可用 segment/index”，具体动作
可以是复用向量、触发 workflow、跟踪 workflow 或发布 segment，不等价于必然跑
workflow。

### Local File 非结构化文档

```text
source_type = local_file

初始：
  source.status = pending
  jobs:
    load        queued/running
    rag_ingest  pending/queued

load:
  |
  |-- create/append 前文件已上传到 Catalog，并取得 file_id
  |-- 检查 file_id 是否已关联 KB raw volume
  |-- 未关联时调用 VolumeFiles.AddFiles
  |-- source_file_id/kb_file_id 继续使用同一 file_id
  v
load succeeded

rag_ingest:
  |
  |-- 前置条件：load succeeded
  |
  |-- 没有 workflow execution
  |     触发 KB RAG workflow
  |     operation_id = workflow_trigger:<workflow_id>
  |
  |-- 已有 workflow_execution_id
  |     跟踪 workflow 状态
  |
  |-- workflow succeeded
        发布 segment version
        写入 source.segment_version_id/index_version
        source.status = succeeded
```

### Local File 结构化上传

```text
结构化文件，例如 csv/xlsx 表格类文件

jobs:
  load

create/append metadata 提交成功：
  |
  |-- 立即创建 import_task/import_task_run
  |-- load.operation_id = import_task:<task_id>
  |-- load running
  v
backend/Mowl/workers 执行结构化 import
  |
  |-- import task 解析并写出目标表
  |-- 回写 structured_table_results
  v
completion hook 或页面 reconcile：
  |
  |-- 观察 import task/run 终态
  |-- 同步表 final binding
  v
load succeeded

写回：
  kb_table_id
  db_name
  table_name

final binding 完整：
  source.status = succeeded
```

结构化上传产出表绑定，不走 `rag_ingest`。

### Catalog Table

```text
source_type = catalog_table

jobs:
  table_clone

table_clone:
  |
  |-- CloneTableForKnowledgeBase
  |-- 把原始表 clone 到 KB database
  v
table_clone succeeded

写回：
  kb_table_id
  db_name
  table_name

final binding 完整：
  source.status = succeeded
```

Catalog table 不通过 workflow completion 反推 source，也不走
`rag_ingest`。

### 状态和 Final Binding

```text
job_status = pending
  动作还没开始，reconcile 可以领取并推进

job_status = queued
  动作已排队或已认领，reconcile 仍可以检查是否需要继续推进

job_status = running
  动作进行中，前端继续轮询 GET source-jobs/sources
  顶层 reconcile_required 为 true，POST 按最久未检查顺序观察有限批次

job_status = succeeded
  只表示该 job 动作完成，不代表 source 一定完成
  还必须检查 source final binding

job_status = failed
  资源通常进入 failed，普通 reconcile 不自动重试
```

Final binding 判断：

```text
Catalog table / structured table:
  kb_table_id 有值
  db_name 有值
  table_name 有值
  => source 完成

普通文件 load/copy:
  kb_file_id 有值
  => 文件绑定完成

RAG 文件:
  kb_file_id 有值
  segment_version_id 有值
  index_version 有值
  => RAG 发布完成
```

RAG trigger 恢复边界：

```text
rag_ingest job:
  job_status = pending/queued
  operation_id = workflow_trigger:<workflow_id>
  workflow_execution_id 为空
  ListFileExecutions(kb_file_id) 查不到 execution

含义：
  backend 有过触发意图，但没有证据证明 workflow execution 已存在

reconcile 行为：
  前置 copy/load 已完成时，可以重新触发或恢复 rag_ingest
```

## 文件 Workflow 语义

文件 source 的 RAG workflow 成功后，backend 读取该 `kb_file_id` 最新
vector rows；只有最新 vector `index_version` 比 source 当前指针更新时，
才发布新的 segment version。

图片索引配置语义：

- 完整图片 embedding 配置决定 workflow 是否尝试为 page image 或 visual
  object 建图像索引；`image_vector_table` 是 backend 生成并治理的目标表
  绑定，create-with-sources 请求体里单独传同名字段不作为启用信号。
- 发布 segment version 时，后端按绑定的 `image_vector_table`、
  `kb_file_id` 和 `index_version` 读取实际 rows；存在匹配 image rows
  才发布图片 chunks。
- 某个文档解析结果没有 page image 或 visual object、因此没有匹配
  image rows 时，segment version 可以只发布文本 chunks；这是“无可索引
  图片”的业务结果，不是降级。
- 图片向量表不可见、schema 不匹配或读取失败必须显式报错，不能被当成
  text-only 成功。

历史 text vector rows 可能缺少 `chunk_index/index_version/disabled` 或
仍为 `index_version=0`。`segments/import-initial` 是用户显式 POST，可在
导入前补齐这些旧字段，把旧 rows 接入当前 KB segment version 治理；该
兼容路径不得扩展成 GET 隐式迁移，也不得吞掉图片表/schema/read 错误。

重复执行语义：

- 同一次结果或同一个当前 `index_version`：不创建重复 segment version；
  job 可以被标记为 succeeded。
- 重新执行并产生更新的 vector `index_version`：backend 导入 chunks，
  创建新的 committed `knowledge_base_segment_versions`，写入
  `knowledge_base_segments` 和召回统计 key，然后切换
  `knowledge_base_sources.segment_version_id/index_version`。

失败语义：

- 标记 source/job failed。
- 不修改 `enabled`。
- 不切换当前 segment 指针。
- 已有 committed 版本继续保留；只要 source 治理状态允许检索，用户仍可
  手动设为当前版本。

重试边界：

- 显式 `source-jobs/reconcile` 会捞取已持久化的 `rag_ingest`
  `pending`、`queued`、`running`、`succeeded` job；这些状态保留在
  作业列表中是为了让 workflow 完成后继续发布 chunk version。
- 结构化本地文件导入会捞取已写入 `import_task:<task_id>` 的 load job。
- 已明确判定为依赖失败并标记 `failed` 的 job 不会被普通 reconcile
  自动重试；需要用户从作业列表触发重试、重新追加 source，或由后续专门
  retry 能力处理。重新追加同一 Catalog source 会复用已有治理 source，
  由该 source 的作业状态承接后续处理。

## 表资源语义

现有 KB table source 使用 owner-owned flow：

```text
AppendModelSources(catalog_table)
  -> create source/job rows from source_table_id
  -> request 返回
页面 POST source-jobs/reconcile
  -> CAS 领取 table_clone
  -> backend Catalog owner 执行 CloneTableForKnowledgeBase
  -> update kb_table_id/db_name/table_name
  -> update semantic model table scope
```

当前不通过 workflow completion 反推或创建 table source。不要用
database/table name 猜测 source。

## 删除与向量保留

删除单个 source 时，backend 会先更新 semantic model scope，再删除
该 source 关联的 `knowledge_base_chunk_recall_stats`、
`knowledge_base_segments`、`knowledge_base_segment_versions` 和
source/job 治理关系。文件 source 的 Catalog 文件和文本/图片 vector rows
会保留，用于后续显式重加时复用；source row 标记为 `removed` 并从列表、
详情、检索范围中隐藏。表 source 会删除 backend 为知识库 clone 出来的
KB table；原始 Catalog 表不属于知识库 owner，不删除。

前端若使用“删除数据源”文案，只表示删除当前知识库里的 source 关系。
该文案不能解释为删除原始 Catalog 文件、原始 Catalog 表或文件向量 rows。

删除整个知识库时，backend 会清理该 `model_id` 下的 segment versions、
segments、recall stats、source/job 关系、raw volume 绑定和 data domain
记录。Catalog 资源仍通过 data domain owner 接口删除，不能由 worker 或
调用方直接写底层存储表。

## 历史 Backfill 边界

`backfill-legacy` 只用于把旧 workflow/lineage 数据接回
`knowledge_base_sources`。
接回后的 source 关系就是知识库问答的授权范围；如果旧 workflow 已有对应
文档索引，问答检索应使用该关联和现有索引返回结果，不要求 source 已切换为
`succeeded` 或已写入当前 segment 指针。

它必须：

- 是显式 `POST`；
- 幂等；
- 单次只处理有限批次，调用方再重新读取 `/sources` 判断是否继续；
- 保留 `enabled`、`expires_at`、tags、force flag、当前 segment 指针和
  当前 index 指针等治理字段；
- Catalog 资产缺失时跳过，不创建虚假资源行。

它不得：

- 导入 chunks；
- 迁移 vector rows；
- 发布 segment version；
- 成为未来 workflow 接入知识库的主路径。

## LLM 检索生效规则

新版 LLM 对话路径从 catalog runtime 的 platform knowledge scope 读取 KB
治理结果。文件 source 只有同时满足以下条件才进入 RAG/visual 检索范围：

- `effective_enabled=true`，即 `enabled && (!expired ||
  force_enabled_after_expiry)`；
- source 已有当前 `segment_version_id`；
- source 已有正数 `index_version`。

runtime 会把 `CurrentIndexVersionByFileID` 传给
`search_rag_chunks/search_visual_image`，检索 SQL 按 `file_id +
index_version` 过滤。禁用、过期且未强制启用、或尚未导入当前 segment
version 的文档不会进入新版 LLM 检索。旧 explore 对话路径不在本文档治理
范围内。

## 维护 Checklist

新增 KB workflow template 时：

1. 确认 backend 在触发 workflow 前已经创建明确 source/job 绑定。
2. template 只包含 parse、index、parsed docset、lineage/register 等数据产物节点。
3. 不添加 worker -> backend 回调节点。
4. 不传内容 payload 字段给治理路径。
5. 补 workflow template 测试，确认不包含治理 finalizer 节点。

调整 reconcile 行为时：

1. 更新 `moi-backend/pkg/session/semantic_model_kb_jobs.go` 及对应 session 测试。
2. 如果 HTTP contract 变化，同步 backend handler 文档和 FE API overview。
3. 如果路由或权限变化，同步 RBAC route matrix。
4. 保持 `source-jobs/reconcile` request body skip；不要记录无意义空 body。

必须保留的回归覆盖：

- pending `rag_ingest` job 能被 reconcile 捞起并发布新版本；
- 重复的当前 `index_version` 不创建重复 chunk version；
- 更新的 `index_version` 发布新版本；
- workflow failed 只标记 source/job failed，不改变 enabled/current pointer；
- 重复追加已有 Catalog file/table 时复用 source，不覆盖治理字段；
- 删除 source/model 会清理 segment versions、segments 和 recall stats；
- runtime scope 只暴露 effective enabled 且有 current segment/index version
  的文件，并把 current index version 传给 RAG/visual 检索；
- `GET /sources` 和 `GET /source-jobs` 保持只读。

## 主要代码入口

- Backend route 和 handler：
  `moi-backend/pkg/handlers/session/semantic_model.go`
- Create/append 编排：
  `moi-backend/pkg/session/semantic_model_kb_create_append.go`
- Catalog 选择展开：
  `moi-backend/pkg/session/semantic_model_kb_selection.go`
- KB data domain 与 workflow 部署：
  `moi-backend/pkg/session/semantic_model_kb_domain.go`
- Source/job intent：
  `moi-backend/pkg/session/semantic_model_kb_source_intents.go`
- Job 投影、reconcile、import task 观察和 RAG 发布入口：
  `moi-backend/pkg/session/semantic_model_kb_jobs.go`
- Segment import/commit：
  `moi-backend/pkg/session/semantic_model_segments.go`
- Catalog、dataconn、workflow adapter：
  `moi-backend/pkg/session/semantic_model_catalog_adapters.go`
- 内置 KB workflow templates：
  `moi-backend/pkg/workflowtemplate/seed.go`
- Backend workflow deployment/client：
  `moi-backend/pkg/workflowv2/deployment.go`、
  `moi-backend/pkg/workflowv2/service.go`
- 结构化 import task 创建和 system workflow 派发：
  `moi-backend/pkg/dataconn/connector_impl.go`、
  `moi-backend/pkg/dataconn/upload.go`、
  `moi-backend/pkg/dataconn/task_impl.go`
- Backend import WorkItem 和 completion hook：
  `moi-backend/pkg/worker/handler.go`、
  `moi-backend/pkg/worker/worker.go`、
  `moi-backend/cmd/main/main.go`
- Catalog table clone owner：
  `moi-backend/pkg/catalog/datacenter_service.go`
- Core volume file/trigger delivery：
  `moi-core/catalog/pkg/api/handlers/volume_file.go`、
  `moi-core/catalog/pkg/triggerdelivery/service.go`
- Core workflow execution 投影：
  `moi-core/catalog/pkg/workflowapp/service.go`
- Mowl workflow runtime：
  `moi-core/mowl/pkg/engine/`
- RAG 与 `s3_to_mo` WorkItems：
  `moi-core/workers/go-worker/pkg/worker/workitems.go`、
  `moi-core/workers/go-worker/pkg/workitems/`
- 前端共享选择器和 Catalog 选择器：
  `moi-frontend/modules/shared-moi-components/src/knowledge-source-select-modal/`、
  `moi-frontend/modules/shared-moi-components/src/catalog-data-selector/`
- 数据侧资源页和智能体侧资源面板：
  `moi-frontend/modules/moi-knowledge/src/pages/knowledge-edit/KnowledgeAdvancedConfigPage.tsx`、
  `moi-frontend/modules/moi-agent/src/pages/agent-chat/components/KBDetailPanel.tsx`

工作流、Task、Case、WorkItem 和 Worker 的通用概念见
`moi-core/docs/guide/CONCEPTS.md`；Catalog 与 Mowl 内部职责见
`moi-core/docs/architecture/COMPONENT_DESIGN.md`。本文只定义这些模块在知识库
资源添加链路中的使用边界。
