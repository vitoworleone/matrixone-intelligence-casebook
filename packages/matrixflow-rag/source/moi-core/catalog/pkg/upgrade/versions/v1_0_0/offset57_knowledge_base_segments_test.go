package v1_0_0

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestVersionOffset60VerifyTenantUpgradeChecksKnowledgeBaseTablesAndSegments(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery("FROM information_schema\\.tables").
		WithArgs("system_resource_display_mapping").
		WillReturnRows(sqlmock.NewRows([]string{"1"}).AddRow(1))
	for _, table := range knowledgeBaseBaseTableNames() {
		mock.ExpectQuery("FROM information_schema\\.tables").
			WithArgs(table).
			WillReturnRows(sqlmock.NewRows([]string{"1"}).AddRow(1))
	}
	for _, column := range []string{"tags", "force_enabled_after_expiry"} {
		mock.ExpectQuery(regexp.QuoteMeta("SELECT 1 FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? AND COLUMN_NAME = ? LIMIT 1")).
			WithArgs("knowledge_base_sources", column).
			WillReturnRows(sqlmock.NewRows([]string{"1"}).AddRow(1))
	}
	for _, table := range knowledgeBaseSegmentTableNames() {
		mock.ExpectQuery("FROM information_schema\\.tables").
			WithArgs(table).
			WillReturnRows(sqlmock.NewRows([]string{"1"}).AddRow(1))
	}
	expectOffset59IndexExists(mock, "knowledge_base_sources", "uk_kbs_model_source_file", false)
	expectOffset59IndexExists(mock, "knowledge_base_sources", "idx_kbs_source_file", true)
	for _, spec := range []struct {
		table   string
		columns []string
	}{
		{"knowledge_base_segment_versions", []string{"version_id", "model_id", "source_id", "kb_file_id", "index_version", "status", "source", "vector_table", "embedding_model"}},
		{"knowledge_base_segments", []string{"segment_id", "version_id", "model_id", "source_id", "kb_file_id", "index_version", "level", "chunk_index", "chunk_id", "identity_key", "enabled"}},
		{"knowledge_base_chunk_recall_stats", []string{"model_id", "source_id", "kb_file_id", "index_version", "level", "chunk_index", "chunk_id", "identity_key", "recall_count"}},
	} {
		for _, column := range spec.columns {
			mock.ExpectQuery(regexp.QuoteMeta("SELECT 1 FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? AND COLUMN_NAME = ? LIMIT 1")).
				WithArgs(spec.table, column).
				WillReturnRows(sqlmock.NewRows([]string{"1"}).AddRow(1))
		}
	}

	if err := HandlerOffset60.VerifyTenantUpgrade(context.Background(), "ws_1", db); err != nil {
		t.Fatalf("VerifyTenantUpgrade: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func expectOffset59IndexExists(mock sqlmock.Sqlmock, table, index string, exists bool) {
	rows := sqlmock.NewRows([]string{"1"})
	if exists {
		rows.AddRow(1)
	}
	mock.ExpectQuery("FROM information_schema\\.STATISTICS").
		WithArgs(table, index).
		WillReturnRows(rows)
}
