package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	agentruntimev2 "github.com/matrixflow/moi-core/agent-runtime-v2"
	moi "github.com/matrixflow/moi-core/go-sdk"
	"github.com/stretchr/testify/require"
)

// knowledgeRuntimeToolCapture records tool names registered for a knowledge A2A
// (agent_code=explore) run so issue regressions can assert scope resolution.
type knowledgeRuntimeToolCapture struct {
	tools chan knowledgeRuntimeToolCaptureEntry
}

type knowledgeRuntimeToolCaptureEntry struct {
	model string
	tools map[string]bool
}

func newKnowledgeRuntimeToolCapture() *knowledgeRuntimeToolCapture {
	return &knowledgeRuntimeToolCapture{tools: make(chan knowledgeRuntimeToolCaptureEntry, 8)}
}

func (c *knowledgeRuntimeToolCapture) StreamResponses(_ context.Context, req agentruntimev2.ResponsesAPIRequest, onEvent func(agentruntimev2.ResponseEvent) error) error {
	names, err := testResponsesToolNameSetFromRequest(req)
	if err != nil {
		return err
	}
	if len(names) > 0 {
		boolNames := make(map[string]bool, len(names))
		for name := range names {
			boolNames[name] = true
		}
		c.tools <- knowledgeRuntimeToolCaptureEntry{model: strings.TrimSpace(req.Model), tools: boolNames}
		if boolNames["submit_final_answer"] {
			return emitTestResponsesToolCall(onEvent, agentruntimev2.ToolCall{
				ID:        "call_submit_final_answer",
				Name:      "submit_final_answer",
				Arguments: json.RawMessage(`{"answer":"ok","sources":[]}`),
				Kind:      agentruntimev2.ToolCallKindFunction,
			})
		}
	}
	return emitTestResponsesText(onEvent, "ok")
}

func (c *knowledgeRuntimeToolCapture) CompactResponses(ctx context.Context, input agentruntimev2.CompactionInput, clientMetadata map[string]string) ([]agentruntimev2.ResponseItem, error) {
	_ = ctx
	_ = input
	_ = clientMetadata
	return []agentruntimev2.ResponseItem{}, nil
}

func (c *knowledgeRuntimeToolCapture) WaitTools(ctx context.Context, modelName string, streamErrs <-chan error) (map[string]bool, error) {
	modelName = strings.TrimSpace(modelName)
	select {
	case entry := <-c.tools:
		if entry.model == modelName {
			return entry.tools, nil
		}
	default:
	}
	for {
		select {
		case entry := <-c.tools:
			if entry.model == modelName {
				return entry.tools, nil
			}
		case err := <-streamErrs:
			return nil, err
		case <-ctx.Done():
			return nil, fmt.Errorf("timed out waiting for knowledge A2A runtime tool registration for model %s: %w", modelName, ctx.Err())
		}
	}
}

func (c *knowledgeRuntimeToolCapture) Drain() {
	for {
		select {
		case <-c.tools:
		default:
			return
		}
	}
}

type knowledgeMockLLMServer struct {
	server *httptest.Server
	once   sync.Once
}

func newKnowledgeMockLLMServer(t *testing.T) *knowledgeMockLLMServer {
	t.Helper()
	capture := &knowledgeMockLLMServer{}
	capture.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			http.NotFound(w, r)
			return
		}
		defer r.Body.Close()
		var req mockAgentLoopRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		writeMockFinalTextMaybeStream(t, w, req.Stream, "ok")
	}))
	return capture
}

func (s *knowledgeMockLLMServer) URL() string {
	return s.server.URL
}

func (s *knowledgeMockLLMServer) Close() {
	s.once.Do(s.server.Close)
}

func knowledgeA2AStreamEventError(raw json.RawMessage) error {
	var envelope struct {
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
		Result struct {
			Kind   string `json:"kind"`
			Status struct {
				State   string `json:"state"`
				Message any    `json:"message"`
			} `json:"status"`
		} `json:"result"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return fmt.Errorf("decode knowledge A2A stream event: %w: %s", err, string(raw))
	}
	if envelope.Error != nil {
		return fmt.Errorf("knowledge A2A stream JSON-RPC error code=%d message=%s", envelope.Error.Code, envelope.Error.Message)
	}
	if envelope.Result.Status.State == "failed" || envelope.Result.Status.State == "canceled" || envelope.Result.Status.State == "rejected" {
		statusMessage, _ := json.Marshal(envelope.Result.Status.Message)
		return fmt.Errorf("knowledge A2A stream ended with state=%s kind=%s message=%s event=%s", envelope.Result.Status.State, envelope.Result.Kind, string(statusMessage), string(raw))
	}
	return nil
}

// captureKnowledgeA2AToolsForSemanticModel drives agent_code=explore A2A stream
// against a semantic model and returns the tool names the runtime registered.
func captureKnowledgeA2AToolsForSemanticModel(t *testing.T, ctx context.Context, client *moi.Client, workspaceID string, semanticModelID int64, label string, runtimeToolCapture *knowledgeRuntimeToolCapture) map[string]bool {
	t.Helper()
	runtimeToolCapture.Drain()
	modelName := fmt.Sprintf("mock-llm-semantic-lineage-tools-%s-%d", label, time.Now().UnixNano())
	llm := newKnowledgeMockLLMServer(t)
	defer llm.Close()
	llmBackendID, cleanupBackend := ensureMockLLMBackend(t, client, workspaceID, fmt.Sprintf("semantic-lineage-tools-%s", label), modelName, llm.URL())
	defer cleanupBackend()

	reqCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	params, err := json.Marshal(map[string]any{
		"message": map[string]any{
			"kind":      "message",
			"role":      "user",
			"messageId": fmt.Sprintf("semantic-lineage-tools-message-%d", semanticModelID),
			"contextId": fmt.Sprintf("semantic-lineage-tools-context-%d", semanticModelID),
			"parts": []map[string]any{{
				"kind": "text",
				"text": "兼容性检查",
			}},
		},
		"model":          modelName,
		"llm_backend_id": llmBackendID,
		"metadata": map[string]any{
			"workspace_id":       workspaceID,
			"semantic_model_ids": []int64{semanticModelID},
		},
	})
	require.NoError(t, err)

	stream, err := client.Agents().A2AStream(reqCtx, moi.AgentA2ARequest{
		AgentSelector: moi.AgentSelector{AgentCode: "explore", WorkspaceID: workspaceID},
		A2ARequest: moi.A2ARequest{
			JSONRPC: "2.0",
			ID:      fmt.Sprintf("semantic-lineage-tools-%d", semanticModelID),
			Method:  moi.A2AMethodStream,
			Params:  json.RawMessage(params),
		},
	})
	require.NoError(t, err)
	drained := make(chan struct{})
	streamErrs := make(chan error, 1)
	go func() {
		defer close(drained)
		for event := range stream {
			if err := knowledgeA2AStreamEventError(event.Data); err != nil {
				select {
				case streamErrs <- err:
				default:
				}
			}
		}
	}()

	tools, err := runtimeToolCapture.WaitTools(reqCtx, modelName, streamErrs)
	cancel()
	<-drained
	require.NoError(t, err)
	return tools
}
