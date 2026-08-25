package v1_0_0

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/matrixflow/moi-core/catalog/pkg/upgrade/versions"
	"github.com/stretchr/testify/require"
)

func TestOffset121Metadata(t *testing.T) {
	metadata := HandlerOffset121.Metadata()
	require.Equal(t, uint32(121), metadata.VersionOffset)
	require.Equal(t, versions.No, metadata.UpgradeSystem)
	require.Equal(t, versions.Yes, metadata.UpgradeTenant)
}

func TestOffset121AddsColumnsWithoutFabricatingAdminRole(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	// Schema only: no roles lookup, no UPDATE backfill of system admin / privilege bit.
	mock.ExpectExec(regexp.QuoteMeta(`ALTER TABLE knowledge_base_source_job_runs ADD COLUMN runtime_effective_role_id VARCHAR(64) NULL COMMENT '创建时已验证 Effective Role 快照' AFTER runtime_actor_moi_user_id`)).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(regexp.QuoteMeta(`ALTER TABLE knowledge_base_source_job_runs ADD COLUMN runtime_is_workspace_owner TINYINT(1) NOT NULL DEFAULT 0 COMMENT '创建时 privilege-class 快照' AFTER runtime_effective_role_id`)).
		WillReturnResult(sqlmock.NewResult(0, 0))

	require.NoError(t, HandlerOffset121.HandleTenantUpgrade(context.Background(), "ws-1", "tenant-1", db))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestOffset121VerifiesKnowledgeBaseRuntimePrincipalColumns(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	expectColumnExists(mock, "knowledge_base_source_job_runs", "runtime_effective_role_id", true)
	expectColumnExists(mock, "knowledge_base_source_job_runs", "runtime_is_workspace_owner", true)

	require.NoError(t, HandlerOffset121.VerifyTenantUpgrade(context.Background(), "ws-1", db))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestOffset121VerifyFailsWhenColumnMissing(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	expectColumnExists(mock, "knowledge_base_source_job_runs", "runtime_effective_role_id", false)

	err = HandlerOffset121.VerifyTenantUpgrade(context.Background(), "ws-1", db)
	require.Error(t, err)
	require.Contains(t, err.Error(), "runtime_effective_role_id missing")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestOffset121VerifyFailsWhenOwnerColumnMissing(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	expectColumnExists(mock, "knowledge_base_source_job_runs", "runtime_effective_role_id", true)
	expectColumnExists(mock, "knowledge_base_source_job_runs", "runtime_is_workspace_owner", false)

	err = HandlerOffset121.VerifyTenantUpgrade(context.Background(), "ws-1", db)
	require.Error(t, err)
	require.Contains(t, err.Error(), "runtime_is_workspace_owner missing")
	require.NoError(t, mock.ExpectationsWereMet())
}
