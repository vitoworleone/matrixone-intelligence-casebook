package session

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/matrixorigin/matrixflow/moi-backend/pkg/coreclient"
	"github.com/matrixorigin/matrixflow/moi-backend/pkg/ctxutil"
	"github.com/matrixorigin/matrixflow/moi-backend/pkg/i18n"
	sagahelper "github.com/matrixorigin/matrixflow/moi-backend/pkg/saga"

	moi "github.com/matrixflow/moi-core/go-sdk"
	"github.com/matrixflow/moi-core/model/common"
	coresaga "github.com/matrixflow/moi-core/saga"
	sagastore "github.com/matrixflow/moi-core/saga/storage"
	"gorm.io/gorm"
)

// orphanSemanticModelDeleteTimeout bounds compensatory SM delete after a failed
// first domain write. Parent cancel/timeout must not leave an orphan model.
const orphanSemanticModelDeleteTimeout = 10 * time.Second

// knowledgeBaseCreateCleanupTimeout bounds full create-failure cleanup via
// deleteModel (workflows, catalog resources, tenant rows, semantic model).
const knowledgeBaseCreateCleanupTimeout = 30 * time.Second

const (
	knowledgeBaseCleanupSagaName       = "semantic-model-create-cleanup-v1"
	knowledgeBaseCleanupSagaPayloadKey = "operation_payload"
)

// KnowledgeBaseCleanupSagaStore lists durable compensations during restart
// recovery. Production uses the Backend's existing Saga state store.
type KnowledgeBaseCleanupSagaStore interface {
	List(context.Context, sagastore.Filter) ([]*coresaga.SagaState, error)
}

var knowledgeBaseCleanupSagaRuntime struct {
	executor coresaga.Executor
	store    KnowledgeBaseCleanupSagaStore
}

// InitKnowledgeBaseCleanupSaga wires the process-wide Backend Saga executor.
// It must run during startup before knowledge-base create requests are served.
func InitKnowledgeBaseCleanupSaga(executor coresaga.Executor, store KnowledgeBaseCleanupSagaStore) {
	knowledgeBaseCleanupSagaRuntime.executor = executor
	knowledgeBaseCleanupSagaRuntime.store = store
}

type knowledgeBaseCleanupSagaPayload struct {
	WorkspaceID string                   `json:"workspace_id"`
	ModelID     int64                    `json:"model_id"`
	Actor       string                   `json:"actor"`
	Cause       string                   `json:"cause"`
	Domain      *KnowledgeBaseDataDomain `json:"domain,omitempty"`
}

func newKnowledgeBaseCleanupSagaPayload(workspaceID string, modelID int64, domain *KnowledgeBaseDataDomain, cause error, actor string) (knowledgeBaseCleanupSagaPayload, error) {
	if strings.TrimSpace(workspaceID) == "" || modelID <= 0 || strings.TrimSpace(actor) == "" {
		return knowledgeBaseCleanupSagaPayload{}, fmt.Errorf("knowledge base cleanup Saga identity is required")
	}
	if cause == nil {
		cause = fmt.Errorf("knowledge base create failed")
	}
	payload := knowledgeBaseCleanupSagaPayload{
		WorkspaceID: workspaceID,
		ModelID:     modelID,
		Actor:       actor,
		Cause:       cause.Error(),
	}
	if domain != nil {
		clone := *domain
		payload.Domain = &clone
	}
	return payload, nil
}

func (p knowledgeBaseCleanupSagaPayload) sagaID() string {
	digest := sha256.Sum256([]byte(p.WorkspaceID + "\x00" + fmt.Sprint(p.ModelID)))
	return "sm_cleanup_" + hex.EncodeToString(digest[:16])
}

func (p knowledgeBaseCleanupSagaPayload) context() (map[string]any, error) {
	encoded, err := json.Marshal(p)
	if err != nil {
		return nil, fmt.Errorf("marshal knowledge base cleanup Saga payload: %w", err)
	}
	return map[string]any{knowledgeBaseCleanupSagaPayloadKey: string(encoded)}, nil
}

func knowledgeBaseCleanupSagaPayloadFromContext(sagaCtx map[string]any) (knowledgeBaseCleanupSagaPayload, error) {
	raw, _ := sagaCtx[knowledgeBaseCleanupSagaPayloadKey].(string)
	if strings.TrimSpace(raw) == "" {
		return knowledgeBaseCleanupSagaPayload{}, fmt.Errorf("knowledge base cleanup Saga payload is missing")
	}
	var payload knowledgeBaseCleanupSagaPayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return knowledgeBaseCleanupSagaPayload{}, fmt.Errorf("decode knowledge base cleanup Saga payload: %w", err)
	}
	if strings.TrimSpace(payload.WorkspaceID) == "" || payload.ModelID <= 0 || strings.TrimSpace(payload.Actor) == "" {
		return knowledgeBaseCleanupSagaPayload{}, fmt.Errorf("knowledge base cleanup Saga payload is invalid")
	}
	return payload, nil
}

func (s *semanticModelService) knowledgeBaseCleanupSaga() (coresaga.Executor, KnowledgeBaseCleanupSagaStore, error) {
	if s != nil && s.knowledgeBaseCleanupSagaExecutor != nil && s.knowledgeBaseCleanupSagaStore != nil {
		return s.knowledgeBaseCleanupSagaExecutor, s.knowledgeBaseCleanupSagaStore, nil
	}
	if knowledgeBaseCleanupSagaRuntime.executor == nil || knowledgeBaseCleanupSagaRuntime.store == nil {
		return nil, nil, fmt.Errorf("knowledge base cleanup Saga is not configured")
	}
	return knowledgeBaseCleanupSagaRuntime.executor, knowledgeBaseCleanupSagaRuntime.store, nil
}

func (s *semanticModelService) rollbackFailedKnowledgeBaseCreate(
	ctx context.Context,
	client *moi.Client,
	wsID string,
	modelID int64,
	domain *KnowledgeBaseDataDomain,
	cause error,
	actor string,
) error {
	if modelID <= 0 {
		return nil
	}
	payload, err := newKnowledgeBaseCleanupSagaPayload(wsID, modelID, domain, cause, actor)
	if err != nil {
		return err
	}
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), knowledgeBaseCreateCleanupTimeout)
	defer cancel()
	executor, _, err := s.knowledgeBaseCleanupSaga()
	if err != nil {
		return err
	}
	return s.runKnowledgeBaseCleanupSaga(cleanupCtx, client, executor, payload)
}

func (s *semanticModelService) runKnowledgeBaseCleanupSaga(ctx context.Context, client *moi.Client, executor coresaga.Executor, payload knowledgeBaseCleanupSagaPayload) error {
	if client == nil {
		return fmt.Errorf("moi client is required for knowledge base cleanup Saga")
	}
	if executor == nil {
		return fmt.Errorf("knowledge base cleanup Saga executor is required")
	}
	sagaCtx, err := payload.context()
	if err != nil {
		return err
	}
	cleanupStep, err := coresaga.NewStepBuilder().
		WithName("compensate_backend_and_core").
		WithAction(func(stepCtx context.Context, persisted map[string]any) error {
			stepPayload, err := knowledgeBaseCleanupSagaPayloadFromContext(persisted)
			if err != nil {
				return err
			}
			return s.compensateKnowledgeBaseCreate(stepCtx, client, stepPayload)
		}).
		WithCompensation(coresaga.NoOpCompensation).
		WithIdempotent(true).
		Build()
	if err != nil {
		return err
	}
	state, err := sagahelper.RunForwardSaga(ctx, executor, payload.sagaID(), knowledgeBaseCleanupSagaName, []coresaga.Step{cleanupStep}, sagaCtx)
	if err != nil {
		return err
	}
	if state == nil || state.Status != coresaga.SagaStatusCompleted {
		return fmt.Errorf("knowledge base cleanup Saga %s did not complete", payload.sagaID())
	}
	return nil
}

// compensateKnowledgeBaseCreate always attempts both database sides. The
// returned joined error keeps the Saga retryable until both are clean.
func (s *semanticModelService) compensateKnowledgeBaseCreate(ctx context.Context, client *moi.Client, payload knowledgeBaseCleanupSagaPayload) error {
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), knowledgeBaseCreateCleanupTimeout)
	defer cancel()

	var backendErr error
	if payload.Domain != nil && payload.Domain.EnsureStatus != kbEnsureStatusReady {
		partial := *payload.Domain
		partial.EnsureStatus = kbEnsureStatusProvisioning
		partial.LastEnsureError = stringPtr(payload.Cause)
		partial.LastCheckedAt = time.Now().Unix()
		if err := s.updateKnowledgeBaseDataDomainIfStatus(cleanupCtx, &partial, payload.Actor, kbEnsureStatusProvisioning); err != nil {
			backendErr = fmt.Errorf("persist partial resources before rollback: %w", err)
		}
	}
	if backendErr == nil {
		backendErr = s.deleteKnowledgeBaseBackendResources(cleanupCtx, payload.WorkspaceID, payload.ModelID)
	}

	coreErr := client.SemanticModels(payload.WorkspaceID).Delete(cleanupCtx, payload.ModelID)
	if moi.IsCode(coreErr, common.ErrorCode_NOT_FOUND) {
		coreErr = nil
	}
	return errors.Join(backendErr, coreErr)
}

// RecoverKnowledgeBaseCreateCleanupSagas resumes compensations abandoned by a
// failed request or process restart. The caller supplies a workspace tenant DB.
func (s *semanticModelService) RecoverKnowledgeBaseCreateCleanupSagas(ctx context.Context, user coreclient.ExecutionUser) error {
	executor, store, err := s.knowledgeBaseCleanupSaga()
	if err != nil {
		return err
	}
	var states []*coresaga.SagaState
	for _, status := range []coresaga.SagaStatus{coresaga.SagaStatusFailed, coresaga.SagaStatusRunning} {
		items, err := store.List(ctx, sagastore.Filter{SagaName: knowledgeBaseCleanupSagaName, Status: status})
		if err != nil {
			return fmt.Errorf("list knowledge base cleanup Sagas: %w", err)
		}
		states = append(states, items...)
	}
	var recoveryErrors []error
	for _, state := range states {
		payload, err := knowledgeBaseCleanupSagaPayloadFromContext(state.Context)
		if err != nil {
			recoveryErrors = append(recoveryErrors, fmt.Errorf("decode knowledge base cleanup Saga %s: %w", state.SagaID, err))
			continue
		}
		if payload.WorkspaceID != ctxutil.WorkspaceIDFrom(ctx) {
			continue
		}
		err = coreclient.Execute(ctx, user, func(callCtx context.Context, client *moi.Client) error {
			return s.runKnowledgeBaseCleanupSaga(callCtx, client, executor, payload)
		})
		if errors.Is(err, coresaga.ErrSagaAlreadyRunning) {
			continue
		}
		if err != nil {
			recoveryErrors = append(recoveryErrors, fmt.Errorf("resume knowledge base cleanup Saga %s: %w", state.SagaID, err))
		}
	}
	return errors.Join(recoveryErrors...)
}

func (s *semanticModelService) CreateModelWithSources(ctx context.Context, params CreateSemanticModelWithSourcesRequest) (*CreateSemanticModelWithSourcesResponse, error) {
	if err := validateKnowledgeBaseName(params.Name); err != nil {
		return nil, err
	}
	if len(params.Sources) == 0 && len(params.SourceSelections) == 0 {
		return nil, semanticModelSourcesRequiredError()
	}
	if err := validateCreateSemanticModelSources(params.Sources); err != nil {
		return nil, err
	}
	if err := validateSemanticModelSourceSelections(params.SourceSelections); err != nil {
		return nil, err
	}
	if s.dataDomainService == nil {
		return nil, fmt.Errorf("catalog data-domain service is not configured")
	}
	// New knowledge base db/volume always hang under the initialized default catalog.
	// The catalog is backend-owned; resolving it does not use the caller's catalog
	// or database read projection.
	if createSourcesHasStructuredLocalFile(params.Sources) && s.localImportService == nil {
		return nil, fmt.Errorf("knowledge base local file import service is not configured")
	}
	wsID, err := callerWorkspaceID(ctx)
	if err != nil {
		return nil, err
	}
	var response *CreateSemanticModelWithSourcesResponse
	err = coreclient.Execute(ctx, coreclient.FromContext(ctx), func(callCtx context.Context, client *moi.Client) error {
		var callErr error
		response, callErr = s.createModelWithSources(callCtx, client, wsID, params)
		return callErr
	})
	return response, err
}

// createKnowledgeBaseModelShell is the shared create skeleton for empty create and
// create-with-sources: resolve catalog, check physical DB name, create SM, begin domain.
// Name conflicts (catalog DB or semantic model ALREADY_EXISTS) return conflict errors.
// Partial create is rolled back by the caller; create never silently repairs another
// request's failed shell.
func (s *semanticModelService) createKnowledgeBaseModelShell(
	ctx context.Context,
	client *moi.Client,
	wsID, name, description string,
	tables, files json.RawMessage,
) (*moi.SemanticModel, *KnowledgeBaseDataDomain, error) {
	catalogID, err := s.resolveDefaultCatalogID(ctx)
	if err != nil {
		return nil, nil, err
	}
	if err := s.ensureKnowledgeBaseDatabaseNameAvailable(ctx, catalogID, name, nil); err != nil {
		return nil, nil, err
	}
	model, err := client.SemanticModels(wsID).Create(ctx, &moi.SemanticModelUpsertRequest{
		Name:        name,
		Description: description,
		Tables:      tables,
		Files:       files,
	})
	if err != nil {
		if moi.IsCode(err, common.ErrorCode_ALREADY_EXISTS) {
			return nil, nil, semanticModelNameExistsError()
		}
		return nil, nil, err
	}
	domain, err := s.beginKnowledgeBaseDataDomain(ctx, model.ID, catalogID, kbEnsureStatusProvisioning, semanticModelActor(ctx))
	if err != nil {
		// First domain write failed after SM create: remove the orphan model so
		// the same name can be retried without ALREADY_EXISTS.
		if deleteErr := deleteOrphanSemanticModel(ctx, client, wsID, model.ID); deleteErr != nil {
			return nil, nil, fmt.Errorf("%w (also failed to delete orphan semantic model %d: %v)", err, model.ID, deleteErr)
		}
		return nil, nil, err
	}
	return model, domain, nil
}

// deleteOrphanSemanticModel removes an SM created before domain begin failed.
// Parent cancel/timeout must not skip compensation (MF-2).
func deleteOrphanSemanticModel(ctx context.Context, client *moi.Client, wsID string, modelID int64) error {
	if client == nil {
		return fmt.Errorf("moi client is required")
	}
	deleteCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), orphanSemanticModelDeleteTimeout)
	defer cancel()
	return client.SemanticModels(wsID).Delete(deleteCtx, modelID)
}

func (s *semanticModelService) CreateModel(ctx context.Context, params CreateSemanticModelRequest) (*SemanticModelInfo, error) {
	// Keep the semantic model name identical to the catalog/MO database name.
	databaseName, err := knowledgeBaseDatabaseName(params.Name)
	if err != nil {
		return nil, err
	}
	if s.dataDomainService == nil {
		return nil, fmt.Errorf("catalog data-domain service is not configured")
	}
	params.Name = databaseName
	wsID, err := callerWorkspaceID(ctx)
	if err != nil {
		return nil, err
	}
	var response *SemanticModelInfo
	err = coreclient.Execute(ctx, coreclient.FromContext(ctx), func(callCtx context.Context, client *moi.Client) error {
		var callErr error
		response, callErr = s.createModel(callCtx, client, wsID, params)
		return callErr
	})
	return response, err
}

// CreateEmptyModel initializes the data-side knowledge base without creating
// sources, jobs, uploads, or RAG workflows.
func (s *semanticModelService) CreateEmptyModel(ctx context.Context, params CreateEmptySemanticModelRequest) (*CreateEmptySemanticModelResponse, error) {
	if err := validateKnowledgeBaseName(params.Name); err != nil {
		return nil, err
	}
	if s.dataDomainService == nil {
		return nil, fmt.Errorf("catalog data-domain service is not configured")
	}
	wsID, err := callerWorkspaceID(ctx)
	if err != nil {
		return nil, err
	}
	var response *CreateEmptySemanticModelResponse
	err = coreclient.Execute(ctx, coreclient.FromContext(ctx), func(callCtx context.Context, client *moi.Client) error {
		var callErr error
		response, callErr = s.createEmptyModel(callCtx, client, wsID, params)
		return callErr
	})
	return response, err
}

func (s *semanticModelService) createEmptyModel(ctx context.Context, client *moi.Client, wsID string, params CreateEmptySemanticModelRequest) (*CreateEmptySemanticModelResponse, error) {
	imageEmbeddingBackendID, err := resolveKnowledgeBaseCreateEmbeddingConfig(ctx, client, wsID, params.ImageIndexEnabled)
	if err != nil {
		return nil, err
	}
	databaseName, err := knowledgeBaseDatabaseName(params.Name)
	if err != nil {
		return nil, err
	}
	actor := semanticModelActor(ctx)
	model, domain, err := s.createKnowledgeBaseModelShell(
		ctx, client, wsID, databaseName, params.Description,
		json.RawMessage("[]"), json.RawMessage(`{"file_ids":[]}`),
	)
	if err != nil {
		return nil, err
	}
	response, err := s.completeEmptyModel(ctx, client, wsID, model, domain, params, imageEmbeddingBackendID, actor)
	if err != nil {
		if rollbackErr := s.rollbackFailedKnowledgeBaseCreate(ctx, client, wsID, model.ID, domain, err, actor); rollbackErr != nil {
			return nil, fmt.Errorf("%w (also failed to roll back create: %v)", err, rollbackErr)
		}
		return nil, err
	}
	return response, nil
}

func (s *semanticModelService) completeEmptyModel(ctx context.Context, client *moi.Client, wsID string, model *moi.SemanticModel, domain *KnowledgeBaseDataDomain, params CreateEmptySemanticModelRequest, imageEmbeddingBackendID, actor string) (*CreateEmptySemanticModelResponse, error) {
	if model == nil {
		return nil, semanticModelNotFoundError()
	}
	if domain == nil {
		return nil, knowledgeBaseDataDomainNotFoundError()
	}
	provisionCtx := ctxutil.WithKnowledgeBaseProvisioning(ctx, domain.CatalogID)
	if err := s.prepareKnowledgeBaseDataDomainResourcesForCreate(provisionCtx, domain, model.Name, params.Description, actor); err != nil {
		return nil, err
	}
	filesBase, err := semanticModelCreateFilesBaseWithFixedIndex(nil, model.ID, params.ImageIndexEnabled, imageEmbeddingBackendID)
	if err != nil {
		return nil, err
	}
	files, err := appendSemanticModelFiles(filesBase, model.ID, nil)
	if err != nil {
		return nil, err
	}
	if _, err := parseKBVectorBinding(files); err != nil {
		return nil, err
	}
	if _, err := client.SemanticModels(wsID).Update(ctx, model.ID, &moi.SemanticModelUpsertRequest{
		Name:        model.Name,
		Description: params.Description,
		Tables:      json.RawMessage("[]"),
		Files:       files,
	}); err != nil {
		return nil, fmt.Errorf("update empty knowledge base: %w", err)
	}
	model.Tables = json.RawMessage("[]")
	model.Files = files
	if err := s.finalizeKnowledgeBaseDataDomainCreate(ctx, domain, actor); err != nil {
		return nil, err
	}
	return &CreateEmptySemanticModelResponse{Model: toSemanticModelInfo(model), DataDomain: domain}, nil
}

func (s *semanticModelService) createModel(ctx context.Context, client *moi.Client, wsID string, params CreateSemanticModelRequest) (*SemanticModelInfo, error) {
	actor := semanticModelActor(ctx)
	model, domain, err := s.createKnowledgeBaseModelShell(
		ctx, client, wsID, params.Name, params.Description, params.Tables, params.Files,
	)
	if err != nil {
		return nil, err
	}
	provisionCtx := ctxutil.WithKnowledgeBaseProvisioning(ctx, domain.CatalogID)
	err = s.prepareKnowledgeBaseDataDomainResourcesForCreate(provisionCtx, domain, params.Name, params.Description, actor)
	if err == nil {
		err = s.finalizeKnowledgeBaseDataDomainCreate(ctx, domain, actor)
	}
	if err != nil {
		if rollbackErr := s.rollbackFailedKnowledgeBaseCreate(ctx, client, wsID, model.ID, domain, err, actor); rollbackErr != nil {
			return nil, fmt.Errorf("%w (also failed to roll back create: %v)", err, rollbackErr)
		}
		return nil, err
	}
	return toSemanticModelInfo(model), nil
}

func (s *semanticModelService) createModelWithSources(ctx context.Context, client *moi.Client, wsID string, params CreateSemanticModelWithSourcesRequest) (*CreateSemanticModelWithSourcesResponse, error) {
	imageEmbeddingBackendID, err := resolveKnowledgeBaseCreateEmbeddingConfig(ctx, client, wsID, params.ImageIndexEnabled)
	if err != nil {
		return nil, err
	}
	selectionSources, err := s.expandSemanticModelSourceSelections(ctx, client, wsID, 0, params.SourceSelections, params.Sources)
	if err != nil {
		return nil, err
	}
	params.SelectionSources = selectionSources
	allSources := appendSourceRequests(params.Sources, selectionSources)
	if len(allSources) == 0 {
		return nil, semanticModelSourcesRequiredError()
	}
	// Re-check after selection expand: explicit + expanded catalog_file may
	// still collide on file_id across volumes in one create request.
	if err := validateCreateSemanticModelSources(allSources); err != nil {
		return nil, err
	}
	databaseName, err := knowledgeBaseDatabaseName(params.Name)
	if err != nil {
		return nil, err
	}
	// Keep the semantic model name identical to the catalog/MO database name.
	params.Name = databaseName
	actor := semanticModelActor(ctx)
	model, domain, err := s.createKnowledgeBaseModelShell(
		ctx, client, wsID, params.Name, params.Description,
		json.RawMessage("[]"), json.RawMessage(`{"file_ids":[]}`),
	)
	if err != nil {
		return nil, err
	}
	response, err := s.completeCreateModelWithSources(ctx, client, wsID, model, domain, params, imageEmbeddingBackendID, actor)
	if err != nil {
		if rollbackErr := s.rollbackFailedKnowledgeBaseCreate(ctx, client, wsID, model.ID, domain, err, actor); rollbackErr != nil {
			return nil, fmt.Errorf("%w (also failed to roll back create: %v)", err, rollbackErr)
		}
		return nil, err
	}
	return response, nil
}

func resolveKnowledgeBaseCreateEmbeddingConfig(ctx context.Context, c *moi.Client, wsID string, imageIndexEnabled bool) (string, error) {
	response, err := c.Embeddings(wsID).ListModels(ctx)
	if err != nil {
		return "", knowledgeBaseEmbeddingCapabilityUnavailableError(err)
	}
	textModelAvailable := false
	imageEmbeddingBackendID := ""
	for _, model := range response.Models {
		modelName := strings.TrimSpace(model.Model)
		if modelName == kbDefaultEmbeddingModel && model.BackendID != 0 {
			textModelAvailable = true
		}
		if imageEmbeddingBackendID == "" && modelName == kbDefaultImageEmbeddingModel && model.BackendID != 0 {
			imageEmbeddingBackendID = strconv.FormatInt(model.BackendID, 10)
		}
	}
	if !textModelAvailable {
		return "", knowledgeBaseEmbeddingModelNotAvailableError(kbDefaultEmbeddingModel)
	}
	if imageIndexEnabled && imageEmbeddingBackendID == "" {
		return "", knowledgeBaseEmbeddingModelNotAvailableError(kbDefaultImageEmbeddingModel)
	}
	if !imageIndexEnabled {
		return "", nil
	}
	return imageEmbeddingBackendID, nil
}

func (s *semanticModelService) completeCreateModelWithSources(ctx context.Context, c *moi.Client, wsID string, model *moi.SemanticModel, domain *KnowledgeBaseDataDomain, params CreateSemanticModelWithSourcesRequest, imageEmbeddingBackendID string, actor string) (*CreateSemanticModelWithSourcesResponse, error) {
	if model == nil {
		return nil, semanticModelNotFoundError()
	}
	if domain == nil {
		return nil, knowledgeBaseDataDomainNotFoundError()
	}
	allSources := appendSourceRequests(params.Sources, params.SelectionSources)
	provisionCtx := ctxutil.WithKnowledgeBaseProvisioning(ctx, domain.CatalogID)
	if err := s.prepareKnowledgeBaseDataDomainResourcesForCreate(provisionCtx, domain, params.Name, params.Description, actor); err != nil {
		return nil, err
	}
	modelID := model.ID
	createFilesBase, err := semanticModelCreateFilesBaseWithFixedIndex(params.Files, modelID, params.ImageIndexEnabled, imageEmbeddingBackendID)
	if err != nil {
		return nil, err
	}
	filesBaseJSON, err := appendSemanticModelFiles(createFilesBase, modelID, nil)
	if err != nil {
		return nil, err
	}
	if _, err := parseKBVectorBinding(filesBaseJSON); err != nil {
		return nil, err
	}
	if hasDocumentParsingSources(allSources) {
		if err := s.ensureKnowledgeBaseRAGWorkflow(ctx, wsID, modelID, params.Name, params.Description, domain, filesBaseJSON); err != nil {
			return nil, fmt.Errorf("ensure knowledge base document parsing workflow: %w", err)
		}
	}

	createResult, err := s.createKnowledgeBaseSourceMetadataIntents(provisionCtx, c, wsID, modelID, domain, allSources, actor, false)
	if err != nil {
		return nil, err
	}

	filesJSON, err := appendSemanticModelFiles(filesBaseJSON, modelID, createResult.fileIDs)
	if err != nil {
		return nil, err
	}
	tablesJSON, err := json.Marshal(createResult.tables)
	if err != nil {
		return nil, fmt.Errorf("marshal semantic model tables: %w", err)
	}
	if _, err := c.SemanticModels(wsID).Update(ctx, modelID, &moi.SemanticModelUpsertRequest{
		Name:        params.Name,
		Description: params.Description,
		Tables:      tablesJSON,
		Files:       filesJSON,
	}); err != nil {
		err = s.markCreateKnowledgeBaseSourcesFailed(ctx, createResult.records, createResult.jobs, err, actor)
		return nil, fmt.Errorf("update semantic model sources: %w", err)
	}
	model.Files = filesJSON
	model.Tables = tablesJSON
	if err := s.runCreateKnowledgeBaseStructuredLocalImports(ctx, &createResult, actor); err != nil {
		return nil, err
	}
	if err := s.finalizeKnowledgeBaseDataDomainCreate(ctx, domain, actor); err != nil {
		return nil, err
	}
	responseRecords := createResult.records
	responseJobs := sanitizeKnowledgeBaseSourceJobRunsForResponse(createResult.jobs)
	responseSources := applyKnowledgeBaseSourceJobStatus(semanticModelSourcesFromRecords(responseRecords), responseJobs)

	return &CreateSemanticModelWithSourcesResponse{
		Model:      toSemanticModelInfo(model),
		DataDomain: domain,
		Sources:    responseSources,
		Jobs:       responseJobs,
	}, nil
}

func (s *semanticModelService) AppendModelSources(ctx context.Context, params AppendSemanticModelSourcesRequest) (*AppendSemanticModelSourcesResponse, error) {
	modelID := int64(params.ModelID)
	if modelID <= 0 {
		return nil, serviceError(ErrCodeBadRequest, i18n.KeySessionInvalidModelID, nil)
	}
	if len(params.Sources) == 0 && len(params.SourceSelections) == 0 {
		return nil, semanticModelSourcesRequiredError()
	}
	if err := validateCreateSemanticModelSources(params.Sources); err != nil {
		return nil, err
	}
	if err := validateSemanticModelSourceSelections(params.SourceSelections); err != nil {
		return nil, err
	}
	if s.dataDomainService == nil {
		return nil, fmt.Errorf("catalog data-domain service is not configured")
	}
	if createSourcesHasStructuredLocalFile(params.Sources) && s.localImportService == nil {
		return nil, fmt.Errorf("knowledge base local file import service is not configured")
	}
	wsID, err := callerWorkspaceID(ctx)
	if err != nil {
		return nil, err
	}
	var response *AppendSemanticModelSourcesResponse
	err = coreclient.Execute(ctx, coreclient.FromContext(ctx), func(callCtx context.Context, client *moi.Client) error {
		var callErr error
		response, callErr = s.appendModelSources(callCtx, client, wsID, modelID, params)
		return callErr
	})
	return response, err
}

func (s *semanticModelService) appendModelSources(ctx context.Context, c *moi.Client, wsID string, modelID int64, params AppendSemanticModelSourcesRequest) (*AppendSemanticModelSourcesResponse, error) {
	selectionSources, err := s.expandSemanticModelSourceSelections(ctx, c, wsID, modelID, params.SourceSelections, params.Sources)
	if err != nil {
		return nil, err
	}
	params.SelectionSources = selectionSources
	allSources := appendSourceRequests(params.Sources, selectionSources)
	if len(allSources) == 0 {
		return nil, semanticModelSourcesRequiredError()
	}
	// Re-check after selection expand for the same file_id collision rule.
	if err := validateCreateSemanticModelSources(allSources); err != nil {
		return nil, err
	}
	actor := semanticModelActor(ctx)
	initialModel, err := c.SemanticModels(wsID).Get(ctx, modelID)
	if err != nil {
		return nil, err
	}
	initialFilesBaseJSON, err := appendSemanticModelFiles(initialModel.Files, modelID, nil)
	if err != nil {
		return nil, err
	}
	if _, err := parseKBVectorBinding(initialFilesBaseJSON); err != nil {
		return nil, err
	}
	domain, err := s.ensureAppendKnowledgeBaseDataDomain(ctx, wsID, modelID, initialModel.Name, initialModel.Description, allSources, actor, initialFilesBaseJSON)
	if err != nil {
		return nil, err
	}

	db := ctxutil.TenantDBFrom(ctx)
	if db == nil {
		return nil, fmt.Errorf("tenant db is required")
	}
	var sourceResult createKnowledgeBaseSourcesResult
	if err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txCtx := ctxutil.WithTenantDB(ctx, tx)
		if err := s.lockKnowledgeBaseDataDomainForAppend(txCtx, modelID); err != nil {
			return err
		}
		model, err := c.SemanticModels(wsID).Get(txCtx, modelID)
		if err != nil {
			return err
		}
		filesBaseJSON, err := appendSemanticModelFiles(model.Files, modelID, nil)
		if err != nil {
			return err
		}
		if _, err := parseKBVectorBinding(filesBaseJSON); err != nil {
			return err
		}
		sourceResult, err = s.createKnowledgeBaseSourceMetadataIntentsInTx(txCtx, c, wsID, modelID, domain, allSources, actor, true)
		if err != nil {
			return err
		}
		createdSources := semanticModelSourcesFromRecords(sourceResult.records)
		createdJobs := sourceResult.jobs
		rollbackCreatedSources := func(cause error) error {
			return s.rollbackCreatedKnowledgeBaseSources(txCtx, createdSources, createdJobs, nil, cause)
		}
		filesJSON, err := appendSemanticModelFiles(filesBaseJSON, modelID, sourceResult.fileIDs)
		if err != nil {
			return rollbackCreatedSources(err)
		}
		tablesJSON, err := appendSemanticModelTables(model.Tables, sourceResult.tables)
		if err != nil {
			return rollbackCreatedSources(err)
		}
		if _, err := c.SemanticModels(wsID).Update(txCtx, modelID, &moi.SemanticModelUpsertRequest{
			Name:        model.Name,
			Description: model.Description,
			Tables:      tablesJSON,
			Files:       filesJSON,
		}); err != nil {
			return rollbackCreatedSources(fmt.Errorf("update semantic model sources: %w", err))
		}
		return nil
	}); err != nil {
		return nil, err
	}
	if err := s.runCreateKnowledgeBaseStructuredLocalImports(ctx, &sourceResult, actor); err != nil {
		return nil, err
	}
	records, err := s.listKnowledgeBaseSources(ctx, modelID)
	if err != nil {
		return nil, fmt.Errorf("list knowledge base sources: %w", err)
	}
	sources := make([]SemanticModelSource, 0, len(records))
	for _, record := range records {
		sources = append(sources, sourceRecordToSemanticModelSource(record))
	}
	jobs, err := s.listKnowledgeBaseSourceJobRuns(ctx, modelID)
	if err != nil {
		return nil, fmt.Errorf("list knowledge base source jobs: %w", err)
	}
	jobs, err = s.enrichKnowledgeBaseSourceJobRunsFromLinkedJobs(ctx, c, wsID, jobs)
	if err != nil {
		return nil, err
	}
	jobs = sanitizeKnowledgeBaseSourceJobRunsForResponse(jobs)
	sources = applyKnowledgeBaseSourceJobStatus(sources, jobs)
	domain, ok, err := s.getKnowledgeBaseDataDomain(ctx, modelID)
	if err != nil {
		return nil, fmt.Errorf("get knowledge base data domain: %w", err)
	}
	if !ok {
		return nil, knowledgeBaseDataDomainNotFoundError()
	}
	out := &AppendSemanticModelSourcesResponse{
		DataDomain: domain,
		Sources:    sources,
		Jobs:       jobs,
	}
	if out != nil && len(sourceResult.jobs) > 0 {
		updatedJobs := make(map[string]KnowledgeBaseSourceJobRun, len(sourceResult.jobs))
		for _, job := range sourceResult.jobs {
			updatedJobs[job.JobID] = job
		}
		for i := range out.Jobs {
			if job, ok := updatedJobs[out.Jobs[i].JobID]; ok {
				out.Jobs[i] = job
			}
		}
		out.Jobs = sanitizeKnowledgeBaseSourceJobRunsForResponse(out.Jobs)
		out.Sources = applyKnowledgeBaseSourceJobStatus(out.Sources, out.Jobs)
	}
	return out, nil
}

func (s *semanticModelService) ensureAppendKnowledgeBaseDataDomain(ctx context.Context, wsID string, modelID int64, name, description string, sources []CreateSemanticModelSourceRequest, actor string, files json.RawMessage) (*KnowledgeBaseDataDomain, error) {
	// Same catalog/db/volume provision path as CreateModel / CreateModelWithSources.
	domain, err := s.provisionKnowledgeBaseDataDomain(ctx, modelID, name, description, actor)
	if err != nil {
		return nil, err
	}

	// Empty-create then first document append never deployed a workflow. Keep
	// Require fail-closed for present-but-broken workflows; only deploy when
	// the deterministic workflow id is missing.
	if hasDocumentParsingSources(sources) {
		if s.workflowService == nil {
			return nil, fmt.Errorf("workflow service is not configured")
		}
		workflowID := knowledgeBaseWorkflowID(wsID, modelID)
		if err := s.workflowService.RequireKnowledgeBaseWorkflow(ctx, workflowID); err != nil {
			if !moi.IsNotFound(err) {
				return nil, fmt.Errorf("require existing knowledge base workflow %q: %w", workflowID, err)
			}
			if err := s.ensureKnowledgeBaseRAGWorkflow(ctx, wsID, modelID, name, description, domain, files); err != nil {
				return nil, fmt.Errorf("ensure knowledge base document parsing workflow: %w", err)
			}
		}
	}
	return domain, nil
}
