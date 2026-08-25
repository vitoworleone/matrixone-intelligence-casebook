package service

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/matrixflow/moi-core/agent-tools/knowledge"
)

const DescribeSchemaMetadataSemanticContext = "semantic_context"

type SchemaReader interface {
	ListTables(ctx context.Context, scope knowledge.WorkspaceScope) ([]string, error)
	ListColumns(ctx context.Context, scope knowledge.WorkspaceScope, tableNames []string) ([]TableColumns, error)
	ListSemanticEntries(ctx context.Context, scope knowledge.WorkspaceScope) ([]knowledge.SemanticEntry, error)
	ReadSampleRows(ctx context.Context, scope knowledge.WorkspaceScope, tableName string, limit int) ([][]any, error)
}

type TableColumns struct {
	TableName string
	Columns   []knowledge.ColumnInfo
}

type DescribeSchemaSource interface {
	FetchDescribeSchema(ctx context.Context, req DescribeSchemaSourceRequest) (*DescribeSchemaSourceData, error)
}

type DescribeSchemaSourceRequest struct {
	WorkspaceID string
	CatalogID   int64
	DBName      string
	TableNames  []string
}

type DescribeSchemaSourceData struct {
	Tables map[string]DescribeSchemaSourceTable
	Raw    any
}

type DescribeSchemaSourceTable struct {
	TableID    int64
	DDL        string
	Columns    []knowledge.ColumnInfo
	SampleRows [][]any
	Comments   map[string]string
}

type DescribeSchemaCache interface {
	GetDescribeSchema(workspaceID string, catalogID int64, dbName string) (*DescribeSchemaSourceData, bool)
	SetDescribeSchema(workspaceID string, catalogID int64, dbName string, data *DescribeSchemaSourceData)
}

type DescribeSchemaHooks struct {
	IsPermissionDeniedError func(error) bool
}

type DescribeSchemaSemanticProvider interface {
	LoadDescribeSchemaSemantics(ctx context.Context, workspaceID string, tableNames []string) (*DescribeSchemaSemanticPayload, error)
}

type DescribeSchemaSemanticState interface {
	Available(ctx context.Context) bool
	HasSemantic(ctx context.Context) bool
	MergeSemanticPayload(ctx context.Context, payload *DescribeSchemaSemanticPayload)
	SemanticSnapshot(ctx context.Context) *DescribeSchemaSemanticContext
	MarkSeenIfAbsent(ctx context.Context, entryType, key string) bool
}

type DescribeSchemaSemanticPayload struct {
	Context DescribeSchemaSemanticContext
	Raw     any
}

type DescribeSchemaSemanticContext struct {
	Dimensions        []DescribeSchemaSemanticEntry `json:"dimensions,omitempty"`
	Facts             []DescribeSchemaSemanticEntry `json:"facts,omitempty"`
	Metrics           []DescribeSchemaSemanticEntry `json:"metrics,omitempty"`
	NamedFilters      []DescribeSchemaSemanticEntry `json:"named_filters,omitempty"`
	Glossary          []DescribeSchemaSemanticEntry `json:"glossary,omitempty"`
	ColumnPreferences []DescribeSchemaSemanticEntry `json:"column_preferences,omitempty"`
	PromptRules       []DescribeSchemaSemanticEntry `json:"prompt_rules,omitempty"`
}

type DescribeSchemaSemanticEntry struct {
	Kind            string   `json:"kind"`
	Key             string   `json:"key,omitempty"`
	Tables          []string `json:"tables,omitempty"`
	Column          string   `json:"column,omitempty"`
	Description     string   `json:"description,omitempty"`
	Expression      string   `json:"expr,omitempty"`
	Term            string   `json:"term,omitempty"`
	Definition      string   `json:"definition,omitempty"`
	Preferred       string   `json:"preferred,omitempty"`
	Deprecated      string   `json:"deprecated,omitempty"`
	Content         string   `json:"content,omitempty"`
	Source          string   `json:"source,omitempty"`
	ModelID         int64    `json:"model_id,omitempty"`
	InjectionStages []string `json:"injection_stages,omitempty"`
	Raw             any      `json:"-"`
}

func (c DescribeSchemaSemanticContext) IsEmpty() bool {
	return len(c.Dimensions) == 0 &&
		len(c.Facts) == 0 &&
		len(c.Metrics) == 0 &&
		len(c.NamedFilters) == 0 &&
		len(c.Glossary) == 0 &&
		len(c.ColumnPreferences) == 0 &&
		len(c.PromptRules) == 0
}

type describeSchemaService struct {
	reader           SchemaReader
	source           DescribeSchemaSource
	cache            DescribeSchemaCache
	accessExecutor   knowledge.SQLExecutor
	semanticProvider DescribeSchemaSemanticProvider
	semanticState    DescribeSchemaSemanticState
	hooks            DescribeSchemaHooks
}

func NewDescribeSchema(deps Deps) knowledge.DescribeSchema {
	return &describeSchemaService{
		reader:           deps.SchemaReader,
		source:           deps.DescribeSchemaSource,
		cache:            deps.DescribeSchemaCache,
		accessExecutor:   deps.DescribeSchemaAccessExecutor,
		semanticProvider: deps.DescribeSchemaSemanticProvider,
		semanticState:    deps.DescribeSchemaSemanticState,
		hooks:            deps.DescribeSchemaHooks,
	}
}

func (s *describeSchemaService) Execute(ctx context.Context, req knowledge.DescribeSchemaRequest) (*knowledge.DescribeSchemaResponse, error) {
	if s != nil && s.source != nil {
		return s.executeSource(ctx, req)
	}
	return s.executeReader(ctx, req)
}

func (s *describeSchemaService) executeSource(ctx context.Context, req knowledge.DescribeSchemaRequest) (*knowledge.DescribeSchemaResponse, error) {
	if s == nil || s.source == nil {
		return nil, fmt.Errorf("describe_schema: source not configured")
	}
	workspaceID := strings.TrimSpace(req.Scope.WorkspaceID)
	if workspaceID == "" {
		return nil, fmt.Errorf("describe_schema: workspace_id required")
	}
	dbName := strings.TrimSpace(req.Scope.DBName)
	if dbName == "" {
		return nil, fmt.Errorf("describe_schema: db_name required")
	}
	tableNames := dedupeNonEmptyCaseFold(req.TableNames)
	schemaData, err := s.fetchSource(ctx, workspaceID, req.Scope.CatalogID, dbName, tableNames)
	if err != nil {
		return nil, fmt.Errorf("describe_schema: %w", err)
	}
	if schemaData == nil || len(schemaData.Tables) == 0 {
		return &knowledge.DescribeSchemaResponse{Tables: nil}, nil
	}

	sampleCap := req.IncludeSamples
	if sampleCap < 0 {
		sampleCap = 0
	}
	want := make(map[string]struct{}, len(tableNames))
	for _, name := range tableNames {
		want[strings.ToLower(name)] = struct{}{}
	}
	selectedTables := make([]string, 0, len(schemaData.Tables))
	for name := range schemaData.Tables {
		if len(want) > 0 {
			if _, ok := want[strings.ToLower(strings.TrimSpace(name))]; !ok {
				continue
			}
		}
		selectedTables = append(selectedTables, name)
	}
	sort.Strings(selectedTables)

	useSemantics := !req.SuppressSemanticEntries && s.semanticState != nil && s.semanticState.Available(ctx)
	if useSemantics {
		s.ensureRunSemantics(ctx, workspaceID, selectedTables)
	}

	out := make([]knowledge.TableDescription, 0, len(selectedTables))
	for _, name := range selectedTables {
		tbl := schemaData.Tables[name]
		desc := knowledge.TableDescription{
			Name:     name,
			TableID:  tbl.TableID,
			DDL:      truncateDDLWithMarker(tbl.DDL, req.MaxDDLChars),
			Columns:  cloneColumns(tbl.Columns),
			Comments: cloneStringMap(tbl.Comments),
		}
		if s.accessExecutor != nil {
			queryable := true
			desc.Queryable = &queryable
			desc.Access = "queryable"
			access, err := s.probeTableAccess(ctx, dbName, name)
			if err != nil {
				return nil, err
			}
			if access == "permission_denied" {
				queryable = false
				desc.Queryable = &queryable
				desc.Access = "permission_denied"
			}
		}
		if sampleCap > 0 && len(tbl.SampleRows) > 0 {
			n := len(tbl.SampleRows)
			if n > sampleCap {
				n = sampleCap
			}
			desc.SampleRows = cloneRows(tbl.SampleRows[:n])
		}
		if useSemantics {
			sem := s.semanticsForTable(ctx, name)
			if !sem.IsEmpty() {
				desc.Metadata = map[string]any{DescribeSchemaMetadataSemanticContext: sem}
				desc.SemanticEntries = semanticEntriesFromDescribeSchemaContext(sem)
			}
		}
		out = append(out, desc)
	}
	return &knowledge.DescribeSchemaResponse{Tables: out}, nil
}

func (s *describeSchemaService) executeReader(ctx context.Context, req knowledge.DescribeSchemaRequest) (*knowledge.DescribeSchemaResponse, error) {
	if s == nil || s.reader == nil {
		return nil, fmt.Errorf("describe_schema: schema reader is not configured")
	}
	if req.IncludeSamples < 0 {
		return nil, fmt.Errorf("describe_schema: include_samples must be >= 0")
	}
	if req.MaxDDLChars < 0 {
		return nil, fmt.Errorf("describe_schema: max_ddl_chars must be >= 0")
	}
	scope := req.Scope
	if strings.TrimSpace(scope.WorkspaceID) == "" {
		return nil, fmt.Errorf("describe_schema: workspace_id is required")
	}
	// Table identity is database.table. Scope.DBName is optional when tables are
	// already fully qualified.
	if strings.TrimSpace(scope.DBName) == "" && !scopeTablesIncludeDatabase(scope.Tables) {
		return nil, fmt.Errorf("describe_schema: db_name is required")
	}
	selectedTables, err := s.resolveTables(ctx, scope, req.TableNames)
	if err != nil {
		return nil, err
	}
	columnItems, err := s.reader.ListColumns(ctx, scope, selectedTables)
	if err != nil {
		return nil, err
	}
	columnsByTable := columnsByTableKey(columnItems)
	var semanticsByTable map[string][]knowledge.SemanticEntry
	if !req.SuppressSemanticEntries {
		semanticEntries, err := s.reader.ListSemanticEntries(ctx, scope)
		if err != nil {
			return nil, err
		}
		semanticsByTable = semanticEntriesByTable(semanticEntries, selectedTables)
	}

	// Collect known DBs from the full scope (not only the selected subset) so a
	// single rewritten label like sales.orders still quotes as `sales`.`orders`
	// when the multi-db scope also owns support.tickets.
	knownDBs := scopeKnownDatabaseNames(append(append([]string(nil), scope.Tables...), selectedTables...), scope.DBName)
	out := make([]knowledge.TableDescription, 0, len(selectedTables))
	for _, table := range selectedTables {
		key := tableKey(table)
		columns := columnsByTable[key]
		if len(columns) == 0 {
			// Also match bare table names returned for single-db scopes.
			if schema, name := splitSQLTableRef(table, knownDBs...); schema != "" && name != "" {
				columns = columnsByTable[tableKey(name)]
			}
		}
		if len(columns) == 0 {
			continue
		}
		desc := knowledge.TableDescription{
			Name:            table,
			DDL:             truncateString(buildCreateTableDDL(table, columns, knownDBs...), req.MaxDDLChars),
			Columns:         cloneColumns(columns),
			SemanticEntries: append([]knowledge.SemanticEntry(nil), semanticsByTable[key]...),
		}
		if len(desc.SemanticEntries) == 0 {
			if schema, name := splitSQLTableRef(table, knownDBs...); schema != "" && name != "" {
				desc.SemanticEntries = append([]knowledge.SemanticEntry(nil), semanticsByTable[tableKey(name)]...)
			}
		}
		if req.IncludeSamples > 0 {
			rows, err := s.reader.ReadSampleRows(ctx, scope, table, req.IncludeSamples)
			if err != nil {
				return nil, err
			}
			desc.SampleRows = cloneRows(rows)
		}
		out = append(out, desc)
	}
	return &knowledge.DescribeSchemaResponse{Tables: out}, nil
}

func (s *describeSchemaService) fetchSource(ctx context.Context, workspaceID string, catalogID int64, dbName string, tableNames []string) (*DescribeSchemaSourceData, error) {
	if s.cache != nil && len(tableNames) == 0 {
		if cached, ok := s.cache.GetDescribeSchema(workspaceID, catalogID, dbName); ok {
			return cached, nil
		}
	}
	data, err := s.source.FetchDescribeSchema(ctx, DescribeSchemaSourceRequest{
		WorkspaceID: workspaceID,
		CatalogID:   catalogID,
		DBName:      dbName,
		TableNames:  append([]string(nil), tableNames...),
	})
	if err != nil {
		return nil, err
	}
	if s.cache != nil && len(tableNames) == 0 && data != nil {
		s.cache.SetDescribeSchema(workspaceID, catalogID, dbName, data)
	}
	return data, nil
}

func (s *describeSchemaService) probeTableAccess(ctx context.Context, dbName, table string) (string, error) {
	if s == nil || s.accessExecutor == nil {
		return "", nil
	}
	sqlText := fmt.Sprintf("SELECT 1 FROM %s LIMIT 0", quoteDescribeSchemaIdentifier(table))
	_, err := s.accessExecutor.ExecuteSQL(ctx, dbName, sqlText)
	if err == nil {
		return "queryable", nil
	}
	if s.hooks.IsPermissionDeniedError != nil && s.hooks.IsPermissionDeniedError(err) {
		return "permission_denied", nil
	}
	return "", fmt.Errorf("describe_schema: probe table access for %q: %w", table, err)
}

func (s *describeSchemaService) ensureRunSemantics(ctx context.Context, workspaceID string, selectedTables []string) {
	if s == nil || s.semanticState == nil || s.semanticProvider == nil || len(selectedTables) == 0 {
		return
	}
	if s.semanticState.HasSemantic(ctx) {
		return
	}
	payload, err := s.semanticProvider.LoadDescribeSchemaSemantics(ctx, workspaceID, selectedTables)
	if err != nil || payload == nil {
		return
	}
	s.semanticState.MergeSemanticPayload(ctx, payload)
}

func (s *describeSchemaService) semanticsForTable(ctx context.Context, table string) DescribeSchemaSemanticContext {
	if s == nil || s.semanticState == nil {
		return DescribeSchemaSemanticContext{}
	}
	sem := s.semanticState.SemanticSnapshot(ctx)
	if sem == nil || sem.IsEmpty() {
		return DescribeSchemaSemanticContext{}
	}
	tnorm := normalizeSemanticTableNameForMatch(table)
	if tnorm == "" {
		return DescribeSchemaSemanticContext{}
	}
	matches := func(entryTables []string) bool {
		for _, name := range entryTables {
			if normalizeSemanticTableNameForMatch(name) == tnorm {
				return true
			}
		}
		return false
	}
	out := DescribeSchemaSemanticContext{}
	for _, entry := range sem.Dimensions {
		if matches(entry.Tables) && s.semanticState.MarkSeenIfAbsent(ctx, "dimension", entry.Key) {
			out.Dimensions = append(out.Dimensions, entry)
		}
	}
	for _, entry := range sem.Facts {
		if matches(entry.Tables) && s.semanticState.MarkSeenIfAbsent(ctx, "fact", entry.Key) {
			out.Facts = append(out.Facts, entry)
		}
	}
	for _, entry := range sem.Metrics {
		if matches(entry.Tables) && s.semanticState.MarkSeenIfAbsent(ctx, "metric", entry.Key) {
			out.Metrics = append(out.Metrics, entry)
		}
	}
	for _, entry := range sem.NamedFilters {
		if matches(entry.Tables) && s.semanticState.MarkSeenIfAbsent(ctx, "named_filter", entry.Key) {
			out.NamedFilters = append(out.NamedFilters, entry)
		}
	}
	for _, entry := range sem.ColumnPreferences {
		if matches(entry.Tables) && s.semanticState.MarkSeenIfAbsent(ctx, "column_preference", entry.Key) {
			out.ColumnPreferences = append(out.ColumnPreferences, entry)
		}
	}
	for _, entry := range sem.Glossary {
		if matches(entry.Tables) && s.semanticState.MarkSeenIfAbsent(ctx, "glossary", entry.Key) {
			out.Glossary = append(out.Glossary, entry)
		}
	}
	for _, entry := range sem.PromptRules {
		if !matches(entry.Tables) {
			continue
		}
		if !promptRuleVisibleToAgent(entry) {
			continue
		}
		if s.semanticState.MarkSeenIfAbsent(ctx, "prompt_rule", promptRuleSeenKey(entry)) {
			out.PromptRules = append(out.PromptRules, entry)
		}
	}
	return out
}

func (s *describeSchemaService) resolveTables(ctx context.Context, scope knowledge.WorkspaceScope, requestedTables []string) ([]string, error) {
	scopeTables := compactStrings(scope.Tables)
	requestedTables = compactStrings(requestedTables)
	if len(scopeTables) > 0 && len(requestedTables) > 0 {
		resolved, err := resolveDescribeSchemaTablesInScope(requestedTables, scopeTables, scope.DBName)
		if err != nil {
			return nil, err
		}
		// Return qualified identities so ListColumns / DDL always see database.table.
		return resolved, nil
	}
	if len(requestedTables) > 0 {
		return qualifyDescribeSchemaTables(requestedTables, scope.DBName), nil
	}
	if len(scopeTables) > 0 {
		return scopeTables, nil
	}
	return s.reader.ListTables(ctx, scope)
}

// resolveDescribeSchemaTablesInScope validates requested tables against scope
// and rewrites unique bare names to their qualified database.table identity.
func resolveDescribeSchemaTablesInScope(requestedTables, scopeTables []string, dbName string) ([]string, error) {
	dbName = strings.TrimSpace(dbName)
	knownDBs := scopeKnownDatabaseNames(scopeTables, dbName)
	allowed := scopeTableIdentitySet(scopeTables, dbName)
	// Bare request names map uniquely to one allowed identity when possible.
	bareToQualified := make(map[string]string, len(allowed))
	allowedDisplay := make(map[string]string, len(allowed))
	for _, scopeTable := range scopeTables {
		schema, name := parseScopeTableIdentity(scopeTable, dbName, knownDBs...)
		if schema == "" || name == "" {
			continue
		}
		identity := qualifiedTableKey(schema, name)
		allowedDisplay[identity] = displayQualifiedTable(schema, name, scopeTable)
		key := tableKey(name)
		if existing, ok := bareToQualified[key]; ok && existing != identity {
			bareToQualified[key] = ""
			continue
		}
		bareToQualified[key] = identity
	}
	out := make([]string, 0, len(requestedTables))
	for _, table := range requestedTables {
		schema, name := parseScopeTableIdentity(table, dbName, knownDBs...)
		if name == "" {
			return nil, fmt.Errorf("describe_schema: table %q is outside selected knowledge table scope", table)
		}
		if schema != "" {
			identity := qualifiedTableKey(schema, name)
			if display, ok := allowedDisplay[identity]; ok {
				out = append(out, display)
				continue
			}
			if _, ok := allowed[identity]; ok {
				out = append(out, displayQualifiedTable(schema, name, table))
				continue
			}
		}
		// Accept bare request when it uniquely matches one allowed database.table,
		// and rewrite to the qualified identity for downstream readers.
		if mapped := bareToQualified[tableKey(name)]; mapped != "" {
			if display, ok := allowedDisplay[mapped]; ok {
				out = append(out, display)
			} else {
				out = append(out, mapped)
			}
			continue
		}
		return nil, fmt.Errorf("describe_schema: table %q is outside selected knowledge table scope", table)
	}
	return out, nil
}

func qualifyDescribeSchemaTables(tables []string, dbName string) []string {
	dbName = strings.TrimSpace(dbName)
	if dbName == "" {
		// No database: leave labels unchanged; do not invent schema from dots.
		return tables
	}
	out := make([]string, 0, len(tables))
	for _, table := range tables {
		schema, name := parseScopeTableIdentity(table, dbName)
		if name == "" {
			out = append(out, table)
			continue
		}
		if schema == "" {
			schema = dbName
		}
		out = append(out, displayQualifiedTable(schema, name, table))
	}
	return out
}

func displayQualifiedTable(schema, name, fallback string) string {
	schema = strings.TrimSpace(schema)
	name = strings.TrimSpace(name)
	if schema == "" || name == "" {
		return strings.TrimSpace(fallback)
	}
	return schema + "." + name
}

func columnsByTableKey(items []TableColumns) map[string][]knowledge.ColumnInfo {
	if len(items) == 0 {
		return nil
	}
	out := make(map[string][]knowledge.ColumnInfo, len(items))
	for _, item := range items {
		key := tableKey(item.TableName)
		if key == "" {
			continue
		}
		out[key] = append(out[key], item.Columns...)
	}
	return out
}

func buildCreateTableDDL(tableName string, columns []knowledge.ColumnInfo, knownDBs ...string) string {
	lines := make([]string, 0, len(columns))
	for _, column := range columns {
		line := "  " + quoteIdentifier(column.Name) + " " + column.Type
		if !column.Nullable {
			line += " NOT NULL"
		}
		if column.PrimaryKey || column.IsPrimaryKey {
			line += " PRIMARY KEY"
		}
		if strings.TrimSpace(column.Comment) != "" {
			line += " COMMENT " + sqlStringLiteral(column.Comment)
		}
		lines = append(lines, line)
	}
	return "CREATE TABLE " + quoteTableName(tableName, knownDBs...) + " (\n" + strings.Join(lines, ",\n") + "\n)"
}

// quoteTableName quotes database.table as `database`.`table` only when the
// left segment matches a known database name. Without a known database, the
// whole string is one identifier — bare physical names like xxx.xxx must not
// become `xxx`.`xxx` just because they contain a dot.
func quoteTableName(tableName string, knownDBs ...string) string {
	trimmed := strings.TrimSpace(tableName)
	if trimmed == "" {
		return quoteIdentifier(tableName)
	}
	if schema, name := splitSQLTableRef(trimmed, knownDBs...); schema != "" && name != "" {
		return quoteIdentifier(schema) + "." + quoteIdentifier(name)
	}
	return quoteIdentifier(trimmed)
}

func semanticEntriesByTable(entries []knowledge.SemanticEntry, selectedTables []string) map[string][]knowledge.SemanticEntry {
	if len(entries) == 0 {
		return nil
	}
	// selected holds full selected labels (possibly database.table). Storage
	// persists entry.Tables as bare names (NormalizeEntryTables strips db
	// prefix), so also index bare table keys for matching.
	selected := tableSet(selectedTables)
	knownDBs := scopeKnownDatabaseNames(selectedTables, "")
	// When selected is a single multi-db label, still need known DB for bare
	// extraction — prefer longest known prefix among selected labels by also
	// treating each selected identity's schema as known when multi-db decode
	// already succeeded via 2+ lefts, or when default is empty but labels are
	// system-encoded db.table from Resolve (schema free of '.').
	if len(knownDBs) == 0 {
		// Resolve always emits database.table. For semantic bare matching we can
		// safely use left segments of selected labels when every selected entry
		// has a left segment free of '.' — only when there is at least one such
		// label. A lone bare physical xxx.xxx still must not invent a DB for
		// quote paths that pass knownDBs separately; here we only extract bare
		// table names for entry matching.
		for _, table := range selectedTables {
			schema, name := decodeResolvedScopeTableLabel(table)
			if schema == "" || name == "" {
				continue
			}
			knownDBs = append(knownDBs, schema)
		}
	}
	bareToSelected := make(map[string][]string, len(selectedTables))
	for _, table := range selectedTables {
		key := tableKey(table)
		if key == "" {
			continue
		}
		_, name := parseScopeTableIdentity(table, "", knownDBs...)
		if name == "" {
			// Fallback: system-encoded Resolve label first-dot when left has no '.'.
			if _, bareName := decodeResolvedScopeTableLabel(table); bareName != "" {
				name = bareName
			} else {
				name = table
			}
		}
		bare := tableKey(name)
		if bare == "" {
			continue
		}
		bareToSelected[bare] = append(bareToSelected[bare], key)
	}
	out := make(map[string][]knowledge.SemanticEntry)
	for _, entry := range entries {
		targetTables := entry.Tables
		if len(targetTables) == 0 {
			targetTables = selectedTables
		}
		for _, table := range targetTables {
			key := tableKey(table)
			if key == "" {
				continue
			}
			if len(selected) == 0 {
				out[key] = append(out[key], entry)
				continue
			}
			// Exact match on selected label (qualified or bare).
			if _, ok := selected[key]; ok {
				out[key] = append(out[key], entry)
				continue
			}
			// Stored bare name matches a selected database.table via bare key.
			// Only attach when the bare name uniquely maps to one qualified table.
			// Cross-db same bare name (sales.orders + support.orders) with an
			// unqualified entry must not fan out to every match — that would
			// cross-link semantics across databases.
			_, entryName := parseScopeTableIdentity(table, "", knownDBs...)
			if entryName == "" {
				entryName = table
			}
			bare := tableKey(entryName)
			matches := bareToSelected[bare]
			if len(matches) != 1 {
				continue
			}
			selectedKey := matches[0]
			out[selectedKey] = append(out[selectedKey], entry)
			// Keep bare key only for the unique mapping so legacy bare lookups
			// still work when there is no ambiguity.
			out[bare] = append(out[bare], entry)
		}
	}
	return out
}

func semanticEntriesFromDescribeSchemaContext(ctx DescribeSchemaSemanticContext) []knowledge.SemanticEntry {
	if ctx.IsEmpty() {
		return nil
	}
	entries := make([]knowledge.SemanticEntry, 0)
	seen := map[string]struct{}{}
	add := func(items []DescribeSchemaSemanticEntry) {
		for _, item := range items {
			kind := strings.TrimSpace(item.Kind)
			key := strings.TrimSpace(item.Key)
			if key == "" && kind == "prompt_rule" {
				key = promptRuleSeenKey(item)
			}
			if kind == "" && key == "" {
				continue
			}
			dedupeKey := fmt.Sprintf("%s:%d:%s", kind, item.ModelID, key)
			if _, ok := seen[dedupeKey]; ok {
				continue
			}
			seen[dedupeKey] = struct{}{}
			entries = append(entries, knowledge.SemanticEntry{
				ModelID: item.ModelID,
				Kind:    kind,
				KeyName: key,
				Tables:  append([]string(nil), item.Tables...),
			})
		}
	}
	add(ctx.Dimensions)
	add(ctx.Facts)
	add(ctx.Metrics)
	add(ctx.NamedFilters)
	add(ctx.Glossary)
	add(ctx.ColumnPreferences)
	add(ctx.PromptRules)
	return entries
}

func quoteIdentifier(value string) string {
	return "`" + strings.ReplaceAll(value, "`", "``") + "`"
}

func sqlStringLiteral(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

func truncateString(value string, maxChars int) string {
	if maxChars <= 0 {
		return value
	}
	runes := []rune(value)
	if len(runes) <= maxChars {
		return value
	}
	return string(runes[:maxChars])
}

func truncateDDLWithMarker(ddl string, maxChars int) string {
	if maxChars <= 0 || ddl == "" {
		return ddl
	}
	if len(ddl) <= maxChars {
		return ddl
	}
	runes := []rune(ddl)
	if len(runes) <= maxChars {
		return ddl
	}
	return string(runes[:maxChars]) + "… (truncated)"
}

func quoteDescribeSchemaIdentifier(name string) string {
	name = strings.TrimSpace(name)
	name = strings.Trim(name, "`\"")
	if name == "" {
		return ""
	}
	parts := strings.Split(name, ".")
	quoted := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(strings.Trim(part, "`\""))
		if part == "" {
			continue
		}
		quoted = append(quoted, "`"+strings.ReplaceAll(part, "`", "``")+"`")
	}
	return strings.Join(quoted, ".")
}

func promptRuleVisibleToAgent(rule DescribeSchemaSemanticEntry) bool {
	if len(rule.InjectionStages) == 0 {
		return true
	}
	for _, stage := range rule.InjectionStages {
		if _, ok := describeSchemaAgentPromptStages[strings.ToLower(strings.TrimSpace(stage))]; ok {
			return true
		}
	}
	return false
}

var describeSchemaAgentPromptStages = map[string]struct{}{
	"planner_policy":    {},
	"sql_generation":    {},
	"sql_followup":      {},
	"sql_regenerate":    {},
	"sql_decomposition": {},
	"executor_rule":     {},
}

func promptRuleSeenKey(rule DescribeSchemaSemanticEntry) string {
	if key := strings.TrimSpace(rule.Source); key != "" {
		return key
	}
	return strings.TrimSpace(rule.Content)
}

func normalizeSemanticTableNameForMatch(raw string) string {
	name := strings.Trim(strings.TrimSpace(strings.ToLower(raw)), "`\"")
	if dot := strings.LastIndex(name, "."); dot >= 0 {
		name = strings.Trim(name[dot+1:], "`\"")
	}
	return name
}

func dedupeNonEmptyCaseFold(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, value := range in {
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

func cloneColumns(in []knowledge.ColumnInfo) []knowledge.ColumnInfo {
	if len(in) == 0 {
		return nil
	}
	return append([]knowledge.ColumnInfo(nil), in...)
}

func cloneRows(in [][]any) [][]any {
	if len(in) == 0 {
		return nil
	}
	out := make([][]any, len(in))
	for i, row := range in {
		out[i] = append([]any(nil), row...)
	}
	return out
}

func cloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
