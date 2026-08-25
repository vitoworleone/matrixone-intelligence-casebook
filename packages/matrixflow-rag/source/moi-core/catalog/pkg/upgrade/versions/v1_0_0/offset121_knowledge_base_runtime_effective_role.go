package v1_0_0

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/matrixflow/moi-core/catalog/pkg/upgrade/versions"
)

// HandlerOffset121 freezes create-time principal facts used to authorize
// deferred knowledge-base RAG dispatch: verified Effective Role and
// privilege-class (IsWorkspaceOwner) snapshot.
//
// Historical rows keep empty runtime_effective_role_id and owner=0. Migration
// must not fabricate a workspace admin role or privilege-class bit: those were
// never verified for runtime_actor_moi_user_id. Jobs without a create-time
// verified principal stay fail-closed and undispatchable.
var HandlerOffset121 = &versionOffset121KnowledgeBaseRuntimeEffectiveRoleHandle{metadata: versions.Version{
	Version: "1.0.0", MinUpgradeVersion: "1.0.0", UpgradeSystem: versions.No, UpgradeTenant: versions.Yes, VersionOffset: 121,
}}

type versionOffset121KnowledgeBaseRuntimeEffectiveRoleHandle struct {
	metadata versions.Version
}

func (v *versionOffset121KnowledgeBaseRuntimeEffectiveRoleHandle) Metadata() versions.Version {
	return v.metadata
}

func (v *versionOffset121KnowledgeBaseRuntimeEffectiveRoleHandle) Prepare(context.Context, *sql.Tx, bool) error {
	return nil
}

func (v *versionOffset121KnowledgeBaseRuntimeEffectiveRoleHandle) HandleSystemUpgrade(context.Context, *sql.Tx) error {
	return nil
}

func (v *versionOffset121KnowledgeBaseRuntimeEffectiveRoleHandle) HandleTenantUpgrade(ctx context.Context, workspaceID, _ string, tenantDB *sql.DB) error {
	if err := execIdempotentAddColumn(ctx, tenantDB, "knowledge_base_source_job_runs", "runtime_effective_role_id",
		`ALTER TABLE knowledge_base_source_job_runs ADD COLUMN runtime_effective_role_id VARCHAR(64) NULL COMMENT '创建时已验证 Effective Role 快照' AFTER runtime_actor_moi_user_id`); err != nil { // i18n-allow: persistent schema metadata; never emitted as a user-facing message
		return fmt.Errorf("add knowledge_base_source_job_runs.runtime_effective_role_id: %w", err)
	}
	if err := execIdempotentAddColumn(ctx, tenantDB, "knowledge_base_source_job_runs", "runtime_is_workspace_owner",
		`ALTER TABLE knowledge_base_source_job_runs ADD COLUMN runtime_is_workspace_owner TINYINT(1) NOT NULL DEFAULT 0 COMMENT '创建时 privilege-class 快照' AFTER runtime_effective_role_id`); err != nil { // i18n-allow: persistent schema metadata; never emitted as a user-facing message
		return fmt.Errorf("add knowledge_base_source_job_runs.runtime_is_workspace_owner: %w", err)
	}
	return nil
}

func (v *versionOffset121KnowledgeBaseRuntimeEffectiveRoleHandle) VerifyTenantUpgrade(ctx context.Context, workspaceID string, tenantDB *sql.DB) error {
	for _, col := range []string{"runtime_effective_role_id", "runtime_is_workspace_owner"} {
		ok, err := columnExists(ctx, tenantDB, "knowledge_base_source_job_runs", col)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("knowledge_base_source_job_runs.%s missing", col)
		}
	}
	return nil
}

func (v *versionOffset121KnowledgeBaseRuntimeEffectiveRoleHandle) HandleCreateFrameworkDeps(*sql.Tx) error {
	return nil
}
