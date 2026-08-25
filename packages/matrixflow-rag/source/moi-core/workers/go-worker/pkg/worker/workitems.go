package worker

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	moi "github.com/matrixflow/moi-core/go-sdk"
	commonpb "github.com/matrixflow/moi-core/model/common"
	"github.com/matrixflow/moi-core/model/mowl"
	"github.com/matrixflow/moi-core/workers/go-worker/pkg/runtime"
	"github.com/matrixflow/moi-core/workers/go-worker/pkg/workitems"
	"github.com/matrixflow/moi-core/workers/go-worker/pkg/workitems/parser"
	"github.com/matrixflow/moi-core/workers/go-worker/pkg/workitems/parser/clients"
	"go.uber.org/zap"
)

type WorkItemRegistration struct {
	Metadata      *mowl.WorkItemMetadata
	Handler       moi.ExternalWorkItemFunc
	StreamHandler moi.StreamWorkItemFunc
}

type CatalogItem struct {
	Name              string                 `json:"name"`
	Description       string                 `json:"description"`
	Version           string                 `json:"version"`
	Isolation         string                 `json:"isolation_level"`
	Stream            bool                   `json:"stream"`
	InputSchema       interface{}            `json:"input_schema"`
	OutputSchema      interface{}            `json:"output_schema"`
	Metadata          *mowl.WorkItemMetadata `json:"metadata"`
	InputUISchema     interface{}            `json:"input_ui_schema"`
	OutputUISchema    interface{}            `json:"output_ui_schema"`
	I18n              map[string]interface{} `json:"i18n,omitempty"`
	I18NDefaultLocale string                 `json:"i18n_default_locale,omitempty"`
}

type Catalog struct {
	Worker string        `json:"worker"`
	Count  int           `json:"count"`
	Items  []CatalogItem `json:"items"`
}

func WorkItems(factory *runtime.ClientFactory, logger *zap.Logger, cfgs ...Config) []WorkItemRegistration {
	if logger == nil {
		logger = zap.NewNop()
	}
	var cfg Config
	if len(cfgs) > 0 {
		cfg = cfgs[0]
	}
	versionRouter := cfg.Parser.ToVersionRouter()
	parserQueues := workitems.NewParserAPIQueues(cfg.Parser.ToParserAPIQueueConfig())
	documentVisualParseRuntimeFields := documentVisualParseInputUIFields(documentVisualParseRuntimeConfig())
	clientOpts := []clients.Option{
		clients.WithVisionModel(cfg.Parser.ResolveVisionModel()),
		clients.WithCaptionModel(cfg.Parser.ResolveCaptionModel()),
	}
	unifiedExtractItem := &workitems.UnifiedExtractor{Factory: factory, Config: cfg.Extract}
	embeddingItem := &workitems.EmbeddingGenerate{Factory: factory}
	fileMetaGet := &workitems.FileMetadataGet{Factory: factory}
	fileItem := &workitems.FileReadText{Factory: factory}
	readDocs := &workitems.FilesReadDocuments{Factory: factory}
	writeDocs := &workitems.FilesWriteDocuments{Factory: factory}
	catalogSourceRead := &workitems.CatalogSourceRead{Factory: factory}
	catalogSourceReadV2 := &workitems.CatalogSourceReadV2{Factory: factory}
	catalogPDFPrepare := &workitems.CatalogPDFPrepare{Factory: factory}
	catalogPDFMerge := &workitems.CatalogPDFMerge{}
	catalogList := &workitems.CatalogList{Factory: factory}
	catalogFilesManifestPlan := &workitems.CatalogFilesManifestPlan{Factory: factory}
	catalogSinkWrite := &workitems.CatalogSinkWrite{Factory: factory}
	// The legacy v2 entry remains registered for builtin templates that still
	// depend on V2 and saved workflow DSLs, but is no longer offered from the
	// user-created node palette.
	documentParse := &workitems.DocumentParse{Factory: factory, ClientOpts: clientOpts, VersionRouter: versionRouter, ParserQueues: parserQueues, ForceParserVersion: "v2"}
	// The user-visible default node is moi:parse. It shares the parse handler with
	// the legacy alias parse:stage_runtime and pins parser_version=v3.
	documentParseV3 := &workitems.DocumentParse{Factory: factory, ClientOpts: clientOpts, VersionRouter: versionRouter, ParserQueues: parserQueues, ForceParserVersion: "v3"}
	documentVisualParse := &workitems.DocumentVisualParse{Factory: factory, VersionRouter: versionRouter, ParserQueues: parserQueues}
	documentVisualParseCodex := &workitems.DocumentVisualParseCodex{Factory: factory, Config: cfg.DocumentVisualCodex.ToWorkItemConfig()}
	documentVisualIndexText := &workitems.DocumentVisualIndexText{Factory: factory, Logger: logger}
	documentVisualIndexImage := &workitems.DocumentVisualIndexImage{Factory: factory, Logger: logger}
	knowledgeIndexBuild := &workitems.KnowledgeIndexBuild{Factory: factory, Logger: logger}
	dataTransform := &workitems.DataTransform{}
	catalogResolve := &workitems.CatalogResolve{Factory: factory}
	databaseResolve := &workitems.DatabaseResolve{Factory: factory}
	volumeResolve := &workitems.VolumeResolve{Factory: factory}
	ensureVolume := &workitems.VolumeEnsure{Factory: factory}
	addToVolume := &workitems.VolumeFilesAdd{Factory: factory}
	listVolumeFiles := &workitems.VolumeFilesList{Factory: factory}
	moveVolumeFiles := &workitems.VolumeFilesMove{Factory: factory}
	removeVolumeFiles := &workitems.VolumeFilesRemove{Factory: factory}
	assetRegister := &workitems.DataAssetRegister{Factory: factory}
	assetLink := &workitems.DataAssetLink{Factory: factory}
	lineageRegister := &workitems.DataLineageRegister{Factory: factory}
	docMap := &workitems.DataDocMapMetadata{Factory: factory}
	tableUpsertJSON := &workitems.DataTableUpsertJSON{Factory: factory}
	runSQL := &workitems.RunSQL{Factory: factory}
	dashboardRefresh := &workitems.DashboardRefresh{Factory: factory, RunSQL: runSQL}
	dataTableRead := &workitems.DataTableRead{RunSQL: runSQL, Factory: factory}
	emailArchiveETL := &workitems.EmailArchiveETL{Factory: factory}
	apiRequest := &workitems.APIRequest{}
	githubRepoRead := &workitems.GitHubRepoRead{Factory: factory}
	githubRepoWrite := &workitems.GitHubRepoWrite{}
	githubRepoWorkspace := &workitems.GitHubRepoWorkspace{Config: cfg.CodexTool.ToGitHubRepoWorkspaceConfig()}
	productSource := &workitems.ProductSource{}
	codexRun := &workitems.CodexRun{Config: cfg.CodexTool.ToWorkItemConfig(), Factory: factory}
	grafanaRead := &workitems.GrafanaRead{}
	kubernetesRead := &workitems.KubernetesRead{}
	feishuConnectionTest := &workitems.FeishuConnectionTest{}
	feishuMessageSend := &workitems.FeishuMessageSend{}
	slackConnectionTest := &workitems.SlackConnectionTest{}
	slackMessageSend := &workitems.SlackMessageSend{}
	llmOutputGenerate := &workitems.LLMOutputGenerate{Factory: factory}
	workflowTrigger := &workitems.WorkflowTrigger{Factory: factory}
	sqlPipeline := &workitems.SQLPipeline{Factory: factory}
	dataVectorWrite := &workitems.DataVectorWrite{Factory: factory, Logger: logger}
	multiLevelIndex := &workitems.MultiLevelIndex{}
	cdhExportS3 := &workitems.CDHExportS3{Factory: factory}
	s3ToMOImport := &workitems.S3ToMOImport{Factory: factory}
	webCrawler := &workitems.WebCrawler{Factory: factory}

	catalogSourceReadDesc := "从 Catalog 选择文件、文件夹、Volume 或数据资产作为工作流输入，输出 raw sources 和 file_ids，不输出文件正文。通常作为读取用户资料、待解析文件、待导入文件的第一个节点；解析类下游接 sources，文件导入类下游接 file_ids。"
	catalogSourceReadDescEN := "Select files, folders, volumes, or data assets from Catalog as workflow input. It outputs raw sources and file_ids without file content. Use it as the first node for user documents, files to parse, or files to import; parser nodes consume sources, and import nodes consume file_ids."
	catalogSourceReadV2Desc := "从 Catalog 选择文件、文件夹、Volume 或数据资产作为工作流输入。新工作流应使用此版本；v2 要求提供非空资源选择，输出稳定一致的 files[]（包含 file_id、name 和 file_name）以及 sources 和 file_ids，不输出文件正文。" // i18n-allow: zh locale workitem summary paired with en description.
	catalogSourceReadV2DescEN := "Select files, folders, volumes, or data assets from Catalog as workflow input. Use this version for new workflows. V2 requires a non-empty resource selector and returns stable canonical files[] entries with file_id, name, and file_name, together with sources and file_ids, without file content."
	catalogPDFPrepareDesc := "Catalog 文件解析系统工作流的内部 PDF 准备节点：提取原生文本、恢复线框表格，并只标记仍需视觉解析的页面。" // i18n-allow: zh locale workitem summary paired with en description.
	catalogPDFPrepareDescEN := "Internal Catalog parsing node that extracts native PDF text, reconstructs ruled tables, and selects only pages that still require visual parsing."
	catalogPDFMergeDesc := "Catalog 文件解析系统工作流的内部 PDF 合并节点：以视觉解析结果严格替换选中页面，并保留其余 PDFium 页面。" // i18n-allow: zh locale workitem summary paired with en description.
	catalogPDFMergeDescEN := "Internal Catalog parsing node that strictly replaces selected pages with visual-parser output while preserving all other PDFium pages."
	catalogListDesc := "分页列出当前工作空间的 Catalog 条目。输入 page_size、可选 page_token 和隐藏 retry_count，输出 items、count、total、next_page_token 和 retry_count；显式循环翻页时把 page_token 绑定为上一轮 next_page_token，循环条件使用 .next_page_token != \"\"。" // i18n-allow: zh locale workitem summary
	catalogListDescEN := "List one page of Catalog entries in the current workspace. Inputs are page_size plus optional page_token and hidden retry_count; outputs include items, count, total, next_page_token, and retry_count. For explicit pagination loops, bind page_token from the previous next_page_token and use condition .next_page_token != \"\"."
	catalogFilesManifestPlanDesc := "把 Catalog 文件来源枚举成稳定的 JSONL 分片清单，只写入 file_id、路径和全局序号，不下载文件正文。用于大批量文件的并行工作流：先生成固定快照，再由下游并行分支各自消费一个清单分片。" // i18n-allow: zh locale workitem summary
	catalogFilesManifestPlanDescEN := "Enumerate Catalog file sources into stable JSONL shard manifests with file_id, path, and global ordinal only. It does not download file content. Use it for large parallel workflows: create a fixed snapshot first, then let downstream parallel branches consume one manifest shard each."
	catalogSinkWriteDesc := "把上游 documents、rows/columns、text、json 或 file_ids 保存为 Catalog 文件。通常作为最终输出节点；rows+columns 会写成带表头的 CSV，输出 file_id/file_ids 可继续交给文件导入或血缘节点。"
	catalogSinkWriteDescEN := "Save upstream documents, rows/columns, text, JSON, or file_ids as Catalog files. It is usually used as the final output node; rows+columns are written as a CSV with a header, and the output file_id/file_ids can be passed to file import or lineage nodes."
	documentParseDesc := "旧版 v2 引擎，供仍依赖 V2 的系统模板和已创建工作流保留；新建通用工作流请用文档解析（moi:parse）。把 Catalog source 或 file_id/file_ids 解析成标准 documents，用于 PDF、DOCX、PPTX、图片、音频、视频等非结构化文件进入切分、抽取、索引或自定义文档处理节点之前。" // i18n-allow: zh WorkItem i18n pack source paired with en description.
	documentParseDescEN := "Legacy v2 engine retained for builtin templates that still depend on V2 and existing workflows; use Parse Document (moi:parse) for new general-purpose workflows. It parses Catalog sources or file_id/file_ids into standard documents before splitting, extraction, indexing, or custom document processing for unstructured files such as PDF, DOCX, PPTX, images, audio, and video."
	parseStageRuntimeDesc := "用 parse-v3 在进程内引擎把 Catalog source 或 file_id/file_ids 解析成标准 documents（parser_version 固定 v3）。覆盖文档（PDF/DOCX/PPTX/图片）、邮件等文件类型，按格式路由到对应解析路径；V3 自带选项 schema（表格增强 / 跨页合并 / 阅读顺序等），是新建通用工作流的默认文档解析节点；旧版 moi:document.parse 仍供依赖 V2 的系统模板和已创建工作流使用。" // i18n-allow: zh WorkItem i18n pack source paired with en description.
	parseStageRuntimeDescEN := "Use parse-v3 in the in-process engine to parse Catalog sources or file_id/file_ids into standard documents (parser_version is fixed to v3). It covers documents (PDF/DOCX/PPTX/images) and email, routing each to its matching parse path. V3 carries its own option schema for table enrichment, cross-page merge, reading order, and related settings. It is the default document parser for new general-purpose workflows; the legacy moi:document.parse node remains for builtin templates that still depend on V2 and existing workflows."
	audioParseDesc := "把音频文件转写成带时间戳 metadata 的标准 documents。用于会议录音、访谈、答辩记录等音频资料进入切分、抽取、索引或保存节点之前。"
	audioParseDescEN := "Transcribe audio files into standard documents with timestamp metadata. Use it before splitting, extraction, indexing, or saving for meeting recordings, interviews, defenses, and similar audio sources."
	videoParseDesc := "抽取视频音轨并转写成带时间戳 metadata 的标准 documents。用于会议录像、培训视频、答辩录像等视频资料进入切分、抽取、索引或保存节点之前。"
	videoParseDescEN := "Extract the audio track from video and transcribe it into standard documents with timestamp metadata. Use it before splitting, extraction, indexing, or saving for meeting videos, training videos, defenses, and similar video sources."
	imageParseDesc := "把图片文件解析成标准 documents。可做 OCR、图片描述或两者结合，用于图片资料进入切分、抽取、索引或保存节点之前。"
	imageParseDescEN := "Parse image files into standard documents. It can run OCR, image captioning, or both before image materials are split, extracted, indexed, or saved."
	cleanTextDesc := "清理文本中的重复空白和换行，输出规整后的 text。用于上游文本进入模型推理、抽取、保存或自定义处理之前。"
	cleanTextDescEN := "Clean repeated whitespace and line breaks from text and output normalized text. Use it before model inference, extraction, saving, or custom processing."
	knowledgeIndexBuildDesc := "把 documents 生成 embedding 并写入知识库/向量索引，用于后续检索和问答。通常接在 document.parse 或 split.documents.length 之后作为最终索引输出节点，输出 written 表示写入数量。"
	knowledgeIndexBuildDescEN := "Generate embeddings from documents and write them into a knowledge base or vector index for later retrieval and Q&A. It usually follows document.parse or split.documents.length as the final indexing output node, and written reports the number of rows written."
	splitLevelDesc := "按 Markdown/标题层级切分 documents，保留来源 metadata。用于已经有标题结构的文档在进入抽取、索引、保存或自定义文档处理前拆成更小片段。"
	splitLevelDescEN := "Split documents by Markdown or heading hierarchy while preserving source metadata. Use it to break documents that already have heading structure into smaller fragments before extraction, indexing, saving, or custom document processing."
	splitDocumentsLengthDesc := "按长度和 overlap 切分 documents，保留来源 metadata 与语义原子性。表格、代码以及包含 fenced code 的 Markdown 容器不会被切断，因此可能超过 chunk_size。" // i18n-allow: zh WorkItem summary paired with the en locale value.
	splitDocumentsLengthDescEN := "Split documents by length and overlap while preserving source metadata and semantic atomicity. Tables, code, and Markdown containers with fenced code remain whole and may exceed chunk_size."
	splitDocumentsLengthInputFields := splitDocumentsLengthInputUIFields()
	splitDocumentsLengthI18N := withInputUIFieldI18NDescriptions(
		i18nPacks("按长度切分文档", "Split Documents by Length", "文档处理", "Document Processing", splitDocumentsLengthDesc, splitDocumentsLengthDescEN, splitDocumentsLengthInputFields, nil), // i18n-allow: zh WorkItem display name and group paired with en locale values.
		"chunk_size",
		splitDocumentsLengthChunkSizeDescEN,
		splitDocumentsLengthChunkSizeDescZH,
	)
	structuredExtractDesc := "用 LLM 从 text、documents 或 files 中抽取结构化 JSON。适合需要语义理解的字段抽取；输出 result/results 可写入数据表、保存为 JSON，或交给自定义算子继续处理。"
	structuredExtractDescEN := "Use an LLM to extract structured JSON from text, documents, or files. It is suitable for field extraction that requires semantic understanding; result/results can be written to data tables, saved as JSON, or passed to custom operators."
	lineageRegisterDesc := "登记数据资产和派生关系，串起 source file、parsed docset、vector index 或其他输出文件的血缘。通常放在解析/写文件/建索引之后，用于治理和可追踪性。"
	lineageRegisterDescEN := "Register data assets and derivation relationships to connect lineage across source files, parsed docsets, vector indexes, or other output files. It is usually placed after parsing, file writing, or indexing for governance and traceability."
	tableUpsertJSONDesc := "把一个结构化 JSON 对象或 values map 写入工作空间数据表。通常接 LLM 抽取或自定义算子的 JSON 输出；写 rows 表格数组时优先用 Catalog CSV 保存或 SQL/导入节点。"
	tableUpsertJSONDescEN := "Write one structured JSON object or values map into a workspace data table. It usually follows JSON output from LLM extraction or custom operators; for rows-style table arrays, prefer Catalog CSV saving or SQL/import nodes."
	emailArchiveETLDesc := "把 RFC822/MIME 邮件归档文件解析成 5 组关系表：邮件路径、邮件头正文、To 收件人、X-To 收件人和被引用原始邮件头。生产/并行模式消费 Catalog JSONL 清单分片或文件引用，按分片写出 Catalog CSV 文件并返回 CSV file_id，适合 50 万级 maildir 归档导入 MatrixOne。" // i18n-allow: zh locale workitem summary
	emailArchiveETLDescEN := "Parse RFC822/MIME email archive files into five relational tables: email path provenance, email headers/body, To recipients, X-To recipients, and quoted original-message headers. Production/parallel mode consumes Catalog JSONL manifest shards or file references, writes shard-local Catalog CSV files, and returns CSV file IDs for importing large maildir archives into MatrixOne."
	runSQLDesc := "在工作空间 MatrixOne 数据库执行 SQL。SELECT 输出 rows，可保存为 Catalog CSV 或交给自定义算子处理；DDL/DML 输出 affected_rows，用于建表、清洗、聚合和写入。"
	dataTableReadDesc := "从工作空间 MatrixOne 表读取数据，作为工作流的数据表输入节点。适合用户明确选择已有 MOI 表并把 rows 交给后续 SQL、LLM、保存或导出节点。"
	dataTableReadDescEN := "Read data from a workspace MatrixOne table as a workflow table input node. Use it when the user explicitly selects an existing MOI table and passes rows to downstream SQL, LLM, saving, or export nodes."
	apiRequestDesc := "向外部 HTTP API、Webhook 或服务端点发起一次请求。输入包含 method、url、headers、body 和 timeout_seconds；输出 status_code、headers 和 body。"
	apiRequestDescEN := "Send one request to an external HTTP API, webhook, or service endpoint. Inputs include method, url, headers, body, and timeout_seconds; outputs include status_code, headers, and body."
	githubRepoReadDesc := "通过平台绑定的 GitHub token 读取仓库标签目录、Issue 丰富上下文与图片证据、Pull Request 和工作流运行信息。" // i18n-allow: zh locale workitem summary paired with en description.
	githubRepoReadDescEN := "Read GitHub label catalogs, repositories, rich Issue context and image evidence, pull requests, and workflow runs through a platform-bound GitHub token."
	githubRepoWriteDesc := "通过平台绑定的 GitHub token 执行 Issue 和 Pull Request 写操作，包括评论、标签、负责人与状态变更、原生审查和受控合并。" // i18n-allow: zh locale workitem summary paired with en description.
	githubRepoWriteDescEN := "Mutate GitHub Issues and Pull Requests through a platform-bound GitHub token, including comments, labels, assignees, state, native reviews, and runtime-controlled merges."
	githubRepoWorkspaceDesc := "确保每个 GitHub 仓库只有一份共享本地 clone，并返回可供 Codex 使用的 workspace_ref。" // i18n-allow: zh locale workitem summary paired with en description.
	githubRepoWorkspaceDescEN := "Ensure one shared local clone per GitHub repository and return a workspace_ref for Codex."
	productSourceDesc := "解析 go-worker 镜像内置的 MOI 产品源码树，返回与发布版本对齐、可供 Codex 使用的 workspace_ref；不访问 GitHub。" // i18n-allow: zh locale workitem summary paired with en description.
	productSourceDescEN := "Resolve the image-baked MOI product source tree and return a version-aligned workspace_ref for Codex without contacting GitHub."
	codexRunDesc := "通过平台绑定的 Codex 通道，运行嵌套 Codex 智能体分析只读源码工作区、图片证据或二者组合；已启动但未产出可接受最终结果的调用会在同一会话中自动续接一次。" // i18n-allow: zh locale workitem summary paired with en description.
	codexRunDescEN := "Run one nested Codex agent through a platform-bound channel to analyze a read-only source workspace, image evidence, or both. A started invocation that ends before an accepted final result is resumed once in the same Codex session."
	grafanaReadDesc := "通过平台绑定的 Grafana service account token 读取健康状态、数据源和 Prometheus/Loki 查询结果。" // i18n-allow: zh locale workitem summary paired with en description.
	grafanaReadDescEN := "Read Grafana health, datasources, and Prometheus/Loki query results through a platform-bound Grafana service account token."
	kubernetesReadDesc := "通过平台绑定的 Kubernetes 凭据读取 namespace、node、pod、deployment、service 和 event。" // i18n-allow: zh locale workitem summary paired with en description.
	kubernetesReadDescEN := "Read Kubernetes namespaces, nodes, pods, deployments, services, and events through a platform-bound Kubernetes credential."
	feishuMessageSendDesc := "通过平台绑定的飞书应用凭证发送文本、富文本或交互卡片消息。" // i18n-allow: zh locale workitem summary paired with en description.
	feishuMessageSendDescEN := "Send text, post, or interactive messages through a platform-bound Feishu app credential."
	feishuConnectionTestDesc := "通过获取 tenant access token 验证飞书 App ID、App Secret 和服务连通性，不发送消息或修改飞书资源。" // i18n-allow: zh locale workitem summary paired with en description.
	feishuConnectionTestDescEN := "Verify Feishu App ID, App Secret, and service reachability by obtaining a tenant access token without sending messages or changing Feishu resources."
	slackConnectionTestDesc := "通过调用 Slack auth.test 验证 Bot Token 和服务连通性，不发送消息或修改 Slack 资源。" // i18n-allow: zh locale workitem summary paired with en description.
	slackConnectionTestDescEN := "Verify a Slack bot token and service reachability with auth.test without sending messages or changing Slack resources."
	slackMessageSendDesc := "通过平台绑定的 Slack Bot Token 发送文本、Block Kit 或附件消息。" // i18n-allow: zh locale workitem summary paired with en description.
	slackMessageSendDescEN := "Send text, Block Kit, or attachment messages through a platform-bound Slack bot token."
	llmOutputGenerateDesc := "调用工作空间 LLM 模型生成自然语言输出。适合对上游文本、问题或明确 prompt 做模型推理，输出 text/result 给保存、抽取或后续处理节点。"
	llmOutputGenerateDescEN := "Call a workspace LLM model to generate natural-language output. Use it for model inference over upstream text, questions, or explicit prompts, and pass text/result to saving, extraction, or downstream processing nodes."
	workflowTriggerDesc := "触发另一个已发布工作流版本，用于产品级工作流编排和链式执行。输入必须显式提供 workflow_version_id 与 task_name，可传 data/vars 给被触发工作流。"
	workflowTriggerDescEN := "Trigger another published workflow version for product-level orchestration and chained execution. Inputs must explicitly provide workflow_version_id and task_name, and may pass data/vars to the triggered workflow."
	sqlProcessDesc := "执行一条工作空间 MatrixOne SQL。必填输入为 sql；可选 table_ref 接收上游「数据读取·MOI·表」的 Catalog 表引用，形成可追溯的 Catalog-to-SQL 数据边。绑定 table_ref 后可在 SQL 中使用 ${source_table} 占位符（运行时替换为带库名的引用）。SELECT/SHOW/WITH 查询输出 rows，DML/DDL 输出 affected_rows、sql、elapsed_ms、truncated。" // i18n-allow: zh locale workitem summary paired with en description.
	sqlProcessDescEN := "Execute one workspace MatrixOne SQL statement. Required input is sql; optional table_ref accepts a Catalog table reference from upstream moi:data.table.read to form a traceable Catalog-to-SQL data edge. When table_ref is bound, SQL may use the ${source_table} placeholder (replaced with the quoted database.table FQN at runtime). SELECT/SHOW/WITH queries output rows; DML/DDL output affected_rows, sql, elapsed_ms, and truncated."
	filesWriteDocumentsDesc := "把 documents 数组写成工作空间 JSONL 文件，输出 file_id/file_ids。用于需要把内存中的解析/切分/自定义文档结果落成文件，再交给保存、导入或血缘节点的场景。"
	filesWriteDocumentsDescEN := "Write a documents array into workspace JSONL files and output file_id/file_ids. Use it when parsed, split, or custom document results in memory need to be persisted before saving, importing, or lineage registration."
	s3ToMOImportDesc := "把 Catalog/S3 中的 CSV、Parquet、ORC 等数据文件导入 MatrixOne 表。Excel 导入时，MatrixOne 数值目标列读取单元格原始数值，其他目标列保留 Excel 格式化显示值，精度和 scale 由目标表类型决定。通常接收上游 file_ids；如果导入 catalog.sink.write 生成的 rows+columns CSV，应设置 start_row=1 跳过表头。" // i18n-allow: zh locale WorkItem metadata paired with en description.
	s3ToMOImportDescEN := "Import CSV, Parquet, ORC, and similar data files from Catalog/S3 into a MatrixOne table. For Excel imports, numeric MatrixOne target columns use raw cell values while other target columns retain formatted cell values; the destination table type controls precision and scale. It usually receives upstream file_ids; when importing a rows+columns CSV generated by catalog.sink.write, set start_row=1 to skip the header."
	embeddingGenerateDesc := "为 documents 生成 embedding，并把向量写回每条文档记录。通常用于自定义向量或索引处理；最终知识库写入优先使用“构建知识库索引”。"
	embeddingGenerateDescEN := "Generate embeddings for documents and write vectors back to each document record. Use it for custom vector or index processing; prefer Build Knowledge Index for final knowledge-base writes."
	s3ToMOImportInputFields := s3ToMOImportInputUIFields()
	s3ToMOImportOutputFields := s3ToMOImportOutputUIFields()
	documentVisualParseDesc := "把 CAD 导出的 PDF/图片图纸解析成 visual manifest，包含页面/对象图片反链和上下文。"
	documentVisualParseDescEN := "Parse CAD-exported PDF/image drawings into a visual manifest with page/object image backlinks and context."
	documentVisualParseAgentDesc := "使用内部 Agent Runtime 将 CAD 导出的 PDF 图纸解析成 Markdown、裁剪图片和 document visual manifest。"
	documentVisualParseAgentDescEN := "Use the internal Agent Runtime to parse CAD-exported PDF drawings into Markdown, cropped images, and a document visual manifest."
	documentVisualIndexTextDesc := "从 document visual manifest 构建文本检索行，并把真实文本 embedding 写入 MatrixOne 向量索引。"
	documentVisualIndexTextDescEN := "Build text retrieval rows from a document visual manifest and write real text embeddings into MatrixOne vector index."
	documentVisualIndexImageDesc := "从 document visual manifest 构建图片检索行，并把真实图片 embedding 写入绑定模型的图片向量索引。"
	documentVisualIndexImageDescEN := "Build image retrieval rows from a document visual manifest and write real image embeddings into a model-bound image vector index."

	return []WorkItemRegistration{
		{
			Metadata: workItemMetadata(
				"Reauthorize and refresh one Data Dashboard chart.",
				false,
				schemaDashboardRefreshInput(),
				schemaRunSQLOutput(),
				&mowl.WorkItemMetadata{NodeId: "moi:data.dashboard.refresh", DisplayName: "moi:data.dashboard.refresh", Category: "moi", Visibility: "internal", Summary: "Reauthorize and refresh one Data Dashboard chart.", SideEffectClass: "read_only", Idempotence: "idempotent", Tags: []string{"internal", "dashboard", "sql"}},
				workItemUISchema("input", "moi:data.dashboard.refresh", "moi:data.dashboard.refresh", "Reauthorize and refresh one Data Dashboard chart.", requiredInputUIFields([]string{"dashboard_id", "chart_id"}, nil)),
				workItemUISchema("output", "moi:data.dashboard.refresh", "moi:data.dashboard.refresh", "Reauthorize and refresh one Data Dashboard chart.", nil),
				nil,
			),
			Handler: dashboardRefresh.Handle,
		},
		{
			Metadata: workItemMetadata(
				feishuConnectionTestDesc,
				false,
				schemaFeishuConnectionTestInput(),
				schemaChannelConnectionTestOutput(),
				&mowl.WorkItemMetadata{
					NodeId:          "moi:feishu.connection.test",
					DisplayName:     "moi:feishu.connection.test",
					Category:        "moi",
					Visibility:      "internal",
					Summary:         feishuConnectionTestDesc,
					SideEffectClass: "write",
					Idempotence:     "non_idempotent",
					Tags:            []string{"internal", "channel", "feishu", "connection-test"},
				},
				workItemUISchema("input", "moi:feishu.connection.test", "moi:feishu.connection.test", feishuConnectionTestDesc, nil),
				workItemUISchema("output", "moi:feishu.connection.test", "moi:feishu.connection.test", feishuConnectionTestDesc, nil),
				i18nPacks("moi:feishu.connection.test", "moi:feishu.connection.test", "内部", "Internal", feishuConnectionTestDesc, feishuConnectionTestDescEN, nil, nil), // i18n-allow: zh locale workitem display group.
			),
			Handler: feishuConnectionTest.Handle,
		},
		{
			Metadata: workItemMetadata(
				slackConnectionTestDesc,
				false,
				schemaSlackConnectionTestInput(),
				schemaChannelConnectionTestOutput(),
				&mowl.WorkItemMetadata{
					NodeId:          "moi:slack.connection.test",
					DisplayName:     "moi:slack.connection.test",
					Category:        "moi",
					Visibility:      "internal",
					Summary:         slackConnectionTestDesc,
					SideEffectClass: "read",
					Idempotence:     "idempotent",
					Tags:            []string{"internal", "channel", "slack", "connection-test"},
				},
				workItemUISchema("input", "moi:slack.connection.test", "moi:slack.connection.test", slackConnectionTestDesc, nil),
				workItemUISchema("output", "moi:slack.connection.test", "moi:slack.connection.test", slackConnectionTestDesc, nil),
				i18nPacks("moi:slack.connection.test", "moi:slack.connection.test", "内部", "Internal", slackConnectionTestDesc, slackConnectionTestDescEN, nil, nil), // i18n-allow: zh locale workitem display group.
			),
			Handler: slackConnectionTest.Handle,
		},
		{
			Metadata: workItemMetadata(
				dataTableReadDesc,
				false,
				schemaDataTableReadInput(),
				schemaDataTableReadOutput(),
				&mowl.WorkItemMetadata{
					NodeId:          "moi:data.table.read",
					DisplayName:     "数据读取 · MOI · 表",
					Category:        "moi",
					Visibility:      "user",
					DisplayGroup:    "数据输入",
					DisplayOrder:    90,
					NodeRole:        "source",
					Tags:            []string{"table", "database", "source", "rows"},
					Summary:         dataTableReadDesc,
					SideEffectClass: "read_only",
					Idempotence:     "idempotent",
				},
				workItemUISchema("input", "moi:data.table.read", "数据读取 · MOI · 表", dataTableReadDesc, dataTableReadInputUIFields()),
				workItemUISchema("output", "moi:data.table.read", "数据读取 · MOI · 表", dataTableReadDesc, nil),
				i18nPacks("数据读取 · MOI · 表", "Data Read · MOI · Table", "数据输入", "Data Input", dataTableReadDesc, dataTableReadDescEN, dataTableReadInputUIFields(), nil),
			),
			Handler: dataTableRead.Handle,
		},
		{
			Metadata: workItemMetadata(
				catalogSourceReadDesc,
				false,
				schemaCatalogSourceReadInput(),
				schemaCatalogSourceReadOutput(),
				&mowl.WorkItemMetadata{
					NodeId:          "moi:catalog.source.read",
					DisplayName:     "Catalog 数据源",
					Category:        "moi",
					Visibility:      "user",
					DisplayGroup:    "数据输入",
					DisplayOrder:    100,
					NodeRole:        "source",
					Tags:            []string{"catalog", "source", "input"},
					Summary:         catalogSourceReadDesc,
					SideEffectClass: "read_only",
					Idempotence:     "idempotent",
				},
				workItemUISchema("input", "moi:catalog.source.read", "Catalog 数据源", catalogSourceReadDesc, []*mowl.WorkItemUIField{
					catalogPickerUIField("source_ref", "Data source", moi.FormFieldResourceTypeVolume, true),
				}),

				workItemUISchema("output", "moi:catalog.source.read", "Catalog 数据源", catalogSourceReadDesc, nil),
				i18nPacks("Catalog 数据源", "Catalog Source", "数据输入", "Data Input", catalogSourceReadDesc, catalogSourceReadDescEN, []*mowl.WorkItemUIField{
					catalogPickerUIField("source_ref", "Data source", moi.FormFieldResourceTypeVolume, true),
				}, nil),
			),
			Handler: catalogSourceRead.Handle,
		},
		{
			Metadata: workItemMetadataWithVersion(
				"v2",
				catalogSourceReadV2Desc,
				false,
				schemaCatalogSourceReadV2Input(),
				schemaCatalogSourceReadV2Output(),
				&mowl.WorkItemMetadata{
					NodeId:          "moi:catalog.source.read.v2",
					DisplayName:     "Catalog 数据源", // i18n-allow: zh locale workitem display name
					Category:        "moi",
					Visibility:      "user",
					DisplayGroup:    "数据输入", // i18n-allow: zh locale workitem display group
					DisplayOrder:    99,
					NodeRole:        "source",
					Tags:            []string{"catalog", "source", "input", "v2", "canonical-files"},
					Summary:         catalogSourceReadV2Desc,
					SideEffectClass: "read_only",
					Idempotence:     "idempotent",
				},
				workItemUISchema("input", "moi:catalog.source.read.v2", "Catalog 数据源", catalogSourceReadV2Desc, catalogSourceReadV2InputUIFields()),                           // i18n-allow: zh locale workitem UI schema
				workItemUISchema("output", "moi:catalog.source.read.v2", "Catalog 数据源", catalogSourceReadV2Desc, nil),                                                         // i18n-allow: zh locale workitem UI schema
				i18nPacks("Catalog 数据源", "Catalog Source", "数据输入", "Data Input", catalogSourceReadV2Desc, catalogSourceReadV2DescEN, catalogSourceReadV2InputUIFields(), nil), // i18n-allow: zh locale workitem i18n pack
			),
			Handler: catalogSourceReadV2.Handle,
		},
		{
			Metadata: workItemMetadata(
				catalogPDFPrepareDesc,
				false,
				schemaCatalogPDFPrepareInput(),
				schemaCatalogPDFPrepareOutput(),
				&mowl.WorkItemMetadata{
					NodeId:          "moi:catalog.file.pdf.prepare",
					DisplayName:     "moi:catalog.file.pdf.prepare",
					Category:        "moi",
					Visibility:      "internal",
					Summary:         catalogPDFPrepareDesc,
					SideEffectClass: "read_only",
					Idempotence:     "idempotent",
					Tags:            []string{"internal", "catalog", "pdf", "parse", "table"},
				},
				workItemUISchema("input", "moi:catalog.file.pdf.prepare", "moi:catalog.file.pdf.prepare", catalogPDFPrepareDesc, nil),
				workItemUISchema("output", "moi:catalog.file.pdf.prepare", "moi:catalog.file.pdf.prepare", catalogPDFPrepareDesc, nil),
				i18nPacks("moi:catalog.file.pdf.prepare", "moi:catalog.file.pdf.prepare", "内部", "Internal", catalogPDFPrepareDesc, catalogPDFPrepareDescEN, nil, nil), // i18n-allow: zh locale workitem display group.
			),
			Handler: catalogPDFPrepare.Handle,
		},
		{
			Metadata: workItemMetadata(
				catalogPDFMergeDesc,
				false,
				schemaCatalogPDFMergeInput(),
				schemaDocumentsOutput(),
				&mowl.WorkItemMetadata{
					NodeId:          "moi:catalog.file.pdf.merge",
					DisplayName:     "moi:catalog.file.pdf.merge",
					Category:        "moi",
					Visibility:      "internal",
					Summary:         catalogPDFMergeDesc,
					SideEffectClass: "read_only",
					Idempotence:     "idempotent",
					Tags:            []string{"internal", "catalog", "pdf", "parse", "merge"},
				},
				workItemUISchema("input", "moi:catalog.file.pdf.merge", "moi:catalog.file.pdf.merge", catalogPDFMergeDesc, nil),
				workItemUISchema("output", "moi:catalog.file.pdf.merge", "moi:catalog.file.pdf.merge", catalogPDFMergeDesc, nil),
				i18nPacks("moi:catalog.file.pdf.merge", "moi:catalog.file.pdf.merge", "内部", "Internal", catalogPDFMergeDesc, catalogPDFMergeDescEN, nil, nil), // i18n-allow: zh locale workitem display group.
			),
			Handler: catalogPDFMerge.Handle,
		},
		{
			Metadata: workItemMetadata(
				catalogListDesc,
				false,
				schemaCatalogListInput(),
				schemaCatalogListOutput(),
				&mowl.WorkItemMetadata{
					NodeId:          "moi:catalog.list",
					DisplayName:     "Catalog 分页列表", // i18n-allow: zh locale workitem display name
					Category:        "moi",
					Visibility:      "user",
					DisplayGroup:    "数据输入", // i18n-allow: zh locale workitem display group
					DisplayOrder:    105,
					NodeRole:        "source",
					Tags:            []string{"catalog", "list", "page", "pagination", "page_token", "next_page_token", "loop"},
					Summary:         catalogListDesc,
					SideEffectClass: "read_only",
					Idempotence:     "idempotent",
				},
				workItemUISchema("input", "moi:catalog.list", "Catalog 分页列表", catalogListDesc, catalogListInputUIFields()),                                                        // i18n-allow: zh locale workitem UI schema
				workItemUISchema("output", "moi:catalog.list", "Catalog 分页列表", catalogListDesc, catalogListOutputUIFields()),                                                      // i18n-allow: zh locale workitem UI schema
				i18nPacks("Catalog 分页列表", "Catalog Page List", "数据输入", "Data Input", catalogListDesc, catalogListDescEN, catalogListInputUIFields(), catalogListOutputUIFields()), // i18n-allow: zh locale workitem i18n pack
			),
			Handler: catalogList.Handle,
		},
		{
			Metadata: workItemMetadata(
				catalogFilesManifestPlanDesc,
				false,
				schemaCatalogFilesManifestPlanInput(),
				schemaCatalogFilesManifestPlanOutput(),
				&mowl.WorkItemMetadata{
					NodeId:          "moi:catalog.files.manifest",
					DisplayName:     "Catalog 文件分片清单", // i18n-allow: zh locale workitem display name
					Category:        "moi",
					Visibility:      "user",
					DisplayGroup:    "数据输入", // i18n-allow: zh locale workitem display group
					DisplayOrder:    110,
					NodeRole:        "source",
					Tags:            []string{"catalog", "manifest", "shard", "files"},
					Summary:         catalogFilesManifestPlanDesc,
					SideEffectClass: "writes_state",
					Idempotence:     "non_idempotent_or_requires_key",
				},
				workItemUISchema("input", "moi:catalog.files.manifest", "Catalog 文件分片清单", catalogFilesManifestPlanDesc, catalogFilesManifestPlanInputUIFields()),                                        // i18n-allow: zh locale workitem UI schema
				workItemUISchema("output", "moi:catalog.files.manifest", "Catalog 文件分片清单", catalogFilesManifestPlanDesc, nil),                                                                           // i18n-allow: zh locale workitem UI schema
				i18nPacks("Catalog 文件分片清单", "Catalog Files Manifest", "数据输入", "Data Input", catalogFilesManifestPlanDesc, catalogFilesManifestPlanDescEN, catalogFilesManifestPlanInputUIFields(), nil), // i18n-allow: zh locale workitem i18n pack
			),
			Handler: catalogFilesManifestPlan.Handle,
		},
		{
			Metadata: workItemMetadata(
				catalogSinkWriteDesc,
				false,
				schemaCatalogSinkWriteInput(),
				schemaCatalogSinkWriteOutput(),
				&mowl.WorkItemMetadata{
					NodeId:          "moi:catalog.sink.write",
					DisplayName:     "保存到 Catalog",
					Category:        "moi",
					Visibility:      "user",
					DisplayGroup:    "输出",
					DisplayOrder:    810,
					NodeRole:        "final_sink",
					FinalOutput:     true,
					Tags:            []string{"catalog", "sink", "output", "csv", "export", "file"},
					Summary:         catalogSinkWriteDesc,
					SideEffectClass: "writes_state",
					Idempotence:     "non_idempotent_or_requires_key",
				},
				workItemUISchema("input", "moi:catalog.sink.write", "保存到 Catalog", catalogSinkWriteDesc, catalogSinkWriteInputUIFields()),

				workItemUISchema("output", "moi:catalog.sink.write", "保存到 Catalog", catalogSinkWriteDesc, nil),
				i18nPacks("保存到 Catalog", "Save to Catalog", "输出", "Output", catalogSinkWriteDesc, catalogSinkWriteDescEN, catalogSinkWriteInputUIFields(), nil),
			),
			Handler: catalogSinkWrite.Handle,
		},
		{
			Metadata: workItemMetadata(
				documentParseDesc,
				false,
				schemaDocumentParseInput(),
				schemaDocumentsOutput(),
				&mowl.WorkItemMetadata{
					NodeId:                "moi:document.parse",
					DisplayName:           "文档解析（旧版）", // i18n-allow: zh WorkItem i18n pack source paired with en display name.
					Category:              "moi",
					Visibility:            "internal",
					DisplayGroup:          "文档处理",
					DisplayOrder:          300,
					NodeRole:              "transform",
					Tags:                  []string{"document", "parse", "pdf", "docx", "pptx", "html", "image", "audio", "video", "ocr", "legacy"},
					Summary:               documentParseDesc,
					SideEffectClass:       "read_only",
					Idempotence:           "idempotent",
					RuntimeConfigContract: documentParseRuntimeConfig(),
				},
				workItemUISchema("input", "moi:document.parse", "文档解析（旧版）", documentParseDesc, documentParseInputUIFields(documentParseRuntimeConfig(), "moi:document.parse")),                                               // i18n-allow: zh WorkItem i18n pack source paired with en input UI title.
				workItemUISchema("output", "moi:document.parse", "文档解析（旧版）", documentParseDesc, nil),                                                                                                                         // i18n-allow: zh WorkItem i18n pack source paired with en output UI title.
				i18nPacks("文档解析（旧版）", "Parse Document (Legacy)", "文档处理", "Document Processing", documentParseDesc, documentParseDescEN, documentParseInputUIFields(documentParseRuntimeConfig(), "moi:document.parse"), nil), // i18n-allow: zh WorkItem i18n pack source paired with en display name and group.
			),
			Handler: documentParse.Handle,
		},
		{
			Metadata: workItemMetadata(
				parseStageRuntimeDesc,
				false,
				schemaParseStageRuntimeInput(),
				schemaParseStageRuntimeOutput(),
				&mowl.WorkItemMetadata{
					NodeId:                "moi:parse",
					DisplayName:           "文档解析", // i18n-allow: zh WorkItem i18n pack source paired with en display name.
					Category:              "moi",
					Visibility:            "user",
					DisplayGroup:          "文档处理",
					DisplayOrder:          305,
					NodeRole:              "transform",
					Tags:                  []string{"document", "parse", "v3", "pdf", "docx", "image", "stage"},
					Summary:               parseStageRuntimeDesc,
					SideEffectClass:       "read_only",
					Idempotence:           "idempotent",
					RuntimeConfigContract: parseStageRuntimeConfig(),
				},
				workItemUISchema("input", "moi:parse", "文档解析", parseStageRuntimeDesc, documentParseInputUIFields(parseStageRuntimeConfig(), "moi:parse")),                                                   // i18n-allow: zh WorkItem i18n pack source paired with en input UI title.
				workItemUISchema("output", "moi:parse", "文档解析", parseStageRuntimeDesc, nil),                                                                                                                 // i18n-allow: zh WorkItem i18n pack source paired with en output UI title.
				i18nPacks("文档解析", "Parse Document", "文档处理", "Document Processing", parseStageRuntimeDesc, parseStageRuntimeDescEN, documentParseInputUIFields(parseStageRuntimeConfig(), "moi:parse"), nil), // i18n-allow: zh WorkItem i18n pack source paired with en display name and group.
			),
			Handler: documentParseV3.Handle,
		},
		{
			Metadata: parseStageRuntimeLegacyMetadata(parseStageRuntimeDesc),
			Handler:  documentParseV3.Handle,
		},
		{
			Metadata: documentVisualMetadata(
				"moi:document_visual.parse",
				documentVisualParseDesc,
				documentVisualParseDescEN,
				"工程图纸解析",
				"Parse Drawing Visuals",
				"文档处理",
				"Document Processing",
				305,
				"transform",
				"read_only",
				[]string{"document_visual", "drawing", "pdf", "image", "parse", "ocr"},
				schemaDocumentVisualParseInput(),
				schemaDocumentVisualParseOutput(),
				documentVisualParseRuntimeConfig(),
				documentVisualParseRuntimeFields,
			),
			Handler: documentVisualParse.Handle,
		},
		{
			Metadata: documentVisualInternalMetadata(documentVisualMetadata(
				"moi:document_visual.parse.agent",
				documentVisualParseAgentDesc,
				documentVisualParseAgentDescEN,
				"Agent Runtime 图纸解析",
				"Parse Drawing Visuals with Agent Runtime",
				"文档处理",
				"Document Processing",
				306,
				"transform",
				"external_io",
				[]string{"document_visual", "drawing", "pdf", "image", "parse", "ocr", "codex"},
				schemaDocumentVisualParseCodexInput(),
				schemaDocumentVisualParseOutput(),
				nil,
				documentVisualCodexInputUIFields(),
			)),
			Handler: documentVisualParseCodex.Handle,
		},
		{
			Metadata: documentVisualMetadata(
				"moi:document_visual.index.text",
				documentVisualIndexTextDesc,
				documentVisualIndexTextDescEN,
				"图纸文本索引",
				"Index Drawing Text",
				"检索/知识索引",
				"Retrieval/Knowledge Index",
				552,
				"indexer",
				"writes_state",
				[]string{"document_visual", "index", "embedding", "vector", "text"},
				schemaDocumentVisualIndexTextInput(),
				schemaDocumentVisualIndexTextOutput(),
				nil,
				documentVisualIndexTextInputUIFields(),
			),
			Handler: documentVisualIndexText.Handle,
		},
		{
			Metadata: documentVisualMetadata(
				"moi:document_visual.index.image",
				documentVisualIndexImageDesc,
				documentVisualIndexImageDescEN,
				"图纸图片索引",
				"Index Drawing Images",
				"检索/知识索引",
				"Retrieval/Knowledge Index",
				553,
				"indexer",
				"writes_state",
				[]string{"document_visual", "index", "embedding", "vector", "image"},
				schemaDocumentVisualIndexImageInput(),
				schemaDocumentVisualIndexImageOutput(),
				nil,
				documentVisualIndexImageInputUIFields(),
			),
			Handler: documentVisualIndexImage.Handle,
		},
		{
			Metadata: workItemMetadata(
				knowledgeIndexBuildDesc,
				false,
				schemaKnowledgeIndexBuildInput(),
				schemaKnowledgeIndexBuildOutput(),
				&mowl.WorkItemMetadata{
					NodeId:                       "moi:knowledge.index.build",
					DisplayName:                  "构建知识库索引",
					Category:                     "moi",
					Visibility:                   "user",
					DisplayGroup:                 "检索/知识索引",
					DisplayOrder:                 500,
					NodeRole:                     "indexer",
					FinalOutput:                  true,
					Tags:                         []string{"knowledge", "index", "retrieval"},
					Summary:                      knowledgeIndexBuildDesc,
					SideEffectClass:              "writes_state",
					Idempotence:                  "idempotent",
					StateIdempotencyContractHash: "sha256:vector-index-upsert-overwrite-v1",
					RequiredFields:               []string{"documents"},
					RuntimeConfigContract:        knowledgeIndexBuildRuntimeConfig(),
				},
				workItemUISchema("input", "moi:knowledge.index.build", "构建知识库索引", knowledgeIndexBuildDesc, knowledgeIndexInputUIFields()),
				workItemUISchema("output", "moi:knowledge.index.build", "构建知识库索引", knowledgeIndexBuildDesc, nil),
				i18nPacks("构建知识库索引", "Build Knowledge Index", "检索/知识索引", "Retrieval/Knowledge Index", knowledgeIndexBuildDesc, knowledgeIndexBuildDescEN, knowledgeIndexInputUIFields(), nil),
			),
			Handler: knowledgeIndexBuild.Handle,
		},
		{Metadata: workItemMetadata("Transform upstream data into text, JSON, documents, rows or table payloads.", false, schemaDataTransformInput(), schemaDataTransformOutput(), &mowl.WorkItemMetadata{NodeId: "moi:data.transform", DisplayName: "moi:data.transform", Category: "moi", Visibility: "internal", Summary: "Transform upstream data into text, JSON, documents, rows or table payloads.", SideEffectClass: "read_only", Idempotence: "idempotent", Tags: []string{"internal"}}, workItemUISchema("input", "moi:data.transform", "moi:data.transform", "Transform upstream data into text, JSON, documents, rows or table payloads.", nil), workItemUISchema("output", "moi:data.transform", "moi:data.transform", "Transform upstream data into text, JSON, documents, rows or table payloads.", nil), nil), Handler: dataTransform.Handle},
		{Metadata: workItemMetadata("Route `sources` into document/image/audio/video arrays based on mime type or file extension.", false, schemaSourcesInput(), schemaRouterOutput(), &mowl.WorkItemMetadata{NodeId: "moi:parser.router.mime", DisplayName: "moi:parser.router.mime", Category: "moi", Visibility: "internal", Summary: "Route sources by MIME type.", SideEffectClass: "read_only", Idempotence: "idempotent", Tags: []string{"internal"}}, workItemUISchema("input", "moi:parser.router.mime", "moi:parser.router.mime", "Route `sources` into document/image/audio/video arrays based on mime type or file extension.", nil), workItemUISchema("output", "moi:parser.router.mime", "moi:parser.router.mime", "Route `sources` into document/image/audio/video arrays based on mime type or file extension.", nil), nil), Handler: workitems.RouteByMIME},
		{Metadata: workItemMetadata("Convert plain text-like sources into standardized `documents` entries.", false, schemaSourcesInput(), schemaDocumentsOutput(), &mowl.WorkItemMetadata{NodeId: "moi:parser.convert.plain", DisplayName: "moi:parser.convert.plain", Category: "moi", Visibility: "internal", Summary: "Convert plain sources to documents.", SideEffectClass: "read_only", Idempotence: "idempotent", Tags: []string{"internal"}}, workItemUISchema("input", "moi:parser.convert.plain", "moi:parser.convert.plain", "Convert plain text-like sources into standardized `documents` entries.", nil), workItemUISchema("output", "moi:parser.convert.plain", "moi:parser.convert.plain", "Convert plain text-like sources into standardized `documents` entries.", nil), nil), Handler: workitems.ConvertPlainToDocument},
		{Metadata: workItemMetadata("Convert HTML sources into structured text and table documents.", false, schemaSourcesInput(), schemaDocumentsOutput(), &mowl.WorkItemMetadata{NodeId: "moi:parser.convert.html", DisplayName: "moi:parser.convert.html", Category: "moi", Visibility: "internal", Summary: "Convert HTML sources to documents.", SideEffectClass: "read_only", Idempotence: "idempotent", Tags: []string{"internal"}}, workItemUISchema("input", "moi:parser.convert.html", "moi:parser.convert.html", "Convert HTML sources into structured text and table documents.", nil), workItemUISchema("output", "moi:parser.convert.html", "moi:parser.convert.html", "Convert HTML sources into structured text and table documents.", nil), nil), Handler: workitems.ConvertHTMLToDocument},
		{Metadata: workItemMetadata("Fetch online web pages from `urls` (or `sources`) and convert bodies into text; optional linked PDF/image extraction via parser API.", false, schemaWebCrawlInput(), schemaWebCrawlOutput(), &mowl.WorkItemMetadata{NodeId: "moi:parser.crawl.web", DisplayName: "moi:parser.crawl.web", Category: "moi", Visibility: "internal", Summary: "Fetch web pages and convert them into documents.", SideEffectClass: "external_io", Idempotence: "idempotent", Tags: []string{"internal"}}, workItemUISchema("input", "moi:parser.crawl.web", "moi:parser.crawl.web", "Fetch online web pages from `urls` (or `sources`) and convert bodies into text; optional linked PDF/image extraction via parser API.", nil), workItemUISchema("output", "moi:parser.crawl.web", "moi:parser.crawl.web", "Fetch online web pages from `urls` (or `sources`) and convert bodies into text; optional linked PDF/image extraction via parser API.", nil), nil), Handler: webCrawler.Handle},
		{Metadata: workItemMetadata("Rich document conversion with optional backend parser endpoints and local fallback logic.", false, schemaSourcesInput(), schemaDocumentsOutput(), &mowl.WorkItemMetadata{NodeId: "moi:parser.convert.document.rich", DisplayName: "moi:parser.convert.document.rich", Category: "moi", Visibility: "internal", Summary: "Rich document conversion.", SideEffectClass: "read_only", Idempotence: "idempotent", Tags: []string{"internal"}}, workItemUISchema("input", "moi:parser.convert.document.rich", "moi:parser.convert.document.rich", "Rich document conversion with optional backend parser endpoints and local fallback logic.", nil), workItemUISchema("output", "moi:parser.convert.document.rich", "moi:parser.convert.document.rich", "Rich document conversion with optional backend parser endpoints and local fallback logic.", nil), nil), Handler: (&workitems.RichConverter{Factory: factory, ClientOpts: clientOpts, Kind: "document", VersionRouter: versionRouter, ParserQueues: parserQueues}).Handle},
		{Metadata: workItemMetadata("PDF-focused rich conversion. Supports `options.pdf_backend_url`; fallback returns empty text when parser unavailable.", false, schemaSourcesInput(), schemaDocumentsOutput(), &mowl.WorkItemMetadata{NodeId: "moi:parser.convert.document.pdf.rich", DisplayName: "moi:parser.convert.document.pdf.rich", Category: "moi", Visibility: "internal", Summary: "PDF rich conversion.", SideEffectClass: "read_only", Idempotence: "idempotent", Tags: []string{"internal"}}, workItemUISchema("input", "moi:parser.convert.document.pdf.rich", "moi:parser.convert.document.pdf.rich", "PDF-focused rich conversion.", nil), workItemUISchema("output", "moi:parser.convert.document.pdf.rich", "moi:parser.convert.document.pdf.rich", "PDF-focused rich conversion.", nil), nil), Handler: (&workitems.RichConverter{Factory: factory, ClientOpts: clientOpts, Kind: "document", Subtype: "pdf", VersionRouter: versionRouter, ParserQueues: parserQueues}).Handle},
		{Metadata: workItemMetadata("DOCX text extraction only. Reads `word/document.xml` and outputs plain text.", false, schemaSourcesInput(), schemaDocumentsOutput(), &mowl.WorkItemMetadata{NodeId: "moi:parser.convert.document.docx.rich", DisplayName: "moi:parser.convert.document.docx.rich", Category: "moi", Visibility: "internal", Summary: "DOCX rich conversion.", SideEffectClass: "read_only", Idempotence: "idempotent", Tags: []string{"internal"}}, workItemUISchema("input", "moi:parser.convert.document.docx.rich", "moi:parser.convert.document.docx.rich", "DOCX text extraction only.", nil), workItemUISchema("output", "moi:parser.convert.document.docx.rich", "moi:parser.convert.document.docx.rich", "DOCX text extraction only.", nil), nil), Handler: (&workitems.RichConverter{Factory: factory, ClientOpts: clientOpts, Kind: "document", Subtype: "docx", VersionRouter: versionRouter, ParserQueues: parserQueues}).Handle},
		{Metadata: workItemMetadata("PPTX-focused rich conversion. Supports `options.pptx_backend_url`; fallback extracts slide XML text.", false, schemaSourcesInput(), schemaDocumentsOutput(), &mowl.WorkItemMetadata{NodeId: "moi:parser.convert.document.pptx.rich", DisplayName: "moi:parser.convert.document.pptx.rich", Category: "moi", Visibility: "internal", Summary: "PPTX rich conversion.", SideEffectClass: "read_only", Idempotence: "idempotent", Tags: []string{"internal"}}, workItemUISchema("input", "moi:parser.convert.document.pptx.rich", "moi:parser.convert.document.pptx.rich", "PPTX-focused rich conversion.", nil), workItemUISchema("output", "moi:parser.convert.document.pptx.rich", "moi:parser.convert.document.pptx.rich", "PPTX-focused rich conversion.", nil), nil), Handler: (&workitems.RichConverter{Factory: factory, ClientOpts: clientOpts, Kind: "document", Subtype: "pptx", VersionRouter: versionRouter, ParserQueues: parserQueues}).Handle},
		{Metadata: workItemMetadata("Image OCR conversion path for document ingestion. Supports `options.image_ocr_backend_url`.", false, schemaSourcesInput(), schemaDocumentsOutput(), &mowl.WorkItemMetadata{NodeId: "moi:parser.convert.document.image_ocr.rich", DisplayName: "moi:parser.convert.document.image_ocr.rich", Category: "moi", Visibility: "internal", Summary: "Image OCR rich conversion.", SideEffectClass: "read_only", Idempotence: "idempotent", Tags: []string{"internal"}}, workItemUISchema("input", "moi:parser.convert.document.image_ocr.rich", "moi:parser.convert.document.image_ocr.rich", "Image OCR conversion path for document ingestion.", nil), workItemUISchema("output", "moi:parser.convert.document.image_ocr.rich", "moi:parser.convert.document.image_ocr.rich", "Image OCR conversion path for document ingestion.", nil), nil), Handler: (&workitems.RichConverter{Factory: factory, ClientOpts: clientOpts, Kind: "document", Subtype: "image_ocr", VersionRouter: versionRouter, ParserQueues: parserQueues}).Handle},
		{Metadata: workItemMetadata(imageParseDesc, false, schemaImageParseInput(), schemaDocumentsOutput(), &mowl.WorkItemMetadata{NodeId: "moi:parser.convert.image.rich", DisplayName: "图片解析", Category: "moi", Visibility: "user", DisplayGroup: "文档处理", DisplayOrder: 314, NodeRole: "transform", Summary: imageParseDesc, SideEffectClass: "external_io", Idempotence: "idempotent", RuntimeConfigContract: imageParseRuntimeConfig(), Tags: []string{"image", "ocr", "caption", "parse"}}, workItemUISchema("input", "moi:parser.convert.image.rich", "图片解析", imageParseDesc, imageParseInputUIFields()), workItemUISchema("output", "moi:parser.convert.image.rich", "图片解析", imageParseDesc, nil), i18nPacks("图片解析", "Parse Image", "文档处理", "Document Processing", imageParseDesc, imageParseDescEN, imageParseInputUIFields(), nil)), Handler: (&workitems.RichConverter{Factory: factory, ClientOpts: clientOpts, Kind: "image", VersionRouter: versionRouter, ParserQueues: parserQueues}).Handle}, // i18n-allow: zh WorkItem i18n pack source paired with en display name and group.
		mediaRichRegistration("moi:parser.convert.audio.rich", "音频解析", "Parse Audio", audioParseDesc, audioParseDescEN, "audio", 315, []string{"audio", "asr", "parse", "vad"}, factory, clientOpts, versionRouter, parserQueues),
		mediaRichRegistration("moi:parser.convert.video.rich", "视频解析", "Parse Video", videoParseDesc, videoParseDescEN, "video", 316, []string{"video", "asr", "parse", "vad"}, factory, clientOpts, versionRouter, parserQueues), // i18n-allow: zh WorkItem i18n pack source paired with en display name.
		{Metadata: workItemMetadata(cleanTextDesc, false, schemaCleanTextInput(), schemaTextOutput(), &mowl.WorkItemMetadata{NodeId: "moi:parser.clean.text", DisplayName: "文本清洗", Category: "moi", Visibility: "user", DisplayGroup: "文档处理", DisplayOrder: 320, NodeRole: "transform", Summary: cleanTextDesc, SideEffectClass: "read_only", Idempotence: "idempotent", Tags: []string{"text", "clean"}}, workItemUISchema("input", "moi:parser.clean.text", "文本清洗", cleanTextDesc, cleanTextInputUIFields()), workItemUISchema("output", "moi:parser.clean.text", "文本清洗", cleanTextDesc, nil), i18nPacks("文本清洗", "Clean Text", "文档处理", "Document Processing", cleanTextDesc, cleanTextDescEN, cleanTextInputUIFields(), nil)), Handler: workitems.CleanText}, // i18n-allow: zh WorkItem i18n pack source paired with en display name and group.
		{Metadata: workItemMetadata("Split long text into chunk windows using `chunk_size` and `overlap`.", false, schemaSplitLengthInput(), schemaSplitLengthOutput(), &mowl.WorkItemMetadata{NodeId: "moi:parser.split.length", DisplayName: "moi:parser.split.length", Category: "moi", Visibility: "internal", Summary: "Split long text into chunks.", SideEffectClass: "read_only", Idempotence: "idempotent", Tags: []string{"internal"}}, workItemUISchema("input", "moi:parser.split.length", "moi:parser.split.length", "Split long text into chunk windows using `chunk_size` and `overlap`.", runtimeConfigInputUIFields(splitLengthRuntimeConfig())), workItemUISchema("output", "moi:parser.split.length", "moi:parser.split.length", "Split long text into chunk windows using `chunk_size` and `overlap`.", nil), nil), Handler: workitems.SplitLength},
		{
			Metadata: workItemMetadata(
				splitLevelDesc,
				false,
				schemaDocumentsInput(),
				schemaDocumentsOutput(),
				&mowl.WorkItemMetadata{
					NodeId:          "moi:parser.split.level",
					DisplayName:     "按层级切分文档",
					Category:        "moi",
					Visibility:      "user",
					DisplayGroup:    "文档处理",
					DisplayOrder:    330,
					Summary:         splitLevelDesc,
					SideEffectClass: "read_only",
					Idempotence:     "idempotent",
					Tags:            []string{"document", "split"},
				},
				workItemUISchema("input", "moi:parser.split.level", "按层级切分文档", splitLevelDesc, splitLevelInputUIFields()),
				workItemUISchema("output", "moi:parser.split.level", "按层级切分文档", splitLevelDesc, nil),
				i18nPacks("按层级切分文档", "Split Documents by Heading", "文档处理", "Document Processing", splitLevelDesc, splitLevelDescEN, splitLevelInputUIFields(), nil),
			),
			Handler: workitems.SplitByLevel,
		},
		{
			Metadata: workItemMetadata(
				splitDocumentsLengthDesc,
				false,
				schemaSplitDocumentsLengthInput(),
				schemaSplitDocumentsOutput(),
				&mowl.WorkItemMetadata{
					NodeId:                "moi:parser.split.documents.length",
					DisplayName:           "按长度切分文档",
					Category:              "moi",
					Visibility:            "user",
					DisplayGroup:          "文档处理",
					DisplayOrder:          320,
					Summary:               splitDocumentsLengthDesc,
					SideEffectClass:       "read_only",
					Idempotence:           "idempotent",
					RuntimeConfigContract: splitLengthRuntimeConfig(),
					Tags:                  []string{"document", "split"},
				},
				workItemUISchema("input", "moi:parser.split.documents.length", "按长度切分文档", splitDocumentsLengthDesc, splitDocumentsLengthInputFields), // i18n-allow: zh WorkItem input title paired with the en locale pack.
				workItemUISchema("output", "moi:parser.split.documents.length", "按长度切分文档", splitDocumentsLengthDesc, nil),                            // i18n-allow: zh WorkItem output title paired with the en locale pack.
				splitDocumentsLengthI18N,
			),
			Handler: workitems.SplitDocumentsLength,
		},
		{Metadata: workItemMetadata("Expand chunk documents into doc/section/chunk multi-level index entries.", false, schemaMultiLevelIndexInput(), schemaDocumentsOutput(), &mowl.WorkItemMetadata{NodeId: "moi:retrieval.index.multilevel", DisplayName: "moi:retrieval.index.multilevel", Category: "moi", Visibility: "internal", Summary: "Expand documents into multi-level index entries.", SideEffectClass: "read_only", Idempotence: "idempotent", Tags: []string{"internal"}}, workItemUISchema("input", "moi:retrieval.index.multilevel", "moi:retrieval.index.multilevel", "Expand chunk documents into doc/section/chunk multi-level index entries.", nil), workItemUISchema("output", "moi:retrieval.index.multilevel", "moi:retrieval.index.multilevel", "Expand chunk documents into doc/section/chunk multi-level index entries.", nil), nil), Handler: multiLevelIndex.Handle},
		{Metadata: workItemMetadata("Repair common malformed JSON input and report validity.", false, schemaJSONRepairInput(), schemaJSONRepairOutput(), &mowl.WorkItemMetadata{NodeId: "moi:parser.json.repair", DisplayName: "moi:parser.json.repair", Category: "moi", Visibility: "internal", Summary: "Repair malformed JSON text.", SideEffectClass: "read_only", Idempotence: "idempotent", Tags: []string{"internal"}}, workItemUISchema("input", "moi:parser.json.repair", "moi:parser.json.repair", "Repair common malformed JSON input.", nil), workItemUISchema("output", "moi:parser.json.repair", "moi:parser.json.repair", "Repair common malformed JSON input.", nil), nil), Handler: workitems.RepairJSON},
		{Metadata: workItemMetadata("Unified structured extraction supporting single text, n_to_1 and n_to_n document modes, file inputs, and JSON Schema-driven field extraction.", false, schemaUnifiedExtractInput(), schemaUnifiedExtractOutput(), &mowl.WorkItemMetadata{NodeId: "moi:llm.extract.structured", DisplayName: "moi:llm.extract.structured", Category: "moi", Visibility: "internal", Summary: "Structured extraction.", SideEffectClass: "read_only", Idempotence: "idempotent", RuntimeConfigContract: structuredExtractRuntimeConfig(), Tags: []string{"internal"}}, workItemUISchema("input", "moi:llm.extract.structured", "moi:llm.extract.structured", "Structured extraction.", structuredExtractInputUIFields()), workItemUISchema("output", "moi:llm.extract.structured", "moi:llm.extract.structured", "Structured extraction.", nil), nil), Handler: unifiedExtractItem.Handle},
		{Metadata: workItemMetadata(structuredExtractDesc, false, schemaUnifiedExtractInput(), schemaUnifiedExtractOutput(), &mowl.WorkItemMetadata{NodeId: "moi:llm.extract.structured.advanced", DisplayName: "结构化抽取", Category: "moi", Visibility: "user", DisplayGroup: "抽取", DisplayOrder: 600, Summary: structuredExtractDesc, SideEffectClass: "read_only", Idempotence: "idempotent", RuntimeConfigContract: structuredExtractRuntimeConfig(), Tags: []string{"extract", "llm"}}, workItemUISchema("input", "moi:llm.extract.structured.advanced", "结构化抽取", structuredExtractDesc, structuredExtractInputUIFields()), workItemUISchema("output", "moi:llm.extract.structured.advanced", "结构化抽取", structuredExtractDesc, nil), i18nPacks("结构化抽取", "Structured Extraction", "抽取", "Extraction", structuredExtractDesc, structuredExtractDescEN, structuredExtractInputUIFields(), nil)), Handler: unifiedExtractItem.Handle},
		{Metadata: workItemMetadata(embeddingGenerateDesc, false, schemaEmbeddingGenerateInput(), schemaDocumentsOutput(), &mowl.WorkItemMetadata{NodeId: "moi:embedding.generate", DisplayName: "文本向量化", Category: "moi", Visibility: "user", DisplayGroup: "智能处理", DisplayOrder: 170, Summary: embeddingGenerateDesc, SideEffectClass: "read_only", Idempotence: "idempotent", Tags: []string{"embedding", "documents"}}, workItemUISchema("input", "moi:embedding.generate", "文本向量化", embeddingGenerateDesc, embeddingGenerateInputUIFields()), workItemUISchema("output", "moi:embedding.generate", "文本向量化", embeddingGenerateDesc, nil), i18nPacks("文本向量化", "Text Embedding", "智能处理", "AI Processing", embeddingGenerateDesc, embeddingGenerateDescEN, embeddingGenerateInputUIFields(), nil)), Handler: embeddingItem.Handle},
		{
			Metadata: workItemMetadata(
				lineageRegisterDesc,
				false,
				schemaDataLineageRegisterInput(),
				schemaDataLineageRegisterOutput(),
				&mowl.WorkItemMetadata{
					NodeId:          "moi:data.lineage.register",
					DisplayName:     "登记数据血缘",
					Category:        "moi",
					Visibility:      "user",
					DisplayGroup:    "治理",
					DisplayOrder:    700,
					Summary:         lineageRegisterDesc,
					SideEffectClass: "writes_state",
					Idempotence:     "idempotent",
					Tags:            []string{"lineage", "governance"},
				},
				workItemUISchema("input", "moi:data.lineage.register", "登记数据血缘", lineageRegisterDesc, dataLineageRegisterInputUIFields()),
				workItemUISchema("output", "moi:data.lineage.register", "登记数据血缘", lineageRegisterDesc, nil),
				i18nPacks("登记数据血缘", "Register Data Lineage", "治理", "Governance", lineageRegisterDesc, lineageRegisterDescEN, dataLineageRegisterInputUIFields(), nil),
			),
			Handler: lineageRegister.Handle,
		},
		{Metadata: workItemMetadata("Register a typed data asset for lineage tracking.", false, schemaDataAssetRegisterInput(), schemaDataAsset(), &mowl.WorkItemMetadata{NodeId: "moi:data.asset.register", DisplayName: "moi:data.asset.register", Category: "moi", Visibility: "internal", Summary: "Register a data asset.", SideEffectClass: "writes_state", Idempotence: "idempotent", Tags: []string{"internal"}}, workItemUISchema("input", "moi:data.asset.register", "moi:data.asset.register", "Register a typed data asset for lineage tracking.", nil), workItemUISchema("output", "moi:data.asset.register", "moi:data.asset.register", "Register a typed data asset for lineage tracking.", nil), nil), Handler: assetRegister.Handle},
		{Metadata: workItemMetadata("Create a derivation link between two typed data assets.", false, schemaDataAssetLinkInput(), schemaDataDerivation(), &mowl.WorkItemMetadata{NodeId: "moi:data.asset.link", DisplayName: "moi:data.asset.link", Category: "moi", Visibility: "internal", Summary: "Create an asset derivation link.", SideEffectClass: "writes_state", Idempotence: "idempotent", Tags: []string{"internal"}}, workItemUISchema("input", "moi:data.asset.link", "moi:data.asset.link", "Create a derivation link between two typed data assets.", nil), workItemUISchema("output", "moi:data.asset.link", "moi:data.asset.link", "Create a derivation link between two typed data assets.", nil), nil), Handler: assetLink.Handle},
		{Metadata: workItemMetadata("Upsert parsed manifest mapping for a data asset.", false, schemaDataDocMapInput(), schemaParsedManifest(), &mowl.WorkItemMetadata{NodeId: "moi:data.doc.map_metadata", DisplayName: "moi:data.doc.map_metadata", Category: "moi", Visibility: "internal", Summary: "Upsert parsed manifest mapping.", SideEffectClass: "writes_state", Idempotence: "idempotent", Tags: []string{"internal"}}, workItemUISchema("input", "moi:data.doc.map_metadata", "moi:data.doc.map_metadata", "Upsert parsed manifest mapping for a data asset.", nil), workItemUISchema("output", "moi:data.doc.map_metadata", "moi:data.doc.map_metadata", "Upsert parsed manifest mapping for a data asset.", nil), nil), Handler: docMap.Handle},
		{Metadata: workItemMetadata(tableUpsertJSONDesc, false, schemaDataTableUpsertJSONInput(), schemaDataTableUpsertJSONOutput(), &mowl.WorkItemMetadata{NodeId: "moi:data.table.upsert_json", DisplayName: "写入数据表", Category: "moi", Visibility: "user", DisplayGroup: "结构化数据", DisplayOrder: 220, Summary: tableUpsertJSONDesc, SideEffectClass: "writes_state", Idempotence: "non_idempotent_or_requires_key", Tags: []string{"table", "json"}}, workItemUISchema("input", "moi:data.table.upsert_json", "写入数据表", tableUpsertJSONDesc, dataTableUpsertJSONInputUIFields()), workItemUISchema("output", "moi:data.table.upsert_json", "写入数据表", tableUpsertJSONDesc, nil), i18nPacks("写入数据表", "Write Data Table", "结构化数据", "Structured Data", tableUpsertJSONDesc, tableUpsertJSONDescEN, dataTableUpsertJSONInputUIFields(), nil)), Handler: tableUpsertJSON.Handle},
		{
			Metadata: workItemMetadata(
				emailArchiveETLDesc,
				false,
				schemaEmailArchiveETLInput(),
				schemaEmailArchiveETLOutput(),
				&mowl.WorkItemMetadata{
					NodeId:          "moi:email.archive.etl",
					DisplayName:     "邮件归档解析", // i18n-allow: zh locale workitem display name
					Category:        "moi",
					Visibility:      "user",
					DisplayGroup:    "结构化数据", // i18n-allow: zh locale workitem display group
					DisplayOrder:    230,
					NodeRole:        "transform",
					Summary:         emailArchiveETLDesc,
					SideEffectClass: "writes_state",
					Idempotence:     "non_idempotent_or_requires_key",
					Tags:            []string{"email", "archive", "etl", "table", "maildir"},
				},
				workItemUISchema("input", "moi:email.archive.etl", "邮件归档解析", emailArchiveETLDesc, emailArchiveETLInputUIFields()),                                       // i18n-allow: zh locale workitem UI schema
				workItemUISchema("output", "moi:email.archive.etl", "邮件归档解析", emailArchiveETLDesc, nil),                                                                 // i18n-allow: zh locale workitem UI schema
				i18nPacks("邮件归档解析", "Parse Email Archive", "结构化数据", "Structured Data", emailArchiveETLDesc, emailArchiveETLDescEN, emailArchiveETLInputUIFields(), nil), // i18n-allow: zh locale workitem i18n pack
			),
			Handler: emailArchiveETL.Handle,
		},
		{
			Metadata: workItemMetadata(
				runSQLDesc,
				false,
				schemaRunSQLInput(),
				schemaRunSQLOutput(),
				&mowl.WorkItemMetadata{
					NodeId:          "moi:data.runsql",
					DisplayName:     "moi:data.runsql",
					Category:        "moi",
					Visibility:      "internal",
					Summary:         runSQLDesc,
					SideEffectClass: "writes_state",
					Idempotence:     "non_idempotent_or_requires_key",
					Tags:            []string{"internal", "sql", "database"},
				},
				workItemUISchema("input", "moi:data.runsql", "moi:data.runsql", runSQLDesc, requiredInputUIFields([]string{"sql"}, nil)),
				workItemUISchema("output", "moi:data.runsql", "moi:data.runsql", runSQLDesc, nil),
				nil,
			),
			Handler: runSQL.Handle,
		},
		{
			Metadata: workItemMetadata(
				sqlProcessDesc,
				false,
				schemaSQLProcessInput(),
				schemaSQLProcessOutput(),
				&mowl.WorkItemMetadata{
					NodeId:          "moi:data.sql.process",
					DisplayName:     "SQL 处理",
					Category:        "moi",
					Visibility:      "user",
					DisplayGroup:    "代码处理",
					DisplayOrder:    200,
					Summary:         sqlProcessDesc,
					SideEffectClass: "writes_state",
					Idempotence:     "non_idempotent_or_requires_key",
					Tags:            []string{"sql", "database", "code"},
				},
				workItemUISchema("input", "moi:data.sql.process", "SQL 处理", sqlProcessDesc, sqlProcessInputUIFields()),
				workItemUISchema("output", "moi:data.sql.process", "SQL 处理", sqlProcessDesc, nil),
				i18nPacks("SQL 处理", "SQL Processing", "代码处理", "Code Processing", sqlProcessDesc, sqlProcessDescEN, sqlProcessInputUIFields(), nil),
			),
			Handler: (&workitems.SQLProcess{RunSQL: runSQL, Factory: factory}).Handle,
		},
		{
			Metadata: workItemMetadata(
				apiRequestDesc,
				false,
				schemaAPIRequestInput(),
				schemaAPIRequestOutput(),
				&mowl.WorkItemMetadata{
					NodeId:          "moi:api.request",
					DisplayName:     "API 调用",
					Category:        "moi",
					Visibility:      "user",
					DisplayGroup:    "数据 IO",
					DisplayOrder:    70,
					Tags:            []string{"api", "http", "webhook", "external"},
					Summary:         apiRequestDesc,
					SideEffectClass: "external_call",
					Idempotence:     "non_idempotent_or_requires_key",
				},
				workItemUISchema("input", "moi:api.request", "API 调用", apiRequestDesc, apiRequestInputUIFields()),
				workItemUISchema("output", "moi:api.request", "API 调用", apiRequestDesc, apiRequestOutputUIFields()),
				i18nPacks("API 调用", "API Request", "数据 IO", "Data IO", apiRequestDesc, apiRequestDescEN, apiRequestInputUIFields(), apiRequestOutputUIFields()),
			),
			Handler: apiRequest.Handle,
		},
		{
			Metadata: workItemMetadata(
				githubRepoReadDesc,
				false,
				schemaGitHubRepoReadInput(),
				schemaChannelMessageSendOutput(),
				&mowl.WorkItemMetadata{
					NodeId:          "moi:github.repo.read",
					DisplayName:     "moi:github.repo.read",
					Category:        "moi",
					Visibility:      "internal",
					Summary:         githubRepoReadDesc,
					SideEffectClass: "read",
					Idempotence:     "idempotent",
					Tags:            []string{"internal", "channel", "github", "repository"},
				},
				workItemUISchema("input", "moi:github.repo.read", "moi:github.repo.read", githubRepoReadDesc, nil),
				workItemUISchema("output", "moi:github.repo.read", "moi:github.repo.read", githubRepoReadDesc, nil),
				i18nPacks("moi:github.repo.read", "moi:github.repo.read", "内部", "Internal", githubRepoReadDesc, githubRepoReadDescEN, nil, nil), // i18n-allow: zh locale workitem display group.
			),
			Handler: githubRepoRead.Handle,
		},
		{
			Metadata: workItemMetadata(
				githubRepoWriteDesc,
				false,
				schemaGitHubRepoWriteInput(),
				schemaChannelMessageSendOutput(),
				&mowl.WorkItemMetadata{
					NodeId:          "moi:github.repo.write",
					DisplayName:     "moi:github.repo.write",
					Category:        "moi",
					Visibility:      "internal",
					Summary:         githubRepoWriteDesc,
					SideEffectClass: "external_call",
					Idempotence:     "non_idempotent_or_requires_key",
					Tags:            []string{"internal", "channel", "github", "repository", "write"},
				},
				workItemUISchema("input", "moi:github.repo.write", "moi:github.repo.write", githubRepoWriteDesc, nil),
				workItemUISchema("output", "moi:github.repo.write", "moi:github.repo.write", githubRepoWriteDesc, nil),
				i18nPacks("moi:github.repo.write", "moi:github.repo.write", "内部", "Internal", githubRepoWriteDesc, githubRepoWriteDescEN, nil, nil), // i18n-allow: zh locale workitem display group.
			),
			Handler: githubRepoWrite.Handle,
		},
		{
			Metadata: workItemMetadata(
				githubRepoWorkspaceDesc,
				false,
				schemaGitHubRepoWorkspaceInput(),
				schemaChannelMessageSendOutput(),
				&mowl.WorkItemMetadata{
					NodeId:          "moi:github.repo.workspace",
					DisplayName:     "moi:github.repo.workspace",
					Category:        "moi",
					Visibility:      "internal",
					Summary:         githubRepoWorkspaceDesc,
					SideEffectClass: "write",
					Idempotence:     "idempotent",
					Tags:            []string{"internal", "channel", "github", "repository", "workspace"},
				},
				workItemUISchema("input", "moi:github.repo.workspace", "moi:github.repo.workspace", githubRepoWorkspaceDesc, nil),
				workItemUISchema("output", "moi:github.repo.workspace", "moi:github.repo.workspace", githubRepoWorkspaceDesc, nil),
				i18nPacks("moi:github.repo.workspace", "moi:github.repo.workspace", "内部", "Internal", githubRepoWorkspaceDesc, githubRepoWorkspaceDescEN, nil, nil), // i18n-allow: zh locale workitem display group.
			),
			Handler: githubRepoWorkspace.Handle,
		},
		{
			Metadata: workItemMetadata(
				productSourceDesc,
				false,
				schemaProductSourceInput(),
				schemaChannelMessageSendOutput(),
				&mowl.WorkItemMetadata{
					NodeId:          "moi:product.source",
					DisplayName:     "moi:product.source",
					Category:        "moi",
					Visibility:      "internal",
					Summary:         productSourceDesc,
					SideEffectClass: "read",
					Idempotence:     "idempotent",
					Tags:            []string{"internal", "product", "source", "offline", "workspace"},
				},
				workItemUISchema("input", "moi:product.source", "moi:product.source", productSourceDesc, nil),
				workItemUISchema("output", "moi:product.source", "moi:product.source", productSourceDesc, nil),
				i18nPacks("moi:product.source", "moi:product.source", "内部", "Internal", productSourceDesc, productSourceDescEN, nil, nil), // i18n-allow: zh locale workitem display group.
			),
			Handler: productSource.Handle,
		},
		{
			Metadata: workItemMetadata(
				codexRunDesc,
				false,
				schemaCodexRunInput(),
				schemaCodexRunOutput(),
				&mowl.WorkItemMetadata{
					NodeId:          "moi:codex.run",
					DisplayName:     "moi:codex.run",
					Category:        "moi",
					Visibility:      "internal",
					Summary:         codexRunDesc,
					SideEffectClass: "read",
					Idempotence:     "idempotent",
					Tags:            []string{"internal", "channel", "codex", "agent", "source"},
				},
				workItemUISchema("input", "moi:codex.run", "moi:codex.run", codexRunDesc, nil),
				workItemUISchema("output", "moi:codex.run", "moi:codex.run", codexRunDesc, nil),
				i18nPacks("moi:codex.run", "moi:codex.run", "内部", "Internal", codexRunDesc, codexRunDescEN, nil, nil), // i18n-allow: zh locale workitem display group.
			),
			Handler: codexRun.Handle,
		},
		{
			Metadata: workItemMetadata(
				grafanaReadDesc,
				false,
				schemaGrafanaReadInput(),
				schemaChannelMessageSendOutput(),
				&mowl.WorkItemMetadata{
					NodeId:          "moi:grafana.read",
					DisplayName:     "moi:grafana.read",
					Category:        "moi",
					Visibility:      "internal",
					Summary:         grafanaReadDesc,
					SideEffectClass: "read",
					Idempotence:     "idempotent",
					Tags:            []string{"internal", "channel", "grafana", "observability"},
				},
				workItemUISchema("input", "moi:grafana.read", "moi:grafana.read", grafanaReadDesc, nil),
				workItemUISchema("output", "moi:grafana.read", "moi:grafana.read", grafanaReadDesc, nil),
				i18nPacks("moi:grafana.read", "moi:grafana.read", "内部", "Internal", grafanaReadDesc, grafanaReadDescEN, nil, nil), // i18n-allow: zh locale workitem display group.
			),
			Handler: grafanaRead.Handle,
		},
		{
			Metadata: workItemMetadata(
				kubernetesReadDesc,
				false,
				schemaKubernetesReadInput(),
				schemaChannelMessageSendOutput(),
				&mowl.WorkItemMetadata{
					NodeId:          "moi:kubernetes.read",
					DisplayName:     "moi:kubernetes.read",
					Category:        "moi",
					Visibility:      "internal",
					Summary:         kubernetesReadDesc,
					SideEffectClass: "read",
					Idempotence:     "idempotent",
					Tags:            []string{"internal", "channel", "kubernetes", "infrastructure"},
				},
				workItemUISchema("input", "moi:kubernetes.read", "moi:kubernetes.read", kubernetesReadDesc, nil),
				workItemUISchema("output", "moi:kubernetes.read", "moi:kubernetes.read", kubernetesReadDesc, nil),
				i18nPacks("moi:kubernetes.read", "moi:kubernetes.read", "内部", "Internal", kubernetesReadDesc, kubernetesReadDescEN, nil, nil), // i18n-allow: zh locale workitem display group.
			),
			Handler: kubernetesRead.Handle,
		},
		{
			Metadata: workItemMetadata(
				feishuMessageSendDesc,
				false,
				schemaFeishuMessageSendInput(),
				schemaChannelMessageSendOutput(),
				&mowl.WorkItemMetadata{
					NodeId:          "moi:feishu.message.send",
					DisplayName:     "moi:feishu.message.send",
					Category:        "moi",
					Visibility:      "internal",
					Summary:         feishuMessageSendDesc,
					SideEffectClass: "external_call",
					Idempotence:     "non_idempotent_or_requires_key",
					Tags:            []string{"internal", "channel", "feishu", "message"},
				},
				workItemUISchema("input", "moi:feishu.message.send", "moi:feishu.message.send", feishuMessageSendDesc, nil),
				workItemUISchema("output", "moi:feishu.message.send", "moi:feishu.message.send", feishuMessageSendDesc, nil),
				i18nPacks("moi:feishu.message.send", "moi:feishu.message.send", "内部", "Internal", feishuMessageSendDesc, feishuMessageSendDescEN, nil, nil), // i18n-allow: zh locale workitem display group.
			),
			Handler: feishuMessageSend.Handle,
		},
		{
			Metadata: workItemMetadata(
				slackMessageSendDesc,
				false,
				schemaSlackMessageSendInput(),
				schemaChannelMessageSendOutput(),
				&mowl.WorkItemMetadata{
					NodeId:          "moi:slack.message.send",
					DisplayName:     "moi:slack.message.send",
					Category:        "moi",
					Visibility:      "internal",
					Summary:         slackMessageSendDesc,
					SideEffectClass: "external_call",
					Idempotence:     "non_idempotent_or_requires_key",
					Tags:            []string{"internal", "channel", "slack", "message"},
				},
				workItemUISchema("input", "moi:slack.message.send", "moi:slack.message.send", slackMessageSendDesc, nil),
				workItemUISchema("output", "moi:slack.message.send", "moi:slack.message.send", slackMessageSendDesc, nil),
				i18nPacks("moi:slack.message.send", "moi:slack.message.send", "内部", "Internal", slackMessageSendDesc, slackMessageSendDescEN, nil, nil), // i18n-allow: zh locale workitem display group.
			),
			Handler: slackMessageSend.Handle,
		},
		{
			Metadata: workItemMetadata(
				llmOutputGenerateDesc,
				false,
				schemaLLMOutputGenerateInput(),
				schemaLLMOutputGenerateOutput(),
				&mowl.WorkItemMetadata{
					NodeId:          "moi:llm.output.generate",
					DisplayName:     "模型推理",
					Category:        "moi",
					Visibility:      "user",
					DisplayGroup:    "智能处理",
					DisplayOrder:    180,
					Tags:            []string{"llm", "inference", "generation", "text"},
					Summary:         llmOutputGenerateDesc,
					SideEffectClass: "external_call",
					Idempotence:     "non_idempotent_or_requires_key",
				},
				workItemUISchema("input", "moi:llm.output.generate", "模型推理", llmOutputGenerateDesc, llmOutputGenerateInputUIFields()),
				workItemUISchema("output", "moi:llm.output.generate", "模型推理", llmOutputGenerateDesc, nil),
				i18nPacks("模型推理", "Model Inference", "智能处理", "AI Processing", llmOutputGenerateDesc, llmOutputGenerateDescEN, llmOutputGenerateInputUIFields(), nil),
			),
			Handler: llmOutputGenerate.Handle,
		},
		{
			Metadata: workItemMetadata(
				workflowTriggerDesc,
				false,
				schemaWorkflowTriggerInput(),
				schemaWorkflowTriggerOutput(),
				&mowl.WorkItemMetadata{
					NodeId:          "moi:workflow.trigger",
					DisplayName:     "工作流触发",
					Category:        "moi",
					Visibility:      "user",
					DisplayGroup:    "流程控制",
					DisplayOrder:    240,
					Tags:            []string{"workflow", "trigger", "orchestration"},
					Summary:         workflowTriggerDesc,
					SideEffectClass: "external_call",
					Idempotence:     "non_idempotent_or_requires_key",
				},
				workItemUISchema("input", "moi:workflow.trigger", "工作流触发", workflowTriggerDesc, workflowTriggerInputUIFields()),
				workItemUISchema("output", "moi:workflow.trigger", "工作流触发", workflowTriggerDesc, nil),
				i18nPacks("工作流触发", "Workflow Trigger", "流程控制", "Flow Control", workflowTriggerDesc, workflowTriggerDescEN, workflowTriggerInputUIFields(), nil),
			),
			Handler: workflowTrigger.Handle,
		},
		{Metadata: workItemMetadata("Execute SQL statements in three phases, with optional replace_by_clone mode for staging-table refreshes using CREATE TABLE ... CLONE.", false, schemaSQLPipelineInput(), schemaSQLPipelineOutput(), &mowl.WorkItemMetadata{NodeId: "moi:data.sql_pipeline", DisplayName: "批量执行 SQL", Category: "moi", Visibility: "internal", Summary: "批量执行 SQL 语句。", SideEffectClass: "writes_state", Idempotence: "non_idempotent_or_requires_key", Tags: []string{"internal", "sql", "database"}}, workItemUISchema("input", "moi:data.sql_pipeline", "批量执行 SQL", "批量执行 SQL 语句。", nil), workItemUISchema("output", "moi:data.sql_pipeline", "批量执行 SQL", "批量执行 SQL 语句。", nil), nil), Handler: sqlPipeline.Handle},
		{Metadata: workItemMetadata("Embed documents and upsert them into the vector index table.", false, schemaVectorWriteInput(), schemaVectorWriteOutput(), &mowl.WorkItemMetadata{NodeId: "moi:data.retrieval.vector.write", DisplayName: "moi:data.retrieval.vector.write", Category: "moi", Visibility: "internal", Summary: "Write documents to vector index.", SideEffectClass: "writes_state", Idempotence: "idempotent", Tags: []string{"internal"}}, workItemUISchema("input", "moi:data.retrieval.vector.write", "moi:data.retrieval.vector.write", "Write documents to vector index.", nil), workItemUISchema("output", "moi:data.retrieval.vector.write", "moi:data.retrieval.vector.write", "Write documents to vector index.", nil), nil), Handler: dataVectorWrite.Handle},
		{Metadata: workItemMetadata("Get file metadata by file_id via moi-core Files API.", false, schemaFileMetadataGetInput(), schemaFileMetadata(), &mowl.WorkItemMetadata{NodeId: "moi:files.metadata.get", DisplayName: "moi:files.metadata.get", Category: "moi", Visibility: "internal", Summary: "Get file metadata.", SideEffectClass: "read_only", Idempotence: "idempotent", Tags: []string{"internal"}}, workItemUISchema("input", "moi:files.metadata.get", "moi:files.metadata.get", "Get file metadata.", nil), workItemUISchema("output", "moi:files.metadata.get", "moi:files.metadata.get", "Get file metadata.", nil), nil), Handler: fileMetaGet.Handle},
		{Metadata: workItemMetadata("Disabled: file content must not be passed through workflow data.", false, schemaFileReadTextInput(), schemaFileReadTextOutput(), &mowl.WorkItemMetadata{NodeId: "moi:files.read_text", DisplayName: "moi:files.read_text", Category: "moi", Visibility: "internal", Summary: "Disabled file text reader.", SideEffectClass: "read_only", Idempotence: "idempotent", Tags: []string{"internal"}}, workItemUISchema("input", "moi:files.read_text", "moi:files.read_text", "Disabled file text reader.", nil), workItemUISchema("output", "moi:files.read_text", "moi:files.read_text", "Disabled file text reader.", nil), nil), Handler: fileItem.Handle},
		{Metadata: workItemMetadata("Disabled: file content must not be passed through workflow data.", false, schemaReadDocsInput(), schemaReadDocsOutput(), &mowl.WorkItemMetadata{NodeId: "moi:files.read_documents", DisplayName: "moi:files.read_documents", Category: "moi", Visibility: "internal", Summary: "Disabled documents file reader.", SideEffectClass: "read_only", Idempotence: "idempotent", Tags: []string{"internal"}}, workItemUISchema("input", "moi:files.read_documents", "moi:files.read_documents", "Disabled documents file reader.", nil), workItemUISchema("output", "moi:files.read_documents", "moi:files.read_documents", "Disabled documents file reader.", nil), nil), Handler: readDocs.Handle},
		{
			Metadata: workItemMetadata(
				filesWriteDocumentsDesc,
				false,
				schemaWriteDocsInput(),
				schemaWriteDocsOutput(),
				&mowl.WorkItemMetadata{
					NodeId:                "moi:files.write_documents",
					DisplayName:           "写出文档文件",
					Category:              "moi",
					Visibility:            "user",
					DisplayGroup:          "文件",
					DisplayOrder:          800,
					Summary:               filesWriteDocumentsDesc,
					SideEffectClass:       "writes_state",
					Idempotence:           "non_idempotent_or_requires_key",
					RuntimeConfigContract: filesWriteDocumentsRuntimeConfig(),
					Tags:                  []string{"file", "documents"},
				},
				workItemUISchema("input", "moi:files.write_documents", "写出文档文件", filesWriteDocumentsDesc, filesWriteDocumentsInputUIFields()),
				workItemUISchema("output", "moi:files.write_documents", "写出文档文件", filesWriteDocumentsDesc, nil),
				i18nPacks("写出文档文件", "Write Document File", "文件", "File", filesWriteDocumentsDesc, filesWriteDocumentsDescEN, filesWriteDocumentsInputUIFields(), nil),
			),
			Handler: writeDocs.Handle,
		},
		{Metadata: workItemMetadata("Resolve catalog numeric ID by catalog name in current workspace.", false, schemaCatalogResolveInput(), schemaCatalogResolveOutput(), &mowl.WorkItemMetadata{NodeId: "moi:catalogs.resolve", DisplayName: "moi:catalogs.resolve", Category: "moi", Visibility: "internal", Summary: "Resolve catalog.", SideEffectClass: "read_only", Idempotence: "idempotent", Tags: []string{"internal"}}, workItemUISchema("input", "moi:catalogs.resolve", "moi:catalogs.resolve", "Resolve catalog.", nil), workItemUISchema("output", "moi:catalogs.resolve", "moi:catalogs.resolve", "Resolve catalog.", nil), nil), Handler: catalogResolve.Handle},
		{Metadata: workItemMetadata("Resolve database numeric ID by (`catalog_id`, `name`).", false, schemaDatabaseResolveInput(), schemaDatabaseResolveOutput(), &mowl.WorkItemMetadata{NodeId: "moi:databases.resolve", DisplayName: "moi:databases.resolve", Category: "moi", Visibility: "internal", Summary: "Resolve database.", SideEffectClass: "read_only", Idempotence: "idempotent", Tags: []string{"internal"}}, workItemUISchema("input", "moi:databases.resolve", "moi:databases.resolve", "Resolve database.", nil), workItemUISchema("output", "moi:databases.resolve", "moi:databases.resolve", "Resolve database.", nil), nil), Handler: databaseResolve.Handle},
		{Metadata: workItemMetadata("Resolve volume numeric ID by (`database_id`, `name`).", false, schemaVolumeResolveInput(), schemaVolumeResolveOutput(), &mowl.WorkItemMetadata{NodeId: "moi:volumes.resolve", DisplayName: "moi:volumes.resolve", Category: "moi", Visibility: "internal", Summary: "Resolve volume.", SideEffectClass: "read_only", Idempotence: "idempotent", Tags: []string{"internal"}}, workItemUISchema("input", "moi:volumes.resolve", "moi:volumes.resolve", "Resolve volume.", nil), workItemUISchema("output", "moi:volumes.resolve", "moi:volumes.resolve", "Resolve volume.", nil), nil), Handler: volumeResolve.Handle},
		{Metadata: workItemMetadata("Ensure a volume exists by (`database_id`, `name`).", false, schemaVolumeEnsureInput(), schemaVolumeEnsureOutput(), &mowl.WorkItemMetadata{NodeId: "moi:volumes.ensure", DisplayName: "moi:volumes.ensure", Category: "moi", Visibility: "internal", Summary: "Ensure volume.", SideEffectClass: "writes_state", Idempotence: "idempotent", Tags: []string{"internal"}}, workItemUISchema("input", "moi:volumes.ensure", "moi:volumes.ensure", "Ensure volume.", nil), workItemUISchema("output", "moi:volumes.ensure", "moi:volumes.ensure", "Ensure volume.", nil), nil), Handler: ensureVolume.Handle},
		{Metadata: workItemMetadata("Add one or multiple file IDs into a volume binding.", false, schemaVolumeFilesAddInput(), schemaVolumeFilesAddOutput(), &mowl.WorkItemMetadata{NodeId: "moi:volumes.files.add", DisplayName: "moi:volumes.files.add", Category: "moi", Visibility: "internal", Summary: "Add files to volume.", SideEffectClass: "writes_state", Idempotence: "idempotent", Tags: []string{"internal"}}, workItemUISchema("input", "moi:volumes.files.add", "moi:volumes.files.add", "Add files to volume.", nil), workItemUISchema("output", "moi:volumes.files.add", "moi:volumes.files.add", "Add files to volume.", nil), nil), Handler: addToVolume.Handle},
		{Metadata: workItemMetadata("List files currently associated with a volume.", false, schemaVolumeFilesListInput(), schemaVolumeFilesListOutput(), &mowl.WorkItemMetadata{NodeId: "moi:volumes.files.list", DisplayName: "moi:volumes.files.list", Category: "moi", Visibility: "internal", Summary: "List volume files.", SideEffectClass: "read_only", Idempotence: "idempotent", Tags: []string{"internal"}}, workItemUISchema("input", "moi:volumes.files.list", "moi:volumes.files.list", "List volume files.", nil), workItemUISchema("output", "moi:volumes.files.list", "moi:volumes.files.list", "List volume files.", nil), nil), Handler: listVolumeFiles.Handle},
		{Metadata: workItemMetadata("Move selected file bindings from source volume to target volume.", false, schemaVolumeFilesMoveInput(), schemaVolumeFilesMoveOutput(), &mowl.WorkItemMetadata{NodeId: "moi:volumes.files.move", DisplayName: "moi:volumes.files.move", Category: "moi", Visibility: "internal", Summary: "Move volume files.", SideEffectClass: "writes_state", Idempotence: "non_idempotent_or_requires_key", Tags: []string{"internal"}}, workItemUISchema("input", "moi:volumes.files.move", "moi:volumes.files.move", "Move volume files.", nil), workItemUISchema("output", "moi:volumes.files.move", "moi:volumes.files.move", "Move volume files.", nil), nil), Handler: moveVolumeFiles.Handle},
		{Metadata: workItemMetadata("Remove selected file bindings from a volume.", false, schemaVolumeFilesRemoveInput(), schemaVolumeFilesRemoveOutput(), &mowl.WorkItemMetadata{NodeId: "moi:volumes.files.remove", DisplayName: "moi:volumes.files.remove", Category: "moi", Visibility: "internal", Summary: "Remove volume files.", SideEffectClass: "writes_state", Idempotence: "non_idempotent_or_requires_key", Tags: []string{"internal"}}, workItemUISchema("input", "moi:volumes.files.remove", "moi:volumes.files.remove", "Remove volume files.", nil), workItemUISchema("output", "moi:volumes.files.remove", "moi:volumes.files.remove", "Remove volume files.", nil), nil), Handler: removeVolumeFiles.Handle},
		{Metadata: workItemMetadata("Export CDH table data to Parquet files via S3.", false, schemaCDHExportS3Input(), schemaCDHExportS3Output(), &mowl.WorkItemMetadata{NodeId: "moi:connector.cdh-s3", DisplayName: "moi:connector.cdh-s3", Category: "moi", Visibility: "internal", Summary: "Export CDH to S3.", SideEffectClass: "external_io", Idempotence: "non_idempotent_or_requires_key", Tags: []string{"internal"}}, workItemUISchema("input", "moi:connector.cdh-s3", "moi:connector.cdh-s3", "Export CDH to S3.", nil), workItemUISchema("output", "moi:connector.cdh-s3", "moi:connector.cdh-s3", "Export CDH to S3.", nil), nil), Handler: cdhExportS3.Handle},
		{Metadata: workItemMetadata(s3ToMOImportDesc, false, schemaS3ToMOImportInput(), schemaS3ToMOImportOutput(), &mowl.WorkItemMetadata{NodeId: "moi:connector.s3-mo", DisplayName: "S3 导入 MatrixOne", Category: "moi", Visibility: "user", DisplayGroup: "结构化数据", DisplayOrder: 240, Summary: s3ToMOImportDesc, SideEffectClass: "external_io", Idempotence: "non_idempotent_or_requires_key", Tags: []string{"connector", "s3", "mo"}}, workItemUISchema("input", "moi:connector.s3-mo", "S3 导入 MatrixOne", s3ToMOImportDesc, s3ToMOImportInputFields), workItemUISchema("output", "moi:connector.s3-mo", "S3 导入 MatrixOne", s3ToMOImportDesc, s3ToMOImportOutputFields), i18nPacks("S3 导入 MatrixOne", "Import S3 to MatrixOne", "结构化数据", "Structured Data", s3ToMOImportDesc, s3ToMOImportDescEN, s3ToMOImportInputFields, s3ToMOImportOutputFields)), Handler: s3ToMOImport.Handle},
		{Metadata: workItemMetadata("Download file, detect MIME type, resolve parser version, load config.", false, schemaParserIntakeInput(), schemaParserIntakeOutput(), &mowl.WorkItemMetadata{NodeId: parser.WIIntake, DisplayName: parser.WIIntake, Category: "moi", Visibility: "internal", Summary: "Parser intake.", SideEffectClass: "read_only", Idempotence: "idempotent", Tags: []string{"internal"}}, workItemUISchema("input", parser.WIIntake, parser.WIIntake, "Parser intake.", nil), workItemUISchema("output", parser.WIIntake, parser.WIIntake, "Parser intake.", nil), nil), Handler: parser.WrapIntake(factory, cfg.Parser.ToVersionRouter())},
		{Metadata: workItemMetadata("Convert Office documents to PDF.", false, schemaParserConvertInput(), schemaParserConvertOutput(), &mowl.WorkItemMetadata{NodeId: parser.WIConvert, DisplayName: parser.WIConvert, Category: "moi", Visibility: "internal", Summary: "Parser convert.", SideEffectClass: "read_only", Idempotence: "idempotent", Tags: []string{"internal"}}, workItemUISchema("input", parser.WIConvert, parser.WIConvert, "Parser convert.", nil), workItemUISchema("output", parser.WIConvert, parser.WIConvert, "Parser convert.", nil), nil), Handler: parser.WrapConvert(factory)},
		{Metadata: workItemMetadata("Paddle table detection and PDF whitening.", false, schemaParserPreprocessInput(), schemaParserPreprocessOutput(), &mowl.WorkItemMetadata{NodeId: parser.WIPreprocess, DisplayName: parser.WIPreprocess, Category: "moi", Visibility: "internal", Summary: "Parser preprocess.", SideEffectClass: "read_only", Idempotence: "idempotent", Tags: []string{"internal"}}, workItemUISchema("input", parser.WIPreprocess, parser.WIPreprocess, "Parser preprocess.", nil), workItemUISchema("output", parser.WIPreprocess, parser.WIPreprocess, "Parser preprocess.", nil), nil), Handler: parser.WrapPreprocess(factory)},
		{Metadata: workItemMetadata("MinerU layout extraction.", false, schemaParserLayoutInput(), schemaParserLayoutOutput(), &mowl.WorkItemMetadata{NodeId: parser.WILayout, DisplayName: parser.WILayout, Category: "moi", Visibility: "internal", Summary: "Parser layout.", SideEffectClass: "read_only", Idempotence: "idempotent", Tags: []string{"internal"}}, workItemUISchema("input", parser.WILayout, parser.WILayout, "Parser layout.", nil), workItemUISchema("output", parser.WILayout, parser.WILayout, "Parser layout.", nil), nil), Handler: parser.WrapLayout(factory)},
		{Metadata: workItemMetadata("Create blocks from layout.", false, schemaParserStructureInput(), schemaParserStructureOutput(), &mowl.WorkItemMetadata{NodeId: parser.WIStructure, DisplayName: parser.WIStructure, Category: "moi", Visibility: "internal", Summary: "Parser structure.", SideEffectClass: "read_only", Idempotence: "idempotent", Tags: []string{"internal"}}, workItemUISchema("input", parser.WIStructure, parser.WIStructure, "Parser structure.", nil), workItemUISchema("output", parser.WIStructure, parser.WIStructure, "Parser structure.", nil), nil), Handler: parser.WrapStructure(factory, clientOpts...)},
		{Metadata: workItemMetadata("Process TEXT/TITLE/LIST/CODE/EQUATION blocks.", false, schemaParserEnrichInput(), schemaParserEnrichOutput(), &mowl.WorkItemMetadata{NodeId: parser.WIEnrichContent, DisplayName: parser.WIEnrichContent, Category: "moi", Visibility: "internal", Summary: "Parser enrich content.", SideEffectClass: "read_only", Idempotence: "idempotent", Tags: []string{"internal"}}, workItemUISchema("input", parser.WIEnrichContent, parser.WIEnrichContent, "Parser enrich content.", nil), workItemUISchema("output", parser.WIEnrichContent, parser.WIEnrichContent, "Parser enrich content.", nil), nil), Handler: parser.WrapEnrichContent(factory, clientOpts...)},
		{Metadata: workItemMetadata("Process TABLE blocks.", false, schemaParserEnrichInput(), schemaParserEnrichOutput(), &mowl.WorkItemMetadata{NodeId: parser.WIEnrichTable, DisplayName: parser.WIEnrichTable, Category: "moi", Visibility: "internal", Summary: "Parser enrich table.", SideEffectClass: "read_only", Idempotence: "idempotent", Tags: []string{"internal"}}, workItemUISchema("input", parser.WIEnrichTable, parser.WIEnrichTable, "Parser enrich table.", nil), workItemUISchema("output", parser.WIEnrichTable, parser.WIEnrichTable, "Parser enrich table.", nil), nil), Handler: parser.WrapEnrichTable(factory, clientOpts...)},
		{Metadata: workItemMetadata("Process IMAGE blocks.", false, schemaParserEnrichInput(), schemaParserEnrichOutput(), &mowl.WorkItemMetadata{NodeId: parser.WIEnrichImage, DisplayName: parser.WIEnrichImage, Category: "moi", Visibility: "internal", Summary: "Parser enrich image.", SideEffectClass: "read_only", Idempotence: "idempotent", Tags: []string{"internal"}}, workItemUISchema("input", parser.WIEnrichImage, parser.WIEnrichImage, "Parser enrich image.", nil), workItemUISchema("output", parser.WIEnrichImage, parser.WIEnrichImage, "Parser enrich image.", nil), nil), Handler: parser.WrapEnrichImage(factory, clientOpts...)},
		{Metadata: workItemMetadata("Merge enriched blocks, sort, assemble final Document[].", false, schemaParserAssembleInput(), schemaParserAssembleOutput(), &mowl.WorkItemMetadata{NodeId: parser.WIAssemble, DisplayName: parser.WIAssemble, Category: "moi", Visibility: "internal", Summary: "Parser assemble.", SideEffectClass: "read_only", Idempotence: "idempotent", Tags: []string{"internal"}}, workItemUISchema("input", parser.WIAssemble, parser.WIAssemble, "Parser assemble.", nil), workItemUISchema("output", parser.WIAssemble, parser.WIAssemble, "Parser assemble.", nil), nil), Handler: parser.WrapAssemble(factory)},
	}
}

func mediaRichRegistration(
	nodeID, displayName, displayNameEN, description, descriptionEN, kind string,
	order int32,
	tags []string,
	factory *runtime.ClientFactory,
	clientOpts []clients.Option,
	versionRouter *parser.VersionRouter,
	parserQueues *workitems.ParserAPIQueues,
) WorkItemRegistration {
	inputFields := mediaParseInputUIFields(kind, nodeID)
	return WorkItemRegistration{
		Metadata: workItemMetadata(
			description,
			false,
			schemaMediaParseUserInput(kind),
			schemaMediaParseUserOutput(kind),
			&mowl.WorkItemMetadata{
				NodeId:                nodeID,
				DisplayName:           displayName,
				Category:              "moi",
				Visibility:            "user",
				DisplayGroup:          "文档处理",
				DisplayOrder:          order,
				NodeRole:              "transform",
				Tags:                  tags,
				Summary:               description,
				SideEffectClass:       "external_io",
				Idempotence:           "idempotent",
				RuntimeConfigContract: mediaParseRuntimeConfig(),
			},
			workItemUISchema("input", nodeID, displayName, description, inputFields),
			workItemUISchema("output", nodeID, displayName, description, mediaParseOutputUIFields(kind)),
			i18nPacks(displayName, displayNameEN, "文档处理", "Document Processing", description, descriptionEN, inputFields, mediaParseOutputUIFields(kind)),
		),
		Handler: (&workitems.RichConverter{
			Factory:       factory,
			ClientOpts:    clientOpts,
			Kind:          kind,
			VersionRouter: versionRouter,
			ParserQueues:  parserQueues,
		}).Handle,
	}
}

func catalogI18nPacksFromMetadata(packs map[int32]string) map[string]interface{} {
	if len(packs) == 0 {
		return nil
	}
	out := make(map[string]interface{}, len(packs))
	for localeValue, raw := range packs {
		locale := commonLanguageName(localeValue)
		var obj interface{}
		if err := json.Unmarshal([]byte(raw), &obj); err != nil {
			out[locale] = raw
			continue
		}
		out[locale] = obj
	}
	return out
}

func catalogI18nDefaultLocaleFromMetadata(locale commonpb.Language) string {
	return commonLanguageName(int32(locale))
}

func commonLanguageName(value int32) string {
	if value == 0 {
		return ""
	}
	return commonpb.Language(value).String()
}

// parseStageRuntimeLegacyMetadata registers the pre-rename NodeId
// "parse:stage_runtime" as an internal alias for moi:parse, reusing the same
// DocumentParse{ForceParserVersion:"v3"} handler. The alias stays internal while
// moi:parse is user-visible. This avoids NOT_FOUND for in-flight branches / saved
// workflow DSLs that still reference the old ID.
func parseStageRuntimeLegacyMetadata(description string) *mowl.WorkItemMetadata {
	nodeID := "parse:stage_runtime"
	inputUIFields := documentParseInputUIFields(parseStageRuntimeConfig(), nodeID)
	return workItemMetadata(
		description,
		false,
		schemaParseStageRuntimeInput(),
		schemaParseStageRuntimeOutput(),
		&mowl.WorkItemMetadata{
			NodeId:                nodeID,
			DisplayName:           nodeID,
			Category:              "moi",
			Visibility:            "internal",
			Summary:               "Legacy internal alias for moi:parse. Use moi:parse for new workflows.",
			SideEffectClass:       "read_only",
			Idempotence:           "idempotent",
			Tags:                  []string{"internal", "legacy", "parse", "v3", "stage"},
			RuntimeConfigContract: parseStageRuntimeConfig(),
		},
		workItemUISchema("input", nodeID, nodeID, description, inputUIFields),
		workItemUISchema("output", nodeID, nodeID, description, nil),
		nil,
	)
}

func documentVisualMetadata(nodeID, description, descriptionEN, displayName, displayNameEN, group, groupEN string, order int32, role, sideEffect string, tags []string, inputSchema, outputSchema *moi.SchemaBuilder, runtimeConfig []*mowl.RuntimeConfigParam, inputUIFields []*mowl.WorkItemUIField) *mowl.WorkItemMetadata {
	return workItemMetadata(
		description,
		false,
		inputSchema,
		outputSchema,
		&mowl.WorkItemMetadata{
			NodeId:                nodeID,
			DisplayName:           displayName,
			Category:              "moi",
			Visibility:            "user",
			DisplayGroup:          group,
			DisplayOrder:          order,
			NodeRole:              role,
			Summary:               description,
			SideEffectClass:       sideEffect,
			Idempotence:           "idempotent",
			Tags:                  tags,
			RuntimeConfigContract: runtimeConfig,
		},
		workItemUISchema("input", nodeID, displayName, description, inputUIFields),
		workItemUISchema("output", nodeID, displayName, description, nil),
		i18nPacks(displayName, displayNameEN, group, groupEN, description, descriptionEN, inputUIFields, nil),
	)
}

func documentVisualInternalMetadata(md *mowl.WorkItemMetadata) *mowl.WorkItemMetadata {
	md.Visibility = "internal"
	for _, tag := range md.Tags {
		if tag == "internal" {
			return md
		}
	}
	md.Tags = append(md.Tags, "internal")
	return md
}

func MarshalCatalog(registrations []WorkItemRegistration) ([]byte, error) {
	items := make([]CatalogItem, 0, len(registrations))
	for _, registration := range registrations {
		md := registration.Metadata
		if md == nil {
			return nil, fmt.Errorf("workitem metadata is required")
		}
		nodeID, err := metadataNodeID(md)
		if err != nil {
			return nil, err
		}
		inRaw := strings.TrimSpace(md.GetInputSchema())
		outRaw := strings.TrimSpace(md.GetOutputSchema())
		var inObj interface{}
		var outObj interface{}
		if err := json.Unmarshal([]byte(inRaw), &inObj); err != nil {
			return nil, fmt.Errorf("parse input schema for %s: %w", nodeID, err)
		}
		if err := json.Unmarshal([]byte(outRaw), &outObj); err != nil {
			return nil, fmt.Errorf("parse output schema for %s: %w", nodeID, err)
		}
		items = append(items, CatalogItem{
			Name:              nodeID,
			Description:       md.GetDescription(),
			Version:           md.GetVersion(),
			Isolation:         md.GetIsolationLevel().String(),
			Stream:            md.GetStream(),
			InputSchema:       inObj,
			OutputSchema:      outObj,
			Metadata:          md,
			InputUISchema:     md.GetInputUiSchema(),
			OutputUISchema:    md.GetOutputUiSchema(),
			I18n:              catalogI18nPacksFromMetadata(md.GetI18NPacks()),
			I18NDefaultLocale: catalogI18nDefaultLocaleFromMetadata(md.GetI18NDefaultLocale()),
		})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	out := Catalog{Worker: "go-worker", Count: len(items), Items: items}
	return json.MarshalIndent(out, "", "  ")
}
