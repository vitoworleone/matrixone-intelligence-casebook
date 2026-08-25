# Agent Prototype Resource Capabilities Roadmap

## 背景

`agent-runtime-a2a.md` 只负责一轮智能体运行：对话、流式进度、任务状态、取消、重新生成和反馈。前端原型里还有大量产品能力并不属于 agent-runtime 的公共运行时能力，例如 Skill/Tool/Knowledge Base 管理、文件上传解析、构建向导、模型配置和资源面板。

这些能力需要单独设计和落地，避免把资源控制面、数据处理管道和 UI 向导塞进 runtime 协议里。本文只描述“原型需要，但不应侵入 agent-runtime 后端抽象”的能力。

## 范围边界

### 属于本文

- Agent 配置管理：名称、描述、头像/图标、系统提示词、模型、运行配置、绑定关系。
- Agent 治理配置：工作流触发绑定、Agent Task Template、审批策略、运行限额、重试策略和资源可见性。
- Skill 管理：系统 Skill、自定义 Skill、版本、override、市场元数据、绑定配置。
- Tool 管理：系统工具、MCP、HTTP API、市场工具、启停、凭证引用、发现和同步。
- Knowledge Base 管理：知识库、文件、标签、有效期、版本、预览、分段、索引和检索配置。
- 文件处理：上传、外部来源导入、解析、分段、向量化、覆盖、重建、错误恢复。
- 模型管理：内置模型、自定义模型、自定义 API/HuggingFace onboarding、默认模型绑定。
- Connection/Credential 管理：外部 API、MCP、Drive、模型服务等凭证引用、测试和轮换。
- Feedback Review：反馈明细、统计、筛选和可追溯关联。
- Conversation/Message 管理：会话列表、归档、置顶、重命名、消息标注、引用、分享和导出。
- 构建向导：从用户意图生成 Agent 草稿、推荐 Skill/Tool/KB、保存为资源配置。
- 资源操作：长耗时资源导入/同步/解析的 operation 状态和重试。
- 前端资源面板迁移：从 prototype mock 切到 REST Resource API。

### 不属于本文

- A2A `message/send`、`message/stream`、`tasks/get`、`tasks/cancel`。
- 运行时 task snapshot、event stream、provider adapter。
- provider 原生工具执行环境，例如 Codex/Claude Code/Astra 的 shell、文件编辑、工作目录生命周期。
- Memory/Memoria 的外部 CRUD、权重管理、可见性配置或检索接口。

Memory/Memoria 不作为本文资源域，也不作为 agent-runtime 公共接口。若某个 provider 内部具备记忆能力，它只能隐藏在 provider 实现里；资源面和前端不能假设存在可配置、可查询或可调权的 Memory 对象。

## 总体原则

1. 资源管理走 REST API，不走 A2A DataPart command。
2. 资源能力是通用平台能力，不按 prototype 页面按钮命名后端接口。
3. Skill 是自然语言工作簿，不是可执行工具；绑定 Skill 不隐式授权 Tool。
4. Tool 是可执行资源，凭证、审批、RBAC、审计由平台控制，不交给 provider 私自管理。
5. KB 文件和分段是 Catalog/fileservice 资源，不是 runtime provider 的内部状态。
6. 构建向导可以生成建议，但落库必须调用显式 REST create/update/bind API。
7. 长耗时资源动作必须返回 operation，不能阻塞普通 CRUD 请求。
8. 每个 PR 只交付一个资源域或一个可验证闭环，避免和 runtime MVP 混在一起。
9. 原型页面里的动作需要映射到通用资源能力，而不是把页面状态或按钮名变成后端 API。
10. Memory/Memoria 不参与资源设计；不要为了对齐原型而引入公共 Memory abstraction。
11. 定时、cron、事件触发和任务编排复用 moi-core Workflow/Mowl 能力；Agent 资源只提供可被工作流调用的任务模板和绑定，不自建 scheduler。

智能体自动化任务、API 调用触发和 Dynamic Service 的详细设计见 [agent-automation-task-design.md](./agent-automation-task-design.md)。

## 实体元数据字典

本节是 agent resource control plane 的权威实体定义。后续 REST API、数据库表、前端 SDK 和 runtime `RuntimeManifest` 都必须从这里投影，不能在页面、A2A DataPart 或 provider adapter 里再发明第二套字段。

定义原则：

- `id` 是稳定主键，`version` 是资源版本，`snapshot_id` 是不可变快照版本；三者不能混用。
- workspace、resource、user、agent、task、message、snapshot、operation 等可寻址 ID 都是不透明 ID；Agent Resource 与 Runtime Read Model 的资源 ID 以 workspace 为唯一性作用域。ID 可以由服务端生成、调用方指定或作为 read-model filter 传入，但不能包含 `/` 或 `\`，也不能把层级路径、provider native run/session id 或外部文件路径塞进主键或查询过滤条件。
- `metadata_json` 只能保存非索引、非权限、非运行语义的扩展展示信息；核心业务字段必须显式建模。
- 任何 secret、token、OAuth code、API key 只能以 `credential_ref` 引用，不能进入 Agent/Skill/Tool/KB/RuntimeManifest 明文字段。资源元数据、operation result/error、模板变量和 schema 中的 `uri/url/href/*_uri/*_url` 字段如果带 userinfo 或 secret/token/API key query，也按敏感信息处理。
- provider 的外部 task/run/session/workspace id 只属于 runtime provider run，不属于资源元数据。
- Memory/Memoria 不是资源实体，也不是 Agent 的可配置元数据。

### 实体归属矩阵

| 实体 | Owner | 对外面 | 是否进入 RuntimeManifest | 说明 |
|---|---|---|---:|---|
| `AgentMetadata` | agentresource | REST `/agents` | 是，冻结最小快照 | Agent 的身份、展示、指令、runtime target 和治理引用 |
| `AgentSkillBinding` | agentresource | REST `/agents/:id/bindings` | 是 | Skill 绑定态、版本策略、配置和 provisioning 摘要 |
| `AgentToolBinding` | agentresource | REST `/agents/:id/bindings` | 是，仅 ToolGateway 模式 | 平台 Tool 授权和执行约束，不代表 provider native tool |
| `AgentKnowledgeBinding` | agentresource | REST `/agents/:id/bindings` | 是 | KB 角色、检索 profile 和引用范围 |
| `RuntimePolicyProfile` | agentresource | REST `/runtime-policy-profiles`；Agent 引用聚合 `/agents/:id/policies` | 是，冻结解析结果 | 并发、预算、trace、artifact、provider capability gate |
| `ApprovalPolicy` | agentresource | REST `/approval-policies` | 是，冻结引用和决策摘要 | 平台资源动作审批，不接管 provider 内部审批流 |
| `GuardrailPolicy` | agentresource | REST policy API | 是，冻结规则版本 | Post-MVP；runtime 负责执行和审计 |
| `AgentTaskTemplate` | agentresource | REST `/agent-task-templates` | 间接进入 | Workflow/Mowl 调用 Agent 的模板，不是 scheduler |
| `AgentWorkflowBinding` | agentresource + Workflow/Mowl | REST `/agent-workflow-bindings` | 任务 metadata 中保存关联 id | 将 Workflow 节点映射到 AgentTaskTemplate |
| `SkillSpec` / `SkillVersion` | agentresource | REST `/skills` | 是 | 自然语言工作簿，不是工具 |
| `ToolSpec` / `ToolSource` | agentresource | REST `/tools` | 是，仅授权摘要 | 平台可执行 Tool；执行必须走 ToolGateway |
| `KnowledgeBase` / `KnowledgeFile` / `KnowledgeSegment` | Catalog semantic model + agentresource compatibility | REST `/semantic-models`；兼容 `/knowledge-bases` | 是，冻结 ref 和版本 | 当前默认 KB 来源是 semantic model；旧 agent KB 元数据只做兼容回退 |
| `ModelConfig` | agentresource | REST `/model-configs` | 是，冻结 ref 和非敏感参数 | 连接、凭证和能力声明，不选择 runtime provider |
| `ProviderDescriptor` / `ProviderProfile` | provider registry + agentresource read model | REST `/agent-runtime-providers` | 是，冻结 capability 决策摘要 | runtime target、profile capability 和 config schema |
| `Connection` / `CredentialRef` | agentresource + secret store | REST `/connections` | 只冻结允许的 ref | 不向前端或 provider 暴露 secret 明文 |
| `Conversation` / `Message` | runtime canonical store + resource read model | REST read model + A2A runtime | task 创建时使用可见历史快照 | runtime 拥有消息追加和 conversation head；resource 管理列表/标注/分享 |
| `FeedbackReview` | runtime feedback store + resource read model | A2A submit + REST review | 否 | runtime 只 record-only 写入，统计筛选走 REST |
| `ResourceOperation` | agentresource | REST `/operations` | 否 | 长耗时资源动作状态，不是 A2A task |
| `RuntimeTask` / `RuntimeManifest` / `RuntimeEvent` | agent-runtime | A2A + REST read model | N/A | 运行态实体，详见 `agent-runtime-a2a.md` |

### 通用资源字段

除特别说明外，资源实体都继承以下字段：

| 字段 | 说明 |
|---|---|
| `id` | 全局唯一资源 id |
| `workspace_id` | workspace 隔离边界 |
| `name` / `description` | 人类可读名称和说明；不能作为权限或路由唯一依据 |
| `status` | `draft | active | disabled | archived`，具体实体可扩展；硬删除不是首批资源生命周期 |
| `visibility` | `private | workspace | shared | system` |
| `owner_user_id` / `maintainer_user_ids` | 资源负责人和维护者 |
| `created_by` / `updated_by` | 审计主体 |
| `created_at` / `updated_at` | 基础生命周期时间；归档由 `status=archived` 表达，专用 `archived_at/deleted_at` 按后续资源治理 PR 增加 |
| `version` | 资源当前版本，修改核心语义字段时递增 |
| `labels` / `tags` | 通用筛选标签；不承载页面布局语义 |
| `annotations` | 用户或系统注释，不能改变资源执行语义 |
| `source_type` / `source_ref` | 系统、市场、导入、工作流模板等来源和快照引用 |
| `metadata_json` | 非核心扩展字段，禁止保存 secret、权限规则、provider run id |

### AgentMetadata

`AgentMetadata` 是 Agent 资源的权威模型。它描述“这个 Agent 是什么、使用哪些资源、用哪个 runtime target 启动、受哪些治理策略约束”，不描述某一次运行的状态。

核心字段：

| 字段 | 说明 |
|---|---|
| `id` / `workspace_id` | Agent 主键和租户边界 |
| `name` / `description` | 展示和搜索用，不作为 provider prompt 的唯一来源 |
| `avatar_ref` / `icon` / `display_tags` / `category` / `sort_order` | 展示元数据，不进入 provider 私有状态 |
| `instruction.system_prompt` | Agent 基础指令；创建 runtime task 时冻结进 manifest |
| `instruction.behavior_rules` | 可选行为约束；必须是通用规则，不能硬编码行业 case |
| `instruction.output_contract_ref` | 可选输出契约引用；具体 schema 属于独立资源或 Skill output contract |
| `instruction.variables_schema` | prompt 变量 schema；用于 Builder/TaskTemplate 校验 |
| `runtime.provider` / `runtime.profile` | runtime facade target 选择，例如 `default/default`、`codex/default`、`astra/default`；`default/default` 当前由 agent-facade 路由到 `agent-runtime-v2` backend |
| `runtime.config_json` | provider/profile 允许的受控配置；禁止保存 provider run/session id 或 secret |
| `model_config_ref` | 模型配置引用；可以为空；非空时必须能在同 workspace 解析到 ModelConfig |
| `model_params_override` | 非敏感模型参数覆盖；ModelConfig schema 深度校验为后续能力 |
| `binding_summary` | Agent 定义自带的默认 Skill/Tool/KB 绑定摘要；workspace 级覆盖以绑定实体为准 |
| `policy_refs.runtime_policy_ref` | 并发、预算、trace、artifact、provider capability gate |
| `policy_refs.approval_policy_ref` | 平台资源动作审批策略 |
| `policy_refs.guardrail_policy_ref` | Post-MVP；输入/输出/工具 guardrail 策略 |
| `workflow_refs.default_task_template_ids` | 可被 Workflow/Mowl 调用的模板引用 |
| `lifecycle.draft_ref` / `published_version` | Builder 草稿和已发布版本 |
| `status` / `version` | 生命周期和版本 |

Agent `status` 至少支持 `draft | active | disabled | archived`。`disabled` 用于临时停用但保留配置和绑定，runtime admission 只能启动 active Agent。

MVP 最小可落字段：`id`、`workspace_id`、`name`、`description`、`avatar_ref`、`icon`、`display_tags`、`instruction.system_prompt`、`model_config_ref` 或默认模型标识、`runtime.provider`、`runtime.profile`、`runtime.config_json`、`status`、`version`、`created_at`、`updated_at`。Agent policy refs 已先支持聚合读写 runtime policy 引用；workflow refs、guardrail 和 model registry 可按后续 PR 补齐；字段语义现在必须固定。

Agent 不包含：

- Memory/Memoria 配置。
- 当前 conversation、message、task、trace 或 artifact 状态。
- provider external task/run/session id。
- Tool secret、模型 API key 或 OAuth token。
- 前端页面布局状态。

### Agent Binding Metadata

Agent 与 Skill/Tool/KB 的关系必须是一等实体，不允许只保存裸 id 数组。裸 id 数组只可作为 API 兼容简写。系统默认 Agent 定义保存在系统租户的普通 `agent_resource_agents` 表中；用户 workspace 勾选知识库或工具时，不复制系统 Agent，而是在用户 workspace 的 `agent_resource_agent_bindings` 中保存 `(workspace_id, agent_workspace_id, agent_id)` 维度的绑定记录。

`AgentSkillBinding`：

| 字段 | 说明 |
|---|---|
| `id` / `agent_id` / `skill_id` | 绑定主键和两端资源 |
| `skill_version` | 绑定的 Skill 版本 |
| `version_policy` | `pinned | latest_active`，默认 `pinned` |
| `enabled` / `priority` | 是否启用和装配顺序 |
| `config_json` | 通过 Skill `parameters_schema` 校验后的参数 |
| `resource_bindings` | Skill 需要的 KB/file/model 角色映射 |
| `resolved_requirements` | 需求解析结果；不能替代 Tool 授权 |
| `provisioning_refs` / `provisioning_status` | 工作流实例、索引任务等预置资源状态 |
| `created_at` / `updated_at` | 审计时间 |

`AgentToolBinding`：

| 字段 | 说明 |
|---|---|
| `id` / `agent_id` / `tool_id` | 绑定主键和两端资源 |
| `tool_version` / `version_policy` | Tool 版本和更新策略 |
| `enabled` / `priority` | 是否启用和展示/装配优先级 |
| `execution_scope` | `in_loop | preflight | postprocess`；`in_loop` 只适用于 ToolGateway 模式 |
| `allowed_actions` | Tool 内部可用动作白名单 |
| `side_effect_class_snapshot` | 绑定时冻结的副作用等级 |
| `approval_policy_ref` | 写操作或外部副作用审批策略 |
| `credential_scope_ref` | 允许该 Agent 使用的凭证范围引用 |
| `redaction_policy_ref` | 参数和结果脱敏策略 |
| `status` | `active | disabled | invalid` |

provider native tool 不属于 `AgentToolBinding`。Codex/Claude Code/Astra 等 provider 的原生 shell、文件编辑、MCP 或 Web 能力只能作为 provider profile capability 和 coarse policy 表达，不能伪装成 MOI ToolSpec。

`AgentKnowledgeBinding`：

| 字段 | 说明 |
|---|---|
| `id` / `agent_id` / `knowledge_base_id` | 绑定主键和两端资源 |
| `kb_version` / `version_policy` | KB 或检索视图版本 |
| `role` | `default | domain | reference | examples` 等自然语言角色 |
| `retrieval_profile_ref` | 默认检索配置引用 |
| `retrieval_overrides` | top_k、阈值等非敏感覆盖 |
| `citation_policy` | 是否要求引用和 locator |
| `scope_filters` | 文件标签、目录、时间范围等可见范围 |
| `enabled` / `status` | 生命周期 |

### Policy Metadata

`RuntimePolicyProfile` 是资源控制面对象，runtime 在 task admission 时解析并冻结为 `RuntimeManifest.runtime_policy`。

核心字段：

- `id`、`workspace_id`、`name`、`description`、`status`、`version`。
- `scope`: `workspace | role | agent | provider_profile`。
- `admission`: conversation/user/agent 并发、排队、去重和 overflow 策略。
- `budgets`: token、费用、wall time、step 数量。
- `retry`: 重试次数、退避、可重试错误分类。
- `data_policy`: 文件输入、外部网络、Catalog 访问、trace 级别。
- `artifact_policy`: 默认保留期、导出权限、删除策略。
- `provider_constraints`: 允许 provider/profile、必须能力、禁止能力。

`ApprovalPolicy`：

- `id`、`workspace_id`、`name`、`scope`、`status`、`version`。
- `rules`: resource type、action、side effect class、risk level、condition。
- `approvers`: user、role、group 或 workflow approval node。
- `timeout_policy`: 超时后拒绝、取消或升级。
- `audit_policy`: 审批输入、输出和脱敏策略。

`GuardrailPolicy` 是 Post-MVP 一等资源：

- `id`、`workspace_id`、`name`、`scope`、`status`、`version`。
- `input_rules`、`output_rules`、`tool_rules`。
- `actions`: `allow | redact | require_approval | block`。
- `explanation_policy` 和 `audit_level`。

### AgentTaskTemplate 和 AgentWorkflowBinding

Agent 不自建调度器。Workflow/Mowl 拥有 cron、event trigger、case state、retry 和 DAG 推进；Agent 侧只提供可调用模板和绑定。

`AgentTaskTemplate`：

| 字段 | 说明 |
|---|---|
| `id` / `workspace_id` / `agent_id` | 模板归属 |
| `name` / `description` | 模板说明 |
| `message_template` | 创建 runtime turn 的用户消息模板 |
| `input_schema` | Workflow 节点传入参数 schema |
| `default_context_refs` | 默认 KB/file/catalog/artifact 引用 |
| `runtime_policy_ref` | 调用时使用的 policy；非空时必须解析到同 workspace 的 `RuntimePolicyProfile` |
| `idempotency_policy` | 去重 key 生成和重复提交处理 |
| `output_contract` | text/artifact/status 输出契约 |
| `status` / `version` | 生命周期 |

`AgentWorkflowBinding`：

| 字段 | 说明 |
|---|---|
| `id` / `workspace_id` | 绑定主键 |
| `workflow_id` / `workflow_version_id` / `node_id` | Workflow/Mowl 侧节点定位 |
| `agent_task_template_id` | 被调用的 AgentTaskTemplate；必须解析到同 workspace 的模板 |
| `input_mapping` | workflow input 到 template input 的映射 |
| `output_mapping` | runtime task 输出到 workflow output 的映射 |
| `failure_mapping` | cancel、timeout、error 到 workflow case 的映射 |
| `status` / `version` | 生命周期 |

### Skill Metadata

`SkillSpec` 是可版本化的自然语言工作簿。

核心字段：

- `id`、`workspace_id`、`name`、`description`、`status`。
- `source_type`: `system | custom | market | workflow_template`。
- `source_ref`: 源模板、市场条目、workflow template snapshot。
- `category`、`icon_ref`、`tags`、`phase`、`market_metadata`。
- `routing_summary` 和可选 examples，用于推荐和轻量路由。
- `instruction_body`，用于 runtime `InstructionBundle`。
- `requirements`: 需要的 Skill、Tool、KB role、文件输入、模型能力。
- `requirements.skill_refs` / `requirements.tool_refs` 是资源需求声明；非空 ref 必须在同 workspace 解析，且不能包含路径分隔符。
- `parameters_schema`: 绑定级配置 schema。
- `output_contract`: 产物类型、引用要求和结构化输出约束。
- `version`、`created_at`、`updated_at`。

`SkillVersion` 保存每次语义变更的不可变副本；`SkillOverride` 保存 workspace 对 system/market Skill 的 patch，并记录 `base_skill_id`、`base_version` 和 `patch_json`。
Skill `status` 至少支持 `draft | active | disabled | archived`。统一执行入口只允许 active Skill，disabled 用于临时下线但保留版本历史和绑定关系。
Agent 创建 runtime task 时只冻结已绑定 active Skill 的 `routing_summary`、`instruction_body`、`requirements`、schema 和非敏感 metadata；当前 agent-runtime-v2 默认 backend 会把 Skill instruction/routing summary 作为 developer/capability prompt 消费。Skill requirements 只用于校验和提示，不替代 Agent Tool binding，也不授权 provider native tool。

### Tool Metadata

`ToolSpec` 是平台可执行资源，和 provider native tool 严格分开。

核心字段：

- `id`、`workspace_id`、`name`、`description`、`status`、`version`。
- `kind`: `system | mcp | http_api | operator | market | code`。
- `category`、`icon_ref`、`tags`、`phase`、`market_metadata`。
- `source_ref`: operator、MCP server、HTTP schema、market package 或 code bundle snapshot。
- `input_schema`、`output_schema`。
- `side_effect_class`: `read | write | external_effect`。
- `credential_ref`、`approval_policy_ref`、`redaction_policy_ref`；首批只保存受控 ref，禁止路径型 ref，存在性和策略解析留给 Credential/Approval 后续 PR。
- `sync_status`、`last_sync_at`、`last_sync_error`。

`ToolSource` 描述工具来源连接和同步策略；`ToolInstallation` 描述从市场或系统模板安装到 workspace 的安装记录。Tool 执行不读取这两个对象的 secret 明文，只读取经过 policy 授权的 `credential_ref`。

### Knowledge Metadata

`KnowledgeBase`：

- `id`、`workspace_id`、`name`、`description`、`status`、`version`。
- `source_type`、`catalog_asset_refs`、`default_retrieval_profile_ref`。
- `tags`、`owner_user_id`、`visibility`。
- `index_status`、`last_indexed_at`、`last_index_error`。

`KnowledgeFile`：

- `id`、`knowledge_base_id`、`workspace_id`、`name`、`mime_type`、`size_bytes`。
- `source_file_ref`、`source_type`、`source_ref`。
- `parse_status`、`index_status`、`parser_snapshot_id`。
- `enabled`、`tags`、`expiry_at`、`version`。

`KnowledgeFileVersion`：

- `id`、`knowledge_file_id`、`version`、`source_file_ref`、`digest`。
- `parser_snapshot_id`、`created_by`、`created_at`、`status`。

`KnowledgeSegment`：

- `id`、`knowledge_file_id`、`knowledge_base_id`、`kind`。
- `content_ref`、`preview`、`locator`、`embedding_ref`。
- `enabled`、`metadata_json`、`version`。

`KnowledgeRetrievalProfile`：

- `id`、`knowledge_base_id`、`search_mode`、`embedding_model_ref`、`rerank_model_ref`。
- `top_k`、`score_threshold`、`hybrid_weights`、`citation_policy`。

`FileSourceImport` 和 `IndexJob` 是异步资源操作的输入和状态对象，必须关联 `ResourceOperation`。

### Conversation、Message 和 Feedback Metadata

`Conversation`：

- `id`、`workspace_id`、`agent_id`、`title`、`pinned`、`archived`。
- `head_task_id`、`last_message_at`、`created_by`、`created_at`、`updated_at`。

`Message`：

- `id`、`conversation_id`、`workspace_id`、`role`、`parts`。
- `task_id`、`parent_message_id`、`created_by`、`created_at`。
- `parts` 可以包含 text/file/data refs，但不能包含 provider 私有路径或 secret。

runtime 拥有 conversation/message 的追加和 head 更新；resource API 只提供列表、筛选、标注、分享、导出等产品层操作。

`MessageAnnotation`：

- `id`、`message_id`、`workspace_id`、`kind`、`range`、`label`、`note`。
- `created_by`、`created_at`、`updated_at`。

`FeedbackReview`：

- `feedback_id`、`workspace_id`、`agent_id`、`conversation_id`、`task_id`、`message_id`。
- `rating`、`comment`、`labels`、`trace_id`、`artifact_id`。
- `created_by`、`created_at`。

### Model、Connection 和 Credential Metadata

`ModelConfig`：

- `id`、`workspace_id`、`provider_ref`、`model_name`、`source_kind`。
- `model_category`、`display_name`、`description`、`capabilities`。
- `parameters`、`parameter_schema`、`limits`、`connection_ref`、`credential_ref`。
- `test_status`、`last_test_error`、`workspace_default`、`status`。
- 非空 `connection_ref` 必须能在同 workspace 解析到 Connection；`credential_ref` 是敏感凭证引用，不在 ModelConfig 内展开。

`ProviderDescriptor`：

- `id`、`name`、`description`、`status`。
- `adapter_kind`: `agent_runtime_v2 | astra | codex | claude_code | custom`。
- `capabilities`: provider 默认能力摘要。
- `profiles`: 可用 ProviderProfile 列表。
- `config_schema`: provider 级配置 schema。
- `workspace_availability`: 当前 workspace 可用性、禁用原因和 credential requirement。

`ProviderProfile`：

- `id`、`provider_id`、`name`、`description`、`status`。
- `tool_mode`、`workspace_mode`、`permission_model`。
- `capability_overrides`。
- `runtime_config_schema`。
- `model_constraints` 和 `policy_constraints`。

Provider/Profile 是 Agent `runtime.provider/profile/config_json` 的配置来源。它只描述能力和 schema，不代表某个 task 已经被授权启动；task admission 仍由 runtime 重新计算 policy、RBAC、provider capability 和 manifest 交集。

`Connection`：

- `id`、`workspace_id`、`kind`、`endpoint`、`auth_type`。
- `capabilities`、`visibility`、`owner_user_id`、`status`。
- `credential_ref`、`config`、`labels`、`annotations`、`metadata`。
- `last_test_status`、`last_tested_at`、`last_test_error` 是只读测试摘要，只能由连接测试 operation 写入，不能由通用 CRUD 请求写入。

`CredentialRef`：

- `id`、`workspace_id`、`connection_id`、`secret_store_ref`。
- `scope`、`expires_at`、`rotation_policy`、`last_test_status`。

REST 响应只能返回 credential 元数据，不能返回 secret 明文。

### Builder 和 Operation Metadata

`AgentDraft`：

- `id`、`workspace_id`、`source_conversation_id`、`source_message_id`。
- `current_step`、`proposal_version`、`status`。
- `agent_patch`、`selected_skill_bindings`、`selected_tool_bindings`、`selected_kb_bindings`。
- `warnings`、`validation_result`、`created_by`、`created_at`、`updated_at`。

`AgentDraftEvent` 保存用户编辑、系统建议、校验结果和保存动作，用于审计和恢复草稿；它不是 runtime event。

`ResourceOperation`：

- `operation_id`、`workspace_id`、`type`、`resource_type`、`resource_id`。
- `status`、`progress`、`message`、`result_json`、`error_json`。
- `idempotency_key`、`created_by`、`created_at`、`updated_at`、`completed_at`。

## 原型能力拆分

| 原型能力 | 所属后端域 | MVP 状态 | 落地方式 |
|---|---|---:|---|
| Agent 列表、新建、编辑、删除 | Agent Resource API | Runtime MVP 需要最小版 | REST `/agents` |
| Agent 绑定 Skill/Tool/KB | Agent Resource API | Resource MVP 已落绑定读写和同 workspace 资源存在性校验 | REST `/agents/:id/bindings` |
| Agent 选择 ModelConfig | Agent Resource API | Resource MVP 已落同 workspace 引用存在性校验；参数 schema 校验 Post-MVP | REST `/agents`、`/model-configs` |
| Agent Task Template | Agent Workflow Integration | Resource MVP 已落元数据 CRUD；真实 Workflow/Mowl 调用执行链路 Post-MVP | REST `/agent-task-templates` |
| 工作流触发、cron、事件调度 | moi-core Workflow/Mowl | Post-MVP | Workflow schedule/event trigger + Agent task template binding |
| Agent 运行限额 | RuntimePolicyProfile Resource | Resource MVP 已落 profile CRUD + Agent refs | REST `/runtime-policy-profiles`、`/agents/:id/policies` |
| Agent 审批策略 | Agent Policy + Approval Policy | Post-MVP | REST `/approval-policies` |
| 对话历史列表 | Conversation Resource API | Runtime MVP 需要 | REST `/conversations` |
| 会话置顶、归档、重命名、删除 | Conversation Resource API | Post-MVP | REST `/conversations/:id` |
| 消息引用、标注、分享、导出 | Message Resource API | Post-MVP | REST `/messages/:id/*` |
| Runtime task 列表和历史筛选 | Runtime Task Read Model | Runtime MVP 需要最小版 | REST `/agent-runtime-tasks` + A2A `tasks/get` |
| 用户反馈明细和统计 | Feedback Review API | Resource MVP 已落明细和 rating 聚合 | REST `/feedback`、`/feedback/stats` |
| Skill 列表、详情、创建、编辑 | Skill Resource API | Resource MVP 已落最小版 | REST `/skills` |
| Skill 版本查询 | Skill Resource API | Resource MVP 已落最小版 | REST `/skills/:id/versions` |
| Skill 版本切换、系统 override | Skill Resource API | Post-MVP | activate/restore、override |
| Skill 分类、标签、市场元数据 | Skill Catalog API | Post-MVP | REST `/skills` |
| Tool 列表、详情、创建、编辑 | Tool Resource API | Resource MVP 已落最小版 | REST `/tools` |
| MCP/HTTP/市场工具发现 | Tool Discovery | Post-MVP | REST `/tools/discover` + operation |
| Tool 凭证配置 | Credential Resource API | Post-MVP | REST credential flow/resource |
| Tool 分类、标签、市场安装 | Tool Catalog API | Post-MVP | REST `/tools/install` |
| KB 列表、详情、创建、编辑 | Semantic Model API | 当前默认知识库能力已落 | REST `/semantic-models` |
| Agent KB 绑定管理 | Agent Resource API | Resource MVP 已落基础绑定校验 | REST `/agents/:id/bindings` |
| KB 文件上传/外部导入/覆盖 | File + KB Pipeline | Post-MVP | fileservice + REST `/knowledge-bases/:id/files` |
| KB 文件解析、分段、索引 | KB Pipeline | Post-MVP | async operation worker |
| KB 分段搜索、编辑、启停 | KB Segment API | Post-MVP | REST `/segments` |
| KB 检索配置和命中测试 | KB Retrieval Profile | Post-MVP | REST `/knowledge-bases/:id/retrieval-profile` |
| 模型配置列表、详情、创建、编辑 | Model Registry | Resource MVP 已落最小版 | REST `/model-configs` |
| 可用模型聚合和 workspace 默认模型 | Model Registry | Post-MVP | REST `/models`、`/model-defaults` |
| 自定义模型/API onboarding | Model Registry + Credential | Post-MVP | REST model config + credential |
| 外部连接元数据列表、详情、创建、编辑 | Connection Resource | Resource MVP 已落最小版 | REST `/connections` |
| 外部连接凭证保存、轮换和测试 | Credential/Test Resource | Post-MVP | REST `/credentials`、`/connections/:id/test` |
| 构建向导 | Builder Orchestration | Post-MVP | draft/proposal + REST save |
| 资源导入/同步状态 | Resource Operation | schema 可先定义 | REST `/operations/:id` |

## 后端模块

建议落在 `moi-core/catalog/pkg/agentruntime/resource` 或后续独立 `agentresource` 包中。不要放进 A2A handler 或 runtime backend adapter。

```
moi-core/catalog/pkg/agentruntime/
  rest/
    agents.go
    conversations.go
    providers.go
    skills.go
    tools.go
    knowledge_bases.go
    model_configs.go
    connections.go
    feedback.go
    runtime_tasks.go
    operations.go
    builder.go
    policies.go
    workflow_bindings.go
    file_sources.go
  resource/
    agent_service.go
    agent_policy_service.go
    agent_task_template_service.go
    workflow_binding_service.go
    conversation_service.go
    message_service.go
    skill_service.go
    tool_service.go
    knowledge_base_service.go
    file_source_service.go
    model_registry_service.go
    connection_service.go
    feedback_review_service.go
    runtime_task_read_service.go
    operation_service.go
    builder_service.go
    repository.go
    validation.go
```

`agent-runtime` 可以依赖这些资源服务读取 AgentMetadata、绑定快照和 provider capability，但资源服务不能依赖 runtime backend。

## Resource API 草案

### Agent

`AgentMetadata` 的权威字段见“实体元数据字典”。本节只给 REST API 的 MVP 落地形态。

MVP 最小落库字段：

- `id`
- `workspace_id`
- `name`
- `description`
- `avatar_ref`
- `icon`
- `display_tags`
- `instruction.system_prompt`
- `model_config_ref` 或 workspace 默认模型标识
- `runtime.provider`
- `runtime.profile`
- `runtime.config_json`
- `status`
- `version`
- `created_at`
- `updated_at`

Endpoints：

| Method | Path | 说明 |
|---|---|---|
| `GET` | `/api/v1/workspaces/:ws/agents` | 列表、分页、搜索 |
| `POST` | `/api/v1/workspaces/:ws/agents` | 创建 Agent |
| `GET` | `/api/v1/workspaces/:ws/agents/:id` | 获取详情 |
| `PATCH` | `/api/v1/workspaces/:ws/agents/:id` | 更新最小配置 |
| `GET` | `/api/v1/workspaces/:ws/agents/:id/bindings` | 查询绑定资源 |
| `PATCH` | `/api/v1/workspaces/:ws/agents/:id/bindings` | 更新绑定摘要 |
| `GET` | `/api/v1/workspaces/:ws/agent-runtime-providers` | 查询可选 runtime provider/profile |
| `GET` | `/api/v1/workspaces/:ws/agent-runtime-providers/:provider_id/profiles/:profile_id` | 查询单个 runtime provider profile |

MVP 先支持 Agent 最小 CRUD、绑定摘要读写、runtime provider/profile discovery，以及 Skill/Tool/KB 同 workspace 存在性校验；delete/archive、绑定级配置、角色映射和版本策略随后续绑定资源表补齐。

`runtime.provider/profile/config_json` 是资源配置，不是绕过 facade 的 backend 直连承诺。前端可以选择 `default/default`、`astra/default`、`codex/default`、`claude-code/default`；runtime 在创建 `RuntimeManifest` 时保留这个选择，并读取冻结后的 Agent snapshot、绑定和 policy 解析结果。provider start 必须经过 agent-facade router；当前默认 backend implementation 是 `agent-runtime-v2`，真实 provider-native adapter 后续只替换后端路由，不改变前端 A2A 协议。provider 是否能原生表达这些配置由 adapter 做能力映射，不由前端直接感知 provider 私有字段。

### Agent Policy、Task Template 和 Workflow Binding

Agent 的运行限额和审批配置属于资源控制面。定时、cron、事件触发和多步骤编排不在 agent-runtime 内自建 scheduler，而是复用 moi-core Workflow/Mowl。Agent 侧只提供 `AgentTaskTemplate`，供 Workflow 节点、schedule 或 event trigger 在合适时机调用，并由 agent-runtime service 创建一次 A2A task。

核心资源：

- `AgentTaskTemplate`: 目标 Agent、message template、input schema、默认文件/KB 引用、runtime policy ref、idempotency policy。
- `AgentWorkflowBinding`: workflow id/version、node id、AgentTaskTemplate ref、输入映射、输出 artifact 映射。
- `RuntimePolicyProfile`: max steps、step timeout、retry count、concurrency limit、queue policy。
- `ApprovalPolicy`: 需要审批的资源动作、审批人/角色、超时策略。

Endpoints：

| Method | Path | 说明 |
|---|---|---|
| `GET` | `/api/v1/workspaces/:ws/agents/:id/policies` | 查询 Agent 策略引用和可解析 runtime policy 摘要 |
| `PUT` | `/api/v1/workspaces/:ws/agents/:id/policies` | 更新 Agent 策略引用；当前校验并解析 runtime_policy_ref |
| `GET` | `/api/v1/workspaces/:ws/runtime-policy-profiles` | 查询可复用运行策略 profile |
| `POST` | `/api/v1/workspaces/:ws/runtime-policy-profiles` | 创建运行策略 profile |
| `GET` | `/api/v1/workspaces/:ws/runtime-policy-profiles/:id` | 查询单个运行策略 profile |
| `PATCH` | `/api/v1/workspaces/:ws/runtime-policy-profiles/:id` | 更新运行策略 profile |
| `GET` | `/api/v1/workspaces/:ws/agent-task-templates` | 查询 Agent Task Template |
| `POST` | `/api/v1/workspaces/:ws/agent-task-templates` | 创建可被工作流调用的 Agent 任务模板 |
| `GET` | `/api/v1/workspaces/:ws/agent-task-templates/:id` | 查询单个 Agent Task Template |
| `PATCH` | `/api/v1/workspaces/:ws/agent-task-templates/:id` | 更新模板输入、策略和生命周期 |
| `POST` | `/api/v1/workspaces/:ws/agent-task-templates/:id/validate` | Post-MVP：校验模板能否创建 runtime task |
| `GET` | `/api/v1/workspaces/:ws/agent-workflow-bindings` | 查询 Agent 与 Workflow 节点绑定 |
| `POST` | `/api/v1/workspaces/:ws/agent-workflow-bindings` | 创建 Workflow 节点到 AgentTaskTemplate 的绑定 |
| `GET` | `/api/v1/workspaces/:ws/agent-workflow-bindings/:id` | 查询单个 Workflow 节点绑定 |
| `PATCH` | `/api/v1/workspaces/:ws/agent-workflow-bindings/:id` | 更新绑定映射和生命周期 |
| `GET` | `/api/v1/workspaces/:ws/approval-policies` | Post-MVP：查询审批策略 |
| `POST` | `/api/v1/workspaces/:ws/approval-policies` | Post-MVP：创建审批策略 |

约束：

- cron、schedule、event source、case retry、workflow case state 归 Workflow/Mowl 管理，不复制一套 `agent_resource_schedules`。
- Workflow 节点只调用 AgentTaskTemplate 创建 runtime task，不能直接操作 provider 会话或 provider task id。
- AgentTaskTemplate 冻结为 RuntimeManifest 时必须重新校验 AgentMetadata、binding、RBAC、policy、provider capability。
- Workflow case id、workflow version id、node id 作为 runtime task metadata 和 trace 关联字段保存，方便从 workflow case 追踪到 A2A task。
- AgentTaskTemplate 和 AgentWorkflowBinding 只能保存模板、同 workspace 资源引用和映射表达式；不得保存 provider run/session id、secret/token/password，也不得保存 Workflow/Mowl 的 cron、case state、retry counter 或 DAG 进度。
- `idempotency_policy.mode=workflow_case` 是默认模式，由 Workflow/Mowl case/node invocation 决定幂等 key；`custom` 模式必须提供 `key_template`，并且只用于 runtime task admission 去重，不替代 Workflow/Mowl 的 case 幂等。
- 审批策略保护平台 Tool、文件、KB、外部连接器等资源动作；provider native tool 的内部审批只能通过 coarse policy 和 audit 表达。
- runtime task 完成后把 text/artifact/status 写回 Workflow output mapping；Workflow 决定后续节点和重试，agent-runtime 不推进 workflow case。
- 首批已落 `AgentTaskTemplate`、`AgentWorkflowBinding` 和 `RuntimePolicyProfile` 的 create/list/get/patch、schema、store 和 router；也已落 Agent 专用 `/agents/:id/policies` 的 policy refs 聚合读写。`AgentTaskTemplate.runtime_policy_ref` 和 `AgentWorkflowBinding.agent_task_template_id` 已做同 workspace 引用校验；list filter 中的 `agent_id`、`scope_ref`、`workflow_id`、`workflow_version_id`、`node_id` 和 `agent_task_template_id` 也按不透明 ID 处理，不能包含路径分隔符。RuntimePolicyProfile 只保存 admission、budget、retry、data/artifact policy、provider constraints 等非敏感策略元数据；task admission 冻结和执行策略仍由 agent-runtime service 负责。Approval、Guardrail、validate/dry-run、Workflow/Mowl 调用执行链路是后续 PR。

### Skill

Skill 是可版本化的自然语言工作簿。

`SkillSpec`、`SkillVersion` 和 `SkillOverride` 的权威字段见“实体元数据字典”。REST 响应可以按列表/详情场景裁剪字段，但不能改变这些字段的语义。

Endpoints：

| Method | Path | 说明 |
|---|---|---|
| `GET` | `/api/v1/workspaces/:ws/skills` | 列表、筛选系统/自定义/市场 |
| `POST` | `/api/v1/workspaces/:ws/skills` | 创建自定义 Skill |
| `GET` | `/api/v1/workspaces/:ws/skills/:id` | 详情 |
| `PATCH` | `/api/v1/workspaces/:ws/skills/:id` | 更新并生成新版本 |
| `GET` | `/api/v1/workspaces/:ws/skills/:id/versions` | 版本列表 |
| `POST` | `/api/v1/workspaces/:ws/skills/:id/execute` | 统一 Skill 执行入口；创建 runtime task admission |
| `POST` | `/api/v1/workspaces/:ws/skills/:id/override` | Post-MVP：基于系统 Skill 创建 override |
| `POST` | `/api/v1/workspaces/:ws/skills/:id/install` | Post-MVP：从市场或系统源安装到 workspace |
| `PATCH` | `/api/v1/workspaces/:ws/agents/:agent_id/skills/:skill_id` | Post-MVP：更新 Agent 与 Skill 的绑定配置 |
| `POST` | `/api/v1/workspaces/:ws/skills/:id/versions/:version/activate` | Post-MVP：切换当前使用版本 |
| `POST` | `/api/v1/workspaces/:ws/skills/:id/versions/:version/restore` | Post-MVP：基于历史版本创建新版本 |

约束：

- Skill requirements 只能声明需要哪些资源类型，不直接授权 Tool；`skill_refs` 和 `tool_refs` 在创建/更新 Skill 时按同 workspace 做存在性校验。
- Skill 绑定时仍必须校验依赖 DAG，禁止循环依赖；DAG 展开和绑定级 provisioning 属于 Post-MVP。
- Skill version 切换不影响历史 runtime task。
- version activate/restore 只影响后续 RuntimeManifest，不能改写历史 task 的 frozen SkillSpec。
- `category`、`tags`、`phase`、`pipeline_ref` 是通用 catalog 元数据，用于筛选、推荐和安装，不承载页面布局语义。
- `skills/:id/execute` 不是 provider 私有接口。它是 Catalog 控制面的统一执行入口，必须经过 API key middleware、workspace 路径和 RBAC wrapper；handler 从认证上下文取得 `user_id` 和原始 API key，传给 `SkillExecutionSubmitter`。
- `skills/:id/execute` 的 `agent_id` 是可选上下文；非空时必须能在同 workspace 解析到 active AgentMetadata，且该 Skill 必须已绑定到该 Agent，然后把该 Agent 的 `runtime.provider/profile` 冻结到 runtime task/manifest，不能退化成开发态 stub 或作为跨 workspace task 关联入口。
- 外部 runtime 远程调用统一入口时也使用 Catalog 用户 API key。这样 runtime 读取文件、KB、ToolGateway 或写回 task/operation 时仍沿用同一个 workspace/user 权限模型，不需要 provider 保存或解释 MOI RBAC。原始 API key 只存在于当前请求上下文；admission 持久化时只保存 `caller.user_id`、`caller.api_key_id/prefix/scopes` 和 `caller.user_api_key_ref=request_context` 这类非敏感引用。
- 显式 `idempotency_key` 的去重作用域必须包含 `workspace_id + agent_id + skill_id + idempotency_key`，落库时保存该作用域派生出的固定长度 key，而不是原始 client key。同一 Agent 重试复用已有 task；不同 Agent 即使执行同一个 Skill 且传入相同 client key，也必须创建不同 runtime task，避免跨 Agent 复用 manifest、runtime profile 或权限上下文。
- 执行入口只冻结 SkillSpec/SkillVersion、可选 Agent runtime target、执行输入、resource refs 和调用身份，随后提交给 agent-runtime task admission。`parts`、`metadata`、`variables`、`parameters` 和 `resource_refs.config` 禁止 secret/token/provider run/session 字段；`resource_refs.id` 禁止路径型 ref，`resource_refs.uri` 禁止 userinfo 和 token/secret query。Skill 不变成 Tool，Skill execute 不绕过 Agent/Tool/KB 绑定校验。
- 首批实现 list/get/create/update、version table、execute endpoint 和默认本地 runtime admission submitter。默认 submitter 只创建标准 runtime task、manifest、turn snapshot、initial event 和 provider_start outbox；不会执行 provider，也不会持久化原始用户 API key。manifest/snapshot/outbox 可以保存非敏感 caller ref，便于后续外部 runtime worker 重新经 Catalog 权限链路解析能力。
- 真实外部 runtime submitter worker、market install、override、activate/restore、AgentSkillBinding DAG 校验后续 PR。

### Tool

Tool 是平台可执行资源，和 provider native tool 分开。

`ToolSpec`、`ToolSource` 和 `ToolInstallation` 的权威字段见“实体元数据字典”。REST 响应可以按列表/详情场景裁剪字段，但 ToolGateway、审批和凭证校验必须基于同一份 ToolSpec 语义。

`GET /tools` 和 `GET /tools/:id` 对系统工具额外返回 `display_name` 与 `display_description`，用于前端展示本地化名称和描述；`name` 与 `description` 仍是稳定契约字段，不能被本地化展示值覆盖，也不能用于执行授权语义。展示字段只在 `source_ref.type=platform_tool` 的系统工具存在本地化文案时返回。Locale 解析先看 `Accept-Language`，再看 `Content-Language`，按 header 中语言范围顺序选择第一个支持的 `zh` 或 `en`；没有支持语言时默认使用英文（en-US）。

Endpoints：

| Method | Path | 说明 |
|---|---|---|
| `GET` | `/api/v1/workspaces/:ws/tools` | 工具列表 |
| `POST` | `/api/v1/workspaces/:ws/tools` | 创建 HTTP/API/自定义工具 |
| `GET` | `/api/v1/workspaces/:ws/tools/:id` | 详情 |
| `PATCH` | `/api/v1/workspaces/:ws/tools/:id` | 更新配置 |
| `POST` | `/api/v1/workspaces/:ws/tools/discover` | Post-MVP：MCP/市场/连接器发现，返回 operation |
| `POST` | `/api/v1/workspaces/:ws/tools/install` | Post-MVP：安装市场工具或连接器模板 |
| `POST` | `/api/v1/workspaces/:ws/tools/:id/sync` | Post-MVP：同步源 schema，返回 operation |
| `PATCH` | `/api/v1/workspaces/:ws/tools/:id/state` | Post-MVP：启用、停用或归档工具 |
| `PATCH` | `/api/v1/workspaces/:ws/agents/:agent_id/tools/:tool_id` | Post-MVP：更新 Agent 与 Tool 的绑定配置 |

Resource MVP 已落 `ToolSpec` 的 list/get/create/update、基础 schema/store/router、分类/标签/side-effect 元数据，以及 credential/policy ref 的基础边界校验。系统工具实现集中在 `moi-core/agent-tools`，Catalog 将它们种到 MOI system workspace 的普通 `agent_resource_tools` 行，并通过同一套只读 overlay 暴露为 `source_ref.type=platform_tool` 的 active `ToolSpec`；runtime manifest 会冻结已经绑定且 runtime gateway executable 的工具。Agent create/update/binding 阶段会拒绝没有 runtime gateway executable implementation 的工具。HTTP 工具已有 `http_api` ToolProvider v1：要求 `source_ref.uri` 为无敏感信息的 http/https URL，`source_ref.config.method` 显式声明 HTTP 方法，请求参数作为 JSON body 发送；带 `credential_ref` 的 HTTP 工具在 credential resolver 落地前会显式调用失败，不会匿名兜底。MCP discover、MCP/code/custom adapter、审批执行拦截、凭证 onboarding 和 AgentToolBinding 配置进入后续 PR。

当前已暴露的系统工具分两类：

- Knowledge bundle：`find_rag_files`、`search_rag_chunks`、`read_parsed_markdown`、`search_parsed_markdown`，全部为 read side-effect，通过 `PlatformToolGateway` 调用现有 RAG/semantic knowledge 能力。选择知识库后，runtime manifest 基于 Agent/会话的知识库绑定把相关知识库工具投影进本轮可用工具集合。
- Workflow bundle：`list_workflow_apps`、`get_workflow_app`、`start_workflow_execution`、`list_workflow_executions`、`get_workflow_execution`、`get_workflow_execution_result`、`cancel_workflow_execution`、`retry_workflow_execution`、`list_file_workflow_executions`。这些工具通过 `WorkflowAppToolRunner` 适配现有 `workflowapp.Service`，使用当前 runtime request 的 workspace、user 和 user API key；启动、取消、重试等写类工具标记为 `side_effect_class=write`，并经过 agent-facade 的 ToolGateway。

市场安装只复制或引用工具模板，不等于授权执行。工具执行仍必须经过 RBAC、凭证可见性、side-effect class 和审批策略校验。

### Knowledge Base

KB 是 Catalog 数据的逻辑视图，运行时只通过授权检索工具消费摘要和引用。当前智能体资源链路默认使用现有知识库能力：前端和 Agent 绑定里的 `knowledge_base_ids` 引用 `/semantic-models` 返回的 semantic model id；AgentService 校验和 runtime manifest 生成时优先通过 tenant `semantic_models` 解析该 id，并投影为 runtime 可见的 `KnowledgeBase` snapshot。agent-facade 默认注入 `PlatformToolGateway` 给 agent-runtime-v2 backend，知识库检索工具会按 manifest snapshot 和本轮 metadata scope 调用现有 RAG/semantic knowledge adapter，返回脱敏后的 entry、chunk、markdown 内容和 catalog asset refs。旧的 agent resource `/knowledge-bases` 元数据端点和 `agent_resource_knowledge_bases` 表只作为兼容层保留，semantic model 未命中时才回退读取。

核心资源的权威字段见“实体元数据字典”：

- `KnowledgeBase`
- `KnowledgeFile`
- `KnowledgeSegment`
- `KnowledgeFileVersion`
- `KnowledgeRetrievalProfile`
- `FileSourceImport`
- `IndexJob`

Endpoints：

| Method | Path | 说明 |
|---|---|---|
| `GET` | `/api/v1/workspaces/:ws/semantic-models` | 当前默认 KB 列表 |
| `POST` | `/api/v1/workspaces/:ws/semantic-models` | 当前默认 KB 创建 |
| `GET` | `/api/v1/workspaces/:ws/semantic-models/:id` | 当前默认 KB 详情 |
| `PUT` | `/api/v1/workspaces/:ws/semantic-models/:id` | 当前默认 KB 更新 |
| `GET` | `/api/v1/workspaces/:ws/semantic-models/:id/entries` | 语义条目列表 |
| `POST` | `/api/v1/workspaces/:ws/semantic-models/:id/entries` | 新增语义条目 |
| `PUT` | `/api/v1/workspaces/:ws/semantic-models/:id/entries/:entry_id` | 更新语义条目 |
| `DELETE` | `/api/v1/workspaces/:ws/semantic-models/:id/entries/:entry_id` | 删除语义条目 |
| `GET` | `/api/v1/workspaces/:ws/knowledge-bases` | 兼容：旧 agent KB 元数据列表 |
| `POST` | `/api/v1/workspaces/:ws/knowledge-bases` | 兼容：旧 agent KB 元数据创建 |
| `GET` | `/api/v1/workspaces/:ws/knowledge-bases/:id` | 兼容：旧 agent KB 元数据详情 |
| `PATCH` | `/api/v1/workspaces/:ws/knowledge-bases/:id` | 兼容：旧 agent KB 元数据更新 |
| `POST` | `/api/v1/workspaces/:ws/knowledge-bases/:id/files` | Post-MVP：绑定已上传文件并触发解析 |
| `GET` | `/api/v1/workspaces/:ws/knowledge-bases/:id/files` | Post-MVP：文件列表 |
| `GET` | `/api/v1/workspaces/:ws/knowledge-bases/:id/files/:file_id` | Post-MVP：文件详情、预览元数据、解析状态 |
| `POST` | `/api/v1/workspaces/:ws/knowledge-bases/:id/files/:file_id/replace` | Post-MVP：覆盖文件并生成新版本，返回 operation |
| `GET` | `/api/v1/workspaces/:ws/knowledge-bases/:id/files/:file_id/versions` | Post-MVP：文件版本列表 |
| `POST` | `/api/v1/workspaces/:ws/knowledge-bases/:id/files/:file_id/versions/:version/activate` | Post-MVP：切换检索使用的文件版本 |
| `POST` | `/api/v1/workspaces/:ws/knowledge-bases/:id/files/:file_id/versions/:version/restore` | Post-MVP：基于历史文件版本重建新版本 |
| `GET` | `/api/v1/workspaces/:ws/knowledge-bases/:id/files/:file_id/segments` | Post-MVP：分段列表 |
| `PATCH` | `/api/v1/workspaces/:ws/knowledge-bases/:id/segments/:segment_id` | Post-MVP：分段编辑/启停 |
| `POST` | `/api/v1/workspaces/:ws/knowledge-bases/:id/segments/reorder` | Post-MVP：调整人工分段顺序 |
| `POST` | `/api/v1/workspaces/:ws/knowledge-bases/:id/files/batch` | Post-MVP：批量启停、删除、重建或打标签，返回 per-item result |
| `GET` | `/api/v1/workspaces/:ws/knowledge-bases/:id/retrieval-profile` | Post-MVP：查询检索配置 |
| `PUT` | `/api/v1/workspaces/:ws/knowledge-bases/:id/retrieval-profile` | Post-MVP：更新检索配置 |
| `POST` | `/api/v1/workspaces/:ws/knowledge-bases/:id/retrieval-test` | Post-MVP：检索命中测试 |
| `POST` | `/api/v1/workspaces/:ws/knowledge-bases/:id/reindex` | Post-MVP：重建索引，返回 operation |

MVP 已落 `KnowledgeBase` 元数据 CRUD 和 `agent_resource_knowledge_bases` 表，但它不是当前智能体前端的默认知识库来源。智能体页面必须复用 `/semantic-models` 和 `/semantic-models/:id/entries`；不能把语义配置写入 agent KB metadata。运行时 snapshot 会把 semantic model 的 `tables`/`files` 投影为 `catalog_asset_refs`，metadata 带 `resource_kind=semantic_model` 和 `semantic_model_id`。对于 semantic model 关联的文件范围，运行时以 `knowledge_base_sources` 中未 removed 且 `status='succeeded'`、`effective_enabled=true` 的 KB file source 为治理检索范围；同一文件已有非 removed source row 时，pending、failed、disabled 或 expired 状态不能通过 legacy 兼容绕过。显式 `semantic_models.files.file_ids` 中没有 source row 的旧文件仍可作为 `governance_mode=legacy_compat` 范围检索。运行时工具执行时不返回静态 KB 摘要，而是通过 `find_rag_files`、`search_rag_chunks`、`read_parsed_markdown`、`search_parsed_markdown` 读取已授权 scope 内的真实检索结果，不写行业规则或自然语言词典。旧 `agent_resource_knowledge_bases` 可保存 `source_type`、`catalog_asset_refs`、`default_retrieval_profile_ref`、`tags`、`owner_user_id`、`visibility`、`index_status` 等通用字段，但只作为兼容读写和 semantic model 未命中时的回退。

文件上传链路：

1. 前端上传文件到 fileservice，得到 `moi://files/...`。
2. 前端调用 KB file endpoint，把文件引用保存为 KB 文件。
3. KB service 创建 parse/index operation。
4. Worker 解析文件、生成分段、写入索引。
5. 前端轮询 operation 或文件状态。

这条链路不属于 agent-runtime。运行时只能使用已授权、已索引的 KB 检索工具，或消费本轮已上传文件引用。本轮对话附件必须进入 `turn_input_snapshot.parts` 的 A2A `FilePart`；当前 `agent-runtime-v2` 内部 backend 会把这些 `moi://files/...` 引用作为附件列表注入本轮 `UserMessage`，保证运行时能看到引用，但这不等价于声明 provider-native 文件输入或自动写入 KB。

文件来源：

- `local_upload`: 前端先上传到 fileservice，再绑定到 KB。
- `catalog_resource`: 选择 MOI Catalog 中已有的数据表、文档、资产或工作流产物，保存 catalog resource ref。
- `external_drive`: Google Drive、S3、OSS、MinIO 等外部来源导入，保存 source ref 和 credential ref。
- `connector_export`: 从平台数据连接器或工作流产物导入。

检索配置需要抽象为 `KnowledgeRetrievalProfile`，不要把具体页面控件暴露为 API。推荐字段包括：

- `search_mode`: `vector | keyword | hybrid`
- `embedding_model_ref`
- `rerank_model_ref`
- `top_k`
- `score_threshold`
- `hybrid_weights`
- `citation_policy`

文件预览、分段编辑和检索测试只读取 KB/file/index 资源，不允许绕过 fileservice 直接给 provider 暴露原始文件系统路径。

批量操作要求：

- `batch` 请求必须带 idempotency key 和操作类型，例如 `delete | enable | disable | tag | reindex`。
- 响应必须返回每个 item 的 `status/result/error`，允许部分失败。
- 删除默认软删除；已被历史 RuntimeManifest 引用的文件版本必须保留可追溯记录。
- 结构化表预览和只读 SQL/query 属于 `catalog_resource` 或 `knowledge_file` 的 preview/query action，必须走 RBAC 和只读语句校验，不能交给 provider native tool 直接查询。

### Conversation 和 Message Resource

Conversation/Message 是产品体验资源，用于历史管理和消息级操作。它们不替代 A2A message/task，也不改变 provider 已完成的运行结果。

核心资源的权威字段见“实体元数据字典”：

- `Conversation`: workspace、agent、title、pinned、archived、last_message_at、head_task_id。
- `Message`: conversation、role、parts、created_at、task_id。
- `MessageAnnotation`: quote、hidden、strikethrough、user label、review note。
- `MessageExport`: export format、scope、operation id、artifact ref。
- `ShareLink`: resource type、resource id、permission、expiry。

Endpoints：

| Method | Path | 说明 |
|---|---|---|
| `GET` | `/api/v1/workspaces/:ws/conversations` | 会话列表，支持 agent、时间、归档、置顶筛选 |
| `POST` | `/api/v1/workspaces/:ws/conversations` | 创建会话壳或从 Agent 打开新会话 |
| `GET` | `/api/v1/workspaces/:ws/conversations/:id` | 会话详情 |
| `PATCH` | `/api/v1/workspaces/:ws/conversations/:id` | 重命名、置顶、归档 |
| `GET` | `/api/v1/workspaces/:ws/conversations/:id/messages` | 消息列表 |
| `PATCH` | `/api/v1/workspaces/:ws/messages/:id/annotations` | Post-MVP：更新消息标注 |
| `POST` | `/api/v1/workspaces/:ws/messages/:id/share` | Post-MVP：创建分享链接 |
| `POST` | `/api/v1/workspaces/:ws/conversations/:id/export` | Post-MVP：导出会话，返回 operation |

约束：

- 消息复制可以完全在前端完成；只有分享、导出、标注、归档等需要持久化的动作进入 Resource API。
- 首批已落 create/list/get/patch 和 message list；message list 对不存在的 conversation 返回 404，避免把权限/资源缺失误投影为空历史；DELETE、annotation、share、export 是 Post-MVP。
- REST conversation/message read model 默认按 workspace RBAC 读取；当前 Agent 页面带 `agent_id` 时必须做归属校验，list 只返回该 Agent 的会话，get/patch/message list 对跨 Agent conversation 返回 404。`agent_id` 不能包含路径分隔符。
- conversation id 和 message id 按 workspace 作用域唯一。重复创建同一 conversation 或重复追加同一 message id 必须按幂等重试返回已有记录；同一个 message id 不能被追加到另一个 conversation。
- Message `parts` 是 runtime/provider 可写入的自由结构，REST message list 必须套用和 runtime task read model 相同的可见字段投影：递归脱敏 secret/token/API key、provider run/session/external task id、带 userinfo 或敏感 query 的 URI，只返回前端可展示内容。
- MessageAnnotation 不能修改原始 A2A 事件，只能作为展示层 overlay。
- Conversation 删除默认软删除，保留 task/audit 的可追溯性。

### Runtime Task Read Model

任务控制仍走 A2A `tasks/get` 和 `tasks/cancel`；重新生成/重跑目标态走 `moi.run.control`，在 controls PR 单独开放。任务列表、历史筛选、任务面板和 workflow case 关联查询是读模型，应该通过 REST 暴露，避免把分页/筛选语义塞进 A2A。

核心字段：

- `task_id`
- `workspace_id`
- `agent_id`
- `conversation_id`
- `workflow_case_id`
- `workflow_version_id`
- `workflow_node_id`
- `state`
- `manifest_id`
- `provider_ref`
- `started_at`
- `completed_at`
- `input_summary`
- `summary`
- `artifact_refs`

Endpoints：

| Method | Path | 说明 |
|---|---|---|
| `GET` | `/api/v1/workspaces/:ws/agent-runtime-tasks` | 任务列表，MVP 支持 agent、agent workspace、context、state、分页筛选；`agent_id`、`agent_workspace_id` 和 `context_id` 不能包含路径分隔符；workflow case 和时间筛选 Post-MVP |
| `GET` | `/api/v1/workspaces/:ws/agent-runtime-tasks/:task_id` | 任务详情读模型；可带 `agent_id` 和 `agent_workspace_id` 做当前 Agent scope 校验；控制仍跳转 A2A `tasks/get/cancel` |
| `GET` | `/api/v1/workspaces/:ws/agent-runtime-tasks/:task_id/events` | 任务事件列表，支持 `after_seq` 增量查询和可选 `agent_id` / `agent_workspace_id` scope 校验 |
| `GET` | `/api/v1/workspaces/:ws/agent-runtime-manifests/:manifest_id` | 读取冻结 RuntimeManifest，支持可选 `agent_id` / `agent_workspace_id` scope 校验 |
| `GET` | `/api/v1/workspaces/:ws/agent-runtime-turn-snapshots/:snapshot_id` | 读取本轮输入或输出快照，支持可选 `agent_id` / `agent_workspace_id` scope 校验 |

MVP 已落地 `agent_runtime_tasks/events/manifests/turn_snapshots` 的 REST 只读投影。它只能读取 runtime task store，不直接查 provider，不启动或取消 task，不推进 conversation head。REST 投影默认只返回 runtime service 可见字段：不返回 provider run/session/external task id；`payload/body/error` 等自由 JSON 字段在出响应前递归脱敏；URI 字段如果带 userinfo 或 secret/token/API key query 也必须整体脱敏；`visible=false` 的事件不进入前端列表。provider snapshot 只能作为补充字段写入 runtime store，且必须先通过同一投影规则过滤。

Task list/detail 会直接返回 `agent_workspace_id`、`turn_input_snapshot_id`、`turn_output_snapshot_id`、`turn_output_committed` 和 `attempt.snapshot_id`，前端用这些不透明 ID 调 `/agent-runtime-turn-snapshots/:snapshot_id` 读取输入/输出快照；不能按命名规则自行拼接 snapshot id。`attempt.*` 仍表示 facade turn input 关系，输出快照只表达当前 task 的可展示结果是否已 committed。Task list/detail 返回脱敏 `input_summary` 作为任务列表标题/目标的轻量来源；脱敏 `summary` 和 `artifact_refs` 只表示输出预览与产物引用；task detail 额外返回脱敏 `output_parts`，详情/对话页可直接回放最终消息。Task 和 Manifest read model 也会投影当前 manifest 的 `tools` 与 `knowledge_bases` 展示摘要，前端展示本轮可用工具/KB 时使用这些脱敏摘要，不需要解析 `body.tools` 或 provider 私有字段。同一 workspace 视角下允许 `system/<agent_id>` 与 `<workspace>/<agent_id>` 同时存在；Agent 页面和 scoped 调用必须携带 `agent_workspace_id`，read model 使用 RuntimeManifest 中冻结的 `agent_workspace_id` 做精确匹配。

### Feedback Review

反馈提交是 runtime 行为，可以通过 A2A `moi.feedback.submit` record-only 写入。反馈查看、筛选和统计是资源读模型。

`FeedbackReview` 的权威字段见“实体元数据字典”。本节只定义 REST review/read model endpoint。

Endpoints：

| Method | Path | 说明 |
|---|---|---|
| `GET` | `/api/v1/workspaces/:ws/feedback` | 反馈明细，MVP 支持 agent、agent workspace、task、message、rating、分页筛选 |
| `GET` | `/api/v1/workspaces/:ws/feedback/stats` | 反馈聚合统计，MVP 支持 agent、agent workspace、task、message、rating 过滤，并按 rating 汇总 |

约束：

- Feedback Review 不调用 provider，也不触发后续 prompt 或工具调用。
- 反馈可关联 trace/span/message/artifact，但导出时必须按 workspace privacy policy 脱敏。
- Feedback canonical store 必须在写入时再次校验 `rating`、`intent=record_only` 和 payload 安全性；即使入口已经经过 A2A admission，也不能允许内部 submitter/provider 绕过校验写入 secret/token/API key 或 provider run/session id。
- MVP 已落地 `agent_runtime_feedbacks` 的 record-only 存储、明细列表和 rating 聚合统计；trace/span 关联和 message annotation join 后续独立 PR。

### Model Registry

原型中的模型选择需要从 mock 切到模型资源。

`ModelConfig` 的权威字段见“实体元数据字典”。本节只定义模型列表、自定义模型配置和默认模型 endpoint。

Endpoints：

| Method | Path | 说明 |
|---|---|---|
| `GET` | `/api/v1/workspaces/:ws/models` | Post-MVP：可用模型列表 |
| `GET` | `/api/v1/workspaces/:ws/model-configs` | 自定义模型配置列表 |
| `POST` | `/api/v1/workspaces/:ws/model-configs` | 创建自定义模型配置 |
| `GET` | `/api/v1/workspaces/:ws/model-configs/:id` | 查询自定义模型配置 |
| `PATCH` | `/api/v1/workspaces/:ws/model-configs/:id` | 更新 |
| `POST` | `/api/v1/workspaces/:ws/model-configs/:id/test` | Post-MVP：连通性测试 |
| `PUT` | `/api/v1/workspaces/:ws/model-defaults/:category` | Post-MVP：设置 workspace 默认模型 |

Resource MVP 已落 `ModelConfig` 元数据 CRUD 和 `agent_resource_model_configs` 表，只保存 `source_kind`、`model_category`、`provider_ref`、`model_name`、`connection_ref`、`credential_ref`、非敏感参数、能力摘要和 limits。ModelConfig 创建/更新时，非空 `connection_ref` 已按 route workspace 做存在性校验，非空 `credential_ref` 已做路径型 ref 拦截；Agent 创建/更新时，非空 `model_config_ref` 也已按 route workspace 做存在性校验，避免跨 workspace 引用。创建 runtime task 时，RuntimeSnapshotResolver 会把 active chat ModelConfig 解析为 manifest `model_config`，并合并 Agent `params_override`；agent-runtime-v2 默认 backend 会消费 `default_model`、`max_tokens`、`max_turns`、`temperature`、`service_tier`、`capabilities`、`supports_image_detail_original`、`input_modalities`、`limits.context_window`、`limits.max_context_window`、`limits.effective_context_window_percent`、`limits.auto_compact_token_limit`、`limits.truncation_policy`、`metadata.wire_api`、`metadata.base_instructions`、`metadata.model_messages`、`metadata.service_tiers`、`metadata.web_search_tool_type`、`metadata.supports_search_tool`、`metadata.apply_patch_tool_type`、`metadata.experimental_supported_tools`、`metadata.supports_reasoning_summaries`、`metadata.supported_reasoning_levels`、`metadata.default_reasoning_level`、`metadata.default_reasoning_summary`、`metadata.support_verbosity` 和 `metadata.default_verbosity`。`metadata.wire_api` 是模型网关协议声明，只允许 `responses` 或 `chat`；缺省按 `responses` 处理，不做运行时探测或自动降级。`capabilities` 按 exact string 匹配 `parallel_tool_calls` 和 `supports_image_detail_original`，不做 trim 或别名匹配；`input_modalities` 使用 Codex 对等的 `text` / `image` 值控制历史图片是否进入模型请求。当 `metadata.base_instructions` 或 `metadata.model_messages.instructions_template` 显式存在时，runtime-v2 按 Codex 对等逻辑把模型指令放入 Responses `instructions`，并把 Agent `system_prompt` 保留为 developer 指令；`model_messages` 使用 Codex 的 `{{ personality }}` 占位符替换规则。`metadata.web_search_tool_type` 接受 Codex 对等的 `text` / `text_and_image`，用于 hosted web search 工具的 `search_content_types`；`metadata.supports_search_tool=true` 才允许 runtime-v2 暴露 `tool_search` 并把大 gateway 工具集延迟加载，否则 gateway 工具按直接暴露处理；`service_tier` 只在 `metadata.service_tiers[].id` 精确包含该值时传给模型请求；`metadata.apply_patch_tool_type` 只接受 Codex 对等的 `freeform`，但不越过具体 runtime profile 对默认环境工具的禁用；`metadata.experimental_supported_tools` 使用 exact tool name 开启实验工具，例如 `test_sync_tool`。`ModelContextWindow` 按 Codex 对等逻辑从 `context_window` 或 `max_context_window` 乘 `effective_context_window_percent` 推导，缺省 percent 为 95，并只用于上下文 token accounting，不作为 run 终止预算。auto-compact 阈值按 Codex 对等逻辑从 `context_window` 或 `max_context_window` 的 90% 推导，显式 `auto_compact_token_limit` 存在时按 context 90% clamp；未配置 compact limits 时不启用自动 compact；`limits.truncation_policy` 使用 Codex 对等的 `{mode:"bytes"|"tokens",limit:n}` 结构控制模型可见工具输出预算，未配置时使用 bytes/10000。Reasoning 默认值仅在 `supports_reasoning_summaries=true` 时应用，verbosity 默认值仅在 `support_verbosity=true` 时应用；显式本轮 `llm_reasoning_effort` 不会被模型默认值覆盖。它不保存 API key、endpoint secret、provider session/run id，也不做真实模型探测或连通性测试。

MVP Agent 可以只使用默认模型或空 `model_config_ref`。可用模型聚合、workspace 默认模型、自定义模型 API onboarding、参数 schema 深度校验和连通性测试进入 Post-MVP。

模型 onboarding 只管理模型连接和能力声明，不做 provider runtime 的实现选择。Agent runtime provider 可以读取 `ModelConfigRef`，但不能要求前端理解 provider 私有模型参数。

### Provider Registry

Provider/Profile 是 Agent runtime target 的配置来源。它描述 provider 能力、profile、config schema 和 workspace 可用性，但不代表某次 task 已经被授权启动。

`ProviderDescriptor` 和 `ProviderProfile` 的权威字段见“实体元数据字典”。本节只定义只读 discovery endpoint；provider adapter 的启动、取消、attach、resume 能力仍归 runtime 文档。

Endpoints：

| Method | Path | 说明 |
|---|---|---|
| `GET` | `/api/v1/workspaces/:ws/agent-runtime-providers` | 查询当前 workspace 可用 runtime provider/profile |
| `GET` | `/api/v1/workspaces/:ws/agent-runtime-providers/:provider_id/profiles/:profile_id` | 查询 profile capability 和 runtime config schema |

约束：

- Provider discovery 是配置时能力发现，不是 task admission 结果。
- Agent create/update 当前必须校验 `runtime.provider/profile` 在 registry 中存在，并拒绝 secret、provider run/session id 等敏感配置。`runtime.config_json` 的 provider schema 深度校验依赖各 provider 提供稳定 schema，作为 Provider Registry 后续 PR 落地；在此之前只能保存非敏感、受控配置。
- task 创建前 runtime 仍必须重新计算 provider capability、RBAC、policy、manifest 和 credential refs 的交集。
- provider workspace availability 可以解释“为什么不可选”，但不能泄漏 provider 私有 endpoint、secret 或运行实例 id。

### Connection 和 Credential

Connection 是可被 Tool、ModelConfig、FileSourceImport、Workflow 节点复用的外部服务连接配置；Credential 保存敏感引用，不把 secret 暴露给前端、prompt 或 provider 私有状态。

核心资源的权威字段见“实体元数据字典”：

- `Connection`: kind、endpoint、auth_type、capabilities、status、owner、visibility。
- `CredentialRef`: secret store ref、scope、expires_at、rotation policy、last_test_status。
- `ConnectionTest`: test operation id、result、error、latency。

支持的 `kind` 至少包括：

- `model_api`
- `huggingface`
- `mcp_server`
- `http_api`
- `external_drive`
- `object_storage`
- `database`
- `webhook`

Endpoints：

| Method | Path | 说明 |
|---|---|---|
| `GET` | `/api/v1/workspaces/:ws/connections` | 连接列表 |
| `POST` | `/api/v1/workspaces/:ws/connections` | 创建连接配置，不直接包含 secret 明文 |
| `GET` | `/api/v1/workspaces/:ws/connections/:id` | 连接详情 |
| `PATCH` | `/api/v1/workspaces/:ws/connections/:id` | 更新非敏感配置 |
| `POST` | `/api/v1/workspaces/:ws/connections/:id/credentials` | Post-MVP：创建或轮换凭证，返回 credential ref |
| `POST` | `/api/v1/workspaces/:ws/connections/:id/test` | Post-MVP：连通性测试，返回 operation 或同步结果 |

Resource MVP 已落 `Connection` 元数据 CRUD 和 `agent_resource_connections` 表。MVP 只接受 `name`、`description`、`status`、`kind`、`endpoint_uri`、`auth_type`、`credential_ref`、`capabilities`、`owner_user_id`、`visibility`、`config`、`labels`、`annotations`、`metadata` 等非敏感字段；`credential_ref` 禁止路径型 ref，`endpoint_uri` 禁止 userinfo 和 token/secret query，`config/metadata` 禁止 secret、provider run id、external run id 和 session id。

ModelConfig 等上层资源只保存 `connection_ref`，并在资源面校验同 workspace 可见性；不会复制 Connection endpoint、credential 或 provider 私有连接状态。

Credential 创建/轮换、OAuth callback/device flow、连通性测试和 `last_test_*` 写入是 Post-MVP。当前 CRUD 可以返回已有只读测试摘要，但不能由前端直接写入测试结果。

约束：

- `credential_ref` 只能引用服务端 secret store，响应中不得返回 token、API key、OAuth code。
- OAuth callback、device code polling、API key 保存由 credential service 完成，业务资源只持有引用。
- provider 只能消费经过 RuntimeManifest 和 policy 允许的 credential ref；provider native 模式默认不下发 MOI secret。

### Builder

构建向导不是 runtime 协议。它是资源草稿生成器。

建议模型：

```go
type AgentDraft struct {
    ID              string
    WorkspaceID     string
    SourceConversationID string
    CurrentStep     string
    Name            string
    Description     string
    AvatarRef       string
    SystemPrompt    string
    ModelConfigRef  string
    RunPolicyRef    string
    SuggestedSkills []SkillSuggestion
    SuggestedTools  []ToolSuggestion
    SuggestedKBs    []KBSuggestion
    SelectedSkills  []DraftBindingSelection
    SelectedTools   []DraftBindingSelection
    SelectedKBs     []DraftBindingSelection
    Warnings         []DraftWarning
    SourceMessageID  string
    ProposalVersion  int
    Status          string // draft | saved | discarded
}
```

Endpoints：

| Method | Path | 说明 |
|---|---|---|
| `POST` | `/api/v1/workspaces/:ws/agent-drafts` | Post-MVP：从自然语言创建 Agent 草稿 |
| `GET` | `/api/v1/workspaces/:ws/agent-drafts/:id` | Post-MVP：获取草稿 |
| `PATCH` | `/api/v1/workspaces/:ws/agent-drafts/:id` | Post-MVP：用户编辑草稿 |
| `POST` | `/api/v1/workspaces/:ws/agent-drafts/:id/propose` | Post-MVP：基于草稿和用户输入生成配置建议 |
| `POST` | `/api/v1/workspaces/:ws/agent-drafts/:id/validate` | Post-MVP：校验草稿完整性、权限和绑定资源 |
| `POST` | `/api/v1/workspaces/:ws/agent-drafts/:id/save` | Post-MVP：保存为 Agent + bindings |

约束：

- Builder 只能生成建议，不直接修改已发布 Agent。
- 保存时必须调用 Agent/Skill/Tool/KB 资源服务重新校验。
- Builder 不能给 Skill 隐式授权 Tool。
- Builder 输出可由 LLM 辅助，但必须经过 schema validation 和 RBAC。
- SourceConversationID 只用于把草稿关联回用户发起的对话，不是 provider session id；CurrentStep 只表达领域步骤，例如 `profile | instructions | resources | review`，不能绑定具体页面组件。
- `propose` 可以调用 LLM 或 deterministic scaffold；PR-R15 先落 deterministic scaffold，LLM proposal 后续独立 PR。
- `AgentDraft` 是构建向导的权威状态；A2A `moi.agent.build_step` 只是对话中的交互投影，不能成为第二套 draft store。
- build response 进入 resource builder service 更新 draft；最后保存仍调用 REST Agent/Binding API。

## Resource Operation

长耗时资源动作统一返回 operation：

```json
{
  "operation_id": "op_123",
  "type": "kb_file_parse",
  "resource_type": "knowledge_file",
  "resource_id": "kf_123",
  "status": "accepted",
  "progress": 0,
  "message": "queued",
  "created_at": "2026-05-13T00:00:00Z"
}
```

状态：

- `accepted`
- `running`
- `succeeded`
- `failed`
- `canceled`

Endpoints：

| Method | Path | 说明 |
|---|---|---|
| `GET` | `/api/v1/workspaces/:ws/operations/:id` | 查询状态 |
| `POST` | `/api/v1/workspaces/:ws/operations/:id/cancel` | 尝试取消 |
| `GET` | `/api/v1/workspaces/:ws/operations` | 列表和筛选 |

Operation 不等同于 runtime task，不进入 A2A Task，不推进 conversation head。

批量资源操作可以复用 Operation envelope，但 `result_json` 必须包含 per-item result，不能只返回整体成功或失败。

MVP 已落 create/list/get/cancel。operation 是资源处理进度，不暴露 provider run/session id。REST 响应中的 `result/error` 也必须做和 runtime read model 一致的递归脱敏，避免历史数据或内部 worker 写入的敏感 URI、secret 或 provider 私有字段泄露给前端。

## 数据表建议

MVP/首批依赖表：

- `agent_resource_agents`
- `agent_resource_operations`
- `agent_resource_skills`
- `agent_resource_skill_versions`
- `agent_resource_tools`
- `agent_resource_knowledge_bases`
- `agent_resource_model_configs`
- `agent_resource_connections`
- `agent_resource_runtime_policy_profiles`
- `agent_resource_agent_task_templates`
- `agent_resource_agent_workflow_bindings`
- `agent_runtime_conversations`
- `agent_runtime_messages`
- `agent_runtime_tasks`
- `agent_runtime_events`
- `agent_runtime_outbox`
- `agent_runtime_manifests`
- `agent_runtime_turn_snapshots`
- `agent_runtime_feedbacks`

这些表是首批落地可以复用的 runtime/catalog store。`agent_resource_*` 是资源控制面表，系统租户和普通租户使用同一套表结构；`agent_runtime_*` 是租户库运行态表。MVP 已先落 `agent_resource_agents`、`agent_resource_agent_bindings`、`agent_resource_operations`、`agent_resource_skills`、`agent_resource_tools`、`agent_resource_knowledge_bases`、`agent_resource_model_configs`、`agent_resource_connections`、`agent_resource_runtime_policy_profiles`、`agent_resource_agent_task_templates`、`agent_resource_agent_workflow_bindings`，以及 runtime `conversations/messages/tasks/events/outbox/manifests/turn_snapshots/feedbacks` 状态骨架。agentresource 只读取 runtime canonical store 生成任务列表、历史筛选和反馈关联视图，不拥有 conversation head、message event 或 task 状态机。

后续资源表：

- `agent_resource_agent_policies`
- `agent_resource_guardrail_policies`
- `agent_resource_message_annotations`
- `agent_resource_message_exports`
- `agent_resource_feedback_reviews`
- `agent_resource_credentials`
- `agent_resource_connection_tests`
- `agent_resource_skill_overrides`
- `agent_resource_tool_sources`
- `agent_resource_tool_installations`
- `agent_resource_knowledge_files`
- `agent_resource_knowledge_file_versions`
- `agent_resource_knowledge_segments`
- `agent_resource_knowledge_retrieval_profiles`
- `agent_resource_file_source_imports`
- `agent_resource_index_jobs`
- `agent_resource_model_defaults`
- `agent_resource_provider_profiles`
- `agent_resource_provider_workspace_availability`
- `agent_resource_drafts`
- `agent_resource_draft_events`
- `agent_resource_approval_policies`
- `agent_resource_share_links`

命名上使用 `agent_resource_*` 表达资源所有权。agent-runtime service 只能拥有 conversation、message、task、event、manifest、snapshot、feedback 等运行态表，不能承担 Agent/Skill/Tool/KB/Model/Connection 的资源 CRUD。

## RBAC

资源权限应该按域拆分：

- `agent.read`
- `agent.write`
- `agent.delete`
- `agent.policy.read`
- `agent.policy.write`
- `agent.task_template.read`
- `agent.task_template.write`
- `agent.workflow_binding.read`
- `agent.workflow_binding.write`
- `approval_policy.read`
- `approval_policy.write`
- `guardrail_policy.read`
- `guardrail_policy.write`
- `runtime_task.read`
- `conversation.read`
- `conversation.write`
- `message.annotation.write`
- `message.export`
- `feedback.read`
- `feedback.stats`
- `skill.read`
- `skill.write`
- `tool.read`
- `tool.write`
- `tool.install`
- `tool.credential.manage`
- `connection.read`
- `connection.write`
- `credential.write`
- `kb.read`
- `kb.write`
- `kb.segment.write`
- `kb.retrieval.write`
- `file_source.import`
- `model.read`
- `model.write`
- `model.default.write`
- `provider.read`
- `provider.profile.read`
- `operation.read`
- `operation.cancel`

运行时控制权限仍属于 runtime 文档，例如 `agent.run`、`task.cancel`、`feedback.submit`。当前 Catalog 路由层先以 `PERM_LLM_INVOKE` 保护具体 Agent 的 A2A JSON-RPC POST 和 Skill execute；AgentCard discovery、provider discovery、conversation、runtime read model 和 feedback review 仍是 read 权限。Workflow schedule/case 权限沿用 moi-core Workflow/Mowl 权限体系；本文只新增 AgentTaskTemplate 和 WorkflowBinding 的资源权限。

新增 API 经过 local-service 时，需要同步配置 RBAC 权限。当前仓库不包含 `mocloud-services/src/local-service/pkg/models/privs_init.go`，本 PR 先在 Catalog router 固定权限声明；local-service 侧按下表登记同一批 path，避免网关放行/拦截与 Catalog wrapper 不一致。

| Path group | Methods | Catalog permission | 说明 |
|---|---|---|---|
| `/api/v1/workspaces/:id/agents` | `GET`、`GET /:agent_id`、`GET /:agent_id/bindings`、`GET /:agent_id/policies` | `PERM_WORKSPACE_READ` | Agent 元数据和绑定/策略只读 |
| `/api/v1/workspaces/:id/agents` | `POST`、`PATCH /:agent_id`、`PATCH /:agent_id/bindings`、`PUT /:agent_id/policies` | `PERM_WORKSPACE_UPDATE` | Agent 元数据、绑定和策略修改 |
| `/api/v1/workspaces/:id/agent-runtime-providers` | `GET` | `PERM_WORKSPACE_READ` | provider/profile capability discovery |
| `/api/v1/workspaces/:id/skills` | `GET`、`GET /:skill_id`、`GET /:skill_id/versions` | `PERM_WORKSPACE_READ` | Skill 元数据只读 |
| `/api/v1/workspaces/:id/skills` | `POST`、`PATCH /:skill_id` | `PERM_WORKSPACE_UPDATE` | Skill 创建和更新 |
| `/api/v1/workspaces/:id/skills/:skill_id/execute` | `POST` | `PERM_LLM_INVOKE` | 统一 Skill 执行 admission，会创建 runtime task |
| `/api/v1/workspaces/:id/tools` | `GET`、`GET /:tool_id` | `PERM_WORKSPACE_READ` | Tool 元数据只读 |
| `/api/v1/workspaces/:id/tools` | `POST`、`PATCH /:tool_id` | `PERM_WORKSPACE_UPDATE` | Tool 元数据修改 |
| `/api/v1/workspaces/:id/knowledge-bases` | `GET`、`GET /:knowledge_base_id` | `PERM_KNOWLEDGE_BASE_READ` | KB 元数据只读 |
| `/api/v1/workspaces/:id/knowledge-bases` | `POST` | `PERM_KNOWLEDGE_BASE_CREATE` | KB 创建 |
| `/api/v1/workspaces/:id/knowledge-bases/:knowledge_base_id` | `PATCH` | `PERM_KNOWLEDGE_BASE_UPDATE` | KB 元数据更新 |
| `/api/v1/workspaces/:id/model-configs`、`/connections`、`/runtime-policy-profiles`、`/conversations`、`/agent-task-templates`、`/agent-workflow-bindings`、`/operations`、`/agent-runtime/*`、`/feedback*` | `GET` | `PERM_WORKSPACE_READ` | 非 KB 资源只读、runtime read model、feedback review |
| `/api/v1/workspaces/:id/model-configs`、`/connections`、`/runtime-policy-profiles`、`/conversations`、`/agent-task-templates`、`/agent-workflow-bindings` | `POST`、`PATCH` | `PERM_WORKSPACE_UPDATE` | 非 KB 控制面资源创建/更新 |
| `/api/v1/workspaces/:id/operations/:operation_id/cancel` | `POST` | `PERM_WORKSPACE_UPDATE` | 资源 operation 取消，不是 A2A task cancel |
| `/api/v1/workspaces/:id/agents/:agent_id/.well-known/agent-card.json` | `GET` | `PERM_WORKSPACE_READ` | A2A AgentCard discovery |
| `/api/v1/workspaces/:id/agents/:agent_id/a2a` | `POST` | `PERM_LLM_INVOKE` | A2A JSON-RPC message/task/feedback 运行时入口 |

## 落地计划

### PR-R0：文档和 API envelope

Scope：

- 新增本文档。
- 定义 REST response envelope、pagination、error code、operation schema。
- 在进入 SDK 或跨模块调用前，把本节实体字典固化到 `moi-core/proto/` 并通过 `make proto` 生成共享类型；当前 Catalog 内部实现可以先使用局部类型验证 store/handler，但不能让 go-sdk/python-sdk 再手写第二套结构。
- 文档索引更新。

Tests：

- `git diff --check`
- `make proto` / proto 生成校验（当 PR-R0 拆出 proto contract 时）。
- JSON schema 加载测试，如后续新增 schema 文件。

### PR-R1：Agent Resource MVP

Scope：

- `agentresource.AgentMetadata` 一等模型、规范化和校验。
- `agent_resource_agents` system table、upgrade offset 和 SQL repository。
- REST `/api/v1/workspaces/:id/agents` create/list/get/patch。
- Agent binding summary 字段可持久化，但 Skill/Tool/KB 资源本体不在本 PR 落地。
- Agent avatar/icon/display metadata 持久化。

不包含：

- Skill/Tool/KB 完整 CRUD。
- 自定义模型。
- KB 文件处理。
- provider capability 查询。
- conversation list/get。
- AgentTaskTemplate/WorkflowBinding/ApprovalPolicy。
- delete/archive operation worker；归档可通过后续 status update 或独立 endpoint 增加。

Tests：

- Agent service/repository unit tests。
- REST handler contract tests。
- router registration tests。
- schema/upgrade tests。

### PR-R2：Provider Discovery 和 Binding Read Model MVP

Scope：

- REST `/api/v1/workspaces/:id/agent-runtime-providers` 查询可选 runtime provider/profile。
- provider/profile 返回 `runtime_available`、`adapter_status`、`tool_mode`、streaming/cancel/A2A profile 等能力摘要；当前静态 registry 暴露默认可用 `default/default` facade target 和 `facade_routed` 的 `astra/default`。`agent-runtime-v2` 是内部 backend implementation，不作为 provider/profile 暴露。
- Agent binding detail/read model：REST `/agents/:id/bindings` 返回 Skill/Tool/KB 绑定摘要、资源状态、资源所属 workspace 和 provider 兼容性 warning。
- Agent binding 最小更新：PATCH `/agents/:id/bindings` 更新 `AgentBindingSummary`；当 `:id` 是系统租户 Agent 且请求 workspace 是用户 workspace 时，写入 `agent_resource_agent_bindings`，不复制系统 Agent 定义。
- 绑定最小 contract 校验：provider/profile 必须存在；`tool_ids`/`tool_refs` 只允许绑定到 `tool_mode=gateway` 的 profile；资源 id 和 workspace id 不允许路径分隔符；裸 id 默认解析到绑定所属 workspace，跨 workspace 资源必须使用带 `workspace_id` 的 ref。

不包含：

- provider native tool 接入。
- `runtime_available=false` 的 planned provider 不启动真实 run；`facade_routed` provider 可以创建 task，但当前仍由 agent-facade 路由到 agent-runtime-v2 默认 backend。
- 启动或探测真实 runtime backend。
- 绑定级配置、角色映射、版本策略和依赖 DAG 校验。

Tests：

- provider discovery unit/handler/router tests。
- binding projection/update/validation tests。

### PR-R3：Conversation 和 Runtime Task Read Model MVP

Scope：

- Conversation list/get。
- Conversation create/rename/archive/pin 的最小资源字段。
- message list 查询。
- Runtime task read model 最小 list/get/events/manifest/turn snapshot。

不包含：

- MessageAnnotation。
- 分享链接。
- 会话导出 operation。
- conversation 删除/恢复。
- 修改 runtime 原始事件。
- task 控制；控制仍走 A2A。

Tests：

- Conversation repository/service tests。
- Archive/pin/audit projection tests。
- Runtime task read filter、handler、router tests。

### PR-R4：Resource Operation 基础

Scope：

- `agent_resource_operations` 表。
- operation create/get/list/cancel skeleton（service/store 具备 create；首批外部 REST 暴露 get/list/cancel）。
- 同步 operation result 支持。
- operation create 的客户端 `idempotency_key` 必须按 `workspace_id + operation_type + resource_type + resource_id + client_key` 派生为固定长度 server key 后落库，不保存原始 client key；REST projection 不暴露 `idempotency_key`、`created_by` 等内部控制字段。

不包含：

- 异步 worker 实现。
- KB parsing。
- 真实资源动作派发；后续由 Skill/Tool/KB、导入或索引服务创建 operation。

Tests：

- Operation state transition tests。
- 幂等 create/cancel tests。
- Handler/router/schema/upgrade tests。

### PR-R5：Task Template 和 Workflow Binding 基础

Scope：

- AgentTaskTemplate CRUD。
- AgentWorkflowBinding CRUD。
- Workflow/Mowl 调用 AgentTaskTemplate 的 contract schema 基础字段。
- 明确 Workflow/Mowl 拥有 schedule/case/retry/DAG，Agent 侧只保存模板和节点映射。
- Workflow template/message parts、context refs 和 binding mappings 的敏感字段拦截。
- `workflow_case` 默认幂等策略和 `custom key_template` 校验。

不包含：

- RuntimePolicyProfile CRUD。
- Template dry-run / validate。
- ApprovalPolicy skeleton 和 Agent policy ref。
- Workflow/Mowl 调用 AgentTaskTemplate 的执行链路。
- 自建 scheduler worker。
- Workflow/Mowl scheduler 实现改造。
- 外部事件/webhook 源实现。
- provider native tool 的细粒度审批。

Tests：

- AgentTaskTemplate validation/store/handler/router tests。
- Workflow binding input/output mapping validation/store/handler/router tests。
- schema/upgrade tests。
- RBAC tests。

### PR-R5b：Agent Policy、Approval 和 Template Validate

Scope：

- RuntimePolicyProfile list/get/create/update。
- `agent_resource_runtime_policy_profiles` 表。
- 基础字段：scope_type/scope_ref、admission、budgets、retry、data_policy、artifact_policy、provider_constraints。
- secret/provider run/session 字段拦截。
- Agent policy ref 聚合读写 `/agents/:id/policies`，当前解析 `runtime_policy_ref`，保留 `approval_policy_ref` 和 `guardrail_policy_ref` 作为未来资源引用。
- AgentTaskTemplate 的 `runtime_policy_ref` 解析到同 workspace RuntimePolicyProfile。

后续 Scope：

- ApprovalPolicy skeleton。
- Template dry-run / validate endpoint。
- Workflow/Mowl 调用 AgentTaskTemplate 前的 contract 校验。

不包含：

- 自建 scheduler worker。
- provider native tool 的细粒度审批。
- runtime admission 执行策略。

Tests：

- Policy validation tests。
- Handler/router/schema/upgrade tests。

Tests 后续：

- Approval policy validation tests。
- Template dry-run contract tests。

### PR-R6：Skill Resource

Scope：

- Skill list/get/create/update。
- Skill version table。
- `requirements.skill_refs` 和 `requirements.tool_refs` 同 workspace 存在性校验。
- Skill unified execute endpoint contract，Catalog API key + workspace RBAC 对齐。
- 默认本地 runtime admission submitter：冻结 Skill/input/resource refs，创建 `agent_runtime_tasks`、manifest、turn snapshot、initial event 和 provider_start outbox。
- category/icon/tags/phase/pipeline_ref 元数据。

不包含：

- workflow template import。
- market install。
- override、activate/restore。
- AgentSkillBinding update 和依赖 DAG 校验。
- 真实外部 runtime submitter worker 或 provider 执行链路；本 PR 只落默认本地 task admission。
- 系统 Skill 默认资源种子。

Tests：

- Skill validation tests。
- Skill requirements ref validation tests。
- Skill execute agent ref validation tests。
- Version creation tests。
- Execute admission identity/API key propagation tests。
- Runtime task/manifest/snapshot/event/outbox admission tests，确保原始用户 API key 不落库。
- Handler API key requirement tests。

### PR-R7：Tool Resource 基础

Scope：

- Tool list/get/create/update。
- `ToolSpec` 基础元数据：kind、source_ref、input/output schema、side_effect_class、credential/policy refs、sync 状态。
- category/icon/tags/phase 元数据。
- credential/policy ref 基础边界校验；不解析未来 Approval/Redaction 资源。

不包含：

- read-only KB search ToolGateway implementation；放到 runtime PR-5。
- AgentToolBinding update 和 provider/tool_mode 兼容校验。
- MCP discover。
- HTTP write tool。
- credential onboarding。
- approval state machine。
- market install。

Tests：

- Tool schema validation tests。
- Side-effect class validation tests。
- Handler/router/schema/upgrade tests。

### PR-R8：Skill/Tool Catalog 和 Market 元数据

Scope：

- Skill install endpoint。
- Tool install endpoint。
- market_metadata/schema。
- 安装来源和 workspace installation 记录。

不包含：

- 远程 marketplace 同步 worker。
- 真实 MCP discover。

Tests：

- Install transaction tests。
- Visibility/RBAC tests。
- Duplicate install idempotency tests。

### PR-R9：KB Resource 基础

Scope：

- KB list/get/create/update。
- `KnowledgeBase` 基础元数据：source_type、catalog_asset_refs、default_retrieval_profile_ref、tags、owner_user_id、visibility、index_status。
- metadata/asset ref 禁止保存 secret、provider run/session id。

不包含：

- AgentKnowledgeBinding update。
- 文件列表 schema。
- 已上传文件引用绑定 endpoint skeleton。
- 文件 tags/expiry/status/version metadata。
- 真实解析、分段、索引 worker。
- 分段编辑。
- 外部 drive import。
- retrieval profile。

Tests：

- KB CRUD tests。
- Catalog asset ref validation tests。
- Handler/router/schema/upgrade tests。

### PR-R10：KB Parse/Index Pipeline

Scope：

- 解析 operation worker。
- 文件状态机：`uploaded | parsing | indexed | failed`。
- 分段写入和索引状态。
- 重试和失败恢复。
- 文件 replace/version endpoint。
- 文件 version activate/restore。

Tests：

- Worker integration tests with fake parser/indexer。
- Operation progress tests。
- Reindex tests。
- Version restore tests。

### PR-R11：KB 高级检索和文件来源

Scope：

- KnowledgeRetrievalProfile CRUD。
- retrieval-test endpoint。
- catalog_resource/external_drive/connector_export import source skeleton。
- 文件预览元数据 endpoint。
- Segment reorder。
- Batch file operation endpoint。
- Structured preview/query action skeleton。

不包含：

- Google Drive 真实 OAuth 完整流程。
- provider 直接读取原始文件。

Tests：

- Retrieval profile validation tests。
- Retrieval test fake index tests。
- File source visibility tests。
- Segment reorder transaction tests。
- Batch operation partial failure tests。
- Read-only query validation tests。

### PR-R12：Connection Resource 基础

Scope：

- Connection list/get/create/update。
- `agent_resource_connections` 表。
- 基础字段：kind、endpoint_uri、auth_type、credential_ref、capabilities、owner_user_id、visibility、config、labels、annotations、metadata。
- endpoint/config/metadata 的 secret/provider run/session 字段拦截。

不包含：

- Credential secret store。
- CredentialRef create/rotate endpoint。
- OAuth callback、device flow 或 provider native secret injection。
- Connection test operation 或 `last_test_*` 写入接口。

Tests：

- Connection validation tests。
- Handler/router/schema/upgrade tests。

Tests 后续：

- Credential visibility tests。
- Connection test operation tests。

### PR-R13：Model Registry

Scope：

- ModelConfig list/get/create/update。
- `agent_resource_model_configs` 表。
- 基础字段：source_kind、model_category、provider_ref、model_name、connection_ref、credential_ref、parameters、parameter_schema、capabilities、limits。
- ModelConfig `connection_ref` 同 workspace 存在性校验。
- Agent `model_config_ref` 同 workspace 存在性校验。
- secret/provider run/session 字段拦截。

Tests：

- Model config validation。
- Model config connection ref tests。
- Agent model binding existence tests。
- Handler/router/schema/upgrade tests。

不包含：

- 可用模型聚合 `/models`。
- workspace default model。
- 连通性测试 skeleton。
- Credential onboarding 或 secret store。
- Agent model params schema 深度校验。

Tests 后续：

- Credential ref visibility tests。
- Agent model params schema tests。

### PR-R14：Feedback Review 增强

Scope：

- Feedback stats REST endpoint，支持 agent/message/task/rating filters 和 rating 聚合。
- trace/message/artifact association fields 后续独立扩展。

不包含：

- 自动评测和训练数据生成。
- 将反馈写回 prompt 或 provider。

Tests：

- Feedback filter tests。
- Stats aggregation tests。
- RBAC visibility tests。

### PR-R15：Builder Draft

Scope：

- `agent-drafts` CRUD。
- 从用户描述生成草稿的 deterministic scaffold。
- 保存草稿为 Agent + bindings。
- CurrentStep、selected bindings、proposal version。
- validate endpoint。
- A2A build_step 只作为 draft projection 的约束说明和 contract test。

不包含：

- LLM 自动推荐高级策略。
- 自动 patch 已发布 Agent。

Tests：

- Draft schema validation。
- Save transaction tests。
- RBAC tests。
- A2A build response -> draft update contract tests。

### PR-R16：Message 分享和导出

Scope：

- MessageAnnotation 基础表和 update endpoint。
- ShareLink resource。
- conversation export operation。
- export artifact ref。

不包含：

- 跨 workspace 公开访问策略。
- 富文本导出美化。

Tests：

- MessageAnnotation overlay tests。
- Share permission tests。
- Export operation tests。
- Artifact ref validation tests。

### PR-R17：前端资源面板迁移

Scope：

- Prototype Agent 面板接 Agent Resource API。
- Skill/Tool/KB 资源面板按已实现域逐步接 REST。
- 禁用未实现按钮，避免承诺不存在的能力。
- Conversation 管理、Runtime task read model、Feedback Review、Builder、Model、KB 高级操作按 PR 能力逐步放开。

Tests：

- 前端 smoke：Agent create/edit、binding summary、KB list、Tool list。
- 手工验证：页面没有显示未实现操作为可点击成功态。

## 验收标准

1. Resource API 和 runtime API 边界清晰：资源 CRUD 不走 A2A；运行时 message/task 控制不走 REST CRUD endpoint，runtime task 列表和历史筛选只通过 REST read model 暴露。
2. 原型里每个可点击资源动作都能映射到 REST endpoint、operation 或明确 disabled。
3. Skill 绑定不隐式授权 Tool。
4. Tool 凭证和副作用策略不暴露给 provider。
5. KB 文件进入知识库必须经过 fileservice + KB operation。
6. Operation 状态可查询、可审计，不混用 runtime task state。
7. 每个资源 PR 有独立测试，不要求 runtime provider 可用。
8. AgentTaskTemplate、WorkflowBinding、审批和运行限额通过资源策略表达；cron/event 调度复用 Workflow/Mowl，不污染 A2A 协议。
9. Conversation/Message 的置顶、归档、标注、分享、导出是展示和协作资源，不能改写 runtime 原始事件。
10. Memory/Memoria 不出现在 REST API、A2A extension、数据库表或前端可配置项中。
11. Feedback Review、Runtime Task Read Model 和 Connection/Credential 都是资源读写面，不调用 provider，不改写 runtime task。

## 与 agent-runtime 的接口

资源系统只向 runtime 提供冻结输入：

- `AgentMetadata`
- `AgentSkillBinding`
- `AgentToolBinding`
- `AgentKnowledgeBinding`
- `ModelConfigRef`
- `RuntimePolicyProfile`
- `ApprovalPolicyRef`
- `AgentTaskTemplate`
- `RuntimeManifest` 构建所需的只读快照

Workflow/Mowl 调用 runtime 时额外传入 `workflow_case_id`、`workflow_version_id`、`workflow_node_id` 等 invocation metadata，用于 task 关联和 trace 回跳；这些字段不是资源系统写给 provider 的能力。

runtime 只读取这些快照并执行一轮 task，不负责资源 CRUD、文件解析、工具发现、市场安装、模型 onboarding 或 workflow 调度。

这个边界可以让后续 Astra、Codex、Claude Code 等 provider 接入时不感知资源管理实现，也能让资源管理独立迭代而不破坏 A2A 运行时协议。
