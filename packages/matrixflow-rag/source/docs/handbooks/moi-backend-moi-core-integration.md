# moi-backend ↔ moi-core 集成手册

> 本文档描述 `moi-backend`（业务网关层）与 `moi-core`（核心引擎层）之间的调用关系、依赖方式及配置方法。

## 1. 架构定位

| 模块 | 角色 | 端口 | 职责 |
|------|------|------|------|
| **moi-backend** | 业务网关 | 8050 | 认证(OAuth/SSO/JWT)、RBAC、Workspace 管理、对 moi-core 的调用编排 |
| **moi-core/catalog** | 核心引擎 | 8081(HTTP) / 8082(gRPC) | 统一元数据管理、内嵌 Mowl 工作流引擎、文件存储、检索 |

调用方向为**单向**：`moi-backend → moi-core`。moi-core 不主动调用 moi-backend。

```
Web Client
  → local-service (RBAC 网关, :8000)
    → moi-backend (:8050)
      → moi-core/catalog (:8081)  ← HTTP via go-sdk
        → mowl 引擎 (内嵌)
        → workers
```

## 2. 三条集成路径

### 2.1 HTTP SDK 调用（运行时主路径）

moi-backend 通过 `moi-core/go-sdk` 以 HTTP 方式调用 moi-core catalog API。

**调用链路：**

```
moi-backend handler (Gin)
  → pkg/caller/caller.go :: NewClient(ctx, endpoint, uid)
    → 从 DB account 表查询 uid 对应的 moi_api_key
    → moi.New(endpoint, apiKey)  // 创建 go-sdk Client
  → client.Catalogs() / .Volumes() / .Workflows() / .Explore() / ...
    → HTTP 请求发往 moi-core catalog :8081
```

**核心入口 — `pkg/caller/caller.go`：**

```go
// NewClient 根据用户 uid 从 DB 查 API Key，创建 moi-core SDK client
func NewClient(ctx context.Context, endpoint, uid string) (*moi.Client, error) {
    var row accountRow
    if err := resolveDB(ctx).Where("uid = ?", uid).First(&row).Error; err != nil {
        return nil, fmt.Errorf("account not found: %w", err)
    }
    if row.MoiAPIKey == "" {
        return nil, fmt.Errorf("user not synced to moi-core")
    }
    return moi.New(endpoint, row.MoiAPIKey)
}
```

各业务模块通过 `caller.NewClient` 或封装的 `newCallerClient` 获取 client 后调用 moi-core。

### 2.2 共享数据模型（编译期依赖）

`moi-backend/go.mod` 通过 `replace` 指令引用本地 moi-core 子模块：

```go
replace github.com/matrixflow/moi-core/go-sdk => ../moi-core/go-sdk
replace github.com/matrixflow/moi-core/model  => ../moi-core/model
replace github.com/matrixflow/moi-core/saga   => ../moi-core/saga
```

| 模块 | 用途 |
|------|------|
| `moi-core/go-sdk` | HTTP SDK client，封装所有 moi-core API 调用 |
| `moi-core/model` | Proto 生成的共享数据结构（catalog/auth/mowl/permission/role/user/workspace 等） |
| `moi-core/saga` | 分布式事务组件（Saga 模式） |

两边共用同一套 Protobuf 数据定义，避免重复定义和序列化不一致。

### 2.3 Saga 分布式事务

用户注册、Workspace 创建等跨系统操作使用 `moi-core/saga` 做事务编排：

```
用户注册流程 (pkg/auth/saga_steps.go):
  Step 1: moi-backend DB 创建 account 记录
  Step 2: 通过 go-sdk 在 moi-core 创建 user + workspace
  Step 3: 通过 go-sdk 在 moi-core 初始化默认 catalog/database/volume
  失败 → saga 自动执行补偿步骤回滚
```

涉及 Saga 的模块：
- `pkg/auth/saga_steps.go` — 用户注册、workspace 创建
- `pkg/userperm/saga_steps.go` — 权限同步
- `pkg/saga/` — Saga helper 封装

## 3. 业务模块调用矩阵

| moi-backend 模块 | 调用的 moi-core 能力 | go-sdk 方法 |
|---|---|---|
| `pkg/auth` | 用户同步、workspace 创建、角色初始化 | `Users()`, `Workspaces()`, `Roles()` |
| `pkg/userperm` | 权限/角色同步到 moi-core | `Roles()`, `Permissions()` |
| `pkg/catalog` | 目录/文件夹、初始化默认 catalog/database/volume | `Catalogs()`, `Databases()`, `Volumes()` |
| `pkg/session` | Session、知识库、消息管理 | `KnowledgeBases()`, `VolumeFiles()` |
| `pkg/explore` | 检索与推理 | `Explore()` |
| `pkg/workflow` | 工作流 DSL 编排、任务调度、流式执行 | `Workflows()`, `Tasks()`, `WorkItems()` |
| `pkg/worker` | Worker 注册、导入导出任务 | `Worker()`, `WorkItems()` |
| `pkg/llm` | LLM 调用代理 | `LLM()` |

知识库 workflow 完成态治理由 backend 基于已持久化 source/job 绑定执行
`POST /newmoi/semantic-models/:model_id/source-jobs/reconcile`，边界见
[知识库 Workflow 治理接入手册](./knowledge-base-workflow-governance.md)。

## 4. 配置说明

### 4.1 moi-backend 侧配置

配置文件：`moi-backend/etc/service.yaml`（从 `service.example.yaml` 复制）

```yaml
# moi-core 连接配置
moicore:
  endpoint: "http://127.0.0.1:8081"   # moi-core catalog HTTP 地址
  apiKey: "<system_api_key>"           # System API Key，必须与 moi-core 的 MOI_SYSTEM_API_KEY 一致
```

环境变量覆盖（前缀 `MOI_`，嵌套用 `_` 分隔）：

```bash
export MOI_MOICORE_ENDPOINT=http://catalog:8081
export MOI_MOICORE_APIKEY=your_system_api_key
```

### 4.2 moi-core 侧配置

配置文件：`moi-core/catalog/etc/config-*.toml`（TOML 格式）

关键配置项：
- HTTP 监听端口（默认 8081）
- gRPC 监听端口（默认 8082）
- `MOI_SYSTEM_API_KEY` 环境变量 — 必须与 moi-backend 的 `moicore.apiKey` 一致

### 4.3 认证机制

moi-backend 与 moi-core 之间使用 **API Key 认证**：

1. **系统级 API Key**：moi-backend 配置中的 `moicore.apiKey`，用于系统初始化操作
2. **用户级 API Key**：每个用户在 moi-backend 的 `account` 表中有 `moi_api_key` 字段，注册时通过 Saga 同步创建。`caller.NewClient` 使用此 key 代表用户调用 moi-core

### 4.4 数据库依赖

两个服务使用同一个 MatrixOne 实例（端口 6001），但使用不同的 database：

| 服务 | Database |
|------|----------|
| moi-backend | `mocloud_meta` |
| moi-core | moi-core 自管理的 database |

## 5. 本地开发快速启动

```bash
# 1. 启动基础设施
make start-env && make wait-mo && make init-env

# 2. 启动 moi-core catalog（先于 moi-backend）
cd moi-core && make compose-up

# 3. 确认 moi-backend 配置中 moicore.endpoint 指向正确地址
# 4. 启动 moi-backend
cd moi-backend && make build && ./moi-backend
```

## 6. 常见问题

| 问题 | 原因 | 解决 |
|------|------|------|
| `user not synced to moi-core` | 用户 account 表的 `moi_api_key` 为空 | 检查注册 Saga 是否正常执行 |
| `caller: GetDB not registered` | auth 包未初始化 | 确保 `caller.RegisterGetDB` 在 init 阶段被调用 |
| moi-core 连接超时 | endpoint 配置错误或 catalog 未启动 | 检查 `moicore.endpoint` 配置，确认 catalog :8081 可达 |
| API Key 认证失败 | moi-backend 与 moi-core 的 system API key 不一致 | 确保两侧 `MOI_SYSTEM_API_KEY` 相同 |
