package session

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/matrixorigin/matrixflow/moi-backend/pkg/ctxutil"
	"github.com/matrixorigin/matrixflow/moi-backend/pkg/i18n"

	moi "github.com/matrixflow/moi-core/go-sdk"
	gmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type reconcileLockDriver struct {
	mu sync.Mutex
}

type reconcileLockConn struct {
	driver *reconcileLockDriver
	locked bool
}

type reconcileLockTx struct {
	conn *reconcileLockConn
}

type reconcileLockRows struct {
	done bool
}

var reconcileLockDriverID atomic.Uint64

func (d *reconcileLockDriver) Open(string) (driver.Conn, error) {
	return &reconcileLockConn{driver: d}, nil
}

func (c *reconcileLockConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("prepare is not supported")
}

func (c *reconcileLockConn) Close() error { return nil }

func (c *reconcileLockConn) Begin() (driver.Tx, error) {
	return c.BeginTx(context.Background(), driver.TxOptions{})
}

func (c *reconcileLockConn) BeginTx(context.Context, driver.TxOptions) (driver.Tx, error) {
	return &reconcileLockTx{conn: c}, nil
}

func (c *reconcileLockConn) QueryContext(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
	c.driver.mu.Lock()
	c.locked = true
	return &reconcileLockRows{}, nil
}

func (c *reconcileLockConn) ExecContext(context.Context, string, []driver.NamedValue) (driver.Result, error) {
	return driver.RowsAffected(1), nil
}

func (tx *reconcileLockTx) Commit() error {
	tx.unlock()
	return nil
}

func (tx *reconcileLockTx) Rollback() error {
	tx.unlock()
	return nil
}

func (tx *reconcileLockTx) unlock() {
	if tx.conn.locked {
		tx.conn.locked = false
		tx.conn.driver.mu.Unlock()
	}
}

func (r *reconcileLockRows) Columns() []string { return []string{"model_id"} }
func (r *reconcileLockRows) Close() error      { return nil }
func (r *reconcileLockRows) Next(values []driver.Value) error {
	if r.done {
		return io.EOF
	}
	r.done = true
	values[0] = int64(77)
	return nil
}

func TestCommitCompletedKnowledgeBaseTableJobsSerializesConcurrentModelUpdates(t *testing.T) {
	dbDriver := &reconcileLockDriver{}
	driverName := fmt.Sprintf("semantic-model-reconcile-lock-%d", reconcileLockDriverID.Add(1))
	sql.Register(driverName, dbDriver)
	sqlDB, err := sql.Open(driverName, "")
	if err != nil {
		t.Fatalf("open lock db: %v", err)
	}
	defer sqlDB.Close()
	sqlDB.SetMaxOpenConns(2)
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: sqlDB, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}

	var modelMu sync.Mutex
	tables := []semanticModelTableSource{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		modelMu.Lock()
		defer modelMu.Unlock()
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id": 77, "name": "kb", "description": "docs", "tables": tables,
				"files": map[string]any{"file_ids": []string{}},
			})
		case http.MethodPut:
			var req moi.SemanticModelUpsertRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Errorf("decode semantic update: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			var nextTables []semanticModelTableSource
			if err := json.Unmarshal(req.Tables, &nextTables); err != nil {
				t.Errorf("decode semantic tables: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			tables = nextTables
			_ = json.NewEncoder(w).Encode(map[string]any{"updated": true})
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()
	client, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer client.Close()

	svc := &semanticModelService{}
	ctx := ctxutil.WithTenantDB(context.Background(), tenantDB)
	start := make(chan struct{})
	errs := make(chan error, 2)
	for i, tableName := range []string{"orders", "customers"} {
		i, tableName := i, tableName
		go func() {
			<-start
			sourceID := "source-" + tableName
			jobID := "job-" + tableName
			errs <- svc.commitCompletedKnowledgeBaseTableJobs(ctx, client, "ws-1", 77, "user-1", nil,
				[]completedKnowledgeBaseTableJob{{
					source: KnowledgeBaseSourceRecord{SourceID: sourceID, ModelID: 77, SourceType: kbSourceTypeCatalogTable, KBTableID: int64Ptr(int64(2001 + i)), DBName: stringPtr("kb_docs"), TableName: stringPtr(tableName), Status: kbSourceStatusSucceeded},
					job:    KnowledgeBaseSourceJobRun{JobID: jobID, JobStatus: kbSourceJobSucceeded},
				}},
			)
		}()
	}
	close(start)
	for range 2 {
		if err := <-errs; err != nil {
			t.Fatalf("concurrent reconcile commit: %v", err)
		}
	}

	modelMu.Lock()
	defer modelMu.Unlock()
	if len(tables) != 1 || tables[0].DBName != "kb_docs" || len(tables[0].TableNames) != 2 {
		t.Fatalf("concurrent semantic tables = %+v", tables)
	}
}

func TestCommitCompletedKnowledgeBaseTableJobsRollsBackBeforeMarkingSuccessWhenCoreUpdateFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id": 77, "name": "kb", "tables": []semanticModelTableSource{},
				"files": map[string]any{"file_ids": []string{}},
			})
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 13, "message": "semantic update failed"})
	}))
	defer server.Close()
	client, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer client.Close()

	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	tenantMock.ExpectBegin()
	tenantMock.ExpectQuery("SELECT model_id.*FROM knowledge_base_data_domains.*FOR UPDATE").
		WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows([]string{"model_id"}).AddRow(int64(77)))
	tenantMock.ExpectRollback()

	err = (&semanticModelService{}).commitCompletedKnowledgeBaseTableJobs(
		ctxutil.WithTenantDB(context.Background(), tenantDB), client, "ws-1", 77, "user-1", nil,
		[]completedKnowledgeBaseTableJob{{
			source: KnowledgeBaseSourceRecord{SourceID: "source-orders", ModelID: 77, SourceType: kbSourceTypeCatalogTable, DBName: stringPtr("kb_docs"), TableName: stringPtr("orders"), Status: kbSourceStatusSucceeded},
			job:    KnowledgeBaseSourceJobRun{JobID: "job-orders", JobStatus: kbSourceJobSucceeded},
		}},
	)
	if err == nil {
		t.Fatal("core update failure should be returned")
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestCommitCompletedKnowledgeBaseTableJobsMarksFailedWhenDataDomainIsMissing(t *testing.T) {
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	tenantMock.ExpectBegin()
	tenantMock.ExpectQuery("SELECT model_id.*FROM knowledge_base_data_domains.*FOR UPDATE").
		WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows([]string{"model_id"}))
	tenantMock.ExpectRollback()
	tenantMock.ExpectExec("UPDATE knowledge_base_sources.*SET status = \\?, error = \\?").
		WithArgs(kbSourceStatusFailed, i18n.KeySessionKnowledgeBaseDataDomainNotFound.String(), "user-1", "source-orders").
		WillReturnResult(sqlmock.NewResult(0, 1))
	tenantMock.ExpectExec("UPDATE knowledge_base_source_job_runs.*SET job_status = \\?, error = \\?").
		WithArgs(kbSourceJobFailed, i18n.KeySessionKnowledgeBaseDataDomainNotFound.String(), "user-1", "job-orders").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = (&semanticModelService{}).commitCompletedKnowledgeBaseTableJobs(
		ctxutil.WithTenantDB(context.Background(), tenantDB), nil, "ws-1", 77, "user-1", nil,
		[]completedKnowledgeBaseTableJob{{
			source: KnowledgeBaseSourceRecord{SourceID: "source-orders", ModelID: 77, SourceType: kbSourceTypeCatalogTable},
			job:    KnowledgeBaseSourceJobRun{JobID: "job-orders"},
		}},
	)
	if !errors.Is(err, errKnowledgeBaseDataDomainLockMissing) {
		t.Fatalf("missing data-domain error = %v", err)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestCommitCompletedKnowledgeBaseTableJobsDoesNotMarkFailedOnDataDomainQueryError(t *testing.T) {
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	tenantMock.ExpectBegin()
	tenantMock.ExpectQuery("SELECT model_id.*FROM knowledge_base_data_domains.*FOR UPDATE").
		WithArgs(int64(77)).
		WillReturnError(errors.New("database unavailable"))
	tenantMock.ExpectRollback()

	err = (&semanticModelService{}).commitCompletedKnowledgeBaseTableJobs(
		ctxutil.WithTenantDB(context.Background(), tenantDB), nil, "ws-1", 77, "user-1", nil,
		[]completedKnowledgeBaseTableJob{{
			source: KnowledgeBaseSourceRecord{SourceID: "source-orders", ModelID: 77, SourceType: kbSourceTypeCatalogTable},
			job:    KnowledgeBaseSourceJobRun{JobID: "job-orders"},
		}},
	)
	if err == nil || errors.Is(err, errKnowledgeBaseDataDomainLockMissing) {
		t.Fatalf("data-domain query error = %v", err)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestCommitCompletedKnowledgeBaseTableJobsSkipsDeletedTableClone(t *testing.T) {
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	tenantMock.ExpectBegin()
	tenantMock.ExpectQuery("SELECT model_id.*FROM knowledge_base_data_domains.*FOR UPDATE").
		WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows([]string{"model_id"}).AddRow(int64(77)))
	tenantMock.ExpectQuery("SELECT COUNT\\(\\*\\).*FROM knowledge_base_sources kbs.*JOIN knowledge_base_source_job_runs jr").
		WithArgs(int64(77), "source-orders", kbSourceStatusRemoved, "job-orders", kbJobTypeTableClone).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(0)))
	tenantMock.ExpectCommit()

	err = (&semanticModelService{}).commitCompletedKnowledgeBaseTableJobs(
		ctxutil.WithTenantDB(context.Background(), tenantDB), nil, "ws-1", 77, "user-1", nil,
		[]completedKnowledgeBaseTableJob{{
			source: KnowledgeBaseSourceRecord{
				SourceID: "source-orders", ModelID: 77, SourceType: kbSourceTypeCatalogTable,
				DBName: stringPtr("kb_docs"), TableName: stringPtr("orders__kb_1234"), Status: kbSourceStatusSucceeded,
			},
			job: KnowledgeBaseSourceJobRun{
				JobID: "job-orders", JobType: kbJobTypeTableClone, JobStatus: kbSourceJobSucceeded,
			},
		}},
	)
	if err != nil {
		t.Fatalf("commit deleted table clone: %v", err)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestCommitCompletedKnowledgeBaseTableJobsSkipsDeletedStructuredLoad(t *testing.T) {
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	tenantMock.ExpectBegin()
	tenantMock.ExpectQuery("SELECT model_id.*FROM knowledge_base_data_domains.*FOR UPDATE").
		WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows([]string{"model_id"}).AddRow(int64(77)))
	tenantMock.ExpectQuery("SELECT COUNT\\(\\*\\).*FROM knowledge_base_sources kbs.*JOIN knowledge_base_source_job_runs jr").
		WithArgs(int64(77), "source-upload", kbSourceStatusRemoved, "job-upload", kbJobTypeLoad).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(0)))
	tenantMock.ExpectCommit()

	err = (&semanticModelService{}).commitCompletedKnowledgeBaseTableJobs(
		ctxutil.WithTenantDB(context.Background(), tenantDB), nil, "ws-1", 77, "user-1", []string{"uploaded-file"},
		[]completedKnowledgeBaseTableJob{{
			source: KnowledgeBaseSourceRecord{
				SourceID: "source-derived-table", ModelID: 77, SourceType: kbSourceTypeCatalogTable,
				DBName: stringPtr("kb_docs"), TableName: stringPtr("orders"), Status: kbSourceStatusSucceeded,
			},
			job:           KnowledgeBaseSourceJobRun{JobID: "job-derived-table", JobType: kbJobTypeLoad, JobStatus: kbSourceJobSucceeded},
			new:           true,
			ownerSourceID: "source-upload",
			ownerJobID:    "job-upload",
			ownerJobType:  kbJobTypeLoad,
		}},
	)
	if err != nil {
		t.Fatalf("commit deleted structured load: %v", err)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}
