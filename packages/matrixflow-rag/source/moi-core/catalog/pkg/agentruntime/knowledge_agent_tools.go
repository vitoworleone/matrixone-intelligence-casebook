package agentruntime

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	agentruntimev2 "github.com/matrixflow/moi-core/agent-runtime-v2"
	"github.com/matrixflow/moi-core/catalog/pkg/agentruntime/a2a"
)

const RuntimeToolNameKnowledgeAgent = "moi_knowledge_agent"

const knowledgeAgentToolInputSchema = `{"type":"object","additionalProperties":false,"required":["instruction"],"properties":{"instruction":{"type":"string","minLength":1,"description":"The only argument for this delegated knowledge lookup request. Put the question and any exact lookup constraints in this text. Never send action, table, records, key, or values as sibling JSON properties."}}}`

type KnowledgeAgentRuntime interface {
	MessageStream(context.Context, json.RawMessage, *json.RawMessage) (<-chan a2a.StreamEvent, *a2a.RPCError)
}

// KnowledgeAgentInvoker keeps the system target separate from the caller's
// tenant workspace. The runtime is bound after service construction because
// the tool gateway itself is an input to that construction.
type KnowledgeAgentInvoker struct {
	agentWorkspaceID string
	agentID          string

	mu      sync.RWMutex
	runtime KnowledgeAgentRuntime
}

func NewKnowledgeAgentInvoker(agentWorkspaceID string, agentID string) *KnowledgeAgentInvoker {
	return &KnowledgeAgentInvoker{
		agentWorkspaceID: agentWorkspaceID,
		agentID:          agentID,
	}
}

func (i *KnowledgeAgentInvoker) Bind(runtime KnowledgeAgentRuntime) {
	if i == nil {
		return
	}
	i.mu.Lock()
	defer i.mu.Unlock()
	i.runtime = runtime
}

func (i *KnowledgeAgentInvoker) currentRuntime() KnowledgeAgentRuntime {
	if i == nil {
		return nil
	}
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.runtime
}

func RuntimeKnowledgeAgentToolExecutors(invoker *KnowledgeAgentInvoker) map[string]PlatformToolExecutorFactory {
	if invoker == nil {
		return nil
	}
	return map[string]PlatformToolExecutorFactory{
		RuntimeToolNameKnowledgeAgent: func(_ context.Context, req ToolInvokeRequest, _ RuntimeToolSnapshotForExecutor) (agentruntimev2.Tool, error) {
			return NewKnowledgeAgentTool(invoker, req), nil
		},
	}
}

func NewKnowledgeAgentTool(invoker *KnowledgeAgentInvoker, req ToolInvokeRequest) agentruntimev2.Tool {
	return agentruntimev2.NewTool(
		RuntimeToolNameKnowledgeAgent,
		"Delegate a natural-language knowledge-base lookup request to the configured Knowledge Explore system agent. This tool accepts only the instruction field. The delegate can access only knowledge bases bound to this caller and cannot modify them.",
		json.RawMessage(knowledgeAgentToolInputSchema),
		func(ctx context.Context, raw json.RawMessage) (*agentruntimev2.ToolResult, error) {
			var args struct {
				Instruction string `json:"instruction"`
			}
			if err := agentruntimev2.StrictDecode(raw, &args); err != nil {
				return nil, agentruntimev2.RespondToModelError(fmt.Sprintf("%s: invalid arguments: %v", RuntimeToolNameKnowledgeAgent, err))
			}
			if args.Instruction == "" || args.Instruction != strings.TrimSpace(args.Instruction) {
				return nil, agentruntimev2.RespondToModelError(fmt.Sprintf("%s: instruction must be non-empty and must not contain leading or trailing whitespace", RuntimeToolNameKnowledgeAgent))
			}
			result, err := invoker.Invoke(ctx, req, args.Instruction)
			if err != nil {
				return nil, agentruntimev2.RespondToModelError(err.Error())
			}
			content, err := json.Marshal(result)
			if err != nil {
				return nil, agentruntimev2.FatalFunctionCallError(fmt.Sprintf("%s: marshal result: %v", RuntimeToolNameKnowledgeAgent, err))
			}
			return &agentruntimev2.ToolResult{Content: string(content), Data: result}, nil
		},
	)
}

func (i *KnowledgeAgentInvoker) Invoke(ctx context.Context, req ToolInvokeRequest, instruction string) (map[string]any, error) {
	if i == nil || strings.TrimSpace(i.agentWorkspaceID) == "" || strings.TrimSpace(i.agentID) == "" {
		return nil, fmt.Errorf("%s: knowledge agent target is not configured", RuntimeToolNameKnowledgeAgent)
	}
	runtime := i.currentRuntime()
	if runtime == nil {
		return nil, fmt.Errorf("%s: knowledge agent runtime is not configured", RuntimeToolNameKnowledgeAgent)
	}
	if req.WorkspaceID == "" || req.WorkspaceID != strings.TrimSpace(req.WorkspaceID) {
		return nil, fmt.Errorf("%s: caller workspace id is required", RuntimeToolNameKnowledgeAgent)
	}
	if req.Manifest.Body == nil {
		return nil, fmt.Errorf("%s: caller manifest is required", RuntimeToolNameKnowledgeAgent)
	}
	knowledge, err := runtimeKnowledgeBasesFromValue(req.Manifest.Body["knowledge_bases"], true)
	if err != nil {
		return nil, fmt.Errorf("%s: read bound knowledge bases: %w", RuntimeToolNameKnowledgeAgent, err)
	}
	semanticModelIDs, err := knowledgeAgentSemanticModelIDs(knowledge)
	if err != nil {
		return nil, err
	}
	if len(semanticModelIDs) == 0 {
		return nil, fmt.Errorf("%s: caller has no bound semantic knowledge base", RuntimeToolNameKnowledgeAgent)
	}
	model, err := runtimeV2ModelFromManifest(req.Manifest)
	if err != nil {
		return nil, fmt.Errorf("%s: read caller model: %w", RuntimeToolNameKnowledgeAgent, err)
	}
	if model == "" {
		return nil, fmt.Errorf("%s: caller model is required", RuntimeToolNameKnowledgeAgent)
	}
	requestScope, ok := RuntimeRequestScopeFromContext(ctx)
	if !ok || requestScope.UserID == "" {
		return nil, fmt.Errorf("%s: caller identity is required", RuntimeToolNameKnowledgeAgent)
	}
	metadata := map[string]any{"semantic_model_ids": semanticModelIDs}
	if effort, ok := req.TurnMetadata["llm_reasoning_effort"].(string); ok && effort != "" {
		metadata["llm_reasoning_effort"] = effort
	}
	backendID, _ := int64FromAny(req.TurnMetadata["llm_backend_id"])
	child := a2a.MessageSendParams{
		AgentID:        i.agentID,
		IdempotencyKey: "knowledge-agent:" + req.TaskID + ":" + req.CallID,
		Model:          model,
		LLMBackendID:   backendID,
		Metadata:       metadata,
		Message: a2a.Message{
			MessageID: "msg-" + req.CallID,
			Kind:      "message",
			Role:      a2a.RoleUser,
			Parts: []a2a.Part{{
				Kind: "text",
				Text: instruction,
			}},
		},
	}
	params, err := json.Marshal(child)
	if err != nil {
		return nil, fmt.Errorf("%s: marshal delegate request: %w", RuntimeToolNameKnowledgeAgent, err)
	}
	childScope := RuntimeRequestScope{
		WorkspaceID:      req.WorkspaceID,
		AgentWorkspaceID: i.agentWorkspaceID,
		AgentID:          i.agentID,
		UserID:           requestScope.UserID,
		UserAPIKey:       requestScope.UserAPIKey,
		EffectiveRoleID:  requestScope.EffectiveRoleID,
		RoleCandidateID:  requestScope.RoleCandidateID,
	}
	requestID := json.RawMessage(`"knowledge-agent"`)
	events, rpcErr := runtime.MessageStream(WithRuntimeRequestScope(ctx, childScope), params, &requestID)
	if rpcErr != nil {
		return nil, fmt.Errorf("%s: start delegate: %s", RuntimeToolNameKnowledgeAgent, rpcErr.Message)
	}
	var streamedAnswer strings.Builder
	for event := range events {
		if event.Response.Error != nil {
			return nil, fmt.Errorf("%s: delegate failed: %s", RuntimeToolNameKnowledgeAgent, event.Response.Error.Message)
		}
		update, ok := event.Response.Result.(a2a.TaskStatusUpdateEvent)
		if !ok {
			continue
		}
		messageText := knowledgeAgentMessageText(update.Status.Message)
		if !update.Final {
			streamedAnswer.WriteString(messageText)
			continue
		}
		if string(update.Status.State) != a2a.TaskStateCompleted {
			return nil, fmt.Errorf("%s: delegate ended in %s", RuntimeToolNameKnowledgeAgent, update.Status.State)
		}
		answer := messageText
		if answer == "" {
			answer = strings.TrimSpace(streamedAnswer.String())
		}
		if answer == "" {
			return nil, fmt.Errorf("%s: delegate completed without an answer", RuntimeToolNameKnowledgeAgent)
		}
		return map[string]any{
			"answer":             answer,
			"semantic_model_ids": semanticModelIDs,
		}, nil
	}
	return nil, fmt.Errorf("%s: delegate stream ended without a final result", RuntimeToolNameKnowledgeAgent)
}

func knowledgeAgentSemanticModelIDs(knowledge []runtimeKnowledgeSnapshot) ([]int64, error) {
	ids := make([]int64, 0, len(knowledge))
	seen := map[int64]struct{}{}
	for _, snapshot := range knowledge {
		id, ok := int64FromAny(snapshot.ID)
		if !ok || id <= 0 {
			return nil, fmt.Errorf("%s: bound knowledge base id must be a positive integer", RuntimeToolNameKnowledgeAgent)
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	return ids, nil
}

func knowledgeAgentMessageText(message *a2a.Message) string {
	if message == nil {
		return ""
	}
	var builder strings.Builder
	for _, part := range message.Parts {
		if part.Kind == "text" {
			builder.WriteString(part.Text)
		}
	}
	return strings.TrimSpace(builder.String())
}
