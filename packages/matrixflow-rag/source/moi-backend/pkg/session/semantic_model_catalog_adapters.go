package session

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"strconv"
	"strings"
	"syscall"

	moi "github.com/matrixflow/moi-core/go-sdk"
	"github.com/matrixflow/moi-core/model/common"
	"github.com/matrixorigin/matrixflow/moi-backend/pkg/catalog"
	"github.com/matrixorigin/matrixflow/moi-backend/pkg/coreclient"
	"github.com/matrixorigin/matrixflow/moi-backend/pkg/ctxutil"
	"github.com/matrixorigin/matrixflow/moi-backend/pkg/dataconn"
	"github.com/matrixorigin/matrixflow/moi-backend/pkg/model"
	"github.com/matrixorigin/matrixflow/moi-backend/pkg/workflowtemplate"
	"github.com/matrixorigin/matrixflow/moi-backend/pkg/workflowv2"
)

type semanticModelCatalogDataDomainAdapter struct {
	service catalog.DataCenterService
}

func NewSemanticModelCatalogDataDomainAdapter(service catalog.DataCenterService) SemanticModelCatalogDataDomainService {
	return &semanticModelCatalogDataDomainAdapter{service: service}
}

// knowledgeBaseIAMSubrequestContext gives each resource lifecycle call a stable
// child request ID while preserving the parent trace and verified IAM role.
func knowledgeBaseIAMSubrequestContext(ctx context.Context, operation string) context.Context {
	trusted, hasTrusted := ctxutil.CoreIAMRequestFrom(ctx)
	requestID := trusted.RequestID
	if requestID == "" {
		requestID = moi.HeadersFromContext(ctx)["X-Request-ID"]
	}
	if requestID == "" {
		return ctx
	}
	derived := requestID + "." + operation
	if len(derived) > 128 {
		digest := sha256.Sum256([]byte(requestID + "\x00" + operation))
		derived = "kb." + hex.EncodeToString(digest[:])
	}
	ctx = moi.ContextWithHeader(ctx, "X-Request-ID", derived)
	if hasTrusted {
		trusted.RequestID = derived
		ctx = ctxutil.WithCoreIAMRequest(ctx, trusted)
	}
	return ctx
}

func (a *semanticModelCatalogDataDomainAdapter) ResolveDefaultCatalogID(ctx context.Context) (int64, error) {
	if a == nil || a.service == nil {
		return 0, fmt.Errorf("catalog data-domain service is not configured")
	}
	// The default Catalog is a system-initialized, backend-owned resource. Resolve
	// it from the catalog reservation metadata instead of the user-scoped catalog
	// list, which would incorrectly require catalog.read/database.read.
	id, err := a.service.ResolveDefaultCatalogID(ctx)
	if err != nil {
		return 0, err
	}
	if id <= 0 {
		return 0, fmt.Errorf("default catalog is not available")
	}
	return id, nil
}

func (a *semanticModelCatalogDataDomainAdapter) ResolveDatabaseByName(ctx context.Context, catalogID int64, name string) (int64, string, bool, error) {
	if a == nil || a.service == nil {
		return 0, "", false, fmt.Errorf("catalog data-domain service is not configured")
	}
	return a.service.ResolveDatabaseByName(ctx, catalogID, name)
}

func (a *semanticModelCatalogDataDomainAdapter) ResolveCatalogIDByDatabaseID(ctx context.Context, databaseID int64) (int64, error) {
	if a == nil || a.service == nil {
		return 0, fmt.Errorf("catalog data-domain service is not configured")
	}
	return a.service.ResolveCatalogIDByDatabaseID(ctx, databaseID)
}

func (a *semanticModelCatalogDataDomainAdapter) CreateDatabase(ctx context.Context, catalogID int64, name, description, displayName string) (int64, error) {
	if a == nil || a.service == nil {
		return 0, fmt.Errorf("catalog data-domain service is not configured")
	}
	ctx = knowledgeBaseIAMSubrequestContext(ctx, "database-create-"+strconv.FormatInt(catalogID, 10)+"-"+name)
	resp, err := a.service.CreateDatabase(ctx, &catalog.CreateDatabaseRequest{
		CatalogID:   int(catalogID),
		Name:        name,
		DisplayName: displayName,
		Description: description,
	})
	if err != nil {
		return 0, err
	}
	if resp == nil || resp.ID == nil || *resp.ID <= 0 {
		return 0, fmt.Errorf("catalog database create returned empty id")
	}
	return int64(*resp.ID), nil
}

func (a *semanticModelCatalogDataDomainAdapter) CreateVolume(ctx context.Context, databaseID int64, name, description string) (int64, error) {
	if a == nil || a.service == nil {
		return 0, fmt.Errorf("catalog data-domain service is not configured")
	}
	ctx = knowledgeBaseIAMSubrequestContext(ctx, "volume-create-"+strconv.FormatInt(databaseID, 10)+"-"+name)
	resp, err := a.service.CreateVolume(ctx, &catalog.CreateVolumeRequest{
		DatabaseID:  int(databaseID),
		Name:        name,
		Description: description,
	})
	if err != nil {
		return 0, err
	}
	if resp == nil || resp.ID == nil || *resp.ID <= 0 {
		return 0, fmt.Errorf("catalog volume create returned empty id")
	}
	return int64(*resp.ID), nil
}

func (a *semanticModelCatalogDataDomainAdapter) ResolveVolumeIDByName(ctx context.Context, databaseID int64, name string) (int64, bool, error) {
	if a == nil || a.service == nil {
		return 0, false, fmt.Errorf("catalog data-domain service is not configured")
	}
	return a.service.ResolveVolumeIDByName(ctx, databaseID, name)
}

func (a *semanticModelCatalogDataDomainAdapter) ListDatabaseTableLeaves(ctx context.Context, params KnowledgeBaseTableLeafListParams) (*KnowledgeBaseTableLeafListResult, error) {
	if a == nil || a.service == nil {
		return nil, fmt.Errorf("catalog data-domain service is not configured")
	}
	resp, err := a.service.ListDatabaseTableLeaves(ctx, &catalog.ListDatabaseTableLeavesRequest{
		DatabaseID: int(params.DatabaseID),
		PageSize:   params.PageSize,
		PageToken:  params.PageToken,
		Search:     params.Search,
	})
	if err != nil {
		return nil, err
	}
	out := &KnowledgeBaseTableLeafListResult{}
	if resp == nil {
		return out, nil
	}
	out.Total = resp.Total
	out.NextPageToken = resp.NextPageToken
	out.Items = make([]KnowledgeBaseTableLeaf, 0, len(resp.Items))
	for _, item := range resp.Items {
		if item == nil {
			continue
		}
		out.Items = append(out.Items, KnowledgeBaseTableLeaf{
			TableID:    int64(item.TableID),
			TableName:  item.TableName,
			DatabaseID: int64(item.DatabaseID),
		})
	}
	return out, nil
}

func (a *semanticModelCatalogDataDomainAdapter) CloneTableForKnowledgeBase(ctx context.Context, sourceTableID, targetDatabaseID int64, idempotencyKey string) (*KnowledgeBaseTableCloneResult, error) {
	if a == nil || a.service == nil {
		return nil, fmt.Errorf("catalog data-domain service is not configured")
	}
	resp, err := a.service.CloneTableForKnowledgeBase(ctx, &catalog.CloneTableForKnowledgeBaseRequest{
		SourceTableID:    sourceTableID,
		TargetDatabaseID: targetDatabaseID,
		IdempotencyKey:   idempotencyKey,
	})
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, fmt.Errorf("catalog table clone returned empty response")
	}
	return &KnowledgeBaseTableCloneResult{
		OperationID: resp.OperationID,
		Status:      resp.Status,
		SourceDB:    resp.SourceDB,
		SourceTable: resp.SourceTable,
		TargetDB:    resp.TargetDB,
		TargetTable: resp.TargetTable,
		TargetID:    resp.TargetID,
		Error:       resp.Error,
	}, nil
}

func (a *semanticModelCatalogDataDomainAdapter) DeleteVolume(ctx context.Context, volumeID int64) error {
	if a == nil || a.service == nil {
		return fmt.Errorf("catalog data-domain service is not configured")
	}
	ctx = knowledgeBaseIAMSubrequestContext(ctx, "volume-delete-"+strconv.FormatInt(volumeID, 10))
	return a.service.DeleteVolume(ctx, &catalog.DeleteByIDRequest{ID: int(volumeID)})
}

func (a *semanticModelCatalogDataDomainAdapter) DeleteDatabase(ctx context.Context, databaseID int64) error {
	if a == nil || a.service == nil {
		return fmt.Errorf("catalog data-domain service is not configured")
	}
	ctx = knowledgeBaseIAMSubrequestContext(ctx, "database-delete-"+strconv.FormatInt(databaseID, 10))
	return a.service.DeleteDatabase(ctx, &catalog.DeleteByIDRequest{ID: int(databaseID)})
}

type semanticModelCatalogFileAdapter struct {
	service catalog.FileService
}

// NewSemanticModelCatalogFileAdapter bridges Catalog file APIs for semantic-model
// knowledge bases. System-only model file download uses coreclient.System() at
// call time rather than storing a long-lived *moi.Client. Browser conversion
// reuses catalog.FileService.PreviewDownloadedFile so KB and Catalog share the
// same Office→PDF / ZIP→markdown logic.
func NewSemanticModelCatalogFileAdapter(service catalog.FileService) SemanticModelCatalogFileService {
	return &semanticModelCatalogFileAdapter{service: service}
}

func (a *semanticModelCatalogFileAdapter) ListFiles(ctx context.Context, params KnowledgeBaseCatalogFileListParams) (*KnowledgeBaseCatalogFileListResult, error) {
	if a == nil || a.service == nil {
		return nil, fmt.Errorf("catalog file service is not configured")
	}
	filters := []*catalog.FileFilter{
		{Name: "volume_id", Values: []string{strconv.FormatInt(params.VolumeID, 10)}},
	}
	if strings.TrimSpace(params.FileName) != "" {
		fuzzy := true
		filters = append(filters, &catalog.FileFilter{Name: "file_name", Values: []string{params.FileName}, Fuzzy: &fuzzy})
	}
	if len(params.FileExt) > 0 {
		filters = append(filters, &catalog.FileFilter{Name: "file_ext", Values: params.FileExt})
	}
	if len(params.FileIDs) > 0 {
		filters = append(filters, &catalog.FileFilter{Name: "file_id", Values: params.FileIDs})
	}
	resp, err := a.service.ListFiles(ctx, &catalog.ListFilesRequest{
		Page:     params.Page,
		PageSize: params.PageSize,
		Filters:  filters,
	})
	if err != nil {
		return nil, err
	}
	out := &KnowledgeBaseCatalogFileListResult{}
	if resp == nil {
		return out, nil
	}
	out.Total = resp.Total
	out.Items = make([]KnowledgeBaseCatalogFileLeaf, 0, len(resp.List))
	for _, item := range resp.List {
		if item == nil {
			continue
		}
		volumeID, _ := strconv.ParseInt(item.VolumeID, 10, 64)
		out.Items = append(out.Items, KnowledgeBaseCatalogFileLeaf{
			FileID:   item.ID,
			FileName: item.Name,
			VolumeID: volumeID,
		})
	}
	return out, nil
}

func (a *semanticModelCatalogFileAdapter) PreviewFile(ctx context.Context, fileID string) (*SemanticModelFilePreview, error) {
	if a == nil || a.service == nil {
		return nil, fmt.Errorf("semantic model artifact client is not configured")
	}
	workspaceID := strings.TrimSpace(ctxutil.WorkspaceIDFrom(ctx))
	if workspaceID == "" {
		return nil, fmt.Errorf("workspace_id is required for semantic model artifact preview")
	}
	var preview *SemanticModelFilePreview
	// Download stays on the system artifact route: callers with only
	// semantic_model.read may lack volume.read on the underlying Catalog file.
	// Conversion is the same FileService path Catalog uses after its own
	// user-scoped download.
	err := coreclient.Execute(ctx, coreclient.System(), func(callCtx context.Context, client *moi.Client) error {
		resp, err := client.Files().DownloadSemanticModelArtifactWithMeta(callCtx, workspaceID, fileID)
		if err != nil {
			return err
		}
		if resp == nil || resp.Body == nil {
			return fmt.Errorf("catalog file preview returned empty response")
		}
		prepared, err := a.service.PreviewDownloadedFile(callCtx, resp.Filename, resp.ContentType, resp.Body)
		if err != nil {
			return err
		}
		if prepared == nil || prepared.Body == nil {
			return fmt.Errorf("catalog file preview returned empty response")
		}
		preview = &SemanticModelFilePreview{
			Filename:    prepared.Filename,
			ContentType: prepared.ContentType,
			Body:        prepared.Body,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return preview, nil
}

func (a *semanticModelCatalogFileAdapter) DeleteFileFromVolume(ctx context.Context, volumeID int64, fileID string) error {
	if a == nil || a.service == nil {
		return fmt.Errorf("catalog file service is not configured")
	}
	return a.service.DeleteFile(ctx, &catalog.DeleteFileRequest{
		ID:       fileID,
		VolumeID: strconv.FormatInt(volumeID, 10),
	})
}

type semanticModelLocalFileImportAdapter struct {
	service dataconn.ConnectorService
}

func NewSemanticModelLocalFileImportAdapter(service dataconn.ConnectorService) SemanticModelLocalFileImportService {
	return &semanticModelLocalFileImportAdapter{service: service}
}

func (a *semanticModelLocalFileImportAdapter) UploadToVolume(ctx context.Context, params KnowledgeBaseLocalFileImportParams) (*KnowledgeBaseLocalFileImportResult, error) {
	if a == nil || a.service == nil {
		return nil, fmt.Errorf("data connection upload service is not configured")
	}
	var files []*multipart.FileHeader
	var cleanup func()
	if params.Reader != nil {
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		part, err := writer.CreateFormFile("file", params.FileName)
		if err != nil {
			return nil, fmt.Errorf("create multipart file: %w", err)
		}
		if _, err := io.Copy(part, params.Reader); err != nil {
			return nil, fmt.Errorf("write multipart file: %w", err)
		}
		if err := writer.Close(); err != nil {
			return nil, fmt.Errorf("close multipart writer: %w", err)
		}
		multipartReader := multipart.NewReader(&body, writer.Boundary())
		form, err := multipartReader.ReadForm(int64(body.Len()))
		if err != nil {
			return nil, fmt.Errorf("read multipart form: %w", err)
		}
		cleanup = func() { _ = form.RemoveAll() }
		defer cleanup()
		files = form.File["file"]
	}
	if params.UploadKind == kbLocalUploadKindStructured {
		connFileIDs, tableConfig, err := a.prepareStructuredLocalFileImport(ctx, params.TableConfig, files)
		if err != nil {
			return nil, err
		}
		resp, err := a.service.Upload(ctx, dataconn.UploadParams{
			TableConfigStr: tableConfig,
		})
		if err != nil {
			return nil, err
		}
		if resp == nil {
			return nil, fmt.Errorf("local file import returned empty response")
		}
		return &KnowledgeBaseLocalFileImportResult{TaskID: resp.TaskID, FileIDs: connFileIDs}, nil
	}
	if params.Reader == nil {
		return nil, fmt.Errorf("local file reader is required")
	}
	resp, err := a.service.Upload(ctx, dataconn.UploadParams{
		Files:          files,
		VolumeID:       strconv.FormatInt(params.VolumeID, 10),
		TableConfigStr: params.TableConfig,
	})
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, fmt.Errorf("local file import returned empty response")
	}
	return &KnowledgeBaseLocalFileImportResult{TaskID: resp.TaskID, FileIDs: resp.FileIDs}, nil
}

func (a *semanticModelLocalFileImportAdapter) prepareStructuredLocalFileImport(ctx context.Context, rawTableConfig string, files []*multipart.FileHeader) ([]string, string, error) {
	connFileIDs, err := structuredTableConfigConnFileIDs(rawTableConfig)
	if err != nil {
		return nil, "", err
	}
	if len(connFileIDs) == 0 {
		uploadResp, err := a.service.LocalUpload(ctx, files)
		if err != nil {
			return nil, "", fmt.Errorf("stage structured local file: %w", err)
		}
		if uploadResp != nil {
			connFileIDs = uploadResp.ConnFileIDs
		}
	}
	if len(connFileIDs) == 0 {
		return nil, "", fmt.Errorf("structured local file staging returned no conn_file_ids")
	}
	for _, id := range connFileIDs {
		if id == "" {
			return nil, "", fmt.Errorf("structured local file staging returned empty conn_file_id")
		}
	}
	tableConfig, err := structuredTableConfigWithConnFileIDs(rawTableConfig, connFileIDs)
	if err != nil {
		return nil, "", err
	}
	return connFileIDs, tableConfig, nil
}

func structuredTableConfigConnFileIDs(raw string) ([]string, error) {
	if raw == "" {
		return nil, fmt.Errorf("table_config is required for structured local upload")
	}
	var multi model.MultiTableConfig
	if err := json.Unmarshal([]byte(raw), &multi); err != nil {
		return nil, fmt.Errorf("parse table_config: %w", err)
	}
	if multi.MultiSheet || len(multi.Tables) > 0 {
		if len(multi.Tables) == 0 {
			return nil, fmt.Errorf("parse table_config: multi-sheet tables is empty")
		}
		var ids []string
		sawEmpty := false
		for i, table := range multi.Tables {
			if table == nil {
				return nil, fmt.Errorf("parse table_config: table %d is nil", i)
			}
			if len(table.ConnFileIds) == 0 {
				sawEmpty = true
				if len(ids) > 0 {
					return nil, fmt.Errorf("table_config.tables[%d].conn_file_ids is empty but another table has conn_file_ids", i)
				}
				continue
			}
			if sawEmpty {
				return nil, fmt.Errorf("table_config.tables[%d].conn_file_ids is set but another table has empty conn_file_ids", i)
			}
			if len(ids) == 0 {
				ids = append([]string(nil), table.ConnFileIds...)
				continue
			}
			if !sameStringSlice(ids, table.ConnFileIds) {
				return nil, fmt.Errorf("table_config.tables[%d].conn_file_ids must match other tables for one structured local file", i)
			}
		}
		return ids, nil
	}
	var single model.TableConfig
	if err := json.Unmarshal([]byte(raw), &single); err != nil {
		return nil, fmt.Errorf("parse table_config: %w", err)
	}
	return append([]string(nil), single.ConnFileIds...), nil
}

func structuredTableConfigWithConnFileIDs(raw string, connFileIDs []string) (string, error) {
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
			table.ConnFileIds = append([]string(nil), connFileIDs...)
		}
		normalized, err := json.Marshal(&multi)
		if err != nil {
			return "", fmt.Errorf("marshal table_config: %w", err)
		}
		return string(normalized), nil
	}
	var single model.TableConfig
	if err := json.Unmarshal([]byte(raw), &single); err != nil {
		return "", fmt.Errorf("parse table_config: %w", err)
	}
	single.ConnFileIds = append([]string(nil), connFileIDs...)
	normalized, err := json.Marshal(&single)
	if err != nil {
		return "", fmt.Errorf("marshal table_config: %w", err)
	}
	return string(normalized), nil
}

func sameStringSlice(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

type semanticModelWorkflowTemplateAdapter struct {
	service workflowtemplate.Service
}

func NewSemanticModelWorkflowTemplateAdapter(service workflowtemplate.Service) SemanticModelWorkflowTemplateService {
	return &semanticModelWorkflowTemplateAdapter{service: service}
}

func (a *semanticModelWorkflowTemplateAdapter) GetByTemplateKey(ctx context.Context, templateKey string) (*model.WorkflowTemplate, error) {
	if a == nil || a.service == nil {
		return nil, fmt.Errorf("workflow template service is not configured")
	}
	return a.service.GetByTemplateKey(ctx, templateKey)
}

type semanticModelWorkflowAdapter struct {
	service workflowv2.Service
}

func NewSemanticModelWorkflowAdapter(service workflowv2.Service) SemanticModelWorkflowService {
	return &semanticModelWorkflowAdapter{service: service}
}

func (a *semanticModelWorkflowAdapter) RequireKnowledgeBaseWorkflow(ctx context.Context, workflowID string) error {
	if a == nil || a.service == nil {
		return fmt.Errorf("workflow service is not configured")
	}
	workflowID = strings.TrimSpace(workflowID)
	if workflowID == "" {
		return fmt.Errorf("workflow id is required")
	}
	ctx = knowledgeBaseIAMSubrequestContext(ctx, "workflow-get-"+workflowID)
	envelope, err := a.service.GetWorkflow(ctx, workflowID)
	if err != nil {
		return err
	}
	if envelope == nil {
		return fmt.Errorf("workflow %q returned empty result", workflowID)
	}
	if strings.TrimSpace(envelope.Workflow.ID) == "" {
		return fmt.Errorf("workflow %q returned empty id", workflowID)
	}
	if envelope.Workflow.ID != workflowID {
		return fmt.Errorf("workflow id mismatch: requested %q, got %q", workflowID, envelope.Workflow.ID)
	}
	return nil
}

func (a *semanticModelWorkflowAdapter) DeployKnowledgeBaseWorkflow(ctx context.Context, params KnowledgeBaseWorkflowDeployRequest) error {
	if a == nil || a.service == nil {
		return fmt.Errorf("workflow service is not configured")
	}
	ctx = knowledgeBaseIAMSubrequestContext(ctx, "workflow-deploy-"+params.WorkflowID)
	inputForm, err := parseKnowledgeBaseWorkflowInputForm(params.InputFormJSON)
	if err != nil {
		return err
	}
	enabled := params.TriggerEnabled
	autoDispatchEnabled := params.AutoDispatchEnabled
	executionMode := strings.TrimSpace(params.ExecutionMode)
	if executionMode == "" {
		executionMode = workflowv2.ExecutionModeVolumeTrigger
	}
	var volumeTrigger *workflowv2.WorkflowVolumeTriggerRequest
	if executionMode == workflowv2.ExecutionModeVolumeTrigger {
		volumeTrigger = &workflowv2.WorkflowVolumeTriggerRequest{
			VolumeID:            params.RawVolumeID,
			Enabled:             &enabled,
			AutoDispatchEnabled: &autoDispatchEnabled,
		}
	}
	_, err = a.service.DeployWorkflow(ctx, &workflowv2.DeployWorkflowRequest{
		WorkflowID:    params.WorkflowID,
		Name:          params.Name,
		Description:   params.Description,
		DSLYAML:       params.DSLYAML,
		SourceType:    workflowv2.SourceTypeManualDSL,
		ExecutionMode: executionMode,
		InputForm:     inputForm,
		DefaultValues: params.DefaultValues,
		VolumeTrigger: volumeTrigger,
	})
	return err
}

func (a *semanticModelWorkflowAdapter) RunKnowledgeBaseWorkflow(ctx context.Context, workflowID string, values map[string]any) (*KnowledgeBaseWorkflowRunResult, error) {
	if a == nil || a.service == nil {
		return nil, fmt.Errorf("workflow service is not configured")
	}
	result, err := a.service.RunWorkflow(ctx, workflowID, &workflowv2.RunWorkflowRequest{Values: values})
	if err != nil {
		return nil, err
	}
	if result == nil || strings.TrimSpace(result.WorkflowRun.ExecutionID) == "" {
		return nil, fmt.Errorf("workflow execution id is required")
	}
	return &KnowledgeBaseWorkflowRunResult{ExecutionID: strings.TrimSpace(result.WorkflowRun.ExecutionID)}, nil
}

func (a *semanticModelWorkflowAdapter) ValidateWorkflowDelete(ctx context.Context, workflowID string) error {
	if a == nil || a.service == nil {
		return fmt.Errorf("workflow service is not configured")
	}
	workflowID = strings.TrimSpace(workflowID)
	if workflowID == "" {
		return fmt.Errorf("workflow id is required")
	}
	return a.service.ValidateWorkflowDelete(ctx, workflowID)
}

// knowledgeBaseWorkflowDeleteTransportRetries is the number of extra DELETE
// attempts after the first call when the transport fails with EOF / reset.
// Catalog workflow IAM delete is idempotent and may already have completed
// server-side (#14614).
const knowledgeBaseWorkflowDeleteTransportRetries = 2

func (a *semanticModelWorkflowAdapter) DeleteWorkflow(ctx context.Context, workflowID string) error {
	if a == nil || a.service == nil {
		return fmt.Errorf("workflow service is not configured")
	}
	workflowID = strings.TrimSpace(workflowID)
	if workflowID == "" {
		return fmt.Errorf("workflow id is required")
	}
	// Stable child request ID for the whole delete+retry+converge sequence so
	// Catalog can replay the same lifecycle operation.
	ctx = knowledgeBaseIAMSubrequestContext(ctx, "workflow-delete-"+workflowID)
	var lastErr error
	for attempt := 0; attempt <= knowledgeBaseWorkflowDeleteTransportRetries; attempt++ {
		_, err := a.service.DeleteWorkflow(ctx, workflowID)
		if err == nil {
			return nil
		}
		if moi.IsCode(err, common.ErrorCode_NOT_FOUND) {
			// Already deleted (or never existed): treat as success for KB teardown.
			return nil
		}
		lastErr = err
		if !isIdempotentDeleteTransportError(err) {
			return err
		}
	}
	// Transport still flaky after retries: confirm deletion via get. If the
	// workflow is gone, Catalog already finished the DELETE successfully.
	if _, convergeErr := a.service.GetWorkflow(ctx, workflowID); convergeErr != nil {
		if moi.IsCode(convergeErr, common.ErrorCode_NOT_FOUND) {
			return nil
		}
		return fmt.Errorf("delete knowledge base workflow %s: %w (converge: %v)", workflowID, lastErr, convergeErr)
	}
	return fmt.Errorf("delete knowledge base workflow %s: %w", workflowID, lastErr)
}

// isIdempotentDeleteTransportError reports transport failures where Catalog may
// have completed an idempotent DELETE while the client only saw a broken
// connection (EOF / reset / unexpected EOF).
func isIdempotentDeleteTransportError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, net.ErrClosed) {
		return true
	}
	if errors.Is(err, syscall.ECONNRESET) || errors.Is(err, syscall.EPIPE) {
		return true
	}
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		if opErr.Err != nil && isIdempotentDeleteTransportError(opErr.Err) {
			return true
		}
		msg := strings.ToLower(opErr.Error())
		if strings.Contains(msg, "connection reset") || strings.Contains(msg, "broken pipe") {
			return true
		}
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "eof") ||
		strings.Contains(msg, "connection reset") ||
		strings.Contains(msg, "broken pipe")
}

func (a *semanticModelWorkflowAdapter) ListFileExecutions(ctx context.Context, fileID string, semanticModelID int64) (*moi.FileExecutionsResponse, error) {
	if a == nil || a.service == nil {
		return nil, fmt.Errorf("workflow service is not configured")
	}
	return a.service.ListSemanticModelFileExecutions(ctx, fileID, semanticModelID)
}

func parseKnowledgeBaseWorkflowInputForm(raw string) (*moi.WorkflowAgentInputForm, error) {
	if raw == "" {
		return nil, nil
	}
	var form moi.WorkflowAgentInputForm
	if err := json.Unmarshal([]byte(raw), &form); err != nil {
		return nil, fmt.Errorf("parse knowledge base workflow input form: %w", err)
	}
	return &form, nil
}
