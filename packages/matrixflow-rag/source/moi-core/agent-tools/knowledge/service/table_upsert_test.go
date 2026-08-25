package service

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/matrixflow/moi-core/agent-tools/knowledge"
)

type upsertTableSchemaReader struct {
	columns []TableColumns
}

func (r upsertTableSchemaReader) ListTables(context.Context, knowledge.WorkspaceScope) ([]string, error) {
	return nil, nil
}

func (r upsertTableSchemaReader) ListColumns(context.Context, knowledge.WorkspaceScope, []string) ([]TableColumns, error) {
	return r.columns, nil
}

func (r upsertTableSchemaReader) ListSemanticEntries(context.Context, knowledge.WorkspaceScope) ([]knowledge.SemanticEntry, error) {
	return nil, nil
}

func (r upsertTableSchemaReader) ReadSampleRows(context.Context, knowledge.WorkspaceScope, string, int) ([][]any, error) {
	return nil, nil
}

type upsertTableMutationExecutor struct {
	dbName string
	sql    string
	args   []any
	scope  knowledge.WorkspaceScope
}

func (e *upsertTableMutationExecutor) ExecuteMutation(ctx context.Context, dbName string, statement string, args ...any) (int64, error) {
	e.dbName = dbName
	e.sql = statement
	e.args = append([]any(nil), args...)
	e.scope = knowledge.ScopeFromContext(ctx)
	return 1, nil
}

func TestUpsertKnowledgeTableUsesOnlySelectedTableAndParameterizedValues(t *testing.T) {
	executor := &upsertTableMutationExecutor{}
	service := NewUpsertKnowledgeTable(Deps{
		SchemaReader: upsertTableSchemaReader{columns: []TableColumns{{
			TableName: "moi_github_identities",
			Columns: []knowledge.ColumnInfo{
				{Name: "github_login", Type: "VARCHAR(255)", PrimaryKey: true},
				{Name: "evidence", Type: "JSON"},
				{Name: "status", Type: "VARCHAR(32)"},
			},
		}}},
		SQLMutationExecutor: executor,
	})
	scope := knowledge.WorkspaceScope{
		WorkspaceID: "workspace_1",
		DBName:      "workspace_db",
		Tables:      []string{"moi_github_identities"},
	}
	response, err := service.Execute(context.Background(), knowledge.UpsertKnowledgeTableRequest{
		Scope:     scope,
		TableName: "moi_github_identities",
		Key:       map[string]any{"github_login": "octocat"},
		Values: map[string]any{
			"evidence": map[string]any{"source": "user confirmation"},
			"status":   "confirmed",
		},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if response == nil || response.RowsAffected != 1 || response.RecordCount != 1 || response.TableName != "moi_github_identities" {
		t.Fatalf("response = %#v", response)
	}
	if executor.dbName != "workspace_db" {
		t.Fatalf("db name = %q, want workspace_db", executor.dbName)
	}
	if !reflect.DeepEqual(executor.scope, scope) {
		t.Fatalf("scope = %#v, want %#v", executor.scope, scope)
	}
	if got, want := executor.sql, "INSERT INTO `moi_github_identities` (`github_login`, `evidence`, `status`) VALUES (?, ?, ?) ON DUPLICATE KEY UPDATE `evidence` = VALUES(`evidence`), `status` = VALUES(`status`)"; got != want {
		t.Fatalf("statement = %q, want %q", got, want)
	}
	if len(executor.args) != 3 || executor.args[0] != "octocat" || executor.args[2] != "confirmed" {
		t.Fatalf("args = %#v", executor.args)
	}
	evidence, ok := executor.args[1].([]byte)
	if !ok || string(evidence) != `{"source":"user confirmation"}` {
		t.Fatalf("evidence argument = %#v", executor.args[1])
	}
}

func TestUpsertKnowledgeTableBatchesRecordsInOneParameterizedStatement(t *testing.T) {
	executor := &upsertTableMutationExecutor{}
	service := NewUpsertKnowledgeTable(Deps{
		SchemaReader: upsertTableSchemaReader{columns: []TableColumns{{
			TableName: "moi_github_identities",
			Columns: []knowledge.ColumnInfo{
				{Name: "github_login", Type: "VARCHAR(255)", PrimaryKey: true},
				{Name: "wecom_user_id", Type: "VARCHAR(255)"},
				{Name: "status", Type: "VARCHAR(32)"},
			},
		}}},
		SQLMutationExecutor: executor,
	})
	response, err := service.Execute(context.Background(), knowledge.UpsertKnowledgeTableRequest{
		Scope: knowledge.WorkspaceScope{
			WorkspaceID: "workspace_1",
			DBName:      "workspace_db",
			Tables:      []string{"moi_github_identities"},
		},
		TableName: "moi_github_identities",
		Records: []knowledge.UpsertKnowledgeTableRecord{
			{Key: map[string]any{"github_login": "alice"}, Values: map[string]any{"wecom_user_id": "wecom-alice", "status": "confirmed"}},
			{Key: map[string]any{"github_login": "bob"}, Values: map[string]any{"wecom_user_id": "wecom-bob", "status": "confirmed"}},
		},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if response == nil || response.RecordCount != 2 || response.Key != nil {
		t.Fatalf("response = %#v", response)
	}
	if got, want := executor.sql, "INSERT INTO `moi_github_identities` (`github_login`, `status`, `wecom_user_id`) VALUES (?, ?, ?), (?, ?, ?) ON DUPLICATE KEY UPDATE `status` = VALUES(`status`), `wecom_user_id` = VALUES(`wecom_user_id`)"; got != want {
		t.Fatalf("statement = %q, want %q", got, want)
	}
	if got, want := executor.args, []any{"alice", "confirmed", "wecom-alice", "bob", "confirmed", "wecom-bob"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("args = %#v, want %#v", got, want)
	}
}

func TestUpsertKnowledgeTableRejectsBatchRecordsWithDifferentColumns(t *testing.T) {
	service := NewUpsertKnowledgeTable(Deps{
		SchemaReader: upsertTableSchemaReader{columns: []TableColumns{{
			TableName: "moi_github_identities",
			Columns: []knowledge.ColumnInfo{
				{Name: "github_login", Type: "VARCHAR(255)", PrimaryKey: true},
				{Name: "wecom_user_id", Type: "VARCHAR(255)"},
				{Name: "status", Type: "VARCHAR(32)"},
			},
		}}},
		SQLMutationExecutor: &upsertTableMutationExecutor{},
	})
	_, err := service.Execute(context.Background(), knowledge.UpsertKnowledgeTableRequest{
		Scope:     knowledge.WorkspaceScope{WorkspaceID: "workspace_1", DBName: "workspace_db", Tables: []string{"moi_github_identities"}},
		TableName: "moi_github_identities",
		Records: []knowledge.UpsertKnowledgeTableRecord{
			{Key: map[string]any{"github_login": "alice"}, Values: map[string]any{"status": "confirmed"}},
			{Key: map[string]any{"github_login": "bob"}, Values: map[string]any{"status": "confirmed", "wecom_user_id": "wecom-bob"}},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "same key and values column sets") {
		t.Fatalf("Execute() error = %v, want batch column-set rejection", err)
	}
}

func TestUpsertKnowledgeTableRejectsUnselectedTable(t *testing.T) {
	service := NewUpsertKnowledgeTable(Deps{
		SchemaReader:        upsertTableSchemaReader{},
		SQLMutationExecutor: &upsertTableMutationExecutor{},
	})
	_, err := service.Execute(context.Background(), knowledge.UpsertKnowledgeTableRequest{
		Scope: knowledge.WorkspaceScope{
			WorkspaceID: "workspace_1",
			DBName:      "workspace_db",
			Tables:      []string{"moi_github_identities"},
		},
		TableName: "unrelated_table",
		Key:       map[string]any{"id": "1"},
		Values:    map[string]any{"status": "confirmed"},
	})
	if err == nil || !strings.Contains(err.Error(), "outside selected knowledge table scope") {
		t.Fatalf("Execute() error = %v, want selected scope rejection", err)
	}
}
