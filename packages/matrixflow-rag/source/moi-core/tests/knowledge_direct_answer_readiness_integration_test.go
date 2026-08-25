package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	agentruntimev2 "github.com/matrixflow/moi-core/agent-runtime-v2"
	agenttools "github.com/matrixflow/moi-core/agent-tools"
	catalogruntime "github.com/matrixflow/moi-core/catalog/pkg/agentruntime"
	"github.com/matrixflow/moi-core/catalog/pkg/embed"
	moi "github.com/matrixflow/moi-core/go-sdk"
	"github.com/matrixflow/moi-core/tests/framework"
	workerpkg "github.com/matrixflow/moi-core/workers/go-worker/pkg/worker"
	"github.com/stretchr/testify/require"
)

const knowledgeDirectAnswerSentinel = "knowledge-direct-answer-ci-sentinel"

// knowledgeDirectAnswerRuntime drives the real explore A2A loop through the
// RAG tool and records the tool result before submitting the final answer.
// This keeps the regression deterministic while still exercising production
// scope resolution, retrieval, and terminal-answer handling.
type knowledgeDirectAnswerRuntime struct {
	ragResults chan string
	answers    chan string
}

func newKnowledgeDirectAnswerRuntime() *knowledgeDirectAnswerRuntime {
	return &knowledgeDirectAnswerRuntime{
		ragResults: make(chan string, 4),
		answers:    make(chan string, 4),
	}
}

func (r *knowledgeDirectAnswerRuntime) StreamResponses(_ context.Context, req agentruntimev2.ResponsesAPIRequest, onEvent func(agentruntimev2.ResponseEvent) error) error {
	names, err := testResponsesToolNameSetFromRequest(req)
	if err != nil {
		return err
	}
	if knowledgeDirectAnswerHasToolResult(req, agenttools.ToolKindSelectFinalSources) {
		return r.emitFinalAnswer(onEvent)
	}
	if len(names) == 0 {
		return emitTestResponsesText(onEvent, "knowledge retrieval is unavailable")
	}
	if _, ok := names[agenttools.ToolKindSearchRAGChunks]; ok && !knowledgeDirectAnswerHasToolResult(req, agenttools.ToolKindSearchRAGChunks) {
		return emitTestResponsesToolCall(onEvent, agentruntimev2.ToolCall{
			ID:        "call_knowledge_direct_answer_search",
			Name:      agenttools.ToolKindSearchRAGChunks,
			Arguments: json.RawMessage(`{"keywords":["knowledge-direct-answer-ci-sentinel"],"max_hits":1}`),
			Kind:      agentruntimev2.ToolCallKindFunction,
		})
	}
	if result, ok := knowledgeDirectAnswerRAGResult(req); ok && !knowledgeDirectAnswerHasToolResult(req, agenttools.ToolKindSelectFinalSources) {
		select {
		case r.ragResults <- result:
		default:
		}
	}
	if _, ok := names[agenttools.ToolKindSelectFinalSources]; ok &&
		knowledgeDirectAnswerHasToolResult(req, agenttools.ToolKindSearchRAGChunks) &&
		!knowledgeDirectAnswerHasToolResult(req, agenttools.ToolKindSelectFinalSources) {
		chunkID := agentResourceRuntimeFirstRAGChunkID(req)
		if chunkID == "" {
			return fmt.Errorf("knowledge runtime did not receive a RAG chunk result")
		}
		args, err := json.Marshal(map[string]any{
			"sources": []map[string]any{{"type": "rag_chunk", "chunk_id": chunkID}},
		})
		if err != nil {
			return err
		}
		return emitTestResponsesToolCall(onEvent, agentruntimev2.ToolCall{
			ID:        "call_knowledge_direct_answer_select_sources",
			Name:      agenttools.ToolKindSelectFinalSources,
			Arguments: args,
			Kind:      agentruntimev2.ToolCallKindFunction,
		})
	}
	return emitTestResponsesText(onEvent, "knowledge direct answer complete")
}

func (r *knowledgeDirectAnswerRuntime) CompactResponses(context.Context, agentruntimev2.CompactionInput, map[string]string) ([]agentruntimev2.ResponseItem, error) {
	return []agentruntimev2.ResponseItem{}, nil
}

func (r *knowledgeDirectAnswerRuntime) emitFinalAnswer(onEvent func(agentruntimev2.ResponseEvent) error) error {
	answer := "direct answer: " + knowledgeDirectAnswerSentinel
	select {
	case r.answers <- answer:
	default:
	}
	return emitTestResponsesText(onEvent, answer)
}

func (r *knowledgeDirectAnswerRuntime) Drain() {
	for {
		select {
		case <-r.ragResults:
		case <-r.answers:
		default:
			return
		}
	}
}

func knowledgeDirectAnswerRAGResult(req agentruntimev2.ResponsesAPIRequest) (string, bool) {
	messages := agentruntimev2.MessagesFromResponseItems(req.Input)
	for i := len(messages) - 1; i >= 0; i-- {
		message := messages[i]
		if message.Role != agentruntimev2.RoleTool {
			continue
		}
		if strings.TrimSpace(message.Name) == agenttools.ToolKindSearchRAGChunks || message.ToolCallID == "call_knowledge_direct_answer_search" {
			return message.Content, true
		}
	}
	return "", false
}

func knowledgeDirectAnswerHasToolResult(req agentruntimev2.ResponsesAPIRequest, toolName string) bool {
	messages := agentruntimev2.MessagesFromResponseItems(req.Input)
	for i := len(messages) - 1; i >= 0; i-- {
		message := messages[i]
		if message.Role != agentruntimev2.RoleTool {
			continue
		}
		if strings.TrimSpace(message.Name) == toolName ||
			(toolName == agenttools.ToolKindSearchRAGChunks && message.ToolCallID == "call_knowledge_direct_answer_search") {
			return true
		}
	}
	return false
}

// TestSemanticModelKnowledgeDirectAnswerReadiness protects the two windows in
// which an indexed document must be queryable before the knowledge-base resource
// page has reconciled governance state:
//
//  1. a workflow parses, indexes, and registers lineage for a KB directly;
//  2. Backend create/append has written an enabled pending source and the
//     default RAG workflow has materialized that source's vectors.
func TestSemanticModelKnowledgeDirectAnswerReadiness(t *testing.T) {
	if testing.Short() {
		t.Skip("skip knowledge direct-answer MatrixOne integration in short mode")
	}

	runtime := newKnowledgeDirectAnswerRuntime()
	framework.RunMOITestsWithOptionsNoParallel(t, framework.TestOptions{
		IsolatedServer: true,
		ServerOptions: []any{
			embed.WithAgentRuntimeGrantKey("test-runtime-grant-key", 600),
			embed.WithAgentRuntimeV2Backend(catalogruntime.NewAgentRuntimeV2Backend(
				catalogruntime.WithAgentRuntimeV2Responses(runtime),
			)),
		},
	}, func(env *framework.TestEnv) {
		env.RequireSharedWorkspace(t)
		ctx := context.Background()
		workspaceID := env.SharedWorkspaceID
		client, err := env.GetSharedClient()
		require.NoError(t, err)

		worker, err := workerpkg.New(workerpkg.Config{
			Endpoint:     env.ServerEndpoint,
			WorkerAPIKey: env.SystemAPIKey,
			WorkerID:     framework.UniqueWorkerID("knowledge-direct-answer"),
		}, nil)
		require.NoError(t, err)
		require.NoError(t, worker.Start(ctx))
		defer func() { _ = worker.Stop() }()

		var embeddingHits atomic.Int64
		embeddingServer := newEmbeddingMockServer(t, &embeddingHits)
		defer embeddingServer.Close()
		createEmbeddingBackend(t, ctx, client, workspaceID, "knowledge-direct-answer-embedding", embeddingServer.URL, []string{"test-embed"})

		modelName := fmt.Sprintf("knowledge-direct-answer-llm-%d", time.Now().UnixNano())
		llm := newKnowledgeMockLLMServer(t)
		defer llm.Close()
		llmBackendID, cleanupLLMBackend := ensureMockLLMBackend(t, client, workspaceID, "knowledge-direct-answer", modelName, llm.URL())
		defer cleanupLLMBackend()

		tenantDB, err := env.OpenOwnerWorkspaceDB(ctx, workspaceID)
		require.NoError(t, err)
		defer tenantDB.Close()

		runScenario := func(t *testing.T, label string, pendingSource bool) {
			t.Helper()
			vectorTable := fmt.Sprintf("kb_direct_answer_%s_%d", label, time.Now().UnixNano())
			modelFiles, err := json.Marshal(map[string]any{
				"file_ids":        []string{},
				"vector_table":    vectorTable,
				"embedding_model": "test-embed",
			})
			require.NoError(t, err)
			model, err := client.SemanticModels(workspaceID).Create(ctx, &moi.SemanticModelUpsertRequest{
				Name:        fmt.Sprintf("knowledge-direct-answer-%s-%d", label, time.Now().UnixNano()),
				Description: "knowledge direct answer readiness regression",
				Tables:      json.RawMessage(`[]`),
				Files:       modelFiles,
			})
			require.NoError(t, err)
			defer func() { _ = client.SemanticModels(workspaceID).Delete(context.Background(), model.ID) }()

			fileName := fmt.Sprintf("knowledge-direct-answer-%s-%d.txt", label, time.Now().UnixNano())
			uploaded, err := client.Files().UploadBytes(ctx, workspaceID, fileName, []byte("The document proves "+knowledgeDirectAnswerSentinel+" is ready for retrieval."))
			require.NoError(t, err)

			sourceID := fmt.Sprintf("kb-direct-answer-%s-%d", label, model.ID)
			if pendingSource {
				_, err = tenantDB.ExecContext(ctx, `
INSERT INTO knowledge_base_sources
    (source_id, model_id, catalog_id, database_id, raw_volume_id, processed_volume_id,
     source_type, source_file_id, kb_file_id, status, enabled, created_by, updated_by)
VALUES (?, ?, 1, 1, 1, 1, 'local_file', ?, ?, 'pending', true, 'ci', 'ci')`,
					sourceID, model.ID, uploaded.FileID, uploaded.FileID)
				require.NoError(t, err)
			}

			workflowYAML := `
workflow:
  name: knowledge-direct-answer
  root: root
root:
  chain:
    - work_item:
        name: parse
        id: moi:document.parse
        input:
          file_id: '{{ .vars.file_id }}'
        save:
          parsed_documents: '.documents'
    - work_item:
        name: split
        id: moi:parser.split.documents.length
        input:
          documents: '{{ .state.parsed_documents }}'
          chunk_size: 128
          overlap: 0
          enable_level_based_split: false
        save:
          chunked_documents: '.documents'
    - work_item:
        name: write_parsed_docset
        id: moi:files.write_documents
        input:
          documents: '{{ .state.chunked_documents }}'
          file_name: '{{ .vars.file_name }}.parsed.jsonl'
        save:
          parsed_file_id: '.file_id'
    - work_item:
        name: build_index
        id: moi:knowledge.index.build
        input:
          documents: '{{ .state.chunked_documents }}'
          table_name: '{{ .vars.vector_table }}'
          embedding_model: '{{ .vars.embedding_model }}'
          embedding_dimension: 3
          file_id: '{{ .vars.file_id }}'
    - work_item:
        name: register_lineage
        id: moi:data.lineage.register
        input:
          source_file_id: '{{ .vars.file_id }}'
          source_file_name: '{{ .vars.file_name }}'
          parsed_file_id: '{{ .state.parsed_file_id }}'
          vector_table: '{{ .vars.vector_table }}'
          embedding_model: '{{ .vars.embedding_model }}'
`
			vars, err := json.Marshal(map[string]any{
				"file_id":         uploaded.FileID,
				"file_name":       fileName,
				"vector_table":    vectorTable,
				"embedding_model": "test-embed",
			})
			require.NoError(t, err)
			beforeEmbeddingHits := embeddingHits.Load()
			task, err := ExecuteWorkflowFromYAMLViaBFF(ctx, client, workspaceID,
				fmt.Sprintf("knowledge-direct-answer-%s-%d", label, time.Now().UnixNano()),
				[]byte(workflowYAML),
				moi.WithTaskData(`{}`),
				moi.WithTaskVars(string(vars)),
			)
			require.NoError(t, err)
			status, err := WaitForBFFTaskCompletion(ctx, client, workspaceID, task.GetId(), 2*time.Minute)
			require.NoError(t, err)
			require.Equal(t, moi.StatusCompleted, moi.Status(status.Status), "workflow failed: %s", status.Error)
			require.Greater(t, embeddingHits.Load(), beforeEmbeddingHits, "workflow must call the configured embedding backend")

			var vectorCount int
			err = tenantDB.QueryRowContext(ctx, fmt.Sprintf("SELECT COUNT(*) FROM `%s` WHERE file_id = ?", vectorTable), uploaded.FileID).Scan(&vectorCount)
			require.NoError(t, err)
			require.Positive(t, vectorCount, "workflow must materialize the source file in its vector table")

			gotModel, err := client.SemanticModels(workspaceID).Get(ctx, model.ID)
			require.NoError(t, err)
			var gotFiles struct {
				FileIDs []string `json:"file_ids"`
			}
			require.NoError(t, json.Unmarshal(gotModel.Files, &gotFiles))
			require.Contains(t, gotFiles.FileIDs, uploaded.FileID, "lineage registration must bind the workflow source to the KB")

			if pendingSource {
				var sourceStatus string
				err = tenantDB.QueryRowContext(ctx, "SELECT status FROM knowledge_base_sources WHERE source_id = ?", sourceID).Scan(&sourceStatus)
				require.NoError(t, err)
				require.Equal(t, "pending", sourceStatus, "the test must query before governance reconciliation publishes the source")
			} else {
				var sourceCount int
				err = tenantDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM knowledge_base_sources WHERE model_id = ?", model.ID).Scan(&sourceCount)
				require.NoError(t, err)
				require.Zero(t, sourceCount, "workflow-created KB has no governance source rows")
			}

			runtime.Drain()
			requestCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()
			params, err := json.Marshal(map[string]any{
				"message": map[string]any{
					"kind":      "message",
					"role":      "user",
					"messageId": fmt.Sprintf("knowledge-direct-answer-message-%d", model.ID),
					"contextId": fmt.Sprintf("knowledge-direct-answer-context-%d", model.ID),
					"parts":     []map[string]any{{"kind": "text", "text": knowledgeDirectAnswerSentinel}},
				},
				"model":          modelName,
				"llm_backend_id": llmBackendID,
				"metadata": map[string]any{
					"workspace_id":       workspaceID,
					"semantic_model_ids": []int64{model.ID},
				},
			})
			require.NoError(t, err)
			stream, err := client.Agents().A2AStreamWithErrors(requestCtx, moi.AgentA2ARequest{
				AgentSelector: moi.AgentSelector{AgentCode: "explore", WorkspaceID: workspaceID},
				A2ARequest: moi.A2ARequest{
					JSONRPC: "2.0",
					ID:      fmt.Sprintf("knowledge-direct-answer-%d", model.ID),
					Method:  moi.A2AMethodStream,
					Params:  json.RawMessage(params),
				},
			})
			require.NoError(t, err)
			for result := range stream {
				require.NoError(t, result.Err)
				require.NoError(t, knowledgeA2AStreamEventError(result.Event.Data))
			}

			select {
			case ragResult := <-runtime.ragResults:
				require.Contains(t, ragResult, knowledgeDirectAnswerSentinel, "the real RAG tool must return the indexed document")
			case <-time.After(5 * time.Second):
				t.Fatal("timed out waiting for search_rag_chunks result")
			}
			select {
			case answer := <-runtime.answers:
				require.Equal(t, "direct answer: "+knowledgeDirectAnswerSentinel, answer)
			case <-time.After(5 * time.Second):
				t.Fatal("timed out waiting for the final answer text")
			}
		}

		t.Run("workflow_parse_complete_direct_answer", func(t *testing.T) {
			runScenario(t, "workflow", false)
		})
		t.Run("knowledge_create_or_append_pending_source_direct_answer", func(t *testing.T) {
			runScenario(t, "pending", true)
		})
	})
}
