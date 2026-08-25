package agentresource

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/matrixflow/moi-core/catalog/pkg/service/storage/transaction"
)

func TestSQLAgentStoreCreateKnowledgeBase(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	now := time.Unix(100, 0)
	lastIndexedAt := time.Unix(90, 0)
	store := NewSQLAgentStore(transaction.NewManager(db), "")
	kb := testKnowledgeBase(now, &lastIndexedAt)

	mock.ExpectExec("INSERT INTO agent_resource_knowledge_bases").
		WithArgs(
			"kb_1", "ws_1", "Supplier catalog", "Approved supplier documents", KnowledgeBaseStatusActive, KnowledgeBaseSourceCatalogResource,
			`[{"type":"catalog","id":"cat_1","role":"source","config":{"scope":"workspace"}}]`, "ret_1", `["bid","supplier"]`, "user_1",
			KnowledgeBaseVisibilityWorkspace, KnowledgeBaseIndexStatusReady, lastIndexedAt, "", 1, `{"env":"test"}`, `{"owner":"ops"}`, `{"domain":"supply"}`,
			"user_1", "user_1", now, now,
		).
		WillReturnResult(sqlmock.NewResult(0, 1))

	created, err := store.CreateKnowledgeBase(context.Background(), kb)
	if err != nil {
		t.Fatalf("CreateKnowledgeBase() error = %v", err)
	}
	if created.ID != kb.ID || created.CatalogAssetRefs[0].ID != "cat_1" {
		t.Fatalf("created = %+v", created)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestSQLAgentStoreGetKnowledgeBasesByIDs(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	now := time.Unix(100, 0)
	lastIndexedAt := time.Unix(90, 0)
	store := NewSQLAgentStore(transaction.NewManager(db), "")
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, workspace_id, name, description, status, source_type, catalog_asset_refs_json,")+`(?s).*`+regexp.QuoteMeta("WHERE workspace_id = ? AND id IN (?,?)")).
		WithArgs("ws_1", "kb_1", "missing").
		WillReturnRows(sqlmock.NewRows(knowledgeBaseRowColumns()).
			AddRow(
				"kb_1", "ws_1", "Supplier catalog", "Approved supplier documents", KnowledgeBaseStatusActive, KnowledgeBaseSourceCatalogResource,
				`[{"type":"catalog","id":"cat_1","role":"source","config":{"scope":"workspace"}}]`, "ret_1", `["bid","supplier"]`, "user_1",
				KnowledgeBaseVisibilityWorkspace, KnowledgeBaseIndexStatusReady, lastIndexedAt, "", 1, `{"env":"test"}`, `{"owner":"ops"}`, `{"domain":"supply"}`,
				"user_1", "user_1", now, now,
			))

	items, err := store.GetKnowledgeBasesByIDs(context.Background(), "ws_1", []string{"kb_1", "missing"})
	if err != nil {
		t.Fatalf("GetKnowledgeBasesByIDs() error = %v", err)
	}
	if len(items) != 1 || items["kb_1"].CatalogAssetRefs[0].ID != "cat_1" {
		t.Fatalf("items = %+v", items)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestSQLAgentStoreListKnowledgeBases(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	now := time.Unix(100, 0)
	lastIndexedAt := time.Unix(90, 0)
	store := NewSQLAgentStore(transaction.NewManager(db), "")

	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM agent_resource_knowledge_bases").
		WithArgs("ws_1", KnowledgeBaseStatusActive, KnowledgeBaseSourceCatalogResource, KnowledgeBaseVisibilityWorkspace, KnowledgeBaseIndexStatusReady, "user_1", "%supplier%", "%supplier%").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, workspace_id, name, description, status, source_type, catalog_asset_refs_json,")).
		WithArgs("ws_1", KnowledgeBaseStatusActive, KnowledgeBaseSourceCatalogResource, KnowledgeBaseVisibilityWorkspace, KnowledgeBaseIndexStatusReady, "user_1", "%supplier%", "%supplier%", 20, 5).
		WillReturnRows(sqlmock.NewRows(knowledgeBaseRowColumns()).
			AddRow(
				"kb_1", "ws_1", "Supplier catalog", "Approved supplier documents", KnowledgeBaseStatusActive, KnowledgeBaseSourceCatalogResource,
				`[{"type":"catalog","id":"cat_1","role":"source","config":{"scope":"workspace"}}]`, "ret_1", `["bid","supplier"]`, "user_1",
				KnowledgeBaseVisibilityWorkspace, KnowledgeBaseIndexStatusReady, lastIndexedAt, "", 1, `{"env":"test"}`, `{"owner":"ops"}`, `{"domain":"supply"}`,
				"user_1", "user_1", now, now,
			))

	items, total, err := store.ListKnowledgeBases(context.Background(), KnowledgeBaseListFilter{
		WorkspaceID: "ws_1",
		Status:      KnowledgeBaseStatusActive,
		SourceType:  KnowledgeBaseSourceCatalogResource,
		Visibility:  KnowledgeBaseVisibilityWorkspace,
		IndexStatus: KnowledgeBaseIndexStatusReady,
		OwnerUserID: "user_1",
		Query:       "supplier",
		Limit:       20,
		Offset:      5,
	})
	if err != nil {
		t.Fatalf("ListKnowledgeBases() error = %v", err)
	}
	if total != 1 || len(items) != 1 || items[0].CatalogAssetRefs[0].ID != "cat_1" {
		t.Fatalf("total=%d items=%+v", total, items)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestSQLAgentStoreUpdateKnowledgeBaseReturnsNotFoundWhenNoRowsAffected(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	store := NewSQLAgentStore(transaction.NewManager(db), "")

	mock.ExpectExec("UPDATE agent_resource_knowledge_bases SET").
		WillReturnResult(sqlmock.NewResult(0, 0))

	_, err = store.UpdateKnowledgeBase(context.Background(), testKnowledgeBase(time.Unix(100, 0), nil))
	if err != ErrKnowledgeBaseNotFound {
		t.Fatalf("UpdateKnowledgeBase() error = %v, want ErrKnowledgeBaseNotFound", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestSQLAgentStoreDeleteKnowledgeBase(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	store := NewSQLAgentStore(transaction.NewManager(db), "")

	mock.ExpectExec("DELETE FROM agent_resource_knowledge_bases").
		WithArgs("ws_1", "kb_1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := store.DeleteKnowledgeBase(context.Background(), "ws_1", "kb_1"); err != nil {
		t.Fatalf("DeleteKnowledgeBase() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestSQLAgentStoreDeleteKnowledgeBaseReturnsNotFoundWhenNoRowsAffected(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	store := NewSQLAgentStore(transaction.NewManager(db), "")

	mock.ExpectExec("DELETE FROM agent_resource_knowledge_bases").
		WithArgs("ws_1", "kb_1").
		WillReturnResult(sqlmock.NewResult(0, 0))

	err = store.DeleteKnowledgeBase(context.Background(), "ws_1", "kb_1")
	if !errors.Is(err, ErrKnowledgeBaseNotFound) {
		t.Fatalf("DeleteKnowledgeBase() error = %v, want ErrKnowledgeBaseNotFound", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func testKnowledgeBase(now time.Time, lastIndexedAt *time.Time) KnowledgeBase {
	return KnowledgeBase{
		ID:          "kb_1",
		WorkspaceID: "ws_1",
		Name:        "Supplier catalog",
		Description: "Approved supplier documents",
		Status:      KnowledgeBaseStatusActive,
		SourceType:  KnowledgeBaseSourceCatalogResource,
		CatalogAssetRefs: []KnowledgeCatalogAssetRef{
			{Type: "catalog", ID: "cat_1", Role: "source", Config: map[string]any{"scope": "workspace"}},
		},
		DefaultRetrievalProfileRef: "ret_1",
		Tags:                       []string{"bid", "supplier"},
		OwnerUserID:                "user_1",
		Visibility:                 KnowledgeBaseVisibilityWorkspace,
		IndexStatus:                KnowledgeBaseIndexStatusReady,
		LastIndexedAt:              lastIndexedAt,
		Version:                    1,
		Labels:                     map[string]string{"env": "test"},
		Annotations:                map[string]string{"owner": "ops"},
		Metadata:                   map[string]any{"domain": "supply"},
		CreatedBy:                  "user_1",
		UpdatedBy:                  "user_1",
		CreatedAt:                  now,
		UpdatedAt:                  now,
	}
}

func knowledgeBaseRowColumns() []string {
	return []string{
		"id", "workspace_id", "name", "description", "status", "source_type", "catalog_asset_refs_json",
		"default_retrieval_profile_ref", "tags_json", "owner_user_id", "visibility", "index_status",
		"last_indexed_at", "last_index_error", "version", "labels_json", "annotations_json", "metadata_json",
		"created_by", "updated_by", "created_at", "updated_at",
	}
}
