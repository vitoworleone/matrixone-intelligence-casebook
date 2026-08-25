package session

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	backendcatalog "github.com/matrixorigin/matrixflow/moi-backend/pkg/catalog"
	"github.com/matrixorigin/matrixflow/moi-backend/pkg/ctxutil"
	"github.com/matrixorigin/matrixflow/moi-backend/pkg/i18n"
	"github.com/matrixorigin/matrixflow/moi-backend/pkg/workflowv2"

	moi "github.com/matrixflow/moi-core/go-sdk"
	"github.com/matrixflow/moi-core/model/common"
)

func createSourcesHasStructuredLocalFile(sources []CreateSemanticModelSourceRequest) bool {
	for _, source := range sources {
		if source.SourceType == kbSourceTypeLocalFile && source.UploadKind == kbLocalUploadKindStructured {
			return true
		}
	}
	return false
}

func isStructuredLoadSourceType(sourceType string) bool {
	return sourceType == kbSourceTypeLocalFile || sourceType == kbSourceTypeCatalogTable
}

func hasDocumentParsingSources(sources []CreateSemanticModelSourceRequest) bool {
	for _, source := range sources {
		switch source.SourceType {
		case kbSourceTypeCatalogFile:
			return true
		case kbSourceTypeLocalFile:
			if source.UploadKind != kbLocalUploadKindStructured {
				return true
			}
		}
	}
	return false
}

func validateKnowledgeBaseCatalogFileExtension(fileName string) error {
	extWithDot := filepath.Ext(fileName)
	if extWithDot == "" {
		return fmt.Errorf("unsupported knowledge base catalog file extension")
	}
	ext := strings.ToLower(extWithDot[1:])
	switch ext {
	case "pdf", "doc", "docx", "ppt", "pptx", "xls", "xlsx", "txt", "md", "htm", "html", "eml", "msg",
		"png", "jpg", "jpeg", "gif", "bmp", "webp", "tif", "tiff", "svg",
		"mp3", "wav", "m4a", "aac", "flac", "ogg", "wma",
		"mp4", "mov", "avi", "mkv", "webm", "mpeg", "wmv", "flv":
		return nil
	default:
		return fmt.Errorf("unsupported knowledge base catalog file extension %q", ext)
	}
}

func semanticModelActor(ctx context.Context) string {
	if userID := ctxutil.UserIDFrom(ctx); userID != "" {
		return userID
	}
	return ctxutil.UIDFrom(ctx)
}

func (s *semanticModelService) ensureKnowledgeBaseRAGWorkflow(ctx context.Context, wsID string, modelID int64, name, description string, domain *KnowledgeBaseDataDomain, files json.RawMessage) error {
	if domain == nil {
		return fmt.Errorf("knowledge base data domain is required")
	}
	if domain.RawVolumeID <= 0 {
		return fmt.Errorf("knowledge base document raw volume is required")
	}
	if domain.ProcessedVolumeID <= 0 {
		return fmt.Errorf("knowledge base processed volume is required")
	}
	if s.workflowTemplateService == nil {
		return fmt.Errorf("workflow template service is not configured")
	}
	if s.workflowService == nil {
		return fmt.Errorf("workflow service is not configured")
	}
	templateKey := knowledgeBaseWorkflowTemplateKey(files)
	tpl, err := s.workflowTemplateService.GetByTemplateKey(ctx, templateKey)
	if err != nil {
		return fmt.Errorf("get workflow template %s: %w", templateKey, err)
	}
	if tpl == nil {
		return fmt.Errorf("workflow template %s returned empty result", templateKey)
	}
	workflowID := knowledgeBaseWorkflowID(wsID, modelID)
	return s.workflowService.DeployKnowledgeBaseWorkflow(ctx, KnowledgeBaseWorkflowDeployRequest{
		WorkflowID:    workflowID,
		Name:          knowledgeBaseWorkflowName(ctx, name, modelID),
		Description:   description,
		DSLYAML:       tpl.DSLYaml,
		InputFormJSON: tpl.InputForm,
		DefaultValues: knowledgeBaseWorkflowDefaultValues(domain, modelID, files),
		ExecutionMode: workflowv2.ExecutionModeOneShot,
	})
}

func knowledgeBaseWorkflowID(wsID string, modelID int64) string {
	return stableID("kb-rag-workflow", strings.TrimSpace(wsID), modelID)
}

func knowledgeBaseMediaWorkflowID(wsID string, modelID int64, templateKey string) string {
	return stableID("kb-rag-workflow", strings.TrimSpace(wsID), modelID, templateKey)
}

func templateKeyForKnowledgeBaseFile(fileName *string) (string, error) {
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(strings.TrimSpace(ptrValue(fileName))), "."))
	switch ext {
	case "pdf", "doc", "docx", "ppt", "pptx", "xls", "xlsx", "txt", "md", "htm", "html", "eml", "msg",
		"png", "jpg", "jpeg", "gif", "bmp", "webp", "tif", "tiff", "svg":
		return kbStandardRAGTemplateKey, nil
	case "mp3", "wav", "m4a", "aac", "flac", "ogg", "wma":
		return kbAudioRAGTemplateKey, nil
	case "mp4", "mov", "avi", "mkv", "webm", "mpeg", "wmv", "flv":
		return kbVideoRAGTemplateKey, nil
	default:
		return "", fmt.Errorf("no knowledge base workflow template supports file extension %q", ext)
	}
}

func (s *semanticModelService) ensureKnowledgeBaseMediaRAGWorkflow(ctx context.Context, wsID string, modelID int64, name, description, templateKey string, domain *KnowledgeBaseDataDomain, files json.RawMessage) (string, error) {
	if templateKey != kbAudioRAGTemplateKey && templateKey != kbVideoRAGTemplateKey {
		return "", fmt.Errorf("unsupported knowledge base media workflow template %q", templateKey)
	}
	if domain == nil || domain.RawVolumeID <= 0 || domain.ProcessedVolumeID <= 0 {
		return "", fmt.Errorf("knowledge base data domain is incomplete")
	}
	if s.workflowTemplateService == nil || s.workflowService == nil {
		return "", fmt.Errorf("knowledge base workflow services are not configured")
	}
	tpl, err := s.workflowTemplateService.GetByTemplateKey(ctx, templateKey)
	if err != nil {
		return "", fmt.Errorf("get workflow template %s: %w", templateKey, err)
	}
	if tpl == nil {
		return "", fmt.Errorf("workflow template %s returned empty result", templateKey)
	}
	workflowID := knowledgeBaseMediaWorkflowID(wsID, modelID, templateKey)
	if err := s.workflowService.DeployKnowledgeBaseWorkflow(ctx, KnowledgeBaseWorkflowDeployRequest{
		WorkflowID:    workflowID,
		Name:          knowledgeBaseWorkflowName(ctx, name, modelID),
		Description:   description,
		DSLYAML:       tpl.DSLYaml,
		InputFormJSON: tpl.InputForm,
		DefaultValues: knowledgeBaseWorkflowDefaultValues(domain, modelID, files),
		ExecutionMode: workflowv2.ExecutionModeOneShot,
	}); err != nil {
		return "", err
	}
	return workflowID, nil
}

// knowledgeBaseDatabaseName returns the catalog/MO database name for a knowledge base.
// Identity and display both use this value: SQL editor and Catalog must show the same identifier.
func knowledgeBaseDatabaseName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", serviceError(ErrCodeBadRequest, i18n.KeyNameRequired, nil)
	}
	if err := backendcatalog.ValidateCatalogIdentifier(name); err != nil {
		return "", serviceError(ErrCodeBadRequest, i18n.KeyNameContainsInvalidChars, nil)
	}
	return name, nil
}

func validateKnowledgeBaseName(name string) error {
	_, err := knowledgeBaseDatabaseName(name)
	return err
}

// resolveDefaultCatalogID is the only session entry that chooses the Catalog
// for newly created knowledge-base databases and volumes.
func (s *semanticModelService) resolveDefaultCatalogID(ctx context.Context) (int64, error) {
	if s.dataDomainService == nil {
		return 0, fmt.Errorf("catalog data-domain service is not configured")
	}
	catalogID, err := s.dataDomainService.ResolveDefaultCatalogID(ctx)
	if err != nil {
		return 0, fmt.Errorf("resolve default catalog: %w", err)
	}
	if catalogID <= 0 {
		return 0, knowledgeBaseDataDomainCatalogRepairFailedError(fmt.Errorf("default catalog id is required"))
	}
	return catalogID, nil
}

// ensureKnowledgeBaseDatabaseNameAvailable rejects names that already exist as a
// database under the default catalog when the domain has not bound one yet.
func (s *semanticModelService) ensureKnowledgeBaseDatabaseNameAvailable(ctx context.Context, catalogID int64, name string, domain *KnowledgeBaseDataDomain) error {
	if domain != nil && domain.DatabaseID > 0 {
		return nil
	}
	if s.dataDomainService == nil {
		return fmt.Errorf("catalog data-domain service is not configured")
	}
	existingID, path, ok, err := s.dataDomainService.ResolveDatabaseByName(ctx, catalogID, name)
	if err != nil {
		return fmt.Errorf("resolve knowledge base database name %s: %w", name, err)
	}
	if ok && existingID > 0 {
		return knowledgeBaseDatabaseNameExistsError(path)
	}
	return nil
}

// beginKnowledgeBaseDataDomain records the domain row before catalog resources exist.
// Create starts in provisioning so no append can claim its row. First append starts
// in failed and claims provisioning in reconcileKnowledgeBaseDataDomain.
// Insert is exclusive: on duplicate key, re-read the existing row instead of
// overwriting it (single-ownership with concurrent first append/create).
func (s *semanticModelService) beginKnowledgeBaseDataDomain(ctx context.Context, modelID, catalogID int64, initialStatus, actor string) (*KnowledgeBaseDataDomain, error) {
	if modelID <= 0 {
		return nil, fmt.Errorf("knowledge base model id is required")
	}
	if catalogID <= 0 {
		return nil, knowledgeBaseDataDomainCatalogRepairFailedError(fmt.Errorf("knowledge base catalog id is required"))
	}
	domain := &KnowledgeBaseDataDomain{
		ModelID:       modelID,
		CatalogID:     catalogID,
		EnsureStatus:  initialStatus,
		LastCheckedAt: time.Now().Unix(),
	}
	if err := s.upsertKnowledgeBaseDataDomain(ctx, domain, actor); err != nil {
		if isDuplicateEntryError(err) {
			fresh, ok, getErr := s.getKnowledgeBaseDataDomain(ctx, modelID)
			if getErr != nil {
				return nil, fmt.Errorf("get knowledge base data domain after insert conflict: %w", getErr)
			}
			if ok {
				return fresh, nil
			}
		}
		return nil, fmt.Errorf("record initial knowledge base data domain: %w", err)
	}
	return domain, nil
}

// provisionKnowledgeBaseDataDomain ensures catalog + database + raw/processed volumes
// are ready for a knowledge base. This is the single resource-provision entry for
// empty create, create-with-sources, recoverable repair, and first source append.
func (s *semanticModelService) provisionKnowledgeBaseDataDomain(ctx context.Context, modelID int64, name, description, actor string) (*KnowledgeBaseDataDomain, error) {
	domain, ok, err := s.getKnowledgeBaseDataDomain(ctx, modelID)
	if err != nil {
		return nil, fmt.Errorf("get knowledge base data domain: %w", err)
	}
	return s.reconcileKnowledgeBaseDataDomain(ctx, domain, ok, modelID, name, description, actor)
}

// reconcileKnowledgeBaseDataDomain is the shared body of provision when the caller
// already loaded (or knows absence of) the domain row.
//
// Ownership rule: only a caller that wins failed→provisioning CAS may create or
// mutate catalog resources. Any externally observed provisioning means another
// request owns the row; this caller must not continue.
func (s *semanticModelService) reconcileKnowledgeBaseDataDomain(ctx context.Context, domain *KnowledgeBaseDataDomain, exists bool, modelID int64, name, description, actor string) (*KnowledgeBaseDataDomain, error) {
	if !exists {
		catalogID, err := s.resolveDefaultCatalogID(ctx)
		if err != nil {
			return nil, err
		}
		var beginErr error
		domain, beginErr = s.beginKnowledgeBaseDataDomain(ctx, modelID, catalogID, kbEnsureStatusFailed, actor)
		if beginErr != nil {
			// Insert conflict with no readable row is rare; surface as-is.
			// When begin succeeds after conflict it returns the existing row —
			// never an unconditional zero-ID overwrite.
			return nil, beginErr
		}
		// begin may have returned an existing row (insert race). Re-enter with
		// exists=true so provisioning/ready is never treated as a fresh claim.
		if domain.EnsureStatus != kbEnsureStatusFailed && domain.EnsureStatus != "" {
			return s.reconcileKnowledgeBaseDataDomain(ctx, domain, true, modelID, name, description, actor)
		}
		// Fresh failed insert (or conflict re-read still failed): claim then finish.
		claimed, claimErr := s.claimKnowledgeBaseDataDomainProvision(ctx, domain.ModelID, actor)
		if claimErr != nil {
			return nil, fmt.Errorf("claim knowledge base data domain provision: %w", claimErr)
		}
		if !claimed {
			fresh, ok, getErr := s.getKnowledgeBaseDataDomain(ctx, modelID)
			if getErr != nil {
				return nil, fmt.Errorf("get knowledge base data domain after begin claim miss: %w", getErr)
			}
			if !ok {
				return nil, knowledgeBaseDataDomainNotFoundError()
			}
			return s.reconcileKnowledgeBaseDataDomain(ctx, fresh, true, modelID, name, description, actor)
		}
		domain.EnsureStatus = kbEnsureStatusProvisioning
		domain.LastEnsureError = nil
		if err := s.finishKnowledgeBaseDataDomainProvision(ctx, domain, name, description, actor); err != nil {
			return nil, err
		}
		return domain, nil
	}
	if domain == nil {
		return nil, knowledgeBaseDataDomainNotFoundError()
	}

	switch domain.EnsureStatus {
	case kbEnsureStatusReady:
		// Ready: catalog repair + raw-volume bookkeeping only; never re-create.
	case kbEnsureStatusProvisioning:
		return nil, knowledgeBaseDataDomainInProgressError(modelID)
	case kbEnsureStatusFailed, "":
		claimed, claimErr := s.claimKnowledgeBaseDataDomainProvision(ctx, domain.ModelID, actor)
		if claimErr != nil {
			return nil, fmt.Errorf("claim knowledge base data domain provision: %w", claimErr)
		}
		if !claimed {
			fresh, ok, getErr := s.getKnowledgeBaseDataDomain(ctx, modelID)
			if getErr != nil {
				return nil, fmt.Errorf("get knowledge base data domain after claim miss: %w", getErr)
			}
			if !ok {
				return nil, knowledgeBaseDataDomainNotFoundError()
			}
			// CAS miss: re-read only; never provision without a new claim.
			return s.reconcileKnowledgeBaseDataDomain(ctx, fresh, true, modelID, name, description, actor)
		}
		domain.EnsureStatus = kbEnsureStatusProvisioning
		domain.LastEnsureError = nil
	default:
		return nil, fmt.Errorf("knowledge base data domain %d has unexpected status %q", modelID, domain.EnsureStatus)
	}

	catalogRepaired, repairErr := s.repairKnowledgeBaseDataDomainCatalog(ctx, domain)
	if repairErr != nil {
		if domain.EnsureStatus == kbEnsureStatusProvisioning {
			_ = s.recordKnowledgeBaseDataDomainFailureIfStatus(ctx, domain, repairErr, actor, kbEnsureStatusProvisioning)
		}
		return nil, repairErr
	}
	if catalogRepaired {
		if err := s.updateKnowledgeBaseDataDomainCatalog(ctx, domain.ModelID, domain.CatalogID, actor); err != nil {
			if domain.EnsureStatus == kbEnsureStatusProvisioning {
				_ = s.recordKnowledgeBaseDataDomainFailureIfStatus(ctx, domain, err, actor, kbEnsureStatusProvisioning)
			}
			return nil, fmt.Errorf("record repaired knowledge base data domain: %w", err)
		}
	}

	// Fully bound ready: bookkeeping only.
	if domain.EnsureStatus == kbEnsureStatusReady &&
		domain.CatalogID > 0 && domain.DatabaseID > 0 && domain.RawVolumeID > 0 && domain.ProcessedVolumeID > 0 {
		if err := s.upsertKnowledgeBaseRawVolume(ctx, domain, kbRawKindDocument, domain.RawVolumeID, kbEnsureStatusReady, nil, actor); err != nil {
			return nil, fmt.Errorf("record knowledge base document raw volume: %w", err)
		}
		return domain, nil
	}

	// Claim holder only.
	if domain.EnsureStatus != kbEnsureStatusProvisioning {
		return nil, fmt.Errorf("knowledge base data domain %d has unexpected status %q after reconcile", modelID, domain.EnsureStatus)
	}
	if err := s.finishKnowledgeBaseDataDomainProvision(ctx, domain, name, description, actor); err != nil {
		return nil, err
	}
	return domain, nil
}

// prepareKnowledgeBaseDataDomainResourcesForCreate claims the new domain and
// persists its Catalog IDs while keeping provisioning ownership. The create
// caller publishes ready only after all remaining create side effects succeed.
func (s *semanticModelService) prepareKnowledgeBaseDataDomainResourcesForCreate(ctx context.Context, domain *KnowledgeBaseDataDomain, name, description, actor string) error {
	if domain == nil {
		return knowledgeBaseDataDomainNotFoundError()
	}
	if domain.CatalogID <= 0 {
		return knowledgeBaseDataDomainCatalogRepairFailedError(fmt.Errorf("knowledge base catalog id is required"))
	}
	switch domain.EnsureStatus {
	case kbEnsureStatusProvisioning:
		// createKnowledgeBaseModelShell inserted this row for the current request.
	case kbEnsureStatusFailed, "":
		claimed, claimErr := s.claimKnowledgeBaseDataDomainProvision(ctx, domain.ModelID, actor)
		if claimErr != nil {
			return fmt.Errorf("claim knowledge base data domain provision: %w", claimErr)
		}
		if !claimed {
			return knowledgeBaseDataDomainInProgressError(domain.ModelID)
		}
		domain.EnsureStatus = kbEnsureStatusProvisioning
		domain.LastEnsureError = nil
	default:
		return fmt.Errorf("knowledge base data domain %d has unexpected status %q", domain.ModelID, domain.EnsureStatus)
	}
	databaseID, rawVolumeID, processedVolumeID, err := s.ensureKnowledgeBaseDataDomainResources(ctx, domain, CreateSemanticModelWithSourcesRequest{
		Name:        name,
		Description: description,
	})
	if err != nil {
		return fmt.Errorf("ensure knowledge base data domain: %w", err)
	}
	domain.DatabaseID = databaseID
	domain.RawVolumeID = rawVolumeID
	domain.ProcessedVolumeID = processedVolumeID
	domain.LastCheckedAt = time.Now().Unix()
	if err := s.updateKnowledgeBaseDataDomainIfStatus(ctx, domain, actor, kbEnsureStatusProvisioning); err != nil {
		return fmt.Errorf("record provisioning knowledge base data domain: %w", err)
	}
	return nil
}

func (s *semanticModelService) finalizeKnowledgeBaseDataDomainCreate(ctx context.Context, domain *KnowledgeBaseDataDomain, actor string) error {
	if domain == nil {
		return knowledgeBaseDataDomainNotFoundError()
	}
	if domain.EnsureStatus != kbEnsureStatusProvisioning ||
		domain.DatabaseID <= 0 || domain.RawVolumeID <= 0 || domain.ProcessedVolumeID <= 0 {
		return fmt.Errorf("knowledge base data domain %d is not ready to finalize", domain.ModelID)
	}
	if err := s.upsertKnowledgeBaseRawVolume(ctx, domain, kbRawKindDocument, domain.RawVolumeID, kbEnsureStatusReady, nil, actor); err != nil {
		return fmt.Errorf("record knowledge base document raw volume: %w", err)
	}
	ready := *domain
	ready.EnsureStatus = kbEnsureStatusReady
	ready.LastEnsureError = nil
	ready.LastCheckedAt = time.Now().Unix()
	if err := s.updateKnowledgeBaseDataDomainIfStatus(ctx, &ready, actor, kbEnsureStatusProvisioning); err != nil {
		return fmt.Errorf("record ready knowledge base data domain: %w", err)
	}
	*domain = ready
	return nil
}

// finishKnowledgeBaseDataDomainProvision creates resources and writes ready.
// Caller must have just won failed→provisioning CAS; domain.EnsureStatus is provisioning.
func (s *semanticModelService) finishKnowledgeBaseDataDomainProvision(ctx context.Context, domain *KnowledgeBaseDataDomain, name, description, actor string) error {
	databaseID, rawVolumeID, processedVolumeID, err := s.ensureKnowledgeBaseDataDomainResources(ctx, domain, CreateSemanticModelWithSourcesRequest{
		Name:        name,
		Description: description,
	})
	if err != nil {
		if recordErr := s.recordKnowledgeBaseDataDomainFailureIfStatus(ctx, domain, err, actor, kbEnsureStatusProvisioning); recordErr != nil {
			return fmt.Errorf("ensure knowledge base data domain: %w; record partial resources: %v", err, recordErr)
		}
		return fmt.Errorf("ensure knowledge base data domain: %w", err)
	}
	domain.DatabaseID = databaseID
	domain.RawVolumeID = rawVolumeID
	domain.ProcessedVolumeID = processedVolumeID
	domain.EnsureStatus = kbEnsureStatusReady
	domain.LastEnsureError = nil
	domain.LastCheckedAt = time.Now().Unix()
	if err := s.updateKnowledgeBaseDataDomainIfStatus(ctx, domain, actor, kbEnsureStatusProvisioning); err != nil {
		return s.handleKnowledgeBaseDataDomainReadyWriteFailure(ctx, domain, databaseID, rawVolumeID, processedVolumeID, err, actor)
	}
	if err := s.upsertKnowledgeBaseRawVolume(ctx, domain, kbRawKindDocument, rawVolumeID, kbEnsureStatusReady, nil, actor); err != nil {
		// ready already committed; do not clobber with unconditional failed.
		return fmt.Errorf("record knowledge base document raw volume: %w", err)
	}
	return nil
}

// handleKnowledgeBaseDataDomainReadyWriteFailure keeps resource IDs when the claim
// holder cannot write ready, without ever unconditionally overwriting a winner.
func (s *semanticModelService) handleKnowledgeBaseDataDomainReadyWriteFailure(
	ctx context.Context,
	domain *KnowledgeBaseDataDomain,
	databaseID, rawVolumeID, processedVolumeID int64,
	readyErr error,
	actor string,
) error {
	// If CAS missed, another writer already moved the row — re-read only.
	if errors.Is(readyErr, errKnowledgeBaseDataDomainCASFailed) {
		fresh, ok, getErr := s.getKnowledgeBaseDataDomain(ctx, domain.ModelID)
		if getErr != nil {
			_ = s.cleanupKnowledgeBaseCatalogResourcesByIDs(ctx, databaseID, rawVolumeID, processedVolumeID)
			return fmt.Errorf("record knowledge base data domain: %w; reload: %v", readyErr, getErr)
		}
		if ok && fresh.EnsureStatus == kbEnsureStatusReady {
			if fresh.DatabaseID == databaseID && fresh.RawVolumeID == rawVolumeID && fresh.ProcessedVolumeID == processedVolumeID {
				*domain = *fresh
				return nil
			}
			if cleanupErr := s.cleanupKnowledgeBaseCatalogResourcesByIDs(ctx, databaseID, rawVolumeID, processedVolumeID); cleanupErr != nil {
				return fmt.Errorf("record knowledge base data domain: %w; cleanup orphan resources: %v", readyErr, cleanupErr)
			}
			*domain = *fresh
			return fmt.Errorf("record knowledge base data domain: %w", readyErr)
		}
		// Not ready (failed / still provisioning owned by someone else): drop our orphans.
		if cleanupErr := s.cleanupKnowledgeBaseCatalogResourcesByIDs(ctx, databaseID, rawVolumeID, processedVolumeID); cleanupErr != nil {
			return fmt.Errorf("record knowledge base data domain: %w; cleanup orphan resources: %v", readyErr, cleanupErr)
		}
		if ok {
			*domain = *fresh
		}
		return fmt.Errorf("record knowledge base data domain: %w", readyErr)
	}

	// Non-CAS error while we still expect to hold provisioning: persist failed+IDs
	// only with WHERE ensure_status = provisioning.
	failDomain := *domain
	failDomain.DatabaseID = databaseID
	failDomain.RawVolumeID = rawVolumeID
	failDomain.ProcessedVolumeID = processedVolumeID
	failDomain.EnsureStatus = kbEnsureStatusFailed
	failDomain.LastEnsureError = stringPtr(readyErr.Error())
	failDomain.LastCheckedAt = time.Now().Unix()
	if recordErr := s.updateKnowledgeBaseDataDomainIfStatus(ctx, &failDomain, actor, kbEnsureStatusProvisioning); recordErr != nil {
		// CAS miss after re-race: re-read only, never unconditional UPDATE.
		fresh, ok, getErr := s.getKnowledgeBaseDataDomain(ctx, domain.ModelID)
		if getErr == nil && ok && fresh.EnsureStatus == kbEnsureStatusReady {
			if fresh.DatabaseID != databaseID || fresh.RawVolumeID != rawVolumeID || fresh.ProcessedVolumeID != processedVolumeID {
				_ = s.cleanupKnowledgeBaseCatalogResourcesByIDs(ctx, databaseID, rawVolumeID, processedVolumeID)
			}
			*domain = *fresh
			return fmt.Errorf("record knowledge base data domain: %w", readyErr)
		}
		if cleanupErr := s.cleanupKnowledgeBaseCatalogResourcesByIDs(ctx, databaseID, rawVolumeID, processedVolumeID); cleanupErr != nil {
			return fmt.Errorf("record knowledge base data domain: %w; record failure with ids: %v; cleanup orphan resources: %v", readyErr, recordErr, cleanupErr)
		}
		if getErr == nil && ok {
			*domain = *fresh
		}
		return fmt.Errorf("record knowledge base data domain: %w; record failure with ids: %v", readyErr, recordErr)
	}
	*domain = failDomain
	return fmt.Errorf("record knowledge base data domain: %w", readyErr)
}

func knowledgeBaseDataDomainInProgressError(modelID int64) error {
	return fmt.Errorf("knowledge base data domain %d is being provisioned", modelID)
}

func knowledgeBaseWorkflowName(ctx context.Context, name string, modelID int64) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return i18n.Tt(ctx, i18n.KeySessionSemanticModelWorkflowNameDefault, map[string]any{"ModelID": modelID})
	}
	return i18n.Tt(ctx, i18n.KeySessionSemanticModelWorkflowNameNamed, map[string]any{"Name": name})
}

func knowledgeBaseWorkflowTemplateKey(files json.RawMessage) string {
	if enabled, _ := knowledgeBaseWorkflowImageIndexDefault(0, files)["enabled"].(bool); enabled {
		return kbStandardRAGImageTemplateKey
	}
	return kbStandardRAGTemplateKey
}

func knowledgeBaseWorkflowDefaultValues(domain *KnowledgeBaseDataDomain, modelID int64, files json.RawMessage) map[string]any {
	imageIndex := knowledgeBaseWorkflowImageIndexDefault(modelID, files)
	return map[string]any{
		"semantic_model_id": modelID,
		"vlm_ocr_model":     kbDefaultVLMOCRModel,
		"source_ref": map[string]any{
			"kind":          "volume",
			"resource_type": "volume",
			"volume_id":     domain.RawVolumeID,
		},
		"output_ref": map[string]any{
			"kind":          "volume",
			"resource_type": "volume",
			"volume_id":     domain.ProcessedVolumeID,
		},
		"vector_index": knowledgeBaseWorkflowVectorIndexDefault(modelID, files, imageIndex),
		"image_index":  imageIndex,
	}
}

func knowledgeBaseWorkflowVectorIndexDefault(modelID int64, raw json.RawMessage, imageIndex map[string]any) map[string]any {
	vectorIndex := map[string]any{
		"vector_table":    defaultKnowledgeBaseVectorTable(modelID),
		"embedding_model": kbDefaultEmbeddingModel,
	}
	if imageIndex != nil {
		vectorIndex["image_index_enabled"] = imageIndex["enabled"]
		vectorIndex["image_vector_table"] = imageIndex["image_vector_table"]
		vectorIndex["image_embedding_model"] = imageIndex["image_embedding_model"]
		vectorIndex["image_embedding_backend_id"] = imageIndex["image_embedding_backend_id"]
		vectorIndex["image_embedding_dimension"] = imageIndex["image_embedding_dimension"]
		vectorIndex["image_preprocess_version"] = imageIndex["image_preprocess_version"]
		vectorIndex["image_distance_metric"] = imageIndex["image_distance_metric"]
	}
	if len(raw) == 0 || string(raw) == "null" {
		return vectorIndex
	}
	var files map[string]json.RawMessage
	if err := json.Unmarshal(raw, &files); err != nil {
		return vectorIndex
	}
	if value := stringJSONValue(files, "vector_table"); value != "" {
		vectorIndex["vector_table"] = value
	}
	if value := stringJSONValue(files, "embedding_model"); value != "" {
		vectorIndex["embedding_model"] = value
	}
	return vectorIndex
}

func knowledgeBaseWorkflowImageIndexDefault(modelID int64, raw json.RawMessage) map[string]any {
	imageIndex := map[string]any{
		"enabled":                    false,
		"image_vector_table":         defaultKnowledgeBaseImageVectorTable(modelID),
		"image_embedding_model":      "",
		"image_embedding_backend_id": "",
		"image_embedding_dimension":  0,
		"image_preprocess_version":   "",
		"image_distance_metric":      "",
	}
	if len(raw) == 0 || string(raw) == "null" {
		return imageIndex
	}
	var files map[string]json.RawMessage
	if err := json.Unmarshal(raw, &files); err != nil {
		return imageIndex
	}
	if value := stringJSONValue(files, "image_vector_table"); value != "" {
		imageIndex["image_vector_table"] = value
	}
	if value := stringJSONValue(files, "image_embedding_model"); value != "" {
		imageIndex["image_embedding_model"] = value
	}
	if value := stringJSONValue(files, "image_embedding_backend_id"); value != "" {
		imageIndex["image_embedding_backend_id"] = value
	}
	if value, ok := intJSONValue(files, "image_embedding_dimension"); ok {
		imageIndex["image_embedding_dimension"] = value
	}
	if value := stringJSONValue(files, "image_preprocess_version"); value != "" {
		imageIndex["image_preprocess_version"] = value
	}
	if value := stringJSONValue(files, "image_distance_metric"); value != "" {
		imageIndex["image_distance_metric"] = value
	}
	imageIndex["enabled"] = knowledgeBaseImageIndexConfigComplete(imageIndex)
	return imageIndex
}

func knowledgeBaseImageIndexConfigComplete(imageIndex map[string]any) bool {
	if imageIndex == nil {
		return false
	}
	for _, key := range []string{"image_embedding_model", "image_embedding_backend_id", "image_preprocess_version", "image_distance_metric"} {
		value, ok := imageIndex[key].(string)
		if !ok || value == "" {
			return false
		}
	}
	dimension, ok := imageIndex["image_embedding_dimension"].(int64)
	if !ok {
		if value, ok := imageIndex["image_embedding_dimension"].(int); ok {
			dimension = int64(value)
		}
	}
	return dimension > 0
}

func (s *semanticModelService) ensureKnowledgeBaseDataDomainResources(ctx context.Context, domain *KnowledgeBaseDataDomain, params CreateSemanticModelWithSourcesRequest) (int64, int64, int64, error) {
	if domain == nil {
		return 0, 0, 0, fmt.Errorf("knowledge base data domain is required")
	}
	if domain.CatalogID <= 0 {
		return 0, 0, 0, fmt.Errorf("knowledge base catalog is required")
	}
	databaseName, err := knowledgeBaseDatabaseName(params.Name)
	if err != nil {
		return 0, 0, 0, err
	}
	if domain.DatabaseID <= 0 {
		// Name availability is checked at create entry. An unbound domain may
		// create the database, but never takes over an existing name collision.
		databaseID, err := s.ensureKnowledgeBaseDatabaseID(ctx, domain.CatalogID, databaseName, params.Description)
		if err != nil {
			return 0, 0, 0, err
		}
		domain.DatabaseID = databaseID
	}
	if domain.RawVolumeID <= 0 {
		rawVolumeID, err := s.ensureKnowledgeBaseVolumeID(ctx, domain.DatabaseID, rawVolumeName(kbRawKindDocument), "Knowledge base document raw source files")
		if err != nil {
			return domain.DatabaseID, 0, 0, err
		}
		domain.RawVolumeID = rawVolumeID
	}
	if domain.ProcessedVolumeID <= 0 {
		processedVolumeID, err := s.ensureKnowledgeBaseVolumeID(ctx, domain.DatabaseID, "processed", "Knowledge base processed files")
		if err != nil {
			return domain.DatabaseID, domain.RawVolumeID, 0, err
		}
		domain.ProcessedVolumeID = processedVolumeID
	}
	return domain.DatabaseID, domain.RawVolumeID, domain.ProcessedVolumeID, nil
}

func (s *semanticModelService) repairKnowledgeBaseDataDomainCatalog(ctx context.Context, domain *KnowledgeBaseDataDomain) (bool, error) {
	if domain == nil {
		return false, knowledgeBaseDataDomainCatalogRepairFailedError(fmt.Errorf("knowledge base data domain is required"))
	}
	if s.dataDomainService == nil {
		return false, fmt.Errorf("catalog data-domain service is not configured")
	}
	if domain.DatabaseID > 0 {
		catalogID, err := s.dataDomainService.ResolveCatalogIDByDatabaseID(ctx, domain.DatabaseID)
		if err != nil || catalogID <= 0 {
			if err == nil {
				err = fmt.Errorf("database %d has no valid parent catalog", domain.DatabaseID)
			}
			return false, knowledgeBaseDataDomainCatalogRepairFailedError(err)
		}
		if domain.CatalogID == catalogID {
			return false, nil
		}
		domain.CatalogID = catalogID
		return true, nil
	}
	if domain.RawVolumeID <= 0 && domain.ProcessedVolumeID <= 0 {
		catalogID, err := s.resolveDefaultCatalogID(ctx)
		if err != nil {
			return false, err
		}
		if domain.CatalogID == catalogID {
			return false, nil
		}
		domain.CatalogID = catalogID
		return true, nil
	}
	if domain.CatalogID > 0 {
		return false, nil
	}
	return false, knowledgeBaseDataDomainCatalogRepairFailedError(fmt.Errorf("knowledge base data domain has no recoverable database"))
}

func (s *semanticModelService) ensureKnowledgeBaseDatabaseID(ctx context.Context, catalogID int64, name, description string) (int64, error) {
	// Display name is empty: identity and UI both use the physical database name.
	// Catalog/SQL surfaces must not project a separate display binding for KB databases.
	databaseID, err := s.dataDomainService.CreateDatabase(ctx, catalogID, name, description, "")
	if err == nil {
		return databaseID, nil
	}
	if !isKnowledgeBaseCatalogDatabaseAlreadyExists(err) {
		return 0, err
	}
	// A name match is not ownership evidence. Recoverable domains persist their
	// database ID before retrying, so an unbound collision must never be reused.
	resolvedID, path, found, resolveErr := s.dataDomainService.ResolveDatabaseByName(ctx, catalogID, name)
	if resolveErr != nil {
		return 0, fmt.Errorf("resolve conflicting knowledge base database %s: %w", name, resolveErr)
	}
	if !found || resolvedID <= 0 || strings.TrimSpace(path) == "" {
		return 0, err
	}
	return 0, knowledgeBaseDatabaseNameExistsError(path)
}

func (s *semanticModelService) ensureKnowledgeBaseVolumeID(ctx context.Context, databaseID int64, name, description string) (int64, error) {
	volumeID, err := s.dataDomainService.CreateVolume(ctx, databaseID, name, description)
	if err == nil {
		return volumeID, nil
	}
	if !isKnowledgeBaseCatalogVolumeAlreadyExists(err) {
		return 0, err
	}
	resolvedID, ok, resolveErr := s.dataDomainService.ResolveVolumeIDByName(ctx, databaseID, name)
	if resolveErr != nil {
		return 0, fmt.Errorf("resolve existing knowledge base volume %s: %w", name, resolveErr)
	}
	if !ok {
		return 0, err
	}
	return resolvedID, nil
}

func rawVolumeName(rawKind string) string {
	switch rawKind {
	case kbRawKindImage:
		return "raw_image"
	case kbRawKindAudioVideo:
		return "raw_audio_video"
	case kbRawKindStructured:
		return "raw_structured"
	default:
		return "raw_document"
	}
}

func rawVolumeDescription(rawKind string) string {
	switch rawKind {
	case kbRawKindImage:
		return "Knowledge base image raw source files"
	case kbRawKindAudioVideo:
		return "Knowledge base audio and video raw source files"
	case kbRawKindStructured:
		return "Knowledge base structured raw source files"
	default:
		return "Knowledge base document raw source files"
	}
}

func rawKindForLocalFile(req CreateSemanticModelSourceRequest) string {
	if req.UploadKind == kbLocalUploadKindStructured {
		return kbRawKindStructured
	}
	return rawKindForFileName(req.FileName)
}

func rawKindForFileName(fileName string) string {
	ext := strings.ToLower(fileName)
	if idx := strings.LastIndex(ext, "."); idx >= 0 {
		ext = ext[idx+1:]
	}
	switch ext {
	case "png", "jpg", "jpeg", "gif", "bmp", "webp", "tif", "tiff", "svg":
		return kbRawKindImage
	case "mp3", "wav", "m4a", "aac", "flac", "ogg", "wma", "mp4", "mov", "avi", "mkv", "webm", "mpeg", "wmv", "flv":
		return kbRawKindAudioVideo
	default:
		return kbRawKindDocument
	}
}

// recordKnowledgeBaseDataDomainFailureIfStatus persists failed status with whatever
// resource IDs are already on domain (including partial create). expectedStatus
// enables CAS so concurrent winners are not overwritten with a zero-ID snapshot.
func (s *semanticModelService) recordKnowledgeBaseDataDomainFailureIfStatus(ctx context.Context, domain *KnowledgeBaseDataDomain, cause error, actor, expectedStatus string) error {
	if domain == nil {
		return fmt.Errorf("knowledge base data domain is required")
	}
	domain.EnsureStatus = kbEnsureStatusFailed
	domain.LastEnsureError = stringPtr(cause.Error())
	domain.LastCheckedAt = time.Now().Unix()
	if err := s.updateKnowledgeBaseDataDomainIfStatus(ctx, domain, actor, expectedStatus); err != nil {
		if errors.Is(err, errKnowledgeBaseDataDomainCASFailed) {
			// Another request owns the row; do not clobber.
			return nil
		}
		return err
	}
	return nil
}

func (s *semanticModelService) deleteKnowledgeBaseCatalogResources(ctx context.Context, modelID int64) error {
	domain, ok, err := s.getKnowledgeBaseDataDomain(ctx, modelID)
	if err != nil {
		return fmt.Errorf("get knowledge base data domain: %w", err)
	}
	if !ok {
		return nil
	}
	if s.dataDomainService == nil {
		return fmt.Errorf("catalog data-domain service is not configured")
	}
	volumeIDs, err := knowledgeBaseOwnedVolumeIDs(ctx, modelID, domain)
	if err != nil {
		return fmt.Errorf("list knowledge base volumes: %w", err)
	}
	for _, volumeID := range volumeIDs {
		if err := s.dataDomainService.DeleteVolume(ctx, volumeID); err != nil {
			if isKnowledgeBaseCatalogVolumeNotFound(err) {
				continue
			}
			return fmt.Errorf("delete volume %d: %w", volumeID, err)
		}
	}
	if domain.DatabaseID > 0 {
		if err := s.dataDomainService.DeleteDatabase(ctx, domain.DatabaseID); err != nil {
			if isKnowledgeBaseCatalogDatabaseNotFound(err) {
				return nil
			}
			return fmt.Errorf("delete database %d: %w", domain.DatabaseID, err)
		}
	}
	return nil
}

// cleanupKnowledgeBaseCatalogResourcesByIDs deletes catalog volumes/database by
// in-memory IDs when the domain row could not be updated with those IDs (MF-1).
func (s *semanticModelService) cleanupKnowledgeBaseCatalogResourcesByIDs(ctx context.Context, databaseID, rawVolumeID, processedVolumeID int64) error {
	if s.dataDomainService == nil {
		return fmt.Errorf("catalog data-domain service is not configured")
	}
	// Detached short timeout: create request cancel must not leave orphan objects.
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
	defer cancel()
	for _, volumeID := range []int64{rawVolumeID, processedVolumeID} {
		if volumeID <= 0 {
			continue
		}
		if err := s.dataDomainService.DeleteVolume(cleanupCtx, volumeID); err != nil {
			if isKnowledgeBaseCatalogVolumeNotFound(err) {
				continue
			}
			return fmt.Errorf("delete volume %d: %w", volumeID, err)
		}
	}
	if databaseID > 0 {
		if err := s.dataDomainService.DeleteDatabase(cleanupCtx, databaseID); err != nil {
			if isKnowledgeBaseCatalogDatabaseNotFound(err) {
				return nil
			}
			return fmt.Errorf("delete database %d: %w", databaseID, err)
		}
	}
	return nil
}

func isKnowledgeBaseCatalogVolumeNotFound(err error) bool {
	return moi.IsCode(err, common.ErrorCode_NOT_FOUND) ||
		moi.IsCode(err, common.ErrorCode_CATALOG_NOT_FOUND) ||
		moi.IsCode(err, common.ErrorCode_VOLUME_NOT_FOUND)
}

func isKnowledgeBaseCatalogDatabaseNotFound(err error) bool {
	return moi.IsCode(err, common.ErrorCode_NOT_FOUND) ||
		moi.IsCode(err, common.ErrorCode_CATALOG_NOT_FOUND) ||
		moi.IsCode(err, common.ErrorCode_DATABASE_NOT_FOUND)
}

func (s *semanticModelService) ensureKnowledgeBaseRawVolume(ctx context.Context, domain *KnowledgeBaseDataDomain, rawKind, actor string) (int64, error) {
	if domain == nil {
		return 0, fmt.Errorf("knowledge base data domain is required")
	}
	if rawKind == "" || rawKind == kbRawKindDocument {
		return domain.RawVolumeID, nil
	}
	db := ctxutil.TenantDBFrom(ctx)
	if db == nil {
		return 0, fmt.Errorf("tenant db is required")
	}
	var rawVolumeID int64
	err := db.WithContext(ctx).Raw(`SELECT raw_volume_id
		FROM knowledge_base_raw_volumes
		WHERE model_id = ? AND raw_kind = ?`, domain.ModelID, rawKind).Row().Scan(&rawVolumeID)
	if err == nil {
		return rawVolumeID, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}
	rawVolumeID, err = s.dataDomainService.CreateVolume(ctx, domain.DatabaseID, rawVolumeName(rawKind), rawVolumeDescription(rawKind))
	if err != nil {
		if isKnowledgeBaseCatalogVolumeAlreadyExists(err) {
			resolvedID, ok, resolveErr := s.dataDomainService.ResolveVolumeIDByName(ctx, domain.DatabaseID, rawVolumeName(rawKind))
			if resolveErr != nil {
				_ = s.upsertKnowledgeBaseRawVolume(ctx, domain, rawKind, 0, kbEnsureStatusFailed, stringPtr(resolveErr.Error()), actor)
				return 0, fmt.Errorf("resolve existing knowledge base raw volume %s: %w", rawVolumeName(rawKind), resolveErr)
			}
			if ok {
				if err := s.upsertKnowledgeBaseRawVolume(ctx, domain, rawKind, resolvedID, kbEnsureStatusReady, nil, actor); err != nil {
					return 0, err
				}
				return resolvedID, nil
			}
		}
		_ = s.upsertKnowledgeBaseRawVolume(ctx, domain, rawKind, 0, kbEnsureStatusFailed, stringPtr(err.Error()), actor)
		return 0, err
	}
	// Volume exists in Catalog before tenant mapping is written. If mapping fails,
	// delete by in-memory ID so the name can be reused (deleteModel cannot see it).
	if err := s.upsertKnowledgeBaseRawVolume(ctx, domain, rawKind, rawVolumeID, kbEnsureStatusReady, nil, actor); err != nil {
		if cleanupErr := s.cleanupKnowledgeBaseCatalogResourcesByIDs(ctx, 0, rawVolumeID, 0); cleanupErr != nil {
			return 0, fmt.Errorf("%w (also failed to delete unmapped volume %d: %v)", err, rawVolumeID, cleanupErr)
		}
		return 0, err
	}
	return rawVolumeID, nil
}

func isKnowledgeBaseCatalogVolumeAlreadyExists(err error) bool {
	return moi.IsCode(err, common.ErrorCode_ALREADY_EXISTS) ||
		moi.IsCode(err, common.ErrorCode_VOLUME_ALREADY_EXISTS)
}

func isKnowledgeBaseCatalogDatabaseAlreadyExists(err error) bool {
	return errors.Is(err, backendcatalog.ErrDatabaseAlreadyExists) ||
		moi.IsCode(err, common.ErrorCode_ALREADY_EXISTS) ||
		moi.IsCode(err, common.ErrorCode_DATABASE_ALREADY_EXISTS)
}
