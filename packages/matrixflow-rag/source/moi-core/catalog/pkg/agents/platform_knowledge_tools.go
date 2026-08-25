package agents

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	agentruntimev2 "github.com/matrixflow/moi-core/agent-runtime-v2"
	"github.com/matrixflow/moi-core/agent-tools/knowledge"
	knowledgeservice "github.com/matrixflow/moi-core/agent-tools/knowledge/service"
	"github.com/matrixflow/moi-core/catalog/pkg/agentruntime"
	embeddingpkg "github.com/matrixflow/moi-core/catalog/pkg/embedding"
	embeddingadapter "github.com/matrixflow/moi-core/catalog/pkg/embedding/adapter"
	"github.com/matrixflow/moi-core/catalog/pkg/service/filestorage"
	"github.com/matrixflow/moi-core/catalog/pkg/service/storage/tenant"
	"github.com/matrixflow/moi-core/catalog/pkg/service/storage/transaction"
)

type PlatformKnowledgeToolBackend struct {
	Registry *knowledge.Registry
	Resolver *PlatformKnowledgeScopeResolver
}

type PlatformKnowledgeFileService interface {
	Read(ctx context.Context, filePath string) ([]byte, error)
}

type PlatformKnowledgeToolOptions struct {
	SQL                 knowledge.SQLExecutor
	SemanticModelStore  PlatformKnowledgeSemanticModelStore
	SourceGovernance    PlatformKnowledgeSourceGovernanceStore
	VectorIndexResolver PlatformKnowledgeVectorIndexResolver
	ImageIndexResolver  PlatformKnowledgeImageIndexResolver
	LegacyIndexResolver PlatformKnowledgeLegacyIndexResolver

	EmbeddingCache           *embeddingpkg.ConfigCache
	EmbeddingRouter          *embeddingpkg.Router
	EmbeddingAdapterRegistry embeddingadapter.Registry
	DefaultEmbeddingModel    string

	ConnectionPool     tenant.ConnectionPool
	UserConnectionPool tenant.UserConnectionPool
	FileStorage        tenant.FileStorage
	FileService        PlatformKnowledgeFileService
}

const (
	platformKnowledgeSourceStatusSucceeded = "succeeded"
	platformKnowledgeSourceStatusPending   = "pending"
	platformKnowledgeGovernanceModeLegacy  = "legacy_compat"
)

func NewPlatformKnowledgeToolBackend(opts PlatformKnowledgeToolOptions) (*PlatformKnowledgeToolBackend, error) {
	sqlExecutor := opts.SQL
	if sqlExecutor == nil {
		if opts.ConnectionPool == nil && opts.UserConnectionPool == nil {
			return nil, fmt.Errorf("build platform knowledge tools: sql executor or database connection pool is required")
		}
		sqlExecutor = &catalogKnowledgeSQLExecutor{
			connPool:     opts.ConnectionPool,
			userConnPool: opts.UserConnectionPool,
		}
	}
	embedder, err := buildPlatformKnowledgeEmbedder(opts)
	if err != nil {
		return nil, err
	}
	parsedMarkdownBackend, err := buildPlatformParsedMarkdownBackend(opts)
	if err != nil {
		return nil, err
	}
	sourceGovernanceStore := opts.SourceGovernance
	if sourceGovernanceStore == nil && opts.ConnectionPool != nil {
		sourceGovernanceStore = newCatalogKnowledgeSourceGovernanceStore(opts.ConnectionPool)
	}
	var tableGovernanceStore PlatformKnowledgeTableSourceGovernanceStore
	if typed, ok := sourceGovernanceStore.(PlatformKnowledgeTableSourceGovernanceStore); ok {
		tableGovernanceStore = typed
	}
	mutationExecutor, _ := sqlExecutor.(knowledge.SQLMutationExecutor)
	registry := knowledgeservice.NewRegistry(knowledgeservice.Deps{
		SQLExecutor:         sqlExecutor,
		SQLMutationExecutor: mutationExecutor,
		QuerySQLHooks: knowledgeservice.QuerySQLHooks{
			TableRefs: platformKnowledgeSourceTableRefs(tableGovernanceStore),
		},
		SchemaReader: &catalogKnowledgeSchemaReader{
			connPool:     opts.ConnectionPool,
			userConnPool: opts.UserConnectionPool,
		},
		VisualSearchBackend: &catalogKnowledgeVisualBackend{
			configCache:     opts.EmbeddingCache,
			router:          opts.EmbeddingRouter,
			adapterRegistry: opts.EmbeddingAdapterRegistry,
			connPool:        opts.ConnectionPool,
			fileStorage:     opts.FileStorage,
			fileService:     opts.FileService,
		},
		ParsedMarkdownBackend: parsedMarkdownBackend,
		Embedder:              embedder,
		DefaultRetrieverConfig: knowledge.RetrieverConfig{
			EmbeddingModel: strings.TrimSpace(opts.DefaultEmbeddingModel),
		},
	})
	return &PlatformKnowledgeToolBackend{
		Registry: registry,
		Resolver: NewPlatformKnowledgeScopeResolver(opts),
	}, nil
}

func PlatformKnowledgeToolExecutors(registry *knowledge.Registry, resolver *PlatformKnowledgeScopeResolver) map[string]agentruntime.PlatformToolExecutorFactory {
	if registry == nil {
		return nil
	}
	var runtimeResolver knowledge.RuntimeScopeResolver
	if resolver != nil {
		runtimeResolver = resolver
	}
	executors := knowledge.RuntimeToolExecutors(registry, runtimeResolver)
	out := make(map[string]agentruntime.PlatformToolExecutorFactory, len(executors))
	for kind, factory := range executors {
		factory := factory
		out[kind] = func(ctx context.Context, req agentruntime.ToolInvokeRequest, tool agentruntime.RuntimeToolSnapshotForExecutor) (agentruntimev2.Tool, error) {
			runtimeScope, _ := agentruntime.RuntimeRequestScopeFromContext(ctx)
			requestScope := knowledge.RuntimeRequestScope{
				WorkspaceID: strings.TrimSpace(runtimeScope.WorkspaceID),
				UserID:      strings.TrimSpace(runtimeScope.UserID),
			}
			if requestScope.WorkspaceID == "" {
				requestScope.WorkspaceID = strings.TrimSpace(req.WorkspaceID)
			}
			if requestScope.UserID == "" {
				requestScope.UserID = strings.TrimSpace(req.Caller.UserID)
			}
			return factory(ctx, knowledge.RuntimeToolRequest{
				WorkspaceID: req.WorkspaceID,
				Manifest: knowledge.RuntimeManifest{
					ID:          req.Manifest.ID,
					WorkspaceID: req.Manifest.WorkspaceID,
					Body:        req.Manifest.Body,
				},
				TurnMetadata: req.TurnMetadata,
				TurnParts:    req.TurnParts,
				RequestScope: requestScope,
				RunState:     agentruntime.RunScopedStateFromContext(ctx),
			}, knowledge.RuntimeToolSnapshot{
				ID:   tool.ID,
				Kind: tool.Kind,
			})
		}
	}
	return out
}

func platformKnowledgeSourceTableRefs(governance PlatformKnowledgeTableSourceGovernanceStore) func(context.Context, []string, string) ([]knowledge.TableRef, error) {
	if governance == nil {
		return nil
	}
	return func(ctx context.Context, tableNames []string, dbName string) ([]knowledge.TableRef, error) {
		scope := knowledge.ScopeFromContext(ctx)
		modelIDs := platformKnowledgeCompactInt64s(scope.SemanticModelIDs)
		if len(modelIDs) == 0 {
			return platformKnowledgeDefaultTableRefs(tableNames, dbName), nil
		}
		requested := make([]PlatformKnowledgeSemanticModelTableRef, 0, len(tableNames))
		defaultDB := strings.TrimSpace(dbName)
		knownDBs := platformKnowledgeScopeKnownDatabaseNames(append(append([]string(nil), tableNames...), scope.Tables...), defaultDB)
		for _, tableName := range tableNames {
			// Use defaultDB / known-DB-aware parse so bare dotted names like
			// test.csv stay intact instead of inventing a schema from dots.
			schema, name := platformKnowledgeParseTableIdentity(tableName, defaultDB, knownDBs...)
			if name == "" {
				name = strings.TrimSpace(tableName)
			}
			refDB := schema
			if refDB == "" {
				refDB = defaultDB
			}
			if refDB == "" || name == "" {
				continue
			}
			requested = append(requested, PlatformKnowledgeSemanticModelTableRef{DBName: refDB, TableName: name})
		}
		requested = platformKnowledgeCompactTableRefs(requested)
		if len(requested) == 0 {
			return nil, nil
		}
		out := make([]knowledge.TableRef, 0, len(requested))
		governed := make(map[string]struct{}, len(requested))
		for _, modelID := range modelIDs {
			records, err := governance.ListTableSourceGovernance(ctx, scope.WorkspaceID, modelID, requested)
			if err != nil {
				return nil, fmt.Errorf("query_sql table refs: resolve semantic model %d table sources: %w", modelID, err)
			}
			tableGovernance := newPlatformKnowledgeGovernanceByTable(records)
			for _, table := range requested {
				if tableGovernance.hasTable(table) {
					governed[platformKnowledgeTableRefKey(table)] = struct{}{}
				}
				if ref, ok := platformKnowledgeSourceTableRefFromRecords(records, table); ok {
					out = append(out, ref)
				}
			}
		}
		out = platformKnowledgeCompactKnowledgeTableRefs(out)
		for _, table := range requested {
			if _, ok := governed[platformKnowledgeTableRefKey(table)]; ok {
				continue
			}
			out = append(out, knowledge.TableRef{DBName: table.DBName, Name: table.TableName})
		}
		out = platformKnowledgeCompactKnowledgeTableRefs(out)
		if len(out) > 0 {
			return out, nil
		}
		return nil, nil
	}
}

func platformKnowledgeSourceTableRefFromRecords(records []PlatformKnowledgeTableSourceGovernanceRecord, table PlatformKnowledgeSemanticModelTableRef) (knowledge.TableRef, bool) {
	tableDBName := strings.TrimSpace(table.DBName)
	tableName := strings.TrimSpace(table.TableName)
	if tableName == "" {
		return knowledge.TableRef{}, false
	}
	for _, record := range records {
		if !record.ragReady() {
			continue
		}
		if !platformKnowledgeTableRecordMatches(record, tableDBName, tableName) {
			continue
		}
		sourceTable := strings.TrimSpace(record.SourceTableName)
		sourceDB := strings.TrimSpace(record.SourceDBName)
		if sourceTable == "" {
			sourceTable = strings.TrimSpace(record.TableName)
		}
		if sourceDB == "" {
			sourceDB = strings.TrimSpace(record.DBName)
		}
		if sourceTable == "" {
			return knowledge.TableRef{}, false
		}
		return knowledge.TableRef{DBName: sourceDB, Name: sourceTable}, true
	}
	return knowledge.TableRef{}, false
}

func platformKnowledgeTableRecordMatches(record PlatformKnowledgeTableSourceGovernanceRecord, dbName string, tableName string) bool {
	recordTableName := strings.TrimSpace(record.TableName)
	if recordTableName == "" || recordTableName != tableName {
		return false
	}
	recordDBName := strings.TrimSpace(record.DBName)
	if dbName == "" || recordDBName == "" {
		return dbName == recordDBName
	}
	return strings.EqualFold(dbName, recordDBName)
}

func platformKnowledgeDefaultTableRefs(tableNames []string, dbName string) []knowledge.TableRef {
	out := make([]knowledge.TableRef, 0, len(tableNames))
	dbName = strings.TrimSpace(dbName)
	knownDBs := platformKnowledgeScopeKnownDatabaseNames(tableNames, dbName)
	for _, tableName := range tableNames {
		tableName = strings.TrimSpace(tableName)
		if tableName == "" {
			continue
		}
		schema, name := platformKnowledgeParseTableIdentity(tableName, dbName, knownDBs...)
		if name == "" {
			name = tableName
		}
		refDB := schema
		if refDB == "" {
			refDB = dbName
		}
		if refDB == "" || name == "" {
			continue
		}
		out = append(out, knowledge.TableRef{DBName: refDB, Name: name})
	}
	return platformKnowledgeCompactKnowledgeTableRefs(out)
}

func platformKnowledgeCompactKnowledgeTableRefs(values []knowledge.TableRef) []knowledge.TableRef {
	out := make([]knowledge.TableRef, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		dbName := strings.TrimSpace(value.DBName)
		tableName := strings.TrimSpace(value.Name)
		if tableName == "" {
			continue
		}
		key := strings.ToLower(dbName) + "\x00" + strings.ToLower(tableName)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		value.DBName = dbName
		value.Name = tableName
		out = append(out, value)
	}
	return out
}

func platformKnowledgeTableRefKey(table PlatformKnowledgeSemanticModelTableRef) string {
	return strings.ToLower(strings.TrimSpace(table.DBName)) + "\x00" + strings.ToLower(strings.TrimSpace(table.TableName))
}

func buildPlatformKnowledgeEmbedder(opts PlatformKnowledgeToolOptions) (knowledge.EmbeddingService, error) {
	if opts.EmbeddingCache == nil || opts.EmbeddingRouter == nil || opts.EmbeddingAdapterRegistry == nil {
		return nil, fmt.Errorf("build platform knowledge tools: embedding router dependencies are required")
	}
	return &catalogKnowledgeEmbeddingClient{
		configCache:     opts.EmbeddingCache,
		router:          opts.EmbeddingRouter,
		adapterRegistry: opts.EmbeddingAdapterRegistry,
	}, nil
}

func buildPlatformParsedMarkdownBackend(opts PlatformKnowledgeToolOptions) (knowledgeservice.ParsedMarkdownBackend, error) {
	if opts.ConnectionPool == nil || opts.FileStorage == nil || opts.FileService == nil {
		return nil, fmt.Errorf("build platform knowledge tools: parsed markdown requires connection pool, file storage, and file service")
	}
	return &catalogParsedMarkdownBackend{
		connPool:    opts.ConnectionPool,
		fileStorage: opts.FileStorage,
		fileService: opts.FileService,
	}, nil
}

type catalogParsedMarkdownBackend struct {
	connPool    tenant.ConnectionPool
	fileStorage tenant.FileStorage
	fileService PlatformKnowledgeFileService
}

func (b *catalogParsedMarkdownBackend) LoadParsedMarkdown(ctx context.Context, scope knowledge.WorkspaceScope, markdownFileID string) (*knowledgeservice.ParsedMarkdownDocument, error) {
	if b == nil {
		return nil, fmt.Errorf("read parsed markdown: catalog parsed markdown backend is not configured")
	}
	if b.connPool == nil {
		return nil, fmt.Errorf("read parsed markdown: connection pool is not configured")
	}
	if b.fileStorage == nil {
		return nil, fmt.Errorf("read parsed markdown: file storage is not configured")
	}
	if b.fileService == nil {
		return nil, fmt.Errorf("read parsed markdown: file service is not configured")
	}
	workspaceID := strings.TrimSpace(scope.WorkspaceID)
	if workspaceID == "" {
		return nil, fmt.Errorf("read parsed markdown: workspace_id is required")
	}
	markdownFileID = strings.TrimSpace(markdownFileID)
	if markdownFileID == "" {
		return nil, fmt.Errorf("read parsed markdown: markdown_file_id is required")
	}
	tm, err := b.connPool.GetTransactionManager(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("read parsed markdown: get transaction manager workspace_id=%s: %w", workspaceID, err)
	}
	fileCtx := tenant.WithTransactionManager(ctx, tm)
	var (
		content  []byte
		fileName string
		metadata map[string]any
	)
	err = tm.RunInTx(fileCtx, func(txCtx context.Context) error {
		sourceFileIDs, err := b.ensureMarkdownAllowed(txCtx, tm, scope, markdownFileID)
		if err != nil {
			return err
		}
		fileMeta, err := b.fileStorage.GetFile(txCtx, markdownFileID)
		if err != nil {
			return fmt.Errorf("get file metadata markdown_file_id=%s: %w", markdownFileID, err)
		}
		filePath := filestorage.BuildPath(workspaceID, markdownFileID)
		data, err := b.fileService.Read(txCtx, filePath)
		if err != nil {
			return fmt.Errorf("read file path=%s: %w", filePath, err)
		}
		content = data
		fileName = fileMeta.GetOriginalName()
		metadata = map[string]any{
			"size_bytes": fileMeta.GetSize(),
			"file_path":  filePath,
		}
		if tags := platformKnowledgeSourceTagsForFileIDs(scope, sourceFileIDs); len(tags) > 0 {
			metadata["source_tags"] = tags
		}
		if fileMeta.GetOriginalName() != "" {
			metadata["original_name"] = fileMeta.GetOriginalName()
		}
		if fileMeta.GetMd5() != "" {
			metadata["md5"] = fileMeta.GetMd5()
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("read parsed markdown: %w", err)
	}
	return &knowledgeservice.ParsedMarkdownDocument{
		Content:  string(content),
		FileName: fileName,
		Metadata: metadata,
	}, nil
}

func (b *catalogParsedMarkdownBackend) ensureMarkdownAllowed(ctx context.Context, tm *transaction.Manager, scope knowledge.WorkspaceScope, markdownFileID string) ([]string, error) {
	scopes := platformKnowledgeMarkdownQueryScopes(scope)
	if len(scopes) == 0 {
		if len(platformKnowledgeScopeFileIDs(scope)) == 0 && len(platformKnowledgeScopeVolumeIDs(scope)) == 0 {
			if platformKnowledgeHasSemanticModelScope(scope) {
				return nil, fmt.Errorf("read parsed markdown: governed knowledge scope has no enabled files")
			}
			return nil, nil
		}
		return nil, fmt.Errorf("read parsed markdown: vector_table is required to validate markdown_file_id scope")
	}
	var fallbackErr error
	for _, queryScope := range scopes {
		sourceFileIDs, err := b.ensureMarkdownAllowedInScope(ctx, tm, queryScope, markdownFileID)
		if err == nil {
			return sourceFileIDs, nil
		}
		if fallbackErr == nil {
			fallbackErr = err
		}
		if errors.Is(err, errPlatformKnowledgeVectorIndexNotReady) ||
			strings.Contains(err.Error(), "markdown_file_id is not present in scoped vector index") ||
			strings.Contains(err.Error(), "markdown_file_id is outside the governed knowledge scope") {
			continue
		}
		return nil, err
	}
	if fallbackErr != nil {
		return nil, fallbackErr
	}
	return nil, fmt.Errorf("read parsed markdown: markdown_file_id is not present in scoped vector index")
}

func (b *catalogParsedMarkdownBackend) ensureMarkdownAllowedInScope(ctx context.Context, tm *transaction.Manager, scope knowledge.WorkspaceScope, markdownFileID string) ([]string, error) {
	allowedFileIDs := platformKnowledgeScopeFileIDs(scope)
	allowedVolumeIDs := platformKnowledgeScopeVolumeIDs(scope)
	hasSemanticScope := platformKnowledgeHasSemanticModelScope(scope)
	if len(allowedFileIDs) == 0 && len(allowedVolumeIDs) == 0 {
		if hasSemanticScope && strings.TrimSpace(scope.VectorTable) == "" && len(scope.RAGSources) == 0 {
			return nil, fmt.Errorf("read parsed markdown: governed knowledge scope has no enabled files")
		}
		if !hasSemanticScope {
			return nil, nil
		}
	}
	vectorTable := strings.TrimSpace(scope.VectorTable)
	if vectorTable == "" {
		vectorTable = knowledge.UniqueRAGSourceValue(scope.RAGSources, func(source knowledge.RAGSource) string {
			return source.VectorTable
		})
	}
	if vectorTable == "" {
		return nil, fmt.Errorf("read parsed markdown: vector_table is required to validate markdown_file_id scope")
	}
	quotedVectorTable, err := platformKnowledgeQuoteQualifiedIdentifier(vectorTable)
	if err != nil {
		return nil, fmt.Errorf("read parsed markdown: invalid vector_table %q", vectorTable)
	}
	executor := tm.Executor(ctx)
	markdownExpr, err := platformKnowledgeMarkdownFileIDExpr(ctx, executor, quotedVectorTable)
	if err != nil {
		return nil, err
	}
	volumeExpr, err := platformKnowledgeVolumeIDExpr(ctx, executor, quotedVectorTable)
	if err != nil {
		return nil, err
	}
	rows, err := executor.QueryContext(ctx,
		fmt.Sprintf(`SELECT DISTINCT file_id, %s AS volume_id FROM %s WHERE %s = ?`, volumeExpr, quotedVectorTable, markdownExpr),
		markdownFileID,
	)
	if err != nil {
		return nil, fmt.Errorf("read parsed markdown: query markdown source file_id: %w", err)
	}
	defer rows.Close()
	sourceRefs, err := platformKnowledgeScanMarkdownSourceRefs(rows)
	if err != nil {
		return nil, fmt.Errorf("read parsed markdown: scan markdown source file_id: %w", err)
	}
	if len(sourceRefs) == 0 {
		return nil, fmt.Errorf("read parsed markdown: markdown_file_id is not present in scoped vector index")
	}
	sourceFileIDs := platformKnowledgeMarkdownSourceFileIDs(sourceRefs)
	if matched := platformKnowledgeIntersectSourceFileIDs(sourceFileIDs, allowedFileIDs); len(matched) > 0 {
		return matched, nil
	}
	if platformKnowledgeMarkdownSourceMatchesVolume(sourceRefs, allowedVolumeIDs) {
		return sourceFileIDs, nil
	}
	if hasSemanticScope {
		matched, err := b.markdownAllowedBySemanticModelSources(ctx, tm, scope, sourceFileIDs)
		if err != nil {
			return nil, err
		}
		if len(matched) > 0 {
			return matched, nil
		}
	}
	return nil, fmt.Errorf("read parsed markdown: markdown_file_id is outside the governed knowledge scope")
}

func platformKnowledgeMarkdownQueryScopes(scope knowledge.WorkspaceScope) []knowledge.WorkspaceScope {
	if len(scope.RAGSources) == 0 {
		if strings.TrimSpace(scope.VectorTable) == "" {
			return nil
		}
		return []knowledge.WorkspaceScope{scope}
	}
	// Parsed markdown drilldown must validate against the vector index owned by
	// the matching KB source. Different associated KBs may use different text
	// vector tables, so a global unique vector_table check is too strict here.
	out := make([]knowledge.WorkspaceScope, 0, len(scope.RAGSources))
	for _, source := range scope.RAGSources {
		if strings.TrimSpace(source.VectorTable) == "" {
			continue
		}
		next := scope
		next.DBName = strings.TrimSpace(source.DBName)
		next.VectorTable = strings.TrimSpace(source.VectorTable)
		next.EmbeddingModel = strings.TrimSpace(source.EmbeddingModel)
		next.FileIDs = nil
		next.VolumeID = ""
		next.RAGSources = []knowledge.RAGSource{source}
		if source.SemanticModelID != 0 {
			next.SemanticModelIDs = []int64{source.SemanticModelID}
		}
		out = append(out, next)
	}
	if len(out) == 0 && strings.TrimSpace(scope.VectorTable) != "" {
		out = append(out, scope)
	}
	return out
}

func platformKnowledgeMarkdownFileIDExpr(ctx context.Context, executor interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}, quotedTable string) (string, error) {
	rows, err := executor.QueryContext(ctx, fmt.Sprintf("SHOW COLUMNS FROM %s", quotedTable))
	if err != nil {
		if platformKnowledgeIsNoSuchTableError(err) {
			return "", errPlatformKnowledgeVectorIndexNotReady
		}
		return "", fmt.Errorf("read parsed markdown: inspect vector index columns: %w", err)
	}
	defer rows.Close()
	columns := make(map[string]struct{})
	columnNames, err := rows.Columns()
	if err != nil {
		return "", fmt.Errorf("read parsed markdown: read vector index columns: %w", err)
	}
	for rows.Next() {
		values := make([]any, len(columnNames))
		dest := make([]any, len(values))
		for i := range values {
			dest[i] = &values[i]
		}
		if err := rows.Scan(dest...); err != nil {
			return "", fmt.Errorf("read parsed markdown: scan vector index columns: %w", err)
		}
		if len(values) > 0 {
			columns[strings.ToLower(strings.TrimSpace(platformKnowledgeValueString(values[0])))] = struct{}{}
		}
	}
	if err := rows.Err(); err != nil {
		return "", fmt.Errorf("read parsed markdown: iterate vector index columns: %w", err)
	}
	parts := []string{
		"NULLIF(JSON_UNQUOTE(JSON_EXTRACT(meta, '$.markdown_file_id')), '')",
		"NULLIF(JSON_UNQUOTE(JSON_EXTRACT(meta, '$.md_file_id')), '')",
	}
	if _, ok := columns["markdown_file_id"]; ok {
		parts = append([]string{"NULLIF(CAST(markdown_file_id AS CHAR), '')"}, parts...)
	}
	if _, ok := columns["md_file_id"]; ok {
		parts = append([]string{"NULLIF(CAST(md_file_id AS CHAR), '')"}, parts...)
	}
	return "COALESCE(" + strings.Join(parts, ", ") + ")", nil
}

func platformKnowledgeVolumeIDExpr(ctx context.Context, executor interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}, quotedTable string) (string, error) {
	rows, err := executor.QueryContext(ctx, fmt.Sprintf("SHOW COLUMNS FROM %s", quotedTable))
	if err != nil {
		if platformKnowledgeIsNoSuchTableError(err) {
			return "", errPlatformKnowledgeVectorIndexNotReady
		}
		return "", fmt.Errorf("read parsed markdown: inspect vector index columns: %w", err)
	}
	defer rows.Close()
	columnNames, err := rows.Columns()
	if err != nil {
		return "", fmt.Errorf("read parsed markdown: read vector index columns: %w", err)
	}
	for rows.Next() {
		values := make([]any, len(columnNames))
		dest := make([]any, len(values))
		for i := range values {
			dest[i] = &values[i]
		}
		if err := rows.Scan(dest...); err != nil {
			return "", fmt.Errorf("read parsed markdown: scan vector index columns: %w", err)
		}
		if len(values) > 0 && strings.EqualFold(strings.TrimSpace(platformKnowledgeValueString(values[0])), "volume_id") {
			return "NULLIF(CAST(volume_id AS CHAR), '')", nil
		}
	}
	if err := rows.Err(); err != nil {
		return "", fmt.Errorf("read parsed markdown: iterate vector index columns: %w", err)
	}
	return "''", nil
}

type platformKnowledgeMarkdownSourceRef struct {
	FileID   string
	VolumeID string
}

func platformKnowledgeScanMarkdownSourceRefs(rows *sql.Rows) ([]platformKnowledgeMarkdownSourceRef, error) {
	out := make([]platformKnowledgeMarkdownSourceRef, 0)
	for rows.Next() {
		var fileID, volumeID sql.NullString
		if err := rows.Scan(&fileID, &volumeID); err != nil {
			return nil, err
		}
		ref := platformKnowledgeMarkdownSourceRef{
			FileID:   strings.TrimSpace(fileID.String),
			VolumeID: strings.TrimSpace(volumeID.String),
		}
		if ref.FileID != "" || ref.VolumeID != "" {
			out = append(out, ref)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func platformKnowledgeMarkdownSourceFileIDs(refs []platformKnowledgeMarkdownSourceRef) []string {
	out := make([]string, 0, len(refs))
	for _, ref := range refs {
		out = append(out, ref.FileID)
	}
	return platformKnowledgeCompactStrings(out)
}

func platformKnowledgeIntersectSourceFileIDs(sourceFileIDs, allowedFileIDs []string) []string {
	sourceFileIDs = platformKnowledgeCompactStrings(sourceFileIDs)
	allowedFileIDs = platformKnowledgeCompactStrings(allowedFileIDs)
	if len(sourceFileIDs) == 0 || len(allowedFileIDs) == 0 {
		return nil
	}
	allowed := make(map[string]struct{}, len(allowedFileIDs))
	for _, fileID := range allowedFileIDs {
		allowed[fileID] = struct{}{}
	}
	matched := make([]string, 0, len(sourceFileIDs))
	for _, fileID := range sourceFileIDs {
		if _, ok := allowed[fileID]; ok {
			matched = append(matched, fileID)
		}
	}
	return platformKnowledgeCompactStrings(matched)
}

func platformKnowledgeMarkdownSourceMatchesVolume(refs []platformKnowledgeMarkdownSourceRef, allowedVolumeIDs []string) bool {
	allowedVolumeIDs = platformKnowledgeCompactStrings(allowedVolumeIDs)
	if len(refs) == 0 || len(allowedVolumeIDs) == 0 {
		return false
	}
	allowed := make(map[string]struct{}, len(allowedVolumeIDs))
	for _, volumeID := range allowedVolumeIDs {
		allowed[volumeID] = struct{}{}
	}
	for _, ref := range refs {
		if _, ok := allowed[strings.TrimSpace(ref.VolumeID)]; ok {
			return true
		}
	}
	return false
}

func (b *catalogParsedMarkdownBackend) markdownAllowedBySemanticModelSources(ctx context.Context, tm *transaction.Manager, scope knowledge.WorkspaceScope, sourceFileIDs []string) ([]string, error) {
	sourceFileIDs = platformKnowledgeCompactStrings(sourceFileIDs)
	if len(sourceFileIDs) == 0 {
		return nil, nil
	}
	if tm == nil {
		return nil, fmt.Errorf("read parsed markdown: transaction manager is required to validate semantic model source scope")
	}
	matched := make([]string, 0, len(sourceFileIDs))
	executor := tm.Executor(ctx)
	for _, modelID := range platformKnowledgeCompactInt64s(scope.SemanticModelIDs) {
		records, err := queryPlatformKnowledgeSourceGovernance(ctx, executor, modelID, sourceFileIDs, time.Now().Unix())
		if err != nil {
			return nil, err
		}
		for _, record := range records {
			if record.ragReady() {
				matched = append(matched, record.FileID)
			}
		}
	}
	return platformKnowledgeCompactStrings(matched), nil
}

func platformKnowledgeScopeFileIDs(scope knowledge.WorkspaceScope) []string {
	out := append([]string(nil), scope.FileIDs...)
	for _, source := range scope.RAGSources {
		out = append(out, source.FileIDs...)
	}
	return platformKnowledgeCompactStrings(out)
}

func platformKnowledgeScopeVolumeIDs(scope knowledge.WorkspaceScope) []string {
	out := []string{scope.VolumeID}
	for _, source := range scope.RAGSources {
		out = append(out, source.VolumeID)
	}
	return platformKnowledgeCompactStrings(out)
}

func platformKnowledgeHasSemanticModelScope(scope knowledge.WorkspaceScope) bool {
	if len(platformKnowledgeCompactInt64s(scope.SemanticModelIDs)) > 0 {
		return true
	}
	for _, source := range scope.RAGSources {
		if source.SemanticModelID != 0 {
			return true
		}
	}
	return false
}

func platformKnowledgeSourceTagsForFileIDs(scope knowledge.WorkspaceScope, fileIDs []string) []string {
	fileIDs = platformKnowledgeCompactStrings(fileIDs)
	if len(fileIDs) == 0 {
		return nil
	}
	out := make([]string, 0)
	for _, source := range scope.RAGSources {
		for _, fileID := range fileIDs {
			out = append(out, source.SourceTagsByFileID[fileID]...)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return append([]string(nil), out...)
}

var platformKnowledgeIdentifierPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func platformKnowledgeValidIdentifier(value string) bool {
	return platformKnowledgeIdentifierPattern.MatchString(strings.TrimSpace(value))
}

func platformKnowledgeQuoteQualifiedIdentifier(value string) (string, error) {
	parts := strings.Split(strings.TrimSpace(value), ".")
	if len(parts) == 0 || len(parts) > 2 {
		return "", fmt.Errorf("invalid qualified identifier")
	}
	quoted := make([]string, 0, len(parts))
	for _, part := range parts {
		if !platformKnowledgeValidIdentifier(part) {
			return "", fmt.Errorf("invalid qualified identifier")
		}
		quoted = append(quoted, platformKnowledgeQuoteIdentifier(part))
	}
	return strings.Join(quoted, "."), nil
}

// platformKnowledgeScopeTableName preserves db.table when the semantic model
// declares a database. Bare table names remain for legacy single-db scopes
// that only store table_names without db_name.
func platformKnowledgeScopeTableName(table PlatformKnowledgeSemanticModelTableRef) string {
	name := strings.TrimSpace(table.TableName)
	if name == "" {
		return ""
	}
	dbName := strings.TrimSpace(table.DBName)
	if dbName == "" {
		return name
	}
	return dbName + "." + name
}

// platformKnowledgeIncomingTableFilter captures turn-metadata table selection
// hints. Bare names match any database; qualified names match exact identity
// only (no bare fan-out from the table part of a qualified hint).
//
// active is true when the caller supplied non-empty scope.Tables and intends
// an intersection filter. An active filter with no matched bare/qualified keys
// yields empty tables (not "keep all model tables").
type platformKnowledgeIncomingTableFilter struct {
	active    bool
	bare      map[string]struct{}
	qualified map[string]struct{}
}

// newPlatformKnowledgeIncomingTableFilter builds selection hints from
// scope.Tables by matching against candidate model tables only. Unmatched
// strings are dropped — no defaultDB pairing or first-dot guessing.
// Qualified hints match exact identity only — sales.orders must not admit
// support.orders. Bare inputs (including dotted bare names like test.csv /
// support.csv) match every model table with the same TableName.
func newPlatformKnowledgeIncomingTableFilter(values []string, candidates []PlatformKnowledgeSemanticModelTableRef) platformKnowledgeIncomingTableFilter {
	out := platformKnowledgeIncomingTableFilter{
		bare:      make(map[string]struct{}),
		qualified: make(map[string]struct{}),
	}
	values = platformKnowledgeCompactStrings(values)
	if len(values) == 0 {
		return out
	}
	out.active = true
	for _, value := range values {
		hint := platformKnowledgeMatchIncomingTableHint(value, candidates)
		if hint.TableName == "" {
			continue
		}
		if hint.DBName != "" {
			// Exact identity when resolved to a database.
			out.qualified[strings.ToLower(hint.DBName)+"\x00"+strings.ToLower(hint.TableName)] = struct{}{}
		}
		if hint.Bare {
			// Bare selection: match any model table with this table name.
			out.bare[strings.ToLower(hint.TableName)] = struct{}{}
		}
	}
	return out
}

func platformKnowledgeFilterTableRefsByIncoming(tables []PlatformKnowledgeSemanticModelTableRef, filter platformKnowledgeIncomingTableFilter) []PlatformKnowledgeSemanticModelTableRef {
	if len(tables) == 0 {
		return tables
	}
	if !filter.active {
		return tables
	}
	out := make([]PlatformKnowledgeSemanticModelTableRef, 0, len(tables))
	for _, table := range tables {
		dbName := strings.TrimSpace(table.DBName)
		name := strings.TrimSpace(table.TableName)
		if name == "" {
			continue
		}
		if dbName != "" {
			key := strings.ToLower(dbName) + "\x00" + strings.ToLower(name)
			if _, ok := filter.qualified[key]; ok {
				out = append(out, table)
				continue
			}
		}
		if _, ok := filter.bare[strings.ToLower(name)]; ok {
			out = append(out, table)
		}
	}
	return out
}

type PlatformKnowledgeScopeResolver struct {
	semantic        PlatformKnowledgeSemanticModelStore
	governance      PlatformKnowledgeSourceGovernanceStore
	tableGovernance PlatformKnowledgeTableSourceGovernanceStore
	vector          PlatformKnowledgeVectorIndexResolver
	image           PlatformKnowledgeImageIndexResolver
	legacyIndex     PlatformKnowledgeLegacyIndexResolver
}

// NewPlatformKnowledgeScopeResolver resolves the concrete table bounds of a
// configured semantic source. It does not initialize embedding or RAG
// execution, so deterministic structured tools can use it independently from
// the full Knowledge tool backend.
func NewPlatformKnowledgeScopeResolver(opts PlatformKnowledgeToolOptions) *PlatformKnowledgeScopeResolver {
	semanticModelStore := opts.SemanticModelStore
	if semanticModelStore == nil && opts.ConnectionPool != nil {
		semanticModelStore = newCatalogKnowledgeSemanticModelStore(opts.ConnectionPool)
	}
	sourceGovernanceStore := opts.SourceGovernance
	if sourceGovernanceStore == nil && opts.ConnectionPool != nil {
		sourceGovernanceStore = newCatalogKnowledgeSourceGovernanceStore(opts.ConnectionPool)
	}
	var tableGovernanceStore PlatformKnowledgeTableSourceGovernanceStore
	if typed, ok := sourceGovernanceStore.(PlatformKnowledgeTableSourceGovernanceStore); ok {
		tableGovernanceStore = typed
	}
	vectorIndexResolver := opts.VectorIndexResolver
	if vectorIndexResolver == nil && opts.ConnectionPool != nil {
		vectorIndexResolver = newCatalogKnowledgeVectorIndexResolver(opts.ConnectionPool)
	}
	imageIndexResolver := opts.ImageIndexResolver
	if imageIndexResolver == nil && opts.ConnectionPool != nil {
		imageIndexResolver = newCatalogKnowledgeVectorIndexResolver(opts.ConnectionPool)
	}
	legacyIndexResolver := opts.LegacyIndexResolver
	if legacyIndexResolver == nil && opts.ConnectionPool != nil {
		legacyIndexResolver = newCatalogKnowledgeLegacyIndexResolver(opts.ConnectionPool)
	}
	return &PlatformKnowledgeScopeResolver{
		semantic:        semanticModelStore,
		governance:      sourceGovernanceStore,
		tableGovernance: tableGovernanceStore,
		vector:          vectorIndexResolver,
		image:           imageIndexResolver,
		legacyIndex:     legacyIndexResolver,
	}
}

func (r *PlatformKnowledgeScopeResolver) ResolveKnowledgeScope(ctx context.Context, scope knowledge.WorkspaceScope) (knowledge.WorkspaceScope, error) {
	modelIDs := platformKnowledgeCompactInt64s(scope.SemanticModelIDs)
	if len(modelIDs) == 0 {
		return knowledge.CompactScope(scope), nil
	}
	if r == nil || r.semantic == nil {
		return scope, fmt.Errorf("semantic model store is required to resolve knowledge scope")
	}
	workspaceID := strings.TrimSpace(scope.WorkspaceID)
	if workspaceID == "" {
		return scope, fmt.Errorf("workspace_id is required to resolve knowledge scope")
	}
	resolved := scope
	resolved.SemanticModelIDs = modelIDs
	ragSources := platformKnowledgeDropSemanticModelRAGSources(scope.RAGSources, modelIDs)
	allFileIDs := make([]string, 0)
	// Collect table refs first so multi-db scopes can emit db.table while
	// single-db scopes keep legacy bare table names + Scope.DBName.
	allTableRefs := make([]PlatformKnowledgeSemanticModelTableRef, 0)
	// Load models once. Candidate model tables let incoming parse match exact
	// identities and bare table names without guessing from database-name sets
	// (MF-8 other-db qualified; MF-10 dotted bare colliding with a DB name).
	type resolvedSemanticModel struct {
		model          *tenant.SemanticModelRecord
		modelTableRefs []PlatformKnowledgeSemanticModelTableRef
	}
	loadedModels := make([]resolvedSemanticModel, 0, len(modelIDs))
	candidateTables := make([]PlatformKnowledgeSemanticModelTableRef, 0)
	for _, modelID := range modelIDs {
		model, err := r.semantic.GetModel(ctx, workspaceID, modelID)
		if err != nil {
			return scope, fmt.Errorf("resolve semantic model %d: %w", modelID, err)
		}
		if model == nil {
			return scope, fmt.Errorf("resolve semantic model %d: empty model", modelID)
		}
		modelTableRefs, err := parseKnowledgeSemanticModelTableRefs(model.Tables, "")
		if err != nil {
			return scope, fmt.Errorf("resolve semantic model %d tables: %w", modelID, err)
		}
		candidateTables = append(candidateTables, modelTableRefs...)
		loadedModels = append(loadedModels, resolvedSemanticModel{
			model:          model,
			modelTableRefs: modelTableRefs,
		})
	}
	// Incoming scope.Tables may be:
	// 1) qualified database.table identities from manifest/upstream;
	// 2) bare names from frontend turn metadata (selection hints).
	// Match against candidate model tables only (exact db.table, then bare
	// TableName). No defaultDB pairing and no first-dot guessing — unmatched
	// strings are ignored so they cannot poison multi-db SQL scope.
	incomingTableFilter := newPlatformKnowledgeIncomingTableFilter(scope.Tables, candidateTables)
	// Collect every semantic-model table first. Scope.DBName is only a later
	// single-db convenience default / USE target — never a collection filter.
	// Filtering here would silently drop multi-db tables when metadata carries
	// a default database (production entrypoints allow metadata.database).
	// When scope.Tables is non-empty, filter model tables to the exact
	// intersection (qualified identity and/or bare TableName match).
	semanticTableDBNames := make([]string, 0, 1)
	for _, loaded := range loadedModels {
		model := loaded.model
		modelTableRefs := loaded.modelTableRefs
		if r.tableGovernance != nil && len(modelTableRefs) > 0 {
			records, err := r.tableGovernance.ListTableSourceGovernance(ctx, workspaceID, model.ID, modelTableRefs)
			if err != nil {
				return scope, fmt.Errorf("resolve semantic model %d table source governance: %w", model.ID, err)
			}
			modelTableRefs = newPlatformKnowledgeGovernanceByTable(records).filterRAGReadyOrLegacyTables(modelTableRefs)
		}
		// Apply bare/qualified selection hints from turn metadata without
		// re-introducing bare identities into the multi-db table set.
		if incomingTableFilter.active {
			modelTableRefs = platformKnowledgeFilterTableRefsByIncoming(modelTableRefs, incomingTableFilter)
		}
		for _, table := range modelTableRefs {
			allTableRefs = append(allTableRefs, table)
			tableDBName := strings.TrimSpace(table.DBName)
			knownDBName := false
			for _, dbName := range semanticTableDBNames {
				if strings.EqualFold(dbName, tableDBName) {
					knownDBName = true
					break
				}
			}
			if tableDBName != "" && !knownDBName {
				semanticTableDBNames = append(semanticTableDBNames, tableDBName)
			}
		}
		modelFiles, modelVolumeIDs, modelFileConfig, err := parseKnowledgeSemanticModelFiles(model.Files)
		if err != nil {
			return scope, fmt.Errorf("resolve semantic model %d files: %w", model.ID, err)
		}
		modelHadExplicitFileScope := len(modelFiles) > 0
		governance := platformKnowledgeGovernanceByFileID{}
		hasSourceGovernance := r.governance != nil
		legacyCompat := false
		sourceRecordCount := 0
		var sourceRecords []PlatformKnowledgeSourceGovernanceRecord
		if r.governance != nil {
			records, err := r.governance.ListSourceGovernance(ctx, workspaceID, model.ID, nil)
			if err != nil {
				return scope, fmt.Errorf("resolve semantic model %d source governance: %w", model.ID, err)
			}
			sourceRecords = records
			sourceRecordCount = len(records)
			governance = newPlatformKnowledgeGovernanceByFileID(records)
			legacyFileIDs := governance.legacyFileIDs(modelFiles)
			modelFiles = governance.filterRAGReadyOrLegacyFileIDs(modelFiles, records)
			legacyCompat = len(legacyFileIDs) > 0 && len(modelFiles) > 0
		}
		ragSource := knowledge.RAGSource{
			SemanticModelID:   model.ID,
			SemanticModelName: model.Name,
			FileIDs:           modelFiles,
			Metadata:          map[string]string{"source": "semantic_model"},
		}
		ragSource.VectorTable = modelFileConfig.VectorTable
		ragSource.EmbeddingModel = modelFileConfig.EmbeddingModel
		ragSource.ImageVectorTable = modelFileConfig.ImageVectorTable
		ragSource.ImageEmbeddingModel = modelFileConfig.ImageEmbeddingModel
		ragSource.ImageEmbeddingBackendID = modelFileConfig.ImageEmbeddingBackendID
		ragSource.ImageEmbeddingDimension = modelFileConfig.ImageEmbeddingDimension
		ragSource.ImagePreprocessVersion = modelFileConfig.ImagePreprocessVersion
		ragSource.ImageDistanceMetric = modelFileConfig.ImageDistanceMetric
		// Each semantic model/KB owns its document indexes. When multiple KBs
		// are associated, resolve each KB into its own RAG source instead of
		// inferring vector tables from shared file_id lineage.
		if !platformKnowledgeHasTextIndexBinding(ragSource.VectorTable, ragSource.EmbeddingModel) {
			ragSource.VectorTable = ""
			ragSource.EmbeddingModel = ""
		}
		if !platformKnowledgeHasImageIndexBinding(ragSource.ImageVectorTable, ragSource.ImageEmbeddingModel) {
			ragSource.ImageVectorTable = ""
			ragSource.ImageEmbeddingModel = ""
			ragSource.ImageEmbeddingBackendID = ""
			ragSource.ImageEmbeddingDimension = 0
			ragSource.ImagePreprocessVersion = ""
			ragSource.ImageDistanceMetric = ""
		}
		if modelFileConfig.ImageIndexConfigFromSet && modelFileConfig.ImageIndexConfigReady && !platformKnowledgeRAGSourceHasCompleteImageIndex(ragSource) {
			return scope, fmt.Errorf("resolve semantic model %d image index config %q: incomplete image index config", model.ID, modelFileConfig.ImageIndexConfigID)
		}
		pendingDefaultFileIDs, err := r.resolvePendingSourceDefaultFileIDs(ctx, workspaceID, ragSource.VectorTable, sourceRecords)
		if err != nil {
			return scope, fmt.Errorf("resolve semantic model %d pending source index files: %w", model.ID, err)
		}
		if len(pendingDefaultFileIDs) > 0 {
			modelFiles = append(modelFiles, pendingDefaultFileIDs...)
			modelFiles = platformKnowledgeCompactStrings(modelFiles)
			ragSource.FileIDs = modelFiles
		}
		// When file source governance exists but has no file rows, this may be:
		// 1. a legacy document KB whose files must be discovered from vector_table;
		// 2. a new table-only KB that still carries backend-owned vector_table;
		// 3. a mixed KB with queryable tables and a legacy document index.
		// Try the legacy lookup first so real legacy/mixed document scopes remain
		// queryable. If that lookup only reports the legacy vector index is not
		// ready, treat it as stale document metadata and skip document RAG.
		// Other lookup errors still fail.
		if hasSourceGovernance && sourceRecordCount == 0 && !modelHadExplicitFileScope && len(modelVolumeIDs) == 0 && len(modelFiles) == 0 && strings.TrimSpace(ragSource.VectorTable) != "" && r.legacyIndex != nil {
			legacyFileIDs, err := r.resolveMaterializedVectorFileIDs(ctx, workspaceID, ragSource.VectorTable)
			if err != nil {
				return scope, fmt.Errorf("resolve semantic model %d legacy index files: %w", model.ID, err)
			}
			if len(legacyFileIDs) > 0 {
				modelFiles = legacyFileIDs
				ragSource.FileIDs = modelFiles
				legacyCompat = true
			}
		}
		if legacyCompat {
			ragSource.Metadata["governance_mode"] = platformKnowledgeGovernanceModeLegacy
		}
		governance.applyToRAGSource(&ragSource)
		allFileIDs = append(allFileIDs, modelFiles...)
		if err := r.applyLegacyIndexConstraints(ctx, workspaceID, &ragSource); err != nil {
			return scope, fmt.Errorf("resolve semantic model %d legacy index versions: %w", model.ID, err)
		}
		// Image lineage only completes metadata for this semantic model's
		// configured image vector table/model; it must not discover another
		// KB's image index by shared file_id.
		if r.image != nil && !modelFileConfig.ImageIndexConfigFromSet && platformKnowledgeHasImageIndexBinding(ragSource.ImageVectorTable, ragSource.ImageEmbeddingModel) && !platformKnowledgeRAGSourceHasCompleteImageIndex(ragSource) {
			image, err := r.image.ResolveImageIndex(ctx, PlatformKnowledgeImageIndexResolveRequest{
				WorkspaceID:         workspaceID,
				UserID:              scope.UserID,
				SemanticModelID:     model.ID,
				ImageVectorTable:    ragSource.ImageVectorTable,
				ImageEmbeddingModel: ragSource.ImageEmbeddingModel,
				FileIDs:             modelFiles,
				VolumeIDs:           modelVolumeIDs,
			})
			if err != nil {
				return scope, fmt.Errorf("resolve semantic model %d image vector index: %w", model.ID, err)
			}
			if image != nil {
				ragSource.ImageVectorTable = strings.TrimSpace(image.ImageVectorTable)
				ragSource.ImageEmbeddingModel = strings.TrimSpace(image.ImageEmbeddingModel)
				ragSource.ImageEmbeddingBackendID = strings.TrimSpace(image.ImageEmbeddingBackendID)
				ragSource.ImageEmbeddingDimension = image.ImageEmbeddingDimension
				ragSource.ImagePreprocessVersion = strings.TrimSpace(image.ImagePreprocessVersion)
				ragSource.ImageDistanceMetric = strings.TrimSpace(image.ImageDistanceMetric)
				for k, v := range image.Metadata {
					if strings.TrimSpace(k) != "" && strings.TrimSpace(v) != "" {
						ragSource.Metadata[strings.TrimSpace(k)] = strings.TrimSpace(v)
					}
				}
			}
		}
		hasDocumentIndex := platformKnowledgeRAGSourceHasTextIndex(ragSource) || platformKnowledgeRAGSourceHasCompleteImageIndex(ragSource)
		hasFileRAGSource := (len(ragSource.FileIDs) > 0 && hasDocumentIndex) || (!hasSourceGovernance && len(modelVolumeIDs) == 0 && !modelHadExplicitFileScope && hasDocumentIndex)
		if hasFileRAGSource {
			ragSources = append(ragSources, ragSource)
		}
		for _, volumeID := range modelVolumeIDs {
			volumeSource := ragSource
			volumeSource.FileIDs = nil
			volumeSource.VolumeID = volumeID
			if volumeSource.VolumeID != "" && (platformKnowledgeRAGSourceHasTextIndex(volumeSource) || platformKnowledgeRAGSourceHasCompleteImageIndex(volumeSource)) {
				ragSources = append(ragSources, volumeSource)
			}
		}
	}
	// Binding-only path: semantic model tables list is empty but CatalogAssetRefs
	// already seeded fully-qualified database.table identities. Decode only via
	// known multi-db prefixes (2+ distinct left segments) or Scope.DBName —
	// never first-dot guess a lone bare physical name into a database.
	if len(allTableRefs) == 0 && len(candidateTables) == 0 {
		bindingTables := platformKnowledgeCompactStrings(scope.Tables)
		bindingDefaultDB := strings.TrimSpace(resolved.DBName)
		// Multi-db labels must not be re-paired under the first-ref convenience
		// defaultDB (would turn support.tickets into sales.support.tickets).
		if lefts := platformKnowledgeScopeLeftDatabaseSegments(bindingTables); len(lefts) >= 2 {
			bindingDefaultDB = ""
		}
		bindingKnownDBs := platformKnowledgeScopeKnownDatabaseNames(bindingTables, bindingDefaultDB)
		for _, tableName := range bindingTables {
			schema, name := platformKnowledgeParseTableIdentity(tableName, bindingDefaultDB, bindingKnownDBs...)
			if schema == "" || name == "" {
				continue
			}
			allTableRefs = append(allTableRefs, PlatformKnowledgeSemanticModelTableRef{
				DBName:    schema,
				TableName: name,
			})
			knownDBName := false
			for _, dbName := range semanticTableDBNames {
				if strings.EqualFold(dbName, schema) {
					knownDBName = true
					break
				}
			}
			if !knownDBName {
				semanticTableDBNames = append(semanticTableDBNames, schema)
			}
		}
	}
	// Table identity is always database.table when the semantic model / scope
	// provides a database. Scope.DBName is only a convenience default for bare
	// legacy names and optional USE database — never the table identity itself.
	tableDBNames := append([]string(nil), semanticTableDBNames...)
	for _, table := range allTableRefs {
		tableDBName := strings.TrimSpace(table.DBName)
		if tableDBName == "" {
			continue
		}
		known := false
		for _, dbName := range tableDBNames {
			if strings.EqualFold(dbName, tableDBName) {
				known = true
				break
			}
		}
		if !known {
			tableDBNames = append(tableDBNames, tableDBName)
		}
	}
	switch len(tableDBNames) {
	case 1:
		// Single database across collected tables: keep metadata default when it
		// matches, otherwise adopt the only known database.
		if strings.TrimSpace(resolved.DBName) == "" || strings.EqualFold(strings.TrimSpace(resolved.DBName), tableDBNames[0]) {
			resolved.DBName = tableDBNames[0]
		} else {
			// Metadata default disagrees with the only model database; prefer model.
			resolved.DBName = tableDBNames[0]
		}
	case 0:
		// No database known yet; bare legacy tables stay bare. Keep incoming
		// Scope.DBName as optional USE default for bare names.
	default:
		// Cross-database scopes must not keep a single default that would hide
		// other databases or make bare SQL ambiguous.
		resolved.DBName = ""
	}
	allTables := make([]string, 0, len(allTableRefs))
	for _, table := range allTableRefs {
		// Prefer db.table whenever DBName is known (from the table ref or the
		// single-db default). Multiple KBs pointing at the same physical table
		// collapse later via platformKnowledgeCompactStrings.
		if name := platformKnowledgeScopeTableName(table); name != "" {
			// If the ref has no DBName but the whole scope resolved to one DB,
			// still emit database.table so callers always see the full identity.
			if strings.TrimSpace(table.DBName) == "" {
				if dbName := strings.TrimSpace(resolved.DBName); dbName != "" {
					if bare := strings.TrimSpace(table.TableName); bare != "" {
						allTables = append(allTables, dbName+"."+bare)
						continue
					}
				}
				// Multi-db / no default: bare names are not addressable — skip.
				// Leaving them would disable SQL tools for the whole scope.
				continue
			}
			allTables = append(allTables, name)
		}
	}
	resolved.RAGSources = knowledge.CompactRAGSources(ragSources)
	resolved.FileIDs = platformKnowledgeCompactStrings(allFileIDs)
	resolved.Tables = platformKnowledgeCompactStrings(allTables)
	resolved.VectorTable = knowledge.UniqueRAGSourceValue(resolved.RAGSources, func(source knowledge.RAGSource) string {
		return source.VectorTable
	})
	resolved.EmbeddingModel = knowledge.UniqueRAGSourceValue(resolved.RAGSources, func(source knowledge.RAGSource) string {
		return source.EmbeddingModel
	})
	resolved.ImageVectorTable = knowledge.UniqueRAGSourceValue(resolved.RAGSources, func(source knowledge.RAGSource) string {
		return source.ImageVectorTable
	})
	resolved.ImageEmbeddingModel = knowledge.UniqueRAGSourceValue(resolved.RAGSources, func(source knowledge.RAGSource) string {
		return source.ImageEmbeddingModel
	})
	resolved.ImageEmbeddingBackendID = knowledge.UniqueRAGSourceValue(resolved.RAGSources, func(source knowledge.RAGSource) string {
		return source.ImageEmbeddingBackendID
	})
	resolved.ImageEmbeddingDimension = knowledge.UniqueRAGSourceIntValue(resolved.RAGSources, func(source knowledge.RAGSource) int {
		return source.ImageEmbeddingDimension
	})
	resolved.ImagePreprocessVersion = knowledge.UniqueRAGSourceValue(resolved.RAGSources, func(source knowledge.RAGSource) string {
		return source.ImagePreprocessVersion
	})
	resolved.ImageDistanceMetric = knowledge.UniqueRAGSourceValue(resolved.RAGSources, func(source knowledge.RAGSource) string {
		return source.ImageDistanceMetric
	})
	return knowledge.CompactScope(resolved), nil
}

// Pending sources have not published governance constraints yet. They may use
// the default file scope only after their own vector table has materialized a
// matching active chunk.
func (r *PlatformKnowledgeScopeResolver) resolvePendingSourceDefaultFileIDs(ctx context.Context, workspaceID, vectorTable string, records []PlatformKnowledgeSourceGovernanceRecord) ([]string, error) {
	pendingFileIDs := platformKnowledgePendingUnpublishedFileIDs(records)
	if len(pendingFileIDs) == 0 || strings.TrimSpace(vectorTable) == "" {
		return nil, nil
	}
	materializedFileIDs, err := r.resolveMaterializedVectorFileIDs(ctx, workspaceID, vectorTable)
	if err != nil {
		return nil, err
	}
	return platformKnowledgeIntersectFileIDs(pendingFileIDs, materializedFileIDs), nil
}

func (r *PlatformKnowledgeScopeResolver) resolveMaterializedVectorFileIDs(ctx context.Context, workspaceID, vectorTable string) ([]string, error) {
	if r == nil || r.legacyIndex == nil || strings.TrimSpace(vectorTable) == "" {
		return nil, nil
	}
	fileIDs, err := r.legacyIndex.ResolveLegacyIndexFileIDs(ctx, PlatformKnowledgeLegacyIndexResolveRequest{
		WorkspaceID: workspaceID,
		VectorTable: vectorTable,
	})
	if errors.Is(err, errPlatformKnowledgeVectorIndexNotReady) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return platformKnowledgeCompactStrings(fileIDs), nil
}

func (r *PlatformKnowledgeScopeResolver) applyLegacyIndexConstraints(ctx context.Context, workspaceID string, source *knowledge.RAGSource) error {
	if r == nil || r.legacyIndex == nil || source == nil {
		return nil
	}
	fileIDs := platformKnowledgeFileIDsWithoutIndexConstraint(source.FileIDs, source.IndexVersionConstraintByFileID, source.CurrentIndexVersionByFileID)
	if len(fileIDs) == 0 || strings.TrimSpace(source.VectorTable) == "" {
		return nil
	}
	constraints, err := r.legacyIndex.ResolveLegacyIndexVersions(ctx, PlatformKnowledgeLegacyIndexResolveRequest{
		WorkspaceID: workspaceID,
		VectorTable: source.VectorTable,
		FileIDs:     fileIDs,
	})
	if err != nil {
		// #12764: a file with no governance record (e.g. a workflow-built KB
		// bound purely via RegisterLineage) legitimately has no known
		// index_version yet when the physical legacy chunk table has not
		// materialized. That is not a hard failure — it just means there is
		// no legacy index-version constraint to apply, mirroring how the
		// sibling ResolveLegacyIndexFileIDs lookup already tolerates this
		// sentinel above.
		if errors.Is(err, errPlatformKnowledgeVectorIndexNotReady) {
			return nil
		}
		return err
	}
	if len(constraints) == 0 {
		return nil
	}
	if source.IndexVersionConstraintByFileID == nil {
		source.IndexVersionConstraintByFileID = map[string]knowledge.RAGIndexVersionConstraint{}
	}
	for fileID, constraint := range constraints {
		fileID = strings.TrimSpace(fileID)
		if fileID == "" {
			continue
		}
		if _, exists := source.IndexVersionConstraintByFileID[fileID]; exists {
			continue
		}
		switch constraint.Kind {
		case knowledge.RAGIndexVersionConstraintValue:
			source.IndexVersionConstraintByFileID[fileID] = knowledge.RAGIndexVersionConstraint{
				Kind:  knowledge.RAGIndexVersionConstraintValue,
				Value: constraint.Value,
			}
		case knowledge.RAGIndexVersionConstraintNull:
			source.IndexVersionConstraintByFileID[fileID] = knowledge.RAGIndexVersionConstraint{Kind: knowledge.RAGIndexVersionConstraintNull}
		}
	}
	return nil
}

func platformKnowledgeFileIDsWithoutIndexConstraint(fileIDs []string, constraints map[string]knowledge.RAGIndexVersionConstraint, current map[string]int64) []string {
	fileIDs = platformKnowledgeCompactStrings(fileIDs)
	if len(fileIDs) == 0 {
		return nil
	}
	out := make([]string, 0, len(fileIDs))
	for _, fileID := range fileIDs {
		if constraint, ok := constraints[fileID]; ok {
			switch constraint.Kind {
			case knowledge.RAGIndexVersionConstraintValue, knowledge.RAGIndexVersionConstraintNull:
				continue
			}
		}
		if version, ok := current[fileID]; ok && version > 0 {
			continue
		}
		out = append(out, fileID)
	}
	return platformKnowledgeCompactStrings(out)
}

func platformKnowledgeDropSemanticModelRAGSources(sources []knowledge.RAGSource, modelIDs []int64) []knowledge.RAGSource {
	modelIDs = platformKnowledgeCompactInt64s(modelIDs)
	if len(modelIDs) == 0 || len(sources) == 0 {
		return append([]knowledge.RAGSource(nil), sources...)
	}
	semanticModels := make(map[int64]struct{}, len(modelIDs))
	for _, modelID := range modelIDs {
		semanticModels[modelID] = struct{}{}
	}
	out := make([]knowledge.RAGSource, 0, len(sources))
	for _, source := range sources {
		if _, ok := semanticModels[source.SemanticModelID]; ok {
			continue
		}
		out = append(out, source)
	}
	return out
}

func platformKnowledgeRAGSourceHasCompleteImageIndex(source knowledge.RAGSource) bool {
	return strings.TrimSpace(source.ImageVectorTable) != "" &&
		strings.TrimSpace(source.ImageEmbeddingModel) != "" &&
		source.ImageEmbeddingDimension > 0 &&
		strings.TrimSpace(source.ImagePreprocessVersion) != "" &&
		strings.TrimSpace(source.ImageDistanceMetric) != ""
}

func platformKnowledgeRAGSourceHasTextIndex(source knowledge.RAGSource) bool {
	return platformKnowledgeHasTextIndexBinding(source.VectorTable, source.EmbeddingModel)
}

func platformKnowledgeHasTextIndexBinding(vectorTable, embeddingModel string) bool {
	return strings.TrimSpace(vectorTable) != "" && strings.TrimSpace(embeddingModel) != ""
}

func platformKnowledgeHasImageIndexBinding(imageVectorTable, imageEmbeddingModel string) bool {
	return strings.TrimSpace(imageVectorTable) != "" && strings.TrimSpace(imageEmbeddingModel) != ""
}

func platformKnowledgeMergeFileIDs(fileIDs []string, records []PlatformKnowledgeSourceGovernanceRecord) []string {
	out := append([]string(nil), fileIDs...)
	for _, record := range records {
		out = append(out, record.FileID)
	}
	return platformKnowledgeCompactStrings(out)
}

type platformKnowledgeGovernanceByFileID map[string]PlatformKnowledgeSourceGovernanceRecord

func (r PlatformKnowledgeSourceGovernanceRecord) ragReady() bool {
	return r.Status == platformKnowledgeSourceStatusSucceeded && r.EffectiveEnabled
}

func platformKnowledgePendingUnpublishedFileIDs(records []PlatformKnowledgeSourceGovernanceRecord) []string {
	out := make([]string, 0, len(records))
	for _, record := range records {
		if record.Status != platformKnowledgeSourceStatusPending || !record.EffectiveEnabled {
			continue
		}
		out = append(out, record.FileID)
	}
	return platformKnowledgeCompactStrings(out)
}

func platformKnowledgeIntersectFileIDs(candidates, materialized []string) []string {
	materializedSet := platformKnowledgeStringSet(materialized)
	out := make([]string, 0, len(candidates))
	for _, fileID := range platformKnowledgeCompactStrings(candidates) {
		if _, ok := materializedSet[fileID]; ok {
			out = append(out, fileID)
		}
	}
	return out
}

func newPlatformKnowledgeGovernanceByFileID(records []PlatformKnowledgeSourceGovernanceRecord) platformKnowledgeGovernanceByFileID {
	out := make(platformKnowledgeGovernanceByFileID, len(records))
	for _, record := range records {
		fileID := strings.TrimSpace(record.FileID)
		if fileID == "" {
			continue
		}
		out[fileID] = record
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func (g platformKnowledgeGovernanceByFileID) filterRAGReadyOrLegacyFileIDs(explicitFileIDs []string, records []PlatformKnowledgeSourceGovernanceRecord) []string {
	fileIDs := platformKnowledgeMergeFileIDs(explicitFileIDs, records)
	fileIDs = platformKnowledgeCompactStrings(fileIDs)
	// #12764: a fileID with no governance record (empty g, or this specific
	// fileID absent from g — e.g. a workflow-built/legacy KB that registers no
	// knowledge_base_sources rows) is not "nothing enabled" by default; it
	// only stays authoritative when it was explicitly bound rather than
	// merged in from a (necessarily governed) record.
	explicit := platformKnowledgeStringSet(explicitFileIDs)
	out := make([]string, 0, len(fileIDs))
	for _, fileID := range fileIDs {
		record, ok := g[fileID]
		if ok {
			if record.ragReady() {
				out = append(out, fileID)
			}
			continue
		}
		if _, legacy := explicit[fileID]; legacy {
			out = append(out, fileID)
		}
	}
	return platformKnowledgeCompactStrings(out)
}

func (g platformKnowledgeGovernanceByFileID) legacyFileIDs(fileIDs []string) []string {
	fileIDs = platformKnowledgeCompactStrings(fileIDs)
	if len(fileIDs) == 0 {
		return nil
	}
	if len(g) == 0 {
		return fileIDs
	}
	out := make([]string, 0, len(fileIDs))
	for _, fileID := range fileIDs {
		if _, ok := g[fileID]; !ok {
			out = append(out, fileID)
		}
	}
	return platformKnowledgeCompactStrings(out)
}

func (g platformKnowledgeGovernanceByFileID) applyToRAGSource(source *knowledge.RAGSource) {
	if source == nil || len(g) == 0 || len(source.FileIDs) == 0 {
		return
	}
	sourceTagsByFileID := make(map[string][]string, len(source.FileIDs))
	sourceRowIDByFileID := make(map[string]string, len(source.FileIDs))
	currentSegmentVersionByFileID := make(map[string]string, len(source.FileIDs))
	currentIndexVersionByFileID := make(map[string]int64, len(source.FileIDs))
	indexVersionConstraintByFileID := make(map[string]knowledge.RAGIndexVersionConstraint, len(source.FileIDs))
	for _, fileID := range source.FileIDs {
		record, ok := g[fileID]
		if !ok || !record.ragReady() {
			continue
		}
		if record.SourceRowID != "" {
			source.SourceRowIDs = append(source.SourceRowIDs, record.SourceRowID)
			sourceRowIDByFileID[fileID] = record.SourceRowID
		}
		if len(record.Tags) > 0 {
			source.SourceTags = append(source.SourceTags, record.Tags...)
			sourceTagsByFileID[fileID] = append(sourceTagsByFileID[fileID], record.Tags...)
		}
		if record.SegmentVersionID != "" && (record.IndexVersionValid || record.IndexVersion > 0) {
			currentSegmentVersionByFileID[fileID] = record.SegmentVersionID
			indexVersionConstraintByFileID[fileID] = knowledge.RAGIndexVersionConstraint{
				Kind:  knowledge.RAGIndexVersionConstraintValue,
				Value: record.IndexVersion,
			}
			if record.IndexVersion > 0 {
				currentIndexVersionByFileID[fileID] = record.IndexVersion
			}
		}
	}
	source.SourceRowIDs = platformKnowledgeCompactStrings(source.SourceRowIDs)
	source.SourceRowIDByFileID = sourceRowIDByFileID
	source.SourceTagsByFileID = sourceTagsByFileID
	source.CurrentSegmentVersionByFileID = currentSegmentVersionByFileID
	source.CurrentIndexVersionByFileID = currentIndexVersionByFileID
	source.IndexVersionConstraintByFileID = indexVersionConstraintByFileID
	if len(source.SourceTags) > 0 {
		if source.Metadata == nil {
			source.Metadata = map[string]string{}
		}
		if _, ok := source.Metadata["source_tags"]; !ok {
			source.Metadata["source_tags"] = strings.Join(source.SourceTags, ",")
		}
	}
}

type platformKnowledgeGovernanceByTable []PlatformKnowledgeTableSourceGovernanceRecord

func (r PlatformKnowledgeTableSourceGovernanceRecord) ragReady() bool {
	return r.Status == platformKnowledgeSourceStatusSucceeded && r.EffectiveEnabled
}

func newPlatformKnowledgeGovernanceByTable(records []PlatformKnowledgeTableSourceGovernanceRecord) platformKnowledgeGovernanceByTable {
	if len(records) == 0 {
		return nil
	}
	out := make(platformKnowledgeGovernanceByTable, 0, len(records))
	for _, record := range records {
		if strings.TrimSpace(record.TableName) == "" {
			continue
		}
		out = append(out, record)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func (g platformKnowledgeGovernanceByTable) filterRAGReadyOrLegacyTables(tables []PlatformKnowledgeSemanticModelTableRef) []PlatformKnowledgeSemanticModelTableRef {
	tables = platformKnowledgeCompactTableRefs(tables)
	if len(g) == 0 {
		return tables
	}
	out := make([]PlatformKnowledgeSemanticModelTableRef, 0, len(tables))
	for _, table := range tables {
		if g.tableRAGReady(table) || !g.hasTable(table) {
			out = append(out, table)
		}
	}
	return platformKnowledgeCompactTableRefs(out)
}

func (g platformKnowledgeGovernanceByTable) tableRAGReady(table PlatformKnowledgeSemanticModelTableRef) bool {
	tableDBName := strings.TrimSpace(table.DBName)
	tableName := strings.TrimSpace(table.TableName)
	for _, record := range g {
		if record.ragReady() && platformKnowledgeTableRecordMatches(record, tableDBName, tableName) {
			return true
		}
	}
	return false
}

func (g platformKnowledgeGovernanceByTable) hasTable(table PlatformKnowledgeSemanticModelTableRef) bool {
	tableDBName := strings.TrimSpace(table.DBName)
	tableName := strings.TrimSpace(table.TableName)
	for _, record := range g {
		if platformKnowledgeTableRecordMatches(record, tableDBName, tableName) {
			return true
		}
	}
	return false
}

var _ interface {
	ResolveKnowledgeScope(context.Context, knowledge.WorkspaceScope) (knowledge.WorkspaceScope, error)
} = (*PlatformKnowledgeScopeResolver)(nil)
