package session

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/matrixorigin/matrixflow/moi-backend/pkg/coreclient"
	"github.com/matrixorigin/matrixflow/moi-backend/pkg/ctxutil"
	"github.com/matrixorigin/matrixflow/moi-backend/pkg/i18n"

	moi "github.com/matrixflow/moi-core/go-sdk"
	"github.com/matrixflow/moi-core/model/common"
	coresaga "github.com/matrixflow/moi-core/saga"
	"gorm.io/gorm"
)

// SemanticModelActionAuthorizer re-checks a Core action for non-HTTP service
// paths (for example deferred catalog_file RAG dispatch). Production wires
// iampep.ServiceActionAuthorizer.
//
// Implementations must honor the platform privilege-class gate: workspace owner
// and superadmin skip Core Authorize; ordinary Effective Roles (including
// ordinary admin Role) re-enter Core PDP.
type SemanticModelActionAuthorizer interface {
	// ReauthorizeAction revalidates the caller's verified Effective Role against
	// actionID on resourceID for ordinary roles. Privilege-class principals
	// (workspace owner / superadmin) skip Core. Fail closed on deny or Core
	// unavailable for ordinary roles.
	ReauthorizeAction(ctx context.Context, workspaceID, actionID, resourceType, resourceID string) (context.Context, error)
}

// SemanticModelCanonicalVolumeResolver maps a recorded volume ID (which may be a
// child volume) to the IAM-grantable canonical root volume. Production wires
// catalogh.NewCoreCanonicalVolumeResolver. Deferred catalog_file volume.read
// must authorize the root; RawVolumeID remains the workflow file location.
type SemanticModelCanonicalVolumeResolver interface {
	ResolveCanonicalRootVolume(ctx context.Context, workspaceID string, volumeID int64) (int64, error)
}

type semanticModelService struct {
	dataDomainService                SemanticModelCatalogDataDomainService
	fileService                      SemanticModelCatalogFileService
	localImportService               SemanticModelLocalFileImportService
	workflowTemplateService          SemanticModelWorkflowTemplateService
	workflowService                  SemanticModelWorkflowService
	actionAuthorizer                 SemanticModelActionAuthorizer
	volumeResolver                   SemanticModelCanonicalVolumeResolver
	openWorkspaceDB                  func(ctx context.Context, wsID, dbName string) (*sql.DB, error)
	knowledgeBaseCleanupSagaExecutor coresaga.Executor
	knowledgeBaseCleanupSagaStore    KnowledgeBaseCleanupSagaStore
}

// NewSemanticModelService creates a new SemanticModelService.
func NewSemanticModelService() SemanticModelService {
	return &semanticModelService{}
}

func NewSemanticModelServiceWithDependencies(dataDomainService SemanticModelCatalogDataDomainService, fileService SemanticModelCatalogFileService) SemanticModelService {
	return NewSemanticModelServiceWithKnowledgeBaseDependencies(dataDomainService, fileService, nil)
}

func NewSemanticModelServiceWithKnowledgeBaseDependencies(dataDomainService SemanticModelCatalogDataDomainService, fileService SemanticModelCatalogFileService, localImportService SemanticModelLocalFileImportService) SemanticModelService {
	return NewSemanticModelServiceWithKnowledgeBaseRuntimeDependencies(dataDomainService, fileService, localImportService, nil, nil)
}

func NewSemanticModelServiceWithKnowledgeBaseRuntimeDependencies(dataDomainService SemanticModelCatalogDataDomainService, fileService SemanticModelCatalogFileService, localImportService SemanticModelLocalFileImportService, workflowTemplateService SemanticModelWorkflowTemplateService, workflowService SemanticModelWorkflowService) SemanticModelService {
	return &semanticModelService{
		dataDomainService:       dataDomainService,
		fileService:             fileService,
		localImportService:      localImportService,
		workflowTemplateService: workflowTemplateService,
		workflowService:         workflowService,
	}
}

// SetActionAuthorizer wires Core reauthorization used before deferred
// catalog_file RAG dispatch reads the external source volume.
func (s *semanticModelService) SetActionAuthorizer(authorizer SemanticModelActionAuthorizer) {
	if s == nil {
		return
	}
	s.actionAuthorizer = authorizer
}

// SetSemanticModelActionAuthorizer wires production IAM onto the concrete
// service returned by the constructors above (SemanticModelService interface).
// Production: cmd/main after NewSemanticModelServiceWithKnowledgeBaseRuntimeDependencies.
func SetSemanticModelActionAuthorizer(svc SemanticModelService, authorizer SemanticModelActionAuthorizer) {
	if concrete, ok := svc.(*semanticModelService); ok {
		concrete.SetActionAuthorizer(authorizer)
	}
}

// SetCanonicalVolumeResolver wires root-volume resolution for deferred
// catalog_file volume.read. IAM grants attach to the canonical root, not a
// child volume ID stored on the source record.
func (s *semanticModelService) SetCanonicalVolumeResolver(resolver SemanticModelCanonicalVolumeResolver) {
	if s == nil {
		return
	}
	s.volumeResolver = resolver
}

// SetSemanticModelCanonicalVolumeResolver wires production Core root resolution
// onto the concrete service returned by the constructors above.
// Production: cmd/main after NewSemanticModelServiceWithKnowledgeBaseRuntimeDependencies.
func SetSemanticModelCanonicalVolumeResolver(svc SemanticModelService, resolver SemanticModelCanonicalVolumeResolver) {
	if concrete, ok := svc.(*semanticModelService); ok {
		concrete.SetCanonicalVolumeResolver(resolver)
	}
}

func (s *semanticModelService) ListModels(ctx context.Context, params ListSemanticModelsRequest) (*ListSemanticModelsResponse, error) {
	wsID, err := callerWorkspaceID(ctx)
	if err != nil {
		return nil, err
	}
	var response *ListSemanticModelsResponse
	err = coreclient.Execute(ctx, coreclient.FromContext(ctx), func(callCtx context.Context, client *moi.Client) error {
		var callErr error
		response, callErr = s.listModels(callCtx, client, wsID, params)
		return callErr
	})
	return response, err
}

func (s *semanticModelService) listModels(ctx context.Context, client *moi.Client, wsID string, params ListSemanticModelsRequest) (*ListSemanticModelsResponse, error) {
	opts := []moi.ListOption{}
	if params.PageSize > 0 {
		opts = append(opts, moi.WithPageSize(int32(params.PageSize)))
	}
	if params.PageToken != "" {
		opts = append(opts, moi.WithPageToken(params.PageToken))
	}
	if params.Search != "" {
		opts = append(opts, moi.WithSearch(params.Search))
	}
	for _, tag := range params.Tags {
		tag = strings.TrimSpace(tag)
		if tag != "" {
			opts = append(opts, moi.WithTag(tag))
		}
	}
	resp, err := client.SemanticModels(wsID).List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	items := make([]*SemanticModelInfo, 0, len(resp.Items))
	for _, m := range resp.Items {
		items = append(items, toSemanticModelInfo(m))
	}
	if err := s.enrichSemanticModelSourceCounts(ctx, items); err != nil {
		return nil, err
	}
	return &ListSemanticModelsResponse{
		Items:         items,
		Total:         resp.Total,
		NextPageToken: resp.NextPageToken,
	}, nil
}

func (s *semanticModelService) ListModelTags(ctx context.Context, params ListSemanticModelsRequest) (*ListSemanticModelTagsResponse, error) {
	wsID, err := callerWorkspaceID(ctx)
	if err != nil {
		return nil, err
	}
	var response *ListSemanticModelTagsResponse
	err = coreclient.Execute(ctx, coreclient.FromContext(ctx), func(callCtx context.Context, client *moi.Client) error {
		var callErr error
		response, callErr = s.listModelTags(callCtx, client, wsID, params)
		return callErr
	})
	return response, err
}

func (s *semanticModelService) listModelTags(ctx context.Context, client *moi.Client, wsID string, params ListSemanticModelsRequest) (*ListSemanticModelTagsResponse, error) {
	opts := []moi.ListOption{}
	if params.Search != "" {
		opts = append(opts, moi.WithSearch(params.Search))
	}
	resp, err := client.SemanticModels(wsID).ListTags(ctx, opts...)
	if err != nil {
		return nil, err
	}
	items := make([]SemanticModelTagStat, 0, len(resp.Items))
	for _, item := range resp.Items {
		items = append(items, SemanticModelTagStat{Tag: item.Tag, Count: item.Count})
	}
	return &ListSemanticModelTagsResponse{Items: items}, nil
}

func (s *semanticModelService) ListModelsByIDs(ctx context.Context, ids []int64, params ListSemanticModelsRequest) (*ListSemanticModelsResponse, error) {
	wsID, err := callerWorkspaceID(ctx)
	if err != nil {
		return nil, err
	}
	var response *ListSemanticModelsResponse
	err = coreclient.Execute(ctx, coreclient.FromContext(ctx), func(callCtx context.Context, client *moi.Client) error {
		var callErr error
		response, callErr = s.listModelsByIDs(callCtx, client, wsID, ids, params)
		return callErr
	})
	return response, err
}

func (s *semanticModelService) listModelsByIDs(ctx context.Context, client *moi.Client, wsID string, ids []int64, params ListSemanticModelsRequest) (*ListSemanticModelsResponse, error) {
	sortedIDs := append([]int64(nil), ids...)
	sort.Slice(sortedIDs, func(i, j int) bool { return sortedIDs[i] > sortedIDs[j] })

	items := make([]*SemanticModelInfo, 0, len(sortedIDs))
	for _, id := range sortedIDs {
		model, err := client.SemanticModels(wsID).Get(ctx, id)
		if moi.IsCode(err, common.ErrorCode_NOT_FOUND) {
			continue
		}
		if err != nil {
			return nil, err
		}
		info := toSemanticModelInfo(model)
		if semanticModelMatchesSearch(info, params.Search) && semanticModelMatchesTags(info, params.Tags) {
			items = append(items, info)
		}
	}

	total := int64(len(items))
	start := 0
	if params.PageToken != "" {
		offset, err := decodeSemanticModelPageToken(params.PageToken)
		if err != nil {
			return nil, invalidPageTokenError()
		}
		start = int(offset)
	}
	if start > len(items) {
		start = len(items)
	}

	pageSize := params.PageSize
	if pageSize <= 0 {
		pageSize = len(items)
	}
	end := len(items)
	nextPageToken := ""
	if pageSize > 0 && start+pageSize < len(items) {
		end = start + pageSize
		nextPageToken = encodeSemanticModelPageToken(int64(end))
	}

	pageItems := items[start:end]
	if err := s.enrichSemanticModelSourceCounts(ctx, pageItems); err != nil {
		return nil, err
	}
	return &ListSemanticModelsResponse{
		Items:         pageItems,
		Total:         total,
		NextPageToken: nextPageToken,
	}, nil
}

type semanticModelPageTokenData struct {
	Offset int64 `json:"offset"`
}

func encodeSemanticModelPageToken(offset int64) string {
	data, _ := json.Marshal(semanticModelPageTokenData{Offset: offset})
	return base64.StdEncoding.EncodeToString(data)
}

func decodeSemanticModelPageToken(token string) (int64, error) {
	if token == "" {
		return 0, nil
	}
	data, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		return 0, err
	}
	var decoded semanticModelPageTokenData
	if err := json.Unmarshal(data, &decoded); err != nil {
		return 0, err
	}
	if decoded.Offset < 0 {
		return 0, fmt.Errorf("invalid offset in page token: %d", decoded.Offset)
	}
	return decoded.Offset, nil
}

func semanticModelMatchesSearch(model *SemanticModelInfo, search string) bool {
	if search == "" {
		return true
	}
	if model == nil {
		return false
	}
	return strings.Contains(model.Name, search) || strings.Contains(model.Description, search)
}

func semanticModelMatchesTags(model *SemanticModelInfo, tags []string) bool {
	filterTags := make([]string, 0, len(tags))
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag != "" {
			filterTags = append(filterTags, tag)
		}
	}
	if len(filterTags) == 0 {
		return true
	}
	if model == nil {
		return false
	}
	var files semanticModelFilesPayload
	if len(model.Files) == 0 || json.Unmarshal(model.Files, &files) != nil {
		return false
	}
	modelTags := make(map[string]struct{}, len(files.Tags))
	for _, tag := range files.Tags {
		tag = strings.TrimSpace(tag)
		if tag != "" {
			modelTags[tag] = struct{}{}
		}
	}
	for _, tag := range filterTags {
		if _, ok := modelTags[tag]; ok {
			return true
		}
	}
	return false
}

func semanticModelUpdateSourcesUnchanged(current *moi.SemanticModel, params UpdateSemanticModelRequest) (bool, error) {
	if current == nil {
		return false, nil
	}
	if params.Tables != nil {
		var currentTables, nextTables any
		if len(current.Tables) > 0 && string(current.Tables) != "null" {
			if err := json.Unmarshal(current.Tables, &currentTables); err != nil {
				return false, fmt.Errorf("decode existing semantic model tables: %w", err)
			}
		}
		if err := json.Unmarshal(params.Tables, &nextTables); err != nil {
			return false, fmt.Errorf("decode semantic model tables update: %w", err)
		}
		if !reflect.DeepEqual(currentTables, nextTables) {
			return false, nil
		}
	}
	if params.Files != nil {
		currentFiles := map[string]any{}
		nextFiles := map[string]any{}
		if len(current.Files) > 0 && string(current.Files) != "null" {
			if err := json.Unmarshal(current.Files, &currentFiles); err != nil {
				return false, fmt.Errorf("decode existing semantic model files: %w", err)
			}
		}
		if err := json.Unmarshal(params.Files, &nextFiles); err != nil {
			return false, semanticModelFilesInvalidError()
		}
		delete(currentFiles, "tags")
		delete(nextFiles, "tags")
		if !reflect.DeepEqual(currentFiles, nextFiles) {
			return false, nil
		}
	}
	return true, nil
}

func (s *semanticModelService) updateModelWithoutSourceMutation(ctx context.Context, client *moi.Client, wsID string, params UpdateSemanticModelRequest) error {
	modelID := int64(params.ModelID)
	current, err := client.SemanticModels(wsID).Get(ctx, modelID)
	if err != nil {
		return err
	}
	if err := assertKnowledgeBaseNameUnchanged(current.Name, params.Name); err != nil {
		return err
	}
	updateReq := &moi.SemanticModelUpsertRequest{
		Name:        current.Name,
		Description: params.Description,
		Tables:      params.Tables,
		Files:       params.Files,
	}
	_, err = client.SemanticModels(wsID).Update(ctx, modelID, updateReq)
	return err
}

func assertKnowledgeBaseNameUnchanged(currentName, requestedName string) error {
	if strings.TrimSpace(requestedName) == "" {
		return serviceError(ErrCodeBadRequest, i18n.KeyNameRequired, nil)
	}
	if strings.TrimSpace(currentName) != strings.TrimSpace(requestedName) {
		return knowledgeBaseNameImmutableError()
	}
	return nil
}

func (s *semanticModelService) UpdateModel(ctx context.Context, params UpdateSemanticModelRequest) error {
	wsID, err := callerWorkspaceID(ctx)
	if err != nil {
		return err
	}
	return coreclient.Execute(ctx, coreclient.FromContext(ctx), func(callCtx context.Context, client *moi.Client) error {
		return s.updateModel(callCtx, client, wsID, params)
	})
}

func (s *semanticModelService) updateModel(ctx context.Context, client *moi.Client, wsID string, params UpdateSemanticModelRequest) error {
	modelID := int64(params.ModelID)
	if params.Tables == nil && params.Files == nil {
		return s.updateModelWithoutSourceMutation(ctx, client, wsID, params)
	}
	requestFileIDs, err := semanticModelFileIDs(params.Files)
	if err != nil {
		return err
	}
	requestFileIDs = appendUniqueStrings(nil, requestFileIDs)
	if len(requestFileIDs) == 0 {
		return s.updateModelWithoutSourceMutation(ctx, client, wsID, params)
	}
	current, err := client.SemanticModels(wsID).Get(ctx, modelID)
	if err != nil {
		return err
	}
	if err := assertKnowledgeBaseNameUnchanged(current.Name, params.Name); err != nil {
		return err
	}
	// Force the immutable created name for any subsequent Core update.
	params.Name = current.Name
	sourcesUnchanged, err := semanticModelUpdateSourcesUnchanged(current, params)
	if err != nil {
		return err
	}
	if sourcesUnchanged {
		// tags/metadata-only: keep existing source identity without re-running source mutation.
		// Reuse the current model already fetched above so name immutability does not re-GET.
		updateReq := &moi.SemanticModelUpsertRequest{
			Name:        current.Name,
			Description: params.Description,
			Tables:      params.Tables,
			Files:       params.Files,
		}
		_, err = client.SemanticModels(wsID).Update(ctx, modelID, updateReq)
		return err
	}
	sources := make([]CreateSemanticModelSourceRequest, 0, len(requestFileIDs))
	for _, fileID := range requestFileIDs {
		sources = append(sources, CreateSemanticModelSourceRequest{
			SourceType: kbSourceTypeCatalogFile,
			FileID:     fileID,
		})
	}
	// Legacy files.file_ids cannot carry volume_id; reject before any side effects.
	if err := validateCreateSemanticModelSources(sources); err != nil {
		return err
	}
	if s.dataDomainService == nil {
		return fmt.Errorf("catalog data-domain service is not configured")
	}
	actor := semanticModelActor(ctx)
	filesBaseJSON, err := semanticModelCreateFilesBase(params.Files, modelID)
	if err != nil {
		return err
	}
	filesBaseJSON, err = appendSemanticModelFiles(filesBaseJSON, modelID, nil)
	if err != nil {
		return err
	}
	if _, err := parseKBVectorBinding(filesBaseJSON); err != nil {
		return err
	}
	domain, err := s.ensureAppendKnowledgeBaseDataDomain(ctx, wsID, modelID, params.Name, params.Description, sources, actor, filesBaseJSON)
	if err != nil {
		return err
	}

	db := ctxutil.TenantDBFrom(ctx)
	if db == nil {
		return fmt.Errorf("tenant db is required")
	}
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txCtx := ctxutil.WithTenantDB(ctx, tx)
		if err := s.lockKnowledgeBaseDataDomainForAppend(txCtx, modelID); err != nil {
			return err
		}
		if _, err := client.SemanticModels(wsID).Get(txCtx, modelID); err != nil {
			return err
		}
		sourceResult, err := s.createKnowledgeBaseSourceMetadataIntentsInTx(txCtx, client, wsID, modelID, domain, sources, actor, true)
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
		updateReq := &moi.SemanticModelUpsertRequest{
			Name:        params.Name,
			Description: params.Description,
			Tables:      params.Tables,
			Files:       filesJSON,
		}
		if _, err := client.SemanticModels(wsID).Update(txCtx, modelID, updateReq); err != nil {
			return rollbackCreatedSources(fmt.Errorf("update semantic model sources: %w", err))
		}
		return nil
	})
}

func (s *semanticModelService) DeleteModel(ctx context.Context, modelID int) error {
	wsID, err := callerWorkspaceID(ctx)
	if err != nil {
		return err
	}
	return coreclient.Execute(ctx, coreclient.FromContext(ctx), func(callCtx context.Context, client *moi.Client) error {
		return s.deleteModel(callCtx, client, wsID, modelID)
	})
}

func (s *semanticModelService) deleteModel(ctx context.Context, client *moi.Client, wsID string, modelID int) error {
	id := int64(modelID)
	if id <= 0 {
		return client.SemanticModels(wsID).Delete(ctx, id)
	}
	if err := s.deleteKnowledgeBaseBackendResources(ctx, wsID, id); err != nil {
		return err
	}
	if err := client.SemanticModels(wsID).Delete(ctx, id); err != nil {
		if !moi.IsCode(err, common.ErrorCode_NOT_FOUND) {
			return fmt.Errorf("delete semantic model %d: %w", id, err)
		}
	}
	return nil
}

// deleteKnowledgeBaseBackendResources owns the Backend-local half of semantic
// model deletion. Create-failure compensation invokes this alongside Core
// deletion without allowing a Backend failure to skip the Core action.
func (s *semanticModelService) deleteKnowledgeBaseBackendResources(ctx context.Context, wsID string, modelID int64) error {
	if s.workflowService == nil {
		return fmt.Errorf("workflow service is required to delete knowledge base workflow")
	}
	if ctxutil.TenantDBFrom(ctx) == nil {
		return fmt.Errorf("tenant db is required")
	}
	workflowIDs := []string{
		knowledgeBaseWorkflowID(wsID, modelID),
		knowledgeBaseMediaWorkflowID(wsID, modelID, kbAudioRAGTemplateKey),
		knowledgeBaseMediaWorkflowID(wsID, modelID, kbVideoRAGTemplateKey),
	}
	for _, workflowID := range workflowIDs {
		if err := s.workflowService.ValidateWorkflowDelete(ctx, workflowID); err != nil {
			if !moi.IsCode(err, common.ErrorCode_NOT_FOUND) {
				if moi.IsCode(err, common.ErrorCode_ALREADY_EXISTS) {
					return knowledgeBaseWorkflowDeleteConflictError(err)
				}
				return fmt.Errorf("validate knowledge base workflow %s delete: %w", workflowID, err)
			}
		}
	}
	for _, workflowID := range workflowIDs {
		if err := s.workflowService.DeleteWorkflow(ctx, workflowID); err != nil {
			if !moi.IsCode(err, common.ErrorCode_NOT_FOUND) {
				if moi.IsCode(err, common.ErrorCode_ALREADY_EXISTS) {
					return knowledgeBaseWorkflowDeleteConflictError(err)
				}
				return fmt.Errorf("delete knowledge base workflow %s: %w", workflowID, err)
			}
		}
	}
	if err := s.deleteKnowledgeBaseCatalogResources(ctx, modelID); err != nil {
		return fmt.Errorf("delete knowledge base catalog resources: %w", err)
	}
	if err := s.deleteKnowledgeBaseRows(ctx, modelID); err != nil {
		return fmt.Errorf("delete knowledge base records: %w", err)
	}
	return nil
}

// GetModel returns a semantic model by ID.
func (s *semanticModelService) GetModel(ctx context.Context, kbID int) (*SemanticModelInfo, error) {
	wsID, err := callerWorkspaceID(ctx)
	if err != nil {
		return nil, err
	}
	var response *SemanticModelInfo
	err = coreclient.Execute(ctx, coreclient.FromContext(ctx), func(callCtx context.Context, client *moi.Client) error {
		var callErr error
		response, callErr = s.getModel(callCtx, client, wsID, kbID)
		return callErr
	})
	return response, err
}

func (s *semanticModelService) getModel(ctx context.Context, client *moi.Client, wsID string, kbID int) (*SemanticModelInfo, error) {
	modelID := int64(kbID)
	if modelID == 0 {
		return nil, semanticModelKBNotFoundError()
	}

	model, err := client.SemanticModels(wsID).Get(ctx, modelID)
	if err != nil {
		return nil, err
	}
	info := toSemanticModelInfo(model)
	if err := s.enrichSemanticModelSourceCounts(ctx, []*SemanticModelInfo{info}); err != nil {
		return nil, err
	}
	return info, nil
}

// Fix #3: ListEntries — propagate resolve errors, return empty only when model doesn't exist yet
func (s *semanticModelService) ListEntries(ctx context.Context, params ListSemanticEntriesRequest) (*ListSemanticEntriesResponse, error) {
	wsID, err := callerWorkspaceID(ctx)
	if err != nil {
		return nil, err
	}
	var response *ListSemanticEntriesResponse
	err = coreclient.Execute(ctx, coreclient.FromContext(ctx), func(callCtx context.Context, client *moi.Client) error {
		var callErr error
		response, callErr = s.listEntries(callCtx, client, wsID, params)
		return callErr
	})
	return response, err
}

func (s *semanticModelService) listEntries(ctx context.Context, client *moi.Client, wsID string, params ListSemanticEntriesRequest) (*ListSemanticEntriesResponse, error) {
	modelID := int64(params.ModelID)
	if modelID == 0 {
		// No semantic model yet — return empty list (not an error per design)
		return &ListSemanticEntriesResponse{Items: []SemanticEntry{}, Total: 0}, nil
	}

	var opts []moi.ListOption
	if params.PageSize > 0 {
		opts = append(opts, moi.WithPageSize(int32(params.PageSize)))
	}
	if params.PageToken != "" {
		opts = append(opts, moi.WithPageToken(params.PageToken))
	}

	resp, err := client.SemanticModels(wsID).ListEntries(ctx, modelID, params.Kind, opts...)
	if err != nil {
		return nil, err
	}

	items := make([]SemanticEntry, 0, len(resp.Items))
	for _, e := range resp.Items {
		items = append(items, toSemanticEntry(e))
	}
	return &ListSemanticEntriesResponse{
		Items:         items,
		Total:         int(resp.Total),
		NextPageToken: resp.NextPageToken,
	}, nil
}

func (s *semanticModelService) CreateEntry(ctx context.Context, params CreateSemanticEntryRequest) (*SemanticEntry, error) {
	wsID, err := callerWorkspaceID(ctx)
	if err != nil {
		return nil, err
	}
	var response *SemanticEntry
	err = coreclient.Execute(ctx, coreclient.FromContext(ctx), func(callCtx context.Context, client *moi.Client) error {
		var callErr error
		response, callErr = s.createEntry(callCtx, client, wsID, params)
		return callErr
	})
	return response, err
}

func (s *semanticModelService) createEntry(ctx context.Context, client *moi.Client, wsID string, params CreateSemanticEntryRequest) (*SemanticEntry, error) {
	modelID := int64(params.ModelID)

	entry, err := client.SemanticModels(wsID).CreateEntry(ctx, modelID, &moi.SemanticEntryUpsertRequest{
		Kind:   params.Kind,
		Key:    params.Key,
		Tables: params.Tables,
		Spec:   params.Spec,
	})
	if err != nil {
		return nil, err
	}
	result := toSemanticEntry(entry)
	return &result, nil
}

func (s *semanticModelService) UpdateEntry(ctx context.Context, params UpdateSemanticEntryRequest) error {
	wsID, err := callerWorkspaceID(ctx)
	if err != nil {
		return err
	}
	return coreclient.Execute(ctx, coreclient.FromContext(ctx), func(callCtx context.Context, client *moi.Client) error {
		return s.updateEntry(callCtx, client, wsID, params)
	})
}

func (s *semanticModelService) updateEntry(ctx context.Context, client *moi.Client, wsID string, params UpdateSemanticEntryRequest) error {
	modelID := int64(params.ModelID)
	if modelID == 0 {
		return semanticModelNotFoundError()
	}

	// Enforce "kind is immutable" constraint: find the existing entry and compare kind.
	existingKind, found, err := s.findEntryKind(ctx, client, wsID, modelID, int64(params.EntryID))
	if err != nil {
		return fmt.Errorf("fetch entry for kind check: %w", err)
	}
	if !found {
		return semanticEntryNotFoundError(params.EntryID)
	}
	if existingKind != params.Kind {
		return semanticKindCannotBeChangedError(existingKind, params.Kind)
	}

	_, err = client.SemanticModels(wsID).UpdateEntry(ctx, modelID, int64(params.EntryID), &moi.SemanticEntryUpsertRequest{
		Kind:   params.Kind,
		Key:    params.Key,
		Tables: params.Tables,
		Spec:   params.Spec,
	})
	return err
}

func (s *semanticModelService) DeleteEntry(ctx context.Context, params DeleteSemanticEntryRequest) error {
	wsID, err := callerWorkspaceID(ctx)
	if err != nil {
		return err
	}
	return coreclient.Execute(ctx, coreclient.FromContext(ctx), func(callCtx context.Context, client *moi.Client) error {
		return s.deleteEntry(callCtx, client, wsID, params)
	})
}

func (s *semanticModelService) deleteEntry(ctx context.Context, client *moi.Client, wsID string, params DeleteSemanticEntryRequest) error {
	modelID := int64(params.ModelID)
	if modelID == 0 {
		return semanticModelNotFoundError()
	}

	return client.SemanticModels(wsID).DeleteEntry(ctx, modelID, int64(params.EntryID))
}

// Import creates entries only on an empty semantic model.
// It refuses models with existing entries and returns on the first failed create without rollback.
func (s *semanticModelService) Import(ctx context.Context, params ImportSemanticModelRequest) (*ImportSemanticModelResponse, error) {
	wsID, err := callerWorkspaceID(ctx)
	if err != nil {
		return nil, err
	}
	var response *ImportSemanticModelResponse
	err = coreclient.Execute(ctx, coreclient.FromContext(ctx), func(callCtx context.Context, client *moi.Client) error {
		var callErr error
		response, callErr = s.importModel(callCtx, client, wsID, params)
		return callErr
	})
	return response, err
}

func (s *semanticModelService) importModel(ctx context.Context, client *moi.Client, wsID string, params ImportSemanticModelRequest) (*ImportSemanticModelResponse, error) {
	modelID := int64(params.ModelID)

	// Check if model already has entries — refuse import if so.
	existing, err := client.SemanticModels(wsID).ListEntries(ctx, modelID, "", moi.WithPageSize(1))
	if err != nil {
		return nil, fmt.Errorf("check existing entries: %w", err)
	}
	if existing.Total > 0 {
		return nil, semanticModelEntriesImportBlockedError()
	}

	// Create entries on the existing model.
	imported := 0
	for _, e := range params.Entries {
		if IsDisabledLegacySemanticEntryTables(e.Tables) {
			continue
		}
		_, err := client.SemanticModels(wsID).CreateEntry(ctx, modelID, &moi.SemanticEntryUpsertRequest{
			Kind:   e.Kind,
			Key:    e.Key,
			Tables: e.Tables,
			Spec:   e.Spec,
		})
		if err != nil {
			return nil, fmt.Errorf("import entry %q: %w", e.Key, err)
		}
		imported++
	}

	return &ImportSemanticModelResponse{
		Imported: imported,
		ModelID:  modelID,
	}, nil
}

func (s *semanticModelService) Export(ctx context.Context, kbID int) (*ExportSemanticModelResponse, error) {
	wsID, err := callerWorkspaceID(ctx)
	if err != nil {
		return nil, err
	}
	var response *ExportSemanticModelResponse
	err = coreclient.Execute(ctx, coreclient.FromContext(ctx), func(callCtx context.Context, client *moi.Client) error {
		var callErr error
		response, callErr = s.exportModel(callCtx, client, wsID, kbID)
		return callErr
	})
	return response, err
}

func (s *semanticModelService) exportModel(ctx context.Context, client *moi.Client, wsID string, kbID int) (*ExportSemanticModelResponse, error) {
	modelID := int64(kbID)
	if modelID == 0 {
		return nil, semanticModelNotFoundError()
	}

	resp, err := client.SemanticModels(wsID).Export(ctx, modelID)
	if err != nil {
		return nil, err
	}

	entries := make([]SemanticEntry, 0, len(resp.Entries))
	for _, e := range resp.Entries {
		entries = append(entries, toSemanticEntry(e))
	}

	return &ExportSemanticModelResponse{
		Model:   *toSemanticModelInfo(resp.Model),
		Entries: entries,
	}, nil
}

// Fix #4: Validate — return valid=false + errors[] instead of propagating 400
func (s *semanticModelService) Validate(ctx context.Context, kbID int) (*ValidateSemanticModelResponse, error) {
	wsID, err := callerWorkspaceID(ctx)
	if err != nil {
		return nil, err
	}
	var response *ValidateSemanticModelResponse
	err = coreclient.Execute(ctx, coreclient.FromContext(ctx), func(callCtx context.Context, client *moi.Client) error {
		var callErr error
		response, callErr = s.validateModel(callCtx, client, wsID, kbID)
		return callErr
	})
	return response, err
}

func (s *semanticModelService) validateModel(ctx context.Context, client *moi.Client, wsID string, kbID int) (*ValidateSemanticModelResponse, error) {
	modelID := int64(kbID)
	if modelID == 0 {
		return nil, semanticModelNotFoundError()
	}

	resp, err := client.SemanticModels(wsID).Validate(ctx, modelID)
	if err != nil {
		// moi-core returns 400 for validation failures — convert to valid=false + errors
		if moi.IsCode(err, common.ErrorCode_INVALID_ARGUMENT) {
			var sdkErr *moi.Error
			if ok := sdkErrorAs(err, &sdkErr); ok {
				message := sdkErr.Message
				if localized, ok := i18n.LocalizeCoreError(ctx, sdkErr); ok {
					message = localized
				}
				return &ValidateSemanticModelResponse{
					Valid:  false,
					Errors: []string{message},
				}, nil
			}
			return &ValidateSemanticModelResponse{
				Valid:  false,
				Errors: []string{err.Error()},
			}, nil
		}
		return nil, err
	}

	return &ValidateSemanticModelResponse{Valid: resp.Valid}, nil
}

// ========== Helpers ==========

// findEntryKind paginates through all entries to find the kind of a specific entry.
func (s *semanticModelService) findEntryKind(ctx context.Context, c *moi.Client, wsID string, modelID, entryID int64) (string, bool, error) {
	pageToken := ""
	for {
		opts := []moi.ListOption{moi.WithPageSize(100)}
		if pageToken != "" {
			opts = append(opts, moi.WithPageToken(pageToken))
		}
		resp, err := c.SemanticModels(wsID).ListEntries(ctx, modelID, "", opts...)
		if err != nil {
			return "", false, err
		}
		for _, e := range resp.Items {
			if e.ID == entryID {
				return e.Kind, true, nil
			}
		}
		if resp.NextPageToken == "" || len(resp.Items) == 0 {
			break
		}
		pageToken = resp.NextPageToken
	}
	return "", false, nil
}

func toSemanticModelInfo(m *moi.SemanticModel) *SemanticModelInfo {
	if m == nil {
		return nil
	}
	tablesJSON, err := semanticModelTablesToRawJSON(m.Tables)
	if err != nil {
		tablesJSON = json.RawMessage("[]")
	}
	return &SemanticModelInfo{
		ID:          m.ID,
		Name:        m.Name,
		Description: m.Description,
		Tables:      tablesJSON,
		Files:       m.Files,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
}

func toSemanticEntry(e *moi.SemanticEntry) SemanticEntry {
	if e == nil {
		return SemanticEntry{}
	}
	return SemanticEntry{
		ID:        e.ID,
		Kind:      e.Kind,
		Key:       e.Key,
		Tables:    e.Tables,
		Spec:      json.RawMessage(e.Spec),
		CreatedAt: e.CreatedAt,
		UpdatedAt: e.UpdatedAt,
	}
}

func sdkErrorAs(err error, target **moi.Error) bool {
	if err == nil {
		return false
	}
	return errors.As(err, target)
}
