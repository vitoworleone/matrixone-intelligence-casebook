// 数据载入任务的共享数据源 — 唯一真相
// data-import.html 用 IMPORT_TASKS_DISPLAY 渲染列表 + 详情（含 schema / files）
// data-import-create.js 用 IMPORT_TASKS_EDIT 填写编辑表单
// 同时 IMPORT_TASKS_DISPLAY[id].schema 也供编辑页渲染"目标表 schema"
(function(){
window.IMPORT_TASKS_DISPLAY = {
      // ====================================================================
      // 【NESR-湖仓项目】实际生产载入任务（7 个：1 Intelie MongoDB + 6 Fiix CMMS REST API）
      // 传感器在载入任务里配 MongoDB 源端聚合（秒→分钟，不同步明细）；供工作流 wf7a（全载入版）使用
      // 目标：客户的 nesr_raw 数据库（在 MOI 中对应 NESR / Bronze 层）
      // ====================================================================
      't_nesr_intelie_sensor': { name: '【NESR-湖仓项目】Intelie 传感器数据同步', connector: 'Intelie 现场传感器 · MongoDB', connType: 'MongoDB（数据库）', dataType: '结构化', mode: '周期性', target: 'NESR / Bronze / intelie_sensor_readings', status: '载入中', created: '2026-04-10 09:30', lastRun: '2026-04-28 14:32', schedule: '每 5 分钟', totalRuns: 4896, totalRows: '36,000', totalSize: '7.2 MB', syncStrategy: 'incremental', incremental: { field: 'datetime', lookback: '10 分钟' }, backfill: { enabled: true, progress: '100%（已完成）', startDate: '2026-03-01', batchSize: '1 天', batchInterval: '5 秒', window: '不限制' }, extra: [
        { label: '源集合', value: 'intelie_prod.fleet_sensor_1s' },
        { label: '源端聚合', value: 'MongoDB Aggregation Pipeline：$group by 分钟桶（秒级→分钟级，仅传结果）' },
        { label: '字段数', value: '12 个' }
      ]},
      't_nesr_fiix_work_orders': { name: '【NESR-湖仓项目】Fiix 工单数据同步', connector: 'Fiix CMMS 维修系统 · REST API', connType: 'REST API', dataType: '结构化', mode: '周期性', target: 'NESR / Bronze / fiix_work_orders', status: '等待中', created: '2026-04-11 04:15', lastRun: '2026-04-28 14:15', schedule: '每小时', totalRuns: 408, totalRows: '280', totalSize: '92 KB', syncStrategy: 'incremental', incremental: { field: 'dtm_date_last_modified', lookback: '1 小时' }, backfill: { enabled: true, progress: '100%（已完成）', startDate: '2024-01-01', batchSize: '1 月', batchInterval: '5 秒', window: '不限制' }, schemaMeta: '源 Fiix API 驼峰字段 → Bronze 下划线列', schema: [
        { srcName: 'id', srcType: 'Number', name: 'work_order_id', type: 'BIGINT', pk: true, nullable: false, desc: '工单 ID' },
        { srcName: 'strCode', srcType: 'String', name: 'code', type: 'VARCHAR(64)', desc: '工单编号' },
        { srcName: 'strDescription', srcType: 'String', name: 'description', type: 'TEXT', desc: '工单描述' },
        { srcName: 'intAssetID', srcType: 'Number', name: 'asset_id', type: 'BIGINT', desc: '关联资产 ID' },
        { srcName: 'intPriorityID', srcType: 'Number', name: 'priority_id', type: 'INT', desc: '优先级 ID' },
        { srcName: 'intWorkOrderStatusID', srcType: 'Number', name: 'status_id', type: 'INT', desc: '状态 ID' },
        { srcName: 'intMaintenanceTypeID', srcType: 'Number', name: 'maintenance_type_id', type: 'INT', desc: '维修类型 ID' },
        { srcName: 'intAssignedToUserID', srcType: 'Number', name: 'assigned_user_id', type: 'BIGINT', desc: '指派人 ID' },
        { srcName: 'dtmDateCreated', srcType: 'DateTime', name: 'created_at', type: 'DATETIME', desc: '创建时间' },
        { srcName: 'dtmDateLastModified', srcType: 'DateTime', name: 'dtm_date_last_modified', type: 'DATETIME', nullable: false, desc: '最后修改时间（增量字段）' }
      ], extra: [
        { label: 'Endpoint', value: '/api/v5/WorkOrder' },
        { label: '字段数', value: '10 个' }
      ]},
      't_nesr_fiix_wo_tasks': { name: '【NESR-湖仓项目】Fiix 工单任务同步', connector: 'Fiix CMMS 维修系统 · REST API', connType: 'REST API', dataType: '结构化', mode: '周期性', target: 'NESR / Bronze / fiix_wo_tasks', status: '等待中', created: '2026-04-11 04:15', lastRun: '2026-04-28 14:10', schedule: '每小时', totalRuns: 408, totalRows: '1,450', totalSize: '420 KB', syncStrategy: 'incremental', incremental: { field: 'dtm_date_last_modified', lookback: '1 小时' }, backfill: { enabled: true, progress: '100%（已完成）', startDate: '2024-01-01', batchSize: '1 月', batchInterval: '5 秒', window: '不限制' }, schemaMeta: '源 Fiix API 驼峰字段 → Bronze 下划线列', schema: [
        { srcName: 'id', srcType: 'Number', name: 'task_id', type: 'BIGINT', pk: true, nullable: false, desc: '任务 ID' },
        { srcName: 'intWorkOrderID', srcType: 'Number', name: 'work_order_id', type: 'BIGINT', nullable: false, desc: '所属工单 ID' },
        { srcName: 'strDescription', srcType: 'String', name: 'description', type: 'TEXT', desc: '任务描述' },
        { srcName: 'intCompleted', srcType: 'Number', name: 'is_completed', type: 'TINYINT', desc: '是否完成（0/1）' },
        { srcName: 'dtmDateCompleted', srcType: 'DateTime', name: 'completed_at', type: 'DATETIME', desc: '完成时间' },
        { srcName: 'dtmDateLastModified', srcType: 'DateTime', name: 'dtm_date_last_modified', type: 'DATETIME', nullable: false, desc: '最后修改时间（增量字段）' }
      ], extra: [
        { label: 'Endpoint', value: '/api/v5/WorkOrderTask' },
        { label: '字段数', value: '6 个' }
      ]},
      't_nesr_fiix_assets': { name: '【NESR-湖仓项目】Fiix 资产层级同步', connector: 'Fiix CMMS 维修系统 · REST API', connType: 'REST API', dataType: '结构化', mode: '周期性', target: 'NESR / Bronze / fiix_assets', status: '等待中', created: '2026-04-11 04:15', lastRun: '2026-04-28 04:00', schedule: '每天 04:00', totalRuns: 18, totalRows: '24', totalSize: '8 KB', syncStrategy: 'full', schemaMeta: '源 Fiix API 驼峰字段 → Bronze 下划线列', schema: [
        { srcName: 'id', srcType: 'Number', name: 'asset_id', type: 'BIGINT', pk: true, nullable: false, desc: '资产 ID' },
        { srcName: 'strName', srcType: 'String', name: 'asset_name', type: 'VARCHAR(255)', desc: '资产名称' },
        { srcName: 'intParentID', srcType: 'Number', name: 'parent_id', type: 'BIGINT', desc: '父资产 ID（树结构）' },
        { srcName: 'strCategory', srcType: 'String', name: 'category', type: 'VARCHAR(128)', desc: '资产类别（含 HP Pump）' },
        { srcName: 'intSiteID', srcType: 'Number', name: 'site_id', type: 'BIGINT', desc: '站点 ID' }
      ], extra: [
        { label: 'Endpoint', value: '/api/v5/Asset' },
        { label: '字段数', value: '5 个' },
        { label: '说明', value: '父子树结构，HP Pump 关键类别用递归 CTE 展开' }
      ]},
      't_nesr_fiix_priorities': { name: '【NESR-湖仓项目】Fiix 优先级字典同步', connector: 'Fiix CMMS 维修系统 · REST API', connType: 'REST API', dataType: '结构化', mode: '周期性', target: 'NESR / Bronze / fiix_priorities', status: '等待中', created: '2026-04-11 04:15', lastRun: '2026-04-28 04:00', schedule: '每天 04:00', totalRuns: 18, totalRows: '5', totalSize: '1 KB', syncStrategy: 'full', schemaMeta: '源 Fiix API 驼峰字段 → Bronze 下划线列', schema: [
        { srcName: 'id', srcType: 'Number', name: 'priority_id', type: 'INT', pk: true, nullable: false, desc: '优先级 ID' },
        { srcName: 'strName', srcType: 'String', name: 'priority_name', type: 'VARCHAR(64)', desc: '优先级名称' },
        { srcName: 'intOrder', srcType: 'Number', name: 'sort_order', type: 'INT', desc: '排序序号' }
      ], extra: [
        { label: 'Endpoint', value: '/api/v5/Priority' },
        { label: '字段数', value: '3 个' }
      ]},
      't_nesr_fiix_wo_statuses': { name: '【NESR-湖仓项目】Fiix 状态字典同步', connector: 'Fiix CMMS 维修系统 · REST API', connType: 'REST API', dataType: '结构化', mode: '周期性', target: 'NESR / Bronze / fiix_wo_statuses', status: '等待中', created: '2026-04-11 04:15', lastRun: '2026-04-28 04:00', schedule: '每天 04:00', totalRuns: 18, totalRows: '6', totalSize: '1 KB', syncStrategy: 'full', schemaMeta: '源 Fiix API 驼峰字段 → Bronze 下划线列', schema: [
        { srcName: 'id', srcType: 'Number', name: 'status_id', type: 'INT', pk: true, nullable: false, desc: '状态 ID' },
        { srcName: 'strName', srcType: 'String', name: 'status_name', type: 'VARCHAR(64)', desc: '状态名称' },
        { srcName: 'intControlID', srcType: 'Number', name: 'control_id', type: 'INT', desc: '控制组 ID（开放 / 进行中 / 关闭）' }
      ], extra: [
        { label: 'Endpoint', value: '/api/v5/WorkOrderStatus' },
        { label: '字段数', value: '3 个' }
      ]},
      't_nesr_fiix_maintenance_types': { name: '【NESR-湖仓项目】Fiix 维修类型字典同步', connector: 'Fiix CMMS 维修系统 · REST API', connType: 'REST API', dataType: '结构化', mode: '周期性', target: 'NESR / Bronze / fiix_maintenance_types', status: '等待中', created: '2026-04-11 04:15', lastRun: '2026-04-28 04:00', schedule: '每天 04:00', totalRuns: 18, totalRows: '8', totalSize: '1 KB', syncStrategy: 'full', schemaMeta: '源 Fiix API 驼峰字段 → Bronze 下划线列', schema: [
        { srcName: 'id', srcType: 'Number', name: 'maintenance_type_id', type: 'INT', pk: true, nullable: false, desc: '维修类型 ID' },
        { srcName: 'strName', srcType: 'String', name: 'type_name', type: 'VARCHAR(64)', desc: '类型名称' },
        { srcName: 'intIsPlanned', srcType: 'Number', name: 'is_planned', type: 'TINYINT', desc: '是否计划内维修（PM 合规计算用）' },
        { srcName: 'dtmDateLastModified', srcType: 'DateTime', name: 'dtm_date_last_modified', type: 'DATETIME', desc: '最后修改时间' }
      ], extra: [
        { label: 'Endpoint', value: '/api/v5/MaintenanceType' },
        { label: '字段数', value: '4 个' },
        { label: '说明', value: '含 is_planned 字段用于 PM 合规计算' }
      ]},

      // ====================================================================
      // 【示例】每种连接器类型一个载入任务（共 41 个），用于演示与回归
      // 命名规则：【示例】<场景简述>，与项目实际任务通过前缀区分
      // ====================================================================

      // ---------- 对象存储 ----------
      't_demo_aliyun_oss': { name: '【示例】阿里云 OSS 营销素材同步', connector: '营销素材库 · 阿里云 OSS', connType: '阿里云 OSS（对象存储）', dataType: '非结构化', mode: '周期性', target: '默认目录 / 示例库 / 营销素材卷', status: '载入中', created: '2026-05-25 13:05', lastRun: '2026-05-25 14:00', schedule: '每小时', totalRuns: 24, totalRows: '186 个文件', totalSize: '4.2 GB', syncStrategy: 'incremental', incremental: { field: '对象 LastModified', lookback: '1 小时' }, extra: [{ label: 'Bucket', value: 'marketing-assets-cn-hangzhou' },{ label: '路径', value: 'marketing-assets/2026/' },{ label: '文件类型', value: 'PDF, PNG, JPG, MP4, ZIP' }], fileMeta: '路径 marketing-assets/2026/ · 监听对象事件（OSS Event Notification）',
        files: [
          { name: '2026-Q2-brand-banner-set.zip',      size: '128 MB', mtime: '2026-05-25 13:42', status: '已载入' },
          { name: 'brand-guidelines-v3.2.pdf',         size: '8.4 MB', mtime: '2026-05-25 12:18', status: '已载入' },
          { name: 'product-launch-demo-v4.mp4',        size: '320 MB', mtime: '2026-05-24 22:30', status: '已载入' },
          { name: 'campaign-creatives-may.zip',        size: '512 MB', mtime: '2026-05-24 16:05', status: '已载入' },
          { name: 'roadshow-photoset-shanghai.zip',    size: '1.2 GB', mtime: '2026-05-23 18:50', status: '已载入' },
          { name: 'social-cards-template-2026.psd',    size: '64 MB',  mtime: '2026-05-23 09:12', status: '已载入' }
        ]},
      't_demo_s3': { name: '【示例】AWS S3 数据归档（双向）', connector: 'NESR 备份桶 · AWS S3', connType: '标准 S3 / MinIO（对象存储）', dataType: '非结构化', mode: '周期性', target: '默认目录 / 示例库 / NESR 备份卷', status: '等待中', created: '2026-05-25 13:10', lastRun: '2026-05-25 12:00', schedule: '每天 02:00 UTC', totalRuns: 30, totalRows: '60 个归档', totalSize: '180 GB', syncStrategy: 'incremental', incremental: { field: 'S3 LastModified', lookback: '1 天' }, extra: [{ label: 'Bucket', value: 'nesr-fleet-backup' },{ label: 'Region', value: 'me-south-1（巴林）' },{ label: '保留策略', value: '90 天后转 Glacier' }], fileMeta: '每日全量快照 + 增量备份；命名规范 <env>-<date>.tar.gz.enc',
        files: [
          { name: 'nesr-prod-2026-05-25.tar.gz.enc',     size: '6.4 GB', mtime: '2026-05-25 02:18', status: '已载入' },
          { name: 'nesr-prod-2026-05-24.tar.gz.enc',     size: '6.3 GB', mtime: '2026-05-24 02:15', status: '已载入' },
          { name: 'nesr-staging-2026-05-25.tar.gz.enc',  size: '2.1 GB', mtime: '2026-05-25 02:30', status: '已载入' },
          { name: 'metabase-export-2026-05-25.parquet',  size: '380 MB', mtime: '2026-05-25 03:45', status: '已载入' }
        ]},
      't_demo_cos': { name: '【示例】腾讯云 COS 课程视频载入', connector: '视频内容库 · 腾讯云 COS', connType: '腾讯云 COS（对象存储）', dataType: '非结构化', mode: '周期性', target: '默认目录 / 示例库 / 视频素材卷', status: '等待中', created: '2026-05-25 13:15', lastRun: '2026-05-25 06:00', schedule: '每天 06:00', totalRuns: 30, totalRows: '420 个视频', totalSize: '86 GB', syncStrategy: 'incremental', incremental: { field: '对象 LastModified', lookback: '1 天' }, extra: [{ label: 'Bucket', value: 'media-library-1250000000' },{ label: '路径', value: 'videos/raw/' },{ label: '后续处理', value: '转码 + 抽帧 → 知识库切片' }], fileMeta: 'videos/raw/ 下按 course_id/episode_id 分层',
        files: [
          { name: 'course-101/episode-12-circuit-analysis.mp4', size: '480 MB', mtime: '2026-05-25 05:18', status: '已载入' },
          { name: 'course-102/episode-03-power-supply.mp4',     size: '520 MB', mtime: '2026-05-25 05:25', status: '已载入' },
          { name: 'course-103/episode-08-mosfet-design.mp4',    size: '610 MB', mtime: '2026-05-25 05:42', status: '已载入' },
          { name: 'course-201/episode-01-intro.mp4',            size: '380 MB', mtime: '2026-05-24 22:10', status: '已载入' }
        ]},
      't_demo_obs': { name: '【示例】华为云 OBS 合规报告归档', connector: '政企归档 · 华为云 OBS', connType: '华为云 OBS（对象存储）', dataType: '非结构化', mode: '周期性', target: '默认目录 / 示例库 / 合规归档卷（导出方向）', status: '载入失败', created: '2026-05-25 13:20', lastRun: '2026-05-25 03:00', schedule: '每周一 03:00', totalRuns: 12, totalRows: '48 份报告', totalSize: '1.4 GB', syncStrategy: 'full', extra: [{ label: 'Bucket', value: 'gov-archive-bj（华北-北京四）' },{ label: '方向', value: '导出（MOI → OBS）' },{ label: '加密', value: 'SSE-KMS' }], fileMeta: '每周导出审计报告到 OBS 长期保留 7 年',
        files: [
          { name: 'compliance-reports/2026-W21-audit.pdf',  size: '12 MB', mtime: '2026-05-25 03:08', status: '已载入' },
          { name: 'compliance-reports/2026-W20-audit.pdf',  size: '11 MB', mtime: '2026-05-18 03:05', status: '已载入' },
          { name: 'compliance-reports/2026-Q1-summary.pdf', size: '28 MB', mtime: '2026-04-01 03:12', status: '已载入' }
        ]},

      // ---------- 分布式文件系统 ----------
      't_demo_hdfs': { name: '【示例】HDFS 行为日志载入', connector: '集团数据湖 · HDFS', connType: 'HDFS（分布式文件系统）', dataType: '非结构化', mode: '周期性', target: '默认目录 / 示例库 / 行为日志卷', status: '回填中', created: '2026-05-25 13:25', lastRun: '2026-05-25 14:00', schedule: '每小时', totalRuns: 120, totalRows: '2,880 个 parquet 文件', totalSize: '420 GB', syncStrategy: 'incremental', incremental: { field: '分区 dt=YYYY-MM-DD/hr=HH', lookback: '2 小时' }, backfill: { enabled: true, progress: '64%', startDate: '2025-11-01', batchSize: '1 天', batchInterval: '15 秒', window: '不限制' }, extra: [{ label: 'NameNode', value: 'hdfs://nn-master.corp.local:8020' },{ label: '路径', value: '/warehouse/events/' },{ label: '格式', value: 'Parquet (Snappy 压缩)' }], fileMeta: '按分区 dt/hr 拉取，单文件 ~150 MB',
        files: [
          { name: '/warehouse/events/dt=2026-05-25/hr=13/part-00000.parquet', size: '156 MB', mtime: '2026-05-25 14:02', status: '已载入' },
          { name: '/warehouse/events/dt=2026-05-25/hr=13/part-00001.parquet', size: '148 MB', mtime: '2026-05-25 14:02', status: '已载入' },
          { name: '/warehouse/events/dt=2026-05-25/hr=12/part-00000.parquet', size: '162 MB', mtime: '2026-05-25 13:02', status: '已载入' },
          { name: '/warehouse/events/dt=2026-05-25/hr=12/part-00001.parquet', size: '154 MB', mtime: '2026-05-25 13:02', status: '已载入' }
        ]},

      // ---------- 数据库 ----------
      't_demo_matrixone': { name: '【示例】MatrixOne 用户行为表同步', connector: 'MatrixOne 生产集群', connType: 'MatrixOne（数据库）', dataType: '结构化', mode: '周期性', target: '默认目录 / 示例库 / user_events', status: '等待中', created: '2026-05-25 13:30', lastRun: '2026-05-25 14:00', schedule: '每小时', totalRuns: 96, totalRows: '8,420,000', totalSize: '1.6 GB', syncStrategy: 'incremental', incremental: { field: 'event_time', lookback: '1 小时' }, schemaMeta: '源表 moi_warehouse.user_events · MO MO 实时写入 + 增量拉取',
        schema: [
          { name: 'event_id',     type: 'BIGINT',    pk: true,  nullable: false, desc: '事件 ID' },
          { name: 'user_id',      type: 'BIGINT',    pk: false, nullable: false, desc: '用户 ID' },
          { name: 'event_type',   type: 'VARCHAR(32)', pk: false, nullable: false, desc: 'click / view / purchase 等' },
          { name: 'event_time',   type: 'TIMESTAMP', pk: false, nullable: false, desc: '事件发生时间（增量字段）' },
          { name: 'session_id',   type: 'VARCHAR(64)', pk: false, desc: '会话 ID' },
          { name: 'page_url',     type: 'VARCHAR(512)', pk: false, desc: '触发页 URL' },
          { name: 'props',        type: 'JSON',      pk: false, desc: '事件附加属性（嵌套对象）' }
        ]},
      't_demo_mysql': { name: '【示例】MySQL 用户中心同步', connector: '用户中心 · MySQL', connType: 'MySQL（数据库）', dataType: '结构化', mode: '周期性', target: '默认目录 / 示例库 / users', status: '等待中', created: '2026-05-25 13:35', lastRun: '2026-05-25 14:00', schedule: '每小时', totalRuns: 96, totalRows: '1,256,000', totalSize: '380 MB', syncStrategy: 'incremental', incremental: { field: 'updated_at', lookback: '1 小时' }, schemaMeta: '源表 user_center.users · 软删（is_deleted 标记） + 增量字段',
        schema: [
          { name: 'id',           type: 'BIGINT',    pk: true,  nullable: false, desc: '用户 ID' },
          { name: 'username',     type: 'VARCHAR(64)', pk: false, nullable: false, desc: '登录名（唯一）' },
          { name: 'email',        type: 'VARCHAR(128)', pk: false, desc: '邮箱' },
          { name: 'phone',        type: 'VARCHAR(20)', pk: false, desc: '手机号（已脱敏）' },
          { name: 'status',       type: 'TINYINT',   pk: false, nullable: false, desc: '0=禁用 1=正常 2=锁定' },
          { name: 'created_at',   type: 'DATETIME',  pk: false, nullable: false, desc: '创建时间' },
          { name: 'updated_at',   type: 'DATETIME',  pk: false, nullable: false, desc: '最后更新时间（增量字段）' },
          { name: 'is_deleted',   type: 'TINYINT',   pk: false, desc: '软删标记' }
        ]},
      't_demo_pg': { name: '【示例】PostgreSQL 财务汇总同步', connector: '财务报表库 · PostgreSQL', connType: 'PostgreSQL（数据库）', dataType: '结构化', mode: '周期性', target: '默认目录 / 示例库 / financial_summary', status: '等待中', created: '2026-05-25 13:40', lastRun: '2026-05-25 06:00', schedule: '每天 06:00', totalRuns: 30, totalRows: '21,600', totalSize: '8.4 MB', syncStrategy: 'incremental', incremental: { field: 'report_date', lookback: '7 天' }, schemaMeta: '源表 reporting.daily_financial_summary · 每日生成 T-1 数据',
        schema: [
          { name: 'report_date',  type: 'DATE',      pk: true,  nullable: false, desc: '报表日期（增量字段）' },
          { name: 'business_unit',type: 'VARCHAR(32)', pk: true, nullable: false, desc: 'BU 编码' },
          { name: 'revenue',      type: 'NUMERIC(18,2)', pk: false, desc: '当日营收（元）' },
          { name: 'cost',         type: 'NUMERIC(18,2)', pk: false, desc: '当日成本' },
          { name: 'gross_profit', type: 'NUMERIC(18,2)', pk: false, desc: '毛利 = revenue - cost' },
          { name: 'gross_margin', type: 'NUMERIC(5,4)',  pk: false, desc: '毛利率' },
          { name: 'currency',     type: 'CHAR(3)',   pk: false, desc: 'ISO 货币码' }
        ]},
      't_demo_mongodb': { name: '【示例】MongoDB 商品目录同步', connector: 'Intelie 传感器数据', connType: 'MongoDB（数据库）', dataType: '结构化', mode: '周期性', target: '默认目录 / 示例库 / products', status: '等待中', created: '2026-05-25 13:45', lastRun: '2026-05-25 14:00', schedule: '每小时', totalRuns: 96, totalRows: '156,000', totalSize: '62 MB', syncStrategy: 'incremental', incremental: { field: 'updatedAt', lookback: '1 小时' }, schemaMeta: 'Collection products · JSON 文档展平 + 嵌套字段保留为 JSON',
        schema: [
          { name: '_id',          type: 'VARCHAR(24)', pk: true,  nullable: false, desc: 'MongoDB ObjectId' },
          { name: 'sku',          type: 'VARCHAR(32)', pk: false, nullable: false, desc: '商品编码' },
          { name: 'name',         type: 'VARCHAR(256)', pk: false, desc: '商品名' },
          { name: 'category',     type: 'VARCHAR(64)', pk: false, desc: '一级分类' },
          { name: 'price',        type: 'DECIMAL(12,2)', pk: false, desc: '当前定价' },
          { name: 'attributes',   type: 'JSON',      pk: false, desc: '嵌套属性（颜色/尺寸/规格...）' },
          { name: 'updatedAt',    type: 'TIMESTAMP', pk: false, nullable: false, desc: '最后更新（增量字段）' }
        ]},
      't_demo_hive': { name: '【示例】Hive DWD 周报加工', connector: 'NESR 数据仓库 Hive', connType: 'Hive（数据库）', dataType: '结构化', mode: '周期性', target: '默认目录 / 示例库 / dwd_weekly_kpi', status: '等待中', created: '2026-05-25 13:50', lastRun: '2026-05-25 04:00', schedule: '每周一 04:00', totalRuns: 12, totalRows: '8,400', totalSize: '12 MB', syncStrategy: 'incremental', incremental: { field: 'partition stat_week', lookback: '0' }, schemaMeta: '源表 dw_dwd.dwd_well_kpi_weekly · 按分区 stat_week 拉取',
        schema: [
          { name: 'well_id',          type: 'VARCHAR(32)', pk: true, nullable: false, desc: '井号' },
          { name: 'stat_week',        type: 'VARCHAR(10)', pk: true, nullable: false, desc: '统计周（2026-W21，分区键）' },
          { name: 'production_bbl',   type: 'DOUBLE',    pk: false, desc: '周产油（桶）' },
          { name: 'utilization_pct',  type: 'DOUBLE',    pk: false, desc: '利用率 %' },
          { name: 'mtbf_hours',       type: 'DOUBLE',    pk: false, desc: '周 MTBF' },
          { name: 'alarm_count',      type: 'INT',       pk: false, desc: '告警次数' },
          { name: 'dwh_etl_time',     type: 'TIMESTAMP', pk: false, desc: 'DWH 加工时间' }
        ]},
      't_demo_sqlserver': { name: '【示例】SQL Server 报价单同步', connector: '汉得 Topcast · SQL Server', connType: 'SQL Server（数据库）', dataType: '结构化', mode: '周期性', target: '默认目录 / 示例库 / quote_orders', status: '载入中', created: '2026-05-25 13:55', lastRun: '2026-05-25 14:00', schedule: '每 30 分钟', totalRuns: 280, totalRows: '38,200', totalSize: '14 MB', syncStrategy: 'incremental', incremental: { field: 'LastModified', lookback: '30 分钟' }, schemaMeta: '源表 TopcastQuote.dbo.QUOTE_HDR · 报价单主表',
        schema: [
          { name: 'QUOTE_ID',     type: 'BIGINT',     pk: true,  nullable: false, desc: '报价单 ID' },
          { name: 'CUSTOMER_EMAIL', type: 'NVARCHAR(128)', pk: false, desc: '客户邮箱' },
          { name: 'ISAI',         type: 'NVARCHAR(64)', pk: false, desc: '内部销售标识' },
          { name: 'SUBJECT',      type: 'NVARCHAR(256)', pk: false, desc: '邮件主题' },
          { name: 'TOTAL_LINES',  type: 'INT',        pk: false, desc: '报价行数' },
          { name: 'CURRENCY',     type: 'CHAR(3)',    pk: false, desc: 'ISO 货币' },
          { name: 'CREATED_AT',   type: 'DATETIME2',  pk: false, nullable: false, desc: '创建时间' },
          { name: 'LAST_MODIFIED',type: 'DATETIME2',  pk: false, nullable: false, desc: '增量字段' }
        ]},
      't_demo_oracle': { name: '【示例】Oracle ERP 总账分录同步', connector: '武汉新芯 · Oracle ERP DB', connType: 'Oracle（数据库）', dataType: '结构化', mode: '周期性', target: '默认目录 / 示例库 / gl_je_lines', status: '等待中', created: '2026-05-25 14:00', lastRun: '2026-05-25 05:00', schedule: '每天 05:00', totalRuns: 30, totalRows: '480,000', totalSize: '156 MB', syncStrategy: 'incremental', incremental: { field: 'LAST_UPDATE_DATE', lookback: '1 天' }, schemaMeta: '源表 GL.JE_LINES · 总账分录明细',
        schema: [
          { name: 'JE_LINE_ID',       type: 'NUMBER(15)', pk: true,  nullable: false, desc: '分录行 ID' },
          { name: 'JE_HEADER_ID',     type: 'NUMBER(15)', pk: false, nullable: false, desc: '所属分录头' },
          { name: 'CODE_COMBINATION_ID', type: 'NUMBER(15)', pk: false, desc: '科目组合 ID' },
          { name: 'ENTERED_DR',       type: 'NUMBER',     pk: false, desc: '借方金额' },
          { name: 'ENTERED_CR',       type: 'NUMBER',     pk: false, desc: '贷方金额' },
          { name: 'PERIOD_NAME',      type: 'VARCHAR2(15)', pk: false, desc: '会计期间' },
          { name: 'LAST_UPDATE_DATE', type: 'DATE',       pk: false, nullable: false, desc: '增量字段' }
        ]},
      't_demo_clickhouse': { name: '【示例】ClickHouse 港股 Tick 同步', connector: '芯联汉得 · ClickHouse 行情库', connType: 'ClickHouse（数据库）', dataType: '结构化', mode: '周期性', target: '默认目录 / 示例库 / hkex_tick', status: '回填中', created: '2026-05-25 14:05', lastRun: '2026-05-25 14:00', schedule: '每分钟', totalRuns: 7200, totalRows: '1,260,000,000', totalSize: '38 GB', syncStrategy: 'incremental', incremental: { field: 'tick_time', lookback: '5 分钟' }, backfill: { enabled: true, progress: '88%', startDate: '2024-01-01', batchSize: '1 天', batchInterval: '3 秒', window: '不限制' }, schemaMeta: '源表 hkex_market.tick · 港股逐笔成交（每秒 50K+ 行）',
        schema: [
          { name: 'tick_time',    type: 'DateTime64(3)', pk: true,  nullable: false, desc: '毫秒级时间戳' },
          { name: 'symbol',       type: 'String',     pk: true,  nullable: false, desc: '股票代码' },
          { name: 'price',        type: 'Decimal(12,4)', pk: false, desc: '成交价' },
          { name: 'volume',       type: 'UInt64',     pk: false, desc: '成交量' },
          { name: 'side',         type: 'Enum8(\\\'B\\\'=1, \\\'S\\\'=2)', pk: false, desc: '买卖方向' },
          { name: 'trade_type',   type: 'LowCardinality(String)', pk: false, desc: '成交类型' }
        ]},
      't_demo_doris': { name: '【示例】Doris 实时指标同步', connector: '业务实时分析 · Apache Doris', connType: 'Apache Doris（数据库）', dataType: '结构化', mode: '周期性', target: '默认目录 / 示例库 / rt_metrics', status: '等待中', created: '2026-05-25 14:10', lastRun: '2026-05-25 14:00', schedule: '每 5 分钟', totalRuns: 8640, totalRows: '180,000', totalSize: '420 MB', syncStrategy: 'incremental', incremental: { field: 'stat_time', lookback: '15 分钟' }, schemaMeta: '源表 rt_analytics.minute_metrics · 分钟级聚合指标',
        schema: [
          { name: 'stat_time',    type: 'DATETIME',  pk: true,  nullable: false, desc: '统计分钟' },
          { name: 'metric_name',  type: 'VARCHAR(64)', pk: true, nullable: false, desc: '指标名（PV/UV/订单数...）' },
          { name: 'dim_channel',  type: 'VARCHAR(32)', pk: true, desc: '渠道维度' },
          { name: 'metric_value', type: 'DOUBLE',    pk: false, desc: '指标值' },
          { name: 'sample_count', type: 'BIGINT',    pk: false, desc: '采样次数' }
        ]},

      // ---------- 邮件 ----------
      't_demo_gmail': { name: '【示例】Gmail 客户邮件载入', connector: '客户服务 Gmail 收件箱', connType: 'Gmail（邮件）', dataType: '结构化', mode: '周期性', target: '默认目录 / 示例库 / gmail_messages', status: '等待中', created: '2026-05-25 14:15', lastRun: '2026-05-25 14:00', schedule: '每 5 分钟', totalRuns: 8640, totalRows: '48,200', totalSize: '186 MB', syncStrategy: 'incremental', incremental: { field: 'internalDate', lookback: '15 分钟' }, schemaMeta: '邮件元数据 + 正文文本（附件落 Volume 不入此表）',
        schema: [
          { name: 'message_id',   type: 'VARCHAR(64)', pk: true,  nullable: false, desc: 'Gmail 全局唯一 ID' },
          { name: 'thread_id',    type: 'VARCHAR(64)', pk: false, desc: '会话线索 ID' },
          { name: 'from_addr',    type: 'VARCHAR(256)', pk: false, nullable: false, desc: '发件人' },
          { name: 'to_addrs',     type: 'JSON',      pk: false, desc: '收件人列表' },
          { name: 'subject',      type: 'VARCHAR(512)', pk: false, desc: '主题' },
          { name: 'body_text',    type: 'TEXT',      pk: false, desc: '正文（HTML 净化后）' },
          { name: 'has_attachment', type: 'BOOLEAN', pk: false, desc: '是否有附件' },
          { name: 'internalDate', type: 'TIMESTAMP', pk: false, nullable: false, desc: '收件时间（增量字段）' }
        ]},
      't_demo_outlook': { name: '【示例】Outlook 法务邮件载入', connector: '法务 Outlook 收件箱', connType: 'Outlook（邮件）', dataType: '结构化', mode: '周期性', target: '默认目录 / 示例库 / outlook_messages', status: '等待中', created: '2026-05-25 14:20', lastRun: '2026-05-25 14:00', schedule: '每 15 分钟', totalRuns: 2880, totalRows: '12,800', totalSize: '52 MB', syncStrategy: 'incremental', incremental: { field: 'receivedDateTime', lookback: '30 分钟' }, schemaMeta: '通过 Outlook REST API 读取，含分类标签',
        schema: [
          { name: 'message_id',       type: 'VARCHAR(128)', pk: true, nullable: false, desc: 'Outlook 消息 ID' },
          { name: 'conversation_id',  type: 'VARCHAR(128)', pk: false, desc: '会话 ID' },
          { name: 'from_addr',        type: 'VARCHAR(256)', pk: false, nullable: false, desc: '发件人' },
          { name: 'subject',          type: 'VARCHAR(512)', pk: false, desc: '主题' },
          { name: 'body_preview',     type: 'VARCHAR(1024)', pk: false, desc: '正文预览' },
          { name: 'categories',       type: 'JSON',      pk: false, desc: 'Outlook 分类标签' },
          { name: 'importance',       type: 'VARCHAR(16)', pk: false, desc: 'low / normal / high' },
          { name: 'receivedDateTime', type: 'TIMESTAMP', pk: false, nullable: false, desc: '收件时间（增量字段）' }
        ]},
      't_demo_ms365_graph': { name: '【示例】Microsoft 365 全员邮件采集', connector: '全员 Microsoft 365 邮箱', connType: 'Microsoft 365 (Graph)（邮件）', dataType: '结构化', mode: '周期性', target: '默认目录 / 示例库 / ms365_messages', status: '回填中', created: '2026-05-25 14:25', lastRun: '2026-05-25 14:00', schedule: '每 5 分钟', totalRuns: 8640, totalRows: '1,560,000', totalSize: '6.2 GB', syncStrategy: 'incremental', incremental: { field: 'receivedDateTime', lookback: '15 分钟' }, backfill: { enabled: true, progress: '38%', startDate: '2025-01-01', batchSize: '1 天', batchInterval: '5 秒', window: '00:00 - 08:00' }, schemaMeta: '通过 Graph delta query 增量；按用户邮箱白名单过滤',
        schema: [
          { name: 'message_id',       type: 'VARCHAR(256)', pk: true,  nullable: false, desc: 'Graph 全局 ID' },
          { name: 'user_principal',   type: 'VARCHAR(256)', pk: false, nullable: false, desc: '所属邮箱（UPN）' },
          { name: 'from_addr',        type: 'VARCHAR(256)', pk: false, desc: '发件人' },
          { name: 'subject',          type: 'VARCHAR(512)', pk: false, desc: '主题' },
          { name: 'body_text',        type: 'TEXT',      pk: false, desc: '正文文本' },
          { name: 'mailbox_folder',   type: 'VARCHAR(64)', pk: false, desc: '收件箱 / 已发送 / 分类文件夹' },
          { name: 'receivedDateTime', type: 'TIMESTAMP', pk: false, nullable: false, desc: '收件时间（增量字段）' }
        ]},
      't_demo_wecom_mail': { name: '【示例】企业微信邮箱采集', connector: '企业微信项目邮箱', connType: '企业微信邮箱（邮件）', dataType: '结构化', mode: '周期性', target: '默认目录 / 示例库 / wecom_mail_messages', status: '等待中', created: '2026-05-25 14:30', lastRun: '2026-05-25 14:00', schedule: '每 10 分钟', totalRuns: 4320, totalRows: '8,600', totalSize: '36 MB', syncStrategy: 'incremental', incremental: { field: 'mail_date', lookback: '30 分钟' }, schemaMeta: 'IMAP 拉取 + 标准邮件解析',
        schema: [
          { name: 'message_id', type: 'VARCHAR(256)', pk: true, nullable: false, desc: 'Message-ID' },
          { name: 'from_addr',  type: 'VARCHAR(256)', pk: false, nullable: false, desc: '发件人' },
          { name: 'subject',    type: 'VARCHAR(512)', pk: false, desc: '主题' },
          { name: 'body_text',  type: 'TEXT',     pk: false, desc: '正文' },
          { name: 'mail_date',  type: 'TIMESTAMP', pk: false, nullable: false, desc: '邮件时间（增量字段）' }
        ]},
      't_demo_qq_mail': { name: '【示例】QQ 邮箱采集', connector: '运营 QQ 邮箱', connType: 'QQ 邮箱（邮件）', dataType: '结构化', mode: '周期性', target: '默认目录 / 示例库 / qq_mail_messages', status: '载入失败', created: '2026-05-25 14:35', lastRun: '2026-05-25 14:00', schedule: '每 10 分钟', totalRuns: 4320, totalRows: '3,400', totalSize: '14 MB', syncStrategy: 'incremental', incremental: { field: 'mail_date', lookback: '30 分钟' }, schemaMeta: 'IMAP/POP3 拉取',
        schema: [
          { name: 'message_id', type: 'VARCHAR(256)', pk: true, nullable: false, desc: 'Message-ID' },
          { name: 'from_addr',  type: 'VARCHAR(256)', pk: false, nullable: false, desc: '发件人' },
          { name: 'subject',    type: 'VARCHAR(512)', pk: false, desc: '主题' },
          { name: 'body_text',  type: 'TEXT',      pk: false, desc: '正文' },
          { name: 'mail_date',  type: 'TIMESTAMP', pk: false, nullable: false, desc: '邮件时间（增量字段）' }
        ]},
      't_demo_imap': { name: '【示例】内部 IMAP 邮箱采集', connector: '内部 IMAP 邮箱', connType: 'IMAP/SMTP 通用（邮件）', dataType: '结构化', mode: '周期性', target: '默认目录 / 示例库 / internal_mail', status: '等待中', created: '2026-05-25 14:40', lastRun: '2026-05-25 14:00', schedule: '每小时', totalRuns: 720, totalRows: '4,200', totalSize: '18 MB', syncStrategy: 'incremental', incremental: { field: 'received_at', lookback: '2 小时' }, schemaMeta: '兜底 IMAP/SMTP，适配任意标准邮件服务器',
        schema: [
          { name: 'message_id', type: 'VARCHAR(256)', pk: true, nullable: false, desc: 'Message-ID' },
          { name: 'from_addr',  type: 'VARCHAR(256)', pk: false, nullable: false, desc: '发件人' },
          { name: 'to_addrs',   type: 'JSON',     pk: false, desc: '收件人列表' },
          { name: 'subject',    type: 'VARCHAR(512)', pk: false, desc: '主题' },
          { name: 'body_text',  type: 'TEXT',      pk: false, desc: '正文' },
          { name: 'received_at',type: 'TIMESTAMP', pk: false, nullable: false, desc: '收件时间（增量字段）' }
        ]},
      't_demo_custom_mail_api': { name: '【示例】自有邮件 API 同步', connector: '自有邮件 API', connType: '自有邮件 API（邮件）', dataType: '结构化', mode: '周期性', target: '默认目录 / 示例库 / custom_mail', status: '等待中', created: '2026-05-25 14:45', lastRun: '2026-05-25 14:00', schedule: '每 5 分钟', totalRuns: 8640, totalRows: '12,400', totalSize: '48 MB', syncStrategy: 'incremental', incremental: { field: 'sent_at', lookback: '15 分钟' }, schemaMeta: '企业自有邮件平台 REST API；按 Bearer Token 认证',
        schema: [
          { name: 'msg_id',    type: 'VARCHAR(64)', pk: true, nullable: false, desc: '内部消息 ID' },
          { name: 'from_user', type: 'VARCHAR(128)', pk: false, nullable: false, desc: '发送用户' },
          { name: 'topic',     type: 'VARCHAR(256)', pk: false, desc: '主题' },
          { name: 'content',   type: 'TEXT',     pk: false, desc: '内容' },
          { name: 'sent_at',   type: 'TIMESTAMP', pk: false, nullable: false, desc: '发送时间（增量字段）' }
        ]},

      // ---------- API ----------
      't_demo_rest_api': { name: '【示例】天气 API 拉取', connector: 'Fiix CMMS 维修系统', connType: 'REST API', dataType: '结构化', mode: '周期性', target: '默认目录 / 示例库 / weather_hourly', status: '等待中', created: '2026-05-25 14:50', lastRun: '2026-05-25 14:00', schedule: '每小时', totalRuns: 720, totalRows: '20,160', totalSize: '4.2 MB', syncStrategy: 'incremental', incremental: { field: 'obs_time', lookback: '2 小时' }, schemaMeta: 'GET /weather?city=...&hour=... · JSON 数据路径 data.observations',
        schema: [
          { name: 'city_code',   type: 'VARCHAR(8)', pk: true, nullable: false, desc: '城市编码' },
          { name: 'obs_time',    type: 'TIMESTAMP', pk: true, nullable: false, desc: '观测时刻（增量字段）' },
          { name: 'temp_c',      type: 'FLOAT',    pk: false, desc: '气温 (°C)' },
          { name: 'humidity_pct',type: 'FLOAT',    pk: false, desc: '湿度 %' },
          { name: 'precip_mm',   type: 'FLOAT',    pk: false, desc: '降水 mm' },
          { name: 'wind_kph',    type: 'FLOAT',    pk: false, desc: '风速 km/h' },
          { name: 'condition',   type: 'VARCHAR(32)', pk: false, desc: '天气描述' }
        ]},
      't_demo_graphql': { name: '【示例】GraphQL 电商订单同步', connector: '电商订单 · GraphQL', connType: 'GraphQL（API）', dataType: '结构化', mode: '周期性', target: '默认目录 / 示例库 / shop_orders', status: '等待中', created: '2026-05-25 14:55', lastRun: '2026-05-25 14:00', schedule: '每 15 分钟', totarRuns: 2880, totalRows: '86,400', totalSize: '42 MB', totalRuns: 2880, syncStrategy: 'incremental', incremental: { field: 'updatedAt', lookback: '30 分钟' }, schemaMeta: 'query { orders(after: $cursor) { ... } } · cursor-based 分页',
        schema: [
          { name: 'order_id',    type: 'VARCHAR(32)', pk: true, nullable: false, desc: '订单 ID' },
          { name: 'customer_id', type: 'BIGINT',  pk: false, nullable: false, desc: '客户 ID' },
          { name: 'total_amount',type: 'DECIMAL(12,2)', pk: false, desc: '订单金额' },
          { name: 'currency',    type: 'CHAR(3)', pk: false, desc: '币种' },
          { name: 'status',      type: 'VARCHAR(16)', pk: false, desc: 'created/paid/shipped/done' },
          { name: 'items',       type: 'JSON',    pk: false, desc: '订单行（嵌套数组）' },
          { name: 'updatedAt',   type: 'TIMESTAMP', pk: false, nullable: false, desc: '增量字段' }
        ]},

      // ---------- 文件协议 ----------
      't_demo_ftp': { name: '【示例】FTP 现场日报载入', connector: 'NESR 现场文件服务器 · FTP', connType: 'FTP（文件协议）', dataType: '非结构化', mode: '周期性', target: '默认目录 / 示例库 / 现场日报卷', status: '等待中', created: '2026-05-25 15:00', lastRun: '2026-05-25 06:30', schedule: '每天 06:30', totalRuns: 30, totalRows: '180 个报告', totalSize: '320 MB', syncStrategy: 'incremental', incremental: { field: '文件 mtime', lookback: '1 天' }, extra: [{ label: 'FTP 路径', value: '/data/well-reports/' },{ label: '模式', value: '被动 (PASV)' }], fileMeta: '现场工程师每日上传，CSV + 扫描件 PDF',
        files: [
          { name: 'well-W001-daily-2026-05-25.csv',  size: '128 KB', mtime: '2026-05-25 06:15', status: '已载入' },
          { name: 'well-W002-daily-2026-05-25.csv',  size: '142 KB', mtime: '2026-05-25 06:20', status: '已载入' },
          { name: 'inspection-2026-05-24-rig7.pdf',  size: '4.2 MB', mtime: '2026-05-24 18:32', status: '已载入' },
          { name: 'safety-checklist-2026-05-24.pdf', size: '1.8 MB', mtime: '2026-05-24 19:05', status: '已载入' }
        ]},
      't_demo_sftp': { name: '【示例】SFTP 客户合同同步', connector: '客户合同同步 · SFTP', connType: 'SFTP（文件协议）', dataType: '非结构化', mode: '周期性', target: '默认目录 / 示例库 / 合同档案卷', status: '载入失败', created: '2026-05-25 15:05', lastRun: '2026-05-25 02:00', schedule: '每天 02:00', totalRuns: 60, totalRows: '128 份合同', totalSize: '52 MB', syncStrategy: 'incremental', incremental: { field: '文件 mtime', lookback: '1 天' }, extra: [{ label: '远程路径', value: '/uploads/contracts/' },{ label: '认证', value: 'SSH 私钥' }], fileMeta: '合作方夜间推送签署合同 PDF',
        files: [
          { name: 'contract-CT-2026-0524-acme.pdf',   size: '380 KB', mtime: '2026-05-24 23:45', status: '已载入' },
          { name: 'contract-CT-2026-0523-globex.pdf', size: '420 KB', mtime: '2026-05-23 22:12', status: '已载入' },
          { name: 'NDA-CT-2026-0522-soylent.pdf',     size: '280 KB', mtime: '2026-05-22 21:30', status: '已载入' },
          { name: 'amend-CT-2025-0418-initech.pdf',   size: '156 KB', mtime: '2026-05-22 09:18', status: '已载入' }
        ]},
      't_demo_sharepoint': { name: '【示例】SharePoint 问卷文档载入', connector: '有临医药 · SharePoint 站点', connType: 'SharePoint（文件协议）', dataType: '非结构化', mode: '周期性', target: '默认目录 / 示例库 / 问卷文档卷', status: '等待中', created: '2026-05-25 15:10', lastRun: '2026-05-25 12:00', schedule: '每 6 小时', totalRuns: 120, totalRows: '420 份问卷', totalSize: '186 MB', syncStrategy: 'incremental', incremental: { field: 'Modified', lookback: '6 小时' }, extra: [{ label: '站点', value: 'youlin.sharepoint.com/sites/cro-agent' },{ label: '文档库', value: 'Documents / 问卷/2026' }], fileMeta: '中心研究员上传 PDF/Word 问卷',
        files: [
          { name: '中心问卷-中医药大学附属医院-2026Q2.docx',         size: '128 KB', mtime: '2026-05-25 11:42', status: '已载入' },
          { name: '中心问卷-协和-肿瘤中心-填报版.pdf',                size: '420 KB', mtime: '2026-05-24 17:08', status: '已载入' },
          { name: '中心问卷-上海市六院-2026Q2.docx',                  size: '108 KB', mtime: '2026-05-24 10:15', status: '已载入' },
          { name: '问卷模板-CRO-Standard-V3.docx',                    size: '96 KB',  mtime: '2026-05-20 09:00', status: '已载入' }
        ]},
      't_demo_smb': { name: '【示例】NAS 部门资料同步', connector: '集团 NAS · SMB 共享', connType: 'NAS (SMB/CIFS)（文件协议）', dataType: '非结构化', mode: '周期性', target: '默认目录 / 示例库 / 部门资料卷', status: '等待中', created: '2026-05-25 15:15', lastRun: '2026-05-25 13:00', schedule: '每小时', totalRuns: 720, totalRows: '2,400 个文件', totalSize: '12 GB', syncStrategy: 'incremental', incremental: { field: '文件 mtime', lookback: '2 小时' }, extra: [{ label: '共享', value: '\\\\nas01\\department-data' },{ label: 'SMB 版本', value: 'SMB 3.0' }], fileMeta: '各部门 Excel/PPT/Word 文档；按部门子目录组织',
        files: [
          { name: 'finance/2026-Q1-budget-final.xlsx',        size: '2.4 MB', mtime: '2026-05-25 12:45', status: '已载入' },
          { name: 'sales/pipeline-may-week4.xlsx',            size: '1.8 MB', mtime: '2026-05-25 11:30', status: '已载入' },
          { name: 'hr/headcount-plan-2026-H2.pptx',           size: '4.2 MB', mtime: '2026-05-24 16:20', status: '已载入' },
          { name: 'rd/architecture-review-2026-may.docx',     size: '820 KB', mtime: '2026-05-24 14:15', status: '已载入' }
        ]},

      // ---------- 消息队列 ----------
      't_demo_kafka': { name: '【示例】Kafka GPS 流实时摄取', connector: 'CDG GPS 流 · Kafka', connType: 'Apache Kafka（消息队列）', dataType: '结构化', mode: '实时', target: '默认目录 / 示例库 / gps_events', status: '载入中', created: '2026-05-25 15:20', lastRun: '实时', schedule: '常驻消费', totalRuns: '—', totalRows: '8,640,000', totalSize: '2.8 GB', syncStrategy: 'incremental', incremental: { field: 'Kafka offset', lookback: '5 分钟' }, schemaMeta: 'topic: taxi.gps.* · consumer group moi-gps-ingest · 微批 5s 写入',
        schema: [
          { name: 'event_time',  type: 'TIMESTAMP(3)', pk: false, nullable: false, desc: '事件时间' },
          { name: 'taxi_id',     type: 'VARCHAR(32)',  pk: false, nullable: false, desc: '出租车 ID' },
          { name: 'driver_id',   type: 'VARCHAR(32)',  pk: false, desc: '司机 ID' },
          { name: 'latitude',    type: 'DOUBLE',       pk: false, desc: '纬度' },
          { name: 'longitude',   type: 'DOUBLE',       pk: false, desc: '经度' },
          { name: 'speed_kph',   type: 'FLOAT',        pk: false, desc: '速度' },
          { name: 'heading_deg', type: 'FLOAT',        pk: false, desc: '航向角' },
          { name: 'status',      type: 'VARCHAR(16)',  pk: false, desc: 'idle/occupied/break' }
        ]},
      't_demo_mqtt': { name: '【示例】MQTT IoT 设备遥测', connector: 'IoT 设备遥测 · MQTT', connType: 'MQTT（消息队列）', dataType: '结构化', mode: '实时', target: '默认目录 / 示例库 / iot_telemetry', status: '载入中', created: '2026-05-25 15:25', lastRun: '实时', schedule: '常驻订阅', totalRuns: '—', totalRows: '12,400,000', totalSize: '3.6 GB', syncStrategy: 'incremental', incremental: { field: 'ts', lookback: '10 分钟' }, schemaMeta: 'topic: iot/+/telemetry · QoS 1 · 微批 10s 写入',
        schema: [
          { name: 'device_id',  type: 'VARCHAR(64)',  pk: false, nullable: false, desc: '设备 ID' },
          { name: 'ts',         type: 'TIMESTAMP(3)', pk: false, nullable: false, desc: '时间戳' },
          { name: 'metric',     type: 'VARCHAR(32)',  pk: false, desc: 'temp/humidity/pressure...' },
          { name: 'value',      type: 'DOUBLE',       pk: false, desc: '指标值' },
          { name: 'unit',       type: 'VARCHAR(16)',  pk: false, desc: '单位' },
          { name: 'site_id',    type: 'VARCHAR(32)',  pk: false, desc: '站点 ID' }
        ]},
      't_demo_rabbitmq': { name: '【示例】RabbitMQ 通知事件消费', connector: '通知服务 · RabbitMQ', connType: 'RabbitMQ（消息队列）', dataType: '结构化', mode: '实时', target: '默认目录 / 示例库 / notification_events', status: '等待中', created: '2026-05-25 15:30', lastRun: '实时', schedule: '常驻消费', totalRuns: '—', totalRows: '42,000', totalSize: '18 MB', syncStrategy: 'incremental', incremental: { field: 'queue offset', lookback: '5 分钟' }, schemaMeta: 'queue: moi.notifications · exchange: notify.direct',
        schema: [
          { name: 'event_id',   type: 'VARCHAR(36)',  pk: true,  nullable: false, desc: 'UUID' },
          { name: 'event_type', type: 'VARCHAR(64)',  pk: false, nullable: false, desc: '事件类型' },
          { name: 'target_user',type: 'VARCHAR(64)',  pk: false, desc: '通知对象' },
          { name: 'channel',    type: 'VARCHAR(16)',  pk: false, desc: 'sms / push / email' },
          { name: 'payload',    type: 'JSON',         pk: false, desc: '事件 payload' },
          { name: 'enqueued_at',type: 'TIMESTAMP(3)', pk: false, nullable: false, desc: '入队时间' }
        ]},

      // ---------- 企业应用 ----------
      't_demo_sap': { name: '【示例】SAP S/4HANA 销售订单同步', connector: '武汉新芯 · SAP S/4HANA', connType: 'SAP S/4HANA（企业应用）', dataType: '结构化', mode: '周期性', target: '默认目录 / 示例库 / sap_sales_orders', status: '等待中', created: '2026-05-25 15:35', lastRun: '2026-05-25 04:30', schedule: '每天 04:30', totalRuns: 30, totalRows: '184,000', totalSize: '68 MB', syncStrategy: 'incremental', incremental: { field: 'LAST_CHANGE_DATE', lookback: '1 天' }, schemaMeta: 'OData 服务 /sap/opu/odata/sap/API_SALES_ORDER_SRV/A_SalesOrder',
        schema: [
          { name: 'SalesOrder',          type: 'VARCHAR(10)', pk: true,  nullable: false, desc: '销售订单号' },
          { name: 'SalesOrderType',      type: 'VARCHAR(4)',  pk: false, desc: '订单类型' },
          { name: 'SoldToParty',         type: 'VARCHAR(10)', pk: false, desc: '售达方' },
          { name: 'TotalNetAmount',      type: 'DECIMAL(15,2)', pk: false, desc: '净额' },
          { name: 'TransactionCurrency', type: 'VARCHAR(5)',  pk: false, desc: '货币' },
          { name: 'CreationDate',        type: 'DATE',     pk: false, desc: '创建日期' },
          { name: 'LAST_CHANGE_DATE',    type: 'TIMESTAMP', pk: false, nullable: false, desc: '增量字段' }
        ]},
      't_demo_oracle_erp': { name: '【示例】Oracle ERP Cloud GL 同步', connector: 'Oracle ERP Cloud · 集团', connType: 'Oracle ERP Cloud（企业应用）', dataType: '结构化', mode: '周期性', target: '默认目录 / 示例库 / erp_gl_balances', status: '等待中', created: '2026-05-25 15:40', lastRun: '2026-05-25 05:00', schedule: '每天 05:00', totalRuns: 30, totalRows: '36,800', totalSize: '14 MB', syncStrategy: 'incremental', incremental: { field: 'LastUpdateDate', lookback: '1 天' }, schemaMeta: 'REST API /fscmRestApi/resources/11.13.18.05/generalLedgerBalances',
        schema: [
          { name: 'BalanceId',      type: 'NUMBER(15)', pk: true,  nullable: false, desc: '余额 ID' },
          { name: 'LedgerId',       type: 'NUMBER(15)', pk: false, desc: '账簿 ID' },
          { name: 'AccountingPeriod',type: 'VARCHAR(15)', pk: false, desc: '会计期' },
          { name: 'Currency',       type: 'VARCHAR(15)', pk: false, desc: '货币' },
          { name: 'BeginBalanceDr', type: 'NUMBER',     pk: false, desc: '期初借方' },
          { name: 'PeriodNetActivityCr', type: 'NUMBER', pk: false, desc: '本期贷方发生额' },
          { name: 'LastUpdateDate', type: 'TIMESTAMP',  pk: false, nullable: false, desc: '增量字段' }
        ]},
      't_demo_u8': { name: '【示例】用友 U8 销售订单同步', connector: '芯导科技 · 用友 U8', connType: '用友 U8（企业应用）', dataType: '结构化', mode: '周期性', target: '默认目录 / 示例库 / u8_sale_orders', status: '等待中', created: '2026-05-25 15:45', lastRun: '2026-05-25 01:00', schedule: '每天 01:00', totalRuns: 30, totalRows: '4,200', totalSize: '2.8 MB', syncStrategy: 'incremental', incremental: { field: 'cmodifytime', lookback: '1 天' }, schemaMeta: '账套 001 / 2026 年度 · 通过 U8 OpenAPI 拉取销售订单主表 SO_SOMain',
        schema: [
          { name: 'cSOCode',     type: 'VARCHAR(30)', pk: true,  nullable: false, desc: '销售订单号' },
          { name: 'cCusCode',    type: 'VARCHAR(20)', pk: false, nullable: false, desc: '客户编码' },
          { name: 'dDate',       type: 'DATE',        pk: false, desc: '订单日期' },
          { name: 'iTotal',      type: 'DECIMAL(20,4)', pk: false, desc: '订单总金额' },
          { name: 'cMaker',      type: 'VARCHAR(20)', pk: false, desc: '制单人' },
          { name: 'cmodifytime', type: 'DATETIME',    pk: false, nullable: false, desc: '增量字段' }
        ]},
      't_demo_kingdee': { name: '【示例】金蝶 K3 物料主数据同步', connector: '集团 · 金蝶 K/3 Cloud', connType: '金蝶 K/3 Cloud（企业应用）', dataType: '结构化', mode: '周期性', target: '默认目录 / 示例库 / kingdee_materials', status: '等待中', created: '2026-05-25 15:50', lastRun: '2026-05-25 03:00', schedule: '每天 03:00', totalRuns: 30, totalRows: '28,400', totalSize: '8.6 MB', syncStrategy: 'incremental', incremental: { field: 'FModifyDate', lookback: '1 天' }, schemaMeta: '通过 WebAPI ExecuteBillQuery 查询 BD_MATERIAL',
        schema: [
          { name: 'FMaterialId', type: 'BIGINT',  pk: true,  nullable: false, desc: '物料内码' },
          { name: 'FNumber',     type: 'VARCHAR(30)', pk: false, nullable: false, desc: '物料编码' },
          { name: 'FName',       type: 'VARCHAR(255)', pk: false, desc: '物料名称' },
          { name: 'FSpecification', type: 'VARCHAR(255)', pk: false, desc: '规格型号' },
          { name: 'FBaseUnitId', type: 'BIGINT',  pk: false, desc: '基本计量单位' },
          { name: 'FModifyDate', type: 'DATETIME', pk: false, nullable: false, desc: '增量字段' }
        ]},
      't_demo_salesforce': { name: '【示例】Salesforce Opportunity 同步', connector: '销售 · Salesforce', connType: 'Salesforce（企业应用）', dataType: '结构化', mode: '周期性', target: '默认目录 / 示例库 / sf_opportunity', status: '载入失败', created: '2026-05-25 15:55', lastRun: '2026-05-25 14:00', schedule: '每小时', totalRuns: 720, totalRows: '12,400', totalSize: '4.2 MB', syncStrategy: 'incremental', incremental: { field: 'LastModifiedDate', lookback: '2 小时' }, schemaMeta: 'SOQL: SELECT ... FROM Opportunity WHERE LastModifiedDate > $watermark',
        schema: [
          { name: 'Id',                type: 'VARCHAR(18)', pk: true,  nullable: false, desc: 'Salesforce 18 位 ID' },
          { name: 'Name',              type: 'VARCHAR(120)', pk: false, nullable: false, desc: '商机名称' },
          { name: 'AccountId',         type: 'VARCHAR(18)', pk: false, desc: '关联客户' },
          { name: 'StageName',         type: 'VARCHAR(40)', pk: false, desc: '阶段' },
          { name: 'Amount',            type: 'DECIMAL(16,2)', pk: false, desc: '金额' },
          { name: 'CloseDate',         type: 'DATE',     pk: false, desc: '预计成交日' },
          { name: 'Probability',       type: 'DECIMAL(5,2)', pk: false, desc: '赢率 %' },
          { name: 'LastModifiedDate',  type: 'TIMESTAMP', pk: false, nullable: false, desc: '增量字段' }
        ]},

      // ---------- 协作平台 ----------
      't_demo_feishu': { name: '【示例】飞书群消息载入', connector: '团队飞书', connType: '飞书（协作平台）', dataType: '结构化', mode: '周期性', target: '默认目录 / 示例库 / feishu_messages', status: '等待中', created: '2026-05-25 16:00', lastRun: '2026-05-25 14:00', schedule: '每 10 分钟', totalRuns: 4320, totalRows: '186,000', totalSize: '82 MB', syncStrategy: 'incremental', incremental: { field: 'create_time', lookback: '30 分钟' }, schemaMeta: 'open-apis/im/v1/messages · 按 chat_id 过滤白名单群组',
        schema: [
          { name: 'message_id',  type: 'VARCHAR(64)', pk: true, nullable: false, desc: '飞书消息 ID' },
          { name: 'chat_id',     type: 'VARCHAR(64)', pk: false, nullable: false, desc: '群 ID' },
          { name: 'sender_id',   type: 'VARCHAR(64)', pk: false, desc: '发送者 open_id' },
          { name: 'msg_type',    type: 'VARCHAR(16)', pk: false, desc: 'text / image / file / post' },
          { name: 'content',     type: 'JSON',     pk: false, desc: '消息内容（嵌套）' },
          { name: 'create_time', type: 'TIMESTAMP', pk: false, nullable: false, desc: '发送时间（增量字段）' }
        ]},
      't_demo_dingtalk': { name: '【示例】钉钉审批数据同步', connector: '集团钉钉', connType: '钉钉（协作平台）', dataType: '结构化', mode: '周期性', target: '默认目录 / 示例库 / ding_approvals', status: '等待中', created: '2026-05-25 16:05', lastRun: '2026-05-25 14:00', schedule: '每小时', totalRuns: 720, totalRows: '38,200', totalSize: '14 MB', syncStrategy: 'incremental', incremental: { field: 'finish_time', lookback: '2 小时' }, schemaMeta: 'topapi/processinstance/listids + 详情拉取',
        schema: [
          { name: 'process_instance_id', type: 'VARCHAR(64)', pk: true, nullable: false, desc: '审批实例 ID' },
          { name: 'process_code',        type: 'VARCHAR(64)', pk: false, nullable: false, desc: '审批模板编码' },
          { name: 'title',               type: 'VARCHAR(256)', pk: false, desc: '标题' },
          { name: 'originator_userid',   type: 'VARCHAR(64)', pk: false, desc: '发起人 ID' },
          { name: 'result',              type: 'VARCHAR(16)', pk: false, desc: 'agree / refuse' },
          { name: 'form_values',         type: 'JSON',     pk: false, desc: '表单字段（嵌套）' },
          { name: 'finish_time',         type: 'TIMESTAMP', pk: false, nullable: false, desc: '完成时间（增量字段）' }
        ]},
      't_demo_wecom_api': { name: '【示例】企业微信通讯录同步', connector: '企业微信项目通知 API', connType: '企业微信 API（协作平台）', dataType: '结构化', mode: '周期性', target: '默认目录 / 示例库 / wecom_users', status: '等待中', created: '2026-05-25 16:10', lastRun: '2026-05-25 08:00', schedule: '每天 08:00', totalRuns: 30, totalRows: '4,800', totalSize: '1.6 MB', syncStrategy: 'full', schemaMeta: 'cgi-bin/user/simplelist + user/get；每日全量刷新通讯录',
        schema: [
          { name: 'userid',       type: 'VARCHAR(64)', pk: true, nullable: false, desc: '企业微信用户 ID' },
          { name: 'name',         type: 'VARCHAR(64)', pk: false, desc: '姓名' },
          { name: 'department',   type: 'JSON',    pk: false, desc: '所属部门 ID 数组' },
          { name: 'position',     type: 'VARCHAR(64)', pk: false, desc: '职位' },
          { name: 'email',        type: 'VARCHAR(128)', pk: false, desc: '邮箱' },
          { name: 'mobile',       type: 'VARCHAR(32)', pk: false, desc: '手机（已脱敏）' },
          { name: 'is_active',    type: 'BOOLEAN', pk: false, desc: '是否激活' }
        ]},
      't_demo_slack': { name: '【示例】Slack 频道消息载入', connector: '海外团队 · Slack', connType: 'Slack（协作平台）', dataType: '结构化', mode: '周期性', target: '默认目录 / 示例库 / slack_messages', status: '等待中', created: '2026-05-25 16:15', lastRun: '2026-05-25 14:00', schedule: '每 10 分钟', totalRuns: 4320, totalRows: '128,400', totalSize: '52 MB', syncStrategy: 'incremental', incremental: { field: 'ts', lookback: '30 分钟' }, schemaMeta: 'conversations.history · 监听白名单 channel 列表',
        schema: [
          { name: 'ts',          type: 'VARCHAR(32)', pk: true, nullable: false, desc: 'Slack 消息时间戳（即唯一 ID）' },
          { name: 'channel_id',  type: 'VARCHAR(16)', pk: true, nullable: false, desc: '频道 ID' },
          { name: 'user_id',     type: 'VARCHAR(16)', pk: false, desc: '发送者' },
          { name: 'text',        type: 'TEXT',     pk: false, desc: '消息文本' },
          { name: 'thread_ts',   type: 'VARCHAR(32)', pk: false, desc: '所属 thread' },
          { name: 'has_files',   type: 'BOOLEAN',  pk: false, desc: '是否含附件' },
          { name: 'reactions',   type: 'JSON',     pk: false, desc: '表情反馈列表' }
        ]},
      't_demo_github': { name: '【示例】GitHub Issue 同步（matrixflow）', connector: 'matrixflow · GitHub', connType: 'GitHub（协作平台）', dataType: '结构化', mode: '周期性', target: '默认目录 / 示例库 / github_issues', status: '等待中', created: '2026-05-25 16:20', lastRun: '2026-05-25 14:00', schedule: '每小时', totalRuns: 720, totalRows: '12,600', totalSize: '32 MB', syncStrategy: 'incremental', incremental: { field: 'updated_at', lookback: '2 小时' }, schemaMeta: 'GET /repos/matrixorigin/matrixflow/issues?since=$watermark',
        schema: [
          { name: 'issue_id',    type: 'BIGINT',  pk: true, nullable: false, desc: 'GitHub Issue 全局 ID' },
          { name: 'number',      type: 'INT',     pk: false, nullable: false, desc: '仓库内 issue 号' },
          { name: 'title',       type: 'VARCHAR(512)', pk: false, desc: '标题' },
          { name: 'state',       type: 'VARCHAR(16)', pk: false, desc: 'open / closed' },
          { name: 'author',      type: 'VARCHAR(64)', pk: false, desc: '作者 login' },
          { name: 'labels',      type: 'JSON',    pk: false, desc: '标签数组' },
          { name: 'body',        type: 'TEXT',    pk: false, desc: '正文 markdown' },
          { name: 'created_at',  type: 'TIMESTAMP', pk: false, nullable: false, desc: '创建时间' },
          { name: 'updated_at',  type: 'TIMESTAMP', pk: false, nullable: false, desc: '最后更新（增量字段）' }
        ]},
      't_demo_yuque': { name: '【示例】语雀文档元数据同步', connector: '内部 · 语雀知识库', connType: '语雀（协作平台）', dataType: '结构化', mode: '周期性', target: '默认目录 / 示例库 / yuque_docs', status: '等待中', created: '2026-05-25 16:25', lastRun: '2026-05-25 06:00', schedule: '每 6 小时', totalRuns: 120, totalRows: '2,840', totalSize: '8.4 MB', syncStrategy: 'incremental', incremental: { field: 'updated_at', lookback: '12 小时' }, schemaMeta: 'GET /api/v2/repos/:namespace/docs · 仅同步元数据，正文按需取',
        schema: [
          { name: 'doc_id',     type: 'BIGINT',  pk: true, nullable: false, desc: '文档 ID' },
          { name: 'slug',       type: 'VARCHAR(128)', pk: false, nullable: false, desc: '文档 slug' },
          { name: 'title',      type: 'VARCHAR(256)', pk: false, desc: '标题' },
          { name: 'namespace',  type: 'VARCHAR(128)', pk: false, desc: '所属知识库' },
          { name: 'author',     type: 'VARCHAR(64)', pk: false, desc: '作者' },
          { name: 'word_count', type: 'INT',     pk: false, desc: '字数' },
          { name: 'tags',       type: 'JSON',    pk: false, desc: '标签数组' },
          { name: 'updated_at', type: 'TIMESTAMP', pk: false, nullable: false, desc: '增量字段' }
        ]}
    };

window.IMPORT_TASKS_EDIT = {
      // ---- NESR-湖仓项目 实际生产载入任务（7 个：1 Intelie MongoDB + 6 Fiix CMMS REST API） ----
      // Intelie MongoDB（源端聚合：秒→分钟，仅传结果）
      't_nesr_intelie_sensor':   { name: '【NESR-湖仓项目】Intelie 传感器数据同步', dataType: 'structured', sourceType: 'connector', connectorValue: 'mongodb', loadMode: 'periodic', stLoadMode: 'periodic', stPeriodicInterval: '5m', dbName: 'intelie_prod', dbTable: 'fleet_sensor_1s', sourcePreprocess: 'aggregation', target: { dir: 'NESR', db: 'Bronze', table: 'intelie_sensor_readings' }, targetMode: 'existing', syncStrategy: 'incremental', incremental: { field: 'datetime', lookback: '10m' }, backfill: true },
      // Fiix CMMS REST API（变化型业务表 → 增量；字典 → 全量）
      't_nesr_fiix_work_orders': { name: '【NESR-湖仓项目】Fiix 工单数据同步',      dataType: 'structured', sourceType: 'connector', connectorValue: 'rest-api', loadMode: 'periodic', stLoadMode: 'periodic', stPeriodicInterval: 'hourly', apiEndpoint: '/api/v5/WorkOrder',        target: { dir: 'NESR', db: 'Bronze', table: 'fiix_work_orders' }, targetMode: 'existing', syncStrategy: 'incremental', incremental: { field: 'dtm_date_last_modified', lookback: '1h' }, backfill: true },
      't_nesr_fiix_wo_tasks':    { name: '【NESR-湖仓项目】Fiix 工单任务同步',      dataType: 'structured', sourceType: 'connector', connectorValue: 'rest-api', loadMode: 'periodic', stLoadMode: 'periodic', stPeriodicInterval: 'hourly', apiEndpoint: '/api/v5/WorkOrderTask',    target: { dir: 'NESR', db: 'Bronze', table: 'fiix_wo_tasks' },    targetMode: 'existing', syncStrategy: 'incremental', incremental: { field: 'dtm_date_last_modified', lookback: '1h' }, backfill: true },
      't_nesr_fiix_assets':      { name: '【NESR-湖仓项目】Fiix 资产层级同步',      dataType: 'structured', sourceType: 'connector', connectorValue: 'rest-api', loadMode: 'periodic', stLoadMode: 'periodic', stPeriodicInterval: 'daily',  apiEndpoint: '/api/v5/Asset',            target: { dir: 'NESR', db: 'Bronze', table: 'fiix_assets' },      targetMode: 'existing', syncStrategy: 'full' },
      't_nesr_fiix_priorities':  { name: '【NESR-湖仓项目】Fiix 优先级字典同步',    dataType: 'structured', sourceType: 'connector', connectorValue: 'rest-api', loadMode: 'periodic', stLoadMode: 'periodic', stPeriodicInterval: 'daily',  apiEndpoint: '/api/v5/Priority',         target: { dir: 'NESR', db: 'Bronze', table: 'fiix_priorities' },  targetMode: 'existing', syncStrategy: 'full' },
      't_nesr_fiix_wo_statuses':    { name: '【NESR-湖仓项目】Fiix 状态字典同步',      dataType: 'structured', sourceType: 'connector', connectorValue: 'rest-api', loadMode: 'periodic', stLoadMode: 'periodic', stPeriodicInterval: 'daily',  apiEndpoint: '/api/v5/WorkOrderStatus',  target: { dir: 'NESR', db: 'Bronze', table: 'fiix_wo_statuses' },    targetMode: 'existing', syncStrategy: 'full' },
      't_nesr_fiix_maintenance_types': { name: '【NESR-湖仓项目】Fiix 维修类型字典同步',  dataType: 'structured', sourceType: 'connector', connectorValue: 'rest-api', loadMode: 'periodic', stLoadMode: 'periodic', stPeriodicInterval: 'daily',  apiEndpoint: '/api/v5/MaintenanceType',  target: { dir: 'NESR', db: 'Bronze', table: 'fiix_maintenance_types' }, targetMode: 'existing', syncStrategy: 'full' },

      // ============ 【示例】每种连接器类型一个示例载入任务 ============
      // 对象存储 / 分布式文件系统：non-structured
      't_demo_aliyun_oss':  { name: '【示例】阿里云 OSS 营销素材同步', dataType: 'unstructured', sourceType: 'connector', connectorValue: 'aliyun-oss',  loadMode: 'periodic', stLoadMode: 'periodic', stPeriodicInterval: 'hourly', target: { dir: '默认目录', db: '示例库', volume: '营销素材卷' }, targetMode: 'existing', syncStrategy: 'incremental', incremental: { field: '对象 LastModified', lookback: '1h' } },
      't_demo_s3':          { name: '【示例】AWS S3 数据归档（双向）',  dataType: 'unstructured', sourceType: 'connector', connectorValue: 's3',          loadMode: 'periodic', stLoadMode: 'periodic', stPeriodicInterval: 'daily',  target: { dir: '默认目录', db: '示例库', volume: 'NESR 备份卷' }, targetMode: 'existing', syncStrategy: 'incremental', incremental: { field: 'S3 LastModified', lookback: '1d' } },
      't_demo_cos':         { name: '【示例】腾讯云 COS 课程视频载入', dataType: 'unstructured', sourceType: 'connector', connectorValue: 'tencent-cos', loadMode: 'periodic', stLoadMode: 'periodic', stPeriodicInterval: 'daily',  target: { dir: '默认目录', db: '示例库', volume: '视频素材卷' }, targetMode: 'existing', syncStrategy: 'incremental', incremental: { field: '对象 LastModified', lookback: '1d' } },
      't_demo_obs':         { name: '【示例】华为云 OBS 合规报告归档', dataType: 'unstructured', sourceType: 'connector', connectorValue: 'huawei-obs',  loadMode: 'periodic', stLoadMode: 'periodic', stPeriodicInterval: '7d',     target: { dir: '默认目录', db: '示例库', volume: '合规归档卷' }, targetMode: 'existing', syncStrategy: 'full' },
      't_demo_hdfs':        { name: '【示例】HDFS 行为日志载入',       dataType: 'unstructured', sourceType: 'connector', connectorValue: 'hdfs',        loadMode: 'periodic', stLoadMode: 'periodic', stPeriodicInterval: 'hourly', target: { dir: '默认目录', db: '示例库', volume: '行为日志卷' }, targetMode: 'existing', syncStrategy: 'incremental', incremental: { field: '分区 dt/hr', lookback: '2h' }, backfill: true },

      // 数据库类（structured）
      't_demo_matrixone':   { name: '【示例】MatrixOne 用户行为表同步',  dataType: 'structured', sourceType: 'connector', connectorValue: 'matrixone',  loadMode: 'periodic', stLoadMode: 'periodic', stPeriodicInterval: 'hourly', dbName: 'moi_warehouse', dbTable: 'user_events',          target: { dir: '默认目录', db: '示例库', table: 'user_events' },        targetMode: 'new', syncStrategy: 'incremental', incremental: { field: 'event_time', lookback: '1h' } },
      't_demo_mysql':       { name: '【示例】MySQL 用户中心同步',          dataType: 'structured', sourceType: 'connector', connectorValue: 'mysql',      loadMode: 'periodic', stLoadMode: 'periodic', stPeriodicInterval: 'hourly', dbName: 'user_center',   dbTable: 'users',                target: { dir: '默认目录', db: '示例库', table: 'users' },              targetMode: 'new', syncStrategy: 'incremental', incremental: { field: 'updated_at', lookback: '1h' } },
      't_demo_pg':          { name: '【示例】PostgreSQL 财务汇总同步',     dataType: 'structured', sourceType: 'connector', connectorValue: 'postgresql', loadMode: 'periodic', stLoadMode: 'periodic', stPeriodicInterval: 'daily',  dbName: 'finance_dw',    dbTable: 'daily_financial_summary', target: { dir: '默认目录', db: '示例库', table: 'financial_summary' }, targetMode: 'new', syncStrategy: 'incremental', incremental: { field: 'report_date', lookback: '7d' } },
      't_demo_mongodb':     { name: '【示例】MongoDB 商品目录同步',       dataType: 'structured', sourceType: 'connector', connectorValue: 'mongodb',    loadMode: 'periodic', stLoadMode: 'periodic', stPeriodicInterval: 'hourly', dbName: 'shop_prod',     dbTable: 'products',             target: { dir: '默认目录', db: '示例库', table: 'products' },           targetMode: 'new', syncStrategy: 'incremental', incremental: { field: 'updatedAt', lookback: '1h' } },
      't_demo_hive':        { name: '【示例】Hive DWD 周报加工',         dataType: 'structured', sourceType: 'connector', connectorValue: 'hive',       loadMode: 'periodic', stLoadMode: 'periodic', stPeriodicInterval: '7d',     dbName: 'nesr_dw_dwd',   dbTable: 'dwd_well_kpi_hourly',  target: { dir: '默认目录', db: '示例库', table: 'dwd_weekly_kpi' },     targetMode: 'new', syncStrategy: 'incremental', incremental: { field: 'partition stat_week', lookback: '0' } },
      't_demo_sqlserver':   { name: '【示例】SQL Server 报价单同步',     dataType: 'structured', sourceType: 'connector', connectorValue: 'sqlserver',  loadMode: 'periodic', stLoadMode: 'periodic', stPeriodicInterval: '30m',    dbName: 'TopcastQuote',  dbTable: 'QUOTE_HDR',            target: { dir: '默认目录', db: '示例库', table: 'quote_orders' },       targetMode: 'new', syncStrategy: 'incremental', incremental: { field: 'LastModified', lookback: '30m' } },
      't_demo_oracle':      { name: '【示例】Oracle ERP 总账分录同步',   dataType: 'structured', sourceType: 'connector', connectorValue: 'oracle',     loadMode: 'periodic', stLoadMode: 'periodic', stPeriodicInterval: 'daily',  dbName: 'ERPPDB1',       dbTable: 'GL_JE_LINES',          target: { dir: '默认目录', db: '示例库', table: 'gl_je_lines' },        targetMode: 'new', syncStrategy: 'incremental', incremental: { field: 'LAST_UPDATE_DATE', lookback: '1d' } },
      't_demo_clickhouse':  { name: '【示例】ClickHouse 港股 Tick 同步', dataType: 'structured', sourceType: 'connector', connectorValue: 'clickhouse', loadMode: 'periodic', stLoadMode: 'periodic', stPeriodicInterval: '1m',     dbName: 'hkex_market',   dbTable: 'tick',                 target: { dir: '默认目录', db: '示例库', table: 'hkex_tick' },          targetMode: 'new', syncStrategy: 'incremental', incremental: { field: 'tick_time', lookback: '5m' }, backfill: true },
      't_demo_doris':       { name: '【示例】Doris 实时指标同步',         dataType: 'structured', sourceType: 'connector', connectorValue: 'doris',      loadMode: 'periodic', stLoadMode: 'periodic', stPeriodicInterval: '5m',     dbName: 'rt_analytics',  dbTable: 'minute_metrics',       target: { dir: '默认目录', db: '示例库', table: 'rt_metrics' },         targetMode: 'new', syncStrategy: 'incremental', incremental: { field: 'stat_time', lookback: '15m' } },

      // 邮件类
      't_demo_gmail':            { name: '【示例】Gmail 客户邮件载入',         dataType: 'structured', sourceType: 'connector', connectorValue: 'gmail',           loadMode: 'periodic', stLoadMode: 'periodic', stPeriodicInterval: '5m',  target: { dir: '默认目录', db: '示例库', table: 'gmail_messages' },   targetMode: 'new', syncStrategy: 'incremental', incremental: { field: 'internalDate', lookback: '15m' } },
      't_demo_outlook':          { name: '【示例】Outlook 法务邮件载入',       dataType: 'structured', sourceType: 'connector', connectorValue: 'outlook',         loadMode: 'periodic', stLoadMode: 'periodic', stPeriodicInterval: '15m', target: { dir: '默认目录', db: '示例库', table: 'outlook_messages' }, targetMode: 'new', syncStrategy: 'incremental', incremental: { field: 'receivedDateTime', lookback: '30m' } },
      't_demo_ms365_graph':      { name: '【示例】Microsoft 365 全员邮件采集', dataType: 'structured', sourceType: 'connector', connectorValue: 'ms365-graph',     loadMode: 'periodic', stLoadMode: 'periodic', stPeriodicInterval: '5m',  target: { dir: '默认目录', db: '示例库', table: 'ms365_messages' },   targetMode: 'new', syncStrategy: 'incremental', incremental: { field: 'receivedDateTime', lookback: '15m' }, backfill: true },
      't_demo_wecom_mail':       { name: '【示例】企业微信邮箱采集',           dataType: 'structured', sourceType: 'connector', connectorValue: 'wecom-mail',      loadMode: 'periodic', stLoadMode: 'periodic', stPeriodicInterval: '10m', target: { dir: '默认目录', db: '示例库', table: 'wecom_mail_messages' }, targetMode: 'new', syncStrategy: 'incremental', incremental: { field: 'mail_date', lookback: '30m' } },
      't_demo_qq_mail':          { name: '【示例】QQ 邮箱采集',                dataType: 'structured', sourceType: 'connector', connectorValue: 'qq-mail',         loadMode: 'periodic', stLoadMode: 'periodic', stPeriodicInterval: '10m', target: { dir: '默认目录', db: '示例库', table: 'qq_mail_messages' }, targetMode: 'new', syncStrategy: 'incremental', incremental: { field: 'mail_date', lookback: '30m' } },
      't_demo_imap':             { name: '【示例】内部 IMAP 邮箱采集',         dataType: 'structured', sourceType: 'connector', connectorValue: 'imap-smtp',       loadMode: 'periodic', stLoadMode: 'periodic', stPeriodicInterval: 'hourly', target: { dir: '默认目录', db: '示例库', table: 'internal_mail' }, targetMode: 'new', syncStrategy: 'incremental', incremental: { field: 'received_at', lookback: '2h' } },
      't_demo_custom_mail_api':  { name: '【示例】自有邮件 API 同步',          dataType: 'structured', sourceType: 'connector', connectorValue: 'custom-mail-api', loadMode: 'periodic', stLoadMode: 'periodic', stPeriodicInterval: '5m',  target: { dir: '默认目录', db: '示例库', table: 'custom_mail' },   targetMode: 'new', syncStrategy: 'incremental', incremental: { field: 'sent_at', lookback: '15m' } },

      // API
      't_demo_rest_api':    { name: '【示例】天气 API 拉取',          dataType: 'structured', sourceType: 'connector', connectorValue: 'rest-api', loadMode: 'periodic', stLoadMode: 'periodic', stPeriodicInterval: 'hourly', apiEndpoint: '/weather', target: { dir: '默认目录', db: '示例库', table: 'weather_hourly' }, targetMode: 'new', syncStrategy: 'incremental', incremental: { field: 'obs_time', lookback: '2h' } },
      't_demo_graphql':     { name: '【示例】GraphQL 电商订单同步',  dataType: 'structured', sourceType: 'connector', connectorValue: 'graphql',  loadMode: 'periodic', stLoadMode: 'periodic', stPeriodicInterval: '15m',    apiEndpoint: 'query orders', target: { dir: '默认目录', db: '示例库', table: 'shop_orders' }, targetMode: 'new', syncStrategy: 'incremental', incremental: { field: 'updatedAt', lookback: '30m' } },

      // 文件协议（unstructured）
      't_demo_ftp':         { name: '【示例】FTP 现场日报载入',      dataType: 'unstructured', sourceType: 'connector', connectorValue: 'ftp',        loadMode: 'periodic', stLoadMode: 'periodic', stPeriodicInterval: 'daily',  target: { dir: '默认目录', db: '示例库', volume: '现场日报卷' }, targetMode: 'existing', syncStrategy: 'incremental', incremental: { field: '文件 mtime', lookback: '1d' } },
      't_demo_sftp':        { name: '【示例】SFTP 客户合同同步',     dataType: 'unstructured', sourceType: 'connector', connectorValue: 'sftp',       loadMode: 'periodic', stLoadMode: 'periodic', stPeriodicInterval: 'daily',  target: { dir: '默认目录', db: '示例库', volume: '合同档案卷' }, targetMode: 'existing', syncStrategy: 'incremental', incremental: { field: '文件 mtime', lookback: '1d' } },
      't_demo_sharepoint':  { name: '【示例】SharePoint 问卷文档载入',dataType: 'unstructured', sourceType: 'connector', connectorValue: 'sharepoint', loadMode: 'periodic', stLoadMode: 'periodic', stPeriodicInterval: '6h',     target: { dir: '默认目录', db: '示例库', volume: '问卷文档卷' }, targetMode: 'existing', syncStrategy: 'incremental', incremental: { field: 'Modified', lookback: '6h' } },
      't_demo_smb':         { name: '【示例】NAS 部门资料同步',      dataType: 'unstructured', sourceType: 'connector', connectorValue: 'smb-nas',    loadMode: 'periodic', stLoadMode: 'periodic', stPeriodicInterval: 'hourly', target: { dir: '默认目录', db: '示例库', volume: '部门资料卷' }, targetMode: 'existing', syncStrategy: 'incremental', incremental: { field: '文件 mtime', lookback: '2h' } },

      // 消息队列（structured + 实时）
      't_demo_kafka':       { name: '【示例】Kafka GPS 流实时摄取',  dataType: 'structured', sourceType: 'connector', connectorValue: 'kafka',    loadMode: 'realtime', stLoadMode: 'realtime', apiEndpoint: 'taxi.gps.*', target: { dir: '默认目录', db: '示例库', table: 'gps_events' },        targetMode: 'new', syncStrategy: 'incremental', incremental: { field: 'Kafka offset', lookback: '5m' } },
      't_demo_mqtt':        { name: '【示例】MQTT IoT 设备遥测',     dataType: 'structured', sourceType: 'connector', connectorValue: 'mqtt',     loadMode: 'realtime', stLoadMode: 'realtime', apiEndpoint: 'iot/+/telemetry', target: { dir: '默认目录', db: '示例库', table: 'iot_telemetry' },     targetMode: 'new', syncStrategy: 'incremental', incremental: { field: 'ts', lookback: '10m' } },
      't_demo_rabbitmq':    { name: '【示例】RabbitMQ 通知事件消费', dataType: 'structured', sourceType: 'connector', connectorValue: 'rabbitmq', loadMode: 'realtime', stLoadMode: 'realtime', apiEndpoint: 'moi.notifications', target: { dir: '默认目录', db: '示例库', table: 'notification_events' }, targetMode: 'new', syncStrategy: 'incremental', incremental: { field: 'queue offset', lookback: '5m' } },

      // 企业应用
      't_demo_sap':         { name: '【示例】SAP S/4HANA 销售订单同步',   dataType: 'structured', sourceType: 'connector', connectorValue: 'sap-s4hana', loadMode: 'periodic', stLoadMode: 'periodic', stPeriodicInterval: 'daily',  apiEndpoint: '/API_SALES_ORDER_SRV/A_SalesOrder', target: { dir: '默认目录', db: '示例库', table: 'sap_sales_orders' }, targetMode: 'new', syncStrategy: 'incremental', incremental: { field: 'LAST_CHANGE_DATE', lookback: '1d' } },
      't_demo_oracle_erp':  { name: '【示例】Oracle ERP Cloud GL 同步',   dataType: 'structured', sourceType: 'connector', connectorValue: 'oracle-erp', loadMode: 'periodic', stLoadMode: 'periodic', stPeriodicInterval: 'daily',  apiEndpoint: '/generalLedgerBalances',            target: { dir: '默认目录', db: '示例库', table: 'erp_gl_balances' }, targetMode: 'new', syncStrategy: 'incremental', incremental: { field: 'LastUpdateDate', lookback: '1d' } },
      't_demo_u8':          { name: '【示例】用友 U8 销售订单同步',       dataType: 'structured', sourceType: 'connector', connectorValue: 'yonyou-u8',  loadMode: 'periodic', stLoadMode: 'periodic', stPeriodicInterval: 'daily',  dbName: '001(2026)', dbTable: 'SO_SOMain',                target: { dir: '默认目录', db: '示例库', table: 'u8_sale_orders' },     targetMode: 'new', syncStrategy: 'incremental', incremental: { field: 'cmodifytime', lookback: '1d' } },
      't_demo_kingdee':     { name: '【示例】金蝶 K3 物料主数据同步',     dataType: 'structured', sourceType: 'connector', connectorValue: 'kingdee-k3', loadMode: 'periodic', stLoadMode: 'periodic', stPeriodicInterval: 'daily',  apiEndpoint: 'BD_MATERIAL',                       target: { dir: '默认目录', db: '示例库', table: 'kingdee_materials' }, targetMode: 'new', syncStrategy: 'incremental', incremental: { field: 'FModifyDate', lookback: '1d' } },
      't_demo_salesforce':  { name: '【示例】Salesforce Opportunity 同步',dataType: 'structured', sourceType: 'connector', connectorValue: 'salesforce', loadMode: 'periodic', stLoadMode: 'periodic', stPeriodicInterval: 'hourly', apiEndpoint: 'SOQL Opportunity',                  target: { dir: '默认目录', db: '示例库', table: 'sf_opportunity' },   targetMode: 'new', syncStrategy: 'incremental', incremental: { field: 'LastModifiedDate', lookback: '2h' } },

      // 协作平台
      't_demo_feishu':      { name: '【示例】飞书群消息载入',        dataType: 'structured', sourceType: 'connector', connectorValue: 'feishu',   loadMode: 'periodic', stLoadMode: 'periodic', stPeriodicInterval: '10m',   apiEndpoint: '/im/v1/messages',     target: { dir: '默认目录', db: '示例库', table: 'feishu_messages' }, targetMode: 'new', syncStrategy: 'incremental', incremental: { field: 'create_time', lookback: '30m' } },
      't_demo_dingtalk':    { name: '【示例】钉钉审批数据同步',      dataType: 'structured', sourceType: 'connector', connectorValue: 'dingtalk', loadMode: 'periodic', stLoadMode: 'periodic', stPeriodicInterval: 'hourly', apiEndpoint: '/processinstance',    target: { dir: '默认目录', db: '示例库', table: 'ding_approvals' }, targetMode: 'new', syncStrategy: 'incremental', incremental: { field: 'finish_time', lookback: '2h' } },
      't_demo_wecom_api':   { name: '【示例】企业微信通讯录同步',    dataType: 'structured', sourceType: 'connector', connectorValue: 'wecom-api',loadMode: 'periodic', stLoadMode: 'periodic', stPeriodicInterval: 'daily',  apiEndpoint: '/cgi-bin/user/list',  target: { dir: '默认目录', db: '示例库', table: 'wecom_users' },    targetMode: 'new', syncStrategy: 'full' },
      't_demo_slack':       { name: '【示例】Slack 频道消息载入',    dataType: 'structured', sourceType: 'connector', connectorValue: 'slack',    loadMode: 'periodic', stLoadMode: 'periodic', stPeriodicInterval: '10m',   apiEndpoint: 'conversations.history', target: { dir: '默认目录', db: '示例库', table: 'slack_messages' }, targetMode: 'new', syncStrategy: 'incremental', incremental: { field: 'ts', lookback: '30m' } },
      't_demo_github':      { name: '【示例】GitHub Issue 同步（matrixflow）', dataType: 'structured', sourceType: 'connector', connectorValue: 'github', loadMode: 'periodic', stLoadMode: 'periodic', stPeriodicInterval: 'hourly', apiEndpoint: '/repos/matrixorigin/matrixflow/issues', target: { dir: '默认目录', db: '示例库', table: 'github_issues' }, targetMode: 'new', syncStrategy: 'incremental', incremental: { field: 'updated_at', lookback: '2h' } },
      't_demo_yuque':       { name: '【示例】语雀文档元数据同步',    dataType: 'structured', sourceType: 'connector', connectorValue: 'yuque',    loadMode: 'periodic', stLoadMode: 'periodic', stPeriodicInterval: '6h',    apiEndpoint: '/api/v2/repos/:ns/docs', target: { dir: '默认目录', db: '示例库', table: 'yuque_docs' },   targetMode: 'new', syncStrategy: 'incremental', incremental: { field: 'updated_at', lookback: '12h' } }
    };
})();
