package v1_0_0

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/matrixflow/moi-core/catalog/pkg/systemresourcedisplay"
	"github.com/matrixflow/moi-core/catalog/pkg/upgrade/versions"
)

func TestVersionOffset60Metadata(t *testing.T) {
	md := HandlerOffset60.Metadata()
	if md.VersionOffset != 60 {
		t.Fatalf("VersionOffset = %d, want 60", md.VersionOffset)
	}
	if md.UpgradeSystem != versions.Yes {
		t.Fatal("expected UpgradeSystem=Yes")
	}
	if md.UpgradeTenant != versions.Yes {
		t.Fatal("expected UpgradeTenant=Yes")
	}
}

func TestVersionOffset60HandleSystemUpgradeRepairsOffset59Collision(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectBegin()
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}

	mock.ExpectExec(regexp.QuoteMeta("ALTER TABLE `compute_resource_spec` ADD COLUMN `description_en` TEXT COMMENT '英文描述' AFTER `description`")).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE `compute_resource_spec` SET `description_en` = '' WHERE `description_en` IS NULL")).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	if err := HandlerOffset60.HandleSystemUpgrade(context.Background(), tx); err != nil {
		t.Fatalf("HandleSystemUpgrade: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestVersionOffset60HandleTenantUpgradeCreatesKnowledgeBaseTables(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	expectOffset54SchemaAndRoleBackfill(mock)
	for _, mapping := range systemresourcedisplay.BuiltinRoleDisplayMappings() {
		expectMappingInsert(mock, mapping)
	}
	for _, table := range []string{
		"knowledge_base_data_domains",
		"knowledge_base_source_jobs",
		"knowledge_base_sources",
		"knowledge_base_source_job_runs",
		"knowledge_base_raw_volumes",
	} {
		mock.ExpectExec(regexp.QuoteMeta("CREATE TABLE IF NOT EXISTS " + table)).
			WillReturnResult(sqlmock.NewResult(0, 0))
	}
	expectOffset59IndexExists(mock, "knowledge_base_sources", "uk_kbs_model_source_file", false)
	mock.ExpectExec(regexp.QuoteMeta("CREATE INDEX `idx_kbs_source_file` ON `knowledge_base_sources` (`model_id`, `source_type`, `source_file_id`)")).
		WillReturnResult(sqlmock.NewResult(0, 0))
	for _, table := range knowledgeBaseSegmentTableNames() {
		mock.ExpectExec(regexp.QuoteMeta("CREATE TABLE IF NOT EXISTS " + table)).
			WillReturnResult(sqlmock.NewResult(0, 0))
	}

	if err := HandlerOffset60.HandleTenantUpgrade(context.Background(), "ws_1", "acc_1", db); err != nil {
		t.Fatalf("HandleTenantUpgrade: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}
