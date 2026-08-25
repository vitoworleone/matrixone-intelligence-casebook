package v1_0_0

import (
	"context"
	"crypto/sha1"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	userservice "github.com/matrixflow/moi-core/catalog/pkg/service/user"
	"github.com/matrixflow/moi-core/catalog/pkg/upgrade/versions"
)

const (
	offset103KnowledgeBaseWorkflowIDNamespace   = "kb-rag-workflow"
	offset103KnowledgeBaseWorkflowIDPrefix      = offset103KnowledgeBaseWorkflowIDNamespace + "-"
	offset103KnowledgeBaseWorkflowSourceType    = "manual_dsl"
	offset103KnowledgeBaseWorkflowExecutionMode = "one_shot"
	offset103KnowledgeBaseAudioWorkflowTemplate = "audio_kb_ingest"
	offset103KnowledgeBaseVideoWorkflowTemplate = "video_kb_ingest"
)

// HandlerOffset103 removes only legacy system-owned knowledge-base workflow
// bindings that block a current user from redeploying the same stable ID.
var HandlerOffset103 = &versionOffset103KnowledgeBaseSystemWorkflowCleanupHandle{metadata: versions.Version{
	Version: "1.0.0", MinUpgradeVersion: "1.0.0", UpgradeSystem: versions.Yes, UpgradeTenant: versions.No, VersionOffset: 103,
}}

type versionOffset103KnowledgeBaseSystemWorkflowCleanupHandle struct {
	metadata versions.Version
}

func (v *versionOffset103KnowledgeBaseSystemWorkflowCleanupHandle) Metadata() versions.Version {
	return v.metadata
}

func (v *versionOffset103KnowledgeBaseSystemWorkflowCleanupHandle) Prepare(context.Context, *sql.Tx, bool) error {
	return nil
}

func (v *versionOffset103KnowledgeBaseSystemWorkflowCleanupHandle) HandleSystemUpgrade(ctx context.Context, tx *sql.Tx) error {
	if tx == nil {
		return fmt.Errorf("system database transaction is nil")
	}

	var systemUserID string
	if err := tx.QueryRowContext(ctx, `
SELECT id
FROM users
WHERE email = ?`, userservice.SystemUserEmail).Scan(&systemUserID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("resolve system user for legacy knowledge-base workflow cleanup: %w", err)
	}

	// The prefix only narrows the candidate read. A row is deleted only after its
	// semantic_model_id recreates the exact stable ID used by knowledge bases.
	rows, err := tx.QueryContext(ctx, `
SELECT id, workspace_id, default_values_json
FROM workflow_app
WHERE user_id = ?
  AND source_type = ?
  AND execution_mode = ?
  AND id LIKE ?`,
		systemUserID,
		offset103KnowledgeBaseWorkflowSourceType,
		offset103KnowledgeBaseWorkflowExecutionMode,
		offset103KnowledgeBaseWorkflowIDPrefix+"%")
	if err != nil {
		return fmt.Errorf("list legacy system-owned knowledge-base workflow candidates: %w", err)
	}
	defer rows.Close()

	workflowIDsByWorkspace := make(map[string]map[string]struct{})
	for rows.Next() {
		var workflowID, workspaceID string
		var defaultValues sql.NullString
		if err := rows.Scan(&workflowID, &workspaceID, &defaultValues); err != nil {
			return fmt.Errorf("scan legacy system-owned knowledge-base workflow candidate: %w", err)
		}
		modelID, ok := offset103SemanticModelID(defaultValues.String)
		if !ok || !offset103IsKnowledgeBaseWorkflowID(workflowID, workspaceID, modelID) {
			continue
		}
		if workflowIDsByWorkspace[workspaceID] == nil {
			workflowIDsByWorkspace[workspaceID] = make(map[string]struct{})
		}
		workflowIDsByWorkspace[workspaceID][workflowID] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("read legacy system-owned knowledge-base workflow candidates: %w", err)
	}

	workspaceIDs := make([]string, 0, len(workflowIDsByWorkspace))
	for workspaceID := range workflowIDsByWorkspace {
		workspaceIDs = append(workspaceIDs, workspaceID)
	}
	sort.Strings(workspaceIDs)
	for _, workspaceID := range workspaceIDs {
		workflowIDs := make([]string, 0, len(workflowIDsByWorkspace[workspaceID]))
		for workflowID := range workflowIDsByWorkspace[workspaceID] {
			workflowIDs = append(workflowIDs, workflowID)
		}
		sort.Strings(workflowIDs)
		args := make([]any, 0, len(workflowIDs)+4)
		args = append(args, workspaceID, systemUserID, offset103KnowledgeBaseWorkflowSourceType, offset103KnowledgeBaseWorkflowExecutionMode)
		for _, workflowID := range workflowIDs {
			args = append(args, workflowID)
		}
		if _, err := tx.ExecContext(ctx, offset103DeleteKnowledgeBaseWorkflowAppsQuery(len(workflowIDs)), args...); err != nil {
			return fmt.Errorf("delete legacy system-owned knowledge-base workflow apps: %w", err)
		}
	}
	return nil
}

func offset103SemanticModelID(defaultValuesJSON string) (int64, bool) {
	var values struct {
		SemanticModelID int64 `json:"semantic_model_id"`
	}
	if err := json.Unmarshal([]byte(defaultValuesJSON), &values); err != nil || values.SemanticModelID <= 0 {
		return 0, false
	}
	return values.SemanticModelID, true
}

func offset103IsKnowledgeBaseWorkflowID(workflowID, workspaceID string, modelID int64) bool {
	for _, expectedID := range offset103KnowledgeBaseWorkflowIDs(workspaceID, modelID) {
		if workflowID == expectedID {
			return true
		}
	}
	return false
}

func offset103KnowledgeBaseWorkflowIDs(workspaceID string, modelID int64) []string {
	workspaceID = strings.TrimSpace(workspaceID)
	return []string{
		offset103StableID(offset103KnowledgeBaseWorkflowIDNamespace, workspaceID, modelID),
		offset103StableID(offset103KnowledgeBaseWorkflowIDNamespace, workspaceID, modelID, offset103KnowledgeBaseAudioWorkflowTemplate),
		offset103StableID(offset103KnowledgeBaseWorkflowIDNamespace, workspaceID, modelID, offset103KnowledgeBaseVideoWorkflowTemplate),
	}
}

func offset103StableID(prefix string, parts ...any) string {
	hash := sha1.New()
	for _, part := range parts {
		_, _ = hash.Write([]byte(fmt.Sprint(part)))
		_, _ = hash.Write([]byte{0})
	}
	return prefix + "-" + hex.EncodeToString(hash.Sum(nil))[:24]
}

func offset103DeleteKnowledgeBaseWorkflowAppsQuery(workflowCount int) string {
	placeholders := make([]string, workflowCount)
	for i := range placeholders {
		placeholders[i] = "?"
	}
	return fmt.Sprintf(`
DELETE FROM workflow_app
WHERE workspace_id = ?
  AND user_id = ?
  AND source_type = ?
  AND execution_mode = ?
  AND id IN (%s)`, strings.Join(placeholders, ", "))
}

func (v *versionOffset103KnowledgeBaseSystemWorkflowCleanupHandle) HandleTenantUpgrade(context.Context, string, string, *sql.DB) error {
	return nil
}

func (v *versionOffset103KnowledgeBaseSystemWorkflowCleanupHandle) VerifyTenantUpgrade(context.Context, string, *sql.DB) error {
	return nil
}

func (v *versionOffset103KnowledgeBaseSystemWorkflowCleanupHandle) HandleCreateFrameworkDeps(*sql.Tx) error {
	return nil
}
