(function () {
  var runtimeTags = ['moi-runtime', 'moi_runtime', '运行时', '系统'];

  function skill(id, name, icon, cat, desc, tags) {
    return {
      id: id,
      name: name,
      nameEn: name,
      icon: icon,
      cat: cat,
      desc: desc,
      descEn: desc,
      source: '系统技能',
      sourceEn: 'System skill',
      phase: 'p1',
      tags: tags,
      tagsEn: tags,
      pipeline: [],
      created: '2026-07-14'
    };
  }

  window.MOI_LIVE_SKILLS = [
    skill('demo_document_summary', '文档解析与总结', '文', 'docs', '读取 Word、PDF 等文档，提取重点并生成结构化摘要。', ['演示', '文档处理', '总结', '系统']),
    skill('demo_spreadsheet_analysis', '表格分析', '表', 'data', '理解 Excel 和 CSV 数据，完成指标计算、对比和异常识别。', ['演示', '表格', '数据分析', '系统']),
    skill('demo_data_visualization', '数据可视化', '图', 'data', '根据数据特征推荐图表，并生成清晰的可视化分析结果。', ['演示', '图表', '数据分析', '系统']),
    skill('demo_meeting_notes', '会议纪要整理', '会', 'docs', '从会议记录中整理结论、行动项、负责人和截止时间。', ['演示', '会议', '办公协作', '系统']),
    skill('demo_web_research', '网页调研与资料汇总', '研', 'agent', '搜索和阅读公开网页，汇总关键信息并保留来源。', ['演示', '调研', '网页', '系统']),
    skill('demo_email_writing', '邮件撰写与润色', '邮', 'agent', '根据沟通目的撰写或润色专业邮件，适配不同语气。', ['演示', '邮件', '办公写作', '系统']),
    skill('live_algorithmic_art', 'Algorithmic Art', '◉', 'agent', 'Create algorithmic and generative art assets.', ['art', 'generative', 'system', 'visual']),
    skill('live_brand_guidelines', 'Brand Guidelines', '◆', 'agent', 'Apply brand guidelines to visual and written deliverables.', ['brand', '设计', 'guidelines', 'system']),
    skill('live_canvas_design', 'Canvas Design', '▧', 'agent', 'Design visual layouts on a canvas-oriented workflow.', ['canvas', 'layout', 'system', 'visual']),
    skill('live_document_coauthoring', 'Document Coauthoring', '✎', 'docs', 'Draft, revise, and coauthor structured documents.', ['documents', 'editing', 'system', 'writing']),
    skill('live_docx', 'DOCX', 'W', 'docs', 'Create, inspect, edit, or convert Word DOCX documents.', ['docx', 'office', 'system', 'word']),
    skill('live_finance_expense_review', 'Finance Expense Review', '¥', 'security', 'Review expense reimbursement documents, invoices, payment records, and policy text for auditable finance risks.', ['audit', 'expense', 'finance', 'risk', 'system']),
    skill('live_frontend_design', 'Frontend Design', '⌘', 'agent', 'Design polished frontend screens and application interfaces.', ['设计', 'frontend', 'system', 'ui']),
    skill('live_internal_communications', 'Internal Communications', '✉', 'agent', 'Write clear internal communications and workplace messages.', ['communications', 'system', 'workplace', 'writing']),
    skill('live_pdf', 'PDF', 'P', 'docs', 'Read, extract, merge, split, convert, OCR, or create PDF files.', ['document', 'ocr', 'pdf', 'system']),
    skill('live_pptx', 'PPTX', '▥', 'docs', 'Create, inspect, edit, or convert PowerPoint PPTX presentations.', ['office', 'pptx', 'slides', 'system']),
    skill('live_slack_gif_creator', 'Slack GIF Creator', 'GIF', 'agent', 'Create compact Slack-ready animated GIF and emoji assets.', ['emoji', 'gif', 'slack', 'system']),
    skill('live_theme_factory', 'Theme Factory', '✦', 'agent', 'Create coherent visual themes and design systems.', ['设计', 'style', 'system', 'theme']),
    skill('live_web_artifacts_builder', 'Web Artifacts Builder', '</>', 'agent', 'Build interactive web artifacts and frontend deliverables.', ['artifact', 'frontend', 'system', 'web']),
    skill('live_xlsx', 'XLSX', 'X', 'data', 'Inspect, edit, calculate, or create Excel XLSX spreadsheets.', ['office', 'spreadsheet', 'system', 'xlsx'])
  ];

  function tool(id, name, cat, icon, desc, options) {
    options = options || {};
    return {
      id: id,
      name: name,
      cat: cat,
      icon: icon,
      desc: desc,
      source: 'system',
      phase: 'p1',
      tags: runtimeTags.slice(),
      input: options.input || '按工具说明传入参数',
      output: options.output || '返回工具执行结果',
      example: options.example || desc,
      params: options.params || [],
      bindable: options.bindable !== false
    };
  }

  window.MOI_LIVE_TOOLS = [
    tool('live_agent_task', '智能体任务工具', 'batch', '▦', '运行基于 CSV 的智能体任务，并通过已配置的后端报告 worker 结果。', {bindable:false}),
    tool('live_collaboration', '协作工具', 'subagent', '◎', '通过已配置的后端暴露多智能体 v1 和 v2 协作工具。', {bindable:false}),
    tool('live_dynamic_tool', '动态工具', 'dynamic', 'ƒ', '暴露由宿主线程注入的动态函数工具或命名空间工具。', {bindable:false}),
    tool('live_feishu_card', '飞书卡片消息', 'comm', '飞', '通过已绑定的飞书通道实例发送 interactive 交互卡片。'),
    tool('live_feishu_post', '飞书富文本消息', 'comm', '飞', '通过已绑定的飞书通道实例发送 post 富文本消息。'),
    tool('live_feishu_text', '飞书文本消息', 'comm', '飞', '通过已绑定的飞书通道实例发送文本消息。'),
    tool('live_github_issue', 'GitHub Issue', 'dev', 'GH', '通过已绑定的 GitHub 通道实例查询 Issue 列表或指定 Issue。最近、最新或按更新时间查询 Issue 时使用 sort=updated 并传 since 或 updated_since；已知标签、里程碑、负责人、创建人或提及人时一并传入；只对选中的 Issue 再调用 get_issue 读取正文。'),
    tool('live_github_pr', 'GitHub Pull Request', 'dev', 'GH', '通过已绑定的 GitHub 通道实例查询 Pull Request 列表或指定 Pull Request。最近或按更新时间查询 PR 时使用 sort=updated，并优先传 state、head 或 base；GitHub PR 列表没有 since 过滤，不要翻页冒充时间窗口。列表结果是精简摘要；只有明确需要更多 PR 时才继续 next_page。'),
    tool('live_github_rate_limit', 'GitHub 速率限制', 'dev', 'GH', '通过已绑定的 GitHub 通道实例读取当前 token 的 API rate limit。'),
    tool('live_github_repo', 'GitHub 仓库', 'dev', 'GH', '通过已绑定的 GitHub 通道实例读取仓库基础信息。'),
    tool('live_github_actions', 'GitHub Actions', 'dev', 'GH', '通过已绑定的 GitHub 通道实例查询 Workflow 和 Workflow Run。最近运行记录用 created 日期过滤，并优先用 branch、event、status、conclusion、actor 或 head_sha 缩小范围；只有明确需要更多运行记录时才翻页。'),
    tool('live_grafana_datasource', 'Grafana 数据源', 'dev', 'G', '通过已绑定的 Grafana 通道实例读取数据源列表。该列表没有 since/time 过滤或分页；worker 返回有界摘要，大量数据源可能被截断。使用返回的 datasource UID 调用 Prometheus 或 Loki 查询工具。'),
    tool('live_grafana_health', 'Grafana 健康状态', 'dev', 'G', '通过已绑定的 Grafana 通道实例读取 /api/health。'),
    tool('live_loki_query', 'Loki 查询', 'dev', 'L', '通过已绑定的 Grafana 通道实例执行 Loki 即时或区间日志查询。最近、最新日志窗口使用带 start、end 的 query_loki_range，并保持 limit 有界。'),
    tool('live_prometheus_query', 'Prometheus 查询', 'dev', 'P', '通过已绑定的 Grafana 通道实例执行 Prometheus 即时或区间查询。最近、最新窗口或趋势分析使用带 start、end、step 的 query_prometheus_range；即时查询只用于单点评估。'),
    tool('live_image_generation', '图像生成工具', 'image', '✦', '通过已配置的独立图像后端生成或编辑图片。', {bindable:false}),
    tool('live_k8s_deployment', 'Kubernetes Deployment', 'dev', 'K8s', '通过已绑定的 Kubernetes 通道实例列出 Deployment。该工具没有 since/time 过滤；已知 namespace 或 selector 时优先限定范围；只有上次结果 has_next=true 时才使用 continue，不要靠翻页冒充最近变更。'),
    tool('live_k8s_event', 'Kubernetes Event', 'dev', 'K8s', '通过已绑定的 Kubernetes 通道实例列出 Event，用于故障排查。该工具没有 since/time 过滤；优先用 namespace、field_selector 或 label_selector 限定排障范围；只有上次结果 has_next=true 时才使用 continue。'),
    tool('live_k8s_namespace', 'Kubernetes Namespace', 'dev', 'K8s', '通过已绑定的 Kubernetes 通道实例列出 Namespace。该工具没有 since/time 过滤；保持 limit 有界；只有上次结果 has_next=true 时才使用 continue，不要靠翻页冒充最近结果。'),
    tool('live_k8s_node', 'Kubernetes Node', 'dev', 'K8s', '通过已绑定的 Kubernetes 通道实例列出 Node。该工具没有 since/time 过滤；能用 label_selector 或 field_selector 时优先限定范围；只有上次结果 has_next=true 时才使用 continue，不要用 continue 模拟时间窗口。'),
    tool('live_k8s_pod', 'Kubernetes Pod', 'dev', 'K8s', '通过已绑定的 Kubernetes 通道实例列出 Pod 或读取指定 Pod。该工具没有 since/time 过滤；最近状态排查时，已知 namespace、name、label_selector 或 field_selector 就必须用来限定范围，保持 limit 有界；只有上次结果 has_next=true 时才使用 continue；选中具体 Pod 后用 get_pod。'),
    tool('live_k8s_service', 'Kubernetes Service', 'dev', 'K8s', '通过已绑定的 Kubernetes 通道实例列出 Service。该工具没有 since/time 过滤；已知 namespace 或 selector 时优先限定范围；只有上次结果 has_next=true 时才使用 continue，不要靠翻页冒充最近变更。'),
    tool('live_k8s_version', 'Kubernetes 版本', 'dev', 'K8s', '通过已绑定的 Kubernetes 通道实例读取 API Server 版本。'),
    tool('live_available_plugins', '列出可用插件', 'discovery', '▧', '列出当前可安装或可连接的插件与连接器候选项。', {bindable:false}),
    tool('live_mcp_tool', 'MCP 工具', 'mcp', 'MCP', '通过 MOI 兼容路由暴露 MCP 资源和运行时 MCP 工具。', {bindable:false}),
    tool('live_memory_tool', '记忆工具', 'memory', '◫', '通过已配置的记忆存储列出、读取、搜索和写入笔记。', {bindable:false}),
    tool('live_qq_mail', 'QQ 邮箱', 'comm', '邮', '通过 IMAP/SMTP 连接 QQ 邮箱，读取、搜索邮件并查看附件。凭证由平台连接解析，工具参数中不要携带授权码。'),
    tool('live_skill_resource', '技能资源工具', 'skill', '技', '列出已启用的技能，并读取完整的技能资源内容。', {bindable:false}),
    tool('live_slack_attachment', 'Slack 附件消息', 'comm', 'S', '通过已绑定的 Slack 通道实例发送 legacy attachment 消息。'),
    tool('live_slack_block_kit', 'Slack Block Kit', 'comm', 'S', '通过已绑定的 Slack 通道实例发送 Block Kit 结构化消息。'),
    tool('live_wecom_mail', '企业微信邮箱', 'comm', '邮', '通过 IMAP/SMTP 连接企业微信邮箱，读取、搜索邮件并查看附件。凭证由平台连接解析，工具参数中不要携带授权码。'),
    tool('live_slack_text', 'Slack 文本消息', 'comm', 'S', '通过已绑定的 Slack 通道实例发送频道文本消息。'),
    tool('live_web_runtime', '网页运行工具', 'web', '🌐', '通过已配置的网页后端运行独立的网页操作。', {bindable:false}),
    tool('live_wecom_connection_test', '企业微信连接测试', 'comm', '企', '通过托管 Java WeCom Worker 测试企业微信应用凭证。'),
    tool('live_wecom_departments', '企业微信部门列表', 'comm', '企', '通过托管 Java WeCom Worker 读取企业微信部门列表。该 API 没有 since/update 时间过滤；用户要求具体部门范围时传 parent_id 和 limit；只有需要完整盘点时才省略 parent_id。'),
    tool('live_wecom_download_media', '企业微信下载素材', 'comm', '企', '通过托管 Java WeCom Worker 按 media_id 下载企业微信临时素材。设置 max_bytes 限制返回给模型的 base64 内容；大文件返回元数据和截断信息。'),
    tool('live_wecom_upload_media', '企业微信上传素材', 'comm', '企', '通过托管 Java WeCom Worker 上传企业微信临时素材并返回 media_id。'),
    tool('live_wecom_recall_message', '企业微信撤回消息', 'comm', '企', '通过托管 Java WeCom Worker 按 msgid 撤回企业微信应用消息。'),
    tool('live_wecom_send_message', '企业微信发送消息', 'comm', '企', '通过托管 Java WeCom Worker 发送企业微信应用 text、markdown、image、file 或 template_card 消息。'),
    tool('live_wecom_send_stats', '企业微信发送统计', 'comm', '企', '通过托管 Java WeCom Worker 读取企业微信应用消息发送统计。按 time_type 查询一个离散统计窗口；该 API 没有 since/start/end 范围字段。'),
    tool('live_wecom_group_bot', '企业微信群机器人', 'comm', '企', '发送企业微信群机器人文本、Markdown、图片、图文、文件或模板卡片消息。'),
    tool('live_wecom_tags', '企业微信标签列表', 'comm', '企', '通过托管 Java WeCom Worker 读取企业微信通讯录标签列表。该 API 没有 since/update 时间过滤；优先用标签做安全的收件人或受众范围选择，再读取成员。'),
    tool('live_wecom_tag_members', '企业微信标签成员', 'comm', '企', '通过托管 Java WeCom Worker 读取标签下的成员和部门范围。该 API 没有 since/update 时间过滤；必须传 tag_id 并保持 limit 有界；大标签结果可能截断。'),
    tool('live_wecom_user_search', '企业微信成员搜索', 'comm', '企', '通过托管 Java WeCom Worker 按部门、状态和关键字搜索企业微信成员。该 API 没有 since/update 时间过滤；已知部门、关键字或状态时传 department_ids、query、status、limit，不要全量搜索通讯录。')
  ];
})();
