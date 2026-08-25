# Knowledge Explore A2A 设计

## 目标

`knowledge-explore` 是知识库模块的数据探索对话页。旧的表格回答、消息历史和引用预览链路已移除，页面直接消费通用 Agent A2A endpoint，并按 A2A 事件展示对话、推理过程、工具/状态时间线和最终回答。

## 架构

```
KnowledgeExplorePage
  ├─ AiChatSessionManager        # 会话列表、搜索、置顶、重命名、删除
  ├─ ChatComposer                # 模型、知识库范围、输入、发送/停止
  └─ A2A message view            # 用户问题、推理、计划、事件、产物、回答

hooks/
  useExploreSessionManager       # 会话 CRUD / 搜索 / 分页
  useExploreA2ARuntimeManager    # A2A 发送、停止、运行态订阅
  useExploreKnowledgeOptions     # 知识库范围选项
  useExploreModelOptions         # LLM 模型选项

services/
  exploreSessionStore            # 会话列表状态
  exploreA2ARuntimeStore         # task/context/stream/messages/runtime LRU
  exploreA2AStreamParser         # A2A response -> UI projection
```

## A2A 请求

发送消息使用 `@moi/shared-moi-api/agent` 的 `streamAgentA2AApi`：

- `agent_code: "explore"`
- `method: "message/stream"`
- `params.taskId/contextId/model/locale`
- `params.message.parts`: 用户问题文本
- `params.metadata`: workspace/session/semantic model/file/table 范围

停止生成使用 `sendAgentA2AApi`：

- `method: "tasks/cancel"`
- `params.id`: 当前 A2A task id

## 事件投影

`exploreA2AStreamParser` 只做协议投影，不写业务规则：

- `status-update` -> timeline、thinking、状态
- `artifact-update` -> answer、assistant delta、plan、artifacts
- `task` -> task/context/status/final
- JSON-RPC error -> failed projection

页面显示：

- thinking: 最近的推理片段
- plan: A2A 计划 artifact
- timeline: 最近的状态/产物事件
- artifacts: 非内部产物标签
- answer: final answer 或 streaming draft markdown

## 会话与配置

会话仍复用 `dialogSession` 的 CRUD API。发送前 `useSessionConfigPersistence` 将本轮选择写入 session config：

- `semantic_models: [{ semantic_model_id }]`
- `llm.model`
- `model`

从知识库卡片进入对话页时，页面只记录待发送的固定知识库范围，不立即创建会话。用户发送第一条消息时才创建固定会话，并用该会话继续发起 A2A stream；如果用户未发送消息就离开，不产生空会话。

Runtime 只保留当前浏览器会话内的 A2A 消息和运行态，不再拉取旧 message history，不再展示表格回答。

## 集成要求

宿主必须注入：

- `httpClient`
- `sseClient`
- `timezone`
- `workspaceId`
- `user`

模块声明必须包含：

```ts
requires: {
  httpClient: true,
  sseClient: true,
}
```
