package v1_0_0

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/matrixflow/moi-core/catalog/pkg/upgrade/versions"
)

// HandlerOffset102 records the MOI user associated with a knowledge-base
// source job for dispatch and retry auditing.
var HandlerOffset102 = &versionOffset102KnowledgeBaseRuntimeActorHandle{metadata: versions.Version{
	Version: "1.0.0", MinUpgradeVersion: "1.0.0", UpgradeSystem: versions.No, UpgradeTenant: versions.Yes, VersionOffset: 102,
}}

type versionOffset102KnowledgeBaseRuntimeActorHandle struct {
	metadata versions.Version
}

func (v *versionOffset102KnowledgeBaseRuntimeActorHandle) Metadata() versions.Version {
	return v.metadata
}

func (v *versionOffset102KnowledgeBaseRuntimeActorHandle) Prepare(context.Context, *sql.Tx, bool) error {
	return nil
}

func (v *versionOffset102KnowledgeBaseRuntimeActorHandle) HandleSystemUpgrade(context.Context, *sql.Tx) error {
	return nil
}

func (v *versionOffset102KnowledgeBaseRuntimeActorHandle) HandleTenantUpgrade(ctx context.Context, _ string, _ string, tenantDB *sql.DB) error {
	if err := execIdempotentAddColumn(ctx, tenantDB, "knowledge_base_source_job_runs", "runtime_actor_moi_user_id",
		`ALTER TABLE knowledge_base_source_job_runs ADD COLUMN runtime_actor_moi_user_id VARCHAR(64) NULL COMMENT '执行主体 MOI 用户ID' AFTER workflow_execution_id`); err != nil { // i18n-allow: persistent schema metadata; never emitted as a user-facing message
		return fmt.Errorf("add knowledge_base_source_job_runs.runtime_actor_moi_user_id: %w", err)
	}
	return nil
}

func (v *versionOffset102KnowledgeBaseRuntimeActorHandle) VerifyTenantUpgrade(ctx context.Context, _ string, tenantDB *sql.DB) error {
	ok, err := columnExists(ctx, tenantDB, "knowledge_base_source_job_runs", "runtime_actor_moi_user_id")
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("knowledge_base_source_job_runs.runtime_actor_moi_user_id missing")
	}
	return nil
}

func (v *versionOffset102KnowledgeBaseRuntimeActorHandle) HandleCreateFrameworkDeps(*sql.Tx) error {
	return nil
}
