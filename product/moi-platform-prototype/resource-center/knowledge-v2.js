(function () {
  'use strict';

  const app = document.getElementById('knowledgeV2App');
  if (!app) return;

  const icons = {
    search: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="11" cy="11" r="7"/><path d="m20 20-3.5-3.5"/></svg>',
    plus: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 5v14M5 12h14"/></svg>',
    fileAdd: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><path d="M6 2h8l4 4v16H6z"/><path d="M14 2v5h5M12 11v7M8.5 14.5h7"/></svg>',
    folder: '<svg viewBox="0 0 24 24" fill="currentColor"><path d="M3 5a2 2 0 0 1 2-2h5l2 2h7a2 2 0 0 1 2 2v11a3 3 0 0 1-3 3H6a3 3 0 0 1-3-3V5Z"/></svg>',
    edit: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 20h9"/><path d="M16.5 3.5a2.1 2.1 0 0 1 3 3L8 18l-4 1 1-4Z"/></svg>',
    trash: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M3 6h18M8 6V4h8v2M19 6l-1 15H6L5 6"/><path d="M10 11v5M14 11v5"/></svg>',
    back: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="m15 18-6-6 6-6"/></svg>',
    file: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M6 2h8l4 4v16H6z"/><path d="M14 2v5h5"/></svg>',
    table: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="4" width="18" height="16" rx="2"/><path d="M3 9h18M9 9v11"/></svg>',
    robot: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="4" y="7" width="16" height="12" rx="3"/><path d="M12 3v4M8 12h.01M16 12h.01M8 16h8"/></svg>',
    send: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="m22 2-7 20-4-9-9-4Z"/><path d="M22 2 11 13"/></svg>',
    metric: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><path d="M4 19V9M10 19V5M16 19v-7M22 19H2"/></svg>',
    schema: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><ellipse cx="12" cy="5" rx="8" ry="3"/><path d="M4 5v7c0 1.7 3.6 3 8 3s8-1.3 8-3V5M4 12v7c0 1.7 3.6 3 8 3s8-1.3 8-3v-7"/></svg>',
    rule: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><path d="M4 3h13a3 3 0 0 1 3 3v15H7a3 3 0 0 1-3-3V3Z"/><path d="M7 17h13M8 8h8M8 12h6"/></svg>',
    constraint: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><path d="m12 2 8 3v6c0 5-3.4 9.2-8 11-4.6-1.8-8-6-8-11V5Z"/><path d="m8.5 12 2.2 2.2 4.8-5"/></svg>',
    relationship: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><rect x="3" y="3" width="6" height="6" rx="1"/><rect x="15" y="15" width="6" height="6" rx="1"/><path d="M9 6h4a4 4 0 0 1 4 4v5M15 18h-4a4 4 0 0 1-4-4V9"/></svg>',
    qa: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><circle cx="12" cy="12" r="9"/><path d="m8 12 2.5 2.5L16 9"/></svg>',
    dynamic: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><path d="m13 2-8 12h7l-1 8 8-12h-7Z"/></svg>',
    download: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 3v12m0 0 5-5m-5 5-5-5M5 21h14"/></svg>'
  };

  const categories = [
    { key: 'metric', name: '指标', desc: '经过确认、有业务名称、能被系统执行的数值计算口径。', icon: 'metric' },
    { key: 'schema', name: '表和字段说明', desc: '描述表或字段自身稳定的业务含义，并与数据库原生注释分别保留。', icon: 'schema' },
    { key: 'business-rule', name: '业务规则', desc: '关联结构化数据表，描述需要智能体判断业务场景后才能选择表、字段、期间或口径的规则。', icon: 'rule' },
    { key: 'constraint', name: '强制约束', desc: '只要指定表参与查询就必须加入、且不能被模型或用户覆盖的固定条件。', icon: 'constraint' },
    { key: 'relationship', name: '表关联', desc: '定义两张表经过确认的连接方式和关联字段。', icon: 'relationship' },
    { key: 'standard-qa', name: '标准问答', desc: '保存高频标准问题及经过业务确认的 SQL，可配置为参考使用，或在用户问题与标准问题高置信度语义一致时强制使用。', icon: 'qa' },
    { key: 'dynamic-query', name: '动态查询', desc: '按需执行预先配置的只读 SQL，为回答提供实时业务数据。', icon: 'dynamic' },
    { key: 'unstructured-rule', name: '业务规则', desc: '关联已完成解析与嵌入的文件，补充需要智能体理解和判断的非结构化业务规则。', icon: 'rule' },
    { key: 'unstructured-qa', name: '标准问答', desc: '基于已解析文件保存经过确认的标准问题和答案，可供智能体参考或在问题语义一致时直接使用。', icon: 'qa' }
  ];

  const semanticGroups = [
    { key: 'structured', name: 'NL2SQL语义', categories: ['metric', 'schema', 'business-rule', 'constraint', 'relationship', 'standard-qa', 'dynamic-query'] },
    { key: 'unstructured', name: 'RAG语义', categories: ['unstructured-rule', 'unstructured-qa'] }
  ];

  const relationshipComparisons = [
    { value: 'equals', label: 'equals', description: '等于', symbol: '=' },
    { value: 'not_equals', label: 'not_equals', description: '不等于', symbol: '<>' },
    { value: 'greater_than', label: 'greater_than', description: '大于', symbol: '>' },
    { value: 'greater_than_or_equal', label: 'greater_than_or_equal', description: '大于等于', symbol: '>=' },
    { value: 'less_than', label: 'less_than', description: '小于', symbol: '<' },
    { value: 'less_than_or_equal', label: 'less_than_or_equal', description: '小于等于', symbol: '<=' }
  ];

  function relationshipComparisonSymbol(value) {
    const comparison = relationshipComparisons.find(function (item) { return item.value === value; });
    return comparison ? comparison.symbol : '=';
  }

  let nextModelId = 20;
  let nextSourceId = 100;
  let nextAssetId = 100;
  const models = [
    { id: 1, name: 'BPC', desc: '用于合并报表指标、期间和组织维度相关查询。', files: 0, tables: 1 }
  ];

  const baseSources = [
    { id: 1, name: 'bpc_consolidated_report', type: '数据表', size: '2,486,320 行', storage: '1.86 GB', path: '默认 / jst_flat_table / bpc_consolidated_report', comment: 'BPC 合并报表明细，用于合并报表指标、期间和组织维度查询', status: '已完成', updated: '2026-07-22 16:07', enabled: true }
  ];
  const sourceStore = {};

  const seedAssets = [
    { id: 1, category: 'metric', name: '营业收入', binding: 'jst_flat_table.bpc_consolidated_report', desc: '科目规则：APL6000；需要置反', updated: '2026-08-08 14:20', fields: { name: '营业收入', assetId: 'bpc_revenue', metricType: 'aggregate', sourceTable: 'jst_flat_table.bpc_consolidated_report', aggregation: 'SUM', field: 'b28_s_sdata', multiplier: '-1', output: 'number', synonyms: '收入, 主营收入', description: '科目规则：APL6000；需要置反' } },
    { id: 2, category: 'metric', name: '主营业务成本', binding: 'jst_flat_table.bpc_consolidated_report', desc: "科目规则：APL6600 + APL6401 + APL5001；不需要置反；其中 APL6600 需满足 b28_s_kgdbveh = 'TFUA1000'", updated: '2026-08-08 11:02', fields: { name: '主营业务成本', assetId: 'main_business_cost', metricType: 'aggregate', sourceTable: 'jst_flat_table.bpc_consolidated_report', aggregation: 'SUM', field: 'b28_s_sdata', multiplier: '1', output: 'number', synonyms: '主营成本, 主业成本', description: "科目规则：APL6600 + APL6401 + APL5001；不需要置反；其中 APL6600 需满足 b28_s_kgdbveh = 'TFUA1000'" } },
    { id: 3, category: 'metric', name: '销售毛利率', binding: '引用已发布指标', desc: '毛利率=（营业收入-主营业务成本）/ 营业收入', updated: '2026-08-07 17:45', fields: { name: '销售毛利率', assetId: 'gross_profit_margin', metricType: 'derived', sourceTable: 'jst_flat_table.bpc_consolidated_report', formula: '(bpc_revenue - main_business_cost) / bpc_revenue', output: 'percent', synonyms: '毛利率', description: '毛利率=（营业收入-主营业务成本）/ 营业收入' } },
    ...buildRequestedMetricSeeds(),
    { id: 4, category: 'schema', name: 'BPC 合并报表明细', binding: 'jst_flat_table.bpc_consolidated_report', desc: '用于合并报表指标、期间和组织维度相关查询。', updated: '2026-08-07 10:32', fields: { name: 'BPC 合并报表明细', assetId: 'bpc_consolidated_report', table: 'jst_flat_table.bpc_consolidated_report', tableDescription: 'BPC 合并报表明细', description: '用于合并报表指标、期间和组织维度相关查询。' } },
    { id: 5, category: 'business-rule', name: '年度期初期间', binding: 'jst_flat_table.bpc_consolidated_report', desc: '当用户查询年度期初数据时，使用上一年度年末期间的数据。', updated: '2026-08-06 18:05', fields: { name: '年度期初期间', assetId: 'annual_opening_period', relatedTables: 'jst_flat_table.bpc_consolidated_report', content: '当用户查询年度期初数据时，使用上一年度年末期间的数据。', ruleContent: '当用户查询年度期初数据时，使用上一年度年末期间的数据。', description: '当用户查询年度期初数据时，使用上一年度年末期间的数据。' } },
    { id: 6, category: 'constraint', name: '人民币币种约束', binding: 'jst_flat_table.bpc_consolidated_report', desc: 'b28_s_kgd4kbn = CNY', updated: '2026-08-06 11:16', fields: { name: '人民币币种约束', assetId: 'bpc_currency_cny', sourceTable: 'jst_flat_table.bpc_consolidated_report', field: 'b28_s_kgd4kbn', operator: '等于（=）', value: 'CNY', description: '查询 BPC 报表时强制限定人民币' } },
    { id: 7, category: 'relationship', name: 'BPC 与组织主数据', binding: 'bpc → organization_master', desc: 'BPC报表数据通过组织编码关联组织主数据', updated: '2026-08-05 15:42', fields: { name: 'BPC 与组织主数据', assetId: 'bpc_organization', leftTable: 'bpc', rightTable: 'organization_master', joinType: 'LEFT JOIN', leftField: 'b28_s_kgd4rtr_kgdxoi5', rightField: 'organization_code', description: 'BPC报表数据通过组织编码关联组织主数据' } },
    { id: 8, category: 'standard-qa', name: 'EO_1000 公司营业收入', binding: '仅供参考', desc: '查询EO_1000公司2025年5月实际人民币营业收入', updated: '2026-08-04 09:30', fields: { name: 'EO_1000 公司营业收入', assetId: 'bpc_revenue_eo_may', usageMode: 'reference', question: 'EO_1000公司2025年5月营业收入是多少', similarQuestions: '查询EO_1000公司2025年5月营收\nEO_1000在2025年5月的营业收入是多少\n2025年5月EO_1000公司实现了多少营业收入', sql: "SELECT base.bpc_revenue AS `营业收入`\nFROM (\n  SELECT SUM(CASE WHEN account_path LIKE '%/APL6000/%' THEN b28_s_sdata * -1 END) AS bpc_revenue\n  FROM bpc_consolidated_report\n  WHERE b28_s_kgdtvnx = 'ACT_LG'\n    AND b28_s_kgd4rtr_kgdxoi5 = 'EO_1000'\n    AND b28_s_kgd353d = '202505'\n    AND b28_s_kgd4kbn = 'CNY'\n) AS base", description: '查询EO_1000公司2025年5月实际人民币营业收入' } },
    { id: 9, category: 'dynamic-query', name: '合并科目映射', binding: '直接注入上下文', desc: '查询合并科目的编码、名称、父级编码和是否置反，用于确定用户提到的科目对应的实际取数范围', updated: '2026-08-03 16:24', fields: { name: '合并科目映射', assetId: 'bpc_account_mapping', resultProcessingMode: 'direct_context', sql: 'SELECT code, name, parent_code, reverse_flag FROM account_mapping', description: '查询合并科目的编码、名称、父级编码和是否置反，用于确定用户提到的科目对应的实际取数范围' } },
    { id: 50, category: 'unstructured-rule', name: 'BPC 指标口径文档规则', binding: 'BPC_指标口径说明.pdf', desc: '当问题涉及文档中定义的指标口径和使用说明时，优先依据该文件判断。', updated: '2026-08-03 15:50', fields: { name: 'BPC 指标口径文档规则', assetId: 'bpc_document_metric_policy', relatedFiles: 'BPC_指标口径说明.pdf', content: '当问题涉及文档中定义的指标口径和使用说明时，优先依据该文件判断。', ruleContent: '当问题涉及文档中定义的指标口径和使用说明时，优先依据该文件判断。', description: '当问题涉及文档中定义的指标口径和使用说明时，优先依据该文件判断。' } },
    { id: 51, category: 'unstructured-qa', name: '营业收入的指标口径是什么', binding: 'BPC_指标口径说明.pdf', desc: '营业收入按合并科目 APL6000 取数，汇总后需要置反。', updated: '2026-08-03 15:40', fields: { name: '营业收入的指标口径是什么', assetId: 'bpc_revenue_definition', usageMode: 'reference', relatedFiles: 'BPC_指标口径说明.pdf', question: '营业收入的指标口径是什么', similarQuestions: '营业收入怎么计算\n营业收入取哪个科目', answer: '营业收入按合并科目 APL6000 取数，汇总后需要置反。', description: '说明 BPC 营业收入指标的标准取数口径。' } }
  ];
  const assetStore = {};

  const agents = [
    { name: 'BPC 财务分析助手', desc: '基于 BPC 合并报表指标完成经营与财务问答', status: '运行中', knowledgeCount: 1, updated: '2026-07-22 16:30' },
    { name: '合并报表校验助手', desc: '校验期间、组织、币种约束与财务指标口径', status: '草稿', knowledgeCount: 1, updated: '2026-07-22 16:18' }
  ];

  const catalogSources = [
    { id: 'catalog-bpc', name: 'bpc_consolidated_report', type: '数据表', size: '2,486,320 行', path: '默认 / jst_flat_table / bpc_consolidated_report', comment: 'BPC 合并报表明细，用于合并报表指标、期间和组织维度查询', updated: '2026-07-22 16:07' },
    { id: 'catalog-org', name: 'organization_master', type: '数据表', size: '1,286 行', path: '默认 / jst_flat_table / organization_master', comment: '组织主数据，包含组织编码、名称及上下级关系', updated: '2026-07-22 15:48' },
    { id: 'catalog-account', name: 'account_mapping', type: '数据表', size: '8,642 行', path: '默认 / jst_flat_table / account_mapping', comment: '合并科目映射，包含科目编码、名称、父级编码和置反标识', updated: '2026-07-21 18:26' },
    { id: 'catalog-bpc-guide', name: 'BPC_指标口径说明.pdf', type: '文件', size: '2.8 MB', path: '默认 / BPC / BPC_指标口径说明.pdf', updated: '2026-07-20 11:35' }
  ];

  const tableComments = {
    bpc: 'BPC 报表数据，包含合并科目、组织、期间和金额等信息',
    bpc_consolidated_report: 'BPC 合并报表明细，用于合并报表指标、期间和组织维度查询',
    organization_master: '组织主数据，包含组织编码、名称及上下级关系',
    account_mapping: '合并科目映射，包含科目编码、名称、父级编码和置反标识'
  };

  function tableCommentFor(tableName, fallback) {
    const shortName = String(tableName || '').split('.').filter(Boolean).pop() || '';
    return String(fallback || tableComments[shortName] || '当前知识库中的业务数据表');
  }

  function buildRequestedMetricSeeds() {
    const sourceTable = 'jst_flat_table.bpc_consolidated_report';
    const account = function (code) { return { type: 'condition', field: 'account_path', comparison: 'contains', value: '/' + code + '/' }; };
    const accounts = function () { return { type: 'group', logic: 'or', conditions: Array.from(arguments).map(account) }; };
    const businessEquals = function (value) { return { type: 'condition', field: 'b28_s_kgdbveh', comparison: 'equals', value: value }; };
    const businessIn = function () { return { type: 'condition', field: 'b28_s_kgdbveh', comparison: 'in', value: Array.from(arguments) }; };
    const accountWithBusiness = function (code, businessFilter) { return { type: 'group', logic: 'and', conditions: [account(code), businessFilter] }; };
    const alternatives = function (qualifiedAccount) { return { type: 'group', logic: 'or', conditions: [qualifiedAccount].concat(Array.prototype.slice.call(arguments, 1).map(account)) }; };
    const aggregateSeeds = [
      ['main_business_revenue', '主营业务收入', 'APL6001', account('APL6001'), 1, ''],
      ['net_profit', '净利润', 'APL2000 + APL6600', accounts('APL2000', 'APL6600'), -1, ''],
      ['total_profit', '利润总额', 'APL3000 + APL6600 + APL5001', accounts('APL3000', 'APL6600', 'APL5001'), -1, ''],
      ['operating_profit', '营业利润', 'APL4000 + APL6600 + APL5001', accounts('APL4000', 'APL6600', 'APL5001'), -1, ''],
      ['total_operating_cost', '营业总成本', 'APL6600 + APL5000', accounts('APL6600', 'APL5000'), 1, '不等于营业成本'],
      ['operating_cost', '营业成本', 'APL6600 + APL5100 + APL5001', alternatives(accountWithBusiness('APL6600', businessEquals('TFUA1000')), 'APL5100', 'APL5001'), 1, "APL6600 需满足 b28_s_kgdbveh = 'TFUA1000'"],
      ['period_expenses', '期间费用', 'APL6600 + APL6690', alternatives(accountWithBusiness('APL6600', businessIn('TFUA2000', 'TFUA3000', 'TFUA4000')), 'APL6690'), 1, "APL6600 需满足 b28_s_kgdbveh IN ('TFUA2000','TFUA3000','TFUA4000')"],
      ['sales_admin_rnd_expenses', '管销研费用', 'APL6600', accountWithBusiness('APL6600', businessIn('TFUA2000', 'TFUA3000', 'TFUA4000')), 1, "需满足 b28_s_kgdbveh IN ('TFUA2000','TFUA3000','TFUA4000')"],
      ['research_and_development_expenses', '研发费用', 'APL6600', accountWithBusiness('APL6600', businessEquals('TFUA4000')), 1, "需满足 b28_s_kgdbveh = 'TFUA4000'"],
      ['manufacturing_expenses', '制造费用', 'APL6600', accountWithBusiness('APL6600', businessEquals('TFUA1000')), 1, "需满足 b28_s_kgdbveh = 'TFUA1000'"],
      ['net_cash_flow', '现金净流量', 'CF0201 + CF0202 + CF0203', accounts('CF0201', 'CF0202', 'CF0203'), 1, ''],
      ['government_subsidy', '政府补贴', 'CF153', account('CF153'), 1, ''],
      ['basic_earnings_per_share', '基本每股收益', 'ET0649', account('ET0649'), -1, ''],
      ['net_profit_attributable_to_parent', '归母净利润', '6901010000', account('6901010000'), 1, ''],
      ['undistributed_profit', '未分配利润', 'ABS4103 + ABS4104', accounts('ABS4103', 'ABS4104'), -1, ''],
      ['labor_cost', '人工成本', '6602* / 6603*', accounts('6602', '6603'), 1, ''],
      ['accounts_receivable', '应收账款', 'ABS1122', { type: 'group', logic: 'and', conditions: [account('ABS1122'), { type: 'condition', field: 'account_path', comparison: 'not_contains', value: '/ABS112202/' }] }, 1, '排除 ABS112202'],
      ['accounts_receivable_book_balance', '应收账款账面余额', 'ABS112201', account('ABS112201'), 1, ''],
      ['other_receivables_book_balance', '其他应收款-账面余额', 'ABS1231', account('ABS1231'), 1, '询问其他应收、其它应收余额或其它应收总额时默认使用此口径']
    ];
    const aggregateAssets = aggregateSeeds.map(function (seed, index) {
      const description = '科目规则：' + seed[2] + '；' + (seed[4] === -1 ? '需要置反' : '不需要置反') + (seed[5] ? '；' + seed[5] : '');
      return { id: 10 + index, category: 'metric', name: seed[1], binding: sourceTable, desc: description, updated: '2026-07-22 16:07:13', fields: { name: seed[1], assetId: seed[0], metricType: 'aggregate', sourceTable: sourceTable, aggregation: 'SUM', aggregationFunction: 'sum', field: 'b28_s_sdata', aggregationField: 'b28_s_sdata', multiplier: seed[4], filterGroup: seed[3], description: description, synonyms: [] } };
    });
    const derivedSeeds = [
      ['current_ratio', '流动比率', '流动比率 = 流动资产 / 流动负债', '(current_assets / current_liabilities)', []],
      ['quick_ratio', '速动比率', '速动比率 =（流动资产 - 存货）/ 流动负债', '((current_assets - inventory) / current_liabilities)', []],
      ['cash_ratio', '现金比例', '现金比例 =（货币资金 + 交易性金融资产）/ 流动负债', '((monetary_funds + trading_financial_assets) / current_liabilities)', ['现金比率']],
      ['debt_to_asset_ratio', '资产负债率', '资产负债率 = 负债总额 / 资产总额', '(total_liabilities / total_assets)', []]
    ];
    return aggregateAssets.concat(derivedSeeds.map(function (seed, index) {
      return { id: 40 + index, category: 'metric', name: seed[1], binding: '引用已发布指标', desc: seed[2], updated: '2026-07-22 16:07:13', fields: { name: seed[1], assetId: seed[0], metricType: 'derived', sourceTable: sourceTable, formula: seed[3], metricFormulaExpression: seed[3], description: seed[2], synonyms: seed[4], output: 'percent' } };
    }));
  }

  const state = {
    page: 'board',
    boardTab: 'knowledge',
    query: '',
    modelId: null,
    detailTab: 'source',
    semCategory: 'metric',
    semanticAdvancedOpen: false,
    semQuery: '',
    semanticPage: 1,
    modal: null,
    pendingConfirm: null,
    semanticImport: null,
    semanticImportMode: 'append',
    createDraft: { name: '', description: '' },
    sourceSelected: [],
    catalogScope: 'tables',
    catalogQuery: '',
    catalogPage: 1,
    catalogTreeOpen: { catalog: true, database: true },
    editingAsset: null,
    assetDraft: null,
    assetError: '',
    assetValidationVisible: false,
    semanticPicker: null,
    semanticPickerQuery: '',
    schemaColumnSearch: '',
    sourceDetailTab: 'columns',
    documentTab: 'preview',
    expiryForce: false,
    pipelineOpen: false,
    chatReturn: 'board',
    chatReturnDetailTab: 'source',
    deletedDemo: { models: [], sources: [], assets: [] },
    messages: [
      { role: 'bot', text: '你好，我是 BPC 财务分析助手。你可以询问营业收入、主营业务成本、利润、现金流或财务比率，我会按已配置的指标口径、期间、组织和币种约束查询。' }
    ]
  };

  function h(value) {
    return String(value == null ? '' : value).replace(/[&<>"]/g, function (ch) {
      return { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' }[ch];
    });
  }
  function attr(value) { return h(value).replace(/'/g, '&#39;'); }
  let clippedContentTooltip = null;
  let clippedContentAnchor = null;
  let clippedContentShowTimer = null;

  function hideClippedContentTooltip() {
    if (clippedContentShowTimer) window.clearTimeout(clippedContentShowTimer);
    clippedContentShowTimer = null;
    if (clippedContentAnchor) clippedContentAnchor.removeAttribute('aria-describedby');
    if (clippedContentTooltip) clippedContentTooltip.remove();
    clippedContentTooltip = null;
    clippedContentAnchor = null;
  }

  function showClippedContentTooltip(anchor) {
    if (!anchor) return;
    const alwaysShowFullText = anchor.classList.contains('kv2-single-line') || anchor.hasAttribute('data-full-text');
    const isClipped = anchor.scrollWidth > anchor.clientWidth + 1 || anchor.scrollHeight > anchor.clientHeight + 1;
    if (!alwaysShowFullText && !isClipped) return;
    const fullText = String(anchor.dataset.fullText || anchor.getAttribute('title') || anchor.textContent || '').trim();
    if (!fullText) return;
    hideClippedContentTooltip();
    anchor.dataset.fullText = fullText;
    anchor.removeAttribute('title');
    const tooltip = document.createElement('div');
    tooltip.id = 'kv2ClippedContentTooltip';
    tooltip.className = 'kv2-content-tooltip';
    tooltip.setAttribute('role', 'tooltip');
    const tooltipBody = document.createElement('div');
    tooltipBody.className = 'kv2-content-tooltip-body';
    tooltipBody.textContent = fullText;
    tooltip.appendChild(tooltipBody);
    document.body.appendChild(tooltip);
    const anchorRect = anchor.getBoundingClientRect();
    const tooltipRect = tooltip.getBoundingClientRect();
    const viewportPadding = 12;
    const gap = 9;
    let left = anchorRect.left + anchorRect.width / 2 - tooltipRect.width / 2;
    left = Math.max(viewportPadding, Math.min(left, window.innerWidth - tooltipRect.width - viewportPadding));
    const fitsAbove = anchorRect.top - tooltipRect.height - gap >= viewportPadding;
    const placement = fitsAbove ? 'top' : 'bottom';
    const top = fitsAbove ? anchorRect.top - tooltipRect.height - gap : anchorRect.bottom + gap;
    const arrowLeft = Math.max(12, Math.min(anchorRect.left + anchorRect.width / 2 - left, tooltipRect.width - 12));
    tooltip.dataset.placement = placement;
    tooltip.style.setProperty('--kv2-tooltip-arrow-left', arrowLeft + 'px');
    tooltip.style.left = left + 'px';
    tooltip.style.top = top + 'px';
    anchor.setAttribute('aria-describedby', tooltip.id);
    clippedContentTooltip = tooltip;
    clippedContentAnchor = anchor;
  }

  function currentModel() { return models.find(function (item) { return item.id === state.modelId; }) || models[0]; }
  function getSources() {
    if (!sourceStore[state.modelId]) sourceStore[state.modelId] = baseSources.map(function (item) { return Object.assign({}, item); });
    return sourceStore[state.modelId];
  }
  function getAssets() {
    if (!assetStore[state.modelId]) assetStore[state.modelId] = seedAssets.map(function (item) { return Object.assign({}, item, { fields: Object.assign({}, item.fields) }); });
    return assetStore[state.modelId];
  }
  function nowText() {
    const d = new Date();
    return d.getFullYear() + '-' + String(d.getMonth() + 1).padStart(2, '0') + '-' + String(d.getDate()).padStart(2, '0') + ' ' + String(d.getHours()).padStart(2, '0') + ':' + String(d.getMinutes()).padStart(2, '0');
  }

  function semanticCategoryLabel(category) {
    if (category === 'business-rule') return '业务规则（结构化）';
    if (category === 'unstructured-rule') return '业务规则（非结构化）';
    if (category === 'standard-qa') return '标准问答（NL2SQL）';
    if (category === 'unstructured-qa') return '标准问答（RAG）';
    const item = categories.find(function (entry) { return entry.key === category; });
    return item ? item.name : category;
  }

  function semanticKindForCategory(category) {
    const map = {
      metric: 'metric', schema: 'table_description', 'business-rule': 'business_rule', 'unstructured-rule': 'business_rule',
      constraint: 'constraint', relationship: 'relationship', 'standard-qa': 'standard_qa', 'unstructured-qa': 'standard_qa', 'dynamic-query': 'dynamic_query'
    };
    return map[category] || category;
  }

  function semanticKeyForAsset(asset) {
    const f = asset && asset.fields || {};
    return String(asset && asset.category === 'schema' ? (f.table || f.assetId || '') : (f.assetId || '')).trim();
  }

  function semanticAssetIdentity(asset) {
    const domain = ['unstructured-rule', 'unstructured-qa'].includes(asset.category) ? 'rag' : 'nl2sql';
    return domain + ':' + semanticKindForCategory(asset.category) + ':' + semanticKeyForAsset(asset);
  }

  function optionalText(target, key, value) {
    const text = String(value == null ? '' : value).trim();
    if (text) target[key] = text;
  }

  function filterConditionToApi(condition) {
    const operator = condition.comparison || condition.operator || 'equals';
    const result = { type: 'condition', field: String(condition.field || ''), operator: operator };
    if (operator === 'is_null' || operator === 'is_not_null') return result;
    if (operator === 'between') result.value = [condition.value, condition.valueEnd];
    else if (operator === 'in' || operator === 'not_in') result.value = listValue(condition.value);
    else result.value = condition.value;
    return result;
  }

  // 条件构建器允许每条连接线分别选择 AND / OR；导出时转换成只含统一 group.logic 的标准 AST，
  // 并按 SQL 的 AND 优先级拆分为 OR 分组，避免丢失界面上配置的计算顺序。
  function filterNodeToApi(node) {
    if (!node) return null;
    if (node.type === 'condition') return filterConditionToApi(node);
    const sourceChildren = (node.conditions || []).filter(hasMeaningfulFilter);
    if (!sourceChildren.length) return null;
    const segments = [];
    let segment = [];
    sourceChildren.forEach(function (child, index) {
      const connector = index ? String(child.connector || node.logic || 'and').toLowerCase() : 'and';
      if (index && connector === 'or') {
        segments.push(segment);
        segment = [];
      }
      const converted = filterNodeToApi(child);
      if (converted) segment.push(converted);
    });
    if (segment.length) segments.push(segment);
    const normalizedSegments = segments.filter(function (items) { return items.length; }).map(function (items) {
      return items.length === 1 ? items[0] : { type: 'group', logic: 'and', conditions: items };
    });
    if (!normalizedSegments.length) return null;
    return normalizedSegments.length === 1 ? normalizedSegments[0] : { type: 'group', logic: 'or', conditions: normalizedSegments };
  }

  function filterNodeFromApi(node) {
    if (!node || typeof node !== 'object' || Array.isArray(node)) throw new Error('filter 必须是条件或条件组对象。');
    if (node.type === 'condition') {
      const operator = String(node.operator || 'equals');
      const condition = { type: 'condition', field: String(node.field || ''), comparison: operator };
      if (operator === 'between') {
        if (!Array.isArray(node.value) || node.value.length !== 2) throw new Error('between 的 value 必须包含两个值。');
        condition.value = node.value[0];
        condition.valueEnd = node.value[1];
      } else if (operator !== 'is_null' && operator !== 'is_not_null') {
        condition.value = (operator === 'in' || operator === 'not_in') ? listValue(node.value) : node.value;
      }
      return condition;
    }
    if (node.type !== 'group') throw new Error('filter.type 仅支持 group 或 condition。');
    const logic = String(node.logic || '').toLowerCase();
    if (!['and', 'or'].includes(logic)) throw new Error('filter.logic 仅支持 and 或 or。');
    if (!Array.isArray(node.conditions) || !node.conditions.length) throw new Error('条件组 conditions 不能为空。');
    return {
      type: 'group', logic: logic, conditions: node.conditions.map(function (child, index) {
        const converted = filterNodeFromApi(child);
        if (index) converted.connector = logic;
        return converted;
      })
    };
  }

  function filterRootFromApi(node) {
    const converted = filterNodeFromApi(node);
    return converted.type === 'group' ? converted : { type: 'group', logic: 'and', conditions: [converted] };
  }

  function formulaExpressionToTree(expression) {
    const text = String(expression || '');
    const tokens = [];
    let index = 0;
    while (index < text.length) {
      if (/\s/.test(text[index])) { index += 1; continue; }
      const rest = text.slice(index);
      const number = rest.match(/^(?:\d+(?:\.\d*)?|\.\d+)/);
      if (number) {
        tokens.push({ type: 'number', value: number[0] });
        index += number[0].length;
        continue;
      }
      const identifier = rest.match(/^[a-z][a-z0-9_-]*/);
      if (identifier) {
        tokens.push({ type: 'metric', value: identifier[0] });
        index += identifier[0].length;
        continue;
      }
      const char = text[index];
      if ('+-*/()'.includes(char)) tokens.push({ type: char, value: char });
      else throw new Error('派生指标公式包含不支持的字符。');
      index += 1;
    }
    let position = 0;
    const operationNames = { '+': 'add', '-': 'subtract', '*': 'multiply', '/': 'divide' };
    const parsePrimary = function () {
      const token = tokens[position];
      if (!token) throw new Error('派生指标公式未完成。');
      if (token.type === 'number') { position += 1; return { type: 'constant', value: Number(token.value) }; }
      if (token.type === 'metric') { position += 1; return { type: 'metric_ref', metric_key: token.value }; }
      if (token.type === '(') {
        position += 1;
        const value = parseExpression();
        if (!tokens[position] || tokens[position].type !== ')') throw new Error('派生指标公式缺少右括号。');
        position += 1;
        return value;
      }
      throw new Error('派生指标公式中存在无效的运算位置。');
    };
    const parseUnary = function () {
      if (tokens[position] && tokens[position].type === '-') {
        position += 1;
        return { type: 'operation', operator: 'subtract', left: { type: 'constant', value: 0 }, right: parseUnary() };
      }
      return parsePrimary();
    };
    const parseTerm = function () {
      let left = parseUnary();
      while (tokens[position] && (tokens[position].type === '*' || tokens[position].type === '/')) {
        const operator = tokens[position].type;
        position += 1;
        left = { type: 'operation', operator: operationNames[operator], left: left, right: parseUnary() };
      }
      return left;
    };
    const parseExpression = function () {
      let left = parseTerm();
      while (tokens[position] && (tokens[position].type === '+' || tokens[position].type === '-')) {
        const operator = tokens[position].type;
        position += 1;
        left = { type: 'operation', operator: operationNames[operator], left: left, right: parseTerm() };
      }
      return left;
    };
    if (!tokens.length) throw new Error('派生指标缺少计算公式。');
    const tree = parseExpression();
    if (position !== tokens.length) throw new Error('派生指标公式结构不正确。');
    return tree;
  }

  function formulaTreeToExpression(node) {
    if (!node || typeof node !== 'object' || Array.isArray(node)) throw new Error('calculation.formula 必须是公式节点对象。');
    if (node.type === 'metric_ref') {
      if (!node.metric_key) throw new Error('metric_ref 缺少 metric_key。');
      return String(node.metric_key);
    }
    if (node.type === 'constant') {
      if (typeof node.value !== 'number' || !Number.isFinite(node.value)) throw new Error('constant.value 必须是数字。');
      return String(node.value);
    }
    if (node.type !== 'operation') throw new Error('公式节点 type 仅支持 operation、metric_ref 或 constant。');
    const symbols = { add: '+', subtract: '-', multiply: '*', divide: '/' };
    if (!symbols[node.operator]) throw new Error('公式 operator 仅支持 add、subtract、multiply 或 divide。');
    return '(' + formulaTreeToExpression(node.left) + ' ' + symbols[node.operator] + ' ' + formulaTreeToExpression(node.right) + ')';
  }

  function semanticAssetToApi(asset) {
    const f = asset.fields || {};
    const description = f.description || asset.desc || '';
    if (asset.category === 'metric') {
      const metricType = f.metricType === 'derived' ? 'derived' : 'aggregate';
      const result = {
        kind: 'metric', metric_key: String(f.assetId || ''), name: String(f.name || asset.name || f.assetId || ''),
        metric_type: metricType, calculation: {}
      };
      optionalText(result, 'description', description);
      const synonyms = listValue(f.synonyms);
      if (synonyms.length) result.synonyms = synonyms;
      if (metricType === 'derived') {
        result.source = { table: String(f.sourceTable || '') };
        result.calculation.formula = formulaExpressionToTree(f.metricFormulaExpression || f.formula || '');
      } else {
        result.source = { table: String(f.sourceTable || '') };
        result.calculation.aggregation = {
          function: String(f.aggregationFunction || f.aggregation || 'sum').toLowerCase(),
          field: String(f.aggregationField || f.field || '')
        };
        const filter = filterNodeToApi(f.filterGroup || defaultFilterGroup('metric', f));
        if (filter) result.calculation.filter = filter;
        const multiplier = Number(f.multiplier == null || f.multiplier === '' ? 1 : f.multiplier);
        result.calculation.multiplier = Number.isFinite(multiplier) ? multiplier : 1;
      }
      if (f.output) {
        result.output = { format: String(f.output) };
        if (f.decimalPlaces != null && f.decimalPlaces !== '') result.output.decimal_places = Number(f.decimalPlaces);
      }
      return result;
    }
    if (asset.category === 'business-rule') {
      const result = {
        kind: 'business_rule', rule_key: String(f.assetId || ''),
        tables: listValue(f.relatedTables || f.sourceTable || asset.binding),
        dynamic_query_keys: listValue(f.dynamicQueryKeys), rule: String(f.ruleContent || f.content || description || '')
      };
      return result;
    }
    if (asset.category === 'constraint') {
      const constraintFilter = filterNodeToApi(f.filterGroup || defaultFilterGroup('constraint', f));
      if (!constraintFilter) throw new Error('强制约束“' + (f.assetId || asset.name || '') + '”缺少 filter。');
      const result = {
        kind: 'constraint', constraint_key: String(f.assetId || ''), table: String(f.sourceTable || ''),
        filter: constraintFilter
      };
      optionalText(result, 'description', description);
      return result;
    }
    if (asset.category === 'relationship') {
      const joinType = String(f.joinType || 'LEFT JOIN').replace(/\s+JOIN$/i, '').toLowerCase();
      const result = {
        kind: 'relationship', relationship_key: String(f.assetId || ''),
        left_table: String(f.leftTable || ''), right_table: String(f.rightTable || ''), join_type: joinType,
        join_conditions: (f.joinConditions || [{ leftField: f.leftField, comparison: 'equals', rightField: f.rightField }]).map(function (condition) {
          const comparison = relationshipComparisons.some(function (item) { return item.value === condition.comparison; }) ? condition.comparison : 'equals';
          return { left_field: String(condition.leftField || ''), operator: comparison, right_field: String(condition.rightField || '') };
        })
      };
      optionalText(result, 'description', description);
      return result;
    }
    if (asset.category === 'standard-qa') {
      const result = {
        kind: 'standard_qa', qa_key: String(f.assetId || ''), usage_mode: f.usageMode === 'force' ? 'force' : 'reference',
        question: String(f.question || ''), sql: String(f.sql || '')
      };
      optionalText(result, 'description', description);
      const questions = listValue(f.similarQuestions);
      if (questions.length) result.similar_questions = questions;
      return result;
    }
    if (asset.category === 'unstructured-qa') {
      const result = {
        kind: 'standard_qa', qa_key: String(f.assetId || ''), usage_mode: f.usageMode === 'force' ? 'force' : 'reference',
        question: String(f.question || ''), answer: String(f.answer || ''), files: listValue(f.relatedFiles || asset.binding)
      };
      optionalText(result, 'description', description);
      const questions = listValue(f.similarQuestions);
      if (questions.length) result.similar_questions = questions;
      return result;
    }
    if (asset.category === 'dynamic-query') {
      const result = {
        kind: 'dynamic_query', query_key: String(f.assetId || ''), description: String(description || ''),
        result_processing_mode: f.resultProcessingMode === 'delegated_analysis' ? 'delegated_analysis' : 'direct_context', sql: String(f.sql || '')
      };
      return result;
    }
    throw new Error('不支持导出语义类型：' + asset.category);
  }

  function schemaAssetToSemanticItems(asset) {
    const f = asset.fields || {};
    const columns = Array.isArray(f.schemaColumns) ? f.schemaColumns : bpcColumns.map(function (column) {
      return { name: column.name, supplementalDescription: '', enabled: true };
    });
    const table = String(f.table || f.assetId || '');
    const tableDescription = { kind: 'table_description', table: table };
    optionalText(tableDescription, 'description', f.description || asset.desc || '');
    return [tableDescription].concat(columns.map(function (column) {
      const fieldDescription = {
        kind: 'field_description', table: table, field: String(column.name || ''),
        nl2sql_enabled: column.enabled !== false
      };
      optionalText(fieldDescription, 'description', column.supplementalDescription || '');
      return fieldDescription;
    }));
  }

  function unstructuredRuleAssetToApi(asset) {
    const f = asset.fields || {};
    return {
      rule_key: String(f.assetId || ''), files: listValue(f.relatedFiles || asset.binding),
      rule: String(f.ruleContent || f.content || f.description || asset.desc || '')
    };
  }

  function semanticBinding(category, fields) {
    const f = fields || {};
    if (category === 'metric') return f.sourceTable || '—';
    if (category === 'schema') return f.table || f.assetId;
    if (category === 'business-rule') return listValue(f.relatedTables).join(', ') || f.assetId;
    if (category === 'unstructured-rule') return listValue(f.relatedFiles).join(', ') || f.assetId;
    if (category === 'unstructured-qa') return listValue(f.relatedFiles).join(', ') || f.assetId;
    if (category === 'constraint') return f.sourceTable || f.assetId;
    if (category === 'relationship') return (f.leftTable || '') + '.' + (f.leftField || '') + ' = ' + (f.rightTable || '') + '.' + (f.rightField || '');
    if (category === 'standard-qa') return f.usageMode === 'force' ? '高置信度语义一致时强制使用' : '仅供参考';
    if (category === 'dynamic-query') return f.resultProcessingMode === 'delegated_analysis' ? '子 Agent 分析' : '直接注入上下文';
    return f.assetId || '';
  }

  function semanticApiIdentity(item) {
    const keys = {
      metric: 'metric_key', business_rule: 'rule_key', constraint: 'constraint_key',
      relationship: 'relationship_key', standard_qa: 'qa_key', dynamic_query: 'query_key'
    };
    if (item.kind === 'table_description') return 'table_description:' + String(item.table || '');
    if (item.kind === 'field_description') return 'field_description:' + String(item.table || '') + '.' + String(item.field || '');
    const keyField = keys[item.kind];
    return item.kind + ':' + String(keyField ? item[keyField] || '' : '');
  }

  function semanticApiToAsset(item, semanticDomain) {
    const kind = String(item.kind || '');
    let category = kind;
    let fields = {};
    let name = '';
    let description = String(item.description || '');
    if (kind === 'metric') {
      if (!item.metric_key || !item.name || !['aggregate', 'derived'].includes(item.metric_type) || !item.calculation) throw new Error('metric 缺少 metric_key、name、metric_type 或 calculation。');
      fields = {
        assetId: String(item.metric_key), name: String(item.name), description: description,
        synonyms: listValue(item.synonyms), metricType: item.metric_type
      };
      if (item.metric_type === 'derived') {
        if (!item.source || !item.source.table) throw new Error('derived 指标缺少 source.table。');
        fields.sourceTable = String(item.source.table);
        fields.metricFormulaExpression = formulaTreeToExpression(item.calculation.formula);
        fields.formula = fields.metricFormulaExpression;
      } else {
        if (!item.source || !item.source.table || !item.calculation.aggregation || !item.calculation.aggregation.field) throw new Error('aggregate 指标缺少 source.table 或 calculation.aggregation。');
        fields.sourceTable = String(item.source.table);
        fields.aggregationFunction = String(item.calculation.aggregation.function || 'sum').toLowerCase();
        fields.aggregation = fields.aggregationFunction.toUpperCase();
        fields.aggregationField = String(item.calculation.aggregation.field);
        fields.field = fields.aggregationField;
        fields.multiplier = item.calculation.multiplier == null ? 1 : item.calculation.multiplier;
        fields.filterGroup = item.calculation.filter ? filterRootFromApi(item.calculation.filter) : { type: 'group', logic: 'and', conditions: [] };
      }
      if (item.output && typeof item.output === 'object') {
        fields.output = item.output.format || item.output.type || '';
        if (item.output.decimal_places != null) fields.decimalPlaces = item.output.decimal_places;
      }
      category = 'metric';
      name = fields.name;
    } else if (kind === 'business_rule') {
      if (!item.rule_key || !String(item.rule || '').trim()) throw new Error('business_rule 缺少 rule_key 或 rule。');
      const ragRule = semanticDomain === 'rag';
      if (ragRule) {
        if (!Array.isArray(item.files) || !item.files.length) throw new Error('RAG business_rule 缺少 files。');
        fields = {
          assetId: String(item.rule_key), name: String(item.rule_key), relatedFiles: listValue(item.files),
          ruleContent: String(item.rule), content: String(item.rule), description: String(item.rule)
        };
        category = 'unstructured-rule';
        name = String(item.rule_key);
        description = String(item.rule);
        return { category: category, name: name, binding: semanticBinding(category, fields), desc: description, fields: fields };
      }
      if (item.tables != null && !Array.isArray(item.tables)) throw new Error('business_rule.tables 必须是数组。');
      if (item.dynamic_query_keys != null && !Array.isArray(item.dynamic_query_keys)) throw new Error('business_rule.dynamic_query_keys 必须是数组。');
      fields = {
        assetId: String(item.rule_key), name: String(item.rule_key), ruleContent: String(item.rule), content: String(item.rule),
        description: String(item.rule), dynamicQueryKeys: listValue(item.dynamic_query_keys),
        relatedTables: item.tables == null ? tableMetadataOptions().map(function (option) { return option.value; }) : listValue(item.tables)
      };
      category = 'business-rule';
      name = String(item.rule_key);
      description = String(item.rule);
    } else if (kind === 'constraint') {
      if (!item.constraint_key || typeof item.table !== 'string' || !item.table.trim() || !item.filter) throw new Error('constraint 缺少 constraint_key、单个 table 字符串或 filter。');
      fields = {
        assetId: String(item.constraint_key), name: String(item.constraint_key), description: description,
        sourceTable: String(item.table), filterGroup: filterRootFromApi(item.filter)
      };
      category = 'constraint';
      name = String(item.constraint_key);
    } else if (kind === 'relationship') {
      if (!item.relationship_key || typeof item.left_table !== 'string' || !item.left_table.trim() || typeof item.right_table !== 'string' || !item.right_table.trim() || !item.join_type || !Array.isArray(item.join_conditions) || !item.join_conditions.length) throw new Error('relationship 缺少 key、左右表、JOIN 类型或关联条件。');
      const joinType = String(item.join_type).toLowerCase();
      if (!['left', 'inner', 'right', 'full'].includes(joinType)) throw new Error('relationship.join_type 仅支持 left、inner、right 或 full。');
      const joinConditions = item.join_conditions.map(function (condition) {
        const comparison = String(condition && condition.operator || 'equals');
        if (!condition || !condition.left_field || !condition.right_field || !relationshipComparisons.some(function (item) { return item.value === comparison; })) throw new Error('join_conditions 的 operator 仅支持 equals、not_equals、greater_than、greater_than_or_equal、less_than 或 less_than_or_equal。');
        return { leftField: String(condition.left_field), comparison: comparison, rightField: String(condition.right_field) };
      });
      fields = {
        assetId: String(item.relationship_key), name: String(item.relationship_key), description: description,
        leftTable: String(item.left_table), rightTable: String(item.right_table), joinType: joinType.toUpperCase() + ' JOIN',
        joinConditions: joinConditions, leftField: joinConditions[0].leftField, rightField: joinConditions[0].rightField
      };
      category = 'relationship';
      name = String(item.relationship_key);
    } else if (kind === 'standard_qa') {
      const documentQa = semanticDomain === 'rag';
      if (!item.qa_key || !String(item.question || '').trim()) throw new Error('standard_qa 缺少 qa_key 或 question。');
      if (documentQa && (!String(item.answer || '').trim() || !Array.isArray(item.files) || !item.files.length)) throw new Error('非结构化 standard_qa 缺少 answer 或 files。');
      if (!documentQa && !String(item.sql || '').trim()) throw new Error('结构化 standard_qa 缺少 sql。');
      const usageMode = item.usage_mode == null ? 'reference' : String(item.usage_mode);
      if (!['reference', 'force'].includes(usageMode)) throw new Error('usage_mode 仅支持 reference 或 force。');
      fields = {
        assetId: String(item.qa_key), name: String(item.question), usageMode: usageMode, description: description,
        question: String(item.question), similarQuestions: listValue(item.similar_questions)
      };
      if (documentQa) {
        fields.answer = String(item.answer);
        fields.relatedFiles = listValue(item.files);
        category = 'unstructured-qa';
      } else {
        fields.sql = String(item.sql);
        category = 'standard-qa';
      }
      name = String(item.question);
    } else if (kind === 'dynamic_query') {
      if (!item.query_key || !String(item.description || '').trim() || !String(item.sql || '').trim()) throw new Error('dynamic_query 缺少 query_key、description 或 sql。');
      const processingMode = item.result_processing_mode == null ? 'direct_context' : String(item.result_processing_mode);
      if (!['direct_context', 'delegated_analysis'].includes(processingMode)) throw new Error('result_processing_mode 仅支持 direct_context 或 delegated_analysis。');
      fields = {
        assetId: String(item.query_key), name: String(item.query_key), description: description,
        resultProcessingMode: processingMode, sql: String(item.sql)
      };
      category = 'dynamic-query';
      name = String(item.query_key);
    } else {
      throw new Error('kind 不受支持：' + (kind || '未填写'));
    }
    return { category: category, name: name, binding: semanticBinding(category, fields), desc: description, fields: fields };
  }

  function tableSemanticApiToAsset(item) {
    if (!item || typeof item !== 'object' || Array.isArray(item)) throw new Error('表和字段说明中存在格式不正确的配置。');
    if (typeof item.table !== 'string' || !item.table.trim()) throw new Error('table 必须是非空字符串。');
    if (item.fields != null && !Array.isArray(item.fields)) throw new Error('fields 必须是数组。');
    const importedFields = item.fields || [];
    importedFields.forEach(function (field) {
      if (!field || !field.field) throw new Error('字段说明中存在缺少 field 的配置。');
    });
    const schemaColumns = bpcColumns.map(function (metadata) {
      const field = importedFields.find(function (entry) { return String(entry.field) === metadata.name; });
      return {
        name: metadata.name, description: metadata.description,
        supplementalDescription: String(field && field.description || ''), enabled: !field || field.nl2sql_enabled !== false
      };
    }).concat(importedFields.filter(function (field) { return !bpcColumns.some(function (metadata) { return metadata.name === String(field.field); }); }).map(function (field) {
      return {
        name: String(field.field), description: '', supplementalDescription: String(field.description || ''),
        enabled: field.nl2sql_enabled !== false
      };
    }));
    const description = String(item.description || '');
    const fields = {
      assetId: String(item.table), table: String(item.table), name: String(item.table), description: description,
      schemaColumns: schemaColumns
    };
    return { category: 'schema', name: String(item.table), binding: String(item.table), desc: description, fields: fields };
  }

  function schemaSemanticItemsToAssets(items) {
    const tables = new Map();
    items.forEach(function (item) {
      const table = String(item.table || '').trim();
      if (!table) throw new Error(item.kind + '.table 必须是非空字符串。');
      if (!tables.has(table)) tables.set(table, { table: table, description: '', fields: [], hasTableDescription: false, fieldNames: new Set() });
      const entry = tables.get(table);
      if (item.kind === 'table_description') {
        if (entry.hasTableDescription) throw new Error('同一张表只能配置一条 table_description：' + table);
        entry.hasTableDescription = true;
        entry.description = String(item.description || '');
        return;
      }
      const field = String(item.field || '').trim();
      if (!field) throw new Error('field_description.field 必须是非空字符串。');
      if (entry.fieldNames.has(field)) throw new Error('字段说明重复：' + table + '.' + field);
      if (item.nl2sql_enabled != null && typeof item.nl2sql_enabled !== 'boolean') throw new Error('field_description.nl2sql_enabled 必须是布尔值。');
      entry.fieldNames.add(field);
      entry.fields.push({
        field: field, description: String(item.description || ''),
        nl2sql_enabled: item.nl2sql_enabled !== false
      });
    });
    return Array.from(tables.values()).map(function (entry) { return tableSemanticApiToAsset(entry); });
  }

  function unstructuredRuleApiToAsset(item) {
    if (!item || typeof item !== 'object' || Array.isArray(item)) throw new Error('unstructured_rules 中存在格式不正确的配置。');
    if (!item.rule_key || !/^[a-z_-]+$/.test(String(item.rule_key)) || String(item.rule_key).length > 50) throw new Error('unstructured_rules.rule_key 格式不正确。');
    if (!Array.isArray(item.files) || !item.files.length || !String(item.rule || '').trim()) throw new Error('unstructured_rules 需要填写 files 和 rule。');
    const fields = {
      assetId: String(item.rule_key), name: String(item.rule_key), relatedFiles: listValue(item.files),
      ruleContent: String(item.rule), content: String(item.rule), description: String(item.rule)
    };
    return {
      category: 'unstructured-rule', name: String(item.rule_key), binding: fields.relatedFiles.join(', '),
      desc: String(item.rule), fields: fields
    };
  }

  function semanticExportPayload() {
    const model = currentModel();
    const assets = getAssets();
    const nl2sqlSemantics = [];
    const ragSemantics = [];
    assets.forEach(function (asset) {
      if (asset.category === 'schema') nl2sqlSemantics.push.apply(nl2sqlSemantics, schemaAssetToSemanticItems(asset));
      else if (asset.category === 'unstructured-rule') ragSemantics.push(Object.assign({ kind: 'business_rule' }, unstructuredRuleAssetToApi(asset)));
      else if (asset.category === 'unstructured-qa') ragSemantics.push(semanticAssetToApi(asset));
      else nl2sqlSemantics.push(semanticAssetToApi(asset));
    });
    return {
      format: 'moi.knowledge.semantics',
      schema_version: '2.0',
      exported_at: new Date().toISOString(),
      knowledge_base: { id: String(model.id), name: model.name },
      semantics: { nl2sql: nl2sqlSemantics, rag: ragSemantics }
    };
  }

  function exportAllSemantics() {
    const payload = semanticExportPayload();
    const blob = new Blob([JSON.stringify(payload, null, 2)], { type: 'application/json;charset=utf-8' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    const date = new Date().toISOString().slice(0, 10).replaceAll('-', '');
    const modelName = String(currentModel().name || 'knowledge-base').replace(/[\\/:*?"<>|]/g, '-');
    link.href = url;
    link.download = modelName + '-语义-' + date + '.json';
    link.click();
    setTimeout(function () { URL.revokeObjectURL(url); }, 0);
  }

  function semanticImportExamplePayload() {
    return {
      format: 'moi.knowledge.semantics',
      schema_version: '2.0',
      exported_at: '2026-08-12T10:00:00.000Z',
      knowledge_base: { id: '1', name: 'BPC' },
      semantics: { nl2sql: [
        {
          kind: 'table_description', table: 'jst_flat_table.bpc_consolidated_report',
          description: 'BPC 合并报表明细数据'
        },
        {
          kind: 'field_description', table: 'jst_flat_table.bpc_consolidated_report', field: 'b28_s_sdata',
          description: '金额，单位为元', nl2sql_enabled: true
        },
        {
          kind: 'metric', metric_key: 'bpc_revenue', name: '营业收入', metric_type: 'aggregate',
          source: { table: 'jst_flat_table.bpc_consolidated_report' },
          calculation: {
            aggregation: { function: 'sum', field: 'b28_s_sdata' },
            filter: { type: 'condition', field: 'account_path', operator: 'contains', value: '/APL6000/' },
            multiplier: -1
          }
        },
        {
          kind: 'business_rule', rule_key: 'annual_opening_period',
          tables: ['jst_flat_table.bpc_consolidated_report'], dynamic_query_keys: [],
          rule: '当用户查询年度期初数据时，使用上一年度年末期间的数据。'
        },
        {
          kind: 'constraint', constraint_key: 'bpc_currency_cny',
          table: 'jst_flat_table.bpc_consolidated_report',
          filter: { type: 'condition', field: 'b28_s_kgd4kbn', operator: 'equals', value: 'CNY' }
        },
        {
          kind: 'relationship', relationship_key: 'bpc_organization', left_table: 'bpc', right_table: 'organization_master',
          join_type: 'left', join_conditions: [{ left_field: 'organization_code', operator: 'equals', right_field: 'organization_code' }]
        },
        {
          kind: 'standard_qa', qa_key: 'bpc_revenue_eo_may', usage_mode: 'reference',
          question: 'EO_1000 公司 2025 年 5 月营业收入是多少', similar_questions: ['查询 EO_1000 公司 2025 年 5 月营收'],
          sql: "SELECT SUM(b28_s_sdata) FROM bpc_consolidated_report WHERE organization_code = 'EO_1000'"
        },
        {
          kind: 'dynamic_query', query_key: 'bpc_account_mapping',
          description: '查询科目编码、名称和父级编码，供业务规则识别科目取数范围。',
          result_processing_mode: 'direct_context', sql: 'SELECT code, name, parent_code FROM account_mapping'
        }
      ], rag: [
        {
          kind: 'business_rule', rule_key: 'bpc_document_metric_policy', files: ['BPC_指标口径说明.pdf'],
          rule: '当问题涉及文档中定义的指标口径和使用说明时，优先依据该文件判断。'
        },
        {
          kind: 'standard_qa', qa_key: 'bpc_revenue_definition', usage_mode: 'reference',
          description: '说明 BPC 营业收入指标的标准取数口径。',
          question: '营业收入的指标口径是什么', similar_questions: ['营业收入怎么计算'],
          answer: '营业收入按合并科目 APL6000 取数，汇总后需要置反。', files: ['BPC_指标口径说明.pdf']
        }
      ] }
    };
  }

  function downloadSemanticImportExample() {
    const blob = new Blob([JSON.stringify(semanticImportExamplePayload(), null, 2)], { type: 'application/json;charset=utf-8' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = '语义导入示例.json';
    link.click();
    setTimeout(function () { URL.revokeObjectURL(url); }, 0);
  }

  function normalizeSemanticImport(payload, fileName) {
    const supportedFormats = new Set(['moi.knowledge.semantics', 'moi.nl2sql.semantics']);
    if (payload && payload.format && !supportedFormats.has(payload.format)) throw new Error('format 不受支持，应为 moi.knowledge.semantics。');
    const version = payload && (payload.schema_version || payload.schemaVersion) || '1.0';
    if (!payload || typeof payload !== 'object' || Array.isArray(payload)) throw new Error('JSON 顶层必须是对象。');
    const versionMajor = String(version).split('.')[0];
    if (!['1', '2'].includes(versionMajor)) throw new Error('不支持 schema_version ' + version + '，当前支持 1.x 和 2.x。');
    const semanticsValue = payload.semantics == null ? [] : payload.semantics;
    const groupedSemantics = semanticsValue && typeof semanticsValue === 'object' && !Array.isArray(semanticsValue);
    if (versionMajor === '2' && !groupedSemantics) throw new Error('schema_version 2.x 的 semantics 必须包含 nl2sql 和 rag 两组。');
    if (groupedSemantics && (!Array.isArray(semanticsValue.nl2sql) || !Array.isArray(semanticsValue.rag))) throw new Error('semantics.nl2sql 和 semantics.rag 必须是数组。');
    const nl2sqlSource = groupedSemantics ? semanticsValue.nl2sql : semanticsValue;
    const ragSource = groupedSemantics ? semanticsValue.rag : [];
    const source = nl2sqlSource.concat(ragSource);
    function inferLegacySemanticDomain(item) {
      const kind = item && String(item.kind || '');
      if (kind === 'business_rule' && item.files != null) return 'rag';
      if (kind === 'standard_qa' && (item.answer != null || item.files != null)) return 'rag';
      return 'nl2sql';
    }
    const sourceDomains = nl2sqlSource.map(function (item) {
      return groupedSemantics ? 'nl2sql' : inferLegacySemanticDomain(item);
    }).concat(ragSource.map(function () { return 'rag'; }));
    // 兼容旧版导出文件；新版表和字段说明均作为带 kind 的语义写入 semantics。
    const tableSource = payload.table_semantics == null ? [] : payload.table_semantics;
    const unstructuredSource = payload.unstructured_rules == null ? [] : payload.unstructured_rules;
    if (!Array.isArray(nl2sqlSource)) throw new Error('semantics.nl2sql 必须是数组。');
    if (!Array.isArray(ragSource)) throw new Error('semantics.rag 必须是数组。');
    if (!Array.isArray(tableSource)) throw new Error('table_semantics 必须是数组。');
    if (!Array.isArray(unstructuredSource)) throw new Error('unstructured_rules 必须是数组。');
    if (!source.length && !tableSource.length && !unstructuredSource.length) throw new Error('JSON 文件中没有可导入的语义数据。');
    const seen = new Set();
    const validKinds = new Set(['table_description', 'field_description', 'metric', 'business_rule', 'constraint', 'relationship', 'standard_qa', 'dynamic_query']);
    const domainKinds = {
      nl2sql: new Set(['table_description', 'field_description', 'metric', 'business_rule', 'constraint', 'relationship', 'standard_qa', 'dynamic_query']),
      rag: new Set(['business_rule', 'standard_qa'])
    };
    const schemaItems = [];
    const semanticAssets = source.map(function (item, index) {
      if (!item || typeof item !== 'object' || Array.isArray(item)) throw new Error('第 ' + (index + 1) + ' 条语义格式不正确。');
      if (!validKinds.has(String(item.kind || ''))) throw new Error('第 ' + (index + 1) + ' 条语义的 kind 不受支持：' + (item.kind || '未填写'));
      const semanticDomain = sourceDomains[index];
      if (!domainKinds[semanticDomain].has(String(item.kind || ''))) throw new Error('第 ' + (index + 1) + ' 条语义的 kind 不属于 ' + semanticDomain.toUpperCase() + ' 语义：' + item.kind);
      if (semanticDomain === 'rag' && item.kind === 'business_rule' && item.tables != null) throw new Error('第 ' + (index + 1) + ' 条 RAG business_rule 应填写 files，不应填写 tables。');
      if (semanticDomain === 'nl2sql' && item.kind === 'business_rule' && item.files != null) throw new Error('第 ' + (index + 1) + ' 条 NL2SQL business_rule 应填写 tables，不应填写 files。');
      if (semanticDomain === 'rag' && item.kind === 'standard_qa' && item.sql != null) throw new Error('第 ' + (index + 1) + ' 条 RAG standard_qa 应填写 answer 和 files，不应填写 sql。');
      if (semanticDomain === 'nl2sql' && item.kind === 'standard_qa' && (item.answer != null || item.files != null)) throw new Error('第 ' + (index + 1) + ' 条 NL2SQL standard_qa 应填写 sql，不应填写 answer 或 files。');
      const identity = semanticDomain + ':' + semanticApiIdentity(item);
      const schemaKind = item.kind === 'table_description' || item.kind === 'field_description';
      if (schemaKind) {
        if (!String(item.table || '').trim() || (item.kind === 'field_description' && !String(item.field || '').trim())) throw new Error('第 ' + (index + 1) + ' 条语义缺少 table 或 field。');
      } else {
        if (!String(item.kind || '').trim() || identity.endsWith(':')) throw new Error('第 ' + (index + 1) + ' 条语义缺少 kind 或稳定标识。');
        const stableKey = semanticApiIdentity(item).split(':').slice(1).join(':');
        if (!/^[a-z_-]+$/.test(stableKey) || stableKey.length > 50) throw new Error('第 ' + (index + 1) + ' 条语义的标识仅支持英文小写字母、下划线（_）和横线（-），且不能超过 50 个字符。');
      }
      if (seen.has(identity)) throw new Error('JSON 文件内存在重复语义：' + semanticApiIdentity(item).split(':').slice(1).join(':'));
      seen.add(identity);
      if (schemaKind) { schemaItems.push(item); return null; }
      try { return semanticApiToAsset(item, semanticDomain); }
      catch (error) { throw new Error('第 ' + (index + 1) + ' 条语义：' + error.message); }
    }).filter(Boolean);
    tableSource.forEach(function (item, index) {
      let legacyItems = [];
      try { legacyItems = schemaAssetToSemanticItems(tableSemanticApiToAsset(item)); }
      catch (error) { throw new Error('旧版 table_semantics 第 ' + (index + 1) + ' 条：' + error.message); }
      legacyItems.forEach(function (legacyItem) {
        const identity = 'nl2sql:' + semanticApiIdentity(legacyItem);
        if (seen.has(identity)) throw new Error('JSON 文件内存在重复表或字段说明：' + identity.split(':').slice(1).join(':'));
        seen.add(identity);
        schemaItems.push(legacyItem);
      });
    });
    let schemaAssets = [];
    try { schemaAssets = schemaSemanticItemsToAssets(schemaItems); }
    catch (error) { throw new Error('表和字段说明：' + error.message); }
    const unstructuredAssets = unstructuredSource.map(function (item, index) {
      const identity = 'rag:business_rule:' + String(item && item.rule_key || '');
      if (seen.has(identity)) throw new Error('unstructured_rules 中存在重复规则：' + String(item && item.rule_key || ''));
      seen.add(identity);
      try { return unstructuredRuleApiToAsset(item); }
      catch (error) { throw new Error('第 ' + (index + 1) + ' 条非结构化规则：' + error.message); }
    });
    const assets = schemaAssets.concat(semanticAssets, unstructuredAssets);
    const currentKeys = new Set(getAssets().map(semanticAssetIdentity));
    const categoryCounts = {};
    const domainCounts = { nl2sql: 0, rag: 0 };
    let conflicts = 0;
    assets.forEach(function (asset) {
      categoryCounts[asset.category] = (categoryCounts[asset.category] || 0) + 1;
      if (['unstructured-rule', 'unstructured-qa'].includes(asset.category)) domainCounts.rag += 1;
      else domainCounts.nl2sql += 1;
      if (currentKeys.has(semanticAssetIdentity(asset))) conflicts += 1;
    });
    return { status: 'ready', fileName: fileName, schemaVersion: String(version), assets: assets, conflicts: conflicts, additions: assets.length - conflicts, categoryCounts: categoryCounts, domainCounts: domainCounts };
  }

  function loadSemanticImportFile(file) {
    if (!file) return;
    file.text().then(function (text) {
      try {
        state.semanticImport = normalizeSemanticImport(JSON.parse(text), file.name);
      } catch (error) {
        state.semanticImport = { status: 'error', fileName: file.name, error: error && error.message ? error.message : 'JSON 文件解析失败。' };
      }
      state.modal = 'semantic-import';
      render();
    }).catch(function () {
      state.semanticImport = { status: 'error', fileName: file.name, error: '无法读取所选文件，请重新选择。' };
      state.modal = 'semantic-import';
      render();
    });
  }

  function importAllSemantics() {
    const preview = state.semanticImport;
    if (!preview || preview.status !== 'ready') return;
    const mode = state.semanticImportMode === 'overwrite' ? 'overwrite' : 'append';
    const current = getAssets();
    const importedAssets = preview.assets.map(function (asset) {
      return Object.assign({}, asset, { updated: nowText(), fields: JSON.parse(JSON.stringify(asset.fields || {})) });
    });
    if (mode === 'overwrite') {
      importedAssets.forEach(function (asset) { asset.id = nextAssetId++; });
      assetStore[state.modelId] = importedAssets;
    } else {
      const existingIdentities = new Set(current.map(semanticAssetIdentity));
      importedAssets.forEach(function (asset) {
        const identity = semanticAssetIdentity(asset);
        if (!existingIdentities.has(identity)) {
          asset.id = nextAssetId++;
          current.push(asset);
          existingIdentities.add(identity);
        }
      });
    }
    state.semanticImport = null;
    state.modal = null;
    state.semQuery = '';
    state.semanticPage = 1;
    render();
  }

  function restoreDeletedDemo(modelId) {
    const restoreAll = modelId == null;
    const modelEntries = state.deletedDemo.models.filter(function (entry) { return restoreAll || entry.item.id === modelId; });
    for (let i = modelEntries.length - 1; i >= 0; i -= 1) {
      const entry = modelEntries[i];
      if (!models.some(function (model) { return model.id === entry.item.id; })) models.splice(Math.min(entry.index, models.length), 0, entry.item);
    }
    state.deletedDemo.models = state.deletedDemo.models.filter(function (entry) { return !restoreAll && entry.item.id !== modelId; });

    const sourceEntries = state.deletedDemo.sources.filter(function (entry) { return restoreAll || entry.modelId === modelId; });
    for (let i = sourceEntries.length - 1; i >= 0; i -= 1) {
      const entry = sourceEntries[i];
      const collection = sourceStore[entry.modelId] || (sourceStore[entry.modelId] = []);
      if (!collection.some(function (source) { return source.id === entry.item.id; })) collection.splice(Math.min(entry.index, collection.length), 0, entry.item);
    }
    state.deletedDemo.sources = state.deletedDemo.sources.filter(function (entry) { return !restoreAll && entry.modelId !== modelId; });

    const assetEntries = state.deletedDemo.assets.filter(function (entry) { return restoreAll || entry.modelId === modelId; });
    for (let i = assetEntries.length - 1; i >= 0; i -= 1) {
      const entry = assetEntries[i];
      const collection = assetStore[entry.modelId] || (assetStore[entry.modelId] = []);
      if (!collection.some(function (asset) { return asset.id === entry.item.id; })) collection.splice(Math.min(entry.index, collection.length), 0, entry.item);
    }
    state.deletedDemo.assets = state.deletedDemo.assets.filter(function (entry) { return !restoreAll && entry.modelId !== modelId; });

    models.forEach(function (model) {
      const sources = sourceStore[model.id];
      if (!sources) return;
      model.files = sources.filter(function (source) { return source.type === '文件'; }).length;
      model.tables = sources.filter(function (source) { return source.type === '数据表'; }).length;
    });
  }

  function render(options) {
    const modalBody = options && options.preserveModalScroll ? app.querySelector('.kv2-modal-body') : null;
    const modalScrollTop = modalBody ? modalBody.scrollTop : null;
    document.body.classList.toggle('kv2-modal-open', Boolean(state.modal));
    app.innerHTML = state.page === 'detail' ? renderDetail() : renderBoard();
    if (modalScrollTop != null) {
      // DOM 重建会移除当前焦点，浏览器可能在下一帧前自动把弹窗滚回上方；先同步恢复一次。
      const immediateModalBody = app.querySelector('.kv2-modal-body');
      if (immediateModalBody) immediateModalBody.scrollTop = modalScrollTop;
      requestAnimationFrame(function () {
        const nextModalBody = app.querySelector('.kv2-modal-body');
        if (nextModalBody) nextModalBody.scrollTop = modalScrollTop;
        positionSemanticPickerDropdown();
      });
    } else if (state.semanticPicker) {
      requestAnimationFrame(positionSemanticPickerDropdown);
    }
  }

  function positionSemanticPickerDropdown() {
    const picker = app.querySelector('.kv2-smart-select.open');
    const dropdown = picker && picker.querySelector('.kv2-smart-select-dropdown');
    const control = picker && picker.querySelector('.kv2-smart-select-control');
    if (!dropdown || !control || typeof control.getBoundingClientRect !== 'function') return;
    const rect = control.getBoundingClientRect();
    const viewportHeight = typeof window === 'undefined' ? 900 : window.innerHeight;
    const estimatedHeight = Math.min(288, dropdown.scrollHeight || 288);
    const openUpward = rect.bottom + estimatedHeight + 12 > viewportHeight;
    dropdown.style.left = Math.max(12, rect.left) + 'px';
    dropdown.style.top = Math.max(12, openUpward ? rect.top - estimatedHeight - 6 : rect.bottom + 6) + 'px';
    dropdown.style.width = rect.width + 'px';
  }

  function renderBoard() {
    const navigation = state.boardTab === 'explore'
      ? '<div class="kv2-board-tabs"><button class="kv2-board-return" data-action="back-from-chat" title="返回知识库">' + icons.back + '<span>返回</span></button><button class="kv2-board-tab active" type="button" aria-current="page">对话</button></div>'
      : '<div class="kv2-board-tabs"><button class="kv2-board-tab active" data-action="board-tab" data-tab="knowledge">知识库2</button><button class="kv2-board-tab" data-action="board-tab" data-tab="explore">对话</button></div>';
    return '<div class="kv2-shell">'
      + navigation
      + '<div class="kv2-board-body" style="' + (state.boardTab === 'explore' ? 'padding:0;overflow:hidden' : '') + '">'
      + (state.boardTab === 'knowledge' ? renderModelBoard() : renderExplore())
      + '</div>' + renderModal() + '</div>';
  }

  function renderModelBoard() {
    const q = state.query.trim().toLowerCase();
    const filtered = models.filter(function (m) { return !q || (m.name + ' ' + m.desc).toLowerCase().includes(q); });
    const cards = filtered.map(renderModelCard).join('');
    const searchEmpty = models.length > 0 && filtered.length === 0 ? '<div class="kv2-empty">没有符合搜索条件的知识库</div>' : '';
    const modelEmpty = models.length === 0 ? '<div class="kv2-board-empty"><span class="kv2-board-empty-icon">' + icons.folder + '</span><h3>还没有知识库</h3><p>创建知识库后，可以添加数据、配置语义资产并开始对话。</p></div>' : '';
    return '<div class="kv2-board-toolbar"><label class="kv2-search"><input data-input="board-search" value="' + attr(state.query) + '" placeholder="搜索知识库名称或描述">' + icons.search + '</label><button class="kv2-primary kv2-create-primary" data-action="open-create"><span class="kv2-primary-icon">' + icons.plus + '</span>创建知识库</button></div>'
      + '<div class="kv2-grid">' + cards + searchEmpty + modelEmpty + '</div>';
  }

  function renderModelCard(model) {
    const stats = (model.files > 0 ? '<span class="kv2-stat">' + model.files + ' 文件</span>' : '') + (model.tables > 0 ? '<span class="kv2-stat">' + model.tables + ' 表</span>' : '');
    return '<div class="kv2-card" tabindex="0" data-action="open-model" data-id="' + model.id + '">'
      + '<div class="kv2-card-body"><div class="kv2-card-head"><div class="kv2-folder-wrap"><div class="kv2-folder">' + icons.folder + '</div></div>'
      + '<div class="kv2-card-actions"><button class="kv2-icon-btn" title="删除" aria-label="删除知识库" data-action="delete-model" data-id="' + model.id + '">' + icons.trash + '</button></div></div>'
      + '<div class="kv2-card-title" title="' + attr(model.name) + '">' + h(model.name) + '</div><div class="kv2-card-remark">' + h(model.desc) + '</div></div>'
      + '<div class="kv2-card-foot">' + stats + '<button class="kv2-dialog-btn" data-action="start-dialog" data-id="' + model.id + '">' + icons.send + '<span>对话</span></button></div></div>';
  }

  function renderDetail() {
    const m = currentModel();
    return '<div class="kv2-detail"><div class="kv2-detail-head"><div class="kv2-detail-titlebar"><button class="kv2-back" data-action="back-board">' + icons.back + '</button><div><h2 class="kv2-detail-title">' + h(m.name) + '</h2><div class="kv2-detail-desc">' + h(m.desc) + '</div></div></div>'
      + '<div class="kv2-detail-actions"><button class="kv2-detail-secondary" data-action="edit-model" data-id="' + m.id + '">' + icons.edit + '<span>编辑信息</span></button><button class="kv2-detail-secondary" data-action="start-dialog" data-id="' + m.id + '">' + icons.send + '<span>对话</span></button><button class="kv2-primary" data-action="add-source">添加数据</button></div></div>'
      + '<div class="kv2-detail-content"><div class="kv2-tabs"><button class="kv2-tab ' + (state.detailTab === 'source' ? 'active' : '') + '" data-action="detail-tab" data-tab="source">数据源</button><button class="kv2-tab ' + (state.detailTab === 'semantic' ? 'active' : '') + '" data-action="detail-tab" data-tab="semantic">语义配置</button><button class="kv2-tab ' + (state.detailTab === 'agents' ? 'active' : '') + '" data-action="detail-tab" data-tab="agents">智能体关联</button></div>'
      + (state.detailTab === 'source' ? renderSources() : state.detailTab === 'semantic' ? renderSemantic() : renderAgents()) + '</div>' + renderModal() + '</div>';
  }

  function renderSources() {
    const sources = getSources();
    const fileCount = sources.filter(function (s) { return s.type === '文件'; }).length;
    const tableCount = sources.filter(function (s) { return s.type === '数据表'; }).length;
    const rows = sources.map(function (s) {
      const tableSource = s.type === '数据表';
      const actions = (tableSource ? '<button class="kv2-link" data-action="source-sql" data-id="' + s.id + '">SQL</button>' : '<button class="kv2-link" data-action="source-expiry" data-id="' + s.id + '">有效期</button><button class="kv2-link kv2-link-icon" data-action="source-download" data-id="' + s.id + '">' + icons.download + '<span>下载</span></button>') + '<button class="kv2-link danger kv2-link-icon" data-action="delete-source" data-id="' + s.id + '">' + icons.trash + '<span>删除</span></button>';
      return '<tr><td><button class="kv2-link kv2-source-name" data-action="source-detail" data-id="' + s.id + '">' + (tableSource ? icons.table : icons.file) + '<span>' + h(s.name) + '</span></button></td><td>' + (tableSource ? '表' : '文件') + '</td><td>' + h(s.size) + '</td><td>' + h(s.path.replaceAll(' / ', '/')) + '</td><td><span class="kv2-tag ' + (s.status === '处理中' ? 'processing' : '') + '">' + h(s.status) + '</span></td><td>' + h(s.updated) + '</td><td><button class="kv2-switch ' + (s.enabled ? '' : 'off') + '" aria-label="启用状态" data-action="toggle-source" data-id="' + s.id + '"></button></td><td><div class="kv2-source-actions">' + actions + '</div></td></tr>';
    }).join('');
    return '<div class="kv2-panel"><div class="kv2-panel-intro"><span>以下按照结构化程度展示知识库已关联的文件和表。点击文档可查看详情。</span><span>共 ' + fileCount + ' 个非结构化文件，' + tableCount + ' 张结构化表</span></div><div class="kv2-table-wrap"><table class="kv2-table"><thead><tr><th style="min-width:360px">名称</th><th>类型</th><th>大小 / 行数</th><th style="min-width:260px">Catalog 路径</th><th>处理状态</th><th>更新时间</th><th>启用状态</th><th style="min-width:220px">操作</th></tr></thead><tbody>' + rows + '</tbody></table></div></div>';
  }

  function renderSemantic() {
    const assets = getAssets();
    const cat = categories.find(function (c) { return c.key === state.semCategory; }) || categories[0];
    const q = state.semQuery.trim().toLowerCase();
    const filtered = assets.filter(function (a) {
      if (a.category !== cat.key) return false;
      if (!q) return true;
      const f = a.fields || {};
      const searchableText = cat.key === 'standard-qa' || cat.key === 'unstructured-qa'
        ? [f.assetId, f.question, listValue(f.relatedFiles).join(' ')].join(' ')
        : cat.key === 'dynamic-query'
          ? String(f.assetId || '')
          : [a.name, a.desc, a.binding, f.assetId].join(' ');
      return searchableText.toLowerCase().includes(q);
    });
    const pageSize = 10;
    const pageCount = Math.max(1, Math.ceil(filtered.length / pageSize));
    const currentPage = Math.min(state.semanticPage, pageCount);
    const visibleAssets = filtered.slice((currentPage - 1) * pageSize, currentPage * pageSize);
    const categoryNavItem = function (c) {
      const count = assets.filter(function (a) { return a.category === c.key; }).length;
      return '<button class="kv2-sem-item ' + (c.key === cat.key ? 'active' : '') + '" data-action="sem-category" data-category="' + c.key + '"><span class="kv2-sem-item-label"><span class="kv2-sem-icon">' + icons[c.icon] + '</span><span>' + h(c.name) + '</span></span><span class="kv2-sem-count">' + count + '</span></button>';
    };
    const activeSemanticGroup = semanticGroups.find(function (group) { return group.categories.includes(cat.key); }) || semanticGroups[0];
    const activeCategories = activeSemanticGroup.categories.map(function (key) { return categories.find(function (category) { return category.key === key; }); }).filter(Boolean);
    let nav = activeCategories.map(categoryNavItem).join('');
    if (activeSemanticGroup.key === 'structured') {
      const commonCategoryKeys = ['metric', 'business-rule', 'relationship', 'standard-qa'];
      const advancedCategoryKeys = ['schema', 'constraint', 'dynamic-query'];
      const commonCategories = commonCategoryKeys.map(function (key) { return categories.find(function (category) { return category.key === key; }); }).filter(Boolean);
      const advancedCategories = advancedCategoryKeys.map(function (key) { return categories.find(function (category) { return category.key === key; }); }).filter(Boolean);
      const advancedActive = advancedCategoryKeys.includes(cat.key);
      const advancedOpen = state.semanticAdvancedOpen || advancedActive;
      nav = commonCategories.map(categoryNavItem).join('')
        + '<div class="kv2-sem-advanced"><button type="button" class="kv2-sem-advanced-toggle ' + (advancedActive ? 'has-active' : '') + '" data-action="toggle-sem-advanced" aria-expanded="' + advancedOpen + '"><span>高级配置</span><span class="kv2-sem-advanced-chevron ' + (advancedOpen ? 'open' : '') + '" aria-hidden="true">›</span></button>'
        + (advancedOpen ? '<div class="kv2-sem-advanced-items">' + advancedCategories.map(categoryNavItem).join('') + '</div>' : '') + '</div>';
    }
    const typeTabs = semanticGroups.map(function (group) {
      const active = group.key === activeSemanticGroup.key;
      return '<button type="button" role="tab" aria-selected="' + active + '" class="kv2-sem-type-tab ' + (active ? 'active' : '') + '" data-action="sem-scope" data-scope="' + group.key + '">' + h(group.name) + '</button>';
    }).join('');
    const action = function (a, schemaOnly) {
      return '<button class="kv2-sem-action-icon" data-action="edit-asset" data-id="' + a.id + '" title="修改" aria-label="修改">' + icons.edit + '</button>' + (schemaOnly ? '' : '<button class="kv2-sem-action-icon danger" data-action="delete-asset" data-id="' + a.id + '" title="删除" aria-label="删除">' + icons.trash + '</button>');
    };
    let headers = '';
    const rows = visibleAssets.map(function (a) {
      const f = a.fields || {};
      const key = f.assetId || a.name;
      if (cat.key === 'metric') {
        headers = '<th style="width:180px">标识</th><th style="width:240px">名称</th><th style="width:120px">指标类型</th><th style="width:240px">关联表</th><th style="width:280px">计算规则</th><th style="width:180px">更新时间</th><th style="width:150px">操作</th>';
        const calculation = f.metricType === 'derived' ? formulaPreviewText(f.formula || f.metricFormulaExpression || '', f.sourceTable) : (f.aggregation || 'SUM') + '(' + (f.field || '—') + ')' + (String(f.multiplier || '1') === '1' ? '' : ' * ' + f.multiplier);
        const metricKind = f.metricType === 'derived' ? '派生指标' : '基础指标';
        const metricKindClass = f.metricType === 'derived' ? 'derived' : 'base';
        return '<tr><td><code class="kv2-code kv2-code-tag">' + h(key) + '</code></td><td><strong>' + h(a.name) + '</strong></td><td><span class="kv2-metric-kind ' + metricKindClass + '">' + metricKind + '</span></td><td class="kv2-muted">' + h(f.sourceTable || a.binding || '—') + '</td><td>' + h(calculation) + '</td><td class="kv2-muted">' + h(a.updated) + '</td><td class="kv2-row-actions">' + action(a, false) + '</td></tr>';
      }
      if (cat.key === 'schema') {
        headers = '<th style="width:320px">表名</th><th>表描述</th><th style="width:180px">更新时间</th><th style="width:140px">操作</th>';
        return '<tr><td>' + h(f.table || a.binding || key) + '</td><td class="kv2-muted">' + h(a.desc || a.name) + '</td><td class="kv2-muted">' + h(a.updated) + '</td><td class="kv2-row-actions">' + action(a, true) + '</td></tr>';
      }
      if (cat.key === 'business-rule') {
        const ruleContent = f.content || a.desc || '—';
        headers = '<th style="width:220px">标识</th><th style="width:280px">关联表</th><th style="width:320px">规则内容</th><th style="width:180px">更新时间</th><th style="width:150px">操作</th>';
        return '<tr><td><code class="kv2-code kv2-code-tag">' + h(key) + '</code></td><td class="kv2-muted">' + h(listValue(f.relatedTables || a.binding).join(', ') || '—') + '</td><td class="kv2-single-line" title="' + attr(ruleContent) + '">' + h(ruleContent) + '</td><td class="kv2-muted">' + h(a.updated) + '</td><td class="kv2-row-actions">' + action(a, false) + '</td></tr>';
      }
      if (cat.key === 'unstructured-rule') {
        const ruleContent = f.content || a.desc || '—';
        headers = '<th style="width:220px">标识</th><th style="width:280px">关联文件</th><th style="width:320px">规则内容</th><th style="width:180px">更新时间</th><th style="width:150px">操作</th>';
        return '<tr><td><code class="kv2-code kv2-code-tag">' + h(key) + '</code></td><td class="kv2-muted">' + h(listValue(f.relatedFiles || a.binding).join(', ') || '—') + '</td><td class="kv2-single-line" title="' + attr(ruleContent) + '">' + h(ruleContent) + '</td><td class="kv2-muted">' + h(a.updated) + '</td><td class="kv2-row-actions">' + action(a, false) + '</td></tr>';
      }
      if (cat.key === 'relationship') {
        headers = '<th style="width:220px">标识</th><th style="width:260px">左表</th><th style="width:150px">JOIN 类型</th><th style="width:260px">右表</th><th style="width:180px">更新时间</th><th style="width:150px">操作</th>';
        return '<tr><td><code class="kv2-code kv2-code-tag">' + h(key) + '</code></td><td class="kv2-muted">' + h(f.leftTable || '—') + '</td><td><span class="kv2-join-type">' + h(f.joinType || 'LEFT JOIN') + '</span></td><td class="kv2-muted">' + h(f.rightTable || '—') + '</td><td class="kv2-muted">' + h(a.updated) + '</td><td class="kv2-row-actions">' + action(a, false) + '</td></tr>';
      }
      if (cat.key === 'standard-qa') {
        const forced = f.usageMode === 'force';
        headers = '<th style="width:220px">标识</th><th>标准问题</th><th style="width:130px">使用方式</th><th style="width:180px">更新时间</th><th style="width:150px">操作</th>';
        return '<tr><td><code class="kv2-code kv2-code-tag">' + h(key) + '</code></td><td>' + h(f.question || '—') + '</td><td><span class="kv2-qa-usage ' + (forced ? 'force' : 'reference') + '">' + (forced ? '强制使用' : '参考使用') + '</span></td><td class="kv2-muted">' + h(a.updated) + '</td><td class="kv2-row-actions">' + action(a, false) + '</td></tr>';
      }
      if (cat.key === 'unstructured-qa') {
        const forced = f.usageMode === 'force';
        headers = '<th style="width:220px">标识</th><th>标准问题</th><th style="width:260px">关联文件</th><th style="width:130px">使用方式</th><th style="width:180px">更新时间</th><th style="width:150px">操作</th>';
        return '<tr><td><code class="kv2-code kv2-code-tag">' + h(key) + '</code></td><td>' + h(f.question || '—') + '</td><td class="kv2-muted">' + h(listValue(f.relatedFiles || a.binding).join(', ') || '—') + '</td><td><span class="kv2-qa-usage ' + (forced ? 'force' : 'reference') + '">' + (forced ? '强制使用' : '参考使用') + '</span></td><td class="kv2-muted">' + h(a.updated) + '</td><td class="kv2-row-actions">' + action(a, false) + '</td></tr>';
      }
      if (cat.key === 'dynamic-query') {
        const delegated = f.resultProcessingMode === 'delegated_analysis';
        const sqlText = f.sql || '—';
        headers = '<th style="width:220px">标识</th><th>SQL</th><th style="width:170px">查询结果处理</th><th style="width:180px">更新时间</th><th style="width:150px">操作</th>';
        return '<tr><td><code class="kv2-code kv2-code-tag">' + h(key) + '</code></td><td><code class="kv2-code kv2-single-line" title="' + attr(sqlText) + '">' + h(sqlText) + '</code></td><td><span class="kv2-processing-mode ' + (delegated ? 'delegated' : 'direct') + '">' + (delegated ? '子 Agent 分析' : '直接注入上下文') + '</span></td><td class="kv2-muted">' + h(a.updated) + '</td><td class="kv2-row-actions">' + action(a, false) + '</td></tr>';
      }
      headers = '<th style="width:220px">标识</th><th style="width:280px">关联表</th><th style="width:180px">更新时间</th><th style="width:150px">操作</th>';
      return '<tr><td><code class="kv2-code kv2-code-tag">' + h(key) + '</code></td><td class="kv2-muted">' + h(listValue(f.relatedTables || f.sourceTable || a.binding).join(', ') || '—') + '</td><td class="kv2-muted">' + h(a.updated) + '</td><td class="kv2-row-actions">' + action(a, false) + '</td></tr>';
    }).join('');
    const pagination = filtered.length > pageSize ? '<div class="kv2-pagination"><span>共 ' + filtered.length + ' 条</span><button type="button" data-action="semantic-page" data-page="' + Math.max(1, currentPage - 1) + '" ' + (currentPage === 1 ? 'disabled' : '') + '>‹</button>' + Array.from({ length: pageCount }, function (_item, index) { const page = index + 1; return '<button type="button" class="' + (page === currentPage ? 'active' : '') + '" data-action="semantic-page" data-page="' + page + '">' + page + '</button>'; }).join('') + '<button type="button" data-action="semantic-page" data-page="' + Math.min(pageCount, currentPage + 1) + '" ' + (currentPage === pageCount ? 'disabled' : '') + '>›</button></div>' : '';
    const placeholders = { metric: '搜索名称、标识或关联表', schema: '搜索表名或表描述', 'business-rule': '搜索标识、规则或关联表', 'unstructured-rule': '搜索标识、规则或关联文件', 'standard-qa': '搜索标识或标准问题', 'unstructured-qa': '搜索标识、标准问题或关联文件', 'dynamic-query': '搜索标识' };
    const globalSemanticActions = '<div class="kv2-sem-global-actions"><input id="kv2SemanticImportFile" class="kv2-visually-hidden" type="file" accept="application/json,.json" data-input="semantic-import-file"><button type="button" class="kv2-btn" data-action="open-semantic-import">导入语义</button><button type="button" class="kv2-btn" data-action="export-semantics">导出语义</button></div>';
    const categoryAction = cat.key === 'schema' ? '' : '<button class="kv2-primary" data-action="new-asset"><span class="kv2-primary-icon">' + icons.plus + '</span>新建' + h(cat.name) + '</button>';
    return '<div class="kv2-panel"><div class="kv2-sem-overview"><p>为知识库补充数据的业务含义和使用规则。NL2SQL语义定义用于查询生成的指标、字段、约束与关系；RAG语义关联已解析文件，提供业务规则和经过确认的标准问答。</p>' + globalSemanticActions + '</div><div class="kv2-semantic-shell"><div class="kv2-sem-type-tabs" role="tablist" aria-label="语义类型">' + typeTabs + '</div><div class="kv2-semantic"><aside class="kv2-sem-nav" aria-label="' + h(activeSemanticGroup.name) + '资产分类">' + nav + '</aside><main class="kv2-sem-main"><div class="kv2-sem-head"><div><h3>' + h(cat.name) + '</h3><p>' + h(cat.desc) + '</p></div>' + categoryAction + '</div><div class="kv2-sem-toolbar"><label class="kv2-sem-search"><span>' + icons.search + '</span><input data-input="sem-search" value="' + attr(state.semQuery) + '" placeholder="' + h(placeholders[cat.key] || '搜索标识或关联表') + '"></label></div>'
      + (rows ? '<div class="kv2-table-wrap"><table class="kv2-table kv2-sem-table"><thead><tr>' + headers + '</tr></thead><tbody>' + rows + '</tbody></table></div>' + pagination : '<div class="kv2-sem-empty">' + (state.semQuery ? '没有匹配的语义资产' : '当前分类暂无语义资产') + '</div>') + '</main></div></div></div>';
  }

  function renderAgents() {
    const rows = agents.map(function (agent) {
      return '<tr><td><span class="kv2-source-name">' + icons.robot + '<span>' + h(agent.name) + '</span></span></td><td>' + h(agent.desc) + '</td><td><span class="kv2-tag ' + (agent.status === '草稿' ? 'processing' : '') + '">' + h(agent.status) + '</span></td><td>' + agent.knowledgeCount + '</td><td>' + h(agent.updated) + '</td><td><button class="kv2-link">前往智能体</button></td></tr>';
    }).join('');
    return '<div class="kv2-panel"><div class="kv2-panel-intro"><span>以下展示显式绑定该知识库 ID 的智能体。</span><span>共 ' + agents.length + ' 个智能体</span></div><div class="kv2-table-wrap"><table class="kv2-table"><thead><tr><th>智能体</th><th>描述</th><th>状态</th><th>绑定知识库数</th><th>更新时间</th><th>操作</th></tr></thead><tbody>' + rows + '</tbody></table></div></div>';
  }

  function renderExplore() {
    const chats = ['EO_1000 五月营业收入', '主营业务成本口径', '年度期初数据查询'];
    const messages = state.messages.map(function (msg) {
      const pipeline = msg.pipeline ? '<div class="kv2-pipeline"><button type="button" data-action="toggle-pipeline"><span>知识库2执行流程</span><b>' + (state.pipelineOpen ? '收起' : '查看执行过程') + '</b></button>' + (state.pipelineOpen ? '<ol><li><span>1</span><div><strong>范围发现</strong><small>定位 BPC 与 bpc_consolidated_report</small></div><em>已完成</em></li><li><span>2</span><div><strong>并行语义候选</strong><small>匹配营业收入、主营业务成本及期间规则</small></div><em>已完成</em></li><li><span>3</span><div><strong>三层指标 SQL</strong><small>编译科目口径并注入 ACT_LG、CNY 强制约束</small></div><em>已完成</em></li><li><span>4</span><div><strong>标准问答校验与执行</strong><small>按 EO_1000、202505 校验并读取 5 条样例数据</small></div><em>已完成</em></li></ol>' : '') + '</div>' : '';
      return '<div class="kv2-message ' + (msg.role === 'user' ? 'user' : '') + '"><div class="kv2-message-meta"><strong>' + (msg.role === 'user' ? 'USER' : 'ASSISTANT') + '</strong><span>刚刚</span></div>' + pipeline + '<div class="kv2-bubble">' + h(msg.text).replace(/\n/g, '<br>') + '</div></div>';
    }).join('');
    return '<div class="kv2-explore"><aside class="kv2-explore-side"><button class="kv2-new-chat" data-action="new-chat">＋ 新建对话</button><div class="kv2-session-search"><span>' + icons.search + '</span><input placeholder="搜索会话"></div><div class="kv2-session-label">最近</div>' + chats.map(function (c, i) { return '<div class="kv2-chat-item ' + (i === 0 ? 'active' : '') + '">' + h(c) + '</div>'; }).join('') + '</aside><main class="kv2-explore-main"><div class="kv2-explore-head"><div class="kv2-explore-title">' + h(chats[0]) + '</div></div><div class="kv2-messages">' + messages + '</div><div class="kv2-composer"><div class="kv2-composer-box"><div class="kv2-selected-knowledge"><span class="kv2-folder-mini">' + icons.folder + '</span><strong>' + h(currentModel().name) + '</strong></div><textarea id="kv2ChatInput" placeholder="请输入问题，按 Enter 发送（Shift+Enter 换行）"></textarea><div class="kv2-composer-foot"><div class="kv2-composer-tools"><button type="button" class="kv2-composer-plus" title="上传图片">＋</button><select class="kv2-model-select"><option>glm-5.2</option><option>gpt-5.2</option></select><select class="kv2-kb-select" data-input="chat-model">' + models.map(function (m) { return '<option value="' + m.id + '" ' + (m.id === state.modelId ? 'selected' : '') + '>' + h(m.name) + '</option>'; }).join('') + '</select></div><button class="kv2-send" data-action="send-chat">' + icons.send + '</button></div></div></div></main></div>';
  }

  function renderModal() {
    if (!state.modal) return '';
    if (state.modal === 'create') return renderCreateModal();
    if (state.modal === 'edit-model') return renderEditModelModal();
    if (state.modal === 'add-source') return renderSourcePicker();
    if (state.modal === 'asset') return renderAssetModal();
    if (state.modal === 'source-detail') return renderSourceDetail();
    if (state.modal === 'source-expiry') return renderSourceExpiry();
    if (state.modal === 'confirm') return renderConfirmModal();
    if (state.modal === 'semantic-import') return renderSemanticImportModal();
    return '';
  }

  function modalFrame(title, body, footer, size) {
    return '<div class="kv2-overlay" data-action="overlay-close"><div class="kv2-modal ' + (size || '') + '" data-modal-stop role="dialog" aria-modal="true" aria-label="' + attr(title) + '"><div class="kv2-modal-head"><h3>' + h(title) + '</h3><button class="kv2-close" data-action="close-modal" aria-label="关闭">×</button></div><div class="kv2-modal-body">' + body + '</div>' + (footer ? '<div class="kv2-modal-foot">' + footer + '</div>' : '') + '</div></div>';
  }

  function openConfirm(config) {
    state.pendingConfirm = config;
    state.modal = 'confirm';
    render();
  }

  function renderConfirmModal() {
    const confirmState = state.pendingConfirm;
    if (!confirmState) return '';
    const body = '<div class="kv2-confirm-copy"><p>' + h(confirmState.message) + '</p><small>' + h(confirmState.description) + '</small></div>';
    const footer = '<button class="kv2-btn" data-action="cancel-confirm">取消</button><button class="kv2-btn primary" data-action="confirm-delete">确认</button>';
    return modalFrame(confirmState.title, body, footer, 'confirm');
  }

  function renderSemanticImportModal() {
    const preview = state.semanticImport;
    if (!preview) return '';
    const ready = preview.status === 'ready';
    const error = preview.status === 'error';
    const categorySummary = ready
      ? '<span>NL2SQL语义 ' + h(preview.domainCounts && preview.domainCounts.nl2sql || 0) + '</span><span>RAG语义 ' + h(preview.domainCounts && preview.domainCounts.rag || 0) + '</span>'
      : '';
    const conflictText = ready && preview.conflicts
      ? '追加时将新增 ' + preview.additions + ' 条，跳过 ' + preview.conflicts + ' 条同类型、同标识的重复语义。'
      : ready ? '追加时将新增全部 ' + preview.additions + ' 条语义，没有重复项。' : '';
    const fileState = ready
      ? '<div class="kv2-import-file valid"><div><strong>' + h(preview.fileName) + '</strong><span>格式校验通过 · schema_version ' + h(preview.schemaVersion) + '</span></div><b>' + preview.assets.length + ' 条</b></div><div class="kv2-import-category-summary">' + categorySummary + '</div><p>' + h(conflictText) + '</p>'
      : error
        ? '<div class="kv2-import-error" role="alert"><strong>无法读取该 JSON</strong><span>' + h(preview.error) + '</span><small>' + h(preview.fileName || '') + '</small></div>'
        : '<button type="button" class="kv2-import-upload" data-action="choose-semantic-import"><strong>选择 JSON 文件</strong><span>选择后先校验文件，不会立即导入</span></button>';
    const mode = state.semanticImportMode === 'overwrite' ? 'overwrite' : 'append';
    const modeChoices = '<section class="kv2-import-section"><h4>导入方式</h4><div class="kv2-import-mode-list" role="radiogroup" aria-label="语义导入方式"><label><input type="radio" name="semanticImportMode" value="append" ' + (mode === 'append' ? 'checked' : '') + '><span><strong>追加 <em>推荐</em></strong><small>保留当前配置；相同类型、相同标识的配置跳过。</small></span></label><label><input type="radio" name="semanticImportMode" value="overwrite" ' + (mode === 'overwrite' ? 'checked' : '') + '><span><strong>覆盖</strong><small>清空当前语义，并使用文件中的配置整体替换。</small></span></label></div></section>';
    const body = '<div class="kv2-import-preview"><p class="kv2-import-lead">导入语义配置到当前知识库。</p><section class="kv2-import-section"><div class="kv2-import-section-head"><h4>导入文件</h4><button type="button" class="kv2-link" data-action="download-semantic-example">下载示例</button></div>' + fileState + (ready || error ? '<button type="button" class="kv2-link kv2-import-reselect" data-action="choose-semantic-import">重新选择文件</button>' : '') + '</section>' + modeChoices + '</div>';
    const footer = '<button class="kv2-btn" data-action="close-modal">取消</button><button class="kv2-btn primary" data-action="confirm-semantic-import" ' + (ready ? '' : 'disabled') + '>开始导入</button>';
    return modalFrame('导入语义', body, footer, 'semantic-import-dialog');
  }

  function executeConfirmedDelete() {
    const confirmState = state.pendingConfirm;
    state.pendingConfirm = null;
    state.modal = null;
    if (!confirmState) { render(); return; }
    if (confirmState.kind === 'model') {
      const model = models.find(function (item) { return item.id === confirmState.id; });
      if (model) {
        const index = models.indexOf(model);
        state.deletedDemo.models.push({ item: model, index: index });
        models.splice(index, 1);
        if (state.modelId === confirmState.id) state.modelId = null;
      }
    }
    if (confirmState.kind === 'asset') {
      const assets = getAssets();
      const asset = assets.find(function (item) { return item.id === confirmState.id; });
      if (asset) {
        const index = assets.indexOf(asset);
        state.deletedDemo.assets.push({ modelId: state.modelId, item: asset, index: index });
        assets.splice(index, 1);
      }
    }
    if (confirmState.kind === 'source') {
      const sources = getSources();
      const source = sources.find(function (item) { return item.id === confirmState.id; });
      if (source) {
        const index = sources.indexOf(source);
        state.deletedDemo.sources.push({ modelId: state.modelId, item: source, index: index });
        sources.splice(index, 1);
        currentModel().files = sources.filter(function (item) { return item.type === '文件'; }).length;
        currentModel().tables = sources.filter(function (item) { return item.type === '数据表'; }).length;
      }
    }
    render();
  }

  function renderCreateModal() {
    const d = state.createDraft;
    const body = '<form id="kv2CreateBase" class="kv2-form"><div class="kv2-field"><label class="required">知识库名称</label><input name="name" maxlength="255" required value="' + attr(d.name) + '" placeholder="请输入知识库名称"></div><div class="kv2-field"><label class="required">描述</label><textarea name="description" rows="5" maxlength="10000" required placeholder="请输入知识库用途及可回答的问题，例如：用于经营指标、期间和组织维度相关查询。">' + h(d.description) + '</textarea></div></form>';
    return modalFrame('新建知识库', body, '<button class="kv2-btn" data-action="close-modal">取消</button><button class="kv2-btn primary" data-action="create-model">创建</button>', 'metadata');
  }

  function renderSourcePicker() {
    const selected = state.sourceSelected;
    const tableScope = state.catalogScope === 'tables';
    const query = state.catalogQuery.trim().toLowerCase();
    const scopeRows = catalogSources.filter(function (source) { return (tableScope ? source.type === '数据表' : source.type === '文件') && (!query || source.name.toLowerCase().includes(query)); });
    const pageSize = 20;
    const pageCount = Math.max(1, Math.ceil(scopeRows.length / pageSize));
    const currentPage = Math.min(state.catalogPage, pageCount);
    const visibleRows = scopeRows.slice((currentPage - 1) * pageSize, currentPage * pageSize);
    const selectableRows = scopeRows.filter(function (source) { return !getSources().some(function (current) { return current.name === source.name; }); });
    const allSelected = selectableRows.length > 0 && selectableRows.every(function (source) { return selected.includes(source.id); });
    const selectedSources = selected.map(function (id) { return catalogSources.find(function (source) { return source.id === id; }); }).filter(Boolean);
    const selectedFiles = selectedSources.filter(function (source) { return source.type === '文件'; }).length;
    const selectedTables = selectedSources.filter(function (source) { return source.type === '数据表'; }).length;
    const catalogOpen = state.catalogTreeOpen.catalog !== false;
    const databaseOpen = state.catalogTreeOpen.database !== false;
    const treeNode = function (node, label, icon, depth, open) {
      return '<button type="button" class="kv2-catalog-tree-node depth-' + depth + '" data-action="toggle-catalog-tree" data-node="' + node + '" aria-expanded="' + open + '"><span class="kv2-catalog-tree-chevron ' + (open ? 'open' : '') + '">›</span><span class="kv2-catalog-nav-icon">' + icon + '</span><span class="kv2-catalog-tree-label">' + h(label) + '</span></button>';
    };
    const treeLeaf = function (scope, label, icon, count) {
      const active = state.catalogScope === scope;
      return '<button type="button" class="kv2-catalog-tree-node depth-3 leaf ' + (active ? 'active' : '') + '" data-action="catalog-scope" data-scope="' + scope + '" aria-current="' + (active ? 'page' : 'false') + '"><span class="kv2-catalog-tree-spacer"></span><span class="kv2-catalog-nav-icon">' + icon + '</span><span class="kv2-catalog-tree-label">' + h(label) + '</span><span class="kv2-catalog-tree-count">' + count + '</span></button>';
    };
    const navigator = '<div class="kv2-catalog-navigator"><div class="kv2-catalog-tree-head"><strong>Catalog 目录</strong><span>按目录层级展开并选择数据</span></div><div class="kv2-catalog-tree" role="tree">'
      + treeNode('catalog', '默认', '□', 1, catalogOpen)
      + (catalogOpen ? '<div role="group">' + treeNode('database', 'jst_flat_table', '▦', 2, databaseOpen)
        + (databaseOpen ? '<div role="group">' + treeLeaf('tables', '数据表', '▦', catalogSources.filter(function (source) { return source.type === '数据表'; }).length) + treeLeaf('files', 'BPC', '▱', catalogSources.filter(function (source) { return source.type === '文件'; }).length) + '</div>' : '') + '</div>' : '')
      + '</div></div>';
    const rows = visibleRows.map(function (source) {
      const existing = getSources().some(function (current) { return current.name === source.name; });
      const sourceName = source.type === '数据表'
        ? '<span class="kv2-catalog-source-copy"><strong>' + h(source.name) + '</strong><small>' + h(tableCommentFor(source.name, source.comment)) + '</small></span>'
        : '<span class="kv2-catalog-source-copy"><strong>' + h(source.name) + '</strong></span>';
      return '<tr><td class="kv2-catalog-check"><input type="checkbox" data-input="source-check" value="' + source.id + '" ' + (existing || selected.includes(source.id) ? 'checked' : '') + ' ' + (existing ? 'disabled' : '') + '></td><td><span class="kv2-catalog-source-name"><span>' + (source.type === '数据表' ? '▦' : '▤') + '</span>' + sourceName + (existing ? '<span class="kv2-catalog-state">已添加</span>' : '') + '</span></td><td>' + h(source.size) + '</td><td>' + h(source.updated) + '</td></tr>';
    }).join('');
    const documentNotice = tableScope ? '' : '<div class="kv2-document-use-notice" role="note"><span class="kv2-document-use-icon">i</span><div><strong>非结构化文档需完成解析与向量嵌入后才会被知识库使用</strong><span>添加后系统将自动处理文档；处理完成前，文档不会参与知识检索和回答。</span></div></div>';
    const pagination = scopeRows.length > pageSize
      ? '<div class="kv2-catalog-pagination"><span>第 ' + currentPage + ' / ' + pageCount + ' 页</span><div><button type="button" data-action="catalog-page" data-page="' + Math.max(1, currentPage - 1) + '" ' + (currentPage === 1 ? 'disabled' : '') + '>上一页</button><button type="button" data-action="catalog-page" data-page="' + Math.min(pageCount, currentPage + 1) + '" ' + (currentPage === pageCount ? 'disabled' : '') + '>下一页</button></div></div>'
      : '';
    const leafPanel = '<div class="kv2-catalog-leaf"><div class="kv2-catalog-leaf-head"><div><span class="kv2-catalog-breadcrumb">默认 / jst_flat_table / ' + (tableScope ? '数据表' : 'BPC') + '</span><strong>' + (tableScope ? '数据表' : 'BPC 文件') + '</strong></div><label class="kv2-catalog-search"><span>' + icons.search + '</span><input data-input="catalog-search" value="' + attr(state.catalogQuery) + '" placeholder="' + (tableScope ? '搜索当前目录下的表' : '搜索当前目录下的文件') + '"></label></div>' + documentNotice + '<div class="kv2-catalog-toolbar"><label><input type="checkbox" data-input="source-select-all" ' + (allSelected ? 'checked' : '') + ' ' + (selectableRows.length ? '' : 'disabled') + '> 全选当前搜索结果</label><span>共 ' + scopeRows.length + ' 项</span></div><div class="kv2-catalog-table-wrap"><table class="kv2-catalog-table"><thead><tr><th></th><th>' + (tableScope ? '表名' : '文件名') + '</th><th>' + (tableScope ? '行数' : '大小') + '</th><th>更新时间</th></tr></thead><tbody>' + (rows || '<tr><td colspan="4" class="kv2-catalog-empty">没有找到匹配的数据</td></tr>') + '</tbody></table></div>' + pagination + '</div>';
    const pickerIntro = '<div class="kv2-source-picker-intro"><div><strong>从 Catalog 选择数据</strong><span>选择数据表或目录卷中的文档，添加到当前知识库。</span></div><div class="kv2-source-picker-count"><strong>' + selected.length + '</strong><span>已选择</span></div></div>';
    const catalogPicker = pickerIntro + '<div class="kv2-catalog-selector-layout">' + navigator + leafPanel + '</div>';
    const footer = '<div class="kv2-source-selection"><strong>共选择 ' + selected.length + ' 项</strong><span>已选 ' + selectedFiles + ' 个文件，' + selectedTables + ' 张表</span></div><div class="kv2-modal-foot-actions"><button class="kv2-btn" data-action="close-modal">取消</button><button class="kv2-btn primary" data-action="save-sources" ' + (selected.length ? '' : 'disabled') + '>添加</button></div>';
    return modalFrame('选择数据', catalogPicker, footer, 'source-picker');
  }

  function renderEditModelModal() {
    const m = currentModel();
    const body = '<form id="kv2EditModel" class="kv2-form"><div class="kv2-field"><label class="required">知识库名称</label><input name="name" maxlength="255" required value="' + attr(m.name) + '" placeholder="请输入知识库名称"></div><div class="kv2-field"><label class="required">备注说明内容</label><div class="kv2-control-help" style="margin:-2px 0 7px">请填写知识库用途及可回答的问题列表，以便Agent精准选择。</div><textarea name="description" rows="4" maxlength="10000" required placeholder="例如：包含公司所有产品的技术文档，可回答关于产品说明、参数配置及故障排除的问题。">' + h(m.desc) + '</textarea></div></form>';
    return modalFrame('编辑知识库元信息', body, '<button class="kv2-btn" data-action="close-modal">取消</button><button class="kv2-btn primary" data-action="save-model-meta">保存</button>', 'metadata');
  }

  function field(name, label, value, type, cls, options) {
    const c = cls ? ' ' + cls : '';
    if (type === 'textarea') return '<div class="kv2-field' + c + '"><label>' + h(label) + '</label><textarea name="' + name + '" ' + (name === 'sql' || name === 'content' ? 'class="kv2-code"' : '') + '>' + h(value || '') + '</textarea></div>';
    if (type === 'select') return '<div class="kv2-field' + c + '"><label>' + h(label) + '</label><select name="' + name + '">' + options.map(function (o) { const v = typeof o === 'object' ? o.value : o; const text = typeof o === 'object' ? o.label : o; return '<option value="' + attr(v) + '" ' + (String(value) === String(v) ? 'selected' : '') + '>' + h(text) + '</option>'; }).join('') + '</select></div>';
    return '<div class="kv2-field' + c + '"><label>' + h(label) + '</label><input name="' + name + '" value="' + attr(value || '') + '" ' + (name === 'assetId' ? 'class="kv2-code"' : '') + '></div>';
  }

  function listValue(value) {
    if (Array.isArray(value)) return value.filter(function (item) {
      return item !== null && item !== undefined && String(item).trim() !== '';
    });
    return String(value || '').split(/[\n,，]/).map(function (item) { return item.trim(); }).filter(Boolean);
  }

  function defaultFilterGroup(category, f) {
    if (f.filterGroup && f.filterGroup.conditions) return JSON.parse(JSON.stringify(f.filterGroup));
    if (category === 'constraint') {
      if (!f.assetId) return { type: 'group', logic: 'and', conditions: [{ type: 'condition', field: '', comparison: 'equals', value: '' }] };
      return { type: 'group', logic: 'and', conditions: [{ type: 'condition', field: f.field || 'b28_s_kgd4kbn', comparison: f.comparison || 'equals', value: f.value || 'CNY' }] };
    }
    if (!f.assetId) return { type: 'group', logic: 'and', conditions: [{ type: 'condition', field: '', comparison: 'equals', value: '' }] };
    if (f.assetId === 'bpc_revenue') {
      return { type: 'group', logic: 'and', conditions: [{ type: 'condition', field: 'account_path', comparison: 'contains', value: '/APL6000/' }] };
    }
    if (f.assetId === 'main_business_cost') {
      return { type: 'group', logic: 'or', conditions: [
        { type: 'group', logic: 'and', conditions: [
          { type: 'condition', field: 'account_path', comparison: 'contains', value: '/APL6600/' },
          { type: 'condition', connector: 'and', field: 'b28_s_kgdbveh', comparison: 'equals', value: 'TFUA1000' }
        ] },
        { type: 'condition', connector: 'or', field: 'account_path', comparison: 'contains', value: '/APL6401/' },
        { type: 'condition', connector: 'or', field: 'account_path', comparison: 'contains', value: '/APL5001/' }
      ] };
    }
    return { type: 'group', logic: 'and', conditions: [] };
  }

  function initAssetDraft(asset) {
    const f = Object.assign({}, asset ? asset.fields : {});
    if (state.semCategory === 'metric') {
      Object.assign(f, {
        assetId: f.assetId || '', name: f.name || '', description: f.description || '',
        metricType: f.metricType || 'aggregate', sourceTable: f.sourceTable || (f.metricType === 'derived' ? '' : 'jst_flat_table.bpc_consolidated_report'),
        aggregationFunction: String(f.aggregationFunction || f.aggregation || 'sum').toLowerCase(),
        aggregationField: f.aggregationField || f.field || 'b28_s_sdata', multiplier: f.multiplier == null ? 1 : f.multiplier,
        metricFormulaExpression: f.metricFormulaExpression || f.formula || '', synonyms: listValue(f.synonyms), output: f.output || 'number', decimalPlaces: f.decimalPlaces == null ? 2 : f.decimalPlaces,
        filterGroup: defaultFilterGroup('metric', f)
      });
    } else if (state.semCategory === 'schema') {
      f.schemaColumns = Array.isArray(f.schemaColumns) ? f.schemaColumns : bpcColumns.map(function (column) { return { name: column.name, description: column.description, supplementalDescription: '', enabled: true }; });
    } else if (state.semCategory === 'constraint') {
      f.sourceTable = f.sourceTable || listValue(f.relatedTables)[0] || '';
      delete f.relatedTables;
      f.filterGroup = defaultFilterGroup('constraint', f);
    } else if (state.semCategory === 'relationship') {
      f.joinConditions = Array.isArray(f.joinConditions) ? f.joinConditions : [{ leftField: f.leftField || '', comparison: 'equals', rightField: f.rightField || '' }];
    } else if (state.semCategory === 'standard-qa' || state.semCategory === 'unstructured-qa') {
      f.usageMode = f.usageMode === 'force' ? 'force' : 'reference';
      f.similarQuestions = listValue(f.similarQuestions);
      if (state.semCategory === 'unstructured-qa') f.relatedFiles = listValue(f.relatedFiles);
    } else if (state.semCategory === 'dynamic-query') {
      f.resultProcessingMode = f.resultProcessingMode === 'delegated_analysis' ? 'delegated_analysis' : 'direct_context';
    } else if (state.semCategory === 'business-rule') {
      f.relatedTables = listValue(f.relatedTables || f.sourceTable);
      if (!f.relatedTables.length) f.relatedTables = tableMetadataOptions().map(function (option) { return option.value; });
      delete f.relatedFiles;
    } else if (state.semCategory === 'unstructured-rule') {
      f.relatedFiles = listValue(f.relatedFiles);
    }
    state.assetDraft = f;
    state.assetError = '';
    state.assetValidationVisible = false;
    state.semanticPicker = null;
    state.semanticPickerQuery = '';
    state.schemaColumnSearch = '';
  }

  function syncAssetDraftFromForm() {
    const form = document.getElementById('kv2AssetForm');
    if (!form || !state.assetDraft) return;
    const data = new FormData(form);
    const values = Object.fromEntries(data.entries());
    Object.keys(values).forEach(function (key) { state.assetDraft[key] = values[key]; });
    if (state.semCategory === 'business-rule') state.assetDraft.relatedTables = data.getAll('relatedTables');
    if (state.semCategory === 'unstructured-rule' || state.semCategory === 'unstructured-qa') state.assetDraft.relatedFiles = data.getAll('relatedFiles');
    if (state.semCategory === 'relationship' && state.assetDraft.joinConditions) {
      state.assetDraft.joinConditions.forEach(function (condition, index) {
        condition.leftField = values['leftField_' + index] || '';
        condition.comparison = values['comparison_' + index] || 'equals';
        condition.rightField = values['rightField_' + index] || '';
      });
    }
  }

  function filterNodeAt(path) {
    let node = state.assetDraft && state.assetDraft.filterGroup;
    if (!node) return null;
    if (!path) return node;
    String(path).split('.').forEach(function (part) { node = node && node.conditions ? node.conditions[Number(part)] : null; });
    return node;
  }

  function removeFilterNode(path) {
    const parts = String(path).split('.');
    const index = Number(parts.pop());
    const parent = filterNodeAt(parts.join('.'));
    if (parent && parent.conditions) parent.conditions.splice(index, 1);
  }

  function focusAssetInput(id) {
    requestAnimationFrame(function () {
      const input = document.getElementById(id);
      if (input) input.focus({ preventScroll: true });
    });
  }

  function addEntry(kind) {
    syncAssetDraftFromForm();
    const synonym = kind === 'synonym';
    const inputId = synonym ? 'kv2SynonymInput' : 'kv2SimilarQuestionInput';
    const listName = synonym ? 'synonyms' : 'similarQuestions';
    const input = document.getElementById(inputId);
    const value = input ? input.value.trim() : '';
    if (!value) { state.assetError = synonym ? '请输入同义词后再添加' : '请输入相似问题后再添加'; render({ preserveModalScroll: true }); focusAssetInput(inputId); return; }
    if ((state.assetDraft[listName] || []).includes(value)) { state.assetError = synonym ? '该同义词已经存在' : '该相似问题已经存在'; render({ preserveModalScroll: true }); focusAssetInput(inputId); return; }
    state.assetDraft[listName].push(value);
    state.assetError = '';
    render({ preserveModalScroll: true });
    focusAssetInput(inputId);
  }

  function insertFormulaToken(token) {
    const input = document.querySelector('[name="metricFormulaExpression"]');
    const value = input ? input.value : String(state.assetDraft.metricFormulaExpression || '');
    const start = input && input.selectionStart != null ? input.selectionStart : value.length;
    const end = input && input.selectionEnd != null ? input.selectionEnd : value.length;
    syncAssetDraftFromForm();
    state.assetDraft.metricFormulaExpression = value.slice(0, start) + token + value.slice(end);
    const nextPosition = start + token.length;
    render({ preserveModalScroll: true });
    requestAnimationFrame(function () {
      const nextInput = document.querySelector('[name="metricFormulaExpression"]');
      if (nextInput) { nextInput.focus(); nextInput.setSelectionRange(nextPosition, nextPosition); }
    });
  }

  function addFilterListValue(input) {
    const value = input.value.trim();
    if (!value) return;
    const path = input.dataset.filterListPath;
    syncAssetDraftFromForm();
    const node = filterNodeAt(path);
    if (!node) return;
    const values = listValue(node.value);
    value.split(/[,，]/).map(function (item) { return item.trim(); }).filter(Boolean).forEach(function (item) { if (!values.includes(item)) values.push(item); });
    node.value = values;
    render({ preserveModalScroll: true });
    requestAnimationFrame(function () {
      const nextInput = app.querySelector('[data-filter-list-path="' + String(path) + '"]');
      if (nextInput) nextInput.focus({ preventScroll: true });
    });
  }

  function assetField(name, label, value, config) {
    const c = config || {};
    const required = c.required ? ' required' : '';
    const requiredAttr = c.required ? ' required' : '';
    const fieldValue = String(value == null ? '' : value);
    const isKey = name === 'assetId' && !c.disabled;
    const isMissing = Boolean(c.required && !fieldValue.trim());
    const hasInvalidKeyFormat = Boolean(isKey && fieldValue.trim() && !/^[a-z_-]+$/.test(fieldValue));
    const invalid = Boolean(state.assetValidationVisible && (isMissing || hasInvalidKeyFormat));
    const invalidAttr = invalid ? ' aria-invalid="true"' : '';
    const placeholder = c.placeholder ? ' placeholder="' + attr(c.placeholder) + '"' : '';
    const pattern = isKey ? '[a-z_-]+' : c.pattern;
    const maxLength = isKey ? 50 : c.maxLength;
    const showCounter = Boolean(c.showCounter && maxLength);
    let control = '';
    if (c.type === 'textarea') {
      control = '<textarea name="' + name + '" rows="' + (c.rows || 3) + '"' + placeholder + requiredAttr + invalidAttr + (maxLength ? ' maxlength="' + maxLength + '"' : '') + (showCounter ? ' data-input="counted-field"' : '') + (c.code ? ' class="kv2-code"' : '') + '>' + h(fieldValue) + '</textarea>';
    } else if (c.type === 'select') {
      control = '<select name="' + name + '"' + requiredAttr + invalidAttr + (c.input ? ' data-input="' + c.input + '"' : '') + '>' + (c.options || []).map(function (option) {
        const optionValue = typeof option === 'object' ? option.value : option;
        const optionLabel = typeof option === 'object' ? option.label : option;
        return '<option value="' + attr(optionValue) + '" ' + (String(value) === String(optionValue) ? 'selected' : '') + '>' + h(optionLabel) + '</option>';
      }).join('') + '</select>';
    } else {
      control = '<input name="' + name + '" type="' + attr(c.inputType || 'text') + '" value="' + attr(fieldValue) + '"' + placeholder + requiredAttr + invalidAttr + (maxLength ? ' maxlength="' + maxLength + '"' : '') + (c.disabled ? ' disabled' : '') + (c.code ? ' class="kv2-code"' : '') + (pattern ? ' pattern="' + attr(pattern) + '"' : '') + (isKey ? ' data-input="asset-key"' : '') + (c.step ? ' step="' + attr(c.step) + '"' : '') + '>';
    }
    const errorMessage = isMissing ? '请填写' + label : '标识仅支持英文小写字母、下划线（_）和横线（-）';
    const helper = isKey
      ? '<div class="kv2-field-meta"><span>用于系统唯一识别，仅支持英文小写字母、下划线（_）和横线（-）</span><span class="kv2-char-count" data-key-counter aria-live="polite">' + fieldValue.length + '/50</span></div>'
      : (showCounter
        ? '<div class="kv2-field-meta kv2-field-counter-only"><span class="kv2-char-count" data-field-counter aria-live="polite">' + fieldValue.length + '/' + maxLength + '</span></div>'
        : (c.help ? '<div class="kv2-control-help">' + h(c.help) + '</div>' : ''));
    return '<div class="kv2-field' + (invalid ? ' invalid' : '') + (c.className ? ' ' + c.className : '') + '"><label class="' + required.trim() + '">' + h(label) + '</label>' + control + (invalid ? '<div class="kv2-field-error">' + h(errorMessage) + '</div>' : '') + helper + '</div>';
  }

  function tableMetadataOptions() {
    return getSources().filter(function (source) { return source.type === '数据表' && source.enabled !== false; }).map(function (source) {
      const path = String(source.path || '').split('/').map(function (part) { return part.trim(); }).filter(Boolean);
      const value = path.length > 1 ? path.slice(-2).join('.') : source.name;
      return { value: value, label: value, description: tableCommentFor(source.name, source.comment) };
    });
  }

  function fieldMetadataOptions() {
    return bpcColumns.map(function (column) { return { value: column.name, label: column.name, description: column.description }; });
  }

  function relationshipTableMetadataOptions() {
    return [
      { value: 'bpc', label: 'bpc', description: tableCommentFor('bpc') },
      { value: 'organization_master', label: 'organization_master', description: tableCommentFor('organization_master') }
    ];
  }

  function relationshipFieldMetadataOptions(table) {
    if (table === 'organization_master') {
      return [
        { value: 'organization_code', label: 'organization_code', description: '组织编码' },
        { value: 'organization_name', label: 'organization_name', description: '组织名称' },
        { value: 'parent_organization_code', label: 'parent_organization_code', description: '上级组织编码' }
      ];
    }
    return fieldMetadataOptions();
  }

  function fileMetadataOptions() {
    return catalogSources.filter(function (source) { return source.type === '文件'; }).map(function (source) {
      return { value: source.name, label: source.name, description: '已完成解析与嵌入' };
    });
  }

  function renderSmartSelect(config) {
    const c = config || {};
    const pickerKey = String(c.key || c.name || c.target || 'picker');
    const listboxId = 'kv2SmartSelectListbox-' + pickerKey.replace(/[^a-zA-Z0-9_-]/g, '-');
    const open = !c.disabled && state.semanticPicker === pickerKey;
    const query = open ? state.semanticPickerQuery.trim().toLowerCase() : '';
    const options = (c.options || []).filter(function (option) {
      return !query || (String(option.label || option.value) + ' ' + String(option.description || '')).toLowerCase().includes(query);
    });
    const values = c.multiple ? listValue(c.value) : [String(c.value || '')].filter(Boolean);
    const selectedOption = (c.options || []).find(function (option) { return String(option.value) === String(c.value); });
    const data = ' data-picker-key="' + attr(pickerKey) + '" data-picker-target="' + attr(c.target || 'asset') + '"' + (c.prop ? ' data-picker-prop="' + attr(c.prop) + '"' : '') + (c.path != null ? ' data-picker-path="' + attr(c.path) + '"' : '') + (c.index != null ? ' data-picker-index="' + attr(c.index) + '"' : '') + (c.multiple ? ' data-picker-multiple="true"' : '');
    const hidden = c.name ? values.map(function (value) { return '<input type="hidden" name="' + attr(c.name) + '" value="' + attr(value) + '">'; }).join('') : '';
    const searchLabel = c.placeholder || '搜索并选择';
    const searchInput = '<input id="kv2SemanticPickerSearch" class="kv2-smart-select-input" autocomplete="off" role="combobox" aria-label="' + attr(searchLabel) + '" aria-autocomplete="list" aria-expanded="' + open + '" aria-controls="' + attr(listboxId) + '" aria-invalid="' + Boolean(c.invalid) + '" data-action="open-semantic-picker" data-input="semantic-picker-search"' + data + ' value="' + attr(open ? state.semanticPickerQuery : '') + '" placeholder="' + attr(values.length && c.multiple ? '' : searchLabel) + '" ' + (c.disabled ? 'disabled' : '') + '>';
    let control = '';
    if (c.multiple) {
      const tags = values.map(function (value, index) {
        const option = (c.options || []).find(function (item) { return String(item.value) === String(value); });
        return '<span class="kv2-smart-select-tag"><span>' + h(option ? option.label : value) + '</span><button type="button" data-action="' + attr(c.removeAction || 'remove-related-file') + '" data-index="' + index + '" aria-label="移除">×</button></span>';
      }).join('');
      control = '<div class="kv2-smart-select-control multiple ' + (open ? 'active' : '') + (c.invalid ? ' invalid' : '') + '">' + tags + searchInput + '<span class="kv2-smart-select-search">' + icons.search + '</span></div>';
    } else if (open) {
      control = '<div class="kv2-smart-select-control active ' + (c.invalid ? 'invalid' : '') + '">' + searchInput + '<span class="kv2-smart-select-search">' + icons.search + '</span></div>';
    } else {
      control = '<button type="button" class="kv2-smart-select-control single ' + (c.invalid ? 'invalid' : '') + '" aria-label="' + attr(searchLabel) + '" aria-haspopup="listbox" aria-expanded="false" aria-controls="' + attr(listboxId) + '" aria-invalid="' + Boolean(c.invalid) + '" data-action="open-semantic-picker"' + data + ' ' + (c.disabled ? 'disabled' : '') + '><span class="' + (selectedOption ? 'kv2-smart-select-value' : 'kv2-smart-select-placeholder') + '">' + h(selectedOption ? selectedOption.label : searchLabel) + '</span><span class="kv2-smart-select-search">' + icons.search + '</span></button>';
    }
    const bulkActions = c.multiple && c.bulkActions
      ? '<div class="kv2-smart-select-bulk"><span>已选 ' + values.length + ' / ' + (c.options || []).length + '</span><div><button type="button" data-action="select-all-semantic-options"' + data + ((c.options || []).length && (c.options || []).every(function (option) { return values.includes(String(option.value)); }) ? ' disabled' : '') + '>全选</button><button type="button" data-action="clear-semantic-options"' + data + (values.length ? '' : ' disabled') + '>清除</button></div></div>'
      : '';
    const dropdown = open ? '<div class="kv2-smart-select-dropdown">' + bulkActions + '<div id="' + attr(listboxId) + '" class="kv2-smart-select-options" role="listbox"' + (c.multiple ? ' aria-multiselectable="true"' : '') + '>' + (options.length ? options.map(function (option) {
      const selected = values.includes(String(option.value));
      return '<button type="button" class="kv2-smart-select-option ' + (selected ? 'selected' : '') + '" data-action="select-semantic-option"' + data + ' data-value="' + attr(option.value) + '" role="option" aria-selected="' + selected + '"><span class="kv2-smart-select-option-copy"><strong>' + h(option.label || option.value) + '</strong>' + (option.description ? '<small>' + h(option.description) + '</small>' : '') + '</span><span class="kv2-smart-select-check">' + (selected ? '✓' : '') + '</span></button>';
    }).join('') : '<div class="kv2-smart-select-empty">没有匹配项</div>') + '</div></div>' : '';
    return '<div class="kv2-smart-select ' + (open ? 'open' : '') + (c.disabled ? ' disabled' : '') + '">' + hidden + control + dropdown + '</div>';
  }

  function smartAssetField(name, label, value, config) {
    const c = config || {};
    return '<div class="kv2-field"><label class="' + (c.required ? 'required' : '') + '">' + h(label) + '</label>' + renderSmartSelect(Object.assign({}, c, { name: name, value: value, target: 'asset', prop: name })) + (c.help ? '<div class="kv2-control-help">' + h(c.help) + '</div>' : '') + '</div>';
  }

  function renderEntryEditor(kind, values) {
    const isSynonym = kind === 'synonym';
    const action = isSynonym ? 'add-synonym' : 'add-similar-question';
    const removeAction = isSynonym ? 'remove-synonym' : 'remove-similar-question';
    const inputId = isSynonym ? 'kv2SynonymInput' : 'kv2SimilarQuestionInput';
    const placeholder = isSynonym ? '输入一个同义词' : '输入一个相似问题';
    const rows = values.length ? values.map(function (value, index) {
      if (isSynonym) return '<span class="kv2-entry-tag">' + h(value) + '<button type="button" data-action="' + removeAction + '" data-index="' + index + '" aria-label="删除">×</button></span>';
      return '<div class="kv2-similar-row"><span>' + h(value) + '</span><button type="button" class="kv2-icon-btn danger" data-action="' + removeAction + '" data-index="' + index + '" aria-label="删除相似问题">' + icons.trash + '</button></div>';
    }).join('') : '';
    return '<div class="kv2-entry-editor ' + (isSynonym ? '' : 'questions') + '"><div class="kv2-entry-input"><input id="' + inputId + '" placeholder="' + placeholder + '"><button type="button" class="kv2-btn" data-action="' + action + '">＋ 添加</button></div><div class="' + (isSynonym ? 'kv2-entry-tags' : 'kv2-similar-list') + '">' + rows + '</div></div>';
  }

  const bpcColumns = [
    ['b28_s_kgd4b76', '合并科目'], ['cpmb_acctype', '科目类型'], ['cpmb_kgprv60', '科目分类'], ['cpmb_hir', '层级标识'],
    ['b28_s_kgd4b76_txtlg', '合并科目描述'], ['b28_s_kgdc8w9', '审计线索'], ['b28_s_kgdc8w9_txtlg', '审计线索描述'],
    ['b28_s_kgdsxrb', '客户_供应商编码'], ['b28_s_kgdsxrb_txtlg', '客户_供应商描述'], ['b28_s_kgd4rtr', '合并单元'],
    ['b28_s_kgd4rtr_txtlg', '合并单元描述'], ['b28_s_kgdp984', '合并变动'], ['b28_s_kgdp984_txtlg', '合并变动描述'],
    ['b28_s_kgd6bc6', '贸易伙伴'], ['b28_s_kgd6bc6_txtlg', '贸易伙伴描述'], ['b28_s_kgdk1oi', '附注维度1'],
    ['b28_s_kgdk1oi_txtlg', '附注维度1描述'], ['b28_s_kgduv2p', '预留维度1'], ['b28_s_kgduv2p_txtlg', '预留维度1描述'],
    ['b28_s_kgdo4wi', '产品组'], ['b28_s_kgdo4wi_txtlg', '产品组描述'], ['b28_s_kgd4kbn', '报表货币'],
    ['b28_s_kgd4kbn_txtlg', '报表货币描述'], ['b28_s_kgdxoi5', '合并范围'], ['b28_s_kgdxoi5_entity', '合并范围_实体'],
    ['b28_s_kgd4rtr_kgdxoi5', '合并单元_合并范围（实际使用）'], ['b28_s_kgd4rtr_kgdxoi5_txtlg', '合并单元_合并范围描述（实际使用）'],
    ['b28_s_kgdxoi5_txtlg', '合并范围描述'], ['b28_s_kgdbez8', '销售订单'], ['b28_s_kgdjz4b', '交易货币'],
    ['b28_s_kgdjz4b_txtlg', '交易货币描述'], ['b28_s_kgd353d', '合并期间'], ['b28_s_kgdbveh', '类型划分'],
    ['b28_s_kgdbveh_txtlg', '类型划分描述'], ['b28_s_kgdtvnx', '类别'], ['b28_s_kgdtvnx_txtlg', '类别描述'],
    ['b28_s_sdata', '数据'], ['account_path', '合并科目父节点路径']
  ].map(function (column) { return { name: column[0], description: column[1] }; });
  const bpcSampleRows = [
    { account: 'APL6000', accountName: '营业收入', organization: 'EO_1000', period: '202505', businessType: '—', category: 'ACT_LG', amount: '-12568000.00', currency: 'CNY' },
    { account: 'APL6600', accountName: '主营业务成本', organization: 'EO_1000', period: '202505', businessType: 'TFUA1000', category: 'ACT_LG', amount: '7392000.00', currency: 'CNY' },
    { account: 'APL6401', accountName: '税金及附加', organization: 'EO_1000', period: '202505', businessType: '—', category: 'ACT_LG', amount: '864000.00', currency: 'CNY' },
    { account: 'APL5001', accountName: '成本调整', organization: 'EO_1000', period: '202505', businessType: '—', category: 'ACT_LG', amount: '420000.00', currency: 'CNY' },
    { account: 'APL3000', accountName: '利润总额', organization: 'EO_1000', period: '202505', businessType: '—', category: 'ACT_LG', amount: '-3892000.00', currency: 'CNY' }
  ];
  // 筛选字段下拉沿用 Catalog 字段元数据，字段名下方展示灰色中文业务说明。
  const filterFields = bpcColumns.map(function (column) { return { value: column.name, label: column.name, description: column.description }; });
  const filterComparisons = [
    { value: 'equals', label: 'equals', description: '等于' }, { value: 'not_equals', label: 'not_equals', description: '不等于' },
    { value: 'greater_than', label: 'greater_than', description: '大于' }, { value: 'greater_than_or_equal', label: 'greater_than_or_equal', description: '大于等于' },
    { value: 'less_than', label: 'less_than', description: '小于' }, { value: 'less_than_or_equal', label: 'less_than_or_equal', description: '小于等于' },
    { value: 'in', label: 'in', description: '属于集合' }, { value: 'not_in', label: 'not_in', description: '不属于集合' },
    { value: 'contains', label: 'contains', description: '包含文本' }, { value: 'not_contains', label: 'not_contains', description: '不包含文本' },
    { value: 'starts_with', label: 'starts_with', description: '以文本开头' }, { value: 'not_starts_with', label: 'not_starts_with', description: '不以文本开头' },
    { value: 'ends_with', label: 'ends_with', description: '以文本结尾' }, { value: 'not_ends_with', label: 'not_ends_with', description: '不以文本结尾' },
    { value: 'between', label: 'between', description: '介于两个值之间' }, { value: 'is_null', label: 'is_null', description: '为空' },
    { value: 'is_not_null', label: 'is_not_null', description: '不为空' }
  ];

  function optionHtml(options, selected) {
    return options.map(function (option) { return '<option value="' + attr(option.value) + '" ' + (String(option.value) === String(selected) ? 'selected' : '') + '>' + h(option.label) + '</option>'; }).join('');
  }

  function validateSqlSyntax(sql) {
    const text = String(sql || '').trim();
    if (!text) return { valid: false, empty: true, message: '请输入 SQL' };
    let clean = '';
    let quote = '';
    let lineComment = false;
    let blockComment = false;
    let depth = 0;
    for (let i = 0; i < text.length; i += 1) {
      const char = text[i];
      const next = text[i + 1];
      if (lineComment) { if (char === '\n') { lineComment = false; clean += '\n'; } else clean += ' '; continue; }
      if (blockComment) { if (char === '*' && next === '/') { blockComment = false; clean += '  '; i += 1; } else clean += ' '; continue; }
      if (quote) {
        clean += ' ';
        if (char === '\\') { clean += ' '; i += 1; continue; }
        if (char === quote && next === quote) { clean += ' '; i += 1; continue; }
        if (char === quote) quote = '';
        continue;
      }
      if (char === '-' && next === '-') { lineComment = true; clean += '  '; i += 1; continue; }
      if (char === '#') { lineComment = true; clean += ' '; continue; }
      if (char === '/' && next === '*') { blockComment = true; clean += '  '; i += 1; continue; }
      if (char === "'" || char === '"' || char === '`') { quote = char; clean += ' '; continue; }
      if (char === '(') depth += 1;
      if (char === ')') { depth -= 1; if (depth < 0) return { valid: false, message: '存在多余的右括号“)”' }; }
      clean += char;
    }
    if (quote) return { valid: false, message: '字符串或标识符引号未闭合' };
    if (blockComment) return { valid: false, message: '块注释未闭合' };
    if (depth > 0) return { valid: false, message: '缺少右括号“)”' };
    const normalized = clean.replace(/\s+/g, ' ').trim();
    if (normalized.replace(/;\s*$/, '').includes(';')) return { valid: false, message: '只允许提交一条 SQL 语句' };
    if (!/^(select\b|with\b)/i.test(normalized)) return { valid: false, message: 'SQL 必须以 SELECT 或 WITH 开头' };
    if (/\b(insert|update|delete|drop|alter|truncate|create|replace|merge|grant|revoke|call|execute)\b/i.test(normalized)) return { valid: false, message: 'SQL 中不能包含写入、结构修改或过程调用语句' };
    if (/^with\b/i.test(normalized) && !/\bselect\b/i.test(normalized)) return { valid: false, message: 'WITH 子句后缺少 SELECT 查询' };
    if (/\bselect\s*(?:distinct\s+)?(?:from\b|;?$)/i.test(normalized)) return { valid: false, message: 'SELECT 后缺少查询字段或表达式' };
    if (/\b(from|join)\s*(?:where\b|join\b|on\b|group\b|order\b|having\b|limit\b|union\b|;?$)/i.test(normalized)) return { valid: false, message: 'FROM 或 JOIN 后缺少表名' };
    if (/\b(where|on|having)\s*(?:group\b|order\b|limit\b|offset\b|union\b|;?$)/i.test(normalized)) return { valid: false, message: '查询条件不完整' };
    if (/[,=+*/<>-]\s*;?$/.test(normalized) || /\b(and|or|where|from|join|on|having|union|by)\s*;?$/i.test(normalized)) return { valid: false, message: 'SQL 末尾存在未完成的表达式或子句' };
    if (/,(\s*)(from|where|group\s+by|order\s+by|having|limit|union)\b/i.test(normalized)) return { valid: false, message: '子句前存在多余的逗号' };
    return { valid: true, message: 'SQL 语法校验通过' };
  }

  function renderSqlEditor(value, rows, placeholder) {
    const sql = String(value || '');
    const result = validateSqlSyntax(sql);
    const active = Boolean(sql.trim()) || state.assetValidationVisible;
    const status = active ? result.message : '输入 SQL 后将自动进行语法和只读校验';
    const statusClass = active ? (result.valid ? 'success' : 'error') : 'neutral';
    return '<div class="kv2-field kv2-sql-editor ' + (active ? (result.valid ? 'valid' : 'invalid') : '') + '"><label class="required">SQL</label><textarea name="sql" rows="' + rows + '" class="kv2-code" required data-input="sql-editor" placeholder="' + attr(placeholder) + '" aria-invalid="' + (active && !result.valid) + '">' + h(sql) + '</textarea><div class="kv2-sql-validation ' + statusClass + '" data-sql-validation role="status">' + (active && result.valid ? '✓ ' : active ? '！ ' : '') + h(status) + '</div><div class="kv2-control-help">仅支持一条只读的 SELECT 或 WITH 查询，不允许写入、修改结构或执行多条语句。</div></div>';
  }

  function filterConditionIssues(condition) {
    const comparison = condition.comparison || 'equals';
    const noValue = comparison === 'is_null' || comparison === 'is_not_null';
    const listMode = comparison === 'in' || comparison === 'not_in';
    return {
      field: !String(condition.field || '').trim(),
      value: !noValue && (listMode ? listValue(condition.value).length === 0 : !String(condition.value || '').trim()),
      valueEnd: comparison === 'between' && !String(condition.valueEnd || '').trim()
    };
  }

  function renderFilterCondition(condition, path, validateEmpty) {
    const noValue = condition.comparison === 'is_null' || condition.comparison === 'is_not_null';
    const range = condition.comparison === 'between';
    const listMode = condition.comparison === 'in' || condition.comparison === 'not_in';
    const listItems = listMode ? listValue(condition.value) : [];
    const issues = state.assetValidationVisible && validateEmpty ? filterConditionIssues(condition) : { field: false, value: false, valueEnd: false };
    const valueInvalid = issues.value || issues.valueEnd;
    const valueControl = noValue ? '<div class="kv2-filter-no-value">此比较方式无需填写值</div>' : range
      ? '<div class="kv2-filter-range"><input data-filter-path="' + path + '" data-filter-prop="value" value="' + attr(condition.value || '') + '" placeholder="起始值"><span>—</span><input data-filter-path="' + path + '" data-filter-prop="valueEnd" value="' + attr(condition.valueEnd || '') + '" placeholder="结束值"></div>'
      : listMode ? '<div class="kv2-filter-tag-input">' + listItems.map(function (item, index) { return '<span class="kv2-entry-tag">' + h(item) + '<button type="button" data-action="remove-filter-value" data-path="' + path + '" data-index="' + index + '">×</button></span>'; }).join('') + '<input data-filter-list-path="' + path + '" placeholder="输入后回车，可填写多个值"></div>'
        : '<input data-filter-path="' + path + '" data-filter-prop="value" value="' + attr(condition.value || '') + '" placeholder="输入比较值">';
    return '<div class="kv2-filter-condition ' + ((issues.field || valueInvalid) ? 'invalid' : '') + '" data-filter-condition-path="' + attr(path) + '"><div class="kv2-filter-control ' + (issues.field ? 'invalid' : '') + '"><span>字段</span>' + renderSmartSelect({ key: 'filter-field-' + path, value: condition.field || '', target: 'filter-field', path: path, options: filterFields, placeholder: '搜索并选择字段', invalid: issues.field }) + (issues.field ? '<small class="kv2-filter-error">请选择字段</small>' : '') + '</div><div class="kv2-filter-control"><span>比较方式</span>' + renderSmartSelect({ key: 'filter-comparison-' + path, value: condition.comparison || 'equals', target: 'filter-comparison', path: path, options: filterComparisons, placeholder: '搜索并选择比较方式' }) + '</div><div class="kv2-filter-control ' + (valueInvalid ? 'invalid' : '') + '"><span>值</span>' + valueControl + (valueInvalid ? '<small class="kv2-filter-error">' + (range ? '请完整填写起始值和结束值' : listMode ? '请至少添加一个值' : '请输入比较值') + '</small>' : '') + '</div><button type="button" class="kv2-icon-btn danger" data-action="remove-filter-node" data-path="' + path + '" aria-label="删除条件" title="删除条件">' + icons.trash + '</button></div>';
  }

  function renderFilterGroup(group, path, nested) {
    const validateEmpty = state.semCategory === 'constraint' || nested || group.conditions.length > 1 || hasMeaningfulFilter(group);
    const children = group.conditions.map(function (node, index) {
      const nodePath = path ? path + '.' + index : String(index);
      const connector = index ? '<select class="kv2-filter-connector" data-filter-path="' + nodePath + '" data-filter-prop="connector"><option value="and" ' + ((node.connector || group.logic) === 'and' ? 'selected' : '') + '>AND</option><option value="or" ' + ((node.connector || group.logic) === 'or' ? 'selected' : '') + '>OR</option></select>' : '';
      return '<div class="kv2-filter-child">' + connector + (node.type === 'group' ? renderFilterGroup(node, nodePath, true) : renderFilterCondition(node, nodePath, validateEmpty)) + '</div>';
    }).join('');
    return '<div class="kv2-filter-group ' + (nested ? 'nested' : 'root') + '"><div class="kv2-filter-group-head"><div><strong>' + (nested ? '条件组' : '筛选逻辑') + '</strong><span>' + (nested ? '组内条件会先计算，再作为一个整体与组外条件组合。' : '每个 AND / OR 都连接上下两条条件，可分别选择。') + '</span></div>' + (nested ? '<button type="button" class="kv2-filter-remove-group" data-action="remove-filter-node" data-path="' + path + '" aria-label="删除条件组" title="删除条件组">' + icons.trash + '</button>' : '') + '</div><div class="kv2-filter-group-body">' + children + '</div><div class="kv2-filter-actions"><button type="button" class="kv2-filter-add-condition" data-action="add-filter-condition" data-path="' + path + '">＋ 添加条件</button><button type="button" class="kv2-filter-add-group" data-action="add-filter-group" data-path="' + path + '">＋ 添加条件组</button></div></div>';
  }

  function sqlComparison(comparison, value, end) {
    const escaped = "'" + String(value || '').replace(/'/g, "''") + "'";
    const map = { equals: '=', not_equals: '<>', greater_than: '>', greater_than_or_equal: '>=', less_than: '<', less_than_or_equal: '<=', contains: 'LIKE', not_contains: 'NOT LIKE', starts_with: 'LIKE', not_starts_with: 'NOT LIKE', ends_with: 'LIKE', not_ends_with: 'NOT LIKE' };
    if (comparison === 'is_null') return 'IS NULL';
    if (comparison === 'is_not_null') return 'IS NOT NULL';
    if (comparison === 'between') return 'BETWEEN ' + escaped + " AND '" + String(end || '').replace(/'/g, "''") + "'";
    if (comparison === 'in' || comparison === 'not_in') return (comparison === 'not_in' ? 'NOT IN' : 'IN') + ' (' + listValue(value).map(function (v) { return "'" + v.replace(/'/g, "''") + "'"; }).join(', ') + ')';
    if (comparison === 'contains' || comparison === 'not_contains') return map[comparison] + " '%" + String(value || '').replace(/'/g, "''") + "%'";
    if (comparison === 'starts_with' || comparison === 'not_starts_with') return map[comparison] + " '" + String(value || '').replace(/'/g, "''") + "%'";
    if (comparison === 'ends_with' || comparison === 'not_ends_with') return map[comparison] + " '%" + String(value || '').replace(/'/g, "''") + "'";
    return (map[comparison] || '=') + ' ' + escaped;
  }

  function hasMeaningfulFilter(node) {
    if (!node) return false;
    if (node.type === 'group' || Array.isArray(node.conditions)) return node.conditions.some(hasMeaningfulFilter);
    if (String(node.field || '').trim()) return true;
    if (String(node.valueEnd || '').trim()) return true;
    return Array.isArray(node.value) ? node.value.length > 0 : Boolean(String(node.value || '').trim());
  }

  function filterSqlLines(node, depth) {
    const indent = '  '.repeat(depth);
    if (node.type === 'condition') {
      return [indent + (node.field || '<field>') + ' ' + sqlComparison(node.comparison || 'equals', node.value, node.valueEnd)];
    }
    const meaningfulChildren = node.conditions.filter(hasMeaningfulFilter);
    if (!meaningfulChildren.length) return [];
    const lines = [indent + '('];
    meaningfulChildren.forEach(function (child, index) {
      const childLines = filterSqlLines(child, depth + 1);
      if (index > 0 && childLines.length) {
        childLines[0] = '  '.repeat(depth + 1) + String(child.connector || node.logic || 'and').toUpperCase() + ' ' + childLines[0].trimStart();
      }
      lines.push.apply(lines, childLines);
    });
    lines.push(indent + ')');
    return lines;
  }

  function filterSql(node) {
    return filterSqlLines(node, 0).join('\n').trimStart();
  }

  function hasInvalidFilter(node, isRoot) {
    if (node.type === 'group' || Array.isArray(node.conditions)) {
      const meaningfulChildren = node.conditions.filter(hasMeaningfulFilter);
      if (!isRoot && meaningfulChildren.length === 0) return true;
      if (isRoot && meaningfulChildren.length === 0) return false;
      return node.conditions.some(function (child) { return !hasMeaningfulFilter(child) || hasInvalidFilter(child, false); });
    }
    if (!hasMeaningfulFilter(node)) return false;
    if (!node.field || !node.comparison) return true;
    if (node.comparison === 'is_null' || node.comparison === 'is_not_null') return false;
    if (Array.isArray(node.value) ? node.value.length === 0 : !String(node.value || '').trim()) return true;
    return node.comparison === 'between' && !String(node.valueEnd || '').trim();
  }

  function focusFirstAssetError() {
    requestAnimationFrame(function () {
      const target = app.querySelector('.kv2-field.invalid input, .kv2-field.invalid textarea, .kv2-field.invalid select, .kv2-filter-control.invalid .kv2-smart-select-control, .kv2-filter-control.invalid input, .kv2-filter-control.invalid select, .kv2-form-error');
      if (!target) return;
      if (typeof target.scrollIntoView === 'function') target.scrollIntoView({ block: 'center', behavior: 'smooth' });
      if (typeof target.focus === 'function' && !target.classList.contains('kv2-form-error')) target.focus();
    });
  }

  function refreshFilterConditionValidation(path) {
    if (!state.assetValidationVisible) return;
    const node = filterNodeAt(path);
    const row = app.querySelector('[data-filter-condition-path="' + String(path) + '"]');
    if (!node || !row) return;
    const issues = filterConditionIssues(node);
    const controls = row.querySelectorAll('.kv2-filter-control');
    const fieldControl = controls[0];
    const valueControl = controls[2];
    if (fieldControl) {
      fieldControl.classList.toggle('invalid', issues.field);
      if (!issues.field) { const error = fieldControl.querySelector('.kv2-filter-error'); if (error) error.remove(); }
    }
    if (valueControl) {
      const valueInvalid = issues.value || issues.valueEnd;
      valueControl.classList.toggle('invalid', valueInvalid);
      if (!valueInvalid) { const error = valueControl.querySelector('.kv2-filter-error'); if (error) error.remove(); }
    }
    row.classList.toggle('invalid', issues.field || issues.value || issues.valueEnd);
  }

  function currentMetricSelfKeys() {
    if (state.semCategory !== 'metric') return [];
    const keys = [
      state.assetDraft && state.assetDraft.assetId,
      state.editingAsset && state.editingAsset.fields && state.editingAsset.fields.assetId
    ].map(function (key) { return String(key || '').trim(); }).filter(Boolean);
    return Array.from(new Set(keys));
  }

  function metricReferenceItems(sourceTable, excludedKeys) {
    const selectedTable = String(sourceTable || '').trim();
    const excluded = new Set(excludedKeys || []);
    const items = getAssets().filter(function (asset) {
      const fields = asset.fields || {};
      return asset.category === 'metric'
        && fields.metricType === 'aggregate'
        && fields.sourceTable === selectedTable
        && asset !== state.editingAsset
        && !excluded.has(String(fields.assetId || ''));
    }).map(function (asset) {
      return { value: asset.fields.assetId, name: asset.name || asset.fields.assetId };
    });
    return items;
  }

  function formulaContainsMetricKey(expression, metricKey) {
    const escapedId = String(metricKey || '').replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
    return Boolean(escapedId && new RegExp('(^|[^a-z_-])' + escapedId + '(?=$|[^a-z_-])').test(String(expression || '')));
  }

  function formulaTokens(expression, references) {
    const text = String(expression || '');
    const referenceIds = references.map(function (item) { return item.value; }).sort(function (a, b) { return b.length - a.length; });
    const tokens = [];
    let index = 0;
    while (index < text.length) {
      if (/\s/.test(text[index])) { index += 1; continue; }
      const reference = referenceIds.find(function (id) {
        if (!text.startsWith(id, index)) return false;
        const next = text[index + id.length] || '';
        return !/[a-z0-9_]/.test(next);
      });
      if (reference) {
        tokens.push({ type: 'operand', value: reference, kind: 'metric' });
        index += reference.length;
        continue;
      }
      const rest = text.slice(index);
      const number = rest.match(/^(?:\d+(?:\.\d*)?|\.\d+)/);
      if (number) {
        tokens.push({ type: 'operand', value: number[0], kind: 'number' });
        index += number[0].length;
        continue;
      }
      const identifier = rest.match(/^[a-z][a-z0-9_-]*/);
      if (identifier) return { error: '指标标识“' + identifier[0] + '”不存在，请从上方选择已有指标' };
      const char = text[index];
      if (char === '(') tokens.push({ type: 'left-parenthesis', value: char });
      else if (char === ')') tokens.push({ type: 'right-parenthesis', value: char });
      else if ('+-*/'.includes(char)) tokens.push({ type: 'operator', value: char });
      else return { error: char === '.' ? '小数格式不正确' : '公式包含不支持的字符' };
      index += 1;
    }
    return { tokens: tokens };
  }

  function formulaStructureMessage(tokens) {
    let expectingOperand = true;
    let parenthesisDepth = 0;
    let operandCount = 0;
    let binaryOperatorCount = 0;
    let previous = null;
    for (const token of tokens) {
      if (token.type === 'operand') {
        if (!expectingOperand) return previous && previous.type === 'right-parenthesis' ? '右括号后缺少运算符' : '两个指标或数值之间缺少运算符';
        operandCount += 1;
        expectingOperand = false;
      } else if (token.type === 'left-parenthesis') {
        if (!expectingOperand) return '“(”前缺少运算符';
        parenthesisDepth += 1;
      } else if (token.type === 'right-parenthesis') {
        if (parenthesisDepth === 0) return '存在多余的右括号“)”';
        if (expectingOperand) return previous && previous.type === 'left-parenthesis' ? '括号内不能为空' : '“)”前缺少指标或数值';
        parenthesisDepth -= 1;
        expectingOperand = false;
      } else if (token.type === 'operator') {
        const unaryMinus = token.value === '-' && expectingOperand && (!previous || previous.type === 'left-parenthesis' || previous.type === 'operator');
        if (expectingOperand && !unaryMinus) return '运算符“' + token.value + '”前缺少指标或数值';
        if (!expectingOperand) {
          binaryOperatorCount += 1;
          expectingOperand = true;
        }
      }
      previous = token;
    }
    if (parenthesisDepth > 0) return '缺少右括号“)”';
    if (expectingOperand) return previous && previous.type === 'operator' ? '运算符“' + previous.value + '”后缺少指标或数值' : '公式未完成';
    if (operandCount < 2 || binaryOperatorCount < 1) return '公式至少需要两个指标或数字，并使用运算符连接';
    return '';
  }

  function formulaValidationMessage(expression) {
    const rawText = String(expression || '');
    const text = rawText.trim();
    if (!text) return '请输入公式';
    if (rawText.length > 200) return '公式长度不能超过 200 个字符';
    if (/[^a-z0-9_+*/().\s-]/.test(text)) return '公式包含不支持的字符';
    const sourceTable = state.assetDraft && state.assetDraft.sourceTable;
    if (!sourceTable) return '请先选择来源表';
    const selfKey = currentMetricSelfKeys().find(function (key) { return formulaContainsMetricKey(text, key); });
    if (selfKey) return '派生指标不能引用自身标识“' + selfKey + '”';
    const references = metricReferenceItems(sourceTable, currentMetricSelfKeys()).slice().sort(function (a, b) { return b.value.length - a.value.length; });
    const tokenized = formulaTokens(text, references);
    return tokenized.error || formulaStructureMessage(tokenized.tokens);
  }

  function formulaPreviewText(expression, table) {
    const sourceTable = table || (state.assetDraft && state.assetDraft.sourceTable);
    const references = metricReferenceItems(sourceTable).slice().sort(function (a, b) { return b.value.length - a.value.length; });
    return references.reduce(function (preview, item) {
      const escapedId = item.value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
      return preview.replace(new RegExp('(^|[^a-z_-])' + escapedId + '(?=$|[^a-z_-])', 'g'), function (_match, prefix) { return prefix + item.name; });
    }, String(expression || ''));
  }

  function renderFilterBuilder(group, constraint) {
    const sql = filterSql(group);
    const keyword = constraint ? 'WHERE' : 'WHEN';
    const preview = sql ? keyword + ' ' + sql : keyword + ' (\n  <condition>\n)';
    return '<div class="kv2-filter-builder-v2">' + renderFilterGroup(group, '', false) + '<div class="kv2-filter-sql"><div><strong>' + (constraint ? 'WHERE 条件预览' : 'SQL 层级预览') + '</strong><span>' + (constraint ? '预览最终追加到查询中的过滤条件；保存后在关联表参与查询时强制生效。' : '用于理解条件组的括号和组合顺序；实际 SQL 仍由指标编译器生成。') + '</span></div><pre data-filter-preview>' + h(preview) + '</pre></div></div>';
  }

  function relationshipSqlPreview(fields) {
    const f = fields || {};
    const leftTable = f.leftTable || '<left_table>';
    const rightTable = f.rightTable || '<right_table>';
    const joinType = f.joinType || 'LEFT JOIN';
    const conditions = (f.joinConditions || []).map(function (condition) {
      const leftField = condition.leftField || '<left_field>';
      const rightField = condition.rightField || '<right_field>';
      return leftTable + '.' + leftField + ' ' + relationshipComparisonSymbol(condition.comparison) + ' ' + rightTable + '.' + rightField;
    });
    if (!conditions.length) conditions.push(leftTable + '.<left_field> = ' + rightTable + '.<right_field>');
    return 'FROM ' + leftTable + '\n' + joinType + ' ' + rightTable + '\n  ON ' + conditions.join('\n AND ');
  }

  function renderStandardQaUsageMode(value, unstructured) {
    const mode = value === 'force' ? 'force' : 'reference';
    const referenceText = unstructured ? '作为回答参考，由系统结合关联文件和其他语义配置综合判断。' : '作为回答和 SQL 生成的参考，由系统结合其他语义配置综合判断。';
    const forceText = unstructured ? '查询意图、业务对象和关键条件一致时，直接使用配置的标准答案。' : '查询意图、业务对象和关键筛选条件一致时，可强制使用配置 SQL。';
    return '<div class="kv2-field kv2-qa-usage-field"><label class="required">使用方式</label><div class="kv2-choice-group" role="radiogroup" aria-label="标准问答使用方式">'
      + '<label class="kv2-choice-option"><input type="radio" name="usageMode" value="reference" required ' + (mode === 'reference' ? 'checked' : '') + '><span><strong>参考使用</strong><small>' + h(referenceText) + '</small></span></label>'
      + '<label class="kv2-choice-option"><input type="radio" name="usageMode" value="force" required ' + (mode === 'force' ? 'checked' : '') + '><span><strong>强制使用</strong><small>' + h(forceText) + '</small></span></label>'
      + '</div></div>';
  }

  function renderDynamicResultProcessingMode(value) {
    const mode = value === 'delegated_analysis' ? 'delegated_analysis' : 'direct_context';
    return '<div class="kv2-field"><label class="required">查询结果处理</label><div class="kv2-choice-group" role="radiogroup" aria-label="动态查询结果处理方式">'
      + '<label class="kv2-choice-option"><input type="radio" name="resultProcessingMode" value="direct_context" required ' + (mode === 'direct_context' ? 'checked' : '') + '><span><strong>直接注入主 Agent 上下文</strong><small>将动态查询的原始结果完整写入主 Agent 上下文，由主 Agent 基于结果继续推理和回答。</small></span></label>'
      + '<label class="kv2-choice-option"><input type="radio" name="resultProcessingMode" value="delegated_analysis" required ' + (mode === 'delegated_analysis' ? 'checked' : '') + '><span><strong>调用子 Agent 分析</strong><small>将动态查询结果交给子 Agent 筛选、汇总和分析，再将子 Agent 的分析结果写入主 Agent 上下文。</small></span></label>'
      + '</div></div>';
  }

  function renderMetricFields(f) {
    let html = '<div class="kv2-form-section kv2-metric-primary"><h4>基本信息</h4>'
      + assetField('assetId', '指标标识', f.assetId, { required: true, placeholder: 'main_business_cost', code: true })
      + assetField('name', '名称', f.name, { required: true, placeholder: '请输入业务人员可识别的名称' })
      + assetField('description', '指标说明', f.description, { type: 'textarea', rows: 3, placeholder: '说明指标口径、适用范围和使用注意事项' })
      + '<div class="kv2-field"><label>同义词</label>' + renderEntryEditor('synonym', f.synonyms || []) + '<div class="kv2-control-help">输入后按回车或点击添加，已添加词条可逐个删除。</div></div>'
      + assetField('metricType', '指标类型', f.metricType || 'aggregate', { required: true, type: 'select', input: 'metric-type', options: [{ value: 'aggregate', label: '基础指标' }, { value: 'derived', label: '派生指标' }], help: '基础指标从一张来源表聚合计算；派生指标只引用已有指标进行四则运算。' }) + '</div><div class="kv2-form-divider"></div>';
    if (f.metricType === 'derived') {
      const sourceSelected = Boolean(f.sourceTable);
      const formulaError = f.metricFormulaExpression ? formulaValidationMessage(f.metricFormulaExpression) : '';
      const metricOptions = metricReferenceItems(f.sourceTable, currentMetricSelfKeys()).map(function (item) { return { value: item.value, label: item.value, description: item.name }; });
      const formulaLength = String(f.metricFormulaExpression || '').length;
      html += '<div class="kv2-form-section"><h4>派生指标计算</h4>'
        + smartAssetField('sourceTable', '来源表', f.sourceTable || '', { required: true, key: 'derived-metric-source-table', options: tableMetadataOptions(), placeholder: '先选择来源表', help: '一期仅支持同一来源表内的基础指标。切换来源表会清空当前公式。' })
        + '<div class="kv2-formula-v2 ' + (sourceSelected ? '' : 'is-disabled') + '"><div class="kv2-formula-head"><strong>派生指标公式</strong><span>' + (sourceSelected ? '仅展示“' + h(f.sourceTable) + '”下的基础指标；选择指标后通过运算符组合，也可以直接输入指标标识、数字和括号。' : '请先选择来源表，系统将只展示该表下可引用的基础指标。') + '</span></div><div class="kv2-formula-toolbar"><div class="kv2-formula-metric-picker">' + renderSmartSelect({ key: 'formula-metric', target: 'formula', value: '', options: metricOptions, placeholder: sourceSelected ? '选择基础指标并插入' : '请先选择来源表', disabled: !sourceSelected }) + '</div><div class="kv2-formula-operators" role="group" aria-label="插入运算符"><button type="button" data-action="insert-formula" data-token=" + " title="加"' + (sourceSelected ? '' : ' disabled') + '>+</button><button type="button" data-action="insert-formula" data-token=" - " title="减"' + (sourceSelected ? '' : ' disabled') + '>−</button><button type="button" data-action="insert-formula" data-token=" * " title="乘"' + (sourceSelected ? '' : ' disabled') + '>×</button><button type="button" data-action="insert-formula" data-token=" / " title="除"' + (sourceSelected ? '' : ' disabled') + '>÷</button><button type="button" data-action="insert-formula" data-token="(" title="左括号"' + (sourceSelected ? '' : ' disabled') + '>(</button><button type="button" data-action="insert-formula" data-token=")" title="右括号"' + (sourceSelected ? '' : ' disabled') + '>)</button></div></div><div class="kv2-formula-input ' + (formulaError ? 'error' : '') + '"><span aria-hidden="true">=</span><input name="metricFormulaExpression" value="' + attr(f.metricFormulaExpression || '') + '" maxlength="200" placeholder="(bpc_revenue - main_business_cost) / bpc_revenue"' + (sourceSelected ? '' : ' disabled') + ' aria-invalid="' + Boolean(formulaError) + '" aria-describedby="kv2FormulaError"><button type="button" class="kv2-formula-clear" data-action="clear-formula"' + (sourceSelected ? '' : ' disabled') + '>清空</button></div><div class="kv2-formula-meta"><div id="kv2FormulaError" class="kv2-formula-live-error" data-formula-error role="status" aria-live="polite">' + h(formulaError) + '</div><span class="kv2-char-count" data-formula-counter aria-live="polite">' + formulaLength + '/200</span></div><div class="kv2-formula-preview"><span>公式预览</span><code data-formula-preview>' + h(f.metricFormulaExpression ? (formulaError ? '公式未完成' : formulaPreviewText(f.metricFormulaExpression)) : '公式未完成') + '</code></div></div></div>';
    } else {
      html += '<div class="kv2-form-section"><h4>基础指标计算</h4>'
      + smartAssetField('sourceTable', '来源表', f.sourceTable || '', { required: true, key: 'metric-source-table', options: tableMetadataOptions(), placeholder: '搜索并选择来源表' })
        + '<div class="kv2-form-grid-three">'
        + assetField('aggregationFunction', '聚合函数', f.aggregationFunction || 'sum', { required: true, type: 'select', options: ['sum', 'avg', 'min', 'max', 'count'] })
        + smartAssetField('aggregationField', '聚合字段', f.aggregationField || '', { required: true, key: 'metric-aggregation-field', options: fieldMetadataOptions(), placeholder: '搜索并选择聚合字段' })
        + assetField('multiplier', '乘数', f.multiplier == null ? 1 : f.multiplier, { inputType: 'number', step: 'any' }) + '</div><div class="kv2-filter-section-title"><strong>固定筛选条件</strong></div>' + renderFilterBuilder(f.filterGroup, false) + '</div>';
    }
    return html;
  }

  function renderAssetModal() {
    const cat = categories.find(function (c) { return c.key === state.semCategory; }) || categories[0];
    const asset = state.editingAsset;
    const f = state.assetDraft || {};
    let fields = '';
    if (cat.key === 'metric') {
      fields = renderMetricFields(f);
    } else if (cat.key === 'schema') {
      const normalizedColumnSearch = state.schemaColumnSearch.trim().toLowerCase();
      const columns = (f.schemaColumns || []).map(function (column, index) { return { column: column, index: index }; }).filter(function (entry) { return !normalizedColumnSearch || (entry.column.name + ' ' + entry.column.description).toLowerCase().includes(normalizedColumnSearch); });
      fields = '<div class="kv2-schema-editor">'
        + assetField('tableDescription', '表描述', f.tableDescription || f.description || 'BPC 合并报表明细', { disabled: true })
        + assetField('description', '表补充说明', f.description || '', { type: 'textarea', rows: 2, placeholder: '输入该表的业务含义和适用场景' })
        + '<div class="kv2-form-divider"></div><label class="kv2-schema-search"><span>' + icons.search + '</span><input value="' + attr(state.schemaColumnSearch) + '" data-input="schema-column-search" placeholder="搜索列名或列描述" aria-label="搜索列名或列描述"></label><div class="kv2-schema-columns"><div class="kv2-schema-row head"><span>列名</span><span>列描述</span><span>列补充说明</span><span>启用</span></div>'
        + (columns.length ? columns.map(function (entry) { const column = entry.column; return '<div class="kv2-schema-row"><input value="' + attr(column.name) + '" disabled><input value="' + attr(column.description) + '" disabled><input value="' + attr(column.supplementalDescription || '') + '" data-schema-index="' + entry.index + '" data-schema-prop="supplementalDescription" placeholder="请输入列补充说明"><button type="button" class="kv2-switch ' + (column.enabled ? '' : 'off') + '" data-action="toggle-schema-column" data-index="' + entry.index + '" aria-label="启用列"></button></div>'; }).join('') : '<div class="kv2-empty">暂无匹配列</div>')
        + '</div><input type="hidden" name="assetId" value="' + attr(f.assetId || f.table || '') + '"><input type="hidden" name="name" value="' + attr(f.name || f.assetId || '表和字段说明') + '"><input type="hidden" name="table" value="' + attr(f.table || f.assetId || '') + '"></div>';
    } else if (cat.key === 'business-rule') {
      fields = '<div class="kv2-form-section kv2-metric-primary">'
        + assetField('assetId', '业务规则标识', f.assetId || '', { required: true, placeholder: 'annual_opening_period', code: true })
        + '<div class="kv2-field"><label class="required">关联表</label>' + renderSmartSelect({ key: 'business-related-tables', name: 'relatedTables', target: 'related-tables', value: f.relatedTables || [], multiple: true, bulkActions: true, options: tableMetadataOptions(), placeholder: '搜索并选择关联表', removeAction: 'remove-related-table' }) + '<div class="kv2-control-help">默认全选当前知识库内的可用表，支持下拉搜索和多选。</div></div>'
        + assetField('ruleContent', '规则内容', f.ruleContent || f.content || '', { required: true, type: 'textarea', rows: 5, placeholder: '请输入自然语言规则描述' }) + '</div>';
    } else if (cat.key === 'unstructured-rule') {
      fields = '<div class="kv2-form-section kv2-metric-primary">'
        + assetField('assetId', '业务规则标识', f.assetId || '', { required: true, placeholder: 'bpc_document_metric_policy', code: true })
        + '<div class="kv2-field"><label class="required">关联文件</label>' + renderSmartSelect({ key: 'business-related-files', name: 'relatedFiles', target: 'related-files', value: f.relatedFiles || [], multiple: true, options: fileMetadataOptions(), placeholder: '搜索并选择已解析文件', removeAction: 'remove-related-file' }) + '<div class="kv2-control-help">支持搜索和多选；仅可关联已完成文档解析与嵌入的非结构化文件。</div></div>'
        + assetField('ruleContent', '规则内容', f.ruleContent || f.content || '', { required: true, type: 'textarea', rows: 5, placeholder: '请输入自然语言规则描述' }) + '</div>';
    } else if (cat.key === 'constraint') {
      fields = '<div class="kv2-form-section kv2-metric-primary">'
        + assetField('assetId', '强制约束标识', f.assetId || '', { required: true, placeholder: 'bpc_currency_cny', code: true })
        + smartAssetField('sourceTable', '关联表', f.sourceTable || '', { required: true, key: 'constraint-source-table', options: tableMetadataOptions(), placeholder: '搜索并选择关联表', help: '一条强制约束只绑定一张表；该表参与查询时，约束条件将被自动加入 SQL。' })
        + '<div class="kv2-filter-section-title"><strong>约束条件</strong></div>' + renderFilterBuilder(f.filterGroup, true) + '</div>';
    } else if (cat.key === 'relationship') {
      const tableOptions = relationshipTableMetadataOptions();
      const relationshipPreview = relationshipSqlPreview(f);
      fields = '<div class="kv2-form-section kv2-metric-primary">'
        + assetField('assetId', '表关联标识', f.assetId || '', { required: true, placeholder: 'bpc_organization', code: true })
        + assetField('description', '表关联说明', f.description || '', { type: 'textarea', rows: 3, maxLength: 200, showCounter: true, placeholder: '说明两张表通过什么业务字段建立关联' })
        + '<div class="kv2-form-divider"></div><div class="kv2-asset-form-grid">'
        + smartAssetField('leftTable', '左表', f.leftTable || '', { required: true, key: 'relationship-left-table', options: tableOptions, placeholder: '搜索并选择左表' })
        + smartAssetField('rightTable', '右表', f.rightTable || '', { required: true, key: 'relationship-right-table', options: tableOptions, placeholder: '搜索并选择右表' }) + '</div>'
        + assetField('joinType', 'JOIN 类型', f.joinType || 'LEFT JOIN', { required: true, type: 'select', options: ['LEFT JOIN', 'INNER JOIN', 'RIGHT JOIN', 'FULL JOIN'] })
        + '<div class="kv2-relationship-list"><strong>关联条件</strong>' + (f.joinConditions || []).map(function (condition, index) { return '<div class="kv2-join-row"><div class="kv2-field"><label class="required">左表字段</label>' + renderSmartSelect({ key: 'relationship-left-field-' + index, target: 'join-field', prop: 'leftField', index: index, value: condition.leftField || '', options: relationshipFieldMetadataOptions(f.leftTable), placeholder: '搜索并选择左表字段', disabled: !f.leftTable }) + '<input type="hidden" name="leftField_' + index + '" value="' + attr(condition.leftField || '') + '"></div><div class="kv2-field"><label class="required">比较方式</label>' + renderSmartSelect({ key: 'relationship-comparison-' + index, target: 'join-comparison', index: index, value: condition.comparison || 'equals', options: relationshipComparisons, placeholder: '搜索并选择比较方式' }) + '<input type="hidden" name="comparison_' + index + '" value="' + attr(condition.comparison || 'equals') + '"></div><div class="kv2-field"><label class="required">右表字段</label>' + renderSmartSelect({ key: 'relationship-right-field-' + index, target: 'join-field', prop: 'rightField', index: index, value: condition.rightField || '', options: relationshipFieldMetadataOptions(f.rightTable), placeholder: '搜索并选择右表字段', disabled: !f.rightTable }) + '<input type="hidden" name="rightField_' + index + '" value="' + attr(condition.rightField || '') + '"></div><button type="button" class="kv2-icon-btn danger" data-action="remove-join-condition" data-index="' + index + '" aria-label="删除关联条件" title="删除关联条件">' + icons.trash + '</button></div>'; }).join('') + '<button type="button" class="kv2-btn kv2-dashed" data-action="add-join-condition">＋ 添加关联条件</button></div>'
        + '<div class="kv2-relationship-sql"><div><strong>JOIN SQL 预览</strong><span>用于确认表关联方向和多条关联条件的组合结果；实际查询 SQL 由系统生成。</span></div><pre data-relationship-preview>' + h(relationshipPreview) + '</pre></div></div>';
    } else if (cat.key === 'standard-qa') {
      fields = '<div class="kv2-form-section kv2-metric-primary">'
        + assetField('assetId', '标准问答标识', f.assetId || '', { required: true, placeholder: 'bpc_revenue_eo', code: true })
        + assetField('description', '标准问答说明', f.description || '', { type: 'textarea', rows: 3, placeholder: '说明该问答对应的业务场景和查询口径' })
        + renderStandardQaUsageMode(f.usageMode)
        + assetField('question', '标准问题', f.question || '', { required: true, type: 'textarea', rows: 2, placeholder: '输入用户最常使用的标准问法' })
        + '<div class="kv2-field"><label>相似问题</label>' + renderEntryEditor('question', f.similarQuestions || []) + '<div class="kv2-control-help">补充同一查询意图的常见表达，用于辅助判断语义一致性；不要添加业务对象、期间或其他关键条件不同的问题。</div></div>'
        + renderSqlEditor(f.sql || '', 7, '输入经过业务确认的完整 SQL') + '</div>';
    } else if (cat.key === 'unstructured-qa') {
      fields = '<div class="kv2-form-section kv2-metric-primary">'
        + assetField('assetId', '标准问答标识', f.assetId || '', { required: true, placeholder: 'bpc_revenue_definition', code: true })
        + assetField('description', '标准问答说明', f.description || '', { type: 'textarea', rows: 3, maxLength: 200, showCounter: true, placeholder: '说明该问答覆盖的知识范围和适用场景' })
        + '<div class="kv2-field"><label class="required">关联文件</label>' + renderSmartSelect({ key: 'qa-related-files', name: 'relatedFiles', target: 'related-files', value: f.relatedFiles || [], multiple: true, options: fileMetadataOptions(), placeholder: '搜索并选择已解析文件', removeAction: 'remove-related-file' }) + '<div class="kv2-control-help">标准答案应来自所选文件；支持搜索和多选。</div></div>'
        + renderStandardQaUsageMode(f.usageMode, true)
        + assetField('question', '标准问题', f.question || '', { required: true, type: 'textarea', rows: 2, placeholder: '输入用户最常使用的标准问法' })
        + '<div class="kv2-field"><label>相似问题</label>' + renderEntryEditor('question', f.similarQuestions || []) + '<div class="kv2-control-help">补充同一问答意图的常见表达；不要添加业务对象或关键条件不同的问题。</div></div>'
        + assetField('answer', '标准答案', f.answer || '', { required: true, type: 'textarea', rows: 6, placeholder: '输入经过业务确认、可直接用于回答用户的标准答案' }) + '</div>';
    } else {
      fields = '<div class="kv2-form-section kv2-metric-primary">'
        + assetField('assetId', '动态查询标识', f.assetId || '', { required: true, placeholder: 'bpc_account_mapping', code: true })
        + assetField('description', '动态查询说明', f.description || '', { type: 'textarea', rows: 3, placeholder: '说明查询目的、返回内容和适用场景' })
        + renderDynamicResultProcessingMode(f.resultProcessingMode)
        + renderSqlEditor(f.sql || '', 8, '输入一条只读的 SELECT 或 WITH 查询') + '</div>';
    }
    const body = '<form id="kv2AssetForm">' + fields + (state.assetError ? '<div class="kv2-form-error" role="alert">' + h(state.assetError) + '</div>' : '') + '</form>';
    const title = cat.key === 'schema' && asset ? '配置表和字段说明 - ' + (f.table || f.assetId || '') : (asset ? '修改' : '新建') + cat.name;
    return modalFrame(title, body, '<button class="kv2-btn" data-action="close-modal">取消</button><button class="kv2-btn primary" data-action="save-asset">' + (asset ? '保存修改' : '创建') + '</button>', 'asset');
  }

  function renderSourceDetail() {
    const source = getSources().find(function (s) { return s.id === state.sourceDetailId; }) || getSources()[0];
    if (source.type === '数据表') {
      const tab = state.sourceDetailTab;
      const tabs = [['columns', '字段 ' + bpcColumns.length], ['statistics', '统计信息'], ['sql', '建表 SQL'], ['sample', '抽样数据 5']].map(function (item) { return '<button type="button" class="kv2-tab ' + (tab === item[0] ? 'active' : '') + '" data-action="source-detail-tab" data-tab="' + item[0] + '">' + item[1] + '</button>'; }).join('');
      const columnRows = bpcColumns.map(function (column) {
        const dataType = column.name === 'b28_s_sdata' ? 'DECIMAL(38, 6)' : column.name === 'account_path' ? 'VARCHAR(1024)' : 'VARCHAR(255)';
        return '<tr><td><code class="kv2-code">' + h(column.name) + '</code></td><td>' + dataType + '</td><td>否</td><td>' + (column.name === 'b28_s_sdata' ? '0' : '—') + '</td><td>' + h(column.description) + '</td></tr>';
      }).join('');
      const columns = '<div class="kv2-table-wrap"><table class="kv2-table kv2-detail-table"><thead><tr><th>列名</th><th>数据类型</th><th>是否主键</th><th>默认值</th><th>描述</th></tr></thead><tbody>' + columnRows + '</tbody></table></div>';
      const statistics = '<div class="kv2-table-wrap"><table class="kv2-table kv2-detail-table"><thead><tr><th>列名</th><th>类型</th><th>非空数</th><th>去重数</th><th>最大值</th><th>最小值</th></tr></thead><tbody><tr><td>b28_s_sdata</td><td>DECIMAL</td><td>2,486,320</td><td>684,293</td><td>98,234,561.35</td><td>-76,812,000.00</td></tr><tr><td>b28_s_kgd353d</td><td>VARCHAR</td><td>2,486,320</td><td>24</td><td>202512</td><td>202401</td></tr><tr><td>b28_s_kgd4rtr_kgdxoi5</td><td>VARCHAR</td><td>2,486,320</td><td>186</td><td>EO_9980</td><td>EO_1000</td></tr><tr><td>b28_s_kgdtvnx</td><td>VARCHAR</td><td>2,486,320</td><td>3</td><td>PLAN_LG</td><td>ACT_LG</td></tr><tr><td>b28_s_kgd4kbn</td><td>VARCHAR</td><td>2,486,320</td><td>5</td><td>USD</td><td>CNY</td></tr></tbody></table></div>';
      const sqlText = 'CREATE TABLE jst_flat_table.bpc_consolidated_report (\n' + bpcColumns.map(function (column) {
        const dataType = column.name === 'b28_s_sdata' ? 'DECIMAL(38, 6)' : column.name === 'account_path' ? 'VARCHAR(1024)' : 'VARCHAR(255)';
        return '  ' + column.name + ' ' + dataType + " COMMENT '" + column.description.replace(/'/g, "''") + "'";
      }).join(',\n') + '\n);';
      const sql = '<pre class="kv2-sql-block">' + h(sqlText) + '</pre>';
      const sampleRows = bpcSampleRows.map(function (row) { return '<tr><td>' + h(row.account) + '</td><td>' + h(row.accountName) + '</td><td>' + h(row.organization) + '</td><td>' + h(row.period) + '</td><td>' + h(row.businessType) + '</td><td>' + h(row.category) + '</td><td>' + h(row.amount) + '</td><td>' + h(row.currency) + '</td></tr>'; }).join('');
      const sample = '<div class="kv2-table-wrap"><table class="kv2-table kv2-detail-table"><thead><tr><th>b28_s_kgd4b76</th><th>b28_s_kgd4b76_txtlg</th><th>b28_s_kgd4rtr_kgdxoi5</th><th>b28_s_kgd353d</th><th>b28_s_kgdbveh</th><th>b28_s_kgdtvnx</th><th>b28_s_sdata</th><th>b28_s_kgd4kbn</th></tr></thead><tbody>' + sampleRows + '</tbody></table></div>';
      const panel = tab === 'statistics' ? statistics : tab === 'sql' ? sql : tab === 'sample' ? sample : columns;
      const body = '<div class="kv2-table-detail-info"><span><b>表名</b>' + h(source.name) + '</span><span><b>行数</b>' + h(source.size) + '</span><span><b>大小</b>' + h(source.storage || '1.86 GB') + '</span><span><b>创建时间</b>2026-07-22 15:52</span><span><b>创建人</b>admin</span><span><b>描述</b>BPC 合并报表明细，用于合并报表指标、期间和组织维度查询</span></div><div class="kv2-inline-tabs">' + tabs + '</div><div class="kv2-table-detail-panel">' + panel + '</div>';
      return modalFrame('表详情 - ' + source.name, body, '<button class="kv2-btn primary" data-action="close-modal">关闭</button>', 'table-detail');
    }
    const preview = '<div class="kv2-document-preview"><div class="kv2-document-page"><h2>' + h(source.name.replace(/\.[^.]+$/, '')) + '</h2><p>MOI 平台提供统一的数据连接、语义建模与智能探索能力，帮助业务人员通过自然语言安全地访问企业数据。</p><p>知识库中的文档完成解析后会生成可治理的分段，并写入文本向量索引。</p></div></div>';
    const info = '<div class="kv2-document-info"><div><span>文件名</span><strong>' + h(source.name) + '</strong></div><div><span>类型</span><strong>文件</strong></div><div><span>大小</span><strong>' + h(source.size) + '</strong></div><div><span>Catalog 路径</span><strong>' + h(source.path.replaceAll(' / ', '/')) + '</strong></div><div><span>处理状态</span><strong>' + h(source.status) + '</strong></div><div><span>更新时间</span><strong>' + h(source.updated) + '</strong></div></div>';
    const body = '<div class="kv2-document-layout"><main class="kv2-document-main"><div class="kv2-inline-tabs"><button type="button" class="kv2-tab ' + (state.documentTab === 'preview' ? 'active' : '') + '" data-action="document-tab" data-tab="preview">预览</button><button type="button" class="kv2-tab ' + (state.documentTab === 'info' ? 'active' : '') + '" data-action="document-tab" data-tab="info">信息</button></div>' + (state.documentTab === 'preview' ? preview : info) + '</main><aside class="kv2-document-segments"><div class="kv2-segment-title">文档分段</div><div class="kv2-segment-tools"><input placeholder="搜索分段"><select><option>原文顺序</option><option>召回优先级</option></select></div><article><div><strong>分段 1</strong><button class="kv2-switch" data-action="toggle-doc-segment"></button></div><p>MOI 平台提供统一的数据连接、语义建模与智能探索能力。</p><button class="kv2-link">编辑</button></article><article><div><strong>分段 2</strong><button class="kv2-switch" data-action="toggle-doc-segment"></button></div><p>知识库中的文档完成解析后会生成可治理的分段。</p><button class="kv2-link">编辑</button></article><div class="kv2-segment-summary">第 1 / 1 页，共 2 条</div></aside></div>';
    return modalFrame(source.name, body, '', 'document');
  }

  function renderSourceExpiry() {
    const source = getSources().find(function (s) { return s.id === state.sourceDetailId; }) || getSources()[0];
    const body = '<section class="kv2-expiry-card"><h4>设置数据源有效期</h4><p>到期后数据源默认不再参与检索，可选择到期后仍强制启用。</p><div class="kv2-field"><label>到期时间</label><input id="kv2ExpiryDate" type="datetime-local" value="' + attr(source.expiresAt || '') + '"></div><div class="kv2-inline-switch"><span>到期后强制启用</span><button type="button" class="kv2-switch ' + (state.expiryForce ? '' : 'off') + '" data-action="toggle-expiry-force"></button></div><div class="kv2-control-help">当前有效期：' + h(source.expiresAt || '未设置') + '</div></section>';
    return modalFrame('有效期 - ' + source.name, body, '<button class="kv2-btn" data-action="close-modal">关闭</button><button class="kv2-btn primary" data-action="save-expiry">保存有效期</button>', 'small');
  }

  function click(action, el, originalTarget) {
    if (action === 'board-tab') { restoreDeletedDemo(); state.boardTab = el.dataset.tab; state.page = 'board'; if (state.boardTab === 'explore') { state.chatReturn = 'board'; if (state.modelId == null || !models.some(function (model) { return model.id === state.modelId; })) state.modelId = models[0] ? models[0].id : null; } render(); return; }
    if (action === 'open-create') { state.createDraft = { name: '', description: '' }; state.modal = 'create'; render(); return; }
    if (action === 'open-model') { const modelId = Number(el.dataset.id); restoreDeletedDemo(modelId); state.modelId = modelId; state.page = 'detail'; state.detailTab = 'source'; render(); return; }
    if (action === 'edit-model') { state.modelId = Number(el.dataset.id); state.modal = 'edit-model'; render(); return; }
    if (action === 'delete-model') { const id = Number(el.dataset.id); const m = models.find(function (x) { return x.id === id; }); if (m) openConfirm({ kind: 'model', id: id, title: '删除知识库', message: '确定删除知识库「' + m.name + '」吗？', description: '删除后，该知识库将从当前列表中移除。' }); return; }
    if (action === 'start-dialog') { state.chatReturn = state.page === 'detail' ? 'detail' : 'board'; state.chatReturnDetailTab = state.detailTab; state.modelId = Number(el.dataset.id); state.page = 'board'; state.boardTab = 'explore'; render(); return; }
    if (action === 'back-from-chat') { if (state.chatReturn === 'detail' && models.some(function (model) { return model.id === state.modelId; })) { restoreDeletedDemo(state.modelId); state.page = 'detail'; state.detailTab = state.chatReturnDetailTab || 'source'; } else { restoreDeletedDemo(); state.page = 'board'; state.boardTab = 'knowledge'; } render(); return; }
    if (action === 'back-board') { state.page = 'board'; state.boardTab = 'knowledge'; state.modal = null; render(); return; }
    if (action === 'detail-tab') { state.detailTab = el.dataset.tab; render(); return; }
    if (action === 'add-source') { state.sourceSelected = []; state.catalogScope = 'tables'; state.catalogQuery = ''; state.catalogPage = 1; state.catalogTreeOpen = { catalog: true, database: true }; state.modal = 'add-source'; render(); return; }
    if (action === 'sem-scope') { const group = semanticGroups.find(function (item) { return item.key === el.dataset.scope; }); if (group && group.categories.length) state.semCategory = group.categories[0]; if (group && group.key === 'structured') state.semanticAdvancedOpen = false; state.semQuery = ''; state.semanticPage = 1; render(); return; }
    if (action === 'toggle-sem-advanced') { const advancedCategoryKeys = ['schema', 'constraint', 'dynamic-query']; if (advancedCategoryKeys.includes(state.semCategory)) { state.semCategory = 'metric'; state.semanticAdvancedOpen = false; state.semQuery = ''; state.semanticPage = 1; } else state.semanticAdvancedOpen = !state.semanticAdvancedOpen; render(); return; }
    if (action === 'sem-category') { state.semCategory = el.dataset.category; if (['schema', 'constraint', 'dynamic-query'].includes(state.semCategory)) state.semanticAdvancedOpen = true; state.semQuery = ''; state.semanticPage = 1; render(); return; }
    if (action === 'semantic-page') { state.semanticPage = Math.max(1, Number(el.dataset.page) || 1); render(); return; }
    if (action === 'open-semantic-import') { state.semanticImport = { status: 'empty' }; state.semanticImportMode = 'append'; state.modal = 'semantic-import'; render(); return; }
    if (action === 'choose-semantic-import') { const input = document.getElementById('kv2SemanticImportFile') || app.querySelector('[data-input="semantic-import-file"]'); if (input) input.click(); return; }
    if (action === 'download-semantic-example') { downloadSemanticImportExample(); return; }
    if (action === 'export-semantics') { exportAllSemantics(); return; }
    if (action === 'confirm-semantic-import') { importAllSemantics(); return; }
    if (action === 'new-asset') { state.editingAsset = null; initAssetDraft(null); state.modal = 'asset'; render(); return; }
    if (action === 'edit-asset') { state.editingAsset = getAssets().find(function (a) { return a.id === Number(el.dataset.id); }) || null; initAssetDraft(state.editingAsset); state.modal = 'asset'; render(); return; }
    if (action === 'delete-asset') { const assets = getAssets(); const a = assets.find(function (x) { return x.id === Number(el.dataset.id); }); if (a) openConfirm({ kind: 'asset', id: a.id, title: '删除语义资产', message: '确定删除「' + a.name + '」吗？', description: '删除后，该配置将不再参与知识库的语义解析。' }); return; }
    if (action === 'toggle-source') { const s = getSources().find(function (x) { return x.id === Number(el.dataset.id); }); if (s) s.enabled = !s.enabled; render(); return; }
    if (action === 'delete-source') { const sources = getSources(); const s = sources.find(function (x) { return x.id === Number(el.dataset.id); }); if (s) openConfirm({ kind: 'source', id: s.id, title: '移除数据源', message: '确定移除数据源「' + s.name + '」吗？', description: '移除后，该数据源将不再参与当前知识库的检索与问答。' }); return; }
    if (action === 'source-detail') { state.sourceDetailId = Number(el.dataset.id); state.sourceDetailTab = 'columns'; state.documentTab = 'preview'; state.modal = 'source-detail'; render(); return; }
    if (action === 'source-detail-tab') { state.sourceDetailTab = el.dataset.tab; render(); return; }
    if (action === 'document-tab') { state.documentTab = el.dataset.tab; render(); return; }
    if (action === 'source-sql') { state.sourceDetailId = Number(el.dataset.id); state.sourceDetailTab = 'sql'; state.modal = 'source-detail'; render(); return; }
    if (action === 'source-expiry') { state.sourceDetailId = Number(el.dataset.id); const expirySource = getSources().find(function (source) { return source.id === state.sourceDetailId; }); state.expiryForce = Boolean(expirySource && expirySource.expiryForce); state.modal = 'source-expiry'; render(); return; }
    if (action === 'toggle-expiry-force') { state.expiryForce = !state.expiryForce; render(); return; }
    if (action === 'save-expiry') { const expirySource = getSources().find(function (source) { return source.id === state.sourceDetailId; }); const expiryInput = document.getElementById('kv2ExpiryDate'); if (expirySource) { expirySource.expiresAt = expiryInput ? expiryInput.value : ''; expirySource.expiryForce = state.expiryForce; } state.modal = null; render(); return; }
    if (action === 'source-download') {
      const downloadSource = getSources().find(function (source) { return source.id === Number(el.dataset.id); });
      if (downloadSource) {
        const blob = new Blob(['MOI 知识库原型下载：' + downloadSource.name], { type: 'text/plain;charset=utf-8' });
        const url = URL.createObjectURL(blob);
        const link = document.createElement('a');
        link.href = url; link.download = downloadSource.name + '.txt'; link.click();
        setTimeout(function () { URL.revokeObjectURL(url); }, 0);
      }
      return;
    }
    if (action === 'toggle-doc-segment') { el.classList.toggle('off'); return; }
    if (action === 'toggle-pipeline') { state.pipelineOpen = !state.pipelineOpen; render(); return; }
    if (action === 'confirm-delete') { executeConfirmedDelete(); return; }
    if (action === 'cancel-confirm') { state.pendingConfirm = null; state.modal = null; render(); return; }
    if (action === 'close-modal') { state.modal = null; state.pendingConfirm = null; state.semanticImport = null; state.editingAsset = null; state.assetDraft = null; state.assetError = ''; state.assetValidationVisible = false; state.semanticPicker = null; state.semanticPickerQuery = ''; render(); return; }
    if (action === 'overlay-close') { if (el === originalTarget) { state.modal = null; state.pendingConfirm = null; state.semanticImport = null; state.editingAsset = null; state.assetDraft = null; state.assetError = ''; state.assetValidationVisible = false; state.semanticPicker = null; state.semanticPickerQuery = ''; render(); } return; }
    if (action === 'open-semantic-picker') {
      if (el.disabled || state.semanticPicker === el.dataset.pickerKey) return;
      syncAssetDraftFromForm();
      state.semanticPicker = el.dataset.pickerKey;
      state.semanticPickerQuery = '';
      render({ preserveModalScroll: true });
      focusAssetInput('kv2SemanticPickerSearch');
      return;
    }
    if (action === 'select-all-semantic-options' || action === 'clear-semantic-options') {
      syncAssetDraftFromForm();
      const target = el.dataset.pickerTarget;
      if (target === 'related-tables') {
        state.assetDraft.relatedTables = action === 'select-all-semantic-options'
          ? tableMetadataOptions().map(function (option) { return option.value; })
          : [];
      }
      state.semanticPicker = el.dataset.pickerKey;
      state.semanticPickerQuery = '';
      state.assetError = '';
      render({ preserveModalScroll: true });
      focusAssetInput('kv2SemanticPickerSearch');
      return;
    }
    if (action === 'select-semantic-option') {
      syncAssetDraftFromForm();
      const value = el.dataset.value || '';
      const target = el.dataset.pickerTarget;
      if (target === 'formula') {
        state.semanticPicker = null;
        state.semanticPickerQuery = '';
        insertFormulaToken(' ' + value + ' ');
        return;
      }
      if (target === 'asset') {
        const prop = el.dataset.pickerProp;
        if (prop === 'sourceTable' && state.semCategory === 'metric' && state.assetDraft.sourceTable !== value) {
          state.assetDraft.aggregationField = '';
          if (state.assetDraft.metricType === 'derived') state.assetDraft.metricFormulaExpression = '';
        }
        if ((prop === 'leftTable' || prop === 'rightTable') && state.assetDraft[prop] !== value) {
          const conditionProp = prop === 'leftTable' ? 'leftField' : 'rightField';
          (state.assetDraft.joinConditions || []).forEach(function (condition) { condition[conditionProp] = ''; });
        }
        state.assetDraft[prop] = value;
      }
      if (target === 'related-files') {
        const values = state.assetDraft.relatedFiles || [];
        if (values.includes(value)) state.assetDraft.relatedFiles = values.filter(function (item) { return item !== value; });
        else state.assetDraft.relatedFiles = values.concat(value);
      }
      if (target === 'related-tables') {
        const values = state.assetDraft.relatedTables || [];
        if (values.includes(value)) state.assetDraft.relatedTables = values.filter(function (item) { return item !== value; });
        else state.assetDraft.relatedTables = values.concat(value);
      }
      if (target === 'filter-field') {
        const node = filterNodeAt(el.dataset.pickerPath);
        if (node) node.field = value;
      }
      if (target === 'filter-comparison') {
        const node = filterNodeAt(el.dataset.pickerPath);
        if (node) node.comparison = value;
      }
      if (target === 'join-field') {
        const condition = (state.assetDraft.joinConditions || [])[Number(el.dataset.pickerIndex)];
        if (condition) condition[el.dataset.pickerProp] = value;
      }
      if (target === 'join-comparison') {
        const condition = (state.assetDraft.joinConditions || [])[Number(el.dataset.pickerIndex)];
        if (condition) condition.comparison = value;
      }
      const keepOpen = el.dataset.pickerMultiple === 'true';
      state.semanticPicker = keepOpen ? el.dataset.pickerKey : null;
      state.semanticPickerQuery = '';
      state.assetError = '';
      render({ preserveModalScroll: true });
      if (keepOpen) focusAssetInput('kv2SemanticPickerSearch');
      return;
    }
    if (action === 'add-synonym') { addEntry('synonym'); return; }
    if (action === 'remove-synonym') { syncAssetDraftFromForm(); state.assetDraft.synonyms.splice(Number(el.dataset.index), 1); state.assetError = ''; render({ preserveModalScroll: true }); return; }
    if (action === 'add-similar-question') { addEntry('question'); return; }
    if (action === 'remove-similar-question') { syncAssetDraftFromForm(); state.assetDraft.similarQuestions.splice(Number(el.dataset.index), 1); state.assetError = ''; render({ preserveModalScroll: true }); return; }
    if (action === 'remove-related-table') { syncAssetDraftFromForm(); state.assetDraft.relatedTables.splice(Number(el.dataset.index), 1); render({ preserveModalScroll: true }); return; }
    if (action === 'remove-related-file') { syncAssetDraftFromForm(); state.assetDraft.relatedFiles.splice(Number(el.dataset.index), 1); render({ preserveModalScroll: true }); return; }
    if (action === 'clear-formula') { syncAssetDraftFromForm(); state.assetDraft.metricFormulaExpression = ''; render({ preserveModalScroll: true }); return; }
    if (action === 'insert-formula') { insertFormulaToken(el.dataset.token || ''); return; }
    if (action === 'add-filter-condition' || action === 'add-filter-group') {
      syncAssetDraftFromForm();
      const group = filterNodeAt(el.dataset.path || '');
      if (group && group.conditions) {
        if (action === 'add-filter-condition') group.conditions.push({ type: 'condition', connector: group.conditions.length ? 'and' : undefined, field: '', comparison: 'equals', value: '' });
        else group.conditions.push({ type: 'group', connector: group.conditions.length ? 'and' : undefined, logic: 'and', conditions: [{ type: 'condition', field: '', comparison: 'equals', value: '' }] });
      }
      render({ preserveModalScroll: true }); return;
    }
    if (action === 'remove-filter-node') { syncAssetDraftFromForm(); removeFilterNode(el.dataset.path); render({ preserveModalScroll: true }); return; }
    if (action === 'remove-filter-value') { syncAssetDraftFromForm(); const filterNode = filterNodeAt(el.dataset.path); if (filterNode) { const values = listValue(filterNode.value); values.splice(Number(el.dataset.index), 1); filterNode.value = values; } render({ preserveModalScroll: true }); return; }
    if (action === 'toggle-schema-column') { syncAssetDraftFromForm(); const column = state.assetDraft.schemaColumns[Number(el.dataset.index)]; if (column) column.enabled = !column.enabled; render({ preserveModalScroll: true }); return; }
    if (action === 'add-join-condition') { syncAssetDraftFromForm(); state.assetDraft.joinConditions.push({ leftField: '', comparison: 'equals', rightField: '' }); render({ preserveModalScroll: true }); return; }
    if (action === 'remove-join-condition') { syncAssetDraftFromForm(); state.assetDraft.joinConditions.splice(Number(el.dataset.index), 1); render({ preserveModalScroll: true }); return; }
    if (action === 'create-model') { createModel(); return; }
    if (action === 'toggle-catalog-tree') { const node = el.dataset.node; state.catalogTreeOpen[node] = !state.catalogTreeOpen[node]; render(); return; }
    if (action === 'catalog-scope') { state.catalogScope = el.dataset.scope; state.catalogQuery = ''; state.catalogPage = 1; render(); return; }
    if (action === 'catalog-page') { state.catalogPage = Math.max(1, Number(el.dataset.page) || 1); render(); return; }
    if (action === 'save-sources') { saveSources(); return; }
    if (action === 'save-model-meta') { saveModelMeta(); return; }
    if (action === 'save-asset') { saveAsset(); return; }
    if (action === 'send-chat') { sendChat(); return; }
    if (action === 'new-chat') { state.messages = [{ role: 'bot', text: '已创建 BPC 新对话。你可以输入组织、期间和指标名称，例如“EO_1000 公司 2025 年 5 月营业收入是多少？”' }]; render(); }
  }

  function createModel() {
    const form = document.getElementById('kv2CreateBase');
    if (!form || !form.reportValidity()) return;
    const data = Object.fromEntries(new FormData(form).entries());
    const model = { id: nextModelId++, name: String(data.name || '').trim(), desc: String(data.description || '').trim(), files: 0, tables: 0 };
    models.unshift(model);
    sourceStore[model.id] = [];
    assetStore[model.id] = [];
    state.createDraft = { name: '', description: '' };
    state.modelId = model.id;
    state.page = 'detail';
    state.detailTab = 'source';
    state.modal = null;
    render();
  }

  function saveSources() {
    if (!state.sourceSelected.length) return;
    const picked = state.sourceSelected.map(function (id) { return catalogSources.find(function (s) { return s.id === id; }); }).filter(Boolean);
    picked.forEach(function (item) {
      if (item && !getSources().some(function (s) { return s.name === item.name; })) getSources().push({ id: nextSourceId++, name: item.name, type: item.type, size: item.size, path: item.path, comment: item.comment || '', status: '处理中', updated: nowText(), enabled: true });
    });
    currentModel().files = getSources().filter(function (s) { return s.type === '文件'; }).length;
    currentModel().tables = getSources().filter(function (s) { return s.type === '数据表'; }).length;
    state.modal = null;
    state.sourceSelected = [];
    render();
  }

  function saveModelMeta() {
    const form = document.getElementById('kv2EditModel');
    if (!form || !form.reportValidity()) return;
    const data = Object.fromEntries(new FormData(form).entries());
    currentModel().name = data.name;
    currentModel().desc = data.description;
    state.modal = null;
    render();
  }

  function saveAsset() {
    const form = document.getElementById('kv2AssetForm');
    if (!form) return;
    syncAssetDraftFromForm();
    state.assetValidationVisible = true;
    // 使用 checkValidity 而非 reportValidity：先稳定渲染字段错误态，避免只触发浏览器原生提示而没有红框反馈。
    if (!form.checkValidity()) { render({ preserveModalScroll: true }); focusFirstAssetError(); return; }
    const f = state.assetDraft;
    state.assetError = '';
    if (!f.assetId || !/^[a-z_-]+$/.test(f.assetId)) state.assetError = '标识仅支持英文小写字母、下划线（_）和横线（-）。';
    else if (f.assetId.length > 50) state.assetError = '标识长度不能超过 50 个字符。';
    if (state.semCategory === 'metric') {
      if (!f.name) state.assetError = '请输入名称';
      if (f.metricType === 'derived') {
        if (!f.sourceTable) state.assetError = '请先选择来源表';
        else state.assetError = formulaValidationMessage(f.metricFormulaExpression);
      }
      if (f.metricType === 'aggregate' && (!f.sourceTable || !f.aggregationField)) state.assetError = '请选择来源表和聚合字段';
      if (f.metricType === 'aggregate' && f.filterGroup && hasInvalidFilter(f.filterGroup, true)) state.assetError = '请完整填写已添加的筛选条件，或删除未完成的条件组。';
      f.synonyms = (f.synonyms || []).slice();
      f.aggregation = String(f.aggregationFunction || 'sum').toUpperCase();
      f.field = f.aggregationField;
      f.formula = f.metricFormulaExpression || '';
    }
    if (state.semCategory === 'business-rule') {
      if (!(f.relatedTables || []).length) state.assetError = '请至少关联一张结构化数据表';
      else if (!String(f.ruleContent || '').trim()) state.assetError = '请输入规则内容';
    }
    if (state.semCategory === 'unstructured-rule') {
      if (!(f.relatedFiles || []).length) state.assetError = '请至少关联一个已完成解析与嵌入的文件';
      else if (!String(f.ruleContent || '').trim()) state.assetError = '请输入规则内容';
    }
    if (state.semCategory === 'unstructured-qa') {
      if (String(f.description || '').length > 200) state.assetError = '标准问答说明不能超过 200 个字符';
      else if (!(f.relatedFiles || []).length) state.assetError = '请至少关联一个已完成解析与嵌入的文件';
      else if (!['reference', 'force'].includes(f.usageMode)) state.assetError = '请选择标准问答的使用方式';
      else if (!String(f.question || '').trim() || !String(f.answer || '').trim()) state.assetError = '请填写标准问题和标准答案';
    }
    if (state.semCategory === 'constraint') {
      if (!f.sourceTable) state.assetError = '请选择关联表';
      else if (!f.filterGroup || !hasMeaningfulFilter(f.filterGroup)) state.assetError = '请至少配置一个约束条件';
      else if (hasInvalidFilter(f.filterGroup, true)) state.assetError = '请完整填写已添加的筛选条件，或删除未完成的条件组。';
    }
    if (state.semCategory === 'relationship') {
      if (String(f.description || '').length > 200) state.assetError = '表关联说明不能超过 200 个字符';
      else if (!f.leftTable || !f.rightTable) state.assetError = '请选择左表和右表';
      else if (!(f.joinConditions || []).length || f.joinConditions.some(function (condition) { return !condition.leftField || !condition.rightField; })) state.assetError = '请完整填写至少一条关联条件';
    }
    if (state.semCategory === 'standard-qa') {
      if (!['reference', 'force'].includes(f.usageMode)) state.assetError = '请选择标准问答的使用方式';
      else if (!String(f.question || '').trim() || !String(f.sql || '').trim()) state.assetError = '请填写标准问题和 SQL';
    }
    if (state.semCategory === 'dynamic-query' && !['direct_context', 'delegated_analysis'].includes(f.resultProcessingMode)) state.assetError = '请选择动态查询结果的处理方式';
    if (state.semCategory === 'standard-qa' || state.semCategory === 'dynamic-query') {
      const sqlValidation = validateSqlSyntax(f.sql);
      if (!sqlValidation.valid) state.assetError = sqlValidation.message;
    }
    if (state.assetError) { render({ preserveModalScroll: true }); focusFirstAssetError(); return; }
    if (state.semCategory === 'schema') f.description = f.description || '';
    if (state.semCategory === 'business-rule' || state.semCategory === 'unstructured-rule') { f.content = f.ruleContent; f.description = f.ruleContent; }
    if (state.semCategory === 'constraint') {
      const firstCondition = f.filterGroup.conditions.find(function (node) { return node.type === 'condition'; });
      if (firstCondition) { f.field = firstCondition.field; f.operator = firstCondition.comparison; f.value = firstCondition.value; }
    }
    if (state.semCategory === 'relationship' && f.joinConditions.length) { f.leftField = f.joinConditions[0].leftField; f.rightField = f.joinConditions[0].rightField; }
    f.name = f.name || (state.editingAsset && state.editingAsset.name) || f.assetId;
    let binding = f.assetId || f.name;
    if (state.semCategory === 'metric') binding = f.sourceTable || '—';
    if (state.semCategory === 'schema') binding = f.table || f.assetId;
    if (state.semCategory === 'business-rule') binding = (f.relatedTables || []).join(', ') || f.assetId;
    if (state.semCategory === 'unstructured-rule') binding = (f.relatedFiles || []).join(', ') || f.assetId;
    if (state.semCategory === 'unstructured-qa') binding = (f.relatedFiles || []).join(', ') || f.assetId;
    if (state.semCategory === 'constraint') binding = f.sourceTable || f.assetId;
    if (state.semCategory === 'relationship') binding = (f.leftTable || '') + '.' + (f.leftField || '') + ' ' + relationshipComparisonSymbol(f.joinConditions[0] && f.joinConditions[0].comparison) + ' ' + (f.rightTable || '') + '.' + (f.rightField || '');
    if (state.semCategory === 'standard-qa') binding = f.usageMode === 'force' ? '高置信度语义一致时强制使用' : '仅供参考';
    if (state.semCategory === 'dynamic-query') binding = f.resultProcessingMode === 'delegated_analysis' ? '子 Agent 分析' : '直接注入上下文';
    if (state.editingAsset) Object.assign(state.editingAsset, { name: f.name, binding: binding, desc: f.description || f.content || '', updated: nowText(), fields: f });
    else getAssets().unshift({ id: nextAssetId++, category: state.semCategory, name: f.name, binding: binding, desc: f.description || f.content || '', updated: nowText(), fields: f });
    state.modal = null;
    state.editingAsset = null;
    state.assetDraft = null;
    render();
  }

  function sendChat() {
    const input = document.getElementById('kv2ChatInput');
    if (!input || !input.value.trim()) return;
    const q = input.value.trim();
    state.messages.push({ role: 'user', text: q });
    state.messages.push({ role: 'bot', pipeline: true, text: '已基于「' + currentModel().name + '」完成语义规划、指标编译、强制约束注入和标准问答校验。\n\n查询范围：EO_1000｜2025 年 5 月｜实际数 ACT_LG｜人民币 CNY\n营业收入：12,568,000.00 元\n主营业务成本：8,676,000.00 元\n销售毛利：3,892,000.00 元\n销售毛利率：30.97%\n\n营业收入按 APL6000 取数并置反；主营业务成本按 APL6600、APL6401、APL5001 汇总。' });
    render();
  }

  app.addEventListener('mouseover', function (e) {
    const clipped = e.target.closest('.kv2-code-tag, .kv2-muted, .kv2-single-line');
    if (!clipped || clipped === clippedContentAnchor) return;
    if (clippedContentShowTimer) window.clearTimeout(clippedContentShowTimer);
    clippedContentShowTimer = window.setTimeout(function () {
      clippedContentShowTimer = null;
      showClippedContentTooltip(clipped);
    }, 140);
  });

  app.addEventListener('mouseout', function (e) {
    const clipped = e.target.closest('.kv2-code-tag, .kv2-muted, .kv2-single-line');
    if (clippedContentShowTimer && clipped && (!e.relatedTarget || !clipped.contains(e.relatedTarget))) {
      window.clearTimeout(clippedContentShowTimer);
      clippedContentShowTimer = null;
    }
    if (!clippedContentAnchor || !clippedContentAnchor.contains(e.target)) return;
    if (e.relatedTarget && clippedContentAnchor.contains(e.relatedTarget)) return;
    hideClippedContentTooltip();
  });

  app.addEventListener('click', function (e) {
    const target = e.target.closest('[data-action]');
    const clickedInsidePicker = Boolean(e.target.closest('.kv2-smart-select'));
    if (state.semanticPicker && !clickedInsidePicker) {
      state.semanticPicker = null;
      state.semanticPickerQuery = '';
      const clickedModalBlank = target && target.dataset.action === 'overlay-close' && e.target.closest('[data-modal-stop]');
      if (!target || clickedModalBlank) {
        const clickedControl = e.target.closest('input, textarea, select, button');
        const controlId = clickedControl && clickedControl.id;
        const controlName = clickedControl && clickedControl.name;
        const selectionStart = clickedControl && clickedControl.selectionStart;
        const selectionEnd = clickedControl && clickedControl.selectionEnd;
        render({ preserveModalScroll: true });
        if (controlId || controlName) requestAnimationFrame(function () {
          const nextControl = controlId
            ? document.getElementById(controlId)
            : Array.from(app.querySelectorAll('[name]')).find(function (control) { return control.name === controlName; });
          if (!nextControl) return;
          nextControl.focus({ preventScroll: true });
          if (nextControl.setSelectionRange && selectionStart != null) nextControl.setSelectionRange(selectionStart, selectionEnd);
        });
        return;
      }
    }
    if (!target) {
      return;
    }
    if (target.dataset.action === 'overlay-close' && e.target.closest('[data-modal-stop]')) return;
    e.stopPropagation();
    click(target.dataset.action, target, e.target);
  });
  app.addEventListener('input', function (e) {
    const type = e.target.dataset.input;
    if (type === 'asset-key') {
      if (state.assetDraft) state.assetDraft.assetId = e.target.value;
      const field = e.target.closest('.kv2-field');
      const counter = field ? field.querySelector('[data-key-counter]') : null;
      if (counter) counter.textContent = e.target.value.length + '/50';
    }
    if (type === 'counted-field') {
      const field = e.target.closest('.kv2-field');
      const counter = field ? field.querySelector('[data-field-counter]') : null;
      if (counter) counter.textContent = e.target.value.length + '/' + e.target.maxLength;
    }
    if (type === 'board-search') { state.query = e.target.value; render(); const input = app.querySelector('[data-input="board-search"]'); if (input) { input.focus(); input.setSelectionRange(state.query.length, state.query.length); } }
    if (type === 'sem-search') { state.semQuery = e.target.value; state.semanticPage = 1; render(); const input = app.querySelector('[data-input="sem-search"]'); if (input) { input.focus(); input.setSelectionRange(state.semQuery.length, state.semQuery.length); } }
    if (type === 'catalog-search') { state.catalogQuery = e.target.value; state.catalogPage = 1; render(); const input = app.querySelector('[data-input="catalog-search"]'); if (input) { input.focus(); input.setSelectionRange(state.catalogQuery.length, state.catalogQuery.length); } }
    if (type === 'schema-column-search') { state.schemaColumnSearch = e.target.value; render(); const input = app.querySelector('[data-input="schema-column-search"]'); if (input) { input.focus(); input.setSelectionRange(state.schemaColumnSearch.length, state.schemaColumnSearch.length); } }
    if (type === 'semantic-picker-search') {
      state.semanticPicker = e.target.dataset.pickerKey;
      state.semanticPickerQuery = e.target.value;
      render({ preserveModalScroll: true });
      const input = document.getElementById('kv2SemanticPickerSearch');
      if (input) { input.focus(); input.setSelectionRange(state.semanticPickerQuery.length, state.semanticPickerQuery.length); }
    }
    if (type === 'sql-editor' && state.assetDraft) {
      state.assetDraft.sql = e.target.value;
      state.assetError = '';
      const result = validateSqlSyntax(e.target.value);
      const editor = e.target.closest('.kv2-sql-editor');
      const status = editor && editor.querySelector('[data-sql-validation]');
      const formError = app.querySelector('.kv2-form-error');
      if (editor) {
        editor.classList.toggle('valid', result.valid);
        editor.classList.toggle('invalid', !result.valid);
      }
      e.target.setAttribute('aria-invalid', String(!result.valid));
      if (status) {
        status.className = 'kv2-sql-validation ' + (result.valid ? 'success' : 'error');
        status.textContent = (result.valid ? '✓ ' : '！ ') + result.message;
      }
      if (formError) formError.remove();
    }
    if (e.target.name === 'metricFormulaExpression' && state.assetDraft) {
      state.assetDraft.metricFormulaExpression = e.target.value;
      const errorMessage = e.target.value ? formulaValidationMessage(e.target.value) : '';
      const preview = app.querySelector('[data-formula-preview]');
      const error = app.querySelector('[data-formula-error]');
      const counter = app.querySelector('[data-formula-counter]');
      const inputRow = e.target.closest('.kv2-formula-input');
      if (preview) preview.textContent = e.target.value && !errorMessage ? formulaPreviewText(e.target.value) : '公式未完成';
      if (error) error.textContent = errorMessage;
      if (counter) counter.textContent = e.target.value.length + '/200';
      if (inputRow) inputRow.classList.toggle('error', Boolean(errorMessage));
      e.target.setAttribute('aria-invalid', String(Boolean(errorMessage)));
    }
    if (e.target.dataset.filterPath != null && state.assetDraft) {
      const node = filterNodeAt(e.target.dataset.filterPath);
      if (node) node[e.target.dataset.filterProp] = e.target.value;
      refreshFilterConditionValidation(e.target.dataset.filterPath);
      const preview = app.querySelector('[data-filter-preview]');
      if (preview) {
        const keyword = state.semCategory === 'constraint' ? 'WHERE' : 'WHEN';
        const sql = filterSql(state.assetDraft.filterGroup);
        preview.textContent = sql ? keyword + ' ' + sql : keyword + ' (\n  <condition>\n)';
      }
    }
    if (e.target.dataset.schemaIndex != null && state.assetDraft) {
      const column = state.assetDraft.schemaColumns[Number(e.target.dataset.schemaIndex)];
      if (column) column[e.target.dataset.schemaProp] = e.target.value;
    }
  });
  app.addEventListener('change', function (e) {
    if (e.target.dataset.input === 'semantic-import-file') {
      const file = e.target.files && e.target.files[0];
      e.target.value = '';
      loadSemanticImportFile(file);
      return;
    }
    if (e.target.name === 'semanticImportMode') { state.semanticImportMode = e.target.value; return; }
    if (e.target.dataset.input === 'source-check') {
      if (e.target.checked && !state.sourceSelected.includes(e.target.value)) state.sourceSelected.push(e.target.value);
      if (!e.target.checked) state.sourceSelected = state.sourceSelected.filter(function (id) { return id !== e.target.value; });
      render();
    }
    if (e.target.dataset.input === 'source-select-all') {
      const tableScope = state.catalogScope === 'tables';
      const query = state.catalogQuery.trim().toLowerCase();
      const selectableIds = catalogSources.filter(function (source) {
        return (tableScope ? source.type === '数据表' : source.type === '文件')
          && (!query || source.name.toLowerCase().includes(query))
          && !getSources().some(function (current) { return current.name === source.name; });
      }).map(function (source) { return source.id; });
      if (e.target.checked) selectableIds.forEach(function (id) { if (!state.sourceSelected.includes(id)) state.sourceSelected.push(id); });
      else state.sourceSelected = state.sourceSelected.filter(function (id) { return !selectableIds.includes(id); });
      render();
    }
    if (e.target.dataset.input === 'chat-model') { state.modelId = Number(e.target.value); render(); }
    if (e.target.dataset.input === 'metric-type') {
      syncAssetDraftFromForm();
      const previousType = state.assetDraft.metricType;
      state.assetDraft.metricType = e.target.value;
      if (e.target.value === 'derived' && previousType !== 'derived') {
        state.assetDraft.sourceTable = '';
        state.assetDraft.metricFormulaExpression = '';
      }
      state.assetError = '';
      render();
    }
    if (state.semCategory === 'relationship' && (e.target.name === 'joinType' || /^comparison_\d+$/.test(e.target.name || ''))) {
      syncAssetDraftFromForm();
      const preview = app.querySelector('[data-relationship-preview]');
      if (preview) preview.textContent = relationshipSqlPreview(state.assetDraft);
    }
    if (e.target.dataset.filterPath != null && state.assetDraft) {
      syncAssetDraftFromForm();
      const node = filterNodeAt(e.target.dataset.filterPath);
      if (node) node[e.target.dataset.filterProp] = e.target.value;
      render({ preserveModalScroll: true });
    }
  });
  app.addEventListener('keydown', function (e) {
    if (e.target.id === 'kv2ChatInput' && e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); sendChat(); }
    if (e.target.id === 'kv2SynonymInput' && e.key === 'Enter') { e.preventDefault(); addEntry('synonym'); }
    if (e.target.id === 'kv2SimilarQuestionInput' && e.key === 'Enter') { e.preventDefault(); addEntry('question'); }
    if (e.target.dataset.filterListPath != null && e.key === 'Enter') { e.preventDefault(); addFilterListValue(e.target); }
    if (e.target.dataset.input === 'semantic-picker-search' && e.key === 'Enter') {
      const firstOption = app.querySelector('.kv2-smart-select.open .kv2-smart-select-option');
      if (firstOption) { e.preventDefault(); click('select-semantic-option', firstOption, firstOption); }
    }
    if ((e.target.classList.contains('kv2-card')) && (e.key === 'Enter' || e.key === ' ')) { e.preventDefault(); click(e.target.dataset.action, e.target, e.target); }
    if (e.key === 'Escape' && state.semanticPicker) { state.semanticPicker = null; state.semanticPickerQuery = ''; render({ preserveModalScroll: true }); return; }
    if (e.key === 'Escape' && state.modal) { state.modal = null; state.pendingConfirm = null; state.editingAsset = null; state.assetDraft = null; render(); }
  });
  app.addEventListener('scroll', function () {
    hideClippedContentTooltip();
    if (state.semanticPicker) requestAnimationFrame(positionSemanticPickerDropdown);
  }, true);
  window.addEventListener('resize', hideClippedContentTooltip);

  state.modelId = models[0].id;
  render();
})();
