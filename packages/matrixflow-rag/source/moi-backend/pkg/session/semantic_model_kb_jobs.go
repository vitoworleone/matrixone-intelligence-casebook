package session

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/matrixorigin/matrixflow/moi-backend/pkg/coreclient"
	"github.com/matrixorigin/matrixflow/moi-backend/pkg/ctxutil"
	"github.com/matrixorigin/matrixflow/moi-backend/pkg/i18n"
	"github.com/matrixorigin/matrixflow/moi-backend/pkg/iampep"
	"github.com/matrixorigin/matrixflow/moi-backend/pkg/logger"
	"github.com/matrixorigin/matrixflow/moi-backend/pkg/model"

	moi "github.com/matrixflow/moi-core/go-sdk"
	"github.com/matrixflow/moi-core/model/common"
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"
)

var errKnowledgeBaseSourceJobFailed = errors.New("knowledge base source job failed")

type completedKnowledgeBaseTableJob struct {
	source        KnowledgeBaseSourceRecord
	job           KnowledgeBaseSourceJobRun
	new           bool
	ownerSourceID string
	ownerJobID    string
	ownerJobType  string
}

func (s *semanticModelService) runKnowledgeBaseTableCloneJob(ctx context.Context, source *KnowledgeBaseSourceRecord, tableJob *KnowledgeBaseSourceJobRun, actor string) (*semanticModelTableSource, bool, error) {
	if source == nil {
		return nil, false, fmt.Errorf("knowledge base source is required")
	}
	if tableJob == nil {
		return nil, false, fmt.Errorf("knowledge base source job run is required")
	}
	if source.SourceTableID == nil || *source.SourceTableID <= 0 {
		return nil, false, fmt.Errorf("source_table_id is required")
	}
	if s.dataDomainService == nil {
		return nil, false, fmt.Errorf("catalog data-domain service is not configured")
	}
	claimed, err := s.claimKnowledgeBaseSourceJobRunning(ctx, tableJob.JobID, actor)
	if err != nil {
		return nil, false, fmt.Errorf("claim catalog table clone job: %w", err)
	}
	if !claimed {
		return nil, false, nil
	}
	tableJob.JobStatus = kbSourceJobRunning
	clone, err := s.dataDomainService.CloneTableForKnowledgeBase(ctx, *source.SourceTableID, source.DatabaseID, tableJob.IdempotencyKey)
	if err != nil {
		if markErr := s.markKnowledgeBaseSourceFailed(ctx, source.SourceID, err.Error(), actor); markErr != nil {
			return nil, true, fmt.Errorf("clone catalog table %d: %w; mark knowledge base source %s failed: %v", *source.SourceTableID, err, source.SourceID, markErr)
		}
		if markErr := s.markKnowledgeBaseSourceJobFailed(ctx, tableJob.JobID, err.Error(), actor); markErr != nil {
			return nil, true, fmt.Errorf("clone catalog table %d: %w; mark knowledge base source job %s failed: %v", *source.SourceTableID, err, tableJob.JobID, markErr)
		}
		return nil, true, fmt.Errorf("%w: clone catalog table %d: %v", errKnowledgeBaseSourceJobFailed, *source.SourceTableID, err)
	}
	source.Status = kbSourceStatusSucceeded
	source.KBTableID = clone.TargetID
	source.DBName = stringPtr(clone.TargetDB)
	source.TableName = stringPtr(clone.TargetTable)
	source.DisplayName = stringPtr(clone.SourceTable)
	sourcePath, _ := json.Marshal([]string{clone.SourceDB, clone.SourceTable})
	source.SourcePath = stringPtr(string(sourcePath))
	tableJob.JobStatus = kbSourceJobSucceeded
	tableJob.OperationID = stringPtr(clone.OperationID)
	tableJob.KBTableID = clone.TargetID
	return &semanticModelTableSource{DBName: clone.TargetDB, TableNames: []string{clone.TargetTable}}, true, nil
}

func (s *semanticModelService) runKnowledgeBaseCatalogFileCopyJob(ctx context.Context, c *moi.Client, wsID string, source *KnowledgeBaseSourceRecord, copyJob *KnowledgeBaseSourceJobRun, actor string) (bool, error) {
	if source == nil {
		return false, fmt.Errorf("knowledge base source is required")
	}
	if copyJob == nil {
		return false, fmt.Errorf("knowledge base source job run is required")
	}
	if source.SourceType != kbSourceTypeCatalogFile {
		return false, nil
	}
	fileID := ptrValue(source.KBFileID)
	if fileID == "" {
		fileID = ptrValue(source.SourceFileID)
	}
	if fileID == "" {
		return false, fmt.Errorf("source_file_id is required")
	}
	if source.RawVolumeID <= 0 {
		return false, fmt.Errorf("raw_volume_id is required")
	}
	claimed, err := s.claimKnowledgeBaseSourceJobRunning(ctx, copyJob.JobID, actor)
	if err != nil {
		return false, fmt.Errorf("claim catalog file copy job: %w", err)
	}
	if !claimed {
		return false, nil
	}
	copyJob.JobStatus = kbSourceJobRunning
	// Catalog files remain on the user volume (volume_id is write-time gate +
	// location pointer only). Keep the copy job for reconcile ordering so
	// rag_ingest can run after bind succeeds; do not AddFiles into KB raw.
	finished, err := s.finishKnowledgeBaseFileBindJob(ctx, source, copyJob, fileID, nil, actor)
	if err != nil {
		return true, fmt.Errorf("record catalog file copy job result %s: %w", fileID, err)
	}
	return finished, nil
}

func (s *semanticModelService) runKnowledgeBaseLocalFileLoadJob(ctx context.Context, c *moi.Client, wsID string, source *KnowledgeBaseSourceRecord, loadJob *KnowledgeBaseSourceJobRun, actor string) (bool, error) {
	if source == nil {
		return false, fmt.Errorf("knowledge base source is required")
	}
	if loadJob == nil {
		return false, fmt.Errorf("knowledge base source job run is required")
	}
	if source.SourceType != kbSourceTypeLocalFile {
		return false, nil
	}
	fileID := firstNonEmptySegmentString(ptrValue(source.KBFileID), ptrValue(source.SourceFileID), ptrValue(loadJob.KBFileID), ptrValue(loadJob.SourceFileID))
	if fileID == "" {
		return false, fmt.Errorf("source_file_id is required")
	}
	if source.RawVolumeID <= 0 {
		return false, fmt.Errorf("raw_volume_id is required")
	}
	claimed, err := s.claimKnowledgeBaseSourceJobRunning(ctx, loadJob.JobID, actor)
	if err != nil {
		return false, fmt.Errorf("claim local file load job: %w", err)
	}
	if !claimed {
		return false, nil
	}
	loadJob.JobStatus = kbSourceJobRunning
	if err := addFileToKnowledgeBaseRawVolume(ctx, c, wsID, source.RawVolumeID, fileID); err != nil {
		if isTerminalKnowledgeBaseFileBindError(err) {
			if _, markErr := s.finishKnowledgeBaseFileBindJob(ctx, source, loadJob, fileID, err, actor); markErr != nil {
				return true, fmt.Errorf("add local file %s to raw volume: %w; mark knowledge base file bind failed: %v", fileID, err, markErr)
			}
			return true, fmt.Errorf("%w: add local file %s to raw volume: %v", errKnowledgeBaseSourceJobFailed, fileID, err)
		}
		if releaseErr := s.releaseKnowledgeBaseFileBindJobClaims(ctx, actor, loadJob.JobID); releaseErr != nil {
			return true, fmt.Errorf("add local file %s to raw volume: %w; release file bind job: %v", fileID, err, releaseErr)
		}
		return true, fmt.Errorf("add local file %s to raw volume: %w", fileID, err)
	}
	finished, err := s.finishKnowledgeBaseFileBindJob(ctx, source, loadJob, fileID, nil, actor)
	if err != nil {
		return true, fmt.Errorf("record local file load job result %s: %w", fileID, err)
	}
	return finished, nil
}

func isTerminalKnowledgeBaseFileBindError(err error) bool {
	return moi.IsCode(err, common.ErrorCode_INVALID_ARGUMENT) ||
		moi.IsCode(err, common.ErrorCode_NOT_FOUND) ||
		moi.IsCode(err, common.ErrorCode_FILE_NOT_FOUND)
}

type knowledgeBaseFileBindJob struct {
	source KnowledgeBaseSourceRecord
	job    KnowledgeBaseSourceJobRun
	fileID string
}

// runKnowledgeBaseFastFileBindJobs advances pending copy/load jobs that bind a
// file to a KB source. Catalog copy jobs only finish job/source state (no
// AddFiles; file stays on the user volume). Local load jobs batch AddFiles into
// the KB raw volume. Terminal batch errors fall back to per-file isolation;
// transient errors release claims for the next reconcile.
func (s *semanticModelService) runKnowledgeBaseFastFileBindJobs(ctx context.Context, c *moi.Client, wsID string, recordsBySourceID map[string]KnowledgeBaseSourceRecord, jobs []KnowledgeBaseSourceJobRun, actor string) error {
	if c == nil {
		return fmt.Errorf("moi client is required")
	}
	type volumeBatchKey struct {
		volumeID        int64
		requireUnlinked bool
	}
	type volumeBatch struct {
		key   volumeBatchKey
		items []knowledgeBaseFileBindJob
	}
	batches := make(map[volumeBatchKey]*volumeBatch)
	batchOrder := make([]volumeBatchKey, 0)
	catalogBinds := make([]knowledgeBaseFileBindJob, 0)
	for _, job := range jobs {
		source, ok := recordsBySourceID[job.SourceID]
		if !ok {
			return fmt.Errorf("knowledge base source %s not found for job %s", job.SourceID, job.JobID)
		}
		fileID := firstNonEmptySegmentString(ptrValue(source.KBFileID), ptrValue(source.SourceFileID), ptrValue(job.KBFileID), ptrValue(job.SourceFileID))
		if fileID == "" || source.RawVolumeID <= 0 {
			return fmt.Errorf("source %s has incomplete raw file binding", source.SourceID)
		}
		if !((source.SourceType == kbSourceTypeCatalogFile && job.JobType == kbJobTypeCopy) ||
			(source.SourceType == kbSourceTypeLocalFile && job.JobType == kbJobTypeLoad)) {
			return fmt.Errorf("source %s job %s: unsupported fast file bind combination source_type=%s job_type=%s", source.SourceID, job.JobID, source.SourceType, job.JobType)
		}
		item := knowledgeBaseFileBindJob{source: source, job: job, fileID: fileID}
		if source.SourceType == kbSourceTypeCatalogFile {
			// Catalog: finish copy job only; file stays on user volume (no KB raw AddFiles).
			catalogBinds = append(catalogBinds, item)
			continue
		}
		key := volumeBatchKey{volumeID: source.RawVolumeID, requireUnlinked: true}
		batch, ok := batches[key]
		if !ok {
			batch = &volumeBatch{key: key}
			batches[key] = batch
			batchOrder = append(batchOrder, key)
		}
		batch.items = append(batch.items, item)
	}

	for i := range catalogBinds {
		item := &catalogBinds[i]
		ok, err := s.claimKnowledgeBaseSourceJobRunning(ctx, item.job.JobID, actor)
		if err != nil {
			return fmt.Errorf("claim file bind job %s: %w", item.job.JobID, err)
		}
		if !ok {
			continue
		}
		item.job.JobStatus = kbSourceJobRunning
		finished, err := s.finishKnowledgeBaseFileBindJob(ctx, &item.source, &item.job, item.fileID, nil, actor)
		if err != nil {
			return fmt.Errorf("finish knowledge base source %s job %s after catalog file copy bind: %w", item.source.SourceID, item.job.JobID, err)
		}
		if finished {
			recordsBySourceID[item.source.SourceID] = item.source
		}
	}

	var retryErr error
	for _, key := range batchOrder {
		batch := batches[key]
		claimedItems := make([]knowledgeBaseFileBindJob, 0, len(batch.items))
		for _, item := range batch.items {
			ok, err := s.claimKnowledgeBaseSourceJobRunning(ctx, item.job.JobID, actor)
			if err != nil {
				return fmt.Errorf("claim file bind job %s: %w", item.job.JobID, err)
			}
			if !ok {
				continue
			}
			item.job.JobStatus = kbSourceJobRunning
			claimedItems = append(claimedItems, item)
		}
		if len(claimedItems) == 0 {
			continue
		}
		opts := []moi.AddFilesOption{}
		if batch.key.requireUnlinked {
			opts = append(opts, moi.WithRequireUnlinked())
		}
		addFiles := func(fileIDs []string) error {
			return c.VolumeFiles().AddFiles(ctx, wsID, batch.key.volumeID, fileIDs, opts...)
		}
		finish := func(item *knowledgeBaseFileBindJob, cause error) error {
			finished, err := s.finishKnowledgeBaseFileBindJob(ctx, &item.source, &item.job, item.fileID, cause, actor)
			if err != nil {
				return err
			}
			if finished {
				recordsBySourceID[item.source.SourceID] = item.source
			}
			return nil
		}
		fileIDs := make([]string, 0, len(claimedItems))
		for _, item := range claimedItems {
			fileIDs = append(fileIDs, item.fileID)
		}
		if err := addFiles(compactUniqueStrings(fileIDs)); err != nil {
			if !isTerminalKnowledgeBaseFileBindError(err) {
				jobIDs := make([]string, 0, len(claimedItems))
				for _, item := range claimedItems {
					jobIDs = append(jobIDs, item.job.JobID)
				}
				if releaseErr := s.releaseKnowledgeBaseFileBindJobClaims(ctx, actor, jobIDs...); releaseErr != nil {
					return fmt.Errorf("add files to raw volume %d: %w; release file bind jobs: %v", batch.key.volumeID, err, releaseErr)
				}
				if retryErr == nil {
					retryErr = fmt.Errorf("add files to raw volume %d: %w", batch.key.volumeID, err)
				}
				continue
			}
			for i := range claimedItems {
				item := &claimedItems[i]
				bindErr := addFiles([]string{item.fileID})
				if bindErr != nil && !isTerminalKnowledgeBaseFileBindError(bindErr) {
					if releaseErr := s.releaseKnowledgeBaseFileBindJobClaims(ctx, actor, item.job.JobID); releaseErr != nil {
						return fmt.Errorf("add file %s to raw volume %d: %w; release file bind job: %v", item.fileID, batch.key.volumeID, bindErr, releaseErr)
					}
					if retryErr == nil {
						retryErr = fmt.Errorf("add file %s to raw volume %d: %w", item.fileID, batch.key.volumeID, bindErr)
					}
					continue
				}
				if err := finish(item, bindErr); err != nil {
					return fmt.Errorf("finish knowledge base source %s job %s after file bind: %w", item.source.SourceID, item.job.JobID, err)
				}
			}
			continue
		}
		for i := range claimedItems {
			item := &claimedItems[i]
			if err := finish(item, nil); err != nil {
				return fmt.Errorf("finish knowledge base source %s job %s after file bind: %w", item.source.SourceID, item.job.JobID, err)
			}
		}
	}
	return retryErr
}

func newKnowledgeBaseTriggerRAGJob(source *KnowledgeBaseSourceRecord, operationID *string, actor string) KnowledgeBaseSourceJobRun {
	job := newKnowledgeBaseJobRun(source, kbJobTypeRAGIngest, stableID("kb-job", source.SourceID, kbJobTypeRAGIngest), actor)
	job.JobStatus = kbSourceJobPending
	job.OperationID = operationID
	job.SourceFileID = source.SourceFileID
	job.KBFileID = source.KBFileID
	return job
}

func newKnowledgeBaseJobRun(source *KnowledgeBaseSourceRecord, jobType, jobID string, _ string) KnowledgeBaseSourceJobRun {
	return KnowledgeBaseSourceJobRun{
		JobID:          jobID,
		SourceID:       source.SourceID,
		ModelID:        source.ModelID,
		JobType:        jobType,
		JobStatus:      kbSourceJobQueued,
		IdempotencyKey: stableID("kb-job-key", source.SourceID, jobType),
	}
}

func (s *semanticModelService) ListSourceJobs(ctx context.Context, params ListSemanticModelSourceJobsParams) (_ *ListSemanticModelSourceJobsResult, retErr error) {
	wsID, err := callerWorkspaceID(ctx)
	if err != nil {
		return nil, err
	}
	var response *ListSemanticModelSourceJobsResult
	err = coreclient.Execute(ctx, coreclient.FromContext(ctx), func(callCtx context.Context, client *moi.Client) error {
		var callErr error
		response, callErr = s.listSourceJobs(callCtx, client, wsID, params)
		return callErr
	})
	return response, err
}

func (s *semanticModelService) listSourceJobs(ctx context.Context, c *moi.Client, wsID string, params ListSemanticModelSourceJobsParams) (_ *ListSemanticModelSourceJobsResult, retErr error) {
	startedAt := time.Now()
	stage := "validation"
	var modelDuration, candidateDuration, jobQueryDuration, sourceQueryDuration, jobEnrichDuration, legacyDuration time.Duration
	var sourceCount, jobCount int
	var enrichStats knowledgeBaseSourceJobEnrichStats
	defer func() {
		outcome := "success"
		errorStage := ""
		if retErr != nil {
			outcome = "error"
			errorStage = stage
		}
		logger.Info("knowledge base source jobs list completed",
			"model_id", int64(params.ModelID),
			"outcome", outcome,
			"error_stage", errorStage,
			"source_count", sourceCount,
			"job_count", jobCount,
			"model_ms", modelDuration.Milliseconds(),
			"candidate_query_ms", candidateDuration.Milliseconds(),
			"job_query_ms", jobQueryDuration.Milliseconds(),
			"source_query_ms", sourceQueryDuration.Milliseconds(),
			"job_enrich_ms", jobEnrichDuration.Milliseconds(),
			"legacy_ms", legacyDuration.Milliseconds(),
			"import_task_calls", enrichStats.importTaskCalls.Load(),
			"import_task_ms", time.Duration(enrichStats.importTaskNanos.Load()).Milliseconds(),
			"file_execution_calls", enrichStats.fileExecutionCalls.Load(),
			"file_execution_ms", time.Duration(enrichStats.fileExecutionNanos.Load()).Milliseconds(),
			"workflow_execution_calls", enrichStats.workflowExecutionCalls.Load(),
			"workflow_execution_ms", time.Duration(enrichStats.workflowExecutionNanos.Load()).Milliseconds(),
			"total_ms", time.Since(startedAt).Milliseconds())
	}()
	modelID := int64(params.ModelID)
	if modelID == 0 {
		return nil, semanticModelNotFoundError()
	}
	stage = "model_query"
	stageStartedAt := time.Now()
	model, err := c.SemanticModels(wsID).Get(ctx, modelID)
	modelDuration = time.Since(stageStartedAt)
	if err != nil {
		return nil, err
	}
	stage = "candidate_query"
	stageStartedAt = time.Now()
	sourceIDs, total, err := s.listKnowledgeBaseSourceJobCandidateIDs(ctx, modelID, kbSourceJobListBatchSize)
	candidateDuration = time.Since(stageStartedAt)
	if err != nil {
		return nil, err
	}
	items := []KnowledgeBaseSourceJobView{}
	if len(sourceIDs) > 0 {
		sourceCount = len(sourceIDs)
		stage = "job_query"
		stageStartedAt = time.Now()
		jobs, err := s.listKnowledgeBaseSourceJobRunsForSourceIDs(ctx, modelID, sourceIDs)
		jobQueryDuration = time.Since(stageStartedAt)
		if err != nil {
			return nil, err
		}
		stage = "source_query"
		stageStartedAt = time.Now()
		records, err := s.listKnowledgeBaseSourceRowsByIDs(ctx, modelID, sourceIDs)
		sourceQueryDuration = time.Since(stageStartedAt)
		if err != nil {
			return nil, err
		}
		jobCount = len(jobs)
		stage = "job_enrich"
		stageStartedAt = time.Now()
		jobs, err = s.enrichKnowledgeBaseSourceJobRunsFromLinkedJobsWithStats(ctx, c, wsID, jobs, &enrichStats)
		jobEnrichDuration = time.Since(stageStartedAt)
		if err != nil {
			return nil, err
		}
		items = knowledgeBaseSourceJobViews(jobs, knowledgeBaseSourceRecordsByID(records))
	}
	reconcileRequired := total > 0
	if !reconcileRequired {
		stage = "legacy"
		stageStartedAt = time.Now()
		reconcileRequired, err = s.legacySourceJobReconcileRequired(ctx, toSemanticModelInfo(model))
		legacyDuration = time.Since(stageStartedAt)
		if err != nil {
			return nil, err
		}
	}
	return &ListSemanticModelSourceJobsResult{Items: items, Total: total, ReconcileRequired: reconcileRequired}, nil
}

const knowledgeBaseSourceJobCandidateFromWhere = `
	FROM knowledge_base_source_job_runs jr
	INNER JOIN knowledge_base_sources kbs ON kbs.model_id = jr.model_id AND kbs.source_id = jr.source_id
	WHERE jr.model_id = ?
	AND kbs.status <> 'removed'
	AND (kbs.status <> 'failed' OR (jr.job_type = 'rag_ingest' AND jr.job_status = 'failed'))
	AND (
			jr.job_status IN ('pending', 'queued', 'running')
				OR (jr.job_status = 'failed' AND (kbs.status <> 'failed' OR jr.job_type = 'rag_ingest'))
				OR (jr.job_status = 'succeeded' AND (
					kbs.status <> 'succeeded'
					OR (kbs.source_type = 'catalog_table' AND (kbs.kb_table_id IS NULL OR kbs.kb_table_id <= 0 OR kbs.db_name IS NULL OR kbs.db_name = '' OR kbs.table_name IS NULL OR kbs.table_name = ''))
					OR (kbs.source_type IN ('catalog_file', 'local_file') AND (
						kbs.kb_file_id IS NULL OR kbs.kb_file_id = ''
						OR (jr.job_type = 'rag_ingest' AND (kbs.segment_version_id IS NULL OR kbs.segment_version_id = '' OR kbs.index_version IS NULL OR kbs.index_version <= 0))
					))
				))
		)`

func (s *semanticModelService) listKnowledgeBaseSourceJobCandidateIDs(ctx context.Context, modelID int64, limit int) ([]string, int, error) {
	db := ctxutil.TenantDBFrom(ctx)
	if db == nil {
		return nil, 0, fmt.Errorf("tenant db is required")
	}
	if limit <= 0 {
		return []string{}, 0, nil
	}
	var total int64
	if err := db.WithContext(ctx).Raw(`SELECT COUNT(DISTINCT kbs.source_id) `+knowledgeBaseSourceJobCandidateFromWhere, modelID).Scan(&total).Error; err != nil {
		return nil, 0, err
	}
	rows, err := db.WithContext(ctx).Raw(`SELECT kbs.source_id
		`+knowledgeBaseSourceJobCandidateFromWhere+`
		GROUP BY kbs.source_id, kbs.updated_at
		ORDER BY
			MIN(CASE
				WHEN jr.job_status IN ('pending', 'queued') THEN 0
				WHEN jr.job_status IN ('succeeded', 'failed') THEN 1
				ELSE 2
			END),
			kbs.updated_at ASC,
			kbs.source_id ASC
		LIMIT ?`, modelID, limit).Rows()
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	sourceIDs := make([]string, 0, limit)
	for rows.Next() {
		var sourceID string
		if err := rows.Scan(&sourceID); err != nil {
			return nil, 0, err
		}
		sourceIDs = append(sourceIDs, sourceID)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return sourceIDs, int(total), nil
}

func (s *semanticModelService) legacySourceJobReconcileRequired(ctx context.Context, model *SemanticModelInfo) (bool, error) {
	if model == nil || model.ID <= 0 {
		return false, nil
	}
	required, err := s.explicitSemanticModelSourceBackfillRequired(ctx, model)
	if err != nil || required {
		return required, err
	}
	required, err = s.legacySourceJobRunExists(ctx, int64(model.ID))
	if err != nil || required {
		return required, err
	}
	required, err = s.rawVolumeLegacySourceExists(ctx, int64(model.ID))
	if err != nil || required {
		return required, err
	}
	return s.lineageLegacySourceExists(ctx, model)
}

func (s *semanticModelService) explicitSemanticModelSourceBackfillRequired(ctx context.Context, model *SemanticModelInfo) (bool, error) {
	db := ctxutil.TenantDBFrom(ctx)
	if db == nil || model == nil || model.ID <= 0 {
		return false, nil
	}
	fileIDs, err := semanticModelFileIDs(model.Files)
	if err != nil {
		return false, err
	}
	if len(fileIDs) > 0 {
		args := make([]any, 0, len(fileIDs)+1)
		for _, fileID := range fileIDs {
			args = append(args, fileID)
		}
		args = append(args, model.ID)
		var missing int
		err := db.WithContext(ctx).Raw(fmt.Sprintf(`SELECT CASE WHEN EXISTS (
			SELECT 1
			FROM (VALUES %s) AS explicit(file_id)
			WHERE NOT EXISTS (
				SELECT 1 FROM knowledge_base_sources kbs
				WHERE kbs.model_id = ? AND (kbs.kb_file_id = explicit.file_id OR kbs.source_file_id = explicit.file_id)
			)
			LIMIT 1
		) THEN 1 ELSE 0 END`, strings.TrimSuffix(strings.Repeat("ROW(?),", len(fileIDs)), ",")), args...).Scan(&missing).Error
		if err != nil || missing == 1 {
			return missing == 1, err
		}
	}
	tables, err := semanticModelTableSources(model.ID, model.Tables)
	if err != nil {
		return false, err
	}
	tableRows := make([]string, 0, len(tables))
	args := make([]any, 0, len(tables)*2+1)
	for _, table := range tables {
		if table.DBName == nil || table.TableName == nil {
			continue
		}
		tableRows = append(tableRows, "ROW(?, ?)")
		args = append(args, *table.DBName, *table.TableName)
	}
	if len(tableRows) > 0 {
		args = append(args, model.ID)
		var missing int
		err := db.WithContext(ctx).Raw(fmt.Sprintf(`SELECT CASE WHEN EXISTS (
			SELECT 1
			FROM (VALUES %s) AS explicit(db_name, table_name)
			WHERE NOT EXISTS (
				SELECT 1 FROM knowledge_base_sources kbs
				WHERE kbs.model_id = ? AND kbs.db_name = explicit.db_name AND kbs.table_name = explicit.table_name
			)
			LIMIT 1
		) THEN 1 ELSE 0 END`, strings.Join(tableRows, ",")), args...).Scan(&missing).Error
		if err != nil || missing == 1 {
			return missing == 1, err
		}
	}
	return false, nil
}

func (s *semanticModelService) legacySourceJobRunExists(ctx context.Context, modelID int64) (bool, error) {
	db := ctxutil.TenantDBFrom(ctx)
	if db == nil {
		return false, nil
	}
	var found int
	err := db.WithContext(ctx).Raw(`SELECT CASE WHEN EXISTS (
		SELECT 1
		FROM knowledge_base_source_jobs legacy
		WHERE legacy.model_id = ?
		  AND COALESCE(NULLIF(legacy.kb_file_id, ''), NULLIF(legacy.source_file_id, '')) IS NOT NULL
		  AND NOT EXISTS (
			SELECT 1 FROM knowledge_base_sources removed
			WHERE removed.model_id = legacy.model_id AND removed.status = 'removed'
			  AND (
				removed.kb_file_id = COALESCE(NULLIF(legacy.kb_file_id, ''), legacy.source_file_id)
				OR removed.source_file_id = legacy.source_file_id
			  )
		  )
		  AND (
			NOT EXISTS (
				SELECT 1 FROM knowledge_base_sources kbs
				WHERE kbs.model_id = legacy.model_id AND kbs.status <> 'removed'
				  AND (
					(NULLIF(legacy.kb_file_id, '') IS NOT NULL AND kbs.kb_file_id = legacy.kb_file_id)
					OR (NULLIF(legacy.kb_file_id, '') IS NULL AND kbs.source_file_id = legacy.source_file_id)
				  )
			)
			OR NOT EXISTS (
				SELECT 1 FROM knowledge_base_source_job_runs jr
				WHERE jr.model_id = legacy.model_id
				  AND COALESCE(NULLIF(jr.kb_file_id, ''), jr.source_file_id) = COALESCE(NULLIF(legacy.kb_file_id, ''), legacy.source_file_id)
				  AND COALESCE(jr.workflow_execution_id, '') = COALESCE(legacy.workflow_execution_id, '')
			)
		  )
		LIMIT 1
	) THEN 1 ELSE 0 END`, modelID).Scan(&found).Error
	return found == 1, err
}

const rawVolumeLegacySourceExistsQueryFormat = `SELECT 1
	FROM volume_files vf
	INNER JOIN volume v ON v.volume_id = vf.volume_id AND v.deleted = FALSE
	INNER JOIN ` + "`file`" + ` f ON f.file_id = vf.file_id
	LEFT JOIN catalog_database cd ON cd.database_id = v.database_id
	LEFT JOIN catalog c ON c.catalog_id = CASE WHEN v.catalog_id > 0 THEN v.catalog_id ELSE cd.catalog_id END
	LEFT JOIN (
		SELECT kb_file_id AS file_id
		FROM knowledge_base_sources
		WHERE model_id = ? AND NULLIF(kb_file_id, '') IS NOT NULL
		UNION ALL
		SELECT source_file_id AS file_id
		FROM knowledge_base_sources
		WHERE model_id = ?
		  AND NULLIF(kb_file_id, '') IS NULL
		  AND NULLIF(source_file_id, '') IS NOT NULL
	) linked ON linked.file_id = vf.file_id
	WHERE vf.volume_id IN (%s)
	  AND COALESCE(CASE WHEN v.catalog_id > 0 THEN v.catalog_id ELSE cd.catalog_id END, 0) > 0
	  AND COALESCE(v.database_id, 0) > 0
	  AND COALESCE(v.volume_name, '') <> ''
	  AND COALESCE(c.catalog_name, '') <> ''
	  AND COALESCE(cd.database_name, '') <> ''
	  AND linked.file_id IS NULL
	LIMIT 1`

func (s *semanticModelService) rawVolumeLegacySourceExists(ctx context.Context, modelID int64) (bool, error) {
	db := ctxutil.TenantDBFrom(ctx)
	if db == nil {
		return false, nil
	}
	volumeIDs, err := knowledgeBaseRawVolumeIDs(ctx, modelID)
	if err != nil || len(volumeIDs) == 0 {
		return false, err
	}
	args := make([]any, 0, len(volumeIDs)+2)
	args = append(args, modelID, modelID)
	for _, volumeID := range volumeIDs {
		args = append(args, volumeID)
	}
	rows, err := db.WithContext(ctx).Raw(
		fmt.Sprintf(rawVolumeLegacySourceExistsQueryFormat, queryPlaceholders(len(volumeIDs))),
		args...,
	).Rows()
	if err != nil {
		return false, err
	}
	defer rows.Close()
	return rows.Next(), rows.Err()
}

func (s *semanticModelService) lineageLegacySourceExists(ctx context.Context, model *SemanticModelInfo) (bool, error) {
	vectorTables, err := semanticModelVectorTables(model.Files)
	if err != nil || len(vectorTables) == 0 {
		return false, err
	}
	db := ctxutil.TenantDBFrom(ctx)
	if db == nil {
		return false, nil
	}
	args := make([]any, 0, len(vectorTables)+1)
	for _, table := range vectorTables {
		args = append(args, table)
	}
	args = append(args, model.ID)
	rows, err := db.WithContext(ctx).Raw(fmt.Sprintf(`SELECT 1
		FROM data_asset vector
		INNER JOIN data_derivation d ON d.target_asset_id = vector.asset_id AND d.kind = 'indexed_from'
		INNER JOIN data_asset root ON root.asset_id = d.root_asset_id AND root.asset_type = 'file'
		LEFT JOIN parsed_manifest pm ON pm.root_asset_id = d.root_asset_id
		WHERE vector.asset_type = 'vector_index'
		  AND vector.asset_ref IN (%s)
		  AND NOT EXISTS (
			SELECT 1 FROM knowledge_base_sources kbs
			WHERE kbs.model_id = ?
			  AND (kbs.kb_file_id = COALESCE(pm.source_file_id, root.asset_ref) OR kbs.source_file_id = COALESCE(pm.source_file_id, root.asset_ref))
		  )
		LIMIT 1`, queryPlaceholders(len(vectorTables))), args...).Rows()
	if err != nil {
		return false, err
	}
	defer rows.Close()
	return rows.Next(), rows.Err()
}

func knowledgeBaseSourceJobViews(jobs []KnowledgeBaseSourceJobRun, sourceIndexes ...map[string]KnowledgeBaseSourceRecord) []KnowledgeBaseSourceJobView {
	sourcesByID := map[string]KnowledgeBaseSourceRecord{}
	if len(sourceIndexes) > 0 && sourceIndexes[0] != nil {
		sourcesByID = sourceIndexes[0]
	}
	type sourceJobGroup struct {
		sourceID string
		jobs     []KnowledgeBaseSourceJobRun
	}
	groups := make([]sourceJobGroup, 0, len(jobs))
	groupBySourceID := make(map[string]int, len(jobs))
	for _, job := range jobs {
		key := job.SourceID
		if key == "" {
			key = job.JobID
		}
		index, ok := groupBySourceID[key]
		if !ok {
			index = len(groups)
			groupBySourceID[key] = index
			groups = append(groups, sourceJobGroup{sourceID: job.SourceID})
		}
		groups[index].jobs = append(groups[index].jobs, job)
	}

	out := make([]KnowledgeBaseSourceJobView, 0, len(groups))
	for _, group := range groups {
		if len(group.jobs) == 0 {
			continue
		}
		source, hasSource := sourcesByID[group.sourceID]
		if !hasSource {
			source = KnowledgeBaseSourceRecord{}
		}
		out = append(out, knowledgeBaseSourceJobView(group.sourceID, group.jobs, source, hasSource))
	}
	return out
}

func knowledgeBaseSourceJobView(sourceID string, jobs []KnowledgeBaseSourceJobRun, source KnowledgeBaseSourceRecord, hasSource bool) KnowledgeBaseSourceJobView {
	selected := selectKnowledgeBaseSourceJobStatus(jobs)
	reconcileRequired := knowledgeBaseSourceJobReconcileRequired(selected, source, hasSource)
	view := KnowledgeBaseSourceJobView{
		JobID:             stableID("kb-source-job", sourceID, "ingest"),
		SourceID:          sourceID,
		JobStatus:         selected.JobStatus,
		Error:             selected.Error,
		ReconcileRequired: reconcileRequired,
	}
	if view.SourceID == "" {
		view.SourceID = selected.SourceID
	}
	if hasJobType(jobs, kbJobTypeTableClone) && !hasJobType(jobs, kbJobTypeRAGIngest) && !hasJobType(jobs, kbJobTypeLoad) && !hasJobType(jobs, kbJobTypeCopy) {
		view.JobID = stableID("kb-source-job", view.SourceID, kbJobTypeTableClone)
	}
	for _, job := range jobs {
		if view.SourceFileID == nil && job.SourceFileID != nil {
			view.SourceFileID = job.SourceFileID
		}
		if view.KBFileID == nil && job.KBFileID != nil {
			view.KBFileID = job.KBFileID
		}
		if view.SourceTableID == nil && job.SourceTableID != nil {
			view.SourceTableID = job.SourceTableID
		}
		if view.KBTableID == nil && job.KBTableID != nil {
			view.KBTableID = job.KBTableID
		}
		if view.UpdatedAt == nil || job.UpdatedAt > *view.UpdatedAt {
			updatedAt := job.UpdatedAt
			view.UpdatedAt = &updatedAt
		}
	}
	return view
}

func knowledgeBaseSourceRecordsByID(records []KnowledgeBaseSourceRecord) map[string]KnowledgeBaseSourceRecord {
	out := make(map[string]KnowledgeBaseSourceRecord, len(records))
	for _, record := range records {
		if record.SourceID == "" {
			continue
		}
		out[record.SourceID] = record
	}
	return out
}

func knowledgeBaseSourceJobReconcileRequired(job KnowledgeBaseSourceJobRun, source KnowledgeBaseSourceRecord, hasSource bool) bool {
	if hasSource && (source.Status == kbSourceStatusFailed || source.Status == kbSourceStatusRemoved) {
		return false
	}
	switch job.JobStatus {
	case kbSourceJobPending, kbSourceJobQueued:
		return true
	case kbSourceJobSucceeded, kbSourceJobFailed:
		return !knowledgeBaseSourceHasFinalBinding(source, hasSource, job)
	default:
		return false
	}
}

func knowledgeBaseSourceHasFinalBinding(source KnowledgeBaseSourceRecord, hasSource bool, job KnowledgeBaseSourceJobRun) bool {
	if !hasSource || source.Status != kbSourceStatusSucceeded {
		return false
	}
	switch source.SourceType {
	case kbSourceTypeCatalogTable:
		return source.KBTableID != nil && *source.KBTableID > 0 &&
			source.DBName != nil && *source.DBName != "" &&
			source.TableName != nil && *source.TableName != ""
	case kbSourceTypeCatalogFile, kbSourceTypeLocalFile:
		if source.KBFileID == nil || *source.KBFileID == "" {
			return false
		}
		if job.JobType == kbJobTypeRAGIngest {
			return source.SegmentVersionID != nil && *source.SegmentVersionID != "" && source.IndexVersion != nil && *source.IndexVersion > 0
		}
		return true
	default:
		return true
	}
}

func selectKnowledgeBaseSourceJobStatus(jobs []KnowledgeBaseSourceJobRun) KnowledgeBaseSourceJobRun {
	if len(jobs) == 0 {
		return KnowledgeBaseSourceJobRun{}
	}
	for _, job := range jobs {
		if job.JobStatus == kbSourceJobFailed {
			return job
		}
	}
	for _, job := range jobs {
		if (job.JobType == kbJobTypeLoad || job.JobType == kbJobTypeCopy || job.JobType == kbJobTypeTableClone) && job.JobStatus != kbSourceJobSucceeded {
			return job
		}
	}
	for _, job := range jobs {
		if job.JobType == kbJobTypeRAGIngest {
			return job
		}
	}
	for _, job := range jobs {
		if job.JobType == kbJobTypeLoad || job.JobType == kbJobTypeCopy || job.JobType == kbJobTypeTableClone {
			return job
		}
	}
	return jobs[0]
}

type knowledgeBaseSourceJobEnrichStats struct {
	importTaskCalls        atomic.Int64
	importTaskNanos        atomic.Int64
	fileExecutionCalls     atomic.Int64
	fileExecutionNanos     atomic.Int64
	workflowExecutionCalls atomic.Int64
	workflowExecutionNanos atomic.Int64
}

func (s *semanticModelService) enrichKnowledgeBaseSourceJobRunsFromLinkedJobs(ctx context.Context, client *moi.Client, wsID string, jobs []KnowledgeBaseSourceJobRun) ([]KnowledgeBaseSourceJobRun, error) {
	return s.enrichKnowledgeBaseSourceJobRunsFromLinkedJobsWithStats(ctx, client, wsID, jobs, nil)
}

func (s *semanticModelService) enrichKnowledgeBaseSourceJobRunsFromLinkedJobsWithStats(ctx context.Context, client *moi.Client, wsID string, jobs []KnowledgeBaseSourceJobRun, stats *knowledgeBaseSourceJobEnrichStats) ([]KnowledgeBaseSourceJobRun, error) {
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(kbSourceListEnrichConcurrency)
	for i := range jobs {
		i := i
		g.Go(func() error {
			if err := gctx.Err(); err != nil {
				return err
			}
			return s.enrichKnowledgeBaseSourceJobRunFromLinkedJob(gctx, client, wsID, &jobs[i], stats)
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	return jobs, nil
}

func (s *semanticModelService) enrichKnowledgeBaseSourceJobRunFromLinkedJob(ctx context.Context, client *moi.Client, wsID string, job *KnowledgeBaseSourceJobRun, stats *knowledgeBaseSourceJobEnrichStats) error {
	if job == nil || isLineageRegistrationSourceJob(job.OperationID) {
		return nil
	}
	if job.JobType != kbJobTypeRAGIngest {
		return nil
	}
	if taskID, ok := importTaskIDFromOperation(job.OperationID); ok {
		startedAt := time.Now()
		if stats != nil {
			stats.importTaskCalls.Add(1)
			defer func() { stats.importTaskNanos.Add(int64(time.Since(startedAt))) }()
		}
		taskState, err := s.getKnowledgeBaseImportTaskState(ctx, taskID)
		if err != nil {
			return err
		}
		job.JobStatus = taskState.Status
		job.Error = taskState.Error
		if job.WorkflowExecutionID == nil || *job.WorkflowExecutionID == "" {
			job.WorkflowExecutionID = taskState.WorkflowExecutionID
		}
		return nil
	}
	if job.JobType == kbJobTypeRAGIngest && job.KBFileID != nil && *job.KBFileID != "" && s.workflowService != nil {
		startedAt := time.Now()
		if stats != nil {
			stats.fileExecutionCalls.Add(1)
		}
		refreshed, err := s.refreshRAGSourceJobFromFileExecutions(ctx, job)
		if stats != nil {
			stats.fileExecutionNanos.Add(int64(time.Since(startedAt)))
		}
		if err != nil {
			return err
		}
		if refreshed || job.WorkflowExecutionID != nil && strings.TrimSpace(*job.WorkflowExecutionID) != "" {
			return nil
		}
	}
	if job.WorkflowExecutionID == nil || *job.WorkflowExecutionID == "" {
		return nil
	}
	if client == nil {
		return fmt.Errorf("moi-core client is required for workflow execution %s", *job.WorkflowExecutionID)
	}
	startedAt := time.Now()
	if stats != nil {
		stats.workflowExecutionCalls.Add(1)
		defer func() { stats.workflowExecutionNanos.Add(int64(time.Since(startedAt))) }()
	}
	exec, err := client.WorkflowApps(wsID).GetExecution(ctx, "", *job.WorkflowExecutionID)
	if err != nil {
		if moi.IsCode(err, common.ErrorCode_NOT_FOUND) {
			msg := fmt.Sprintf("workflow execution %s not found: %v", *job.WorkflowExecutionID, err)
			job.JobStatus = kbSourceJobFailed
			job.Error = &msg
			return nil
		}
		return fmt.Errorf("get workflow execution %s for source job %s: %w", *job.WorkflowExecutionID, job.JobID, err)
	}
	status, errMsg := workflowExecutionToKnowledgeBaseJobStatus(exec)
	job.JobStatus = status
	if errMsg != nil {
		job.Error = errMsg
	}
	return nil
}

func (s *semanticModelService) refreshRAGSourceJobFromFileExecutions(ctx context.Context, job *KnowledgeBaseSourceJobRun) (bool, error) {
	if s.workflowService == nil {
		return false, fmt.Errorf("workflow service is required to list file executions for source job %s", job.JobID)
	}
	resp, err := s.workflowService.ListFileExecutions(ctx, *job.KBFileID, job.ModelID)
	if err != nil {
		return false, fmt.Errorf("list workflow executions for source job %s file %s: %w", job.JobID, *job.KBFileID, err)
	}
	if resp == nil {
		return false, fmt.Errorf("list workflow executions for source job %s file %s returned empty response", job.JobID, *job.KBFileID)
	}
	var selected *moi.FileExecutionSummary
	for i := range resp.Executions {
		item := resp.Executions[i]
		if selected == nil || fileExecutionSummaryTimestamp(item).After(fileExecutionSummaryTimestamp(*selected)) {
			selected = &resp.Executions[i]
		}
	}
	if selected == nil {
		return false, nil
	}
	job.WorkflowExecutionID = stringPtr(selected.ExecutionID)
	status, errMsg := fileExecutionSummaryToKnowledgeBaseJobStatus(*selected)
	job.JobStatus = status
	if errMsg != nil {
		job.Error = errMsg
	} else {
		job.Error = nil
	}
	return true, nil
}

func fileExecutionSummaryTimestamp(item moi.FileExecutionSummary) time.Time {
	for _, value := range []string{item.UpdatedAt, item.EndedAt, item.StartedAt, item.CreatedAt} {
		if value == "" {
			continue
		}
		for _, layout := range []string{time.RFC3339, "2006-01-02 15:04:05"} {
			parsed, err := time.Parse(layout, value)
			if err == nil {
				return parsed
			}
		}
	}
	return time.Time{}
}

func (s *semanticModelService) reconcileRAGIngestSourceJob(ctx context.Context, c *moi.Client, wsID string, record KnowledgeBaseSourceRecord, job *KnowledgeBaseSourceJobRun, actor string) error {
	if job == nil {
		return fmt.Errorf("knowledge base source job run is required")
	}
	if job.KBFileID == nil || *job.KBFileID == "" {
		return nil
	}
	if isDeferredCatalogFileRAGJob(record, job) {
		return s.reconcileDeferredCatalogFileRAGJob(ctx, c, wsID, record, job, actor)
	}
	if job.JobStatus == kbSourceJobSucceeded && isLineageRegistrationSourceJob(job.OperationID) {
		published, err := s.publishExternalWorkflowSegmentVersion(ctx, c, wsID, record, *job.KBFileID, job, actor)
		if err != nil {
			if markErr := s.markKnowledgeBaseSourceFailed(ctx, record.SourceID, err.Error(), actor); markErr != nil {
				return fmt.Errorf("publish lineage segment version for source %s: %w; mark knowledge base source failed: %v", record.SourceID, err, markErr)
			}
			if markErr := s.markKnowledgeBaseSourceJobFailed(ctx, job.JobID, err.Error(), actor); markErr != nil {
				return fmt.Errorf("publish lineage segment version for source %s: %w; mark knowledge base source job %s failed: %v", record.SourceID, err, job.JobID, markErr)
			}
			return fmt.Errorf("%w: publish lineage segment version for source %s: %v", errKnowledgeBaseSourceJobFailed, record.SourceID, err)
		}
		if !published {
			if err := s.updateKnowledgeBaseSourceJobRunSucceeded(ctx, job, actor); err != nil {
				return fmt.Errorf("update lineage rag ingest job: %w", err)
			}
		}
		return nil
	}
	previousStatus := job.JobStatus
	previousExecutionID := strings.TrimSpace(ptrValue(job.WorkflowExecutionID))
	refreshed, err := s.refreshRAGSourceJobFromFileExecutions(ctx, job)
	if err != nil {
		return err
	}
	if !refreshed {
		if err := s.triggerDeferredRAGIngestSourceJob(ctx, c, wsID, record, job, actor); err != nil {
			return err
		}
		return nil
	}
	if previousStatus == kbSourceJobFailed && job.JobStatus == kbSourceJobFailed && strings.TrimSpace(ptrValue(job.WorkflowExecutionID)) == previousExecutionID {
		return s.markKnowledgeBaseSourceJobFailedChecked(ctx, job.JobID, actor)
	}
	if previousStatus == kbSourceJobFailed && strings.TrimSpace(ptrValue(job.WorkflowExecutionID)) != previousExecutionID {
		job.Error = nil
		if err := s.markKnowledgeBaseSourcePending(ctx, record.SourceID, actor); err != nil {
			return fmt.Errorf("restore retried rag source: %w", err)
		}
		if err := s.upsertKnowledgeBaseSourceJobRun(ctx, job, actor); err != nil {
			return fmt.Errorf("record retried rag workflow execution: %w", err)
		}
	}
	switch job.JobStatus {
	case kbSourceJobSucceeded:
		published, err := s.publishExternalWorkflowSegmentVersion(ctx, c, wsID, record, *job.KBFileID, job, actor)
		if err != nil {
			if isKnowledgeBaseRAGIndexNotReadyError(err) || errors.Is(err, errExternalWorkflowVectorBindingMismatch) {
				return s.triggerRAGIngestForUnusableCompletedExecution(ctx, c, wsID, record, job, actor)
			}
			if strings.Contains(err.Error(), "update rag ingest job") {
				return fmt.Errorf("publish workflow segment version for source %s: %w", record.SourceID, err)
			}
			if markErr := s.markKnowledgeBaseSourceFailed(ctx, record.SourceID, err.Error(), actor); markErr != nil {
				return fmt.Errorf("publish workflow segment version for source %s: %w; mark knowledge base source failed: %v", record.SourceID, err, markErr)
			}
			if markErr := s.markKnowledgeBaseSourceJobFailed(ctx, job.JobID, err.Error(), actor); markErr != nil {
				return fmt.Errorf("publish workflow segment version for source %s: %w; mark knowledge base source job %s failed: %v", record.SourceID, err, job.JobID, markErr)
			}
			return fmt.Errorf("%w: publish workflow segment version for source %s: %v", errKnowledgeBaseSourceJobFailed, record.SourceID, err)
		}
		if !published {
			if err := s.updateKnowledgeBaseSourceJobRunSucceeded(ctx, job, actor); err != nil {
				return fmt.Errorf("update rag ingest job: %w", err)
			}
		}
	case kbSourceJobFailed:
		errMsg := "workflow ingest failed"
		if job.Error != nil && *job.Error != "" {
			errMsg = *job.Error
		}
		if err := s.markKnowledgeBaseSourceFailed(ctx, record.SourceID, errMsg, actor); err != nil {
			return err
		}
		if err := s.markKnowledgeBaseSourceJobFailed(ctx, job.JobID, errMsg, actor); err != nil {
			return err
		}
		return fmt.Errorf("%w: workflow ingest failed for source %s: %s", errKnowledgeBaseSourceJobFailed, record.SourceID, errMsg)
	case kbSourceJobRunning:
		if err := s.markKnowledgeBaseSourceJobRunning(ctx, job.JobID, actor); err != nil {
			return err
		}
	}
	return nil
}

func isLineageRegistrationSourceJob(operationID *string) bool {
	return operationID != nil && strings.HasPrefix(*operationID, "lineage_register:")
}

func (s *semanticModelService) triggerRAGIngestForUnusableCompletedExecution(ctx context.Context, c *moi.Client, wsID string, record KnowledgeBaseSourceRecord, job *KnowledgeBaseSourceJobRun, actor string) error {
	job.JobStatus = kbSourceJobQueued
	job.WorkflowExecutionID = nil
	return s.triggerDeferredRAGIngestSourceJob(ctx, c, wsID, record, job, actor)
}

func isKnowledgeBaseRAGIndexNotReadyError(err error) bool {
	if err == nil {
		return false
	}
	if isVectorTableUnavailableError(err) {
		return true
	}
	var svcErr *ServiceError
	if !errors.As(err, &svcErr) || svcErr.Err == nil {
		return false
	}
	errText := svcErr.Err.Error()
	return strings.Contains(errText, i18n.KeySessionSegmentRowsUnavailable.String()) ||
		strings.Contains(errText, i18n.KeySessionWorkflowSegmentRowsUnavailable.String())
}

func (s *semanticModelService) triggerDeferredRAGIngestSourceJob(ctx context.Context, c *moi.Client, wsID string, record KnowledgeBaseSourceRecord, job *KnowledgeBaseSourceJobRun, actor string) error {
	if !shouldTriggerDeferredRAGIngestSourceJob(job) {
		return nil
	}
	ready := false
	var err error
	switch record.SourceType {
	case kbSourceTypeLocalFile:
		ready, err = s.localFileLoadCompletedForRAG(ctx, job.ModelID, job.SourceID)
	case kbSourceTypeCatalogFile:
		ready, err = s.catalogFileCopyCompletedForRAG(ctx, job.ModelID, job.SourceID)
	default:
		return nil
	}
	if err != nil {
		return err
	}
	if !ready {
		return nil
	}
	return s.claimAndTriggerDeferredRAGIngestSourceJob(ctx, c, wsID, record, job, actor)
}

func shouldTriggerDeferredRAGIngestSourceJob(job *KnowledgeBaseSourceJobRun) bool {
	if job == nil || job.JobType != kbJobTypeRAGIngest {
		return false
	}
	if job.JobStatus != kbSourceJobPending && job.JobStatus != kbSourceJobQueued {
		return false
	}
	if job.WorkflowExecutionID != nil && *job.WorkflowExecutionID != "" {
		return false
	}
	if job.OperationID == nil || *job.OperationID == "" {
		return true
	}
	return strings.HasPrefix(*job.OperationID, "workflow_trigger:")
}

// rehydrateKnowledgeBaseRAGJobPrincipal restores the job-frozen MOI user and
// VerifiedEffectiveRole into ctx so deferred dispatch can re-enter Core under
// that principal. It does not restore privilege-class bypass.
//
// Durable freeze keeps actor + role for audit and as the RoleCandidate for
// ReauthorizeAction. runtime_is_workspace_owner may still be stored at create
// time for audit, but must not reconstruct IsWorkspaceOwner /
// BusinessActionAuthorized: ReauthorizeAction would then skip Core and retain
// create-time owner access after binding/lifecycle/policy changes.
//
// Product identity attach goes through coreclient.WithIdentity (reviewed
// boundary). Do not call ctxutil.WithMoiUserID from pkg/session.
// Deferred dispatch must not inherit a callback/system principal; missing
// actor/role fails closed.
func rehydrateKnowledgeBaseRAGJobPrincipal(ctx context.Context, job *KnowledgeBaseSourceJobRun) (context.Context, error) {
	if job == nil {
		return ctx, fmt.Errorf("knowledge base source job run is required")
	}
	actorID := strings.TrimSpace(ptrValue(job.RuntimeActorMOIUserID))
	roleID := strings.TrimSpace(ptrValue(job.RuntimeEffectiveRoleID))
	if actorID == "" || roleID == "" {
		return ctx, fmt.Errorf("rag ingest job missing frozen runtime principal")
	}
	user, ok := coreclient.FromUserID(actorID)
	if !ok {
		return ctx, fmt.Errorf("rag ingest job frozen actor is not a valid catalog user id")
	}
	ctx = coreclient.WithIdentity(ctx, user)
	trusted, _ := ctxutil.CoreIAMRequestFrom(ctx)
	// Clear prior allow facts and privilege bypass so reauth always re-enters
	// Core under the frozen role (binding/lifecycle/policy checked live).
	// job.RuntimeIsWorkspaceOwner is intentionally ignored here.
	trusted.VerifiedEffectiveRoleID = roleID
	trusted.RoleCandidateID = ""
	trusted.BusinessActionAuthorized = false
	trusted.AuthorizedActionFacts = nil
	trusted.IsWorkspaceOwner = false
	trusted.WorkspaceAccessVerified = false
	return ctxutil.WithCoreIAMRequest(ctx, trusted), nil
}

// isKnowledgeBaseRAGAuthRetryableError reports transient Core / resolve-root
// failures that should leave the job pending for a later reconcile.
// Permanent: deny, NOT_FOUND, invalid root, missing wiring.
// Retryable: IAM Core transport/decision unavailable, ResolveRoot
// UNAVAILABLE/TIMEOUT, and network/deadline failures from the resolver hop.
func isKnowledgeBaseRAGAuthRetryableError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, iampep.ErrCoreUnavailable) ||
		errors.Is(err, iampep.ErrCoreDecisionUnknown) ||
		errors.Is(err, iampep.ErrCoreDecisionError) {
		return true
	}
	// ResolveCanonicalRootVolume / ResolveRoot surface SDK codes, not iampep errors.
	if moi.IsCode(err, common.ErrorCode_UNAVAILABLE) ||
		moi.IsCode(err, common.ErrorCode_TIMEOUT) {
		return true
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return true
	}
	var netErr net.Error
	return errors.As(err, &netErr)
}

// reauthorizeSemanticModelUse re-checks semantic_model.use under the rehydrated
// job principal. The returned allow context is action-scoped and may be used for
// built-in KB workflow / vector reuse hops that sit under semantic_model.use.
// It must not be reused for volume.read.
func (s *semanticModelService) reauthorizeSemanticModelUse(ctx context.Context, wsID string, modelID int64) (context.Context, error) {
	if modelID <= 0 {
		return ctx, fmt.Errorf("semantic model id is required for semantic_model.use reauthorization")
	}
	if s == nil || s.actionAuthorizer == nil {
		return ctx, fmt.Errorf("semantic_model.use reauthorization is unavailable")
	}
	wsID = strings.TrimSpace(wsID)
	if wsID == "" {
		return ctx, fmt.Errorf("workspace is required for semantic_model.use reauthorization")
	}
	authorizedCtx, err := s.actionAuthorizer.ReauthorizeAction(
		ctx,
		wsID,
		"semantic_model.use",
		"semantic_model",
		strconv.FormatInt(modelID, 10),
	)
	if err != nil {
		return ctx, fmt.Errorf("reauthorize semantic_model.use on model %d: %w", modelID, err)
	}
	return authorizedCtx, nil
}

// checkCatalogFileVolumeRead re-checks volume.read on the canonical root of the
// recorded external source volume before deferred catalog_file RAG dispatch.
// Identity is file_id; RawVolumeID is only the write-time gate / location pointer
// used by Workflow to read the original file. IAM volume.read grants attach to
// the canonical root (same as the sync create/append path), so a child volume
// ID must be resolved first.
//
// The allow context from ReauthorizeAction is intentionally discarded: volume
// allow must not cross into workflow/run hops (action-scoped). Callers keep the
// semantic_model.use allow context for dispatch.
//
// Authorization always re-enters Core via ReauthorizeAction under the rehydrated
// (non-privileged) job principal so current role binding/lifecycle and
// volume.read policy are checked live. Deny / missing resolver fail closed
// permanently; Core/resolve transport unavailability is retryable for the caller.
func (s *semanticModelService) checkCatalogFileVolumeRead(ctx context.Context, wsID string, volumeID int64) error {
	if volumeID <= 0 {
		return fmt.Errorf("catalog file source volume_id is required for volume.read reauthorization")
	}
	if s == nil || s.actionAuthorizer == nil {
		return fmt.Errorf("catalog file volume.read reauthorization is unavailable")
	}
	if s.volumeResolver == nil {
		// Do not fall back to RawVolumeID: IAM grants attach to the canonical root.
		// Missing wiring must fail closed rather than authorize a child volume.
		return fmt.Errorf("catalog file volume.read reauthorization is unavailable: canonical volume resolver is not configured")
	}
	wsID = strings.TrimSpace(wsID)
	if wsID == "" {
		return fmt.Errorf("workspace is required for volume.read reauthorization")
	}
	rootID, err := s.volumeResolver.ResolveCanonicalRootVolume(ctx, wsID, volumeID)
	if err != nil {
		return fmt.Errorf("resolve canonical root volume for volume.read reauthorization: %w", err)
	}
	if rootID <= 0 {
		return fmt.Errorf("catalog file source has no canonical root volume for volume.read reauthorization")
	}
	// Reauth from rehydrated principal facts, not from another action's allow ctx.
	// Caller should pass principalCtx (post-rehydrate), not volume/use allow ctx.
	// Authorize the canonical root; keep RawVolumeID only for workflow I/O.
	if _, err := s.actionAuthorizer.ReauthorizeAction(
		ctx,
		wsID,
		"volume.read",
		"volume",
		strconv.FormatInt(rootID, 10),
	); err != nil {
		return fmt.Errorf("reauthorize volume.read on volume %d: %w", rootID, err)
	}
	return nil
}

// handleKnowledgeBaseRAGAuthError maps reauth failures: Core unavailable stays
// retryable (job pending); deny/other permanent-fail the deferred job.
func (s *semanticModelService) handleKnowledgeBaseRAGAuthError(ctx context.Context, record KnowledgeBaseSourceRecord, job *KnowledgeBaseSourceJobRun, actor string, authErr error, what string) error {
	if isKnowledgeBaseRAGAuthRetryableError(authErr) {
		return fmt.Errorf("%s temporarily unavailable: %w", what, authErr)
	}
	return s.failDeferredRAGIngestSourceJob(ctx, record, job, actor, authErr)
}

func (s *semanticModelService) claimAndTriggerDeferredRAGIngestSourceJob(ctx context.Context, c *moi.Client, wsID string, record KnowledgeBaseSourceRecord, job *KnowledgeBaseSourceJobRun, actor string) error {
	return s.claimAndTriggerDeferredRAGIngestSourceJobWithAuth(ctx, c, wsID, record, job, actor, nil, nil)
}

// claimAndTriggerDeferredRAGIngestSourceJobWithAuth is the dispatch entry.
// When principalCtx/kbCtx are already established (catalog reconcile after a
// failed vector-reuse attempt), pass them to skip a second rehydrate +
// semantic_model.use Core call. Direct callers leave both nil.
//
// Auth order when not pre-authorized:
//  1. rehydrate job-frozen principal
//  2. reauth semantic_model.use -> kbCtx for workflow hops
//  3. catalog_file only: volume.read gate (allow discarded) before claim/run
func (s *semanticModelService) claimAndTriggerDeferredRAGIngestSourceJobWithAuth(
	ctx context.Context,
	c *moi.Client,
	wsID string,
	record KnowledgeBaseSourceRecord,
	job *KnowledgeBaseSourceJobRun,
	actor string,
	principalCtx context.Context,
	kbCtx context.Context,
) error {
	if principalCtx == nil || kbCtx == nil {
		var err error
		principalCtx, err = rehydrateKnowledgeBaseRAGJobPrincipal(ctx, job)
		if err != nil {
			return s.failDeferredRAGIngestSourceJob(ctx, record, job, actor, err)
		}
		var useErr error
		kbCtx, useErr = s.reauthorizeSemanticModelUse(principalCtx, wsID, job.ModelID)
		if useErr != nil {
			return s.handleKnowledgeBaseRAGAuthError(ctx, record, job, actor, useErr, "semantic_model.use reauthorization")
		}
	}
	if record.SourceType == kbSourceTypeCatalogFile {
		// Gate only: do not propagate volume allow into workflow ctx.
		// Always from principalCtx (not use-allow ctx), even when pre-authorized.
		if volErr := s.checkCatalogFileVolumeRead(principalCtx, wsID, record.RawVolumeID); volErr != nil {
			return s.handleKnowledgeBaseRAGAuthError(ctx, record, job, actor, volErr, "catalog file volume.read reauthorization")
		}
	}
	operationID := "workflow_trigger:" + knowledgeBaseWorkflowID(wsID, job.ModelID)
	claimed, err := s.claimDeferredRAGIngestSourceJob(kbCtx, record, job, operationID, actor)
	if err != nil {
		return err
	}
	if !claimed {
		return nil
	}
	templateKey, err := resolveTemplateKeyForKnowledgeBaseFile(kbCtx, c, wsID, &record, job)
	if err != nil {
		return s.failDeferredRAGIngestSourceJob(ctx, record, job, actor, err)
	}
	workflowID := knowledgeBaseWorkflowID(wsID, job.ModelID)
	if templateKey == kbAudioRAGTemplateKey || templateKey == kbVideoRAGTemplateKey {
		workflowID = knowledgeBaseMediaWorkflowID(wsID, job.ModelID, templateKey)
	}
	if templateKey == kbAudioRAGTemplateKey || templateKey == kbVideoRAGTemplateKey {
		domain, ok, err := s.getKnowledgeBaseDataDomain(kbCtx, job.ModelID)
		if err != nil {
			return s.failDeferredRAGIngestSourceJob(ctx, record, job, actor, fmt.Errorf("get knowledge base data domain for media rag ingest: %w", err))
		}
		if !ok || domain == nil {
			return s.failDeferredRAGIngestSourceJob(ctx, record, job, actor, knowledgeBaseDataDomainNotFoundError())
		}
		model, err := c.SemanticModels(wsID).Get(kbCtx, job.ModelID)
		if err != nil {
			return s.failDeferredRAGIngestSourceJob(ctx, record, job, actor, fmt.Errorf("get semantic model for media workflow: %w", err))
		}
		if _, err := s.ensureKnowledgeBaseMediaRAGWorkflow(kbCtx, wsID, job.ModelID, model.Name, model.Description, templateKey, domain, model.Files); err != nil {
			return s.failDeferredRAGIngestSourceJob(ctx, record, job, actor, fmt.Errorf("ensure knowledge base media workflow: %w", err))
		}
	}
	sourceFileID := firstNonEmptySegmentString(
		ptrValue(record.SourceFileID),
		ptrValue(job.SourceFileID),
		ptrValue(record.KBFileID),
		ptrValue(job.KBFileID),
	)
	result, err := s.workflowService.RunKnowledgeBaseWorkflow(kbCtx, workflowID, map[string]any{
		"source_ref": map[string]any{
			"kind":          "file",
			"resource_type": "file",
			"file_id":       sourceFileID,
			"file_ids":      []string{sourceFileID},
			"file_name":     strings.TrimSpace(ptrValue(record.DisplayName)),
			"volume_id":     record.RawVolumeID,
		},
	})
	if err != nil {
		return s.failDeferredRAGIngestSourceJob(ctx, record, job, actor, fmt.Errorf("run knowledge base workflow: %w", err))
	}
	if result == nil || strings.TrimSpace(result.ExecutionID) == "" {
		return s.failDeferredRAGIngestSourceJob(ctx, record, job, actor, fmt.Errorf("knowledge base workflow execution id is required"))
	}
	job.WorkflowExecutionID = stringPtr(strings.TrimSpace(result.ExecutionID))
	if err := s.upsertKnowledgeBaseSourceJobRun(ctx, job, actor); err != nil {
		return fmt.Errorf("record knowledge base workflow execution: %w", err)
	}
	return nil
}

func resolveTemplateKeyForKnowledgeBaseFile(ctx context.Context, c *moi.Client, wsID string, record *KnowledgeBaseSourceRecord, job *KnowledgeBaseSourceJobRun) (string, error) {
	if record != nil {
		if templateKey, err := templateKeyForKnowledgeBaseFile(record.DisplayName); err == nil {
			return templateKey, nil
		}
	}
	if c == nil || job == nil || job.KBFileID == nil || strings.TrimSpace(*job.KBFileID) == "" {
		return "", fmt.Errorf("knowledge base file metadata is required to select a workflow template")
	}
	metadata, err := c.Files().Get(ctx, wsID, strings.TrimSpace(*job.KBFileID))
	if err != nil {
		return "", fmt.Errorf("get knowledge base file metadata for workflow template selection: %w", err)
	}
	if metadata == nil || strings.TrimSpace(metadata.GetOriginalName()) == "" {
		return "", fmt.Errorf("knowledge base file original name is required to select a workflow template")
	}
	fileName := strings.TrimSpace(metadata.GetOriginalName())
	if record != nil {
		record.DisplayName = stringPtr(fileName)
	}
	return templateKeyForKnowledgeBaseFile(stringPtr(fileName))
}

func (s *semanticModelService) failDeferredRAGIngestSourceJob(ctx context.Context, record KnowledgeBaseSourceRecord, job *KnowledgeBaseSourceJobRun, actor string, err error) error {
	if markErr := s.markKnowledgeBaseSourceFailed(ctx, record.SourceID, err.Error(), actor); markErr != nil {
		return fmt.Errorf("trigger deferred rag ingest source job %s file %s: %w; mark knowledge base source %s failed: %v", job.SourceID, *job.KBFileID, err, record.SourceID, markErr)
	}
	if markErr := s.markKnowledgeBaseSourceJobFailed(ctx, job.JobID, err.Error(), actor); markErr != nil {
		return fmt.Errorf("trigger deferred rag ingest source job %s file %s: %w; mark knowledge base source job %s failed: %v", job.SourceID, *job.KBFileID, err, job.JobID, markErr)
	}
	return fmt.Errorf("%w: trigger deferred rag ingest source job %s file %s: %v", errKnowledgeBaseSourceJobFailed, job.SourceID, *job.KBFileID, err)
}

func isDeferredCatalogFileRAGJob(record KnowledgeBaseSourceRecord, job *KnowledgeBaseSourceJobRun) bool {
	return record.SourceType == kbSourceTypeCatalogFile &&
		job != nil &&
		job.JobType == kbJobTypeRAGIngest &&
		(job.JobStatus == kbSourceJobPending || job.JobStatus == kbSourceJobQueued) &&
		(job.OperationID == nil || *job.OperationID == "")
}

func (s *semanticModelService) reconcileDeferredCatalogFileRAGJob(ctx context.Context, c *moi.Client, wsID string, record KnowledgeBaseSourceRecord, job *KnowledgeBaseSourceJobRun, actor string) error {
	ready, err := s.catalogFileCopyCompletedForRAG(ctx, job.ModelID, job.SourceID)
	if err != nil {
		return err
	}
	if !ready {
		return nil
	}
	// Order: rehydrate -> semantic_model.use -> try reuse (no volume) ->
	// if not reused, dispatch with the same principalCtx/kbCtx + volume.read once.
	// Reuse must not require volume.read; volume gate runs only before workflow dispatch.
	principalCtx, err := rehydrateKnowledgeBaseRAGJobPrincipal(ctx, job)
	if err != nil {
		return s.failDeferredRAGIngestSourceJob(ctx, record, job, actor, err)
	}
	kbCtx, useErr := s.reauthorizeSemanticModelUse(principalCtx, wsID, job.ModelID)
	if useErr != nil {
		return s.handleKnowledgeBaseRAGAuthError(ctx, record, job, actor, useErr, "semantic_model.use reauthorization")
	}
	reused, err := s.tryReuseDeferredCatalogFileRAGJob(kbCtx, c, wsID, record, job, actor)
	if err != nil {
		return err
	}
	if reused {
		return nil
	}
	return s.claimAndTriggerDeferredRAGIngestSourceJobWithAuth(ctx, c, wsID, record, job, actor, principalCtx, kbCtx)
}

func (s *semanticModelService) tryReuseDeferredCatalogFileRAGJob(ctx context.Context, c *moi.Client, wsID string, record KnowledgeBaseSourceRecord, job *KnowledgeBaseSourceJobRun, actor string) (bool, error) {
	if job == nil {
		return false, fmt.Errorf("knowledge base source job run is required")
	}
	model, err := c.SemanticModels(wsID).Get(ctx, job.ModelID)
	if err != nil {
		return false, err
	}
	files, err := appendSemanticModelFiles(model.Files, job.ModelID, nil)
	if err != nil {
		return false, err
	}
	binding, err := parseKBVectorBinding(files)
	if err != nil {
		return false, err
	}
	reuseRecord := record
	if ptrValue(reuseRecord.SourceFileID) == "" {
		reuseRecord.SourceFileID = job.SourceFileID
	}
	if ptrValue(reuseRecord.KBFileID) == "" {
		reuseRecord.KBFileID = job.KBFileID
	}
	reused, operationID, err := s.tryReuseCatalogFileVectors(ctx, reuseRecord, binding, actor)
	if err != nil {
		return false, fmt.Errorf("reuse catalog file vectors %s: %w", ptrValue(reuseRecord.KBFileID), err)
	}
	if !reused {
		return false, nil
	}
	job.JobStatus = kbSourceJobSucceeded
	job.OperationID = stringPtr(operationID)
	job.WorkflowExecutionID = nil
	job.SourceFileID = reuseRecord.SourceFileID
	job.KBFileID = reuseRecord.KBFileID
	job.Error = nil
	if err := s.upsertKnowledgeBaseSourceJobRun(ctx, job, actor); err != nil {
		return false, fmt.Errorf("record deferred catalog file vector reuse job: %w", err)
	}
	return true, nil
}

func (s *semanticModelService) claimDeferredRAGIngestSourceJob(ctx context.Context, record KnowledgeBaseSourceRecord, job *KnowledgeBaseSourceJobRun, operationID, actor string) (bool, error) {
	if job == nil {
		return false, fmt.Errorf("knowledge base source job run is required")
	}
	db := ctxutil.TenantDBFrom(ctx)
	if db == nil {
		return false, fmt.Errorf("tenant db is required")
	}
	res := db.WithContext(ctx).Exec(`UPDATE knowledge_base_source_job_runs
		SET job_status = ?, operation_id = ?, source_file_id = ?, kb_file_id = ?, error = NULL, updated_by = ?
		WHERE job_id = ? AND model_id = ? AND source_id = ? AND job_type = ?
		  AND job_status IN (?, ?)
		  AND (workflow_execution_id IS NULL OR workflow_execution_id = '')
		  AND (operation_id IS NULL OR operation_id = '' OR operation_id = ?)`,
		kbSourceJobRunning, operationID, record.SourceFileID, record.KBFileID, actor,
		job.JobID, job.ModelID, job.SourceID, kbJobTypeRAGIngest, kbSourceJobPending, kbSourceJobQueued, operationID)
	if res.Error != nil {
		return false, res.Error
	}
	if res.RowsAffected == 0 {
		return false, nil
	}
	job.JobStatus = kbSourceJobRunning
	job.OperationID = stringPtr(operationID)
	job.SourceFileID = record.SourceFileID
	job.KBFileID = record.KBFileID
	job.Error = nil
	return true, nil
}

func (s *semanticModelService) catalogFileCopyCompletedForRAG(ctx context.Context, modelID int64, sourceID string) (bool, error) {
	db := ctxutil.TenantDBFrom(ctx)
	if db == nil {
		return false, fmt.Errorf("tenant db is required")
	}
	var status string
	err := db.WithContext(ctx).Raw(`SELECT job_status
		FROM knowledge_base_source_job_runs
		WHERE model_id = ? AND source_id = ? AND job_type = ?
		LIMIT 1`, modelID, sourceID, kbJobTypeCopy).Row().Scan(&status)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, fmt.Errorf("knowledge base copy job not found for source %s", sourceID)
		}
		return false, err
	}
	return status == kbSourceJobSucceeded, nil
}

func (s *semanticModelService) localFileLoadCompletedForRAG(ctx context.Context, modelID int64, sourceID string) (bool, error) {
	db := ctxutil.TenantDBFrom(ctx)
	if db == nil {
		return false, fmt.Errorf("tenant db is required")
	}
	var operationID sql.NullString
	var status string
	err := db.WithContext(ctx).Raw(`SELECT job_status, operation_id
		FROM knowledge_base_source_job_runs
		WHERE model_id = ? AND source_id = ? AND job_type = ?
		LIMIT 1`, modelID, sourceID, kbJobTypeLoad).Row().Scan(&status, &operationID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, fmt.Errorf("knowledge base load job not found for source %s", sourceID)
		}
		return false, err
	}
	taskID, ok := importTaskIDFromOperation(nullStringPtr(operationID))
	if !ok {
		return status == kbSourceJobSucceeded, nil
	}
	state, err := s.getKnowledgeBaseImportTaskState(ctx, taskID)
	if err != nil {
		return false, err
	}
	return state.Status == kbSourceJobSucceeded, nil
}

func nullStringPtr(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	return &value.String
}

func (s *semanticModelService) publishExternalWorkflowSegmentVersion(ctx context.Context, c *moi.Client, wsID string, record KnowledgeBaseSourceRecord, kbFileID string, job *KnowledgeBaseSourceJobRun, actor string) (bool, error) {
	if kbFileID == "" {
		return false, fmt.Errorf("kb_file_id is required")
	}
	if c == nil {
		return false, fmt.Errorf("moi client is required")
	}
	model, err := c.SemanticModels(wsID).Get(ctx, record.ModelID)
	if err != nil {
		return false, err
	}
	binding, err := parseKBVectorBinding(model.Files)
	if err != nil {
		return false, err
	}
	nextRecord := record
	nextRecord.KBFileID = stringPtr(kbFileID)
	segments, indexVersion, err := s.importExternalWorkflowSegmentsFromVectorRows(ctx, nextRecord, binding)
	if err != nil {
		if isExternalWorkflowVersionAlreadyCurrent(record, kbFileID, indexVersion) {
			return false, nil
		}
		return false, err
	}
	if isExternalWorkflowVersionAlreadyCurrent(record, kbFileID, indexVersion) {
		return false, nil
	}
	versionID := stableID("kb-segver", record.SourceID, indexVersion, kbSegmentSourceExternal)
	if err := prepareNextSegmentVersion(nextRecord, versionID, indexVersion, segments); err != nil {
		return false, err
	}
	base := SemanticModelSegmentMutationBase{
		BaseSegmentVersionID: record.SegmentVersionID,
		BaseIndexVersion:     record.IndexVersion,
	}
	err = s.commitSegmentVersionWithTxHook(ctx, nextRecord, binding, kbSegmentSourceExternal, versionID, indexVersion, segments, kbSegmentMaterialization{}, base, func(tx *gorm.DB) error {
		if job == nil {
			return nil
		}
		if err := updateKnowledgeBaseSourceJobRunSucceededWithTx(tx, job, actor); err != nil {
			return fmt.Errorf("update rag ingest job: %w", err)
		}
		return nil
	})
	if err != nil {
		var duplicateVersionErr *duplicateSegmentVersionInsertError
		if errors.As(err, &duplicateVersionErr) && duplicateVersionErr.versionID == versionID {
			alreadyCommitted, checkErr := externalWorkflowSegmentVersionAlreadyCommitted(ctx, record, kbFileID, versionID, indexVersion)
			if checkErr != nil {
				return false, checkErr
			}
			if alreadyCommitted {
				return false, nil
			}
		}
		return false, err
	}
	return true, nil
}

func isExternalWorkflowVersionAlreadyCurrent(record KnowledgeBaseSourceRecord, kbFileID string, indexVersion int64) bool {
	return record.KBFileID != nil && *record.KBFileID == kbFileID &&
		record.IndexVersion != nil && *record.IndexVersion == indexVersion &&
		record.SegmentVersionID != nil && *record.SegmentVersionID != ""
}

func externalWorkflowSegmentVersionAlreadyCommitted(ctx context.Context, record KnowledgeBaseSourceRecord, kbFileID, versionID string, indexVersion int64) (bool, error) {
	db := ctxutil.TenantDBFrom(ctx)
	if db == nil {
		return false, fmt.Errorf("tenant db is required")
	}
	var count int64
	if err := db.WithContext(ctx).Raw(`SELECT COUNT(1)
		FROM knowledge_base_segment_versions v
		INNER JOIN knowledge_base_sources s ON s.model_id = v.model_id AND s.source_id = v.source_id
		WHERE v.version_id = ? AND v.model_id = ? AND v.source_id = ? AND v.kb_file_id = ? AND v.index_version = ? AND v.status = ? AND v.source = ?
		  AND s.kb_file_id = ? AND s.segment_version_id = ? AND s.index_version = ? AND s.status = ?`,
		versionID, record.ModelID, record.SourceID, kbFileID, indexVersion, kbSegmentStatusCommitted, kbSegmentSourceExternal,
		kbFileID, versionID, indexVersion, kbSourceStatusSucceeded).Row().Scan(&count); err != nil {
		return false, err
	}
	return count == 1, nil
}

func importTaskIDFromOperation(operationID *string) (string, bool) {
	if operationID == nil {
		return "", false
	}
	const prefix = "import_task:"
	if !strings.HasPrefix(*operationID, prefix) {
		return "", false
	}
	taskID := strings.TrimPrefix(*operationID, prefix)
	return taskID, taskID != ""
}

func isStructuredLocalLoadJob(job KnowledgeBaseSourceJobRun) bool {
	if _, ok := importTaskIDFromOperation(job.OperationID); ok {
		return true
	}
	if _, ok, _ := parseLocalStructuredUploadOperation(job.OperationID); ok {
		return true
	}
	return false
}

func (s *semanticModelService) getKnowledgeBaseImportTaskState(ctx context.Context, taskID string) (*KnowledgeBaseImportTaskState, error) {
	db := ctxutil.TenantDBFrom(ctx)
	if db == nil {
		return nil, fmt.Errorf("tenant db is required")
	}
	var task model.ImportTask
	if err := db.WithContext(ctx).
		Select("id", "status", "task_meta").
		Where("id = ?", taskID).
		First(&task).Error; err != nil {
		return nil, fmt.Errorf("get import task %s: %w", taskID, err)
	}
	state := &KnowledgeBaseImportTaskState{TaskID: taskID, Status: importTaskStatusToKnowledgeBaseJobStatus(task.Status)}
	if task.Status == model.ImportTaskStatusFailed {
		state.Error = importTaskErrorFromMeta(task.TaskMeta)
	}
	var run model.ImportTaskRun
	if err := db.WithContext(ctx).
		Select("id", "workflow_execution_id", "status", "error_message").
		Where("import_task_id = ?", taskID).
		Order("created_at DESC, id DESC").
		First(&run).Error; err == nil {
		state.Status = importTaskRunStatusToKnowledgeBaseJobStatus(run.Status, state.Status)
		if run.ErrorMessage != "" {
			state.Error = stringPtr(run.ErrorMessage)
		}
		if run.WorkflowExecutionID != "" {
			state.WorkflowExecutionID = stringPtr(run.WorkflowExecutionID)
		}
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("get import task run %s: %w", taskID, err)
	}
	return state, nil
}

func (s *semanticModelService) structuredTableResultsForLoadJob(ctx context.Context, job *KnowledgeBaseSourceJobRun) ([]model.StructuredTableResult, bool, error) {
	if job == nil || job.JobType != kbJobTypeLoad {
		return nil, false, nil
	}
	taskID, ok := importTaskIDFromOperation(job.OperationID)
	if !ok {
		return nil, false, nil
	}
	db := ctxutil.TenantDBFrom(ctx)
	if db == nil {
		return nil, false, fmt.Errorf("tenant db is required")
	}
	var task model.ImportTask
	if err := db.WithContext(ctx).
		Select("id", "status", "task_meta").
		Where("id = ?", taskID).
		First(&task).Error; err != nil {
		return nil, false, fmt.Errorf("get import task %s: %w", taskID, err)
	}
	completed := task.Status == model.ImportTaskStatusFinished
	taskJobStatus := importTaskStatusToKnowledgeBaseJobStatus(task.Status)
	if !completed {
		var run model.ImportTaskRun
		runErr := db.WithContext(ctx).
			Select("id", "status", "error_message").
			Where("import_task_id = ?", taskID).
			Order("created_at DESC, id DESC").
			First(&run).Error
		if runErr == nil {
			switch importTaskRunStatusToKnowledgeBaseJobStatus(run.Status, taskJobStatus) {
			case kbSourceJobSucceeded:
				completed = true
			case kbSourceJobFailed:
				errMsg := run.ErrorMessage
				if errMsg == "" {
					errMsg = fmt.Sprintf("import task run status %d", run.Status)
				}
				return nil, false, fmt.Errorf("import task %s failed: %s", taskID, errMsg)
			}
		} else if !errors.Is(runErr, gorm.ErrRecordNotFound) {
			return nil, false, fmt.Errorf("get import task run %s: %w", taskID, runErr)
		}
	}
	if !completed {
		if taskJobStatus == kbSourceJobFailed {
			errMsg := "import task failed"
			if taskErr := importTaskErrorFromMeta(task.TaskMeta); taskErr != nil && *taskErr != "" {
				errMsg = *taskErr
			}
			return nil, false, fmt.Errorf("import task %s failed: %s", taskID, errMsg)
		}
		return nil, false, nil
	}
	var payload struct {
		StructuredTableResults []model.StructuredTableResult `json:"structured_table_results"`
	}
	if err := json.Unmarshal([]byte(task.TaskMeta), &payload); err != nil {
		return nil, false, fmt.Errorf("parse import task %s structured_table_results: %w", taskID, err)
	}
	results := make([]model.StructuredTableResult, 0, len(payload.StructuredTableResults))
	for i := range payload.StructuredTableResults {
		result := payload.StructuredTableResults[i]
		if result.TableID <= 0 && result.DBName != "" && result.TableName != "" {
			ref, err := s.resolveCatalogTableByDatabaseAndName(ctx, result.DatabaseID, result.DBName, result.TableName)
			if err != nil {
				return nil, false, fmt.Errorf("resolve catalog table for import task %s structured result %s.%s: %w", taskID, result.DBName, result.TableName, err)
			}
			result.TableID = ref.tableID
			if result.DatabaseID <= 0 {
				result.DatabaseID = ref.databaseID
			}
			result.DBName = ref.dbName
			result.TableName = ref.tableName
		}
		if result.TableID > 0 && (result.DBName == "" || result.TableName == "") {
			ref, err := s.resolveCatalogTableByID(ctx, result.TableID)
			if err != nil {
				return nil, false, fmt.Errorf("resolve catalog table %d for import task %s structured result: %w", result.TableID, taskID, err)
			}
			if result.DatabaseID <= 0 {
				result.DatabaseID = ref.databaseID
			}
			result.DBName = ref.dbName
			result.TableName = ref.tableName
		}
		if result.TableID > 0 && result.DBName != "" && result.TableName != "" {
			results = append(results, result)
		}
	}
	if len(results) > 0 {
		return results, true, nil
	}
	return nil, false, fmt.Errorf("import task %s completed without structured_table_results", taskID)
}

func (s *semanticModelService) ensureLegacyStructuredLoadImportTask(ctx context.Context, record KnowledgeBaseSourceRecord, job *KnowledgeBaseSourceJobRun, actor string) (bool, error) {
	// Legacy compatibility only: new structured local uploads create import_task
	// before this reconcile step and store just import_task:<id> in operation_id.
	payload, ok, err := parseLocalStructuredUploadOperation(job.OperationID)
	if err != nil || !ok {
		return false, err
	}
	if s.localImportService == nil {
		return false, fmt.Errorf("knowledge base local file import service is not configured")
	}
	if record.RawVolumeID <= 0 {
		return false, fmt.Errorf("raw_volume_id is required")
	}
	imported, err := s.localImportService.UploadToVolume(ctx, KnowledgeBaseLocalFileImportParams{
		VolumeID:    record.RawVolumeID,
		FileName:    payload.FileName,
		FileID:      payload.FileID,
		UploadKind:  kbLocalUploadKindStructured,
		TableConfig: payload.TableConfig,
	})
	if err != nil {
		return false, fmt.Errorf("create local file import task %s: %w", payload.FileName, err)
	}
	if imported == nil || imported.TaskID == "" || len(imported.FileIDs) == 0 || imported.FileIDs[0] == "" {
		return false, fmt.Errorf("local file import returned incomplete result")
	}
	job.JobStatus = kbSourceJobRunning
	job.OperationID = stringPtr("import_task:" + imported.TaskID)
	job.SourceFileID = stringPtr(payload.FileID)
	job.KBFileID = stringPtr(payload.FileID)
	job.Error = nil
	if imported.WorkflowExecutionID != nil && *imported.WorkflowExecutionID != "" {
		job.WorkflowExecutionID = imported.WorkflowExecutionID
	}
	if err := s.upsertKnowledgeBaseSourceJobRun(ctx, job, actor); err != nil {
		return false, fmt.Errorf("record local file load job result %s: %w", payload.FileName, err)
	}
	return true, nil
}

func importTaskStatusToKnowledgeBaseJobStatus(status model.ImportTaskStatus) string {
	switch status {
	case model.ImportTaskStatusFinished:
		return kbSourceJobSucceeded
	case model.ImportTaskStatusFailed:
		return kbSourceJobFailed
	case model.ImportTaskStatusCreated, model.ImportTaskStatusUploading, model.ImportTaskStatusPausing, model.ImportTaskStatusPaused:
		return kbSourceJobRunning
	default:
		return kbSourceJobPending
	}
}

func importTaskRunStatusToKnowledgeBaseJobStatus(status model.ImportTaskRunStatus, fallback string) string {
	switch status {
	case model.ImportTaskRunStatusCompleted:
		return kbSourceJobSucceeded
	case model.ImportTaskRunStatusFailed, model.ImportTaskRunStatusCancelled:
		return kbSourceJobFailed
	case model.ImportTaskRunStatusCreated, model.ImportTaskRunStatusRunning:
		return kbSourceJobRunning
	default:
		return fallback
	}
}

func importTaskErrorFromMeta(raw string) *string {
	if raw == "" {
		return nil
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil
	}
	for _, key := range []string{"reason", "error", "error_message"} {
		if value, ok := payload[key].(string); ok && value != "" {
			return &value
		}
	}
	return nil
}

func workflowExecutionToKnowledgeBaseJobStatus(exec *moi.WorkflowExecutionEnvelope) (string, *string) {
	if exec == nil {
		return kbSourceJobPending, nil
	}
	switch exec.Execution.Status {
	case "completed", "succeeded", "success":
		return kbSourceJobSucceeded, nil
	case "failed", "cancelled":
		errMsg := exec.Execution.Error
		if errMsg == "" {
			errMsg = exec.Execution.CaseError
		}
		if errMsg == "" {
			errMsg = exec.Execution.Status
		}
		return kbSourceJobFailed, &errMsg
	case "running", "triggered", "scheduled", "preparing", "pending", "paused":
		return kbSourceJobRunning, nil
	default:
		return kbSourceJobPending, nil
	}
}

func fileExecutionSummaryToKnowledgeBaseJobStatus(exec moi.FileExecutionSummary) (string, *string) {
	switch exec.Status {
	case "completed", "succeeded", "success":
		return kbSourceJobSucceeded, nil
	case "failed", "cancelled":
		errMsg := exec.Error
		if errMsg == "" {
			errMsg = exec.Status
		}
		return kbSourceJobFailed, &errMsg
	case "running", "triggered", "scheduled", "preparing", "pending", "paused":
		return kbSourceJobRunning, nil
	default:
		return kbSourceJobPending, nil
	}
}

func (s *semanticModelService) RunPendingKnowledgeBaseSourceJobs(ctx context.Context, params RunPendingKnowledgeBaseSourceJobsParams) error {
	wsID, err := callerWorkspaceID(ctx)
	if err != nil {
		return err
	}
	return coreclient.Execute(ctx, coreclient.FromContext(ctx), func(callCtx context.Context, client *moi.Client) error {
		return s.runPendingKnowledgeBaseSourceJobs(callCtx, client, wsID, params.ModelID, semanticModelActor(callCtx), true, true, kbSourceJobReconcileBatchSize, false)
	})
}

func (s *semanticModelService) ReconcileKnowledgeBaseSourceJobs(ctx context.Context, params ReconcileKnowledgeBaseSourceJobsParams) error {
	wsID, err := callerWorkspaceID(ctx)
	if err != nil {
		return err
	}
	return coreclient.Execute(ctx, coreclient.FromContext(ctx), func(callCtx context.Context, client *moi.Client) error {
		actor := semanticModelActor(callCtx)
		// Historical jobs without a create-time freeze stay fail-closed; no adopt.
		return s.reconcileKnowledgeBaseSourceJobs(callCtx, client, wsID, params.ModelID, actor, kbSourceJobReconcileBackfillBatchSize, true, true)
	})
}

func KnowledgeBaseModelIDForImportTask(ctx context.Context, db *gorm.DB, taskID string) (int64, bool, error) {
	if db == nil {
		return 0, false, fmt.Errorf("tenant db is required")
	}
	operationID := "import_task:" + taskID
	var modelID int64
	err := db.WithContext(ctx).Raw(`SELECT model_id
		FROM knowledge_base_source_job_runs
		WHERE job_type = ? AND operation_id = ?
		LIMIT 1`, kbJobTypeLoad, operationID).Scan(&modelID).Error
	if err != nil {
		return 0, false, err
	}
	if modelID == 0 {
		return 0, false, nil
	}
	return modelID, true, nil
}

func (s *semanticModelService) reconcileKnowledgeBaseSourceJobs(ctx context.Context, c *moi.Client, wsID string, modelID int64, actor string, backfillLimit int, reconcileStructuredLoads bool, reconcileRAGIngest bool) error {
	if backfillLimit <= 0 {
		return nil
	}
	if backfillLimit > 0 {
		if _, err := s.backfillLegacySourcesBatch(ctx, c, wsID, modelID, actor, backfillLimit); err != nil {
			return err
		}
	}
	return s.runPendingKnowledgeBaseSourceJobs(ctx, c, wsID, modelID, actor, reconcileStructuredLoads, reconcileRAGIngest, kbSourceJobReconcileBatchSize, true)
}

func (s *semanticModelService) runPendingKnowledgeBaseSourceJobs(ctx context.Context, c *moi.Client, wsID string, modelID int64, actor string, reconcileStructuredLoads bool, reconcileRAGIngest bool, limit int, prioritized bool) error {
	if modelID == 0 {
		return semanticModelNotFoundError()
	}
	if limit <= 0 {
		return nil
	}
	var fastBindJobs []KnowledgeBaseSourceJobRun
	var copyJobs []KnowledgeBaseSourceJobRun
	var err error
	if prioritized {
		fastBindJobs, err = s.listFastKnowledgeBaseFileBindJobs(ctx, modelID, kbSourceJobFastBindBatchSize)
		if err != nil {
			return err
		}
	} else if reconcileRAGIngest {
		copyJobs, err = s.listPendingKnowledgeBaseSourceJobRuns(ctx, modelID, kbJobTypeCopy, true, limit)
		if err != nil {
			return err
		}
	}
	var pendingJobs []KnowledgeBaseSourceJobRun
	if prioritized {
		pendingJobs, err = s.listKnowledgeBaseSourceJobRunsByStatuses(ctx, modelID, kbJobTypeTableClone, []string{kbSourceJobPending, kbSourceJobQueued, kbSourceJobRunning}, kbSourceJobReconcileBatchSize)
	} else {
		pendingJobs, err = s.listPendingKnowledgeBaseSourceJobRuns(ctx, modelID, kbJobTypeTableClone, reconcileRAGIngest, limit)
	}
	if err != nil {
		return err
	}
	var loadJobs []KnowledgeBaseSourceJobRun
	if reconcileStructuredLoads {
		if prioritized {
			finalizeLoads, err := s.listStructuredLoadKnowledgeBaseSourceJobRunsByStatuses(ctx, modelID, []string{kbSourceJobSucceeded}, kbSourceJobReconcileBatchSize, false)
			if err != nil {
				return err
			}
			waitingLoads, err := s.listStructuredLoadKnowledgeBaseSourceJobRunsByStatuses(ctx, modelID, []string{kbSourceJobQueued, kbSourceJobRunning}, kbSourceJobReconcileBatchSize, true)
			if err != nil {
				return err
			}
			loadJobs = append(finalizeLoads, waitingLoads...)
		} else {
			loadJobs, err = s.listStructuredLoadKnowledgeBaseSourceJobRuns(ctx, modelID, limit)
		}
		if err != nil {
			return err
		}
	}
	var ragJobs []KnowledgeBaseSourceJobRun
	if reconcileRAGIngest && s.workflowService != nil {
		if prioritized {
			finalizeJobs, err := s.listRAGIngestKnowledgeBaseSourceJobRuns(ctx, modelID, []string{kbSourceJobSucceeded}, kbSourceJobReconcileBatchSize, false)
			if err != nil {
				return err
			}
			dispatchJobs, err := s.listRAGIngestKnowledgeBaseSourceJobRuns(ctx, modelID, []string{kbSourceJobPending, kbSourceJobQueued}, kbSourceJobRAGDispatchBatchSize, false)
			if err != nil {
				return err
			}
			waitingJobs, err := s.listRAGIngestKnowledgeBaseSourceJobRuns(ctx, modelID, []string{kbSourceJobRunning}, kbSourceJobReconcileBatchSize, true)
			if err != nil {
				return err
			}
			ragJobs = append(finalizeJobs, dispatchJobs...)
			ragJobs = append(ragJobs, waitingJobs...)
			failedJobs, err := s.listRAGIngestKnowledgeBaseSourceJobRuns(ctx, modelID, []string{kbSourceJobFailed}, kbSourceJobReconcileBatchSize, true)
			if err != nil {
				return err
			}
			ragJobs = append(ragJobs, failedJobs...)
		} else {
			ragJobs, err = s.listPendingKnowledgeBaseSourceJobRuns(ctx, modelID, kbJobTypeRAGIngest, true, limit)
		}
		if err != nil {
			return err
		}
	}
	if len(fastBindJobs) == 0 && len(copyJobs) == 0 && len(pendingJobs) == 0 && len(loadJobs) == 0 && len(ragJobs) == 0 {
		return nil
	}
	jobSourceIDs := make([]string, 0, len(fastBindJobs)+len(copyJobs)+len(pendingJobs)+len(loadJobs)+len(ragJobs))
	for _, jobs := range [][]KnowledgeBaseSourceJobRun{fastBindJobs, copyJobs, pendingJobs, loadJobs, ragJobs} {
		for _, job := range jobs {
			jobSourceIDs = append(jobSourceIDs, job.SourceID)
		}
	}
	var records []KnowledgeBaseSourceRecord
	if prioritized {
		records, err = s.listKnowledgeBaseSourceRowsByIDs(ctx, modelID, compactUniqueStrings(jobSourceIDs))
	} else {
		records, err = s.listKnowledgeBaseSources(ctx, modelID)
	}
	if err != nil {
		return err
	}
	recordsBySourceID := make(map[string]KnowledgeBaseSourceRecord, len(records))
	for _, record := range records {
		recordsBySourceID[record.SourceID] = record
	}
	if prioritized {
		if err := s.runKnowledgeBaseFastFileBindJobs(ctx, c, wsID, recordsBySourceID, fastBindJobs, actor); err != nil {
			return err
		}
	}
	for _, job := range copyJobs {
		record, ok := recordsBySourceID[job.SourceID]
		if !ok {
			return fmt.Errorf("knowledge base source %s not found for job %s", job.SourceID, job.JobID)
		}
		claimed, err := s.runKnowledgeBaseCatalogFileCopyJob(ctx, c, wsID, &record, &job, actor)
		if err != nil {
			if errors.Is(err, errKnowledgeBaseSourceJobFailed) {
				continue
			}
			return err
		}
		if !claimed {
			continue
		}
		recordsBySourceID[record.SourceID] = record
	}
	completedStructuredFileIDs := []string{}
	completed := make([]completedKnowledgeBaseTableJob, 0, len(pendingJobs))
	for _, job := range pendingJobs {
		record, ok := recordsBySourceID[job.SourceID]
		if !ok {
			return fmt.Errorf("knowledge base source %s not found for job %s", job.SourceID, job.JobID)
		}
		table, claimed, err := s.runKnowledgeBaseTableCloneJob(ctx, &record, &job, actor)
		if err != nil {
			if errors.Is(err, errKnowledgeBaseSourceJobFailed) {
				continue
			}
			return err
		}
		if !claimed {
			continue
		}
		if table != nil {
			completed = append(completed, completedKnowledgeBaseTableJob{source: record, job: job})
		}
	}
	for _, job := range loadJobs {
		record, ok := recordsBySourceID[job.SourceID]
		if !ok {
			continue
		}
		if record.SourceType == kbSourceTypeLocalFile && !isStructuredLocalLoadJob(job) {
			if job.JobStatus == kbSourceJobSucceeded || job.JobStatus == kbSourceJobFailed {
				continue
			}
			claimed, err := s.runKnowledgeBaseLocalFileLoadJob(ctx, c, wsID, &record, &job, actor)
			if err != nil {
				if errors.Is(err, errKnowledgeBaseSourceJobFailed) {
					continue
				}
				return err
			}
			if !claimed {
				continue
			}
			recordsBySourceID[record.SourceID] = record
			continue
		}
		if !isStructuredLoadSourceType(record.SourceType) {
			continue
		}
		tableID := int64(0)
		if record.KBTableID != nil {
			tableID = *record.KBTableID
		}
		if tableID <= 0 && record.SourceTableID != nil {
			tableID = *record.SourceTableID
		}
		if record.Status == kbSourceStatusSucceeded &&
			tableID > 0 &&
			record.DBName != nil && *record.DBName != "" &&
			record.TableName != nil && *record.TableName != "" {
			continue
		}
		createdImportTask, err := s.ensureLegacyStructuredLoadImportTask(ctx, record, &job, actor)
		if err != nil {
			err = fmt.Errorf("create structured load import task for job %s source %s: %w", job.JobID, job.SourceID, err)
			if markErr := s.markKnowledgeBaseSourceFailed(ctx, record.SourceID, err.Error(), actor); markErr != nil {
				return fmt.Errorf("%w; mark knowledge base source %s failed: %v", err, record.SourceID, markErr)
			}
			if markErr := s.markKnowledgeBaseSourceJobFailed(ctx, job.JobID, err.Error(), actor); markErr != nil {
				return fmt.Errorf("%w; mark knowledge base source job %s failed: %v", err, job.JobID, markErr)
			}
			continue
		}
		if createdImportTask {
			continue
		}
		results, ok, err := s.structuredTableResultsForLoadJob(ctx, &job)
		if err != nil {
			err = fmt.Errorf("resolve structured load result for job %s source %s: %w", job.JobID, job.SourceID, err)
			if markErr := s.markKnowledgeBaseSourceFailed(ctx, record.SourceID, err.Error(), actor); markErr != nil {
				return fmt.Errorf("%w; mark knowledge base source %s failed: %v", err, record.SourceID, markErr)
			}
			if markErr := s.markKnowledgeBaseSourceJobFailed(ctx, job.JobID, err.Error(), actor); markErr != nil {
				return fmt.Errorf("%w; mark knowledge base source job %s failed: %v", err, job.JobID, markErr)
			}
			continue
		}
		if !ok {
			if err := s.markKnowledgeBaseSourceJobRunning(ctx, job.JobID, actor); err != nil {
				return err
			}
			continue
		}
		completedStructuredFileIDs = appendUniqueStrings(completedStructuredFileIDs, compactNonEmptyStrings(
			ptrValue(record.SourceFileID),
			ptrValue(record.KBFileID),
			ptrValue(job.SourceFileID),
			ptrValue(job.KBFileID),
		))
		baseRecord := record
		baseJob := job
		for i := range results {
			result := results[i]
			source := baseRecord
			source.SourceType = kbSourceTypeCatalogTable
			source.SourceFileID = nil
			source.SourceTableID = nil
			source.KBFileID = nil
			source.Status = kbSourceStatusSucceeded
			source.Error = nil
			source.KBTableID = int64Ptr(result.TableID)
			source.DBName = stringPtr(result.DBName)
			source.TableName = stringPtr(result.TableName)
			source.DisplayName = stringPtr(result.TableName)

			run := baseJob
			if i > 0 {
				source.SourceID = stableID("kb-source", modelID, kbSourceTypeCatalogTable, result.TableID)
				run = newKnowledgeBaseJobRun(&source, kbJobTypeLoad, stableID("kb-job", source.SourceID, kbJobTypeLoad), actor)
				run.OperationID = baseJob.OperationID
				run.WorkflowExecutionID = baseJob.WorkflowExecutionID
			}
			run.JobStatus = kbSourceJobSucceeded
			run.SourceFileID = nil
			run.SourceTableID = nil
			run.KBFileID = nil
			run.KBTableID = int64Ptr(result.TableID)
			run.Error = nil

			completed = append(completed, completedKnowledgeBaseTableJob{
				source:        source,
				job:           run,
				new:           i > 0,
				ownerSourceID: baseRecord.SourceID,
				ownerJobID:    baseJob.JobID,
				ownerJobType:  baseJob.JobType,
			})
		}
	}
	for _, job := range ragJobs {
		record, ok := recordsBySourceID[job.SourceID]
		if !ok || record.SourceType == kbSourceTypeCatalogTable {
			continue
		}
		if err := s.reconcileRAGIngestSourceJob(ctx, c, wsID, record, &job, actor); err != nil {
			if errors.Is(err, errKnowledgeBaseSourceJobFailed) {
				continue
			}
			return err
		}
	}
	if len(completed) == 0 {
		return nil
	}
	return s.commitCompletedKnowledgeBaseTableJobs(ctx, c, wsID, modelID, actor, completedStructuredFileIDs, completed)
}

func (s *semanticModelService) commitCompletedKnowledgeBaseTableJobs(ctx context.Context, c *moi.Client, wsID string, modelID int64, actor string, completedStructuredFileIDs []string, completed []completedKnowledgeBaseTableJob) error {
	db := ctxutil.TenantDBFrom(ctx)
	if db == nil {
		return fmt.Errorf("tenant db is required")
	}
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txCtx := ctxutil.WithTenantDB(ctx, tx)
		if err := s.lockKnowledgeBaseDataDomainForAppend(txCtx, modelID); err != nil {
			return err
		}
		active := make([]completedKnowledgeBaseTableJob, 0, len(completed))
		completedTableTargets := make(map[string][]string)
		activeOwners := make(map[string]bool)
		for _, item := range completed {
			ownerSourceID := item.ownerSourceID
			if ownerSourceID == "" {
				ownerSourceID = item.source.SourceID
			}
			ownerJobID := item.ownerJobID
			if ownerJobID == "" {
				ownerJobID = item.job.JobID
			}
			ownerJobType := item.ownerJobType
			if ownerJobType == "" {
				ownerJobType = item.job.JobType
			}
			if ownerJobType == kbJobTypeTableClone || ownerJobType == kbJobTypeLoad {
				ownerKey := ownerSourceID + "\x00" + ownerJobID + "\x00" + ownerJobType
				ownerActive, checked := activeOwners[ownerKey]
				if !checked {
					var count int64
					if err := tx.WithContext(ctx).Raw(`SELECT COUNT(*)
						FROM knowledge_base_sources kbs
						JOIN knowledge_base_source_job_runs jr
							ON jr.model_id = kbs.model_id AND jr.source_id = kbs.source_id
						WHERE kbs.model_id = ? AND kbs.source_id = ? AND kbs.status <> ?
							AND jr.job_id = ? AND jr.job_type = ?`,
						modelID, ownerSourceID, kbSourceStatusRemoved, ownerJobID, ownerJobType).Scan(&count).Error; err != nil {
						return fmt.Errorf("validate knowledge base table completion: %w", err)
					}
					ownerActive = count > 0
					activeOwners[ownerKey] = ownerActive
				}
				if !ownerActive {
					continue
				}
			}
			active = append(active, item)
			if item.source.DBName != nil && item.source.TableName != nil {
				completedTableTargets[*item.source.DBName] = append(completedTableTargets[*item.source.DBName], *item.source.TableName)
			}
		}
		if len(active) == 0 {
			return nil
		}
		model, err := c.SemanticModels(wsID).Get(txCtx, modelID)
		if err != nil {
			return err
		}
		filesJSON := model.Files
		for _, fileID := range completedStructuredFileIDs {
			nextFiles, err := removeSemanticModelFileID(filesJSON, fileID)
			if err != nil {
				return err
			}
			filesJSON = nextFiles
		}
		tablesJSON, err := appendSemanticModelTables(model.Tables, mapToSemanticModelTables(completedTableTargets))
		if err != nil {
			return err
		}
		if _, err := c.SemanticModels(wsID).Update(txCtx, modelID, &moi.SemanticModelUpsertRequest{
			Name:        model.Name,
			Description: model.Description,
			Tables:      tablesJSON,
			Files:       filesJSON,
		}); err != nil {
			return fmt.Errorf("update semantic model table sources: %w", err)
		}
		for _, item := range active {
			if item.new {
				if _, err := insertKnowledgeBaseSourceIdempotentWithTx(tx.WithContext(ctx), &item.source, actor); err != nil {
					return fmt.Errorf("insert structured load table source %s: %w", item.source.SourceID, err)
				}
				if err := upsertKnowledgeBaseSourceJobRunWithTx(tx.WithContext(ctx), &item.job, actor); err != nil {
					return fmt.Errorf("upsert structured load table job %s: %w", item.job.JobID, err)
				}
				continue
			}
			if err := s.updateKnowledgeBaseTableSourceSucceeded(txCtx, &item.source, actor); err != nil {
				return fmt.Errorf("update catalog table source: %w", err)
			}
			if err := s.updateKnowledgeBaseSourceJobRunSucceeded(txCtx, &item.job, actor); err != nil {
				return fmt.Errorf("update catalog table clone job: %w", err)
			}
		}
		return nil
	})
	if !errors.Is(err, errKnowledgeBaseDataDomainLockMissing) {
		return err
	}
	out := err
	msg := knowledgeBaseDataDomainNotFoundError().Error()
	for _, item := range completed {
		if markErr := s.markKnowledgeBaseSourceFailed(ctx, item.source.SourceID, msg, actor); markErr != nil {
			out = fmt.Errorf("%w; mark knowledge base source %s failed: %v", out, item.source.SourceID, markErr)
		}
		if markErr := s.markKnowledgeBaseSourceJobFailed(ctx, item.job.JobID, msg, actor); markErr != nil {
			out = fmt.Errorf("%w; mark knowledge base source job %s failed: %v", out, item.job.JobID, markErr)
		}
	}
	return out
}

// Fix #3: ListEntries — propagate resolve errors, return empty only when model doesn't exist yet

func (s *semanticModelService) insertKnowledgeBaseSourceJobRun(ctx context.Context, job *KnowledgeBaseSourceJobRun, actor string) error {
	if job == nil {
		return fmt.Errorf("knowledge base source job run is required")
	}
	db := ctxutil.TenantDBFrom(ctx)
	if db == nil {
		return fmt.Errorf("tenant db is required")
	}
	return db.WithContext(ctx).Exec(`INSERT INTO knowledge_base_source_job_runs
		(job_id, source_id, model_id, job_type, job_status, idempotency_key, operation_id, workflow_execution_id, runtime_actor_moi_user_id, runtime_effective_role_id, runtime_is_workspace_owner, source_file_id, kb_file_id, source_table_id, kb_table_id, retry_count, next_retry_at, error, created_by, updated_by)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		job.JobID, job.SourceID, job.ModelID, job.JobType, job.JobStatus, job.IdempotencyKey, job.OperationID, job.WorkflowExecutionID,
		job.RuntimeActorMOIUserID, job.RuntimeEffectiveRoleID, job.RuntimeIsWorkspaceOwner,
		job.SourceFileID, job.KBFileID, job.SourceTableID, job.KBTableID, job.RetryCount, job.NextRetryAt, job.Error, actor, actor).Error
}

func (s *semanticModelService) upsertKnowledgeBaseSourceJobRun(ctx context.Context, job *KnowledgeBaseSourceJobRun, actor string) error {
	if job == nil {
		return fmt.Errorf("knowledge base source job run is required")
	}
	db := ctxutil.TenantDBFrom(ctx)
	if db == nil {
		return fmt.Errorf("tenant db is required")
	}
	return upsertKnowledgeBaseSourceJobRunWithTx(db.WithContext(ctx), job, actor)
}

func upsertKnowledgeBaseSourceJobRunWithTx(tx *gorm.DB, job *KnowledgeBaseSourceJobRun, actor string) error {
	if job == nil {
		return fmt.Errorf("knowledge base source job run is required")
	}
	res := tx.Exec(`UPDATE knowledge_base_source_job_runs
		SET runtime_actor_moi_user_id = ?, runtime_effective_role_id = ?, runtime_is_workspace_owner = ?, job_status = ?, operation_id = ?, workflow_execution_id = ?, source_file_id = ?, kb_file_id = ?, source_table_id = ?, kb_table_id = ?, retry_count = ?, next_retry_at = ?, error = ?, updated_by = ?
		WHERE job_id = ?`,
		job.RuntimeActorMOIUserID, job.RuntimeEffectiveRoleID, job.RuntimeIsWorkspaceOwner, job.JobStatus, job.OperationID, job.WorkflowExecutionID,
		job.SourceFileID, job.KBFileID, job.SourceTableID, job.KBTableID,
		job.RetryCount, job.NextRetryAt, job.Error, actor, job.JobID)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		insert := tx.Exec(`INSERT INTO knowledge_base_source_job_runs
			(job_id, source_id, model_id, job_type, job_status, idempotency_key, operation_id, workflow_execution_id, runtime_actor_moi_user_id, runtime_effective_role_id, runtime_is_workspace_owner, source_file_id, kb_file_id, source_table_id, kb_table_id, retry_count, next_retry_at, error, created_by, updated_by)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			job.JobID, job.SourceID, job.ModelID, job.JobType, job.JobStatus, job.IdempotencyKey, job.OperationID, job.WorkflowExecutionID,
			job.RuntimeActorMOIUserID, job.RuntimeEffectiveRoleID, job.RuntimeIsWorkspaceOwner,
			job.SourceFileID, job.KBFileID, job.SourceTableID, job.KBTableID, job.RetryCount, job.NextRetryAt, job.Error, actor, actor)
		if insert.Error != nil && isDuplicateEntryError(insert.Error) {
			return nil
		}
		return insert.Error
	}
	return nil
}

func (s *semanticModelService) markKnowledgeBaseSourceJobRunning(ctx context.Context, jobID, actor string) error {
	db := ctxutil.TenantDBFrom(ctx)
	if db == nil {
		return fmt.Errorf("tenant db is required")
	}
	return db.WithContext(ctx).Exec(`UPDATE knowledge_base_source_job_runs
		SET job_status = ?, updated_by = ?, updated_at = CURRENT_TIMESTAMP
		WHERE job_id = ? AND job_status IN (?, ?)`,
		kbSourceJobRunning, actor, jobID, kbSourceJobQueued, kbSourceJobRunning).Error
}

func (s *semanticModelService) claimKnowledgeBaseSourceJobRunning(ctx context.Context, jobID, actor string) (bool, error) {
	db := ctxutil.TenantDBFrom(ctx)
	if db == nil {
		return false, fmt.Errorf("tenant db is required")
	}
	res := db.WithContext(ctx).Exec(`UPDATE knowledge_base_source_job_runs
		SET job_status = ?, updated_by = ?, updated_at = CURRENT_TIMESTAMP
		WHERE job_id = ? AND (job_status IN (?, ?) OR (job_status = ? AND updated_at < DATE_SUB(CURRENT_TIMESTAMP, INTERVAL ? SECOND)))`,
		kbSourceJobRunning, actor, jobID, kbSourceJobPending, kbSourceJobQueued,
		kbSourceJobRunning, int(kbSourceJobClaimLease/time.Second))
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected == 1, nil
}

// releaseKnowledgeBaseFileBindJobClaims makes a synchronous bind-path failure
// (typically local-file AddFiles) immediately retryable. The running-state CAS
// avoids overwriting terminal jobs; the claim lease remains the fallback when
// this release cannot run. Not attempt fencing for work past the claim lease.
func (s *semanticModelService) releaseKnowledgeBaseFileBindJobClaims(ctx context.Context, actor string, jobIDs ...string) error {
	db := ctxutil.TenantDBFrom(ctx)
	if db == nil {
		return fmt.Errorf("tenant db is required")
	}
	jobIDs = compactUniqueStrings(jobIDs)
	if len(jobIDs) == 0 {
		return nil
	}
	placeholders := make([]string, len(jobIDs))
	args := []any{kbSourceJobQueued, actor, kbSourceJobRunning}
	for i, jobID := range jobIDs {
		placeholders[i] = "?"
		args = append(args, jobID)
	}
	return db.WithContext(ctx).Exec(`UPDATE knowledge_base_source_job_runs
		SET job_status = ?, updated_by = ?, updated_at = CURRENT_TIMESTAMP
		WHERE job_status = ? AND job_id IN (`+strings.Join(placeholders, ",")+`)`, args...).Error
}

func (s *semanticModelService) updateKnowledgeBaseSourceJobRunSucceeded(ctx context.Context, job *KnowledgeBaseSourceJobRun, actor string) error {
	db := ctxutil.TenantDBFrom(ctx)
	if db == nil {
		return fmt.Errorf("tenant db is required")
	}
	return updateKnowledgeBaseSourceJobRunSucceededWithTx(db.WithContext(ctx), job, actor)
}

func updateKnowledgeBaseSourceJobRunSucceededWithTx(tx *gorm.DB, job *KnowledgeBaseSourceJobRun, actor string) error {
	return tx.Exec(`UPDATE knowledge_base_source_job_runs
		SET job_status = ?, operation_id = ?, workflow_execution_id = ?, kb_file_id = ?, kb_table_id = ?, error = NULL, updated_by = ?
		WHERE job_id = ?`,
		job.JobStatus, job.OperationID, job.WorkflowExecutionID, job.KBFileID, job.KBTableID, actor, job.JobID).Error
}

func (s *semanticModelService) markKnowledgeBaseSourceJobFailed(ctx context.Context, jobID, msg, actor string) error {
	db := ctxutil.TenantDBFrom(ctx)
	if db == nil {
		return fmt.Errorf("tenant db is required")
	}
	return db.WithContext(ctx).Exec(`UPDATE knowledge_base_source_job_runs
		SET job_status = ?, error = ?, updated_by = ?
		WHERE job_id = ?`,
		kbSourceJobFailed, msg, actor, jobID).Error
}

// finishKnowledgeBaseFileBindJob atomically commits a running copy/load job and
// its source. Catalog copy only advances job state (operation_id stays the
// stable protocol string catalog_file_link:<file_id>; no volume AddFiles).
// Local load may also refresh source file/volume fields after KB raw AddFiles.
// Removed sources and jobs no longer in running state are idempotent no-ops.
func (s *semanticModelService) finishKnowledgeBaseFileBindJob(ctx context.Context, source *KnowledgeBaseSourceRecord, job *KnowledgeBaseSourceJobRun, fileID string, cause error, actor string) (bool, error) {
	if source == nil {
		return false, fmt.Errorf("knowledge base source is required")
	}
	if job == nil {
		return false, fmt.Errorf("knowledge base source job run is required")
	}
	db := ctxutil.TenantDBFrom(ctx)
	if db == nil {
		return false, fmt.Errorf("tenant db is required")
	}
	sourceFileID := firstNonEmptySegmentString(
		ptrValue(source.SourceFileID),
		ptrValue(job.SourceFileID),
		fileID,
	)
	finished := false
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var status string
		if err := tx.Raw(`SELECT status FROM knowledge_base_sources WHERE source_id = ? FOR UPDATE`, source.SourceID).Row().Scan(&status); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return knowledgeBaseSourceNotFoundError()
			}
			return err
		}
		if status == kbSourceStatusRemoved {
			return nil
		}

		// Protocol strings are historical; see kbJobOp*Prefix constants.
		operationID := kbJobOpCatalogFileBindPrefix + fileID
		if job.JobType == kbJobTypeLoad {
			operationID = kbJobOpLocalFileBindPrefix + fileID
		}
		var jobResult *gorm.DB
		if cause != nil {
			jobResult = tx.Exec(`UPDATE knowledge_base_source_job_runs
				SET job_status = ?, error = ?, updated_by = ?
				WHERE job_id = ? AND source_id = ? AND job_status = ?`,
				kbSourceJobFailed, cause.Error(), actor, job.JobID, source.SourceID, kbSourceJobRunning)
		} else {
			jobResult = tx.Exec(`UPDATE knowledge_base_source_job_runs
				SET job_status = ?, operation_id = ?, source_file_id = ?, kb_file_id = ?, error = NULL, updated_by = ?
				WHERE job_id = ? AND source_id = ? AND job_status = ?`,
				kbSourceJobSucceeded, operationID, sourceFileID, fileID, actor, job.JobID, source.SourceID, kbSourceJobRunning)
		}
		if jobResult.Error != nil {
			return jobResult.Error
		}
		if jobResult.RowsAffected == 0 {
			return nil
		}

		if cause != nil {
			if err := tx.Exec(`UPDATE knowledge_base_sources
				SET status = ?, error = ?, updated_by = ?
				WHERE source_id = ? AND status <> ?`,
				kbSourceStatusFailed, cause.Error(), actor, source.SourceID, kbSourceStatusRemoved).Error; err != nil {
				return err
			}
		} else if job.JobType == kbJobTypeLoad {
			if err := tx.Exec(`UPDATE knowledge_base_sources
				SET source_file_id = ?, kb_file_id = ?, raw_volume_id = ?, display_name = ?, error = NULL, updated_by = ?
				WHERE source_id = ? AND status <> ?`,
				sourceFileID, fileID, source.RawVolumeID, source.DisplayName, actor, source.SourceID, kbSourceStatusRemoved).Error; err != nil {
				return err
			}
		}
		finished = true
		return nil
	})
	if err != nil || !finished {
		return finished, err
	}
	if cause != nil {
		source.Status = kbSourceStatusFailed
		source.Error = stringPtr(cause.Error())
		job.JobStatus = kbSourceJobFailed
		job.Error = stringPtr(cause.Error())
		return true, nil
	}
	job.SourceFileID = stringPtr(sourceFileID)
	job.KBFileID = stringPtr(fileID)
	if job.JobType == kbJobTypeLoad {
		source.SourceFileID = stringPtr(sourceFileID)
		source.KBFileID = stringPtr(fileID)
		source.Error = nil
		job.OperationID = stringPtr(kbJobOpLocalFileBindPrefix + fileID)
	} else {
		job.OperationID = stringPtr(kbJobOpCatalogFileBindPrefix + fileID)
	}
	job.JobStatus = kbSourceJobSucceeded
	job.Error = nil
	return true, nil
}

func (s *semanticModelService) markKnowledgeBaseSourceJobFailedChecked(ctx context.Context, jobID, actor string) error {
	db := ctxutil.TenantDBFrom(ctx)
	if db == nil {
		return fmt.Errorf("tenant db is required")
	}
	return db.WithContext(ctx).Exec(`UPDATE knowledge_base_source_job_runs
		SET updated_by = ?, updated_at = CURRENT_TIMESTAMP
		WHERE job_id = ? AND job_status = ?`, actor, jobID, kbSourceJobFailed).Error
}

func (s *semanticModelService) listKnowledgeBaseSourceJobRuns(ctx context.Context, modelID int64) ([]KnowledgeBaseSourceJobRun, error) {
	db := ctxutil.TenantDBFrom(ctx)
	if db == nil {
		return []KnowledgeBaseSourceJobRun{}, nil
	}
	rows, err := db.WithContext(ctx).Raw(`SELECT job_id, source_id, model_id, job_type, job_status, idempotency_key, operation_id, workflow_execution_id, runtime_actor_moi_user_id, runtime_effective_role_id, runtime_is_workspace_owner, source_file_id, kb_file_id, source_table_id, kb_table_id, retry_count, next_retry_at, error, UNIX_TIMESTAMP(created_at), UNIX_TIMESTAMP(updated_at)
		FROM knowledge_base_source_job_runs
		WHERE model_id = ?
		ORDER BY created_at ASC, job_id ASC`, modelID).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]KnowledgeBaseSourceJobRun, 0)
	for rows.Next() {
		var job KnowledgeBaseSourceJobRun
		if err := rows.Scan(&job.JobID, &job.SourceID, &job.ModelID, &job.JobType, &job.JobStatus, &job.IdempotencyKey, &job.OperationID, &job.WorkflowExecutionID, &job.RuntimeActorMOIUserID, &job.RuntimeEffectiveRoleID, &job.RuntimeIsWorkspaceOwner, &job.SourceFileID, &job.KBFileID, &job.SourceTableID, &job.KBTableID, &job.RetryCount, &job.NextRetryAt, &job.Error, &job.CreatedAt, &job.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, job)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *semanticModelService) listFastKnowledgeBaseFileBindJobs(ctx context.Context, modelID int64, limit int) ([]KnowledgeBaseSourceJobRun, error) {
	db := ctxutil.TenantDBFrom(ctx)
	if db == nil || limit <= 0 {
		return []KnowledgeBaseSourceJobRun{}, nil
	}
	rows, err := db.WithContext(ctx).Raw(`SELECT jr.job_id, jr.source_id, jr.model_id, jr.job_type, jr.job_status, jr.idempotency_key, jr.operation_id, jr.workflow_execution_id, jr.runtime_actor_moi_user_id, jr.runtime_effective_role_id, jr.runtime_is_workspace_owner, jr.source_file_id, jr.kb_file_id, jr.source_table_id, jr.kb_table_id, jr.retry_count, jr.next_retry_at, jr.error, UNIX_TIMESTAMP(jr.created_at), UNIX_TIMESTAMP(jr.updated_at)
		FROM knowledge_base_source_job_runs jr
		INNER JOIN knowledge_base_sources kbs ON kbs.model_id = jr.model_id AND kbs.source_id = jr.source_id
		LEFT JOIN knowledge_base_raw_volumes krv ON krv.model_id = kbs.model_id AND krv.raw_volume_id = kbs.raw_volume_id
		WHERE jr.model_id = ? AND jr.job_status IN (?, ?, ?)
		  AND kbs.status NOT IN (?, ?)
		  AND (jr.job_type = ? OR (jr.job_type = ? AND kbs.source_type = ? AND COALESCE(krv.raw_kind, '') <> ?))
		ORDER BY jr.created_at ASC, jr.job_id ASC
		LIMIT ?`, modelID, kbSourceJobPending, kbSourceJobQueued, kbSourceJobRunning, kbSourceStatusFailed, kbSourceStatusRemoved, kbJobTypeCopy, kbJobTypeLoad, kbSourceTypeLocalFile, kbRawKindStructured, limit).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanKnowledgeBaseSourceJobRuns(rows)
}

func (s *semanticModelService) listKnowledgeBaseSourceJobRunsByStatuses(ctx context.Context, modelID int64, jobType string, statuses []string, limit int) ([]KnowledgeBaseSourceJobRun, error) {
	db := ctxutil.TenantDBFrom(ctx)
	if db == nil || len(statuses) == 0 || limit <= 0 {
		return []KnowledgeBaseSourceJobRun{}, nil
	}
	placeholders := make([]string, len(statuses))
	args := []any{modelID, jobType}
	for i, status := range statuses {
		placeholders[i] = "?"
		args = append(args, status)
	}
	args = append(args, kbSourceStatusFailed, kbSourceStatusRemoved, limit)
	rows, err := db.WithContext(ctx).Raw(`SELECT jr.job_id, jr.source_id, jr.model_id, jr.job_type, jr.job_status, jr.idempotency_key, jr.operation_id, jr.workflow_execution_id, jr.runtime_actor_moi_user_id, jr.runtime_effective_role_id, jr.runtime_is_workspace_owner, jr.source_file_id, jr.kb_file_id, jr.source_table_id, jr.kb_table_id, jr.retry_count, jr.next_retry_at, jr.error, UNIX_TIMESTAMP(jr.created_at), UNIX_TIMESTAMP(jr.updated_at)
		FROM knowledge_base_source_job_runs jr
		INNER JOIN knowledge_base_sources kbs ON kbs.model_id = jr.model_id AND kbs.source_id = jr.source_id
		WHERE jr.model_id = ? AND jr.job_type = ? AND jr.job_status IN (`+strings.Join(placeholders, ",")+`)
		  AND kbs.status NOT IN (?, ?)
		ORDER BY jr.created_at ASC, jr.job_id ASC
		LIMIT ?`, args...).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanKnowledgeBaseSourceJobRuns(rows)
}

func (s *semanticModelService) listRAGIngestKnowledgeBaseSourceJobRuns(ctx context.Context, modelID int64, statuses []string, limit int, oldestCheckedFirst bool) ([]KnowledgeBaseSourceJobRun, error) {
	db := ctxutil.TenantDBFrom(ctx)
	if db == nil || len(statuses) == 0 || limit <= 0 {
		return []KnowledgeBaseSourceJobRun{}, nil
	}
	placeholders := make([]string, len(statuses))
	args := []any{modelID, kbJobTypeRAGIngest}
	for i, status := range statuses {
		placeholders[i] = "?"
		args = append(args, status)
	}
	args = append(args, kbSourceStatusRemoved, kbSourceStatusFailed, kbSourceJobFailed, kbSourceJobSucceeded, kbSourceStatusSucceeded, limit)
	orderBy := "jr.created_at ASC, jr.job_id ASC"
	if oldestCheckedFirst {
		orderBy = "jr.updated_at ASC, jr.job_id ASC"
	}
	rows, err := db.WithContext(ctx).Raw(`SELECT jr.job_id, jr.source_id, jr.model_id, jr.job_type, jr.job_status, jr.idempotency_key, jr.operation_id, jr.workflow_execution_id, jr.runtime_actor_moi_user_id, jr.runtime_effective_role_id, jr.runtime_is_workspace_owner, jr.source_file_id, jr.kb_file_id, jr.source_table_id, jr.kb_table_id, jr.retry_count, jr.next_retry_at, jr.error, UNIX_TIMESTAMP(jr.created_at), UNIX_TIMESTAMP(jr.updated_at)
		FROM knowledge_base_source_job_runs jr
		INNER JOIN knowledge_base_sources kbs ON kbs.model_id = jr.model_id AND kbs.source_id = jr.source_id
		WHERE jr.model_id = ? AND jr.job_type = ? AND jr.job_status IN (`+strings.Join(placeholders, ",")+`)
		  AND kbs.status <> ?
		  AND (kbs.status <> ? OR jr.job_status = ?)
		  AND (jr.job_status <> ? OR kbs.status <> ? OR kbs.kb_file_id IS NULL OR kbs.kb_file_id = '' OR kbs.segment_version_id IS NULL OR kbs.segment_version_id = '' OR kbs.index_version IS NULL OR kbs.index_version <= 0)
		ORDER BY `+orderBy+`
		LIMIT ?`, args...).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanKnowledgeBaseSourceJobRuns(rows)
}

func scanKnowledgeBaseSourceJobRuns(rows *sql.Rows) ([]KnowledgeBaseSourceJobRun, error) {
	out := make([]KnowledgeBaseSourceJobRun, 0)
	for rows.Next() {
		var job KnowledgeBaseSourceJobRun
		if err := rows.Scan(&job.JobID, &job.SourceID, &job.ModelID, &job.JobType, &job.JobStatus, &job.IdempotencyKey, &job.OperationID, &job.WorkflowExecutionID, &job.RuntimeActorMOIUserID, &job.RuntimeEffectiveRoleID, &job.RuntimeIsWorkspaceOwner, &job.SourceFileID, &job.KBFileID, &job.SourceTableID, &job.KBTableID, &job.RetryCount, &job.NextRetryAt, &job.Error, &job.CreatedAt, &job.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, job)
	}
	return out, rows.Err()
}

func (s *semanticModelService) listPendingKnowledgeBaseSourceJobRuns(ctx context.Context, modelID int64, jobType string, includePending bool, limit int) ([]KnowledgeBaseSourceJobRun, error) {
	db := ctxutil.TenantDBFrom(ctx)
	if db == nil {
		return []KnowledgeBaseSourceJobRun{}, nil
	}
	if limit <= 0 {
		return []KnowledgeBaseSourceJobRun{}, nil
	}
	statuses := []string{kbSourceJobQueued, kbSourceJobRunning}
	if includePending {
		statuses = []string{kbSourceJobPending, kbSourceJobQueued, kbSourceJobRunning}
	}
	placeholders := make([]string, len(statuses))
	args := []any{modelID, jobType}
	for i, status := range statuses {
		placeholders[i] = "?"
		args = append(args, status)
	}
	query := `SELECT jr.job_id, jr.source_id, jr.model_id, jr.job_type, jr.job_status, jr.idempotency_key, jr.operation_id, jr.workflow_execution_id, jr.runtime_actor_moi_user_id, jr.runtime_effective_role_id, jr.runtime_is_workspace_owner, jr.source_file_id, jr.kb_file_id, jr.source_table_id, jr.kb_table_id, jr.retry_count, jr.next_retry_at, jr.error, UNIX_TIMESTAMP(jr.created_at), UNIX_TIMESTAMP(jr.updated_at)
		FROM knowledge_base_source_job_runs jr
		INNER JOIN knowledge_base_sources kbs ON kbs.model_id = jr.model_id AND kbs.source_id = jr.source_id
		WHERE jr.model_id = ? AND jr.job_type = ? AND jr.job_status IN (` + strings.Join(placeholders, ",") + `)
		  AND kbs.status NOT IN ('` + kbSourceStatusFailed + `', '` + kbSourceStatusRemoved + `')
		ORDER BY jr.created_at ASC, jr.job_id ASC
		LIMIT ?`
	args = append(args, limit)
	if jobType == kbJobTypeRAGIngest {
		query = `SELECT jr.job_id, jr.source_id, jr.model_id, jr.job_type, jr.job_status, jr.idempotency_key, jr.operation_id, jr.workflow_execution_id, jr.runtime_actor_moi_user_id, jr.runtime_effective_role_id, jr.runtime_is_workspace_owner, jr.source_file_id, jr.kb_file_id, jr.source_table_id, jr.kb_table_id, jr.retry_count, jr.next_retry_at, jr.error, UNIX_TIMESTAMP(jr.created_at), UNIX_TIMESTAMP(jr.updated_at)
			FROM knowledge_base_source_job_runs jr
			INNER JOIN knowledge_base_sources kbs ON kbs.model_id = jr.model_id AND kbs.source_id = jr.source_id
			WHERE jr.model_id = ? AND jr.job_type = ? AND jr.job_status IN (?, ?, ?, ?, ?)
			  AND kbs.status <> '` + kbSourceStatusRemoved + `'
			  AND (jr.job_status <> '` + kbSourceJobSucceeded + `' OR kbs.status <> '` + kbSourceStatusSucceeded + `' OR kbs.kb_file_id IS NULL OR kbs.kb_file_id = '' OR kbs.segment_version_id IS NULL OR kbs.segment_version_id = '' OR kbs.index_version IS NULL)
			ORDER BY jr.created_at ASC, jr.job_id ASC
			LIMIT ?`
		args = []any{modelID, jobType, kbSourceJobPending, kbSourceJobQueued, kbSourceJobRunning, kbSourceJobSucceeded, kbSourceJobFailed, limit}
	}
	rows, err := db.WithContext(ctx).Raw(query, args...).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]KnowledgeBaseSourceJobRun, 0)
	for rows.Next() {
		var job KnowledgeBaseSourceJobRun
		if err := rows.Scan(&job.JobID, &job.SourceID, &job.ModelID, &job.JobType, &job.JobStatus, &job.IdempotencyKey, &job.OperationID, &job.WorkflowExecutionID, &job.RuntimeActorMOIUserID, &job.RuntimeEffectiveRoleID, &job.RuntimeIsWorkspaceOwner, &job.SourceFileID, &job.KBFileID, &job.SourceTableID, &job.KBTableID, &job.RetryCount, &job.NextRetryAt, &job.Error, &job.CreatedAt, &job.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, job)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *semanticModelService) listStructuredLoadKnowledgeBaseSourceJobRuns(ctx context.Context, modelID int64, limit int) ([]KnowledgeBaseSourceJobRun, error) {
	db := ctxutil.TenantDBFrom(ctx)
	if db == nil {
		return []KnowledgeBaseSourceJobRun{}, nil
	}
	if limit <= 0 {
		return []KnowledgeBaseSourceJobRun{}, nil
	}
	rows, err := db.WithContext(ctx).Raw(`SELECT jr.job_id, jr.source_id, jr.model_id, jr.job_type, jr.job_status, jr.idempotency_key, jr.operation_id, jr.workflow_execution_id, jr.runtime_actor_moi_user_id, jr.runtime_effective_role_id, jr.runtime_is_workspace_owner, jr.source_file_id, jr.kb_file_id, jr.source_table_id, jr.kb_table_id, jr.retry_count, jr.next_retry_at, jr.error, UNIX_TIMESTAMP(jr.created_at), UNIX_TIMESTAMP(jr.updated_at)
		FROM knowledge_base_source_job_runs jr
		INNER JOIN knowledge_base_sources kbs ON kbs.model_id = jr.model_id AND kbs.source_id = jr.source_id
		LEFT JOIN knowledge_base_raw_volumes krv ON krv.model_id = kbs.model_id AND krv.raw_volume_id = kbs.raw_volume_id
		WHERE jr.model_id = ? AND jr.job_type = ? AND jr.job_status IN (?, ?, ?)
		  AND kbs.status NOT IN ('`+kbSourceStatusFailed+`', ?)
		  AND (kbs.source_type = ? OR (kbs.source_type = ? AND COALESCE(krv.raw_kind, ?) <> ''))
		  AND (jr.job_status <> '`+kbSourceJobSucceeded+`' OR kbs.status <> '`+kbSourceStatusSucceeded+`' OR kbs.kb_table_id IS NULL OR kbs.kb_table_id <= 0 OR kbs.db_name IS NULL OR kbs.db_name = '' OR kbs.table_name IS NULL OR kbs.table_name = '')
		ORDER BY jr.created_at ASC, jr.job_id ASC
		LIMIT ?`,
		modelID, kbJobTypeLoad, kbSourceJobQueued, kbSourceJobRunning, kbSourceJobSucceeded,
		kbSourceStatusRemoved, kbSourceTypeCatalogTable, kbSourceTypeLocalFile, kbRawKindStructured, limit).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]KnowledgeBaseSourceJobRun, 0)
	for rows.Next() {
		var job KnowledgeBaseSourceJobRun
		if err := rows.Scan(&job.JobID, &job.SourceID, &job.ModelID, &job.JobType, &job.JobStatus, &job.IdempotencyKey, &job.OperationID, &job.WorkflowExecutionID, &job.RuntimeActorMOIUserID, &job.RuntimeEffectiveRoleID, &job.RuntimeIsWorkspaceOwner, &job.SourceFileID, &job.KBFileID, &job.SourceTableID, &job.KBTableID, &job.RetryCount, &job.NextRetryAt, &job.Error, &job.CreatedAt, &job.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, job)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *semanticModelService) listStructuredLoadKnowledgeBaseSourceJobRunsByStatuses(ctx context.Context, modelID int64, statuses []string, limit int, oldestCheckedFirst bool) ([]KnowledgeBaseSourceJobRun, error) {
	db := ctxutil.TenantDBFrom(ctx)
	if db == nil || len(statuses) == 0 || limit <= 0 {
		return []KnowledgeBaseSourceJobRun{}, nil
	}
	placeholders := make([]string, len(statuses))
	args := []any{modelID, kbJobTypeLoad}
	for i, status := range statuses {
		placeholders[i] = "?"
		args = append(args, status)
	}
	args = append(args, kbSourceStatusRemoved, kbSourceTypeCatalogTable, kbSourceTypeLocalFile, kbRawKindStructured, kbSourceJobSucceeded, kbSourceStatusSucceeded, limit)
	orderBy := "jr.created_at ASC, jr.job_id ASC"
	if oldestCheckedFirst {
		orderBy = "jr.updated_at ASC, jr.job_id ASC"
	}
	rows, err := db.WithContext(ctx).Raw(`SELECT jr.job_id, jr.source_id, jr.model_id, jr.job_type, jr.job_status, jr.idempotency_key, jr.operation_id, jr.workflow_execution_id, jr.runtime_actor_moi_user_id, jr.runtime_effective_role_id, jr.runtime_is_workspace_owner, jr.source_file_id, jr.kb_file_id, jr.source_table_id, jr.kb_table_id, jr.retry_count, jr.next_retry_at, jr.error, UNIX_TIMESTAMP(jr.created_at), UNIX_TIMESTAMP(jr.updated_at)
		FROM knowledge_base_source_job_runs jr
		INNER JOIN knowledge_base_sources kbs ON kbs.model_id = jr.model_id AND kbs.source_id = jr.source_id
		LEFT JOIN knowledge_base_raw_volumes krv ON krv.model_id = kbs.model_id AND krv.raw_volume_id = kbs.raw_volume_id
		WHERE jr.model_id = ? AND jr.job_type = ? AND jr.job_status IN (`+strings.Join(placeholders, ",")+`)
		  AND kbs.status NOT IN ('`+kbSourceStatusFailed+`', ?)
		  AND (kbs.source_type = ? OR (kbs.source_type = ? AND COALESCE(krv.raw_kind, ?) <> ''))
		  AND (jr.job_status <> ? OR kbs.status <> ? OR kbs.kb_table_id IS NULL OR kbs.kb_table_id <= 0 OR kbs.db_name IS NULL OR kbs.db_name = '' OR kbs.table_name IS NULL OR kbs.table_name = '')
		ORDER BY `+orderBy+`
		LIMIT ?`, args...).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanKnowledgeBaseSourceJobRuns(rows)
}
