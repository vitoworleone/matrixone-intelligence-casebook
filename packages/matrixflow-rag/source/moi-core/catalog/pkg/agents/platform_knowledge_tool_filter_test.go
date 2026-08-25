package agents

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-sql-driver/mysql"

	agentruntimev2 "github.com/matrixflow/moi-core/agent-runtime-v2"
	agenttools "github.com/matrixflow/moi-core/agent-tools"
	"github.com/matrixflow/moi-core/agent-tools/knowledge"
	"github.com/matrixflow/moi-core/catalog/pkg/agentresource"
	"github.com/matrixflow/moi-core/catalog/pkg/agentruntime"
	"github.com/matrixflow/moi-core/catalog/pkg/agentruntime/a2a"
	"github.com/matrixflow/moi-core/catalog/pkg/service/storage/tenant"
	"github.com/matrixflow/moi-core/catalog/pkg/service/storage/transaction"
)

func TestPlatformKnowledgeToolFilterRAGOnlyScopeHidesSQLTools(t *testing.T) {
	files, err := json.Marshal(map[string]any{
		"vector_table":    "kb_resume_index",
		"embedding_model": "bge-m3",
		"file_ids":        []string{},
	})
	if err != nil {
		t.Fatalf("marshal files: %v", err)
	}
	filter := NewPlatformKnowledgeToolFilter(&PlatformKnowledgeScopeResolver{
		semantic: platformKnowledgeFilterTestSemanticStore{
			model: &tenant.SemanticModelRecord{
				ID:     2,
				Name:   "resume kb",
				Tables: json.RawMessage(`[]`),
				Files:  files,
			},
		},
	})
	tools, err := filter.FilterTools(context.Background(), agentruntime.RuntimeToolFilterRequest{
		Scope:    agentruntime.RuntimeRequestScope{WorkspaceID: "ws_1", AgentID: "explore", UserID: "user_1"},
		Instance: platformKnowledgeFilterTestDescriptor(t),
		Metadata: map[string]any{
			"semantic_model_ids": []any{float64(2)},
		},
	})
	if err != nil {
		t.Fatalf("FilterTools() error = %v", err)
	}
	if !platformKnowledgeFilterTestContains(tools, agenttools.ToolKindSearchRAGChunks) {
		t.Fatalf("RAG-only tools missing search_rag_chunks: %+v", tools)
	}
	if !platformKnowledgeFilterTestContains(tools, agenttools.ToolKindSelectFinalSources) {
		t.Fatalf("RAG-only tools missing select final sources tool: %+v", tools)
	}
	if platformKnowledgeFilterTestContains(tools, agenttools.ToolKindSubmitFinalAnswer) {
		t.Fatalf("RAG-only scope exposed legacy submit final answer tool: %+v", tools)
	}
	for _, hidden := range []string{agenttools.ToolKindDescribeSchema, agenttools.ToolKindQuerySQL, agenttools.ToolKindComputeResultTable} {
		if platformKnowledgeFilterTestContains(tools, hidden) {
			t.Fatalf("RAG-only scope exposed %s: %+v", hidden, tools)
		}
	}

	instruction, err := filter.FilterInstruction(context.Background(), agentruntime.RuntimeInstructionFilterRequest{
		Scope: agentruntime.RuntimeRequestScope{WorkspaceID: "ws_1", AgentID: "explore", UserID: "user_1"},
		Instance: agentruntime.AgentInstance{
			Tools: tools,
		},
		Metadata: map[string]any{
			"semantic_model_ids": []any{float64(2)},
		},
	})
	if err != nil {
		t.Fatalf("FilterInstruction() error = %v", err)
	}
	if !containsAll(instruction.SystemPrompt, "Document retrieval", "search_rag_chunks", "rag_chunk") {
		t.Fatalf("RAG-only prompt missing RAG guidance:\n%s", instruction.SystemPrompt)
	}
	if containsAny(instruction.SystemPrompt, "query_sql", "describe_schema", "NL2SQL evidence", "Mixed-source obligation") {
		t.Fatalf("RAG-only prompt contains SQL/mixed guidance:\n%s", instruction.SystemPrompt)
	}
}

// TestIssue13326Offset84BindingKeepsSearchRAGChunksVisible verifies the
// production semantic scope resolver and tool filter consume the exact files
// binding persisted by offset 84.
func TestIssue13326Offset84BindingKeepsSearchRAGChunksVisible(t *testing.T) {
	files := json.RawMessage(`{"file_ids":["legacy_file_1"],"vector_table":"kb_840000002_text_index","embedding_model":"bge-m3","preserved":"yes"}`)
	filter := NewPlatformKnowledgeToolFilter(&PlatformKnowledgeScopeResolver{
		semantic: platformKnowledgeFilterTestSemanticStore{
			model: &tenant.SemanticModelRecord{
				ID:     840000002,
				Name:   "offset84 migrated legacy model",
				Tables: json.RawMessage(`[]`),
				Files:  files,
			},
		},
	})
	tools, err := filter.FilterTools(context.Background(), agentruntime.RuntimeToolFilterRequest{
		Scope:    agentruntime.RuntimeRequestScope{WorkspaceID: "ws_1", AgentID: "explore", UserID: "user_1"},
		Instance: platformKnowledgeFilterTestDescriptor(t),
		Metadata: map[string]any{
			"semantic_model_ids": []any{float64(840000002)},
		},
	})
	if err != nil {
		t.Fatalf("FilterTools() error = %v", err)
	}
	if !platformKnowledgeFilterTestContains(tools, agenttools.ToolKindSearchRAGChunks) {
		t.Fatalf("offset84 migrated binding hides search_rag_chunks: %+v", tools)
	}
}

func TestPlatformKnowledgeToolFilterDefaultExploreDropsExtraChannelTools(t *testing.T) {
	files, err := json.Marshal(map[string]any{
		"vector_table":    "kb_resume_index",
		"embedding_model": "bge-m3",
		"file_ids":        []string{},
	})
	if err != nil {
		t.Fatalf("marshal files: %v", err)
	}
	filter := NewPlatformKnowledgeToolFilter(&PlatformKnowledgeScopeResolver{
		semantic: platformKnowledgeFilterTestSemanticStore{
			model: &tenant.SemanticModelRecord{
				ID:     2,
				Name:   "resume kb",
				Tables: json.RawMessage(`[]`),
				Files:  files,
			},
		},
	})
	instance := platformKnowledgeFilterTestDescriptor(t)
	instance.Tools = append(instance.Tools,
		issue11017ToolSnapshot(t, agenttools.ToolKindMoiGitHubTools),
		issue11017ToolSnapshot(t, agenttools.ToolKindReadFile),
	)

	tools, err := filter.FilterTools(context.Background(), agentruntime.RuntimeToolFilterRequest{
		Scope:    agentruntime.RuntimeRequestScope{WorkspaceID: "ws_1", AgentID: "explore", UserID: "user_1"},
		Instance: instance,
		Metadata: map[string]any{
			"semantic_model_ids": []any{float64(2)},
		},
	})
	if err != nil {
		t.Fatalf("FilterTools() error = %v", err)
	}
	if platformKnowledgeFilterTestContains(tools, agenttools.ToolKindMoiGitHubTools) {
		t.Fatalf("default explore exposed extra GitHub tool: %+v", tools)
	}
	if !platformKnowledgeFilterTestContains(tools, agenttools.ToolKindReadFile) {
		t.Fatalf("default explore should keep runtime delivery tool read_file: %+v", tools)
	}
	if platformKnowledgeFilterTestContains(tools, agenttools.ToolKindSubmitFinalAnswer) {
		t.Fatalf("default explore exposed legacy submit final answer tool: %+v", tools)
	}
}

func TestPlatformKnowledgeToolFilterCustomAgentKeepsExtraChannelTools(t *testing.T) {
	files, err := json.Marshal(map[string]any{
		"vector_table":    "kb_resume_index",
		"embedding_model": "bge-m3",
		"file_ids":        []string{},
	})
	if err != nil {
		t.Fatalf("marshal files: %v", err)
	}
	filter := NewPlatformKnowledgeToolFilter(&PlatformKnowledgeScopeResolver{
		semantic: platformKnowledgeFilterTestSemanticStore{
			model: &tenant.SemanticModelRecord{
				ID:     2,
				Name:   "resume kb",
				Tables: json.RawMessage(`[]`),
				Files:  files,
			},
		},
	})
	instance := platformKnowledgeFilterTestDescriptor(t)
	instance.AgentID = "custom_agent"
	instance.Tools = append(instance.Tools, issue11017ToolSnapshot(t, agenttools.ToolKindMoiGitHubTools))

	tools, err := filter.FilterTools(context.Background(), agentruntime.RuntimeToolFilterRequest{
		Scope:    agentruntime.RuntimeRequestScope{WorkspaceID: "ws_1", AgentID: "custom_agent", UserID: "user_1"},
		Instance: instance,
		Metadata: map[string]any{
			"semantic_model_ids": []any{float64(2)},
		},
	})
	if err != nil {
		t.Fatalf("FilterTools() error = %v", err)
	}
	if !platformKnowledgeFilterTestContains(tools, agenttools.ToolKindMoiGitHubTools) {
		t.Fatalf("custom agent should keep explicitly bound GitHub tool: %+v", tools)
	}
	if platformKnowledgeFilterTestContains(tools, agenttools.ToolKindSubmitFinalAnswer) {
		t.Fatalf("custom knowledge scope exposed legacy submit final answer tool: %+v", tools)
	}
}

func TestPlatformKnowledgeToolFilterTurnSelectedKBDoesNotUseBindingTables(t *testing.T) {
	filter := NewPlatformKnowledgeToolFilter(&PlatformKnowledgeScopeResolver{
		semantic: platformKnowledgeFilterTestSemanticStore{
			model: &tenant.SemanticModelRecord{
				ID:     60002,
				Name:   "20个",
				Tables: json.RawMessage(`[]`),
				Files:  json.RawMessage(`{}`),
			},
		},
	})
	instance := platformKnowledgeFilterTestDescriptor(t)
	instance.KnowledgeBases = []agentruntime.AgentKnowledge{platformKnowledgeFilterTestBindingKnowledge()}
	metadata := map[string]any{
		"semantic_model_ids": []any{float64(60002)},
	}

	tools, err := filter.FilterTools(context.Background(), agentruntime.RuntimeToolFilterRequest{
		Scope:    agentruntime.RuntimeRequestScope{WorkspaceID: "ws_1", AgentID: "explore", UserID: "user_1"},
		Instance: instance,
		Metadata: metadata,
	})
	if err != nil {
		t.Fatalf("FilterTools() error = %v", err)
	}
	for _, hidden := range []string{agenttools.ToolKindDescribeSchema, agenttools.ToolKindQuerySQL, agenttools.ToolKindComputeResultTable} {
		if platformKnowledgeFilterTestContains(tools, hidden) {
			t.Fatalf("selected KB without tables exposed %s from binding: %+v", hidden, tools)
		}
	}

}

func TestRuntimeServiceMessageSendSelectedTablelessKBDoesNotPersistBindingSQLScope(t *testing.T) {
	scopeResolver := &PlatformKnowledgeScopeResolver{
		semantic: platformKnowledgeFilterTestSemanticStore{
			model: &tenant.SemanticModelRecord{
				ID:     60002,
				Name:   "20个",
				Tables: json.RawMessage(`[]`),
				Files:  json.RawMessage(`{}`),
			},
		},
	}
	filter := NewPlatformKnowledgeToolFilter(scopeResolver)
	instance := platformKnowledgeFilterTestDescriptor(t)
	instance.AgentVersion = "review-12985"
	instance.DisplayName = "Knowledge Explore"
	instance.RuntimeProvider = agentruntime.ProviderAgentFacade
	instance.RuntimeProfile = agentruntime.ProfileDefault
	instance.Capabilities = agentruntime.BackendCapabilities{
		A2AProfile:             agentruntime.A2AProfileStreaming,
		StreamingMode:          agentruntime.StreamingNative,
		CancelMode:             agentruntime.CancelFacadeOnly,
		ToolMode:               agentruntime.ToolModeGateway,
		StateTransitionHistory: true,
	}
	instance.Model = agentruntime.AgentModel{DefaultModel: "review-model"}
	instance.KnowledgeBases = []agentruntime.AgentKnowledge{platformKnowledgeFilterTestBindingKnowledge()}
	store := agentruntime.NewInMemoryRuntimeStore()
	service := agentruntime.NewAgentRuntimeV2Service(
		store,
		testRuntimeInstanceResolver{instance: &instance},
		agentruntime.WithAgentRuntimeV2Backend(agentruntime.NewAgentRuntimeV2Backend(
			agentruntime.WithAgentRuntimeV2Runner(platformKnowledgeFilterCompletedRunner{}),
		)),
		agentruntime.WithRuntimeToolFilter("explore", filter),
	)
	ctx := agentruntime.WithRuntimeRequestScope(context.Background(), agentruntime.RuntimeRequestScope{
		WorkspaceID:     "ws_1",
		AgentID:         "explore",
		UserID:          "user_1",
		EffectiveRoleID: "role_1",
	})
	result, rpcErr := service.MessageSend(ctx, json.RawMessage(`{
		"message": {
			"role": "user",
			"messageId": "msg_review_selected_tableless",
			"contextId": "ctx_review_selected_tableless",
			"parts": [{"kind":"text","text":"查询当前选中知识库"}],
			"metadata": {"semantic_model_ids": [60002]}
		}
	}`))
	if rpcErr != nil {
		t.Fatalf("MessageSend() rpc error = %+v", rpcErr)
	}
	task := result.(*a2a.Task)
	stored, err := store.GetTask(context.Background(), "ws_1", task.ID)
	if err != nil {
		t.Fatalf("GetTask() error = %v", err)
	}
	manifest, err := store.GetManifest(context.Background(), "ws_1", stored.ManifestID)
	if err != nil {
		t.Fatalf("GetManifest() error = %v", err)
	}
	toolIDs := platformKnowledgeManifestToolIDs(manifest.Body["tools"])
	for _, hidden := range []string{agenttools.ToolKindDescribeSchema, agenttools.ToolKindQuerySQL, agenttools.ToolKindComputeResultTable} {
		if _, ok := toolIDs[hidden]; ok {
			t.Fatalf("stored manifest exposed %s for table-less selected KB: %+v", hidden, toolIDs)
		}
	}
	knowledgeIDs := platformKnowledgeManifestKnowledgeIDs(manifest.Body["knowledge_bases"])
	if _, ok := knowledgeIDs["50022"]; ok {
		t.Fatalf("stored manifest leaked unselected binding KB 50022: %+v", knowledgeIDs)
	}
	manifestJSON, err := json.Marshal(manifest.Body)
	if err != nil {
		t.Fatalf("marshal manifest body: %v", err)
	}
	if containsAny(string(manifestJSON), "kb_50022_t_60008", "dimproductsubcategory") {
		t.Fatalf("stored manifest leaked binding table scope: %s", manifestJSON)
	}
	input, err := store.GetTurnSnapshotForTask(context.Background(), "ws_1", task.ID, agentruntime.RuntimeSnapshotKindTurnInput)
	if err != nil {
		t.Fatalf("GetTurnSnapshotForTask() error = %v", err)
	}
	inputJSON, err := json.Marshal(input.Body)
	if err != nil {
		t.Fatalf("marshal turn input snapshot: %v", err)
	}
	if containsAny(string(inputJSON), "kb_50022_t_60008", "dimproductsubcategory") {
		t.Fatalf("turn input snapshot leaked binding table scope: %s", inputJSON)
	}
}

func TestPlatformKnowledgeToolFilterDBNameOnlyScopeHidesSQLTools(t *testing.T) {
	filter := NewPlatformKnowledgeToolFilter(&PlatformKnowledgeScopeResolver{})
	descriptor := platformKnowledgeFilterTestDescriptor(t)
	descriptor.PolicySummary = map[string]any{}
	tools, err := filter.FilterTools(context.Background(), agentruntime.RuntimeToolFilterRequest{
		Scope:    agentruntime.RuntimeRequestScope{WorkspaceID: "ws_1", AgentID: "explore", UserID: "user_1"},
		Instance: descriptor,
		Metadata: map[string]any{
			"db_name": "kb_50022_t_60008",
		},
	})
	if err != nil {
		t.Fatalf("FilterTools() error = %v", err)
	}
	for _, hidden := range []string{agenttools.ToolKindDescribeSchema, agenttools.ToolKindQuerySQL, agenttools.ToolKindComputeResultTable} {
		if platformKnowledgeFilterTestContains(tools, hidden) {
			t.Fatalf("DB-name-only scope exposed %s without explicit table scope: %+v", hidden, tools)
		}
	}
	if descriptor.PolicySummary["runtime_unavailable_reason"] != platformKnowledgeToolRuntimeScopeUnavailableReason {
		t.Fatalf("runtime policy summary = %+v, want scope unavailable reason", descriptor.PolicySummary)
	}
}

func TestPlatformKnowledgeToolFilterKeepsBindingTablesWithoutTurnSelection(t *testing.T) {
	filter := NewPlatformKnowledgeToolFilter(&PlatformKnowledgeScopeResolver{
		semantic: platformKnowledgeFilterTestSemanticStore{
			model: &tenant.SemanticModelRecord{
				ID:     50022,
				Name:   "sssa",
				Tables: json.RawMessage(`[]`),
				Files:  json.RawMessage(`{}`),
			},
		},
	})
	instance := platformKnowledgeFilterTestDescriptor(t)
	instance.KnowledgeBases = []agentruntime.AgentKnowledge{platformKnowledgeFilterTestBindingKnowledge()}

	tools, err := filter.FilterTools(context.Background(), agentruntime.RuntimeToolFilterRequest{
		Scope:    agentruntime.RuntimeRequestScope{WorkspaceID: "ws_1", AgentID: "explore", UserID: "user_1"},
		Instance: instance,
	})
	if err != nil {
		t.Fatalf("FilterTools() error = %v", err)
	}
	for _, visible := range []string{agenttools.ToolKindDescribeSchema, agenttools.ToolKindQuerySQL} {
		if !platformKnowledgeFilterTestContains(tools, visible) {
			t.Fatalf("binding table scope missing %s: %+v", visible, tools)
		}
	}
	if platformKnowledgeFilterTestContains(tools, agenttools.ToolKindComputeResultTable) {
		t.Fatalf("binding table scope still exposed compute_result_table: %+v", tools)
	}

}

func TestPlatformKnowledgeToolFilterEmbeddingOnlyScopeHidesRAGTools(t *testing.T) {
	files, err := json.Marshal(map[string]any{
		"embedding_model": "bge-m3",
	})
	if err != nil {
		t.Fatalf("marshal files: %v", err)
	}
	filter := NewPlatformKnowledgeToolFilter(&PlatformKnowledgeScopeResolver{
		semantic: platformKnowledgeFilterTestSemanticStore{
			model: &tenant.SemanticModelRecord{
				ID:     4,
				Name:   "incomplete rag kb",
				Tables: json.RawMessage(`[]`),
				Files:  files,
			},
		},
	})
	tools, err := filter.FilterTools(context.Background(), agentruntime.RuntimeToolFilterRequest{
		Scope:    agentruntime.RuntimeRequestScope{WorkspaceID: "ws_1", AgentID: "explore", UserID: "user_1"},
		Instance: platformKnowledgeFilterTestDescriptor(t),
		Metadata: map[string]any{
			"semantic_model_ids": []any{float64(4)},
		},
	})
	if err != nil {
		t.Fatalf("FilterTools() error = %v", err)
	}
	for _, hidden := range []string{agenttools.ToolKindSearchRAGChunks, agenttools.ToolKindFindRAGFiles, agenttools.ToolKindReadParsedMarkdown, agenttools.ToolKindSearchParsedMarkdown} {
		if platformKnowledgeFilterTestContains(tools, hidden) {
			t.Fatalf("embedding-only scope exposed %s: %+v", hidden, tools)
		}
	}
	if !platformKnowledgeFilterTestContains(tools, agenttools.ToolKindSelectFinalSources) {
		t.Fatalf("missing select final sources tool: %+v", tools)
	}
	if platformKnowledgeFilterTestContains(tools, agenttools.ToolKindSubmitFinalAnswer) {
		t.Fatalf("embedding-only scope exposed legacy submit final answer tool: %+v", tools)
	}

	instruction, err := filter.FilterInstruction(context.Background(), agentruntime.RuntimeInstructionFilterRequest{
		Scope: agentruntime.RuntimeRequestScope{WorkspaceID: "ws_1", AgentID: "explore", UserID: "user_1"},
		Instance: agentruntime.AgentInstance{
			Tools: tools,
		},
		Metadata: map[string]any{
			"semantic_model_ids": []any{float64(4)},
		},
	})
	if err != nil {
		t.Fatalf("FilterInstruction() error = %v", err)
	}
	if containsAny(instruction.SystemPrompt, "search_rag_chunks", "query_sql", "Mixed-source obligation") {
		t.Fatalf("empty-source prompt contains unavailable tool guidance:\n%s", instruction.SystemPrompt)
	}
	if !containsAll(instruction.SystemPrompt, "select_final_sources", "Do not call submit_final_answer") {
		t.Fatalf("empty-source prompt missing cite-then-write guidance:\n%s", instruction.SystemPrompt)
	}
}

func TestPlatformKnowledgeScopeHasTextRAGAllowsDirectVectorTableOnly(t *testing.T) {
	if !platformKnowledgeScopeHasTextRAG(knowledge.WorkspaceScope{VectorTable: "legacy_text_vec"}) {
		t.Fatalf("direct vector_table-only scope should expose text RAG tools")
	}
	if !platformKnowledgeScopeHasTextRAG(knowledge.WorkspaceScope{
		RAGSources: []knowledge.RAGSource{{VectorTable: "legacy_text_vec"}},
	}) {
		t.Fatalf("direct RAG source with vector_table only should expose text RAG tools")
	}
}

func TestPlatformKnowledgeScopeHasTextRAGRequiresSemanticEmbeddingModel(t *testing.T) {
	if platformKnowledgeScopeHasTextRAG(knowledge.WorkspaceScope{
		SemanticModelIDs: []int64{42},
		VectorTable:      "semantic_text_vec",
	}) {
		t.Fatalf("semantic model scope should not expose text RAG tools without embedding_model")
	}
	if platformKnowledgeScopeHasTextRAG(knowledge.WorkspaceScope{
		RAGSources: []knowledge.RAGSource{{
			SemanticModelID: 42,
			VectorTable:     "semantic_text_vec",
		}},
	}) {
		t.Fatalf("semantic RAG source should not expose text RAG tools without embedding_model")
	}
}

func TestPlatformKnowledgeToolFilterTableScopeHidesRAGTools(t *testing.T) {
	tables, err := json.Marshal([]map[string]any{{
		"db_name":     "retail",
		"table_names": []string{"orders"},
	}})
	if err != nil {
		t.Fatalf("marshal tables: %v", err)
	}
	filter := NewPlatformKnowledgeToolFilter(&PlatformKnowledgeScopeResolver{
		semantic: platformKnowledgeFilterTestSemanticStore{
			model: &tenant.SemanticModelRecord{
				ID:     3,
				Name:   "table kb",
				Tables: tables,
				Files:  json.RawMessage(`{}`),
			},
		},
	})
	tools, err := filter.FilterTools(context.Background(), agentruntime.RuntimeToolFilterRequest{
		Scope:    agentruntime.RuntimeRequestScope{WorkspaceID: "ws_1", AgentID: "explore", UserID: "user_1"},
		Instance: platformKnowledgeFilterTestDescriptor(t),
		Metadata: map[string]any{
			"database":           "retail",
			"semantic_model_ids": []any{float64(3)},
		},
	})
	if err != nil {
		t.Fatalf("FilterTools() error = %v", err)
	}
	for _, visible := range []string{agenttools.ToolKindDescribeSchema, agenttools.ToolKindQuerySQL, agenttools.ToolKindSelectFinalSources} {
		if !platformKnowledgeFilterTestContains(tools, visible) {
			t.Fatalf("table scope missing %s: %+v", visible, tools)
		}
	}
	if platformKnowledgeFilterTestContains(tools, agenttools.ToolKindComputeResultTable) {
		t.Fatalf("table scope still exposed compute_result_table: %+v", tools)
	}
	if platformKnowledgeFilterTestContains(tools, agenttools.ToolKindSubmitFinalAnswer) {
		t.Fatalf("table scope exposed legacy submit final answer tool: %+v", tools)
	}
	if platformKnowledgeFilterTestContains(tools, agenttools.ToolKindSearchRAGChunks) {
		t.Fatalf("table-only scope exposed search_rag_chunks: %+v", tools)
	}

	instruction, err := filter.FilterInstruction(context.Background(), agentruntime.RuntimeInstructionFilterRequest{
		Scope: agentruntime.RuntimeRequestScope{WorkspaceID: "ws_1", AgentID: "explore", UserID: "user_1"},
		Instance: agentruntime.AgentInstance{
			Tools: tools,
		},
		Metadata: map[string]any{
			"database":           "retail",
			"semantic_model_ids": []any{float64(3)},
		},
	})
	if err != nil {
		t.Fatalf("FilterInstruction() error = %v", err)
	}
	if !containsAll(instruction.SystemPrompt, "Tenant scope", "scope database: retail", "SQL allowed tables (constrains SQL tools only; do NOT query outside this list): retail.orders", "SQL tools", "query_sql", "NL2SQL evidence", "omit `table_names` or pass `[]`", "Never infer table names from user wording", "valid MySQL 8 syntax", "current_date injected in MOI Runtime Scope", "Do not treat the latest year present in a table as last year", "Pure scalar SELECT without FROM") {
		t.Fatalf("table-only prompt missing SQL guidance:\n%s", instruction.SystemPrompt)
	}
	if strings.Contains(instruction.SystemPrompt, "describe_schema(table_names=...)") {
		t.Fatalf("table-only prompt still encourages guessed table_names:\n%s", instruction.SystemPrompt)
	}
	if containsAny(instruction.SystemPrompt, "search_rag_chunks", "Document retrieval", "Mixed-source obligation", "rag_chunk") {
		t.Fatalf("table-only prompt contains RAG/mixed guidance:\n%s", instruction.SystemPrompt)
	}
}

func TestPlatformKnowledgeToolFilterInstructionMatchesUnavailableSQLTools(t *testing.T) {
	filter := NewPlatformKnowledgeToolFilter(&PlatformKnowledgeScopeResolver{
		semantic: platformKnowledgeFilterTestSemanticStore{
			model: &tenant.SemanticModelRecord{
				ID:     91,
				Name:   "unresolved table kb",
				Tables: json.RawMessage(`[{"db_name":"","table_names":["moi_github_identities"]}]`),
				Files:  json.RawMessage(`{}`),
			},
		},
	})
	instance := platformKnowledgeFilterTestDescriptor(t)
	instruction, err := filter.FilterInstruction(context.Background(), agentruntime.RuntimeInstructionFilterRequest{
		Scope:    agentruntime.RuntimeRequestScope{WorkspaceID: "ws_1", AgentID: "explore", UserID: "user_1"},
		Instance: instance,
		Metadata: map[string]any{
			"semantic_model_ids": []any{float64(91)},
		},
	})
	if err != nil {
		t.Fatalf("FilterInstruction() error = %v", err)
	}
	if containsAny(instruction.SystemPrompt, "## SQL tools\n", "- Schema: call `describe_schema`", "- If a table is returned with `queryable=false`", "For NL2SQL evidence") {
		t.Fatalf("prompt contains unavailable SQL tool guidance:\n%s", instruction.SystemPrompt)
	}
}

func TestPlatformKnowledgeToolFilterNeverExposesTableMutation(t *testing.T) {
	tables := json.RawMessage(`[{"db_name":"engineering","table_names":["moi_github_identities"]}]`)
	filter := NewPlatformKnowledgeToolFilter(&PlatformKnowledgeScopeResolver{
		semantic: platformKnowledgeFilterTestSemanticStore{
			model: &tenant.SemanticModelRecord{ID: 92, Name: "engineering directory", Tables: tables, Files: json.RawMessage(`{}`)},
		},
	})
	instance := platformKnowledgeFilterTestDescriptor(t)
	instance.Tools = append(instance.Tools, issue11017ToolSnapshot(t, agenttools.ToolKindUpsertKnowledgeTable))
	metadata := map[string]any{"database": "engineering", "semantic_model_ids": []any{float64(92)}}

	tools, err := filter.FilterTools(context.Background(), agentruntime.RuntimeToolFilterRequest{
		Scope:    agentruntime.RuntimeRequestScope{WorkspaceID: "ws_1", AgentID: "explore", UserID: "user_1"},
		Instance: instance,
		Metadata: metadata,
	})
	if err != nil {
		t.Fatalf("FilterTools(untrusted) error = %v", err)
	}
	if platformKnowledgeFilterTestContains(tools, agenttools.ToolKindUpsertKnowledgeTable) {
		t.Fatalf("untrusted knowledge task exposed upsert_knowledge_table: %+v", tools)
	}

	nonDirectoryFilter := NewPlatformKnowledgeToolFilter(&PlatformKnowledgeScopeResolver{
		semantic: platformKnowledgeFilterTestSemanticStore{
			model: &tenant.SemanticModelRecord{ID: 93, Name: "other table", Tables: json.RawMessage(`[{"db_name":"engineering","table_names":["other_table"]}]`), Files: json.RawMessage(`{}`)},
		},
	})
	tools, err = nonDirectoryFilter.FilterTools(context.Background(), agentruntime.RuntimeToolFilterRequest{
		Scope:    agentruntime.RuntimeRequestScope{WorkspaceID: "ws_1", AgentID: "explore", UserID: "user_1"},
		Instance: instance,
		Metadata: map[string]any{
			"database":           "engineering",
			"semantic_model_ids": []any{float64(93)},
		},
	})
	if err != nil {
		t.Fatalf("FilterTools(non-directory trusted delegate) error = %v", err)
	}
	if platformKnowledgeFilterTestContains(tools, agenttools.ToolKindUpsertKnowledgeTable) {
		t.Fatalf("trusted delegate exposed upsert_knowledge_table outside the engineering directory: %+v", tools)
	}
}

func TestPlatformKnowledgeToolFilterDerivesDatabaseFromSemanticModel(t *testing.T) {
	resolver := &PlatformKnowledgeScopeResolver{
		semantic: platformKnowledgeFilterTestSemanticStore{
			model: &tenant.SemanticModelRecord{
				ID:     94,
				Name:   "engineering directory",
				Tables: json.RawMessage(`[{"db_name":"moi","table_names":["moi_github_identities"]}]`),
				Files:  json.RawMessage(`{}`),
			},
		},
	}
	scope, err := resolver.ResolveKnowledgeScope(context.Background(), knowledge.WorkspaceScope{
		WorkspaceID:      "ws_1",
		UserID:           "user_1",
		SemanticModelIDs: []int64{94},
	})
	if err != nil {
		t.Fatalf("ResolveKnowledgeScope() error = %v", err)
	}
	if scope.DBName != "moi" || !reflect.DeepEqual(scope.Tables, []string{"moi.moi_github_identities"}) {
		t.Fatalf("scope database/tables = %q/%#v, want moi/[moi.moi_github_identities]", scope.DBName, scope.Tables)
	}

	instance := platformKnowledgeFilterTestDescriptor(t)
	instance.Tools = append(instance.Tools, issue11017ToolSnapshot(t, agenttools.ToolKindUpsertKnowledgeTable))
	tools, err := NewPlatformKnowledgeToolFilter(resolver).FilterTools(context.Background(), agentruntime.RuntimeToolFilterRequest{
		Scope:    agentruntime.RuntimeRequestScope{WorkspaceID: "ws_1", AgentID: "explore", UserID: "user_1"},
		Instance: instance,
		Metadata: map[string]any{
			"semantic_model_ids": []any{float64(94)},
		},
	})
	if err != nil {
		t.Fatalf("FilterTools() error = %v", err)
	}
	for _, kind := range []string{agenttools.ToolKindDescribeSchema, agenttools.ToolKindQuerySQL} {
		if !platformKnowledgeFilterTestContains(tools, kind) {
			t.Fatalf("semantic model scope missing %s: %+v", kind, tools)
		}
	}
	if platformKnowledgeFilterTestContains(tools, agenttools.ToolKindUpsertKnowledgeTable) {
		t.Fatalf("semantic model scope exposed upsert_knowledge_table: %+v", tools)
	}
}

func TestPlatformKnowledgeScopeResolverAllowsMultiDatabaseSemanticModelTables(t *testing.T) {
	resolver := &PlatformKnowledgeScopeResolver{
		semantic: platformKnowledgeFilterTestSemanticStore{
			model: &tenant.SemanticModelRecord{
				ID:     95,
				Name:   "multiple database tables",
				Tables: json.RawMessage(`[{"db_name":"sales","table_names":["orders"]},{"db_name":"support","table_names":["tickets"]}]`),
				Files:  json.RawMessage(`{}`),
			},
		},
	}
	scope, err := resolver.ResolveKnowledgeScope(context.Background(), knowledge.WorkspaceScope{
		WorkspaceID:      "ws_1",
		UserID:           "user_1",
		SemanticModelIDs: []int64{95},
	})
	if err != nil {
		t.Fatalf("ResolveKnowledgeScope() error = %v", err)
	}
	if scope.DBName != "" {
		t.Fatalf("DBName = %q, want empty multi-database scope", scope.DBName)
	}
	if !reflect.DeepEqual(scope.Tables, []string{"sales.orders", "support.tickets"}) {
		t.Fatalf("Tables = %#v, want [sales.orders support.tickets]", scope.Tables)
	}
	caps := platformKnowledgeScopeToolCapabilities(scope)
	if !caps.sql {
		t.Fatalf("multi-database qualified tables should enable SQL tools: %#v", caps)
	}
}

// MF-1: metadata default database must not silently drop other-db tables.
func TestPlatformKnowledgeScopeResolverKeepsMultiDBTablesWhenMetadataDBNamePresent(t *testing.T) {
	resolver := &PlatformKnowledgeScopeResolver{
		semantic: platformKnowledgeFilterTestSemanticStore{
			model: &tenant.SemanticModelRecord{
				ID:     96,
				Name:   "multi db with metadata default",
				Tables: json.RawMessage(`[{"db_name":"sales","table_names":["orders"]},{"db_name":"support","table_names":["tickets"]}]`),
				Files:  json.RawMessage(`{}`),
			},
		},
	}
	scope, err := resolver.ResolveKnowledgeScope(context.Background(), knowledge.WorkspaceScope{
		WorkspaceID:      "ws_1",
		UserID:           "user_1",
		DBName:           "sales",
		SemanticModelIDs: []int64{96},
	})
	if err != nil {
		t.Fatalf("ResolveKnowledgeScope() error = %v", err)
	}
	if scope.DBName != "" {
		t.Fatalf("DBName = %q, want empty after multi-db resolve", scope.DBName)
	}
	if !reflect.DeepEqual(scope.Tables, []string{"sales.orders", "support.tickets"}) {
		t.Fatalf("Tables = %#v, want both databases preserved", scope.Tables)
	}
}

// Live regression: frontend seeds bare tables + multi-db semantic model (t2:
// kb_2.dimproductsubcategory + kb_1.test). Bare pre-seeds must not poison
// SQL capabilities (trace_e3d0bdaa… failed with unsupported describe_schema).
func TestPlatformKnowledgeScopeResolverBareMetadataTablesDoNotPoisonMultiDBSQL(t *testing.T) {
	resolver := &PlatformKnowledgeScopeResolver{
		semantic: platformKnowledgeFilterTestSemanticStore{
			model: &tenant.SemanticModelRecord{
				ID:   2,
				Name: "t2",
				Tables: json.RawMessage(`[
					{"db_name":"kb_2","table_names":["dimproductsubcategory"]},
					{"db_name":"kb_1","table_names":["test"]}
				]`),
				Files: json.RawMessage(`{}`),
			},
		},
	}
	scope, err := resolver.ResolveKnowledgeScope(context.Background(), knowledge.WorkspaceScope{
		WorkspaceID: "ws_1",
		UserID:      "user_1",
		// Frontend seeds bare names; no top-level database (multi-kb).
		Tables:           []string{"dimproductsubcategory", "test"},
		SemanticModelIDs: []int64{2},
	})
	if err != nil {
		t.Fatalf("ResolveKnowledgeScope() error = %v", err)
	}
	if scope.DBName != "" {
		t.Fatalf("DBName = %q, want empty multi-database scope", scope.DBName)
	}
	if !reflect.DeepEqual(scope.Tables, []string{"kb_1.test", "kb_2.dimproductsubcategory"}) &&
		!reflect.DeepEqual(scope.Tables, []string{"kb_2.dimproductsubcategory", "kb_1.test"}) {
		t.Fatalf("Tables = %#v, want qualified multi-db identities only", scope.Tables)
	}
	for _, table := range scope.Tables {
		if !strings.Contains(table, ".") {
			t.Fatalf("Tables = %#v, bare name must not remain after multi-db resolve", scope.Tables)
		}
	}
	caps := platformKnowledgeScopeToolCapabilities(scope)
	if !caps.sql {
		t.Fatalf("SQL capabilities disabled; bare metadata poisoned multi-db scope: tables=%#v db=%q caps=%#v",
			scope.Tables, scope.DBName, caps)
	}
}

// Single-db model with bare metadata tables still resolves to database.table
// and keeps SQL tools (good path: t1 / trace_1948b003…).
func TestPlatformKnowledgeScopeResolverBareMetadataTablesSingleDBStillSQL(t *testing.T) {
	resolver := &PlatformKnowledgeScopeResolver{
		semantic: platformKnowledgeFilterTestSemanticStore{
			model: &tenant.SemanticModelRecord{
				ID:   1,
				Name: "t1",
				Tables: json.RawMessage(`[
					{"db_name":"kb_1","table_names":["dimproductsubcategory","test"]}
				]`),
				Files: json.RawMessage(`{}`),
			},
		},
	}
	scope, err := resolver.ResolveKnowledgeScope(context.Background(), knowledge.WorkspaceScope{
		WorkspaceID:      "ws_1",
		UserID:           "user_1",
		DBName:           "kb_1",
		Tables:           []string{"dimproductsubcategory", "test"},
		SemanticModelIDs: []int64{1},
	})
	if err != nil {
		t.Fatalf("ResolveKnowledgeScope() error = %v", err)
	}
	if scope.DBName != "kb_1" {
		t.Fatalf("DBName = %q, want kb_1", scope.DBName)
	}
	if !reflect.DeepEqual(scope.Tables, []string{"kb_1.dimproductsubcategory", "kb_1.test"}) {
		t.Fatalf("Tables = %#v, want qualified single-db identities", scope.Tables)
	}
	caps := platformKnowledgeScopeToolCapabilities(scope)
	if !caps.sql {
		t.Fatalf("single-db bare metadata should keep SQL tools: %#v", caps)
	}
}

func TestPlatformKnowledgeParseTableIdentityPreservesDottedBareTableWithDefaultDB(t *testing.T) {
	schema, name := platformKnowledgeParseTableIdentity("test.csv", "ffff_15")
	if schema != "ffff_15" || name != "test.csv" {
		t.Fatalf("got {%q, %q}, want {ffff_15, test.csv}", schema, name)
	}
	schema, name = platformKnowledgeParseTableIdentity("ffff_15.test.csv", "ffff_15")
	if schema != "ffff_15" || name != "test.csv" {
		t.Fatalf("qualified got {%q, %q}, want {ffff_15, test.csv}", schema, name)
	}
	refs := platformKnowledgeDefaultTableRefs([]string{"test.csv"}, "ffff_15")
	if len(refs) != 1 || refs[0].DBName != "ffff_15" || refs[0].Name != "test.csv" {
		t.Fatalf("default table refs = %#v, want ffff_15/test.csv", refs)
	}
}

// MF-3 regression: binding/runtime seeds bare dotted table_name=test.csv with
// Scope.DBName=ffff_15 (CatalogAssetRefs config). Must not first-dot split into
// {DB:test, Table:csv} and then filter away the real model table.
func TestPlatformKnowledgeScopeResolverPreservesDottedBareTableWithDefaultDB(t *testing.T) {
	resolver := &PlatformKnowledgeScopeResolver{
		semantic: platformKnowledgeFilterTestSemanticStore{
			model: &tenant.SemanticModelRecord{
				ID:     97,
				Name:   "csv table kb",
				Tables: json.RawMessage(`[{"db_name":"ffff_15","table_names":["test.csv"]}]`),
				Files:  json.RawMessage(`{}`),
			},
		},
	}
	scope, err := resolver.ResolveKnowledgeScope(context.Background(), knowledge.WorkspaceScope{
		WorkspaceID:      "ws_1",
		UserID:           "user_1",
		DBName:           "ffff_15",
		Tables:           []string{"test.csv"},
		SemanticModelIDs: []int64{97},
	})
	if err != nil {
		t.Fatalf("ResolveKnowledgeScope() error = %v", err)
	}
	if scope.DBName != "ffff_15" {
		t.Fatalf("DBName = %q, want ffff_15", scope.DBName)
	}
	if !reflect.DeepEqual(scope.Tables, []string{"ffff_15.test.csv"}) {
		t.Fatalf("Tables = %#v, want [ffff_15.test.csv]", scope.Tables)
	}
	caps := platformKnowledgeScopeToolCapabilities(scope)
	if !caps.sql {
		t.Fatalf("dotted bare table with default DB should keep SQL tools: %#v", caps)
	}
}

// MF-7: qualified selection hint sales.orders must not also bare-match
// support.orders and widen the SQL allowed scope.
func TestPlatformKnowledgeScopeResolverQualifiedHintDoesNotWidenToOtherDBSameTable(t *testing.T) {
	resolver := &PlatformKnowledgeScopeResolver{
		semantic: platformKnowledgeFilterTestSemanticStore{
			model: &tenant.SemanticModelRecord{
				ID:   98,
				Name: "cross-db same table name",
				Tables: json.RawMessage(`[
					{"db_name":"sales","table_names":["orders"]},
					{"db_name":"support","table_names":["orders"]}
				]`),
				Files: json.RawMessage(`{}`),
			},
		},
	}
	scope, err := resolver.ResolveKnowledgeScope(context.Background(), knowledge.WorkspaceScope{
		WorkspaceID:      "ws_1",
		UserID:           "user_1",
		Tables:           []string{"sales.orders"},
		SemanticModelIDs: []int64{98},
	})
	if err != nil {
		t.Fatalf("ResolveKnowledgeScope() error = %v", err)
	}
	if !reflect.DeepEqual(scope.Tables, []string{"sales.orders"}) {
		t.Fatalf("Tables = %#v, want only sales.orders (qualified hint must not admit support.orders)", scope.Tables)
	}
	if scope.DBName != "sales" {
		t.Fatalf("DBName = %q, want sales (single remaining database)", scope.DBName)
	}
}

// MF-8: Scope.DBName=sales must not rewrite other-database qualified hint
// support.orders into sales.support.orders (d68114eb3 regression).
func TestPlatformKnowledgeScopeResolverOtherDBQualifiedHintKeepsIdentityWithDefaultDB(t *testing.T) {
	resolver := &PlatformKnowledgeScopeResolver{
		semantic: platformKnowledgeFilterTestSemanticStore{
			model: &tenant.SemanticModelRecord{
				ID:   99,
				Name: "multi db with metadata default",
				Tables: json.RawMessage(`[
					{"db_name":"sales","table_names":["orders"]},
					{"db_name":"support","table_names":["orders"]}
				]`),
				Files: json.RawMessage(`{}`),
			},
		},
	}
	scope, err := resolver.ResolveKnowledgeScope(context.Background(), knowledge.WorkspaceScope{
		WorkspaceID:      "ws_1",
		UserID:           "user_1",
		DBName:           "sales",
		Tables:           []string{"support.orders"},
		SemanticModelIDs: []int64{99},
	})
	if err != nil {
		t.Fatalf("ResolveKnowledgeScope() error = %v", err)
	}
	if !reflect.DeepEqual(scope.Tables, []string{"support.orders"}) {
		t.Fatalf("Tables = %#v, want [support.orders] (must not rewrite to sales.support.orders)", scope.Tables)
	}
	if scope.DBName != "support" {
		t.Fatalf("DBName = %q, want support (only selected database)", scope.DBName)
	}
	candidates := []PlatformKnowledgeSemanticModelTableRef{
		{DBName: "sales", TableName: "orders"},
		{DBName: "support", TableName: "orders"},
	}
	hint := platformKnowledgeMatchIncomingTableHint("support.orders", candidates)
	if hint.DBName != "support" || hint.TableName != "orders" || hint.Bare {
		t.Fatalf("matchIncoming = %+v, want {support,orders,Bare:false}", hint)
	}
	hint = platformKnowledgeMatchIncomingTableHint("test.csv", []PlatformKnowledgeSemanticModelTableRef{
		{DBName: "ffff_15", TableName: "test.csv"},
	})
	if hint.DBName != "" || hint.TableName != "test.csv" || !hint.Bare {
		t.Fatalf("matchIncoming dotted bare = %+v, want {,test.csv,Bare:true}", hint)
	}
}

// MF-10: dotted bare table support.csv under defaultDB=ffff_15 must not be
// misread as database support + table csv just because the model also owns
// database "support".
func TestPlatformKnowledgeScopeResolverDottedBareTableNotConfusedWithKnownDBName(t *testing.T) {
	resolver := &PlatformKnowledgeScopeResolver{
		semantic: platformKnowledgeFilterTestSemanticStore{
			model: &tenant.SemanticModelRecord{
				ID:   100,
				Name: "dotted bare vs db name collision",
				Tables: json.RawMessage(`[
					{"db_name":"ffff_15","table_names":["support.csv"]},
					{"db_name":"support","table_names":["orders"]}
				]`),
				Files: json.RawMessage(`{}`),
			},
		},
	}
	scope, err := resolver.ResolveKnowledgeScope(context.Background(), knowledge.WorkspaceScope{
		WorkspaceID:      "ws_1",
		UserID:           "user_1",
		DBName:           "ffff_15",
		Tables:           []string{"support.csv"},
		SemanticModelIDs: []int64{100},
	})
	if err != nil {
		t.Fatalf("ResolveKnowledgeScope() error = %v", err)
	}
	if !reflect.DeepEqual(scope.Tables, []string{"ffff_15.support.csv"}) {
		t.Fatalf("Tables = %#v, want [ffff_15.support.csv] (not support.csv as db.table)", scope.Tables)
	}
	if scope.DBName != "ffff_15" {
		t.Fatalf("DBName = %q, want ffff_15", scope.DBName)
	}
	caps := platformKnowledgeScopeToolCapabilities(scope)
	if !caps.sql {
		t.Fatalf("SQL capabilities disabled for dotted bare under defaultDB: %#v", caps)
	}
	candidates := []PlatformKnowledgeSemanticModelTableRef{
		{DBName: "ffff_15", TableName: "support.csv"},
		{DBName: "support", TableName: "orders"},
	}
	hint := platformKnowledgeMatchIncomingTableHint("support.csv", candidates)
	if hint.DBName != "" || hint.TableName != "support.csv" || !hint.Bare {
		t.Fatalf("matchIncoming = %+v, want {,support.csv,Bare:true}", hint)
	}
	// Must not invent support.csv as database support + table csv.
	wrong := platformKnowledgeMatchIncomingTableHint("support.csv", candidates)
	if wrong.DBName == "support" && wrong.TableName == "csv" {
		t.Fatalf("matchIncoming must not first-dot invent support/csv: %+v", wrong)
	}
}

// Bare "orders" across multi-db models expands via bare TableName match (frontend
// seed). Qualified "sales.orders" stays exact — must not admit support.orders.
func TestPlatformKnowledgeMatchIncomingTableHintExactAndBare(t *testing.T) {
	candidates := []PlatformKnowledgeSemanticModelTableRef{
		{DBName: "sales", TableName: "orders"},
		{DBName: "support", TableName: "orders"},
		{DBName: "ffff_15", TableName: "test.csv"},
	}
	if hint := platformKnowledgeMatchIncomingTableHint("sales.orders", candidates); hint.DBName != "sales" || hint.TableName != "orders" || hint.Bare {
		t.Fatalf("qualified = %+v, want exact sales.orders", hint)
	}
	if hint := platformKnowledgeMatchIncomingTableHint("orders", candidates); !hint.Bare || hint.TableName != "orders" || hint.DBName != "" {
		t.Fatalf("bare = %+v, want bare orders selection", hint)
	}
	if hint := platformKnowledgeMatchIncomingTableHint("missing", candidates); hint.TableName != "" {
		t.Fatalf("unmatched = %+v, want empty (no guessing)", hint)
	}
	if hint := platformKnowledgeMatchIncomingTableHint("test.csv", candidates); !hint.Bare || hint.TableName != "test.csv" {
		t.Fatalf("dotted bare = %+v, want bare test.csv", hint)
	}
}

func TestPlatformKnowledgeScopeResolverPreservesQualifiedTablesWithoutSemanticModel(t *testing.T) {
	scope, err := (&PlatformKnowledgeScopeResolver{}).ResolveKnowledgeScope(context.Background(), knowledge.WorkspaceScope{
		WorkspaceID: "ws_1",
		UserID:      "user_1",
		Tables:      []string{"sales.orders", "support.tickets"},
	})
	if err != nil {
		t.Fatalf("ResolveKnowledgeScope() error = %v", err)
	}
	if scope.DBName != "" {
		t.Fatalf("DBName = %q, want empty multi-database scope", scope.DBName)
	}
	if !reflect.DeepEqual(scope.Tables, []string{"sales.orders", "support.tickets"}) {
		t.Fatalf("Tables = %#v, want qualified multi-db tables preserved", scope.Tables)
	}
}

func TestPlatformKnowledgeToolFilterMixedScopeKeepsSQLAndRAGPrompt(t *testing.T) {
	tables, err := json.Marshal([]map[string]any{{
		"db_name":     "retail",
		"table_names": []string{"orders"},
	}})
	if err != nil {
		t.Fatalf("marshal tables: %v", err)
	}
	files, err := json.Marshal(map[string]any{
		"vector_table":    "kb_retail_index",
		"embedding_model": "bge-m3",
		"file_ids":        []string{"file_1"},
	})
	if err != nil {
		t.Fatalf("marshal files: %v", err)
	}
	filter := NewPlatformKnowledgeToolFilter(&PlatformKnowledgeScopeResolver{
		semantic: platformKnowledgeFilterTestSemanticStore{
			model: &tenant.SemanticModelRecord{
				ID:     5,
				Name:   "mixed kb",
				Tables: tables,
				Files:  files,
			},
		},
	})
	tools, err := filter.FilterTools(context.Background(), agentruntime.RuntimeToolFilterRequest{
		Scope:    agentruntime.RuntimeRequestScope{WorkspaceID: "ws_1", AgentID: "explore", UserID: "user_1"},
		Instance: platformKnowledgeFilterTestDescriptor(t),
		Metadata: map[string]any{
			"database":           "retail",
			"semantic_model_ids": []any{float64(5)},
		},
	})
	if err != nil {
		t.Fatalf("FilterTools() error = %v", err)
	}
	for _, visible := range []string{agenttools.ToolKindDescribeSchema, agenttools.ToolKindQuerySQL, agenttools.ToolKindSearchRAGChunks, agenttools.ToolKindSelectFinalSources} {
		if !platformKnowledgeFilterTestContains(tools, visible) {
			t.Fatalf("mixed scope missing %s: %+v", visible, tools)
		}
	}
	if platformKnowledgeFilterTestContains(tools, agenttools.ToolKindSubmitFinalAnswer) {
		t.Fatalf("mixed scope exposed legacy submit final answer tool: %+v", tools)
	}
	instruction, err := filter.FilterInstruction(context.Background(), agentruntime.RuntimeInstructionFilterRequest{
		Scope: agentruntime.RuntimeRequestScope{WorkspaceID: "ws_1", AgentID: "explore", UserID: "user_1"},
		Instance: agentruntime.AgentInstance{
			Tools: tools,
		},
		Metadata: map[string]any{
			"database":           "retail",
			"semantic_model_ids": []any{float64(5)},
		},
	})
	if err != nil {
		t.Fatalf("FilterInstruction() error = %v", err)
	}
	if !containsAll(instruction.SystemPrompt, "Tenant scope", "scope database: retail", "SQL allowed tables (constrains SQL tools only; do NOT query outside this list): retail.orders", "document corpus: available through registered document tools: `find_rag_files`, `search_rag_chunks`, `read_parsed_markdown`, `search_parsed_markdown`", "SQL tools", "Document retrieval", "Mixed-source obligation", "query_sql", "search_rag_chunks", "SQL allowed tables do not describe or limit the document corpus", "rag_chunk", "NL2SQL evidence") {
		t.Fatalf("mixed prompt missing SQL/RAG guidance:\n%s", instruction.SystemPrompt)
	}
}

func TestPlatformKnowledgeToolFilterRAGPromptNamesOnlyRegisteredTools(t *testing.T) {
	files, err := json.Marshal(map[string]any{
		"vector_table":    "kb_retail_index",
		"embedding_model": "bge-m3",
		"file_ids":        []string{"file_1"},
	})
	if err != nil {
		t.Fatalf("marshal files: %v", err)
	}
	filter := NewPlatformKnowledgeToolFilter(&PlatformKnowledgeScopeResolver{
		semantic: platformKnowledgeFilterTestSemanticStore{
			model: &tenant.SemanticModelRecord{
				ID:     6,
				Name:   "rag kb",
				Tables: json.RawMessage(`[]`),
				Files:  files,
			},
		},
	})
	tools, err := filter.FilterTools(context.Background(), agentruntime.RuntimeToolFilterRequest{
		Scope: agentruntime.RuntimeRequestScope{WorkspaceID: "ws_1", AgentID: "explore", UserID: "user_1"},
		Instance: agentruntime.AgentInstance{
			Tools: []agentruntime.AgentTool{
				issue11017ToolSnapshot(t, agenttools.ToolKindSearchRAGChunks),
				issue11017ToolSnapshot(t, agenttools.ToolKindSelectFinalSources),
				issue11017ToolSnapshot(t, agenttools.ToolKindSubmitFinalAnswer),
			},
		},
		Metadata: map[string]any{
			"semantic_model_ids": []any{float64(6)},
		},
	})
	if err != nil {
		t.Fatalf("FilterTools() error = %v", err)
	}
	instruction, err := filter.FilterInstruction(context.Background(), agentruntime.RuntimeInstructionFilterRequest{
		Scope: agentruntime.RuntimeRequestScope{WorkspaceID: "ws_1", AgentID: "explore", UserID: "user_1"},
		Instance: agentruntime.AgentInstance{
			Tools: tools,
		},
		Metadata: map[string]any{
			"semantic_model_ids": []any{float64(6)},
		},
	})
	if err != nil {
		t.Fatalf("FilterInstruction() error = %v", err)
	}
	if !containsAll(instruction.SystemPrompt, "document corpus: available through registered document tools: `search_rag_chunks`", "For evidence extraction, call `search_rag_chunks`", "`keywords`") {
		t.Fatalf("RAG prompt missing registered search_rag_chunks guidance:\n%s", instruction.SystemPrompt)
	}
	if strings.Contains(instruction.SystemPrompt, "keywards") {
		t.Fatalf("RAG prompt uses obsolete keywards argument:\n%s", instruction.SystemPrompt)
	}
	if containsAny(instruction.SystemPrompt, "find_rag_files", "read_parsed_markdown", "search_parsed_markdown") {
		t.Fatalf("RAG prompt names unavailable document tools:\n%s", instruction.SystemPrompt)
	}
}

func TestPlatformKnowledgeToolFilterKeepsCustomPrompt(t *testing.T) {
	filter := NewPlatformKnowledgeToolFilter(&PlatformKnowledgeScopeResolver{})
	instruction, err := filter.FilterInstruction(context.Background(), agentruntime.RuntimeInstructionFilterRequest{
		Instance: agentruntime.AgentInstance{
			Instruction: agentruntime.AgentInstruction{
				SystemPrompt: "custom knowledge explore routing with search_rag_chunks",
			},
			Tools: []agentruntime.AgentTool{
				issue11017ToolSnapshot(t, agenttools.ToolKindSelectFinalSources),
				issue11017ToolSnapshot(t, agenttools.ToolKindSubmitFinalAnswer),
			},
		},
	})
	if err != nil {
		t.Fatalf("FilterInstruction() error = %v", err)
	}
	if instruction.SystemPrompt != "custom knowledge explore routing with search_rag_chunks" {
		t.Fatalf("custom prompt was changed: %q", instruction.SystemPrompt)
	}
}

func TestPlatformKnowledgeToolFilterKeepsRulesAppendedToLegacyDefaultPrompt(t *testing.T) {
	filter := NewPlatformKnowledgeToolFilter(&PlatformKnowledgeScopeResolver{})
	for _, locale := range []string{"zh-CN", "en-US"} {
		t.Run(locale, func(t *testing.T) {
			legacyPrompt := platformKnowledgeFilterTestLegacyKnowledgeExplorePrompt(t, locale)
			if !platformKnowledgeExploreUsesDefaultPrompt(legacyPrompt) {
				t.Fatalf("legacy %s default prompt was not recognized", locale)
			}
			rule := "User-added rule: preserve this instruction."
			if locale == "zh-CN" {
				rule = "用户追加规则：保留此指令。"
			}
			promptWithRule := legacyPrompt + "\n\n" + rule
			if platformKnowledgeExploreUsesDefaultPrompt(promptWithRule) {
				t.Fatalf("legacy %s prompt with an appended rule was recognized as default", locale)
			}

			instruction, err := filter.FilterInstruction(context.Background(), agentruntime.RuntimeInstructionFilterRequest{
				Instance: agentruntime.AgentInstance{
					Instruction: agentruntime.AgentInstruction{SystemPrompt: promptWithRule},
					Tools: []agentruntime.AgentTool{
						issue11017ToolSnapshot(t, agenttools.ToolKindSelectFinalSources),
					},
				},
			})
			if err != nil {
				t.Fatalf("FilterInstruction() error = %v", err)
			}
			if instruction.SystemPrompt != promptWithRule {
				t.Fatalf("legacy %s prompt with an appended rule was changed:\n%s", locale, instruction.SystemPrompt)
			}
		})
	}
}

func TestPlatformKnowledgeToolFilterRewritesChineseDefaultPromptForFilteredTools(t *testing.T) {
	filter := NewPlatformKnowledgeToolFilter(&PlatformKnowledgeScopeResolver{
		semantic: platformKnowledgeFilterTestSemanticStore{
			model: &tenant.SemanticModelRecord{
				ID:     7,
				Name:   "empty kb",
				Tables: json.RawMessage(`[]`),
				Files:  json.RawMessage(`{}`),
			},
		},
	})
	instance := platformKnowledgeFilterTestDescriptor(t)
	instance.Instruction.SystemPrompt = platformKnowledgeFilterTestLegacyKnowledgeExplorePrompt(t, "zh-CN")
	instruction, err := filter.FilterInstruction(context.Background(), agentruntime.RuntimeInstructionFilterRequest{
		Scope:    agentruntime.RuntimeRequestScope{WorkspaceID: "ws_1", AgentID: "explore", UserID: "user_1"},
		Instance: instance,
		Metadata: map[string]any{"semantic_model_ids": []any{float64(7)}},
	})
	if err != nil {
		t.Fatalf("FilterInstruction() error = %v", err)
	}
	if containsAny(instruction.SystemPrompt, "find_rag_files", "search_rag_chunks", "read_parsed_markdown", "search_parsed_markdown") {
		t.Fatalf("filtered Chinese default prompt mentions unavailable RAG tools:\n%s", instruction.SystemPrompt)
	}
}

func TestPlatformKnowledgeScopeResolverDoesNotResolveLineageWhenSemanticImageIndexIsComplete(t *testing.T) {
	files, err := json.Marshal(map[string]any{
		"file_ids":                   []string{"file_1"},
		"vector_table":               "text_vec",
		"embedding_model":            "bge-m3",
		"image_vector_table":         "kb_drawing_img",
		"image_embedding_model":      "efficientnet-b3",
		"image_embedding_backend_id": "local-efficientnet-b3",
		"image_embedding_dimension":  1536,
		"image_preprocess_version":   "efficientnet-b3-v1",
		"image_distance_metric":      "cosine",
	})
	if err != nil {
		t.Fatalf("marshal files: %v", err)
	}
	image := &platformKnowledgeFilterTestImageIndexResolver{
		result: &PlatformKnowledgeImageIndexResolveResult{
			ImageVectorTable:        "other_img",
			ImageEmbeddingModel:     "efficientnet-b3",
			ImageEmbeddingDimension: 1536,
			ImagePreprocessVersion:  "efficientnet-b3-v1",
			ImageDistanceMetric:     "cosine",
		},
	}
	filter := NewPlatformKnowledgeToolFilter(&PlatformKnowledgeScopeResolver{
		semantic: platformKnowledgeFilterTestSemanticStore{
			model: &tenant.SemanticModelRecord{
				ID:     7,
				Name:   "drawing kb",
				Tables: json.RawMessage(`[]`),
				Files:  files,
			},
		},
		image: image,
	})
	tools, err := filter.FilterTools(context.Background(), agentruntime.RuntimeToolFilterRequest{
		Scope:    agentruntime.RuntimeRequestScope{WorkspaceID: "ws_1", AgentID: "explore", UserID: "user_1"},
		Instance: platformKnowledgeFilterTestDescriptor(t),
		Metadata: map[string]any{
			"semantic_model_ids": []any{float64(7)},
		},
	})
	if err != nil {
		t.Fatalf("FilterTools() error = %v", err)
	}
	if image.calls != 0 {
		t.Fatalf("image resolver calls = %d, want 0", image.calls)
	}
	if !platformKnowledgeFilterTestContains(tools, agenttools.ToolKindSearchVisualImage) {
		t.Fatalf("drawing scope missing search_visual_image: %+v", tools)
	}
}

func TestPlatformKnowledgeToolFilterVisualPromptUsesQueryVisual(t *testing.T) {
	files, err := json.Marshal(map[string]any{
		"file_ids":                   []string{"file_1"},
		"vector_table":               "text_vec",
		"embedding_model":            "bge-m3",
		"image_vector_table":         "kb_drawing_img",
		"image_embedding_model":      "efficientnet-b3",
		"image_embedding_backend_id": "local-efficientnet-b3",
		"image_embedding_dimension":  1536,
		"image_preprocess_version":   "efficientnet-b3-v1",
		"image_distance_metric":      "cosine",
	})
	if err != nil {
		t.Fatalf("marshal files: %v", err)
	}
	filter := NewPlatformKnowledgeToolFilter(&PlatformKnowledgeScopeResolver{
		semantic: platformKnowledgeFilterTestSemanticStore{
			model: &tenant.SemanticModelRecord{
				ID:     8,
				Name:   "visual kb",
				Tables: json.RawMessage(`[]`),
				Files:  files,
			},
		},
	})
	tools, err := filter.FilterTools(context.Background(), agentruntime.RuntimeToolFilterRequest{
		Scope:    agentruntime.RuntimeRequestScope{WorkspaceID: "ws_1", AgentID: "explore", UserID: "user_1"},
		Instance: platformKnowledgeFilterTestDescriptor(t),
		Metadata: map[string]any{
			"semantic_model_ids": []any{float64(8)},
		},
	})
	if err != nil {
		t.Fatalf("FilterTools() error = %v", err)
	}
	instruction, err := filter.FilterInstruction(context.Background(), agentruntime.RuntimeInstructionFilterRequest{
		Scope: agentruntime.RuntimeRequestScope{WorkspaceID: "ws_1", AgentID: "explore", UserID: "user_1"},
		Instance: agentruntime.AgentInstance{
			Tools: tools,
		},
		Metadata: map[string]any{
			"semantic_model_ids": []any{float64(8)},
		},
	})
	if err != nil {
		t.Fatalf("FilterInstruction() error = %v", err)
	}
	if !containsAll(instruction.SystemPrompt, "search_visual_image", "query_visual") {
		t.Fatalf("visual prompt missing query_visual guidance:\n%s", instruction.SystemPrompt)
	}
	if strings.Contains(instruction.SystemPrompt, "query_image") {
		t.Fatalf("visual prompt contains unsupported query_image parameter:\n%s", instruction.SystemPrompt)
	}
}

func TestPlatformKnowledgeScopeResolverProjectsActiveImageIndexConfig(t *testing.T) {
	files, err := json.Marshal(map[string]any{
		"file_ids":                     []string{"file_1"},
		"vector_table":                 "text_vec",
		"embedding_model":              "bge-m3",
		"active_image_index_config_id": "clip_v2",
		"image_index_configs": []map[string]any{
			{
				"id":                         "efficientnet_v1",
				"image_vector_table":         "kb_img_efficientnet",
				"image_embedding_model":      "efficientnet-b3",
				"image_embedding_backend_id": "local-efficientnet-b3",
				"image_embedding_dimension":  1536,
				"image_preprocess_version":   "efficientnet-b3-v1",
				"image_distance_metric":      "cosine",
			},
			{
				"id":                         "clip_v2",
				"image_vector_table":         "kb_img_clip",
				"image_embedding_model":      "clip-vit-large",
				"image_embedding_backend_id": "local-clip",
				"image_embedding_dimension":  768,
				"image_preprocess_version":   "clip-v2-rgb-224",
				"image_distance_metric":      "cosine",
			},
		},
	})
	if err != nil {
		t.Fatalf("marshal files: %v", err)
	}
	image := &platformKnowledgeFilterTestImageIndexResolver{
		result: &PlatformKnowledgeImageIndexResolveResult{
			ImageVectorTable:        "other_img",
			ImageEmbeddingModel:     "efficientnet-b3",
			ImageEmbeddingDimension: 1536,
			ImagePreprocessVersion:  "efficientnet-b3-v1",
			ImageDistanceMetric:     "cosine",
		},
	}
	resolver := &PlatformKnowledgeScopeResolver{
		semantic: platformKnowledgeFilterTestSemanticStore{
			model: &tenant.SemanticModelRecord{
				ID:     8,
				Name:   "multi image kb",
				Tables: json.RawMessage(`[]`),
				Files:  files,
			},
		},
		image: image,
	}

	scope, err := resolver.ResolveKnowledgeScope(context.Background(), knowledge.WorkspaceScope{
		WorkspaceID:      "ws_1",
		UserID:           "user_1",
		SemanticModelIDs: []int64{8},
	})
	if err != nil {
		t.Fatalf("ResolveKnowledgeScope() error = %v", err)
	}
	if image.calls != 0 {
		t.Fatalf("image resolver calls = %d, want 0", image.calls)
	}
	if len(scope.RAGSources) != 1 {
		t.Fatalf("rag sources = %#v, want one source", scope.RAGSources)
	}
	got := scope.RAGSources[0]
	if got.ImageVectorTable != "kb_img_clip" || got.ImageEmbeddingModel != "clip-vit-large" || got.ImageEmbeddingBackendID != "local-clip" || got.ImageEmbeddingDimension != 768 || got.ImagePreprocessVersion != "clip-v2-rgb-224" || got.ImageDistanceMetric != "cosine" {
		t.Fatalf("rag source = %#v, want active image index config projected", got)
	}
	if scope.ImageVectorTable != "kb_img_clip" || scope.ImageEmbeddingModel != "clip-vit-large" || scope.ImageEmbeddingBackendID != "local-clip" || scope.ImageEmbeddingDimension != 768 || scope.ImagePreprocessVersion != "clip-v2-rgb-224" || scope.ImageDistanceMetric != "cosine" {
		t.Fatalf("scope = %#v, want active image index config projected", scope)
	}
}

func TestPlatformKnowledgeScopeResolverSuppressesPendingActiveImageIndexConfig(t *testing.T) {
	files, err := json.Marshal(map[string]any{
		"file_ids":                     []string{"file_1"},
		"vector_table":                 "text_vec",
		"embedding_model":              "bge-m3",
		"image_vector_table":           "legacy_img",
		"image_embedding_model":        "efficientnet-b3",
		"image_embedding_backend_id":   "legacy-efficientnet-b3",
		"image_embedding_dimension":    1536,
		"image_preprocess_version":     "efficientnet-b3-v1",
		"image_distance_metric":        "cosine",
		"active_image_index_config_id": "pending_cfg",
		"image_index_configs": []map[string]any{
			{
				"id":                         "pending_cfg",
				"image_vector_table":         "missing_active_img",
				"image_embedding_model":      "efficientnet-b3",
				"image_embedding_backend_id": "local-efficientnet-b3",
				"image_embedding_dimension":  1536,
				"image_preprocess_version":   "efficientnet-b3-v1",
				"image_distance_metric":      "cosine",
				"status":                     "pending",
			},
		},
	})
	if err != nil {
		t.Fatalf("marshal files: %v", err)
	}
	image := &platformKnowledgeFilterTestImageIndexResolver{
		result: &PlatformKnowledgeImageIndexResolveResult{
			ImageVectorTable:        "legacy_img",
			ImageEmbeddingModel:     "efficientnet-b3",
			ImageEmbeddingBackendID: "legacy-efficientnet-b3",
			ImageEmbeddingDimension: 1536,
			ImagePreprocessVersion:  "efficientnet-b3-v1",
			ImageDistanceMetric:     "cosine",
		},
	}
	resolver := &PlatformKnowledgeScopeResolver{
		semantic: platformKnowledgeFilterTestSemanticStore{
			model: &tenant.SemanticModelRecord{
				ID:     11,
				Name:   "pending image kb",
				Tables: json.RawMessage(`[]`),
				Files:  files,
			},
		},
		image: image,
	}

	scope, err := resolver.ResolveKnowledgeScope(context.Background(), knowledge.WorkspaceScope{
		WorkspaceID:      "ws_1",
		UserID:           "user_1",
		SemanticModelIDs: []int64{11},
	})
	if err != nil {
		t.Fatalf("ResolveKnowledgeScope() error = %v", err)
	}
	if image.calls != 0 {
		t.Fatalf("image resolver calls = %d, want 0", image.calls)
	}
	if len(scope.RAGSources) != 1 {
		t.Fatalf("rag sources = %#v, want one text source", scope.RAGSources)
	}
	got := scope.RAGSources[0]
	if got.ImageVectorTable != "" || got.ImageEmbeddingModel != "" || got.ImageEmbeddingBackendID != "" || got.ImageEmbeddingDimension != 0 || got.ImagePreprocessVersion != "" || got.ImageDistanceMetric != "" {
		t.Fatalf("rag source = %#v, want pending active image config to suppress visual index", got)
	}
	if scope.ImageVectorTable != "" || scope.ImageEmbeddingModel != "" || scope.ImageEmbeddingBackendID != "" || scope.ImageEmbeddingDimension != 0 || scope.ImagePreprocessVersion != "" || scope.ImageDistanceMetric != "" {
		t.Fatalf("scope = %#v, want pending active image config to suppress visual index", scope)
	}

	filter := NewPlatformKnowledgeToolFilter(resolver)
	tools, err := filter.FilterTools(context.Background(), agentruntime.RuntimeToolFilterRequest{
		Scope:    agentruntime.RuntimeRequestScope{WorkspaceID: "ws_1", AgentID: "explore", UserID: "user_1"},
		Instance: platformKnowledgeFilterTestDescriptor(t),
		Metadata: map[string]any{
			"semantic_model_ids": []any{float64(11)},
		},
	})
	if err != nil {
		t.Fatalf("FilterTools() error = %v", err)
	}
	if platformKnowledgeFilterTestContains(tools, agenttools.ToolKindSearchVisualImage) {
		t.Fatalf("pending active image config exposed search_visual_image: %+v", tools)
	}
	if !platformKnowledgeFilterTestContains(tools, agenttools.ToolKindSearchRAGChunks) {
		t.Fatalf("pending active image config removed text RAG tool: %+v", tools)
	}
}

func TestPlatformKnowledgeScopeResolverRejectsMissingActiveImageIndexConfig(t *testing.T) {
	files, err := json.Marshal(map[string]any{
		"file_ids":                     []string{"file_1"},
		"vector_table":                 "text_vec",
		"embedding_model":              "bge-m3",
		"active_image_index_config_id": "missing",
		"image_index_configs": []map[string]any{
			{
				"id":                        "clip_v2",
				"image_vector_table":        "kb_img_clip",
				"image_embedding_model":     "clip-vit-large",
				"image_embedding_dimension": 768,
				"image_preprocess_version":  "clip-v2-rgb-224",
				"image_distance_metric":     "cosine",
			},
		},
	})
	if err != nil {
		t.Fatalf("marshal files: %v", err)
	}
	image := &platformKnowledgeFilterTestImageIndexResolver{}
	resolver := &PlatformKnowledgeScopeResolver{
		semantic: platformKnowledgeFilterTestSemanticStore{
			model: &tenant.SemanticModelRecord{
				ID:     9,
				Name:   "multi image kb",
				Tables: json.RawMessage(`[]`),
				Files:  files,
			},
		},
		image: image,
	}

	_, err = resolver.ResolveKnowledgeScope(context.Background(), knowledge.WorkspaceScope{
		WorkspaceID:      "ws_1",
		UserID:           "user_1",
		SemanticModelIDs: []int64{9},
	})
	if err == nil || !strings.Contains(err.Error(), `active_image_index_config_id "missing" not found`) {
		t.Fatalf("ResolveKnowledgeScope() error = %v, want missing active config error", err)
	}
	if image.calls != 0 {
		t.Fatalf("image resolver calls = %d, want 0", image.calls)
	}
}

func TestPlatformKnowledgeScopeResolverRejectsIncompleteActiveImageIndexConfig(t *testing.T) {
	files, err := json.Marshal(map[string]any{
		"file_ids":                     []string{"file_1"},
		"vector_table":                 "text_vec",
		"embedding_model":              "bge-m3",
		"active_image_index_config_id": "clip_v2",
		"image_index_configs": []map[string]any{
			{
				"id":                    "clip_v2",
				"image_vector_table":    "kb_img_clip",
				"image_embedding_model": "clip-vit-large",
				"image_distance_metric": "cosine",
			},
		},
	})
	if err != nil {
		t.Fatalf("marshal files: %v", err)
	}
	image := &platformKnowledgeFilterTestImageIndexResolver{}
	resolver := &PlatformKnowledgeScopeResolver{
		semantic: platformKnowledgeFilterTestSemanticStore{
			model: &tenant.SemanticModelRecord{
				ID:     10,
				Name:   "multi image kb",
				Tables: json.RawMessage(`[]`),
				Files:  files,
			},
		},
		image: image,
	}

	_, err = resolver.ResolveKnowledgeScope(context.Background(), knowledge.WorkspaceScope{
		WorkspaceID:      "ws_1",
		UserID:           "user_1",
		SemanticModelIDs: []int64{10},
	})
	if err == nil || !strings.Contains(err.Error(), "incomplete image index config") {
		t.Fatalf("ResolveKnowledgeScope() error = %v, want incomplete active config error", err)
	}
	if image.calls != 0 {
		t.Fatalf("image resolver calls = %d, want 0", image.calls)
	}
}

func TestPlatformKnowledgeScopeResolverCompletesPartialSemanticImageIndexFromLineage(t *testing.T) {
	files, err := json.Marshal(map[string]any{
		"file_ids":              []string{"file_1"},
		"vector_table":          "text_vec",
		"embedding_model":       "bge-m3",
		"image_vector_table":    "kb_drawing_img",
		"image_embedding_model": "efficientnet-b3",
	})
	if err != nil {
		t.Fatalf("marshal files: %v", err)
	}
	image := &platformKnowledgeFilterTestImageIndexResolver{
		result: &PlatformKnowledgeImageIndexResolveResult{
			ImageVectorTable:        "kb_drawing_img",
			ImageEmbeddingModel:     "efficientnet-b3",
			ImageEmbeddingBackendID: "local-efficientnet-b3",
			ImageEmbeddingDimension: 1536,
			ImagePreprocessVersion:  "efficientnet-b3-v1",
			ImageDistanceMetric:     "cosine",
		},
	}
	resolver := &PlatformKnowledgeScopeResolver{
		semantic: platformKnowledgeFilterTestSemanticStore{
			model: &tenant.SemanticModelRecord{
				ID:     7,
				Name:   "drawing kb",
				Tables: json.RawMessage(`[]`),
				Files:  files,
			},
		},
		image: image,
	}

	scope, err := resolver.ResolveKnowledgeScope(context.Background(), knowledge.WorkspaceScope{
		WorkspaceID:      "ws_1",
		UserID:           "user_1",
		SemanticModelIDs: []int64{7},
	})
	if err != nil {
		t.Fatalf("ResolveKnowledgeScope() error = %v", err)
	}
	if image.calls != 1 {
		t.Fatalf("image resolver calls = %d, want 1", image.calls)
	}
	if image.req.WorkspaceID != "ws_1" || image.req.UserID != "user_1" || image.req.SemanticModelID != 7 || image.req.ImageVectorTable != "kb_drawing_img" || image.req.ImageEmbeddingModel != "efficientnet-b3" || len(image.req.FileIDs) != 1 || image.req.FileIDs[0] != "file_1" {
		t.Fatalf("image resolver req = %#v, want semantic model scope", image.req)
	}
	if len(scope.RAGSources) != 1 {
		t.Fatalf("rag sources = %#v, want one source", scope.RAGSources)
	}
	got := scope.RAGSources[0]
	if got.ImageVectorTable != "kb_drawing_img" || got.ImageEmbeddingModel != "efficientnet-b3" || got.ImageEmbeddingDimension != 1536 || got.ImagePreprocessVersion != "efficientnet-b3-v1" || got.ImageDistanceMetric != "cosine" {
		t.Fatalf("rag source = %#v, want image index metadata completed from lineage", got)
	}
}

func TestPlatformKnowledgeScopeResolverDoesNotDiscoverTextIndexWithoutSemanticVectorBinding(t *testing.T) {
	files, err := json.Marshal(map[string]any{
		"file_ids":        []string{"file_1"},
		"embedding_model": "bge-m3",
	})
	if err != nil {
		t.Fatalf("marshal files: %v", err)
	}
	vector := &platformKnowledgeFilterTestVectorIndexResolver{
		result: &PlatformKnowledgeVectorIndexResolveResult{
			VectorTable:    "other_kb_text_vec",
			EmbeddingModel: "bge-m3",
		},
	}
	resolver := &PlatformKnowledgeScopeResolver{
		semantic: platformKnowledgeFilterTestSemanticStore{
			model: &tenant.SemanticModelRecord{
				ID:     18,
				Name:   "partial text kb",
				Tables: json.RawMessage(`[]`),
				Files:  files,
			},
		},
		vector: vector,
	}

	scope, err := resolver.ResolveKnowledgeScope(context.Background(), knowledge.WorkspaceScope{
		WorkspaceID:      "ws_1",
		UserID:           "user_1",
		SemanticModelIDs: []int64{18},
	})
	if err != nil {
		t.Fatalf("ResolveKnowledgeScope() error = %v", err)
	}
	if vector.calls != 0 {
		t.Fatalf("vector resolver calls = %d, want 0 without semantic model vector_table", vector.calls)
	}
	if len(scope.RAGSources) != 0 {
		t.Fatalf("rag scope = %#v, want no document RAG without semantic vector binding", scope.RAGSources)
	}
	if scope.VectorTable != "" || scope.EmbeddingModel != "" {
		t.Fatalf("text scope = %q/%q, want empty", scope.VectorTable, scope.EmbeddingModel)
	}
}

func TestQueryPlatformKnowledgeSourceGovernanceRequiresEnabledForForcedExpired(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()
	rows := sqlmock.NewRows([]string{"source_id", "kb_file_id", "status", "enabled", "expires_at", "force_enabled_after_expiry", "tags", "segment_version_id", "index_version"}).
		AddRow("source_enabled", "file_enabled", "succeeded", true, int64(0), false, `[" policy ","policy",""]`, nil, nil).
		AddRow("source_forced", "file_forced", "succeeded", true, int64(100), true, `["forced"]`, nil, nil).
		AddRow("source_disabled_forced", "file_disabled_forced", "succeeded", false, int64(100), true, `["disabled"]`, nil, nil)
	mock.ExpectQuery("SELECT source_id, kb_file_id, status, COALESCE\\(enabled, 1\\), expires_at, force_enabled_after_expiry, tags, segment_version_id, index_version").
		WithArgs(int64(12), "file_enabled", "file_forced", "file_disabled_forced").
		WillReturnRows(rows)

	records, err := queryPlatformKnowledgeSourceGovernance(context.Background(), db, 12, []string{"file_enabled", "file_forced", "file_disabled_forced"}, 200)
	if err != nil {
		t.Fatalf("queryPlatformKnowledgeSourceGovernance: %v", err)
	}
	got := make(map[string]PlatformKnowledgeSourceGovernanceRecord, len(records))
	for _, record := range records {
		got[record.FileID] = record
	}
	if !got["file_enabled"].EffectiveEnabled {
		t.Fatalf("enabled source effective = %+v", got["file_enabled"])
	}
	if !got["file_forced"].EffectiveEnabled {
		t.Fatalf("forced expired source effective = %+v", got["file_forced"])
	}
	if got["file_disabled_forced"].EffectiveEnabled {
		t.Fatalf("disabled forced source effective = %+v", got["file_disabled_forced"])
	}
	if want := []string{" policy ", "policy", ""}; !reflect.DeepEqual(got["file_enabled"].Tags, want) {
		t.Fatalf("enabled source tags = %#v, want %#v", got["file_enabled"].Tags, want)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestQueryPlatformKnowledgeTableSourceGovernanceReadsResolvedTableIdentity(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()
	rows := sqlmock.NewRows([]string{"source_id", "db_name", "table_name", "source_db_name", "source_table_name", "source_table_id", "kb_table_id", "status", "enabled", "expires_at", "force_enabled_after_expiry"}).
		AddRow("source_disabled", "kb_sales", "orders", "sales", "raw_orders", "101", "201", "succeeded", false, nil, false).
		AddRow("source_enabled", "kb_sales", "customers", "sales", "raw_customers", "102", "202", "succeeded", true, nil, false).
		AddRow("source_expired", "kb_sales", "expired_orders", "", "", "", "", "succeeded", true, int64(1000), false).
		AddRow("source_forced_expired", "kb_sales", "forced_orders", "", "", "", "", "succeeded", true, int64(1000), true)
	mock.ExpectQuery("(?s)SELECT\\s+kbs\\.source_id,.*kbs\\.status <> 'removed'").
		WithArgs(int64(16)).
		WillReturnRows(rows)

	records, err := queryPlatformKnowledgeTableSourceGovernance(context.Background(), db, 16, 2000)
	if err != nil {
		t.Fatalf("queryPlatformKnowledgeTableSourceGovernance: %v", err)
	}
	if len(records) != 4 {
		t.Fatalf("records = %#v, want 4", records)
	}
	if records[0].DBName != "kb_sales" || records[0].TableName != "orders" || records[0].SourceDBName != "sales" || records[0].SourceTableName != "raw_orders" || records[0].KBTableID != "201" || records[0].Enabled || records[0].EffectiveEnabled {
		t.Fatalf("disabled table record = %#v", records[0])
	}
	if records[1].DBName != "kb_sales" || records[1].TableName != "customers" || records[1].SourceDBName != "sales" || records[1].SourceTableName != "raw_customers" || records[1].SourceTableID != "102" || !records[1].Enabled || !records[1].EffectiveEnabled {
		t.Fatalf("enabled table record = %#v", records[1])
	}
	if !records[2].Expired || records[2].EffectiveEnabled {
		t.Fatalf("expired table record = %#v", records[2])
	}
	if !records[3].Expired || !records[3].ForceEnabledAfterExpiry || !records[3].EffectiveEnabled {
		t.Fatalf("forced expired table record = %#v", records[3])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestPlatformKnowledgeSourceTableRefsUseCatalogSourceTableIdentity(t *testing.T) {
	store := platformKnowledgeFilterTestGovernanceStore{
		tableRecords: []PlatformKnowledgeTableSourceGovernanceRecord{
			{
				SourceRowID:      "source_resume_candidates",
				DBName:           "简历知识库_1",
				TableName:        "resume_candidates",
				SourceDBName:     "cv_demo_db",
				SourceTableName:  "resume_candidates",
				SourceTableID:    "101",
				KBTableID:        "201",
				Status:           "succeeded",
				EffectiveEnabled: true,
			},
		},
	}
	hook := platformKnowledgeSourceTableRefs(store)
	if hook == nil {
		t.Fatalf("expected table refs hook")
	}
	ctx := knowledge.ContextWithScope(context.Background(), knowledge.WorkspaceScope{
		WorkspaceID:      "ws_1",
		SemanticModelIDs: []int64{16},
	})
	refs, err := hook(ctx, []string{"resume_candidates"}, "简历知识库_1")
	if err != nil {
		t.Fatalf("table refs hook returned err: %v", err)
	}
	if len(refs) != 1 {
		t.Fatalf("refs = %#v, want 1", refs)
	}
	if refs[0].DBName != "cv_demo_db" || refs[0].Name != "resume_candidates" {
		t.Fatalf("refs[0] = %#v, want cv_demo_db.resume_candidates", refs[0])
	}
}

func TestQueryPlatformKnowledgeSourceGovernanceAllowsEmptyFileIDs(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()
	rows := sqlmock.NewRows([]string{"source_id", "kb_file_id", "status", "enabled", "expires_at", "force_enabled_after_expiry", "tags", "segment_version_id", "index_version"}).
		AddRow("source_pdf", "pdf_file", "succeeded", true, int64(0), false, nil, "seg_pdf", int64(3)).
		AddRow("source_docx", "default_docx", "succeeded", true, int64(0), false, nil, "seg_docx", int64(2))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT source_id, kb_file_id, status, COALESCE(enabled, 1), expires_at, force_enabled_after_expiry, tags, segment_version_id, index_version\nFROM knowledge_base_sources\nWHERE model_id = ? AND kb_file_id IS NOT NULL AND kb_file_id != '' AND status <> 'removed' AND source_type IN ('local_file', 'catalog_file')")).
		WithArgs(int64(15)).
		WillReturnRows(rows)

	records, err := queryPlatformKnowledgeSourceGovernance(context.Background(), db, 15, nil, 200)
	if err != nil {
		t.Fatalf("queryPlatformKnowledgeSourceGovernance: %v", err)
	}
	if len(records) != 2 || records[0].FileID != "pdf_file" || records[1].FileID != "default_docx" {
		t.Fatalf("records = %+v, want all source file records", records)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

type platformKnowledgeFilterTestConnectionPool struct {
	db *sql.DB
}

func (p *platformKnowledgeFilterTestConnectionPool) GetConnection(context.Context, string) (*sql.DB, error) {
	return p.db, nil
}

func (p *platformKnowledgeFilterTestConnectionPool) GetDBExecutor(context.Context, string) (tenant.DBExecutor, error) {
	return p.db, nil
}

func (p *platformKnowledgeFilterTestConnectionPool) GetTransactionManager(context.Context, string) (*transaction.Manager, error) {
	return transaction.NewManager(p.db), nil
}

func (p *platformKnowledgeFilterTestConnectionPool) GetTx(ctx context.Context, _ string) (*sql.Tx, error) {
	return p.db.BeginTx(ctx, nil)
}

func (p *platformKnowledgeFilterTestConnectionPool) Close() error {
	return nil
}

func TestCatalogKnowledgeSourceGovernanceStoreListSourceGovernanceAllowsEmptyFileIDs(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()
	rows := sqlmock.NewRows([]string{"source_id", "kb_file_id", "status", "enabled", "expires_at", "force_enabled_after_expiry", "tags", "segment_version_id", "index_version"}).
		AddRow("source_pdf", "pdf_file", "succeeded", true, int64(0), false, nil, "seg_pdf", int64(3))
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT source_id, kb_file_id, status, COALESCE\\(enabled, 1\\), expires_at, force_enabled_after_expiry, tags, segment_version_id, index_version").
		WithArgs(int64(15)).
		WillReturnRows(rows)
	mock.ExpectCommit()

	store := newCatalogKnowledgeSourceGovernanceStore(&platformKnowledgeFilterTestConnectionPool{db: db})
	records, err := store.ListSourceGovernance(context.Background(), "ws_1", 15, nil)
	if err != nil {
		t.Fatalf("ListSourceGovernance: %v", err)
	}
	if len(records) != 1 || records[0].FileID != "pdf_file" || !records[0].EffectiveEnabled {
		t.Fatalf("records = %+v, want queried enabled source file record", records)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestPlatformKnowledgeScopeResolverAppliesDocumentGovernance(t *testing.T) {
	files, err := json.Marshal(map[string]any{
		"file_ids":                  []string{"file_enabled", "file_disabled", "file_forced", "file_legacy", "file_unmanaged"},
		"vector_table":              "text_vec",
		"embedding_model":           "bge-m3",
		"image_vector_table":        "img_vec",
		"image_embedding_model":     "clip",
		"image_embedding_dimension": 768,
		"image_preprocess_version":  "clip-v1",
		"image_distance_metric":     "cosine",
	})
	if err != nil {
		t.Fatalf("marshal files: %v", err)
	}
	resolver := &PlatformKnowledgeScopeResolver{
		semantic: platformKnowledgeFilterTestSemanticStore{
			model: &tenant.SemanticModelRecord{
				ID:     12,
				Name:   "governed kb",
				Tables: json.RawMessage(`[]`),
				Files:  files,
			},
		},
		governance: platformKnowledgeFilterTestGovernanceStore{
			records: []PlatformKnowledgeSourceGovernanceRecord{
				{SourceRowID: "source_enabled", FileID: "file_enabled", Status: "succeeded", EffectiveEnabled: true, Tags: []string{" policy ", "policy", ""}, SegmentVersionID: "seg_v1", IndexVersion: 1},
				{SourceRowID: "source_disabled", FileID: "file_disabled", Status: "succeeded", EffectiveEnabled: false, Tags: []string{"disabled"}},
				{SourceRowID: "source_forced", FileID: "file_forced", Status: "succeeded", EffectiveEnabled: true, Expired: true, ForceEnabledAfterExpiry: true, Tags: []string{"forced", " forced "}, SegmentVersionID: "seg_v2", IndexVersion: 2},
				{SourceRowID: "source_legacy", FileID: "file_legacy", Status: "succeeded", EffectiveEnabled: true, Tags: []string{"legacy"}},
			},
		},
	}

	scope, err := resolver.ResolveKnowledgeScope(context.Background(), knowledge.WorkspaceScope{
		WorkspaceID:      "ws_1",
		UserID:           "user_1",
		SemanticModelIDs: []int64{12},
	})
	if err != nil {
		t.Fatalf("ResolveKnowledgeScope() error = %v", err)
	}
	if len(scope.FileIDs) != 4 || !containsAll(strings.Join(scope.FileIDs, ","), "file_enabled", "file_forced", "file_legacy", "file_unmanaged") || containsAny(strings.Join(scope.FileIDs, ","), "file_disabled") {
		t.Fatalf("scope file ids = %#v, want ready governed files plus explicit legacy file", scope.FileIDs)
	}
	if len(scope.RAGSources) != 1 {
		t.Fatalf("rag sources = %#v, want one governed source", scope.RAGSources)
	}
	got := scope.RAGSources[0]
	if len(got.FileIDs) != 4 || !containsAll(strings.Join(got.FileIDs, ","), "file_enabled", "file_forced", "file_legacy", "file_unmanaged") || containsAny(strings.Join(got.FileIDs, ","), "file_disabled") {
		t.Fatalf("rag source file ids = %#v, want disabled filtered and explicit legacy file retained", got.FileIDs)
	}
	if len(got.SourceRowIDs) != 3 || !containsAll(strings.Join(got.SourceRowIDs, ","), "source_enabled", "source_forced", "source_legacy") {
		t.Fatalf("source row ids = %#v", got.SourceRowIDs)
	}
	if got.Metadata["governance_mode"] != platformKnowledgeGovernanceModeLegacy {
		t.Fatalf("metadata = %#v, want legacy governance mode", got.Metadata)
	}
	if got.SourceRowIDByFileID["file_enabled"] != "source_enabled" || got.SourceRowIDByFileID["file_forced"] != "source_forced" || got.SourceRowIDByFileID["file_legacy"] != "source_legacy" {
		t.Fatalf("source row id by file = %#v", got.SourceRowIDByFileID)
	}
	if got.CurrentSegmentVersionByFileID["file_enabled"] != "seg_v1" || got.CurrentSegmentVersionByFileID["file_forced"] != "seg_v2" {
		t.Fatalf("current segment versions = %#v", got.CurrentSegmentVersionByFileID)
	}
	if got.CurrentIndexVersionByFileID["file_enabled"] != 1 || got.CurrentIndexVersionByFileID["file_forced"] != 2 {
		t.Fatalf("current index versions = %#v", got.CurrentIndexVersionByFileID)
	}
	if _, ok := got.CurrentIndexVersionByFileID["file_legacy"]; ok {
		t.Fatalf("legacy file without current index version should not get version constraint: %#v", got.CurrentIndexVersionByFileID)
	}
	wantTags := []string{" policy ", "policy", "", "forced", " forced ", "legacy"}
	if !reflect.DeepEqual(got.SourceTags, wantTags) {
		t.Fatalf("source tags = %#v, want %#v", got.SourceTags, wantTags)
	}
	if tags, want := got.SourceTagsByFileID["file_enabled"], []string{" policy ", "policy", ""}; !reflect.DeepEqual(tags, want) {
		t.Fatalf("file_enabled tags = %#v, want %#v", tags, want)
	}
	if tags, want := got.SourceTagsByFileID["file_forced"], []string{"forced", " forced "}; !reflect.DeepEqual(tags, want) {
		t.Fatalf("file_forced tags = %#v, want %#v", tags, want)
	}
	if tags, want := got.SourceTagsByFileID["file_legacy"], []string{"legacy"}; !reflect.DeepEqual(tags, want) {
		t.Fatalf("file_legacy tags = %#v, want %#v", tags, want)
	}
}

// Regression for #12764: a workflow-built KB has files.file_ids bound (by
// lineage registration) but zero knowledge_base_sources rows — source
// governance never managed it. Zero governance records must mean "no
// governance for this model", not "nothing enabled": dropping the bound file
// IDs here is what filtered out every RAG tool (tool_count=0) and made
// Knowledge Explore answer "no accessible content" over a fully indexed KB.
// Models that DO have governance rows keep the strict filter (the disabled /
// unmanaged cases pinned by TestPlatformKnowledgeScopeResolverAppliesDocumentGovernance).
func TestPlatformKnowledgeScopeResolverKeepsBoundFilesWhenNoGovernanceRecords(t *testing.T) {
	files, err := json.Marshal(map[string]any{
		"file_ids":        []string{"file_wf"},
		"vector_table":    "text_vec",
		"embedding_model": "bge-m3",
	})
	if err != nil {
		t.Fatalf("marshal files: %v", err)
	}
	resolver := &PlatformKnowledgeScopeResolver{
		semantic: platformKnowledgeFilterTestSemanticStore{
			model: &tenant.SemanticModelRecord{
				ID:     13,
				Name:   "workflow-built kb",
				Tables: json.RawMessage(`[]`),
				Files:  files,
			},
		},
		governance: platformKnowledgeFilterTestGovernanceStore{},
	}

	scope, err := resolver.ResolveKnowledgeScope(context.Background(), knowledge.WorkspaceScope{
		WorkspaceID:      "ws_1",
		UserID:           "user_1",
		SemanticModelIDs: []int64{13},
	})
	if err != nil {
		t.Fatalf("ResolveKnowledgeScope() error = %v", err)
	}
	if len(scope.RAGSources) != 1 {
		t.Fatalf("rag sources = %#v, want the ungoverned model kept", scope.RAGSources)
	}
	got := scope.RAGSources[0]
	if len(got.FileIDs) != 1 || got.FileIDs[0] != "file_wf" {
		t.Fatalf("rag source file ids = %#v, want [file_wf]", got.FileIDs)
	}
	if got.VectorTable != "text_vec" {
		t.Fatalf("rag source vector table = %q, want text_vec", got.VectorTable)
	}
	if scope.VectorTable != "text_vec" {
		t.Fatalf("resolved scope vector table = %q, want text_vec (drives hasTextRAG gating)", scope.VectorTable)
	}
}

func TestPlatformKnowledgeScopeResolverKeepsExplicitLegacyFileWithoutGovernance(t *testing.T) {
	files, err := json.Marshal(map[string]any{
		"file_ids":        []string{"legacy_file"},
		"vector_table":    "text_vec",
		"embedding_model": "bge-m3",
	})
	if err != nil {
		t.Fatalf("marshal files: %v", err)
	}
	resolver := &PlatformKnowledgeScopeResolver{
		semantic: platformKnowledgeFilterTestSemanticStore{
			model: &tenant.SemanticModelRecord{
				ID:     12,
				Name:   "legacy kb",
				Tables: json.RawMessage(`[]`),
				Files:  files,
			},
		},
		governance: platformKnowledgeFilterTestGovernanceStore{},
	}

	scope, err := resolver.ResolveKnowledgeScope(context.Background(), knowledge.WorkspaceScope{
		WorkspaceID:      "ws_1",
		UserID:           "user_1",
		SemanticModelIDs: []int64{12},
	})
	if err != nil {
		t.Fatalf("ResolveKnowledgeScope() error = %v", err)
	}
	if len(scope.RAGSources) != 1 {
		t.Fatalf("rag sources = %#v, want one legacy source", scope.RAGSources)
	}
	got := scope.RAGSources[0]
	if len(got.FileIDs) != 1 || got.FileIDs[0] != "legacy_file" {
		t.Fatalf("rag source file ids = %#v, want legacy_file", got.FileIDs)
	}
	if len(got.SourceRowIDs) != 0 || len(got.SourceRowIDByFileID) != 0 {
		t.Fatalf("source row metadata = %#v/%#v, want none for legacy file", got.SourceRowIDs, got.SourceRowIDByFileID)
	}
	if got.Metadata["governance_mode"] != platformKnowledgeGovernanceModeLegacy {
		t.Fatalf("metadata = %#v, want legacy governance mode", got.Metadata)
	}
}

func TestPlatformKnowledgeScopeResolverDoesNotLegacyBypassPendingGovernance(t *testing.T) {
	files, err := json.Marshal(map[string]any{
		"file_ids":        []string{"ready_file", "pending_file", "failed_file", "legacy_file"},
		"vector_table":    "text_vec",
		"embedding_model": "bge-m3",
	})
	if err != nil {
		t.Fatalf("marshal files: %v", err)
	}
	resolver := &PlatformKnowledgeScopeResolver{
		semantic: platformKnowledgeFilterTestSemanticStore{
			model: &tenant.SemanticModelRecord{
				ID:     12,
				Name:   "mixed kb",
				Tables: json.RawMessage(`[]`),
				Files:  files,
			},
		},
		governance: platformKnowledgeFilterTestGovernanceStore{
			records: []PlatformKnowledgeSourceGovernanceRecord{
				{SourceRowID: "source_ready", FileID: "ready_file", Status: "succeeded", EffectiveEnabled: true},
				{SourceRowID: "source_pending", FileID: "pending_file", Status: "pending", EffectiveEnabled: true},
				{SourceRowID: "source_failed", FileID: "failed_file", Status: "failed", EffectiveEnabled: true},
			},
		},
	}

	scope, err := resolver.ResolveKnowledgeScope(context.Background(), knowledge.WorkspaceScope{
		WorkspaceID:      "ws_1",
		UserID:           "user_1",
		SemanticModelIDs: []int64{12},
	})
	if err != nil {
		t.Fatalf("ResolveKnowledgeScope() error = %v", err)
	}
	if len(scope.RAGSources) != 1 {
		t.Fatalf("rag sources = %#v, want one source", scope.RAGSources)
	}
	got := scope.RAGSources[0]
	if len(got.FileIDs) != 2 || !containsAll(strings.Join(got.FileIDs, ","), "ready_file", "legacy_file") || containsAny(strings.Join(got.FileIDs, ","), "pending_file", "failed_file") {
		t.Fatalf("rag source file ids = %#v, want ready plus legacy only", got.FileIDs)
	}
	if got.SourceRowIDByFileID["ready_file"] != "source_ready" {
		t.Fatalf("source row by file = %#v, want ready source only", got.SourceRowIDByFileID)
	}
	if _, ok := got.SourceRowIDByFileID["pending_file"]; ok {
		t.Fatalf("pending source row leaked into rag source: %#v", got.SourceRowIDByFileID)
	}
	if got.Metadata["governance_mode"] != platformKnowledgeGovernanceModeLegacy {
		t.Fatalf("metadata = %#v, want legacy governance mode because legacy_file remains", got.Metadata)
	}
}

func TestPlatformKnowledgeScopeResolverHidesPendingOnlyGovernedFileWithoutMaterializedIndex(t *testing.T) {
	files, err := json.Marshal(map[string]any{
		"file_ids":        []string{"pending_file"},
		"vector_table":    "text_vec",
		"embedding_model": "bge-m3",
	})
	if err != nil {
		t.Fatalf("marshal files: %v", err)
	}
	resolver := &PlatformKnowledgeScopeResolver{
		semantic: platformKnowledgeFilterTestSemanticStore{
			model: &tenant.SemanticModelRecord{
				ID:     12,
				Name:   "pending kb",
				Tables: json.RawMessage(`[]`),
				Files:  files,
			},
		},
		governance: platformKnowledgeFilterTestGovernanceStore{
			records: []PlatformKnowledgeSourceGovernanceRecord{
				{SourceRowID: "source_pending", FileID: "pending_file", Status: "pending", EffectiveEnabled: true},
			},
		},
		legacyIndex: &platformKnowledgeFilterTestLegacyIndexResolver{},
	}

	scope, err := resolver.ResolveKnowledgeScope(context.Background(), knowledge.WorkspaceScope{
		WorkspaceID:      "ws_1",
		UserID:           "user_1",
		SemanticModelIDs: []int64{12},
	})
	if err != nil {
		t.Fatalf("ResolveKnowledgeScope() error = %v", err)
	}
	if len(scope.FileIDs) != 0 || len(scope.RAGSources) != 0 {
		t.Fatalf("scope = %+v, want no RAG files for pending-only governed source", scope)
	}
	if scope.VectorTable != "" || scope.EmbeddingModel != "" {
		t.Fatalf("top-level vector config = %q/%q, want cleared", scope.VectorTable, scope.EmbeddingModel)
	}
}

func TestPlatformKnowledgeScopeResolverUsesMaterializedPendingSourceBeforeGovernancePublish(t *testing.T) {
	files, err := json.Marshal(map[string]any{
		"file_ids":        []string{"pending_file"},
		"vector_table":    "text_vec",
		"embedding_model": "bge-m3",
	})
	if err != nil {
		t.Fatalf("marshal files: %v", err)
	}
	legacy := &platformKnowledgeFilterTestLegacyIndexResolver{
		fileIDs: []string{"other_file", "pending_file"},
		constraints: map[string]knowledge.RAGIndexVersionConstraint{
			"pending_file": {Kind: knowledge.RAGIndexVersionConstraintValue, Value: 7},
		},
	}
	resolver := &PlatformKnowledgeScopeResolver{
		semantic: platformKnowledgeFilterTestSemanticStore{
			model: &tenant.SemanticModelRecord{
				ID:     12,
				Name:   "pending kb",
				Tables: json.RawMessage(`[]`),
				Files:  files,
			},
		},
		governance: platformKnowledgeFilterTestGovernanceStore{
			records: []PlatformKnowledgeSourceGovernanceRecord{
				{SourceRowID: "source_pending", FileID: "pending_file", Status: "pending", EffectiveEnabled: true, Tags: []string{"not-yet-published"}},
			},
		},
		legacyIndex: legacy,
	}

	scope, err := resolver.ResolveKnowledgeScope(context.Background(), knowledge.WorkspaceScope{
		WorkspaceID:      "ws_1",
		UserID:           "user_1",
		SemanticModelIDs: []int64{12},
	})
	if err != nil {
		t.Fatalf("ResolveKnowledgeScope() error = %v", err)
	}
	if len(scope.FileIDs) != 1 || scope.FileIDs[0] != "pending_file" {
		t.Fatalf("scope file ids = %#v, want materialized pending file", scope.FileIDs)
	}
	if len(scope.RAGSources) != 1 {
		t.Fatalf("rag sources = %#v, want one source", scope.RAGSources)
	}
	got := scope.RAGSources[0]
	if len(got.FileIDs) != 1 || got.FileIDs[0] != "pending_file" {
		t.Fatalf("rag source file ids = %#v, want materialized pending file", got.FileIDs)
	}
	if len(got.SourceRowIDs) != 0 || got.SourceRowIDByFileID != nil || len(got.SourceTags) != 0 {
		t.Fatalf("pending source leaked unpublished governance into RAG source: %#v", got)
	}
	if constraint, ok := got.IndexVersionConstraintByFileID["pending_file"]; !ok || constraint.Kind != knowledge.RAGIndexVersionConstraintValue || constraint.Value != 7 {
		t.Fatalf("index version constraints = %#v, want materialized vector constraint", got.IndexVersionConstraintByFileID)
	}
	if len(legacy.fileCalls) != 1 || legacy.fileCalls[0].VectorTable != "text_vec" {
		t.Fatalf("legacy file resolver calls = %#v, want current KB vector table", legacy.fileCalls)
	}
}

func TestPlatformKnowledgeScopeResolverAppliesLegacyIndexConstraints(t *testing.T) {
	files, err := json.Marshal(map[string]any{
		"file_ids":        []string{"file_min", "file_zero", "file_null", "file_pointer"},
		"vector_table":    "text_vec",
		"embedding_model": "bge-m3",
	})
	if err != nil {
		t.Fatalf("marshal files: %v", err)
	}
	legacy := &platformKnowledgeFilterTestLegacyIndexResolver{
		constraints: map[string]knowledge.RAGIndexVersionConstraint{
			"file_min":  {Kind: knowledge.RAGIndexVersionConstraintValue, Value: 3},
			"file_zero": {Kind: knowledge.RAGIndexVersionConstraintValue, Value: 0},
			"file_null": {Kind: knowledge.RAGIndexVersionConstraintNull},
		},
	}
	resolver := &PlatformKnowledgeScopeResolver{
		semantic: platformKnowledgeFilterTestSemanticStore{
			model: &tenant.SemanticModelRecord{
				ID:     12,
				Name:   "legacy governed kb",
				Tables: json.RawMessage(`[]`),
				Files:  files,
			},
		},
		governance: platformKnowledgeFilterTestGovernanceStore{
			records: []PlatformKnowledgeSourceGovernanceRecord{
				{SourceRowID: "source_min", FileID: "file_min", Status: "succeeded", EffectiveEnabled: true},
				{SourceRowID: "source_zero", FileID: "file_zero", Status: "succeeded", EffectiveEnabled: true},
				{SourceRowID: "source_null", FileID: "file_null", Status: "succeeded", EffectiveEnabled: true},
				{SourceRowID: "source_pointer", FileID: "file_pointer", Status: "succeeded", EffectiveEnabled: true, SegmentVersionID: "seg_pointer", IndexVersion: 9, IndexVersionValid: true},
			},
		},
		legacyIndex: legacy,
	}

	scope, err := resolver.ResolveKnowledgeScope(context.Background(), knowledge.WorkspaceScope{
		WorkspaceID:      "ws_1",
		UserID:           "user_1",
		SemanticModelIDs: []int64{12},
	})
	if err != nil {
		t.Fatalf("ResolveKnowledgeScope() error = %v", err)
	}
	if len(legacy.calls) != 1 {
		t.Fatalf("legacy resolver calls = %#v, want one call", legacy.calls)
	}
	if got := strings.Join(legacy.calls[0].FileIDs, ","); got != "file_min,file_zero,file_null" {
		t.Fatalf("legacy resolver file ids = %q, want missing-pointer files only", got)
	}
	got := scope.RAGSources[0]
	if got.CurrentIndexVersionByFileID["file_pointer"] != 9 {
		t.Fatalf("pointer current index versions = %#v", got.CurrentIndexVersionByFileID)
	}
	if constraint := got.IndexVersionConstraintByFileID["file_pointer"]; constraint.Kind != knowledge.RAGIndexVersionConstraintValue || constraint.Value != 9 {
		t.Fatalf("pointer constraint = %#v, want value 9", constraint)
	}
	if constraint := got.IndexVersionConstraintByFileID["file_min"]; constraint.Kind != knowledge.RAGIndexVersionConstraintValue || constraint.Value != 3 {
		t.Fatalf("file_min constraint = %#v, want value 3", constraint)
	}
	if constraint := got.IndexVersionConstraintByFileID["file_zero"]; constraint.Kind != knowledge.RAGIndexVersionConstraintValue || constraint.Value != 0 {
		t.Fatalf("file_zero constraint = %#v, want value 0", constraint)
	}
	if constraint := got.IndexVersionConstraintByFileID["file_null"]; constraint.Kind != knowledge.RAGIndexVersionConstraintNull {
		t.Fatalf("file_null constraint = %#v, want NULL", constraint)
	}
}

// TestPlatformKnowledgeScopeResolverToleratesLegacyIndexVersionNotReadyForZeroGovernanceFile
// mirrors #12764's workflow-built KB shape: a semantic model whose files.file_ids
// was bound purely by RegisterLineage (no knowledge_base_sources row ever
// written), so the file carries no governance record and therefore no known
// index_version. Before the physical vector/chunk table exists,
// ResolveLegacyIndexVersions legitimately reports errPlatformKnowledgeVectorIndexNotReady
// (the same sentinel the sibling ResolveLegacyIndexFileIDs path already
// tolerates). That must not fail the whole knowledge scope resolution — it
// must simply mean "no legacy index-version constraint to apply" and keep the
// file discoverable, matching the semantics of hasFileRAGSource in
// ResolveKnowledgeScope.
func TestPlatformKnowledgeScopeResolverToleratesLegacyIndexVersionNotReadyForZeroGovernanceFile(t *testing.T) {
	files, err := json.Marshal(map[string]any{
		"file_ids":        []string{"wf_built_source"},
		"vector_table":    "wf_built_text_vec",
		"embedding_model": "bge-m3",
	})
	if err != nil {
		t.Fatalf("marshal files: %v", err)
	}
	legacy := &platformKnowledgeFilterTestLegacyIndexResolver{
		versionErr: errPlatformKnowledgeVectorIndexNotReady,
	}
	resolver := &PlatformKnowledgeScopeResolver{
		semantic: platformKnowledgeFilterTestSemanticStore{
			model: &tenant.SemanticModelRecord{
				ID:     12764,
				Name:   "workflow built kb",
				Tables: json.RawMessage(`[]`),
				Files:  files,
			},
		},
		governance:  platformKnowledgeFilterTestGovernanceStore{},
		legacyIndex: legacy,
	}

	scope, err := resolver.ResolveKnowledgeScope(context.Background(), knowledge.WorkspaceScope{
		WorkspaceID:      "ws_1",
		UserID:           "user_1",
		SemanticModelIDs: []int64{12764},
	})
	if err != nil {
		t.Fatalf("ResolveKnowledgeScope() error = %v, want workflow-built file to stay discoverable despite legacy index not being ready yet", err)
	}
	if len(legacy.calls) != 1 {
		t.Fatalf("legacy version resolver calls = %#v, want one call", legacy.calls)
	}
	if len(scope.RAGSources) != 1 || len(scope.RAGSources[0].FileIDs) != 1 || scope.RAGSources[0].FileIDs[0] != "wf_built_source" {
		t.Fatalf("rag sources = %#v, want wf_built_source to remain bound", scope.RAGSources)
	}
	if len(scope.FileIDs) != 1 || scope.FileIDs[0] != "wf_built_source" {
		t.Fatalf("scope file ids = %#v, want wf_built_source", scope.FileIDs)
	}
}

func TestQueryPlatformKnowledgeLegacyIndexVersionConstraintsSelectsMinAndNull(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()
	mock.ExpectQuery(regexp.QuoteMeta("SHOW COLUMNS FROM `text_vec`")).
		WillReturnRows(sqlmock.NewRows([]string{"Field", "Type"}).
			AddRow("file_id", "varchar").
			AddRow("index_version", "bigint").
			AddRow("level", "varchar").
			AddRow("disabled", "tinyint"))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT DISTINCT file_id, index_version FROM `text_vec` WHERE file_id IN (?,?) AND COALESCE(disabled, 0) = 0 AND level = 'chunk' ORDER BY file_id, index_version")).
		WithArgs("file_a", "file_b").
		WillReturnRows(sqlmock.NewRows([]string{"file_id", "index_version"}).
			AddRow("file_a", int64(5)).
			AddRow("file_a", int64(3)).
			AddRow("file_a", nil).
			AddRow("file_b", nil))

	constraints, err := queryPlatformKnowledgeLegacyIndexVersionConstraints(context.Background(), db, "text_vec", []string{"file_a", "file_b"})
	if err != nil {
		t.Fatalf("queryPlatformKnowledgeLegacyIndexVersionConstraints: %v", err)
	}
	if got := constraints["file_a"]; got.Kind != knowledge.RAGIndexVersionConstraintValue || got.Value != 3 {
		t.Fatalf("file_a constraint = %#v, want min value 3", got)
	}
	if got := constraints["file_b"]; got.Kind != knowledge.RAGIndexVersionConstraintNull {
		t.Fatalf("file_b constraint = %#v, want NULL", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestQueryPlatformKnowledgeLegacyIndexVersionConstraintsTreatsMissingVectorTableAsNotReady(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()
	mock.ExpectQuery(regexp.QuoteMeta("SHOW COLUMNS FROM `kb_2_text_index`")).
		WillReturnError(&mysql.MySQLError{Number: 1146, Message: "no such table moi.kb_2_text_index"})

	_, err = queryPlatformKnowledgeLegacyIndexVersionConstraints(context.Background(), db, "kb_2_text_index", []string{"file-1"})
	if !errors.Is(err, errPlatformKnowledgeVectorIndexNotReady) {
		t.Fatalf("error = %v, want index not ready", err)
	}
	if strings.Contains(err.Error(), "SHOW COLUMNS") || strings.Contains(err.Error(), "no such table") {
		t.Fatalf("error leaked storage detail: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestPlatformKnowledgeScopeResolverUsesKnowledgeBaseSourcesAsFileAuthority(t *testing.T) {
	files, err := json.Marshal(map[string]any{
		"file_ids":        []string{"default_docx"},
		"vector_table":    "kb_ffff_mr4zmh953e1l",
		"embedding_model": "bge-m3",
	})
	if err != nil {
		t.Fatalf("marshal files: %v", err)
	}
	tables, err := json.Marshal([]map[string]any{{
		"db_name":     "ffff_15",
		"table_names": []string{"test.csv"},
	}})
	if err != nil {
		t.Fatalf("marshal tables: %v", err)
	}
	resolver := &PlatformKnowledgeScopeResolver{
		semantic: platformKnowledgeFilterTestSemanticStore{
			model: &tenant.SemanticModelRecord{
				ID:     15,
				Name:   "ffff",
				Tables: tables,
				Files:  files,
			},
		},
		governance: platformKnowledgeFilterTestGovernanceStore{
			records: []PlatformKnowledgeSourceGovernanceRecord{
				{SourceRowID: "source_pdf", FileID: "pdf_file", Status: "succeeded", EffectiveEnabled: true, SegmentVersionID: "seg_pdf", IndexVersion: 3},
				{SourceRowID: "source_docx", FileID: "default_docx", Status: "succeeded", EffectiveEnabled: true, SegmentVersionID: "seg_docx", IndexVersion: 2},
			},
		},
	}

	scope, err := resolver.ResolveKnowledgeScope(context.Background(), knowledge.WorkspaceScope{
		WorkspaceID:      "ws_1",
		UserID:           "user_1",
		DBName:           "ffff_15",
		SemanticModelIDs: []int64{15},
	})
	if err != nil {
		t.Fatalf("ResolveKnowledgeScope() error = %v", err)
	}
	if scope.DBName != "ffff_15" || len(scope.Tables) != 1 || scope.Tables[0] != "ffff_15.test.csv" {
		t.Fatalf("structured scope db/tables = %q/%#v, want ffff_15/ffff_15.test.csv", scope.DBName, scope.Tables)
	}
	if len(scope.RAGSources) != 1 {
		t.Fatalf("rag sources = %#v, want one source", scope.RAGSources)
	}
	got := scope.RAGSources[0]
	if got.DBName != "" {
		t.Fatalf("rag db = %q, want empty to use tenant connection default database", got.DBName)
	}
	if got.VectorTable != "kb_ffff_mr4zmh953e1l" || got.EmbeddingModel != "bge-m3" {
		t.Fatalf("rag index = %q/%q, want kb_ffff_mr4zmh953e1l/bge-m3", got.VectorTable, got.EmbeddingModel)
	}
	if len(got.FileIDs) != 2 || !containsAll(strings.Join(got.FileIDs, ","), "pdf_file", "default_docx") {
		t.Fatalf("rag file ids = %#v, want source-authoritative pdf_file and default_docx", got.FileIDs)
	}
	if got.CurrentIndexVersionByFileID["pdf_file"] != 3 || got.CurrentIndexVersionByFileID["default_docx"] != 2 {
		t.Fatalf("current index versions = %#v", got.CurrentIndexVersionByFileID)
	}
}

func TestPlatformKnowledgeScopeResolverAllowsUnqualifiedRAGSourceWithoutExplicitDBName(t *testing.T) {
	files, err := json.Marshal(map[string]any{
		"file_ids":        []string{"file_1"},
		"vector_table":    "kb_docs",
		"embedding_model": "bge-m3",
	})
	if err != nil {
		t.Fatalf("marshal files: %v", err)
	}
	resolver := &PlatformKnowledgeScopeResolver{
		semantic: platformKnowledgeFilterTestSemanticStore{
			model: &tenant.SemanticModelRecord{
				ID:     16,
				Name:   "docs",
				Tables: json.RawMessage(`[]`),
				Files:  files,
			},
		},
	}

	scope, err := resolver.ResolveKnowledgeScope(context.Background(), knowledge.WorkspaceScope{
		WorkspaceID:      "ws_1",
		UserID:           "user_1",
		SemanticModelIDs: []int64{16},
	})
	if err != nil {
		t.Fatalf("ResolveKnowledgeScope() error = %v", err)
	}
	if len(scope.RAGSources) != 1 {
		t.Fatalf("rag sources = %#v, want one source", scope.RAGSources)
	}
	got := scope.RAGSources[0]
	if got.DBName != "" {
		t.Fatalf("rag db = %q, want empty to use tenant connection default database", got.DBName)
	}
	if got.VectorTable != "kb_docs" || got.EmbeddingModel != "bge-m3" {
		t.Fatalf("rag index = %q/%q, want kb_docs/bge-m3", got.VectorTable, got.EmbeddingModel)
	}
}

func TestPlatformKnowledgeScopeResolverAppliesTableGovernance(t *testing.T) {
	tables, err := json.Marshal([]map[string]any{
		{
			"db_name":     "sales",
			"table_names": []string{"orders", "customers"},
		},
		{
			"db_name":     "support",
			"table_names": []string{"orders"},
		},
	})
	if err != nil {
		t.Fatalf("marshal tables: %v", err)
	}
	resolver := &PlatformKnowledgeScopeResolver{
		semantic: platformKnowledgeFilterTestSemanticStore{
			model: &tenant.SemanticModelRecord{
				ID:     15,
				Name:   "governed tables",
				Tables: tables,
				Files:  json.RawMessage(`{}`),
			},
		},
		tableGovernance: platformKnowledgeFilterTestGovernanceStore{
			tableRecords: []PlatformKnowledgeTableSourceGovernanceRecord{
				{SourceRowID: "source_orders", DBName: "sales", TableName: "orders", KBTableID: "kb_orders", Status: "succeeded", Enabled: false, EffectiveEnabled: false},
				{SourceRowID: "source_customers", DBName: "sales", TableName: "customers", KBTableID: "kb_customers", Status: "succeeded", Enabled: true, EffectiveEnabled: true},
			},
		},
	}

	scope, err := resolver.ResolveKnowledgeScope(context.Background(), knowledge.WorkspaceScope{
		WorkspaceID:      "ws_1",
		UserID:           "user_1",
		DBName:           "sales",
		SemanticModelIDs: []int64{15},
	})
	if err != nil {
		t.Fatalf("ResolveKnowledgeScope() error = %v", err)
	}
	// Metadata DBName must not drop other-db tables. Governance only disables
	// sales.orders; sales.customers stays enabled and support.orders is legacy
	// (no governance row) so it remains. Multi-db result clears Scope.DBName.
	if scope.DBName != "" {
		t.Fatalf("DBName = %q, want empty multi-database scope", scope.DBName)
	}
	if !reflect.DeepEqual(scope.Tables, []string{"sales.customers", "support.orders"}) {
		t.Fatalf("scope tables = %#v, want [sales.customers support.orders]", scope.Tables)
	}

	got := newPlatformKnowledgeGovernanceByTable([]PlatformKnowledgeTableSourceGovernanceRecord{
		{DBName: "sales", TableName: "orders", Status: "succeeded", Enabled: false},
	}).filterRAGReadyOrLegacyTables([]PlatformKnowledgeSemanticModelTableRef{{DBName: "support", TableName: "orders"}})
	if len(got) != 1 || got[0].DBName != "support" || got[0].TableName != "orders" {
		t.Fatalf("cross-db table refs = %#v, want support.orders retained", got)
	}
}

func TestPlatformKnowledgeScopeResolverAllowsTableOnlyScopeWhenDefaultVectorTableIsMissing(t *testing.T) {
	files, err := json.Marshal(map[string]any{
		"file_ids":        []string{},
		"vector_table":    "kb_50021_t_missing_vector",
		"embedding_model": "bge-m3",
	})
	if err != nil {
		t.Fatalf("marshal files: %v", err)
	}
	tables, err := json.Marshal([]map[string]any{{
		"db_name":     "kb_50021_t_60008",
		"table_names": []string{"dimproductsubcategory"},
	}})
	if err != nil {
		t.Fatalf("marshal tables: %v", err)
	}
	legacy := &platformKnowledgeFilterTestLegacyIndexResolver{
		fileErr: errPlatformKnowledgeVectorIndexNotReady,
	}
	resolver := &PlatformKnowledgeScopeResolver{
		semantic: platformKnowledgeFilterTestSemanticStore{
			model: &tenant.SemanticModelRecord{
				ID:     50021,
				Name:   "table only kb",
				Tables: tables,
				Files:  files,
			},
		},
		governance:  platformKnowledgeFilterTestGovernanceStore{},
		legacyIndex: legacy,
	}

	scope, err := resolver.ResolveKnowledgeScope(context.Background(), knowledge.WorkspaceScope{
		WorkspaceID:      "ws_1",
		UserID:           "user_1",
		DBName:           "kb_50021_t_60008",
		SemanticModelIDs: []int64{50021},
	})
	if err != nil {
		t.Fatalf("ResolveKnowledgeScope() error = %v", err)
	}
	if len(legacy.fileCalls) != 1 {
		t.Fatalf("legacy file resolver calls = %#v, want one legacy document lookup", legacy.fileCalls)
	}
	if len(scope.Tables) != 1 || scope.Tables[0] != "kb_50021_t_60008.dimproductsubcategory" {
		t.Fatalf("scope tables = %#v, want kb_50021_t_60008.dimproductsubcategory", scope.Tables)
	}
	if len(scope.RAGSources) != 0 || len(scope.FileIDs) != 0 {
		t.Fatalf("rag scope = %#v file_ids=%#v, want table-only scope", scope.RAGSources, scope.FileIDs)
	}
	if scope.VectorTable != "" || scope.EmbeddingModel != "" {
		t.Fatalf("top-level vector config = %q/%q, want cleared for table-only scope", scope.VectorTable, scope.EmbeddingModel)
	}
}

func TestPlatformKnowledgeScopeResolverKeepsLegacyDocumentsWhenTablesAlsoExist(t *testing.T) {
	files, err := json.Marshal(map[string]any{
		"file_ids":        []string{},
		"vector_table":    "kb_mixed_legacy_vector",
		"embedding_model": "bge-m3",
	})
	if err != nil {
		t.Fatalf("marshal files: %v", err)
	}
	tables, err := json.Marshal([]map[string]any{{
		"db_name":     "kb_mixed",
		"table_names": []string{"orders"},
	}})
	if err != nil {
		t.Fatalf("marshal tables: %v", err)
	}
	legacy := &platformKnowledgeFilterTestLegacyIndexResolver{
		fileIDs: []string{"legacy_doc"},
		constraints: map[string]knowledge.RAGIndexVersionConstraint{
			"legacy_doc": {Kind: knowledge.RAGIndexVersionConstraintValue, Value: 9},
		},
	}
	resolver := &PlatformKnowledgeScopeResolver{
		semantic: platformKnowledgeFilterTestSemanticStore{
			model: &tenant.SemanticModelRecord{
				ID:     50022,
				Name:   "mixed legacy kb",
				Tables: tables,
				Files:  files,
			},
		},
		governance:  platformKnowledgeFilterTestGovernanceStore{},
		legacyIndex: legacy,
	}

	scope, err := resolver.ResolveKnowledgeScope(context.Background(), knowledge.WorkspaceScope{
		WorkspaceID:      "ws_1",
		UserID:           "user_1",
		DBName:           "kb_mixed",
		SemanticModelIDs: []int64{50022},
	})
	if err != nil {
		t.Fatalf("ResolveKnowledgeScope() error = %v", err)
	}
	if len(scope.Tables) != 1 || scope.Tables[0] != "kb_mixed.orders" {
		t.Fatalf("scope tables = %#v, want kb_mixed.orders", scope.Tables)
	}
	if len(scope.RAGSources) != 1 || len(scope.RAGSources[0].FileIDs) != 1 || scope.RAGSources[0].FileIDs[0] != "legacy_doc" {
		t.Fatalf("rag sources = %#v, want legacy document source", scope.RAGSources)
	}
	if len(legacy.fileCalls) != 1 || len(legacy.calls) != 1 {
		t.Fatalf("legacy calls = files:%#v versions:%#v, want one lookup and one version constraint call", legacy.fileCalls, legacy.calls)
	}
}

func TestPlatformKnowledgeScopeResolverDoesNotUseGlobalImageIndexWithoutSemanticImageBinding(t *testing.T) {
	files, err := json.Marshal(map[string]any{
		"file_ids":        []string{},
		"vector_table":    "kb_text_only_legacy_vector",
		"embedding_model": "bge-m3",
	})
	if err != nil {
		t.Fatalf("marshal files: %v", err)
	}
	legacy := &platformKnowledgeFilterTestLegacyIndexResolver{
		fileIDs: []string{"legacy_doc"},
	}
	image := &platformKnowledgeFilterTestImageIndexResolver{
		result: &PlatformKnowledgeImageIndexResolveResult{
			ImageVectorTable:        "unrelated_img",
			ImageEmbeddingModel:     "efficientnet-b3",
			ImageEmbeddingDimension: 1536,
			ImagePreprocessVersion:  "efficientnet-b3-v1",
			ImageDistanceMetric:     "cosine",
		},
	}
	resolver := &PlatformKnowledgeScopeResolver{
		semantic: platformKnowledgeFilterTestSemanticStore{
			model: &tenant.SemanticModelRecord{
				ID:     50025,
				Name:   "text only legacy kb",
				Tables: json.RawMessage(`[]`),
				Files:  files,
			},
		},
		governance:  platformKnowledgeFilterTestGovernanceStore{},
		legacyIndex: legacy,
		image:       image,
	}

	scope, err := resolver.ResolveKnowledgeScope(context.Background(), knowledge.WorkspaceScope{
		WorkspaceID:      "ws_1",
		UserID:           "user_1",
		SemanticModelIDs: []int64{50025},
	})
	if err != nil {
		t.Fatalf("ResolveKnowledgeScope() error = %v", err)
	}
	if len(legacy.fileCalls) != 1 {
		t.Fatalf("legacy file resolver calls = %#v, want one legacy document lookup", legacy.fileCalls)
	}
	if image.calls != 0 {
		t.Fatalf("image resolver calls = %d, want 0 without semantic image vector binding", image.calls)
	}
	if len(scope.RAGSources) != 1 || len(scope.RAGSources[0].FileIDs) != 1 || scope.RAGSources[0].FileIDs[0] != "legacy_doc" {
		t.Fatalf("rag sources = %#v, want text legacy document source", scope.RAGSources)
	}
	if scope.ImageVectorTable != "" || scope.ImageEmbeddingModel != "" || scope.ImageEmbeddingDimension != 0 {
		t.Fatalf("image scope = %q/%q/%d, want no visual scope", scope.ImageVectorTable, scope.ImageEmbeddingModel, scope.ImageEmbeddingDimension)
	}
}

func TestPlatformKnowledgeScopeResolverRequiresOwnImageVectorTableBeforeImageLineage(t *testing.T) {
	files, err := json.Marshal(map[string]any{
		"file_ids":              []string{"file_1"},
		"vector_table":          "text_vec",
		"embedding_model":       "bge-m3",
		"image_embedding_model": "efficientnet-b3",
	})
	if err != nil {
		t.Fatalf("marshal files: %v", err)
	}
	image := &platformKnowledgeFilterTestImageIndexResolver{
		result: &PlatformKnowledgeImageIndexResolveResult{
			ImageVectorTable:        "other_kb_img",
			ImageEmbeddingModel:     "efficientnet-b3",
			ImageEmbeddingDimension: 1536,
			ImagePreprocessVersion:  "efficientnet-b3-v1",
			ImageDistanceMetric:     "cosine",
		},
	}
	resolver := &PlatformKnowledgeScopeResolver{
		semantic: platformKnowledgeFilterTestSemanticStore{
			model: &tenant.SemanticModelRecord{
				ID:     50026,
				Name:   "partial image kb",
				Tables: json.RawMessage(`[]`),
				Files:  files,
			},
		},
		image: image,
	}

	scope, err := resolver.ResolveKnowledgeScope(context.Background(), knowledge.WorkspaceScope{
		WorkspaceID:      "ws_1",
		UserID:           "user_1",
		SemanticModelIDs: []int64{50026},
	})
	if err != nil {
		t.Fatalf("ResolveKnowledgeScope() error = %v", err)
	}
	if image.calls != 0 {
		t.Fatalf("image resolver calls = %d, want 0 without semantic image_vector_table", image.calls)
	}
	if len(scope.RAGSources) != 1 {
		t.Fatalf("rag sources = %#v, want one text source", scope.RAGSources)
	}
	got := scope.RAGSources[0]
	if got.ImageVectorTable != "" || got.ImageEmbeddingModel != "" || got.ImageEmbeddingDimension != 0 {
		t.Fatalf("rag source image scope = %#v, want no visual scope", got)
	}
	if scope.ImageVectorTable != "" || scope.ImageEmbeddingModel != "" || scope.ImageEmbeddingDimension != 0 {
		t.Fatalf("image scope = %q/%q/%d, want no visual scope", scope.ImageVectorTable, scope.ImageEmbeddingModel, scope.ImageEmbeddingDimension)
	}
}

func TestPlatformKnowledgeScopeResolverSkipsMissingLegacyIndexWhenTablesAreNotReady(t *testing.T) {
	files, err := json.Marshal(map[string]any{
		"file_ids":        []string{},
		"vector_table":    "kb_pending_table_vector",
		"embedding_model": "bge-m3",
	})
	if err != nil {
		t.Fatalf("marshal files: %v", err)
	}
	tables, err := json.Marshal([]map[string]any{{
		"db_name":     "kb_pending",
		"table_names": []string{"orders"},
	}})
	if err != nil {
		t.Fatalf("marshal tables: %v", err)
	}
	legacy := &platformKnowledgeFilterTestLegacyIndexResolver{
		fileErr: errPlatformKnowledgeVectorIndexNotReady,
	}
	resolver := &PlatformKnowledgeScopeResolver{
		semantic: platformKnowledgeFilterTestSemanticStore{
			model: &tenant.SemanticModelRecord{
				ID:     50023,
				Name:   "pending table kb",
				Tables: tables,
				Files:  files,
			},
		},
		governance: platformKnowledgeFilterTestGovernanceStore{},
		tableGovernance: platformKnowledgeFilterTestGovernanceStore{
			tableRecords: []PlatformKnowledgeTableSourceGovernanceRecord{
				{SourceRowID: "source_orders", DBName: "kb_pending", TableName: "orders", Status: "pending", EffectiveEnabled: true},
			},
		},
		legacyIndex: legacy,
	}

	scope, err := resolver.ResolveKnowledgeScope(context.Background(), knowledge.WorkspaceScope{
		WorkspaceID:      "ws_1",
		UserID:           "user_1",
		DBName:           "kb_pending",
		SemanticModelIDs: []int64{50023},
	})
	if err != nil {
		t.Fatalf("ResolveKnowledgeScope() error = %v", err)
	}
	if len(legacy.fileCalls) != 1 {
		t.Fatalf("legacy file resolver calls = %#v, want one missing-index lookup", legacy.fileCalls)
	}
	if len(scope.Tables) != 0 || len(scope.RAGSources) != 0 {
		t.Fatalf("scope = %#v, want no unavailable table or stale document source", scope)
	}
}

func TestPlatformKnowledgeScopeResolverKeepsReadyTableWhenGovernedDocumentIsPending(t *testing.T) {
	files, err := json.Marshal(map[string]any{
		"file_ids":        []string{"pending_doc"},
		"vector_table":    "kb_table_pending_doc_vector",
		"embedding_model": "bge-m3",
	})
	if err != nil {
		t.Fatalf("marshal files: %v", err)
	}
	tables, err := json.Marshal([]map[string]any{{
		"db_name":     "kb_ready_table",
		"table_names": []string{"orders"},
	}})
	if err != nil {
		t.Fatalf("marshal tables: %v", err)
	}
	resolver := &PlatformKnowledgeScopeResolver{
		semantic: platformKnowledgeFilterTestSemanticStore{
			model: &tenant.SemanticModelRecord{
				ID:     50024,
				Name:   "ready table pending doc kb",
				Tables: tables,
				Files:  files,
			},
		},
		governance: platformKnowledgeFilterTestGovernanceStore{
			records: []PlatformKnowledgeSourceGovernanceRecord{
				{SourceRowID: "source_pending_doc", FileID: "pending_doc", Status: "pending", EffectiveEnabled: true},
			},
		},
		tableGovernance: platformKnowledgeFilterTestGovernanceStore{
			tableRecords: []PlatformKnowledgeTableSourceGovernanceRecord{
				{SourceRowID: "source_orders", DBName: "kb_ready_table", TableName: "orders", Status: "succeeded", EffectiveEnabled: true},
			},
		},
	}

	scope, err := resolver.ResolveKnowledgeScope(context.Background(), knowledge.WorkspaceScope{
		WorkspaceID:      "ws_1",
		UserID:           "user_1",
		DBName:           "kb_ready_table",
		SemanticModelIDs: []int64{50024},
	})
	if err != nil {
		t.Fatalf("ResolveKnowledgeScope() error = %v", err)
	}
	if len(scope.Tables) != 1 || scope.Tables[0] != "kb_ready_table.orders" {
		t.Fatalf("scope tables = %#v, want ready table kb_ready_table.orders", scope.Tables)
	}
	if len(scope.FileIDs) != 0 || len(scope.RAGSources) != 0 {
		t.Fatalf("rag scope = %#v file_ids=%#v, want pending document skipped", scope.RAGSources, scope.FileIDs)
	}
}

func TestPlatformKnowledgeScopeResolverFiltersPendingTableGovernance(t *testing.T) {
	tables, err := json.Marshal([]map[string]any{{
		"db_name":     "sales",
		"table_names": []string{"orders", "customers", "legacy_table"},
	}})
	if err != nil {
		t.Fatalf("marshal tables: %v", err)
	}
	resolver := &PlatformKnowledgeScopeResolver{
		semantic: platformKnowledgeFilterTestSemanticStore{
			model: &tenant.SemanticModelRecord{
				ID:     15,
				Name:   "governed tables",
				Tables: tables,
				Files:  json.RawMessage(`{}`),
			},
		},
		tableGovernance: platformKnowledgeFilterTestGovernanceStore{
			tableRecords: []PlatformKnowledgeTableSourceGovernanceRecord{
				{SourceRowID: "source_orders", DBName: "sales", TableName: "orders", Status: "pending", Enabled: true, EffectiveEnabled: true},
				{SourceRowID: "source_customers", DBName: "sales", TableName: "customers", Status: "succeeded", Enabled: true, EffectiveEnabled: true},
			},
		},
	}

	scope, err := resolver.ResolveKnowledgeScope(context.Background(), knowledge.WorkspaceScope{
		WorkspaceID:      "ws_1",
		UserID:           "user_1",
		DBName:           "sales",
		SemanticModelIDs: []int64{15},
	})
	if err != nil {
		t.Fatalf("ResolveKnowledgeScope() error = %v", err)
	}
	if len(scope.Tables) != 2 || !containsAll(strings.Join(scope.Tables, ","), "customers", "legacy_table") || containsAny(strings.Join(scope.Tables, ","), "orders") {
		t.Fatalf("scope tables = %#v, want succeeded governed table plus explicit legacy table", scope.Tables)
	}
}

func TestPlatformKnowledgeSourceTableRefsDoNotDefaultPendingGovernance(t *testing.T) {
	store := platformKnowledgeFilterTestGovernanceStore{
		tableRecords: []PlatformKnowledgeTableSourceGovernanceRecord{
			{
				SourceRowID:      "source_orders",
				DBName:           "kb_sales",
				TableName:        "orders",
				SourceDBName:     "raw_sales",
				SourceTableName:  "orders",
				Status:           "pending",
				EffectiveEnabled: true,
			},
		},
	}
	hook := platformKnowledgeSourceTableRefs(store)
	if hook == nil {
		t.Fatalf("expected table refs hook")
	}
	ctx := knowledge.ContextWithScope(context.Background(), knowledge.WorkspaceScope{
		WorkspaceID:      "ws_1",
		SemanticModelIDs: []int64{16},
	})
	refs, err := hook(ctx, []string{"orders"}, "kb_sales")
	if err != nil {
		t.Fatalf("table refs hook returned err: %v", err)
	}
	if len(refs) != 0 {
		t.Fatalf("refs = %#v, want pending governed table hidden without default fallback", refs)
	}
}

func TestPlatformKnowledgeToolFilterHidesRAGWhenGovernanceRemovesAllFiles(t *testing.T) {
	files, err := json.Marshal(map[string]any{
		"file_ids":        []string{"file_disabled"},
		"vector_table":    "text_vec",
		"embedding_model": "bge-m3",
	})
	if err != nil {
		t.Fatalf("marshal files: %v", err)
	}
	filter := NewPlatformKnowledgeToolFilter(&PlatformKnowledgeScopeResolver{
		semantic: platformKnowledgeFilterTestSemanticStore{
			model: &tenant.SemanticModelRecord{
				ID:     13,
				Name:   "disabled kb",
				Tables: json.RawMessage(`[]`),
				Files:  files,
			},
		},
		governance: platformKnowledgeFilterTestGovernanceStore{
			records: []PlatformKnowledgeSourceGovernanceRecord{
				{SourceRowID: "source_disabled", FileID: "file_disabled", Status: "succeeded", EffectiveEnabled: false},
			},
		},
	})

	tools, err := filter.FilterTools(context.Background(), agentruntime.RuntimeToolFilterRequest{
		Scope:    agentruntime.RuntimeRequestScope{WorkspaceID: "ws_1", AgentID: "explore", UserID: "user_1"},
		Instance: platformKnowledgeFilterTestDescriptor(t),
		Metadata: map[string]any{
			"semantic_model_ids": []any{float64(13)},
		},
	})
	if err != nil {
		t.Fatalf("FilterTools() error = %v", err)
	}
	for _, hidden := range []string{agenttools.ToolKindSearchRAGChunks, agenttools.ToolKindFindRAGFiles, agenttools.ToolKindReadParsedMarkdown, agenttools.ToolKindSearchParsedMarkdown} {
		if platformKnowledgeFilterTestContains(tools, hidden) {
			t.Fatalf("governance-filtered scope exposed %s: %+v", hidden, tools)
		}
	}
}

func TestPlatformKnowledgeScopeResolverDropsMetadataVectorScopeWhenGovernanceRemovesAllFiles(t *testing.T) {
	files, err := json.Marshal(map[string]any{
		"file_ids":        []string{"file_disabled"},
		"vector_table":    "text_vec",
		"embedding_model": "bge-m3",
	})
	if err != nil {
		t.Fatalf("marshal files: %v", err)
	}
	resolver := &PlatformKnowledgeScopeResolver{
		semantic: platformKnowledgeFilterTestSemanticStore{
			model: &tenant.SemanticModelRecord{
				ID:     14,
				Name:   "disabled kb",
				Tables: json.RawMessage(`[]`),
				Files:  files,
			},
		},
		governance: platformKnowledgeFilterTestGovernanceStore{
			records: []PlatformKnowledgeSourceGovernanceRecord{
				{SourceRowID: "source_disabled", FileID: "file_disabled", Status: "succeeded", EffectiveEnabled: false},
			},
		},
	}

	scope, err := resolver.ResolveKnowledgeScope(context.Background(), knowledge.WorkspaceScope{
		WorkspaceID:      "ws_1",
		UserID:           "user_1",
		SemanticModelIDs: []int64{14},
		FileIDs:          []string{"file_disabled"},
		VectorTable:      "text_vec",
		EmbeddingModel:   "bge-m3",
		RAGSources: []knowledge.RAGSource{{
			SemanticModelID: 14,
			VectorTable:     "text_vec",
			EmbeddingModel:  "bge-m3",
			Metadata:        map[string]string{"source": "manifest_knowledge"},
		}},
	})
	if err != nil {
		t.Fatalf("ResolveKnowledgeScope() error = %v", err)
	}
	if len(scope.FileIDs) != 0 {
		t.Fatalf("scope file ids = %#v, want empty after governance", scope.FileIDs)
	}
	if len(scope.RAGSources) != 0 {
		t.Fatalf("rag sources = %#v, want metadata vector source removed", scope.RAGSources)
	}
	if scope.VectorTable != "" || scope.EmbeddingModel != "" {
		t.Fatalf("top-level vector config = %q/%q, want cleared", scope.VectorTable, scope.EmbeddingModel)
	}
}

func TestPlatformKnowledgeScopeResolverSkipsMissingLegacyIndexForEmptyKnowledgeBase(t *testing.T) {
	files, err := json.Marshal(map[string]any{
		"file_ids":        []string{},
		"vector_table":    "text_vec",
		"embedding_model": "bge-m3",
	})
	if err != nil {
		t.Fatalf("marshal files: %v", err)
	}
	legacy := &platformKnowledgeFilterTestLegacyIndexResolver{
		fileErr: errPlatformKnowledgeVectorIndexNotReady,
	}
	resolver := &PlatformKnowledgeScopeResolver{
		semantic: platformKnowledgeFilterTestSemanticStore{
			model: &tenant.SemanticModelRecord{
				ID:     15,
				Name:   "unlinked kb",
				Tables: json.RawMessage(`[]`),
				Files:  files,
			},
		},
		governance:  platformKnowledgeFilterTestGovernanceStore{},
		legacyIndex: legacy,
	}

	scope, err := resolver.ResolveKnowledgeScope(context.Background(), knowledge.WorkspaceScope{
		WorkspaceID:      "ws_1",
		UserID:           "user_1",
		SemanticModelIDs: []int64{15},
		VectorTable:      "text_vec",
		EmbeddingModel:   "bge-m3",
	})
	if err != nil {
		t.Fatalf("ResolveKnowledgeScope() error = %v", err)
	}
	if len(legacy.fileCalls) != 1 {
		t.Fatalf("legacy file resolver calls = %#v, want one missing-index lookup", legacy.fileCalls)
	}
	if len(scope.RAGSources) != 0 {
		t.Fatalf("rag sources = %#v, want empty after source relation removal", scope.RAGSources)
	}
	if scope.VectorTable != "" || scope.EmbeddingModel != "" {
		t.Fatalf("top-level vector config = %q/%q, want cleared", scope.VectorTable, scope.EmbeddingModel)
	}
}

func TestPlatformKnowledgeScopeResolverKeepsVectorOnlyLegacyFilesWhenGovernanceHasNoRows(t *testing.T) {
	files, err := json.Marshal(map[string]any{
		"file_ids":        []string{},
		"vector_table":    "text_vec",
		"embedding_model": "bge-m3",
	})
	if err != nil {
		t.Fatalf("marshal files: %v", err)
	}
	legacy := &platformKnowledgeFilterTestLegacyIndexResolver{
		fileIDs: []string{"legacy_vector_file"},
		constraints: map[string]knowledge.RAGIndexVersionConstraint{
			"legacy_vector_file": {Kind: knowledge.RAGIndexVersionConstraintValue, Value: 7},
		},
	}
	resolver := &PlatformKnowledgeScopeResolver{
		semantic: platformKnowledgeFilterTestSemanticStore{
			model: &tenant.SemanticModelRecord{
				ID:     16,
				Name:   "vector only legacy kb",
				Tables: json.RawMessage(`[]`),
				Files:  files,
			},
		},
		governance:  platformKnowledgeFilterTestGovernanceStore{},
		legacyIndex: legacy,
	}

	scope, err := resolver.ResolveKnowledgeScope(context.Background(), knowledge.WorkspaceScope{
		WorkspaceID:      "ws_1",
		UserID:           "user_1",
		SemanticModelIDs: []int64{16},
	})
	if err != nil {
		t.Fatalf("ResolveKnowledgeScope() error = %v", err)
	}
	if len(legacy.fileCalls) != 1 {
		t.Fatalf("legacy file resolver calls = %#v, want one call", legacy.fileCalls)
	}
	if legacy.fileCalls[0].VectorTable != "text_vec" {
		t.Fatalf("legacy file resolver vector table = %q, want text_vec", legacy.fileCalls[0].VectorTable)
	}
	if len(scope.FileIDs) != 1 || scope.FileIDs[0] != "legacy_vector_file" {
		t.Fatalf("scope file ids = %#v, want legacy vector file", scope.FileIDs)
	}
	if len(scope.RAGSources) != 1 {
		t.Fatalf("rag sources = %#v, want one legacy source", scope.RAGSources)
	}
	got := scope.RAGSources[0]
	if len(got.FileIDs) != 1 || got.FileIDs[0] != "legacy_vector_file" {
		t.Fatalf("rag source file ids = %#v, want legacy vector file", got.FileIDs)
	}
	if got.Metadata["governance_mode"] != platformKnowledgeGovernanceModeLegacy {
		t.Fatalf("metadata = %#v, want legacy governance mode", got.Metadata)
	}
	if got.SourceRowIDByFileID != nil || len(got.SourceRowIDs) != 0 {
		t.Fatalf("source row metadata = %#v/%#v, want none for vector-only legacy file", got.SourceRowIDByFileID, got.SourceRowIDs)
	}
	if constraint, ok := got.IndexVersionConstraintByFileID["legacy_vector_file"]; !ok || constraint.Kind != knowledge.RAGIndexVersionConstraintValue || constraint.Value != 7 {
		t.Fatalf("index version constraints = %#v, want legacy vector file version 7", got.IndexVersionConstraintByFileID)
	}
}

func TestPlatformKnowledgeScopeResolverUsesMaterializedPendingGovernedRowWithoutExplicitFileBinding(t *testing.T) {
	files, err := json.Marshal(map[string]any{
		"file_ids":        []string{},
		"vector_table":    "text_vec",
		"embedding_model": "bge-m3",
	})
	if err != nil {
		t.Fatalf("marshal files: %v", err)
	}
	legacy := &platformKnowledgeFilterTestLegacyIndexResolver{
		fileIDs: []string{"pending_file"},
	}
	resolver := &PlatformKnowledgeScopeResolver{
		semantic: platformKnowledgeFilterTestSemanticStore{
			model: &tenant.SemanticModelRecord{
				ID:     17,
				Name:   "pending governed kb",
				Tables: json.RawMessage(`[]`),
				Files:  files,
			},
		},
		governance: platformKnowledgeFilterTestGovernanceStore{
			records: []PlatformKnowledgeSourceGovernanceRecord{
				{SourceRowID: "source_pending", FileID: "pending_file", Status: "pending", EffectiveEnabled: true},
			},
		},
		legacyIndex: legacy,
	}

	scope, err := resolver.ResolveKnowledgeScope(context.Background(), knowledge.WorkspaceScope{
		WorkspaceID:      "ws_1",
		UserID:           "user_1",
		SemanticModelIDs: []int64{17},
	})
	if err != nil {
		t.Fatalf("ResolveKnowledgeScope() error = %v", err)
	}
	if len(legacy.fileCalls) != 1 {
		t.Fatalf("legacy file resolver calls = %#v, want current KB vector lookup", legacy.fileCalls)
	}
	if len(scope.FileIDs) != 1 || scope.FileIDs[0] != "pending_file" {
		t.Fatalf("scope file ids = %#v, want materialized pending source", scope.FileIDs)
	}
	if len(scope.RAGSources) != 1 || len(scope.RAGSources[0].FileIDs) != 1 || scope.RAGSources[0].FileIDs[0] != "pending_file" {
		t.Fatalf("rag sources = %#v, want materialized pending source", scope.RAGSources)
	}
	if len(scope.RAGSources[0].SourceRowIDs) != 0 {
		t.Fatalf("pending source should not apply governance metadata: %#v", scope.RAGSources[0])
	}
}

func TestPlatformKnowledgeScopeResolverDoesNotUseFailedOrDisabledPendingSourceBeforeGovernancePublish(t *testing.T) {
	files, err := json.Marshal(map[string]any{
		"vector_table":    "text_vec",
		"embedding_model": "bge-m3",
	})
	if err != nil {
		t.Fatalf("marshal files: %v", err)
	}
	legacy := &platformKnowledgeFilterTestLegacyIndexResolver{fileIDs: []string{"failed_file", "disabled_pending_file"}}
	resolver := &PlatformKnowledgeScopeResolver{
		semantic: platformKnowledgeFilterTestSemanticStore{
			model: &tenant.SemanticModelRecord{ID: 18, Name: "not queryable pending kb", Tables: json.RawMessage(`[]`), Files: files},
		},
		governance: platformKnowledgeFilterTestGovernanceStore{
			records: []PlatformKnowledgeSourceGovernanceRecord{
				{SourceRowID: "source_failed", FileID: "failed_file", Status: "failed", EffectiveEnabled: true},
				{SourceRowID: "source_disabled", FileID: "disabled_pending_file", Status: "pending", EffectiveEnabled: false},
			},
		},
		legacyIndex: legacy,
	}

	scope, err := resolver.ResolveKnowledgeScope(context.Background(), knowledge.WorkspaceScope{
		WorkspaceID: "ws_1", UserID: "user_1", SemanticModelIDs: []int64{18},
	})
	if err != nil {
		t.Fatalf("ResolveKnowledgeScope() error = %v", err)
	}
	if len(legacy.fileCalls) != 0 || len(scope.FileIDs) != 0 || len(scope.RAGSources) != 0 {
		t.Fatalf("scope = %#v, legacy calls = %#v, want failed/disabled sources hidden", scope, legacy.fileCalls)
	}
}

func TestCatalogParsedMarkdownBackendRejectsEmptyGovernedScope(t *testing.T) {
	backend := &catalogParsedMarkdownBackend{}
	_, err := backend.ensureMarkdownAllowed(context.Background(), nil, knowledge.WorkspaceScope{
		WorkspaceID:      "ws_1",
		SemanticModelIDs: []int64{14},
	}, "markdown_1")
	if err == nil || !strings.Contains(err.Error(), "governed knowledge scope has no enabled files") {
		t.Fatalf("ensureMarkdownAllowed() error = %v, want empty governed scope error", err)
	}
}

func TestCatalogParsedMarkdownBackendTreatsMissingVectorTableAsNotReady(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()
	tm := transaction.NewManager(db)
	backend := &catalogParsedMarkdownBackend{}

	mock.ExpectQuery(regexp.QuoteMeta("SHOW COLUMNS FROM `kb_2_text_index`")).
		WillReturnError(&mysql.MySQLError{Number: 1146, Message: "no such table moi.kb_2_text_index"})

	_, err = backend.ensureMarkdownAllowed(context.Background(), tm, knowledge.WorkspaceScope{
		WorkspaceID:      "ws_1",
		SemanticModelIDs: []int64{14},
		FileIDs:          []string{"file-1"},
		VectorTable:      "kb_2_text_index",
	}, "md-1")
	if !errors.Is(err, errPlatformKnowledgeVectorIndexNotReady) {
		t.Fatalf("ensureMarkdownAllowed() error = %v, want index not ready", err)
	}
	if strings.Contains(err.Error(), "SHOW COLUMNS") || strings.Contains(err.Error(), "no such table") {
		t.Fatalf("error leaked storage detail: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestCatalogParsedMarkdownBackendAllowsQualifiedVectorTable(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()
	tm := transaction.NewManager(db)
	backend := &catalogParsedMarkdownBackend{}

	mock.ExpectQuery(regexp.QuoteMeta("SHOW COLUMNS FROM `idx_db`.`kb_text_idx`")).
		WillReturnRows(sqlmock.NewRows([]string{"Field", "Type"}).
			AddRow("file_id", "varchar").
			AddRow("markdown_file_id", "varchar").
			AddRow("volume_id", "varchar").
			AddRow("meta", "json"))
	mock.ExpectQuery(regexp.QuoteMeta("SHOW COLUMNS FROM `idx_db`.`kb_text_idx`")).
		WillReturnRows(sqlmock.NewRows([]string{"Field", "Type"}).
			AddRow("file_id", "varchar").
			AddRow("markdown_file_id", "varchar").
			AddRow("volume_id", "varchar").
			AddRow("meta", "json"))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT DISTINCT file_id, NULLIF(CAST(volume_id AS CHAR), '') AS volume_id FROM `idx_db`.`kb_text_idx` WHERE COALESCE(NULLIF(CAST(markdown_file_id AS CHAR), ''), NULLIF(JSON_UNQUOTE(JSON_EXTRACT(meta, '$.markdown_file_id')), ''), NULLIF(JSON_UNQUOTE(JSON_EXTRACT(meta, '$.md_file_id')), '')) = ?")).
		WithArgs("md-1").
		WillReturnRows(sqlmock.NewRows([]string{"file_id", "volume_id"}).AddRow("file-1", "vol-1"))

	sourceFileIDs, err := backend.ensureMarkdownAllowed(context.Background(), tm, knowledge.WorkspaceScope{
		WorkspaceID:      "ws_1",
		SemanticModelIDs: []int64{14},
		FileIDs:          []string{"file-1"},
		VectorTable:      "idx_db.kb_text_idx",
	}, "md-1")
	if err != nil {
		t.Fatalf("ensureMarkdownAllowed() error = %v", err)
	}
	if len(sourceFileIDs) != 1 || sourceFileIDs[0] != "file-1" {
		t.Fatalf("source file ids = %#v, want file-1", sourceFileIDs)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestCatalogParsedMarkdownBackendChecksEachRAGSourceVectorTable(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()
	tm := transaction.NewManager(db)
	backend := &catalogParsedMarkdownBackend{}

	for _, table := range []string{"`idx_a`.`kb_text_idx_a`", "`idx_b`.`kb_text_idx_b`"} {
		mock.ExpectQuery(regexp.QuoteMeta("SHOW COLUMNS FROM " + table)).
			WillReturnRows(sqlmock.NewRows([]string{"Field", "Type"}).
				AddRow("file_id", "varchar").
				AddRow("markdown_file_id", "varchar").
				AddRow("volume_id", "varchar").
				AddRow("meta", "json"))
		mock.ExpectQuery(regexp.QuoteMeta("SHOW COLUMNS FROM " + table)).
			WillReturnRows(sqlmock.NewRows([]string{"Field", "Type"}).
				AddRow("file_id", "varchar").
				AddRow("markdown_file_id", "varchar").
				AddRow("volume_id", "varchar").
				AddRow("meta", "json"))
		rows := sqlmock.NewRows([]string{"file_id", "volume_id"})
		if strings.Contains(table, "idx_b") {
			rows.AddRow("file-b", "vol-b")
		}
		mock.ExpectQuery(regexp.QuoteMeta("SELECT DISTINCT file_id, NULLIF(CAST(volume_id AS CHAR), '') AS volume_id FROM " + table + " WHERE COALESCE(NULLIF(CAST(markdown_file_id AS CHAR), ''), NULLIF(JSON_UNQUOTE(JSON_EXTRACT(meta, '$.markdown_file_id')), ''), NULLIF(JSON_UNQUOTE(JSON_EXTRACT(meta, '$.md_file_id')), '')) = ?")).
			WithArgs("md-1").
			WillReturnRows(rows)
	}

	sourceFileIDs, err := backend.ensureMarkdownAllowed(context.Background(), tm, knowledge.WorkspaceScope{
		WorkspaceID:      "ws_1",
		SemanticModelIDs: []int64{101, 102},
		RAGSources: []knowledge.RAGSource{
			{
				SemanticModelID: 101,
				VectorTable:     "idx_a.kb_text_idx_a",
				EmbeddingModel:  "bge-m3",
				FileIDs:         []string{"file-a"},
			},
			{
				SemanticModelID: 102,
				VectorTable:     "idx_b.kb_text_idx_b",
				EmbeddingModel:  "bge-m3",
				FileIDs:         []string{"file-b"},
			},
		},
	}, "md-1")
	if err != nil {
		t.Fatalf("ensureMarkdownAllowed() error = %v", err)
	}
	if len(sourceFileIDs) != 1 || sourceFileIDs[0] != "file-b" {
		t.Fatalf("source file ids = %#v, want file-b", sourceFileIDs)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestCatalogParsedMarkdownBackendAllowsSemanticModelSourceWhenModelFilesEmpty(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()
	tm := transaction.NewManager(db)
	backend := &catalogParsedMarkdownBackend{}

	mock.ExpectQuery(regexp.QuoteMeta("SHOW COLUMNS FROM `idx_db`.`kb_text_idx`")).
		WillReturnRows(sqlmock.NewRows([]string{"Field", "Type"}).
			AddRow("file_id", "varchar").
			AddRow("meta", "json"))
	mock.ExpectQuery(regexp.QuoteMeta("SHOW COLUMNS FROM `idx_db`.`kb_text_idx`")).
		WillReturnRows(sqlmock.NewRows([]string{"Field", "Type"}).
			AddRow("file_id", "varchar").
			AddRow("meta", "json"))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT DISTINCT file_id, '' AS volume_id FROM `idx_db`.`kb_text_idx` WHERE COALESCE(NULLIF(JSON_UNQUOTE(JSON_EXTRACT(meta, '$.markdown_file_id')), ''), NULLIF(JSON_UNQUOTE(JSON_EXTRACT(meta, '$.md_file_id')), '')) = ?")).
		WithArgs("md-1").
		WillReturnRows(sqlmock.NewRows([]string{"file_id", "volume_id"}).AddRow("file-1", ""))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT source_id, kb_file_id, status, COALESCE(enabled, 1), expires_at, force_enabled_after_expiry, tags, segment_version_id, index_version\nFROM knowledge_base_sources\nWHERE model_id = ? AND kb_file_id IS NOT NULL AND kb_file_id != '' AND status <> 'removed' AND source_type IN ('local_file', 'catalog_file') AND kb_file_id IN (?)")).
		WithArgs(int64(14), "file-1").
		WillReturnRows(sqlmock.NewRows([]string{"source_id", "kb_file_id", "status", "enabled", "expires_at", "force_enabled_after_expiry", "tags", "segment_version_id", "index_version"}).
			AddRow("source-1", "file-1", "succeeded", true, nil, false, nil, nil, nil))

	sourceFileIDs, err := backend.ensureMarkdownAllowed(context.Background(), tm, knowledge.WorkspaceScope{
		WorkspaceID:      "ws_1",
		SemanticModelIDs: []int64{14},
		VectorTable:      "idx_db.kb_text_idx",
	}, "md-1")
	if err != nil {
		t.Fatalf("ensureMarkdownAllowed() error = %v", err)
	}
	if len(sourceFileIDs) != 1 || sourceFileIDs[0] != "file-1" {
		t.Fatalf("source file ids = %#v, want file-1", sourceFileIDs)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestCatalogParsedMarkdownBackendRejectsPendingOrFailedSemanticModelSource(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()
	tm := transaction.NewManager(db)
	backend := &catalogParsedMarkdownBackend{}

	mock.ExpectQuery(regexp.QuoteMeta("SHOW COLUMNS FROM `idx_db`.`kb_text_idx`")).
		WillReturnRows(sqlmock.NewRows([]string{"Field", "Type"}).
			AddRow("file_id", "varchar").
			AddRow("meta", "json"))
	mock.ExpectQuery(regexp.QuoteMeta("SHOW COLUMNS FROM `idx_db`.`kb_text_idx`")).
		WillReturnRows(sqlmock.NewRows([]string{"Field", "Type"}).
			AddRow("file_id", "varchar").
			AddRow("meta", "json"))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT DISTINCT file_id, '' AS volume_id FROM `idx_db`.`kb_text_idx` WHERE COALESCE(NULLIF(JSON_UNQUOTE(JSON_EXTRACT(meta, '$.markdown_file_id')), ''), NULLIF(JSON_UNQUOTE(JSON_EXTRACT(meta, '$.md_file_id')), '')) = ?")).
		WithArgs("md-1").
		WillReturnRows(sqlmock.NewRows([]string{"file_id", "volume_id"}).
			AddRow("file-pending", "").
			AddRow("file-failed", ""))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT source_id, kb_file_id, status, COALESCE(enabled, 1), expires_at, force_enabled_after_expiry, tags, segment_version_id, index_version\nFROM knowledge_base_sources\nWHERE model_id = ? AND kb_file_id IS NOT NULL AND kb_file_id != '' AND status <> 'removed' AND source_type IN ('local_file', 'catalog_file') AND kb_file_id IN (?,?)")).
		WithArgs(int64(14), "file-pending", "file-failed").
		WillReturnRows(sqlmock.NewRows([]string{"source_id", "kb_file_id", "status", "enabled", "expires_at", "force_enabled_after_expiry", "tags", "segment_version_id", "index_version"}).
			AddRow("source-pending", "file-pending", "pending", true, nil, false, nil, nil, nil).
			AddRow("source-failed", "file-failed", "failed", true, nil, false, nil, nil, nil))

	_, err = backend.ensureMarkdownAllowed(context.Background(), tm, knowledge.WorkspaceScope{
		WorkspaceID:      "ws_1",
		SemanticModelIDs: []int64{14},
		VectorTable:      "idx_db.kb_text_idx",
	}, "md-1")
	if err == nil || !strings.Contains(err.Error(), "markdown_file_id is outside the governed knowledge scope") {
		t.Fatalf("ensureMarkdownAllowed() error = %v, want governed scope rejection", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestCatalogParsedMarkdownBackendAllowsVolumeScopedMarkdown(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()
	tm := transaction.NewManager(db)
	backend := &catalogParsedMarkdownBackend{}

	mock.ExpectQuery(regexp.QuoteMeta("SHOW COLUMNS FROM `idx_db`.`kb_text_idx`")).
		WillReturnRows(sqlmock.NewRows([]string{"Field", "Type"}).
			AddRow("file_id", "varchar").
			AddRow("md_file_id", "varchar").
			AddRow("volume_id", "varchar").
			AddRow("meta", "json"))
	mock.ExpectQuery(regexp.QuoteMeta("SHOW COLUMNS FROM `idx_db`.`kb_text_idx`")).
		WillReturnRows(sqlmock.NewRows([]string{"Field", "Type"}).
			AddRow("file_id", "varchar").
			AddRow("md_file_id", "varchar").
			AddRow("volume_id", "varchar").
			AddRow("meta", "json"))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT DISTINCT file_id, NULLIF(CAST(volume_id AS CHAR), '') AS volume_id FROM `idx_db`.`kb_text_idx` WHERE COALESCE(NULLIF(CAST(md_file_id AS CHAR), ''), NULLIF(JSON_UNQUOTE(JSON_EXTRACT(meta, '$.markdown_file_id')), ''), NULLIF(JSON_UNQUOTE(JSON_EXTRACT(meta, '$.md_file_id')), '')) = ?")).
		WithArgs("md-1").
		WillReturnRows(sqlmock.NewRows([]string{"file_id", "volume_id"}).AddRow("file-1", "vol-1"))

	sourceFileIDs, err := backend.ensureMarkdownAllowed(context.Background(), tm, knowledge.WorkspaceScope{
		WorkspaceID:      "ws_1",
		SemanticModelIDs: []int64{14},
		VectorTable:      "idx_db.kb_text_idx",
		RAGSources: []knowledge.RAGSource{{
			SemanticModelID: 14,
			VolumeID:        "vol-1",
			VectorTable:     "idx_db.kb_text_idx",
		}},
	}, "md-1")
	if err != nil {
		t.Fatalf("ensureMarkdownAllowed() error = %v", err)
	}
	if len(sourceFileIDs) != 1 || sourceFileIDs[0] != "file-1" {
		t.Fatalf("source file ids = %#v, want file-1", sourceFileIDs)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestPlatformKnowledgeSourceTagsForFileIDsPreservesStoredTagValues(t *testing.T) {
	scope := knowledge.WorkspaceScope{
		RAGSources: []knowledge.RAGSource{
			{
				SourceTagsByFileID: map[string][]string{
					"file-1": {" policy ", "policy", ""},
					"file-2": {"finance"},
				},
			},
			{
				SourceTagsByFileID: map[string][]string{
					"file-1": {"policy", " risk "},
				},
			},
		},
	}

	got := platformKnowledgeSourceTagsForFileIDs(scope, []string{" file-1 ", "file-2", "file-1"})
	want := []string{" policy ", "policy", "", "finance", "policy", " risk "}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("source tags = %#v, want %#v", got, want)
	}
}

type platformKnowledgeFilterTestSemanticStore struct {
	model *tenant.SemanticModelRecord
}

func (s platformKnowledgeFilterTestSemanticStore) GetModel(_ context.Context, _ string, _ int64) (*tenant.SemanticModelRecord, error) {
	return s.model, nil
}

type platformKnowledgeFilterTestGovernanceStore struct {
	records      []PlatformKnowledgeSourceGovernanceRecord
	tableRecords []PlatformKnowledgeTableSourceGovernanceRecord
}

func (s platformKnowledgeFilterTestGovernanceStore) ListSourceGovernance(_ context.Context, _ string, _ int64, fileIDs []string) ([]PlatformKnowledgeSourceGovernanceRecord, error) {
	if len(fileIDs) == 0 {
		return append([]PlatformKnowledgeSourceGovernanceRecord(nil), s.records...), nil
	}
	allowed := map[string]struct{}{}
	for _, fileID := range fileIDs {
		allowed[fileID] = struct{}{}
	}
	out := make([]PlatformKnowledgeSourceGovernanceRecord, 0, len(s.records))
	for _, record := range s.records {
		if _, ok := allowed[record.FileID]; ok {
			out = append(out, record)
		}
	}
	return out, nil
}

func (s platformKnowledgeFilterTestGovernanceStore) ListTableSourceGovernance(_ context.Context, _ string, _ int64, tables []PlatformKnowledgeSemanticModelTableRef) ([]PlatformKnowledgeTableSourceGovernanceRecord, error) {
	allowed := map[string]struct{}{}
	for _, table := range tables {
		allowed[strings.TrimSpace(table.DBName)+"\x00"+strings.TrimSpace(table.TableName)] = struct{}{}
	}
	out := make([]PlatformKnowledgeTableSourceGovernanceRecord, 0, len(s.tableRecords))
	for _, record := range s.tableRecords {
		if _, ok := allowed[strings.TrimSpace(record.DBName)+"\x00"+strings.TrimSpace(record.TableName)]; ok {
			out = append(out, record)
		}
	}
	return out, nil
}

type platformKnowledgeFilterTestLegacyIndexResolver struct {
	constraints map[string]knowledge.RAGIndexVersionConstraint
	fileIDs     []string
	fileErr     error
	versionErr  error
	calls       []PlatformKnowledgeLegacyIndexResolveRequest
	fileCalls   []PlatformKnowledgeLegacyIndexResolveRequest
}

func (r *platformKnowledgeFilterTestLegacyIndexResolver) ResolveLegacyIndexVersions(_ context.Context, req PlatformKnowledgeLegacyIndexResolveRequest) (map[string]knowledge.RAGIndexVersionConstraint, error) {
	r.calls = append(r.calls, PlatformKnowledgeLegacyIndexResolveRequest{
		WorkspaceID: req.WorkspaceID,
		VectorTable: req.VectorTable,
		FileIDs:     append([]string(nil), req.FileIDs...),
	})
	if r.versionErr != nil {
		return nil, r.versionErr
	}
	out := make(map[string]knowledge.RAGIndexVersionConstraint, len(req.FileIDs))
	for _, fileID := range req.FileIDs {
		if constraint, ok := r.constraints[fileID]; ok {
			out[fileID] = constraint
		}
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

func (r *platformKnowledgeFilterTestLegacyIndexResolver) ResolveLegacyIndexFileIDs(_ context.Context, req PlatformKnowledgeLegacyIndexResolveRequest) ([]string, error) {
	r.fileCalls = append(r.fileCalls, PlatformKnowledgeLegacyIndexResolveRequest{
		WorkspaceID: req.WorkspaceID,
		VectorTable: req.VectorTable,
		FileIDs:     append([]string(nil), req.FileIDs...),
	})
	if r.fileErr != nil {
		return nil, r.fileErr
	}
	return append([]string(nil), r.fileIDs...), nil
}

type platformKnowledgeFilterTestImageIndexResolver struct {
	calls  int
	req    PlatformKnowledgeImageIndexResolveRequest
	result *PlatformKnowledgeImageIndexResolveResult
}

func (r *platformKnowledgeFilterTestImageIndexResolver) ResolveImageIndex(_ context.Context, req PlatformKnowledgeImageIndexResolveRequest) (*PlatformKnowledgeImageIndexResolveResult, error) {
	r.calls++
	r.req = req
	return r.result, nil
}

type platformKnowledgeFilterTestVectorIndexResolver struct {
	calls  int
	req    PlatformKnowledgeVectorIndexResolveRequest
	result *PlatformKnowledgeVectorIndexResolveResult
}

func (r *platformKnowledgeFilterTestVectorIndexResolver) ResolveVectorIndex(_ context.Context, req PlatformKnowledgeVectorIndexResolveRequest) (*PlatformKnowledgeVectorIndexResolveResult, error) {
	r.calls++
	r.req = req
	return r.result, nil
}

func platformKnowledgeFilterTestDescriptor(t *testing.T) agentruntime.AgentInstance {
	t.Helper()
	return agentruntime.AgentInstance{
		WorkspaceID: "ws_1",
		AgentID:     "explore",
		Tools: []agentruntime.AgentTool{
			issue11017ToolSnapshot(t, agenttools.ToolKindSearchRAGChunks),
			issue11017ToolSnapshot(t, agenttools.ToolKindFindRAGFiles),
			issue11017ToolSnapshot(t, agenttools.ToolKindReadParsedMarkdown),
			issue11017ToolSnapshot(t, agenttools.ToolKindSearchParsedMarkdown),
			issue11017ToolSnapshot(t, agenttools.ToolKindSearchVisualImage),
			issue11017ToolSnapshot(t, agenttools.ToolKindDescribeSchema),
			issue11017ToolSnapshot(t, agenttools.ToolKindQuerySQL),
			platformKnowledgeRemovedComputeResultTableSnapshot(),
			issue11017ToolSnapshot(t, agenttools.ToolKindSelectFinalSources),
			issue11017ToolSnapshot(t, agenttools.ToolKindSubmitFinalAnswer),
		},
	}
}

func TestPlatformKnowledgeRuntimeClockUsesAuthoritativeInteractiveDate(t *testing.T) {
	filter := NewPlatformKnowledgeToolFilter(&PlatformKnowledgeScopeResolver{})
	filter.now = func() time.Time {
		return time.Date(2026, 8, 12, 16, 0, 0, 0, time.UTC)
	}
	clock, err := filter.knowledgeRuntimeClock(nil)
	if err != nil {
		t.Fatalf("knowledgeRuntimeClock() error = %v", err)
	}
	if clock.CurrentDate != "2026-08-13" || clock.Timezone != "Asia/Shanghai" {
		t.Fatalf("runtime clock = %+v", clock)
	}
}

func TestPlatformKnowledgeRuntimeClockUsesFrozenAutomationDate(t *testing.T) {
	filter := NewPlatformKnowledgeToolFilter(&PlatformKnowledgeScopeResolver{})
	filter.now = func() time.Time {
		return time.Date(2026, 8, 12, 16, 0, 0, 0, time.UTC)
	}
	clock, err := filter.knowledgeRuntimeClock(map[string]any{
		"source_type":          platformKnowledgeAutomationSourceType,
		"execution_local_date": "2026-08-09",
		"execution_timezone":   "UTC",
	})
	if err != nil {
		t.Fatalf("knowledgeRuntimeClock() error = %v", err)
	}
	if clock.CurrentDate != "2026-08-09" || clock.Timezone != "UTC" {
		t.Fatalf("runtime clock = %+v", clock)
	}
}

func platformKnowledgeRemovedComputeResultTableSnapshot() agentruntime.AgentTool {
	return agentruntime.AgentTool{
		ID:   agenttools.ToolKindComputeResultTable,
		Name: "Compute Result Table",
		Kind: agenttools.ToolKindComputeResultTable,
	}
}

func platformKnowledgeFilterTestLegacyKnowledgeExplorePrompt(t *testing.T, locale string) string {
	t.Helper()
	defaults, err := agentresource.BuildSystemAgentDefaults(time.Unix(100, 0).UTC())
	if err != nil {
		t.Fatalf("BuildSystemAgentDefaults() error = %v", err)
	}
	for _, agent := range defaults {
		if agent.ID != agentresource.AgentDefaultKnowledgeExploreID {
			continue
		}
		prompt := agent.Instruction.SystemPrompt
		currentIdentity := "You are Data Agent, a data-grounded agent for Matrixflow knowledge bases."
		legacyIdentity := "You are Knowledge Explore, a data-grounded agent for Matrixflow knowledge bases."
		switch locale {
		case "zh-CN":
			i18n, ok := agent.Metadata["i18n"].(map[string]any)
			if !ok {
				t.Fatalf("knowledge explore i18n metadata = %#v", agent.Metadata)
			}
			localized, ok := i18n[locale].(map[string]any)
			if !ok {
				t.Fatalf("knowledge explore %s metadata = %#v", locale, i18n)
			}
			var okPrompt bool
			prompt, okPrompt = localized["system_prompt"].(string)
			if !okPrompt {
				t.Fatalf("knowledge explore %s system prompt = %#v", locale, localized["system_prompt"])
			}
			currentIdentity = "你是 Matrixflow 数据智能体，一个只能基于工具证据回答问题的数据探索 Agent。"
			legacyIdentity = "你是 Matrixflow Knowledge Explore，一个只能基于工具证据回答问题的数据探索 Agent。"
		case "en-US":
		default:
			t.Fatalf("unsupported locale %q", locale)
		}
		legacyPrompt := strings.Replace(prompt, currentIdentity, legacyIdentity, 1)
		if legacyPrompt == prompt {
			t.Fatalf("knowledge explore %s prompt did not contain current identity", locale)
		}
		return strings.TrimSpace(legacyPrompt)
	}
	t.Fatalf("knowledge explore system agent was not found")
	return ""
}

func platformKnowledgeFilterTestBindingKnowledge() agentruntime.AgentKnowledge {
	return agentruntime.AgentKnowledge{
		ID:   "50022",
		Name: "sssa",
		CatalogAssetRefs: []map[string]any{{
			"type": "table",
			"id":   "dimproductsubcategory",
			"config": map[string]any{
				"db_name":    "kb_50022_t_60008",
				"table_name": "dimproductsubcategory",
			},
		}},
	}
}

type platformKnowledgeFilterCompletedRunner struct{}

func (platformKnowledgeFilterCompletedRunner) RunStream(context.Context, agentruntimev2.Options) (<-chan agentruntimev2.Event, <-chan *agentruntimev2.Result) {
	events := make(chan agentruntimev2.Event)
	results := make(chan *agentruntimev2.Result, 1)
	close(events)
	results <- &agentruntimev2.Result{
		FinishReason: agentruntimev2.FinishComplete,
		FinalText:    "完成",
		Messages: []agentruntimev2.Message{
			{Role: agentruntimev2.RoleUser, Content: "查询当前选中知识库"},
			{
				Role: agentruntimev2.RoleAssistant,
				ToolCalls: []agentruntimev2.ToolCall{{
					ID:   "call_select",
					Name: agenttools.ToolKindSelectFinalSources,
				}},
			},
			{
				Role:       agentruntimev2.RoleTool,
				Name:       agenttools.ToolKindSelectFinalSources,
				ToolCallID: "call_select",
				Content:    `{"ok":true,"accepted":true,"selected":true,"sources":[]}`,
			},
			{Role: agentruntimev2.RoleAssistant, Content: "完成"},
		},
	}
	close(results)
	return events, results
}

func platformKnowledgeManifestToolIDs(value any) map[string]struct{} {
	out := make(map[string]struct{})
	for _, item := range platformKnowledgeManifestMaps(value) {
		id, _ := item["id"].(string)
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		out[id] = struct{}{}
	}
	return out
}

func platformKnowledgeManifestKnowledgeIDs(value any) map[string]struct{} {
	out := make(map[string]struct{})
	for _, item := range platformKnowledgeManifestMaps(value) {
		id, _ := item["id"].(string)
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		out[id] = struct{}{}
	}
	return out
}

func platformKnowledgeManifestMaps(value any) []map[string]any {
	switch typed := value.(type) {
	case []map[string]any:
		return typed
	case []any:
		out := make([]map[string]any, 0, len(typed))
		for _, item := range typed {
			mapped, ok := item.(map[string]any)
			if !ok {
				continue
			}
			out = append(out, mapped)
		}
		return out
	default:
		return nil
	}
}

func platformKnowledgeFilterTestContains(tools []agentruntime.AgentTool, id string) bool {
	for _, tool := range tools {
		if tool.ID == id {
			return true
		}
	}
	return false
}

func containsAll(text string, needles ...string) bool {
	for _, needle := range needles {
		if !strings.Contains(text, needle) {
			return false
		}
	}
	return true
}

func containsAny(text string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(text, needle) {
			return true
		}
	}
	return false
}
