package sessionh

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	moi "github.com/matrixflow/moi-core/go-sdk"
	"github.com/matrixflow/moi-core/model/common"
	iampb "github.com/matrixflow/moi-core/model/iam"
	"github.com/matrixorigin/matrixflow/moi-backend/pkg/ctxutil"
	"github.com/matrixorigin/matrixflow/moi-backend/pkg/i18n"
	"github.com/matrixorigin/matrixflow/moi-backend/pkg/iampep"
	"github.com/matrixorigin/matrixflow/moi-backend/pkg/middleware"
	session "github.com/matrixorigin/matrixflow/moi-backend/pkg/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	gmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// MockSemanticModelService is a mock implementation of SemanticModelService.
type MockSemanticModelService struct {
	mock.Mock
}

type semanticModelArtifactPreviewFileService struct {
	previewCalls []string
}

func TestMapServiceErrorRedactsVectorTableDiagnostics(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPatch, "/semantic-models/1/sources/source-1/governance", nil)
	err := &session.ServiceError{
		Code: session.ErrCodeBadRequest,
		Err:  i18n.WrapError(i18n.KeySessionVectorTableUnavailable, fmt.Errorf("table secret_vector does not exist"), nil),
	}

	mapServiceError(c, err)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.JSONEq(t, `{"code":"ErrParamInvalid","msg":"Vector table does not exist or is not visible","data":null}`, rec.Body.String())
	require.NotContains(t, rec.Body.String(), "secret_vector")
}

func (f *semanticModelArtifactPreviewFileService) UploadFile(context.Context, string, io.Reader) (string, error) {
	return "", fmt.Errorf("unexpected artifact preview UploadFile call")
}

func (f *semanticModelArtifactPreviewFileService) AddFilesToVolume(context.Context, int64, []string) error {
	return fmt.Errorf("unexpected artifact preview AddFilesToVolume call")
}

func (f *semanticModelArtifactPreviewFileService) ListFiles(context.Context, session.KnowledgeBaseCatalogFileListParams) (*session.KnowledgeBaseCatalogFileListResult, error) {
	return nil, fmt.Errorf("unexpected artifact preview ListFiles call")
}

func (f *semanticModelArtifactPreviewFileService) PreviewFile(_ context.Context, fileID string) (*session.SemanticModelArtifactPreview, error) {
	f.previewCalls = append(f.previewCalls, fileID)
	return &session.SemanticModelArtifactPreview{
		Filename:    fileID + ".png",
		ContentType: "image/png",
		Body:        io.NopCloser(strings.NewReader("trusted-png-bytes")),
	}, nil
}

func (f *semanticModelArtifactPreviewFileService) DeleteFileFromVolume(context.Context, int64, string) error {
	return fmt.Errorf("unexpected artifact preview DeleteFileFromVolume call")
}

func testStringPtr(value string) *string {
	return &value
}

type semanticModelAllowIAMClient struct{}

func (semanticModelAllowIAMClient) Authorize(_ context.Context, req *iampb.AuthorizeRequest) (*iampb.AuthorizeDecision, error) {
	return semanticModelDecision(req, iampb.AuthorizeDecisionKind_AUTHORIZE_DECISION_KIND_ALLOW), nil
}

type semanticModelDependencyResolverStub struct{}

func (semanticModelDependencyResolverStub) ResolveSourceDependencies(_ context.Context, _ string, sources []session.CreateSemanticModelSourceRequest) ([]iampep.ResourceAuthorization, error) {
	out := make([]iampep.ResourceAuthorization, 0, len(sources))
	for _, source := range sources {
		switch source.SourceType {
		case "catalog_file":
			if source.VolumeID <= 0 {
				return nil, fmt.Errorf("catalog file %s requires volume_id", source.FileID)
			}
			// Mirror production: authorize the supplied volume root id as string.
			out = append(out, iampep.ResourceAuthorization{ActionID: "volume.read", Resource: iampep.ResourceDescriptor{ResourceType: iampep.ResourceTypeVolume, ResourceID: strconv.FormatInt(source.VolumeID, 10)}})
		case "catalog_table":
			out = append(out, iampep.ResourceAuthorization{ActionID: "table.read", Resource: iampep.ResourceDescriptor{ResourceType: iampep.ResourceTypeTable, ResourceID: strconv.FormatInt(source.TableID, 10)}})
		}
	}
	return out, nil
}

func (semanticModelDependencyResolverStub) ResolveSelectionDependencies(_ context.Context, _ string, selections []session.SemanticModelSourceSelectionRequest) ([]iampep.ResourceAuthorization, error) {
	out := make([]iampep.ResourceAuthorization, 0, len(selections))
	for _, selection := range selections {
		switch selection.Kind {
		case semanticSelectionDatabase:
			out = append(out, iampep.ResourceAuthorization{ActionID: "database.read", Resource: iampep.ResourceDescriptor{ResourceType: iampep.ResourceTypeDatabase, ResourceID: strconv.FormatInt(selection.DatabaseID, 10)}})
		case semanticSelectionVolume:
			// Mirror production: authorize the selection volume id (root resolution is covered by CoreSemanticModelDependencyResolver tests).
			out = append(out, iampep.ResourceAuthorization{ActionID: "volume.read", Resource: iampep.ResourceDescriptor{ResourceType: iampep.ResourceTypeVolume, ResourceID: strconv.FormatInt(selection.VolumeID, 10)}})
		}
	}
	return out, nil
}

func (semanticModelDependencyResolverStub) ResolveLegacyDependencies(context.Context, string, []byte, []byte) ([]iampep.ResourceAuthorization, error) {
	return nil, nil
}

func (semanticModelDependencyResolverStub) ResolveBackfillDependencies(context.Context, string, int64) ([]iampep.ResourceAuthorization, error) {
	return nil, nil
}

type semanticModelIAMClientFunc func(context.Context, *iampb.AuthorizeRequest) (*iampb.AuthorizeDecision, error)

func (f semanticModelIAMClientFunc) Authorize(ctx context.Context, req *iampb.AuthorizeRequest) (*iampb.AuthorizeDecision, error) {
	return f(ctx, req)
}

func semanticModelDecision(req *iampb.AuthorizeRequest, decision iampb.AuthorizeDecisionKind) *iampb.AuthorizeDecision {
	result := &iampb.AuthorizeDecision{Decision: decision, DecisionLogId: "semantic-model-test", RequestId: req.GetRequestId(), TraceId: req.GetTraceId()}
	if decision == iampb.AuthorizeDecisionKind_AUTHORIZE_DECISION_KIND_ALLOW {
		result.ErrorCode = iampb.IAMErrorCode_IAM_ERROR_CODE_ALLOWED
		result.VerifiedEffectiveRoleId = "role-1"
		result.Versions = &iampb.VersionVector{PolicyVersion: "1", SchemaVersion: "1", RoleGraphVersion: "1", OwnershipVersion: "1", LifecycleVersion: "1"}
	} else {
		result.ErrorCode = iampb.IAMErrorCode_IAM_ERROR_CODE_PERMISSION_DENIED
		result.I18NKey = "iam.permissionDenied"
	}
	return result
}

func (m *MockSemanticModelService) CreateModel(ctx context.Context, params session.CreateSemanticModelRequest) (*session.SemanticModelInfo, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*session.SemanticModelInfo), args.Error(1)
}

func (m *MockSemanticModelService) CreateEmptyModel(ctx context.Context, params session.CreateEmptySemanticModelRequest) (*session.CreateEmptySemanticModelResponse, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*session.CreateEmptySemanticModelResponse), args.Error(1)
}

func (m *MockSemanticModelService) CreateModelWithSources(ctx context.Context, params session.CreateSemanticModelWithSourcesRequest) (*session.CreateSemanticModelWithSourcesResponse, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*session.CreateSemanticModelWithSourcesResponse), args.Error(1)
}

func (m *MockSemanticModelService) UploadLocalFile(ctx context.Context, fileName string, reader io.Reader) (string, error) {
	args := m.Called(ctx, fileName, reader)
	return args.String(0), args.Error(1)
}

func (m *MockSemanticModelService) AppendModelSources(ctx context.Context, params session.AppendSemanticModelSourcesRequest) (*session.AppendSemanticModelSourcesResponse, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*session.AppendSemanticModelSourcesResponse), args.Error(1)
}

func (m *MockSemanticModelService) ListModels(ctx context.Context, params session.ListSemanticModelsRequest) (*session.ListSemanticModelsResponse, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*session.ListSemanticModelsResponse), args.Error(1)
}

func (m *MockSemanticModelService) ListModelTags(ctx context.Context, params session.ListSemanticModelsRequest) (*session.ListSemanticModelTagsResponse, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*session.ListSemanticModelTagsResponse), args.Error(1)
}

func (m *MockSemanticModelService) ListModelsByIDs(ctx context.Context, ids []int64, params session.ListSemanticModelsRequest) (*session.ListSemanticModelsResponse, error) {
	args := m.Called(ctx, ids, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*session.ListSemanticModelsResponse), args.Error(1)
}

func (m *MockSemanticModelService) UpdateModel(ctx context.Context, params session.UpdateSemanticModelRequest) error {
	args := m.Called(ctx, params)
	return args.Error(0)
}

func (m *MockSemanticModelService) DeleteModel(ctx context.Context, modelID int) error {
	args := m.Called(ctx, modelID)
	return args.Error(0)
}

func (m *MockSemanticModelService) DeleteSource(ctx context.Context, params session.DeleteSemanticModelSourceParams) error {
	args := m.Called(ctx, params)
	return args.Error(0)
}

func (m *MockSemanticModelService) GetModel(ctx context.Context, kbID int) (*session.SemanticModelInfo, error) {
	args := m.Called(ctx, kbID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*session.SemanticModelInfo), args.Error(1)
}

func (m *MockSemanticModelService) PreviewArtifact(ctx context.Context, modelID int, fileID string) (*session.SemanticModelArtifactPreview, error) {
	args := m.Called(ctx, modelID, fileID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*session.SemanticModelArtifactPreview), args.Error(1)
}

func (m *MockSemanticModelService) PreviewSourceFile(ctx context.Context, modelID int, fileID string) (*session.SemanticModelFilePreview, error) {
	args := m.Called(ctx, modelID, fileID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*session.SemanticModelFilePreview), args.Error(1)
}

func (m *MockSemanticModelService) ListSources(ctx context.Context, params session.ListSemanticModelSourcesParams) (*session.ListSemanticModelSourcesResult, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*session.ListSemanticModelSourcesResult), args.Error(1)
}

func (m *MockSemanticModelService) GetSourceDocument(ctx context.Context, params session.GetSemanticModelSourceDocumentParams) (*session.SemanticModelSourceDocument, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*session.SemanticModelSourceDocument), args.Error(1)
}

func (m *MockSemanticModelService) ResolveLegacySourceIAMDependencies(ctx context.Context, tables, files json.RawMessage) ([]session.CreateSemanticModelSourceRequest, error) {
	args := m.Called(ctx, tables, files)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]session.CreateSemanticModelSourceRequest), args.Error(1)
}

func (m *MockSemanticModelService) ResolveBackfillSourceIAMDependencies(ctx context.Context, modelID int64) ([]session.CreateSemanticModelSourceRequest, error) {
	args := m.Called(ctx, modelID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]session.CreateSemanticModelSourceRequest), args.Error(1)
}

func (m *MockSemanticModelService) CheckSourceExistence(ctx context.Context, params session.CheckSemanticModelSourceExistenceParams) (*session.CheckSemanticModelSourceExistenceResult, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*session.CheckSemanticModelSourceExistenceResult), args.Error(1)
}

func (m *MockSemanticModelService) PreviewSourceSelectionCounts(ctx context.Context, params session.PreviewSemanticModelSourceSelectionsRequest) (*session.PreviewSemanticModelSourceSelectionsResponse, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*session.PreviewSemanticModelSourceSelectionsResponse), args.Error(1)
}

func (m *MockSemanticModelService) UpdateSourceGovernance(ctx context.Context, params session.UpdateSemanticModelSourceGovernanceParams) (*session.UpdateSemanticModelSourceGovernanceResult, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*session.UpdateSemanticModelSourceGovernanceResult), args.Error(1)
}

func (m *MockSemanticModelService) ImportInitialSegments(ctx context.Context, params session.ImportInitialSemanticModelSegmentsParams) (*session.SemanticModelSegmentMutationResult, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*session.SemanticModelSegmentMutationResult), args.Error(1)
}

func (m *MockSemanticModelService) UpdateSegment(ctx context.Context, params session.UpdateSemanticModelSegmentParams) (*session.SemanticModelSegmentMutationResult, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*session.SemanticModelSegmentMutationResult), args.Error(1)
}

func (m *MockSemanticModelService) CreateSegment(ctx context.Context, params session.CreateSemanticModelSegmentParams) (*session.SemanticModelSegmentMutationResult, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*session.SemanticModelSegmentMutationResult), args.Error(1)
}

func (m *MockSemanticModelService) UpdateSegmentEnabled(ctx context.Context, params session.UpdateSemanticModelSegmentEnabledParams) (*session.SemanticModelSegmentMutationResult, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*session.SemanticModelSegmentMutationResult), args.Error(1)
}

func (m *MockSemanticModelService) DeleteSegment(ctx context.Context, params session.DeleteSemanticModelSegmentParams) (*session.SemanticModelSegmentMutationResult, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*session.SemanticModelSegmentMutationResult), args.Error(1)
}

func (m *MockSemanticModelService) ReembedSegments(ctx context.Context, params session.ReembedSemanticModelSegmentsParams) (*session.SemanticModelSegmentMutationResult, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*session.SemanticModelSegmentMutationResult), args.Error(1)
}

func (m *MockSemanticModelService) SetCurrentSegmentVersion(ctx context.Context, params session.SetCurrentSemanticModelSegmentVersionParams) (*session.SemanticModelSegmentMutationResult, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*session.SemanticModelSegmentMutationResult), args.Error(1)
}

func (m *MockSemanticModelService) ListSourceJobs(ctx context.Context, params session.ListSemanticModelSourceJobsParams) (*session.ListSemanticModelSourceJobsResult, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*session.ListSemanticModelSourceJobsResult), args.Error(1)
}

func (m *MockSemanticModelService) BackfillLegacySources(ctx context.Context, params session.BackfillLegacyKnowledgeBaseSourcesParams) error {
	args := m.Called(ctx, params)
	return args.Error(0)
}

func (m *MockSemanticModelService) RunPendingKnowledgeBaseSourceJobs(ctx context.Context, params session.RunPendingKnowledgeBaseSourceJobsParams) error {
	args := m.Called(ctx, params)
	return args.Error(0)
}

func (m *MockSemanticModelService) ReconcileKnowledgeBaseSourceJobs(ctx context.Context, params session.ReconcileKnowledgeBaseSourceJobsParams) error {
	args := m.Called(ctx, params)
	return args.Error(0)
}

func (m *MockSemanticModelService) ListEntries(ctx context.Context, params session.ListSemanticEntriesRequest) (*session.ListSemanticEntriesResponse, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*session.ListSemanticEntriesResponse), args.Error(1)
}

func (m *MockSemanticModelService) CreateEntry(ctx context.Context, params session.CreateSemanticEntryRequest) (*session.SemanticEntry, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*session.SemanticEntry), args.Error(1)
}

func (m *MockSemanticModelService) UpdateEntry(ctx context.Context, params session.UpdateSemanticEntryRequest) error {
	args := m.Called(ctx, params)
	return args.Error(0)
}

func (m *MockSemanticModelService) DeleteEntry(ctx context.Context, params session.DeleteSemanticEntryRequest) error {
	args := m.Called(ctx, params)
	return args.Error(0)
}

func (m *MockSemanticModelService) Import(ctx context.Context, params session.ImportSemanticModelRequest) (*session.ImportSemanticModelResponse, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*session.ImportSemanticModelResponse), args.Error(1)
}

func (m *MockSemanticModelService) Export(ctx context.Context, kbID int) (*session.ExportSemanticModelResponse, error) {
	args := m.Called(ctx, kbID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*session.ExportSemanticModelResponse), args.Error(1)
}

func (m *MockSemanticModelService) Validate(ctx context.Context, kbID int) (*session.ValidateSemanticModelResponse, error) {
	args := m.Called(ctx, kbID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*session.ValidateSemanticModelResponse), args.Error(1)
}

func setupSemanticModelRouter(svc session.SemanticModelService) *gin.Engine {
	return setupSemanticModelRouterWithIAM(svc, semanticModelAllowIAMClient{})
}

func setupSemanticModelRouterWithIAM(svc session.SemanticModelService, client iampep.AuthorizeClient) *gin.Engine {
	return setupSemanticModelRouterWithIAMAndDependencies(svc, client, semanticModelDependencyResolverStub{})
}

func setupSemanticModelRouterWithIAMAndDependencies(svc session.SemanticModelService, client iampep.AuthorizeClient, resolver SemanticModelDependencyResolver) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(i18n.LanguageMiddleware())
	r.Use(func(c *gin.Context) {
		c.Set(middleware.KeyUserID, "user-1")
		c.Set(middleware.KeyWorkspaceID, "ws-1")
		c.Next()
	})
	c := &SemanticModelController{Svc: svc, IAMPEP: iampep.BindingConfig{Client: client}, DependencyResolver: resolver}
	c.RegisterRoutes(r.Group("/newmoi"))
	return r
}

func setupSemanticModelRouterWithTenantDB(svc session.SemanticModelService, tenantDB *gorm.DB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(i18n.LanguageMiddleware())
	r.Use(func(c *gin.Context) {
		c.Request = c.Request.WithContext(ctxutil.WithTenantDB(c.Request.Context(), tenantDB))
		c.Set(middleware.KeyUserID, "user-1")
		c.Set(middleware.KeyWorkspaceID, "ws-1")
		c.Next()
	})
	controller := &SemanticModelController{
		Svc:                svc,
		IAMPEP:             iampep.BindingConfig{Client: semanticModelAllowIAMClient{}},
		DependencyResolver: semanticModelDependencyResolverStub{},
	}
	controller.RegisterRoutes(r.Group("/newmoi"))
	return r
}

func TestSemanticModelController_CreateDeniedBeforeService(t *testing.T) {
	svc := new(MockSemanticModelService)
	router := setupSemanticModelRouterWithIAM(svc, semanticModelIAMClientFunc(func(_ context.Context, req *iampb.AuthorizeRequest) (*iampb.AuthorizeDecision, error) {
		return semanticModelDecision(req, iampb.AuthorizeDecisionKind_AUTHORIZE_DECISION_KIND_DENY), nil
	}))
	req := httptest.NewRequest(http.MethodPost, "/newmoi/semantic-models", strings.NewReader(`{"name":"denied"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusForbidden, w.Code, w.Body.String())
	svc.AssertNotCalled(t, "CreateModel", mock.Anything, mock.Anything)
}

func TestSemanticModelController_PreviewArtifactStreamsForCoreAllowedReaderAndAdmin(t *testing.T) {
	for _, tc := range []struct {
		name       string
		roleID     string
		superAdmin bool
	}{
		{name: "knowledge base reader", roleID: "role-kb-reader"},
		{name: "super admin", roleID: "role-super-admin", superAdmin: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc := new(MockSemanticModelService)
			svc.On("PreviewArtifact", mock.MatchedBy(func(ctx context.Context) bool {
				trusted, ok := ctxutil.CoreIAMRequestFrom(ctx)
				return ok && trusted.VerifiedEffectiveRoleID == tc.roleID
			}), 42, "page-image-9").Return(&session.SemanticModelArtifactPreview{
				Filename:    "page-9.png",
				ContentType: "image/png",
				Body:        io.NopCloser(strings.NewReader("png-bytes")),
			}, nil)

			var authorizedAction string
			var authorizedResourceID string
			router := setupSemanticModelRouterWithIAM(svc, semanticModelIAMClientFunc(func(_ context.Context, req *iampb.AuthorizeRequest) (*iampb.AuthorizeDecision, error) {
				authorizedAction = req.GetActionId()
				if req.GetResource() != nil {
					authorizedResourceID = req.GetResource().GetResourceId()
				}
				decision := semanticModelDecision(req, iampb.AuthorizeDecisionKind_AUTHORIZE_DECISION_KIND_ALLOW)
				decision.VerifiedEffectiveRoleId = tc.roleID
				if tc.superAdmin {
					decision.Source = iampep.AuthorizeDecisionSourceSuperAdmin
					decision.Versions.PolicyVersion = ""
					decision.Versions.RoleGraphVersion = ""
				}
				return decision, nil
			}))

			req := httptest.NewRequest(http.MethodGet, "/newmoi/semantic-models/42/artifacts/page-image-9/preview", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			require.Equal(t, http.StatusOK, w.Code, w.Body.String())
			assert.Equal(t, "semantic_model.read", authorizedAction)
			assert.Equal(t, "42", authorizedResourceID)
			assert.Equal(t, "image/png", w.Header().Get("Content-Type"))
			assert.Contains(t, w.Header().Get("Content-Disposition"), "page-9.png")
			assert.Equal(t, "png-bytes", w.Body.String())
			svc.AssertExpectations(t)
		})
	}
}

func TestSemanticModelController_PreviewArtifactDeniedBeforeService(t *testing.T) {
	svc := new(MockSemanticModelService)
	router := setupSemanticModelRouterWithIAM(svc, semanticModelIAMClientFunc(func(_ context.Context, req *iampb.AuthorizeRequest) (*iampb.AuthorizeDecision, error) {
		return semanticModelDecision(req, iampb.AuthorizeDecisionKind_AUTHORIZE_DECISION_KIND_DENY), nil
	}))

	req := httptest.NewRequest(http.MethodGet, "/newmoi/semantic-models/42/artifacts/page-image-9/preview", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusForbidden, w.Code, w.Body.String())
	svc.AssertNotCalled(t, "PreviewArtifact", mock.Anything, mock.Anything, mock.Anything)
}

func TestSemanticModelController_PreviewSourceFileStreamsForCoreAllowedReaderAndAdmin(t *testing.T) {
	for _, tc := range []struct {
		name       string
		roleID     string
		superAdmin bool
	}{
		{name: "knowledge base reader", roleID: "role-kb-reader"},
		{name: "super admin", roleID: "role-super-admin", superAdmin: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc := new(MockSemanticModelService)
			svc.On("PreviewSourceFile", mock.MatchedBy(func(ctx context.Context) bool {
				trusted, ok := ctxutil.CoreIAMRequestFrom(ctx)
				return ok && trusted.VerifiedEffectiveRoleID == tc.roleID
			}), 42, "source-file-42").Return(&session.SemanticModelFilePreview{
				Filename:    "source.pdf",
				ContentType: "application/pdf",
				Body:        io.NopCloser(strings.NewReader("pdf-bytes")),
			}, nil)

			var authorizedAction string
			var authorizedResourceID string
			router := setupSemanticModelRouterWithIAM(svc, semanticModelIAMClientFunc(func(_ context.Context, req *iampb.AuthorizeRequest) (*iampb.AuthorizeDecision, error) {
				authorizedAction = req.GetActionId()
				if req.GetResource() != nil {
					authorizedResourceID = req.GetResource().GetResourceId()
				}
				decision := semanticModelDecision(req, iampb.AuthorizeDecisionKind_AUTHORIZE_DECISION_KIND_ALLOW)
				decision.VerifiedEffectiveRoleId = tc.roleID
				if tc.superAdmin {
					decision.Source = iampep.AuthorizeDecisionSourceSuperAdmin
					decision.Versions.PolicyVersion = ""
					decision.Versions.RoleGraphVersion = ""
				}
				return decision, nil
			}))

			req := httptest.NewRequest(http.MethodGet, "/newmoi/semantic-models/42/sources/file/source-file-42/preview", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			require.Equal(t, http.StatusOK, w.Code, w.Body.String())
			assert.Equal(t, "semantic_model.read", authorizedAction)
			assert.Equal(t, "42", authorizedResourceID)
			assert.Equal(t, "application/pdf", w.Header().Get("Content-Type"))
			assert.Contains(t, w.Header().Get("Content-Disposition"), "source.pdf")
			assert.Equal(t, "pdf-bytes", w.Body.String())
			svc.AssertExpectations(t)
		})
	}
}

func TestSemanticModelController_PreviewSourceFileDeniedBeforeService(t *testing.T) {
	svc := new(MockSemanticModelService)
	router := setupSemanticModelRouterWithIAM(svc, semanticModelIAMClientFunc(func(_ context.Context, req *iampb.AuthorizeRequest) (*iampb.AuthorizeDecision, error) {
		return semanticModelDecision(req, iampb.AuthorizeDecisionKind_AUTHORIZE_DECISION_KIND_DENY), nil
	}))

	req := httptest.NewRequest(http.MethodGet, "/newmoi/semantic-models/42/sources/file/source-file-42/preview", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusForbidden, w.Code, w.Body.String())
	svc.AssertNotCalled(t, "PreviewSourceFile", mock.Anything, mock.Anything, mock.Anything)
}

func TestSemanticModelController_PreviewArtifactNotAssociatedReturnsNotFound(t *testing.T) {
	svc := new(MockSemanticModelService)
	svc.On("PreviewArtifact", mock.Anything, 42, "unrelated-file").Return(nil, &session.ServiceError{
		Code: session.ErrCodeNotFound,
		Msg:  "semantic model artifact not found",
	})
	router := setupSemanticModelRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/newmoi/semantic-models/42/artifacts/unrelated-file/preview", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusNotFound, w.Code, w.Body.String())
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "ErrNotFound", body["code"])
	svc.AssertExpectations(t)
}

func TestSemanticModelController_PreviewArtifactRealServiceWorkflowLineageIntegration(t *testing.T) {
	const workflowAssociationQuery = `(?s)SELECT COUNT\(\*\).*FROM semantic_models sm.*INNER JOIN data_asset vector_asset.*INNER JOIN data_derivation indexed_derivation.*LEFT JOIN parsed_manifest pm.*WHERE sm\.id = \?.*pm\.parsed_file_id = \?.*artifact_derivation.*artifact\.asset_ref = \?`
	const workflowVectorTablesQuery = `(?s)SELECT JSON_UNQUOTE\(JSON_EXTRACT\(files, '\$\.vector_table'\)\) AS table_name.*UNION.*JSON_UNQUOTE\(JSON_EXTRACT\(files, '\$\.image_vector_table'\)\) AS table_name`

	for _, tc := range []struct {
		name         string
		fileID       string
		associations int64
		wantStatus   int
		wantBody     string
		wantCalls    []string
	}{
		{
			name:         "trusted parser artifact streams",
			fileID:       "trusted-page-image",
			associations: 1,
			wantStatus:   http.StatusOK,
			wantBody:     "trusted-png-bytes",
			wantCalls:    []string{"trusted-page-image"},
		},
		{
			name:         "forged artifact fails closed",
			fileID:       "caller-forged-image",
			associations: 0,
			wantStatus:   http.StatusNotFound,
			wantCalls:    []string{},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tenantSQL, tenantMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
			require.NoError(t, err)
			t.Cleanup(func() { _ = tenantSQL.Close() })
			tenantDB, err := gorm.Open(gmysql.New(gmysql.Config{Conn: tenantSQL, SkipInitializeWithVersion: true}), &gorm.Config{})
			require.NoError(t, err)
			tenantMock.ExpectQuery(workflowAssociationQuery).
				WithArgs(
					int64(42),
					tc.fileID,
					tc.fileID,
				).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(tc.associations))
			if tc.associations == 0 {
				tenantMock.ExpectQuery(workflowVectorTablesQuery).
					WithArgs(int64(42), int64(42)).
					WillReturnRows(sqlmock.NewRows([]string{"table_name"}))
			}

			fileSvc := &semanticModelArtifactPreviewFileService{previewCalls: []string{}}
			svc := session.NewSemanticModelServiceWithDependencies(nil, fileSvc)
			router := setupSemanticModelRouterWithTenantDB(svc, tenantDB)
			req := httptest.NewRequest(http.MethodGet, "/newmoi/semantic-models/42/artifacts/"+tc.fileID+"/preview", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			require.Equal(t, tc.wantStatus, w.Code, w.Body.String())
			if tc.wantStatus == http.StatusOK {
				require.Equal(t, tc.wantBody, w.Body.String())
			} else {
				var body map[string]any
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
				require.Equal(t, "ErrNotFound", body["code"])
			}
			require.Equal(t, tc.wantCalls, fileSvc.previewCalls)
			require.NoError(t, tenantMock.ExpectationsWereMet())
		})
	}
}

func TestSemanticModelController_PreviewArtifactRejectsInvalidModelIDBeforeService(t *testing.T) {
	for _, modelID := range []string{"-1", "not-a-number"} {
		t.Run(modelID, func(t *testing.T) {
			svc := new(MockSemanticModelService)
			router := setupSemanticModelRouter(svc)

			req := httptest.NewRequest(http.MethodGet, "/newmoi/semantic-models/"+modelID+"/artifacts/page-image-9/preview", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
			var body map[string]any
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
			assert.Equal(t, "ErrParamInvalid", body["code"])
			svc.AssertNotCalled(t, "PreviewArtifact", mock.Anything, mock.Anything, mock.Anything)
		})
	}
}

func TestFixSemanticModelControllerPreviewArtifactLocalizesInvalidFileID(t *testing.T) {
	tests := []struct {
		locale      string
		wantMessage string
	}{
		{locale: "en-US", wantMessage: "Invalid parameter"},
		{locale: "zh-CN", wantMessage: "参数无效"},
	}

	for _, tc := range tests {
		t.Run(tc.locale, func(t *testing.T) {
			svc := new(MockSemanticModelService)
			router := setupSemanticModelRouter(svc)
			req := httptest.NewRequest(http.MethodGet, "/newmoi/semantic-models/42/artifacts/%20/preview", nil)
			req.Header.Set("Accept-Language", tc.locale)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
			var body map[string]any
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
			assert.Equal(t, "ErrParamInvalid", body["code"])
			assert.Equal(t, tc.wantMessage, body["msg"])
			svc.AssertNotCalled(t, "PreviewArtifact", mock.Anything, mock.Anything, mock.Anything)
		})
	}
}

func TestSemanticModelController_AppendSourcesSecondPermissionDeniedBeforeService(t *testing.T) {
	svc := new(MockSemanticModelService)
	var actions []string
	router := setupSemanticModelRouterWithIAM(svc, semanticModelIAMClientFunc(func(_ context.Context, req *iampb.AuthorizeRequest) (*iampb.AuthorizeDecision, error) {
		actions = append(actions, req.GetActionId())
		decision := iampb.AuthorizeDecisionKind_AUTHORIZE_DECISION_KIND_ALLOW
		if req.GetActionId() == "semantic_model.update" {
			decision = iampb.AuthorizeDecisionKind_AUTHORIZE_DECISION_KIND_DENY
		}
		return semanticModelDecision(req, decision), nil
	}))
	req := httptest.NewRequest(http.MethodPost, "/newmoi/semantic-models/42/sources", strings.NewReader(`{"sources":[{"source_type":"catalog_file","file_id":"file-1"}]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusForbidden, w.Code, w.Body.String())
	require.Equal(t, []string{"semantic_model.use", "semantic_model.update"}, actions)
	svc.AssertNotCalled(t, "AppendModelSources", mock.Anything, mock.Anything)
}

func TestSemanticModelController_CreateSelectionPreviewRequiresCreateAndSourceParents(t *testing.T) {
	svc := new(MockSemanticModelService)
	var actions []string
	router := setupSemanticModelRouterWithIAM(svc, semanticModelIAMClientFunc(func(_ context.Context, req *iampb.AuthorizeRequest) (*iampb.AuthorizeDecision, error) {
		actions = append(actions, req.GetActionId())
		return semanticModelDecision(req, iampb.AuthorizeDecisionKind_AUTHORIZE_DECISION_KIND_ALLOW), nil
	}))
	expected := session.PreviewSemanticModelSourceSelectionsRequest{SourceSelections: []session.SemanticModelSourceSelectionRequest{
		{Kind: semanticSelectionDatabase, DatabaseID: 42, AllSelected: true},
		{Kind: semanticSelectionVolume, VolumeID: 11, AllSelected: true},
	}}
	svc.On("PreviewSourceSelectionCounts", mock.Anything, expected).Return(&session.PreviewSemanticModelSourceSelectionsResponse{FileCount: 2, TableCount: 3, TotalCount: 5}, nil)

	req := httptest.NewRequest(http.MethodPost, "/newmoi/semantic-models/source-selections/preview", strings.NewReader(`{"source_selections":[{"kind":"database_tables","database_id":42,"all_selected":true},{"kind":"volume_files","volume_id":11,"all_selected":true}]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Equal(t, []string{"semantic_model.create", "database.read", "volume.read"}, actions)
	svc.AssertExpectations(t)
}

func TestSemanticModelController_SourceExistenceRequiresModelRead(t *testing.T) {
	svc := new(MockSemanticModelService)
	var actions []string
	router := setupSemanticModelRouterWithIAM(svc, semanticModelIAMClientFunc(func(_ context.Context, req *iampb.AuthorizeRequest) (*iampb.AuthorizeDecision, error) {
		actions = append(actions, req.GetActionId())
		return semanticModelDecision(req, iampb.AuthorizeDecisionKind_AUTHORIZE_DECISION_KIND_ALLOW), nil
	}))
	expected := session.CheckSemanticModelSourceExistenceParams{ModelID: 42, FileIDs: []string{"file-1"}, TableIDs: []int64{7}}
	svc.On("CheckSourceExistence", mock.Anything, expected).Return(&session.CheckSemanticModelSourceExistenceResult{FileIDs: []string{"file-1"}, TableIDs: []int64{}}, nil)

	req := httptest.NewRequest(http.MethodPost, "/newmoi/semantic-models/42/sources/existence", strings.NewReader(`{"file_ids":["file-1"],"table_ids":[7]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Equal(t, []string{"semantic_model.read"}, actions)
	svc.AssertExpectations(t)
}

func TestSemanticModelController_AppendSelectionPreviewDependencyDeniedBeforeService(t *testing.T) {
	svc := new(MockSemanticModelService)
	var actions []string
	router := setupSemanticModelRouterWithIAM(svc, semanticModelIAMClientFunc(func(_ context.Context, req *iampb.AuthorizeRequest) (*iampb.AuthorizeDecision, error) {
		actions = append(actions, req.GetActionId())
		decision := iampb.AuthorizeDecisionKind_AUTHORIZE_DECISION_KIND_ALLOW
		if req.GetActionId() == "volume.read" {
			decision = iampb.AuthorizeDecisionKind_AUTHORIZE_DECISION_KIND_DENY
		}
		return semanticModelDecision(req, decision), nil
	}))

	req := httptest.NewRequest(http.MethodPost, "/newmoi/semantic-models/42/source-selections/preview", strings.NewReader(`{"source_selections":[{"kind":"volume_files","volume_id":11,"all_selected":true}]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusForbidden, w.Code, w.Body.String())
	require.Equal(t, []string{"semantic_model.update", "volume.read"}, actions)
	svc.AssertNotCalled(t, "PreviewSourceSelectionCounts", mock.Anything, mock.Anything)
}

func TestSemanticModelController_SelectionPreviewRejectsInvalidSelectionBeforeService(t *testing.T) {
	svc := new(MockSemanticModelService)
	var actions []string
	client := semanticModelIAMClientFunc(func(_ context.Context, req *iampb.AuthorizeRequest) (*iampb.AuthorizeDecision, error) {
		actions = append(actions, req.GetActionId())
		return semanticModelDecision(req, iampb.AuthorizeDecisionKind_AUTHORIZE_DECISION_KIND_ALLOW), nil
	})
	router := setupSemanticModelRouterWithIAMAndDependencies(svc, client, &CoreSemanticModelDependencyResolver{})

	req := httptest.NewRequest(http.MethodPost, "/newmoi/semantic-models/source-selections/preview", strings.NewReader(`{"source_selections":[{"kind":"unknown","database_id":42,"all_selected":true}]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	require.Equal(t, []string{"semantic_model.create"}, actions)
	svc.AssertNotCalled(t, "PreviewSourceSelectionCounts", mock.Anything, mock.Anything)
}

func TestSemanticModelController_DeleteModel_WorkflowConflictReturnsLocalized409(t *testing.T) {
	svc := new(MockSemanticModelService)
	router := setupSemanticModelRouter(svc)

	err := &session.ServiceError{
		Code: session.ErrCodeConflict,
		Err:  i18n.NewError(i18n.KeySessionKnowledgeBaseWorkflowDeleteConflict, nil),
	}
	svc.On("DeleteModel", mock.Anything, 42).Return(err)

	req := httptest.NewRequest(http.MethodDelete, "/newmoi/semantic-models/42", nil)
	req.Header.Set("Accept-Language", "zh-CN")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusConflict, w.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, "ErrConflict", body["code"])
	require.Equal(t, "知识库仍有未完成的工作流任务，请等待任务完成或取消后再删除", body["msg"])
	require.NotContains(t, w.Body.String(), "active execution")
	svc.AssertExpectations(t)
}

func TestSemanticModelController_GetModel_Success(t *testing.T) {
	svc := new(MockSemanticModelService)
	router := setupSemanticModelRouter(svc)

	expected := &session.SemanticModelInfo{ID: 1, Name: "test", Tables: json.RawMessage(`["orders"]`)}
	svc.On("GetModel", mock.Anything, 42).Return(expected, nil)

	req := httptest.NewRequest(http.MethodGet, "/newmoi/semantic-models/42", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSemanticModelController_GetModel_NotFound(t *testing.T) {
	svc := new(MockSemanticModelService)
	router := setupSemanticModelRouter(svc)

	svc.On("GetModel", mock.Anything, 99).Return(nil, &session.ServiceError{Code: session.ErrCodeNotFound, Err: i18n.NewError(i18n.KeySessionSemanticModelKBNotFound, nil)})

	req := httptest.NewRequest(http.MethodGet, "/newmoi/semantic-models/99", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestSemanticModelController_GetModel_CoreSDKErrorInfo(t *testing.T) {
	tests := []struct {
		name           string
		acceptLanguage string
		want           string
	}{
		{
			name:           "en-US",
			acceptLanguage: "en-US",
			want:           "entry entry-42 not found",
		},
		{
			name:           "zh-CN",
			acceptLanguage: "zh-CN",
			want:           "条目 entry-42 不存在",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := new(MockSemanticModelService)
			router := setupSemanticModelRouter(svc)

			svc.On("GetModel", mock.Anything, 99).Return(nil, &moi.Error{
				Code:    common.ErrorCode_NOT_FOUND,
				Message: "raw producer diagnostic",
				Details: common.NewErrorInfoDetails("SESSION_ENTRY_NOT_FOUND", "moi-core.session", map[string]string{
					"entry_id": "entry-42",
				}),
			})

			req := httptest.NewRequest(http.MethodGet, "/newmoi/semantic-models/99", nil)
			req.Header.Set("Accept-Language", tt.acceptLanguage)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusNotFound, w.Code)
			var resp map[string]interface{}
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
			msg, ok := resp["msg"].(string)
			require.True(t, ok)
			assert.Equal(t, tt.want, msg)
			assert.NotContains(t, msg, "raw producer diagnostic")
			assert.NotContains(t, msg, "[3]")
			coreErr, ok := resp["error"].(map[string]interface{})
			require.True(t, ok)
			assert.Equal(t, "SESSION_ENTRY_NOT_FOUND", coreErr["reason"])
			assert.Equal(t, "moi-core.session", coreErr["domain"])
			metadata, ok := coreErr["metadata"].(map[string]interface{})
			require.True(t, ok)
			assert.Equal(t, "entry-42", metadata["entry_id"])
		})
	}
}

func TestSemanticModelController_GetModel_CoreSDKErrorWithoutReasonDoesNotLegacyLocalize(t *testing.T) {
	svc := new(MockSemanticModelService)
	router := setupSemanticModelRouter(svc)

	svc.On("GetModel", mock.Anything, 99).Return(nil, &moi.Error{
		Code:    common.ErrorCode_NOT_FOUND,
		Message: "semantic model not found",
	})

	req := httptest.NewRequest(http.MethodGet, "/newmoi/semantic-models/99", nil)
	req.Header.Set("Accept-Language", "zh-CN")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "资源不存在", resp["msg"])
	assert.NotContains(t, resp["msg"], "semantic model not found")
	assert.NotContains(t, resp, "error")
}

func TestSemanticModelController_GetModel_UnknownStructuredCoreReasonUsesSystemMessage(t *testing.T) {
	tests := []struct {
		name           string
		acceptLanguage string
		want           string
	}{
		{
			name:           "en-US",
			acceptLanguage: "en-US",
			want:           "Service temporarily unavailable because the backend error reason is not mapped.",
		},
		{
			name:           "zh-CN",
			acceptLanguage: "zh-CN",
			want:           "服务暂时不可用，后端错误原因尚未映射。",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := new(MockSemanticModelService)
			router := setupSemanticModelRouter(svc)

			svc.On("GetModel", mock.Anything, 99).Return(nil, &moi.Error{
				Code:    common.ErrorCode_NOT_FOUND,
				Message: "raw producer diagnostic",
				Details: common.NewErrorInfoDetails("CORE_UNKNOWN_REASON", "moi-core.session", nil),
			})

			req := httptest.NewRequest(http.MethodGet, "/newmoi/semantic-models/99", nil)
			req.Header.Set("Accept-Language", tt.acceptLanguage)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusNotFound, w.Code)
			var resp map[string]interface{}
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
			msg, ok := resp["msg"].(string)
			require.True(t, ok)
			assert.Equal(t, tt.want, msg)
			assert.NotContains(t, msg, "raw producer diagnostic")
			assert.NotContains(t, msg, "[3]")
			coreErr, ok := resp["error"].(map[string]interface{})
			require.True(t, ok)
			assert.Equal(t, "CORE_UNKNOWN_REASON", coreErr["reason"])
			assert.Equal(t, "moi-core.session", coreErr["domain"])
			metadata, ok := coreErr["metadata"].(map[string]interface{})
			require.True(t, ok)
			assert.Empty(t, metadata)
		})
	}
}

func TestSemanticModelController_GetModel_InternalError(t *testing.T) {
	svc := new(MockSemanticModelService)
	router := setupSemanticModelRouter(svc)

	svc.On("GetModel", mock.Anything, 99).Return(nil, assert.AnError)

	req := httptest.NewRequest(http.MethodGet, "/newmoi/semantic-models/99", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSemanticModelController_CreateModelWithSources_Success(t *testing.T) {
	svc := new(MockSemanticModelService)
	router := setupSemanticModelRouter(svc)

	expected := &session.CreateSemanticModelWithSourcesResponse{
		Model: &session.SemanticModelInfo{
			ID:     7,
			Name:   "kb_docs",
			Tables: json.RawMessage(`[]`),
			Files: json.RawMessage(`{
				"file_ids":[],
				"vector_table":"kb_7_text_index",
				"embedding_model":"bge-m3",
				"image_vector_table":"kb_7_image_index",
				"image_embedding_model":"efficientnet-b3",
				"image_embedding_backend_id":"42",
				"image_embedding_dimension":1536,
				"image_preprocess_version":"efficientnet-b3-v1-rgb-300-letterbox-imagenet",
				"image_distance_metric":"cosine"
			}`),
		},
		DataDomain: &session.KnowledgeBaseDataDomain{
			ModelID:           7,
			CatalogID:         3,
			DatabaseID:        11,
			RawVolumeID:       12,
			ProcessedVolumeID: 13,
			EnsureStatus:      "ready",
		},
		Jobs: []session.KnowledgeBaseSourceJobRun{{
			JobID:     "job-1",
			SourceID:  "source-1",
			ModelID:   7,
			JobType:   "rag_ingest",
			JobStatus: "queued",
		}},
	}
	svc.On("CreateModelWithSources", mock.Anything, mock.MatchedBy(func(req session.CreateSemanticModelWithSourcesRequest) bool {
		return req.Name == "kb_docs" &&
			req.ImageIndexEnabled &&
			len(req.Sources) == 2 &&
			req.Sources[0].SourceType == "local_file" &&
			req.Sources[0].FileName == "a.txt" &&
			req.Sources[0].FileID == "uploaded-file" &&
			req.Sources[1].SourceType == "catalog_file" &&
			req.Sources[1].FileID == "source-file-1" &&
			req.Sources[1].VolumeID == 41
	})).Return(expected, nil)

	body := `{"name":"kb_docs","image_index_enabled":true,"sources":[{"source_type":"local_file","file_name":"a.txt","file_id":"uploaded-file"},{"source_type":"catalog_file","file_id":"source-file-1","volume_id":41}]}`
	req := httptest.NewRequest(http.MethodPost, "/newmoi/semantic-models/create-with-sources", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)
	var resp struct {
		Code string `json:"code"`
		Data struct {
			Model      *session.SemanticModelInfo          `json:"model"`
			DataDomain *session.KnowledgeBaseDataDomain    `json:"data_domain"`
			Jobs       []session.KnowledgeBaseSourceJobRun `json:"jobs"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, "OK", resp.Code)
	require.NotNil(t, resp.Data.Model)
	require.Equal(t, int64(7), resp.Data.Model.ID)
	var files struct {
		EmbeddingModel          string `json:"embedding_model"`
		ImageEmbeddingModel     string `json:"image_embedding_model"`
		ImageEmbeddingBackendID string `json:"image_embedding_backend_id"`
		ImageEmbeddingDimension int    `json:"image_embedding_dimension"`
		ImagePreprocessVersion  string `json:"image_preprocess_version"`
		ImageDistanceMetric     string `json:"image_distance_metric"`
	}
	require.NoError(t, json.Unmarshal(resp.Data.Model.Files, &files))
	require.Equal(t, "bge-m3", files.EmbeddingModel)
	require.Equal(t, "efficientnet-b3", files.ImageEmbeddingModel)
	require.Equal(t, "42", files.ImageEmbeddingBackendID)
	require.Equal(t, 1536, files.ImageEmbeddingDimension)
	require.Equal(t, "efficientnet-b3-v1-rgb-300-letterbox-imagenet", files.ImagePreprocessVersion)
	require.Equal(t, "cosine", files.ImageDistanceMetric)
	require.NotNil(t, resp.Data.DataDomain)
	require.Equal(t, int64(12), resp.Data.DataDomain.RawVolumeID)
	require.Len(t, resp.Data.Jobs, 1)
	svc.AssertExpectations(t)
}

func TestSemanticModelController_CreateEmptyModel_Success(t *testing.T) {
	svc := new(MockSemanticModelService)
	router := setupSemanticModelRouter(svc)

	expected := &session.CreateEmptySemanticModelResponse{
		Model: &session.SemanticModelInfo{ID: 7, Name: "kb_docs", Tables: json.RawMessage(`[]`)},
		DataDomain: &session.KnowledgeBaseDataDomain{
			ModelID: 7, CatalogID: 3, DatabaseID: 11, RawVolumeID: 12, ProcessedVolumeID: 13, EnsureStatus: "ready",
		},
	}
	svc.On("CreateEmptyModel", mock.Anything, session.CreateEmptySemanticModelRequest{
		Name: "kb_docs", Description: "data side", ImageIndexEnabled: true,
	}).Return(expected, nil)

	req := httptest.NewRequest(http.MethodPost, "/newmoi/semantic-models/create-empty", strings.NewReader(`{"name":"kb_docs","description":"data side","image_index_enabled":true}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	var response struct {
		Code string                                    `json:"code"`
		Data *session.CreateEmptySemanticModelResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	require.Equal(t, "OK", response.Code)
	require.NotNil(t, response.Data)
	require.NotNil(t, response.Data.Model)
	require.Equal(t, int64(7), response.Data.Model.ID)
	require.NotNil(t, response.Data.DataDomain)
	require.Equal(t, int64(12), response.Data.DataDomain.RawVolumeID)
	svc.AssertExpectations(t)
}

func TestSemanticModelController_CreateModelWithSourcesRejectsTargetCatalogID(t *testing.T) {
	svc := new(MockSemanticModelService)
	router := setupSemanticModelRouter(svc)

	body := `{"name":"kb_docs","target_catalog_id":99,"sources":[{"source_type":"catalog_file","file_id":"file-1","volume_id":41}]}`
	req := httptest.NewRequest(http.MethodPost, "/newmoi/semantic-models/create-with-sources", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Language", "zh-CN")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	var response struct {
		Code string `json:"code"`
		Msg  string `json:"msg"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	require.Equal(t, "ErrParamInvalid", response.Code)
	require.Equal(t, "target_catalog_id 已不再支持；新建知识库 data domain 始终使用默认 Catalog", response.Msg)
	svc.AssertNotCalled(t, "CreateModelWithSources", mock.Anything, mock.Anything)
}

func TestSemanticModelController_CreateModelWithSources_BindingErrorLocalized(t *testing.T) {
	svc := new(MockSemanticModelService)
	router := setupSemanticModelRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/newmoi/semantic-models/create-with-sources", strings.NewReader(`{`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Language", "zh-CN")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	var body struct {
		Code string `json:"code"`
		Msg  string `json:"msg"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, "ErrParamInvalid", body.Code)
	require.Equal(t, "参数无效", body.Msg)
	require.NotContains(t, body.Msg, "unexpected EOF")
	svc.AssertExpectations(t)
}

func TestSemanticModelController_CreateModelWithSources_EmbeddingCapabilityErrorLocalized(t *testing.T) {
	for _, tc := range []struct {
		language string
		want     string
	}{
		{language: "zh-CN", want: "无法验证知识库所需的向量模型，请稍后重试"},
		{language: "en-US", want: "Unable to verify the embedding models required by the knowledge base. Please try again later"},
	} {
		t.Run(tc.language, func(t *testing.T) {
			svc := new(MockSemanticModelService)
			router := setupSemanticModelRouter(svc)
			svc.On("CreateModelWithSources", mock.Anything, mock.Anything).Return(nil, &session.ServiceError{
				Code: session.ErrCodeInternal,
				Err: i18n.WrapError(
					i18n.KeySessionKnowledgeBaseEmbeddingCapabilityUnavailable,
					fmt.Errorf("secret core response"),
					nil,
				),
			})

			req := httptest.NewRequest(http.MethodPost, "/newmoi/semantic-models/create-with-sources", strings.NewReader(`{"name":"kb_docs","sources":[{"source_type":"catalog_file","file_id":"file-1","volume_id":41}]}`))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Accept-Language", tc.language)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			require.Equal(t, http.StatusInternalServerError, w.Code)
			var body struct {
				Code string `json:"code"`
				Msg  string `json:"msg"`
			}
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
			require.Equal(t, "ErrServer", body.Code)
			require.Equal(t, tc.want, body.Msg)
			require.NotContains(t, body.Msg, "secret core response")
			svc.AssertExpectations(t)
		})
	}
}

func TestSemanticModelController_RejectsDeprecatedContentBase64(t *testing.T) {
	tests := []struct {
		name  string
		path  string
		body  string
		setup func(*MockSemanticModelService)
	}{
		{
			name: "create",
			path: "/newmoi/semantic-models/create-with-sources",
			body: `{"name":"kb_docs","sources":[{"source_type":"local_file","file_name":"a.txt","file_id":"uploaded-file","content_base64":"YQ=="}]}`,
			setup: func(svc *MockSemanticModelService) {
				svc.On("CreateModelWithSources", mock.Anything, mock.MatchedBy(func(req session.CreateSemanticModelWithSourcesRequest) bool {
					return len(req.Sources) == 1 && len(req.Sources[0].DeprecatedContentBase64) > 0
				})).Return(nil, &session.ServiceError{
					Code: session.ErrCodeBadRequest,
					Err:  i18n.NewError(i18n.KeySessionSemanticModelContentBase64Unsupported, map[string]any{"Index": 0}),
				})
			},
		},
		{
			name: "append",
			path: "/newmoi/semantic-models/42/sources",
			body: `{"sources":[{"source_type":"local_file","file_name":"a.txt","file_id":"uploaded-file","content_base64":"YQ=="}]}`,
			setup: func(svc *MockSemanticModelService) {
				svc.On("AppendModelSources", mock.Anything, mock.MatchedBy(func(req session.AppendSemanticModelSourcesRequest) bool {
					return req.ModelID == 42 && len(req.Sources) == 1 && len(req.Sources[0].DeprecatedContentBase64) > 0
				})).Return(nil, &session.ServiceError{
					Code: session.ErrCodeBadRequest,
					Err:  i18n.NewError(i18n.KeySessionSemanticModelContentBase64Unsupported, map[string]any{"Index": 0}),
				})
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := new(MockSemanticModelService)
			tc.setup(svc)
			router := setupSemanticModelRouter(svc)
			req := httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Accept-Language", "zh-CN")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
			var body struct {
				Code string `json:"code"`
				Msg  string `json:"msg"`
			}
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
			require.Equal(t, "ErrParamInvalid", body.Code)
			require.Equal(t, "sources[0].content_base64 已不再支持，请先上传文件并使用 file_id", body.Msg)
			svc.AssertExpectations(t)
		})
	}
}

func TestSemanticModelController_CreateModelWithSourcesSelectionOnlyAuthorizesSelectionDependencies(t *testing.T) {
	svc := new(MockSemanticModelService)
	var actions []string
	router := setupSemanticModelRouterWithIAM(svc, semanticModelIAMClientFunc(func(_ context.Context, req *iampb.AuthorizeRequest) (*iampb.AuthorizeDecision, error) {
		actions = append(actions, req.GetActionId())
		return semanticModelDecision(req, iampb.AuthorizeDecisionKind_AUTHORIZE_DECISION_KIND_ALLOW), nil
	}))

	expected := &session.CreateSemanticModelWithSourcesResponse{Model: &session.SemanticModelInfo{ID: 7, Name: "kb_from_catalog"}}
	svc.On("CreateModelWithSources", mock.Anything, mock.MatchedBy(func(req session.CreateSemanticModelWithSourcesRequest) bool {
		return req.Name == "kb_from_catalog" &&
			len(req.Sources) == 0 &&
			len(req.SourceSelections) == 1 &&
			req.SourceSelections[0].Kind == "database_tables" &&
			req.SourceSelections[0].DatabaseID == 3 &&
			len(req.SourceSelections[0].SelectedTableIDs) == 1 &&
			req.SourceSelections[0].SelectedTableIDs[0] == 1
	})).Return(expected, nil)

	body := `{"name":"kb_from_catalog","sources":[],"source_selections":[{"kind":"database_tables","database_id":3,"all_selected":false,"selected_table_ids":[1]}]}`
	req := httptest.NewRequest(http.MethodPost, "/newmoi/semantic-models/create-with-sources", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	require.Equal(t, []string{"semantic_model.create", "database.read"}, actions)
	svc.AssertExpectations(t)
}

// TestSemanticModelController_CreateModelWithSourcesSelectionOnlyRealResolver pins
// the #13918 regression in its production shape: the real resolver, unlike the
// test stub, hard-fails on empty sources, so a selection-only request must be
// authorized via database.read and reach the handler instead of failing with
// ErrIAMRequestInvalid.
func TestSemanticModelController_CreateModelWithSourcesSelectionOnlyRealResolver(t *testing.T) {
	svc := new(MockSemanticModelService)
	var actions []string
	router := setupSemanticModelRouterWithIAMAndDependencies(svc, semanticModelIAMClientFunc(func(_ context.Context, req *iampb.AuthorizeRequest) (*iampb.AuthorizeDecision, error) {
		actions = append(actions, req.GetActionId())
		return semanticModelDecision(req, iampb.AuthorizeDecisionKind_AUTHORIZE_DECISION_KIND_ALLOW), nil
	}), &CoreSemanticModelDependencyResolver{})

	expected := &session.CreateSemanticModelWithSourcesResponse{Model: &session.SemanticModelInfo{ID: 7, Name: "kb_real_resolver"}}
	svc.On("CreateModelWithSources", mock.Anything, mock.MatchedBy(func(req session.CreateSemanticModelWithSourcesRequest) bool {
		return req.Name == "kb_real_resolver" && len(req.Sources) == 0 &&
			len(req.SourceSelections) == 1 && req.SourceSelections[0].Kind == "database_tables" &&
			req.SourceSelections[0].DatabaseID == 3
	})).Return(expected, nil)

	req := httptest.NewRequest(http.MethodPost, "/newmoi/semantic-models/create-with-sources", strings.NewReader(`{"name":"kb_real_resolver","sources":[],"source_selections":[{"kind":"database_tables","database_id":3,"all_selected":false,"selected_table_ids":[1]}]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	require.Equal(t, []string{"semantic_model.create", "database.read"}, actions)
	svc.AssertExpectations(t)
}

func TestSemanticModelController_CreateModelWithSourcesAcceptsTwoUploadedLocalFiles(t *testing.T) {
	svc := new(MockSemanticModelService)
	var actions []string
	router := setupSemanticModelRouterWithIAMAndDependencies(svc, semanticModelIAMClientFunc(func(_ context.Context, req *iampb.AuthorizeRequest) (*iampb.AuthorizeDecision, error) {
		actions = append(actions, req.GetActionId())
		return semanticModelDecision(req, iampb.AuthorizeDecisionKind_AUTHORIZE_DECISION_KIND_ALLOW), nil
	}), &CoreSemanticModelDependencyResolver{})

	expected := &session.CreateSemanticModelWithSourcesResponse{Model: &session.SemanticModelInfo{ID: 7, Name: "kb_two_local_files"}}
	svc.On("CreateModelWithSources", mock.Anything, mock.MatchedBy(func(req session.CreateSemanticModelWithSourcesRequest) bool {
		return req.Name == "kb_two_local_files" && len(req.Sources) == 2 &&
			req.Sources[0].SourceType == semanticSourceLocalFile && req.Sources[0].FileName == "first.txt" && req.Sources[0].FileID == "uploaded-file-1" &&
			req.Sources[1].SourceType == semanticSourceLocalFile && req.Sources[1].FileName == "second.txt" && req.Sources[1].FileID == "uploaded-file-2"
	})).Return(expected, nil)

	req := httptest.NewRequest(http.MethodPost, "/newmoi/semantic-models/create-with-sources", strings.NewReader(`{"name":"kb_two_local_files","sources":[{"source_type":"local_file","file_name":"first.txt","file_id":"uploaded-file-1"},{"source_type":"local_file","file_name":"second.txt","file_id":"uploaded-file-2"}]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	require.Equal(t, []string{"semantic_model.create"}, actions)
	svc.AssertExpectations(t)
}

func TestSemanticModelController_CreateModelWithSourcesRejectsIncompleteLocalFileAsParamError(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
	}{
		{
			name: "missing file id",
			body: `{"name":"kb_missing_file_id","sources":[{"source_type":"local_file","file_name":"first.txt"}]}`,
		},
		{
			name: "blank file name",
			body: `{"name":"kb_blank_file_name","sources":[{"source_type":"local_file","file_name":" ","file_id":"uploaded-file-1"}]}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc := new(MockSemanticModelService)
			var actions []string
			router := setupSemanticModelRouterWithIAMAndDependencies(svc, semanticModelIAMClientFunc(func(_ context.Context, req *iampb.AuthorizeRequest) (*iampb.AuthorizeDecision, error) {
				actions = append(actions, req.GetActionId())
				return semanticModelDecision(req, iampb.AuthorizeDecisionKind_AUTHORIZE_DECISION_KIND_ALLOW), nil
			}), &CoreSemanticModelDependencyResolver{})

			req := httptest.NewRequest(http.MethodPost, "/newmoi/semantic-models/create-with-sources", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Accept-Language", "zh-CN")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
			var response struct {
				Code string `json:"code"`
				Msg  string `json:"msg"`
			}
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
			require.Equal(t, "ErrParamInvalid", response.Code)
			require.Equal(t, "local_file 来源必须提供非空的 file_name 和 file_id", response.Msg)
			require.NotEmpty(t, w.Header().Get(iampep.HeaderRequestID))
			require.Equal(t, []string{"semantic_model.create"}, actions)
			svc.AssertNotCalled(t, "CreateModelWithSources", mock.Anything, mock.Anything)
		})
	}
}

func TestSemanticModelController_CreateModelWithSourcesDirectSourcesKeepDependencyAuthorization(t *testing.T) {
	svc := new(MockSemanticModelService)
	var actions []string
	router := setupSemanticModelRouterWithIAM(svc, semanticModelIAMClientFunc(func(_ context.Context, req *iampb.AuthorizeRequest) (*iampb.AuthorizeDecision, error) {
		actions = append(actions, req.GetActionId())
		return semanticModelDecision(req, iampb.AuthorizeDecisionKind_AUTHORIZE_DECISION_KIND_ALLOW), nil
	}))

	expected := &session.CreateSemanticModelWithSourcesResponse{Model: &session.SemanticModelInfo{ID: 7, Name: "kb_tables"}}
	svc.On("CreateModelWithSources", mock.Anything, mock.MatchedBy(func(req session.CreateSemanticModelWithSourcesRequest) bool {
		return req.Name == "kb_tables" &&
			len(req.Sources) == 1 &&
			req.Sources[0].SourceType == "catalog_table" &&
			req.Sources[0].TableID == 9
	})).Return(expected, nil)

	req := httptest.NewRequest(http.MethodPost, "/newmoi/semantic-models/create-with-sources", strings.NewReader(`{"name":"kb_tables","sources":[{"source_type":"catalog_table","table_id":9}]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	require.Equal(t, []string{"semantic_model.create", "table.read"}, actions)
	svc.AssertExpectations(t)
}

func TestSemanticModelController_CreateModelWithSourcesMixedSourcesAndSelectionsDeduplicate(t *testing.T) {
	svc := new(MockSemanticModelService)
	var actions []string
	router := setupSemanticModelRouterWithIAM(svc, semanticModelIAMClientFunc(func(_ context.Context, req *iampb.AuthorizeRequest) (*iampb.AuthorizeDecision, error) {
		actions = append(actions, req.GetActionId())
		return semanticModelDecision(req, iampb.AuthorizeDecisionKind_AUTHORIZE_DECISION_KIND_ALLOW), nil
	}))

	expected := &session.CreateSemanticModelWithSourcesResponse{Model: &session.SemanticModelInfo{ID: 7, Name: "kb_mixed"}}
	svc.On("CreateModelWithSources", mock.Anything, mock.MatchedBy(func(req session.CreateSemanticModelWithSourcesRequest) bool {
		return req.Name == "kb_mixed" &&
			len(req.Sources) == 1 &&
			req.Sources[0].SourceType == "catalog_file" &&
			req.Sources[0].FileID == "file-1" &&
			req.Sources[0].VolumeID == 41 &&
			len(req.SourceSelections) == 1 &&
			req.SourceSelections[0].Kind == "volume_files" &&
			req.SourceSelections[0].VolumeID == 41
	})).Return(expected, nil)

	// Same volume_id on direct catalog_file and volume_files selection so
	// RequireDependencies deduplicates to a single volume.read (production
	// also dedupes after root resolution when both map to one root).
	req := httptest.NewRequest(http.MethodPost, "/newmoi/semantic-models/create-with-sources", strings.NewReader(`{"name":"kb_mixed","sources":[{"source_type":"catalog_file","file_id":"file-1","volume_id":41}],"source_selections":[{"kind":"volume_files","volume_id":41,"all_selected":true}]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	require.Equal(t, []string{"semantic_model.create", "volume.read"}, actions)
	svc.AssertExpectations(t)
}

func TestSemanticModelController_CreateModelWithSourcesCatalogFileRequiresVolumeIDBeforeService(t *testing.T) {
	svc := new(MockSemanticModelService)
	var actions []string
	router := setupSemanticModelRouterWithIAM(svc, semanticModelIAMClientFunc(func(_ context.Context, req *iampb.AuthorizeRequest) (*iampb.AuthorizeDecision, error) {
		actions = append(actions, req.GetActionId())
		return semanticModelDecision(req, iampb.AuthorizeDecisionKind_AUTHORIZE_DECISION_KIND_ALLOW), nil
	}))

	req := httptest.NewRequest(http.MethodPost, "/newmoi/semantic-models/create-with-sources", strings.NewReader(`{"name":"kb_no_volume","sources":[{"source_type":"catalog_file","file_id":"file-1"}]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Dependency extraction fails closed before any volume.read authorize call.
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	require.Equal(t, []string{"semantic_model.create"}, actions)
	svc.AssertNotCalled(t, "CreateModelWithSources", mock.Anything, mock.Anything)
}

func TestSemanticModelController_CreateModelWithSourcesMultiVolumeCatalogFilesAuthorizeDistinctRoots(t *testing.T) {
	svc := new(MockSemanticModelService)
	var actions []string
	var volumeResourceIDs []string
	router := setupSemanticModelRouterWithIAM(svc, semanticModelIAMClientFunc(func(_ context.Context, req *iampb.AuthorizeRequest) (*iampb.AuthorizeDecision, error) {
		actions = append(actions, req.GetActionId())
		if req.GetActionId() == "volume.read" && req.GetResource() != nil {
			volumeResourceIDs = append(volumeResourceIDs, req.GetResource().GetResourceId())
		}
		return semanticModelDecision(req, iampb.AuthorizeDecisionKind_AUTHORIZE_DECISION_KIND_ALLOW), nil
	}))

	expected := &session.CreateSemanticModelWithSourcesResponse{Model: &session.SemanticModelInfo{ID: 8, Name: "kb_multi_vol"}}
	svc.On("CreateModelWithSources", mock.Anything, mock.MatchedBy(func(req session.CreateSemanticModelWithSourcesRequest) bool {
		return req.Name == "kb_multi_vol" &&
			len(req.Sources) == 2 &&
			req.Sources[0].SourceType == "catalog_file" && req.Sources[0].FileID == "shared-file" && req.Sources[0].VolumeID == 41 &&
			req.Sources[1].SourceType == "catalog_file" && req.Sources[1].FileID == "shared-file" && req.Sources[1].VolumeID == 52
	})).Return(expected, nil)

	req := httptest.NewRequest(http.MethodPost, "/newmoi/semantic-models/create-with-sources", strings.NewReader(`{"name":"kb_multi_vol","sources":[{"source_type":"catalog_file","file_id":"shared-file","volume_id":41},{"source_type":"catalog_file","file_id":"shared-file","volume_id":52}]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	require.Equal(t, []string{"semantic_model.create", "volume.read", "volume.read"}, actions)
	require.Equal(t, []string{"41", "52"}, volumeResourceIDs)
	svc.AssertExpectations(t)
}

func TestSemanticModelController_CreateModelWithSourcesEmptySourcesAndSelectionsRejectedBeforeService(t *testing.T) {
	svc := new(MockSemanticModelService)
	router := setupSemanticModelRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/newmoi/semantic-models/create-with-sources", strings.NewReader(`{"name":"kb_empty","sources":[]}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Language", "zh-CN")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	var body struct {
		Code string `json:"code"`
		Msg  string `json:"msg"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, "ErrParamInvalid", body.Code)
	require.Equal(t, "参数无效", body.Msg)
	svc.AssertNotCalled(t, "CreateModelWithSources", mock.Anything, mock.Anything)
}

func TestSemanticModelController_CreateModelWithSourcesSelectionDependencyDeniedBeforeService(t *testing.T) {
	svc := new(MockSemanticModelService)
	var actions []string
	router := setupSemanticModelRouterWithIAM(svc, semanticModelIAMClientFunc(func(_ context.Context, req *iampb.AuthorizeRequest) (*iampb.AuthorizeDecision, error) {
		actions = append(actions, req.GetActionId())
		decision := iampb.AuthorizeDecisionKind_AUTHORIZE_DECISION_KIND_ALLOW
		if req.GetActionId() == "database.read" {
			decision = iampb.AuthorizeDecisionKind_AUTHORIZE_DECISION_KIND_DENY
		}
		return semanticModelDecision(req, decision), nil
	}))

	req := httptest.NewRequest(http.MethodPost, "/newmoi/semantic-models/create-with-sources", strings.NewReader(`{"name":"kb_denied","sources":[],"source_selections":[{"kind":"database_tables","database_id":3,"all_selected":true}]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusForbidden, w.Code, w.Body.String())
	require.Equal(t, []string{"semantic_model.create", "database.read"}, actions)
	svc.AssertNotCalled(t, "CreateModelWithSources", mock.Anything, mock.Anything)
}

func TestSemanticModelController_AppendSourcesSelectionOnlyAuthorizesSelectionDependencies(t *testing.T) {
	svc := new(MockSemanticModelService)
	var actions []string
	router := setupSemanticModelRouterWithIAM(svc, semanticModelIAMClientFunc(func(_ context.Context, req *iampb.AuthorizeRequest) (*iampb.AuthorizeDecision, error) {
		actions = append(actions, req.GetActionId())
		return semanticModelDecision(req, iampb.AuthorizeDecisionKind_AUTHORIZE_DECISION_KIND_ALLOW), nil
	}))

	expected := &session.AppendSemanticModelSourcesResponse{Sources: []session.SemanticModelSource{{SourceID: "source-1", ModelID: 42}}}
	svc.On("AppendModelSources", mock.Anything, mock.MatchedBy(func(req session.AppendSemanticModelSourcesRequest) bool {
		return req.ModelID == 42 &&
			len(req.Sources) == 0 &&
			len(req.SourceSelections) == 1 &&
			req.SourceSelections[0].Kind == "volume_files" &&
			req.SourceSelections[0].VolumeID == 11
	})).Return(expected, nil)

	req := httptest.NewRequest(http.MethodPost, "/newmoi/semantic-models/42/sources", strings.NewReader(`{"sources":[],"source_selections":[{"kind":"volume_files","volume_id":11,"all_selected":true}]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Equal(t, []string{"semantic_model.use", "semantic_model.update", "volume.read"}, actions)
	svc.AssertExpectations(t)
}

func TestSemanticModelController_AppendSources_ServiceErrorLocalized(t *testing.T) {
	svc := new(MockSemanticModelService)
	router := setupSemanticModelRouter(svc)

	svc.On("AppendModelSources", mock.Anything, mock.MatchedBy(func(req session.AppendSemanticModelSourcesRequest) bool {
		return req.ModelID == 42 && len(req.Sources) == 1 && req.Sources[0].VolumeID == 41
	})).Return(nil, &session.ServiceError{
		Code: session.ErrCodeBadRequest,
		Err:  i18n.NewError(i18n.KeySessionSemanticModelSourcesRequired, nil),
	})

	req := httptest.NewRequest(http.MethodPost, "/newmoi/semantic-models/42/sources", strings.NewReader(`{"sources":[{"source_type":"catalog_file","file_id":"file-1","volume_id":41}]}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Language", "zh-CN")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	var body struct {
		Code string `json:"code"`
		Msg  string `json:"msg"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, "ErrParamInvalid", body.Code)
	require.Equal(t, "至少需要一个来源", body.Msg)
	require.NotContains(t, body.Msg, "at least one source is required")
	svc.AssertExpectations(t)
}

func TestSemanticModelController_ListSources_Success(t *testing.T) {
	svc := new(MockSemanticModelService)
	router := setupSemanticModelRouter(svc)

	displayName := "orders"
	dbName := "sales_db"
	tableName := "orders"
	ingestStatus := "unsupported"
	expected := &session.ListSemanticModelSourcesResult{
		Items: []session.SemanticModelSource{
			{
				RowID:        "42:table:sales_db::orders",
				SourceType:   session.SemanticModelSourceTypeTable,
				ModelID:      42,
				ResourceID:   "sales_db::orders",
				DisplayName:  &displayName,
				Path:         []string{"sales_db", "orders"},
				DBName:       &dbName,
				TableName:    &tableName,
				IngestStatus: &ingestStatus,
			},
		},
		Total: 1,
	}
	svc.On("ListSources", mock.Anything, session.ListSemanticModelSourcesParams{ModelID: 42}).Return(expected, nil)

	req := httptest.NewRequest(http.MethodGet, "/newmoi/semantic-models/42/sources", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var body struct {
		Code string `json:"code"`
		Data struct {
			Items []session.SemanticModelSource `json:"items"`
			Total int                           `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, "OK", body.Code)
	require.Equal(t, 1, body.Data.Total)
	require.Len(t, body.Data.Items, 1)
	assert.Equal(t, "42:table:sales_db::orders", body.Data.Items[0].RowID)
	assert.Equal(t, session.SemanticModelSourceTypeTable, body.Data.Items[0].SourceType)
	assert.Equal(t, int64(42), body.Data.Items[0].ModelID)
	assert.Equal(t, "sales_db::orders", body.Data.Items[0].ResourceID)
	require.NotNil(t, body.Data.Items[0].DisplayName)
	assert.Equal(t, "orders", *body.Data.Items[0].DisplayName)
	require.NotNil(t, body.Data.Items[0].DBName)
	assert.Equal(t, "sales_db", *body.Data.Items[0].DBName)
	require.NotNil(t, body.Data.Items[0].TableName)
	assert.Equal(t, "orders", *body.Data.Items[0].TableName)
	require.NotNil(t, body.Data.Items[0].IngestStatus)
	assert.Equal(t, "unsupported", *body.Data.Items[0].IngestStatus)
	assert.Nil(t, body.Data.Items[0].Enabled)
	assert.Nil(t, body.Data.Items[0].ExpiresAt)
	assert.Nil(t, body.Data.Items[0].SegmentVersionID)
	assert.Nil(t, body.Data.Items[0].Error)
	svc.AssertExpectations(t)
}

func TestSemanticModelController_ListSources_ForwardsPagination(t *testing.T) {
	svc := new(MockSemanticModelService)
	router := setupSemanticModelRouter(svc)
	svc.On("ListSources", mock.Anything, session.ListSemanticModelSourcesParams{ModelID: 42, Page: 2, PageSize: 10}).
		Return(&session.ListSemanticModelSourcesResult{Items: []session.SemanticModelSource{}, Total: 12, Page: 2, PageSize: 10}, nil)

	req := httptest.NewRequest(http.MethodGet, "/newmoi/semantic-models/42/sources?page=2&page_size=10", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	svc.AssertExpectations(t)
}

func TestSemanticModelController_ListSources_RejectsInvalidPagination(t *testing.T) {
	for _, rawQuery := range []string{"page=0", "page_size=101"} {
		t.Run(rawQuery, func(t *testing.T) {
			svc := new(MockSemanticModelService)
			router := setupSemanticModelRouter(svc)
			req := httptest.NewRequest(http.MethodGet, "/newmoi/semantic-models/42/sources?"+rawQuery, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			require.Equal(t, http.StatusBadRequest, w.Code)
			svc.AssertNotCalled(t, "ListSources", mock.Anything, mock.Anything)
		})
	}
}

func TestSemanticModelController_AppendSources_Success(t *testing.T) {
	svc := new(MockSemanticModelService)
	router := setupSemanticModelRouter(svc)

	expected := &session.AppendSemanticModelSourcesResponse{
		DataDomain: &session.KnowledgeBaseDataDomain{
			ModelID:           42,
			CatalogID:         3,
			DatabaseID:        11,
			RawVolumeID:       12,
			ProcessedVolumeID: 13,
			EnsureStatus:      "ready",
		},
		Sources: []session.SemanticModelSource{{
			SourceID:   "source-1",
			SourceType: session.SemanticModelSourceTypeFile,
			ModelID:    42,
			ResourceID: "kb-file-1",
		}},
		Jobs: []session.KnowledgeBaseSourceJobRun{{
			JobID:     "job-1",
			SourceID:  "source-1",
			ModelID:   42,
			JobType:   "rag_ingest",
			JobStatus: "queued",
		}},
	}
	svc.On("AppendModelSources", mock.Anything, mock.MatchedBy(func(req session.AppendSemanticModelSourcesRequest) bool {
		return req.ModelID == 42 &&
			len(req.Sources) == 1 &&
			req.Sources[0].SourceType == "catalog_file" &&
			req.Sources[0].FileID == "source-file-1" &&
			req.Sources[0].VolumeID == 41
	})).Return(expected, nil)

	req := httptest.NewRequest(http.MethodPost, "/newmoi/semantic-models/42/sources", strings.NewReader(`{"sources":[{"source_type":"catalog_file","file_id":"source-file-1","volume_id":41}]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var body struct {
		Code string `json:"code"`
		Data struct {
			DataDomain *session.KnowledgeBaseDataDomain    `json:"data_domain"`
			Sources    []session.SemanticModelSource       `json:"sources"`
			Jobs       []session.KnowledgeBaseSourceJobRun `json:"jobs"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, "OK", body.Code)
	require.NotNil(t, body.Data.DataDomain)
	require.Equal(t, int64(42), body.Data.DataDomain.ModelID)
	require.Len(t, body.Data.Sources, 1)
	require.Len(t, body.Data.Jobs, 1)
	svc.AssertExpectations(t)
}

func TestSemanticModelController_ListSources_NotFound(t *testing.T) {
	svc := new(MockSemanticModelService)
	router := setupSemanticModelRouter(svc)

	svc.On("ListSources", mock.Anything, session.ListSemanticModelSourcesParams{ModelID: 99}).
		Return(nil, &session.ServiceError{Code: session.ErrCodeNotFound, Msg: "semantic model not found"})

	req := httptest.NewRequest(http.MethodGet, "/newmoi/semantic-models/99/sources", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	svc.AssertExpectations(t)
}

func TestSemanticModelController_GetSourceDocument_Success(t *testing.T) {
	svc := new(MockSemanticModelService)
	var actions []string
	router := setupSemanticModelRouterWithIAMAndDependencies(svc, semanticModelIAMClientFunc(func(_ context.Context, req *iampb.AuthorizeRequest) (*iampb.AuthorizeDecision, error) {
		actions = append(actions, req.GetActionId())
		return semanticModelDecision(req, iampb.AuthorizeDecisionKind_AUTHORIZE_DECISION_KIND_ALLOW), nil
	}), nil)

	expected := &session.SemanticModelSourceDocument{
		Source: session.SemanticModelSource{
			RowID:            "source-file-1",
			SourceType:       session.SemanticModelSourceTypeFile,
			ModelID:          42,
			ResourceID:       "kb-file-1",
			Tags:             []string{"finance"},
			EffectiveEnabled: true,
		},
		Preview: session.SemanticModelSourcePreview{
			Available: false,
		},
		FileInfo: session.SemanticModelSourceFileInfo{
			Tags:             []string{"finance"},
			EffectiveEnabled: true,
		},
		SegmentStatus: session.SemanticModelSegmentStatus{Available: false},
	}
	svc.On("GetSourceDocument", mock.Anything, session.GetSemanticModelSourceDocumentParams{ModelID: 42, SourceID: "source-file-1"}).Return(expected, nil)

	req := httptest.NewRequest(http.MethodGet, "/newmoi/semantic-models/42/sources/source-file-1/document", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var body struct {
		Code string                              `json:"code"`
		Data session.SemanticModelSourceDocument `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, "OK", body.Code)
	require.False(t, body.Data.Preview.Available)
	require.Nil(t, body.Data.Preview.Content)
	require.Nil(t, body.Data.Preview.Reason)
	require.Nil(t, body.Data.SegmentStatus.Reason)
	require.NotContains(t, w.Body.String(), `"reason"`)
	require.Equal(t, []string{"finance"}, body.Data.Source.Tags)
	require.Equal(t, []string{"semantic_model.read"}, actions)
	svc.AssertExpectations(t)
}

func TestSemanticModelController_GetSourceDocument_DeniedBeforeService(t *testing.T) {
	svc := new(MockSemanticModelService)
	var actions []string
	router := setupSemanticModelRouterWithIAMAndDependencies(svc, semanticModelIAMClientFunc(func(_ context.Context, req *iampb.AuthorizeRequest) (*iampb.AuthorizeDecision, error) {
		actions = append(actions, req.GetActionId())
		return semanticModelDecision(req, iampb.AuthorizeDecisionKind_AUTHORIZE_DECISION_KIND_DENY), nil
	}), nil)

	req := httptest.NewRequest(http.MethodGet, "/newmoi/semantic-models/42/sources/source-file-1/document", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusForbidden, w.Code, w.Body.String())
	require.Equal(t, []string{"semantic_model.read"}, actions)
	svc.AssertNotCalled(t, "GetSourceDocument", mock.Anything, mock.Anything)
}

func TestSemanticModelController_GetSourceDocument_RejectsBlankSourceID(t *testing.T) {
	svc := new(MockSemanticModelService)
	router := setupSemanticModelRouterWithIAMAndDependencies(svc, semanticModelAllowIAMClient{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/newmoi/semantic-models/42/sources/%20/document", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	svc.AssertNotCalled(t, "GetSourceDocument", mock.Anything, mock.Anything)
}

func TestSemanticModelController_GetSourceDocument_PreservesNonBlankSourceID(t *testing.T) {
	svc := new(MockSemanticModelService)
	router := setupSemanticModelRouterWithIAMAndDependencies(svc, semanticModelAllowIAMClient{}, nil)
	expectedParams := session.GetSemanticModelSourceDocumentParams{ModelID: 42, SourceID: " source-file-1 "}
	svc.On("GetSourceDocument", mock.Anything, expectedParams).Return(&session.SemanticModelSourceDocument{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/newmoi/semantic-models/42/sources/%20source-file-1%20/document", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	svc.AssertExpectations(t)
}

func TestSemanticModelController_UpdateSourceGovernance_BindsPatchFields(t *testing.T) {
	svc := new(MockSemanticModelService)
	router := setupSemanticModelRouter(svc)

	expected := &session.UpdateSemanticModelSourceGovernanceResult{
		Source: session.SemanticModelSource{
			RowID:        "source-file-1",
			SourceType:   session.SemanticModelSourceTypeFile,
			ModelID:      42,
			ResourceID:   "kb-file-1",
			Tags:         []string{"finance", "policy"},
			ForceEnabled: true,
		},
	}
	svc.On("UpdateSourceGovernance", mock.Anything, mock.MatchedBy(func(params session.UpdateSemanticModelSourceGovernanceParams) bool {
		return params.ModelID == 42 &&
			params.SourceID == "source-file-1" &&
			params.Tags != nil &&
			assert.ObjectsAreEqual([]string{"finance", "policy"}, *params.Tags) &&
			params.ExpiresAt.Set &&
			params.ExpiresAt.Value == nil &&
			params.Enabled != nil &&
			!*params.Enabled &&
			params.ForceEnabledAfterExpiry != nil &&
			*params.ForceEnabledAfterExpiry
	})).Return(expected, nil)

	req := httptest.NewRequest(http.MethodPatch, "/newmoi/semantic-models/42/sources/source-file-1/governance", strings.NewReader(`{"tags":["finance","policy"],"expires_at":null,"enabled":false,"force_enabled_after_expiry":true}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var body struct {
		Code string                                            `json:"code"`
		Data session.UpdateSemanticModelSourceGovernanceResult `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, "OK", body.Code)
	require.True(t, body.Data.Source.ForceEnabled)
	require.Equal(t, []string{"finance", "policy"}, body.Data.Source.Tags)
	svc.AssertExpectations(t)
}

func TestSemanticModelController_DeleteSource_BindsPathParams(t *testing.T) {
	svc := new(MockSemanticModelService)
	router := setupSemanticModelRouter(svc)

	svc.On("DeleteSource", mock.Anything, session.DeleteSemanticModelSourceParams{
		ModelID:  42,
		SourceID: "source-file-1",
	}).Return(nil)

	req := httptest.NewRequest(http.MethodDelete, "/newmoi/semantic-models/42/sources/source-file-1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var body struct {
		Code string                   `json:"code"`
		Data session.MutationResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, "OK", body.Code)
	require.True(t, body.Data.Deleted)
	svc.AssertExpectations(t)
}

func TestSemanticModelController_CreateSegmentKeepsCallerArtifactIDsVisibleForExplicitRejection(t *testing.T) {
	svc := new(MockSemanticModelService)
	router := setupSemanticModelRouter(svc)

	var received session.CreateSemanticModelSegmentParams
	svc.On("CreateSegment", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			received = args.Get(1).(session.CreateSemanticModelSegmentParams)
		}).
		Return(nil, &session.ServiceError{Code: session.ErrCodeBadRequest, Msg: "artifact identity is parser-owned"})

	req := httptest.NewRequest(
		http.MethodPost,
		"/newmoi/semantic-models/42/sources/source-file-1/segments",
		strings.NewReader(`{
			"base_segment_version_id":"segment-v1",
			"base_index_version":4,
			"level":"chunk",
			"content":"caller-authored text",
			"image_file_id":"artifact-from-another-model",
			"page_image_file_id":"page-artifact-from-another-model"
		}`),
	)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	require.Equal(t, 42, received.ModelID)
	require.Equal(t, "source-file-1", received.SourceID)
	require.NotNil(t, received.ImageFileID)
	require.NotNil(t, received.PageImageFileID)
	require.Equal(t, "artifact-from-another-model", *received.ImageFileID)
	require.Equal(t, "page-artifact-from-another-model", *received.PageImageFileID)
	svc.AssertExpectations(t)
}

func TestSemanticModelController_DeleteSegment_BindsPathAndBase(t *testing.T) {
	svc := new(MockSemanticModelService)
	router := setupSemanticModelRouter(svc)

	baseVersionID := "segment-v1"
	baseIndexVersion := int64(4)
	expected := &session.SemanticModelSegmentMutationResult{
		Document: session.SemanticModelSourceDocument{
			Source: session.SemanticModelSource{
				RowID:      "source-file-1",
				SourceType: session.SemanticModelSourceTypeFile,
				ModelID:    42,
				ResourceID: "kb-file-1",
			},
			SegmentStatus:           session.SemanticModelSegmentStatus{Available: true},
			CurrentSegmentVersionID: &baseVersionID,
			CurrentIndexVersion:     &baseIndexVersion,
		},
	}
	svc.On("DeleteSegment", mock.Anything, session.DeleteSemanticModelSegmentParams{
		ModelID:   42,
		SourceID:  "source-file-1",
		SegmentID: "segment-1",
		SemanticModelSegmentMutationBase: session.SemanticModelSegmentMutationBase{
			BaseSegmentVersionID: &baseVersionID,
			BaseIndexVersion:     &baseIndexVersion,
		},
	}).Return(expected, nil)

	req := httptest.NewRequest(http.MethodDelete, "/newmoi/semantic-models/42/sources/source-file-1/segments/segment-1", strings.NewReader(`{"base_segment_version_id":"segment-v1","base_index_version":4}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var body struct {
		Code string                                     `json:"code"`
		Data session.SemanticModelSegmentMutationResult `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, "OK", body.Code)
	require.Equal(t, "source-file-1", body.Data.Document.Source.RowID)
	svc.AssertExpectations(t)
}

func TestSemanticModelController_ListSourceJobs_Success(t *testing.T) {
	svc := new(MockSemanticModelService)
	router := setupSemanticModelRouter(svc)

	expected := &session.ListSemanticModelSourceJobsResult{
		Items: []session.KnowledgeBaseSourceJobView{{
			JobID:     "job-1",
			SourceID:  "source-1",
			JobStatus: "queued",
			KBFileID:  testStringPtr("kb-file-1"),
		}},
		Total: 1,
	}
	svc.On("ListSourceJobs", mock.Anything, session.ListSemanticModelSourceJobsParams{ModelID: 42}).Return(expected, nil)

	req := httptest.NewRequest(http.MethodGet, "/newmoi/semantic-models/42/source-jobs", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var body struct {
		Code string `json:"code"`
		Data struct {
			Items []session.KnowledgeBaseSourceJobView `json:"items"`
			Total int                                  `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, "OK", body.Code)
	require.Equal(t, 1, body.Data.Total)
	require.Len(t, body.Data.Items, 1)
	require.Equal(t, "job-1", body.Data.Items[0].JobID)
	require.Equal(t, "queued", body.Data.Items[0].JobStatus)
	require.NotNil(t, body.Data.Items[0].KBFileID)
	require.Equal(t, "kb-file-1", *body.Data.Items[0].KBFileID)
	require.NotContains(t, w.Body.String(), "job_type")
	require.NotContains(t, w.Body.String(), "idempotency_key")
	require.NotContains(t, w.Body.String(), "workflow_execution_id")
	svc.AssertExpectations(t)
}

func TestSemanticModelController_ReconcileSourceJobs_Success(t *testing.T) {
	svc := new(MockSemanticModelService)
	router := setupSemanticModelRouter(svc)

	svc.On("ReconcileKnowledgeBaseSourceJobs", mock.Anything, session.ReconcileKnowledgeBaseSourceJobsParams{ModelID: 42}).Return(nil)

	req := httptest.NewRequest(http.MethodPost, "/newmoi/semantic-models/42/source-jobs/reconcile", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var body struct {
		Code string                   `json:"code"`
		Data session.MutationResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, "OK", body.Code)
	require.True(t, body.Data.Updated)
	svc.AssertExpectations(t)
}

func TestSemanticModelController_BackfillLegacySources_Success(t *testing.T) {
	svc := new(MockSemanticModelService)
	router := setupSemanticModelRouter(svc)

	svc.On("BackfillLegacySources", mock.Anything, session.BackfillLegacyKnowledgeBaseSourcesParams{ModelID: 42}).Return(nil)

	req := httptest.NewRequest(http.MethodPost, "/newmoi/semantic-models/42/sources/backfill-legacy", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var body struct {
		Code string                   `json:"code"`
		Data session.MutationResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, "OK", body.Code)
	require.True(t, body.Data.Updated)
	svc.AssertExpectations(t)
}

func TestSemanticModelController_BackfillLegacySources_Error(t *testing.T) {
	svc := new(MockSemanticModelService)
	router := setupSemanticModelRouter(svc)

	svc.On("BackfillLegacySources", mock.Anything, session.BackfillLegacyKnowledgeBaseSourcesParams{ModelID: 42}).
		Return(&session.ServiceError{
			Code: session.ErrCodeBadRequest,
			Err:  i18n.NewError(i18n.KeySessionSemanticModelSourceRequired, nil),
		})

	req := httptest.NewRequest(http.MethodPost, "/newmoi/semantic-models/42/sources/backfill-legacy", nil)
	req.Header.Set("Accept-Language", "zh-CN")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	var body struct {
		Code string `json:"code"`
		Msg  string `json:"msg"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, "ErrParamInvalid", body.Code)
	require.Equal(t, "语义模型来源不能为空", body.Msg)
	require.NotContains(t, body.Msg, "legacy source is invalid")
	svc.AssertExpectations(t)
}

func TestSemanticModelController_ReconcileSourceJobs_Error(t *testing.T) {
	svc := new(MockSemanticModelService)
	router := setupSemanticModelRouter(svc)

	svc.On("ReconcileKnowledgeBaseSourceJobs", mock.Anything, session.ReconcileKnowledgeBaseSourceJobsParams{ModelID: 42}).
		Return(&session.ServiceError{
			Code: session.ErrCodeBadRequest,
			Err:  i18n.NewError(i18n.KeySessionSemanticModelSourcesRequired, nil),
		})

	req := httptest.NewRequest(http.MethodPost, "/newmoi/semantic-models/42/source-jobs/reconcile", nil)
	req.Header.Set("Accept-Language", "zh-CN")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	var body struct {
		Code string `json:"code"`
		Msg  string `json:"msg"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, "ErrParamInvalid", body.Code)
	require.Equal(t, "至少需要一个来源", body.Msg)
	require.NotContains(t, body.Msg, "workflow ingest failed")
	svc.AssertExpectations(t)
}

func TestSemanticModelController_ReconcileSourceJobs_VolumeWriteDenied(t *testing.T) {
	svc := new(MockSemanticModelService)
	router := setupSemanticModelRouter(svc)

	svc.On("ReconcileKnowledgeBaseSourceJobs", mock.Anything, session.ReconcileKnowledgeBaseSourceJobsParams{ModelID: 42}).
		Return(fmt.Errorf("add files to raw volume: %w", &moi.Error{
			Code:    common.ErrorCode_PERMISSION_DENIED,
			Message: "volume.write denied",
		}))

	req := httptest.NewRequest(http.MethodPost, "/newmoi/semantic-models/42/source-jobs/reconcile", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusForbidden, w.Code)
	var body struct {
		Code string `json:"code"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, "ErrForbidden", body.Code)
	svc.AssertExpectations(t)
}

func TestSemanticModelController_CreateEntry_Success(t *testing.T) {
	svc := new(MockSemanticModelService)
	router := setupSemanticModelRouter(svc)

	expected := &session.SemanticEntry{ID: 1, Kind: "metric", Key: "gmv"}
	svc.On("CreateEntry", mock.Anything, mock.Anything).Return(expected, nil)

	body := `{"kind":"metric","key":"gmv","spec":{"expr":"SUM(amount)"}}`
	req := httptest.NewRequest(http.MethodPost, "/newmoi/semantic-models/42/entries", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestSemanticModelController_CreateEntry_InvalidKind(t *testing.T) {
	svc := new(MockSemanticModelService)
	router := setupSemanticModelRouter(svc)

	body := `{"kind":"invalid_kind","key":"test","spec":{"column":"x"}}`
	req := httptest.NewRequest(http.MethodPost, "/newmoi/semantic-models/42/entries", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSemanticModelController_CreateEntry_DuplicateKeyReturnsConflict(t *testing.T) {
	svc := new(MockSemanticModelService)
	router := setupSemanticModelRouter(svc)

	svc.On("CreateEntry", mock.Anything, mock.Anything).Return(nil, &moi.Error{
		Code:    common.ErrorCode_INVALID_ARGUMENT,
		Message: "duplicate key qa_term_zephyr in entries",
		Reason:  "SESSION_DUPLICATE_ENTRY_KEY",
		Domain:  "moi-core.session",
		Metadata: map[string]string{
			"key": "qa_term_zephyr",
		},
	})

	body := `{"kind":"glossary","key":"qa_term_zephyr","spec":{"term":"zephyr","definition":"wind"}}`
	req := httptest.NewRequest(http.MethodPost, "/newmoi/semantic-models/42/entries", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Language", "zh-CN")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusConflict, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, "ErrConflict", resp["code"])
	require.Contains(t, resp["msg"], "重复的 key qa_term_zephyr")
	svc.AssertExpectations(t)
}

func TestSemanticModelController_Import_DuplicateKey(t *testing.T) {
	svc := new(MockSemanticModelService)
	router := setupSemanticModelRouter(svc)

	body := `{"entries":[{"kind":"metric","key":"gmv","spec":{"expr":"SUM(a)"}},{"kind":"metric","key":"gmv","spec":{"expr":"SUM(b)"}}]}`
	req := httptest.NewRequest(http.MethodPost, "/newmoi/semantic-models/42/import", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Language", "zh-CN")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	var resp map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "ErrConflict", resp["code"])
	assert.Contains(t, resp["msg"], "重复的 key")
}

func TestSemanticModelController_Import_TypeMismatchIncludesSpecField(t *testing.T) {
	svc := new(MockSemanticModelService)
	router := setupSemanticModelRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/newmoi/semantic-models/42/import", strings.NewReader(`{"entries":[{"kind":"dimension","key":"city","spec":{"column":1}}]}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Language", "en-US")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	var response struct {
		Code string `json:"code"`
		Msg  string `json:"msg"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	require.Equal(t, "ErrParamInvalid", response.Code)
	require.Contains(t, response.Msg, "entries[0]")
	require.Contains(t, response.Msg, "spec.column")
	require.NotContains(t, response.Msg, "<no value>")
	svc.AssertExpectations(t)
}

func TestSemanticModelController_Import_ServiceDuplicateKeyReturnsConflict(t *testing.T) {
	svc := new(MockSemanticModelService)
	router := setupSemanticModelRouter(svc)

	duplicateErr := &moi.Error{
		Code:    common.ErrorCode_INVALID_ARGUMENT,
		Message: "duplicate key existing_key in entries",
		Reason:  "SESSION_DUPLICATE_ENTRY_KEY",
		Domain:  "moi-core.session",
		Metadata: map[string]string{
			"key": "existing_key",
		},
	}
	svc.On("Import", mock.Anything, mock.Anything).
		Return(nil, fmt.Errorf("import entry %q: %w", "existing_key", duplicateErr))

	body := `{"entries":[{"kind":"metric","key":"existing_key","spec":{"expr":"SUM(a)"}}]}`
	req := httptest.NewRequest(http.MethodPost, "/newmoi/semantic-models/42/import", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Language", "zh-CN")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusConflict, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, "ErrConflict", resp["code"])
	require.Contains(t, resp["msg"], "重复的 key existing_key")
	svc.AssertExpectations(t)
}

func TestSemanticModelController_Import_HasExistingEntries(t *testing.T) {
	svc := new(MockSemanticModelService)
	router := setupSemanticModelRouter(svc)

	svc.On("Import", mock.Anything, mock.Anything).
		Return(nil, &session.ServiceError{Code: session.ErrCodeBadRequest, Err: i18n.NewError(i18n.KeySessionModelEntriesImportBlocked, nil)})

	body := `{"entries":[{"kind":"metric","key":"gmv","spec":{"expr":"SUM(a)"}}]}`
	req := httptest.NewRequest(http.MethodPost, "/newmoi/semantic-models/42/import", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Language", "zh-CN")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Contains(t, resp["msg"], "已有条目")
	svc.AssertExpectations(t)
}

func TestSemanticModelController_Import_Success(t *testing.T) {
	svc := new(MockSemanticModelService)
	router := setupSemanticModelRouter(svc)

	svc.On("Import", mock.Anything, mock.Anything).
		Return(&session.ImportSemanticModelResponse{Imported: 2, ModelID: 42}, nil)

	body := `{"entries":[{"kind":"metric","key":"gmv","spec":{"expr":"SUM(a)"}},{"kind":"dimension","key":"city","spec":{"column":"t.city"}}]}`
	req := httptest.NewRequest(http.MethodPost, "/newmoi/semantic-models/42/import", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "OK", resp["code"])
	data := resp["data"].(map[string]interface{})
	assert.Equal(t, float64(2), data["imported"])
	assert.Equal(t, float64(42), data["model_id"])
	svc.AssertExpectations(t)
}

func TestSemanticModelController_Import_PropagatesRequestTraceIDsToService(t *testing.T) {
	var captured context.Context
	svc := new(MockSemanticModelService)
	router := setupSemanticModelRouter(svc)
	svc.On("Import", mock.MatchedBy(func(ctx context.Context) bool {
		captured = ctx
		return true
	}), mock.Anything).Return(&session.ImportSemanticModelResponse{Imported: 1, ModelID: 42}, nil)

	req := httptest.NewRequest(http.MethodPost, "/newmoi/semantic-models/42/import", strings.NewReader(`{"entries":[{"kind":"metric","key":"gmv","spec":{"expr":"SUM(a)"}}]}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-ID", "req-import-1")
	req.Header.Set("X-Trace-ID", "trace-import-1")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, captured)

	var gotReq, gotTrace string
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		gotReq = r.Header.Get("X-Request-ID")
		gotTrace = r.Header.Get("X-Trace-ID")
		rw.Header().Set("Content-Type", "application/json")
		_, _ = rw.Write([]byte(`{"id":"ws-1","name":"ws"}`))
	}))
	defer server.Close()
	client, err := moi.New(server.URL, "system-key")
	require.NoError(t, err)
	defer client.Close()
	_, err = client.Workspaces().Get(captured, "ws-1")
	require.NoError(t, err)
	require.Equal(t, "req-import-1", gotReq)
	require.Equal(t, "trace-import-1", gotTrace)
	svc.AssertExpectations(t)
}

func TestSemanticModelController_Import_SkipsDisabledLegacyEntries(t *testing.T) {
	svc := new(MockSemanticModelService)
	router := setupSemanticModelRouter(svc)

	svc.On("Import", mock.Anything, mock.MatchedBy(func(params session.ImportSemanticModelRequest) bool {
		return len(params.Entries) == 1 && params.Entries[0].Key == "gmv"
	})).Return(&session.ImportSemanticModelResponse{Imported: 1, ModelID: 42}, nil)

	body := `{"entries":[{"kind":"logic_text","key":"disabled_legacy_rule","tables":["__disabled_legacy_obsolete_rule__"],"spec":{"content":"disabled","injection_stages":["planner_policy"]}},{"kind":"metric","key":"gmv","spec":{"expr":"SUM(a)"}}]}`
	req := httptest.NewRequest(http.MethodPost, "/newmoi/semantic-models/42/import", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	svc.AssertExpectations(t)
}

func TestSemanticModelController_Validate_Success(t *testing.T) {
	svc := new(MockSemanticModelService)
	router := setupSemanticModelRouter(svc)

	svc.On("Validate", mock.Anything, 42).Return(&session.ValidateSemanticModelResponse{Valid: true}, nil)

	req := httptest.NewRequest(http.MethodPost, "/newmoi/semantic-models/42/validate", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSemanticModelController_ListModelTags_StaticRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := new(MockSemanticModelService)
	svc.On("ListModelTags", mock.Anything, session.ListSemanticModelsRequest{
		PageSize:  0,
		PageToken: "",
		Search:    "Ops",
	}).Return(&session.ListSemanticModelTagsResponse{
		Items: []session.SemanticModelTagStat{{Tag: "ops", Count: 2}},
	}, nil).Once()

	r := setupSemanticModelRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/newmoi/semantic-models/tags?search=Ops", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var body struct {
		Code string `json:"code"`
		Data struct {
			Items []session.SemanticModelTagStat `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, "OK", body.Code)
	require.Len(t, body.Data.Items, 1)
	assert.Equal(t, "ops", body.Data.Items[0].Tag)
	assert.Equal(t, int64(2), body.Data.Items[0].Count)
	svc.AssertNotCalled(t, "GetModel", mock.Anything, mock.Anything)
	svc.AssertExpectations(t)
}

func TestSemanticModelController_ListModels_ForwardsTagsFilter(t *testing.T) {
	svc := new(MockSemanticModelService)
	svc.On("ListModels", mock.Anything, session.ListSemanticModelsRequest{
		PageSize:  20,
		PageToken: "",
		Search:    "sales",
		Tags:      []string{"finance", "ops"},
	}).Return(&session.ListSemanticModelsResponse{
		Items: []*session.SemanticModelInfo{{ID: 7, Name: "sales"}},
		Total: 1,
	}, nil).Once()

	r := setupSemanticModelRouter(svc)
	req := httptest.NewRequest(http.MethodGet, "/newmoi/semantic-models?search=sales&tags=finance&tags=ops&tags=finance", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	svc.AssertNotCalled(t, "ListModelsByIDs", mock.Anything, mock.Anything, mock.Anything)
	svc.AssertExpectations(t)
}

func TestSemanticModelController_ListModelTagsDelegatesCanonicalFilteringToCore(t *testing.T) {
	svc := new(MockSemanticModelService)
	svc.On("ListModelTags", mock.Anything, session.ListSemanticModelsRequest{
		Search: "Ops",
	}).Return(&session.ListSemanticModelTagsResponse{
		Items: []session.SemanticModelTagStat{{Tag: "ops", Count: 2}},
	}, nil).Once()

	r := setupSemanticModelRouter(svc)
	req := httptest.NewRequest(http.MethodGet, "/newmoi/semantic-models/tags?search=Ops", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	svc.AssertNotCalled(t, "ListModelsByIDs", mock.Anything, mock.Anything, mock.Anything)
	svc.AssertExpectations(t)
}

func TestSemanticModelController_ListSourcesWithoutPaginationKeepsFullListSemantics(t *testing.T) {
	svc := new(MockSemanticModelService)
	router := setupSemanticModelRouter(svc)

	expected := &session.ListSemanticModelSourcesResult{Items: []session.SemanticModelSource{}, Total: 0, Page: 1, PageSize: 0}
	svc.On("ListSources", mock.Anything, session.ListSemanticModelSourcesParams{ModelID: 42}).Return(expected, nil)

	req := httptest.NewRequest(http.MethodGet, "/newmoi/semantic-models/42/sources", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	svc.AssertExpectations(t)
}

func TestSemanticModelController_CheckSourceExistence_Success(t *testing.T) {
	svc := new(MockSemanticModelService)
	router := setupSemanticModelRouter(svc)

	expected := &session.CheckSemanticModelSourceExistenceResult{
		FileIDs:  []string{"file-1"},
		TableIDs: []int64{1002},
	}
	svc.On("CheckSourceExistence", mock.Anything, session.CheckSemanticModelSourceExistenceParams{
		ModelID:  42,
		FileIDs:  []string{"file-1", "file-2"},
		TableIDs: []int64{1001, 1002},
	}).Return(expected, nil)

	req := httptest.NewRequest(http.MethodPost, "/newmoi/semantic-models/42/sources/existence", strings.NewReader(`{"file_ids":["file-1","file-2"],"table_ids":[1001,1002]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var body struct {
		Code string `json:"code"`
		Data struct {
			FileIDs  []string `json:"file_ids"`
			TableIDs []int64  `json:"table_ids"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, "OK", body.Code)
	assert.Equal(t, []string{"file-1"}, body.Data.FileIDs)
	assert.Equal(t, []int64{1002}, body.Data.TableIDs)
	svc.AssertExpectations(t)
}

func TestSemanticModelController_PreviewSourceSelectionCounts(t *testing.T) {
	selection := session.SemanticModelSourceSelectionRequest{
		Kind:        "volume_files",
		VolumeID:    42,
		AllSelected: true,
		Filters:     session.SemanticModelSourceSelectionFilters{FileName: "report"},
	}
	tests := []struct {
		name    string
		path    string
		modelID int
	}{
		{name: "create", path: "/newmoi/semantic-models/source-selections/preview"},
		{name: "append", path: "/newmoi/semantic-models/42/source-selections/preview", modelID: 42},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := new(MockSemanticModelService)
			router := setupSemanticModelRouter(svc)
			svc.On("PreviewSourceSelectionCounts", mock.Anything, session.PreviewSemanticModelSourceSelectionsRequest{
				ModelID:          tt.modelID,
				SourceSelections: []session.SemanticModelSourceSelectionRequest{selection},
			}).Return(&session.PreviewSemanticModelSourceSelectionsResponse{FileCount: 140, TotalCount: 140}, nil)

			req := httptest.NewRequest(http.MethodPost, tt.path, strings.NewReader(`{"source_selections":[{"kind":"volume_files","volume_id":42,"all_selected":true,"filters":{"file_name":"report"}}]}`))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			require.Equal(t, http.StatusOK, w.Code, w.Body.String())
			var body struct {
				Code string `json:"code"`
				Data struct {
					FileCount  int `json:"file_count"`
					TableCount int `json:"table_count"`
					TotalCount int `json:"total_count"`
				} `json:"data"`
			}
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
			assert.Equal(t, "OK", body.Code)
			assert.Equal(t, 140, body.Data.FileCount)
			assert.Equal(t, 0, body.Data.TableCount)
			assert.Equal(t, 140, body.Data.TotalCount)
			svc.AssertExpectations(t)
		})
	}
}

func TestSemanticModelController_ListModelsDelegatesCanonicalFilteringToCore(t *testing.T) {
	svc := new(MockSemanticModelService)
	svc.On("ListModels", mock.Anything, session.ListSemanticModelsRequest{
		PageSize:  20,
		PageToken: "",
		Search:    "",
	}).Return(&session.ListSemanticModelsResponse{
		Items: []*session.SemanticModelInfo{{ID: 100, Name: "core-filtered"}},
		Total: 1,
	}, nil)
	r := setupSemanticModelRouter(svc)
	req := httptest.NewRequest(http.MethodGet, "/newmoi/semantic-models", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	svc.AssertNotCalled(t, "ListModelsByIDs", mock.Anything, mock.Anything, mock.Anything)
	svc.AssertExpectations(t)
}

func TestSemanticModelController_UploadLocalFile_CreatePathSuccess(t *testing.T) {
	svc := new(MockSemanticModelService)
	router := setupSemanticModelRouter(svc)

	svc.On("UploadLocalFile", mock.Anything, " notes.txt ", mock.Anything).Return("uploaded-file-1", nil)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", " notes.txt ")
	require.NoError(t, err)
	_, err = part.Write([]byte("hello knowledge"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	req := httptest.NewRequest(http.MethodPost, "/newmoi/semantic-models/local-files/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, "OK", resp["code"])
	data, ok := resp["data"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "uploaded-file-1", data["file_id"])
	svc.AssertExpectations(t)
}

func TestSemanticModelController_UploadLocalFile_AppendPathSuccess(t *testing.T) {
	svc := new(MockSemanticModelService)
	router := setupSemanticModelRouter(svc)

	svc.On("UploadLocalFile", mock.Anything, "notes.txt", mock.Anything).Return("uploaded-file-2", nil)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "notes.txt")
	require.NoError(t, err)
	_, err = part.Write([]byte("hello knowledge"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	req := httptest.NewRequest(http.MethodPost, "/newmoi/semantic-models/42/local-files/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, "OK", resp["code"])
	data, ok := resp["data"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "uploaded-file-2", data["file_id"])
	svc.AssertExpectations(t)
}

func TestWithSemanticModelCoreHeadersLeavesRoleForCoreclientExecute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Request.Header.Set("X-Request-ID", "req-1")
	c.Request.Header.Set("X-Trace-ID", "trace-1")
	c.Request.Header.Set("Authorization", "Bearer should-not-copy")
	c.Request.Header.Set("Cookie", "session=nope")
	c.Request.Header.Set(iampep.HeaderMOIRoleID, "untrusted-role")
	c.Request = c.Request.WithContext(ctxutil.WithCoreIAMRequest(c.Request.Context(), ctxutil.CoreIAMRequestContext{
		VerifiedEffectiveRoleID: "verified-role",
	}))

	var gotReq, gotTrace, gotAuth, gotCookie, gotRole string
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		gotReq = r.Header.Get("X-Request-ID")
		gotTrace = r.Header.Get("X-Trace-ID")
		gotAuth = r.Header.Get("Authorization")
		gotCookie = r.Header.Get("Cookie")
		gotRole = r.Header.Get(iampep.HeaderMOIRoleID)
		rw.Header().Set("Content-Type", "application/json")
		_, _ = rw.Write([]byte(`{"id":"ws-1","name":"ws"}`))
	}))
	defer server.Close()
	client, err := moi.New(server.URL, "system-key")
	require.NoError(t, err)
	defer client.Close()

	ctx := withSemanticModelCoreHeaders(c.Request.Context(), c)
	_, err = client.Workspaces().Get(ctx, "ws-1")
	require.NoError(t, err)
	require.Equal(t, "req-1", gotReq)
	require.Equal(t, "trace-1", gotTrace)
	require.Empty(t, gotAuth)
	require.Empty(t, gotCookie)
	require.Empty(t, gotRole)
	require.Equal(t, "verified-role", ctxutil.CoreIAMEffectiveRoleFrom(ctx))

	// Missing trace falls back to request id.
	c.Request.Header.Del("X-Trace-ID")
	gotReq, gotTrace = "", ""
	ctx = withSemanticModelCoreHeaders(c.Request.Context(), c)
	_, err = client.Workspaces().Get(ctx, "ws-1")
	require.NoError(t, err)
	require.Equal(t, "req-1", gotReq)
	require.Equal(t, "req-1", gotTrace)
}

func TestSemanticModelController_UploadLocalFile_PropagatesRequestTraceIDsToService(t *testing.T) {
	var captured context.Context
	svc := new(MockSemanticModelService)
	router := setupSemanticModelRouter(svc)
	svc.On("UploadLocalFile", mock.MatchedBy(func(ctx context.Context) bool {
		captured = ctx
		return true
	}), "notes.txt", mock.Anything).Return("uploaded-file-1", nil)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "notes.txt")
	require.NoError(t, err)
	_, err = part.Write([]byte("hello knowledge"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	req := httptest.NewRequest(http.MethodPost, "/newmoi/semantic-models/local-files/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-Request-ID", "req-upload-2")
	req.Header.Set("X-Trace-ID", "trace-upload-2")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, captured)

	var gotReq, gotTrace string
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		gotReq = r.Header.Get("X-Request-ID")
		gotTrace = r.Header.Get("X-Trace-ID")
		rw.Header().Set("Content-Type", "application/json")
		_, _ = rw.Write([]byte(`{"id":"ws-1","name":"ws"}`))
	}))
	defer server.Close()
	client, err := moi.New(server.URL, "system-key")
	require.NoError(t, err)
	defer client.Close()
	_, err = client.Workspaces().Get(captured, "ws-1")
	require.NoError(t, err)
	require.Equal(t, "req-upload-2", gotReq)
	require.Equal(t, "trace-upload-2", gotTrace)
	svc.AssertExpectations(t)
}

func TestSemanticModelController_UploadLocalFile_MissingFile(t *testing.T) {
	svc := new(MockSemanticModelService)
	router := setupSemanticModelRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/newmoi/semantic-models/local-files/upload", strings.NewReader(""))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=x")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, "ErrParamInvalid", resp["code"])
	svc.AssertNotCalled(t, "UploadLocalFile", mock.Anything, mock.Anything, mock.Anything)
}
