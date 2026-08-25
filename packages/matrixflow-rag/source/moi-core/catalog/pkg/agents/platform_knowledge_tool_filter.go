package agents

import (
	"context"
	"crypto/sha256"
	"errors"
	"strings"
	"time"

	agenttools "github.com/matrixflow/moi-core/agent-tools"
	"github.com/matrixflow/moi-core/agent-tools/knowledge"
	"github.com/matrixflow/moi-core/catalog/pkg/agentautomation"
	"github.com/matrixflow/moi-core/catalog/pkg/agentruntime"
)

type PlatformKnowledgeToolFilter struct {
	resolver *PlatformKnowledgeScopeResolver
	now      func() time.Time
}

const platformKnowledgeToolRuntimeScopeUnavailableReason = "tool_runtime_scope_unavailable"

// platformKnowledgeRuntimeTimezone matches the product default used by agent
// automation execution time. Interactive knowledge turns have no cron TZ, so
// relative dates such as last year resolve against Asia/Shanghai.
const platformKnowledgeRuntimeTimezone = "Asia/Shanghai"

const platformKnowledgeRelativeDateRuntimeGuidance = "" +
	"Resolve relative date expressions such as last year, this year, last month, and yesterday against this current_date unless the user gave an explicit date or year. " +
	"Do not infer wall-clock time from business table freshness (for example MAX(year) or MAX(created_at)). " +
	"Zero rows for the resolved period is a valid result; do not relabel a different year as last year. " +
	"Do not use SELECT NOW() to interpret the user's relative dates."

// platformKnowledgeAutomationSourceType matches agentautomation.TaskSourceTypeAgentAutomationTask.
// Automation turns already carry an immutable execution_local_date; interactive
// turns omit source_type and fall back to the wall-clock below.
const platformKnowledgeAutomationSourceType = "agent_automation_task"

func NewPlatformKnowledgeToolFilter(resolver *PlatformKnowledgeScopeResolver) *PlatformKnowledgeToolFilter {
	return &PlatformKnowledgeToolFilter{resolver: resolver}
}

func (f *PlatformKnowledgeToolFilter) FilterTools(ctx context.Context, req agentruntime.RuntimeToolFilterRequest) ([]agentruntime.AgentTool, error) {
	if !platformKnowledgeHasToolSnapshot(req.Instance.Tools) {
		return req.Instance.Tools, nil
	}
	scope, err := f.runtimeScope(ctx, req.Scope, req.Instance, req.Metadata)
	if err != nil {
		return nil, err
	}
	capabilities := platformKnowledgeScopeToolCapabilities(scope)
	platformKnowledgeAnnotateSQLScopeUnavailable(&req.Instance, capabilities)
	return platformKnowledgeFilterTools(req.Instance, capabilities), nil
}

func platformKnowledgeFilterTools(instance agentruntime.AgentInstance, capabilities platformKnowledgeToolCapabilities) []agentruntime.AgentTool {
	strictToolSet := platformKnowledgeExploreStrictToolSet(instance)
	filtered := make([]agentruntime.AgentTool, 0, len(instance.Tools))
	for _, tool := range instance.Tools {
		if !platformKnowledgeToolSnapshot(tool) {
			if !strictToolSet || platformKnowledgeNonKnowledgeToolAllowed(tool.Kind) {
				filtered = append(filtered, tool)
			}
			continue
		}
		if platformKnowledgeToolAllowed(tool.Kind, capabilities) {
			filtered = append(filtered, tool)
		}
	}
	return filtered
}

func platformKnowledgeAnnotateSQLScopeUnavailable(instance *agentruntime.AgentInstance, capabilities platformKnowledgeToolCapabilities) {
	if instance == nil || capabilities.sql {
		return
	}
	var unavailable []string
	for _, tool := range instance.Tools {
		switch tool.Kind {
		case agenttools.ToolKindDescribeSchema, agenttools.ToolKindQuerySQL:
			unavailable = append(unavailable, tool.ID)
		}
	}
	if len(unavailable) == 0 || instance.PolicySummary == nil {
		return
	}
	instance.PolicySummary["runtime_unavailable_reason"] = platformKnowledgeToolRuntimeScopeUnavailableReason
	instance.PolicySummary["runtime_unavailable_tools"] = unavailable
	instance.PolicySummary["runtime_unavailable_message"] = "SQL tools require an accessible data source with database and table scope."
}

func (f *PlatformKnowledgeToolFilter) FilterInstruction(ctx context.Context, req agentruntime.RuntimeInstructionFilterRequest) (agentruntime.AgentInstruction, error) {
	instruction := req.Instance.Instruction
	if !platformKnowledgeHasToolSnapshot(req.Instance.Tools) {
		return instruction, nil
	}
	if !platformKnowledgeExploreUsesDefaultPrompt(instruction.SystemPrompt) {
		return instruction, nil
	}
	if f == nil || f.resolver == nil {
		return instruction, errors.New("platform knowledge tool filter resolver is not configured")
	}
	scope, err := f.runtimeScope(ctx, req.Scope, req.Instance, req.Metadata)
	if err != nil {
		return instruction, err
	}
	filtered := platformKnowledgeFilterTools(req.Instance, platformKnowledgeScopeToolCapabilities(scope))
	instruction.SystemPrompt = platformKnowledgeExploreSystemPromptForTools(filtered, scope)
	return instruction, nil
}

func (f *PlatformKnowledgeToolFilter) BuildRuntimeSystemPrompt(ctx context.Context, req agentruntime.RuntimeSystemPromptRequest) (string, error) {
	if !platformKnowledgeHasToolSnapshot(req.Instance.Tools) {
		return "", nil
	}
	scope, err := f.runtimeScope(ctx, req.Scope, req.Instance, req.Metadata)
	if err != nil {
		return "", err
	}
	clock, err := f.knowledgeRuntimeClock(req.Metadata)
	if err != nil {
		return "", err
	}
	return platformKnowledgeRuntimeSystemPromptForTools(req.Instance.Tools, scope, clock), nil
}

type knowledgeRuntimeClock struct {
	CurrentDate string
	Timezone    string
}

func (f *PlatformKnowledgeToolFilter) knowledgeRuntimeClock(metadata map[string]any) (knowledgeRuntimeClock, error) {
	if clock, ok, err := knowledgeRuntimeClockFromAutomationMetadata(metadata); err != nil || ok {
		return clock, err
	}
	return f.interactiveRuntimeClock()
}

func knowledgeRuntimeClockFromAutomationMetadata(metadata map[string]any) (knowledgeRuntimeClock, bool, error) {
	if metadataString(metadata, "source_type") != platformKnowledgeAutomationSourceType {
		return knowledgeRuntimeClock{}, false, nil
	}
	date := metadataString(metadata, "execution_local_date")
	if date == "" {
		return knowledgeRuntimeClock{}, false, errors.New("automation execution_local_date is required")
	}
	if _, err := time.Parse("2006-01-02", date); err != nil {
		return knowledgeRuntimeClock{}, false, errors.New("automation execution_local_date must be YYYY-MM-DD")
	}
	timezone := metadataString(metadata, "execution_timezone")
	if timezone == "" {
		return knowledgeRuntimeClock{}, false, errors.New("automation execution_timezone is required")
	}
	if _, err := agentautomation.LoadAutomationLocation(timezone); err != nil {
		return knowledgeRuntimeClock{}, false, errors.New("automation execution_timezone is invalid")
	}
	return knowledgeRuntimeClock{CurrentDate: date, Timezone: timezone}, true, nil
}

func (f *PlatformKnowledgeToolFilter) interactiveRuntimeClock() (knowledgeRuntimeClock, error) {
	now := time.Now()
	if f != nil && f.now != nil {
		now = f.now()
	}
	loc, err := time.LoadLocation(platformKnowledgeRuntimeTimezone)
	if err != nil {
		return knowledgeRuntimeClock{}, err
	}
	return knowledgeRuntimeClock{
		CurrentDate: now.In(loc).Format("2006-01-02"),
		Timezone:    platformKnowledgeRuntimeTimezone,
	}, nil
}

func metadataString(metadata map[string]any, key string) string {
	if metadata == nil {
		return ""
	}
	value, ok := metadata[key]
	if !ok || value == nil {
		return ""
	}
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(text)
}

func (f *PlatformKnowledgeToolFilter) runtimeScope(ctx context.Context, scope agentruntime.RuntimeRequestScope, descriptor agentruntime.AgentInstance, metadata map[string]any) (knowledge.WorkspaceScope, error) {
	if f == nil || f.resolver == nil {
		return knowledge.WorkspaceScope{}, errors.New("platform knowledge tool filter resolver is not configured")
	}
	return knowledge.RuntimeToolScope(ctx, knowledge.RuntimeToolRequest{
		WorkspaceID: scope.WorkspaceID,
		Manifest: knowledge.RuntimeManifest{
			WorkspaceID: scope.WorkspaceID,
			Body: map[string]any{
				"knowledge_bases": platformKnowledgeFilterKnowledgeManifest(descriptor.KnowledgeBases),
			},
		},
		TurnMetadata: metadata,
		RequestScope: knowledge.RuntimeRequestScope{
			WorkspaceID: scope.WorkspaceID,
			UserID:      scope.UserID,
		},
	}, f.resolver)
}

type platformKnowledgeToolCapabilities struct {
	sql                  bool
	rag                  bool
	visual               bool
	findRAGFiles         bool
	searchRAGChunks      bool
	readParsedMarkdown   bool
	searchParsedMarkdown bool
}

func platformKnowledgeScopeToolCapabilities(scope knowledge.WorkspaceScope) platformKnowledgeToolCapabilities {
	// SQL is available when the scope has at least one queryable table identity.
	// Table identity is database.table (or bare only when Scope.DBName supplies
	// the database for legacy scopes).
	return platformKnowledgeToolCapabilities{
		sql:    platformKnowledgeScopeHasSQLTables(scope),
		rag:    platformKnowledgeScopeHasTextRAG(scope),
		visual: platformKnowledgeScopeHasVisualIndex(scope),
	}
}

func platformKnowledgeScopeHasSQLTables(scope knowledge.WorkspaceScope) bool {
	tables := platformKnowledgeCompactStrings(scope.Tables)
	if len(tables) == 0 {
		return false
	}
	defaultDB := strings.TrimSpace(scope.DBName)
	// Addressable when every table has an explicit database (defaultDB or a
	// multi-db known prefix). Do not treat "has a dot" as qualified.
	knownDBs := platformKnowledgeScopeKnownDatabaseNames(tables, defaultDB)
	for _, table := range tables {
		schema, name := platformKnowledgeParseTableIdentity(table, defaultDB, knownDBs...)
		if name == "" {
			return false
		}
		if schema == "" && defaultDB == "" {
			// Bare table with no database cannot be addressed as database.table.
			return false
		}
	}
	return true
}

// platformKnowledgeScopeLeftDatabaseSegments returns distinct first segments of
// dotted labels whose left part has no '.' (candidate database names).
func platformKnowledgeScopeLeftDatabaseSegments(values []string) []string {
	lefts := make([]string, 0, len(values))
	leftSeen := make(map[string]struct{}, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		trimmed = strings.Trim(trimmed, "`\"")
		if trimmed == "" {
			continue
		}
		idx := strings.Index(trimmed, ".")
		if idx <= 0 || idx >= len(trimmed)-1 {
			continue
		}
		left := strings.TrimSpace(trimmed[:idx])
		if left == "" || strings.Contains(left, ".") {
			continue
		}
		key := strings.ToLower(left)
		if _, ok := leftSeen[key]; ok {
			continue
		}
		leftSeen[key] = struct{}{}
		lefts = append(lefts, left)
	}
	return lefts
}

// platformKnowledgeScopeKnownDatabaseNames mirrors agent-tools known-DB
// collection: defaultDB plus multi-db left segments only when 2+ distinct
// databases appear (never self-bootstrap a lone bare xxx.xxx into a DB).
func platformKnowledgeScopeKnownDatabaseNames(values []string, defaultDB string) []string {
	out := make([]string, 0, len(values)+1)
	seen := make(map[string]struct{}, len(values)+1)
	add := func(db string) {
		db = strings.TrimSpace(db)
		if db == "" || strings.Contains(db, ".") {
			return
		}
		key := strings.ToLower(db)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		out = append(out, db)
	}
	add(defaultDB)
	lefts := platformKnowledgeScopeLeftDatabaseSegments(values)
	if len(lefts) >= 2 {
		for _, left := range lefts {
			add(left)
		}
	}
	return out
}

func platformKnowledgeScopeHasTextRAG(scope knowledge.WorkspaceScope) bool {
	if platformKnowledgeHasTextIndexBinding(scope.VectorTable, scope.EmbeddingModel) {
		return true
	}
	// Direct/legacy RAG scopes may provide only vector_table and rely on the
	// retrieval service default embedding model. Semantic KB sources stay
	// stricter because their queryable index must be declared by that KB.
	if strings.TrimSpace(scope.VectorTable) != "" && len(platformKnowledgeCompactInt64s(scope.SemanticModelIDs)) == 0 {
		return true
	}
	for _, source := range scope.RAGSources {
		if platformKnowledgeRAGSourceHasTextIndex(source) {
			return true
		}
		if strings.TrimSpace(source.VectorTable) != "" && !platformKnowledgeIsSemanticModelRAGSource(source) {
			return true
		}
	}
	return false
}

func platformKnowledgeIsSemanticModelRAGSource(source knowledge.RAGSource) bool {
	if strings.TrimSpace(source.Metadata["source"]) == "semantic_model" {
		return true
	}
	return source.SemanticModelID != 0
}

func platformKnowledgeScopeHasVisualIndex(scope knowledge.WorkspaceScope) bool {
	if platformKnowledgeHasCompleteVisualConfig(
		scope.ImageVectorTable,
		scope.ImageEmbeddingModel,
		scope.ImageEmbeddingDimension,
		scope.ImagePreprocessVersion,
		scope.ImageDistanceMetric,
	) {
		return true
	}
	for _, source := range scope.RAGSources {
		if platformKnowledgeHasCompleteVisualConfig(
			source.ImageVectorTable,
			source.ImageEmbeddingModel,
			source.ImageEmbeddingDimension,
			source.ImagePreprocessVersion,
			source.ImageDistanceMetric,
		) {
			return true
		}
	}
	return false
}

func platformKnowledgeHasCompleteVisualConfig(table, model string, dimension int, preprocessVersion, distanceMetric string) bool {
	return table != "" && model != "" && dimension > 0 && preprocessVersion != "" && distanceMetric != ""
}

func platformKnowledgeToolSnapshot(tool agentruntime.AgentTool) bool {
	return agenttools.IsKnowledgeToolKind(tool.Kind)
}

func platformKnowledgeHasToolSnapshot(tools []agentruntime.AgentTool) bool {
	for _, tool := range tools {
		if platformKnowledgeToolSnapshot(tool) {
			return true
		}
	}
	return false
}

func platformKnowledgeToolAllowed(kind string, capabilities platformKnowledgeToolCapabilities) bool {
	switch kind {
	case agenttools.ToolKindSelectFinalSources:
		return true
	case agenttools.ToolKindComputeResultTable:
		return false
	case agenttools.ToolKindDescribeSchema, agenttools.ToolKindQuerySQL:
		return capabilities.sql
	case agenttools.ToolKindFindRAGFiles, agenttools.ToolKindSearchRAGChunks, agenttools.ToolKindReadParsedMarkdown, agenttools.ToolKindSearchParsedMarkdown:
		return capabilities.rag
	case agenttools.ToolKindSearchVisualImage:
		return capabilities.visual
	default:
		return false
	}
}

func platformKnowledgeNonKnowledgeToolAllowed(kind string) bool {
	switch strings.TrimSpace(kind) {
	case agenttools.ToolKindReadArtifact,
		agenttools.ToolKindReadArtifactPage,
		agenttools.ToolKindWriteFile,
		agenttools.ToolKindReadFile:
		return true
	default:
		return false
	}
}

func platformKnowledgeExploreStrictToolSet(descriptor agentruntime.AgentInstance) bool {
	if strings.TrimSpace(descriptor.AgentID) == "explore" {
		return true
	}
	return strings.TrimSpace(descriptor.Instruction.SystemPrompt) != "" &&
		platformKnowledgeExploreUsesDefaultPrompt(descriptor.Instruction.SystemPrompt)
}

func platformKnowledgeExploreUsesDefaultPrompt(prompt string) bool {
	trimmed := strings.TrimSpace(prompt)
	if trimmed == "" {
		return true
	}
	digest := sha256.Sum256([]byte(trimmed))
	return digest == platformKnowledgeExploreLegacyChineseDefaultPromptDigest ||
		digest == platformKnowledgeExploreLegacyEnglishDefaultPromptDigest ||
		digest == platformKnowledgeExploreSQLOnlyChineseDefaultPromptDigest ||
		digest == platformKnowledgeExploreSQLOnlyEnglishDefaultPromptDigest
}

var (
	// d24d43a77f persisted this exact zh-CN Knowledge Explore default prompt.
	platformKnowledgeExploreLegacyChineseDefaultPromptDigest = [sha256.Size]byte{
		0x1e, 0xa8, 0x99, 0xf2, 0xda, 0x0e, 0xe7, 0x5f,
		0xbe, 0x3f, 0x08, 0xa8, 0x4f, 0xa4, 0xe0, 0xb0,
		0x1b, 0x34, 0xf8, 0x0c, 0x98, 0x47, 0xe1, 0xef,
		0x8d, 0xb5, 0xc9, 0x65, 0xee, 0x59, 0x93, 0x44,
	}
	// d24d43a77f persisted this exact en-US Knowledge Explore default prompt.
	platformKnowledgeExploreLegacyEnglishDefaultPromptDigest = [sha256.Size]byte{
		0x06, 0xef, 0x4e, 0x79, 0x63, 0x60, 0x30, 0x80,
		0xde, 0xd9, 0x9a, 0x15, 0x01, 0x33, 0x73, 0x90,
		0x25, 0xde, 0xfd, 0xda, 0xe9, 0xff, 0x92, 0xed,
		0x43, 0x17, 0x65, 0x7b, 0x9d, 0xb0, 0xee, 0xa3,
	}
	// Knowledge Explore identity of the CRT-removed zh-CN default prompt.
	platformKnowledgeExploreSQLOnlyChineseDefaultPromptDigest = [sha256.Size]byte{
		0x1d, 0x16, 0x55, 0xd7, 0x3b, 0x9e, 0x81, 0x55,
		0x84, 0xb4, 0xc6, 0x14, 0xd6, 0xde, 0xbb, 0x68,
		0x68, 0x7c, 0xde, 0x91, 0x3f, 0xa3, 0xd9, 0x9e,
		0x40, 0x52, 0x47, 0xc0, 0xfe, 0xf1, 0x4b, 0x64,
	}
	// Knowledge Explore identity of the CRT-removed en-US default prompt.
	platformKnowledgeExploreSQLOnlyEnglishDefaultPromptDigest = [sha256.Size]byte{
		0xb6, 0x65, 0x0e, 0x01, 0xff, 0x46, 0x61, 0xfe,
		0xdb, 0xe6, 0xda, 0xe4, 0x0e, 0xf2, 0x96, 0xdb,
		0xe3, 0xca, 0xfe, 0xaa, 0x3a, 0x2b, 0x83, 0x4b,
		0x75, 0xa9, 0x1a, 0xde, 0x41, 0x0e, 0xcf, 0xc7,
	}
)

func platformKnowledgeExploreSystemPromptForTools(tools []agentruntime.AgentTool, scope knowledge.WorkspaceScope) string {
	capabilities := platformKnowledgeToolCapabilitiesFromTools(tools)
	prompt := "" +
		"You are Knowledge Explore, a data-grounded agent for Matrixflow knowledge bases.\n\n" +
		"## Data source mode\n\n" +
		"Use only the tools registered in this agent descriptor. Do not invent tools, tables, columns, files, or semantic keys.\n\n"
	if capabilities.sql || capabilities.rag || capabilities.visual || scope.DBName != "" || len(scope.Tables) > 0 {
		prompt += "## Tenant scope (hard boundary)\n\n"
		prompt += "- scope database: "
		if scope.DBName != "" {
			prompt += scope.DBName
		} else if len(scope.Tables) > 0 {
			prompt += "(use database.table names from the SQL allowed tables list)"
		} else {
			prompt += "(unspecified)"
		}
		prompt += "\n"
		if len(scope.Tables) > 0 {
			prompt += "- SQL allowed tables (constrains SQL tools only; do NOT query outside this list): " + strings.Join(scope.Tables, ", ") + "\n"
			prompt += "- Reference tables as database.table using names from the SQL allowed tables list.\n"
		}
		if capabilities.rag {
			prompt += "- document corpus: available through registered document tools: " + platformKnowledgeToolList(capabilities.ragToolNames()) + "\n"
		}
		prompt += "\n"
	}
	if capabilities.sql && capabilities.rag {
		prompt += "When the selected knowledge scope contains mixed tables + files, unless the user explicitly restricts the task to one source family, you MUST query both source families before the final answer: SQL tools for structured/table facts and registered document tools for file evidence. A SQL-only final answer is forbidden in mixed mode. Do not stop after SQL just because a table query returned rows.\n\n"
	}
	if capabilities.sql {
		prompt += "" +
			"## SQL tools\n\n" +
			"- Schema: call `describe_schema` before generating SQL. If you have not already selected exact table names from the SQL allowed tables list, omit `table_names` or pass `[]` so the tool returns the selected scope. Only pass `table_names` when every name is copied from the SQL allowed tables list or a prior `describe_schema` result. Never infer table names from user wording.\n" +
			"- SQL table references must use `database.table` names from the SQL allowed tables list.\n" +
			"- Dialect: `query_sql` accepts only valid MySQL 8 syntax. Reserved words used as identifiers/aliases must be quoted with backticks or renamed to a non-reserved alias such as now_value. Do not emit MatrixOne-only syntax that MySQL rejects.\n" +
			"- Relative dates in the user question must be resolved against the current_date injected in MOI Runtime Scope. Do not treat the latest year present in a table as last year, and do not use SELECT NOW() to interpret those expressions.\n" +
			"- Pure scalar SELECT without FROM is allowed when you only need expressions such as SELECT NOW() AS now_value. Table queries must still reference in-scope physical tables.\n" +
			"- If a table is returned with `queryable=false` or `access=\"permission_denied\"`, do not call `query_sql` for that table; answer that this part of the request is unavailable within the current user's data access.\n" +
			"- Query: use whatever SQL strategy fits the question: a single comprehensive SELECT, staged exploration, or iterative refinement.\n" +
			"- For open-ended analysis that does not name a dimension or metric, do not answer that the dimension is missing before inspection. Use `describe_schema` and semantic context from its result to choose relevant in-scope dimensions and indicators, then run bounded SQL.\n" +
			"- Express unit conversions, ratios, growth rates, differences, shares, ranks, unions, and joins in SQL via `query_sql`. Prefer a new SELECT over reshaping previous result rows locally.\n" +
			"- When a SQL decision depends on returned `semantic_entries`, pass the exact used semantic keys to `query_sql.semantic_claims`.\n\n"
	}
	if capabilities.rag {
		prompt += "" +
			"## Document retrieval\n\n" +
			"- Document retrieval is open RAG exploration over files and evidence chunks. Use only the registered document tools for this agent: " + platformKnowledgeToolList(capabilities.ragToolNames()) + ". Decide what evidence is missing, fill only that gap, and do not browse the whole corpus just to preview it. Do not answer from memory.\n" +
			"- First understand the user's question. Extract the objects, time/version, source granularity, fields/indicators, and changes or comparisons that must be covered. Multi-object, multi-time, multi-source, or broad questions must not stop at the first few matching chunks.\n" +
			"- A zero-row document result means the current retrieval wording or source scope did not hit. Before concluding evidence is absent, adjust the evidence query, source selection, or field wording based on what previous tool outputs showed. Do not repeat the same empty call.\n"
		if capabilities.findRAGFiles && capabilities.searchRAGChunks {
			prompt += "" +
				"- Separate source discovery from evidence extraction. Before extracting values, first infer the required time coverage and source granularity from the user's wording, then choose candidate files whose source metadata matches that coverage and granularity.\n" +
				"- For source discovery, call `find_rag_files` with the object, time, and document type you inferred from the question.\n" +
				"- For evidence extraction, call `search_rag_chunks` with `keywords` that name the required object, time, source/table, and field meanings; pass relevant `file_ids` when you have already located candidate files. Do not pass the complete user question as one keyword.\n"
		} else if capabilities.searchRAGChunks {
			prompt += "- For evidence extraction, call `search_rag_chunks` with `keywords` that name the required object, time, source/table, and field meanings. Do not pass the complete user question as one keyword.\n"
		} else if capabilities.findRAGFiles {
			prompt += "- For source discovery, call `find_rag_files` with the object, time, and document type you inferred from the question. If no evidence-extraction tool is registered, report that limitation instead of inventing document contents.\n"
		}
		prompt += "\n"
	}
	if capabilities.visual {
		prompt += "" +
			"## Visual retrieval\n\n" +
			"- If the task asks about drawings, screenshots, visual objects, or image similarity, use `search_visual_image` for visual evidence. For text-to-drawing search, pass `query_text`; for image matching, pass `query_visual` using the 1-based image number of the current user message image input parts.\n\n"
	}
	if capabilities.sql && capabilities.rag {
		prompt += "## Mixed-source obligation\n\n" +
			"If you use SQL tools in mixed tables + files mode, you must also use the registered document tools before the final answer. SQL allowed tables do not describe or limit the document corpus; do not conclude that a topic is absent from files just because it is absent from table names. If the document search returns no relevant evidence, say that the file side was searched but had no supporting evidence; do not omit the file side silently.\n\n"
	}
	prompt += "" +
		"If the required tool is unavailable, or the selected knowledge scope does not contain the required source type, say which data or tool is missing and stop short of unsupported conclusions.\n\n" +
		"## Answering\n\n" +
		"Base every factual claim on tool results from this task. Keep the final answer concise and business-facing.\n\n" +
		"Use cite-then-write: call `select_final_sources` once after retrieval, then write the final user-facing answer and stop. Do not call submit_final_answer."
	if capabilities.rag {
		if capabilities.searchRAGChunks {
			prompt += " For RAG evidence, cite `rag_chunk` with `chunk_id` or `chunk_ids` copied from `search_rag_chunks`."
		} else {
			prompt += " For document evidence, cite only identifiers returned by registered document tools."
		}
	}
	if capabilities.visual {
		prompt += " For visual evidence, cite `visual_hit` with `object_id`, `image_file_id`, or `page_image_file_id` copied from `search_visual_image`."
	}
	if capabilities.sql {
		prompt += " For NL2SQL evidence, cite `sql_result` with the exact `artifact_id` returned by `query_sql`."
	}
	return prompt
}

func platformKnowledgeRuntimeSystemPromptForTools(tools []agentruntime.AgentTool, scope knowledge.WorkspaceScope, clock knowledgeRuntimeClock) string {
	capabilities := platformKnowledgeToolCapabilitiesFromTools(tools)
	if !capabilities.sql && scope.DBName == "" && len(scope.Tables) == 0 {
		return ""
	}
	prompt := "" +
		"## MOI Runtime Scope\n\n" +
		"This section is injected by MOI for the current turn. It contains runtime resource scope.\n\n"
	if clock.CurrentDate != "" || clock.Timezone != "" {
		prompt += "Current time (server-owned, authoritative):\n"
		if clock.CurrentDate != "" {
			prompt += "- current_date: " + clock.CurrentDate + "\n"
		}
		if clock.Timezone != "" {
			prompt += "- timezone: " + clock.Timezone + "\n"
		}
		prompt += "\n" + platformKnowledgeRelativeDateRuntimeGuidance + "\n\n"
	}
	if capabilities.sql || scope.DBName != "" || len(scope.Tables) > 0 {
		prompt += "Structured SQL scope:\n"
		if scope.DBName != "" {
			prompt += "- db_name: " + scope.DBName + "\n"
		} else if len(scope.Tables) > 0 {
			prompt += "- db_name: (use database.table names from table_names)\n"
		} else {
			prompt += "- db_name: (unavailable)\n"
		}
		if len(scope.Tables) > 0 {
			prompt += "- table_names: " + strings.Join(scope.Tables, ", ") + "\n"
		}
		prompt += "\n"
		if len(scope.Tables) > 0 {
			prompt += "Only query tables listed in `table_names`.\n"
			prompt += "Reference every SQL table as `database.table` using names from `table_names`.\n"
		}
	}
	return prompt
}

func platformKnowledgeToolCapabilitiesFromTools(tools []agentruntime.AgentTool) platformKnowledgeToolCapabilities {
	var capabilities platformKnowledgeToolCapabilities
	for _, tool := range tools {
		switch tool.Kind {
		case agenttools.ToolKindDescribeSchema, agenttools.ToolKindQuerySQL:
			capabilities.sql = true
		case agenttools.ToolKindFindRAGFiles:
			capabilities.rag = true
			capabilities.findRAGFiles = true
		case agenttools.ToolKindSearchRAGChunks:
			capabilities.rag = true
			capabilities.searchRAGChunks = true
		case agenttools.ToolKindReadParsedMarkdown:
			capabilities.rag = true
			capabilities.readParsedMarkdown = true
		case agenttools.ToolKindSearchParsedMarkdown:
			capabilities.rag = true
			capabilities.searchParsedMarkdown = true
		case agenttools.ToolKindSearchVisualImage:
			capabilities.visual = true
		}
	}
	return capabilities
}

func (c platformKnowledgeToolCapabilities) ragToolNames() []string {
	names := make([]string, 0, 4)
	if c.findRAGFiles {
		names = append(names, agenttools.ToolKindFindRAGFiles)
	}
	if c.searchRAGChunks {
		names = append(names, agenttools.ToolKindSearchRAGChunks)
	}
	if c.readParsedMarkdown {
		names = append(names, agenttools.ToolKindReadParsedMarkdown)
	}
	if c.searchParsedMarkdown {
		names = append(names, agenttools.ToolKindSearchParsedMarkdown)
	}
	return names
}

func platformKnowledgeToolList(names []string) string {
	if len(names) == 0 {
		return "(none)"
	}
	quoted := make([]string, 0, len(names))
	for _, name := range names {
		quoted = append(quoted, "`"+name+"`")
	}
	return strings.Join(quoted, ", ")
}

func platformKnowledgeFilterKnowledgeManifest(knowledgeSnapshots []agentruntime.AgentKnowledge) []map[string]any {
	if len(knowledgeSnapshots) == 0 {
		return nil
	}
	out := make([]map[string]any, 0, len(knowledgeSnapshots))
	for _, snapshot := range knowledgeSnapshots {
		item := map[string]any{
			"id":          snapshot.ID,
			"name":        snapshot.Name,
			"description": snapshot.Description,
			"metadata":    platformKnowledgeFilterCloneMap(snapshot.Metadata),
		}
		if len(snapshot.CatalogAssetRefs) > 0 {
			refs := make([]map[string]any, 0, len(snapshot.CatalogAssetRefs))
			for _, ref := range snapshot.CatalogAssetRefs {
				refs = append(refs, platformKnowledgeFilterCloneMap(ref))
			}
			item["catalog_asset_refs"] = refs
		}
		out = append(out, item)
	}
	return out
}

func platformKnowledgeFilterCloneMap(input map[string]any) map[string]any {
	if input == nil {
		return nil
	}
	out := make(map[string]any, len(input))
	for key, value := range input {
		out[key] = platformKnowledgeFilterCloneAny(value)
	}
	return out
}

func platformKnowledgeFilterCloneAny(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return platformKnowledgeFilterCloneMap(typed)
	case []any:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, platformKnowledgeFilterCloneAny(item))
		}
		return out
	case []map[string]any:
		out := make([]map[string]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, platformKnowledgeFilterCloneMap(item))
		}
		return out
	case []string:
		return append([]string(nil), typed...)
	default:
		return value
	}
}
