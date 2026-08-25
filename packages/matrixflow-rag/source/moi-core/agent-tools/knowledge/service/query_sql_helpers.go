package service

import (
	"fmt"
	"strings"

	"github.com/pingcap/tidb/pkg/parser/ast"
)

func validateReadOnlySQL(sqlText string) error {
	validationSQL := querySQLValidationBody(sqlText)
	if validationSQL == "" {
		return fmt.Errorf("query_sql: sql is required")
	}
	stmt, err := parseSingleMySQLStatement(validationSQL)
	if err != nil {
		return fmt.Errorf("query_sql: parse sql: %w%s", err, mysqlParseGuidance(err))
	}
	switch stmt.(type) {
	case *ast.SelectStmt, *ast.SetOprStmt:
		return nil
	default:
		return fmt.Errorf("query_sql: only SELECT or WITH SELECT statements are allowed")
	}
}

// mysqlParseGuidance appends model-facing MySQL dialect hints for common parse
// failures. query_sql validates with a MySQL-compatible parser (TiDB); MatrixOne
// may accept some looser forms that MySQL rejects, so the model must emit strict
// MySQL 8 syntax.
func mysqlParseGuidance(err error) string {
	if err == nil {
		return ""
	}
	msg := strings.ToLower(err.Error())
	// Reserved keyword used as unquoted identifier/alias is a frequent model
	// failure mode (e.g. AS current_time). Point the model at the fix.
	if strings.Contains(msg, "current_time") ||
		strings.Contains(msg, "current_date") ||
		strings.Contains(msg, "current_timestamp") ||
		strings.Contains(msg, "localtime") ||
		strings.Contains(msg, "localtimestamp") {
		return "; SQL must be valid MySQL 8 syntax. Reserved words used as identifiers/aliases must be backtick-quoted (for example AS `current_time`) or renamed to a non-reserved alias such as now_value"
	}
	if strings.Contains(msg, "near") {
		return "; SQL must be valid MySQL 8 syntax. Quote reserved-word identifiers with backticks and avoid MatrixOne-only dialect extensions"
	}
	return ""
}

func querySQLValidationBody(sqlText string) string {
	_, body := splitLeadingQuerySQLRewriteHint(trimSQLForSubquery(sqlText))
	return strings.TrimSpace(body)
}

func validateSQLTablesInScope(sqlText, dbName string, scopeTables []string, requirePhysicalTables bool) error {
	refs := collectPhysicalTableRefs(sqlText)
	if len(refs) == 0 && requirePhysicalTables && !isPureScalarSelectSQL(sqlText) {
		return fmt.Errorf("query_sql: at least one physical table is required (pure scalar SELECT without FROM is allowed; table queries must reference in-scope tables)")
	}
	scopeTables = compactStrings(scopeTables)
	if len(scopeTables) == 0 {
		return nil
	}
	dbName = strings.TrimSpace(dbName)
	// Allowed set is always database.table identities.
	allowed := scopeTableIdentitySet(scopeTables, dbName)
	for _, ref := range refs {
		table := ref.display()
		schema := strings.TrimSpace(ref.schema)
		name := strings.TrimSpace(ref.name)
		if name == "" {
			return fmt.Errorf("query_sql: table %q is outside selected knowledge table scope", table)
		}
		if schema == "" {
			schema = dbName
		}
		if schema == "" {
			return fmt.Errorf("query_sql: table %q must be referenced as database.table", table)
		}
		if _, ok := allowed[qualifiedTableKey(schema, name)]; !ok {
			return fmt.Errorf("query_sql: table %q is outside selected knowledge table scope", table)
		}
	}
	return nil
}

func scopeTablesIncludeDatabase(values []string) bool {
	// True when at least one scope label decodes to an explicit database via a
	// known default or multi-db known prefixes — never by "has a dot" alone.
	defaultDB := ""
	knownDBs := scopeKnownDatabaseNames(values, defaultDB)
	for _, value := range values {
		schema, name := parseScopeTableIdentity(value, defaultDB, knownDBs...)
		if schema != "" && name != "" {
			return true
		}
	}
	return false
}

// scopeTableIdentitySet builds the set of database.table identities from scope
// table strings. Table names may themselves contain '.' (e.g. test.csv); those
// stay intact when the entry is bare or when the database prefix is known.
// Without a known database, dotted bare names are skipped (not guessed).
func scopeTableIdentitySet(values []string, defaultDB string) map[string]struct{} {
	out := make(map[string]struct{}, len(values))
	defaultDB = strings.TrimSpace(defaultDB)
	knownDBs := scopeKnownDatabaseNames(values, defaultDB)
	for _, value := range values {
		schema, name := parseScopeTableIdentity(value, defaultDB, knownDBs...)
		if schema == "" || name == "" {
			continue
		}
		out[qualifiedTableKey(schema, name)] = struct{}{}
	}
	return out
}

// scopeKnownDatabaseNames collects database names that may prefix scope labels.
//
// Sources:
//  1. defaultDB (Scope.DBName / single-db default)
//  2. left segments of multi-db system labels only when there are 2+ distinct
//     left segments (true multi-db Resolve emit). A lone dotted string such as
//     bare physical "xxx.xxx" must NOT self-bootstrap into database "xxx".
//
// Database names never contain '.'.
func scopeKnownDatabaseNames(values []string, defaultDB string) []string {
	out := make([]string, 0, len(values)+1)
	seen := make(map[string]struct{}, len(values)+1)
	add := func(db string) {
		db = strings.TrimSpace(db)
		if db == "" || strings.Contains(db, ".") {
			return
		}
		key := strings.ToLower(db)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		out = append(out, db)
	}
	add(defaultDB)

	lefts := make([]string, 0, len(values))
	leftSeen := make(map[string]struct{}, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		trimmed = strings.Trim(trimmed, "`\"")
		if trimmed == "" {
			continue
		}
		idx := strings.Index(trimmed, ".")
		if idx <= 0 || idx >= len(trimmed)-1 {
			continue
		}
		left := strings.TrimSpace(trimmed[:idx])
		if left == "" || strings.Contains(left, ".") {
			continue
		}
		key := strings.ToLower(left)
		if _, ok := leftSeen[key]; ok {
			continue
		}
		leftSeen[key] = struct{}{}
		lefts = append(lefts, left)
	}
	// Multi-db system labels always span 2+ databases after Resolve. A single
	// left segment with no defaultDB is ambiguous (could be bare physical
	// xxx.xxx) — do not invent a database from it.
	if len(lefts) >= 2 {
		for _, left := range lefts {
			add(left)
		}
	}
	return out
}

func qualifiedTableKey(schema, name string) string {
	return strings.ToLower(strings.TrimSpace(schema)) + "." + strings.ToLower(strings.TrimSpace(name))
}

// decodeResolvedScopeTableLabel decodes a system-encoded Resolve label
// database.table where the database name never contains '.'. This is NOT for
// untrusted bare physical names: callers must only use it on labels already
// produced as database.table (scope emit / selected identities).
func decodeResolvedScopeTableLabel(value string) (string, string) {
	trimmed := strings.TrimSpace(value)
	trimmed = strings.Trim(trimmed, "`\"")
	if trimmed == "" {
		return "", ""
	}
	idx := strings.Index(trimmed, ".")
	if idx <= 0 || idx >= len(trimmed)-1 {
		return "", trimmed
	}
	schema := strings.TrimSpace(trimmed[:idx])
	name := strings.Trim(strings.TrimSpace(trimmed[idx+1:]), "`\"")
	if schema == "" || name == "" || strings.Contains(schema, ".") {
		return "", trimmed
	}
	return schema, name
}

// parseScopeTableIdentity resolves a scope table string into (database, table).
//
// Rules (no "has dot ⇒ qualified" guessing):
//  1. When defaultDB is set: strip defaultDB. prefix, else whole string is bare
//     table name (may contain '.') paired with defaultDB.
//  2. When defaultDB is empty: only strip a prefix that matches a known database
//     name (knownDBs). Otherwise the whole string is bare — schema stays empty.
//
// Multi-db system labels (sales.orders, ffff_15.test.csv) decode when those
// databases are known (defaultDB or 2+ multi-db left segments). Bare physical
// xxx.xxx without a known DB is never split into database xxx + table xxx.
func parseScopeTableIdentity(value, defaultDB string, knownDBs ...string) (string, string) {
	trimmed := strings.TrimSpace(value)
	trimmed = strings.Trim(trimmed, "`\"")
	if trimmed == "" {
		return "", ""
	}
	defaultDB = strings.TrimSpace(defaultDB)
	if defaultDB != "" {
		prefix := defaultDB + "."
		if len(trimmed) > len(prefix) && strings.EqualFold(trimmed[:len(prefix)], prefix) {
			return defaultDB, strings.Trim(strings.TrimSpace(trimmed[len(prefix):]), "`\"")
		}
		// Bare table name (may contain '.'); pair with the default database.
		return defaultDB, trimmed
	}
	// No default database: only accept an explicit known database prefix.
	// Prefer the longest matching known DB so nested names stay stable.
	bestDB := ""
	bestName := ""
	for _, db := range knownDBs {
		db = strings.TrimSpace(db)
		if db == "" || strings.Contains(db, ".") {
			continue
		}
		prefix := db + "."
		if len(trimmed) > len(prefix) && strings.EqualFold(trimmed[:len(prefix)], prefix) {
			name := strings.Trim(strings.TrimSpace(trimmed[len(prefix):]), "`\"")
			if name == "" {
				continue
			}
			if len(db) > len(bestDB) {
				bestDB = db
				bestName = name
			}
		}
	}
	if bestDB != "" {
		return bestDB, bestName
	}
	// Unknown / bare (including dotted physical names): do not invent a database.
	return "", trimmed
}

func tableSet(values []string) map[string]struct{} {
	out := make(map[string]struct{}, len(values))
	for _, value := range values {
		key := tableKey(value)
		if key != "" {
			out[key] = struct{}{}
		}
	}
	return out
}

func tableKey(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, "`\"")
	return strings.ToLower(value)
}

// splitSQLTableRef splits a table reference only when the left segment matches
// a known database name. Without known DBs this returns bare (schema empty).
func splitSQLTableRef(value string, knownDBs ...string) (string, string) {
	return parseScopeTableIdentity(value, "", knownDBs...)
}

func compactStrings(values []string) []string {
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		key := strings.ToLower(value)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, value)
	}
	return out
}
