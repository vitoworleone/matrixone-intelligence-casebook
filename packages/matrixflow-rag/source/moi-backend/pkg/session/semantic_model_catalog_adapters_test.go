package session

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	moi "github.com/matrixflow/moi-core/go-sdk"
	"github.com/matrixflow/moi-core/model/common"
	"github.com/matrixorigin/matrixflow/moi-backend/pkg/catalog"
	"github.com/matrixorigin/matrixflow/moi-backend/pkg/coreclient"
	"github.com/matrixorigin/matrixflow/moi-backend/pkg/ctxutil"
	"github.com/matrixorigin/matrixflow/moi-backend/pkg/dataconn"
	"github.com/matrixorigin/matrixflow/moi-backend/pkg/i18n"
	"github.com/matrixorigin/matrixflow/moi-backend/pkg/model"
	"github.com/matrixorigin/matrixflow/moi-backend/pkg/workflowv2"
)

func TestSemanticModelCatalogDataDomainAdapterSeparatesIAMCreateRequestIDs(t *testing.T) {
	svc := &fakeCatalogDataDomainServiceForSemanticModel{}
	adapter := NewSemanticModelCatalogDataDomainAdapter(svc)
	ctx := moi.ContextWithHeaders(context.Background(), map[string]string{
		"X-Request-ID":  "req-1",
		"X-Trace-ID":    "trace-1",
		"X-MOI-Role-ID": "role-1",
	})
	ctx = ctxutil.WithCoreIAMRequest(ctx, ctxutil.CoreIAMRequestContext{
		RequestID:               "req-1",
		TraceID:                 "trace-1",
		VerifiedEffectiveRoleID: "role-1",
	})

	if catalogID, err := adapter.ResolveDefaultCatalogID(ctx); err != nil {
		t.Fatalf("ResolveDefaultCatalogID: %v", err)
	} else if catalogID != 7 {
		t.Fatalf("ResolveDefaultCatalogID = %d, want reserved catalog 7", catalogID)
	}
	if svc.listCatalogsCalled {
		t.Fatal("ResolveDefaultCatalogID must not call the user-scoped catalog list")
	}
	if _, err := adapter.CreateDatabase(ctx, 7, "kb_42", "", ""); err != nil {
		t.Fatalf("CreateDatabase: %v", err)
	}
	if _, err := adapter.CreateVolume(ctx, 11, "raw_document", ""); err != nil {
		t.Fatalf("CreateVolume raw: %v", err)
	}
	if _, err := adapter.CreateVolume(ctx, 11, "processed", ""); err != nil {
		t.Fatalf("CreateVolume processed: %v", err)
	}

	want := []string{
		"req-1",
		"req-1.database-create-7-kb_42",
		"req-1.volume-create-11-raw_document",
		"req-1.volume-create-11-processed",
	}
	if strings.Join(svc.requestIDs, ",") != strings.Join(want, ",") {
		t.Fatalf("request IDs = %v, want %v", svc.requestIDs, want)
	}
	for i, headers := range svc.headers {
		if headers["X-Trace-ID"] != "trace-1" || headers["X-MOI-Role-ID"] != "role-1" {
			t.Fatalf("headers[%d] = %v, want trace and role preserved", i, headers)
		}
		trusted := svc.trustedIAM[i]
		if trusted.RequestID != want[i] || trusted.TraceID != "trace-1" || trusted.VerifiedEffectiveRoleID != "role-1" {
			t.Fatalf("trusted IAM[%d] = %+v, want derived request ID with trace and role preserved", i, trusted)
		}
	}

	longCtx := moi.ContextWithHeader(context.Background(), "X-Request-ID", strings.Repeat("r", 128))
	first := moi.HeadersFromContext(knowledgeBaseIAMSubrequestContext(longCtx, "catalog-create"))["X-Request-ID"]
	second := moi.HeadersFromContext(knowledgeBaseIAMSubrequestContext(longCtx, "catalog-create"))["X-Request-ID"]
	other := moi.HeadersFromContext(knowledgeBaseIAMSubrequestContext(longCtx, "database-create-7-kb_42"))["X-Request-ID"]
	if len(first) > 128 || first != second || first == other {
		t.Fatalf("hashed request IDs = %q, %q, %q", first, second, other)
	}
	if _, ok := ctxutil.KnowledgeBaseProvisioningCatalogIDFrom(knowledgeBaseIAMSubrequestContext(ctx, "volume-create-11-raw_document")); ok {
		t.Fatal("generic resource adapter must not create a knowledge-base provisioning scope")
	}
}

func TestSemanticModelCatalogDataDomainAdapterPreservesProvisioningScope(t *testing.T) {
	svc := &fakeCatalogDataDomainServiceForSemanticModel{}
	adapter := NewSemanticModelCatalogDataDomainAdapter(svc)
	ctx := ctxutil.WithKnowledgeBaseProvisioning(context.Background(), 7)

	if _, err := adapter.CreateDatabase(ctx, 7, "kb_42", "", ""); err != nil {
		t.Fatalf("CreateDatabase: %v", err)
	}
	if _, err := adapter.CreateVolume(ctx, 11, "raw_document", ""); err != nil {
		t.Fatalf("CreateVolume raw: %v", err)
	}
	if _, err := adapter.CreateVolume(ctx, 11, "processed", ""); err != nil {
		t.Fatalf("CreateVolume processed: %v", err)
	}
	if len(svc.provisioningCatalogIDs) != 3 {
		t.Fatalf("provisioning catalog IDs = %v, want three entries", svc.provisioningCatalogIDs)
	}
	for i, catalogID := range svc.provisioningCatalogIDs {
		if catalogID != 7 {
			t.Fatalf("provisioning catalog ID[%d] = %d, want 7", i, catalogID)
		}
	}
}

func TestKnowledgeBaseIAMSubrequestContextReachesCoreTransport(t *testing.T) {
	var gotRequestID, gotTraceID, gotRoleID string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRequestID = r.Header.Get("X-Request-ID")
		gotTraceID = r.Header.Get("X-Trace-ID")
		gotRoleID = r.Header.Get("X-MOI-Role-ID")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":12,"name":"raw_document","database_id":11}`))
	}))
	defer server.Close()

	ctx := moi.ContextWithHeaders(context.Background(), map[string]string{
		"X-Request-ID": "req-transport",
		"X-Trace-ID":   "trace-transport",
	})
	ctx = ctxutil.WithCoreIAMRequest(ctx, ctxutil.CoreIAMRequestContext{
		RequestID:               "req-transport",
		TraceID:                 "trace-transport",
		VerifiedEffectiveRoleID: "role-transport",
	})
	ctx = knowledgeBaseIAMSubrequestContext(ctx, "volume-create-11-raw_document")
	client, err := moi.New(server.URL, "test-api-key", i18n.LocaleRequestOption(ctx), i18n.CoreIAMRequestOption())
	if err != nil {
		t.Fatalf("NewMoiClient: %v", err)
	}
	defer client.Close()
	if _, err := client.Volumes().Create(ctx, "ws-1", 11, "raw_document"); err != nil {
		t.Fatalf("Create volume: %v", err)
	}

	if gotRequestID != "req-transport.volume-create-11-raw_document" {
		t.Fatalf("request ID = %q, want derived volume request ID", gotRequestID)
	}
	if gotTraceID != "trace-transport" || gotRoleID != "role-transport" {
		t.Fatalf("trace/role = %q/%q, want original trace and verified role", gotTraceID, gotRoleID)
	}
}

func TestSemanticModelAdaptersSeparateIAMDeleteRequestIDs(t *testing.T) {
	ctx := moi.ContextWithHeaders(context.Background(), map[string]string{
		"X-Request-ID": "req-delete",
		"X-Trace-ID":   "trace-delete",
	})
	ctx = ctxutil.WithCoreIAMRequest(ctx, ctxutil.CoreIAMRequestContext{
		RequestID:               "req-delete",
		TraceID:                 "trace-delete",
		VerifiedEffectiveRoleID: "role-delete",
	})

	catalogService := &fakeCatalogDataDomainServiceForSemanticModel{}
	catalogAdapter := NewSemanticModelCatalogDataDomainAdapter(catalogService)
	for _, volumeID := range []int64{12, 13} {
		if err := catalogAdapter.DeleteVolume(ctx, volumeID); err != nil {
			t.Fatalf("DeleteVolume(%d): %v", volumeID, err)
		}
	}
	if err := catalogAdapter.DeleteDatabase(ctx, 11); err != nil {
		t.Fatalf("DeleteDatabase(11): %v", err)
	}
	workflowService := &fakeWorkflowDeleteServiceForSemanticModel{}
	workflowAdapter := NewSemanticModelWorkflowAdapter(workflowService)
	for _, workflowID := range []string{"workflow-document", "workflow-audio"} {
		if err := workflowAdapter.DeleteWorkflow(ctx, workflowID); err != nil {
			t.Fatalf("DeleteWorkflow(%s): %v", workflowID, err)
		}
	}

	wantCatalogIDs := []string{"req-delete.volume-delete-12", "req-delete.volume-delete-13", "req-delete.database-delete-11"}
	if strings.Join(catalogService.requestIDs, ",") != strings.Join(wantCatalogIDs, ",") {
		t.Fatalf("catalog request IDs = %v, want %v", catalogService.requestIDs, wantCatalogIDs)
	}
	wantWorkflowIDs := []string{"req-delete.workflow-delete-workflow-document", "req-delete.workflow-delete-workflow-audio"}
	if strings.Join(workflowService.requestIDs, ",") != strings.Join(wantWorkflowIDs, ",") {
		t.Fatalf("workflow request IDs = %v, want %v", workflowService.requestIDs, wantWorkflowIDs)
	}
	for i, trusted := range append(catalogService.trustedIAM, workflowService.trustedIAM...) {
		if trusted.TraceID != "trace-delete" || trusted.VerifiedEffectiveRoleID != "role-delete" {
			t.Fatalf("trusted IAM[%d] = %+v, want trace and role preserved", i, trusted)
		}
	}
}

func TestSemanticModelWorkflowAdapterSeparatesIAMDeployRequestIDs(t *testing.T) {
	ctx := moi.ContextWithHeaders(context.Background(), map[string]string{
		"X-Request-ID": "req-deploy",
		"X-Trace-ID":   "trace-deploy",
	})
	ctx = ctxutil.WithCoreIAMRequest(ctx, ctxutil.CoreIAMRequestContext{
		RequestID:               "req-deploy",
		TraceID:                 "trace-deploy",
		VerifiedEffectiveRoleID: "role-deploy",
	})
	svc := &fakeWorkflowDeployServiceForSemanticModel{}
	adapter := NewSemanticModelWorkflowAdapter(svc)
	for _, workflowID := range []string{"workflow-audio", "workflow-audio", "workflow-video"} {
		if err := adapter.DeployKnowledgeBaseWorkflow(ctx, KnowledgeBaseWorkflowDeployRequest{WorkflowID: workflowID, InputFormJSON: `{}`}); err != nil {
			t.Fatalf("DeployKnowledgeBaseWorkflow(%s): %v", workflowID, err)
		}
	}
	want := []string{
		"req-deploy.workflow-deploy-workflow-audio",
		"req-deploy.workflow-deploy-workflow-audio",
		"req-deploy.workflow-deploy-workflow-video",
	}
	if strings.Join(svc.requestIDs, ",") != strings.Join(want, ",") {
		t.Fatalf("deploy request IDs = %v, want %v", svc.requestIDs, want)
	}
	for i, trusted := range svc.trustedIAM {
		if trusted.RequestID != want[i] || trusted.TraceID != "trace-deploy" || trusted.VerifiedEffectiveRoleID != "role-deploy" {
			t.Fatalf("trusted IAM[%d] = %+v", i, trusted)
		}
	}
}

type fakeCatalogDataDomainServiceForSemanticModel struct {
	catalog.DataCenterService
	requestIDs             []string
	headers                []map[string]string
	trustedIAM             []ctxutil.CoreIAMRequestContext
	provisioningCatalogIDs []int64
	listCatalogsCalled     bool
}

func (f *fakeCatalogDataDomainServiceForSemanticModel) capture(ctx context.Context) {
	headers := moi.HeadersFromContext(ctx)
	f.requestIDs = append(f.requestIDs, headers["X-Request-ID"])
	f.headers = append(f.headers, headers)
	trusted, _ := ctxutil.CoreIAMRequestFrom(ctx)
	f.trustedIAM = append(f.trustedIAM, trusted)
	catalogID, ok := ctxutil.KnowledgeBaseProvisioningCatalogIDFrom(ctx)
	if !ok {
		catalogID = 0
	}
	f.provisioningCatalogIDs = append(f.provisioningCatalogIDs, catalogID)
}

func (f *fakeCatalogDataDomainServiceForSemanticModel) ListCatalogs(ctx context.Context) (*catalog.ListCatalogsResponse, error) {
	f.listCatalogsCalled = true
	return nil, fmt.Errorf("ListCatalogs must not be used to resolve the default catalog")
}

func (f *fakeCatalogDataDomainServiceForSemanticModel) ResolveDefaultCatalogID(ctx context.Context) (int64, error) {
	f.capture(ctx)
	return 7, nil
}

func (f *fakeCatalogDataDomainServiceForSemanticModel) CreateDatabase(ctx context.Context, _ *catalog.CreateDatabaseRequest) (*catalog.CreateDatabaseResponse, error) {
	f.capture(ctx)
	id := 11
	return &catalog.CreateDatabaseResponse{ID: &id}, nil
}

func (f *fakeCatalogDataDomainServiceForSemanticModel) CreateVolume(ctx context.Context, _ *catalog.CreateVolumeRequest) (*catalog.CreateVolumeResponse, error) {
	f.capture(ctx)
	id := 12 + len(f.requestIDs)
	return &catalog.CreateVolumeResponse{ID: &id}, nil
}

func (f *fakeCatalogDataDomainServiceForSemanticModel) DeleteVolume(ctx context.Context, _ *catalog.DeleteByIDRequest) error {
	f.capture(ctx)
	return nil
}

func (f *fakeCatalogDataDomainServiceForSemanticModel) DeleteDatabase(ctx context.Context, _ *catalog.DeleteByIDRequest) error {
	f.capture(ctx)
	return nil
}

type fakeWorkflowDeleteServiceForSemanticModel struct {
	workflowv2.Service
	requestIDs []string
	trustedIAM []ctxutil.CoreIAMRequestContext
}

type fakeWorkflowDeployServiceForSemanticModel struct {
	workflowv2.Service
	requestIDs []string
	trustedIAM []ctxutil.CoreIAMRequestContext
}

func (f *fakeWorkflowDeployServiceForSemanticModel) DeployWorkflow(ctx context.Context, _ *workflowv2.DeployWorkflowRequest) (*workflowv2.WorkflowDeploymentEnvelope, error) {
	f.requestIDs = append(f.requestIDs, moi.HeadersFromContext(ctx)["X-Request-ID"])
	trusted, _ := ctxutil.CoreIAMRequestFrom(ctx)
	f.trustedIAM = append(f.trustedIAM, trusted)
	return &workflowv2.WorkflowDeploymentEnvelope{}, nil
}

func (f *fakeWorkflowDeleteServiceForSemanticModel) DeleteWorkflow(ctx context.Context, _ string) (*workflowv2.DeleteEnvelope, error) {
	f.requestIDs = append(f.requestIDs, moi.HeadersFromContext(ctx)["X-Request-ID"])
	trusted, _ := ctxutil.CoreIAMRequestFrom(ctx)
	f.trustedIAM = append(f.trustedIAM, trusted)
	return &workflowv2.DeleteEnvelope{Deleted: true}, nil
}

type flakyWorkflowDeleteServiceForSemanticModel struct {
	workflowv2.Service
	deleteCalls   int
	getCalls      int
	deleteErrors  []error
	getErr        error
	requestIDs    []string
	getRequestIDs []string
}

func (f *flakyWorkflowDeleteServiceForSemanticModel) DeleteWorkflow(ctx context.Context, _ string) (*workflowv2.DeleteEnvelope, error) {
	f.requestIDs = append(f.requestIDs, moi.HeadersFromContext(ctx)["X-Request-ID"])
	idx := f.deleteCalls
	f.deleteCalls++
	if idx < len(f.deleteErrors) && f.deleteErrors[idx] != nil {
		return nil, f.deleteErrors[idx]
	}
	return &workflowv2.DeleteEnvelope{Deleted: true}, nil
}

func (f *flakyWorkflowDeleteServiceForSemanticModel) GetWorkflow(ctx context.Context, workflowID string) (*workflowv2.WorkflowEnvelope, error) {
	f.getRequestIDs = append(f.getRequestIDs, moi.HeadersFromContext(ctx)["X-Request-ID"])
	f.getCalls++
	if f.getErr != nil {
		return nil, f.getErr
	}
	return &workflowv2.WorkflowEnvelope{Workflow: workflowv2.WorkflowDetail{ID: workflowID}}, nil
}

func TestIsIdempotentDeleteTransportError(t *testing.T) {
	if !isIdempotentDeleteTransportError(io.EOF) {
		t.Fatal("io.EOF should be treated as transport error")
	}
	if !isIdempotentDeleteTransportError(fmt.Errorf("Delete %q: %w", "http://moi-catalog:8081/...", io.EOF)) {
		t.Fatal("wrapped EOF should be treated as transport error")
	}
	if isIdempotentDeleteTransportError(errors.New("permission denied")) {
		t.Fatal("business errors must not be treated as transport errors")
	}
}

func TestSemanticModelWorkflowAdapterDeleteWorkflowRetriesEOFThenSucceeds(t *testing.T) {
	svc := &flakyWorkflowDeleteServiceForSemanticModel{
		deleteErrors: []error{io.EOF, nil},
	}
	adapter := NewSemanticModelWorkflowAdapter(svc)
	ctx := moi.ContextWithHeaders(context.Background(), map[string]string{
		"X-Request-ID": "req-eof",
		"X-Trace-ID":   "trace-eof",
	})
	ctx = ctxutil.WithCoreIAMRequest(ctx, ctxutil.CoreIAMRequestContext{
		RequestID:               "req-eof",
		TraceID:                 "trace-eof",
		VerifiedEffectiveRoleID: "role-eof",
	})
	if err := adapter.DeleteWorkflow(ctx, "kb-rag-workflow-abc"); err != nil {
		t.Fatalf("DeleteWorkflow: %v", err)
	}
	if svc.deleteCalls != 2 {
		t.Fatalf("deleteCalls = %d, want 2", svc.deleteCalls)
	}
	if svc.getCalls != 0 {
		t.Fatalf("getCalls = %d, want 0 (retry succeeded)", svc.getCalls)
	}
	wantID := "req-eof.workflow-delete-kb-rag-workflow-abc"
	for i, id := range svc.requestIDs {
		if id != wantID {
			t.Fatalf("delete requestIDs[%d] = %q, want %q", i, id, wantID)
		}
	}
}

func TestSemanticModelWorkflowAdapterDeleteWorkflowEOFConvergesWhenAlreadyDeleted(t *testing.T) {
	svc := &flakyWorkflowDeleteServiceForSemanticModel{
		deleteErrors: []error{io.EOF, io.EOF, io.EOF},
		getErr:       &moi.Error{Code: common.ErrorCode_NOT_FOUND, Message: "workflow not found"},
	}
	adapter := NewSemanticModelWorkflowAdapter(svc)
	ctx := moi.ContextWithHeaders(context.Background(), map[string]string{
		"X-Request-ID": "2cedbf62919884266c9229e87a7d0648",
		"X-Trace-ID":   "trace-kb-delete",
	})
	ctx = ctxutil.WithCoreIAMRequest(ctx, ctxutil.CoreIAMRequestContext{
		RequestID:               "2cedbf62919884266c9229e87a7d0648",
		TraceID:                 "trace-kb-delete",
		VerifiedEffectiveRoleID: "role-kb",
	})

	if err := adapter.DeleteWorkflow(ctx, "kb-rag-workflow-3c88315f123a7def52512013"); err != nil {
		t.Fatalf("DeleteWorkflow: %v", err)
	}
	if svc.deleteCalls != knowledgeBaseWorkflowDeleteTransportRetries+1 {
		t.Fatalf("deleteCalls = %d, want %d", svc.deleteCalls, knowledgeBaseWorkflowDeleteTransportRetries+1)
	}
	if svc.getCalls != 1 {
		t.Fatalf("getCalls = %d, want 1", svc.getCalls)
	}
	wantID := "2cedbf62919884266c9229e87a7d0648.workflow-delete-kb-rag-workflow-3c88315f123a7def52512013"
	for i, id := range svc.requestIDs {
		if id != wantID {
			t.Fatalf("delete requestIDs[%d] = %q, want %q", i, id, wantID)
		}
	}
	if len(svc.getRequestIDs) != 1 || svc.getRequestIDs[0] != wantID {
		t.Fatalf("get request IDs = %v, want [%q]", svc.getRequestIDs, wantID)
	}
}

func TestSemanticModelWorkflowAdapterDeleteWorkflowEOFStillExistsReturnsError(t *testing.T) {
	svc := &flakyWorkflowDeleteServiceForSemanticModel{
		deleteErrors: []error{io.EOF, io.EOF, io.EOF},
		getErr:       nil, // workflow still present
	}
	adapter := NewSemanticModelWorkflowAdapter(svc)
	ctx := moi.ContextWithHeaders(context.Background(), map[string]string{"X-Request-ID": "req-1", "X-Trace-ID": "tr-1"})
	ctx = ctxutil.WithCoreIAMRequest(ctx, ctxutil.CoreIAMRequestContext{RequestID: "req-1", TraceID: "tr-1", VerifiedEffectiveRoleID: "role-1"})
	err := adapter.DeleteWorkflow(ctx, "wf-still-there")
	if err == nil {
		t.Fatal("expected error when workflow still exists after transport failures")
	}
	if !strings.Contains(err.Error(), "EOF") && !errors.Is(err, io.EOF) {
		t.Fatalf("error should preserve transport failure, got %v", err)
	}
	if svc.getCalls != 1 {
		t.Fatalf("getCalls = %d, want 1", svc.getCalls)
	}
}

func TestParseKnowledgeBaseWorkflowInputFormRejectsInvalidJSON(t *testing.T) {
	_, err := parseKnowledgeBaseWorkflowInputForm("{invalid-json")
	if err == nil || !strings.Contains(err.Error(), "parse knowledge base workflow input form") {
		t.Fatalf("parseKnowledgeBaseWorkflowInputForm error = %v", err)
	}
}

func TestSemanticModelLocalFileImportAdapterStagesStructuredFileAndInjectsConnFileIDs(t *testing.T) {
	svc := &fakeConnectorServiceForSemanticModel{
		localUploadIDs: []string{"conn-file-1"},
		uploadTaskID:   "task-1",
	}
	adapter := NewSemanticModelLocalFileImportAdapter(svc)
	tableConfig := `{"sheet_name":"orders.csv","new_table":true,"database_id":11,"conn_file_ids":[],"create_table":{"name":"orders","tableColumn":[{"column":"id","dataType":"VARCHAR","col_num_in_file":1}]}}`

	result, err := adapter.UploadToVolume(context.Background(), KnowledgeBaseLocalFileImportParams{
		VolumeID:    42,
		FileName:    "orders.csv",
		Reader:      strings.NewReader("id\n1\n"),
		UploadKind:  kbLocalUploadKindStructured,
		TableConfig: tableConfig,
	})
	if err != nil {
		t.Fatalf("UploadToVolume: %v", err)
	}
	if result.TaskID != "task-1" || len(result.FileIDs) != 1 || result.FileIDs[0] != "conn-file-1" {
		t.Fatalf("result = %+v, want task-1 and staged conn file", result)
	}
	if svc.localUploadCalls != 1 {
		t.Fatalf("localUploadCalls = %d, want 1", svc.localUploadCalls)
	}
	if got := string(svc.localUploadContent); got != "id\n1\n" {
		t.Fatalf("local upload content = %q", got)
	}
	if len(svc.uploadParams.Files) != 0 {
		t.Fatalf("structured import task should not upload duplicate files, got %d", len(svc.uploadParams.Files))
	}
	if svc.uploadParams.VolumeID != "" {
		t.Fatalf("structured import task VolumeID = %q, want empty", svc.uploadParams.VolumeID)
	}
	var got model.TableConfig
	if err := json.Unmarshal([]byte(svc.uploadParams.TableConfigStr), &got); err != nil {
		t.Fatalf("unmarshal table_config: %v", err)
	}
	if len(got.ConnFileIds) != 1 || got.ConnFileIds[0] != "conn-file-1" {
		t.Fatalf("conn_file_ids = %v, want [conn-file-1]", got.ConnFileIds)
	}
}

func TestSemanticModelLocalFileImportAdapterUsesExistingStructuredConnFileIDs(t *testing.T) {
	svc := &fakeConnectorServiceForSemanticModel{uploadTaskID: "task-1"}
	adapter := NewSemanticModelLocalFileImportAdapter(svc)
	tableConfig := `{"sheet_name":"orders.csv","new_table":true,"database_id":11,"conn_file_ids":["existing-file-1"],"create_table":{"name":"orders","tableColumn":[{"column":"id","dataType":"VARCHAR","col_num_in_file":1}]}}`

	result, err := adapter.UploadToVolume(context.Background(), KnowledgeBaseLocalFileImportParams{
		VolumeID:    42,
		FileName:    "orders.csv",
		Reader:      strings.NewReader("id\n1\n"),
		UploadKind:  kbLocalUploadKindStructured,
		TableConfig: tableConfig,
	})
	if err != nil {
		t.Fatalf("UploadToVolume: %v", err)
	}
	if result.TaskID != "task-1" || len(result.FileIDs) != 1 || result.FileIDs[0] != "existing-file-1" {
		t.Fatalf("result = %+v, want existing file id", result)
	}
	if svc.localUploadCalls != 0 {
		t.Fatalf("localUploadCalls = %d, want 0", svc.localUploadCalls)
	}
	if len(svc.uploadParams.Files) != 0 {
		t.Fatalf("structured import task should not upload duplicate files, got %d", len(svc.uploadParams.Files))
	}
	if svc.uploadParams.VolumeID != "" {
		t.Fatalf("structured import task VolumeID = %q, want empty", svc.uploadParams.VolumeID)
	}
}

func TestSemanticModelCatalogFileAdapterListFilesBridgesSelectionFilters(t *testing.T) {
	svc := &fakeCatalogFileServiceForSemanticModel{
		listResp: &catalog.ListFilesResponse{
			Total: 2,
			List: []*catalog.FileFolderDTO{
				{ID: "file-1", Name: "report.pdf", VolumeID: "42"},
				{ID: "file-2", Name: "summary.docx", VolumeID: "42"},
			},
		},
	}
	adapter := NewSemanticModelCatalogFileAdapter(svc)

	resp, err := adapter.ListFiles(context.Background(), KnowledgeBaseCatalogFileListParams{
		VolumeID: 42,
		Page:     2,
		PageSize: 20,
		FileName: "quarterly",
		FileExt:  []string{"pdf", "docx"},
		FileIDs:  []string{"file-1", "file-2"},
	})
	if err != nil {
		t.Fatalf("ListFiles: %v", err)
	}
	if svc.listReq == nil {
		t.Fatal("ListFiles request was not captured")
	}
	if svc.listReq.Page != 2 || svc.listReq.PageSize != 20 {
		t.Fatalf("pagination = page %d size %d, want 2/20", svc.listReq.Page, svc.listReq.PageSize)
	}
	assertCatalogFileFilter(t, svc.listReq.Filters, "volume_id", []string{"42"}, false)
	assertCatalogFileFilter(t, svc.listReq.Filters, "file_name", []string{"quarterly"}, true)
	assertCatalogFileFilter(t, svc.listReq.Filters, "file_ext", []string{"pdf", "docx"}, false)
	assertCatalogFileFilter(t, svc.listReq.Filters, "file_id", []string{"file-1", "file-2"}, false)
	if resp.Total != 2 || len(resp.Items) != 2 || resp.Items[0].FileID != "file-1" || resp.Items[0].VolumeID != 42 {
		t.Fatalf("response = %+v, want bridged file leaves", resp)
	}
}

func TestSemanticModelCatalogFileAdapterPreviewUsesWorkspaceScopedSystemArtifactRoute(t *testing.T) {
	var requests []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path+" key="+r.Header.Get("X-API-Key"))
		if r.URL.Path != "/api/v1/system/workspaces/ws-current/semantic-model-artifacts/page-image-9/download" {
			http.Error(w, "regular file route is forbidden without volume.read", http.StatusForbidden)
			return
		}
		w.Header().Set("Content-Disposition", "attachment; filename*=UTF-8''page-image-9.png")
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte("png-bytes"))
	}))
	defer server.Close()

	if err := coreclient.Configure(coreclient.Config{
		Endpoint:     server.URL,
		SystemAPIKey: "system-key",
		HTTPClient:   server.Client(),
	}); err != nil {
		t.Fatalf("configure coreclient: %v", err)
	}

	adapter := NewSemanticModelCatalogFileAdapter(&fakeCatalogFileServiceForSemanticModel{})
	ctx := ctxutil.WithWorkspaceID(context.Background(), "ws-current")
	result, err := adapter.PreviewFile(ctx, "page-image-9")
	if err != nil {
		t.Fatalf("PreviewFile: %v", err)
	}
	defer result.Body.Close()
	body, err := io.ReadAll(result.Body)
	if err != nil {
		t.Fatalf("read preview body: %v", err)
	}
	if result.Filename != "page-image-9.png" || result.ContentType != "image/png" || string(body) != "png-bytes" {
		t.Fatalf("preview = (%q, %q, %q)", result.Filename, result.ContentType, string(body))
	}
	wantRequests := []string{"GET /api/v1/system/workspaces/ws-current/semantic-model-artifacts/page-image-9/download key=system-key"}
	if strings.Join(requests, "\n") != strings.Join(wantRequests, "\n") {
		t.Fatalf("core requests = %v, want %v", requests, wantRequests)
	}
}

type stubOfficeConverterForSemanticModel struct {
	calls []string
}

func (s *stubOfficeConverterForSemanticModel) ConvertToPDF(_ context.Context, fileBytes []byte, inputExt string) ([]byte, error) {
	s.calls = append(s.calls, inputExt)
	return []byte("pdf-from-" + inputExt + "-" + string(fileBytes)), nil
}

func TestSemanticModelCatalogFileAdapterPreviewConvertsOfficeToPDF(t *testing.T) {
	var requests []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		if r.URL.Path != "/api/v1/system/workspaces/ws-current/semantic-model-artifacts/deck-1/download" {
			http.Error(w, "unexpected path", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Disposition", "attachment; filename*=UTF-8''quarterly.pptx")
		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.presentationml.presentation")
		_, _ = w.Write([]byte("pptx-bytes"))
	}))
	defer server.Close()

	if err := coreclient.Configure(coreclient.Config{
		Endpoint:     server.URL,
		SystemAPIKey: "system-key",
		HTTPClient:   server.Client(),
	}); err != nil {
		t.Fatalf("configure coreclient: %v", err)
	}

	converter := &stubOfficeConverterForSemanticModel{}
	adapter := NewSemanticModelCatalogFileAdapter(catalog.NewFileService(converter))
	ctx := ctxutil.WithWorkspaceID(context.Background(), "ws-current")
	result, err := adapter.PreviewFile(ctx, "deck-1")
	if err != nil {
		t.Fatalf("PreviewFile: %v", err)
	}
	defer result.Body.Close()
	body, err := io.ReadAll(result.Body)
	if err != nil {
		t.Fatalf("read preview body: %v", err)
	}
	if result.Filename != "quarterly.pdf" || result.ContentType != "application/pdf" {
		t.Fatalf("preview headers = (%q, %q), want quarterly.pdf / application/pdf", result.Filename, result.ContentType)
	}
	if string(body) != "pdf-from-pptx-pptx-bytes" {
		t.Fatalf("preview body = %q, want converted pdf bytes", string(body))
	}
	if len(converter.calls) != 1 || converter.calls[0] != "pptx" {
		t.Fatalf("converter calls = %v, want [pptx]", converter.calls)
	}
	if len(requests) != 1 {
		t.Fatalf("core requests = %v, want one system download", requests)
	}
}

func TestSemanticModelCatalogFileAdapterPreviewOfficeRequiresConverter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Disposition", "attachment; filename*=UTF-8''notes.docx")
		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
		_, _ = w.Write([]byte("docx-bytes"))
	}))
	defer server.Close()

	if err := coreclient.Configure(coreclient.Config{
		Endpoint:     server.URL,
		SystemAPIKey: "system-key",
		HTTPClient:   server.Client(),
	}); err != nil {
		t.Fatalf("configure coreclient: %v", err)
	}

	adapter := NewSemanticModelCatalogFileAdapter(catalog.NewFileService(nil))
	ctx := ctxutil.WithWorkspaceID(context.Background(), "ws-current")
	_, err := adapter.PreviewFile(ctx, "doc-1")
	if err == nil {
		t.Fatal("PreviewFile() error = nil, want office converter missing")
	}
	if !strings.Contains(err.Error(), "office converter not configured") {
		t.Fatalf("PreviewFile() error = %v, want converter missing", err)
	}
}

func assertCatalogFileFilter(t *testing.T, filters []*catalog.FileFilter, name string, values []string, fuzzy bool) {
	t.Helper()
	for _, filter := range filters {
		if filter == nil || filter.Name != name {
			continue
		}
		if strings.Join(filter.Values, ",") != strings.Join(values, ",") {
			t.Fatalf("filter %s values = %+v, want %+v", name, filter.Values, values)
		}
		gotFuzzy := filter.Fuzzy != nil && *filter.Fuzzy
		if gotFuzzy != fuzzy {
			t.Fatalf("filter %s fuzzy = %v, want %v", name, gotFuzzy, fuzzy)
		}
		return
	}
	t.Fatalf("filter %s not found in %+v", name, filters)
}

type fakeCatalogFileServiceForSemanticModel struct {
	catalog.FileService
	listReq  *catalog.ListFilesRequest
	listResp *catalog.ListFilesResponse
	listErr  error
}

func (f *fakeCatalogFileServiceForSemanticModel) ListFiles(_ context.Context, req *catalog.ListFilesRequest) (*catalog.ListFilesResponse, error) {
	f.listReq = req
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.listResp, nil
}

// PreviewDownloadedFile passthrough keeps image/raw preview tests independent of
// a real FileService converter while still satisfying the FileService surface.
func (f *fakeCatalogFileServiceForSemanticModel) PreviewDownloadedFile(_ context.Context, filename, contentType string, body io.ReadCloser) (*catalog.PreviewFileResponse, error) {
	return &catalog.PreviewFileResponse{
		Filename:    filename,
		ContentType: contentType,
		Body:        body,
	}, nil
}

type fakeConnectorServiceForSemanticModel struct {
	localUploadIDs     []string
	localUploadCalls   int
	localUploadContent []byte
	localUploadErr     error
	uploadTaskID       string
	uploadParams       dataconn.UploadParams
	uploadErr          error
}

func (f *fakeConnectorServiceForSemanticModel) Get(context.Context, string) (*dataconn.Connector, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeConnectorServiceForSemanticModel) List(context.Context, dataconn.ConnectorListParams) (*dataconn.ConnectorListResult, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeConnectorServiceForSemanticModel) Create(context.Context, dataconn.ConnectorCreateParams) (*dataconn.Connector, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeConnectorServiceForSemanticModel) Update(context.Context, dataconn.ConnectorUpdateParams) error {
	return errors.New("not implemented")
}

func (f *fakeConnectorServiceForSemanticModel) Delete(context.Context, string) error {
	return errors.New("not implemented")
}

func (f *fakeConnectorServiceForSemanticModel) Validate(context.Context, dataconn.ConnectorCreateParams) (bool, error) {
	return false, errors.New("not implemented")
}

func (f *fakeConnectorServiceForSemanticModel) FileList(context.Context, dataconn.ConnectorFileListParams) (*dataconn.ConnectorFileListResult, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeConnectorServiceForSemanticModel) Upload(_ context.Context, params dataconn.UploadParams) (*dataconn.UploadResponse, error) {
	f.uploadParams = params
	if f.uploadErr != nil {
		return nil, f.uploadErr
	}
	return &dataconn.UploadResponse{TaskID: f.uploadTaskID, Success: true}, nil
}

func (f *fakeConnectorServiceForSemanticModel) LocalUpload(_ context.Context, files []*multipart.FileHeader) (*dataconn.LocalUploadResponse, error) {
	f.localUploadCalls++
	if f.localUploadErr != nil {
		return nil, f.localUploadErr
	}
	if len(files) > 0 {
		rc, err := files[0].Open()
		if err != nil {
			return nil, err
		}
		content, readErr := io.ReadAll(rc)
		closeErr := rc.Close()
		if readErr != nil {
			return nil, readErr
		}
		if closeErr != nil {
			return nil, closeErr
		}
		f.localUploadContent = content
	}
	return &dataconn.LocalUploadResponse{ConnFileIDs: f.localUploadIDs}, nil
}

func (f *fakeConnectorServiceForSemanticModel) PreviewFile(context.Context, dataconn.PreviewFileParams) (*dataconn.PreviewFileResult, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeConnectorServiceForSemanticModel) DBListDatabases(context.Context, dataconn.DBListDatabasesParams) ([]string, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeConnectorServiceForSemanticModel) DBListSchemas(context.Context, dataconn.DBListSchemasParams) ([]string, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeConnectorServiceForSemanticModel) DBListTables(context.Context, dataconn.DBListTablesParams) ([]string, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeConnectorServiceForSemanticModel) DBGetSourceSchema(context.Context, dataconn.DBGetSourceSchemaParams) ([]dataconn.ColumnSchema, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeConnectorServiceForSemanticModel) FileDownload(context.Context, string) (string, error) {
	return "", errors.New("not implemented")
}

func (f *fakeConnectorServiceForSemanticModel) FileStream(context.Context, string) (string, []byte, error) {
	return "", nil, errors.New("not implemented")
}

func (f *fakeConnectorServiceForSemanticModel) GetLangfuseSyncStatus(context.Context, string) (*dataconn.LangfuseSyncStatusResult, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeConnectorServiceForSemanticModel) GetLangfuseSyncSessions(context.Context, dataconn.LangfuseSyncSessionsParams) (*dataconn.LangfuseSyncSessionsResult, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeConnectorServiceForSemanticModel) GetLangfuseSyncSessionTurns(context.Context, string, string) (*dataconn.LangfuseSyncSessionTurnsResult, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeConnectorServiceForSemanticModel) RequireLangfuseConnector(context.Context, string) (*dataconn.Connector, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeConnectorServiceForSemanticModel) ActivateLangfuseSync(context.Context, string) (*dataconn.Connector, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeConnectorServiceForSemanticModel) FileDelete(context.Context, string) error {
	return errors.New("not implemented")
}
