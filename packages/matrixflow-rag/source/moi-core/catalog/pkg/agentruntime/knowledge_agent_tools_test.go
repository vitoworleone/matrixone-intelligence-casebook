package agentruntime

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/matrixflow/moi-core/catalog/pkg/agentruntime/a2a"
)

func TestKnowledgeAgentToolSchemaUsesInstructionEnvelopeOnly(t *testing.T) {
	var schema struct {
		AdditionalProperties bool     `json:"additionalProperties"`
		Required             []string `json:"required"`
		Properties           map[string]struct {
			Description string `json:"description"`
		} `json:"properties"`
	}
	if err := json.Unmarshal([]byte(knowledgeAgentToolInputSchema), &schema); err != nil {
		t.Fatalf("unmarshal knowledge agent tool schema: %v", err)
	}
	instruction, ok := schema.Properties["instruction"]
	if schema.AdditionalProperties || !reflect.DeepEqual(schema.Required, []string{"instruction"}) || len(schema.Properties) != 1 || !ok {
		t.Fatalf("knowledge agent schema = %#v, want an instruction-only envelope", schema)
	}
	description := instruction.Description
	if !strings.Contains(description, "only argument") ||
		!strings.Contains(description, "Never send action, table, records, key, or values") {
		t.Fatalf("instruction schema description = %q", description)
	}
}

type knowledgeAgentRuntimeStub struct {
	params a2a.MessageSendParams
	scope  RuntimeRequestScope
	events []a2a.StreamEvent
	rpcErr *a2a.RPCError
}

func (s *knowledgeAgentRuntimeStub) MessageStream(ctx context.Context, raw json.RawMessage, _ *json.RawMessage) (<-chan a2a.StreamEvent, *a2a.RPCError) {
	if err := json.Unmarshal(raw, &s.params); err != nil {
		return nil, a2a.NewError(a2a.CodeInvalidParams, err.Error(), nil)
	}
	s.scope, _ = RuntimeRequestScopeFromContext(ctx)
	events := make(chan a2a.StreamEvent, len(s.events))
	for _, event := range s.events {
		events <- event
	}
	close(events)
	return events, s.rpcErr
}

func TestKnowledgeAgentInvokerDelegatesOnlyBoundKnowledge(t *testing.T) {
	runtime := &knowledgeAgentRuntimeStub{events: []a2a.StreamEvent{
		{Response: a2a.JSONRPCResponse{Result: a2a.TaskStatusUpdateEvent{
			Final: false,
			Status: a2a.TaskStatus{
				State:   a2a.TaskState(a2a.TaskStateWorking),
				Message: &a2a.Message{Role: a2a.RoleAgent, Parts: []a2a.Part{{Kind: "text", Text: "  identity is confirmed  "}}},
			},
		}}},
		{Response: a2a.JSONRPCResponse{Result: a2a.TaskStatusUpdateEvent{
			Final:  true,
			Status: a2a.TaskStatus{State: a2a.TaskState(a2a.TaskStateCompleted)},
		}}},
	}}
	invoker := NewKnowledgeAgentInvoker("system_agents", "knowledge_explore")
	invoker.Bind(runtime)
	ctx := withTestRuntimeRequestScope(context.Background(), RuntimeRequestScope{
		WorkspaceID:      "workspace_1",
		AgentWorkspaceID: "caller_agents",
		AgentID:          "issue_operator",
		UserID:           "user_1",
		UserAPIKey:       "api_key_1",
	})
	result, err := invoker.Invoke(ctx, ToolInvokeRequest{
		WorkspaceID: "workspace_1",
		TaskID:      "task_1",
		CallID:      "call_1",
		Manifest: RuntimeManifest{AgentID: "github-issue-operator", Body: map[string]any{
			"model": "gpt-5",
			"knowledge_bases": []map[string]any{
				{"id": "17", "catalog_asset_refs": []map[string]any{}},
				{"id": "17", "catalog_asset_refs": []map[string]any{}},
			},
		}},
		TurnMetadata: map[string]any{
			"llm_backend_id":       int64(23),
			"llm_reasoning_effort": "high",
		},
	}, "Find the owner for octocat")
	if err != nil {
		t.Fatalf("Invoke() error = %v", err)
	}
	if got, want := result["answer"], "identity is confirmed"; got != want {
		t.Fatalf("answer = %#v, want %#v", got, want)
	}
	if got, want := result["semantic_model_ids"], []int64{17}; !reflect.DeepEqual(got, want) {
		t.Fatalf("semantic model ids = %#v, want %#v", got, want)
	}
	if runtime.params.AgentID != "knowledge_explore" || runtime.params.Model != "gpt-5" || runtime.params.LLMBackendID != 23 {
		t.Fatalf("delegate params = %#v", runtime.params)
	}
	if runtime.params.IdempotencyKey != "knowledge-agent:task_1:call_1" {
		t.Fatalf("idempotency key = %q", runtime.params.IdempotencyKey)
	}
	if runtime.params.Message.MessageID != "msg-call_1" {
		t.Fatalf("delegate message id = %q", runtime.params.Message.MessageID)
	}
	if got := runtime.params.Message.Parts; len(got) != 1 || got[0].Kind != "text" || got[0].Text != "Find the owner for octocat" {
		t.Fatalf("delegate message = %#v", runtime.params.Message)
	}
	if got, want := runtime.params.Metadata["semantic_model_ids"], []any{float64(17)}; !reflect.DeepEqual(got, want) {
		t.Fatalf("delegate semantic model ids = %#v, want %#v", got, want)
	}
	if runtime.params.Metadata["llm_reasoning_effort"] != "high" {
		t.Fatalf("delegate metadata = %#v", runtime.params.Metadata)
	}
	if got, want := runtime.scope, (RuntimeRequestScope{
		WorkspaceID:      "workspace_1",
		AgentWorkspaceID: "system_agents",
		AgentID:          "knowledge_explore",
		UserID:           "user_1",
		UserAPIKey:       "api_key_1",
		EffectiveRoleID:  "role_1",
	}); !reflect.DeepEqual(got, want) {
		t.Fatalf("delegate scope = %#v, want %#v", got, want)
	}
}

func TestKnowledgeAgentInvokerDelegatePassesRuntimeMessageValidation(t *testing.T) {
	invoker := NewKnowledgeAgentInvoker("system_agents", "knowledge_explore")
	invoker.Bind(newTestRuntimeService())
	ctx := withTestRuntimeRequestScope(context.Background(), RuntimeRequestScope{
		WorkspaceID:      "workspace_1",
		AgentWorkspaceID: "caller_agents",
		AgentID:          "issue_operator",
		UserID:           "user_1",
	})

	_, err := invoker.Invoke(ctx, ToolInvokeRequest{
		WorkspaceID: "workspace_1",
		TaskID:      "task_1",
		CallID:      "call_1",
		Manifest: RuntimeManifest{Body: map[string]any{
			"model": "gpt-5",
			"knowledge_bases": []map[string]any{
				{"id": "17", "catalog_asset_refs": []map[string]any{}},
			},
		}},
	}, "Find the owner for octocat")
	if err == nil || !strings.Contains(err.Error(), "runtime provider is unavailable") {
		t.Fatalf("Invoke() error = %v, want provider failure after runtime message validation", err)
	}
}

func TestKnowledgeAgentInvokerRejectsCallerWithoutBoundKnowledge(t *testing.T) {
	invoker := NewKnowledgeAgentInvoker("system_agents", "knowledge_explore")
	invoker.Bind(&knowledgeAgentRuntimeStub{})
	_, err := invoker.Invoke(
		withTestRuntimeRequestScope(context.Background(), RuntimeRequestScope{UserID: "user_1"}),
		ToolInvokeRequest{
			WorkspaceID: "workspace_1",
			Manifest:    RuntimeManifest{Body: map[string]any{"model": "gpt-5", "knowledge_bases": []map[string]any{}}},
		},
		"Find the owner",
	)
	if err == nil || !strings.Contains(err.Error(), "has no bound semantic knowledge base") {
		t.Fatalf("Invoke() error = %v, want missing knowledge rejection", err)
	}
}
