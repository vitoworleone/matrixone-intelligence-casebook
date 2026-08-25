package session

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	mysqlDriver "github.com/go-sql-driver/mysql"
	moi "github.com/matrixflow/moi-core/go-sdk"
	"github.com/matrixflow/moi-core/model/common"
	coresaga "github.com/matrixflow/moi-core/saga"
	sagastore "github.com/matrixflow/moi-core/saga/storage"
	backendcatalog "github.com/matrixorigin/matrixflow/moi-backend/pkg/catalog"
	"github.com/matrixorigin/matrixflow/moi-backend/pkg/coreclient"
	"github.com/matrixorigin/matrixflow/moi-backend/pkg/ctxutil"
	"github.com/matrixorigin/matrixflow/moi-backend/pkg/i18n"
	"github.com/matrixorigin/matrixflow/moi-backend/pkg/iampep"
	"github.com/matrixorigin/matrixflow/moi-backend/pkg/workflowv2"
	gmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// Create / data-domain provision tests for knowledge bases.
// Split from semantic_model_service_test.go to reduce monolith cognitive load.

func TestResolveKnowledgeBaseCreateEmbeddingConfigUsesAvailableSelfHostedBackend(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isKnowledgeBaseEmbeddingModelsRequest(r) {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		writeKnowledgeBaseEmbeddingModelsResponse(t, w, 42)
	}))
	defer server.Close()

	client, err := moi.New(server.URL, "caller-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer client.Close()

	backendID, err := resolveKnowledgeBaseCreateEmbeddingConfig(context.Background(), client, "ws-1", true)
	if err != nil {
		t.Fatalf("resolveKnowledgeBaseCreateEmbeddingConfig: %v", err)
	}
	if backendID != "42" {
		t.Fatalf("image embedding backend id = %q, want self-hosted backend 42", backendID)
	}
}

func TestResolveKnowledgeBaseCreateEmbeddingConfigRejectsMissingFixedImageModelWhenEnabled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isKnowledgeBaseEmbeddingModelsRequest(r) {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"models": []map[string]any{{"model": "bge-m3", "backend_id": int64(41), "backend_name": "self-hosted-text"}},
		})
	}))
	defer server.Close()

	client, err := moi.New(server.URL, "caller-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer client.Close()

	_, err = resolveKnowledgeBaseCreateEmbeddingConfig(context.Background(), client, "ws-1", true)
	if !IsServiceError(err, ErrCodeBadRequest) || !strings.Contains(err.Error(), i18n.KeySessionKnowledgeBaseEmbeddingModelNotAvailable.String()) {
		t.Fatalf("resolve error = %v, want explicit bad request for missing efficientnet-b3", err)
	}
}

func TestResolveKnowledgeBaseCreateEmbeddingConfigSkipsImageModelWhenDisabled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isKnowledgeBaseEmbeddingModelsRequest(r) {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"models": []map[string]any{{"model": "bge-m3", "backend_id": int64(41), "backend_name": "self-hosted-text"}},
		})
	}))
	defer server.Close()

	client, err := moi.New(server.URL, "caller-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer client.Close()

	backendID, err := resolveKnowledgeBaseCreateEmbeddingConfig(context.Background(), client, "ws-1", false)
	if err != nil {
		t.Fatalf("resolveKnowledgeBaseCreateEmbeddingConfig: %v", err)
	}
	if backendID != "" {
		t.Fatalf("image embedding backend id = %q, want empty when disabled", backendID)
	}
}

func TestCreateModelWithDocumentSourcesUsesCallerWorkflowBoundary(t *testing.T) {
	modelID := int64(77)
	sourceFileID := "source-file"
	effectiveRoleID := "role-current-caller"
	semanticGetCount := 0
	workflowDeployCount := 0
	workflowRunCount := 0
	var workflowDeployReq moi.WorkflowDeploymentRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case isKnowledgeBaseEmbeddingModelsRequest(r):
			if got := r.Header.Get("X-API-Key"); got != "system-key" {
				t.Fatalf("embedding capability authorization = %q, want system key", got)
			}
			writeKnowledgeBaseEmbeddingModelsResponse(t, w, 42)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/ws-1/files/source-file/download":
			t.Fatalf("unexpected DownloadWithMeta during create-with-sources short request")
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models":
			requireSemanticModelExecutionHeaders(t, r)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":          modelID,
				"name":        "kb_docs",
				"description": "docs",
				"tables":      []any{},
				"files":       map[string]any{"file_ids": []string{}},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			semanticGetCount++
			t.Fatalf("unexpected semantic model GET during create-with-sources vector reuse")
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			requireSemanticModelExecutionHeaders(t, r)
			var req struct {
				Tables []semanticModelTableSource `json:"tables"`
				Files  semanticModelFilesPayload  `json:"files"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode semantic update: %v", err)
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
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/workspaces/ws-1/workflow-deployments":
			requireSemanticModelExecutionHeaders(t, r)
			if got := r.Header.Get(iampep.HeaderMOIRoleID); got != effectiveRoleID {
				t.Fatalf("workflow deploy role = %q, want %q", got, effectiveRoleID)
			}
			workflowDeployCount++
			if err := json.NewDecoder(r.Body).Decode(&workflowDeployReq); err != nil {
				t.Fatalf("decode workflow deployment: %v", err)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"deployment": map[string]any{
					"workflow_app_id":     workflowDeployReq.WorkflowID,
					"workflow_def_id":     "def-kb",
					"workflow_version_id": "ver-kb",
					"workflow_name":       workflowDeployReq.Name,
					"version":             1,
					"execution_mode":      workflowv2.ExecutionModeOneShot,
				},
			})
		case r.Method == http.MethodGet && workflowDeployReq.WorkflowID != "" && r.URL.Path == "/api/v1/workspaces/ws-1/workflow-apps/"+workflowDeployReq.WorkflowID:
			requireSemanticModelExecutionHeaders(t, r)
			if got := r.Header.Get(iampep.HeaderMOIRoleID); got != effectiveRoleID {
				t.Fatalf("workflow get role = %q, want %q", got, effectiveRoleID)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code": 0,
				"data": map[string]any{"workflow": map[string]any{
					"id":                         workflowDeployReq.WorkflowID,
					"name":                       workflowDeployReq.Name,
					"source_type":                workflowv2.SourceTypeManualDSL,
					"status":                     "ready",
					"execution_mode":             workflowv2.ExecutionModeOneShot,
					"dsl_yaml":                   workflowDeployReq.DSLYAML,
					"runtime_fields_json":        workflowDeployReq.InputFormJSON,
					"default_values_json":        workflowDeployReq.DefaultValuesJSON,
					"moi_workflow_def_id":        "def-kb",
					"latest_workflow_version_id": "ver-kb",
				}},
			})
		case r.Method == http.MethodPost && workflowDeployReq.WorkflowID != "" && r.URL.Path == "/api/v1/workspaces/ws-1/workflow-apps/"+workflowDeployReq.WorkflowID+"/executions":
			requireSemanticModelExecutionHeaders(t, r)
			if got := r.Header.Get(iampep.HeaderMOIRoleID); got != effectiveRoleID {
				t.Fatalf("workflow execution role = %q, want %q", got, effectiveRoleID)
			}
			var req moi.WorkflowExecutionCreateRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode workflow execution: %v", err)
			}
			workflowRunCount++
			_ = json.NewEncoder(w).Encode(map[string]any{"execution": map[string]any{
				"execution_id":   "exec-kb",
				"workflow_id":    workflowDeployReq.WorkflowID,
				"status":         "triggered",
				"execution_mode": workflowv2.ExecutionModeOneShot,
			}})
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/volumes/"):
			t.Fatalf("unexpected raw-volume side effect after vector reuse: %s %s", r.Method, r.URL.String())
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	dataDomainSvc := &fakeSemanticModelDataDomainService{databaseID: 11, volumeIDs: []int64{12, 13}}
	workflowTemplateSvc := &fakeSemanticModelWorkflowTemplateService{}
	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()
	configureSemanticModelTestCore(t, server.URL)
	workflowSvc := NewSemanticModelWorkflowAdapter(workflowv2.NewCoreService())
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
	tenantMock.ExpectExec("INSERT INTO knowledge_base_data_domains").
		WillReturnResult(sqlmock.NewResult(1, 1))
	expectKnowledgeBaseCreateDomainPrepare(tenantMock, modelID, 3, 11, 12, 13)
	tenantMock.ExpectBegin()
	expectCatalogFileSourceOriginLookupMiss(tenantMock, sourceFileID)
	tenantMock.ExpectExec("INSERT INTO knowledge_base_sources").
		WillReturnResult(sqlmock.NewResult(1, 1))
	tenantMock.ExpectExec("INSERT INTO knowledge_base_source_job_runs").
		WillReturnResult(sqlmock.NewResult(1, 1))
	tenantMock.ExpectExec("INSERT INTO knowledge_base_source_job_runs").
		WillReturnResult(sqlmock.NewResult(2, 1))
	tenantMock.ExpectCommit()
	expectKnowledgeBaseCreateDomainFinalize(tenantMock, modelID, 3, 11, 12, 13)

	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithAuthMode(ctx, "api_key")
	ctx = ctxutil.WithUserID(ctx, "user-1")
	ctx = ctxutil.WithMoiUserID(ctx, "moi-user-1")
	ctx = ctxutil.WithCoreIAMRequest(ctx, ctxutil.CoreIAMRequestContext{VerifiedEffectiveRoleID: effectiveRoleID})
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)

	resp, err := svc.CreateModelWithSources(ctx, CreateSemanticModelWithSourcesRequest{
		Name:        "kb_docs",
		Description: "docs",
		Sources: []CreateSemanticModelSourceRequest{{
			SourceType: kbSourceTypeCatalogFile,
			FileID:     sourceFileID,
			VolumeID:   41,
		}},
	})
	if err != nil {
		t.Fatalf("CreateModelWithSources: %v", err)
	}
	if resp.Model.ID != modelID || resp.Model.Files == nil {
		t.Fatalf("model = %+v", resp.Model)
	}
	if len(resp.Jobs) != 2 || resp.Jobs[0].JobStatus != kbSourceJobPending || resp.Jobs[1].JobStatus != kbSourceJobPending || resp.Jobs[1].OperationID != nil {
		t.Fatalf("jobs = %+v, want deferred catalog file jobs", resp.Jobs)
	}
	if semanticGetCount != 0 {
		t.Fatalf("semantic GET count = %d, want 0 during reuse", semanticGetCount)
	}
	if workflowDeployCount != 1 || workflowDeployReq.ExecutionMode != workflowv2.ExecutionModeOneShot || len(workflowTemplateSvc.calls) != 1 || workflowTemplateSvc.calls[0] != kbStandardRAGTemplateKey {
		t.Fatalf("workflow deploy count=%d request=%+v template calls=%+v", workflowDeployCount, workflowDeployReq, workflowTemplateSvc.calls)
	}
	workflowRun, err := workflowSvc.RunKnowledgeBaseWorkflow(ctx, workflowDeployReq.WorkflowID, map[string]any{
		"source_ref": map[string]any{
			"kind":          "file",
			"resource_type": "file",
			"file_id":       sourceFileID,
			"file_ids":      []string{sourceFileID},
			"ids":           []string{sourceFileID},
		},
	})
	if err != nil {
		t.Fatalf("RunKnowledgeBaseWorkflow: %v", err)
	}
	if workflowRun == nil || workflowRun.ExecutionID != "exec-kb" || workflowRunCount != 1 {
		t.Fatalf("workflow run=%+v count=%d", workflowRun, workflowRunCount)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceCreateModelMapsDuplicateNameConflict(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/workspaces/ws-1/semantic-models" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(&common.ErrorResponse{
			Code:      common.ErrorCode_ALREADY_EXISTS,
			Message:   "semantic model already exists",
			RequestId: "req-duplicate",
		})
	}))
	defer server.Close()

	svc := newSemanticModelTestServiceWithDependencies(t, server.URL, nil, &fakeSemanticModelDataDomainService{}, nil)
	ctx := semanticModelServiceTestContext(i18n.LocaleZhCN.String())

	_, err := svc.CreateModel(ctx, CreateSemanticModelRequest{Name: "kb_docs"})
	if !IsServiceError(err, ErrCodeConflict) {
		t.Fatalf("CreateModel error = %v, want conflict service error", err)
	}
	msg, ok := i18n.Message(ctx, err)
	if !ok {
		t.Fatalf("CreateModel error is not localized: %T", err)
	}
	if msg != "语义模型名称已存在" {
		t.Fatalf("localized message = %q", msg)
	}
}

func TestSemanticModelServiceCreateModelCreatesReadyKnowledgeBaseResources(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/workspaces/ws-1/semantic-models" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": int64(77), "name": "kb_docs", "description": "docs",
			"tables": []any{}, "files": map[string]any{"file_ids": []string{}},
		})
	}))
	defer server.Close()

	dataDomainSvc := &fakeSemanticModelDataDomainService{databaseID: 11, volumeIDs: []int64{12, 13}}
	svc := newSemanticModelTestServiceWithDependencies(t, server.URL, nil, dataDomainSvc, nil)
	db, mock := newSemanticModelTenantDB(t)
	mock.ExpectExec("INSERT INTO knowledge_base_data_domains").WillReturnResult(sqlmock.NewResult(1, 1))
	expectKnowledgeBaseCreateDomainPrepare(mock, 77, 3, 11, 12, 13)
	expectKnowledgeBaseCreateDomainFinalize(mock, 77, 3, 11, 12, 13)
	ctx := ctxutil.WithTenantDB(semanticModelServiceTestContext(i18n.LocaleZhCN.String()), db)

	model, err := svc.CreateModel(ctx, CreateSemanticModelRequest{Name: "kb_docs", Description: "docs"})
	if err != nil {
		t.Fatalf("CreateModel: %v", err)
	}
	if model.ID != 77 || model.Name != "kb_docs" {
		t.Fatalf("created model = %+v", model)
	}
	wantCalls := []string{
		"resolve_default_catalog", "resolve-database:kb_docs", "database:kb_docs",
		"raw_document:Knowledge base document raw source files", "processed:Knowledge base processed files",
	}
	if !sameStringSlice(dataDomainSvc.calls, wantCalls) {
		t.Fatalf("data domain calls = %+v, want %+v", dataDomainSvc.calls, wantCalls)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceCreateEmptyModelInitializesDataDomainWithoutSources(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case isKnowledgeBaseEmbeddingModelsRequest(r):
			writeKnowledgeBaseEmbeddingModelsResponse(t, w, 42)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id": int64(77), "name": "kb_docs", "description": "docs",
				"tables": []any{}, "files": map[string]any{"file_ids": []string{}},
			})
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			var request struct {
				Tables []json.RawMessage `json:"tables"`
				Files  struct {
					FileIDs                 []string `json:"file_ids"`
					VectorTable             string   `json:"vector_table"`
					EmbeddingModel          string   `json:"embedding_model"`
					ImageEmbeddingBackendID string   `json:"image_embedding_backend_id"`
				} `json:"files"`
			}
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatalf("decode semantic update: %v", err)
			}
			if len(request.Tables) != 0 || len(request.Files.FileIDs) != 0 {
				t.Fatalf("empty knowledge base wrote sources: tables=%+v files=%+v", request.Tables, request.Files.FileIDs)
			}
			if request.Files.VectorTable != defaultKnowledgeBaseVectorTable(77) || request.Files.EmbeddingModel != kbDefaultEmbeddingModel || request.Files.ImageEmbeddingBackendID != "42" {
				t.Fatalf("fixed index config = %+v", request.Files)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"updated": true})
		default:
			t.Fatalf("unexpected core request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	dataDomainSvc := &fakeSemanticModelDataDomainService{databaseID: 11, volumeIDs: []int64{12, 13}}
	svc := newSemanticModelTestServiceWithDependencies(t, server.URL, nil, dataDomainSvc, nil)
	db, mock := newSemanticModelTenantDB(t)
	mock.ExpectExec("INSERT INTO knowledge_base_data_domains").WillReturnResult(sqlmock.NewResult(1, 1))
	expectKnowledgeBaseCreateDomainPrepare(mock, 77, 3, 11, 12, 13)
	expectKnowledgeBaseCreateDomainFinalize(mock, 77, 3, 11, 12, 13)
	ctx := ctxutil.WithTenantDB(semanticModelServiceTestContext(i18n.LocaleZhCN.String()), db)

	response, err := svc.CreateEmptyModel(ctx, CreateEmptySemanticModelRequest{
		Name: "kb_docs", Description: "docs", ImageIndexEnabled: true,
	})
	if err != nil {
		t.Fatalf("CreateEmptyModel: %v", err)
	}
	if response.Model == nil || response.Model.ID != 77 || response.DataDomain == nil || response.DataDomain.DatabaseID != 11 {
		t.Fatalf("response = %+v", response)
	}
	wantCalls := []string{
		"resolve_default_catalog", "resolve-database:kb_docs", "database:kb_docs",
		"raw_document:Knowledge base document raw source files", "processed:Knowledge base processed files",
	}
	if !sameStringSlice(dataDomainSvc.calls, wantCalls) {
		t.Fatalf("data domain calls = %+v, want %+v", dataDomainSvc.calls, wantCalls)
	}
	if !reflect.DeepEqual(dataDomainSvc.provisioningCatalogIDs, []int64{3, 3, 3}) {
		t.Fatalf("provisioning catalog IDs = %+v, want [3 3 3]", dataDomainSvc.provisioningCatalogIDs)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceCreateModelRejectsConflictingNameWithoutRepair(t *testing.T) {
	// Existing catalog DB with the same physical name must return conflict.
	// Create never adopts or repairs another request's half-created shell.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("unexpected core request on name conflict: %s %s", r.Method, r.URL.String())
	}))
	defer server.Close()

	dataDomainSvc := &fakeSemanticModelDataDomainService{
		databaseByName: map[string]int64{"kb_docs": 11},
	}
	svc := newSemanticModelTestServiceWithDependencies(t, server.URL, nil, dataDomainSvc, nil)
	ctx := semanticModelServiceTestContext(i18n.LocaleZhCN.String())

	_, err := svc.CreateModel(ctx, CreateSemanticModelRequest{Name: "kb_docs", Description: "docs"})
	if !IsServiceError(err, ErrCodeConflict) {
		t.Fatalf("CreateModel error = %v, want conflict", err)
	}
	if !sameStringSlice(dataDomainSvc.calls, []string{"resolve_default_catalog", "resolve-database:kb_docs"}) {
		t.Fatalf("data domain calls = %+v, want name availability only", dataDomainSvc.calls)
	}
}

func TestSemanticModelServiceCreateModelRollsBackPartialResourcesOnProvisionFailure(t *testing.T) {
	const modelID = int64(77)
	var deleteSMCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id": modelID, "name": "kb_docs", "description": "docs",
				"tables": []any{}, "files": map[string]any{"file_ids": []string{}},
			})
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			deleteSMCalls++
			_ = json.NewEncoder(w).Encode(map[string]any{"deleted": true})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	dataDomainSvc := &fakeSemanticModelDataDomainService{
		databaseID: 11,
		volumeIDs:  []int64{12},
		volumeErrs: map[string]error{"processed": errors.New("processed volume unavailable")},
	}
	workflowSvc := &fakeSemanticModelWorkflowService{}
	svc := newSemanticModelTestServiceWithKnowledgeBaseRuntimeDependencies(t, server.URL, nil, dataDomainSvc, nil, nil, nil, workflowSvc)
	db, mock := newSemanticModelTenantDB(t)
	mock.ExpectExec("INSERT INTO knowledge_base_data_domains").WillReturnResult(sqlmock.NewResult(1, 1))
	// Create retains provisioning ownership; rollback persists partial IDs then deletes.
	expectRollbackFailedKnowledgeBaseCreate(mock, modelID, 3, 11, 12, 0)
	ctx := ctxutil.WithTenantDB(semanticModelServiceTestContext(i18n.LocaleZhCN.String()), db)

	_, err := svc.CreateModel(ctx, CreateSemanticModelRequest{Name: "kb_docs", Description: "docs"})
	if err == nil {
		t.Fatal("CreateModel error = nil, want provision failure")
	}
	if IsServiceError(err, ErrCodeConflict) {
		t.Fatalf("CreateModel error = %v, want non-conflict provision failure", err)
	}
	if deleteSMCalls != 1 {
		t.Fatalf("semantic model delete calls = %d, want 1", deleteSMCalls)
	}
	wantCalls := []string{
		"resolve_default_catalog", "resolve-database:kb_docs", "database:kb_docs",
		"raw_document:Knowledge base document raw source files",
		"processed:Knowledge base processed files",
		"delete-volume:12", "delete-database:11",
	}
	if !sameStringSlice(dataDomainSvc.calls, wantCalls) {
		t.Fatalf("data domain calls = %+v, want %+v", dataDomainSvc.calls, wantCalls)
	}
	// deleteModel always attempts the three KB workflow IDs (NOT_FOUND is fine).
	if len(workflowSvc.deletes) != 3 {
		t.Fatalf("workflow deletes = %+v, want 3 lifecycle deletes", workflowSvc.deletes)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestRollbackFailedKnowledgeBaseCreateUsesDetachedContextAfterCancel(t *testing.T) {
	// Parent cancel must not abort: re-persist in-memory IDs under detached ctx, then deleteModel.
	const modelID = int64(77)
	var deleteSMCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodDelete && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77" {
			deleteSMCalls++
			_ = json.NewEncoder(w).Encode(map[string]any{"deleted": true})
			return
		}
		t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
	}))
	defer server.Close()

	client, err := moi.New(server.URL, "caller-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer client.Close()

	dataDomainSvc := &fakeSemanticModelDataDomainService{databaseID: 11}
	workflowSvc := &fakeSemanticModelWorkflowService{}
	svc := &semanticModelService{
		dataDomainService: dataDomainSvc,
		workflowService:   workflowSvc,
	}
	installSemanticModelCleanupSagaTestStore(t, svc)
	db, mock := newSemanticModelTenantDB(t)
	// In-memory partial IDs are persisted while retaining provisioning ownership.
	expectRollbackFailedKnowledgeBaseCreate(mock, modelID, 3, 11, 12, 13)

	parent, cancel := context.WithCancel(semanticModelServiceTestContext(i18n.LocaleZhCN.String()))
	cancel()
	ctx := ctxutil.WithTenantDB(parent, db)

	domain := &KnowledgeBaseDataDomain{
		ModelID:           modelID,
		CatalogID:         3,
		DatabaseID:        11,
		RawVolumeID:       12,
		ProcessedVolumeID: 13,
		EnsureStatus:      kbEnsureStatusProvisioning,
	}
	cause := errors.New("processed volume unavailable")
	if err := svc.rollbackFailedKnowledgeBaseCreate(ctx, client, "ws-1", modelID, domain, cause, "user-1"); err != nil {
		t.Fatalf("rollbackFailedKnowledgeBaseCreate: %v", err)
	}
	if deleteSMCalls != 1 {
		t.Fatalf("semantic model delete calls = %d, want 1 after canceled-parent rollback", deleteSMCalls)
	}
	if !sameStringSet(dataDomainSvc.calls, []string{"delete-volume:12", "delete-volume:13", "delete-database:11"}) {
		t.Fatalf("data domain calls = %+v, want catalog cleanup despite cancel", dataDomainSvc.calls)
	}
	if len(workflowSvc.deletes) != 3 {
		t.Fatalf("workflow deletes = %+v, want lifecycle deletes despite cancel", workflowSvc.deletes)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestRecoverKnowledgeBaseCreateCleanupSagasSkipsOtherWorkspace(t *testing.T) {
	svc := &semanticModelService{}
	store := sagastore.NewMemoryStorage()
	svc.knowledgeBaseCleanupSagaExecutor = coresaga.NewExecutor(coresaga.ExecutorConfig{Storage: store, MaxRetries: -1})
	svc.knowledgeBaseCleanupSagaStore = store

	payload, err := newKnowledgeBaseCleanupSagaPayload("other-workspace", 77, nil, errors.New("create failed"), "user-1")
	if err != nil {
		t.Fatalf("new cleanup payload: %v", err)
	}
	sagaCtx, err := payload.context()
	if err != nil {
		t.Fatalf("cleanup payload context: %v", err)
	}
	state := coresaga.NewSagaState(payload.sagaID(), knowledgeBaseCleanupSagaName)
	state.Status = coresaga.SagaStatusFailed
	state.Context = sagaCtx
	if err := store.Save(context.Background(), state); err != nil {
		t.Fatalf("save cleanup Saga: %v", err)
	}

	ctx := ctxutil.WithWorkspaceID(context.Background(), "current-workspace")
	if err := svc.RecoverKnowledgeBaseCreateCleanupSagas(ctx, coreclient.ExecutionUser{}); err != nil {
		t.Fatalf("recover cleanup Sagas: %v", err)
	}
	got, err := store.Load(context.Background(), payload.sagaID())
	if err != nil {
		t.Fatalf("load cleanup Saga: %v", err)
	}
	if got.Status != coresaga.SagaStatusFailed {
		t.Fatalf("other workspace Saga status = %s, want failed", got.Status)
	}
}

func TestRollbackFailedKnowledgeBaseCreateDeletesCoreWhenPartialIDPersistFails(t *testing.T) {
	// Backend ownership must remain when its partial receipt cannot be persisted,
	// but that failure must not skip the independent Core compensation.
	const modelID = int64(77)
	var deleteSMCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodDelete && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77" {
			deleteSMCalls++
			_ = json.NewEncoder(w).Encode(map[string]any{"deleted": true})
			return
		}
		t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
	}))
	defer server.Close()

	client, err := moi.New(server.URL, "caller-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer client.Close()

	dataDomainSvc := &fakeSemanticModelDataDomainService{databaseID: 11}
	workflowSvc := &fakeSemanticModelWorkflowService{}
	svc := &semanticModelService{
		dataDomainService: dataDomainSvc,
		workflowService:   workflowSvc,
	}
	installSemanticModelCleanupSagaTestStore(t, svc)
	db, mock := newSemanticModelTenantDB(t)
	mock.ExpectExec("UPDATE knowledge_base_data_domains\\s+SET catalog_id = \\?, database_id = \\?, raw_volume_id = \\?, processed_volume_id = \\?, ensure_status = \\?, last_ensure_error = \\?, last_checked_at = \\?, updated_by = \\?\\s+WHERE model_id = \\? AND ensure_status = \\?").
		WithArgs(int64(3), int64(11), int64(12), int64(13), kbEnsureStatusProvisioning, sqlmock.AnyArg(), sqlmock.AnyArg(), "user-1", modelID, kbEnsureStatusProvisioning).
		WillReturnError(errors.New("tenant db unavailable"))

	ctx := ctxutil.WithTenantDB(semanticModelServiceTestContext(i18n.LocaleZhCN.String()), db)
	domain := &KnowledgeBaseDataDomain{
		ModelID:           modelID,
		CatalogID:         3,
		DatabaseID:        11,
		RawVolumeID:       12,
		ProcessedVolumeID: 13,
		EnsureStatus:      kbEnsureStatusProvisioning,
	}
	err = svc.rollbackFailedKnowledgeBaseCreate(ctx, client, "ws-1", modelID, domain, errors.New("provision failed"), "user-1")
	if err == nil || !strings.Contains(err.Error(), "persist partial resources before rollback") {
		t.Fatalf("rollback error = %v, want persist partial resources before rollback", err)
	}
	if deleteSMCalls != 1 {
		t.Fatalf("semantic model delete calls = %d, want 1 when partial ID persist fails", deleteSMCalls)
	}
	if len(workflowSvc.deletes) != 0 {
		t.Fatalf("workflow deletes = %+v, want none when partial ID persist fails", workflowSvc.deletes)
	}
	if len(dataDomainSvc.calls) != 0 {
		t.Fatalf("data domain calls = %+v, want none when partial ID persist fails", dataDomainSvc.calls)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestRollbackFailedKnowledgeBaseCreateDeletesCoreWhenProvisioningCASMisses(t *testing.T) {
	const modelID = int64(77)
	var deleteSMCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77" {
			deleteSMCalls++
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"deleted": true})
			return
		}
		t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
	}))
	defer server.Close()

	client, err := moi.New(server.URL, "caller-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer client.Close()

	dataDomainSvc := &fakeSemanticModelDataDomainService{databaseID: 11}
	workflowSvc := &fakeSemanticModelWorkflowService{}
	svc := &semanticModelService{
		dataDomainService: dataDomainSvc,
		workflowService:   workflowSvc,
	}
	installSemanticModelCleanupSagaTestStore(t, svc)
	db, mock := newSemanticModelTenantDB(t)
	mock.ExpectExec("UPDATE knowledge_base_data_domains\\s+SET catalog_id = \\?, database_id = \\?, raw_volume_id = \\?, processed_volume_id = \\?, ensure_status = \\?, last_ensure_error = \\?, last_checked_at = \\?, updated_by = \\?\\s+WHERE model_id = \\? AND ensure_status = \\?").
		WithArgs(int64(3), int64(11), int64(12), int64(13), kbEnsureStatusProvisioning, sqlmock.AnyArg(), sqlmock.AnyArg(), "user-1", modelID, kbEnsureStatusProvisioning).
		WillReturnResult(sqlmock.NewResult(0, 0))

	ctx := ctxutil.WithTenantDB(semanticModelServiceTestContext(i18n.LocaleZhCN.String()), db)
	domain := &KnowledgeBaseDataDomain{
		ModelID:           modelID,
		CatalogID:         3,
		DatabaseID:        11,
		RawVolumeID:       12,
		ProcessedVolumeID: 13,
		EnsureStatus:      kbEnsureStatusProvisioning,
	}
	err = svc.rollbackFailedKnowledgeBaseCreate(ctx, client, "ws-1", modelID, domain, errors.New("provision failed"), "user-1")
	if err == nil || !strings.Contains(err.Error(), "persist partial resources before rollback") {
		t.Fatalf("rollback error = %v, want ownership-loss error", err)
	}
	if deleteSMCalls != 1 || len(workflowSvc.deletes) != 0 || len(dataDomainSvc.calls) != 0 {
		t.Fatalf("cleanup after ownership loss: semantic=%d workflows=%+v catalog=%+v", deleteSMCalls, workflowSvc.deletes, dataDomainSvc.calls)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceCreateModelDeletesCoreWhenCatalogCleanupFails(t *testing.T) {
	// A Backend cleanup failure retains Backend ownership for retry but must not
	// block the independent Core delete.
	const modelID = int64(77)
	var deleteSMCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id": modelID, "name": "kb_docs", "description": "docs",
				"tables": []any{}, "files": map[string]any{"file_ids": []string{}},
			})
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			deleteSMCalls++
			_ = json.NewEncoder(w).Encode(map[string]any{"deleted": true})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	dataDomainSvc := &fakeSemanticModelDataDomainService{
		databaseID: 11,
		volumeIDs:  []int64{12},
		volumeErrs: map[string]error{"processed": errors.New("processed volume unavailable")},
		deleteErr:  errors.New("catalog cleanup failed"),
	}
	workflowSvc := &fakeSemanticModelWorkflowService{}
	svc := newSemanticModelTestServiceWithKnowledgeBaseRuntimeDependencies(t, server.URL, nil, dataDomainSvc, nil, nil, nil, workflowSvc)
	db, mock := newSemanticModelTenantDB(t)
	mock.ExpectExec("INSERT INTO knowledge_base_data_domains").WillReturnResult(sqlmock.NewResult(1, 1))
	// Create retains provisioning ownership; catalog delete failure keeps the
	// Backend receipt for retry while the Core delete still executes.
	expectRollbackFailedKnowledgeBaseCreateCatalogOnly(mock, modelID, 3, 11, 12, 0)
	ctx := ctxutil.WithTenantDB(semanticModelServiceTestContext(i18n.LocaleZhCN.String()), db)

	_, err := svc.CreateModel(ctx, CreateSemanticModelRequest{Name: "kb_docs", Description: "docs"})
	if err == nil || !strings.Contains(err.Error(), "catalog cleanup failed") {
		t.Fatalf("CreateModel error = %v, want catalog cleanup failure surfaced from rollback", err)
	}
	if deleteSMCalls != 1 {
		t.Fatalf("semantic model delete calls = %d, want 1 when catalog cleanup fails", deleteSMCalls)
	}
	// First volume delete fails → stop; owner metadata remains (no tenant DELETE).
	if !sameStringSet(dataDomainSvc.calls, []string{
		"resolve_default_catalog", "resolve-database:kb_docs", "database:kb_docs",
		"raw_document:Knowledge base document raw source files",
		"processed:Knowledge base processed files",
		"delete-volume:12",
	}) {
		t.Fatalf("data domain calls = %+v", dataDomainSvc.calls)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceCreateModelReportsConflictingCatalogPath(t *testing.T) {
	// Name-availability fails before SemanticModels.Create; still wire a full Core
	// fixture so this test does not depend on other tests configuring the global client.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("unexpected core request before catalog name conflict: %s %s", r.Method, r.URL.String())
	}))
	defer server.Close()

	dataDomainSvc := &fakeSemanticModelDataDomainService{databaseByName: map[string]int64{"kb_docs": 11}}
	svc := newSemanticModelTestServiceWithDependencies(t, server.URL, nil, dataDomainSvc, nil)
	ctx := semanticModelServiceTestContext(i18n.LocaleZhCN.String())

	_, err := svc.CreateModel(ctx, CreateSemanticModelRequest{Name: "kb_docs"})
	if !IsServiceError(err, ErrCodeConflict) {
		t.Fatalf("CreateModel error = %v, want conflict", err)
	}
	msg, ok := i18n.Message(ctx, err)
	if !ok || msg != "知识库名称不可用：与 Catalog 数据库“默认/kb_docs”重名，请更换知识库名称" {
		t.Fatalf("localized conflict = %q, localized=%v", msg, ok)
	}
	if !sameStringSlice(dataDomainSvc.calls, []string{"resolve_default_catalog", "resolve-database:kb_docs"}) {
		t.Fatalf("data domain calls = %+v", dataDomainSvc.calls)
	}
}

func TestSemanticModelServiceCreateModelRollsBackSemanticModelWhenDomainBeginFails(t *testing.T) {
	const modelID = int64(77)
	var deleteCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id": modelID, "name": "kb_docs", "description": "docs",
				"tables": []any{}, "files": map[string]any{"file_ids": []string{}},
			})
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			deleteCalls++
			_ = json.NewEncoder(w).Encode(map[string]any{"deleted": true})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	dataDomainSvc := &fakeSemanticModelDataDomainService{}
	svc := newSemanticModelTestServiceWithDependencies(t, server.URL, nil, dataDomainSvc, nil)
	// No tenant DB → beginKnowledgeBaseDataDomain fails after SM create.
	ctx := semanticModelServiceTestContext(i18n.LocaleZhCN.String())

	_, err := svc.CreateModel(ctx, CreateSemanticModelRequest{Name: "kb_docs", Description: "docs"})
	if err == nil {
		t.Fatal("CreateModel error = nil, want domain begin failure")
	}
	if IsServiceError(err, ErrCodeConflict) {
		t.Fatalf("CreateModel error = %v, want non-conflict domain failure", err)
	}
	if deleteCalls != 1 {
		t.Fatalf("semantic model delete calls = %d, want 1", deleteCalls)
	}
	if !sameStringSlice(dataDomainSvc.calls, []string{"resolve_default_catalog", "resolve-database:kb_docs"}) {
		t.Fatalf("data domain calls = %+v", dataDomainSvc.calls)
	}
}

func TestSemanticModelServiceCreateModelRejectsInvalidCatalogIdentifier(t *testing.T) {
	svc := newSemanticModelTestService(t, "http://127.0.0.1:1", nil)
	ctx := semanticModelServiceTestContext(i18n.LocaleZhCN.String())

	_, err := svc.CreateModel(ctx, CreateSemanticModelRequest{Name: "Bad-Name"})
	if !IsServiceError(err, ErrCodeBadRequest) {
		t.Fatalf("CreateModel error = %v, want bad_request for invalid catalog identifier", err)
	}
}

func TestSemanticModelServiceCreateWithSourcesMapsDuplicateNameConflict(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if isKnowledgeBaseEmbeddingModelsRequest(r) {
			writeKnowledgeBaseEmbeddingModelsResponse(t, w, 42)
			return
		}
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/workspaces/ws-1/semantic-models" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(&common.ErrorResponse{
			Code:      common.ErrorCode_ALREADY_EXISTS,
			Message:   "semantic model already exists",
			RequestId: "req-duplicate",
		})
	}))
	defer server.Close()

	configureSemanticModelTestCore(t, server.URL)
	svc := newSemanticModelTestServiceWithKnowledgeBaseRuntimeDependencies(t, server.URL, nil, &fakeSemanticModelDataDomainService{}, nil, nil, nil, nil)
	ctx := semanticModelServiceTestContext(i18n.LocaleZhCN.String())

	_, err := svc.CreateModelWithSources(ctx, CreateSemanticModelWithSourcesRequest{
		Name:        "kb_docs",
		Description: "docs",
		Sources: []CreateSemanticModelSourceRequest{{
			SourceType: kbSourceTypeCatalogTable,
			TableID:    1001,
		}},
	})
	if !IsServiceError(err, ErrCodeConflict) {
		t.Fatalf("CreateModelWithSources error = %v, want conflict service error", err)
	}
	msg, ok := i18n.Message(ctx, err)
	if !ok {
		t.Fatalf("CreateModelWithSources error is not localized: %T", err)
	}
	if msg != "语义模型名称已存在" {
		t.Fatalf("localized message = %q", msg)
	}
	if strings.Contains(msg, "request_id") || strings.Contains(msg, "semantic model already exists") {
		t.Fatalf("localized message leaked raw core error: %q", msg)
	}
}

func TestSemanticModelServiceCreateWithSourcesResolvesDefaultCatalogBeforeCreate(t *testing.T) {
	semanticModelCreateCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isKnowledgeBaseEmbeddingModelsRequest(r) {
			writeKnowledgeBaseEmbeddingModelsResponse(t, w, 42)
			return
		}
		semanticModelCreateCalled = true
		t.Fatalf("unexpected semantic model request before default catalog resolves: %s %s", r.Method, r.URL.String())
	}))
	defer server.Close()

	dataDomainSvc := &fakeSemanticModelDataDomainService{defaultCatalogErr: errors.New("default catalog is not available")}
	configureSemanticModelTestCore(t, server.URL)
	svc := newSemanticModelTestServiceWithKnowledgeBaseRuntimeDependencies(t, server.URL, nil, dataDomainSvc, nil, nil, nil, nil)
	ctx := semanticModelServiceTestContext(i18n.LocaleZhCN.String())

	_, err := svc.CreateModelWithSources(ctx, CreateSemanticModelWithSourcesRequest{
		Name:        "kb_docs",
		Description: "docs",
		Sources: []CreateSemanticModelSourceRequest{{
			SourceType: kbSourceTypeCatalogTable,
			TableID:    1001,
		}},
	})
	if err == nil || !strings.Contains(err.Error(), "resolve default catalog: default catalog is not available") {
		t.Fatalf("CreateModelWithSources error = %v, want default catalog error", err)
	}
	if semanticModelCreateCalled {
		t.Fatal("semantic model create should not be called when default catalog is unavailable")
	}
	if len(dataDomainSvc.calls) != 1 || dataDomainSvc.calls[0] != "resolve_default_catalog" {
		t.Fatalf("data domain calls = %+v, want default catalog resolution only", dataDomainSvc.calls)
	}
}

func TestSemanticModelServiceCreateWithCatalogTableSourcesUsesDefaultCatalog(t *testing.T) {
	semanticModelCreateCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isKnowledgeBaseEmbeddingModelsRequest(r) {
			writeKnowledgeBaseEmbeddingModelsResponse(t, w, 42)
			return
		}
		semanticModelCreateCalled = true
		http.Error(w, "stop after default catalog resolution", http.StatusInternalServerError)
	}))
	defer server.Close()

	dataDomainSvc := &fakeSemanticModelDataDomainService{
		defaultCatalogID: 9,
	}
	configureSemanticModelTestCore(t, server.URL)
	svc := newSemanticModelTestServiceWithKnowledgeBaseRuntimeDependencies(t, server.URL, nil, dataDomainSvc, nil, nil, nil, nil)
	ctx := semanticModelServiceTestContext(i18n.LocaleZhCN.String())

	_, err := svc.CreateModelWithSources(ctx, CreateSemanticModelWithSourcesRequest{
		Name:        "kb_docs",
		Description: "docs",
		Sources: []CreateSemanticModelSourceRequest{{
			SourceType: kbSourceTypeCatalogTable,
			TableID:    1001,
		}},
	})
	if err == nil {
		t.Fatal("CreateModelWithSources error = nil, want semantic model create error")
	}
	if !semanticModelCreateCalled {
		t.Fatal("semantic model create should be called after default catalog resolution")
	}
	if !reflect.DeepEqual(dataDomainSvc.calls, []string{"resolve_default_catalog", "resolve-database:kb_docs"}) {
		t.Fatalf("data domain calls = %+v, want catalog ensure then name availability resolve", dataDomainSvc.calls)
	}
}

func TestSemanticModelServiceCreateWithSourcesChecksEmbeddingCapabilityBeforeSideEffects(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isKnowledgeBaseEmbeddingModelsRequest(r) {
			t.Fatalf("unexpected request before embedding capability resolves: %s %s", r.Method, r.URL.String())
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"models": []any{}})
	}))
	defer server.Close()

	dataDomainSvc := &fakeSemanticModelDataDomainService{}
	fileSvc := &fakeSemanticModelFileService{}
	configureSemanticModelTestCore(t, server.URL)
	svc := newSemanticModelTestServiceWithKnowledgeBaseRuntimeDependencies(t, server.URL, nil, dataDomainSvc, fileSvc, nil, nil, nil)
	ctx := semanticModelServiceTestContext(i18n.LocaleZhCN.String())

	_, err := svc.CreateModelWithSources(ctx, CreateSemanticModelWithSourcesRequest{
		Name: "kb_docs",
		Sources: []CreateSemanticModelSourceRequest{{
			SourceType: kbSourceTypeLocalFile,
			FileName:   "docs.txt",
			FileID:     "uploaded-file",
		}},
	})
	if !IsServiceError(err, ErrCodeBadRequest) {
		t.Fatalf("CreateModelWithSources error = %v, want missing fixed model error", err)
	}
	if len(dataDomainSvc.calls) != 0 {
		t.Fatalf("data domain calls = %+v, want no catalog side effects", dataDomainSvc.calls)
	}
}

func TestResolveKnowledgeBaseCreateEmbeddingConfigLocalizesCapabilityFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isKnowledgeBaseEmbeddingModelsRequest(r) {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		http.Error(w, "secret upstream failure", http.StatusInternalServerError)
	}))
	defer server.Close()

	client, err := moi.New(server.URL, "caller-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer client.Close()

	_, err = resolveKnowledgeBaseCreateEmbeddingConfig(context.Background(), client, "ws-1", false)
	if !IsServiceError(err, ErrCodeInternal) {
		t.Fatalf("resolve error = %v, want internal localized service error", err)
	}
	for _, locale := range []struct {
		name string
		want string
	}{
		{name: i18n.LocaleZhCN.String(), want: "无法验证知识库所需的向量模型，请稍后重试"},
		{name: i18n.LocaleEnUS.String(), want: "Unable to verify the embedding models required by the knowledge base. Please try again later"},
	} {
		ctx := semanticModelServiceTestContext(locale.name)
		message, ok := i18n.Message(ctx, err)
		if !ok || message != locale.want {
			t.Fatalf("locale %s message = %q, ok=%v", locale.name, message, ok)
		}
		if strings.Contains(message, "secret upstream failure") {
			t.Fatalf("locale %s leaked upstream error: %q", locale.name, message)
		}
	}
}

func TestSemanticModelServiceCreateWithSourcesCreatesDomainCopiesFilesAndDirectlyLinksTables(t *testing.T) {
	semanticUpdateCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case serveCatalogTableDetail(t, w, r, 1001, 3, 11, "catalog", "sales", "orders"):
		case isKnowledgeBaseEmbeddingModelsRequest(r):
			writeKnowledgeBaseEmbeddingModelsResponse(t, w, 42)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models":
			requireSemanticModelExecutionHeaders(t, r)
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 77, "name": "kb_docs", "description": "docs", "tables": []any{}, "files": map[string]any{"file_ids": []string{}}})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			t.Fatalf("unexpected semantic model GET during create-with-sources short request")
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/ws-1/files/source-file/download":
			requireSemanticModelExecutionHeaders(t, r)
			w.Header().Set("Content-Disposition", `attachment; filename="source.pdf"`)
			w.Header().Set("Content-Type", "application/pdf")
			_, _ = w.Write([]byte("source content"))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/ws-1/volumes/12/files/detail":
			t.Fatalf("unexpected raw-volume detail lookup during create-with-sources short request")
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/workspaces/ws-1/volumes/12/files":
			t.Fatalf("unexpected AddFiles during create-with-sources short request")
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/workspaces/ws-1/volumes/12/files/trigger":
			t.Fatalf("unexpected TriggerFiles during create-with-sources short request")
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			requireSemanticModelExecutionHeaders(t, r)
			var req struct {
				Tables []semanticModelTableSource `json:"tables"`
				Files  struct {
					FileIDs        []string `json:"file_ids"`
					VectorTable    string   `json:"vector_table"`
					EmbeddingModel string   `json:"embedding_model"`
				} `json:"files"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode semantic update: %v", err)
			}
			if len(req.Files.FileIDs) != 0 {
				t.Fatalf("semantic files update = %+v", req.Files.FileIDs)
			}
			if req.Files.VectorTable != "kb_77_text_index" {
				t.Fatalf("semantic files vector_table = %q", req.Files.VectorTable)
			}
			if req.Files.EmbeddingModel != "bge-m3" {
				t.Fatalf("semantic files embedding_model = %q", req.Files.EmbeddingModel)
			}
			semanticUpdateCount++
			if semanticUpdateCount != 1 {
				t.Fatalf("unexpected semantic update count %d with tables %+v", semanticUpdateCount, req.Tables)
			}
			if len(req.Tables) != 1 || req.Tables[0].DBName != "sales" || !sameStringSet(req.Tables[0].TableNames, []string{"orders"}) {
				t.Fatalf("semantic tables update = %+v", req.Tables)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"updated": true})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	dataDomainSvc := &fakeSemanticModelDataDomainService{
		databaseID: 11,
		volumeIDs:  []int64{12, 13, 14},
	}
	localImportSvc := &queuedSemanticModelLocalFileImportService{results: []KnowledgeBaseLocalFileImportResult{{
		TaskID:  "task-structured-create",
		FileIDs: []string{"conn-structured-file"},
	}}}
	workflowSvc := &fakeSemanticModelWorkflowService{}
	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()
	svc := newSemanticModelTestServiceWithKnowledgeBaseRuntimeDependencies(t, server.URL, systemClient, dataDomainSvc, nil, localImportSvc, &fakeSemanticModelWorkflowTemplateService{}, workflowSvc)

	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	tenantMock.ExpectExec("INSERT INTO knowledge_base_data_domains").
		WillReturnResult(sqlmock.NewResult(1, 1))
	expectKnowledgeBaseCreateDomainPrepare(tenantMock, 77, 3, 11, 12, 13)
	tenantMock.ExpectBegin()
	tenantMock.ExpectExec("INSERT INTO knowledge_base_sources").
		WillReturnResult(sqlmock.NewResult(1, 1))
	tenantMock.ExpectExec("INSERT INTO knowledge_base_source_job_runs").
		WillReturnResult(sqlmock.NewResult(1, 1))
	tenantMock.ExpectExec("INSERT INTO knowledge_base_source_job_runs").
		WillReturnResult(sqlmock.NewResult(2, 1))
	tenantMock.ExpectExec("INSERT INTO knowledge_base_sources").
		WillReturnResult(sqlmock.NewResult(2, 1))
	tenantMock.ExpectExec("INSERT INTO knowledge_base_source_job_runs").
		WillReturnResult(sqlmock.NewResult(3, 1))
	// Catalog metadata is always resolved on create (authoritative volume + display name).
	expectCatalogFileSourceOriginLookupMissWithMeta(tenantMock, "source-file", 99, "source.pdf")
	tenantMock.ExpectExec("INSERT INTO knowledge_base_sources").
		WillReturnResult(sqlmock.NewResult(3, 1))
	tenantMock.ExpectExec("INSERT INTO knowledge_base_source_job_runs").
		WillReturnResult(sqlmock.NewResult(4, 1))
	tenantMock.ExpectExec("INSERT INTO knowledge_base_source_job_runs").
		WillReturnResult(sqlmock.NewResult(5, 1))
	tenantMock.ExpectExec("INSERT INTO knowledge_base_sources").
		WillReturnResult(sqlmock.NewResult(4, 1))
	tenantMock.ExpectCommit()
	structuredCreateSourceID := stableID("kb-source", int64(77), kbSourceTypeLocalFile, "kb-structured-file")
	structuredCreateJobID := stableID("kb-job", structuredCreateSourceID, kbJobTypeLoad)
	tenantMock.ExpectExec("UPDATE knowledge_base_source_job_runs").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), kbSourceJobRunning, exactShortOperationIDArg{value: "import_task:task-structured-create"}, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), "user-1", structuredCreateJobID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	expectKnowledgeBaseCreateDomainFinalize(tenantMock, 77, 3, 11, 12, 13)

	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithUserID(ctx, "user-1")
	ctx = withKnowledgeBaseCreatePrincipal(ctx)
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)

	structuredTableConfig := `{"database_id":11,"conn_file_ids":["conn-structured-file"],"new_table":true,"create_table":{"name":"structured_table","tableColumn":[{"column":"col","dataType":"VARCHAR","col_num_in_file":1}]}}`
	if len(structuredTableConfig) <= 128 {
		t.Fatalf("structured table config length = %d, want > 128", len(structuredTableConfig))
	}
	resp, err := svc.CreateModelWithSources(ctx, CreateSemanticModelWithSourcesRequest{
		Name:              "kb_docs",
		Description:       "docs",
		ImageIndexEnabled: true,
		Sources: []CreateSemanticModelSourceRequest{
			{SourceType: kbSourceTypeLocalFile, FileName: "local.txt", FileID: "kb-unstructured-file"},
			{SourceType: kbSourceTypeLocalFile, FileName: "structured.csv", FileID: "kb-structured-file", UploadKind: kbLocalUploadKindStructured, TableConfig: structuredTableConfig},
			{SourceType: kbSourceTypeCatalogFile, FileID: "source-file", VolumeID: 99},
			{SourceType: kbSourceTypeCatalogTable, TableID: 1001},
		},
	})
	if err != nil {
		t.Fatalf("CreateModelWithSources: %v", err)
	}
	if resp.Model.ID != 77 || resp.DataDomain.DatabaseID != 11 || resp.DataDomain.RawVolumeID != 12 || resp.DataDomain.ProcessedVolumeID != 13 {
		t.Fatalf("response = %+v", resp)
	}
	if resp.DataDomain.CatalogID != 3 {
		t.Fatalf("catalog_id = %d, want 3", resp.DataDomain.CatalogID)
	}
	if len(dataDomainSvc.calls) == 0 || dataDomainSvc.calls[0] != "resolve_default_catalog" {
		t.Fatalf("data domain calls = %+v, want default catalog resolution first", dataDomainSvc.calls)
	}
	if len(resp.Sources) != 4 || resp.Sources[0].ResourceID != "kb-unstructured-file" || resp.Sources[1].SourceType != SemanticModelSourceTypeTable || resp.Sources[1].ResourceID != "" || resp.Sources[2].ResourceID != "source-file" || resp.Sources[3].SourceType != SemanticModelSourceTypeTable || resp.Sources[3].ResourceID != "1001" || resp.Sources[3].SourceTableID == nil || *resp.Sources[3].SourceTableID != 1001 || resp.Sources[3].KBTableID != nil {
		t.Fatalf("sources = %+v", resp.Sources)
	}
	if resp.Sources[0].IngestStatus == nil || *resp.Sources[0].IngestStatus != kbSourceStatusPending {
		t.Fatalf("local source ingest status = %+v", resp.Sources[0].IngestStatus)
	}
	for _, call := range dataDomainSvc.calls {
		if strings.HasPrefix(call, "table:") {
			t.Fatalf("create should not run catalog table clone, got calls = %+v", dataDomainSvc.calls)
		}
	}
	if len(resp.Jobs) != 5 {
		t.Fatalf("jobs = %+v", resp.Jobs)
	}
	if len(workflowSvc.deploys) != 1 {
		t.Fatalf("workflow deploys = %+v, want 1", workflowSvc.deploys)
	}
	assertKnowledgeBaseWorkflowDeploy(t, workflowSvc.deploys[0], 77, 12, 13)
	if !strings.Contains(workflowSvc.deploys[0].DSLYAML, "standard-rag-image-index-pipeline") || !strings.Contains(workflowSvc.deploys[0].DSLYAML, "moi:document_visual.index.image") {
		t.Fatalf("new knowledge base should use fixed image-index RAG template, got %q", workflowSvc.deploys[0].DSLYAML)
	}
	if resp.Jobs[0].JobType != kbJobTypeLoad || resp.Jobs[0].JobStatus != kbSourceJobQueued || resp.Jobs[0].OperationID != nil {
		t.Fatalf("local load job = %+v", resp.Jobs[0])
	}
	if resp.Jobs[1].JobType != kbJobTypeRAGIngest || resp.Jobs[1].WorkflowExecutionID != nil {
		t.Fatalf("local rag job = %+v", resp.Jobs[1])
	}
	if resp.Jobs[2].JobType != kbJobTypeLoad || resp.Jobs[2].OperationID == nil || *resp.Jobs[2].OperationID != "import_task:task-structured-create" || len(*resp.Jobs[2].OperationID) > 128 || resp.Jobs[2].JobStatus != kbSourceJobRunning {
		t.Fatalf("structured load job = %+v", resp.Jobs[2])
	}
	if resp.Sources[1].SourceFileID != nil || resp.Sources[1].KBFileID != nil || resp.Jobs[2].SourceFileID != nil || resp.Jobs[2].KBFileID != nil {
		t.Fatalf("structured upload exposes file relation: source=%+v job=%+v", resp.Sources[1], resp.Jobs[2])
	}
	if resp.Jobs[3].JobType != kbJobTypeCopy || resp.Jobs[4].JobType != kbJobTypeRAGIngest {
		t.Fatalf("job relationship order = %+v", resp.Jobs)
	}
	if resp.Jobs[3].JobStatus != kbSourceJobPending || resp.Jobs[4].JobStatus != kbSourceJobPending {
		t.Fatalf("deferred catalog jobs = %+v", resp.Jobs[3:])
	}
	if len(localImportSvc.calls) != 1 {
		t.Fatalf("local import service calls = %+v", localImportSvc.calls)
	}
	if localImportSvc.calls[0].UploadKind != kbLocalUploadKindStructured || localImportSvc.calls[0].FileName != "structured.csv" || localImportSvc.calls[0].FileID != "kb-structured-file" || localImportSvc.calls[0].VolumeID != 12 {
		t.Fatalf("structured local import call = %+v", localImportSvc.calls[0])
	}
	if len(localImportSvc.calls[0].TableConfig) <= 128 || !strings.Contains(localImportSvc.calls[0].TableConfig, "conn-structured-file") || !strings.Contains(localImportSvc.calls[0].TableConfig, "structured_table") {
		t.Fatalf("structured local import table_config = %q", localImportSvc.calls[0].TableConfig)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestEnsureDataDomainUsesKnowledgeBaseNameAsDatabaseName(t *testing.T) {
	dataDomainSvc := &fakeSemanticModelDataDomainService{
		databaseID:   11,
		databaseName: "sales_aqi_kb",
		volumeIDs:    []int64{12, 13},
	}
	svc := &semanticModelService{dataDomainService: dataDomainSvc}
	domain := &KnowledgeBaseDataDomain{ModelID: 77, CatalogID: 3}

	databaseID, rawVolumeID, processedVolumeID, err := svc.ensureKnowledgeBaseDataDomainResources(context.Background(), domain, CreateSemanticModelWithSourcesRequest{
		Name:        "sales_aqi_kb",
		Description: "docs",
		Sources: []CreateSemanticModelSourceRequest{{
			SourceType: kbSourceTypeCatalogTable,
			TableID:    1001,
		}},
	})
	if err != nil {
		t.Fatalf("ensureKnowledgeBaseDataDomainResources: %v", err)
	}
	if databaseID != 11 || rawVolumeID != 12 || processedVolumeID != 13 {
		t.Fatalf("resource ids = database %d raw %d processed %d", databaseID, rawVolumeID, processedVolumeID)
	}
	wantCalls := []string{
		"database:sales_aqi_kb",
		"raw_document:Knowledge base document raw source files",
		"processed:Knowledge base processed files",
	}
	if !sameStringSlice(dataDomainSvc.calls, wantCalls) {
		t.Fatalf("data domain calls = %+v, want %+v", dataDomainSvc.calls, wantCalls)
	}
	if dataDomainSvc.databaseDisplay != "" {
		t.Fatalf("database display name = %q, want empty", dataDomainSvc.databaseDisplay)
	}
}

func TestEnsureDataDomainResourcesRejectsUnboundExistingDatabase(t *testing.T) {
	dataDomainSvc := &fakeSemanticModelDataDomainService{
		databaseID:     11,
		databaseErr:    backendcatalog.ErrDatabaseAlreadyExists,
		databaseByName: map[string]int64{"sales_aqi_kb": 11},
		volumeErrs: map[string]error{
			rawVolumeName(kbRawKindDocument): &moi.Error{Code: common.ErrorCode_ALREADY_EXISTS, Message: "Volume name already exists"},
			"processed":                      &moi.Error{Code: common.ErrorCode_ALREADY_EXISTS, Message: "Volume name already exists"},
		},
		volumeByName: map[string]int64{
			rawVolumeName(kbRawKindDocument): 12,
			"processed":                      13,
		},
	}
	svc := &semanticModelService{dataDomainService: dataDomainSvc}
	domain := &KnowledgeBaseDataDomain{ModelID: 77, CatalogID: 3}

	_, _, _, err := svc.ensureKnowledgeBaseDataDomainResources(context.Background(), domain, CreateSemanticModelWithSourcesRequest{
		Name:        "sales_aqi_kb",
		Description: "docs",
	})
	if !IsServiceError(err, ErrCodeConflict) {
		t.Fatalf("ensureKnowledgeBaseDataDomainResources error = %v, want conflict", err)
	}
	wantCalls := []string{
		"database:sales_aqi_kb",
		"resolve-database:sales_aqi_kb",
	}
	if !sameStringSlice(dataDomainSvc.calls, wantCalls) {
		t.Fatalf("data domain calls = %+v, want %+v", dataDomainSvc.calls, wantCalls)
	}
}

func TestIssue12779ExistingReadyDataDomainRestoresStaleCatalogWithoutCreatingDatabase(t *testing.T) {
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
		}).AddRow(int64(77), int64(99), int64(11), int64(12), int64(13), kbEnsureStatusReady, nil, int64(100)))
	tenantMock.ExpectExec("UPDATE knowledge_base_data_domains\\s+SET catalog_id = \\?, updated_by = \\?\\s+WHERE model_id = \\?").
		WithArgs(int64(3), "user-1", int64(77)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	// Fully bound ready domains still reconcile the document raw-volume row.
	expectUpsertKnowledgeBaseRawVolume(tenantMock, 77, kbRawKindDocument, false)

	dataDomainSvc := &fakeSemanticModelDataDomainService{databaseCatalogID: 3}
	svc := &semanticModelService{dataDomainService: dataDomainSvc}
	ctx := ctxutil.WithTenantDB(context.Background(), tenantDB)
	domain, err := svc.ensureAppendKnowledgeBaseDataDomain(ctx, "ws-1", 77, "Legacy KB", "docs", []CreateSemanticModelSourceRequest{{
		SourceType: kbSourceTypeCatalogTable,
		TableID:    1001,
	}}, "user-1", nil)
	if err != nil {
		t.Fatalf("ensureAppendKnowledgeBaseDataDomain: %v", err)
	}
	if domain == nil || domain.CatalogID != 3 || domain.DatabaseID != 11 || domain.RawVolumeID != 12 || domain.ProcessedVolumeID != 13 {
		t.Fatalf("domain = %+v", domain)
	}
	if !sameStringSlice(dataDomainSvc.calls, []string{"database_catalog:11"}) {
		t.Fatalf("data domain service calls = %+v, want database parent lookup", dataDomainSvc.calls)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestRepairKnowledgeBaseDataDomainCatalogRestoresParentFromDatabase(t *testing.T) {
	dataDomainSvc := &fakeSemanticModelDataDomainService{databaseCatalogID: 3}
	svc := &semanticModelService{dataDomainService: dataDomainSvc}
	domain := &KnowledgeBaseDataDomain{ModelID: 77, CatalogID: -1, DatabaseID: 11, RawVolumeID: 12, ProcessedVolumeID: 13, EnsureStatus: kbEnsureStatusReady}

	changed, err := svc.repairKnowledgeBaseDataDomainCatalog(context.Background(), domain)
	if err != nil {
		t.Fatalf("repairKnowledgeBaseDataDomainCatalog: %v", err)
	}
	if !changed || domain.CatalogID != 3 {
		t.Fatalf("changed=%v domain=%+v, want catalog 3", changed, domain)
	}
	if !sameStringSlice(dataDomainSvc.calls, []string{"database_catalog:11"}) {
		t.Fatalf("data domain calls = %+v", dataDomainSvc.calls)
	}
}

func TestRepairKnowledgeBaseDataDomainCatalogRestoresStalePositiveCatalogFromDatabase(t *testing.T) {
	dataDomainSvc := &fakeSemanticModelDataDomainService{databaseCatalogID: 3}
	svc := &semanticModelService{dataDomainService: dataDomainSvc}
	domain := &KnowledgeBaseDataDomain{ModelID: 77, CatalogID: 99, DatabaseID: 11, RawVolumeID: 12, ProcessedVolumeID: 13, EnsureStatus: kbEnsureStatusReady}

	changed, err := svc.repairKnowledgeBaseDataDomainCatalog(context.Background(), domain)
	if err != nil {
		t.Fatalf("repairKnowledgeBaseDataDomainCatalog: %v", err)
	}
	if !changed || domain.CatalogID != 3 {
		t.Fatalf("changed=%v domain=%+v, want catalog 3", changed, domain)
	}
	if !sameStringSlice(dataDomainSvc.calls, []string{"database_catalog:11"}) {
		t.Fatalf("data domain calls = %+v", dataDomainSvc.calls)
	}
}

func TestRepairKnowledgeBaseDataDomainCatalogKeepsMatchingParent(t *testing.T) {
	dataDomainSvc := &fakeSemanticModelDataDomainService{databaseCatalogID: 3}
	svc := &semanticModelService{dataDomainService: dataDomainSvc}
	domain := &KnowledgeBaseDataDomain{ModelID: 77, CatalogID: 3, DatabaseID: 11, RawVolumeID: 12, ProcessedVolumeID: 13, EnsureStatus: kbEnsureStatusReady}

	changed, err := svc.repairKnowledgeBaseDataDomainCatalog(context.Background(), domain)
	if err != nil {
		t.Fatalf("repairKnowledgeBaseDataDomainCatalog: %v", err)
	}
	if changed || domain.CatalogID != 3 {
		t.Fatalf("changed=%v domain=%+v, want unchanged catalog 3", changed, domain)
	}
	if !sameStringSlice(dataDomainSvc.calls, []string{"database_catalog:11"}) {
		t.Fatalf("data domain calls = %+v", dataDomainSvc.calls)
	}
}

func TestRepairKnowledgeBaseDataDomainCatalogRejectsStalePositiveCatalogWithoutDatabaseAccess(t *testing.T) {
	dataDomainSvc := &fakeSemanticModelDataDomainService{databaseCatalogErr: errors.New("database not found")}
	svc := &semanticModelService{dataDomainService: dataDomainSvc}
	ctx := i18n.WithLocale(context.Background(), i18n.LocaleZhCN)
	domain := &KnowledgeBaseDataDomain{ModelID: 77, CatalogID: 99, DatabaseID: 11, RawVolumeID: 12, ProcessedVolumeID: 13, EnsureStatus: kbEnsureStatusReady}

	changed, err := svc.repairKnowledgeBaseDataDomainCatalog(ctx, domain)
	if err == nil || changed {
		t.Fatalf("changed=%v err=%v, want repair error", changed, err)
	}
	if !IsServiceError(err, ErrCodeConflict) {
		t.Fatalf("error = %v, want conflict service error", err)
	}
	if msg, ok := i18n.Message(ctx, err); !ok || msg != "无法安全修复知识库 Catalog 关联" {
		t.Fatalf("localized repair message = %q, ok=%v", msg, ok)
	}
	if domain.CatalogID != 99 {
		t.Fatalf("catalog id = %d, want unchanged stale value", domain.CatalogID)
	}
	if !sameStringSlice(dataDomainSvc.calls, []string{"database_catalog:11"}) {
		t.Fatalf("data domain calls = %+v", dataDomainSvc.calls)
	}
}

func TestRepairKnowledgeBaseDataDomainCatalogBindsEmptyFailedDomain(t *testing.T) {
	dataDomainSvc := &fakeSemanticModelDataDomainService{defaultCatalogID: 3}
	svc := &semanticModelService{dataDomainService: dataDomainSvc}
	domain := &KnowledgeBaseDataDomain{ModelID: 77, CatalogID: -1, EnsureStatus: kbEnsureStatusFailed}

	changed, err := svc.repairKnowledgeBaseDataDomainCatalog(context.Background(), domain)
	if err != nil {
		t.Fatalf("repairKnowledgeBaseDataDomainCatalog: %v", err)
	}
	if !changed || domain.CatalogID != 3 {
		t.Fatalf("changed=%v domain=%+v, want catalog 3", changed, domain)
	}
	if !sameStringSlice(dataDomainSvc.calls, []string{"resolve_default_catalog"}) {
		t.Fatalf("data domain calls = %+v", dataDomainSvc.calls)
	}
}

func TestRepairKnowledgeBaseDataDomainCatalogRebindsLegacyEmptyDomain(t *testing.T) {
	dataDomainSvc := &fakeSemanticModelDataDomainService{defaultCatalogID: 3}
	svc := &semanticModelService{dataDomainService: dataDomainSvc}
	domain := &KnowledgeBaseDataDomain{ModelID: 77, CatalogID: 99, EnsureStatus: kbEnsureStatusFailed}

	changed, err := svc.repairKnowledgeBaseDataDomainCatalog(context.Background(), domain)
	if err != nil {
		t.Fatalf("repairKnowledgeBaseDataDomainCatalog: %v", err)
	}
	if !changed || domain.CatalogID != 3 {
		t.Fatalf("changed=%v domain=%+v, want catalog 3", changed, domain)
	}
	if !sameStringSlice(dataDomainSvc.calls, []string{"resolve_default_catalog"}) {
		t.Fatalf("data domain calls = %+v, want shared Catalog ensure", dataDomainSvc.calls)
	}
}

func TestUpdateKnowledgeBaseDataDomainCatalogRejectsDeletedDomain(t *testing.T) {
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	tenantMock.ExpectExec("UPDATE knowledge_base_data_domains\\s+SET catalog_id = \\?, updated_by = \\?\\s+WHERE model_id = \\?").
		WithArgs(int64(3), "user-1", int64(77)).
		WillReturnResult(sqlmock.NewResult(0, 0))

	ctx := ctxutil.WithTenantDB(context.Background(), tenantDB)
	err = (&semanticModelService{}).updateKnowledgeBaseDataDomainCatalog(ctx, 77, 3, "user-1")
	if err == nil || !IsServiceError(err, ErrCodeConflict) {
		t.Fatalf("updateKnowledgeBaseDataDomainCatalog error = %v, want conflict", err)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestUpdateKnowledgeBaseDataDomainRejectsDeletedDomain(t *testing.T) {
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	tenantMock.ExpectExec("UPDATE knowledge_base_data_domains").
		WillReturnResult(sqlmock.NewResult(0, 0))

	ctx := ctxutil.WithTenantDB(context.Background(), tenantDB)
	err = (&semanticModelService{}).updateKnowledgeBaseDataDomain(ctx, &KnowledgeBaseDataDomain{
		ModelID:           77,
		CatalogID:         3,
		DatabaseID:        11,
		RawVolumeID:       12,
		ProcessedVolumeID: 13,
		EnsureStatus:      kbEnsureStatusReady,
	}, "user-1")
	if err == nil || !IsServiceError(err, ErrCodeConflict) {
		t.Fatalf("updateKnowledgeBaseDataDomain error = %v, want conflict", err)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestRepairKnowledgeBaseDataDomainCatalogDoesNotCreateForUnrecoverableDomain(t *testing.T) {
	dataDomainSvc := &fakeSemanticModelDataDomainService{}
	svc := &semanticModelService{dataDomainService: dataDomainSvc}
	ctx := i18n.WithLocale(context.Background(), i18n.LocaleZhCN)
	domain := &KnowledgeBaseDataDomain{ModelID: 77, CatalogID: -1, RawVolumeID: 12}

	changed, err := svc.repairKnowledgeBaseDataDomainCatalog(ctx, domain)
	if err == nil || changed {
		t.Fatalf("changed=%v err=%v, want repair error", changed, err)
	}
	if !IsServiceError(err, ErrCodeConflict) {
		t.Fatalf("error = %v, want conflict service error", err)
	}
	if msg, ok := i18n.Message(ctx, err); !ok || msg != "无法安全修复知识库 Catalog 关联" {
		t.Fatalf("localized repair message = %q, ok=%v", msg, ok)
	}
	if len(dataDomainSvc.calls) != 0 {
		t.Fatalf("data domain calls = %+v, want no Catalog ensure", dataDomainSvc.calls)
	}
}

func TestIssue12782EnsureAppendRepairsLegacyInvalidIdentifierFailedDataDomain(t *testing.T) {
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	legacyErr := "ensure knowledge base data domain: invalid catalog identifier: name must start with a lowercase letter"
	tenantMock.ExpectQuery("SELECT model_id, catalog_id, database_id, raw_volume_id, processed_volume_id, ensure_status, last_ensure_error, last_checked_at").
		WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows([]string{
			"model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "ensure_status", "last_ensure_error", "last_checked_at",
		}).AddRow(int64(77), int64(3), int64(0), int64(0), int64(0), kbEnsureStatusFailed, legacyErr, int64(100)))
	expectKnowledgeBaseDataDomainClaimAndReady(tenantMock, 77, 3, 11, 12, 13)
	expectUpsertKnowledgeBaseRawVolume(tenantMock, 77, kbRawKindDocument, false)

	dataDomainSvc := &fakeSemanticModelDataDomainService{databaseID: 11, volumeIDs: []int64{12, 13}, databaseName: "sales_ops_analysis"}
	svc := &semanticModelService{dataDomainService: dataDomainSvc}
	ctx := ctxutil.WithTenantDB(context.Background(), tenantDB)
	domain, err := svc.ensureAppendKnowledgeBaseDataDomain(ctx, "ws-1", 77, "sales_ops_analysis", "docs", []CreateSemanticModelSourceRequest{{
		SourceType: kbSourceTypeCatalogTable,
		TableID:    1001,
	}}, "user-1", nil)
	if err != nil {
		t.Fatalf("ensureAppendKnowledgeBaseDataDomain: %v", err)
	}
	if domain == nil || domain.DatabaseID != 11 || domain.RawVolumeID != 12 || domain.ProcessedVolumeID != 13 || domain.EnsureStatus != kbEnsureStatusReady || domain.LastEnsureError != nil {
		t.Fatalf("domain = %+v", domain)
	}
	wantCalls := []string{
		"resolve_default_catalog",
		"database:sales_ops_analysis",
		"raw_document:Knowledge base document raw source files",
		"processed:Knowledge base processed files",
	}
	if !sameStringSet(dataDomainSvc.calls, wantCalls) {
		t.Fatalf("data domain calls = %+v, want %+v", dataDomainSvc.calls, wantCalls)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestEnsureAppendRepairsFailedVolumesForBoundDatabase(t *testing.T) {
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	previousErr := "ensure knowledge base data domain: Volume name already exists"
	tenantMock.ExpectQuery("SELECT model_id, catalog_id, database_id, raw_volume_id, processed_volume_id, ensure_status, last_ensure_error, last_checked_at").
		WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows([]string{
			"model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "ensure_status", "last_ensure_error", "last_checked_at",
		}).AddRow(int64(77), int64(3), int64(11), int64(0), int64(0), kbEnsureStatusFailed, previousErr, int64(100)))
	expectKnowledgeBaseDataDomainClaimAndReady(tenantMock, 77, 3, 11, 12, 13)
	expectUpsertKnowledgeBaseRawVolume(tenantMock, 77, kbRawKindDocument, false)

	dataDomainSvc := &fakeSemanticModelDataDomainService{
		databaseID: 11,
		volumeIDs:  []int64{13},
		volumeErrs: map[string]error{
			rawVolumeName(kbRawKindDocument): &moi.Error{Code: common.ErrorCode_ALREADY_EXISTS, Message: "Volume name already exists"},
		},
		volumeByName: map[string]int64{rawVolumeName(kbRawKindDocument): 12},
	}
	svc := &semanticModelService{dataDomainService: dataDomainSvc}
	ctx := ctxutil.WithTenantDB(context.Background(), tenantDB)
	domain, err := svc.ensureAppendKnowledgeBaseDataDomain(ctx, "ws-1", 77, "sales_aqi_kb", "docs", []CreateSemanticModelSourceRequest{{
		SourceType: kbSourceTypeCatalogTable,
		TableID:    1001,
	}}, "user-1", nil)
	if err != nil {
		t.Fatalf("ensureAppendKnowledgeBaseDataDomain: %v", err)
	}
	if domain == nil || domain.DatabaseID != 11 || domain.RawVolumeID != 12 || domain.ProcessedVolumeID != 13 || domain.EnsureStatus != kbEnsureStatusReady || domain.LastEnsureError != nil {
		t.Fatalf("domain = %+v", domain)
	}
	wantCalls := []string{
		"raw_document:Knowledge base document raw source files",
		"resolve-volume:raw_document",
		"processed:Knowledge base processed files",
	}
	if !sameStringSlice(dataDomainSvc.calls, wantCalls) {
		t.Fatalf("data domain calls = %+v, want %+v", dataDomainSvc.calls, wantCalls)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func assertKnowledgeBaseWorkflowDeploy(t *testing.T, deploy KnowledgeBaseWorkflowDeployRequest, modelID, rawVolumeID, processedVolumeID int64) {
	t.Helper()

	if deploy.WorkflowID != knowledgeBaseWorkflowID("ws-1", modelID) {
		t.Fatalf("workflow_id = %q", deploy.WorkflowID)
	}
	if deploy.ExecutionMode != workflowv2.ExecutionModeOneShot || deploy.RawVolumeID != 0 || deploy.TriggerEnabled {
		t.Fatalf("workflow execution mode = %q, volume %d, trigger enabled %v", deploy.ExecutionMode, deploy.RawVolumeID, deploy.TriggerEnabled)
	}
	if deploy.AutoDispatchEnabled {
		t.Fatalf("workflow auto dispatch enabled = %v, want false", deploy.AutoDispatchEnabled)
	}
	if !strings.Contains(deploy.DSLYAML, "moi:document.parse") || !strings.Contains(deploy.DSLYAML, "moi:knowledge.index.build") {
		t.Fatalf("workflow dsl = %q", deploy.DSLYAML)
	}
	sourceRef, ok := deploy.DefaultValues["source_ref"].(map[string]any)
	if !ok || sourceRef["volume_id"] != rawVolumeID {
		t.Fatalf("source_ref = %+v", deploy.DefaultValues["source_ref"])
	}
	if _, exists := sourceRef["ids"]; exists {
		t.Fatalf("source_ref.ids must not be set: %+v", sourceRef)
	}
	outputRef, ok := deploy.DefaultValues["output_ref"].(map[string]any)
	if !ok || outputRef["volume_id"] != processedVolumeID {
		t.Fatalf("output_ref = %+v", deploy.DefaultValues["output_ref"])
	}
	if _, exists := outputRef["ids"]; exists {
		t.Fatalf("output_ref.ids must not be set: %+v", outputRef)
	}
	vectorIndex, ok := deploy.DefaultValues["vector_index"].(map[string]any)
	if !ok || vectorIndex["vector_table"] != defaultKnowledgeBaseVectorTable(modelID) {
		t.Fatalf("vector_index = %+v", deploy.DefaultValues["vector_index"])
	}
	if vectorIndex["embedding_model"] != kbDefaultEmbeddingModel {
		t.Fatalf("embedding_model = %+v", vectorIndex["embedding_model"])
	}
}

func TestCreateModelWithSourcesFilesUsesBackendOwnedVectorTables(t *testing.T) {
	createFiles, err := semanticModelCreateFilesBase(
		json.RawMessage(`{
			"file_ids": ["client-file"],
			"vector_table": "kb_custom_text_idx",
			"embedding_model": "BAAI/bge-large-zh-v1.5",
			"image_vector_table": "kb_custom_text_idx_img",
			"image_embedding_model": "efficientnet-b3",
			"image_embedding_backend_id": "-30010",
			"image_embedding_dimension": 1536,
			"image_preprocess_version": "efficientnet-b3-v1-rgb-300-letterbox-imagenet",
			"image_distance_metric": "cosine"
		}`),
		77,
	)
	if err != nil {
		t.Fatalf("semanticModelCreateFilesBase: %v", err)
	}
	files, err := appendSemanticModelFiles(
		createFiles,
		77,
		[]string{"kb-file-1"},
	)
	if err != nil {
		t.Fatalf("appendSemanticModelFiles: %v", err)
	}
	var out struct {
		FileIDs                 []string `json:"file_ids"`
		VectorTable             string   `json:"vector_table"`
		EmbeddingModel          string   `json:"embedding_model"`
		ImageVectorTable        string   `json:"image_vector_table"`
		ImageEmbeddingModel     string   `json:"image_embedding_model"`
		ImageEmbeddingBackendID string   `json:"image_embedding_backend_id"`
		ImageEmbeddingDimension int      `json:"image_embedding_dimension"`
		ImagePreprocessVersion  string   `json:"image_preprocess_version"`
		ImageDistanceMetric     string   `json:"image_distance_metric"`
	}
	if err := json.Unmarshal(files, &out); err != nil {
		t.Fatalf("unmarshal files: %v", err)
	}
	if len(out.FileIDs) != 1 || out.FileIDs[0] != "kb-file-1" {
		t.Fatalf("file_ids = %+v", out.FileIDs)
	}
	if out.VectorTable != defaultKnowledgeBaseVectorTable(77) || out.EmbeddingModel != "BAAI/bge-large-zh-v1.5" {
		t.Fatalf("text vector config = %+v", out)
	}
	if out.ImageVectorTable != defaultKnowledgeBaseImageVectorTable(77) || out.ImageEmbeddingModel != "efficientnet-b3" || out.ImageEmbeddingBackendID != "-30010" || out.ImageEmbeddingDimension != 1536 || out.ImagePreprocessVersion != "efficientnet-b3-v1-rgb-300-letterbox-imagenet" || out.ImageDistanceMetric != "cosine" {
		t.Fatalf("image vector config = %+v", out)
	}

	defaults := knowledgeBaseWorkflowDefaultValues(
		&KnowledgeBaseDataDomain{RawVolumeID: 12, ProcessedVolumeID: 13},
		77,
		files,
	)
	if defaults["vlm_ocr_model"] != kbDefaultVLMOCRModel {
		t.Fatalf("workflow vlm_ocr_model = %+v", defaults["vlm_ocr_model"])
	}
	vectorIndex, ok := defaults["vector_index"].(map[string]any)
	if !ok {
		t.Fatalf("vector_index = %+v", defaults["vector_index"])
	}
	if vectorIndex["vector_table"] != defaultKnowledgeBaseVectorTable(77) || vectorIndex["embedding_model"] != "BAAI/bge-large-zh-v1.5" {
		t.Fatalf("workflow text vector_index = %+v", vectorIndex)
	}
	if vectorIndex["image_index_enabled"] != true || vectorIndex["image_vector_table"] != defaultKnowledgeBaseImageVectorTable(77) || vectorIndex["image_embedding_model"] != "efficientnet-b3" || vectorIndex["image_embedding_backend_id"] != "-30010" || vectorIndex["image_embedding_dimension"] != 1536 || vectorIndex["image_preprocess_version"] != "efficientnet-b3-v1-rgb-300-letterbox-imagenet" || vectorIndex["image_distance_metric"] != "cosine" {
		t.Fatalf("workflow vector_index image fields = %+v", vectorIndex)
	}
	imageIndex, ok := defaults["image_index"].(map[string]any)
	if !ok {
		t.Fatalf("image_index = %+v", defaults["image_index"])
	}
	if imageIndex["enabled"] != true || imageIndex["image_vector_table"] != defaultKnowledgeBaseImageVectorTable(77) || imageIndex["image_embedding_model"] != "efficientnet-b3" || imageIndex["image_embedding_backend_id"] != "-30010" || imageIndex["image_embedding_dimension"] != 1536 || imageIndex["image_preprocess_version"] != "efficientnet-b3-v1-rgb-300-letterbox-imagenet" || imageIndex["image_distance_metric"] != "cosine" {
		t.Fatalf("workflow image_index = %+v", imageIndex)
	}
	if got := knowledgeBaseWorkflowTemplateKey(files); got != kbStandardRAGImageTemplateKey {
		t.Fatalf("workflow template key = %q, want image index template", got)
	}

	textOnlyCreateFiles, err := semanticModelCreateFilesBase(json.RawMessage(`{"file_ids":["client-file"],"vector_table":"kb_custom_text_idx","embedding_model":"BAAI/bge-large-zh-v1.5"}`), 77)
	if err != nil {
		t.Fatalf("semanticModelCreateFilesBase text-only: %v", err)
	}
	textOnlyFiles, err := appendSemanticModelFiles(textOnlyCreateFiles, 77, nil)
	if err != nil {
		t.Fatalf("appendSemanticModelFiles text-only: %v", err)
	}
	textOnlyDefaults := knowledgeBaseWorkflowDefaultValues(
		&KnowledgeBaseDataDomain{RawVolumeID: 12, ProcessedVolumeID: 13},
		77,
		textOnlyFiles,
	)
	textOnlyImageIndex, ok := textOnlyDefaults["image_index"].(map[string]any)
	if !ok {
		t.Fatalf("text-only image_index = %+v", textOnlyDefaults["image_index"])
	}
	if textOnlyImageIndex["enabled"] != false {
		t.Fatalf("text-only image_index should be disabled = %+v", textOnlyImageIndex)
	}
	textOnlyVectorIndex, ok := textOnlyDefaults["vector_index"].(map[string]any)
	if !ok {
		t.Fatalf("text-only vector_index = %+v", textOnlyDefaults["vector_index"])
	}
	if textOnlyVectorIndex["image_index_enabled"] != false || textOnlyVectorIndex["image_vector_table"] != defaultKnowledgeBaseImageVectorTable(77) || textOnlyVectorIndex["image_embedding_model"] != "" || textOnlyVectorIndex["image_embedding_backend_id"] != "" || textOnlyVectorIndex["image_embedding_dimension"] != 0 || textOnlyVectorIndex["image_preprocess_version"] != "" || textOnlyVectorIndex["image_distance_metric"] != "" {
		t.Fatalf("text-only vector_index should carry disabled image defaults = %+v", textOnlyVectorIndex)
	}
	if got := knowledgeBaseWorkflowTemplateKey(textOnlyFiles); got != kbStandardRAGTemplateKey {
		t.Fatalf("text-only workflow template key = %q, want standard RAG", got)
	}
	noImageCreateFiles, err := semanticModelCreateFilesBase(json.RawMessage(`{"file_ids":[],"vector_table":"kb_custom_text_idx","embedding_model":"BAAI/bge-large-zh-v1.5","image_vector_table":"kb_custom_text_idx_img"}`), 77)
	if err != nil {
		t.Fatalf("semanticModelCreateFilesBase no-image-model: %v", err)
	}
	noImageModel, err := appendSemanticModelFiles(noImageCreateFiles, 77, nil)
	if err != nil {
		t.Fatalf("appendSemanticModelFiles no-image-model: %v", err)
	}
	var noImageOut struct {
		ImageVectorTable string `json:"image_vector_table"`
	}
	if err := json.Unmarshal(noImageModel, &noImageOut); err != nil {
		t.Fatalf("unmarshal no-image model files: %v", err)
	}
	if noImageOut.ImageVectorTable != "" {
		t.Fatalf("client-only image_vector_table should not be persisted, got %q", noImageOut.ImageVectorTable)
	}
	noImageModelDefaults := knowledgeBaseWorkflowDefaultValues(&KnowledgeBaseDataDomain{RawVolumeID: 12, ProcessedVolumeID: 13}, 77, noImageModel)
	noImageModelIndex, ok := noImageModelDefaults["image_index"].(map[string]any)
	if !ok {
		t.Fatalf("no-image-model image_index = %+v", noImageModelDefaults["image_index"])
	}
	if noImageModelIndex["enabled"] != false {
		t.Fatalf("image_index without image embedding model should be disabled = %+v", noImageModelIndex)
	}
	if got := knowledgeBaseWorkflowTemplateKey(noImageModel); got != kbStandardRAGTemplateKey {
		t.Fatalf("no-image-model workflow template key = %q, want standard RAG", got)
	}

	partialImageCreateFiles, err := semanticModelCreateFilesBase(json.RawMessage(`{"image_embedding_model":"efficientnet-b3"}`), 77)
	if err != nil {
		t.Fatalf("semanticModelCreateFilesBase partial-image: %v", err)
	}
	partialImageModel, err := appendSemanticModelFiles(partialImageCreateFiles, 77, nil)
	if err != nil {
		t.Fatalf("appendSemanticModelFiles partial-image: %v", err)
	}
	var partialImageOut struct {
		ImageVectorTable string `json:"image_vector_table"`
	}
	if err := json.Unmarshal(partialImageModel, &partialImageOut); err != nil {
		t.Fatalf("unmarshal partial-image model files: %v", err)
	}
	if partialImageOut.ImageVectorTable != "" {
		t.Fatalf("partial image embedding config should not bind image_vector_table, got %q", partialImageOut.ImageVectorTable)
	}
	if got := knowledgeBaseWorkflowTemplateKey(partialImageModel); got != kbStandardRAGTemplateKey {
		t.Fatalf("partial-image workflow template key = %q, want standard RAG", got)
	}
}

func TestCreateModelWithSourcesFilesUsesFixedIndexConfig(t *testing.T) {
	const resolvedImageBackendID = "42"
	createFiles, err := semanticModelCreateFilesBaseWithFixedIndex(
		json.RawMessage(`{
			"file_ids":["client-file"],
			"embedding_model":"client-text-model",
			"image_embedding_model":"client-image-model",
			"image_embedding_backend_id":"client-backend",
			"image_embedding_dimension":12,
			"image_preprocess_version":"client-preprocess",
			"image_distance_metric":"dot"
		}`),
		77,
		true,
		resolvedImageBackendID,
	)
	if err != nil {
		t.Fatalf("semanticModelCreateFilesBaseWithFixedIndex: %v", err)
	}

	var out semanticModelFilesPayload
	if err := json.Unmarshal(createFiles, &out); err != nil {
		t.Fatalf("unmarshal fixed create files: %v", err)
	}
	if len(out.FileIDs) != 0 {
		t.Fatalf("file_ids = %+v, want backend-managed empty list", out.FileIDs)
	}
	if out.VectorTable != defaultKnowledgeBaseVectorTable(77) || out.EmbeddingModel != kbDefaultEmbeddingModel {
		t.Fatalf("text index config = %+v", out)
	}
	if out.ImageVectorTable != defaultKnowledgeBaseImageVectorTable(77) ||
		out.ImageEmbeddingModel != kbDefaultImageEmbeddingModel ||
		out.ImageEmbeddingBackendID != resolvedImageBackendID ||
		out.ImageEmbeddingDimension != kbDefaultImageEmbeddingDimension ||
		out.ImagePreprocessVersion != kbDefaultImagePreprocessVersion ||
		out.ImageDistanceMetric != kbDefaultImageDistanceMetric {
		t.Fatalf("image index config = %+v", out)
	}
	if got := knowledgeBaseWorkflowTemplateKey(createFiles); got != kbStandardRAGImageTemplateKey {
		t.Fatalf("workflow template key = %q, want %q", got, kbStandardRAGImageTemplateKey)
	}
}

func TestCreateModelWithSourcesFilesSkipsImageIndexByDefault(t *testing.T) {
	createFiles, err := semanticModelCreateFilesBaseWithFixedIndex(
		json.RawMessage(`{
			"image_vector_table":"client-image-index",
			"image_embedding_model":"client-image-model",
			"image_embedding_backend_id":"client-backend",
			"image_embedding_dimension":12,
			"image_preprocess_version":"client-preprocess",
			"image_distance_metric":"dot",
			"image_index_configs":[{"id":"client-image-config"}]
		}`),
		77,
		false,
		"",
	)
	if err != nil {
		t.Fatalf("semanticModelCreateFilesBaseWithFixedIndex: %v", err)
	}

	var out map[string]json.RawMessage
	if err := json.Unmarshal(createFiles, &out); err != nil {
		t.Fatalf("unmarshal fixed create files: %v", err)
	}
	if stringJSONValue(out, "embedding_model") != kbDefaultEmbeddingModel {
		t.Fatalf("embedding_model = %q, want %q", stringJSONValue(out, "embedding_model"), kbDefaultEmbeddingModel)
	}
	for _, key := range []string{
		"image_vector_table",
		"image_embedding_model",
		"image_embedding_backend_id",
		"image_embedding_dimension",
		"image_preprocess_version",
		"image_distance_metric",
		"image_index_configs",
	} {
		if _, ok := out[key]; ok {
			t.Fatalf("%s should be omitted when image indexing is disabled: %+v", key, out)
		}
	}
	if got := knowledgeBaseWorkflowTemplateKey(createFiles); got != kbStandardRAGTemplateKey {
		t.Fatalf("workflow template key = %q, want %q", got, kbStandardRAGTemplateKey)
	}
}

func sameStringSet(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	seen := make(map[string]int, len(got))
	for _, item := range got {
		seen[item]++
	}
	for _, item := range want {
		if seen[item] == 0 {
			return false
		}
		seen[item]--
	}
	return true
}

func stringSliceFromAny(value any) ([]string, bool) {
	items, ok := value.([]any)
	if !ok {
		return nil, false
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		text, ok := item.(string)
		if !ok {
			return nil, false
		}
		out = append(out, text)
	}
	return out, true
}

func TestEnsureAppendClearsFailedReadyDataDomain(t *testing.T) {
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
		}).AddRow(int64(77), int64(3), int64(11), int64(12), int64(13), kbEnsureStatusFailed, "create volume failed", int64(100)))
	expectClaimKnowledgeBaseDataDomainProvision(tenantMock, 77)
	expectUpdateKnowledgeBaseDataDomainReadyCAS(tenantMock, 3, 11, 12, 13, 77)
	expectUpsertKnowledgeBaseRawVolume(tenantMock, 77, kbRawKindDocument, false)

	dataDomainSvc := &fakeSemanticModelDataDomainService{}
	svc := &semanticModelService{dataDomainService: dataDomainSvc}
	ctx := ctxutil.WithTenantDB(context.Background(), tenantDB)

	domain, err := svc.ensureAppendKnowledgeBaseDataDomain(ctx, "ws-1", 77, "kb_docs", "docs", []CreateSemanticModelSourceRequest{{
		SourceType: kbSourceTypeCatalogTable,
		TableID:    1001,
	}}, "user-1", nil)
	if err != nil {
		t.Fatalf("ensureAppendKnowledgeBaseDataDomain: %v", err)
	}
	if domain == nil || domain.EnsureStatus != kbEnsureStatusReady || domain.LastEnsureError != nil || domain.DatabaseID != 11 || domain.RawVolumeID != 12 || domain.ProcessedVolumeID != 13 {
		t.Fatalf("domain = %+v", domain)
	}
	if len(dataDomainSvc.calls) != 0 {
		t.Fatalf("data domain service calls = %+v, want none", dataDomainSvc.calls)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestEnsureAppendFailsClosedWhenExistingWorkflowIsUnavailable(t *testing.T) {
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
	// Ready fully-bound domains still reconcile the document raw-volume row.
	expectUpsertKnowledgeBaseRawVolume(tenantMock, 77, kbRawKindDocument, false)
	requireErr := errors.New("existing knowledge base workflow is unavailable")
	dataDomainSvc := &fakeSemanticModelDataDomainService{}
	workflowSvc := &fakeSemanticModelWorkflowService{requireErr: requireErr}
	svc := &semanticModelService{dataDomainService: dataDomainSvc, workflowService: workflowSvc}
	ctx := ctxutil.WithTenantDB(context.Background(), tenantDB)

	_, err = svc.ensureAppendKnowledgeBaseDataDomain(ctx, "ws-1", 77, "kb_docs", "docs", []CreateSemanticModelSourceRequest{{
		SourceType: kbSourceTypeCatalogFile,
		FileID:     "source-file",
		VolumeID:   41,
	}}, "user-1", nil)
	if !errors.Is(err, requireErr) {
		t.Fatalf("ensureAppendKnowledgeBaseDataDomain error = %v, want %v", err, requireErr)
	}
	if !reflect.DeepEqual(workflowSvc.requires, []string{knowledgeBaseWorkflowID("ws-1", 77)}) {
		t.Fatalf("required workflows = %+v", workflowSvc.requires)
	}
	if len(workflowSvc.deploys) != 0 {
		t.Fatalf("deploys = %+v, want none for non-not-found require errors", workflowSvc.deploys)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestEnsureAppendDeploysDocumentWorkflowWhenMissing(t *testing.T) {
	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	// Empty-create path: no data domain yet. Append must ensure domain resources
	// and deploy the missing document workflow instead of fail-closed Require.
	tenantMock.ExpectQuery("SELECT model_id, catalog_id, database_id, raw_volume_id, processed_volume_id, ensure_status, last_ensure_error, last_checked_at").
		WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows([]string{
			"model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "ensure_status", "last_ensure_error", "last_checked_at",
		}))
	tenantMock.ExpectExec("INSERT INTO knowledge_base_data_domains").
		WillReturnResult(sqlmock.NewResult(1, 1))
	expectKnowledgeBaseDataDomainClaimAndReady(tenantMock, 77, 3, 11, 12, 13)
	expectUpsertKnowledgeBaseRawVolume(tenantMock, 77, kbRawKindDocument, false)

	notFoundErr := &moi.Error{Code: common.ErrorCode_NOT_FOUND, Message: "workflow not found"}
	dataDomainSvc := &fakeSemanticModelDataDomainService{databaseID: 11, volumeIDs: []int64{12, 13}}
	workflowTemplateSvc := &fakeSemanticModelWorkflowTemplateService{}
	workflowSvc := &fakeSemanticModelWorkflowService{requireErr: notFoundErr}
	svc := &semanticModelService{
		dataDomainService:       dataDomainSvc,
		workflowTemplateService: workflowTemplateSvc,
		workflowService:         workflowSvc,
	}
	ctx := ctxutil.WithTenantDB(context.Background(), tenantDB)
	files := json.RawMessage(`{"file_ids":[],"vector_table":"kb_77_vector","embedding_model":"BAAI/bge-m3"}`)

	domain, err := svc.ensureAppendKnowledgeBaseDataDomain(ctx, "ws-1", 77, "kb_docs", "docs", []CreateSemanticModelSourceRequest{{
		SourceType: kbSourceTypeCatalogFile,
		FileID:     "source-file",
		VolumeID:   41,
	}}, "user-1", files)
	if err != nil {
		t.Fatalf("ensureAppendKnowledgeBaseDataDomain: %v", err)
	}
	if domain == nil || domain.EnsureStatus != kbEnsureStatusReady || domain.DatabaseID != 11 || domain.RawVolumeID != 12 || domain.ProcessedVolumeID != 13 {
		t.Fatalf("domain = %+v", domain)
	}
	if !reflect.DeepEqual(workflowSvc.requires, []string{knowledgeBaseWorkflowID("ws-1", 77)}) {
		t.Fatalf("required workflows = %+v", workflowSvc.requires)
	}
	if len(workflowSvc.deploys) != 1 {
		t.Fatalf("deploys = %+v, want one missing-workflow deploy", workflowSvc.deploys)
	}
	if got, want := workflowSvc.deploys[0].WorkflowID, knowledgeBaseWorkflowID("ws-1", 77); got != want {
		t.Fatalf("deployed workflow id = %q, want %q", got, want)
	}
	if len(workflowTemplateSvc.calls) == 0 {
		t.Fatalf("workflow template was not loaded for missing-workflow ensure")
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestEnsureAppendRepairsIncompleteFailedDataDomain(t *testing.T) {
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
		}).AddRow(int64(77), int64(3), int64(11), int64(0), int64(13), kbEnsureStatusFailed, "create volume failed", int64(100)))
	expectKnowledgeBaseDataDomainClaimAndReady(tenantMock, 77, 3, 11, 12, 13)
	expectUpsertKnowledgeBaseRawVolume(tenantMock, 77, kbRawKindDocument, false)

	dataDomainSvc := &fakeSemanticModelDataDomainService{databaseID: 11, volumeIDs: []int64{12}}
	svc := &semanticModelService{dataDomainService: dataDomainSvc}
	ctx := ctxutil.WithTenantDB(context.Background(), tenantDB)

	domain, err := svc.ensureAppendKnowledgeBaseDataDomain(ctx, "ws-1", 77, "kb_docs", "docs", []CreateSemanticModelSourceRequest{{
		SourceType: kbSourceTypeCatalogTable,
		TableID:    1001,
	}}, "user-1", nil)
	if err != nil {
		t.Fatalf("ensureAppendKnowledgeBaseDataDomain: %v", err)
	}
	if domain == nil || domain.EnsureStatus != kbEnsureStatusReady || domain.LastEnsureError != nil || domain.DatabaseID != 11 || domain.RawVolumeID != 12 || domain.ProcessedVolumeID != 13 {
		t.Fatalf("domain = %+v", domain)
	}
	wantCalls := []string{"raw_document:Knowledge base document raw source files"}
	if !sameStringSlice(dataDomainSvc.calls, wantCalls) {
		t.Fatalf("data domain calls = %+v, want %+v", dataDomainSvc.calls, wantCalls)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestEnsureAppendKnowledgeBaseDataDomainRepairsFailedStateWithoutReplacingExistingWorkflow(t *testing.T) {
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
		}).AddRow(int64(77), int64(3), int64(11), int64(12), int64(13), kbEnsureStatusFailed, "err.workflow.required_input_missing map[Input:vector_index.image_vector_table]", int64(100)))
	expectClaimKnowledgeBaseDataDomainProvision(tenantMock, 77)
	expectUpdateKnowledgeBaseDataDomainReadyCAS(tenantMock, 3, 11, 12, 13, 77)
	expectUpsertKnowledgeBaseRawVolume(tenantMock, 77, kbRawKindDocument, true)

	workflowTemplateSvc := &fakeSemanticModelWorkflowTemplateService{}
	workflowSvc := &fakeSemanticModelWorkflowService{}
	svc := &semanticModelService{
		dataDomainService:       &fakeSemanticModelDataDomainService{},
		workflowTemplateService: workflowTemplateSvc,
		workflowService:         workflowSvc,
	}
	ctx := ctxutil.WithTenantDB(context.Background(), tenantDB)
	domain, err := svc.ensureAppendKnowledgeBaseDataDomain(ctx, "ws-1", 77, "kb_docs", "docs", []CreateSemanticModelSourceRequest{{
		SourceType: kbSourceTypeCatalogFile,
		FileID:     "source-file",
		VolumeID:   41,
	}}, "user-1", nil)
	if err != nil {
		t.Fatalf("ensureAppendKnowledgeBaseDataDomain: %v", err)
	}
	if domain == nil || domain.EnsureStatus != kbEnsureStatusReady || domain.LastEnsureError != nil {
		t.Fatalf("domain = %+v", domain)
	}
	assertDocumentAppendPreservesExistingWorkflow(t, workflowTemplateSvc, workflowSvc, "ws-1", 77)
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestCreateKnowledgeBaseSourceMetadataIntentsReusesActiveLocalFileSource(t *testing.T) {
	db, mock := newSemanticModelTenantDB(t)
	ctx := ctxutil.WithTenantDB(context.Background(), db)
	sourceID := stableID("kb-source", int64(77), kbSourceTypeLocalFile, "file-1")
	existing := KnowledgeBaseSourceRecord{
		SourceID: sourceID, ModelID: 77, CatalogID: 3, DatabaseID: 11, RawVolumeID: 12, ProcessedVolumeID: 13,
		SourceType: kbSourceTypeLocalFile, SourceFileID: stringPtr("file-1"), KBFileID: stringPtr("file-1"), Status: kbSourceStatusSucceeded,
	}
	mock.ExpectBegin()
	mock.ExpectQuery("WHERE kbs.model_id = \\? AND kbs.source_id = \\?").
		WithArgs(int64(77), sourceID).
		WillReturnRows(knowledgeBaseSourceRecordRows(existing))
	mock.ExpectCommit()

	svc := &semanticModelService{}
	result, err := svc.createKnowledgeBaseSourceMetadataIntents(ctx, nil, "ws-1", 77, &KnowledgeBaseDataDomain{
		ModelID: 77, CatalogID: 3, DatabaseID: 11, RawVolumeID: 12, ProcessedVolumeID: 13,
	}, []CreateSemanticModelSourceRequest{{
		SourceType: kbSourceTypeLocalFile, FileID: "file-1", FileName: "a.pdf",
	}}, "user-1", true)
	if err != nil {
		t.Fatalf("createKnowledgeBaseSourceMetadataIntents: %v", err)
	}
	if len(result.records) != 1 || result.records[0].SourceID != sourceID || len(result.jobs) != 0 {
		t.Fatalf("result = %+v, want existing source without duplicate jobs", result)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestCreateKnowledgeBaseSourceMetadataIntentsReactivatesRemovedLocalFileSource(t *testing.T) {
	db, mock := newSemanticModelTenantDB(t)
	ctx := ctxutil.WithTenantDB(context.Background(), db)
	ctx = ctxutil.WithMoiUserID(ctx, "moi-user-create")
	ctx = ctxutil.WithCoreIAMRequest(ctx, ctxutil.CoreIAMRequestContext{VerifiedEffectiveRoleID: "role-create"})
	sourceID := stableID("kb-source", int64(77), kbSourceTypeLocalFile, "file-1")
	existing := KnowledgeBaseSourceRecord{
		SourceID: sourceID, ModelID: 77, CatalogID: 3, DatabaseID: 11, RawVolumeID: 12, ProcessedVolumeID: 13,
		SourceType: kbSourceTypeLocalFile, SourceFileID: stringPtr("file-1"), KBFileID: stringPtr("file-1"), Status: kbSourceStatusRemoved,
	}
	mock.ExpectBegin()
	mock.ExpectQuery("WHERE kbs.model_id = \\? AND kbs.source_id = \\?").
		WithArgs(int64(77), sourceID).
		WillReturnRows(knowledgeBaseSourceRecordRows(existing))
	mock.ExpectExec("UPDATE knowledge_base_sources").
		WillReturnResult(sqlmock.NewResult(0, 1))
	for range 2 {
		expectKnowledgeBaseSourceJobRunUpsertMiss(mock)
		mock.ExpectExec("INSERT INTO knowledge_base_source_job_runs").
			WillReturnResult(sqlmock.NewResult(0, 1))
	}
	mock.ExpectCommit()

	svc := &semanticModelService{}
	result, err := svc.createKnowledgeBaseSourceMetadataIntents(ctx, nil, "ws-1", 77, &KnowledgeBaseDataDomain{
		ModelID: 77, CatalogID: 3, DatabaseID: 11, RawVolumeID: 12, ProcessedVolumeID: 13,
	}, []CreateSemanticModelSourceRequest{{
		SourceType: kbSourceTypeLocalFile, FileID: "file-1", FileName: "a.pdf",
	}}, "user-1", true)
	if err != nil {
		t.Fatalf("createKnowledgeBaseSourceMetadataIntents: %v", err)
	}
	if len(result.records) != 1 || result.records[0].SourceID != sourceID || len(result.jobs) != 2 {
		t.Fatalf("result = %+v, want reactivated source and fresh jobs", result)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestSemanticModelServiceCreateWithSourcesDefersCatalogZipWithoutDownload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case isKnowledgeBaseEmbeddingModelsRequest(r):
			writeKnowledgeBaseEmbeddingModelsResponse(t, w, 42)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/ws-1/files/source-zip/download":
			t.Fatalf("unexpected DownloadWithMeta during create-with-sources short request")
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models":
			requireSemanticModelExecutionHeaders(t, r)
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 77, "name": "kb_docs", "description": "docs", "tables": []any{}, "files": map[string]any{"file_ids": []string{}}})
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
			t.Fatalf("unexpected raw-volume side effect during create-with-sources short request: %s %s", r.Method, r.URL.String())
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	dataDomainSvc := &fakeSemanticModelDataDomainService{databaseID: 11, volumeIDs: []int64{12, 13}}
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
	tenantMock.ExpectExec("INSERT INTO knowledge_base_data_domains").
		WillReturnResult(sqlmock.NewResult(1, 1))
	expectKnowledgeBaseCreateDomainPrepare(tenantMock, 77, 3, 11, 12, 13)
	tenantMock.ExpectBegin()
	expectCatalogFileSourceOriginLookupMiss(tenantMock, "source-zip")
	tenantMock.ExpectExec("INSERT INTO knowledge_base_sources").
		WillReturnResult(sqlmock.NewResult(1, 1))
	tenantMock.ExpectExec("INSERT INTO knowledge_base_source_job_runs").
		WillReturnResult(sqlmock.NewResult(1, 1))
	tenantMock.ExpectExec("INSERT INTO knowledge_base_source_job_runs").
		WillReturnResult(sqlmock.NewResult(2, 1))
	tenantMock.ExpectCommit()
	expectKnowledgeBaseCreateDomainFinalize(tenantMock, 77, 3, 11, 12, 13)

	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithUserID(ctx, "user-1")
	ctx = withKnowledgeBaseCreatePrincipal(ctx)
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)

	resp, err := svc.CreateModelWithSources(ctx, CreateSemanticModelWithSourcesRequest{
		Name:        "kb_docs",
		Description: "docs",
		Sources: []CreateSemanticModelSourceRequest{{
			SourceType: kbSourceTypeCatalogFile,
			FileID:     "source-zip",
			VolumeID:   41,
		}},
	})
	if err != nil {
		t.Fatalf("CreateModelWithSources: %v", err)
	}
	if len(resp.Jobs) != 2 || resp.Jobs[0].JobStatus != kbSourceJobPending || resp.Jobs[1].JobStatus != kbSourceJobPending {
		t.Fatalf("jobs = %+v", resp.Jobs)
	}
	wantCalls := []string{
		"resolve_default_catalog",
		"resolve-database:kb_docs",
		"database:kb_docs",
		"raw_document:Knowledge base document raw source files",
		"processed:Knowledge base processed files",
	}
	if !sameStringSlice(dataDomainSvc.calls, wantCalls) {
		t.Fatalf("data domain calls = %+v, want %+v", dataDomainSvc.calls, wantCalls)
	}
	if len(workflowSvc.deploys) != 1 || len(workflowTemplateSvc.calls) != 1 {
		t.Fatalf("workflow deploys=%+v template calls=%+v", workflowSvc.deploys, workflowTemplateSvc.calls)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceCreateWithSourcesRollsBackWhenSemanticUpdateFails(t *testing.T) {
	var deleteSMCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case isKnowledgeBaseEmbeddingModelsRequest(r):
			writeKnowledgeBaseEmbeddingModelsResponse(t, w, 42)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models":
			requireSemanticModelExecutionHeaders(t, r)
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 77, "name": "kb_docs", "description": "docs", "tables": []any{}, "files": map[string]any{"file_ids": []string{}}})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/ws-1/volumes/12/files/detail":
			t.Fatalf("unexpected raw-volume detail lookup during create-with-sources short request")
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/workspaces/ws-1/volumes/12/files":
			t.Fatalf("unexpected AddFiles during create-with-sources short request")
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/workspaces/ws-1/volumes/12/files/trigger":
			t.Fatalf("unexpected TriggerFiles during create-with-sources short request")
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			requireSemanticModelExecutionHeaders(t, r)
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]any{"code": 13, "message": "semantic update failed"})
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			deleteSMCalls++
			_ = json.NewEncoder(w).Encode(map[string]any{"deleted": true})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	dataDomainSvc := &fakeSemanticModelDataDomainService{databaseID: 11, volumeIDs: []int64{12, 13}}
	localImportSvc := &fakeSemanticModelLocalFileImportService{}
	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()
	workflowSvc := &fakeSemanticModelWorkflowService{}
	svc := newSemanticModelTestServiceWithKnowledgeBaseRuntimeDependencies(t, server.URL, systemClient, dataDomainSvc, nil, localImportSvc, &fakeSemanticModelWorkflowTemplateService{}, workflowSvc)

	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	tenantMock.ExpectExec("INSERT INTO knowledge_base_data_domains").
		WillReturnResult(sqlmock.NewResult(1, 1))
	expectKnowledgeBaseCreateDomainPrepare(tenantMock, 77, 3, 11, 12, 13)
	tenantMock.ExpectBegin()
	tenantMock.ExpectExec("INSERT INTO knowledge_base_sources").
		WillReturnResult(sqlmock.NewResult(1, 1))
	tenantMock.ExpectExec("INSERT INTO knowledge_base_source_job_runs").
		WillReturnResult(sqlmock.NewResult(1, 1))
	tenantMock.ExpectExec("INSERT INTO knowledge_base_source_job_runs").
		WillReturnResult(sqlmock.NewResult(1, 1))
	tenantMock.ExpectCommit()
	tenantMock.ExpectExec("UPDATE knowledge_base_sources").
		WillReturnResult(sqlmock.NewResult(1, 1))
	tenantMock.ExpectExec("UPDATE knowledge_base_source_job_runs").
		WillReturnResult(sqlmock.NewResult(1, 1))
	tenantMock.ExpectExec("UPDATE knowledge_base_source_job_runs").
		WillReturnResult(sqlmock.NewResult(1, 1))
	expectRollbackFailedKnowledgeBaseCreate(tenantMock, 77, 3, 11, 12, 13)

	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithUserID(ctx, "user-1")
	ctx = withKnowledgeBaseCreatePrincipal(ctx)
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)

	_, err = svc.CreateModelWithSources(ctx, CreateSemanticModelWithSourcesRequest{
		Name:        "kb_docs",
		Description: "docs",
		Sources:     []CreateSemanticModelSourceRequest{{SourceType: kbSourceTypeLocalFile, FileName: "local.txt", FileID: "kb-local-file"}},
	})
	if err == nil || !strings.Contains(err.Error(), "update semantic model sources") {
		t.Fatalf("CreateModelWithSources error = %v", err)
	}
	if len(workflowSvc.deploys) != 1 {
		t.Fatalf("workflow deploys = %+v, want 1", workflowSvc.deploys)
	}
	// create-with-sources deployed a workflow; rollback via deleteModel must delete it.
	wantWorkflowID := knowledgeBaseWorkflowID("ws-1", 77)
	if len(workflowSvc.deletes) != 3 {
		t.Fatalf("workflow deletes = %+v, want 3 lifecycle deletes after deployed workflow", workflowSvc.deletes)
	}
	foundDocWorkflow := false
	for _, id := range workflowSvc.deletes {
		if id == wantWorkflowID {
			foundDocWorkflow = true
			break
		}
	}
	if !foundDocWorkflow {
		t.Fatalf("workflow deletes = %+v, want document workflow %q", workflowSvc.deletes, wantWorkflowID)
	}
	if deleteSMCalls != 1 {
		t.Fatalf("semantic model delete calls = %d, want 1 after create rollback", deleteSMCalls)
	}
	if !strings.Contains(strings.Join(dataDomainSvc.calls, ","), "delete-volume:12") ||
		!strings.Contains(strings.Join(dataDomainSvc.calls, ","), "delete-volume:13") ||
		!strings.Contains(strings.Join(dataDomainSvc.calls, ","), "delete-database:11") {
		t.Fatalf("data domain calls = %+v, want catalog resource rollback", dataDomainSvc.calls)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceCreateWithSourcesDefersLocalFileSideEffects(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case isKnowledgeBaseEmbeddingModelsRequest(r):
			writeKnowledgeBaseEmbeddingModelsResponse(t, w, 42)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models":
			requireSemanticModelExecutionHeaders(t, r)
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 77, "name": "kb_docs", "description": "docs", "tables": []any{}, "files": map[string]any{"file_ids": []string{}}})
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			requireSemanticModelExecutionHeaders(t, r)
			_ = json.NewEncoder(w).Encode(map[string]any{"updated": true})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/ws-1/volumes/12/files/detail":
			t.Fatalf("unexpected raw-volume detail lookup during create-with-sources short request")
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/workspaces/ws-1/volumes/12/files":
			t.Fatalf("unexpected AddFiles during create-with-sources short request")
		default:
			t.Fatalf("unexpected request after side effect failure: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	dataDomainSvc := &fakeSemanticModelDataDomainService{databaseID: 11, volumeIDs: []int64{12, 13}}
	localImportSvc := &fakeSemanticModelLocalFileImportService{taskID: "import-task-local", fileIDs: []string{"kb-local-file"}}
	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()
	workflowSvc := &fakeSemanticModelWorkflowService{}
	svc := newSemanticModelTestServiceWithKnowledgeBaseRuntimeDependencies(t, server.URL, systemClient, dataDomainSvc, nil, localImportSvc, &fakeSemanticModelWorkflowTemplateService{}, workflowSvc)

	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	tenantMock.ExpectExec("INSERT INTO knowledge_base_data_domains").
		WillReturnResult(sqlmock.NewResult(1, 1))
	expectKnowledgeBaseCreateDomainPrepare(tenantMock, 77, 3, 11, 12, 13)
	tenantMock.ExpectBegin()
	tenantMock.ExpectExec("INSERT INTO knowledge_base_sources").
		WillReturnResult(sqlmock.NewResult(1, 1))
	tenantMock.ExpectExec("INSERT INTO knowledge_base_source_job_runs").
		WillReturnResult(sqlmock.NewResult(1, 1))
	tenantMock.ExpectExec("INSERT INTO knowledge_base_source_job_runs").
		WillReturnResult(sqlmock.NewResult(1, 1))
	tenantMock.ExpectCommit()
	expectKnowledgeBaseCreateDomainFinalize(tenantMock, 77, 3, 11, 12, 13)

	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithUserID(ctx, "user-1")
	ctx = withKnowledgeBaseCreatePrincipal(ctx)
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)

	resp, err := svc.CreateModelWithSources(ctx, CreateSemanticModelWithSourcesRequest{
		Name:        "kb_docs",
		Description: "docs",
		Sources:     []CreateSemanticModelSourceRequest{{SourceType: kbSourceTypeLocalFile, FileName: "local.txt", FileID: "kb-local-file"}},
	})
	if err != nil {
		t.Fatalf("CreateModelWithSources: %v", err)
	}
	if len(resp.Jobs) != 2 || resp.Jobs[0].JobType != kbJobTypeLoad || resp.Jobs[0].OperationID != nil || resp.Jobs[0].JobStatus != kbSourceJobQueued {
		t.Fatalf("jobs = %+v, want deferred local file load", resp.Jobs)
	}
	if len(resp.Sources) != 1 || resp.Sources[0].ResourceID != "kb-local-file" || resp.Sources[0].IngestStatus == nil || *resp.Sources[0].IngestStatus != kbSourceStatusPending {
		t.Fatalf("sources = %+v, want enabled pending local file source", resp.Sources)
	}
	if len(workflowSvc.deploys) != 1 ||
		!strings.Contains(workflowSvc.deploys[0].DSLYAML, "moi:document.parse") ||
		!strings.Contains(workflowSvc.deploys[0].DSLYAML, "moi:knowledge.index.build") {
		t.Fatalf("workflow deploys = %+v, want default document RAG workflow", workflowSvc.deploys)
	}
	if len(localImportSvc.calls) != 0 {
		t.Fatalf("local import should not run for unstructured file_id path: %+v", localImportSvc.calls)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceCreateWithSourcesRollsBackSourceIntentsWhenIntentWriteFails(t *testing.T) {
	var deleteSMCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case isKnowledgeBaseEmbeddingModelsRequest(r):
			writeKnowledgeBaseEmbeddingModelsResponse(t, w, 42)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models":
			requireSemanticModelExecutionHeaders(t, r)
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 77, "name": "kb_docs", "description": "docs", "tables": []any{}, "files": map[string]any{"file_ids": []string{}}})
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/workspaces/ws-1/semantic-models/77":
			deleteSMCalls++
			_ = json.NewEncoder(w).Encode(map[string]any{"deleted": true})
		default:
			t.Fatalf("unexpected request after intent failure: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	dataDomainSvc := &fakeSemanticModelDataDomainService{databaseID: 11, volumeIDs: []int64{12, 13}}
	localImportSvc := &fakeSemanticModelLocalFileImportService{}
	workflowSvc := &fakeSemanticModelWorkflowService{}
	systemClient, err := moi.New(server.URL, "system-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer systemClient.Close()
	svc := newSemanticModelTestServiceWithKnowledgeBaseRuntimeDependencies(t, server.URL, systemClient, dataDomainSvc, nil, localImportSvc, &fakeSemanticModelWorkflowTemplateService{}, workflowSvc)

	tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("tenant sqlmock: %v", err)
	}
	defer tenantSQL.Close()
	tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tenant gorm: %v", err)
	}
	tenantMock.ExpectExec("INSERT INTO knowledge_base_data_domains").
		WillReturnResult(sqlmock.NewResult(1, 1))
	expectKnowledgeBaseCreateDomainPrepare(tenantMock, 77, 3, 11, 12, 13)
	tenantMock.ExpectBegin()
	tenantMock.ExpectExec("INSERT INTO knowledge_base_sources").
		WillReturnResult(sqlmock.NewResult(1, 1))
	tenantMock.ExpectExec("INSERT INTO knowledge_base_source_job_runs").
		WillReturnResult(sqlmock.NewResult(1, 1))
	tenantMock.ExpectExec("INSERT INTO knowledge_base_source_job_runs").
		WillReturnResult(sqlmock.NewResult(1, 1))
	tenantMock.ExpectExec("INSERT INTO knowledge_base_sources").
		WillReturnError(errors.New("insert source failed"))
	tenantMock.ExpectRollback()
	expectRollbackFailedKnowledgeBaseCreate(tenantMock, 77, 3, 11, 12, 13)

	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithMoiApiKey(ctx, "caller-key")
	ctx = ctxutil.WithUserID(ctx, "user-1")
	ctx = withKnowledgeBaseCreatePrincipal(ctx)
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)

	_, err = svc.CreateModelWithSources(ctx, CreateSemanticModelWithSourcesRequest{
		Name:        "kb_docs",
		Description: "docs",
		Sources: []CreateSemanticModelSourceRequest{
			{SourceType: kbSourceTypeLocalFile, FileName: "local.txt", FileID: "kb-local-file"},
			{SourceType: kbSourceTypeLocalFile, FileName: "other.txt", FileID: "kb-other-file"},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "record local file source") {
		t.Fatalf("CreateModelWithSources error = %v", err)
	}
	if len(workflowSvc.deploys) != 1 {
		t.Fatalf("workflow deploys = %+v, want KB-level ensure before intent failure", workflowSvc.deploys)
	}
	if len(workflowSvc.deletes) != 3 {
		t.Fatalf("workflow deletes = %+v, want lifecycle deletes after deploy-then-fail", workflowSvc.deletes)
	}
	if len(localImportSvc.calls) != 0 {
		t.Fatalf("local import should not run after intent failure: %+v", localImportSvc.calls)
	}
	if deleteSMCalls != 1 {
		t.Fatalf("semantic model delete calls = %d, want 1 after create rollback", deleteSMCalls)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestCreateKnowledgeBaseSourceMetadataIntentsDefersOnlyFileJobsAndLinksTables(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/workspaces/ws-1/tables/1001" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"table":    map[string]any{"id": 1001, "catalog_id": 3, "database_id": 11, "name": "orders"},
			"database": map[string]any{"id": 11, "name": "sales"},
			"catalog":  map[string]any{"id": 3, "name": "catalog"},
		})
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

	expectCatalogFileSourceOriginLookupMiss(tenantMock, "source-file")
	tenantMock.ExpectExec("INSERT INTO knowledge_base_sources").
		WillReturnResult(sqlmock.NewResult(1, 1))
	tenantMock.ExpectExec("INSERT INTO knowledge_base_source_job_runs").
		WillReturnResult(sqlmock.NewResult(1, 1))
	tenantMock.ExpectExec("INSERT INTO knowledge_base_source_job_runs").
		WillReturnResult(sqlmock.NewResult(2, 1))
	tenantMock.ExpectExec("INSERT INTO knowledge_base_sources").
		WillReturnResult(sqlmock.NewResult(3, 1))

	ctx := ctxutil.WithTenantDB(context.Background(), tenantDB)
	ctx = ctxutil.WithMoiUserID(ctx, "moi-user-create")
	ctx = ctxutil.WithCoreIAMRequest(ctx, ctxutil.CoreIAMRequestContext{VerifiedEffectiveRoleID: "role-create"})
	svc := &semanticModelService{}
	result, err := svc.createKnowledgeBaseSourceMetadataIntentsInTx(ctx, client, "ws-1", 77, &KnowledgeBaseDataDomain{
		ModelID:           77,
		CatalogID:         3,
		DatabaseID:        11,
		RawVolumeID:       12,
		ProcessedVolumeID: 13,
	}, []CreateSemanticModelSourceRequest{
		{SourceType: kbSourceTypeCatalogFile, FileID: "source-file", FileName: "source.pdf", VolumeID: 41},
		{SourceType: kbSourceTypeCatalogTable, TableID: 1001},
	}, "user-1", false)
	if err != nil {
		t.Fatalf("createKnowledgeBaseSourceMetadataIntentsInTx: %v", err)
	}
	if len(result.jobs) != 2 {
		t.Fatalf("jobs = %+v, want 2 file jobs", result.jobs)
	}
	statusByType := make(map[string]string, len(result.jobs))
	for _, job := range result.jobs {
		statusByType[job.JobType] = job.JobStatus
	}
	if statusByType[kbJobTypeCopy] != kbSourceJobPending {
		t.Fatalf("copy job status = %q, want pending", statusByType[kbJobTypeCopy])
	}
	if statusByType[kbJobTypeRAGIngest] != kbSourceJobPending {
		t.Fatalf("rag job status = %q, want pending", statusByType[kbJobTypeRAGIngest])
	}
	if len(result.tables) != 1 || result.tables[0].DBName != "sales" || !sameStringSet(result.tables[0].TableNames, []string{"orders"}) {
		t.Fatalf("tables = %+v, want direct catalog table", result.tables)
	}
	if err := tenantMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("tenant sql expectations: %v", err)
	}
}

func TestSemanticModelServiceCreateWithSourcesRequiresDataDomainService(t *testing.T) {
	svc := newSemanticModelTestServiceWithKnowledgeBaseDependencies(t, "", nil, nil, nil, &fakeSemanticModelLocalFileImportService{})
	_, err := svc.CreateModelWithSources(context.Background(), CreateSemanticModelWithSourcesRequest{
		Name:    "kb_docs",
		Sources: []CreateSemanticModelSourceRequest{{SourceType: kbSourceTypeLocalFile, FileName: "a.txt", FileID: "file-a"}},
	})
	if err == nil || !strings.Contains(err.Error(), "catalog data-domain service is not configured") {
		t.Fatalf("error = %v, want data-domain service error", err)
	}
}

func TestPrepareKnowledgeBaseDataDomainResourcesForCreateRejectsClaimMiss(t *testing.T) {
	// A create request never joins another provision owner or winner.
	db, mock := newSemanticModelTenantDB(t)
	ctx := ctxutil.WithTenantDB(context.Background(), db)
	dataDomainSvc := &fakeSemanticModelDataDomainService{databaseID: 11, volumeIDs: []int64{12, 13}}
	svc := &semanticModelService{dataDomainService: dataDomainSvc}

	domain := &KnowledgeBaseDataDomain{ModelID: 77, CatalogID: 3, EnsureStatus: kbEnsureStatusFailed}
	mock.ExpectExec("UPDATE knowledge_base_data_domains\\s+SET ensure_status = \\?, last_ensure_error = NULL, last_checked_at = \\?, updated_by = \\?\\s+WHERE model_id = \\? AND ensure_status = \\?").
		WithArgs(kbEnsureStatusProvisioning, sqlmock.AnyArg(), "user-1", int64(77), kbEnsureStatusFailed).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := svc.prepareKnowledgeBaseDataDomainResourcesForCreate(ctx, domain, "kb_docs", "docs", "user-1")
	if err == nil || !strings.Contains(err.Error(), "being provisioned") {
		t.Fatalf("prepare error = %v, want in-progress", err)
	}
	if len(dataDomainSvc.calls) != 0 {
		t.Fatalf("data domain calls = %+v, want none", dataDomainSvc.calls)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestReconcileKnowledgeBaseDataDomainLoserSeesProvisioning(t *testing.T) {
	// Concurrent loser: claim miss + re-read provisioning → in-progress, no resource create.
	db, mock := newSemanticModelTenantDB(t)
	ctx := ctxutil.WithTenantDB(context.Background(), db)
	dataDomainSvc := &fakeSemanticModelDataDomainService{databaseID: 99, volumeIDs: []int64{98, 97}}
	svc := &semanticModelService{dataDomainService: dataDomainSvc}

	domain := &KnowledgeBaseDataDomain{ModelID: 77, CatalogID: 3, EnsureStatus: kbEnsureStatusFailed}
	mock.ExpectExec("UPDATE knowledge_base_data_domains\\s+SET ensure_status = \\?, last_ensure_error = NULL, last_checked_at = \\?, updated_by = \\?\\s+WHERE model_id = \\? AND ensure_status = \\?").
		WithArgs(kbEnsureStatusProvisioning, sqlmock.AnyArg(), "loser", int64(77), kbEnsureStatusFailed).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("SELECT model_id, catalog_id, database_id, raw_volume_id, processed_volume_id, ensure_status, last_ensure_error, last_checked_at").
		WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows([]string{
			"model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "ensure_status", "last_ensure_error", "last_checked_at",
		}).AddRow(int64(77), int64(3), int64(0), int64(0), int64(0), kbEnsureStatusProvisioning, nil, int64(100)))

	_, err := svc.reconcileKnowledgeBaseDataDomain(ctx, domain, true, 77, "kb_docs", "docs", "loser")
	if err == nil || !strings.Contains(err.Error(), "being provisioned") {
		t.Fatalf("error = %v, want in-progress", err)
	}
	if len(dataDomainSvc.calls) != 0 {
		t.Fatalf("data domain calls = %+v, want none", dataDomainSvc.calls)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestReconcileKnowledgeBaseDataDomainInsertConflictReloadsProvisioning(t *testing.T) {
	// Both first appends observed no row. The loser INSERT conflicts after the
	// winner claimed provisioning, so it must re-read and create no resources.
	db, mock := newSemanticModelTenantDB(t)
	ctx := ctxutil.WithTenantDB(context.Background(), db)
	dataDomainSvc := &fakeSemanticModelDataDomainService{}
	svc := &semanticModelService{dataDomainService: dataDomainSvc}

	mock.ExpectExec("INSERT INTO knowledge_base_data_domains").
		WillReturnError(&mysqlDriver.MySQLError{Number: 1062, Message: "Duplicate entry '77' for key 'PRIMARY'"})
	mock.ExpectQuery("SELECT model_id, catalog_id, database_id, raw_volume_id, processed_volume_id, ensure_status, last_ensure_error, last_checked_at").
		WithArgs(int64(77)).
		WillReturnRows(sqlmock.NewRows([]string{
			"model_id", "catalog_id", "database_id", "raw_volume_id", "processed_volume_id", "ensure_status", "last_ensure_error", "last_checked_at",
		}).AddRow(int64(77), int64(3), int64(0), int64(0), int64(0), kbEnsureStatusProvisioning, nil, int64(100)))

	_, err := svc.reconcileKnowledgeBaseDataDomain(ctx, nil, false, 77, "kb_docs", "docs", "loser")
	if err == nil || !strings.Contains(err.Error(), "being provisioned") {
		t.Fatalf("error = %v, want in-progress", err)
	}
	if !sameStringSlice(dataDomainSvc.calls, []string{"resolve_default_catalog"}) {
		t.Fatalf("data domain calls = %+v, want catalog lookup only", dataDomainSvc.calls)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestDeleteOrphanSemanticModelIgnoresCanceledParentContext(t *testing.T) {
	// MF-2: parent cancel/timeout must not prevent compensatory SM delete.
	const modelID = int64(77)
	var deleteCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodDelete || r.URL.Path != "/api/v1/workspaces/ws-1/semantic-models/77" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		deleteCalls++
		if err := r.Context().Err(); err != nil {
			t.Errorf("delete request context already failed: %v", err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"deleted": true})
	}))
	defer server.Close()

	client, err := moi.New(server.URL, "caller-key")
	if err != nil {
		t.Fatalf("moi.New: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // parent already canceled — without WithoutCancel, Delete would fail immediately
	if err := ctx.Err(); !errors.Is(err, context.Canceled) {
		t.Fatalf("parent ctx err = %v, want canceled", err)
	}
	if err := deleteOrphanSemanticModel(ctx, client, "ws-1", modelID); err != nil {
		t.Fatalf("deleteOrphanSemanticModel: %v", err)
	}
	if deleteCalls != 1 {
		t.Fatalf("delete calls = %d, want 1", deleteCalls)
	}
}

func TestClaimKnowledgeBaseDataDomainProvisionIsCAS(t *testing.T) {
	// MF-3: only one claimer moves failed → provisioning.
	db, mock := newSemanticModelTenantDB(t)
	ctx := ctxutil.WithTenantDB(context.Background(), db)
	svc := &semanticModelService{}

	mock.ExpectExec("UPDATE knowledge_base_data_domains\\s+SET ensure_status = \\?, last_ensure_error = NULL, last_checked_at = \\?, updated_by = \\?\\s+WHERE model_id = \\? AND ensure_status = \\?").
		WithArgs(kbEnsureStatusProvisioning, sqlmock.AnyArg(), "user-1", int64(77), kbEnsureStatusFailed).
		WillReturnResult(sqlmock.NewResult(0, 1))
	ok, err := svc.claimKnowledgeBaseDataDomainProvision(ctx, 77, "user-1")
	if err != nil || !ok {
		t.Fatalf("first claim = (%v, %v), want true,nil", ok, err)
	}

	mock.ExpectExec("UPDATE knowledge_base_data_domains\\s+SET ensure_status = \\?, last_ensure_error = NULL, last_checked_at = \\?, updated_by = \\?\\s+WHERE model_id = \\? AND ensure_status = \\?").
		WithArgs(kbEnsureStatusProvisioning, sqlmock.AnyArg(), "user-2", int64(77), kbEnsureStatusFailed).
		WillReturnResult(sqlmock.NewResult(0, 0))
	ok, err = svc.claimKnowledgeBaseDataDomainProvision(ctx, 77, "user-2")
	if err != nil || ok {
		t.Fatalf("second claim = (%v, %v), want false,nil", ok, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}
