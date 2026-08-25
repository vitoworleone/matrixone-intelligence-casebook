package session

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	mysqlDriver "github.com/go-sql-driver/mysql"
	moi "github.com/matrixflow/moi-core/go-sdk"
	"github.com/matrixflow/moi-core/model/common"
	"github.com/matrixflow/moi-core/model/internalservice"
	coresaga "github.com/matrixflow/moi-core/saga"
	sagastore "github.com/matrixflow/moi-core/saga/storage"
	"github.com/matrixorigin/matrixflow/moi-backend/pkg/coreclient"
	"github.com/matrixorigin/matrixflow/moi-backend/pkg/ctxutil"
	"github.com/matrixorigin/matrixflow/moi-backend/pkg/i18n"
	"github.com/matrixorigin/matrixflow/moi-backend/pkg/iampep"
	"github.com/matrixorigin/matrixflow/moi-backend/pkg/model"
	"github.com/matrixorigin/matrixflow/moi-backend/pkg/workflowv2"
	"github.com/mattn/go-sqlite3"
	"golang.org/x/text/language"
	gmysql "gorm.io/driver/mysql"
	gsqlite "gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// fakeSemanticModelActionAuthorizer records ReauthorizeAction calls for
// deferred RAG semantic_model.use / volume.read regression tests.
type fakeSemanticModelActionAuthorizer struct {
	mu          sync.Mutex
	calls       []struct{ workspaceID, actionID, resourceType, resourceID string }
	err         error
	errByAction map[string]error
	roleOut     string
}

// fakeSemanticModelCanonicalVolumeResolver maps child volume IDs to canonical
// roots for deferred volume.read tests. Identity when unmapped.
type fakeSemanticModelCanonicalVolumeResolver struct {
	roots map[int64]int64
	err   error
	calls []struct {
		workspaceID string
		volumeID    int64
	}
}

func (f *fakeSemanticModelCanonicalVolumeResolver) ResolveCanonicalRootVolume(_ context.Context, workspaceID string, volumeID int64) (int64, error) {
	if f != nil {
		f.calls = append(f.calls, struct {
			workspaceID string
			volumeID    int64
		}{workspaceID: workspaceID, volumeID: volumeID})
	}
	if f != nil && f.err != nil {
		return 0, f.err
	}
	if f != nil && f.roots != nil {
		if root, ok := f.roots[volumeID]; ok {
			return root, nil
		}
	}
	return volumeID, nil
}

func (f *fakeSemanticModelActionAuthorizer) ReauthorizeAction(ctx context.Context, workspaceID, actionID, resourceType, resourceID string) (context.Context, error) {
	f.mu.Lock()
	f.calls = append(f.calls, struct{ workspaceID, actionID, resourceType, resourceID string }{
		workspaceID: workspaceID, actionID: actionID, resourceType: resourceType, resourceID: resourceID,
	})
	f.mu.Unlock()
	if f.errByAction != nil {
		if err, ok := f.errByAction[actionID]; ok && err != nil {
			return ctx, err
		}
	}
	if f.err != nil {
		return ctx, f.err
	}
	roleOut := f.roleOut
	if roleOut == "" {
		if trusted, ok := ctxutil.CoreIAMRequestFrom(ctx); ok {
			roleOut = trusted.VerifiedEffectiveRoleID
		}
	}
	if roleOut != "" {
		trusted, _ := ctxutil.CoreIAMRequestFrom(ctx)
		return ctxutil.WithCoreIAMRequest(ctx, ctxutil.CoreIAMRequestContext{
			RequestID:                trusted.RequestID,
			TraceID:                  trusted.TraceID,
			VerifiedEffectiveRoleID:  roleOut,
			IsWorkspaceOwner:         trusted.IsWorkspaceOwner,
			WorkspaceAccessVerified:  true,
			BusinessActionAuthorized: true,
		}), nil
	}
	return ctx, nil
}

type fakeSemanticModelWorkflowTemplateService struct {
	template *model.WorkflowTemplate
	err      error
	calls    []string
}

type fakeSemanticModelWorkflowV2Service struct {
	workflowv2.Service
	service                workflowv2.Service
	getWorkflow            func(ctx context.Context, workflowID string) (*workflowv2.WorkflowEnvelope, error)
	validateWorkflowDelete func(ctx context.Context, workflowID string) error
	updateWorkflow         func(ctx context.Context, workflowID string, req *workflowv2.UpdateWorkflowRequest) (*workflowv2.WorkflowEnvelope, error)
}

func (f *fakeSemanticModelWorkflowV2Service) ListWorkflowRuns(ctx context.Context, workflowID string, req *workflowv2.ListWorkflowRunsRequest) (*workflowv2.WorkflowExecutionListEnvelope, error) {
	if f.service != nil {
		return f.service.ListWorkflowRuns(ctx, workflowID, req)
	}
	return &workflowv2.WorkflowExecutionListEnvelope{}, nil
}

func (f *fakeSemanticModelWorkflowV2Service) GetWorkflow(ctx context.Context, workflowID string) (*workflowv2.WorkflowEnvelope, error) {
	if f.getWorkflow != nil {
		return f.getWorkflow(ctx, workflowID)
	}
	if f.service != nil {
		return f.service.GetWorkflow(ctx, workflowID)
	}
	return &workflowv2.WorkflowEnvelope{}, nil
}

func (f *fakeSemanticModelWorkflowV2Service) UpdateWorkflow(ctx context.Context, workflowID string, req *workflowv2.UpdateWorkflowRequest) (*workflowv2.WorkflowEnvelope, error) {
	if f.updateWorkflow == nil {
		return nil, errors.New("unexpected UpdateWorkflow call")
	}
	return f.updateWorkflow(ctx, workflowID, req)
}

func (f *fakeSemanticModelWorkflowV2Service) ValidateWorkflowDelete(ctx context.Context, workflowID string) error {
	if f.validateWorkflowDelete == nil {
		return nil
	}
	return f.validateWorkflowDelete(ctx, workflowID)
}

func (f *fakeSemanticModelWorkflowTemplateService) GetByTemplateKey(_ context.Context, templateKey string) (*model.WorkflowTemplate, error) {
	f.calls = append(f.calls, templateKey)
	if f.err != nil {
		return nil, f.err
	}
	if f.template != nil {
		return f.template, nil
	}
	if templateKey == kbStandardRAGImageTemplateKey {
		return &model.WorkflowTemplate{
			TemplateKey: kbStandardRAGImageTemplateKey,
			Name:        "文档知识库准备（含图片索引）",
			Description: "standard rag with image index",
			DSLYaml:     "workflow:\n  name: standard-rag-image-index-pipeline\n  root: root\nroot:\n  chain:\n    - work_item:\n        name: parse_documents\n        id: moi:document.parse\n    - work_item:\n        name: build_index\n        id: moi:knowledge.index.build\n    - work_item:\n        name: build_image_index\n        id: moi:document_visual.index.image\n",
			InputForm:   `{"fields":[{"field_id":"source_ref","required":true,"bind_to":"vars.source_ref"},{"field_id":"vector_index","required":true,"bind_to":{"vector_table":"vars.vector_index.vector_table","embedding_model":"vars.vector_index.embedding_model","image_index_enabled":"vars.image_index.enabled","image_vector_table":"vars.image_index.image_vector_table","image_embedding_model":"vars.image_index.image_embedding_model","image_embedding_backend_id":"vars.image_index.image_embedding_backend_id","image_embedding_dimension":"vars.image_index.image_embedding_dimension","image_preprocess_version":"vars.image_index.image_preprocess_version","image_distance_metric":"vars.image_index.image_distance_metric"}},{"field_id":"output_ref","required":true,"bind_to":"vars.output_ref"}]}`,
			IsBuiltin:   true,
		}, nil
	}
	return &model.WorkflowTemplate{
		TemplateKey: kbStandardRAGTemplateKey,
		Name:        "文档知识库准备",
		Description: "standard rag",
		DSLYaml:     "workflow:\n  name: standard-rag-pipeline\n  root: root\nroot:\n  chain:\n    - work_item:\n        name: parse_documents\n        id: moi:document.parse\n    - work_item:\n        name: build_index\n        id: moi:knowledge.index.build\n",
		InputForm:   `{"fields":[{"field_id":"source_ref","required":true,"bind_to":"vars.source_ref"},{"field_id":"vector_index","required":true,"bind_to":{"vector_table":"vars.vector_index.vector_table","embedding_model":"vars.vector_index.embedding_model"}},{"field_id":"output_ref","required":true,"bind_to":"vars.output_ref"}]}`,
		IsBuiltin:   true,
	}, nil
}

type fakeSemanticModelWorkflowService struct {
	mu                     sync.Mutex
	deploys                []KnowledgeBaseWorkflowDeployRequest
	requires               []string
	validates              []string
	deletes                []string
	err                    error
	requireErr             error
	onDeploy               func(KnowledgeBaseWorkflowDeployRequest)
	fileExecutions         map[string]*moi.FileExecutionsResponse
	listFileExecutionCalls []string
	listFileExecutionsErr  error
	onListFileExecutions   func(context.Context, string) error
	validateErr            error
	deleteErr              error
	runs                   []struct {
		moiUserID       string
		effectiveRoleID string
		workflowID      string
		values          map[string]any
	}
	runResult *KnowledgeBaseWorkflowRunResult
	runErr    error
}

func (f *fakeSemanticModelWorkflowService) RunKnowledgeBaseWorkflow(ctx context.Context, workflowID string, values map[string]any) (*KnowledgeBaseWorkflowRunResult, error) {
	trustedIAM, _ := ctxutil.CoreIAMRequestFrom(ctx)
	f.runs = append(f.runs, struct {
		moiUserID       string
		effectiveRoleID string
		workflowID      string
		values          map[string]any
	}{
		moiUserID:       ctxutil.MoiUserIDFrom(ctx),
		effectiveRoleID: trustedIAM.VerifiedEffectiveRoleID,
		workflowID:      workflowID,
		values:          values,
	})
	if f.runErr != nil {
		return nil, f.runErr
	}
	if f.runResult != nil {
		return f.runResult, nil
	}
	return &KnowledgeBaseWorkflowRunResult{ExecutionID: "exec-workflow"}, nil
}

func (f *fakeSemanticModelWorkflowService) DeployKnowledgeBaseWorkflow(_ context.Context, params KnowledgeBaseWorkflowDeployRequest) error {
	f.deploys = append(f.deploys, params)
	if f.onDeploy != nil {
		f.onDeploy(params)
	}
	return f.err
}

func (f *fakeSemanticModelWorkflowService) RequireKnowledgeBaseWorkflow(_ context.Context, workflowID string) error {
	f.requires = append(f.requires, workflowID)
	return f.requireErr
}

func assertDocumentAppendPreservesExistingWorkflow(t *testing.T, templateSvc *fakeSemanticModelWorkflowTemplateService, workflowSvc *fakeSemanticModelWorkflowService, wsID string, modelID int64) {
	t.Helper()
	if len(templateSvc.calls) != 0 || len(workflowSvc.deploys) != 0 {
		t.Fatalf("ordinary append replaced the existing workflow: template calls=%+v deploys=%+v", templateSvc.calls, workflowSvc.deploys)
	}
	wantWorkflowID := knowledgeBaseWorkflowID(wsID, modelID)
	if !reflect.DeepEqual(workflowSvc.requires, []string{wantWorkflowID}) {
		t.Fatalf("required workflows = %+v, want exactly deterministic workflow %q", workflowSvc.requires, wantWorkflowID)
	}
}

func assertStructuredOnlyAppendDoesNotTouchDocumentWorkflow(t *testing.T, templateSvc *fakeSemanticModelWorkflowTemplateService, workflowSvc *fakeSemanticModelWorkflowService) {
	t.Helper()
	if len(workflowSvc.requires) != 0 || len(templateSvc.calls) != 0 || len(workflowSvc.deploys) != 0 {
		t.Fatalf("structured-only append touched document workflow: requires=%+v template calls=%+v deploys=%+v", workflowSvc.requires, templateSvc.calls, workflowSvc.deploys)
	}
}

func assertPublicDocumentMutationFailsClosedAtWorkflowGuard(t *testing.T, requireErr error, mutate func(context.Context, SemanticModelService) error) {
	t.Helper()
	var semanticGetCount int
	var semanticPutCount int
	var unexpectedCoreRequests []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			semanticGetCount++
			requireSemanticModelExecutionHeaders(t, r)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id": 77, "name": "kb_docs", "description": "docs", "tables": []any{},
				"files": map[string]any{"file_ids": []string{}, "vector_table": "existing_vector", "embedding_model": "existing_embedding"},
			})
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			semanticPutCount++
			w.WriteHeader(http.StatusInternalServerError)
		default:
			unexpectedCoreRequests = append(unexpectedCoreRequests, r.Method+" "+r.URL.String())
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	dataDomainSvc := &fakeSemanticModelDataDomainService{}
	workflowTemplateSvc := &fakeSemanticModelWorkflowTemplateService{}
	workflowSvc := &fakeSemanticModelWorkflowService{requireErr: requireErr}
	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()
	svc := newSemanticModelTestServiceWithKnowledgeBaseRuntimeDependencies(t, server.URL, systemClient, dataDomainSvc, nil, nil, workflowTemplateSvc, workflowSvc)

	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	// Domain must be ready before the workflow guard runs so empty-KB append can
	// create missing resources first, then fail closed only on non-not-found Require errors.
	tenantMock.ExpectQuery("SELECT model_id, catalog_id, database_id, raw_volume_id, processed_volume_id, ensure_status, last_ensure_error, last_checked_at").
		WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows([]string{
			"model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "ensure_status", "last_ensure_error", "last_checked_at",
		}).AddRow(int64(77), int64(3), int64(11), int64(12), int64(13), kbEnsureStatusReady, nil, int64(100)))
	// Fully bound ready domains still reconcile the document raw-volume row.
	expectUpsertKnowledgeBaseRawVolume(tenantMock, 77, kbRawKindDocument, false)

	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithUserID(ctx, "user-1")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)

	err = mutate(ctx, svc)
	if !errors.Is(err, requireErr) {
		t.Fatalf("public document mutation error = %v, want workflow guard error %v", err, requireErr)
	}
	if semanticGetCount > 1 || semanticPutCount != 0 || len(unexpectedCoreRequests) != 0 {
		t.Fatalf("core side effects after workflow guard failure: semantic GET=%d PUT=%d unexpected=%+v", semanticGetCount, semanticPutCount, unexpectedCoreRequests)
	}
	if len(dataDomainSvc.calls) != 0 {
		t.Fatalf("data-domain resource side effects = %+v, want none", dataDomainSvc.calls)
	}
	assertDocumentAppendPreservesExistingWorkflow(t, workflowTemplateSvc, workflowSvc, "ws-1", 77)
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant side effects: %v", err)
	}
}

func (f *fakeSemanticModelWorkflowService) ValidateWorkflowDelete(_ context.Context, workflowID string) error {
	f.validates = append(f.validates, workflowID)
	if f.validateErr != nil {
		return f.validateErr
	}
	return f.err
}

func (f *fakeSemanticModelWorkflowService) DeleteWorkflow(_ context.Context, workflowID string) error {
	f.deletes = append(f.deletes, workflowID)
	if f.deleteErr != nil {
		return f.deleteErr
	}
	return f.err
}

func (f *fakeSemanticModelWorkflowService) ListFileExecutions(ctx context.Context, fileID string, _ int64) (*moi.FileExecutionsResponse, error) {
	f.mu.Lock()
	f.listFileExecutionCalls = append(f.listFileExecutionCalls, fileID)
	err := f.listFileExecutionsErr
	resp := f.fileExecutions[fileID]
	f.mu.Unlock()
	if f.onListFileExecutions != nil {
		if hookErr := f.onListFileExecutions(ctx, fileID); hookErr != nil {
			return nil, hookErr
		}
	}
	if err != nil {
		return nil, err
	}
	if f.fileExecutions != nil {
		if resp != nil {
			return resp, nil
		}
	}
	return &moi.FileExecutionsResponse{Executions: []moi.FileExecutionSummary{}}, nil
}

func TestSemanticModelWorkflowAdapterValidateWorkflowDeleteDelegatesOwnerPreflight(t *testing.T) {
	var calls []string
	conflictErr := &moi.Error{Code: common.ErrorCode_ALREADY_EXISTS, Message: "workflow wf-1 has active volume dispatch job job-1 in status waiting"}
	service := &fakeSemanticModelWorkflowV2Service{validateWorkflowDelete: func(_ context.Context, workflowID string) error {
		if workflowID != "wf-1" {
			t.Fatalf("workflowID = %s, want wf-1", workflowID)
		}
		calls = append(calls, workflowID)
		return conflictErr
	}}
	adapter := NewSemanticModelWorkflowAdapter(service)

	err := adapter.ValidateWorkflowDelete(context.Background(), "wf-1")
	if !errors.Is(err, conflictErr) {
		t.Fatalf("ValidateWorkflowDelete error = %v, want %v", err, conflictErr)
	}
	if len(calls) != 1 || calls[0] != "wf-1" {
		t.Fatalf("validate calls = %+v, want wf-1", calls)
	}
}

func TestSemanticModelWorkflowAdapterValidateWorkflowDeleteDoesNotUpdateWorkflow(t *testing.T) {
	var calls []string
	service := &fakeSemanticModelWorkflowV2Service{
		validateWorkflowDelete: func(_ context.Context, workflowID string) error {
			calls = append(calls, workflowID)
			return nil
		},
		updateWorkflow: func(_ context.Context, workflowID string, req *workflowv2.UpdateWorkflowRequest) (*workflowv2.WorkflowEnvelope, error) {
			t.Fatalf("UpdateWorkflow must not be called, got workflowID=%s req=%+v", workflowID, req)
			return nil, nil
		},
	}
	adapter := NewSemanticModelWorkflowAdapter(service)

	if err := adapter.ValidateWorkflowDelete(context.Background(), "wf-1"); err != nil {
		t.Fatalf("ValidateWorkflowDelete: %v", err)
	}
	if len(calls) != 1 || calls[0] != "wf-1" {
		t.Fatalf("validate calls = %+v, want wf-1", calls)
	}
}

func TestSemanticModelWorkflowAdapterRequiresExactExistingWorkflow(t *testing.T) {
	upstreamErr := errors.New("get workflow failed")
	tests := []struct {
		name         string
		response     *workflowv2.WorkflowEnvelope
		getErr       error
		wantErr      bool
		wantUpstream bool
	}{
		{name: "matching workflow", response: &workflowv2.WorkflowEnvelope{Workflow: workflowv2.WorkflowDetail{ID: "wf-1"}}},
		{name: "nil workflow", wantErr: true},
		{name: "empty workflow id", response: &workflowv2.WorkflowEnvelope{}, wantErr: true},
		{name: "mismatched workflow id", response: &workflowv2.WorkflowEnvelope{Workflow: workflowv2.WorkflowDetail{ID: "wf-other"}}, wantErr: true},
		{name: "upstream failure", getErr: upstreamErr, wantErr: true, wantUpstream: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var calls []string
			service := &fakeSemanticModelWorkflowV2Service{getWorkflow: func(_ context.Context, workflowID string) (*workflowv2.WorkflowEnvelope, error) {
				calls = append(calls, workflowID)
				return tt.response, tt.getErr
			}}
			adapter := NewSemanticModelWorkflowAdapter(service)
			requirer, ok := adapter.(interface {
				RequireKnowledgeBaseWorkflow(context.Context, string) error
			})
			if !ok {
				t.Fatal("semantic-model workflow adapter does not expose an existence-only knowledge-base workflow guard")
			}

			err := requirer.RequireKnowledgeBaseWorkflow(context.Background(), "wf-1")
			if tt.wantErr && err == nil {
				t.Fatal("RequireKnowledgeBaseWorkflow error is nil, want fail-closed error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("RequireKnowledgeBaseWorkflow: %v", err)
			}
			if tt.wantUpstream && !errors.Is(err, upstreamErr) {
				t.Fatalf("RequireKnowledgeBaseWorkflow error = %v, want upstream error %v", err, upstreamErr)
			}
			if !reflect.DeepEqual(calls, []string{"wf-1"}) {
				t.Fatalf("GetWorkflow calls = %+v, want wf-1", calls)
			}
		})
	}
}

type fakeSemanticModelDataDomainService struct {
	defaultCatalogID       int64
	databaseCatalogID      int64
	databaseCatalogErr     error
	databaseID             int64
	databaseName           string
	databaseDisplay        string
	databaseByName         map[string]int64
	databaseErr            error
	volumeIDs              []int64
	volumeByName           map[string]int64
	volumeErrs             map[string]error
	calls                  []string
	provisioningCatalogIDs []int64
	defaultCatalogErr      error
	cloneErr               error
	cloneTargetTable       string
	deleteErr              error
	deleteVolumeErr        error
	deleteDatabaseErr      error
	beforeDelete           func() error
	listTableLeaves        func(KnowledgeBaseTableLeafListParams) (*KnowledgeBaseTableLeafListResult, error)
}

func (f *fakeSemanticModelDataDomainService) ResolveDefaultCatalogID(_ context.Context) (int64, error) {
	f.calls = append(f.calls, "resolve_default_catalog")
	if f.defaultCatalogErr != nil {
		return 0, f.defaultCatalogErr
	}
	if f.defaultCatalogID > 0 {
		return f.defaultCatalogID, nil
	}
	return 3, nil
}

func (f *fakeSemanticModelDataDomainService) ResolveCatalogIDByDatabaseID(_ context.Context, databaseID int64) (int64, error) {
	if f.databaseCatalogErr != nil {
		f.calls = append(f.calls, fmt.Sprintf("database_catalog:%d", databaseID))
		return 0, f.databaseCatalogErr
	}
	if f.databaseCatalogID > 0 {
		f.calls = append(f.calls, fmt.Sprintf("database_catalog:%d", databaseID))
		return f.databaseCatalogID, nil
	}
	return 3, nil
}

func (f *fakeSemanticModelDataDomainService) CreateDatabase(ctx context.Context, catalogID int64, name, description, displayName string) (int64, error) {
	f.calls = append(f.calls, "database:"+name)
	f.recordProvisioningCatalogID(ctx)
	if catalogID != 3 || description != "docs" {
		return 0, errors.New("unexpected database create request")
	}
	if f.databaseErr != nil {
		return 0, f.databaseErr
	}
	if f.databaseName != "" && name != f.databaseName {
		return 0, errors.New("unexpected database name")
	}
	if f.databaseName == "" {
		f.databaseName = name
	}
	f.databaseDisplay = displayName
	return f.databaseID, nil
}

func (f *fakeSemanticModelDataDomainService) ResolveDatabaseByName(_ context.Context, catalogID int64, name string) (int64, string, bool, error) {
	f.calls = append(f.calls, "resolve-database:"+name)
	expectedCatalogID := int64(3)
	if f.defaultCatalogID > 0 {
		expectedCatalogID = f.defaultCatalogID
	}
	if catalogID != expectedCatalogID {
		return 0, "", false, errors.New("unexpected catalog id")
	}
	if f.databaseByName == nil {
		return 0, "", false, nil
	}
	id := f.databaseByName[name]
	return id, "默认/" + name, id > 0, nil
}

func (f *fakeSemanticModelDataDomainService) CreateVolume(ctx context.Context, databaseID int64, name, description string) (int64, error) {
	f.calls = append(f.calls, name+":"+description)
	f.recordProvisioningCatalogID(ctx)
	if databaseID != f.databaseID {
		return 0, errors.New("unexpected database id")
	}
	if f.volumeErrs != nil && f.volumeErrs[name] != nil {
		return 0, f.volumeErrs[name]
	}
	if len(f.volumeIDs) == 0 {
		return 0, errors.New("no fake volume id")
	}
	id := f.volumeIDs[0]
	f.volumeIDs = f.volumeIDs[1:]
	return id, nil
}

func (f *fakeSemanticModelDataDomainService) recordProvisioningCatalogID(ctx context.Context) {
	catalogID, ok := ctxutil.KnowledgeBaseProvisioningCatalogIDFrom(ctx)
	if !ok {
		catalogID = 0
	}
	f.provisioningCatalogIDs = append(f.provisioningCatalogIDs, catalogID)
}

func (f *fakeSemanticModelDataDomainService) ResolveVolumeIDByName(_ context.Context, databaseID int64, name string) (int64, bool, error) {
	f.calls = append(f.calls, "resolve-volume:"+name)
	if databaseID != f.databaseID {
		return 0, false, errors.New("unexpected database id")
	}
	if f.volumeByName == nil {
		return 0, false, nil
	}
	id := f.volumeByName[name]
	return id, id > 0, nil
}

func (f *fakeSemanticModelDataDomainService) ListDatabaseTableLeaves(_ context.Context, params KnowledgeBaseTableLeafListParams) (*KnowledgeBaseTableLeafListResult, error) {
	if f.listTableLeaves != nil {
		return f.listTableLeaves(params)
	}
	return &KnowledgeBaseTableLeafListResult{
		Items: []KnowledgeBaseTableLeaf{
			{TableID: 1001, TableName: "orders", DatabaseID: params.DatabaseID},
		},
		Total: 1,
	}, nil
}

func TestSemanticModelServiceExpandCrossSearchSelectionsAsUnion(t *testing.T) {
	t.Run("files", func(t *testing.T) {
		fileSvc := &fakeSemanticModelFileService{listFiles: func(params KnowledgeBaseCatalogFileListParams) (*KnowledgeBaseCatalogFileListResult, error) {
			if len(params.FileIDs) > 0 {
				return &KnowledgeBaseCatalogFileListResult{Items: []KnowledgeBaseCatalogFileLeaf{{FileID: params.FileIDs[0], FileName: params.FileIDs[0] + ".pdf", VolumeID: params.VolumeID}}, Total: 1}, nil
			}
			start, end := 1, 100
			if params.FileName == "2026" {
				start, end = 91, 140
			}
			items := make([]KnowledgeBaseCatalogFileLeaf, 0, end-start+1)
			for id := start; id <= end; id++ {
				items = append(items, KnowledgeBaseCatalogFileLeaf{FileID: fmt.Sprintf("file-%d", id), FileName: fmt.Sprintf("file-%d.pdf", id), VolumeID: params.VolumeID})
			}
			return &KnowledgeBaseCatalogFileListResult{Items: items, Total: len(items)}, nil
		}}
		svc := &semanticModelService{fileService: fileSvc}
		sources, err := svc.expandSemanticModelSourceSelections(context.Background(), nil, "ws-1", 0, []SemanticModelSourceSelectionRequest{
			{Kind: kbSelectionKindVolumeFiles, VolumeID: 42, AllSelected: true, Filters: SemanticModelSourceSelectionFilters{FileName: "report"}},
			{Kind: kbSelectionKindVolumeFiles, VolumeID: 42, AllSelected: true, Filters: SemanticModelSourceSelectionFilters{FileName: "2026"}},
		}, nil)
		if err != nil {
			t.Fatalf("expandSemanticModelSourceSelections: %v", err)
		}
		if len(sources) != 140 {
			t.Fatalf("source count = %d, want 140 unique files", len(sources))
		}
		sources, err = svc.expandSemanticModelSourceSelections(context.Background(), nil, "ws-1", 0, []SemanticModelSourceSelectionRequest{
			{Kind: kbSelectionKindVolumeFiles, VolumeID: 42, SelectedFileIDs: []string{"file-1"}},
			{Kind: kbSelectionKindVolumeFiles, VolumeID: 42, AllSelected: true, ExcludedFileIDs: []string{"file-95"}, Filters: SemanticModelSourceSelectionFilters{FileName: "report"}},
			{Kind: kbSelectionKindVolumeFiles, VolumeID: 42, AllSelected: true, ExcludedFileIDs: []string{"file-95"}, Filters: SemanticModelSourceSelectionFilters{FileName: "2026"}},
		}, nil)
		if err != nil {
			t.Fatalf("expandSemanticModelSourceSelections with exclusions: %v", err)
		}
		if len(sources) != 139 {
			t.Fatalf("source count = %d, want 139 after excluding the shared file and deduplicating explicit file-1", len(sources))
		}
	})

	t.Run("tables", func(t *testing.T) {
		dataDomain := &fakeSemanticModelDataDomainService{listTableLeaves: func(params KnowledgeBaseTableLeafListParams) (*KnowledgeBaseTableLeafListResult, error) {
			start, end := int64(1), int64(100)
			if params.Search == "2026" {
				start, end = 91, 140
			}
			items := make([]KnowledgeBaseTableLeaf, 0, end-start+1)
			for id := start; id <= end; id++ {
				items = append(items, KnowledgeBaseTableLeaf{TableID: id, TableName: fmt.Sprintf("table-%d", id), DatabaseID: params.DatabaseID})
			}
			return &KnowledgeBaseTableLeafListResult{Items: items, Total: len(items)}, nil
		}}
		svc := &semanticModelService{dataDomainService: dataDomain}
		sources, err := svc.expandSemanticModelSourceSelections(context.Background(), nil, "ws-1", 0, []SemanticModelSourceSelectionRequest{
			{Kind: kbSelectionKindDatabaseTables, DatabaseID: 11, AllSelected: true, Filters: SemanticModelSourceSelectionFilters{TableName: "report"}},
			{Kind: kbSelectionKindDatabaseTables, DatabaseID: 11, AllSelected: true, Filters: SemanticModelSourceSelectionFilters{TableName: "2026"}},
		}, nil)
		if err != nil {
			t.Fatalf("expandSemanticModelSourceSelections: %v", err)
		}
		if len(sources) != 140 {
			t.Fatalf("source count = %d, want 140 unique tables", len(sources))
		}
		sources, err = svc.expandSemanticModelSourceSelections(context.Background(), nil, "ws-1", 0, []SemanticModelSourceSelectionRequest{
			{Kind: kbSelectionKindDatabaseTables, DatabaseID: 11, AllSelected: true, ExcludedTableIDs: []int64{95}, Filters: SemanticModelSourceSelectionFilters{TableName: "report"}},
			{Kind: kbSelectionKindDatabaseTables, DatabaseID: 11, AllSelected: true, ExcludedTableIDs: []int64{95}, Filters: SemanticModelSourceSelectionFilters{TableName: "2026"}},
		}, nil)
		if err != nil {
			t.Fatalf("expandSemanticModelSourceSelections with exclusions: %v", err)
		}
		if len(sources) != 139 {
			t.Fatalf("source count = %d, want 139 after excluding the shared table", len(sources))
		}
	})
}

func (f *fakeSemanticModelDataDomainService) CloneTableForKnowledgeBase(_ context.Context, sourceTableID, targetDatabaseID int64, idempotencyKey string) (*KnowledgeBaseTableCloneResult, error) {
	f.calls = append(f.calls, "table:"+idempotencyKey)
	if f.cloneErr != nil {
		return nil, f.cloneErr
	}
	if sourceTableID != 1001 || targetDatabaseID != f.databaseID {
		return nil, errors.New("unexpected table clone request")
	}
	targetDB := f.databaseName
	if targetDB == "" {
		targetDB = "kb_docs"
	}
	targetID := int64(2001)
	targetTable := f.cloneTargetTable
	if targetTable == "" {
		targetTable = "orders"
	}
	return &KnowledgeBaseTableCloneResult{
		OperationID: "table-clone-op-1",
		Status:      "succeeded",
		SourceDB:    "catalog_db",
		SourceTable: "orders",
		TargetDB:    targetDB,
		TargetTable: targetTable,
		TargetID:    &targetID,
	}, nil
}

func (f *fakeSemanticModelDataDomainService) DeleteVolume(_ context.Context, volumeID int64) error {
	if f.beforeDelete != nil {
		if err := f.beforeDelete(); err != nil {
			return err
		}
	}
	f.calls = append(f.calls, fmt.Sprintf("delete-volume:%d", volumeID))
	if f.deleteVolumeErr != nil {
		return f.deleteVolumeErr
	}
	return f.deleteErr
}

func (f *fakeSemanticModelDataDomainService) DeleteDatabase(_ context.Context, databaseID int64) error {
	if f.beforeDelete != nil {
		if err := f.beforeDelete(); err != nil {
			return err
		}
	}
	f.calls = append(f.calls, fmt.Sprintf("delete-database:%d", databaseID))
	if f.deleteDatabaseErr != nil {
		return f.deleteDatabaseErr
	}
	return f.deleteErr
}

type fakeSemanticModelFileService struct {
	deleted      []string
	deleteErr    error
	beforeDelete func() error
	listFiles    func(KnowledgeBaseCatalogFileListParams) (*KnowledgeBaseCatalogFileListResult, error)
	listCalls    []KnowledgeBaseCatalogFileListParams
	previewFn    func(context.Context, string) (*SemanticModelArtifactPreview, error)
	previewCalls []string
}

func (f *fakeSemanticModelFileService) ListFiles(_ context.Context, params KnowledgeBaseCatalogFileListParams) (*KnowledgeBaseCatalogFileListResult, error) {
	f.listCalls = append(f.listCalls, params)
	if f.listFiles != nil {
		return f.listFiles(params)
	}
	items := make([]KnowledgeBaseCatalogFileLeaf, 0, len(params.FileIDs))
	if len(params.FileIDs) == 0 {
		items = append(items, KnowledgeBaseCatalogFileLeaf{FileID: "catalog-file-1", FileName: "catalog-file-1.pdf", VolumeID: params.VolumeID})
	} else {
		for _, fileID := range params.FileIDs {
			items = append(items, KnowledgeBaseCatalogFileLeaf{FileID: fileID, FileName: fileID + ".pdf", VolumeID: params.VolumeID})
		}
	}
	return &KnowledgeBaseCatalogFileListResult{Items: items, Total: len(items)}, nil
}

func (f *fakeSemanticModelFileService) PreviewFile(ctx context.Context, fileID string) (*SemanticModelArtifactPreview, error) {
	f.previewCalls = append(f.previewCalls, fileID)
	if f.previewFn != nil {
		return f.previewFn(ctx, fileID)
	}
	return nil, errors.New("unexpected semantic model artifact preview")
}

func (f *fakeSemanticModelFileService) DeleteFileFromVolume(_ context.Context, volumeID int64, fileID string) error {
	if f.beforeDelete != nil {
		if err := f.beforeDelete(); err != nil {
			return err
		}
	}
	f.deleted = append(f.deleted, fmt.Sprintf("%d:%s", volumeID, fileID))
	return f.deleteErr
}

const semanticModelSegmentArtifactIdentityReadOnlyKey = "err.session.semantic_model.segment_artifact_identity_read_only"

const semanticModelWorkflowArtifactAssociationQuery = `(?s)SELECT COUNT\(\*\).*FROM semantic_models sm.*INNER JOIN data_asset vector_asset.*INNER JOIN data_derivation indexed_derivation.*LEFT JOIN parsed_manifest pm.*WHERE sm\.id = \?.*pm\.parsed_file_id = \?.*artifact_derivation.*artifact\.asset_ref = \?`
const semanticModelWorkflowVectorTablesQuery = `(?s)SELECT JSON_UNQUOTE\(JSON_EXTRACT\(files, '\$\.vector_table'\)\) AS table_name.*UNION.*JSON_UNQUOTE\(JSON_EXTRACT\(files, '\$\.image_vector_table'\)\) AS table_name`

func expectSemanticModelWorkflowArtifactAssociation(mock sqlmock.Sqlmock, modelID int64, fileID string) *sqlmock.ExpectedQuery {
	return mock.ExpectQuery(semanticModelWorkflowArtifactAssociationQuery).WithArgs(modelID, fileID, fileID)
}

func expectSemanticModelWorkflowVectorTables(mock sqlmock.Sqlmock, modelID int64) *sqlmock.ExpectedQuery {
	return mock.ExpectQuery(semanticModelWorkflowVectorTablesQuery).WithArgs(modelID, modelID)
}

func TestSemanticModelService_CreateSegmentRejectsCallerArtifactIDsBeforeDependencies(t *testing.T) {
	stringPointer := func(value string) *string { return &value }
	baseVersionID := "segment-v1"
	baseIndexVersion := int64(4)
	content := "caller-authored text"
	for _, tc := range []struct {
		name            string
		imageFileID     *string
		pageImageFileID *string
	}{
		{name: "image artifact", imageFileID: stringPointer("artifact-from-another-model")},
		{name: "page artifact", pageImageFileID: stringPointer("page-artifact-from-another-model")},
		{name: "empty image artifact", imageFileID: stringPointer("")},
		{name: "whitespace image artifact", imageFileID: stringPointer(" \t")},
		{name: "empty page artifact", pageImageFileID: stringPointer("")},
		{name: "whitespace page artifact", pageImageFileID: stringPointer("\n ")},
		{
			name:            "both artifacts",
			imageFileID:     stringPointer("artifact-from-another-model"),
			pageImageFileID: stringPointer("page-artifact-from-another-model"),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tenantDB, tenantMock := newSemanticModelTenantDB(t)
			fileSvc := &fakeSemanticModelFileService{}
			svc := &semanticModelService{fileService: fileSvc}
			// Artifact rejection is pure validation before mutateSegments /
			// coreclient.Execute; no HTTP stub is required.
			ctx := ctxutil.WithTenantDB(semanticModelServiceTestContext(i18n.LocaleEnUS.String()), tenantDB)

			result, err := svc.CreateSegment(ctx, CreateSemanticModelSegmentParams{
				ModelID:  42,
				SourceID: "source-file-1",
				SemanticModelSegmentMutationBase: SemanticModelSegmentMutationBase{
					BaseSegmentVersionID: &baseVersionID,
					BaseIndexVersion:     &baseIndexVersion,
				},
				Level:           kbSegmentLevelChunk,
				Content:         &content,
				ImageFileID:     tc.imageFileID,
				PageImageFileID: tc.pageImageFileID,
			})
			var serviceErr *ServiceError
			if result != nil || !errors.As(err, &serviceErr) || serviceErr.Code != ErrCodeBadRequest {
				t.Fatalf("CreateSegment() = (%v, %v), want dedicated bad request", result, err)
			}
			if serviceErr.Err == nil || serviceErr.Err.Error() != semanticModelSegmentArtifactIdentityReadOnlyKey {
				t.Fatalf("CreateSegment() underlying error = %v, want exact key %q", serviceErr.Err, semanticModelSegmentArtifactIdentityReadOnlyKey)
			}
			for _, locale := range []struct {
				tag  language.Tag
				want string
			}{
				{tag: i18n.LocaleEnUS, want: "Segment artifact identity is parser-owned and cannot be set manually"},
				{tag: i18n.LocaleZhCN, want: "分段解析产物标识由解析流程管理，不能手动设置"},
			} {
				message, ok := i18n.Message(i18n.WithLocale(context.Background(), locale.tag), err)
				if !ok || message != locale.want {
					t.Fatalf("CreateSegment() localized message for %s = (%q, %t), want %q", locale.tag, message, ok, locale.want)
				}
			}
			if err := tenantMock.ExpectationsWereMet(); err != nil {
				t.Fatalf("CreateSegment() touched tenant DB before artifact rejection: %v", err)
			}
			if len(fileSvc.listCalls) != 0 || len(fileSvc.previewCalls) != 0 || len(fileSvc.deleted) != 0 {
				t.Fatalf("CreateSegment() touched file service before artifact rejection: %+v", fileSvc)
			}
		})
	}
}

func TestSemanticModelService_CreateSegmentTextOnlyDoesNotUseArtifactReadOnlyError(t *testing.T) {
	tenantDB, tenantMock := newSemanticModelTenantDB(t)
	fileSvc := &fakeSemanticModelFileService{}
	svc := &semanticModelService{fileService: fileSvc}
	// Text-only create passes artifact validation and enters mutateSegments via
	// coreclient.Execute. A failing Catalog stub proves we reached the core path.
	var coreHits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		coreHits.Add(1)
		http.Error(w, "text-only create reached core client", http.StatusServiceUnavailable)
	}))
	t.Cleanup(server.Close)
	if err := coreclient.Configure(coreclient.Config{
		Endpoint:     server.URL,
		SystemAPIKey: "system-key",
		HTTPClient:   server.Client(),
		Timeout:      5 * time.Second,
	}); err != nil {
		t.Fatalf("configure coreclient: %v", err)
	}
	ctx := ctxutil.WithTenantDB(semanticModelServiceTestContext(i18n.LocaleEnUS.String()), tenantDB)
	baseVersionID := "segment-v1"
	baseIndexVersion := int64(4)
	content := "caller-authored text"

	result, err := svc.CreateSegment(ctx, CreateSemanticModelSegmentParams{
		ModelID:  42,
		SourceID: "source-file-1",
		SemanticModelSegmentMutationBase: SemanticModelSegmentMutationBase{
			BaseSegmentVersionID: &baseVersionID,
			BaseIndexVersion:     &baseIndexVersion,
		},
		Level:   kbSegmentLevelChunk,
		Content: &content,
	})
	if result != nil || err == nil {
		t.Fatalf("CreateSegment() = (%v, %v), want text-only path to continue to core client", result, err)
	}
	if strings.Contains(err.Error(), semanticModelSegmentArtifactIdentityReadOnlyKey) {
		t.Fatalf("CreateSegment() text-only error = %v, must not use artifact read-only error", err)
	}
	if coreHits.Load() == 0 {
		t.Fatalf("CreateSegment() made no core HTTP calls; text-only path must enter coreclient.Execute")
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("text-only CreateSegment() tenant expectations: %v", err)
	}
}

func TestSemanticModelService_PreviewArtifactRequiresSourceAssociation(t *testing.T) {
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	expectSemanticModelWorkflowArtifactAssociation(tenantMock, 42, "page-image-9").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	fileSvc := &fakeSemanticModelFileService{
		previewFn: func(_ context.Context, fileID string) (*SemanticModelArtifactPreview, error) {
			if fileID != "page-image-9" {
				t.Fatalf("preview file id = %q", fileID)
			}
			return &SemanticModelArtifactPreview{
				Filename:    "page-image-9.png",
				ContentType: "image/png",
				Body:        io.NopCloser(strings.NewReader("png-bytes")),
			}, nil
		},
	}
	svc := &semanticModelService{fileService: fileSvc}
	result, err := svc.PreviewArtifact(ctxutil.WithTenantDB(context.Background(), tenantDB), 42, "page-image-9")
	if err != nil {
		t.Fatalf("PreviewArtifact() error = %v", err)
	}
	defer result.Body.Close()
	body, err := io.ReadAll(result.Body)
	if err != nil {
		t.Fatalf("read preview body: %v", err)
	}
	if result.Filename != "page-image-9.png" || result.ContentType != "image/png" || string(body) != "png-bytes" {
		t.Fatalf("preview result = (%q, %q, %q)", result.Filename, result.ContentType, string(body))
	}
	if !reflect.DeepEqual(fileSvc.previewCalls, []string{"page-image-9"}) {
		t.Fatalf("preview calls = %v", fileSvc.previewCalls)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant expectations: %v", err)
	}
}

func TestSemanticModelService_PreviewArtifactRejectsUnassociatedFile(t *testing.T) {
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	expectSemanticModelWorkflowArtifactAssociation(tenantMock, 42, "other-workspace-file").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	expectSemanticModelWorkflowVectorTables(tenantMock, 42).
		WillReturnRows(sqlmock.NewRows([]string{"table_name"}))

	fileSvc := &fakeSemanticModelFileService{}
	svc := &semanticModelService{fileService: fileSvc}
	result, err := svc.PreviewArtifact(ctxutil.WithTenantDB(context.Background(), tenantDB), 42, "other-workspace-file")
	if result != nil || !IsServiceError(err, ErrCodeNotFound) {
		t.Fatalf("PreviewArtifact() = (%v, %v), want not found", result, err)
	}
	if len(fileSvc.previewCalls) != 0 {
		t.Fatalf("unassociated file reached preview service: %v", fileSvc.previewCalls)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant expectations: %v", err)
	}
}

func TestSemanticModelService_PreviewArtifactRejectsUnrelatedArtifactBeforeSystemRead(t *testing.T) {
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	expectSemanticModelWorkflowArtifactAssociation(tenantMock, 42, "artifact-from-another-model").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	expectSemanticModelWorkflowVectorTables(tenantMock, 42).
		WillReturnRows(sqlmock.NewRows([]string{"table_name"}))

	fileSvc := &fakeSemanticModelFileService{}
	svc := &semanticModelService{fileService: fileSvc}
	result, err := svc.PreviewArtifact(
		ctxutil.WithTenantDB(context.Background(), tenantDB),
		42,
		"artifact-from-another-model",
	)
	if result != nil || !IsServiceError(err, ErrCodeNotFound) {
		t.Fatalf("PreviewArtifact() = (%v, %v), want not found for caller-forged association", result, err)
	}
	if len(fileSvc.previewCalls) != 0 {
		t.Fatalf("caller-forged association reached system artifact client: %v", fileSvc.previewCalls)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant expectations: %v", err)
	}
}

func TestSemanticModelService_PreviewArtifactFailsClosedOnAssociationLookupError(t *testing.T) {
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	expectSemanticModelWorkflowArtifactAssociation(tenantMock, 42, "page-image-9").
		WillReturnError(errors.New("tenant database unavailable"))

	fileSvc := &fakeSemanticModelFileService{}
	svc := &semanticModelService{fileService: fileSvc}
	result, err := svc.PreviewArtifact(ctxutil.WithTenantDB(context.Background(), tenantDB), 42, "page-image-9")
	if result != nil || !IsServiceError(err, ErrCodeInternal) {
		t.Fatalf("PreviewArtifact() = (%v, %v), want internal error", result, err)
	}
	if len(fileSvc.previewCalls) != 0 {
		t.Fatalf("association lookup failure reached preview service: %v", fileSvc.previewCalls)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant expectations: %v", err)
	}
}

func TestSemanticModelService_PreviewArtifactWrapsFilePreviewFailure(t *testing.T) {
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	expectSemanticModelWorkflowArtifactAssociation(tenantMock, 42, "page-image-9").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	fileSvc := &fakeSemanticModelFileService{
		previewFn: func(context.Context, string) (*SemanticModelArtifactPreview, error) {
			return nil, errors.New("private storage details")
		},
	}
	svc := &semanticModelService{fileService: fileSvc}
	result, err := svc.PreviewArtifact(ctxutil.WithTenantDB(context.Background(), tenantDB), 42, "page-image-9")
	if result != nil || !IsServiceError(err, ErrCodeInternal) {
		t.Fatalf("PreviewArtifact() = (%v, %v), want internal error", result, err)
	}
	if err == nil || err.Error() != semanticModelArtifactUnavailableMessage {
		t.Fatalf("PreviewArtifact() public error = %v, want %q", err, semanticModelArtifactUnavailableMessage)
	}
	if !reflect.DeepEqual(fileSvc.previewCalls, []string{"page-image-9"}) {
		t.Fatalf("preview calls = %v", fileSvc.previewCalls)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant expectations: %v", err)
	}
}

func TestSemanticModelService_PreviewArtifactUsesWorkflowLineage(t *testing.T) {
	const imageID = "11111111-1111-4111-8111-111111111111"
	const layoutID = "22222222-2222-4222-8222-222222222222"
	const metadataImageID = "33333333-3333-4333-8333-333333333333"
	const otherModelLayoutID = "44444444-4444-4444-8444-444444444444"
	tests := []struct {
		name          string
		fileID        string
		wantPreviewID string
	}{
		{name: "parsed document", fileID: "parsed-file-42", wantPreviewID: "parsed-file-42"},
		{name: "derived document artifact", fileID: imageID, wantPreviewID: imageID},
		{name: "workflow output artifact", fileID: "workflow-output-42", wantPreviewID: "workflow-output-42"},
		{name: "layout artifact stored in vector metadata", fileID: layoutID, wantPreviewID: layoutID},
		{name: "image artifact stored in vector metadata", fileID: metadataImageID + ".PNG", wantPreviewID: metadataImageID},
		{name: "normalized image suffix", fileID: imageID + ".PNG", wantPreviewID: imageID},
		{name: "root source is not an artifact", fileID: "source-file-42"},
		{name: "artifact from another model", fileID: "derived-file-43"},
		{name: "metadata artifact from another model", fileID: otherModelLayoutID},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tenantDB := openWorkflowLineagePreviewSQLiteDB(t)
			insertWorkflowLineageFixture(t, tenantDB, 42, "vector-table-42", "source-file-42", "parsed-file-42", imageID, "workflow-output-42")
			insertWorkflowLineageFixture(t, tenantDB, 43, "vector-table-43", "source-file-43", "parsed-file-43", "derived-file-43", "workflow-output-43")
			insertWorkflowVectorArtifactMetadataFixture(t, tenantDB, "vector-table-42", "source-file-42", map[string]string{
				"layout_file_id": layoutID,
				"image_url":      metadataImageID + ".jpg",
			})
			insertWorkflowVectorArtifactMetadataFixture(t, tenantDB, "vector-table-43", "source-file-43", map[string]string{
				"layout_file_id": otherModelLayoutID,
			})

			fileSvc := &fakeSemanticModelFileService{
				previewFn: func(_ context.Context, fileID string) (*SemanticModelArtifactPreview, error) {
					return &SemanticModelArtifactPreview{
						Filename:    fileID + ".png",
						ContentType: "image/png",
						Body:        io.NopCloser(strings.NewReader("png-bytes")),
					}, nil
				},
			}
			svc := &semanticModelService{fileService: fileSvc}
			result, err := svc.PreviewArtifact(ctxutil.WithTenantDB(context.Background(), tenantDB), 42, tc.fileID)

			if tc.wantPreviewID == "" {
				if result != nil || !IsServiceError(err, ErrCodeNotFound) {
					t.Fatalf("PreviewArtifact() = (%v, %v), want not found", result, err)
				}
				if len(fileSvc.previewCalls) != 0 {
					t.Fatalf("unassociated artifact reached system preview: %v", fileSvc.previewCalls)
				}
				return
			}
			if err != nil || result == nil || result.Body == nil {
				t.Fatalf("PreviewArtifact() = (%v, %v), want workflow-associated preview", result, err)
			}
			_ = result.Body.Close()
			if !reflect.DeepEqual(fileSvc.previewCalls, []string{tc.wantPreviewID}) {
				t.Fatalf("preview calls = %v, want [%s]", fileSvc.previewCalls, tc.wantPreviewID)
			}
		})
	}
}

const workflowLineagePreviewSQLiteDriver = "sqlite3_workflow_lineage_preview"

var registerWorkflowLineagePreviewSQLiteDriver sync.Once

func openWorkflowLineagePreviewSQLiteDB(t *testing.T) *gorm.DB {
	t.Helper()
	registerWorkflowLineagePreviewSQLiteDriver.Do(func() {
		sql.Register(workflowLineagePreviewSQLiteDriver, &sqlite3.SQLiteDriver{ConnectHook: func(conn *sqlite3.SQLiteConn) error {
			if err := conn.RegisterFunc("JSON_UNQUOTE", func(value any) any { return value }, true); err != nil {
				return err
			}
			return conn.RegisterFunc("REGEXP", func(pattern, value any) (bool, error) {
				if pattern == nil || value == nil {
					return false, nil
				}
				return regexp.MatchString(fmt.Sprint(pattern), fmt.Sprint(value))
			}, true)
		}})
	})
	tenantDB, err := gorm.Open(gsqlite.New(gsqlite.Config{DriverName: workflowLineagePreviewSQLiteDriver, DSN: ":memory:"}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open workflow lineage preview SQLite DB: %v", err)
	}
	sqlDB, err := tenantDB.DB()
	if err != nil {
		t.Fatalf("open workflow lineage preview SQL DB: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	for _, statement := range []string{
		`CREATE TABLE semantic_models (id INTEGER PRIMARY KEY, files TEXT NOT NULL)`,
		`CREATE TABLE data_asset (asset_id TEXT PRIMARY KEY, asset_type TEXT NOT NULL, asset_ref TEXT NOT NULL)`,
		`CREATE TABLE data_derivation (root_asset_id TEXT NOT NULL, target_asset_id TEXT NOT NULL, kind TEXT NOT NULL)`,
		`CREATE TABLE parsed_manifest (root_asset_id TEXT NOT NULL, source_file_id TEXT NOT NULL, parsed_file_id TEXT NOT NULL)`,
	} {
		if err := tenantDB.Exec(statement).Error; err != nil {
			t.Fatalf("create workflow lineage preview fixture table: %v", err)
		}
	}
	return tenantDB
}

func insertWorkflowLineageFixture(t *testing.T, db *gorm.DB, modelID int64, vectorTable, sourceFileID, parsedFileID, derivedFileID, outputFileID string) {
	t.Helper()
	quotedTable, err := quoteQualifiedSQLIdentifier(vectorTable)
	if err != nil {
		t.Fatalf("quote vector table %q: %v", vectorTable, err)
	}
	if err := db.Exec(fmt.Sprintf(`CREATE TABLE %s (file_id TEXT, meta TEXT)`, quotedTable)).Error; err != nil {
		t.Fatalf("create workflow vector fixture table: %v", err)
	}
	rootAssetID := fmt.Sprintf("root-asset-%d", modelID)
	vectorAssetID := fmt.Sprintf("vector-asset-%d", modelID)
	parsedAssetID := fmt.Sprintf("parsed-asset-%d", modelID)
	derivedAssetID := fmt.Sprintf("derived-asset-%d", modelID)
	outputAssetID := fmt.Sprintf("output-asset-%d", modelID)

	for _, statement := range []struct {
		query string
		args  []any
	}{
		{query: `INSERT INTO semantic_models (id, files) VALUES (?, ?)`, args: []any{modelID, fmt.Sprintf(`{"vector_table":%q}`, vectorTable)}},
		{query: `INSERT INTO data_asset (asset_id, asset_type, asset_ref) VALUES (?, 'file', ?)`, args: []any{rootAssetID, sourceFileID}},
		{query: `INSERT INTO data_asset (asset_id, asset_type, asset_ref) VALUES (?, 'vector_index', ?)`, args: []any{vectorAssetID, vectorTable}},
		{query: `INSERT INTO data_asset (asset_id, asset_type, asset_ref) VALUES (?, 'file', ?)`, args: []any{parsedAssetID, parsedFileID}},
		{query: `INSERT INTO data_asset (asset_id, asset_type, asset_ref) VALUES (?, 'file', ?)`, args: []any{derivedAssetID, derivedFileID}},
		{query: `INSERT INTO data_asset (asset_id, asset_type, asset_ref) VALUES (?, 'file', ?)`, args: []any{outputAssetID, outputFileID}},
		{query: `INSERT INTO data_derivation (root_asset_id, target_asset_id, kind) VALUES (?, ?, 'indexed_from')`, args: []any{rootAssetID, vectorAssetID}},
		{query: `INSERT INTO data_derivation (root_asset_id, target_asset_id, kind) VALUES (?, ?, 'derived_file_from')`, args: []any{rootAssetID, derivedAssetID}},
		{query: `INSERT INTO data_derivation (root_asset_id, target_asset_id, kind) VALUES (?, ?, 'transformed_from')`, args: []any{rootAssetID, outputAssetID}},
		{query: `INSERT INTO parsed_manifest (root_asset_id, source_file_id, parsed_file_id) VALUES (?, ?, ?)`, args: []any{rootAssetID, sourceFileID, parsedFileID}},
	} {
		if err := db.Exec(statement.query, statement.args...).Error; err != nil {
			t.Fatalf("insert workflow lineage fixture: %v", err)
		}
	}
}

func insertWorkflowVectorArtifactMetadataFixture(t *testing.T, db *gorm.DB, vectorTable, sourceFileID string, metadata map[string]string) {
	t.Helper()
	quotedTable, err := quoteQualifiedSQLIdentifier(vectorTable)
	if err != nil {
		t.Fatalf("quote vector table %q: %v", vectorTable, err)
	}
	encodedMetadata, err := json.Marshal(metadata)
	if err != nil {
		t.Fatalf("marshal workflow vector artifact metadata: %v", err)
	}
	if err := db.Exec(fmt.Sprintf(`INSERT INTO %s (file_id, meta) VALUES (?, ?)`, quotedTable), sourceFileID, string(encodedMetadata)).Error; err != nil {
		t.Fatalf("insert workflow vector artifact metadata: %v", err)
	}
}

func TestSemanticModelArtifactImageIDMatchesRAGImageReferences(t *testing.T) {
	const artifactID = "11111111-1111-4111-8111-111111111111"
	for _, ref := range []string{artifactID, artifactID + ".jpg", artifactID + ".PNG", artifactID + ".tiff"} {
		if got := semanticModelArtifactImageID(ref); got != artifactID {
			t.Fatalf("semanticModelArtifactImageID(%q) = %q, want %q", ref, got, artifactID)
		}
	}
	for _, ref := range []string{artifactID + ".svg", "page-image-a3", "prefix-" + artifactID} {
		if got := semanticModelArtifactImageID(ref); got != "" {
			t.Fatalf("semanticModelArtifactImageID(%q) = %q, want empty", ref, got)
		}
	}
}

func TestSemanticModelService_PreviewArtifactFailsClosedWithoutTenantDB(t *testing.T) {
	fileSvc := &fakeSemanticModelFileService{}
	svc := &semanticModelService{fileService: fileSvc}
	result, err := svc.PreviewArtifact(context.Background(), 42, "page-image-9")
	if result != nil || !IsServiceError(err, ErrCodeInternal) {
		t.Fatalf("PreviewArtifact() = (%v, %v), want internal error", result, err)
	}
	if len(fileSvc.previewCalls) != 0 {
		t.Fatalf("missing tenant DB reached preview service: %v", fileSvc.previewCalls)
	}
}

func TestSemanticModelService_PreviewArtifactRejectsInvalidIdentity(t *testing.T) {
	fileSvc := &fakeSemanticModelFileService{}
	svc := &semanticModelService{fileService: fileSvc}
	for _, tc := range []struct {
		name    string
		modelID int
		fileID  string
	}{
		{name: "zero model", modelID: 0, fileID: "page-image-9"},
		{name: "blank file", modelID: 42, fileID: " "},
		{name: "non canonical file", modelID: 42, fileID: " page-image-9 "},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result, err := svc.PreviewArtifact(context.Background(), tc.modelID, tc.fileID)
			if result != nil || !IsServiceError(err, ErrCodeBadRequest) {
				t.Fatalf("PreviewArtifact() = (%v, %v), want bad request", result, err)
			}
		})
	}
	if len(fileSvc.previewCalls) != 0 {
		t.Fatalf("invalid identity reached preview service: %v", fileSvc.previewCalls)
	}
}

func TestSemanticModelService_PreviewSourceFileUsesWorkflowLineage(t *testing.T) {
	tests := []struct {
		name        string
		fileID      string
		wantPreview bool
	}{
		{name: "workflow source document", fileID: "source-file-42", wantPreview: true},
		{name: "parsed document is not a source preview", fileID: "parsed-file-42"},
		{name: "source from another model", fileID: "source-file-43"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tenantDB := openWorkflowLineagePreviewSQLiteDB(t)
			insertWorkflowLineageFixture(t, tenantDB, 42, "vector-table-42", "source-file-42", "parsed-file-42", "derived-file-42", "workflow-output-42")
			insertWorkflowLineageFixture(t, tenantDB, 43, "vector-table-43", "source-file-43", "parsed-file-43", "derived-file-43", "workflow-output-43")

			fileSvc := &fakeSemanticModelFileService{
				previewFn: func(_ context.Context, fileID string) (*SemanticModelArtifactPreview, error) {
					return &SemanticModelArtifactPreview{
						Filename:    fileID + ".pdf",
						ContentType: "application/pdf",
						Body:        io.NopCloser(strings.NewReader("pdf-bytes")),
					}, nil
				},
			}
			svc := &semanticModelService{fileService: fileSvc}
			result, err := svc.PreviewSourceFile(ctxutil.WithTenantDB(context.Background(), tenantDB), 42, tc.fileID)

			if !tc.wantPreview {
				if result != nil || !IsServiceError(err, ErrCodeNotFound) {
					t.Fatalf("PreviewSourceFile() = (%v, %v), want not found", result, err)
				}
				if len(fileSvc.previewCalls) != 0 {
					t.Fatalf("unassociated source reached system preview: %v", fileSvc.previewCalls)
				}
				return
			}
			if err != nil || result == nil || result.Body == nil {
				t.Fatalf("PreviewSourceFile() = (%v, %v), want workflow-associated preview", result, err)
			}
			_ = result.Body.Close()
			if !reflect.DeepEqual(fileSvc.previewCalls, []string{tc.fileID}) {
				t.Fatalf("source preview calls = %v, want [%s]", fileSvc.previewCalls, tc.fileID)
			}
		})
	}
}

func TestSemanticModelService_PreviewSourceFileUsesModelVectorRowsWithoutCatalogLineage(t *testing.T) {
	tenantDB := openWorkflowLineagePreviewSQLiteDB(t)
	for _, fixture := range []struct {
		modelID     int64
		vectorTable string
		fileID      string
	}{
		{modelID: 42, vectorTable: "vector-table-42", fileID: "shared-source-file"},
		{modelID: 43, vectorTable: "vector-table-43", fileID: "shared-source-file"},
		{modelID: 44, vectorTable: "vector-table-44", fileID: "another-source-file"},
	} {
		quotedTable, err := quoteQualifiedSQLIdentifier(fixture.vectorTable)
		if err != nil {
			t.Fatalf("quote vector table %q: %v", fixture.vectorTable, err)
		}
		if err := tenantDB.Exec(fmt.Sprintf(`CREATE TABLE %s (file_id TEXT, meta TEXT)`, quotedTable)).Error; err != nil {
			t.Fatalf("create workflow vector fixture table: %v", err)
		}
		if err := tenantDB.Exec(`INSERT INTO semantic_models (id, files) VALUES (?, ?)`, fixture.modelID, fmt.Sprintf(`{"vector_table":%q}`, fixture.vectorTable)).Error; err != nil {
			t.Fatalf("insert workflow vector fixture model: %v", err)
		}
		if err := tenantDB.Exec(fmt.Sprintf(`INSERT INTO %s (file_id, meta) VALUES (?, '{}')`, quotedTable), fixture.fileID).Error; err != nil {
			t.Fatalf("insert workflow vector fixture row: %v", err)
		}
	}

	for _, tc := range []struct {
		name        string
		modelID     int
		fileID      string
		wantPreview bool
	}{
		{name: "same original document reused by another knowledge base", modelID: 42, fileID: "shared-source-file", wantPreview: true},
		{name: "reused original document is available from each knowledge base", modelID: 43, fileID: "shared-source-file", wantPreview: true},
		{name: "source only indexed by another knowledge base", modelID: 42, fileID: "another-source-file"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fileSvc := &fakeSemanticModelFileService{
				previewFn: func(_ context.Context, fileID string) (*SemanticModelArtifactPreview, error) {
					return &SemanticModelArtifactPreview{
						Filename:    fileID + ".pdf",
						ContentType: "application/pdf",
						Body:        io.NopCloser(strings.NewReader("pdf-bytes")),
					}, nil
				},
			}
			svc := &semanticModelService{fileService: fileSvc}
			result, err := svc.PreviewSourceFile(ctxutil.WithTenantDB(context.Background(), tenantDB), tc.modelID, tc.fileID)

			if !tc.wantPreview {
				if result != nil || !IsServiceError(err, ErrCodeNotFound) {
					t.Fatalf("PreviewSourceFile() = (%v, %v), want not found", result, err)
				}
				if len(fileSvc.previewCalls) != 0 {
					t.Fatalf("unassociated source reached system preview: %v", fileSvc.previewCalls)
				}
				return
			}
			if err != nil || result == nil || result.Body == nil {
				t.Fatalf("PreviewSourceFile() = (%v, %v), want vector-associated preview", result, err)
			}
			_ = result.Body.Close()
			if !reflect.DeepEqual(fileSvc.previewCalls, []string{tc.fileID}) {
				t.Fatalf("source preview calls = %v, want [%s]", fileSvc.previewCalls, tc.fileID)
			}
		})
	}
}

func TestSemanticModelService_PreviewSourceFileRejectsInvalidIdentity(t *testing.T) {
	fileSvc := &fakeSemanticModelFileService{}
	svc := &semanticModelService{fileService: fileSvc}
	for _, tc := range []struct {
		name    string
		modelID int
		fileID  string
	}{
		{name: "zero model", fileID: "source-file"},
		{name: "blank file", modelID: 42, fileID: " "},
		{name: "non canonical file", modelID: 42, fileID: " source-file "},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result, err := svc.PreviewSourceFile(context.Background(), tc.modelID, tc.fileID)
			if result != nil || !IsServiceError(err, ErrCodeBadRequest) {
				t.Fatalf("PreviewSourceFile() = (%v, %v), want bad request", result, err)
			}
		})
	}
	if len(fileSvc.previewCalls) != 0 {
		t.Fatalf("invalid source identity reached file preview: %v", fileSvc.previewCalls)
	}
}

type fakeSemanticModelLocalFileImportService struct {
	fileName     string
	body         string
	volumeID     int64
	uploadKind   string
	tableConfig  string
	calls        []KnowledgeBaseLocalFileImportParams
	taskID       string
	taskIDs      []string
	fileIDs      []string
	beforeUpload func() error
}

func (f *fakeSemanticModelLocalFileImportService) UploadToVolume(_ context.Context, params KnowledgeBaseLocalFileImportParams) (*KnowledgeBaseLocalFileImportResult, error) {
	if f.beforeUpload != nil {
		if err := f.beforeUpload(); err != nil {
			return nil, err
		}
	}
	body := []byte(nil)
	if params.Reader != nil {
		var err error
		body, err = io.ReadAll(params.Reader)
		if err != nil {
			return nil, err
		}
	}
	f.fileName = params.FileName
	f.body = string(body)
	f.volumeID = params.VolumeID
	f.uploadKind = params.UploadKind
	f.tableConfig = params.TableConfig
	f.calls = append(f.calls, KnowledgeBaseLocalFileImportParams{
		VolumeID:    params.VolumeID,
		FileName:    params.FileName,
		FileID:      params.FileID,
		UploadKind:  params.UploadKind,
		TableConfig: params.TableConfig,
	})
	taskID := f.taskID
	if len(f.taskIDs) > 0 {
		taskID = f.taskIDs[0]
		f.taskIDs = f.taskIDs[1:]
	}
	if taskID == "" {
		taskID = fmt.Sprintf("import-task-%d", len(f.calls))
	}
	fileIDs := append([]string{}, f.fileIDs...)
	if len(f.fileIDs) > 0 {
		f.fileIDs = nil
	}
	if len(fileIDs) == 0 {
		fileIDs = []string{fmt.Sprintf("kb-local-file-%d", len(f.calls))}
	}
	return &KnowledgeBaseLocalFileImportResult{TaskID: taskID, FileIDs: fileIDs}, nil
}

func expectKnowledgeBaseCatalogResourceLookup(mock sqlmock.Sqlmock, modelID, catalogID, databaseID, rawVolumeID, processedVolumeID int64, rawVolumeIDs ...int64) {
	mock.ExpectQuery("SELECT model_id, catalog_id, database_id, raw_volume_id, processed_volume_id, ensure_status, last_ensure_error, last_checked_at").
		WithArgs(modelID).
		WillReturnRows(sqlmock.NewRows([]string{
			"model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "ensure_status", "last_ensure_error", "last_checked_at",
		}).AddRow(modelID, catalogID, databaseID, rawVolumeID, processedVolumeID, kbEnsureStatusReady, nil, int64(100)))
	rawRows := sqlmock.NewRows([]string{"raw_volume_id"})
	for _, volumeID := range rawVolumeIDs {
		rawRows.AddRow(volumeID)
	}
	mock.ExpectQuery("SELECT raw_volume_id\\s+FROM knowledge_base_raw_volumes").
		WithArgs(modelID).
		WillReturnRows(rawRows)
	mock.ExpectQuery("SELECT raw_volume_id\\s+FROM knowledge_base_data_domains").
		WithArgs(modelID).
		WillReturnRows(sqlmock.NewRows([]string{"raw_volume_id"}).AddRow(rawVolumeID))
}

type queuedSemanticModelLocalFileImportService struct {
	calls        []KnowledgeBaseLocalFileImportParams
	results      []KnowledgeBaseLocalFileImportResult
	beforeUpload func() error
}

func (f *queuedSemanticModelLocalFileImportService) UploadToVolume(_ context.Context, params KnowledgeBaseLocalFileImportParams) (*KnowledgeBaseLocalFileImportResult, error) {
	if f.beforeUpload != nil {
		if err := f.beforeUpload(); err != nil {
			return nil, err
		}
	}
	if params.Reader != nil {
		if _, err := io.ReadAll(params.Reader); err != nil {
			return nil, err
		}
	}
	f.calls = append(f.calls, KnowledgeBaseLocalFileImportParams{
		VolumeID:    params.VolumeID,
		FileName:    params.FileName,
		FileID:      params.FileID,
		UploadKind:  params.UploadKind,
		TableConfig: params.TableConfig,
	})
	if len(f.results) == 0 {
		return nil, errors.New("missing queued import result")
	}
	result := f.results[0]
	f.results = f.results[1:]
	return &result, nil
}

type exactShortOperationIDArg struct {
	value string
}

func (a exactShortOperationIDArg) Match(v driver.Value) bool {
	var got string
	switch value := v.(type) {
	case string:
		got = value
	case []byte:
		got = string(value)
	default:
		return false
	}
	return got == a.value &&
		len(got) <= 128 &&
		!strings.Contains(got, "table_config") &&
		!strings.Contains(got, "conn_file_ids")
}

type stringContainsArg struct {
	parts []string
}

func (a stringContainsArg) Match(v driver.Value) bool {
	var got string
	switch value := v.(type) {
	case string:
		got = value
	case []byte:
		got = string(value)
	default:
		return false
	}
	for _, part := range a.parts {
		if !strings.Contains(got, part) {
			return false
		}
	}
	return true
}

func expectUpsertKnowledgeBaseRawVolume(mock sqlmock.Sqlmock, modelID int64, rawKind string, exists bool) {
	mock.ExpectQuery("SELECT count\\(\\*\\) FROM `knowledge_base_raw_volumes` WHERE model_id = \\? AND raw_kind = \\?").
		WithArgs(modelID, rawKind).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(map[bool]int{false: 0, true: 1}[exists]))
	if exists {
		mock.ExpectExec("UPDATE knowledge_base_raw_volumes").
			WillReturnResult(sqlmock.NewResult(1, 1))
		return
	}
	mock.ExpectExec("INSERT INTO knowledge_base_raw_volumes").
		WillReturnResult(sqlmock.NewResult(1, 1))
}

// expectClaimKnowledgeBaseDataDomainProvision matches failed → provisioning CAS.
func expectClaimKnowledgeBaseDataDomainProvision(mock sqlmock.Sqlmock, modelID int64) {
	mock.ExpectExec("UPDATE knowledge_base_data_domains\\s+SET ensure_status = \\?, last_ensure_error = NULL, last_checked_at = \\?, updated_by = \\?\\s+WHERE model_id = \\? AND ensure_status = \\?").
		WithArgs(kbEnsureStatusProvisioning, sqlmock.AnyArg(), sqlmock.AnyArg(), modelID, kbEnsureStatusFailed).
		WillReturnResult(sqlmock.NewResult(0, 1))
}

// expectUpdateKnowledgeBaseDataDomainReadyCAS matches provisioning → ready with resource IDs.
func expectUpdateKnowledgeBaseDataDomainReadyCAS(mock sqlmock.Sqlmock, catalogID, databaseID, rawVolumeID, processedVolumeID, modelID int64) {
	mock.ExpectExec("UPDATE knowledge_base_data_domains\\s+SET catalog_id = \\?, database_id = \\?, raw_volume_id = \\?, processed_volume_id = \\?, ensure_status = \\?, last_ensure_error = \\?, last_checked_at = \\?, updated_by = \\?\\s+WHERE model_id = \\? AND ensure_status = \\?").
		WithArgs(catalogID, databaseID, rawVolumeID, processedVolumeID, kbEnsureStatusReady, nil, sqlmock.AnyArg(), sqlmock.AnyArg(), modelID, kbEnsureStatusProvisioning).
		WillReturnResult(sqlmock.NewResult(0, 1))
}

// expectKnowledgeBaseDataDomainClaimAndReady is the common claim + ready pair for append/repair.
func expectKnowledgeBaseDataDomainClaimAndReady(mock sqlmock.Sqlmock, modelID, catalogID, databaseID, rawVolumeID, processedVolumeID int64) {
	expectClaimKnowledgeBaseDataDomainProvision(mock, modelID)
	expectUpdateKnowledgeBaseDataDomainReadyCAS(mock, catalogID, databaseID, rawVolumeID, processedVolumeID, modelID)
}

func expectKnowledgeBaseCreateDomainPrepare(mock sqlmock.Sqlmock, modelID, catalogID, databaseID, rawVolumeID, processedVolumeID int64) {
	mock.ExpectExec("UPDATE knowledge_base_data_domains\\s+SET catalog_id = \\?, database_id = \\?, raw_volume_id = \\?, processed_volume_id = \\?, ensure_status = \\?, last_ensure_error = \\?, last_checked_at = \\?, updated_by = \\?\\s+WHERE model_id = \\? AND ensure_status = \\?").
		WithArgs(catalogID, databaseID, rawVolumeID, processedVolumeID, kbEnsureStatusProvisioning, nil, sqlmock.AnyArg(), sqlmock.AnyArg(), modelID, kbEnsureStatusProvisioning).
		WillReturnResult(sqlmock.NewResult(0, 1))
}

func expectKnowledgeBaseCreateDomainFinalize(mock sqlmock.Sqlmock, modelID, catalogID, databaseID, rawVolumeID, processedVolumeID int64) {
	expectUpsertKnowledgeBaseRawVolume(mock, modelID, kbRawKindDocument, false)
	expectUpdateKnowledgeBaseDataDomainReadyCAS(mock, catalogID, databaseID, rawVolumeID, processedVolumeID, modelID)
}

// expectRollbackPersistPartialIDsCASHit re-persists in-memory IDs while retaining
// provisioning ownership under the detached cleanup context.
func expectRollbackPersistPartialIDsCASHit(mock sqlmock.Sqlmock, modelID, catalogID, databaseID, rawVolumeID, processedVolumeID int64) {
	mock.ExpectExec("UPDATE knowledge_base_data_domains\\s+SET catalog_id = \\?, database_id = \\?, raw_volume_id = \\?, processed_volume_id = \\?, ensure_status = \\?, last_ensure_error = \\?, last_checked_at = \\?, updated_by = \\?\\s+WHERE model_id = \\? AND ensure_status = \\?").
		WithArgs(catalogID, databaseID, rawVolumeID, processedVolumeID, kbEnsureStatusProvisioning, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), modelID, kbEnsureStatusProvisioning).
		WillReturnResult(sqlmock.NewResult(0, 1))
}

// expectRollbackFailedKnowledgeBaseCreate matches tenant SQL for create-path cleanup
// via deleteModel after provision or post-shell steps fail (workflow/catalog deletes
// are asserted via fake service calls).
func expectRollbackFailedKnowledgeBaseCreate(mock sqlmock.Sqlmock, modelID, catalogID, databaseID, rawVolumeID, processedVolumeID int64) {
	expectRollbackPersistPartialIDsCASHit(mock, modelID, catalogID, databaseID, rawVolumeID, processedVolumeID)
	// deleteModel → deleteKnowledgeBaseCatalogResources (owner lookup).
	mock.ExpectQuery("SELECT model_id, catalog_id, database_id, raw_volume_id, processed_volume_id, ensure_status, last_ensure_error, last_checked_at").
		WithArgs(modelID).
		WillReturnRows(sqlmock.NewRows([]string{
			"model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "ensure_status", "last_ensure_error", "last_checked_at",
		}).AddRow(modelID, catalogID, databaseID, rawVolumeID, processedVolumeID, kbEnsureStatusProvisioning, "create failed", int64(100)))
	mock.ExpectQuery("SELECT raw_volume_id\\s+FROM knowledge_base_raw_volumes\\s+WHERE model_id = \\? AND raw_volume_id > 0").
		WithArgs(modelID).
		WillReturnRows(sqlmock.NewRows([]string{"raw_volume_id"}))
	rawDomainRows := sqlmock.NewRows([]string{"raw_volume_id"})
	if rawVolumeID > 0 {
		rawDomainRows.AddRow(rawVolumeID)
	}
	mock.ExpectQuery("SELECT raw_volume_id\\s+FROM knowledge_base_data_domains\\s+WHERE model_id = \\? AND raw_volume_id > 0").
		WithArgs(modelID).
		WillReturnRows(rawDomainRows)
	// deleteModel only deletes tenant rows after catalog cleanup succeeds.
	for _, stmt := range []string{
		"DELETE FROM knowledge_base_chunk_recall_stats WHERE model_id = \\?",
		"DELETE FROM knowledge_base_segments WHERE model_id = \\?",
		"DELETE FROM knowledge_base_segment_versions WHERE model_id = \\?",
		"DELETE FROM knowledge_base_source_job_runs WHERE model_id = \\?",
		"DELETE FROM knowledge_base_sources WHERE model_id = \\?",
		"DELETE FROM knowledge_base_raw_volumes WHERE model_id = \\?",
		"DELETE FROM knowledge_base_source_jobs WHERE model_id = \\?",
		"DELETE FROM knowledge_base_data_domains WHERE model_id = \\?",
	} {
		mock.ExpectExec(stmt).WithArgs(modelID).WillReturnResult(sqlmock.NewResult(0, 0))
	}
}

// expectRollbackFailedKnowledgeBaseCreateCatalogOnly is deleteModel stopping after
// catalog delete fails: owner metadata (tenant rows + SM) must remain.
func expectRollbackFailedKnowledgeBaseCreateCatalogOnly(mock sqlmock.Sqlmock, modelID, catalogID, databaseID, rawVolumeID, processedVolumeID int64) {
	expectRollbackPersistPartialIDsCASHit(mock, modelID, catalogID, databaseID, rawVolumeID, processedVolumeID)
	mock.ExpectQuery("SELECT model_id, catalog_id, database_id, raw_volume_id, processed_volume_id, ensure_status, last_ensure_error, last_checked_at").
		WithArgs(modelID).
		WillReturnRows(sqlmock.NewRows([]string{
			"model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "ensure_status", "last_ensure_error", "last_checked_at",
		}).AddRow(modelID, catalogID, databaseID, rawVolumeID, processedVolumeID, kbEnsureStatusProvisioning, "create failed", int64(100)))
	mock.ExpectQuery("SELECT raw_volume_id\\s+FROM knowledge_base_raw_volumes\\s+WHERE model_id = \\? AND raw_volume_id > 0").
		WithArgs(modelID).
		WillReturnRows(sqlmock.NewRows([]string{"raw_volume_id"}))
	rawDomainRows := sqlmock.NewRows([]string{"raw_volume_id"})
	if rawVolumeID > 0 {
		rawDomainRows.AddRow(rawVolumeID)
	}
	mock.ExpectQuery("SELECT raw_volume_id\\s+FROM knowledge_base_data_domains\\s+WHERE model_id = \\? AND raw_volume_id > 0").
		WithArgs(modelID).
		WillReturnRows(rawDomainRows)
}

func TestEnsureKnowledgeBaseRawVolumeRecoversExistingVolumeMapping(t *testing.T) {
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}

	rawKind := kbRawKindStructured
	rawName := rawVolumeName(rawKind)
	tenantMock.ExpectQuery("SELECT raw_volume_id\\s+FROM knowledge_base_raw_volumes").
		WithArgs(int64(77), rawKind).
		WillReturnRows(sqlmock.NewRows([]string{"raw_volume_id"}))
	expectUpsertKnowledgeBaseRawVolume(tenantMock, 77, rawKind, false)

	svc := &semanticModelService{dataDomainService: &fakeSemanticModelDataDomainService{
		databaseID: 11,
		volumeErrs: map[string]error{
			rawName: &moi.Error{Code: common.ErrorCode_ALREADY_EXISTS, Message: "Volume name already exists"},
		},
		volumeByName: map[string]int64{rawName: 42},
	}}
	ctx := ctxutil.WithTenantDB(context.Background(), tenantDB)
	domain := &KnowledgeBaseDataDomain{
		ModelID:           77,
		CatalogID:         3,
		DatabaseID:        11,
		RawVolumeID:       12,
		ProcessedVolumeID: 13,
	}

	got, err := svc.ensureKnowledgeBaseRawVolume(ctx, domain, rawKind, "user-1")
	if err != nil {
		t.Fatalf("ensureKnowledgeBaseRawVolume: %v", err)
	}
	if got != 42 {
		t.Fatalf("raw volume id = %d, want 42", got)
	}
	if !sameStringSlice(svc.dataDomainService.(*fakeSemanticModelDataDomainService).calls, []string{
		rawName + ":" + rawVolumeDescription(rawKind),
		"resolve-volume:" + rawName,
	}) {
		t.Fatalf("data domain calls = %+v", svc.dataDomainService.(*fakeSemanticModelDataDomainService).calls)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func expectNoKnowledgeBaseRawVolumeFiles(mock sqlmock.Sqlmock, modelID int64) {
	mock.ExpectQuery("SELECT raw_volume_id\\s+FROM knowledge_base_raw_volumes").
		WithArgs(modelID).
		WillReturnRows(sqlmock.NewRows([]string{"raw_volume_id"}))
	mock.ExpectQuery("SELECT raw_volume_id\\s+FROM knowledge_base_data_domains").
		WithArgs(modelID).
		WillReturnRows(sqlmock.NewRows([]string{"raw_volume_id"}))
}

func expectKnowledgeBaseRawVolumeFileMetadata(mock sqlmock.Sqlmock, modelID int64, volumeID int64, rows *sqlmock.Rows) {
	mock.ExpectQuery("SELECT raw_volume_id\\s+FROM knowledge_base_raw_volumes").
		WithArgs(modelID).
		WillReturnRows(sqlmock.NewRows([]string{"raw_volume_id"}).AddRow(volumeID))
	mock.ExpectQuery("SELECT raw_volume_id\\s+FROM knowledge_base_data_domains").
		WithArgs(modelID).
		WillReturnRows(sqlmock.NewRows([]string{"raw_volume_id"}))
	mock.ExpectQuery("(?s)SELECT vf\\.file_id, COALESCE\\(CASE WHEN v\\.catalog_id.*WHERE vf\\.volume_id = \\?.*COALESCE\\(CASE WHEN v\\.catalog_id > 0 THEN v\\.catalog_id ELSE cd\\.catalog_id END, 0\\) > 0.*COALESCE\\(v\\.database_id, 0\\) > 0.*COALESCE\\(v\\.volume_name, ''\\) <> ''.*COALESCE\\(c\\.catalog_name, ''\\) <> ''.*COALESCE\\(cd\\.database_name, ''\\) <> ''").
		WithArgs(volumeID, kbLegacyBackfillBatchSize, 0).
		WillReturnRows(rows)
}

func rawVolumeFileMetadataRows(start, end int) *sqlmock.Rows {
	rows := sqlmock.NewRows([]string{
		"file_id", "catalog_id", "database_id", "volume_id", "size", "updated_at", "catalog_name", "database_name", "volume_name", "file_name",
	})
	for i := start; i <= end; i++ {
		rows.AddRow(
			fmt.Sprintf("raw-file-%03d", i),
			int64(3),
			int64(11),
			int64(7001),
			int64(1024+i),
			int64(1782705000+i),
			"catalog",
			"database",
			"raw-volume",
			fmt.Sprintf("doc-%03d.pdf", i),
		)
	}
	return rows
}

func newSemanticModelTenantDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()

	sqlDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("new tenant sqlmock: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	db, err := gorm.Open(gmysql.New(gmysql.Config{Conn: sqlDB, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	return db, mock
}

func semanticModelServiceTestContext(locale string) context.Context {
	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	if locale == i18n.LocaleZhCN.String() {
		return i18n.WithLocale(ctx, i18n.LocaleZhCN)
	}
	return i18n.WithLocale(ctx, i18n.LocaleEnUS)
}

// withKnowledgeBaseCreatePrincipal injects the create-time MOI user and
// VerifiedEffectiveRole that freezeKnowledgeBaseSourceJobRuntimePrincipal requires
// for RAG job writes. Tests that intentionally omit identity must not call this.
func withKnowledgeBaseCreatePrincipal(ctx context.Context) context.Context {
	if strings.TrimSpace(ctxutil.MoiUserIDFrom(ctx)) == "" {
		ctx = ctxutil.WithMoiUserID(ctx, "moi-user-1")
	}
	trusted, _ := ctxutil.CoreIAMRequestFrom(ctx)
	if strings.TrimSpace(trusted.VerifiedEffectiveRoleID) == "" {
		trusted.VerifiedEffectiveRoleID = "role-create"
		ctx = ctxutil.WithCoreIAMRequest(ctx, trusted)
	}
	return ctx
}

func requireSemanticModelExecutionHeaders(t *testing.T, r *http.Request) {
	t.Helper()
	if got := r.Header.Get("X-API-Key"); got != "system-key" {
		t.Fatalf("transport authorization = %q, want system key", got)
	}
	// coreclient.FromContext prefers Catalog UserID over durable API key. Either
	// execution mode is valid; freeze/rehydrate paths commonly carry MoiUserID.
	apiKey := r.Header.Get(internalservice.HeaderExecutionUserAPIKey)
	userID := r.Header.Get(internalservice.HeaderExecutionUserID)
	if apiKey == "caller-key" {
		return
	}
	if userID != "" {
		return
	}
	t.Fatalf("execution identity missing: api_key=%q user_id=%q, want caller-key or catalog user id", apiKey, userID)
}

func TestSemanticModelServiceListUsesCallerClientWhenSystemClientConfigured(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/workspaces/ws-1/semantic-models" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		requireSemanticModelExecutionHeaders(t, r)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"items": []map[string]any{
				{"id": 200, "name": "kb-authorized"},
			},
			"total": 1,
		})
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()

	svc := newSemanticModelTestService(t, server.URL, systemClient)
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	tenantMock.ExpectQuery("SELECT .*source_id.*FROM knowledge_base_sources").
		WithArgs(int64(200)).
		WillReturnRows(sqlmock.NewRows([]string{
			"source_id", "model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "source_type",
			"source_file_id", "source_table_id", "kb_file_id", "kb_table_id", "display_name", "source_path",
			"db_name", "table_name", "status", "error", "enabled", "expires_at", "tags", "force_enabled_after_expiry", "segment_version_id", "index_version",
			"size_bytes", "row_count", "created_by", "updated_by", "updated_at",
		}).
			AddRow("source-file-1", int64(200), int64(3), int64(11), int64(0), int64(0), kbSourceTypeCatalogFile, "source-file-1", nil, "kb-file-1", nil, "doc-1.pdf", `["raw","doc-1.pdf"]`, nil, nil, kbSourceStatusSucceeded, nil, true, nil, nil, false, nil, nil, nil, nil, nil, nil, nil).
			AddRow("source-volume-file-1", int64(200), int64(3), int64(11), int64(12), int64(0), kbSourceTypeCatalogFile, "source-file-2", nil, "kb-file-2", nil, "doc-2.pdf", `["raw","doc-2.pdf"]`, nil, nil, kbSourceStatusSucceeded, nil, true, nil, nil, false, nil, nil, nil, nil, nil, nil, nil).
			AddRow("source-table-1", int64(200), int64(3), int64(11), int64(0), int64(0), kbSourceTypeCatalogTable, nil, int64(1001), nil, int64(2001), "orders", `["sales_db","orders"]`, "sales_db", "orders", kbSourceStatusSucceeded, nil, true, nil, nil, false, nil, nil, nil, nil, nil, nil, nil))
	expectEmptyLegacySourceJobs(tenantMock, 200)
	expectNoKnowledgeBaseRawVolumeFiles(tenantMock, 200)
	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)

	resp, err := svc.ListModels(ctx, ListSemanticModelsRequest{PageSize: 20})
	if err != nil {
		t.Fatalf("ListModels: %v", err)
	}
	if resp.Total != 1 || len(resp.Items) != 1 || resp.Items[0].ID != 200 {
		t.Fatalf("ListModels response = %+v", resp)
	}
	if resp.Items[0].SourceCounts != (SemanticModelSourceCounts{Files: 2, Tables: 1, Total: 3}) {
		t.Fatalf("source counts = %+v", resp.Items[0].SourceCounts)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestIssue12732ListModelsBatchesSourceCountsPreservingLegacyCandidates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/workspaces/ws-1/semantic-models" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		requireSemanticModelExecutionHeaders(t, r)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"items": []map[string]any{
				{
					"id":     301,
					"name":   "kb-with-explicit-legacy",
					"files":  map[string]any{"file_ids": []string{"explicit-file-301"}},
					"tables": []map[string]any{{"db_name": "sales", "table_names": []string{"orders"}}},
				},
				{
					"id":     302,
					"name":   "kb-with-removed-source",
					"files":  map[string]any{"file_ids": []string{}},
					"tables": []map[string]any{},
				},
				{
					"id":     303,
					"name":   "kb-with-legacy-job",
					"files":  map[string]any{"file_ids": []string{}},
					"tables": []map[string]any{},
				},
				{
					"id":     304,
					"name":   "kb-with-raw-volume",
					"files":  map[string]any{"file_ids": []string{}},
					"tables": []map[string]any{},
				},
				{
					"id":     305,
					"name":   "kb-with-lineage",
					"files":  map[string]any{"file_ids": []string{}, "vector_table": "kb_305_text_index"},
					"tables": []map[string]any{},
				},
			},
			"total": 5,
		})
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()

	svc := newSemanticModelTestService(t, server.URL, systemClient)
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}

	tenantMock.ExpectQuery("(?s)SELECT kbs\\.source_id AS source_id.*FROM knowledge_base_sources kbs.*WHERE kbs\\.model_id IN \\(\\?,\\?,\\?,\\?,\\?\\)").
		WithArgs(int64(301), int64(302), int64(303), int64(304), int64(305)).
		WillReturnRows(sqlmock.NewRows([]string{
			"source_id", "model_id", "source_type", "source_file_id", "source_table_id", "kb_file_id", "kb_table_id", "db_name", "table_name", "status",
		}).
			AddRow("source-managed-301", int64(301), kbSourceTypeCatalogFile, "managed-file-301", nil, "managed-file-301", nil, nil, nil, kbSourceStatusSucceeded).
			AddRow("source-removed-302", int64(302), kbSourceTypeCatalogFile, "job-file-302", nil, "job-file-302", nil, nil, nil, kbSourceStatusRemoved))
	tenantMock.ExpectQuery("SELECT COALESCE\\(CASE WHEN v\\.catalog_id > 0 THEN v\\.catalog_id ELSE cd\\.catalog_id END").
		WithArgs("explicit-file-301").
		WillReturnRows(sqlmock.NewRows([]string{
			"catalog_id", "database_id", "volume_id", "size", "updated_at", "catalog_name", "database_name", "volume_name", "file_path", "file_name",
		}).AddRow(int64(3), int64(11), int64(12), int64(1024), int64(1782705001), "catalog", "database", "volume", "", "explicit.txt"))
	tenantMock.ExpectQuery("(?s)SELECT t\\.table_id, t\\.database_id, t\\.catalog_id.*FROM catalog_table t.*WHERE cd\\.database_name = \\? AND t\\.table_name = \\?").
		WithArgs("sales", "orders").
		WillReturnRows(sqlmock.NewRows([]string{"table_id", "database_id", "catalog_id", "database_name", "catalog_name"}).
			AddRow(int64(9001), int64(11), int64(3), "sales", "catalog"))
	tenantMock.ExpectQuery("(?s)SELECT id, model_id, source_type, source_file_id.*FROM knowledge_base_source_jobs.*WHERE model_id IN \\(\\?,\\?,\\?,\\?,\\?\\)").
		WithArgs(int64(301), int64(302), int64(303), int64(304), int64(305)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "model_id", "source_type", "source_file_id", "kb_file_id", "raw_volume_id", "job_status", "error", "segment_version_id", "index_version", "workflow_execution_id",
		}).
			AddRow(int64(1), int64(302), kbSourceTypeCatalogFile, nil, "job-file-302", int64(0), kbSourceJobSucceeded, nil, nil, nil, nil).
			AddRow(int64(2), int64(303), kbSourceTypeCatalogFile, nil, "job-file-303", int64(0), kbSourceJobSucceeded, nil, nil, nil, nil))
	tenantMock.ExpectQuery("SELECT COALESCE\\(CASE WHEN v\\.catalog_id > 0 THEN v\\.catalog_id ELSE cd\\.catalog_id END").
		WithArgs("job-file-303").
		WillReturnRows(sqlmock.NewRows([]string{
			"catalog_id", "database_id", "volume_id", "size", "updated_at", "catalog_name", "database_name", "volume_name", "file_path", "file_name",
		}).AddRow(int64(3), int64(11), int64(12), int64(2048), int64(1782705002), "catalog", "database", "volume", "", "job.txt"))
	tenantMock.ExpectQuery("SELECT model_id, raw_volume_id\\s+FROM knowledge_base_raw_volumes").
		WithArgs(int64(301), int64(302), int64(303), int64(304), int64(305)).
		WillReturnRows(sqlmock.NewRows([]string{"model_id", "raw_volume_id"}).
			AddRow(int64(304), int64(7001)).
			AddRow(int64(304), int64(7002)))
	tenantMock.ExpectQuery("SELECT model_id, raw_volume_id\\s+FROM knowledge_base_data_domains").
		WithArgs(int64(301), int64(302), int64(303), int64(304), int64(305)).
		WillReturnRows(sqlmock.NewRows([]string{"model_id", "raw_volume_id"}))
	tenantMock.ExpectQuery("SELECT vf\\.file_id, COALESCE\\(CASE WHEN v\\.catalog_id").
		WithArgs(int64(7001), int64(7002)).
		WillReturnRows(rawVolumeFileMetadataRows(1, 1).
			AddRow("raw-file-invalid", int64(3), int64(0), int64(7002), int64(2048), int64(1782705100), "catalog", "", "raw-volume", "invalid.pdf"))
	tenantMock.ExpectQuery("(?s)SELECT DISTINCT vector\\.asset_ref, COALESCE\\(pm\\.source_file_id, root\\.asset_ref\\) AS file_id.*WHERE vector\\.asset_type = 'vector_index'.*AND vector\\.asset_ref IN \\(\\?\\)").
		WithArgs("kb_305_text_index").
		WillReturnRows(sqlmock.NewRows([]string{"asset_ref", "file_id"}).AddRow("kb_305_text_index", "lineage-file-305"))
	tenantMock.ExpectQuery("SELECT COALESCE\\(CASE WHEN v\\.catalog_id > 0 THEN v\\.catalog_id ELSE cd\\.catalog_id END").
		WithArgs("lineage-file-305").
		WillReturnRows(sqlmock.NewRows([]string{
			"catalog_id", "database_id", "volume_id", "size", "updated_at", "catalog_name", "database_name", "volume_name", "file_path", "file_name",
		}).AddRow(int64(3), int64(11), int64(12), int64(4096), int64(1782705003), "catalog", "database", "volume", "", "lineage.txt"))

	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)

	resp, err := svc.ListModels(ctx, ListSemanticModelsRequest{PageSize: 12})
	if err != nil {
		t.Fatalf("ListModels: %v", err)
	}
	if resp.Total != 5 || len(resp.Items) != 5 {
		t.Fatalf("ListModels response = %+v", resp)
	}
	got := map[int64]SemanticModelSourceCounts{}
	for _, item := range resp.Items {
		got[item.ID] = item.SourceCounts
	}
	want := map[int64]SemanticModelSourceCounts{
		301: {Files: 2, Tables: 1, Total: 3},
		302: {},
		303: {Files: 1, Total: 1},
		304: {Files: 1, Total: 1},
		305: {Files: 1, Total: 1},
	}
	for modelID, wantCounts := range want {
		if got[modelID] != wantCounts {
			t.Fatalf("model %d source counts = %+v, want %+v", modelID, got[modelID], wantCounts)
		}
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelSourceCountsBatchSkipsStructuredRawVolumes(t *testing.T) {
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}

	tenantMock.ExpectQuery("(?s)SELECT kbs\\.source_id AS source_id.*FROM knowledge_base_sources kbs.*WHERE kbs\\.model_id IN \\(\\?,\\?\\)").
		WithArgs(int64(401), int64(402)).
		WillReturnRows(sqlmock.NewRows([]string{
			"source_id", "model_id", "source_type", "source_file_id", "source_table_id", "kb_file_id", "kb_table_id", "db_name", "table_name", "status",
		}).AddRow("source-table-401", int64(401), kbSourceTypeCatalogTable, nil, nil, nil, int64(9001), "kb_401", "orders", kbSourceStatusSucceeded))
	tenantMock.ExpectQuery("(?s)SELECT id, model_id, source_type, source_file_id.*FROM knowledge_base_source_jobs.*WHERE model_id IN \\(\\?,\\?\\)").
		WithArgs(int64(401), int64(402)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "model_id", "source_type", "source_file_id", "kb_file_id", "raw_volume_id", "job_status", "error", "segment_version_id", "index_version", "workflow_execution_id",
		}))
	tenantMock.ExpectQuery("SELECT model_id, raw_volume_id\\s+FROM knowledge_base_raw_volumes\\s+WHERE model_id IN \\(\\?,\\?\\) AND raw_volume_id > 0\\s+AND COALESCE\\(raw_kind, ''\\) <> 'structured'").
		WithArgs(int64(401), int64(402)).
		WillReturnRows(sqlmock.NewRows([]string{"model_id", "raw_volume_id"}))
	tenantMock.ExpectQuery("SELECT model_id, raw_volume_id\\s+FROM knowledge_base_data_domains").
		WithArgs(int64(401), int64(402)).
		WillReturnRows(sqlmock.NewRows([]string{"model_id", "raw_volume_id"}))

	ctx := ctxutil.WithTenantDB(context.Background(), tenantDB)
	svc := &semanticModelService{}
	counts, err := svc.semanticModelSourceCountsBatch(ctx, []*SemanticModelInfo{{ID: 401}, {ID: 402}})
	if err != nil {
		t.Fatalf("semanticModelSourceCountsBatch: %v", err)
	}
	if got, want := counts[401], (SemanticModelSourceCounts{Tables: 1, Total: 1}); got != want {
		t.Fatalf("model 401 source counts = %+v, want %+v", got, want)
	}
	if got := counts[402]; got != (SemanticModelSourceCounts{}) {
		t.Fatalf("model 402 source counts = %+v, want empty", got)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceGetModelSourceCountsIncludesAllLegacyRawVolumeCandidates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/workspaces/ws-1/semantic-models/2010" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		requireSemanticModelExecutionHeaders(t, r)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":     2010,
			"name":   "kb-raw-volume",
			"files":  map[string]any{"file_ids": []string{}},
			"tables": []map[string]any{},
		})
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()

	svc := newSemanticModelTestService(t, server.URL, systemClient)
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	tenantMock.ExpectQuery("SELECT .*source_id.*FROM knowledge_base_sources").
		WithArgs(int64(2010)).
		WillReturnRows(sqlmock.NewRows([]string{
			"source_id", "model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "source_type",
			"source_file_id", "source_table_id", "kb_file_id", "kb_table_id", "display_name", "source_path",
			"db_name", "table_name", "status", "error", "enabled", "expires_at", "tags", "force_enabled_after_expiry", "segment_version_id", "index_version",
			"size_bytes", "row_count", "created_by", "updated_by", "updated_at",
		}))
	expectEmptyLegacySourceJobs(tenantMock, 2010)
	tenantMock.ExpectQuery("SELECT raw_volume_id\\s+FROM knowledge_base_raw_volumes").
		WithArgs(int64(2010)).
		WillReturnRows(sqlmock.NewRows([]string{"raw_volume_id"}).AddRow(int64(7001)))
	tenantMock.ExpectQuery("SELECT raw_volume_id\\s+FROM knowledge_base_data_domains").
		WithArgs(int64(2010)).
		WillReturnRows(sqlmock.NewRows([]string{"raw_volume_id"}))
	tenantMock.ExpectQuery("SELECT vf\\.file_id, COALESCE\\(CASE WHEN v\\.catalog_id").
		WithArgs(int64(7001), kbLegacyBackfillBatchSize, 0).
		WillReturnRows(rawVolumeFileMetadataRows(1, kbLegacyBackfillBatchSize))
	tenantMock.ExpectQuery("SELECT vf\\.file_id, COALESCE\\(CASE WHEN v\\.catalog_id").
		WithArgs(int64(7001), kbLegacyBackfillBatchSize, kbLegacyBackfillBatchSize).
		WillReturnRows(rawVolumeFileMetadataRows(kbLegacyBackfillBatchSize+1, kbLegacyBackfillBatchSize+1))

	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)

	resp, err := svc.GetModel(ctx, 2010)
	if err != nil {
		t.Fatalf("GetModel: %v", err)
	}
	want := SemanticModelSourceCounts{Files: int64(kbLegacyBackfillBatchSize + 1), Total: int64(kbLegacyBackfillBatchSize + 1)}
	if resp.SourceCounts != want {
		t.Fatalf("source counts = %+v, want %+v", resp.SourceCounts, want)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceListSourcesReturnsBackendContractRows(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/ws-1/workflow-apps/executions/exec-rag-1" {
			requireSemanticModelExecutionHeaders(t, r)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"execution": map[string]any{"execution_id": "exec-rag-1", "status": "running"}})
			return
		}
		if r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/7" {
			requireSemanticModelExecutionHeaders(t, r)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":     7,
				"name":   "kb",
				"files":  map[string]any{"file_ids": []string{}},
				"tables": []map[string]any{},
			})
			return
		}
		if r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/ws-1/tables/2001" {
			requireSemanticModelExecutionHeaders(t, r)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"table": map[string]any{
					"id":          2001,
					"name":        "orders_current",
					"database_id": 31,
					"catalog_id":  21,
					"updated_at":  1782705900,
				},
				"database": map[string]any{
					"id":         31,
					"name":       "catalog_db",
					"catalog_id": 21,
					"display_bindings": []map[string]any{{
						"field":         "name",
						"display_owner": "moi_backend",
						"display_key":   "moi_backend.resource.literal_default_text",
						"default_text":  "Operations Knowledge",
					}},
				},
				"catalog": map[string]any{
					"id":   21,
					"name": "main_catalog",
					"display_bindings": []map[string]any{{
						"field":         "name",
						"display_owner": "moi_backend",
						"display_key":   "moi_backend.resource.literal_default_text",
						"default_text":  "Admin Catalog",
					}},
				},
			})
			return
		}
		t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()

	svc := newSemanticModelTestService(t, server.URL, systemClient)
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	statsSQL, statsMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("stats sqlmock: %v", err)
	}
	defer statsSQL.Close()
	svc.(*semanticModelService).openWorkspaceDB = func(_ context.Context, _ string, dbName string) (*sql.DB, error) {
		if dbName != "catalog_db" {
			t.Fatalf("stats db name = %q, want catalog_db", dbName)
		}
		return statsSQL, nil
	}
	tenantMock.ExpectQuery("SELECT .*source_id.*FROM knowledge_base_sources").
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{
			"source_id", "model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "source_type",
			"source_file_id", "source_table_id", "kb_file_id", "kb_table_id", "display_name", "source_path",
			"db_name", "table_name", "status", "error", "enabled", "expires_at", "tags", "force_enabled_after_expiry", "segment_version_id", "index_version",
			"size_bytes", "row_count", "created_by", "updated_by", "updated_at",
		}).
			AddRow("source-file-1", int64(7), int64(3), int64(11), int64(12), int64(13), kbSourceTypeCatalogFile, "source-file", nil, "kb-file", nil, "doc.pdf", `["raw","doc.pdf"]`, nil, nil, kbSourceStatusPending, nil, true, int64(1), `["finance","policy"]`, true, "seg-current", int64(12), int64(2048), nil, "creator-account-1", "updater-account-1", int64(1782705000)).
			AddRow("source-file-disabled-forced", int64(7), int64(3), int64(11), int64(12), int64(13), kbSourceTypeCatalogFile, "source-disabled", nil, "kb-file-disabled", nil, "disabled.pdf", `["raw","disabled.pdf"]`, nil, nil, kbSourceStatusSucceeded, nil, false, int64(1), nil, true, nil, nil, nil, nil, "creator-account-2", "updater-account-2", int64(1782705100)).
			AddRow("source-structured-pending", int64(7), int64(3), int64(11), int64(14), int64(13), kbSourceTypeCatalogTable, nil, nil, nil, nil, "structured_orders", nil, nil, nil, kbSourceStatusPending, nil, true, nil, nil, false, nil, nil, nil, nil, "creator-account-3", "updater-account-3", int64(1782705150)).
			AddRow("source-structured-incomplete-succeeded", int64(7), int64(3), int64(11), int64(14), int64(13), kbSourceTypeCatalogTable, nil, nil, nil, nil, "structured_customers", nil, nil, nil, kbSourceStatusSucceeded, nil, true, nil, nil, false, nil, nil, nil, nil, "creator-account-4", "updater-account-4", int64(1782705175)).
			AddRow("source-table-1", int64(7), int64(3), int64(11), int64(12), int64(13), kbSourceTypeCatalogTable, nil, int64(1001), nil, int64(2001), "orders", `["catalog_db","orders"]`, "kb_docs", "orders", kbSourceStatusSucceeded, nil, nil, nil, nil, false, nil, nil, nil, nil, "creator-account-table", "updater-account-table", int64(1782705200)))
	// List prefers source_file_id for catalog metadata (authoritative physical file).
	tenantMock.ExpectQuery("SELECT .*f.size, UNIX_TIMESTAMP").
		WithArgs("source-file", "source-disabled").
		WillReturnRows(sqlmock.NewRows([]string{
			"file_id", "catalog_id", "database_id", "volume_id", "size", "updated_at", "catalog_name", "database_name", "volume_name", "file_path", "file_name",
		}).
			AddRow("source-file", int64(21), int64(31), int64(12), int64(4096), int64(1782705800), "Admin Catalog", "Operations Knowledge", "raw_document", "policies/2026", "doc-current.pdf").
			AddRow("source-disabled", int64(21), int64(31), int64(12), int64(8192), int64(1782705850), "main_catalog", "raw_db", "documents", "", "disabled-current.pdf"))
	tenantMock.ExpectQuery("SELECT .*job_id.*source_id.*model_id.*job_type.*job_status.*idempotency_key").
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{
			"job_id", "source_id", "model_id", "job_type", "job_status", "idempotency_key", "operation_id",
			"workflow_execution_id", "runtime_actor_moi_user_id", "runtime_effective_role_id", "runtime_is_workspace_owner", "source_file_id", "kb_file_id", "source_table_id", "kb_table_id",
			"retry_count", "next_retry_at", "error", "created_at", "updated_at",
		}).
			AddRow("job-load-1", "source-file-1", int64(7), kbJobTypeCopy, kbSourceJobSucceeded, "idem-load-1", "copy:1", nil, nil, nil, false, "source-file", "kb-file", nil, nil, int64(0), nil, nil, int64(100), int64(100)).
			AddRow("job-rag-1", "source-file-1", int64(7), kbJobTypeRAGIngest, kbSourceJobQueued, "idem-rag-1", nil, "exec-rag-1", nil, nil, false, "source-file", "kb-file", nil, nil, int64(0), nil, nil, int64(100), int64(100)).
			AddRow("job-table-1", "source-table-1", int64(7), kbJobTypeTableClone, kbSourceJobSucceeded, "idem-table-1", "table_clone:1", nil, nil, nil, false, nil, nil, int64(1001), int64(2001), int64(0), nil, nil, int64(100), int64(100)))
	statsMock.ExpectQuery("SELECT tbl_name, mo_table_rows\\(db_name, tbl_name\\), mo_table_size\\(db_name, tbl_name\\) FROM \\(VALUES ROW\\(\\?, \\?\\)\\) AS requested\\(db_name, tbl_name\\)").
		WithArgs("catalog_db", "orders_current").
		WillReturnRows(sqlmock.NewRows([]string{"tbl_name", "row_count", "size_bytes"}).
			AddRow("orders_current", int64(12345), int64(65536)))
	expectEmptyLegacySourceJobs(tenantMock, 7)
	expectNoKnowledgeBaseRawVolumeFiles(tenantMock, 7)
	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)

	resp, err := svc.ListSources(ctx, ListSemanticModelSourcesParams{ModelID: 7})
	if err != nil {
		t.Fatalf("ListSources: %v", err)
	}
	if resp.Total != 5 || len(resp.Items) != 5 {
		t.Fatalf("ListSources response = %+v", resp)
	}

	file := resp.Items[0]
	if file.RowID != "source-file-1" || file.SourceType != SemanticModelSourceTypeFile || file.ModelID != 7 || file.ResourceID != "kb-file" {
		t.Fatalf("file source = %+v", file)
	}
	if file.IngestStatus == nil || *file.IngestStatus != kbSourceStatusPending {
		t.Fatalf("file ingest status = %v", file.IngestStatus)
	}
	if file.Enabled == nil || !*file.Enabled || file.ExpiresAt == nil || *file.ExpiresAt != 1 || file.SegmentVersionID == nil || *file.SegmentVersionID != "seg-current" || file.Error != nil {
		t.Fatalf("file governance fields = %+v", file)
	}
	if !file.Expired || !file.EffectiveEnabled || !file.ForceEnabled || file.IndexVersion == nil || *file.IndexVersion != 12 {
		t.Fatalf("file effective governance = %+v", file)
	}
	if !sameStringSet(file.Tags, []string{"finance", "policy"}) {
		t.Fatalf("file tags = %+v", file.Tags)
	}
	if file.CreatedBy == nil || *file.CreatedBy != "creator-account-1" || file.UpdatedBy == nil || *file.UpdatedBy != "updater-account-1" {
		t.Fatalf("file source operators = %+v", file)
	}
	if file.DisplayName == nil || *file.DisplayName != "doc-current.pdf" {
		t.Fatalf("file current display name = %v", file.DisplayName)
	}
	if file.SourcePath == nil || *file.SourcePath != "Admin Catalog/Operations Knowledge/raw_document" || file.SizeBytes == nil || *file.SizeBytes != 4096 || file.RowCount != nil || file.UpdatedAt == nil || *file.UpdatedAt != 1782705800 {
		t.Fatalf("file source metadata = %+v", file)
	}
	disabledForced := resp.Items[1]
	if disabledForced.RowID != "source-file-disabled-forced" || disabledForced.Enabled == nil || *disabledForced.Enabled || !disabledForced.ForceEnabled || disabledForced.EffectiveEnabled {
		t.Fatalf("disabled forced file governance = %+v", disabledForced)
	}
	pendingStructured := resp.Items[2]
	if pendingStructured.RowID != "source-structured-pending" || pendingStructured.SourceType != SemanticModelSourceTypeTable || pendingStructured.DisplayName == nil || *pendingStructured.DisplayName != "structured_orders" {
		t.Fatalf("pending structured source = %+v", pendingStructured)
	}
	if pendingStructured.ResourceID != "" || pendingStructured.SourceFileID != nil || pendingStructured.KBFileID != nil {
		t.Fatalf("pending structured source should not expose file relation: %+v", pendingStructured)
	}
	incompleteStructured := resp.Items[3]
	if incompleteStructured.RowID != "source-structured-incomplete-succeeded" || incompleteStructured.SourceType != SemanticModelSourceTypeTable || incompleteStructured.DisplayName == nil || *incompleteStructured.DisplayName != "structured_customers" {
		t.Fatalf("incomplete structured source = %+v", incompleteStructured)
	}
	if incompleteStructured.ResourceID != "" || incompleteStructured.IngestStatus == nil || *incompleteStructured.IngestStatus != kbSourceStatusPending {
		t.Fatalf("incomplete structured source should stay pending without table association: %+v", incompleteStructured)
	}
	table := resp.Items[4]
	if table.RowID != "source-table-1" || table.SourceType != SemanticModelSourceTypeTable || table.ResourceID != "2001" {
		t.Fatalf("table source = %+v", table)
	}
	if table.DisplayName == nil || *table.DisplayName != "orders_current" {
		t.Fatalf("table display name = %v", table.DisplayName)
	}
	if table.DBName == nil || *table.DBName != "catalog_db" || table.TableName == nil || *table.TableName != "orders_current" {
		t.Fatalf("table db/table fields = %+v", table)
	}
	if table.SourcePath == nil || *table.SourcePath != "Admin Catalog/Operations Knowledge" || table.RowCount == nil || *table.RowCount != 12345 || table.SizeBytes == nil || *table.SizeBytes != 65536 || table.UpdatedAt == nil || *table.UpdatedAt != 1782705900 {
		t.Fatalf("table current metadata = %+v", table)
	}
	if table.CreatedBy == nil || *table.CreatedBy != "creator-account-table" || table.UpdatedBy == nil || *table.UpdatedBy != "updater-account-table" {
		t.Fatalf("table source operators = %+v", table)
	}

	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
	if err := statsMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("stats sql expectations: %v", err)
	}
}

func TestSemanticModelServiceListSourcesKeepsRAGFilePendingUntilSegmentVersionPublished(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/workspaces/ws-1/semantic-models/7" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		requireSemanticModelExecutionHeaders(t, r)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":     7,
			"name":   "kb",
			"files":  map[string]any{"file_ids": []string{}},
			"tables": []map[string]any{},
		})
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()

	svc := newSemanticModelTestService(t, server.URL, systemClient)
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}

	tenantMock.ExpectQuery("SELECT .*source_id.*FROM knowledge_base_sources").
		WithArgs(int64(7)).
		WillReturnRows(knowledgeBaseSourceRecordRows(KnowledgeBaseSourceRecord{
			SourceID:          "source-file-1",
			ModelID:           7,
			CatalogID:         3,
			DatabaseID:        11,
			RawVolumeID:       12,
			ProcessedVolumeID: 13,
			SourceType:        kbSourceTypeCatalogFile,
			SourceFileID:      stringPtr("source-file"),
			KBFileID:          stringPtr("kb-file"),
			DisplayName:       stringPtr("doc.pdf"),
			Status:            kbSourceStatusPending,
			Enabled:           boolPtr(true),
		}))
	tenantMock.ExpectQuery("SELECT .*f.size, UNIX_TIMESTAMP").
		WithArgs("source-file").
		WillReturnRows(sqlmock.NewRows([]string{
			"file_id", "catalog_id", "database_id", "volume_id", "size", "updated_at", "catalog_name", "database_name", "volume_name", "file_path", "file_name",
		}).AddRow("source-file", int64(21), int64(31), int64(12), int64(4096), int64(1782705800), "main_catalog", "raw_db", "documents", "policies/2026", "doc-current.pdf"))
	tenantMock.ExpectQuery("SELECT .*job_id.*source_id.*model_id.*job_type.*job_status.*idempotency_key").
		WithArgs(int64(7)).
		WillReturnRows(knowledgeBaseSourceJobRunRows().
			AddRow("job-rag-1", "source-file-1", int64(7), kbJobTypeRAGIngest, kbSourceJobSucceeded, "idem-rag-1", "workflow_trigger:"+knowledgeBaseWorkflowID("ws-1", 7), nil, nil, nil, false, "source-file", "kb-file", nil, nil, int64(0), nil, nil, int64(100), int64(100)))
	expectEmptyLegacySourceJobs(tenantMock, 7)
	expectNoKnowledgeBaseRawVolumeFiles(tenantMock, 7)

	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)

	resp, err := svc.ListSources(ctx, ListSemanticModelSourcesParams{ModelID: 7})
	if err != nil {
		t.Fatalf("ListSources: %v", err)
	}
	if resp.Total != 1 || resp.Page != 1 || resp.PageSize != 1 || len(resp.Items) != 1 {
		t.Fatalf("ListSources response = %+v", resp)
	}
	source := resp.Items[0]
	if source.IngestStatus == nil || *source.IngestStatus != kbSourceStatusPending {
		t.Fatalf("ingest status = %v, want pending until segment/index pointer is published", source.IngestStatus)
	}
	if source.SegmentVersionID != nil || source.IndexVersion != nil {
		t.Fatalf("source version pointers = %v/%v, want none before publish", source.SegmentVersionID, source.IndexVersion)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceListSourcesReturnsSemanticModelFileCandidateWithoutBackfill(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/workspaces/ws-1/semantic-models/88" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":     88,
			"name":   "legacy-kb",
			"files":  map[string]any{"file_ids": []string{"external-file"}},
			"tables": []map[string]any{},
		})
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()

	svc := newSemanticModelTestService(t, server.URL, systemClient)
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	sourceColumns := []string{
		"source_id", "model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "source_type",
		"source_file_id", "source_table_id", "kb_file_id", "kb_table_id", "display_name", "source_path",
		"db_name", "table_name", "status", "error", "enabled", "expires_at", "tags", "force_enabled_after_expiry", "segment_version_id", "index_version",
		"size_bytes", "row_count", "updated_at",
	}
	tenantMock.ExpectQuery("SELECT .*source_id.*FROM knowledge_base_sources").
		WithArgs(int64(88)).
		WillReturnRows(sqlmock.NewRows(sourceColumns))
	expectEmptySourceJobRuns(tenantMock, 88)
	expectEmptyLegacySourceJobs(tenantMock, 88)
	tenantMock.ExpectQuery("SELECT COALESCE\\(CASE WHEN v\\.catalog_id").
		WithArgs("external-file").
		WillReturnRows(sqlmock.NewRows([]string{
			"catalog_id", "database_id", "volume_id", "size", "updated_at", "catalog_name", "database_name", "volume_name", "file_path", "file_name",
		}).AddRow(int64(301), int64(401), int64(501), int64(194162), int64(1782875758), "fangyuan", "test", "pdf", "20C114774.pdf", "20C114774.pdf"))
	expectNoKnowledgeBaseRawVolumeFiles(tenantMock, 88)

	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)

	resp, err := svc.ListSources(ctx, ListSemanticModelSourcesParams{ModelID: 88})
	if err != nil {
		t.Fatalf("ListSources: %v", err)
	}
	if resp.Total != 1 || len(resp.Items) != 1 {
		t.Fatalf("ListSources response = %+v", resp)
	}
	item := resp.Items[0]
	if item.SourceID != "" || item.RowID == "" || item.ResourceID != "external-file" || item.GovernanceStatus != SemanticModelSourceGovernanceLegacyUnbound {
		t.Fatalf("candidate source = %+v", item)
	}
	if item.LegacyOrigin == nil || *item.LegacyOrigin != SemanticModelSourceLegacyOriginExplicit {
		t.Fatalf("candidate origin = %v", item.LegacyOrigin)
	}
	if !resp.LegacyBackfillRequired {
		t.Fatalf("legacy backfill required = false, want true for missing semantic model file")
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceBackfillLegacySourcesCreatesMissingSourceAndJobRun(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/workspaces/ws-1/semantic-models/77" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":     77,
			"name":   "legacy-kb",
			"files":  map[string]any{"file_ids": []string{"kb-file-legacy"}},
			"tables": []map[string]any{},
		})
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()

	svc := newSemanticModelTestService(t, server.URL, systemClient)
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	sourceColumns := []string{
		"source_id", "model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "source_type",
		"source_file_id", "source_table_id", "kb_file_id", "kb_table_id", "display_name", "source_path",
		"db_name", "table_name", "status", "error", "enabled", "expires_at", "tags", "force_enabled_after_expiry", "segment_version_id", "index_version",
		"size_bytes", "row_count", "updated_at",
	}
	tenantMock.ExpectBegin()
	tenantMock.ExpectQuery("SELECT .*source_id.*FROM knowledge_base_sources").
		WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows(sourceColumns))
	tenantMock.ExpectQuery("SELECT id, model_id, source_type, source_file_id").
		WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "model_id", "source_type", "source_file_id", "kb_file_id", "raw_volume_id", "job_status", "error", "segment_version_id", "index_version", "workflow_execution_id",
		}).AddRow(int64(12), int64(77), kbSourceTypeCatalogFile, "catalog-file-1", "kb-file-legacy", int64(12), kbSourceJobSucceeded, nil, nil, int64(1), "exec-legacy-1"))
	expectEmptySourceJobRuns(tenantMock, 77)
	tenantMock.ExpectQuery("SELECT model_id, catalog_id, database_id, raw_volume_id, processed_volume_id, ensure_status, last_ensure_error, last_checked_at").
		WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows([]string{
			"model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "ensure_status", "last_ensure_error", "last_checked_at",
		}).AddRow(int64(77), int64(3), int64(11), int64(12), int64(13), kbEnsureStatusReady, nil, int64(100)))
	tenantMock.ExpectQuery("SELECT COALESCE\\(CASE WHEN v\\.catalog_id").
		WithArgs("kb-file-legacy").
		WillReturnRows(sqlmock.NewRows([]string{
			"catalog_id", "database_id", "volume_id", "size", "updated_at", "catalog_name", "database_name", "volume_name", "file_path", "file_name",
		}).AddRow(int64(3), int64(11), int64(12), int64(2048), int64(1782705000), "catalog", "kb_docs", "raw", "legacy.pdf", "legacy.pdf"))
	expectNoKnowledgeBaseRawVolumeFiles(tenantMock, 77)
	tenantMock.ExpectExec("INSERT INTO knowledge_base_sources").
		WillReturnResult(sqlmock.NewResult(1, 1))
	tenantMock.ExpectExec("UPDATE knowledge_base_sources\\s+SET source_file_id = COALESCE").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), int64(12), kbSourceJobSucceeded, nil, int64(1), "user-1", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	tenantMock.ExpectExec("UPDATE knowledge_base_source_job_runs").
		WillReturnResult(sqlmock.NewResult(0, 0))
	tenantMock.ExpectExec("INSERT INTO knowledge_base_source_job_runs").
		WillReturnResult(sqlmock.NewResult(1, 1))
	tenantMock.ExpectCommit()

	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)

	if err := svc.BackfillLegacySources(ctx, BackfillLegacyKnowledgeBaseSourcesParams{ModelID: 77}); err != nil {
		t.Fatalf("BackfillLegacySources: %v", err)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceBackfillLegacySourcesCreatesSemanticModelFileSource(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/workspaces/ws-1/semantic-models/77" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":     77,
			"name":   "legacy-kb",
			"files":  map[string]any{"file_ids": []string{"external-file"}},
			"tables": []map[string]any{},
		})
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()

	svc := newSemanticModelTestService(t, server.URL, systemClient)
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	sourceColumns := []string{
		"source_id", "model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "source_type",
		"source_file_id", "source_table_id", "kb_file_id", "kb_table_id", "display_name", "source_path",
		"db_name", "table_name", "status", "error", "enabled", "expires_at", "tags", "force_enabled_after_expiry", "segment_version_id", "index_version",
		"size_bytes", "row_count", "updated_at",
	}
	tenantMock.ExpectBegin()
	tenantMock.ExpectQuery("SELECT .*source_id.*FROM knowledge_base_sources").
		WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows(sourceColumns))
	expectEmptyLegacySourceJobs(tenantMock, 77)
	tenantMock.ExpectQuery("SELECT model_id, catalog_id, database_id, raw_volume_id, processed_volume_id, ensure_status, last_ensure_error, last_checked_at").
		WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows([]string{
			"model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "ensure_status", "last_ensure_error", "last_checked_at",
		}))
	tenantMock.ExpectQuery("SELECT COALESCE\\(CASE WHEN v\\.catalog_id").
		WithArgs("external-file").
		WillReturnRows(sqlmock.NewRows([]string{
			"catalog_id", "database_id", "volume_id", "size", "updated_at", "catalog_name", "database_name", "volume_name", "file_path", "file_name",
		}).AddRow(int64(301), int64(401), int64(501), int64(194162), int64(1782875758), "fangyuan", "test", "pdf", "20C114774.pdf", "20C114774.pdf"))
	expectNoKnowledgeBaseRawVolumeFiles(tenantMock, 77)
	tenantMock.ExpectExec("INSERT INTO knowledge_base_sources").
		WillReturnResult(sqlmock.NewResult(1, 1))
	tenantMock.ExpectCommit()

	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)

	if err := svc.BackfillLegacySources(ctx, BackfillLegacyKnowledgeBaseSourcesParams{ModelID: 77}); err != nil {
		t.Fatalf("BackfillLegacySources: %v", err)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceBackfillLegacySourcesCreatesLineageFileSourceAndRAGJob(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/workspaces/ws-1/semantic-models/77" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":   77,
			"name": "legacy-kb",
			"files": map[string]any{
				"file_ids":           []string{},
				"vector_table":       "kb_77_text_index",
				"image_vector_table": "kb_77_image_index",
			},
			"tables": []map[string]any{},
		})
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()

	svc := newSemanticModelTestService(t, server.URL, systemClient)
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	sourceColumns := []string{
		"source_id", "model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "source_type",
		"source_file_id", "source_table_id", "kb_file_id", "kb_table_id", "display_name", "source_path",
		"db_name", "table_name", "status", "error", "enabled", "expires_at", "tags", "force_enabled_after_expiry", "segment_version_id", "index_version",
		"size_bytes", "row_count", "updated_at",
	}
	tenantMock.ExpectBegin()
	tenantMock.ExpectQuery("SELECT .*source_id.*FROM knowledge_base_sources").
		WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows(sourceColumns))
	expectEmptyLegacySourceJobs(tenantMock, 77)
	tenantMock.ExpectQuery("SELECT model_id, catalog_id, database_id, raw_volume_id, processed_volume_id, ensure_status, last_ensure_error, last_checked_at").
		WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows([]string{
			"model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "ensure_status", "last_ensure_error", "last_checked_at",
		}))
	expectNoKnowledgeBaseRawVolumeFiles(tenantMock, 77)
	tenantMock.ExpectQuery("SELECT DISTINCT COALESCE\\(pm\\.source_file_id, root\\.asset_ref\\) AS file_id").
		WithArgs("kb_77_text_index", "kb_77_image_index", int64(77), kbLegacyBackfillBatchSize).
		WillReturnRows(sqlmock.NewRows([]string{"file_id"}).AddRow("lineage-file-1"))
	tenantMock.ExpectQuery("SELECT COALESCE\\(CASE WHEN v\\.catalog_id").
		WithArgs("lineage-file-1").
		WillReturnRows(sqlmock.NewRows([]string{
			"catalog_id", "database_id", "volume_id", "size", "updated_at", "catalog_name", "database_name", "volume_name", "file_path", "file_name",
		}).AddRow(int64(301), int64(401), int64(501), int64(194162), int64(1782875758), "fangyuan", "test", "pdf", "20C114774.pdf", "20C114774.pdf"))
	tenantMock.ExpectExec("INSERT INTO knowledge_base_sources").
		WillReturnResult(sqlmock.NewResult(1, 1))
	tenantMock.ExpectExec("UPDATE knowledge_base_source_job_runs").
		WithArgs(nil, nil, false, kbSourceJobSucceeded, "lineage_register:lineage-file-1", nil, "lineage-file-1", "lineage-file-1", nil, nil, int64(0), nil, nil, "user-1", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 0))
	tenantMock.ExpectExec("INSERT INTO knowledge_base_source_job_runs").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), int64(77), kbJobTypeRAGIngest, kbSourceJobSucceeded, sqlmock.AnyArg(), "lineage_register:lineage-file-1", nil, nil, nil, false, "lineage-file-1", "lineage-file-1", nil, nil, int64(0), nil, nil, "user-1", "user-1").
		WillReturnResult(sqlmock.NewResult(1, 1))
	tenantMock.ExpectCommit()

	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)

	if err := svc.BackfillLegacySources(ctx, BackfillLegacyKnowledgeBaseSourcesParams{ModelID: 77}); err != nil {
		t.Fatalf("BackfillLegacySources: %v", err)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceBackfillLegacySourcesSkipsMissingSemanticModelFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/workspaces/ws-1/semantic-models/77" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":     77,
			"name":   "legacy-kb",
			"files":  map[string]any{"file_ids": []string{"missing-file"}},
			"tables": []map[string]any{},
		})
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()

	svc := newSemanticModelTestService(t, server.URL, systemClient)
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	sourceColumns := []string{
		"source_id", "model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "source_type",
		"source_file_id", "source_table_id", "kb_file_id", "kb_table_id", "display_name", "source_path",
		"db_name", "table_name", "status", "error", "enabled", "expires_at", "tags", "force_enabled_after_expiry", "segment_version_id", "index_version",
		"size_bytes", "row_count", "updated_at",
	}
	tenantMock.ExpectBegin()
	tenantMock.ExpectQuery("SELECT .*source_id.*FROM knowledge_base_sources").
		WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows(sourceColumns))
	expectEmptyLegacySourceJobs(tenantMock, 77)
	tenantMock.ExpectQuery("SELECT model_id, catalog_id, database_id, raw_volume_id, processed_volume_id, ensure_status, last_ensure_error, last_checked_at").
		WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows([]string{
			"model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "ensure_status", "last_ensure_error", "last_checked_at",
		}))
	tenantMock.ExpectQuery("SELECT COALESCE\\(CASE WHEN v\\.catalog_id").
		WithArgs("missing-file").
		WillReturnRows(sqlmock.NewRows([]string{
			"file_id", "catalog_id", "database_id", "volume_id", "size", "updated_at", "catalog_name", "database_name", "volume_name", "file_path", "file_name",
		}))
	expectNoKnowledgeBaseRawVolumeFiles(tenantMock, 77)
	tenantMock.ExpectCommit()

	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)

	if err := svc.BackfillLegacySources(ctx, BackfillLegacyKnowledgeBaseSourcesParams{ModelID: 77}); err != nil {
		t.Fatalf("BackfillLegacySources: %v", err)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceBackfillLegacySourcesCreatesSemanticModelTableSource(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/workspaces/ws-1/semantic-models/77" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":     77,
			"name":   "legacy-kb",
			"files":  map[string]any{"file_ids": []string{}},
			"tables": []map[string]any{{"db_name": "external_db", "table_names": []string{"orders"}}},
		})
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()

	svc := newSemanticModelTestService(t, server.URL, systemClient)
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	sourceColumns := []string{
		"source_id", "model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "source_type",
		"source_file_id", "source_table_id", "kb_file_id", "kb_table_id", "display_name", "source_path",
		"db_name", "table_name", "status", "error", "enabled", "expires_at", "tags", "force_enabled_after_expiry", "segment_version_id", "index_version",
		"size_bytes", "row_count", "updated_at",
	}
	tenantMock.ExpectBegin()
	tenantMock.ExpectQuery("SELECT .*source_id.*FROM knowledge_base_sources").
		WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows(sourceColumns))
	expectEmptyLegacySourceJobs(tenantMock, 77)
	tenantMock.ExpectQuery("SELECT model_id, catalog_id, database_id, raw_volume_id, processed_volume_id, ensure_status, last_ensure_error, last_checked_at").
		WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows([]string{
			"model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "ensure_status", "last_ensure_error", "last_checked_at",
		}))
	tenantMock.ExpectQuery("SELECT t\\.table_id, t\\.database_id, t\\.catalog_id").
		WithArgs("external_db", "orders").
		WillReturnRows(sqlmock.NewRows([]string{
			"table_id", "database_id", "catalog_id", "database_name", "catalog_name",
		}).AddRow(int64(3001), int64(301), int64(201), "external_db", "external_catalog"))
	expectNoKnowledgeBaseRawVolumeFiles(tenantMock, 77)
	tenantMock.ExpectExec("INSERT INTO knowledge_base_sources").
		WillReturnResult(sqlmock.NewResult(1, 1))
	tenantMock.ExpectCommit()

	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)

	if err := svc.BackfillLegacySources(ctx, BackfillLegacyKnowledgeBaseSourcesParams{ModelID: 77}); err != nil {
		t.Fatalf("BackfillLegacySources: %v", err)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelTableSourceRecordDoesNotMarkSourceTableAsKBTable(t *testing.T) {
	record, err := semanticModelTableSourceRecord(77, catalogTableSourceRef{
		tableID:    3001,
		databaseID: 301,
		catalogID:  201,
		dbName:     "external_db",
		tableName:  "orders",
		path:       []string{"external_catalog", "external_db", "orders"},
	})
	if err != nil {
		t.Fatalf("semanticModelTableSourceRecord: %v", err)
	}
	if record.SourceTableID == nil || *record.SourceTableID != 3001 {
		t.Fatalf("source_table_id = %v, want 3001", record.SourceTableID)
	}
	if record.KBTableID != nil {
		t.Fatalf("kb_table_id = %v, want nil", *record.KBTableID)
	}
	source := sourceRecordToSemanticModelSource(record)
	if source.ResourceID != "3001" || source.KBTableID != nil || source.SourceTableID == nil || *source.SourceTableID != 3001 {
		t.Fatalf("semantic model source = %+v", source)
	}
}

func TestSemanticModelServiceBackfillLegacySourcesCreatesRawVolumeSourcesInBatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/workspaces/ws-1/semantic-models/77" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":     77,
			"name":   "legacy-kb",
			"files":  map[string]any{"file_ids": []string{}},
			"tables": []map[string]any{},
		})
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()

	svc := newSemanticModelTestService(t, server.URL, systemClient)
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	sourceColumns := []string{
		"source_id", "model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "source_type",
		"source_file_id", "source_table_id", "kb_file_id", "kb_table_id", "display_name", "source_path",
		"db_name", "table_name", "status", "error", "enabled", "expires_at", "tags", "force_enabled_after_expiry", "segment_version_id", "index_version",
		"size_bytes", "row_count", "updated_at",
	}
	tenantMock.ExpectBegin()
	tenantMock.ExpectQuery("SELECT .*source_id.*FROM knowledge_base_sources").
		WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows(sourceColumns))
	expectEmptyLegacySourceJobs(tenantMock, 77)
	tenantMock.ExpectQuery("SELECT model_id, catalog_id, database_id, raw_volume_id, processed_volume_id, ensure_status, last_ensure_error, last_checked_at").
		WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows([]string{
			"model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "ensure_status", "last_ensure_error", "last_checked_at",
		}).AddRow(int64(77), int64(3), int64(11), int64(12), int64(13), kbEnsureStatusReady, nil, int64(100)))
	expectKnowledgeBaseRawVolumeFileMetadata(tenantMock, 77, 12, sqlmock.NewRows([]string{
		"file_id", "catalog_id", "database_id", "volume_id", "size", "updated_at", "catalog_name", "database_name", "volume_name", "file_name",
	}).AddRow("raw-file-1", int64(3), int64(11), int64(12), int64(2048), int64(1782705000), "catalog", "kb_docs", "raw_document", "a.pdf").
		AddRow("raw-file-2", int64(3), int64(11), int64(12), int64(4096), int64(1782705100), "catalog", "kb_docs", "raw_document", "b.pdf"))
	tenantMock.ExpectExec("INSERT INTO knowledge_base_sources").
		WillReturnResult(sqlmock.NewResult(1, 1))
	tenantMock.ExpectExec("INSERT INTO knowledge_base_sources").
		WillReturnResult(sqlmock.NewResult(2, 1))
	tenantMock.ExpectCommit()

	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)

	if err := svc.BackfillLegacySources(ctx, BackfillLegacyKnowledgeBaseSourcesParams{ModelID: 77}); err != nil {
		t.Fatalf("BackfillLegacySources: %v", err)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceBackfillLegacySourcesIsIdempotentOnDuplicateSource(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/workspaces/ws-1/semantic-models/77" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":     77,
			"name":   "legacy-kb",
			"files":  map[string]any{"file_ids": []string{}},
			"tables": []map[string]any{},
		})
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()

	svc := newSemanticModelTestService(t, server.URL, systemClient)
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	sourceColumns := []string{
		"source_id", "model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "source_type",
		"source_file_id", "source_table_id", "kb_file_id", "kb_table_id", "display_name", "source_path",
		"db_name", "table_name", "status", "error", "enabled", "expires_at", "tags", "force_enabled_after_expiry", "segment_version_id", "index_version",
		"size_bytes", "row_count", "updated_at",
	}
	tenantMock.ExpectBegin()
	tenantMock.ExpectQuery("SELECT .*source_id.*FROM knowledge_base_sources").
		WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows(sourceColumns))
	expectEmptyLegacySourceJobs(tenantMock, 77)
	tenantMock.ExpectQuery("SELECT model_id, catalog_id, database_id, raw_volume_id, processed_volume_id, ensure_status, last_ensure_error, last_checked_at").
		WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows([]string{
			"model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "ensure_status", "last_ensure_error", "last_checked_at",
		}).AddRow(int64(77), int64(3), int64(11), int64(12), int64(13), kbEnsureStatusReady, nil, int64(100)))
	expectKnowledgeBaseRawVolumeFileMetadata(tenantMock, 77, 12, sqlmock.NewRows([]string{
		"file_id", "catalog_id", "database_id", "volume_id", "size", "updated_at", "catalog_name", "database_name", "volume_name", "file_name",
	}).AddRow("raw-file-1", int64(3), int64(11), int64(12), int64(2048), int64(1782705000), "catalog", "kb_docs", "raw_document", "a.pdf"))
	tenantMock.ExpectExec("INSERT INTO knowledge_base_sources").
		WillReturnError(&mysqlDriver.MySQLError{Number: 1062, Message: "Duplicate entry"})
	tenantMock.ExpectQuery("SELECT kbs\\.source_id AS source_id").
		WithArgs(int64(77), "raw-file-1").
		WillReturnRows(sqlmock.NewRows(sourceColumns).
			AddRow("source-existing", int64(77), int64(3), int64(11), int64(12), int64(13), kbSourceTypeLocalFile, "raw-file-1", nil, "raw-file-1", nil, "a.pdf", `["catalog","kb_docs","raw_document"]`, nil, nil, kbSourceStatusSucceeded, nil, true, nil, `[]`, false, nil, nil, nil, nil, int64(1782705000)))
	tenantMock.ExpectCommit()

	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)

	if err := svc.BackfillLegacySources(ctx, BackfillLegacyKnowledgeBaseSourcesParams{ModelID: 77}); err != nil {
		t.Fatalf("BackfillLegacySources: %v", err)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestUpsertKnowledgeBaseSourceJobRunWithTxIgnoresDuplicateInsert(t *testing.T) {
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	tenantMock.ExpectExec("UPDATE knowledge_base_source_job_runs").
		WillReturnResult(sqlmock.NewResult(0, 0))
	tenantMock.ExpectExec("INSERT INTO knowledge_base_source_job_runs").
		WillReturnError(&mysqlDriver.MySQLError{Number: 1062, Message: "Duplicate entry"})

	job := &KnowledgeBaseSourceJobRun{
		JobID:          "job-1",
		SourceID:       "source-1",
		ModelID:        77,
		JobType:        kbJobTypeRAGIngest,
		JobStatus:      kbSourceJobSucceeded,
		IdempotencyKey: "idem-1",
		SourceFileID:   stringPtr("source-file"),
		KBFileID:       stringPtr("kb-file"),
	}
	if err := upsertKnowledgeBaseSourceJobRunWithTx(tenantDB, job, "user-1"); err != nil {
		t.Fatalf("upsertKnowledgeBaseSourceJobRunWithTx: %v", err)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceBackfillLegacySourcesPreservesGovernanceFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/workspaces/ws-1/semantic-models/77" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":     77,
			"name":   "legacy-kb",
			"files":  map[string]any{"file_ids": []string{"kb-file-legacy"}},
			"tables": []map[string]any{},
		})
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()

	svc := newSemanticModelTestService(t, server.URL, systemClient)
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	sourceColumns := []string{
		"source_id", "model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "source_type",
		"source_file_id", "source_table_id", "kb_file_id", "kb_table_id", "display_name", "source_path",
		"db_name", "table_name", "status", "error", "enabled", "expires_at", "tags", "force_enabled_after_expiry", "segment_version_id", "index_version",
		"size_bytes", "row_count", "updated_at",
	}
	tenantMock.ExpectBegin()
	tenantMock.ExpectQuery("SELECT .*source_id.*FROM knowledge_base_sources").
		WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows(sourceColumns).
			AddRow("source-existing", int64(77), int64(3), int64(11), int64(12), int64(13), kbSourceTypeCatalogFile, nil, nil, "kb-file-legacy", nil, "legacy.pdf", `["catalog","kb_docs","raw"]`, nil, nil, kbSourceStatusPending, nil, false, int64(1782700000), `["keep"]`, true, nil, nil, int64(2048), nil, int64(1782705000)))
	tenantMock.ExpectQuery("SELECT id, model_id, source_type, source_file_id").
		WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "model_id", "source_type", "source_file_id", "kb_file_id", "raw_volume_id", "job_status", "error", "segment_version_id", "index_version", "workflow_execution_id",
		}).AddRow(int64(12), int64(77), kbSourceTypeCatalogFile, "catalog-file-1", "kb-file-legacy", int64(12), kbSourceJobSucceeded, nil, nil, int64(1), "exec-legacy-1"))
	expectEmptySourceJobRuns(tenantMock, 77)
	tenantMock.ExpectQuery("SELECT model_id, catalog_id, database_id, raw_volume_id, processed_volume_id, ensure_status, last_ensure_error, last_checked_at").
		WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows([]string{
			"model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "ensure_status", "last_ensure_error", "last_checked_at",
		}).AddRow(int64(77), int64(3), int64(11), int64(12), int64(13), kbEnsureStatusReady, nil, int64(100)))
	expectNoKnowledgeBaseRawVolumeFiles(tenantMock, 77)
	tenantMock.ExpectExec("UPDATE knowledge_base_sources\\s+SET source_file_id = COALESCE").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), int64(12), kbSourceJobSucceeded, nil, int64(1), "user-1", "source-existing").
		WillReturnResult(sqlmock.NewResult(1, 1))
	tenantMock.ExpectExec("UPDATE knowledge_base_source_job_runs").
		WillReturnResult(sqlmock.NewResult(1, 1))
	tenantMock.ExpectCommit()

	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)

	if err := svc.BackfillLegacySources(ctx, BackfillLegacyKnowledgeBaseSourcesParams{ModelID: 77}); err != nil {
		t.Fatalf("BackfillLegacySources: %v", err)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceBackfillLegacySourcesSkipsRemovedSourceRelation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/workspaces/ws-1/semantic-models/77" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":     77,
			"name":   "legacy-kb",
			"files":  map[string]any{"file_ids": []string{}},
			"tables": []map[string]any{},
		})
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()

	svc := newSemanticModelTestService(t, server.URL, systemClient)
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}

	tenantMock.ExpectBegin()
	tenantMock.ExpectQuery("SELECT .*source_id.*FROM knowledge_base_sources kbs .*WHERE kbs\\.model_id = \\? ORDER BY").
		WithArgs(int64(77)).
		WillReturnRows(knowledgeBaseSourceRecordRows(KnowledgeBaseSourceRecord{
			SourceID:          "source-removed",
			ModelID:           77,
			CatalogID:         3,
			DatabaseID:        11,
			RawVolumeID:       12,
			ProcessedVolumeID: 13,
			SourceType:        kbSourceTypeCatalogFile,
			SourceFileID:      stringPtr("file-removed"),
			KBFileID:          stringPtr("file-removed"),
			DisplayName:       stringPtr("removed.pdf"),
			Status:            kbSourceStatusRemoved,
		}))
	tenantMock.ExpectQuery("SELECT id, model_id, source_type, source_file_id").
		WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "model_id", "source_type", "source_file_id", "kb_file_id", "raw_volume_id", "job_status", "error", "segment_version_id", "index_version", "workflow_execution_id",
		}).AddRow(int64(12), int64(77), kbSourceTypeCatalogFile, "file-removed", "file-removed", int64(12), kbSourceJobSucceeded, nil, nil, int64(1), "exec-removed"))
	expectEmptySourceJobRuns(tenantMock, 77)
	tenantMock.ExpectQuery("SELECT model_id, catalog_id, database_id, raw_volume_id, processed_volume_id, ensure_status, last_ensure_error, last_checked_at").
		WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows([]string{
			"model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "ensure_status", "last_ensure_error", "last_checked_at",
		}).AddRow(int64(77), int64(3), int64(11), int64(12), int64(13), kbEnsureStatusReady, nil, int64(100)))
	expectNoKnowledgeBaseRawVolumeFiles(tenantMock, 77)
	tenantMock.ExpectCommit()

	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)

	if err := svc.BackfillLegacySources(ctx, BackfillLegacyKnowledgeBaseSourcesParams{ModelID: 77}); err != nil {
		t.Fatalf("BackfillLegacySources: %v", err)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceBackfillLegacySourcesCreatesSourceWhenOnlySourceFileMatches(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/workspaces/ws-1/semantic-models/77" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":     77,
			"name":   "legacy-kb",
			"files":  map[string]any{"file_ids": []string{"kb-file-new"}},
			"tables": []map[string]any{},
		})
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()

	svc := newSemanticModelTestService(t, server.URL, systemClient)
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	sourceColumns := []string{
		"source_id", "model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "source_type",
		"source_file_id", "source_table_id", "kb_file_id", "kb_table_id", "display_name", "source_path",
		"db_name", "table_name", "status", "error", "enabled", "expires_at", "tags", "force_enabled_after_expiry", "segment_version_id", "index_version",
		"size_bytes", "row_count", "updated_at",
	}
	tenantMock.ExpectBegin()
	tenantMock.ExpectQuery("SELECT .*source_id.*FROM knowledge_base_sources").
		WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows(sourceColumns).
			AddRow("source-existing", int64(77), int64(3), int64(11), int64(12), int64(13), kbSourceTypeCatalogFile, "catalog-file-1", nil, "kb-file-old", nil, "old.pdf", `["catalog","kb_docs","raw"]`, nil, nil, kbSourceStatusSucceeded, nil, true, nil, `[]`, false, nil, nil, int64(2048), nil, int64(1782705000)))
	tenantMock.ExpectQuery("SELECT id, model_id, source_type, source_file_id").
		WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "model_id", "source_type", "source_file_id", "kb_file_id", "raw_volume_id", "job_status", "error", "segment_version_id", "index_version", "workflow_execution_id",
		}).AddRow(int64(12), int64(77), kbSourceTypeCatalogFile, "catalog-file-1", "kb-file-new", int64(12), kbSourceJobSucceeded, nil, nil, int64(1), "exec-legacy-1"))
	expectEmptySourceJobRuns(tenantMock, 77)
	tenantMock.ExpectQuery("SELECT model_id, catalog_id, database_id, raw_volume_id, processed_volume_id, ensure_status, last_ensure_error, last_checked_at").
		WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows([]string{
			"model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "ensure_status", "last_ensure_error", "last_checked_at",
		}).AddRow(int64(77), int64(3), int64(11), int64(12), int64(13), kbEnsureStatusReady, nil, int64(100)))
	tenantMock.ExpectQuery("SELECT COALESCE\\(CASE WHEN v\\.catalog_id").
		WithArgs("kb-file-new").
		WillReturnRows(sqlmock.NewRows([]string{
			"catalog_id", "database_id", "volume_id", "size", "updated_at", "catalog_name", "database_name", "volume_name", "file_path", "file_name",
		}).AddRow(int64(3), int64(11), int64(12), int64(2048), int64(1782705000), "catalog", "kb_docs", "raw", "new.pdf", "new.pdf"))
	expectNoKnowledgeBaseRawVolumeFiles(tenantMock, 77)
	tenantMock.ExpectExec("INSERT INTO knowledge_base_sources").
		WillReturnResult(sqlmock.NewResult(1, 1))
	tenantMock.ExpectExec("UPDATE knowledge_base_sources\\s+SET source_file_id = COALESCE").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), int64(12), kbSourceJobSucceeded, nil, int64(1), "user-1", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	tenantMock.ExpectExec("UPDATE knowledge_base_source_job_runs").
		WillReturnResult(sqlmock.NewResult(0, 0))
	tenantMock.ExpectExec("INSERT INTO knowledge_base_source_job_runs").
		WillReturnResult(sqlmock.NewResult(1, 1))
	tenantMock.ExpectCommit()

	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)

	if err := svc.BackfillLegacySources(ctx, BackfillLegacyKnowledgeBaseSourcesParams{ModelID: 77}); err != nil {
		t.Fatalf("BackfillLegacySources: %v", err)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceLegacyBackfillRequiredUsesKBFileIdentity(t *testing.T) {
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	tenantMock.ExpectQuery("SELECT id, model_id, source_type, source_file_id").
		WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "model_id", "source_type", "source_file_id", "kb_file_id", "raw_volume_id", "job_status", "error", "segment_version_id", "index_version", "workflow_execution_id",
		}).AddRow(int64(12), int64(77), kbSourceTypeCatalogFile, "catalog-file-1", "kb-file-new", int64(12), kbSourceJobSucceeded, nil, nil, int64(1), "exec-legacy-1"))
	expectNoKnowledgeBaseRawVolumeFiles(tenantMock, 77)

	svc := &semanticModelService{}
	ctx := ctxutil.WithTenantDB(context.Background(), tenantDB)
	required, err := svc.legacyBackfillRequired(ctx, &SemanticModelInfo{ID: 77}, []KnowledgeBaseSourceRecord{{
		ModelID:      77,
		SourceType:   kbSourceTypeCatalogFile,
		SourceFileID: stringPtr("catalog-file-1"),
		KBFileID:     stringPtr("kb-file-old"),
	}}, nil)
	if err != nil {
		t.Fatalf("legacyBackfillRequired: %v", err)
	}
	if !required {
		t.Fatal("legacyBackfillRequired = false, want true when source_file_id matches but kb_file_id differs")
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceListSourcesReturnsRawVolumeCandidateWithoutBackfill(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/workspaces/ws-1/semantic-models/91" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":     91,
			"name":   "kb-with-raw-volume-files",
			"files":  map[string]any{"file_ids": []string{}},
			"tables": []map[string]any{},
		})
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()

	svc := newSemanticModelTestService(t, server.URL, systemClient)
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	sourceColumns := []string{
		"source_id", "model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "source_type",
		"source_file_id", "source_table_id", "kb_file_id", "kb_table_id", "display_name", "source_path",
		"db_name", "table_name", "status", "error", "enabled", "expires_at", "tags", "force_enabled_after_expiry", "segment_version_id", "index_version",
		"size_bytes", "row_count", "updated_at",
	}
	tenantMock.ExpectQuery("SELECT .*source_id.*FROM knowledge_base_sources").
		WithArgs(int64(91)).
		WillReturnRows(sqlmock.NewRows(sourceColumns))
	expectEmptySourceJobRuns(tenantMock, 91)
	tenantMock.ExpectQuery("SELECT id, model_id, source_type, source_file_id").
		WithArgs(int64(91)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "model_id", "source_type", "source_file_id", "kb_file_id", "raw_volume_id", "job_status", "error", "segment_version_id", "index_version", "workflow_execution_id",
		}))
	expectKnowledgeBaseRawVolumeFileMetadata(tenantMock, 91, 12, sqlmock.NewRows([]string{
		"file_id", "catalog_id", "database_id", "volume_id", "size", "updated_at", "catalog_name", "database_name", "volume_name", "file_name",
	}).AddRow("raw-file-1", int64(3), int64(11), int64(12), int64(2048), int64(1782705000), "catalog", "kb", "raw", "raw-file.pdf"))

	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)

	resp, err := svc.ListSources(ctx, ListSemanticModelSourcesParams{ModelID: 91})
	if err != nil {
		t.Fatalf("ListSources: %v", err)
	}
	if resp.Total != 1 || len(resp.Items) != 1 {
		t.Fatalf("ListSources response = %+v", resp)
	}
	item := resp.Items[0]
	if item.SourceID != "" || item.ResourceID != "raw-file-1" || item.GovernanceStatus != SemanticModelSourceGovernanceLegacyUnbound {
		t.Fatalf("raw volume candidate = %+v", item)
	}
	if !resp.LegacyBackfillRequired {
		t.Fatalf("legacy backfill required = false, want true for missing raw volume file")
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceListSourcesSkipsStructuredRawVolumeCandidate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/workspaces/ws-1/semantic-models/93" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":     93,
			"name":   "kb-with-structured-raw-volume",
			"files":  map[string]any{"file_ids": []string{}},
			"tables": []map[string]any{},
		})
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()

	svc := newSemanticModelTestService(t, server.URL, systemClient)
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	tenantMock.ExpectQuery("SELECT .*source_id.*FROM knowledge_base_sources").
		WithArgs(int64(93)).
		WillReturnRows(sqlmock.NewRows([]string{
			"source_id", "model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "source_type",
			"source_file_id", "source_table_id", "kb_file_id", "kb_table_id", "display_name", "source_path",
			"db_name", "table_name", "status", "error", "enabled", "expires_at", "tags", "force_enabled_after_expiry", "segment_version_id", "index_version",
			"size_bytes", "row_count", "updated_at",
		}))
	expectEmptySourceJobRuns(tenantMock, 93)
	tenantMock.ExpectQuery("SELECT id, model_id, source_type, source_file_id").
		WithArgs(int64(93)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "model_id", "source_type", "source_file_id", "kb_file_id", "raw_volume_id", "job_status", "error", "segment_version_id", "index_version", "workflow_execution_id",
		}))
	tenantMock.ExpectQuery("SELECT raw_volume_id\\s+FROM knowledge_base_raw_volumes\\s+WHERE model_id = \\? AND raw_volume_id > 0\\s+AND COALESCE\\(raw_kind, ''\\) <> 'structured'").
		WithArgs(int64(93)).
		WillReturnRows(sqlmock.NewRows([]string{"raw_volume_id"}))
	tenantMock.ExpectQuery("SELECT raw_volume_id\\s+FROM knowledge_base_data_domains").
		WithArgs(int64(93)).
		WillReturnRows(sqlmock.NewRows([]string{"raw_volume_id"}))

	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)

	resp, err := svc.ListSources(ctx, ListSemanticModelSourcesParams{ModelID: 93})
	if err != nil {
		t.Fatalf("ListSources: %v", err)
	}
	if resp.Total != 0 || len(resp.Items) != 0 {
		t.Fatalf("ListSources response = %+v", resp)
	}
	if resp.LegacyBackfillRequired {
		t.Fatalf("legacy backfill required = true, want false for structured raw volume")
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceListSourcesHidesRemovedSourceAndSuppressesLegacyCandidate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/workspaces/ws-1/semantic-models/92" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":     92,
			"name":   "kb-with-removed-source",
			"files":  map[string]any{"file_ids": []string{}},
			"tables": []map[string]any{},
		})
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()

	svc := newSemanticModelTestService(t, server.URL, systemClient)
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}

	tenantMock.ExpectQuery("SELECT .*source_id.*FROM knowledge_base_sources kbs .*WHERE kbs\\.model_id = \\? ORDER BY").
		WithArgs(int64(92)).
		WillReturnRows(knowledgeBaseSourceRecordRows(KnowledgeBaseSourceRecord{
			SourceID:          "source-removed",
			ModelID:           92,
			CatalogID:         3,
			DatabaseID:        11,
			RawVolumeID:       12,
			ProcessedVolumeID: 13,
			SourceType:        kbSourceTypeCatalogFile,
			SourceFileID:      stringPtr("file-removed"),
			KBFileID:          stringPtr("file-removed"),
			DisplayName:       stringPtr("removed.pdf"),
			Status:            kbSourceStatusRemoved,
		}))
	expectEmptySourceJobRuns(tenantMock, 92)
	tenantMock.ExpectQuery("SELECT id, model_id, source_type, source_file_id").
		WithArgs(int64(92)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "model_id", "source_type", "source_file_id", "kb_file_id", "raw_volume_id", "job_status", "error", "segment_version_id", "index_version", "workflow_execution_id",
		}).AddRow(int64(12), int64(92), kbSourceTypeCatalogFile, "file-removed", "file-removed", int64(12), kbSourceJobSucceeded, nil, nil, int64(1), "exec-removed"))
	expectNoKnowledgeBaseRawVolumeFiles(tenantMock, 92)

	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)

	resp, err := svc.ListSources(ctx, ListSemanticModelSourcesParams{ModelID: 92})
	if err != nil {
		t.Fatalf("ListSources: %v", err)
	}
	if resp.Total != 0 || len(resp.Items) != 0 {
		t.Fatalf("ListSources response = %+v, want no visible removed source", resp)
	}
	if resp.LegacyBackfillRequired {
		t.Fatalf("legacy backfill required = true, want false for removed source identity")
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceListSourcesDoesNotBackfillSemanticModelTableSource(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/89":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":     89,
				"name":   "legacy-kb",
				"files":  map[string]any{"file_ids": []string{}},
				"tables": []map[string]any{{"db_name": "external_db", "table_names": []string{"orders"}}},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/ws-1/tables/3001":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"table":    map[string]any{"id": 3001, "name": "orders", "database_id": 301, "catalog_id": 201, "updated_at": 1782705900},
				"database": map[string]any{"id": 301, "name": "external_db", "catalog_id": 201},
				"catalog":  map[string]any{"id": 201, "name": "external_catalog"},
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()

	svc := newSemanticModelTestService(t, server.URL, systemClient)
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	sourceColumns := []string{
		"source_id", "model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "source_type",
		"source_file_id", "source_table_id", "kb_file_id", "kb_table_id", "display_name", "source_path",
		"db_name", "table_name", "status", "error", "enabled", "expires_at", "tags", "force_enabled_after_expiry", "segment_version_id", "index_version",
		"size_bytes", "row_count", "updated_at",
	}
	tenantMock.ExpectQuery("SELECT .*source_id.*FROM knowledge_base_sources").
		WithArgs(int64(89)).
		WillReturnRows(sqlmock.NewRows(sourceColumns))
	expectEmptySourceJobRuns(tenantMock, 89)
	tenantMock.ExpectQuery("SELECT id, model_id, source_type, source_file_id").
		WithArgs(int64(89)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "model_id", "source_type", "source_file_id", "kb_file_id", "raw_volume_id", "job_status", "error", "segment_version_id", "index_version", "workflow_execution_id",
		}))
	tenantMock.ExpectQuery("SELECT t\\.table_id, t\\.database_id, t\\.catalog_id").
		WithArgs("external_db", "orders").
		WillReturnRows(sqlmock.NewRows([]string{
			"table_id", "database_id", "catalog_id", "database_name", "catalog_name",
		}).AddRow(int64(3001), int64(301), int64(201), "external_db", "external_catalog"))
	expectNoKnowledgeBaseRawVolumeFiles(tenantMock, 89)

	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)

	resp, err := svc.ListSources(ctx, ListSemanticModelSourcesParams{ModelID: 89})
	if err != nil {
		t.Fatalf("ListSources: %v", err)
	}
	if resp.Total != 1 || len(resp.Items) != 1 {
		t.Fatalf("ListSources response = %+v", resp)
	}
	item := resp.Items[0]
	if item.SourceID != "" || item.SourceType != SemanticModelSourceTypeTable || item.ResourceID != "3001" || item.GovernanceStatus != SemanticModelSourceGovernanceLegacyUnbound {
		t.Fatalf("table candidate = %+v", item)
	}
	if !resp.LegacyBackfillRequired {
		t.Fatalf("legacy backfill required = false, want true for semantic model table candidate")
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceListSourcesDoesNotResolveSemanticModelTableBackfill(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/workspaces/ws-1/semantic-models/90" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":     90,
			"name":   "legacy-kb",
			"files":  map[string]any{"file_ids": []string{}},
			"tables": []map[string]any{{"db_name": "external_db", "table_names": []string{"orders"}}},
		})
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()

	svc := newSemanticModelTestService(t, server.URL, systemClient)
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	tenantMock.ExpectQuery("SELECT .*source_id.*FROM knowledge_base_sources").
		WithArgs(int64(90)).
		WillReturnRows(sqlmock.NewRows([]string{
			"source_id", "model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "source_type",
			"source_file_id", "source_table_id", "kb_file_id", "kb_table_id", "display_name", "source_path",
			"db_name", "table_name", "status", "error", "enabled", "expires_at", "tags", "force_enabled_after_expiry", "segment_version_id", "index_version",
			"size_bytes", "row_count", "updated_at",
		}))
	expectEmptySourceJobRuns(tenantMock, 90)
	tenantMock.ExpectQuery("SELECT id, model_id, source_type, source_file_id").
		WithArgs(int64(90)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "model_id", "source_type", "source_file_id", "kb_file_id", "raw_volume_id", "job_status", "error", "segment_version_id", "index_version", "workflow_execution_id",
		}))
	tenantMock.ExpectQuery("SELECT t\\.table_id, t\\.database_id, t\\.catalog_id").
		WithArgs("external_db", "orders").
		WillReturnRows(sqlmock.NewRows([]string{
			"table_id", "database_id", "catalog_id", "database_name", "catalog_name",
		}))
	expectNoKnowledgeBaseRawVolumeFiles(tenantMock, 90)

	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)

	resp, err := svc.ListSources(ctx, ListSemanticModelSourcesParams{ModelID: 90})
	if err != nil {
		t.Fatalf("ListSources: %v", err)
	}
	if resp.Total != 0 || len(resp.Items) != 0 {
		t.Fatalf("read-only ListSources should not fabricate semantic model table source rows: %+v", resp)
	}
	if resp.LegacyBackfillRequired {
		t.Fatalf("legacy backfill required = true, want false for missing catalog table without candidate")
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceListSourcesReturnsLineageCandidateRows(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/workspaces/ws-1/semantic-models/20006" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		requireSemanticModelExecutionHeaders(t, r)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":   20006,
			"name": "l4",
			"files": map[string]any{
				"file_ids":           []string{},
				"parents":            []string{},
				"vector_table":       "kb_drawing_mqunm4k8nr7o",
				"image_vector_table": "kb_drawing_mqunm4k8nr7o_img",
			},
			"tables": []map[string]any{},
		})
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()

	svc := newSemanticModelTestService(t, server.URL, systemClient)
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	tenantMock.ExpectQuery("SELECT .*source_id.*FROM knowledge_base_sources").
		WithArgs(int64(20006)).
		WillReturnRows(sqlmock.NewRows([]string{
			"source_id", "model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "source_type",
			"source_file_id", "source_table_id", "kb_file_id", "kb_table_id", "display_name", "source_path",
			"db_name", "table_name", "status", "error", "enabled", "expires_at", "tags", "force_enabled_after_expiry", "segment_version_id", "index_version",
			"size_bytes", "row_count", "updated_at",
		}))
	expectEmptySourceJobRuns(tenantMock, 20006)
	expectEmptyLegacySourceJobs(tenantMock, 20006)
	expectNoKnowledgeBaseRawVolumeFiles(tenantMock, 20006)
	tenantMock.ExpectQuery("SELECT DISTINCT COALESCE\\(pm\\.source_file_id, root\\.asset_ref\\) AS file_id").
		WithArgs("kb_drawing_mqunm4k8nr7o", "kb_drawing_mqunm4k8nr7o_img", int64(20006), kbLegacyBackfillBatchSize).
		WillReturnRows(sqlmock.NewRows([]string{"file_id"}).AddRow("source-file-lineage"))
	tenantMock.ExpectQuery("SELECT COALESCE\\(CASE WHEN v\\.catalog_id").
		WithArgs("source-file-lineage").
		WillReturnRows(sqlmock.NewRows([]string{
			"catalog_id", "database_id", "volume_id", "size", "updated_at", "catalog_name", "database_name", "volume_name", "file_path", "file_name",
		}).AddRow(int64(301), int64(401), int64(501), int64(194162), int64(1782875758), "fangyuan", "test", "pdf", "20C114774.pdf", "20C114774.pdf"))
	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)

	resp, err := svc.ListSources(ctx, ListSemanticModelSourcesParams{ModelID: 20006})
	if err != nil {
		t.Fatalf("ListSources: %v", err)
	}
	if resp.Total != 1 || len(resp.Items) != 1 {
		t.Fatalf("ListSources should expose lineage candidate row: %+v", resp)
	}
	item := resp.Items[0]
	if item.SourceID != "" || item.ResourceID != "source-file-lineage" || item.GovernanceStatus != SemanticModelSourceGovernanceLegacyUnbound {
		t.Fatalf("lineage candidate = %+v", item)
	}
	if item.LegacyOrigin == nil || *item.LegacyOrigin != SemanticModelSourceLegacyOriginLineage {
		t.Fatalf("lineage candidate origin = %v", item.LegacyOrigin)
	}
	if !resp.LegacyBackfillRequired {
		t.Fatalf("legacy backfill required = false, want true")
	}

	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceListSourcesReturnsGetError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/workspaces/ws-1/semantic-models/7" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 3, "message": "not found"})
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()

	svc := newSemanticModelTestService(t, server.URL, systemClient)
	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")

	_, err = svc.ListSources(ctx, ListSemanticModelSourcesParams{ModelID: 7})
	if err == nil {
		t.Fatal("ListSources error is nil")
	}
	var serviceErr *ServiceError
	if errors.As(err, &serviceErr) {
		t.Fatalf("ListSources should return original SDK error, got service error: %+v", serviceErr)
	}
}

func TestSemanticModelSourceServiceErrorsAreLocalized(t *testing.T) {
	cases := []struct {
		name    string
		err     error
		code    ServiceErrorCode
		wantMsg string
	}{
		{
			name:    "source not found",
			err:     knowledgeBaseSourceNotFoundError(),
			code:    ErrCodeNotFound,
			wantMsg: "知识库来源不存在",
		},
		{
			name:    "data domain not found",
			err:     knowledgeBaseDataDomainNotFoundError(),
			code:    ErrCodeNotFound,
			wantMsg: "知识库数据域不存在",
		},
		{
			name:    "files invalid",
			err:     semanticModelFilesInvalidError(),
			code:    ErrCodeBadRequest,
			wantMsg: "语义模型文件无效",
		},
		{
			name:    "tables invalid",
			err:     semanticModelTablesInvalidError(),
			code:    ErrCodeBadRequest,
			wantMsg: "语义模型表配置无效",
		},
		{
			name:    "segment embedding response invalid",
			err:     segmentEmbeddingResponseInvalidError(),
			code:    ErrCodeInternal,
			wantMsg: "分段 embedding 响应无效",
		},
		{
			name:    "image embedding response invalid",
			err:     imageEmbeddingResponseInvalidError(),
			code:    ErrCodeInternal,
			wantMsg: "图片 embedding 响应无效",
		},
	}
	ctx := i18n.WithLocale(context.Background(), i18n.LocaleZhCN)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var serviceErr *ServiceError
			if !errors.As(tc.err, &serviceErr) {
				t.Fatalf("error = %T, want ServiceError", tc.err)
			}
			if serviceErr.Code != tc.code {
				t.Fatalf("code = %s, want %s", serviceErr.Code, tc.code)
			}
			if serviceErr.Msg != "" {
				t.Fatalf("Msg = %q, want empty raw message", serviceErr.Msg)
			}
			got, ok := i18n.Message(ctx, tc.err)
			if !ok {
				t.Fatalf("error is not localized: %v", tc.err)
			}
			if got != tc.wantMsg {
				t.Fatalf("localized message = %q, want %q", got, tc.wantMsg)
			}
		})
	}
}

func TestSemanticModelServiceListSourcesFiltersMissingCatalogSourceWithoutWriteSideEffects(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/workspaces/ws-1/semantic-models/7" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":    7,
			"name":  "kb",
			"files": map[string]any{"file_ids": []string{"missing-kb-file"}},
		})
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()

	svc := newSemanticModelTestService(t, server.URL, systemClient)
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	tenantMock.MatchExpectationsInOrder(false)
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	tenantMock.ExpectQuery("SELECT .*source_id.*FROM knowledge_base_sources").
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{
			"source_id", "model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "source_type",
			"source_file_id", "source_table_id", "kb_file_id", "kb_table_id", "display_name", "source_path",
			"db_name", "table_name", "status", "error", "enabled", "expires_at", "tags", "force_enabled_after_expiry", "segment_version_id", "index_version",
			"size_bytes", "row_count", "updated_at",
		}).AddRow("source-file-1", int64(7), int64(3), int64(11), int64(12), int64(13), kbSourceTypeCatalogFile, "source-file", nil, "missing-kb-file", nil, "doc.pdf", nil, nil, nil, kbSourceStatusSucceeded, nil, true, nil, nil, false, nil, nil, nil, nil, int64(100)))
	// Authoritative metadata file id is source_file_id; missing location fails the row.
	tenantMock.ExpectQuery("SELECT .*f.size, UNIX_TIMESTAMP").
		WithArgs("source-file").
		WillReturnRows(sqlmock.NewRows([]string{
			"file_id", "catalog_id", "database_id", "volume_id", "size", "updated_at", "catalog_name", "database_name", "volume_name", "file_path", "file_name",
		}))
	tenantMock.ExpectQuery("SELECT .*job_id.*source_id.*model_id.*job_type.*job_status.*idempotency_key").
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{
			"job_id", "source_id", "model_id", "job_type", "job_status", "idempotency_key", "operation_id",
			"workflow_execution_id", "runtime_actor_moi_user_id", "runtime_effective_role_id", "runtime_is_workspace_owner", "source_file_id", "kb_file_id", "source_table_id", "kb_table_id",
			"retry_count", "next_retry_at", "error", "created_at", "updated_at",
		}))
	expectEmptyLegacySourceJobs(tenantMock, 7)
	expectNoKnowledgeBaseRawVolumeFiles(tenantMock, 7)

	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)

	resp, err := svc.ListSources(ctx, ListSemanticModelSourcesParams{ModelID: 7})
	if err != nil {
		t.Fatalf("ListSources: %v", err)
	}
	if resp.Total != 1 || len(resp.Items) != 1 {
		t.Fatalf("missing catalog source should stay visible as failed without repair write: %+v", resp)
	}
	source := resp.Items[0]
	if source.RowID != "source-file-1" || source.IngestStatus == nil || *source.IngestStatus != kbSourceStatusFailed {
		t.Fatalf("missing catalog source status = %+v", source)
	}
	if source.Error == nil || !strings.Contains(*source.Error, "catalog file source-file not found") {
		t.Fatalf("missing catalog source error = %v", source.Error)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceListSourcesKeepsPendingLocalFileWhenVolumeMetadataNotReady(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/workspaces/ws-1/semantic-models/7" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":    7,
			"name":  "kb",
			"files": map[string]any{"file_ids": []string{"pending-kb-file"}},
		})
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()

	svc := newSemanticModelTestService(t, server.URL, systemClient)
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	tenantMock.MatchExpectationsInOrder(false)
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	tenantMock.ExpectQuery("SELECT .*source_id.*FROM knowledge_base_sources").
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{
			"source_id", "model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "source_type",
			"source_file_id", "source_table_id", "kb_file_id", "kb_table_id", "display_name", "source_path",
			"db_name", "table_name", "status", "error", "enabled", "expires_at", "tags", "force_enabled_after_expiry", "segment_version_id", "index_version",
			"size_bytes", "row_count", "updated_at",
		}).AddRow("source-file-1", int64(7), int64(3), int64(11), int64(12), int64(13), kbSourceTypeLocalFile, "pending-source-file", nil, "pending-kb-file", nil, "pending.pdf", nil, nil, nil, kbSourceStatusPending, nil, true, nil, nil, false, nil, nil, nil, nil, int64(100)))
	// Pending local file still has source_file_id; list looks it up and keeps pending on miss.
	tenantMock.ExpectQuery("SELECT .*f.size, UNIX_TIMESTAMP").
		WithArgs("pending-source-file").
		WillReturnRows(sqlmock.NewRows([]string{
			"catalog_id", "database_id", "volume_id", "size", "updated_at", "catalog_name", "database_name", "volume_name", "file_path", "file_name",
		}))
	tenantMock.ExpectQuery("SELECT .*job_id.*source_id.*model_id.*job_type.*job_status.*idempotency_key").
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{
			"job_id", "source_id", "model_id", "job_type", "job_status", "idempotency_key", "operation_id",
			"workflow_execution_id", "runtime_actor_moi_user_id", "runtime_effective_role_id", "runtime_is_workspace_owner", "source_file_id", "kb_file_id", "source_table_id", "kb_table_id",
			"retry_count", "next_retry_at", "error", "created_at", "updated_at",
		}))
	expectEmptyLegacySourceJobs(tenantMock, 7)
	expectNoKnowledgeBaseRawVolumeFiles(tenantMock, 7)

	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)

	resp, err := svc.ListSources(ctx, ListSemanticModelSourcesParams{ModelID: 7})
	if err != nil {
		t.Fatalf("ListSources: %v", err)
	}
	if resp.Total != 1 || len(resp.Items) != 1 {
		t.Fatalf("pending local file source should stay visible while metadata is not ready: %+v", resp)
	}
	source := resp.Items[0]
	if source.RowID != "source-file-1" || source.ResourceID != "pending-kb-file" || source.DisplayName == nil || *source.DisplayName != "pending.pdf" {
		t.Fatalf("pending local file fallback row = %+v", source)
	}
	if len(source.Path) != 0 || source.SourcePath != nil || source.SizeBytes != nil {
		t.Fatalf("pending local file should not fabricate missing metadata: %+v", source)
	}
	if source.IngestStatus == nil || *source.IngestStatus != kbSourceStatusPending {
		t.Fatalf("pending local file status = %v", source.IngestStatus)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceListSourcesReturnsFailedSourceWithoutFileID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/workspaces/ws-1/semantic-models/7" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":    7,
			"name":  "kb",
			"files": map[string]any{"file_ids": []string{}},
		})
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()

	svc := newSemanticModelTestService(t, server.URL, systemClient)
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	sourceErr := "create local file import task bad.md: task_id is required"
	tenantMock.ExpectQuery("SELECT .*source_id.*FROM knowledge_base_sources").
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{
			"source_id", "model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "source_type",
			"source_file_id", "source_table_id", "kb_file_id", "kb_table_id", "display_name", "source_path",
			"db_name", "table_name", "status", "error", "enabled", "expires_at", "tags", "force_enabled_after_expiry", "segment_version_id", "index_version",
			"size_bytes", "row_count", "updated_at",
		}).AddRow("source-file-1", int64(7), int64(3), int64(11), int64(12), int64(13), kbSourceTypeLocalFile, nil, nil, nil, nil, "bad.md", nil, nil, nil, kbSourceStatusFailed, sourceErr, true, nil, nil, false, nil, nil, nil, nil, int64(100)))
	tenantMock.ExpectQuery("SELECT .*job_id.*source_id.*model_id.*job_type.*job_status.*idempotency_key").
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{
			"job_id", "source_id", "model_id", "job_type", "job_status", "idempotency_key", "operation_id",
			"workflow_execution_id", "runtime_actor_moi_user_id", "runtime_effective_role_id", "runtime_is_workspace_owner", "source_file_id", "kb_file_id", "source_table_id", "kb_table_id",
			"retry_count", "next_retry_at", "error", "created_at", "updated_at",
		}))
	expectEmptyLegacySourceJobs(tenantMock, 7)
	expectNoKnowledgeBaseRawVolumeFiles(tenantMock, 7)

	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)

	resp, err := svc.ListSources(ctx, ListSemanticModelSourcesParams{ModelID: 7})
	if err != nil {
		t.Fatalf("ListSources: %v", err)
	}
	if resp.Total != 1 || len(resp.Items) != 1 {
		t.Fatalf("failed source without file id should stay visible: %+v", resp)
	}
	source := resp.Items[0]
	if source.RowID != "source-file-1" || source.ResourceID != "" || source.IngestStatus == nil || *source.IngestStatus != kbSourceStatusFailed {
		t.Fatalf("failed source without file id row = %+v", source)
	}
	if source.Error == nil || *source.Error != sourceErr {
		t.Fatalf("failed source without file id error = %v", source.Error)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceListSourcesPaginatesVisibleSources(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/workspaces/ws-1/semantic-models/7" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":   7,
			"name": "kb",
		})
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()

	svc := newSemanticModelTestService(t, server.URL, systemClient)
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	tenantMock.ExpectQuery("SELECT .*source_id.*FROM knowledge_base_sources").
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{
			"source_id", "model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "source_type",
			"source_file_id", "source_table_id", "kb_file_id", "kb_table_id", "display_name", "source_path",
			"db_name", "table_name", "status", "error", "enabled", "expires_at", "tags", "force_enabled_after_expiry", "segment_version_id", "index_version",
			"size_bytes", "row_count", "updated_at",
		}).
			AddRow("source-table-1", int64(7), int64(3), int64(11), int64(12), int64(13), kbSourceTypeCatalogTable, nil, nil, nil, nil, "orders", nil, "db", "orders", kbSourceStatusPending, nil, true, nil, nil, false, nil, nil, nil, nil, int64(100)).
			AddRow("source-table-2", int64(7), int64(3), int64(11), int64(12), int64(13), kbSourceTypeCatalogTable, nil, nil, nil, nil, "customers", nil, "db", "customers", kbSourceStatusPending, nil, true, nil, nil, false, nil, nil, nil, nil, int64(101)))
	tenantMock.ExpectQuery("SELECT .*job_id.*source_id.*model_id.*job_type.*job_status.*idempotency_key").
		WithArgs(int64(7), "source-table-2").
		WillReturnRows(sqlmock.NewRows([]string{
			"job_id", "source_id", "model_id", "job_type", "job_status", "idempotency_key", "operation_id",
			"workflow_execution_id", "runtime_actor_moi_user_id", "runtime_effective_role_id", "runtime_is_workspace_owner", "source_file_id", "kb_file_id", "source_table_id", "kb_table_id",
			"retry_count", "next_retry_at", "error", "created_at", "updated_at",
		}))
	expectEmptyLegacySourceJobs(tenantMock, 7)
	expectNoKnowledgeBaseRawVolumeFiles(tenantMock, 7)

	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)

	resp, err := svc.ListSources(ctx, ListSemanticModelSourcesParams{ModelID: 7, Page: 2, PageSize: 1})
	if err != nil {
		t.Fatalf("ListSources: %v", err)
	}
	if resp.Total != 2 || resp.Page != 2 || resp.PageSize != 1 || len(resp.Items) != 1 {
		t.Fatalf("paginated response = %+v", resp)
	}
	if resp.Items[0].RowID != "source-table-2" {
		t.Fatalf("page 2 source = %+v", resp.Items[0])
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestListSourcesPageBoundsLargePage(t *testing.T) {
	start, end := listSourcesPageBounds(10, int(^uint(0)>>1), 10)
	if start != 10 || end != 10 {
		t.Fatalf("page bounds = (%d, %d), want (10, 10)", start, end)
	}
}

func TestSemanticModelServiceListSourcesPaginationKeepsLegacyBackfillSignalGlobal(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/workspaces/ws-1/semantic-models/7" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":   7,
			"name": "kb",
		})
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()

	svc := newSemanticModelTestService(t, server.URL, systemClient)
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	tenantMock.ExpectQuery("SELECT .*source_id.*FROM knowledge_base_sources").
		WithArgs(int64(7)).
		WillReturnRows(knowledgeBaseSourceRecordRows(
			KnowledgeBaseSourceRecord{
				SourceID:          "source-file-1",
				ModelID:           7,
				CatalogID:         3,
				DatabaseID:        11,
				RawVolumeID:       12,
				ProcessedVolumeID: 13,
				SourceType:        kbSourceTypeCatalogFile,
				SourceFileID:      stringPtr("catalog-file-1"),
				KBFileID:          stringPtr("kb-file-1"),
				DisplayName:       stringPtr("legacy.pdf"),
				Status:            kbSourceStatusSucceeded,
				Enabled:           boolPtr(true),
			},
			KnowledgeBaseSourceRecord{
				SourceID:          "source-table-1",
				ModelID:           7,
				CatalogID:         3,
				DatabaseID:        11,
				RawVolumeID:       12,
				ProcessedVolumeID: 13,
				SourceType:        kbSourceTypeCatalogTable,
				DisplayName:       stringPtr("orders"),
				DBName:            stringPtr("db"),
				TableName:         stringPtr("orders"),
				Status:            kbSourceStatusPending,
				Enabled:           boolPtr(true),
			},
		))
	tenantMock.ExpectQuery("SELECT .*job_id.*source_id.*model_id.*job_type.*job_status.*idempotency_key").
		WithArgs(int64(7), "source-table-1").
		WillReturnRows(knowledgeBaseSourceJobRunRows())
	tenantMock.ExpectQuery("SELECT id, model_id, source_type, source_file_id").
		WithArgs(int64(7)).
		WillReturnRows(knowledgeBaseSourceJobRows().
			AddRow(int64(1), int64(7), kbSourceTypeCatalogFile, "catalog-file-1", "kb-file-1", int64(12), kbSourceJobSucceeded, nil, nil, int64(1), "legacy-exec-1"))
	expectNoKnowledgeBaseRawVolumeFiles(tenantMock, 7)
	tenantMock.ExpectQuery("SELECT CASE WHEN EXISTS.*FROM knowledge_base_source_jobs legacy.*LIMIT 1").
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(0))

	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)

	resp, err := svc.ListSources(ctx, ListSemanticModelSourcesParams{ModelID: 7, Page: 2, PageSize: 1})
	if err != nil {
		t.Fatalf("ListSources: %v", err)
	}
	if resp.Total != 2 || resp.Page != 2 || resp.PageSize != 1 || len(resp.Items) != 1 {
		t.Fatalf("paginated response = %+v", resp)
	}
	if resp.Items[0].RowID != "source-table-1" {
		t.Fatalf("page 2 source = %+v", resp.Items[0])
	}
	if resp.LegacyBackfillRequired {
		t.Fatalf("legacy backfill required = true, want false when legacy source and job run already exist")
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceGetSourceDocumentReturnsGovernanceAndSegments(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/workspaces/ws-1/semantic-models/7" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		requireSemanticModelExecutionHeaders(t, r)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":    7,
			"name":  "kb",
			"files": map[string]any{"file_ids": []string{"kb-file"}},
		})
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()

	svc := newSemanticModelTestService(t, server.URL, systemClient)
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	enabled := false
	expiresAt := time.Now().Add(-time.Hour).Unix()
	tenantMock.ExpectQuery("SELECT .*source_id.*FROM knowledge_base_sources").
		WithArgs(int64(7), "source-file-1").
		WillReturnRows(sqlmock.NewRows([]string{
			"source_id", "model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "source_type",
			"source_file_id", "source_table_id", "kb_file_id", "kb_table_id", "display_name", "source_path",
			"db_name", "table_name", "status", "error", "enabled", "expires_at", "tags", "force_enabled_after_expiry", "segment_version_id", "index_version",
		}).AddRow("source-file-1", int64(7), int64(3), int64(11), int64(12), int64(13), kbSourceTypeCatalogFile, "source-file", nil, "kb-file", nil, "doc.pdf", `["raw","doc.pdf"]`, nil, nil, kbSourceStatusSucceeded, nil, enabled, expiresAt, `["finance"]`, true, "seg-v1", int64(9)))
	// Detail uses the same single authoritative source_file_id + recorded volume as list.
	expectCatalogFileMetadataBatchAtVolume(tenantMock, 12, []string{"source-file"}, "doc.pdf")
	tenantMock.ExpectQuery("SELECT version_id, index_version").
		WithArgs(int64(7), "source-file-1").
		WillReturnRows(sqlmock.NewRows([]string{
			"version_id", "index_version", "base_version_id", "base_index_version", "status", "source", "chunk_count", "enabled_chunk_count", "created_by", "updated_by", "created_at", "updated_at",
		}).AddRow("seg-v1", int64(9), nil, nil, kbSegmentStatusCommitted, kbSegmentSourceInitial, int64(1), int64(1), "user-1", "user-1", int64(100), int64(101)))
	tenantMock.ExpectQuery("SELECT s\\.segment_id").
		WithArgs(int64(7), "source-file-1", "seg-v1").
		WillReturnRows(sqlmock.NewRows([]string{
			"segment_id", "version_id", "model_id", "source_id", "kb_file_id", "index_version", "level", "chunk_index", "chunk_id",
			"content", "ocr_text", "image_description", "image_file_id", "page_image_file_id", "bbox", "word_count", "recall_count", "enabled", "metadata", "created_at", "updated_at",
		}).AddRow("seg-1", "seg-v1", int64(7), "source-file-1", "kb-file", int64(9), "chunk", int64(0), nil, "first chunk", nil, nil, "image-file-1", "page-image-1", `{"x":1}`, int64(11), int64(3), true, `{"segment_type":"image","volume_id":13,"raw_file_id":"kb-file","layout_file_id":"layout-file"}`, int64(100), int64(101)))

	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)

	doc, err := svc.GetSourceDocument(ctx, GetSemanticModelSourceDocumentParams{ModelID: 7, SourceID: "source-file-1"})
	if err != nil {
		t.Fatalf("GetSourceDocument: %v", err)
	}
	if doc.Source.DisplayName == nil || *doc.Source.DisplayName != "doc.pdf" {
		t.Fatalf("display_name = %+v, want doc.pdf", doc.Source.DisplayName)
	}
	if !doc.Preview.Available || doc.Preview.Content != nil {
		t.Fatalf("preview should only expose availability: %+v", doc.Preview)
	}
	if !doc.SegmentStatus.Available || doc.SegmentStatus.Total != 1 {
		t.Fatalf("segment status should reflect current chunks: %+v", doc.SegmentStatus)
	}
	if doc.Source.IndexVersion == nil || *doc.Source.IndexVersion != 9 || doc.FileInfo.IndexVersion == nil || *doc.FileInfo.IndexVersion != 9 {
		t.Fatalf("index version not returned: %+v", doc)
	}
	if !doc.Source.Expired || doc.Source.EffectiveEnabled || !doc.Source.ForceEnabled || !sameStringSet(doc.Source.Tags, []string{"finance"}) {
		t.Fatalf("source governance = %+v", doc.Source)
	}
	if len(doc.SegmentVersions) != 1 || doc.SegmentVersions[0].VersionID != "seg-v1" || !doc.SegmentVersions[0].Current {
		t.Fatalf("segment versions = %+v", doc.SegmentVersions)
	}
	if doc.CurrentSegmentVersionID == nil || *doc.CurrentSegmentVersionID != "seg-v1" || doc.CurrentIndexVersion == nil || *doc.CurrentIndexVersion != 9 {
		t.Fatalf("current version = %+v/%+v", doc.CurrentSegmentVersionID, doc.CurrentIndexVersion)
	}
	if len(doc.Segments) != 1 || doc.Segments[0].SegmentID != "seg-1" || doc.Segments[0].RecallCount != 3 || doc.Segments[0].WordCount != 11 {
		t.Fatalf("segments = %+v", doc.Segments)
	}
	if doc.Segments[0].SegmentType != "image" || doc.Segments[0].ImageFileID == nil || *doc.Segments[0].ImageFileID != "image-file-1" {
		t.Fatalf("image segment API contract = %+v", doc.Segments[0])
	}
	if string(doc.Segments[0].Metadata) != `{"volume_id":13}` {
		t.Fatalf("segment metadata = %s, want volume_id only", string(doc.Segments[0].Metadata))
	}
	payload, err := json.Marshal(doc.Segments[0])
	if err != nil {
		t.Fatalf("marshal document segment: %v", err)
	}
	for _, unexpected := range []string{`"model_id":7`, `"source_id":"source-file-1"`, `"kb_file_id":"kb-file"`, `"index_version":9`, `"bbox"`, `"raw_file_id"`, `"layout_file_id"`} {
		if strings.Contains(string(payload), unexpected) {
			t.Fatalf("document segment payload contains %s: %s", unexpected, string(payload))
		}
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceGetSourceDocumentEnrichesDisplayNameFromCatalogMetadata(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/workspaces/ws-1/semantic-models/7" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		requireSemanticModelExecutionHeaders(t, r)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":    7,
			"name":  "kb",
			"files": map[string]any{"file_ids": []string{"kb-file"}},
		})
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()

	svc := newSemanticModelTestService(t, server.URL, systemClient)
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	enabled := true
	// Stored display_name fell back to file id; detail path must recover real file name at recorded volume.
	tenantMock.ExpectQuery("SELECT .*source_id.*FROM knowledge_base_sources").
		WithArgs(int64(7), "source-file-1").
		WillReturnRows(sqlmock.NewRows([]string{
			"source_id", "model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "source_type",
			"source_file_id", "source_table_id", "kb_file_id", "kb_table_id", "display_name", "source_path",
			"db_name", "table_name", "status", "error", "enabled", "expires_at", "tags", "force_enabled_after_expiry", "segment_version_id", "index_version",
		}).AddRow("source-file-1", int64(7), int64(3), int64(11), int64(12), int64(13), kbSourceTypeCatalogFile, "source-file", nil, "kb-file", nil, "kb-file", nil, nil, nil, kbSourceStatusSucceeded, nil, enabled, nil, `[]`, false, "seg-v1", int64(1)))
	expectCatalogFileMetadataBatchAtVolume(tenantMock, 12, []string{"source-file"}, "MatrixOne_User_Guide.pdf")
	tenantMock.ExpectQuery("SELECT version_id, index_version").
		WithArgs(int64(7), "source-file-1").
		WillReturnRows(sqlmock.NewRows([]string{
			"version_id", "index_version", "base_version_id", "base_index_version", "status", "source", "chunk_count", "enabled_chunk_count", "created_by", "updated_by", "created_at", "updated_at",
		}).AddRow("seg-v1", int64(1), nil, nil, kbSegmentStatusCommitted, kbSegmentSourceInitial, int64(0), int64(0), "user-1", "user-1", int64(100), int64(101)))
	tenantMock.ExpectQuery("SELECT s\\.segment_id").
		WithArgs(int64(7), "source-file-1", "seg-v1").
		WillReturnRows(sqlmock.NewRows([]string{
			"segment_id", "version_id", "model_id", "source_id", "kb_file_id", "index_version", "level", "chunk_index", "chunk_id",
			"content", "ocr_text", "image_description", "image_file_id", "page_image_file_id", "bbox", "word_count", "recall_count", "enabled", "metadata", "created_at", "updated_at",
		}))

	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)

	doc, err := svc.GetSourceDocument(ctx, GetSemanticModelSourceDocumentParams{ModelID: 7, SourceID: "source-file-1"})
	if err != nil {
		t.Fatalf("GetSourceDocument: %v", err)
	}
	if doc.Source.DisplayName == nil || *doc.Source.DisplayName != "MatrixOne_User_Guide.pdf" {
		t.Fatalf("display_name = %+v, want MatrixOne_User_Guide.pdf", doc.Source.DisplayName)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceGetSourceDocumentKeepsReadOnlySourceBoundaries(t *testing.T) {
	tests := []struct {
		name       string
		sourceID   string
		sourceRows *sqlmock.Rows
		wantCode   ServiceErrorCode
	}{
		{
			name:       "missing source",
			sourceID:   "missing-source",
			sourceRows: sqlmock.NewRows([]string{"source_id"}),
			wantCode:   ErrCodeNotFound,
		},
		{
			name:       "source owned by another model",
			sourceID:   "source-from-model-8",
			sourceRows: sqlmock.NewRows([]string{"source_id"}),
			wantCode:   ErrCodeNotFound,
		},
		{
			name:     "table source remains unsupported",
			sourceID: "source-table-1",
			sourceRows: sqlmock.NewRows([]string{
				"source_id", "model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "source_type",
				"source_file_id", "source_table_id", "kb_file_id", "kb_table_id", "display_name", "source_path",
				"db_name", "table_name", "status", "error", "enabled", "expires_at", "tags", "force_enabled_after_expiry", "segment_version_id", "index_version",
				"size_bytes", "row_count", "created_by", "updated_by", "updated_at",
			}).AddRow("source-table-1", int64(7), int64(3), int64(11), int64(12), int64(13), kbSourceTypeCatalogTable, nil, int64(42), nil, int64(84), "orders", nil, "db", "orders", kbSourceStatusSucceeded, nil, true, nil, nil, false, nil, nil, nil, int64(10), "user-1", "user-1", int64(100)),
			wantCode: ErrCodeBadRequest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.URL.Path != "/api/v1/workspaces/ws-1/semantic-models/7" {
					t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{
					"id":    7,
					"name":  "kb",
					"files": map[string]any{"file_ids": []string{"legacy-file"}},
				})
			}))
			defer server.Close()

			systemClient, err := moi.New(server.URL, "system-key")
			if err != nil {
				t.Fatalf("moi.New: %v", err)
			}
			defer systemClient.Close()

			svc := newSemanticModelTestService(t, server.URL, systemClient)
			tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
			if err != nil {
				t.Fatalf("tenant sqlmock: %v", err)
			}
			defer tenantSQL.Close()
			tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
			if err != nil {
				t.Fatalf("open tenant gorm: %v", err)
			}
			tenantMock.ExpectQuery("SELECT .*source_id.*FROM knowledge_base_sources").
				WithArgs(int64(7), tc.sourceID).
				WillReturnRows(tc.sourceRows)

			ctx := ctxutil.WithUID(context.Background(), "user-1")
			ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
			ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
			ctx = ctxutil.WithTenantDB(ctx, tenantDB)

			_, err = svc.GetSourceDocument(ctx, GetSemanticModelSourceDocumentParams{ModelID: 7, SourceID: tc.sourceID})
			var serviceErr *ServiceError
			if !errors.As(err, &serviceErr) || serviceErr.Code != tc.wantCode {
				t.Fatalf("GetSourceDocument error = %v, want code %s", err, tc.wantCode)
			}
			if err := tenantMock.ExpectationsWereMet(); err != nil {
				t.Fatalf("tenant sql expectations: %v", err)
			}
		})
	}
}

func TestSemanticModelServiceGetSourceDocumentDoesNotReturnEnglishEmptyReasons(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/workspaces/ws-1/semantic-models/7" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":    7,
			"name":  "kb",
			"files": map[string]any{"file_ids": []string{"kb-file"}},
		})
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()

	svc := newSemanticModelTestService(t, server.URL, systemClient)
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	tenantMock.ExpectQuery("SELECT .*source_id.*FROM knowledge_base_sources").
		WithArgs(int64(7), "source-file-1").
		WillReturnRows(sqlmock.NewRows([]string{
			"source_id", "model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "source_type",
			"source_file_id", "source_table_id", "kb_file_id", "kb_table_id", "display_name", "source_path",
			"db_name", "table_name", "status", "error", "enabled", "expires_at", "tags", "force_enabled_after_expiry", "segment_version_id", "index_version",
			"size_bytes", "row_count", "created_by", "updated_by", "updated_at",
		}).AddRow("source-file-1", int64(7), int64(3), int64(11), int64(12), int64(13), kbSourceTypeCatalogFile, "source-file", nil, "kb-file", nil, "doc.pdf", `["raw","doc.pdf"]`, nil, nil, kbSourceStatusSucceeded, nil, true, nil, `["finance"]`, false, nil, nil, int64(123), nil, "user-1", "user-1", int64(100)))
	expectCatalogFileMetadataBatchAtVolume(tenantMock, 12, []string{"source-file"}, "doc.pdf")
	tenantMock.ExpectQuery("SELECT version_id, index_version").
		WithArgs(int64(7), "source-file-1").
		WillReturnRows(sqlmock.NewRows([]string{
			"version_id", "index_version", "base_version_id", "base_index_version", "status", "source", "chunk_count", "enabled_chunk_count", "created_by", "updated_by", "created_at", "updated_at",
		}))

	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)

	doc, err := svc.GetSourceDocument(ctx, GetSemanticModelSourceDocumentParams{ModelID: 7, SourceID: "source-file-1"})
	if err != nil {
		t.Fatalf("GetSourceDocument: %v", err)
	}
	if doc.Preview.Available || doc.Preview.Content != nil || doc.Preview.Reason != nil {
		t.Fatalf("unavailable preview should not include display reason: %+v", doc.Preview)
	}
	if doc.SegmentStatus.Available || doc.SegmentStatus.Total != 0 || doc.SegmentStatus.Reason != nil {
		t.Fatalf("unavailable segment status should not include display reason: %+v", doc.SegmentStatus)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelSegmentMaterializerCreatesVersionAwareVectorRows(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/workspaces/ws-1/embeddings" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		var req struct {
			Model string   `json:"model"`
			Input []string `json:"input"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode embedding request: %v", err)
		}
		if req.Model != "embed-model" || !sameStringSet(req.Input, []string{"chunk text"}) {
			t.Fatalf("embedding request = %+v", req)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"object": "list",
			"model":  "embed-model",
			"data": []map[string]any{{
				"object":    "embedding",
				"index":     0,
				"embedding": []float32{0.1, 0.2, 0.3},
			}},
		})
	}))
	defer server.Close()

	client, err := moi.New(server.URL, "caller-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer client.Close()

	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	expectVectorTableSchemaColumns(tenantMock, "kb_text_idx", "vecf32(3)")

	content := "chunk text"
	chunkIndex := int64(0)
	segment := kbSegmentRecord{
		VersionID:    "seg-v6",
		ModelID:      7,
		SourceID:     "source-file-1",
		KBFileID:     "kb-file",
		IndexVersion: 6,
		Level:        kbSegmentLevelChunk,
		ChunkIndex:   &chunkIndex,
		IdentityKey:  chunkIdentityKey(&chunkIndex, nil),
		Content:      &content,
		Enabled:      true,
	}

	svc := &semanticModelService{}
	ctx := ctxutil.WithTenantDB(context.Background(), tenantDB)
	materialized, err := svc.materializeSegments(ctx, client, "ws-1", kbVectorBinding{
		VectorTable:    "kb_text_idx",
		EmbeddingModel: "embed-model",
	}, []kbSegmentRecord{segment})
	if err != nil {
		t.Fatalf("materializeSegments: %v", err)
	}
	rows := materialized.TextRows
	if len(rows) != 1 {
		t.Fatalf("rows = %+v", rows)
	}
	if len(materialized.ImageRows) != 0 {
		t.Fatalf("image rows = %+v", materialized.ImageRows)
	}
	if rows[0].RowID != stableID("kbsegrow", "kb-file", int64(6), kbSegmentLevelChunk, "idx:0") {
		t.Fatalf("row id = %q", rows[0].RowID)
	}
	if rows[0].Content != "chunk text" || rows[0].Embedding != "[0.1,0.2,0.3]" {
		t.Fatalf("row payload = %+v", rows[0])
	}
	var meta map[string]any
	if err := json.Unmarshal([]byte(rows[0].Metadata), &meta); err != nil {
		t.Fatalf("unmarshal row metadata: %v", err)
	}
	if meta["segment_version_id"] != "seg-v6" || meta["index_version"].(float64) != 6 || meta["identity_key"] != "idx:0" {
		t.Fatalf("row metadata is not version aware: %v", meta)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelSegmentMaterializerRejectsDimensionMismatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/workspaces/ws-1/embeddings" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"object": "list",
			"model":  "embed-model",
			"data": []map[string]any{{
				"object":    "embedding",
				"index":     0,
				"embedding": []float32{0.1, 0.2, 0.3},
			}},
		})
	}))
	defer server.Close()

	client, err := moi.New(server.URL, "caller-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer client.Close()

	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	expectVectorTableSchemaColumns(tenantMock, "kb_text_idx", "vecf32(4)")

	content := "chunk text"
	chunkIndex := int64(0)
	segment := kbSegmentRecord{
		VersionID:    "seg-v6",
		ModelID:      7,
		SourceID:     "source-file-1",
		KBFileID:     "kb-file",
		IndexVersion: 6,
		Level:        kbSegmentLevelChunk,
		ChunkIndex:   &chunkIndex,
		IdentityKey:  chunkIdentityKey(&chunkIndex, nil),
		Content:      &content,
		Enabled:      true,
	}

	svc := &semanticModelService{}
	ctx := ctxutil.WithTenantDB(context.Background(), tenantDB)
	_, err = svc.materializeSegments(ctx, client, "ws-1", kbVectorBinding{
		VectorTable:    "kb_text_idx",
		EmbeddingModel: "embed-model",
	}, []kbSegmentRecord{segment})
	if err == nil || !strings.Contains(err.Error(), i18n.KeySessionSegmentEmbeddingDimensionMismatch.String()) {
		t.Fatalf("materializeSegments error = %v", err)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelSegmentMaterializerCreatesImageVectorRows(t *testing.T) {
	var sawTextEmbedding bool
	var sawImageDownload bool
	var sawImageEmbedding bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/workspaces/ws-1/embeddings":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode embedding request: %v", err)
			}
			if _, ok := body["images"]; ok {
				sawImageEmbedding = true
				if body["model"] != "image-embed" || body["preprocess_version"] != "image-v1" || body["backend_id"] != float64(5) {
					t.Fatalf("image embedding request = %+v", body)
				}
				_ = json.NewEncoder(w).Encode(map[string]any{
					"object": "list",
					"model":  "image-embed",
					"data": []map[string]any{{
						"object":    "embedding",
						"index":     0,
						"embedding": []float64{0.7, 0.8, 0.9, 1.0},
					}},
					"metadata": map[string]any{
						"preprocess_version": "image-v1",
						"distance_metric":    "cosine",
					},
				})
				return
			}
			sawTextEmbedding = true
			input, _ := body["input"].([]any)
			if body["model"] != "embed-model" || len(input) != 1 || input[0] != "chunk text\nocr text\nimage caption" {
				t.Fatalf("text embedding request = %+v", body)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"object": "list",
				"model":  "embed-model",
				"data": []map[string]any{{
					"object":    "embedding",
					"index":     0,
					"embedding": []float32{0.1, 0.2, 0.3},
				}},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/ws-1/files/page-image/download":
			sawImageDownload = true
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write([]byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	client, err := moi.New(server.URL, "caller-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer client.Close()

	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	expectVectorTableSchemaColumns(tenantMock, "kb_text_idx", "vecf32(3)")
	expectVectorTableSchemaColumns(tenantMock, "kb_image_idx", "vecf32(4)")

	content := "chunk text"
	ocr := "ocr text"
	desc := "image caption"
	pageImageID := "page-image"
	chunkIndex := int64(2)
	metadata := json.RawMessage(`{"page_number":3,"volume_id":"9103"}`)
	segment := kbSegmentRecord{
		VersionID:        "seg-v6",
		ModelID:          7,
		SourceID:         "source-file-1",
		KBFileID:         "kb-file",
		IndexVersion:     6,
		Level:            kbSegmentLevelChunk,
		ChunkIndex:       &chunkIndex,
		IdentityKey:      chunkIdentityKey(&chunkIndex, nil),
		Content:          &content,
		OCRText:          &ocr,
		ImageDescription: &desc,
		PageImageFileID:  &pageImageID,
		Metadata:         metadata,
		Enabled:          true,
	}

	svc := &semanticModelService{}
	ctx := ctxutil.WithTenantDB(context.Background(), tenantDB)
	materialized, err := svc.materializeSegments(ctx, client, "ws-1", kbVectorBinding{
		VectorTable:             "kb_text_idx",
		EmbeddingModel:          "embed-model",
		ImageVectorTable:        "kb_image_idx",
		ImageEmbeddingModel:     "image-embed",
		ImageEmbeddingBackendID: "5",
		ImageEmbeddingDimension: 4,
		ImagePreprocessVersion:  "image-v1",
		ImageDistanceMetric:     "cosine",
	}, []kbSegmentRecord{segment})
	if err != nil {
		t.Fatalf("materializeSegments: %v", err)
	}
	if !sawTextEmbedding || !sawImageDownload || !sawImageEmbedding {
		t.Fatalf("embedding calls text=%v download=%v image=%v", sawTextEmbedding, sawImageDownload, sawImageEmbedding)
	}
	if len(materialized.TextRows) != 1 || len(materialized.ImageRows) != 1 {
		t.Fatalf("materialized rows = %+v", materialized)
	}
	imageRow := materialized.ImageRows[0]
	if imageRow.RowID != stableID("kbimgsegrow", "kb-file", int64(6), kbSegmentLevelChunk, "idx:2", "page-image") {
		t.Fatalf("image row id = %q", imageRow.RowID)
	}
	if imageRow.PageNumber == nil || *imageRow.PageNumber != 3 {
		t.Fatalf("image row page_number = %+v", imageRow.PageNumber)
	}
	var meta map[string]any
	if err := json.Unmarshal([]byte(imageRow.Metadata), &meta); err != nil {
		t.Fatalf("unmarshal image row metadata: %v", err)
	}
	for key, want := range map[string]any{
		"modality":                   "image",
		"asset_kind":                 "document_visual",
		"source_id":                  "source-file-1",
		"segment_version_id":         "seg-v6",
		"identity_key":               "idx:2",
		"page_image_file_id":         "page-image",
		"image_embedding_model":      "image-embed",
		"image_embedding_backend_id": "5",
		"preprocess_version":         "image-v1",
		"distance_metric":            "cosine",
	} {
		if got := meta[key]; got != want {
			t.Fatalf("image metadata[%s] = %v, want %v; all=%v", key, got, want, meta)
		}
	}
	if meta["index_version"].(float64) != 6 || meta["chunk_index"].(float64) != 2 || meta["page_number"].(float64) != 3 {
		t.Fatalf("image metadata version/chunk/page = %v", meta)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelSegmentMaterializerCopyForwardsUnchangedRows(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("unexpected HTTP request during copy-forward: %s %s", r.Method, r.URL.String())
	}))
	defer server.Close()

	client, err := moi.New(server.URL, "caller-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer client.Close()

	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}

	oldContent := "audio transcript"
	oldImageID := "page-image"
	oldIndex := int64(0)
	oldMetadata := json.RawMessage(`{"page_number":2,"start_ms":0,"end_ms":1000}`)
	old := kbSegmentRecord{
		VersionID:       "seg-v5",
		ModelID:         7,
		SourceID:        "source-file-1",
		KBFileID:        "kb-file",
		IndexVersion:    5,
		Level:           kbSegmentLevelChunk,
		ChunkIndex:      &oldIndex,
		IdentityKey:     chunkIdentityKey(&oldIndex, nil),
		Content:         &oldContent,
		PageImageFileID: &oldImageID,
		Metadata:        oldMetadata,
		Enabled:         true,
	}
	newIndex := int64(1)
	next := old
	next.VersionID = "seg-v6"
	next.IndexVersion = 6
	next.ChunkIndex = &newIndex
	next.IdentityKey = chunkIdentityKey(&newIndex, nil)
	next.Metadata = json.RawMessage(`{"page_number":2,"start_ms":100,"end_ms":1100,"speaker_id":"speaker-a"}`)
	next.ReuseFrom = &old

	oldTextRowID := segmentVectorRowID(old)
	oldImageRowID := segmentImageVectorRowID(old, oldImageID)
	tenantMock.ExpectQuery("SELECT embedding, meta FROM `kb_text_idx`").
		WithArgs(oldTextRowID, "kb-file", int64(5), false).
		WillReturnRows(sqlmock.NewRows([]string{"embedding", "meta"}).AddRow("[0.1,0.2,0.3]", `{"segment_version_id":"seg-v5"}`))
	tenantMock.ExpectQuery("SELECT embedding, meta FROM `kb_image_idx`").
		WithArgs(oldImageRowID, "kb-file", int64(5), false).
		WillReturnRows(sqlmock.NewRows([]string{"embedding", "meta"}).AddRow("[0.7,0.8,0.9,1.0]", `{"image_embedding_model":"image-embed","image_embedding_backend_id":"5","image_embedding_dimension":4,"image_preprocess_version":"image-v1","image_distance_metric":"cosine","embedding_backend_metadata":{"preprocess_version":"image-v1","distance_metric":"cosine"}}`))
	expectVectorTableSchemaColumns(tenantMock, "kb_text_idx", "vecf32(3)")
	expectVectorTableSchemaColumns(tenantMock, "kb_image_idx", "vecf32(4)")

	svc := &semanticModelService{}
	ctx := ctxutil.WithTenantDB(context.Background(), tenantDB)
	materialized, err := svc.materializeSegmentsForMutation(ctx, client, "ws-1", kbVectorBinding{
		VectorTable:             "kb_text_idx",
		EmbeddingModel:          "embed-model",
		ImageVectorTable:        "kb_image_idx",
		ImageEmbeddingModel:     "image-embed",
		ImageEmbeddingBackendID: "5",
		ImageEmbeddingDimension: 4,
		ImagePreprocessVersion:  "image-v1",
		ImageDistanceMetric:     "cosine",
	}, []kbSegmentRecord{next}, kbSegmentSourceEdit)
	if err != nil {
		t.Fatalf("materializeSegmentsForMutation: %v", err)
	}
	if len(materialized.TextRows) != 1 || len(materialized.ImageRows) != 1 {
		t.Fatalf("materialized rows = %+v", materialized)
	}
	if materialized.TextRows[0].RowID != segmentVectorRowID(next) || materialized.TextRows[0].Embedding != "[0.1,0.2,0.3]" {
		t.Fatalf("text row not copied to new id/version: %+v", materialized.TextRows[0])
	}
	var textMeta map[string]any
	if err := json.Unmarshal([]byte(materialized.TextRows[0].Metadata), &textMeta); err != nil {
		t.Fatalf("unmarshal text metadata: %v", err)
	}
	if textMeta["segment_version_id"] != "seg-v6" || textMeta["index_version"].(float64) != 6 || textMeta["start_ms"].(float64) != 100 || textMeta["speaker_id"] != "speaker-a" {
		t.Fatalf("text metadata was not regenerated from next segment: %v", textMeta)
	}
	if textMeta["vector_table"] != "kb_text_idx" || textMeta["embedding_model"] != "embed-model" {
		t.Fatalf("text metadata missing current embedding binding: %v", textMeta)
	}
	if materialized.ImageRows[0].RowID != segmentImageVectorRowID(next, oldImageID) || materialized.ImageRows[0].Embedding != "[0.7,0.8,0.9,1.0]" {
		t.Fatalf("image row not copied to new id/version: %+v", materialized.ImageRows[0])
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelSegmentMaterializerCopyForwardsExternalTextRowByIdentity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("unexpected HTTP request during external text copy-forward: %s %s", r.Method, r.URL.String())
	}))
	defer server.Close()

	client, err := moi.New(server.URL, "caller-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer client.Close()

	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}

	content := "external workflow chunk"
	chunkID := "document_text:chunk:alpha"
	old := kbSegmentRecord{
		VersionID:    "seg-v5",
		ModelID:      7,
		SourceID:     "source-file-1",
		KBFileID:     "kb-file",
		IndexVersion: 5,
		Level:        kbSegmentLevelChunk,
		ChunkID:      &chunkID,
		IdentityKey:  chunkIdentityKey(nil, &chunkID),
		Content:      &content,
		Enabled:      true,
	}
	next := old
	next.VersionID = "seg-v6"
	next.IndexVersion = 6
	next.ReuseFrom = &old

	tenantMock.ExpectQuery("SELECT embedding, meta FROM `kb_text_idx`").
		WithArgs(segmentVectorRowID(old), "kb-file", int64(5), false).
		WillReturnRows(sqlmock.NewRows([]string{"embedding", "meta"}))
	expectReusableVectorRowIdentityByChunkID(tenantMock, "kb_text_idx", "kb-file", int64(5), kbSegmentLevelChunk, chunkID, "").
		WillReturnRows(sqlmock.NewRows([]string{"embedding", "meta"}).AddRow("[0.1,0.2,0.3]", `{"chunk_id":"document_text:chunk:alpha"}`))
	expectVectorTableSchemaColumns(tenantMock, "kb_text_idx", "vecf32(3)")

	svc := &semanticModelService{}
	ctx := ctxutil.WithTenantDB(context.Background(), tenantDB)
	materialized, err := svc.materializeSegmentsForMutation(ctx, client, "ws-1", kbVectorBinding{
		VectorTable:    "kb_text_idx",
		EmbeddingModel: "embed-model",
	}, []kbSegmentRecord{next}, kbSegmentSourceEdit)
	if err != nil {
		t.Fatalf("materializeSegmentsForMutation: %v", err)
	}
	if len(materialized.TextRows) != 1 || materialized.TextRows[0].RowID != segmentVectorRowID(next) || materialized.TextRows[0].Embedding != "[0.1,0.2,0.3]" {
		t.Fatalf("text row not copied from external row: %+v", materialized.TextRows)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelSegmentMaterializerReembedsTextWhenReusableTextBindingDiffers(t *testing.T) {
	var sawTextEmbedding bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/workspaces/ws-1/embeddings" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		var req struct {
			Model string   `json:"model"`
			Input []string `json:"input"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode embedding request: %v", err)
		}
		sawTextEmbedding = true
		if req.Model != "embed-model" || !sameStringSet(req.Input, []string{"chunk text"}) {
			t.Fatalf("embedding request = %+v", req)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"object": "list",
			"model":  "embed-model",
			"data": []map[string]any{{
				"object":    "embedding",
				"index":     0,
				"embedding": []float32{0.4, 0.5, 0.6},
			}},
		})
	}))
	defer server.Close()

	client, err := moi.New(server.URL, "caller-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer client.Close()

	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}

	content := "chunk text"
	chunkIndex := int64(0)
	old := kbSegmentRecord{
		VersionID:    "seg-v5",
		ModelID:      7,
		SourceID:     "source-file-1",
		KBFileID:     "kb-file",
		IndexVersion: 5,
		Level:        kbSegmentLevelChunk,
		ChunkIndex:   &chunkIndex,
		IdentityKey:  chunkIdentityKey(&chunkIndex, nil),
		Content:      &content,
		Enabled:      true,
	}
	next := old
	next.VersionID = "seg-v6"
	next.IndexVersion = 6
	next.ReuseFrom = &old

	tenantMock.ExpectQuery("SELECT embedding, meta FROM `kb_text_idx`").
		WithArgs(segmentVectorRowID(old), "kb-file", int64(5), false).
		WillReturnRows(sqlmock.NewRows([]string{"embedding", "meta"}).AddRow("[9.9,9.8,9.7]", `{"vector_table":"kb_text_idx","embedding_model":"old-embed-model"}`))
	expectVectorTableSchemaColumns(tenantMock, "kb_text_idx", "vecf32(3)")

	svc := &semanticModelService{}
	ctx := ctxutil.WithTenantDB(context.Background(), tenantDB)
	materialized, err := svc.materializeSegmentsForMutation(ctx, client, "ws-1", kbVectorBinding{
		VectorTable:    "kb_text_idx",
		EmbeddingModel: "embed-model",
	}, []kbSegmentRecord{next}, kbSegmentSourceEdit)
	if err != nil {
		t.Fatalf("materializeSegmentsForMutation error = %v", err)
	}
	if !sawTextEmbedding {
		t.Fatal("text embedding was not called after reusable text binding mismatch")
	}
	if len(materialized.TextRows) != 1 || materialized.TextRows[0].Embedding != "[0.4,0.5,0.6]" {
		t.Fatalf("text rows = %+v", materialized.TextRows)
	}
	var textMeta map[string]any
	if err := json.Unmarshal([]byte(materialized.TextRows[0].Metadata), &textMeta); err != nil {
		t.Fatalf("unmarshal text metadata: %v", err)
	}
	if textMeta["vector_table"] != "kb_text_idx" || textMeta["embedding_model"] != "embed-model" {
		t.Fatalf("text metadata missing current embedding binding: %v", textMeta)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelSegmentMaterializerRejectsInvalidReusableTextMetadata(t *testing.T) {
	var embeddingCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		embeddingCalls++
		http.Error(w, "embedding should not be called", http.StatusInternalServerError)
	}))
	defer server.Close()

	client, err := moi.New(server.URL, "caller-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer client.Close()

	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}

	content := "chunk text"
	chunkIndex := int64(0)
	old := kbSegmentRecord{
		VersionID:    "seg-v5",
		ModelID:      7,
		SourceID:     "source-file-1",
		KBFileID:     "kb-file",
		IndexVersion: 5,
		Level:        kbSegmentLevelChunk,
		ChunkIndex:   &chunkIndex,
		IdentityKey:  chunkIdentityKey(&chunkIndex, nil),
		Content:      &content,
		Enabled:      true,
	}
	next := old
	next.VersionID = "seg-v6"
	next.IndexVersion = 6
	next.ReuseFrom = &old

	tenantMock.ExpectQuery("SELECT embedding, meta FROM `kb_text_idx`").
		WithArgs(segmentVectorRowID(old), "kb-file", int64(5), false).
		WillReturnRows(sqlmock.NewRows([]string{"embedding", "meta"}).AddRow("[9.9,9.8,9.7]", "not-json"))

	svc := &semanticModelService{}
	ctx := ctxutil.WithTenantDB(context.Background(), tenantDB)
	_, err = svc.materializeSegmentsForMutation(ctx, client, "ws-1", kbVectorBinding{
		VectorTable:    "kb_text_idx",
		EmbeddingModel: "embed-model",
	}, []kbSegmentRecord{next}, kbSegmentSourceEdit)
	if err == nil || !strings.Contains(err.Error(), "decode previous segment vector row") || !strings.Contains(err.Error(), "invalid JSON") {
		t.Fatalf("materializeSegmentsForMutation error = %v", err)
	}
	if embeddingCalls != 0 {
		t.Fatalf("embedding calls = %d, want 0", embeddingCalls)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelSegmentMaterializerReembedsTextButCopyForwardsImage(t *testing.T) {
	var sawTextEmbedding bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/workspaces/ws-1/embeddings" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		var req struct {
			Model string   `json:"model"`
			Input []string `json:"input"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode embedding request: %v", err)
		}
		sawTextEmbedding = true
		if req.Model != "embed-model" || !sameStringSet(req.Input, []string{"chunk text\nnew caption"}) {
			t.Fatalf("embedding request = %+v", req)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"object": "list",
			"model":  "embed-model",
			"data": []map[string]any{{
				"object":    "embedding",
				"index":     0,
				"embedding": []float32{0.4, 0.5, 0.6},
			}},
		})
	}))
	defer server.Close()

	client, err := moi.New(server.URL, "caller-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer client.Close()

	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}

	content := "chunk text"
	oldDesc := "old caption"
	newDesc := "new caption"
	imageID := "page-image"
	chunkIndex := int64(0)
	old := kbSegmentRecord{
		VersionID:        "seg-v5",
		ModelID:          7,
		SourceID:         "source-file-1",
		KBFileID:         "kb-file",
		IndexVersion:     5,
		Level:            kbSegmentLevelChunk,
		ChunkIndex:       &chunkIndex,
		IdentityKey:      chunkIdentityKey(&chunkIndex, nil),
		Content:          &content,
		ImageDescription: &oldDesc,
		PageImageFileID:  &imageID,
		Metadata:         json.RawMessage(`{"page_number":3}`),
		Enabled:          true,
	}
	next := old
	next.VersionID = "seg-v6"
	next.IndexVersion = 6
	next.ImageDescription = &newDesc
	next.ReuseFrom = &old

	tenantMock.ExpectQuery("SELECT embedding, meta FROM `kb_image_idx`").
		WithArgs(segmentImageVectorRowID(old, imageID), "kb-file", int64(5), false).
		WillReturnRows(sqlmock.NewRows([]string{"embedding", "meta"}).AddRow("[0.7,0.8,0.9,1.0]", `{"image_embedding_model":"image-embed","image_embedding_backend_id":"5","image_embedding_dimension":4,"image_preprocess_version":"image-v1","image_distance_metric":"cosine"}`))
	expectVectorTableSchemaColumns(tenantMock, "kb_text_idx", "vecf32(3)")
	expectVectorTableSchemaColumns(tenantMock, "kb_image_idx", "vecf32(4)")

	svc := &semanticModelService{}
	ctx := ctxutil.WithTenantDB(context.Background(), tenantDB)
	materialized, err := svc.materializeSegmentsForMutation(ctx, client, "ws-1", kbVectorBinding{
		VectorTable:             "kb_text_idx",
		EmbeddingModel:          "embed-model",
		ImageVectorTable:        "kb_image_idx",
		ImageEmbeddingModel:     "image-embed",
		ImageEmbeddingBackendID: "5",
		ImageEmbeddingDimension: 4,
		ImagePreprocessVersion:  "image-v1",
		ImageDistanceMetric:     "cosine",
	}, []kbSegmentRecord{next}, kbSegmentSourceEdit)
	if err != nil {
		t.Fatalf("materializeSegmentsForMutation: %v", err)
	}
	if !sawTextEmbedding {
		t.Fatal("text embedding was not called for changed image description")
	}
	if len(materialized.TextRows) != 1 || materialized.TextRows[0].Embedding != "[0.4,0.5,0.6]" {
		t.Fatalf("text rows = %+v", materialized.TextRows)
	}
	if len(materialized.ImageRows) != 1 || materialized.ImageRows[0].Embedding != "[0.7,0.8,0.9,1.0]" {
		t.Fatalf("image row was not copy-forwarded: %+v", materialized.ImageRows)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelSegmentMaterializerReembedsTextWhenReusableTextRowMissing(t *testing.T) {
	var sawTextEmbedding bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/workspaces/ws-1/embeddings" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		var req struct {
			Model string   `json:"model"`
			Input []string `json:"input"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode embedding request: %v", err)
		}
		sawTextEmbedding = true
		if req.Model != "embed-model" || !sameStringSet(req.Input, []string{"chunk text"}) {
			t.Fatalf("embedding request = %+v", req)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"object": "list",
			"model":  "embed-model",
			"data": []map[string]any{{
				"object":    "embedding",
				"index":     0,
				"embedding": []float32{0.1, 0.2, 0.3},
			}},
		})
	}))
	defer server.Close()

	client, err := moi.New(server.URL, "caller-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer client.Close()

	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}

	content := "chunk text"
	chunkIndex := int64(0)
	old := kbSegmentRecord{
		VersionID:    "seg-v5",
		ModelID:      7,
		SourceID:     "source-file-1",
		KBFileID:     "kb-file",
		IndexVersion: 5,
		Level:        kbSegmentLevelChunk,
		ChunkIndex:   &chunkIndex,
		IdentityKey:  chunkIdentityKey(&chunkIndex, nil),
		Content:      &content,
		Enabled:      true,
	}
	next := old
	next.VersionID = "seg-v6"
	next.IndexVersion = 6
	next.ReuseFrom = &old

	tenantMock.ExpectQuery("SELECT embedding, meta FROM `kb_text_idx`").
		WithArgs(segmentVectorRowID(old), "kb-file", int64(5), false).
		WillReturnRows(sqlmock.NewRows([]string{"embedding", "meta"}))
	expectReusableVectorRowIdentityByChunkIndex(tenantMock, "kb_text_idx", "kb-file", int64(5), kbSegmentLevelChunk, chunkIndex).
		WillReturnRows(sqlmock.NewRows([]string{"embedding", "meta"}))
	expectVectorTableSchemaColumns(tenantMock, "kb_text_idx", "vecf32(3)")

	svc := &semanticModelService{}
	ctx := ctxutil.WithTenantDB(context.Background(), tenantDB)
	materialized, err := svc.materializeSegmentsForMutation(ctx, client, "ws-1", kbVectorBinding{
		VectorTable:    "kb_text_idx",
		EmbeddingModel: "embed-model",
	}, []kbSegmentRecord{next}, kbSegmentSourceEdit)
	if err != nil {
		t.Fatalf("materializeSegmentsForMutation error = %v", err)
	}
	if !sawTextEmbedding {
		t.Fatal("text embedding was not called after reusable text row miss")
	}
	if len(materialized.TextRows) != 1 || materialized.TextRows[0].Embedding != "[0.1,0.2,0.3]" {
		t.Fatalf("text rows = %+v", materialized.TextRows)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelSegmentMaterializerImageChunkReembedsTextWhenTextRowMissing(t *testing.T) {
	var sawTextEmbedding bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/workspaces/ws-1/embeddings" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		var req struct {
			Model string   `json:"model"`
			Input []string `json:"input"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode embedding request: %v", err)
		}
		sawTextEmbedding = true
		if req.Model != "embed-model" || !sameStringSet(req.Input, []string{"这是电话图标"}) {
			t.Fatalf("embedding request = %+v", req)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"object": "list",
			"model":  "embed-model",
			"data": []map[string]any{{
				"object":    "embedding",
				"index":     0,
				"embedding": []float32{0.1, 0.2, 0.3},
			}},
		})
	}))
	defer server.Close()

	client, err := moi.New(server.URL, "caller-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer client.Close()

	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}

	ocr := "这是电话图标"
	imageID := "page-image"
	chunkIndex := int64(0)
	old := kbSegmentRecord{
		VersionID:       "seg-v5",
		ModelID:         7,
		SourceID:        "source-file-1",
		KBFileID:        "kb-file",
		IndexVersion:    5,
		Level:           kbSegmentLevelChunk,
		ChunkIndex:      &chunkIndex,
		IdentityKey:     chunkIdentityKey(&chunkIndex, nil),
		OCRText:         &ocr,
		PageImageFileID: &imageID,
		Metadata:        json.RawMessage(`{"page_number":1}`),
		Enabled:         true,
	}
	next := old
	next.VersionID = "seg-v6"
	next.IndexVersion = 6
	next.ReuseFrom = &old

	tenantMock.ExpectQuery("SELECT embedding, meta FROM `kb_text_idx`").
		WithArgs(segmentVectorRowID(old), "kb-file", int64(5), false).
		WillReturnRows(sqlmock.NewRows([]string{"embedding", "meta"}))
	expectReusableVectorRowIdentityByChunkIndex(tenantMock, "kb_text_idx", "kb-file", int64(5), kbSegmentLevelChunk, chunkIndex).
		WillReturnRows(sqlmock.NewRows([]string{"embedding", "meta"}))
	tenantMock.ExpectQuery("SELECT embedding, meta FROM `kb_image_idx`").
		WithArgs(segmentImageVectorRowID(old, imageID), "kb-file", int64(5), false).
		WillReturnRows(sqlmock.NewRows([]string{"embedding", "meta"}).AddRow("[0.7,0.8,0.9,1.0]", `{"image_embedding_model":"image-embed","image_embedding_backend_id":"5","image_embedding_dimension":4,"image_preprocess_version":"image-v1","image_distance_metric":"cosine"}`))
	expectVectorTableSchemaColumns(tenantMock, "kb_text_idx", "vecf32(3)")
	expectVectorTableSchemaColumns(tenantMock, "kb_image_idx", "vecf32(4)")

	svc := &semanticModelService{}
	ctx := ctxutil.WithTenantDB(context.Background(), tenantDB)
	materialized, err := svc.materializeSegmentsForMutation(ctx, client, "ws-1", kbVectorBinding{
		VectorTable:             "kb_text_idx",
		EmbeddingModel:          "embed-model",
		ImageVectorTable:        "kb_image_idx",
		ImageEmbeddingModel:     "image-embed",
		ImageEmbeddingBackendID: "5",
		ImageEmbeddingDimension: 4,
		ImagePreprocessVersion:  "image-v1",
		ImageDistanceMetric:     "cosine",
	}, []kbSegmentRecord{next}, kbSegmentSourceEdit)
	if err != nil {
		t.Fatalf("materializeSegmentsForMutation error = %v", err)
	}
	if !sawTextEmbedding {
		t.Fatal("text embedding was not called after image chunk text row miss")
	}
	if len(materialized.TextRows) != 1 || materialized.TextRows[0].Embedding != "[0.1,0.2,0.3]" {
		t.Fatalf("text rows = %+v", materialized.TextRows)
	}
	if len(materialized.ImageRows) != 1 || materialized.ImageRows[0].Embedding != "[0.7,0.8,0.9,1.0]" {
		t.Fatalf("image rows = %+v", materialized.ImageRows)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelSegmentMaterializerFailsMissingReusableImageRowWithoutDownload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("unexpected HTTP request after missing reusable image row: %s %s", r.Method, r.URL.String())
	}))
	defer server.Close()

	client, err := moi.New(server.URL, "caller-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer client.Close()

	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}

	imageID := "page-image"
	chunkIndex := int64(0)
	old := kbSegmentRecord{
		VersionID:       "seg-v5",
		ModelID:         7,
		SourceID:        "source-file-1",
		KBFileID:        "kb-file",
		IndexVersion:    5,
		Level:           kbSegmentLevelChunk,
		ChunkIndex:      &chunkIndex,
		IdentityKey:     chunkIdentityKey(&chunkIndex, nil),
		PageImageFileID: &imageID,
		Metadata:        json.RawMessage(`{"page_number":1}`),
		Enabled:         true,
	}
	next := old
	next.VersionID = "seg-v6"
	next.IndexVersion = 6
	next.ReuseFrom = &old

	tenantMock.ExpectQuery("SELECT embedding, meta FROM `kb_image_idx`").
		WithArgs(segmentImageVectorRowID(old, imageID), "kb-file", int64(5), false).
		WillReturnRows(sqlmock.NewRows([]string{"embedding", "meta"}))
	expectReusableVectorRowIdentityByChunkIndex(tenantMock, "kb_image_idx", "kb-file", int64(5), kbSegmentLevelChunk, chunkIndex, imageID).
		WillReturnRows(sqlmock.NewRows([]string{"embedding", "meta"}))

	svc := &semanticModelService{}
	ctx := ctxutil.WithTenantDB(context.Background(), tenantDB)
	_, err = svc.materializeSegmentsForMutation(ctx, client, "ws-1", kbVectorBinding{
		VectorTable:             "kb_text_idx",
		EmbeddingModel:          "embed-model",
		ImageVectorTable:        "kb_image_idx",
		ImageEmbeddingModel:     "image-embed",
		ImageEmbeddingBackendID: "5",
		ImageEmbeddingDimension: 4,
		ImagePreprocessVersion:  "image-v1",
		ImageDistanceMetric:     "cosine",
	}, []kbSegmentRecord{next}, kbSegmentSourceEdit)
	if err == nil || !strings.Contains(err.Error(), errReusableVectorRowNotFound.Error()) {
		t.Fatalf("materializeSegmentsForMutation error = %v, want reusable row not found", err)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelSegmentMaterializerReembedsNonReusableUnchangedImageRow(t *testing.T) {
	var sawImageDownload bool
	var sawImageEmbedding bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/ws-1/files/page-image/download":
			sawImageDownload = true
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write([]byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/workspaces/ws-1/embeddings":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode image embedding request: %v", err)
			}
			if _, ok := body["images"]; !ok {
				t.Fatalf("unexpected text embedding request: %+v", body)
			}
			sawImageEmbedding = true
			_ = json.NewEncoder(w).Encode(map[string]any{
				"object": "list",
				"model":  "image-embed",
				"data": []map[string]any{{
					"object":    "embedding",
					"index":     0,
					"embedding": []float64{0.2, 0.3, 0.4, 0.5},
				}},
				"metadata": map[string]any{
					"preprocess_version": "image-v1",
					"distance_metric":    "cosine",
				},
			})
		default:
			t.Fatalf("unexpected HTTP request after non-reusable unchanged image row: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	client, err := moi.New(server.URL, "caller-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer client.Close()

	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}

	imageID := "page-image"
	chunkIndex := int64(0)
	old := kbSegmentRecord{
		VersionID:       "seg-v5",
		ModelID:         7,
		SourceID:        "source-file-1",
		KBFileID:        "kb-file",
		IndexVersion:    5,
		Level:           kbSegmentLevelChunk,
		ChunkIndex:      &chunkIndex,
		IdentityKey:     chunkIdentityKey(&chunkIndex, nil),
		PageImageFileID: &imageID,
		Metadata:        json.RawMessage(`{"page_number":1}`),
		Enabled:         true,
	}
	next := old
	next.VersionID = "seg-v6"
	next.IndexVersion = 6
	next.ReuseFrom = &old

	tenantMock.ExpectQuery("SELECT embedding, meta FROM `kb_image_idx`").
		WithArgs(segmentImageVectorRowID(old, imageID), "kb-file", int64(5), false).
		WillReturnRows(sqlmock.NewRows([]string{"embedding", "meta"}).AddRow("[0.7,0.8,0.9,1.0]", `{"image_embedding_model":"old-image-embed","image_embedding_backend_id":"5","image_embedding_dimension":4,"image_preprocess_version":"old-image-v1","image_distance_metric":"cosine"}`))
	expectVectorTableSchemaColumns(tenantMock, "kb_text_idx", "vecf32(3)")
	expectVectorTableSchemaColumns(tenantMock, "kb_image_idx", "vecf32(4)")

	svc := &semanticModelService{}
	ctx := ctxutil.WithTenantDB(context.Background(), tenantDB)
	materialized, err := svc.materializeSegmentsForMutation(ctx, client, "ws-1", kbVectorBinding{
		VectorTable:             "kb_text_idx",
		EmbeddingModel:          "embed-model",
		ImageVectorTable:        "kb_image_idx",
		ImageEmbeddingModel:     "image-embed",
		ImageEmbeddingBackendID: "5",
		ImageEmbeddingDimension: 4,
		ImagePreprocessVersion:  "image-v1",
		ImageDistanceMetric:     "cosine",
	}, []kbSegmentRecord{next}, kbSegmentSourceEdit)
	if err != nil {
		t.Fatalf("materializeSegmentsForMutation error = %v", err)
	}
	if !sawImageDownload || !sawImageEmbedding {
		t.Fatalf("image materialization calls download=%v embedding=%v", sawImageDownload, sawImageEmbedding)
	}
	if len(materialized.TextRows) != 0 || len(materialized.ImageRows) != 1 {
		t.Fatalf("materialized rows = %+v", materialized)
	}
	if materialized.ImageRows[0].Embedding != "[0.2,0.3,0.4,0.5]" {
		t.Fatalf("image row was not reembedded: %+v", materialized.ImageRows[0])
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelSegmentMaterializerCreatesTextChunkWhenExternalImageRowUsesWorkflowID(t *testing.T) {
	var sawTextEmbedding bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/workspaces/ws-1/embeddings" {
			t.Fatalf("unexpected HTTP request during create text chunk: %s %s", r.Method, r.URL.String())
		}
		var req struct {
			Model string   `json:"model"`
			Input []string `json:"input"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode embedding request: %v", err)
		}
		sawTextEmbedding = true
		if req.Model != "embed-model" || !sameStringSet(req.Input, []string{"new text chunk"}) {
			t.Fatalf("embedding request = %+v", req)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"object": "list",
			"model":  "embed-model",
			"data": []map[string]any{{
				"object":    "embedding",
				"index":     0,
				"embedding": []float32{0.4, 0.5, 0.6},
			}},
		})
	}))
	defer server.Close()

	client, err := moi.New(server.URL, "caller-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer client.Close()

	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}

	imageID := "page-image"
	imageChunkID := "document_visual_image:page:2b182a11ef458c23b6fe74a8"
	oldImage := kbSegmentRecord{
		VersionID:       "seg-v5",
		ModelID:         7,
		SourceID:        "source-file-1",
		KBFileID:        "kb-file",
		IndexVersion:    5,
		Level:           kbSegmentLevelChunk,
		ChunkID:         &imageChunkID,
		IdentityKey:     chunkIdentityKey(nil, &imageChunkID),
		PageImageFileID: &imageID,
		Metadata:        json.RawMessage(`{"page_number":1,"chunk_id":"document_visual_image:page:2b182a11ef458c23b6fe74a8","page_image_file_id":"page-image"}`),
		Enabled:         true,
	}
	nextImage := oldImage
	nextImage.VersionID = "seg-v6"
	nextImage.IndexVersion = 6
	nextImage.ReuseFrom = &oldImage

	newIndex := int64(0)
	newContent := "new text chunk"
	newText := kbSegmentRecord{
		VersionID:    "seg-v6",
		ModelID:      7,
		SourceID:     "source-file-1",
		KBFileID:     "kb-file",
		IndexVersion: 6,
		Level:        kbSegmentLevelChunk,
		ChunkIndex:   &newIndex,
		IdentityKey:  chunkIdentityKey(&newIndex, nil),
		Content:      &newContent,
		Enabled:      true,
	}

	tenantMock.ExpectQuery("SELECT embedding, meta FROM `kb_image_idx`").
		WithArgs(segmentImageVectorRowID(oldImage, imageID), "kb-file", int64(5), false).
		WillReturnRows(sqlmock.NewRows([]string{"embedding", "meta"}))
	expectReusableVectorRowIdentityByChunkID(tenantMock, "kb_image_idx", "kb-file", int64(5), kbSegmentLevelChunk, imageChunkID, imageID).
		WillReturnRows(sqlmock.NewRows([]string{"embedding", "meta"}).AddRow("[0.7,0.8,0.9,1.0]", `{"image_embedding_model":"image-embed","image_embedding_backend_id":"5","image_embedding_dimension":4,"image_preprocess_version":"image-v1","image_distance_metric":"cosine"}`))
	expectVectorTableSchemaColumns(tenantMock, "kb_text_idx", "vecf32(3)")
	expectVectorTableSchemaColumns(tenantMock, "kb_image_idx", "vecf32(4)")

	svc := &semanticModelService{}
	ctx := ctxutil.WithTenantDB(context.Background(), tenantDB)
	materialized, err := svc.materializeSegmentsForMutation(ctx, client, "ws-1", kbVectorBinding{
		VectorTable:             "kb_text_idx",
		EmbeddingModel:          "embed-model",
		ImageVectorTable:        "kb_image_idx",
		ImageEmbeddingModel:     "image-embed",
		ImageEmbeddingBackendID: "5",
		ImageEmbeddingDimension: 4,
		ImagePreprocessVersion:  "image-v1",
		ImageDistanceMetric:     "cosine",
	}, []kbSegmentRecord{newText, nextImage}, kbSegmentSourceCreate)
	if err != nil {
		t.Fatalf("materializeSegmentsForMutation error = %v", err)
	}
	if !sawTextEmbedding {
		t.Fatal("text embedding was not called for new chunk")
	}
	if len(materialized.TextRows) != 1 || materialized.TextRows[0].Content != "new text chunk" || materialized.TextRows[0].Embedding != "[0.4,0.5,0.6]" {
		t.Fatalf("text rows = %+v", materialized.TextRows)
	}
	if len(materialized.ImageRows) != 1 || materialized.ImageRows[0].Embedding != "[0.7,0.8,0.9,1.0]" {
		t.Fatalf("image rows = %+v, want external workflow row copy-forwarded", materialized.ImageRows)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestPrependSegmentRecordShiftsOnlySameLevelChunkIndexes(t *testing.T) {
	contentA := "first chunk"
	contentB := "second chunk"
	contentOtherLevel := "section chunk"
	contentChunkID := "id chunk"
	index0 := int64(0)
	index1 := int64(1)
	otherLevelIndex := int64(0)
	chunkID := "manual-id"
	createdContent := "created chunk"
	createdIndex := int64(0)

	next := prependSegmentRecord([]kbSegmentRecord{
		{
			Level:      kbSegmentLevelChunk,
			ChunkIndex: &index0,
			Content:    &contentA,
			Enabled:    true,
		},
		{
			Level:      kbSegmentLevelChunk,
			ChunkIndex: &index1,
			Content:    &contentB,
			Enabled:    true,
		},
		{
			Level:      "section",
			ChunkIndex: &otherLevelIndex,
			Content:    &contentOtherLevel,
			Enabled:    true,
		},
		{
			Level:   kbSegmentLevelChunk,
			ChunkID: &chunkID,
			Content: &contentChunkID,
			Enabled: true,
		},
	}, kbSegmentRecord{
		Level:      kbSegmentLevelChunk,
		ChunkIndex: &createdIndex,
		Content:    &createdContent,
		Enabled:    true,
	})

	record := KnowledgeBaseSourceRecord{
		ModelID:  7,
		SourceID: "source-file-1",
		KBFileID: &[]string{"kb-file-1"}[0],
	}
	if err := prepareNextSegmentVersion(record, "segment-v2", 5, next); err != nil {
		t.Fatalf("prepareNextSegmentVersion: %v", err)
	}

	if len(next) != 5 {
		t.Fatalf("next segments = %+v", next)
	}
	if next[0].Content == nil || *next[0].Content != createdContent || next[0].ChunkIndex == nil || *next[0].ChunkIndex != 0 || next[0].IdentityKey != "idx:0" {
		t.Fatalf("created segment should be persisted first with chunk_index 0: %+v", next[0])
	}
	if next[1].Content == nil || *next[1].Content != contentA || next[1].ChunkIndex == nil || *next[1].ChunkIndex != 1 || next[1].IdentityKey != "idx:1" {
		t.Fatalf("first old chunk should shift to index 1: %+v", next[1])
	}
	if next[2].Content == nil || *next[2].Content != contentB || next[2].ChunkIndex == nil || *next[2].ChunkIndex != 2 || next[2].IdentityKey != "idx:2" {
		t.Fatalf("second old chunk should shift to index 2: %+v", next[2])
	}
	if next[3].Content == nil || *next[3].Content != contentOtherLevel || next[3].ChunkIndex == nil || *next[3].ChunkIndex != 0 || next[3].IdentityKey != "idx:0" {
		t.Fatalf("other level chunk should keep its index: %+v", next[3])
	}
	if next[4].Content == nil || *next[4].Content != contentChunkID || next[4].ChunkIndex != nil || next[4].ChunkID == nil || *next[4].ChunkID != chunkID || next[4].IdentityKey != "id:manual-id" {
		t.Fatalf("chunk_id segment should keep identity: %+v", next[4])
	}
	if next[0].SegmentID != stableID("kb-segment", "segment-v2", kbSegmentLevelChunk, "idx:0") ||
		next[1].SegmentID != stableID("kb-segment", "segment-v2", kbSegmentLevelChunk, "idx:1") ||
		next[2].SegmentID != stableID("kb-segment", "segment-v2", kbSegmentLevelChunk, "idx:2") {
		t.Fatalf("segment ids should reflect shifted chunk indexes: %+v", []string{next[0].SegmentID, next[1].SegmentID, next[2].SegmentID})
	}
}

func TestMutateClonedSegmentByCurrentIDUsesCurrentSegmentIDs(t *testing.T) {
	contentA := "first chunk"
	contentB := "second chunk"
	replacement := "updated chunk"
	current := []kbSegmentRecord{
		{SegmentID: "seg-a", Content: &contentA, Enabled: true},
		{SegmentID: "seg-b", Content: &contentB, Enabled: true},
	}
	cloned := cloneSegmentRecords(current)
	matched := mutateClonedSegmentByCurrentID(current, cloned, "seg-b", func(seg *kbSegmentRecord) {
		seg.Content = &replacement
		seg.Enabled = false
		seg.WordCount = segmentWordCount(*seg)
	})

	if !matched {
		t.Fatal("mutateClonedSegmentByCurrentID did not match existing segment")
	}
	if cloned[0].SegmentID != "" || cloned[1].SegmentID != "" {
		t.Fatalf("clone should keep generated ids empty before prepareNextSegmentVersion: %+v", cloned)
	}
	if cloned[0].Content == nil || *cloned[0].Content != contentA || !cloned[0].Enabled {
		t.Fatalf("non-target segment changed: %+v", cloned[0])
	}
	if cloned[1].Content == nil || *cloned[1].Content != replacement || cloned[1].Enabled {
		t.Fatalf("target segment was not updated: %+v", cloned[1])
	}
	if cloned[1].WordCount != int64(len([]rune(replacement))) {
		t.Fatalf("target word count = %d, want %d", cloned[1].WordCount, len([]rune(replacement)))
	}
	if mutateClonedSegmentByCurrentID(current, cloned, "missing-seg", func(*kbSegmentRecord) {}) {
		t.Fatal("missing segment should not match")
	}
}

func TestRemoveClonedSegmentByCurrentIDRemovesOnlyTarget(t *testing.T) {
	contentA := "first chunk"
	contentB := "second chunk"
	contentC := "third chunk"
	current := []kbSegmentRecord{
		{SegmentID: "seg-a", VersionID: "segment-v1", IndexVersion: 4, Content: &contentA, Enabled: true},
		{SegmentID: "seg-b", VersionID: "segment-v1", IndexVersion: 4, Content: &contentB, Enabled: false},
		{SegmentID: "seg-c", VersionID: "segment-v1", IndexVersion: 4, Content: &contentC, Enabled: true},
	}

	next, matched := removeClonedSegmentByCurrentID(current, "seg-b")
	if !matched {
		t.Fatal("removeClonedSegmentByCurrentID did not match existing segment")
	}
	if len(next) != 2 {
		t.Fatalf("next len = %d, want 2: %+v", len(next), next)
	}
	if next[0].Content == nil || *next[0].Content != contentA || next[1].Content == nil || *next[1].Content != contentC {
		t.Fatalf("unexpected remaining segments: %+v", next)
	}
	for _, seg := range next {
		if seg.SegmentID != "" || seg.VersionID != "" || seg.IndexVersion != 0 {
			t.Fatalf("new version clone should clear generated identity fields: %+v", seg)
		}
	}
	if current[1].SegmentID != "seg-b" || current[1].VersionID != "segment-v1" || current[1].IndexVersion != 4 {
		t.Fatalf("current historical record was mutated: %+v", current[1])
	}
	if _, matched := removeClonedSegmentByCurrentID(current, "missing-seg"); matched {
		t.Fatal("missing segment should not match")
	}
}

func TestValidateVectorTableSchemaRejectsUnknownEmbeddingType(t *testing.T) {
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	expectVectorTableSchemaColumns(tenantMock, "kb_text_idx", "json")

	_, err = validateVectorTableSchema(context.Background(), tenantDB, "kb_text_idx", 3)
	if err == nil || !strings.Contains(err.Error(), i18n.KeySessionVectorTableEmbeddingColumnInvalid.String()) {
		t.Fatalf("validateVectorTableSchema error = %v", err)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestParseKBVectorBindingRequiresExplicitVectorTableAndEmbeddingModel(t *testing.T) {
	if _, err := parseKBVectorBinding(json.RawMessage(`{"file_ids":["kb-file"],"embedding_model":"embed-model"}`)); err == nil || !strings.Contains(err.Error(), i18n.KeySessionSegmentVectorTableRequired.String()) {
		t.Fatalf("parseKBVectorBinding missing vector_table error = %v", err)
	}
	if _, err := parseKBVectorBinding(json.RawMessage(`{"file_ids":["kb-file"],"vector_table":"kb_text_idx"}`)); err == nil || !strings.Contains(err.Error(), i18n.KeySessionSegmentEmbeddingModelRequired.String()) {
		t.Fatalf("parseKBVectorBinding missing embedding_model error = %v", err)
	}
}

func TestImportInitialSegmentsAfterAppendUsesPreservedVectorTable(t *testing.T) {
	appendedFiles, err := appendSemanticModelFiles(json.RawMessage(`{
		"file_ids": ["existing-file"],
		"vector_table": "kb_ttt_mr4ybawvtdbn",
		"embedding_model": "embed-model"
	}`), 14, []string{"kb-file"})
	if err != nil {
		t.Fatalf("appendSemanticModelFiles: %v", err)
	}
	binding, err := parseKBVectorBinding(appendedFiles)
	if err != nil {
		t.Fatalf("parseKBVectorBinding: %v", err)
	}
	if binding.VectorTable != "kb_ttt_mr4ybawvtdbn" {
		t.Fatalf("vector_table = %q, want preserved external workflow table", binding.VectorTable)
	}

	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	expectVectorTableSchemaColumns(tenantMock, "kb_ttt_mr4ybawvtdbn", "vecf32(3)")
	expectInitialTextVectorLegacyBackfill(tenantMock, "kb_ttt_mr4ybawvtdbn", "kb-file")
	tenantMock.ExpectQuery("SELECT MAX\\(index_version\\) FROM `kb_ttt_mr4ybawvtdbn`").
		WithArgs("kb-file").
		WillReturnRows(sqlmock.NewRows([]string{"max"}).AddRow(int64(5)))
	tenantMock.ExpectQuery("(?s)SELECT id, content, meta, level, chunk_index, index_version.*level = 'chunk'").
		WithArgs("kb-file", int64(5)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "content", "meta", "level", "chunk_index", "index_version"}).
			AddRow("row-external-1", "external workflow chunk", `{"chunk_id":"external-chunk-1"}`, "chunk", int64(0), int64(5)))

	svc := &semanticModelService{}
	ctx := ctxutil.WithTenantDB(context.Background(), tenantDB)
	segments, indexVersion, err := svc.importInitialSegmentsFromVectorRows(ctx, nil, appendedFiles, KnowledgeBaseSourceRecord{
		SourceID: "source-file-1",
		ModelID:  int64(14),
		Status:   kbSourceStatusSucceeded,
		KBFileID: stringPtr("kb-file"),
	}, binding)

	if err != nil {
		t.Fatalf("importInitialSegmentsFromVectorRows: %v", err)
	}
	if indexVersion != 5 || len(segments) != 1 {
		t.Fatalf("indexVersion=%d segments=%+v", indexVersion, segments)
	}
	if segments[0].Content == nil || *segments[0].Content != "external workflow chunk" {
		t.Fatalf("segment content = %+v", segments[0].Content)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestAppendSemanticModelFilesOnlyFillsMissingVectorBindings(t *testing.T) {
	files, err := appendSemanticModelFiles(json.RawMessage(`{
		"file_ids": [],
		"vector_table": "external_text_idx",
		"embedding_model": "embed-model",
		"image_embedding_model": "efficientnet-b3",
		"image_embedding_backend_id": "-30010",
		"image_embedding_dimension": 1536,
		"image_preprocess_version": "efficientnet-b3-v1-rgb-300-letterbox-imagenet",
		"image_distance_metric": "cosine"
	}`), 77, []string{"kb-file"})
	if err != nil {
		t.Fatalf("appendSemanticModelFiles: %v", err)
	}
	var out semanticModelFilesPayload
	if err := json.Unmarshal(files, &out); err != nil {
		t.Fatalf("unmarshal files: %v", err)
	}
	if out.VectorTable != "external_text_idx" || out.EmbeddingModel != "embed-model" {
		t.Fatalf("text vector binding = %+v", out)
	}
	if out.ImageVectorTable != defaultKnowledgeBaseImageVectorTable(77) {
		t.Fatalf("image_vector_table = %q, want default for missing complete image binding", out.ImageVectorTable)
	}

	files, err = appendSemanticModelFiles(json.RawMessage(`{
		"file_ids": [],
		"vector_table": "external_text_idx",
		"embedding_model": "embed-model",
		"image_vector_table": "external_image_idx",
		"image_embedding_model": "efficientnet-b3",
		"image_embedding_backend_id": "-30010",
		"image_embedding_dimension": 1536,
		"image_preprocess_version": "efficientnet-b3-v1-rgb-300-letterbox-imagenet",
		"image_distance_metric": "cosine"
	}`), 77, []string{"kb-file"})
	if err != nil {
		t.Fatalf("appendSemanticModelFiles existing image binding: %v", err)
	}
	if err := json.Unmarshal(files, &out); err != nil {
		t.Fatalf("unmarshal files with existing image binding: %v", err)
	}
	if out.VectorTable != "external_text_idx" || out.ImageVectorTable != "external_image_idx" {
		t.Fatalf("existing vector bindings were not preserved: %+v", out)
	}
}

func expectVectorTableSchemaColumns(mock sqlmock.Sqlmock, tableName, embeddingType string) {
	rows := sqlmock.NewRows([]string{"COLUMN_NAME", "COLUMN_TYPE"}).
		AddRow("id", "varchar(128)").
		AddRow("embedding", embeddingType).
		AddRow("content", "text").
		AddRow("meta", "json").
		AddRow("file_id", "varchar(128)").
		AddRow("page_number", "int").
		AddRow("level", "varchar(64)").
		AddRow("chunk_index", "bigint").
		AddRow("index_version", "bigint").
		AddRow("disabled", "tinyint")
	mock.ExpectQuery("SELECT COLUMN_NAME, COLUMN_TYPE FROM information_schema\\.COLUMNS").
		WithArgs(tableName).
		WillReturnRows(rows)
}

func TestEnsureVectorReuseTargetTableReturnsCreateTableError(t *testing.T) {
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}

	expectVectorTableSchemaColumns(tenantMock, "source_vec", "vecf32(3)")
	tenantMock.ExpectQuery("SELECT COLUMN_NAME, COLUMN_TYPE FROM information_schema\\.COLUMNS").
		WithArgs("target_vec").
		WillReturnRows(sqlmock.NewRows([]string{"COLUMN_NAME", "COLUMN_TYPE"}))
	tenantMock.ExpectExec("CREATE TABLE `target_vec` LIKE `source_vec`").
		WillReturnError(errors.New("permission denied"))

	ok, err := ensureVectorReuseTargetTable(context.Background(), tenantDB, "source_vec", "target_vec", 3)
	if err == nil || !strings.Contains(err.Error(), "create vector reuse target table target_vec like source_vec") {
		t.Fatalf("ensureVectorReuseTargetTable error = %v", err)
	}
	if ok {
		t.Fatalf("ensureVectorReuseTargetTable ok = true, want false")
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestPublishCatalogFileVectorReuseSeparatesTextAndImageChunkIndexIdentities(t *testing.T) {
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}

	record := KnowledgeBaseSourceRecord{
		SourceID: "source-file-1",
		ModelID:  77,
		KBFileID: stringPtr("kb-file"),
	}
	binding := kbVectorBinding{
		VectorTable:             "kb_text_idx",
		EmbeddingModel:          "text-embedding",
		ImageVectorTable:        "kb_image_idx",
		ImageEmbeddingModel:     "image-embedding",
		ImageEmbeddingBackendID: "5",
		ImageEmbeddingDimension: 4,
		ImagePreprocessVersion:  "image-preprocess-v1",
		ImageDistanceMetric:     "cosine",
	}

	expectVectorTableSchemaColumns(tenantMock, "kb_text_idx", "vecf32(3)")
	expectVectorTableSchemaColumns(tenantMock, "kb_text_idx", "vecf32(3)")
	tenantMock.ExpectQuery("SELECT MAX\\(index_version\\) FROM `kb_text_idx` WHERE file_id = \\? AND index_version > 0").
		WithArgs("kb-file").
		WillReturnRows(sqlmock.NewRows([]string{"MAX(index_version)"}).AddRow(int64(2)))
	tenantMock.ExpectQuery("SELECT id, embedding, content, meta, level, chunk_index, index_version, NULL\\s+FROM `kb_text_idx`").
		WithArgs("kb-file", int64(2)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "embedding", "content", "meta", "level", "chunk_index", "index_version", "page_number"}).
			AddRow("text-row-0", "[0.1,0.2,0.3]", "text content", `{"chunk_id":"text-chunk-0"}`, kbSegmentLevelChunk, int64(0), int64(2), nil))
	expectVectorTableSchemaColumns(tenantMock, "kb_image_idx", "vecf32(4)")
	expectVectorTableSchemaColumns(tenantMock, "kb_image_idx", "vecf32(4)")
	tenantMock.ExpectQuery("SELECT id, embedding, content, meta, level, chunk_index, index_version, page_number\\s+FROM `kb_image_idx`").
		WithArgs("kb-file", int64(2)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "embedding", "content", "meta", "level", "chunk_index", "index_version", "page_number"}).
			AddRow("image-row-0", "[0.4,0.5,0.6,0.7]", "page text", `{"chunk_id":"image-chunk-0","image_file_id":"image-file-0","page_image_file_id":"page-image-0","image_description":"image desc"}`, kbSegmentLevelChunk, int64(0), int64(2), int64(1)))

	versionID := stableID("kb-segver", record.SourceID, int64(1), kbSegmentSourceExternal)
	textSegmentID := stableID("kb-segment", versionID, kbSegmentLevelChunk, "idx:0")
	imageSegmentID := stableID("kb-segment", versionID, kbSegmentLevelChunk, "image:idx:0")
	if textSegmentID == imageSegmentID {
		t.Fatal("test setup produced identical text and image segment ids")
	}
	targetTextSegment := kbSegmentRecord{
		KBFileID:     "kb-file",
		IndexVersion: 1,
		Level:        kbSegmentLevelChunk,
		ChunkIndex:   int64Ptr(0),
		IdentityKey:  "idx:0",
	}
	targetImageSegment := targetTextSegment
	targetImageSegment.IdentityKey = "image:idx:0"
	targetImageRowID := segmentImageVectorRowID(targetImageSegment, "image-file-0")
	tenantMock.ExpectQuery("SELECT embedding, meta FROM `kb_text_idx`").
		WithArgs(segmentVectorRowID(targetTextSegment), "kb-file", int64(1), false).
		WillReturnRows(sqlmock.NewRows([]string{"embedding", "meta"}).
			AddRow("[0.1,0.2,0.3]", `{"vector_table":"external_text_idx","embedding_model":"text-embedding"}`))
	tenantMock.ExpectQuery("SELECT embedding, meta FROM `kb_image_idx`").
		WithArgs(targetImageRowID, "kb-file", int64(1), false).
		WillReturnRows(sqlmock.NewRows([]string{"embedding", "meta"}).
			AddRow("[0.4,0.5,0.6,0.7]", `{"image_embedding_model":"image-embedding","image_embedding_backend_id":"other-backend","image_embedding_dimension":4,"image_preprocess_version":"image-preprocess-v1","image_distance_metric":"cosine"}`))
	tenantMock.ExpectBegin()
	tenantMock.ExpectExec("INSERT INTO knowledge_base_segment_versions").
		WillReturnResult(sqlmock.NewResult(1, 1))
	// Sorted map columns produce one multi-row batch for segments, then one for recall stats.
	tenantMock.ExpectExec("INSERT INTO [`\"]?knowledge_base_segments[`\"]?").
		WithArgs(
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), "user-1", true, "idx:0", sqlmock.AnyArg(), sqlmock.AnyArg(), int64(1), "kb-file", kbSegmentLevelChunk, sqlmock.AnyArg(), int64(77), sqlmock.AnyArg(), sqlmock.AnyArg(), textSegmentID, "source-file-1", "user-1", versionID, sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), "user-1", true, "image:idx:0", sqlmock.AnyArg(), sqlmock.AnyArg(), int64(1), "kb-file", kbSegmentLevelChunk, sqlmock.AnyArg(), int64(77), sqlmock.AnyArg(), sqlmock.AnyArg(), imageSegmentID, "source-file-1", "user-1", versionID, sqlmock.AnyArg(),
		).
		WillReturnResult(sqlmock.NewResult(2, 2))
	tenantMock.ExpectExec("INSERT INTO [`\"]?knowledge_base_chunk_recall_stats[`\"]?").
		WithArgs(
			sqlmock.AnyArg(), sqlmock.AnyArg(), "idx:0", int64(1), "kb-file", kbSegmentLevelChunk, int64(77), int64(0), "source-file-1",
			sqlmock.AnyArg(), sqlmock.AnyArg(), "image:idx:0", int64(1), "kb-file", kbSegmentLevelChunk, int64(77), int64(0), "source-file-1",
		).
		WillReturnResult(sqlmock.NewResult(2, 2))
	tenantMock.ExpectExec("UPDATE knowledge_base_sources").
		WillReturnResult(sqlmock.NewResult(1, 1))
	tenantMock.ExpectCommit()

	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)
	svc := &semanticModelService{}
	reused, err := svc.publishCatalogFileVectorReuse(ctx, record, binding, catalogFileVectorReuseCandidate{VectorTable: "kb_text_idx", ReuseFileID: "kb-file"}, catalogFileVectorReuseCandidate{VectorTable: "kb_image_idx", ReuseFileID: "kb-file"}, "user-1")
	if err != nil {
		t.Fatalf("publishCatalogFileVectorReuse: %v", err)
	}
	if !reused {
		t.Fatal("publishCatalogFileVectorReuse reused = false, want true")
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestPublishCatalogFileVectorReuseTreatsCommittedDuplicateVersionBeforeCopyAsIdempotent(t *testing.T) {
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}

	record := KnowledgeBaseSourceRecord{
		SourceID: "source-file-1",
		ModelID:  77,
		KBFileID: stringPtr("kb-file"),
	}
	binding := kbVectorBinding{
		VectorTable:    "kb_target_idx",
		EmbeddingModel: "text-embedding",
	}

	expectVectorTableSchemaColumns(tenantMock, "source_text_idx", "vecf32(3)")
	expectVectorTableSchemaColumns(tenantMock, "kb_target_idx", "vecf32(3)")
	tenantMock.ExpectQuery("SELECT MAX\\(index_version\\) FROM `source_text_idx` WHERE file_id = \\? AND index_version > 0").
		WithArgs("source-file").
		WillReturnRows(sqlmock.NewRows([]string{"MAX(index_version)"}).AddRow(int64(1)))
	tenantMock.ExpectQuery("SELECT id, embedding, content, meta, level, chunk_index, index_version, NULL\\s+FROM `source_text_idx`").
		WithArgs("source-file", int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "embedding", "content", "meta", "level", "chunk_index", "index_version", "page_number"}).
			AddRow("text-row-0", "[0.1,0.2,0.3]", "text content", `{"chunk_id":"text-chunk-0"}`, kbSegmentLevelChunk, int64(0), int64(1), nil))

	versionID := stableID("kb-segver", record.SourceID, int64(1), kbSegmentSourceExternal)
	targetTextSegment := kbSegmentRecord{
		KBFileID:     "kb-file",
		IndexVersion: 1,
		Level:        kbSegmentLevelChunk,
		ChunkIndex:   int64Ptr(0),
		IdentityKey:  "idx:0",
	}
	tenantMock.ExpectQuery("SELECT embedding, meta FROM `kb_target_idx`").
		WithArgs(segmentVectorRowID(targetTextSegment), "kb-file", int64(1), false).
		WillReturnRows(sqlmock.NewRows([]string{"embedding", "meta"}))
	tenantMock.ExpectBegin()
	tenantMock.ExpectExec("INSERT INTO knowledge_base_segment_versions").
		WillReturnError(&mysqlDriver.MySQLError{Number: 1062, Message: fmt.Sprintf("Duplicate entry '%s' for key 'version_id'", versionID)})
	tenantMock.ExpectRollback()
	tenantMock.ExpectQuery("SELECT COUNT\\(1\\).*FROM knowledge_base_segment_versions").
		WithArgs(versionID, int64(77), record.SourceID, "kb-file", int64(1), kbSegmentStatusCommitted, kbSegmentSourceExternal, "kb-file", versionID, int64(1), kbSourceStatusSucceeded).
		WillReturnRows(sqlmock.NewRows([]string{"COUNT(1)"}).AddRow(int64(1)))

	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)
	svc := &semanticModelService{}
	reused, err := svc.publishCatalogFileVectorReuse(ctx, record, binding, catalogFileVectorReuseCandidate{VectorTable: "source_text_idx", ReuseFileID: "source-file"}, catalogFileVectorReuseCandidate{}, "user-1")
	if err != nil {
		t.Fatalf("publishCatalogFileVectorReuse: %v", err)
	}
	if !reused {
		t.Fatal("publishCatalogFileVectorReuse reused = false, want true")
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestTextVectorMetadataMatchesBindingUsesEmbeddingModelBeforeVectorTable(t *testing.T) {
	binding := kbVectorBinding{
		VectorTable:    "kb_text_idx",
		EmbeddingModel: "text-embedding",
	}
	tests := []struct {
		name string
		meta map[string]any
		want bool
	}{
		{
			name: "same embedding ignores different vector table",
			meta: map[string]any{
				"vector_table":    "external_text_idx",
				"embedding_model": "text-embedding",
			},
			want: true,
		},
		{
			name: "different embedding rejects same vector table",
			meta: map[string]any{
				"vector_table":    "kb_text_idx",
				"embedding_model": "old-text-embedding",
			},
			want: false,
		},
		{
			name: "legacy vector table only remains compatible",
			meta: map[string]any{
				"vector_table": "kb_text_idx",
			},
			want: true,
		},
		{
			name: "legacy empty metadata remains compatible",
			meta: map[string]any{},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := textVectorMetadataMatchesBinding(tt.meta, binding); got != tt.want {
				t.Fatalf("textVectorMetadataMatchesBinding() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReusableTargetVectorRowExistsRejectsTextEmbeddingMismatch(t *testing.T) {
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}

	seg := kbSegmentRecord{
		KBFileID:     "kb-file",
		IndexVersion: 1,
		Level:        kbSegmentLevelChunk,
		ChunkIndex:   int64Ptr(0),
		IdentityKey:  "idx:0",
	}
	rowID := segmentVectorRowID(seg)
	tenantMock.ExpectQuery("SELECT embedding, meta FROM `kb_text_idx`").
		WithArgs(rowID, "kb-file", int64(1), false).
		WillReturnRows(sqlmock.NewRows([]string{"embedding", "meta"}).
			AddRow("[0.1,0.2,0.3]", `{"vector_table":"kb_text_idx","embedding_model":"old-text-embedding"}`))

	exists, err := reusableTargetVectorRowExists(context.Background(), tenantDB, "`kb_text_idx`", kbVectorInsert{
		Segment: seg,
		RowID:   rowID,
	}, kbVectorBinding{
		VectorTable:    "kb_text_idx",
		EmbeddingModel: "text-embedding",
	}, false)
	if err == nil || !strings.Contains(err.Error(), "mismatched binding metadata") {
		t.Fatalf("reusableTargetVectorRowExists error = %v, want mismatched binding metadata", err)
	}
	if exists {
		t.Fatal("reusableTargetVectorRowExists exists = true, want false")
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestImageVectorMetadataMatchesBindingRequiresImageVectorSpace(t *testing.T) {
	binding := kbVectorBinding{
		ImageEmbeddingModel:     "image-embedding",
		ImageEmbeddingBackendID: "5",
		ImageEmbeddingDimension: 4,
		ImagePreprocessVersion:  "image-preprocess-v1",
		ImageDistanceMetric:     "cosine",
	}
	tests := []struct {
		name string
		meta map[string]any
		want bool
	}{
		{
			name: "same vector space ignores backend",
			meta: map[string]any{
				"image_embedding_model":      "image-embedding",
				"image_embedding_backend_id": "other-backend",
				"image_embedding_dimension":  4,
				"image_preprocess_version":   "image-preprocess-v1",
				"image_distance_metric":      "cosine",
			},
			want: true,
		},
		{
			name: "different model rejects",
			meta: map[string]any{
				"image_embedding_model":     "other-image-embedding",
				"image_embedding_dimension": 4,
			},
			want: false,
		},
		{
			name: "different dimension rejects",
			meta: map[string]any{
				"image_embedding_model":     "image-embedding",
				"image_embedding_dimension": 8,
			},
			want: false,
		},
		{
			name: "different preprocess rejects",
			meta: map[string]any{
				"image_embedding_model":     "image-embedding",
				"image_embedding_dimension": 4,
				"image_preprocess_version":  "other-preprocess",
				"image_distance_metric":     "cosine",
			},
			want: false,
		},
		{
			name: "different distance rejects",
			meta: map[string]any{
				"image_embedding_model":     "image-embedding",
				"image_embedding_dimension": 4,
				"image_preprocess_version":  "image-preprocess-v1",
				"image_distance_metric":     "l2",
			},
			want: false,
		},
		{
			name: "legacy embedding field names remain compatible",
			meta: map[string]any{
				"embedding_model":     "image-embedding",
				"embedding_dimension": 4,
				"preprocess_version":  "image-preprocess-v1",
				"distance_metric":     "cosine",
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := imageVectorMetadataMatchesBinding(tt.meta, binding); got != tt.want {
				t.Fatalf("imageVectorMetadataMatchesBinding() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSelectImageVectorReuseCandidateUsesModelAndDimension(t *testing.T) {
	candidate, ok, err := selectImageVectorReuseCandidate([]catalogFileVectorReuseCandidate{
		{
			VectorTable: "external_image_idx",
			Meta:        sql.NullString{String: `{"index_modality":"image","image_embedding_model":"image-embedding","image_embedding_backend_id":"other-backend","image_embedding_dimension":4,"preprocess_version":"image-preprocess-v1","distance_metric":"cosine"}`, Valid: true},
		},
	}, kbVectorBinding{
		ImageVectorTable:        "kb_image_idx",
		ImageEmbeddingModel:     "image-embedding",
		ImageEmbeddingBackendID: "5",
		ImageEmbeddingDimension: 4,
		ImagePreprocessVersion:  "image-preprocess-v1",
		ImageDistanceMetric:     "cosine",
	})
	if err != nil {
		t.Fatalf("selectImageVectorReuseCandidate: %v", err)
	}
	if !ok || candidate.VectorTable != "external_image_idx" {
		t.Fatalf("candidate = %+v ok=%v, want external_image_idx", candidate, ok)
	}

	_, ok, err = selectImageVectorReuseCandidate([]catalogFileVectorReuseCandidate{
		{
			VectorTable: "external_image_idx",
			Meta:        sql.NullString{String: `{"index_modality":"image","image_embedding_model":"image-embedding","image_embedding_dimension":8}`, Valid: true},
		},
	}, kbVectorBinding{
		ImageVectorTable:        "kb_image_idx",
		ImageEmbeddingModel:     "image-embedding",
		ImageEmbeddingDimension: 4,
	})
	if err != nil {
		t.Fatalf("selectImageVectorReuseCandidate dimension mismatch: %v", err)
	}
	if ok {
		t.Fatal("selectImageVectorReuseCandidate ok = true for mismatched dimension")
	}

	_, ok, err = selectImageVectorReuseCandidate([]catalogFileVectorReuseCandidate{
		{
			VectorTable: "external_image_idx",
			Meta:        sql.NullString{String: `{"index_modality":"image","image_embedding_model":"image-embedding","image_embedding_dimension":4,"preprocess_version":"other-preprocess","distance_metric":"cosine"}`, Valid: true},
		},
	}, kbVectorBinding{
		ImageVectorTable:        "kb_image_idx",
		ImageEmbeddingModel:     "image-embedding",
		ImageEmbeddingDimension: 4,
		ImagePreprocessVersion:  "image-preprocess-v1",
		ImageDistanceMetric:     "cosine",
	})
	if err != nil {
		t.Fatalf("selectImageVectorReuseCandidate preprocess mismatch: %v", err)
	}
	if ok {
		t.Fatal("selectImageVectorReuseCandidate ok = true for mismatched preprocess")
	}
}

func TestTryReuseCatalogFileVectorsSkipsTextOnlyTargetWithImageSource(t *testing.T) {
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}

	tenantMock.ExpectQuery("SELECT vector\\.asset_ref, vector\\.meta").
		WithArgs("source-file", "source-file", "source-file", "source-file").
		WillReturnRows(sqlmock.NewRows([]string{"asset_ref", "meta", "root_asset_ref", "source_file_id", "parsed_file_id"}).
			AddRow("external_text_idx", `{"embedding_model":"text-embedding"}`, "source-file", "source-file", "source-file").
			AddRow("external_image_idx", `{"index_modality":"image"}`, "source-file", "source-file", "source-file"))

	ctx := ctxutil.WithTenantDB(context.Background(), tenantDB)
	reused, operationID, err := (&semanticModelService{}).tryReuseCatalogFileVectors(ctx, KnowledgeBaseSourceRecord{
		SourceID:     "source-1",
		ModelID:      77,
		SourceFileID: stringPtr("source-file"),
		KBFileID:     stringPtr("source-file"),
	}, kbVectorBinding{
		VectorTable:    "kb_text_idx",
		EmbeddingModel: "text-embedding",
	}, "user-1")
	if err != nil {
		t.Fatalf("tryReuseCatalogFileVectors: %v", err)
	}
	if reused || operationID != "" {
		t.Fatalf("tryReuseCatalogFileVectors = (%v, %q), want (false, empty)", reused, operationID)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func expectCatalogFileVectorReuseCandidatesEmpty(mock sqlmock.Sqlmock, fileID string) {
	mock.ExpectQuery("SELECT vector\\.asset_ref, vector\\.meta").
		WithArgs(fileID, fileID, fileID, fileID).
		WillReturnRows(sqlmock.NewRows([]string{"asset_ref", "meta", "root_asset_ref", "source_file_id", "parsed_file_id"}))
}

func expectCatalogFileTextVectorReuseSucceeded(mock sqlmock.Sqlmock, modelID int64, sourceID, fileID, sourceTable, targetTable, embeddingModel string, nestedTx bool) string {
	return expectCatalogFileTextVectorReuseSucceededForReusableFile(mock, modelID, sourceID, fileID, fileID, sourceTable, targetTable, embeddingModel, nestedTx)
}

func expectCatalogFileTextVectorReuseSucceededForReusableFile(mock sqlmock.Sqlmock, modelID int64, sourceID, kbFileID, reusableFileID, sourceTable, targetTable, embeddingModel string, nestedTx bool) string {
	mock.ExpectQuery("SELECT vector\\.asset_ref, vector\\.meta").
		WithArgs(kbFileID, kbFileID, kbFileID, kbFileID).
		WillReturnRows(sqlmock.NewRows([]string{"asset_ref", "meta", "root_asset_ref", "source_file_id", "parsed_file_id"}).
			AddRow(sourceTable, fmt.Sprintf(`{"embedding_model":%q}`, embeddingModel), reusableFileID, reusableFileID, kbFileID))
	expectVectorTableSchemaColumns(mock, sourceTable, "vecf32(3)")
	expectVectorTableSchemaColumns(mock, targetTable, "vecf32(3)")
	mock.ExpectQuery(fmt.Sprintf("SELECT MAX\\(index_version\\) FROM `%s` WHERE file_id = \\? AND index_version > 0", sourceTable)).
		WithArgs(reusableFileID).
		WillReturnRows(sqlmock.NewRows([]string{"MAX(index_version)"}).AddRow(int64(2)))
	mock.ExpectQuery(fmt.Sprintf("SELECT id, embedding, content, meta, level, chunk_index, index_version, NULL\\s+FROM `%s`", sourceTable)).
		WithArgs(reusableFileID, int64(2)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "embedding", "content", "meta", "level", "chunk_index", "index_version", "page_number"}).
			AddRow("text-row-0", "[0.1,0.2,0.3]", "text content", `{"chunk_id":"text-chunk-0"}`, kbSegmentLevelChunk, int64(0), int64(2), nil))

	targetSegment := kbSegmentRecord{
		KBFileID:     kbFileID,
		IndexVersion: 1,
		Level:        kbSegmentLevelChunk,
		ChunkIndex:   int64Ptr(0),
		IdentityKey:  "idx:0",
	}
	mock.ExpectQuery(fmt.Sprintf("SELECT embedding, meta FROM `%s`", targetTable)).
		WithArgs(segmentVectorRowID(targetSegment), kbFileID, int64(1), false).
		WillReturnRows(sqlmock.NewRows([]string{"embedding", "meta"}).
			AddRow("[0.1,0.2,0.3]", fmt.Sprintf(`{"vector_table":%q,"embedding_model":%q}`, sourceTable, embeddingModel)))

	versionID := stableID("kb-segver", sourceID, int64(1), kbSegmentSourceExternal)
	segmentID := stableID("kb-segment", versionID, kbSegmentLevelChunk, "idx:0")
	if nestedTx {
		mock.ExpectExec("SAVEPOINT").WillReturnResult(sqlmock.NewResult(0, 0))
	} else {
		mock.ExpectBegin()
	}
	mock.ExpectExec("INSERT INTO knowledge_base_segment_versions").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO [`\"]?knowledge_base_segments[`\"]?").
		WithArgs(
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), "user-1", true, "idx:0", sqlmock.AnyArg(), sqlmock.AnyArg(), int64(1), kbFileID, kbSegmentLevelChunk, sqlmock.AnyArg(), modelID, sqlmock.AnyArg(), sqlmock.AnyArg(), segmentID, sourceID, "user-1", versionID, sqlmock.AnyArg(),
		).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO [`\"]?knowledge_base_chunk_recall_stats[`\"]?").
		WithArgs(
			sqlmock.AnyArg(), sqlmock.AnyArg(), "idx:0", int64(1), kbFileID, kbSegmentLevelChunk, modelID, int64(0), sourceID,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("UPDATE knowledge_base_sources").
		WillReturnResult(sqlmock.NewResult(1, 1))
	if !nestedTx {
		mock.ExpectCommit()
	}
	return fmt.Sprintf("vector_reuse:%s:%s", sourceTable, targetTable)
}

func knowledgeBaseSourceJobRunRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"job_id", "source_id", "model_id", "job_type", "job_status", "idempotency_key", "operation_id",
		"workflow_execution_id", "runtime_actor_moi_user_id", "runtime_effective_role_id", "runtime_is_workspace_owner", "source_file_id", "kb_file_id", "source_table_id", "kb_table_id",
		"retry_count", "next_retry_at", "error", "created_at", "updated_at",
	})
}

func knowledgeBaseSourceJobRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "model_id", "source_type", "source_file_id", "kb_file_id", "raw_volume_id",
		"job_status", "error", "segment_version_id", "index_version", "workflow_execution_id",
	})
}

func expectLegacyBackfillNoop(mock sqlmock.Sqlmock, modelID int64) {
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*source_id.*FROM knowledge_base_sources").
		WithArgs(modelID).
		WillReturnRows(knowledgeBaseSourceRecordRows())
	mock.ExpectQuery("SELECT id, model_id, source_type, source_file_id").
		WithArgs(modelID).
		WillReturnRows(knowledgeBaseSourceJobRows())
	mock.ExpectQuery("SELECT model_id, catalog_id, database_id, raw_volume_id, processed_volume_id, ensure_status, last_ensure_error, last_checked_at").
		WithArgs(modelID).
		WillReturnRows(sqlmock.NewRows([]string{
			"model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "ensure_status", "last_ensure_error", "last_checked_at",
		}))
	mock.ExpectQuery("SELECT raw_volume_id\\s+FROM knowledge_base_raw_volumes\\s+WHERE model_id = \\? AND raw_volume_id > 0").
		WithArgs(modelID).
		WillReturnRows(sqlmock.NewRows([]string{"raw_volume_id"}))
	mock.ExpectQuery("SELECT raw_volume_id\\s+FROM knowledge_base_data_domains\\s+WHERE model_id = \\? AND raw_volume_id > 0").
		WithArgs(modelID).
		WillReturnRows(sqlmock.NewRows([]string{"raw_volume_id"}))
	mock.ExpectCommit()
}

func expectPendingSourceJobRunsEmpty(mock sqlmock.Sqlmock, modelID int64, jobType string) {
	args := []driver.Value{modelID, jobType, kbSourceJobQueued, kbSourceJobRunning}
	if jobType == kbJobTypeLoad {
		args = append(args, kbSourceJobSucceeded, kbSourceStatusRemoved, kbSourceTypeCatalogTable, kbSourceTypeLocalFile, kbRawKindStructured, kbSourceJobReconcileBatchSize)
	} else {
		args = append(args, kbSourceJobReconcileBatchSize)
	}
	mock.ExpectQuery("SELECT .*job_id.*source_id.*model_id.*job_type.*job_status.*idempotency_key").
		WithArgs(args...).
		WillReturnRows(knowledgeBaseSourceJobRunRows())
}

func expectPendingSourceJobRunsWithPendingEmpty(mock sqlmock.Sqlmock, modelID int64, jobType string, limits ...int) {
	limit := kbSourceJobReconcileBatchSize
	if len(limits) > 0 {
		limit = limits[0]
	}
	mock.ExpectQuery("SELECT .*job_id.*source_id.*model_id.*job_type.*job_status.*idempotency_key").
		WithArgs(modelID, jobType, kbSourceJobPending, kbSourceJobQueued, kbSourceJobRunning, limit).
		WillReturnRows(knowledgeBaseSourceJobRunRows())
}

func expectReusableVectorRowIdentityByChunkIndex(mock sqlmock.Sqlmock, tableName, fileID string, indexVersion int64, level string, chunkIndex int64, imageFileID ...string) *sqlmock.ExpectedQuery {
	query := fmt.Sprintf("SELECT embedding, meta FROM `%s` WHERE file_id = ? AND index_version = ? AND disabled = ? AND level = ? AND chunk_index = ?", tableName)
	args := []driver.Value{fileID, indexVersion, false, level, chunkIndex}
	if len(imageFileID) > 0 && imageFileID[0] != "" {
		query += " AND (JSON_EXTRACT(meta, '$.image_file_id') = ? OR JSON_EXTRACT(meta, '$.page_image_file_id') = ?)"
		args = append(args, imageFileID[0], imageFileID[0])
	}
	query += " LIMIT 2"
	return mock.ExpectQuery(regexp.QuoteMeta(query)).WithArgs(args...)
}

func expectReusableVectorRowIdentityByChunkID(mock sqlmock.Sqlmock, tableName, fileID string, indexVersion int64, level, chunkID, imageFileID string) *sqlmock.ExpectedQuery {
	query := fmt.Sprintf("SELECT embedding, meta FROM `%s` WHERE file_id = ? AND index_version = ? AND disabled = ? AND level = ? AND JSON_EXTRACT(meta, '$.chunk_id') = ?", tableName)
	args := []driver.Value{fileID, indexVersion, false, level, chunkID}
	if imageFileID != "" {
		query += " AND (JSON_EXTRACT(meta, '$.image_file_id') = ? OR JSON_EXTRACT(meta, '$.page_image_file_id') = ?)"
		args = append(args, imageFileID, imageFileID)
	}
	query += " LIMIT 2"
	return mock.ExpectQuery(regexp.QuoteMeta(query)).WithArgs(args...)
}

func expectInitialTextVectorLegacyBackfill(mock sqlmock.Sqlmock, tableName, fileID string, rowIDs ...string) {
	expectInitialTextVectorLegacyBackfillVersion(mock, tableName, fileID, 1, rowIDs...)
}

func expectInitialTextVectorLegacyBackfillVersion(mock sqlmock.Sqlmock, tableName, fileID string, indexVersion int64, rowIDs ...string) {
	quoted := "`" + tableName + "`"
	mock.ExpectExec("UPDATE "+regexp.QuoteMeta(quoted)+" SET index_version").
		WithArgs(indexVersion, fileID).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("UPDATE "+regexp.QuoteMeta(quoted)+" SET disabled").
		WithArgs(false, fileID).
		WillReturnResult(sqlmock.NewResult(0, 0))
	rows := sqlmock.NewRows([]string{"id"})
	for _, rowID := range rowIDs {
		rows.AddRow(rowID)
	}
	mock.ExpectQuery("SELECT id FROM " + regexp.QuoteMeta(quoted) + " WHERE file_id").
		WithArgs(fileID).
		WillReturnRows(rows)
	for idx, rowID := range rowIDs {
		mock.ExpectExec("UPDATE "+regexp.QuoteMeta(quoted)+" SET chunk_index").
			WithArgs(int64(idx), rowID).
			WillReturnResult(sqlmock.NewResult(0, 1))
	}
}

func expectEmptyLegacySourceJobs(mock sqlmock.Sqlmock, modelID int64) {
	mock.ExpectQuery("SELECT id, model_id, source_type, source_file_id").
		WithArgs(modelID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "model_id", "source_type", "source_file_id", "kb_file_id", "raw_volume_id", "job_status", "error", "segment_version_id", "index_version", "workflow_execution_id",
		}))
}

func expectEmptyKnowledgeBaseSourceCounts(mock sqlmock.Sqlmock, modelID int64) {
	mock.ExpectQuery("SELECT .*source_id.*FROM knowledge_base_sources").
		WithArgs(modelID).
		WillReturnRows(sqlmock.NewRows([]string{
			"source_id", "model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "source_type",
			"source_file_id", "source_table_id", "kb_file_id", "kb_table_id", "display_name", "source_path",
			"db_name", "table_name", "status", "error", "enabled", "expires_at", "tags", "force_enabled_after_expiry", "segment_version_id", "index_version",
			"size_bytes", "row_count", "created_by", "updated_by", "updated_at",
		}))
	expectEmptyLegacySourceJobs(mock, modelID)
	expectNoKnowledgeBaseRawVolumeFiles(mock, modelID)
}

func expectEmptySourceJobRuns(mock sqlmock.Sqlmock, modelID int64) {
	mock.ExpectQuery("SELECT .*job_id.*source_id.*model_id.*job_type.*job_status.*idempotency_key").
		WithArgs(modelID).
		WillReturnRows(sqlmock.NewRows([]string{
			"job_id", "source_id", "model_id", "job_type", "job_status", "idempotency_key", "operation_id",
			"workflow_execution_id", "runtime_actor_moi_user_id", "runtime_effective_role_id", "runtime_is_workspace_owner", "source_file_id", "kb_file_id", "source_table_id", "kb_table_id",
			"retry_count", "next_retry_at", "error", "created_at", "updated_at",
		}))
}

func expectSourceJobCandidates(mock sqlmock.Sqlmock, modelID int64, sourceIDs []string, jobs *sqlmock.Rows, records *sqlmock.Rows) {
	mock.ExpectQuery("SELECT COUNT\\(DISTINCT kbs\\.source_id\\).*FROM knowledge_base_source_job_runs jr").
		WithArgs(modelID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(len(sourceIDs)))
	mock.ExpectQuery("SELECT kbs\\.source_id.*FROM knowledge_base_source_job_runs jr").
		WithArgs(modelID, kbSourceJobListBatchSize).
		WillReturnRows(func() *sqlmock.Rows {
			rows := sqlmock.NewRows([]string{"source_id"})
			for _, sourceID := range sourceIDs {
				rows.AddRow(sourceID)
			}
			return rows
		}())
	if len(sourceIDs) == 0 {
		return
	}
	args := make([]driver.Value, 0, len(sourceIDs)+1)
	args = append(args, modelID)
	for _, sourceID := range sourceIDs {
		args = append(args, sourceID)
	}
	mock.ExpectQuery("SELECT .*job_id.*source_id.*model_id.*job_type.*job_status.*idempotency_key").
		WithArgs(args...).
		WillReturnRows(jobs)
	mock.ExpectQuery("SELECT .*source_id.*FROM knowledge_base_sources kbs").
		WithArgs(append(args, kbSourceStatusRemoved)...).
		WillReturnRows(records)
}

func expectNoLegacySourceJobReconcileWork(mock sqlmock.Sqlmock, modelID int64, vectorTables ...string) {
	mock.ExpectQuery("SELECT CASE WHEN EXISTS.*FROM knowledge_base_source_jobs legacy").
		WithArgs(modelID).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(0))
	expectNoKnowledgeBaseRawVolumeFiles(mock, modelID)
	if len(vectorTables) == 0 {
		return
	}
	args := make([]driver.Value, 0, len(vectorTables)+1)
	for _, table := range vectorTables {
		args = append(args, table)
	}
	args = append(args, modelID)
	mock.ExpectQuery("SELECT 1.*FROM data_asset vector").
		WithArgs(args...).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}))
}

func TestEnsureInitialImportVectorTableSchemaBackfillsLegacyColumns(t *testing.T) {
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	tenantMock.ExpectQuery("SELECT COLUMN_NAME, COLUMN_TYPE FROM information_schema\\.COLUMNS").
		WithArgs("kb_text_idx").
		WillReturnRows(sqlmock.NewRows([]string{"COLUMN_NAME", "COLUMN_TYPE"}).
			AddRow("id", "varchar(128)").
			AddRow("embedding", "vecf32(3)").
			AddRow("content", "text").
			AddRow("meta", "json").
			AddRow("file_id", "varchar(128)").
			AddRow("level", "varchar(64)"))
	tenantMock.ExpectExec("ALTER TABLE `kb_text_idx` ADD COLUMN chunk_index INT NULL").
		WillReturnResult(sqlmock.NewResult(0, 0))
	tenantMock.ExpectExec("ALTER TABLE `kb_text_idx` ADD COLUMN index_version BIGINT DEFAULT 0").
		WillReturnResult(sqlmock.NewResult(0, 0))
	tenantMock.ExpectExec("ALTER TABLE `kb_text_idx` ADD COLUMN disabled TINYINT\\(1\\) DEFAULT 0").
		WillReturnResult(sqlmock.NewResult(0, 0))
	tenantMock.ExpectExec("UPDATE `kb_text_idx` SET index_version").
		WithArgs(int64(1), "kb-file").
		WillReturnResult(sqlmock.NewResult(0, 2))
	tenantMock.ExpectExec("UPDATE `kb_text_idx` SET disabled").
		WithArgs(false, "kb-file").
		WillReturnResult(sqlmock.NewResult(0, 2))
	tenantMock.ExpectQuery("SELECT id FROM `kb_text_idx` WHERE file_id").
		WithArgs("kb-file").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("row-a").AddRow("row-b"))
	tenantMock.ExpectExec("UPDATE `kb_text_idx` SET chunk_index").
		WithArgs(int64(0), "row-a").
		WillReturnResult(sqlmock.NewResult(0, 1))
	tenantMock.ExpectExec("UPDATE `kb_text_idx` SET chunk_index").
		WithArgs(int64(1), "row-b").
		WillReturnResult(sqlmock.NewResult(0, 1))

	schema, err := ensureInitialImportVectorTableSchema(context.Background(), tenantDB, "kb_text_idx", "`kb_text_idx`", "kb-file", 1)
	if err != nil {
		t.Fatalf("ensureInitialImportVectorTableSchema: %v", err)
	}
	for _, column := range []string{"chunk_index", "index_version", "disabled"} {
		if _, ok := schema.Columns[column]; !ok {
			t.Fatalf("schema missing backfilled column %s: %+v", column, schema.Columns)
		}
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestImportSegmentsFromVectorRowsKeepsTextRowsTextOnly(t *testing.T) {
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	tenantMock.ExpectQuery("SELECT id, content, meta, level, chunk_index, index_version").
		WithArgs("kb-file", int64(3)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "content", "meta", "level", "chunk_index", "index_version"}).
			AddRow("row-text-1", "chunk text", `{"source_block_type":"IMAGE","s3_image_url":"image-file-1","page_image_file_id":"page-image-1","ocr_text":"ocr text","caption":"image caption","bbox":[1,2,3,4]}`, "chunk", int64(5), int64(3)))

	svc := &semanticModelService{}
	segments, err := svc.importSegmentsFromVectorRows(context.Background(), tenantDB, "`kb_text_idx`", "", KnowledgeBaseSourceRecord{KBFileID: stringPtr("kb-file")}, 3)
	if err != nil {
		t.Fatalf("importSegmentsFromVectorRows: %v", err)
	}
	if len(segments) != 1 {
		t.Fatalf("segments len = %d, want 1: %+v", len(segments), segments)
	}
	seg := segments[0]
	if seg.ChunkIndex == nil || *seg.ChunkIndex != 5 || seg.IdentityKey != "idx:5" {
		t.Fatalf("segment identity = chunk_index:%v identity:%q", seg.ChunkIndex, seg.IdentityKey)
	}
	if seg.Content == nil || *seg.Content != "chunk text" {
		t.Fatalf("segment content = %+v", seg.Content)
	}
	if seg.ImageFileID != nil {
		t.Fatalf("text segment image_file_id = %+v, want nil", seg.ImageFileID)
	}
	if seg.PageImageFileID != nil {
		t.Fatalf("text segment page_image_file_id = %+v, want nil", seg.PageImageFileID)
	}
	if seg.OCRText != nil {
		t.Fatalf("text segment ocr_text = %+v, want nil", seg.OCRText)
	}
	if seg.ImageDescription != nil {
		t.Fatalf("text segment image_description = %+v, want nil", seg.ImageDescription)
	}
	if len(seg.BBox) != 0 {
		t.Fatalf("text segment bbox = %s, want empty", string(seg.BBox))
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestImportSegmentsFromVectorRowsImportsTextAndImageRowsSeparately(t *testing.T) {
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	tenantMock.ExpectQuery("SELECT id, content, meta, level, chunk_index, index_version").
		WithArgs("kb-file", int64(4)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "content", "meta", "level", "chunk_index", "index_version"}).
			AddRow("row-text-2", "text content", `{"chunk_id":"text-chunk"}`, "chunk", int64(2), int64(4)))
	tenantMock.ExpectQuery("SELECT id, content, meta, level, chunk_index, index_version").
		WithArgs("kb-file", int64(4)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "content", "meta", "level", "chunk_index", "index_version"}).
			AddRow("row-image-2", "", `{"segment_type":"image","chunk_id":"document_visual_image:visual_object:obj-2","image_file_id":"image-file-2","page_image_file_id":"page-image-2","image_description":"image desc","bbox":[5,6,7,8]}`, "chunk", nil, int64(4)))

	svc := &semanticModelService{}
	segments, err := svc.importSegmentsFromVectorRows(context.Background(), tenantDB, "`kb_text_idx`", "`kb_image_idx`", KnowledgeBaseSourceRecord{KBFileID: stringPtr("kb-file")}, 4)
	if err != nil {
		t.Fatalf("importSegmentsFromVectorRows: %v", err)
	}
	if len(segments) != 2 {
		t.Fatalf("segments len = %d, want 2: %+v", len(segments), segments)
	}
	textSeg := segments[0]
	if textSeg.Content == nil || *textSeg.Content != "text content" {
		t.Fatalf("text segment content = %+v", textSeg.Content)
	}
	imageSeg := segments[1]
	if imageSeg.ChunkIndex != nil || imageSeg.ChunkID == nil || *imageSeg.ChunkID != "document_visual_image:visual_object:obj-2" {
		t.Fatalf("image segment identity chunk_index=%+v chunk_id=%+v", imageSeg.ChunkIndex, imageSeg.ChunkID)
	}
	if imageSeg.ImageFileID == nil || *imageSeg.ImageFileID != "image-file-2" {
		t.Fatalf("image segment image_file_id = %+v", imageSeg.ImageFileID)
	}
	if imageSeg.PageImageFileID == nil || *imageSeg.PageImageFileID != "page-image-2" {
		t.Fatalf("image segment page_image_file_id = %+v", imageSeg.PageImageFileID)
	}
	if imageSeg.ImageDescription == nil || *imageSeg.ImageDescription != "image desc" {
		t.Fatalf("image segment image_description = %+v", imageSeg.ImageDescription)
	}
	if string(imageSeg.BBox) != `[5,6,7,8]` {
		t.Fatalf("image segment bbox = %s", string(imageSeg.BBox))
	}
	segmentType, _, _ := documentSegmentCanonicalMetadata(imageSeg.Metadata)
	if segmentType != "image" {
		t.Fatalf("materialized image segment type = %q, want image; metadata=%s", segmentType, imageSeg.Metadata)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestImportSegmentsFromVectorRowsRejectsImageRowWithoutIdentity(t *testing.T) {
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	tenantMock.ExpectQuery("SELECT id, content, meta, level, chunk_index, index_version").
		WithArgs("kb-file", int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "content", "meta", "level", "chunk_index", "index_version"}))
	tenantMock.ExpectQuery("SELECT id, content, meta, level, chunk_index, index_version").
		WithArgs("kb-file", int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "content", "meta", "level", "chunk_index", "index_version"}).
			AddRow("image-row-no-identity", "nearby text", `{"image_file_id":"image-file","page_image_file_id":"page-image","caption":"caption","page_number":7}`, "chunk", nil, int64(1)))

	svc := &semanticModelService{}
	_, err = svc.importSegmentsFromVectorRows(context.Background(), tenantDB, "`kb_text_idx`", "`kb_image_idx`", KnowledgeBaseSourceRecord{KBFileID: stringPtr("kb-file")}, 1)
	if err == nil || !strings.Contains(err.Error(), i18n.KeySessionSegmentIdentityRequired.String()) {
		t.Fatalf("importSegmentsFromVectorRows error = %v, want missing identity", err)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestImportInitialSegmentsFromVectorRowsRejectsPendingSource(t *testing.T) {
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	tenantMock.ExpectQuery("SELECT .*job_id.*source_id.*model_id.*job_type.*job_status.*idempotency_key").
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{
			"job_id", "source_id", "model_id", "job_type", "job_status", "idempotency_key", "operation_id",
			"workflow_execution_id", "runtime_actor_moi_user_id", "runtime_effective_role_id", "runtime_is_workspace_owner", "source_file_id", "kb_file_id", "source_table_id", "kb_table_id",
			"retry_count", "next_retry_at", "error", "created_at", "updated_at",
		}).AddRow("job-rag-1", "source-file-1", int64(7), kbJobTypeRAGIngest, kbSourceJobPending, "key", nil, nil, nil, nil, false, stringPtr("kb-file"), stringPtr("kb-file"), nil, nil, int64(0), nil, nil, int64(1), int64(1)))

	svc := &semanticModelService{}
	ctx := ctxutil.WithTenantDB(context.Background(), tenantDB)
	_, _, err = svc.importInitialSegmentsFromVectorRows(ctx, nil, nil, KnowledgeBaseSourceRecord{
		SourceID: "source-file-1",
		ModelID:  int64(7),
		Status:   kbSourceStatusPending,
		KBFileID: stringPtr("kb-file"),
	}, kbVectorBinding{
		VectorTable:    "kb_text_idx",
		EmbeddingModel: "embed-model",
	})

	var serviceErr *ServiceError
	if !errors.As(err, &serviceErr) || serviceErr.Code != ErrCodeBadRequest || !strings.Contains(serviceErr.Err.Error(), i18n.KeySessionSourceParsingIncomplete.String()) {
		t.Fatalf("importInitialSegmentsFromVectorRows error = %v, want source parsing bad request", err)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestImportInitialSegmentsFromVectorRowsAllowsSucceededRAGJobWhenSourceStatusStale(t *testing.T) {
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	tenantMock.ExpectQuery("SELECT .*job_id.*source_id.*model_id.*job_type.*job_status.*idempotency_key").
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{
			"job_id", "source_id", "model_id", "job_type", "job_status", "idempotency_key", "operation_id",
			"workflow_execution_id", "runtime_actor_moi_user_id", "runtime_effective_role_id", "runtime_is_workspace_owner", "source_file_id", "kb_file_id", "source_table_id", "kb_table_id",
			"retry_count", "next_retry_at", "error", "created_at", "updated_at",
		}).
			AddRow("job-load-1", "source-file-1", int64(7), kbJobTypeLoad, kbSourceJobSucceeded, "key-load", nil, nil, nil, nil, false, stringPtr("kb-file"), stringPtr("kb-file"), nil, nil, int64(0), nil, nil, int64(1), int64(1)).
			AddRow("job-rag-1", "source-file-1", int64(7), kbJobTypeRAGIngest, kbSourceJobSucceeded, "key-rag", nil, nil, nil, nil, false, stringPtr("kb-file"), stringPtr("kb-file"), nil, nil, int64(0), nil, nil, int64(1), int64(1)))
	expectVectorTableSchemaColumns(tenantMock, "kb_text_idx", "vecf32(3)")
	expectInitialTextVectorLegacyBackfill(tenantMock, "kb_text_idx", "kb-file")
	tenantMock.ExpectQuery("SELECT MAX\\(index_version\\) FROM `kb_text_idx`").
		WithArgs("kb-file").
		WillReturnRows(sqlmock.NewRows([]string{"max"}).AddRow(int64(4)))
	tenantMock.ExpectQuery("SELECT id, content, meta, level, chunk_index, index_version").
		WithArgs("kb-file", int64(4)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "content", "meta", "level", "chunk_index", "index_version"}).
			AddRow("row-text-1", "chunk text", `{"chunk_id":"text-chunk-1"}`, "chunk", nil, int64(4)))

	svc := &semanticModelService{}
	ctx := ctxutil.WithTenantDB(context.Background(), tenantDB)
	segments, indexVersion, err := svc.importInitialSegmentsFromVectorRows(ctx, nil, nil, KnowledgeBaseSourceRecord{
		SourceID: "source-file-1",
		ModelID:  int64(7),
		Status:   kbSourceStatusPending,
		KBFileID: stringPtr("kb-file"),
	}, kbVectorBinding{
		VectorTable:    "kb_text_idx",
		EmbeddingModel: "embed-model",
	})

	if err != nil {
		t.Fatalf("importInitialSegmentsFromVectorRows: %v", err)
	}
	if indexVersion != 4 || len(segments) != 1 {
		t.Fatalf("indexVersion=%d segments=%+v", indexVersion, segments)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestImportInitialSegmentsFromVectorRowsAllowsRefreshedSucceededRAGJobWhenSourceStatusStale(t *testing.T) {
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	workflowID := knowledgeBaseWorkflowID("ws-1", 7)
	tenantMock.ExpectQuery("SELECT .*job_id.*source_id.*model_id.*job_type.*job_status.*idempotency_key").
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{
			"job_id", "source_id", "model_id", "job_type", "job_status", "idempotency_key", "operation_id",
			"workflow_execution_id", "runtime_actor_moi_user_id", "runtime_effective_role_id", "runtime_is_workspace_owner", "source_file_id", "kb_file_id", "source_table_id", "kb_table_id",
			"retry_count", "next_retry_at", "error", "created_at", "updated_at",
		}).
			AddRow("job-load-1", "source-file-1", int64(7), kbJobTypeLoad, kbSourceJobSucceeded, "key-load", nil, nil, nil, nil, false, stringPtr("kb-file"), stringPtr("kb-file"), nil, nil, int64(0), nil, nil, int64(1), int64(1)).
			AddRow("job-rag-1", "source-file-1", int64(7), kbJobTypeRAGIngest, kbSourceJobPending, "key-rag", stringPtr("workflow_trigger:"+workflowID), nil, nil, nil, false, stringPtr("kb-file"), stringPtr("kb-file"), nil, nil, int64(0), nil, nil, int64(1), int64(1)))
	expectVectorTableSchemaColumns(tenantMock, "kb_text_idx", "vecf32(3)")
	expectInitialTextVectorLegacyBackfill(tenantMock, "kb_text_idx", "kb-file")
	tenantMock.ExpectQuery("SELECT MAX\\(index_version\\) FROM `kb_text_idx`").
		WithArgs("kb-file").
		WillReturnRows(sqlmock.NewRows([]string{"max"}).AddRow(int64(4)))
	tenantMock.ExpectQuery("SELECT id, content, meta, level, chunk_index, index_version").
		WithArgs("kb-file", int64(4)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "content", "meta", "level", "chunk_index", "index_version"}).
			AddRow("row-text-1", "chunk text", `{"chunk_id":"text-chunk-1"}`, "chunk", nil, int64(4)))

	workflowSvc := &fakeSemanticModelWorkflowService{fileExecutions: map[string]*moi.FileExecutionsResponse{
		"kb-file": {
			Executions: []moi.FileExecutionSummary{
				{ExecutionID: "exec-rag-new", WorkflowID: workflowID, Status: "succeeded"},
			},
			Total: 1,
		},
	}}
	svc := &semanticModelService{workflowService: workflowSvc}
	ctx := ctxutil.WithTenantDB(context.Background(), tenantDB)
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	segments, indexVersion, err := svc.importInitialSegmentsFromVectorRows(ctx, nil, nil, KnowledgeBaseSourceRecord{
		SourceID: "source-file-1",
		ModelID:  int64(7),
		Status:   kbSourceStatusPending,
		KBFileID: stringPtr("kb-file"),
	}, kbVectorBinding{
		VectorTable:    "kb_text_idx",
		EmbeddingModel: "embed-model",
	})

	if err != nil {
		t.Fatalf("importInitialSegmentsFromVectorRows: %v", err)
	}
	if indexVersion != 4 || len(segments) != 1 {
		t.Fatalf("indexVersion=%d segments=%+v", indexVersion, segments)
	}
	if len(workflowSvc.listFileExecutionCalls) != 1 || workflowSvc.listFileExecutionCalls[0] != "kb-file" {
		t.Fatalf("ListFileExecutions calls = %+v", workflowSvc.listFileExecutionCalls)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestImportInitialSegmentsFromVectorRowsRejectsUnavailableImageVectorTable(t *testing.T) {
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	expectVectorTableSchemaColumns(tenantMock, "kb_text_idx", "vecf32(3)")
	expectInitialTextVectorLegacyBackfill(tenantMock, "kb_text_idx", "kb-file")
	tenantMock.ExpectQuery("SELECT MAX\\(index_version\\) FROM `kb_text_idx`").
		WithArgs("kb-file").
		WillReturnRows(sqlmock.NewRows([]string{"max"}).AddRow(int64(3)))
	tenantMock.ExpectQuery("SELECT id, content, meta, level, chunk_index, index_version").
		WithArgs("kb-file", int64(3)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "content", "meta", "level", "chunk_index", "index_version"}).
			AddRow("row-text-1", "chunk text", `{"chunk_id":"text-chunk-1"}`, "chunk", nil, int64(3)))
	tenantMock.ExpectQuery("SELECT COLUMN_NAME, COLUMN_TYPE FROM information_schema\\.COLUMNS").
		WithArgs("kb_image_idx").
		WillReturnRows(sqlmock.NewRows([]string{"COLUMN_NAME", "COLUMN_TYPE"}))

	svc := &semanticModelService{}
	ctx := ctxutil.WithTenantDB(context.Background(), tenantDB)
	segments, indexVersion, err := svc.importInitialSegmentsFromVectorRows(ctx, nil, nil, KnowledgeBaseSourceRecord{
		SourceID: "source-file-1",
		ModelID:  int64(7),
		Status:   kbSourceStatusSucceeded,
		KBFileID: stringPtr("kb-file"),
	}, kbVectorBinding{
		VectorTable:         "kb_text_idx",
		EmbeddingModel:      "embed-model",
		ImageVectorTable:    "kb_image_idx",
		ImageEmbeddingModel: "image-embed",
	})

	if err == nil || !strings.Contains(err.Error(), i18n.KeySessionVectorTableUnavailable.String()) {
		t.Fatalf("importInitialSegmentsFromVectorRows error = %v, indexVersion=%d segments=%+v", err, indexVersion, segments)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestImportInitialSegmentsFromVectorRowsUsesTextIndexVersionWhenImageTableHasNoMatchingRows(t *testing.T) {
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	expectVectorTableSchemaColumns(tenantMock, "kb_text_idx", "vecf32(3)")
	expectInitialTextVectorLegacyBackfill(tenantMock, "kb_text_idx", "kb-file")
	tenantMock.ExpectQuery("SELECT MAX\\(index_version\\) FROM `kb_text_idx`").
		WithArgs("kb-file").
		WillReturnRows(sqlmock.NewRows([]string{"max"}).AddRow(int64(3)))
	tenantMock.ExpectQuery("SELECT id, content, meta, level, chunk_index, index_version").
		WithArgs("kb-file", int64(3)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "content", "meta", "level", "chunk_index", "index_version"}).
			AddRow("row-text-1", "chunk text", `{"chunk_id":"text-chunk-1"}`, "chunk", nil, int64(3)))
	expectVectorTableSchemaColumns(tenantMock, "kb_image_idx", "vecf32(4)")
	tenantMock.ExpectQuery("SELECT id, content, meta, level, chunk_index, index_version").
		WithArgs("kb-file", int64(3)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "content", "meta", "level", "chunk_index", "index_version"}))
	tenantMock.ExpectQuery("SELECT id, embedding, content, meta, page_number, level").
		WithArgs("kb-file").
		WillReturnRows(sqlmock.NewRows([]string{"id", "embedding", "content", "meta", "page_number", "level"}))

	svc := &semanticModelService{}
	ctx := ctxutil.WithTenantDB(context.Background(), tenantDB)
	segments, indexVersion, err := svc.importInitialSegmentsFromVectorRows(ctx, nil, nil, KnowledgeBaseSourceRecord{
		SourceID: "source-file-1",
		ModelID:  int64(7),
		Status:   kbSourceStatusSucceeded,
		KBFileID: stringPtr("kb-file"),
	}, kbVectorBinding{
		VectorTable:         "kb_text_idx",
		EmbeddingModel:      "embed-model",
		ImageVectorTable:    "kb_image_idx",
		ImageEmbeddingModel: "image-embed",
	})

	if err != nil {
		t.Fatalf("importInitialSegmentsFromVectorRows: %v", err)
	}
	if indexVersion != 3 || len(segments) != 1 {
		t.Fatalf("indexVersion=%d segments=%+v", indexVersion, segments)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestImportInitialSegmentsFromVectorRowsBackfillsLegacyZeroIndexRows(t *testing.T) {
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	expectVectorTableSchemaColumns(tenantMock, "kb_text_idx", "vecf32(3)")
	expectInitialTextVectorLegacyBackfill(tenantMock, "kb_text_idx", "kb-file")
	tenantMock.ExpectQuery("SELECT MAX\\(index_version\\) FROM `kb_text_idx`").
		WithArgs("kb-file").
		WillReturnRows(sqlmock.NewRows([]string{"max"}).AddRow(int64(1)))
	tenantMock.ExpectQuery("SELECT id, content, meta, level, chunk_index, index_version").
		WithArgs("kb-file", int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "content", "meta", "level", "chunk_index", "index_version"}).
			AddRow("legacy-row-1", "legacy chunk", `{"chunk_id":"legacy-chunk-1"}`, "chunk", nil, int64(1)))

	svc := &semanticModelService{}
	ctx := ctxutil.WithTenantDB(context.Background(), tenantDB)
	segments, indexVersion, err := svc.importInitialSegmentsFromVectorRows(ctx, nil, nil, KnowledgeBaseSourceRecord{
		SourceID:     "source-file-1",
		ModelID:      int64(7),
		Status:       kbSourceStatusSucceeded,
		KBFileID:     stringPtr("kb-file"),
		IndexVersion: int64Ptr(0),
	}, kbVectorBinding{
		VectorTable:    "kb_text_idx",
		EmbeddingModel: "embed-model",
	})

	if err != nil {
		t.Fatalf("importInitialSegmentsFromVectorRows: %v", err)
	}
	if indexVersion != 1 || len(segments) != 1 {
		t.Fatalf("indexVersion=%d segments=%+v", indexVersion, segments)
	}
	if segments[0].ChunkID == nil || *segments[0].ChunkID != "legacy-chunk-1" {
		t.Fatalf("legacy segment identity = %+v", segments[0])
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestImportInitialSegmentsFromVectorRowsBackfillsLegacyRowsToRecordIndexVersion(t *testing.T) {
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	expectVectorTableSchemaColumns(tenantMock, "kb_text_idx", "vecf32(3)")
	expectInitialTextVectorLegacyBackfillVersion(tenantMock, "kb_text_idx", "kb-file", 3)
	tenantMock.ExpectQuery("SELECT id, content, meta, level, chunk_index, index_version").
		WithArgs("kb-file", int64(3)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "content", "meta", "level", "chunk_index", "index_version"}).
			AddRow("legacy-row-3", "legacy chunk", `{"chunk_id":"legacy-chunk-3"}`, "chunk", nil, int64(3)))

	svc := &semanticModelService{}
	ctx := ctxutil.WithTenantDB(context.Background(), tenantDB)
	segments, indexVersion, err := svc.importInitialSegmentsFromVectorRows(ctx, nil, nil, KnowledgeBaseSourceRecord{
		SourceID:     "source-file-1",
		ModelID:      int64(7),
		Status:       kbSourceStatusSucceeded,
		KBFileID:     stringPtr("kb-file"),
		IndexVersion: int64Ptr(3),
	}, kbVectorBinding{
		VectorTable:    "kb_text_idx",
		EmbeddingModel: "embed-model",
	})

	if err != nil {
		t.Fatalf("importInitialSegmentsFromVectorRows: %v", err)
	}
	if indexVersion != 3 || len(segments) != 1 {
		t.Fatalf("indexVersion=%d segments=%+v", indexVersion, segments)
	}
	if segments[0].ChunkID == nil || *segments[0].ChunkID != "legacy-chunk-3" {
		t.Fatalf("legacy segment identity = %+v", segments[0])
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestImportExternalWorkflowSegmentsFromVectorRowsUsesCanonicalSourceFileID(t *testing.T) {
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	expectVectorTableSchemaColumns(tenantMock, "kb_text_idx", "vecf32(3)")
	tenantMock.ExpectQuery("SELECT MAX\\(index_version\\) FROM `kb_text_idx`").
		WithArgs("source-file").
		WillReturnRows(sqlmock.NewRows([]string{"max"}).AddRow(int64(4)))
	tenantMock.ExpectQuery("SELECT id, content, meta, level, chunk_index, index_version").
		WithArgs("source-file", int64(4)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "content", "meta", "level", "chunk_index", "index_version"}).
			AddRow("row-text-1", "workflow chunk", `{"chunk_id":"text-chunk-4"}`, "chunk", nil, int64(4)))
	expectVectorTableSchemaColumns(tenantMock, "kb_image_idx", "vecf32(4)")
	tenantMock.ExpectQuery("SELECT id, content, meta, level, chunk_index, index_version").
		WithArgs("source-file", int64(4)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "content", "meta", "level", "chunk_index", "index_version"}))

	svc := &semanticModelService{}
	ctx := ctxutil.WithTenantDB(context.Background(), tenantDB)
	segments, indexVersion, err := svc.importExternalWorkflowSegmentsFromVectorRows(ctx, KnowledgeBaseSourceRecord{
		SourceID:     "source-file-1",
		ModelID:      int64(7),
		SourceFileID: stringPtr("source-file"),
		KBFileID:     stringPtr("kb-file"),
		IndexVersion: int64Ptr(3),
	}, kbVectorBinding{
		VectorTable:         "kb_text_idx",
		EmbeddingModel:      "embed-model",
		ImageVectorTable:    "kb_image_idx",
		ImageEmbeddingModel: "image-embed",
	})
	if err != nil {
		t.Fatalf("importExternalWorkflowSegmentsFromVectorRows: %v", err)
	}
	if indexVersion != 4 || len(segments) != 1 {
		t.Fatalf("indexVersion=%d segments=%+v", indexVersion, segments)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestRepairLegacyImageRowsForInitialImportCopiesVersionedRows(t *testing.T) {
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	tenantMock.ExpectQuery("SELECT id, embedding, content, meta, page_number, level").
		WithArgs("kb-file").
		WillReturnRows(sqlmock.NewRows([]string{"id", "embedding", "content", "meta", "page_number", "level"}).
			AddRow("legacy-image-row", "[0.1,0.2,0.3]", "nearby text", `{"asset_kind":"document_visual","scope":"visual_object","object_id":"obj-1","image_file_id":"image-file-1","page_image_file_id":"page-image-1","chunk_index":123,"caption":"caption"}`, int64(7), "chunk"))
	tenantMock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM `kb_image_idx`").
		WithArgs("kb-file", int64(9)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(0)))
	expectVectorTableSchemaColumns(tenantMock, "kb_image_idx", "vecf32(3)")
	tenantMock.ExpectBegin()
	tenantMock.ExpectExec("INSERT INTO `kb_image_idx`").
		WithArgs(sqlmock.AnyArg(), "[0.1,0.2,0.3]", "nearby text", sqlmock.AnyArg(), "kb-file", sqlmock.AnyArg(), "chunk", nil, nil, nil, int64(9), false).
		WillReturnResult(sqlmock.NewResult(1, 1))
	tenantMock.ExpectCommit()

	repaired, err := repairLegacyImageRowsForInitialImport(context.Background(), tenantDB, "kb_image_idx", "`kb_image_idx`", KnowledgeBaseSourceRecord{KBFileID: stringPtr("kb-file")}, 9)
	if err != nil {
		t.Fatalf("repairLegacyImageRowsForInitialImport: %v", err)
	}
	if !repaired {
		t.Fatalf("repairLegacyImageRowsForInitialImport repaired = false, want true")
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestRepairLegacyImageRowsForInitialImportRejectsBadLegacyRowsBeforeExistingCheck(t *testing.T) {
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	tenantMock.ExpectQuery("SELECT id, embedding, content, meta, page_number, level").
		WithArgs("kb-file").
		WillReturnRows(sqlmock.NewRows([]string{"id", "embedding", "content", "meta", "page_number", "level"}).
			AddRow("legacy-image-row", "[0.1,0.2,0.3]", "nearby text", `{"asset_kind":"document_visual","scope":"visual_object","image_file_id":"image-file-1"}`, int64(7), "chunk"))

	repaired, err := repairLegacyImageRowsForInitialImport(context.Background(), tenantDB, "kb_image_idx", "`kb_image_idx`", KnowledgeBaseSourceRecord{KBFileID: stringPtr("kb-file")}, 9)
	if err == nil || !strings.Contains(err.Error(), i18n.KeySessionLegacyImageVectorIdentityMissing.String()) {
		t.Fatalf("repairLegacyImageRowsForInitialImport error = %v, want missing identity", err)
	}
	if repaired {
		t.Fatalf("repairLegacyImageRowsForInitialImport repaired = true, want false")
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceUpdateSourceGovernanceSyncsVectorDisabled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/workspaces/ws-1/semantic-models/7" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		requireSemanticModelExecutionHeaders(t, r)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":   7,
			"name": "kb",
			"files": map[string]any{
				"file_ids":           []string{"kb-file"},
				"vector_table":       "kb_text_idx",
				"image_vector_table": "kb_image_idx",
			},
		})
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()

	svc := newSemanticModelTestService(t, server.URL, systemClient)
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	tenantMock.ExpectQuery("SELECT .*source_id.*FROM knowledge_base_sources").
		WithArgs(int64(7), "source-file-1").
		WillReturnRows(sqlmock.NewRows([]string{
			"source_id", "model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "source_type",
			"source_file_id", "source_table_id", "kb_file_id", "kb_table_id", "display_name", "source_path",
			"db_name", "table_name", "status", "error", "enabled", "expires_at", "tags", "force_enabled_after_expiry", "segment_version_id", "index_version",
		}).AddRow("source-file-1", int64(7), int64(3), int64(11), int64(12), int64(13), kbSourceTypeCatalogFile, "source-file", nil, "kb-file", nil, "doc.pdf", nil, nil, nil, kbSourceStatusSucceeded, nil, true, nil, nil, false, nil, int64(2)))
	tenantMock.ExpectBegin()
	tenantMock.ExpectExec("UPDATE knowledge_base_sources").
		WillReturnResult(sqlmock.NewResult(0, 1))
	tenantMock.ExpectExec("UPDATE `kb_text_idx` SET disabled = \\? WHERE file_id = \\?").
		WithArgs(true, "kb-file").
		WillReturnResult(sqlmock.NewResult(0, 1))
	tenantMock.ExpectExec("UPDATE `kb_image_idx` SET disabled = \\? WHERE file_id = \\?").
		WithArgs(true, "kb-file").
		WillReturnResult(sqlmock.NewResult(0, 1))
	tenantMock.ExpectCommit()

	enabled := false
	expiresAt := time.Now().Add(time.Hour).Unix()
	tags := []string{"finance", "policy"}
	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)
	ctx = ctxutil.WithUserID(ctx, "user-1")

	result, err := svc.UpdateSourceGovernance(ctx, UpdateSemanticModelSourceGovernanceParams{
		ModelID:   7,
		SourceID:  "source-file-1",
		Tags:      &tags,
		ExpiresAt: OptionalInt64{Set: true, Value: &expiresAt},
		Enabled:   &enabled,
	})
	if err != nil {
		t.Fatalf("UpdateSourceGovernance: %v", err)
	}
	if result.Source.Enabled == nil || *result.Source.Enabled || result.Source.EffectiveEnabled {
		t.Fatalf("updated source enabled/effective = %+v", result.Source)
	}
	if !sameStringSet(result.Source.Tags, tags) || result.Source.ExpiresAt == nil || *result.Source.ExpiresAt != expiresAt {
		t.Fatalf("updated source tags/expires = %+v", result.Source)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceUpdateSourceGovernanceTagsDoNotRequireVectorTables(t *testing.T) {
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	tenantMock.ExpectBegin()
	tenantMock.ExpectExec("UPDATE knowledge_base_sources").
		WillReturnResult(sqlmock.NewResult(0, 1))
	tenantMock.ExpectCommit()

	tags := []string{"finance", "policy"}
	tagsJSON := `["finance","policy"]`
	record := KnowledgeBaseSourceRecord{
		SourceID:   "source-file-1",
		ModelID:    7,
		SourceType: kbSourceTypeCatalogFile,
		KBFileID:   stringPtr("kb-file"),
		Enabled:    boolPtr(true),
	}
	ctx := ctxutil.WithTenantDB(context.Background(), tenantDB)

	result, err := (&semanticModelService{}).updateKnowledgeBaseSourceGovernance(
		ctx,
		record,
		UpdateSemanticModelSourceGovernanceParams{Tags: &tags},
		&tagsJSON,
		json.RawMessage(`{"vector_table":"missing_text_idx","image_vector_table":"missing_image_idx"}`),
		"user-1",
	)
	if err != nil {
		t.Fatalf("updateKnowledgeBaseSourceGovernance: %v", err)
	}
	updatedSource := sourceRecordToSemanticModelSource(result)
	if !sameStringSet(updatedSource.Tags, tags) {
		t.Fatalf("updated source tags = %#v, want %#v", updatedSource.Tags, tags)
	}
	if result.Enabled == nil || !*result.Enabled {
		t.Fatalf("updated source enabled = %#v, want true", result.Enabled)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceUpdateSourceGovernanceSupportsQualifiedVectorTables(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/workspaces/ws-1/semantic-models/7" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":   7,
			"name": "kb",
			"files": map[string]any{
				"file_ids":           []string{"kb-file"},
				"vector_table":       "idx_db.kb_text_idx",
				"image_vector_table": "idx_db.kb_image_idx",
			},
		})
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()

	svc := newSemanticModelTestService(t, server.URL, systemClient)
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	tenantMock.ExpectQuery("SELECT .*source_id.*FROM knowledge_base_sources").
		WithArgs(int64(7), "source-file-1").
		WillReturnRows(sqlmock.NewRows([]string{
			"source_id", "model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "source_type",
			"source_file_id", "source_table_id", "kb_file_id", "kb_table_id", "display_name", "source_path",
			"db_name", "table_name", "status", "error", "enabled", "expires_at", "tags", "force_enabled_after_expiry", "segment_version_id", "index_version",
		}).AddRow("source-file-1", int64(7), int64(3), int64(11), int64(12), int64(13), kbSourceTypeCatalogFile, "source-file", nil, "kb-file", nil, "doc.pdf", nil, nil, nil, kbSourceStatusSucceeded, nil, true, nil, nil, false, nil, int64(2)))
	tenantMock.ExpectBegin()
	tenantMock.ExpectExec("UPDATE knowledge_base_sources").
		WillReturnResult(sqlmock.NewResult(0, 1))
	tenantMock.ExpectExec("UPDATE `idx_db`\\.`kb_text_idx` SET disabled = \\? WHERE file_id = \\?").
		WithArgs(true, "kb-file").
		WillReturnResult(sqlmock.NewResult(0, 1))
	tenantMock.ExpectExec("UPDATE `idx_db`\\.`kb_image_idx` SET disabled = \\? WHERE file_id = \\?").
		WithArgs(true, "kb-file").
		WillReturnResult(sqlmock.NewResult(0, 1))
	tenantMock.ExpectCommit()

	enabled := false
	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)

	result, err := svc.UpdateSourceGovernance(ctx, UpdateSemanticModelSourceGovernanceParams{
		ModelID:  7,
		SourceID: "source-file-1",
		Enabled:  &enabled,
	})
	if err != nil {
		t.Fatalf("UpdateSourceGovernance: %v", err)
	}
	if result.Source.EffectiveEnabled {
		t.Fatalf("updated source effective = %+v", result.Source)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceUpdateSourceGovernanceFailsWhenVectorSyncFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/workspaces/ws-1/semantic-models/7" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":   7,
			"name": "kb",
			"files": map[string]any{
				"file_ids":           []string{"kb-file"},
				"vector_table":       "kb_text_idx",
				"image_vector_table": "kb_image_idx",
			},
		})
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()

	svc := newSemanticModelTestService(t, server.URL, systemClient)
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	tenantMock.ExpectQuery("SELECT .*source_id.*FROM knowledge_base_sources").
		WithArgs(int64(7), "source-file-1").
		WillReturnRows(sqlmock.NewRows([]string{
			"source_id", "model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "source_type",
			"source_file_id", "source_table_id", "kb_file_id", "kb_table_id", "display_name", "source_path",
			"db_name", "table_name", "status", "error", "enabled", "expires_at", "tags", "force_enabled_after_expiry", "segment_version_id", "index_version",
		}).AddRow("source-file-1", int64(7), int64(3), int64(11), int64(12), int64(13), kbSourceTypeCatalogFile, "source-file", nil, "kb-file", nil, "doc.pdf", nil, nil, nil, kbSourceStatusSucceeded, nil, true, nil, nil, false, nil, int64(2)))
	tenantMock.ExpectBegin()
	tenantMock.ExpectExec("UPDATE knowledge_base_sources").
		WillReturnResult(sqlmock.NewResult(0, 1))
	tenantMock.ExpectExec("UPDATE `kb_text_idx` SET disabled = \\? WHERE file_id = \\?").
		WithArgs(true, "kb-file").
		WillReturnResult(sqlmock.NewResult(0, 1))
	tenantMock.ExpectExec("UPDATE `kb_image_idx` SET disabled = \\? WHERE file_id = \\?").
		WithArgs(true, "kb-file").
		WillReturnError(errors.New("image vector table unavailable"))
	tenantMock.ExpectRollback()

	enabled := false
	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)

	_, err = svc.UpdateSourceGovernance(ctx, UpdateSemanticModelSourceGovernanceParams{
		ModelID:  7,
		SourceID: "source-file-1",
		Enabled:  &enabled,
	})
	if !IsServiceError(err, ErrCodeBadRequest) || !i18n.IsKey(err, i18n.KeySessionVectorTableUnavailable) {
		t.Fatalf("UpdateSourceGovernance error = %v, want structured vector-table unavailable error", err)
	}
	if !strings.Contains(err.Error(), "update vector table kb_image_idx disabled") {
		t.Fatalf("UpdateSourceGovernance error = %v, want retained internal cause", err)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceUpdateSourceGovernanceReturnsNotFoundWhenSourceUpdateMisses(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/workspaces/ws-1/semantic-models/7" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":   7,
			"name": "kb",
			"files": map[string]any{
				"file_ids":     []string{"kb-file"},
				"vector_table": "kb_text_idx",
			},
		})
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()

	svc := newSemanticModelTestService(t, server.URL, systemClient)
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	tenantMock.ExpectQuery("SELECT .*source_id.*FROM knowledge_base_sources").
		WithArgs(int64(7), "source-file-1").
		WillReturnRows(sqlmock.NewRows([]string{
			"source_id", "model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "source_type",
			"source_file_id", "source_table_id", "kb_file_id", "kb_table_id", "display_name", "source_path",
			"db_name", "table_name", "status", "error", "enabled", "expires_at", "tags", "force_enabled_after_expiry", "segment_version_id", "index_version",
		}).AddRow("source-file-1", int64(7), int64(3), int64(11), int64(12), int64(13), kbSourceTypeCatalogFile, "source-file", nil, "kb-file", nil, "doc.pdf", nil, nil, nil, kbSourceStatusSucceeded, nil, true, nil, nil, false, nil, int64(2)))
	tenantMock.ExpectBegin()
	tenantMock.ExpectExec("UPDATE knowledge_base_sources").
		WillReturnResult(sqlmock.NewResult(0, 0))
	tenantMock.ExpectRollback()

	enabled := false
	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)

	_, err = svc.UpdateSourceGovernance(ctx, UpdateSemanticModelSourceGovernanceParams{
		ModelID:  7,
		SourceID: "source-file-1",
		Enabled:  &enabled,
	})
	var serviceErr *ServiceError
	if !errors.As(err, &serviceErr) || serviceErr.Code != ErrCodeNotFound {
		t.Fatalf("UpdateSourceGovernance error = %v, want not found", err)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceUpdateSourceGovernanceReturnsNotFoundWhenSourceRowMissing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/workspaces/ws-1/semantic-models/7" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":   7,
			"name": "kb",
			"files": map[string]any{
				"file_ids":     []string{"kb-file"},
				"vector_table": "kb_text_idx",
			},
		})
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()

	svc := newSemanticModelTestService(t, server.URL, systemClient)
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	tenantMock.ExpectQuery("SELECT .*source_id.*FROM knowledge_base_sources").
		WithArgs(int64(7), "source-file-missing").
		WillReturnRows(sqlmock.NewRows([]string{
			"source_id", "model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "source_type",
			"source_file_id", "source_table_id", "kb_file_id", "kb_table_id", "display_name", "source_path",
			"db_name", "table_name", "status", "error", "enabled", "expires_at", "tags", "force_enabled_after_expiry", "segment_version_id", "index_version",
		}))

	enabled := false
	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)

	_, err = svc.UpdateSourceGovernance(ctx, UpdateSemanticModelSourceGovernanceParams{
		ModelID:  7,
		SourceID: "source-file-missing",
		Enabled:  &enabled,
	})
	var serviceErr *ServiceError
	if !errors.As(err, &serviceErr) || serviceErr.Code != ErrCodeNotFound {
		t.Fatalf("UpdateSourceGovernance error = %v, want not found", err)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceUpdateTableSourceGovernanceEnabledAndExpiresAt(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/workspaces/ws-1/semantic-models/7" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":   7,
			"name": "kb",
		})
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()

	svc := newSemanticModelTestService(t, server.URL, systemClient)
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	tenantMock.ExpectQuery("SELECT .*source_id.*FROM knowledge_base_sources").
		WithArgs(int64(7), "source-table-1").
		WillReturnRows(sqlmock.NewRows([]string{
			"source_id", "model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "source_type",
			"source_file_id", "source_table_id", "kb_file_id", "kb_table_id", "display_name", "source_path",
			"db_name", "table_name", "status", "error", "enabled", "expires_at", "tags", "force_enabled_after_expiry", "segment_version_id", "index_version",
		}).AddRow("source-table-1", int64(7), int64(3), int64(11), int64(12), int64(13), kbSourceTypeCatalogTable, nil, int64(1001), nil, int64(2001), "orders", nil, "catalog_db", "orders", kbSourceStatusSucceeded, nil, true, nil, nil, false, nil, nil))
	tenantMock.ExpectExec("UPDATE knowledge_base_sources").
		WillReturnResult(sqlmock.NewResult(0, 1))

	enabled := false
	expiresAt := int64(1782748799)
	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)

	result, err := svc.UpdateSourceGovernance(ctx, UpdateSemanticModelSourceGovernanceParams{
		ModelID:  7,
		SourceID: "source-table-1",
		Enabled:  &enabled,
		ExpiresAt: OptionalInt64{
			Set:   true,
			Value: &expiresAt,
		},
	})
	if err != nil {
		t.Fatalf("UpdateSourceGovernance: %v", err)
	}
	if result.Source.Enabled == nil || *result.Source.Enabled {
		t.Fatalf("updated table source enabled = %+v", result.Source)
	}
	if result.Source.SourceType != SemanticModelSourceTypeTable {
		t.Fatalf("updated table source type = %+v", result.Source)
	}
	if result.Source.ExpiresAt == nil || *result.Source.ExpiresAt != expiresAt {
		t.Fatalf("updated table source expires_at = %+v", result.Source)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceUpdateTableSourceGovernanceRejectsDocumentFields(t *testing.T) {
	forceEnabled := true
	tests := []struct {
		name   string
		params UpdateSemanticModelSourceGovernanceParams
	}{
		{
			name: "tags",
			params: UpdateSemanticModelSourceGovernanceParams{
				Tags: &[]string{"finance"},
			},
		},
		{
			name: "force enabled after expiry",
			params: UpdateSemanticModelSourceGovernanceParams{
				ForceEnabledAfterExpiry: &forceEnabled,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.URL.Path != "/api/v1/workspaces/ws-1/semantic-models/7" {
					t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{
					"id":   7,
					"name": "kb",
				})
			}))
			defer server.Close()

			systemClient, err := moi.New(server.URL, "system-key")
			if err != nil {
				t.Fatalf("moi.New: %v", err)
			}
			defer systemClient.Close()

			svc := newSemanticModelTestService(t, server.URL, systemClient)
			tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
			if err != nil {
				t.Fatalf("tenant sqlmock: %v", err)
			}
			defer tenantSQL.Close()
			tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
			if err != nil {
				t.Fatalf("open tenant gorm: %v", err)
			}
			tenantMock.ExpectQuery("SELECT .*source_id.*FROM knowledge_base_sources").
				WithArgs(int64(7), "source-table-1").
				WillReturnRows(sqlmock.NewRows([]string{
					"source_id", "model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "source_type",
					"source_file_id", "source_table_id", "kb_file_id", "kb_table_id", "display_name", "source_path",
					"db_name", "table_name", "status", "error", "enabled", "expires_at", "tags", "force_enabled_after_expiry", "segment_version_id", "index_version",
				}).AddRow("source-table-1", int64(7), int64(3), int64(11), int64(12), int64(13), kbSourceTypeCatalogTable, nil, int64(1001), nil, int64(2001), "orders", nil, "catalog_db", "orders", kbSourceStatusSucceeded, nil, true, nil, nil, false, nil, nil))

			ctx := ctxutil.WithUID(context.Background(), "user-1")
			ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
			ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
			ctx = ctxutil.WithTenantDB(ctx, tenantDB)
			tt.params.ModelID = 7
			tt.params.SourceID = "source-table-1"

			_, err = svc.UpdateSourceGovernance(ctx, tt.params)
			var serviceErr *ServiceError
			if !errors.As(err, &serviceErr) || serviceErr.Code != ErrCodeBadRequest {
				t.Fatalf("UpdateSourceGovernance error = %v, want bad request", err)
			}
			if err := tenantMock.ExpectationsWereMet(); err != nil {
				t.Fatalf("tenant sql expectations: %v", err)
			}
		})
	}
}

func TestSemanticModelServiceUpdateTableSourceGovernanceReturnsNotFoundWhenSourceUpdateMisses(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/workspaces/ws-1/semantic-models/7" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":   7,
			"name": "kb",
		})
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()

	svc := newSemanticModelTestService(t, server.URL, systemClient)
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	tenantMock.ExpectQuery("SELECT .*source_id.*FROM knowledge_base_sources").
		WithArgs(int64(7), "source-table-1").
		WillReturnRows(sqlmock.NewRows([]string{
			"source_id", "model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "source_type",
			"source_file_id", "source_table_id", "kb_file_id", "kb_table_id", "display_name", "source_path",
			"db_name", "table_name", "status", "error", "enabled", "expires_at", "tags", "force_enabled_after_expiry", "segment_version_id", "index_version",
		}).AddRow("source-table-1", int64(7), int64(3), int64(11), int64(12), int64(13), kbSourceTypeCatalogTable, nil, int64(1001), nil, int64(2001), "orders", nil, "catalog_db", "orders", kbSourceStatusSucceeded, nil, true, nil, nil, false, nil, nil))
	tenantMock.ExpectExec("UPDATE knowledge_base_sources").
		WillReturnResult(sqlmock.NewResult(0, 0))

	enabled := false
	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)

	_, err = svc.UpdateSourceGovernance(ctx, UpdateSemanticModelSourceGovernanceParams{
		ModelID:  7,
		SourceID: "source-table-1",
		Enabled:  &enabled,
	})
	var serviceErr *ServiceError
	if !errors.As(err, &serviceErr) || serviceErr.Code != ErrCodeNotFound {
		t.Fatalf("UpdateSourceGovernance error = %v, want not found", err)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func isKnowledgeBaseEmbeddingModelsRequest(r *http.Request) bool {
	return r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/ws-1/embeddings/models"
}

func writeKnowledgeBaseEmbeddingModelsResponse(t *testing.T, w http.ResponseWriter, imageBackendID int64) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{
		"models": []map[string]any{
			{"model": "bge-m3", "backend_id": int64(41), "backend_name": "self-hosted-text"},
			{"model": "efficientnet-b3", "backend_id": imageBackendID, "backend_name": "self-hosted-image"},
		},
	}); err != nil {
		t.Fatalf("encode embedding models response: %v", err)
	}
}

func configureSemanticModelTestCore(t *testing.T, endpoint string) {
	t.Helper()
	if endpoint == "" || endpoint == "http://unused" {
		return
	}
	if err := coreclient.Configure(coreclient.Config{
		Endpoint:     endpoint,
		SystemAPIKey: "system-key",
		Timeout:      5 * time.Second,
	}); err != nil {
		t.Fatalf("configure moi-core test client: %v", err)
	}
}

func newSemanticModelTestService(t *testing.T, endpoint string, _ *moi.Client) SemanticModelService {
	t.Helper()
	configureSemanticModelTestCore(t, endpoint)
	return withSemanticModelCleanupSagaTestStore(t, NewSemanticModelService())
}

func newSemanticModelTestServiceWithDependencies(t *testing.T, endpoint string, _ *moi.Client, dataDomainService SemanticModelCatalogDataDomainService, fileService SemanticModelCatalogFileService) SemanticModelService {
	t.Helper()
	configureSemanticModelTestCore(t, endpoint)
	return withSemanticModelCleanupSagaTestStore(t, NewSemanticModelServiceWithDependencies(dataDomainService, fileService))
}

func newSemanticModelTestServiceWithKnowledgeBaseDependencies(t *testing.T, endpoint string, _ *moi.Client, dataDomainService SemanticModelCatalogDataDomainService, fileService SemanticModelCatalogFileService, localImportService SemanticModelLocalFileImportService) SemanticModelService {
	t.Helper()
	configureSemanticModelTestCore(t, endpoint)
	return withSemanticModelCleanupSagaTestStore(t, NewSemanticModelServiceWithKnowledgeBaseDependencies(dataDomainService, fileService, localImportService))
}

func newSemanticModelTestServiceWithKnowledgeBaseRuntimeDependencies(t *testing.T, endpoint string, _ *moi.Client, dataDomainService SemanticModelCatalogDataDomainService, fileService SemanticModelCatalogFileService, localImportService SemanticModelLocalFileImportService, workflowTemplateService SemanticModelWorkflowTemplateService, workflowService SemanticModelWorkflowService) SemanticModelService {
	t.Helper()
	configureSemanticModelTestCore(t, endpoint)
	return withSemanticModelCleanupSagaTestStore(t, NewSemanticModelServiceWithKnowledgeBaseRuntimeDependencies(dataDomainService, fileService, localImportService, workflowTemplateService, workflowService))
}

func withSemanticModelCleanupSagaTestStore(t *testing.T, svc SemanticModelService) SemanticModelService {
	t.Helper()
	concrete, ok := svc.(*semanticModelService)
	if !ok {
		t.Fatalf("semantic model service type = %T, want *semanticModelService", svc)
	}
	installSemanticModelCleanupSagaTestStore(t, concrete)
	return concrete
}

func installSemanticModelCleanupSagaTestStore(t *testing.T, svc *semanticModelService) {
	t.Helper()
	store := sagastore.NewMemoryStorage()
	svc.knowledgeBaseCleanupSagaExecutor = coresaga.NewExecutor(coresaga.ExecutorConfig{Storage: store, MaxRetries: -1})
	svc.knowledgeBaseCleanupSagaStore = store
}

func TestQuoteQualifiedSQLIdentifier(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "single table", input: "kb_text_idx", want: "`kb_text_idx`"},
		{name: "qualified table", input: "idx_db.kb_text_idx", want: "`idx_db`.`kb_text_idx`"},
		{name: "empty component", input: "idx_db.", wantErr: true},
		{name: "too many components", input: "a.b.c", wantErr: true},
		{name: "invalid component", input: "idx_db.kb text", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := quoteQualifiedSQLIdentifier(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("quoteQualifiedSQLIdentifier(%q) error is nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("quoteQualifiedSQLIdentifier(%q): %v", tt.input, err)
			}
			if got != tt.want {
				t.Fatalf("quoteQualifiedSQLIdentifier(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestKnowledgeBaseDatabaseNameMatchesKnowledgeBaseName(t *testing.T) {
	tests := []struct {
		name    string
		want    string
		wantErr bool
	}{
		{name: "sales_aqi_kb", want: "sales_aqi_kb"},
		{name: "产品文档", want: "产品文档"},
		{name: "SalesAQI", wantErr: true}, // uppercase rejected by catalog identifier rules
		{name: "Sales AQI KB", wantErr: true},
		{name: "", wantErr: true},
		{name: "-bad", wantErr: true},
	}
	for _, tt := range tests {
		got, err := knowledgeBaseDatabaseName(tt.name)
		if tt.wantErr {
			if err == nil {
				t.Fatalf("knowledgeBaseDatabaseName(%q) error = nil, want error", tt.name)
			}
			continue
		}
		if err != nil {
			t.Fatalf("knowledgeBaseDatabaseName(%q): %v", tt.name, err)
		}
		if got != tt.want {
			t.Fatalf("knowledgeBaseDatabaseName(%q) = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestSemanticModelServiceRunPendingSourceJobsClonesTableAndUpdatesSemanticModelScope(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			requireSemanticModelExecutionHeaders(t, r)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":          77,
				"name":        "kb_docs",
				"description": "docs",
				"tables":      []any{},
				"files":       map[string]any{"file_ids": []string{"kb-local-file"}},
			})
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			requireSemanticModelExecutionHeaders(t, r)
			var req struct {
				Tables []semanticModelTableSource `json:"tables"`
				Files  struct {
					FileIDs []string `json:"file_ids"`
				} `json:"files"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode semantic update: %v", err)
			}
			if len(req.Tables) != 1 || req.Tables[0].DBName != "kb_docs" || len(req.Tables[0].TableNames) != 1 || req.Tables[0].TableNames[0] != "orders" {
				t.Fatalf("semantic tables update = %+v", req.Tables)
			}
			if len(req.Files.FileIDs) != 1 || req.Files.FileIDs[0] != "kb-local-file" {
				t.Fatalf("semantic files update = %+v", req.Files.FileIDs)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"updated": true})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	dataDomainSvc := &fakeSemanticModelDataDomainService{databaseID: 11}
	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()
	svc := newSemanticModelTestServiceWithDependencies(t, server.URL, systemClient, dataDomainSvc, nil)

	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}

	expectPendingSourceJobRunsWithPendingEmpty(tenantMock, 77, kbJobTypeCopy)
	tenantMock.ExpectQuery("SELECT .*job_id.*source_id.*model_id.*job_type.*job_status.*idempotency_key").
		WithArgs(int64(77), kbJobTypeTableClone, kbSourceJobPending, kbSourceJobQueued, kbSourceJobRunning, kbSourceJobReconcileBatchSize).
		WillReturnRows(sqlmock.NewRows([]string{
			"job_id", "source_id", "model_id", "job_type", "job_status", "idempotency_key", "operation_id",
			"workflow_execution_id", "runtime_actor_moi_user_id", "runtime_effective_role_id", "runtime_is_workspace_owner", "source_file_id", "kb_file_id", "source_table_id", "kb_table_id",
			"retry_count", "next_retry_at", "error", "created_at", "updated_at",
		}).AddRow("job-table-1", "source-table-1", int64(77), kbJobTypeTableClone, kbSourceJobQueued, "idem-table-1", nil, nil, nil, nil, false, nil, nil, int64(1001), nil, int64(0), nil, nil, int64(100), int64(100)))
	tenantMock.ExpectQuery("SELECT .*job_id.*source_id.*model_id.*job_type.*job_status.*idempotency_key").
		WithArgs(int64(77), kbJobTypeLoad, kbSourceJobQueued, kbSourceJobRunning, kbSourceJobSucceeded, kbSourceStatusRemoved, kbSourceTypeCatalogTable, kbSourceTypeLocalFile, kbRawKindStructured, kbSourceJobReconcileBatchSize).
		WillReturnRows(sqlmock.NewRows([]string{
			"job_id", "source_id", "model_id", "job_type", "job_status", "idempotency_key", "operation_id",
			"workflow_execution_id", "runtime_actor_moi_user_id", "runtime_effective_role_id", "runtime_is_workspace_owner", "source_file_id", "kb_file_id", "source_table_id", "kb_table_id",
			"retry_count", "next_retry_at", "error", "created_at", "updated_at",
		}))
	tenantMock.ExpectQuery("SELECT .*source_id.*FROM knowledge_base_sources").
		WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows([]string{
			"source_id", "model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "source_type",
			"source_file_id", "source_table_id", "kb_file_id", "kb_table_id", "display_name", "source_path",
			"db_name", "table_name", "status", "error", "enabled", "expires_at", "tags", "force_enabled_after_expiry", "segment_version_id", "index_version",
		}).AddRow("source-table-1", int64(77), int64(3), int64(11), int64(12), int64(13), kbSourceTypeCatalogTable, nil, int64(1001), nil, nil, "orders", `["catalog_db","orders"]`, nil, nil, kbSourceStatusPending, nil, nil, nil, nil, false, nil, nil))
	tenantMock.ExpectExec("UPDATE knowledge_base_source_job_runs").
		WithArgs(kbSourceJobRunning, "user-1", "job-table-1", kbSourceJobPending, kbSourceJobQueued, kbSourceJobRunning, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	tenantMock.ExpectBegin()
	tenantMock.ExpectQuery("SELECT model_id.*FROM knowledge_base_data_domains.*FOR UPDATE").
		WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows([]string{"model_id"}).AddRow(int64(77)))
	tenantMock.ExpectQuery("SELECT COUNT\\(\\*\\).*FROM knowledge_base_sources kbs.*JOIN knowledge_base_source_job_runs jr").
		WithArgs(int64(77), "source-table-1", kbSourceStatusRemoved, "job-table-1", kbJobTypeTableClone).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(1)))
	tenantMock.ExpectExec("UPDATE knowledge_base_sources").
		WillReturnResult(sqlmock.NewResult(1, 1))
	tenantMock.ExpectExec("UPDATE knowledge_base_source_job_runs").
		WillReturnResult(sqlmock.NewResult(1, 1))
	tenantMock.ExpectCommit()

	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithUserID(ctx, "user-1")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)

	if err := svc.RunPendingKnowledgeBaseSourceJobs(ctx, RunPendingKnowledgeBaseSourceJobsParams{ModelID: 77}); err != nil {
		t.Fatalf("RunPendingKnowledgeBaseSourceJobs: %v", err)
	}
	if len(dataDomainSvc.calls) != 1 || !strings.HasPrefix(dataDomainSvc.calls[0], "table:") {
		t.Fatalf("data domain calls = %+v", dataDomainSvc.calls)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceReconcileSourceJobsReconcilesStructuredLocalLoadTable(t *testing.T) {
	var semanticGetCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			semanticGetCount++
			if semanticGetCount == 1 {
				_ = json.NewEncoder(w).Encode(map[string]any{
					"id":          77,
					"name":        "kb_docs",
					"description": "docs",
					"tables":      []semanticModelTableSource{},
					"files":       map[string]any{"file_ids": []string{}},
				})
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":          77,
				"name":        "kb_docs",
				"description": "docs",
				"tables":      []semanticModelTableSource{{DBName: "kb_docs", TableNames: []string{"existing_orders"}}},
				"files":       map[string]any{"file_ids": []string{"kb-structured-file"}},
			})
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			var req struct {
				Tables []semanticModelTableSource `json:"tables"`
				Files  struct {
					FileIDs []string `json:"file_ids"`
				} `json:"files"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode semantic update: %v", err)
			}
			if len(req.Tables) != 1 || req.Tables[0].DBName != "kb_docs" || !sameStringSet(req.Tables[0].TableNames, []string{"existing_orders", "structured_orders"}) {
				t.Fatalf("semantic tables update = %+v", req.Tables)
			}
			if len(req.Files.FileIDs) != 0 {
				t.Fatalf("semantic files update = %+v, want no structured file relation", req.Files.FileIDs)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"updated": true})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()
	svc := newSemanticModelTestServiceWithDependencies(t, server.URL, systemClient, &fakeSemanticModelDataDomainService{databaseID: 11}, nil)

	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}

	expectLegacyBackfillNoop(tenantMock, 77)
	tenantMock.ExpectQuery("SELECT jr.job_id.*COALESCE\\(krv.raw_kind").
		WillReturnRows(sqlmock.NewRows([]string{
			"job_id", "source_id", "model_id", "job_type", "job_status", "idempotency_key", "operation_id",
			"workflow_execution_id", "runtime_actor_moi_user_id", "runtime_effective_role_id", "runtime_is_workspace_owner", "source_file_id", "kb_file_id", "source_table_id", "kb_table_id",
			"retry_count", "next_retry_at", "error", "created_at", "updated_at",
		}))
	tenantMock.ExpectQuery("SELECT .*job_id.*source_id.*model_id.*job_type.*job_status.*idempotency_key").
		WithArgs(int64(77), kbJobTypeTableClone, kbSourceJobPending, kbSourceJobQueued, kbSourceJobRunning, kbSourceStatusFailed, kbSourceStatusRemoved, kbSourceJobReconcileBatchSize).
		WillReturnRows(sqlmock.NewRows([]string{
			"job_id", "source_id", "model_id", "job_type", "job_status", "idempotency_key", "operation_id",
			"workflow_execution_id", "runtime_actor_moi_user_id", "runtime_effective_role_id", "runtime_is_workspace_owner", "source_file_id", "kb_file_id", "source_table_id", "kb_table_id",
			"retry_count", "next_retry_at", "error", "created_at", "updated_at",
		}))
	tenantMock.ExpectQuery("SELECT .*job_id.*source_id.*model_id.*job_type.*job_status.*idempotency_key").
		WithArgs(int64(77), kbJobTypeLoad, kbSourceJobSucceeded, kbSourceStatusRemoved, kbSourceTypeCatalogTable, kbSourceTypeLocalFile, kbRawKindStructured, kbSourceJobSucceeded, kbSourceStatusSucceeded, kbSourceJobReconcileBatchSize).
		WillReturnRows(sqlmock.NewRows([]string{
			"job_id", "source_id", "model_id", "job_type", "job_status", "idempotency_key", "operation_id",
			"workflow_execution_id", "runtime_actor_moi_user_id", "runtime_effective_role_id", "runtime_is_workspace_owner", "source_file_id", "kb_file_id", "source_table_id", "kb_table_id",
			"retry_count", "next_retry_at", "error", "created_at", "updated_at",
		}))
	tenantMock.ExpectQuery("SELECT .*job_id.*source_id.*model_id.*job_type.*job_status.*idempotency_key").
		WithArgs(int64(77), kbJobTypeLoad, kbSourceJobQueued, kbSourceJobRunning, kbSourceStatusRemoved, kbSourceTypeCatalogTable, kbSourceTypeLocalFile, kbRawKindStructured, kbSourceJobSucceeded, kbSourceStatusSucceeded, kbSourceJobReconcileBatchSize).
		WillReturnRows(sqlmock.NewRows([]string{
			"job_id", "source_id", "model_id", "job_type", "job_status", "idempotency_key", "operation_id",
			"workflow_execution_id", "runtime_actor_moi_user_id", "runtime_effective_role_id", "runtime_is_workspace_owner", "source_file_id", "kb_file_id", "source_table_id", "kb_table_id",
			"retry_count", "next_retry_at", "error", "created_at", "updated_at",
		}).AddRow("job-load-1", "source-local-1", int64(77), kbJobTypeLoad, kbSourceJobRunning, "idem-load-1", "import_task:task-structured-1", nil, nil, nil, false, "kb-structured-file", "kb-structured-file", nil, nil, int64(0), nil, nil, int64(100), int64(100)))
	tenantMock.ExpectQuery("SELECT .*source_id.*FROM knowledge_base_sources").
		WithArgs(int64(77), "source-local-1", kbSourceStatusRemoved).
		WillReturnRows(sqlmock.NewRows([]string{
			"source_id", "model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "source_type",
			"source_file_id", "source_table_id", "kb_file_id", "kb_table_id", "display_name", "source_path",
			"db_name", "table_name", "status", "error", "enabled", "expires_at", "tags", "force_enabled_after_expiry", "segment_version_id", "index_version",
		}).
			AddRow("source-local-1", int64(77), int64(3), int64(11), int64(12), int64(13), kbSourceTypeLocalFile, "kb-structured-file", nil, "kb-structured-file", nil, "structured.csv", nil, nil, nil, kbSourceStatusPending, nil, nil, nil, nil, false, nil, nil))
	tenantMock.ExpectQuery("SELECT .* FROM `import_task` WHERE id = \\? ORDER BY `import_task`.`id` LIMIT \\?").
		WithArgs("task-structured-1", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "status", "task_meta"}).
			AddRow("task-structured-1", model.ImportTaskStatusFinished, `{"structured_table_results":[{"database_id":11,"table_id":2001,"db_name":"kb_docs","table_name":"structured_orders","lines":12}]}`))
	tenantMock.ExpectBegin()
	tenantMock.ExpectQuery("SELECT model_id.*FROM knowledge_base_data_domains.*FOR UPDATE").
		WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows([]string{"model_id"}).AddRow(int64(77)))
	tenantMock.ExpectQuery("SELECT COUNT\\(\\*\\).*FROM knowledge_base_sources kbs.*JOIN knowledge_base_source_job_runs jr").
		WithArgs(int64(77), "source-local-1", kbSourceStatusRemoved, "job-load-1", kbJobTypeLoad).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(1)))
	tenantMock.ExpectExec("UPDATE knowledge_base_sources").
		WillReturnResult(sqlmock.NewResult(1, 1))
	tenantMock.ExpectExec("UPDATE knowledge_base_source_job_runs").
		WillReturnResult(sqlmock.NewResult(1, 1))
	tenantMock.ExpectCommit()

	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithUserID(ctx, "user-1")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)

	if err := svc.ReconcileKnowledgeBaseSourceJobs(ctx, ReconcileKnowledgeBaseSourceJobsParams{ModelID: 77}); err != nil {
		t.Fatalf("ReconcileKnowledgeBaseSourceJobs: %v", err)
	}
	if semanticGetCount != 2 {
		t.Fatalf("semantic get count = %d, want 2", semanticGetCount)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceRunPendingSourceJobsReconcilesStructuredLoadMultipleTables(t *testing.T) {
	modelID := int64(77)
	secondSourceID := stableID("kb-source", modelID, kbSourceTypeCatalogTable, int64(2002))
	secondJobID := stableID("kb-job", secondSourceID, kbJobTypeLoad)
	secondJobKey := stableID("kb-job-key", secondSourceID, kbJobTypeLoad)
	operationID := "import_task:task-structured-1"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":          modelID,
				"name":        "kb_docs",
				"description": "docs",
				"tables":      []semanticModelTableSource{},
				"files":       map[string]any{"file_ids": []string{"kb-structured-file"}},
			})
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			var req struct {
				Tables []semanticModelTableSource `json:"tables"`
				Files  struct {
					FileIDs []string `json:"file_ids"`
				} `json:"files"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode semantic update: %v", err)
			}
			if len(req.Tables) != 1 || req.Tables[0].DBName != "kb_docs" || !sameStringSet(req.Tables[0].TableNames, []string{"sheet_orders", "sheet_customers"}) {
				t.Fatalf("semantic tables update = %+v", req.Tables)
			}
			if len(req.Files.FileIDs) != 0 {
				t.Fatalf("semantic files update = %+v, want no structured file relation", req.Files.FileIDs)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"updated": true})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()
	svc := newSemanticModelTestServiceWithDependencies(t, server.URL, systemClient, &fakeSemanticModelDataDomainService{databaseID: 11}, nil)

	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}

	expectPendingSourceJobRunsWithPendingEmpty(tenantMock, modelID, kbJobTypeCopy)
	tenantMock.ExpectQuery("SELECT .*job_id.*source_id.*model_id.*job_type.*job_status.*idempotency_key").
		WithArgs(modelID, kbJobTypeTableClone, kbSourceJobPending, kbSourceJobQueued, kbSourceJobRunning, kbSourceJobReconcileBatchSize).
		WillReturnRows(knowledgeBaseSourceJobRunRows())
	tenantMock.ExpectQuery("SELECT .*job_id.*source_id.*model_id.*job_type.*job_status.*idempotency_key").
		WithArgs(modelID, kbJobTypeLoad, kbSourceJobQueued, kbSourceJobRunning, kbSourceJobSucceeded, kbSourceStatusRemoved, kbSourceTypeCatalogTable, kbSourceTypeLocalFile, kbRawKindStructured, kbSourceJobReconcileBatchSize).
		WillReturnRows(knowledgeBaseSourceJobRunRows().
			AddRow("job-load-1", "source-local-1", modelID, kbJobTypeLoad, kbSourceJobRunning, "idem-load-1", operationID, nil, nil, nil, false, "kb-structured-file", "kb-structured-file", nil, nil, int64(0), nil, nil, int64(100), int64(100)))
	tenantMock.ExpectQuery("SELECT .*source_id.*FROM knowledge_base_sources").
		WithArgs(modelID).
		WillReturnRows(sqlmock.NewRows([]string{
			"source_id", "model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "source_type",
			"source_file_id", "source_table_id", "kb_file_id", "kb_table_id", "display_name", "source_path",
			"db_name", "table_name", "status", "error", "enabled", "expires_at", "tags", "force_enabled_after_expiry", "segment_version_id", "index_version",
		}).
			AddRow("source-local-1", modelID, int64(3), int64(11), int64(14), int64(13), kbSourceTypeLocalFile, "kb-structured-file", nil, "kb-structured-file", nil, "structured.xlsx", nil, nil, nil, kbSourceStatusPending, nil, nil, nil, nil, false, nil, nil))
	tenantMock.ExpectQuery("SELECT .* FROM `import_task` WHERE id = \\? ORDER BY `import_task`.`id` LIMIT \\?").
		WithArgs("task-structured-1", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "status", "task_meta"}).
			AddRow("task-structured-1", model.ImportTaskStatusFinished, `{"structured_table_results":[{"database_id":11,"table_id":2001,"db_name":"kb_docs","table_name":"sheet_orders","lines":12},{"database_id":11,"table_id":2002,"db_name":"kb_docs","table_name":"sheet_customers","lines":8}]}`))
	tenantMock.ExpectBegin()
	tenantMock.ExpectQuery("SELECT model_id.*FROM knowledge_base_data_domains.*FOR UPDATE").
		WithArgs(modelID).
		WillReturnRows(sqlmock.NewRows([]string{"model_id"}).AddRow(modelID))
	tenantMock.ExpectQuery("SELECT COUNT\\(\\*\\).*FROM knowledge_base_sources kbs.*JOIN knowledge_base_source_job_runs jr").
		WithArgs(modelID, "source-local-1", kbSourceStatusRemoved, "job-load-1", kbJobTypeLoad).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(1)))
	tenantMock.ExpectExec("UPDATE knowledge_base_sources").
		WillReturnResult(sqlmock.NewResult(1, 1))
	tenantMock.ExpectExec("UPDATE knowledge_base_source_job_runs").
		WillReturnResult(sqlmock.NewResult(1, 1))
	tenantMock.ExpectExec("INSERT INTO knowledge_base_sources").
		WithArgs(secondSourceID, modelID, int64(3), int64(11), int64(14), int64(13), kbSourceTypeCatalogTable, nil, nil, nil, int64(2002), "sheet_customers", nil, "kb_docs", "sheet_customers", kbSourceStatusSucceeded, nil, nil, nil, nil, false, nil, nil, "user-1", "user-1").
		WillReturnResult(sqlmock.NewResult(1, 1))
	tenantMock.ExpectExec("UPDATE knowledge_base_source_job_runs").
		WithArgs(nil, nil, false, kbSourceJobSucceeded, operationID, nil, nil, nil, nil, int64(2002), int64(0), nil, nil, "user-1", secondJobID).
		WillReturnResult(sqlmock.NewResult(0, 0))
	tenantMock.ExpectExec("INSERT INTO knowledge_base_source_job_runs").
		WithArgs(secondJobID, secondSourceID, modelID, kbJobTypeLoad, kbSourceJobSucceeded, secondJobKey, operationID, nil, nil, nil, false, nil, nil, nil, int64(2002), int64(0), nil, nil, "user-1", "user-1").
		WillReturnResult(sqlmock.NewResult(1, 1))
	tenantMock.ExpectCommit()

	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithUserID(ctx, "user-1")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)

	if err := svc.RunPendingKnowledgeBaseSourceJobs(ctx, RunPendingKnowledgeBaseSourceJobsParams{ModelID: modelID}); err != nil {
		t.Fatalf("RunPendingKnowledgeBaseSourceJobs: %v", err)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceRunPendingSourceJobsReconcilesSucceededStructuredTableLoad(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":          77,
				"name":        "kb_docs",
				"description": "docs",
				"tables":      []semanticModelTableSource{},
				"files":       map[string]any{"file_ids": []string{}},
			})
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			var req struct {
				Tables []semanticModelTableSource `json:"tables"`
				Files  struct {
					FileIDs []string `json:"file_ids"`
				} `json:"files"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode semantic update: %v", err)
			}
			if len(req.Tables) != 1 || req.Tables[0].DBName != "kb_docs" || !sameStringSet(req.Tables[0].TableNames, []string{"structured_orders"}) {
				t.Fatalf("semantic tables update = %+v", req.Tables)
			}
			if len(req.Files.FileIDs) != 0 {
				t.Fatalf("semantic files update = %+v, want no structured file relation", req.Files.FileIDs)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"updated": true})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()
	svc := newSemanticModelTestServiceWithDependencies(t, server.URL, systemClient, &fakeSemanticModelDataDomainService{databaseID: 11}, nil)

	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}

	expectPendingSourceJobRunsWithPendingEmpty(tenantMock, 77, kbJobTypeCopy)
	tenantMock.ExpectQuery("SELECT .*job_id.*source_id.*model_id.*job_type.*job_status.*idempotency_key").
		WithArgs(int64(77), kbJobTypeTableClone, kbSourceJobPending, kbSourceJobQueued, kbSourceJobRunning, kbSourceJobReconcileBatchSize).
		WillReturnRows(knowledgeBaseSourceJobRunRows())
	tenantMock.ExpectQuery("SELECT .*job_id.*source_id.*model_id.*job_type.*job_status.*idempotency_key").
		WithArgs(int64(77), kbJobTypeLoad, kbSourceJobQueued, kbSourceJobRunning, kbSourceJobSucceeded, kbSourceStatusRemoved, kbSourceTypeCatalogTable, kbSourceTypeLocalFile, kbRawKindStructured, kbSourceJobReconcileBatchSize).
		WillReturnRows(knowledgeBaseSourceJobRunRows().
			AddRow("job-load-1", "source-structured-1", int64(77), kbJobTypeLoad, kbSourceJobSucceeded, "idem-load-1", "import_task:task-structured-1", nil, nil, nil, false, nil, nil, nil, nil, int64(0), nil, nil, int64(100), int64(100)))
	tenantMock.ExpectQuery("SELECT .*source_id.*FROM knowledge_base_sources").
		WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows([]string{
			"source_id", "model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "source_type",
			"source_file_id", "source_table_id", "kb_file_id", "kb_table_id", "display_name", "source_path",
			"db_name", "table_name", "status", "error", "enabled", "expires_at", "tags", "force_enabled_after_expiry", "segment_version_id", "index_version",
		}).
			AddRow("source-structured-1", int64(77), int64(3), int64(11), int64(14), int64(13), kbSourceTypeCatalogTable, nil, nil, nil, nil, "structured_orders", nil, nil, nil, kbSourceStatusSucceeded, nil, nil, nil, nil, false, nil, nil))
	tenantMock.ExpectQuery("SELECT .* FROM `import_task` WHERE id = \\? ORDER BY `import_task`.`id` LIMIT \\?").
		WithArgs("task-structured-1", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "status", "task_meta"}).
			AddRow("task-structured-1", model.ImportTaskStatusUploading, `{"structured_table_results":[{"database_id":11,"table_id":2001,"db_name":"kb_docs","table_name":"structured_orders","lines":12}]}`))
	tenantMock.ExpectQuery("SELECT .* FROM `import_task_run` WHERE import_task_id = \\? ORDER BY created_at DESC, id DESC,`import_task_run`.`id` LIMIT \\?").
		WithArgs("task-structured-1", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "status"}).
			AddRow("run-structured-1", model.ImportTaskRunStatusCompleted))
	tenantMock.ExpectBegin()
	tenantMock.ExpectQuery("SELECT model_id.*FROM knowledge_base_data_domains.*FOR UPDATE").
		WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows([]string{"model_id"}).AddRow(int64(77)))
	tenantMock.ExpectQuery("SELECT COUNT\\(\\*\\).*FROM knowledge_base_sources kbs.*JOIN knowledge_base_source_job_runs jr").
		WithArgs(int64(77), "source-structured-1", kbSourceStatusRemoved, "job-load-1", kbJobTypeLoad).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(1)))
	tenantMock.ExpectExec("UPDATE knowledge_base_sources").
		WillReturnResult(sqlmock.NewResult(1, 1))
	tenantMock.ExpectExec("UPDATE knowledge_base_source_job_runs").
		WillReturnResult(sqlmock.NewResult(1, 1))
	tenantMock.ExpectCommit()

	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithUserID(ctx, "user-1")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)

	if err := svc.RunPendingKnowledgeBaseSourceJobs(ctx, RunPendingKnowledgeBaseSourceJobsParams{ModelID: 77}); err != nil {
		t.Fatalf("RunPendingKnowledgeBaseSourceJobs: %v", err)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceRunPendingSourceJobsSkipsFailedStructuredLoad(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()
	svc := newSemanticModelTestServiceWithDependencies(t, server.URL, systemClient, &fakeSemanticModelDataDomainService{databaseID: 11}, nil)

	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}

	expectPendingSourceJobRunsWithPendingEmpty(tenantMock, 77, kbJobTypeCopy)
	tenantMock.ExpectQuery("SELECT .*job_id.*source_id.*model_id.*job_type.*job_status.*idempotency_key").
		WithArgs(int64(77), kbJobTypeTableClone, kbSourceJobPending, kbSourceJobQueued, kbSourceJobRunning, kbSourceJobReconcileBatchSize).
		WillReturnRows(knowledgeBaseSourceJobRunRows())
	tenantMock.ExpectQuery("SELECT .*job_id.*source_id.*model_id.*job_type.*job_status.*idempotency_key").
		WithArgs(int64(77), kbJobTypeLoad, kbSourceJobQueued, kbSourceJobRunning, kbSourceJobSucceeded, kbSourceStatusRemoved, kbSourceTypeCatalogTable, kbSourceTypeLocalFile, kbRawKindStructured, kbSourceJobReconcileBatchSize).
		WillReturnRows(knowledgeBaseSourceJobRunRows())

	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithUserID(ctx, "user-1")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)

	if err := svc.RunPendingKnowledgeBaseSourceJobs(ctx, RunPendingKnowledgeBaseSourceJobsParams{ModelID: 77}); err != nil {
		t.Fatalf("RunPendingKnowledgeBaseSourceJobs: %v", err)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceRunPendingSourceJobsMarksCompletedStructuredLoadWithoutResultFailed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()
	svc := newSemanticModelTestServiceWithDependencies(t, server.URL, systemClient, &fakeSemanticModelDataDomainService{databaseID: 11}, nil)

	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}

	expectPendingSourceJobRunsWithPendingEmpty(tenantMock, 77, kbJobTypeCopy)
	tenantMock.ExpectQuery("SELECT .*job_id.*source_id.*model_id.*job_type.*job_status.*idempotency_key").
		WithArgs(int64(77), kbJobTypeTableClone, kbSourceJobPending, kbSourceJobQueued, kbSourceJobRunning, kbSourceJobReconcileBatchSize).
		WillReturnRows(knowledgeBaseSourceJobRunRows())
	tenantMock.ExpectQuery("SELECT .*job_id.*source_id.*model_id.*job_type.*job_status.*idempotency_key").
		WithArgs(int64(77), kbJobTypeLoad, kbSourceJobQueued, kbSourceJobRunning, kbSourceJobSucceeded, kbSourceStatusRemoved, kbSourceTypeCatalogTable, kbSourceTypeLocalFile, kbRawKindStructured, kbSourceJobReconcileBatchSize).
		WillReturnRows(knowledgeBaseSourceJobRunRows().
			AddRow("job-load-1", "source-structured-1", int64(77), kbJobTypeLoad, kbSourceJobSucceeded, "idem-load-1", "import_task:task-structured-1", nil, nil, nil, false, nil, nil, nil, nil, int64(0), nil, nil, int64(100), int64(100)))
	tenantMock.ExpectQuery("SELECT .*source_id.*FROM knowledge_base_sources").
		WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows([]string{
			"source_id", "model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "source_type",
			"source_file_id", "source_table_id", "kb_file_id", "kb_table_id", "display_name", "source_path",
			"db_name", "table_name", "status", "error", "enabled", "expires_at", "tags", "force_enabled_after_expiry", "segment_version_id", "index_version",
		}).
			AddRow("source-structured-1", int64(77), int64(3), int64(11), int64(14), int64(13), kbSourceTypeCatalogTable, nil, nil, nil, nil, "structured_orders", nil, nil, nil, kbSourceStatusPending, nil, nil, nil, nil, false, nil, nil))
	tenantMock.ExpectQuery("SELECT .* FROM `import_task` WHERE id = \\? ORDER BY `import_task`.`id` LIMIT \\?").
		WithArgs("task-structured-1", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "status", "task_meta"}).
			AddRow("task-structured-1", model.ImportTaskStatusFinished, `{}`))
	tenantMock.ExpectExec("UPDATE knowledge_base_sources").
		WillReturnResult(sqlmock.NewResult(1, 1))
	tenantMock.ExpectExec("UPDATE knowledge_base_source_job_runs").
		WillReturnResult(sqlmock.NewResult(1, 1))

	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithUserID(ctx, "user-1")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)

	err = svc.RunPendingKnowledgeBaseSourceJobs(ctx, RunPendingKnowledgeBaseSourceJobsParams{ModelID: 77})
	if err != nil {
		t.Fatalf("RunPendingKnowledgeBaseSourceJobs: %v", err)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceRunPendingSourceJobsMarksFailedImportTaskStructuredLoadFailed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()
	svc := newSemanticModelTestServiceWithDependencies(t, server.URL, systemClient, &fakeSemanticModelDataDomainService{databaseID: 11}, nil)

	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}

	expectPendingSourceJobRunsWithPendingEmpty(tenantMock, 77, kbJobTypeCopy)
	tenantMock.ExpectQuery("SELECT .*job_id.*source_id.*model_id.*job_type.*job_status.*idempotency_key").
		WithArgs(int64(77), kbJobTypeTableClone, kbSourceJobPending, kbSourceJobQueued, kbSourceJobRunning, kbSourceJobReconcileBatchSize).
		WillReturnRows(knowledgeBaseSourceJobRunRows())
	tenantMock.ExpectQuery("SELECT .*job_id.*source_id.*model_id.*job_type.*job_status.*idempotency_key").
		WithArgs(int64(77), kbJobTypeLoad, kbSourceJobQueued, kbSourceJobRunning, kbSourceJobSucceeded, kbSourceStatusRemoved, kbSourceTypeCatalogTable, kbSourceTypeLocalFile, kbRawKindStructured, kbSourceJobReconcileBatchSize).
		WillReturnRows(knowledgeBaseSourceJobRunRows().
			AddRow("job-load-1", "source-structured-1", int64(77), kbJobTypeLoad, kbSourceJobRunning, "idem-load-1", "import_task:task-structured-1", nil, nil, nil, false, nil, nil, nil, nil, int64(0), nil, nil, int64(100), int64(100)))
	tenantMock.ExpectQuery("SELECT .*source_id.*FROM knowledge_base_sources").
		WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows([]string{
			"source_id", "model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "source_type",
			"source_file_id", "source_table_id", "kb_file_id", "kb_table_id", "display_name", "source_path",
			"db_name", "table_name", "status", "error", "enabled", "expires_at", "tags", "force_enabled_after_expiry", "segment_version_id", "index_version",
		}).
			AddRow("source-structured-1", int64(77), int64(3), int64(11), int64(14), int64(13), kbSourceTypeCatalogTable, nil, nil, nil, nil, "structured_orders", nil, nil, nil, kbSourceStatusPending, nil, nil, nil, nil, false, nil, nil))
	tenantMock.ExpectQuery("SELECT .* FROM `import_task` WHERE id = \\? ORDER BY `import_task`.`id` LIMIT \\?").
		WithArgs("task-structured-1", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "status", "task_meta"}).
			AddRow("task-structured-1", model.ImportTaskStatusFailed, `{"error_message":"load failed"}`))
	tenantMock.ExpectQuery("SELECT .* FROM `import_task_run` WHERE import_task_id = \\?").
		WithArgs("task-structured-1", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "status", "error_message"}))
	tenantMock.ExpectExec("UPDATE knowledge_base_sources").
		WithArgs(kbSourceStatusFailed, stringContainsArg{parts: []string{"import task task-structured-1 failed", "load failed"}}, "user-1", "source-structured-1").
		WillReturnResult(sqlmock.NewResult(1, 1))
	tenantMock.ExpectExec("UPDATE knowledge_base_source_job_runs").
		WithArgs(kbSourceJobFailed, stringContainsArg{parts: []string{"import task task-structured-1 failed", "load failed"}}, "user-1", "job-load-1").
		WillReturnResult(sqlmock.NewResult(1, 1))

	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithUserID(ctx, "user-1")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)

	if err := svc.RunPendingKnowledgeBaseSourceJobs(ctx, RunPendingKnowledgeBaseSourceJobsParams{ModelID: 77}); err != nil {
		t.Fatalf("RunPendingKnowledgeBaseSourceJobs: %v", err)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestKnowledgeBaseModelIDForImportTask(t *testing.T) {
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}

	tenantMock.ExpectQuery("SELECT model_id").
		WithArgs(kbJobTypeLoad, "import_task:task-structured-1").
		WillReturnRows(sqlmock.NewRows([]string{"model_id"}).AddRow(int64(77)))
	modelID, ok, err := KnowledgeBaseModelIDForImportTask(context.Background(), tenantDB, "task-structured-1")
	if err != nil {
		t.Fatalf("KnowledgeBaseModelIDForImportTask: %v", err)
	}
	if !ok || modelID != 77 {
		t.Fatalf("modelID = %d, ok = %v, want 77 true", modelID, ok)
	}

	tenantMock.ExpectQuery("SELECT model_id").
		WithArgs(kbJobTypeLoad, "import_task:task-unrelated").
		WillReturnRows(sqlmock.NewRows([]string{"model_id"}))
	modelID, ok, err = KnowledgeBaseModelIDForImportTask(context.Background(), tenantDB, "task-unrelated")
	if err != nil {
		t.Fatalf("KnowledgeBaseModelIDForImportTask missing: %v", err)
	}
	if ok || modelID != 0 {
		t.Fatalf("missing modelID = %d, ok = %v, want 0 false", modelID, ok)
	}

	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceListSourceJobsRequiresTenantDB(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/workspaces/ws-1/semantic-models/77" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 77, "name": "kb_docs", "tables": []any{}, "files": map[string]any{}})
	}))
	defer server.Close()
	configureSemanticModelTestCore(t, server.URL)
	svc := &semanticModelService{}
	_, err := svc.ListSourceJobs(semanticModelServiceTestContext("en-US"), ListSemanticModelSourceJobsParams{ModelID: 77})
	if err == nil || !strings.Contains(err.Error(), "tenant db is required") {
		t.Fatalf("ListSourceJobs error = %v, want tenant db required", err)
	}
}

func TestSemanticModelServiceListSourceJobsReturnsPersistedCloneFailure(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			callCount++
			requireSemanticModelExecutionHeaders(t, r)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":          77,
				"name":        "kb_docs",
				"description": "docs",
				"tables":      []any{},
				"files": map[string]any{
					"file_ids":                   []string{},
					"vector_table":               "client_text_idx",
					"embedding_model":            "BAAI/bge-large-zh-v1.5",
					"image_vector_table":         "client_image_idx",
					"image_embedding_model":      "efficientnet-b3",
					"image_embedding_backend_id": "-30010",
					"image_embedding_dimension":  1536,
					"image_preprocess_version":   "efficientnet-b3-v1-rgb-300-letterbox-imagenet",
					"image_distance_metric":      "cosine",
				},
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()
	dataDomainSvc := &fakeSemanticModelDataDomainService{databaseID: 11, cloneErr: errors.New("copy denied")}
	svc := newSemanticModelTestServiceWithDependencies(t, server.URL, systemClient, dataDomainSvc, nil)

	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}

	expectSourceJobCandidates(tenantMock, 77, []string{"source-table-1"}, knowledgeBaseSourceJobRunRows().
		AddRow("job-table-1", "source-table-1", int64(77), kbJobTypeTableClone, kbSourceJobFailed, "idem-table-1", nil, nil, nil, nil, false, nil, nil, int64(1001), nil, int64(0), nil, "copy denied", int64(100), int64(101)),
		knowledgeBaseSourceRecordRows(KnowledgeBaseSourceRecord{SourceID: "source-table-1", ModelID: 77, SourceType: kbSourceTypeCatalogTable, SourceTableID: int64Ptr(1001), Status: kbSourceStatusPending, Enabled: boolPtr(true)}))

	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithUserID(ctx, "user-1")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)

	resp, err := svc.ListSourceJobs(ctx, ListSemanticModelSourceJobsParams{ModelID: 77})
	if err != nil {
		t.Fatalf("ListSourceJobs: %v", err)
	}
	if resp.Total != 1 || len(resp.Items) != 1 {
		t.Fatalf("source jobs response = %+v", resp)
	}
	if resp.Items[0].JobStatus != kbSourceJobFailed || resp.Items[0].Error == nil || *resp.Items[0].Error != "copy denied" {
		t.Fatalf("source job failure = %+v", resp.Items[0])
	}
	if len(dataDomainSvc.calls) != 0 {
		t.Fatalf("ListSourceJobs should be read-only, data domain calls = %+v", dataDomainSvc.calls)
	}
	if callCount != 1 {
		t.Fatalf("semantic model get count = %d, want 1", callCount)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceListSourceJobsDoesNotMaterializeLegacyJobsWhenPersistedWorkExists(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/workspaces/ws-1/semantic-models/77" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":     77,
			"name":   "kb_docs",
			"tables": []any{},
			"files":  map[string]any{},
		})
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()
	svc := newSemanticModelTestService(t, server.URL, systemClient)
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}

	expectSourceJobCandidates(tenantMock, 77, []string{"source-new-table"}, knowledgeBaseSourceJobRunRows().
		AddRow("job-new-table", "source-new-table", int64(77), kbJobTypeTableClone, kbSourceJobQueued, "idem-new-table", nil, nil, nil, nil, false, nil, nil, int64(1001), nil, int64(0), nil, nil, int64(100), int64(100)),
		knowledgeBaseSourceRecordRows(KnowledgeBaseSourceRecord{SourceID: "source-new-table", ModelID: 77, SourceType: kbSourceTypeCatalogTable, SourceTableID: int64Ptr(1001), Status: kbSourceStatusPending, Enabled: boolPtr(true)}))

	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)

	resp, err := svc.ListSourceJobs(ctx, ListSemanticModelSourceJobsParams{ModelID: 77})
	if err != nil {
		t.Fatalf("ListSourceJobs: %v", err)
	}
	if resp.Total != 1 || len(resp.Items) != 1 || !resp.ReconcileRequired {
		t.Fatalf("source jobs response = %+v, want one persisted candidate", resp)
	}
	if resp.Items[0].SourceID != "source-new-table" {
		t.Fatalf("persisted source jobs = %+v", resp.Items)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceListSourceJobsReturnsLineageOnlyReconcileSignal(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/workspaces/ws-1/semantic-models/77" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":     77,
			"name":   "kb_docs",
			"tables": []any{},
			"files": map[string]any{
				"vector_table": "kb_77_text_index",
			},
		})
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()
	workflowSvc := &fakeSemanticModelWorkflowService{}
	svc := newSemanticModelTestServiceWithKnowledgeBaseRuntimeDependencies(t, server.URL, systemClient, nil, nil, nil, nil, workflowSvc)
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}

	expectSourceJobCandidates(tenantMock, 77, nil, nil, nil)
	tenantMock.ExpectQuery("SELECT CASE WHEN EXISTS.*FROM knowledge_base_source_jobs legacy").
		WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(0))
	expectNoKnowledgeBaseRawVolumeFiles(tenantMock, 77)
	tenantMock.ExpectQuery("SELECT 1.*FROM data_asset vector").
		WithArgs("kb_77_text_index", int64(77)).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(1))

	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)

	resp, err := svc.ListSourceJobs(ctx, ListSemanticModelSourceJobsParams{ModelID: 77})
	if err != nil {
		t.Fatalf("ListSourceJobs: %v", err)
	}
	if resp.Total != 0 || len(resp.Items) != 0 || !resp.ReconcileRequired {
		t.Fatalf("source jobs response = %+v, want only lineage reconcile signal", resp)
	}
	if len(workflowSvc.listFileExecutionCalls) != 0 {
		t.Fatalf("lineage projection should not query file executions: %+v", workflowSvc.listFileExecutionCalls)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceListSourceJobsReturnsExplicitSourceOnlyReconcileSignal(t *testing.T) {
	tests := []struct {
		name       string
		files      any
		tables     any
		expectMiss func(sqlmock.Sqlmock)
	}{
		{
			name:   "explicit file",
			files:  map[string]any{"file_ids": []string{"explicit-file-1"}},
			tables: []any{},
			expectMiss: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT CASE WHEN EXISTS.*FROM \\(VALUES ROW\\(\\?\\)\\).*NOT EXISTS.*FROM knowledge_base_sources kbs.*kb_file_id.*source_file_id.*LIMIT 1").
					WithArgs("explicit-file-1", int64(77)).
					WillReturnRows(sqlmock.NewRows([]string{"missing"}).AddRow(1))
			},
		},
		{
			name:   "explicit table",
			files:  map[string]any{},
			tables: []semanticModelTableSource{{DBName: "sales", TableNames: []string{"orders"}}},
			expectMiss: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT CASE WHEN EXISTS.*FROM \\(VALUES ROW\\(\\?, \\?\\)\\).*NOT EXISTS.*FROM knowledge_base_sources kbs.*db_name.*table_name.*LIMIT 1").
					WithArgs("sales", "orders", int64(77)).
					WillReturnRows(sqlmock.NewRows([]string{"missing"}).AddRow(1))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.URL.Path != "/api/v1/workspaces/ws-1/semantic-models/77" {
					t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{
					"id": 77, "name": "kb_docs", "tables": tt.tables, "files": tt.files,
				})
			}))
			defer server.Close()
			systemClient, err := moi.New(server.URL, "system-key")
			if err != nil {
				t.Fatalf("moi.New: %v", err)
			}
			defer systemClient.Close()
			svc := newSemanticModelTestService(t, server.URL, systemClient)
			tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
			if err != nil {
				t.Fatalf("tenant sqlmock: %v", err)
			}
			defer tenantSQL.Close()
			tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
			if err != nil {
				t.Fatalf("open tenant gorm: %v", err)
			}
			expectSourceJobCandidates(tenantMock, 77, nil, nil, nil)
			tt.expectMiss(tenantMock)
			ctx := ctxutil.WithUID(context.Background(), "user-1")
			ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
			ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
			ctx = ctxutil.WithTenantDB(ctx, tenantDB)
			resp, err := svc.ListSourceJobs(ctx, ListSemanticModelSourceJobsParams{ModelID: 77})
			if err != nil {
				t.Fatalf("ListSourceJobs: %v", err)
			}
			if len(resp.Items) != 0 || resp.Total != 0 || !resp.ReconcileRequired {
				t.Fatalf("source jobs response = %+v, want only explicit source reconcile signal", resp)
			}
			if err := tenantMock.ExpectationsWereMet(); err != nil {
				t.Fatalf("tenant sql expectations: %v", err)
			}
		})
	}
}

func TestExplicitSemanticModelSourceBackfillRequiredUsesOneQueryForTenThousandResources(t *testing.T) {
	tests := []struct {
		name      string
		model     func(t *testing.T) *SemanticModelInfo
		query     string
		queryArgs func() []driver.Value
	}{
		{
			name: "files",
			model: func(t *testing.T) *SemanticModelInfo {
				fileIDs := make([]string, 10_000)
				for i := range fileIDs {
					fileIDs[i] = fmt.Sprintf("explicit-file-%05d", i)
				}
				raw, err := json.Marshal(map[string]any{"file_ids": fileIDs})
				if err != nil {
					t.Fatalf("marshal files: %v", err)
				}
				return &SemanticModelInfo{ID: 77, Files: raw}
			},
			query: "SELECT CASE WHEN EXISTS.*FROM \\(VALUES ROW\\(\\?\\)",
			queryArgs: func() []driver.Value {
				args := make([]driver.Value, 0, 10_001)
				for i := 0; i < 10_000; i++ {
					args = append(args, fmt.Sprintf("explicit-file-%05d", i))
				}
				return append(args, int64(77))
			},
		},
		{
			name: "tables",
			model: func(t *testing.T) *SemanticModelInfo {
				tableNames := make([]string, 10_000)
				for i := range tableNames {
					tableNames[i] = fmt.Sprintf("explicit_table_%05d", i)
				}
				raw, err := json.Marshal([]semanticModelTableSource{{DBName: "sales", TableNames: tableNames}})
				if err != nil {
					t.Fatalf("marshal tables: %v", err)
				}
				return &SemanticModelInfo{ID: 77, Tables: raw}
			},
			query: "SELECT CASE WHEN EXISTS.*FROM \\(VALUES ROW\\(\\?, \\?\\)",
			queryArgs: func() []driver.Value {
				args := make([]driver.Value, 0, 20_001)
				for i := 0; i < 10_000; i++ {
					args = append(args, "sales", fmt.Sprintf("explicit_table_%05d", i))
				}
				return append(args, int64(77))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
			if err != nil {
				t.Fatalf("tenant sqlmock: %v", err)
			}
			defer tenantSQL.Close()
			tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
			if err != nil {
				t.Fatalf("open tenant gorm: %v", err)
			}
			tenantMock.ExpectQuery(tt.query).
				WithArgs(tt.queryArgs()...).
				WillReturnRows(sqlmock.NewRows([]string{"missing"}).AddRow(0))
			ctx := ctxutil.WithTenantDB(context.Background(), tenantDB)
			required, err := (&semanticModelService{}).explicitSemanticModelSourceBackfillRequired(ctx, tt.model(t))
			if err != nil {
				t.Fatalf("explicitSemanticModelSourceBackfillRequired: %v", err)
			}
			if required {
				t.Fatal("all explicit resources are present, want no backfill")
			}
			if err := tenantMock.ExpectationsWereMet(); err != nil {
				t.Fatalf("tenant sql expectations: %v", err)
			}
		})
	}
}

func TestSemanticModelServiceListSourceJobsReturnsNoWorkSignal(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/workspaces/ws-1/semantic-models/77" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 77, "name": "kb_docs", "tables": []any{}, "files": map[string]any{}})
	}))
	defer server.Close()
	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()
	svc := newSemanticModelTestService(t, server.URL, systemClient)
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	expectSourceJobCandidates(tenantMock, 77, nil, nil, nil)
	expectNoLegacySourceJobReconcileWork(tenantMock, 77)
	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)
	resp, err := svc.ListSourceJobs(ctx, ListSemanticModelSourceJobsParams{ModelID: 77})
	if err != nil {
		t.Fatalf("ListSourceJobs: %v", err)
	}
	if len(resp.Items) != 0 || resp.Total != 0 || resp.ReconcileRequired {
		t.Fatalf("source jobs response = %+v, want no work", resp)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestListKnowledgeBaseSourceJobCandidateIDsLimitsItemsButKeepsTotal(t *testing.T) {
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	tenantMock.ExpectQuery("SELECT COUNT\\(DISTINCT kbs\\.source_id\\).*FROM knowledge_base_source_job_runs jr").
		WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(40))
	rows := sqlmock.NewRows([]string{"source_id"})
	for i := 0; i < kbSourceJobListBatchSize; i++ {
		rows.AddRow(fmt.Sprintf("source-%02d", i))
	}
	tenantMock.ExpectQuery("SELECT kbs\\.source_id.*ORDER BY.*LIMIT \\?").
		WithArgs(int64(77), kbSourceJobListBatchSize).
		WillReturnRows(rows)

	ctx := ctxutil.WithTenantDB(context.Background(), tenantDB)
	sourceIDs, total, err := (&semanticModelService{}).listKnowledgeBaseSourceJobCandidateIDs(ctx, 77, kbSourceJobListBatchSize)
	if err != nil {
		t.Fatalf("listKnowledgeBaseSourceJobCandidateIDs: %v", err)
	}
	if len(sourceIDs) != kbSourceJobListBatchSize || total != 40 {
		t.Fatalf("candidate ids = %d, total = %d, want %d and 40", len(sourceIDs), total, kbSourceJobListBatchSize)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestLegacySourceJobReconcileRequiredUsesBoundedExistenceChecks(t *testing.T) {
	tests := []struct {
		name       string
		legacy     bool
		raw        bool
		lineage    bool
		want       bool
		withVector bool
	}{
		{name: "legacy job", legacy: true, want: true},
		{name: "raw volume", raw: true, want: true},
		{name: "lineage", lineage: true, want: true, withVector: true},
		{name: "no work", withVector: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
			if err != nil {
				t.Fatalf("tenant sqlmock: %v", err)
			}
			defer tenantSQL.Close()
			tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
			if err != nil {
				t.Fatalf("open tenant gorm: %v", err)
			}
			legacyValue := 0
			if tt.legacy {
				legacyValue = 1
			}
			tenantMock.ExpectQuery("SELECT CASE WHEN EXISTS.*FROM knowledge_base_source_jobs legacy.*LIMIT 1").WithArgs(int64(77)).WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(legacyValue))
			if !tt.legacy {
				if tt.raw {
					tenantMock.ExpectQuery("SELECT raw_volume_id\\s+FROM knowledge_base_raw_volumes").
						WithArgs(int64(77)).
						WillReturnRows(sqlmock.NewRows([]string{"raw_volume_id"}).AddRow(int64(12)))
					tenantMock.ExpectQuery("SELECT raw_volume_id\\s+FROM knowledge_base_data_domains").
						WithArgs(int64(77)).
						WillReturnRows(sqlmock.NewRows([]string{"raw_volume_id"}))
					expectRawVolumeLegacySourceQuery(tenantMock, 77, []int64{12}, true)
				} else {
					expectNoKnowledgeBaseRawVolumeFiles(tenantMock, 77)
				}
			}
			model := &SemanticModelInfo{ID: 77}
			if tt.withVector {
				model.Files = json.RawMessage(`{"vector_table":"kb_77_text_index"}`)
			}
			if !tt.legacy && !tt.raw && tt.withVector {
				lineageRows := sqlmock.NewRows([]string{"exists"})
				if tt.lineage {
					lineageRows.AddRow(1)
				}
				tenantMock.ExpectQuery("SELECT 1.*FROM data_asset vector.*NOT EXISTS.*LIMIT 1").
					WithArgs("kb_77_text_index", int64(77)).
					WillReturnRows(lineageRows)
			}
			ctx := ctxutil.WithTenantDB(context.Background(), tenantDB)
			got, err := (&semanticModelService{}).legacySourceJobReconcileRequired(ctx, model)
			if err != nil {
				t.Fatalf("legacySourceJobReconcileRequired: %v", err)
			}
			if got != tt.want {
				t.Fatalf("required = %v, want %v", got, tt.want)
			}
			if err := tenantMock.ExpectationsWereMet(); err != nil {
				t.Fatalf("tenant sql expectations: %v", err)
			}
		})
	}
}

func expectRawVolumeLegacySourceQuery(mock sqlmock.Sqlmock, modelID int64, volumeIDs []int64, exists bool) {
	query := fmt.Sprintf(rawVolumeLegacySourceExistsQueryFormat, queryPlaceholders(len(volumeIDs)))
	args := make([]driver.Value, 0, len(volumeIDs)+2)
	args = append(args, modelID, modelID)
	for _, volumeID := range volumeIDs {
		args = append(args, volumeID)
	}
	rows := sqlmock.NewRows([]string{"exists"})
	if exists {
		rows.AddRow(1)
	}
	mock.ExpectQuery(regexp.QuoteMeta(query)).WithArgs(args...).WillReturnRows(rows)
}

func TestRawVolumeLegacySourceExistsQueryUsesCanonicalIdentityPrecedence(t *testing.T) {
	query := fmt.Sprintf(rawVolumeLegacySourceExistsQueryFormat, "?")
	for _, clause := range []string{
		"SELECT kb_file_id AS file_id",
		"WHERE model_id = ? AND NULLIF(kb_file_id, '') IS NOT NULL",
		"SELECT source_file_id AS file_id",
		"AND NULLIF(kb_file_id, '') IS NULL",
		"AND NULLIF(source_file_id, '') IS NOT NULL",
		"AND linked.file_id IS NULL",
		"LIMIT 1",
	} {
		if !strings.Contains(query, clause) {
			t.Fatalf("raw-volume anti-join query missing %q", clause)
		}
	}
	if strings.Contains(query, "kb_file_id = vf.file_id OR") {
		t.Fatal("raw-volume anti-join query must not restore the correlated cross-identity OR")
	}
}

func TestRawVolumeLegacySourceExistsReturnsWorkFromSingleAntiJoinQuery(t *testing.T) {
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	tenantMock.ExpectQuery("SELECT raw_volume_id\\s+FROM knowledge_base_raw_volumes").
		WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows([]string{"raw_volume_id"}).AddRow(int64(12)))
	tenantMock.ExpectQuery("SELECT raw_volume_id\\s+FROM knowledge_base_data_domains").
		WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows([]string{"raw_volume_id"}))
	expectRawVolumeLegacySourceQuery(tenantMock, 77, []int64{12}, true)

	ctx := ctxutil.WithTenantDB(context.Background(), tenantDB)
	required, err := (&semanticModelService{}).rawVolumeLegacySourceExists(ctx, 77)
	if err != nil {
		t.Fatalf("rawVolumeLegacySourceExists: %v", err)
	}
	if !required {
		t.Fatal("raw volume missing file should require reconcile")
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestRawVolumeLegacySourceExistsReturnsNoWorkWithFixedQueryCountForLargeLinkedVolume(t *testing.T) {
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	tenantMock.ExpectQuery("SELECT raw_volume_id\\s+FROM knowledge_base_raw_volumes").
		WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows([]string{"raw_volume_id"}).AddRow(int64(12)).AddRow(int64(13)))
	tenantMock.ExpectQuery("SELECT raw_volume_id\\s+FROM knowledge_base_data_domains").
		WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows([]string{"raw_volume_id"}))
	// The database has already proven that no unmatched file exists. The service
	// issues this single anti-join query regardless of whether the volume contains
	// one row or many former application-side batches.
	expectRawVolumeLegacySourceQuery(tenantMock, 77, []int64{12, 13}, false)

	ctx := ctxutil.WithTenantDB(context.Background(), tenantDB)
	required, err := (&semanticModelService{}).rawVolumeLegacySourceExists(ctx, 77)
	if err != nil {
		t.Fatalf("rawVolumeLegacySourceExists: %v", err)
	}
	if required {
		t.Fatal("fully linked raw volume should not require reconcile")
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestRawVolumeLegacySourceExistsHandlesLargeFullyLinkedVolume(t *testing.T) {
	db, err := gorm.Open(gsqlite.Open("file:raw-volume-legacy-exists?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	for _, statement := range []string{
		`CREATE TABLE knowledge_base_raw_volumes (model_id INTEGER, raw_volume_id INTEGER, raw_kind TEXT)`,
		`CREATE TABLE knowledge_base_data_domains (model_id INTEGER, raw_volume_id INTEGER)`,
		`CREATE TABLE volume_files (id INTEGER PRIMARY KEY, volume_id INTEGER, file_id TEXT)`,
		`CREATE TABLE volume (volume_id INTEGER PRIMARY KEY, deleted INTEGER, catalog_id INTEGER, database_id INTEGER, volume_name TEXT)`,
		`CREATE TABLE ` + "`file`" + ` (file_id TEXT PRIMARY KEY)`,
		`CREATE TABLE catalog_database (database_id INTEGER PRIMARY KEY, catalog_id INTEGER, database_name TEXT)`,
		`CREATE TABLE catalog (catalog_id INTEGER PRIMARY KEY, catalog_name TEXT)`,
		`CREATE TABLE knowledge_base_sources (source_id TEXT PRIMARY KEY, model_id INTEGER, source_type TEXT, kb_file_id TEXT, source_file_id TEXT)`,
		`CREATE UNIQUE INDEX uk_volume_file ON volume_files (volume_id, file_id)`,
		`CREATE INDEX idx_kbs_kb_file ON knowledge_base_sources (kb_file_id)`,
		`CREATE INDEX idx_kbs_source_file ON knowledge_base_sources (model_id, source_type, source_file_id)`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("create sqlite fixture schema: %v", err)
		}
	}
	for _, statement := range []string{
		`INSERT INTO knowledge_base_raw_volumes (model_id, raw_volume_id, raw_kind) VALUES (77, 12, 'document')`,
		`INSERT INTO volume (volume_id, deleted, catalog_id, database_id, volume_name) VALUES (12, 0, 3, 11, 'raw')`,
		`INSERT INTO catalog_database (database_id, catalog_id, database_name) VALUES (11, 3, 'kb')`,
		`INSERT INTO catalog (catalog_id, catalog_name) VALUES (3, 'catalog')`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("insert sqlite fixture metadata: %v", err)
		}
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("sqlite sql db: %v", err)
	}
	tx, err := sqlDB.Begin()
	if err != nil {
		t.Fatalf("begin sqlite fixture transaction: %v", err)
	}
	fileStmt, err := tx.Prepare(`INSERT INTO ` + "`file`" + ` (file_id) VALUES (?)`)
	if err != nil {
		t.Fatalf("prepare file insert: %v", err)
	}
	defer fileStmt.Close()
	volumeFileStmt, err := tx.Prepare(`INSERT INTO volume_files (id, volume_id, file_id) VALUES (?, 12, ?)`)
	if err != nil {
		t.Fatalf("prepare volume file insert: %v", err)
	}
	defer volumeFileStmt.Close()
	sourceStmt, err := tx.Prepare(`INSERT INTO knowledge_base_sources (source_id, model_id, source_type, kb_file_id, source_file_id) VALUES (?, 77, 'local_file', ?, ?)`)
	if err != nil {
		t.Fatalf("prepare source insert: %v", err)
	}
	defer sourceStmt.Close()
	linkedFileCount := 3*kbLegacyBackfillBatchSize + 1
	for i := 1; i <= linkedFileCount; i++ {
		fileID := fmt.Sprintf("raw-file-%03d", i)
		if _, err := fileStmt.Exec(fileID); err != nil {
			t.Fatalf("insert file %d: %v", i, err)
		}
		if _, err := volumeFileStmt.Exec(i, fileID); err != nil {
			t.Fatalf("insert volume file %d: %v", i, err)
		}
		var kbFileID, sourceFileID any
		switch i % 3 {
		case 0:
			kbFileID = fileID
		case 1:
			sourceFileID = fileID
		default:
			kbFileID = fileID
			sourceFileID = "origin-" + fileID
		}
		if _, err := sourceStmt.Exec(fmt.Sprintf("source-%03d", i), kbFileID, sourceFileID); err != nil {
			t.Fatalf("insert knowledge source %d: %v", i, err)
		}
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit sqlite fixture transaction: %v", err)
	}

	ctx := ctxutil.WithTenantDB(context.Background(), db)
	required, err := (&semanticModelService{}).rawVolumeLegacySourceExists(ctx, 77)
	if err != nil {
		t.Fatalf("fully linked rawVolumeLegacySourceExists: %v", err)
	}
	if required {
		t.Fatal("large fully linked raw volume should not require reconcile")
	}

	missingFileID := "raw-file-missing"
	if err := db.Exec(`INSERT INTO `+"`file`"+` (file_id) VALUES (?)`, missingFileID).Error; err != nil {
		t.Fatalf("insert missing file metadata: %v", err)
	}
	if err := db.Exec(`INSERT INTO volume_files (id, volume_id, file_id) VALUES (?, 12, ?)`, linkedFileCount+1, missingFileID).Error; err != nil {
		t.Fatalf("insert missing volume file: %v", err)
	}
	required, err = (&semanticModelService{}).rawVolumeLegacySourceExists(ctx, 77)
	if err != nil {
		t.Fatalf("missing rawVolumeLegacySourceExists: %v", err)
	}
	if !required {
		t.Fatal("unlinked file after a large linked prefix should require reconcile")
	}
}

func TestRunningSourceJobsRotateByOldestUpdatedAtAcrossRounds(t *testing.T) {
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	queryArgs := []driver.Value{int64(77), kbJobTypeRAGIngest, kbSourceJobRunning, kbSourceStatusRemoved, kbSourceStatusFailed, kbSourceJobFailed, kbSourceJobSucceeded, kbSourceStatusSucceeded, 1}
	for round, jobID := range []string{"job-running-oldest", "job-running-next"} {
		tenantMock.ExpectQuery("SELECT .*FROM knowledge_base_source_job_runs jr.*ORDER BY jr\\.updated_at ASC, jr\\.job_id ASC.*LIMIT \\?").
			WithArgs(queryArgs...).
			WillReturnRows(knowledgeBaseSourceJobRunRows().AddRow(jobID, "source-file-1", int64(77), kbJobTypeRAGIngest, kbSourceJobRunning, "idem-"+jobID, nil, nil, nil, nil, false, "source-file", "kb-file", nil, nil, int64(0), nil, nil, int64(100), int64(100+round)))
		if round == 0 {
			tenantMock.ExpectExec("UPDATE knowledge_base_source_job_runs.*updated_at = CURRENT_TIMESTAMP").
				WithArgs(kbSourceJobRunning, "actor", jobID, kbSourceJobQueued, kbSourceJobRunning).
				WillReturnResult(sqlmock.NewResult(0, 1))
		}
	}
	ctx := ctxutil.WithTenantDB(context.Background(), tenantDB)
	svc := &semanticModelService{}
	first, err := svc.listRAGIngestKnowledgeBaseSourceJobRuns(ctx, 77, []string{kbSourceJobRunning}, 1, true)
	if err != nil || len(first) != 1 || first[0].JobID != "job-running-oldest" {
		t.Fatalf("first running batch = %+v, err = %v", first, err)
	}
	if err := svc.markKnowledgeBaseSourceJobRunning(ctx, first[0].JobID, "actor"); err != nil {
		t.Fatalf("mark first running job checked: %v", err)
	}
	second, err := svc.listRAGIngestKnowledgeBaseSourceJobRuns(ctx, 77, []string{kbSourceJobRunning}, 1, true)
	if err != nil || len(second) != 1 || second[0].JobID != "job-running-next" {
		t.Fatalf("second running batch = %+v, err = %v", second, err)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestFailedRAGSourceJobsRotateByOldestUpdatedAtAcrossRounds(t *testing.T) {
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	queryArgs := []driver.Value{int64(77), kbJobTypeRAGIngest, kbSourceJobFailed, kbSourceStatusRemoved, kbSourceStatusFailed, kbSourceJobFailed, kbSourceJobSucceeded, kbSourceStatusSucceeded, 1}
	for round, jobID := range []string{"job-failed-oldest", "job-failed-next"} {
		tenantMock.ExpectQuery("SELECT .*FROM knowledge_base_source_job_runs jr.*ORDER BY jr\\.updated_at ASC, jr\\.job_id ASC.*LIMIT \\?").
			WithArgs(queryArgs...).
			WillReturnRows(knowledgeBaseSourceJobRunRows().AddRow(jobID, "source-file-1", int64(77), kbJobTypeRAGIngest, kbSourceJobFailed, "idem-"+jobID, nil, "exec-failed", nil, nil, false, "source-file", "kb-file", nil, nil, int64(0), nil, "parse failed", int64(100), int64(100+round)))
		if round == 0 {
			tenantMock.ExpectExec("UPDATE knowledge_base_source_job_runs.*updated_at = CURRENT_TIMESTAMP").
				WithArgs("actor", jobID, kbSourceJobFailed).
				WillReturnResult(sqlmock.NewResult(0, 1))
		}
	}
	ctx := ctxutil.WithTenantDB(context.Background(), tenantDB)
	svc := &semanticModelService{}
	first, err := svc.listRAGIngestKnowledgeBaseSourceJobRuns(ctx, 77, []string{kbSourceJobFailed}, 1, true)
	if err != nil || len(first) != 1 || first[0].JobID != "job-failed-oldest" {
		t.Fatalf("first failed batch = %+v, err = %v", first, err)
	}
	if err := svc.markKnowledgeBaseSourceJobFailedChecked(ctx, first[0].JobID, "actor"); err != nil {
		t.Fatalf("mark first failed job checked: %v", err)
	}
	second, err := svc.listRAGIngestKnowledgeBaseSourceJobRuns(ctx, 77, []string{kbSourceJobFailed}, 1, true)
	if err != nil || len(second) != 1 || second[0].JobID != "job-failed-next" {
		t.Fatalf("second failed batch = %+v, err = %v", second, err)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestKnowledgeBaseSourceJobCandidateMatchesFailedSourceExecutionRule(t *testing.T) {
	if !strings.Contains(knowledgeBaseSourceJobCandidateFromWhere, "kbs.status <> 'failed' OR (jr.job_type = 'rag_ingest' AND jr.job_status = 'failed')") {
		t.Fatalf("candidate query must exclude pending/running RAG jobs owned by failed sources: %s", knowledgeBaseSourceJobCandidateFromWhere)
	}
}

func TestSemanticModelServiceStructuredRunningJobsRotateAcrossReconcileRounds(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/workspaces/ws-1/semantic-models/77" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": 77, "name": "kb_docs", "tables": []any{}, "files": map[string]any{},
		})
	}))
	defer server.Close()
	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()
	svc := newSemanticModelTestService(t, server.URL, systemClient)
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}

	expectRound := func(start, count int) {
		expectLegacyBackfillNoop(tenantMock, 77)
		tenantMock.ExpectQuery("SELECT jr.job_id.*COALESCE\\(krv.raw_kind").
			WillReturnRows(knowledgeBaseSourceJobRunRows())
		tenantMock.ExpectQuery("SELECT .*FROM knowledge_base_source_job_runs jr.*jr.job_type = \\?.*ORDER BY jr.created_at ASC").
			WithArgs(int64(77), kbJobTypeTableClone, kbSourceJobPending, kbSourceJobQueued, kbSourceJobRunning, kbSourceStatusFailed, kbSourceStatusRemoved, kbSourceJobReconcileBatchSize).
			WillReturnRows(knowledgeBaseSourceJobRunRows())
		tenantMock.ExpectQuery("SELECT .*FROM knowledge_base_source_job_runs jr.*jr.job_type = \\?.*ORDER BY jr.created_at ASC").
			WithArgs(int64(77), kbJobTypeLoad, kbSourceJobSucceeded, kbSourceStatusRemoved, kbSourceTypeCatalogTable, kbSourceTypeLocalFile, kbRawKindStructured, kbSourceJobSucceeded, kbSourceStatusSucceeded, kbSourceJobReconcileBatchSize).
			WillReturnRows(knowledgeBaseSourceJobRunRows())
		jobRows := knowledgeBaseSourceJobRunRows()
		records := make([]KnowledgeBaseSourceRecord, 0, count)
		sourceArgs := make([]driver.Value, 0, count+2)
		sourceArgs = append(sourceArgs, int64(77))
		for i := start; i < start+count; i++ {
			jobID := fmt.Sprintf("job-structured-%02d", i)
			sourceID := fmt.Sprintf("source-structured-%02d", i)
			fileID := fmt.Sprintf("structured-file-%02d", i)
			taskID := fmt.Sprintf("task-structured-%02d", i)
			jobRows.AddRow(jobID, sourceID, int64(77), kbJobTypeLoad, kbSourceJobRunning, "idem-"+jobID, "import_task:"+taskID, nil, nil, nil, false, fileID, fileID, nil, nil, int64(0), nil, nil, int64(100), int64(100+i))
			records = append(records, KnowledgeBaseSourceRecord{SourceID: sourceID, ModelID: 77, RawVolumeID: 12, SourceType: kbSourceTypeLocalFile, SourceFileID: stringPtr(fileID), KBFileID: stringPtr(fileID), Status: kbSourceStatusPending, Enabled: boolPtr(true)})
			sourceArgs = append(sourceArgs, sourceID)
		}
		sourceArgs = append(sourceArgs, kbSourceStatusRemoved)
		tenantMock.ExpectQuery("SELECT .*FROM knowledge_base_source_job_runs jr.*jr.job_type = \\?.*ORDER BY jr.updated_at ASC, jr.job_id ASC.*LIMIT \\?").
			WithArgs(int64(77), kbJobTypeLoad, kbSourceJobQueued, kbSourceJobRunning, kbSourceStatusRemoved, kbSourceTypeCatalogTable, kbSourceTypeLocalFile, kbRawKindStructured, kbSourceJobSucceeded, kbSourceStatusSucceeded, kbSourceJobReconcileBatchSize).
			WillReturnRows(jobRows)
		tenantMock.ExpectQuery("SELECT .*source_id.*FROM knowledge_base_sources kbs.*source_id IN.*ORDER BY kbs.source_id ASC").
			WithArgs(sourceArgs...).
			WillReturnRows(knowledgeBaseSourceRecordRows(records...))
		for i := start; i < start+count; i++ {
			jobID := fmt.Sprintf("job-structured-%02d", i)
			taskID := fmt.Sprintf("task-structured-%02d", i)
			tenantMock.ExpectQuery("SELECT .* FROM `import_task` WHERE id = \\? ORDER BY `import_task`.`id` LIMIT \\?").
				WithArgs(taskID, 1).
				WillReturnRows(sqlmock.NewRows([]string{"id", "status", "task_meta"}).AddRow(taskID, model.ImportTaskStatusUploading, `{}`))
			tenantMock.ExpectQuery("SELECT .* FROM `import_task_run` WHERE import_task_id = \\? ORDER BY created_at DESC, id DESC,`import_task_run`.`id` LIMIT \\?").
				WithArgs(taskID, 1).
				WillReturnRows(sqlmock.NewRows([]string{"id", "status", "error_message"}))
			tenantMock.ExpectExec("UPDATE knowledge_base_source_job_runs.*updated_at = CURRENT_TIMESTAMP").
				WithArgs(kbSourceJobRunning, "user-1", jobID, kbSourceJobQueued, kbSourceJobRunning).
				WillReturnResult(sqlmock.NewResult(0, 1))
		}
	}

	expectRound(0, kbSourceJobReconcileBatchSize)
	expectRound(kbSourceJobReconcileBatchSize, 1)
	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithUserID(ctx, "user-1")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)
	for round := 0; round < 2; round++ {
		if err := svc.ReconcileKnowledgeBaseSourceJobs(ctx, ReconcileKnowledgeBaseSourceJobsParams{ModelID: 77}); err != nil {
			t.Fatalf("ReconcileKnowledgeBaseSourceJobs round %d: %v", round+1, err)
		}
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestKnowledgeBaseSourceJobViewsMergesFileLoadAndRAGJobs(t *testing.T) {
	kbFileID := "kb-file-1"
	sourceFileID := "source-file-1"
	views := knowledgeBaseSourceJobViews([]KnowledgeBaseSourceJobRun{
		{
			JobID:        "job-rag-1",
			SourceID:     "source-1",
			JobType:      kbJobTypeRAGIngest,
			JobStatus:    kbSourceJobSucceeded,
			SourceFileID: &sourceFileID,
			KBFileID:     &kbFileID,
			UpdatedAt:    200,
		},
		{
			JobID:        "job-load-1",
			SourceID:     "source-1",
			JobType:      kbJobTypeLoad,
			JobStatus:    kbSourceJobSucceeded,
			SourceFileID: &sourceFileID,
			KBFileID:     &kbFileID,
			UpdatedAt:    100,
		},
	})

	if len(views) != 1 {
		t.Fatalf("views = %+v, want one merged source job", views)
	}
	view := views[0]
	if view.SourceID != "source-1" || view.JobStatus != kbSourceJobSucceeded || !view.ReconcileRequired {
		t.Fatalf("merged view = %+v, want succeeded reconcile-required source job", view)
	}
	if view.KBFileID == nil || *view.KBFileID != kbFileID {
		t.Fatalf("kb_file_id = %v, want %s", view.KBFileID, kbFileID)
	}
	if view.UpdatedAt == nil || *view.UpdatedAt != 200 {
		t.Fatalf("updated_at = %v, want 200", view.UpdatedAt)
	}
}

func TestKnowledgeBaseSourceJobViewsMarksSucceededLoadJobReconcileRequired(t *testing.T) {
	views := knowledgeBaseSourceJobViews([]KnowledgeBaseSourceJobRun{
		{
			JobID:     "job-load-1",
			SourceID:  "source-structured-1",
			JobType:   kbJobTypeLoad,
			JobStatus: kbSourceJobSucceeded,
			UpdatedAt: 100,
		},
	})

	if len(views) != 1 {
		t.Fatalf("views = %+v, want one source job", views)
	}
	view := views[0]
	if view.SourceID != "source-structured-1" || view.JobStatus != kbSourceJobSucceeded || !view.ReconcileRequired {
		t.Fatalf("view = %+v, want succeeded reconcile-required load job", view)
	}
}

func TestKnowledgeBaseSourceJobViewsMarksCompleteSucceededJobNotReconcileRequired(t *testing.T) {
	kbFileID := "kb-file-1"
	segmentVersionID := "seg-v1"
	indexVersion := int64(1)
	source := KnowledgeBaseSourceRecord{
		SourceID:         "source-file-1",
		SourceType:       kbSourceTypeCatalogFile,
		Status:           kbSourceStatusSucceeded,
		KBFileID:         &kbFileID,
		SegmentVersionID: &segmentVersionID,
		IndexVersion:     &indexVersion,
	}
	views := knowledgeBaseSourceJobViews([]KnowledgeBaseSourceJobRun{
		{
			JobID:     "job-rag-1",
			SourceID:  source.SourceID,
			JobType:   kbJobTypeRAGIngest,
			JobStatus: kbSourceJobSucceeded,
			KBFileID:  &kbFileID,
			UpdatedAt: 100,
		},
	}, map[string]KnowledgeBaseSourceRecord{source.SourceID: source})

	if len(views) != 1 {
		t.Fatalf("views = %+v, want one source job", views)
	}
	if views[0].ReconcileRequired {
		t.Fatalf("view = %+v, want completed succeeded job reconcile_required=false", views[0])
	}
}

func TestKnowledgeBaseSourceJobViewsMarksSucceededJobMissingFinalBindingReconcileRequired(t *testing.T) {
	source := KnowledgeBaseSourceRecord{
		SourceID:   "source-table-1",
		SourceType: kbSourceTypeCatalogTable,
		Status:     kbSourceStatusSucceeded,
	}
	views := knowledgeBaseSourceJobViews([]KnowledgeBaseSourceJobRun{
		{
			JobID:     "job-load-1",
			SourceID:  source.SourceID,
			JobType:   kbJobTypeLoad,
			JobStatus: kbSourceJobSucceeded,
			UpdatedAt: 100,
		},
	}, map[string]KnowledgeBaseSourceRecord{source.SourceID: source})

	if len(views) != 1 {
		t.Fatalf("views = %+v, want one source job", views)
	}
	if !views[0].ReconcileRequired {
		t.Fatalf("view = %+v, want missing final binding reconcile_required=true", views[0])
	}
}

func TestKnowledgeBaseSourceJobViewsMarksFailedSourceTerminal(t *testing.T) {
	source := KnowledgeBaseSourceRecord{
		SourceID:   "source-file-1",
		SourceType: kbSourceTypeCatalogFile,
		Status:     kbSourceStatusFailed,
	}
	views := knowledgeBaseSourceJobViews([]KnowledgeBaseSourceJobRun{
		{
			JobID:     "job-rag-1",
			SourceID:  source.SourceID,
			JobType:   kbJobTypeRAGIngest,
			JobStatus: kbSourceJobPending,
			UpdatedAt: 100,
		},
	}, map[string]KnowledgeBaseSourceRecord{source.SourceID: source})

	if len(views) != 1 {
		t.Fatalf("views = %+v, want one source job", views)
	}
	if views[0].ReconcileRequired {
		t.Fatalf("view = %+v, want failed source reconcile_required=false", views[0])
	}
}

func TestKnowledgeBaseSourceJobViewsMarksPendingJobsReconcileRequired(t *testing.T) {
	for _, jobType := range []string{kbJobTypeCopy, kbJobTypeTableClone, kbJobTypeRAGIngest, kbJobTypeLoad} {
		views := knowledgeBaseSourceJobViews([]KnowledgeBaseSourceJobRun{
			{
				JobID:     "job-" + jobType,
				SourceID:  "source-" + jobType,
				JobType:   jobType,
				JobStatus: kbSourceJobPending,
				UpdatedAt: 100,
			},
		})
		if len(views) != 1 {
			t.Fatalf("views for %s = %+v, want one source job", jobType, views)
		}
		if !views[0].ReconcileRequired {
			t.Fatalf("view for %s = %+v, want reconcile_required", jobType, views[0])
		}
	}
}

func TestKnowledgeBaseSourceJobViewsMarksRunningJobsPollOnly(t *testing.T) {
	for _, jobType := range []string{kbJobTypeCopy, kbJobTypeTableClone, kbJobTypeRAGIngest, kbJobTypeLoad} {
		views := knowledgeBaseSourceJobViews([]KnowledgeBaseSourceJobRun{
			{
				JobID:     "job-" + jobType,
				SourceID:  "source-" + jobType,
				JobType:   jobType,
				JobStatus: kbSourceJobRunning,
				UpdatedAt: 100,
			},
		})
		if len(views) != 1 {
			t.Fatalf("views for %s = %+v, want one source job", jobType, views)
		}
		if views[0].ReconcileRequired {
			t.Fatalf("view for %s = %+v, want running job reconcile_required=false", jobType, views[0])
		}
	}
}

func TestKnowledgeBaseSourceJobReconcileRequiredMatrix(t *testing.T) {
	kbFileID := "kb-file-1"
	segmentVersionID := "segment-v1"
	validIndexVersion := int64(1)
	zeroIndexVersion := int64(0)
	completedFileSource := KnowledgeBaseSourceRecord{
		SourceID:         "source-file-1",
		SourceType:       kbSourceTypeCatalogFile,
		Status:           kbSourceStatusSucceeded,
		KBFileID:         &kbFileID,
		SegmentVersionID: &segmentVersionID,
		IndexVersion:     &validIndexVersion,
	}

	tests := []struct {
		name      string
		job       KnowledgeBaseSourceJobRun
		source    KnowledgeBaseSourceRecord
		hasSource bool
		want      bool
	}{
		{
			name: "pending job",
			job:  KnowledgeBaseSourceJobRun{JobStatus: kbSourceJobPending},
			want: true,
		},
		{
			name: "running job",
			job:  KnowledgeBaseSourceJobRun{JobStatus: kbSourceJobRunning},
			want: false,
		},
		{
			name: "succeeded job without source binding",
			job:  KnowledgeBaseSourceJobRun{JobType: kbJobTypeRAGIngest, JobStatus: kbSourceJobSucceeded},
			want: true,
		},
		{
			name:      "succeeded job with complete final binding",
			job:       KnowledgeBaseSourceJobRun{JobType: kbJobTypeRAGIngest, JobStatus: kbSourceJobSucceeded},
			source:    completedFileSource,
			hasSource: true,
			want:      false,
		},
		{
			name: "external failed job before source failure is persisted",
			job:  KnowledgeBaseSourceJobRun{JobType: kbJobTypeRAGIngest, JobStatus: kbSourceJobFailed},
			source: KnowledgeBaseSourceRecord{
				SourceType: kbSourceTypeCatalogFile,
				Status:     kbSourceStatusPending,
			},
			hasSource: true,
			want:      true,
		},
		{
			name:      "persisted source failure",
			job:       KnowledgeBaseSourceJobRun{JobStatus: kbSourceJobFailed},
			source:    KnowledgeBaseSourceRecord{Status: kbSourceStatusFailed},
			hasSource: true,
			want:      false,
		},
		{
			name:      "removed source",
			job:       KnowledgeBaseSourceJobRun{JobStatus: kbSourceJobPending},
			source:    KnowledgeBaseSourceRecord{Status: kbSourceStatusRemoved},
			hasSource: true,
			want:      false,
		},
		{
			name: "zero index version is not a complete RAG binding",
			job:  KnowledgeBaseSourceJobRun{JobType: kbJobTypeRAGIngest, JobStatus: kbSourceJobSucceeded},
			source: KnowledgeBaseSourceRecord{
				SourceID:         "source-file-2",
				SourceType:       kbSourceTypeCatalogFile,
				Status:           kbSourceStatusSucceeded,
				KBFileID:         &kbFileID,
				SegmentVersionID: &segmentVersionID,
				IndexVersion:     &zeroIndexVersion,
			},
			hasSource: true,
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := knowledgeBaseSourceJobReconcileRequired(tt.job, tt.source, tt.hasSource); got != tt.want {
				t.Fatalf("knowledgeBaseSourceJobReconcileRequired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSemanticModelServiceSkipsUnclaimedCopyLoadAndTableCloneJobs(t *testing.T) {
	var requests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/workspaces/ws-1/volumes/12/files" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()
	client, err := moi.New(server.URL, "caller-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer client.Close()

	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}

	dataDomainSvc := &fakeSemanticModelDataDomainService{databaseID: 11}
	svc := newSemanticModelTestServiceWithDependencies(t, "", nil, dataDomainSvc, nil).(*semanticModelService)
	ctx := ctxutil.WithTenantDB(context.Background(), tenantDB)
	actor := "user-1"
	fileID := "file-1"
	tableID := int64(1001)

	for _, jobID := range []string{"job-copy", "job-load", "job-fast", "job-table-clone"} {
		tenantMock.ExpectExec("UPDATE knowledge_base_source_job_runs").
			WithArgs(kbSourceJobRunning, actor, jobID, kbSourceJobPending, kbSourceJobQueued, kbSourceJobRunning, sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(0, 0))
	}

	claimed, err := svc.runKnowledgeBaseCatalogFileCopyJob(ctx, nil, "ws-1", &KnowledgeBaseSourceRecord{
		SourceID: "source-copy", SourceType: kbSourceTypeCatalogFile, SourceFileID: &fileID, RawVolumeID: 12,
	}, &KnowledgeBaseSourceJobRun{JobID: "job-copy"}, actor)
	if err != nil || claimed {
		t.Fatalf("unclaimed copy = (%v, %v), want false, nil", claimed, err)
	}

	claimed, err = svc.runKnowledgeBaseLocalFileLoadJob(ctx, client, "ws-1", &KnowledgeBaseSourceRecord{
		SourceID: "source-load", SourceType: kbSourceTypeLocalFile, SourceFileID: &fileID, RawVolumeID: 12,
	}, &KnowledgeBaseSourceJobRun{JobID: "job-load"}, actor)
	if err != nil || claimed {
		t.Fatalf("unclaimed load = (%v, %v), want false, nil", claimed, err)
	}
	if err := svc.runKnowledgeBaseFastFileBindJobs(ctx, client, "ws-1", map[string]KnowledgeBaseSourceRecord{
		"source-fast": {SourceID: "source-fast", SourceType: kbSourceTypeLocalFile, SourceFileID: &fileID, RawVolumeID: 12},
	}, []KnowledgeBaseSourceJobRun{{JobID: "job-fast", SourceID: "source-fast", JobType: kbJobTypeLoad}}, actor); err != nil {
		t.Fatalf("unclaimed fast load: %v", err)
	}

	table, claimed, err := svc.runKnowledgeBaseTableCloneJob(ctx, &KnowledgeBaseSourceRecord{
		SourceID: "source-table", SourceType: kbSourceTypeCatalogTable, SourceTableID: &tableID,
	}, &KnowledgeBaseSourceJobRun{JobID: "job-table-clone"}, actor)
	if err != nil || claimed || table != nil {
		t.Fatalf("unclaimed table clone = (%+v, %v, %v), want nil, false, nil", table, claimed, err)
	}
	if len(dataDomainSvc.calls) != 0 {
		t.Fatalf("unclaimed table clone should not call data domain service: %+v", dataDomainSvc.calls)
	}
	if requests != 0 {
		t.Fatalf("unclaimed file jobs made %d AddFiles requests", requests)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceTableCloneKeepsSourceDisplayName(t *testing.T) {
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	tenantMock.ExpectExec("UPDATE knowledge_base_source_job_runs").
		WillReturnResult(sqlmock.NewResult(0, 1))

	dataDomainSvc := &fakeSemanticModelDataDomainService{
		databaseID:       11,
		cloneTargetTable: "orders__kb_1234",
	}
	svc := newSemanticModelTestServiceWithDependencies(t, "", nil, dataDomainSvc, nil).(*semanticModelService)
	sourceTableID := int64(1001)
	source := &KnowledgeBaseSourceRecord{
		SourceID: "source-table", SourceType: kbSourceTypeCatalogTable,
		SourceTableID: &sourceTableID, DatabaseID: 11,
	}
	job := &KnowledgeBaseSourceJobRun{JobID: "job-table-clone", IdempotencyKey: "idem-1"}
	table, claimed, err := svc.runKnowledgeBaseTableCloneJob(
		ctxutil.WithTenantDB(context.Background(), tenantDB), source, job, "user-1",
	)
	if err != nil || !claimed {
		t.Fatalf("runKnowledgeBaseTableCloneJob = (%+v, %v, %v)", table, claimed, err)
	}
	if source.DisplayName == nil || *source.DisplayName != "orders" {
		t.Fatalf("display name = %v, want source table name", source.DisplayName)
	}
	if source.TableName == nil || *source.TableName != "orders__kb_1234" {
		t.Fatalf("table name = %v, want cloned target name", source.TableName)
	}
	if table == nil || !sameStringSet(table.TableNames, []string{"orders__kb_1234"}) {
		t.Fatalf("semantic table = %+v", table)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceClaimsStaleRunningSourceJob(t *testing.T) {
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}

	tenantMock.ExpectExec("UPDATE knowledge_base_source_job_runs.*updated_at = CURRENT_TIMESTAMP.*updated_at <").
		WithArgs(kbSourceJobRunning, "user-1", "job-stale", kbSourceJobPending, kbSourceJobQueued, kbSourceJobRunning, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	claimed, err := (&semanticModelService{}).claimKnowledgeBaseSourceJobRunning(
		ctxutil.WithTenantDB(context.Background(), tenantDB), "job-stale", "user-1",
	)
	if err != nil || !claimed {
		t.Fatalf("claim stale running job = (%v, %v), want true, nil", claimed, err)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceListSourceJobsDoesNotSyncLoadJobs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			requireSemanticModelExecutionHeaders(t, r)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":          77,
				"name":        "kb_docs",
				"description": "docs",
				"tables":      []any{},
				"files": map[string]any{
					"file_ids":                     []string{},
					"vector_table":                 "client_text_idx",
					"embedding_model":              "BAAI/bge-large-zh-v1.5",
					"image_vector_table":           "client_image_idx",
					"image_embedding_model":        "efficientnet-b3",
					"image_embedding_backend_id":   "-30010",
					"image_embedding_dimension":    1536,
					"image_preprocess_version":     "efficientnet-b3-v1-rgb-300-letterbox-imagenet",
					"image_distance_metric":        "cosine",
					"active_image_index_config_id": "image-index-config-1",
				},
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()
	dataDomainSvc := &fakeSemanticModelDataDomainService{databaseID: 11, volumeIDs: []int64{12, 13}}
	workflowTemplateSvc := &fakeSemanticModelWorkflowTemplateService{}
	workflowSvc := &fakeSemanticModelWorkflowService{}
	svc := newSemanticModelTestServiceWithKnowledgeBaseRuntimeDependencies(t, server.URL, systemClient, dataDomainSvc, nil, nil, workflowTemplateSvc, workflowSvc)

	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}

	expectSourceJobCandidates(tenantMock, 77, []string{"source-file-1"}, knowledgeBaseSourceJobRunRows().
		AddRow("job-load-1", "source-file-1", int64(77), kbJobTypeLoad, kbSourceJobQueued, "idem-load-1", "import_task:747507000002", "exec-load-owned-by-dataconn", nil, nil, false, "source-file", "kb-file", nil, nil, int64(0), nil, nil, int64(100), int64(101)),
		knowledgeBaseSourceRecordRows(KnowledgeBaseSourceRecord{SourceID: "source-file-1", ModelID: 77, RawVolumeID: 12, SourceType: kbSourceTypeLocalFile, SourceFileID: stringPtr("source-file"), KBFileID: stringPtr("kb-file"), Status: kbSourceStatusPending, Enabled: boolPtr(true)}))

	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)

	resp, err := svc.ListSourceJobs(ctx, ListSemanticModelSourceJobsParams{ModelID: 77})
	if err != nil {
		t.Fatalf("ListSourceJobs: %v", err)
	}
	if resp.Total != 1 || len(resp.Items) != 1 {
		t.Fatalf("source jobs response = %+v", resp)
	}
	job := resp.Items[0]
	if job.JobStatus != kbSourceJobQueued {
		t.Fatalf("job status = %q, want queued", job.JobStatus)
	}
	if job.Error != nil {
		t.Fatalf("job error = %v, want nil", job.Error)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceListSourceJobsMarksMissingWorkflowExecutionFailed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			requireSemanticModelExecutionHeaders(t, r)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":          77,
				"name":        "kb_docs",
				"description": "docs",
				"tables":      []any{},
				"files": map[string]any{
					"file_ids":                     []string{},
					"vector_table":                 "client_text_idx",
					"embedding_model":              "BAAI/bge-large-zh-v1.5",
					"image_vector_table":           "client_image_idx",
					"image_embedding_model":        "efficientnet-b3",
					"image_embedding_backend_id":   "-30010",
					"image_embedding_dimension":    1536,
					"image_preprocess_version":     "efficientnet-b3-v1-rgb-300-letterbox-imagenet",
					"image_distance_metric":        "cosine",
					"active_image_index_config_id": "image-index-config-1",
				},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/ws-1/workflow-apps/executions/missing-exec":
			requireSemanticModelExecutionHeaders(t, r)
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]any{"code": common.ErrorCode_NOT_FOUND, "message": "workflow not found"})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()
	dataDomainSvc := &fakeSemanticModelDataDomainService{databaseID: 11, volumeIDs: []int64{12, 13}}
	svc := newSemanticModelTestServiceWithKnowledgeBaseRuntimeDependencies(t, server.URL, systemClient, dataDomainSvc, nil, nil, nil, nil)

	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}

	expectSourceJobCandidates(tenantMock, 77, []string{"source-file-1"}, knowledgeBaseSourceJobRunRows().
		AddRow("job-rag-1", "source-file-1", int64(77), kbJobTypeRAGIngest, kbSourceJobPending, "idem-rag-1", nil, "missing-exec", nil, nil, false, "source-file", "kb-file", nil, nil, int64(0), nil, nil, int64(100), int64(101)),
		knowledgeBaseSourceRecordRows(KnowledgeBaseSourceRecord{SourceID: "source-file-1", ModelID: 77, RawVolumeID: 12, SourceType: kbSourceTypeCatalogFile, SourceFileID: stringPtr("source-file"), KBFileID: stringPtr("kb-file"), Status: kbSourceStatusPending, Enabled: boolPtr(true)}))

	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)

	resp, err := svc.ListSourceJobs(ctx, ListSemanticModelSourceJobsParams{ModelID: 77})
	if err != nil {
		t.Fatalf("ListSourceJobs: %v", err)
	}
	if resp.Total != 1 || len(resp.Items) != 1 {
		t.Fatalf("source jobs response = %+v", resp)
	}
	job := resp.Items[0]
	if job.JobStatus != kbSourceJobFailed {
		t.Fatalf("job status = %q, want failed", job.JobStatus)
	}
	if job.Error == nil || !strings.Contains(*job.Error, "workflow execution missing-exec not found") {
		t.Fatalf("job error = %v, want missing workflow execution error", job.Error)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceListSourceJobsUsesLatestFileExecutionForRAGIngest(t *testing.T) {
	// KB source jobs are scoped by kb_file_id. A newer execution for the same
	// KB file can come from a replacement workflow and must still drive the
	// displayed job state.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			requireSemanticModelExecutionHeaders(t, r)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":          77,
				"name":        "kb_docs",
				"description": "docs",
				"tables":      []any{},
				"files":       map[string]any{"file_ids": []string{}},
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()
	workflowID := knowledgeBaseWorkflowID("ws-1", 77)
	workflowSvc := &fakeSemanticModelWorkflowService{fileExecutions: map[string]*moi.FileExecutionsResponse{
		"kb-file": {
			Executions: []moi.FileExecutionSummary{
				{ExecutionID: "exec-rag-old", WorkflowID: workflowID, Status: "succeeded", UpdatedAt: "2026-07-03T09:00:00Z"},
				{ExecutionID: "exec-other", WorkflowID: "other-workflow", Status: "succeeded"},
				{ExecutionID: "exec-rag-new", WorkflowID: "extra-workflow", Status: "failed", Error: "parse failed", UpdatedAt: "2026-07-03T10:00:00Z"},
			},
			Total: 3,
		},
	}}
	svc := newSemanticModelTestServiceWithKnowledgeBaseRuntimeDependencies(t, server.URL, systemClient, nil, nil, nil, nil, workflowSvc)

	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}

	expectRAGSourceJobRows(tenantMock, 77, "workflow_trigger:"+workflowID, nil, KnowledgeBaseSourceRecord{})

	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)

	resp, err := svc.ListSourceJobs(ctx, ListSemanticModelSourceJobsParams{ModelID: 77})
	if err != nil {
		t.Fatalf("ListSourceJobs: %v", err)
	}
	if resp.Total != 1 || len(resp.Items) != 1 {
		t.Fatalf("source jobs response = %+v", resp)
	}
	job := resp.Items[0]
	if job.JobStatus != kbSourceJobFailed {
		t.Fatalf("job status = %q, want failed", job.JobStatus)
	}
	if job.Error == nil || *job.Error != "parse failed" {
		t.Fatalf("job error = %v, want parse failed", job.Error)
	}
	if job.KBFileID == nil || *job.KBFileID != "kb-file" {
		t.Fatalf("kb_file_id = %v, want kb-file", job.KBFileID)
	}
	if len(workflowSvc.listFileExecutionCalls) != 1 || workflowSvc.listFileExecutionCalls[0] != "kb-file" {
		t.Fatalf("ListFileExecutions calls = %+v", workflowSvc.listFileExecutionCalls)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceListSourceJobsMarksRunningFileExecutionPollOnly(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			requireSemanticModelExecutionHeaders(t, r)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":          77,
				"name":        "kb_docs",
				"description": "docs",
				"tables":      []any{},
				"files":       map[string]any{"file_ids": []string{}},
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()
	workflowID := knowledgeBaseWorkflowID("ws-1", 77)
	workflowSvc := &fakeSemanticModelWorkflowService{fileExecutions: map[string]*moi.FileExecutionsResponse{
		"kb-file": {
			Executions: []moi.FileExecutionSummary{
				{ExecutionID: "exec-rag-running", WorkflowID: workflowID, Status: "running", UpdatedAt: "2026-07-03T10:00:00Z"},
			},
			Total: 1,
		},
	}}
	svc := newSemanticModelTestServiceWithKnowledgeBaseRuntimeDependencies(t, server.URL, systemClient, nil, nil, nil, nil, workflowSvc)

	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}

	expectRAGSourceJobRows(tenantMock, 77, "workflow_trigger:"+workflowID, nil, KnowledgeBaseSourceRecord{})

	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)

	resp, err := svc.ListSourceJobs(ctx, ListSemanticModelSourceJobsParams{ModelID: 77})
	if err != nil {
		t.Fatalf("ListSourceJobs: %v", err)
	}
	if resp.Total != 1 || len(resp.Items) != 1 {
		t.Fatalf("source jobs response = %+v", resp)
	}
	job := resp.Items[0]
	if job.JobStatus != kbSourceJobRunning {
		t.Fatalf("job status = %q, want running", job.JobStatus)
	}
	if job.ReconcileRequired {
		t.Fatalf("job = %+v, want reconcile_required=false while workflow is running", job)
	}
	if job.KBFileID == nil || *job.KBFileID != "kb-file" {
		t.Fatalf("kb_file_id = %v, want kb-file", job.KBFileID)
	}
	if len(workflowSvc.listFileExecutionCalls) != 1 || workflowSvc.listFileExecutionCalls[0] != "kb-file" {
		t.Fatalf("ListFileExecutions calls = %+v", workflowSvc.listFileExecutionCalls)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceListSourceJobsKeepsPendingWhenRAGIngestFileExecutionNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			requireSemanticModelExecutionHeaders(t, r)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":          77,
				"name":        "kb_docs",
				"description": "docs",
				"tables":      []any{},
				"files":       map[string]any{"file_ids": []string{}},
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()
	workflowID := knowledgeBaseWorkflowID("ws-1", 77)
	workflowSvc := &fakeSemanticModelWorkflowService{fileExecutions: map[string]*moi.FileExecutionsResponse{
		"kb-file": {
			Executions: []moi.FileExecutionSummary{},
			Total:      0,
		},
	}}
	svc := newSemanticModelTestServiceWithKnowledgeBaseRuntimeDependencies(t, server.URL, systemClient, nil, nil, nil, nil, workflowSvc)

	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}

	expectRAGSourceJobRows(tenantMock, 77, "workflow_trigger:"+workflowID, nil, KnowledgeBaseSourceRecord{})

	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)

	resp, err := svc.ListSourceJobs(ctx, ListSemanticModelSourceJobsParams{ModelID: 77})
	if err != nil {
		t.Fatalf("ListSourceJobs: %v", err)
	}
	if resp.Total != 1 || len(resp.Items) != 1 {
		t.Fatalf("source jobs response = %+v", resp)
	}
	job := resp.Items[0]
	if job.JobStatus != kbSourceJobPending {
		t.Fatalf("job status = %q, want pending", job.JobStatus)
	}
	if job.Error != nil {
		t.Fatalf("job error = %v, want nil", job.Error)
	}
	if job.KBFileID == nil || *job.KBFileID != "kb-file" {
		t.Fatalf("kb_file_id = %v, want kb-file", job.KBFileID)
	}
	if len(workflowSvc.listFileExecutionCalls) != 1 || workflowSvc.listFileExecutionCalls[0] != "kb-file" {
		t.Fatalf("ListFileExecutions calls = %+v", workflowSvc.listFileExecutionCalls)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceFastFileBindJobsBatchesAddFilesByRawVolume(t *testing.T) {
	// Local load jobs batch AddFiles by KB raw volume. Catalog copy jobs finish
	// without AddFiles (file stays on the user volume).
	var addFilesCalls int
	var gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/workspaces/ws-1/volumes/12/files" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		addFilesCalls++
		gotAuth = r.Header.Get("X-API-Key")
		var payload struct {
			FileIDs []string `json:"file_ids"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode add files request: %v", err)
		}
		if !sameStringSet(payload.FileIDs, []string{"file-1", "file-2"}) {
			t.Fatalf("add files payload = %+v", payload.FileIDs)
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()
	client, err := moi.New(server.URL, "caller-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer client.Close()

	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	for _, jobID := range []string{"job-load-1", "job-load-2"} {
		tenantMock.ExpectExec("UPDATE knowledge_base_source_job_runs").
			WithArgs(kbSourceJobRunning, "actor", jobID, kbSourceJobPending, kbSourceJobQueued, kbSourceJobRunning, sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(0, 1))
	}
	for _, sourceID := range []string{"source-1", "source-2"} {
		tenantMock.ExpectBegin()
		tenantMock.ExpectQuery("SELECT status FROM knowledge_base_sources").
			WithArgs(sourceID).
			WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow(kbSourceStatusPending))
		tenantMock.ExpectExec("UPDATE knowledge_base_source_job_runs").WillReturnResult(sqlmock.NewResult(0, 1))
		tenantMock.ExpectExec("UPDATE knowledge_base_sources").WillReturnResult(sqlmock.NewResult(0, 1))
		tenantMock.ExpectCommit()
	}
	ctx := ctxutil.WithTenantDB(context.Background(), tenantDB)
	svc := &semanticModelService{}
	records := map[string]KnowledgeBaseSourceRecord{
		"source-1": {SourceID: "source-1", SourceType: kbSourceTypeLocalFile, RawVolumeID: 12, SourceFileID: stringPtr("file-1"), KBFileID: stringPtr("file-1")},
		"source-2": {SourceID: "source-2", SourceType: kbSourceTypeLocalFile, RawVolumeID: 12, SourceFileID: stringPtr("file-2"), KBFileID: stringPtr("file-2")},
	}
	jobs := []KnowledgeBaseSourceJobRun{
		{JobID: "job-load-1", SourceID: "source-1", ModelID: 77, JobType: kbJobTypeLoad, JobStatus: kbSourceJobPending, IdempotencyKey: "key-1"},
		{JobID: "job-load-2", SourceID: "source-2", ModelID: 77, JobType: kbJobTypeLoad, JobStatus: kbSourceJobQueued, IdempotencyKey: "key-2"},
	}
	if err := svc.runKnowledgeBaseFastFileBindJobs(ctx, client, "ws-1", records, jobs, "actor"); err != nil {
		t.Fatalf("run fast file bind jobs: %v", err)
	}
	if addFilesCalls != 1 {
		t.Fatalf("AddFiles calls = %d, want one healthy batch", addFilesCalls)
	}
	if gotAuth != "caller-key" {
		t.Fatalf("local AddFiles auth = %q, want caller-key", gotAuth)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceFastFileBindJobsPermissionDeniedReleasesBatchForImmediateRetry(t *testing.T) {
	var requests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 77, "name": "kb_docs", "tables": []any{}, "files": map[string]any{}})
			return
		}
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/workspaces/ws-1/volumes/12/files" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		requests++
		if requests > 1 {
			w.WriteHeader(http.StatusCreated)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"code":7,"message":"volume.write denied"}`))
	}))
	defer server.Close()
	client, err := moi.New(server.URL, "caller-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer client.Close()

	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	records := []KnowledgeBaseSourceRecord{
		{SourceID: "source-local-1", ModelID: 77, SourceType: kbSourceTypeLocalFile, RawVolumeID: 12, SourceFileID: stringPtr("local-file-1"), KBFileID: stringPtr("local-file-1"), Status: kbSourceStatusPending, Enabled: boolPtr(true)},
		{SourceID: "source-local-2", ModelID: 77, SourceType: kbSourceTypeLocalFile, RawVolumeID: 12, SourceFileID: stringPtr("local-file-2"), KBFileID: stringPtr("local-file-2"), Status: kbSourceStatusPending, Enabled: boolPtr(true)},
	}
	expectRound := func() {
		expectLegacyBackfillNoop(tenantMock, 77)
		tenantMock.ExpectQuery("SELECT jr.job_id.*COALESCE\\(krv.raw_kind").
			WithArgs(int64(77), kbSourceJobPending, kbSourceJobQueued, kbSourceJobRunning, kbSourceStatusFailed, kbSourceStatusRemoved, kbJobTypeCopy, kbJobTypeLoad, kbSourceTypeLocalFile, kbRawKindStructured, kbSourceJobFastBindBatchSize).
			WillReturnRows(knowledgeBaseSourceJobRunRows().
				AddRow("job-local-1", "source-local-1", int64(77), kbJobTypeLoad, kbSourceJobQueued, "key-local-1", nil, nil, nil, nil, false, "local-file-1", "local-file-1", nil, nil, int64(0), nil, nil, int64(100), int64(100)).
				AddRow("job-local-2", "source-local-2", int64(77), kbJobTypeLoad, kbSourceJobQueued, "key-local-2", nil, nil, nil, nil, false, "local-file-2", "local-file-2", nil, nil, int64(0), nil, nil, int64(100), int64(100)))
		tenantMock.ExpectQuery("SELECT .*FROM knowledge_base_source_job_runs jr.*jr.job_type = \\?.*ORDER BY jr.created_at ASC").
			WithArgs(int64(77), kbJobTypeTableClone, kbSourceJobPending, kbSourceJobQueued, kbSourceJobRunning, kbSourceStatusFailed, kbSourceStatusRemoved, kbSourceJobReconcileBatchSize).
			WillReturnRows(knowledgeBaseSourceJobRunRows())
		tenantMock.ExpectQuery("SELECT .*source_id.*FROM knowledge_base_sources kbs.*source_id IN.*ORDER BY kbs.source_id ASC").
			WithArgs(int64(77), "source-local-1", "source-local-2", kbSourceStatusRemoved).
			WillReturnRows(knowledgeBaseSourceRecordRows(records...))
	}
	expectRound()
	for _, jobID := range []string{"job-local-1", "job-local-2"} {
		tenantMock.ExpectExec("UPDATE knowledge_base_source_job_runs").
			WithArgs(kbSourceJobRunning, "actor", jobID, kbSourceJobPending, kbSourceJobQueued, kbSourceJobRunning, sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(0, 1))
	}
	tenantMock.ExpectExec("UPDATE knowledge_base_source_job_runs").
		WithArgs(kbSourceJobQueued, "actor", kbSourceJobRunning, "job-local-1", "job-local-2").
		WillReturnResult(sqlmock.NewResult(0, 2))
	expectRound()
	for _, jobID := range []string{"job-local-1", "job-local-2"} {
		tenantMock.ExpectExec("UPDATE knowledge_base_source_job_runs").
			WithArgs(kbSourceJobRunning, "actor", jobID, kbSourceJobPending, kbSourceJobQueued, kbSourceJobRunning, sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(0, 1))
	}
	for _, sourceID := range []string{"source-local-1", "source-local-2"} {
		tenantMock.ExpectBegin()
		tenantMock.ExpectQuery("SELECT status FROM knowledge_base_sources").
			WithArgs(sourceID).
			WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow(kbSourceStatusPending))
		tenantMock.ExpectExec("UPDATE knowledge_base_source_job_runs").WillReturnResult(sqlmock.NewResult(0, 1))
		tenantMock.ExpectExec("UPDATE knowledge_base_sources").WillReturnResult(sqlmock.NewResult(0, 1))
		tenantMock.ExpectCommit()
	}

	ctx := ctxutil.WithTenantDB(context.Background(), tenantDB)
	svc := &semanticModelService{}
	err = svc.reconcileKnowledgeBaseSourceJobs(ctx, client, "ws-1", 77, "actor", 1, false, false)
	if err == nil || !strings.Contains(err.Error(), "volume.write denied") {
		t.Fatalf("error = %v, want volume.write denial", err)
	}
	if err := svc.reconcileKnowledgeBaseSourceJobs(ctx, client, "ws-1", 77, "actor", 1, false, false); err != nil {
		t.Fatalf("immediate retry after permission recovery: %v", err)
	}
	if requests != 2 {
		t.Fatalf("AddFiles requests = %d, want first failure and immediate retry", requests)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("permission denial must release the full batch for immediate retry: %v", err)
	}
}

func TestSemanticModelServiceFastFileBindJobsUseCallerForLocalAndCatalogFiles(t *testing.T) {
	// Catalog copy finishes without AddFiles (user volume); only local load hits VolumeFiles.
	// Catalog jobs are drained first, then local volume batches.
	type addCall struct {
		auth            string
		fileIDs         []string
		requireUnlinked bool
	}
	var calls []addCall
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/workspaces/ws-1/volumes/12/files" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var payload struct {
			FileIDs         []string `json:"file_ids"`
			RequireUnlinked bool     `json:"require_unlinked"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode add files request: %v", err)
		}
		calls = append(calls, addCall{auth: r.Header.Get("X-API-Key"), fileIDs: append([]string(nil), payload.FileIDs...), requireUnlinked: payload.RequireUnlinked})
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	callerClient, err := moi.New(server.URL, "caller-key")
	if err != nil {
		t.Fatalf("moi.New caller: %v", err)
	}
	defer callerClient.Close()
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	// catalog copy first (finish only; no KB raw AddFiles)
	tenantMock.ExpectExec("UPDATE knowledge_base_source_job_runs").
		WithArgs(kbSourceJobRunning, "actor", "job-copy-1", kbSourceJobPending, kbSourceJobQueued, kbSourceJobRunning, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	tenantMock.ExpectBegin()
	tenantMock.ExpectQuery("SELECT status FROM knowledge_base_sources").
		WithArgs("source-catalog").
		WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow(kbSourceStatusPending))
	tenantMock.ExpectExec("UPDATE knowledge_base_source_job_runs").WillReturnResult(sqlmock.NewResult(0, 1))
	tenantMock.ExpectCommit()
	// local load then batches AddFiles into KB raw
	tenantMock.ExpectExec("UPDATE knowledge_base_source_job_runs").
		WithArgs(kbSourceJobRunning, "actor", "job-local-1", kbSourceJobPending, kbSourceJobQueued, kbSourceJobRunning, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	tenantMock.ExpectBegin()
	tenantMock.ExpectQuery("SELECT status FROM knowledge_base_sources").
		WithArgs("source-local").
		WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow(kbSourceStatusPending))
	tenantMock.ExpectExec("UPDATE knowledge_base_source_job_runs").WillReturnResult(sqlmock.NewResult(0, 1))
	tenantMock.ExpectExec("UPDATE knowledge_base_sources").WillReturnResult(sqlmock.NewResult(0, 1))
	tenantMock.ExpectCommit()

	ctx := ctxutil.WithTenantDB(context.Background(), tenantDB)
	svc := &semanticModelService{}
	records := map[string]KnowledgeBaseSourceRecord{
		"source-local":   {SourceID: "source-local", SourceType: kbSourceTypeLocalFile, RawVolumeID: 12, SourceFileID: stringPtr("local-file"), KBFileID: stringPtr("local-file")},
		"source-catalog": {SourceID: "source-catalog", SourceType: kbSourceTypeCatalogFile, RawVolumeID: 12, SourceFileID: stringPtr("catalog-file"), KBFileID: stringPtr("catalog-file")},
	}
	jobs := []KnowledgeBaseSourceJobRun{
		{JobID: "job-local-1", SourceID: "source-local", ModelID: 77, JobType: kbJobTypeLoad, JobStatus: kbSourceJobPending, IdempotencyKey: "key-local"},
		{JobID: "job-copy-1", SourceID: "source-catalog", ModelID: 77, JobType: kbJobTypeCopy, JobStatus: kbSourceJobPending, IdempotencyKey: "key-copy"},
	}
	if err := svc.runKnowledgeBaseFastFileBindJobs(ctx, callerClient, "ws-1", records, jobs, "actor"); err != nil {
		t.Fatalf("run fast file bind jobs: %v", err)
	}
	if len(calls) != 1 {
		t.Fatalf("AddFiles calls = %d, want only local load (catalog skips KB raw AddFiles)", len(calls))
	}
	if calls[0].auth != "caller-key" {
		t.Fatalf("AddFiles auth = %q, want caller-key", calls[0].auth)
	}
	if len(calls[0].fileIDs) != 1 || calls[0].fileIDs[0] != "local-file" || !calls[0].requireUnlinked {
		t.Fatalf("local AddFiles call = %+v, want require_unlinked", calls[0])
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceFastFileBindJobsTerminalErrorIsolatesTransientRetry(t *testing.T) {
	// Isolation/retry applies to local load AddFiles batches only (catalog has no AddFiles).
	var calls [][]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			FileIDs []string `json:"file_ids"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode add files request: %v", err)
		}
		calls = append(calls, append([]string(nil), payload.FileIDs...))
		w.Header().Set("Content-Type", "application/json")
		if len(payload.FileIDs) > 1 || payload.FileIDs[0] == "missing-file" {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"code":3,"message":"file not found"}`))
			return
		}
		if payload.FileIDs[0] == "retry-file" {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"code":14,"message":"temporarily unavailable"}`))
			return
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()
	client, err := moi.New(server.URL, "caller-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer client.Close()

	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	for _, jobID := range []string{"job-missing", "job-retry", "job-good"} {
		tenantMock.ExpectExec("UPDATE knowledge_base_source_job_runs").
			WithArgs(kbSourceJobRunning, "actor", jobID, kbSourceJobPending, kbSourceJobQueued, kbSourceJobRunning, sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(0, 1))
	}
	tenantMock.ExpectBegin()
	tenantMock.ExpectQuery("SELECT status FROM knowledge_base_sources").
		WithArgs("source-missing").
		WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow(kbSourceStatusPending))
	tenantMock.ExpectExec("UPDATE knowledge_base_source_job_runs").WillReturnResult(sqlmock.NewResult(0, 1))
	tenantMock.ExpectExec("UPDATE knowledge_base_sources").WillReturnResult(sqlmock.NewResult(0, 1))
	tenantMock.ExpectCommit()
	tenantMock.ExpectExec("UPDATE knowledge_base_source_job_runs").
		WithArgs(kbSourceJobQueued, "actor", kbSourceJobRunning, "job-retry").
		WillReturnResult(sqlmock.NewResult(0, 1))
	tenantMock.ExpectBegin()
	tenantMock.ExpectQuery("SELECT status FROM knowledge_base_sources").
		WithArgs("source-good").
		WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow(kbSourceStatusPending))
	tenantMock.ExpectExec("UPDATE knowledge_base_source_job_runs").WillReturnResult(sqlmock.NewResult(0, 1))
	tenantMock.ExpectExec("UPDATE knowledge_base_sources").WillReturnResult(sqlmock.NewResult(0, 1))
	tenantMock.ExpectCommit()

	ctx := ctxutil.WithTenantDB(context.Background(), tenantDB)
	records := map[string]KnowledgeBaseSourceRecord{
		"source-missing": {SourceID: "source-missing", SourceType: kbSourceTypeLocalFile, RawVolumeID: 12, SourceFileID: stringPtr("missing-file"), KBFileID: stringPtr("missing-file")},
		"source-retry":   {SourceID: "source-retry", SourceType: kbSourceTypeLocalFile, RawVolumeID: 12, SourceFileID: stringPtr("retry-file"), KBFileID: stringPtr("retry-file")},
		"source-good":    {SourceID: "source-good", SourceType: kbSourceTypeLocalFile, RawVolumeID: 12, SourceFileID: stringPtr("good-file"), KBFileID: stringPtr("good-file")},
	}
	jobs := []KnowledgeBaseSourceJobRun{
		{JobID: "job-missing", SourceID: "source-missing", JobType: kbJobTypeLoad, JobStatus: kbSourceJobPending},
		{JobID: "job-retry", SourceID: "source-retry", JobType: kbJobTypeLoad, JobStatus: kbSourceJobPending},
		{JobID: "job-good", SourceID: "source-good", JobType: kbJobTypeLoad, JobStatus: kbSourceJobPending},
	}
	err = (&semanticModelService{}).runKnowledgeBaseFastFileBindJobs(ctx, client, "ws-1", records, jobs, "actor")
	if err == nil || !strings.Contains(err.Error(), "temporarily unavailable") {
		t.Fatalf("run fast file bind jobs error = %v, want transient retry error", err)
	}
	if len(calls) != 4 || len(calls[0]) != 3 || calls[1][0] != "missing-file" || calls[2][0] != "retry-file" || calls[3][0] != "good-file" {
		t.Fatalf("AddFiles calls = %v, want batch then terminal per-file isolation", calls)
	}
	if records["source-missing"].Status != kbSourceStatusFailed {
		t.Fatalf("missing source status = %q, want failed", records["source-missing"].Status)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestFinishKnowledgeBaseFileBindJobSkipsRemovedSource(t *testing.T) {
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	tenantMock.ExpectBegin()
	tenantMock.ExpectQuery("SELECT status FROM knowledge_base_sources").
		WithArgs("source-removed").
		WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow(kbSourceStatusRemoved))
	tenantMock.ExpectCommit()

	finished, err := (&semanticModelService{}).finishKnowledgeBaseFileBindJob(
		ctxutil.WithTenantDB(context.Background(), tenantDB),
		&KnowledgeBaseSourceRecord{SourceID: "source-removed"},
		&KnowledgeBaseSourceJobRun{JobID: "job-deleted", JobType: kbJobTypeLoad},
		"file-1", errors.New("file not found"), "actor",
	)
	if err != nil {
		t.Fatalf("mark file bind failed: %v", err)
	}
	if finished {
		t.Fatal("removed source must not be changed")
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestFinishKnowledgeBaseFileBindJobDoesNotOverwriteCompletedJob(t *testing.T) {
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	tenantMock.ExpectBegin()
	tenantMock.ExpectQuery("SELECT status FROM knowledge_base_sources").
		WithArgs("source-active").
		WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow(kbSourceStatusPending))
	tenantMock.ExpectExec("UPDATE knowledge_base_source_job_runs").WillReturnResult(sqlmock.NewResult(0, 0))
	tenantMock.ExpectCommit()

	finished, err := (&semanticModelService{}).finishKnowledgeBaseFileBindJob(
		ctxutil.WithTenantDB(context.Background(), tenantDB),
		&KnowledgeBaseSourceRecord{SourceID: "source-active"},
		&KnowledgeBaseSourceJobRun{JobID: "job-completed", JobType: kbJobTypeLoad},
		"file-1", errors.New("file not found"), "actor",
	)
	if err != nil {
		t.Fatalf("finish file bind job: %v", err)
	}
	if finished {
		t.Fatal("completed job must not be overwritten by a stale failure")
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestFinishKnowledgeBaseFileBindJobPreservesCanonicalSourceAcrossKnowledgeBaseImports(t *testing.T) {
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	tenantMock.ExpectBegin()
	tenantMock.ExpectQuery("SELECT status FROM knowledge_base_sources").
		WithArgs("source-a3").
		WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow(kbSourceStatusPending))
	tenantMock.ExpectExec("UPDATE knowledge_base_source_job_runs").
		WithArgs(kbSourceJobSucceeded, kbJobOpCatalogFileBindPrefix+"a3", "a", "a3", "actor", "job-a3", "source-a3", kbSourceJobRunning).
		WillReturnResult(sqlmock.NewResult(0, 1))
	tenantMock.ExpectCommit()

	source := &KnowledgeBaseSourceRecord{SourceID: "source-a3", SourceFileID: stringPtr("a"), KBFileID: stringPtr("a3")}
	job := &KnowledgeBaseSourceJobRun{JobID: "job-a3", SourceID: "source-a3", JobType: kbJobTypeCopy, SourceFileID: stringPtr("a"), KBFileID: stringPtr("a3")}
	finished, err := (&semanticModelService{}).finishKnowledgeBaseFileBindJob(
		ctxutil.WithTenantDB(context.Background(), tenantDB), source, job, "a3", nil, "actor",
	)
	if err != nil || !finished {
		t.Fatalf("finish file bind job = (%v, %v), want true, nil", finished, err)
	}
	if ptrValue(source.SourceFileID) != "a" || ptrValue(source.KBFileID) != "a3" {
		t.Fatalf("source = %+v, want source_file_id=a and kb_file_id=a3", source)
	}
	if ptrValue(job.SourceFileID) != "a" || ptrValue(job.KBFileID) != "a3" {
		t.Fatalf("job = %+v, want source_file_id=a and kb_file_id=a3", job)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestKnowledgeBaseFileBindErrorClassification(t *testing.T) {
	tests := []struct {
		name string
		code common.ErrorCode
		want bool
	}{
		{name: "invalid argument", code: common.ErrorCode_INVALID_ARGUMENT, want: true},
		{name: "not found", code: common.ErrorCode_NOT_FOUND, want: true},
		{name: "file not found", code: common.ErrorCode_FILE_NOT_FOUND, want: true},
		{name: "permission denied", code: common.ErrorCode_PERMISSION_DENIED, want: false},
		{name: "internal", code: common.ErrorCode_INTERNAL, want: false},
		{name: "unavailable", code: common.ErrorCode_UNAVAILABLE, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isTerminalKnowledgeBaseFileBindError(&moi.Error{Code: tt.code}); got != tt.want {
				t.Fatalf("isTerminalKnowledgeBaseFileBindError(%v) = %v, want %v", tt.code, got, tt.want)
			}
		})
	}
}

func TestSemanticModelServiceLocalFileLoadJobUsesCallerWithoutListFilesDetail(t *testing.T) {
	var requests []string
	var gotAuth string
	var gotReq string
	var gotTrace string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		gotAuth = r.Header.Get("X-API-Key")
		gotReq = r.Header.Get("X-Request-ID")
		gotTrace = r.Header.Get("X-Trace-ID")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/workspaces/ws-1/volumes/12/files":
			w.WriteHeader(http.StatusCreated)
		case strings.Contains(r.URL.Path, "/files/detail"):
			t.Fatalf("local file load must not call ListFilesDetail: %s %s", r.Method, r.URL.Path)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	callerClient, err := moi.New(server.URL, "caller-key")
	if err != nil {
		t.Fatalf("moi.New caller: %v", err)
	}
	defer callerClient.Close()
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	tenantMock.ExpectExec("UPDATE knowledge_base_source_job_runs").
		WithArgs(kbSourceJobRunning, "actor", "job-load-1", kbSourceJobPending, kbSourceJobQueued, kbSourceJobRunning, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	tenantMock.ExpectBegin()
	tenantMock.ExpectQuery("SELECT status FROM knowledge_base_sources").
		WithArgs("source-local").
		WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow(kbSourceStatusPending))
	tenantMock.ExpectExec("UPDATE knowledge_base_source_job_runs").WillReturnResult(sqlmock.NewResult(0, 1))
	tenantMock.ExpectExec("UPDATE knowledge_base_sources").WillReturnResult(sqlmock.NewResult(0, 1))
	tenantMock.ExpectCommit()

	ctx := ctxutil.WithTenantDB(context.Background(), tenantDB)
	ctx = moi.ContextWithHeaders(ctx, map[string]string{
		"X-Request-ID": "req-local-add",
		"X-Trace-ID":   "trace-local-add",
	})
	svc := &semanticModelService{}
	source := &KnowledgeBaseSourceRecord{
		SourceID:     "source-local",
		SourceType:   kbSourceTypeLocalFile,
		RawVolumeID:  12,
		SourceFileID: stringPtr("local-file"),
		KBFileID:     stringPtr("local-file"),
	}
	job := &KnowledgeBaseSourceJobRun{
		JobID:          "job-load-1",
		SourceID:       "source-local",
		ModelID:        77,
		JobType:        kbJobTypeLoad,
		JobStatus:      kbSourceJobPending,
		IdempotencyKey: "key-local",
	}
	claimed, err := svc.runKnowledgeBaseLocalFileLoadJob(ctx, callerClient, "ws-1", source, job, "actor")
	if err != nil {
		t.Fatalf("run local file load: %v", err)
	}
	if !claimed {
		t.Fatal("expected job claimed")
	}
	if gotAuth != "caller-key" {
		t.Fatalf("AddFiles auth = %q, want caller-key", gotAuth)
	}
	if gotReq != "req-local-add" || gotTrace != "trace-local-add" {
		t.Fatalf("AddFiles trace headers = (%q, %q)", gotReq, gotTrace)
	}
	if len(requests) != 1 || requests[0] != "POST /api/v1/workspaces/ws-1/volumes/12/files" {
		t.Fatalf("requests = %v, want only AddFiles", requests)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceLocalFileLoadJobPermissionDeniedReleasesClaimForImmediateRetry(t *testing.T) {
	var requests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/workspaces/ws-1/volumes/12/files" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if requests == 1 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"code":7,"message":"volume.write denied"}`))
			return
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()
	callerClient, err := moi.New(server.URL, "caller-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer callerClient.Close()

	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	tenantMock.ExpectExec("UPDATE knowledge_base_source_job_runs").
		WithArgs(kbSourceJobRunning, "actor", "job-load-1", kbSourceJobPending, kbSourceJobQueued, kbSourceJobRunning, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	tenantMock.ExpectExec("UPDATE knowledge_base_source_job_runs").
		WithArgs(kbSourceJobQueued, "actor", kbSourceJobRunning, "job-load-1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	tenantMock.ExpectExec("UPDATE knowledge_base_source_job_runs").
		WithArgs(kbSourceJobRunning, "actor", "job-load-1", kbSourceJobPending, kbSourceJobQueued, kbSourceJobRunning, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	tenantMock.ExpectBegin()
	tenantMock.ExpectQuery("SELECT status FROM knowledge_base_sources").
		WithArgs("source-local").
		WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow(kbSourceStatusPending))
	tenantMock.ExpectExec("UPDATE knowledge_base_source_job_runs").WillReturnResult(sqlmock.NewResult(0, 1))
	tenantMock.ExpectExec("UPDATE knowledge_base_sources").WillReturnResult(sqlmock.NewResult(0, 1))
	tenantMock.ExpectCommit()
	ctx := ctxutil.WithTenantDB(context.Background(), tenantDB)
	svc := &semanticModelService{}
	source := &KnowledgeBaseSourceRecord{
		SourceID:     "source-local",
		SourceType:   kbSourceTypeLocalFile,
		RawVolumeID:  12,
		SourceFileID: stringPtr("local-file"),
		KBFileID:     stringPtr("local-file"),
	}
	job := &KnowledgeBaseSourceJobRun{
		JobID:          "job-load-1",
		SourceID:       "source-local",
		ModelID:        77,
		JobType:        kbJobTypeLoad,
		JobStatus:      kbSourceJobPending,
		IdempotencyKey: "key-local",
	}
	claimed, err := svc.runKnowledgeBaseLocalFileLoadJob(ctx, callerClient, "ws-1", source, job, "actor")
	if !claimed {
		t.Fatal("permission denial must claim the job before AddFiles")
	}
	if err == nil || !strings.Contains(err.Error(), "volume.write denied") {
		t.Fatalf("error = %v, want volume.write denial", err)
	}
	claimed, err = svc.runKnowledgeBaseLocalFileLoadJob(ctx, callerClient, "ws-1", source, job, "actor")
	if err != nil || !claimed {
		t.Fatalf("immediate retry after permission recovery: claimed=%v err=%v", claimed, err)
	}
	if requests != 2 {
		t.Fatalf("requests = %d, want first failure and immediate retry", requests)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceListSourceJobsDoesNotCompleteRAGFromExecutionWithoutSegment(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			requireSemanticModelExecutionHeaders(t, r)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":          77,
				"name":        "kb_docs",
				"description": "docs",
				"tables":      []any{},
				"files":       map[string]any{"file_ids": []string{}},
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()
	workflowSvc := &fakeSemanticModelWorkflowService{fileExecutions: map[string]*moi.FileExecutionsResponse{
		"kb-file": {
			Executions: []moi.FileExecutionSummary{
				{ExecutionID: "exec-old-completed", WorkflowID: "old-workflow", Status: "succeeded", UpdatedAt: "2026-07-03T10:00:00Z"},
			},
			Total: 1,
		},
	}}
	svc := newSemanticModelTestServiceWithKnowledgeBaseRuntimeDependencies(t, server.URL, systemClient, nil, nil, nil, nil, workflowSvc)

	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}

	expectRAGSourceJobRows(tenantMock, 77, "workflow_trigger:"+knowledgeBaseWorkflowID("ws-1", 77), nil, KnowledgeBaseSourceRecord{
		SourceID:     "source-file-1",
		ModelID:      77,
		RawVolumeID:  12,
		SourceType:   kbSourceTypeCatalogFile,
		SourceFileID: stringPtr("source-file"),
		KBFileID:     stringPtr("kb-file"),
		Status:       kbSourceStatusPending,
		Enabled:      boolPtr(true),
	})

	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)

	resp, err := svc.ListSourceJobs(ctx, ListSemanticModelSourceJobsParams{ModelID: 77})
	if err != nil {
		t.Fatalf("ListSourceJobs: %v", err)
	}
	if resp.Total != 1 || len(resp.Items) != 1 {
		t.Fatalf("source jobs response = %+v", resp)
	}
	job := resp.Items[0]
	if job.JobStatus != kbSourceJobSucceeded || !job.ReconcileRequired {
		t.Fatalf("job = %+v, want succeeded with reconcile_required=true", job)
	}
	if len(workflowSvc.listFileExecutionCalls) != 1 || workflowSvc.listFileExecutionCalls[0] != "kb-file" {
		t.Fatalf("ListFileExecutions calls = %+v", workflowSvc.listFileExecutionCalls)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceAppendModelSourcesDefersCatalogZipAndPreservesExistingWorkflow(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/ws-1/files/source-zip/download":
			t.Fatalf("unexpected DownloadWithMeta during append short request")
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			requireSemanticModelExecutionHeaders(t, r)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":          77,
				"name":        "kb_docs",
				"description": "docs",
				"tables":      []any{},
				"files":       map[string]any{"file_ids": []string{}},
			})
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			requireSemanticModelExecutionHeaders(t, r)
			var req struct {
				Files semanticModelFilesPayload `json:"files"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode semantic update: %v", err)
			}
			if len(req.Files.FileIDs) != 0 {
				t.Fatalf("semantic file_ids = %+v, want deferred", req.Files.FileIDs)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"updated": true})
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/volumes/"):
			t.Fatalf("unexpected raw-volume side effect during append short request: %s %s", r.Method, r.URL.String())
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	dataDomainSvc := &fakeSemanticModelDataDomainService{databaseID: 11}
	workflowSvc := &fakeSemanticModelWorkflowService{}
	workflowTemplateSvc := &fakeSemanticModelWorkflowTemplateService{}
	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()
	svc := newSemanticModelTestServiceWithKnowledgeBaseRuntimeDependencies(t, server.URL, systemClient, dataDomainSvc, nil, nil, workflowTemplateSvc, workflowSvc)

	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	tenantMock.ExpectQuery("SELECT model_id, catalog_id, database_id, raw_volume_id, processed_volume_id, ensure_status, last_ensure_error, last_checked_at").
		WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows([]string{
			"model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "ensure_status", "last_ensure_error", "last_checked_at",
		}).AddRow(int64(77), int64(3), int64(11), int64(12), int64(13), kbEnsureStatusReady, nil, int64(100)))
	expectUpsertKnowledgeBaseRawVolume(tenantMock, 77, kbRawKindDocument, false)
	expectAppendSourcesTransactionLock(tenantMock, 77)
	expectCatalogFileSourceLookupMiss(tenantMock, 77, "source-zip")
	tenantMock.ExpectExec("INSERT INTO knowledge_base_sources").
		WillReturnResult(sqlmock.NewResult(1, 1))
	expectKnowledgeBaseSourceJobRunUpsertMiss(tenantMock)
	tenantMock.ExpectExec("INSERT INTO knowledge_base_source_job_runs").
		WillReturnResult(sqlmock.NewResult(1, 1))
	expectKnowledgeBaseSourceJobRunUpsertMiss(tenantMock)
	tenantMock.ExpectExec("INSERT INTO knowledge_base_source_job_runs").
		WillReturnResult(sqlmock.NewResult(2, 1))
	sourceID := stableID("kb-source", int64(77), kbSourceTypeCatalogFile, "source-zip")
	tenantMock.ExpectCommit()
	tenantMock.ExpectQuery("SELECT .*source_id.*FROM knowledge_base_sources").
		WithArgs(int64(77)).
		WillReturnRows(knowledgeBaseSourceRecordRows(KnowledgeBaseSourceRecord{
			SourceID:          sourceID,
			ModelID:           77,
			CatalogID:         3,
			DatabaseID:        11,
			RawVolumeID:       12,
			ProcessedVolumeID: 13,
			SourceType:        kbSourceTypeCatalogFile,
			SourceFileID:      stringPtr("source-zip"),
			KBFileID:          stringPtr("source-zip"),
			DisplayName:       stringPtr("source-zip"),
			Status:            kbSourceStatusPending,
		}))
	tenantMock.ExpectQuery("SELECT .*job_id.*source_id.*model_id.*job_type.*job_status.*idempotency_key").
		WithArgs(int64(77)).
		WillReturnRows(knowledgeBaseSourceJobRunRows().
			AddRow(stableID("kb-job", sourceID, kbJobTypeCopy), sourceID, int64(77), kbJobTypeCopy, kbSourceJobPending, stableID("kb-job-key", sourceID, kbJobTypeCopy), nil, nil, nil, nil, false, "source-zip", "source-zip", nil, nil, int64(0), nil, nil, int64(100), int64(100)).
			AddRow(stableID("kb-job", sourceID, kbJobTypeRAGIngest), sourceID, int64(77), kbJobTypeRAGIngest, kbSourceJobPending, stableID("kb-job-key", sourceID, kbJobTypeRAGIngest), nil, nil, nil, nil, false, "source-zip", "source-zip", nil, nil, int64(0), nil, nil, int64(100), int64(100)))
	tenantMock.ExpectQuery("SELECT model_id, catalog_id, database_id, raw_volume_id, processed_volume_id, ensure_status, last_ensure_error, last_checked_at").
		WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows([]string{
			"model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "ensure_status", "last_ensure_error", "last_checked_at",
		}).AddRow(int64(77), int64(3), int64(11), int64(12), int64(13), kbEnsureStatusReady, nil, int64(100)))
	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithUserID(ctx, "user-1")
	ctx = withKnowledgeBaseCreatePrincipal(ctx)
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)

	resp, err := svc.AppendModelSources(ctx, AppendSemanticModelSourcesRequest{
		ModelID: 77,
		Sources: []CreateSemanticModelSourceRequest{{
			SourceType: kbSourceTypeCatalogFile,
			FileID:     "source-zip",
			VolumeID:   41,
		}},
	})
	if err != nil {
		t.Fatalf("AppendModelSources: %v", err)
	}
	if len(resp.Jobs) != 2 || resp.Jobs[0].JobStatus != kbSourceJobPending || resp.Jobs[1].JobStatus != kbSourceJobPending {
		t.Fatalf("jobs = %+v", resp.Jobs)
	}
	if len(dataDomainSvc.calls) != 0 {
		t.Fatalf("data domain calls = %+v, want none", dataDomainSvc.calls)
	}
	assertDocumentAppendPreservesExistingWorkflow(t, workflowTemplateSvc, workflowSvc, "ws-1", 77)
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceAppendModelSourcesFailsClosedWhenExistingWorkflowCannotBeRequired(t *testing.T) {
	tests := []struct {
		name       string
		requireErr error
	}{
		// Missing workflow is handled by ensure-when-not-found; these cases cover
		// present-but-broken / lookup failures that must stay fail-closed.
		{name: "workflow unavailable", requireErr: errors.New("existing knowledge base workflow is unavailable")},
		{name: "workflow lookup failure", requireErr: errors.New("get knowledge base workflow failed")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertPublicDocumentMutationFailsClosedAtWorkflowGuard(t, tt.requireErr, func(ctx context.Context, svc SemanticModelService) error {
				_, err := svc.AppendModelSources(ctx, AppendSemanticModelSourcesRequest{
					ModelID: 77,
					Sources: []CreateSemanticModelSourceRequest{{SourceType: kbSourceTypeCatalogFile, FileID: "source-file", VolumeID: 41}},
				})
				return err
			})
		})
	}
}

func TestSemanticModelServiceUpdateModelFilesRejectsMissingVolumeIDBeforeWorkflowGuard(t *testing.T) {
	// Legacy files.file_ids cannot carry volume_id; reject before workflow side effects.
	// Document workflow fail-closed coverage lives on AppendModelSources instead.
	// UpdateModel first GETs current model to detect tag-only updates; return a
	// different source set so the volume_id gate is exercised.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":          77,
				"name":        "kb_docs",
				"description": "docs",
				"tables":      []any{},
				"files":       map[string]any{"file_ids": []string{}},
			})
			return
		}
		t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
	}))
	defer server.Close()
	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()
	svc := newSemanticModelTestService(t, server.URL, systemClient)
	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	err = svc.UpdateModel(ctx, UpdateSemanticModelRequest{
		ModelID:     77,
		Name:        "kb_docs",
		Description: "updated docs",
		Tables:      json.RawMessage(`[]`),
		Files:       json.RawMessage(`{"file_ids":["source-file"],"vector_table":"existing_vector","embedding_model":"existing_embedding"}`),
	})
	if err == nil || !strings.Contains(err.Error(), "volume_id") {
		t.Fatalf("UpdateModel error = %v, want volume_id required", err)
	}
}

func TestSemanticModelServiceUpdateModelRejectsRename(t *testing.T) {
	const modelID = int64(77)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":          modelID,
				"name":        "kb_docs",
				"description": "docs",
				"tables":      []any{},
				"files":       map[string]any{"file_ids": []string{}},
			})
			return
		}
		t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()
	svc := newSemanticModelTestService(t, server.URL, systemClient)

	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")

	err = svc.UpdateModel(ctx, UpdateSemanticModelRequest{
		ModelID:     int(modelID),
		Name:        "kb_docs_renamed",
		Description: "renamed docs",
		Tables:      json.RawMessage(`[]`),
		Files:       json.RawMessage(`{"file_ids":[]}`),
	})
	if err == nil {
		t.Fatal("UpdateModel error = nil, want name immutable")
	}
	var serviceErr *ServiceError
	if !errors.As(err, &serviceErr) || serviceErr.Code != ErrCodeBadRequest {
		t.Fatalf("UpdateModel error = %v, want bad request name immutable", err)
	}
}

func TestSemanticModelServiceUpdateModelMetadataKeepsNameAndOmitsDisplayBinding(t *testing.T) {
	const modelID = int64(77)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":          modelID,
				"name":        "kb_docs",
				"description": "docs",
				"tables":      []any{},
				"files":       map[string]any{"file_ids": []string{}},
			})
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			requireSemanticModelExecutionHeaders(t, r)
			var req map[string]json.RawMessage
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode semantic update: %v", err)
			}
			if got := string(req["name"]); got != `"kb_docs"` {
				t.Fatalf("name = %s, want kb_docs", got)
			}
			if got := string(req["description"]); got != `"renamed docs"` {
				t.Fatalf("description = %s, want renamed docs", got)
			}
			if _, ok := req["knowledge_base_database_display_name"]; ok {
				t.Fatalf("database display binding must be omitted: %s", req["knowledge_base_database_display_name"])
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"updated": true})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()
	svc := newSemanticModelTestService(t, server.URL, systemClient)

	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")

	err = svc.UpdateModel(ctx, UpdateSemanticModelRequest{
		ModelID:     int(modelID),
		Name:        "kb_docs",
		Description: "renamed docs",
		Tables:      json.RawMessage(`[]`),
		Files:       json.RawMessage(`{"file_ids":[]}`),
	})
	if err != nil {
		t.Fatalf("UpdateModel: %v", err)
	}
}

func TestSemanticModelServiceUpdateModelMetadataPreservesSourcesWithoutDataDomain(t *testing.T) {
	const modelID = int64(77)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":          modelID,
				"name":        "kb_docs",
				"description": "docs",
				"tables":      []any{},
				"files":       map[string]any{"file_ids": []string{}},
			})
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			var req map[string]json.RawMessage
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode semantic update: %v", err)
			}
			if got := string(req["name"]); got != `"kb_docs"` {
				t.Fatalf("name = %s, want kb_docs", got)
			}
			if got := string(req["description"]); got != `"new docs"` {
				t.Fatalf("description = %s, want new docs", got)
			}
			if _, ok := req["tables"]; ok {
				t.Fatalf("tables must be omitted from metadata update: %s", req["tables"])
			}
			if _, ok := req["files"]; ok {
				t.Fatalf("files must be omitted from metadata update: %s", req["files"])
			}
			if _, ok := req["knowledge_base_database_display_name"]; ok {
				t.Fatalf("database display binding must be omitted without a data domain")
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"updated": true})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()
	svc := newSemanticModelTestService(t, server.URL, systemClient)

	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")

	err = svc.UpdateModel(ctx, UpdateSemanticModelRequest{
		ModelID:     int(modelID),
		Name:        "kb_docs",
		Description: "new docs",
	})
	if err != nil {
		t.Fatalf("UpdateModel: %v", err)
	}
}

func TestSemanticModelServiceUpdateModelMetadataPreservesSourcesWithUnreadyDataDomain(t *testing.T) {
	const modelID = int64(77)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":          modelID,
				"name":        "kb_docs",
				"description": "docs",
				"tables":      []any{},
				"files":       map[string]any{"file_ids": []string{}},
			})
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			var req map[string]json.RawMessage
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode semantic update: %v", err)
			}
			if got := string(req["name"]); got != `"kb_docs"` {
				t.Fatalf("name = %s, want kb_docs", got)
			}
			if got := string(req["description"]); got != `"new docs"` {
				t.Fatalf("description = %s, want new docs", got)
			}
			if _, ok := req["tables"]; ok {
				t.Fatalf("tables must be omitted from metadata update: %s", req["tables"])
			}
			if _, ok := req["files"]; ok {
				t.Fatalf("files must be omitted from metadata update: %s", req["files"])
			}
			if _, ok := req["knowledge_base_database_display_name"]; ok {
				t.Fatalf("database display binding must be omitted for an unready data domain")
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"updated": true})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()
	svc := newSemanticModelTestService(t, server.URL, systemClient)

	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")

	err = svc.UpdateModel(ctx, UpdateSemanticModelRequest{
		ModelID:     int(modelID),
		Name:        "kb_docs",
		Description: "new docs",
	})
	if err != nil {
		t.Fatalf("UpdateModel: %v", err)
	}
}

func TestSemanticModelServiceUpdateModelEmptyFileIDsPreservesUnreadyDataDomain(t *testing.T) {
	const modelID = int64(77)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":          modelID,
				"name":        "kb_docs",
				"description": "docs",
				"tables":      []any{},
				"files":       map[string]any{"file_ids": []string{}},
			})
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			requireSemanticModelExecutionHeaders(t, r)
			var req map[string]json.RawMessage
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode semantic update: %v", err)
			}
			if got := string(req["tables"]); got != `[]` {
				t.Fatalf("tables = %s, want []", got)
			}
			if got := string(req["files"]); got != `{"file_ids":[]}` {
				t.Fatalf("files = %s, want empty file_ids", got)
			}
			if _, ok := req["knowledge_base_database_display_name"]; ok {
				t.Fatalf("database display binding must be omitted for an unready data domain")
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"updated": true})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()
	svc := newSemanticModelTestService(t, server.URL, systemClient)

	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")

	err = svc.UpdateModel(ctx, UpdateSemanticModelRequest{
		ModelID:     int(modelID),
		Name:        "kb_docs",
		Description: "new docs",
		Tables:      json.RawMessage(`[]`),
		Files:       json.RawMessage(`{"file_ids":[]}`),
	})
	if err != nil {
		t.Fatalf("UpdateModel: %v", err)
	}
}

func TestSemanticModelServiceUpdateModelMetadataOmitsDatabaseDisplayBindingWithReadyDomain(t *testing.T) {
	const modelID = int64(77)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":          modelID,
				"name":        "kb_docs",
				"description": "docs",
				"tables":      []any{},
				"files":       map[string]any{"file_ids": []string{}},
			})
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			requireSemanticModelExecutionHeaders(t, r)
			var req map[string]json.RawMessage
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode semantic update: %v", err)
			}
			if got := string(req["name"]); got != `"kb_docs"` {
				t.Fatalf("name = %s, want kb_docs", got)
			}
			if got := string(req["description"]); got != `"new docs"` {
				t.Fatalf("description = %s, want new docs", got)
			}
			if _, ok := req["tables"]; ok {
				t.Fatalf("tables must be omitted from metadata update: %s", req["tables"])
			}
			if _, ok := req["files"]; ok {
				t.Fatalf("files must be omitted from metadata update: %s", req["files"])
			}
			if _, ok := req["knowledge_base_database_display_name"]; ok {
				t.Fatalf("database display binding must be omitted: %s", req["knowledge_base_database_display_name"])
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"updated": true})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()
	svc := newSemanticModelTestService(t, server.URL, systemClient)

	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")

	err = svc.UpdateModel(ctx, UpdateSemanticModelRequest{
		ModelID:     int(modelID),
		Name:        "kb_docs",
		Description: "new docs",
	})
	if err != nil {
		t.Fatalf("UpdateModel: %v", err)
	}
}

func TestSemanticModelServiceUpdateModelRejectsCatalogFileIDsWithoutVolumeID(t *testing.T) {
	// Legacy UpdateModel files.file_ids cannot supply volume_id; catalog_file
	// writes must go through sources / source_selections with volume_id.
	// A prior GET is required so tag-only updates can short-circuit; return a
	// different file set so this path still hits the volume_id rejection.
	modelID := int64(77)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":          modelID,
				"name":        "kb_docs",
				"description": "docs",
				"tables":      []any{},
				"files":       map[string]any{"file_ids": []string{}},
			})
			return
		}
		t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()
	svc := newSemanticModelTestService(t, server.URL, systemClient)

	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithUserID(ctx, "user-1")

	err = svc.UpdateModel(ctx, UpdateSemanticModelRequest{
		ModelID:     int(modelID),
		Name:        "kb_docs",
		Description: "new docs",
		Tables:      json.RawMessage(`[]`),
		Files:       json.RawMessage(`{"file_ids":["parsed-file"]}`),
	})
	if err == nil || !strings.Contains(err.Error(), "volume_id") {
		t.Fatalf("UpdateModel error = %v, want volume_id required", err)
	}
}

func TestSemanticModelServiceUpdateModelTagOnlyPreservesSourceVersions(t *testing.T) {
	const modelID = int64(77)
	const fileID = "kb-existing-file"
	var getCount int
	var updatedFiles semanticModelFilesPayload
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			getCount++
			requireSemanticModelExecutionHeaders(t, r)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":          modelID,
				"name":        "kb_docs",
				"description": "docs",
				"tables":      []any{},
				"files": map[string]any{
					"file_ids":        []string{fileID},
					"vector_table":    "kb_text_idx",
					"embedding_model": "embed-model",
					"tags":            []string{"old"},
				},
			})
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			requireSemanticModelExecutionHeaders(t, r)
			var req struct {
				Files semanticModelFilesPayload `json:"files"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode semantic update: %v", err)
			}
			updatedFiles = req.Files
			_ = json.NewEncoder(w).Encode(map[string]any{"updated": true})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()
	dataDomainSvc := &fakeSemanticModelDataDomainService{databaseID: 11}
	workflowTemplateSvc := &fakeSemanticModelWorkflowTemplateService{}
	workflowSvc := &fakeSemanticModelWorkflowService{}
	svc := newSemanticModelTestServiceWithKnowledgeBaseRuntimeDependencies(t, server.URL, systemClient, dataDomainSvc, nil, nil, workflowTemplateSvc, workflowSvc)

	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithUserID(ctx, "user-1")
	err = svc.UpdateModel(ctx, UpdateSemanticModelRequest{
		ModelID:     int(modelID),
		Name:        "kb_docs",
		Description: "docs",
		Tables:      json.RawMessage(`[]`),
		Files:       json.RawMessage(`{"file_ids":["kb-existing-file"],"vector_table":"kb_text_idx","embedding_model":"embed-model","tags":["new"]}`),
	})
	if err != nil {
		t.Fatalf("UpdateModel: %v", err)
	}
	if getCount != 1 {
		t.Fatalf("semantic get count = %d, want one source comparison", getCount)
	}
	if len(updatedFiles.FileIDs) != 1 || updatedFiles.FileIDs[0] != fileID || len(updatedFiles.Tags) != 1 || updatedFiles.Tags[0] != "new" {
		t.Fatalf("updated files = %+v", updatedFiles)
	}
	if len(dataDomainSvc.calls) != 0 {
		t.Fatalf("data domain service calls = %+v, want none", dataDomainSvc.calls)
	}
}

func TestSemanticModelServiceAppendModelSourcesCreatesImportTaskForStructuredLocalFile(t *testing.T) {
	var semanticGetCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			semanticGetCount++
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":          77,
				"name":        "kb_docs",
				"description": "docs",
				"tables":      []any{},
				"files":       map[string]any{"file_ids": []string{}},
			})
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			var req struct {
				Tables []semanticModelTableSource `json:"tables"`
				Files  semanticModelFilesPayload  `json:"files"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode semantic update: %v", err)
			}
			if len(req.Tables) != 0 || len(req.Files.FileIDs) != 0 {
				t.Fatalf("semantic update = tables %+v files %+v, want deferred structured import", req.Tables, req.Files.FileIDs)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"updated": true})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	dataDomainSvc := &fakeSemanticModelDataDomainService{databaseID: 11, volumeIDs: []int64{14}}
	localImportSvc := &queuedSemanticModelLocalFileImportService{results: []KnowledgeBaseLocalFileImportResult{{
		TaskID:  "task-structured-append",
		FileIDs: []string{"conn-structured-file"},
	}}}
	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()
	workflowTemplateSvc := &fakeSemanticModelWorkflowTemplateService{}
	workflowSvc := &fakeSemanticModelWorkflowService{}
	svc := newSemanticModelTestServiceWithKnowledgeBaseRuntimeDependencies(t, server.URL, systemClient, dataDomainSvc, nil, localImportSvc, workflowTemplateSvc, workflowSvc)

	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	tenantMock.ExpectQuery("SELECT model_id, catalog_id, database_id, raw_volume_id, processed_volume_id, ensure_status, last_ensure_error, last_checked_at").
		WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows([]string{
			"model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "ensure_status", "last_ensure_error", "last_checked_at",
		}).AddRow(int64(77), int64(3), int64(11), int64(12), int64(13), kbEnsureStatusReady, nil, int64(100)))
	expectUpsertKnowledgeBaseRawVolume(tenantMock, 77, kbRawKindDocument, false)
	expectAppendSourcesTransactionLock(tenantMock, 77)
	sourceID := stableID("kb-source", int64(77), kbSourceTypeLocalFile, "kb-structured-file")
	jobID := stableID("kb-job", sourceID, kbJobTypeLoad)
	tenantMock.ExpectQuery("WHERE kbs.model_id = \\? AND kbs.source_id = \\?").
		WithArgs(int64(77), sourceID).
		WillReturnRows(knowledgeBaseSourceRecordRows())
	tenantMock.ExpectExec("INSERT INTO knowledge_base_sources").
		WillReturnResult(sqlmock.NewResult(1, 1))
	expectKnowledgeBaseSourceJobRunUpsertMiss(tenantMock)
	tenantMock.ExpectExec("INSERT INTO knowledge_base_source_job_runs").
		WillReturnResult(sqlmock.NewResult(1, 1))
	tenantMock.ExpectCommit()
	tenantMock.ExpectExec("UPDATE knowledge_base_source_job_runs").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), kbSourceJobRunning, exactShortOperationIDArg{value: "import_task:task-structured-append"}, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), "user-1", jobID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	tenantMock.ExpectQuery("SELECT .*source_id.*FROM knowledge_base_sources").
		WithArgs(int64(77)).
		WillReturnRows(knowledgeBaseSourceRecordRows(KnowledgeBaseSourceRecord{
			SourceID:          sourceID,
			ModelID:           77,
			CatalogID:         3,
			DatabaseID:        11,
			RawVolumeID:       12,
			ProcessedVolumeID: 13,
			SourceType:        kbSourceTypeCatalogTable,
			DisplayName:       stringPtr("structured_table"),
			Status:            kbSourceStatusPending,
		}))
	tenantMock.ExpectQuery("SELECT .*job_id.*source_id.*model_id.*job_type.*job_status.*idempotency_key").
		WithArgs(int64(77)).
		WillReturnRows(knowledgeBaseSourceJobRunRows().
			AddRow(jobID, sourceID, int64(77), kbJobTypeLoad, kbSourceJobQueued, stableID("kb-job-key", sourceID, kbJobTypeLoad), nil, nil, nil, nil, false, nil, nil, nil, nil, int64(0), nil, nil, int64(100), int64(100)))
	tenantMock.ExpectQuery("SELECT model_id, catalog_id, database_id, raw_volume_id, processed_volume_id, ensure_status, last_ensure_error, last_checked_at").
		WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows([]string{
			"model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "ensure_status", "last_ensure_error", "last_checked_at",
		}).AddRow(int64(77), int64(3), int64(11), int64(12), int64(13), kbEnsureStatusReady, nil, int64(100)))
	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithUserID(ctx, "user-1")
	ctx = withKnowledgeBaseCreatePrincipal(ctx)
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)

	structuredTableConfig := `{"database_id":11,"conn_file_ids":["conn-structured-file"],"new_table":true,"create_table":{"name":"structured_table","description":"long description to keep this payload above operation id size","tableColumn":[{"column":"col","dataType":"VARCHAR","col_num_in_file":1}]}}`
	if len(structuredTableConfig) <= 128 {
		t.Fatalf("structured table config length = %d, want > 128", len(structuredTableConfig))
	}
	resp, err := svc.AppendModelSources(ctx, AppendSemanticModelSourcesRequest{
		ModelID: 77,
		Sources: []CreateSemanticModelSourceRequest{{
			SourceType:  kbSourceTypeLocalFile,
			FileName:    "structured.csv",
			FileID:      "kb-structured-file",
			UploadKind:  kbLocalUploadKindStructured,
			TableConfig: structuredTableConfig,
		}},
	})
	if err != nil {
		t.Fatalf("AppendModelSources: %v", err)
	}
	if len(resp.Jobs) != 1 || resp.Jobs[0].JobID != jobID || resp.Jobs[0].JobStatus != kbSourceJobRunning || resp.Jobs[0].OperationID == nil || *resp.Jobs[0].OperationID != "import_task:task-structured-append" || len(*resp.Jobs[0].OperationID) > 128 {
		t.Fatalf("jobs = %+v", resp.Jobs)
	}
	if len(resp.Sources) != 1 || resp.Sources[0].SourceType != SemanticModelSourceTypeTable || resp.Sources[0].SourceFileID != nil || resp.Sources[0].KBFileID != nil {
		t.Fatalf("structured sources = %+v", resp.Sources)
	}
	if resp.Jobs[0].SourceFileID != nil || resp.Jobs[0].KBFileID != nil {
		t.Fatalf("structured job exposes file relation = %+v", resp.Jobs[0])
	}
	if len(localImportSvc.calls) != 1 {
		t.Fatalf("local import calls = %+v", localImportSvc.calls)
	}
	if localImportSvc.calls[0].VolumeID != 12 || localImportSvc.calls[0].UploadKind != kbLocalUploadKindStructured || localImportSvc.calls[0].FileName != "structured.csv" || localImportSvc.calls[0].FileID != "kb-structured-file" {
		t.Fatalf("local import call = %+v", localImportSvc.calls[0])
	}
	if len(localImportSvc.calls[0].TableConfig) <= 128 || !strings.Contains(localImportSvc.calls[0].TableConfig, "conn-structured-file") || !strings.Contains(localImportSvc.calls[0].TableConfig, "structured_table") {
		t.Fatalf("local import table_config = %q", localImportSvc.calls[0].TableConfig)
	}
	if semanticGetCount != 2 {
		t.Fatalf("semantic get count = %d, want 2", semanticGetCount)
	}
	assertStructuredOnlyAppendDoesNotTouchDocumentWorkflow(t, workflowTemplateSvc, workflowSvc)
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceAppendModelSourcesEnsuresMissingDataDomainBeforeWritingSources(t *testing.T) {
	var sawInitialUpdate bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			requireSemanticModelExecutionHeaders(t, r)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":          77,
				"name":        "kb_docs",
				"description": "docs",
				"tables":      []any{},
				"files": map[string]any{
					"file_ids":                     []string{},
					"vector_table":                 "client_text_idx",
					"embedding_model":              "BAAI/bge-large-zh-v1.5",
					"image_vector_table":           "client_image_idx",
					"image_embedding_model":        "efficientnet-b3",
					"image_embedding_backend_id":   "-30010",
					"image_embedding_dimension":    1536,
					"image_preprocess_version":     "efficientnet-b3-v1-rgb-300-letterbox-imagenet",
					"image_distance_metric":        "cosine",
					"active_image_index_config_id": "image-index-config-1",
				},
			})
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			requireSemanticModelExecutionHeaders(t, r)
			var req struct {
				Name        string                     `json:"name"`
				Description string                     `json:"description"`
				Tables      []semanticModelTableSource `json:"tables"`
				Files       semanticModelFilesPayload  `json:"files"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode semantic update: %v", err)
			}
			if req.Name != "kb_docs" || req.Description != "docs" {
				t.Fatalf("semantic update identity = %+v", req)
			}
			if len(req.Files.FileIDs) != 0 {
				t.Fatalf("semantic files update = %+v", req.Files.FileIDs)
			}
			if req.Files.VectorTable != "client_text_idx" || req.Files.EmbeddingModel != "BAAI/bge-large-zh-v1.5" {
				t.Fatalf("semantic text vector binding = %+v", req.Files)
			}
			if req.Files.ImageVectorTable != "client_image_idx" || req.Files.ImageEmbeddingModel != "efficientnet-b3" || req.Files.ImageEmbeddingBackendID != "-30010" || req.Files.ImageEmbeddingDimension != 1536 || req.Files.ImagePreprocessVersion != "efficientnet-b3-v1-rgb-300-letterbox-imagenet" || req.Files.ImageDistanceMetric != "cosine" {
				t.Fatalf("semantic image vector binding = %+v", req.Files)
			}
			if len(req.Tables) != 0 {
				t.Fatalf("semantic tables update = %+v", req.Tables)
			}
			sawInitialUpdate = true
			_ = json.NewEncoder(w).Encode(map[string]any{"updated": true})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/ws-1/files/source-file/download":
			requireSemanticModelExecutionHeaders(t, r)
			w.Header().Set("Content-Disposition", `attachment; filename="source.txt"`)
			_, _ = w.Write([]byte("source content"))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/workspaces/ws-1/volumes/12/files":
			t.Fatalf("unexpected AddFiles during append short request")
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/workspaces/ws-1/volumes/12/files/trigger":
			t.Fatalf("unexpected TriggerFiles during append short request")
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	dataDomainSvc := &fakeSemanticModelDataDomainService{databaseID: 11, volumeIDs: []int64{12, 13}}
	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()
	workflowSvc := &fakeSemanticModelWorkflowService{}
	workflowTemplateSvc := &fakeSemanticModelWorkflowTemplateService{}
	svc := newSemanticModelTestServiceWithKnowledgeBaseRuntimeDependencies(t, server.URL, systemClient, dataDomainSvc, nil, nil, workflowTemplateSvc, workflowSvc)

	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	tenantMock.ExpectQuery("SELECT model_id, catalog_id, database_id, raw_volume_id, processed_volume_id, ensure_status, last_ensure_error, last_checked_at").
		WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows([]string{
			"model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "ensure_status", "last_ensure_error", "last_checked_at",
		}))
	tenantMock.ExpectExec("INSERT INTO knowledge_base_data_domains").
		WillReturnResult(sqlmock.NewResult(1, 1))
	expectKnowledgeBaseDataDomainClaimAndReady(tenantMock, 77, 3, 11, 12, 13)
	expectUpsertKnowledgeBaseRawVolume(tenantMock, 77, kbRawKindDocument, false)
	expectAppendSourcesTransactionLock(tenantMock, 77)
	expectCatalogFileSourceLookupMiss(tenantMock, 77, "source-file")
	tenantMock.ExpectExec("INSERT INTO knowledge_base_sources").
		WillReturnResult(sqlmock.NewResult(1, 1))
	expectKnowledgeBaseSourceJobRunUpsertMiss(tenantMock)
	tenantMock.ExpectExec("INSERT INTO knowledge_base_source_job_runs").
		WillReturnResult(sqlmock.NewResult(1, 1))
	expectKnowledgeBaseSourceJobRunUpsertMiss(tenantMock)
	tenantMock.ExpectExec("INSERT INTO knowledge_base_source_job_runs").
		WillReturnResult(sqlmock.NewResult(2, 1))
	tenantMock.ExpectCommit()
	tenantMock.ExpectQuery("SELECT .*source_id.*FROM knowledge_base_sources").
		WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows([]string{
			"source_id", "model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "source_type",
			"source_file_id", "source_table_id", "kb_file_id", "kb_table_id", "display_name", "source_path",
			"db_name", "table_name", "status", "error", "enabled", "expires_at", "tags", "force_enabled_after_expiry", "segment_version_id", "index_version",
		}).AddRow(stableID("kb-source", int64(77), kbSourceTypeCatalogFile, "source-file"), int64(77), int64(3), int64(11), int64(12), int64(13), kbSourceTypeCatalogFile, "source-file", nil, "source-file", nil, "source-file", nil, nil, nil, kbSourceStatusPending, nil, nil, nil, nil, false, nil, nil))
	tenantMock.ExpectQuery("SELECT .*job_id.*source_id.*model_id.*job_type.*job_status.*idempotency_key").
		WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows([]string{
			"job_id", "source_id", "model_id", "job_type", "job_status", "idempotency_key", "operation_id",
			"workflow_execution_id", "runtime_actor_moi_user_id", "runtime_effective_role_id", "runtime_is_workspace_owner", "source_file_id", "kb_file_id", "source_table_id", "kb_table_id",
			"retry_count", "next_retry_at", "error", "created_at", "updated_at",
		}).
			AddRow(stableID("kb-job", stableID("kb-source", int64(77), kbSourceTypeCatalogFile, "source-file"), kbJobTypeCopy), stableID("kb-source", int64(77), kbSourceTypeCatalogFile, "source-file"), int64(77), kbJobTypeCopy, kbSourceJobPending, stableID("kb-job-key", stableID("kb-source", int64(77), kbSourceTypeCatalogFile, "source-file"), kbJobTypeCopy), nil, nil, nil, nil, false, "source-file", "source-file", nil, nil, int64(0), nil, nil, int64(100), int64(100)).
			AddRow(stableID("kb-job", stableID("kb-source", int64(77), kbSourceTypeCatalogFile, "source-file"), kbJobTypeRAGIngest), stableID("kb-source", int64(77), kbSourceTypeCatalogFile, "source-file"), int64(77), kbJobTypeRAGIngest, kbSourceJobPending, stableID("kb-job-key", stableID("kb-source", int64(77), kbSourceTypeCatalogFile, "source-file"), kbJobTypeRAGIngest), nil, nil, nil, nil, false, "source-file", "source-file", nil, nil, int64(0), nil, nil, int64(100), int64(100)))
	tenantMock.ExpectQuery("SELECT model_id, catalog_id, database_id, raw_volume_id, processed_volume_id, ensure_status, last_ensure_error, last_checked_at").
		WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows([]string{
			"model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "ensure_status", "last_ensure_error", "last_checked_at",
		}).AddRow(int64(77), int64(3), int64(11), int64(12), int64(13), kbEnsureStatusReady, nil, int64(100)))
	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithUserID(ctx, "user-1")
	ctx = withKnowledgeBaseCreatePrincipal(ctx)
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)

	resp, err := svc.AppendModelSources(ctx, AppendSemanticModelSourcesRequest{
		ModelID: 77,
		Sources: []CreateSemanticModelSourceRequest{{
			SourceType: kbSourceTypeCatalogFile,
			FileID:     "source-file",
			VolumeID:   41,
		}},
	})
	if err != nil {
		t.Fatalf("AppendModelSources: %v", err)
	}
	if resp.DataDomain == nil || resp.DataDomain.DatabaseID != 11 || resp.DataDomain.RawVolumeID != 12 || resp.DataDomain.ProcessedVolumeID != 13 {
		t.Fatalf("data domain = %+v", resp.DataDomain)
	}
	if len(resp.Sources) != 1 || resp.Sources[0].ResourceID != "source-file" {
		t.Fatalf("sources = %+v", resp.Sources)
	}
	if len(resp.Jobs) != 2 || resp.Jobs[0].JobStatus != kbSourceJobPending || resp.Jobs[1].JobStatus != kbSourceJobPending {
		t.Fatalf("jobs = %+v", resp.Jobs)
	}
	if !sawInitialUpdate {
		t.Fatalf("semantic model update was not called")
	}
	if len(dataDomainSvc.calls) < 4 || dataDomainSvc.calls[0] != "resolve_default_catalog" || !strings.HasPrefix(dataDomainSvc.calls[1], "database:") {
		t.Fatalf("data domain service calls = %+v", dataDomainSvc.calls)
	}
	assertDocumentAppendPreservesExistingWorkflow(t, workflowTemplateSvc, workflowSvc, "ws-1", 77)
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceAppendModelSourcesReusesCatalogFileVectorsForPlainKnowledgeBase(t *testing.T) {
	modelID := int64(77)
	sourceFileID := "source-file"
	sourceID := stableID("kb-source", modelID, kbSourceTypeCatalogFile, sourceFileID)
	semanticGetCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/ws-1/files/source-file/download":
			t.Fatalf("unexpected DownloadWithMeta during append short request")
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			semanticGetCount++
			if semanticGetCount > 2 {
				t.Fatalf("unexpected stale semantic model GET during append vector reuse")
			}
			requireSemanticModelExecutionHeaders(t, r)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":          modelID,
				"name":        "plain_kb",
				"description": "docs",
				"tables":      []any{},
				"files":       map[string]any{"file_ids": []string{}},
			})
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			requireSemanticModelExecutionHeaders(t, r)
			var req struct {
				Name        string                     `json:"name"`
				Description string                     `json:"description"`
				Tables      []semanticModelTableSource `json:"tables"`
				Files       semanticModelFilesPayload  `json:"files"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode semantic update: %v", err)
			}
			if req.Name != "plain_kb" || req.Description != "docs" {
				t.Fatalf("semantic identity = %+v", req)
			}
			if len(req.Tables) != 0 {
				t.Fatalf("semantic tables update = %+v", req.Tables)
			}
			if len(req.Files.FileIDs) != 0 {
				t.Fatalf("semantic file_ids = %+v", req.Files.FileIDs)
			}
			if req.Files.VectorTable != defaultKnowledgeBaseVectorTable(modelID) || req.Files.EmbeddingModel != kbDefaultEmbeddingModel {
				t.Fatalf("semantic vector binding = %+v", req.Files)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"updated": true})
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/volumes/"):
			t.Fatalf("unexpected raw-volume side effect after vector reuse: %s %s", r.Method, r.URL.String())
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	dataDomainSvc := &fakeSemanticModelDataDomainService{databaseID: 11, volumeIDs: []int64{12, 13}}
	workflowTemplateSvc := &fakeSemanticModelWorkflowTemplateService{}
	workflowSvc := &fakeSemanticModelWorkflowService{}
	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()
	svc := newSemanticModelTestServiceWithKnowledgeBaseRuntimeDependencies(t, server.URL, systemClient, dataDomainSvc, nil, nil, workflowTemplateSvc, workflowSvc)

	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	tenantMock.ExpectQuery("SELECT model_id, catalog_id, database_id, raw_volume_id, processed_volume_id, ensure_status, last_ensure_error, last_checked_at").
		WithArgs(modelID).
		WillReturnRows(sqlmock.NewRows([]string{
			"model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "ensure_status", "last_ensure_error", "last_checked_at",
		}))
	tenantMock.ExpectExec("INSERT INTO knowledge_base_data_domains").
		WillReturnResult(sqlmock.NewResult(1, 1))
	expectKnowledgeBaseDataDomainClaimAndReady(tenantMock, modelID, 3, 11, 12, 13)
	expectUpsertKnowledgeBaseRawVolume(tenantMock, modelID, kbRawKindDocument, false)
	expectAppendSourcesTransactionLock(tenantMock, modelID)
	expectCatalogFileSourceLookupMiss(tenantMock, modelID, sourceFileID)
	tenantMock.ExpectExec("INSERT INTO knowledge_base_sources").
		WillReturnResult(sqlmock.NewResult(1, 1))
	expectKnowledgeBaseSourceJobRunUpsertMiss(tenantMock)
	tenantMock.ExpectExec("INSERT INTO knowledge_base_source_job_runs").
		WillReturnResult(sqlmock.NewResult(1, 1))
	expectKnowledgeBaseSourceJobRunUpsertMiss(tenantMock)
	tenantMock.ExpectExec("INSERT INTO knowledge_base_source_job_runs").
		WillReturnResult(sqlmock.NewResult(2, 1))
	tenantMock.ExpectCommit()
	tenantMock.ExpectQuery("SELECT .*source_id.*FROM knowledge_base_sources kbs .*WHERE kbs\\.model_id = \\? AND kbs\\.status <> 'removed' ORDER BY").
		WithArgs(modelID).
		WillReturnRows(knowledgeBaseSourceRecordRows(KnowledgeBaseSourceRecord{
			SourceID:          sourceID,
			ModelID:           modelID,
			CatalogID:         3,
			DatabaseID:        11,
			RawVolumeID:       12,
			ProcessedVolumeID: 13,
			SourceType:        kbSourceTypeCatalogFile,
			SourceFileID:      stringPtr(sourceFileID),
			KBFileID:          stringPtr(sourceFileID),
			DisplayName:       stringPtr(sourceFileID),
			Status:            kbSourceStatusPending,
		}))
	tenantMock.ExpectQuery("SELECT .*job_id.*source_id.*model_id.*job_type.*job_status.*idempotency_key").
		WithArgs(modelID).
		WillReturnRows(knowledgeBaseSourceJobRunRows().
			AddRow(stableID("kb-job", sourceID, kbJobTypeCopy), sourceID, modelID, kbJobTypeCopy, kbSourceJobPending, stableID("kb-job-key", sourceID, kbJobTypeCopy), nil, nil, nil, nil, false, sourceFileID, sourceFileID, nil, nil, int64(0), nil, nil, int64(100), int64(100)).
			AddRow(stableID("kb-job", sourceID, kbJobTypeRAGIngest), sourceID, modelID, kbJobTypeRAGIngest, kbSourceJobPending, stableID("kb-job-key", sourceID, kbJobTypeRAGIngest), nil, nil, nil, nil, false, sourceFileID, sourceFileID, nil, nil, int64(0), nil, nil, int64(100), int64(100)))
	tenantMock.ExpectQuery("SELECT model_id, catalog_id, database_id, raw_volume_id, processed_volume_id, ensure_status, last_ensure_error, last_checked_at").
		WithArgs(modelID).
		WillReturnRows(sqlmock.NewRows([]string{
			"model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "ensure_status", "last_ensure_error", "last_checked_at",
		}).AddRow(modelID, int64(3), int64(11), int64(12), int64(13), kbEnsureStatusReady, nil, int64(100)))
	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithUserID(ctx, "user-1")
	ctx = withKnowledgeBaseCreatePrincipal(ctx)
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)

	resp, err := svc.AppendModelSources(ctx, AppendSemanticModelSourcesRequest{
		ModelID: int(modelID),
		Sources: []CreateSemanticModelSourceRequest{{
			SourceType: kbSourceTypeCatalogFile,
			FileID:     sourceFileID,
			VolumeID:   41,
		}},
	})
	if err != nil {
		t.Fatalf("AppendModelSources: %v", err)
	}
	if resp.DataDomain == nil || resp.DataDomain.EnsureStatus != kbEnsureStatusReady || resp.DataDomain.RawVolumeID != 12 {
		t.Fatalf("data domain = %+v", resp.DataDomain)
	}
	if len(resp.Sources) != 1 || resp.Sources[0].ResourceID != sourceFileID || resp.Sources[0].IngestStatus == nil || *resp.Sources[0].IngestStatus != kbSourceStatusPending {
		t.Fatalf("sources = %+v", resp.Sources)
	}
	if len(resp.Jobs) != 2 || resp.Jobs[0].JobStatus != kbSourceJobPending || resp.Jobs[1].JobStatus != kbSourceJobPending || resp.Jobs[1].OperationID != nil {
		t.Fatalf("jobs = %+v, want deferred catalog file jobs", resp.Jobs)
	}
	if semanticGetCount != 2 {
		t.Fatalf("semantic GET count = %d, want initial + locked reads only", semanticGetCount)
	}
	assertDocumentAppendPreservesExistingWorkflow(t, workflowTemplateSvc, workflowSvc, "ws-1", modelID)
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceAppendModelSourcesReactivatesRemovedCatalogFileSourceWithExistingTargetVectorRow(t *testing.T) {
	var sawSemanticUpdate bool
	sourceID := stableID("kb-source", int64(77), kbSourceTypeCatalogFile, "source-file")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			requireSemanticModelExecutionHeaders(t, r)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":          77,
				"name":        "kb_docs",
				"description": "docs",
				"tables":      []any{},
				"files": map[string]any{
					"file_ids":        []string{},
					"vector_table":    "client_text_idx",
					"embedding_model": "BAAI/bge-large-zh-v1.5",
				},
			})
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			requireSemanticModelExecutionHeaders(t, r)
			var req struct {
				Files semanticModelFilesPayload `json:"files"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode semantic update: %v", err)
			}
			if len(req.Files.FileIDs) != 0 {
				t.Fatalf("semantic files update = %+v", req.Files.FileIDs)
			}
			if req.Files.VectorTable != "client_text_idx" || req.Files.EmbeddingModel != "BAAI/bge-large-zh-v1.5" {
				t.Fatalf("semantic vector binding = %+v", req.Files)
			}
			sawSemanticUpdate = true
			_ = json.NewEncoder(w).Encode(map[string]any{"updated": true})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/ws-1/files/source-file/download":
			requireSemanticModelExecutionHeaders(t, r)
			w.Header().Set("Content-Disposition", `attachment; filename="source.txt"`)
			_, _ = w.Write([]byte("source content"))
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/volumes/"):
			t.Fatalf("unexpected raw-volume side effect after reactivated vector reuse: %s %s", r.Method, r.URL.String())
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()
	dataDomainSvc := &fakeSemanticModelDataDomainService{databaseID: 11, volumeIDs: []int64{12, 13}}
	workflowTemplateSvc := &fakeSemanticModelWorkflowTemplateService{}
	workflowSvc := &fakeSemanticModelWorkflowService{}
	svc := newSemanticModelTestServiceWithKnowledgeBaseRuntimeDependencies(t, server.URL, systemClient, dataDomainSvc, nil, nil, workflowTemplateSvc, workflowSvc)
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}

	tenantMock.ExpectQuery("SELECT model_id, catalog_id, database_id, raw_volume_id, processed_volume_id, ensure_status, last_ensure_error, last_checked_at").
		WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows([]string{
			"model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "ensure_status", "last_ensure_error", "last_checked_at",
		}).AddRow(int64(77), int64(3), int64(11), int64(12), int64(13), kbEnsureStatusReady, nil, int64(100)))
	expectUpsertKnowledgeBaseRawVolume(tenantMock, 77, kbRawKindDocument, false)
	expectAppendSourcesTransactionLock(tenantMock, 77)
	expectCatalogFileSourceLookupHit(tenantMock, 77, "source-file", KnowledgeBaseSourceRecord{
		SourceID:          sourceID,
		ModelID:           77,
		CatalogID:         3,
		DatabaseID:        11,
		RawVolumeID:       12,
		ProcessedVolumeID: 13,
		SourceType:        kbSourceTypeCatalogFile,
		SourceFileID:      stringPtr("source-file"),
		KBFileID:          stringPtr("source-file"),
		DisplayName:       stringPtr("source.txt"),
		Status:            kbSourceStatusRemoved,
	})
	// Reactivate re-resolves catalog location; raw_volume_id becomes source volume 41.
	expectCatalogFileSourceOriginLookupMissWithMeta(tenantMock, "source-file", 41, "source-file.pdf")
	tenantMock.ExpectExec("UPDATE knowledge_base_sources").
		WithArgs(
			int64(3), int64(11), int64(41), int64(13), kbSourceTypeCatalogFile, "source-file", nil, "source-file", nil, "source-file.pdf", nil, nil, nil,
			kbSourceStatusPending, true, "user-1", int64(77), sourceID, kbSourceStatusRemoved,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))
	expectKnowledgeBaseSourceJobRunUpsertMiss(tenantMock)
	tenantMock.ExpectExec("INSERT INTO knowledge_base_source_job_runs").
		WillReturnResult(sqlmock.NewResult(1, 1))
	expectKnowledgeBaseSourceJobRunUpsertMiss(tenantMock)
	tenantMock.ExpectExec("INSERT INTO knowledge_base_source_job_runs").
		WillReturnResult(sqlmock.NewResult(2, 1))
	tenantMock.ExpectCommit()
	tenantMock.ExpectQuery("SELECT .*source_id.*FROM knowledge_base_sources kbs .*WHERE kbs\\.model_id = \\? AND kbs\\.status <> 'removed' ORDER BY").
		WithArgs(int64(77)).
		WillReturnRows(knowledgeBaseSourceRecordRows(KnowledgeBaseSourceRecord{
			SourceID:          sourceID,
			ModelID:           77,
			CatalogID:         3,
			DatabaseID:        11,
			RawVolumeID:       12,
			ProcessedVolumeID: 13,
			SourceType:        kbSourceTypeCatalogFile,
			SourceFileID:      stringPtr("source-file"),
			KBFileID:          stringPtr("source-file"),
			DisplayName:       stringPtr("source-file"),
			Status:            kbSourceStatusPending,
		}))
	tenantMock.ExpectQuery("SELECT .*job_id.*source_id.*model_id.*job_type.*job_status.*idempotency_key").
		WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows([]string{
			"job_id", "source_id", "model_id", "job_type", "job_status", "idempotency_key", "operation_id",
			"workflow_execution_id", "runtime_actor_moi_user_id", "runtime_effective_role_id", "runtime_is_workspace_owner", "source_file_id", "kb_file_id", "source_table_id", "kb_table_id",
			"retry_count", "next_retry_at", "error", "created_at", "updated_at",
		}).
			AddRow(stableID("kb-job", sourceID, kbJobTypeCopy), sourceID, int64(77), kbJobTypeCopy, kbSourceJobPending, stableID("kb-job-key", sourceID, kbJobTypeCopy), nil, nil, nil, nil, false, "source-file", "source-file", nil, nil, int64(0), nil, nil, int64(100), int64(100)).
			AddRow(stableID("kb-job", sourceID, kbJobTypeRAGIngest), sourceID, int64(77), kbJobTypeRAGIngest, kbSourceJobPending, stableID("kb-job-key", sourceID, kbJobTypeRAGIngest), nil, nil, nil, nil, false, "source-file", "source-file", nil, nil, int64(0), nil, nil, int64(100), int64(100)))
	tenantMock.ExpectQuery("SELECT model_id, catalog_id, database_id, raw_volume_id, processed_volume_id, ensure_status, last_ensure_error, last_checked_at").
		WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows([]string{
			"model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "ensure_status", "last_ensure_error", "last_checked_at",
		}).AddRow(int64(77), int64(3), int64(11), int64(12), int64(13), kbEnsureStatusReady, nil, int64(100)))
	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithUserID(ctx, "user-1")
	ctx = withKnowledgeBaseCreatePrincipal(ctx)
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)

	resp, err := svc.AppendModelSources(ctx, AppendSemanticModelSourcesRequest{
		ModelID: 77,
		Sources: []CreateSemanticModelSourceRequest{{
			SourceType: kbSourceTypeCatalogFile,
			FileID:     "source-file",
			VolumeID:   41,
		}},
	})
	if err != nil {
		t.Fatalf("AppendModelSources: %v", err)
	}
	if !sawSemanticUpdate {
		t.Fatal("semantic model scope update was not called")
	}
	if len(resp.Sources) != 1 || resp.Sources[0].SourceID != sourceID || resp.Sources[0].ResourceID != "source-file" || resp.Sources[0].IngestStatus == nil || *resp.Sources[0].IngestStatus != kbSourceStatusPending {
		t.Fatalf("sources = %+v", resp.Sources)
	}
	if len(resp.Jobs) != 2 || resp.Jobs[0].JobType != kbJobTypeCopy || resp.Jobs[1].JobType != kbJobTypeRAGIngest || resp.Jobs[0].JobStatus != kbSourceJobPending || resp.Jobs[1].OperationID != nil {
		t.Fatalf("jobs = %+v, want deferred jobs", resp.Jobs)
	}
	assertDocumentAppendPreservesExistingWorkflow(t, workflowTemplateSvc, workflowSvc, "ws-1", 77)
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceAppendModelSourcesCleansCreatedRowsWhenSemanticUpdateFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			requireSemanticModelExecutionHeaders(t, r)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":          77,
				"name":        "kb_docs",
				"description": "docs",
				"tables":      []any{},
				"files":       map[string]any{"file_ids": []string{}},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/ws-1/files/source-file/download":
			requireSemanticModelExecutionHeaders(t, r)
			w.Header().Set("Content-Disposition", `attachment; filename="source.txt"`)
			_, _ = w.Write([]byte("source content"))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/workspaces/ws-1/files":
			requireSemanticModelExecutionHeaders(t, r)
			_ = json.NewEncoder(w).Encode(map[string]any{"file_id": "source-file"})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/workspaces/ws-1/volumes/12/files":
			requireSemanticModelExecutionHeaders(t, r)
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/workspaces/ws-1/volumes/12/files/trigger":
			requireSemanticModelExecutionHeaders(t, r)
			_ = json.NewEncoder(w).Encode(map[string]any{"triggered": 1})
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			requireSemanticModelExecutionHeaders(t, r)
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]any{"code": 13, "message": "semantic update failed"})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	dataDomainSvc := &fakeSemanticModelDataDomainService{databaseID: 11}
	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()
	fileSvc := &fakeSemanticModelFileService{}
	workflowTemplateSvc := &fakeSemanticModelWorkflowTemplateService{}
	workflowSvc := &fakeSemanticModelWorkflowService{}
	svc := newSemanticModelTestServiceWithKnowledgeBaseRuntimeDependencies(t, server.URL, systemClient, dataDomainSvc, fileSvc, nil, workflowTemplateSvc, workflowSvc)

	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	tenantMock.ExpectQuery("SELECT model_id, catalog_id, database_id, raw_volume_id, processed_volume_id, ensure_status, last_ensure_error, last_checked_at").
		WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows([]string{
			"model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "ensure_status", "last_ensure_error", "last_checked_at",
		}).AddRow(int64(77), int64(3), int64(11), int64(12), int64(13), kbEnsureStatusReady, nil, int64(100)))
	expectUpsertKnowledgeBaseRawVolume(tenantMock, 77, kbRawKindDocument, false)
	expectAppendSourcesTransactionLock(tenantMock, 77)
	expectCatalogFileSourceLookupMiss(tenantMock, 77, "source-file")
	tenantMock.ExpectExec("INSERT INTO knowledge_base_sources").
		WillReturnResult(sqlmock.NewResult(1, 1))
	expectKnowledgeBaseSourceJobRunUpsertMiss(tenantMock)
	tenantMock.ExpectExec("INSERT INTO knowledge_base_source_job_runs").
		WillReturnResult(sqlmock.NewResult(1, 1))
	expectKnowledgeBaseSourceJobRunUpsertMiss(tenantMock)
	tenantMock.ExpectExec("INSERT INTO knowledge_base_source_job_runs").
		WillReturnResult(sqlmock.NewResult(2, 1))
	tenantMock.ExpectRollback()

	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithUserID(ctx, "user-1")
	ctx = withKnowledgeBaseCreatePrincipal(ctx)
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)

	_, err = svc.AppendModelSources(ctx, AppendSemanticModelSourcesRequest{
		ModelID: 77,
		Sources: []CreateSemanticModelSourceRequest{{
			SourceType: kbSourceTypeCatalogFile,
			FileID:     "source-file",
			VolumeID:   41,
		}},
	})
	if err == nil || !strings.Contains(err.Error(), "update semantic model sources") {
		t.Fatalf("AppendModelSources error = %v", err)
	}
	assertDocumentAppendPreservesExistingWorkflow(t, workflowTemplateSvc, workflowSvc, "ws-1", 77)
	if len(fileSvc.deleted) != 0 {
		t.Fatalf("deleted copied files = %+v", fileSvc.deleted)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceAppendModelSourcesRestoresModelWhenRunPendingFails(t *testing.T) {
	originalTables := []semanticModelTableSource{{
		DBName:     "analytics",
		TableNames: []string{"existing_orders"},
	}}
	originalFiles := map[string]any{
		"file_ids":        []string{"existing-file"},
		"vector_table":    "existing_vector",
		"embedding_model": "existing_embedding",
	}
	var sawAppendUpdate bool
	var updateCount int

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			requireSemanticModelExecutionHeaders(t, r)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":          77,
				"name":        "kb_docs",
				"description": "docs",
				"tables":      originalTables,
				"files":       originalFiles,
			})
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			requireSemanticModelExecutionHeaders(t, r)
			var req struct {
				Name        string                     `json:"name"`
				Description string                     `json:"description"`
				Tables      []semanticModelTableSource `json:"tables"`
				Files       map[string]any             `json:"files"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode semantic update: %v", err)
			}
			if req.Name != "kb_docs" || req.Description != "docs" {
				t.Fatalf("semantic update identity = %+v", req)
			}
			updateCount++
			fileIDs, ok := stringSliceFromAny(req.Files["file_ids"])
			switch {
			case updateCount == 1 && len(req.Tables) == 1 && req.Tables[0].DBName == "analytics" && sameStringSet(req.Tables[0].TableNames, []string{"existing_orders"}) && ok && sameStringSet(fileIDs, []string{"existing-file"}):
				sawAppendUpdate = true
			default:
				t.Fatalf("semantic update payload = %+v", req)
			}
			if req.Files["vector_table"] != "existing_vector" || req.Files["embedding_model"] != "existing_embedding" {
				t.Fatalf("semantic files fields were not preserved: %+v", req.Files)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"updated": true})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/ws-1/files/source-file/download":
			requireSemanticModelExecutionHeaders(t, r)
			w.Header().Set("Content-Disposition", `attachment; filename="source.txt"`)
			_, _ = w.Write([]byte("source content"))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/workspaces/ws-1/files":
			requireSemanticModelExecutionHeaders(t, r)
			_ = json.NewEncoder(w).Encode(map[string]any{"file_id": "source-file"})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/workspaces/ws-1/volumes/12/files":
			t.Fatalf("unexpected AddFiles during append short request")
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/workspaces/ws-1/volumes/12/files/trigger":
			t.Fatalf("unexpected TriggerFiles during append short request")
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	dataDomainSvc := &fakeSemanticModelDataDomainService{databaseID: 11}
	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()
	fileSvc := &fakeSemanticModelFileService{}
	workflowTemplateSvc := &fakeSemanticModelWorkflowTemplateService{}
	workflowSvc := &fakeSemanticModelWorkflowService{}
	svc := newSemanticModelTestServiceWithKnowledgeBaseRuntimeDependencies(t, server.URL, systemClient, dataDomainSvc, fileSvc, nil, workflowTemplateSvc, workflowSvc)

	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	tenantMock.ExpectQuery("SELECT model_id, catalog_id, database_id, raw_volume_id, processed_volume_id, ensure_status, last_ensure_error, last_checked_at").
		WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows([]string{
			"model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "ensure_status", "last_ensure_error", "last_checked_at",
		}).AddRow(int64(77), int64(3), int64(11), int64(12), int64(13), kbEnsureStatusReady, nil, int64(100)))
	expectUpsertKnowledgeBaseRawVolume(tenantMock, 77, kbRawKindDocument, false)
	expectAppendSourcesTransactionLock(tenantMock, 77)
	expectCatalogFileSourceLookupMiss(tenantMock, 77, "source-file")
	tenantMock.ExpectExec("INSERT INTO knowledge_base_sources").
		WillReturnResult(sqlmock.NewResult(1, 1))
	expectKnowledgeBaseSourceJobRunUpsertMiss(tenantMock)
	tenantMock.ExpectExec("INSERT INTO knowledge_base_source_job_runs").
		WillReturnResult(sqlmock.NewResult(1, 1))
	expectKnowledgeBaseSourceJobRunUpsertMiss(tenantMock)
	tenantMock.ExpectExec("INSERT INTO knowledge_base_source_job_runs").
		WillReturnResult(sqlmock.NewResult(2, 1))
	sourceID := stableID("kb-source", int64(77), kbSourceTypeCatalogFile, "source-file")
	tenantMock.ExpectCommit()
	tenantMock.ExpectQuery("SELECT .*source_id.*FROM knowledge_base_sources kbs .*WHERE kbs\\.model_id = \\? AND kbs\\.status <> 'removed' ORDER BY").
		WithArgs(int64(77)).
		WillReturnRows(knowledgeBaseSourceRecordRows(KnowledgeBaseSourceRecord{
			SourceID:          sourceID,
			ModelID:           77,
			CatalogID:         3,
			DatabaseID:        11,
			RawVolumeID:       12,
			ProcessedVolumeID: 13,
			SourceType:        kbSourceTypeCatalogFile,
			SourceFileID:      stringPtr("source-file"),
			KBFileID:          stringPtr("source-file"),
			DisplayName:       stringPtr("source-file"),
			Status:            kbSourceStatusPending,
		}))
	tenantMock.ExpectQuery("SELECT .*job_id.*source_id.*model_id.*job_type.*job_status.*idempotency_key").
		WithArgs(int64(77)).
		WillReturnRows(knowledgeBaseSourceJobRunRows().
			AddRow(stableID("kb-job", sourceID, kbJobTypeCopy), sourceID, int64(77), kbJobTypeCopy, kbSourceJobPending, stableID("kb-job-key", sourceID, kbJobTypeCopy), nil, nil, nil, nil, false, "source-file", "source-file", nil, nil, int64(0), nil, nil, int64(100), int64(100)).
			AddRow(stableID("kb-job", sourceID, kbJobTypeRAGIngest), sourceID, int64(77), kbJobTypeRAGIngest, kbSourceJobPending, stableID("kb-job-key", sourceID, kbJobTypeRAGIngest), nil, nil, nil, nil, false, "source-file", "source-file", nil, nil, int64(0), nil, nil, int64(100), int64(100)))
	tenantMock.ExpectQuery("SELECT model_id, catalog_id, database_id, raw_volume_id, processed_volume_id, ensure_status, last_ensure_error, last_checked_at").
		WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows([]string{
			"model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "ensure_status", "last_ensure_error", "last_checked_at",
		}).AddRow(int64(77), int64(3), int64(11), int64(12), int64(13), kbEnsureStatusReady, nil, int64(100)))
	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithUserID(ctx, "user-1")
	ctx = withKnowledgeBaseCreatePrincipal(ctx)
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)

	resp, err := svc.AppendModelSources(ctx, AppendSemanticModelSourcesRequest{
		ModelID: 77,
		Sources: []CreateSemanticModelSourceRequest{{
			SourceType: kbSourceTypeCatalogFile,
			FileID:     "source-file",
			VolumeID:   41,
		}},
	})
	if err != nil {
		t.Fatalf("AppendModelSources: %v", err)
	}
	assertDocumentAppendPreservesExistingWorkflow(t, workflowTemplateSvc, workflowSvc, "ws-1", 77)
	if !sawAppendUpdate || updateCount != 1 {
		t.Fatalf("semantic updates saw append=%v count=%d", sawAppendUpdate, updateCount)
	}
	if len(resp.Jobs) != 2 || resp.Jobs[0].JobStatus != kbSourceJobPending || resp.Jobs[1].JobStatus != kbSourceJobPending {
		t.Fatalf("jobs = %+v", resp.Jobs)
	}
	if len(fileSvc.deleted) != 0 {
		t.Fatalf("deleted copied files = %+v", fileSvc.deleted)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceAppendModelSourcesRollsBackPartialAppendFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			requireSemanticModelExecutionHeaders(t, r)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":          77,
				"name":        "kb_docs",
				"description": "docs",
				"tables":      []any{},
				"files":       map[string]any{"file_ids": []string{}},
			})
		case strings.Contains(r.URL.Path, "/files/") && strings.Contains(r.URL.Path, "/download"):
			t.Fatalf("unexpected DownloadWithMeta during append short request: %s %s", r.Method, r.URL.String())
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/workspaces/ws-1/files":
			requireSemanticModelExecutionHeaders(t, r)
			_ = json.NewEncoder(w).Encode(map[string]any{"file_id": "source-file"})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/workspaces/ws-1/volumes/12/files":
			requireSemanticModelExecutionHeaders(t, r)
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/workspaces/ws-1/volumes/12/files/trigger":
			requireSemanticModelExecutionHeaders(t, r)
			_ = json.NewEncoder(w).Encode(map[string]any{"triggered": 1})
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			t.Fatalf("semantic model must not be updated after partial append failure")
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	dataDomainSvc := &fakeSemanticModelDataDomainService{databaseID: 11}
	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()
	fileSvc := &fakeSemanticModelFileService{}
	workflowTemplateSvc := &fakeSemanticModelWorkflowTemplateService{}
	workflowSvc := &fakeSemanticModelWorkflowService{}
	svc := newSemanticModelTestServiceWithKnowledgeBaseRuntimeDependencies(t, server.URL, systemClient, dataDomainSvc, fileSvc, nil, workflowTemplateSvc, workflowSvc)

	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	tenantMock.ExpectQuery("SELECT model_id, catalog_id, database_id, raw_volume_id, processed_volume_id, ensure_status, last_ensure_error, last_checked_at").
		WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows([]string{
			"model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "ensure_status", "last_ensure_error", "last_checked_at",
		}).AddRow(int64(77), int64(3), int64(11), int64(12), int64(13), kbEnsureStatusReady, nil, int64(100)))
	expectUpsertKnowledgeBaseRawVolume(tenantMock, 77, kbRawKindDocument, false)
	expectAppendSourcesTransactionLock(tenantMock, 77)
	expectCatalogFileSourceLookupMiss(tenantMock, 77, "source-file")
	tenantMock.ExpectExec("INSERT INTO knowledge_base_sources").
		WillReturnResult(sqlmock.NewResult(1, 1))
	expectKnowledgeBaseSourceJobRunUpsertMiss(tenantMock)
	tenantMock.ExpectExec("INSERT INTO knowledge_base_source_job_runs").
		WillReturnResult(sqlmock.NewResult(1, 1))
	expectKnowledgeBaseSourceJobRunUpsertMiss(tenantMock)
	tenantMock.ExpectExec("INSERT INTO knowledge_base_source_job_runs").
		WillReturnResult(sqlmock.NewResult(2, 1))
	expectCatalogFileSourceLookupMiss(tenantMock, 77, "missing-file")
	tenantMock.ExpectExec("INSERT INTO knowledge_base_sources").
		WillReturnError(errors.New("metadata insert failed"))
	tenantMock.ExpectRollback()
	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithUserID(ctx, "user-1")
	ctx = withKnowledgeBaseCreatePrincipal(ctx)
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)

	_, err = svc.AppendModelSources(ctx, AppendSemanticModelSourcesRequest{
		ModelID: 77,
		Sources: []CreateSemanticModelSourceRequest{
			{SourceType: kbSourceTypeCatalogFile, FileID: "source-file", VolumeID: 41},
			{SourceType: kbSourceTypeCatalogFile, FileID: "missing-file", VolumeID: 41},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "metadata insert failed") {
		t.Fatalf("AppendModelSources error = %v", err)
	}
	assertDocumentAppendPreservesExistingWorkflow(t, workflowTemplateSvc, workflowSvc, "ws-1", 77)
	if len(fileSvc.deleted) != 0 {
		t.Fatalf("deleted copied files = %+v", fileSvc.deleted)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceAppendModelSourcesReusesExistingCatalogSourcesAndDedupesScope(t *testing.T) {
	currentTables := []semanticModelTableSource{{
		DBName:     "kb_docs",
		TableNames: []string{"orders"},
	}}
	currentFiles := map[string]any{
		"file_ids":        []string{"kb-existing-file"},
		"vector_table":    "kb_text_idx",
		"embedding_model": "embed-model",
	}
	var updateFiles []string
	var updateTables []semanticModelTableSource
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":          77,
				"name":        "kb_docs",
				"description": "docs",
				"tables":      currentTables,
				"files":       currentFiles,
			})
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			var req struct {
				Name        string                     `json:"name"`
				Description string                     `json:"description"`
				Tables      []semanticModelTableSource `json:"tables"`
				Files       map[string]any             `json:"files"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode semantic update: %v", err)
			}
			if req.Name != "kb_docs" || req.Description != "docs" {
				t.Fatalf("semantic update identity = %+v", req)
			}
			var ok bool
			updateFiles, ok = stringSliceFromAny(req.Files["file_ids"])
			if !ok {
				t.Fatalf("semantic update files = %+v", req.Files)
			}
			updateTables = req.Tables
			currentFiles = req.Files
			currentTables = req.Tables
			_ = json.NewEncoder(w).Encode(map[string]any{"updated": true})
		case strings.Contains(r.URL.Path, "/files/source-file"):
			t.Fatalf("catalog file should not be copied on reuse: %s %s", r.Method, r.URL.Path)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	dataDomainSvc := &fakeSemanticModelDataDomainService{databaseID: 11}
	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()
	workflowTemplateSvc := &fakeSemanticModelWorkflowTemplateService{}
	workflowSvc := &fakeSemanticModelWorkflowService{}
	svc := newSemanticModelTestServiceWithKnowledgeBaseRuntimeDependencies(t, server.URL, systemClient, dataDomainSvc, nil, nil, workflowTemplateSvc, workflowSvc)

	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	tenantMock.ExpectQuery("SELECT model_id, catalog_id, database_id, raw_volume_id, processed_volume_id, ensure_status, last_ensure_error, last_checked_at").
		WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows([]string{
			"model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "ensure_status", "last_ensure_error", "last_checked_at",
		}).AddRow(int64(77), int64(3), int64(11), int64(12), int64(13), kbEnsureStatusReady, nil, int64(100)))
	expectUpsertKnowledgeBaseRawVolume(tenantMock, 77, kbRawKindDocument, false)
	expectAppendSourcesTransactionLock(tenantMock, 77)
	existingFile := KnowledgeBaseSourceRecord{
		SourceID:   "source-existing-file",
		ModelID:    77,
		CatalogID:  3,
		DatabaseID: 11,
		// Catalog file provenance is the selected source volume, not domain raw.
		RawVolumeID:       41,
		ProcessedVolumeID: 13,
		SourceType:        kbSourceTypeCatalogFile,
		SourceFileID:      stringPtr("source-file"),
		KBFileID:          stringPtr("kb-existing-file"),
		DisplayName:       stringPtr("doc.pdf"),
		Status:            kbSourceStatusSucceeded,
		Enabled:           boolPtr(false),
		ExpiresAt:         int64Ptr(1782700000),
		Tags:              stringPtr(`["keep"]`),
		ForceEnabled:      true,
		SegmentVersionID:  stringPtr("seg-current"),
		IndexVersion:      int64Ptr(9),
	}
	existingTable := KnowledgeBaseSourceRecord{
		SourceID:          "source-existing-table",
		ModelID:           77,
		CatalogID:         3,
		DatabaseID:        11,
		RawVolumeID:       12,
		ProcessedVolumeID: 13,
		SourceType:        kbSourceTypeCatalogTable,
		SourceTableID:     int64Ptr(1001),
		KBTableID:         int64Ptr(2001),
		DisplayName:       stringPtr("orders"),
		DBName:            stringPtr("kb_docs"),
		TableName:         stringPtr("orders"),
		Status:            kbSourceStatusSucceeded,
		Enabled:           boolPtr(true),
		SegmentVersionID:  stringPtr("table-seg-current"),
		IndexVersion:      int64Ptr(3),
	}
	expectCatalogFileSourceLookupHit(tenantMock, 77, "source-file", existingFile)
	expectCatalogTableSourceLookupHit(tenantMock, 77, 1001, existingTable)
	tenantMock.ExpectCommit()
	tenantMock.ExpectQuery("SELECT .*source_id.*FROM knowledge_base_sources").
		WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows([]string{
			"source_id", "model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "source_type",
			"source_file_id", "source_table_id", "kb_file_id", "kb_table_id", "display_name", "source_path",
			"db_name", "table_name", "status", "error", "enabled", "expires_at", "tags", "force_enabled_after_expiry", "segment_version_id", "index_version",
		}).
			AddRow("source-existing-file", int64(77), int64(3), int64(11), int64(41), int64(13), kbSourceTypeCatalogFile, "source-file", nil, "kb-existing-file", nil, "doc.pdf", nil, nil, nil, kbSourceStatusSucceeded, nil, false, int64(1782700000), `["keep"]`, true, "seg-current", int64(9)).
			AddRow("source-existing-table", int64(77), int64(3), int64(11), int64(12), int64(13), kbSourceTypeCatalogTable, nil, int64(1001), nil, int64(2001), "orders", nil, "kb_docs", "orders", kbSourceStatusSucceeded, nil, true, nil, nil, false, "table-seg-current", int64(3)))
	tenantMock.ExpectQuery("SELECT .*job_id.*source_id.*model_id.*job_type.*job_status.*idempotency_key").
		WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows([]string{
			"job_id", "source_id", "model_id", "job_type", "job_status", "idempotency_key", "operation_id",
			"workflow_execution_id", "runtime_actor_moi_user_id", "runtime_effective_role_id", "runtime_is_workspace_owner", "source_file_id", "kb_file_id", "source_table_id", "kb_table_id",
			"retry_count", "next_retry_at", "error", "created_at", "updated_at",
		}))
	tenantMock.ExpectQuery("SELECT model_id, catalog_id, database_id, raw_volume_id, processed_volume_id, ensure_status, last_ensure_error, last_checked_at").
		WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows([]string{
			"model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "ensure_status", "last_ensure_error", "last_checked_at",
		}).AddRow(int64(77), int64(3), int64(11), int64(12), int64(13), kbEnsureStatusReady, nil, int64(100)))
	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithUserID(ctx, "user-1")
	ctx = withKnowledgeBaseCreatePrincipal(ctx)
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)

	resp, err := svc.AppendModelSources(ctx, AppendSemanticModelSourcesRequest{
		ModelID: 77,
		Sources: []CreateSemanticModelSourceRequest{
			{SourceType: kbSourceTypeCatalogFile, FileID: "source-file", VolumeID: 41},
			{SourceType: kbSourceTypeCatalogTable, TableID: 1001},
		},
	})
	if err != nil {
		t.Fatalf("AppendModelSources: %v", err)
	}
	if !sameStringSet(updateFiles, []string{"kb-existing-file"}) {
		t.Fatalf("semantic update files = %+v", updateFiles)
	}
	if len(updateTables) != 1 || updateTables[0].DBName != "kb_docs" || !sameStringSet(updateTables[0].TableNames, []string{"orders"}) {
		t.Fatalf("semantic update tables = %+v", updateTables)
	}
	if len(resp.Jobs) != 0 {
		t.Fatalf("jobs = %+v, want none for reused sources", resp.Jobs)
	}
	if len(resp.Sources) != 2 {
		t.Fatalf("sources = %+v", resp.Sources)
	}
	fileSource := resp.Sources[0]
	if fileSource.RowID != "source-existing-file" || fileSource.Enabled == nil || *fileSource.Enabled || fileSource.ExpiresAt == nil || *fileSource.ExpiresAt != 1782700000 || !fileSource.ForceEnabled || fileSource.SegmentVersionID == nil || *fileSource.SegmentVersionID != "seg-current" || fileSource.IndexVersion == nil || *fileSource.IndexVersion != 9 {
		t.Fatalf("file source governance was not preserved: %+v", fileSource)
	}
	if len(dataDomainSvc.calls) != 0 {
		t.Fatalf("data domain service calls = %+v, want none", dataDomainSvc.calls)
	}
	assertDocumentAppendPreservesExistingWorkflow(t, workflowTemplateSvc, workflowSvc, "ws-1", 77)
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceAppendModelSourcesTreatsStructuredTableAsExistingRelation(t *testing.T) {
	currentTables := []semanticModelTableSource{{
		DBName:     "sales",
		TableNames: []string{"orders"},
	}}
	currentFiles := map[string]any{
		"file_ids":        []string{},
		"vector_table":    "kb_text_idx",
		"embedding_model": "embed-model",
	}
	var updateTables []semanticModelTableSource
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":          77,
				"name":        "kb_docs",
				"description": "docs",
				"tables":      currentTables,
				"files":       currentFiles,
			})
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			var req struct {
				Name        string                     `json:"name"`
				Description string                     `json:"description"`
				Tables      []semanticModelTableSource `json:"tables"`
				Files       map[string]any             `json:"files"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode semantic update: %v", err)
			}
			updateTables = req.Tables
			currentFiles = req.Files
			currentTables = req.Tables
			_ = json.NewEncoder(w).Encode(map[string]any{"updated": true})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	dataDomainSvc := &fakeSemanticModelDataDomainService{databaseID: 11}
	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()
	workflowTemplateSvc := &fakeSemanticModelWorkflowTemplateService{}
	workflowSvc := &fakeSemanticModelWorkflowService{}
	svc := newSemanticModelTestServiceWithKnowledgeBaseRuntimeDependencies(t, server.URL, systemClient, dataDomainSvc, nil, nil, workflowTemplateSvc, workflowSvc)

	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	tenantMock.ExpectQuery("SELECT model_id, catalog_id, database_id, raw_volume_id, processed_volume_id, ensure_status, last_ensure_error, last_checked_at").
		WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows([]string{
			"model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "ensure_status", "last_ensure_error", "last_checked_at",
		}).AddRow(int64(77), int64(3), int64(11), int64(12), int64(13), kbEnsureStatusReady, nil, int64(100)))
	expectUpsertKnowledgeBaseRawVolume(tenantMock, 77, kbRawKindDocument, false)
	expectAppendSourcesTransactionLock(tenantMock, 77)
	existingStructuredTable := KnowledgeBaseSourceRecord{
		SourceID:          "source-structured-table",
		ModelID:           77,
		CatalogID:         3,
		DatabaseID:        11,
		RawVolumeID:       12,
		ProcessedVolumeID: 13,
		SourceType:        kbSourceTypeCatalogTable,
		KBTableID:         int64Ptr(1001),
		DisplayName:       stringPtr("orders"),
		DBName:            stringPtr("sales"),
		TableName:         stringPtr("orders"),
		Status:            kbSourceStatusSucceeded,
		Enabled:           boolPtr(true),
	}
	expectCatalogTableSourceLookupHit(tenantMock, 77, 1001, existingStructuredTable)
	tenantMock.ExpectCommit()
	sourcesColumns := []string{
		"source_id", "model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "source_type",
		"source_file_id", "source_table_id", "kb_file_id", "kb_table_id", "display_name", "source_path",
		"db_name", "table_name", "status", "error", "enabled", "expires_at", "tags", "force_enabled_after_expiry", "segment_version_id", "index_version",
	}
	tenantMock.ExpectQuery("SELECT .*source_id.*FROM knowledge_base_sources").
		WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows(sourcesColumns).
			AddRow("source-structured-table", int64(77), int64(3), int64(11), int64(12), int64(13), kbSourceTypeCatalogTable, nil, nil, nil, int64(1001), "orders", nil, "sales", "orders", kbSourceStatusSucceeded, nil, true, nil, nil, false, nil, nil))
	tenantMock.ExpectQuery("SELECT .*job_id.*source_id.*model_id.*job_type.*job_status.*idempotency_key").
		WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows([]string{
			"job_id", "source_id", "model_id", "job_type", "job_status", "idempotency_key", "operation_id",
			"workflow_execution_id", "runtime_actor_moi_user_id", "runtime_effective_role_id", "runtime_is_workspace_owner", "source_file_id", "kb_file_id", "source_table_id", "kb_table_id",
			"retry_count", "next_retry_at", "error", "created_at", "updated_at",
		}))
	tenantMock.ExpectQuery("SELECT model_id, catalog_id, database_id, raw_volume_id, processed_volume_id, ensure_status, last_ensure_error, last_checked_at").
		WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows([]string{
			"model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "ensure_status", "last_ensure_error", "last_checked_at",
		}).AddRow(int64(77), int64(3), int64(11), int64(12), int64(13), kbEnsureStatusReady, nil, int64(100)))
	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithUserID(ctx, "user-1")
	ctx = withKnowledgeBaseCreatePrincipal(ctx)
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)

	resp, err := svc.AppendModelSources(ctx, AppendSemanticModelSourcesRequest{
		ModelID: 77,
		Sources: []CreateSemanticModelSourceRequest{
			{SourceType: kbSourceTypeCatalogTable, TableID: 1001},
		},
	})
	if err != nil {
		t.Fatalf("AppendModelSources: %v", err)
	}
	if len(resp.Jobs) != 0 {
		t.Fatalf("jobs = %+v, want none for existing table relation", resp.Jobs)
	}
	if len(resp.Sources) != 1 || resp.Sources[0].RowID != "source-structured-table" || resp.Sources[0].KBTableID == nil || *resp.Sources[0].KBTableID != 1001 {
		t.Fatalf("sources = %+v, want existing structured table relation", resp.Sources)
	}
	if len(dataDomainSvc.calls) != 0 {
		t.Fatalf("data domain service calls = %+v, want no clone", dataDomainSvc.calls)
	}
	if len(updateTables) != 1 || updateTables[0].DBName != "sales" || !sameStringSet(updateTables[0].TableNames, []string{"orders"}) {
		t.Fatalf("semantic update tables = %+v, want unchanged existing table", updateTables)
	}
	assertStructuredOnlyAppendDoesNotTouchDocumentWorkflow(t, workflowTemplateSvc, workflowSvc)
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceAppendModelSourcesAddsCatalogTableDespiteFailedStructuredLoad(t *testing.T) {
	var tableUpdates [][]semanticModelTableSource
	currentTables := []semanticModelTableSource{}
	currentFiles := map[string]any{
		"file_ids":        []string{},
		"vector_table":    "kb_text_idx",
		"embedding_model": "embed-model",
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case serveCatalogTableDetail(t, w, r, 1001, 3, 11, "catalog", "sales", "orders"):
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":          77,
				"name":        "kb_docs",
				"description": "docs",
				"tables":      currentTables,
				"files":       currentFiles,
			})
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			var req struct {
				Name        string                     `json:"name"`
				Description string                     `json:"description"`
				Tables      []semanticModelTableSource `json:"tables"`
				Files       map[string]any             `json:"files"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode semantic update: %v", err)
			}
			tableUpdates = append(tableUpdates, req.Tables)
			currentTables = req.Tables
			currentFiles = req.Files
			_ = json.NewEncoder(w).Encode(map[string]any{"updated": true})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	dataDomainSvc := &fakeSemanticModelDataDomainService{databaseID: 11}
	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()
	workflowTemplateSvc := &fakeSemanticModelWorkflowTemplateService{}
	workflowSvc := &fakeSemanticModelWorkflowService{}
	svc := newSemanticModelTestServiceWithKnowledgeBaseRuntimeDependencies(t, server.URL, systemClient, dataDomainSvc, nil, nil, workflowTemplateSvc, workflowSvc)

	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}

	failedSourceID := "source-failed-structured-table"
	directSourceID := stableID("kb-source", int64(77), kbSourceTypeCatalogTable, int64(1001))
	tenantMock.ExpectQuery("SELECT model_id, catalog_id, database_id, raw_volume_id, processed_volume_id, ensure_status, last_ensure_error, last_checked_at").
		WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows([]string{
			"model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "ensure_status", "last_ensure_error", "last_checked_at",
		}).AddRow(int64(77), int64(3), int64(11), int64(12), int64(13), kbEnsureStatusReady, nil, int64(100)))
	expectUpsertKnowledgeBaseRawVolume(tenantMock, 77, kbRawKindDocument, false)
	expectAppendSourcesTransactionLock(tenantMock, 77)
	failedStructuredTable := KnowledgeBaseSourceRecord{
		SourceID:          failedSourceID,
		ModelID:           77,
		CatalogID:         3,
		DatabaseID:        11,
		RawVolumeID:       12,
		ProcessedVolumeID: 13,
		SourceType:        kbSourceTypeCatalogTable,
		KBTableID:         int64Ptr(1001),
		DisplayName:       stringPtr("rtrrfgf"),
		DBName:            stringPtr("kb_docs"),
		TableName:         stringPtr("rtrrfgf"),
		Status:            kbSourceStatusFailed,
		Error:             stringPtr("previous clone failed"),
		Enabled:           boolPtr(true),
	}
	expectCatalogTableSourceLookupMiss(tenantMock, 77, 1001)
	tenantMock.ExpectExec("INSERT INTO knowledge_base_sources").
		WillReturnResult(sqlmock.NewResult(1, 1))
	tenantMock.ExpectCommit()
	tenantMock.ExpectQuery("SELECT .*source_id.*FROM knowledge_base_sources").
		WithArgs(int64(77)).
		WillReturnRows(knowledgeBaseSourceRecordRows(
			failedStructuredTable,
			KnowledgeBaseSourceRecord{
				SourceID:      directSourceID,
				ModelID:       77,
				CatalogID:     3,
				DatabaseID:    11,
				SourceType:    kbSourceTypeCatalogTable,
				SourceTableID: int64Ptr(1001),
				DisplayName:   stringPtr("orders"),
				SourcePath:    stringPtr(`["catalog","sales"]`),
				DBName:        stringPtr("sales"),
				TableName:     stringPtr("orders"),
				Status:        kbSourceStatusSucceeded,
				Enabled:       boolPtr(true),
				Tags:          stringPtr("[]"),
			},
		))
	tenantMock.ExpectQuery("SELECT .*job_id.*source_id.*model_id.*job_type.*job_status.*idempotency_key").
		WithArgs(int64(77)).
		WillReturnRows(knowledgeBaseSourceJobRunRows())
	tenantMock.ExpectQuery("SELECT model_id, catalog_id, database_id, raw_volume_id, processed_volume_id, ensure_status, last_ensure_error, last_checked_at").
		WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows([]string{
			"model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "ensure_status", "last_ensure_error", "last_checked_at",
		}).AddRow(int64(77), int64(3), int64(11), int64(12), int64(13), kbEnsureStatusReady, nil, int64(100)))
	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithUserID(ctx, "user-1")
	ctx = withKnowledgeBaseCreatePrincipal(ctx)
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)

	resp, err := svc.AppendModelSources(ctx, AppendSemanticModelSourcesRequest{
		ModelID: 77,
		Sources: []CreateSemanticModelSourceRequest{
			{SourceType: kbSourceTypeCatalogTable, TableID: 1001},
		},
	})
	if err != nil {
		t.Fatalf("AppendModelSources: %v", err)
	}
	if len(resp.Sources) != 2 || resp.Sources[0].RowID != failedSourceID || resp.Sources[1].RowID != directSourceID || resp.Sources[1].ResourceID != "1001" || resp.Sources[1].IngestStatus == nil || *resp.Sources[1].IngestStatus != kbSourceStatusSucceeded {
		t.Fatalf("sources = %+v, want failed structured source plus direct catalog table", resp.Sources)
	}
	if len(resp.Jobs) != 0 {
		t.Fatalf("jobs = %+v, want none for direct catalog table", resp.Jobs)
	}
	if len(dataDomainSvc.calls) != 0 {
		t.Fatalf("data domain service calls = %+v, want no clone in append request", dataDomainSvc.calls)
	}
	if len(tableUpdates) != 1 || len(tableUpdates[0]) != 1 || tableUpdates[0][0].DBName != "sales" || !sameStringSet(tableUpdates[0][0].TableNames, []string{"orders"}) {
		t.Fatalf("semantic table updates = %+v, want direct catalog table", tableUpdates)
	}
	assertStructuredOnlyAppendDoesNotTouchDocumentWorkflow(t, workflowTemplateSvc, workflowSvc)
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceAppendModelSourcesRejectsInaccessibleCatalogTable(t *testing.T) {
	semanticUpdateCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			requireSemanticModelExecutionHeaders(t, r)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":          77,
				"name":        "kb_docs",
				"description": "docs",
				"tables":      []any{},
				"files": map[string]any{
					"file_ids":        []string{},
					"vector_table":    "kb_text_idx",
					"embedding_model": "embed-model",
				},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/ws-1/tables/1001":
			requireSemanticModelExecutionHeaders(t, r)
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"code":7,"message":"catalog table access denied"}`))
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			semanticUpdateCalls++
			t.Fatalf("unexpected semantic model update")
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	dataDomainSvc := &fakeSemanticModelDataDomainService{databaseID: 11}
	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()
	workflowTemplateSvc := &fakeSemanticModelWorkflowTemplateService{}
	workflowSvc := &fakeSemanticModelWorkflowService{}
	svc := newSemanticModelTestServiceWithKnowledgeBaseRuntimeDependencies(t, server.URL, systemClient, dataDomainSvc, nil, nil, workflowTemplateSvc, workflowSvc)

	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	tenantMock.ExpectQuery("SELECT model_id, catalog_id, database_id, raw_volume_id, processed_volume_id, ensure_status, last_ensure_error, last_checked_at").
		WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows([]string{
			"model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "ensure_status", "last_ensure_error", "last_checked_at",
		}).AddRow(int64(77), int64(3), int64(11), int64(12), int64(13), kbEnsureStatusReady, nil, int64(100)))
	expectUpsertKnowledgeBaseRawVolume(tenantMock, 77, kbRawKindDocument, false)
	expectAppendSourcesTransactionLock(tenantMock, 77)
	expectCatalogTableSourceLookupMiss(tenantMock, 77, 1001)
	tenantMock.ExpectRollback()

	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithUserID(ctx, "user-1")
	ctx = withKnowledgeBaseCreatePrincipal(ctx)
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)

	_, err = svc.AppendModelSources(ctx, AppendSemanticModelSourcesRequest{
		ModelID: 77,
		Sources: []CreateSemanticModelSourceRequest{
			{SourceType: kbSourceTypeCatalogTable, TableID: 1001},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "catalog table access denied") {
		t.Fatalf("AppendModelSources error = %v, want catalog table access denied", err)
	}
	if semanticUpdateCalls != 0 {
		t.Fatalf("semantic updates = %d, want none", semanticUpdateCalls)
	}
	if len(dataDomainSvc.calls) != 0 {
		t.Fatalf("data domain service calls = %+v, want no clone", dataDomainSvc.calls)
	}
	assertStructuredOnlyAppendDoesNotTouchDocumentWorkflow(t, workflowTemplateSvc, workflowSvc)
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceAppendModelSourcesReactivatesRemovedCatalogTableSource(t *testing.T) {
	tests := []struct {
		name    string
		request AppendSemanticModelSourcesRequest
	}{
		{
			name: "explicit source",
			request: AppendSemanticModelSourcesRequest{
				ModelID: 77,
				Sources: []CreateSemanticModelSourceRequest{
					{SourceType: kbSourceTypeCatalogTable, TableID: 1001},
				},
			},
		},
		{
			name: "database table selection",
			request: AppendSemanticModelSourcesRequest{
				ModelID: 77,
				SourceSelections: []SemanticModelSourceSelectionRequest{{
					Kind:        kbSelectionKindDatabaseTables,
					DatabaseID:  11,
					AllSelected: true,
				}},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			currentTables := []semanticModelTableSource{}
			currentFiles := map[string]any{
				"file_ids":        []string{},
				"vector_table":    "kb_text_idx",
				"embedding_model": "embed-model",
			}
			var tableUpdates [][]semanticModelTableSource
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch {
				case serveCatalogTableDetail(t, w, r, 1001, 3, 11, "catalog", "sales", "orders"):
				case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
					_ = json.NewEncoder(w).Encode(map[string]any{
						"id":          77,
						"name":        "kb_docs",
						"description": "docs",
						"tables":      currentTables,
						"files":       currentFiles,
					})
				case r.Method == http.MethodPut && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
					var req struct {
						Tables []semanticModelTableSource `json:"tables"`
						Files  map[string]any             `json:"files"`
					}
					if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
						t.Fatalf("decode semantic update: %v", err)
					}
					tableUpdates = append(tableUpdates, req.Tables)
					currentTables = req.Tables
					currentFiles = req.Files
					_ = json.NewEncoder(w).Encode(map[string]any{"updated": true})
				default:
					t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
				}
			}))
			defer server.Close()

			dataDomainSvc := &fakeSemanticModelDataDomainService{databaseID: 11}
			systemClient, err := moi.New(server.URL, "system-key")
			if err != nil {
				t.Fatalf("moi.New: %v", err)
			}
			defer systemClient.Close()
			workflowTemplateSvc := &fakeSemanticModelWorkflowTemplateService{}
			workflowSvc := &fakeSemanticModelWorkflowService{}
			svc := newSemanticModelTestServiceWithKnowledgeBaseRuntimeDependencies(t, server.URL, systemClient, dataDomainSvc, nil, nil, workflowTemplateSvc, workflowSvc)

			tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
			if err != nil {
				t.Fatalf("tenant sqlmock: %v", err)
			}
			defer tenantSQL.Close()
			tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
			if err != nil {
				t.Fatalf("open tenant gorm: %v", err)
			}
			if len(tc.request.SourceSelections) > 0 {
				tenantMock.ExpectQuery("SELECT source_table_id, kb_table_id\\s+FROM knowledge_base_sources").
					WithArgs(int64(77), kbSourceTypeCatalogTable, kbSourceStatusRemoved, int64(1001), int64(1001)).
					WillReturnRows(sqlmock.NewRows([]string{"source_table_id", "kb_table_id"}))
			}

			sourceID := stableID("kb-source", int64(77), kbSourceTypeCatalogTable, int64(1001))
			tenantMock.ExpectQuery("SELECT model_id, catalog_id, database_id, raw_volume_id, processed_volume_id, ensure_status, last_ensure_error, last_checked_at").
				WithArgs(int64(77)).
				WillReturnRows(sqlmock.NewRows([]string{
					"model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "ensure_status", "last_ensure_error", "last_checked_at",
				}).AddRow(int64(77), int64(3), int64(11), int64(12), int64(13), kbEnsureStatusReady, nil, int64(100)))
			expectUpsertKnowledgeBaseRawVolume(tenantMock, 77, kbRawKindDocument, false)
			expectAppendSourcesTransactionLock(tenantMock, 77)
			expectCatalogTableSourceLookupHit(tenantMock, 77, 1001, KnowledgeBaseSourceRecord{
				SourceID:          sourceID,
				ModelID:           77,
				CatalogID:         3,
				DatabaseID:        11,
				RawVolumeID:       12,
				ProcessedVolumeID: 13,
				SourceType:        kbSourceTypeCatalogTable,
				SourceTableID:     int64Ptr(1001),
				KBTableID:         int64Ptr(2001),
				DisplayName:       stringPtr("orders"),
				DBName:            stringPtr("kb_docs"),
				TableName:         stringPtr("orders"),
				Status:            kbSourceStatusRemoved,
				Error:             stringPtr("removed by user"),
				Enabled:           boolPtr(true),
			})
			tenantMock.ExpectExec("UPDATE knowledge_base_sources").
				WithArgs(int64(3), int64(11), int64(0), int64(0), int64(1001), nil, "orders", `["catalog","sales"]`, "sales", "orders", kbSourceStatusSucceeded, true, "user-1", sourceID, int64(77), kbSourceTypeCatalogTable).
				WillReturnResult(sqlmock.NewResult(0, 1))
			tenantMock.ExpectCommit()
			tenantMock.ExpectQuery("SELECT .*source_id.*FROM knowledge_base_sources").
				WithArgs(int64(77)).
				WillReturnRows(knowledgeBaseSourceRecordRows(KnowledgeBaseSourceRecord{
					SourceID:      sourceID,
					ModelID:       77,
					CatalogID:     3,
					DatabaseID:    11,
					SourceType:    kbSourceTypeCatalogTable,
					SourceTableID: int64Ptr(1001),
					DisplayName:   stringPtr("orders"),
					SourcePath:    stringPtr(`["catalog","sales"]`),
					DBName:        stringPtr("sales"),
					TableName:     stringPtr("orders"),
					Status:        kbSourceStatusSucceeded,
					Enabled:       boolPtr(true),
					Tags:          stringPtr("[]"),
				}))
			tenantMock.ExpectQuery("SELECT .*job_id.*source_id.*model_id.*job_type.*job_status.*idempotency_key").
				WithArgs(int64(77)).
				WillReturnRows(knowledgeBaseSourceJobRunRows())
			tenantMock.ExpectQuery("SELECT model_id, catalog_id, database_id, raw_volume_id, processed_volume_id, ensure_status, last_ensure_error, last_checked_at").
				WithArgs(int64(77)).
				WillReturnRows(sqlmock.NewRows([]string{
					"model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "ensure_status", "last_ensure_error", "last_checked_at",
				}).AddRow(int64(77), int64(3), int64(11), int64(12), int64(13), kbEnsureStatusReady, nil, int64(100)))
			ctx := ctxutil.WithUID(context.Background(), "user-1")
			ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
			ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
			ctx = ctxutil.WithUserID(ctx, "user-1")
			ctx = ctxutil.WithTenantDB(ctx, tenantDB)

			resp, err := svc.AppendModelSources(ctx, tc.request)
			if err != nil {
				t.Fatalf("AppendModelSources: %v", err)
			}
			if len(resp.Sources) != 1 || resp.Sources[0].RowID != sourceID || resp.Sources[0].ResourceID != "1001" || resp.Sources[0].KBTableID != nil || resp.Sources[0].IngestStatus == nil || *resp.Sources[0].IngestStatus != kbSourceStatusSucceeded {
				t.Fatalf("sources = %+v, want reactivated direct source", resp.Sources)
			}
			if len(resp.Jobs) != 0 {
				t.Fatalf("jobs = %+v, want none for direct catalog table", resp.Jobs)
			}
			if len(tableUpdates) != 1 || len(tableUpdates[0]) != 1 || tableUpdates[0][0].DBName != "sales" || !sameStringSet(tableUpdates[0][0].TableNames, []string{"orders"}) {
				t.Fatalf("semantic table updates = %+v, want direct catalog table", tableUpdates)
			}
			assertStructuredOnlyAppendDoesNotTouchDocumentWorkflow(t, workflowTemplateSvc, workflowSvc)
			if err := tenantMock.ExpectationsWereMet(); err != nil {
				t.Fatalf("tenant sql expectations: %v", err)
			}
		})
	}
}

func TestSemanticModelServiceExpandSelectionExcludesExistingKnowledgeBaseSources(t *testing.T) {
	tests := []struct {
		name       string
		selection  SemanticModelSourceSelectionRequest
		service    *semanticModelService
		expect     func(sqlmock.Sqlmock)
		wantIDs    []string
		wantErrKey i18n.Key
	}{
		{
			name:      "table",
			selection: SemanticModelSourceSelectionRequest{Kind: kbSelectionKindDatabaseTables, DatabaseID: 11, AllSelected: true},
			service:   &semanticModelService{dataDomainService: &fakeSemanticModelDataDomainService{}},
			expect: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT source_table_id, kb_table_id\\s+FROM knowledge_base_sources").
					WithArgs(int64(77), kbSourceTypeCatalogTable, kbSourceStatusRemoved, int64(1001), int64(1001)).
					WillReturnRows(sqlmock.NewRows([]string{"source_table_id", "kb_table_id"}).AddRow(int64(1001), nil))
			},
			wantIDs: []string{},
		},
		{
			name:      "file",
			selection: SemanticModelSourceSelectionRequest{Kind: kbSelectionKindVolumeFiles, VolumeID: 42, AllSelected: true},
			service: &semanticModelService{fileService: &fakeSemanticModelFileService{listFiles: func(params KnowledgeBaseCatalogFileListParams) (*KnowledgeBaseCatalogFileListResult, error) {
				return &KnowledgeBaseCatalogFileListResult{Items: []KnowledgeBaseCatalogFileLeaf{
					{FileID: "file-1", FileName: "file-1.pdf", VolumeID: params.VolumeID},
					{FileID: "file-2", FileName: "file-2.pdf", VolumeID: params.VolumeID},
				}, Total: 2}, nil
			}}},
			expect: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT source_file_id, kb_file_id, raw_volume_id\\s+FROM knowledge_base_sources").
					WithArgs(int64(77), kbSourceTypeCatalogFile, kbSourceStatusRemoved, "file-1", "file-2", "file-1", "file-2").
					WillReturnRows(sqlmock.NewRows([]string{"source_file_id", "kb_file_id", "raw_volume_id"}).AddRow("file-1", nil, int64(42)))
			},
			wantIDs: []string{"file-2"},
		},
		{
			name:      "explicit file ids",
			selection: SemanticModelSourceSelectionRequest{Kind: kbSelectionKindVolumeFiles, VolumeID: 42, SelectedFileIDs: []string{"file-1", "file-2"}},
			service: &semanticModelService{fileService: &fakeSemanticModelFileService{listFiles: func(params KnowledgeBaseCatalogFileListParams) (*KnowledgeBaseCatalogFileListResult, error) {
				return &KnowledgeBaseCatalogFileListResult{Items: []KnowledgeBaseCatalogFileLeaf{
					{FileID: "file-1", FileName: "file-1.pdf", VolumeID: params.VolumeID},
					{FileID: "file-2", FileName: "file-2.pdf", VolumeID: params.VolumeID},
				}, Total: 2}, nil
			}}},
			expect: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT source_file_id, kb_file_id, raw_volume_id\\s+FROM knowledge_base_sources").
					WithArgs(int64(77), kbSourceTypeCatalogFile, kbSourceStatusRemoved, "file-1", "file-2", "file-1", "file-2").
					WillReturnRows(sqlmock.NewRows([]string{"source_file_id", "kb_file_id", "raw_volume_id"}).AddRow("file-1", nil, int64(42)))
			},
			wantIDs: []string{"file-2"},
		},
		{
			name:      "file volume conflict with existing source",
			selection: SemanticModelSourceSelectionRequest{Kind: kbSelectionKindVolumeFiles, VolumeID: 52, SelectedFileIDs: []string{"shared-file"}},
			service: &semanticModelService{fileService: &fakeSemanticModelFileService{listFiles: func(params KnowledgeBaseCatalogFileListParams) (*KnowledgeBaseCatalogFileListResult, error) {
				return &KnowledgeBaseCatalogFileListResult{Items: []KnowledgeBaseCatalogFileLeaf{
					{FileID: "shared-file", FileName: "shared.pdf", VolumeID: params.VolumeID},
				}, Total: 1}, nil
			}}},
			expect: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT source_file_id, kb_file_id, raw_volume_id\\s+FROM knowledge_base_sources").
					WithArgs(int64(77), kbSourceTypeCatalogFile, kbSourceStatusRemoved, "shared-file", "shared-file").
					WillReturnRows(sqlmock.NewRows([]string{"source_file_id", "kb_file_id", "raw_volume_id"}).AddRow("shared-file", "shared-file", int64(41)))
			},
			wantErrKey: i18n.KeySessionSemanticModelCatalogFileVolumeConflict,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
			if err != nil {
				t.Fatalf("tenant sqlmock: %v", err)
			}
			defer tenantSQL.Close()
			tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
			if err != nil {
				t.Fatalf("open tenant gorm: %v", err)
			}
			tt.expect(tenantMock)
			ctx := ctxutil.WithTenantDB(context.Background(), tenantDB)
			sources, err := tt.service.expandSemanticModelSourceSelections(ctx, nil, "ws-1", 77, []SemanticModelSourceSelectionRequest{tt.selection}, nil)
			if tt.wantErrKey.String() != "" {
				if err == nil {
					t.Fatalf("expandSemanticModelSourceSelections = %+v, want error %s", sources, tt.wantErrKey)
				}
				if !i18n.IsKey(err, tt.wantErrKey) {
					t.Fatalf("error key = %v, want %s", err, tt.wantErrKey)
				}
			} else {
				if err != nil {
					t.Fatalf("expandSemanticModelSourceSelections: %v", err)
				}
				gotIDs := make([]string, 0, len(sources))
				for _, source := range sources {
					gotIDs = append(gotIDs, source.FileID)
				}
				if !reflect.DeepEqual(gotIDs, tt.wantIDs) {
					t.Fatalf("source file IDs = %+v, want %+v after excluding existing sources", gotIDs, tt.wantIDs)
				}
			}
			if err := tenantMock.ExpectationsWereMet(); err != nil {
				t.Fatalf("tenant sql expectations: %v", err)
			}
		})
	}
}

func TestSemanticModelServiceExpandVolumeFileSelectionAllSelectedUsesFiltersExcludesAndPages(t *testing.T) {
	fileSvc := &fakeSemanticModelFileService{}
	fileSvc.listFiles = func(params KnowledgeBaseCatalogFileListParams) (*KnowledgeBaseCatalogFileListResult, error) {
		switch params.Page {
		case 1:
			return &KnowledgeBaseCatalogFileListResult{
				Items: []KnowledgeBaseCatalogFileLeaf{
					{FileID: "file-1", FileName: "file-1.pdf", VolumeID: params.VolumeID},
					{FileID: "file-2", FileName: "file-2.pdf", VolumeID: params.VolumeID},
					{FileID: "file-3", FileName: "file-3.docx", VolumeID: params.VolumeID},
				},
				Total: kbSourceSelectionBatchSize + 1,
			}, nil
		case 2:
			return &KnowledgeBaseCatalogFileListResult{
				Items: []KnowledgeBaseCatalogFileLeaf{
					{FileID: "file-101", FileName: "file-101.docx", VolumeID: params.VolumeID},
				},
				Total: kbSourceSelectionBatchSize + 1,
			}, nil
		default:
			t.Fatalf("unexpected page %d", params.Page)
			return nil, nil
		}
	}

	svc := &semanticModelService{fileService: fileSvc}
	sources, err := svc.expandVolumeFileSelection(context.Background(), SemanticModelSourceSelectionRequest{
		Kind:            kbSelectionKindVolumeFiles,
		VolumeID:        42,
		AllSelected:     true,
		ExcludedFileIDs: []string{"file-2"},
		Filters: SemanticModelSourceSelectionFilters{
			FileName: "quarterly",
			FileExt:  []string{".PDF", "docx", "pdf", ""},
		},
	}, map[string]int64{"file-3": 42})
	if err != nil {
		t.Fatalf("expandVolumeFileSelection: %v", err)
	}
	if len(sources) != 2 || sources[0].FileID != "file-1" || sources[1].FileID != "file-101" {
		t.Fatalf("sources = %+v, want file-1 and file-101 after exclude/seen filtering", sources)
	}
	if len(fileSvc.listCalls) != 2 {
		t.Fatalf("list calls = %+v, want two paged calls", fileSvc.listCalls)
	}
	firstCall := fileSvc.listCalls[0]
	if firstCall.VolumeID != 42 || firstCall.Page != 1 || firstCall.PageSize != kbSourceSelectionBatchSize || firstCall.FileName != "quarterly" {
		t.Fatalf("first list call = %+v, want volume/page/search filters", firstCall)
	}
	if len(firstCall.FileExt) != 2 || firstCall.FileExt[0] != "pdf" || firstCall.FileExt[1] != "docx" {
		t.Fatalf("file_ext filters = %+v, want normalized unique pdf/docx", firstCall.FileExt)
	}
	if fileSvc.listCalls[1].Page != 2 {
		t.Fatalf("second list call = %+v, want page 2", fileSvc.listCalls[1])
	}
}

func TestSemanticModelServiceExpandVolumeFileSelectionAllowsSpreadsheetFiles(t *testing.T) {
	fileSvc := &fakeSemanticModelFileService{
		listFiles: func(params KnowledgeBaseCatalogFileListParams) (*KnowledgeBaseCatalogFileListResult, error) {
			return &KnowledgeBaseCatalogFileListResult{
				Items: []KnowledgeBaseCatalogFileLeaf{
					{FileID: "file-xls", FileName: "legacy.XLS", VolumeID: params.VolumeID},
					{FileID: "file-xlsx", FileName: "report.xlsx", VolumeID: params.VolumeID},
				},
				Total: 2,
			}, nil
		},
	}

	svc := &semanticModelService{fileService: fileSvc}
	sources, err := svc.expandVolumeFileSelection(context.Background(), SemanticModelSourceSelectionRequest{
		Kind:        kbSelectionKindVolumeFiles,
		VolumeID:    42,
		AllSelected: true,
	}, nil)
	if err != nil {
		t.Fatalf("expandVolumeFileSelection: %v", err)
	}
	if len(sources) != 2 || sources[0].FileID != "file-xls" || sources[1].FileID != "file-xlsx" {
		t.Fatalf("sources = %+v, want Catalog XLS and XLSX files", sources)
	}
}

func TestSemanticModelServiceExpandVolumeFileSelectionRejectsVolumeConflict(t *testing.T) {
	// Same file_id already claimed at volume 41 must fail when expanded from volume 52.
	fileSvc := &fakeSemanticModelFileService{
		listFiles: func(params KnowledgeBaseCatalogFileListParams) (*KnowledgeBaseCatalogFileListResult, error) {
			return &KnowledgeBaseCatalogFileListResult{
				Items: []KnowledgeBaseCatalogFileLeaf{
					{FileID: "shared-file", FileName: "shared.pdf", VolumeID: params.VolumeID},
				},
				Total: 1,
			}, nil
		},
	}
	svc := &semanticModelService{fileService: fileSvc}
	_, err := svc.expandVolumeFileSelection(context.Background(), SemanticModelSourceSelectionRequest{
		Kind:            kbSelectionKindVolumeFiles,
		VolumeID:        52,
		SelectedFileIDs: []string{"shared-file"},
	}, map[string]int64{"shared-file": 41})
	if !IsServiceError(err, ErrCodeBadRequest) {
		t.Fatalf("error = %v, want bad request", err)
	}
	if !i18n.IsKey(err, i18n.KeySessionSemanticModelCatalogFileVolumeConflict) {
		t.Fatalf("error key = %v, want catalog_file_volume_conflict", err)
	}

	// Same volume re-claim is idempotent skip, not an error.
	sources, err := svc.expandVolumeFileSelection(context.Background(), SemanticModelSourceSelectionRequest{
		Kind:            kbSelectionKindVolumeFiles,
		VolumeID:        41,
		SelectedFileIDs: []string{"shared-file"},
	}, map[string]int64{"shared-file": 41})
	if err != nil {
		t.Fatalf("same-volume expand: %v", err)
	}
	if len(sources) != 0 {
		t.Fatalf("same-volume sources = %+v, want empty skip", sources)
	}
}

func TestSemanticModelServiceExpandVolumeFileSelectionRejectsInvalidResults(t *testing.T) {
	tests := []struct {
		name      string
		selection SemanticModelSourceSelectionRequest
		listFiles func(KnowledgeBaseCatalogFileListParams) (*KnowledgeBaseCatalogFileListResult, error)
		wantErr   string
	}{
		{
			name: "all selected wrong volume",
			selection: SemanticModelSourceSelectionRequest{
				Kind:        kbSelectionKindVolumeFiles,
				VolumeID:    42,
				AllSelected: true,
			},
			listFiles: func(KnowledgeBaseCatalogFileListParams) (*KnowledgeBaseCatalogFileListResult, error) {
				return &KnowledgeBaseCatalogFileListResult{
					Items: []KnowledgeBaseCatalogFileLeaf{{FileID: "file-1", FileName: "file-1.pdf", VolumeID: 99}},
					Total: 1,
				}, nil
			},
			wantErr: "does not belong to volume 42",
		},
		{
			name: "selected file wrong volume",
			selection: SemanticModelSourceSelectionRequest{
				Kind:            kbSelectionKindVolumeFiles,
				VolumeID:        42,
				SelectedFileIDs: []string{"file-1"},
			},
			listFiles: func(KnowledgeBaseCatalogFileListParams) (*KnowledgeBaseCatalogFileListResult, error) {
				return &KnowledgeBaseCatalogFileListResult{
					Items: []KnowledgeBaseCatalogFileLeaf{{FileID: "file-1", FileName: "file-1.pdf", VolumeID: 99}},
					Total: 1,
				}, nil
			},
			wantErr: "does not belong to volume 42",
		},
		{
			name: "all selected empty result",
			selection: SemanticModelSourceSelectionRequest{
				Kind:        kbSelectionKindVolumeFiles,
				VolumeID:    42,
				AllSelected: true,
			},
			listFiles: func(KnowledgeBaseCatalogFileListParams) (*KnowledgeBaseCatalogFileListResult, error) {
				return &KnowledgeBaseCatalogFileListResult{}, nil
			},
			wantErr: "matched no files",
		},
		{
			name: "unsupported extension",
			selection: SemanticModelSourceSelectionRequest{
				Kind:        kbSelectionKindVolumeFiles,
				VolumeID:    42,
				AllSelected: true,
			},
			listFiles: func(params KnowledgeBaseCatalogFileListParams) (*KnowledgeBaseCatalogFileListResult, error) {
				return &KnowledgeBaseCatalogFileListResult{
					Items: []KnowledgeBaseCatalogFileLeaf{{FileID: "file-1", FileName: "archive.zip", VolumeID: params.VolumeID}},
					Total: 1,
				}, nil
			},
			wantErr: "unsupported knowledge base catalog file extension",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := &semanticModelService{fileService: &fakeSemanticModelFileService{listFiles: tc.listFiles}}
			_, err := svc.expandVolumeFileSelection(context.Background(), tc.selection, nil)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expandVolumeFileSelection error = %v, want containing %q", err, tc.wantErr)
			}
		})
	}
}

func TestSemanticModelServiceFindCatalogTableSourcePrefersSucceededDirectTable(t *testing.T) {
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}

	succeeded := KnowledgeBaseSourceRecord{
		SourceID:          "source-succeeded-table",
		ModelID:           77,
		CatalogID:         3,
		DatabaseID:        11,
		RawVolumeID:       12,
		ProcessedVolumeID: 13,
		SourceType:        kbSourceTypeCatalogTable,
		SourceTableID:     int64Ptr(1001),
		DisplayName:       stringPtr("orders"),
		DBName:            stringPtr("kb_docs"),
		TableName:         stringPtr("orders"),
		Status:            kbSourceStatusSucceeded,
		Enabled:           boolPtr(true),
	}
	tenantMock.ExpectQuery("ORDER BY CASE WHEN kbs\\.status = \\? AND kbs\\.source_table_id IS NOT NULL AND kbs\\.source_table_id > 0").
		WithArgs(int64(77), kbSourceTypeCatalogTable, int64(1001), int64(1001), kbSourceStatusSucceeded).
		WillReturnRows(knowledgeBaseSourceRecordRows(succeeded))

	ctx := ctxutil.WithTenantDB(context.Background(), tenantDB)
	svc := &semanticModelService{}
	record, found, err := svc.findCatalogTableSourceBySourceTableID(ctx, 77, 1001)
	if err != nil {
		t.Fatalf("findCatalogTableSourceBySourceTableID: %v", err)
	}
	if !found || record.SourceID != "source-succeeded-table" || record.Status != kbSourceStatusSucceeded {
		t.Fatalf("record = %+v, found = %v", record, found)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceAppendModelSourcesMergesExistingFieldsAndReturnsPersistedRows(t *testing.T) {
	var semanticGetCount int
	var sawInitialUpdate bool
	currentTables := []semanticModelTableSource{{
		DBName:     "analytics",
		TableNames: []string{"existing_orders"},
	}}
	currentFiles := map[string]any{
		"file_ids":                     []string{"existing-file"},
		"parents":                      []string{"volume-12"},
		"volume_ids":                   []string{"12"},
		"volumes":                      []semanticModelVolumeSource{{VolumeID: "12", Parents: []string{"root"}, Path: []string{"docs"}}},
		"vector_table":                 "existing_vector",
		"embedding_model":              "existing_embedding",
		"image_vector_table":           "existing_image_vector",
		"image_embedding_model":        "existing_image_embedding",
		"image_embedding_dimension":    768,
		"active_image_index_config_id": "image-index-config-1",
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case serveCatalogTableDetail(t, w, r, 1001, 3, 11, "catalog", "sales", "orders"):
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			requireSemanticModelExecutionHeaders(t, r)
			semanticGetCount++
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":          77,
				"name":        "kb_docs",
				"description": "docs",
				"tables":      currentTables,
				"files":       currentFiles,
			})
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			requireSemanticModelExecutionHeaders(t, r)
			var req struct {
				Name        string                     `json:"name"`
				Description string                     `json:"description"`
				Tables      []semanticModelTableSource `json:"tables"`
				Files       map[string]any             `json:"files"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode semantic update: %v", err)
			}
			if req.Name != "kb_docs" || req.Description != "docs" {
				t.Fatalf("semantic update identity = %+v", req)
			}
			if req.Files["vector_table"] != "existing_vector" || req.Files["embedding_model"] != "existing_embedding" || req.Files["image_vector_table"] != "existing_image_vector" || req.Files["image_embedding_model"] != "existing_image_embedding" || req.Files["active_image_index_config_id"] != "image-index-config-1" {
				t.Fatalf("semantic files fields were not preserved: %+v", req.Files)
			}
			fileIDs, ok := stringSliceFromAny(req.Files["file_ids"])
			if !ok || !sameStringSet(fileIDs, []string{"existing-file"}) {
				t.Fatalf("semantic files update = %+v", req.Files["file_ids"])
			}
			parents, parentsOK := stringSliceFromAny(req.Files["parents"])
			volumeIDs, volumeIDsOK := stringSliceFromAny(req.Files["volume_ids"])
			volumes, volumesOK := req.Files["volumes"].([]any)
			if !parentsOK || !sameStringSet(parents, []string{"volume-12"}) || !volumeIDsOK || !sameStringSet(volumeIDs, []string{"12"}) || !volumesOK || len(volumes) != 1 {
				t.Fatalf("semantic volume fields were not preserved: %+v", req.Files)
			}
			if len(req.Tables) == 2 && req.Tables[0].DBName == "analytics" && sameStringSet(req.Tables[0].TableNames, []string{"existing_orders"}) && req.Tables[1].DBName == "sales" && sameStringSet(req.Tables[1].TableNames, []string{"orders"}) {
				sawInitialUpdate = true
			} else {
				t.Fatalf("semantic tables update = %+v", req.Tables)
			}
			currentTables = req.Tables
			currentFiles = req.Files
			_ = json.NewEncoder(w).Encode(map[string]any{"updated": true})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/ws-1/files/source-file/download":
			requireSemanticModelExecutionHeaders(t, r)
			w.Header().Set("Content-Disposition", `attachment; filename="source.txt"`)
			_, _ = w.Write([]byte("source content"))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/workspaces/ws-1/files":
			requireSemanticModelExecutionHeaders(t, r)
			_ = json.NewEncoder(w).Encode(map[string]any{"file_id": "source-file"})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/workspaces/ws-1/volumes/12/files":
			t.Fatalf("unexpected AddFiles during append short request")
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/workspaces/ws-1/volumes/12/files/trigger":
			t.Fatalf("unexpected TriggerFiles during append short request")
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	dataDomainSvc := &fakeSemanticModelDataDomainService{databaseID: 11}
	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()
	workflowTemplateSvc := &fakeSemanticModelWorkflowTemplateService{}
	workflowSvc := &fakeSemanticModelWorkflowService{}
	svc := newSemanticModelTestServiceWithKnowledgeBaseRuntimeDependencies(t, server.URL, systemClient, dataDomainSvc, nil, nil, workflowTemplateSvc, workflowSvc)

	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	domainRows := sqlmock.NewRows([]string{
		"model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "ensure_status", "last_ensure_error", "last_checked_at",
	}).AddRow(int64(77), int64(3), int64(11), int64(12), int64(13), kbEnsureStatusReady, nil, int64(100))
	tenantMock.ExpectQuery("SELECT model_id, catalog_id, database_id, raw_volume_id, processed_volume_id, ensure_status, last_ensure_error, last_checked_at").
		WithArgs(int64(77)).
		WillReturnRows(domainRows)
	expectUpsertKnowledgeBaseRawVolume(tenantMock, 77, kbRawKindDocument, false)
	expectAppendSourcesTransactionLock(tenantMock, 77)
	expectCatalogFileSourceLookupMiss(tenantMock, 77, "source-file")
	tenantMock.ExpectExec("INSERT INTO knowledge_base_sources").
		WillReturnResult(sqlmock.NewResult(1, 1))
	expectKnowledgeBaseSourceJobRunUpsertMiss(tenantMock)
	tenantMock.ExpectExec("INSERT INTO knowledge_base_source_job_runs").
		WillReturnResult(sqlmock.NewResult(1, 1))
	expectKnowledgeBaseSourceJobRunUpsertMiss(tenantMock)
	tenantMock.ExpectExec("INSERT INTO knowledge_base_source_job_runs").
		WillReturnResult(sqlmock.NewResult(1, 1))
	expectCatalogTableSourceLookupMiss(tenantMock, 77, 1001)
	tenantMock.ExpectExec("INSERT INTO knowledge_base_sources").
		WillReturnResult(sqlmock.NewResult(2, 1))
	tenantMock.ExpectCommit()
	tenantMock.ExpectQuery("SELECT .*source_id.*FROM knowledge_base_sources").
		WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows([]string{
			"source_id", "model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "source_type",
			"source_file_id", "source_table_id", "kb_file_id", "kb_table_id", "display_name", "source_path",
			"db_name", "table_name", "status", "error", "enabled", "expires_at", "tags", "force_enabled_after_expiry", "segment_version_id", "index_version",
		}).
			AddRow(stableID("kb-source", int64(77), kbSourceTypeCatalogFile, "source-file"), int64(77), int64(3), int64(11), int64(12), int64(13), kbSourceTypeCatalogFile, "source-file", nil, "source-file", nil, "source-file", nil, nil, nil, kbSourceStatusPending, nil, nil, nil, nil, false, nil, nil).
			AddRow(stableID("kb-source", int64(77), kbSourceTypeCatalogTable, int64(1001)), int64(77), int64(3), int64(11), int64(0), int64(0), kbSourceTypeCatalogTable, nil, int64(1001), nil, nil, "orders", `["catalog","sales"]`, "sales", "orders", kbSourceStatusSucceeded, nil, true, nil, "[]", false, nil, nil))
	tenantMock.ExpectQuery("SELECT .*job_id.*source_id.*model_id.*job_type.*job_status.*idempotency_key").
		WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows([]string{
			"job_id", "source_id", "model_id", "job_type", "job_status", "idempotency_key", "operation_id",
			"workflow_execution_id", "runtime_actor_moi_user_id", "runtime_effective_role_id", "runtime_is_workspace_owner", "source_file_id", "kb_file_id", "source_table_id", "kb_table_id",
			"retry_count", "next_retry_at", "error", "created_at", "updated_at",
		}).
			AddRow(stableID("kb-job", stableID("kb-source", int64(77), kbSourceTypeCatalogFile, "source-file"), kbJobTypeCopy), stableID("kb-source", int64(77), kbSourceTypeCatalogFile, "source-file"), int64(77), kbJobTypeCopy, kbSourceJobPending, stableID("kb-job-key", stableID("kb-source", int64(77), kbSourceTypeCatalogFile, "source-file"), kbJobTypeCopy), nil, nil, nil, nil, false, "source-file", "source-file", nil, nil, int64(0), nil, nil, int64(100), int64(100)).
			AddRow(stableID("kb-job", stableID("kb-source", int64(77), kbSourceTypeCatalogFile, "source-file"), kbJobTypeRAGIngest), stableID("kb-source", int64(77), kbSourceTypeCatalogFile, "source-file"), int64(77), kbJobTypeRAGIngest, kbSourceJobPending, stableID("kb-job-key", stableID("kb-source", int64(77), kbSourceTypeCatalogFile, "source-file"), kbJobTypeRAGIngest), nil, nil, nil, nil, false, "source-file", "source-file", nil, nil, int64(0), nil, nil, int64(100), int64(100)))
	tenantMock.ExpectQuery("SELECT model_id, catalog_id, database_id, raw_volume_id, processed_volume_id, ensure_status, last_ensure_error, last_checked_at").
		WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows([]string{
			"model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "ensure_status", "last_ensure_error", "last_checked_at",
		}).AddRow(int64(77), int64(3), int64(11), int64(12), int64(13), kbEnsureStatusReady, nil, int64(100)))
	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithUserID(ctx, "user-1")
	ctx = withKnowledgeBaseCreatePrincipal(ctx)
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)

	resp, err := svc.AppendModelSources(ctx, AppendSemanticModelSourcesRequest{
		ModelID: 77,
		Sources: []CreateSemanticModelSourceRequest{
			{SourceType: kbSourceTypeCatalogFile, FileID: "source-file", VolumeID: 41},
			{SourceType: kbSourceTypeCatalogTable, TableID: 1001},
		},
	})
	if err != nil {
		t.Fatalf("AppendModelSources: %v", err)
	}
	if resp.DataDomain == nil || resp.DataDomain.EnsureStatus != kbEnsureStatusReady {
		t.Fatalf("data domain = %+v", resp.DataDomain)
	}
	if len(resp.Sources) != 2 || resp.Sources[0].ResourceID != "source-file" || resp.Sources[1].ResourceID != "1001" || resp.Sources[1].KBTableID != nil {
		t.Fatalf("sources = %+v", resp.Sources)
	}
	if len(resp.Jobs) != 2 {
		t.Fatalf("jobs = %+v", resp.Jobs)
	}
	if !sawInitialUpdate {
		t.Fatalf("semantic update was not called")
	}
	if semanticGetCount != 2 {
		t.Fatalf("semantic model get count = %d, want 2", semanticGetCount)
	}
	if len(dataDomainSvc.calls) != 0 {
		t.Fatalf("data domain service calls = %+v, want no table clone in append request", dataDomainSvc.calls)
	}
	assertDocumentAppendPreservesExistingWorkflow(t, workflowTemplateSvc, workflowSvc, "ws-1", 77)
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceCatalogFileSourceInsertUsesKBFileIdentity(t *testing.T) {
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}

	source := &KnowledgeBaseSourceRecord{
		SourceID:          stableID("kb-source", int64(77), kbSourceTypeCatalogFile, "kb-copy-new"),
		ModelID:           77,
		CatalogID:         3,
		DatabaseID:        11,
		RawVolumeID:       12,
		ProcessedVolumeID: 13,
		SourceType:        kbSourceTypeCatalogFile,
		SourceFileID:      stringPtr("source-file"),
		KBFileID:          stringPtr("kb-copy-new"),
		DisplayName:       stringPtr("source.txt"),
		Status:            kbSourceStatusPending,
	}
	tenantMock.ExpectExec("INSERT INTO knowledge_base_sources").
		WithArgs(source.SourceID, source.ModelID, source.CatalogID, source.DatabaseID, source.RawVolumeID, source.ProcessedVolumeID, source.SourceType,
			source.SourceFileID, source.SourceTableID, source.KBFileID, source.KBTableID, source.DisplayName, source.SourcePath,
			source.DBName, source.TableName, source.Status, source.Error, source.Enabled, source.ExpiresAt, source.Tags, source.ForceEnabled, source.SegmentVersionID, source.IndexVersion,
			"user-1", "user-1").
		WillReturnResult(sqlmock.NewResult(1, 1))

	ctx := ctxutil.WithTenantDB(context.Background(), tenantDB)
	svc := &semanticModelService{}
	if err := svc.insertCatalogFileSourceProcessing(ctx, source, "user-1"); err != nil {
		t.Fatalf("insertCatalogFileSourceProcessing: %v", err)
	}
	if stableID("kb-source", int64(77), kbSourceTypeCatalogFile, "kb-copy-new") == stableID("kb-source", int64(77), kbSourceTypeCatalogFile, "kb-copy-other") {
		t.Fatal("catalog file source identity must use kb_file_id")
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceAppendCatalogFileSourceKeepsCanonicalSourceFileID(t *testing.T) {
	tenantDB, tenantMock := newSemanticModelTenantDB(t)
	tenantMock.ExpectQuery(`(?s)SELECT source_file_id\s+FROM knowledge_base_sources\s+WHERE kb_file_id = \?\s+AND source_type IN \(\?, \?\)\s+AND source_file_id IS NOT NULL\s+AND source_file_id <> ''\s+ORDER BY created_at ASC, source_id ASC\s+LIMIT 1`).
		WithArgs("a2", kbSourceTypeCatalogFile, kbSourceTypeLocalFile).
		WillReturnRows(sqlmock.NewRows([]string{"source_file_id"}).AddRow("a"))
	// Exact (source_file_id, selection volume): wrong client file_name is ignored.
	expectCatalogFileMetadataBatchAtVolume(tenantMock, 41, []string{"a"}, "canonical.pdf")
	tenantMock.ExpectExec("INSERT INTO knowledge_base_sources").
		WillReturnResult(sqlmock.NewResult(1, 1))
	tenantMock.ExpectExec("INSERT INTO knowledge_base_source_job_runs").
		WillReturnResult(sqlmock.NewResult(1, 1))
	tenantMock.ExpectExec("INSERT INTO knowledge_base_source_job_runs").
		WillReturnResult(sqlmock.NewResult(2, 1))

	ctx := withKnowledgeBaseCreatePrincipal(ctxutil.WithTenantDB(context.Background(), tenantDB))
	source, jobs, _, err := (&semanticModelService{}).appendCatalogFileSourceIntent(
		ctx,
		"ws-1",
		77,
		&KnowledgeBaseDataDomain{CatalogID: 3, DatabaseID: 11, RawVolumeID: 12, ProcessedVolumeID: 13},
		CreateSemanticModelSourceRequest{SourceType: kbSourceTypeCatalogFile, FileID: "a2", FileName: "client-wrong-name.pdf", VolumeID: 41},
		"user-1",
		false,
	)
	if err != nil {
		t.Fatalf("appendCatalogFileSourceIntent: %v", err)
	}
	if source == nil || ptrValue(source.SourceFileID) != "a" || ptrValue(source.KBFileID) != "a2" {
		t.Fatalf("source = %+v, want source_file_id=a and kb_file_id=a2", source)
	}
	if source.RawVolumeID != 41 {
		t.Fatalf("raw_volume_id = %d, want selection volume 41 (not domain raw 12)", source.RawVolumeID)
	}
	if ptrValue(source.DisplayName) != "canonical.pdf" {
		t.Fatalf("display_name = %q, want catalog metadata name (not client-wrong-name.pdf)", ptrValue(source.DisplayName))
	}
	if source.SourceID != stableID("kb-source", int64(77), kbSourceTypeCatalogFile, "a2") {
		t.Fatalf("source id = %q, want kb file identity", source.SourceID)
	}
	if len(jobs) != 2 {
		t.Fatalf("jobs = %+v, want copy and rag jobs", jobs)
	}
	for _, job := range jobs {
		if ptrValue(job.SourceFileID) != "a" || ptrValue(job.KBFileID) != "a2" {
			t.Fatalf("job = %+v, want source_file_id=a and kb_file_id=a2", job)
		}
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceAppendCatalogFileSourceUsesSelectedOlderVolume(t *testing.T) {
	// Same file present on volume 41 (older) and 52 (newer). Caller-selected volume 41 must win.
	tenantDB, tenantMock := newSemanticModelTenantDB(t)
	tenantMock.ExpectQuery(`(?s)SELECT source_file_id\s+FROM knowledge_base_sources\s+WHERE kb_file_id = \?\s+AND source_type IN \(\?, \?\)\s+AND source_file_id IS NOT NULL\s+AND source_file_id <> ''\s+ORDER BY created_at ASC, source_id ASC\s+LIMIT 1`).
		WithArgs("shared-file", kbSourceTypeCatalogFile, kbSourceTypeLocalFile).
		WillReturnRows(sqlmock.NewRows([]string{"source_file_id"}))
	// Metadata may list multiple volumes for the same file_id; selected volume_id 41 must be used,
	// not a newest-row guess (which would prefer 52).
	tenantMock.ExpectQuery("SELECT f.file_id, COALESCE.*WHERE f.file_id IN").
		WithArgs("shared-file").
		WillReturnRows(sqlmock.NewRows([]string{"file_id", "catalog_id", "database_id", "volume_id", "size", "updated_at", "catalog_name", "database_name", "volume_name", "file_path", "file_name"}).
			AddRow("shared-file", int64(21), int64(32), int64(41), int64(4096), int64(100), "default", "source_knowledge_base", "raw_document", "", "document.pdf").
			AddRow("shared-file", int64(21), int64(31), int64(52), int64(4096), int64(200), "default", "importing_knowledge_base", "raw_document", "", "document.pdf"))
	tenantMock.ExpectExec("INSERT INTO knowledge_base_sources").
		WillReturnResult(sqlmock.NewResult(1, 1))
	tenantMock.ExpectExec("INSERT INTO knowledge_base_source_job_runs").
		WillReturnResult(sqlmock.NewResult(1, 1))
	tenantMock.ExpectExec("INSERT INTO knowledge_base_source_job_runs").
		WillReturnResult(sqlmock.NewResult(2, 1))

	ctx := withKnowledgeBaseCreatePrincipal(ctxutil.WithTenantDB(context.Background(), tenantDB))
	source, _, _, err := (&semanticModelService{}).appendCatalogFileSourceIntent(
		ctx,
		"ws-1",
		77,
		&KnowledgeBaseDataDomain{CatalogID: 3, DatabaseID: 11, RawVolumeID: 12, ProcessedVolumeID: 13},
		CreateSemanticModelSourceRequest{SourceType: kbSourceTypeCatalogFile, FileID: "shared-file", VolumeID: 41},
		"user-1",
		false,
	)
	if err != nil {
		t.Fatalf("appendCatalogFileSourceIntent: %v", err)
	}
	if source == nil || source.RawVolumeID != 41 {
		t.Fatalf("raw_volume_id = %v, want selected older volume 41 (not newest 52)", source)
	}
	if ptrValue(source.DisplayName) != "document.pdf" {
		t.Fatalf("display_name = %q", ptrValue(source.DisplayName))
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceAppendCatalogFileSourceRejectsMissingVolumeID(t *testing.T) {
	// Direct catalog_file sources must supply volume_id; no volume_files fallback.
	_, _, _, err := (&semanticModelService{}).appendCatalogFileSourceIntent(
		context.Background(),
		"ws-1",
		77,
		&KnowledgeBaseDataDomain{CatalogID: 3, DatabaseID: 11, RawVolumeID: 12, ProcessedVolumeID: 13},
		CreateSemanticModelSourceRequest{SourceType: kbSourceTypeCatalogFile, FileID: "shared-file"},
		"user-1",
		false,
	)
	if err == nil || !strings.Contains(err.Error(), "volume_id") {
		t.Fatalf("error = %v, want volume_id required", err)
	}
}

func TestSemanticModelServiceAppendCatalogFileSourceRejectsVolumeConflict(t *testing.T) {
	// Existing source at volume 41; append same file at volume 52 must fail closed
	// instead of silently reusing the first row.
	tenantDB, tenantMock := newSemanticModelTenantDB(t)
	existing := KnowledgeBaseSourceRecord{
		SourceID:     stableID("kb-source", int64(77), kbSourceTypeCatalogFile, "shared-file"),
		ModelID:      77,
		CatalogID:    3,
		DatabaseID:   11,
		RawVolumeID:  41,
		SourceType:   kbSourceTypeCatalogFile,
		SourceFileID: stringPtr("shared-file"),
		KBFileID:     stringPtr("shared-file"),
		Status:       kbSourceStatusSucceeded,
	}
	expectCatalogFileSourceLookupHit(tenantMock, 77, "shared-file", existing)

	_, _, _, err := (&semanticModelService{}).appendCatalogFileSourceIntent(
		ctxutil.WithTenantDB(context.Background(), tenantDB),
		"ws-1",
		77,
		&KnowledgeBaseDataDomain{CatalogID: 3, DatabaseID: 11, RawVolumeID: 12, ProcessedVolumeID: 13},
		CreateSemanticModelSourceRequest{SourceType: kbSourceTypeCatalogFile, FileID: "shared-file", VolumeID: 52},
		"user-1",
		true,
	)
	if !IsServiceError(err, ErrCodeBadRequest) {
		t.Fatalf("error = %v, want bad request", err)
	}
	if !i18n.IsKey(err, i18n.KeySessionSemanticModelCatalogFileVolumeConflict) {
		t.Fatalf("error key = %v, want catalog_file_volume_conflict", err)
	}
	zhCtx := semanticModelServiceTestContext(i18n.LocaleZhCN.String())
	msg, ok := i18n.Message(zhCtx, err)
	if !ok || msg != "该文件已加入当前知识库，请勿从其他数据卷重复添加" {
		t.Fatalf("zh message = %q, ok = %v", msg, ok)
	}
	enCtx := semanticModelServiceTestContext(i18n.LocaleEnUS.String())
	msg, ok = i18n.Message(enCtx, err)
	if !ok || msg != "This file is already in the knowledge base. Do not add it again from another volume" {
		t.Fatalf("en message = %q, ok = %v", msg, ok)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceAppendCatalogFileSourceReusesSameVolume(t *testing.T) {
	// Same file + same volume remains idempotent reuse.
	tenantDB, tenantMock := newSemanticModelTenantDB(t)
	existing := KnowledgeBaseSourceRecord{
		SourceID:     stableID("kb-source", int64(77), kbSourceTypeCatalogFile, "shared-file"),
		ModelID:      77,
		CatalogID:    3,
		DatabaseID:   11,
		RawVolumeID:  41,
		SourceType:   kbSourceTypeCatalogFile,
		SourceFileID: stringPtr("shared-file"),
		KBFileID:     stringPtr("shared-file"),
		Status:       kbSourceStatusSucceeded,
	}
	expectCatalogFileSourceLookupHit(tenantMock, 77, "shared-file", existing)

	source, jobs, fileID, err := (&semanticModelService{}).appendCatalogFileSourceIntent(
		ctxutil.WithTenantDB(context.Background(), tenantDB),
		"ws-1",
		77,
		&KnowledgeBaseDataDomain{CatalogID: 3, DatabaseID: 11, RawVolumeID: 12, ProcessedVolumeID: 13},
		CreateSemanticModelSourceRequest{SourceType: kbSourceTypeCatalogFile, FileID: "shared-file", VolumeID: 41},
		"user-1",
		true,
	)
	if err != nil {
		t.Fatalf("appendCatalogFileSourceIntent: %v", err)
	}
	if source == nil || source.SourceID != existing.SourceID || source.RawVolumeID != 41 {
		t.Fatalf("source = %+v, want existing volume 41 source", source)
	}
	if len(jobs) != 0 || fileID != "shared-file" {
		t.Fatalf("jobs=%+v fileID=%q, want reuse without new jobs", jobs, fileID)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestKnowledgeBaseSplitFileIDListAndDetailUseSourceFileAtVolume(t *testing.T) {
	// Persisted source_file_id=a, kb_file_id=a2, raw_volume_id=41. Only a@41 exists.
	// List and detail must both succeed with the same display name (source id).
	tenantDB, tenantMock := newSemanticModelTenantDB(t)
	expectCatalogFileMetadataBatchAtVolume(tenantMock, 41, []string{"a"}, "canonical.pdf")
	ctx := ctxutil.WithTenantDB(context.Background(), tenantDB)
	record := KnowledgeBaseSourceRecord{
		SourceID:     "source-split",
		SourceType:   kbSourceTypeCatalogFile,
		RawVolumeID:  41,
		SourceFileID: stringPtr("a"),
		KBFileID:     stringPtr("a2"),
		DisplayName:  stringPtr("stale-client-name.pdf"),
		Status:       kbSourceStatusSucceeded,
	}
	got, err := (&semanticModelService{}).enrichKnowledgeBaseSourceRecordsMetadata(ctx, nil, "ws-1", []KnowledgeBaseSourceRecord{record})
	if err != nil {
		t.Fatalf("list enrich: %v", err)
	}
	if len(got) != 1 || got[0].Status == kbSourceStatusFailed {
		t.Fatalf("list record = %+v, want success", got)
	}
	if ptrValue(got[0].DisplayName) != "canonical.pdf" {
		t.Fatalf("list display_name = %q, want canonical.pdf", ptrValue(got[0].DisplayName))
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("list tenant sql expectations: %v", err)
	}

	tenantDB2, tenantMock2 := newSemanticModelTenantDB(t)
	expectCatalogFileMetadataBatchAtVolume(tenantMock2, 41, []string{"a"}, "canonical.pdf")
	detail := record
	if err := (&semanticModelService{}).enrichKnowledgeBaseFileSourceDisplayName(ctxutil.WithTenantDB(context.Background(), tenantDB2), &detail); err != nil {
		t.Fatalf("detail enrich: %v", err)
	}
	if detail.Status == kbSourceStatusFailed {
		t.Fatalf("detail marked failed: %v", ptrValue(detail.Error))
	}
	if ptrValue(detail.DisplayName) != "canonical.pdf" {
		t.Fatalf("detail display_name = %q, want same as list", ptrValue(detail.DisplayName))
	}
	if err := tenantMock2.ExpectationsWereMet(); err != nil {
		t.Fatalf("detail tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceReconcileTriggersDeferredLocalRAGAfterImportFinished(t *testing.T) {
	var triggeredFiles []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/workspaces/ws-1/volumes/12/files/trigger":
			var req struct {
				FileIDs []string `json:"file_ids"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode trigger files: %v", err)
			}
			triggeredFiles = append(triggeredFiles, req.FileIDs...)
			_ = json.NewEncoder(w).Encode(map[string]any{"triggered": 1})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()
	client, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer client.Close()

	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	tenantMock.ExpectQuery("SELECT job_status, operation_id").
		WithArgs(int64(77), "source-file-1", kbJobTypeLoad).
		WillReturnRows(sqlmock.NewRows([]string{"job_status", "operation_id"}).AddRow(kbSourceJobRunning, "import_task:import-task-1"))
	tenantMock.ExpectQuery("SELECT .* FROM `import_task` WHERE id = \\? ORDER BY `import_task`.`id` LIMIT \\?").
		WithArgs("import-task-1", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "status", "task_meta"}).AddRow("import-task-1", model.ImportTaskStatusFinished, "{}"))
	tenantMock.ExpectQuery("SELECT .* FROM `import_task_run` WHERE import_task_id = \\? ORDER BY created_at DESC, id DESC,`import_task_run`.`id` LIMIT \\?").
		WithArgs("import-task-1", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "workflow_execution_id", "status", "error_message"}))
	tenantMock.ExpectExec("UPDATE knowledge_base_source_job_runs").
		WithArgs(kbSourceJobRunning, "workflow_trigger:"+knowledgeBaseWorkflowID("ws-1", 77), "source-file", "kb-file", "user-1", "job-rag-1", int64(77), "source-file-1", kbJobTypeRAGIngest, kbSourceJobPending, kbSourceJobQueued, "workflow_trigger:"+knowledgeBaseWorkflowID("ws-1", 77)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	tenantMock.ExpectExec("UPDATE knowledge_base_source_job_runs").WillReturnResult(sqlmock.NewResult(1, 1))

	workflowSvc := &fakeSemanticModelWorkflowService{fileExecutions: map[string]*moi.FileExecutionsResponse{
		"kb-file": {Executions: []moi.FileExecutionSummary{}, Total: 0},
	}}
	authorizer := &fakeSemanticModelActionAuthorizer{roleOut: "role-runtime"}
	svc := &semanticModelService{workflowService: workflowSvc, actionAuthorizer: authorizer}
	ctx := ctxutil.WithTenantDB(context.Background(), tenantDB)
	record := KnowledgeBaseSourceRecord{
		SourceID:     "source-file-1",
		ModelID:      77,
		RawVolumeID:  12,
		SourceType:   kbSourceTypeLocalFile,
		SourceFileID: stringPtr("source-file"),
		KBFileID:     stringPtr("kb-file"),
		DisplayName:  stringPtr("document.pdf"),
		CreatedBy:    stringPtr("bff-principal"),
	}
	job := KnowledgeBaseSourceJobRun{
		JobID:                  "job-rag-1",
		SourceID:               "source-file-1",
		ModelID:                77,
		JobType:                kbJobTypeRAGIngest,
		JobStatus:              kbSourceJobPending,
		IdempotencyKey:         "idem-rag-1",
		SourceFileID:           stringPtr("source-file"),
		KBFileID:               stringPtr("kb-file"),
		RuntimeActorMOIUserID:  stringPtr("moi-runtime-user"),
		RuntimeEffectiveRoleID: stringPtr("role-runtime"),
	}

	if err := svc.reconcileRAGIngestSourceJob(ctx, client, "ws-1", record, &job, "user-1"); err != nil {
		t.Fatalf("reconcileRAGIngestSourceJob: %v", err)
	}
	if len(triggeredFiles) != 0 || len(workflowSvc.runs) != 1 {
		t.Fatalf("triggered files = %+v, workflow runs = %+v", triggeredFiles, workflowSvc.runs)
	}
	if job.JobStatus != kbSourceJobRunning || job.OperationID == nil || *job.OperationID != "workflow_trigger:"+knowledgeBaseWorkflowID("ws-1", 77) {
		t.Fatalf("job after reconcile = %+v", job)
	}
	if len(workflowSvc.listFileExecutionCalls) != 1 || workflowSvc.listFileExecutionCalls[0] != "kb-file" {
		t.Fatalf("ListFileExecutions calls = %+v", workflowSvc.listFileExecutionCalls)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceReconcileRetriesClaimedLocalRAGWhenExecutionMissing(t *testing.T) {
	var triggeredFiles []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/workspaces/ws-1/volumes/12/files/trigger":
			var req struct {
				FileIDs []string `json:"file_ids"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode trigger files: %v", err)
			}
			triggeredFiles = append(triggeredFiles, req.FileIDs...)
			_ = json.NewEncoder(w).Encode(map[string]any{"triggered": 1})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()
	client, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer client.Close()

	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	operationID := "workflow_trigger:" + knowledgeBaseWorkflowID("ws-1", 77)
	tenantMock.ExpectQuery("SELECT job_status, operation_id").
		WithArgs(int64(77), "source-file-1", kbJobTypeLoad).
		WillReturnRows(sqlmock.NewRows([]string{"job_status", "operation_id"}).AddRow(kbSourceJobSucceeded, kbJobOpLocalFileBindPrefix+"kb-file"))
	tenantMock.ExpectExec("UPDATE knowledge_base_source_job_runs").
		WithArgs(kbSourceJobRunning, operationID, "source-file", "kb-file", "user-1", "job-rag-1", int64(77), "source-file-1", kbJobTypeRAGIngest, kbSourceJobPending, kbSourceJobQueued, operationID).
		WillReturnResult(sqlmock.NewResult(1, 1))
	tenantMock.ExpectExec("UPDATE knowledge_base_source_job_runs").WillReturnResult(sqlmock.NewResult(1, 1))

	workflowSvc := &fakeSemanticModelWorkflowService{fileExecutions: map[string]*moi.FileExecutionsResponse{
		"kb-file": {Executions: []moi.FileExecutionSummary{}, Total: 0},
	}}
	authorizer := &fakeSemanticModelActionAuthorizer{roleOut: "role-runtime"}
	svc := &semanticModelService{workflowService: workflowSvc, actionAuthorizer: authorizer}
	ctx := ctxutil.WithTenantDB(context.Background(), tenantDB)
	record := KnowledgeBaseSourceRecord{
		SourceID:     "source-file-1",
		ModelID:      77,
		RawVolumeID:  12,
		SourceType:   kbSourceTypeLocalFile,
		SourceFileID: stringPtr("source-file"),
		KBFileID:     stringPtr("kb-file"),
		DisplayName:  stringPtr("document.pdf"),
		CreatedBy:    stringPtr("bff-principal"),
	}
	job := KnowledgeBaseSourceJobRun{
		JobID:                  "job-rag-1",
		SourceID:               "source-file-1",
		ModelID:                77,
		JobType:                kbJobTypeRAGIngest,
		JobStatus:              kbSourceJobQueued,
		IdempotencyKey:         "idem-rag-1",
		OperationID:            stringPtr(operationID),
		SourceFileID:           stringPtr("source-file"),
		KBFileID:               stringPtr("kb-file"),
		RuntimeActorMOIUserID:  stringPtr("moi-runtime-user"),
		RuntimeEffectiveRoleID: stringPtr("role-runtime"),
	}

	if err := svc.reconcileRAGIngestSourceJob(ctx, client, "ws-1", record, &job, "user-1"); err != nil {
		t.Fatalf("reconcileRAGIngestSourceJob: %v", err)
	}
	if len(triggeredFiles) != 0 || len(workflowSvc.runs) != 1 {
		t.Fatalf("triggered files = %+v, workflow runs = %+v", triggeredFiles, workflowSvc.runs)
	}
	if job.JobStatus != kbSourceJobRunning || job.OperationID == nil || *job.OperationID != operationID {
		t.Fatalf("job after reconcile = %+v", job)
	}
	if len(workflowSvc.listFileExecutionCalls) != 1 || workflowSvc.listFileExecutionCalls[0] != "kb-file" {
		t.Fatalf("ListFileExecutions calls = %+v", workflowSvc.listFileExecutionCalls)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceReconcileDoesNotTriggerRAGWhenExecutionKnown(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("unexpected trigger request for job with workflow execution id: %s %s", r.Method, r.URL.String())
	}))
	defer server.Close()
	client, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer client.Close()

	workflowSvc := &fakeSemanticModelWorkflowService{fileExecutions: map[string]*moi.FileExecutionsResponse{
		"kb-file": {Executions: []moi.FileExecutionSummary{}, Total: 0},
	}}
	svc := &semanticModelService{workflowService: workflowSvc}
	record := KnowledgeBaseSourceRecord{
		SourceID:     "source-file-1",
		ModelID:      77,
		RawVolumeID:  12,
		SourceType:   kbSourceTypeLocalFile,
		SourceFileID: stringPtr("source-file"),
		KBFileID:     stringPtr("kb-file"),
	}
	operationID := "workflow_trigger:" + knowledgeBaseWorkflowID("ws-1", 77)
	job := KnowledgeBaseSourceJobRun{
		JobID:               "job-rag-1",
		SourceID:            "source-file-1",
		ModelID:             77,
		JobType:             kbJobTypeRAGIngest,
		JobStatus:           kbSourceJobQueued,
		IdempotencyKey:      "idem-rag-1",
		OperationID:         stringPtr(operationID),
		WorkflowExecutionID: stringPtr("exec-known"),
		SourceFileID:        stringPtr("source-file"),
		KBFileID:            stringPtr("kb-file"),
	}

	if err := svc.reconcileRAGIngestSourceJob(context.Background(), client, "ws-1", record, &job, "user-1"); err != nil {
		t.Fatalf("reconcileRAGIngestSourceJob: %v", err)
	}
	if job.OperationID == nil || *job.OperationID != operationID || job.WorkflowExecutionID == nil || *job.WorkflowExecutionID != "exec-known" {
		t.Fatalf("job after reconcile = %+v", job)
	}
	if len(workflowSvc.listFileExecutionCalls) != 1 || workflowSvc.listFileExecutionCalls[0] != "kb-file" {
		t.Fatalf("ListFileExecutions calls = %+v", workflowSvc.listFileExecutionCalls)
	}
}

func TestSemanticModelServiceReconcileTriggersKBRAGWhenCompletedExecutionHasNoTargetRows(t *testing.T) {
	var triggeredFiles []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":          77,
				"name":        "kb_docs",
				"description": "docs",
				"tables":      []any{},
				"files": map[string]any{
					"file_ids":        []string{"source-file"},
					"vector_table":    "kb_text_idx",
					"embedding_model": "embed-model",
				},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/workspaces/ws-1/volumes/12/files/trigger":
			var req struct {
				FileIDs []string `json:"file_ids"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode trigger files: %v", err)
			}
			triggeredFiles = append(triggeredFiles, req.FileIDs...)
			_ = json.NewEncoder(w).Encode(map[string]any{"triggered": 1})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()
	client, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer client.Close()

	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	expectVectorTableSchemaColumns(tenantMock, "kb_text_idx", "vecf32(3)")
	tenantMock.ExpectQuery("SELECT MAX\\(index_version\\) FROM `kb_text_idx` WHERE file_id = \\?").
		WithArgs("source-file").
		WillReturnRows(sqlmock.NewRows([]string{"MAX(index_version)"}).AddRow(nil))
	tenantMock.ExpectQuery("SELECT job_status").
		WithArgs(int64(77), "source-file-1", kbJobTypeCopy).
		WillReturnRows(sqlmock.NewRows([]string{"job_status"}).AddRow(kbSourceJobSucceeded))
	tenantMock.ExpectExec("UPDATE knowledge_base_source_job_runs").
		WithArgs(kbSourceJobRunning, "workflow_trigger:"+knowledgeBaseWorkflowID("ws-1", 77), "source-file", "source-file", "user-1", "job-rag-1", int64(77), "source-file-1", kbJobTypeRAGIngest, kbSourceJobPending, kbSourceJobQueued, "workflow_trigger:"+knowledgeBaseWorkflowID("ws-1", 77)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	tenantMock.ExpectExec("UPDATE knowledge_base_source_job_runs").WillReturnResult(sqlmock.NewResult(1, 1))

	workflowSvc := &fakeSemanticModelWorkflowService{fileExecutions: map[string]*moi.FileExecutionsResponse{
		"source-file": {Executions: []moi.FileExecutionSummary{{ExecutionID: "exec-old-completed", Status: "succeeded"}}, Total: 1},
	}}
	authorizer := &fakeSemanticModelActionAuthorizer{roleOut: "role-runtime"}
	svc := &semanticModelService{
		workflowService:  workflowSvc,
		actionAuthorizer: authorizer,
		volumeResolver:   &fakeSemanticModelCanonicalVolumeResolver{},
	}
	ctx := ctxutil.WithTenantDB(context.Background(), tenantDB)
	record := KnowledgeBaseSourceRecord{
		SourceID:     "source-file-1",
		ModelID:      77,
		RawVolumeID:  12,
		SourceType:   kbSourceTypeCatalogFile,
		SourceFileID: stringPtr("source-file"),
		KBFileID:     stringPtr("source-file"),
		DisplayName:  stringPtr("document.pdf"),
		CreatedBy:    stringPtr("bff-principal"),
	}
	job := KnowledgeBaseSourceJobRun{
		JobID:                  "job-rag-1",
		SourceID:               "source-file-1",
		ModelID:                77,
		JobType:                kbJobTypeRAGIngest,
		JobStatus:              kbSourceJobQueued,
		IdempotencyKey:         "idem-rag-1",
		OperationID:            stringPtr("workflow_trigger:" + knowledgeBaseWorkflowID("ws-1", 77)),
		SourceFileID:           stringPtr("source-file"),
		KBFileID:               stringPtr("source-file"),
		RuntimeActorMOIUserID:  stringPtr("moi-runtime-user"),
		RuntimeEffectiveRoleID: stringPtr("role-runtime"),
	}

	if err := svc.reconcileRAGIngestSourceJob(ctx, client, "ws-1", record, &job, "user-1"); err != nil {
		t.Fatalf("reconcileRAGIngestSourceJob: %v", err)
	}
	if len(triggeredFiles) != 0 || len(workflowSvc.runs) != 1 {
		t.Fatalf("triggered files = %+v, workflow runs = %+v", triggeredFiles, workflowSvc.runs)
	}
	if job.JobStatus != kbSourceJobRunning || ptrValue(job.WorkflowExecutionID) != "exec-workflow" {
		t.Fatalf("job after reconcile = %+v", job)
	}
	if len(workflowSvc.listFileExecutionCalls) != 1 || workflowSvc.listFileExecutionCalls[0] != "source-file" {
		t.Fatalf("ListFileExecutions calls = %+v", workflowSvc.listFileExecutionCalls)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceReconcileTriggersKBRAGWhenCompletedExecutionVectorModelMismatches(t *testing.T) {
	var triggeredFiles []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":          77,
				"name":        "kb_docs",
				"description": "docs",
				"tables":      []any{},
				"files": map[string]any{
					"file_ids":        []string{"source-file"},
					"vector_table":    "kb_text_idx",
					"embedding_model": "embed-model",
				},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/workspaces/ws-1/volumes/12/files/trigger":
			var req struct {
				FileIDs []string `json:"file_ids"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode trigger files: %v", err)
			}
			triggeredFiles = append(triggeredFiles, req.FileIDs...)
			_ = json.NewEncoder(w).Encode(map[string]any{"triggered": 1})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()
	client, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer client.Close()

	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	expectVectorTableSchemaColumns(tenantMock, "kb_text_idx", "vecf32(3)")
	tenantMock.ExpectQuery("SELECT MAX\\(index_version\\) FROM `kb_text_idx` WHERE file_id = \\?").
		WithArgs("source-file").
		WillReturnRows(sqlmock.NewRows([]string{"MAX(index_version)"}).AddRow(int64(2)))
	tenantMock.ExpectQuery("SELECT id, content, meta, level, chunk_index, index_version").
		WithArgs("source-file", int64(2)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "content", "meta", "level", "chunk_index", "index_version"}).
			AddRow("row-v2-0", "workflow chunk", `{"chunk_id":"chunk-v2-0","embedding_model":"other-model"}`, "chunk", int64(0), int64(2)))
	tenantMock.ExpectQuery("SELECT job_status").
		WithArgs(int64(77), "source-file-1", kbJobTypeCopy).
		WillReturnRows(sqlmock.NewRows([]string{"job_status"}).AddRow(kbSourceJobSucceeded))
	tenantMock.ExpectExec("UPDATE knowledge_base_source_job_runs").
		WithArgs(kbSourceJobRunning, "workflow_trigger:"+knowledgeBaseWorkflowID("ws-1", 77), "source-file", "source-file", "user-1", "job-rag-1", int64(77), "source-file-1", kbJobTypeRAGIngest, kbSourceJobPending, kbSourceJobQueued, "workflow_trigger:"+knowledgeBaseWorkflowID("ws-1", 77)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	tenantMock.ExpectExec("UPDATE knowledge_base_source_job_runs").WillReturnResult(sqlmock.NewResult(1, 1))

	workflowSvc := &fakeSemanticModelWorkflowService{fileExecutions: map[string]*moi.FileExecutionsResponse{
		"source-file": {Executions: []moi.FileExecutionSummary{{ExecutionID: "exec-old-completed", Status: "succeeded"}}, Total: 1},
	}}
	authorizer := &fakeSemanticModelActionAuthorizer{roleOut: "role-runtime"}
	svc := &semanticModelService{
		workflowService:  workflowSvc,
		actionAuthorizer: authorizer,
		volumeResolver:   &fakeSemanticModelCanonicalVolumeResolver{},
	}
	ctx := ctxutil.WithTenantDB(context.Background(), tenantDB)
	record := KnowledgeBaseSourceRecord{
		SourceID:     "source-file-1",
		ModelID:      77,
		RawVolumeID:  12,
		SourceType:   kbSourceTypeCatalogFile,
		SourceFileID: stringPtr("source-file"),
		KBFileID:     stringPtr("source-file"),
		DisplayName:  stringPtr("document.pdf"),
		CreatedBy:    stringPtr("bff-principal"),
	}
	job := KnowledgeBaseSourceJobRun{
		JobID:                  "job-rag-1",
		SourceID:               "source-file-1",
		ModelID:                77,
		JobType:                kbJobTypeRAGIngest,
		JobStatus:              kbSourceJobQueued,
		IdempotencyKey:         "idem-rag-1",
		OperationID:            stringPtr("workflow_trigger:" + knowledgeBaseWorkflowID("ws-1", 77)),
		SourceFileID:           stringPtr("source-file"),
		KBFileID:               stringPtr("source-file"),
		RuntimeActorMOIUserID:  stringPtr("moi-runtime-user"),
		RuntimeEffectiveRoleID: stringPtr("role-runtime"),
	}

	if err := svc.reconcileRAGIngestSourceJob(ctx, client, "ws-1", record, &job, "user-1"); err != nil {
		t.Fatalf("reconcileRAGIngestSourceJob: %v", err)
	}
	if len(triggeredFiles) != 0 || len(workflowSvc.runs) != 1 {
		t.Fatalf("triggered files = %+v, workflow runs = %+v", triggeredFiles, workflowSvc.runs)
	}
	if job.JobStatus != kbSourceJobRunning || ptrValue(job.WorkflowExecutionID) != "exec-workflow" {
		t.Fatalf("job after reconcile = %+v", job)
	}
	if len(workflowSvc.listFileExecutionCalls) != 1 || workflowSvc.listFileExecutionCalls[0] != "source-file" {
		t.Fatalf("ListFileExecutions calls = %+v", workflowSvc.listFileExecutionCalls)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceReconcileReusesDeferredCatalogFileVectorsBeforeTrigger(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":          77,
				"name":        "kb_docs",
				"description": "docs",
				"tables":      []any{},
				"files": map[string]any{
					"file_ids":        []string{"source-file"},
					"vector_table":    "kb_text_idx",
					"embedding_model": "embed-model",
				},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/workspaces/ws-1/volumes/12/files/trigger":
			t.Fatalf("unexpected default workflow trigger after vector reuse: %s %s", r.Method, r.URL.String())
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()
	client, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer client.Close()

	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	sourceID := stableID("kb-source", int64(77), kbSourceTypeCatalogFile, "source-file")
	tenantMock.ExpectQuery("SELECT job_status").
		WithArgs(int64(77), sourceID, kbJobTypeCopy).
		WillReturnRows(sqlmock.NewRows([]string{"job_status"}).AddRow(kbSourceJobSucceeded))
	reuseOperation := expectCatalogFileTextVectorReuseSucceeded(tenantMock, 77, sourceID, "source-file", "external_text_idx", "kb_text_idx", "embed-model", false)
	tenantMock.ExpectExec("UPDATE knowledge_base_source_job_runs").
		WithArgs("moi-runtime-user", "role-runtime", false, kbSourceJobSucceeded, reuseOperation, nil, "source-file", "source-file", nil, nil, int64(0), nil, nil, "user-1", "job-rag-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	workflowSvc := &fakeSemanticModelWorkflowService{fileExecutions: map[string]*moi.FileExecutionsResponse{
		"source-file": {Executions: []moi.FileExecutionSummary{{ExecutionID: "exec-running", Status: "running"}}, Total: 1},
	}}
	authorizer := &fakeSemanticModelActionAuthorizer{roleOut: "role-runtime"}
	svc := &semanticModelService{workflowService: workflowSvc, actionAuthorizer: authorizer}
	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)
	record := KnowledgeBaseSourceRecord{
		SourceID:     sourceID,
		ModelID:      77,
		RawVolumeID:  12,
		SourceType:   kbSourceTypeCatalogFile,
		SourceFileID: stringPtr("source-file"),
		KBFileID:     stringPtr("source-file"),
	}
	job := KnowledgeBaseSourceJobRun{
		JobID:                  "job-rag-1",
		SourceID:               sourceID,
		ModelID:                77,
		JobType:                kbJobTypeRAGIngest,
		JobStatus:              kbSourceJobPending,
		IdempotencyKey:         "idem-rag-1",
		SourceFileID:           stringPtr("source-file"),
		KBFileID:               stringPtr("source-file"),
		RuntimeActorMOIUserID:  stringPtr("moi-runtime-user"),
		RuntimeEffectiveRoleID: stringPtr("role-runtime"),
	}

	if err := svc.reconcileRAGIngestSourceJob(ctx, client, "ws-1", record, &job, "user-1"); err != nil {
		t.Fatalf("reconcileRAGIngestSourceJob: %v", err)
	}
	if job.JobStatus != kbSourceJobSucceeded || job.OperationID == nil || *job.OperationID != reuseOperation {
		t.Fatalf("job after reconcile = %+v, want vector reuse success %q", job, reuseOperation)
	}
	if len(workflowSvc.listFileExecutionCalls) != 0 {
		t.Fatalf("ListFileExecutions calls = %+v", workflowSvc.listFileExecutionCalls)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceReconcileTriggersDeferredCatalogRAGWhenVectorReuseMissing(t *testing.T) {
	var triggeredFiles []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":          77,
				"name":        "kb_docs",
				"description": "docs",
				"tables":      []any{},
				"files": map[string]any{
					"file_ids":        []string{"source-file"},
					"vector_table":    "kb_text_idx",
					"embedding_model": "embed-model",
				},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/workspaces/ws-1/volumes/12/files/trigger":
			var req struct {
				FileIDs []string `json:"file_ids"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode trigger files: %v", err)
			}
			triggeredFiles = append(triggeredFiles, req.FileIDs...)
			_ = json.NewEncoder(w).Encode(map[string]any{"triggered": 1})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()
	client, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer client.Close()

	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	sourceID := stableID("kb-source", int64(77), kbSourceTypeCatalogFile, "source-file")
	tenantMock.ExpectQuery("SELECT job_status").
		WithArgs(int64(77), sourceID, kbJobTypeCopy).
		WillReturnRows(sqlmock.NewRows([]string{"job_status"}).AddRow(kbSourceJobSucceeded))
	expectCatalogFileVectorReuseCandidatesEmpty(tenantMock, "source-file")
	tenantMock.ExpectExec("UPDATE knowledge_base_source_job_runs").
		WithArgs(kbSourceJobRunning, "workflow_trigger:"+knowledgeBaseWorkflowID("ws-1", 77), "source-file", "source-file", "user-1", "job-rag-1", int64(77), sourceID, kbJobTypeRAGIngest, kbSourceJobPending, kbSourceJobQueued, "workflow_trigger:"+knowledgeBaseWorkflowID("ws-1", 77)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	tenantMock.ExpectExec("UPDATE knowledge_base_source_job_runs").WillReturnResult(sqlmock.NewResult(1, 1))

	workflowSvc := &fakeSemanticModelWorkflowService{fileExecutions: map[string]*moi.FileExecutionsResponse{
		"source-file": {Executions: []moi.FileExecutionSummary{{ExecutionID: "exec-running", Status: "running"}}, Total: 1},
	}}
	authorizer := &fakeSemanticModelActionAuthorizer{roleOut: "role-caller"}
	volumeResolver := &fakeSemanticModelCanonicalVolumeResolver{}
	svc := &semanticModelService{
		workflowService:  workflowSvc,
		actionAuthorizer: authorizer,
		volumeResolver:   volumeResolver,
	}
	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithMoiUserID(ctx, "moi-user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)
	ctx = ctxutil.WithCoreIAMRequest(ctx, ctxutil.CoreIAMRequestContext{
		RequestID: "req-catalog-rag", TraceID: "tr-catalog-rag", VerifiedEffectiveRoleID: "role-caller",
	})
	record := KnowledgeBaseSourceRecord{
		SourceID:     sourceID,
		ModelID:      77,
		RawVolumeID:  12,
		SourceType:   kbSourceTypeCatalogFile,
		SourceFileID: stringPtr("source-file"),
		KBFileID:     stringPtr("source-file"),
		DisplayName:  stringPtr("document.pdf"),
		CreatedBy:    stringPtr("bff-principal"),
	}
	job := KnowledgeBaseSourceJobRun{
		JobID:                  "job-rag-1",
		SourceID:               sourceID,
		ModelID:                77,
		JobType:                kbJobTypeRAGIngest,
		JobStatus:              kbSourceJobPending,
		IdempotencyKey:         "idem-rag-1",
		SourceFileID:           stringPtr("source-file"),
		KBFileID:               stringPtr("source-file"),
		RuntimeActorMOIUserID:  stringPtr("moi-runtime-user"),
		RuntimeEffectiveRoleID: stringPtr("role-runtime"),
	}

	if err := svc.reconcileRAGIngestSourceJob(ctx, client, "ws-1", record, &job, "user-1"); err != nil {
		t.Fatalf("reconcileRAGIngestSourceJob: %v", err)
	}
	if len(triggeredFiles) != 0 || len(workflowSvc.runs) != 1 {
		t.Fatalf("triggered files = %+v, workflow runs = %+v", triggeredFiles, workflowSvc.runs)
	}
	if job.JobStatus != kbSourceJobRunning || job.OperationID == nil || *job.OperationID != "workflow_trigger:"+knowledgeBaseWorkflowID("ws-1", 77) {
		t.Fatalf("job after reconcile = %+v", job)
	}
	if len(workflowSvc.listFileExecutionCalls) != 0 {
		t.Fatalf("ListFileExecutions calls = %+v", workflowSvc.listFileExecutionCalls)
	}
	// Order: semantic_model.use once (reuse attempt + dispatch reuse same kbCtx), then volume.read.
	if len(authorizer.calls) != 2 {
		t.Fatalf("reauth calls = %+v, want use, volume.read", authorizer.calls)
	}
	if authorizer.calls[0].actionID != "semantic_model.use" || authorizer.calls[0].resourceID != "77" {
		t.Fatalf("call[0] = %+v, want semantic_model.use 77", authorizer.calls[0])
	}
	if authorizer.calls[1].actionID != "volume.read" || authorizer.calls[1].resourceID != "12" {
		t.Fatalf("call[1] = %+v, want volume.read 12", authorizer.calls[1])
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestClaimAndTriggerDeferredCatalogRAGFailsClosedWhenVolumeReadRevoked(t *testing.T) {
	// Queued catalog_file RAG must reauthorize volume.read before claim/dispatch.
	// A revoked or disabled Effective Role fails closed and never runs workflow.
	workflowSvc := &fakeSemanticModelWorkflowService{
		runResult: &KnowledgeBaseWorkflowRunResult{ExecutionID: "must-not-run"},
	}
	authorizer := &fakeSemanticModelActionAuthorizer{
		roleOut:     "role-create-time",
		errByAction: map[string]error{"volume.read": iampep.ErrCoreDecisionDeny},
	}
	// Identity resolver: RawVolumeID is already the root in this fixture.
	volumeResolver := &fakeSemanticModelCanonicalVolumeResolver{}
	svc := &semanticModelService{
		workflowService:  workflowSvc,
		actionAuthorizer: authorizer,
		volumeResolver:   volumeResolver,
	}

	tenantDB, tenantMock := newSemanticModelTenantDB(t)
	// failDeferredRAGIngestSourceJob marks source then job failed.
	tenantMock.ExpectExec("UPDATE knowledge_base_sources").
		WillReturnResult(sqlmock.NewResult(0, 1))
	tenantMock.ExpectExec("UPDATE knowledge_base_source_job_runs").
		WillReturnResult(sqlmock.NewResult(0, 1))

	ctx := ctxutil.WithTenantDB(context.Background(), tenantDB)
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiUserID(ctx, "moi-user-revoked")
	ctx = ctxutil.WithCoreIAMRequest(ctx, ctxutil.CoreIAMRequestContext{
		RequestID: "req-revoked", TraceID: "tr-revoked", VerifiedEffectiveRoleID: "role-disabled",
	})
	record := KnowledgeBaseSourceRecord{
		SourceID: "source-catalog", ModelID: 77, RawVolumeID: 41,
		SourceType:   kbSourceTypeCatalogFile,
		SourceFileID: stringPtr("catalog-file"), KBFileID: stringPtr("catalog-file"),
		DisplayName: stringPtr("policy.pdf"),
	}
	job := &KnowledgeBaseSourceJobRun{
		JobID: "job-rag-revoked", SourceID: record.SourceID, ModelID: 77,
		JobType: kbJobTypeRAGIngest, JobStatus: kbSourceJobPending,
		SourceFileID: record.SourceFileID, KBFileID: record.KBFileID,
		RuntimeActorMOIUserID:  stringPtr("moi-user-create-time"),
		RuntimeEffectiveRoleID: stringPtr("role-create-time"),
	}

	err := svc.claimAndTriggerDeferredRAGIngestSourceJob(ctx, nil, "ws-1", record, job, "bff-principal")
	if err == nil || !errors.Is(err, errKnowledgeBaseSourceJobFailed) {
		t.Fatalf("error = %v, want errKnowledgeBaseSourceJobFailed", err)
	}
	if !strings.Contains(err.Error(), "volume.read") {
		t.Fatalf("error = %v, want volume.read reauth failure", err)
	}
	if len(workflowSvc.runs) != 0 {
		t.Fatalf("workflow must not run after volume.read deny, runs = %+v", workflowSvc.runs)
	}
	if len(authorizer.calls) != 2 {
		t.Fatalf("reauth calls = %+v, want use then volume.read", authorizer.calls)
	}
	if authorizer.calls[0].actionID != "semantic_model.use" || authorizer.calls[0].resourceID != "77" {
		t.Fatalf("first reauth = %+v, want semantic_model.use model 77", authorizer.calls[0])
	}
	if authorizer.calls[1].actionID != "volume.read" || authorizer.calls[1].resourceID != "41" {
		t.Fatalf("second reauth = %+v, want volume.read 41", authorizer.calls[1])
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestClaimAndTriggerDeferredCatalogRAGAuthorizesCanonicalRootVolume(t *testing.T) {
	// RawVolumeID may be a child volume; IAM volume.read must hit the canonical root.
	// Workflow still uses RawVolumeID for file I/O (not asserted here — gate only).
	workflowSvc := &fakeSemanticModelWorkflowService{
		runResult: &KnowledgeBaseWorkflowRunResult{ExecutionID: "exec-root-auth"},
	}
	authorizer := &fakeSemanticModelActionAuthorizer{roleOut: "role-ordinary"}
	volumeResolver := &fakeSemanticModelCanonicalVolumeResolver{
		roots: map[int64]int64{41: 9001},
	}
	svc := &semanticModelService{
		workflowService:  workflowSvc,
		actionAuthorizer: authorizer,
		volumeResolver:   volumeResolver,
	}

	tenantDB, tenantMock := newSemanticModelTenantDB(t)
	// claimDeferredRAGIngestSourceJob then post-dispatch upsert of execution id.
	tenantMock.ExpectExec("UPDATE knowledge_base_source_job_runs").
		WillReturnResult(sqlmock.NewResult(0, 1))
	tenantMock.ExpectExec("UPDATE knowledge_base_source_job_runs").
		WillReturnResult(sqlmock.NewResult(0, 1))

	ctx := ctxutil.WithTenantDB(context.Background(), tenantDB)
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	record := KnowledgeBaseSourceRecord{
		SourceID: "source-catalog", ModelID: 77, RawVolumeID: 41,
		SourceType:   kbSourceTypeCatalogFile,
		SourceFileID: stringPtr("catalog-file"), KBFileID: stringPtr("catalog-file"),
		DisplayName: stringPtr("policy.pdf"),
	}
	job := &KnowledgeBaseSourceJobRun{
		JobID: "job-rag-root", SourceID: record.SourceID, ModelID: 77,
		JobType: kbJobTypeRAGIngest, JobStatus: kbSourceJobPending,
		SourceFileID: record.SourceFileID, KBFileID: record.KBFileID,
		RuntimeActorMOIUserID:  stringPtr("moi-user-ordinary"),
		RuntimeEffectiveRoleID: stringPtr("role-ordinary"),
	}

	if err := svc.claimAndTriggerDeferredRAGIngestSourceJob(ctx, nil, "ws-1", record, job, "bff-principal"); err != nil {
		t.Fatalf("claimAndTriggerDeferredRAGIngestSourceJob: %v", err)
	}
	if len(volumeResolver.calls) != 1 || volumeResolver.calls[0].volumeID != 41 {
		t.Fatalf("volume resolve calls = %+v, want raw volume 41 once", volumeResolver.calls)
	}
	if len(authorizer.calls) != 2 {
		t.Fatalf("reauth calls = %+v, want use then volume.read", authorizer.calls)
	}
	if authorizer.calls[0].actionID != "semantic_model.use" || authorizer.calls[0].resourceID != "77" {
		t.Fatalf("first reauth = %+v, want semantic_model.use 77", authorizer.calls[0])
	}
	if authorizer.calls[1].actionID != "volume.read" || authorizer.calls[1].resourceID != "9001" {
		t.Fatalf("second reauth = %+v, want volume.read 9001 (canonical root)", authorizer.calls[1])
	}
	if len(workflowSvc.runs) != 1 {
		t.Fatalf("workflow runs = %+v, want one dispatch after root volume.read", workflowSvc.runs)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestClaimAndTriggerDeferredCatalogRAGFailsClosedWithoutAuthorizer(t *testing.T) {
	// Production always wires the authorizer; missing wiring must not skip the gate.
	workflowSvc := &fakeSemanticModelWorkflowService{
		runResult: &KnowledgeBaseWorkflowRunResult{ExecutionID: "must-not-run"},
	}
	svc := &semanticModelService{workflowService: workflowSvc}
	tenantDB, tenantMock := newSemanticModelTenantDB(t)
	tenantMock.ExpectExec("UPDATE knowledge_base_sources").WillReturnResult(sqlmock.NewResult(0, 1))
	tenantMock.ExpectExec("UPDATE knowledge_base_source_job_runs").WillReturnResult(sqlmock.NewResult(0, 1))
	ctx := ctxutil.WithTenantDB(context.Background(), tenantDB)
	record := KnowledgeBaseSourceRecord{
		SourceID: "source-catalog", ModelID: 77, RawVolumeID: 41,
		SourceType: kbSourceTypeCatalogFile, SourceFileID: stringPtr("f"), KBFileID: stringPtr("f"),
	}
	job := &KnowledgeBaseSourceJobRun{
		JobID: "job-rag", SourceID: record.SourceID, ModelID: 77,
		JobType: kbJobTypeRAGIngest, JobStatus: kbSourceJobQueued,
		SourceFileID: record.SourceFileID, KBFileID: record.KBFileID,
		RuntimeActorMOIUserID:  stringPtr("moi-user-create-time"),
		RuntimeEffectiveRoleID: stringPtr("role-create-time"),
	}
	err := svc.claimAndTriggerDeferredRAGIngestSourceJob(ctx, nil, "ws-1", record, job, "actor")
	if err == nil || !errors.Is(err, errKnowledgeBaseSourceJobFailed) {
		t.Fatalf("error = %v, want fail-closed without authorizer", err)
	}
	if !strings.Contains(err.Error(), "semantic_model.use reauthorization is unavailable") {
		t.Fatalf("error = %v, want unavailable authorizer", err)
	}
	if len(workflowSvc.runs) != 0 {
		t.Fatalf("workflow runs = %+v, want none", workflowSvc.runs)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceReconcileSkipsDeferredLocalRAGWhenAlreadyClaimed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("unexpected trigger request after another reconcile claimed the job: %s %s", r.Method, r.URL.String())
	}))
	defer server.Close()
	client, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer client.Close()

	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	tenantMock.ExpectQuery("SELECT job_status, operation_id").
		WithArgs(int64(77), "source-file-1", kbJobTypeLoad).
		WillReturnRows(sqlmock.NewRows([]string{"job_status", "operation_id"}).AddRow(kbSourceJobRunning, "import_task:import-task-1"))
	tenantMock.ExpectQuery("SELECT .* FROM `import_task` WHERE id = \\? ORDER BY `import_task`.`id` LIMIT \\?").
		WithArgs("import-task-1", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "status", "task_meta"}).AddRow("import-task-1", model.ImportTaskStatusFinished, "{}"))
	tenantMock.ExpectQuery("SELECT .* FROM `import_task_run` WHERE import_task_id = \\? ORDER BY created_at DESC, id DESC,`import_task_run`.`id` LIMIT \\?").
		WithArgs("import-task-1", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "workflow_execution_id", "status", "error_message"}))
	tenantMock.ExpectExec("UPDATE knowledge_base_source_job_runs").
		WithArgs(kbSourceJobRunning, "workflow_trigger:"+knowledgeBaseWorkflowID("ws-1", 77), "source-file", "kb-file", "user-1", "job-rag-1", int64(77), "source-file-1", kbJobTypeRAGIngest, kbSourceJobPending, kbSourceJobQueued, "workflow_trigger:"+knowledgeBaseWorkflowID("ws-1", 77)).
		WillReturnResult(sqlmock.NewResult(0, 0))

	workflowSvc := &fakeSemanticModelWorkflowService{fileExecutions: map[string]*moi.FileExecutionsResponse{
		"kb-file": {Executions: []moi.FileExecutionSummary{}, Total: 0},
	}}
	authorizer := &fakeSemanticModelActionAuthorizer{roleOut: "role-runtime"}
	svc := &semanticModelService{workflowService: workflowSvc, actionAuthorizer: authorizer}
	ctx := ctxutil.WithTenantDB(context.Background(), tenantDB)
	record := KnowledgeBaseSourceRecord{
		SourceID:     "source-file-1",
		ModelID:      77,
		RawVolumeID:  12,
		SourceType:   kbSourceTypeLocalFile,
		SourceFileID: stringPtr("source-file"),
		KBFileID:     stringPtr("kb-file"),
		DisplayName:  stringPtr("document.pdf"),
	}
	job := KnowledgeBaseSourceJobRun{
		JobID:                  "job-rag-1",
		SourceID:               "source-file-1",
		ModelID:                77,
		JobType:                kbJobTypeRAGIngest,
		JobStatus:              kbSourceJobPending,
		IdempotencyKey:         "idem-rag-1",
		SourceFileID:           stringPtr("source-file"),
		KBFileID:               stringPtr("kb-file"),
		RuntimeActorMOIUserID:  stringPtr("moi-runtime-user"),
		RuntimeEffectiveRoleID: stringPtr("role-runtime"),
	}

	if err := svc.reconcileRAGIngestSourceJob(ctx, client, "ws-1", record, &job, "user-1"); err != nil {
		t.Fatalf("reconcileRAGIngestSourceJob: %v", err)
	}
	if job.JobStatus != kbSourceJobPending || job.OperationID != nil {
		t.Fatalf("unclaimed job should not be mutated: %+v", job)
	}
	if len(workflowSvc.runs) != 0 {
		t.Fatalf("unclaimed job must not run a workflow: %+v", workflowSvc.runs)
	}
	if len(workflowSvc.listFileExecutionCalls) != 1 || workflowSvc.listFileExecutionCalls[0] != "kb-file" {
		t.Fatalf("ListFileExecutions calls = %+v", workflowSvc.listFileExecutionCalls)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceReconcileDoesNotTriggerDeferredLocalRAGBeforeImportFinished(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("unexpected request before import finished: %s %s", r.Method, r.URL.String())
	}))
	defer server.Close()
	client, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer client.Close()

	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	tenantMock.ExpectQuery("SELECT job_status, operation_id").
		WithArgs(int64(77), "source-file-1", kbJobTypeLoad).
		WillReturnRows(sqlmock.NewRows([]string{"job_status", "operation_id"}).AddRow(kbSourceJobRunning, "import_task:import-task-1"))
	tenantMock.ExpectQuery("SELECT .* FROM `import_task` WHERE id = \\? ORDER BY `import_task`.`id` LIMIT \\?").
		WithArgs("import-task-1", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "status", "task_meta"}).AddRow("import-task-1", model.ImportTaskStatusCreated, "{}"))
	tenantMock.ExpectQuery("SELECT .* FROM `import_task_run` WHERE import_task_id = \\? ORDER BY created_at DESC, id DESC,`import_task_run`.`id` LIMIT \\?").
		WithArgs("import-task-1", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "workflow_execution_id", "status", "error_message"}))

	workflowSvc := &fakeSemanticModelWorkflowService{fileExecutions: map[string]*moi.FileExecutionsResponse{
		"kb-file": {Executions: []moi.FileExecutionSummary{}, Total: 0},
	}}
	svc := &semanticModelService{workflowService: workflowSvc}
	ctx := ctxutil.WithTenantDB(context.Background(), tenantDB)
	record := KnowledgeBaseSourceRecord{
		SourceID:     "source-file-1",
		ModelID:      77,
		RawVolumeID:  12,
		SourceType:   kbSourceTypeLocalFile,
		SourceFileID: stringPtr("source-file"),
		KBFileID:     stringPtr("kb-file"),
	}
	job := KnowledgeBaseSourceJobRun{
		JobID:          "job-rag-1",
		SourceID:       "source-file-1",
		ModelID:        77,
		JobType:        kbJobTypeRAGIngest,
		JobStatus:      kbSourceJobPending,
		IdempotencyKey: "idem-rag-1",
		SourceFileID:   stringPtr("source-file"),
		KBFileID:       stringPtr("kb-file"),
	}

	if err := svc.reconcileRAGIngestSourceJob(ctx, client, "ws-1", record, &job, "user-1"); err != nil {
		t.Fatalf("reconcileRAGIngestSourceJob: %v", err)
	}
	if job.OperationID != nil || job.JobStatus != kbSourceJobPending {
		t.Fatalf("job should stay pending before import finishes: %+v", job)
	}
	if len(workflowSvc.listFileExecutionCalls) != 1 || workflowSvc.listFileExecutionCalls[0] != "kb-file" {
		t.Fatalf("ListFileExecutions calls = %+v", workflowSvc.listFileExecutionCalls)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceRunPendingSourceJobsPublishesExternalWorkflowSegmentVersionFromLatestFileExecution(t *testing.T) {
	// Reconcile locates the latest execution through the bound kb_file_id, then
	// imports that execution's rows by the canonical source_file_id.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			requireSemanticModelExecutionHeaders(t, r)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":          77,
				"name":        "kb_docs",
				"description": "docs",
				"tables":      []semanticModelTableSource{},
				"files": map[string]any{
					"file_ids":        []string{"kb-old-file", "kb-new-file"},
					"vector_table":    "kb_text_idx",
					"embedding_model": "embed-model",
				},
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()
	workflowID := "extra-workflow"
	workflowSvc := &fakeSemanticModelWorkflowService{fileExecutions: map[string]*moi.FileExecutionsResponse{
		"kb-new-file": {
			Executions: []moi.FileExecutionSummary{
				{ExecutionID: "exec-rag-new", WorkflowID: workflowID, Status: "succeeded"},
			},
			Total: 1,
		},
	}}
	svc := newSemanticModelTestServiceWithKnowledgeBaseRuntimeDependencies(t, server.URL, systemClient, nil, nil, nil, nil, workflowSvc)

	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}

	expectPendingSourceJobRunsWithPendingEmpty(tenantMock, 77, kbJobTypeCopy)
	tenantMock.ExpectQuery("SELECT .*job_id.*source_id.*model_id.*job_type.*job_status.*idempotency_key").
		WithArgs(int64(77), kbJobTypeTableClone, kbSourceJobPending, kbSourceJobQueued, kbSourceJobRunning, kbSourceJobReconcileBatchSize).
		WillReturnRows(sqlmock.NewRows([]string{
			"job_id", "source_id", "model_id", "job_type", "job_status", "idempotency_key", "operation_id",
			"workflow_execution_id", "runtime_actor_moi_user_id", "runtime_effective_role_id", "runtime_is_workspace_owner", "source_file_id", "kb_file_id", "source_table_id", "kb_table_id",
			"retry_count", "next_retry_at", "error", "created_at", "updated_at",
		}))
	tenantMock.ExpectQuery("SELECT .*job_id.*source_id.*model_id.*job_type.*job_status.*idempotency_key").
		WithArgs(int64(77), kbJobTypeLoad, kbSourceJobQueued, kbSourceJobRunning, kbSourceJobSucceeded, kbSourceStatusRemoved, kbSourceTypeCatalogTable, kbSourceTypeLocalFile, kbRawKindStructured, kbSourceJobReconcileBatchSize).
		WillReturnRows(sqlmock.NewRows([]string{
			"job_id", "source_id", "model_id", "job_type", "job_status", "idempotency_key", "operation_id",
			"workflow_execution_id", "runtime_actor_moi_user_id", "runtime_effective_role_id", "runtime_is_workspace_owner", "source_file_id", "kb_file_id", "source_table_id", "kb_table_id",
			"retry_count", "next_retry_at", "error", "created_at", "updated_at",
		}))
	tenantMock.ExpectQuery("SELECT .*job_id.*source_id.*model_id.*job_type.*job_status.*idempotency_key").
		WithArgs(int64(77), kbJobTypeRAGIngest, kbSourceJobPending, kbSourceJobQueued, kbSourceJobRunning, kbSourceJobSucceeded, kbSourceJobFailed, kbSourceJobReconcileBatchSize).
		WillReturnRows(sqlmock.NewRows([]string{
			"job_id", "source_id", "model_id", "job_type", "job_status", "idempotency_key", "operation_id",
			"workflow_execution_id", "runtime_actor_moi_user_id", "runtime_effective_role_id", "runtime_is_workspace_owner", "source_file_id", "kb_file_id", "source_table_id", "kb_table_id",
			"retry_count", "next_retry_at", "error", "created_at", "updated_at",
		}).AddRow("job-rag-1", "source-file-1", int64(77), kbJobTypeRAGIngest, kbSourceJobSucceeded, "idem-rag-1", "workflow_trigger:"+knowledgeBaseWorkflowID("ws-1", 77), nil, nil, nil, false, "source-file", "kb-new-file", nil, nil, int64(0), nil, nil, int64(100), int64(101)))
	tenantMock.ExpectQuery("SELECT .*source_id.*FROM knowledge_base_sources").
		WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows([]string{
			"source_id", "model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "source_type",
			"source_file_id", "source_table_id", "kb_file_id", "kb_table_id", "display_name", "source_path",
			"db_name", "table_name", "status", "error", "enabled", "expires_at", "tags", "force_enabled_after_expiry", "segment_version_id", "index_version",
		}).AddRow("source-file-1", int64(77), int64(3), int64(11), int64(12), int64(13), kbSourceTypeCatalogFile, "source-file", nil, "kb-old-file", nil, "doc.pdf", nil, nil, nil, kbSourceStatusSucceeded, nil, true, nil, `["tag-a"]`, false, "seg-v1", int64(1)))
	expectVectorTableSchemaColumns(tenantMock, "kb_text_idx", "vecf32(3)")
	tenantMock.ExpectQuery("SELECT MAX\\(index_version\\) FROM `kb_text_idx` WHERE file_id = \\?").
		WithArgs("source-file").
		WillReturnRows(sqlmock.NewRows([]string{"MAX(index_version)"}).AddRow(int64(2)))
	tenantMock.ExpectQuery("SELECT id, content, meta, level, chunk_index, index_version").
		WithArgs("source-file", int64(2)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "content", "meta", "level", "chunk_index", "index_version"}).
			AddRow("row-v2-0", "workflow chunk", `{"chunk_id":"chunk-v2-0"}`, "chunk", int64(0), int64(2)))
	tenantMock.ExpectBegin()
	tenantMock.ExpectExec("INSERT INTO knowledge_base_segment_versions").
		WillReturnResult(sqlmock.NewResult(1, 1))
	tenantMock.ExpectExec("INSERT INTO [`\"]?knowledge_base_segments[`\"]?").
		WillReturnResult(sqlmock.NewResult(1, 1))
	tenantMock.ExpectExec("INSERT INTO [`\"]?knowledge_base_chunk_recall_stats[`\"]?").
		WillReturnResult(sqlmock.NewResult(1, 1))
	tenantMock.ExpectExec("UPDATE knowledge_base_sources").
		WillReturnResult(sqlmock.NewResult(1, 1))
	tenantMock.ExpectExec("UPDATE knowledge_base_source_job_runs").
		WillReturnResult(sqlmock.NewResult(1, 1))
	tenantMock.ExpectCommit()

	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithUserID(ctx, "user-1")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)

	if err := svc.RunPendingKnowledgeBaseSourceJobs(ctx, RunPendingKnowledgeBaseSourceJobsParams{ModelID: 77}); err != nil {
		t.Fatalf("RunPendingKnowledgeBaseSourceJobs: %v", err)
	}
	if len(workflowSvc.listFileExecutionCalls) != 1 || workflowSvc.listFileExecutionCalls[0] != "kb-new-file" {
		t.Fatalf("ListFileExecutions calls = %+v", workflowSvc.listFileExecutionCalls)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceRunPendingSourceJobsTriggersRAGWhenVectorTableNotReady(t *testing.T) {
	var triggeredFiles []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			requireSemanticModelExecutionHeaders(t, r)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":          77,
				"name":        "kb_docs",
				"description": "docs",
				"tables":      []semanticModelTableSource{},
				"files": map[string]any{
					"file_ids":        []string{"kb-file"},
					"vector_table":    "kb_text_idx",
					"embedding_model": "embed-model",
				},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/workspaces/ws-1/volumes/12/files/trigger":
			var req struct {
				FileIDs []string `json:"file_ids"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode trigger files: %v", err)
			}
			triggeredFiles = append(triggeredFiles, req.FileIDs...)
			_ = json.NewEncoder(w).Encode(map[string]any{"triggered": 1})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()
	workflowSvc := &fakeSemanticModelWorkflowService{fileExecutions: map[string]*moi.FileExecutionsResponse{
		"kb-file": {
			Executions: []moi.FileExecutionSummary{
				{ExecutionID: "exec-rag-1", WorkflowID: "workflow-rag-1", Status: "succeeded"},
			},
			Total: 1,
		},
	}}
	svc := newSemanticModelTestServiceWithKnowledgeBaseRuntimeDependencies(t, server.URL, systemClient, nil, nil, nil, nil, workflowSvc)

	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}

	expectPendingSourceJobRunsWithPendingEmpty(tenantMock, 77, kbJobTypeCopy)
	expectPendingSourceJobRunsWithPendingEmpty(tenantMock, 77, kbJobTypeTableClone)
	expectPendingSourceJobRunsEmpty(tenantMock, 77, kbJobTypeLoad)
	tenantMock.ExpectQuery("SELECT .*job_id.*source_id.*model_id.*job_type.*job_status.*idempotency_key").
		WithArgs(int64(77), kbJobTypeRAGIngest, kbSourceJobPending, kbSourceJobQueued, kbSourceJobRunning, kbSourceJobSucceeded, kbSourceJobFailed, kbSourceJobReconcileBatchSize).
		WillReturnRows(knowledgeBaseSourceJobRunRows().
			AddRow("job-rag-1", "source-file-1", int64(77), kbJobTypeRAGIngest, kbSourceJobPending, "idem-rag-1", "workflow_trigger:"+knowledgeBaseWorkflowID("ws-1", 77), nil, "moi-runtime-user", "role-runtime", false, "source-file", "kb-file", nil, nil, int64(0), nil, nil, int64(100), int64(101)))
	tenantMock.ExpectQuery("SELECT .*source_id.*FROM knowledge_base_sources").
		WithArgs(int64(77)).
		WillReturnRows(knowledgeBaseSourceRecordRows(KnowledgeBaseSourceRecord{
			SourceID:          "source-file-1",
			ModelID:           77,
			CatalogID:         3,
			DatabaseID:        11,
			RawVolumeID:       12,
			ProcessedVolumeID: 13,
			SourceType:        kbSourceTypeCatalogFile,
			SourceFileID:      stringPtr("source-file"),
			KBFileID:          stringPtr("kb-file"),
			DisplayName:       stringPtr("doc.pdf"),
			Status:            kbSourceStatusPending,
			Enabled:           boolPtr(true),
			CreatedBy:         stringPtr("bff-principal"),
		}))
	tenantMock.ExpectQuery("SELECT COLUMN_NAME, COLUMN_TYPE FROM information_schema\\.COLUMNS").
		WithArgs("kb_text_idx").
		WillReturnRows(sqlmock.NewRows([]string{"COLUMN_NAME", "COLUMN_TYPE"}))
	tenantMock.ExpectQuery("SELECT job_status").
		WithArgs(int64(77), "source-file-1", kbJobTypeCopy).
		WillReturnRows(sqlmock.NewRows([]string{"job_status"}).AddRow(kbSourceJobSucceeded))
	tenantMock.ExpectExec("UPDATE knowledge_base_source_job_runs").
		WithArgs(kbSourceJobRunning, "workflow_trigger:"+knowledgeBaseWorkflowID("ws-1", 77), "source-file", "kb-file", "user-1", "job-rag-1", int64(77), "source-file-1", kbJobTypeRAGIngest, kbSourceJobPending, kbSourceJobQueued, "workflow_trigger:"+knowledgeBaseWorkflowID("ws-1", 77)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	tenantMock.ExpectExec("UPDATE knowledge_base_source_job_runs").WillReturnResult(sqlmock.NewResult(1, 1))

	concrete, ok := svc.(*semanticModelService)
	if !ok {
		t.Fatalf("svc type = %T, want *semanticModelService", svc)
	}
	concrete.actionAuthorizer = &fakeSemanticModelActionAuthorizer{roleOut: "role-runtime"}
	concrete.volumeResolver = &fakeSemanticModelCanonicalVolumeResolver{}
	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithUserID(ctx, "user-1")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)

	if err := concrete.RunPendingKnowledgeBaseSourceJobs(ctx, RunPendingKnowledgeBaseSourceJobsParams{ModelID: 77}); err != nil {
		t.Fatalf("RunPendingKnowledgeBaseSourceJobs: %v", err)
	}
	if len(triggeredFiles) != 0 || len(workflowSvc.runs) != 1 {
		t.Fatalf("triggered files = %+v, workflow runs = %+v", triggeredFiles, workflowSvc.runs)
	}
	if len(workflowSvc.listFileExecutionCalls) != 1 || workflowSvc.listFileExecutionCalls[0] != "kb-file" {
		t.Fatalf("ListFileExecutions calls = %+v", workflowSvc.listFileExecutionCalls)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceRunPendingSourceJobsTreatsConcurrentDuplicateVersionAsIdempotent(t *testing.T) {
	sourceID := "source-file-1"
	kbFileID := "kb-new-file"
	indexVersion := int64(2)
	versionID := stableID("kb-segver", sourceID, indexVersion, kbSegmentSourceExternal)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":          77,
				"name":        "kb_docs",
				"description": "docs",
				"tables":      []semanticModelTableSource{},
				"files": map[string]any{
					"file_ids":        []string{"kb-old-file", kbFileID},
					"vector_table":    "kb_text_idx",
					"embedding_model": "embed-model",
				},
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()
	workflowID := "extra-workflow"
	operationID := "workflow_trigger:" + knowledgeBaseWorkflowID("ws-1", 77)
	workflowSvc := &fakeSemanticModelWorkflowService{fileExecutions: map[string]*moi.FileExecutionsResponse{
		kbFileID: {
			Executions: []moi.FileExecutionSummary{
				{ExecutionID: "exec-rag-new", WorkflowID: workflowID, Status: "succeeded"},
			},
			Total: 1,
		},
	}}
	svc := newSemanticModelTestServiceWithKnowledgeBaseRuntimeDependencies(t, server.URL, systemClient, nil, nil, nil, nil, workflowSvc)

	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}

	expectPendingSourceJobRunsWithPendingEmpty(tenantMock, 77, kbJobTypeCopy)
	expectPendingSourceJobRunsWithPendingEmpty(tenantMock, 77, kbJobTypeTableClone)
	expectPendingSourceJobRunsEmpty(tenantMock, 77, kbJobTypeLoad)
	tenantMock.ExpectQuery("SELECT .*job_id.*source_id.*model_id.*job_type.*job_status.*idempotency_key").
		WithArgs(int64(77), kbJobTypeRAGIngest, kbSourceJobPending, kbSourceJobQueued, kbSourceJobRunning, kbSourceJobSucceeded, kbSourceJobFailed, kbSourceJobReconcileBatchSize).
		WillReturnRows(knowledgeBaseSourceJobRunRows().
			AddRow("job-rag-1", sourceID, int64(77), kbJobTypeRAGIngest, kbSourceJobSucceeded, "idem-rag-1", operationID, nil, nil, nil, false, "source-file", kbFileID, nil, nil, int64(0), nil, nil, int64(100), int64(101)))
	tenantMock.ExpectQuery("SELECT .*source_id.*FROM knowledge_base_sources").
		WithArgs(int64(77)).
		WillReturnRows(knowledgeBaseSourceRecordRows(KnowledgeBaseSourceRecord{
			SourceID:          sourceID,
			ModelID:           77,
			CatalogID:         3,
			DatabaseID:        11,
			RawVolumeID:       12,
			ProcessedVolumeID: 13,
			SourceType:        kbSourceTypeCatalogFile,
			SourceFileID:      stringPtr("source-file"),
			KBFileID:          stringPtr("kb-old-file"),
			DisplayName:       stringPtr("doc.pdf"),
			Status:            kbSourceStatusSucceeded,
			Enabled:           boolPtr(true),
			Tags:              stringPtr(`["tag-a"]`),
			SegmentVersionID:  stringPtr("seg-v1"),
			IndexVersion:      int64Ptr(1),
		}))
	expectVectorTableSchemaColumns(tenantMock, "kb_text_idx", "vecf32(3)")
	tenantMock.ExpectQuery("SELECT MAX\\(index_version\\) FROM `kb_text_idx` WHERE file_id = \\?").
		WithArgs("source-file").
		WillReturnRows(sqlmock.NewRows([]string{"MAX(index_version)"}).AddRow(indexVersion))
	tenantMock.ExpectQuery("SELECT id, content, meta, level, chunk_index, index_version").
		WithArgs("source-file", indexVersion).
		WillReturnRows(sqlmock.NewRows([]string{"id", "content", "meta", "level", "chunk_index", "index_version"}).
			AddRow("row-v2-0", "workflow chunk", `{"chunk_id":"chunk-v2-0"}`, "chunk", int64(0), indexVersion))
	tenantMock.ExpectBegin()
	tenantMock.ExpectExec("INSERT INTO knowledge_base_segment_versions").
		WillReturnError(&mysqlDriver.MySQLError{Number: 1062, Message: fmt.Sprintf("Duplicate entry '%s' for key 'version_id'", versionID)})
	tenantMock.ExpectRollback()
	tenantMock.ExpectQuery("SELECT COUNT\\(1\\).*FROM knowledge_base_segment_versions").
		WithArgs(versionID, int64(77), sourceID, kbFileID, indexVersion, kbSegmentStatusCommitted, kbSegmentSourceExternal, kbFileID, versionID, indexVersion, kbSourceStatusSucceeded).
		WillReturnRows(sqlmock.NewRows([]string{"COUNT(1)"}).AddRow(int64(1)))
	tenantMock.ExpectExec("UPDATE knowledge_base_source_job_runs").
		WithArgs(kbSourceJobSucceeded, operationID, "exec-rag-new", kbFileID, nil, "user-1", "job-rag-1").
		WillReturnResult(sqlmock.NewResult(1, 1))

	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithUserID(ctx, "user-1")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)

	if err := svc.RunPendingKnowledgeBaseSourceJobs(ctx, RunPendingKnowledgeBaseSourceJobsParams{ModelID: 77}); err != nil {
		t.Fatalf("RunPendingKnowledgeBaseSourceJobs: %v", err)
	}
	if len(workflowSvc.listFileExecutionCalls) != 1 || workflowSvc.listFileExecutionCalls[0] != kbFileID {
		t.Fatalf("ListFileExecutions calls = %+v", workflowSvc.listFileExecutionCalls)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceRunPendingSourceJobsSkipsExternalWorkflowSameIndexVersion(t *testing.T) {
	sourceID := "source-file-1"
	workflowID := "extra-workflow"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			requireSemanticModelExecutionHeaders(t, r)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":          77,
				"name":        "kb_docs",
				"description": "docs",
				"tables":      []semanticModelTableSource{},
				"files": map[string]any{
					"file_ids":        []string{"kb-current-file"},
					"vector_table":    "kb_text_idx",
					"embedding_model": "embed-model",
				},
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()
	workflowSvc := &fakeSemanticModelWorkflowService{fileExecutions: map[string]*moi.FileExecutionsResponse{
		"kb-current-file": {
			Executions: []moi.FileExecutionSummary{
				{ExecutionID: "exec-rag-current", WorkflowID: workflowID, Status: "succeeded"},
			},
			Total: 1,
		},
	}}
	svc := newSemanticModelTestServiceWithKnowledgeBaseRuntimeDependencies(t, server.URL, systemClient, nil, nil, nil, nil, workflowSvc)

	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}

	expectPendingSourceJobRunsWithPendingEmpty(tenantMock, 77, kbJobTypeCopy)
	expectPendingSourceJobRunsWithPendingEmpty(tenantMock, 77, kbJobTypeTableClone)
	expectPendingSourceJobRunsEmpty(tenantMock, 77, kbJobTypeLoad)
	tenantMock.ExpectQuery("SELECT .*job_id.*source_id.*model_id.*job_type.*job_status.*idempotency_key").
		WithArgs(int64(77), kbJobTypeRAGIngest, kbSourceJobPending, kbSourceJobQueued, kbSourceJobRunning, kbSourceJobSucceeded, kbSourceJobFailed, kbSourceJobReconcileBatchSize).
		WillReturnRows(knowledgeBaseSourceJobRunRows().
			AddRow("job-rag-1", sourceID, int64(77), kbJobTypeRAGIngest, kbSourceJobSucceeded, "idem-rag-1", "workflow_trigger:"+knowledgeBaseWorkflowID("ws-1", 77), nil, nil, nil, false, "source-file", "kb-current-file", nil, nil, int64(0), nil, nil, int64(100), int64(101)))
	tenantMock.ExpectQuery("SELECT .*source_id.*FROM knowledge_base_sources").
		WithArgs(int64(77)).
		WillReturnRows(knowledgeBaseSourceRecordRows(KnowledgeBaseSourceRecord{
			SourceID:          sourceID,
			ModelID:           77,
			CatalogID:         3,
			DatabaseID:        11,
			RawVolumeID:       12,
			ProcessedVolumeID: 13,
			SourceType:        kbSourceTypeCatalogFile,
			SourceFileID:      stringPtr("source-file"),
			KBFileID:          stringPtr("kb-current-file"),
			DisplayName:       stringPtr("doc.pdf"),
			Status:            kbSourceStatusSucceeded,
			Enabled:           boolPtr(true),
			Tags:              stringPtr(`["tag-a"]`),
			SegmentVersionID:  stringPtr("seg-v2"),
			IndexVersion:      int64Ptr(2),
		}))
	expectVectorTableSchemaColumns(tenantMock, "kb_text_idx", "vecf32(3)")
	tenantMock.ExpectQuery("SELECT MAX\\(index_version\\) FROM `kb_text_idx` WHERE file_id = \\?").
		WithArgs("source-file").
		WillReturnRows(sqlmock.NewRows([]string{"MAX(index_version)"}).AddRow(int64(2)))
	tenantMock.ExpectExec("UPDATE knowledge_base_source_job_runs").
		WillReturnResult(sqlmock.NewResult(1, 1))

	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithUserID(ctx, "user-1")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)

	if err := svc.RunPendingKnowledgeBaseSourceJobs(ctx, RunPendingKnowledgeBaseSourceJobsParams{ModelID: 77}); err != nil {
		t.Fatalf("RunPendingKnowledgeBaseSourceJobs: %v", err)
	}
	if len(workflowSvc.listFileExecutionCalls) != 1 || workflowSvc.listFileExecutionCalls[0] != "kb-current-file" {
		t.Fatalf("ListFileExecutions calls = %+v", workflowSvc.listFileExecutionCalls)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceRunPendingSourceJobsRollsBackPublishedVersionWhenJobUpdateFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":          77,
				"name":        "kb_docs",
				"description": "docs",
				"tables":      []semanticModelTableSource{},
				"files": map[string]any{
					"file_ids":        []string{"kb-old-file", "kb-new-file"},
					"vector_table":    "kb_text_idx",
					"embedding_model": "embed-model",
				},
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()
	workflowID := knowledgeBaseWorkflowID("ws-1", 77)
	workflowSvc := &fakeSemanticModelWorkflowService{fileExecutions: map[string]*moi.FileExecutionsResponse{
		"kb-new-file": {
			Executions: []moi.FileExecutionSummary{
				{ExecutionID: "exec-rag-new", WorkflowID: workflowID, Status: "succeeded"},
			},
			Total: 1,
		},
	}}
	svc := newSemanticModelTestServiceWithKnowledgeBaseRuntimeDependencies(t, server.URL, systemClient, nil, nil, nil, nil, workflowSvc)

	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}

	expectPendingSourceJobRunsWithPendingEmpty(tenantMock, 77, kbJobTypeCopy)
	tenantMock.ExpectQuery("SELECT .*job_id.*source_id.*model_id.*job_type.*job_status.*idempotency_key").
		WithArgs(int64(77), kbJobTypeTableClone, kbSourceJobPending, kbSourceJobQueued, kbSourceJobRunning, kbSourceJobReconcileBatchSize).
		WillReturnRows(sqlmock.NewRows([]string{
			"job_id", "source_id", "model_id", "job_type", "job_status", "idempotency_key", "operation_id",
			"workflow_execution_id", "runtime_actor_moi_user_id", "runtime_effective_role_id", "runtime_is_workspace_owner", "source_file_id", "kb_file_id", "source_table_id", "kb_table_id",
			"retry_count", "next_retry_at", "error", "created_at", "updated_at",
		}))
	tenantMock.ExpectQuery("SELECT .*job_id.*source_id.*model_id.*job_type.*job_status.*idempotency_key").
		WithArgs(int64(77), kbJobTypeLoad, kbSourceJobQueued, kbSourceJobRunning, kbSourceJobSucceeded, kbSourceStatusRemoved, kbSourceTypeCatalogTable, kbSourceTypeLocalFile, kbRawKindStructured, kbSourceJobReconcileBatchSize).
		WillReturnRows(sqlmock.NewRows([]string{
			"job_id", "source_id", "model_id", "job_type", "job_status", "idempotency_key", "operation_id",
			"workflow_execution_id", "runtime_actor_moi_user_id", "runtime_effective_role_id", "runtime_is_workspace_owner", "source_file_id", "kb_file_id", "source_table_id", "kb_table_id",
			"retry_count", "next_retry_at", "error", "created_at", "updated_at",
		}))
	tenantMock.ExpectQuery("SELECT .*job_id.*source_id.*model_id.*job_type.*job_status.*idempotency_key").
		WithArgs(int64(77), kbJobTypeRAGIngest, kbSourceJobPending, kbSourceJobQueued, kbSourceJobRunning, kbSourceJobSucceeded, kbSourceJobFailed, kbSourceJobReconcileBatchSize).
		WillReturnRows(sqlmock.NewRows([]string{
			"job_id", "source_id", "model_id", "job_type", "job_status", "idempotency_key", "operation_id",
			"workflow_execution_id", "runtime_actor_moi_user_id", "runtime_effective_role_id", "runtime_is_workspace_owner", "source_file_id", "kb_file_id", "source_table_id", "kb_table_id",
			"retry_count", "next_retry_at", "error", "created_at", "updated_at",
		}).AddRow("job-rag-1", "source-file-1", int64(77), kbJobTypeRAGIngest, kbSourceJobRunning, "idem-rag-1", "workflow_trigger:"+workflowID, nil, nil, nil, false, "source-file", "kb-new-file", nil, nil, int64(0), nil, nil, int64(100), int64(101)))
	tenantMock.ExpectQuery("SELECT .*source_id.*FROM knowledge_base_sources").
		WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows([]string{
			"source_id", "model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "source_type",
			"source_file_id", "source_table_id", "kb_file_id", "kb_table_id", "display_name", "source_path",
			"db_name", "table_name", "status", "error", "enabled", "expires_at", "tags", "force_enabled_after_expiry", "segment_version_id", "index_version",
		}).AddRow("source-file-1", int64(77), int64(3), int64(11), int64(12), int64(13), kbSourceTypeCatalogFile, "source-file", nil, "kb-old-file", nil, "doc.pdf", nil, nil, nil, kbSourceStatusSucceeded, nil, true, nil, `["tag-a"]`, false, "seg-v1", int64(1)))
	expectVectorTableSchemaColumns(tenantMock, "kb_text_idx", "vecf32(3)")
	tenantMock.ExpectQuery("SELECT MAX\\(index_version\\) FROM `kb_text_idx` WHERE file_id = \\?").
		WithArgs("source-file").
		WillReturnRows(sqlmock.NewRows([]string{"MAX(index_version)"}).AddRow(int64(2)))
	tenantMock.ExpectQuery("SELECT id, content, meta, level, chunk_index, index_version").
		WithArgs("source-file", int64(2)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "content", "meta", "level", "chunk_index", "index_version"}).
			AddRow("row-v2-0", "workflow chunk", `{"chunk_id":"chunk-v2-0"}`, "chunk", int64(0), int64(2)))
	tenantMock.ExpectBegin()
	tenantMock.ExpectExec("INSERT INTO knowledge_base_segment_versions").
		WillReturnResult(sqlmock.NewResult(1, 1))
	tenantMock.ExpectExec("INSERT INTO [`\"]?knowledge_base_segments[`\"]?").
		WillReturnResult(sqlmock.NewResult(1, 1))
	tenantMock.ExpectExec("INSERT INTO [`\"]?knowledge_base_chunk_recall_stats[`\"]?").
		WillReturnResult(sqlmock.NewResult(1, 1))
	tenantMock.ExpectExec("UPDATE knowledge_base_sources").
		WillReturnResult(sqlmock.NewResult(1, 1))
	tenantMock.ExpectExec("UPDATE knowledge_base_source_job_runs").
		WillReturnError(errors.New("job update failed"))
	tenantMock.ExpectRollback()

	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithUserID(ctx, "user-1")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)

	err = svc.RunPendingKnowledgeBaseSourceJobs(ctx, RunPendingKnowledgeBaseSourceJobsParams{ModelID: 77})
	if err == nil || !strings.Contains(err.Error(), "job update failed") {
		t.Fatalf("RunPendingKnowledgeBaseSourceJobs error = %v", err)
	}
	if len(workflowSvc.listFileExecutionCalls) != 1 || workflowSvc.listFileExecutionCalls[0] != "kb-new-file" {
		t.Fatalf("ListFileExecutions calls = %+v", workflowSvc.listFileExecutionCalls)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestCommitSegmentVersionUsesTwoBatchesFor101SegmentsWithoutSavepoint(t *testing.T) {
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}

	const n = 101
	record := KnowledgeBaseSourceRecord{
		SourceID: "source-file-1",
		ModelID:  77,
		KBFileID: stringPtr("kb-file"),
	}
	binding := kbVectorBinding{VectorTable: "kb_text_idx", EmbeddingModel: "embed-model"}
	versionID := "seg-v-batch-101"
	segments := make([]kbSegmentRecord, 0, n)
	for i := 0; i < n; i++ {
		idx := int64(i)
		segments = append(segments, kbSegmentRecord{
			SegmentID:    fmt.Sprintf("seg-%03d", i),
			VersionID:    versionID,
			ModelID:      record.ModelID,
			SourceID:     record.SourceID,
			KBFileID:     "kb-file",
			IndexVersion: 2,
			Level:        kbSegmentLevelChunk,
			ChunkIndex:   &idx,
			IdentityKey:  fmt.Sprintf("idx:%d", i),
			Enabled:      true,
			WordCount:    1,
		})
	}

	tenantMock.ExpectBegin()
	tenantMock.ExpectExec("INSERT INTO knowledge_base_segment_versions").
		WithArgs(versionID, int64(77), "source-file-1", "kb-file", int64(2), nil, nil, kbSegmentStatusCommitted, kbSegmentSourceExternal, int64(n), int64(n), "kb_text_idx", "embed-model", nil, nil, "user-1", "user-1").
		WillReturnResult(sqlmock.NewResult(1, 1))
	// Two CreateInBatches calls of 100 and 1 for segments, then the same for recall stats.
	// SkipDefaultTransaction keeps the outer transaction as the only BEGIN/COMMIT owner.
	tenantMock.ExpectExec("INSERT INTO [`\"]?knowledge_base_segments[`\"]?").
		WillReturnResult(sqlmock.NewResult(100, 100))
	tenantMock.ExpectExec("INSERT INTO [`\"]?knowledge_base_segments[`\"]?").
		WillReturnResult(sqlmock.NewResult(1, 1))
	tenantMock.ExpectExec("INSERT INTO [`\"]?knowledge_base_chunk_recall_stats[`\"]?").
		WillReturnResult(sqlmock.NewResult(100, 100))
	tenantMock.ExpectExec("INSERT INTO [`\"]?knowledge_base_chunk_recall_stats[`\"]?").
		WillReturnResult(sqlmock.NewResult(1, 1))
	tenantMock.ExpectExec("UPDATE knowledge_base_sources").
		WillReturnResult(sqlmock.NewResult(1, 1))
	tenantMock.ExpectCommit()

	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)
	svc := &semanticModelService{}
	if err := svc.commitSegmentVersionWithTxHook(ctx, record, binding, kbSegmentSourceExternal, versionID, 2, segments, kbSegmentMaterialization{}, SemanticModelSegmentMutationBase{}, nil); err != nil {
		t.Fatalf("commitSegmentVersionWithTxHook: %v", err)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestCommitSegmentVersionRollsBackOuterTransactionWhenSecondSegmentBatchFails(t *testing.T) {
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}

	const n = 101
	record := KnowledgeBaseSourceRecord{
		SourceID: "source-file-1",
		ModelID:  77,
		KBFileID: stringPtr("kb-file"),
	}
	binding := kbVectorBinding{VectorTable: "kb_text_idx", EmbeddingModel: "embed-model"}
	versionID := "seg-v-batch-fail"
	segments := make([]kbSegmentRecord, 0, n)
	for i := 0; i < n; i++ {
		idx := int64(i)
		segments = append(segments, kbSegmentRecord{
			SegmentID:    fmt.Sprintf("seg-%03d", i),
			VersionID:    versionID,
			ModelID:      record.ModelID,
			SourceID:     record.SourceID,
			KBFileID:     "kb-file",
			IndexVersion: 2,
			Level:        kbSegmentLevelChunk,
			ChunkIndex:   &idx,
			IdentityKey:  fmt.Sprintf("idx:%d", i),
			Enabled:      true,
			WordCount:    1,
		})
	}

	tenantMock.ExpectBegin()
	tenantMock.ExpectExec("INSERT INTO knowledge_base_segment_versions").
		WillReturnResult(sqlmock.NewResult(1, 1))
	tenantMock.ExpectExec("INSERT INTO [`\"]?knowledge_base_segments[`\"]?").
		WillReturnResult(sqlmock.NewResult(100, 100))
	tenantMock.ExpectExec("INSERT INTO [`\"]?knowledge_base_segments[`\"]?").
		WillReturnError(errors.New("second segment batch failed"))
	// Outer rollback only; SkipDefaultTransaction means no ROLLBACK TO SAVEPOINT.
	tenantMock.ExpectRollback()

	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)
	svc := &semanticModelService{}
	err = svc.commitSegmentVersionWithTxHook(ctx, record, binding, kbSegmentSourceExternal, versionID, 2, segments, kbSegmentMaterialization{}, SemanticModelSegmentMutationBase{}, func(tx *gorm.DB) error {
		t.Fatal("txHook must not run after batch failure")
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "second segment batch failed") {
		t.Fatalf("commitSegmentVersionWithTxHook error = %v", err)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceRunPendingSourceJobsFailureDoesNotDisableOrSwitchCurrentVersion(t *testing.T) {
	workflowID := "workflow-rag-1"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":          77,
				"name":        "kb_docs",
				"description": "docs",
				"tables":      []any{},
				"files":       map[string]any{"file_ids": []string{"kb-old-file"}},
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()
	workflowSvc := &fakeSemanticModelWorkflowService{
		fileExecutions: map[string]*moi.FileExecutionsResponse{
			"kb-new-file": {
				Executions: []moi.FileExecutionSummary{{
					ExecutionID: "exec-rag-1",
					WorkflowID:  workflowID,
					Status:      "failed",
					Error:       "parse failed",
				}},
				Total: 1,
			},
		},
	}
	svc := newSemanticModelTestServiceWithKnowledgeBaseRuntimeDependencies(t, server.URL, systemClient, nil, nil, nil, nil, workflowSvc)
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}

	expectPendingSourceJobRunsWithPendingEmpty(tenantMock, 77, kbJobTypeCopy)
	tenantMock.ExpectQuery("SELECT .*job_id.*source_id.*model_id.*job_type.*job_status.*idempotency_key").
		WithArgs(int64(77), kbJobTypeTableClone, kbSourceJobPending, kbSourceJobQueued, kbSourceJobRunning, kbSourceJobReconcileBatchSize).
		WillReturnRows(sqlmock.NewRows([]string{
			"job_id", "source_id", "model_id", "job_type", "job_status", "idempotency_key", "operation_id",
			"workflow_execution_id", "runtime_actor_moi_user_id", "runtime_effective_role_id", "runtime_is_workspace_owner", "source_file_id", "kb_file_id", "source_table_id", "kb_table_id",
			"retry_count", "next_retry_at", "error", "created_at", "updated_at",
		}))
	tenantMock.ExpectQuery("SELECT .*job_id.*source_id.*model_id.*job_type.*job_status.*idempotency_key").
		WithArgs(int64(77), kbJobTypeLoad, kbSourceJobQueued, kbSourceJobRunning, kbSourceJobSucceeded, kbSourceStatusRemoved, kbSourceTypeCatalogTable, kbSourceTypeLocalFile, kbRawKindStructured, kbSourceJobReconcileBatchSize).
		WillReturnRows(sqlmock.NewRows([]string{
			"job_id", "source_id", "model_id", "job_type", "job_status", "idempotency_key", "operation_id",
			"workflow_execution_id", "runtime_actor_moi_user_id", "runtime_effective_role_id", "runtime_is_workspace_owner", "source_file_id", "kb_file_id", "source_table_id", "kb_table_id",
			"retry_count", "next_retry_at", "error", "created_at", "updated_at",
		}))
	tenantMock.ExpectQuery("SELECT .*job_id.*source_id.*model_id.*job_type.*job_status.*idempotency_key").
		WithArgs(int64(77), kbJobTypeRAGIngest, kbSourceJobPending, kbSourceJobQueued, kbSourceJobRunning, kbSourceJobSucceeded, kbSourceJobFailed, kbSourceJobReconcileBatchSize).
		WillReturnRows(sqlmock.NewRows([]string{
			"job_id", "source_id", "model_id", "job_type", "job_status", "idempotency_key", "operation_id",
			"workflow_execution_id", "runtime_actor_moi_user_id", "runtime_effective_role_id", "runtime_is_workspace_owner", "source_file_id", "kb_file_id", "source_table_id", "kb_table_id",
			"retry_count", "next_retry_at", "error", "created_at", "updated_at",
		}).AddRow("job-rag-1", "source-file-1", int64(77), kbJobTypeRAGIngest, kbSourceJobRunning, "idem-rag-1", "workflow_trigger:"+workflowID, nil, nil, nil, false, "source-file", "kb-new-file", nil, nil, int64(0), nil, nil, int64(100), int64(101)))
	tenantMock.ExpectQuery("SELECT .*source_id.*FROM knowledge_base_sources").
		WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows([]string{
			"source_id", "model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "source_type",
			"source_file_id", "source_table_id", "kb_file_id", "kb_table_id", "display_name", "source_path",
			"db_name", "table_name", "status", "error", "enabled", "expires_at", "tags", "force_enabled_after_expiry", "segment_version_id", "index_version",
		}).AddRow("source-file-1", int64(77), int64(3), int64(11), int64(12), int64(13), kbSourceTypeCatalogFile, "source-file", nil, "kb-old-file", nil, "doc.pdf", nil, nil, nil, kbSourceStatusSucceeded, nil, true, nil, `["tag-a"]`, false, "seg-v1", int64(1)))
	tenantMock.ExpectExec("UPDATE knowledge_base_sources\\s+SET status = \\?, error = \\?, updated_by = \\?\\s+WHERE source_id = \\?").
		WithArgs(kbSourceStatusFailed, "parse failed", "user-1", "source-file-1").
		WillReturnResult(sqlmock.NewResult(1, 1))
	tenantMock.ExpectExec("UPDATE knowledge_base_source_job_runs").
		WithArgs(kbSourceJobFailed, "parse failed", "user-1", "job-rag-1").
		WillReturnResult(sqlmock.NewResult(1, 1))

	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithUserID(ctx, "user-1")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)

	if err := svc.RunPendingKnowledgeBaseSourceJobs(ctx, RunPendingKnowledgeBaseSourceJobsParams{ModelID: 77}); err != nil {
		t.Fatalf("RunPendingKnowledgeBaseSourceJobs: %v", err)
	}
	if len(workflowSvc.listFileExecutionCalls) != 1 || workflowSvc.listFileExecutionCalls[0] != "kb-new-file" {
		t.Fatalf("ListFileExecutions calls = %+v", workflowSvc.listFileExecutionCalls)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func expectRAGSourceJobRows(mock sqlmock.Sqlmock, modelID int64, operationID string, workflowExecutionID *string, record KnowledgeBaseSourceRecord) {
	if record.SourceID == "" {
		record = KnowledgeBaseSourceRecord{SourceID: "source-file-1", ModelID: modelID, RawVolumeID: 12, SourceType: kbSourceTypeCatalogFile, SourceFileID: stringPtr("source-file"), KBFileID: stringPtr("kb-file"), Status: kbSourceStatusPending, Enabled: boolPtr(true)}
	}
	expectSourceJobCandidates(mock, modelID, []string{"source-file-1"}, knowledgeBaseSourceJobRunRows().
		AddRow("job-rag-1", "source-file-1", modelID, kbJobTypeRAGIngest, kbSourceJobPending, "idem-rag-1", operationID, workflowExecutionID, nil, nil, false, "source-file", "kb-file", nil, nil, int64(0), nil, nil, int64(100), int64(101)),
		knowledgeBaseSourceRecordRows(record))
}

func knowledgeBaseSourceRecordRows(records ...KnowledgeBaseSourceRecord) *sqlmock.Rows {
	rows := sqlmock.NewRows([]string{
		"source_id", "model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "source_type",
		"source_file_id", "source_table_id", "kb_file_id", "kb_table_id", "display_name", "source_path",
		"db_name", "table_name", "status", "error", "enabled", "expires_at", "tags", "force_enabled_after_expiry", "segment_version_id", "index_version",
		"size_bytes", "row_count", "created_by", "updated_by", "updated_at",
	})
	for _, record := range records {
		rows.AddRow(record.SourceID, record.ModelID, record.CatalogID, record.DatabaseID, record.RawVolumeID, record.ProcessedVolumeID, record.SourceType,
			stringValue(record.SourceFileID), int64Value(record.SourceTableID), stringValue(record.KBFileID), int64Value(record.KBTableID), stringValue(record.DisplayName), stringValue(record.SourcePath),
			stringValue(record.DBName), stringValue(record.TableName), record.Status, stringValue(record.Error), boolValue(record.Enabled), int64Value(record.ExpiresAt), stringValue(record.Tags), record.ForceEnabled, stringValue(record.SegmentVersionID), int64Value(record.IndexVersion),
			int64Value(record.SizeBytes), int64Value(record.RowCount), stringValue(record.CreatedBy), stringValue(record.UpdatedBy), int64Value(record.UpdatedAt))
	}
	return rows
}

func stringValue(value *string) driver.Value {
	if value == nil {
		return nil
	}
	return *value
}

func int64Value(value *int64) driver.Value {
	if value == nil {
		return nil
	}
	return *value
}

func boolValue(value *bool) driver.Value {
	if value == nil {
		return nil
	}
	return *value
}

func boolPtr(value bool) *bool {
	return &value
}

func expectCatalogFileSourceLookupMiss(mock sqlmock.Sqlmock, modelID int64, sourceFileID string) {
	// Append/create paths that already carry VolumeID on the request.
	mock.ExpectQuery("SELECT .*FROM knowledge_base_sources kbs").
		WithArgs(modelID, kbSourceTypeCatalogFile, sourceFileID, sourceFileID).
		WillReturnRows(knowledgeBaseSourceRecordRows())
	expectCatalogFileSourceOriginLookupMissWithMeta(mock, sourceFileID, 41, sourceFileID+".pdf")
}

func expectCatalogFileSourceOriginLookupMiss(mock sqlmock.Sqlmock, kbFileID string) {
	expectCatalogFileSourceOriginLookupMissWithMeta(mock, kbFileID, 41, kbFileID+".pdf")
}

// expectCatalogFileSourceOriginLookupMissWithMeta stubs resolve miss + exact (file_id, volume_id)
// when the write request already carries VolumeID.
func expectCatalogFileSourceOriginLookupMissWithMeta(mock sqlmock.Sqlmock, kbFileID string, volumeID int64, fileName string) {
	mock.ExpectQuery(`(?s)SELECT source_file_id\s+FROM knowledge_base_sources\s+WHERE kb_file_id = \?\s+AND source_type IN \(\?, \?\)\s+AND source_file_id IS NOT NULL\s+AND source_file_id <> ''\s+ORDER BY created_at ASC, source_id ASC\s+LIMIT 1`).
		WithArgs(kbFileID, kbSourceTypeCatalogFile, kbSourceTypeLocalFile).
		WillReturnRows(sqlmock.NewRows([]string{"source_file_id"}))
	expectCatalogFileMetadataBatchAtVolume(mock, volumeID, []string{kbFileID}, fileName)
}

// expectCatalogFileMetadataBatchAtVolume stubs list/detail exact (file_id, volume_id) enrichment.
// fileIDs is the order of unique file ids in the batch query (same as lookup order).
func expectCatalogFileMetadataBatchAtVolume(mock sqlmock.Sqlmock, volumeID int64, fileIDs []string, fileName string) {
	args := make([]driver.Value, len(fileIDs))
	rows := sqlmock.NewRows([]string{"file_id", "catalog_id", "database_id", "volume_id", "size", "updated_at", "catalog_name", "database_name", "volume_name", "file_path", "file_name"})
	for i, fileID := range fileIDs {
		args[i] = fileID
		rows.AddRow(fileID, int64(2), int64(20), volumeID, int64(100), int64(200), "user_catalog", "user_db", "user_vol", "", fileName)
	}
	mock.ExpectQuery("SELECT f.file_id, COALESCE.*WHERE f.file_id IN").
		WithArgs(args...).
		WillReturnRows(rows)
}

func expectCatalogFileSourceLookupHit(mock sqlmock.Sqlmock, modelID int64, sourceFileID string, record KnowledgeBaseSourceRecord) {
	mock.ExpectQuery("SELECT .*FROM knowledge_base_sources kbs").
		WithArgs(modelID, kbSourceTypeCatalogFile, sourceFileID, sourceFileID).
		WillReturnRows(knowledgeBaseSourceRecordRows(record))
}

func expectCatalogTableSourceLookupMiss(mock sqlmock.Sqlmock, modelID int64, sourceTableID int64) {
	mock.ExpectQuery("SELECT .*FROM knowledge_base_sources kbs").
		WithArgs(modelID, kbSourceTypeCatalogTable, sourceTableID, sourceTableID, kbSourceStatusSucceeded).
		WillReturnRows(knowledgeBaseSourceRecordRows())
}

func expectCatalogTableSourceLookupHit(mock sqlmock.Sqlmock, modelID int64, sourceTableID int64, record KnowledgeBaseSourceRecord) {
	mock.ExpectQuery("SELECT .*FROM knowledge_base_sources kbs").
		WithArgs(modelID, kbSourceTypeCatalogTable, sourceTableID, sourceTableID, kbSourceStatusSucceeded).
		WillReturnRows(knowledgeBaseSourceRecordRows(record))
}

func serveCatalogTableDetail(t *testing.T, w http.ResponseWriter, r *http.Request, tableID, catalogID, databaseID int64, catalogName, databaseName, tableName string) bool {
	t.Helper()
	if r.Method != http.MethodGet || r.URL.Path != fmt.Sprintf("/api/v1/workspaces/ws-1/tables/%d", tableID) {
		return false
	}
	requireSemanticModelExecutionHeaders(t, r)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"table": map[string]any{
			"id":          tableID,
			"catalog_id":  catalogID,
			"database_id": databaseID,
			"name":        tableName,
		},
		"database": map[string]any{
			"id":   databaseID,
			"name": databaseName,
		},
		"catalog": map[string]any{
			"id":   catalogID,
			"name": catalogName,
		},
	})
	return true
}

func expectDeleteKnowledgeBaseSourceSegments(mock sqlmock.Sqlmock, modelID int64, sourceID string) {
	for _, table := range []string{"knowledge_base_chunk_recall_stats", "knowledge_base_segments", "knowledge_base_segment_versions"} {
		mock.ExpectExec("DELETE FROM "+table).
			WithArgs(modelID, sourceID).
			WillReturnResult(sqlmock.NewResult(0, 1))
	}
}

func expectMarkKnowledgeBaseSourceRemoved(mock sqlmock.Sqlmock, modelID int64, sourceID string, actor string) {
	mock.ExpectExec("UPDATE knowledge_base_sources").
		WithArgs(kbSourceStatusRemoved, actor, modelID, sourceID, kbSourceStatusRemoved).
		WillReturnResult(sqlmock.NewResult(0, 1))
}

func expectKnowledgeBaseDataDomainLock(mock sqlmock.Sqlmock, modelID int64) {
	mock.ExpectQuery("SELECT model_id.*FROM knowledge_base_data_domains.*FOR UPDATE").
		WithArgs(modelID).
		WillReturnRows(sqlmock.NewRows([]string{"model_id"}).AddRow(modelID))
}

func expectKnowledgeBaseDataDomainLockMissing(mock sqlmock.Sqlmock, modelID int64) {
	mock.ExpectQuery("SELECT model_id.*FROM knowledge_base_data_domains.*FOR UPDATE").
		WithArgs(modelID).
		WillReturnRows(sqlmock.NewRows([]string{"model_id"}))
}

func expectKnowledgeBaseSourceDeleteLocks(mock sqlmock.Sqlmock, modelID int64, sourceID string) {
	mock.ExpectQuery("SELECT source_id.*FROM knowledge_base_sources.*FOR UPDATE").
		WithArgs(modelID, sourceID).
		WillReturnRows(sqlmock.NewRows([]string{"source_id"}).AddRow(sourceID))
	mock.ExpectQuery("SELECT job_id,.*CASE WHEN job_status.*FROM knowledge_base_source_job_runs.*FOR UPDATE").
		WithArgs(kbSourceJobRunning, int(kbSourceJobClaimLease/time.Second), modelID, sourceID).
		WillReturnRows(sqlmock.NewRows([]string{"job_id", "active_claim"}).AddRow("completed-job", 0))
}

func expectKnowledgeBaseDataDomain(mock sqlmock.Sqlmock, modelID, rawVolumeID int64) {
	mock.ExpectQuery("SELECT model_id, catalog_id, database_id, raw_volume_id, processed_volume_id, ensure_status, last_ensure_error, last_checked_at\\s+FROM knowledge_base_data_domains").
		WithArgs(modelID).
		WillReturnRows(sqlmock.NewRows([]string{"model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "ensure_status", "last_ensure_error", "last_checked_at"}).
			AddRow(modelID, 3, 11, rawVolumeID, 13, kbEnsureStatusReady, nil, int64(1)))
}

func expectedKnowledgeBaseWorkflowIDs(wsID string, modelID int64) []string {
	return []string{
		knowledgeBaseWorkflowID(wsID, modelID),
		knowledgeBaseMediaWorkflowID(wsID, modelID, kbAudioRAGTemplateKey),
		knowledgeBaseMediaWorkflowID(wsID, modelID, kbVideoRAGTemplateKey),
	}
}

func TestIssue11520DocumentSegmentCanonicalMetadataPreservesZeroTimestamp(t *testing.T) {
	segments := semanticModelDocumentSegments([]SemanticModelSegment{{
		SegmentID: "segment-1",
		Metadata:  json.RawMessage(`{"segment_type":"transcript","start_ms":0,"end_ms":1250,"private":"hidden","volume_id":12}`),
	}})
	if len(segments) != 1 || segments[0].SegmentType != "transcript" {
		t.Fatalf("segments = %+v", segments)
	}
	if segments[0].StartMS == nil || *segments[0].StartMS != 0 || segments[0].EndMS == nil || *segments[0].EndMS != 1250 {
		t.Fatalf("time range = %v..%v", segments[0].StartMS, segments[0].EndMS)
	}
	if string(segments[0].Metadata) != `{"volume_id":12}` {
		t.Fatalf("metadata = %s", segments[0].Metadata)
	}
}

func TestIssue11520TemplateKeyForKnowledgeBaseFile(t *testing.T) {
	tests := []struct {
		fileName string
		want     string
		wantErr  bool
	}{
		{fileName: "meeting.M4A", want: kbAudioRAGTemplateKey},
		{fileName: "recording.wma", want: kbAudioRAGTemplateKey},
		{fileName: "meeting.MP4", want: kbVideoRAGTemplateKey},
		{fileName: "recording.wmv", want: kbVideoRAGTemplateKey},
		{fileName: "notes.pdf", want: kbStandardRAGTemplateKey},
		{fileName: "diagram.tiff", want: kbStandardRAGTemplateKey},
		{fileName: "inventory.XLSX", want: kbStandardRAGTemplateKey},
		{fileName: "legacy.xls", want: kbStandardRAGTemplateKey},
		{fileName: "no-extension", wantErr: true},
	}
	for _, tt := range tests {
		got, err := templateKeyForKnowledgeBaseFile(stringPtr(tt.fileName))
		if (err != nil) != tt.wantErr || got != tt.want {
			t.Errorf("templateKeyForKnowledgeBaseFile(%q) = %q, %v; want %q, error=%v", tt.fileName, got, err, tt.want, tt.wantErr)
		}
	}
}

func TestIssue11520RAGJobUsesLatestKnowledgeBaseFileExecution(t *testing.T) {
	workflowSvc := &fakeSemanticModelWorkflowService{fileExecutions: map[string]*moi.FileExecutionsResponse{
		"file-1": {Executions: []moi.FileExecutionSummary{
			{ExecutionID: "exec-owned", Status: "running", UpdatedAt: "2026-07-12T10:00:00Z"},
			{ExecutionID: "exec-newer", Status: "succeeded", UpdatedAt: "2026-07-12T11:00:00Z"},
		}},
	}}
	svc := &semanticModelService{workflowService: workflowSvc}
	job := &KnowledgeBaseSourceJobRun{JobID: "job-1", KBFileID: stringPtr("file-1"), WorkflowExecutionID: stringPtr("exec-owned")}
	refreshed, err := svc.refreshRAGSourceJobFromFileExecutions(context.Background(), job)
	if err != nil {
		t.Fatalf("refreshRAGSourceJobFromFileExecutions: %v", err)
	}
	if !refreshed || ptrValue(job.WorkflowExecutionID) != "exec-newer" || job.JobStatus != kbSourceJobSucceeded {
		t.Fatalf("job = %+v", job)
	}
}

func TestIssue11520FailedRAGJobAdoptsRetriedWorkflowExecution(t *testing.T) {
	workflowSvc := &fakeSemanticModelWorkflowService{fileExecutions: map[string]*moi.FileExecutionsResponse{
		"file-1": {Executions: []moi.FileExecutionSummary{
			{ExecutionID: "exec-failed", WorkflowID: "workflow-1", Status: "failed", UpdatedAt: "2026-07-12T10:00:00Z"},
			{ExecutionID: "exec-retry", WorkflowID: "workflow-1", Status: "running", UpdatedAt: "2026-07-12T11:00:00Z"},
		}},
	}}
	svc := &semanticModelService{workflowService: workflowSvc}
	job := &KnowledgeBaseSourceJobRun{
		JobID: "job-1", JobStatus: kbSourceJobFailed, KBFileID: stringPtr("file-1"), WorkflowExecutionID: stringPtr("exec-failed"),
	}
	refreshed, err := svc.refreshRAGSourceJobFromFileExecutions(context.Background(), job)
	if err != nil {
		t.Fatalf("refreshRAGSourceJobFromFileExecutions: %v", err)
	}
	if !refreshed || ptrValue(job.WorkflowExecutionID) != "exec-retry" || job.JobStatus != kbSourceJobRunning {
		t.Fatalf("job = %+v", job)
	}
}

func TestIssue11520FailedRAGJobPersistsRetriedWorkflowExecution(t *testing.T) {
	tenantDB, tenantMock := newSemanticModelTenantDB(t)
	tenantMock.ExpectExec("UPDATE knowledge_base_sources").
		WithArgs(kbSourceStatusPending, "user-1", "source-1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	tenantMock.ExpectExec("UPDATE knowledge_base_source_job_runs").WillReturnResult(sqlmock.NewResult(1, 1))
	tenantMock.ExpectExec("UPDATE knowledge_base_source_job_runs").WillReturnResult(sqlmock.NewResult(1, 1))
	workflowSvc := &fakeSemanticModelWorkflowService{fileExecutions: map[string]*moi.FileExecutionsResponse{
		"file-1": {Executions: []moi.FileExecutionSummary{
			{ExecutionID: "exec-failed", WorkflowID: "workflow-1", Status: "failed", UpdatedAt: "2026-07-12T10:00:00Z"},
			{ExecutionID: "exec-retry", WorkflowID: "workflow-1", Status: "running", UpdatedAt: "2026-07-12T11:00:00Z"},
		}},
	}}
	svc := &semanticModelService{workflowService: workflowSvc}
	job := &KnowledgeBaseSourceJobRun{
		JobID: "job-1", SourceID: "source-1", ModelID: 77, JobType: kbJobTypeRAGIngest, JobStatus: kbSourceJobFailed,
		KBFileID: stringPtr("file-1"), WorkflowExecutionID: stringPtr("exec-failed"), Error: stringPtr("parse failed"),
	}
	ctx := ctxutil.WithTenantDB(context.Background(), tenantDB)
	if err := svc.reconcileRAGIngestSourceJob(ctx, nil, "ws-1", KnowledgeBaseSourceRecord{SourceID: "source-1", ModelID: 77, Status: kbSourceStatusFailed}, job, "user-1"); err != nil {
		t.Fatalf("reconcileRAGIngestSourceJob: %v", err)
	}
	if ptrValue(job.WorkflowExecutionID) != "exec-retry" || job.JobStatus != kbSourceJobRunning || job.Error != nil {
		t.Fatalf("job = %+v", job)
	}
	workflowSvc.fileExecutions["file-1"] = &moi.FileExecutionsResponse{Executions: []moi.FileExecutionSummary{
		{ExecutionID: "exec-retry", WorkflowID: "workflow-1", Status: "succeeded", UpdatedAt: "2026-07-12T12:00:00Z"},
	}}
	refreshed, err := svc.refreshRAGSourceJobFromFileExecutions(ctx, job)
	if err != nil || !refreshed || job.JobStatus != kbSourceJobSucceeded || ptrValue(job.WorkflowExecutionID) != "exec-retry" {
		t.Fatalf("second reconcile refresh = %+v, refreshed=%v, err=%v", job, refreshed, err)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestIssue11520RAGJobDoesNotCrossIdentityWhenFileExecutionIndexLags(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("RAG enrichment must not query execution with the system identity: %s %s", r.Method, r.URL.String())
	}))
	defer server.Close()
	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()
	workflowSvc := &fakeSemanticModelWorkflowService{fileExecutions: map[string]*moi.FileExecutionsResponse{
		"file-1": {Executions: []moi.FileExecutionSummary{}},
	}}
	svc := &semanticModelService{workflowService: workflowSvc}
	jobs, err := svc.enrichKnowledgeBaseSourceJobRunsFromLinkedJobs(context.Background(), systemClient, "ws-1", []KnowledgeBaseSourceJobRun{{
		JobID: "job-1", JobType: kbJobTypeRAGIngest, JobStatus: kbSourceJobRunning,
		KBFileID: stringPtr("file-1"), WorkflowExecutionID: stringPtr("exec-1"),
	}})
	if err != nil {
		t.Fatalf("enrichKnowledgeBaseSourceJobRunsFromLinkedJobs: %v", err)
	}
	if len(jobs) != 1 || ptrValue(jobs[0].WorkflowExecutionID) != "exec-1" || jobs[0].JobStatus != kbSourceJobRunning {
		t.Fatalf("jobs = %+v", jobs)
	}
}

func TestSemanticModelServiceKnowledgeBaseJobEnrichmentUsesUnifiedClient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || !strings.HasPrefix(r.URL.Path, "/api/v1/workspaces/ws-1/workflow-apps/executions/") {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		if got := r.Header.Get("X-API-Key"); got != "system-key" {
			t.Fatalf("authorization = %q, want system key", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"execution": map[string]any{
			"execution_id": strings.TrimPrefix(r.URL.Path, "/api/v1/workspaces/ws-1/workflow-apps/executions/"),
			"status":       "running",
		}})
	}))
	defer server.Close()

	configureSemanticModelTestCore(t, server.URL)
	svc := &semanticModelService{}
	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")

	jobs := []KnowledgeBaseSourceJobRun{
		{JobID: "job-1", JobType: kbJobTypeRAGIngest, WorkflowExecutionID: stringPtr("exec-1")},
		{JobID: "job-2", JobType: kbJobTypeRAGIngest, WorkflowExecutionID: stringPtr("exec-2")},
	}
	var got []KnowledgeBaseSourceJobRun
	err := coreclient.Execute(ctx, coreclient.FromContext(ctx), func(callCtx context.Context, client *moi.Client) error {
		var callErr error
		got, callErr = svc.enrichKnowledgeBaseSourceJobRunsFromLinkedJobs(callCtx, client, "ws-1", jobs)
		return callErr
	})
	if err != nil {
		t.Fatalf("enrich knowledge base source jobs: %v", err)
	}
	for i := range got {
		if got[i].JobStatus != kbSourceJobRunning {
			t.Fatalf("job[%d] status = %q, want %q", i, got[i].JobStatus, kbSourceJobRunning)
		}
	}
}

func TestSemanticModelServiceKnowledgeBaseJobEnrichmentSkipsNonRAGJobs(t *testing.T) {
	jobs := []KnowledgeBaseSourceJobRun{
		{JobID: "job-load", JobType: kbJobTypeLoad, JobStatus: kbSourceJobQueued, OperationID: stringPtr("import_task:task-1"), WorkflowExecutionID: stringPtr("exec-load")},
		{JobID: "job-clone", JobType: kbJobTypeTableClone, JobStatus: kbSourceJobRunning, WorkflowExecutionID: stringPtr("exec-clone")},
		{JobID: "job-copy", JobType: kbJobTypeCopy, JobStatus: kbSourceJobPending, WorkflowExecutionID: stringPtr("exec-copy")},
	}
	stats := &knowledgeBaseSourceJobEnrichStats{}
	got, err := (&semanticModelService{}).enrichKnowledgeBaseSourceJobRunsFromLinkedJobsWithStats(context.Background(), nil, "ws-1", jobs, stats)
	if err != nil {
		t.Fatalf("enrich knowledge base source jobs: %v", err)
	}
	if got[0].JobStatus != kbSourceJobQueued || got[1].JobStatus != kbSourceJobRunning || got[2].JobStatus != kbSourceJobPending {
		t.Fatalf("jobs = %+v", got)
	}
	if stats.importTaskCalls.Load() != 0 || stats.fileExecutionCalls.Load() != 0 || stats.workflowExecutionCalls.Load() != 0 {
		t.Fatalf("enrichment stats = %+v", stats)
	}
}

func TestIssue13313SourceJobEnrichmentUsesBoundedConcurrency(t *testing.T) {
	for _, jobCount := range []int{1, 10, 32} {
		t.Run(fmt.Sprintf("jobs_%d", jobCount), func(t *testing.T) {
			var active atomic.Int64
			var peak atomic.Int64
			workflowSvc := &fakeSemanticModelWorkflowService{
				fileExecutions: make(map[string]*moi.FileExecutionsResponse, jobCount),
				onListFileExecutions: func(ctx context.Context, _ string) error {
					current := active.Add(1)
					defer active.Add(-1)
					for {
						previous := peak.Load()
						if current <= previous || peak.CompareAndSwap(previous, current) {
							break
						}
					}
					select {
					case <-time.After(15 * time.Millisecond):
						return nil
					case <-ctx.Done():
						return ctx.Err()
					}
				},
			}
			jobs := make([]KnowledgeBaseSourceJobRun, jobCount)
			for i := range jobs {
				fileID := fmt.Sprintf("file-%02d", i)
				executionID := fmt.Sprintf("exec-%02d", i)
				workflowSvc.fileExecutions[fileID] = &moi.FileExecutionsResponse{Executions: []moi.FileExecutionSummary{{ExecutionID: executionID, Status: "running"}}}
				jobs[i] = KnowledgeBaseSourceJobRun{JobID: fmt.Sprintf("job-%02d", i), ModelID: 77, JobType: kbJobTypeRAGIngest, JobStatus: kbSourceJobPending, KBFileID: stringPtr(fileID)}
			}
			stats := &knowledgeBaseSourceJobEnrichStats{}
			startedAt := time.Now()
			got, err := (&semanticModelService{workflowService: workflowSvc}).enrichKnowledgeBaseSourceJobRunsFromLinkedJobsWithStats(context.Background(), nil, "ws-1", jobs, stats)
			elapsed := time.Since(startedAt)
			if err != nil {
				t.Fatalf("enrich jobs: %v", err)
			}
			wantPeak := int64(jobCount)
			if wantPeak > kbSourceListEnrichConcurrency {
				wantPeak = kbSourceListEnrichConcurrency
			}
			if peak.Load() != wantPeak {
				t.Fatalf("peak concurrency = %d, want %d", peak.Load(), wantPeak)
			}
			if stats.fileExecutionCalls.Load() != int64(jobCount) {
				t.Fatalf("file execution calls = %d, want %d", stats.fileExecutionCalls.Load(), jobCount)
			}
			for i := range got {
				if got[i].JobID != fmt.Sprintf("job-%02d", i) || ptrValue(got[i].WorkflowExecutionID) != fmt.Sprintf("exec-%02d", i) || got[i].JobStatus != kbSourceJobRunning {
					t.Fatalf("job[%d] = %+v", i, got[i])
				}
			}
			if jobCount == 32 && elapsed >= 400*time.Millisecond {
				t.Fatalf("32 jobs took %s, want bounded concurrent enrichment", elapsed)
			}
		})
	}
}

func TestIssue13313SourceJobEnrichmentCancelsSlowCallsAfterError(t *testing.T) {
	sentinel := errors.New("file execution lookup failed")
	slowStarted := make(chan struct{})
	slowCanceled := make(chan struct{})
	workflowSvc := &fakeSemanticModelWorkflowService{
		onListFileExecutions: func(ctx context.Context, fileID string) error {
			switch fileID {
			case "file-slow":
				close(slowStarted)
				<-ctx.Done()
				close(slowCanceled)
				return ctx.Err()
			case "file-error":
				<-slowStarted
				return sentinel
			default:
				return nil
			}
		},
	}
	jobs := []KnowledgeBaseSourceJobRun{
		{JobID: "job-slow", ModelID: 77, JobType: kbJobTypeRAGIngest, KBFileID: stringPtr("file-slow")},
		{JobID: "job-error", ModelID: 77, JobType: kbJobTypeRAGIngest, KBFileID: stringPtr("file-error")},
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err := (&semanticModelService{workflowService: workflowSvc}).enrichKnowledgeBaseSourceJobRunsFromLinkedJobs(ctx, nil, "ws-1", jobs)
	if !errors.Is(err, sentinel) {
		t.Fatalf("enrich error = %v, want sentinel", err)
	}
	select {
	case <-slowCanceled:
	default:
		t.Fatal("slow file execution lookup was not canceled")
	}
}

func TestIssue13313SourceJobEnrichmentPropagatesParentContextCancellation(t *testing.T) {
	callStarted := make(chan struct{})
	workflowSvc := &fakeSemanticModelWorkflowService{
		onListFileExecutions: func(ctx context.Context, _ string) error {
			close(callStarted)
			<-ctx.Done()
			return ctx.Err()
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	result := make(chan error, 1)
	go func() {
		_, err := (&semanticModelService{workflowService: workflowSvc}).enrichKnowledgeBaseSourceJobRunsFromLinkedJobs(ctx, nil, "ws-1", []KnowledgeBaseSourceJobRun{{
			JobID: "job-1", ModelID: 77, JobType: kbJobTypeRAGIngest, KBFileID: stringPtr("file-1"),
		}})
		result <- err
	}()
	select {
	case <-callStarted:
	case <-ctx.Done():
		t.Fatal("file execution lookup did not start before timeout")
	}
	cancel()
	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("enrich error = %v, want context canceled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("enrichment did not return after parent context cancellation")
	}
}

func TestIssue13313FileSourceMetadataUsesSingleBatchQuery(t *testing.T) {
	tenantDB, tenantMock := newSemanticModelTenantDB(t)
	records := make([]KnowledgeBaseSourceRecord, 10)
	args := make([]driver.Value, 10)
	rows := sqlmock.NewRows([]string{"file_id", "catalog_id", "database_id", "volume_id", "size", "updated_at", "catalog_name", "database_name", "volume_name", "file_path", "file_name"})
	for i := range records {
		fileID := fmt.Sprintf("file-%02d", i)
		args[i] = fileID
		rows.AddRow(fileID, int64(1), int64(2), int64(3), int64(100+i), int64(200+i), "catalog", "database", "volume", "", fileID+".pdf")
		records[i] = KnowledgeBaseSourceRecord{SourceID: fmt.Sprintf("source-%02d", i), RawVolumeID: 3, SourceType: kbSourceTypeCatalogFile, KBFileID: stringPtr(fileID)}
	}
	tenantMock.ExpectQuery("SELECT f.file_id, COALESCE.*WHERE f.file_id IN").WithArgs(args...).WillReturnRows(rows)
	stats := &knowledgeBaseSourceMetadataEnrichStats{}
	ctx := ctxutil.WithTenantDB(context.Background(), tenantDB)
	got, err := (&semanticModelService{}).enrichKnowledgeBaseSourceRecordsMetadataWithStats(ctx, nil, "ws-1", records, stats)
	if err != nil {
		t.Fatalf("enrich metadata: %v", err)
	}
	if len(got) != 10 || stats.fileQueryCalls.Load() != 1 {
		t.Fatalf("records = %d, file queries = %d", len(got), stats.fileQueryCalls.Load())
	}
	for i := range got {
		if ptrValue(got[i].DisplayName) != fmt.Sprintf("file-%02d.pdf", i) {
			t.Fatalf("record[%d] = %+v", i, got[i])
		}
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestKnowledgeBaseFileSourceMetadataUsesRecordedRawVolume(t *testing.T) {
	tenantDB, tenantMock := newSemanticModelTenantDB(t)
	tenantMock.ExpectQuery("SELECT f.file_id, COALESCE.*WHERE f.file_id IN").
		WithArgs("shared-file").
		WillReturnRows(sqlmock.NewRows([]string{"file_id", "catalog_id", "database_id", "volume_id", "size", "updated_at", "catalog_name", "database_name", "volume_name", "file_path", "file_name"}).
			AddRow("shared-file", int64(21), int64(31), int64(52), int64(4096), int64(200), "default", "importing_knowledge_base", "raw_document", "", "document.pdf").
			AddRow("shared-file", int64(21), int64(32), int64(41), int64(4096), int64(100), "default", "source_knowledge_base", "raw_document", "", "document.pdf"))
	ctx := ctxutil.WithTenantDB(context.Background(), tenantDB)
	got, err := (&semanticModelService{}).enrichKnowledgeBaseSourceRecordsMetadata(ctx, nil, "ws-1", []KnowledgeBaseSourceRecord{{
		SourceID:    "source-file",
		SourceType:  kbSourceTypeCatalogFile,
		RawVolumeID: 41,
		KBFileID:    stringPtr("shared-file"),
	}})
	if err != nil {
		t.Fatalf("enrich metadata: %v", err)
	}
	if gotPath := ptrValue(got[0].SourcePath); gotPath != `["default","source_knowledge_base","raw_document"]` {
		t.Fatalf("source path = %s", gotPath)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestKnowledgeBaseFileSourceMetadataMarksFailedWhenRecordedVolumeMissing(t *testing.T) {
	// Old rows may store domain raw_volume_id while the catalog file only lives on
	// another volume. Exact (file_id, volume_id) must not invent a different volume.
	tenantDB, tenantMock := newSemanticModelTenantDB(t)
	tenantMock.ExpectQuery("SELECT f.file_id, COALESCE.*WHERE f.file_id IN").
		WithArgs("catalog-only-file").
		WillReturnRows(sqlmock.NewRows([]string{"file_id", "catalog_id", "database_id", "volume_id", "size", "updated_at", "catalog_name", "database_name", "volume_name", "file_path", "file_name"}).
			AddRow("catalog-only-file", int64(2), int64(2), int64(2), int64(1039724), int64(300), "user_catalog", "user_db", "user_vol", "", "MatrixOne_Introduction.pdf"))
	ctx := ctxutil.WithTenantDB(context.Background(), tenantDB)
	got, err := (&semanticModelService{}).enrichKnowledgeBaseSourceRecordsMetadata(ctx, nil, "ws-1", []KnowledgeBaseSourceRecord{{
		SourceID:     "source-catalog",
		SourceType:   kbSourceTypeCatalogFile,
		RawVolumeID:  3, // stale domain raw volume; file is only on volume 2
		SourceFileID: stringPtr("catalog-only-file"),
		KBFileID:     stringPtr("catalog-only-file"),
		DisplayName:  stringPtr("catalog-only-file"),
	}})
	if err != nil {
		t.Fatalf("enrich metadata: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("records = %d", len(got))
	}
	if got[0].Status != kbSourceStatusFailed {
		t.Fatalf("status = %q, want failed when recorded volume is missing", got[0].Status)
	}
	if ptrValue(got[0].Error) == "" {
		t.Fatal("expected missing-location error on stale volume row")
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestKnowledgeBaseFileSourceMetadataExactMatchOnSourceVolume(t *testing.T) {
	// New catalog_file rows store the original volume id; list must exact-match it.
	tenantDB, tenantMock := newSemanticModelTenantDB(t)
	tenantMock.ExpectQuery("SELECT f.file_id, COALESCE.*WHERE f.file_id IN").
		WithArgs("catalog-only-file").
		WillReturnRows(sqlmock.NewRows([]string{"file_id", "catalog_id", "database_id", "volume_id", "size", "updated_at", "catalog_name", "database_name", "volume_name", "file_path", "file_name"}).
			AddRow("catalog-only-file", int64(2), int64(2), int64(2), int64(1039724), int64(300), "user_catalog", "user_db", "user_vol", "", "MatrixOne_Introduction.pdf"))
	ctx := ctxutil.WithTenantDB(context.Background(), tenantDB)
	got, err := (&semanticModelService{}).enrichKnowledgeBaseSourceRecordsMetadata(ctx, nil, "ws-1", []KnowledgeBaseSourceRecord{{
		SourceID:     "source-catalog",
		SourceType:   kbSourceTypeCatalogFile,
		RawVolumeID:  2,
		SourceFileID: stringPtr("catalog-only-file"),
		KBFileID:     stringPtr("catalog-only-file"),
		DisplayName:  stringPtr("client-wrong-name.pdf"),
	}})
	if err != nil {
		t.Fatalf("enrich metadata: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("records = %d", len(got))
	}
	if ptrValue(got[0].DisplayName) != "MatrixOne_Introduction.pdf" {
		t.Fatalf("display_name = %q, want catalog metadata", ptrValue(got[0].DisplayName))
	}
	if got[0].Status == kbSourceStatusFailed {
		t.Fatalf("status failed with error=%v", ptrValue(got[0].Error))
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestEnrichKnowledgeBaseFileSourceDisplayNameMatchesListAndIgnoresClientName(t *testing.T) {
	tenantDB, tenantMock := newSemanticModelTenantDB(t)
	tenantMock.ExpectQuery("SELECT f.file_id, COALESCE.*WHERE f.file_id IN").
		WithArgs("file-1").
		WillReturnRows(sqlmock.NewRows([]string{"file_id", "catalog_id", "database_id", "volume_id", "size", "updated_at", "catalog_name", "database_name", "volume_name", "file_path", "file_name"}).
			AddRow("file-1", int64(2), int64(2), int64(41), int64(10), int64(100), "c", "d", "v", "", "real-name.pdf"))
	ctx := ctxutil.WithTenantDB(context.Background(), tenantDB)
	record := KnowledgeBaseSourceRecord{
		SourceID:     "source-1",
		SourceType:   kbSourceTypeCatalogFile,
		RawVolumeID:  41,
		SourceFileID: stringPtr("file-1"),
		KBFileID:     stringPtr("file-1"),
		DisplayName:  stringPtr("client-wrong-name.pdf"),
		Status:       kbSourceStatusSucceeded,
	}
	if err := (&semanticModelService{}).enrichKnowledgeBaseFileSourceDisplayName(ctx, &record); err != nil {
		t.Fatalf("enrich detail: %v", err)
	}
	if ptrValue(record.DisplayName) != "real-name.pdf" {
		t.Fatalf("detail display_name = %q, want real-name.pdf", ptrValue(record.DisplayName))
	}
	if record.Status == kbSourceStatusFailed {
		t.Fatalf("detail marked failed: %v", ptrValue(record.Error))
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestEnrichKnowledgeBaseFileSourceDisplayNameMarksFailedWhenRecordedVolumeMissing(t *testing.T) {
	// Simulate pre-fix catalog_file row: domain raw volume, file only on another volume.
	tenantDB, tenantMock := newSemanticModelTenantDB(t)
	tenantMock.ExpectQuery("SELECT f.file_id, COALESCE.*WHERE f.file_id IN").
		WithArgs("file-1").
		WillReturnRows(sqlmock.NewRows([]string{"file_id", "catalog_id", "database_id", "volume_id", "size", "updated_at", "catalog_name", "database_name", "volume_name", "file_path", "file_name"}).
			AddRow("file-1", int64(2), int64(2), int64(41), int64(10), int64(100), "c", "d", "v", "", "real-name.pdf"))
	ctx := ctxutil.WithTenantDB(context.Background(), tenantDB)
	record := KnowledgeBaseSourceRecord{
		SourceID:     "source-old",
		SourceType:   kbSourceTypeCatalogFile,
		RawVolumeID:  12, // domain raw; not linked
		SourceFileID: stringPtr("file-1"),
		KBFileID:     stringPtr("file-1"),
		DisplayName:  stringPtr("stale-name.pdf"),
		Status:       kbSourceStatusSucceeded,
	}
	if err := (&semanticModelService{}).enrichKnowledgeBaseFileSourceDisplayName(ctx, &record); err != nil {
		t.Fatalf("enrich detail: %v", err)
	}
	if record.Status != kbSourceStatusFailed {
		t.Fatalf("status = %q, want failed for stale volume location", record.Status)
	}
	if ptrValue(record.Error) == "" {
		t.Fatal("expected error message for missing location")
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}
func TestIssue13313FileSourceMetadataSplitsOversizedQuery(t *testing.T) {
	tenantDB, tenantMock := newSemanticModelTenantDB(t)
	locations := make([]catalogFileSourceLocation, kbSourceFileMetadataQueryBatchSize+1)
	firstArgs := make([]driver.Value, kbSourceFileMetadataQueryBatchSize)
	for i := range locations {
		fileID := fmt.Sprintf("file-%04d", i)
		locations[i] = catalogFileSourceLocation{fileID: fileID, volumeID: 3}
		if i < len(firstArgs) {
			firstArgs[i] = fileID
		}
	}
	columns := []string{"file_id", "catalog_id", "database_id", "volume_id", "size", "updated_at", "catalog_name", "database_name", "volume_name", "file_path", "file_name"}
	tenantMock.ExpectQuery("SELECT f.file_id, COALESCE.*WHERE f.file_id IN").
		WithArgs(firstArgs...).
		WillReturnRows(sqlmock.NewRows(columns))
	tenantMock.ExpectQuery("SELECT f.file_id, COALESCE.*WHERE f.file_id IN").
		WithArgs(locations[len(locations)-1].fileID).
		WillReturnRows(sqlmock.NewRows(columns))
	ctx := ctxutil.WithTenantDB(context.Background(), tenantDB)
	metadata, err := currentCatalogFileMetadataBatch(ctx, locations)
	if err != nil {
		t.Fatalf("batch metadata: %v", err)
	}
	if len(metadata) != 0 {
		t.Fatalf("metadata = %d rows, want empty mock result", len(metadata))
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestIssue13313TableSourceMetadataReusesDatabaseStatsQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || !strings.HasPrefix(r.URL.Path, "/api/v1/workspaces/ws-1/tables/") {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		tableID := strings.TrimPrefix(r.URL.Path, "/api/v1/workspaces/ws-1/tables/")
		name := map[string]string{"1001": "orders", "1002": "customers"}[tableID]
		numericTableID := int64(1001)
		if tableID == "1002" {
			numericTableID = 1002
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"table":    map[string]any{"id": numericTableID, "name": name, "database_id": 31, "catalog_id": 21, "updated_at": 100},
			"database": map[string]any{"id": 31, "name": "sales", "catalog_id": 21},
			"catalog":  map[string]any{"id": 21, "name": "main"},
		})
	}))
	defer server.Close()
	client, err := moi.New(server.URL, "caller-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer client.Close()
	statsSQL, statsMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("stats sqlmock: %v", err)
	}
	statsMock.ExpectQuery("SELECT tbl_name, mo_table_rows\\(db_name, tbl_name\\), mo_table_size\\(db_name, tbl_name\\) FROM \\(VALUES ROW\\(\\?, \\?\\), ROW\\(\\?, \\?\\)\\) AS requested\\(db_name, tbl_name\\)").
		WithArgs("sales", "orders", "sales", "customers").
		WillReturnRows(sqlmock.NewRows([]string{"tbl_name", "row_count", "size_bytes"}).AddRow("orders", int64(10), int64(100)).AddRow("customers", int64(20), int64(200)))
	var openCalls atomic.Int64
	svc := &semanticModelService{openWorkspaceDB: func(_ context.Context, _ string, dbName string) (*sql.DB, error) {
		if dbName != "sales" {
			t.Fatalf("db name = %q", dbName)
		}
		openCalls.Add(1)
		return statsSQL, nil
	}}
	records := []KnowledgeBaseSourceRecord{
		{SourceID: "source-orders", SourceType: kbSourceTypeCatalogTable, KBTableID: int64Ptr(1001)},
		{SourceID: "source-customers", SourceType: kbSourceTypeCatalogTable, KBTableID: int64Ptr(1002)},
	}
	metadataStats := &knowledgeBaseSourceMetadataEnrichStats{}
	got, err := svc.enrichKnowledgeBaseSourceRecordsMetadataWithStats(context.Background(), client, "ws-1", records, metadataStats)
	if err != nil {
		t.Fatalf("enrich metadata: %v", err)
	}
	if len(got) != 2 || openCalls.Load() != 1 || metadataStats.tableStatusCalls.Load() != 1 {
		t.Fatalf("records = %d, open calls = %d, status calls = %d", len(got), openCalls.Load(), metadataStats.tableStatusCalls.Load())
	}
	if ptrValue(got[0].TableName) != "orders" || ptrValue(got[1].TableName) != "customers" || got[0].RowCount == nil || *got[0].RowCount != 10 || got[1].RowCount == nil || *got[1].RowCount != 20 {
		t.Fatalf("table records = %+v", got)
	}
	if err := statsMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("stats sql expectations: %v", err)
	}
}

func TestIssue11520VideoRAGUsesVideoTemplateForSingleFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id": 77, "name": "media kb", "description": "media", "tables": []any{},
				"files": map[string]any{"file_ids": []string{"video-1"}, "vector_table": "kb_vector", "embedding_model": "bge-m3"},
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()
	client, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer client.Close()

	tenantDB, tenantMock := newSemanticModelTenantDB(t)
	tenantMock.ExpectExec("UPDATE knowledge_base_source_job_runs").WillReturnResult(sqlmock.NewResult(1, 1))
	expectKnowledgeBaseDataDomain(tenantMock, 77, 12)
	tenantMock.ExpectExec("UPDATE knowledge_base_source_job_runs").WillReturnResult(sqlmock.NewResult(1, 1))
	workflowSvc := &fakeSemanticModelWorkflowService{runResult: &KnowledgeBaseWorkflowRunResult{ExecutionID: "exec-video-1"}}
	templateSvc := &fakeSemanticModelWorkflowTemplateService{}
	authorizer := &fakeSemanticModelActionAuthorizer{roleOut: "role-runtime"}
	svc := &semanticModelService{workflowService: workflowSvc, workflowTemplateService: templateSvc, actionAuthorizer: authorizer}
	ctx := ctxutil.WithTenantDB(context.Background(), tenantDB)
	record := KnowledgeBaseSourceRecord{
		SourceID: "source-video", ModelID: 77, RawVolumeID: 42, SourceType: kbSourceTypeLocalFile,
		SourceFileID: stringPtr("video-1"), KBFileID: stringPtr("video-1"), DisplayName: stringPtr("meeting.mp4"), CreatedBy: stringPtr("bff-principal"),
	}
	job := &KnowledgeBaseSourceJobRun{
		JobID: "job-rag", SourceID: record.SourceID, ModelID: 77, JobType: kbJobTypeRAGIngest,
		JobStatus: kbSourceJobPending, SourceFileID: stringPtr("video-1"), KBFileID: stringPtr("video-1"), RuntimeActorMOIUserID: stringPtr("moi-runtime-user"), RuntimeEffectiveRoleID: stringPtr("role-runtime"),
	}
	if err := svc.claimAndTriggerDeferredRAGIngestSourceJob(ctx, client, "ws-1", record, job, "user-1"); err != nil {
		t.Fatalf("claimAndTriggerDeferredRAGIngestSourceJob: %v", err)
	}
	if len(templateSvc.calls) != 1 || templateSvc.calls[0] != kbVideoRAGTemplateKey {
		t.Fatalf("template calls = %+v", templateSvc.calls)
	}
	if len(workflowSvc.deploys) != 1 || workflowSvc.deploys[0].ExecutionMode != "one_shot" {
		t.Fatalf("deploys = %+v", workflowSvc.deploys)
	}
	if len(workflowSvc.runs) != 1 || workflowSvc.runs[0].workflowID != knowledgeBaseMediaWorkflowID("ws-1", 77, kbVideoRAGTemplateKey) {
		t.Fatalf("runs = %+v", workflowSvc.runs)
	}
	sourceRef, _ := workflowSvc.runs[0].values["source_ref"].(map[string]any)
	if sourceRef["file_id"] != "video-1" || sourceRef["kind"] != "file" || sourceRef["volume_id"] != int64(42) || sourceRef["file_name"] != "meeting.mp4" {
		t.Fatalf("source_ref = %+v", sourceRef)
	}
	if !sameStringSlice(sourceRef["file_ids"].([]string), []string{"video-1"}) {
		t.Fatalf("source_ref file ids = %+v", sourceRef)
	}
	if _, exists := sourceRef["ids"]; exists {
		t.Fatalf("source_ref.ids must not be set: %+v", sourceRef)
	}
	if job.WorkflowExecutionID == nil || *job.WorkflowExecutionID != "exec-video-1" {
		t.Fatalf("job = %+v", job)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestFreezeKnowledgeBaseSourceJobRuntimePrincipal(t *testing.T) {
	t.Run("freezes moi user and verified role", func(t *testing.T) {
		ctx := ctxutil.WithUserID(context.Background(), "bff-principal")
		ctx = ctxutil.WithMoiUserID(ctx, "moi-user")
		ctx = ctxutil.WithCoreIAMRequest(ctx, ctxutil.CoreIAMRequestContext{VerifiedEffectiveRoleID: "role-create"})
		job := &KnowledgeBaseSourceJobRun{JobType: kbJobTypeRAGIngest}

		if err := freezeKnowledgeBaseSourceJobRuntimePrincipal(ctx, job); err != nil {
			t.Fatalf("freeze: %v", err)
		}
		if got := ptrValue(job.RuntimeActorMOIUserID); got != "moi-user" {
			t.Fatalf("runtime actor = %q, want moi-user", got)
		}
		if got := ptrValue(job.RuntimeEffectiveRoleID); got != "role-create" {
			t.Fatalf("runtime role = %q, want role-create", got)
		}
	})

	t.Run("rag create fails closed without verified role", func(t *testing.T) {
		ctx := ctxutil.WithMoiUserID(context.Background(), "moi-user")
		job := &KnowledgeBaseSourceJobRun{JobType: kbJobTypeRAGIngest}
		if err := freezeKnowledgeBaseSourceJobRuntimePrincipal(ctx, job); err == nil {
			t.Fatal("expected freeze error for missing role")
		}
	})

	t.Run("non-rag bind may omit role", func(t *testing.T) {
		ctx := ctxutil.WithMoiUserID(context.Background(), "moi-user")
		job := &KnowledgeBaseSourceJobRun{JobType: kbJobTypeLoad}
		if err := freezeKnowledgeBaseSourceJobRuntimePrincipal(ctx, job); err != nil {
			t.Fatalf("freeze load job: %v", err)
		}
		if got := ptrValue(job.RuntimeActorMOIUserID); got != "moi-user" {
			t.Fatalf("runtime actor = %q", got)
		}
		if job.RuntimeEffectiveRoleID != nil {
			t.Fatalf("role = %v, want nil for non-rag", job.RuntimeEffectiveRoleID)
		}
	})
}

func TestKnowledgeBaseWorkflowDispatchUsesFrozenPrincipalNotRequestCtx(t *testing.T) {
	// Deferred dispatch rehydrates job-frozen identity. Request/callback
	// principals must not override the create-time actor+role snapshot.
	const (
		modelID            = int64(77)
		creationAuditActor = "moi-user-a"
		creationRole       = "role-a"
		currentCaller      = "moi-user-b"
		currentRole        = "role-b"
		backendActor       = "bff-principal-b"
	)

	tenantDB, tenantMock := newSemanticModelTenantDB(t)
	record := KnowledgeBaseSourceRecord{
		SourceID:     "source-a",
		ModelID:      modelID,
		RawVolumeID:  12,
		SourceType:   kbSourceTypeLocalFile,
		SourceFileID: stringPtr("source-file"),
		KBFileID:     stringPtr("kb-file"),
		DisplayName:  stringPtr("document.pdf"),
	}
	job := &KnowledgeBaseSourceJobRun{
		JobID:                  "job-rag-a",
		SourceID:               record.SourceID,
		ModelID:                modelID,
		JobType:                kbJobTypeRAGIngest,
		JobStatus:              kbSourceJobPending,
		IdempotencyKey:         "idem-rag-a",
		SourceFileID:           record.SourceFileID,
		KBFileID:               record.KBFileID,
		RuntimeActorMOIUserID:  stringPtr(creationAuditActor),
		RuntimeEffectiveRoleID: stringPtr(creationRole),
	}
	workflowID := knowledgeBaseWorkflowID("ws-1", modelID)
	operationID := "workflow_trigger:" + workflowID
	tenantMock.ExpectExec("UPDATE knowledge_base_source_job_runs").
		WithArgs(
			kbSourceJobRunning,
			operationID,
			"source-file",
			"kb-file",
			backendActor,
			job.JobID,
			modelID,
			record.SourceID,
			kbJobTypeRAGIngest,
			kbSourceJobPending,
			kbSourceJobQueued,
			operationID,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))
	tenantMock.ExpectExec("UPDATE knowledge_base_source_job_runs").
		WithArgs(
			creationAuditActor,
			creationRole,
			false,
			kbSourceJobRunning,
			operationID,
			"exec-by-a",
			"source-file",
			"kb-file",
			nil,
			nil,
			0,
			nil,
			nil,
			backendActor,
			job.JobID,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	workflowSvc := &fakeSemanticModelWorkflowService{
		runResult: &KnowledgeBaseWorkflowRunResult{ExecutionID: "exec-by-a"},
	}
	authorizer := &fakeSemanticModelActionAuthorizer{roleOut: creationRole}
	svc := &semanticModelService{workflowService: workflowSvc, actionAuthorizer: authorizer}
	ctx := ctxutil.WithTenantDB(context.Background(), tenantDB)
	ctx = ctxutil.WithUserID(ctx, backendActor)
	// Callback / passive reconcile principal differs from frozen job principal.
	ctx = ctxutil.WithMoiUserID(ctx, currentCaller)
	ctx = ctxutil.WithCoreIAMRequest(ctx, ctxutil.CoreIAMRequestContext{VerifiedEffectiveRoleID: currentRole})

	if err := svc.claimAndTriggerDeferredRAGIngestSourceJob(ctx, nil, "ws-1", record, job, backendActor); err != nil {
		t.Fatalf("claimAndTriggerDeferredRAGIngestSourceJob: %v", err)
	}
	if len(workflowSvc.runs) != 1 {
		t.Fatalf("workflow runs = %+v", workflowSvc.runs)
	}
	if run := workflowSvc.runs[0]; run.moiUserID != creationAuditActor || run.effectiveRoleID != creationRole || run.workflowID != workflowID {
		t.Fatalf("workflow run identity = %+v, want frozen %s/%s", run, creationAuditActor, creationRole)
	}
	sourceRef, ok := workflowSvc.runs[0].values["source_ref"].(map[string]any)
	if !ok || sourceRef["file_id"] != "source-file" {
		t.Fatalf("workflow source_ref = %+v, want original source file", workflowSvc.runs[0].values["source_ref"])
	}
	fileIDs, ok := sourceRef["file_ids"].([]string)
	if !ok || !sameStringSlice(fileIDs, []string{"source-file"}) {
		t.Fatalf("workflow source_ref.file_ids = %+v, want original source file", sourceRef["file_ids"])
	}
	if got := ptrValue(job.RuntimeActorMOIUserID); got != creationAuditActor {
		t.Fatalf("runtime actor audit = %q, want creation actor %q", got, creationAuditActor)
	}
	if got := ptrValue(job.RuntimeEffectiveRoleID); got != creationRole {
		t.Fatalf("runtime role audit = %q, want creation role %q", got, creationRole)
	}
	if got := ptrValue(job.WorkflowExecutionID); got != "exec-by-a" {
		t.Fatalf("workflow execution = %q", got)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestClaimAndTriggerDeferredRAGFailsClosedWhenFrozenPrincipalMissing(t *testing.T) {
	workflowSvc := &fakeSemanticModelWorkflowService{
		runResult: &KnowledgeBaseWorkflowRunResult{ExecutionID: "must-not-run"},
	}
	svc := &semanticModelService{workflowService: workflowSvc}
	tenantDB, tenantMock := newSemanticModelTenantDB(t)
	tenantMock.ExpectExec("UPDATE knowledge_base_sources").WillReturnResult(sqlmock.NewResult(0, 1))
	tenantMock.ExpectExec("UPDATE knowledge_base_source_job_runs").WillReturnResult(sqlmock.NewResult(0, 1))
	ctx := ctxutil.WithTenantDB(context.Background(), tenantDB)
	// Request has a principal, but the job row does not — must not inherit request.
	ctx = ctxutil.WithMoiUserID(ctx, "moi-request")
	ctx = ctxutil.WithCoreIAMRequest(ctx, ctxutil.CoreIAMRequestContext{VerifiedEffectiveRoleID: "role-request"})
	record := KnowledgeBaseSourceRecord{
		SourceID: "source-a", ModelID: 77, RawVolumeID: 12,
		SourceType: kbSourceTypeLocalFile, SourceFileID: stringPtr("f"), KBFileID: stringPtr("f"),
	}
	job := &KnowledgeBaseSourceJobRun{
		JobID: "job-rag", SourceID: record.SourceID, ModelID: 77,
		JobType: kbJobTypeRAGIngest, JobStatus: kbSourceJobPending,
		SourceFileID: record.SourceFileID, KBFileID: record.KBFileID,
	}
	err := svc.claimAndTriggerDeferredRAGIngestSourceJob(ctx, nil, "ws-1", record, job, "actor")
	if err == nil || !errors.Is(err, errKnowledgeBaseSourceJobFailed) {
		t.Fatalf("error = %v, want fail-closed missing frozen principal", err)
	}
	if !strings.Contains(err.Error(), "missing frozen runtime principal") {
		t.Fatalf("error = %v, want missing frozen runtime principal", err)
	}
	if len(workflowSvc.runs) != 0 {
		t.Fatalf("workflow runs = %+v, want none", workflowSvc.runs)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestClaimAndTriggerDeferredCatalogRAGRetriesWhenCoreUnavailable(t *testing.T) {
	// Core transport/decision unavailability must leave the job pending for later
	// reconcile; permanent fail would strand catalog_file RAG after transient Core outages.
	workflowSvc := &fakeSemanticModelWorkflowService{
		runResult: &KnowledgeBaseWorkflowRunResult{ExecutionID: "must-not-run"},
	}
	authorizer := &fakeSemanticModelActionAuthorizer{err: iampep.ErrCoreUnavailable}
	svc := &semanticModelService{
		workflowService:  workflowSvc,
		actionAuthorizer: authorizer,
		volumeResolver:   &fakeSemanticModelCanonicalVolumeResolver{},
	}
	// No SQL fail marks expected — job stays pending.
	ctx := context.Background()
	record := KnowledgeBaseSourceRecord{
		SourceID: "source-catalog", ModelID: 77, RawVolumeID: 41,
		SourceType: kbSourceTypeCatalogFile, SourceFileID: stringPtr("catalog-file"), KBFileID: stringPtr("catalog-file"),
	}
	job := &KnowledgeBaseSourceJobRun{
		JobID: "job-rag-core-down", SourceID: record.SourceID, ModelID: 77,
		JobType: kbJobTypeRAGIngest, JobStatus: kbSourceJobPending,
		SourceFileID: record.SourceFileID, KBFileID: record.KBFileID,
		RuntimeActorMOIUserID:  stringPtr("moi-user-create-time"),
		RuntimeEffectiveRoleID: stringPtr("role-create-time"),
	}
	err := svc.claimAndTriggerDeferredRAGIngestSourceJob(ctx, nil, "ws-1", record, job, "bff-principal")
	if err == nil {
		t.Fatal("expected temporary unavailable error")
	}
	if errors.Is(err, errKnowledgeBaseSourceJobFailed) {
		t.Fatalf("error = %v, must not permanent-fail on Core unavailable", err)
	}
	if !errors.Is(err, iampep.ErrCoreUnavailable) {
		t.Fatalf("error = %v, want ErrCoreUnavailable", err)
	}
	if len(workflowSvc.runs) != 0 {
		t.Fatalf("workflow runs = %+v, want none", workflowSvc.runs)
	}
	if job.JobStatus != kbSourceJobPending {
		t.Fatalf("job status = %q, want still pending", job.JobStatus)
	}
}

func TestRehydrateKnowledgeBaseRAGJobPrincipalDoesNotRestorePrivilegeBypass(t *testing.T) {
	// Create-time owner bit is audit-only. Rehydrate must clear privilege facts so
	// ReauthorizeAction re-enters Core instead of short-circuiting on IsWorkspaceOwner.
	job := &KnowledgeBaseSourceJobRun{
		JobID:                   "job-owner-audit",
		JobType:                 kbJobTypeRAGIngest,
		RuntimeActorMOIUserID:   stringPtr("moi-user-owner"),
		RuntimeEffectiveRoleID:  stringPtr("role-owner-create"),
		RuntimeIsWorkspaceOwner: true,
	}
	// Ambient callback looks privileged; rehydrate must not keep that either.
	ctx := ctxutil.WithCoreIAMRequest(context.Background(), ctxutil.CoreIAMRequestContext{
		VerifiedEffectiveRoleID:  "role-callback",
		IsWorkspaceOwner:         true,
		WorkspaceAccessVerified:  true,
		BusinessActionAuthorized: true,
		AuthorizedActionFacts: []internalservice.AuthorizedActionFact{{
			ActionID:     "stale.action",
			ResourceType: "volume",
			ResourceID:   "1",
		}},
	})

	out, err := rehydrateKnowledgeBaseRAGJobPrincipal(ctx, job)
	if err != nil {
		t.Fatalf("rehydrate: %v", err)
	}
	trusted, ok := ctxutil.CoreIAMRequestFrom(out)
	if !ok {
		t.Fatal("expected CoreIAM request on rehydrated ctx")
	}
	if got := strings.TrimSpace(trusted.VerifiedEffectiveRoleID); got != "role-owner-create" {
		t.Fatalf("VerifiedEffectiveRoleID = %q, want job role", got)
	}
	if trusted.IsWorkspaceOwner {
		t.Fatal("IsWorkspaceOwner must stay false after rehydrate (no privilege bypass)")
	}
	if trusted.BusinessActionAuthorized {
		t.Fatal("BusinessActionAuthorized must stay false after rehydrate")
	}
	if trusted.WorkspaceAccessVerified {
		t.Fatal("WorkspaceAccessVerified must stay false after rehydrate")
	}
	if len(trusted.AuthorizedActionFacts) != 0 {
		t.Fatalf("AuthorizedActionFacts = %v, want cleared", trusted.AuthorizedActionFacts)
	}
	if strings.TrimSpace(trusted.RoleCandidateID) != "" {
		t.Fatalf("RoleCandidateID = %q, want empty", trusted.RoleCandidateID)
	}
	if got := ctxutil.MoiUserIDFrom(out); got != "moi-user-owner" {
		t.Fatalf("MoiUserID = %q, want frozen actor", got)
	}
}

func TestClaimAndTriggerDeferredCatalogRAGReauthsEvenWhenJobFrozeOwner(t *testing.T) {
	// Frozen RuntimeIsWorkspaceOwner must not skip semantic_model.use / volume.read.
	// Deny on volume.read must permanent-fail (live Core path), not inherit owner allow.
	workflowSvc := &fakeSemanticModelWorkflowService{
		runResult: &KnowledgeBaseWorkflowRunResult{ExecutionID: "must-not-run"},
	}
	authorizer := &fakeSemanticModelActionAuthorizer{
		errByAction: map[string]error{
			"volume.read": iampep.ErrCoreDecisionDeny,
		},
		roleOut: "role-owner-create",
	}
	tenantDB, tenantMock := newSemanticModelTenantDB(t)
	tenantMock.ExpectExec("UPDATE knowledge_base_sources").WillReturnResult(sqlmock.NewResult(0, 1))
	tenantMock.ExpectExec("UPDATE knowledge_base_source_job_runs").WillReturnResult(sqlmock.NewResult(0, 1))
	svc := &semanticModelService{
		workflowService:  workflowSvc,
		actionAuthorizer: authorizer,
		volumeResolver:   &fakeSemanticModelCanonicalVolumeResolver{},
	}
	ctx := ctxutil.WithTenantDB(context.Background(), tenantDB)
	record := KnowledgeBaseSourceRecord{
		SourceID: "source-catalog", ModelID: 77, RawVolumeID: 41,
		SourceType: kbSourceTypeCatalogFile, SourceFileID: stringPtr("catalog-file"), KBFileID: stringPtr("catalog-file"),
	}
	job := &KnowledgeBaseSourceJobRun{
		JobID: "job-rag-owner-audit", SourceID: record.SourceID, ModelID: 77,
		JobType: kbJobTypeRAGIngest, JobStatus: kbSourceJobPending,
		SourceFileID: record.SourceFileID, KBFileID: record.KBFileID,
		RuntimeActorMOIUserID:   stringPtr("moi-user-owner"),
		RuntimeEffectiveRoleID:  stringPtr("role-owner-create"),
		RuntimeIsWorkspaceOwner: true,
	}

	err := svc.claimAndTriggerDeferredRAGIngestSourceJob(ctx, nil, "ws-1", record, job, "bff-principal")
	if err == nil || !errors.Is(err, errKnowledgeBaseSourceJobFailed) {
		t.Fatalf("error = %v, want permanent fail on volume.read deny", err)
	}
	if len(workflowSvc.runs) != 0 {
		t.Fatalf("workflow runs = %+v, want none", workflowSvc.runs)
	}
	// Must have attempted both reauth steps (use then volume.read) under rehydrated principal.
	if len(authorizer.calls) < 2 {
		t.Fatalf("authorizer calls = %+v, want semantic_model.use then volume.read", authorizer.calls)
	}
	if authorizer.calls[0].actionID != "semantic_model.use" {
		t.Fatalf("first reauth = %+v, want semantic_model.use", authorizer.calls[0])
	}
	if authorizer.calls[1].actionID != "volume.read" {
		t.Fatalf("second reauth = %+v, want volume.read", authorizer.calls[1])
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestClaimAndTriggerDeferredCatalogRAGRetriesWhenResolveRootUnavailable(t *testing.T) {
	// ResolveRoot UNAVAILABLE/TIMEOUT must not permanent-fail for deferred catalog
	// RAG: root resolution runs before volume.read. Job stays pending.
	workflowSvc := &fakeSemanticModelWorkflowService{
		runResult: &KnowledgeBaseWorkflowRunResult{ExecutionID: "must-not-run"},
	}
	authorizer := &fakeSemanticModelActionAuthorizer{roleOut: "role-ordinary"}
	volumeResolver := &fakeSemanticModelCanonicalVolumeResolver{
		err: &moi.Error{Code: common.ErrorCode_UNAVAILABLE, Message: "resolve root temporarily unavailable"},
	}
	svc := &semanticModelService{
		workflowService:  workflowSvc,
		actionAuthorizer: authorizer,
		volumeResolver:   volumeResolver,
	}
	// No tenant DB / fail UPDATEs: retryable path must not mark source/job failed.
	ctx := context.Background()
	record := KnowledgeBaseSourceRecord{
		SourceID: "source-catalog", ModelID: 77, RawVolumeID: 41,
		SourceType: kbSourceTypeCatalogFile, SourceFileID: stringPtr("catalog-file"), KBFileID: stringPtr("catalog-file"),
	}
	job := &KnowledgeBaseSourceJobRun{
		JobID: "job-rag-resolve-down", SourceID: record.SourceID, ModelID: 77,
		JobType: kbJobTypeRAGIngest, JobStatus: kbSourceJobPending,
		SourceFileID: record.SourceFileID, KBFileID: record.KBFileID,
		RuntimeActorMOIUserID:  stringPtr("moi-user-ordinary"),
		RuntimeEffectiveRoleID: stringPtr("role-ordinary"),
	}

	err := svc.claimAndTriggerDeferredRAGIngestSourceJob(ctx, nil, "ws-1", record, job, "bff-principal")
	if err == nil {
		t.Fatal("expected temporary unavailable error from resolve root")
	}
	if errors.Is(err, errKnowledgeBaseSourceJobFailed) {
		t.Fatalf("error = %v, must not permanent-fail on resolve-root UNAVAILABLE", err)
	}
	if !moi.IsCode(err, common.ErrorCode_UNAVAILABLE) {
		t.Fatalf("error = %v, want moi UNAVAILABLE", err)
	}
	if !strings.Contains(err.Error(), "temporarily unavailable") {
		t.Fatalf("error = %v, want temporarily unavailable wrapper", err)
	}
	if len(workflowSvc.runs) != 0 {
		t.Fatalf("workflow runs = %+v, want none", workflowSvc.runs)
	}
	if len(authorizer.calls) != 1 || authorizer.calls[0].actionID != "semantic_model.use" {
		t.Fatalf("reauth calls = %+v, want only semantic_model.use before resolve fails", authorizer.calls)
	}
	if len(volumeResolver.calls) != 1 || volumeResolver.calls[0].volumeID != 41 {
		t.Fatalf("resolve calls = %+v, want raw volume 41 once", volumeResolver.calls)
	}
	if job.JobStatus != kbSourceJobPending {
		t.Fatalf("job status = %q, want still pending", job.JobStatus)
	}
}

func TestClaimAndTriggerDeferredCatalogRAGFailsClosedWithoutVolumeResolver(t *testing.T) {
	// Missing canonical resolver must not authorize RawVolumeID as a fallback.
	workflowSvc := &fakeSemanticModelWorkflowService{
		runResult: &KnowledgeBaseWorkflowRunResult{ExecutionID: "must-not-run"},
	}
	authorizer := &fakeSemanticModelActionAuthorizer{roleOut: "role-ordinary"}
	svc := &semanticModelService{workflowService: workflowSvc, actionAuthorizer: authorizer}
	tenantDB, tenantMock := newSemanticModelTenantDB(t)
	tenantMock.ExpectExec("UPDATE knowledge_base_sources").WillReturnResult(sqlmock.NewResult(0, 1))
	tenantMock.ExpectExec("UPDATE knowledge_base_source_job_runs").WillReturnResult(sqlmock.NewResult(0, 1))
	ctx := ctxutil.WithTenantDB(context.Background(), tenantDB)
	record := KnowledgeBaseSourceRecord{
		SourceID: "source-catalog", ModelID: 77, RawVolumeID: 41,
		SourceType: kbSourceTypeCatalogFile, SourceFileID: stringPtr("f"), KBFileID: stringPtr("f"),
	}
	job := &KnowledgeBaseSourceJobRun{
		JobID: "job-rag", SourceID: record.SourceID, ModelID: 77,
		JobType: kbJobTypeRAGIngest, JobStatus: kbSourceJobQueued,
		SourceFileID: record.SourceFileID, KBFileID: record.KBFileID,
		RuntimeActorMOIUserID:  stringPtr("moi-user"),
		RuntimeEffectiveRoleID: stringPtr("role-ordinary"),
	}
	err := svc.claimAndTriggerDeferredRAGIngestSourceJob(ctx, nil, "ws-1", record, job, "actor")
	if err == nil || !errors.Is(err, errKnowledgeBaseSourceJobFailed) {
		t.Fatalf("error = %v, want fail-closed without volume resolver", err)
	}
	if !strings.Contains(err.Error(), "canonical volume resolver is not configured") {
		t.Fatalf("error = %v, want missing resolver message", err)
	}
	if len(workflowSvc.runs) != 0 {
		t.Fatalf("workflow runs = %+v, want none", workflowSvc.runs)
	}
	if len(authorizer.calls) != 1 || authorizer.calls[0].actionID != "semantic_model.use" {
		t.Fatalf("reauth calls = %+v, want only use (no volume.read fallback)", authorizer.calls)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestIssue11520KnowledgeBaseCatalogFileExtensions(t *testing.T) {
	for _, name := range []string{"image.tiff", "audio.m4a", "video.avi", "video.webm", "legacy.xls", "report.XLSX"} {
		if err := validateKnowledgeBaseCatalogFileExtension(name); err != nil {
			t.Fatalf("validate %s: %v", name, err)
		}
	}
	if err := validateKnowledgeBaseCatalogFileExtension("archive.zip"); err == nil {
		t.Fatal("zip should remain unsupported")
	}
}

func expectDeleteKnowledgeBaseModelSegments(mock sqlmock.Sqlmock, modelID int64) {
	for _, table := range []string{"knowledge_base_chunk_recall_stats", "knowledge_base_segments", "knowledge_base_segment_versions"} {
		mock.ExpectExec("DELETE FROM " + table).
			WithArgs(modelID).
			WillReturnResult(sqlmock.NewResult(0, 1))
	}
}

func expectAppendSourcesTransactionLock(mock sqlmock.Sqlmock, modelID int64) {
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT model_id\\s+FROM knowledge_base_data_domains\\s+WHERE model_id = \\?\\s+FOR UPDATE").
		WithArgs(modelID).
		WillReturnRows(sqlmock.NewRows([]string{"model_id"}).AddRow(modelID))
}

func expectKnowledgeBaseSourceJobRunUpsertMiss(mock sqlmock.Sqlmock) {
	mock.ExpectExec("UPDATE knowledge_base_source_job_runs").
		WillReturnResult(sqlmock.NewResult(0, 0))
}

func TestValidateKnowledgeBaseStructuredTableConfigRequiresKnowledgeBaseDatabase(t *testing.T) {
	if err := validateKnowledgeBaseStructuredTableConfig(`{"database_id":11,"new_table":true,"create_table":{"name":"orders","tableColumn":[{"column":"id","dataType":"INT"}]}}`, 11); err != nil {
		t.Fatalf("validate single table_config: %v", err)
	}
	normalized, err := normalizeKnowledgeBaseStructuredTableConfig(`{"new_table":true,"create_table":{"name":"orders","tableColumn":[{"column":"id","dataType":"INT"}]}}`, 11)
	if err != nil {
		t.Fatalf("normalize single table_config without database_id: %v", err)
	}
	var single model.TableConfig
	if err := json.Unmarshal([]byte(normalized), &single); err != nil {
		t.Fatalf("parse normalized single table_config: %v", err)
	}
	if single.DatabaseId != 11 {
		t.Fatalf("normalized database_id = %d, want 11", single.DatabaseId)
	}
	if err := validateKnowledgeBaseStructuredTableConfig(`{"database_id":12,"new_table":true,"create_table":{"name":"orders","tableColumn":[{"column":"id","dataType":"INT"}]}}`, 11); err == nil || !strings.Contains(err.Error(), "knowledge base database 11") {
		t.Fatalf("validate external database error = %v", err)
	}
	if err := validateKnowledgeBaseStructuredTableConfig(`{"multi_sheet":true,"tables":[{"database_id":11,"new_table":true,"create_table":{"name":"orders","tableColumn":[{"column":"id","dataType":"INT"}]}}]}`, 11); err != nil {
		t.Fatalf("validate multi table_config: %v", err)
	}
	normalized, err = normalizeKnowledgeBaseStructuredTableConfig(`{"multi_sheet":true,"tables":[{"new_table":true,"create_table":{"name":"orders","tableColumn":[{"column":"id","dataType":"INT"}]}}]}`, 11)
	if err != nil {
		t.Fatalf("normalize multi table_config without database_id: %v", err)
	}
	var multi model.MultiTableConfig
	if err := json.Unmarshal([]byte(normalized), &multi); err != nil {
		t.Fatalf("parse normalized multi table_config: %v", err)
	}
	if len(multi.Tables) != 1 || multi.Tables[0].DatabaseId != 11 {
		t.Fatalf("normalized multi database_id = %+v, want 11", multi.Tables)
	}
	if err := validateKnowledgeBaseStructuredTableConfig(`{"multi_sheet":true,"tables":[{"database_id":12,"new_table":true,"create_table":{"name":"orders","tableColumn":[{"column":"id","dataType":"INT"}]}}]}`, 11); err == nil || !strings.Contains(err.Error(), "knowledge base database 11") {
		t.Fatalf("validate multi external database error = %v", err)
	}
	if err := validateKnowledgeBaseStructuredTableConfig(`{"database_id":11,"new_table":true,"create_table":{"name":"orders"}}`, 11); err == nil || !strings.Contains(err.Error(), "tableColumn is required") {
		t.Fatalf("validate empty table columns error = %v", err)
	}
}

func TestValidateCreateSemanticModelSourcesRequiresCatalogFileVolumeID(t *testing.T) {
	if err := validateCreateSemanticModelSources([]CreateSemanticModelSourceRequest{{
		SourceType: kbSourceTypeCatalogFile,
		FileID:     "catalog-file",
		VolumeID:   41,
	}}); err != nil {
		t.Fatalf("validate catalog_file with volume_id: %v", err)
	}
	if err := validateCreateSemanticModelSources([]CreateSemanticModelSourceRequest{{
		SourceType: kbSourceTypeCatalogFile,
		FileID:     "catalog-file",
	}}); err == nil || !strings.Contains(err.Error(), "volume_id") {
		t.Fatalf("validate missing volume_id error = %v", err)
	}
}

func TestValidateCreateSemanticModelSourcesRejectsDuplicateCatalogFileIDs(t *testing.T) {
	// Same file under two volumes: volume conflict (identity is file_id only).
	err := validateCreateSemanticModelSources([]CreateSemanticModelSourceRequest{
		{SourceType: kbSourceTypeCatalogFile, FileID: "shared-file", VolumeID: 41},
		{SourceType: kbSourceTypeCatalogFile, FileID: "shared-file", VolumeID: 52},
	})
	if !IsServiceError(err, ErrCodeBadRequest) {
		t.Fatalf("error = %v, want bad request", err)
	}
	if !i18n.IsKey(err, i18n.KeySessionSemanticModelCatalogFileVolumeConflict) {
		t.Fatalf("error key = %v, want catalog_file_volume_conflict", err)
	}
	// Same file twice under the same volume: duplicate in request.
	err = validateCreateSemanticModelSources([]CreateSemanticModelSourceRequest{
		{SourceType: kbSourceTypeCatalogFile, FileID: "shared-file", VolumeID: 41},
		{SourceType: kbSourceTypeCatalogFile, FileID: "shared-file", VolumeID: 41},
	})
	if !IsServiceError(err, ErrCodeBadRequest) {
		t.Fatalf("error = %v, want bad request", err)
	}
	if !i18n.IsKey(err, i18n.KeySessionSemanticModelCatalogFileDuplicateInRequest) {
		t.Fatalf("error key = %v, want catalog_file_duplicate_in_request", err)
	}
	zhCtx := semanticModelServiceTestContext(i18n.LocaleZhCN.String())
	msg, ok := i18n.Message(zhCtx, err)
	if !ok || msg != "同一文件在一次添加中只能选择一次，请移除重复项后重试" {
		t.Fatalf("zh message = %q, ok = %v", msg, ok)
	}
	enCtx := semanticModelServiceTestContext(i18n.LocaleEnUS.String())
	msg, ok = i18n.Message(enCtx, err)
	if !ok || msg != "The same file can only be added once in a single request. Remove the duplicate and try again" {
		t.Fatalf("en message = %q, ok = %v", msg, ok)
	}
	// Distinct files under different volumes remain valid.
	if err := validateCreateSemanticModelSources([]CreateSemanticModelSourceRequest{
		{SourceType: kbSourceTypeCatalogFile, FileID: "file-a", VolumeID: 41},
		{SourceType: kbSourceTypeCatalogFile, FileID: "file-b", VolumeID: 52},
	}); err != nil {
		t.Fatalf("validate distinct catalog files: %v", err)
	}
}

func TestValidateCreateSemanticModelSourcesRequiresUploadedLocalFileID(t *testing.T) {
	if err := validateCreateSemanticModelSources([]CreateSemanticModelSourceRequest{{
		SourceType: kbSourceTypeLocalFile,
		FileName:   "local.txt",
		FileID:     "local-file-id",
	}}); err != nil {
		t.Fatalf("validate local file_id source: %v", err)
	}
	if err := validateCreateSemanticModelSources([]CreateSemanticModelSourceRequest{{
		SourceType: kbSourceTypeLocalFile,
		FileName:   "local.txt",
	}}); err == nil || !strings.Contains(err.Error(), "file_id") {
		t.Fatalf("validate missing file_id error = %v", err)
	}
}

func TestSemanticModelServiceRejectsDeprecatedContentBase64(t *testing.T) {
	var body struct {
		Sources []CreateSemanticModelSourceRequest `json:"sources"`
	}
	if err := json.Unmarshal([]byte(`{"sources":[{"source_type":"local_file","file_name":"local.txt","file_id":"uploaded-file","content_base64":"YQ=="}]}`), &body); err != nil {
		t.Fatalf("decode request: %v", err)
	}
	if len(body.Sources) != 1 || len(body.Sources[0].DeprecatedContentBase64) == 0 {
		t.Fatalf("decoded sources = %+v, want deprecated field preserved for validation", body.Sources)
	}

	svc := &semanticModelService{}
	ctx := semanticModelServiceTestContext(i18n.LocaleZhCN.String())
	tests := []struct {
		name string
		call func() error
	}{
		{
			name: "create",
			call: func() error {
				_, err := svc.CreateModelWithSources(ctx, CreateSemanticModelWithSourcesRequest{Name: "kb_docs", Sources: body.Sources})
				return err
			},
		},
		{
			name: "append",
			call: func() error {
				_, err := svc.AppendModelSources(ctx, AppendSemanticModelSourcesRequest{ModelID: 42, Sources: body.Sources})
				return err
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.call()
			if !IsServiceError(err, ErrCodeBadRequest) {
				t.Fatalf("error = %v, want bad request", err)
			}
			msg, ok := i18n.Message(ctx, err)
			if !ok || msg != "sources[0].content_base64 已不再支持，请先上传文件并使用 file_id" {
				t.Fatalf("localized message = %q, ok = %v", msg, ok)
			}
		})
	}
}

func TestValidateSemanticModelSourceSelectionsRejectsNormalizedOrDuplicateValues(t *testing.T) {
	tests := []SemanticModelSourceSelectionRequest{
		{Kind: kbSelectionKindVolumeFiles, VolumeID: 42, SelectedFileIDs: []string{" file-1 "}},
		{Kind: kbSelectionKindVolumeFiles, VolumeID: 42, SelectedFileIDs: []string{"file-1", "file-1"}},
		{Kind: kbSelectionKindVolumeFiles, VolumeID: 42, AllSelected: true, Filters: SemanticModelSourceSelectionFilters{FileExt: []string{".PDF"}}},
		{Kind: kbSelectionKindVolumeFiles, VolumeID: 42, AllSelected: true, Filters: SemanticModelSourceSelectionFilters{FileExt: []string{"pdf", "pdf"}}},
		{Kind: kbSelectionKindDatabaseTables, DatabaseID: 11, SelectedTableIDs: []int64{1001, 1001}},
	}
	for _, selection := range tests {
		if err := validateSemanticModelSourceSelections([]SemanticModelSourceSelectionRequest{selection}); err == nil {
			t.Fatalf("validateSemanticModelSourceSelections(%+v) error = nil, want invalid input", selection)
		}
	}
}

func TestHasDocumentParsingSourcesOnlyMatchesDocumentFileSources(t *testing.T) {
	if hasDocumentParsingSources([]CreateSemanticModelSourceRequest{{SourceType: kbSourceTypeLocalFile, UploadKind: kbLocalUploadKindStructured}}) {
		t.Fatal("structured local file should not require RAG workflow")
	}
	if hasDocumentParsingSources([]CreateSemanticModelSourceRequest{{SourceType: kbSourceTypeCatalogTable, TableID: 1001}}) {
		t.Fatal("catalog table should not require RAG workflow")
	}
	if !hasDocumentParsingSources([]CreateSemanticModelSourceRequest{{SourceType: kbSourceTypeLocalFile}}) {
		t.Fatal("unstructured local file should require RAG workflow")
	}
	if !hasDocumentParsingSources([]CreateSemanticModelSourceRequest{{SourceType: kbSourceTypeCatalogFile, FileID: "file-1"}}) {
		t.Fatal("catalog file should require RAG workflow")
	}
}

func TestSemanticModelSourcesSkipsFilesCoveredBySelectedVolume(t *testing.T) {
	model := &moi.SemanticModel{
		ID:    9,
		Files: json.RawMessage(`{"file_ids":["file-1"],"parents":["volume-volume-1"],"volume_ids":["volume-1"]}`),
	}

	items, err := semanticModelSourcesFromModel(model)
	if err != nil {
		t.Fatalf("semanticModelSourcesFromModel: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("items = %+v, want only selected volume", items)
	}
	if items[0].SourceType != SemanticModelSourceTypeVolume || items[0].ResourceID != "volume-1" {
		t.Fatalf("item = %+v, want volume-1", items[0])
	}
}

func TestSemanticModelServiceDeleteSourceRemovesFileRelationWithoutDataDomain(t *testing.T) {
	var updatedFiles json.RawMessage
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":          77,
				"name":        "kb",
				"description": "docs",
				"files": map[string]any{
					"file_ids":           []string{"kb-file", "other-file"},
					"vector_table":       "kb_text_idx",
					"image_vector_table": "kb_image_idx",
				},
				"tables": []map[string]any{},
			})
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			var req struct {
				Files json.RawMessage `json:"files"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode semantic update: %v", err)
			}
			updatedFiles = req.Files
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 77, "name": "kb"})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()
	fileSvc := &fakeSemanticModelFileService{}
	svc := newSemanticModelTestServiceWithKnowledgeBaseRuntimeDependencies(t, server.URL, systemClient, nil, fileSvc, nil, nil, nil)

	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	tenantMock.ExpectBegin()
	expectKnowledgeBaseDataDomainLockMissing(tenantMock, 77)
	expectKnowledgeBaseSourceDeleteLocks(tenantMock, 77, "source-file-1")
	tenantMock.ExpectQuery("SELECT .*source_id.*FROM knowledge_base_sources").
		WithArgs(int64(77), "source-file-1").
		WillReturnRows(sqlmock.NewRows([]string{
			"source_id", "model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "source_type",
			"source_file_id", "source_table_id", "kb_file_id", "kb_table_id", "display_name", "source_path", "db_name", "table_name",
			"status", "error", "enabled", "expires_at", "tags", "force_enabled_after_expiry", "segment_version_id", "index_version",
		}).AddRow("source-file-1", int64(77), int64(3), int64(11), int64(12), int64(13), kbSourceTypeCatalogFile, "source-file", nil, "kb-file", nil, "doc.pdf", nil, nil, nil, kbSourceStatusSucceeded, nil, true, nil, nil, false, nil, nil))
	expectDeleteKnowledgeBaseSourceSegments(tenantMock, 77, "source-file-1")
	expectMarkKnowledgeBaseSourceRemoved(tenantMock, 77, "source-file-1", "user-1")
	tenantMock.ExpectExec("DELETE FROM knowledge_base_source_job_runs").
		WithArgs(int64(77), "source-file-1").
		WillReturnResult(sqlmock.NewResult(0, 2))
	tenantMock.ExpectCommit()

	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)

	if err := svc.DeleteSource(ctx, DeleteSemanticModelSourceParams{ModelID: 77, SourceID: "source-file-1"}); err != nil {
		t.Fatalf("DeleteSource: %v", err)
	}
	if len(fileSvc.deleted) != 0 {
		t.Fatalf("file delete calls = %+v", fileSvc.deleted)
	}
	var files struct {
		FileIDs []string `json:"file_ids"`
	}
	if err := json.Unmarshal(updatedFiles, &files); err != nil {
		t.Fatalf("unmarshal updated files: %v", err)
	}
	if !sameStringSet(files.FileIDs, []string{"other-file"}) {
		t.Fatalf("updated semantic files = %s", string(updatedFiles))
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceDeleteSourceRejectsActiveJobBeforeCoreMutation(t *testing.T) {
	coreRequests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		coreRequests++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()
	svc := newSemanticModelTestServiceWithKnowledgeBaseRuntimeDependencies(t, server.URL, systemClient, nil, nil, nil, nil, nil)

	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	tenantMock.ExpectBegin()
	expectKnowledgeBaseDataDomainLock(tenantMock, 77)
	tenantMock.ExpectQuery("SELECT source_id.*FROM knowledge_base_sources.*FOR UPDATE").
		WithArgs(int64(77), "source-file-1").
		WillReturnRows(sqlmock.NewRows([]string{"source_id"}).AddRow("source-file-1"))
	tenantMock.ExpectQuery("SELECT job_id,.*CASE WHEN job_status.*FROM knowledge_base_source_job_runs.*FOR UPDATE").
		WithArgs(kbSourceJobRunning, int(kbSourceJobClaimLease/time.Second), int64(77), "source-file-1").
		WillReturnRows(sqlmock.NewRows([]string{"job_id", "active_claim"}).AddRow("job-file-1", 1))
	tenantMock.ExpectRollback()

	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)
	err = svc.DeleteSource(ctx, DeleteSemanticModelSourceParams{ModelID: 77, SourceID: "source-file-1"})
	if !IsServiceError(err, ErrCodeConflict) {
		t.Fatalf("DeleteSource error = %v, want conflict", err)
	}
	if coreRequests != 0 {
		t.Fatalf("core requests = %d, want 0", coreRequests)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceDeleteSourceRetriesWhenSemanticScopeAlreadyRemoved(t *testing.T) {
	var updatedFiles json.RawMessage
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":          77,
				"name":        "kb",
				"description": "docs",
				"files": map[string]any{
					"file_ids":           []string{"other-file"},
					"vector_table":       "kb_text_idx",
					"image_vector_table": "kb_image_idx",
				},
				"tables": []map[string]any{},
			})
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			var req struct {
				Files json.RawMessage `json:"files"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode semantic update: %v", err)
			}
			updatedFiles = req.Files
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 77, "name": "kb"})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()
	fileSvc := &fakeSemanticModelFileService{}
	svc := newSemanticModelTestServiceWithKnowledgeBaseRuntimeDependencies(t, server.URL, systemClient, nil, fileSvc, nil, nil, nil)

	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	tenantMock.ExpectBegin()
	expectKnowledgeBaseDataDomainLock(tenantMock, 77)
	expectKnowledgeBaseSourceDeleteLocks(tenantMock, 77, "source-file-1")
	tenantMock.ExpectQuery("SELECT .*source_id.*FROM knowledge_base_sources").
		WithArgs(int64(77), "source-file-1").
		WillReturnRows(sqlmock.NewRows([]string{
			"source_id", "model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "source_type",
			"source_file_id", "source_table_id", "kb_file_id", "kb_table_id", "display_name", "source_path", "db_name", "table_name",
			"status", "error", "enabled", "expires_at", "tags", "force_enabled_after_expiry", "segment_version_id", "index_version",
		}).AddRow("source-file-1", int64(77), int64(3), int64(11), int64(12), int64(13), kbSourceTypeCatalogFile, "source-file", nil, "kb-file", nil, "doc.pdf", nil, nil, nil, kbSourceStatusSucceeded, nil, true, nil, nil, false, nil, nil))
	expectDeleteKnowledgeBaseSourceSegments(tenantMock, 77, "source-file-1")
	expectMarkKnowledgeBaseSourceRemoved(tenantMock, 77, "source-file-1", "user-1")
	tenantMock.ExpectExec("DELETE FROM knowledge_base_source_job_runs").
		WithArgs(int64(77), "source-file-1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	tenantMock.ExpectCommit()

	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)

	if err := svc.DeleteSource(ctx, DeleteSemanticModelSourceParams{ModelID: 77, SourceID: "source-file-1"}); err != nil {
		t.Fatalf("DeleteSource: %v", err)
	}
	if len(fileSvc.deleted) != 0 {
		t.Fatalf("file delete calls = %+v", fileSvc.deleted)
	}
	var files struct {
		FileIDs []string `json:"file_ids"`
	}
	if err := json.Unmarshal(updatedFiles, &files); err != nil {
		t.Fatalf("unmarshal updated files: %v", err)
	}
	if !sameStringSet(files.FileIDs, []string{"other-file"}) {
		t.Fatalf("updated semantic files = %s", string(updatedFiles))
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceDeleteSourceDoesNotDeleteCatalogFileWhenSemanticUpdateFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":          77,
				"name":        "kb",
				"description": "docs",
				"files": map[string]any{
					"file_ids":           []string{"kb-file", "other-file"},
					"vector_table":       "kb_text_idx",
					"image_vector_table": "kb_image_idx",
				},
				"tables": []map[string]any{},
			})
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			http.Error(w, `{"code":"ErrUpdate","error":"semantic update failed"}`, http.StatusInternalServerError)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()
	fileSvc := &fakeSemanticModelFileService{}
	svc := newSemanticModelTestServiceWithKnowledgeBaseRuntimeDependencies(t, server.URL, systemClient, nil, fileSvc, nil, nil, nil)

	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	tenantMock.ExpectBegin()
	expectKnowledgeBaseDataDomainLock(tenantMock, 77)
	expectKnowledgeBaseSourceDeleteLocks(tenantMock, 77, "source-file-1")
	tenantMock.ExpectQuery("SELECT .*source_id.*FROM knowledge_base_sources").
		WithArgs(int64(77), "source-file-1").
		WillReturnRows(sqlmock.NewRows([]string{
			"source_id", "model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "source_type",
			"source_file_id", "source_table_id", "kb_file_id", "kb_table_id", "display_name", "source_path", "db_name", "table_name",
			"status", "error", "enabled", "expires_at", "tags", "force_enabled_after_expiry", "segment_version_id", "index_version",
		}).AddRow("source-file-1", int64(77), int64(3), int64(11), int64(12), int64(13), kbSourceTypeCatalogFile, "source-file", nil, "kb-file", nil, "doc.pdf", nil, nil, nil, kbSourceStatusSucceeded, nil, true, nil, nil, false, nil, nil))
	tenantMock.ExpectRollback()

	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)

	err = svc.DeleteSource(ctx, DeleteSemanticModelSourceParams{ModelID: 77, SourceID: "source-file-1"})
	if err == nil || !strings.Contains(err.Error(), "update semantic model source scope") {
		t.Fatalf("DeleteSource error = %v", err)
	}
	if len(fileSvc.deleted) != 0 {
		t.Fatalf("file delete calls = %+v, want none", fileSvc.deleted)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceDeleteSourceRemovesTableRelationWithoutDeletingCatalogTable(t *testing.T) {
	var updatedTables json.RawMessage
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":     77,
				"name":   "kb",
				"files":  map[string]any{"file_ids": []string{}},
				"tables": []map[string]any{{"db_name": "catalog_db", "table_names": []string{"orders", "customers"}}},
			})
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			var req struct {
				Tables json.RawMessage `json:"tables"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode semantic update: %v", err)
			}
			updatedTables = req.Tables
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 77, "name": "kb"})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()
	dataDomainSvc := &fakeSemanticModelDataDomainService{}
	svc := newSemanticModelTestServiceWithKnowledgeBaseRuntimeDependencies(t, server.URL, systemClient, dataDomainSvc, nil, nil, nil, nil)

	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	tenantMock.ExpectBegin()
	expectKnowledgeBaseDataDomainLock(tenantMock, 77)
	expectKnowledgeBaseSourceDeleteLocks(tenantMock, 77, "source-table-1")
	tenantMock.ExpectQuery("SELECT .*source_id.*FROM knowledge_base_sources").
		WithArgs(int64(77), "source-table-1").
		WillReturnRows(sqlmock.NewRows([]string{
			"source_id", "model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "source_type",
			"source_file_id", "source_table_id", "kb_file_id", "kb_table_id", "display_name", "source_path", "db_name", "table_name",
			"status", "error", "enabled", "expires_at", "tags", "force_enabled_after_expiry", "segment_version_id", "index_version",
		}).AddRow("source-table-1", int64(77), int64(3), int64(11), int64(12), int64(13), kbSourceTypeCatalogTable, nil, int64(1001), nil, int64(2001), "orders", nil, "catalog_db", "orders", kbSourceStatusSucceeded, nil, nil, nil, nil, false, nil, nil))
	expectDeleteKnowledgeBaseSourceSegments(tenantMock, 77, "source-table-1")
	expectMarkKnowledgeBaseSourceRemoved(tenantMock, 77, "source-table-1", "user-1")
	tenantMock.ExpectExec("DELETE FROM knowledge_base_source_job_runs").
		WithArgs(int64(77), "source-table-1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	tenantMock.ExpectCommit()

	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)

	if err := svc.DeleteSource(ctx, DeleteSemanticModelSourceParams{ModelID: 77, SourceID: "source-table-1"}); err != nil {
		t.Fatalf("DeleteSource: %v", err)
	}
	if len(dataDomainSvc.calls) != 0 {
		t.Fatalf("data domain calls = %+v, want none", dataDomainSvc.calls)
	}
	var tables []semanticModelTableSource
	if err := json.Unmarshal(updatedTables, &tables); err != nil {
		t.Fatalf("unmarshal updated tables: %v", err)
	}
	if len(tables) != 1 || tables[0].DBName != "catalog_db" || !sameStringSet(tables[0].TableNames, []string{"customers"}) {
		t.Fatalf("updated semantic tables = %s", string(updatedTables))
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceDeleteSourceRemovesIncompleteFailedTableRelation(t *testing.T) {
	var updatedTables json.RawMessage
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":     77,
				"name":   "kb",
				"files":  map[string]any{"file_ids": []string{}},
				"tables": []map[string]any{},
			})
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			var req struct {
				Tables json.RawMessage `json:"tables"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode semantic update: %v", err)
			}
			updatedTables = req.Tables
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 77, "name": "kb"})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()
	dataDomainSvc := &fakeSemanticModelDataDomainService{}
	svc := newSemanticModelTestServiceWithKnowledgeBaseRuntimeDependencies(t, server.URL, systemClient, dataDomainSvc, nil, nil, nil, nil)

	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	tenantMock.ExpectBegin()
	expectKnowledgeBaseDataDomainLock(tenantMock, 77)
	expectKnowledgeBaseSourceDeleteLocks(tenantMock, 77, "source-table-1")
	tenantMock.ExpectQuery("SELECT .*source_id.*FROM knowledge_base_sources").
		WithArgs(int64(77), "source-table-1").
		WillReturnRows(sqlmock.NewRows([]string{
			"source_id", "model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "source_type",
			"source_file_id", "source_table_id", "kb_file_id", "kb_table_id", "display_name", "source_path", "db_name", "table_name",
			"status", "error", "enabled", "expires_at", "tags", "force_enabled_after_expiry", "segment_version_id", "index_version",
		}).AddRow("source-table-1", int64(77), int64(3), int64(11), int64(14), int64(13), kbSourceTypeCatalogTable, nil, nil, nil, nil, "structured_orders", nil, nil, nil, kbSourceStatusFailed, "previous reconcile failed", nil, nil, nil, false, nil, nil))
	expectDeleteKnowledgeBaseSourceSegments(tenantMock, 77, "source-table-1")
	expectMarkKnowledgeBaseSourceRemoved(tenantMock, 77, "source-table-1", "user-1")
	tenantMock.ExpectExec("DELETE FROM knowledge_base_source_job_runs").
		WithArgs(int64(77), "source-table-1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	tenantMock.ExpectCommit()

	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)

	if err := svc.DeleteSource(ctx, DeleteSemanticModelSourceParams{ModelID: 77, SourceID: "source-table-1"}); err != nil {
		t.Fatalf("DeleteSource: %v", err)
	}
	if len(dataDomainSvc.calls) != 0 {
		t.Fatalf("data domain calls = %+v, want none", dataDomainSvc.calls)
	}
	var tables []semanticModelTableSource
	if err := json.Unmarshal(updatedTables, &tables); err != nil {
		t.Fatalf("unmarshal updated tables: %v", err)
	}
	if len(tables) != 0 {
		t.Fatalf("updated semantic tables = %s, want empty", string(updatedTables))
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceDeleteSourceDoesNotDeleteOriginalCatalogTable(t *testing.T) {
	cases := []struct {
		name      string
		kbTableID *int64
	}{
		{name: "legacy source table only"},
		{name: "legacy source table misrecorded as kb table", kbTableID: int64Ptr(1001)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var updatedTables json.RawMessage
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch {
				case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
					_ = json.NewEncoder(w).Encode(map[string]any{
						"id":     77,
						"name":   "kb",
						"files":  map[string]any{"file_ids": []string{}},
						"tables": []map[string]any{{"db_name": "external_db", "table_names": []string{"orders", "customers"}}},
					})
				case r.Method == http.MethodPut && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
					var req struct {
						Tables json.RawMessage `json:"tables"`
					}
					if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
						t.Fatalf("decode semantic update: %v", err)
					}
					updatedTables = req.Tables
					_ = json.NewEncoder(w).Encode(map[string]any{"id": 77, "name": "kb"})
				default:
					t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
				}
			}))
			defer server.Close()

			systemClient, err := moi.New(server.URL, "system-key")
			if err != nil {
				t.Fatalf("moi.New: %v", err)
			}
			defer systemClient.Close()
			dataDomainSvc := &fakeSemanticModelDataDomainService{}
			svc := newSemanticModelTestServiceWithKnowledgeBaseRuntimeDependencies(t, server.URL, systemClient, dataDomainSvc, nil, nil, nil, nil)

			tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
			if err != nil {
				t.Fatalf("tenant sqlmock: %v", err)
			}
			defer tenantSQL.Close()
			tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
			if err != nil {
				t.Fatalf("open tenant gorm: %v", err)
			}
			tenantMock.ExpectBegin()
			expectKnowledgeBaseDataDomainLock(tenantMock, 77)
			expectKnowledgeBaseSourceDeleteLocks(tenantMock, 77, "source-table-1")
			tenantMock.ExpectQuery("SELECT .*source_id.*FROM knowledge_base_sources").
				WithArgs(int64(77), "source-table-1").
				WillReturnRows(sqlmock.NewRows([]string{
					"source_id", "model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "source_type",
					"source_file_id", "source_table_id", "kb_file_id", "kb_table_id", "display_name", "source_path", "db_name", "table_name",
					"status", "error", "enabled", "expires_at", "tags", "force_enabled_after_expiry", "segment_version_id", "index_version",
				}).AddRow("source-table-1", int64(77), int64(3), int64(11), int64(12), int64(13), kbSourceTypeCatalogTable, nil, int64(1001), nil, int64Value(tc.kbTableID), "orders", nil, "external_db", "orders", kbSourceStatusSucceeded, nil, nil, nil, nil, false, nil, nil))
			expectDeleteKnowledgeBaseSourceSegments(tenantMock, 77, "source-table-1")
			expectMarkKnowledgeBaseSourceRemoved(tenantMock, 77, "source-table-1", "user-1")
			tenantMock.ExpectExec("DELETE FROM knowledge_base_source_job_runs").
				WithArgs(int64(77), "source-table-1").
				WillReturnResult(sqlmock.NewResult(0, 1))
			tenantMock.ExpectCommit()

			ctx := ctxutil.WithUID(context.Background(), "user-1")
			ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
			ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
			ctx = ctxutil.WithTenantDB(ctx, tenantDB)

			if err := svc.DeleteSource(ctx, DeleteSemanticModelSourceParams{ModelID: 77, SourceID: "source-table-1"}); err != nil {
				t.Fatalf("DeleteSource: %v", err)
			}
			if len(dataDomainSvc.calls) != 0 {
				t.Fatalf("data domain calls = %+v, want none", dataDomainSvc.calls)
			}
			var tables []semanticModelTableSource
			if err := json.Unmarshal(updatedTables, &tables); err != nil {
				t.Fatalf("unmarshal updated tables: %v", err)
			}
			if len(tables) != 1 || tables[0].DBName != "external_db" || !sameStringSet(tables[0].TableNames, []string{"customers"}) {
				t.Fatalf("updated semantic tables = %s", string(updatedTables))
			}
			if err := tenantMock.ExpectationsWereMet(); err != nil {
				t.Fatalf("tenant sql expectations: %v", err)
			}
		})
	}
}

func TestSemanticModelServicePartialDataDomainFailureRetainsResourcesForCreateRollback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
	}))
	defer server.Close()

	dataDomainSvc := &fakeSemanticModelDataDomainService{
		databaseID: 11,
		volumeIDs:  []int64{12},
		volumeErrs: map[string]error{"processed": errors.New("processed volume unavailable")},
	}
	workflowSvc := &fakeSemanticModelWorkflowService{}
	svc := newSemanticModelTestServiceWithKnowledgeBaseRuntimeDependencies(t, server.URL, nil, dataDomainSvc, nil, nil, nil, workflowSvc)
	impl := svc.(*semanticModelService)
	db, mock := newSemanticModelTenantDB(t)
	ctx := semanticModelServiceTestContext(i18n.LocaleZhCN.String())
	ctx = ctxutil.WithTenantDB(ctx, db)

	// Create retains provisioning ownership and the in-memory IDs for its outer rollback.
	expectClaimKnowledgeBaseDataDomainProvision(mock, 77)
	domain := &KnowledgeBaseDataDomain{ModelID: 77, CatalogID: 3, EnsureStatus: kbEnsureStatusFailed}
	_, err := impl.completeCreateModelWithSources(ctx, nil, "ws-1", &moi.SemanticModel{ID: 77}, domain, CreateSemanticModelWithSourcesRequest{
		Name: "kb_docs", Description: "docs",
	}, "42", "user-1")
	if err == nil || !strings.Contains(err.Error(), "processed volume unavailable") {
		t.Fatalf("completeCreateModelWithSources error = %v", err)
	}
	if domain.DatabaseID != 11 || domain.RawVolumeID != 12 || domain.ProcessedVolumeID != 0 || domain.EnsureStatus != kbEnsureStatusProvisioning {
		t.Fatalf("partial provisioning domain = %+v", domain)
	}
	if !sameStringSet(dataDomainSvc.calls, []string{
		"database:kb_docs", "raw_document:Knowledge base document raw source files", "processed:Knowledge base processed files",
	}) {
		t.Fatalf("data domain calls = %+v", dataDomainSvc.calls)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceDeleteModelDeletesKnowledgeBaseRowsAndCatalogResources(t *testing.T) {
	var deletedSemanticModel bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			deletedSemanticModel = true
			_ = json.NewEncoder(w).Encode(map[string]any{"deleted": true})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()
	dataDomainSvc := &fakeSemanticModelDataDomainService{}
	fileSvc := &fakeSemanticModelFileService{}
	workflowSvc := &fakeSemanticModelWorkflowService{}
	svc := newSemanticModelTestServiceWithKnowledgeBaseRuntimeDependencies(t, server.URL, systemClient, dataDomainSvc, fileSvc, &fakeSemanticModelLocalFileImportService{}, nil, workflowSvc)

	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	expectKnowledgeBaseCatalogResourceLookup(tenantMock, 77, 3, 11, 12, 13, 12, 14)
	expectDeleteKnowledgeBaseModelSegments(tenantMock, 77)
	for _, table := range []string{"knowledge_base_source_job_runs", "knowledge_base_sources", "knowledge_base_raw_volumes", "knowledge_base_source_jobs", "knowledge_base_data_domains"} {
		tenantMock.ExpectExec("DELETE FROM " + table).WithArgs(int64(77)).WillReturnResult(sqlmock.NewResult(1, 1))
	}

	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)
	if err := svc.DeleteModel(ctx, 77); err != nil {
		t.Fatalf("DeleteModel: %v", err)
	}
	if !deletedSemanticModel {
		t.Fatal("semantic model was not deleted")
	}
	if !sameStringSlice(workflowSvc.deletes, expectedKnowledgeBaseWorkflowIDs("ws-1", 77)) {
		t.Fatalf("workflow deletes = %+v", workflowSvc.deletes)
	}
	wantDataDomainCalls := []string{"delete-volume:12", "delete-volume:13", "delete-volume:14", "delete-database:11"}
	if !sameStringSet(dataDomainSvc.calls, wantDataDomainCalls) {
		t.Fatalf("data domain delete calls = %+v, want %+v", dataDomainSvc.calls, wantDataDomainCalls)
	}
	if len(fileSvc.deleted) != 0 {
		t.Fatalf("file delete calls = %+v, want none", fileSvc.deleted)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceDeleteModelContinuesWhenKnowledgeBaseWorkflowAlreadyMissing(t *testing.T) {
	var deletedSemanticModel bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			deletedSemanticModel = true
			_ = json.NewEncoder(w).Encode(map[string]any{"deleted": true})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()
	dataDomainSvc := &fakeSemanticModelDataDomainService{}
	fileSvc := &fakeSemanticModelFileService{}
	workflowSvc := &fakeSemanticModelWorkflowService{deleteErr: &moi.Error{Code: common.ErrorCode_NOT_FOUND, Message: "workflow not found"}}
	svc := newSemanticModelTestServiceWithKnowledgeBaseRuntimeDependencies(t, server.URL, systemClient, dataDomainSvc, fileSvc, &fakeSemanticModelLocalFileImportService{}, nil, workflowSvc)

	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	expectKnowledgeBaseCatalogResourceLookup(tenantMock, 77, 3, 11, 12, 13, 12)
	expectDeleteKnowledgeBaseModelSegments(tenantMock, 77)
	for _, table := range []string{"knowledge_base_source_job_runs", "knowledge_base_sources", "knowledge_base_raw_volumes", "knowledge_base_source_jobs", "knowledge_base_data_domains"} {
		tenantMock.ExpectExec("DELETE FROM " + table).WithArgs(int64(77)).WillReturnResult(sqlmock.NewResult(1, 1))
	}

	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)
	if err := svc.DeleteModel(ctx, 77); err != nil {
		t.Fatalf("DeleteModel: %v", err)
	}
	if !deletedSemanticModel {
		t.Fatal("semantic model was not deleted")
	}
	if !sameStringSlice(workflowSvc.deletes, expectedKnowledgeBaseWorkflowIDs("ws-1", 77)) {
		t.Fatalf("workflow deletes = %+v", workflowSvc.deletes)
	}
	wantDataDomainCalls := []string{"delete-volume:12", "delete-volume:13", "delete-database:11"}
	if !sameStringSet(dataDomainSvc.calls, wantDataDomainCalls) {
		t.Fatalf("data domain delete calls = %+v, want %+v", dataDomainSvc.calls, wantDataDomainCalls)
	}
	if len(fileSvc.deleted) != 0 {
		t.Fatalf("file delete calls = %+v, want none", fileSvc.deleted)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceDeleteModelFailsFastWhenTenantDBMissing(t *testing.T) {
	var deletedSemanticModel bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodDelete && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77" {
			deletedSemanticModel = true
			_ = json.NewEncoder(w).Encode(map[string]any{"deleted": true})
			return
		}
		t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()
	dataDomainSvc := &fakeSemanticModelDataDomainService{}
	fileSvc := &fakeSemanticModelFileService{}
	workflowSvc := &fakeSemanticModelWorkflowService{}
	svc := newSemanticModelTestServiceWithKnowledgeBaseRuntimeDependencies(t, server.URL, systemClient, dataDomainSvc, fileSvc, &fakeSemanticModelLocalFileImportService{}, nil, workflowSvc)

	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	err = svc.DeleteModel(ctx, 77)
	if err == nil || !strings.Contains(err.Error(), "tenant db is required") {
		t.Fatalf("DeleteModel error = %v", err)
	}
	if deletedSemanticModel {
		t.Fatal("semantic model must not be deleted when tenant DB is missing")
	}
	if len(workflowSvc.validates) != 0 {
		t.Fatalf("workflow validates = %+v, want none", workflowSvc.validates)
	}
	if len(workflowSvc.deletes) != 0 {
		t.Fatalf("workflow deletes = %+v, want none", workflowSvc.deletes)
	}
	if len(dataDomainSvc.calls) != 0 {
		t.Fatalf("data domain delete calls = %+v, want none", dataDomainSvc.calls)
	}
	if len(fileSvc.deleted) != 0 {
		t.Fatalf("file delete calls = %+v, want none", fileSvc.deleted)
	}
}

func TestSemanticModelServiceDeleteModelDoesNotDeleteSemanticModelWhenKnowledgeBaseWorkflowIsActive(t *testing.T) {
	var deletedSemanticModel bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodDelete && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77" {
			deletedSemanticModel = true
			_ = json.NewEncoder(w).Encode(map[string]any{"deleted": true})
			return
		}
		t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()
	dataDomainSvc := &fakeSemanticModelDataDomainService{}
	fileSvc := &fakeSemanticModelFileService{}
	workflowSvc := &fakeSemanticModelWorkflowService{validateErr: &moi.Error{Code: common.ErrorCode_ALREADY_EXISTS, Message: "workflow has active execution exec-1 in status triggered"}}
	svc := newSemanticModelTestServiceWithKnowledgeBaseRuntimeDependencies(t, server.URL, systemClient, dataDomainSvc, fileSvc, &fakeSemanticModelLocalFileImportService{}, nil, workflowSvc)

	tenantDB, tenantMock := newSemanticModelTenantDB(t)
	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)
	err = svc.DeleteModel(ctx, 77)
	if !IsServiceError(err, ErrCodeConflict) || !strings.Contains(err.Error(), i18n.KeySessionKnowledgeBaseWorkflowDeleteConflict.String()) {
		t.Fatalf("DeleteModel error = %v", err)
	}
	if deletedSemanticModel {
		t.Fatal("semantic model must not be deleted when workflow deletion is blocked")
	}
	if len(workflowSvc.validates) != 1 {
		t.Fatalf("workflow validates = %+v, want one validation", workflowSvc.validates)
	}
	if len(workflowSvc.deletes) != 0 {
		t.Fatalf("workflow deletes = %+v, want none", workflowSvc.deletes)
	}
	if len(dataDomainSvc.calls) != 0 {
		t.Fatalf("data domain delete calls = %+v, want none", dataDomainSvc.calls)
	}
	if len(fileSvc.deleted) != 0 {
		t.Fatalf("file delete calls = %+v, want none", fileSvc.deleted)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceDeleteModelDoesNotDeleteSemanticModelWhenKnowledgeBaseWorkflowHasDispatchJob(t *testing.T) {
	var deletedSemanticModel bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodDelete && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77" {
			deletedSemanticModel = true
			_ = json.NewEncoder(w).Encode(map[string]any{"deleted": true})
			return
		}
		t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()
	dataDomainSvc := &fakeSemanticModelDataDomainService{}
	fileSvc := &fakeSemanticModelFileService{}
	workflowSvc := &fakeSemanticModelWorkflowService{validateErr: &moi.Error{Code: common.ErrorCode_ALREADY_EXISTS, Message: "workflow has active volume dispatch job job-1 in status waiting"}}
	svc := newSemanticModelTestServiceWithKnowledgeBaseRuntimeDependencies(t, server.URL, systemClient, dataDomainSvc, fileSvc, &fakeSemanticModelLocalFileImportService{}, nil, workflowSvc)

	tenantDB, tenantMock := newSemanticModelTenantDB(t)
	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)
	err = svc.DeleteModel(ctx, 77)
	if !IsServiceError(err, ErrCodeConflict) || !strings.Contains(err.Error(), i18n.KeySessionKnowledgeBaseWorkflowDeleteConflict.String()) {
		t.Fatalf("DeleteModel error = %v", err)
	}
	if deletedSemanticModel {
		t.Fatal("semantic model must not be deleted when workflow dispatch job blocks deletion")
	}
	if len(workflowSvc.validates) != 1 {
		t.Fatalf("workflow validates = %+v, want one validation", workflowSvc.validates)
	}
	if len(workflowSvc.deletes) != 0 {
		t.Fatalf("workflow deletes = %+v, want none", workflowSvc.deletes)
	}
	if len(dataDomainSvc.calls) != 0 {
		t.Fatalf("data domain delete calls = %+v, want none", dataDomainSvc.calls)
	}
	if len(fileSvc.deleted) != 0 {
		t.Fatalf("file delete calls = %+v, want none", fileSvc.deleted)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceDeleteModelKeepsSemanticModelWhenKnowledgeBaseRowCleanupFails(t *testing.T) {
	var deletedSemanticModel bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodDelete && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77" {
			deletedSemanticModel = true
			_ = json.NewEncoder(w).Encode(map[string]any{"deleted": true})
			return
		}
		t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()
	dataDomainSvc := &fakeSemanticModelDataDomainService{}
	svc := newSemanticModelTestServiceWithKnowledgeBaseRuntimeDependencies(t, server.URL, systemClient, dataDomainSvc, &fakeSemanticModelFileService{}, &fakeSemanticModelLocalFileImportService{}, nil, &fakeSemanticModelWorkflowService{})

	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	expectKnowledgeBaseCatalogResourceLookup(tenantMock, 77, 3, 11, 12, 13, 12)
	tenantMock.ExpectExec("DELETE FROM knowledge_base_chunk_recall_stats").WithArgs(int64(77)).WillReturnError(errors.New("delete rows failed"))
	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)
	err = svc.DeleteModel(ctx, 77)
	if err == nil || !strings.Contains(err.Error(), "delete knowledge base records") || !strings.Contains(err.Error(), "delete rows failed") {
		t.Fatalf("DeleteModel error = %v", err)
	}
	if deletedSemanticModel {
		t.Fatal("semantic model must not be deleted when knowledge base row cleanup fails")
	}
	wantDataDomainCalls := []string{"delete-volume:12", "delete-volume:13", "delete-database:11"}
	if !sameStringSet(dataDomainSvc.calls, wantDataDomainCalls) {
		t.Fatalf("data domain delete calls = %+v, want %+v", dataDomainSvc.calls, wantDataDomainCalls)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceDeleteModelKeepsRowsWhenCatalogCleanupFails(t *testing.T) {
	var deletedSemanticModel bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodDelete && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77" {
			deletedSemanticModel = true
			_ = json.NewEncoder(w).Encode(map[string]any{"deleted": true})
			return
		}
		t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()
	dataDomainSvc := &fakeSemanticModelDataDomainService{deleteErr: errors.New("catalog cleanup failed")}
	svc := newSemanticModelTestServiceWithKnowledgeBaseRuntimeDependencies(t, server.URL, systemClient, dataDomainSvc, &fakeSemanticModelFileService{}, &fakeSemanticModelLocalFileImportService{}, nil, &fakeSemanticModelWorkflowService{})

	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	expectKnowledgeBaseCatalogResourceLookup(tenantMock, 77, 3, 11, 12, 13, 12)

	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)
	err = svc.DeleteModel(ctx, 77)
	if err == nil || !strings.Contains(err.Error(), "delete knowledge base catalog resources") || !strings.Contains(err.Error(), "catalog cleanup failed") {
		t.Fatalf("DeleteModel error = %v", err)
	}
	if deletedSemanticModel {
		t.Fatal("semantic model must not be deleted when catalog cleanup fails")
	}
	if !sameStringSet(dataDomainSvc.calls, []string{"delete-volume:12"}) {
		t.Fatalf("data domain delete calls = %+v", dataDomainSvc.calls)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceDeleteModelContinuesWhenCatalogResourcesAlreadyMissing(t *testing.T) {
	var deletedSemanticModel bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodDelete && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77" {
			deletedSemanticModel = true
			_ = json.NewEncoder(w).Encode(map[string]any{"deleted": true})
			return
		}
		t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()
	dataDomainSvc := &fakeSemanticModelDataDomainService{
		deleteVolumeErr:   fmt.Errorf("delete volume 12: %w", &moi.Error{Code: common.ErrorCode_NOT_FOUND, Message: "Volume not found"}),
		deleteDatabaseErr: fmt.Errorf("delete database 11: %w", &moi.Error{Code: common.ErrorCode_CATALOG_NOT_FOUND, Message: "Catalog not found"}),
	}
	workflowSvc := &fakeSemanticModelWorkflowService{}
	svc := newSemanticModelTestServiceWithKnowledgeBaseRuntimeDependencies(t, server.URL, systemClient, dataDomainSvc, &fakeSemanticModelFileService{}, &fakeSemanticModelLocalFileImportService{}, nil, workflowSvc)

	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	expectKnowledgeBaseCatalogResourceLookup(tenantMock, 77, 3, 11, 12, 13, 12)
	expectDeleteKnowledgeBaseModelSegments(tenantMock, 77)
	for _, table := range []string{"knowledge_base_source_job_runs", "knowledge_base_sources", "knowledge_base_raw_volumes", "knowledge_base_source_jobs", "knowledge_base_data_domains"} {
		tenantMock.ExpectExec("DELETE FROM " + table).WithArgs(int64(77)).WillReturnResult(sqlmock.NewResult(1, 1))
	}

	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)
	if err := svc.DeleteModel(ctx, 77); err != nil {
		t.Fatalf("DeleteModel: %v", err)
	}
	if !deletedSemanticModel {
		t.Fatal("semantic model was not deleted")
	}
	if !sameStringSlice(workflowSvc.deletes, expectedKnowledgeBaseWorkflowIDs("ws-1", 77)) {
		t.Fatalf("workflow deletes = %+v", workflowSvc.deletes)
	}
	wantDataDomainCalls := []string{"delete-volume:12", "delete-volume:13", "delete-database:11"}
	if !sameStringSet(dataDomainSvc.calls, wantDataDomainCalls) {
		t.Fatalf("data domain delete calls = %+v, want %+v", dataDomainSvc.calls, wantDataDomainCalls)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceDeleteModelSkipsCatalogCleanupWhenDataDomainMissing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodDelete && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77" {
			_ = json.NewEncoder(w).Encode(map[string]any{"deleted": true})
			return
		}
		t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()
	svc := newSemanticModelTestServiceWithKnowledgeBaseRuntimeDependencies(t, server.URL, systemClient, nil, nil, nil, nil, &fakeSemanticModelWorkflowService{})

	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	tenantMock.ExpectQuery("SELECT model_id, catalog_id, database_id, raw_volume_id, processed_volume_id, ensure_status, last_ensure_error, last_checked_at").
		WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows([]string{
			"model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "ensure_status", "last_ensure_error", "last_checked_at",
		}))
	expectDeleteKnowledgeBaseModelSegments(tenantMock, 77)
	for _, table := range []string{"knowledge_base_source_job_runs", "knowledge_base_sources", "knowledge_base_raw_volumes", "knowledge_base_source_jobs", "knowledge_base_data_domains"} {
		tenantMock.ExpectExec("DELETE FROM " + table).WithArgs(int64(77)).WillReturnResult(sqlmock.NewResult(1, 1))
	}

	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)
	if err := svc.DeleteModel(ctx, 77); err != nil {
		t.Fatalf("DeleteModel: %v", err)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceDeleteModelReportsSemanticDeleteFailureAfterResourceCleanup(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodDelete && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77" {
			http.Error(w, `{"code":"ErrDelete","error":"semantic delete failed"}`, http.StatusInternalServerError)
			return
		}
		t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()
	dataDomainSvc := &fakeSemanticModelDataDomainService{}
	fileSvc := &fakeSemanticModelFileService{}
	workflowSvc := &fakeSemanticModelWorkflowService{}
	svc := newSemanticModelTestServiceWithKnowledgeBaseRuntimeDependencies(t, server.URL, systemClient, dataDomainSvc, fileSvc, &fakeSemanticModelLocalFileImportService{}, nil, workflowSvc)

	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	expectKnowledgeBaseCatalogResourceLookup(tenantMock, 77, 3, 11, 12, 13, 12)
	expectDeleteKnowledgeBaseModelSegments(tenantMock, 77)
	for _, table := range []string{"knowledge_base_source_job_runs", "knowledge_base_sources", "knowledge_base_raw_volumes", "knowledge_base_source_jobs", "knowledge_base_data_domains"} {
		tenantMock.ExpectExec("DELETE FROM " + table).WithArgs(int64(77)).WillReturnResult(sqlmock.NewResult(1, 1))
	}
	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)
	err = svc.DeleteModel(ctx, 77)
	if err == nil || !strings.Contains(err.Error(), "semantic delete failed") {
		t.Fatalf("DeleteModel error = %v", err)
	}
	if !sameStringSlice(workflowSvc.validates, expectedKnowledgeBaseWorkflowIDs("ws-1", 77)) {
		t.Fatalf("workflow validates = %+v", workflowSvc.validates)
	}
	if !sameStringSlice(workflowSvc.deletes, expectedKnowledgeBaseWorkflowIDs("ws-1", 77)) {
		t.Fatalf("workflow deletes = %+v", workflowSvc.deletes)
	}
	wantDataDomainCalls := []string{"delete-volume:12", "delete-volume:13", "delete-database:11"}
	if !sameStringSet(dataDomainSvc.calls, wantDataDomainCalls) {
		t.Fatalf("data domain delete calls = %+v, want %+v", dataDomainSvc.calls, wantDataDomainCalls)
	}
	if len(fileSvc.deleted) != 0 {
		t.Fatalf("file delete calls = %+v, want none", fileSvc.deleted)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceDeleteModelCleansResourcesWhenSemanticModelAlreadyDeleted(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodDelete && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77" {
			http.Error(w, `{"code":"NOT_FOUND","error":"semantic model not found"}`, http.StatusNotFound)
			return
		}
		t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()
	dataDomainSvc := &fakeSemanticModelDataDomainService{}
	fileSvc := &fakeSemanticModelFileService{}
	workflowSvc := &fakeSemanticModelWorkflowService{}
	svc := newSemanticModelTestServiceWithKnowledgeBaseRuntimeDependencies(t, server.URL, systemClient, dataDomainSvc, fileSvc, &fakeSemanticModelLocalFileImportService{}, nil, workflowSvc)

	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	expectKnowledgeBaseCatalogResourceLookup(tenantMock, 77, 3, 11, 12, 13, 12)
	expectDeleteKnowledgeBaseModelSegments(tenantMock, 77)
	for _, table := range []string{"knowledge_base_source_job_runs", "knowledge_base_sources", "knowledge_base_raw_volumes", "knowledge_base_source_jobs", "knowledge_base_data_domains"} {
		tenantMock.ExpectExec("DELETE FROM " + table).WithArgs(int64(77)).WillReturnResult(sqlmock.NewResult(1, 1))
	}

	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)
	if err := svc.DeleteModel(ctx, 77); err != nil {
		t.Fatalf("DeleteModel: %v", err)
	}
	if !sameStringSlice(workflowSvc.deletes, expectedKnowledgeBaseWorkflowIDs("ws-1", 77)) {
		t.Fatalf("workflow deletes = %+v", workflowSvc.deletes)
	}
	wantDataDomainCalls := []string{"delete-volume:12", "delete-volume:13", "delete-database:11"}
	if !sameStringSet(dataDomainSvc.calls, wantDataDomainCalls) {
		t.Fatalf("data domain delete calls = %+v, want %+v", dataDomainSvc.calls, wantDataDomainCalls)
	}
	if len(fileSvc.deleted) != 0 {
		t.Fatalf("file delete calls = %+v, want none", fileSvc.deleted)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceListModelsByIDsUsesCallerClient(t *testing.T) {
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		if r.Method != http.MethodGet {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		requireSemanticModelExecutionHeaders(t, r)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/workspaces/ws-1/semantic-models/200":
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 200, "name": "kb-authorized"})
		case "/api/v1/workspaces/ws-1/semantic-models/300":
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]any{"code": 3, "message": "not found"})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()

	svc := newSemanticModelTestService(t, server.URL, systemClient)
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	expectEmptyKnowledgeBaseSourceCounts(tenantMock, 200)
	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)

	resp, err := svc.ListModelsByIDs(ctx, []int64{200, 300}, ListSemanticModelsRequest{PageSize: 20})
	if err != nil {
		t.Fatalf("ListModelsByIDs: %v", err)
	}
	if resp.Total != 1 || len(resp.Items) != 1 || resp.Items[0].ID != 200 {
		t.Fatalf("ListModelsByIDs response = %+v", resp)
	}
	if len(paths) != 2 {
		t.Fatalf("request paths = %+v, want 2 requests", paths)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceListModelsByIDsPaginatesSearchesAndFiltersTags(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requireSemanticModelExecutionHeaders(t, r)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/workspaces/ws-1/semantic-models/100":
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 100, "name": "alpha", "description": "finance"})
		case "/api/v1/workspaces/ws-1/semantic-models/200":
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 200, "name": "beta", "description": "sales", "files": map[string]any{"tags": []string{"finance"}}})
		case "/api/v1/workspaces/ws-1/semantic-models/300":
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 300, "name": "gamma", "description": "sales", "files": map[string]any{"tags": []string{"sales"}}})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()

	svc := newSemanticModelTestService(t, server.URL, systemClient)
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	expectEmptyKnowledgeBaseSourceCounts(tenantMock, 200)
	expectEmptyKnowledgeBaseSourceCounts(tenantMock, 200)
	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)

	params := ListSemanticModelsRequest{
		PageSize:  1,
		PageToken: encodeSemanticModelPageToken(1),
		Search:    "sales",
		Tags:      []string{"finance", "sales"},
	}
	resp, err := svc.ListModelsByIDs(ctx, []int64{100, 200, 300}, params)
	if err != nil {
		t.Fatalf("ListModelsByIDs: %v", err)
	}
	if resp.Total != 2 || len(resp.Items) != 1 || resp.Items[0].ID != 200 || resp.NextPageToken != "" {
		t.Fatalf("ListModelsByIDs response = %+v", resp)
	}
	resp, err = svc.ListModelsByIDs(ctx, []int64{300, 100, 200}, params)
	if err != nil {
		t.Fatalf("ListModelsByIDs shuffled: %v", err)
	}
	if resp.Total != 2 || len(resp.Items) != 1 || resp.Items[0].ID != 200 || resp.NextPageToken != "" {
		t.Fatalf("ListModelsByIDs shuffled response = %+v", resp)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelMatchesTags(t *testing.T) {
	model := &SemanticModelInfo{
		Files: json.RawMessage(`{"tags":["finance","policy"]}`),
	}

	tests := []struct {
		name  string
		model *SemanticModelInfo
		tags  []string
		want  bool
	}{
		{
			name:  "matches any requested tag",
			model: model,
			tags:  []string{"sales", " policy "},
			want:  true,
		},
		{
			name:  "does not match any requested tag",
			model: model,
			tags:  []string{"sales", "operations"},
			want:  false,
		},
		{
			name: "whitespace only is not a filter",
			tags: []string{" ", "\t"},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := semanticModelMatchesTags(tt.model, tt.tags); got != tt.want {
				t.Fatalf("semanticModelMatchesTags(%v) = %v, want %v", tt.tags, got, tt.want)
			}
		})
	}
}

func TestApplyKnowledgeBaseSourceJobStatusUsesLoadJobForStructuredTableSource(t *testing.T) {
	items := []SemanticModelSource{
		{
			RowID:      "source-structured-table",
			SourceID:   "source-structured-table",
			SourceType: SemanticModelSourceTypeTable,
			ModelID:    77,
			ResourceID: "10007",
			KBTableID:  int64Ptr(10007),
			DBName:     stringPtr("kb_docs"),
			TableName:  stringPtr("structured_orders"),
		},
	}
	jobs := []KnowledgeBaseSourceJobRun{
		{
			JobID:     "job-load-structured-table",
			SourceID:  "source-structured-table",
			ModelID:   77,
			JobType:   kbJobTypeLoad,
			JobStatus: kbSourceJobSucceeded,
			KBTableID: int64Ptr(10007),
		},
	}

	got := applyKnowledgeBaseSourceJobStatus(items, jobs)
	if len(got) != 1 || got[0].IngestStatus == nil || *got[0].IngestStatus != kbSourceStatusSucceeded {
		t.Fatalf("structured table source status = %+v, want succeeded", got)
	}
}

func TestApplyKnowledgeBaseSourceJobStatusKeepsStructuredTablePendingUntilAssociationComplete(t *testing.T) {
	items := []SemanticModelSource{
		{
			RowID:      "source-structured-table",
			SourceID:   "source-structured-table",
			SourceType: SemanticModelSourceTypeTable,
			ModelID:    77,
		},
	}
	jobs := []KnowledgeBaseSourceJobRun{
		{
			JobID:     "job-load-structured-table",
			SourceID:  "source-structured-table",
			ModelID:   77,
			JobType:   kbJobTypeLoad,
			JobStatus: kbSourceJobSucceeded,
			KBTableID: int64Ptr(10007),
		},
	}

	got := applyKnowledgeBaseSourceJobStatus(items, jobs)
	if len(got) != 1 || got[0].IngestStatus == nil || *got[0].IngestStatus != kbSourceStatusPending {
		t.Fatalf("structured table source status = %+v, want pending until source table association is persisted", got)
	}
}

func TestApplyKnowledgeBaseSourceJobStatusKeepsRAGFilePendingUntilSegmentVersionPublished(t *testing.T) {
	items := []SemanticModelSource{
		{
			RowID:      "source-rag-file",
			SourceID:   "source-rag-file",
			SourceType: SemanticModelSourceTypeFile,
			ModelID:    77,
			ResourceID: "kb-file",
			KBFileID:   stringPtr("kb-file"),
		},
	}
	jobs := []KnowledgeBaseSourceJobRun{
		{
			JobID:     "job-rag-file",
			SourceID:  "source-rag-file",
			ModelID:   77,
			JobType:   kbJobTypeRAGIngest,
			JobStatus: kbSourceJobSucceeded,
			KBFileID:  stringPtr("kb-file"),
		},
	}

	got := applyKnowledgeBaseSourceJobStatus(items, jobs)
	if len(got) != 1 || got[0].IngestStatus == nil || *got[0].IngestStatus != kbSourceStatusPending {
		t.Fatalf("rag file source status = %+v, want pending until segment version is published", got)
	}
}

func TestApplyKnowledgeBaseSourceJobStatusMarksRAGFileSucceededWithCurrentSegmentVersion(t *testing.T) {
	items := []SemanticModelSource{
		{
			RowID:            "source-rag-file",
			SourceID:         "source-rag-file",
			SourceType:       SemanticModelSourceTypeFile,
			ModelID:          77,
			ResourceID:       "kb-file",
			KBFileID:         stringPtr("kb-file"),
			SegmentVersionID: stringPtr("segment-v1"),
			IndexVersion:     int64Ptr(1),
		},
	}
	jobs := []KnowledgeBaseSourceJobRun{
		{
			JobID:     "job-rag-file",
			SourceID:  "source-rag-file",
			ModelID:   77,
			JobType:   kbJobTypeRAGIngest,
			JobStatus: kbSourceJobSucceeded,
			KBFileID:  stringPtr("kb-file"),
		},
	}

	got := applyKnowledgeBaseSourceJobStatus(items, jobs)
	if len(got) != 1 || got[0].IngestStatus == nil || *got[0].IngestStatus != kbSourceStatusSucceeded {
		t.Fatalf("rag file source status = %+v, want succeeded", got)
	}
}

func TestApplyKnowledgeBaseSourceJobStatusPreservesLegacyRAGFileSucceededWithIndexVersion(t *testing.T) {
	items := []SemanticModelSource{
		{
			RowID:        "source-rag-file",
			SourceID:     "source-rag-file",
			SourceType:   SemanticModelSourceTypeFile,
			ModelID:      77,
			ResourceID:   "kb-file",
			KBFileID:     stringPtr("kb-file"),
			IndexVersion: int64Ptr(1),
		},
	}
	operationID := "legacy_source_job:12"
	jobs := []KnowledgeBaseSourceJobRun{
		{
			JobID:       "job-rag-file",
			SourceID:    "source-rag-file",
			ModelID:     77,
			JobType:     kbJobTypeRAGIngest,
			JobStatus:   kbSourceJobSucceeded,
			OperationID: &operationID,
			KBFileID:    stringPtr("kb-file"),
		},
	}

	got := applyKnowledgeBaseSourceJobStatus(items, jobs)
	if len(got) != 1 || got[0].IngestStatus == nil || *got[0].IngestStatus != kbSourceStatusSucceeded {
		t.Fatalf("legacy rag file source status = %+v, want succeeded", got)
	}
}

func TestSemanticModelServiceValidateLocalizesStructuredCoreErrors(t *testing.T) {
	tests := []struct {
		name   string
		locale string
		want   string
	}{
		{
			name:   "en-US",
			locale: "en-US",
			want:   "semantic model validation failed",
		},
		{
			name:   "zh-CN",
			locale: "zh-CN",
			want:   "语义模型校验失败",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rawDiagnostic := "entry metric revenue requires unknown table orders_missing"
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost || r.URL.Path != "/api/v1/workspaces/ws-1/semantic-models/42/validate" {
					t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
				}
				requireSemanticModelExecutionHeaders(t, r)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"code":    int32(common.ErrorCode_INVALID_ARGUMENT),
					"message": rawDiagnostic,
					"details": common.NewErrorInfoDetails("SESSION_VALIDATION_FAILED", "moi-core.session", nil),
				})
			}))
			defer server.Close()

			systemClient, err := moi.New(server.URL, "system-key")
			if err != nil {
				t.Fatalf("moi.New: %v", err)
			}
			defer systemClient.Close()

			svc := newSemanticModelTestService(t, server.URL, systemClient)
			resp, err := svc.Validate(semanticModelServiceTestContext(tt.locale), 42)
			if err != nil {
				t.Fatalf("Validate: %v", err)
			}
			if resp.Valid || len(resp.Errors) != 1 || resp.Errors[0] != tt.want {
				t.Fatalf("Validate response = %+v, want valid=false error %q", resp, tt.want)
			}
			if resp.Errors[0] == rawDiagnostic || strings.Contains(resp.Errors[0], "orders_missing") {
				t.Fatalf("Validate leaked field-level producer diagnostic: %+v", resp)
			}
		})
	}
}

func TestSemanticModelServiceValidateKeepsLegacySDKErrorMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/workspaces/ws-1/semantic-models/42/validate" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code":    int32(common.ErrorCode_INVALID_ARGUMENT),
			"message": "legacy validation diagnostic",
		})
	}))
	defer server.Close()

	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()

	svc := newSemanticModelTestService(t, server.URL, systemClient)
	resp, err := svc.Validate(semanticModelServiceTestContext("zh-CN"), 42)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if resp.Valid || len(resp.Errors) != 1 || resp.Errors[0] != "legacy validation diagnostic" {
		t.Fatalf("Validate response = %+v", resp)
	}
}

func TestSemanticModelServiceCheckSourceExistenceReturnsCurrentPageMembership(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/workspaces/ws-1/semantic-models/77" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		requireSemanticModelExecutionHeaders(t, r)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 77, "name": "kb_docs"})
	}))
	defer server.Close()

	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	ctx := ctxutil.WithTenantDB(semanticModelServiceTestContext("en-US"), tenantDB)

	tenantMock.ExpectQuery("SELECT source_file_id, kb_file_id, raw_volume_id\\s+FROM knowledge_base_sources").
		WithArgs(int64(77), kbSourceTypeCatalogFile, kbSourceStatusRemoved, "file-1", "file-2", "kb-file-3", "file-1", "file-2", "kb-file-3").
		WillReturnRows(sqlmock.NewRows([]string{"source_file_id", "kb_file_id", "raw_volume_id"}).
			AddRow("file-1", nil, int64(42)).
			AddRow(nil, "kb-file-3", int64(42)))
	tenantMock.ExpectQuery("SELECT source_table_id, kb_table_id\\s+FROM knowledge_base_sources").
		WithArgs(int64(77), kbSourceTypeCatalogTable, kbSourceStatusRemoved, int64(1001), int64(1002), int64(2002), int64(1001), int64(1002), int64(2002)).
		WillReturnRows(sqlmock.NewRows([]string{"source_table_id", "kb_table_id"}).
			AddRow(int64(1002), nil).
			AddRow(nil, int64(2002)))

	configureSemanticModelTestCore(t, server.URL)
	svc := &semanticModelService{}
	result, err := svc.CheckSourceExistence(ctx, CheckSemanticModelSourceExistenceParams{
		ModelID:  77,
		FileIDs:  []string{"file-1", "file-2", "kb-file-3"},
		TableIDs: []int64{1001, 1002, 2002},
	})
	if err != nil {
		t.Fatalf("CheckSourceExistence: %v", err)
	}
	if got, want := strings.Join(result.FileIDs, ","), "file-1,kb-file-3"; got != want {
		t.Fatalf("file ids = %q, want %q", got, want)
	}
	if got, want := fmt.Sprint(result.TableIDs), "[1002 2002]"; got != want {
		t.Fatalf("table ids = %s, want %s", got, want)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestSemanticModelServiceCheckSourceExistenceRejectsInvalidIDs(t *testing.T) {
	tests := []CheckSemanticModelSourceExistenceParams{
		{ModelID: 77, FileIDs: []string{" file-1 "}},
		{ModelID: 77, FileIDs: []string{"file-1", "file-1"}},
		{ModelID: 77, TableIDs: []int64{0}},
		{ModelID: 77, TableIDs: []int64{1001, 1001}},
	}
	svc := &semanticModelService{}
	for _, params := range tests {
		if _, err := svc.CheckSourceExistence(semanticModelServiceTestContext("en-US"), params); err == nil {
			t.Fatalf("CheckSourceExistence(%+v) error = nil, want invalid input", params)
		}
	}
}

func TestSemanticModelServiceCheckSourceExistenceRequiresTenantDB(t *testing.T) {
	svc := &semanticModelService{}
	_, err := svc.CheckSourceExistence(semanticModelServiceTestContext("en-US"), CheckSemanticModelSourceExistenceParams{
		ModelID: 77,
		FileIDs: []string{"file-1"},
	})
	if err == nil || !strings.Contains(err.Error(), "tenant db is required") {
		t.Fatalf("CheckSourceExistence error = %v, want tenant db required", err)
	}
}

func TestSemanticModelServiceCheckSourceExistenceRejectsMissingModel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/workspaces/ws-1/semantic-models/77" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{"code": common.ErrorCode_NOT_FOUND, "message": "semantic model not found"})
	}))
	defer server.Close()
	tenantDB, tenantMock := newSemanticModelTenantDB(t)
	ctx := ctxutil.WithTenantDB(semanticModelServiceTestContext("en-US"), tenantDB)

	configureSemanticModelTestCore(t, server.URL)
	svc := &semanticModelService{}
	_, err := svc.CheckSourceExistence(ctx, CheckSemanticModelSourceExistenceParams{
		ModelID: 77,
		FileIDs: []string{"file-1"},
	})
	if !moi.IsCode(err, common.ErrorCode_NOT_FOUND) {
		t.Fatalf("CheckSourceExistence error = %v, want NOT_FOUND", err)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelService_UploadLocalFileRejectsInvalidInput(t *testing.T) {
	svc := &semanticModelService{}
	if _, err := svc.UploadLocalFile(context.Background(), " ", strings.NewReader("x")); err == nil {
		t.Fatal("expected empty file name error")
	}
	if _, err := svc.UploadLocalFile(context.Background(), "doc.txt", nil); err == nil {
		t.Fatal("expected nil reader error")
	}
}

func TestSemanticModelService_UploadLocalFile_UsesSystemClient(t *testing.T) {
	var requests []string
	var gotPath string
	var gotAuth string
	var gotName string
	var gotBody string
	var gotReq string
	var gotTrace string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/workspaces/ws-1/files" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
			return
		}
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("X-API-Key")
		gotReq = r.Header.Get("X-Request-ID")
		gotTrace = r.Header.Get("X-Trace-ID")
		file, header, err := r.FormFile("file")
		if err != nil {
			t.Errorf("FormFile: %v", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		defer file.Close()
		body, err := io.ReadAll(file)
		if err != nil {
			t.Errorf("ReadAll: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		gotName = header.Filename
		gotBody = string(body)
		_ = json.NewEncoder(w).Encode(map[string]any{"file_id": "system-uploaded-1", "original_name": header.Filename})
	}))
	defer server.Close()

	configureSemanticModelTestCore(t, server.URL)
	fileSvc := &fakeSemanticModelFileService{}
	svc := &semanticModelService{fileService: fileSvc}
	if _, err := svc.UploadLocalFile(context.Background(), "doc.txt", strings.NewReader("payload")); err == nil || !strings.Contains(err.Error(), "workspace_id not found") {
		t.Fatalf("missing workspace error = %v", err)
	}
	ctx := ctxutil.WithWorkspaceID(context.Background(), "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithCoreIAMRequest(ctx, ctxutil.CoreIAMRequestContext{
		RequestID: "req-upload-local",
		TraceID:   "trace-upload-local",
	})
	fileID, err := svc.UploadLocalFile(ctx, " doc.txt ", strings.NewReader("payload"))
	if err != nil {
		t.Fatalf("UploadLocalFile() error = %v", err)
	}
	if fileID != "system-uploaded-1" {
		t.Fatalf("UploadLocalFile() fileID = %q, want system-uploaded-1", fileID)
	}
	if gotPath != "/api/v1/workspaces/ws-1/files" {
		t.Fatalf("upload path = %q", gotPath)
	}
	if !strings.Contains(gotAuth, "system-key") {
		t.Fatalf("upload auth = %q, want system key", gotAuth)
	}
	if gotReq != "req-upload-local" || gotTrace != "trace-upload-local" {
		t.Fatalf("upload trace headers = (%q, %q)", gotReq, gotTrace)
	}
	if gotName != " doc.txt " || gotBody != "payload" {
		t.Fatalf("uploaded file = (%q, %q)", gotName, gotBody)
	}
	if len(requests) != 1 || requests[0] != "POST /api/v1/workspaces/ws-1/files" {
		t.Fatalf("requests = %v, want only file upload", requests)
	}
}

func TestSemanticModelService_UploadLocalFile_RejectsInvalidCoreResponse(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       map[string]any
		wantError  string
	}{
		{name: "upload failure", statusCode: http.StatusInternalServerError, body: map[string]any{"message": "upload failed"}, wantError: "upload local file"},
		{name: "empty file id", statusCode: http.StatusCreated, body: map[string]any{"file_id": ""}, wantError: "empty file_id"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.statusCode)
				_ = json.NewEncoder(w).Encode(tt.body)
			}))
			defer server.Close()

			configureSemanticModelTestCore(t, server.URL)
			svc := &semanticModelService{}
			ctx := ctxutil.WithWorkspaceID(context.Background(), "ws-1")
			ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
			if _, err := svc.UploadLocalFile(ctx, "doc.txt", strings.NewReader("payload")); err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("UploadLocalFile() error = %v, want %q", err, tt.wantError)
			}
		})
	}
}
