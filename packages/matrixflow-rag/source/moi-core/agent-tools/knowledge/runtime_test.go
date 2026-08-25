package knowledge

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	agentruntimev2 "github.com/matrixflow/moi-core/agent-runtime-v2"
)

func TestRuntimeKnowledgeToolRequiresRunContextWithoutRunState(t *testing.T) {
	tool := runtimeKnowledgeTool{
		Tool: agentruntimev2.NewTool("echo", "Echo", nil, func(context.Context, json.RawMessage) (*agentruntimev2.ToolResult, error) {
			return &agentruntimev2.ToolResult{Data: map[string]any{"ok": true}}, nil
		}),
	}

	_, err := tool.Execute(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "knowledge runtime run context is not configured") {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestRuntimeKnowledgeToolUsesExistingRunContextWithoutRunState(t *testing.T) {
	rc := NewRunContext()
	tool := runtimeKnowledgeTool{
		Tool: agentruntimev2.NewTool("echo", "Echo", nil, func(ctx context.Context, _ json.RawMessage) (*agentruntimev2.ToolResult, error) {
			if RunContextFrom(ctx) != rc {
				t.Fatalf("run context = %p, want %p", RunContextFrom(ctx), rc)
			}
			return &agentruntimev2.ToolResult{Data: map[string]any{"ok": true}}, nil
		}),
	}

	if _, err := tool.Execute(ContextWithRunContext(context.Background(), rc), nil); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestRuntimeKnowledgeToolReusesRunStateContextAcrossCalls(t *testing.T) {
	state := &runtimeKnowledgeToolTestRunState{values: map[string]any{}}
	var seen []*RunContext
	tool := runtimeKnowledgeTool{
		Tool: agentruntimev2.NewTool("echo", "Echo", nil, func(ctx context.Context, _ json.RawMessage) (*agentruntimev2.ToolResult, error) {
			seen = append(seen, RunContextFrom(ctx))
			return &agentruntimev2.ToolResult{Data: map[string]any{"ok": true}}, nil
		}),
		runState: state,
	}

	if _, err := tool.Execute(context.Background(), nil); err != nil {
		t.Fatalf("first Execute() error = %v", err)
	}
	if _, err := tool.Execute(context.Background(), nil); err != nil {
		t.Fatalf("second Execute() error = %v", err)
	}
	if len(seen) != 2 || seen[0] == nil || seen[0] != seen[1] {
		t.Fatalf("run contexts = %+v", seen)
	}
}

func TestRuntimeSelectFinalSourcesAllowsEmptySourcesWithoutCitableEvidenceTool(t *testing.T) {
	state := &runtimeKnowledgeToolTestRunState{values: map[string]any{}}
	factory := RuntimeToolExecutors(&Registry{}, nil)[ToolNameSelectFinalSources]
	tool, err := factory(context.Background(), RuntimeToolRequest{
		Manifest: RuntimeManifest{Body: map[string]any{
			"tools": []any{map[string]any{
				"id":   ToolNameSelectFinalSources,
				"kind": ToolNameSelectFinalSources,
			}},
		}},
		RunState: state,
	}, RuntimeToolSnapshot{ID: ToolNameSelectFinalSources, Kind: ToolNameSelectFinalSources})
	if err != nil {
		t.Fatalf("select factory returned error: %v", err)
	}
	invocationTool, ok := tool.(agentruntimev2.InvocationTool)
	if !ok {
		t.Fatalf("runtime select tool does not implement InvocationTool")
	}
	result, err := invocationTool.ExecuteInvocation(context.Background(), agentruntimev2.ToolInvocation{CallID: "call_select"}, json.RawMessage(`{"sources":[]}`))
	if err != nil {
		t.Fatalf("select returned error: %v", err)
	}
	if output := result.Data.(map[string]any); output["source_count"] != 0 {
		t.Fatalf("source_count = %v, want 0", output["source_count"])
	}
	runContext, ok := state.values[runtimeRunStateKey].(*RunContext)
	if !ok {
		t.Fatalf("run context = %#v", state.values[runtimeRunStateKey])
	}
	if selected, ok := runContext.SelectedFinalAnswerSources(); !ok || len(selected) != 0 {
		t.Fatalf("selected sources = (%v,%t), want empty selected", selected, ok)
	}
}

func TestRuntimeSelectFinalSourcesRequiresRetrievalWhenCitableEvidenceToolIsExecutable(t *testing.T) {
	state := &runtimeKnowledgeToolTestRunState{values: map[string]any{}}
	factory := RuntimeToolExecutors(&Registry{}, nil)[ToolNameSelectFinalSources]
	tool, err := factory(context.Background(), RuntimeToolRequest{
		Manifest: RuntimeManifest{Body: map[string]any{
			"tools": []any{
				map[string]any{"id": ToolNameSearchRAGChunks, "kind": ToolNameSearchRAGChunks},
				map[string]any{"id": ToolNameSelectFinalSources, "kind": ToolNameSelectFinalSources},
			},
		}},
		RunState: state,
	}, RuntimeToolSnapshot{ID: ToolNameSelectFinalSources, Kind: ToolNameSelectFinalSources})
	if err != nil {
		t.Fatalf("select factory returned error: %v", err)
	}
	invocationTool, ok := tool.(agentruntimev2.InvocationTool)
	if !ok {
		t.Fatalf("runtime select tool does not implement InvocationTool")
	}
	_, err = invocationTool.ExecuteInvocation(context.Background(), agentruntimev2.ToolInvocation{CallID: "call_select"}, json.RawMessage(`{"sources":[]}`))
	if err == nil || !strings.Contains(err.Error(), "sources cannot be empty before a citable evidence retrieval completes") {
		t.Fatalf("select error = %v, want missing retrieval rejection", err)
	}
}

func TestCompactRAGSourcesPreservesDocumentGovernanceMetadata(t *testing.T) {
	got := CompactRAGSources([]RAGSource{
		{
			SemanticModelID: 42,
			VectorTable:     "text_vec",
			EmbeddingModel:  "bge-m3",
			SourceRowIDs:    []string{" source_a ", "source_a"},
			FileIDs:         []string{"file_a"},
			SourceTags:      []string{" policy ", "policy", ""},
			SourceRowIDByFileID: map[string]string{
				" file_a ": " source_a ",
			},
			SourceTagsByFileID: map[string][]string{
				" file_a ": {" policy ", "finance", ""},
			},
			IndexVersionConstraintByFileID: map[string]RAGIndexVersionConstraint{
				" file_a ": {Kind: RAGIndexVersionConstraintValue, Value: 0},
			},
			Metadata: map[string]string{" source_tags ": " policy,finance "},
		},
		{
			SemanticModelID: 42,
			VectorTable:     "text_vec",
			EmbeddingModel:  "bge-m3",
			SourceRowIDs:    []string{"source_b"},
			FileIDs:         []string{"file_b", "file_a"},
			SourceTags:      []string{"finance", " policy "},
			SourceRowIDByFileID: map[string]string{
				"file_b": "source_b",
			},
			SourceTagsByFileID: map[string][]string{
				"file_a": {"risk", " policy "},
				"file_b": {"finance"},
			},
			IndexVersionConstraintByFileID: map[string]RAGIndexVersionConstraint{
				"file_b": {Kind: RAGIndexVersionConstraintNull},
			},
			Metadata: map[string]string{"source_row_ids": "source_a,source_b"},
		},
	})

	if len(got) != 1 {
		t.Fatalf("CompactRAGSources() length = %d, want 1: %+v", len(got), got)
	}
	source := got[0]
	if strings.Join(source.SourceRowIDs, ",") != "source_a,source_b" {
		t.Fatalf("source row ids = %#v", source.SourceRowIDs)
	}
	if strings.Join(source.FileIDs, ",") != "file_a,file_b" {
		t.Fatalf("file ids = %#v", source.FileIDs)
	}
	if source.SourceRowIDByFileID["file_a"] != "source_a" || source.SourceRowIDByFileID["file_b"] != "source_b" {
		t.Fatalf("source row id by file = %#v", source.SourceRowIDByFileID)
	}
	wantSourceTags := []string{" policy ", "policy", "", "finance", " policy "}
	if !reflect.DeepEqual(source.SourceTags, wantSourceTags) {
		t.Fatalf("source tags = %#v, want %#v", source.SourceTags, wantSourceTags)
	}
	if got, want := source.SourceTagsByFileID["file_a"], []string{" policy ", "finance", "", "risk", " policy "}; !reflect.DeepEqual(got, want) {
		t.Fatalf("file_a tags = %#v, want %#v", got, want)
	}
	if got, want := source.SourceTagsByFileID["file_b"], []string{"finance"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("file_b tags = %#v, want %#v", got, want)
	}
	if source.Metadata["source_tags"] != "policy,finance" || source.Metadata["source_row_ids"] != "source_a,source_b" {
		t.Fatalf("metadata = %#v", source.Metadata)
	}
	if got := source.IndexVersionConstraintByFileID["file_a"]; got.Kind != RAGIndexVersionConstraintValue || got.Value != 0 {
		t.Fatalf("file_a index version constraint = %#v, want value 0", got)
	}
	if got := source.IndexVersionConstraintByFileID["file_b"]; got.Kind != RAGIndexVersionConstraintNull {
		t.Fatalf("file_b index version constraint = %#v, want NULL", got)
	}
}

func TestRuntimeToolScopeTurnSelectedSemanticModelSkipsUnselectedManifestTables(t *testing.T) {
	scope, err := RuntimeToolScope(context.Background(), RuntimeToolRequest{
		WorkspaceID: "ws_1",
		Manifest: RuntimeManifest{
			WorkspaceID: "ws_1",
			Body: map[string]any{
				"knowledge_bases": []map[string]any{runtimeToolScopeTestBindingKnowledge()},
			},
		},
		TurnMetadata: map[string]any{
			"semantic_model_ids": []any{float64(60002)},
		},
	}, nil)
	if err != nil {
		t.Fatalf("RuntimeToolScope() error = %v", err)
	}
	if scope.DBName != "" {
		t.Fatalf("DBName = %q, want empty", scope.DBName)
	}
	if len(scope.Tables) != 0 {
		t.Fatalf("Tables = %#v, want empty", scope.Tables)
	}
	if !reflect.DeepEqual(scope.SemanticModelIDs, []int64{60002}) {
		t.Fatalf("SemanticModelIDs = %#v, want [60002]", scope.SemanticModelIDs)
	}
}

func TestRuntimeToolScopeWithoutTurnSelectedSemanticModelKeepsManifestTables(t *testing.T) {
	scope, err := RuntimeToolScope(context.Background(), RuntimeToolRequest{
		WorkspaceID: "ws_1",
		Manifest: RuntimeManifest{
			WorkspaceID: "ws_1",
			Body: map[string]any{
				"knowledge_bases": []map[string]any{runtimeToolScopeTestBindingKnowledge()},
			},
		},
	}, nil)
	if err != nil {
		t.Fatalf("RuntimeToolScope() error = %v", err)
	}
	if scope.DBName != "kb_50022_t_60008" {
		t.Fatalf("DBName = %q, want binding db", scope.DBName)
	}
	if !reflect.DeepEqual(scope.Tables, []string{"kb_50022_t_60008.dimproductsubcategory"}) {
		t.Fatalf("Tables = %#v, want qualified binding identity", scope.Tables)
	}
	if !reflect.DeepEqual(scope.SemanticModelIDs, []int64{50022}) {
		t.Fatalf("SemanticModelIDs = %#v, want [50022]", scope.SemanticModelIDs)
	}
}

// Manifest multi-db table refs must keep database.table identities (not flatten
// to bare table_name). Incomplete refs contribute no SQL table scope.
func TestRuntimeToolScopeManifestPreservesMultiDBTableIdentities(t *testing.T) {
	scope, err := RuntimeToolScope(context.Background(), RuntimeToolRequest{
		WorkspaceID: "ws_1",
		Manifest: RuntimeManifest{
			WorkspaceID: "ws_1",
			Body: map[string]any{
				"knowledge_bases": []map[string]any{{
					"id":   "42",
					"name": "multi",
					"catalog_asset_refs": []map[string]any{
						{
							"type": "table",
							"id":   "orders",
							"config": map[string]any{
								"db_name":    "sales",
								"table_name": "orders",
							},
						},
						{
							"type": "table",
							"id":   "tickets",
							"config": map[string]any{
								"db_name":    "support",
								"table_name": "tickets",
							},
						},
						{
							// Incomplete: missing db_name — must not seed bare name.
							"type": "table",
							"id":   "orphan",
							"config": map[string]any{
								"table_name": "orphan",
							},
						},
						{
							// Incomplete: missing table_name — must not guess from id/name.
							"type": "table",
							"id":   "asset-42",
							"config": map[string]any{
								"db_name": "sales",
							},
						},
						{
							// Incomplete: name/refID must not substitute for table_name.
							"type": "table",
							"id":   "asset-99",
							"config": map[string]any{
								"db_name": "sales",
								"name":    "should_not_use",
							},
						},
					},
				}},
			},
		},
	}, nil)
	if err != nil {
		t.Fatalf("RuntimeToolScope() error = %v", err)
	}
	if !reflect.DeepEqual(scope.Tables, []string{"sales.orders", "support.tickets"}) {
		t.Fatalf("Tables = %#v, want qualified multi-db identities only (no refID/name guess)", scope.Tables)
	}
	// First complete ref still seeds convenience DBName; Resolve clears multi-db.
	if scope.DBName != "sales" {
		t.Fatalf("DBName = %q, want sales (first complete ref)", scope.DBName)
	}
}

func runtimeToolScopeTestBindingKnowledge() map[string]any {
	return map[string]any{
		"id":   "50022",
		"name": "sssa",
		"catalog_asset_refs": []map[string]any{{
			"type": "table",
			"id":   "dimproductsubcategory",
			"config": map[string]any{
				"db_name":    "kb_50022_t_60008",
				"table_name": "dimproductsubcategory",
			},
		}},
	}
}

type runtimeKnowledgeToolTestRunState struct {
	values map[string]any
}

func (s *runtimeKnowledgeToolTestRunState) GetOrCreate(key string, create func() any) any {
	if value, ok := s.values[key]; ok {
		return value
	}
	value := create()
	s.values[key] = value
	return value
}
