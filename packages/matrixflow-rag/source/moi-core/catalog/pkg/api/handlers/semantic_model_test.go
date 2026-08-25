package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/matrixflow/moi-core/catalog/pkg/agentresource"
	ginctx "github.com/matrixflow/moi-core/catalog/pkg/api"
	"github.com/matrixflow/moi-core/catalog/pkg/iamcore"
	servicebase "github.com/matrixflow/moi-core/catalog/pkg/service"
	semanticservice "github.com/matrixflow/moi-core/catalog/pkg/service/semantic"
	"github.com/matrixflow/moi-core/catalog/pkg/service/storage/tenant"
	"github.com/matrixflow/moi-core/catalog/pkg/service/storage/transaction"
	"github.com/matrixflow/moi-core/model/auth"
	catalogpb "github.com/matrixflow/moi-core/model/catalog"
	"github.com/matrixflow/moi-core/model/common"
	authzcore "github.com/matrixorigin/matrixflow/shared/authz/pkg/core"
)

type mockSemanticConnectionPool struct {
	err error
}

func (p *mockSemanticConnectionPool) GetConnection(_ context.Context, _ string) (*sql.DB, error) {
	return nil, p.err
}

func (p *mockSemanticConnectionPool) GetDBExecutor(_ context.Context, _ string) (tenant.DBExecutor, error) {
	return nil, p.err
}

func (p *mockSemanticConnectionPool) GetTransactionManager(_ context.Context, _ string) (*transaction.Manager, error) {
	if p.err != nil {
		return nil, p.err
	}
	db, dbMock, _ := sqlmock.New()
	dbMock.ExpectBegin()
	dbMock.ExpectCommit()
	return transaction.NewManager(db), nil
}

func (p *mockSemanticConnectionPool) GetTx(_ context.Context, _ string) (*sql.Tx, error) {
	return nil, p.err
}

func (p *mockSemanticConnectionPool) Close() error {
	return nil
}

type mockSemanticConnectionPoolWithRollback struct {
	err  error
	mock sqlmock.Sqlmock
}

func (p *mockSemanticConnectionPoolWithRollback) GetConnection(_ context.Context, _ string) (*sql.DB, error) {
	return nil, p.err
}

func (p *mockSemanticConnectionPoolWithRollback) GetDBExecutor(_ context.Context, _ string) (tenant.DBExecutor, error) {
	return nil, p.err
}

func (p *mockSemanticConnectionPoolWithRollback) GetTransactionManager(_ context.Context, _ string) (*transaction.Manager, error) {
	if p.err != nil {
		return nil, p.err
	}
	db, dbMock, _ := sqlmock.New()
	dbMock.ExpectBegin()
	dbMock.ExpectRollback()
	p.mock = dbMock
	return transaction.NewManager(db), nil
}

func (p *mockSemanticConnectionPoolWithRollback) GetTx(_ context.Context, _ string) (*sql.Tx, error) {
	return nil, p.err
}

func (p *mockSemanticConnectionPoolWithRollback) Close() error {
	return nil
}

type mockSemanticModelStorage struct {
	mock.Mock
}

type semanticModelIAMRegistrarStub struct{}

func (semanticModelIAMRegistrarStub) Register(_ context.Context, req iamcore.SemanticModelResourceRegisterRequest) (*tenant.IAMResourceOwnershipResult, error) {
	return &tenant.IAMResourceOwnershipResult{WorkspaceID: req.WorkspaceID, ResourceType: iamcore.IAMResourceSemanticModel, ResourceID: req.SemanticModelID, OwnerRoleID: "role-1", AuthorizedRoleID: "role-1", Status: "active", OwnershipVersion: 1}, nil
}

func (semanticModelIAMRegistrarStub) RegisterAuthorized(_ context.Context, req iamcore.SemanticModelResourceRegisterRequest, roleID string) (*tenant.IAMResourceOwnershipResult, error) {
	return &tenant.IAMResourceOwnershipResult{WorkspaceID: req.WorkspaceID, ResourceType: iamcore.IAMResourceSemanticModel, ResourceID: req.SemanticModelID, OwnerRoleID: roleID, AuthorizedRoleID: roleID, Status: "active", OwnershipVersion: 1}, nil
}

func (semanticModelIAMRegistrarStub) BeginDelete(_ context.Context, req iamcore.SemanticModelResourceDeleteRequest) (*tenant.IAMResourceOwnershipResult, error) {
	return &tenant.IAMResourceOwnershipResult{WorkspaceID: req.WorkspaceID, ResourceType: iamcore.IAMResourceSemanticModel, ResourceID: req.SemanticModelID, OwnerRoleID: "role-1", AuthorizedRoleID: "role-1", Status: "deleting", OwnershipVersion: 2}, nil
}

func (semanticModelIAMRegistrarStub) FinalizeDelete(context.Context, iamcore.SemanticModelResourceDeleteRequest, int64, string) error {
	return nil
}

func (m *mockSemanticModelStorage) CreateSemanticModel(ctx context.Context, model *tenant.SemanticModelRecord) (*tenant.SemanticModelRecord, error) {
	args := m.Called(ctx, model)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*tenant.SemanticModelRecord), args.Error(1)
}

func (m *mockSemanticModelStorage) GetSemanticModel(ctx context.Context, modelID int64) (*tenant.SemanticModelRecord, error) {
	args := m.Called(ctx, modelID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*tenant.SemanticModelRecord), args.Error(1)
}

func (m *mockSemanticModelStorage) GetSemanticModelForUpdate(ctx context.Context, modelID int64) (*tenant.SemanticModelRecord, error) {
	args := m.Called(ctx, modelID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*tenant.SemanticModelRecord), args.Error(1)
}

func (m *mockSemanticModelStorage) LockSemanticModelsForUpdate(ctx context.Context, modelIDs []int64) ([]*tenant.SemanticModelRecord, error) {
	args := m.Called(ctx, modelIDs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*tenant.SemanticModelRecord), args.Error(1)
}

func (m *mockSemanticModelStorage) ListSemanticModels(ctx context.Context, opts ...tenant.ListOption) ([]*tenant.SemanticModelRecord, int64, error) {
	args := m.Called(ctx, opts)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*tenant.SemanticModelRecord), args.Get(1).(int64), args.Error(2)
}

func (m *mockSemanticModelStorage) ListSemanticModelTags(ctx context.Context, opts ...tenant.ListOption) ([]tenant.SemanticModelTagStat, error) {
	args := m.Called(ctx, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]tenant.SemanticModelTagStat), args.Error(1)
}

func (m *mockSemanticModelStorage) UpdateSemanticModel(ctx context.Context, model *tenant.SemanticModelRecord) error {
	args := m.Called(ctx, model)
	return args.Error(0)
}

func (m *mockSemanticModelStorage) UpsertSystemResourceDisplayMapping(ctx context.Context, mapping tenant.SystemResourceDisplayMapping) error {
	args := m.Called(ctx, mapping)
	return args.Error(0)
}

func (m *mockSemanticModelStorage) GetDatabase(ctx context.Context, databaseID int64) (*catalogpb.Database, error) {
	args := m.Called(ctx, databaseID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*catalogpb.Database), args.Error(1)
}

func (m *mockSemanticModelStorage) DeleteSemanticModel(ctx context.Context, modelID int64) error {
	args := m.Called(ctx, modelID)
	return args.Error(0)
}

func (m *mockSemanticModelStorage) CreateSemanticEntry(ctx context.Context, entry *tenant.SemanticEntryRecord) (*tenant.SemanticEntryRecord, error) {
	args := m.Called(ctx, entry)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*tenant.SemanticEntryRecord), args.Error(1)
}

func (m *mockSemanticModelStorage) GetSemanticEntry(ctx context.Context, modelID, entryID int64) (*tenant.SemanticEntryRecord, error) {
	args := m.Called(ctx, modelID, entryID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*tenant.SemanticEntryRecord), args.Error(1)
}

func (m *mockSemanticModelStorage) ListSemanticEntries(ctx context.Context, modelID int64, kind string, opts ...tenant.ListOption) ([]*tenant.SemanticEntryRecord, int64, error) {
	args := m.Called(ctx, modelID, kind, opts)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*tenant.SemanticEntryRecord), args.Get(1).(int64), args.Error(2)
}

func (m *mockSemanticModelStorage) UpdateSemanticEntry(ctx context.Context, entry *tenant.SemanticEntryRecord) error {
	args := m.Called(ctx, entry)
	return args.Error(0)
}

func (m *mockSemanticModelStorage) DeleteSemanticEntry(ctx context.Context, modelID, entryID int64) error {
	args := m.Called(ctx, modelID, entryID)
	return args.Error(0)
}

func setupSemanticModelTestRouter(handler *SemanticModelHandler) *gin.Engine {
	return setupSemanticModelTestRouterWithBackendExecution(handler, false)
}

func setupSemanticModelTestRouterWithBackendExecution(handler *SemanticModelHandler, delegated bool) *gin.Engine {
	registrar := semanticModelIAMRegistrarStub{}
	transactions, err := servicebase.NewTransactionalService(handler.pool)
	if err != nil {
		panic(err)
	}
	handler.
		WithIAM(&workflowIAMAuthorizerStub{}, &workflowIAMFilterStub{plan: iamcore.IAMResourceAccessFilterPlan{AllResources: true}}, registrar).
		WithCreationService(semanticservice.NewService(transactions, handler.storage, registrar, zap.NewNop()))
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		ginctx.SetAPIKeyInfo(c, &auth.APIKey{Uid: "test-user"})
		if delegated {
			ginctx.SetAuthenticatedBackendExecution(c, ginctx.BackendExecutionContext{
				WorkspaceID:              "ws-123",
				WorkspaceAccessVerified:  true,
				BusinessActionAuthorized: true,
			})
		}
		c.Request.Header.Set("X-Request-ID", "semantic-model-test-request")
		c.Request.Header.Set("X-Trace-ID", "semantic-model-test-trace")
		c.Next()
	})

	group := r.Group("/api/v1/workspaces/:id/semantic-models")
	group.POST("", handler.Create)
	group.GET("", handler.List)
	group.GET("/tags", handler.ListTags)
	group.POST("/import", handler.Import)
	group.GET("/:model_id", handler.Get)
	group.PUT("/:model_id", handler.Update)
	group.DELETE("/:model_id", handler.Delete)
	group.GET("/:model_id/export", handler.Export)
	group.POST("/:model_id/validate", handler.Validate)
	group.POST("/:model_id/entries", handler.CreateEntry)
	group.GET("/:model_id/entries", handler.ListEntries)
	group.PUT("/:model_id/entries/:entry_id", handler.UpdateEntry)
	group.DELETE("/:model_id/entries/:entry_id", handler.DeleteEntry)
	return r
}

func setupSemanticModelTestRouterNoAuth(handler *SemanticModelHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	group := r.Group("/api/v1/workspaces/:id/semantic-models")
	group.POST("", handler.Create)
	return r
}

// structuredTables returns a JSON string for the structured tables format.
// e.g. structuredTables("orders", "users") → `[{"db_name":"","table_names":["orders","users"],"parents":[]}]`
func structuredTables(names ...string) string {
	if len(names) == 0 {
		return `[]`
	}
	quoted := make([]string, len(names))
	for i, n := range names {
		quoted[i] = `"` + n + `"`
	}
	return `[{"db_name":"","table_names":[` + strings.Join(quoted, ",") + `],"parents":[]}]`
}

func structuredTablesJSON(names ...string) json.RawMessage {
	return json.RawMessage(structuredTables(names...))
}

func TestExtractTableNamesFromJSONPreservesFileStyleTableNames(t *testing.T) {
	raw := json.RawMessage(`[
		{"db_name":"232323_10006","table_names":["dimproductsubcategory.csv","test.csv"],"parents":[]}
	]`)

	got, err := extractTableNamesFromJSON(raw)
	require.NoError(t, err)
	require.Equal(t, []string{"dimproductsubcategory.csv", "test.csv"}, got)
}

func TestExtractTableNamesFromJSONRejectsDBQualifiedDuplicate(t *testing.T) {
	raw := json.RawMessage(`[
		{"db_name":"sales","table_names":["orders","sales.orders"],"parents":[]}
	]`)

	_, err := extractTableNamesFromJSON(raw)
	require.Error(t, err)
	require.Contains(t, err.Error(), `duplicate table "orders"`)
}

func TestExtractTableNamesFromJSONRejectsQuotedDuplicate(t *testing.T) {
	raw := json.RawMessage("[" +
		`{"db_name":"sales","table_names":["orders","` + "`orders`" + `"],"parents":[]}` +
		"]")

	_, err := extractTableNamesFromJSON(raw)
	require.Error(t, err)
	require.Contains(t, err.Error(), `duplicate table "orders"`)
}

func TestSemanticModelHandler_Create_Success(t *testing.T) {
	storage := new(mockSemanticModelStorage)
	pool := &mockSemanticConnectionPool{}
	handler := NewSemanticModelHandler(pool, storage, zap.NewNop())
	router := setupSemanticModelTestRouter(handler)

	storage.On("CreateSemanticModel", mock.Anything, mock.MatchedBy(func(model *tenant.SemanticModelRecord) bool {
		if model == nil {
			return false
		}
		return model.Name == "SalesModel" &&
			model.CreatedBy == "test-user" &&
			model.UpdatedBy == "test-user" &&
			len(model.Tables) > 0 &&
			model.TableSetHash != ""
	})).Return(&tenant.SemanticModelRecord{
		ID:           101,
		Name:         "SalesModel",
		Description:  "sales semantic model",
		Tables:       json.RawMessage(`["orders","users"]`),
		TableSetHash: "mock-hash",
		CreatedBy:    "test-user",
		UpdatedBy:    "test-user",
	}, nil).Once()

	body := `{"name":"SalesModel","description":"sales semantic model","tables":` + structuredTables("Users", "orders") + `}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/ws-123/semantic-models", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var resp semanticModelResponse
	unmarshalAPIData(t, w.Body.Bytes(), &resp)
	assert.Equal(t, int64(101), resp.ID)
	assert.JSONEq(t, `["orders","users"]`, string(resp.Tables))
	storage.AssertExpectations(t)
}

func TestSemanticModelHandler_UpdateEnsuresKnowledgeBaseDatabaseDisplayNameInTransaction(t *testing.T) {
	storage := new(mockSemanticModelStorage)
	pool := &mockSemanticConnectionPool{}
	handler := NewSemanticModelHandler(pool, storage, zap.NewNop())
	router := setupSemanticModelTestRouterWithBackendExecution(handler, true)

	storage.On("GetSemanticModelForUpdate", mock.Anything, int64(101)).
		Return(&tenant.SemanticModelRecord{ID: 101, Name: "SalesModel"}, nil).
		Once()
	storage.On("UpdateSemanticModel", mock.Anything, mock.MatchedBy(func(model *tenant.SemanticModelRecord) bool {
		if model == nil {
			return false
		}
		return model.ID == 101 &&
			model.Name == "SalesModelRenamed" &&
			model.Description == "renamed model" &&
			model.UpdatedBy == "test-user" &&
			model.TableSetHash != ""
	})).Return(nil).Once()
	storage.On("GetDatabase", mock.Anything, int64(11)).
		Return(&catalogpb.Database{Id: 11}, nil).
		Once()
	storage.On("UpsertSystemResourceDisplayMapping", mock.Anything, mock.MatchedBy(func(mapping tenant.SystemResourceDisplayMapping) bool {
		return mapping.ResourceType == "database" &&
			mapping.ResourceID == "11" &&
			mapping.Field == "name" &&
			mapping.DisplayOwner == "moi_backend" &&
			mapping.DisplayKey == "moi_backend.resource.literal_default_text" &&
			mapping.DefaultText == "SalesModelRenamed"
	})).Return(nil).Once()

	body := `{"name":"SalesModelRenamed","description":"renamed model","tables":` + structuredTables("orders") + `,"knowledge_base_database_display_name":{"database_id":11,"display_name":"SalesModelRenamed"}}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/workspaces/ws-123/semantic-models/101", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	storage.AssertExpectations(t)
}

func TestSemanticModelHandler_UpdateRejectsKnowledgeBaseDatabaseDisplayNameWithoutBackendExecution(t *testing.T) {
	storage := new(mockSemanticModelStorage)
	pool := &mockSemanticConnectionPool{}
	handler := NewSemanticModelHandler(pool, storage, zap.NewNop())
	router := setupSemanticModelTestRouter(handler)

	body := `{"name":"SalesModelRenamed","description":"renamed model","tables":` + structuredTables("orders") + `,"knowledge_base_database_display_name":{"database_id":11,"display_name":"SalesModelRenamed"}}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/workspaces/ws-123/semantic-models/101", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	storage.AssertNotCalled(t, "UpdateSemanticModel", mock.Anything, mock.Anything)
	storage.AssertNotCalled(t, "UpsertSystemResourceDisplayMapping", mock.Anything, mock.Anything)
}

func TestSemanticModelHandler_UpdateWithBackendExecutionStillRequiresUserAuthorization(t *testing.T) {
	storage := new(mockSemanticModelStorage)
	pool := &mockSemanticConnectionPool{}
	handler := NewSemanticModelHandler(pool, storage, zap.NewNop())
	router := setupSemanticModelTestRouterWithBackendExecution(handler, true)
	authorizer := &workflowIAMAuthorizerStub{err: authzcore.ErrDenied}
	handler.WithIAM(authorizer, &workflowIAMFilterStub{plan: iamcore.IAMResourceAccessFilterPlan{AllResources: true}}, semanticModelIAMRegistrarStub{})

	body := `{"name":"SalesModelRenamed","description":"renamed model","tables":` + structuredTables("orders") + `,"knowledge_base_database_display_name":{"database_id":11,"display_name":"SalesModelRenamed"}}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/workspaces/ws-123/semantic-models/101", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Equal(t, 1, authorizer.calls)
	assert.Equal(t, iamcore.IAMActionSemanticModelUpdate, authorizer.actionID)
	storage.AssertNotCalled(t, "UpdateSemanticModel", mock.Anything, mock.Anything)
	storage.AssertNotCalled(t, "UpsertSystemResourceDisplayMapping", mock.Anything, mock.Anything)
}

func TestSemanticModelHandler_UpdateRollsBackWhenKnowledgeBaseDatabaseDisplayNameFails(t *testing.T) {
	storage := new(mockSemanticModelStorage)
	pool := &mockSemanticConnectionPoolWithRollback{}
	handler := NewSemanticModelHandler(pool, storage, zap.NewNop())
	router := setupSemanticModelTestRouterWithBackendExecution(handler, true)
	var semanticModelUpdateCtx context.Context
	var displayMappingUpdateCtx context.Context

	storage.On("GetSemanticModelForUpdate", mock.Anything, int64(101)).
		Return(&tenant.SemanticModelRecord{ID: 101, Name: "SalesModel"}, nil).
		Once()
	storage.On("UpdateSemanticModel", mock.Anything, mock.AnythingOfType("*tenant.SemanticModelRecord")).
		Run(func(args mock.Arguments) {
			semanticModelUpdateCtx = args.Get(0).(context.Context)
		}).
		Return(nil).
		Once()
	storage.On("GetDatabase", mock.Anything, int64(11)).
		Return(&catalogpb.Database{Id: 11}, nil).
		Once()
	storage.On("UpsertSystemResourceDisplayMapping", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			displayMappingUpdateCtx = args.Get(0).(context.Context)
		}).
		Return(errors.New("display mapping write failed")).
		Once()

	body := `{"name":"SalesModelRenamed","description":"renamed model","tables":` + structuredTables("orders") + `,"knowledge_base_database_display_name":{"database_id":11,"display_name":"SalesModelRenamed"}}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/workspaces/ws-123/semantic-models/101", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	storage.AssertExpectations(t)
	semanticModelTM := tenant.GetTransactionManagerFromContext(semanticModelUpdateCtx)
	displayMappingTM := tenant.GetTransactionManagerFromContext(displayMappingUpdateCtx)
	require.NotNil(t, semanticModelTM)
	require.Same(t, semanticModelTM, displayMappingTM)
	semanticModelTx, ok := semanticModelTM.GetExecutor(semanticModelUpdateCtx).(*sql.Tx)
	require.True(t, ok)
	displayMappingTx, ok := displayMappingTM.GetExecutor(displayMappingUpdateCtx).(*sql.Tx)
	require.True(t, ok)
	require.Same(t, semanticModelTx, displayMappingTx)
	require.NotNil(t, pool.mock)
	require.NoError(t, pool.mock.ExpectationsWereMet())
}

func TestSemanticModelHandler_Create_Duplicate(t *testing.T) {
	storage := new(mockSemanticModelStorage)
	pool := &mockSemanticConnectionPoolWithRollback{}
	handler := NewSemanticModelHandler(pool, storage, zap.NewNop())
	router := setupSemanticModelTestRouter(handler)

	storage.On("CreateSemanticModel", mock.Anything, mock.AnythingOfType("*tenant.SemanticModelRecord")).
		Return(nil, tenant.ErrSemanticModelAlreadyExist).
		Once()

	body := `{"name":"SalesModel","tables":` + structuredTables("orders") + `}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/ws-123/semantic-models", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	storage.AssertExpectations(t)
}

func TestSemanticModelHandler_Create_PoolError(t *testing.T) {
	storage := new(mockSemanticModelStorage)
	pool := &mockSemanticConnectionPool{err: errors.New("connection failed")}
	handler := NewSemanticModelHandler(pool, storage, zap.NewNop())
	router := setupSemanticModelTestRouter(handler)

	body := `{"name":"SalesModel","tables":` + structuredTables("orders") + `}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/ws-123/semantic-models", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestSemanticModelHandler_Create_Unauthenticated(t *testing.T) {
	storage := new(mockSemanticModelStorage)
	pool := &mockSemanticConnectionPool{}
	handler := NewSemanticModelHandler(pool, storage, zap.NewNop())
	router := setupSemanticModelTestRouterNoAuth(handler)

	body := `{"name":"SalesModel","tables":` + structuredTables("orders") + `}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/ws-123/semantic-models", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestSemanticModelHandler_Import_ValidationFailure(t *testing.T) {
	storage := new(mockSemanticModelStorage)
	pool := &mockSemanticConnectionPool{}
	handler := NewSemanticModelHandler(pool, storage, zap.NewNop())
	router := setupSemanticModelTestRouter(handler)

	body := `{
		"name":"SalesModel",
		"tables":` + structuredTables("orders") + `,
		"entries":[
			{"kind":"metric","key":"gmv","tables":["orders"],"spec":{"expr":"sum(amount)","requires_join":"rel_orders_users"}}
		]
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/ws-123/semantic-models/import", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSemanticModelHandler_Import_BatchSuccess(t *testing.T) {
	storage := new(mockSemanticModelStorage)
	pool := &mockSemanticConnectionPool{}
	handler := NewSemanticModelHandler(pool, storage, zap.NewNop())
	router := setupSemanticModelTestRouter(handler)

	storage.On("CreateSemanticModel", mock.Anything, mock.MatchedBy(func(model *tenant.SemanticModelRecord) bool {
		return model != nil && model.Name == "SalesModelA"
	})).Return(&tenant.SemanticModelRecord{
		ID:           101,
		Name:         "SalesModelA",
		Tables:       structuredTablesJSON("orders"),
		TableSetHash: "hash-a",
	}, nil).Once()
	storage.On("CreateSemanticModel", mock.Anything, mock.MatchedBy(func(model *tenant.SemanticModelRecord) bool {
		return model != nil && model.Name == "SalesModelB"
	})).Return(&tenant.SemanticModelRecord{
		ID:           102,
		Name:         "SalesModelB",
		Tables:       structuredTablesJSON("users"),
		TableSetHash: "hash-b",
	}, nil).Once()

	body := `[
		{"name":"SalesModelA","tables":` + structuredTables("orders") + `},
		{"name":"SalesModelB","tables":` + structuredTables("users") + `}
	]`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/ws-123/semantic-models/import", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var resp semanticModelImportBatchResponse
	unmarshalAPIData(t, w.Body.Bytes(), &resp)
	require.Len(t, resp.Items, 2)
	assert.Equal(t, int64(2), resp.Total)
	assert.Equal(t, "SalesModelA", resp.Items[0].Name)
	assert.Equal(t, "SalesModelB", resp.Items[1].Name)
	storage.AssertExpectations(t)
}

func TestSemanticModelHandler_Import_SkipsDisabledLegacyEntries(t *testing.T) {
	storage := new(mockSemanticModelStorage)
	pool := &mockSemanticConnectionPool{}
	handler := NewSemanticModelHandler(pool, storage, zap.NewNop())
	router := setupSemanticModelTestRouter(handler)

	storage.On("CreateSemanticModel", mock.Anything, mock.MatchedBy(func(model *tenant.SemanticModelRecord) bool {
		return model != nil && model.Name == "SalesModel"
	})).Return(&tenant.SemanticModelRecord{
		ID:           101,
		Name:         "SalesModel",
		Tables:       structuredTablesJSON("orders"),
		TableSetHash: "hash-a",
	}, nil).Once()
	storage.On("CreateSemanticEntry", mock.Anything, mock.MatchedBy(func(entry *tenant.SemanticEntryRecord) bool {
		return entry != nil && entry.KeyName == "gmv" && entry.Kind == "metric"
	})).Return(&tenant.SemanticEntryRecord{ID: 201, ModelID: 101, Kind: "metric", KeyName: "gmv"}, nil).Once()

	body := `{
		"name":"SalesModel",
		"tables":` + structuredTables("orders") + `,
		"entries":[
			{"kind":"logic_text","key":"disabled_legacy_rule","tables":["__disabled_legacy_obsolete_rule__"],"spec":{"content":"disabled","injection_stages":["planner_policy"]}},
			{"kind":"metric","key":"gmv","tables":["orders"],"spec":{"expr":"SUM(amount)"}}
		]
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/ws-123/semantic-models/import", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	storage.AssertExpectations(t)
}

func TestSemanticModelHandler_Import_BatchValidationFailure(t *testing.T) {
	storage := new(mockSemanticModelStorage)
	pool := &mockSemanticConnectionPool{}
	handler := NewSemanticModelHandler(pool, storage, zap.NewNop())
	router := setupSemanticModelTestRouter(handler)

	body := `[
		{"name":"SalesModelA","tables":` + structuredTables("orders") + `},
		{"name":"","tables":` + structuredTables("users") + `}
	]`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/ws-123/semantic-models/import", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	storage.AssertExpectations(t)
}

func TestSemanticModelHandler_Import_BatchConflictRollback(t *testing.T) {
	storage := new(mockSemanticModelStorage)
	pool := &mockSemanticConnectionPoolWithRollback{}
	handler := NewSemanticModelHandler(pool, storage, zap.NewNop())
	router := setupSemanticModelTestRouter(handler)

	storage.On("CreateSemanticModel", mock.Anything, mock.MatchedBy(func(model *tenant.SemanticModelRecord) bool {
		return model != nil && model.Name == "SalesModelA"
	})).Return(&tenant.SemanticModelRecord{
		ID:           101,
		Name:         "SalesModelA",
		Tables:       structuredTablesJSON("orders"),
		TableSetHash: "hash-a",
	}, nil).Once()
	storage.On("CreateSemanticModel", mock.Anything, mock.MatchedBy(func(model *tenant.SemanticModelRecord) bool {
		return model != nil && model.Name == "SalesModelB"
	})).Return(nil, tenant.ErrSemanticModelAlreadyExist).Once()

	body := `[
		{"name":"SalesModelA","tables":` + structuredTables("orders") + `},
		{"name":"SalesModelB","tables":` + structuredTables("users") + `}
	]`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/ws-123/semantic-models/import", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	storage.AssertExpectations(t)
}

func TestSemanticModelHandler_Validate_Success(t *testing.T) {
	storage := new(mockSemanticModelStorage)
	pool := &mockSemanticConnectionPool{}
	handler := NewSemanticModelHandler(pool, storage, zap.NewNop())
	router := setupSemanticModelTestRouter(handler)

	storage.On("GetSemanticModel", mock.Anything, int64(1)).
		Return(&tenant.SemanticModelRecord{
			ID:     1,
			Name:   "SalesModel",
			Tables: structuredTablesJSON("orders"),
		}, nil).
		Once()
	storage.On("ListSemanticEntries", mock.Anything, int64(1), "", mock.Anything).
		Return([]*tenant.SemanticEntryRecord{
			{
				ID:      10,
				ModelID: 1,
				Kind:    "dimension",
				KeyName: "order_id",
				Tables:  []string{"orders"},
				Spec:    json.RawMessage(`{"column":"orders.id"}`),
			},
		}, int64(1), nil).
		Once()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/ws-123/semantic-models/1/validate", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Valid bool `json:"valid"`
	}
	unmarshalAPIData(t, w.Body.Bytes(), &resp)
	assert.True(t, resp.Valid)
	storage.AssertExpectations(t)
}

func TestSemanticModelHandler_Validate_InvalidSpec(t *testing.T) {
	storage := new(mockSemanticModelStorage)
	pool := &mockSemanticConnectionPoolWithRollback{}
	handler := NewSemanticModelHandler(pool, storage, zap.NewNop())
	router := setupSemanticModelTestRouter(handler)

	storage.On("GetSemanticModel", mock.Anything, int64(1)).
		Return(&tenant.SemanticModelRecord{
			ID:     1,
			Name:   "SalesModel",
			Tables: structuredTablesJSON("orders"),
		}, nil).
		Once()
	storage.On("ListSemanticEntries", mock.Anything, int64(1), "", mock.Anything).
		Return([]*tenant.SemanticEntryRecord{
			{
				ID:      11,
				ModelID: 1,
				Kind:    "metric",
				KeyName: "gmv",
				Tables:  []string{"orders"},
				Spec:    json.RawMessage(`{"expr":"sum(amount)","requires_join":"missing_relationship"}`),
			},
		}, int64(1), nil).
		Once()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/ws-123/semantic-models/1/validate", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	resp := parseAPIErrorResponse(t, w.Body.Bytes())
	assert.Equal(t, "SESSION_RELATIONSHIP_NOT_FOUND", resp.Details[common.ErrorInfoReasonDetailKey])
	assert.Equal(t, "moi-core.session", resp.Details[common.ErrorInfoDomainDetailKey])
	assert.Equal(t, "gmv", resp.Details[common.ErrorInfoMetadataDetailPrefix+"entry_key"])
	assert.Equal(t, "spec.requires_join", resp.Details[common.ErrorInfoMetadataDetailPrefix+"field"])
	assert.Equal(t, "missing_relationship", resp.Details[common.ErrorInfoMetadataDetailPrefix+"relationship_key"])
	storage.AssertExpectations(t)
}

func TestSemanticModelHandler_CreateEntry_ValidationReasonDetails(t *testing.T) {
	storage := new(mockSemanticModelStorage)
	pool := &mockSemanticConnectionPoolWithRollback{}
	handler := NewSemanticModelHandler(pool, storage, zap.NewNop())
	router := setupSemanticModelTestRouter(handler)

	storage.On("GetSemanticModel", mock.Anything, int64(1)).
		Return(&tenant.SemanticModelRecord{
			ID:     1,
			Name:   "SalesModel",
			Tables: structuredTablesJSON("orders"),
		}, nil).
		Once()
	storage.On("ListSemanticEntries", mock.Anything, int64(1), "", mock.Anything).
		Return([]*tenant.SemanticEntryRecord{}, int64(0), nil).
		Once()

	body := `{
		"kind":"default_constraint",
		"key":"currency_default",
		"tables":["orders"],
		"spec":{"column":"currency","operator":"LIKE","values":["CNY"]}
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/ws-123/semantic-models/1/entries", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	resp := parseAPIErrorResponse(t, w.Body.Bytes())
	assert.Equal(t, "SESSION_DEFAULT_CONSTRAINT_OPERATOR_INVALID", resp.Details[common.ErrorInfoReasonDetailKey])
	assert.Equal(t, "moi-core.session", resp.Details[common.ErrorInfoDomainDetailKey])
	storage.AssertExpectations(t)
}

func TestSemanticModelHandler_Validate_ModelNotFound(t *testing.T) {
	storage := new(mockSemanticModelStorage)
	pool := &mockSemanticConnectionPoolWithRollback{}
	handler := NewSemanticModelHandler(pool, storage, zap.NewNop())
	router := setupSemanticModelTestRouter(handler)

	storage.On("GetSemanticModel", mock.Anything, int64(1)).
		Return(nil, tenant.ErrSemanticModelNotFound).
		Once()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/ws-123/semantic-models/1/validate", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	resp := parseAPIErrorResponse(t, w.Body.Bytes())
	assert.Equal(t, "SESSION_SEMANTIC_MODEL_NOT_FOUND", resp.Details[common.ErrorInfoReasonDetailKey])
	assert.Equal(t, "moi-core.session", resp.Details[common.ErrorInfoDomainDetailKey])
	storage.AssertExpectations(t)
}

func TestSemanticModelHandler_Validate_StorageError(t *testing.T) {
	storage := new(mockSemanticModelStorage)
	pool := &mockSemanticConnectionPoolWithRollback{}
	handler := NewSemanticModelHandler(pool, storage, zap.NewNop())
	router := setupSemanticModelTestRouter(handler)

	storage.On("GetSemanticModel", mock.Anything, int64(1)).
		Return(&tenant.SemanticModelRecord{
			ID:     1,
			Name:   "SalesModel",
			Tables: structuredTablesJSON("orders"),
		}, nil).
		Once()
	storage.On("ListSemanticEntries", mock.Anything, int64(1), "", mock.Anything).
		Return(nil, int64(0), errors.New("storage exploded")).
		Once()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/ws-123/semantic-models/1/validate", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	storage.AssertExpectations(t)
}

func TestSemanticModelHandler_CreateEntry_ValidationFailure(t *testing.T) {
	storage := new(mockSemanticModelStorage)
	pool := &mockSemanticConnectionPoolWithRollback{}
	handler := NewSemanticModelHandler(pool, storage, zap.NewNop())
	router := setupSemanticModelTestRouter(handler)

	storage.On("GetSemanticModel", mock.Anything, int64(1)).
		Return(&tenant.SemanticModelRecord{
			ID:     1,
			Name:   "SalesModel",
			Tables: structuredTablesJSON("orders"),
		}, nil).
		Once()
	storage.On("ListSemanticEntries", mock.Anything, int64(1), "", mock.Anything).
		Return([]*tenant.SemanticEntryRecord{}, int64(0), nil).
		Once()

	body := `{
		"kind":"metric",
		"key":"gmv",
		"tables":["orders"],
		"spec":{"expr":"sum(amount)","requires_join":"missing_relationship"}
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/ws-123/semantic-models/1/entries", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	resp := parseAPIErrorResponse(t, w.Body.Bytes())
	assert.Equal(t, "SESSION_RELATIONSHIP_NOT_FOUND", resp.Details[common.ErrorInfoReasonDetailKey])
	assert.Equal(t, "moi-core.session", resp.Details[common.ErrorInfoDomainDetailKey])
	assert.Equal(t, "gmv", resp.Details[common.ErrorInfoMetadataDetailPrefix+"entry_key"])
	assert.Equal(t, "spec.requires_join", resp.Details[common.ErrorInfoMetadataDetailPrefix+"field"])
	assert.Equal(t, "missing_relationship", resp.Details[common.ErrorInfoMetadataDetailPrefix+"relationship_key"])
	storage.AssertExpectations(t)
}

func TestSemanticModelHandler_UpdateEntry_ValidationFailure(t *testing.T) {
	storage := new(mockSemanticModelStorage)
	pool := &mockSemanticConnectionPoolWithRollback{}
	handler := NewSemanticModelHandler(pool, storage, zap.NewNop())
	router := setupSemanticModelTestRouter(handler)

	storage.On("GetSemanticModel", mock.Anything, int64(1)).
		Return(&tenant.SemanticModelRecord{
			ID:     1,
			Name:   "SalesModel",
			Tables: structuredTablesJSON("orders"),
		}, nil).
		Once()
	storage.On("GetSemanticEntry", mock.Anything, int64(1), int64(7)).
		Return(&tenant.SemanticEntryRecord{
			ID:      7,
			ModelID: 1,
			Kind:    "dimension",
			KeyName: "order_id",
			Tables:  []string{"orders"},
			Spec:    json.RawMessage(`{"column":"orders.id"}`),
		}, nil).
		Once()
	storage.On("ListSemanticEntries", mock.Anything, int64(1), "", mock.Anything).
		Return([]*tenant.SemanticEntryRecord{
			{
				ID:      7,
				ModelID: 1,
				Kind:    "dimension",
				KeyName: "order_id",
				Tables:  []string{"orders"},
				Spec:    json.RawMessage(`{"column":"orders.id"}`),
			},
		}, int64(1), nil).
		Once()

	body := `{
		"kind":"metric",
		"key":"gmv",
		"tables":["orders"],
		"spec":{"expr":"sum(amount)","requires_join":"missing_relationship"}
	}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/workspaces/ws-123/semantic-models/1/entries/7", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	resp := parseAPIErrorResponse(t, w.Body.Bytes())
	assert.Equal(t, "SESSION_RELATIONSHIP_NOT_FOUND", resp.Details[common.ErrorInfoReasonDetailKey])
	assert.Equal(t, "moi-core.session", resp.Details[common.ErrorInfoDomainDetailKey])
	assert.Equal(t, "gmv", resp.Details[common.ErrorInfoMetadataDetailPrefix+"entry_key"])
	assert.Equal(t, "spec.requires_join", resp.Details[common.ErrorInfoMetadataDetailPrefix+"field"])
	assert.Equal(t, "missing_relationship", resp.Details[common.ErrorInfoMetadataDetailPrefix+"relationship_key"])
	storage.AssertExpectations(t)
}

func TestSemanticModelHandler_Create_InvalidJSON(t *testing.T) {
	storage := new(mockSemanticModelStorage)
	pool := &mockSemanticConnectionPool{}
	handler := NewSemanticModelHandler(pool, storage, zap.NewNop())
	router := setupSemanticModelTestRouter(handler)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/ws-123/semantic-models", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSemanticModelHandler_Create_MissingName(t *testing.T) {
	storage := new(mockSemanticModelStorage)
	pool := &mockSemanticConnectionPool{}
	handler := NewSemanticModelHandler(pool, storage, zap.NewNop())
	router := setupSemanticModelTestRouter(handler)

	body := `{"tables":` + structuredTables("orders") + `}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/ws-123/semantic-models", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSemanticModelHandler_Validate_InvalidModelID(t *testing.T) {
	storage := new(mockSemanticModelStorage)
	pool := &mockSemanticConnectionPool{}
	handler := NewSemanticModelHandler(pool, storage, zap.NewNop())
	router := setupSemanticModelTestRouter(handler)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/ws-123/semantic-models/invalid/validate", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSemanticModelHandler_UpdateEntry_InvalidEntryID(t *testing.T) {
	storage := new(mockSemanticModelStorage)
	pool := &mockSemanticConnectionPool{}
	handler := NewSemanticModelHandler(pool, storage, zap.NewNop())
	router := setupSemanticModelTestRouter(handler)

	body := `{"kind":"dimension","key":"order_id","spec":{"column":"orders.id"}}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/workspaces/ws-123/semantic-models/1/entries/invalid", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSemanticModelHandler_Import_InvalidJSON(t *testing.T) {
	storage := new(mockSemanticModelStorage)
	pool := &mockSemanticConnectionPool{}
	handler := NewSemanticModelHandler(pool, storage, zap.NewNop())
	router := setupSemanticModelTestRouter(handler)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/ws-123/semantic-models/import", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSemanticModelHandler_CreateEntry_InvalidJSON(t *testing.T) {
	storage := new(mockSemanticModelStorage)
	pool := &mockSemanticConnectionPool{}
	handler := NewSemanticModelHandler(pool, storage, zap.NewNop())
	router := setupSemanticModelTestRouter(handler)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/ws-123/semantic-models/1/entries", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSemanticModelHandler_UpdateEntry_InvalidJSON(t *testing.T) {
	storage := new(mockSemanticModelStorage)
	pool := &mockSemanticConnectionPool{}
	handler := NewSemanticModelHandler(pool, storage, zap.NewNop())
	router := setupSemanticModelTestRouter(handler)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/workspaces/ws-123/semantic-models/1/entries/7", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSemanticModelHandler_Create_DuplicateTables(t *testing.T) {
	storage := new(mockSemanticModelStorage)
	pool := &mockSemanticConnectionPool{}
	handler := NewSemanticModelHandler(pool, storage, zap.NewNop())
	router := setupSemanticModelTestRouter(handler)

	body := `{"name":"SalesModel","tables":` + structuredTables("orders", "Orders") + `}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/ws-123/semantic-models", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSemanticModelHandler_Create_EmptyTables(t *testing.T) {
	storage := new(mockSemanticModelStorage)
	pool := &mockSemanticConnectionPool{}
	handler := NewSemanticModelHandler(pool, storage, zap.NewNop())
	router := setupSemanticModelTestRouter(handler)

	storage.On("CreateSemanticModel", mock.Anything, mock.MatchedBy(func(model *tenant.SemanticModelRecord) bool {
		return model != nil && model.Name == "EmptyTablesModel"
	})).Return(&tenant.SemanticModelRecord{
		ID:   201,
		Name: "EmptyTablesModel",
	}, nil).Once()

	body := `{"name":"EmptyTablesModel","description":"no tables yet","tables":[]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/ws-123/semantic-models", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	storage.AssertExpectations(t)
}

func TestSemanticModelHandler_Create_NilTables(t *testing.T) {
	storage := new(mockSemanticModelStorage)
	pool := &mockSemanticConnectionPool{}
	handler := NewSemanticModelHandler(pool, storage, zap.NewNop())
	router := setupSemanticModelTestRouter(handler)

	storage.On("CreateSemanticModel", mock.Anything, mock.MatchedBy(func(model *tenant.SemanticModelRecord) bool {
		return model != nil && model.Name == "NilTablesModel"
	})).Return(&tenant.SemanticModelRecord{
		ID:   202,
		Name: "NilTablesModel",
	}, nil).Once()

	body := `{"name":"NilTablesModel","description":"tables omitted"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/ws-123/semantic-models", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	storage.AssertExpectations(t)
}

func TestSemanticModelHandler_CreateEntry_Unauthenticated(t *testing.T) {
	storage := new(mockSemanticModelStorage)
	pool := &mockSemanticConnectionPool{}
	handler := NewSemanticModelHandler(pool, storage, zap.NewNop())

	gin.SetMode(gin.TestMode)
	router := gin.New()
	group := router.Group("/api/v1/workspaces/:id/semantic-models")
	group.POST("/:model_id/entries", handler.CreateEntry)

	body := `{"kind":"dimension","key":"order_id","spec":{"column":"orders.id"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/ws-123/semantic-models/1/entries", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestSemanticModelHandler_Validate_Unauthenticated(t *testing.T) {
	storage := new(mockSemanticModelStorage)
	pool := &mockSemanticConnectionPool{}
	handler := NewSemanticModelHandler(pool, storage, zap.NewNop())

	gin.SetMode(gin.TestMode)
	router := gin.New()
	group := router.Group("/api/v1/workspaces/:id/semantic-models")
	group.POST("/:model_id/validate", handler.Validate)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/ws-123/semantic-models/1/validate", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestSemanticModelHandler_CreateEntry_StorageAlreadyExists(t *testing.T) {
	storage := new(mockSemanticModelStorage)
	pool := &mockSemanticConnectionPoolWithRollback{}
	handler := NewSemanticModelHandler(pool, storage, zap.NewNop())
	router := setupSemanticModelTestRouter(handler)

	storage.On("GetSemanticModel", mock.Anything, int64(1)).
		Return(&tenant.SemanticModelRecord{
			ID:     1,
			Name:   "SalesModel",
			Tables: structuredTablesJSON("orders"),
		}, nil).
		Once()
	storage.On("ListSemanticEntries", mock.Anything, int64(1), "", mock.Anything).
		Return([]*tenant.SemanticEntryRecord{}, int64(0), nil).
		Once()
	storage.On("CreateSemanticEntry", mock.Anything, mock.AnythingOfType("*tenant.SemanticEntryRecord")).
		Return(nil, tenant.ErrSemanticEntryAlreadyExist).
		Once()

	body := `{"kind":"dimension","key":"order_id","tables":["orders"],"spec":{"column":"orders.id"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/ws-123/semantic-models/1/entries", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	storage.AssertExpectations(t)
}

func TestSemanticModelHandler_UpdateEntry_StorageNotFound(t *testing.T) {
	storage := new(mockSemanticModelStorage)
	pool := &mockSemanticConnectionPoolWithRollback{}
	handler := NewSemanticModelHandler(pool, storage, zap.NewNop())
	router := setupSemanticModelTestRouter(handler)

	storage.On("GetSemanticModel", mock.Anything, int64(1)).
		Return(&tenant.SemanticModelRecord{
			ID:     1,
			Name:   "SalesModel",
			Tables: structuredTablesJSON("orders"),
		}, nil).
		Once()
	storage.On("GetSemanticEntry", mock.Anything, int64(1), int64(7)).
		Return(nil, tenant.ErrSemanticEntryNotFound).
		Once()

	body := `{"kind":"dimension","key":"order_id","tables":["orders"],"spec":{"column":"orders.id"}}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/workspaces/ws-123/semantic-models/1/entries/7", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	resp := parseAPIErrorResponse(t, w.Body.Bytes())
	assert.Equal(t, "SESSION_ENTRY_NOT_FOUND", resp.Details[common.ErrorInfoReasonDetailKey])
	assert.Equal(t, "moi-core.session", resp.Details[common.ErrorInfoDomainDetailKey])
	assert.Equal(t, "7", resp.Details[common.ErrorInfoMetadataDetailPrefix+"entry_id"])
	storage.AssertExpectations(t)
}

func TestSemanticModelHandler_Import_CreateConflict(t *testing.T) {
	storage := new(mockSemanticModelStorage)
	pool := &mockSemanticConnectionPoolWithRollback{}
	handler := NewSemanticModelHandler(pool, storage, zap.NewNop())
	router := setupSemanticModelTestRouter(handler)

	storage.On("CreateSemanticModel", mock.Anything, mock.AnythingOfType("*tenant.SemanticModelRecord")).
		Return(nil, tenant.ErrSemanticModelAlreadyExist).
		Once()

	bodyObj := map[string]any{
		"name":   "SalesModel",
		"tables": []map[string]any{{"db_name": "", "table_names": []string{"orders"}, "parents": []string{}}},
		"entries": []map[string]any{
			{
				"kind": "dimension",
				"key":  "order_id",
				"spec": map[string]any{"column": "orders.id"},
			},
		},
	}
	body, err := json.Marshal(bodyObj)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/ws-123/semantic-models/import", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	storage.AssertExpectations(t)
}

func TestSemanticModelHandler_List_Success(t *testing.T) {
	storage := new(mockSemanticModelStorage)
	pool := &mockSemanticConnectionPool{}
	handler := NewSemanticModelHandler(pool, storage, zap.NewNop())
	router := setupSemanticModelTestRouter(handler)

	storage.On("ListSemanticModels", mock.Anything, mock.Anything).
		Return([]*tenant.SemanticModelRecord{
			{
				ID:           1,
				Name:         "SalesModel",
				Tables:       structuredTablesJSON("orders"),
				TableSetHash: "hash-1",
			},
		}, int64(1), nil).
		Once()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/workspaces/ws-123/semantic-models?page_size=10", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp semanticModelListResponse
	unmarshalAPIData(t, w.Body.Bytes(), &resp)
	require.Len(t, resp.Items, 1)
	assert.Equal(t, int64(1), resp.Total)
	assert.Equal(t, "SalesModel", resp.Items[0].Name)
	storage.AssertExpectations(t)
}

func TestSemanticModelHandler_ListTags_Success(t *testing.T) {
	storage := new(mockSemanticModelStorage)
	pool := &mockSemanticConnectionPool{}
	handler := NewSemanticModelHandler(pool, storage, zap.NewNop())
	router := setupSemanticModelTestRouter(handler)

	storage.On("ListSemanticModelTags", mock.Anything, mock.Anything).
		Return([]tenant.SemanticModelTagStat{{Tag: "finance", Count: 2}, {Tag: "ops", Count: 1}}, nil).
		Once()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/workspaces/ws-123/semantic-models/tags?search=Sales", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp semanticModelTagListResponse
	unmarshalAPIData(t, w.Body.Bytes(), &resp)
	require.Len(t, resp.Items, 2)
	assert.Equal(t, "finance", resp.Items[0].Tag)
	assert.Equal(t, int64(2), resp.Items[0].Count)
	storage.AssertExpectations(t)
}

func TestSemanticModelHandler_Get_NotFound(t *testing.T) {
	storage := new(mockSemanticModelStorage)
	pool := &mockSemanticConnectionPoolWithRollback{}
	handler := NewSemanticModelHandler(pool, storage, zap.NewNop())
	router := setupSemanticModelTestRouter(handler)

	storage.On("GetSemanticModel", mock.Anything, int64(1)).
		Return(nil, tenant.ErrSemanticModelNotFound).
		Once()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/workspaces/ws-123/semantic-models/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	storage.AssertExpectations(t)
}

func TestSemanticModelHandler_Update_Success(t *testing.T) {
	storage := new(mockSemanticModelStorage)
	pool := &mockSemanticConnectionPool{}
	handler := NewSemanticModelHandler(pool, storage, zap.NewNop())
	router := setupSemanticModelTestRouter(handler)

	storage.On("GetSemanticModelForUpdate", mock.Anything, int64(1)).
		Return(&tenant.SemanticModelRecord{ID: 1, Name: "SalesModel"}, nil).
		Once()
	storage.On("UpdateSemanticModel", mock.Anything, mock.MatchedBy(func(model *tenant.SemanticModelRecord) bool {
		if model == nil {
			return false
		}
		return model.ID == 1 &&
			model.Name == "SalesModelV2" &&
			model.UpdatedBy == "test-user" &&
			len(model.Tables) > 0 &&
			model.Files == nil
	})).Return(nil).Once()

	body := `{"name":"SalesModelV2","description":"updated","tables":` + structuredTables("users", "orders") + `}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/workspaces/ws-123/semantic-models/1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	storage.AssertExpectations(t)
}

func TestSemanticModelHandler_Delete_Success(t *testing.T) {
	storage := new(mockSemanticModelStorage)
	pool := &mockSemanticConnectionPool{}
	handler := NewSemanticModelHandler(pool, storage, zap.NewNop())
	router := setupSemanticModelTestRouter(handler)

	storage.On("GetSemanticModelForUpdate", mock.Anything, int64(1)).
		Return(&tenant.SemanticModelRecord{ID: 1, Name: "SalesModel"}, nil).
		Once()
	storage.On("DeleteSemanticModel", mock.Anything, int64(1)).
		Return(nil).
		Once()

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/workspaces/ws-123/semantic-models/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	data, ok := body["data"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, true, data["deleted"])
	storage.AssertExpectations(t)
}

type mockSemanticKnowledgeDeleteHandler struct {
	mock.Mock
}

func (m *mockSemanticKnowledgeDeleteHandler) HandleSemanticKnowledgeBaseDeleted(ctx context.Context, workspaceID string, modelID int64, modelName, userID string) (agentresource.SemanticKnowledgeBaseDeleteStats, error) {
	args := m.Called(ctx, workspaceID, modelID, modelName, userID)
	return args.Get(0).(agentresource.SemanticKnowledgeBaseDeleteStats), args.Error(1)
}

func (m *mockSemanticKnowledgeDeleteHandler) HandleSemanticKnowledgeBaseRenamed(ctx context.Context, workspaceID string, modelID int64, oldName, newName string) (agentresource.SemanticKnowledgeBaseDeleteStats, error) {
	args := m.Called(ctx, workspaceID, modelID, oldName, newName)
	return args.Get(0).(agentresource.SemanticKnowledgeBaseDeleteStats), args.Error(1)
}

func TestSemanticModelHandler_Update_InvalidatesOldNamePackageVersions(t *testing.T) {
	storage := new(mockSemanticModelStorage)
	pool := &mockSemanticConnectionPool{}
	cleaner := new(mockSemanticKnowledgeDeleteHandler)
	handler := NewSemanticModelHandler(pool, storage, zap.NewNop()).WithSemanticKnowledgeBaseDeleteHandler(cleaner)
	router := setupSemanticModelTestRouter(handler)

	storage.On("GetSemanticModelForUpdate", mock.Anything, int64(20001)).
		Return(&tenant.SemanticModelRecord{ID: 20001, Name: "kb_old"}, nil).
		Once()
	cleaner.On("HandleSemanticKnowledgeBaseRenamed", mock.Anything, "ws-123", int64(20001), "kb_old", "kb_new").
		Return(agentresource.SemanticKnowledgeBaseDeleteStats{NeedsConfigurationCount: 1}, nil).
		Once()
	storage.On("UpdateSemanticModel", mock.Anything, mock.MatchedBy(func(model *tenant.SemanticModelRecord) bool {
		return model != nil && model.ID == 20001 && model.Name == "kb_new"
	})).Return(nil).Once()

	body := `{"name":"kb_new","description":"renamed","tables":` + structuredTables("orders") + `}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/workspaces/ws-123/semantic-models/20001", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	storage.AssertExpectations(t)
	cleaner.AssertExpectations(t)
}

func TestSemanticModelHandler_Delete_CleansAgentReferences(t *testing.T) {
	storage := new(mockSemanticModelStorage)
	pool := &mockSemanticConnectionPool{}
	cleaner := new(mockSemanticKnowledgeDeleteHandler)
	handler := NewSemanticModelHandler(pool, storage, zap.NewNop()).WithSemanticKnowledgeBaseDeleteHandler(cleaner)
	router := setupSemanticModelTestRouter(handler)

	storage.On("GetSemanticModelForUpdate", mock.Anything, int64(20001)).
		Return(&tenant.SemanticModelRecord{ID: 20001, Name: "kb_sales"}, nil).
		Once()
	cleaner.On("HandleSemanticKnowledgeBaseDeleted", mock.Anything, "ws-123", int64(20001), "kb_sales", "test-user").
		Return(agentresource.SemanticKnowledgeBaseDeleteStats{UnboundAgents: 1, UnboundAgentBindings: 1, NeedsConfigurationCount: 2}, nil).
		Once()
	storage.On("DeleteSemanticModel", mock.Anything, int64(20001)).
		Return(nil).
		Once()

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/workspaces/ws-123/semantic-models/20001", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	storage.AssertExpectations(t)
	cleaner.AssertExpectations(t)
}

func TestSemanticModelHandler_Delete_RollsBackWhenAgentCleanupFails(t *testing.T) {
	storage := new(mockSemanticModelStorage)
	pool := &mockSemanticConnectionPoolWithRollback{}
	cleaner := new(mockSemanticKnowledgeDeleteHandler)
	handler := NewSemanticModelHandler(pool, storage, zap.NewNop()).WithSemanticKnowledgeBaseDeleteHandler(cleaner)
	router := setupSemanticModelTestRouter(handler)

	storage.On("GetSemanticModelForUpdate", mock.Anything, int64(20001)).
		Return(&tenant.SemanticModelRecord{ID: 20001, Name: "kb_sales"}, nil).
		Once()
	cleaner.On("HandleSemanticKnowledgeBaseDeleted", mock.Anything, "ws-123", int64(20001), "kb_sales", "test-user").
		Return(agentresource.SemanticKnowledgeBaseDeleteStats{}, errors.New("agent cleanup failed")).
		Once()

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/workspaces/ws-123/semantic-models/20001", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	storage.AssertNotCalled(t, "DeleteSemanticModel", mock.Anything, mock.Anything)
	storage.AssertExpectations(t)
	cleaner.AssertExpectations(t)
}

func TestSemanticModelHandler_ListEntries_Success(t *testing.T) {
	storage := new(mockSemanticModelStorage)
	pool := &mockSemanticConnectionPool{}
	handler := NewSemanticModelHandler(pool, storage, zap.NewNop())
	router := setupSemanticModelTestRouter(handler)

	storage.On("GetSemanticModel", mock.Anything, int64(1)).
		Return(&tenant.SemanticModelRecord{
			ID:     1,
			Name:   "SalesModel",
			Tables: structuredTablesJSON("orders"),
		}, nil).
		Once()
	storage.On("ListSemanticEntries", mock.Anything, int64(1), "dimension", mock.Anything).
		Return([]*tenant.SemanticEntryRecord{
			{
				ID:      10,
				ModelID: 1,
				Kind:    "dimension",
				KeyName: "order_id",
				Tables:  []string{"orders"},
				Spec:    json.RawMessage(`{"column":"orders.id"}`),
			},
		}, int64(1), nil).
		Once()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/workspaces/ws-123/semantic-models/1/entries?kind=Dimension", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp semanticEntryListResponse
	unmarshalAPIData(t, w.Body.Bytes(), &resp)
	require.Len(t, resp.Items, 1)
	assert.Equal(t, "dimension", resp.Items[0].Kind)
	storage.AssertExpectations(t)
}

func TestSemanticModelHandler_Export_Success(t *testing.T) {
	storage := new(mockSemanticModelStorage)
	pool := &mockSemanticConnectionPool{}
	handler := NewSemanticModelHandler(pool, storage, zap.NewNop())
	router := setupSemanticModelTestRouter(handler)

	storage.On("GetSemanticModel", mock.Anything, int64(1)).
		Return(&tenant.SemanticModelRecord{
			ID:           1,
			Name:         "SalesModel",
			Tables:       structuredTablesJSON("orders"),
			TableSetHash: "hash-1",
		}, nil).
		Once()
	storage.On("ListSemanticEntries", mock.Anything, int64(1), "", mock.Anything).
		Return([]*tenant.SemanticEntryRecord{
			{
				ID:      10,
				ModelID: 1,
				Kind:    "dimension",
				KeyName: "order_id",
				Tables:  []string{"orders"},
				Spec:    json.RawMessage(`{"column":"orders.id"}`),
			},
		}, int64(1), nil).
		Once()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/workspaces/ws-123/semantic-models/1/export", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]json.RawMessage
	unmarshalAPIData(t, w.Body.Bytes(), &resp)

	var model semanticModelResponse
	require.NoError(t, json.Unmarshal(resp["model"], &model))
	assert.Equal(t, int64(1), model.ID)

	var entries []semanticEntryResponse
	require.NoError(t, json.Unmarshal(resp["entries"], &entries))
	require.Len(t, entries, 1)
	assert.Equal(t, "order_id", entries[0].Key)
	storage.AssertExpectations(t)
}

func TestSemanticModelHandler_DeleteEntry_Success(t *testing.T) {
	storage := new(mockSemanticModelStorage)
	pool := &mockSemanticConnectionPool{}
	handler := NewSemanticModelHandler(pool, storage, zap.NewNop())
	router := setupSemanticModelTestRouter(handler)

	storage.On("GetSemanticModel", mock.Anything, int64(1)).
		Return(&tenant.SemanticModelRecord{
			ID:     1,
			Name:   "SalesModel",
			Tables: structuredTablesJSON("orders"),
		}, nil).
		Once()
	storage.On("DeleteSemanticEntry", mock.Anything, int64(1), int64(10)).
		Return(nil).
		Once()

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/workspaces/ws-123/semantic-models/1/entries/10", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	storage.AssertExpectations(t)
}

func TestValidateSemanticModelSpec_LogicText_Success(t *testing.T) {
	err := validateSemanticModelSpec(
		structuredTablesJSON("orders"),
		[]semanticEntryInput{
			{
				Kind:   "logic_text",
				Key:    "default_time_scope",
				Tables: []string{"orders"},
				Spec:   json.RawMessage(`{"content":"如果只给年份默认取12月","injection_stages":["planner_policy","sql_generation"],"priority":10}`),
			},
		},
	)
	require.NoError(t, err)
}

func TestValidateSemanticModelSpec_LogicText_InvalidStage(t *testing.T) {
	err := validateSemanticModelSpec(
		structuredTablesJSON("orders"),
		[]semanticEntryInput{
			{
				Kind:   "logic_text",
				Key:    "bad_stage",
				Tables: []string{"orders"},
				Spec:   json.RawMessage(`{"content":"x","injection_stages":["unknown_stage"]}`),
			},
		},
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported stage")
}

func TestValidateSemanticModelSpec_LogicText_EmptyContent(t *testing.T) {
	err := validateSemanticModelSpec(
		structuredTablesJSON("orders"),
		[]semanticEntryInput{
			{
				Kind:   "logic_text",
				Key:    "empty_content",
				Tables: []string{"orders"},
				Spec:   json.RawMessage(`{"content":"   ","injection_stages":["sql_generation"]}`),
			},
		},
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "logic_text.content is required")
}
