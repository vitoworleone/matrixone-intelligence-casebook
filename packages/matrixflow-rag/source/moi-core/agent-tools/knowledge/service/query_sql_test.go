package service

import (
	"context"
	"strings"
	"testing"

	"github.com/matrixflow/moi-core/agent-tools/knowledge"
)

type stubQuerySQLExecutor struct {
	results []*knowledge.SQLExecutionResult
	dbNames []string
	sqls    []string
}

func (s *stubQuerySQLExecutor) ExecuteSQL(_ context.Context, dbName string, sqlText string) (*knowledge.SQLExecutionResult, error) {
	s.dbNames = append(s.dbNames, dbName)
	s.sqls = append(s.sqls, sqlText)
	if len(s.results) == 0 {
		return &knowledge.SQLExecutionResult{}, nil
	}
	result := s.results[0]
	s.results = s.results[1:]
	return result, nil
}

func TestQuerySQLWrapsPreserveLeadingRewriteHint(t *testing.T) {
	sql := `/*+ { "rewrites" : { "d.orders": "select id from d.orders where region = 'east'" } } */ SELECT * FROM orders ORDER BY id;`

	countSQL := wrapQuerySQLCount(sql)
	if !strings.HasPrefix(countSQL, `/*+ { "rewrites"`) {
		t.Fatalf("count SQL should keep rewrite hint at statement head, got %q", countSQL)
	}
	if !strings.Contains(countSQL, "SELECT COUNT(1) AS __query_sql_total_count") {
		t.Fatalf("count SQL missing outer count wrapper: %q", countSQL)
	}
	if strings.Contains(countSQL, "(\n/*+") {
		t.Fatalf("count SQL should not leave rewrite hint inside subquery: %q", countSQL)
	}

	previewSQL := wrapQuerySQLPreview(sql, 50)
	if !strings.HasPrefix(previewSQL, `/*+ { "rewrites"`) {
		t.Fatalf("preview SQL should keep rewrite hint at statement head, got %q", previewSQL)
	}
	if !strings.Contains(previewSQL, "query_sql_preview_source LIMIT 50") {
		t.Fatalf("preview SQL missing outer preview wrapper: %q", previewSQL)
	}
	if strings.Contains(previewSQL, "(\n/*+") {
		t.Fatalf("preview SQL should not leave rewrite hint inside subquery: %q", previewSQL)
	}
}

func TestQuerySQLWrapsLeaveNonRewriteLeadingCommentInsideSubquery(t *testing.T) {
	sql := `/* regular comment */ SELECT * FROM orders`

	countSQL := wrapQuerySQLCount(sql)
	if strings.HasPrefix(countSQL, "/* regular comment */") {
		t.Fatalf("non-rewrite comment should not be promoted ahead of count wrapper: %q", countSQL)
	}
	if !strings.Contains(countSQL, "/* regular comment */ SELECT * FROM orders") {
		t.Fatalf("non-rewrite comment should remain in wrapped SQL: %q", countSQL)
	}
}

func TestQuerySQLValidateAllowsLeadingRewriteHint(t *testing.T) {
	sql := `/*+ {"rewrites":{"d.orders":"SELECT * FROM d.orders"}} */ SELECT * FROM orders`
	if err := validateReadOnlySQL(sql); err != nil {
		t.Fatalf("validateReadOnlySQL returned error: %v", err)
	}
}

func TestQuerySQLValidateAllowsTrailingSemicolon(t *testing.T) {
	sql := `SELECT * FROM orders ORDER BY id;`
	if err := validateReadOnlySQL(sql); err != nil {
		t.Fatalf("validateReadOnlySQL returned error: %v", err)
	}
}

func TestQuerySQLValidateAllowsNonRewriteLeadingComment(t *testing.T) {
	sql := `/* regular comment */ SELECT * FROM orders`
	if err := validateReadOnlySQL(sql); err != nil {
		t.Fatalf("validateReadOnlySQL returned error: %v", err)
	}
}

func TestQuerySQLValidateAllowsWithSelect(t *testing.T) {
	sql := `WITH recent AS (SELECT * FROM orders) SELECT * FROM recent`
	if err := validateReadOnlySQL(sql); err != nil {
		t.Fatalf("validateReadOnlySQL returned error: %v", err)
	}
}

func TestQuerySQLValidateRejectsWithWriteStatements(t *testing.T) {
	for _, sql := range []string{
		`WITH recent AS (SELECT id FROM orders) UPDATE orders SET status = 'done' WHERE id IN (SELECT id FROM recent)`,
		`WITH recent AS (SELECT id FROM orders) DELETE FROM orders WHERE id IN (SELECT id FROM recent)`,
	} {
		if err := validateReadOnlySQL(sql); err == nil {
			t.Fatalf("validateReadOnlySQL(%q) returned nil, want error", sql)
		}
	}
}

func TestQuerySQLValidateRejectsMultipleStatements(t *testing.T) {
	sql := `SELECT * FROM orders; SELECT * FROM customers`
	if err := validateReadOnlySQL(sql); err == nil {
		t.Fatalf("validateReadOnlySQL(%q) returned nil, want error", sql)
	}
}

func TestQuerySQLValidateRejectsUnquotedReservedAlias(t *testing.T) {
	// MySQL 8 treats CURRENT_TIME as reserved; unquoted AS current_time is illegal.
	// MatrixOne may accept this form, but query_sql validates MySQL-compatible syntax.
	err := validateReadOnlySQL(`SELECT NOW() AS current_time`)
	if err == nil {
		t.Fatal("validateReadOnlySQL(AS current_time) = nil, want MySQL parse error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "parse sql") {
		t.Fatalf("error = %q, want parse sql prefix", msg)
	}
	if !strings.Contains(msg, "MySQL 8") && !strings.Contains(msg, "backtick") {
		t.Fatalf("error = %q, want MySQL dialect guidance for reserved alias", msg)
	}
}

func TestQuerySQLValidateAllowsQuotedReservedAliasAndSafeAlias(t *testing.T) {
	for _, sql := range []string{
		"SELECT NOW() AS `current_time`",
		"SELECT NOW() AS now_value",
		"SELECT NOW()",
		"SELECT DATE_ADD('2026-08-07', INTERVAL 1 DAY) AS today_cst",
	} {
		if err := validateReadOnlySQL(sql); err != nil {
			t.Fatalf("validateReadOnlySQL(%q) error = %v", sql, err)
		}
	}
}

func TestQuerySQLScopeAllowsPureScalarSelectWithoutPhysicalTable(t *testing.T) {
	for _, sql := range []string{
		"SELECT NOW() AS now_value",
		"SELECT NOW() AS `current_time`",
		"SELECT 1 AS one",
		"SELECT DATE_ADD('2026-08-07', INTERVAL 1 DAY) AS today_cst",
	} {
		if err := validateSQLTablesInScope(sql, "db_1", []string{"db_1.orders"}, true); err != nil {
			t.Fatalf("validateSQLTablesInScope(%q) error = %v, want pure scalar allowed", sql, err)
		}
		if !isPureScalarSelectSQL(sql) {
			t.Fatalf("isPureScalarSelectSQL(%q) = false, want true", sql)
		}
	}
}

func TestQuerySQLScopeStillRequiresPhysicalTableForNonScalar(t *testing.T) {
	// CTE-only scaffolding without a physical table stays rejected.
	err := validateSQLTablesInScope(
		`WITH recent AS (SELECT 1 AS id) SELECT id FROM recent`,
		"db_1",
		[]string{"db_1.orders"},
		true,
	)
	if err == nil {
		t.Fatal("validateSQLTablesInScope(CTE-only) = nil, want physical table required")
	}
	if !strings.Contains(err.Error(), "physical table") {
		t.Fatalf("error = %q, want physical table required", err)
	}
}

func TestQuerySQLValidateScopeAllowsQuotedTableNameWithDot(t *testing.T) {
	err := validateSQLTablesInScope("SELECT * FROM `test.csv`", "ffff_15", []string{"test.csv"}, true)
	if err != nil {
		t.Fatalf("validateSQLTablesInScope() error = %v", err)
	}
}

// MF-3: bare dotted table names must pair with Scope.DBName, not first-dot split.
func TestQuerySQLTableRefsPreservesDottedBareTableName(t *testing.T) {
	svc := &querySQLService{}
	refs, err := svc.tableRefs(context.Background(), []string{"test.csv"}, "ffff_15")
	if err != nil {
		t.Fatalf("tableRefs() error = %v", err)
	}
	if len(refs) != 1 || refs[0].DBName != "ffff_15" || refs[0].Name != "test.csv" {
		t.Fatalf("tableRefs = %#v, want {DBName:ffff_15 Name:test.csv}", refs)
	}
	refs, err = svc.tableRefs(context.Background(), []string{"ffff_15.test.csv"}, "ffff_15")
	if err != nil {
		t.Fatalf("tableRefs(qualified) error = %v", err)
	}
	if len(refs) != 1 || refs[0].DBName != "ffff_15" || refs[0].Name != "test.csv" {
		t.Fatalf("tableRefs(qualified) = %#v, want {DBName:ffff_15 Name:test.csv}", refs)
	}
}

func TestQuerySQLValidateScopeAllowsSameDatabaseQualifiedScopeTables(t *testing.T) {
	for _, sql := range []string{
		"SELECT * FROM orders",
		"SELECT * FROM ffff_15.orders",
	} {
		if err := validateSQLTablesInScope(sql, "ffff_15", []string{"ffff_15.orders"}, true); err != nil {
			t.Fatalf("validateSQLTablesInScope(%q) error = %v", sql, err)
		}
	}
}

func TestQuerySQLValidateScopeRejectsOtherDatabase(t *testing.T) {
	err := validateSQLTablesInScope("SELECT * FROM other.orders", "ffff_15", []string{"orders"}, true)
	if err == nil || !strings.Contains(err.Error(), "outside selected knowledge table scope") {
		t.Fatalf("validateSQLTablesInScope() error = %v, want outside selected knowledge table scope", err)
	}
}

func TestQuerySQLValidateScopeAllowsMultiDatabaseQualifiedTables(t *testing.T) {
	scopeTables := []string{"sales.orders", "support.tickets"}
	for _, sql := range []string{
		"SELECT * FROM sales.orders",
		"SELECT o.id, t.id FROM sales.orders o JOIN support.tickets t ON o.id = t.order_id",
	} {
		if err := validateSQLTablesInScope(sql, "", scopeTables, true); err != nil {
			t.Fatalf("validateSQLTablesInScope(%q) error = %v", sql, err)
		}
	}
}

func TestQuerySQLValidateScopeMultiDatabaseRequiresQualification(t *testing.T) {
	err := validateSQLTablesInScope("SELECT * FROM orders", "", []string{"sales.orders", "support.tickets"}, true)
	if err == nil || !strings.Contains(err.Error(), "must be referenced as database.table") {
		t.Fatalf("validateSQLTablesInScope() error = %v, want database.table requirement", err)
	}
}

func TestQuerySQLValidateScopeMultiDatabaseRejectsOutOfScope(t *testing.T) {
	err := validateSQLTablesInScope("SELECT * FROM inventory.sku", "", []string{"sales.orders", "support.tickets"}, true)
	if err == nil || !strings.Contains(err.Error(), "outside selected knowledge table scope") {
		t.Fatalf("validateSQLTablesInScope() error = %v, want outside scope", err)
	}
}

// Bare physical dotted name without defaultDB must not become database.table.
func TestQuerySQLValidateScopeRejectsBareDottedPhysicalWithoutDB(t *testing.T) {
	// SQL parser may see unquoted xxx.xxx as schema.table; without a known DB
	// in scope the allowed set is empty / does not invent identity.
	err := validateSQLTablesInScope("SELECT * FROM xxx.xxx", "", []string{"xxx.xxx"}, true)
	if err == nil {
		t.Fatalf("validateSQLTablesInScope() = nil, want reject bare dotted without known DB")
	}
}

func TestParseScopeTableIdentityDoesNotGuessBareDottedWithoutKnownDB(t *testing.T) {
	schema, name := parseScopeTableIdentity("xxx.xxx", "")
	if schema != "" || name != "xxx.xxx" {
		t.Fatalf("got {%q,%q}, want {,xxx.xxx} (no invent database)", schema, name)
	}
	schema, name = parseScopeTableIdentity("sales.orders", "", "sales", "support")
	if schema != "sales" || name != "orders" {
		t.Fatalf("got {%q,%q}, want {sales,orders}", schema, name)
	}
	schema, name = parseScopeTableIdentity("ffff_15.test.csv", "ffff_15")
	if schema != "ffff_15" || name != "test.csv" {
		t.Fatalf("got {%q,%q}, want {ffff_15,test.csv}", schema, name)
	}
}

func TestQuerySQLRequiresDBNameUnlessMultiDatabaseScope(t *testing.T) {
	svc := NewQuerySQL(Deps{
		SQLExecutor: &stubQuerySQLExecutor{},
	})
	_, err := svc.Execute(context.Background(), knowledge.QuerySQLRequest{
		Scope: knowledge.WorkspaceScope{WorkspaceID: "ws_1", Tables: []string{"orders"}},
		SQL:   "SELECT * FROM orders",
	})
	if err == nil || !strings.Contains(err.Error(), "db_name is required") {
		t.Fatalf("Execute() error = %v, want db_name required", err)
	}

	executor := &stubQuerySQLExecutor{
		results: []*knowledge.SQLExecutionResult{
			{Columns: []string{"count"}, Rows: [][]any{{1}}},
			{Columns: []string{"id"}, Rows: [][]any{{1}}},
		},
	}
	svc = NewQuerySQL(Deps{SQLExecutor: executor})
	resp, err := svc.Execute(context.Background(), knowledge.QuerySQLRequest{
		Scope: knowledge.WorkspaceScope{WorkspaceID: "ws_1", Tables: []string{"sales.orders", "support.tickets"}},
		SQL:   "SELECT * FROM sales.orders",
	})
	if err != nil {
		t.Fatalf("Execute() multi-database scope error = %v", err)
	}
	if resp == nil || resp.RowCount != 1 {
		t.Fatalf("Execute() response = %#v, want one row", resp)
	}
	if len(executor.dbNames) == 0 || executor.dbNames[0] != "" {
		t.Fatalf("Execute() dbNames = %#v, want empty selected database for multi-db scope", executor.dbNames)
	}
}

func TestQuerySQLPreviewMarksTruncated(t *testing.T) {
	executor := &stubQuerySQLExecutor{
		results: []*knowledge.SQLExecutionResult{
			{Columns: []string{"count"}, Rows: [][]any{{3}}},
			{Columns: []string{"name", "amount"}, Rows: [][]any{
				{"a", 10},
				{"b", 20},
			}},
		},
	}
	requirePhysicalTables := false
	svc := NewQuerySQL(Deps{
		SQLExecutor: executor,
		QuerySQLHooks: QuerySQLHooks{
			RequirePhysicalTables: &requirePhysicalTables,
		},
	})

	resp, err := svc.Execute(context.Background(), knowledge.QuerySQLRequest{
		Scope:   knowledge.WorkspaceScope{WorkspaceID: "ws_1", DBName: "db_1"},
		SQL:     "SELECT 'a' AS name, 10 AS amount",
		MaxRows: 2,
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if len(resp.Rows) != 2 {
		t.Fatalf("rows length = %d, want 2", len(resp.Rows))
	}
	if !resp.Truncated {
		t.Fatalf("Truncated = false, want true")
	}
	if resp.TotalCount != 3 {
		t.Fatalf("TotalCount = %d, want 3", resp.TotalCount)
	}
}
