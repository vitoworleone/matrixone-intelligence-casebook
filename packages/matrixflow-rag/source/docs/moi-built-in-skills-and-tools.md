# MOI 内置 Skill 与 Tool 清单

更新时间：2026-08-11。

本文只盘点当前代码中的内置能力，不定义目标架构或改造方案。

事实源：

- Momo：`moi-core/catalog/pkg/agentresource/systemagents/zero/agent.json`
- Skill：`moi-core/catalog/pkg/agentresource/systemskills/*/{skill.json,SKILL.md}`
- Tool：`moi-core/agent-tools/tooldefs/*.json`
- System Agent：`moi-core/catalog/pkg/agentresource/systemagents/*/agent.json`

## 汇总

- 内置 Skill：22 个。
- Momo 当前绑定的 Skill：12 个。
- 其他 System Skill：10 个。
- 内置 Tool：103 个。
- Tool visibility：39 个 `user_selectable`、63 个 `internal`、1 个未声明。

Tool visibility 缩写：

- `U`：`user_selectable`
- `I`：`internal`
- `—`：未声明

注意：这里记录的是 Catalog 元数据，不代表完整的服务端授权边界。Agent Builder 当前使用 `runtime_tool_visibility=all` 构造 Tool Inventory。

`agent_visible` 只适用于 `I` Tool：`U` Tool 始终出现在 Momo 与 Momo foundation 派生 Agent 的运行时 Tool surface；`I` Tool 默认不出现，只有 `agent_visible=true` 才出现。当前 19 个 Internal Tool 满足该条件，它们是 Momo 的 Agent Builder、Momo GitHub Skill 的 15 个细粒度 Tool，以及其他 System Skill 所需的 3 个 System Code Tool。所有 Skill 不受此字段限制。专用 System Agent 的显式 Tool 绑定不经过该过滤。

## Skill

### Momo 当前绑定的 Skill

| Skill ID | 名称 | 用途 | Tool 依赖 |
| --- | --- | --- | --- |
| `moi.agent.momo.skill.agent-builder` | Agent Builder | 创建或修改 Agent，提交 Candidate 等待用户确认 | `agent_builder` |
| `moi.agent.momo.skill.pdf` | PDF | PDF 读取、生成、转换、OCR、图片提取和表单处理 | 无静态 `tool_refs` |
| `moi.agent.momo.skill.xlsx` | XLSX | Excel、CSV、TSV 的读取、编辑、分析和生成 | 无静态 `tool_refs` |
| `moi.agent.momo.skill.docx` | DOCX | Word 文档和模板的读取、编辑和生成 | 无静态 `tool_refs` |
| `moi.agent.momo.skill.pptx` | PPTX | PowerPoint 的读取、编辑、合并、拆分和生成 | 无静态 `tool_refs` |
| `moi.agent.momo.skill.github` | GitHub | GitHub 仓库、Issue、PR、Actions、Release、Project 和通知 | 20 个细粒度 GitHub Tool |
| `moi.agent.momo.skill.qq-mail` | QQ Mail | QQ 邮箱搜索、读取、附件和纯文本发信 | `moi_qq_mail` |
| `moi.agent.momo.skill.feishu` | Feishu | 发送飞书文本、富文本 Post 和交互卡片 | 3 个 Feishu Tool |
| `moi.agent.momo.skill.slack` | Slack | 向已知 Slack 频道或会话发送文本、Block Kit 或 legacy attachment 消息 | 3 个 Slack Tool |
| `moi.agent.momo.skill.wecom` | Enterprise WeChat | 企业微信通讯录、消息、媒体和发送统计 | 10 个 WeCom Tool |
| `moi.agent.momo.skill.wecom-group-robot` | Enterprise WeChat Group Robot | 企业微信群机器人发送消息 | `moi_wecom_robot_message` |
| `moi.agent.momo.skill.wecom-mail` | Enterprise WeChat Mail | 企业微信邮箱搜索、读取和附件导入 | `moi_wecom_mail` |

### 其他 System Skill

| Skill ID | 名称 | 用途 | Tool 依赖 |
| --- | --- | --- | --- |
| `algorithmic-art` | Algorithmic Art | 使用代码生成算法艺术 | `system.web_artifact.build`、`system.media.gif_create`、`system.skill_tools.smoke` |
| `brand-guidelines` | Brand Guidelines | 应用品牌视觉和写作规范；当前正文是 Anthropic 品牌规范 | `system.web_artifact.build`、`system.skill_tools.smoke` |
| `canvas-design` | Canvas Design | 海报、静态画布和视觉设计 | `system.web_artifact.build`、`system.skill_tools.smoke` |
| `finance-expense-review` | Finance Expense Review | 审查发票、报销单、付款记录和制度文本 | 无静态 `tool_refs` |
| `frontend-design` | Frontend Design | 前端页面与 UI 视觉设计 | `system.web_artifact.build`、`system.skill_tools.smoke` |
| `grafana` | Grafana | 查看 Grafana 健康状态和数据源，查询 Prometheus 指标与 Loki 日志 | `moi_grafana_health`、`moi_grafana_datasources`、`moi_grafana_prometheus_query`、`moi_grafana_loki_query` |
| `slack-gif-creator` | Slack GIF Creator | 创建适合 Slack 的 GIF 和 Emoji | `system.media.gif_create`、`system.skill_tools.smoke` |
| `kubernetes` | Kubernetes | 只读查看集群版本、Namespace、Node、Pod、Deployment、Service 和 Event | 7 个细粒度 Kubernetes Tool |
| `theme-factory` | Theme Factory | 为文档、网页和幻灯片生成主题 | `system.web_artifact.build`、`system.skill_tools.smoke` |
| `web-artifacts-builder` | Web Artifacts Builder | 构建交互式 HTML/Web Artifact | `system.web_artifact.build`、`system.skill_tools.smoke` |

## Tool

<details open>
<summary>Agent Authoring 与 Channel Binding（2）</summary>

| Tool ID | 用途 | Visibility |
| --- | --- | --- |
| `agent_builder` | 提交完整 Agent Candidate，由 Catalog 校验并交给用户确认 | I（`agent_visible`） |
| `request_channel_binding` | 请求用户绑定 Skill 所需的授权 Channel | — |

</details>

<details>
<summary>Runtime 基础设施（4）</summary>

| Tool ID | 用途 | Visibility |
| --- | --- | --- |
| `read_artifact` | 读取当前任务产物 | I |
| `read_artifact_page` | 读取产物的指定分片 | I |
| `write_file` | 写入聊天可见、可下载文件 | I |
| `read_file` | 按 `file_id` 读取 Workspace FileService 文件 | I |

</details>

<details>
<summary>Developer（2）</summary>

| Tool ID | 用途 | Visibility |
| --- | --- | --- |
| `moi_codex_run` | 委托 Codex 分析代码或图片证据 | U |
| `moi_product_source` | 获取部署版本对应的 MOI 产品源码 Workspace | U |

</details>

<details>
<summary>Web 与图片（4）</summary>

| Tool ID | 用途 | Visibility |
| --- | --- | --- |
| `moi_web_search` | 搜索公网并返回来源信息 | U |
| `moi_web_open` | 打开网页并提取正文 | U |
| `moi_hosted_web_search` | 使用模型 Provider 原生 Web Search | I |
| `moi_hosted_image_generation` | 使用模型 Provider 原生图片生成 | I |

</details>

<details>
<summary>通讯能力（24）</summary>

| Tool ID | 用途 | Visibility |
| --- | --- | --- |
| `moi_feishu_text_message` | 发送飞书文本消息 | U |
| `moi_feishu_post_message` | 发送飞书富文本 Post | U |
| `moi_feishu_card_message` | 发送飞书交互卡片 | U |
| `moi_feishu_message` | 聚合飞书消息传输 | I |
| `moi_slack_text_message` | 发送 Slack 文本 | U |
| `moi_slack_block_message` | 发送 Slack Block Kit | U |
| `moi_slack_attachment_message` | 发送 Slack Legacy Attachment | U |
| `moi_slack_message` | 聚合 Slack 消息传输 | I |
| `moi_qq_mail` | QQ 邮箱搜索、读取、附件和纯文本发信 | U |
| `moi_wecom_mail` | 企业微信邮箱搜索、读取和附件导入 | U |
| `moi_wecom_connection_test` | 测试企业微信应用凭证 | U |
| `moi_wecom_department_list` | 查询企业微信部门 | U |
| `moi_wecom_user_search` | 搜索企业微信用户 | U |
| `moi_wecom_tag_list` | 查询企业微信标签 | U |
| `moi_wecom_tag_users` | 查询企业微信标签成员 | U |
| `moi_wecom_message_send` | 发送企业微信应用消息 | U |
| `moi_wecom_message_recall` | 撤回企业微信应用消息 | U |
| `moi_wecom_message_stats` | 查询企业微信消息发送统计 | U |
| `moi_wecom_media_upload` | 上传企业微信临时媒体 | U |
| `moi_wecom_media_download` | 下载企业微信临时媒体 | U |
| `moi_wecom_robot_message` | 使用企业微信群机器人发送消息 | U |
| `moi_wecom_directory` | 聚合企业微信通讯录操作 | I |
| `moi_wecom_media` | 聚合企业微信媒体操作 | I |
| `moi_wecom_message` | 聚合企业微信消息操作 | I |

</details>

<details>
<summary>GitHub（22）</summary>

| Tool ID | 用途 | Visibility |
| --- | --- | --- |
| `moi_github_authenticated_user` | 获取绑定 Token 对应的 GitHub 账号 | I |
| `moi_github_repository` | 列出仓库或读取仓库元数据 | U |
| `moi_github_repository_code` | 读取分支、Commit、文件和 Git Tree | I |
| `moi_github_search` | 搜索代码、Commit、Issue 或 PR | I |
| `moi_github_issues` | 查询 Issue、评论和 Project Item 信息 | U |
| `moi_github_create_issue` | 创建 Issue | I |
| `moi_github_issue_manage` | 评论、编辑或重开 Issue | I |
| `moi_github_pull_requests` | 查询 PR、文件、Commit、评论、Review 和 Check | U |
| `moi_github_create_pull_request` | 在已有 Head/Base 分支间创建 PR | I |
| `moi_github_pull_request_collaborate` | 评论、Review、请求 Reviewer 或转 Ready | I |
| `moi_github_workflows` | 查询 Actions Workflow、Run、Job、日志和 Artifact | U |
| `moi_github_actions_control` | Dispatch、重跑或取消 Actions | I |
| `moi_github_rate_limit` | 查询 GitHub API 限额 | U |
| `moi_github_labels_milestones` | 查询 Label 和 Milestone | I |
| `moi_github_labels_milestones_manage` | 创建或修改 Label 和 Milestone | I |
| `moi_github_releases` | 查询 Release 或下载受限大小的资源 | I |
| `moi_github_release_manage` | 创建、编辑 Release 和上传资源 | I |
| `moi_github_projects` | 查询 Project V2、Field、Option、Iteration 和 Item | I |
| `moi_github_project_manage` | 修改 Project V2、Item 和 Field | I |
| `moi_github_inbox` | 查询通知、分配 Issue 和 Review 请求 | I |
| `moi_github_tools` | 聚合 GitHub 只读传输 | I |
| `moi_github_write_tools` | 聚合 GitHub 写传输 | I |

</details>

<details>
<summary>Grafana 与 Kubernetes（13）</summary>

| Tool ID | 用途 | Visibility |
| --- | --- | --- |
| `moi_grafana_health` | 查询 Grafana 健康状态 | U |
| `moi_grafana_datasources` | 查询 Grafana Datasource | U |
| `moi_grafana_prometheus_query` | 执行 Prometheus Instant/Range Query | U |
| `moi_grafana_loki_query` | 执行 Loki Instant/Range Query | U |
| `moi_grafana_tools` | 聚合 Grafana 查询能力 | I |
| `moi_kubernetes_version` | 查询 Kubernetes API Server 版本 | U |
| `moi_kubernetes_namespaces` | 查询 Namespace | U |
| `moi_kubernetes_nodes` | 查询 Node | U |
| `moi_kubernetes_pods` | 查询或读取 Pod | U |
| `moi_kubernetes_deployments` | 查询 Deployment | U |
| `moi_kubernetes_services` | 查询 Service | U |
| `moi_kubernetes_events` | 查询 Event | U |
| `moi_kubernetes_tools` | 聚合 Kubernetes 查询能力 | I |

</details>

<details>
<summary>Knowledge 与结构化数据（13）</summary>

| Tool ID | 用途 | Visibility |
| --- | --- | --- |
| `find_rag_files` | 按文件级条件检索知识库来源 | I |
| `search_rag_chunks` | 检索文本片段证据 | I |
| `search_visual_image` | 检索图纸、页面和视觉证据 | I |
| `read_parsed_markdown` | 分页读取解析后的 Markdown | I |
| `search_parsed_markdown` | 在解析后的 Markdown 中精确搜索 | I |
| `describe_schema` | 查看结构化表或语义模型 Schema | I |
| `query_sql` | 对授权结构化数据执行只读 SQL | I |
| `upsert_knowledge_table` | 写入用户明确确认的结构化事实 | I |
| `compute_result_table` | 对 SQL 或既有结果进行确定性计算 | I |
| `select_final_sources` | 选择最终回答实际使用的证据 | I |
| `submit_final_answer` | 提交带规范证据引用的最终回答 | I |
| `moi_knowledge_agent` | 把自然语言知识查询委托给 Knowledge Explore Agent | I |
| `moi_github_identity_lookup` | 查询已确认的 GitHub 与企业微信身份映射 | I |

</details>

<details>
<summary>Workflow（16）</summary>

| Tool ID | 用途 | Visibility |
| --- | --- | --- |
| `list_workflow_apps` | 查询当前用户可用的 Workflow App | I |
| `get_workflow_app` | 查看 Workflow App 详情和输入表单 | I |
| `start_workflow_execution` | 启动一次 Workflow 执行 | I |
| `list_workflow_executions` | 查询 Workflow 执行记录 | I |
| `get_workflow_execution` | 查询单次 Workflow 执行详情 | I |
| `get_workflow_execution_result` | 刷新并读取 Workflow 执行结果 | I |
| `get_latest_failed_workflow_execution` | 查询最近失败执行、当前定义和失败信息 | I |
| `cancel_workflow_execution` | 取消运行中的 Workflow | I |
| `retry_workflow_execution` | 使用原输入重试 Workflow | I |
| `list_file_workflow_executions` | 查询与产出文件关联的 Workflow 执行 | I |
| `browse_capability_groups` | 浏览可见 WorkItem 的能力分组 | I |
| `inspect_workitem` | 查看 WorkItem 契约、UI 字段和代码 | I |
| `search_workitems` | 按能力关键词搜索 WorkItem | I |
| `inspect_workflow_design_manual` | 读取 Reversible Workflow IR 规范 | I |
| `inspect_workflow_dag` | 读取当前 Turn 中用户绘制的 Workflow DAG | I |
| `submit_workflow_candidate` | 提交 Reversible IR Workflow Candidate | I |

</details>

<details>
<summary>System Code（3）</summary>

| Tool ID | 用途 | Visibility |
| --- | --- | --- |
| `system.skill_tools.smoke` | 检查 Custom Tool Worker 的系统 Skill 依赖 | I |
| `system.web_artifact.build` | 构建并验证 Web Artifact | I |
| `system.media.gif_create` | 使用平台媒体链路生成 GIF | I |

</details>

## 当前 System Agent 的显式 Tool 绑定

| System Agent | 当前显式 Tool |
| --- | --- |
| GitHub Issue Operator | Repository、Issues、Create Issue、PR、Issue Manage、Search、Labels、GitHub 聚合读取、Projects、Project Manage、Codex、Identity Lookup、WeCom Message、WeCom Robot |
| GitHub PR Reviewer | PR、PR Collaborate、GitHub 聚合读取、Codex、Identity Lookup、WeCom Message、WeCom Media Download、WeCom Robot |
| Knowledge Explore | RAG、Visual Search、Parsed Markdown、Schema、SQL、Compute、Source Selection、Artifact/File |
| Ops Agent | Grafana 聚合、Kubernetes 聚合、Codex、GitHub 聚合/Repository/Code、WeCom Message/Robot |
| Workflow Designer | WorkItem Discovery、Design Manual、DAG、Latest Failure、Candidate Submit、User Input、File Read/Write |
| Momo | 没有直接 `tool_refs`；Tool 由 Skill、Channel 和 Runtime 能力物化 |
