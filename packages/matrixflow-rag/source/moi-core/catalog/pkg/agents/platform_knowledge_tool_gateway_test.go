package agents

import (
	"context"
	"encoding/json"
	"testing"

	agenttools "github.com/matrixflow/moi-core/agent-tools"
	"github.com/matrixflow/moi-core/agent-tools/knowledge"
	"github.com/matrixflow/moi-core/catalog/pkg/agentruntime"
)

func TestPlatformKnowledgeGatewayInvokesSubmitFinalAnswerWithoutSideEffectAuthorizer(t *testing.T) {
	gateway := agentruntime.NewPlatformToolGateway(
		agentruntime.NewStaticToolGateway(),
		agentruntime.WithPlatformToolExecutors(PlatformKnowledgeToolExecutors(&knowledge.Registry{}, nil)),
	)

	for _, testCase := range []struct {
		name        string
		prepare     func(context.Context) context.Context
		arguments   json.RawMessage
		sourceCount int
	}{
		{
			name: "empty_sources",
			prepare: func(ctx context.Context) context.Context {
				rc := knowledge.NewRunContext()
				rc.RecordRAGChunksArtifact("rag_chunks_empty", knowledge.SearchRAGChunksResponse{})
				return knowledge.ContextWithRunContext(ctx, rc)
			},
			arguments:   json.RawMessage(`{"answer":"没有在当前知识范围内找到相关结果。","sources":[]}`),
			sourceCount: 0,
		},
		{
			name: "rag_source",
			prepare: func(ctx context.Context) context.Context {
				rc := knowledge.NewRunContext()
				rc.RecordRAGChunksArtifact("rag_chunks_call_1", knowledge.SearchRAGChunksResponse{Chunks: []knowledge.RAGChunkHit{{
					ChunkID:  "chunk_1",
					FileID:   "file_1",
					FileName: "new_moi.md",
					Content:  "new moi update flow",
				}}})
				return knowledge.ContextWithRunContext(ctx, rc)
			},
			arguments:   json.RawMessage(`{"answer":"new_moi.md 记录了相关流程。","sources":[{"type":"rag_chunk","chunk_id":"chunk_1"}]}`),
			sourceCount: 1,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			ctx := agentruntime.ContextWithRunScopedState(context.Background())
			if testCase.prepare != nil {
				ctx = testCase.prepare(ctx)
			}

			result, err := gateway.Invoke(ctx, agentruntime.ToolInvokeRequest{
				WorkspaceID: "ws_1",
				Manifest:    platformKnowledgeSubmitFinalAnswerManifest(t),
				TaskID:      "task_1",
				ToolID:      agenttools.ToolKindSubmitFinalAnswer,
				CallID:      "call_submit",
				Arguments:   testCase.arguments,
			})
			if err != nil {
				t.Fatalf("Invoke() error = %v", err)
			}
			if result.Output["accepted"] != true || result.Output["source_count"] != testCase.sourceCount {
				t.Fatalf("result output = %+v", result.Output)
			}
		})
	}
}

func TestPlatformKnowledgeGatewayUsesToolCallerScopeWhenRuntimeScopeMissing(t *testing.T) {
	find := &recordingPlatformKnowledgeFindRAGFiles{}
	gateway := agentruntime.NewPlatformToolGateway(
		agentruntime.NewStaticToolGateway(),
		agentruntime.WithPlatformToolExecutors(PlatformKnowledgeToolExecutors(&knowledge.Registry{FindRAGFiles: find}, nil)),
	)

	result, err := gateway.Invoke(agentruntime.ContextWithRunScopedState(context.Background()), agentruntime.ToolInvokeRequest{
		WorkspaceID: "ws_1",
		Manifest:    platformKnowledgeFindRAGFilesManifest(t),
		TaskID:      "task_1",
		ToolID:      agenttools.ToolKindFindRAGFiles,
		CallID:      "call_find",
		Arguments:   json.RawMessage(`{"query":"integration evidence","top_k":1}`),
		Caller: agentruntime.ToolCaller{
			UserID: "user_1",
		},
	})
	if err != nil {
		t.Fatalf("Invoke() error = %v", err)
	}
	if result.Output["row_count"] != float64(0) {
		t.Fatalf("result output = %+v", result.Output)
	}
	if find.request.Scope.WorkspaceID != "ws_1" || find.request.Scope.UserID != "user_1" {
		t.Fatalf("knowledge scope = %+v", find.request.Scope)
	}
}

type recordingPlatformKnowledgeFindRAGFiles struct {
	request knowledge.FindRAGFilesRequest
}

func (r *recordingPlatformKnowledgeFindRAGFiles) Execute(_ context.Context, req knowledge.FindRAGFilesRequest) (*knowledge.FindRAGFilesResponse, error) {
	r.request = req
	return &knowledge.FindRAGFilesResponse{Files: []knowledge.RAGFileRef{}, RowCount: 0}, nil
}

func platformKnowledgeSubmitFinalAnswerManifest(t *testing.T) agentruntime.RuntimeManifest {
	t.Helper()
	definition, ok := agenttools.FindDefinition(agenttools.ToolKindSubmitFinalAnswer)
	if !ok {
		t.Fatalf("tool definition %s not found", agenttools.ToolKindSubmitFinalAnswer)
	}
	return agentruntime.RuntimeManifest{
		ID:          "manifest_1",
		WorkspaceID: "ws_1",
		TaskID:      "task_1",
		Body: map[string]any{
			"tools": []map[string]any{{
				"id":                definition.ID,
				"name":              definition.Name,
				"kind":              definition.ID,
				"description":       definition.Description,
				"side_effect_class": definition.SideEffectClass,
				"input_schema":      definition.InputSchema,
				"output_schema":     definition.OutputSchema,
				"metadata":          definition.Metadata,
			}},
		},
	}
}

func platformKnowledgeFindRAGFilesManifest(t *testing.T) agentruntime.RuntimeManifest {
	t.Helper()
	definition, ok := agenttools.FindDefinition(agenttools.ToolKindFindRAGFiles)
	if !ok {
		t.Fatalf("tool definition %s not found", agenttools.ToolKindFindRAGFiles)
	}
	return agentruntime.RuntimeManifest{
		ID:          "manifest_1",
		WorkspaceID: "ws_1",
		TaskID:      "task_1",
		Body: map[string]any{
			"tools": []map[string]any{{
				"id":                definition.ID,
				"name":              definition.Name,
				"kind":              definition.ID,
				"description":       definition.Description,
				"side_effect_class": definition.SideEffectClass,
				"input_schema":      definition.InputSchema,
				"output_schema":     definition.OutputSchema,
				"metadata":          definition.Metadata,
			}},
		},
	}
}
