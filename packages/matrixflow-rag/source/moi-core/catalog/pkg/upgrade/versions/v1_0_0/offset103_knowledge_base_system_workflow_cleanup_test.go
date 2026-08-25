package v1_0_0

import (
	"context"
	"database/sql"
	"regexp"
	"sort"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	userservice "github.com/matrixflow/moi-core/catalog/pkg/service/user"
	"github.com/matrixflow/moi-core/catalog/pkg/upgrade/versions"
	"github.com/stretchr/testify/require"
)

func TestOffset103Metadata(t *testing.T) {
	metadata := HandlerOffset103.Metadata()
	require.Equal(t, uint32(103), metadata.VersionOffset)
	require.Equal(t, versions.Yes, metadata.UpgradeSystem)
	require.Equal(t, versions.No, metadata.UpgradeTenant)
}

func TestOffset103DeletesOnlyVerifiedLegacySystemOwnedKnowledgeBaseWorkflowApps(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectBegin()
	tx, err := db.Begin()
	require.NoError(t, err)
	mock.ExpectQuery(regexp.QuoteMeta(`
SELECT id
FROM users
WHERE email = ?`)).
		WithArgs(userservice.SystemUserEmail).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("system-user"))
	mock.ExpectQuery(regexp.QuoteMeta(`
SELECT id, workspace_id, default_values_json
FROM workflow_app
WHERE user_id = ?
  AND source_type = ?
  AND execution_mode = ?
  AND id LIKE ?`)).
		WithArgs(
			"system-user",
			offset103KnowledgeBaseWorkflowSourceType,
			offset103KnowledgeBaseWorkflowExecutionMode,
			offset103KnowledgeBaseWorkflowIDPrefix+"%",
		).
		WillReturnRows(sqlmock.NewRows([]string{"id", "workspace_id", "default_values_json"}).
			AddRow(offset103KnowledgeBaseWorkflowIDs("workspace-1", 101)[0], "workspace-1", `{"semantic_model_id":101}`).
			AddRow(offset103KnowledgeBaseWorkflowIDs("workspace-1", 101)[1], "workspace-1", `{"semantic_model_id":101}`).
			AddRow(offset103KnowledgeBaseWorkflowIDs("workspace-1", 101)[2], "workspace-1", `{"semantic_model_id":101}`).
			AddRow("kb-rag-workflow-ffffffffffffffffffffffff", "workspace-1", `{"semantic_model_id":101}`))
	workflowIDs := offset103KnowledgeBaseWorkflowIDs("workspace-1", 101)
	sort.Strings(workflowIDs)
	mock.ExpectExec(regexp.QuoteMeta(offset103DeleteKnowledgeBaseWorkflowAppsQuery(len(workflowIDs)))).
		WithArgs(
			"workspace-1",
			"system-user",
			offset103KnowledgeBaseWorkflowSourceType,
			offset103KnowledgeBaseWorkflowExecutionMode,
			workflowIDs[0],
			workflowIDs[1],
			workflowIDs[2],
		).
		WillReturnResult(sqlmock.NewResult(0, 3))

	require.NoError(t, HandlerOffset103.HandleSystemUpgrade(context.Background(), tx))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestOffset103SkipsUnrecognizedKnowledgeBaseWorkflowPrefix(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectBegin()
	tx, err := db.Begin()
	require.NoError(t, err)
	mock.ExpectQuery(regexp.QuoteMeta(`
SELECT id
FROM users
WHERE email = ?`)).
		WithArgs(userservice.SystemUserEmail).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("system-user"))
	mock.ExpectQuery(regexp.QuoteMeta(`
SELECT id, workspace_id, default_values_json
FROM workflow_app
WHERE user_id = ?
  AND source_type = ?
  AND execution_mode = ?
  AND id LIKE ?`)).
		WithArgs(
			"system-user",
			offset103KnowledgeBaseWorkflowSourceType,
			offset103KnowledgeBaseWorkflowExecutionMode,
			offset103KnowledgeBaseWorkflowIDPrefix+"%",
		).
		WillReturnRows(sqlmock.NewRows([]string{"id", "workspace_id", "default_values_json"}).
			AddRow("kb-rag-workflow-ffffffffffffffffffffffff", "workspace-1", `{"semantic_model_id":101}`).
			AddRow("kb-rag-workflow-eeeeeeeeeeeeeeeeeeeeeeee", "workspace-1", nil))

	require.NoError(t, HandlerOffset103.HandleSystemUpgrade(context.Background(), tx))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestOffset103DoesNothingWhenSystemUserDoesNotExist(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectBegin()
	tx, err := db.Begin()
	require.NoError(t, err)
	mock.ExpectQuery(regexp.QuoteMeta(`
SELECT id
FROM users
WHERE email = ?`)).
		WithArgs(userservice.SystemUserEmail).
		WillReturnError(sql.ErrNoRows)

	require.NoError(t, HandlerOffset103.HandleSystemUpgrade(context.Background(), tx))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestOffset103RejectsNilSystemTransaction(t *testing.T) {
	require.EqualError(t, HandlerOffset103.HandleSystemUpgrade(context.Background(), (*sql.Tx)(nil)), "system database transaction is nil")
}
