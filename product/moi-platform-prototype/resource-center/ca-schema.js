// MOI 通用行业知识库的数据源结构。仅展示结构，不包含真实业务数据。
window.CA_DB_SCHEMA = {
  db: 'moi_demo_knowledge',
  dir: '示例行业知识库',
  tables: [
    {
      name: 'knowledge_documents',
      note: '通用业务文档与摘要',
      rows: '12,480 行',
      cols: [
        { n: 'document_id', t: 'varchar(64)', pk: true, c: '文档唯一标识' },
        { n: 'title', t: 'varchar(255)', pk: false, c: '文档标题' },
        { n: 'domain', t: 'varchar(64)', pk: false, c: '制造、零售或企业协作' },
        { n: 'content', t: 'text', pk: false, c: '脱敏后的正文内容' },
        { n: 'updated_at', t: 'timestamp', pk: false, c: '更新时间' }
      ]
    },
    {
      name: 'knowledge_chunks',
      note: '文档切片与向量索引',
      rows: '48,960 行',
      cols: [
        { n: 'chunk_id', t: 'varchar(64)', pk: true, c: '切片唯一标识' },
        { n: 'document_id', t: 'varchar(64)', pk: false, c: '所属文档' },
        { n: 'chunk_content', t: 'text', pk: false, c: '文本切片' },
        { n: 'embedding', t: 'vecf32(1024)', pk: false, c: '语义向量' },
        { n: 'access_scope', t: 'varchar(128)', pk: false, c: '访问范围' }
      ]
    },
    {
      name: 'business_metrics',
      note: '通用经营指标口径',
      rows: '8,760 行',
      cols: [
        { n: 'metric_id', t: 'varchar(64)', pk: true, c: '指标标识' },
        { n: 'metric_name', t: 'varchar(128)', pk: false, c: '指标名称' },
        { n: 'definition', t: 'text', pk: false, c: '计算口径' },
        { n: 'domain', t: 'varchar(64)', pk: false, c: '业务域' },
        { n: 'owner_role', t: 'varchar(64)', pk: false, c: '责任角色' }
      ]
    },
    {
      name: 'policy_rules',
      note: '通用权限与合规规则',
      rows: '320 行',
      cols: [
        { n: 'rule_id', t: 'varchar(64)', pk: true, c: '规则标识' },
        { n: 'rule_name', t: 'varchar(128)', pk: false, c: '规则名称' },
        { n: 'rule_type', t: 'varchar(64)', pk: false, c: '权限、脱敏或导出控制' },
        { n: 'rule_content', t: 'text', pk: false, c: '规则说明' },
        { n: 'enabled', t: 'boolean', pk: false, c: '是否启用' }
      ]
    }
  ]
};
