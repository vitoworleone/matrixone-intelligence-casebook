package agentresource

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/matrixflow/moi-core/catalog/pkg/service/storage/tenant"
	"github.com/matrixflow/moi-core/catalog/pkg/service/storage/transaction"
)

type SemanticKnowledgeBaseResolver struct {
	pool            tenant.ConnectionPool
	storage         tenant.SemanticModelStorage
	runtimeToolRefs []AgentBindingResourceRef
}

var _ AgentPackageKnowledgeBaseRefResolver = (*SemanticKnowledgeBaseResolver)(nil)
var _ AgentBindingKnowledgeBaseBatchResolver = (*SemanticKnowledgeBaseResolver)(nil)

type SemanticKnowledgeBaseResolverOption func(*SemanticKnowledgeBaseResolver)

func WithSemanticKnowledgeBaseRuntimeToolRefs(refs []AgentBindingResourceRef) SemanticKnowledgeBaseResolverOption {
	return func(r *SemanticKnowledgeBaseResolver) {
		r.runtimeToolRefs = cloneAgentBindingResourceRefs(refs)
	}
}

func NewSemanticKnowledgeBaseResolver(pool tenant.ConnectionPool, storage tenant.SemanticModelStorage, opts ...SemanticKnowledgeBaseResolverOption) *SemanticKnowledgeBaseResolver {
	resolver := &SemanticKnowledgeBaseResolver{pool: pool, storage: storage}
	for _, opt := range opts {
		if opt != nil {
			opt(resolver)
		}
	}
	return resolver
}

func (r *SemanticKnowledgeBaseResolver) ResolveKnowledgeBase(ctx context.Context, workspaceID, kbID string) (*KnowledgeBase, error) {
	if r == nil || r.pool == nil || r.storage == nil {
		return nil, fmt.Errorf("semantic knowledge resolver is not configured")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := validateOpaqueID(workspaceID, "workspace_id", wrapInvalidKnowledgeBaseID); err != nil {
		return nil, err
	}
	modelID, err := strconv.ParseInt(strings.TrimSpace(kbID), 10, 64)
	if err != nil || modelID <= 0 {
		return nil, ErrKnowledgeBaseNotFound
	}
	tm, err := r.pool.GetTransactionManager(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("get semantic model transaction manager: %w", err)
	}
	ctx = tenant.WithTransactionManager(ctx, tm)

	var model *tenant.SemanticModelRecord
	var governance []semanticKnowledgeSourceGovernanceRecord
	err = tm.RunInTx(ctx, func(txCtx context.Context) error {
		var getErr error
		model, getErr = r.storage.GetSemanticModel(txCtx, modelID)
		if getErr != nil {
			return getErr
		}
		governance, getErr = querySemanticKnowledgeSourceGovernance(txCtx, tm.Executor(txCtx), modelID, time.Now().Unix())
		return getErr
	}, transaction.ReadOnly())
	if err != nil {
		if errors.Is(err, tenant.ErrSemanticModelNotFound) {
			return nil, ErrKnowledgeBaseNotFound
		}
		return nil, err
	}
	if model == nil {
		return nil, ErrKnowledgeBaseNotFound
	}
	return semanticModelRecordToKnowledgeBase(workspaceID, model, governance, r.runtimeToolRefs), nil
}

func (r *SemanticKnowledgeBaseResolver) ResolveKnowledgeBases(ctx context.Context, workspaceID string, kbIDs []string) (map[string]KnowledgeBase, error) {
	if r == nil || r.pool == nil || r.storage == nil {
		return nil, fmt.Errorf("semantic knowledge resolver is not configured")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := validateOpaqueID(workspaceID, "workspace_id", wrapInvalidKnowledgeBaseID); err != nil {
		return nil, err
	}
	requestedByModelID := make(map[int64][]string, len(kbIDs))
	modelIDs := make([]string, 0, len(kbIDs))
	for _, kbID := range kbIDs {
		modelID, err := strconv.ParseInt(strings.TrimSpace(kbID), 10, 64)
		if err != nil || modelID <= 0 {
			continue
		}
		if _, ok := requestedByModelID[modelID]; !ok {
			modelIDs = append(modelIDs, strconv.FormatInt(modelID, 10))
		}
		requestedByModelID[modelID] = append(requestedByModelID[modelID], kbID)
	}
	resolved := make(map[string]KnowledgeBase, len(kbIDs))
	if len(modelIDs) == 0 {
		return resolved, nil
	}
	tm, err := r.pool.GetTransactionManager(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("get semantic model transaction manager: %w", err)
	}
	ctx = tenant.WithTransactionManager(ctx, tm)

	var (
		models            []*tenant.SemanticModelRecord
		governanceByModel map[int64][]semanticKnowledgeSourceGovernanceRecord
	)
	err = tm.RunInTx(ctx, func(txCtx context.Context) error {
		var listErr error
		models, _, listErr = r.storage.ListSemanticModels(txCtx,
			tenant.WithPageSize(tenant.MaxPageSize),
			tenant.WithFilter("ids", modelIDs, false),
		)
		if listErr != nil {
			return listErr
		}
		resolvedModelIDs := make([]int64, 0, len(models))
		for _, model := range models {
			if model != nil {
				resolvedModelIDs = append(resolvedModelIDs, model.ID)
			}
		}
		var governanceErr error
		governanceByModel, governanceErr = querySemanticKnowledgeSourceGovernanceByModelIDs(txCtx, tm.Executor(txCtx), resolvedModelIDs, time.Now().Unix())
		return governanceErr
	}, transaction.ReadOnly())
	if err != nil {
		return nil, err
	}
	for _, model := range models {
		if model == nil {
			continue
		}
		kb := semanticModelRecordToKnowledgeBase(workspaceID, model, governanceByModel[model.ID], r.runtimeToolRefs)
		if kb == nil {
			continue
		}
		for _, requestedID := range requestedByModelID[model.ID] {
			resolved[requestedID] = *kb
		}
	}
	return resolved, nil
}

func (r *SemanticKnowledgeBaseResolver) ListKnowledgeBases(ctx context.Context, filter KnowledgeBaseListFilter) ([]KnowledgeBase, int, error) {
	if r == nil || r.pool == nil || r.storage == nil {
		return nil, 0, fmt.Errorf("semantic knowledge resolver is not configured")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := validateOpaqueID(filter.WorkspaceID, "workspace_id", wrapInvalidKnowledgeBaseID); err != nil {
		return nil, 0, err
	}
	if err := validateOptionalOpaqueID(filter.OwnerUserID, "owner_user_id", wrapInvalidKnowledgeBaseID); err != nil {
		return nil, 0, err
	}
	if filter.Status != "" && filter.Status != KnowledgeBaseStatusActive {
		return nil, 0, nil
	}
	if filter.SourceType != "" && filter.SourceType != KnowledgeBaseSourceCatalogResource {
		return nil, 0, nil
	}
	if filter.Visibility != "" && filter.Visibility != KnowledgeBaseVisibilityWorkspace {
		return nil, 0, nil
	}
	if filter.IndexStatus != "" && filter.IndexStatus != KnowledgeBaseIndexStatusReady {
		return nil, 0, nil
	}

	limit, offset := normalizeResourceListPagination(filter.Limit, filter.Offset)
	pageSize := int32(limit)
	pageToken := tenant.EncodePageToken(int64(offset))
	exactName := filter.Name != ""

	opts := []tenant.ListOption{tenant.WithPageSize(pageSize), tenant.WithPageToken(pageToken)}
	if exactName {
		opts = []tenant.ListOption{tenant.WithPageSize(tenant.MaxPageSize), tenant.WithFilter("search", []string{filter.Name}, true)}
	} else if filter.Query != "" {
		opts = append(opts, tenant.WithFilter("search", []string{filter.Query}, true))
	}

	tm, err := r.pool.GetTransactionManager(ctx, filter.WorkspaceID)
	if err != nil {
		return nil, 0, fmt.Errorf("get semantic model transaction manager: %w", err)
	}
	ctx = tenant.WithTransactionManager(ctx, tm)

	var (
		models            []*tenant.SemanticModelRecord
		total             int64
		governanceByModel map[int64][]semanticKnowledgeSourceGovernanceRecord
	)
	err = tm.RunInTx(ctx, func(txCtx context.Context) error {
		var listErr error
		models, total, listErr = r.storage.ListSemanticModels(txCtx, opts...)
		if listErr != nil {
			return listErr
		}
		modelIDs := make([]int64, 0, len(models))
		for _, model := range models {
			if model == nil {
				continue
			}
			if exactName && model.Name != filter.Name {
				continue
			}
			if filter.OwnerUserID != "" && model.CreatedBy != filter.OwnerUserID {
				continue
			}
			modelIDs = append(modelIDs, model.ID)
		}
		nowUnix := time.Now().Unix()
		executor := tm.Executor(txCtx)
		var governanceErr error
		governanceByModel, governanceErr = querySemanticKnowledgeSourceGovernanceByModelIDs(txCtx, executor, modelIDs, nowUnix)
		if governanceErr != nil {
			return governanceErr
		}
		return nil
	}, transaction.ReadOnly())
	if err != nil {
		return nil, 0, err
	}

	items := make([]KnowledgeBase, 0, len(models))
	for _, model := range models {
		if model == nil {
			continue
		}
		if exactName && model.Name != filter.Name {
			continue
		}
		if filter.OwnerUserID != "" && model.CreatedBy != filter.OwnerUserID {
			continue
		}
		kb := semanticModelRecordToKnowledgeBase(filter.WorkspaceID, model, governanceByModel[model.ID], r.runtimeToolRefs)
		if kb == nil {
			continue
		}
		items = append(items, *kb)
	}
	if exactName || filter.OwnerUserID != "" {
		return items, len(items), nil
	}
	return items, int(total), nil
}

func semanticModelRecordToKnowledgeBase(workspaceID string, model *tenant.SemanticModelRecord, governance []semanticKnowledgeSourceGovernanceRecord, runtimeToolRefs []AgentBindingResourceRef) *KnowledgeBase {
	if model == nil {
		return nil
	}
	updatedAt := time.Unix(model.UpdatedAt, 0).UTC()
	metadata := map[string]any{
		"resource_kind":      "semantic_model",
		"semantic_model_id":  model.ID,
		"table_set_hash":     model.TableSetHash,
		"created_by":         model.CreatedBy,
		"updated_by":         model.UpdatedBy,
		"knowledge_provider": "semantic_models",
	}
	if len(runtimeToolRefs) > 0 {
		metadata[KnowledgeBaseMetadataRuntimeToolRefsKey] = KnowledgeBaseRuntimeToolRefsMetadata(runtimeToolRefs)
	}
	kb := KnowledgeBase{
		ID:               strconv.FormatInt(model.ID, 10),
		WorkspaceID:      workspaceID,
		Name:             model.Name,
		Description:      model.Description,
		Status:           KnowledgeBaseStatusActive,
		SourceType:       KnowledgeBaseSourceCatalogResource,
		CatalogAssetRefs: semanticModelAssetRefs(model, governance),
		Visibility:       KnowledgeBaseVisibilityWorkspace,
		IndexStatus:      KnowledgeBaseIndexStatusReady,
		LastIndexedAt:    &updatedAt,
		Version:          1,
		Metadata:         metadata,
		CreatedBy:        model.CreatedBy,
		UpdatedBy:        model.UpdatedBy,
		CreatedAt:        time.Unix(model.CreatedAt, 0).UTC(),
		UpdatedAt:        updatedAt,
	}
	return &kb
}

func semanticModelAssetRefs(model *tenant.SemanticModelRecord, governance []semanticKnowledgeSourceGovernanceRecord) []KnowledgeCatalogAssetRef {
	if model == nil {
		return nil
	}
	result := make([]KnowledgeCatalogAssetRef, 0)
	result = append(result, semanticModelTableAssetRefs(model.Tables, governance)...)
	result = append(result, semanticModelFileAssetRefs(model.Files, governance)...)
	return result
}

func semanticModelTableAssetRefs(raw json.RawMessage, governance []semanticKnowledgeSourceGovernanceRecord) []KnowledgeCatalogAssetRef {
	if len(raw) == 0 {
		return nil
	}
	var tables []struct {
		DBName     string   `json:"db_name"`
		TableNames []string `json:"table_names"`
		Parents    []string `json:"parents"`
	}
	if err := json.Unmarshal(raw, &tables); err != nil {
		return nil
	}
	governanceByTable := newSemanticKnowledgeGovernanceByTable(governance)
	result := make([]KnowledgeCatalogAssetRef, 0)
	for _, table := range tables {
		dbName := strings.TrimSpace(table.DBName)
		for _, name := range table.TableNames {
			tableName := strings.TrimSpace(name)
			if tableName == "" {
				continue
			}
			if governanceByTable.tableDisabled(dbName, tableName) {
				continue
			}
			id := tableName
			if dbName != "" {
				id = dbName + "." + tableName
			}
			result = append(result, KnowledgeCatalogAssetRef{
				Type: "table",
				ID:   id,
				Config: compactAnyMap(map[string]any{
					"name":       tableName,
					"type":       "table",
					"db_name":    dbName,
					"table_name": tableName,
					"parents":    table.Parents,
				}),
			})
		}
	}
	return result
}

func semanticModelFileAssetRefs(raw json.RawMessage, governance []semanticKnowledgeSourceGovernanceRecord) []KnowledgeCatalogAssetRef {
	if len(raw) == 0 {
		return nil
	}
	var files struct {
		FileIDs   []string `json:"file_ids"`
		Parents   []string `json:"parents"`
		VolumeIDs []string `json:"volume_ids"`
		Volumes   []struct {
			VolumeID string   `json:"volume_id"`
			Parents  []string `json:"parents"`
			Path     []string `json:"path"`
		} `json:"volumes"`
	}
	if err := json.Unmarshal(raw, &files); err != nil {
		return nil
	}
	governanceByFileID := semanticKnowledgeGovernanceByFileID(governance)
	result := make([]KnowledgeCatalogAssetRef, 0, len(files.FileIDs)+len(files.VolumeIDs)+len(files.Volumes))
	for _, id := range files.FileIDs {
		fileID := strings.TrimSpace(id)
		if fileID == "" {
			continue
		}
		record, governed := governanceByFileID[fileID]
		if governed && !record.EffectiveEnabled {
			continue
		}
		result = append(result, KnowledgeCatalogAssetRef{
			Type: "file",
			ID:   fileID,
			Config: compactAnyMap(map[string]any{
				"name":          fileID,
				"type":          "file",
				"file_id":       fileID,
				"parents":       files.Parents,
				"source_row_id": record.SourceRowID,
				"source_tags":   record.Tags,
			}),
		})
	}
	for _, id := range files.VolumeIDs {
		volumeID := strings.TrimSpace(id)
		if volumeID == "" {
			continue
		}
		result = append(result, KnowledgeCatalogAssetRef{
			Type: "volume",
			ID:   volumeID,
			Config: compactAnyMap(map[string]any{
				"name":      volumeID,
				"type":      "volume",
				"volume_id": volumeID,
				"parents":   files.Parents,
			}),
		})
	}
	for _, volume := range files.Volumes {
		volumeID := strings.TrimSpace(volume.VolumeID)
		if volumeID == "" {
			continue
		}
		name := strings.Join(nonEmptyStrings(volume.Path), "/")
		if name == "" {
			name = volumeID
		}
		result = append(result, KnowledgeCatalogAssetRef{
			Type: "volume",
			ID:   volumeID,
			Config: compactAnyMap(map[string]any{
				"name":      name,
				"type":      "volume",
				"volume_id": volumeID,
				"parents":   volume.Parents,
				"path":      volume.Path,
			}),
		})
	}
	return result
}

type semanticKnowledgeSourceGovernanceRecord struct {
	SourceRowID             string
	FileID                  string
	DBName                  string
	TableName               string
	SourceTableID           string
	KBTableID               string
	EffectiveEnabled        bool
	ForceEnabledAfterExpiry bool
	Tags                    []string
}

func querySemanticKnowledgeSourceGovernance(ctx context.Context, executor interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}, modelID int64, nowUnix int64) ([]semanticKnowledgeSourceGovernanceRecord, error) {
	if modelID <= 0 {
		return nil, nil
	}
	governanceByModel, err := querySemanticKnowledgeSourceGovernanceByModelIDs(ctx, executor, []int64{modelID}, nowUnix)
	if err != nil {
		return nil, err
	}
	return governanceByModel[modelID], nil
}

func querySemanticKnowledgeSourceGovernanceByModelIDs(ctx context.Context, executor interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}, modelIDs []int64, nowUnix int64) (map[int64][]semanticKnowledgeSourceGovernanceRecord, error) {
	uniqueModelIDs := make([]int64, 0, len(modelIDs))
	seen := make(map[int64]struct{}, len(modelIDs))
	for _, modelID := range modelIDs {
		if modelID <= 0 {
			continue
		}
		if _, ok := seen[modelID]; ok {
			continue
		}
		seen[modelID] = struct{}{}
		uniqueModelIDs = append(uniqueModelIDs, modelID)
	}
	governanceByModel := make(map[int64][]semanticKnowledgeSourceGovernanceRecord, len(uniqueModelIDs))
	if len(uniqueModelIDs) == 0 {
		return governanceByModel, nil
	}
	placeholders := make([]string, len(uniqueModelIDs))
	args := make([]any, len(uniqueModelIDs))
	for index, modelID := range uniqueModelIDs {
		placeholders[index] = "?"
		args[index] = modelID
	}
	rows, err := executor.QueryContext(ctx, `SELECT
    kbs.model_id,
    kbs.source_id,
    COALESCE(kbs.kb_file_id, ''),
    COALESCE(NULLIF(kbs.db_name, ''), kbd.database_name, srcd.database_name, ''),
    COALESCE(NULLIF(kbs.table_name, ''), kbt.table_name, srct.table_name, ''),
    COALESCE(CAST(kbs.source_table_id AS CHAR), ''),
    COALESCE(CAST(kbs.kb_table_id AS CHAR), ''),
    COALESCE(kbs.enabled, 1),
    kbs.expires_at,
    kbs.force_enabled_after_expiry,
    kbs.tags
FROM knowledge_base_sources kbs
LEFT JOIN catalog_table kbt ON kbt.table_id = kbs.kb_table_id
LEFT JOIN catalog_database kbd ON kbd.database_id = kbt.database_id
LEFT JOIN catalog_table srct ON srct.table_id = kbs.source_table_id
LEFT JOIN catalog_database srcd ON srcd.database_id = srct.database_id
WHERE kbs.model_id IN (`+strings.Join(placeholders, ",")+`) AND kbs.source_type IN ('local_file', 'catalog_file', 'catalog_table') AND kbs.status <> 'removed'`, args...)
	if err != nil {
		return nil, fmt.Errorf("query semantic knowledge source governance: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var (
			modelID       int64
			sourceID      string
			fileID        string
			dbName        string
			tableName     string
			sourceTableID string
			kbTableID     string
			enabled       bool
			expiresAt     sql.NullInt64
			forceEnabled  bool
			tagsRaw       sql.NullString
		)
		if err := rows.Scan(&modelID, &sourceID, &fileID, &dbName, &tableName, &sourceTableID, &kbTableID, &enabled, &expiresAt, &forceEnabled, &tagsRaw); err != nil {
			return nil, fmt.Errorf("scan semantic knowledge source governance: %w", err)
		}
		expired := expiresAt.Valid && expiresAt.Int64 > 0 && nowUnix > expiresAt.Int64
		governanceByModel[modelID] = append(governanceByModel[modelID], semanticKnowledgeSourceGovernanceRecord{
			SourceRowID:             strings.TrimSpace(sourceID),
			FileID:                  strings.TrimSpace(fileID),
			DBName:                  strings.TrimSpace(dbName),
			TableName:               strings.TrimSpace(tableName),
			SourceTableID:           strings.TrimSpace(sourceTableID),
			KBTableID:               strings.TrimSpace(kbTableID),
			EffectiveEnabled:        enabled && (!expired || forceEnabled),
			ForceEnabledAfterExpiry: forceEnabled,
			Tags:                    parseSemanticKnowledgeSourceTags(tagsRaw),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate semantic knowledge source governance: %w", err)
	}
	return governanceByModel, nil
}

func semanticKnowledgeGovernanceByFileID(records []semanticKnowledgeSourceGovernanceRecord) map[string]semanticKnowledgeSourceGovernanceRecord {
	out := make(map[string]semanticKnowledgeSourceGovernanceRecord, len(records))
	for _, record := range records {
		fileID := strings.TrimSpace(record.FileID)
		if fileID == "" {
			continue
		}
		out[fileID] = record
	}
	return out
}

type semanticKnowledgeTableGovernance []semanticKnowledgeSourceGovernanceRecord

func newSemanticKnowledgeGovernanceByTable(records []semanticKnowledgeSourceGovernanceRecord) semanticKnowledgeTableGovernance {
	out := make(semanticKnowledgeTableGovernance, 0, len(records))
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

func (g semanticKnowledgeTableGovernance) tableDisabled(dbName string, tableName string) bool {
	dbName = strings.TrimSpace(dbName)
	tableName = strings.TrimSpace(tableName)
	for _, record := range g {
		if record.EffectiveEnabled || strings.TrimSpace(record.TableName) != tableName {
			continue
		}
		recordDBName := strings.TrimSpace(record.DBName)
		if dbName == "" || recordDBName == "" {
			if dbName == recordDBName {
				return true
			}
			continue
		}
		if strings.EqualFold(dbName, recordDBName) {
			return true
		}
	}
	return false
}

func parseSemanticKnowledgeSourceTags(raw sql.NullString) []string {
	if !raw.Valid || raw.String == "" {
		return nil
	}
	var tags []string
	if err := json.Unmarshal([]byte(raw.String), &tags); err != nil {
		return nil
	}
	return tags
}

func nonEmptyStrings(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func compactAnyMap(input map[string]any) map[string]any {
	result := make(map[string]any, len(input))
	for key, value := range input {
		switch typed := value.(type) {
		case string:
			if strings.TrimSpace(typed) == "" {
				continue
			}
		case []string:
			if len(typed) == 0 {
				continue
			}
		case nil:
			continue
		}
		result[key] = value
	}
	if len(result) == 0 {
		return nil
	}
	return result
}
