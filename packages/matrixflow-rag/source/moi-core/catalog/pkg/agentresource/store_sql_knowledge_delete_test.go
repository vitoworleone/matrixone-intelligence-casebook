package agentresource

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/matrixflow/moi-core/catalog/pkg/service/storage/transaction"
)

func TestSQLAgentStoreListAgentsReferencingKnowledgeBaseForUpdate(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	store := NewSQLAgentStore(transaction.NewManager(db), "")
	now := time.Unix(100, 0)
	bindingJSON := `{"knowledge_base_refs":[{"id":"20001","workspace_id":"ws_1","kind":"knowledge_base"},{"id":"20002","workspace_id":"ws_1","kind":"knowledge_base"}]}`

	mock.ExpectQuery("SELECT id, workspace_id, name, description, avatar_ref, icon, display_tags_json, category, sort_order[\\s\\S]*FOR UPDATE").
		WithArgs("ws_1", "%20001%").
		WillReturnRows(sqlmock.NewRows(agentRowColumns()).
			AddRow("agent_1", "ws_1", "agent", "", "", "", "[]", "", 0,
				`{}`, `{}`, `{}`, bindingJSON, `{}`, `{}`, `{}`,
				"active", 1, 1, `{}`, `{}`, "custom", "", `{}`,
				"u1", "u1", now, now))

	agents, err := store.ListAgentsReferencingKnowledgeBaseForUpdate(context.Background(), "ws_1", "20001")
	if err != nil {
		t.Fatalf("ListAgentsReferencingKnowledgeBaseForUpdate() error = %v", err)
	}
	if len(agents) != 1 || agents[0].ID != "agent_1" {
		t.Fatalf("agents = %+v", agents)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestSQLAgentStoreListAgentBindingsReferencingKnowledgeBaseForUpdate(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	store := NewSQLAgentStore(transaction.NewManager(db), "")
	now := time.Unix(100, 0)
	bindingJSON := `{"knowledge_base_refs":[{"id":"20001","workspace_id":"ws_1","kind":"knowledge_base"}]}`

	mock.ExpectQuery("SELECT workspace_id, agent_workspace_id, agent_id, binding_summary_json[\\s\\S]*FOR UPDATE").
		WithArgs("ws_1", "%20001%").
		WillReturnRows(sqlmock.NewRows([]string{
			"workspace_id", "agent_workspace_id", "agent_id", "binding_summary_json",
			"created_by", "updated_by", "created_at", "updated_at",
		}).AddRow("ws_1", "system", "explore", bindingJSON, "u1", "u1", now, now))

	bindings, err := store.ListAgentBindingsReferencingKnowledgeBaseForUpdate(context.Background(), "ws_1", "20001")
	if err != nil {
		t.Fatalf("ListAgentBindingsReferencingKnowledgeBaseForUpdate() error = %v", err)
	}
	if len(bindings) != 1 || bindings[0].AgentID != "explore" {
		t.Fatalf("bindings = %+v", bindings)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestSQLAgentStoreListNonDisabledAgentVersions(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	store := NewSQLAgentStore(transaction.NewManager(db), "")
	now := time.Unix(100, 0)
	// Candidate scan must remain unlocked; require query end without FOR UPDATE.
	mock.ExpectQuery("(?s)SELECT workspace_id, agent_id, version, foundation_ref, source_digest, min_moi_version, manifest_json, status, diagnostics_json.*ORDER BY agent_id, version$").
		WithArgs("ws_1", AgentVersionStatusDisabled).
		WillReturnRows(sqlmock.NewRows([]string{
			"workspace_id", "agent_id", "version", "foundation_ref", "source_digest", "min_moi_version", "manifest_json", "status", "diagnostics_json",
			"loaded_by", "loaded_at", "disabled_by", "disabled_at",
		}).AddRow("ws_1", "agent_1", "1.0.0", "", "digest", "1.0.0", `{}`, AgentVersionStatusRunnable, `null`, "u1", now, "", nil))

	versions, err := store.ListNonDisabledAgentVersions(context.Background(), "ws_1")
	if err != nil {
		t.Fatalf("ListNonDisabledAgentVersions() error = %v", err)
	}
	if len(versions) != 1 || versions[0].Version != "1.0.0" {
		t.Fatalf("versions = %+v", versions)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestSQLAgentStoreGetAgentVersionForUpdate(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	store := NewSQLAgentStore(transaction.NewManager(db), "")
	now := time.Unix(100, 0)
	mock.ExpectQuery("SELECT workspace_id, agent_id, version, foundation_ref, source_digest, min_moi_version, manifest_json, status, diagnostics_json[\\s\\S]*FOR UPDATE").
		WithArgs("ws_1", "agent_1", "1.0.0").
		WillReturnRows(sqlmock.NewRows([]string{
			"workspace_id", "agent_id", "version", "foundation_ref", "source_digest", "min_moi_version", "manifest_json", "status", "diagnostics_json",
			"loaded_by", "loaded_at", "disabled_by", "disabled_at",
		}).AddRow("ws_1", "agent_1", "1.0.0", "", "digest", "1.0.0", `{}`, AgentVersionStatusRunnable, `null`, "u1", now, "", nil))

	got, err := store.GetAgentVersionForUpdate(context.Background(), "ws_1", "agent_1", "1.0.0")
	if err != nil {
		t.Fatalf("GetAgentVersionForUpdate() error = %v", err)
	}
	if got.AgentID != "agent_1" || got.Version != "1.0.0" {
		t.Fatalf("got = %+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}
