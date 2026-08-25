package session

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/matrixorigin/matrixflow/moi-backend/pkg/ctxutil"
	"github.com/matrixorigin/matrixflow/moi-backend/pkg/model"

	moi "github.com/matrixflow/moi-core/go-sdk"
	"gorm.io/gorm"
)

type createKnowledgeBaseSourcesResult struct {
	records          []KnowledgeBaseSourceRecord
	jobs             []KnowledgeBaseSourceJobRun
	cleanupRecords   []KnowledgeBaseSourceRecord
	fileIDs          []string
	tables           []semanticModelTableSource
	localUploadPlans []knowledgeBaseLocalUploadPlan
}

type knowledgeBaseLocalUploadPlan struct {
	recordIndex int
	loadJobIdx  int
	ragJobIdx   int
	request     CreateSemanticModelSourceRequest
	rawVolumeID int64
}

const kbLocalStructuredUploadOperationPrefix = "local_structured_upload:"

type knowledgeBaseLocalStructuredUploadOperation struct {
	FileName    string `json:"file_name"`
	FileID      string `json:"file_id"`
	TableConfig string `json:"table_config"`
}

func (s *semanticModelService) createKnowledgeBaseSourceMetadataIntents(ctx context.Context, c *moi.Client, wsID string, modelID int64, domain *KnowledgeBaseDataDomain, requests []CreateSemanticModelSourceRequest, actor string, reuseExisting bool) (createKnowledgeBaseSourcesResult, error) {
	result := createKnowledgeBaseSourcesResult{
		records:        make([]KnowledgeBaseSourceRecord, 0, len(requests)),
		jobs:           make([]KnowledgeBaseSourceJobRun, 0, len(requests)*2),
		cleanupRecords: make([]KnowledgeBaseSourceRecord, 0, len(requests)),
		fileIDs:        make([]string, 0, len(requests)),
	}
	db := ctxutil.TenantDBFrom(ctx)
	if db == nil {
		return result, fmt.Errorf("tenant db is required")
	}
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txCtx := ctxutil.WithTenantDB(ctx, tx)
		txResult, err := s.createKnowledgeBaseSourceMetadataIntentsInTx(txCtx, c, wsID, modelID, domain, requests, actor, reuseExisting)
		if err != nil {
			return err
		}
		result = txResult
		return nil
	})
	return result, err
}

func (s *semanticModelService) createKnowledgeBaseSourceMetadataIntentsInTx(ctx context.Context, c *moi.Client, wsID string, modelID int64, domain *KnowledgeBaseDataDomain, requests []CreateSemanticModelSourceRequest, actor string, reuseExisting bool) (createKnowledgeBaseSourcesResult, error) {
	result := createKnowledgeBaseSourcesResult{
		records:        make([]KnowledgeBaseSourceRecord, 0, len(requests)),
		jobs:           make([]KnowledgeBaseSourceJobRun, 0, len(requests)*2),
		cleanupRecords: make([]KnowledgeBaseSourceRecord, 0, len(requests)),
		fileIDs:        make([]string, 0, len(requests)),
	}
	tablesByDB := map[string][]string{}
	for _, req := range requests {
		switch req.SourceType {
		case kbSourceTypeLocalFile:
			record, jobs, plan, err := s.createLocalFileSourceIntent(ctx, wsID, modelID, domain, req, actor, reuseExisting)
			if err != nil {
				return result, err
			}
			recordIndex := -1
			if record != nil {
				result.records = append(result.records, *record)
				recordIndex = len(result.records) - 1
				if len(jobs) > 0 {
					result.cleanupRecords = append(result.cleanupRecords, *record)
				}
			}
			loadJobIdx := len(result.jobs)
			result.jobs = append(result.jobs, jobs...)
			if recordIndex >= 0 && req.UploadKind == kbLocalUploadKindStructured && len(jobs) > 0 {
				plan.recordIndex = recordIndex
				plan.loadJobIdx = loadJobIdx
				plan.ragJobIdx = -1
				result.localUploadPlans = append(result.localUploadPlans, plan)
			}
		case kbSourceTypeCatalogFile:
			record, jobs, fileID, err := s.appendCatalogFileSourceIntent(ctx, wsID, modelID, domain, req, actor, reuseExisting)
			if err != nil {
				return result, err
			}
			if record != nil {
				result.records = append(result.records, *record)
				if len(jobs) > 0 {
					result.cleanupRecords = append(result.cleanupRecords, *record)
				}
			}
			result.jobs = append(result.jobs, jobs...)
			if fileID != "" {
				result.fileIDs = append(result.fileIDs, fileID)
			}
		case kbSourceTypeCatalogTable:
			record, table, err := s.appendCatalogTableSource(ctx, c, wsID, modelID, req, actor, reuseExisting)
			if err != nil {
				return result, err
			}
			if record != nil {
				result.records = append(result.records, *record)
			}
			if table != nil {
				tablesByDB[table.DBName] = append(tablesByDB[table.DBName], table.TableNames...)
			}
		default:
			return result, semanticModelSourceTypeUnsupportedError(req.SourceType)
		}
	}
	result.tables = mapToSemanticModelTables(tablesByDB)
	return result, nil
}

func (s *semanticModelService) createLocalFileSourceIntent(ctx context.Context, wsID string, modelID int64, domain *KnowledgeBaseDataDomain, req CreateSemanticModelSourceRequest, actor string, reuseExisting bool) (*KnowledgeBaseSourceRecord, []KnowledgeBaseSourceJobRun, knowledgeBaseLocalUploadPlan, error) {
	if req.UploadKind == kbLocalUploadKindStructured {
		normalized, err := normalizeKnowledgeBaseStructuredTableConfig(req.TableConfig, domain.DatabaseID)
		if err != nil {
			return nil, nil, knowledgeBaseLocalUploadPlan{}, err
		}
		req.TableConfig = normalized
	}
	rawVolumeID := domain.RawVolumeID
	if req.UploadKind != kbLocalUploadKindStructured {
		var err error
		rawVolumeID, err = s.ensureKnowledgeBaseRawVolume(ctx, domain, rawKindForLocalFile(req), actor)
		if err != nil {
			return nil, nil, knowledgeBaseLocalUploadPlan{}, fmt.Errorf("ensure local file raw volume %s: %w", req.FileName, err)
		}
	}
	enabled := true
	sourceType := kbSourceTypeLocalFile
	displayName := req.FileName
	if req.UploadKind == kbLocalUploadKindStructured {
		sourceType = kbSourceTypeCatalogTable
		displayName = knowledgeBaseStructuredUploadDisplayName(req.TableConfig, req.FileName)
	}
	source := &KnowledgeBaseSourceRecord{
		SourceID:          stableID("kb-source", modelID, kbSourceTypeLocalFile, req.FileID),
		ModelID:           modelID,
		CatalogID:         domain.CatalogID,
		DatabaseID:        domain.DatabaseID,
		RawVolumeID:       rawVolumeID,
		ProcessedVolumeID: domain.ProcessedVolumeID,
		SourceType:        sourceType,
		DisplayName:       stringPtr(displayName),
		Status:            kbSourceStatusPending,
		Enabled:           &enabled,
	}
	if req.UploadKind != kbLocalUploadKindStructured {
		source.SourceFileID = stringPtr(req.FileID)
		source.KBFileID = stringPtr(req.FileID)
	}
	reactivateRemoved := false
	if reuseExisting {
		existing, found, err := s.findKnowledgeBaseSourceByID(ctx, modelID, source.SourceID)
		if err != nil {
			return nil, nil, knowledgeBaseLocalUploadPlan{}, fmt.Errorf("find existing local file source: %w", err)
		}
		if found {
			if !isKnowledgeBaseSourceRemoved(existing) {
				return &existing, nil, knowledgeBaseLocalUploadPlan{}, nil
			}
			source.SourceID = existing.SourceID
			reactivateRemoved = true
		}
	}
	if reactivateRemoved {
		if err := s.reactivateKnowledgeBaseSource(ctx, source, actor); err != nil {
			return nil, nil, knowledgeBaseLocalUploadPlan{}, fmt.Errorf("reactivate local file source: %w", err)
		}
	} else if err := s.insertKnowledgeBaseSource(ctx, source, actor); err != nil {
		return nil, nil, knowledgeBaseLocalUploadPlan{}, fmt.Errorf("record local file source: %w", err)
	}
	loadJob := newKnowledgeBaseJobRun(source, kbJobTypeLoad, stableID("kb-job", source.SourceID, kbJobTypeLoad), actor)
	loadJob.JobStatus = kbSourceJobQueued
	loadJob.SourceFileID = source.SourceFileID
	loadJob.KBFileID = source.KBFileID
	jobs := []KnowledgeBaseSourceJobRun{loadJob}
	if err := s.writeKnowledgeBaseSourceJobRun(ctx, &jobs[0], actor, reuseExisting); err != nil {
		return nil, nil, knowledgeBaseLocalUploadPlan{}, fmt.Errorf("record local file load job: %w", err)
	}
	if req.UploadKind != kbLocalUploadKindStructured {
		ragJob := newKnowledgeBaseTriggerRAGJob(source, nil, actor)
		jobs = append(jobs, ragJob)
		if err := s.writeKnowledgeBaseSourceJobRun(ctx, &jobs[1], actor, reuseExisting); err != nil {
			return nil, nil, knowledgeBaseLocalUploadPlan{}, fmt.Errorf("record local file parse workflow trigger job: %w", err)
		}
	}
	plan := knowledgeBaseLocalUploadPlan{
		request:     req,
		rawVolumeID: rawVolumeID,
	}
	return source, jobs, plan, nil
}

func (s *semanticModelService) writeKnowledgeBaseSourceJobRun(ctx context.Context, job *KnowledgeBaseSourceJobRun, actor string, reuseExisting bool) error {
	if err := freezeKnowledgeBaseSourceJobRuntimePrincipal(ctx, job); err != nil {
		return err
	}
	if reuseExisting {
		return s.upsertKnowledgeBaseSourceJobRun(ctx, job, actor)
	}
	return s.insertKnowledgeBaseSourceJobRun(ctx, job, actor)
}

// freezeKnowledgeBaseSourceJobRuntimePrincipal freezes the create-time MOI user
// and VerifiedEffectiveRole onto the job. Optional runtime_is_workspace_owner is
// recorded for audit only — deferred rehydrate must not restore privilege-class
// bypass from it. RAG jobs require actor+role; missing identity must fail create
// so reconcile never inherits a callback principal. Historical rows without
// freeze stay fail-closed.
func freezeKnowledgeBaseSourceJobRuntimePrincipal(ctx context.Context, job *KnowledgeBaseSourceJobRun) error {
	if job == nil {
		return fmt.Errorf("knowledge base source job run is required")
	}
	runtimeActorID := strings.TrimSpace(ctxutil.MoiUserIDFrom(ctx))
	roleID := ""
	isOwner := false
	if trusted, ok := ctxutil.CoreIAMRequestFrom(ctx); ok {
		roleID = strings.TrimSpace(trusted.VerifiedEffectiveRoleID)
		// Audit-only snapshot of create-time privilege-class; not used as live
		// PEP bypass on deferred dispatch (see rehydrateKnowledgeBaseRAGJobPrincipal).
		isOwner = trusted.IsWorkspaceOwner
	}
	if runtimeActorID == "" {
		job.RuntimeActorMOIUserID = nil
	} else {
		job.RuntimeActorMOIUserID = stringPtr(runtimeActorID)
	}
	if roleID == "" {
		job.RuntimeEffectiveRoleID = nil
	} else {
		job.RuntimeEffectiveRoleID = stringPtr(roleID)
	}
	job.RuntimeIsWorkspaceOwner = isOwner
	if job.JobType == kbJobTypeRAGIngest {
		if runtimeActorID == "" || roleID == "" {
			return fmt.Errorf("rag ingest job requires runtime actor and verified effective role")
		}
	}
	return nil
}

func (s *semanticModelService) runCreateKnowledgeBaseStructuredLocalImports(ctx context.Context, result *createKnowledgeBaseSourcesResult, actor string) error {
	if result == nil {
		return nil
	}
	for _, plan := range result.localUploadPlans {
		if plan.request.UploadKind != kbLocalUploadKindStructured {
			continue
		}
		if err := s.runCreateStructuredLocalImport(ctx, result, plan, actor); err != nil {
			return err
		}
	}
	return nil
}

func (s *semanticModelService) runCreateStructuredLocalImport(ctx context.Context, result *createKnowledgeBaseSourcesResult, plan knowledgeBaseLocalUploadPlan, actor string) error {
	if plan.request.UploadKind != kbLocalUploadKindStructured {
		return nil
	}

	imported, err := s.localImportService.UploadToVolume(ctx, KnowledgeBaseLocalFileImportParams{
		VolumeID:    plan.rawVolumeID,
		FileName:    plan.request.FileName,
		FileID:      plan.request.FileID,
		UploadKind:  plan.request.UploadKind,
		TableConfig: plan.request.TableConfig,
	})
	if err != nil {
		return s.markCreateKnowledgeBasePlanFailed(ctx, result, plan.recordIndex, []int{plan.loadJobIdx, plan.ragJobIdx}, fmt.Errorf("create local file import task %s: %w", plan.request.FileName, err), actor)
	}
	if imported == nil || imported.TaskID == "" || len(imported.FileIDs) == 0 || imported.FileIDs[0] == "" {
		return s.markCreateKnowledgeBasePlanFailed(ctx, result, plan.recordIndex, []int{plan.loadJobIdx, plan.ragJobIdx}, fmt.Errorf("local file import returned incomplete result"), actor)
	}
	record := &result.records[plan.recordIndex]
	record.RawVolumeID = plan.rawVolumeID
	loadJob := &result.jobs[plan.loadJobIdx]
	loadJob.JobStatus = kbSourceJobRunning
	loadJob.OperationID = stringPtr("import_task:" + imported.TaskID)
	if imported.WorkflowExecutionID != nil && *imported.WorkflowExecutionID != "" {
		loadJob.WorkflowExecutionID = imported.WorkflowExecutionID
	}
	if err := s.upsertKnowledgeBaseSourceJobRun(ctx, loadJob, actor); err != nil {
		return fmt.Errorf("record local file load job result %s: %w", plan.request.FileName, err)
	}
	return nil
}

func addFileToKnowledgeBaseRawVolume(ctx context.Context, c *moi.Client, wsID string, rawVolumeID int64, fileID string) error {
	if c == nil {
		return fmt.Errorf("moi client is required")
	}
	if rawVolumeID <= 0 {
		return fmt.Errorf("raw volume id is required")
	}
	if strings.TrimSpace(fileID) == "" {
		return fmt.Errorf("file_id is required")
	}
	// Local upload path only: attach an already-uploaded file_id to the KB raw
	// volume. AddFiles is idempotent in core (already-attached file_ids are no-ops).
	// Catalog files never call this helper; they stay on the user volume.
	return c.VolumeFiles().AddFiles(ctx, wsID, rawVolumeID, []string{fileID}, moi.WithRequireUnlinked())
}

func validateKnowledgeBaseStructuredTableConfig(raw string, databaseID int64) error {
	_, err := normalizeKnowledgeBaseStructuredTableConfig(raw, databaseID)
	return err
}

func normalizeKnowledgeBaseStructuredTableConfig(raw string, databaseID int64) (string, error) {
	if raw == "" {
		return "", fmt.Errorf("table_config is required for structured local upload")
	}
	var multi model.MultiTableConfig
	if err := json.Unmarshal([]byte(raw), &multi); err != nil {
		return "", fmt.Errorf("parse table_config: %w", err)
	}
	if multi.MultiSheet || len(multi.Tables) > 0 {
		if len(multi.Tables) == 0 {
			return "", fmt.Errorf("parse table_config: multi-sheet tables is empty")
		}
		for i, table := range multi.Tables {
			if table == nil {
				return "", fmt.Errorf("parse table_config: table %d is nil", i)
			}
			if table.DatabaseId == 0 {
				table.DatabaseId = databaseID
			}
			if table.DatabaseId != databaseID {
				return "", fmt.Errorf("table_config.tables[%d].database_id must be knowledge base database %d", i, databaseID)
			}
			if err := validateStructuredTableDefinition(table, fmt.Sprintf("table_config.tables[%d]", i)); err != nil {
				return "", err
			}
		}
		normalized, err := json.Marshal(&multi)
		if err != nil {
			return "", fmt.Errorf("normalize table_config: %w", err)
		}
		return string(normalized), nil
	}
	var single model.TableConfig
	if err := json.Unmarshal([]byte(raw), &single); err != nil {
		return "", fmt.Errorf("parse table_config: %w", err)
	}
	if single.DatabaseId == 0 {
		single.DatabaseId = databaseID
	}
	if single.DatabaseId != databaseID {
		return "", fmt.Errorf("table_config.database_id must be knowledge base database %d", databaseID)
	}
	if err := validateStructuredTableDefinition(&single, "table_config"); err != nil {
		return "", err
	}
	normalized, err := json.Marshal(&single)
	if err != nil {
		return "", fmt.Errorf("normalize table_config: %w", err)
	}
	return string(normalized), nil
}

func validateStructuredTableDefinition(table *model.TableConfig, path string) error {
	if table == nil {
		return fmt.Errorf("%s is required", path)
	}
	if !table.NewTable {
		return nil
	}
	if table.CreateTable == nil {
		return fmt.Errorf("%s.create_table is required for structured local upload", path)
	}
	if len(table.CreateTable.TableColumn) == 0 {
		return fmt.Errorf("%s.create_table.tableColumn is required for structured local upload", path)
	}
	for i, column := range table.CreateTable.TableColumn {
		if column == nil {
			return fmt.Errorf("%s.create_table.tableColumn[%d] is nil", path, i)
		}
		if strings.TrimSpace(column.Column) == "" {
			return fmt.Errorf("%s.create_table.tableColumn[%d].column is required", path, i)
		}
		if strings.TrimSpace(column.DataType) == "" {
			return fmt.Errorf("%s.create_table.tableColumn[%d].dataType is required", path, i)
		}
	}
	return nil
}

func knowledgeBaseStructuredUploadDisplayName(rawTableConfig, fallback string) string {
	var multi model.MultiTableConfig
	if err := json.Unmarshal([]byte(rawTableConfig), &multi); err == nil && (multi.MultiSheet || len(multi.Tables) > 0) {
		for _, table := range multi.Tables {
			if table == nil || table.CreateTable == nil || table.CreateTable.Name == "" {
				continue
			}
			return table.CreateTable.Name
		}
	}
	var single model.TableConfig
	if err := json.Unmarshal([]byte(rawTableConfig), &single); err == nil && single.CreateTable != nil && single.CreateTable.Name != "" {
		return single.CreateTable.Name
	}
	return fallback
}

func (s *semanticModelService) resolveCatalogFileSourceFileID(ctx context.Context, fileID string) (string, error) {
	db := ctxutil.TenantDBFrom(ctx)
	if db == nil {
		return "", fmt.Errorf("tenant db is required")
	}

	var sourceFileID sql.NullString
	err := db.WithContext(ctx).Raw(`SELECT source_file_id
		FROM knowledge_base_sources
		WHERE kb_file_id = ?
			AND source_type IN (?, ?)
			AND source_file_id IS NOT NULL
			AND source_file_id <> ''
		ORDER BY created_at ASC, source_id ASC
		LIMIT 1`, fileID, kbSourceTypeCatalogFile, kbSourceTypeLocalFile).Row().Scan(&sourceFileID)
	if errors.Is(err, sql.ErrNoRows) {
		return fileID, nil
	}
	if err != nil {
		return "", err
	}
	if sourceFileID.Valid && strings.TrimSpace(sourceFileID.String) != "" {
		return sourceFileID.String, nil
	}
	return fileID, nil
}

func (s *semanticModelService) appendCatalogFileSourceIntent(ctx context.Context, wsID string, modelID int64, domain *KnowledgeBaseDataDomain, req CreateSemanticModelSourceRequest, actor string, reuseExisting bool) (*KnowledgeBaseSourceRecord, []KnowledgeBaseSourceJobRun, string, error) {
	// Catalog files stay on the user volume: identity is file_id; volume_id is a
	// write-time IAM/membership gate and location pointer stored on the source row.
	// copy job finishes without AddFiles into KB raw. Never guess volume from volume_files.
	if req.VolumeID <= 0 {
		return nil, nil, "", fmt.Errorf("catalog file %s requires volume_id", strings.TrimSpace(req.FileID))
	}
	var existing KnowledgeBaseSourceRecord
	reactivateRemoved := false
	if reuseExisting {
		foundRecord, found, err := s.findCatalogFileSourceByFileID(ctx, modelID, req.FileID)
		if err != nil {
			return nil, nil, "", fmt.Errorf("find existing catalog file source %s: %w", req.FileID, err)
		}
		if found {
			existing = foundRecord
			if isKnowledgeBaseSourceRemoved(existing) {
				// Removed rows may be reactivated with the caller's selected
				// volume; re-resolve metadata and rewrite raw_volume_id.
				reactivateRemoved = true
			} else {
				// Active source identity is file_id only. A second volume for
				// the same file would collide on source_id or silently reuse
				// the first row's volume provenance.
				if existing.RawVolumeID > 0 && existing.RawVolumeID != req.VolumeID {
					return nil, nil, "", semanticModelCatalogFileVolumeConflictError()
				}
				if existing.Status == kbSourceStatusSucceeded {
					return &existing, nil, ptrValue(existing.KBFileID), nil
				}
				return &existing, nil, "", nil
			}
		}
	}
	sourceFileID, err := s.resolveCatalogFileSourceFileID(ctx, req.FileID)
	if err != nil {
		return nil, nil, "", fmt.Errorf("resolve catalog file source %s: %w", req.FileID, err)
	}
	meta, err := currentCatalogFileMetadataAtVolume(ctx, sourceFileID, req.VolumeID)
	if err != nil {
		return nil, nil, "", fmt.Errorf("resolve catalog file location %s at volume %d: %w", sourceFileID, req.VolumeID, err)
	}
	fileName := strings.TrimSpace(meta.fileName)
	if fileName == "" {
		return nil, nil, "", fmt.Errorf("catalog file %s has empty display name", sourceFileID)
	}
	if meta.volumeID <= 0 {
		return nil, nil, "", fmt.Errorf("catalog file %s has no active volume", sourceFileID)
	}
	enabled := true
	source := &KnowledgeBaseSourceRecord{
		SourceID:          stableID("kb-source", modelID, kbSourceTypeCatalogFile, req.FileID),
		ModelID:           modelID,
		CatalogID:         domain.CatalogID,
		DatabaseID:        domain.DatabaseID,
		RawVolumeID:       meta.volumeID,
		ProcessedVolumeID: domain.ProcessedVolumeID,
		SourceType:        kbSourceTypeCatalogFile,
		SourceFileID:      stringPtr(sourceFileID),
		KBFileID:          stringPtr(req.FileID),
		DisplayName:       stringPtr(fileName),
		Status:            kbSourceStatusPending,
		Enabled:           &enabled,
	}
	if reactivateRemoved {
		source.SourceID = existing.SourceID
		if err := s.reactivateKnowledgeBaseSource(ctx, source, actor); err != nil {
			return nil, nil, "", fmt.Errorf("reactivate catalog file source: %w", err)
		}
	} else if err := s.insertCatalogFileSourceProcessing(ctx, source, actor); err != nil {
		return nil, nil, "", fmt.Errorf("record catalog file source: %w", err)
	}

	copyJob := newKnowledgeBaseJobRun(source, kbJobTypeCopy, stableID("kb-job", source.SourceID, kbJobTypeCopy), actor)
	copyJob.JobStatus = kbSourceJobPending
	copyJob.SourceFileID = source.SourceFileID
	copyJob.KBFileID = source.KBFileID
	if err := s.writeKnowledgeBaseSourceJobRun(ctx, &copyJob, actor, reuseExisting); err != nil {
		return nil, nil, "", fmt.Errorf("record catalog file copy job: %w", err)
	}
	ragJob := newKnowledgeBaseTriggerRAGJob(source, nil, actor)
	ragJob.SourceFileID = source.SourceFileID
	ragJob.KBFileID = source.KBFileID
	if err := s.writeKnowledgeBaseSourceJobRun(ctx, &ragJob, actor, reuseExisting); err != nil {
		return nil, nil, "", fmt.Errorf("record catalog file parse workflow trigger job: %w", err)
	}
	return source, []KnowledgeBaseSourceJobRun{copyJob, ragJob}, "", nil
}

func parseLocalStructuredUploadOperation(operationID *string) (knowledgeBaseLocalStructuredUploadOperation, bool, error) {
	if operationID == nil || !strings.HasPrefix(*operationID, kbLocalStructuredUploadOperationPrefix) {
		return knowledgeBaseLocalStructuredUploadOperation{}, false, nil
	}
	raw := strings.TrimPrefix(*operationID, kbLocalStructuredUploadOperationPrefix)
	var payload knowledgeBaseLocalStructuredUploadOperation
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return knowledgeBaseLocalStructuredUploadOperation{}, true, fmt.Errorf("parse local structured upload operation: %w", err)
	}
	if strings.TrimSpace(payload.FileID) == "" {
		return knowledgeBaseLocalStructuredUploadOperation{}, true, fmt.Errorf("local structured upload file_id is required")
	}
	if strings.TrimSpace(payload.FileName) == "" {
		return knowledgeBaseLocalStructuredUploadOperation{}, true, fmt.Errorf("local structured upload file_name is required")
	}
	if strings.TrimSpace(payload.TableConfig) == "" {
		return knowledgeBaseLocalStructuredUploadOperation{}, true, fmt.Errorf("local structured upload table_config is required")
	}
	return payload, true, nil
}

func sanitizeKnowledgeBaseSourceJobRunsForResponse(jobs []KnowledgeBaseSourceJobRun) []KnowledgeBaseSourceJobRun {
	if len(jobs) == 0 {
		return jobs
	}
	out := make([]KnowledgeBaseSourceJobRun, len(jobs))
	copy(out, jobs)
	for i := range out {
		if _, ok, _ := parseLocalStructuredUploadOperation(out[i].OperationID); ok {
			out[i].OperationID = nil
		}
	}
	return out
}

type catalogFileVectorReuseCandidate struct {
	VectorTable string
	Meta        sql.NullString
	ReuseFileID string
}

func (s *semanticModelService) tryReuseCatalogFileVectors(ctx context.Context, record KnowledgeBaseSourceRecord, binding kbVectorBinding, actor string) (bool, string, error) {
	if record.KBFileID == nil || strings.TrimSpace(*record.KBFileID) == "" {
		return false, "", nil
	}
	candidates, err := queryCatalogFileVectorReuseCandidates(ctx, record)
	if err != nil {
		return false, "", err
	}
	textCandidate, ok, err := selectTextVectorReuseCandidate(candidates, binding)
	if err != nil || !ok {
		return false, "", err
	}
	sameFileCandidates := make([]catalogFileVectorReuseCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.ReuseFileID == textCandidate.ReuseFileID {
			sameFileCandidates = append(sameFileCandidates, candidate)
		}
	}
	imageCandidate := catalogFileVectorReuseCandidate{}
	if binding.ImageVectorTable == "" {
		for _, candidate := range sameFileCandidates {
			meta, err := parseVectorAssetMeta(candidate.Meta)
			if err != nil {
				return false, "", err
			}
			if strings.TrimSpace(vectorAssetMetaString(meta, "index_modality")) == "image" {
				return false, "", nil
			}
		}
	} else {
		imageCandidate, ok, err = selectImageVectorReuseCandidate(sameFileCandidates, binding)
		if err != nil || !ok {
			return false, "", err
		}
	}
	reused, err := s.publishCatalogFileVectorReuse(ctx, record, binding, textCandidate, imageCandidate, actor)
	if err != nil || !reused {
		return false, "", err
	}
	return true, fmt.Sprintf("vector_reuse:%s:%s", textCandidate.VectorTable, binding.VectorTable), nil
}

func queryCatalogFileVectorReuseCandidates(ctx context.Context, record KnowledgeBaseSourceRecord) ([]catalogFileVectorReuseCandidate, error) {
	db := ctxutil.TenantDBFrom(ctx)
	if db == nil {
		return nil, fmt.Errorf("tenant db is required")
	}
	fileID := ptrValue(record.SourceFileID)
	if fileID == "" {
		fileID = ptrValue(record.KBFileID)
	}
	if fileID == "" {
		return nil, nil
	}
	rows, err := db.WithContext(ctx).Raw(`SELECT vector.asset_ref, vector.meta, root.asset_ref, pm.source_file_id, pm.parsed_file_id
			FROM data_asset vector
			INNER JOIN data_derivation d ON d.target_asset_id = vector.asset_id AND d.kind = 'indexed_from'
			INNER JOIN data_asset root ON root.asset_id = d.root_asset_id AND root.asset_type = 'file'
			LEFT JOIN parsed_manifest pm ON pm.root_asset_id = d.root_asset_id
		WHERE vector.asset_type = 'vector_index'
		  AND (
		    root.asset_ref = ?
		    OR pm.source_file_id = ?
		    OR pm.parsed_file_id = ?
		    OR EXISTS (
		      SELECT 1
		        FROM data_derivation file_derivation
		        INNER JOIN data_asset file_asset ON file_asset.asset_id = file_derivation.target_asset_id AND file_asset.asset_type = 'file'
		       WHERE file_derivation.root_asset_id = d.root_asset_id
		         AND file_derivation.kind IN ('transformed_from', 'derived_file_from')
		         AND file_asset.asset_ref = ?
		    )
		  )
		ORDER BY d.updated_at DESC, d.id DESC`, fileID, fileID, fileID, fileID).Rows()
	if err != nil {
		return nil, fmt.Errorf("query catalog file vector lineage: %w", err)
	}
	defer rows.Close()
	out := []catalogFileVectorReuseCandidate{}
	for rows.Next() {
		var candidate catalogFileVectorReuseCandidate
		var rootFileID, sourceFileID, parsedFileID sql.NullString
		if err := rows.Scan(&candidate.VectorTable, &candidate.Meta, &rootFileID, &sourceFileID, &parsedFileID); err != nil {
			return nil, err
		}
		candidate.VectorTable = strings.TrimSpace(candidate.VectorTable)
		candidate.ReuseFileID = reusableCatalogVectorFileID(fileID, rootFileID.String, sourceFileID.String, parsedFileID.String)
		if candidate.VectorTable != "" && candidate.ReuseFileID != "" {
			out = append(out, candidate)
		}
	}
	return out, rows.Err()
}

func reusableCatalogVectorFileID(selectedFileID, rootFileID, sourceFileID, parsedFileID string) string {
	if selectedFileID == "" {
		return ""
	}
	if rootFileID == selectedFileID {
		return rootFileID
	}
	if sourceFileID == selectedFileID {
		return sourceFileID
	}
	if parsedFileID == selectedFileID {
		if sourceFileID != "" {
			return sourceFileID
		}
		return rootFileID
	}
	if sourceFileID != "" {
		return sourceFileID
	}
	return rootFileID
}

func selectTextVectorReuseCandidate(candidates []catalogFileVectorReuseCandidate, binding kbVectorBinding) (catalogFileVectorReuseCandidate, bool, error) {
	return selectVectorReuseCandidate(candidates, binding.VectorTable, func(candidate catalogFileVectorReuseCandidate) (bool, error) {
		meta, err := parseVectorAssetMeta(candidate.Meta)
		if err != nil {
			return false, err
		}
		if strings.TrimSpace(vectorAssetMetaString(meta, "index_modality")) == "image" {
			return false, nil
		}
		return strings.TrimSpace(vectorAssetMetaString(meta, "embedding_model")) == binding.EmbeddingModel, nil
	})
}

func selectImageVectorReuseCandidate(candidates []catalogFileVectorReuseCandidate, binding kbVectorBinding) (catalogFileVectorReuseCandidate, bool, error) {
	return selectVectorReuseCandidate(candidates, binding.ImageVectorTable, func(candidate catalogFileVectorReuseCandidate) (bool, error) {
		meta, err := parseVectorAssetMeta(candidate.Meta)
		if err != nil {
			return false, err
		}
		if strings.TrimSpace(vectorAssetMetaString(meta, "index_modality")) != "image" {
			return false, nil
		}
		if strings.TrimSpace(firstNonEmptySegmentString(vectorAssetMetaString(meta, "image_embedding_model"), vectorAssetMetaString(meta, "embedding_model"))) != binding.ImageEmbeddingModel {
			return false, nil
		}
		dimension := vectorAssetMetaInt(meta, "image_embedding_dimension")
		if dimension == 0 {
			dimension = vectorAssetMetaInt(meta, "embedding_dimension")
		}
		if dimension != binding.ImageEmbeddingDimension {
			return false, nil
		}
		if strings.TrimSpace(firstNonEmptySegmentString(vectorAssetMetaString(meta, "image_preprocess_version"), vectorAssetMetaString(meta, "preprocess_version"))) != binding.ImagePreprocessVersion {
			return false, nil
		}
		if strings.TrimSpace(firstNonEmptySegmentString(vectorAssetMetaString(meta, "image_distance_metric"), vectorAssetMetaString(meta, "distance_metric"))) != binding.ImageDistanceMetric {
			return false, nil
		}
		return true, nil
	})
}

func selectVectorReuseCandidate(candidates []catalogFileVectorReuseCandidate, targetTable string, matches func(catalogFileVectorReuseCandidate) (bool, error)) (catalogFileVectorReuseCandidate, bool, error) {
	var latest catalogFileVectorReuseCandidate
	for _, candidate := range candidates {
		ok, err := matches(candidate)
		if err != nil {
			return catalogFileVectorReuseCandidate{}, false, err
		}
		if !ok {
			continue
		}
		if candidate.VectorTable == targetTable {
			return candidate, true, nil
		}
		if latest.VectorTable == "" {
			latest = candidate
		}
	}
	if latest.VectorTable == "" {
		return catalogFileVectorReuseCandidate{}, false, nil
	}
	return latest, true, nil
}

func parseVectorAssetMeta(metaJSON sql.NullString) (map[string]any, error) {
	if !metaJSON.Valid || strings.TrimSpace(metaJSON.String) == "" {
		return map[string]any{}, nil
	}
	meta := map[string]any{}
	if err := json.Unmarshal([]byte(metaJSON.String), &meta); err != nil {
		return nil, fmt.Errorf("parse vector asset metadata: %w", err)
	}
	return meta, nil
}

func vectorAssetMetaString(meta map[string]any, key string) string {
	switch v := meta[key].(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	default:
		if v == nil {
			return ""
		}
		return fmt.Sprint(v)
	}
}

func vectorAssetMetaInt(meta map[string]any, key string) int {
	switch v := meta[key].(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case json.Number:
		n, _ := v.Int64()
		return int(n)
	case string:
		n, _ := strconv.Atoi(strings.TrimSpace(v))
		return n
	default:
		return 0
	}
}

func (s *semanticModelService) appendCatalogTableSource(ctx context.Context, c *moi.Client, wsID string, modelID int64, req CreateSemanticModelSourceRequest, actor string, reuseExisting bool) (*KnowledgeBaseSourceRecord, *semanticModelTableSource, error) {
	existingSourceID := ""
	if reuseExisting {
		existing, found, err := s.findCatalogTableSourceBySourceTableID(ctx, modelID, req.TableID)
		if err != nil {
			return nil, nil, fmt.Errorf("find existing catalog table source %d: %w", req.TableID, err)
		}
		if found {
			isDirectSource := existing.SourceTableID != nil && *existing.SourceTableID == req.TableID
			if !isDirectSource {
				if !isKnowledgeBaseSourceRemoved(existing) {
					return &existing, nil, nil
				}
			} else {
				reusable := existing.Status == kbSourceStatusSucceeded &&
					existing.DBName != nil && *existing.DBName != "" &&
					existing.TableName != nil && *existing.TableName != ""
				if reusable {
					table := &semanticModelTableSource{DBName: *existing.DBName, TableNames: []string{*existing.TableName}}
					return &existing, table, nil
				}
				existingSourceID = existing.SourceID
			}
		}
	}

	tableRef, err := s.resolveCatalogTableForKnowledgeBase(ctx, c, wsID, req.TableID)
	if err != nil {
		return nil, nil, fmt.Errorf("resolve catalog table %d: %w", req.TableID, err)
	}
	source, err := semanticModelTableSourceRecord(modelID, tableRef)
	if err != nil {
		return nil, nil, err
	}
	updateExistingBySourceID := false
	if existingSourceID != "" {
		source.SourceID = existingSourceID
		updateExistingBySourceID = true
	}
	if updateExistingBySourceID {
		if err := s.upsertCatalogTableSourceProcessing(ctx, &source, actor, true); err != nil {
			return nil, nil, fmt.Errorf("record catalog table source: %w", err)
		}
	} else if err := s.insertKnowledgeBaseSource(ctx, &source, actor); err != nil {
		return nil, nil, fmt.Errorf("record catalog table source: %w", err)
	}
	return &source, &semanticModelTableSource{DBName: tableRef.dbName, TableNames: []string{tableRef.tableName}}, nil
}

func (s *semanticModelService) resolveCatalogTableForKnowledgeBase(ctx context.Context, c *moi.Client, wsID string, tableID int64) (catalogTableSourceRef, error) {
	if tableID <= 0 {
		return catalogTableSourceRef{}, fmt.Errorf("table_id is required")
	}
	if c == nil {
		return catalogTableSourceRef{}, fmt.Errorf("catalog client is required")
	}
	if wsID == "" {
		return catalogTableSourceRef{}, fmt.Errorf("workspace id is required")
	}
	detail, err := c.Databases().GetTable(ctx, wsID, tableID)
	if err != nil {
		return catalogTableSourceRef{}, err
	}
	if detail == nil || detail.Table == nil || detail.Database == nil ||
		detail.Table.Id != tableID || detail.Table.DatabaseId != detail.Database.Id ||
		detail.Database.Name == "" || detail.Table.Name == "" {
		return catalogTableSourceRef{}, fmt.Errorf("catalog table %d metadata is incomplete", tableID)
	}
	catalogName := ""
	if detail.Catalog != nil {
		catalogName = detail.Catalog.Name
	}
	return catalogTableSourceRef{
		tableID:    detail.Table.Id,
		databaseID: detail.Database.Id,
		catalogID:  detail.Table.CatalogId,
		dbName:     detail.Database.Name,
		tableName:  detail.Table.Name,
		path:       compactNonEmptyStrings(catalogName, detail.Database.Name),
	}, nil
}
