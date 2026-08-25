package agentresource

import (
	"context"
	"fmt"
	"strings"
)

func (s *SQLAgentStore) ListAgentsReferencingKnowledgeBaseForUpdate(ctx context.Context, workspaceID, knowledgeBaseID string) ([]AgentMetadata, error) {
	if s == nil || s.tm == nil {
		return nil, fmt.Errorf("agent resource store is not configured")
	}
	if workspaceID == "" {
		return nil, fmt.Errorf("workspace_id is required")
	}
	if knowledgeBaseID == "" {
		return nil, fmt.Errorf("knowledge_base_id is required")
	}
	// Candidate filter is best-effort; final matching is structured after FOR UPDATE.
	like := "%" + escapeSQLLike(knowledgeBaseID) + "%"
	rows, err := s.tm.GetExecutor(ctx).QueryContext(ctx, fmt.Sprintf(`
SELECT id, workspace_id, name, description, avatar_ref, icon, display_tags_json, category, sort_order,
       instruction_json, runtime_json, model_json, binding_summary_json, policy_refs_json, workflow_refs_json, lifecycle_json,
       status, version, schema_version, labels_json, annotations_json, source_type, source_ref, metadata_json,
       created_by, updated_by, created_at, updated_at
FROM %s
WHERE workspace_id = ?
  AND binding_summary_json LIKE ?
ORDER BY id
FOR UPDATE`, s.table("agent_resource_agents")), workspaceID, like)
	if err != nil {
		return nil, fmt.Errorf("list agents referencing knowledge base: %w", err)
	}
	defer rows.Close()

	var out []AgentMetadata
	for rows.Next() {
		agent, err := s.scanAgent(rows)
		if err != nil {
			return nil, err
		}
		if _, changed := stripKnowledgeBaseFromBindingSummary(agent.Binding, workspaceID, knowledgeBaseID); !changed {
			continue
		}
		out = append(out, *agent)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate agents referencing knowledge base: %w", err)
	}
	return out, nil
}

func (s *SQLAgentStore) ListAgentBindingsReferencingKnowledgeBaseForUpdate(ctx context.Context, workspaceID, knowledgeBaseID string) ([]AgentBindingRecord, error) {
	if s == nil || s.tm == nil {
		return nil, fmt.Errorf("agent binding store is not configured")
	}
	if workspaceID == "" {
		return nil, fmt.Errorf("workspace_id is required")
	}
	if knowledgeBaseID == "" {
		return nil, fmt.Errorf("knowledge_base_id is required")
	}
	like := "%" + escapeSQLLike(knowledgeBaseID) + "%"
	rows, err := s.tm.GetExecutor(ctx).QueryContext(ctx, fmt.Sprintf(`
SELECT workspace_id, agent_workspace_id, agent_id, binding_summary_json,
       created_by, updated_by, created_at, updated_at
FROM %s
WHERE workspace_id = ?
  AND binding_summary_json LIKE ?
ORDER BY agent_workspace_id, agent_id
FOR UPDATE`, s.table("agent_resource_agent_bindings")), workspaceID, like)
	if err != nil {
		return nil, fmt.Errorf("list agent bindings referencing knowledge base: %w", err)
	}
	defer rows.Close()

	var out []AgentBindingRecord
	for rows.Next() {
		record, err := s.scanAgentBinding(rows)
		if err != nil {
			return nil, err
		}
		if _, changed := stripKnowledgeBaseFromBindingSummary(record.Binding, workspaceID, knowledgeBaseID); !changed {
			continue
		}
		out = append(out, *record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate agent bindings referencing knowledge base: %w", err)
	}
	return out, nil
}

func (s *SQLAgentStore) ListNonDisabledAgentVersions(ctx context.Context, workspaceID string) ([]AgentVersionRecord, error) {
	if s == nil || s.tm == nil {
		return nil, fmt.Errorf("agent version store is not configured")
	}
	if workspaceID == "" {
		return nil, fmt.Errorf("workspace_id is required")
	}
	// Non-locking candidate scan. Callers lock only matched versions by PK.
	rows, err := s.tm.GetExecutor(ctx).QueryContext(ctx, fmt.Sprintf(`
SELECT workspace_id, agent_id, version, foundation_ref, source_digest, min_moi_version, manifest_json, status, diagnostics_json,
       loaded_by, loaded_at, disabled_by, disabled_at
FROM %s
WHERE workspace_id = ?
  AND status <> ?
ORDER BY agent_id, version`, s.table("agent_resource_agent_versions")), workspaceID, AgentVersionStatusDisabled)
	if err != nil {
		return nil, fmt.Errorf("list non-disabled agent versions: %w", err)
	}
	defer rows.Close()

	var out []AgentVersionRecord
	for rows.Next() {
		record, err := s.scanAgentVersion(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate non-disabled agent versions: %w", err)
	}
	return out, nil
}

func (s *SQLAgentStore) GetAgentVersionForUpdate(ctx context.Context, workspaceID, agentID, version string) (*AgentVersionRecord, error) {
	if s == nil || s.tm == nil {
		return nil, fmt.Errorf("agent version store is not configured")
	}
	return s.scanAgentVersion(s.tm.GetExecutor(ctx).QueryRowContext(ctx, fmt.Sprintf(`
SELECT workspace_id, agent_id, version, foundation_ref, source_digest, min_moi_version, manifest_json, status, diagnostics_json,
       loaded_by, loaded_at, disabled_by, disabled_at
FROM %s
WHERE workspace_id = ? AND agent_id = ? AND version = ?
FOR UPDATE`, s.table("agent_resource_agent_versions")), workspaceID, agentID, version))
}

func escapeSQLLike(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `%`, `\%`)
	value = strings.ReplaceAll(value, `_`, `\_`)
	return value
}
