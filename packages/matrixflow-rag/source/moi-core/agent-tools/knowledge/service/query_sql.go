package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/matrixflow/moi-core/agent-tools/knowledge"
)

const (
	defaultQuerySQLMaxRows = 200
)

type QuerySQLHooks struct {
	NormalizeSQL           func(sql string) (string, bool, error)
	TableNames             func(sql string) []string
	TableRefs              func(ctx context.Context, tableNames []string, dbName string) ([]knowledge.TableRef, error)
	DiagnoseExecutionError func(ctx context.Context, req QuerySQLErrorDiagnosticRequest) error
	DefaultRows            int
	RequirePhysicalTables  *bool
}

type QuerySQLErrorDiagnosticRequest struct {
	WorkspaceID         string
	DBName              string
	SQL                 string
	Err                 error
	Phase               string
	SchemaGateValidated bool
}

type querySQLService struct {
	sql   knowledge.SQLExecutor
	hooks QuerySQLHooks
}

func NewQuerySQL(deps Deps) knowledge.QuerySQL {
	return &querySQLService{
		sql:   deps.SQLExecutor,
		hooks: deps.QuerySQLHooks,
	}
}

func (s *querySQLService) Execute(ctx context.Context, req knowledge.QuerySQLRequest) (*knowledge.QuerySQLResponse, error) {
	if s == nil || s.sql == nil {
		return nil, fmt.Errorf("query_sql: sql executor is not configured")
	}
	dbName := strings.TrimSpace(req.Scope.DBName)
	sqlText := strings.TrimSpace(req.SQL)
	if sqlText == "" {
		return nil, fmt.Errorf("query_sql: sql is required")
	}
	if s.hooks.NormalizeSQL != nil {
		normalized, changed, err := s.hooks.NormalizeSQL(sqlText)
		if err != nil {
			return nil, fmt.Errorf("query_sql: normalize sql: %w", err)
		}
		if changed {
			sqlText = normalized
		}
	}
	if err := validateReadOnlySQL(sqlText); err != nil {
		return nil, err
	}
	// Table identity is database.table. Scope.DBName is optional when every
	// scope table is already fully qualified.
	if dbName == "" && !scopeTablesIncludeDatabase(req.Scope.Tables) {
		return nil, fmt.Errorf("query_sql: db_name is required")
	}
	if err := validateSQLTablesInScope(sqlText, dbName, req.Scope.Tables, s.requirePhysicalTables()); err != nil {
		return nil, err
	}
	maxRows := req.MaxRows
	if maxRows <= 0 {
		maxRows = s.defaultRows()
	}

	tableNames := s.tableNames(sqlText)
	ctx = knowledge.ContextWithScope(ctx, req.Scope)
	tables, err := s.tableRefs(ctx, tableNames, dbName)
	if err != nil {
		return nil, err
	}
	exec, totalRows, err := s.executeSQLForRequest(ctx, req.Scope.WorkspaceID, dbName, sqlText, maxRows, req.SchemaGateValidated)
	if err != nil {
		return nil, err
	}
	if exec == nil {
		return nil, fmt.Errorf("query_sql: executor returned nil result")
	}

	totalCount := reconcileSQLTotalCount(exec, totalRows)
	result := knowledge.QuerySQLResponse{
		SQL:                sqlText,
		DBName:             dbName,
		Columns:            append([]string(nil), exec.Columns...),
		Rows:               append([][]any(nil), exec.Rows...),
		RowCount:           totalCount,
		TotalCount:         totalCount,
		MaxRows:            maxRows,
		Truncated:          totalCount > len(exec.Rows),
		TableNames:         append([]string(nil), tableNames...),
		Tables:             append([]knowledge.TableRef(nil), tables...),
		SemanticKeysUsed:   compactStrings(req.SemanticClaims),
		AppliedConstraints: append([]knowledge.SQLAppliedConstraint(nil), req.AppliedConstraints...),
	}
	if maxRows > 0 && len(result.Rows) > maxRows {
		result.Rows = result.Rows[:maxRows]
		result.Truncated = true
	}
	if result.RowCount == 0 {
		result.RowCount = result.TotalCount
	}
	if result.TotalCount == 0 {
		result.TotalCount = result.RowCount
	}
	if result.MaxRows == 0 {
		result.MaxRows = maxRows
	}
	return &result, nil
}

func (s *querySQLService) executeSQLForRequest(
	ctx context.Context,
	workspaceID string,
	dbName string,
	sqlText string,
	maxRows int,
	schemaGateValidated bool,
) (*knowledge.SQLExecutionResult, int, error) {
	totalCount, err := s.executeSQLCount(ctx, workspaceID, dbName, sqlText, schemaGateValidated)
	if err != nil {
		return nil, 0, err
	}
	exec, err := s.sql.ExecuteSQL(ctx, dbName, wrapQuerySQLPreview(sqlText, maxRows))
	if err != nil {
		return nil, 0, s.wrapSQLError(ctx, workspaceID, dbName, sqlText, err, "execute", schemaGateValidated)
	}
	return exec, totalCount, nil
}

func (s *querySQLService) executeSQLCount(ctx context.Context, workspaceID, dbName, sqlText string, schemaGateValidated bool) (int, error) {
	exec, err := s.sql.ExecuteSQL(ctx, dbName, wrapQuerySQLCount(sqlText))
	if err != nil {
		return 0, s.wrapSQLError(ctx, workspaceID, dbName, sqlText, err, "count", schemaGateValidated)
	}
	totalCount, err := extractQuerySQLCount(exec)
	if err != nil {
		return 0, fmt.Errorf("query_sql count: %w", err)
	}
	return totalCount, nil
}

func (s *querySQLService) wrapSQLError(ctx context.Context, workspaceID, dbName, sqlText string, err error, phase string, schemaGateValidated bool) error {
	if err == nil {
		return nil
	}
	out := err
	if s.hooks.DiagnoseExecutionError != nil {
		diagnosed := s.hooks.DiagnoseExecutionError(ctx, QuerySQLErrorDiagnosticRequest{
			WorkspaceID:         workspaceID,
			DBName:              dbName,
			SQL:                 sqlText,
			Err:                 err,
			Phase:               phase,
			SchemaGateValidated: schemaGateValidated,
		})
		if diagnosed != nil {
			out = diagnosed
		}
	}
	if phase == "count" {
		return fmt.Errorf("query_sql count: %w", out)
	}
	return fmt.Errorf("query_sql: %w", out)
}

func (s *querySQLService) tableNames(sqlText string) []string {
	if s.hooks.TableNames != nil {
		return s.hooks.TableNames(sqlText)
	}
	return ExtractPhysicalTableNames(sqlText)
}

func (s *querySQLService) tableRefs(ctx context.Context, tableNames []string, dbName string) ([]knowledge.TableRef, error) {
	if s.hooks.TableRefs != nil {
		return s.hooks.TableRefs(ctx, tableNames, dbName)
	}
	out := make([]knowledge.TableRef, 0, len(tableNames))
	defaultDB := strings.TrimSpace(dbName)
	knownDBs := scopeKnownDatabaseNames(tableNames, defaultDB)
	for _, table := range tableNames {
		if strings.TrimSpace(table) == "" {
			continue
		}
		// Prefer defaultDB / known-DB-aware parse so bare dotted names (test.csv)
		// pair with the scope database instead of inventing a schema from dots.
		schema, name := parseScopeTableIdentity(table, defaultDB, knownDBs...)
		if schema == "" {
			schema = defaultDB
		}
		if name == "" {
			name = table
		}
		// Without an explicit database, do not emit a guessed identity.
		if schema == "" || name == "" {
			continue
		}
		out = append(out, knowledge.TableRef{DBName: schema, Name: name})
	}
	return out, nil
}

func (s *querySQLService) defaultRows() int {
	if s.hooks.DefaultRows > 0 {
		return s.hooks.DefaultRows
	}
	return defaultQuerySQLMaxRows
}

func (s *querySQLService) requirePhysicalTables() bool {
	if s.hooks.RequirePhysicalTables != nil {
		return *s.hooks.RequirePhysicalTables
	}
	return true
}

func reconcileSQLTotalCount(exec *knowledge.SQLExecutionResult, totalCount int) int {
	if totalCount > 0 {
		return totalCount
	}
	if exec != nil && exec.TotalCount > 0 {
		return exec.TotalCount
	}
	if exec != nil {
		return len(exec.Rows)
	}
	return 0
}

func wrapQuerySQLCount(sqlText string) string {
	hint, body := splitLeadingQuerySQLRewriteHint(trimSQLForSubquery(sqlText))
	prefix := ""
	if hint != "" {
		prefix = hint + "\n"
	}
	return fmt.Sprintf(
		"%sSELECT COUNT(1) AS __query_sql_total_count FROM (\n%s\n) AS query_sql_count_source",
		prefix,
		body,
	)
}

func wrapQuerySQLPreview(sqlText string, maxRows int) string {
	if maxRows <= 0 {
		maxRows = defaultQuerySQLMaxRows
	}
	hint, body := splitLeadingQuerySQLRewriteHint(trimSQLForSubquery(sqlText))
	prefix := ""
	if hint != "" {
		prefix = hint + "\n"
	}
	return fmt.Sprintf(
		"%sSELECT * FROM (\n%s\n) AS query_sql_preview_source LIMIT %d",
		prefix,
		body,
		maxRows,
	)
}

func trimSQLForSubquery(sqlText string) string {
	trimmed := strings.TrimSpace(sqlText)
	for strings.HasSuffix(trimmed, ";") {
		trimmed = strings.TrimSpace(strings.TrimSuffix(trimmed, ";"))
	}
	return trimmed
}

func splitLeadingQuerySQLRewriteHint(sqlText string) (string, string) {
	trimmed := strings.TrimSpace(sqlText)
	if !strings.HasPrefix(trimmed, "/*+") {
		return "", trimmed
	}
	end := strings.Index(trimmed, "*/")
	if end < 0 {
		return "", trimmed
	}
	hint := strings.TrimSpace(trimmed[:end+2])
	if !strings.Contains(strings.ToLower(hint), `"rewrites"`) {
		return "", trimmed
	}
	return hint, strings.TrimSpace(trimmed[end+2:])
}

func extractQuerySQLCount(exec *knowledge.SQLExecutionResult) (int, error) {
	if exec == nil {
		return 0, errors.New("executor returned nil count result")
	}
	if len(exec.Rows) == 0 || len(exec.Rows[0]) == 0 {
		return 0, errors.New("executor returned empty count result")
	}
	return querySQLCountCellToInt(exec.Rows[0][0])
}

func querySQLCountCellToInt(value any) (int, error) {
	switch typed := value.(type) {
	case int:
		if typed < 0 {
			return 0, fmt.Errorf("negative count %d", typed)
		}
		return typed, nil
	case int8:
		if typed < 0 {
			return 0, fmt.Errorf("negative count %d", typed)
		}
		return int(typed), nil
	case int16:
		if typed < 0 {
			return 0, fmt.Errorf("negative count %d", typed)
		}
		return int(typed), nil
	case int32:
		if typed < 0 {
			return 0, fmt.Errorf("negative count %d", typed)
		}
		return int(typed), nil
	case int64:
		if typed < 0 || typed > int64(maxQuerySQLCountInt()) {
			return 0, fmt.Errorf("count value out of range %d", typed)
		}
		return int(typed), nil
	case uint:
		return int(typed), nil
	case uint8:
		return int(typed), nil
	case uint16:
		return int(typed), nil
	case uint32:
		return int(typed), nil
	case uint64:
		if typed > uint64(maxQuerySQLCountInt()) {
			return 0, fmt.Errorf("count value out of range %d", typed)
		}
		return int(typed), nil
	case float32:
		return querySQLCountFloatToInt(float64(typed))
	case float64:
		return querySQLCountFloatToInt(typed)
	case string:
		return querySQLCountStringToInt(typed)
	case []byte:
		return querySQLCountStringToInt(string(typed))
	default:
		return 0, fmt.Errorf("unsupported count value type %T", value)
	}
}

func querySQLCountFloatToInt(value float64) (int, error) {
	if value < 0 || value > float64(maxQuerySQLCountInt()) || value != math.Trunc(value) {
		return 0, fmt.Errorf("invalid count value %v", value)
	}
	return int(value), nil
}

func querySQLCountStringToInt(value string) (int, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0, errors.New("empty count value")
	}
	parsed, err := strconv.ParseInt(trimmed, 10, 0)
	if err != nil {
		return 0, fmt.Errorf("parse count %q: %w", value, err)
	}
	if parsed < 0 {
		return 0, fmt.Errorf("negative count %d", parsed)
	}
	return int(parsed), nil
}

func maxQuerySQLCountInt() int {
	return int(^uint(0) >> 1)
}
