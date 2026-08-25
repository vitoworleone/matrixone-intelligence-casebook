# 使用 SDK 开发应用 — 完整指南

本文档面向**开发人员**：如何用 **Go SDK** 与 **Python SDK** 从零开发自己的应用。按 **Use Case** 组织，每个场景分别给出 Go 与 Python 的用法。
**概念先行**：工作流、任务、WorkItem、Worker、动态服务等概念请先阅读 [核心概念](./CONCEPTS.md)；本文只讲在代码中如何创建客户端、调用 API、定义工作流与运行任务。

**文档中的代码**：以下示例多为片段（可能省略部分 import 或变量定义）；完整可运行示例见 [go-sdk/examples](../go-sdk/examples) 与 [python-sdk](../python-sdk) 的示例目录。

---

## 目录

- [1. 前置条件](#1-前置条件)
- [1.1 快速体验：5 分钟跑通一次工作流](#11-快速体验5-分钟跑通一次工作流)
- [2. Use Case 1：创建客户端](#2-use-case-1创建客户端)
- [3. Use Case 2：工作区与元数据](#3-use-case-2工作区与元数据不涉及工作流)
- [4. Use Case 3：工作流 + 任务 + Worker](#4-use-case-3工作流--任务--worker可执行流程图)
- [5. Use Case 4：动态服务](#5-use-case-4动态服务按名称--输入即调即用)
- [6. Use Case 5：用户与 API Key](#6-use-case-5用户与-api-key)
- [7. Use Case 6：回调机制](#7-use-case-6回调机制)
- [7.5 Use Case 6.5：CDH 元数据管理](#75-use-case-65cdh-元数据管理)
- [8. 错误处理](#8-错误处理)
- [9. API 能力索引](#9-api-能力索引go--python-对照)
- [10. DSL 定义工作流](#10-dsl-定义工作流代码方式与-yaml-方式)
- [11. 获取当前用户可用的 WorkItem](#11-获取当前用户可用的-workitem)
- [12. 自定义 Worker](#12-自定义-worker)
- [12.4 Runtime 临时 Worker（按需拉起与回收）](#124-runtime-临时-worker按需拉起与回收)
- [13. Use Case 7：工作流结果适配（Parse Results）](#13-use-case-7工作流结果适配parse-results)
- [14. Use Case 8：知识库与 NL2SQL 语义管理](#14-use-case-8知识库与-nl2sql-语义管理)
- [15. Use Case 9：LLM 代理与 Embedding 代理](#15-use-case-9llm-代理与-embedding-代理)
- [16. 文档与示例汇总](#16-文档与示例汇总)

---

## 1. 前置条件

- 已部署或本地启动 **Catalog 服务**（含内嵌 Mowl 或独立 Mowl），并知道 **HTTP 基地址**（如 `http://localhost:8081`）。若尚未部署，请先参考 [部署说明](./DEPLOYMENT.md) 进行本地或 K8s 部署。
- 已有一个有效的 **API Key**。获取方式：**本地开发**可使用 `core-cli apikey generate`（见 [cli/README.md](../cli/README.md)）；或通过登录/注册接口获取；**系统/管理员**可为指定用户创建 Key。认证与注册接口见 [catalog-api.md](../api/catalog-api.md)。
- **Go**：项目里已引入 `go get github.com/matrixflow/moi-core/go-sdk`。
- **Python**：已安装 `pip install -e .`；若使用 Worker（gRPC）需 `pip install -e ".[grpc]"`，且从项目根执行 `make proto-python` 生成 gRPC 代码。

### 1.1 快速体验：5 分钟跑通一次工作流

若你希望先**整体跑通**再按章节细看，可按以下步骤做一次最小闭环：

1. **启动服务**：按 [DEPLOYMENT.md](./DEPLOYMENT.md) 用 Docker Compose 或本地二进制启动 Catalog（含 Mowl），记下 endpoint（如 `http://localhost:8081`）。
2. **获取 API Key**：使用 `core-cli apikey generate` 或登录/注册接口得到 API Key。
3. **创建客户端**：用 §2 的代码创建 Client（endpoint + apiKey）。
4. **创建工作区与工作流**：用 §3 创建 workspace；用 §4.1 创建一条工作流定义、用 DSL 建一个仅含 1 个 WorkItem 的版本并 **Publish**。
5. **执行一次**：方式一 — 启动一个 Worker（§4.3），注册该 WorkItem 并 `Connect`，再用 §4.2 在 Worker 里 `ExecuteByWorkflowVersion` 或 `ExecuteByWorkflowName` 创建任务，观察任务执行完成。方式二 — 若图中只用内建节点（如 `catalog:http.request`），可直接用 HTTP 或 `client.Tasks().Create` 创建任务，无需 Worker。

完成后即可按需跳转到 §3（元数据/文件）、§5（动态服务）、§7（回调）等章节深入。

---

## 2. Use Case 1：创建客户端

所有 API 都通过 **Client** 访问；Client 需要 **endpoint** 和 **apiKey**。

### Go

```go
package main

import (
    "context"
    "log"
    "time"

    moi "github.com/matrixflow/moi-core/go-sdk"
)

func main() {
    client, err := moi.New(
        "http://localhost:8081",
        "your-api-key",
        moi.WithTimeout(60*time.Second),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    ctx := context.Background()
    // 后续示例均假设已有 client 和 ctx
}
```

**常用可选参数**：`WithTimeout`（默认 60s）、`WithHTTPClient`、`WithLogger`。SDK **不**提供 HTTP 重试；需要重试时由业务层在确认幂等后自行实现。若要做 **Worker**（执行工作项或创建任务），需 **WithWorkerID**：

```go
client, err := moi.New("http://localhost:8081", "your-api-key", moi.WithWorkerID("my-worker"))
worker := client.Worker("workspace-id")
// worker.RegisterWorkItem(...) 再 worker.Connect(ctx)
```

### Python

```python
import moi

client = moi.new(
    "http://localhost:8081",
    "your-api-key",
    timeout=60.0,
)
# 使用完毕可 client.close()

# 若要做 Worker，传入 worker_id
client = moi.new("http://localhost:8081", "your-api-key", worker_id="my-worker")
worker = client.worker("workspace-id")
# worker.register_work_item(...) 再 worker.connect()
```

---

## 3. Use Case 2：工作区与元数据（不涉及工作流）

典型步骤：创建/获取工作区 → 创建目录 → 同步数据库元数据 → 创建卷 → 上传/下载文件。**数据库在 MatrixOne 侧创建**，通过 `SyncMetadata` / `sync_metadata` 将元数据同步到 moi-core（SDK 无 `Create` 方法）。**Volume 支持层级关系**：可在某 Volume 下创建子 Volume（parent），形成树形结构；支持获取子卷列表与从根到某卷的路径，见 [§3.4 Volume 层级关系](#34-volume-层级关系)。

**ID 与类型约定**：便于与 API 一致，约定如下。**Go**：`workspace_id`、`task_id` 等为 `string`；`catalog_id`、`database_id`、`volume_id` 等为 `int64`（如 `GetId()` 返回）；`file_id` 为 `string`。**Python**：对应为 `str` 与 `int`；`file_id` 为 `str`。返回对象中的 `parent_id`（子卷的父卷 ID）类型与 `volume_id` 一致。具体以各 SDK 方法签名为准。

### 3.1 基本链路（Go）

```go
// 创建或获取工作区
ws, err := client.Workspaces().Create(ctx, "my-workspace", moi.WithWorkspaceDescription("描述"))
if err != nil { log.Fatal(err) }
workspaceID := ws.GetId()

// 创建目录
catalog, _ := client.Catalogs().Create(ctx, workspaceID, "main-catalog", moi.WithComment("主目录"))

// 同步数据库元数据（数据库在 MatrixOne 侧创建，通过 SyncMetadata 同步到 moi-core）
syncResp, _ := client.Databases().SyncMetadata(ctx, workspaceID, "analytics", catalog.GetId())
dbID := syncResp.GetDatabase().GetId()

// 创建卷
volume, _ := client.Volumes().Create(ctx, workspaceID, dbID, "raw-data", moi.WithVolumeComment("原始数据"))

// 上传文件到卷
r := strings.NewReader(`{"event": "test"}`)
file, err := client.Volumes().Upload(ctx, workspaceID, volume.GetId(), "event.json", r)
if err != nil { log.Fatal(err) }
log.Printf("Uploaded file ID: %s", file.GetFileId())

// 列出卷内文件（分页）、下载文件
files, _ := client.Volumes().ListFiles(ctx, workspaceID, volume.GetId(), moi.WithFilesPageSize(20))
for _, f := range files { _ = f.GetFileId(); _ = f.GetName() }
rc, err := client.Volumes().Download(ctx, workspaceID, file.GetFileId())
if err == nil { defer rc.Close(); /* io.Copy(...) */ }
```

### 3.2 基本链路（Python）

```python
# 创建或获取工作区
ws = client.workspaces().create("my-workspace", description="描述")
workspace_id = ws["id"]

# 创建目录
catalog = client.catalogs().create(workspace_id, "main-catalog", comment="主目录")

# 同步数据库元数据（数据库在 MatrixOne 侧创建，通过 sync_metadata 同步到 moi-core）
sync_resp = client.databases().sync_metadata(workspace_id, "analytics", catalog["id"])
db_id = sync_resp["database"]["id"]

# 创建卷
volume = client.volumes().create(workspace_id, db_id, "raw-data", comment="原始数据")

# 上传文件到卷
import io
data = io.BytesIO(b'{"event": "test"}')
file_info = client.volumes().upload(workspace_id, volume["id"], "event.json", data)
print("Uploaded file ID:", file_info.get("file_id"))

# 列出卷内文件（分页）、下载文件
list_resp = client.volumes().list_files(workspace_id, volume["id"], page_size=20)
for f in list_resp.get("items", []): pass  # f["file_id"], f["name"]
content = client.volumes().download(workspace_id, file_info["file_id"])
```

### 3.3 Catalog 资源名称契约

Catalog 资源的新建和重命名使用同一套名称规则。规则作用于调用方提交的原始字符串；服务不会自动去除空格、转换大小写或替换字符。

- 长度为 1–255 个 Unicode 字符。
- 首字符只能是小写英文字母（`a-z`）、中文汉字、数字（`0-9`）或下划线（`_`）。
- 后续字符还可以使用连字符（`-`）和点号（`.`）。
- 大写英文字母、空白、反引号、控制字符及其他字符不受支持。

例如，`sales_2026`、`客户数据`、`raw-data.v1` 合法；`Sales`、` raw-data`、`raw data`、`-raw` 不合法。

该规则适用于 Catalog 创建和重命名、Volume 创建和重命名（包括通过 `parent_id` 创建的子 Volume），以及带用户创建意图的 Database 和 Table 同步。Database 没有独立的 Catalog 重命名 API。普通后台元数据发现、读取、列表及仅修改备注不会重新校验或改写历史名称。

不合法名称返回 HTTP `400 Bad Request` 和 `INVALID_ARGUMENT`；错误详情包含 `field=name` 和 `reason=CATALOG_IDENTIFIER_INVALID`。服务拒绝原始输入，不会静默修复。历史数据不迁移、不重写，只有创建新资源或显式提交新名称时才要求满足当前契约。

### 3.4 Volume 层级关系

Volume 支持**父子层级**：同一 Database 下可创建「无 parent」的根 Volume，也可在某个 Volume 下创建子 Volume（指定 `parent_id`），形成树形结构。可用接口：

- **创建子 Volume**：创建时传入父 Volume ID。
- **GetChildren / get_children**：获取某 Volume 下的直接子 Volume 列表（分页）。
- **GetPath / get_path**：获取从根 Volume 到当前 Volume 的路径（用于展示面包屑或完整路径）。

#### Go

```go
// 先创建父 Volume（不传 parent 即为根）
parent, _ := client.Volumes().Create(ctx, workspaceID, dbID, "parent-folder", moi.WithVolumeComment("父目录"))

// 在父 Volume 下创建子 Volume
child, _ := client.Volumes().Create(ctx, workspaceID, dbID, "child-folder",
    moi.WithVolumeComment("子目录"),
    moi.WithParentVolume(parent.GetId()),
)

// 获取某 Volume 的直接子 Volume 列表
childrenResp, _ := client.Volumes().GetChildren(ctx, workspaceID, parent.GetId(), moi.WithPageSize(20))
for _, v := range childrenResp.GetItems() {
    log.Printf("child: %s (parent_id=%d)", v.GetName(), v.GetParentId())
}

// 获取从根到某 Volume 的路径（如 root -> level1 -> level2）
pathResp, _ := client.Volumes().GetPath(ctx, workspaceID, child.GetId())
for _, v := range pathResp.GetItems() {
    log.Printf("path segment: %s", v.GetName())
}
```

#### Python

```python
# 先创建父 Volume（不传 parent_id 即为根）
parent = client.volumes().create(workspace_id, db_id, "parent-folder", comment="父目录")

# 在父 Volume 下创建子 Volume
child = client.volumes().create(
    workspace_id, db_id, "child-folder",
    comment="子目录",
    parent_id=parent["id"],
)

# 获取某 Volume 的直接子 Volume 列表
children_resp = client.volumes().get_children(workspace_id, parent["id"], page_size=20)
for v in children_resp.get("items", []):
    print("child:", v["name"], "parent_id=", v.get("parent_id"))

# 获取从根到某 Volume 的路径
path_resp = client.volumes().get_path(workspace_id, child["id"])
for v in path_resp.get("items", []):
    print("path segment:", v["name"])
```

更多：列出、分页、更新、删除见 [go-sdk-examples.md](../examples/go-sdk-examples.md) 与 [python-sdk/README.md](../python-sdk/README.md)。

---

## 4. Use Case 3：工作流 + 任务 + Worker（可执行流程图）

需要三条线：**定义图**（工作流 + 版本）、**触发执行**（任务）、**执行节点**（Worker 注册 WorkItem）。
用代码构建图的语法（Chain / Parallel / Xor / Or / Loop、WorkItem 等）见 [DSL.md](./DSL.md)。

### 4.1 定义工作流与版本

#### Go

```go
import "github.com/matrixflow/moi-core/go-sdk/dsl"

workspaceID := "your-workspace-id"

def, err := client.Workflows(workspaceID).Create(ctx, "my-pipeline", moi.WithWorkflowDefDescription("数据管道"))
if err != nil { log.Fatal(err) }

builder := dsl.Workflow("my-net", "root").Chain(
    dsl.WorkItem("step1", "my-workitem"),
    dsl.WorkItem("step2", "another-item"),
)
version, err := client.WorkflowVersions(workspaceID).CreateByBuilder(ctx, def.GetId(), builder)
if err != nil { log.Fatal(err) }

err = client.WorkflowVersions(workspaceID).Publish(ctx, version.GetId())
if err != nil { log.Fatal(err) }
// 只有发布后，按名称执行（ExecuteByWorkflowName）或动态服务才会使用该版本；按 version_id 执行则 draft 也可。
```

#### Python

```python
from moi.dsl import Workflow, WorkItem, workflow_to_dict

workspace_id = "your-workspace-id"

wf_def = client.workflows(workspace_id).create("my-pipeline", description="数据管道")
workflow_dict = workflow_to_dict(
    Workflow("my-net", "root").Chain(
        WorkItem("step1", "my-workitem"),
        WorkItem("step2", "another-item"),
    )
)
version = client.workflow_versions(workspace_id).create(
    wf_def["id"], workflow_dict, description="v1"
)
client.workflow_versions(workspace_id).publish(version["id"])
# 只有发布后，按名称执行或动态服务才会使用该版本；按 version_id 执行则 draft 也可。
```

### 4.2 创建任务（触发一次执行）

每个 `client.Worker(workspaceID)` 对应**一个工作区**的连接；同一 Client 可对多个工作区分别创建多个 Worker 并各自 `Connect`，或只连接一个工作区。

#### Go

**方式 A：仅 HTTP 创建任务（不跑 Worker 时）**

```go
task, err := client.Tasks(workspaceID).Create(ctx, "run-001",
    moi.WithTaskWorkflowVersionID(version.GetId()),
    moi.WithTaskData(`{"input": "value"}`),
)
```

**方式 B：在 Worker 进程里创建任务（推荐）**

```go
worker := client.Worker(workspaceID)
if err := worker.Connect(ctx); err != nil { log.Fatal(err) }
defer worker.Disconnect()

task, err := worker.ExecuteByWorkflowVersion("run-001", version.GetId(), moi.WithData(`{}`))
// 或 task, err := worker.ExecuteByWorkflowName("run-001", "my-pipeline", moi.WithData(`{}`))
```

#### Python

**方式 A：仅 HTTP 创建任务**

```python
task = client.tasks(workspace_id).create(
    "run-001",
    workflow_version_id=version["id"],
    data='{"input": "value"}',
)
```

**方式 B：在 Worker 里创建任务**

```python
worker = client.worker(workspace_id)
worker.connect()
try:
    task = worker.execute_by_workflow_version("run-001", version["id"], moi.with_data("{}"))
    # 或 task = worker.execute_by_workflow_name("run-001", "my-pipeline", moi.with_data("{}"))
finally:
    worker.disconnect()
```

### 4.3 实现并注册 WorkItem（Worker 侧）

图中若有 WorkItem 节点，必须有**已连接并注册了对应名字的 WorkItem** 的 Worker，否则该节点会一直等待。
**Handler 签名**：Go 为 `func(ctx, wctx WorkItemContext, msg *mowl.MowlMessage) (*mowl.MowlMessage, error)`，输入输出从 `msg` 的 Data/Vars 与返回的 MowlMessage 传递；Python 类似，见 §12。完整选项（Schema、流式等）见 [go-sdk-api.md](../api/go-sdk-api.md) 与 [python-sdk/README.md](../python-sdk/README.md)。

#### Go

```go
worker := client.Worker(workspaceID)

err := worker.RegisterWorkItem("my-workitem", &mowl.WorkItemMetadata{
    IsolationLevel: mowl.IsolationLevel_PRIVATE,
}, func(ctx context.Context, wctx moi.WorkItemContext, msg *mowl.MowlMessage) (*mowl.MowlMessage, error) {
    data, vars := msg.GetData(), msg.GetVars()
    _ = data; _ = vars
    return &mowl.MowlMessage{Status: "COMPLETED", Data: `{"status": "ok"}`, Vars: msg.GetVars()}, nil
})
if err != nil { log.Fatal(err) }

if err := worker.Connect(ctx); err != nil { log.Fatal(err) }
defer worker.Disconnect()

select {}
```

#### Python

```python
import moi
from moi import enums
from moi._proto.mowl import mowl_pb2

def handle_my_workitem(ctx, wctx, msg):
    # msg 为 MowlMessage，含 Data、Vars；返回 MowlMessage
    return mowl_pb2.MowlMessage(Status=moi.Status.COMPLETED, Data='{"status": "ok"}', Vars=msg.Vars or "")

worker = client.worker(workspace_id)
metadata = mowl_pb2.WorkItemMetadata(isolation_level=enums.ISOLATION_LEVEL_PRIVATE)
worker.register_work_item("my-workitem", metadata, handle_my_workitem)
worker.connect()
# 保持运行以接收调度
```

更多：Stream WorkItem、回调、通知见 [go-sdk-api.md](../api/go-sdk-api.md)、[go-sdk-examples.md](../examples/go-sdk-examples.md) 与 [python-sdk/README.md](../python-sdk/README.md)。

---

## 5. Use Case 4：动态服务（按名称 + 输入即调即用）

动态服务 = 已发布的工作流版本 + 声明了 input/output Schema。调用方只需**工作流名称 + 输入 JSON**，无需先 CreateTask。

### 5.1 创建并发布为动态服务

#### Go

```go
def, _ := client.Workflows(workspaceID).Create(ctx, "web-scraper", moi.WithWorkflowDefDescription("网页抓取服务"))

inputSchema := moi.NewSchema().
    Property("url", moi.StringSchema().Description("URL")).
    Required("url")
outputSchema := moi.NewSchema().
    Property("content", moi.StringSchema()).
    Property("status_code", moi.IntegerSchema())

builder := dsl.Workflow("scraper", "root").Chain(dsl.WorkItem("fetch", "http-fetcher"))
version, err := client.WorkflowVersions(workspaceID).CreateByBuilder(ctx, def.GetId(), builder,
    moi.WithVersionDynamicService(inputSchema, outputSchema),
    moi.WithVersionResultMode(mowl.ResultMode_RESULT_MODE_ONESHOT),
)
if err != nil { log.Fatal(err) }

client.WorkflowVersions(workspaceID).Publish(ctx, version.GetId())
```

#### Python

```python
from moi.schema_builder import new_schema, string_schema, integer_schema
from moi.dsl import Workflow, WorkItem, workflow_to_dict
from moi import enums

wf_def = client.workflows(workspace_id).create("web-scraper", description="网页抓取服务")

input_schema = new_schema().property("url", string_schema().description("URL")).required("url")
output_schema = new_schema().property("content", string_schema()).property("status_code", integer_schema())

workflow_dict = workflow_to_dict(Workflow("scraper", "root").Chain(WorkItem("fetch", "http-fetcher")))
version = client.workflow_versions(workspace_id).create(
    wf_def["id"], workflow_dict,
    workflow_type=enums.WORKFLOW_TYPE_DYNAMIC_SERVICE,
    input_schema=input_schema.build(),
    output_schema=output_schema.build(),
    result_mode=enums.RESULT_MODE_ONESHOT,
)
client.workflow_versions(workspace_id).publish(version["id"])
```

### 5.2 调用动态服务

#### Go

```go
worker := client.Worker(workspaceID)
// 若图中含 WorkItem，仍需 worker.Connect() 并注册 "http-fetcher"

result, err := worker.InvokeDynamicServiceSync(ctx, "web-scraper", `{"url": "https://example.com"}`)
if err != nil { log.Fatal(err) }
fmt.Printf("Result: %s\n", result.Result)
```

#### Python

```python
worker = client.worker(workspace_id)
# 若图中含 WorkItem，仍需 worker.connect() 并注册 "http-fetcher"
worker.connect()
result = worker.invoke_dynamic_service_sync("web-scraper", '{"url": "https://example.com"}')
print("Result:", result.Result)
```

**HTTP**：`POST /api/v1/workspaces/:id/dynamic-services/invoke`，Body: `{"service_name": "...", "type": "operator", "payload": {"url": "..."}}`，见 [catalog-api.md](../api/catalog-api.md)。

---

## 6. Use Case 5：用户与 API Key

- **当前用户**：为自己创建 API Key、列出/删除。
- **系统用户**：使用 System API Key 的客户端可为**指定用户**创建 API Key（如运维为业务用户建 Key）。

### Go

```go
// 当前用户创建 Key
created, err := client.APIKeys().Create(ctx, "my-key", moi.WithScopes("*"), moi.WithExpiresInDays(30))
if err != nil { log.Fatal(err) }
// 创建时返回的 Key 明文仅此一次，务必保存
fmt.Println("New API Key:", created.Key)

// 系统用户为指定用户创建 Key
systemClient, _ := moi.New(endpoint, systemAPIKey)
created, err = systemClient.APIKeys().Create(ctx, "user-key", moi.WithUserID("user-123"))
```

### Python

```python
# 当前用户创建 Key
created = client.apikeys().create("my-key", scopes=["*"], expires_in_days=30)
# 创建时返回的 key 明文仅此一次，务必保存
print("New API Key:", created.get("key"))

# 系统用户为指定用户创建 Key
system_client = moi.new(endpoint, system_api_key)
created = system_client.apikeys().create("user-key", user_id="user-123")
```

---

## 7. Use Case 6：回调机制

回调有两种常见用法：**任务/工作流完成时通知**（创建任务时配置 NotificationConfig，引擎在状态变化时主动推送），以及**工作流内的回调节点**（图中某一步为 Callback 节点，执行到该节点时引擎向 HTTP 或 Worker 发起一次调用）。

### 7.1 任务完成时 HTTP 回调

创建任务时指定 **HTTP 通知**：工作流完成、失败或取消时，引擎向指定 URL 发送 HTTP 请求（如 POST），便于与外部系统或 Webhook 集成。

#### Go

```go
// 创建任务时附带 HTTP 通知配置
notification := moi.NewHTTPNotification("https://your-app.com/webhook/workflow-done").
    WithMethod("POST").
    WithTimeout(30).
    WithHeaders(map[string]string{"X-Custom": "value"}).
    Build()

task, err := client.Tasks(workspaceID).Create(ctx, "task-with-callback",
    moi.WithTaskWorkflowVersionID(versionID),
    moi.WithTaskData(`{"input": "data"}`),
    moi.WithTaskNotification(notification),
)
```

服务端在工作流状态变化时会向该 URL 发送请求，Body 为 JSON，通常包含 `task_id`、`case_id`、`status`（如 `COMPLETED`/`FAILED`）、`timestamp` 等字段；具体字段以引擎或 Catalog 的约定为准，可参考 Catalog 或 Mowl 的 Notification 相关定义。

#### Python

```python
from moi.task import new_http_notification

notification = (
    new_http_notification("https://your-app.com/webhook/workflow-done")
    .with_method("POST")
    .with_timeout(30)
    .build()
)
# 若用 dict 形式：notification = {"type": "http", "url": "...", "method": "POST"}

task = client.tasks(workspace_id).create(
    "task-with-callback",
    workflow_version_id=version_id,
    data='{"input": "data"}',
    notification=notification,
)
```

### 7.2 任务完成时 Worker 回调与「等待完成」

创建任务时指定 **Worker 通知**（`worker_id` 为当前 Worker 的 ID）：引擎在工作流完成/失败等状态变化时，向该 Worker 发送回调消息。Worker 端可**注册处理函数**处理通知，或**阻塞等待**某次任务的通知，实现「发起任务后等完成再继续」的流程。

#### Go

```go
client, _ := moi.New(endpoint, apiKey, moi.WithWorkerID("my-worker-id"))
worker := client.Worker(workspaceID)

// 可选：注册工作流通知处理函数（不等待时也可仅用 handler 做业务逻辑）
worker.AddWorkflowNotifyHandler("my-handler", func(ctx context.Context, n *mowl.WorkflowNotification) {
    log.Printf("workflow notification: task=%s case=%s status=%s", n.TaskId, n.CaseId, n.Status)
}, moi.WithNotifyStates(mowl.StatusCompleted, mowl.StatusFailed))

if err := worker.Connect(ctx); err != nil { log.Fatal(err) }
defer worker.Disconnect()

// 创建任务时指定「通知到本 Worker」
notification := moi.NewWorkerNotification("my-worker-id").
    WithMessage("workflow_completed").
    WithNotifyWorkflowStates("COMPLETED", "FAILED").
    Build()

task, err := worker.ExecuteByWorkflowVersion("run-001", versionID,
    moi.WithData(`{}`),
    moi.WithNotificationConfig(notification),
)
if err != nil { log.Fatal(err) }

// 阻塞直到收到该任务的工作流完成/失败通知
n, err := worker.WaitForWorkflowNotification(ctx,
    moi.WithTaskID(task.GetId()),
    moi.WithNotifyStates(mowl.StatusCompleted, mowl.StatusFailed),
)
if err != nil { log.Fatal(err) }
log.Printf("task finished: status=%s", n.Status)
```

#### Python

```python
worker = client.worker(workspace_id)
worker_id = "my-worker-id"  # 与创建 Client 时 worker_id 一致

# 可选：注册工作流通知处理函数
def on_workflow_notification(_ctx, notification):
    print("workflow notification:", notification.task_id, notification.status)

worker.add_workflow_notify_handler("my-handler", on_workflow_notification, moi.with_notify_states(moi.Status.COMPLETED, moi.Status.FAILED))
worker.connect()

# 创建任务时指定通知到本 Worker
from moi.task import new_worker_notification
notification = new_worker_notification(worker_id).with_message("workflow_completed").build()

task = worker.execute_by_workflow_version(
    "run-001", version_id,
    moi.with_data("{}"),
    moi.with_notification_config(notification),
)

# 阻塞直到收到该任务的工作流完成/失败通知
n = worker.wait_for_workflow_notification(
    None,
    moi.with_task_id(task.id),
    moi.with_notify_states(moi.Status.COMPLETED, moi.Status.FAILED),
)
print("task finished:", n.status if n else "timeout/cancelled")
```

### 7.3 工作流内的回调节点（Callback 节点）

在**工作流图**中插入 **Callback** 节点：执行到该节点时，引擎会向配置的 **HTTP URL** 或 **Worker** 发起一次回调（HTTP 请求或 Worker 消息），常用于「某一步完成后通知外部」或「人工审批后由回调继续」。

#### Go

```go
import "github.com/matrixflow/moi-core/go-sdk/dsl"

// HTTP 回调：执行到该节点时 POST 到 URL
builder := dsl.Workflow("demo", "root").Chain(
    dsl.WorkItem("step1", "my-workitem"),
    dsl.Callback("notify", dsl.CallbackConfig{
        URL:     "https://your-app.com/webhook/step-done",
        Method:  "POST",
        Timeout: 10,
    }),
    dsl.WorkItem("step2", "my-workitem"),
)

// Worker 回调：向指定 Worker 发消息，由该 Worker 的 RegisterCallbackHandler 处理
builder := dsl.Workflow("demo", "root").Chain(
    dsl.WorkItem("step1", "my-workitem"),
    dsl.Callback("notify-worker", dsl.CallbackConfig{
        WorkerID: "worker-123",
        Message:  "step1_done",
    }),
    dsl.WorkItem("step2", "my-workitem"),
)
```

#### Python

```python
from moi.dsl import Workflow, WorkItem, Callback, workflow_to_dict

# HTTP 回调
builder = (
    Workflow("demo", "root")
    .Chain(
        WorkItem("step1", "my-workitem"),
        Callback("notify", url="https://your-app.com/webhook/step-done", method="POST", timeout=10),
        WorkItem("step2", "my-workitem"),
    )
)

# Worker 回调（url 留空时仅发 Worker 消息）
builder = (
    Workflow("demo", "root")
    .Chain(
        WorkItem("step1", "my-workitem"),
        Callback("notify-worker", "", worker_id="worker-123", message="step1_done"),
        WorkItem("step2", "my-workitem"),
    )
)
workflow_dict = workflow_to_dict(builder)
```

#### YAML 示例（回调节点）

```yaml
workflow:
  name: "demo"
  root: "root"
root:
  chain:
    - work_item: { name: "step1", id: "my-workitem" }
    - callback:
        name: "notify"
        url: "https://your-app.com/webhook/step-done"
        method: "POST"
        timeout: 10
    - work_item: { name: "step2", id: "my-workitem" }
```

**小结**：任务级通知（§7.1、§7.2）在**创建任务**时配置，由引擎在**工作流/任务状态变化**时主动推送；Callback 节点（§7.3）是**图中的一步**，执行到该节点时触发一次 HTTP 或 Worker 回调。更多配置（NotifyNodeStates、NotifyNodeIDs 等）见 [go-sdk-api.md](../api/go-sdk-api.md) 的 NotificationBuilder 与 [DSL.md](./DSL.md) 的 Callback 节点。

---

## 7.5 Use Case 6.5：CDH 元数据管理

CDH（Cloudera Distribution for Hadoop）元数据管理允许在 Workspace 内添加多个 CDH 集群配置，从 Hive Metastore 同步数据库、表、列级别的元数据。每个配置有独立的 `config_id`，所有元数据按 `config_id` 隔离。

SDK 通过 `client.CDH(workspaceID)` 获取 CDH 服务实例，支持配置 CRUD、元数据同步、查询和健康检查。

### Go

```go
workspaceID := "ws-123"
cdh := client.CDH(workspaceID)

// 创建 CDH 配置
config, err := cdh.CreateConfig(ctx, "production-cdh", "hive-metastore.example.com", 9083, "6.3.2",
    moi.WithCDHConnectTimeout(30),
)
if err != nil { log.Fatal(err) }
configID := config.GetId()

// 健康检查
health, err := cdh.HealthCheck(ctx, configID)
if err != nil { log.Fatal(err) }
log.Printf("CDH status: %s", health.GetStatus())

// 同步元数据（创建周期性同步工作流，从 Hive Metastore 定期采集指定数据库的库/表/列信息）
syncResp, err := cdh.SyncMetadata(ctx, configID, "default", "0 */6 * * *")
if err != nil { log.Fatal(err) }
log.Printf("Synced: %d tables, %d updated, %d deleted",
    syncResp.GetTablesSynced(), syncResp.GetTablesUpdated(), syncResp.GetTablesDeleted())

// 停止周期性同步
err = cdh.StopSync(ctx, configID)
if err != nil { log.Fatal(err) }

// 列出已同步的数据库
dbs, err := cdh.ListDatabases(ctx, configID)
if err != nil { log.Fatal(err) }
for _, db := range dbs.GetItems() {
    log.Printf("Database: %s (id=%d)", db.GetName(), db.GetId())
}

// 列出表
tables, err := cdh.ListTables(ctx, configID, dbs.GetItems()[0].GetId())
if err != nil { log.Fatal(err) }
for _, t := range tables.GetItems() {
    log.Printf("Table: %s", t.GetName())
}

// 获取表详情（含列信息）
table, err := cdh.GetTable(ctx, configID, dbs.GetItems()[0].GetId(), tables.GetItems()[0].GetId())
if err != nil { log.Fatal(err) }
for _, col := range table.GetColumns() {
    log.Printf("  Column: %s (%s)", col.GetName(), col.GetDataType())
}

// 更新配置
_, err = cdh.UpdateConfig(ctx, configID, moi.WithCDHHost("new-host.example.com"))

// 删除配置（级联删除关联的所有元数据）
err = cdh.DeleteConfig(ctx, configID)
```

### Python

```python
workspace_id = "ws-123"
cdh = client.cdh(workspace_id)

# 创建 CDH 配置
config = cdh.create_config("production-cdh", "hive-metastore.example.com", 9083, "6.3.2",
    connect_timeout=30)
config_id = config["id"]

# 健康检查
health = cdh.health_check(config_id)
print("CDH status:", health.get("status"))

# 同步元数据
sync_resp = cdh.sync_metadata(config_id, "default")
print(f"Synced: {sync_resp['tables_synced']} tables")

# 列出已同步的数据库
dbs = cdh.list_databases(config_id)
for db in dbs.get("items", []):
    print(f"Database: {db['name']} (id={db['id']})")

# 列出表
tables = cdh.list_tables(config_id, dbs["items"][0]["id"])
for t in tables.get("items", []):
    print(f"Table: {t['name']}")

# 获取表详情（含列信息）
table = cdh.get_table(config_id, dbs["items"][0]["id"], tables["items"][0]["id"])
for col in table.get("columns", []):
    print(f"  Column: {col['name']} ({col['data_type']})")

# 更新配置
cdh.update_config(config_id, host="new-host.example.com")

# 删除配置（级联删除关联的所有元数据）
cdh.delete_config(config_id)
```

CDH 相关错误码：`CDH_CONNECTION_FAILED`(2100)、`CDH_DATABASE_NOT_FOUND`(2101)、`CDH_CONFIG_NOT_FOUND`(2104) 等，完整列表见 [catalog-api.md](../api/catalog-api.md) 的 CDH 错误码表。

---

## 8. 错误处理

两边均将服务端错误统一为**错误类型 + 错误码**，可用错误码做分支处理。

### Go

```go
import "github.com/matrixflow/moi-core/model/common"

_, err := client.Catalogs().Get(ctx, workspaceID, catalogID)
if err != nil {
    if moi.IsCode(err, common.ErrorCode_CATALOG_NOT_FOUND) {
        log.Println("Catalog not found")
        return
    }
    if moi.IsCode(err, common.ErrorCode_PERMISSION_DENIED) {
        log.Println("Permission denied")
        return
    }
    log.Fatal(err)
}
```

### Python

```python
import moi

try:
    client.catalogs().get(workspace_id, catalog_id)
except moi.Error as e:
    if moi.is_code(e, moi.ErrorCode.CATALOG_NOT_FOUND):
        print("Catalog not found")
    elif moi.is_code(e, moi.ErrorCode.PERMISSION_DENIED):
        print("Permission denied")
    else:
        raise
```

常用错误码：`CATALOG_NOT_FOUND`、`UNAUTHENTICATED`、`PERMISSION_DENIED`、`INVALID_ARGUMENT` 等。Go 使用 `common.ErrorCode_XXX`，Python 使用 `moi.ErrorCode.XXX`（与 proto `common.ErrorCode` 对齐）。**完整错误码列表**见 [proto/common/error.proto](../proto/common/error.proto) 中的 `ErrorCode` 枚举。

---

## 9. API 能力索引（Go / Python 对照）

| 能力           | Go                                        | Python                                    | 典型用途 / 常用方法                                                                                                                                                                                        |
| -------------- | ----------------------------------------- | ----------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 工作区         | `client.Workspaces()`                     | `client.workspaces()`                     | 创建/获取/列出工作区                                                                                                                                                                                       |
| 目录           | `client.Catalogs()`                       | `client.catalogs()`                       | 在工作区内创建目录、List/Get/Update/Delete                                                                                                                                                                 |
| 数据库         | `client.Databases()`                      | `client.databases()`                      | 同步元数据（SyncMetadata）、Get/List/ListTables/ListVolumes（无 Create/Update/Delete）                                                                                                                     |
| 卷             | `client.Volumes()`                        | `client.volumes()`                        | 在数据库下创建卷（含层级）、Upload/ListFiles/Download/GetChildren/GetPath                                                                                                                                  |
| 文件           | `client.Files()`                          | `client.files()`                          | 按 file_id 上传/下载/删除；DownloadBytes、DownloadToFile 等                                                                                                                                                |
| 用户           | `client.Users()`                          | `client.users()`                          | 用户信息、登录/注册（见 catalog API）                                                                                                                                                                      |
| API Key        | `client.APIKeys()`                        | `client.apikeys()`                        | 当前用户或系统为指定用户创建/列出/删除 Key                                                                                                                                                                 |
| 角色/权限      | (removed in M8)                           | (removed in M8)                           | Core SDK 的旧 Role/Permission facade 已删除。产品角色、binding、Default Role、继承和 policy 改用根目录 Product SDK 的 canonical IAM client；运行时鉴权只调用 Core PDP。该变更不向后兼容。 | Core SDK 旧角色与 permission definition 查询物理移除 |
| 工作流定义     | `client.Workflows(workspaceID)`           | `client.workflows(workspace_id)`          | Create/Get/List/Update/Delete 工作流定义                                                                                                                                                                   |
| 工作流版本     | `client.WorkflowVersions(workspaceID)`    | `client.workflow_versions(workspace_id)`  | Go: CreateByBuilder/CreateByDSLFile；Python: create/create_by_yaml；两边均支持 Publish、List                                                                                                               |
| 任务           | `client.Tasks(workspaceID)`               | `client.tasks(workspace_id)`              | Create/Get/List/Cancel；创建时指定 version_id、data、notification                                                                                                                                          |
| WorkItem 列表  | `client.WorkItems(workspaceID)`           | `client.work_items(workspace_id)`         | List 当前用户可用工作项（node_id + 元数据），供画图时选用                                                                                                                                                  |
| Worker         | `client.Worker(workspaceID)`              | `client.worker(workspace_id)`             | Go: RegisterWorkItem/Connect/ExecuteByWorkflowVersion/Name/InvokeDynamicServiceSync；Python: register_work_item/connect/execute_by_workflow_version/name/invoke_dynamic_service_sync；两边均支持通知与等待 |
| CDH            | `client.CDH(workspaceID)`                 | `client.cdh(workspace_id)`                | CDH 配置 CRUD（CreateConfig/GetConfig/ListConfigs/UpdateConfig/DeleteConfig）、元数据同步（SyncMetadata）、查询（ListDatabases/GetDatabase/ListTables/GetTable）、健康检查（HealthCheck）                  |
| 语义模型       | `client.SemanticModels(workspaceID)`      | `client.semantic_models(workspace_id)`    | Create/Get/List/Update/Delete 语义模型，Entry CRUD，Import/Export/Validate                                                                                                                                 |
| LLM 代理       | `client.LLM(workspaceID)`                 | `client.llm(workspace_id)`                | Backend/Endpoint/Router 管理、ChatCompletion（流式）、Session/Message/Tag CRUD                                                                                                                             |
| Embedding 代理 | `client.Embeddings(workspaceID)`          | `client.embeddings(workspace_id)`         | CreateEmbeddings（OpenAI 兼容）；Backend/Endpoint 通过 HTTP API 管理                                                                                                                                       |

详细方法签名与选项见 [go-sdk-api.md](../api/go-sdk-api.md)、[go-sdk-examples.md](../examples/go-sdk-examples.md)、[python-sdk/README.md](../python-sdk/README.md)。

---

## 10. DSL 定义工作流（代码方式与 YAML 方式）

工作流由**节点**与**控制流**组成：节点类型包括 WorkItem（由 Worker 执行）、JQ（数据变换/条件）、Subnet（子网）、Callback（回调）；控制流包括顺序（Chain）、并行（Parallel）、条件分支（Xor）、多路 OR（Or）、循环（Loop）。
完整语法与连接器细节见 [DSL.md](./DSL.md)；内建与已注册工作项说明见 [WORKITEMS.md](../workflow/WORKITEMS.md)。

### 10.1 工作流控制流与定义对照

| 控制流           | 含义                          | Go 代码                                                         | Python 代码                                                     | YAML（root 下）                                                          |
| ---------------- | ----------------------------- | --------------------------------------------------------------- | --------------------------------------------------------------- | ------------------------------------------------------------------------ |
| **顺序**         | 步骤依次执行                  | `Workflow(...).Chain(n1, n2, ...)`                              | `Workflow(...).Chain(n1, n2, ...)`                              | `chain: [ node1, node2, ... ]`                                           |
| **并行**         | 多节点同时执行，可 Merge 汇聚 | `.Parallel(n1, n2).Merge(jqNode)`                               | `.Parallel(n1, n2).Merge(jqNode)`                               | `parallel: { nodes: [...], merge: {...}, post_chain: [...] }`            |
| **条件分支 XOR** | 按条件走一条分支              | `.Chain(cond).Xor(NewBranch(query, node), ...)`                 | `.Chain(cond).Xor(NewBranch(query, node), ...)`                 | `chain: [cond]` + `xor: { branches: [{ query, node }], post_chain }`     |
| **多路 OR**      | 多条件对应多分支，可并发      | `.Or(conditions, node1, node2, ...)`                            | `.Or(conditions, node1, node2, ...)`                            | `or: { conditions: [...], branches: [...] }`                             |
| **循环**         | 条件为真时重复执行 body       | `.Loop(init, condition, query, bodyFn, exit)`                   | 使用 `loop` 结构（见 DSL.md / loader）                          | `loop: { init, condition, condition_query, body, exit, post_chain }`     |
| **子网**         | 引用已定义的子流程            | `.DefineSubnet("netA", fn).Chain(..., Subnet("callA", "netA"))` | `.DefineSubnet("netA", fn).Chain(..., Subnet("callA", "netA"))` | 顶层 `subnets: { netA: { chain: [...] } }`，节点 `subnet: { name, net }` |

**节点类型**：`WorkItem("节点名", workItemID)`、`JQ("节点名", "jq表达式")`、`Subnet("节点名", "子网名")`、`Callback("节点名", config)`。YAML 中每个节点为 `work_item` / `jq` / `subnet` / `callback` 之一，见 [DSL.md §9](./DSL.md#9-使用-yaml-文件构建工作流)。

**工作流 State**：引擎提供工作流级别的共享键值存储（`.state`），生命周期与单次执行绑定。节点通过 `save:` 声明写入，通过 `{{ .state.<key> }}` 模板语法读取。详见 [DSL.md §10](./DSL.md#10-工作流-state跨节点共享状态)。

### 10.2 代码方式（Go / Python）

#### Go

```go
import "github.com/matrixflow/moi-core/go-sdk/dsl"

// 顺序 + 条件分支
builder := dsl.Workflow("my-net", "root").
    Chain(
        dsl.WorkItem("prepare", "my-workitem"),
        dsl.JQ("cond", `if .x > 0 then . + {"branch": "a"} else . + {"branch": "b"} end`),
    ).
    Xor(
        dsl.NewBranch(`.branch == "a"`, dsl.WorkItem("pathA", "worker-a")),
        dsl.NewBranch(`.branch == "b"`, dsl.WorkItem("pathB", "worker-b")),
    ).
    Chain(dsl.WorkItem("after", "worker-after"))

wf := builder.Build()   // *mowl.Workflow，可交给 CreateByBuilder
// 创建版本
version, err := client.WorkflowVersions(workspaceID).CreateByBuilder(ctx, workflowDefID, builder)
```

#### Python

```python
from moi.dsl import Workflow, WorkItem, JQ, NewBranch, workflow_to_dict

builder = (
    Workflow("my-net", "root")
    .Chain(
        WorkItem("prepare", "my-workitem"),
        JQ("cond", 'if .x > 0 then . + {"branch": "a"} else . + {"branch": "b"} end'),
    )
    .Xor(
        NewBranch('.branch == "a"', WorkItem("pathA", "worker-a")),
        NewBranch('.branch == "b"', WorkItem("pathB", "worker-b")),
    )
    .Chain(WorkItem("after", "worker-after"))
)
workflow_dict = workflow_to_dict(builder)   # 与 proto Workflow 兼容的 dict
# 创建版本
version = client.workflow_versions(workspace_id).create(workflow_def_id, workflow_dict)
```

### 10.3 YAML 方式（Go / Python）

YAML 顶层结构：`workflow`（name、root）、`subnets`（可选）、`root`（chain / parallel / xor / or / loop）。节点为 `work_item`、`jq`、`subnet`、`callback` 之一，格式见 [DSL.md §9](./DSL.md#9-使用-yaml-文件构建工作流)。

**示例 YAML（线性 + 分支）：**

```yaml
workflow:
  name: "demo"
  root: "root"
root:
  chain:
    - work_item: { name: "prepare", id: "my-workitem" }
    - jq:
        {
          name: "cond",
          expr: 'if .x > 0 then . + {"branch": "a"} else . + {"branch": "b"} end',
        }
  xor:
    branches:
      - query: '.branch == "a"'
        node: { work_item: { name: "pathA", id: "worker-a" } }
      - query: '.branch == "b"'
        node: { work_item: { name: "pathB", id: "worker-b" } }
    post_chain:
      - work_item: { name: "after", id: "worker-after" }
```

#### Go：从 YAML 加载并创建版本

```go
import "github.com/matrixflow/moi-core/go-sdk/dsl"

// 仅加载为 *mowl.Workflow（内存）
wf, err := dsl.LoadWorkflowFromFile("workflow.yaml")
// 或 dsl.LoadWorkflowFromBytes(yamlBytes)

// 通过 SDK 从 YAML 创建工作流版本
version, err := client.WorkflowVersions(workspaceID).CreateByDSLFile(ctx, workflowID, "workflow.yaml")
// 或 CreateByDSLBytes(ctx, workflowID, yamlBytes)
```

#### Python：从 YAML 加载并创建版本

```python
from moi.dsl import load_yaml, load_yaml_file

# 从字符串或文件加载为 workflow dict（与 proto 兼容）
workflow_dict = load_yaml(yaml_string)
# 或
workflow_dict = load_yaml_file("workflow.yaml")

# 通过 SDK 从 YAML 创建版本
version = client.workflow_versions(workspace_id).create_by_yaml(
    workflow_id, yaml_content, description="from yaml"
)
# yaml_content 可为 str 或 bytes
```

---

## 11. 获取当前用户可用的 WorkItem

工作流中的 **WorkItem 节点**需要绑定一个 **工作项 ID**（`node_id`）：要么是**内建**的（如 `catalog:http.request`），要么是**已注册**的（某 Worker 调用 `RegisterWorkItem` 后对当前用户可见）。
列出「当前用户可用工作项」可得到所有可用的 `node_id` 及元数据（描述、input/output schema、隔离级别等），便于在画工作流时选用。内建节点与已注册工作项的区别、可见范围见 [WORKITEMS.md](../workflow/WORKITEMS.md)。

### Go

```go
workitems, err := client.WorkItems(workspaceID).List(ctx)
if err != nil { ... }
// workitems: map[string]*mowl.WorkItemMetadataList
// key = node_id（在 DSL 中用作 dsl.WorkItem("步骤名", node_id) 的第二个参数）
for nodeID, list := range workitems {
    for _, meta := range list.GetItems() {
        fmt.Printf("node_id=%s description=%s version=%s\n",
            nodeID, meta.GetDescription(), meta.GetVersion())
    }
}
```

### Python

```python
workitems = client.work_items(workspace_id).list()
# workitems: dict[str, dict]，key 为 node_id，value 形如 {"items": [...]}
for node_id, meta_list in workitems.items():
    for meta in meta_list.get("items", []):
        print("node_id=", node_id, "description=", meta.get("description"), "version=", meta.get("version"))
```

**在 DSL 中选用**：做数据变换/条件用 `JQ`，做 HTTP 请求（无需外部 Worker）用 `WorkItem("步骤名", "catalog:http.request")`，做由 Worker 执行的步骤时从列表里选一个 **node_id**，写 `WorkItem("步骤名", node_id)`（Go）或 `WorkItem("步骤名", node_id)`（Python）。
更多选用建议见 [WORKITEMS.md §5](../workflow/WORKITEMS.md#5-在工作流中选用工作项的建议流程)。

---

## 12. 自定义 Worker

**Worker** 负责两件事：**执行图中 WorkItem 节点**（引擎把「执行某 workitem」的请求发给已连接并注册了该名字的 Worker），以及**主动创建任务**（如 `ExecuteByWorkflowVersion` / `ExecuteByWorkflowName`）。
自定义 Worker = 用 SDK 创建带 `worker_id` 的客户端 → **注册一个或多个 WorkItem 处理函数**（名字与图中 `WorkItem("节点名", workItemID)` 的 **workItemID** 一致）→ **Connect** 后引擎才会把执行请求发过来。

### 12.1 注册 WorkItem 与 Connect（Go）

```go
client, _ := moi.New(endpoint, apiKey, moi.WithWorkerID("my-worker"))
worker := client.Worker(workspaceID)

// 注册名为 "my-workitem" 的处理函数（图中需写 dsl.WorkItem("步骤名", "my-workitem")）
err := worker.RegisterWorkItem("my-workitem", &mowl.WorkItemMetadata{
    Description:    "我的工作项",
    IsolationLevel: mowl.IsolationLevel_PRIVATE,  // 或 IsolationLevel_PUBLIC / IsolationLevel_SHARED
}, func(ctx context.Context, wctx moi.WorkItemContext, msg *mowl.MowlMessage) (*mowl.MowlMessage, error) {
    // 从 msg 取 data/vars，执行业务逻辑，返回结果
    data := msg.GetData()
    vars := msg.GetVars()
    // ... 处理 ...
    return &mowl.MowlMessage{Data: `{"result": "ok"}`, Status: "COMPLETED"}, nil
})
if err != nil { log.Fatal(err) }

worker.Connect(ctx)
defer worker.Disconnect()
// 此后引擎会向该 Worker 派发「执行 my-workitem」的请求
```

**可选：声明 input/output schema**（便于动态服务校验与文档）：

```go
inputSchema := moi.NewSchema().
    Property("url", moi.StringSchema().Description("目标 URL")).
    Required("url")
outputSchema := moi.NewSchema().
    Property("status", moi.StringSchema())

err := worker.RegisterWorkItemWithOptions("my-workitem", metadata, handler,
    moi.WithInputSchemaBuilder(inputSchema),
    moi.WithOutputSchemaBuilder(outputSchema),
)
```

### 12.2 注册 WorkItem 与 Connect（Python）

Worker 消息类型（如 `MowlMessage`、`WorkItemMetadata`）来自 `make proto-python` 生成的代码（`moi._proto.mowl.mowl_pb2`）；枚举常量（如 workflow/result_mode/isolation）建议优先使用稳定导出 `moi.enums`。若未生成，请在项目根目录执行 `make proto-python`。

```python
import moi
from moi import enums
from moi._proto.mowl import mowl_pb2

client = moi.new(endpoint, api_key, worker_id="my-worker")
worker = client.worker(workspace_id)

def handle_my_workitem(ctx, wctx, msg):
    # msg 为 MowlMessage，含 Data、Vars 等；执行业务逻辑后返回 MowlMessage
    return mowl_pb2.MowlMessage(Status=moi.Status.COMPLETED, Data='{"result": "ok"}')

metadata = mowl_pb2.WorkItemMetadata(
    description="我的工作项",
    isolation_level=enums.ISOLATION_LEVEL_PRIVATE,
)
worker.register_work_item("my-workitem", metadata, handle_my_workitem)
worker.connect()
# 此后引擎会向该 Worker 派发「执行 my-workitem」的请求
# 保持进程运行以接收调度
```

也可返回 `(result_msg, None)` 表示成功、`(None, error)` 表示失败；或使用 `register_work_item_with_options` 配合 `with_input_schema_builder` / `with_output_schema_builder`（传入 SchemaBuilder）或 `with_input_schema` / `with_output_schema`（传入 JSON 字符串）声明 schema：

```python
from moi import new_schema, string_schema, with_input_schema_builder, with_output_schema_builder

input_schema = new_schema().property("url", string_schema().description("目标 URL")).required("url")
output_schema = new_schema().property("status", string_schema())

worker.register_work_item_with_options(
    "my-workitem", metadata, handle_my_workitem,
    with_input_schema_builder(input_schema),
    with_output_schema_builder(output_schema),
)
```

### 12.3 要点小结

| 项                     | 说明                                                                                                                                                                                                                                                                                               |
| ---------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **worker_id**          | 创建 Client 时传入，用于引擎区分不同 Worker 进程。                                                                                                                                                                                                                                                 |
| **注册名与图一致**     | `RegisterWorkItem(name, ...)` 的 `name` 必须与工作流里 `WorkItem("节点名", name)` 的第二个参数一致。                                                                                                                                                                                               |
| **Connect 后才有调度** | 未 `Connect` 时引擎不会把执行请求发给该 Worker；图中用到该 workitem 的节点会一直等待。                                                                                                                                                                                                             |
| **多副本一致性**       | 同一用户可用多个 Worker 副本注册同一 `node_id`；若元数据（`isolation_level`/`version`/`input_schema`/`output_schema`）不一致，副本在 `Connect` 时会被拒绝并返回 `ALREADY_EXISTS`。                                                                                                                 |
| **流式 / 双模式**      | 当工作项需**持续上报进度**或**多次产出中间结果**（如长时任务、流式输出）时，可使用 `RegisterStreamWorkItem`（流式）或 `RegisterDualModeWorkItem`（同时支持一次返回与流式）；Python 有对应流式注册。详见 [go-sdk-api.md](../api/go-sdk-api.md) 与 [python-sdk/README.md](../python-sdk/README.md)。 |

### 12.3.1 Handler 返回值约定（Go）

Go handler 签名为 `func(ctx, wctx, msg) (*mowl.MowlMessage, error)`，框架根据返回值走三条路径：

| 返回值                                              | 框架行为                                                                                       | 适用场景                                                   |
| --------------------------------------------------- | ---------------------------------------------------------------------------------------------- | ---------------------------------------------------------- |
| `return nil, err`                                   | 框架包装为 `{Status: "FAILED", Error: err.Error()}`                                            | 业务出错，直接上报错误                                     |
| `return nil, nil`                                   | 框架用 `msg.Data` 和 `msg.Vars` 包装为 `{Status: "COMPLETED", Data: msg.Data, Vars: msg.Vars}` | **推荐**：大多数场景，处理完毕后将结果写入 `msg.Data` 即可 |
| `return &mowl.MowlMessage{Status: "...", ...}, nil` | 框架直接使用该返回值（要求 `Status` 非空，否则视为 FAILED）                                    | 需要自定义 Status 或完全控制响应字段的高级场景             |

**推荐做法**：绝大多数 handler 应使用 `return nil, nil` 模式——将结果 JSON 写入 `msg.Data`，由框架自动包装为 COMPLETED 响应。这是最简洁的写法，也是内置 worker 采用的标准模式。

```go
// ✅ 推荐：修改 msg.Data，返回 nil, nil
func myHandler(ctx context.Context, wctx moi.WorkItemContext, msg *mowl.MowlMessage) (*mowl.MowlMessage, error) {
    // 解析输入
    var in MyInput
    if err := json.Unmarshal([]byte(msg.Data), &in); err != nil {
        return nil, fmt.Errorf("parse input: %w", err)
    }
    // 处理业务逻辑...
    out, _ := json.Marshal(MyOutput{Result: "ok"})
    msg.Data = string(out)
    return nil, nil  // 框架自动包装为 COMPLETED
}
```

**注意**：由于 `return nil, nil` 时返回值为 nil，在**单元测试中直接调用 handler 函数**时不要使用返回的 `*MowlMessage`，而应从传入的 `msg.Data` 读取结果：

```go
// ✅ 测试时从 msg.Data 读取结果
msg := &mowl.MowlMessage{Data: `{"input": "value"}`}
_, err := myHandler(context.Background(), nil, msg)
assert.NoError(t, err)
// 结果在 msg.Data 中，而非返回值
var result MyOutput
json.Unmarshal([]byte(msg.Data), &result)
```

只有在需要**自定义 Status**（如中间态 `WAITING`、`PAUSED`）或需要在响应中同时携带 `Error` 字段（partial failure）等高级场景时，才需要返回非 nil 的 `*MowlMessage`，且必须设置 `Status` 字段。

### 12.4 Runtime 临时 Worker（按需拉起与回收）

除了自己**长期运行** Worker 进程并 `Connect` 外，还可以由 **Catalog Runtime** 在**执行前按需拉起** Worker、在**执行结束或服务停止后自动回收**，无需单独部署 Worker 常驻进程。这类由 Runtime 管理生命周期的 Worker 称为 **Runtime 临时 Worker**（或动态 Worker）。

**两种场景**：

| 场景           | 生命周期                                                                                                                                                    | 用法                                             |
| -------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------ |
| **任务级**     | 创建任务时传入 `runtime_spec_json`；Catalog 在任务开始前拉起 Worker、等待其向 Mowl 注册后开始执行；任务结束（成功/失败/取消）后自动回收。                   | 适合单次任务需要专用 Worker、不希望常驻进程。    |
| **动态服务级** | 创建并发布工作流版本时传入 `runtime_spec_json`；Publish 后 Catalog 启动 Worker 并保持运行，多次 Invoke 复用同一批 Worker；停止服务或 Deprecate 版本后回收。 | 适合动态服务需要专用 Worker 池、由服务版本绑定。 |

**前置条件**：Catalog 配置中 `[runtime] enabled = true`，且至少启用一种 Provider（如 `[runtime.local]` 或 `[runtime.cloud]`）。Local 支持 Docker 镜像或本地二进制；Cloud 支持 K8s Pod。配置说明见 [DESIGN.md §4.4](./DESIGN.md) 与 `catalog/etc/config-*.toml`。

**JSON 结构**：`runtime_spec_json` 为 JSON 字符串，根结构为 `{"workers": [ ... ]}`。每个元素为 WorkerSpec，常用字段：`worker_id`（必填，与图中 WorkItem 对应）、`provider_type`（可选，`"local"`/`"cloud"`，空则自动选择）、`source.type`（`"image"` 或 `"binary"`）、`source.image.repository`（镜像地址）或 `source.binary.path`（本地二进制路径）、`env`（环境变量）、`startup_timeout`（等待注册秒数，默认 60）。Catalog 会在拉起前注入 `mowl_endpoint`；`api_key` 需在 spec 或环境中提供，供 Worker 连接 Mowl 使用。完整字段见 [DESIGN.md §4.4](./DESIGN.md) 的 WorkerSpec 表。

#### 任务级：创建任务时指定 RuntimeSpec（Go）

```go
runtimeSpecJSON := `{"workers":[{"worker_id":"task-worker-1","provider_type":"local","source":{"type":"image","image":{"repository":"my-registry.io/my-worker:v1"}},"startup_timeout":60}]}`

task, err := client.Tasks(workspaceID).Create(ctx, "run-001",
    moi.WithTaskWorkflowVersionID(versionID),
    moi.WithTaskData(`{}`),
    moi.WithTaskRuntimeSpecJson(runtimeSpecJSON),
)
// Catalog 会先拉起 Worker，等待注册后再创建任务；任务结束后自动回收 Worker
```

#### 任务级：创建任务时指定 RuntimeSpec（Python）

```python
runtime_spec_json = '''{"workers":[{"worker_id":"task-worker-1","provider_type":"local","source":{"type":"image","image":{"repository":"my-registry.io/my-worker:v1"}},"startup_timeout":60}]}'''

task = client.tasks(workspace_id).create(
    "run-001",
    workflow_version_id=version_id,
    data="{}",
    runtime_spec_json=runtime_spec_json,
)
```

#### 动态服务级：发布版本时指定 RuntimeSpec（Go）

```go
runtimeSpecJSON := `{"workers":[{"worker_id":"svc-worker-1","provider_type":"local","source":{"type":"image","image":{"repository":"my-registry.io/dynamic-svc-worker:v1"}}}],"flow_control":{"timeout_seconds":30,"max_concurrency":8,"rate_limit_per_min":120}}`

version, err := client.WorkflowVersions(workspaceID).CreateByBuilder(ctx, defID, builder,
    moi.WithVersionDynamicService(inputSchema, outputSchema),
    moi.WithVersionResultMode(mowl.ResultMode_RESULT_MODE_ONESHOT),
    moi.WithVersionRuntimeSpecJson(runtimeSpecJSON),
)
client.WorkflowVersions(workspaceID).Publish(ctx, version.GetId())
// Publish 后 Catalog 会启动该 Worker，动态服务 Invoke 时复用；停止服务或 Deprecate 后回收
```

#### 动态服务级：发布版本时指定 RuntimeSpec（Python）

```python
from moi import enums

runtime_spec_json = '{"workers":[{"worker_id":"svc-worker-1","provider_type":"local","source":{"type":"image","image":{"repository":"my-registry.io/dynamic-svc-worker:v1"}}}],"flow_control":{"timeout_seconds":30,"max_concurrency":8,"rate_limit_per_min":120}}'

version = client.workflow_versions(workspace_id).create(
    wf_def_id, workflow_dict,
    workflow_type=enums.WORKFLOW_TYPE_DYNAMIC_SERVICE,
    input_schema=..., output_schema=...,
    result_mode=enums.RESULT_MODE_ONESHOT,
    runtime_spec_json=runtime_spec_json,
)
client.workflow_versions(workspace_id).publish(version["id"])
```

`flow_control` 由中心 `mowl-engine` 在创建 transient case 前执行；`timeout_seconds`、`max_concurrency`、`rate_limit_per_min` 为 `0` 时表示不启用对应限制。

更多：Provider 类型、WorkerSpec 完整字段、OrphanGC、StartupCleanup 见 [DESIGN.md §4.4](./DESIGN.md) 与 [catalog/pkg/runtime](../catalog/pkg/runtime)。

---

## 13. Use Case 7：工作流结果适配（Parse Results）

该能力用于把历史解析结果处理（view/modify/export）逻辑迁移到 moi-core。
底层类型统一使用 protobuf `catalog.ParseResult`，同时 SDK 提供更易用的构造 helper。

### Go

```go
import catalogpb "github.com/matrixflow/moi-core/model/catalog"

svc := client.ParseResults(workspaceID)
comp := moi.NewParseResult(
    "原始内容",
    moi.WithParseResultID("r1"),
    moi.WithParseResultBlockType("text"),
    moi.WithParseResultIndex(1),
    moi.WithParseResultMeta(map[string]interface{}{"lang": "zh"}),
)

view, err := svc.View(ctx, moi.ParseResultParserDocument, []*catalogpb.ParseResult{comp})
modified, err := svc.Modify(ctx, moi.ParseResultParserDocument, comp, "修正后内容")
files, err := svc.Export(ctx, moi.ParseResultParserDocument, "demo", []*catalogpb.ParseResult{modified})
_ = view
_ = files
```

### Python

```python
comp = moi.new_parse_result(
    "原始内容",
    result_id="r1",
    block_type="text",
    index=1,
    meta={"lang": "zh"},
)

svc = client.parse_results(workspace_id)
view = svc.view(moi.DOCUMENT_PARSER, [comp])
modified = svc.modify(moi.DOCUMENT_PARSER, comp, "修正后内容")
files = svc.export(moi.DOCUMENT_PARSER, "demo", [modified])
```

---

## 14. Use Case 8：语义模型管理

语义模型（Semantic Model）用于管理与数据源绑定的语义知识，供查询改写、意图分解、NL2SQL 等 WorkItem 自动拉取使用。典型流程：**创建语义模型 → 添加语义条目 → 在工作流输入的 `config.data_source.semantic_models` 中引用语义模型 ID**。WorkItem（如 `question_rewrite`、`intent_decompose`、`nl2sql_exec`）会根据 `config.data_source.semantic_models` 自动从语义模型拉取对应类型的语义数据。

### 14.1 语义模型 CRUD

#### Go

```go
// 创建语义模型
model, err := client.SemanticModels(workspaceID).Create(ctx, &moi.SemanticModelUpsertRequest{
    Name:   "sales-model",
    Tables: json.RawMessage(`[{"db_name":"mydb","table_names":["orders","products"],"parents":[]}]`),
})
modelID := model.ID

// RAG 知识库使用同一张 semantic_models 表。向量表和 embedding 模型写入 files 的结构化字段，
// 文件/volume 的索引血缘继续由 data_asset + data_derivation 记录。
ragModel, err := client.SemanticModels(workspaceID).Create(ctx, &moi.SemanticModelUpsertRequest{
    Name:   "docs",
    Tables: json.RawMessage(`[]`),
    Files:  json.RawMessage(`{"file_ids":[],"parents":[],"vector_table":"kb_docs","embedding_model":"bge-m3"}`),
})

// 如果只需要登记/补齐向量资产元数据，可在 CreateAsset 中使用 replace_meta。
vectorAsset, err := client.DataAssets(workspaceID).CreateAsset(ctx, "kb_docs",
    moi.WithDataAssetType("vector_index"),
    moi.WithDataAssetMetaMap(map[string]interface{}{"embedding_model": "bge-m3"}),
    moi.WithDataAssetReplaceMeta(),
)

// 获取
model, err = client.SemanticModels(workspaceID).Get(ctx, modelID)

// 列出
listResp, err := client.SemanticModels(workspaceID).List(ctx, moi.WithPageSize(20))

// 更新
_, err = client.SemanticModels(workspaceID).Update(ctx, modelID, &moi.SemanticModelUpsertRequest{
    Name:   "sales-model-v2",
    Tables: json.RawMessage(`[{"db_name":"mydb","table_names":["orders","products"],"parents":[]}]`),
})

// 删除
err = client.SemanticModels(workspaceID).Delete(ctx, modelID)
```

#### Python

```python
# 创建语义模型
model = client.semantic_models(workspace_id).create(
    name="sales-model",
    tables=[{"db_name": "mydb", "table_names": ["orders", "products"], "parents": []}],
)
model_id = model["id"]

# RAG 知识库同样创建 semantic model，并把向量索引元数据写入 files。
rag_model = client.semantic_models(workspace_id).create(
    name="docs",
    tables=[],
    files={"file_ids": [], "parents": [], "vector_table": "kb_docs", "embedding_model": "bge-m3"},
)

# 获取
model = client.semantic_models(workspace_id).get(model_id)

# 列出
list_resp = client.semantic_models(workspace_id).list(page_size=20)

# 更新
client.semantic_models(workspace_id).update(
    model_id, name="sales-model-v2",
    tables=[{"db_name": "mydb", "table_names": ["orders", "products"], "parents": []}],
)

# 删除
client.semantic_models(workspace_id).delete(model_id)
```

### 14.2 语义条目管理

语义条目绑定在语义模型下，按 `kind` 分类。支持的类型：

| kind                | 用途                                 | 消费方                                                |
| ------------------- | ------------------------------------ | ----------------------------------------------------- |
| `logic_text`        | 业务逻辑规则（如价格规则、分类规则） | `question_rewrite`、`intent_decompose`、`nl2sql_exec` |
| `glossary`          | 术语表                               | `nl2sql_exec`                                         |
| `verified_query`    | 已验证的 SQL 查询                    | `nl2sql_exec`                                         |
| `metric`            | 指标定义                             | `nl2sql_exec`                                         |
| `dimension`         | 维度定义                             | `nl2sql_exec`                                         |
| `relationship`      | 表关联关系                           | `nl2sql_exec`                                         |
| `column_preference` | 列偏好                               | `nl2sql_exec`                                         |
| `named_filter`      | 命名过滤器                           | `nl2sql_exec`                                         |
| `fact`              | 事实定义                             | `nl2sql_exec`                                         |
| `sql_resultset`     | 可调用 SQL 结果集                    | `nl2sql_exec` / `resolve_sql_resultset`               |

#### Go

```go
// 创建语义条目（logic_text 类型）
entry, err := client.SemanticModels(workspaceID).CreateEntry(ctx, modelID, &moi.SemanticEntryUpsertRequest{
    Kind:   "logic_text",
    Key:    "price_rule",
    Tables: []string{"orders"},
    Spec:   json.RawMessage(`{"content":"价格单位为元","injection_stages":["planner_policy","sql_generation"]}`),
})

// 列出
entryList, err := client.SemanticModels(workspaceID).ListEntries(ctx, modelID, "logic_text")

// 删除
err = client.SemanticModels(workspaceID).DeleteEntry(ctx, modelID, entry.ID)
```

`sql_resultset` 可选配置 `retrieval.embedding_model`。配置后，当结果集行数超过 100 行时，`resolve_sql_resultset` 会先用全文相似度和该 embedding 模型召回候选行，再把最多 100 行候选交给隔离 resolver 判断，避免把完整大结果集直接塞进 LLM。

```json
{
  "sql": "SELECT code, label FROM dim_account_source",
  "description": "合并科目编码映射",
  "retrieval": {
    "enabled": true,
    "embedding_model": "BAAI/bge-m3"
  }
}
```

#### Python

```python
# 创建语义条目
entry = client.semantic_models(workspace_id).create_entry(
    model_id, kind="logic_text", key="price_rule",
    tables=["orders"],
    spec={"content": "价格单位为元", "injection_stages": ["planner_policy", "sql_generation"]},
)

# 列出
entries = client.semantic_models(workspace_id).list_entries(model_id, kind="logic_text")

# 删除
client.semantic_models(workspace_id).delete_entry(model_id, entry["id"])
```

### 14.3 在查询 WorkItem 中使用语义模型

在工作流输入中通过 `config.data_source.semantic_models` 传入语义模型 ID 列表。相关 WorkItem 会自动拉取对应类型的语义条目：

```json
{
  "request": {
    "question": "查询所有产品的价格",
    "config": {
      "data_source": {
        "tables": {
          "db_name": "mydb",
          "table_list": ["products"]
        },
        "semantic_models": [{ "semantic_model_id": 123 }]
      }
    }
  }
}
```

各 WorkItem 的语义消费逻辑：

- `question_rewrite`：从语义模型拉取 `logic_text` 类型条目，提取语义关键词用于问题改写
- `intent_decompose`：从语义模型拉取 `logic_text` 类型条目作为业务逻辑上下文，辅助意图分解
- `nl2sql_exec`：从知识库拉取所有类型知识（`table_description`、`logic`、`glossary`、`synonym`），构建 NL2SQL 上下文

---

## 15. Use Case 9：LLM 代理与 Embedding 代理

moi-core 提供 **LLM 代理** 和 **Embedding 代理** 能力：统一管理多个模型后端（Backend）及其端点（Endpoint），通过路由策略（Router）自动选择可用端点，对外暴露 OpenAI 兼容的 API。WorkItem（如 `nl2sql_exec`、`insight_generate`）通过代理访问 LLM 和 Embedding，无需直连模型服务。

### 15.1 架构概述

```
SDK / WorkItem
    │
    ├─► LLM Proxy:       POST /api/v1/workspaces/:id/llm/chat/completions
    │   └─► Router → Backend(s) → Endpoint(s) → 实际 LLM 服务（如 Qianwen、OpenAI）
    │
    └─► Embedding Proxy:  POST /api/v1/workspaces/:id/embeddings
        └─► Router → Backend(s) → Endpoint(s) → 实际 Embedding 服务（如 SiliconFlow）
```

每个工作区独立管理 Backend/Endpoint/Router 配置。LLM 和 Embedding 各自独立一套。

### 15.2 LLM Backend 管理

Backend 代表一个模型提供商（如 Qianwen、OpenAI），包含 API Key、支持的模型列表、超时配置。每个 Backend 下可挂多个 Endpoint（地址），支持上下线。

#### Go

```go
llm := client.LLM(workspaceID)

// 创建 Backend
backend, err := llm.CreateBackend(ctx,
    moi.WithBackendName("qianwen"),
    moi.WithBackendType(catalogpb.BackendType_QIANWEN),
    moi.WithBackendAPIKey("sk-xxx"),
    moi.WithBackendTimeout(120),
    moi.WithBackendModels([]string{"qwen3-max", "qwen-turbo"}),
)

// 添加 Endpoint
endpoint, err := llm.CreateEndpoint(ctx, backend.Id,
    moi.WithEndpointAddress("https://dashscope.aliyuncs.com/api/v1"),
)

// 列出 / 获取 / 更新 / 删除
backends, err := llm.ListBackends(ctx)
b, err := llm.GetBackend(ctx, backend.Id)
b, err = llm.UpdateBackend(ctx, backend.Id, moi.WithUpdateBackendModels([]string{"qwen3-max"}))
err = llm.DeleteBackend(ctx, backend.Id)

// 上下线 Endpoint
err = llm.SetEndpointStatus(ctx, backend.Id, endpoint.Id,
    moi.WithEndpointStatus(catalogpb.EndpointStatus_OFFLINE),
)
```

#### Python

```python
llm = client.llm(workspace_id)

# 创建 Backend（type_ 对应 proto BackendType 枚举值）
backend = llm.create_backend("qianwen", 3,  # BackendType_QIANWEN = 3
    api_key_encrypted="sk-xxx",
    timeout_seconds=120,
    models=["qwen3-max", "qwen-turbo"],
)

# 添加 Endpoint
endpoint = llm.create_endpoint(backend["id"], "https://dashscope.aliyuncs.com/api/v1")

# 列出 / 获取 / 更新 / 删除
backends = llm.list_backends()
b = llm.get_backend(backend["id"])
b = llm.update_backend(backend["id"], models=["qwen3-max"])
llm.delete_backend(backend["id"])

# 上下线 Endpoint
llm.set_endpoint_status(backend["id"], endpoint["id"], 1)  # 1 = OFFLINE
```

### 15.3 LLM Router 配置

Router 控制请求如何分发到多个 Backend/Endpoint。

| 配置项                          | 说明                       | 默认值      |
| ------------------------------- | -------------------------- | ----------- |
| `strategy`                      | 路由策略（ROUND_ROBIN 等） | ROUND_ROBIN |
| `health_check_interval_seconds` | 健康检查间隔               | 30          |
| `max_retries`                   | 最大重试次数               | 2           |
| `enable_session_affinity`       | 会话亲和性                 | false       |

#### Go

```go
config, err := llm.GetRouterConfig(ctx)
config, err = llm.PutRouterConfig(ctx,
    moi.WithRouterStrategy(catalogpb.RouterStrategy_ROUND_ROBIN),
    moi.WithRouterMaxRetries(3),
    moi.WithRouterSessionAffinity(true),
)
```

#### Python

```python
config = llm.get_router_config()
config = llm.put_router_config(strategy=0, max_retries=3, enable_session_affinity=True)
```

### 15.4 LLM Chat Completion（流式）

通过代理调用 LLM，OpenAI 兼容格式，支持 SSE 流式输出。

#### Go

```go
ch, err := client.LLM(workspaceID).ChatCompletion(ctx, "解释 Go 的 goroutine", "qwen3-max",
    moi.WithChatTemperature(0.7),
    moi.WithChatMaxTokens(2048),
)
if err != nil { log.Fatal(err) }
for delta := range ch {
    fmt.Print(delta)
}
```

#### Python

```python
for delta in client.llm(workspace_id).chat_completion("解释 Go 的 goroutine", "qwen3-max",
    temperature=0.7, max_tokens=2048):
    print(delta, end="")
```

**Proxy Extension**：可附带 `proxy` 扩展字段，用于消息记录、会话绑定、mock 测试等：

```go
ch, err := llm.ChatCompletion(ctx, "hi", "qwen3-max",
    moi.WithProxyExtension(&catalogpb.ProxyExtension{
        Source: "workflow", SessionId: 123, RecordMessage: true,
    }),
)
```

### 15.5 Embedding 代理

Embedding 代理与 LLM 代理架构相同（独立的 Backend/Endpoint/Router），对外暴露 OpenAI 兼容的 Embedding API。Backend/Endpoint 管理通过 HTTP API 操作（路径为 `/api/v1/workspaces/:id/embeddings/backends/...`），SDK 提供 `CreateEmbeddings` 方法调用代理。

#### Go

```go
resp, err := client.Embeddings(workspaceID).CreateEmbeddings(ctx, "BAAI/bge-m3",
    []string{"MatrixFlow is a data platform", "Hello world"},
    moi.WithEmbeddingEncodingFormat("float"),
)
// resp.Data 包含每个输入的 embedding 向量
```

#### Python

```python
resp = client.embeddings(workspace_id).create_embeddings(
    "BAAI/bge-m3",
    ["MatrixFlow is a data platform", "Hello world"],
    encoding_format="float",
)
# resp["data"] 包含每个输入的 embedding 向量
```

#### Embedding Backend 管理（HTTP API）

Embedding 的 Backend/Endpoint/Router 管理目前通过 HTTP API 直接操作：

| 操作            | HTTP 方法 | 路径                                                                                   |
| --------------- | --------- | -------------------------------------------------------------------------------------- |
| 列出 Backend    | GET       | `/api/v1/workspaces/:id/embeddings/backends`                                           |
| 创建 Backend    | POST      | `/api/v1/workspaces/:id/embeddings/backends`                                           |
| 获取 Backend    | GET       | `/api/v1/workspaces/:id/embeddings/backends/:backend_id`                               |
| 更新 Backend    | PUT       | `/api/v1/workspaces/:id/embeddings/backends/:backend_id`                               |
| 删除 Backend    | DELETE    | `/api/v1/workspaces/:id/embeddings/backends/:backend_id`                               |
| 添加 Endpoint   | POST      | `/api/v1/workspaces/:id/embeddings/backends/:backend_id/endpoints`                     |
| 上下线 Endpoint | PUT       | `/api/v1/workspaces/:id/embeddings/backends/:backend_id/endpoints/:endpoint_id/status` |

### 15.6 在 WorkItem 中的使用

查询与洞察 WorkItem 通过代理访问 LLM 和 Embedding：

- `nl2sql_exec`：通过 `client.LLM(workspaceID).BaseURL()` 获取 LLM 代理地址，使用 WorkItem context 中的 `execution_context.user_api_key` 调用
- `insight_generate`：同上，流式调用 LLM 生成洞察
- `question_rewrite`：通过 DSPy 框架配置 LLM 代理地址
- RAG ingest 工作流：通过 Embedding 代理生成向量

WorkItem 不直连模型服务，统一走代理，由代理负责路由、重试、API Key 管理。

---

## 16. 文档与示例汇总

| 文档                                                 | 内容                                                                           |
| ---------------------------------------------------- | ------------------------------------------------------------------------------ |
| [CONCEPTS.md](./CONCEPTS.md)                         | 工作流、任务、WorkItem、Worker、动态服务概念                                   |
| [go-sdk-api.md](../api/go-sdk-api.md)                | Go SDK 方法、选项、错误类型                                                    |
| [go-sdk-examples.md](../examples/go-sdk-examples.md) | Go 按能力分类的示例代码                                                        |
| [python-sdk/README.md](../python-sdk/README.md)      | Python SDK 安装、快速开始、服务与 API 对应                                     |
| [DSL.md](./DSL.md)                                   | 使用 DSL 构建工作流：Chain/Parallel/Xor/Or/Loop、WorkItem/JQ/Subnet、YAML/JSON |
| [WORKITEMS.md](../workflow/WORKITEMS.md)             | 可用工作项列表、API/SDK 查询、在工作流中选用                                   |
| [catalog-api.md](../api/catalog-api.md)              | Catalog HTTP 接口（认证、工作区、工作流、任务、动态服务 Invoke）               |
| [DEPLOYMENT.md](./DEPLOYMENT.md)                     | 本地与 K8s 部署、环境变量、API Key 生成                                        |
| [cli/README.md](../cli/README.md)                    | core-cli 命令行工具（catalog、workspace、workflow、task 等）                    |

按上述 Use Case 与索引即可用 **Go 或 Python SDK** 完成：元数据与文件管理、可执行工作流、动态服务、用户与 API Key 等开发。
