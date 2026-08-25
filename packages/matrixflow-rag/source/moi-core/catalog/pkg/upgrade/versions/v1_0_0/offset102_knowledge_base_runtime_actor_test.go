package v1_0_0

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/matrixflow/moi-core/catalog/pkg/upgrade/versions"
	"github.com/stretchr/testify/require"
)

func TestOffset102Metadata(t *testing.T) {
	metadata := HandlerOffset102.Metadata()
	require.Equal(t, uint32(102), metadata.VersionOffset)
	require.Equal(t, versions.No, metadata.UpgradeSystem)
	require.Equal(t, versions.Yes, metadata.UpgradeTenant)
}

func TestOffset102AddsKnowledgeBaseRuntimeActorColumn(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectExec(regexp.QuoteMeta(`ALTER TABLE knowledge_base_source_job_runs ADD COLUMN runtime_actor_moi_user_id VARCHAR(64) NULL COMMENT '执行主体 MOI 用户ID' AFTER workflow_execution_id`)).
		WillReturnResult(sqlmock.NewResult(0, 0))

	require.NoError(t, HandlerOffset102.HandleTenantUpgrade(context.Background(), "ws-1", "tenant-1", db))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestOffset102VerifiesKnowledgeBaseRuntimeActorColumn(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	expectColumnExists(mock, "knowledge_base_source_job_runs", "runtime_actor_moi_user_id", true)

	require.NoError(t, HandlerOffset102.VerifyTenantUpgrade(context.Background(), "ws-1", db))
	require.NoError(t, mock.ExpectationsWereMet())
}
