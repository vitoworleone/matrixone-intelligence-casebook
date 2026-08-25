package session

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	backendcatalog "github.com/matrixorigin/matrixflow/moi-backend/pkg/catalog"
	"github.com/matrixorigin/matrixflow/moi-backend/pkg/coreclient"
	"github.com/matrixorigin/matrixflow/moi-backend/pkg/ctxutil"
	"github.com/matrixorigin/matrixflow/moi-backend/pkg/logger"

	mysqlDriver "github.com/go-sql-driver/mysql"
	moi "github.com/matrixflow/moi-core/go-sdk"
	"github.com/matrixflow/moi-core/model/common"
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"
)

func (s *semanticModelService) ResolveLegacySourceIAMDependencies(ctx context.Context, tables, files json.RawMessage) ([]CreateSemanticModelSourceRequest, error) {
	// Legacy Create/Update tables/files payloads cannot carry volume_id.
	// catalog_file writes (and their volume IAM) go through sources /
	// source_selections. UpdateModel rejects files.file_ids before side effects.
	// CreateModel only stores model metadata and does not open catalog files.
	// Authorize table targets only; ignore file_ids for IAM resolution.
	if len(files) > 0 && string(files) != "null" {
		if _, err := semanticModelFileIDs(files); err != nil {
			return nil, err
		}
	}
	out := make([]CreateSemanticModelSourceRequest, 0)
	var tableGroups []semanticModelTableSource
	if len(tables) > 0 && string(tables) != "null" {
		if err := json.Unmarshal(tables, &tableGroups); err != nil {
			return nil, semanticModelTablesInvalidError()
		}
	}
	tableNames := make([]semanticModelTableName, 0)
	for _, group := range tableGroups {
		if strings.TrimSpace(group.DBName) == "" || len(group.TableNames) == 0 {
			return nil, semanticModelTablesInvalidError()
		}
		for _, tableName := range group.TableNames {
			if strings.TrimSpace(tableName) == "" {
				return nil, semanticModelTablesInvalidError()
			}
			tableNames = append(tableNames, semanticModelTableName{dbName: group.DBName, tableName: tableName})
		}
	}
	tableNames = uniqueSemanticModelTableNamesSorted(tableNames)
	resolved, err := batchResolveCatalogTableIDsByName(ctx, tableNames)
	if err != nil {
		return nil, err
	}
	if len(resolved) != len(tableNames) {
		return nil, fmt.Errorf("semantic model table dependency resolution is incomplete")
	}
	for _, tableName := range tableNames {
		tableID := resolved[tableName.key()]
		if tableID <= 0 {
			return nil, fmt.Errorf("semantic model table dependency is not canonical")
		}
		out = append(out, CreateSemanticModelSourceRequest{SourceType: kbSourceTypeCatalogTable, TableID: tableID})
	}
	return out, nil
}

func (s *semanticModelService) ResolveBackfillSourceIAMDependencies(ctx context.Context, modelID int64) ([]CreateSemanticModelSourceRequest, error) {
	if modelID <= 0 {
		return nil, semanticModelNotFoundError()
	}
	wsID, err := callerWorkspaceID(ctx)
	if err != nil {
		return nil, err
	}
	var response []CreateSemanticModelSourceRequest
	err = coreclient.Execute(ctx, coreclient.FromContext(ctx), func(callCtx context.Context, client *moi.Client) error {
		var callErr error
		response, callErr = s.resolveBackfillSourceIAMDependencies(callCtx, client, wsID, modelID)
		return callErr
	})
	return response, err
}

func (s *semanticModelService) resolveBackfillSourceIAMDependencies(ctx context.Context, client *moi.Client, wsID string, modelID int64) ([]CreateSemanticModelSourceRequest, error) {
	model, err := client.SemanticModels(wsID).Get(ctx, modelID)
	if err != nil {
		return nil, err
	}
	existing, err := s.listKnowledgeBaseSourceRows(ctx, modelID, true)
	if err != nil {
		return nil, err
	}
	jobs, err := s.listKnowledgeBaseSourceJobs(ctx, modelID)
	if err != nil {
		return nil, err
	}
	candidates, err := s.collectLegacySourceCandidateRecordsWithJobs(ctx, toSemanticModelInfo(model), existing, jobs, kbLegacyBackfillBatchSize)
	if err != nil {
		return nil, err
	}
	out := make([]CreateSemanticModelSourceRequest, 0, len(candidates))
	for _, candidate := range candidates {
		record := candidate.record
		if record.SourceType == kbSourceTypeCatalogTable {
			tableID := int64(0)
			if record.SourceTableID != nil {
				tableID = *record.SourceTableID
			} else if record.KBTableID != nil {
				tableID = *record.KBTableID
			}
			if tableID <= 0 {
				return nil, fmt.Errorf("legacy semantic model table source has no canonical table")
			}
			out = append(out, CreateSemanticModelSourceRequest{SourceType: kbSourceTypeCatalogTable, TableID: tableID})
			continue
		}
		fileID := ""
		if record.SourceFileID != nil {
			fileID = strings.TrimSpace(*record.SourceFileID)
		}
		if fileID == "" && record.KBFileID != nil {
			fileID = strings.TrimSpace(*record.KBFileID)
		}
		if fileID == "" {
			return nil, fmt.Errorf("legacy semantic model file source has no canonical file")
		}
		// Authorize the recorded volume so multi-volume files do not hit
		// ResolveCanonicalFileRoots ambiguity before backfill can proceed.
		volumeID := record.RawVolumeID
		if volumeID <= 0 {
			return nil, fmt.Errorf("legacy semantic model file source %s has no volume_id", fileID)
		}
		out = append(out, CreateSemanticModelSourceRequest{
			SourceType: kbSourceTypeCatalogFile,
			FileID:     fileID,
			VolumeID:   volumeID,
		})
	}
	return out, nil
}

const unsupportedSourceField = "unsupported"

var errKnowledgeBaseDataDomainLockMissing = errors.New("knowledge base data domain lock row is missing")
var errKnowledgeBaseDataDomainCASFailed = errors.New("knowledge base data domain CAS update failed")

const (
	kbEnsureStatusReady  = "ready"
	kbEnsureStatusFailed = "failed"
	// kbEnsureStatusProvisioning is a short-lived CAS claim while creating
	// catalog database/volumes so concurrent repair cannot clobber a winner.
	kbEnsureStatusProvisioning            = "provisioning"
	kbSourceStatusPending                 = "pending"
	kbSourceStatusSucceeded               = "succeeded"
	kbSourceStatusFailed                  = "failed"
	kbSourceStatusRemoved                 = "removed"
	kbSourceJobQueued                     = "queued"
	kbSourceJobPending                    = "pending"
	kbSourceJobRunning                    = "running"
	kbSourceJobSucceeded                  = "succeeded"
	kbSourceJobFailed                     = "failed"
	kbSourceTypeLocalFile                 = "local_file"
	kbSourceTypeCatalogFile               = "catalog_file"
	kbSourceTypeCatalogTable              = "catalog_table"
	kbLocalUploadKindStructured           = "structured"
	kbLocalUploadKindUnstructured         = "unstructured"
	kbRawKindDocument                     = "document"
	kbRawKindImage                        = "image"
	kbRawKindAudioVideo                   = "audio_video"
	kbRawKindStructured                   = "structured"
	kbJobTypeLoad                         = "load"
	kbJobTypeCopy                         = "copy"
	kbJobTypeTableClone                   = "table_clone"
	kbJobTypeRAGIngest                    = "rag_ingest"
	kbDefaultEmbeddingModel               = "bge-m3"
	kbDefaultImageEmbeddingModel          = "efficientnet-b3"
	kbDefaultImageEmbeddingDimension      = 1536
	kbDefaultImagePreprocessVersion       = "efficientnet-b3-v1-rgb-300-letterbox-imagenet"
	kbDefaultImageDistanceMetric          = "cosine"
	kbDefaultVLMOCRModel                  = "qwen3-vl-plus"
	kbStandardRAGTemplateKey              = "standard_rag"
	kbStandardRAGImageTemplateKey         = "standard_rag_with_image_index"
	kbAudioRAGTemplateKey                 = "audio_kb_ingest"
	kbVideoRAGTemplateKey                 = "video_kb_ingest"
	kbLegacyBackfillBatchSize             = 200
	kbSourceSelectionBatchSize            = 100
	kbSourceFileMetadataQueryBatchSize    = 1000
	kbSourceJobReconcileBackfillBatchSize = 20
	kbSourceJobReconcileBatchSize         = 8
	kbSourceJobFastBindBatchSize          = 64
	kbSourceJobRAGDispatchBatchSize       = 16
	kbSourceJobListBatchSize              = 32
	kbSourceListEnrichConcurrency         = 4
	// The lease only recovers claims left running by interruption or failed release;
	// handled transient file-bind errors are explicitly returned to queued.
	kbSourceJobClaimLease = 30 * time.Minute
	// Job operation_id prefixes are stable wire/DB protocol strings (keep value).
	// Symbol names use Bind; catalog_file never AddFiles into KB raw volume.
	kbJobOpCatalogFileBindPrefix  = "catalog_file_link:"
	kbJobOpLocalFileBindPrefix    = "local_file_link:"
	kbSelectionKindDatabaseTables = "database_tables"
	kbSelectionKindVolumeFiles    = "volume_files"
)

const knowledgeBaseSourceSelectColumns = `kbs.source_id AS source_id, kbs.model_id AS model_id, kbs.catalog_id AS catalog_id, kbs.database_id AS database_id, kbs.raw_volume_id AS raw_volume_id, kbs.processed_volume_id AS processed_volume_id, kbs.source_type AS source_type, kbs.source_file_id AS source_file_id, kbs.source_table_id AS source_table_id, kbs.kb_file_id AS kb_file_id, kbs.kb_table_id AS kb_table_id, kbs.display_name AS display_name, kbs.source_path AS source_path, kbs.db_name AS db_name, kbs.table_name AS table_name, kbs.status AS status, kbs.error AS error, kbs.enabled AS enabled, kbs.expires_at AS expires_at, kbs.tags AS tags, kbs.force_enabled_after_expiry AS force_enabled_after_expiry, kbs.segment_version_id AS segment_version_id, kbs.index_version AS index_version, NULL AS size_bytes, NULL AS row_count, kbs.created_by AS created_by, kbs.updated_by AS updated_by, UNIX_TIMESTAMP(kbs.updated_at) AS updated_at`

type knowledgeBaseSourceGovernanceState struct {
	Enabled          bool
	Expired          bool
	EffectiveEnabled bool
}

type knowledgeBaseSourceMissingError struct {
	msg string
}

func (e knowledgeBaseSourceMissingError) Error() string {
	return e.msg
}

type knowledgeBaseSourceInvalidError struct {
	msg string
}

func (e knowledgeBaseSourceInvalidError) Error() string {
	return e.msg
}

func isServiceNotFound(err error) bool {
	var svcErr *ServiceError
	return errors.As(err, &svcErr) && svcErr.Code == ErrCodeNotFound
}

func (s *semanticModelService) ListSources(ctx context.Context, params ListSemanticModelSourcesParams) (_ *ListSemanticModelSourcesResult, retErr error) {
	wsID, err := callerWorkspaceID(ctx)
	if err != nil {
		return nil, err
	}
	var response *ListSemanticModelSourcesResult
	err = coreclient.Execute(ctx, coreclient.FromContext(ctx), func(callCtx context.Context, client *moi.Client) error {
		var callErr error
		response, callErr = s.listSources(callCtx, client, wsID, params)
		return callErr
	})
	return response, err
}

func (s *semanticModelService) listSources(ctx context.Context, c *moi.Client, wsID string, params ListSemanticModelSourcesParams) (_ *ListSemanticModelSourcesResult, retErr error) {
	startedAt := time.Now()
	stage := "validation"
	var modelDuration, sourceQueryDuration, metadataDuration, jobQueryDuration, jobEnrichDuration, legacyDuration time.Duration
	var sourceCount, jobCount, legacyCandidateCount int
	var loggedPage, loggedPageSize int
	var metadataStats knowledgeBaseSourceMetadataEnrichStats
	var jobEnrichStats knowledgeBaseSourceJobEnrichStats
	defer func() {
		outcome := "success"
		errorStage := ""
		if retErr != nil {
			outcome = "error"
			errorStage = stage
		}
		logger.Info("knowledge base sources list completed",
			"model_id", int64(params.ModelID),
			"outcome", outcome,
			"error_stage", errorStage,
			"page", loggedPage,
			"page_size", loggedPageSize,
			"source_count", sourceCount,
			"job_count", jobCount,
			"legacy_candidate_count", legacyCandidateCount,
			"model_ms", modelDuration.Milliseconds(),
			"source_query_ms", sourceQueryDuration.Milliseconds(),
			"metadata_ms", metadataDuration.Milliseconds(),
			"job_query_ms", jobQueryDuration.Milliseconds(),
			"job_enrich_ms", jobEnrichDuration.Milliseconds(),
			"legacy_ms", legacyDuration.Milliseconds(),
			"file_metadata_calls", metadataStats.fileQueryCalls.Load(),
			"file_metadata_ms", time.Duration(metadataStats.fileQueryNanos.Load()).Milliseconds(),
			"table_metadata_calls", metadataStats.tableLookupCalls.Load(),
			"table_metadata_ms", time.Duration(metadataStats.tableLookupNanos.Load()).Milliseconds(),
			"table_status_calls", metadataStats.tableStatusCalls.Load(),
			"table_status_ms", time.Duration(metadataStats.tableStatusNanos.Load()).Milliseconds(),
			"import_task_calls", jobEnrichStats.importTaskCalls.Load(),
			"import_task_ms", time.Duration(jobEnrichStats.importTaskNanos.Load()).Milliseconds(),
			"file_execution_calls", jobEnrichStats.fileExecutionCalls.Load(),
			"file_execution_ms", time.Duration(jobEnrichStats.fileExecutionNanos.Load()).Milliseconds(),
			"workflow_execution_calls", jobEnrichStats.workflowExecutionCalls.Load(),
			"workflow_execution_ms", time.Duration(jobEnrichStats.workflowExecutionNanos.Load()).Milliseconds(),
			"total_ms", time.Since(startedAt).Milliseconds())
	}()
	modelID := int64(params.ModelID)
	if modelID == 0 {
		return nil, semanticModelNotFoundError()
	}
	pageScopedJobs := params.Page > 0 || params.PageSize > 0
	page, pageSize := normalizeListSourcesPagination(params.Page, params.PageSize)
	loggedPage, loggedPageSize = page, pageSize

	stage = "model_query"
	stageStartedAt := time.Now()
	model, err := c.SemanticModels(wsID).Get(ctx, modelID)
	modelDuration = time.Since(stageStartedAt)
	if err != nil {
		return nil, err
	}
	stage = "source_query"
	stageStartedAt = time.Now()
	sourceRecords, err := s.listKnowledgeBaseSourceRows(ctx, modelID, true)
	sourceQueryDuration = time.Since(stageStartedAt)
	if err != nil {
		return nil, err
	}
	records := activeKnowledgeBaseSourceRecords(sourceRecords)
	sourceCount = len(records)
	start, end := listSourcesPageBounds(len(records), page, pageSize)
	items := make([]SemanticModelSource, 0, end-start)
	pagedRecords := []KnowledgeBaseSourceRecord{}
	if start < len(records) {
		recordEnd := end
		if recordEnd > len(records) {
			recordEnd = len(records)
		}
		pagedRecords = append([]KnowledgeBaseSourceRecord(nil), records[start:recordEnd]...)
		stage = "metadata_enrich"
		stageStartedAt = time.Now()
		pagedRecords, err = s.enrichKnowledgeBaseSourceRecordsMetadataWithStats(ctx, c, wsID, pagedRecords, &metadataStats)
		metadataDuration = time.Since(stageStartedAt)
		if err != nil {
			return nil, err
		}
		for _, record := range pagedRecords {
			items = append(items, sourceRecordToSemanticModelSource(record))
		}
	}
	pagedSourceIDs := make([]string, 0, len(pagedRecords))
	for _, record := range pagedRecords {
		pagedSourceIDs = append(pagedSourceIDs, record.SourceID)
	}
	var jobs []KnowledgeBaseSourceJobRun
	stage = "job_query"
	stageStartedAt = time.Now()
	if pageScopedJobs {
		jobs, err = s.listKnowledgeBaseSourceJobRunsForSourceIDs(ctx, modelID, pagedSourceIDs)
	} else {
		jobs, err = s.listKnowledgeBaseSourceJobRuns(ctx, modelID)
	}
	jobQueryDuration = time.Since(stageStartedAt)
	if err != nil {
		return nil, err
	}
	jobCount = len(jobs)
	stage = "job_enrich"
	stageStartedAt = time.Now()
	jobs, err = s.enrichKnowledgeBaseSourceJobRunsFromLinkedJobsWithStats(ctx, c, wsID, jobs, &jobEnrichStats)
	jobEnrichDuration = time.Since(stageStartedAt)
	if err != nil {
		return nil, err
	}
	if len(items) > 0 {
		items = applyKnowledgeBaseSourceJobStatus(items, jobs)
	}
	stage = "legacy"
	stageStartedAt = time.Now()
	legacyJobs, err := s.listKnowledgeBaseSourceJobs(ctx, modelID)
	if err != nil {
		return nil, err
	}
	candidates, err := s.legacySourceCandidateRowsWithJobs(ctx, toSemanticModelInfo(model), sourceRecords, legacyJobs, kbLegacyBackfillBatchSize)
	legacyDuration = time.Since(stageStartedAt)
	if err != nil {
		return nil, err
	}
	legacyCandidateCount = len(candidates)
	total := len(records) + len(candidates)
	start, end = listSourcesPageBounds(total, page, pageSize)
	if end > len(records) {
		candidateStart := start - len(records)
		if candidateStart < 0 {
			candidateStart = 0
		}
		candidateEnd := end - len(records)
		if candidateEnd > len(candidates) {
			candidateEnd = len(candidates)
		}
		if candidateStart < candidateEnd {
			items = append(items, candidates[candidateStart:candidateEnd]...)
		}
	}
	required := len(candidates) > 0
	if !required {
		if pageScopedJobs && len(legacyJobs) > 0 {
			required, err = s.legacySourceJobRunExists(ctx, modelID)
			legacyDuration = time.Since(stageStartedAt)
			if err != nil {
				return nil, err
			}
		} else {
			required = legacySourceJobRunsBackfillRequired(sourceRecords, jobs, legacyJobs)
		}
	}
	if !pageScopedJobs {
		pageSize = total
	}
	loggedPageSize = pageSize
	return &ListSemanticModelSourcesResult{Items: items, Total: total, Page: page, PageSize: pageSize, LegacyBackfillRequired: required}, nil
}

func (s *semanticModelService) CheckSourceExistence(ctx context.Context, params CheckSemanticModelSourceExistenceParams) (*CheckSemanticModelSourceExistenceResult, error) {
	modelID := int64(params.ModelID)
	if modelID == 0 {
		return nil, semanticModelNotFoundError()
	}

	if err := validateUniqueExactStrings(params.FileIDs); err != nil {
		return nil, semanticModelSourceFieldInvalidError(0, "file_ids")
	}
	if err := validateUniquePositiveInt64s(params.TableIDs); err != nil {
		return nil, semanticModelSourceFieldInvalidError(0, "table_ids")
	}
	fileIDs := params.FileIDs
	tableIDs := params.TableIDs
	result := &CheckSemanticModelSourceExistenceResult{
		FileIDs:  []string{},
		TableIDs: []int64{},
	}
	if len(fileIDs) == 0 && len(tableIDs) == 0 {
		return result, nil
	}

	db := ctxutil.TenantDBFrom(ctx)
	if db == nil {
		return nil, fmt.Errorf("tenant db is required")
	}
	wsID, err := callerWorkspaceID(ctx)
	if err != nil {
		return nil, err
	}
	err = coreclient.Execute(ctx, coreclient.FromContext(ctx), func(callCtx context.Context, client *moi.Client) error {
		if _, callErr := client.SemanticModels(wsID).Get(callCtx, modelID); callErr != nil {
			return callErr
		}

		if len(fileIDs) > 0 {
			existing, queryErr := s.existingKnowledgeBaseCatalogFileIDs(callCtx, db, modelID, fileIDs)
			if queryErr != nil {
				return queryErr
			}
			result.FileIDs = orderedExistingStrings(fileIDs, existing)
		}
		if len(tableIDs) > 0 {
			existing, queryErr := s.existingKnowledgeBaseCatalogTableIDs(callCtx, db, modelID, tableIDs)
			if queryErr != nil {
				return queryErr
			}
			result.TableIDs = orderedExistingInt64s(tableIDs, existing)
		}
		return nil
	})
	return result, err
}

func (s *semanticModelService) existingKnowledgeBaseCatalogFileIDs(ctx context.Context, db *gorm.DB, modelID int64, fileIDs []string) (map[string]struct{}, error) {
	volumes, err := s.existingKnowledgeBaseCatalogFileVolumes(ctx, db, modelID, fileIDs)
	if err != nil {
		return nil, err
	}
	out := make(map[string]struct{}, len(volumes))
	for fileID := range volumes {
		out[fileID] = struct{}{}
	}
	return out, nil
}

// existingKnowledgeBaseCatalogFileVolumes returns active catalog_file sources keyed by
// file_id with the recorded raw_volume_id (user source volume for catalog_file).
func (s *semanticModelService) existingKnowledgeBaseCatalogFileVolumes(ctx context.Context, db *gorm.DB, modelID int64, fileIDs []string) (map[string]int64, error) {
	requested := stringSet(fileIDs)
	rows, err := db.WithContext(ctx).Raw(`SELECT source_file_id, kb_file_id, raw_volume_id
		FROM knowledge_base_sources
		WHERE model_id = ? AND source_type = ? AND status <> ? AND (source_file_id IN ? OR kb_file_id IN ?)`,
		modelID, kbSourceTypeCatalogFile, kbSourceStatusRemoved, fileIDs, fileIDs).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[string]int64, len(fileIDs))
	for rows.Next() {
		var sourceFileID, kbFileID sql.NullString
		var rawVolumeID int64
		if err := rows.Scan(&sourceFileID, &kbFileID, &rawVolumeID); err != nil {
			return nil, err
		}
		if sourceFileID.Valid {
			if _, ok := requested[sourceFileID.String]; ok {
				out[sourceFileID.String] = rawVolumeID
			}
		}
		if kbFileID.Valid {
			if _, ok := requested[kbFileID.String]; ok {
				out[kbFileID.String] = rawVolumeID
			}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *semanticModelService) existingKnowledgeBaseCatalogTableIDs(ctx context.Context, db *gorm.DB, modelID int64, tableIDs []int64) (map[int64]struct{}, error) {
	requested := int64Set(tableIDs)
	rows, err := db.WithContext(ctx).Raw(`SELECT source_table_id, kb_table_id
		FROM knowledge_base_sources
		WHERE model_id = ? AND source_type = ? AND status <> ? AND (source_table_id IN ? OR kb_table_id IN ?)`,
		modelID, kbSourceTypeCatalogTable, kbSourceStatusRemoved, tableIDs, tableIDs).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[int64]struct{}, len(tableIDs))
	for rows.Next() {
		var sourceTableID, kbTableID sql.NullInt64
		if err := rows.Scan(&sourceTableID, &kbTableID); err != nil {
			return nil, err
		}
		if sourceTableID.Valid {
			if _, ok := requested[sourceTableID.Int64]; ok {
				out[sourceTableID.Int64] = struct{}{}
			}
		}
		if kbTableID.Valid {
			if _, ok := requested[kbTableID.Int64]; ok {
				out[kbTableID.Int64] = struct{}{}
			}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func orderedExistingStrings(ids []string, existing map[string]struct{}) []string {
	out := make([]string, 0, len(existing))
	for _, id := range ids {
		if _, ok := existing[id]; ok {
			out = append(out, id)
		}
	}
	return out
}

func orderedExistingInt64s(ids []int64, existing map[int64]struct{}) []int64 {
	out := make([]int64, 0, len(existing))
	for _, id := range ids {
		if _, ok := existing[id]; ok {
			out = append(out, id)
		}
	}
	return out
}

func normalizeListSourcesPagination(page, pageSize int) (int, int) {
	if page <= 0 && pageSize <= 0 {
		return 1, int(^uint(0) >> 1)
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

func listSourcesPageBounds(total, page, pageSize int) (int, int) {
	if total <= 0 {
		return 0, 0
	}
	if page > (total-1)/pageSize+1 {
		return total, total
	}
	start := (page - 1) * pageSize
	if start >= total {
		return total, total
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return start, end
}

type knowledgeBaseSourceMetadataEnrichStats struct {
	fileQueryCalls   atomic.Int64
	fileQueryNanos   atomic.Int64
	tableLookupCalls atomic.Int64
	tableLookupNanos atomic.Int64
	tableStatusCalls atomic.Int64
	tableStatusNanos atomic.Int64
}

type knowledgeBaseTableSourceMetadata struct {
	dbName    string
	tableName string
	updatedAt int64
	path      []string
	stats     tableStats
}

func (s *semanticModelService) enrichKnowledgeBaseSourceRecordsMetadata(ctx context.Context, c *moi.Client, wsID string, records []KnowledgeBaseSourceRecord) ([]KnowledgeBaseSourceRecord, error) {
	return s.enrichKnowledgeBaseSourceRecordsMetadataWithStats(ctx, c, wsID, records, nil)
}

func (s *semanticModelService) enrichKnowledgeBaseSourceRecordsMetadataWithStats(ctx context.Context, c *moi.Client, wsID string, records []KnowledgeBaseSourceRecord, stats *knowledgeBaseSourceMetadataEnrichStats) ([]KnowledgeBaseSourceRecord, error) {
	if len(records) == 0 {
		return records, nil
	}
	metadataErrors := make([]error, len(records))
	fileRecordLocations := make(map[int]catalogFileSourceLocation)
	fileLocations := make([]catalogFileSourceLocation, 0, len(records))
	tableMetadata := make([]*knowledgeBaseTableSourceMetadata, len(records))
	for i := range records {
		switch records[i].SourceType {
		case kbSourceTypeLocalFile, kbSourceTypeCatalogFile:
			fileID := knowledgeBaseSourceRecordFileID(records[i])
			if fileID == "" {
				metadataErrors[i] = knowledgeBaseSourceInvalidError{msg: fmt.Sprintf("knowledge base file source %s has no file id", records[i].SourceID)}
				continue
			}
			location := catalogFileSourceLocation{fileID: fileID, volumeID: records[i].RawVolumeID}
			fileRecordLocations[i] = location
			fileLocations = append(fileLocations, location)
		}
	}
	if len(fileLocations) > 0 {
		startedAt := time.Now()
		if stats != nil {
			stats.fileQueryCalls.Add(1)
		}
		metadataByLocation, err := currentCatalogFileMetadataBatch(ctx, fileLocations)
		if stats != nil {
			stats.fileQueryNanos.Add(int64(time.Since(startedAt)))
		}
		if err != nil {
			return nil, err
		}
		for i, location := range fileRecordLocations {
			metadata, ok := metadataByLocation[location]
			if !ok {
				metadataErrors[i] = knowledgeBaseSourceMissingError{msg: fmt.Sprintf("catalog file %s not found", location.fileID)}
				continue
			}
			if metadata.err != nil {
				var missing knowledgeBaseSourceMissingError
				if errors.As(metadata.err, &missing) {
					metadataErrors[i] = metadata.err
				} else {
					metadataErrors[i] = fmt.Errorf("get catalog file %s for knowledge base source %s: %w", location.fileID, records[i].SourceID, metadata.err)
				}
				continue
			}
			metadataErrors[i] = applyKnowledgeBaseFileSourceMetadata(&records[i], metadata.metadata)
		}
	}

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(kbSourceListEnrichConcurrency)
	for i := range records {
		if records[i].SourceType != kbSourceTypeCatalogTable {
			continue
		}
		tableID := knowledgeBaseSourceRecordTableID(records[i])
		if tableID <= 0 {
			continue
		}
		i := i
		g.Go(func() error {
			startedAt := time.Now()
			if stats != nil {
				stats.tableLookupCalls.Add(1)
				defer func() { stats.tableLookupNanos.Add(int64(time.Since(startedAt))) }()
			}
			detail, err := c.Databases().GetTable(gctx, wsID, tableID)
			if err != nil {
				if moi.IsCode(err, common.ErrorCode_NOT_FOUND) {
					metadataErrors[i] = knowledgeBaseSourceMissingError{msg: fmt.Sprintf("catalog table %d not found", tableID)}
					return nil
				}
				return fmt.Errorf("get catalog table %d for knowledge base source %s: %w", tableID, records[i].SourceID, err)
			}
			if detail == nil || detail.Table == nil || detail.Database == nil {
				return fmt.Errorf("catalog table %d metadata is incomplete", tableID)
			}
			if detail.Database.Name == "" || detail.Table.Name == "" {
				return fmt.Errorf("catalog table %d metadata has empty database or table name", tableID)
			}
			tableMetadata[i] = &knowledgeBaseTableSourceMetadata{
				dbName: detail.Database.Name, tableName: detail.Table.Name, updatedAt: detail.Table.UpdatedAt,
				path: compactNonEmptyStrings(backendcatalog.CatalogDisplayName(gctx, detail.Catalog), backendcatalog.DatabaseDisplayName(gctx, detail.Database)),
			}
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	tableIndexesByDB := make(map[string][]int)
	for i, metadata := range tableMetadata {
		if metadata != nil {
			tableIndexesByDB[metadata.dbName] = append(tableIndexesByDB[metadata.dbName], i)
		}
	}
	g, gctx = errgroup.WithContext(ctx)
	g.SetLimit(kbSourceListEnrichConcurrency)
	for dbName, indexes := range tableIndexesByDB {
		dbName, indexes := dbName, indexes
		g.Go(func() error {
			startedAt := time.Now()
			if stats != nil {
				stats.tableStatusCalls.Add(1)
				defer func() { stats.tableStatusNanos.Add(int64(time.Since(startedAt))) }()
			}
			db, err := s.openWorkspaceDBForStats(gctx, c, wsID, dbName)
			if err != nil {
				return fmt.Errorf("open workspace db %s: %w", dbName, err)
			}
			defer db.Close()
			tableNames := make([]string, 0, len(indexes))
			for _, i := range indexes {
				tableNames = append(tableNames, tableMetadata[i].tableName)
			}
			tableStats, err := queryTableStats(gctx, db, dbName, tableNames)
			if err != nil {
				return fmt.Errorf("get table stats for database %s: %w", dbName, err)
			}
			for _, i := range indexes {
				stat, ok := tableStats[tableMetadata[i].tableName]
				if !ok {
					return fmt.Errorf("table stats returned no row for table %s.%s", dbName, tableMetadata[i].tableName)
				}
				tableMetadata[i].stats = stat
			}
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	for i, metadata := range tableMetadata {
		if metadata != nil {
			metadataErrors[i] = applyKnowledgeBaseTableSourceMetadata(&records[i], *metadata)
		}
	}

	filtered := make([]KnowledgeBaseSourceRecord, 0, len(records))
	for i := range records {
		err := metadataErrors[i]
		if err != nil {
			var missing knowledgeBaseSourceMissingError
			if errors.As(err, &missing) {
				if records[i].SourceType == kbSourceTypeLocalFile && records[i].Status == kbSourceStatusPending {
					filtered = append(filtered, records[i])
					continue
				}
			}
			if knowledgeBaseSourceMetadataFailure(err) {
				markKnowledgeBaseSourceRecordMetadataFailed(&records[i], err)
				filtered = append(filtered, records[i])
				continue
			}
			return nil, err
		}
		filtered = append(filtered, records[i])
	}
	return filtered, nil
}

// knowledgeBaseSourceRecordFileID returns the authoritative catalog metadata file
// id for list/detail enrichment. Prefer source_file_id (physical file on the
// recorded volume). Fall back to kb_file_id only when source is unset (legacy
// rows or local files still mid-upload).
func knowledgeBaseSourceRecordFileID(record KnowledgeBaseSourceRecord) string {
	if record.SourceFileID != nil && strings.TrimSpace(*record.SourceFileID) != "" {
		return strings.TrimSpace(*record.SourceFileID)
	}
	if record.KBFileID != nil && *record.KBFileID != "" {
		return *record.KBFileID
	}
	return ""
}

func knowledgeBaseSourceRecordTableID(record KnowledgeBaseSourceRecord) int64 {
	if record.KBTableID != nil && *record.KBTableID > 0 {
		return *record.KBTableID
	}
	if record.SourceTableID != nil {
		return *record.SourceTableID
	}
	return 0
}

func applyKnowledgeBaseFileSourceMetadata(record *KnowledgeBaseSourceRecord, metadata catalogFileSourceMetadata) error {
	record.DisplayName = stringPtr(metadata.fileName)
	record.SizeBytes = int64Ptr(metadata.sizeBytes)
	record.UpdatedAt = int64Ptr(metadata.updatedAt)
	sourcePath, err := json.Marshal(metadata.path)
	if err != nil {
		return fmt.Errorf("marshal file source path: %w", err)
	}
	record.SourcePath = stringPtr(string(sourcePath))
	return nil
}

func applyKnowledgeBaseTableSourceMetadata(record *KnowledgeBaseSourceRecord, metadata knowledgeBaseTableSourceMetadata) error {
	record.DisplayName = stringPtr(metadata.tableName)
	record.DBName = stringPtr(metadata.dbName)
	record.TableName = stringPtr(metadata.tableName)
	record.RowCount = int64Ptr(metadata.stats.rowCount)
	record.SizeBytes = int64Ptr(metadata.stats.sizeBytes)
	record.UpdatedAt = int64Ptr(metadata.updatedAt)
	sourcePath, err := json.Marshal(metadata.path)
	if err != nil {
		return fmt.Errorf("marshal table source path: %w", err)
	}
	record.SourcePath = stringPtr(string(sourcePath))
	return nil
}

func knowledgeBaseSourceMetadataFailure(err error) bool {
	var missing knowledgeBaseSourceMissingError
	if errors.As(err, &missing) {
		return true
	}
	var invalid knowledgeBaseSourceInvalidError
	return errors.As(err, &invalid)
}

func markKnowledgeBaseSourceRecordMetadataFailed(record *KnowledgeBaseSourceRecord, err error) {
	if record == nil || err == nil {
		return
	}
	record.Status = kbSourceStatusFailed
	if record.Error == nil || *record.Error == "" {
		record.Error = stringPtr(err.Error())
	}
}

type catalogFileSourceMetadata struct {
	catalogID  int64
	databaseID int64
	volumeID   int64
	fileName   string
	sizeBytes  int64
	updatedAt  int64
	path       []string
}

type catalogFileSourceMetadataResult struct {
	metadata catalogFileSourceMetadata
	err      error
}

type catalogFileSourceLocation struct {
	fileID   string
	volumeID int64
}

func currentCatalogFileMetadataBatch(ctx context.Context, locations []catalogFileSourceLocation) (map[catalogFileSourceLocation]catalogFileSourceMetadataResult, error) {
	db := ctxutil.TenantDBFrom(ctx)
	if db == nil {
		return nil, fmt.Errorf("tenant db is required")
	}
	uniqueLocations := make([]catalogFileSourceLocation, 0, len(locations))
	locationsByFileID := make(map[string][]catalogFileSourceLocation, len(locations))
	uniqueFileIDs := make([]string, 0, len(locations))
	seenLocations := make(map[catalogFileSourceLocation]struct{}, len(locations))
	for _, location := range locations {
		if location.fileID == "" {
			continue
		}
		if _, ok := seenLocations[location]; ok {
			continue
		}
		seenLocations[location] = struct{}{}
		uniqueLocations = append(uniqueLocations, location)
		if _, ok := locationsByFileID[location.fileID]; !ok {
			uniqueFileIDs = append(uniqueFileIDs, location.fileID)
		}
		locationsByFileID[location.fileID] = append(locationsByFileID[location.fileID], location)
	}
	if len(uniqueLocations) == 0 {
		return map[catalogFileSourceLocation]catalogFileSourceMetadataResult{}, nil
	}
	result := make(map[catalogFileSourceLocation]catalogFileSourceMetadataResult, len(uniqueLocations))
	for start := 0; start < len(uniqueFileIDs); start += kbSourceFileMetadataQueryBatchSize {
		end := start + kbSourceFileMetadataQueryBatchSize
		if end > len(uniqueFileIDs) {
			end = len(uniqueFileIDs)
		}
		fileIDs := uniqueFileIDs[start:end]
		batchLocations := make([]catalogFileSourceLocation, 0, len(fileIDs))
		for _, fileID := range fileIDs {
			batchLocations = append(batchLocations, locationsByFileID[fileID]...)
		}
		batch, err := queryCurrentCatalogFileMetadataBatch(ctx, db, batchLocations)
		if err != nil {
			return nil, err
		}
		for location, metadata := range batch {
			result[location] = metadata
		}
	}
	return result, nil
}

func queryCurrentCatalogFileMetadataBatch(ctx context.Context, db *gorm.DB, locations []catalogFileSourceLocation) (map[catalogFileSourceLocation]catalogFileSourceMetadataResult, error) {
	fileIDs := make([]string, 0, len(locations))
	requested := make(map[catalogFileSourceLocation]struct{}, len(locations))
	seenFileIDs := make(map[string]struct{}, len(locations))
	for _, location := range locations {
		if _, ok := requested[location]; ok {
			continue
		}
		requested[location] = struct{}{}
		if _, ok := seenFileIDs[location.fileID]; ok {
			continue
		}
		seenFileIDs[location.fileID] = struct{}{}
		fileIDs = append(fileIDs, location.fileID)
	}
	if len(fileIDs) == 0 {
		return map[catalogFileSourceLocation]catalogFileSourceMetadataResult{}, nil
	}
	query := `SELECT f.file_id, COALESCE(CASE WHEN v.catalog_id > 0 THEN v.catalog_id ELSE cd.catalog_id END, 0) AS catalog_id, COALESCE(v.database_id, 0) AS database_id, COALESCE(vf.volume_id, 0) AS volume_id, f.size, UNIX_TIMESTAMP(COALESCE(vf.updated_at, f.updated_at)) AS updated_at, COALESCE(NULLIF(catalog_name_display.default_text, ''), c.catalog_name, '') AS catalog_name, COALESCE(NULLIF(database_name_display.default_text, ''), cd.database_name, '') AS database_name, COALESCE(NULLIF(volume_name_display.default_text, ''), v.volume_name, '') AS volume_name, COALESCE(vf.file_path, '') AS file_path, COALESCE(vf.file_name, f.original_name) AS file_name
		FROM ` + "`file`" + ` f
		LEFT JOIN volume_files vf ON vf.file_id = f.file_id
		LEFT JOIN volume v ON v.volume_id = vf.volume_id AND v.deleted = FALSE
		LEFT JOIN catalog_database cd ON cd.database_id = v.database_id
		LEFT JOIN catalog c ON c.catalog_id = CASE WHEN v.catalog_id > 0 THEN v.catalog_id ELSE cd.catalog_id END
		LEFT JOIN system_resource_display_mapping catalog_name_display ON catalog_name_display.resource_type = 'catalog' AND catalog_name_display.resource_id = CAST(c.catalog_id AS CHAR) AND catalog_name_display.field = 'name' AND catalog_name_display.display_owner = 'moi_backend'
		LEFT JOIN system_resource_display_mapping database_name_display ON database_name_display.resource_type = 'database' AND database_name_display.resource_id = CAST(cd.database_id AS CHAR) AND database_name_display.field = 'name' AND database_name_display.display_owner = 'moi_backend'
		LEFT JOIN system_resource_display_mapping volume_name_display ON volume_name_display.resource_type = 'volume' AND volume_name_display.resource_id = CAST(v.volume_id AS CHAR) AND volume_name_display.field = 'name' AND volume_name_display.display_owner = 'moi_backend'
		WHERE f.file_id IN (` + queryPlaceholders(len(fileIDs)) + `)
		ORDER BY f.file_id ASC, vf.updated_at DESC, vf.id DESC`
	rows, err := db.WithContext(ctx).Raw(query, stringArgs(fileIDs)...).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[catalogFileSourceLocation]catalogFileSourceMetadataResult, len(locations))
	scanned := make(map[catalogFileSourceLocation]struct{}, len(locations))
	// Exact (file_id, volume_id) only. Callers must persist the authoritative
	// volume for catalog_file sources (original volume, not KB domain raw).
	for rows.Next() {
		var fileID, catalogName, databaseName, volumeName, filePath, fileName string
		var catalogID, databaseID, volumeID, sizeBytes, updatedAt int64
		if err := rows.Scan(&fileID, &catalogID, &databaseID, &volumeID, &sizeBytes, &updatedAt, &catalogName, &databaseName, &volumeName, &filePath, &fileName); err != nil {
			return nil, err
		}
		metadata := catalogFileSourceMetadata{catalogID: catalogID, databaseID: databaseID, volumeID: volumeID, fileName: fileName, sizeBytes: sizeBytes, updatedAt: updatedAt, path: compactNonEmptyStrings(catalogName, databaseName, volumeName)}
		metadataResult := catalogFileSourceMetadataResult{}
		switch {
		case fileName == "":
			metadataResult.err = fmt.Errorf("catalog file %s has empty display name", fileID)
		case volumeName == "":
			metadataResult.err = knowledgeBaseSourceMissingError{msg: fmt.Sprintf("catalog file %s is not linked to an active volume", fileID)}
		case catalogID <= 0 || databaseID <= 0 || volumeID <= 0:
			metadataResult.err = knowledgeBaseSourceMissingError{msg: fmt.Sprintf("catalog file %s has incomplete catalog location", fileID)}
		default:
			metadataResult.metadata = metadata
		}
		location := catalogFileSourceLocation{fileID: fileID, volumeID: volumeID}
		if _, ok := requested[location]; !ok {
			continue
		}
		if _, ok := scanned[location]; ok {
			continue
		}
		scanned[location] = struct{}{}
		result[location] = metadataResult
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

// currentCatalogFileMetadataAtVolume resolves catalog metadata for an exact
// (file_id, volume_id) location. Used by write intents that already have a
// trusted selection/direct volume.
func currentCatalogFileMetadataAtVolume(ctx context.Context, fileID string, volumeID int64) (catalogFileSourceMetadata, error) {
	if strings.TrimSpace(fileID) == "" {
		return catalogFileSourceMetadata{}, knowledgeBaseSourceMissingError{msg: "catalog file id is required"}
	}
	if volumeID <= 0 {
		return catalogFileSourceMetadata{}, fmt.Errorf("catalog file %s requires volume_id", fileID)
	}
	batch, err := currentCatalogFileMetadataBatch(ctx, []catalogFileSourceLocation{{fileID: fileID, volumeID: volumeID}})
	if err != nil {
		return catalogFileSourceMetadata{}, err
	}
	result, ok := batch[catalogFileSourceLocation{fileID: fileID, volumeID: volumeID}]
	if !ok {
		return catalogFileSourceMetadata{}, knowledgeBaseSourceMissingError{msg: fmt.Sprintf("catalog file %s not found at volume %d", fileID, volumeID)}
	}
	if result.err != nil {
		return catalogFileSourceMetadata{}, result.err
	}
	return result.metadata, nil
}

// currentCatalogFileMetadata is retained for legacy/backfill paths that only
// have a file_id. New write paths must use currentCatalogFileMetadataAtVolume.
func currentCatalogFileMetadata(ctx context.Context, fileID string) (catalogFileSourceMetadata, error) {
	db := ctxutil.TenantDBFrom(ctx)
	if db == nil {
		return catalogFileSourceMetadata{}, fmt.Errorf("tenant db is required")
	}
	rows, err := db.WithContext(ctx).Raw(`SELECT COALESCE(CASE WHEN v.catalog_id > 0 THEN v.catalog_id ELSE cd.catalog_id END, 0) AS catalog_id, COALESCE(v.database_id, 0) AS database_id, COALESCE(vf.volume_id, 0) AS volume_id, f.size, UNIX_TIMESTAMP(COALESCE(vf.updated_at, f.updated_at)) AS updated_at, COALESCE(NULLIF(catalog_name_display.default_text, ''), c.catalog_name, '') AS catalog_name, COALESCE(NULLIF(database_name_display.default_text, ''), cd.database_name, '') AS database_name, COALESCE(NULLIF(volume_name_display.default_text, ''), v.volume_name, '') AS volume_name, COALESCE(vf.file_path, '') AS file_path, COALESCE(vf.file_name, f.original_name) AS file_name
		FROM `+"`file`"+` f
		LEFT JOIN volume_files vf ON vf.file_id = f.file_id
		LEFT JOIN volume v ON v.volume_id = vf.volume_id AND v.deleted = FALSE
		LEFT JOIN catalog_database cd ON cd.database_id = v.database_id
		LEFT JOIN catalog c ON c.catalog_id = CASE WHEN v.catalog_id > 0 THEN v.catalog_id ELSE cd.catalog_id END
		LEFT JOIN system_resource_display_mapping catalog_name_display ON catalog_name_display.resource_type = 'catalog' AND catalog_name_display.resource_id = CAST(c.catalog_id AS CHAR) AND catalog_name_display.field = 'name' AND catalog_name_display.display_owner = 'moi_backend'
		LEFT JOIN system_resource_display_mapping database_name_display ON database_name_display.resource_type = 'database' AND database_name_display.resource_id = CAST(cd.database_id AS CHAR) AND database_name_display.field = 'name' AND database_name_display.display_owner = 'moi_backend'
		LEFT JOIN system_resource_display_mapping volume_name_display ON volume_name_display.resource_type = 'volume' AND volume_name_display.resource_id = CAST(v.volume_id AS CHAR) AND volume_name_display.field = 'name' AND volume_name_display.display_owner = 'moi_backend'
		WHERE f.file_id = ?
		ORDER BY vf.updated_at DESC, vf.id DESC
		LIMIT 1`, fileID).Rows()
	if err != nil {
		return catalogFileSourceMetadata{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		return catalogFileSourceMetadata{}, knowledgeBaseSourceMissingError{msg: fmt.Sprintf("catalog file %s not found", fileID)}
	}
	var catalogID, databaseID, volumeID, sizeBytes, updatedAt int64
	var catalogName, databaseName, volumeName, filePath, fileName string
	if err := rows.Scan(&catalogID, &databaseID, &volumeID, &sizeBytes, &updatedAt, &catalogName, &databaseName, &volumeName, &filePath, &fileName); err != nil {
		return catalogFileSourceMetadata{}, err
	}
	if err := rows.Err(); err != nil {
		return catalogFileSourceMetadata{}, err
	}
	if fileName == "" {
		return catalogFileSourceMetadata{}, fmt.Errorf("catalog file %s has empty display name", fileID)
	}
	if volumeName == "" {
		return catalogFileSourceMetadata{}, knowledgeBaseSourceMissingError{msg: fmt.Sprintf("catalog file %s is not linked to an active volume", fileID)}
	}
	if catalogID <= 0 || databaseID <= 0 || volumeID <= 0 {
		return catalogFileSourceMetadata{}, knowledgeBaseSourceMissingError{msg: fmt.Sprintf("catalog file %s has incomplete catalog location", fileID)}
	}
	return catalogFileSourceMetadata{
		catalogID:  catalogID,
		databaseID: databaseID,
		volumeID:   volumeID,
		fileName:   fileName,
		sizeBytes:  sizeBytes,
		updatedAt:  updatedAt,
		path:       compactNonEmptyStrings(catalogName, databaseName, volumeName),
	}, nil
}

func (s *semanticModelService) openWorkspaceDBForStats(ctx context.Context, c *moi.Client, wsID, dbName string) (*sql.DB, error) {
	if s.openWorkspaceDB != nil {
		return s.openWorkspaceDB(ctx, wsID, dbName)
	}
	return openWorkspaceDBWithClient(ctx, c, wsID, dbName)
}

func openWorkspaceDBWithClient(ctx context.Context, c *moi.Client, wsID, dbName string) (*sql.DB, error) {
	if c == nil {
		return nil, fmt.Errorf("moi client is required")
	}
	opts := []moi.DBOpenOption{moi.WithDBMultiStatements(true)}
	if dbName != "" {
		opts = append(opts, moi.WithDBName(dbName))
	}
	conn, err := c.Workspaces().GetDBConnection(ctx, wsID)
	if err != nil {
		return nil, err
	}
	db, err := moi.OpenCloudNonUserDBFromConnection(conn, opts...)
	if err != nil {
		return nil, err
	}
	return db, nil
}

func (s *semanticModelService) getKnowledgeBaseSourceAfterEnsure(ctx context.Context, model *SemanticModelInfo, modelID int64, sourceID string) (KnowledgeBaseSourceRecord, error) {
	records, err := s.listKnowledgeBaseSourceRows(ctx, modelID, true)
	if err != nil {
		return KnowledgeBaseSourceRecord{}, err
	}
	inserted, err := s.ensureKnowledgeBaseSourceRowsForSemanticModel(ctx, model, records)
	if err != nil {
		return KnowledgeBaseSourceRecord{}, err
	}
	if !inserted {
		return KnowledgeBaseSourceRecord{}, knowledgeBaseSourceNotFoundError()
	}
	return s.getKnowledgeBaseSource(ctx, modelID, sourceID)
}

func (s *semanticModelService) ensureKnowledgeBaseSourceRowsForSemanticModel(ctx context.Context, model *SemanticModelInfo, existing []KnowledgeBaseSourceRecord) (bool, error) {
	if model == nil || model.ID <= 0 {
		return false, nil
	}
	db := ctxutil.TenantDBFrom(ctx)
	if db == nil {
		return false, nil
	}
	records, err := s.explicitSemanticModelSourceRecords(ctx, model, existing)
	if err != nil {
		return false, err
	}
	existingFiles := make(map[string]struct{}, len(existing))
	for _, record := range existing {
		if record.KBFileID != nil && *record.KBFileID != "" {
			existingFiles[*record.KBFileID] = struct{}{}
		}
		if record.SourceFileID != nil && *record.SourceFileID != "" {
			existingFiles[*record.SourceFileID] = struct{}{}
		}
	}
	for _, record := range records {
		if record.KBFileID != nil && *record.KBFileID != "" {
			existingFiles[*record.KBFileID] = struct{}{}
		}
		if record.SourceFileID != nil && *record.SourceFileID != "" {
			existingFiles[*record.SourceFileID] = struct{}{}
		}
	}
	rawVolumeFiles, err := knowledgeBaseRawVolumeFileMetadata(ctx, model.ID)
	if err != nil {
		return false, err
	}
	for _, file := range rawVolumeFiles {
		if _, ok := existingFiles[file.fileID]; ok {
			continue
		}
		sourcePath, err := json.Marshal(file.path)
		if err != nil {
			return false, fmt.Errorf("marshal raw volume file source path: %w", err)
		}
		enabled := true
		record := KnowledgeBaseSourceRecord{
			SourceID:     stableID("kb-source", model.ID, kbSourceTypeLocalFile, file.fileID),
			ModelID:      model.ID,
			CatalogID:    file.catalogID,
			DatabaseID:   file.databaseID,
			RawVolumeID:  file.volumeID,
			SourceType:   kbSourceTypeLocalFile,
			SourceFileID: stringPtr(file.fileID),
			KBFileID:     stringPtr(file.fileID),
			DisplayName:  stringPtr(file.fileName),
			SourcePath:   stringPtr(string(sourcePath)),
			Status:       kbSourceStatusSucceeded,
			Enabled:      &enabled,
			Tags:         stringPtr("[]"),
		}
		records = append(records, record)
		existingFiles[file.fileID] = struct{}{}
	}
	if len(records) == 0 {
		return false, nil
	}
	actor := ctxutil.UIDFrom(ctx)
	if actor == "" {
		actor = "system"
	}
	if err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for i := range records {
			if err := insertKnowledgeBaseSourceWithTx(tx, &records[i], actor); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return false, err
	}
	return true, nil
}

func (s *semanticModelService) GetSourceDocument(ctx context.Context, params GetSemanticModelSourceDocumentParams) (*SemanticModelSourceDocument, error) {
	wsID, err := callerWorkspaceID(ctx)
	if err != nil {
		return nil, err
	}
	var response *SemanticModelSourceDocument
	err = coreclient.Execute(ctx, coreclient.FromContext(ctx), func(callCtx context.Context, client *moi.Client) error {
		var callErr error
		response, callErr = s.getSourceDocument(callCtx, client, wsID, params)
		return callErr
	})
	return response, err
}

func (s *semanticModelService) getSourceDocument(ctx context.Context, client *moi.Client, wsID string, params GetSemanticModelSourceDocumentParams) (*SemanticModelSourceDocument, error) {
	modelID := int64(params.ModelID)
	if modelID == 0 || params.SourceID == "" {
		return nil, semanticModelSourceDocumentRequiredError()
	}
	model, err := client.SemanticModels(wsID).Get(ctx, modelID)
	if err != nil {
		return nil, err
	}
	record, err := s.getKnowledgeBaseSource(ctx, modelID, params.SourceID)
	if err != nil {
		return nil, err
	}
	if record.SourceType == kbSourceTypeCatalogTable {
		return nil, tableSourceDocumentUnsupportedError()
	}
	if err := s.enrichKnowledgeBaseFileSourceDisplayName(ctx, &record); err != nil {
		return nil, err
	}
	return s.buildSourceDocument(ctx, model.Files, record, params.SegmentVersionID)
}

// enrichKnowledgeBaseFileSourceDisplayName always refreshes display_name/size/path
// from catalog metadata for file sources so detail matches list (catalog is
// authoritative; client-supplied file_name is not). Uses the same single
// authoritative file id + exact (file_id, volume_id) as list enrichment.
// Missing metadata marks the source failed without inventing another volume or
// falling back across source_file_id / kb_file_id.
func (s *semanticModelService) enrichKnowledgeBaseFileSourceDisplayName(ctx context.Context, record *KnowledgeBaseSourceRecord) error {
	if record == nil {
		return nil
	}
	switch record.SourceType {
	case kbSourceTypeLocalFile, kbSourceTypeCatalogFile:
	default:
		return nil
	}
	fileID := knowledgeBaseSourceRecordFileID(*record)
	if fileID == "" {
		return nil
	}
	if record.RawVolumeID > 0 {
		meta, err := currentCatalogFileMetadataAtVolume(ctx, fileID, record.RawVolumeID)
		if err != nil {
			if knowledgeBaseSourceMetadataFailure(err) {
				markKnowledgeBaseSourceRecordMetadataFailed(record, err)
				return nil
			}
			return err
		}
		return applyKnowledgeBaseFileSourceMetadata(record, meta)
	}
	// Local pending uploads may not yet have raw_volume_id; leave display as-is.
	if record.SourceType == kbSourceTypeLocalFile && record.Status == kbSourceStatusPending {
		return nil
	}
	missing := knowledgeBaseSourceMissingError{msg: fmt.Sprintf("catalog file %s has no recorded volume", fileID)}
	markKnowledgeBaseSourceRecordMetadataFailed(record, missing)
	return nil
}

func (s *semanticModelService) UpdateSourceGovernance(ctx context.Context, params UpdateSemanticModelSourceGovernanceParams) (*UpdateSemanticModelSourceGovernanceResult, error) {
	wsID, err := callerWorkspaceID(ctx)
	if err != nil {
		return nil, err
	}
	var response *UpdateSemanticModelSourceGovernanceResult
	err = coreclient.Execute(ctx, coreclient.FromContext(ctx), func(callCtx context.Context, client *moi.Client) error {
		var callErr error
		response, callErr = s.updateSourceGovernance(callCtx, client, wsID, params)
		return callErr
	})
	return response, err
}

func (s *semanticModelService) updateSourceGovernance(ctx context.Context, client *moi.Client, wsID string, params UpdateSemanticModelSourceGovernanceParams) (*UpdateSemanticModelSourceGovernanceResult, error) {
	modelID := int64(params.ModelID)
	if modelID == 0 || params.SourceID == "" {
		return nil, semanticModelSourceRequiredError()
	}
	model, err := client.SemanticModels(wsID).Get(ctx, modelID)
	if err != nil {
		return nil, err
	}
	record, err := s.getKnowledgeBaseSource(ctx, modelID, params.SourceID)
	if err != nil {
		return nil, err
	}
	if record.SourceType == kbSourceTypeCatalogTable {
		if params.Tags != nil || params.ForceEnabledAfterExpiry != nil {
			return nil, tableSourceGovernanceUnsupportedError()
		}
		nextRecord, err := s.updateKnowledgeBaseTableSourceGovernance(ctx, record, params, ctxutil.UIDFrom(ctx))
		if err != nil {
			return nil, err
		}
		return &UpdateSemanticModelSourceGovernanceResult{
			Source: sourceRecordToSemanticModelSource(nextRecord),
		}, nil
	}
	tagsJSON, err := marshalKnowledgeBaseSourceTags(params.Tags)
	if err != nil {
		return nil, err
	}
	nextRecord, err := s.updateKnowledgeBaseSourceGovernance(ctx, record, params, tagsJSON, toSemanticModelInfo(model).Files, ctxutil.UIDFrom(ctx))
	if err != nil {
		return nil, err
	}
	return &UpdateSemanticModelSourceGovernanceResult{
		Source: sourceRecordToSemanticModelSource(nextRecord),
	}, nil
}

func (s *semanticModelService) DeleteSource(ctx context.Context, params DeleteSemanticModelSourceParams) error {
	wsID, err := callerWorkspaceID(ctx)
	if err != nil {
		return err
	}
	return coreclient.Execute(ctx, coreclient.FromContext(ctx), func(callCtx context.Context, client *moi.Client) error {
		return s.deleteSource(callCtx, client, wsID, params)
	})
}

func (s *semanticModelService) deleteSource(ctx context.Context, client *moi.Client, wsID string, params DeleteSemanticModelSourceParams) error {
	modelID := int64(params.ModelID)
	if modelID == 0 || params.SourceID == "" {
		return semanticModelSourceRequiredError()
	}
	db := ctxutil.TenantDBFrom(ctx)
	if db == nil {
		return fmt.Errorf("tenant db is required")
	}
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txCtx := ctxutil.WithTenantDB(ctx, tx)
		if err := s.lockKnowledgeBaseDataDomainForAppend(txCtx, modelID); err != nil && !errors.Is(err, errKnowledgeBaseDataDomainLockMissing) {
			return err
		}
		if err := lockKnowledgeBaseSourceAndJobsForDelete(txCtx, tx, modelID, params.SourceID); err != nil {
			return err
		}
		model, err := client.SemanticModels(wsID).Get(txCtx, modelID)
		if err != nil {
			return err
		}
		record, err := s.getKnowledgeBaseSource(txCtx, modelID, params.SourceID)
		if err != nil {
			if !isServiceNotFound(err) {
				return err
			}
			record, err = s.getKnowledgeBaseSourceAfterEnsure(txCtx, toSemanticModelInfo(model), modelID, params.SourceID)
			if err != nil {
				return err
			}
		}
		modelInfo := toSemanticModelInfo(model)
		nextFiles, nextTables, err := removeKnowledgeBaseSourceFromSemanticModel(modelInfo, record)
		if err != nil {
			return err
		}
		if _, err := client.SemanticModels(wsID).Update(txCtx, modelID, &moi.SemanticModelUpsertRequest{
			Name:        model.Name,
			Description: model.Description,
			Tables:      nextTables,
			Files:       nextFiles,
		}); err != nil {
			return fmt.Errorf("update semantic model source scope: %w", err)
		}
		return s.deleteKnowledgeBaseSourceRelationWithTx(txCtx, tx, record, nextFiles, false)
	})
}

// Delete takes locks in the same source -> jobs order as reconcile finalization.
// Rejecting an unexpired running claim prevents an old request from reaching a
// source/job pair deleted and recreated with the same stable IDs.
func lockKnowledgeBaseSourceAndJobsForDelete(ctx context.Context, tx *gorm.DB, modelID int64, sourceID string) error {
	rows, err := tx.WithContext(ctx).Raw(`SELECT source_id
		FROM knowledge_base_sources
		WHERE model_id = ? AND source_id = ?
		FOR UPDATE`, modelID, sourceID).Rows()
	if err != nil {
		return fmt.Errorf("lock knowledge base source for delete: %w", err)
	}
	for rows.Next() {
		var lockedSourceID string
		if err := rows.Scan(&lockedSourceID); err != nil {
			rows.Close()
			return fmt.Errorf("scan knowledge base source delete lock: %w", err)
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("lock knowledge base source delete rows: %w", err)
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("close knowledge base source delete lock: %w", err)
	}

	rows, err = tx.WithContext(ctx).Raw(`SELECT job_id,
			CASE WHEN job_status = ? AND updated_at >= DATE_SUB(CURRENT_TIMESTAMP, INTERVAL ? SECOND) THEN 1 ELSE 0 END AS active_claim
		FROM knowledge_base_source_job_runs
		WHERE model_id = ? AND source_id = ?
		FOR UPDATE`, kbSourceJobRunning, int(kbSourceJobClaimLease/time.Second), modelID, sourceID).Rows()
	if err != nil {
		return fmt.Errorf("lock knowledge base source jobs for delete: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var jobID string
		var activeClaim int
		if err := rows.Scan(&jobID, &activeClaim); err != nil {
			return fmt.Errorf("scan knowledge base source job delete lock: %w", err)
		}
		if activeClaim != 0 {
			return knowledgeBaseSourceDeleteConflictError()
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("lock knowledge base source jobs for delete rows: %w", err)
	}
	return nil
}

func (s *semanticModelService) listKnowledgeBaseSourceJobs(ctx context.Context, modelID int64) ([]KnowledgeBaseSourceJob, error) {
	db := ctxutil.TenantDBFrom(ctx)
	if db == nil {
		return []KnowledgeBaseSourceJob{}, nil
	}
	rows, err := db.WithContext(ctx).Raw(`SELECT id, model_id, source_type, source_file_id, kb_file_id, raw_volume_id, job_status, error, segment_version_id, index_version, workflow_execution_id
		FROM knowledge_base_source_jobs
		WHERE model_id = ?
		ORDER BY id ASC`, modelID).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	jobs := make([]KnowledgeBaseSourceJob, 0)
	for rows.Next() {
		var job KnowledgeBaseSourceJob
		if err := rows.Scan(
			&job.ID,
			&job.ModelID,
			&job.SourceType,
			&job.SourceFileID,
			&job.KBFileID,
			&job.RawVolumeID,
			&job.JobStatus,
			&job.Error,
			&job.SegmentVersionID,
			&job.IndexVersion,
			&job.WorkflowExecutionID,
		); err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return jobs, nil
}

func (s *semanticModelService) insertKnowledgeBaseSource(ctx context.Context, source *KnowledgeBaseSourceRecord, actor string) error {
	if source == nil {
		return fmt.Errorf("knowledge base source is required")
	}
	db := ctxutil.TenantDBFrom(ctx)
	if db == nil {
		return fmt.Errorf("tenant db is required")
	}
	return insertKnowledgeBaseSourceWithTx(db.WithContext(ctx), source, actor)
}

func insertKnowledgeBaseSourceWithTx(tx *gorm.DB, source *KnowledgeBaseSourceRecord, actor string) error {
	if source == nil {
		return fmt.Errorf("knowledge base source is required")
	}
	return tx.Exec(`INSERT INTO knowledge_base_sources
		(source_id, model_id, catalog_id, database_id, raw_volume_id, processed_volume_id, source_type, source_file_id, source_table_id, kb_file_id, kb_table_id, display_name, source_path, db_name, table_name, status, error, enabled, expires_at, tags, force_enabled_after_expiry, segment_version_id, index_version, created_by, updated_by)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		source.SourceID, source.ModelID, source.CatalogID, source.DatabaseID, source.RawVolumeID, source.ProcessedVolumeID, source.SourceType,
		source.SourceFileID, source.SourceTableID, source.KBFileID, source.KBTableID, source.DisplayName, source.SourcePath,
		source.DBName, source.TableName, source.Status, source.Error, source.Enabled, source.ExpiresAt, source.Tags, source.ForceEnabled, source.SegmentVersionID, source.IndexVersion,
		actor, actor).Error
}

func insertKnowledgeBaseSourceIdempotentWithTx(tx *gorm.DB, source *KnowledgeBaseSourceRecord, actor string) (bool, error) {
	err := insertKnowledgeBaseSourceWithTx(tx, source, actor)
	if err == nil {
		return true, nil
	}
	if isDuplicateEntryError(err) {
		return false, nil
	}
	return false, err
}

func isKnowledgeBaseSourceRemoved(record KnowledgeBaseSourceRecord) bool {
	return record.Status == kbSourceStatusRemoved
}

func activeKnowledgeBaseSourceRecords(records []KnowledgeBaseSourceRecord) []KnowledgeBaseSourceRecord {
	if len(records) == 0 {
		return nil
	}
	out := make([]KnowledgeBaseSourceRecord, 0, len(records))
	for _, record := range records {
		if isKnowledgeBaseSourceRemoved(record) {
			continue
		}
		out = append(out, record)
	}
	return out
}

func (s *semanticModelService) reactivateKnowledgeBaseSource(ctx context.Context, source *KnowledgeBaseSourceRecord, actor string) error {
	if source == nil {
		return fmt.Errorf("knowledge base source is required")
	}
	db := ctxutil.TenantDBFrom(ctx)
	if db == nil {
		return fmt.Errorf("tenant db is required")
	}
	res := db.WithContext(ctx).Exec(`UPDATE knowledge_base_sources
		SET catalog_id = ?, database_id = ?, raw_volume_id = ?, processed_volume_id = ?, source_type = ?, source_file_id = ?, source_table_id = ?, kb_file_id = ?, kb_table_id = ?, display_name = ?, source_path = ?, db_name = ?, table_name = ?, status = ?, error = NULL, enabled = ?, expires_at = NULL, tags = NULL, force_enabled_after_expiry = FALSE, segment_version_id = NULL, index_version = NULL, updated_by = ?
		WHERE model_id = ? AND source_id = ? AND status = ?`,
		source.CatalogID, source.DatabaseID, source.RawVolumeID, source.ProcessedVolumeID, source.SourceType, source.SourceFileID, source.SourceTableID, source.KBFileID, source.KBTableID,
		source.DisplayName, source.SourcePath, source.DBName, source.TableName, source.Status, source.Enabled, actor, source.ModelID, source.SourceID, kbSourceStatusRemoved)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected != 1 {
		return knowledgeBaseSourceNotFoundError()
	}
	return nil
}

func (s *semanticModelService) findCatalogFileSourceByFileID(ctx context.Context, modelID int64, fileID string) (KnowledgeBaseSourceRecord, bool, error) {
	if fileID == "" {
		return KnowledgeBaseSourceRecord{}, false, nil
	}
	db := ctxutil.TenantDBFrom(ctx)
	if db == nil {
		return KnowledgeBaseSourceRecord{}, false, fmt.Errorf("tenant db is required")
	}
	rows, err := db.WithContext(ctx).Raw(`SELECT `+knowledgeBaseSourceSelectColumns+`
		FROM knowledge_base_sources kbs
		WHERE kbs.model_id = ? AND kbs.source_type = ? AND (kbs.source_file_id = ? OR kbs.kb_file_id = ?)
		ORDER BY kbs.created_at ASC, kbs.source_id ASC
		LIMIT 1`, modelID, kbSourceTypeCatalogFile, fileID, fileID).Rows()
	if err != nil {
		return KnowledgeBaseSourceRecord{}, false, err
	}
	return scanFirstKnowledgeBaseSourceRecord(rows)
}

func (s *semanticModelService) findKnowledgeBaseSourceByID(ctx context.Context, modelID int64, sourceID string) (KnowledgeBaseSourceRecord, bool, error) {
	if sourceID == "" {
		return KnowledgeBaseSourceRecord{}, false, nil
	}
	db := ctxutil.TenantDBFrom(ctx)
	if db == nil {
		return KnowledgeBaseSourceRecord{}, false, fmt.Errorf("tenant db is required")
	}
	rows, err := db.WithContext(ctx).Raw(`SELECT `+knowledgeBaseSourceSelectColumns+`
		FROM knowledge_base_sources kbs
		WHERE kbs.model_id = ? AND kbs.source_id = ?
		LIMIT 1`, modelID, sourceID).Rows()
	if err != nil {
		return KnowledgeBaseSourceRecord{}, false, err
	}
	return scanFirstKnowledgeBaseSourceRecord(rows)
}

func (s *semanticModelService) findCatalogTableSourceBySourceTableID(ctx context.Context, modelID int64, sourceTableID int64) (KnowledgeBaseSourceRecord, bool, error) {
	if sourceTableID <= 0 {
		return KnowledgeBaseSourceRecord{}, false, nil
	}
	db := ctxutil.TenantDBFrom(ctx)
	if db == nil {
		return KnowledgeBaseSourceRecord{}, false, fmt.Errorf("tenant db is required")
	}
	// A table is already associated with a knowledge base whether it was selected
	// directly or generated by a structured upload.
	rows, err := db.WithContext(ctx).Raw(`SELECT `+knowledgeBaseSourceSelectColumns+`
			FROM knowledge_base_sources kbs
			WHERE kbs.model_id = ? AND kbs.source_type = ?
				AND (kbs.source_table_id = ? OR kbs.kb_table_id = ?)
			ORDER BY
				CASE
					WHEN kbs.status = ?
						AND kbs.source_table_id IS NOT NULL AND kbs.source_table_id > 0
						AND kbs.db_name IS NOT NULL AND kbs.db_name <> ''
						AND kbs.table_name IS NOT NULL AND kbs.table_name <> ''
					THEN 0
					ELSE 1
				END,
				kbs.created_at ASC, kbs.source_id ASC
			LIMIT 1`, modelID, kbSourceTypeCatalogTable, sourceTableID, sourceTableID, kbSourceStatusSucceeded).Rows()
	if err != nil {
		return KnowledgeBaseSourceRecord{}, false, err
	}
	return scanFirstKnowledgeBaseSourceRecord(rows)
}

func findKnowledgeBaseSourceByRecordIdentity(tx *gorm.DB, source *KnowledgeBaseSourceRecord) (KnowledgeBaseSourceRecord, bool, error) {
	if source == nil {
		return KnowledgeBaseSourceRecord{}, false, nil
	}
	var rows *sql.Rows
	var err error
	switch {
	case source.KBFileID != nil && *source.KBFileID != "":
		rows, err = tx.Raw(`SELECT `+knowledgeBaseSourceSelectColumns+`
			FROM knowledge_base_sources kbs
			WHERE kbs.model_id = ? AND kbs.kb_file_id = ?
			ORDER BY kbs.created_at ASC, kbs.source_id ASC
			LIMIT 1`, source.ModelID, *source.KBFileID).Rows()
	case source.SourceFileID != nil && *source.SourceFileID != "":
		rows, err = tx.Raw(`SELECT `+knowledgeBaseSourceSelectColumns+`
			FROM knowledge_base_sources kbs
			WHERE kbs.model_id = ? AND kbs.source_file_id = ?
			ORDER BY kbs.created_at ASC, kbs.source_id ASC
			LIMIT 1`, source.ModelID, *source.SourceFileID).Rows()
	case source.KBTableID != nil && *source.KBTableID > 0:
		rows, err = tx.Raw(`SELECT `+knowledgeBaseSourceSelectColumns+`
			FROM knowledge_base_sources kbs
			WHERE kbs.model_id = ? AND kbs.kb_table_id = ?
			ORDER BY kbs.created_at ASC, kbs.source_id ASC
			LIMIT 1`, source.ModelID, *source.KBTableID).Rows()
	case source.SourceTableID != nil && *source.SourceTableID > 0:
		rows, err = tx.Raw(`SELECT `+knowledgeBaseSourceSelectColumns+`
			FROM knowledge_base_sources kbs
			WHERE kbs.model_id = ? AND kbs.source_table_id = ?
			ORDER BY kbs.created_at ASC, kbs.source_id ASC
			LIMIT 1`, source.ModelID, *source.SourceTableID).Rows()
	default:
		return KnowledgeBaseSourceRecord{}, false, nil
	}
	if err != nil {
		return KnowledgeBaseSourceRecord{}, false, err
	}
	return scanFirstKnowledgeBaseSourceRecord(rows)
}

func isDuplicateEntryError(err error) bool {
	if err == nil {
		return false
	}
	var mysqlErr *mysqlDriver.MySQLError
	if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
		return true
	}
	return strings.Contains(err.Error(), "Duplicate entry")
}

func (s *semanticModelService) insertCatalogFileSourceProcessing(ctx context.Context, source *KnowledgeBaseSourceRecord, actor string) error {
	if source == nil {
		return fmt.Errorf("knowledge base source is required")
	}
	db := ctxutil.TenantDBFrom(ctx)
	if db == nil {
		return fmt.Errorf("tenant db is required")
	}
	return insertKnowledgeBaseSourceWithTx(db.WithContext(ctx), source, actor)
}

func (s *semanticModelService) upsertCatalogTableSourceProcessing(ctx context.Context, source *KnowledgeBaseSourceRecord, actor string, updateExistingBySourceID bool) error {
	if source == nil {
		return fmt.Errorf("knowledge base source is required")
	}
	db := ctxutil.TenantDBFrom(ctx)
	if db == nil {
		return fmt.Errorf("tenant db is required")
	}
	if updateExistingBySourceID {
		res := db.WithContext(ctx).Exec(`UPDATE knowledge_base_sources
			SET catalog_id = ?, database_id = ?, raw_volume_id = ?, processed_volume_id = ?, source_table_id = ?, kb_table_id = ?, display_name = ?, source_path = ?, db_name = ?, table_name = ?, status = ?, error = NULL, enabled = ?, expires_at = NULL, tags = NULL, force_enabled_after_expiry = FALSE, segment_version_id = NULL, index_version = NULL, updated_by = ?
			WHERE source_id = ? AND model_id = ? AND source_type = ?`,
			source.CatalogID, source.DatabaseID, source.RawVolumeID, source.ProcessedVolumeID, source.SourceTableID, source.KBTableID, source.DisplayName, source.SourcePath, source.DBName, source.TableName, source.Status, source.Enabled, actor,
			source.SourceID, source.ModelID, source.SourceType)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 0 {
			return nil
		}
	}
	res := db.WithContext(ctx).Exec(`UPDATE knowledge_base_sources
		SET catalog_id = ?, database_id = ?, raw_volume_id = ?, processed_volume_id = ?, kb_table_id = ?, display_name = ?, source_path = ?, db_name = ?, table_name = ?, status = ?, error = NULL, enabled = ?, expires_at = NULL, tags = NULL, force_enabled_after_expiry = FALSE, segment_version_id = NULL, index_version = NULL, updated_by = ?
		WHERE model_id = ? AND source_type = ? AND source_table_id = ?`,
		source.CatalogID, source.DatabaseID, source.RawVolumeID, source.ProcessedVolumeID, source.KBTableID, source.DisplayName, source.SourcePath, source.DBName, source.TableName, source.Status, source.Enabled, actor,
		source.ModelID, source.SourceType, source.SourceTableID)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return insertKnowledgeBaseSourceWithTx(db.WithContext(ctx), source, actor)
	}
	return nil
}

func (s *semanticModelService) updateKnowledgeBaseTableSourceSucceeded(ctx context.Context, source *KnowledgeBaseSourceRecord, actor string) error {
	db := ctxutil.TenantDBFrom(ctx)
	if db == nil {
		return fmt.Errorf("tenant db is required")
	}
	res := db.WithContext(ctx).Exec(`UPDATE knowledge_base_sources
		SET source_type = ?, source_file_id = ?, source_table_id = ?, kb_file_id = ?, kb_table_id = ?, display_name = ?, source_path = ?, db_name = ?, table_name = ?, status = ?, error = NULL, updated_by = ?
		WHERE source_id = ?`,
		source.SourceType, source.SourceFileID, source.SourceTableID, source.KBFileID, source.KBTableID, source.DisplayName, source.SourcePath, source.DBName, source.TableName, source.Status, actor, source.SourceID)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected != 1 {
		return knowledgeBaseSourceNotFoundError()
	}
	return nil
}

func (s *semanticModelService) markKnowledgeBaseSourceFailed(ctx context.Context, sourceID, msg, actor string) error {
	db := ctxutil.TenantDBFrom(ctx)
	if db == nil {
		return fmt.Errorf("tenant db is required")
	}
	return db.WithContext(ctx).Exec(`UPDATE knowledge_base_sources
		SET status = ?, error = ?, updated_by = ?
		WHERE source_id = ?`,
		kbSourceStatusFailed, msg, actor, sourceID).Error
}

func (s *semanticModelService) markKnowledgeBaseSourcePending(ctx context.Context, sourceID, actor string) error {
	db := ctxutil.TenantDBFrom(ctx)
	if db == nil {
		return fmt.Errorf("tenant db is required")
	}
	return db.WithContext(ctx).Exec(`UPDATE knowledge_base_sources
		SET status = ?, error = NULL, updated_by = ?
		WHERE source_id = ?`, kbSourceStatusPending, actor, sourceID).Error
}

func (s *semanticModelService) markCreateKnowledgeBaseSourcesFailed(ctx context.Context, records []KnowledgeBaseSourceRecord, jobs []KnowledgeBaseSourceJobRun, cause error, actor string) error {
	if cause == nil {
		return nil
	}
	out := cause
	msg := cause.Error()
	for i := range records {
		records[i].Status = kbSourceStatusFailed
		records[i].Error = stringPtr(msg)
		if err := s.markKnowledgeBaseSourceFailed(ctx, records[i].SourceID, msg, actor); err != nil {
			out = fmt.Errorf("%w; mark knowledge base source %s failed: %v", out, records[i].SourceID, err)
		}
	}
	for i := range jobs {
		jobs[i].JobStatus = kbSourceJobFailed
		jobs[i].Error = stringPtr(msg)
		if err := s.markKnowledgeBaseSourceJobFailed(ctx, jobs[i].JobID, msg, actor); err != nil {
			out = fmt.Errorf("%w; mark knowledge base source job %s failed: %v", out, jobs[i].JobID, err)
		}
	}
	return out
}

func (s *semanticModelService) markCreateKnowledgeBasePlanFailed(ctx context.Context, result *createKnowledgeBaseSourcesResult, recordIndex int, jobIndexes []int, cause error, actor string) error {
	if cause == nil {
		return nil
	}
	out := cause
	msg := cause.Error()
	if result != nil && recordIndex >= 0 && recordIndex < len(result.records) {
		record := &result.records[recordIndex]
		record.Status = kbSourceStatusFailed
		record.Error = stringPtr(msg)
		if err := s.markKnowledgeBaseSourceFailed(ctx, record.SourceID, msg, actor); err != nil {
			out = fmt.Errorf("%w; mark knowledge base source %s failed: %v", out, record.SourceID, err)
		}
	}
	if result != nil {
		for _, jobIndex := range jobIndexes {
			if jobIndex < 0 || jobIndex >= len(result.jobs) {
				continue
			}
			job := &result.jobs[jobIndex]
			job.JobStatus = kbSourceJobFailed
			job.Error = stringPtr(msg)
			if err := s.markKnowledgeBaseSourceJobFailed(ctx, job.JobID, msg, actor); err != nil {
				out = fmt.Errorf("%w; mark knowledge base source job %s failed: %v", out, job.JobID, err)
			}
		}
	}
	return out
}

func (s *semanticModelService) lockKnowledgeBaseDataDomainForAppend(ctx context.Context, modelID int64) error {
	db := ctxutil.TenantDBFrom(ctx)
	if db == nil {
		return fmt.Errorf("tenant db is required")
	}
	rows, err := db.WithContext(ctx).Raw(`SELECT model_id
		FROM knowledge_base_data_domains
		WHERE model_id = ?
		FOR UPDATE`, modelID).Rows()
	if err != nil {
		return fmt.Errorf("lock knowledge base data domain: %w", err)
	}
	defer rows.Close()
	if !rows.Next() {
		return fmt.Errorf("%w: %w", errKnowledgeBaseDataDomainLockMissing, knowledgeBaseDataDomainNotFoundError())
	}
	var lockedModelID int64
	if err := rows.Scan(&lockedModelID); err != nil {
		return fmt.Errorf("scan locked knowledge base data domain: %w", err)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("lock knowledge base data domain rows: %w", err)
	}
	return nil
}

func (s *semanticModelService) rollbackCreatedKnowledgeBaseSources(ctx context.Context, _ []SemanticModelSource, _ []KnowledgeBaseSourceJobRun, records []KnowledgeBaseSourceRecord, cause error) error {
	if cleanupErr := s.cleanupCopiedKnowledgeBaseFiles(ctx, records); cleanupErr != nil {
		cause = fmt.Errorf("%w; cleanup copied knowledge base files: %v", cause, cleanupErr)
	}
	return cause
}

func (s *semanticModelService) cleanupCopiedKnowledgeBaseFiles(ctx context.Context, records []KnowledgeBaseSourceRecord) error {
	if len(records) == 0 {
		return nil
	}
	for _, source := range records {
		if source.SourceType != kbSourceTypeLocalFile || source.KBFileID == nil || source.RawVolumeID <= 0 {
			continue
		}
		if s.fileService == nil {
			return fmt.Errorf("catalog file service is not configured")
		}
		if err := s.fileService.DeleteFileFromVolume(ctx, source.RawVolumeID, *source.KBFileID); err != nil {
			return fmt.Errorf("delete copied file %s from raw volume %d: %w", *source.KBFileID, source.RawVolumeID, err)
		}
	}
	return nil
}

func (s *semanticModelService) deleteKnowledgeBaseSourceRelationWithTx(ctx context.Context, tx *gorm.DB, record KnowledgeBaseSourceRecord, modelFiles json.RawMessage, disableVectors bool) error {
	if tx == nil {
		return fmt.Errorf("tenant transaction is required")
	}
	actor := semanticModelActor(ctx)
	if err := tx.Exec(`DELETE FROM knowledge_base_chunk_recall_stats WHERE model_id = ? AND source_id = ?`, record.ModelID, record.SourceID).Error; err != nil {
		return err
	}
	if err := tx.Exec(`DELETE FROM knowledge_base_segments WHERE model_id = ? AND source_id = ?`, record.ModelID, record.SourceID).Error; err != nil {
		return err
	}
	if err := tx.Exec(`DELETE FROM knowledge_base_segment_versions WHERE model_id = ? AND source_id = ?`, record.ModelID, record.SourceID).Error; err != nil {
		return err
	}
	res := tx.Exec(`UPDATE knowledge_base_sources
			SET status = ?, error = NULL, enabled = FALSE, expires_at = NULL, tags = NULL, force_enabled_after_expiry = FALSE, segment_version_id = NULL, index_version = NULL, updated_by = ?
			WHERE model_id = ? AND source_id = ? AND status <> ?`,
		kbSourceStatusRemoved, actor, record.ModelID, record.SourceID, kbSourceStatusRemoved)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected != 1 {
		return knowledgeBaseSourceNotFoundError()
	}
	if err := tx.Exec(`DELETE FROM knowledge_base_source_job_runs WHERE model_id = ? AND source_id = ?`, record.ModelID, record.SourceID).Error; err != nil {
		return err
	}
	if disableVectors {
		if err := syncKnowledgeBaseSourceVectorDisabled(tx, record, modelFiles, true); err != nil {
			return err
		}
	}
	return nil
}

func (s *semanticModelService) listKnowledgeBaseSources(ctx context.Context, modelID int64) ([]KnowledgeBaseSourceRecord, error) {
	return s.listKnowledgeBaseSourceRows(ctx, modelID, false)
}

func (s *semanticModelService) listKnowledgeBaseSourceJobRunsForSourceIDs(ctx context.Context, modelID int64, sourceIDs []string) ([]KnowledgeBaseSourceJobRun, error) {
	if len(sourceIDs) == 0 {
		return []KnowledgeBaseSourceJobRun{}, nil
	}
	db := ctxutil.TenantDBFrom(ctx)
	if db == nil {
		return []KnowledgeBaseSourceJobRun{}, nil
	}
	rows, err := db.WithContext(ctx).Raw(`SELECT job_id, source_id, model_id, job_type, job_status, idempotency_key, operation_id, workflow_execution_id, runtime_actor_moi_user_id, runtime_effective_role_id, runtime_is_workspace_owner, source_file_id, kb_file_id, source_table_id, kb_table_id, retry_count, next_retry_at, error, UNIX_TIMESTAMP(created_at), UNIX_TIMESTAMP(updated_at)
		FROM knowledge_base_source_job_runs
		WHERE model_id = ? AND source_id IN ?
		ORDER BY source_id ASC, created_at ASC, job_id ASC`, modelID, sourceIDs).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	jobs := make([]KnowledgeBaseSourceJobRun, 0)
	for rows.Next() {
		var job KnowledgeBaseSourceJobRun
		if err := rows.Scan(&job.JobID, &job.SourceID, &job.ModelID, &job.JobType, &job.JobStatus, &job.IdempotencyKey, &job.OperationID, &job.WorkflowExecutionID, &job.RuntimeActorMOIUserID, &job.RuntimeEffectiveRoleID, &job.RuntimeIsWorkspaceOwner, &job.SourceFileID, &job.KBFileID, &job.SourceTableID, &job.KBTableID, &job.RetryCount, &job.NextRetryAt, &job.Error, &job.CreatedAt, &job.UpdatedAt); err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	return jobs, rows.Err()
}

func (s *semanticModelService) listKnowledgeBaseSourceRowsByIDs(ctx context.Context, modelID int64, sourceIDs []string) ([]KnowledgeBaseSourceRecord, error) {
	if len(sourceIDs) == 0 {
		return []KnowledgeBaseSourceRecord{}, nil
	}
	db := ctxutil.TenantDBFrom(ctx)
	if db == nil {
		return []KnowledgeBaseSourceRecord{}, nil
	}
	rows, err := db.WithContext(ctx).Raw(`SELECT kbs.source_id AS source_id, kbs.model_id AS model_id, kbs.catalog_id AS catalog_id, kbs.database_id AS database_id, kbs.raw_volume_id AS raw_volume_id, kbs.processed_volume_id AS processed_volume_id, kbs.source_type AS source_type, kbs.source_file_id AS source_file_id, kbs.source_table_id AS source_table_id, kbs.kb_file_id AS kb_file_id, kbs.kb_table_id AS kb_table_id, kbs.display_name AS display_name, kbs.source_path AS source_path, kbs.db_name AS db_name, kbs.table_name AS table_name, kbs.status AS status, kbs.error AS error, kbs.enabled AS enabled, kbs.expires_at AS expires_at, kbs.tags AS tags, kbs.force_enabled_after_expiry AS force_enabled_after_expiry, kbs.segment_version_id AS segment_version_id, kbs.index_version AS index_version, f.size AS size_bytes, NULL AS row_count, kbs.created_by AS created_by, kbs.updated_by AS updated_by, UNIX_TIMESTAMP(kbs.updated_at) AS updated_at
		FROM knowledge_base_sources kbs
		LEFT JOIN `+"`file`"+` f ON f.file_id = kbs.kb_file_id
		WHERE kbs.model_id = ? AND kbs.source_id IN ? AND kbs.status <> ?
		ORDER BY kbs.source_id ASC`, modelID, sourceIDs, kbSourceStatusRemoved).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]KnowledgeBaseSourceRecord, 0, len(sourceIDs))
	for rows.Next() {
		record, err := scanKnowledgeBaseSourceRecord(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, record)
	}
	return out, rows.Err()
}

func (s *semanticModelService) listKnowledgeBaseSourceRows(ctx context.Context, modelID int64, includeRemoved bool) ([]KnowledgeBaseSourceRecord, error) {
	db := ctxutil.TenantDBFrom(ctx)
	if db == nil {
		return []KnowledgeBaseSourceRecord{}, nil
	}
	statusFilter := " AND kbs.status <> '" + kbSourceStatusRemoved + "'"
	if includeRemoved {
		statusFilter = ""
	}
	rows, err := db.WithContext(ctx).Raw(`SELECT kbs.source_id AS source_id, kbs.model_id AS model_id, kbs.catalog_id AS catalog_id, kbs.database_id AS database_id, kbs.raw_volume_id AS raw_volume_id, kbs.processed_volume_id AS processed_volume_id, kbs.source_type AS source_type, kbs.source_file_id AS source_file_id, kbs.source_table_id AS source_table_id, kbs.kb_file_id AS kb_file_id, kbs.kb_table_id AS kb_table_id, kbs.display_name AS display_name, kbs.source_path AS source_path, kbs.db_name AS db_name, kbs.table_name AS table_name, kbs.status AS status, kbs.error AS error, kbs.enabled AS enabled, kbs.expires_at AS expires_at, kbs.tags AS tags, kbs.force_enabled_after_expiry AS force_enabled_after_expiry, kbs.segment_version_id AS segment_version_id, kbs.index_version AS index_version, f.size AS size_bytes, NULL AS row_count, kbs.created_by AS created_by, kbs.updated_by AS updated_by, UNIX_TIMESTAMP(kbs.updated_at) AS updated_at
		FROM knowledge_base_sources kbs
		LEFT JOIN `+"`file`"+` f ON f.file_id = kbs.kb_file_id
		WHERE kbs.model_id = ?`+statusFilter+`
		ORDER BY kbs.created_at ASC, kbs.source_id ASC`, modelID).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]KnowledgeBaseSourceRecord, 0)
	for rows.Next() {
		source, err := scanKnowledgeBaseSourceRecord(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, source)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *semanticModelService) getKnowledgeBaseSource(ctx context.Context, modelID int64, sourceID string) (KnowledgeBaseSourceRecord, error) {
	db := ctxutil.TenantDBFrom(ctx)
	if db == nil {
		return KnowledgeBaseSourceRecord{}, knowledgeBaseSourceNotFoundError()
	}
	rows, err := db.WithContext(ctx).Raw(`SELECT kbs.source_id AS source_id, kbs.model_id AS model_id, kbs.catalog_id AS catalog_id, kbs.database_id AS database_id, kbs.raw_volume_id AS raw_volume_id, kbs.processed_volume_id AS processed_volume_id, kbs.source_type AS source_type, kbs.source_file_id AS source_file_id, kbs.source_table_id AS source_table_id, kbs.kb_file_id AS kb_file_id, kbs.kb_table_id AS kb_table_id, kbs.display_name AS display_name, kbs.source_path AS source_path, kbs.db_name AS db_name, kbs.table_name AS table_name, kbs.status AS status, kbs.error AS error, kbs.enabled AS enabled, kbs.expires_at AS expires_at, kbs.tags AS tags, kbs.force_enabled_after_expiry AS force_enabled_after_expiry, kbs.segment_version_id AS segment_version_id, kbs.index_version AS index_version, f.size AS size_bytes, NULL AS row_count, kbs.created_by AS created_by, kbs.updated_by AS updated_by, UNIX_TIMESTAMP(kbs.updated_at) AS updated_at
		FROM knowledge_base_sources kbs
		LEFT JOIN `+"`file`"+` f ON f.file_id = kbs.kb_file_id
		WHERE kbs.model_id = ? AND kbs.source_id = ? AND kbs.status <> '`+kbSourceStatusRemoved+`'
		LIMIT 1`, modelID, sourceID).Rows()
	if err != nil {
		return KnowledgeBaseSourceRecord{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		return KnowledgeBaseSourceRecord{}, knowledgeBaseSourceNotFoundError()
	}
	source, err := scanKnowledgeBaseSourceRecord(rows)
	if err != nil {
		return KnowledgeBaseSourceRecord{}, err
	}
	if err := rows.Err(); err != nil {
		return KnowledgeBaseSourceRecord{}, err
	}
	return source, nil
}

func scanFirstKnowledgeBaseSourceRecord(rows *sql.Rows) (KnowledgeBaseSourceRecord, bool, error) {
	defer rows.Close()
	if !rows.Next() {
		return KnowledgeBaseSourceRecord{}, false, rows.Err()
	}
	record, err := scanKnowledgeBaseSourceRecord(rows)
	if err != nil {
		return KnowledgeBaseSourceRecord{}, false, err
	}
	if err := rows.Err(); err != nil {
		return KnowledgeBaseSourceRecord{}, false, err
	}
	return record, true, nil
}

func scanKnowledgeBaseSourceRecord(rows *sql.Rows) (KnowledgeBaseSourceRecord, error) {
	columns, err := rows.Columns()
	if err != nil {
		return KnowledgeBaseSourceRecord{}, err
	}
	var source KnowledgeBaseSourceRecord
	discard := make([]sql.RawBytes, len(columns))
	dest := make([]any, len(columns))
	for i, column := range columns {
		switch column {
		case "source_id":
			dest[i] = &source.SourceID
		case "model_id":
			dest[i] = &source.ModelID
		case "catalog_id":
			dest[i] = &source.CatalogID
		case "database_id":
			dest[i] = &source.DatabaseID
		case "raw_volume_id":
			dest[i] = &source.RawVolumeID
		case "processed_volume_id":
			dest[i] = &source.ProcessedVolumeID
		case "source_type":
			dest[i] = &source.SourceType
		case "source_file_id":
			dest[i] = &source.SourceFileID
		case "source_table_id":
			dest[i] = &source.SourceTableID
		case "kb_file_id":
			dest[i] = &source.KBFileID
		case "kb_table_id":
			dest[i] = &source.KBTableID
		case "display_name":
			dest[i] = &source.DisplayName
		case "source_path":
			dest[i] = &source.SourcePath
		case "db_name":
			dest[i] = &source.DBName
		case "table_name":
			dest[i] = &source.TableName
		case "status":
			dest[i] = &source.Status
		case "error":
			dest[i] = &source.Error
		case "enabled":
			dest[i] = &source.Enabled
		case "expires_at":
			dest[i] = &source.ExpiresAt
		case "tags":
			dest[i] = &source.Tags
		case "force_enabled_after_expiry":
			dest[i] = &source.ForceEnabled
		case "segment_version_id":
			dest[i] = &source.SegmentVersionID
		case "index_version":
			dest[i] = &source.IndexVersion
		case "size_bytes":
			dest[i] = &source.SizeBytes
		case "row_count":
			dest[i] = &source.RowCount
		case "created_by":
			dest[i] = &source.CreatedBy
		case "updated_by":
			dest[i] = &source.UpdatedBy
		case "updated_at":
			dest[i] = &source.UpdatedAt
		default:
			dest[i] = &discard[i]
		}
	}
	if err := rows.Scan(dest...); err != nil {
		return KnowledgeBaseSourceRecord{}, err
	}
	return source, nil
}

func (s *semanticModelService) updateKnowledgeBaseSourceGovernance(ctx context.Context, record KnowledgeBaseSourceRecord, params UpdateSemanticModelSourceGovernanceParams, tagsJSON *string, modelFiles json.RawMessage, actor string) (KnowledgeBaseSourceRecord, error) {
	db := ctxutil.TenantDBFrom(ctx)
	if db == nil {
		return KnowledgeBaseSourceRecord{}, fmt.Errorf("tenant db is required")
	}
	if actor == "" {
		actor = "system"
	}
	returned := KnowledgeBaseSourceRecord{}
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		next := record
		currentGovernance := knowledgeBaseSourceGovernance(record)
		if params.Tags != nil {
			next.Tags = tagsJSON
		}
		if params.ExpiresAt.Set {
			next.ExpiresAt = params.ExpiresAt.Value
		}
		if params.Enabled != nil {
			next.Enabled = params.Enabled
		}
		if params.ForceEnabledAfterExpiry != nil {
			next.ForceEnabled = *params.ForceEnabledAfterExpiry
		}
		if params.Enabled != nil {
			if !*params.Enabled {
				next.ForceEnabled = false
			} else if !currentGovernance.EffectiveEnabled && knowledgeBaseSourceGovernance(next).Expired {
				next.ForceEnabled = true
			}
		}
		res := tx.Exec(`UPDATE knowledge_base_sources
				SET tags = ?, expires_at = ?, enabled = ?, force_enabled_after_expiry = ?, updated_by = ?
				WHERE model_id = ? AND source_id = ?`,
			next.Tags, next.ExpiresAt, next.Enabled, next.ForceEnabled, actor, record.ModelID, record.SourceID)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			return knowledgeBaseSourceNotFoundError()
		}
		nextGovernance := knowledgeBaseSourceGovernance(next)
		if currentGovernance.EffectiveEnabled != nextGovernance.EffectiveEnabled {
			disabled := !nextGovernance.EffectiveEnabled
			if err := syncKnowledgeBaseSourceVectorDisabled(tx, next, modelFiles, disabled); err != nil {
				return err
			}
		}
		returned = next
		return nil
	})
	if err != nil {
		return KnowledgeBaseSourceRecord{}, err
	}
	return returned, nil
}

func (s *semanticModelService) updateKnowledgeBaseTableSourceGovernance(ctx context.Context, record KnowledgeBaseSourceRecord, params UpdateSemanticModelSourceGovernanceParams, actor string) (KnowledgeBaseSourceRecord, error) {
	db := ctxutil.TenantDBFrom(ctx)
	if db == nil {
		return KnowledgeBaseSourceRecord{}, fmt.Errorf("tenant db is required")
	}
	if actor == "" {
		actor = "system"
	}
	next := record
	currentGovernance := knowledgeBaseSourceGovernance(record)
	if params.Enabled != nil {
		next.Enabled = params.Enabled
	}
	if params.ExpiresAt.Set {
		next.ExpiresAt = params.ExpiresAt.Value
	}
	if params.Enabled != nil {
		if !*params.Enabled {
			next.ForceEnabled = false
		} else if !currentGovernance.EffectiveEnabled && knowledgeBaseSourceGovernance(next).Expired {
			next.ForceEnabled = true
		}
	}
	res := db.WithContext(ctx).Exec(`UPDATE knowledge_base_sources
			SET enabled = ?, expires_at = ?, force_enabled_after_expiry = ?, updated_by = ?
			WHERE model_id = ? AND source_id = ?`,
		next.Enabled, next.ExpiresAt, next.ForceEnabled, actor, record.ModelID, record.SourceID)
	if res.Error != nil {
		return KnowledgeBaseSourceRecord{}, res.Error
	}
	if res.RowsAffected != 1 {
		return KnowledgeBaseSourceRecord{}, knowledgeBaseSourceNotFoundError()
	}
	return next, nil
}

func syncKnowledgeBaseSourceVectorDisabled(tx *gorm.DB, record KnowledgeBaseSourceRecord, modelFiles json.RawMessage, disabled bool) error {
	if record.KBFileID == nil || *record.KBFileID == "" {
		return nil
	}
	var files map[string]any
	if len(modelFiles) > 0 {
		if err := json.Unmarshal(modelFiles, &files); err != nil {
			return fmt.Errorf("parse semantic model files: %w", err)
		}
	}
	for _, key := range []string{"vector_table", "image_vector_table"} {
		table, _ := files[key].(string)
		if strings.TrimSpace(table) == "" {
			continue
		}
		quotedTable, err := quoteQualifiedSQLIdentifier(table)
		if err != nil {
			return err
		}
		if err := tx.Exec(fmt.Sprintf("UPDATE %s SET disabled = ? WHERE file_id = ?", quotedTable), disabled, *record.KBFileID).Error; err != nil {
			return vectorTableUnavailableWithCauseError(fmt.Errorf("update vector table %s disabled: %w", table, err))
		}
	}
	return nil
}

func quoteQualifiedSQLIdentifier(name string) (string, error) {
	parts := strings.Split(name, ".")
	if len(parts) == 0 || len(parts) > 2 {
		return "", fmt.Errorf("invalid vector table name: %q", name)
	}
	quoted := make([]string, 0, len(parts))
	for _, part := range parts {
		if !isValidSQLIdentifier(part) {
			return "", fmt.Errorf("invalid vector table name: %q", name)
		}
		quoted = append(quoted, fmt.Sprintf("`%s`", sanitizeSQLIdentifier(part)))
	}
	return strings.Join(quoted, "."), nil
}

func applyKnowledgeBaseSourceJobStatus(items []SemanticModelSource, jobs []KnowledgeBaseSourceJobRun) []SemanticModelSource {
	jobsBySourceID := make(map[string][]KnowledgeBaseSourceJobRun, len(jobs))
	for _, job := range jobs {
		jobsBySourceID[job.SourceID] = append(jobsBySourceID[job.SourceID], job)
	}
	for i := range items {
		sourceJobs := jobsBySourceID[items[i].SourceID]
		if len(sourceJobs) == 0 {
			continue
		}
		if items[i].IngestStatus != nil && *items[i].IngestStatus == kbSourceStatusFailed && items[i].Error != nil && *items[i].Error != "" {
			continue
		}
		status, errMsg, segmentVersionID := deriveKnowledgeBaseSourceStatus(items[i].SourceType, sourceJobs)
		if items[i].SourceType == SemanticModelSourceTypeTable && status == kbSourceStatusSucceeded && !semanticModelTableSourceAssociationComplete(items[i]) {
			status = kbSourceStatusPending
		}
		if items[i].SourceType == SemanticModelSourceTypeFile && status == kbSourceStatusSucceeded &&
			hasJobType(sourceJobs, kbJobTypeRAGIngest) && !semanticModelFileSourceCurrentSegmentReady(items[i], sourceJobs) {
			status = kbSourceStatusPending
		}
		items[i].IngestStatus = &status
		items[i].Error = errMsg
		if segmentVersionID != nil {
			items[i].SegmentVersionID = segmentVersionID
		}
	}
	return items
}

func semanticModelFileSourceCurrentSegmentReady(source SemanticModelSource, jobs []KnowledgeBaseSourceJobRun) bool {
	if source.SourceType != SemanticModelSourceTypeFile {
		return false
	}
	if source.IndexVersion == nil || *source.IndexVersion <= 0 {
		return false
	}
	if source.SegmentVersionID != nil && *source.SegmentVersionID != "" {
		return true
	}
	return hasLegacyRAGSourceJob(jobs)
}

func hasLegacyRAGSourceJob(jobs []KnowledgeBaseSourceJobRun) bool {
	for _, job := range jobs {
		if job.JobType == kbJobTypeRAGIngest && job.OperationID != nil && strings.HasPrefix(*job.OperationID, "legacy_source_job:") {
			return true
		}
	}
	return false
}

func semanticModelTableSourceAssociationComplete(source SemanticModelSource) bool {
	if source.SourceType != SemanticModelSourceTypeTable {
		return false
	}
	hasTableID := source.KBTableID != nil && *source.KBTableID > 0
	if !hasTableID {
		hasTableID = source.ResourceID != ""
	}
	return hasTableID &&
		source.DBName != nil && *source.DBName != "" &&
		source.TableName != nil && *source.TableName != ""
}

func knowledgeBaseTableSourceRecordAssociationComplete(record KnowledgeBaseSourceRecord) bool {
	if record.SourceType != kbSourceTypeCatalogTable {
		return false
	}
	tableID := int64(0)
	if record.KBTableID != nil {
		tableID = *record.KBTableID
	}
	if tableID <= 0 && record.SourceTableID != nil {
		tableID = *record.SourceTableID
	}
	return tableID > 0 &&
		record.DBName != nil && *record.DBName != "" &&
		record.TableName != nil && *record.TableName != ""
}

func deriveKnowledgeBaseSourceStatus(sourceType SemanticModelSourceType, jobs []KnowledgeBaseSourceJobRun) (string, *string, *string) {
	var segmentVersionID *string
	for _, job := range jobs {
		if job.JobStatus == kbSourceJobFailed {
			return kbSourceStatusFailed, job.Error, segmentVersionID
		}
	}
	if sourceType == SemanticModelSourceTypeTable {
		hasTableLoad := false
		tableLoadSucceeded := true
		for _, job := range jobs {
			if job.JobType == kbJobTypeTableClone {
				if job.JobStatus == kbSourceJobSucceeded {
					return kbSourceStatusSucceeded, nil, segmentVersionID
				}
				return kbSourceStatusPending, job.Error, segmentVersionID
			}
			if job.JobType == kbJobTypeLoad || job.JobType == kbJobTypeCopy {
				hasTableLoad = true
				if job.JobStatus != kbSourceJobSucceeded {
					tableLoadSucceeded = false
				}
			}
		}
		if hasTableLoad {
			if tableLoadSucceeded {
				return kbSourceStatusSucceeded, nil, segmentVersionID
			}
			for _, job := range jobs {
				if job.JobType == kbJobTypeLoad || job.JobType == kbJobTypeCopy {
					return kbSourceStatusPending, job.Error, segmentVersionID
				}
			}
		}
		return kbSourceStatusPending, nil, segmentVersionID
	}
	loadSucceeded := true
	hasLoad := false
	for _, job := range jobs {
		if job.JobType != kbJobTypeLoad && job.JobType != kbJobTypeCopy {
			continue
		}
		hasLoad = true
		if job.JobStatus != kbSourceJobSucceeded {
			loadSucceeded = false
		}
	}
	if hasLoad && !loadSucceeded {
		return kbSourceStatusPending, nil, segmentVersionID
	}
	if hasLoad && sourceType == SemanticModelSourceTypeFile && !hasJobType(jobs, kbJobTypeRAGIngest) {
		return kbSourceStatusSucceeded, nil, segmentVersionID
	}
	for _, job := range jobs {
		if job.JobType != kbJobTypeRAGIngest {
			continue
		}
		if job.JobStatus == kbSourceJobSucceeded {
			return kbSourceStatusSucceeded, nil, segmentVersionID
		}
		return kbSourceStatusPending, job.Error, segmentVersionID
	}
	return kbSourceStatusPending, nil, segmentVersionID
}

func hasJobType(jobs []KnowledgeBaseSourceJobRun, jobType string) bool {
	for _, job := range jobs {
		if job.JobType == jobType {
			return true
		}
	}
	return false
}

func sourceRecordToSemanticModelSource(record KnowledgeBaseSourceRecord) SemanticModelSource {
	var sourceType SemanticModelSourceType
	resourceID := ""
	sourceResourceID := ""
	kbResourceID := ""
	switch record.SourceType {
	case kbSourceTypeLocalFile:
		sourceType = SemanticModelSourceTypeFile
		if record.KBFileID != nil {
			resourceID = *record.KBFileID
			kbResourceID = *record.KBFileID
		}
		if record.SourceFileID != nil {
			sourceResourceID = *record.SourceFileID
		}
	case kbSourceTypeCatalogFile:
		sourceType = SemanticModelSourceTypeFile
		if record.KBFileID != nil {
			resourceID = *record.KBFileID
			kbResourceID = *record.KBFileID
		}
		if record.SourceFileID != nil {
			sourceResourceID = *record.SourceFileID
		}
	case kbSourceTypeCatalogTable:
		sourceType = SemanticModelSourceTypeTable
		if record.KBTableID != nil && *record.KBTableID > 0 {
			resourceID = strconv.FormatInt(*record.KBTableID, 10)
			kbResourceID = resourceID
		}
		if record.SourceTableID != nil && *record.SourceTableID > 0 {
			sourceResourceID = strconv.FormatInt(*record.SourceTableID, 10)
			if resourceID == "" && record.Status == kbSourceStatusSucceeded {
				resourceID = sourceResourceID
			}
		}
	default:
		sourceType = SemanticModelSourceTypeFile
	}
	path := []string{}
	if record.SourcePath != nil && *record.SourcePath != "" {
		_ = json.Unmarshal([]byte(*record.SourcePath), &path)
	}
	var sourcePath *string
	if len(path) > 0 {
		sourcePath = stringPtr(strings.Join(path, "/"))
	}
	status := record.Status
	if sourceType == SemanticModelSourceTypeTable && status == kbSourceStatusSucceeded && !knowledgeBaseTableSourceRecordAssociationComplete(record) {
		status = kbSourceStatusPending
	}
	governance := knowledgeBaseSourceGovernance(record)
	row := SemanticModelSource{
		RowID:            record.SourceID,
		SourceID:         record.SourceID,
		SourceType:       sourceType,
		ModelID:          record.ModelID,
		ResourceID:       resourceID,
		SourceFileID:     record.SourceFileID,
		KBFileID:         record.KBFileID,
		SourceTableID:    record.SourceTableID,
		KBTableID:        record.KBTableID,
		DisplayName:      record.DisplayName,
		Path:             path,
		SourcePath:       sourcePath,
		DBName:           record.DBName,
		TableName:        record.TableName,
		SizeBytes:        record.SizeBytes,
		RowCount:         record.RowCount,
		IngestStatus:     &status,
		Enabled:          record.Enabled,
		ExpiresAt:        record.ExpiresAt,
		Expired:          governance.Expired,
		EffectiveEnabled: governance.EffectiveEnabled,
		ForceEnabled:     record.ForceEnabled,
		Tags:             parseKnowledgeBaseSourceTags(record.Tags),
		SegmentVersionID: record.SegmentVersionID,
		IndexVersion:     record.IndexVersion,
		CreatedBy:        record.CreatedBy,
		UpdatedBy:        record.UpdatedBy,
		UpdatedAt:        record.UpdatedAt,
		Error:            record.Error,
		GovernanceStatus: SemanticModelSourceGovernanceManaged,
	}
	if sourceResourceID != "" {
		row.SourceResourceID = &sourceResourceID
	}
	if kbResourceID != "" {
		row.KBResourceID = &kbResourceID
	}
	return row
}

func semanticModelSourcesFromRecords(records []KnowledgeBaseSourceRecord) []SemanticModelSource {
	sources := make([]SemanticModelSource, 0, len(records))
	for _, record := range records {
		sources = append(sources, sourceRecordToSemanticModelSource(record))
	}
	return sources
}

func knowledgeBaseSourceGovernance(record KnowledgeBaseSourceRecord) knowledgeBaseSourceGovernanceState {
	enabled := true
	if record.Enabled != nil {
		enabled = *record.Enabled
	}
	expired := false
	if record.ExpiresAt != nil && *record.ExpiresAt > 0 {
		expired = time.Now().Unix() > *record.ExpiresAt
	}
	effective := enabled && (!expired || record.ForceEnabled)
	return knowledgeBaseSourceGovernanceState{
		Enabled:          enabled,
		Expired:          expired,
		EffectiveEnabled: effective,
	}
}

func parseKnowledgeBaseSourceTags(raw *string) []string {
	if raw == nil || *raw == "" {
		return nil
	}
	var tags []string
	if err := json.Unmarshal([]byte(*raw), &tags); err != nil {
		return nil
	}
	return tags
}

func marshalKnowledgeBaseSourceTags(tags *[]string) (*string, error) {
	if tags == nil {
		return nil, nil
	}
	raw, err := json.Marshal(*tags)
	if err != nil {
		return nil, fmt.Errorf("marshal knowledge base source tags: %w", err)
	}
	out := string(raw)
	return &out, nil
}
