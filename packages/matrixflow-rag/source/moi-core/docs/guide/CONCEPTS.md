# MOI-Core 核心概念

本文档用通俗语言说明：**工作流、任务、工作项、Worker、动态服务** 分别是什么、彼此关系，以及开发时如何理解它们。读完后再看 [SDK 开发指南](./SDK_GUIDE.md) 和 [go-sdk-api.md](../api/go-sdk-api.md) 即可上手开发。

---

## 1. 概念总览

| 概念 | 一句话 | 谁创建/谁用 |
|------|--------|-------------|
| **工作流 (Workflow)** | 一张「流程图」的**定义**，包含节点和边，可多版本 | 开发者通过 SDK/API 创建、更新 |
| **工作流版本 (Workflow Version)** | 某次保存的流程图快照，有 draft / published 状态 | 开发者创建；**只有 published** 才能被按「名称」执行 |
| **任务 (Task)** | **一次具体执行**：用哪个工作流、带什么输入、何时跑（立即/定时） | 开发者或调度器创建；引擎调度执行 |
| **工作项 (WorkItem)** | 工作流图里的**一个节点**对应的「可执行单元」，有名字和输入输出约定 | 开发者在 Worker 里**实现**并**注册** |
| **Worker** | **执行工作项的进程**：连上引擎、注册自己能跑哪些 WorkItem、收任务并执行、回报结果 | 开发者部署并运行；一个 Worker 可注册多个 WorkItem |
| **动态服务 (Dynamic Service)** | 一种**按名称 + 输入即调即用**的工作流：有输入/输出 Schema，支持 oneshot 或 stream | 开发者把工作流版本标成「动态服务」并发布；通过 HTTP 或 SDK 调用 |
| **State** | 工作流执行期间的**共享键值存储**，通过 `save:` 写入、`{{ .state.<key> }}` 读取 | 引擎自动管理；开发者在 DSL 中声明 |

关系可以简化为：

```
工作流定义 (Workflow)  ──有多个──►  工作流版本 (Version)  ──可标为──►  动态服务
        │                                    │
        │ 一次执行                            │ 按名称/版本执行
        ▼                                    ▼
      任务 (Task)  ──由引擎调度──►  把图中的节点 (WorkItem) 发给 Worker 执行
                                        │
                                        ▼
                              Worker 进程：注册 WorkItem、执行、返回结果
```

---

## 2. 工作流 (Workflow)

- **是什么**：工作流是一张**有向图**的元数据（名称、描述等），不包含「某次保存的图内容」。图的内容在**版本**里。
- **谁创建**：通过 `client.Workflows(workspaceID).Create(ctx, "my-workflow", ...)` 创建一条「工作流定义」。
- **图里有什么**：节点（如 `WorkItem("step1", "my-workitem")`、条件 `JQ(...)` 等）和边（顺序、分支、并行、循环）。版本里保存的是整张图的 JSON/YAML 表示。**用代码构建图**的完整语法见 [使用 DSL 构建工作流](./DSL.md)。
- **和任务的关系**：任务必须绑定「某个工作流版本」；执行时引擎按该版本的图调度。

---

## 3. 工作流版本 (Workflow Version)

- **是什么**：某条工作流定义下的一次「保存快照」，包含完整图结构 + 可选字段（如动态服务的 input_schema、output_schema、result_mode）。
- **状态**：
  - **draft**：可编辑、可删；可以按 **version_id** 被任务引用并执行。
  - **published**：可被「按工作流名称」解析为「当前已发布版本」；按名称执行时只用 published。
  - **deprecated**：不可再发布，可删。
- **谁创建**：`client.WorkflowVersions(workspaceID).CreateByBuilder(ctx, workflowID, builder, ...)` 或等价 API；若做**动态服务**，需在创建版本时加上 `WithVersionDynamicService(inputSchema, outputSchema)` 等选项。
- **发布**：`WorkflowVersions(workspaceID).Publish(ctx, versionID)`；发布后该版本成为该工作流「按名称」解析的目标。

---

## 4. 任务 (Task)

- **是什么**：**一次执行**的请求。指定：用哪个工作流版本（或按名称用已发布版本）、输入数据（JSON）、是否定时（cron）、是否临时任务等。
- **谁创建**：
  - **HTTP/控制面**：`client.Tasks(workspaceID).Create(ctx, "task-name", WithTaskWorkflowVersionID(versionID), WithTaskData(...))`。
  - **Worker 侧**：`worker.ExecuteByWorkflowVersion("task-name", versionID, WithData(...))` 或 `worker.ExecuteByWorkflowName("task-name", "workflow-name", WithData(...))`。
- **执行方式**：
  - **一次性**：创建后引擎立即调度（或由 Worker 拉取）执行。
  - **定时**：带 `WithTaskCronExpression("CRON_TZ=UTC 0 0 0 * * *")` 等，按 cron 周期触发。页面创建的本地墙上时间 cron 使用 `CRON_TZ=<IANA timezone>` 前缀，例如 `CRON_TZ=Asia/Shanghai 0 0 13 * * *`；未带时区前缀的历史 cron 会按产品默认时区 `Asia/Shanghai` 兼容。
- **和 Case 的关系**：每次执行会产生一个 **Case**（运行实例）；Case 下有 WorkItem 执行记录、Token、日志等。日常开发多数只关心 Task 的创建与状态，Case 由引擎内部管理。

---

## 5. 工作项 (WorkItem) 与节点

- **是什么**：工作流图中的一个**节点**在运行时对应一个「工作项」。节点有类型，例如：
  - **外部工作项**：对应你写的业务逻辑，由 **Worker** 执行；在 DSL 里写成 `dsl.WorkItem("节点名", "workitem-name")`，其中 `workitem-name` 是你在 Worker 里注册的名字。
  - **条件节点**：如 `dsl.JQ("cond", ".value > 0")`，由引擎求值，不发给 Worker。
- **WorkItem 实现**：在 Worker 进程里，你用 `worker.RegisterWorkItem("workitem-name", metadata, func(ctx, data, vars) (result, error) { ... })` 注册一个处理函数；引擎在运行到该节点时，通过 gRPC 把输入发给该 Worker，Worker 执行完后返回结果，引擎再驱动下游节点。**系统提供了哪些可用工作项**（内建节点 + 已注册工作项）、如何查询与在工作流中引用，见 [可用工作项说明](../workflow/WORKITEMS.md)。
- **谁创建**：
  - **图里的节点**：在创建「工作流版本」时用 DSL 或 YAML 定义（如 `WorkItem("step1", "my-workitem")`）。
  - **可执行实现**：在 Worker 代码里用 `RegisterWorkItem` 注册，并 `worker.Connect(ctx)` 连上引擎；引擎会把你注册的 workitem 名单上报，调度时按节点类型和名字匹配。

---

## 6. Worker

- **是什么**：一个**长期运行的进程**，职责是：
  1. 连接 Mowl 引擎（gRPC）；
  2. 注册自己（`RegisterWorker`）以及能执行的 **WorkItem** 列表（`RegisterWorkItem`）；
  3. 通过 **WorkerSession** 双向流接收「执行某工作项」的请求；
  4. 执行本地注册的 handler，把结果/事件回传引擎；
  5. 可选：主动创建任务（`ExecuteByWorkflowVersion` / `ExecuteByWorkflowName`）、接收工作流/节点级通知。
- **谁用**：需要**执行图中「外部工作项」节点**时，必须有至少一个 Worker 注册了对应名字的 WorkItem；否则该节点会一直等待。
- **创建方式**：`client, _ := moi.New(endpoint, apiKey, moi.WithWorkerID("my-worker"))`，然后 `worker := client.Worker(workspaceID)`，`worker.RegisterWorkItem(...)`，最后 `worker.Connect(ctx)`。详见 [SDK_GUIDE.md](./SDK_GUIDE.md) 和 [go-sdk-api.md](../api/go-sdk-api.md)。

---

## 7. 动态服务 (Dynamic Service)

- **是什么**：把一条**已发布的工作流**当作「服务」来用：调用方只提供 **工作流名称 + 输入 JSON**，引擎按名称找到最新 published 版本并执行，然后返回结果（或流式输出）。和「先创建 Task 再轮询状态」不同，动态服务是**即调即用**的接口形态。
- **和普通工作流的区别**：
  - 创建版本时必须声明类型为「动态服务」并带上 **input_schema、output_schema**，可选 **result_mode**（oneshot / stream）。
  - 调用方式：HTTP `POST /api/v1/workspaces/:id/dynamic-services/invoke`（请求体包含 `service_name` 和 `payload`），或 SDK `worker.InvokeDynamicServiceSync` / `InvokeDynamicServiceStream`。
- **谁创建**：开发者创建 Workflow 定义 → 用 DSL 建版本时加 `WithVersionDynamicService(inputSchema, outputSchema)`、`WithVersionResultMode(oneshot/stream)` → `Publish(versionID)`。
- **谁调用**：任何有 API Key 的客户端（HTTP 或 SDK）；若为 stream 模式，则通过 SDK 的 `InvokeDynamicServiceStream` 收流式结果。
- **流控归属**：动态服务版本可在 `runtime_spec_json.flow_control` 中声明 `timeout_seconds`、`max_concurrency`、`rate_limit_per_min`。这些限制由中心 `mowl-engine` 在创建 transient case 前统一执行；Catalog 与业务后端只负责发布配置和转发调用，不保存调用期流控状态。

---

## 8. 工作流 State

- **是什么**：State 是一个**工作流级别的共享键值存储**，生命周期与单次工作流执行（case）绑定。与 `.data`（上一节点输出）和 `.vars`（预设变量）不同，`.state` 在整个工作流执行期间**持续累积**，任意节点均可读写。
- **写入**：在节点上声明 `save:` 字段（YAML）或调用 `.Save(map[string]string{...})`（Go SDK Builder）/ `.Save({...})`（Python SDK Builder），节点执行完成后引擎将指定字段写入 State。
- **读取**：在节点的 `input` 模板中使用 `{{ .state.<key> }}` 语法引用之前节点写入的值。
- **典型场景**：链式工作流中，步骤 A 的输出需要被步骤 C 引用（步骤 B 的输出覆盖了 `.data`）。例如 RAG Ingest 流水线中，`register_raw_asset` 将 `asset_id` 写入 `state.raw_asset_id`，后续的 `link_parsed_from_raw` 通过 `{{ .state.raw_asset_id }}` 读取。
- **并发安全**：并行节点同时写入 State 时，引擎保证不同 key 的写入均被保留；相同 key 遵循 last-writer-wins 语义。
- **大小限制**：State 序列化后不超过 1MB。

---

## 9. 如何串起来（开发视角）

1. **只做元数据/文件**：用 `Workspaces`、`Catalogs`、`Databases`、`Volumes`、`Files` 等 API 即可，不涉及工作流。
2. **要做「可重复执行的工作流」**：
   创建 Workflow 定义 → 创建 Version（DSL 里包含 `WorkItem("xxx", "my-workitem")`）→ 可选 Publish → 创建 Task（指定 version_id 或 workflow 名称）→ 需要有 Worker 注册了 `my-workitem` 并 `Connect`，任务才会被完整执行。
3. **要做「像 API 一样被调用的服务」**：
   同上，但版本创建时用 `WithVersionDynamicService` 并 Publish；调用方用「动态服务 Invoke」接口（HTTP 或 `InvokeDynamicServiceSync`/`InvokeDynamicServiceStream`），无需先 CreateTask。
4. **要在工作流中跨节点传递数据**：
   在节点上声明 `save:` 写入 State，在后续节点的 `input` 中用 `{{ .state.<key> }}` 读取。适用于链式流水线中需要引用更早步骤输出的场景。

下一步请阅读 [SDK 开发指南](./SDK_GUIDE.md)，按步骤用 SDK 创建客户端、工作流、任务或动态服务并运行。
