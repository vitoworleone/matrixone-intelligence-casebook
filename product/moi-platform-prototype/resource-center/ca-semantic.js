// MOI 通用行业知识库语义配置。仅用于公开演示，不含客户或个人数据。
window.CA_SEMANTIC = {
  version: {
    tag: 'moi-demo-v1',
    message: '通用行业语义示例',
    entry_count: 10,
    created_at_str: '2026-08-31 10:00'
  },
  caSem: {
    understand: [
      {
        id: 'understand-rewrite',
        title: '补全业务问题',
        enabled: true,
        desc: '将口语、省略和指代整理成可检索的业务问题。',
        prompt: '## 当前日期 {current_date}\n## 历史对话 {history}\n## 用户问题 {raw_query}\n\n请在不改变原意的前提下，补全时间、对象和指标，输出一句完整的检索问题。'
      },
      {
        id: 'understand-clarify',
        title: '判断是否需要澄清',
        enabled: true,
        desc: '在缺少数据范围、时间或对象时发起追问。',
        prompt: '## 用户问题 {rewritten_query}\n## 可用知识范围 {kb_scope}\n\n判断信息是否足以回答；不足时仅输出一条简洁的澄清问题。'
      }
    ],
    entity: [
      {
        id: 'entity-business',
        title: '抽取业务对象与指标',
        enabled: true,
        desc: '识别数据域、指标、时间范围、部门和筛选条件。',
        prompt: '从问题中抽取 domain、metric、time_range、department 和 filters；未出现的字段输出 null。仅输出 JSON。'
      },
      {
        id: 'entity-security',
        title: '识别权限与敏感条件',
        enabled: true,
        desc: '识别访问范围、敏感字段和导出限制。',
        prompt: '判断问题是否涉及个人信息、财务数据、导出或跨部门访问；输出 risk_level 与 required_permission。仅输出 JSON。'
      }
    ],
    route: [
      {
        id: 'route-manufacturing',
        title: '制造运营分析',
        enabled: true,
        desc: '设备、质量、工单和产能相关问题。',
        rule: '当问题包含设备、工单、良率、产能、停机或质量等概念时，路由到 manufacturing_ops。',
        sample: ['本周设备停机时长最高的产线有哪些？', '近三个月的工单按优先级如何分布？'],
        style: '先给出结论和口径，再用表格或要点说明趋势、异常和建议动作。'
      },
      {
        id: 'route-retail',
        title: '零售运营分析',
        enabled: true,
        desc: '订单、库存、会员和活动效果相关问题。',
        rule: '当问题包含订单、库存、门店、会员、活动或复购等概念时，路由到 retail_ops。',
        sample: ['本月各区域的订单完成率如何？', '哪些商品的库存周转需要关注？'],
        style: '说明统计范围与时间区间，突出可执行的运营建议，不臆造缺失数据。'
      },
      {
        id: 'route-enterprise',
        title: '企业知识问答',
        enabled: true,
        desc: '制度、文档、项目与流程相关问题。',
        rule: '当问题涉及制度、合同、项目、流程、文档或知识检索时，路由到 enterprise_knowledge。',
        sample: ['项目交付流程需要哪些审批材料？', '查找最新的数据导出规范。'],
        style: '严格依据召回内容作答；信息不足时明确说明并提示查阅来源。'
      }
    ],
    generate: [
      {
        id: 'generate-grounded',
        title: '基于证据生成回答',
        enabled: true,
        desc: '仅依据检索到的内容生成可追溯回答。',
        prompt: '## 用户问题 {rewritten_query}\n## 召回片段 {retrieved_chunks}\n## 回答风格 {style}\n\n仅依据召回片段回答。先给结论，再给关键证据和建议；没有足够证据时明确说明。'
      }
    ],
    postprocess: [
      {
        id: 'postprocess-security',
        title: '敏感信息与权限校验',
        enabled: true,
        desc: '隐藏不应展示的个人信息、凭据和受限数据。',
        prompt: '检查回答是否泄露个人信息、密钥、内部地址或超出权限范围的数据；必要时以脱敏说明替代。'
      },
      {
        id: 'postprocess-citation',
        title: '引用与格式整理',
        enabled: true,
        desc: '整理层级与来源，便于复核。',
        prompt: '保留原有事实；使用标题、要点和参考来源整理答案，不添加未经证实的信息。'
      }
    ]
  }
};
