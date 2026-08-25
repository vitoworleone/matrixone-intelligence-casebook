package agents

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/matrixflow/moi-core/agent-tools/knowledge"
	knowledgeservice "github.com/matrixflow/moi-core/agent-tools/knowledge/service"
	embeddingpkg "github.com/matrixflow/moi-core/catalog/pkg/embedding"
	embeddingadapter "github.com/matrixflow/moi-core/catalog/pkg/embedding/adapter"
	"github.com/matrixflow/moi-core/catalog/pkg/service/storage/tenant"
)

var errPlatformKnowledgeVectorIndexNotReady = errors.New("knowledge base vector index is not ready; wait for document parsing to complete and retry")

type PlatformKnowledgeSemanticModelStore interface {
	GetModel(ctx context.Context, workspaceID string, modelID int64) (*tenant.SemanticModelRecord, error)
}

type PlatformKnowledgeVectorIndexResolver interface {
	ResolveVectorIndex(ctx context.Context, req PlatformKnowledgeVectorIndexResolveRequest) (*PlatformKnowledgeVectorIndexResolveResult, error)
}

type PlatformKnowledgeImageIndexResolver interface {
	ResolveImageIndex(ctx context.Context, req PlatformKnowledgeImageIndexResolveRequest) (*PlatformKnowledgeImageIndexResolveResult, error)
}

type PlatformKnowledgeLegacyIndexResolver interface {
	ResolveLegacyIndexVersions(ctx context.Context, req PlatformKnowledgeLegacyIndexResolveRequest) (map[string]knowledge.RAGIndexVersionConstraint, error)
	ResolveLegacyIndexFileIDs(ctx context.Context, req PlatformKnowledgeLegacyIndexResolveRequest) ([]string, error)
}

type PlatformKnowledgeSourceGovernanceStore interface {
	ListSourceGovernance(ctx context.Context, workspaceID string, modelID int64, fileIDs []string) ([]PlatformKnowledgeSourceGovernanceRecord, error)
}

type PlatformKnowledgeTableSourceGovernanceStore interface {
	ListTableSourceGovernance(ctx context.Context, workspaceID string, modelID int64, tables []PlatformKnowledgeSemanticModelTableRef) ([]PlatformKnowledgeTableSourceGovernanceRecord, error)
}

type PlatformKnowledgeSourceGovernanceRecord struct {
	SourceRowID             string
	FileID                  string
	Status                  string
	Enabled                 bool
	Expired                 bool
	EffectiveEnabled        bool
	ForceEnabledAfterExpiry bool
	Tags                    []string
	SegmentVersionID        string
	IndexVersion            int64
	IndexVersionValid       bool
}

type PlatformKnowledgeTableSourceGovernanceRecord struct {
	SourceRowID             string
	DBName                  string
	TableName               string
	SourceDBName            string
	SourceTableName         string
	SourceTableID           string
	KBTableID               string
	Status                  string
	Enabled                 bool
	Expired                 bool
	EffectiveEnabled        bool
	ForceEnabledAfterExpiry bool
}

type PlatformKnowledgeSemanticModelTableRef struct {
	DBName    string
	TableName string
}

type PlatformKnowledgeVectorIndexResolveRequest struct {
	WorkspaceID     string
	UserID          string
	SemanticModelID int64
	VectorTable     string
	EmbeddingModel  string
	FileIDs         []string
	VolumeIDs       []string
	Metadata        map[string]string
}

type PlatformKnowledgeVectorIndexResolveResult struct {
	VectorTable    string
	EmbeddingModel string
	Metadata       map[string]string
}

type PlatformKnowledgeImageIndexResolveRequest struct {
	WorkspaceID         string
	UserID              string
	SemanticModelID     int64
	ImageVectorTable    string
	ImageEmbeddingModel string
	FileIDs             []string
	VolumeIDs           []string
}

type PlatformKnowledgeImageIndexResolveResult struct {
	ImageVectorTable        string
	ImageEmbeddingModel     string
	ImageEmbeddingBackendID string
	ImageEmbeddingDimension int
	ImagePreprocessVersion  string
	ImageDistanceMetric     string
	Metadata                map[string]string
}

type PlatformKnowledgeLegacyIndexResolveRequest struct {
	WorkspaceID string
	VectorTable string
	FileIDs     []string
}

type catalogKnowledgeSQLExecutor struct {
	connPool     tenant.ConnectionPool
	userConnPool tenant.UserConnectionPool
}

func (e *catalogKnowledgeSQLExecutor) ExecuteSQL(ctx context.Context, dbName string, sqlText string) (*knowledge.SQLExecutionResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	sqlText = strings.TrimSpace(sqlText)
	if sqlText == "" {
		return nil, fmt.Errorf("catalog knowledge sql executor: sql is required")
	}
	scope := knowledge.ScopeFromContext(ctx)
	db, err := catalogKnowledgeUserDB(ctx, e.connPool, e.userConnPool, scope.WorkspaceID, scope.UserID)
	if err != nil {
		return nil, err
	}
	conn, err := db.Conn(ctx)
	if err != nil {
		return nil, fmt.Errorf("catalog knowledge sql executor: acquire connection: %w", err)
	}
	defer conn.Close()

	if dbName = strings.TrimSpace(dbName); dbName != "" {
		useSQL, err := platformKnowledgeUseDatabaseSQL(dbName)
		if err != nil {
			return nil, err
		}
		if _, err := conn.ExecContext(ctx, useSQL); err != nil {
			return nil, fmt.Errorf("catalog knowledge sql executor: use database %q: %w", dbName, err)
		}
	}

	rows, err := conn.QueryContext(ctx, sqlText)
	if err != nil {
		return nil, fmt.Errorf("catalog knowledge sql executor: query failed: %w", err)
	}
	defer rows.Close()

	result, err := platformKnowledgeScanExecutionRows(rows)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (e *catalogKnowledgeSQLExecutor) ExecuteMutation(ctx context.Context, dbName string, sqlText string, args ...any) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if strings.TrimSpace(sqlText) == "" {
		return 0, fmt.Errorf("catalog knowledge sql executor: sql is required")
	}
	scope := knowledge.ScopeFromContext(ctx)
	db, err := catalogKnowledgeUserDB(ctx, e.connPool, e.userConnPool, scope.WorkspaceID, scope.UserID)
	if err != nil {
		return 0, err
	}
	conn, err := db.Conn(ctx)
	if err != nil {
		return 0, fmt.Errorf("catalog knowledge sql executor: acquire connection: %w", err)
	}
	defer conn.Close()

	useSQL, err := platformKnowledgeUseDatabaseSQL(strings.TrimSpace(dbName))
	if err != nil {
		return 0, err
	}
	if _, err := conn.ExecContext(ctx, useSQL); err != nil {
		return 0, fmt.Errorf("catalog knowledge sql executor: use database %q: %w", dbName, err)
	}
	result, err := conn.ExecContext(ctx, sqlText, args...)
	if err != nil {
		return 0, fmt.Errorf("catalog knowledge sql executor: mutation failed: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("catalog knowledge sql executor: mutation rows affected: %w", err)
	}
	return rowsAffected, nil
}

type catalogKnowledgeSchemaReader struct {
	connPool     tenant.ConnectionPool
	userConnPool tenant.UserConnectionPool
}

func (r *catalogKnowledgeSchemaReader) ListTables(ctx context.Context, scope knowledge.WorkspaceScope) ([]string, error) {
	if tables := platformKnowledgeCompactStrings(scope.Tables); len(tables) > 0 {
		return tables, nil
	}
	db, err := r.userDB(ctx, scope)
	if err != nil {
		return nil, err
	}
	dbName := strings.TrimSpace(scope.DBName)
	if dbName == "" {
		return nil, fmt.Errorf("describe_schema: db_name is required")
	}
	return queryKnowledgeDatabaseTables(ctx, db, dbName)
}

func (r *catalogKnowledgeSchemaReader) ListColumns(ctx context.Context, scope knowledge.WorkspaceScope, tableNames []string) ([]knowledgeservice.TableColumns, error) {
	db, err := r.userDB(ctx, scope)
	if err != nil {
		return nil, err
	}
	tableNames = platformKnowledgeCompactStrings(tableNames)
	if len(tableNames) == 0 {
		return nil, nil
	}
	// Group by database using full scope.Tables for knownDB collection — not
	// only the selected subset. Agent describe often requests one table from a
	// multi-db scope (DBName=""); selected-only knownDB has a single left
	// segment and would refuse to decode sales.orders.
	byDB, order, err := platformKnowledgeGroupTablesByDatabase(scope.Tables, tableNames, scope.DBName)
	if err != nil {
		return nil, err
	}
	out := make([]knowledgeservice.TableColumns, 0, len(tableNames))
	for _, dbName := range order {
		requested := byDB[dbName]
		bareNames := make([]string, 0, len(requested))
		for _, table := range requested {
			_, name := platformKnowledgeParseTableIdentity(table, dbName)
			if name == "" {
				name = strings.TrimSpace(table)
			}
			bareNames = append(bareNames, name)
		}
		items, err := queryKnowledgeInformationSchemaColumns(ctx, db, dbName, bareNames)
		if err != nil {
			return nil, err
		}
		// Remap bare TABLE_NAME results back to the caller's qualified labels.
		byBare := make(map[string][]knowledge.ColumnInfo, len(items))
		for _, item := range items {
			byBare[strings.ToLower(strings.TrimSpace(item.TableName))] = item.Columns
		}
		for _, table := range requested {
			_, name := platformKnowledgeParseTableIdentity(table, dbName)
			if name == "" {
				name = strings.TrimSpace(table)
			}
			columns := byBare[strings.ToLower(name)]
			if len(columns) == 0 {
				continue
			}
			out = append(out, knowledgeservice.TableColumns{
				TableName: table,
				Columns:   columns,
			})
		}
	}
	return out, nil
}

// platformKnowledgeGroupTablesByDatabase groups selected table labels by
// database for multi-db describe/query paths.
//
// knownDB collection uses scopeTables (full Resolve scope) union selected
// tableNames, matching ReadSampleRows / describe_schema DDL. Using only the
// selected subset breaks multi-db scopes when the agent describes one table:
// len(lefts)==1 never seeds knownDBs under empty defaultDB.
func platformKnowledgeGroupTablesByDatabase(scopeTables, tableNames []string, defaultDB string) (map[string][]string, []string, error) {
	tableNames = platformKnowledgeCompactStrings(tableNames)
	if len(tableNames) == 0 {
		return nil, nil, nil
	}
	defaultDB = strings.TrimSpace(defaultDB)
	knownSource := append(append([]string(nil), platformKnowledgeCompactStrings(scopeTables)...), tableNames...)
	knownDBs := platformKnowledgeScopeKnownDatabaseNames(knownSource, defaultDB)
	byDB := make(map[string][]string, len(tableNames))
	order := make([]string, 0, len(tableNames))
	for _, table := range tableNames {
		schema, name := platformKnowledgeParseTableIdentity(table, defaultDB, knownDBs...)
		if name == "" {
			name = strings.TrimSpace(table)
		}
		dbName := schema
		if dbName == "" {
			dbName = defaultDB
		}
		if dbName == "" || name == "" {
			return nil, nil, fmt.Errorf("describe_schema: table %q is missing database qualification", table)
		}
		if _, ok := byDB[dbName]; !ok {
			order = append(order, dbName)
		}
		// Keep the caller's table label (possibly db.table) for response matching.
		byDB[dbName] = append(byDB[dbName], table)
	}
	return byDB, order, nil
}

func (r *catalogKnowledgeSchemaReader) ListSemanticEntries(ctx context.Context, scope knowledge.WorkspaceScope) ([]knowledge.SemanticEntry, error) {
	db, err := r.userDB(ctx, scope)
	if err != nil {
		return nil, err
	}
	return queryKnowledgeSemanticEntries(ctx, db, scope.SemanticModelIDs)
}

func (r *catalogKnowledgeSchemaReader) ReadSampleRows(ctx context.Context, scope knowledge.WorkspaceScope, tableName string, limit int) ([][]any, error) {
	db, err := r.userDB(ctx, scope)
	if err != nil {
		return nil, err
	}
	defaultDB := strings.TrimSpace(scope.DBName)
	// Same knownDB source as ListColumns: full scope plus the requested label.
	knownSource := append(append([]string(nil), platformKnowledgeCompactStrings(scope.Tables)...), tableName)
	knownDBs := platformKnowledgeScopeKnownDatabaseNames(knownSource, defaultDB)
	schema, name := platformKnowledgeParseTableIdentity(tableName, defaultDB, knownDBs...)
	if name == "" {
		name = strings.TrimSpace(tableName)
	}
	dbName := schema
	if dbName == "" {
		dbName = defaultDB
	}
	if dbName == "" || name == "" {
		return nil, fmt.Errorf("describe_schema: table %q is missing database qualification", tableName)
	}
	return queryKnowledgeSampleRows(ctx, db, dbName, name, limit)
}

func (r *catalogKnowledgeSchemaReader) userDB(ctx context.Context, scope knowledge.WorkspaceScope) (*sql.DB, error) {
	if r == nil {
		return nil, fmt.Errorf("describe_schema: catalog schema reader is not configured")
	}
	workspaceID := strings.TrimSpace(scope.WorkspaceID)
	if workspaceID == "" {
		return nil, fmt.Errorf("describe_schema: workspace_id is required")
	}
	// Multi-db scopes leave Scope.DBName empty; connection is tenant-scoped and
	// queries use fully qualified identifiers or information_schema filters.
	db, err := catalogKnowledgeUserDB(ctx, r.connPool, r.userConnPool, workspaceID, scope.UserID)
	if err != nil {
		return nil, fmt.Errorf("describe_schema: get user database connection: %w", err)
	}
	return db, nil
}

// platformKnowledgeParseTableIdentity resolves a scope/SQL table string into
// (database, table). Encoding rule matches agent-tools parseScopeTableIdentity:
// when a database is known we store "database.table", where the table part may
// contain '.' (strip only against a known/default database prefix).
//
// Without defaultDB, only prefixes listed in knownDBs are accepted. Bare
// physical names like xxx.xxx are never first-dot guessed into a database.
// Callers that match turn/binding strings against model tables should use
// platformKnowledgeMatchIncomingTableHint instead.
func platformKnowledgeParseTableIdentity(value, defaultDB string, knownDBs ...string) (string, string) {
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
	// No known database: do not invent one from dots.
	return "", strings.Trim(trimmed, "`\"")
}

// platformKnowledgeIncomingTableHint is the resolved form of one scope.Tables
// entry used for selection filtering against model tables.
type platformKnowledgeIncomingTableHint struct {
	DBName    string
	TableName string
	// Bare is true when the original string was a bare (possibly dotted) name
	// rather than an explicit database.table identity. Bare hints may match
	// any model table with the same table name; qualified hints are exact only.
	Bare bool
}

// platformKnowledgeMatchIncomingTableHint matches turn/binding table strings
// against candidate semantic-model tables only. Match order:
//  1. exact candidate identity db.table (table may contain '.')
//  2. bare candidate TableName match (whole string equals TableName)
//
// No fallback guessing: unmatched strings are dropped (empty hint). This keeps
// multi-db scopes from inventing identities and from first-dot splitting
// dotted bare names that collide with another model database (MF-10).
func platformKnowledgeMatchIncomingTableHint(value string, candidates []PlatformKnowledgeSemanticModelTableRef) platformKnowledgeIncomingTableHint {
	trimmed := strings.TrimSpace(value)
	trimmed = strings.Trim(trimmed, "`\"")
	if trimmed == "" {
		return platformKnowledgeIncomingTableHint{}
	}

	// Exact identity: full "db.table" string against each candidate. Table names
	// may contain '.', so never first-dot split the incoming string for this step.
	for _, candidate := range candidates {
		dbName := strings.TrimSpace(candidate.DBName)
		tableName := strings.TrimSpace(candidate.TableName)
		if dbName == "" || tableName == "" {
			continue
		}
		if strings.EqualFold(dbName+"."+tableName, trimmed) {
			return platformKnowledgeIncomingTableHint{
				DBName:    dbName,
				TableName: tableName,
				Bare:      false,
			}
		}
	}

	// Bare table-name match (whole incoming string equals candidate.TableName).
	// Multiple matches stay bare so filter can select every same-named table
	// (frontend multi-db seed). Unique match still records as bare selection.
	var bareMatches []PlatformKnowledgeSemanticModelTableRef
	for _, candidate := range candidates {
		tableName := strings.TrimSpace(candidate.TableName)
		if tableName == "" {
			continue
		}
		if strings.EqualFold(tableName, trimmed) {
			bareMatches = append(bareMatches, candidate)
		}
	}
	if len(bareMatches) >= 1 {
		return platformKnowledgeIncomingTableHint{
			TableName: trimmed,
			Bare:      true,
		}
	}
	return platformKnowledgeIncomingTableHint{}
}

type catalogKnowledgeVisualBackend struct {
	configCache     *embeddingpkg.ConfigCache
	router          *embeddingpkg.Router
	adapterRegistry embeddingadapter.Registry
	connPool        tenant.ConnectionPool
	fileStorage     tenant.FileStorage
	fileService     PlatformKnowledgeFileService
}

func (b *catalogKnowledgeVisualBackend) ReadImageFile(ctx context.Context, scope knowledge.WorkspaceScope, fileID string) ([]byte, string, error) {
	raw, err := readPlatformFileBytes(ctx, b.connPool, b.fileStorage, b.fileService, scope.WorkspaceID, fileID)
	if err != nil {
		return nil, "", err
	}
	return raw, http.DetectContentType(raw), nil
}

func (b *catalogKnowledgeVisualBackend) CreateImageEmbedding(ctx context.Context, workspaceID string, req knowledgeservice.VisualImageEmbeddingRequest) ([]float64, map[string]any, error) {
	if b.configCache == nil || b.router == nil || b.adapterRegistry == nil {
		return nil, nil, fmt.Errorf("embedding router is not configured")
	}
	if len(req.Raw) == 0 {
		return nil, nil, fmt.Errorf("image bytes are empty")
	}
	mimeType := strings.TrimSpace(strings.Split(req.MimeType, ";")[0])
	if mimeType == "" || mimeType == "application/octet-stream" {
		mimeType = http.DetectContentType(req.Raw)
	}
	mimeType = strings.TrimSpace(strings.Split(mimeType, ";")[0])
	if !strings.HasPrefix(mimeType, "image/") {
		return nil, nil, fmt.Errorf("image content type must be image/*, got %q", mimeType)
	}
	// Align with go-worker document_visual.index.image: TaaS multimodal embedding requires
	// type=embedding_multimodal and input[].content[].image_url data URL (not legacy images[]).
	// Prefer index-bound backend_id for both router selection and request body.
	// Catalog routing is numeric-only: non-empty non-integer ids fail closed
	// (do not silently fall back to model-only selection).
	backendID, err := parseImageEmbeddingBackendID(req.BackendID)
	if err != nil {
		return nil, nil, err
	}
	imageURL := "data:" + mimeType + ";base64," + base64.StdEncoding.EncodeToString(req.Raw)
	body := map[string]any{
		"model": req.Model,
		"type":  "embedding_multimodal",
		"input": []map[string]any{
			{
				"content": []map[string]any{
					{
						"type": "image_url",
						"image_url": map[string]string{
							"url": imageURL,
						},
					},
				},
			},
		},
		"embedding_mode":     "fusion",
		"output_cardinality": "one_per_input",
		"encoding_format":    "float",
	}
	if backendID != 0 {
		body["backend_id"] = backendID
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal image embedding request: %w", err)
	}
	workspaceCfg, err := b.configCache.GetOrLoad(ctx, workspaceID)
	if err != nil {
		return nil, nil, fmt.Errorf("load embedding config for workspace %s: %w", workspaceID, err)
	}
	backend, endpoint := b.router.SelectBackendAndEndpointWithModelAndBackendID(workspaceCfg, req.Model, backendID)
	if backend == nil || endpoint == nil {
		if backendID != 0 {
			return nil, nil, fmt.Errorf("no available image embedding backend for workspace %s model %s backend_id %d", workspaceID, req.Model, backendID)
		}
		return nil, nil, fmt.Errorf("no available image embedding backend for workspace %s model %s", workspaceID, req.Model)
	}
	adapter, ok := b.adapterRegistry.Get(backend.Type)
	if !ok {
		return nil, nil, fmt.Errorf("embedding adapter not found for backend type %s", backend.Type.String())
	}
	resp, err := adapter.Embeddings(ctx, backend, endpoint, &embeddingadapter.EmbeddingRequest{Body: payload})
	if err != nil {
		return nil, nil, fmt.Errorf("create image embedding model=%s: %w", req.Model, err)
	}
	var parsed struct {
		Data []struct {
			Embedding []float64 `json:"embedding"`
		} `json:"data"`
		Metadata map[string]any `json:"metadata,omitempty"`
	}
	if err := json.Unmarshal(resp.Body, &parsed); err != nil {
		return nil, nil, fmt.Errorf("decode image embedding response: %w", err)
	}
	if len(parsed.Data) != 1 {
		return nil, nil, fmt.Errorf("image embedding response count mismatch: got %d want 1", len(parsed.Data))
	}
	embedding := append([]float64(nil), parsed.Data[0].Embedding...)
	if len(embedding) == 0 {
		return nil, nil, fmt.Errorf("image embedding is empty")
	}
	return embedding, parsed.Metadata, nil
}

// parseImageEmbeddingBackendID returns the numeric catalog backend id.
// Empty means "unbound" (model-only selection). Non-empty non-integer is an error.
func parseImageEmbeddingBackendID(raw string) (int64, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, nil
	}
	backendID, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("image embedding backend_id %q must be a numeric catalog backend id", value)
	}
	if backendID == 0 {
		return 0, fmt.Errorf("image embedding backend_id must be a non-zero numeric catalog backend id")
	}
	return backendID, nil
}

func (b *catalogKnowledgeVisualBackend) ResolveVisualScopeFileIDs(ctx context.Context, scope knowledge.WorkspaceScope) ([]string, bool, error) {
	fileIDs := append([]string(nil), scope.FileIDs...)
	volumeIDs := []string{scope.VolumeID}
	for _, source := range scope.RAGSources {
		fileIDs = append(fileIDs, source.FileIDs...)
		volumeIDs = append(volumeIDs, source.VolumeID)
	}
	fileIDs = platformKnowledgeCompactStrings(fileIDs)
	volumeIDs = platformKnowledgeCompactStrings(volumeIDs)
	if len(fileIDs) == 0 && len(volumeIDs) == 0 {
		return nil, false, nil
	}
	if len(volumeIDs) == 0 {
		return fileIDs, true, nil
	}
	volumeFileIDs, err := b.fileIDsForVolumes(ctx, strings.TrimSpace(scope.WorkspaceID), volumeIDs)
	if err != nil {
		return nil, true, err
	}
	return platformKnowledgeCompactStrings(append(fileIDs, volumeFileIDs...)), true, nil
}

func (b *catalogKnowledgeVisualBackend) fileIDsForVolumes(ctx context.Context, workspaceID string, volumeIDs []string) ([]string, error) {
	if strings.TrimSpace(workspaceID) == "" {
		return nil, fmt.Errorf("visual search: workspace_id is required to resolve volume scope")
	}
	if b.connPool == nil {
		return nil, fmt.Errorf("visual search: connection pool is required to resolve volume scope")
	}
	args, err := platformKnowledgeParseVolumeIDArgs(volumeIDs)
	if err != nil {
		return nil, err
	}
	if len(args) == 0 {
		return nil, nil
	}
	tm, err := b.connPool.GetTransactionManager(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("get transaction manager workspace_id=%s: %w", workspaceID, err)
	}
	fileCtx := tenant.WithTransactionManager(ctx, tm)
	placeholders := strings.TrimRight(strings.Repeat("?,", len(args)), ",")
	var out []string
	if err := tm.RunInTx(fileCtx, func(txCtx context.Context) error {
		rows, err := tm.Executor(txCtx).QueryContext(txCtx, fmt.Sprintf(`SELECT DISTINCT file_id FROM volume_files WHERE volume_id IN (%s)`, placeholders), args...)
		if err != nil {
			return fmt.Errorf("query visual scope volume files: %w", err)
		}
		defer rows.Close()
		out, err = platformKnowledgeScanStringRows(rows)
		return err
	}); err != nil {
		return nil, err
	}
	return out, nil
}

func queryKnowledgeDatabaseTables(ctx context.Context, db *sql.DB, dbName string) ([]string, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT TABLE_NAME FROM information_schema.tables WHERE TABLE_SCHEMA = ? AND TABLE_TYPE = 'BASE TABLE' ORDER BY TABLE_NAME`,
		dbName,
	)
	if err != nil {
		return nil, fmt.Errorf("describe_schema: query database tables: %w", err)
	}
	defer rows.Close()
	return platformKnowledgeScanStringRows(rows)
}

func queryKnowledgeInformationSchemaColumns(ctx context.Context, db *sql.DB, dbName string, tableNames []string) ([]knowledgeservice.TableColumns, error) {
	tableNames = platformKnowledgeCompactStrings(tableNames)
	if len(tableNames) == 0 {
		return nil, nil
	}
	placeholders := strings.TrimRight(strings.Repeat("?,", len(tableNames)), ",")
	args := append([]any{dbName}, platformKnowledgeStringsToAny(tableNames)...)
	rows, err := db.QueryContext(ctx,
		fmt.Sprintf(`SELECT TABLE_NAME, COLUMN_NAME, COLUMN_TYPE, COLUMN_COMMENT, IS_NULLABLE, COLUMN_KEY
FROM information_schema.columns
WHERE TABLE_SCHEMA = ? AND TABLE_NAME IN (%s)
ORDER BY TABLE_NAME, ORDINAL_POSITION`, placeholders),
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("describe_schema: query information_schema.columns: %w", err)
	}
	defer rows.Close()
	columnsByTable := make(map[string][]knowledge.ColumnInfo, len(tableNames))
	tableNamesByKey := make(map[string]string, len(tableNames))
	for rows.Next() {
		var tableName, columnName, columnType, columnComment, isNullable, columnKey string
		if err := rows.Scan(&tableName, &columnName, &columnType, &columnComment, &isNullable, &columnKey); err != nil {
			return nil, fmt.Errorf("describe_schema: scan information_schema.columns: %w", err)
		}
		key := strings.ToLower(tableName)
		tableNamesByKey[key] = tableName
		columnsByTable[key] = append(columnsByTable[key], knowledge.ColumnInfo{
			Name:       columnName,
			Type:       columnType,
			Comment:    columnComment,
			Nullable:   strings.EqualFold(strings.TrimSpace(isNullable), "YES"),
			PrimaryKey: strings.EqualFold(strings.TrimSpace(columnKey), "PRI"),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("describe_schema: iterate information_schema.columns: %w", err)
	}
	out := make([]knowledgeservice.TableColumns, 0, len(columnsByTable))
	for _, tableName := range tableNames {
		key := strings.ToLower(tableName)
		columns := columnsByTable[key]
		if len(columns) == 0 {
			continue
		}
		actualTableName := tableNamesByKey[key]
		if actualTableName == "" {
			actualTableName = tableName
		}
		out = append(out, knowledgeservice.TableColumns{
			TableName: actualTableName,
			Columns:   columns,
		})
	}
	return out, nil
}

func queryKnowledgeSemanticEntries(ctx context.Context, db *sql.DB, modelIDs []int64) ([]knowledge.SemanticEntry, error) {
	modelIDs = platformKnowledgeCompactInt64s(modelIDs)
	if len(modelIDs) == 0 {
		return nil, nil
	}
	placeholders := strings.TrimRight(strings.Repeat("?,", len(modelIDs)), ",")
	rows, err := db.QueryContext(ctx,
		fmt.Sprintf(`SELECT model_id, kind, key_name, tables, spec
FROM semantic_entries
WHERE model_id IN (%s)
ORDER BY model_id ASC, id ASC`, placeholders),
		platformKnowledgeInt64sToAny(modelIDs)...,
	)
	if err != nil {
		return nil, fmt.Errorf("describe_schema: query semantic_entries: %w", err)
	}
	defer rows.Close()

	out := make([]knowledge.SemanticEntry, 0)
	for rows.Next() {
		var (
			modelID   int64
			kind      string
			keyName   string
			tablesRaw sql.NullString
			specRaw   []byte
		)
		if err := rows.Scan(&modelID, &kind, &keyName, &tablesRaw, &specRaw); err != nil {
			return nil, fmt.Errorf("describe_schema: scan semantic_entries: %w", err)
		}
		entryTables := platformKnowledgeDecodeStringArray(tablesRaw.String)
		var spec json.RawMessage
		if len(specRaw) > 0 {
			spec = append(json.RawMessage(nil), specRaw...)
		}
		entry := knowledge.SemanticEntry{
			ModelID: modelID,
			Kind:    kind,
			KeyName: keyName,
			Tables:  entryTables,
			Spec:    spec,
		}
		out = append(out, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("describe_schema: iterate semantic_entries: %w", err)
	}
	return out, nil
}

func queryKnowledgeSampleRows(ctx context.Context, db *sql.DB, dbName, tableName string, limit int) ([][]any, error) {
	if limit <= 0 {
		return nil, nil
	}
	sqlText := fmt.Sprintf("SELECT * FROM %s.%s LIMIT %d",
		platformKnowledgeQuoteIdentifier(dbName),
		platformKnowledgeQuoteIdentifier(tableName),
		limit,
	)
	rows, err := db.QueryContext(ctx, sqlText)
	if err != nil {
		return nil, fmt.Errorf("describe_schema: sample table %q: %w", tableName, err)
	}
	defer rows.Close()
	result, err := platformKnowledgeScanExecutionRows(rows)
	if err != nil {
		return nil, fmt.Errorf("describe_schema: scan sample table %q: %w", tableName, err)
	}
	return result.Rows, nil
}

func platformKnowledgeDecodeStringArray(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "null" {
		return nil
	}
	var values []string
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return nil
	}
	return platformKnowledgeCompactStrings(values)
}

type catalogKnowledgeEmbeddingClient struct {
	configCache     *embeddingpkg.ConfigCache
	router          *embeddingpkg.Router
	adapterRegistry embeddingadapter.Registry
}

func (e *catalogKnowledgeEmbeddingClient) CreateEmbedding(ctx context.Context, workspaceID, model string, texts []string) ([][]float64, error) {
	if e == nil || e.configCache == nil || e.router == nil || e.adapterRegistry == nil {
		return nil, fmt.Errorf("catalog knowledge embedding client is not configured")
	}
	cfg, err := e.configCache.GetOrLoad(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("load embedding config for workspace %s: %w", workspaceID, err)
	}
	backend, endpoint := e.router.SelectBackendAndEndpoint(cfg, model)
	if backend == nil || endpoint == nil {
		return nil, fmt.Errorf("no available embedding backend for workspace %s model %s", workspaceID, model)
	}
	adapter, ok := e.adapterRegistry.Get(backend.Type)
	if !ok {
		return nil, fmt.Errorf("embedding adapter not found for backend type %s", backend.Type.String())
	}
	body, err := json.Marshal(catalogKnowledgeEmbeddingRequest{Model: model, Input: texts})
	if err != nil {
		return nil, fmt.Errorf("marshal embedding request: %w", err)
	}
	resp, err := adapter.Embeddings(ctx, backend, endpoint, &embeddingadapter.EmbeddingRequest{Body: body})
	if err != nil {
		return nil, fmt.Errorf("embedding call failed: %w", err)
	}
	var parsed catalogKnowledgeEmbeddingResponse
	if err := json.Unmarshal(resp.Body, &parsed); err != nil {
		preview := string(resp.Body)
		if len(preview) > 200 {
			preview = preview[:200]
		}
		return nil, fmt.Errorf("parse embedding response (body_len=%d body_prefix=%q): %w", len(resp.Body), preview, err)
	}
	result := make([][]float64, len(parsed.Data))
	for i, d := range parsed.Data {
		result[i] = d.Embedding
	}
	return result, nil
}

type catalogKnowledgeEmbeddingRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type catalogKnowledgeEmbeddingResponse struct {
	Data []struct {
		Embedding []float64 `json:"embedding"`
	} `json:"data"`
}

type catalogKnowledgeSemanticModelStore struct {
	pool    tenant.ConnectionPool
	storage tenant.SemanticModelStorage
}

func newCatalogKnowledgeSemanticModelStore(pool tenant.ConnectionPool) *catalogKnowledgeSemanticModelStore {
	return &catalogKnowledgeSemanticModelStore{
		pool:    pool,
		storage: tenant.NewTenantStorage(""),
	}
}

func (s *catalogKnowledgeSemanticModelStore) GetModel(ctx context.Context, workspaceID string, modelID int64) (*tenant.SemanticModelRecord, error) {
	if s == nil || s.pool == nil || s.storage == nil {
		return nil, fmt.Errorf("catalog knowledge semantic model store is not configured")
	}
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return nil, fmt.Errorf("workspace_id is required for semantic model lookup")
	}
	tm, err := s.pool.GetTransactionManager(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	ctx = tenant.WithTransactionManager(ctx, tm)
	var model *tenant.SemanticModelRecord
	if err := tm.RunInTx(ctx, func(txCtx context.Context) error {
		model, err = s.storage.GetSemanticModel(txCtx, modelID)
		return err
	}); err != nil {
		return nil, err
	}
	return model, nil
}

type catalogKnowledgeSourceGovernanceStore struct {
	pool tenant.ConnectionPool
}

func newCatalogKnowledgeSourceGovernanceStore(pool tenant.ConnectionPool) *catalogKnowledgeSourceGovernanceStore {
	return &catalogKnowledgeSourceGovernanceStore{pool: pool}
}

func (s *catalogKnowledgeSourceGovernanceStore) ListSourceGovernance(ctx context.Context, workspaceID string, modelID int64, fileIDs []string) ([]PlatformKnowledgeSourceGovernanceRecord, error) {
	if s == nil || s.pool == nil {
		return nil, fmt.Errorf("catalog knowledge source governance store is not configured")
	}
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return nil, fmt.Errorf("workspace_id is required for source governance lookup")
	}
	fileIDs = platformKnowledgeCompactStrings(fileIDs)
	if modelID <= 0 {
		return nil, nil
	}
	tm, err := s.pool.GetTransactionManager(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	ctx = tenant.WithTransactionManager(ctx, tm)
	var out []PlatformKnowledgeSourceGovernanceRecord
	if err := tm.RunInTx(ctx, func(txCtx context.Context) error {
		executor := tm.Executor(txCtx)
		var err error
		out, err = queryPlatformKnowledgeSourceGovernance(txCtx, executor, modelID, fileIDs, time.Now().Unix())
		return err
	}); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *catalogKnowledgeSourceGovernanceStore) ListTableSourceGovernance(ctx context.Context, workspaceID string, modelID int64, tables []PlatformKnowledgeSemanticModelTableRef) ([]PlatformKnowledgeTableSourceGovernanceRecord, error) {
	if s == nil || s.pool == nil {
		return nil, fmt.Errorf("catalog knowledge source governance store is not configured")
	}
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return nil, fmt.Errorf("workspace_id is required for table source governance lookup")
	}
	tables = platformKnowledgeCompactTableRefs(tables)
	if modelID <= 0 || len(tables) == 0 {
		return nil, nil
	}
	tm, err := s.pool.GetTransactionManager(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	ctx = tenant.WithTransactionManager(ctx, tm)
	var out []PlatformKnowledgeTableSourceGovernanceRecord
	if err := tm.RunInTx(ctx, func(txCtx context.Context) error {
		executor := tm.Executor(txCtx)
		var err error
		out, err = queryPlatformKnowledgeTableSourceGovernance(txCtx, executor, modelID, time.Now().Unix())
		return err
	}); err != nil {
		return nil, err
	}
	return out, nil
}

func queryPlatformKnowledgeSourceGovernance(ctx context.Context, executor interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}, modelID int64, fileIDs []string, nowUnix int64) ([]PlatformKnowledgeSourceGovernanceRecord, error) {
	fileIDs = platformKnowledgeCompactStrings(fileIDs)
	if modelID <= 0 {
		return nil, nil
	}
	args := []any{modelID}
	fileFilter := ""
	if len(fileIDs) > 0 {
		placeholders := strings.TrimRight(strings.Repeat("?,", len(fileIDs)), ",")
		fileFilter = fmt.Sprintf(" AND kb_file_id IN (%s)", placeholders)
		args = append(args, platformKnowledgeStringsToAny(fileIDs)...)
	}
	rows, err := executor.QueryContext(ctx,
		fmt.Sprintf(`SELECT source_id, kb_file_id, status, COALESCE(enabled, 1), expires_at, force_enabled_after_expiry, tags, segment_version_id, index_version
FROM knowledge_base_sources
WHERE model_id = ? AND kb_file_id IS NOT NULL AND kb_file_id != '' AND status <> 'removed' AND source_type IN ('local_file', 'catalog_file')%s`, fileFilter),
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("query knowledge base source governance: %w", err)
	}
	defer rows.Close()
	out := make([]PlatformKnowledgeSourceGovernanceRecord, 0, len(fileIDs))
	for rows.Next() {
		var (
			sourceID         string
			fileID           string
			status           string
			enabled          bool
			expiresAt        sql.NullInt64
			forceEnabled     bool
			tagsRaw          sql.NullString
			segmentVersionID sql.NullString
			indexVersion     sql.NullInt64
		)
		if err := rows.Scan(&sourceID, &fileID, &status, &enabled, &expiresAt, &forceEnabled, &tagsRaw, &segmentVersionID, &indexVersion); err != nil {
			return nil, fmt.Errorf("scan knowledge base source governance: %w", err)
		}
		expired := expiresAt.Valid && expiresAt.Int64 > 0 && nowUnix > expiresAt.Int64
		out = append(out, PlatformKnowledgeSourceGovernanceRecord{
			SourceRowID:             strings.TrimSpace(sourceID),
			FileID:                  strings.TrimSpace(fileID),
			Status:                  status,
			Enabled:                 enabled,
			Expired:                 expired,
			EffectiveEnabled:        enabled && (!expired || forceEnabled),
			ForceEnabledAfterExpiry: forceEnabled,
			Tags:                    parsePlatformKnowledgeSourceTags(tagsRaw),
			SegmentVersionID:        strings.TrimSpace(segmentVersionID.String),
			IndexVersion:            indexVersion.Int64,
			IndexVersionValid:       indexVersion.Valid,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate knowledge base source governance: %w", err)
	}
	return out, nil
}

func queryPlatformKnowledgeTableSourceGovernance(ctx context.Context, executor interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}, modelID int64, nowUnix int64) ([]PlatformKnowledgeTableSourceGovernanceRecord, error) {
	if modelID <= 0 {
		return nil, nil
	}
	rows, err := executor.QueryContext(ctx, `SELECT
    kbs.source_id,
    COALESCE(NULLIF(kbs.db_name, ''), kbd.database_name, ''),
    COALESCE(NULLIF(kbs.table_name, ''), kbt.table_name, ''),
    COALESCE(srcd.database_name, ''),
    COALESCE(srct.table_name, ''),
    COALESCE(CAST(kbs.source_table_id AS CHAR), ''),
    COALESCE(CAST(kbs.kb_table_id AS CHAR), ''),
    kbs.status,
    COALESCE(kbs.enabled, 1),
    kbs.expires_at,
    kbs.force_enabled_after_expiry
FROM knowledge_base_sources kbs
LEFT JOIN catalog_table kbt ON kbt.table_id = kbs.kb_table_id
LEFT JOIN catalog_database kbd ON kbd.database_id = kbt.database_id
LEFT JOIN catalog_table srct ON srct.table_id = kbs.source_table_id
LEFT JOIN catalog_database srcd ON srcd.database_id = srct.database_id
WHERE kbs.model_id = ? AND kbs.source_type = 'catalog_table' AND kbs.status <> 'removed'`, modelID)
	if err != nil {
		return nil, fmt.Errorf("query knowledge base table source governance: %w", err)
	}
	defer rows.Close()
	out := make([]PlatformKnowledgeTableSourceGovernanceRecord, 0)
	for rows.Next() {
		var record PlatformKnowledgeTableSourceGovernanceRecord
		var expiresAt sql.NullInt64
		if err := rows.Scan(&record.SourceRowID, &record.DBName, &record.TableName, &record.SourceDBName, &record.SourceTableName, &record.SourceTableID, &record.KBTableID, &record.Status, &record.Enabled, &expiresAt, &record.ForceEnabledAfterExpiry); err != nil {
			return nil, fmt.Errorf("scan knowledge base table source governance: %w", err)
		}
		record.SourceRowID = strings.TrimSpace(record.SourceRowID)
		record.DBName = strings.TrimSpace(record.DBName)
		record.TableName = strings.TrimSpace(record.TableName)
		record.SourceDBName = strings.TrimSpace(record.SourceDBName)
		record.SourceTableName = strings.TrimSpace(record.SourceTableName)
		record.SourceTableID = strings.TrimSpace(record.SourceTableID)
		record.KBTableID = strings.TrimSpace(record.KBTableID)
		record.Expired = expiresAt.Valid && expiresAt.Int64 > 0 && nowUnix > expiresAt.Int64
		record.EffectiveEnabled = record.Enabled && (!record.Expired || record.ForceEnabledAfterExpiry)
		out = append(out, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate knowledge base table source governance: %w", err)
	}
	return out, nil
}

func parsePlatformKnowledgeSourceTags(raw sql.NullString) []string {
	if !raw.Valid || raw.String == "" {
		return nil
	}
	var tags []string
	if err := json.Unmarshal([]byte(raw.String), &tags); err != nil {
		return nil
	}
	return tags
}

type catalogKnowledgeVectorIndexResolver struct {
	pool tenant.ConnectionPool
}

func newCatalogKnowledgeVectorIndexResolver(pool tenant.ConnectionPool) *catalogKnowledgeVectorIndexResolver {
	return &catalogKnowledgeVectorIndexResolver{pool: pool}
}

type catalogKnowledgeLegacyIndexResolver struct {
	pool tenant.ConnectionPool
}

func newCatalogKnowledgeLegacyIndexResolver(pool tenant.ConnectionPool) *catalogKnowledgeLegacyIndexResolver {
	return &catalogKnowledgeLegacyIndexResolver{pool: pool}
}

type resolvedKnowledgeVectorIndex struct {
	vectorTable    string
	embeddingModel string
}

type resolvedKnowledgeImageIndex struct {
	imageVectorTable        string
	imageEmbeddingModel     string
	imageEmbeddingBackendID string
	imageEmbeddingDimension int
	imagePreprocessVersion  string
	imageDistanceMetric     string
}

func (r *catalogKnowledgeVectorIndexResolver) ResolveVectorIndex(ctx context.Context, req PlatformKnowledgeVectorIndexResolveRequest) (*PlatformKnowledgeVectorIndexResolveResult, error) {
	if r == nil || r.pool == nil {
		return nil, fmt.Errorf("catalog knowledge vector index resolver is not configured")
	}
	workspaceID := strings.TrimSpace(req.WorkspaceID)
	if workspaceID == "" {
		return nil, fmt.Errorf("workspace_id is required for vector index lookup")
	}
	tm, err := r.pool.GetTransactionManager(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	ctx = tenant.WithTransactionManager(ctx, tm)
	var indexes []resolvedKnowledgeVectorIndex
	if err := tm.RunInTx(ctx, func(txCtx context.Context) error {
		executor := tm.Executor(txCtx)
		indexes, err = queryKnowledgeResolvedVectorIndexes(txCtx, executor, platformKnowledgeCompactStrings(req.FileIDs), platformKnowledgeCompactStrings(req.VolumeIDs))
		return err
	}); err != nil {
		return nil, err
	}
	indexes = filterKnowledgeResolvedVectorIndexes(indexes, req.VectorTable, req.EmbeddingModel)
	vectorTable, embeddingModel, err := uniqueKnowledgeResolvedVectorIndex(indexes)
	if err != nil {
		return nil, err
	}
	if vectorTable == "" {
		return nil, nil
	}
	metadata := map[string]string{
		"source": "catalog.data_lineage",
	}
	if req.SemanticModelID > 0 {
		metadata["semantic_model_id"] = strconv.FormatInt(req.SemanticModelID, 10)
	}
	return &PlatformKnowledgeVectorIndexResolveResult{
		VectorTable:    vectorTable,
		EmbeddingModel: embeddingModel,
		Metadata:       metadata,
	}, nil
}

func (r *catalogKnowledgeVectorIndexResolver) ResolveImageIndex(ctx context.Context, req PlatformKnowledgeImageIndexResolveRequest) (*PlatformKnowledgeImageIndexResolveResult, error) {
	if r == nil || r.pool == nil {
		return nil, fmt.Errorf("catalog knowledge image index resolver is not configured")
	}
	workspaceID := strings.TrimSpace(req.WorkspaceID)
	if workspaceID == "" {
		return nil, fmt.Errorf("workspace_id is required for image vector index lookup")
	}
	tm, err := r.pool.GetTransactionManager(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	ctx = tenant.WithTransactionManager(ctx, tm)
	var indexes []resolvedKnowledgeImageIndex
	if err := tm.RunInTx(ctx, func(txCtx context.Context) error {
		executor := tm.Executor(txCtx)
		rootAssetIDs := make([]string, 0)
		fileAssetIDs, err := queryKnowledgeFileAssetIDs(txCtx, executor, platformKnowledgeCompactStrings(req.FileIDs))
		if err != nil {
			return err
		}
		if len(fileAssetIDs) > 0 {
			fileRoots, err := queryKnowledgeRootAssetIDsForAssetIDs(txCtx, executor, fileAssetIDs)
			if err != nil {
				return err
			}
			rootAssetIDs = append(rootAssetIDs, fileRoots...)
		}
		if len(req.VolumeIDs) > 0 {
			volumeRoots, err := queryKnowledgeRootAssetIDsForVolumes(txCtx, executor, platformKnowledgeCompactStrings(req.VolumeIDs))
			if err != nil {
				return err
			}
			rootAssetIDs = append(rootAssetIDs, volumeRoots...)
		}
		rootAssetIDs = platformKnowledgeCompactStrings(rootAssetIDs)
		if len(rootAssetIDs) == 0 {
			return nil
		}
		indexes, err = queryKnowledgeIndexedImageVectorAssets(txCtx, executor, rootAssetIDs)
		return err
	}); err != nil {
		return nil, err
	}
	indexes = filterKnowledgeResolvedImageIndexes(indexes, req.ImageVectorTable, req.ImageEmbeddingModel)
	index, err := uniqueKnowledgeResolvedImageIndex(indexes)
	if err != nil {
		return nil, err
	}
	if index.imageVectorTable == "" {
		return nil, nil
	}
	metadata := map[string]string{
		"source": "catalog.data_lineage",
	}
	if req.SemanticModelID > 0 {
		metadata["semantic_model_id"] = strconv.FormatInt(req.SemanticModelID, 10)
	}
	return &PlatformKnowledgeImageIndexResolveResult{
		ImageVectorTable:        index.imageVectorTable,
		ImageEmbeddingModel:     index.imageEmbeddingModel,
		ImageEmbeddingBackendID: index.imageEmbeddingBackendID,
		ImageEmbeddingDimension: index.imageEmbeddingDimension,
		ImagePreprocessVersion:  index.imagePreprocessVersion,
		ImageDistanceMetric:     index.imageDistanceMetric,
		Metadata:                metadata,
	}, nil
}

func (r *catalogKnowledgeLegacyIndexResolver) ResolveLegacyIndexVersions(ctx context.Context, req PlatformKnowledgeLegacyIndexResolveRequest) (map[string]knowledge.RAGIndexVersionConstraint, error) {
	if r == nil || r.pool == nil {
		return nil, fmt.Errorf("catalog knowledge legacy index resolver is not configured")
	}
	workspaceID := strings.TrimSpace(req.WorkspaceID)
	if workspaceID == "" {
		return nil, fmt.Errorf("workspace_id is required for legacy index version lookup")
	}
	vectorTable := strings.TrimSpace(req.VectorTable)
	fileIDs := platformKnowledgeCompactStrings(req.FileIDs)
	if vectorTable == "" || len(fileIDs) == 0 {
		return nil, nil
	}
	tm, err := r.pool.GetTransactionManager(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	ctx = tenant.WithTransactionManager(ctx, tm)
	var out map[string]knowledge.RAGIndexVersionConstraint
	if err := tm.RunInTx(ctx, func(txCtx context.Context) error {
		var err error
		out, err = queryPlatformKnowledgeLegacyIndexVersionConstraints(txCtx, tm.Executor(txCtx), vectorTable, fileIDs)
		return err
	}); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *catalogKnowledgeLegacyIndexResolver) ResolveLegacyIndexFileIDs(ctx context.Context, req PlatformKnowledgeLegacyIndexResolveRequest) ([]string, error) {
	if r == nil || r.pool == nil {
		return nil, fmt.Errorf("catalog knowledge legacy index resolver is not configured")
	}
	workspaceID := strings.TrimSpace(req.WorkspaceID)
	if workspaceID == "" {
		return nil, fmt.Errorf("workspace_id is required for legacy index file lookup")
	}
	vectorTable := strings.TrimSpace(req.VectorTable)
	if vectorTable == "" {
		return nil, nil
	}
	tm, err := r.pool.GetTransactionManager(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	ctx = tenant.WithTransactionManager(ctx, tm)
	var out []string
	if err := tm.RunInTx(ctx, func(txCtx context.Context) error {
		var err error
		out, err = queryPlatformKnowledgeLegacyIndexFileIDs(txCtx, tm.Executor(txCtx), vectorTable)
		return err
	}); err != nil {
		return nil, err
	}
	return out, nil
}

type platformKnowledgeLegacyIndexSelection struct {
	hasValue bool
	value    int64
	hasNull  bool
}

func queryPlatformKnowledgeLegacyIndexFileIDs(ctx context.Context, executor interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}, vectorTable string) ([]string, error) {
	if strings.TrimSpace(vectorTable) == "" {
		return nil, nil
	}
	quotedTable, err := platformKnowledgeQuoteQualifiedIdentifier(vectorTable)
	if err != nil {
		return nil, fmt.Errorf("legacy index file lookup: invalid vector_table %q", vectorTable)
	}
	columns, err := queryPlatformKnowledgeColumnSet(ctx, executor, quotedTable, "legacy index file lookup")
	if err != nil {
		return nil, err
	}
	if _, ok := columns["file_id"]; !ok {
		return nil, nil
	}
	where := []string{"file_id IS NOT NULL", "file_id != ''"}
	if _, ok := columns["disabled"]; ok {
		where = append(where, "COALESCE(disabled, 0) = 0")
	}
	if _, ok := columns["level"]; ok {
		where = append(where, "level = 'chunk'")
	}
	rows, err := executor.QueryContext(ctx,
		fmt.Sprintf(`SELECT DISTINCT file_id FROM %s WHERE %s ORDER BY file_id`, quotedTable, strings.Join(where, " AND ")),
	)
	if err != nil {
		return nil, fmt.Errorf("legacy index file lookup: query vector rows: %w", err)
	}
	defer rows.Close()
	out := make([]string, 0)
	for rows.Next() {
		var fileID string
		if err := rows.Scan(&fileID); err != nil {
			return nil, fmt.Errorf("legacy index file lookup: scan vector row: %w", err)
		}
		out = append(out, fileID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("legacy index file lookup: iterate vector rows: %w", err)
	}
	return platformKnowledgeCompactStrings(out), nil
}

func queryPlatformKnowledgeLegacyIndexVersionConstraints(ctx context.Context, executor interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}, vectorTable string, fileIDs []string) (map[string]knowledge.RAGIndexVersionConstraint, error) {
	fileIDs = platformKnowledgeCompactStrings(fileIDs)
	if strings.TrimSpace(vectorTable) == "" || len(fileIDs) == 0 {
		return nil, nil
	}
	quotedTable, err := platformKnowledgeQuoteQualifiedIdentifier(vectorTable)
	if err != nil {
		return nil, fmt.Errorf("legacy index version lookup: invalid vector_table %q", vectorTable)
	}
	columns, err := queryPlatformKnowledgeColumnSet(ctx, executor, quotedTable, "legacy index version lookup")
	if err != nil {
		return nil, err
	}
	if _, ok := columns["index_version"]; !ok {
		return nil, nil
	}
	placeholders := strings.TrimRight(strings.Repeat("?,", len(fileIDs)), ",")
	where := []string{fmt.Sprintf("file_id IN (%s)", placeholders)}
	if _, ok := columns["disabled"]; ok {
		where = append(where, "COALESCE(disabled, 0) = 0")
	}
	if _, ok := columns["level"]; ok {
		where = append(where, "level = 'chunk'")
	}
	rows, err := executor.QueryContext(ctx,
		fmt.Sprintf(`SELECT DISTINCT file_id, index_version FROM %s WHERE %s ORDER BY file_id, index_version`, quotedTable, strings.Join(where, " AND ")),
		platformKnowledgeStringsToAny(fileIDs)...,
	)
	if err != nil {
		return nil, fmt.Errorf("legacy index version lookup: query vector rows: %w", err)
	}
	defer rows.Close()
	selected := make(map[string]platformKnowledgeLegacyIndexSelection, len(fileIDs))
	for rows.Next() {
		var (
			fileID          string
			rawIndexVersion any
		)
		if err := rows.Scan(&fileID, &rawIndexVersion); err != nil {
			return nil, fmt.Errorf("legacy index version lookup: scan vector row: %w", err)
		}
		fileID = strings.TrimSpace(fileID)
		if fileID == "" {
			continue
		}
		version, valid, err := platformKnowledgeLegacyIndexVersion(rawIndexVersion)
		if err != nil {
			return nil, fmt.Errorf("legacy index version lookup: parse index_version file_id=%s: %w", fileID, err)
		}
		item := selected[fileID]
		if valid {
			if !item.hasValue || version < item.value {
				item.hasValue = true
				item.value = version
			}
		} else {
			item.hasNull = true
		}
		selected[fileID] = item
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("legacy index version lookup: iterate vector rows: %w", err)
	}
	out := make(map[string]knowledge.RAGIndexVersionConstraint, len(selected))
	for fileID, item := range selected {
		if item.hasValue {
			out[fileID] = knowledge.RAGIndexVersionConstraint{
				Kind:  knowledge.RAGIndexVersionConstraintValue,
				Value: item.value,
			}
			continue
		}
		if item.hasNull {
			out[fileID] = knowledge.RAGIndexVersionConstraint{Kind: knowledge.RAGIndexVersionConstraintNull}
		}
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

func queryPlatformKnowledgeColumnSet(ctx context.Context, executor interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}, quotedTable string, operation string) (map[string]struct{}, error) {
	rows, err := executor.QueryContext(ctx, fmt.Sprintf("SHOW COLUMNS FROM %s", quotedTable))
	if err != nil {
		if platformKnowledgeIsNoSuchTableError(err) {
			return nil, errPlatformKnowledgeVectorIndexNotReady
		}
		return nil, fmt.Errorf("%s: inspect vector index columns: %w", operation, err)
	}
	defer rows.Close()
	columnNames, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("%s: read vector index columns: %w", operation, err)
	}
	out := make(map[string]struct{})
	for rows.Next() {
		values := make([]any, len(columnNames))
		dest := make([]any, len(values))
		for i := range values {
			dest[i] = &values[i]
		}
		if err := rows.Scan(dest...); err != nil {
			return nil, fmt.Errorf("%s: scan vector index columns: %w", operation, err)
		}
		if len(values) > 0 {
			if name := strings.ToLower(strings.TrimSpace(platformKnowledgeValueString(values[0]))); name != "" {
				out[name] = struct{}{}
			}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: iterate vector index columns: %w", operation, err)
	}
	return out, nil
}

func platformKnowledgeIsNoSuchTableError(err error) bool {
	var mysqlErr *mysql.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == 1146
}

func platformKnowledgeLegacyIndexVersion(value any) (int64, bool, error) {
	switch typed := value.(type) {
	case nil:
		return 0, false, nil
	case int:
		return int64(typed), true, nil
	case int8:
		return int64(typed), true, nil
	case int16:
		return int64(typed), true, nil
	case int32:
		return int64(typed), true, nil
	case int64:
		return typed, true, nil
	case uint:
		if uint64(typed) > uint64(^uint64(0)>>1) {
			return 0, false, fmt.Errorf("uint value overflows int64")
		}
		return int64(typed), true, nil
	case uint8:
		return int64(typed), true, nil
	case uint16:
		return int64(typed), true, nil
	case uint32:
		return int64(typed), true, nil
	case uint64:
		if typed > uint64(^uint64(0)>>1) {
			return 0, false, fmt.Errorf("uint64 value overflows int64")
		}
		return int64(typed), true, nil
	case []byte:
		return platformKnowledgeParseLegacyIndexVersionString(string(typed))
	case string:
		return platformKnowledgeParseLegacyIndexVersionString(typed)
	default:
		return platformKnowledgeParseLegacyIndexVersionString(fmt.Sprint(typed))
	}
}

func platformKnowledgeParseLegacyIndexVersionString(value string) (int64, bool, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false, nil
	}
	version, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, false, err
	}
	return version, true, nil
}

func queryKnowledgeResolvedVectorIndexes(ctx context.Context, executor interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}, fileIDs, volumeIDs []string) ([]resolvedKnowledgeVectorIndex, error) {
	rootAssetIDs := make([]string, 0)
	fileAssetIDs, err := queryKnowledgeFileAssetIDs(ctx, executor, fileIDs)
	if err != nil {
		return nil, err
	}
	if len(fileAssetIDs) > 0 {
		fileRoots, err := queryKnowledgeRootAssetIDsForAssetIDs(ctx, executor, fileAssetIDs)
		if err != nil {
			return nil, err
		}
		rootAssetIDs = append(rootAssetIDs, fileRoots...)
	}
	if len(volumeIDs) > 0 {
		volumeRoots, err := queryKnowledgeRootAssetIDsForVolumes(ctx, executor, volumeIDs)
		if err != nil {
			return nil, err
		}
		rootAssetIDs = append(rootAssetIDs, volumeRoots...)
	}
	rootAssetIDs = platformKnowledgeCompactStrings(rootAssetIDs)
	if len(rootAssetIDs) == 0 {
		return nil, nil
	}
	return queryKnowledgeIndexedVectorAssets(ctx, executor, rootAssetIDs)
}

func queryKnowledgeFileAssetIDs(ctx context.Context, executor interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}, fileIDs []string) ([]string, error) {
	fileIDs = platformKnowledgeCompactStrings(fileIDs)
	if len(fileIDs) == 0 {
		return nil, nil
	}
	placeholders := strings.TrimRight(strings.Repeat("?,", len(fileIDs)), ",")
	args := append([]any{tenant.DataAssetTypeFile}, platformKnowledgeStringsToAny(fileIDs)...)
	rows, err := executor.QueryContext(ctx,
		fmt.Sprintf(`SELECT asset_id FROM data_asset WHERE asset_type = ? AND asset_ref IN (%s)`, placeholders),
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("query file data assets: %w", err)
	}
	defer rows.Close()
	out := make([]string, 0, len(fileIDs))
	for rows.Next() {
		var assetID string
		if err := rows.Scan(&assetID); err != nil {
			return nil, err
		}
		out = append(out, assetID)
	}
	return platformKnowledgeCompactStrings(out), rows.Err()
}

func queryKnowledgeRootAssetIDsForAssetIDs(ctx context.Context, executor interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}, assetIDs []string) ([]string, error) {
	assetIDs = platformKnowledgeCompactStrings(assetIDs)
	if len(assetIDs) == 0 {
		return nil, nil
	}
	placeholders := strings.TrimRight(strings.Repeat("?,", len(assetIDs)), ",")
	args := platformKnowledgeStringsToAny(assetIDs)
	args = append(args, platformKnowledgeStringsToAny(assetIDs)...)
	args = append(args, platformKnowledgeStringsToAny(assetIDs)...)
	rows, err := executor.QueryContext(ctx,
		fmt.Sprintf(`SELECT DISTINCT root_asset_id FROM data_derivation WHERE root_asset_id IN (%s) OR source_asset_id IN (%s) OR target_asset_id IN (%s)`, placeholders, placeholders, placeholders),
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("query file lineage roots: %w", err)
	}
	defer rows.Close()
	return platformKnowledgeScanStringRows(rows)
}

func queryKnowledgeRootAssetIDsForVolumes(ctx context.Context, executor interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}, volumeIDs []string) ([]string, error) {
	ids, err := platformKnowledgeParseVolumeIDArgs(volumeIDs)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, nil
	}
	placeholders := strings.TrimRight(strings.Repeat("?,", len(ids)), ",")
	args := append([]any{tenant.DataAssetTypeFile}, ids...)
	args = append(args, ids...)
	rows, err := executor.QueryContext(ctx,
		fmt.Sprintf(`SELECT DISTINCT d.root_asset_id
FROM data_derivation d
JOIN data_asset a ON a.asset_id = d.root_asset_id
LEFT JOIN volume_files vf ON vf.file_id = a.asset_ref
WHERE a.asset_type = ? AND (a.volume_id IN (%s) OR vf.volume_id IN (%s))`, placeholders, placeholders),
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("query volume lineage roots: %w", err)
	}
	defer rows.Close()
	return platformKnowledgeScanStringRows(rows)
}

func queryKnowledgeIndexedVectorAssets(ctx context.Context, executor interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}, rootAssetIDs []string) ([]resolvedKnowledgeVectorIndex, error) {
	rootAssetIDs = platformKnowledgeCompactStrings(rootAssetIDs)
	if len(rootAssetIDs) == 0 {
		return nil, nil
	}
	placeholders := strings.TrimRight(strings.Repeat("?,", len(rootAssetIDs)), ",")
	args := []any{"indexed_from", tenant.DataAssetTypeVectorIndex}
	args = append(args, platformKnowledgeStringsToAny(rootAssetIDs)...)
	rows, err := executor.QueryContext(ctx,
		fmt.Sprintf(`SELECT va.asset_ref, va.meta FROM data_derivation d JOIN data_asset va ON va.asset_id = d.target_asset_id WHERE d.kind = ? AND va.asset_type = ? AND d.root_asset_id IN (%s) ORDER BY d.updated_at DESC, d.id DESC`, placeholders),
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("query indexed vector assets: %w", err)
	}
	defer rows.Close()
	out := make([]resolvedKnowledgeVectorIndex, 0)
	for rows.Next() {
		var (
			vectorTable string
			metaJSON    sql.NullString
		)
		if err := rows.Scan(&vectorTable, &metaJSON); err != nil {
			return nil, err
		}
		vectorTable = strings.TrimSpace(vectorTable)
		if vectorTable == "" {
			continue
		}
		embeddingModel, isImage, err := knowledgeTextEmbeddingModelFromMeta(metaJSON)
		if err != nil {
			return nil, fmt.Errorf("parse vector asset metadata for %s: %w", vectorTable, err)
		}
		if isImage {
			continue
		}
		out = append(out, resolvedKnowledgeVectorIndex{
			vectorTable:    vectorTable,
			embeddingModel: embeddingModel,
		})
	}
	return out, rows.Err()
}

func queryKnowledgeIndexedImageVectorAssets(ctx context.Context, executor interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}, rootAssetIDs []string) ([]resolvedKnowledgeImageIndex, error) {
	rootAssetIDs = platformKnowledgeCompactStrings(rootAssetIDs)
	if len(rootAssetIDs) == 0 {
		return nil, nil
	}
	placeholders := strings.TrimRight(strings.Repeat("?,", len(rootAssetIDs)), ",")
	args := []any{"indexed_from", tenant.DataAssetTypeVectorIndex}
	args = append(args, platformKnowledgeStringsToAny(rootAssetIDs)...)
	rows, err := executor.QueryContext(ctx,
		fmt.Sprintf(`SELECT va.asset_ref, va.meta FROM data_derivation d JOIN data_asset va ON va.asset_id = d.target_asset_id WHERE d.kind = ? AND va.asset_type = ? AND d.root_asset_id IN (%s) ORDER BY d.updated_at DESC, d.id DESC`, placeholders),
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("query indexed image vector assets: %w", err)
	}
	defer rows.Close()
	out := make([]resolvedKnowledgeImageIndex, 0)
	for rows.Next() {
		var (
			vectorTable string
			metaJSON    sql.NullString
		)
		if err := rows.Scan(&vectorTable, &metaJSON); err != nil {
			return nil, err
		}
		vectorTable = strings.TrimSpace(vectorTable)
		if vectorTable == "" {
			continue
		}
		index, isImage, err := knowledgeImageIndexFromMeta(vectorTable, metaJSON)
		if err != nil {
			return nil, fmt.Errorf("parse image vector asset metadata for %s: %w", vectorTable, err)
		}
		if !isImage {
			continue
		}
		out = append(out, index)
	}
	return out, rows.Err()
}

func uniqueKnowledgeResolvedVectorIndex(indexes []resolvedKnowledgeVectorIndex) (string, string, error) {
	vectorTable := ""
	embeddingModel := ""
	for _, index := range indexes {
		currentTable := strings.TrimSpace(index.vectorTable)
		currentModel := strings.TrimSpace(index.embeddingModel)
		if currentTable == "" {
			continue
		}
		if currentModel == "" {
			return "", "", fmt.Errorf("resolve vector index %s: embedding_model is missing from catalog data asset metadata", currentTable)
		}
		if vectorTable != "" && vectorTable != currentTable {
			return "", "", fmt.Errorf("resolve vector index: multiple vector tables selected: %s, %s", vectorTable, currentTable)
		}
		if embeddingModel != "" && embeddingModel != currentModel {
			return "", "", fmt.Errorf("resolve vector index: multiple embedding models selected: %s, %s", embeddingModel, currentModel)
		}
		vectorTable = currentTable
		embeddingModel = currentModel
	}
	return vectorTable, embeddingModel, nil
}

func filterKnowledgeResolvedVectorIndexes(indexes []resolvedKnowledgeVectorIndex, vectorTable, embeddingModel string) []resolvedKnowledgeVectorIndex {
	vectorTable = strings.TrimSpace(vectorTable)
	embeddingModel = strings.TrimSpace(embeddingModel)
	if vectorTable == "" && embeddingModel == "" {
		return indexes
	}
	out := make([]resolvedKnowledgeVectorIndex, 0, len(indexes))
	for _, index := range indexes {
		if vectorTable != "" && strings.TrimSpace(index.vectorTable) != vectorTable {
			continue
		}
		if embeddingModel != "" && strings.TrimSpace(index.embeddingModel) != embeddingModel {
			continue
		}
		out = append(out, index)
	}
	return out
}

func uniqueKnowledgeResolvedImageIndex(indexes []resolvedKnowledgeImageIndex) (resolvedKnowledgeImageIndex, error) {
	out := resolvedKnowledgeImageIndex{}
	for _, index := range indexes {
		currentTable := strings.TrimSpace(index.imageVectorTable)
		if currentTable == "" {
			continue
		}
		if strings.TrimSpace(index.imageEmbeddingModel) == "" {
			return resolvedKnowledgeImageIndex{}, fmt.Errorf("resolve image vector index %s: image_embedding_model is missing from catalog data asset metadata", currentTable)
		}
		if index.imageEmbeddingDimension <= 0 {
			return resolvedKnowledgeImageIndex{}, fmt.Errorf("resolve image vector index %s: image_embedding_dimension is missing from catalog data asset metadata", currentTable)
		}
		if strings.TrimSpace(index.imagePreprocessVersion) == "" {
			return resolvedKnowledgeImageIndex{}, fmt.Errorf("resolve image vector index %s: preprocess_version is missing from catalog data asset metadata", currentTable)
		}
		if strings.TrimSpace(index.imageDistanceMetric) == "" {
			return resolvedKnowledgeImageIndex{}, fmt.Errorf("resolve image vector index %s: distance_metric is missing from catalog data asset metadata", currentTable)
		}
		if out.imageVectorTable != "" && out.imageVectorTable != currentTable {
			return resolvedKnowledgeImageIndex{}, fmt.Errorf("resolve image vector index: multiple image vector tables selected: %s, %s", out.imageVectorTable, currentTable)
		}
		if out.imageEmbeddingModel != "" && out.imageEmbeddingModel != index.imageEmbeddingModel {
			return resolvedKnowledgeImageIndex{}, fmt.Errorf("resolve image vector index: multiple image embedding models selected: %s, %s", out.imageEmbeddingModel, index.imageEmbeddingModel)
		}
		if out.imageEmbeddingBackendID != "" && index.imageEmbeddingBackendID != "" && out.imageEmbeddingBackendID != index.imageEmbeddingBackendID {
			return resolvedKnowledgeImageIndex{}, fmt.Errorf("resolve image vector index: multiple image embedding backend ids selected: %s, %s", out.imageEmbeddingBackendID, index.imageEmbeddingBackendID)
		}
		if out.imageEmbeddingDimension > 0 && out.imageEmbeddingDimension != index.imageEmbeddingDimension {
			return resolvedKnowledgeImageIndex{}, fmt.Errorf("resolve image vector index: multiple image embedding dimensions selected: %d, %d", out.imageEmbeddingDimension, index.imageEmbeddingDimension)
		}
		if out.imagePreprocessVersion != "" && out.imagePreprocessVersion != index.imagePreprocessVersion {
			return resolvedKnowledgeImageIndex{}, fmt.Errorf("resolve image vector index: multiple image preprocess versions selected: %s, %s", out.imagePreprocessVersion, index.imagePreprocessVersion)
		}
		if out.imageDistanceMetric != "" && out.imageDistanceMetric != index.imageDistanceMetric {
			return resolvedKnowledgeImageIndex{}, fmt.Errorf("resolve image vector index: multiple image distance metrics selected: %s, %s", out.imageDistanceMetric, index.imageDistanceMetric)
		}
		out = index
	}
	return out, nil
}

func filterKnowledgeResolvedImageIndexes(indexes []resolvedKnowledgeImageIndex, imageVectorTable, imageEmbeddingModel string) []resolvedKnowledgeImageIndex {
	imageVectorTable = strings.TrimSpace(imageVectorTable)
	imageEmbeddingModel = strings.TrimSpace(imageEmbeddingModel)
	if imageVectorTable == "" && imageEmbeddingModel == "" {
		return indexes
	}
	out := make([]resolvedKnowledgeImageIndex, 0, len(indexes))
	for _, index := range indexes {
		if imageVectorTable != "" && strings.TrimSpace(index.imageVectorTable) != imageVectorTable {
			continue
		}
		if imageEmbeddingModel != "" && strings.TrimSpace(index.imageEmbeddingModel) != imageEmbeddingModel {
			continue
		}
		out = append(out, index)
	}
	return out
}

func knowledgeTextEmbeddingModelFromMeta(metaJSON sql.NullString) (string, bool, error) {
	if !metaJSON.Valid || strings.TrimSpace(metaJSON.String) == "" {
		return "", false, nil
	}
	var meta map[string]any
	if err := json.Unmarshal([]byte(metaJSON.String), &meta); err != nil {
		return "", false, fmt.Errorf("invalid data_asset.meta JSON: %w", err)
	}
	isImage := strings.TrimSpace(platformKnowledgeValueString(meta["index_modality"])) == "image"
	return strings.TrimSpace(platformKnowledgeValueString(meta["embedding_model"])), isImage, nil
}

func knowledgeImageIndexFromMeta(vectorTable string, metaJSON sql.NullString) (resolvedKnowledgeImageIndex, bool, error) {
	if !metaJSON.Valid || strings.TrimSpace(metaJSON.String) == "" {
		return resolvedKnowledgeImageIndex{}, false, nil
	}
	var meta map[string]any
	if err := json.Unmarshal([]byte(metaJSON.String), &meta); err != nil {
		return resolvedKnowledgeImageIndex{}, false, fmt.Errorf("invalid data_asset.meta JSON: %w", err)
	}
	if strings.TrimSpace(platformKnowledgeValueString(meta["index_modality"])) != "image" {
		return resolvedKnowledgeImageIndex{}, false, nil
	}
	return resolvedKnowledgeImageIndex{
		imageVectorTable:        strings.TrimSpace(vectorTable),
		imageEmbeddingModel:     strings.TrimSpace(platformKnowledgeValueString(meta["image_embedding_model"])),
		imageEmbeddingBackendID: strings.TrimSpace(platformKnowledgeValueString(meta["image_embedding_backend_id"])),
		imageEmbeddingDimension: platformKnowledgeIntFromMeta(meta, "image_embedding_dimension"),
		imagePreprocessVersion:  strings.TrimSpace(platformKnowledgeValueString(meta["preprocess_version"])),
		imageDistanceMetric:     strings.TrimSpace(platformKnowledgeValueString(meta["distance_metric"])),
	}, true, nil
}

type knowledgeSemanticModelFilesJSON struct {
	FileIDs                  []string                                   `json:"file_ids"`
	Parents                  []string                                   `json:"parents,omitempty"`
	VolumeIDs                []string                                   `json:"volume_ids,omitempty"`
	Volumes                  []knowledgeSemanticModelVolume             `json:"volumes,omitempty"`
	Volume                   *knowledgeSemanticModelVolume              `json:"volume,omitempty"`
	VectorTable              string                                     `json:"vector_table,omitempty"`
	EmbeddingModel           string                                     `json:"embedding_model,omitempty"`
	ImageVectorTable         string                                     `json:"image_vector_table,omitempty"`
	ImageEmbeddingModel      string                                     `json:"image_embedding_model,omitempty"`
	ImageEmbeddingBackendID  string                                     `json:"image_embedding_backend_id,omitempty"`
	ImageEmbeddingDimension  int                                        `json:"image_embedding_dimension,omitempty"`
	ImagePreprocessVersion   string                                     `json:"image_preprocess_version,omitempty"`
	ImageDistanceMetric      string                                     `json:"image_distance_metric,omitempty"`
	ImageIndexConfigs        []knowledgeSemanticModelImageIndexConfig   `json:"image_index_configs,omitempty"`
	ActiveImageIndexConfigID string                                     `json:"active_image_index_config_id,omitempty"`
	ImageIndexStatus         string                                     `json:"image_index_status,omitempty"`
	ImageIndexFileStatuses   []knowledgeSemanticModelImageIndexFileStat `json:"image_index_file_statuses,omitempty"`
}

type knowledgeSemanticModelVolume struct {
	VolumeID any      `json:"volume_id"`
	Parents  []string `json:"parents,omitempty"`
	Path     []string `json:"path,omitempty"`
}

type knowledgeSemanticModelImageIndexConfig struct {
	ID                      string `json:"id,omitempty"`
	Name                    string `json:"name,omitempty"`
	ImageVectorTable        string `json:"image_vector_table,omitempty"`
	ImageEmbeddingModel     string `json:"image_embedding_model,omitempty"`
	ImageEmbeddingBackendID string `json:"image_embedding_backend_id,omitempty"`
	ImageEmbeddingDimension int    `json:"image_embedding_dimension,omitempty"`
	ImagePreprocessVersion  string `json:"image_preprocess_version,omitempty"`
	ImageDistanceMetric     string `json:"image_distance_metric,omitempty"`
	ImageScope              string `json:"image_scope,omitempty"`
	Status                  string `json:"status,omitempty"`
}

type knowledgeSemanticModelImageIndexFileStat struct {
	FileID        string `json:"file_id,omitempty"`
	ConfigID      string `json:"config_id,omitempty"`
	Status        string `json:"status,omitempty"`
	ErrorCode     string `json:"error_code,omitempty"`
	ErrorMessage  string `json:"error_message,omitempty"`
	IndexedImages int    `json:"indexed_images,omitempty"`
}

type knowledgeSemanticModelFileConfig struct {
	VectorTable             string
	EmbeddingModel          string
	ImageVectorTable        string
	ImageEmbeddingModel     string
	ImageEmbeddingBackendID string
	ImageEmbeddingDimension int
	ImagePreprocessVersion  string
	ImageDistanceMetric     string
	ImageIndexConfigID      string
	ImageIndexConfigFromSet bool
	ImageIndexConfigReady   bool
}

func parseKnowledgeSemanticModelFiles(raw json.RawMessage) ([]string, []string, knowledgeSemanticModelFileConfig, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return nil, nil, knowledgeSemanticModelFileConfig{}, nil
	}
	var files knowledgeSemanticModelFilesJSON
	if err := json.Unmarshal(raw, &files); err != nil {
		return nil, nil, knowledgeSemanticModelFileConfig{}, fmt.Errorf("files must be structured JSON object: %w", err)
	}
	volumeIDs := platformKnowledgeCompactStrings(files.VolumeIDs)
	if files.Volume != nil {
		if volumeID := strings.TrimSpace(platformKnowledgeValueString(files.Volume.VolumeID)); volumeID != "" {
			volumeIDs = append(volumeIDs, volumeID)
		}
	}
	for _, volume := range files.Volumes {
		if volumeID := strings.TrimSpace(platformKnowledgeValueString(volume.VolumeID)); volumeID != "" {
			volumeIDs = append(volumeIDs, volumeID)
		}
	}
	config := knowledgeSemanticModelFileConfig{
		VectorTable:             strings.TrimSpace(files.VectorTable),
		EmbeddingModel:          strings.TrimSpace(files.EmbeddingModel),
		ImageVectorTable:        strings.TrimSpace(files.ImageVectorTable),
		ImageEmbeddingModel:     strings.TrimSpace(files.ImageEmbeddingModel),
		ImageEmbeddingBackendID: strings.TrimSpace(files.ImageEmbeddingBackendID),
		ImageEmbeddingDimension: files.ImageEmbeddingDimension,
		ImagePreprocessVersion:  strings.TrimSpace(files.ImagePreprocessVersion),
		ImageDistanceMetric:     strings.TrimSpace(files.ImageDistanceMetric),
	}
	if imageConfig, ok, err := activeKnowledgeSemanticModelImageIndexConfig(files); err != nil {
		return nil, nil, knowledgeSemanticModelFileConfig{}, err
	} else if ok {
		config.ImageIndexConfigID = strings.TrimSpace(imageConfig.ID)
		config.ImageIndexConfigFromSet = true
		config.ImageIndexConfigReady = knowledgeSemanticModelImageIndexConfigReady(imageConfig)
		if config.ImageIndexConfigReady {
			config.ImageVectorTable = strings.TrimSpace(imageConfig.ImageVectorTable)
			config.ImageEmbeddingModel = strings.TrimSpace(imageConfig.ImageEmbeddingModel)
			config.ImageEmbeddingBackendID = strings.TrimSpace(imageConfig.ImageEmbeddingBackendID)
			config.ImageEmbeddingDimension = imageConfig.ImageEmbeddingDimension
			config.ImagePreprocessVersion = strings.TrimSpace(imageConfig.ImagePreprocessVersion)
			config.ImageDistanceMetric = strings.TrimSpace(imageConfig.ImageDistanceMetric)
		} else {
			config.ImageVectorTable = ""
			config.ImageEmbeddingModel = ""
			config.ImageEmbeddingBackendID = ""
			config.ImageEmbeddingDimension = 0
			config.ImagePreprocessVersion = ""
			config.ImageDistanceMetric = ""
		}
	}
	return platformKnowledgeCompactStrings(files.FileIDs), platformKnowledgeCompactStrings(volumeIDs), config, nil
}

func activeKnowledgeSemanticModelImageIndexConfig(files knowledgeSemanticModelFilesJSON) (knowledgeSemanticModelImageIndexConfig, bool, error) {
	if len(files.ImageIndexConfigs) == 0 {
		return knowledgeSemanticModelImageIndexConfig{}, false, nil
	}
	activeID := strings.TrimSpace(files.ActiveImageIndexConfigID)
	if activeID == "" {
		return knowledgeSemanticModelImageIndexConfig{}, false, nil
	}
	for _, config := range files.ImageIndexConfigs {
		if strings.TrimSpace(config.ID) == activeID {
			return config, true, nil
		}
	}
	return knowledgeSemanticModelImageIndexConfig{}, false, fmt.Errorf("active_image_index_config_id %q not found in image_index_configs", activeID)
}

func knowledgeSemanticModelImageIndexConfigReady(config knowledgeSemanticModelImageIndexConfig) bool {
	status := strings.TrimSpace(config.Status)
	return status == "" || status == "ready"
}

type knowledgeSemanticModelTablesJSON struct {
	DBName     string   `json:"db_name"`
	TableNames []string `json:"table_names"`
}

func parseKnowledgeSemanticModelTableRefs(raw json.RawMessage, dbName string) ([]PlatformKnowledgeSemanticModelTableRef, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return nil, nil
	}
	var structured []knowledgeSemanticModelTablesJSON
	if err := json.Unmarshal(raw, &structured); err == nil {
		tables := make([]PlatformKnowledgeSemanticModelTableRef, 0)
		// Never filter structured model tables by the optional default dbName.
		// Callers collect every table first; single-db default is decided only
		// after the full set is known (see ResolveKnowledgeScope).
		for _, item := range structured {
			itemDBName := strings.TrimSpace(item.DBName)
			effectiveDBName := itemDBName
			if effectiveDBName == "" {
				effectiveDBName = strings.TrimSpace(dbName)
			}
			for _, tableName := range item.TableNames {
				tables = append(tables, PlatformKnowledgeSemanticModelTableRef{
					DBName:    effectiveDBName,
					TableName: tableName,
				})
			}
		}
		if len(tables) > 0 || len(structured) > 0 {
			return platformKnowledgeCompactTableRefs(tables), nil
		}
	}
	var legacy []string
	if err := json.Unmarshal(raw, &legacy); err != nil {
		return nil, fmt.Errorf("tables must be structured JSON array: %w", err)
	}
	tables := make([]PlatformKnowledgeSemanticModelTableRef, 0, len(legacy))
	for _, tableName := range legacy {
		tables = append(tables, PlatformKnowledgeSemanticModelTableRef{
			DBName:    strings.TrimSpace(dbName),
			TableName: tableName,
		})
	}
	return platformKnowledgeCompactTableRefs(tables), nil
}

func platformKnowledgeCompactTableRefs(values []PlatformKnowledgeSemanticModelTableRef) []PlatformKnowledgeSemanticModelTableRef {
	out := make([]PlatformKnowledgeSemanticModelTableRef, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		dbName := strings.TrimSpace(value.DBName)
		tableName := strings.TrimSpace(value.TableName)
		if tableName == "" {
			continue
		}
		key := dbName + "\x00" + tableName
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, PlatformKnowledgeSemanticModelTableRef{
			DBName:    dbName,
			TableName: tableName,
		})
	}
	return out
}

func catalogKnowledgeUserDB(ctx context.Context, connPool tenant.ConnectionPool, userConnPool tenant.UserConnectionPool, workspaceID string, userID string) (*sql.DB, error) {
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return nil, fmt.Errorf("workspace_id is required for knowledge database access")
	}
	if userConnPool != nil {
		userID = strings.TrimSpace(userID)
		if userID == "" {
			return nil, fmt.Errorf("user_id is required for knowledge database access")
		}
		return userConnPool.GetUserConnection(ctx, workspaceID, userID)
	}
	if connPool == nil {
		return nil, fmt.Errorf("connection pool is not configured")
	}
	return connPool.GetConnection(ctx, workspaceID)
}

func platformKnowledgeUseDatabaseSQL(database string) (string, error) {
	database = strings.TrimSpace(database)
	if database == "" {
		return "", fmt.Errorf("database is required")
	}
	if strings.ContainsRune(database, 0) {
		return "", fmt.Errorf("database contains NUL byte")
	}
	return "USE " + platformKnowledgeQuoteIdentifier(database), nil
}

func platformKnowledgeQuoteIdentifier(value string) string {
	return "`" + strings.ReplaceAll(value, "`", "``") + "`"
}

func platformKnowledgeScanExecutionRows(rows *sql.Rows) (*knowledge.SQLExecutionResult, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("catalog knowledge sql executor: read columns: %w", err)
	}
	result := &knowledge.SQLExecutionResult{
		Columns: columns,
		Rows:    [][]any{},
	}
	for rows.Next() {
		values := make([]any, len(columns))
		dest := make([]any, len(columns))
		for i := range values {
			dest[i] = &values[i]
		}
		if err := rows.Scan(dest...); err != nil {
			return nil, fmt.Errorf("catalog knowledge sql executor: scan row: %w", err)
		}
		row := make([]any, len(columns))
		for i := range values {
			row[i] = platformKnowledgeSQLValue(values[i])
		}
		result.Rows = append(result.Rows, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("catalog knowledge sql executor: iterate rows: %w", err)
	}
	result.TotalCount = len(result.Rows)
	return result, nil
}

func platformKnowledgeSQLValue(value any) any {
	switch typed := value.(type) {
	case nil:
		return nil
	case []byte:
		return string(typed)
	case time.Time:
		return typed.Format(time.RFC3339Nano)
	default:
		return typed
	}
}

func platformKnowledgeValueString(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(typed)
	case []byte:
		return strings.TrimSpace(string(typed))
	case fmt.Stringer:
		return strings.TrimSpace(typed.String())
	case int:
		return strconv.Itoa(typed)
	case int64:
		return strconv.FormatInt(typed, 10)
	case float64:
		if typed == float64(int64(typed)) {
			return strconv.FormatInt(int64(typed), 10)
		}
		return strconv.FormatFloat(typed, 'f', -1, 64)
	default:
		return strings.TrimSpace(fmt.Sprint(typed))
	}
}

func platformKnowledgeIntFromMeta(meta map[string]any, key string) int {
	if meta == nil {
		return 0
	}
	switch typed := meta[key].(type) {
	case int:
		return typed
	case int8:
		return int(typed)
	case int16:
		return int(typed)
	case int32:
		return int(typed)
	case int64:
		return int(typed)
	case uint:
		return int(typed)
	case uint8:
		return int(typed)
	case uint16:
		return int(typed)
	case uint32:
		return int(typed)
	case uint64:
		return int(typed)
	case float32:
		n := int(typed)
		if float32(n) == typed {
			return n
		}
	case float64:
		n := int(typed)
		if float64(n) == typed {
			return n
		}
	case json.Number:
		n, _ := strconv.Atoi(typed.String())
		return n
	case string:
		n, _ := strconv.Atoi(strings.TrimSpace(typed))
		return n
	}
	return 0
}

func platformKnowledgeCompactStrings(values []string) []string {
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

func platformKnowledgeStringSet(values []string) map[string]struct{} {
	values = platformKnowledgeCompactStrings(values)
	out := make(map[string]struct{}, len(values))
	for _, value := range values {
		out[value] = struct{}{}
	}
	return out
}

func platformKnowledgeCompactInt64s(values []int64) []int64 {
	out := make([]int64, 0, len(values))
	seen := make(map[int64]struct{}, len(values))
	for _, value := range values {
		if value <= 0 {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func platformKnowledgeScanStringRows(rows *sql.Rows) ([]string, error) {
	out := make([]string, 0)
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			return nil, err
		}
		out = append(out, value)
	}
	return platformKnowledgeCompactStrings(out), rows.Err()
}

func platformKnowledgeParseVolumeIDArgs(values []string) ([]any, error) {
	values = platformKnowledgeCompactStrings(values)
	out := make([]any, 0, len(values))
	for _, value := range values {
		id, err := strconv.ParseInt(value, 10, 64)
		if err != nil || id <= 0 {
			return nil, fmt.Errorf("invalid volume_id %q", value)
		}
		out = append(out, id)
	}
	return out, nil
}

func platformKnowledgeStringsToAny(values []string) []any {
	out := make([]any, 0, len(values))
	for _, value := range values {
		out = append(out, value)
	}
	return out
}

func platformKnowledgeInt64sToAny(values []int64) []any {
	out := make([]any, 0, len(values))
	for _, value := range values {
		out = append(out, value)
	}
	return out
}

var (
	_ knowledge.SQLExecutor                  = (*catalogKnowledgeSQLExecutor)(nil)
	_ knowledge.SQLMutationExecutor          = (*catalogKnowledgeSQLExecutor)(nil)
	_ knowledge.EmbeddingService             = (*catalogKnowledgeEmbeddingClient)(nil)
	_ knowledgeservice.SchemaReader          = (*catalogKnowledgeSchemaReader)(nil)
	_ knowledgeservice.VisualSearchBackend   = (*catalogKnowledgeVisualBackend)(nil)
	_ knowledgeservice.ParsedMarkdownBackend = (*catalogParsedMarkdownBackend)(nil)
	_ PlatformKnowledgeSourceGovernanceStore = (*catalogKnowledgeSourceGovernanceStore)(nil)
)
