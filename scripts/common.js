// === MOI i18n System ===
var MOI_LANG = localStorage.getItem('moi_lang') || 'zh';
var MOI_I18N = {
  '仪表盘':{en:'Dashboard'},'概览':{en:'Overview'},'数据看板':{en:'Dashboard'},'数据连接':{en:'Data Connection'},'连接器':{en:'Connectors'},'数据载入':{en:'Data Import'},'数据导出':{en:'Data Export'},'数据处理':{en:'Data Processing'},'工作流':{en:'Workflows'},'SQL 编辑器':{en:'SQL Editor'},'资源中心':{en:'Resources'},'计算资源':{en:'Compute'},'数据分享':{en:'Data Sharing'},'知识库':{en:'Knowledge Base'},'监测':{en:'Monitoring'},'SQL 历史':{en:'SQL History'},'作业':{en:'Jobs'},'日志':{en:'Logs'},'告警':{en:'Alerts'},'告警规则':{en:'Alert Rules'},'通知对象':{en:'Notification Target'},'告警记录':{en:'Alert History'},'用户权限':{en:'Access Control'},'用户管理':{en:'User Management'},'角色权限':{en:'Roles & Permissions'},
  '智能体':{en:'Agent'},'数据':{en:'Data'},'应用':{en:'Apps'},'默认工作区':{en:'Default Workspace'},'联系我们':{en:'Contact Us'},'请联系邮箱':{en:'Contact Email'},'查看文档':{en:'Documentation'},'语言':{en:'Language'},'时区':{en:'Timezone'},'账户':{en:'Account'},'用户':{en:'User'},'账号管理':{en:'Account Settings'},'访问凭据':{en:'Access Credentials'},'计费中心':{en:'Billing'},'退出登录':{en:'Sign Out'},
  '创建连接器':{en:'Create Connector'},'编辑连接器':{en:'Edit Connector'},'搜索连接器名称/创建人':{en:'Search connectors...'},'名称':{en:'Name'},'类型':{en:'Type'},'数据源':{en:'Data Source'},'用途':{en:'Purpose'},'创建时间':{en:'Created'},'操作':{en:'Actions'},'载入':{en:'Import'},'导出':{en:'Export'},'权限':{en:'Permissions'},'编辑':{en:'Edit'},'删除':{en:'Delete'},'对象存储':{en:'Object Storage'},'分布式文件系统':{en:'Distributed FS'},'数据库':{en:'Database'},'连接器名称':{en:'Connector Name'},'请输入连接器名称':{en:'Enter connector name'},'连接信息':{en:'Connection Info'},'连接测试':{en:'Test Connection'},'取 消':{en:'Cancel'},'创 建':{en:'Create'},'保 存':{en:'Save'},'认证方式':{en:'Authentication'},'主机':{en:'Host'},'端口':{en:'Port'},'用户名':{en:'Username'},'密码':{en:'Password'},
  '阿里云 OSS':{en:'Alibaba Cloud OSS'},'标准 S3':{en:'Standard S3'},'地区':{en:'Region'},'文件路径':{en:'File Path'},'NameNode 地址':{en:'NameNode Address'},'请输入用户名':{en:'Enter username'},'请输入密码':{en:'Enter password'},'请输入数据库名':{en:'Enter database name'},'请输入 Access Key':{en:'Enter Access Key'},'请输入 Secret Key':{en:'Enter Secret Key'},'请输入 Bucket 名称':{en:'Enter Bucket name'},'请输入AccessKey ID':{en:'Enter AccessKey ID'},'请输入AccessKey Secret':{en:'Enter AccessKey Secret'},'华东1 （杭州）':{en:'East China 1 (Hangzhou)'},'华东2 （上海）':{en:'East China 2 (Shanghai)'},'华北2 （北京）':{en:'North China 2 (Beijing)'},'华南1 （深圳）':{en:'South China 1 (Shenzhen)'},
  '请输入 HiveServer2 主机地址':{en:'Enter HiveServer2 host address'},'请输入端口号（默认：10000）':{en:'Enter port (default: 10000)'},'请输入 Hive 用户名':{en:'Enter Hive username'},'请输入 Hive 密码':{en:'Enter Hive password'},'请输入 MongoDB 主机地址':{en:'Enter MongoDB host address'},'用户名密码':{en:'Username & Password'},'无认证':{en:'No Auth'},'认证库':{en:'Auth Database'},'读取偏好':{en:'Read Preference'},'primary（主节点）':{en:'primary (Primary)'},'secondary（从节点）':{en:'secondary (Secondary)'},'secondaryPreferred（推荐）':{en:'secondaryPreferred (Recommended)'},'nearest（最近节点）':{en:'nearest (Nearest)'},
  'Key 名称':{en:'Key Name'},'Key 值':{en:'Key Value'},'放置位置':{en:'Location'},'Header（请求头）':{en:'Header'},'Query（URL 参数）':{en:'Query (URL Param)'},'请输入 API Key':{en:'Enter API Key'},'请输入 Bearer Token':{en:'Enter Bearer Token'},'请输入 Client ID':{en:'Enter Client ID'},'请输入 Client Secret':{en:'Enter Client Secret'},'超时时间':{en:'Timeout'},'默认请求头':{en:'Default Headers'},
  '+ 添加主机端口':{en:'+ Add Host & Port'},'Keytab 文件':{en:'Keytab File'},'Krb5 配置文件':{en:'Krb5 Config File'},'添加 Keytab 文件':{en:'Add Keytab file'},'添加 Krb5 文件':{en:'Add Krb5 file'},'选择文件':{en:'Select file'},
  'Intelie 传感器数据':{en:'Intelie Sensor Data'},'Fiix CMMS 维修系统':{en:'Fiix CMMS Maintenance'},
  '请输入连接器名称':{en:'Enter connector name'},'确定删除连接器「':{en:'Are you sure you want to delete connector "'},'」？\n\n删除后，使用该连接器的载入/导出任务将无法执行。':{en:'"?\n\nAfter deletion, import/export tasks using this connector will not work.'},'连接器「':{en:'Connector "'},'」已保存（模拟）':{en:'" saved (simulated)'},'」已删除（模拟）':{en:'" deleted (simulated)'},'连接测试中...\n\n（模拟）连接成功 ✅':{en:'Testing connection...\n\n(Simulated) Connection successful ✅'},'刷新':{en:'Refresh'},
  '载入数据':{en:'Import Data'},'新建载入任务':{en:'New Import Task'},'搜索任务名称/创建人':{en:'Search tasks...'},'任务':{en:'Task'},'任务名称':{en:'Task Name'},'载入类型':{en:'Import Type'},'载入模式':{en:'Import Mode'},'目标位置':{en:'Target'},'状态':{en:'Status'},'非结构化':{en:'Unstructured'},'结构化':{en:'Structured'},'一次性':{en:'One-time'},'周期性':{en:'Periodic'},'完成':{en:'Completed'},'运行中':{en:'Running'},'失败':{en:'Failed'},'详情':{en:'Details'},'重试':{en:'Retry'},
  '非结构化数据':{en:'Unstructured Data'},'结构化数据':{en:'Structured Data'},'连接器载入':{en:'Connector Import'},'本地上传':{en:'Local Upload'},'网页采集':{en:'Web Scraping'},'一次载入':{en:'One-time'},'周期载入':{en:'Periodic'},'实时更新':{en:'Real-time'},'载入位置':{en:'Target Location'},'选择已有表':{en:'Existing Table'},'新建表':{en:'New Table'},'表定义':{en:'Table Definition'},'表映射':{en:'Column Mapping'},'数据回填':{en:'Data Backfill'},'载入前预处理':{en:'Pre-processing'},'增量同步配置':{en:'Incremental Sync'},'增量字段':{en:'Incremental Field'},'回溯窗口':{en:'Lookback Window'},'初次载入规则':{en:'Initial Load Rule'},'在已有数据后追加':{en:'Append to existing'},'清空已有数据后追加':{en:'Truncate then append'},
  '工作流名称':{en:'Workflow Name'},'分支':{en:'Branches'},'优先级':{en:'Priority'},'最近运行':{en:'Last Run'},'已完成':{en:'Completed'},'待运行':{en:'Pending'},'已停止':{en:'Stopped'},'新建工作流':{en:'New Workflow'},
  '数据检查':{en:'Data Check'},'重命名':{en:'Rename'},'校验规则':{en:'Validation Rules'},'添加规则':{en:'Add Rule'},'校验结果':{en:'Results'},'运行检查':{en:'Run Check'},'非空':{en:'Not Null'},'唯一':{en:'Unique'},'范围':{en:'Range'},'枚举值':{en:'Enum'},'外键关系':{en:'Foreign Key'},'空值率':{en:'Null Rate'},'新鲜度':{en:'Freshness'},'自定义 SQL':{en:'Custom SQL'},'仅记录':{en:'Log Only'},'标记':{en:'Flag'},'分流':{en:'Split'},
  '载入任务详情':{en:'Import Task Details'},'连接器':{en:'Connector'},'连接器类型':{en:'Connector Type'},'数据类型':{en:'Data Type'},'调度周期':{en:'Schedule'},'累计运行':{en:'Total Runs'},'累计行数':{en:'Total Rows'},'累计大小':{en:'Total Size'},'文件类型':{en:'File Types'},'连接详情':{en:'Connection Details'},'错误信息':{en:'Error Info'},'次':{en:' runs'},'最近运行':{en:'Last Run'},
  '【NESR-湖仓项目】Intelie 传感器数据同步':{en:'[NESR-Lakehouse] Intelie Sensor Data Sync'},'【NESR-湖仓项目】Fiix 工单数据同步':{en:'[NESR-Lakehouse] Fiix Work Order Sync'},'【NESR-湖仓项目】Fiix 资产数据同步':{en:'[NESR-Lakehouse] Fiix Asset Data Sync'},
  '本地文件':{en:'Local File'},'S3（对象存储）':{en:'S3 (Object Storage)'},'OSS（对象存储）':{en:'OSS (Object Storage)'},'MySQL（数据库）':{en:'MySQL (Database)'},'Hive（数据仓库）':{en:'Hive (Data Warehouse)'},'MongoDB（数据库）':{en:'MongoDB (Database)'},
  'NESR / Silver / sensor_readings_1min':{en:'NESR / Silver / sensor_readings_1min'},'NESR / Bronze / work_orders':{en:'NESR / Bronze / work_orders'},'NESR / Bronze / assets':{en:'NESR / Bronze / assets'},
  '每天 04:00 AST':{en:'Daily 04:00 AST'},'每天 04:30 AST':{en:'Daily 04:30 AST'},
  '1 小时':{en:'1 hour'},'24 小时':{en:'24 hours'},'任务详情加载中...':{en:'Loading task details...'},
  'MongoDB 数据库':{en:'MongoDB Database'},'主键冲突处理':{en:'Primary Key Conflict'},'替换冲突行':{en:'Replace conflicting rows'},'日均新增':{en:'Daily New Rows'},'分页策略':{en:'Pagination Strategy'},'偏移量分页（offset + limit）':{en:'Offset pagination (offset + limit)'},'数据路径':{en:'Data Path'},'字段数':{en:'Field Count'},'Intelie 传感器数据 (MongoDB)':{en:'Intelie Sensor Data (MongoDB)'},'Fiix CMMS 维修系统 (REST API)':{en:'Fiix CMMS Maintenance (REST API)'},'增量字段：':{en:'Incremental Field: '},'回溯窗口：':{en:'Lookback Window: '},
  // Workflow
  '搜索工作流名称':{en:'Search workflows...'},'计算资源':{en:'Compute'},'调度方式':{en:'Schedule Type'},'手动触发':{en:'Manual'},'周期调度':{en:'Scheduled'},'数据触发':{en:'Data Trigger'},'高':{en:'High'},'中':{en:'Medium'},'低':{en:'Low'},'基础模式':{en:'Basic'},'专业模式':{en:'Advanced'},
  // Workflow edit
  '编辑工作流':{en:'Edit Workflow'},'未命名工作流':{en:'Untitled Workflow'},'主分支':{en:'Main Branch'},'对比分支':{en:'Compare Branches'},'新建分支':{en:'New Branch'},'分支列表':{en:'Branch List'},'分支对比':{en:'Branch Compare'},
  '优先级':{en:'Priority'},'通知':{en:'Notification'},'通知设置':{en:'Notification Settings'},'失败通知':{en:'Failure Notification'},'成功通知':{en:'Success Notification'},'通知方式':{en:'Notification Method'},'接收人':{en:'Recipients'},'关闭':{en:'Close'},'开启':{en:'On'},
  '站内消息':{en:'In-app Message'},'邮件':{en:'Email'},'企业微信':{en:'WeCom'},'飞书':{en:'Feishu'},'钉钉':{en:'DingTalk'},
  '保存草稿成功':{en:'Draft saved'},'工作流已提交运行':{en:'Workflow submitted for execution'},
  '添加节点':{en:'Add Node'},'系统节点':{en:'System Nodes'},'自定义节点':{en:'Custom Nodes'},'搜索节点...':{en:'Search nodes...'},'点击添加 · 拖拽到画布精确放置':{en:'Click to add · Drag to canvas for precise placement'},
  '选择数据源路径':{en:'Select Data Source Path'},'请选择':{en:'Select'},
  '卷 / 表':{en:'Volume / Table'},
  '删除分支':{en:'Delete Branch'},'同时删除该分支已处理的数据':{en:'Also delete processed data from this branch'},'删除后不可恢复，该分支的处理逻辑（节点和连线）将被永久移除。':{en:'This cannot be undone. The branch processing logic (nodes and connections) will be permanently removed.'},
  '新建分支':{en:'New Branch'},'分支名称':{en:'Branch Name'},'基于分支':{en:'Based on Branch'},'将复制所选分支的全部节点和连线':{en:'All nodes and connections from the selected branch will be copied'},'请输入分支名称':{en:'Enter branch name'},
  '数据 IO':{en:'Data IO'},'非结构化处理':{en:'Unstructured Processing'},'结构化处理':{en:'Structured Processing'},'编程':{en:'Coding'},'流程控制':{en:'Flow Control'},
  '暂无自定义节点':{en:'No custom nodes'},'使用 Python 处理节点创建后可保存为自定义节点':{en:'Create with Python Process node and save as custom node'},'没有匹配的节点':{en:'No matching nodes'},
  '数据源类型':{en:'Data Source Type'},'数据源路径':{en:'Data Source Path'},'文件范围':{en:'File Range'},'按文件类型':{en:'By File Type'},'按文件':{en:'By File'},'请选择数据源路径...':{en:'Select data source path...'},'请选择 Catalog 表...':{en:'Select Catalog table...'},
  '选择表':{en:'Select Table'},'表信息':{en:'Table Info'},'过滤条件':{en:'Filter Condition'},'输出变量格式':{en:'Output Variable Format'},'文件列表':{en:'File List'},
  '保存路径':{en:'Save Path'},'请选择保存路径...':{en:'Select save path...'},'保存状态':{en:'Save Status'},
  '图片解析':{en:'Image Parse'},'解析内容筛选':{en:'Content Filter'},'图片描述':{en:'Image Caption'},'语言':{en:'Language'},
  '算子版本':{en:'Operator Version'},'新版本可用':{en:'New version available'},'升级':{en:'Upgrade'},'更改版本':{en:'Change Version'},'更改算子版本':{en:'Change Operator Version'},'当前':{en:'Current'},'最新':{en:'Latest'},
  '文档':{en:'docs'},'图片':{en:'Images'},'音频':{en:'Audio'},'视频':{en:'Video'},'通用文档':{en:'General Doc'},'表格':{en:'Table'},
  '文本':{en:'Text'},'分段方式：':{en:'Chunking Method:'},'层级智能感知':{en:'Hierarchical Awareness'},'按标志符分段':{en:'By Separator'},'分段标志符':{en:'Separators'},'分段最大长度：':{en:'Max Chunk Length:'},'上下文重叠（字符）：':{en:'Context Overlap (chars):'},
  '2个换行符':{en:'2 newlines'},'1个换行符':{en:'1 newline'},'句号':{en:'Period'},'分号':{en:'Semicolon'},'逗号':{en:'Comma'},'空格':{en:'Space'},'+ 添加标志符':{en:'+ Add Separator'},
  '请选择文件类型...':{en:'Select file types...'},'搜索文件名或类型...':{en:'Search filename or type...'},
  '图片描述模型':{en:'Image Caption Model'},'降噪':{en:'Denoise'},'语音切片':{en:'Voice Slicing'},'最小静音间隔':{en:'Min Silence Gap'},'最大语音时长':{en:'Max Voice Duration'},'语音模型':{en:'Speech Model'},
  '嵌入模型':{en:'Embedding Model'},'批量大小':{en:'Batch Size'},
  'Python 代码':{en:'Python Code'},'格式化':{en:'Format'},'全屏':{en:'Fullscreen'},'输入表':{en:'Input Tables'},'输出表':{en:'Output Tables'},'输入参数':{en:'Input Params'},'输出参数':{en:'Output Params'},'从代码自动解析':{en:'Auto-parsed from code'},'超时时间':{en:'Timeout'},'运行调试':{en:'Run Debug'},'清除':{en:'Clear'},'运行结果':{en:'Run Result'},
  'SQL 语句':{en:'SQL Statement'},'预览结果':{en:'Preview Result'},'从 SQL 语句中自动识别':{en:'Auto-detected from SQL'},'未检测到':{en:'Not detected'},'前 10 行':{en:'First 10 rows'},
  '条件设置':{en:'Condition Settings'},'其他（默认分支）':{en:'Other (Default Branch)'},'不满足以上任何条件时走此分支':{en:'Routes here when no conditions above are met'},'+ 添加条件分支':{en:'+ Add Condition Branch'},'分支 ':{en:'Branch '},'条件匹配':{en:'Condition Match'},'其他':{en:'Other'},
  '提取范围':{en:'Extraction Scope'},'每份文件单独提取':{en:'Extract per file'},'所有文件一起提取':{en:'Extract all files together'},'提取模型':{en:'Extraction Model'},'大语言模型':{en:'LLM'},'多模态模型':{en:'Multimodal Model'},'对话（文本）':{en:'Chat (Text)'},'对话（多模态）':{en:'Chat (Multimodal)'},'嵌入（文本）':{en:'Embedding (Text)'},'嵌入（多模态）':{en:'Embedding (Multimodal)'},'文生图':{en:'Text-to-Image'},'文生视频':{en:'Text-to-Video'},'文转音':{en:'TTS'},'音转文':{en:'ASR'},'重排序':{en:'Rerank'},'信息提取格式':{en:'Extraction Format'},'表单配置':{en:'Form Config'},'JSON 配置':{en:'JSON Config'},'自定义':{en:'Custom'},'合同信息':{en:'Contract Info'},'财务报告（含表格）':{en:'Financial Report (with tables)'},'人才简历':{en:'Resume'},'发票信息':{en:'Invoice Info'},'+ 添加字段':{en:'+ Add Field'},
  '生成风格':{en:'Generation Style'},'参数配置项':{en:'Parameter Config'},'生成样本数':{en:'Sample Count'},'数据格式':{en:'Data Format'},'自定义格式':{en:'Custom Format'},'字段管理':{en:'Field Management'},'字段配置':{en:'Field Config'},'样例预览':{en:'Sample Preview'},
  '敏感信息打码':{en:'PII Masking'},'文本标准化':{en:'Text Normalization'},'特殊字符删除':{en:'Special Char Removal'},'特殊字符过滤':{en:'Special Char Filter'},'敏感词过滤':{en:'Sensitive Word Filter'},'数据去重':{en:'Deduplication'},'翻译':{en:'Translation'},'文本标准化':{en:'Text Normalization'},'繁体转简体':{en:'Traditional to Simplified'},
  '英文':{en:'English'},'中文':{en:'Chinese'},'日文':{en:'Japanese'},'韩文':{en:'Korean'},'法文':{en:'French'},'德文':{en:'German'},
  '活跃':{en:'Active'},'已完成':{en:'Completed'},'预览':{en:'Preview'},
  '至少需要 2 个分支才能对比':{en:'At least 2 branches are needed to compare'},
  '手动点击"运行"按钮触发，不设置自动调度。':{en:'Manually click "Run" to trigger. No automatic scheduling.'},'频率':{en:'Frequency'},'时间':{en:'Time'},'时区':{en:'Timezone'},'生效日期':{en:'Effective Date'},'每小时':{en:'Hourly'},'每天':{en:'Daily'},'每周':{en:'Weekly'},'每月':{en:'Monthly'},'自定义 Cron':{en:'Custom Cron'},
  '触发源':{en:'Trigger Source'},'监听节点':{en:'Watch Node'},'暂无数据读取节点':{en:'No data read nodes'},'该路径有新数据写入时自动触发工作流':{en:'Workflow triggers automatically when new data is written to this path'},
  '输入用户名或邮箱...':{en:'Enter username or email...'},
  '正在分析两个分支的数据处理差异...':{en:'Analyzing data processing differences between branches...'},'两个分支的配置完全相同，没有差异。':{en:'Both branches have identical configurations, no differences.'},'配置相同':{en:'Same config'},'项参数不同':{en:' params differ'},'仅存在于「':{en:'Only exists in "'},'处理管线不同：':{en:'Processing pipelines differ:'},'两个分支的处理管线相同（':{en:'Both branches have the same pipeline ('},'），差异在节点参数配置：':{en:'), differences are in node parameter configs:'},
  '连线：':{en:'Connections: '},' 条 / ':{en:' / '},' 条':{en:''},
  '没有识别到要添加的节点类型':{en:'Could not identify the node type to add'},'没有识别到要删除的节点':{en:'Could not identify the node to delete'},'画布上没有该类型的节点':{en:'No node of this type on canvas'},'已添加「':{en:'Added "'},'」节点':{en:'" node'},'已删除「':{en:'Deleted "'},'已将分段大小修改为 ':{en:'Changed chunk size to '},
  '我理解了你的需求，正在思考最佳方案...（模拟中）':{en:'I understand your request, thinking of the best approach... (simulated)'},'已按你的描述调整工作流':{en:'Workflow adjusted per your description'},'正在生成工作流...':{en:'Generating workflow...'},
  '共':{en:'Total'},'个文件':{en:' files'},'回退':{en:'Rollback'},
  '已添加':{en:'Added'},'节点':{en:'node'},'已删除':{en:'Deleted'},
  '全屏编辑器功能开发中':{en:'Fullscreen editor is under development'},
  '变更说明：':{en:'Changelog: '},'建议在新分支中测试后再应用到主分支。':{en:'Recommend testing in a new branch before applying to main.'},'将恢复到该版本的处理逻辑。':{en:'Will restore to this version\'s processing logic.'},
  '开始':{en:'Start'},'结束':{en:'End'},'数据读取':{en:'Data Read'},'数据保存':{en:'Data Save'},'文档解析':{en:'Doc Parse'},'图片解析':{en:'Image Parse'},'音频解析':{en:'Audio Parse'},'视频解析':{en:'Video Parse'},'分段':{en:'Chunking'},'文本嵌入':{en:'Embedding'},'清洗':{en:'Cleaning'},'信息提取':{en:'Extraction'},'训练数据生成':{en:'Training Data Gen'},'SQL 处理':{en:'SQL Process'},'Python 处理':{en:'Python Process'},'条件分支':{en:'Condition'},'智能分段':{en:'Smart Chunking'},'情感分类':{en:'Sentiment'},
  '读取传感器数据':{en:'Read Sensor Data'},'读取 Intelie 传感器数据':{en:'Read Intelie Sensor Data'},'读取 Intelie 1分钟级数据':{en:'Read Intelie 1-min Data'},'读取 Fiix 维修数据':{en:'Read Fiix Maintenance Data'},'读取 Fiix 工单数据':{en:'Read Fiix Work Orders'},'读取 Fiix 资产数据':{en:'Read Fiix Assets'},'1秒→1分钟聚合':{en:'1s→1min Aggregation'},'状态分类':{en:'State Classification'},'会话检测':{en:'Session Detection'},'KPI 计算':{en:'KPI Calculation'},'设备可靠性 KPI':{en:'Equipment Reliability KPI'},'保存 KPI 结果':{en:'Save KPI Results'},'字典去重 + 清洗':{en:'Dict Dedup + Clean'},'资产层级解析':{en:'Asset Hierarchy'},'工单资产展开':{en:'WO Asset Expand'},'工时汇总':{en:'Labor Summary'},'保存处理结果':{en:'Save Results'},
  '读取产品文档':{en:'Read Product Docs'},'保存向量数据':{en:'Save Vector Data'},'读取合同文件':{en:'Read Contracts'},'保存提取结果':{en:'Save Extraction'},'读取销售数据':{en:'Read Sales Data'},'数据清洗':{en:'Data Cleaning'},'保存清洗结果':{en:'Save Cleaned Data'},'读取用户反馈':{en:'Read User Feedback'},'保存分类结果':{en:'Save Classification'},'读取技术文档':{en:'Read Tech Docs'},'读取音频文件':{en:'Read Audio Files'},'保存转写结果':{en:'Save Transcription'},
  // Catalog
  '目录':{en:'Directory'},'库':{en:'Database'},'表':{en:'Table'},'卷':{en:'Volume'},'算子':{en:'Operator'},'模型':{en:'Model'},'系统默认':{en:'System Default'},'新建库':{en:'New Database'},'新建目录':{en:'New Directory'},'上传文件':{en:'Upload Files'},'使用':{en:'Use'},'基本信息':{en:'Basic Info'},'列定义':{en:'Column Definition'},'数据预览':{en:'Data Preview'},'创建人':{en:'Created By'},'更新人':{en:'Updated By'},'更新时间':{en:'Updated'},'描述':{en:'Description'},'行数':{en:'Rows'},'大小':{en:'Size'},'路径':{en:'Path'},'锁定':{en:'Locked'},'搜索节点':{en:'Search nodes'},
  // Data import create
  '请选择连接器':{en:'Select connector'},'请选择数据库':{en:'Select database'},'请选择文件':{en:'Select file'},'选择文件（支持 csv、xls、xlsx）':{en:'Select file (csv, xls, xlsx)'},'点击或拖拽文件到此处上传':{en:'Click or drag files to upload'},'支持 CSV、Excel（xlsx、xls），最大 200MB':{en:'Supports CSV, Excel (xlsx, xls), max 200MB'},
  '载入方式':{en:'Import Method'},'解压策略':{en:'Unzip Strategy'},'重复文件处理':{en:'Duplicate Handling'},'载入范围':{en:'Import Scope'},'文件类型过滤':{en:'File Type Filter'},'路径正则':{en:'Path Regex'},'判断规则':{en:'Detection Rule'},'处理方式':{en:'Action'},'文件名':{en:'Filename'},'文件内容（MD5）':{en:'File Content (MD5)'},'保持结构':{en:'Keep Structure'},'扁平化结构':{en:'Flatten'},'跳过':{en:'Skip'},'覆盖':{en:'Overwrite'},'追加':{en:'Append'},
  '主键':{en:'PK'},'列名':{en:'Column'},'数据类型':{en:'Data Type'},'长度':{en:'Length'},'默认值':{en:'Default'},'列描述':{en:'Description'},'行数据信息':{en:'Sample Data'},'使用标题作为列名':{en:'Use header as column names'},'标题所在行':{en:'Header row'},'从第几行开始导入':{en:'Start from row'},'主键冲突处理':{en:'PK Conflict'},'导入失败':{en:'Fail on conflict'},'跳过冲突行':{en:'Skip conflicts'},'表名':{en:'Table Name'},'表描述':{en:'Table Description'},'智能推荐':{en:'Smart Match'},'按序映射':{en:'Sequential'},'数据源表':{en:'Source Table'},'导入目标表':{en:'Target Table'},'映射方式':{en:'Mapping Mode'},
  '脚本类型':{en:'Script Type'},'预览输出 Schema':{en:'Preview Output Schema'},'回填起始时间':{en:'Backfill Start'},'从最早可用数据开始':{en:'From earliest available'},'指定起始时间':{en:'Specify start time'},'批次大小':{en:'Batch Size'},'批次间隔':{en:'Batch Interval'},'执行窗口':{en:'Execution Window'},'不限制（全天执行）':{en:'No limit (24h)'},
  // Edit mode
  '编辑载入任务':{en:'Edit Import Task'},'保存配置':{en:'Save Config'},'载入任务配置已保存（模拟）':{en:'Import task config saved (simulated)'},'变更将在下次调度时生效。':{en:'Changes will take effect on next scheduled run.'},'水位线将重置，下次载入会重新拉取数据。确认保存？':{en:'Watermark will be reset, next import will re-fetch data. Confirm save?'},
  // Apps mode
  '我的应用':{en:'My Apps'},'创建应用':{en:'Create App'},'搜索应用...':{en:'Search apps...'},'运行中':{en:'Running'},'草稿':{en:'Draft'},'已停止':{en:'Stopped'},'关联智能体':{en:'Linked Agents'},'访问量':{en:'Visits'},'最近编辑':{en:'Last Edited'},'打开应用':{en:'Open App'},'编辑应用':{en:'Edit App'},'个智能体':{en:' agents'},'次访问':{en:' visits'},'还没有创建任何应用':{en:'No apps created yet'},'用自然语言描述你想要的应用，MOI 帮你构建':{en:'Describe the app you want in natural language, MOI builds it for you'},'创建第一个应用':{en:'Create Your First App'},'全部':{en:'All'},'客户管理':{en:'Customer Management'},'数据看板':{en:'Dashboard'},'审批流程':{en:'Approval Flow'},'客服系统':{en:'Customer Service'},'从模板开始':{en:'Start from Template'},'从零开始':{en:'Start from Scratch'},
  '选择列':{en:'Select column'},'条规则通过':{en:' rules passed'},'条需要关注':{en:' need attention'},'点击"运行检查"执行校验':{en:'Click "Run Check" to validate'},'校验中':{en:'Validating...'},
  // Dashboard
  '资源概览':{en:'Resource Overview'},'任务执行':{en:'Task Execution'},'最近活动':{en:'Recent Activity'},'快捷操作':{en:'Quick Actions'},'数据对象':{en:'Data Objects'},'工作流数量':{en:'Workflows'},'计算资源数':{en:'Compute Resources'},'知识库数':{en:'Knowledge Bases'},'近 7 天':{en:'Last 7 Days'},
  // Compute
  '创建资源':{en:'Create Resource'},'启动':{en:'Start'},'停止':{en:'Stop'},'规格':{en:'Spec'},'费用':{en:'Cost'},'自动关机':{en:'Auto Shutdown'},'CPU 通用型':{en:'CPU General'},'CPU 内存型':{en:'CPU Memory'},'GPU 推理型':{en:'GPU Inference'},'GPU 训练型':{en:'GPU Training'},
  // Data share
  '创建分享':{en:'Create Share'},'分享链接':{en:'Share Link'},'有效期':{en:'Expiry'},'访问统计':{en:'Access Stats'},'只读':{en:'Read Only'},'可下载':{en:'Downloadable'},
  // Knowledge base
  '新建知识库':{en:'New Knowledge Base'},'文件数':{en:'Files'},'表数':{en:'Tables'},'卷数':{en:'Volumes'},'关联智能体':{en:'Linked Agents'},'处理状态':{en:'Processing Status'},'就绪':{en:'Ready'},'处理中':{en:'Processing'},'异常':{en:'Error'},'数据源':{en:'Data Source'},'检索设置':{en:'Retrieval Settings'},'语义配置':{en:'Semantic Config'},'向量检索':{en:'Vector Search'},'全文检索':{en:'Full-text Search'},'混合检索':{en:'Hybrid Search'},'业务逻辑':{en:'Business Logic'},'表和列的补充说明':{en:'Table & Column Notes'},'SQL 结果集':{en:'SQL Result Sets'},'优化案例':{en:'Optimization Cases'},'指标与维度':{en:'Metrics & Dimensions'},'表关系':{en:'Table Relations'},
  // User management
  '邀请用户':{en:'Invite User'},'角色':{en:'Role'},'工作区管理员':{en:'Workspace Admin'},'数据开发者':{en:'Data Developer'},'智能体开发者':{en:'Agent Developer'},'只读成员':{en:'Read-only Member'},
  // Account
  '个人信息':{en:'Profile'},'安全设置':{en:'Security'},'修改密码':{en:'Change Password'},'绑定邮箱':{en:'Bind Email'},'绑定手机':{en:'Bind Phone'},
  // Billing
  '费用总览':{en:'Cost Overview'},'用量明细':{en:'Usage Details'},'收支明细':{en:'Transactions'},'设置':{en:'Settings'},'充值':{en:'Top Up'},'Credit 余额':{en:'Credit Balance'},'当日消费':{en:'Today'},'当月消费':{en:'This Month'},'上月消费':{en:'Last Month'},'预计可用天数':{en:'Est. Days Left'},'消费趋势':{en:'Spending Trend'},'月账单':{en:'Monthly Bills'},'账期':{en:'Period'},'出账日期':{en:'Bill Date'},'总消耗':{en:'Total Cost'},'订阅服务':{en:'Subscriptions'},'余额告警':{en:'Balance Alert'},'消费保护':{en:'Spending Protection'},'自动充值':{en:'Auto Top-up'},'通知渠道':{en:'Notification Channels'},'发票设置':{en:'Invoice Settings'},'支付方式':{en:'Payment Method'},'支付宝':{en:'Alipay'},'微信支付':{en:'WeChat Pay'},'对公汇款':{en:'Bank Transfer'},'充值金额':{en:'Amount'},'一次性充值':{en:'One-time'},'月度自动充值':{en:'Monthly Auto'},
  // Data export
  '运行失败':{en:'Failed Runs'},'未运行':{en:'Not Running'},'创建导出':{en:'Create Export'},'导出目标':{en:'Export Target'},'导出格式':{en:'Export Format'},
  '导出至模型/数据集平台':{en:'Export to Model/Dataset Platform'},'导出数据集用于模型训练':{en:'Export datasets for model training'},'导出到对象存储':{en:'Export to Object Storage'},'将数据备份到对象存储中':{en:'Back up data to object storage'},'导出至数据库':{en:'Export to Database'},'将文件或数据发布到数据库中，用于数据检索':{en:'Publish files or data to a database for data retrieval'},
  '导出任务列表':{en:'Export Task List'},'文件详情':{en:'File Details'},'完成时间':{en:'Completed At'},
  '导出到OSS':{en:'Export to OSS'},'导出到标准 S3':{en:'Export to Standard S3'},'导出到MatrixOne':{en:'Export to MatrixOne'},'导出到 Hugging Face':{en:'Export to Hugging Face'},'导出到 LLaMA Factory':{en:'Export to LLaMA Factory'},
  '选择要导出的文件':{en:'Select files to export'},'选择导出位置':{en:'Select export location'},'压缩方式':{en:'Compression'},'不压缩':{en:'No compression'},'开始导出':{en:'Start Export'},
  '请输入任务名称':{en:'Enter task name'},'请选择或新建一个表':{en:'Select or create a table'},
  '导出任务_':{en:'Export Task_'},'Hugging Face 连接器':{en:'Hugging Face Connector'},'LLaMA Factory 连接器':{en:'LLaMA Factory Connector'},
  '暂无可用连接器，请先创建连接器':{en:'No connectors available. Please create one first.'},
  // SQL editor
  '运行':{en:'Run'},'格式化':{en:'Format'},'导出结果':{en:'Export Results'},'查询历史':{en:'Query History'},'表结构':{en:'Table Schema'},
  // Genesis Chat
  '聊天测试':{en:'Chat Test'},'选择模型，调整参数，验证效果':{en:'Select model, adjust parameters, verify results'},'温度':{en:'Temperature'},'精确':{en:'Precise'},'创意':{en:'Creative'},'频率惩罚':{en:'Frequency Penalty'},'频率惩罚，减少重复词汇':{en:'Frequency penalty, reduce repeated tokens'},'存在惩罚':{en:'Presence Penalty'},'存在惩罚，鼓励讨论新话题':{en:'Presence penalty, encourage new topics'},'最大输出 Token':{en:'Max Tokens'},'随机种子':{en:'Seed'},'可选':{en:'Optional'},'留空为随机':{en:'Leave empty for random'},'固定种子可复现结果':{en:'Fixed seed for reproducible results'},'流式输出':{en:'Streaming'},'清空对话':{en:'Clear Chat'},'选择模型后开始对话':{en:'Select a model to start chatting'},'左侧可调整参数和系统提示词':{en:'Adjust parameters and system prompt on the left'},'输入消息，Enter 发送，Shift+Enter 换行...':{en:'Type a message, Enter to send, Shift+Enter for new line...'},'发送':{en:'Send'},'本次对话累计':{en:'Session total'},
  // Genesis Dashboard
  '看板':{en:'Dashboard'},'实时更新':{en:'Live'},'当前 QPS':{en:'Current QPS'},'活跃连接数':{en:'Active Connections'},'模型可用率':{en:'Model Availability'},'队列等待':{en:'Queue Waiting'},'今日请求数':{en:'Today Requests'},'今日 Token 消耗':{en:'Today Token Usage'},'今日费用':{en:'Today Cost'},'今日成功率':{en:'Today Success Rate'},'最近 24 小时请求量':{en:'Last 24h Request Volume'},'模型健康度':{en:'Model Health'},'最近异常':{en:'Recent Errors'},'查看全部日志':{en:'View All Logs'},'延迟':{en:'Latency'},'错误率':{en:'Error Rate'},
  // Genesis Usage
  '用量':{en:'Usage'},'今天':{en:'Today'},'本周':{en:'This Week'},'本月':{en:'This Month'},'自定义':{en:'Custom'},'查询':{en:'Query'},'按模型':{en:'By Model'},'按密钥':{en:'By Key'},'导出':{en:'Export'},'合计':{en:'Total'},'模型费用占比':{en:'Model Cost Breakdown'},
  // Genesis Logs
  '日志':{en:'Logs'},'搜索密钥名称...':{en:'Search key name...'},'搜索来源 IP...':{en:'Search source IP...'},'请求详情':{en:'Request Details'},'请求 ID':{en:'Request ID'},'请求内容':{en:'Request Content'},'响应内容':{en:'Response Content'},'错误信息':{en:'Error Info'},'重置':{en:'Reset'},'上一页':{en:'Previous'},'下一页':{en:'Next'},
  // Genesis Keys
  '密钥':{en:'Keys'},'创建密钥':{en:'Create Key'},'编辑密钥':{en:'Edit Key'},'更新密钥':{en:'Regenerate Key'},'删除密钥':{en:'Delete Key'},'禁用密钥':{en:'Disable Key'},'密钥名称':{en:'Key Name'},'可用模型':{en:'Available Models'},'全部模型':{en:'All Models'},'指定模型':{en:'Specific Models'},'额度限制':{en:'Quota Limit'},'总额度':{en:'Total Quota'},'日额度上限':{en:'Daily Limit'},'速率限制':{en:'Rate Limit'},'并发数':{en:'Concurrency'},'过期时间':{en:'Expiry'},'永不过期':{en:'Never'},'一天':{en:'1 Day'},'一个月':{en:'1 Month'},
  // Genesis Guide
  '使用':{en:'Guide'},'了解如何在各种场景下使用 Genesis 模型服务':{en:'Learn how to use Genesis model services in various scenarios'},'API 调用':{en:'API Call'},'MOI 智能体':{en:'MOI Agent'},'MOI 工作流':{en:'MOI Workflow'},'MOI 知识库':{en:'MOI Knowledge Base'},'API 基础信息':{en:'API Basics'},'认证方式':{en:'Authentication'},'兼容格式':{en:'Compatible Format'},'API 调用示例':{en:'API Call Examples'},'常见问题':{en:'FAQ'},'在 MOI 智能体中使用':{en:'Use in MOI Agent'},'在 MOI 工作流中使用':{en:'Use in MOI Workflow'},'在 MOI 知识库中使用':{en:'Use in MOI Knowledge Base'},
  // Admin
  '管理后台':{en:'Admin'},'总览':{en:'Overview'},'渠道管理':{en:'Channel Management'},'模型管理':{en:'Model Management'},'邀请用户':{en:'Invite User'},'超级管理员':{en:'Super Admin'},'管理员':{en:'Admin'},'普通用户':{en:'User'},'注册时间':{en:'Registered'},'最近登录':{en:'Last Login'},'添加渠道':{en:'Add Channel'},'渠道名称':{en:'Channel Name'},'支持模型':{en:'Supported Models'},'权重':{en:'Weight'},'添加模型':{en:'Add Model'},'显示名称':{en:'Display Name'},'渠道':{en:'Channel'},'输入价格':{en:'Input Price'},'输出价格':{en:'Output Price'},'返回 Genesis 前台':{en:'Back to Genesis'},
  // Common
  '搜索':{en:'Search'},'确定':{en:'OK'},'取消':{en:'Cancel'},'保存':{en:'Save'},'关闭':{en:'Close'},'返回':{en:'Back'},'提交':{en:'Submit'},'新建':{en:'New'},'添加':{en:'Add'},'请选择':{en:'Select'},'暂无数据':{en:'No data'},'加载中':{en:'Loading'},'确认':{en:'Confirm'},'全选':{en:'Select All'},'已选':{en:'Selected'},'刷新':{en:'Refresh'},
  // SQL Editor (dynamic)
  '当前语句':{en:'Current Statement'},'请先在编辑器中输入 SQL 语句':{en:'Please enter a SQL statement in the editor'},'当前语句为空':{en:'Current statement is empty'},'执行成功':{en:'Execution Successful'},'返回':{en:'Returned'},'耗时':{en:'Duration'},'行':{en:'rows'},'暂无包含表的数据库':{en:'No databases with tables'},'柱状图':{en:'Bar'},'折线图':{en:'Line'},'散点图':{en:'Scatter'},'面积图':{en:'Area'},'图表类型':{en:'Chart Type'},'X 轴':{en:'X Axis'},'Y 轴':{en:'Y Axis'},'+ 添加 Y 轴':{en:'+ Add Y Axis'},'无可用数据生成图表':{en:'No data available for chart'},'请先执行查询以生成图表':{en:'Run a query first to generate chart'},'恢复':{en:'Restore'},'最小化':{en:'Minimize'},'最大化':{en:'Maximize'},'工作簿':{en:'Workbook'},'确定删除该工作簿？':{en:'Delete this workbook?'},'处理数据库':{en:'Processing DB'},'用户行为表':{en:'User Behavior Table'},'执行':{en:'Execute'},'点击"执行"运行 SQL 查看结果':{en:'Click "Execute" to run SQL and view results'},'数据库':{en:'Database'},
  // Workflow (dynamic)
  '【NESR-湖仓项目】设备可靠性数据处理':{en:'[NESR-Lakehouse] Equipment Reliability Processing'},'默认计算资源':{en:'Default Compute'},'GPU 计算集群':{en:'GPU Compute Cluster'},'高性能计算集群':{en:'High-perf Compute Cluster'},'暂无工作流':{en:'No workflows'},'确定删除该工作流？':{en:'Delete this workflow?'},'已恢复，下次新建工作流时将显示模板选择':{en:'Restored. Template selection will show next time.'},'停止':{en:'Stop'},
  // Compute (dynamic)
  '标准型':{en:'Standard'},'内存型':{en:'Memory'},'工作区默认计算资源，适用于常规数据处理任务':{en:'Default workspace compute for general data processing'},'用于 AI 推理和嵌入计算':{en:'For AI inference and embedding'},'高内存配置，用于大文件解析和数据增强':{en:'High memory for large file parsing and augmentation'},'轻量级资源，用于 Notebook 开发调试':{en:'Lightweight for Notebook development'},'【NESR-湖仓项目】设备可靠性数据处理':{en:'[NESR-Lakehouse] Equipment Reliability Processing'},'数据质量分析':{en:'Data Quality Analysis'},'文本嵌入批处理':{en:'Text Embedding Batch'},'自定义去敏处理':{en:'Custom Desensitization'},'合同关键信息提取':{en:'Contract Key Info Extraction'},'空闲':{en:'Idle'},'暂停计算资源':{en:'Suspend Compute'},'暂停后计算资源将停止运行，不再产生 Credit 消耗。':{en:'After suspension, the compute resource will stop and no longer consume Credits.'},'关联的工作负载将被中断。确定暂停？':{en:'Associated workloads will be interrupted. Confirm suspend?'},'删除计算资源':{en:'Delete Compute'},'该资源正在运行中，删除后关联工作负载将被中断。':{en:'This resource is running. Deletion will interrupt associated workloads.'},'确定删除计算资源「':{en:'Delete compute resource "'},'」？此操作不可恢复。':{en:'"? This cannot be undone.'},'不自动暂停':{en:'No auto-suspend'},'分钟无活动':{en:'min inactive'},'默认':{en:'Default'},'节点数':{en:'Nodes'},'资源使用率':{en:'Resource Usage'},'内存':{en:'Memory'},'当前 Credit 消耗':{en:'Current Credit Usage'},'关联工作负载':{en:'Associated Workloads'},'暂无关联工作负载':{en:'No associated workloads'},'规格详情':{en:'Spec Details'},'规格 ID':{en:'Spec ID'},'系列':{en:'Family'},'核':{en:'cores'},'块':{en:'units'},'Credit 单价':{en:'Credit Price'},'暂无计算资源':{en:'No compute resources'},'新建计算资源':{en:'New Compute Resource'},
  '已暂停':{en:'Suspended'},'启动中':{en:'Starting'},'扩缩容中':{en:'Scaling'},'异常':{en:'Error'},'暂停':{en:'Suspend'},'小时':{en:'hour'},'节点':{en:'node'},'分钟':{en:'min'},'个':{en:''},'张三':{en:'Zhang San'},'李四':{en:'Li Si'},'王五':{en:'Wang Wu'},'系统':{en:'System'},'用户':{en:'User'},
  // Data Share (dynamic)
  '默认目录':{en:'Default Directory'},'原始数据库':{en:'Raw Database'},'样例卷':{en:'Sample Data Volume'},'客户卷':{en:'Customer Data Volume'},'产品信息表':{en:'Product Info Table'},'解析结果':{en:'Parse Results'},'文档分段结果表':{en:'Doc Chunking Results Table'},'订单汇总表':{en:'Order Summary Table'},'开发目录':{en:'Dev Directory'},'测试数据库':{en:'Test Database'},'测试卷':{en:'Test Data Volume'},'产品数据共享':{en:'Product Data Share'},'处理结果分享':{en:'Processing Results Share'},'默认 / 原始数据库 / 样例卷':{en:'Default / Raw DB / Sample Data Volume'},'默认 / 原始数据库 / 产品信息表':{en:'Default / Raw DB / Product Info Table'},'默认 / 处理数据库':{en:'Default / Processing DB'},'数据分析项目':{en:'Data Analysis Project'},'只读':{en:'Read Only'},'外部客户数据':{en:'External Customer Data'},'客户画像表':{en:'Customer Profile Table'},'测试工作区':{en:'Test Workspace'},'外部客户画像':{en:'External Customer Profile'},'默认 / 原始数据库':{en:'Default / Raw DB'},'模型训练数据集':{en:'Model Training Dataset'},'训练卷':{en:'Training Data Volume'},'行业知识库':{en:'Industry Knowledge Base'},'行业文档目录':{en:'Industry Doc Directory'},'已订阅':{en:'Subscribed'},'未订阅':{en:'Unsubscribed'},'取消订阅':{en:'Unsubscribe'},'订阅':{en:'Subscribe'},'请选择工作区':{en:'Select workspace'},'外部工作区':{en:'External Workspace'},'添加发布':{en:'Add Publication'},'编辑发布':{en:'Edit Publication'},'删除发布':{en:'Delete Publication'},'请选择发布对象':{en:'Please select a publish object'},'请选择发布目标工作区':{en:'Please select target workspace'},'确定删除该发布？删除后目标工作区将无法再访问此数据。':{en:'Delete this publication? Target workspaces will lose access.'},'确定取消订阅？取消后将无法在本工作区访问该数据。':{en:'Unsubscribe? You will lose access to this data.'},'该对象为目录级别，将订阅到工作区根级别':{en:'This is a directory-level object, will subscribe at workspace root'},'请选择一个目录作为订阅位置':{en:'Select a directory as subscription location'},'请选择一个库作为订阅位置':{en:'Select a database as subscription location'},'目录级别对象将直接订阅到工作区根级别':{en:'Directory-level objects subscribe at workspace root'},'工作区根级别':{en:'Workspace Root'},'请选择订阅位置':{en:'Please select subscription location'},'暂无发布数据':{en:'No publications'},'暂无可订阅的数据':{en:'No data available to subscribe'},
  // Catalog (dynamic)
  '未知类型':{en:'Unknown Type'},'库名':{en:'DB Name'},'库描述':{en:'DB Description'},'卷数量':{en:'Volumes'},'表数量':{en:'Tables'},'算子数量':{en:'Operators'},'模型数量':{en:'Models'},'暂无库，点击上方\u201c新建库\u201d按钮创建':{en:'No databases. Click "New Database" above to create one.'},'暂无子项，点击上方\u201c创建\u201d按钮添加卷、表、算子或模型':{en:'No items. Click "Create" above to add volumes, tables, operators or models.'},'子项目数':{en:'Sub-items'},'修改人':{en:'Updated By'},'文件':{en:'Files'},'卷名':{en:'Volume Name'},'文件数':{en:'File Count'},'总大小':{en:'Total Size'},'文件名':{en:'Filename'},'暂无文件':{en:'No files'},'下载':{en:'Download'},'表名':{en:'Table Name'},'列':{en:'Columns'},'统计':{en:'Statistics'},'数据抽样':{en:'Data Sample'},'列名':{en:'Column'},'数据类型':{en:'Data Type'},'是否主键':{en:'Primary Key'},'默认值':{en:'Default'},'无列定义':{en:'No column definitions'},'最大值':{en:'Max'},'最小值':{en:'Min'},'无统计数据':{en:'No statistics'},'无表定义':{en:'No table definition'},'无数据':{en:'No data'},'算子类型':{en:'Operator Type'},'当前版本':{en:'Current Version'},'最后更新':{en:'Last Updated'},'发布到':{en:'Published To'},'版本历史':{en:'Version History'},'代码':{en:'Code'},'模型类别':{en:'Model Category'},'接入方式':{en:'Access Method'},'供应商':{en:'Provider'},'基础地址':{en:'Base URL'},'模型 ID':{en:'Model ID'},'信任远程代码':{en:'Trust Remote Code'},'框架':{en:'Framework'},'使用模型 API':{en:'Use Model API'},'API 地址':{en:'API Endpoint'},'复制':{en:'Copy'},'模型名称（请求中使用）':{en:'Model Name (used in requests)'},'调用示例':{en:'Usage Example'},'已复制到剪贴板':{en:'Copied to clipboard'},'系统默认，不可操作':{en:'System default, not editable'},'新建卷':{en:'New Volume'},'新建表':{en:'New Table'},'新建算子':{en:'New Operator'},'新建模型':{en:'New Model'},'卷名称':{en:'Volume Name'},'新建子项':{en:'New Item'},'目录名称':{en:'Directory Name'},'系统默认项不可删除':{en:'System defaults cannot be deleted'},'确定删除':{en:'Confirm delete '},'？此操作不可恢复。':{en:'? This cannot be undone.'},'请选择模型类别':{en:'Please select a model category'},'暂无目录，点击上方\u201c新建目录\u201d按钮创建':{en:'No directories. Click "New Directory" above to create one.'},'目录名称':{en:'Directory Name'},'目录描述':{en:'Directory Description'},'库数量':{en:'Databases'},'数据检查':{en:'Data Check'},'校验规则':{en:'Validation Rules'},'校验结果':{en:'Results'},'条规则通过':{en:' rules passed'},'条需要关注':{en:' need attention'},'载入任务已创建':{en:'Import task created'},'个文件已添加到卷中':{en:' files added to volume'},'创建数据载入任务':{en:'Create Data Import Task'},'上传失败':{en:'Upload failed'},
  // Data import create page (additional)
  '载入策略':{en:'Import Strategy'},'载入方式：':{en:'Import Method:'},'连接器：':{en:'Connector:'},'载入目标':{en:'Target'},'请选择卷（Volume）':{en:'Select volume'},'支持 PDF、Word、TXT、图片等，最大 200MB':{en:'Supports PDF, Word, TXT, images, etc. Max 200MB'},'忽略目录结构':{en:'Ignore directory structure'},'保持原始目录':{en:'Keep original structure'},'更新方式：':{en:'Update Method:'},'文件名相同':{en:'Same filename'},'图片':{en:'Images'},'音频':{en:'Audio'},'视频':{en:'Video'},'文件选择':{en:'File Selection'},'文件大小':{en:'File Size'},
  '网页采集配置':{en:'Web Scraping Config'},'网页地址：':{en:'Web URL:'},'加载页面':{en:'Load Page'},'快捷模板：':{en:'Quick Templates:'},'深交所公告':{en:'SZSE Announcements'},'上交所公告':{en:'SSE Announcements'},'巨潮资讯':{en:'CNINFO'},'页面预览（静态快照）':{en:'Page Preview (Static Snapshot)'},'缩放':{en:'Zoom'},'选择区域':{en:'Select Region'},'清除':{en:'Clear'},
  '采集规则预览':{en:'Scraping Rule Preview'},'采集深度：':{en:'Scraping Depth:'},'仅当前页（深度 0）':{en:'Current page only (depth 0)'},'当前页 + 子页（深度 1）':{en:'Current + sub-pages (depth 1)'},'递归 2 层':{en:'Recursive 2 levels'},'递归 3 层':{en:'Recursive 3 levels'},'自动翻页：':{en:'Auto Pagination:'},'开启（自动翻到最后一页）':{en:'On (auto to last page)'},'开启（限制最大页数）':{en:'On (limit max pages)'},'关闭（仅采集当前页）':{en:'Off (current page only)'},'最大页数：':{en:'Max Pages:'},
  '采集内容：':{en:'Content Type:'},'全部采集（页面 + 文件）':{en:'All (pages + files)'},'仅采集页面':{en:'Pages only'},'仅采集文件':{en:'Files only'},'请求间隔':{en:'Request Interval'},'秒':{en:'sec'},'并发数':{en:'Concurrency'},
  '选择文件：':{en:'Select File:'},'请选择文件（支持 csv、xls、xlsx）':{en:'Select file (csv, xls, xlsx)'},'检测到 Excel 工作表，请选择要导入的 Sheet：':{en:'Excel sheets detected, select sheets to import:'},'选择表：':{en:'Select Table:'},'请选择 Endpoint':{en:'Select Endpoint'},
  '分页策略：':{en:'Pagination:'},'游标分页（cursor / next_token）':{en:'Cursor pagination (cursor / next_token)'},'页码分页（page + per_page）':{en:'Page pagination (page + per_page)'},'无分页（单次返回全部）':{en:'No pagination (single response)'},'数据路径：':{en:'Data Path:'},
  '请选择载入目标':{en:'Select import target'},'为每个 Sheet 选择载入目标':{en:'Select target for each sheet'},
  '每小时':{en:'Hourly'},'每天':{en:'Daily'},'每周':{en:'Weekly'},'每月':{en:'Monthly'},'自定义 Cron':{en:'Custom Cron'},
  '增量更新（跳过已采集的 URL 和相同文件）':{en:'Incremental (skip collected URLs and same files)'},'全量覆盖（每次重新采集）':{en:'Full overwrite (re-scrape each time)'},
  '1分钟':{en:'1 min'},'5分钟':{en:'5 min'},'10分钟':{en:'10 min'},'30分钟':{en:'30 min'},'1小时':{en:'1 hour'},'2小时':{en:'2 hours'},'4小时':{en:'4 hours'},'6小时':{en:'6 hours'},'12小时':{en:'12 hours'},'1天':{en:'1 day'},'7天':{en:'7 days'},
  '10 分钟':{en:'10 min'},'30 分钟':{en:'30 min'},'3 小时':{en:'3 hours'},'6 小时':{en:'6 hours'},'12 小时':{en:'12 hours'},'1 天':{en:'1 day'},'1 周':{en:'1 week'},'1 个月':{en:'1 month'},
  '3 秒':{en:'3 sec'},'5 秒':{en:'5 sec'},'10 秒':{en:'10 sec'},'30 秒':{en:'30 sec'},'60 秒':{en:'60 sec'},
  '请选择增量字段':{en:'Select incremental field'},'无回溯':{en:'No lookback'},'回填起始时间':{en:'Backfill Start Time'},'无间隔':{en:'No interval'},
  '全量分区同步':{en:'Full Partition Sync'},'分区字段':{en:'Partition Fields'},'请选择分区字段':{en:'Select partition fields'},
  '载入前预处理脚本':{en:'Pre-processing script'},'脚本类型：':{en:'Script Type:'},'预览输出 Schema ▶':{en:'Preview Output Schema ▶'},'预处理输出 Schema（将用于下方表定义）：':{en:'Pre-processing output schema (used for table definition below):'},
  '行开始导入':{en:'row to start import'},'从第':{en:'From'},'主键冲突处理：':{en:'PK Conflict Handling:'},'请输入表名':{en:'Enter table name'},'请输入表描述':{en:'Enter table description'},'抽样预览':{en:'Sample Preview'},
  '将数据源表的列映射到目标表的列，主键列必须映射，其余列可选择不映射':{en:'Map source columns to target columns. Primary key columns must be mapped, others are optional.'},'推荐类型':{en:'Suggested Type'},'源列映射':{en:'Source Mapping'},'列类型':{en:'Column Type'},
  '已选中 0 个文件':{en:'0 files selected'},'创建并开始载入':{en:'Create & Start Import'},'仅支持 csv、xls、xlsx 格式文件':{en:'Only csv, xls, xlsx files supported'},'选择载入目标':{en:'Select Import Target'},
  '选择载入位置':{en:'Select Import Location'},'非结构化数据需要载入到卷（Volume）中':{en:'Unstructured data must be imported into a Volume'},
  'CSV 配置':{en:'CSV Config'},'分隔符':{en:'Separator'},'定界符':{en:'Delimiter'},'反斜杠转义':{en:'Backslash Escape'},'逗号 (,)':{en:'Comma (,)'},'分号 (;)':{en:'Semicolon (;)'},'制表符 (\\t)':{en:'Tab (\\t)'},'管道符号 (|)':{en:'Pipe (|)'},'空格 ( )':{en:'Space ( )'},'双引号 ("")':{en:'Double quote ("")'},"单引号 (')":{en:"Single quote (')"},'波浪号 (~)':{en:'Tilde (~)'},'无':{en:'None'},
  '第 1 条 / 共 5 条':{en:'1 of 5'},'换一条':{en:'Next'},
  '点击页面元素选择采集区域，蓝色高亮表示已选中':{en:'Click page elements to select scraping regions. Blue highlight = selected'},'加载页面后，点击"选择区域"在预览中标记要采集的区域':{en:'After loading, click "Select Region" to mark areas to scrape'},
  '不映射':{en:'No mapping'},'已有表':{en:'Existing'},'文件夹':{en:'Folder'},'全部文件':{en:'All files'},
  '新建目录（模拟）':{en:'New directory (simulated)'},'新建库（模拟）':{en:'New library (simulated)'},'新建卷（模拟）':{en:'New volume (simulated)'},
  '请先选择连接器':{en:'Please select a connector first'},'请输入网页地址':{en:'Please enter a web URL'},'载入任务创建成功（模拟）':{en:'Import task created (simulated)'},'采集任务创建成功（模拟）':{en:'Scraping task created (simulated)'},
  '内容区域':{en:'Content'},'文件下载':{en:'File Download'},'翻页控件':{en:'Pagination'},'搜索区域':{en:'Search Area'},'查询按钮':{en:'Search Button'},'排除区域':{en:'Exclude Area'},'跟踪链接':{en:'Follow Links'},
  '推荐（时间戳字段）':{en:'Recommended (timestamp fields)'},'其他字段':{en:'Other fields'},
  '检测到嵌套 JSON 字段，当前已打平第一层。':{en:'Nested JSON fields detected, first level flattened.'},'部分字段已深度打平，仍有嵌套 JSON 字段。':{en:'Some fields deep-flattened, nested JSON fields remain.'},'所有嵌套字段已深度打平。':{en:'All nested fields deep-flattened.'},'全部深度打平':{en:'Deep flatten all'},'还原全部':{en:'Restore all'},'展开':{en:'Expand'},'收起':{en:'Collapse'},
  '文档':{en:'docs'},'区域':{en:'regions'},'目标地址':{en:'Target URL'},'层':{en:'levels'},'最多':{en:'Max'},'页':{en:'pages'},'全部':{en:'All'},'并发':{en:'Concurrency'},'选区数量':{en:'Selected regions'},
  // Tooltip translations
  '「卷」是存放非结构化数据的单位，请选择或创建一个「卷」用于存储载入的原始数据。您可以在 Catalog 功能中查看该卷及其中的数据。':{en:'A "Volume" is a unit for storing unstructured data. Select or create a volume for the imported raw data. You can view the volume and its data in Catalog.'},
  '一次性载入适合仅需导入一次的场景；周期性载入适合定期更新数据的需求，并可设置具体周期（如每小时或每日）。':{en:'One-time import is for single imports; periodic import is for regular data updates with configurable intervals (hourly, daily, etc.).'},
  '设置压缩文件的解压方式。保持结构将维护原有文件夹层级；扁平化结构将所有文件放在同一级目录。':{en:'Set extraction method for compressed files. Keep structure maintains folder hierarchy; flatten puts all files in one directory.'},
  '为避免重复导入相同文件，提供根据文件名和文件内容（MD5）的文件去重功能。':{en:'Deduplication based on filename and content (MD5) to avoid importing duplicate files.'},
  '载入范围由文件类型与路径正则表达式共同限定，二者为「且」关系。当路径正则为空时，仅按文件类型筛选。对于压缩文件，路径正则匹配基于解压前的压缩包原始路径；匹配成功后，包内文件在解压后再按文件类型规则筛选。':{en:'Import scope is defined by file types AND path regex. When path regex is empty, only file type filtering applies. For compressed files, path regex matches against the original archive path; after matching, files inside are filtered by file type rules after extraction.'},
  '使用正则表达式匹配文件路径，只处理匹配的文件。':{en:'Use regex to match file paths. Only matching files will be processed.'},
  '选择 Catalog 中的卷用于存储采集的数据。':{en:'Select a volume in Catalog to store scraped data.'},
  '每次请求之间的等待时间，避免对目标网站造成过大压力。建议设置 1-3 秒。':{en:'Wait time between requests to avoid overloading the target website. Recommended: 1-3 seconds.'},
  '同时发起的请求数量。并发越高采集越快，但可能触发目标网站的反爬机制。':{en:'Number of concurrent requests. Higher concurrency = faster scraping, but may trigger anti-scraping mechanisms.'},

  // Page: data-import.html — Status badges with emoji prefixes
  '✓ 完成':{en:'✓ Completed'},'⟳ 运行中':{en:'⟳ Running'},'✕ 失败':{en:'✕ Failed'},

  // App Dev - Agent page
  '对话历史':{en:'Chat History'},'新建对话':{en:'New Chat'},'技能库':{en:'Skill Library'},'工具库':{en:'Tool Library'},'任务':{en:'Tasks'},'智能体列表':{en:'Agent List'},'任务列表':{en:'Task List'},'创建知识库':{en:'Create Knowledge Base'},'新建技能':{en:'New Skill'},'新建工具':{en:'New Tool'},'新建智能体':{en:'New Agent'},'暂无智能体':{en:'No agents'},'新智能体':{en:'New Agent'},'新对话':{en:'New Chat'},'刚刚':{en:'Just now'},
  '智能体描述':{en:'Agent Description'},'点击更换头像':{en:'Click to change avatar'},'点击头像可更换表情或图片':{en:'Click avatar to change emoji or image'},'系统提示词':{en:'System Prompt'},'输入系统提示词...':{en:'Enter system prompt...'},'AI 改写':{en:'AI Rewrite'},'改写':{en:'Rewrite'},'表情符号':{en:'Emoji'},'上传图片':{en:'Upload Image'},'选择知识库':{en:'Select Knowledge Base'},'选择技能':{en:'Select Skill'},'上传文件':{en:'Upload File'},'更多操作':{en:'More Actions'},'技能':{en:'Skills'},'工具':{en:'Tools'},
  '文档知识问答':{en:'Document Q&A'},'图文混合问答':{en:'Image-Text Q&A'},'精准问答':{en:'Precise Q&A'},'对话式检索':{en:'Conversational Search'},'表格提取':{en:'Table Extraction'},'实体关系抽取':{en:'Entity Relation Extraction'},'事件抽取':{en:'Event Extraction'},'QA 数据生成':{en:'QA Data Generation'},'文档摘要':{en:'Document Summary'},'文档翻译':{en:'Document Translation'},'内容改写润色':{en:'Content Rewriting'},'NL2SQL 查询':{en:'NL2SQL Query'},'数据洞察分析':{en:'Data Insight Analysis'},'文档分类标注':{en:'Document Classification'},'相似文档检索':{en:'Similar Document Search'},
  '网页浏览':{en:'Web Browsing'},'智能搜索':{en:'Smart Search'},'竞品监控':{en:'Competitor Monitoring'},'搜索引擎':{en:'Search Engine'},'学术论文检索':{en:'Academic Paper Search'},'RSS 订阅聚合':{en:'RSS Feed Aggregation'},'网页截图':{en:'Web Screenshot'},'链接预览':{en:'Link Preview'},'价格追踪':{en:'Price Tracking'},
  '邮件管理':{en:'Email Management'},'钉钉集成':{en:'DingTalk Integration'},'飞书集成':{en:'Feishu Integration'},'短信通知':{en:'SMS Notification'},'微信公众号':{en:'WeChat Official Account'},'日程管理':{en:'Calendar Management'},'项目管理':{en:'Project Management'},'笔记同步':{en:'Note Sync'},'待办清单':{en:'Todo List'},'会议纪要':{en:'Meeting Minutes'},'文件管理':{en:'File Management'},'时间追踪':{en:'Time Tracking'},'习惯打卡':{en:'Habit Tracker'},'剪贴板历史':{en:'Clipboard History'},
  '代码审查':{en:'Code Review'},'代码生成':{en:'Code Generation'},'调试助手':{en:'Debug Assistant'},'测试生成':{en:'Test Generation'},'API 文档生成':{en:'API Doc Generation'},'数据库迁移':{en:'Database Migration'},'正则表达式':{en:'Regex'},'代码重构':{en:'Code Refactoring'},'日志分析':{en:'Log Analysis'},'监控告警':{en:'Monitoring & Alerts'},'服务部署':{en:'Service Deployment'},'数据库运维':{en:'Database Operations'},
  '文章写作':{en:'Article Writing'},'SEO 优化':{en:'SEO Optimization'},'社交媒体':{en:'Social Media'},'营销文案':{en:'Marketing Copy'},'视频脚本':{en:'Video Script'},'播客转录':{en:'Podcast Transcription'},'PPT 生成':{en:'PPT Generation'},'多语言内容':{en:'Multilingual Content'},'内容日历':{en:'Content Calendar'},'标题优化':{en:'Title Optimization'},
  'CSV/Excel 分析':{en:'CSV/Excel Analysis'},'数据可视化':{en:'Data Visualization'},'ETL 流水线':{en:'ETL Pipeline'},'报表生成':{en:'Report Generation'},'异常检测':{en:'Anomaly Detection'},'用户行为分析':{en:'User Behavior Analysis'},'A/B 测试分析':{en:'A/B Test Analysis'},
  // App Builder
  '未命名应用':{en:'Untitled App'},'已发布':{en:'Published'},'分享':{en:'Share'},'发布':{en:'Publish'},'应用构建对话':{en:'App Builder Chat'},'用自然语言描述你想要的应用':{en:'Describe the app you want in natural language'},'应用模板':{en:'App Templates'},'客户管理':{en:'Customer Management'},'审批流程':{en:'Approval Workflow'},'数据看板':{en:'Dashboard'},'客服系统':{en:'Help Desk'},'库存管理':{en:'Inventory Management'},'从零开始':{en:'Start from Scratch'},
  '桌面端':{en:'Desktop'},'平板':{en:'Tablet'},'手机':{en:'Mobile'},'搜索客户...':{en:'Search customers...'},'累计订单':{en:'Total Orders'},'累计金额':{en:'Total Amount'},'健康度':{en:'Health Score'},'历史订单':{en:'Order History'},'订单号':{en:'Order No.'},'日期':{en:'Date'},'产品':{en:'Product'},'金额':{en:'Amount'},'客户':{en:'Customers'},'订单':{en:'Orders'},'自定义应用':{en:'Custom App'},
  // Notebook
  '从模板开始':{en:'Start from Template'},'工作流算子使用教程':{en:'Workflow Operator Tutorial'},'数据探索与分析':{en:'Data Exploration & Analysis'},'自定义算子开发':{en:'Custom Operator Development'},'搜索 Notebook':{en:'Search Notebook'},'全部语言':{en:'All Languages'},'混合':{en:'Mixed'},'新建 Notebook':{en:'New Notebook'},'算子引用':{en:'Operator References'},'暂无 Notebook':{en:'No Notebooks'},'打开':{en:'Open'},'确定删除该 Notebook？':{en:'Delete this Notebook?'},'系统算子':{en:'System Operator'},'自定义算子':{en:'Custom Operator'},
  // Notebook Editor
  '未命名 Notebook':{en:'Untitled Notebook'},'已保存':{en:'Saved'},'GPU 推理资源':{en:'GPU Inference'},'大规模数据处理':{en:'Large-scale Processing'},'开发测试资源':{en:'Dev/Test Resource'},'保存版本':{en:'Save Version'},'全部运行':{en:'Run All'},'智能编写':{en:'AI Assist'},'就绪':{en:'Ready'},'新建算子':{en:'New Operator'},'搜索 Catalog...':{en:'Search Catalog...'},'版本对比':{en:'Version Compare'},'版本历史':{en:'Version History'},'手动保存':{en:'Manual Save'},'自动保存':{en:'Auto Save'},'AI 采纳':{en:'AI Accepted'},'至少保留一个 Cell':{en:'Keep at least one Cell'},'名称（中文）':{en:'Name (Chinese)'},'标识（英文）':{en:'Identifier (English)'},'创建者':{en:'Creator'},'当前用户':{en:'Current User'},'最新更新时间':{en:'Last Updated'},'功能描述':{en:'Function Description'},'输入':{en:'Input'},'输出':{en:'Output'},'发布算子':{en:'Publish Operator'},'发送':{en:'Send'},
  // Website - Login
  '手机号':{en:'Phone'},'请输入手机号':{en:'Enter phone number'},'中国 +86':{en:'China +86'},'请输入验证码':{en:'Enter verification code'},'获取验证码':{en:'Get Code'},'密码登录':{en:'Password Login'},'短信验证码登录':{en:'SMS Code Login'},'请输入邮箱地址':{en:'Enter email address'},'请输入密码':{en:'Enter password'},'登录 / 注册':{en:'Login / Register'},'其他登录方式':{en:'Other Login Methods'},'还没有账号？':{en:'Don\'t have an account?'},'立即注册':{en:'Register Now'},'忘记密码？':{en:'Forgot Password?'},
  // Website - Register
  '创建账号':{en:'Create Account'},'姓':{en:'Last Name'},'名':{en:'First Name'},'公司':{en:'Company'},'邮箱地址':{en:'Email Address'},'请输入您的姓':{en:'Enter your last name'},'请输入您的名':{en:'Enter your first name'},'请输入您的公司名称':{en:'Enter your company name'},'请填写有效的邮箱地址':{en:'Enter a valid email address'},'确认密码':{en:'Confirm Password'},'请确认您的密码':{en:'Confirm your password'},'注册':{en:'Register'},'已有账号？':{en:'Already have an account?'},'立即登录':{en:'Login Now'},
  // Website - Homepage
  '产品':{en:'Products'},'解决方案':{en:'Solutions'},'客户案例':{en:'Case Studies'},'体验中心':{en:'Experience Center'},'工作台':{en:'Workspace'},'免费试用':{en:'Free Trial'},'以数生智':{en:'Data to Intelligence'},'以智驭数':{en:'Intelligence to Data'},'产品体验':{en:'Product Experience'},'白皮书':{en:'White Paper'},'进入控制台':{en:'Go to Console'},'核心能力':{en:'Core Capabilities'},'数据管理':{en:'Data Management'},'智能应用':{en:'Smart Applications'},'权限管理':{en:'Permission Management'},'告警监控':{en:'Alert Monitoring'},'账号管理功能待实现':{en:'Account management coming soon'},'计费中心功能待实现':{en:'Billing center coming soon'},
  // Page: data-import-create.html
  '支持多种文件类型混合上传，如文档、图片、音频、视频等':{en:'Supports mixed file types: documents, images, audio, video, etc.'},'支持结构化文件批量上传，包括 csv、xlsx、xls，并将结构化数据导入表中':{en:'Supports batch upload of structured files (csv, xlsx, xls) and imports data into tables'},
  '请输入列名':{en:'Enter column name'},'输入公司名称或代码':{en:'Enter company name or code'},'输入搜索条件，如：金盘科技':{en:'Enter search term, e.g. Jinpan Tech'},
  '证券代码':{en:'Securities Code'},'简称':{en:'Short Name'},'公告标题':{en:'Announcement Title'},'上一页':{en:'Previous'},'下一页':{en:'Next'},
  '查询':{en:'Query'},'上市公司公告':{en:'Listed Company Announcements'},'（模拟数据）':{en:'(Simulated Data)'},
  '文件夹':{en:'Folder'},'全部文件':{en:'All files'},'不映射':{en:'No mapping'},'已有表':{en:'Existing'},
  '还原全部':{en:'Restore all'},'全部深度打平':{en:'Deep flatten all'},
  '未填写':{en:'Not filled'},'点击"选择区域"在预览中标记采集区域':{en:'Click "Select Region" to mark scraping areas in preview'},
  '排除区域：':{en:'Exclude areas: '},'目标地址：':{en:'Target URL: '},'请求间隔：':{en:'Request interval: '},'选区数量：':{en:'Selected regions: '},
  '推荐类型':{en:'Suggested Type'},'源列映射':{en:'Source Mapping'},'列类型':{en:'Column Type'},
  // Page: dashboard.html
  'API 管理':{en:'API Management'},'查看全部':{en:'View All'},'任务执行状态（近 7 天）':{en:'Task Execution Status (Last 7 Days)'},'告警与通知':{en:'Alerts & Notifications'},
  '10 分钟前':{en:'10 min ago'},'32 分钟前':{en:'32 min ago'},'1 小时前':{en:'1 hour ago'},'2 小时前':{en:'2 hours ago'},'3 小时前':{en:'3 hours ago'},'昨天':{en:'Yesterday'},
  '持续':{en:'Duration'},
  // Page: user-mgmt.html
  '用户数：5':{en:'Users: 5'},'当前角色':{en:'Current Role'},'用户 ID':{en:'User ID'},'账号标识':{en:'Account ID'},'创建时间 ⇅':{en:'Created ⇅'},'备注':{en:'Notes'},
  '10 条/页':{en:'10 / page'},'20 条/页':{en:'20 / page'},
  '修改角色':{en:'Change Role'},'修改信息':{en:'Edit Info'},'新建用户':{en:'New User'},
  '工作区昵称：':{en:'Workspace Nickname: '},'请选择角色（可多选）':{en:'Select roles (multi-select)'},'备注：':{en:'Notes: '},'标签：':{en:'Tags: '},'用户：':{en:'User: '},'默认角色：':{en:'Default Role: '},
  '登录时自动激活的角色':{en:'Role activated on login'},'次要角色权限：':{en:'Secondary Role Permissions: '},
  '开启后，当前会话的权限 = 主要角色 + 所有次要角色的权限交集。关闭后仅使用主要角色权限。':{en:'When enabled, session permissions = primary role + intersection of all secondary roles. When disabled, only primary role permissions apply.'},
  '用户名：':{en:'Username: '},'操作日志':{en:'Operation Log'},'操作人':{en:'Operator'},
  '用户标签管理':{en:'User Tag Management'},'管理用户标签定义。标签可在行权限规则中通过 @标签名 引用，运行时替换为用户的标签值。':{en:'Manage user tag definitions. Tags can be referenced in row permission rules via @tagname, replaced at runtime with user tag values.'},
  '标签名':{en:'Tag Name'},'编辑标签':{en:'Edit Tag'},'标签名：':{en:'Tag Name: '},'描述：':{en:'Description: '},
  '搜索用户名/备注':{en:'Search username/notes'},'请输入标签值':{en:'Enter tag value'},'请输入手机号':{en:'Enter phone number'},'在此工作区内显示的名称（可选）':{en:'Display name in this workspace (optional)'},'请输入备注信息':{en:'Enter notes'},'可选':{en:'Optional'},'描述（可选）':{en:'Description (optional)'},
  '请输入账号':{en:'Please enter account'},'请至少选择一个角色':{en:'Please select at least one role'},'标签名不能为空':{en:'Tag name cannot be empty'},'请输入标签名':{en:'Please enter tag name'},'标签名已存在':{en:'Tag name already exists'},
  '用户「':{en:'User "'},'确定删除用户「':{en:'Delete user "'},'确定删除标签「':{en:'Delete tag "'},
  '暂无用户标签，请先在"用户标签"中创建':{en:'No user tags. Please create one in "User Tags" first.'},
  // Page: role-perm.html
  '角色数：4':{en:'Roles: 4'},'角色 ID':{en:'Role ID'},'角色名':{en:'Role Name'},'创建时间 ⇅':{en:'Created ⇅'},'更新时间 ⇅':{en:'Updated ⇅'},
  '权限查看':{en:'View Permissions'},'修改':{en:'Modify'},'继承':{en:'Inherit'},'禁用':{en:'Disable'},
  '暂未继承任何角色':{en:'No inherited roles'},'移除':{en:'Remove'},'选择要继承的角色...':{en:'Select role to inherit...'},'暂无角色继承此角色':{en:'No roles inherit this role'},'选择要授权的角色...':{en:'Select role to authorize...'},
  '箭头表示继承方向：A → B 表示 A 继承 B 的权限':{en:'Arrow shows inheritance: A → B means A inherits B permissions'},'暂无继承关系':{en:'No inheritance relationships'},'（间接）':{en:'(indirect)'},'无关联角色':{en:'No related roles'},
  '清空':{en:'Clear All'},'暂未选择权限':{en:'No permissions selected'},
  '新建角色':{en:'New Role'},'全局权限':{en:'Global Permissions'},'对象权限':{en:'Object Permissions'},'添加对象权限':{en:'Add Object Permission'},'对象类别':{en:'Object Category'},'对象名称':{en:'Object Name'},'权限列表':{en:'Permission List'},
  '确 认':{en:'Confirm'},'企业信息':{en:'Enterprise Info'},'列权限':{en:'Column Permissions'},'行权限':{en:'Row Permissions'},'样例数据':{en:'Sample Data'},'设置行权限':{en:'Set Row Permissions'},
  '角色继承':{en:'Role Inheritance'},'继承角色':{en:'Inherited Roles'},'授权角色':{en:'Authorized Roles'},'查看':{en:'View'},'添加行权限':{en:'Add Row Permission'},'角色继承关系图':{en:'Role Inheritance Graph'},
  '+ 添加行过滤条件':{en:'+ Add Row Filter'},'行过滤规则':{en:'Row Filter Rules'},'请选择列':{en:'Select column'},'+ 添加表达式':{en:'+ Add Expression'},'选择用户标签（运行时替换为用户的标签值）':{en:'Select user tag (replaced with user tag value at runtime)'},
  '+ 添加对象权限':{en:'+ Add Object Permission'},'自动布局':{en:'Auto Layout'},
  '拖拽节点移动 · 从节点边缘拖出连线创建继承（A→B 表示 A 继承 B） · 点击连线删除 · 滚轮缩放':{en:'Drag nodes · Drag from edge to create inheritance (A→B = A inherits B) · Click connection to delete · Scroll to zoom'},
  '搜索角色名/备注':{en:'Search role name/notes'},'有继承关系的角色不允许删除':{en:'Roles with inheritance cannot be deleted'},'请输入值，输入 @ 引用用户标签':{en:'Enter value, type @ to reference user tag'},'请输入角色名':{en:'Enter role name'},'搜索对象名称':{en:'Search object name'},'搜索列名':{en:'Search column name'},
  '添加失败：会形成循环继承关系':{en:'Failed: would create circular inheritance'},'角色「':{en:'Role "'},'请至少选择一项权限':{en:'Please select at least one permission'},'请选择对象类别':{en:'Please select object category'},'请选择对象名称':{en:'Please select object name'},'请至少添加一条表达式':{en:'Please add at least one expression'},
  '确定删除此对象权限？':{en:'Delete this object permission?'},'确定删除此行权限规则？':{en:'Delete this row permission rule?'},'删除继承关系：':{en:'Delete inheritance: '},
  // Page: account/account.html
  '简体中文':{en:'Simplified Chinese'},'上传头像':{en:'Upload Avatar'},'移除头像':{en:'Remove Avatar'},'用户 ID':{en:'User ID'},'姓名':{en:'Name'},'未设置':{en:'Not Set'},'公司':{en:'Company'},'注册时间':{en:'Registered'},'最近登录':{en:'Last Login'},'手机号':{en:'Phone'},'未绑定':{en:'Not Bound'},'绑定':{en:'Bind'},'邮箱':{en:'Email'},'未关联':{en:'Not Linked'},'关联':{en:'Link'},'微信':{en:'WeChat'},'登录密码':{en:'Login Password'},'已设置':{en:'Set'},'两步验证':{en:'Two-Factor Auth'},'未开启':{en:'Not Enabled'},'登录设备':{en:'Login Devices'},'管理':{en:'Manage'},
  'UTC+08:00 （中国标准时间）':{en:'UTC+08:00 (China Standard Time)'},'UTC-05:00 （美东）':{en:'UTC-05:00 (US Eastern)'},'UTC+09:00 （日本）':{en:'UTC+09:00 (Japan)'},
  '计费中心 →':{en:'Billing →'},'注销账号':{en:'Delete Account'},'永久删除账号及所有关联数据，此操作不可恢复':{en:'Permanently delete account and all associated data. This cannot be undone.'},'导出数据':{en:'Export Data'},'导出该账号的所有操作记录和个人数据':{en:'Export all operation records and personal data for this account'},
  '退出':{en:'Leave'},'暂无工作区':{en:'No workspaces'},'默认密钥':{en:'Default Key'},'撤销':{en:'Revoke'},'暂无 API Key':{en:'No API Keys'},
  // Page: account/billing.html
  '预计可用':{en:'Est. Available'},'近一周':{en:'Last Week'},'近一个月':{en:'Last Month'},'当月':{en:'Current Month'},'按计费项':{en:'By Billing Item'},'计费项说明':{en:'Billing Item Description'},'计费类型':{en:'Billing Type'},'计费内容':{en:'Billing Content'},'计费单位':{en:'Billing Unit'},
  'SQL 运行':{en:'SQL Execution'},'AI 调用':{en:'AI Calls'},'数据存储':{en:'Data Storage'},'公网流量':{en:'Public Network Traffic'},
  '按工作区':{en:'By Workspace'},'消费趋势（条形图）':{en:'Spending Trend (Bar Chart)'},'费用构成（饼图）':{en:'Cost Breakdown (Pie Chart)'},'月账单概览':{en:'Monthly Bill Overview'},
  '总消耗 (credit)':{en:'Total Cost (credit)'},'用量消耗 (credit)':{en:'Usage Cost (credit)'},'月费订阅 (credit)':{en:'Monthly Subscription (credit)'},'计费中':{en:'In Progress'},'已出账':{en:'Billed'},'合计':{en:'Total'},
  '统计周期':{en:'Statistics Period'},'日':{en:'Day'},'统计项':{en:'Statistics Item'},'计费项':{en:'Billing Item'},'工作区':{en:'Workspace'},
  '待结算':{en:'Pending'},'已结算':{en:'Settled'},'重置':{en:'Reset'},'账单号':{en:'Bill No.'},'消费时间':{en:'Consumption Time'},'单价':{en:'Unit Price'},'用量':{en:'Usage'},'消费 (credit)':{en:'Cost (credit)'},
  '交易单号':{en:'Transaction No.'},'交易时间':{en:'Transaction Time'},'收支类型':{en:'Income/Expense'},'交易类型':{en:'Transaction Type'},'交易渠道':{en:'Transaction Channel'},'交易金额':{en:'Transaction Amount'},'发票':{en:'Invoice'},
  '当月订阅合计：':{en:'Monthly Subscription Total: '},'按月扣费的增值服务，开通后每月 1 日自动从 Credit 余额扣除':{en:'Monthly billed add-on services, auto-deducted from Credit balance on the 1st of each month'},'已开通':{en:'Active'},'未开通':{en:'Inactive'},'credit/月':{en:'credit/month'},'专属支持':{en:'Dedicated Support'},'企业身份认证':{en:'Enterprise Identity'},
  '启用告警':{en:'Enable Alert'},'提醒阈值':{en:'Warning Threshold'},'紧急阈值':{en:'Critical Threshold'},'新账号消费速率限制':{en:'New Account Rate Limit'},'月度消费上限':{en:'Monthly Spending Cap'},'cr/月':{en:'cr/month'},'余额不足时暂停服务':{en:'Suspend Service on Zero Balance'},'启用自动充值':{en:'Enable Auto Top-up'},'触发阈值':{en:'Trigger Threshold'},
  '接收邮箱：':{en:'Recipient Email: '},'企业微信 / 飞书 / 钉钉':{en:'WeCom / Feishu / DingTalk'},'需先在账号管理中绑定':{en:'Please bind in Account Settings first'},
  '发票类型':{en:'Invoice Type'},'增值税普通发票':{en:'VAT General Invoice'},'增值税专用发票':{en:'VAT Special Invoice'},'发票抬头':{en:'Invoice Title'},'税号':{en:'Tax ID'},
  '充值 Credit':{en:'Top Up Credits'},'当前余额':{en:'Current Balance'},'实时到账':{en:'Instant'},'大额度充值':{en:'Large Amount Top-up'},'选择充值包（购买后永不过期）':{en:'Select package (never expires)'},'支付金额':{en:'Payment Amount'},'获得 Credit':{en:'Credits Received'},'确认充值':{en:'Confirm Top-up'},'支付宝充值':{en:'Alipay Top-up'},
  '搜索账单号':{en:'Search bill no.'},'搜索交易单号':{en:'Search transaction no.'},'输入金额':{en:'Enter amount'},'请选择充值包或输入金额':{en:'Please select a package or enter amount'},
  '确定要关闭「':{en:'Confirm closing "'},'确定要开通「':{en:'Confirm enabling "'},
  '总消耗 Credit':{en:'Total Credits'},
  // Page: knowledge-base.html
  '创建知识库':{en:'Create Knowledge Base'},'选择数据':{en:'Select Data'},'从 Catalog 中选择目录、库、卷或表添加到知识库（不含算子和模型）':{en:'Select directories, databases, volumes or tables from Catalog to add to knowledge base (excluding operators and models)'},'已选 0 项':{en:'0 items selected'},
  '添加数据':{en:'Add Data'},'浏览数据':{en:'Browse data'},'已选内容':{en:'Selected items'},'数据明细':{en:'Data details'},'搜索当前目录':{en:'Search this directory'},'返回上一级':{en:'Back one level'},'全选当前范围':{en:'Select current scope'},'资源类型':{en:'Resource type'},'不支持':{en:'Unsupported'},'暂未选择数据':{en:'No data selected'},'请从上方文件树选择 Catalog、Database、数据表或 Volume':{en:'Select a Catalog, Database, table, or Volume from the file tree above'},
  '表详情':{en:'Table Details'},'抽样数据':{en:'Sample Data'},'暂无统计信息':{en:'No statistics available'},'在此编写或调整针对该表的查询语句，实际执行以环境配置为准。':{en:'Write or adjust queries for this table. Actual execution depends on environment config.'},
  '处理结果':{en:'Processing Results'},'原文件预览':{en:'File Preview'},'分段结果':{en:'Chunking Results'},'新建分段':{en:'New Chunk'},'第 1 / 1 页':{en:'Page 1 / 1'},
  '新增通配符':{en:'New Wildcard'},'通配符可以帮助模型更好地理解和处理用户问题中的变量部分。':{en:'Wildcards help the model better understand and handle variable parts in user questions.'},'每个通配符可以包含多个枚举值，用于匹配用户问题中的变量部分':{en:'Each wildcard can contain multiple enum values to match variable parts in user questions'},
  '新增问法':{en:'New Query Pattern'},'添加问法描述和对应的SQL语句，帮助模型学习正确的查询逻辑。':{en:'Add query descriptions and corresponding SQL to help the model learn correct query logic.'},'请确保SQL语句语法正确，这将作为模型学习的标准答案':{en:'Ensure SQL syntax is correct. This will serve as the standard answer for model learning.'},
  '新增 SQL 结果集定义':{en:'New SQL Result Set Definition'},'配置表和列补充说明':{en:'Configure Table & Column Notes'},'表补充说明':{en:'Table Notes'},'列补充说明':{en:'Column Notes'},'启用':{en:'Enable'},
  '新增逻辑解释':{en:'New Logic Explanation'},'选择关联的表':{en:'Select Related Table'},'系统智能判断':{en:'System Auto-detect'},'将由模型根据用户问题内容进行智能判断选择性生效。':{en:'The model will intelligently determine applicability based on user question content.'},'全局类型':{en:'Global Type'},'全局型业务逻辑对全部用户问题生效。':{en:'Global business logic applies to all user questions.'},
  '新增指标':{en:'New Metric'},'用户提问中可能使用的其他叫法':{en:'Other names users may use when asking'},'可执行的 SQL 聚合表达式，将注入到生成的 SQL 中':{en:'Executable SQL aggregate expression, injected into generated SQL'},
  '聚合（SUM/COUNT/AVG 等）':{en:'Aggregate (SUM/COUNT/AVG etc.)'},'派生（多指标组合计算）':{en:'Derived (multi-metric calculation)'},'过滤（带 WHERE 条件的指标）':{en:'Filtered (metric with WHERE condition)'},
  '新增维度':{en:'New Dimension'},'帮助模型理解维度的取值范围，不需要穷举':{en:'Help the model understand dimension value ranges, no need to enumerate all'},'新增表关系':{en:'New Table Relation'},
  '结果集名称':{en:'Result Set Name'},'通配符':{en:'Wildcard'},'数据值':{en:'Data Value'},'问法描述':{en:'Query Description'},'预期正确执行的完整SQL':{en:'Expected Correct SQL'},'指标名称':{en:'Metric Name'},'同义词':{en:'Synonyms'},'计算公式':{en:'Formula'},'关联表':{en:'Related Table'},'语义模式':{en:'Semantic Mode'},'维度名称':{en:'Dimension Name'},'所属表.列':{en:'Table.Column'},'枚举值样本':{en:'Enum Value Samples'},'主表':{en:'Primary Table'},'JOIN 类型':{en:'JOIN Type'},'JOIN 条件':{en:'JOIN Condition'},'说明':{en:'Description'},
  '问题':{en:'Question'},'期望命中片段':{en:'Expected Hit Segment'},'评估报告':{en:'Evaluation Report'},'基于 Golden Set 评估当前检索策略的效果':{en:'Evaluate current retrieval strategy based on Golden Set'},'暂无评估记录，请先添加 Golden Set 后运行评估':{en:'No evaluation records. Add Golden Set first and run evaluation.'},'自动优化':{en:'Auto Optimize'},'请先运行评估确认基线分数，再启动自动优化':{en:'Run evaluation first to confirm baseline score before starting auto-optimization'},
  '选择一个对话或新建对话开始提问':{en:'Select a conversation or start a new one'},'选择要查询的知识库（可多选）':{en:'Select knowledge bases to query (multi-select)'},
  '搜索知识库名称或描述...':{en:'Search knowledge base name or description...'},'请输入':{en:'Please enter'},'搜索业务逻辑关键词':{en:'Search business logic keywords'},'搜索表名':{en:'Search table name'},'搜索结果集名称':{en:'Search result set name'},'搜索通配符':{en:'Search wildcards'},'搜索问法':{en:'Search query patterns'},'搜索指标名称或同义词':{en:'Search metric name or synonyms'},'搜索维度名称':{en:'Search dimension name'},
  '输入你的问题...':{en:'Enter your question...'},'输入查询 SQL':{en:'Enter query SQL'},'请输入通配符名称':{en:'Enter wildcard name'},'请输入内容，回车提交':{en:'Enter content, press Enter to submit'},
  '请输入结果集名称':{en:'Enter result set name'},'请输入 SQL':{en:'Enter SQL'},'请输入描述':{en:'Enter description'},'请输入表补充说明':{en:'Enter table notes'},'请输入业务逻辑解释':{en:'Enter business logic explanation'},'请输入列补充说明':{en:'Enter column notes'},'请输入知识库名称':{en:'Enter knowledge base name'},
  '删除对话':{en:'Delete Conversation'},'请先确认至少一条 Golden Set 记录':{en:'Please confirm at least one Golden Set record'},'请至少选择一个适用场景':{en:'Please select at least one applicable scenario'},'最多选择 20 个文件':{en:'Maximum 20 files'},
  '确定从数据源中移除「':{en:'Remove from data source: "'},'确定删除该分段？':{en:'Delete this chunk?'},'确定删除此逻辑解释？':{en:'Delete this logic explanation?'},'确定删除此 SQL 结果集？':{en:'Delete this SQL result set?'},'确定删除此通配符？':{en:'Delete this wildcard?'},'确定删除此问法？':{en:'Delete this query pattern?'},'确定删除知识库「':{en:'Delete knowledge base "'},'确定删除此对话？':{en:'Delete this conversation?'},
  '✓ 已确认':{en:'✓ Confirmed'},'待确认':{en:'Pending Confirmation'},'综合命中率':{en:'Overall Hit Rate'},'评估条数':{en:'Evaluation Count'},'当前策略':{en:'Current Strategy'},'命中':{en:'Hit'},'得分':{en:'Score'},'Top-1 片段':{en:'Top-1 Segment'},
  '选择上传类型':{en:'Select Upload Type'},'非结构化文件':{en:'Unstructured Files'},'非结构化文件（最多 20 个，单个 ≤20MB）':{en:'Unstructured files (max 20, each ≤20MB)'},'结构化数据（最多 20 个，单个 ≤20MB）':{en:'Structured data (max 20, each ≤20MB)'},
  '下一步':{en:'Next Step'},'上一步':{en:'Previous Step'},'从本地选择文件上传，快速创建知识库':{en:'Upload files from local to quickly create knowledge base'},'从 Catalog 选择':{en:'Select from Catalog'},'选择 MOI 已处理的数据表或文档':{en:'Select MOI processed data tables or documents'},'← 更换数据源':{en:'← Change Data Source'},'暂无对话记录':{en:'No conversation history'},
  '总行数：':{en:'Total Rows: '},'更多列级统计可在接入分析服务后展示。':{en:'More column-level statistics available after connecting analytics service.'},
  '暂无智能体关联此知识库':{en:'No agents linked to this knowledge base'},'请描述知识库包含的内容及范围说明。建议详细说明可用的实体列表、文档主题、关键字等信息。':{en:'Describe the content and scope of the knowledge base. Include available entities, document topics, keywords, etc.'},'填写知识库备注有利于在多知识库的情况下更准确地查找数据':{en:'Knowledge base notes help find data more accurately when multiple knowledge bases exist'},
  // Page: app-dev/index.html
  '对话历史':{en:'Chat History'},'技能库':{en:'Skill Library'},'工具库':{en:'Tool Library'},'新智能体':{en:'New Agent'},'智能体描述':{en:'Agent Description'},'点击头像可更换表情或图片':{en:'Click avatar to change emoji or image'},'系统提示词':{en:'System Prompt'},'AI 改写':{en:'AI Rewrite'},'改写':{en:'Rewrite'},
  '添加工作区':{en:'Add Workspace'},'创建可能需要一点时间':{en:'Creation may take a moment'},'创建':{en:'Create'},'选择知识库':{en:'Select Knowledge Base'},'选择技能':{en:'Select Skill'},'暂无智能体':{en:'No agents'},'有什么可以帮助你？':{en:'How can I help you?'},'开始你的第一个智能体':{en:'Start your first agent'},
  '思考中…':{en:'Thinking…'},'应用':{en:'Apply'},
  '智能体名称':{en:'Agent Name'},'提示词':{en:'Prompt'},'全部智能体':{en:'All Agents'},'触发设置':{en:'Trigger Settings'},'触发方式':{en:'Trigger Method'},'执行轨迹':{en:'Execution Trace'},
  '暂无知识库':{en:'No knowledge bases'},'暂无技能':{en:'No skills'},
  '没有找到匹配表情':{en:'No matching emoji found'},'按分类筛选':{en:'Filter by Category'},'内置':{en:'Built-in'},'市场':{en:'Market'},
  '没有匹配的技能':{en:'No matching skills'},'筛选条件':{en:'Filter Criteria'},'清除全部':{en:'Clear All'},'来源':{en:'Source'},'分类':{en:'Category'},'已安装':{en:'Installed'},'安装':{en:'Install'},
  '系统算子':{en:'System Operators'},'自定义算子':{en:'Custom Operators'},'没有匹配的工具':{en:'No matching tools'},'引入算子':{en:'Import Operator'},'外部接入':{en:'External Integration'},'MCP 协议 / HTTP API':{en:'MCP Protocol / HTTP API'},'引入':{en:'Add'},
  'MCP Server 地址':{en:'MCP Server Address'},'工具名称':{en:'Tool Name'},'请求方式 & URL':{en:'Request Method & URL'},'Body 模板':{en:'Body Template'},
  '算子信息':{en:'Operator Info'},'引入版本':{en:'Import Version'},'版本状态':{en:'Version Status'},'协议':{en:'Protocol'},'请求':{en:'Request'},'使用此工具的技能':{en:'Skills using this tool'},'参数定义':{en:'Parameter Definition'},'参数名':{en:'Param Name'},'必填':{en:'Required'},'是':{en:'Yes'},'输入 / 输出':{en:'Input / Output'},'输入':{en:'Input'},'输出':{en:'Output'},'使用示例':{en:'Usage Example'},
  '批量删除':{en:'Batch Delete'},'审批策略':{en:'Approval Policy'},'当前策略':{en:'Current Policy'},'审批项：':{en:'Approval Items: '},'高级配置':{en:'Advanced Config'},'搜索技能':{en:'Search Skills'},'搜索工具':{en:'Search Tools'},'搜索知识库':{en:'Search Knowledge Bases'},
  '总记忆':{en:'Total Memories'},'长期记忆':{en:'Long-term Memory'},'用户加强':{en:'User Reinforced'},'系统记忆':{en:'System Memory'},'暂无记忆':{en:'No memories'},'Memoria 记忆':{en:'Memoria Memory'},
  '系统内置模型':{en:'System Built-in Models'},'自定义模型':{en:'Custom Models'},'头像':{en:'Avatar'},'添加自定义模型':{en:'Add Custom Model'},'模型配置':{en:'Model Config'},'导入设置':{en:'Import Settings'},'视觉模型':{en:'Vision Model'},'OCR 模型':{en:'OCR Model'},'文本嵌入模型':{en:'Text Embedding Model'},'重排序模型':{en:'Reranking Model'},
  '已发布':{en:'Published'},'分享':{en:'Share'},'发布':{en:'Publish'},
  '批准执行':{en:'Approve'},'拒绝':{en:'Reject'},'对话数':{en:'Conversations'},'技能数':{en:'Skills'},'任务数':{en:'Tasks'},'用户反馈':{en:'User Feedback'},'已绑定技能':{en:'Bound Skills'},'+ 添加':{en:'+ Add'},'解绑':{en:'Unbind'},'暂无绑定技能':{en:'No bound skills'},'已绑定知识库':{en:'Bound Knowledge Bases'},'暂无绑定知识库':{en:'No bound knowledge bases'},'编辑提示词':{en:'Edit Prompt'},'未设置系统提示词':{en:'No system prompt set'},'+ 添加模型':{en:'+ Add Model'},'选择模型':{en:'Select Model'},
  '加强记忆':{en:'Reinforce Memory'},'引用':{en:'Quote'},'划除':{en:'Strike'},'点赞':{en:'Like'},'点踩':{en:'Dislike'},'反馈状态':{en:'Feedback Status'},'用户问题':{en:'User Question'},'回复结果':{en:'Reply Result'},
  '请输入智能体名称':{en:'Please enter agent name'},'请输入模型名称':{en:'Please enter model name'},'请输入工具名称':{en:'Please enter tool name'},'请输入 MCP Server 地址':{en:'Please enter MCP Server address'},'请输入 API URL':{en:'Please enter API URL'},'请输入测试问题':{en:'Please enter test question'},
  '删除此对话？':{en:'Delete this conversation?'},'确定删除智能体「':{en:'Delete agent "'},'确定删除此自定义技能？':{en:'Delete this custom skill?'},'确定删除此知识库？':{en:'Delete this knowledge base?'},
  // Page: app-dev/app-builder.html
  '客户管理':{en:'Customer Management'},'审批流程':{en:'Approval Workflow'},'数据看板':{en:'Dashboard'},'客服系统':{en:'Customer Service'},'库存管理':{en:'Inventory Management'},'从零开始':{en:'From Scratch'},
  '客户':{en:'Customers'},'订单':{en:'Orders'},'历史订单':{en:'Order History'},'订单号':{en:'Order No.'},'日期':{en:'Date'},'产品':{en:'Product'},'金额':{en:'Amount'},'累计订单':{en:'Total Orders'},'累计金额':{en:'Total Amount'},'健康度':{en:'Health Score'},'优':{en:'Excellent'},
  '未命名应用':{en:'Untitled App'},'桌面端':{en:'Desktop'},'平板':{en:'Tablet'},'手机':{en:'Mobile'},
};
function t(key){if(MOI_LANG==='zh')return key;var e=MOI_I18N[key];return(e&&e[MOI_LANG])||key;}
function switchLang(lang){MOI_LANG=lang;localStorage.setItem('moi_lang',lang);window.location.reload();}
// Hide bottom-section (用户/API管理/MCP) in data mode sidebar
(function() {
  function hideBottomSection() {
    var bs = document.querySelector('.sidebar .bottom-section');
    if (bs) { bs.style.display = 'none'; }
    var dividers = document.querySelectorAll('.sidebar .section-divider');
    if (dividers.length) dividers[dividers.length - 1].style.display = 'none';
  }
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', hideBottomSection);
  } else {
    hideBottomSection();
  }
})();

// Snowflake-style sidebar: flat menu with group titles
(function() {
  var sidebarMenu = [
    { group: '', items: [
      { label: '概览', href: 'dashboard/dashboard.html', icon: '<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round"><rect x="1" y="1" width="6" height="6" rx="1"/><rect x="9" y="1" width="6" height="3" rx="1"/><rect x="9" y="6" width="6" height="3" rx="1"/><rect x="1" y="9" width="6" height="6" rx="1"/><rect x="9" y="11" width="6" height="4" rx="1"/></svg>' }
    ]},
    { group: '数据连接', items: [
      { label: '连接器', href: 'data-connection/connector.html', icon: '<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round"><rect x="1" y="3" width="14" height="4" rx="1.5"/><rect x="1" y="9" width="14" height="4" rx="1.5"/><circle cx="4" cy="5" r="0.7" fill="currentColor"/><circle cx="4" cy="11" r="0.7" fill="currentColor"/></svg>' },
      { label: '数据载入', href: 'data-connection/data-import.html', icon: '<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round"><path d="M8 2v9M5 8l3 3 3-3"/><path d="M2 13h12"/></svg>' },
      { label: '数据导出', href: 'data-connection/data-export.html', icon: '<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round"><path d="M8 11V2M5 5l3-3 3 3"/><path d="M2 13h12"/></svg>' }
    ]},
    { group: '数据处理', items: [
      { label: '工作流', href: 'data-processing/workflow.html', icon: '<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round"><rect x="1" y="4" width="5" height="4" rx="1"/><rect x="10" y="2" width="5" height="4" rx="1"/><rect x="10" y="9" width="5" height="4" rx="1"/><path d="M6 6h4M6 6V5h4M6 6v5h4"/></svg>' },
      { label: 'Notebook', href: 'data-processing/notebook.html', icon: '<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="2" width="10" height="12" rx="1"/><path d="M6 2v12M3.2 5h2M3.2 8h2M3.2 11h2"/></svg>' },
      { label: 'SQL 编辑器', href: 'data-processing/sql-editor.html', icon: '<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round"><path d="M2 3h12M2 7h9M2 11h5"/><circle cx="13" cy="11" r="2.5"/></svg>' },
      { label: '数据看板', href: 'data-processing/data-dashboard.html', icon: '<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round"><rect x="1.5" y="1.5" width="13" height="13" rx="2"/><path d="M4.5 11V7.5M8 11V5M11.5 11V9"/></svg>' }
    ]},
    { group: '资源中心', items: [
      { label: 'Catalog', href: 'resource-center/catalog.html', icon: '<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round"><ellipse cx="8" cy="4" rx="6" ry="2.5"/><path d="M2 4v4c0 1.38 2.69 2.5 6 2.5s6-1.12 6-2.5V4"/><path d="M2 8v4c0 1.38 2.69 2.5 6 2.5s6-1.12 6-2.5V8"/></svg>' },
      { label: '计算资源', href: 'resource-center/compute.html', icon: '<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round"><rect x="1" y="4" width="6" height="7" rx="1"/><rect x="9" y="4" width="6" height="7" rx="1"/><path d="M4 2v2M12 2v2M4 11v2M12 11v2"/></svg>' },
      { label: '数据分享', href: 'resource-center/data-share.html', icon: '<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round"><path d="M10 2l3 3-3 3"/><path d="M13 5H7"/><path d="M6 14l-3-3 3-3"/><path d="M3 11h6"/></svg>' },
      { label: '知识库', href: 'resource-center/knowledge-base.html', icon: '<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round"><path d="M2 2h8l3 3v8a1 1 0 0 1-1 1H2a1 1 0 0 1-1-1V3a1 1 0 0 1 1-1z"/><path d="M4 7h7M4 10h5"/></svg>' }
    ]},
    { group: '监测', items: [
      { label: 'SQL 历史', href: 'monitor/sql-history.html', icon: '<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round"><circle cx="8" cy="8" r="6"/><path d="M8 5v3l2 2"/></svg>' },
      { label: '作业', href: 'monitor/job.html', icon: '<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round"><rect x="2" y="3" width="5" height="10" rx="1"/><rect x="9" y="6" width="5" height="7" rx="1"/></svg>' },
      { label: '日志', href: 'monitor/audit-log.html', icon: '<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round"><path d="M2 3h12M2 6h12M2 9h8M2 12h5"/></svg>' }
    ]},
    { group: '告警', items: [
      { label: '告警规则', href: 'monitor/alert-rules.html', icon: '<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round"><path d="M4 6a4 4 0 0 1 8 0c0 2 1 3.5 1.5 4.5H2.5C3 10 4 8 4 6z"/><path d="M6 11v.5a2 2 0 0 0 4 0V11"/></svg>' },
      { label: '通知对象', href: 'monitor/notify-targets.html', icon: '<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round"><circle cx="8" cy="5" r="3"/><path d="M2 14c0-3 2.69-4.5 6-4.5s6 1.5 6 4.5"/></svg>' },
      { label: '告警记录', href: 'monitor/alert-records.html', icon: '<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round"><rect x="2" y="2" width="12" height="12" rx="1.5"/><path d="M2 5h12"/><path d="M5 2v3"/><path d="M11 2v3"/></svg>' }
    ]},
    { group: '用户权限', items: [
      { label: '用户管理', href: 'user-perm/user-mgmt.html', icon: '<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round"><circle cx="6" cy="5" r="3"/><path d="M1 14c0-2.5 2.24-4 5-4"/><circle cx="12" cy="5" r="2"/><path d="M10 14c0-2 1-3 2-3s2 1 2 3"/></svg>' },
      { label: '角色权限', href: 'user-perm/role-perm.html', icon: '<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round"><path d="M8 1L2 4v4c0 3.5 2.5 6.5 6 7.5 3.5-1 6-4 6-7.5V4L8 1z"/><path d="M6 8l1.5 1.5L10 6.5"/></svg>' }
    ]}
  ];

  function buildSidebar() {
    var sidebar = document.querySelector('.sidebar');
    if (!sidebar) return;
    var menuSection = sidebar.querySelector('.menu-section');
    if (!menuSection) return;

    // Detect base path from current page
    var path = window.location.pathname;
    var basePath = '';
    if (path.indexOf('/data-connection/') !== -1 || path.indexOf('/data-processing/') !== -1 ||
        path.indexOf('/resource-center/') !== -1 || path.indexOf('/user-perm/') !== -1 ||
        path.indexOf('/monitor/') !== -1 || path.indexOf('/account/') !== -1) {
      basePath = '../';
    } else if (path.indexOf('/dashboard/') !== -1) {
      basePath = '../';
    }

    var html = '<div class="sidebar-toggle"><button onclick="toggleSidebarCollapse()" title="展开"><svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M9 18l6-6-6-6"/></svg></button></div>';
    sidebarMenu.forEach(function(group) {
      if (group.group) html += '<div class="menu-group-title">' + t(group.group) + '</div>';
      group.items.forEach(function(item) {
        var href = item.href === '#' ? '#' : basePath + item.href;
        var isActive = item.href !== '#' && path.split('/').pop() === item.href.split('/').pop();
        var label = t(item.label);
        html += '<a class="menu-item' + (isActive ? ' active' : '') + '" href="' + href + '" data-label="' + label + '">';
        html += '<span class="icon">' + item.icon + '</span>';
        html += '<span class="label">' + label + '</span>';
        // Add collapse toggle inline with dashboard
        if (item.label === '概览') {
          html += '<span class="sidebar-collapse-btn" onclick="event.preventDefault();event.stopPropagation();toggleSidebarCollapse()" title="折叠/展开"><svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M15 18l-6-6 6-6"/></svg></span>';
        }
        html += '</a>';
      });
    });

    menuSection.innerHTML = html;
    // Remove old sub-menus, section-divider, bottom-section
    var old = sidebar.querySelectorAll('.sub-menu, .section-divider, .bottom-section');
    old.forEach(function(el) { el.remove(); });

    // Remove legacy bottom icon bar (API Key 已并入账户-访问凭据，SQL 连接串移至 SQL 编辑器)
    var existingBottom = sidebar.querySelector('.bottom-icons');
    if (existingBottom) existingBottom.remove();
    // Restore collapsed state
    if (localStorage.getItem('moi_sidebar_collapsed') === '1') {
      sidebar.classList.add('collapsed');
      var main = document.querySelector('.main');
      if (main) main.style.marginLeft = '52px';
    }
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', buildSidebar);
  } else {
    buildSidebar();
  }
})();

// === Build Top Bar dynamically for all pages ===
function buildTopBar() {
  var topBar = document.querySelector('.top-bar');
  if (!topBar) return;
  // Skip if top bar already has content (e.g. special pages)
  if (topBar.children.length > 0) return;

  // Detect base path
  var path = window.location.pathname;
  // file:// previews include the full filesystem path. Only inspect the app
  // route after /html/ so a username such as /Users/admin/ is not mistaken
  // for the product's /admin/ route.
  if (window.location.protocol === 'file:') {
    var htmlRootIndex = path.lastIndexOf('/html/');
    if (htmlRootIndex !== -1) path = path.slice(htmlRootIndex + 5);
  }
  var basePath = '';
  if (path.indexOf('/dashboard/') !== -1 || path.indexOf('/data-connection/') !== -1 ||
      path.indexOf('/data-processing/') !== -1 || path.indexOf('/resource-center/') !== -1 ||
      path.indexOf('/user-perm/') !== -1 || path.indexOf('/monitor/') !== -1 || path.indexOf('/account/') !== -1 ||
      path.indexOf('/app-dev/') !== -1 || path.indexOf('/taas/') !== -1 ||
      path.indexOf('/matrixone/') !== -1 ||
      path.indexOf('/admin/') !== -1) {
    basePath = '../';
  }

  // Account pages (billing, account management) skip workspace selector and mode switch
  var isAccountPage = path.indexOf('/account/') !== -1;
  // Genesis pages skip workspace selector, mode switch, and Genesis button
  var isTaasPage = path.indexOf('/taas/') !== -1;
  // MatrixOne 云数据库控制台：独立产品，跟 Genesis 一样只留 Logo + 产品切换器 + 头像
  var isMatrixOnePage = path.indexOf('/matrixone/') !== -1;
  // Admin pages skip workspace selector, mode switch, and Genesis button
  var isAdminPage = path.indexOf('/admin/') !== -1;
  // Detect if current page is agent mode (app-dev) or apps mode
  var isAppMode = path.indexOf('/app-dev/index.html') !== -1;
  var isAppsMode = path.indexOf('/app-dev/apps.html') !== -1;

  // Mode switch: three modes — apps, agent, data
  var appsActiveClass = isAppsMode ? ' active' : '';
  var agentActiveClass = isAppMode ? ' active' : '';
  var dataActiveClass = (!isAppMode && !isAppsMode) ? ' active' : '';
  var appsOnclick = isAppsMode ? '' : ' onclick="switchToAppsMode(\'' + basePath + '\')"';
  var agentOnclick = isAppMode ? '' : ' onclick="switchToAgentMode(\'' + basePath + '\')"';
  var dataOnclick = (isAppMode || isAppsMode) ? ' onclick="switchToDataMode(\'' + basePath + '\')"' : '';

  // Admin pages get a minimal top bar
  if (isAdminPage) {
    topBar.innerHTML = ''
      + '<a class="logo" href="' + basePath + 'website/index.html" style="text-decoration:none;color:inherit"><img src="' + basePath + 'images/logo-blue.svg" alt="MOI"></a>'
      + '<div class="spacer"></div>'
      + '<div class="right-actions">'
      +   '<div class="user-avatar-wrap" id="consoleAvatarWrap">'
      +     '<div class="user-avatar" id="consoleAvatarBtn" title="账户">A</div>'
      +     '<div class="user-popover" id="consolePopover">'
      +       '<div class="popover-header"><div class="popover-avatar">A</div><div class="popover-info"><div class="popover-name">admin</div><div class="popover-email">admin@matrixorigin.cn</div></div></div>'
      +       '<div class="popover-menu"><button class="popover-item danger" onclick="consoleLogout()">退出登录</button></div>'
      +     '</div>'
      +   '</div>'
      + '</div>';
    return;
  }

  // 九宫格 app launcher（跨产品切换）
  var curProduct = isTaasPage ? 'genesis' : (isMatrixOnePage ? 'matrixone' : 'workspace');
  function appTile(key, logo, name, sub, onclick, disabled, mode) {
    var isCur = key === curProduct;
    // mode='wordmark'：方框里放完整官方字标（如 MatrixOne 的 logo-matrixone.svg），下面仍有名称 + 介绍
    var ico = mode === 'wordmark'
      ? '<span class="applauncher-ico"><img class="al-wm" src="' + basePath + 'images/' + logo + '" alt="' + name + '"></span>'
      : '<span class="applauncher-ico"><img src="' + basePath + 'images/' + logo + '" alt="' + name + '"></span>';
    return '<div class="applauncher-tile' + (isCur ? ' current' : '') + (disabled ? ' disabled' : '') + '"'
      + (!isCur && onclick ? ' onclick="' + onclick + '"' : '') + '>'
      + ico
      + '<span class="applauncher-name">' + name + '</span>'
      + '<span class="applauncher-sub">' + sub + '</span>'
      + '</div>';
  }
  var gridIcon = '<svg class="al-grid" width="21" height="21" viewBox="0 0 24 24"><defs><linearGradient id="alGrad" x1="3" y1="3" x2="21" y2="21" gradientUnits="userSpaceOnUse"><stop stop-color="#004af0"/><stop offset="1" stop-color="#00d4aa"/></linearGradient></defs><circle cx="5" cy="5" r="1.9"/><circle cx="12" cy="5" r="1.9"/><circle cx="19" cy="5" r="1.9"/><circle cx="5" cy="12" r="1.9"/><circle cx="12" cy="12" r="1.9"/><circle cx="19" cy="12" r="1.9"/><circle cx="5" cy="19" r="1.9"/><circle cx="12" cy="19" r="1.9"/><circle cx="19" cy="19" r="1.9"/></svg>';
  var appLauncherHtml = ''
    + '<div class="dropdown" id="appLauncherWrap">'
    +   '<div class="action-btn applauncher-btn" onclick="toggleDropdown(\'appLauncherDD\')" title="切换产品 · 工作区 / Genesis / MatrixOne">' + gridIcon + '</div>'
    +   '<div class="dropdown-content right" id="appLauncherDD" style="min-width:348px;padding:12px;background:#fff;-webkit-backdrop-filter:none;backdrop-filter:none;border:1px solid #ebedf2;box-shadow:0 12px 34px rgba(20,33,64,0.15)">'
    +     '<div class="applauncher-grid">'
    +       appTile('workspace', 'logo-workspace.svg', 'AI Studio', '从数据到 AI 应用的端到端生成', 'gotoWorkspace(\'' + basePath + '\')', false)
    +       appTile('genesis', 'logo-genesis.svg', 'Genesis', 'TaaS 模型服务', 'gotoGenesis()', false)
    +       appTile('matrixone', 'logo-matrixone.svg', 'MatrixOne', '超融合数据库', 'gotoMatrixOne()', false, 'wordmark')
    +     '</div>'
    +   '</div>'
    + '</div>';

  topBar.innerHTML = ''
    + '<a class="logo" href="' + basePath + 'website/index.html" style="text-decoration:none;color:inherit"><img src="' + basePath + 'images/logo-blue.svg" alt="MOI"></a>'
    + (isAccountPage || isTaasPage || isMatrixOnePage ? '' : '<div class="ws-mode-group">'
    +   '<div class="ws-selector">'
    +     '<div class="ws-trigger" onclick="toggleWsPanel()">'
    +       '<span class="ws-dot"></span>'
    +       '<span id="wsCurrentName">默认工作区</span>'
    +       '<span class="ws-switch-icon"><svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round" stroke-linejoin="round"><path d="M6 9l6 6 6-6"/></svg></span>'
    +     '</div>'
    +     '<div class="ws-panel" id="wsPanel"></div>'
    +   '</div>'
    +   '<span class="ws-mode-sep"></span>'
    +   '<div class="mode-switch" id="modeSwitch">'
    +     '<div class="mode-btn' + appsActiveClass + '" data-mode="apps"' + appsOnclick + '><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><rect x="2" y="3" width="20" height="18" rx="3"/><path d="M2 9h20"/><circle cx="6" cy="6" r="1" fill="currentColor" stroke="none"/><circle cx="9.5" cy="6" r="1" fill="currentColor" stroke="none"/><path d="M7 14h10M7 17.5h6"/></svg>' + t('应用') + '</div>'
    +     '<div class="mode-btn' + agentActiveClass + '" data-mode="agent"' + agentOnclick + '><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="18" height="18" rx="3"/><circle cx="9" cy="10" r="1.5" fill="currentColor" stroke="none"/><circle cx="15" cy="10" r="1.5" fill="currentColor" stroke="none"/><path d="M8 15c1 1.5 3 2 4 2s3-.5 4-2"/></svg>' + t('智能体') + '</div>'
    +     '<div class="mode-btn' + dataActiveClass + '" data-mode="data"' + dataOnclick + '><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><ellipse cx="12" cy="6" rx="8" ry="3"/><path d="M4 6v6c0 1.66 3.58 3 8 3s8-1.34 8-3V6"/><path d="M4 12v6c0 1.66 3.58 3 8 3s8-1.34 8-3v-6"/></svg>' + t('数据') + '</div>'
    +   '</div>'
    + '</div>')
    + '<div class="spacer"></div>'
    + '<div class="right-actions">'
    // 运营入口:签到邀请(奖励中心弹窗),仅前台产品与账户页顶栏,管理端不显示;红点 = 今日未签提示
    + (isAdminPage || isTaasPage || isMatrixOnePage ? '' :
        '<button id="rwTopBtn" onclick="openRewardsModal()" title="每日签到 · 邀请有礼" '
        + 'style="position:relative;display:inline-flex;align-items:center;gap:6px;height:32px;padding:0 13px;margin-right:4px;border:1px solid rgba(0,74,240,0.18);border-radius:16px;background:rgba(0,74,240,0.05);color:#004af0;font-size:12.5px;font-weight:500;cursor:pointer;white-space:nowrap" '
        + 'onmouseover="this.style.borderColor=\'#004af0\';this.style.background=\'rgba(0,74,240,0.09)\'" onmouseout="this.style.borderColor=\'rgba(0,74,240,0.18)\';this.style.background=\'rgba(0,74,240,0.05)\'">'
        + '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M20 12v10H4V12M2 7h20v5H2zM12 22V7M12 7a3 3 0 1 0-3-3c0 1.5 1.5 3 3 3zM12 7a3 3 0 1 1 3-3c0 1.5-1.5 3-3 3z"/></svg>'
        + '签到邀请'
        + '<span id="rwTopDot" style="position:absolute;top:-2px;right:2px;width:8px;height:8px;border-radius:50%;background:#ff4d4f;border:1.5px solid #fff"></span>'
        + '</button>')
    +   appLauncherHtml
    +   '<div class="user-avatar-wrap" id="consoleAvatarWrap">'
    +     '<div class="user-avatar" id="consoleAvatarBtn" title="账户">U</div>'
    +     '<div class="user-popover" id="consolePopover">'
    +       '<div class="popover-header">'
    +         '<div class="popover-avatar">U</div>'
    +         '<div class="popover-info">'
    +           '<div class="popover-name" id="consolePopName">用户</div>'
    +           '<div class="popover-email" id="consolePopEmail"></div>'
    +         '</div>'
    +       '</div>'
    +       '<div class="popover-menu">'
    +         '<button class="popover-item" onclick="window.open(\'' + basePath + 'account/account.html\',\'_blank\')">账号管理</button>'
    +         '<button class="popover-item" onclick="window.open(\'' + basePath + 'account/credentials.html\',\'_blank\')">访问凭据</button>'
    +         '<button class="popover-item popover-between" onclick="window.open(\'' + basePath + 'account/billing.html\',\'_blank\')"><span>计费中心</span><span class="pop-credit" id="popCredit">— cr</span></button>'
    +         '<button class="popover-item popover-between" onclick="openRewardsModal()"><span>奖励中心</span><span class="pop-meta">签到 · 邀请</span></button>'
    +         '<button class="popover-item" onclick="window.open(\'' + basePath + 'admin/index.html\',\'_blank\')">管理后台</button>'
    +         '<div class="popover-sep"></div>'
    +         '<button class="popover-item popover-between" onclick="sessionStorage.setItem(\'moi_portal_stay\',\'1\');location.href=\'' + basePath + 'website/portal.html\'" title="回到服务站点选择页,切换到其他云"><span>服务站点</span><span class="pop-meta">' + (function(){ try { var cs = JSON.parse(localStorage.getItem('moi_current_site') || 'null'); return (cs && cs.cloud) ? cs.cloud : '阿里云'; } catch (e) { return '阿里云'; } })() + ' ›</span></button>'
    +         '<button class="popover-item popover-between" onclick="event.stopPropagation();switchLang(MOI_LANG===\'zh\'?\'en\':\'zh\')"><span>语言</span><span class="pop-meta">' + (MOI_LANG === 'en' ? 'English' : '简体中文') + '</span></button>'
    +         '<button class="popover-item popover-between"><span>时区</span><span class="pop-meta">UTC+08:00</span></button>'
    +         '<button class="popover-item danger" onclick="consoleLogout()">退出登录</button>'
    +       '</div>'
    +     '</div>'
    +   '</div>'
    + '</div>';

  // Create workspace modal if it doesn't exist (skip for account pages)
  if (!isAccountPage && !isTaasPage && !isAdminPage && !isMatrixOnePage && (!document.getElementById('addWsModal') || !document.querySelector('#addWsModal .ws-modal'))) {
    var modalEl = document.getElementById('addWsModal');
    if (!modalEl) {
      modalEl = document.createElement('div');
      modalEl.id = 'addWsModal';
      document.body.appendChild(modalEl);
    }
    modalEl.className = 'ws-modal-overlay';
    modalEl.setAttribute('onclick', 'if(event.target===this)closeAddWsModal()');
    modalEl.innerHTML = '<div class="ws-modal">'
      + '<div class="ws-modal-header"><h3>添加工作区</h3><button class="ws-modal-close" onclick="closeAddWsModal()">✕</button></div>'
      + '<div class="ws-modal-tip">创建可能需要一点时间，可在工作区切换列表中查看状态</div>'
      + '<input class="ws-modal-input" id="addWsName" type="text" placeholder="请输入名称" maxlength="50" onkeydown="if(event.key===\'Enter\')confirmAddWs()">'
      + '<div class="ws-modal-hint">工作区名称将用于标识您的工作空间，建议使用有意义的名称</div>'
      + '<div style="margin-top:12px;font-size:13px;color:rgba(0,0,0,0.65)">地区</div>'
      + '<select class="ws-modal-input" id="addWsRegion" style="margin-top:6px;cursor:pointer">'
      +   '<option>华东-1</option><option>华北-1</option><option>华南-1</option>'
      + '</select>'
      + '<div class="ws-modal-hint">工作区归属当前服务站点的所选地区，创建后不可变更；数据与运行不出站点</div>'
      + '<div class="ws-modal-footer">'
      +   '<button class="ws-btn ws-btn-cancel" onclick="closeAddWsModal()">取 消</button>'
      +   '<button class="ws-btn ws-btn-create" onclick="confirmAddWs()">创 建</button>'
      + '</div></div>';
  }

  // Re-init workspace panel after building top bar (skip for account pages)
  if (!isAccountPage && !isTaasPage && !isAdminPage && !isMatrixOnePage) renderWsPanel();
}

// Save current data-mode page path before switching to other modes
function saveDataPageAndSwitch(basePath, target) {
  var path = window.location.pathname;
  var match = path.match(/(dashboard|data-connection|data-processing|resource-center|user-perm)\/[^?#]+/);
  if (match) {
    localStorage.setItem('moi_last_data_page', match[0]);
  }
  localStorage.setItem('moi_mode', target);
  location.href = basePath + (target === 'apps' ? 'app-dev/apps.html' : 'app-dev/index.html');
}

// Switch to apps mode
function switchToAppsMode(basePath) {
  var path = window.location.pathname;
  // Save data page if coming from data mode
  if (path.indexOf('/app-dev/') === -1) {
    var match = path.match(/(dashboard|data-connection|data-processing|resource-center|user-perm)\/[^?#]+/);
    if (match) localStorage.setItem('moi_last_data_page', match[0]);
  }
  localStorage.setItem('moi_mode', 'apps');
  location.href = basePath + 'app-dev/apps.html';
}

// Switch to agent mode
function switchToAgentMode(basePath) {
  var path = window.location.pathname;
  // Save data page if coming from data mode
  if (path.indexOf('/app-dev/') === -1) {
    var match = path.match(/(dashboard|data-connection|data-processing|resource-center|user-perm)\/[^?#]+/);
    if (match) localStorage.setItem('moi_last_data_page', match[0]);
  }
  localStorage.setItem('moi_mode', 'app');
  location.href = basePath + 'app-dev/index.html';
}

// Switch from app/agent mode back to data mode (restore last data page)
function switchToDataMode(basePath) {
  var last = localStorage.getItem('moi_last_data_page');
  localStorage.setItem('moi_mode', 'data');
  location.href = last ? basePath + last : basePath + 'dashboard/dashboard.html';
}

// Auto-call buildTopBar on DOMContentLoaded
(function() {
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', function() { buildTopBar(); applyPageI18n(); startI18nObserver(); initMomo(); });
  } else {
    buildTopBar();
    applyPageI18n();
    startI18nObserver();
    initMomo();
  }
})();

// ===== MOMO 智能助手 (Genie-style) =====
var MOMO_MSGS_KEY = 'moi_momo_msgs';
var MOMO_OPEN_KEY = 'moi_momo_open';
var MOMO_ALWAYS_KEY = 'moi_momo_always_approve';
var momoMsgs = [];

function momoBasePath() {
  var path = window.location.pathname;
  var dirs = ['/dashboard/','/data-connection/','/data-processing/','/resource-center/','/user-perm/','/monitor/','/account/','/app-dev/','/taas/','/matrixone/','/admin/'];
  for (var i = 0; i < dirs.length; i++) { if (path.indexOf(dirs[i]) !== -1) return '../'; }
  return '';
}
// 多会话存储：moi_momo_sessions = [{id,title,msgs,ts}]，moi_momo_cur = 当前会话 id
var MOMO_SESS_KEY = 'moi_momo_sessions', MOMO_CUR_KEY = 'moi_momo_cur';
function momoSessTitle(msgs) {
  for (var i = 0; i < (msgs || []).length; i++) {
    if (msgs[i].role === 'user' && msgs[i].text) { var t = String(msgs[i].text).replace(/\s+/g, ' ').trim(); return t.length > 18 ? t.slice(0, 18) + '…' : t; }
  }
  return '新会话';
}
function momoSessionsAll() {
  var arr; try { arr = JSON.parse(localStorage.getItem(MOMO_SESS_KEY) || 'null'); } catch(e) { arr = null; }
  if (!arr || !arr.length) {
    var old = []; try { old = JSON.parse(localStorage.getItem(MOMO_MSGS_KEY) || '[]'); } catch(e) {}
    arr = [{ id: 's' + Date.now(), title: momoSessTitle(old), msgs: old, ts: Date.now() }];
    localStorage.setItem(MOMO_SESS_KEY, JSON.stringify(arr));
    localStorage.setItem(MOMO_CUR_KEY, arr[0].id);
  }
  return arr;
}
function momoCurId() {
  var arr = momoSessionsAll(), cur = localStorage.getItem(MOMO_CUR_KEY);
  if (!cur || !arr.some(function(s) { return s.id === cur; })) { cur = arr[0].id; localStorage.setItem(MOMO_CUR_KEY, cur); }
  return cur;
}
function momoLoad() {
  var arr = momoSessionsAll(), cur = momoCurId(), s = arr.filter(function(x) { return x.id === cur; })[0];
  momoMsgs = (s && s.msgs) ? s.msgs : [];
}
function momoSave() {
  try {
    var arr = momoSessionsAll(), cur = momoCurId(), found = false;
    for (var i = 0; i < arr.length; i++) { if (arr[i].id === cur) { arr[i].msgs = momoMsgs; arr[i].title = momoSessTitle(momoMsgs); arr[i].ts = Date.now(); found = true; break; } }
    if (!found) arr.unshift({ id: cur, title: momoSessTitle(momoMsgs), msgs: momoMsgs, ts: Date.now() });
    localStorage.setItem(MOMO_SESS_KEY, JSON.stringify(arr));
  } catch(e) {}
}
function momoNewSession() {
  var arr = momoSessionsAll(), id = 's' + Date.now();
  arr.unshift({ id: id, title: '新会话', msgs: [], ts: Date.now() });
  localStorage.setItem(MOMO_SESS_KEY, JSON.stringify(arr));
  localStorage.setItem(MOMO_CUR_KEY, id);
  momoMsgs = [];
  document.body.classList.remove('momo-sessions-on', 'momo-settings-on');
  momoRender();
  setTimeout(function() { var i = document.getElementById('momoInput'); if (i) i.focus(); }, 50);
}
function momoSwitchSession(id) {
  localStorage.setItem(MOMO_CUR_KEY, id); momoLoad();
  document.body.classList.remove('momo-sessions-on'); momoRender();
}
function momoDeleteSession(id, ev) {
  if (ev) ev.stopPropagation();
  var arr = momoSessionsAll().filter(function(s) { return s.id !== id; });
  if (!arr.length) arr = [{ id: 's' + Date.now(), title: '新会话', msgs: [], ts: Date.now() }];
  localStorage.setItem(MOMO_SESS_KEY, JSON.stringify(arr));
  if (localStorage.getItem(MOMO_CUR_KEY) === id) { localStorage.setItem(MOMO_CUR_KEY, arr[0].id); momoLoad(); momoRender(); }
  momoRenderSessions();
}
function momoToggleSessions() {
  var on = document.body.classList.toggle('momo-sessions-on');
  document.body.classList.remove('momo-settings-on');
  if (on) momoRenderSessions();
}
function momoRenderSessions() {
  var el = document.getElementById('momoSessions'); if (!el) return;
  var arr = momoSessionsAll().slice().sort(function(a, b) { return (b.ts || 0) - (a.ts || 0); }), cur = momoCurId();
  var rows = arr.map(function(s) {
    return '<div class="msx-row' + (s.id === cur ? ' cur' : '') + '" onclick="momoSwitchSession(\'' + s.id + '\')">'
      + '<svg class="msx-ico" viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/></svg>'
      + '<span class="msx-title">' + momoEsc(s.title || '新会话') + '</span>'
      + '<button class="msx-del" title="删除会话" onclick="momoDeleteSession(\'' + s.id + '\',event)"><svg viewBox="0 0 24 24" width="13" height="13" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round" stroke-linejoin="round"><path d="M18 6L6 18M6 6l12 12"/></svg></button>'
      + '</div>';
  }).join('');
  el.innerHTML = '<div class="msx-head"><span>历史会话</span><button class="msx-new" onclick="momoNewSession()">＋ 新会话</button></div><div class="msx-list">' + rows + '</div>';
}
function momoEsc(s) { return (s == null ? '' : String(s)).replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;'); }
function momoChipEsc(s) { return String(s).replace(/\\/g,'\\\\').replace(/'/g,"\\'"); }

function initMomo() {
  if (document.getElementById('momoPanel')) return;
  var bp = momoBasePath();
  // MOMO 是 MOI 平台全局功能：右下角悬浮按钮（缩起=logo，点开=对话框），覆盖所有页面
  var fab = document.createElement('button');
  fab.className = 'momo-fab'; fab.id = 'momoBtn'; fab.title = 'MOMO 智能助手';
  fab.onclick = momoToggle;
  fab.innerHTML = '<img src="' + bp + 'images/momo.svg" alt="MOMO">';
  document.body.appendChild(fab);
  var panel = document.createElement('div');
  panel.className = 'momo-panel'; panel.id = 'momoPanel';
  panel.innerHTML = ''
    + '<div class="momo-head">'
    +   '<div class="momo-head-title"><img src="' + bp + 'images/momo.svg" alt="MOMO"><div><div class="momo-name">MOMO</div><div class="momo-sub">MOI 智能助手</div></div></div>'
    +   '<div class="momo-head-actions">'
    +     '<button class="momo-icon-btn" title="新会话" onclick="momoNewSession()"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M12 20h9"/><path d="M16.5 3.5a2.12 2.12 0 0 1 3 3L7 19l-4 1 1-4Z"/></svg></button>'
    +     '<button class="momo-icon-btn" id="momoHistBtn" title="历史会话" onclick="momoToggleSessions()"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M3 3v5h5"/><path d="M3.05 13A9 9 0 1 0 6 5.3L3 8"/><path d="M12 7v5l3.5 2"/></svg></button>'
    +     '<button class="momo-icon-btn" id="momoSetBtn" title="设置" onclick="momoToggleSettings()"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 1 1-2.83-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z"/></svg></button>'
    +     '<button class="momo-icon-btn" title="关闭" onclick="momoToggle()"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M18 6L6 18M6 6l12 12"/></svg></button>'
    +   '</div>'
    + '</div>'
    + '<div class="momo-sessions" id="momoSessions"></div>'
    + '<div class="momo-resize" title="拖动调整大小"></div>'
    + '<div class="momo-body" id="momoBody"></div>'
    + '<div class="momo-settings" id="momoSettings"></div>'
    + '<div class="momo-input-wrap">'
    +   '<textarea class="momo-input" id="momoInput" rows="1" placeholder="问 MOMO，或让它帮你操作 MOI…" onkeydown="momoInputKey(event)" oninput="momoAutoGrow(this)"></textarea>'
    +   '<button class="momo-send" id="momoSendBtn" onclick="momoSendOrStop()" title="发送"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M22 2L11 13M22 2l-7 20-4-9-9-4 20-7z"/></svg></button>'
    + '</div>'
    + '<div class="momo-foot">'
    +   '<a class="momo-foot-link" href="https://docs.matrixorigin.cn/zh/m1intelligence/" target="_blank"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round"><path d="M2 3h6a4 4 0 0 1 4 4v14a3 3 0 0 0-3-3H2z"/><path d="M22 3h-6a4 4 0 0 0-4 4v14a3 3 0 0 1 3-3h7z"/></svg>使用文档</a>'
    +   '<span class="momo-foot-dot"></span>'
    +   '<a class="momo-foot-link" href="mailto:contact@matrixorigin.cn"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round"><rect x="2" y="4" width="20" height="16" rx="2"/><path d="M22 6L12 13 2 6"/></svg>联系我们</a>'
    + '</div>';
  document.body.appendChild(panel);
  momoLoad();
  momoInitPrefs();
  momoDragResizeInit();
  momoRender();
  if (localStorage.getItem(MOMO_OPEN_KEY) === '1') document.body.classList.add('momo-open');
}

function momoToggle() {
  var open = document.body.classList.toggle('momo-open');
  localStorage.setItem(MOMO_OPEN_KEY, open ? '1' : '0');
  if (open) { momoApplyView(); momoRender(); setTimeout(function(){ var i = document.getElementById('momoInput'); if (i) i.focus(); }, 60); }
  else { document.body.classList.remove('momo-settings-on'); }
}
function momoClear() { momoMsgs = []; momoSave(); momoRender(); }

// ===== MOMO 设置视图（⋯ 打开）：审批模式 / 面板视图 / MCP 服务 =====
var MOMO_MCP_SERVERS = [
  { key: 'slack', name: 'Slack', slug: 'slack', desc: '搜索消息、读取频道和对话' },
  { key: 'm365', name: 'Microsoft 365', slug: 'microsoft365', desc: '搜索 SharePoint、Outlook 邮件、Teams 消息' },
  { key: 'gdrive', name: 'Google Drive', slug: 'googledrive', desc: '搜索和读取 Docs、Sheets、Slides' },
  { key: 'gcal', name: 'Google Calendar', slug: 'googlecalendar', desc: '查看日程和会议' },
  { key: 'gmail', name: 'Gmail', slug: 'gmail', desc: '搜索和读取邮件' },
  { key: 'glean', name: 'Glean', slug: 'glean', desc: '搜索企业知识库' },
  { key: 'github', name: 'GitHub', slug: 'github', desc: '搜索代码、Issues、PR' },
  { key: 'atlassian', name: 'Atlassian', slug: 'atlassian', desc: '搜索 Jira 工单、Confluence 页面' }
];
function momoToggleSettings() {
  var on = document.body.classList.toggle('momo-settings-on');
  var btn = document.getElementById('momoSetBtn'); if (btn) btn.classList.toggle('on', on);
  if (on) momoRenderSettings();
}
function momoApplyView() {
  var v = localStorage.getItem('moi_momo_view') || 'side';
  document.body.classList.toggle('momo-view-side', v === 'side');
  document.body.classList.toggle('momo-view-float', v !== 'side');
  momoApplyFloatBox();
}
// 悬浮态：应用已保存的位置/大小；侧边态：清掉内联让 CSS 停靠生效
function momoApplyFloatBox() {
  var panel = document.getElementById('momoPanel'); if (!panel) return;
  if (document.body.classList.contains('momo-view-float')) {
    var box; try { box = JSON.parse(localStorage.getItem('moi_momo_floatbox') || 'null'); } catch(e) { box = null; }
    if (box) { panel.style.left = box.l + 'px'; panel.style.top = box.t + 'px'; panel.style.width = box.w + 'px'; panel.style.height = box.h + 'px'; panel.style.right = 'auto'; panel.style.bottom = 'auto'; }
  } else {
    panel.style.left = panel.style.top = panel.style.width = panel.style.height = panel.style.right = panel.style.bottom = '';
  }
}
// 悬浮态：拖动头部移动、拖左上角手柄拉伸
function momoDragResizeInit() {
  var panel = document.getElementById('momoPanel'); if (!panel) return;
  var head = panel.querySelector('.momo-head'), rez = panel.querySelector('.momo-resize');
  function isFloat() { return document.body.classList.contains('momo-view-float'); }
  function toBox() {
    var r = panel.getBoundingClientRect();
    panel.style.left = r.left + 'px'; panel.style.top = r.top + 'px';
    panel.style.width = r.width + 'px'; panel.style.height = r.height + 'px';
    panel.style.right = 'auto'; panel.style.bottom = 'auto';
  }
  function saveBox() { try { localStorage.setItem('moi_momo_floatbox', JSON.stringify({ l: parseInt(panel.style.left), t: parseInt(panel.style.top), w: parseInt(panel.style.width), h: parseInt(panel.style.height) })); } catch(e) {} }
  function drag(e) {
    if (!isFloat() || e.target.closest('.momo-icon-btn') || e.button !== 0) return;
    e.preventDefault(); toBox();
    var sx = e.clientX, sy = e.clientY, sl = parseInt(panel.style.left), st = parseInt(panel.style.top);
    function mv(ev) {
      panel.style.left = Math.max(6, Math.min(window.innerWidth - 90, sl + ev.clientX - sx)) + 'px';
      panel.style.top = Math.max(6, Math.min(window.innerHeight - 56, st + ev.clientY - sy)) + 'px';
    }
    function up() { document.removeEventListener('mousemove', mv); document.removeEventListener('mouseup', up); document.body.classList.remove('momo-dragging'); saveBox(); }
    document.body.classList.add('momo-dragging'); document.addEventListener('mousemove', mv); document.addEventListener('mouseup', up);
  }
  function resize(e) {
    if (!isFloat() || e.button !== 0) return;
    e.preventDefault(); e.stopPropagation(); toBox();
    var sx = e.clientX, sy = e.clientY, sl = parseInt(panel.style.left), st = parseInt(panel.style.top), sw = parseInt(panel.style.width), sh = parseInt(panel.style.height);
    function mv(ev) {
      var nw = Math.max(300, Math.min(760, sw + (sx - ev.clientX))), nh = Math.max(360, Math.min(window.innerHeight - 30, sh + (sy - ev.clientY)));
      panel.style.width = nw + 'px'; panel.style.height = nh + 'px';
      panel.style.left = (sl + sw - nw) + 'px'; panel.style.top = (st + sh - nh) + 'px';
    }
    function up() { document.removeEventListener('mousemove', mv); document.removeEventListener('mouseup', up); document.body.classList.remove('momo-dragging'); saveBox(); }
    document.body.classList.add('momo-dragging'); document.addEventListener('mousemove', mv); document.addEventListener('mouseup', up);
  }
  if (head) head.addEventListener('mousedown', drag);
  if (rez) rez.addEventListener('mousedown', resize);
}
function momoSetApproval(mode) {
  localStorage.setItem('moi_momo_approval', mode);
  if (mode === 'auto') localStorage.setItem(MOMO_ALWAYS_KEY, '1'); else localStorage.removeItem(MOMO_ALWAYS_KEY);
  momoRenderSettings();
}
function momoSetView(v) { localStorage.setItem('moi_momo_view', v); momoApplyView(); momoRenderSettings(); }
function momoMcpToggle(key, btn) {
  var k = 'moi_mcp_' + key, now = localStorage.getItem(k) === '1';
  if (now) { localStorage.removeItem(k); btn.classList.remove('connected'); btn.textContent = '连接'; }
  else { localStorage.setItem(k, '1'); btn.classList.add('connected'); btn.textContent = '已连接'; }
}
function momoInitPrefs() {
  if (!localStorage.getItem('moi_momo_approval')) { localStorage.setItem('moi_momo_approval', 'auto'); localStorage.setItem(MOMO_ALWAYS_KEY, '1'); }
  if (!localStorage.getItem('moi_momo_view')) localStorage.setItem('moi_momo_view', 'side');
  momoApplyView();
}
function momoRenderSettings() {
  var el = document.getElementById('momoSettings'); if (!el) return;
  var approval = localStorage.getItem('moi_momo_approval') || 'auto';
  var view = localStorage.getItem('moi_momo_view') || 'side';
  function opt(active, onclick, title, desc) {
    return '<button class="ms-opt' + (active ? ' on' : '') + '" onclick="' + onclick + '">'
      + '<span class="ms-radio"></span>'
      + '<span class="ms-opt-text"><span class="ms-opt-title">' + title + '</span><span class="ms-opt-desc">' + desc + '</span></span>'
      + '</button>';
  }
  var mcp = MOMO_MCP_SERVERS.map(function(s) {
    var c = localStorage.getItem('moi_mcp_' + s.key) === '1';
    return '<div class="ms-mcp-row">'
      + '<img class="ms-mcp-ico" src="https://cdn.simpleicons.org/' + s.slug + '" alt="" onerror="this.style.visibility=\'hidden\'">'
      + '<span class="ms-mcp-meta"><span class="ms-mcp-name">' + s.name + '</span><span class="ms-mcp-desc">' + s.desc + '</span></span>'
      + '<button class="ms-mcp-btn' + (c ? ' connected' : '') + '" onclick="momoMcpToggle(\'' + s.key + '\',this)">' + (c ? '已连接' : '连接') + '</button>'
      + '</div>';
  }).join('');
  el.innerHTML = ''
    + '<button class="ms-back" onclick="momoToggleSettings()"><svg viewBox="0 0 24 24" width="15" height="15" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M15 18l-6-6 6-6"/></svg>返回对话</button>'
    + '<div class="ms-section"><div class="ms-title">审批模式</div><div class="ms-note">控制 MOMO 调用工具（执行代码、建文件、改资产）前是否需要你确认。</div>'
    +   opt(approval === 'ask', 'momoSetApproval(\'ask\')', 'Ask First · 每次询问', '每次调用工具前弹出确认，你点批准才执行。适合谨慎场景。')
    +   opt(approval === 'auto', 'momoSetApproval(\'auto\')', 'Auto-approve · 自动执行', '工具自动执行，仅在高风险操作时拦截。适合日常，体验更流畅。')
    + '</div>'
    + '<div class="ms-section"><div class="ms-title">面板视图</div>'
    +   opt(view === 'float', 'momoSetView(\'float\')', '悬浮', '右下角浮动小窗，不占用页面空间。')
    +   opt(view === 'side', 'momoSetView(\'side\')', '侧边', '停靠右侧、缩窄页面内容，MOMO 独占一栏（类似 Databricks）。')
    + '</div>'
    + '<div class="ms-section"><div class="ms-title">MCP 服务</div><div class="ms-note">通过 MCP 协议把外部服务接入对话，连接后可直接搜索 / 读取其内容。</div>'
    +   '<div class="ms-mcp-list">' + mcp + '</div>'
    +   '<button class="ms-add-server" onclick="alert(\'添加自定义 MCP Server（原型）\')"><svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 5v14M5 12h14"/></svg>添加 Server</button>'
    + '</div>'
    + '<div class="ms-section"><div class="ms-title">模型与接口</div>'
    +   '<button class="ms-link-btn" onclick="momoSettings()">配置 API 地址 / Key / 模型 ›</button>'
    + '</div>';
}
function momoAutoGrow(el) { el.style.height = 'auto'; el.style.height = Math.min(el.scrollHeight, 120) + 'px'; }
function momoInputKey(e) {
  // 中文/日文等输入法组词时按回车是"确认候选词"，不应触发发送（e.isComposing / keyCode 229）
  if (e.key === 'Enter' && !e.shiftKey && !e.isComposing && e.keyCode !== 229) { e.preventDefault(); momoSend(); }
}
function momoScroll() { var b = document.getElementById('momoBody'); if (b) setTimeout(function(){ b.scrollTop = b.scrollHeight; }, 0); }

function momoRender() {
  var body = document.getElementById('momoBody');
  if (!body) return;
  var html = '';
  if (!momoMsgs.length) {
    html = momoWelcome();
  } else {
    momoMsgs.forEach(function(m, idx) {
      if (m.role === 'user') {
        html += '<div class="momo-row user"><div class="momo-bubble user">' + momoEsc(m.text) + '</div></div>';
      } else {
        html += '<div class="momo-row momo">';
        html += '<div class="momo-bubble momo">';
        if (m.html) html += '<div class="momo-text">' + m.html + '</div>';
        if (m.action) html += momoActionCard(m, idx);
        if (m.content && m.done && !m.action) html += momoActionsBar(m, idx);
        if (m.chips && m.chips.length) html += momoChips(m.chips);
        html += '</div></div>';
      }
    });
  }
  body.innerHTML = html;
  momoScroll();
}

function momoWelcome() {
  return '<div class="momo-welcome">'
    + '<img class="momo-welcome-logo" src="' + momoBasePath() + 'images/momo.svg" alt="MOMO">'
    + '<div class="momo-welcome-title">嗨，我是 MOMO 👋</div>'
    + '<div class="momo-welcome-sub">我是 MOI 的产品助手，随时为您解答产品问题——功能怎么用、概念是什么、怎么选方案都可以问我；也能帮您直接完成操作，比如创建工作流、配置知识库，涉及变更的操作会先请您确认再执行。</div>'
    + momoChips(['MOI 是什么，解决什么问题？','MOI 有哪些数据处理节点？','帮我创建一个 RAG 工作流','知识库怎么配置语义？'])
    + '</div>';
}
function momoChips(chips) {
  var h = '<div class="momo-chips">';
  chips.forEach(function(c){ h += '<button class="momo-chip" onclick="momoChip(\'' + momoChipEsc(c) + '\')">' + momoEsc(c) + '</button>'; });
  return h + '</div>';
}
function momoActionCard(m, idx) {
  var a = m.action;
  var tagTxt = { pending:'待你确认', executing:'执行中', done:'已完成', rejected:'已拒绝' };
  var tagCls = { pending:'wait', executing:'run', done:'ok', rejected:'no' };
  var h = '<div class="momo-action"><div class="momo-action-head"><span class="momo-lang">' + momoEsc((a.lang||'code').toUpperCase()) + '</span>';
  h += '<span class="momo-action-tag ' + (tagCls[a.status]||'wait') + '">' + (tagTxt[a.status]||'待你确认') + '</span></div>';
  h += '<pre class="momo-code">' + momoEsc(a.code) + '</pre>';
  if (a.status === 'pending') {
    h += '<div class="momo-action-btns">'
      + '<button class="momo-ab primary" onclick="momoAction(' + idx + ',\'approve\')">同意</button>'
      + '<button class="momo-ab" onclick="momoAction(' + idx + ',\'reject\')">拒绝</button>'
      + '<button class="momo-ab ghost" onclick="momoAction(' + idx + ',\'always\')">始终同意</button>'
      + '</div>';
  } else if (a.status === 'executing') {
    h += '<div class="momo-action-status run">● 执行中…</div>';
  } else if (a.status === 'done') {
    h += '<div class="momo-action-status ok">✓ ' + momoEsc(a.doneText||'已完成') + (a.auto ? ' <span class="momo-auto">（已按「始终同意」自动执行）</span>' : '') + '</div>';
  } else if (a.status === 'rejected') {
    h += '<div class="momo-action-status no">✕ 已拒绝该操作</div>';
  }
  return h + '</div>';
}

function momoChip(text) { var i = document.getElementById('momoInput'); if (i) i.value = text; momoSend(); }
function momoSend() {
  if (momoBusy) return;
  var inp = document.getElementById('momoInput');
  if (!inp) return;
  var text = (inp.value || '').trim();
  if (!text) return;
  inp.value = ''; momoAutoGrow(inp);
  momoMsgs.push({ role:'user', text:text });

  if (momoCfgComplete()) {
    // 已配置模型：基于 moi-product-design.md 作答（流式，可中途停止）
    var bot = { role:'momo', html:'<span class="momo-cursor"></span>', content:'' };
    momoMsgs.push(bot);
    momoSave(); momoRender();
    momoAbort = (typeof AbortController !== 'undefined') ? new AbortController() : null;
    momoSetBusy(true);
    momoEnsureDoc().then(function(doc) {
      var msgs = [{ role:'system', content: momoSystemPrompt(doc) }];
      momoMsgs.forEach(function(m) {
        if (m.role === 'user') msgs.push({ role:'user', content:m.text });
        else if (m.role === 'momo' && m.content) msgs.push({ role:'assistant', content:m.content });
      });
      return momoLLM(msgs, function(partial) {
        bot.content = partial;
        var mi = partial.indexOf('```moi-action');
        if (mi !== -1) {
          var pre = partial.slice(0, mi).trim();
          bot.html = (pre ? momoMd(pre) : '') + '<div style="font-size:12px;color:rgba(0,0,0,0.42);margin-top:6px">⏳ 正在准备操作…</div>';
        } else {
          bot.html = momoMd(partial) + '<span class="momo-cursor"></span>';
        }
        momoRender();
      }, momoAbort ? momoAbort.signal : null);
    }).then(function(full) {
      var parsed = momoParseAction(full || '');
      if (parsed.action) {
        bot.content = parsed.text || '';
        bot.html = parsed.text ? momoMd(parsed.text) : '';
        bot.action = parsed.action;
        if (localStorage.getItem(MOMO_ALWAYS_KEY) === '1') { bot.action.status = 'done'; bot.action.auto = true; }
      } else {
        bot.content = full || '（未返回内容）';
        bot.html = momoMd(bot.content);
      }
      bot.done = true;
      momoSave(); momoRender();
      if (full && !parsed.action) momoSuggestFollowups(bot, text);
    }).catch(function(err) {
      if (err && (err.name === 'AbortError' || err.message === 'aborted' || err.message === 'The user aborted a request.')) {
        bot.html = (bot.content ? momoMd(bot.content) + ' ' : '') + '<span class="momo-stopped">（已停止）</span>';
        if (bot.content) bot.done = true;
      } else {
        bot.content = '';
        bot.html = '<p style="color:#cf1322;margin:0">⚠️ ' + momoEsc(err.message) + '</p>'
          + '<p style="color:rgba(0,0,0,0.45);font-size:12px;margin:6px 0 0">点右上角 ⚙️ 检查地址 / Key / 模型；若是 CORS 报错，请换支持浏览器跨域的接口或本地模型。</p>';
      }
      momoSave(); momoRender();
    }).then(function() {
      momoSetBusy(false); momoAbort = null;
    });
  } else {
    // 未配置模型：演示回复（脚本化）
    var reply = momoReply(text);
    reply.role = 'momo';
    if (reply.action && localStorage.getItem(MOMO_ALWAYS_KEY) === '1') { reply.action.status = 'done'; reply.action.auto = true; }
    momoMsgs.push(reply);
    momoSave(); momoRender();
  }
}
function momoAction(idx, act) {
  var m = momoMsgs[idx];
  if (!m || !m.action) return;
  if (act === 'reject') { m.action.status = 'rejected'; momoSave(); momoRender(); return; }
  if (act === 'always') { localStorage.setItem(MOMO_ALWAYS_KEY, '1'); }
  m.action.status = 'executing'; momoSave(); momoRender();
  setTimeout(function(){ m.action.status = 'done'; momoSave(); momoRender(); }, 800);
}

function momoReply(q) {
  if (/接入|外部|连接器|数据库|connector|接数据|接数/.test(q)) {
    return {
      html: '<p>接入一个外部数据库分三步：</p>'
        + '<ol><li>到 <b>数据连接 · 连接器</b> 新建对应类型的连接器（MySQL / PostgreSQL / Oracle…），填连接信息。</li>'
        + '<li>在 <b>Catalog</b> 里把它登记为<b>外部目录</b>（数据留源库，不搬迁）。</li>'
        + '<li>之后在 <b>SQL 编辑器 · 外部目录</b> 模式下，用该库原生方言直接查。</li></ol>',
      chips: ['帮我建一个 PostgreSQL 连接器','外部目录和数据载入有什么区别？']
    };
  }
  if (/rag|工作流|workflow|流程|建.*流/.test(q)) {
    return {
      html: '<p>好的，我帮你创建一个 <b>RAG 文档准备</b> 工作流，链路：数据读取 → 文档解析 → 分段 → 文本嵌入 → 数据存储。将提交下面这个 API：</p>',
      action: {
        lang:'json', status:'pending',
        doneText:'已创建工作流「RAG 文档准备」，含 5 个节点。去 数据处理 · 工作流 即可打开编辑。',
        code: 'POST /api/v1/workflows\n{\n  "name": "RAG 文档准备",\n  "nodes": [\n    { "type": "dataReadVolume", "volume": "vol_docs" },\n    { "type": "docParse" },\n    { "type": "chunk", "size": 512, "overlap": 64 },\n    { "type": "embed", "model": "bge-m3" },\n    { "type": "dataSaveTable", "table": "kb_chunks" }\n  ],\n  "edges": [[0,1],[1,2],[2,3],[3,4]]\n}'
      }
    };
  }
  if (/营收|财务|利润|报表|查询|查一下|跑.*sql|gmv|指标|营业额/.test(q)) {
    return {
      html: '<p>我在 <b>财务报表库（PostgreSQL）</b> 上帮你跑这段查询，取近 30 天各 BU 营收：</p>',
      action: {
        lang:'sql', status:'pending',
        doneText:'查询完成，返回 6 行（华东 1240 万、华南 980 万、华北 760 万…），已为你打开结果。',
        code: "SELECT business_unit,\n       SUM(revenue)::numeric(18,2) AS revenue\nFROM reporting.daily_financial_summary\nWHERE report_date >= CURRENT_DATE - INTERVAL '30 days'\nGROUP BY business_unit\nORDER BY revenue DESC;"
      }
    };
  }
  if (/节点|处理|算子|node|能做什么|功能/.test(q)) {
    return {
      html: '<p>MOI 工作流的处理节点分四组：</p>'
        + '<ul><li><b>数据 IO</b>：读取 / 存储 / 导出</li>'
        + '<li><b>智能处理</b>：文档·图片·音视频解析、分段、嵌入、清洗、信息提取、AI 推理</li>'
        + '<li><b>代码处理</b>：SQL 处理、Python 处理</li>'
        + '<li><b>流程控制</b>：条件分支、汇合判断、子工作流、工作流触发</li></ul>',
      chips: ['帮我建一个 RAG 工作流']
    };
  }
  return {
    html: '<p>我可以<b>解答 MOI 使用问题</b>，也能<b>直接帮你操作</b>——建工作流、运行 SQL、接数据源等。涉及操作我会先把 API/SQL 给你过目，你同意了再执行。试试：</p>',
    chips: ['怎么接入一个外部数据库？','帮我建一个 RAG 工作流','查财务报表库近 30 天营收','MOI 有哪些数据处理节点？']
  };
}

// ===== MOMO · LLM（基于 docs/moi-product-design.md 的真实问答） =====
// 模型配置只保存在浏览器 localStorage，不写入代码、不进 Git。
function momoCfg() {
  return {
    base: (localStorage.getItem('momo_api_base') || '').trim(),
    key:  (localStorage.getItem('momo_api_key')  || '').trim(),
    model:(localStorage.getItem('momo_model')    || '').trim()
  };
}
function momoCfgComplete() { var c = momoCfg(); return !!(c.base && c.key && c.model); }

// 生成态 + 停止控制
var momoBusy = false;
var momoAbort = null;
var MOMO_ICON_SEND = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M22 2L11 13M22 2l-7 20-4-9-9-4 20-7z"/></svg>';
var MOMO_ICON_STOP = '<svg viewBox="0 0 24 24" fill="currentColor"><rect x="6.5" y="6.5" width="11" height="11" rx="2"/></svg>';
function momoSendOrStop() { if (momoBusy) momoStop(); else momoSend(); }
function momoStop() { if (momoAbort) { try { momoAbort.abort(); } catch (e) {} } }
function momoSetBusy(b) {
  momoBusy = b;
  var btn = document.getElementById('momoSendBtn');
  if (!btn) return;
  if (b) { btn.classList.add('stop'); btn.title = '停止生成'; btn.innerHTML = MOMO_ICON_STOP; }
  else { btn.classList.remove('stop'); btn.title = '发送'; btn.innerHTML = MOMO_ICON_SEND; }
}

var _momoDocPromise = null;
function momoEnsureDoc() {
  if (_momoDocPromise) return _momoDocPromise;
  if (location.protocol === 'file:') {
    return Promise.reject(new Error('MOMO 不能用 file:// 直接打开网页使用。请在项目根目录运行  python3 -m http.server 8137 ，再用  http://localhost:8137/html/...  打开本页面。'));
  }
  _momoDocPromise = fetch(momoBasePath() + '../docs/moi-product-design.md').then(function(r) {
    if (!r.ok) throw new Error('产品文档加载失败（HTTP ' + r.status + '），请确认通过本地服务器（http://localhost…）访问。');
    return r.text();
  }).catch(function(e) { _momoDocPromise = null; throw e; });
  return _momoDocPromise;
}
function momoSystemPrompt(doc) {
  return '你是 MOMO，MOI（MatrixOne Intelligence）数据智能平台的产品助手。你唯一的知识来源是下面 <doc></doc> 中的《MOI 产品概述》文档。\n'
    + '回答规则：\n'
    + '1. 只依据文档内容回答，不要编造文档之外的功能、数字或细节。\n'
    + '2. 文档没有写到的，明确说"产品文档里没有写到这一点"，并建议向产品同事确认，不要猜测。\n'
    + '3. 用简洁、专业、友好的中文回答，可用小标题和要点列表组织。\n'
    + '4. 不要在回答里标注数据来源、章节出处或"依据：…"，直接给结论。\n'
    + '5. 当用户要求在 MOI 里【执行操作】（创建 / 搭建工作流、运行 SQL、接入数据源、创建知识库、导入数据等会改变平台状态的动作）时：先用一两句话说明你要做什么，然后输出一个"操作块"，格式严格如下——用 ```moi-action 围栏，头部是 lang / done 两行 key: value，--- 之后放要执行的脚本或命令（原型阶段可用合理的示例 API 调用 / SQL / 伪代码）：\n'
    + '```moi-action\n'
    + 'lang: json\n'
    + 'done: 已创建工作流「RAG 文档准备」，含 5 个节点。\n'
    + '---\n'
    + 'POST /api/v1/workflows\n'
    + '{ "name": "RAG 文档准备", "nodes": [ ... ] }\n'
    + '```\n'
    + '只有"执行类"请求才输出操作块；纯问答（解释 / 查询知识）就正常用文字回答、不要输出操作块；一次最多一个操作块。\n\n'
    + '<doc>\n' + doc + '\n</doc>';
}
// 从模型回复里解析"操作块"（```moi-action ... ```）。返回 { text, action|null }
function momoParseAction(raw) {
  var s = String(raw || '');
  var m = s.match(/```moi-action\s*([\s\S]*?)```/);
  if (!m) return { text: s, action: null };
  var inner = m[1] || '';
  var intro = (s.slice(0, m.index) + s.slice(m.index + m[0].length)).trim();
  var parts = inner.split(/\n-{3,}\n/);
  var head = parts.length > 1 ? parts[0] : '';
  var code = (parts.length > 1 ? parts.slice(1).join('\n---\n') : inner).trim();
  var meta = {};
  head.split('\n').forEach(function (line) {
    var i = line.indexOf(':');
    if (i > 0) meta[line.slice(0, i).trim().toLowerCase()] = line.slice(i + 1).trim();
  });
  return {
    text: intro,
    action: { lang: meta.lang || 'bash', code: code, status: 'pending', doneText: meta.done || meta.donetext || '已执行完成。' }
  };
}

function momoRawChat(base, key, model, messages, stream, onDelta, signal) {
  var url = base.replace(/\/+$/, '') + '/chat/completions';
  return fetch(url, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', 'Authorization': 'Bearer ' + key },
    body: JSON.stringify({ model: model, messages: messages, temperature: 0.2, stream: !!stream }),
    signal: signal || undefined
  }).then(function(resp) {
    if (!resp.ok) {
      return resp.text().catch(function() { return ''; }).then(function(t) {
        throw new Error('HTTP ' + resp.status + (t ? (' · ' + t.slice(0, 200)) : ''));
      });
    }
    var ct = resp.headers.get('content-type') || '';
    if (!stream || ct.indexOf('text/event-stream') === -1) {
      return resp.json().then(function(j) {
        var c = (j.choices && j.choices[0] && ((j.choices[0].message && j.choices[0].message.content) || j.choices[0].text)) || '';
        if (onDelta) onDelta(c);
        return c;
      });
    }
    var reader = resp.body.getReader(), dec = new TextDecoder(), buf = '', full = '';
    function pump() {
      return reader.read().then(function(res) {
        if (res.done) return full;
        buf += dec.decode(res.value, { stream: true });
        var idx;
        while ((idx = buf.indexOf('\n')) >= 0) {
          var line = buf.slice(0, idx).trim(); buf = buf.slice(idx + 1);
          if (!line) continue;
          if (line.indexOf('data:') === 0) line = line.slice(5).trim();
          if (line === '[DONE]') return full;
          try {
            var j = JSON.parse(line);
            var d = j.choices && j.choices[0] && j.choices[0].delta && j.choices[0].delta.content;
            if (d) { full += d; if (onDelta) onDelta(full); }
          } catch (e) { /* 忽略保活/不完整分片 */ }
        }
        return pump();
      });
    }
    return pump();
  });
}
function momoLLM(messages, onDelta, signal) { var c = momoCfg(); return momoRawChat(c.base, c.key, c.model, messages, true, onDelta, signal); }

// 回答完成后，后台再轻量问一次模型：用户接下来可能想问的几个相关问题（渲染为可点击的 chip）
function momoSuggestFollowups(bot, lastQuestion) {
  if (!momoCfgComplete()) return;
  var c = momoCfg();
  var msgs = [
    { role: 'system', content: '你是 MOI（MatrixOne Intelligence）数据智能平台的产品助手。根据下面这一轮问答，列出 3 个用户可能想接着问的、与 MOI 产品相关的简短问题。要求：每行一个问题；不要序号或任何符号；每个不超过 20 个字；不要与已问的重复；只输出问题本身，不要任何其它文字。' },
    { role: 'user', content: '已问：' + (lastQuestion || '') + '\n回答：' + (bot.content || '').slice(0, 1200) + '\n\n请给出 3 个相关的后续问题：' }
  ];
  momoRawChat(c.base, c.key, c.model, msgs, false, null).then(function (txt) {
    var lines = String(txt || '').split('\n').map(function (s) {
      return s.replace(/^\s*(\d+\s*[\.\)、]|[-*•·])\s*/, '').replace(/^["“]+|["”]+$/g, '').trim();
    }).filter(function (s) { return s && s.length <= 30; });
    var chips = lines.slice(0, 3);
    if (chips.length) { bot.chips = chips; momoSave(); momoRender(); }
  }).catch(function () { /* 关联问题生成失败就不显示，静默 */ });
}

// 回答下方的基础操作：点赞 / 点踩 / 复制回答
var MOMO_ICON_UP = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"><path d="M14 9V5a3 3 0 0 0-3-3l-4 9v11h11.28a2 2 0 0 0 2-1.7l1.38-9a2 2 0 0 0-2-2.3zM7 22H4a2 2 0 0 1-2-2v-7a2 2 0 0 1 2-2h3"/></svg>';
var MOMO_ICON_DOWN = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"><path d="M10 15v4a3 3 0 0 0 3 3l4-9V2H5.72a2 2 0 0 0-2 1.7l-1.38 9a2 2 0 0 0 2 2.3zm7-13h2.67A2.31 2.31 0 0 1 22 4v7a2.31 2.31 0 0 1-2.33 2H17"/></svg>';
var MOMO_ICON_COPY = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"><rect x="9" y="9" width="13" height="13" rx="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>';
var MOMO_ICON_CHECK = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M20 6L9 17l-5-5"/></svg>';
function momoActionsBar(m, idx) {
  var fb = m.feedback || '';
  return '<div class="momo-msg-actions">'
    + '<button class="momo-mab' + (fb === 'up' ? ' active' : '') + '" title="点赞" onclick="momoFeedback(' + idx + ',\'up\')">' + MOMO_ICON_UP + '</button>'
    + '<button class="momo-mab' + (fb === 'down' ? ' active down' : '') + '" title="点踩" onclick="momoFeedback(' + idx + ',\'down\')">' + MOMO_ICON_DOWN + '</button>'
    + '<button class="momo-mab" title="复制回答" onclick="momoCopy(' + idx + ',this)">' + MOMO_ICON_COPY + '</button>'
    + '</div>';
}
function momoFeedback(idx, v) {
  var m = momoMsgs[idx]; if (!m) return;
  m.feedback = (m.feedback === v) ? null : v; // 再点一次取消
  momoSave(); momoRender();
}
function momoFallbackCopy(txt) {
  try { var ta = document.createElement('textarea'); ta.value = txt; ta.style.position = 'fixed'; ta.style.opacity = '0'; document.body.appendChild(ta); ta.select(); document.execCommand('copy'); document.body.removeChild(ta); } catch (e) {}
}
function momoCopy(idx, btn) {
  var m = momoMsgs[idx]; if (!m) return;
  var txt = m.content || '';
  function ok() {
    if (!btn) return;
    btn.innerHTML = MOMO_ICON_CHECK; btn.title = '已复制'; btn.classList.add('copied');
    setTimeout(function () { btn.innerHTML = MOMO_ICON_COPY; btn.title = '复制回答'; btn.classList.remove('copied'); }, 1300);
  }
  if (navigator.clipboard && navigator.clipboard.writeText) {
    navigator.clipboard.writeText(txt).then(ok, function () { momoFallbackCopy(txt); ok(); });
  } else { momoFallbackCopy(txt); ok(); }
}
function momoMd(md) {
  var s = String(md).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
  s = s.replace(/`([^`]+)`/g, '<code>$1</code>');
  s = s.replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>');
  var lines = s.split('\n'), html = '', inUl = false, inOl = false;
  function close() { if (inUl) { html += '</ul>'; inUl = false; } if (inOl) { html += '</ol>'; inOl = false; } }
  for (var i = 0; i < lines.length; i++) {
    var t = lines[i].trim();
    if (!t) { close(); continue; }
    var h = t.match(/^(#{1,6})\s+(.*)$/);
    if (h) { close(); var lv = Math.min(5, h[1].length + 2); html += '<h' + lv + '>' + h[2] + '</h' + lv + '>'; continue; }
    var ul = t.match(/^[-*]\s+(.*)$/);
    if (ul) { if (!inUl) { close(); html += '<ul>'; inUl = true; } html += '<li>' + ul[1] + '</li>'; continue; }
    var ol = t.match(/^\d+\.\s+(.*)$/);
    if (ol) { if (!inOl) { close(); html += '<ol>'; inOl = true; } html += '<li>' + ol[1] + '</li>'; continue; }
    close(); html += '<p>' + t + '</p>';
  }
  close(); return html;
}
function momoSettings() {
  var mask = document.getElementById('momoModal');
  if (!mask) {
    mask = document.createElement('div');
    mask.className = 'momo-modal-mask'; mask.id = 'momoModal';
    mask.innerHTML = '<div class="momo-modal">'
      + '<h4><span>MOMO 模型设置</span><button class="momo-icon-btn" onclick="momoSettingsClose()"><svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M18 6L6 18M6 6l12 12"/></svg></button></h4>'
      + '<div class="mm-body">'
      +   '<label>API 地址（OpenAI 兼容 Base URL）</label><input id="momoCfgBase" placeholder="https://api.openai.com/v1" autocomplete="off" spellcheck="false">'
      +   '<label>API Key</label><input id="momoCfgKey" type="password" placeholder="sk-..." autocomplete="off">'
      +   '<label>模型名称</label><input id="momoCfgModel" placeholder="gpt-4o-mini" autocomplete="off" spellcheck="false">'
      +   '<div class="mm-note">🔒 密钥只保存在<b>本浏览器（localStorage）</b>，不写入代码、不提交 Git、不会上传 GitHub。换浏览器或清缓存后需重填。<br>⚙️ 接口需兼容 OpenAI <code>/chat/completions</code>；文档较长，建议用<b>长上下文模型</b>。<br>⚠️ 直连报 CORS（Failed to fetch）时：运行本地代理 <code>node html/scripts/momo-proxy.js</code>，把地址改填 <code>http://localhost:8788/&lt;完整上游URL&gt;</code>；或改用支持浏览器跨域的接口 / 本地模型。<br>未配置时 MOMO 给演示回复；配置后基于《MOI 产品概述》文档作答。</div>'
      +   '<div class="mm-actions"><button class="mm-btn" onclick="momoSettingsTest()">测试连接</button><button class="mm-btn primary" onclick="momoSettingsSave()">保存</button></div>'
      +   '<div class="mm-result" id="momoCfgResult"></div>'
      + '</div></div>';
    mask.onclick = function(e) { if (e.target === mask) momoSettingsClose(); };
    document.body.appendChild(mask);
  }
  var c = momoCfg();
  document.getElementById('momoCfgBase').value = c.base || 'https://api.openai.com/v1';
  document.getElementById('momoCfgKey').value = c.key || '';
  document.getElementById('momoCfgModel').value = c.model || 'gpt-4o-mini';
  var r = document.getElementById('momoCfgResult'); r.className = 'mm-result'; r.textContent = '';
  mask.classList.add('show');
}
function momoSettingsClose() { var m = document.getElementById('momoModal'); if (m) m.classList.remove('show'); }
function momoSettingsSave() {
  localStorage.setItem('momo_api_base', document.getElementById('momoCfgBase').value.trim());
  localStorage.setItem('momo_api_key', document.getElementById('momoCfgKey').value.trim());
  localStorage.setItem('momo_model', document.getElementById('momoCfgModel').value.trim());
  momoSettingsClose();
  if (!momoMsgs.length) momoRender();
}
function momoSettingsTest() {
  var r = document.getElementById('momoCfgResult'); r.className = 'mm-result show'; r.style.color = 'rgba(0,0,0,0.5)'; r.textContent = '正在测试…';
  var base = document.getElementById('momoCfgBase').value.trim(),
      key = document.getElementById('momoCfgKey').value.trim(),
      model = document.getElementById('momoCfgModel').value.trim();
  if (!base || !key || !model) { r.style.color = '#cf1322'; r.textContent = '请先填完整 API 地址、Key 和模型。'; return; }
  momoRawChat(base, key, model, [{ role: 'user', content: '回复两个字：你好' }], false, null).then(function(t) {
    r.style.color = '#0b8a5e'; r.textContent = '✓ 连接成功，模型回复："' + (t || '').slice(0, 40) + '"';
  }).catch(function(e) {
    r.style.color = '#cf1322'; r.textContent = '✗ ' + e.message;
  });
}

// === Universal page i18n ===
// 翻译规则：
//   1. 文本节点：精确匹配 MOI_I18N 字典的 key，替换为对应语言的 value
//   2. placeholder 属性：同上
//   3. title 属性：同上
//   4. data-tip 属性：同上
//   5. 页面标题：匹配 "MOI - xxx" 中的 xxx
//   6. select > option：翻译 option 的 textContent
//   跳过：script / style / textarea / code / pre / svg 内部的文本
//   跳过：已翻译的节点（通过 data-i18n-done 标记避免重复）
//
// MutationObserver 自动监听 DOM 变化，任何 innerHTML 赋值、appendChild 等
// 动态渲染都会自动触发翻译，无需在业务代码中手动调用 t() 或 applyPageI18n()。

function _i18nTranslateNode(root) {
  if (MOI_LANG === 'zh') return;
  var walker = document.createTreeWalker(
    root,
    NodeFilter.SHOW_TEXT,
    { acceptNode: function(node) {
      var p = node.parentElement;
      if (!p) return NodeFilter.FILTER_REJECT;
      var tag = p.tagName;
      if (tag === 'SCRIPT' || tag === 'STYLE' || tag === 'TEXTAREA' || tag === 'CODE' || tag === 'PRE') return NodeFilter.FILTER_REJECT;
      if (p.closest('svg')) return NodeFilter.FILTER_REJECT;
      if (node.textContent.trim().length === 0) return NodeFilter.FILTER_REJECT;
      return NodeFilter.FILTER_ACCEPT;
    }}
  );
  var node;
  while (node = walker.nextNode()) {
    var text = node.textContent.trim();
    if (text && MOI_I18N[text] && MOI_I18N[text][MOI_LANG]) {
      node.textContent = node.textContent.replace(text, MOI_I18N[text][MOI_LANG]);
    }
  }
  // Translate attributes within this root
  root.querySelectorAll && root.querySelectorAll('input[placeholder], textarea[placeholder]').forEach(function(el) {
    var ph = el.placeholder.trim();
    if (ph && MOI_I18N[ph] && MOI_I18N[ph][MOI_LANG]) el.placeholder = MOI_I18N[ph][MOI_LANG];
  });
  root.querySelectorAll && root.querySelectorAll('[title]').forEach(function(el) {
    var tt = el.title.trim();
    if (tt && MOI_I18N[tt] && MOI_I18N[tt][MOI_LANG]) el.title = MOI_I18N[tt][MOI_LANG];
  });
  root.querySelectorAll && root.querySelectorAll('[data-tip]').forEach(function(el) {
    var tip = el.getAttribute('data-tip').trim();
    if (tip && MOI_I18N[tip] && MOI_I18N[tip][MOI_LANG]) el.setAttribute('data-tip', MOI_I18N[tip][MOI_LANG]);
  });
  root.querySelectorAll && root.querySelectorAll('select option').forEach(function(el) {
    var ot = el.textContent.trim();
    if (ot && MOI_I18N[ot] && MOI_I18N[ot][MOI_LANG]) el.textContent = MOI_I18N[ot][MOI_LANG];
  });
}

function applyPageI18n() {
  if (MOI_LANG === 'zh') return;
  _i18nTranslateNode(document.body);
  // Translate page title
  var titleMatch = document.title.match(/MOI - (.+)/);
  if (titleMatch && MOI_I18N[titleMatch[1]] && MOI_I18N[titleMatch[1]][MOI_LANG]) {
    document.title = 'MOI - ' + MOI_I18N[titleMatch[1]][MOI_LANG];
  }
}

// MutationObserver: 自动翻译所有动态渲染的 DOM
var _i18nObserver = null;
function startI18nObserver() {
  if (MOI_LANG === 'zh' || _i18nObserver) return;
  _i18nObserver = new MutationObserver(function(mutations) {
    mutations.forEach(function(m) {
      m.addedNodes.forEach(function(node) {
        if (node.nodeType === 1) _i18nTranslateNode(node);        // Element
        else if (node.nodeType === 3) {                            // Text node
          var text = node.textContent.trim();
          if (text && MOI_I18N[text] && MOI_I18N[text][MOI_LANG]) {
            node.textContent = node.textContent.replace(text, MOI_I18N[text][MOI_LANG]);
          }
        }
      });
      // characterData changes (text node content changed in-place)
      if (m.type === 'characterData' && m.target.nodeType === 3) {
        var p = m.target.parentElement;
        if (p && !['SCRIPT','STYLE','TEXTAREA','CODE','PRE'].includes(p.tagName) && !p.closest('svg')) {
          var text = m.target.textContent.trim();
          if (text && MOI_I18N[text] && MOI_I18N[text][MOI_LANG]) {
            m.target.textContent = m.target.textContent.replace(text, MOI_I18N[text][MOI_LANG]);
          }
        }
      }
    });
  });
  _i18nObserver.observe(document.body, { childList: true, subtree: true, characterData: true });
}

function toggleSidebarCollapse() {
  var sidebar = document.querySelector('.sidebar');
  var main = document.querySelector('.main');
  if (!sidebar) return;
  sidebar.classList.toggle('collapsed');
  var collapsed = sidebar.classList.contains('collapsed');
  if (main) main.style.marginLeft = collapsed ? '52px' : '';
  localStorage.setItem('moi_sidebar_collapsed', collapsed ? '1' : '0');
}

// Placeholder functions for bottom icons (will be replaced with real modals)
var SQL_QUERY_RESOURCES = [
  { id:'default-query', name:'默认 SQL 资源', spec:'SQL 算力档 · S（平台预置）' },
  { id:'analytics-q',   name:'分析 SQL 资源', spec:'SQL 算力档 · M' },
  { id:'bi-q',          name:'BI 专用资源',   spec:'SQL 算力档 · L' }
];
function openSqlConnModal() {
  var overlay = document.getElementById('sqlConnModal');
  if (!overlay) {
    var H = 'freetier-01.cn-hangzhou.cluster.matrixonecloud.cn';
    var P = '6001';
    var U = '019c481d-113a-72af-a0c2-7e77e3315300:admin:accountadmin';
    var PWD = 'Moi@2026secure';
    window._sqlConn = {h:H,p:P,u:U,pwd:PWD};

    overlay = document.createElement('div');
    overlay.id = 'sqlConnModal';
    overlay.style.cssText = 'display:none;position:fixed;top:0;left:0;right:0;bottom:0;background:rgba(0,0,0,0.45);z-index:500;align-items:center;justify-content:center';
    overlay.onclick = function(e) { if (e.target === overlay) closeSqlConnModal(); };
    overlay.innerHTML = '<div style="background:#fff;border-radius:12px;width:600px;box-shadow:0 6px 30px rgba(0,0,0,0.12);padding:24px 28px;max-height:90vh;overflow-y:auto">'
      + '<div style="display:flex;align-items:center;justify-content:space-between;margin-bottom:20px">'
      + '<span style="font-size:16px;font-weight:600;color:rgba(0,0,0,0.88)">SQL 连接串</span>'
      + '<button onclick="closeSqlConnModal()" style="width:32px;height:32px;border:none;background:none;cursor:pointer;border-radius:6px;font-size:18px;color:rgba(0,0,0,0.45);display:flex;align-items:center;justify-content:center" onmouseover="this.style.background=\'rgba(0,0,0,0.04)\'" onmouseout="this.style.background=\'none\'">✕</button>'
      + '</div>'
      + '<div style="margin-bottom:20px"><select style="width:100%;height:40px;padding:0 12px;border:1px solid #d9d9d9;border-radius:6px;font-size:13px;outline:none;cursor:pointer;background:#fff;appearance:auto" id="sqlConnType" onchange="updateSqlCmd()">'
      + '<option value="public">公网连接</option><option value="private">私网连接</option></select></div>'
      // Private link info (hidden by default)
      + '<div id="sqlPrivateInfo" style="display:none;background:#fafafa;border:1px solid #f0f0f0;border-radius:8px;padding:16px 20px;margin-bottom:20px">'
      + '<div style="font-size:13px;color:rgba(0,0,0,0.65);margin-bottom:12px">请在连接 MatrixOne Intelligence 实例前在你的 VPC 中设置一个私网终端节点</div>'
      + '<div style="font-size:12px;color:rgba(0,0,0,0.45);margin-bottom:4px">服务名称：</div>'
      + '<div style="position:relative;background:#fff;border:1px solid #f0f0f0;border-radius:6px;padding:10px 40px 10px 14px;font-family:monospace;font-size:13px;color:rgba(0,0,0,0.88);margin-bottom:10px">com.aliyuncs.privatelink.cn-hangzhou.epsrv-bp1zd5e6puhana8d9ip4'
      + '<button onclick="copyText(\'com.aliyuncs.privatelink.cn-hangzhou.epsrv-bp1zd5e6puhana8d9ip4\')" style="position:absolute;top:8px;right:8px;width:24px;height:24px;border:1px solid #e5e6eb;border-radius:4px;background:#fff;cursor:pointer;display:flex;align-items:center;justify-content:center;color:rgba(0,0,0,0.35)" title="复制"><svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="9" y="9" width="13" height="13" rx="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg></button></div>'
      + '<div style="display:flex;gap:24px;font-size:13px"><div><span style="color:rgba(0,0,0,0.45)">可用区 ID：</span><span style="color:rgba(0,0,0,0.88)">cn-hangzhou-j,cn-hangzhou-k</span></div>'
      + '<div><span style="color:rgba(0,0,0,0.45)">地区 ID：</span><span style="color:rgba(0,0,0,0.88)">cn-hangzhou</span></div></div>'
      + '</div>'
      + '<div style="font-size:15px;font-weight:600;color:rgba(0,0,0,0.88);margin-bottom:12px">SQL 客户端</div>'
      + '<div style="font-size:12px;color:rgba(0,0,0,0.45);margin-bottom:6px">选择客户端</div>'
      + '<select style="width:100%;height:40px;padding:0 12px;border:1px solid #1677ff;border-radius:6px;font-size:13px;outline:none;cursor:pointer;background:#fff;appearance:auto;color:rgba(0,0,0,0.88)" id="sqlClientType" onchange="updateSqlCmd()">'
      + '<option value="mysql">MySQL</option><option value="jdbc">JDBC</option><option value="python">Python</option><option value="go">Go</option></select>'
      + '<div style="font-size:15px;font-weight:600;color:rgba(0,0,0,0.88);margin:22px 0 6px">SQL 计算资源</div>'
      + '<div style="font-size:12px;color:rgba(0,0,0,0.45);margin-bottom:12px">连接串通过SQL 计算资源执行 SQL；<b style="color:rgba(0,0,0,0.7)">绑定后才会生成对应的连接串</b>。可同时绑定多个，每个资源一条独立连接串。</div>'
      + '<div style="background:#f0f7ff;border:1px solid #d6e4ff;border-radius:8px;padding:12px 14px;margin-bottom:12px;display:flex;gap:10px">'
      + '<svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="#1677ff" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" style="flex-shrink:0;margin-top:1px"><circle cx="12" cy="12" r="10"/><line x1="12" y1="16" x2="12" y2="12"/><line x1="12" y1="8" x2="12.01" y2="8"/></svg>'
      + '<div style="font-size:12px;color:rgba(0,0,0,0.65);line-height:1.7">'
      + '<div style="font-weight:600;color:rgba(0,0,0,0.8);margin-bottom:2px">想在 MOI 的「SQL 历史」中保存查询结果？</div>'
      + '仅带返回结果的语句会被保存（SELECT / SHOW / DESC / EXECUTE）；其中 SELECT 语句须以固定注释开头，例如：<br>'
      + '<code style="display:inline-block;background:#fff;border:1px solid #e5e6eb;border-radius:4px;padding:3px 8px;font-size:11px;margin-top:4px">/* cloud_user */ /* save_result */ SELECT a FROM t1;</code>'
      + '</div></div>'
      + '<div id="sqlBoundList"></div>'
      + '<div id="sqlBindRow" style="margin-top:12px"></div>'
      + '</div>';
    document.body.appendChild(overlay);
  }
  updateSqlCmd();
  overlay.style.display = 'flex';
}

function updateSqlCmd() {
  var c = window._sqlConn;
  var client = (document.getElementById('sqlClientType') || {}).value || 'mysql';
  var connType = (document.getElementById('sqlConnType') || {}).value || 'public';
  var isPrivate = connType === 'private';
  var privateInfo = document.getElementById('sqlPrivateInfo');
  if (privateInfo) privateInfo.style.display = isPrivate ? '' : 'none';

  if (!window._sqlBound) window._sqlBound = [];
  window._sqlConnFulls = {};
  var listEl = document.getElementById('sqlBoundList');
  if (!listEl) return;

  if (!window._sqlBound.length) {
    listEl.innerHTML = '<div style="background:#fafafa;border:1px dashed #e0e0e0;border-radius:8px;padding:26px;text-align:center;color:rgba(0,0,0,0.4);font-size:13px">尚未绑定 SQL 计算资源<br><span style="font-size:12px;color:rgba(0,0,0,0.3)">绑定一个 SQL 计算资源后，这里会生成对应的连接串</span></div>';
  } else {
    var copyIcon = '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="9" y="9" width="13" height="13" rx="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>';
    listEl.innerHTML = window._sqlBound.map(function(id) {
      var r = SQL_QUERY_RESOURCES.find(function(x){ return x.id === id; });
      if (!r) return '';
      var host = isPrivate ? '&lt;privatelink_endpoint_domain&gt;' : (r.id + '.' + c.h);
      var hostFull = isPrivate ? '<privatelink_endpoint_domain>' : (r.id + '.' + c.h);
      var cmd = buildSqlCmd(client, host, hostFull, c);
      window._sqlConnFulls[id] = cmd.full;
      return '<div style="border:1px solid #eef0f4;border-radius:10px;padding:14px 16px;margin-bottom:12px">'
        + '<div style="display:flex;align-items:center;justify-content:space-between;margin-bottom:10px">'
        + '<div><span style="font-weight:600;font-size:13px;color:rgba(0,0,0,0.85)">' + r.name + '</span><span style="font-size:11px;color:rgba(0,0,0,0.4);margin-left:8px">' + r.spec + '</span></div>'
        + '<button onclick="unbindSqlResource(\'' + r.id + '\')" style="border:none;background:none;color:#ff4d4f;font-size:12px;cursor:pointer">解绑</button></div>'
        + '<div style="position:relative;background:#fafafa;border:1px solid #f0f0f0;border-radius:8px;padding:14px 44px 14px 16px;font-family:\'SF Mono\',Monaco,Consolas,monospace;font-size:13px;color:rgba(0,0,0,0.88);line-height:1.6;word-break:break-all">' + cmd.display
        + '<button onclick="copySqlConnFor(\'' + r.id + '\')" style="position:absolute;top:10px;right:10px;width:28px;height:28px;border:1px solid #e5e6eb;border-radius:4px;background:#fff;cursor:pointer;display:flex;align-items:center;justify-content:center;color:rgba(0,0,0,0.35)" title="复制（含密码）">' + copyIcon + '</button></div>'
        + '</div>';
    }).join('');
  }

  var unbound = SQL_QUERY_RESOURCES.filter(function(r){ return window._sqlBound.indexOf(r.id) === -1; });
  var bindRow = document.getElementById('sqlBindRow');
  if (bindRow) {
    if (unbound.length) {
      bindRow.innerHTML = '<div style="display:flex;gap:8px"><select id="sqlBindSelect" style="flex:1;height:36px;padding:0 10px;border:1px solid #d9d9d9;border-radius:6px;font-size:13px;background:#fff;cursor:pointer">'
        + unbound.map(function(r){ return '<option value="' + r.id + '">' + r.name + ' · ' + r.spec + '</option>'; }).join('')
        + '</select><button onclick="bindSqlResource()" style="height:36px;padding:0 16px;border:1px solid #1677ff;background:#1677ff;color:#fff;border-radius:6px;font-size:13px;cursor:pointer;white-space:nowrap">+ 绑定 SQL 计算资源</button></div>';
    } else {
      bindRow.innerHTML = '<div style="font-size:12px;color:rgba(0,0,0,0.3)">已绑定全部可用SQL 计算资源</div>';
    }
  }
}
function buildSqlCmd(client, host, hostFull, c) {
  var display = '', full = '';
  if (client === 'mysql') {
    display = 'mysql -c flag -h ' + host + ' -P ' + c.p + ' -u ' + c.u + '  -p ********';
    full = 'mysql -c flag -h ' + hostFull + ' -P ' + c.p + ' -u ' + c.u + '  -p ' + c.pwd;
  } else if (client === 'jdbc') {
    display = 'jdbc:mysql://' + host + ':' + c.p + '/&lt;your_database&gt;?user=' + c.u + '&amp;password=&lt;your_password&gt;&amp;enabledTLSProtocols=TLSv1.2';
    full = 'jdbc:mysql://' + hostFull + ':' + c.p + '/<your_database>?user=' + c.u + '&password=' + c.pwd + '&enabledTLSProtocols=TLSv1.2';
  } else if (client === 'python') {
    display = "host='" + host + "', port=" + c.p + ", user='" + c.u + "', password='&lt;your_password&gt;'";
    full = "host='" + hostFull + "', port=" + c.p + ", user='" + c.u + "', password='" + c.pwd + "'";
  } else if (client === 'go') {
    display = '"mysql", "' + c.u.replace(/:/g,'#') + ':&lt;your_password&gt;@tcp(' + host + ':' + c.p + ')/"';
    full = '"mysql", "' + c.u.replace(/:/g,'#') + ':' + c.pwd + '@tcp(' + hostFull + ':' + c.p + ')/"';
  }
  return { display: display, full: full };
}
function bindSqlResource() {
  var sel = document.getElementById('sqlBindSelect');
  if (!sel || !sel.value) return;
  if (!window._sqlBound) window._sqlBound = [];
  if (window._sqlBound.indexOf(sel.value) === -1) window._sqlBound.push(sel.value);
  updateSqlCmd();
}
function unbindSqlResource(id) {
  window._sqlBound = (window._sqlBound || []).filter(function(x){ return x !== id; });
  updateSqlCmd();
}
function copySqlConnFor(id) {
  var full = (window._sqlConnFulls || {})[id];
  if (!full) return;
  if (navigator.clipboard) { navigator.clipboard.writeText(full).then(function(){ alert('已复制（含密码）'); }); }
  else { var ta = document.createElement('textarea'); ta.value = full; document.body.appendChild(ta); ta.select(); document.execCommand('copy'); document.body.removeChild(ta); alert('已复制（含密码）'); }
}

function closeSqlConnModal() {
  var m = document.getElementById('sqlConnModal');
  if (m) m.style.display = 'none';
}

function copySqlConn() {
  var full = document.getElementById('sqlConnFull').value;
  if (navigator.clipboard) {
    navigator.clipboard.writeText(full).then(function() { alert('已复制（含密码）'); });
  } else {
    var ta = document.createElement('textarea');
    ta.value = full; document.body.appendChild(ta); ta.select();
    document.execCommand('copy'); document.body.removeChild(ta);
    alert('已复制（含密码）');
  }
}

function copyText(text) {
  if (navigator.clipboard) {
    navigator.clipboard.writeText(text).then(function() { alert('已复制'); });
  } else {
    var ta = document.createElement('textarea');
    ta.value = text; document.body.appendChild(ta); ta.select();
    document.execCommand('copy'); document.body.removeChild(ta);
    alert('已复制');
  }
}

// Toggle dropdown menus
function toggleDropdown(id) {
  document.querySelectorAll('.dropdown-content').forEach(d => {
    if (d.id !== id) d.classList.remove('show');
  });
  document.getElementById(id).classList.toggle('show');
}

// Close dropdowns on outside click
document.addEventListener('click', function(e) {
  if (!e.target.closest('.dropdown') && !e.target.closest('.product-switch') && !e.target.closest('.workspace-select')) {
    document.querySelectorAll('.dropdown-content').forEach(d => d.classList.remove('show'));
  }
});

// Toggle sidebar sub-menus (allow multiple open)
function toggleSubMenu(id, el) {
  const sub = document.getElementById(id);
  const arrow = el.querySelector('.arrow');
  const isOpen = sub.classList.contains('open');

  if (isOpen) {
    sub.classList.remove('open');
    if (arrow) arrow.classList.remove('open');
  } else {
    sub.classList.add('open');
    if (arrow) arrow.classList.add('open');
  }
}

// Auto-expand sidebar sub-menu based on current page
(function() {
  const path = window.location.pathname;
  const links = document.querySelectorAll('.sidebar a');
  links.forEach(link => {
    const href = link.getAttribute('href');
    if (href && path.endsWith(href.replace(/^\.\.\//, '').replace(/^\.\//, ''))) {
      link.classList.add('active');
      // Expand parent sub-menu if it's a sub-item
      const subMenu = link.closest('.sub-menu');
      if (subMenu) {
        subMenu.classList.add('open');
        const parentItem = subMenu.previousElementSibling;
        if (parentItem) {
          parentItem.classList.add('active');
          const arrow = parentItem.querySelector('.arrow');
          if (arrow) arrow.classList.add('open');
        }
      }
    }
  });
})();

// === Console user avatar & popover ===
(function() {
  function initAvatar() {
    // Always bind click event for popover toggle, regardless of user data
    var btn = document.getElementById('consoleAvatarBtn');
    if (btn && !btn._popoverBound) {
      btn._popoverBound = true;
      btn.addEventListener('click', function(e) {
        e.stopPropagation();
        var pop = document.getElementById('consolePopover');
        if (pop) pop.classList.toggle('show');
      });
    }

    var raw = localStorage.getItem('moi_user');
    if (!raw) return;
    try {
      var user = JSON.parse(raw);
      var initial = user.name ? user.name.charAt(0).toUpperCase() : 'U';
      var email = user.email || (user.username ? user.username + '@matrixorigin.cn' : (user.phone || '') + '@matrixorigin.cn');

      var avatarBtn = document.getElementById('consoleAvatarBtn');
      var popAvatar = document.querySelector('#consolePopover .popover-avatar');
      var popName = document.getElementById('consolePopName');
      var popEmail = document.getElementById('consolePopEmail');

      if (avatarBtn) {
        if (user.avatarUrl) {
          avatarBtn.textContent = '';
          avatarBtn.style.backgroundImage = 'url(' + user.avatarUrl + ')';
          avatarBtn.style.backgroundSize = 'cover';
          avatarBtn.style.backgroundPosition = 'center';
        } else {
          avatarBtn.textContent = initial;
        }
        avatarBtn.title = user.name || '用户';
      }
      if (popAvatar) {
        if (user.avatarUrl) {
          popAvatar.textContent = '';
          popAvatar.style.backgroundImage = 'url(' + user.avatarUrl + ')';
          popAvatar.style.backgroundSize = 'cover';
          popAvatar.style.backgroundPosition = 'center';
        } else {
          popAvatar.textContent = initial;
        }
      }
      if (popName) popName.textContent = user.name || '用户';
      if (popEmail) popEmail.textContent = email;
    } catch(e) {}
  }
  // Run after buildTopBar has created the DOM elements
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', initAvatar);
  } else {
    // If DOM already loaded, wait a tick for buildTopBar to finish
    setTimeout(initAvatar, 0);
  }
  // Close popover on outside click (can be registered immediately)
  document.addEventListener('click', function(e) {
    const pop = document.getElementById('consolePopover');
    if (pop && !e.target.closest('.user-avatar-wrap')) pop.classList.remove('show');
  });
})();

function consoleLogout() {
  localStorage.removeItem('moi_user');
  var path = window.location.pathname;
  if (path.indexOf('/data-connection/') !== -1 || path.indexOf('/resource-center/') !== -1 || path.indexOf('/data-processing/') !== -1 || path.indexOf('/app-dev/') !== -1 || path.indexOf('/user-perm/') !== -1 || path.indexOf('/dashboard/') !== -1 || path.indexOf('/account/') !== -1 || path.indexOf('/admin/') !== -1 || path.indexOf('/taas/') !== -1) {
    window.location.href = '../website/login.html';
  } else {
    window.location.href = 'website/login.html';
  }
}

// === Credit Balance Display ===
(function() {
  // Initialize default credit balance
  if (!localStorage.getItem('moi_credit')) {
    localStorage.setItem('moi_credit', '86.50');
  }

  function initCreditBadge() {
    if (window.location.pathname.indexOf('/admin/') !== -1) return;
    var rightActions = document.querySelector('.top-bar .right-actions');
    if (!rightActions) return;

    // Inject CSS
    if (!document.getElementById('creditBadgeStyle')) {
      var style = document.createElement('style');
      style.id = 'creditBadgeStyle';
      style.textContent = '.credit-badge{display:flex;align-items:center;gap:5px;padding:4px 12px;border-radius:16px;background:transparent;border:1px solid #e5e7eb;cursor:pointer;margin-right:4px;transition:all .2s;font-size:13px;}'
        + '.credit-badge:hover{background:#f8f9fc;border-color:#d1d5db;}'
        + '.credit-badge .credit-val{font-weight:600;color:#389e0d;}'
        + '.credit-badge .credit-label{color:rgba(0,0,0,0.35);font-size:12px;}'
        + '.credit-badge.low .credit-val{color:#d46b08;}'
        + '.credit-badge.low .credit-label{color:#d46b08;}'
        + '.credit-badge.critical .credit-val{color:#cf1322;}'
        + '.credit-badge.critical .credit-label{color:#cf1322;}';
      document.head.appendChild(style);
    }

    // Create badge element
    var badge = document.createElement('div');
    badge.className = 'credit-badge';
    badge.title = '剩余 Credit（点击前往计费中心）';
    badge.onclick = function() {
      // 处在 /html/<模块>/ 子目录下就回退一级到 html 根，再进 account/billing.html（覆盖 taas / app-dev 等所有模块）
      var bp = /\/html\/[^/]+\//.test(window.location.pathname) ? '../' : '';
      window.open(bp + 'account/billing.html', '_blank');
    };

    function updateBadge() {
      var credit = parseFloat(localStorage.getItem('moi_credit') || '0');
      // credit 已从顶栏移除，余额「消化」进头像（账号）菜单的计费中心一行
      var pc = document.getElementById('popCredit');
      if (pc) {
        pc.textContent = (Math.round(credit * 100) / 100) + ' Credit';
        pc.className = 'pop-credit' + (credit < 5 ? ' critical' : (credit < 20 ? ' low' : ''));
      }
    }

    updateBadge();

    // Add Genesis button before credit badge (skip on Genesis pages)
    var _isTaas = window.location.pathname.indexOf('/taas/') !== -1;
    var _isAdmin = window.location.pathname.indexOf('/admin/') !== -1;
    if (false) {  // Genesis 入口已移入左上角产品切换器，不再单独占右上角
      if (!document.getElementById('taasBtnStyle')) {
        var taasStyle = document.createElement('style');
        taasStyle.id = 'taasBtnStyle';
        taasStyle.textContent = '.taas-btn{position:relative;display:inline-flex;align-items:center;gap:5px;padding:4px 14px;border-radius:16px;background:rgba(99,102,241,0.08);color:#5b5fc7;cursor:pointer;margin-right:6px;transition:all .2s;font-size:12px;font-weight:600;line-height:1;letter-spacing:0.3px;border:1px solid rgba(99,102,241,0.15);text-decoration:none;white-space:nowrap;}'
          + '.taas-btn svg{display:block;flex-shrink:0;}'
          + '.taas-btn:hover{background:rgba(99,102,241,0.14);border-color:rgba(99,102,241,0.25);transform:translateY(-1px);box-shadow:0 2px 8px rgba(99,102,241,0.12);}'
          + '.taas-btn[data-tip]::after{content:attr(data-tip);position:absolute;top:calc(100% + 11px);right:0;width:262px;background:rgba(255,255,255,0.85);backdrop-filter:blur(16px) saturate(180%);-webkit-backdrop-filter:blur(16px) saturate(180%);color:rgba(20,33,64,0.78);font-size:12px;font-weight:400;line-height:1.7;letter-spacing:0;text-align:left;white-space:normal;padding:12px 14px;border:1px solid rgba(0,74,240,0.12);border-radius:12px;box-shadow:0 10px 30px rgba(20,33,64,0.16),0 2px 8px rgba(20,33,64,0.06);z-index:9999;pointer-events:none;opacity:0;transform:translateY(-6px) scale(0.97);transform-origin:top right;transition:opacity .2s cubic-bezier(.16,1,.3,1),transform .2s cubic-bezier(.16,1,.3,1);}'
          + '.taas-btn[data-tip]::before{content:\'\';position:absolute;top:calc(100% + 6px);right:18px;width:11px;height:11px;background:rgba(255,255,255,0.85);backdrop-filter:blur(16px) saturate(180%);-webkit-backdrop-filter:blur(16px) saturate(180%);border-left:1px solid rgba(0,74,240,0.12);border-top:1px solid rgba(0,74,240,0.12);transform:rotate(45deg);z-index:10000;pointer-events:none;opacity:0;transition:opacity .2s cubic-bezier(.16,1,.3,1);}'
          + '.taas-btn[data-tip]:hover::after{opacity:1;transform:translateY(0) scale(1);}'
          + '.taas-btn[data-tip]:hover::before{opacity:1;}';
        document.head.appendChild(taasStyle);
      }
      var taasBtn = document.createElement('a');
      taasBtn.className = 'taas-btn';
      taasBtn.href = (function(){ var p=window.location.pathname; if(p.indexOf('/dashboard/')!==-1||p.indexOf('/data-connection/')!==-1||p.indexOf('/data-processing/')!==-1||p.indexOf('/resource-center/')!==-1||p.indexOf('/user-perm/')!==-1||p.indexOf('/account/')!==-1||p.indexOf('/app-dev/')!==-1) return '../taas/taas.html'; return 'taas/taas.html'; })();
      taasBtn.target = '_blank';
      taasBtn.setAttribute('data-tip', 'Genesis 是 MOI 的统一模型网关：一个 API 接入对话 / 嵌入 / OCR / 文生图等所有主流大模型，自动路由与计量计费。点击进入控制台。');
      taasBtn.innerHTML = '<svg width="18" height="18" viewBox="0 0 48 48" fill="none" stroke="currentColor"><ellipse cx="24" cy="24" rx="17" ry="8" stroke-width="2.6" transform="rotate(-45 24 24)"/><path d="M15 33 L15 15 L33 33 L33 15" stroke-width="3.8" stroke-linecap="round" stroke-linejoin="round"/><circle cx="15" cy="15" r="3.6" fill="currentColor" stroke="none"/><circle cx="33" cy="33" r="3.6" fill="currentColor" stroke="none"/></svg>Genesis';
      rightActions.insertBefore(taasBtn, badge);
    }

    // 刷新按钮已移除（与浏览器刷新重复）；顶栏右侧只留：MOMO · 点点点 · 头像

    // Listen for credit changes from other tabs
    window.addEventListener('storage', function(e) {
      if (e.key === 'moi_credit') updateBadge();
    });
  }

  // Run after buildTopBar has created the DOM
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', initCreditBadge);
  } else {
    setTimeout(initCreditBadge, 0);
  }
})();

// === Workspace Manager (shared across all pages) ===
(function() {
  var ver = localStorage.getItem('moi_ws_ver');
  if (ver !== '2') {
    localStorage.removeItem('moi_workspaces');
    localStorage.removeItem('moi_current_ws');
    localStorage.setItem('moi_ws_ver', '2');
  }
})();

function getWorkspaces() {
  var ws = localStorage.getItem('moi_workspaces');
  if (!ws) {
    var defaults = [
      { id: 'ws1', name: '默认工作区', owner: 'me', region: '华东-1' },
      { id: 'ws2', name: '数据分析项目', owner: 'me', region: '华东-1' },
      { id: 'ws3', name: '0818 演示', owner: 'me', region: '华北-1' },
      { id: 'ws4', name: '测试工作区', ownerName: '陈 jeff', owner: 'shared', region: '华东-1' },
      { id: 'ws5', name: 'MOI 2602 的工作区', ownerName: 'MOI 2602', owner: 'shared', region: '华南-1' },
      { id: 'ws6', name: 'Project test1', ownerName: '许 跃蓬', owner: 'shared', region: '华东-1' }
    ];
    localStorage.setItem('moi_workspaces', JSON.stringify(defaults));
    localStorage.setItem('moi_current_ws', 'ws1');
    return defaults;
  }
  return JSON.parse(ws);
}
function saveWorkspaces(list) { localStorage.setItem('moi_workspaces', JSON.stringify(list)); }
function getCurrentWsId() { return localStorage.getItem('moi_current_ws') || 'ws1'; }

function renderWsPanel() {
  var panel = document.getElementById('wsPanel');
  if (!panel) return;
  var list = getWorkspaces();
  var curId = getCurrentWsId();
  var mine = list.filter(function(w) { return w.owner === 'me'; });
  var shared = list.filter(function(w) { return w.owner !== 'me'; });
  var html = '<div class="ws-section-title">我的工作区</div>';
  mine.forEach(function(w) {
    html += '<div class="ws-item' + (w.id === curId ? ' active' : '') + '" data-id="' + w.id + '" onclick="selectWs(\'' + w.id + '\')">'
      + '<span class="ws-dot"></span>'
      + '<span class="ws-name">' + w.name + '</span>'
      + '<span style="flex-shrink:0;font-size:10.5px;color:#2b4acb;background:rgba(0,74,240,0.06);border:1px solid rgba(0,74,240,0.12);border-radius:4px;padding:0 5px;margin-left:6px">' + (w.region || '华东-1') + '</span>'
      + '<span class="ws-actions">'
        + '<button class="ws-act-btn" onclick="event.stopPropagation();copyWsId(\'' + w.id + '\',this)" title="复制工作区 ID"><svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="9" y="9" width="13" height="13" rx="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg></button>'
        + '<button class="ws-act-btn" onclick="event.stopPropagation();renameWs(\'' + w.id + '\')" title="重命名">✏️</button>'
        + (w.id !== 'ws1' ? '<button class="ws-act-btn danger" onclick="event.stopPropagation();deleteWs(\'' + w.id + '\')" title="删除">🗑️</button>' : '')
      + '</span></div>';
  });
  if (shared.length) {
    html += '<div class="ws-divider"></div><div class="ws-section-title">我加入的工作区</div>';
    shared.forEach(function(w) {
      html += '<div class="ws-item' + (w.id === curId ? ' active' : '') + '" data-id="' + w.id + '" onclick="selectWs(\'' + w.id + '\')">'
        + '<span class="ws-dot"></span>'
        + '<span class="ws-name">' + w.name + '</span>'
        + '<span style="flex-shrink:0;font-size:10.5px;color:#2b4acb;background:rgba(0,74,240,0.06);border:1px solid rgba(0,74,240,0.12);border-radius:4px;padding:0 5px;margin-left:6px">' + (w.region || '华东-1') + '</span>'
        + '<span class="ws-actions">'
          + '<button class="ws-act-btn" onclick="event.stopPropagation();copyWsId(\'' + w.id + '\',this)" title="复制工作区 ID"><svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="9" y="9" width="13" height="13" rx="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg></button>'
        + '</span>'
        + '<span class="ws-owner-badge">' + (w.ownerName || '未知') + '</span>'
        + '</div>';
    });
  }
  html += '<div class="ws-divider"></div><div class="ws-add" onclick="addWs()">＋ 添加工作区</div>';
  panel.innerHTML = html;
  var cur = list.find(function(w) { return w.id === curId; });
  var nameEl = document.getElementById('wsCurrentName');
  if (nameEl) nameEl.textContent = cur ? cur.name : '默认工作区';
}

function toggleWsPanel() {
  var p = document.getElementById('wsPanel');
  if (!p) return;
  if (p.classList.contains('show')) { p.classList.remove('show'); return; }
  renderWsPanel();
  p.classList.add('show');
}
function selectWs(id) {
  localStorage.setItem('moi_current_ws', id);
  var p = document.getElementById('wsPanel');
  if (p) p.classList.remove('show');
  renderWsPanel();
}
function gotoGenesis() {
  var p = window.location.pathname, base = '';
  if (p.indexOf('/dashboard/') !== -1 || p.indexOf('/data-connection/') !== -1 || p.indexOf('/data-processing/') !== -1 || p.indexOf('/resource-center/') !== -1 || p.indexOf('/user-perm/') !== -1 || p.indexOf('/account/') !== -1 || p.indexOf('/app-dev/') !== -1 || p.indexOf('/monitor/') !== -1) base = '../';
  window.location.href = base + 'taas/taas.html';
}
function gotoMatrixOne() {
  var path = window.location.pathname;
  var dirs = ['/dashboard/','/data-connection/','/data-processing/','/resource-center/','/user-perm/','/monitor/','/account/','/app-dev/','/taas/','/matrixone/','/admin/'];
  var base = '';
  for (var i = 0; i < dirs.length; i++) { if (path.indexOf(dirs[i]) !== -1) { base = '../'; break; } }
  window.location.href = base + 'matrixone/matrixone.html';
}
// 从 Genesis / 其他产品切回工作区（落到上次数据页或仪表盘）
function gotoWorkspace(basePath) {
  basePath = basePath || '';
  var last = localStorage.getItem('moi_last_data_page');
  window.location.href = last ? basePath + last : basePath + 'dashboard/dashboard.html';
}
function addWs() {
  var p = document.getElementById('wsPanel');
  if (p) p.classList.remove('show');
  var modal = document.getElementById('addWsModal');
  if (!modal) return;
  document.getElementById('addWsName').value = '';
  // 地区选项跟随当前服务站点(由站点门户写入;缺省为默认站点的地区)
  var regionSel = document.getElementById('addWsRegion');
  try {
    var site = JSON.parse(localStorage.getItem('moi_current_site') || 'null');
    if (regionSel && site && site.regions && site.regions.length) {
      regionSel.innerHTML = site.regions.map(function(r) { return '<option>' + r + '</option>'; }).join('');
    }
  } catch (e) {}
  modal.classList.add('show');
  setTimeout(function() { document.getElementById('addWsName').focus(); }, 100);
}
function closeAddWsModal() {
  var modal = document.getElementById('addWsModal');
  if (modal) modal.classList.remove('show');
}
function confirmAddWs() {
  var name = document.getElementById('addWsName').value.trim();
  if (!name) { document.getElementById('addWsName').focus(); return; }
  var list = getWorkspaces();
  var id = 'ws' + Date.now();
  var regionSel = document.getElementById('addWsRegion');
  list.push({ id: id, name: name, owner: 'me', region: regionSel ? regionSel.value : '华东-1' });
  saveWorkspaces(list);
  localStorage.setItem('moi_current_ws', id);
  closeAddWsModal();
  renderWsPanel();
}
function renameWs(id) {
  var list = getWorkspaces();
  var ws = list.find(function(w) { return w.id === id; });
  if (!ws) return;
  var el = document.querySelector('.ws-item[data-id="' + id + '"] .ws-name');
  el.innerHTML = '<input value="' + ws.name + '" onblur="saveRename(\'' + id + '\',this.value)" onkeydown="if(event.key===\'Enter\')this.blur()" autofocus>';
  el.querySelector('input').focus();
  el.querySelector('input').select();
}
function saveRename(id, val) {
  var list = getWorkspaces();
  var ws = list.find(function(w) { return w.id === id; });
  if (ws && val.trim()) { ws.name = val.trim(); saveWorkspaces(list); }
  renderWsPanel();
}
function deleteWs(id) {
  if (!confirm('确定删除该工作区？')) return;
  var list = getWorkspaces().filter(function(w) { return w.id !== id; });
  saveWorkspaces(list);
  if (getCurrentWsId() === id && list.length) localStorage.setItem('moi_current_ws', list[0].id);
  renderWsPanel();
}

function copyWsId(id, btn) {
  if (navigator.clipboard) {
    navigator.clipboard.writeText(id).then(function() {
      var orig = btn.innerHTML;
      btn.innerHTML = '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="#52c41a" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><path d="M20 6L9 17l-5-5"/></svg>';
      btn.title = '已复制';
      setTimeout(function() { btn.innerHTML = orig; btn.title = '复制工作区 ID'; }, 1500);
    });
  } else {
    var ta = document.createElement('textarea');
    ta.value = id; document.body.appendChild(ta); ta.select();
    document.execCommand('copy'); document.body.removeChild(ta);
    alert('已复制工作区 ID: ' + id);
  }
}

// Close ws-panel on outside click
document.addEventListener('click', function(e) {
  if (!e.target.closest('.ws-selector')) {
    var p = document.getElementById('wsPanel');
    if (p) p.classList.remove('show');
  }
});

// Init workspace name on page load
renderWsPanel();

// === Save last visited data-dev page ===
(function() {
  var path = window.location.pathname;
  var dataPages = ['/data-connection/', '/data-processing/', '/resource-center/'];
  var isDataPage = dataPages.some(function(p) { return path.indexOf(p) !== -1; });
  if (isDataPage) {
    // Store relative path from html/ root (e.g. "dashboard/dashboard.html")
    var match = path.match(/(data-connection|data-processing|resource-center)\/.+\.html$/);
    if (match) localStorage.setItem('moi_last_data_page', match[0]);
  }
})();

// Refresh button - spin animation + simulate data refresh
function refreshData(btn) {
  if (btn.classList.contains('spinning')) return;
  btn.classList.add('spinning');
  setTimeout(function() { btn.classList.remove('spinning'); }, 600);
}


// === Shared Notebook Data (used by notebook.html list & workflow-edit.html professional mode) ===
var MOI_NOTEBOOKS = [
  { id: 'nb1', name: '产品文档处理脚本', lang: 'python', langName: 'Python',
    desc: '产品文档全流程处理，包含解析、分段、清洗和嵌入',
    updated: '2026-03-14 10:30', author: '张三',
    operators: [{name:'解析',type:'system'},{name:'分段',type:'system'},{name:'清洗',type:'system'},{name:'嵌入',type:'system'}] },
  { id: 'nb2', name: '合同关键信息提取', lang: 'python', langName: 'Python',
    desc: '从合同 PDF 中提取甲乙方、金额、日期等关键字段',
    updated: '2026-03-13 16:20', author: '李四',
    operators: [{name:'解析',type:'system'},{name:'提取',type:'system'},{name:'合同字段映射',type:'custom'}] },
  { id: 'nb3', name: 'Catalog 数据质量分析', lang: 'sql', langName: 'SQL',
    desc: '对 Catalog 中的处理数据进行质量统计和异常检测',
    updated: '2026-03-12 09:00', author: '张三',
    operators: [] },
  { id: 'nb4', name: '自定义去敏处理', lang: 'python', langName: 'Python',
    desc: '对文档中的手机号、身份证号等敏感信息进行脱敏处理',
    updated: '2026-03-11 14:45', author: '王五',
    operators: [{name:'解析',type:'system'},{name:'清洗',type:'system'},{name:'PII 脱敏',type:'custom'},{name:'正则替换',type:'custom'}] },
  { id: 'nb5', name: '多模态数据处理流水线', lang: 'mixed', langName: '混合',
    desc: '图文混合文档的端到端处理，含 Python 处理逻辑和 SQL 分析',
    updated: '2026-03-10 11:00', author: '李四',
    operators: [{name:'解析',type:'system'},{name:'分段',type:'system'},{name:'嵌入',type:'system'},{name:'增强',type:'system'},{name:'图片描述生成',type:'custom'}] }
];


// ===== Neural Network Background =====
(function() {
  function injectBg() {
    if (document.querySelector('.grid-bg')) return;
    if (window.location.pathname.indexOf('/taas/') !== -1) return;
    if (window.location.pathname.indexOf('/admin/') !== -1) return;
    var wrap = document.createElement('div');
    wrap.className = 'grid-bg';
    var canvas = document.createElement('canvas');
    wrap.appendChild(canvas);
    document.body.insertBefore(wrap, document.body.firstChild);

    var ctx = canvas.getContext('2d');
    var dpr = Math.min(window.devicePixelRatio || 1, 2);
    var w, h;
    var mouseX = -9999, mouseY = -9999;
    var nodes = [], edges = [];
    var nodeCount = 45;
    var maxEdgeDist = 140;

    function resize() {
      w = window.innerWidth; h = window.innerHeight;
      canvas.width = w * dpr; canvas.height = h * dpr;
      canvas.style.width = w + 'px'; canvas.style.height = h + 'px';
      ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
      initNodes();
    }

    function initNodes() {
      nodes = [];
      for (var i = 0; i < nodeCount; i++) {
        nodes.push({
          x: Math.random() * w,
          y: Math.random() * h,
          vx: (Math.random() - 0.5) * 0.3,
          vy: (Math.random() - 0.5) * 0.3,
          r: Math.random() * 2 + 1.5,
          phase: Math.random() * Math.PI * 2,
          layer: Math.floor(Math.random() * 3) // 0=input, 1=hidden, 2=output
        });
      }
      buildEdges();
    }

    function buildEdges() {
      edges = [];
      for (var i = 0; i < nodes.length; i++) {
        for (var j = i + 1; j < nodes.length; j++) {
          var dx = nodes[i].x - nodes[j].x;
          var dy = nodes[i].y - nodes[j].y;
          var dist = Math.sqrt(dx * dx + dy * dy);
          if (dist < maxEdgeDist) {
            edges.push({ a: i, b: j, pulseOffset: Math.random() * Math.PI * 2 });
          }
        }
      }
    }

    resize();
    window.addEventListener('resize', resize);
    document.addEventListener('mousemove', function(e) { mouseX = e.clientX; mouseY = e.clientY; });

    var time = 0;
    var edgeRebuildTimer = 0;
    var pulses = []; // Energy pulses traveling along edges
    var mouseTrail = []; // Mouse trail points

    function draw() {
      time += 0.015;
      edgeRebuildTimer++;
      ctx.clearRect(0, 0, w, h);

      // Update mouse trail
      if (mouseX > 0 && mouseY > 0) {
        mouseTrail.push({ x: mouseX, y: mouseY, age: 0 });
      }
      mouseTrail = mouseTrail.filter(function(p) { p.age += 0.02; return p.age < 1; });

      // Draw mouse trail
      for (var i = 1; i < mouseTrail.length; i++) {
        var p = mouseTrail[i], pp = mouseTrail[i - 1];
        var a = (1 - p.age) * 0.08;
        ctx.strokeStyle = 'rgba(0, 120, 240, ' + a + ')';
        ctx.lineWidth = (1 - p.age) * 2;
        ctx.beginPath();
        ctx.moveTo(pp.x, pp.y);
        ctx.lineTo(p.x, p.y);
        ctx.stroke();
      }

      // Background breathing lamp — subtle color waves
      var b1 = Math.sin(time * 0.3) * 0.5 + 0.5;
      var b2 = Math.sin(time * 0.3 + 2.1) * 0.5 + 0.5;
      var b3 = Math.sin(time * 0.3 + 4.2) * 0.5 + 0.5;

      ctx.fillStyle = 'rgba(0, 50, 200, ' + (b1 * 0.005) + ')';
      ctx.fillRect(0, 0, w, h);
      ctx.fillStyle = 'rgba(0, 190, 170, ' + (b2 * 0.004) + ')';
      ctx.fillRect(0, 0, w, h);
      ctx.fillStyle = 'rgba(120, 60, 220, ' + (b3 * 0.003) + ')';
      ctx.fillRect(0, 0, w, h);

      // Update nodes
      for (var i = 0; i < nodes.length; i++) {
        var n = nodes[i];
        n.x += n.vx;
        n.y += n.vy;
        n.phase += 0.02;
        // Soft bounce
        if (n.x < 0 || n.x > w) n.vx *= -1;
        if (n.y < 0 || n.y > h) n.vy *= -1;
        n.x = Math.max(0, Math.min(w, n.x));
        n.y = Math.max(0, Math.min(h, n.y));
      }

      // Rebuild edges periodically
      if (edgeRebuildTimer > 60) { buildEdges(); edgeRebuildTimer = 0; }

      // Draw edges with pulse
      for (var i = 0; i < edges.length; i++) {
        var e = edges[i];
        var a = nodes[e.a], b = nodes[e.b];
        var dx = a.x - b.x, dy = a.y - b.y;
        var dist = Math.sqrt(dx * dx + dy * dy);
        if (dist > maxEdgeDist) continue;

        var baseAlpha = (1 - dist / maxEdgeDist) * 0.2;

        // Pulse traveling along edge
        var pulse = Math.sin(time * 2 + e.pulseOffset) * 0.5 + 0.5;

        // Brighter near mouse
        var mx = (a.x + b.x) / 2, my = (a.y + b.y) / 2;
        var md = Math.sqrt((mx - mouseX) * (mx - mouseX) + (my - mouseY) * (my - mouseY));
        var mouseBoost = md < 200 ? (1 - md / 200) * 0.25 : 0;

        var alpha = baseAlpha + pulse * 0.04 + mouseBoost;

        // Gradient along edge
        var grad = ctx.createLinearGradient(a.x, a.y, b.x, b.y);
        grad.addColorStop(0, 'rgba(0, 74, 240, ' + alpha * 0.5 + ')');
        grad.addColorStop(pulse, 'rgba(0, 180, 200, ' + alpha * 1.5 + ')');
        grad.addColorStop(1, 'rgba(100, 80, 220, ' + alpha * 0.5 + ')');

        ctx.strokeStyle = grad;
        ctx.lineWidth = 0.6 + mouseBoost * 3;
        ctx.beginPath();
        ctx.moveTo(a.x, a.y);
        ctx.lineTo(b.x, b.y);
        ctx.stroke();

        // Pulse dot traveling along edge
        var px = a.x + (b.x - a.x) * pulse;
        var py = a.y + (b.y - a.y) * pulse;
        var dotAlpha = alpha * 2;
        if (dotAlpha > 0.03) {
          ctx.fillStyle = 'rgba(0, 180, 220, ' + Math.min(dotAlpha, 0.4) + ')';
          ctx.beginPath();
          ctx.arc(px, py, 1.2 + mouseBoost * 2, 0, Math.PI * 2);
          ctx.fill();
        }
      }

      // Spawn random energy pulses
      if (Math.random() < 0.03 && edges.length > 0) {
        var ei = Math.floor(Math.random() * edges.length);
        pulses.push({ edge: ei, t: 0, speed: 0.01 + Math.random() * 0.02, size: 2 + Math.random() * 2 });
      }

      // Draw energy pulses
      pulses = pulses.filter(function(p) {
        p.t += p.speed;
        if (p.t > 1 || p.edge >= edges.length) return false;
        var e = edges[p.edge];
        var a = nodes[e.a], b = nodes[e.b];
        if (!a || !b) return false;
        var px = a.x + (b.x - a.x) * p.t;
        var py = a.y + (b.y - a.y) * p.t;
        // Bright core
        ctx.fillStyle = 'rgba(0, 212, 170, 0.6)';
        ctx.beginPath();
        ctx.arc(px, py, p.size * 0.5, 0, Math.PI * 2);
        ctx.fill();
        // Glow
        var g = ctx.createRadialGradient(px, py, 0, px, py, p.size * 3);
        g.addColorStop(0, 'rgba(0, 212, 170, 0.2)');
        g.addColorStop(1, 'rgba(0, 212, 170, 0)');
        ctx.fillStyle = g;
        ctx.beginPath();
        ctx.arc(px, py, p.size * 3, 0, Math.PI * 2);
        ctx.fill();
        return true;
      });

      // Draw nodes
      for (var i = 0; i < nodes.length; i++) {
        var n = nodes[i];
        var glow = Math.sin(n.phase) * 0.5 + 0.5;
        var dx = n.x - mouseX, dy = n.y - mouseY;
        var md = Math.sqrt(dx * dx + dy * dy);
        var near = md < 180;

        // Colors by layer
        var colors = [
          [0, 74, 240],   // input: blue
          [0, 180, 170],  // hidden: cyan
          [139, 124, 240] // output: purple
        ];
        var c = colors[n.layer];

        // Outer glow
        if (near) {
          var ga = (1 - md / 180) * 0.08;
          var gs = n.r + (1 - md / 180) * 8;
          ctx.fillStyle = 'rgba(' + c[0] + ',' + c[1] + ',' + c[2] + ',' + ga + ')';
          ctx.beginPath();
          ctx.arc(n.x, n.y, gs, 0, Math.PI * 2);
          ctx.fill();
        }

        // Core
        var coreAlpha = 0.2 + glow * 0.12 + (near ? (1 - md / 180) * 0.3 : 0);
        var coreR = n.r + glow * 0.8 + (near ? (1 - md / 180) * 2.5 : 0);
        ctx.fillStyle = 'rgba(' + c[0] + ',' + c[1] + ',' + c[2] + ',' + coreAlpha + ')';
        ctx.beginPath();
        ctx.arc(n.x, n.y, coreR, 0, Math.PI * 2);
        ctx.fill();

        // Bright center
        ctx.fillStyle = 'rgba(255, 255, 255, ' + (coreAlpha * 0.4) + ')';
        ctx.beginPath();
        ctx.arc(n.x, n.y, coreR * 0.4, 0, Math.PI * 2);
        ctx.fill();
      }

      // Mouse ripple ring
      if (mouseX > 0 && mouseY > 0) {
        var rippleR = (time * 30) % 60;
        var rippleA = (1 - rippleR / 60) * 0.08;
        ctx.strokeStyle = 'rgba(0, 120, 240, ' + rippleA + ')';
        ctx.lineWidth = 1;
        ctx.beginPath();
        ctx.arc(mouseX, mouseY, rippleR, 0, Math.PI * 2);
        ctx.stroke();
        // Second ring offset
        var rippleR2 = ((time * 30) + 30) % 60;
        var rippleA2 = (1 - rippleR2 / 60) * 0.05;
        ctx.strokeStyle = 'rgba(0, 180, 200, ' + rippleA2 + ')';
        ctx.beginPath();
        ctx.arc(mouseX, mouseY, rippleR2, 0, Math.PI * 2);
        ctx.stroke();
      }

      requestAnimationFrame(draw);
    }
    draw();
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', injectBg);
  } else {
    injectBg();
  }
})();

// ===== 操作图标 hover 文字气泡:旧写法兜底 =====
// 部分页面的 .op-btn 仍以 title 提供提示(原生 tooltip 延迟高)。首次 hover 时把 title
// 迁移为 data-tip,交给 common.css 的即时气泡;新代码请直接用 data-tip(见 op-icons.js)。
(function () {
  document.addEventListener('mouseover', function (e) {
    var t = e.target;
    var btn = t && t.closest ? t.closest('.op-btn[title]') : null;
    if (!btn) return;
    btn.setAttribute('data-tip', btn.getAttribute('title'));
    btn.removeAttribute('title');
  });
})();

// === 奖励中心弹窗(签到 + 邀请):全局注入,头像菜单「奖励中心」触发 ===
var rwState = { done: 3, today: false, tab: 'checkin' };
// 累计签到:15 天一轮,每日 +100;本轮累计第 3 / 7 / 15 次里程碑额外 +200 / +500 / +1000,断签不清进度,轮结束重置
var RW_PLAN = (function() {
  var a = [];
  for (var i = 0; i < 15; i++) a.push({ d: '第 ' + (i + 1) + ' 次', amt: 100, bonus: i === 2 ? 200 : (i === 6 ? 500 : (i === 14 ? 1000 : 0)) });
  return a;
})();

function openRewardsModal() {
  if (!document.getElementById('rwModalStyle')) {
    var st = document.createElement('style');
    st.id = 'rwModalStyle';
    st.textContent = ''
      + '.rw-overlay{position:fixed;inset:0;background:rgba(0,0,0,0.4);z-index:3000;display:none;align-items:center;justify-content:center}'
      + '.rw-overlay.open{display:flex}'
      + '.rw-modal{background:#fff;border-radius:14px;width:560px;max-height:82vh;overflow-y:auto;box-shadow:0 8px 40px rgba(0,0,0,0.16)}'
      + '.rw-m-head{display:flex;align-items:center;justify-content:space-between;padding:16px 22px 0}'
      + '.rw-m-title{font-size:16px;font-weight:700;color:rgba(0,0,0,0.85)}'
      + '.rw-m-close{border:none;background:none;font-size:15px;color:rgba(0,0,0,0.4);cursor:pointer;padding:4px}'
      + '.rw-m-close:hover{color:rgba(0,0,0,0.7)}'
      + '.rw-tabs{display:flex;gap:20px;padding:8px 22px 0;border-bottom:1px solid #f0f0f0}'
      + '.rw-tab{padding:8px 2px;font-size:13.5px;color:rgba(0,0,0,0.55);cursor:pointer;border-bottom:2px solid transparent;margin-bottom:-1px}'
      + '.rw-tab.on{color:#1677ff;font-weight:600;border-bottom-color:#1677ff}'
      + '.rw-body{padding:18px 22px 22px}'
      + '.rw-desc{font-size:12.5px;color:rgba(0,0,0,0.45);line-height:1.7;margin-bottom:14px}'
      + '.rw-strip{display:grid;grid-template-columns:repeat(5,1fr);gap:9px;margin-bottom:16px}'
      + '.rw-d{border:1px solid #eef0f4;border-radius:11px;padding:10px 4px 8px;text-align:center;position:relative;background:#fafbfd;transition:all .15s}'
      + '.rw-d .dd{font-size:10.5px;color:rgba(0,0,0,0.4)}'
      + '.rw-d .aa{font-size:13px;font-weight:700;color:rgba(0,0,0,0.72);margin-top:3px}'
      + '.rw-d .bb{margin-top:4px;height:16px;display:flex;align-items:center;justify-content:center}'
      + '.rw-d .bb .cap{font-size:9.5px;padding:0 7px;line-height:15px;border-radius:8px;background:rgba(245,34,45,0.08);color:#f5222d;font-weight:600}'
      + '.rw-d.done{background:linear-gradient(160deg,#fff8ec,#ffefd8);border-color:#ffd591}'
      + '.rw-d.done .aa{color:#d46b08}.rw-d.done .dd{color:rgba(212,107,8,0.55)}'
      + '.rw-d.today{background:#fff;border:1.5px solid #fa8c16;box-shadow:0 4px 14px rgba(250,140,22,0.18)}'
      + '.rw-d.today .dd{color:#fa8c16;font-weight:600}'
      + '.rw-d.grand{background:linear-gradient(160deg,#fffbe0,#ffe14d);border-color:#ffd400;box-shadow:0 2px 10px rgba(255,212,0,0.35)}'
      + '.rw-d.grand .aa{color:#8a5a00}.rw-d.grand .dd{color:#a06800}'
      + '.rw-d .tk{position:absolute;top:-6px;right:-6px;width:17px;height:17px;border-radius:50%;background:linear-gradient(135deg,#fa8c16,#ffc53d);color:#fff;font-size:10px;display:flex;align-items:center;justify-content:center;box-shadow:0 2px 6px rgba(250,140,22,0.4)}'
      + '.rw-bar{display:flex;align-items:center;justify-content:space-between}'
      + '.rw-stat{font-size:13px;color:rgba(0,0,0,0.6)}.rw-stat b{color:#fa8c16;font-size:15px}'
      + '.rw-go{background:linear-gradient(135deg,#fa8c16,#ffa940);color:#fff;border:none;padding:8px 24px;border-radius:8px;font-size:13.5px;font-weight:600;cursor:pointer;box-shadow:0 4px 12px rgba(250,140,22,0.3)}'
      + '.rw-go:hover{background:linear-gradient(135deg,#ffa940,#ffc069)}.rw-go:disabled{background:#d9d9d9;box-shadow:none;cursor:not-allowed}'
      + '.rw-linkrow{display:flex;gap:8px;margin-bottom:12px}'
      + '.rw-link{flex:1;height:34px;display:flex;align-items:center;padding:0 12px;background:#f7f9fc;border:1px solid #e8ecf2;border-radius:8px;font-family:monospace;font-size:11.5px;color:rgba(0,0,0,0.65);overflow:hidden;white-space:nowrap;text-overflow:ellipsis}'
      + '.rw-cp{background:#1677ff;color:#fff;border:none;padding:0 14px;border-radius:8px;font-size:12.5px;cursor:pointer;white-space:nowrap}'
      + '.rw-cp:hover{background:#4096ff}'
      + '.rw-cp.ghost{background:#fff;color:#1677ff;border:1px solid #1677ff}'
      + '.rw-meta{display:flex;gap:22px;margin-bottom:6px}'
      + '.rw-meta .v{font-size:17px;font-weight:700;color:rgba(0,0,0,0.85)}.rw-meta .v small{font-size:11px;font-weight:400;color:rgba(0,0,0,0.4)}'
      + '.rw-meta .l{font-size:11.5px;color:rgba(0,0,0,0.45);margin-top:1px}'
      + '.rw-tb{width:100%;border-collapse:collapse;margin-top:10px}'
      + '.rw-tb th{text-align:left;padding:7px 10px;font-size:11.5px;color:rgba(0,0,0,0.4);font-weight:600;background:#fafafa;border-bottom:1px solid #f0f0f0}'
      + '.rw-tb td{padding:9px 10px;font-size:12.5px;color:rgba(0,0,0,0.78);border-bottom:1px solid #f5f5f5}'
      + '.rw-tb tr:last-child td{border-bottom:none}'
      + '.rw-st{display:inline-block;font-size:11px;padding:0 7px;border-radius:4px;line-height:18px}'
      + '.rw-st.ok{background:#f6ffed;color:#389e0d;border:1px solid #b7eb8f}'
      + '.rw-st.wt{background:#fff7e6;color:#d46b08;border:1px solid #ffd591}'
      + '.rw-rule{font-size:11.5px;color:rgba(0,0,0,0.45);background:#f7f9fc;border:1px solid #eef2f7;border-radius:8px;padding:8px 12px;margin-top:12px;line-height:1.7}'
      + '.rw-tst{position:fixed;top:70px;left:50%;transform:translateX(-50%);background:rgba(0,0,0,0.78);color:#fff;padding:8px 16px;border-radius:8px;font-size:12.5px;z-index:3100;display:none}';
    document.head.appendChild(st);
  }
  var ov = document.getElementById('rwOverlay');
  if (!ov) {
    ov = document.createElement('div');
    ov.id = 'rwOverlay';
    ov.className = 'rw-overlay';
    ov.onclick = function(e) { if (e.target === ov) closeRewardsModal(); };
    ov.innerHTML = '<div class="rw-modal">'
      + '<div class="rw-m-head"><div class="rw-m-title">奖励中心</div><button class="rw-m-close" onclick="closeRewardsModal()">✕</button></div>'
      + '<div class="rw-tabs">'
      + '<div class="rw-tab" id="rwTab-checkin" onclick="rwSwitch(\'checkin\')">每日签到</div>'
      + '<div class="rw-tab" id="rwTab-invite" onclick="rwSwitch(\'invite\')">邀请有礼</div>'
      + '</div>'
      + '<div class="rw-body" id="rwBody"></div>'
      + '</div>';
    document.body.appendChild(ov);
    var tst = document.createElement('div');
    tst.id = 'rwTst'; tst.className = 'rw-tst';
    document.body.appendChild(tst);
  }
  ov.classList.add('open');
  rwSwitch(rwState.tab);
}
function closeRewardsModal() {
  var ov = document.getElementById('rwOverlay');
  if (ov) ov.classList.remove('open');
}
function rwSwitch(tab) {
  rwState.tab = tab;
  var a = document.getElementById('rwTab-checkin'), b = document.getElementById('rwTab-invite');
  if (a) a.classList.toggle('on', tab === 'checkin');
  if (b) b.classList.toggle('on', tab === 'invite');
  document.getElementById('rwBody').innerHTML = tab === 'checkin' ? rwCheckinHtml() : rwInviteHtml();
}
function rwCheckinHtml() {
  var gained = 0;
  for (var i = 0; i < rwState.done; i++) gained += RW_PLAN[i].amt + RW_PLAN[i].bonus;
  return '<div class="rw-desc">每日签到得 100 Credit，累计签到会有额外奖励。<b>活动有效期为注册后 20 天</b>（您的活动截止 2026-09-10，还剩 12 天），奖励直接计入账户余额。</div>'
    + '<div class="rw-strip">' + RW_PLAN.map(function(p, i) {
      var cls = 'rw-d' + (i < rwState.done ? ' done' : '') + (i === rwState.done && !rwState.today ? ' today' : '') + (i === 14 ? ' grand' : '');
      return '<div class="' + cls + '">' + (i < rwState.done ? '<span class="tk">✓</span>' : '')
        + '<div class="dd">' + p.d + '</div><div class="aa">+' + p.amt + '</div><div class="bb">' + (p.bonus ? '<span class="cap">额外 +' + p.bonus + '</span>' : '') + '</div></div>';
    }).join('') + '</div>'
    + '<div class="rw-bar">'
    + '<div class="rw-stat">本轮已签 <b>' + rwState.done + '</b> 次 · 已得 <b>' + gained + '</b> cr</div>'
    + '<button class="rw-go" ' + (rwState.today ? 'disabled' : '') + ' onclick="rwCheckin()">' + (rwState.today ? '今日已签到' : '今日签到 +100 cr') + '</button>'
    + '</div>';
}
function rwCheckin() {
  if (rwState.today) return;
  var p = RW_PLAN[rwState.done];
  rwState.done++; rwState.today = true;
  rwSwitch('checkin');
  var dot = document.getElementById('rwTopDot');
  if (dot) dot.style.display = 'none';
  rwToastMsg('签到成功 +' + p.amt + ' cr' + (p.bonus ? '，本轮累计 ' + rwState.done + ' 次额外 +' + p.bonus + ' cr' : ''));
}
// 邀请链接指向原型注册页(带 invite 参数),按当前访问方式(file:// / localhost)动态生成,复制即可打开
function rwInviteUrl() {
  var p = location.pathname;
  var i = p.indexOf('/html/');
  var base = i >= 0 ? p.slice(0, i + 6) : p.replace(/[^/]*$/, '');
  return location.protocol + '//' + (location.host || '') + base + 'website/register.html?invite=MOI-U8F3KD';
}
function rwInviteHtml() {
  return '<div class="rw-desc">邀请新用户注册 MOI：对方通过您的链接完成注册激活后，您获得 <b>500 Credit</b> / 人，对方获得 <b>200 Credit</b>；每个账户最多可邀请 <b>10</b> 人。手机 / GitHub 注册即时激活；邮箱注册需完成邮箱验证。</div>'
    + '<div class="rw-linkrow">'
    + '<div class="rw-link" id="rwInvLink">' + rwInviteUrl() + '</div>'
    + '<button class="rw-cp" onclick="rwCopy(document.getElementById(\'rwInvLink\').textContent,\'邀请链接已复制\')">复制链接</button>'
    + '</div>'
    + '<div style="font-size:11.5px;color:rgba(0,0,0,0.45);margin:2px 0 10px">将链接发给好友，对方通过此链接完成注册激活后，奖励自动发放到双方账户。</div>'
    + '<div class="rw-meta">'
    + '<div><div class="v">3 <small>/ 10 人</small></div><div class="l">已成功邀请</div></div>'
    + '<div><div class="v">1,500 <small>cr</small></div><div class="l">累计获得</div></div>'
    + '<div><div class="v">7 <small>人</small></div><div class="l">剩余名额</div></div>'
    + '</div>'
    + '<table class="rw-tb"><thead><tr><th>被邀请用户</th><th>注册方式</th><th>注册时间</th><th>状态</th><th style="text-align:right">获得奖励</th></tr></thead><tbody>'
    + '<tr><td>138****6621</td><td>手机</td><td>2026-08-20 14:32</td><td><span class="rw-st ok">已发放</span></td><td style="text-align:right;color:#1677ff;font-weight:600">+500 cr</td></tr>'
    + '<tr><td>oc***hub</td><td>GitHub</td><td>2026-08-15 09:18</td><td><span class="rw-st ok">已发放</span></td><td style="text-align:right;color:#1677ff;font-weight:600">+500 cr</td></tr>'
    + '<tr><td>li***@163.com</td><td>邮箱</td><td>2026-08-27 20:05</td><td><span class="rw-st wt">待激活</span></td><td style="text-align:right;color:rgba(0,0,0,0.35)">—</td></tr>'
    + '</tbody></table>'
    + '<div class="rw-rule">被邀请用户须为新用户，完成注册激活后奖励自动发放（手机 / GitHub 注册即时激活并发放；邮箱注册待完成邮箱验证后发放）。同一设备 / 同一实名 / 同一支付账号重复注册不计入；检测到异常刷取将回收 Credit 并可能限制账户功能。</div>';
}
function rwCopy(text, msg) {
  if (navigator.clipboard) navigator.clipboard.writeText(text);
  rwToastMsg(msg);
}
var rwTstTimer = null;
function rwToastMsg(msg) {
  var t = document.getElementById('rwTst');
  if (!t) return;
  t.textContent = msg; t.style.display = 'block';
  clearTimeout(rwTstTimer);
  rwTstTimer = setTimeout(function() { t.style.display = 'none'; }, 2400);
}

// === 问题反馈浮窗:右下角悬浮入口(MOMO 上方),对话式新增反馈 + 反馈历史两页签 ===
var fbwState = { tab: 'new', step: 0 };
var FBW_LIST = [
  { id: 'FB-20260826-012', title: '工作流定时任务在 02:00 未按时触发', cat: '缺陷', st: 'replied', stLabel: '已回复', time: '2026-08-26', reply: '官方回复：已确认为有效反馈，100 Credit 已发放至您的账户。问题定位为调度时区配置缺陷，修复将随下一次发布上线；期间可将触发时间避开整点。' },
  { id: 'FB-20260822-007', title: '希望知识库支持按文件夹批量更新', cat: '需求', st: 'doing', stLabel: '处理中', time: '2026-08-22', reply: '' },
  { id: 'FB-20260815-003', title: 'CSV 导入含中文表头时列名乱码', cat: '缺陷', st: 'done', stLabel: '已解决', time: '2026-08-15', reply: '官方回复：编码探测缺陷已修复并上线，感谢反馈。' }
];

function fbwEnsure() {
  if (document.getElementById('fbwPanel')) return;
  var st = document.createElement('style');
  st.id = 'fbwStyle';
  st.textContent = ''
    + '.fbw-fab{position:fixed;right:0;bottom:190px;z-index:1000;width:34px;padding:11px 0 13px;border-radius:10px 0 0 10px;background:linear-gradient(180deg,#1677ff,#3f8dff);color:#fff;display:flex;flex-direction:column;align-items:center;gap:5px;cursor:pointer;box-shadow:-3px 4px 16px rgba(22,119,255,0.28);transition:all .18s}'
    + '.fbw-fab span{font-size:12px;line-height:1.35;font-weight:500;letter-spacing:1px}'
    + '.fbw-fab:hover{transform:translateX(-2px);box-shadow:-5px 6px 20px rgba(22,119,255,0.36)}'
    + 'body.fbw-open .fbw-fab{opacity:0;pointer-events:none}'
    + '.fbw-panel{position:fixed;right:22px;bottom:60px;z-index:1001;width:460px;height:680px;max-height:calc(100vh - 110px);background:#fff;border-radius:14px;box-shadow:0 12px 48px rgba(0,0,0,0.18);display:none;flex-direction:column;overflow:hidden}'
    + 'body.fbw-open .fbw-panel{display:flex}'
    + '.fbw-head{display:flex;align-items:center;gap:9px;padding:13px 16px 9px}'
    + '.fbw-ava{width:30px;height:30px;border-radius:9px;background:linear-gradient(135deg,#1677ff,#00d4aa);display:flex;align-items:center;justify-content:center;color:#fff;flex-shrink:0}'
    + '.fbw-name{font-size:13.5px;font-weight:700;color:rgba(0,0,0,0.85)}'
    + '.fbw-tag{font-size:10.5px;color:rgba(0,0,0,0.4)}'
    + '.fbw-close{margin-left:auto;border:none;background:none;font-size:14px;color:rgba(0,0,0,0.4);cursor:pointer;padding:4px}'
    + '.fbw-close:hover{color:rgba(0,0,0,0.7)}'
    + '.fbw-tabs{display:flex;gap:18px;padding:0 16px;border-bottom:1px solid #f0f0f0}'
    + '.fbw-tab{padding:7px 2px;font-size:12.5px;color:rgba(0,0,0,0.55);cursor:pointer;border-bottom:2px solid transparent;margin-bottom:-1px}'
    + '.fbw-tab.on{color:#1677ff;font-weight:600;border-bottom-color:#1677ff}'
    + '.fbw-msgs{flex:1;overflow-y:auto;padding:14px;display:flex;flex-direction:column;gap:11px}'
    + '.fbw-msg{display:flex;gap:8px;max-width:94%}'
    + '.fbw-msg .bub{padding:8px 12px;border-radius:9px;font-size:12.5px;line-height:1.7;color:rgba(0,0,0,0.8)}'
    + '.fbw-msg.bot .bub{background:#f5f7fa;border-top-left-radius:3px}'
    + '.fbw-msg.me{align-self:flex-end;flex-direction:row-reverse}'
    + '.fbw-msg.me .bub{background:#e6f4ff;border-top-right-radius:3px}'
    + '.fbw-mini{width:24px;height:24px;border-radius:7px;flex-shrink:0;display:flex;align-items:center;justify-content:center;font-size:11px;color:#fff}'
    + '.fbw-mini.bot{background:linear-gradient(135deg,#1677ff,#00d4aa)}'
    + '.fbw-mini.me{background:#8c8c8c}'
    + '.fbw-ticket{border:1px solid #dbe7ff;background:#f7faff;border-radius:9px;padding:10px 12px;margin-top:7px}'
    + '.fbw-ticket .tt{font-size:12px;font-weight:700;color:rgba(0,0,0,0.78);margin-bottom:6px}'
    + '.fbw-ticket .tr2{display:flex;font-size:12px;line-height:1.9}'
    + '.fbw-ticket .tr2 .k{width:56px;color:rgba(0,0,0,0.45);flex-shrink:0}'
    + '.fbw-ticket .tr2 .v{color:rgba(0,0,0,0.78)}'
    + '.fbw-ticket .ta{margin-top:8px;display:flex;gap:8px}'
    + '.fbw-btn{border:none;border-radius:7px;padding:6px 14px;font-size:12px;font-weight:500;cursor:pointer}'
    + '.fbw-btn.pri{background:#1677ff;color:#fff}.fbw-btn.pri:hover{background:#4096ff}'
    + '.fbw-btn.ghost{background:#fff;color:rgba(0,0,0,0.65);border:1px solid #d9d9d9}'
    + '.fbw-inrow{display:flex;gap:8px;padding:11px 14px;border-top:1px solid #f5f5f5}'
    + '.fbw-in{flex:1;height:34px;border:1px solid #d9d9d9;border-radius:8px;padding:0 12px;font-size:12.5px;outline:none}'
    + '.fbw-in:focus{border-color:#1677ff;box-shadow:0 0 0 2px rgba(22,119,255,0.1)}'
    + '.fbw-send{background:#1677ff;color:#fff;border:none;border-radius:8px;padding:0 16px;font-size:12.5px;cursor:pointer}'
    + '.fbw-send:hover{background:#4096ff}'
    + '.fbw-list{flex:1;overflow-y:auto}'
    + '.fbw-item{padding:11px 16px;border-bottom:1px solid #f5f5f5;cursor:pointer}'
    + '.fbw-item:hover{background:#fafcff}'
    + '.fbw-item .t{font-size:12.5px;color:rgba(0,0,0,0.82);font-weight:500;line-height:1.5}'
    + '.fbw-item .m{display:flex;align-items:center;gap:7px;margin-top:4px;font-size:11px;color:rgba(0,0,0,0.4);flex-wrap:wrap}'
    + '.fbw-st{display:inline-block;font-size:10.5px;padding:0 6px;border-radius:4px;line-height:17px}'
    + '.fbw-st.doing{background:#e6f4ff;color:#1677ff;border:1px solid #91caff}'
    + '.fbw-st.replied{background:#fff7e6;color:#d46b08;border:1px solid #ffd591}'
    + '.fbw-st.done{background:#f6ffed;color:#389e0d;border:1px solid #b7eb8f}'
    + '.fbw-cat{display:inline-block;font-size:10.5px;padding:0 6px;border-radius:4px;line-height:17px;background:#f5f5f5;color:rgba(0,0,0,0.55)}'
    + '.fbw-reply{margin-top:7px;background:#f7f9fc;border-radius:8px;padding:8px 10px;font-size:11.5px;color:rgba(0,0,0,0.6);line-height:1.65;display:none}'
    + '.fbw-item.open .fbw-reply{display:block}';
  document.head.appendChild(st);

  var fab = document.createElement('div');
  fab.className = 'fbw-fab';
  fab.title = '问题反馈';
  fab.onclick = openFeedbackPanel;
  fab.innerHTML = '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M21 11.5a8.38 8.38 0 0 1-.9 3.8 8.5 8.5 0 0 1-7.6 4.7 8.38 8.38 0 0 1-3.8-.9L3 21l1.9-5.7a8.38 8.38 0 0 1-.9-3.8 8.5 8.5 0 0 1 4.7-7.6 8.38 8.38 0 0 1 3.8-.9h.5a8.48 8.48 0 0 1 8 8v.5z"/></svg><span>反</span><span>馈</span>';
  document.body.appendChild(fab);

  var panel = document.createElement('div');
  panel.id = 'fbwPanel';
  panel.className = 'fbw-panel';
  panel.innerHTML = ''
    + '<div class="fbw-head">'
    + '<div class="fbw-ava"><svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M12 8V4H8"/><rect x="4" y="8" width="16" height="12" rx="2"/><path d="M2 14h2M20 14h2M15 13v2M9 13v2"/></svg></div>'
    + '<div><div class="fbw-name">反馈助手</div><div class="fbw-tag">MOI 智能体驱动 · 自动分类并生成工单</div></div>'
    + '<button class="fbw-close" onclick="closeFeedbackPanel()">✕</button>'
    + '</div>'
    + '<div class="fbw-tabs">'
    + '<div class="fbw-tab" id="fbwTab-new" onclick="fbwSwitch(\'new\')">新增反馈</div>'
    + '<div class="fbw-tab" id="fbwTab-history" onclick="fbwSwitch(\'history\')">反馈历史</div>'
    + '</div>'
    + '<div class="fbw-msgs" id="fbwMsgs" style="display:none"></div>'
    + '<div class="fbw-inrow" id="fbwInrow" style="display:none"><input class="fbw-in" id="fbwIn" placeholder="描述您遇到的问题或建议…" onkeydown="if(event.key===\'Enter\')fbwSend()"><button class="fbw-send" onclick="fbwSend()">发送</button></div>'
    + '<div class="fbw-list" id="fbwList" style="display:none"></div>';
  document.body.appendChild(panel);

  fbwAddMsg('bot', '您好，我是 MOI 反馈助手。请直接描述您遇到的问题或建议——文字即可，我会帮您补齐关键信息、自动分类并生成工单。');
}

function openFeedbackPanel() {
  fbwEnsure();
  document.body.classList.add('fbw-open');
  fbwSwitch(fbwState.tab);
}
function closeFeedbackPanel() { document.body.classList.remove('fbw-open'); }
function fbwSwitch(tab) {
  fbwState.tab = tab;
  document.getElementById('fbwTab-new').classList.toggle('on', tab === 'new');
  document.getElementById('fbwTab-history').classList.toggle('on', tab === 'history');
  document.getElementById('fbwMsgs').style.display = tab === 'new' ? 'flex' : 'none';
  document.getElementById('fbwInrow').style.display = tab === 'new' ? 'flex' : 'none';
  document.getElementById('fbwList').style.display = tab === 'history' ? 'block' : 'none';
  if (tab === 'history') fbwRenderList();
}
function fbwRenderList() {
  document.getElementById('fbwList').innerHTML = FBW_LIST.map(function(f) {
    return '<div class="fbw-item" onclick="this.classList.toggle(\'open\')">'
      + '<div class="t">' + f.title + '</div>'
      + '<div class="m"><span class="fbw-cat">' + f.cat + '</span><span class="fbw-st ' + f.st + '">' + f.stLabel + '</span><span>' + f.id + '</span><span>' + f.time + '</span></div>'
      + (f.reply ? '<div class="fbw-reply">' + f.reply + '</div>' : '')
      + '</div>';
  }).join('');
}
function fbwAddMsg(who, html) {
  var box = document.getElementById('fbwMsgs');
  if (!box) return;
  box.insertAdjacentHTML('beforeend',
    '<div class="fbw-msg ' + who + '"><div class="fbw-mini ' + who + '">' + (who === 'bot' ? 'A' : '我') + '</div><div class="bub">' + html + '</div></div>');
  box.scrollTop = box.scrollHeight;
}
function fbwSend() {
  var input = document.getElementById('fbwIn');
  var text = input.value.trim();
  if (!text) return;
  fbwAddMsg('me', text.replace(/&/g, '&amp;').replace(/</g, '&lt;'));
  input.value = '';
  if (fbwState.step === 0) {
    fbwState.step = 1;
    setTimeout(function() {
      fbwAddMsg('bot', '收到。为了更快定位，请补充两点：1）问题发生在哪个模块（如 数据载入 / 工作流 / 知识库 / 智能体）？2）大概什么时间发生的，是否可复现？');
    }, 450);
  } else if (fbwState.step === 1) {
    fbwState.step = 2;
    setTimeout(function() {
      fbwAddMsg('bot', '信息齐了，我根据描述生成了这张工单，请确认：'
        + '<div class="fbw-ticket"><div class="tt">工单草稿</div>'
        + '<div class="tr2"><span class="k">分类</span><span class="v">缺陷</span></div>'
        + '<div class="tr2"><span class="k">模块</span><span class="v">数据 · 数据载入</span></div>'
        + '<div class="tr2"><span class="k">摘要</span><span class="v" id="fbwSummary">载入任务异常（据您的描述自动归纳）</span></div>'
        + '<div class="tr2"><span class="k">优先级</span><span class="v">中（影响使用，存在绕过方式）</span></div>'
        + '<div class="ta"><button class="fbw-btn pri" onclick="fbwSubmit(this)">确认提交</button><button class="fbw-btn ghost" onclick="fbwAddMsg(\'bot\',\'好的，请继续补充，我会更新工单内容。\')">再补充</button></div>'
        + '</div>');
    }, 550);
  } else {
    setTimeout(function() { fbwAddMsg('bot', '已记录并同步到工单。还有其他问题欢迎随时反馈。'); }, 400);
  }
}
function fbwSubmit(btn) {
  btn.disabled = true;
  var id = 'FB-20260828-0' + (FBW_LIST.length + 13);
  FBW_LIST.unshift({ id: id, title: document.getElementById('fbwSummary').textContent, cat: '缺陷', st: 'doing', stLabel: '处理中', time: '2026-08-28', reply: '' });
  fbwAddMsg('bot', '工单 <b>' + id + '</b> 已提交，可在「反馈历史」页签跟踪进展。感谢反馈！');
}
// 页面加载即挂右缘反馈入口——仅前台产品页面;管理后台与官网门户(登录/注册)不挂
if (document.body && location.pathname.indexOf('/admin/') < 0 && location.pathname.indexOf('/website/') < 0) fbwEnsure();
