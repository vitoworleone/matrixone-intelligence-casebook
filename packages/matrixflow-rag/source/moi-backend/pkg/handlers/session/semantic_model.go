package sessionh

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	moi "github.com/matrixflow/moi-core/go-sdk"
	"github.com/matrixflow/moi-core/model/common"
	"github.com/matrixorigin/matrixflow/moi-backend/pkg/ctxutil"
	"github.com/matrixorigin/matrixflow/moi-backend/pkg/i18n"
	"github.com/matrixorigin/matrixflow/moi-backend/pkg/iampep"
	"github.com/matrixorigin/matrixflow/moi-backend/pkg/logger"
	"github.com/matrixorigin/matrixflow/moi-backend/pkg/middleware"
	session "github.com/matrixorigin/matrixflow/moi-backend/pkg/session"
)

type coreErrorResponse struct {
	Reason   string            `json:"reason"`
	Domain   string            `json:"domain"`
	Metadata map[string]string `json:"metadata"`
}

const (
	coreDomainSession           = "moi-core.session"
	coreReasonDuplicateEntryKey = "SESSION_DUPLICATE_ENTRY_KEY"
)

func serviceErrorBody(errCode string, m string, err error) gin.H {
	body := gin.H{"code": errCode, "msg": m, "data": nil}
	coreErr, ok := i18n.ParseCoreError(err)
	if !ok {
		return body
	}
	metadata := coreErr.Metadata
	if metadata == nil {
		metadata = map[string]string{}
	}
	body["error"] = coreErrorResponse{
		Reason:   coreErr.Reason,
		Domain:   coreErr.Domain,
		Metadata: metadata,
	}
	return body
}

// SemanticModelController handles semantic model and entry management endpoints.
type SemanticModelController struct {
	Svc                session.SemanticModelService
	IAMPEP             iampep.BindingConfig
	DependencyResolver SemanticModelDependencyResolver
}

// RegisterRoutes registers RESTful semantic model routes.
// RegisterRoutes registers RESTful semantic model routes.
//
//	GET    /semantic-models/:model_id                       → GetModel
//	GET    /semantic-models/:model_id/sources               → ListSources
//	POST   /semantic-models/:model_id/sources               → AppendSources
//	DELETE /semantic-models/:model_id/sources/:source_row_id → DeleteSource
//	GET    /semantic-models/:model_id/sources/file/:file_id/preview → PreviewSourceFile (path value is file_id)
//	GET    /semantic-models/:model_id/sources/:source_row_id/document → GetSourceDocument
//	PATCH  /semantic-models/:model_id/sources/:source_row_id/governance → UpdateSourceGovernance
//	POST   /semantic-models/:model_id/sources/:source_row_id/segments/import-initial → ImportInitialSegments
//	PATCH  /semantic-models/:model_id/sources/:source_row_id/segments/:segment_id → UpdateSegment
//	POST   /semantic-models/:model_id/sources/:source_row_id/segments → CreateSegment
//	PATCH  /semantic-models/:model_id/sources/:source_row_id/segments/:segment_id/enabled → UpdateSegmentEnabled
//	DELETE /semantic-models/:model_id/sources/:source_row_id/segments/:segment_id → DeleteSegment
//	POST   /semantic-models/:model_id/sources/:source_row_id/segments/re-embedding → ReembedSegments
//	PATCH  /semantic-models/:model_id/sources/:source_row_id/segment-versions/:version_id/current → SetCurrentSegmentVersion
//	GET    /semantic-models/:model_id/source-jobs           → ListSourceJobs
//	POST   /semantic-models/:model_id/source-jobs/reconcile → ReconcileSourceJobs
//	GET    /semantic-models/:model_id/entries                → ListEntries
//	POST   /semantic-models/:model_id/entries                → CreateEntry
//	PUT    /semantic-models/:model_id/entries/:entry_id      → UpdateEntry
//	DELETE /semantic-models/:model_id/entries/:entry_id      → DeleteEntry
//	POST   /semantic-models/:model_id/import                 → Import
//	GET    /semantic-models/:model_id/export                 → Export
//	POST   /semantic-models/:model_id/validate               → Validate
//
// RegisterRoutes registers RESTful semantic model routes.
//
//	POST   /semantic-models                                  → CreateModel
//	POST   /semantic-models/create-empty                     → CreateEmptyModel
//	POST   /semantic-models/local-files/upload               → UploadLocalFile
//	POST   /semantic-models/source-selections/preview        → PreviewSourceSelectionCounts
//	GET    /semantic-models                                  → ListModels
//	GET    /semantic-models/tags                             → ListModelTags
//	GET    /semantic-models/:model_id                        → GetModel
//	GET    /semantic-models/:model_id/sources                → ListSources
//	POST   /semantic-models/:model_id/sources/existence      → CheckSourceExistence
//	POST   /semantic-models/:model_id/source-selections/preview → PreviewModelSourceSelectionCounts
//	POST   /semantic-models/:model_id/local-files/upload     → UploadLocalFile
//	POST   /semantic-models/:model_id/sources                → AppendSources
//	DELETE /semantic-models/:model_id/sources/:source_row_id → DeleteSource
//	GET    /semantic-models/:model_id/sources/file/:file_id/preview → PreviewSourceFile (path value is file_id)
//	GET    /semantic-models/:model_id/sources/:source_row_id/document → GetSourceDocument
//	PATCH  /semantic-models/:model_id/sources/:source_row_id/governance → UpdateSourceGovernance
//	POST   /semantic-models/:model_id/sources/:source_row_id/segments/import-initial → ImportInitialSegments
//	PATCH  /semantic-models/:model_id/sources/:source_row_id/segments/:segment_id → UpdateSegment
//	POST   /semantic-models/:model_id/sources/:source_row_id/segments → CreateSegment
//	PATCH  /semantic-models/:model_id/sources/:source_row_id/segments/:segment_id/enabled → UpdateSegmentEnabled
//	DELETE /semantic-models/:model_id/sources/:source_row_id/segments/:segment_id → DeleteSegment
//	POST   /semantic-models/:model_id/sources/:source_row_id/segments/re-embedding → ReembedSegments
//	PATCH  /semantic-models/:model_id/sources/:source_row_id/segment-versions/:version_id/current → SetCurrentSegmentVersion
//	GET    /semantic-models/:model_id/source-jobs            → ListSourceJobs
//	POST   /semantic-models/:model_id/source-jobs/reconcile  → ReconcileSourceJobs
//	PUT    /semantic-models/:model_id                        → UpdateModel
//	DELETE /semantic-models/:model_id                        → DeleteModel
//	GET    /semantic-models/:model_id/entries                → ListEntries
//	POST   /semantic-models/:model_id/entries                → CreateEntry
//	PUT    /semantic-models/:model_id/entries/:entry_id      → UpdateEntry
//	DELETE /semantic-models/:model_id/entries/:entry_id      → DeleteEntry
//	POST   /semantic-models/:model_id/import                 → Import
//	GET    /semantic-models/:model_id/export                 → Export
//	POST   /semantic-models/:model_id/validate
func (sc *SemanticModelController) RegisterRoutes(r *gin.RouterGroup) {
	col := r.Group("/semantic-models")
	col.POST("", iampep.Require(sc.workspaceBinding("create", "semantic_model.create")), sc.requireLegacyDependencies(), sc.CreateModel)
	col.POST("/create-empty", iampep.Require(sc.workspaceBinding("create", "semantic_model.create")), sc.CreateEmptyModel)
	col.POST("/create-with-sources", iampep.Require(sc.workspaceBinding("create", "semantic_model.create")), sc.requireSourceAndSelectionDependencies(), sc.CreateModelWithSources)
	col.POST("/local-files/upload", iampep.Require(sc.workspaceBinding("create", "semantic_model.create")), sc.UploadLocalFile)
	col.POST("/source-selections/preview", iampep.Require(sc.workspaceBinding("create", "semantic_model.create")), sc.requireSelectionDependencies(), sc.PreviewSourceSelectionCounts)
	// Core applies semantic_model.read before count and pagination.
	col.GET("", iampep.PrepareCoreRequestContext(sc.IAMPEP, "semantic_model"), sc.ListModels)
	col.GET("/tags", iampep.PrepareCoreRequestContext(sc.IAMPEP, "semantic_model"), sc.ListModelTags)

	sm := col.Group("/:model_id")
	sm.GET("", iampep.Require(sc.objectBinding("read", "semantic_model.read")), sc.GetModel)
	sm.GET("/artifacts/:file_id/preview", iampep.Require(sc.objectBinding("read", "semantic_model.read")), sc.PreviewArtifact)
	sm.GET("/sources", iampep.Require(sc.objectBinding("read", "semantic_model.read")), sc.ListSources)
	sm.POST("/sources/existence", iampep.Require(sc.objectBinding("read", "semantic_model.read")), sc.CheckSourceExistence)
	sm.POST("/source-selections/preview", iampep.Require(sc.objectBinding("update", "semantic_model.update")), sc.requireSelectionDependencies(), sc.PreviewModelSourceSelectionCounts)
	sm.POST("/local-files/upload", iampep.Require(sc.objectBinding("update", "semantic_model.update")), sc.UploadLocalFile)
	sm.POST("/sources", iampep.Require(sc.objectBinding("use", "semantic_model.use")), iampep.Require(sc.objectBinding("update", "semantic_model.update")), sc.requireSourceAndSelectionDependencies(), sc.AppendSources)
	sm.POST("/sources/backfill-legacy", iampep.Require(sc.objectBinding("use", "semantic_model.use")), iampep.Require(sc.objectBinding("update", "semantic_model.update")), sc.requireBackfillDependencies(), sc.BackfillLegacySources)
	sm.DELETE("/sources/:source_row_id", iampep.Require(sc.objectBinding("delete", "semantic_model.delete")), sc.DeleteSource)
	// Source preview has a separate static file branch and therefore accepts a File ID.
	sm.GET("/sources/file/:file_id/preview", iampep.Require(sc.objectBinding("read", "semantic_model.read")), sc.PreviewSourceFile)
	sm.GET("/sources/:source_row_id/document", iampep.Require(sc.objectBinding("read", "semantic_model.read")), sc.GetSourceDocument)
	sm.PATCH("/sources/:source_row_id/governance", iampep.Require(sc.objectBinding("update", "semantic_model.update")), sc.UpdateSourceGovernance)
	sm.POST("/sources/:source_row_id/segments/import-initial", iampep.Require(sc.objectBinding("use", "semantic_model.use")), iampep.Require(sc.objectBinding("update", "semantic_model.update")), sc.ImportInitialSegments)
	sm.PATCH("/sources/:source_row_id/segments/:segment_id", iampep.Require(sc.objectBinding("update", "semantic_model.update")), sc.UpdateSegment)
	sm.POST("/sources/:source_row_id/segments", iampep.Require(sc.objectBinding("update", "semantic_model.update")), sc.CreateSegment)
	sm.PATCH("/sources/:source_row_id/segments/:segment_id/enabled", iampep.Require(sc.objectBinding("update", "semantic_model.update")), sc.UpdateSegmentEnabled)
	sm.DELETE("/sources/:source_row_id/segments/:segment_id", iampep.Require(sc.objectBinding("delete", "semantic_model.delete")), sc.DeleteSegment)
	sm.POST("/sources/:source_row_id/segments/re-embedding", iampep.Require(sc.objectBinding("use", "semantic_model.use")), iampep.Require(sc.objectBinding("update", "semantic_model.update")), sc.ReembedSegments)
	sm.PATCH("/sources/:source_row_id/segment-versions/:version_id/current", iampep.Require(sc.objectBinding("update", "semantic_model.update")), sc.SetCurrentSegmentVersion)
	sm.GET("/source-jobs", iampep.Require(sc.objectBinding("read", "semantic_model.read")), sc.ListSourceJobs)
	sm.POST("/source-jobs/reconcile", iampep.Require(sc.objectBinding("use", "semantic_model.use")), iampep.Require(sc.objectBinding("update", "semantic_model.update")), sc.ReconcileSourceJobs)
	sm.PUT("", iampep.Require(sc.objectBinding("update", "semantic_model.update")), sc.requireLegacyDependencies(), sc.UpdateModel)
	sm.DELETE("", iampep.Require(sc.objectBinding("delete", "semantic_model.delete")), sc.DeleteModel)
	sm.GET("/entries", iampep.Require(sc.objectBinding("read", "semantic_model.read")), sc.ListEntries)
	sm.POST("/entries", iampep.Require(sc.objectBinding("update", "semantic_model.update")), sc.CreateEntry)
	sm.PUT("/entries/:entry_id", iampep.Require(sc.objectBinding("update", "semantic_model.update")), sc.UpdateEntry)
	sm.DELETE("/entries/:entry_id", iampep.Require(sc.objectBinding("delete", "semantic_model.delete")), sc.DeleteEntry)
	sm.POST("/import", iampep.Require(sc.objectBinding("use", "semantic_model.use")), iampep.Require(sc.objectBinding("update", "semantic_model.update")), sc.Import)
	sm.GET("/export", iampep.Require(sc.objectBinding("read", "semantic_model.read")), sc.Export)
	sm.POST("/validate", iampep.Require(sc.objectBinding("use", "semantic_model.use")), sc.Validate)
}

func (sc *SemanticModelController) requireSelectionDependencies() gin.HandlerFunc {
	return iampep.RequireDependencies(sc.IAMPEP, "semantic_model", iampep.ResourceAuthorizationExtractorFunc(func(c *gin.Context) ([]iampep.ResourceAuthorization, error) {
		if sc.DependencyResolver == nil {
			return nil, fmt.Errorf("semantic model dependency resolver is unavailable")
		}
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			return nil, fmt.Errorf("read semantic model selection body: %w", err)
		}
		c.Request.Body = io.NopCloser(bytes.NewReader(body))
		var request struct {
			SourceSelections []session.SemanticModelSourceSelectionRequest `json:"source_selections"`
		}
		if err := json.Unmarshal(body, &request); err != nil {
			return nil, semanticModelDependencyRequestError{err: fmt.Errorf("decode semantic model selection body: %w", err)}
		}
		if len(request.SourceSelections) == 0 {
			return nil, semanticModelDependencyRequestError{err: fmt.Errorf("semantic model source selections are required")}
		}
		return sc.DependencyResolver.ResolveSelectionDependencies(c.Request.Context(), middleware.GetWorkspaceID(c), request.SourceSelections)
	}))
}

func (sc *SemanticModelController) requireBackfillDependencies() gin.HandlerFunc {
	return iampep.RequireDependencies(sc.IAMPEP, "semantic_model", iampep.ResourceAuthorizationExtractorFunc(func(c *gin.Context) ([]iampep.ResourceAuthorization, error) {
		if sc.DependencyResolver == nil {
			return nil, fmt.Errorf("semantic model dependency resolver is unavailable")
		}
		modelID, err := strconv.ParseInt(c.Param("model_id"), 10, 64)
		if err != nil || modelID <= 0 {
			return nil, semanticModelDependencyRequestError{err: fmt.Errorf("semantic model identity is invalid")}
		}
		return sc.DependencyResolver.ResolveBackfillDependencies(c.Request.Context(), middleware.GetWorkspaceID(c), modelID)
	}))
}

func (sc *SemanticModelController) requireLegacyDependencies() gin.HandlerFunc {
	return iampep.RequireDependencies(sc.IAMPEP, "semantic_model", iampep.ResourceAuthorizationExtractorFunc(func(c *gin.Context) ([]iampep.ResourceAuthorization, error) {
		if sc.DependencyResolver == nil {
			return nil, fmt.Errorf("semantic model dependency resolver is unavailable")
		}
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			return nil, fmt.Errorf("read semantic model body: %w", err)
		}
		c.Request.Body = io.NopCloser(bytes.NewReader(body))
		var request struct {
			Tables json.RawMessage `json:"tables"`
			Files  json.RawMessage `json:"files"`
		}
		if err := json.Unmarshal(body, &request); err != nil {
			return nil, semanticModelDependencyRequestError{err: fmt.Errorf("decode semantic model body: %w", err)}
		}
		return sc.DependencyResolver.ResolveLegacyDependencies(c.Request.Context(), middleware.GetWorkspaceID(c), request.Tables, request.Files)
	}))
}

// requireSourceAndSelectionDependencies authorizes the union of direct source
// and source-selection dependencies declared by a create-with-sources or
// append-sources request; RequireDependencies deduplicates the merged set.
func (sc *SemanticModelController) requireSourceAndSelectionDependencies() gin.HandlerFunc {
	return iampep.RequireDependencies(sc.IAMPEP, "semantic_model", iampep.ResourceAuthorizationExtractorFunc(func(c *gin.Context) ([]iampep.ResourceAuthorization, error) {
		if sc.DependencyResolver == nil {
			return nil, fmt.Errorf("semantic model dependency resolver is unavailable")
		}
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			return nil, fmt.Errorf("read semantic model source body: %w", err)
		}
		c.Request.Body = io.NopCloser(bytes.NewReader(body))
		var request struct {
			Sources          []session.CreateSemanticModelSourceRequest    `json:"sources"`
			SourceSelections []session.SemanticModelSourceSelectionRequest `json:"source_selections"`
		}
		if err := json.Unmarshal(body, &request); err != nil {
			return nil, semanticModelDependencyRequestError{err: fmt.Errorf("decode semantic model source body: %w", err)}
		}
		if len(request.Sources) == 0 && len(request.SourceSelections) == 0 {
			return nil, semanticModelDependencyRequestError{err: fmt.Errorf("semantic model sources or source selections are required")}
		}
		out := make([]iampep.ResourceAuthorization, 0, len(request.Sources)+len(request.SourceSelections))
		if len(request.Sources) > 0 {
			resolved, resolveErr := sc.DependencyResolver.ResolveSourceDependencies(c.Request.Context(), middleware.GetWorkspaceID(c), request.Sources)
			if resolveErr != nil {
				return nil, resolveErr
			}
			out = append(out, resolved...)
		}
		if len(request.SourceSelections) > 0 {
			resolved, resolveErr := sc.DependencyResolver.ResolveSelectionDependencies(c.Request.Context(), middleware.GetWorkspaceID(c), request.SourceSelections)
			if resolveErr != nil {
				return nil, resolveErr
			}
			out = append(out, resolved...)
		}
		return out, nil
	}))
}

func (sc *SemanticModelController) workspaceBinding(action, actionID string) iampep.RouteBinding {
	return sc.IAMPEP.WorkspaceDirectCreateRouteBinding("semantic_model", action, actionID)
}

func (sc *SemanticModelController) objectBinding(action, actionID string) iampep.RouteBinding {
	return sc.IAMPEP.ObjectRouteBinding("semantic_model", action, actionID, iampep.ResourceTypeSemanticModel, iampep.PathParamExtractor(iampep.ResourceTypeSemanticModel, "model_id"))
}

func parseModelID(c *gin.Context) (int, error) {
	idStr := c.Param("model_id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		return 0, i18n.NewError(i18n.KeySessionInvalidModelID, nil)
	}
	return id, nil
}

func parseEntryID(c *gin.Context) (int, error) {
	idStr := c.Param("entry_id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		return 0, i18n.NewError(i18n.KeySessionInvalidEntryID, nil)
	}
	return id, nil
}

func (sc *SemanticModelController) PreviewArtifact(c *gin.Context) {
	modelID, err := parseModelID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "ErrParamInvalid", "msg": i18n.PublicErrorMessage(c.Request.Context(), err, i18n.KeyParamInvalid), "data": nil})
		return
	}
	fileID := c.Param("file_id")
	if fileID == "" || strings.TrimSpace(fileID) != fileID {
		c.JSON(http.StatusBadRequest, gin.H{"code": "ErrParamInvalid", "msg": i18n.T(c.Request.Context(), i18n.KeyParamInvalid), "data": nil})
		return
	}

	result, err := sc.Svc.PreviewArtifact(withSemanticModelCoreHeaders(c.Request.Context(), c), modelID, fileID)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	sc.streamSemanticModelFilePreview(c, modelID, "file_id", fileID, "semantic model artifact preview is unavailable", result)
}

func (sc *SemanticModelController) PreviewSourceFile(c *gin.Context) {
	modelID, err := parseModelID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "ErrParamInvalid", "msg": i18n.PublicErrorMessage(c.Request.Context(), err, i18n.KeyParamInvalid), "data": nil})
		return
	}
	fileID := c.Param("file_id")
	if fileID == "" || strings.TrimSpace(fileID) != fileID {
		c.JSON(http.StatusBadRequest, gin.H{"code": "ErrParamInvalid", "msg": i18n.T(c.Request.Context(), i18n.KeyParamInvalid), "data": nil})
		return
	}

	result, err := sc.Svc.PreviewSourceFile(withSemanticModelCoreHeaders(c.Request.Context(), c), modelID, fileID)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	sc.streamSemanticModelFilePreview(c, modelID, "file_id", fileID, "semantic model source file preview is unavailable", result)
}

func (sc *SemanticModelController) streamSemanticModelFilePreview(
	c *gin.Context,
	modelID int,
	previewIDField string,
	previewID string,
	unavailableMessage string,
	result *session.SemanticModelFilePreview,
) {
	if result == nil || result.Body == nil {
		mapServiceError(c, &session.ServiceError{Code: session.ErrCodeInternal, Msg: unavailableMessage})
		return
	}
	defer result.Body.Close()

	contentType := strings.TrimSpace(result.ContentType)
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	if result.Filename != "" {
		c.Header("Content-Disposition", "inline; filename*=UTF-8''"+url.QueryEscape(result.Filename))
	}
	c.Header("Content-Type", contentType)
	c.Status(http.StatusOK)
	if _, err := io.Copy(c.Writer, result.Body); err != nil {
		logger.Error("semantic model file preview stream failed",
			"workspace_id", ctxutil.WorkspaceIDFrom(c.Request.Context()),
			"model_id", modelID,
			previewIDField, previewID,
			"request_id", c.GetHeader("X-Request-ID"),
			"error", err,
		)
	}
}

// mapServiceError maps service/SDK errors to appropriate HTTP responses.
func mapServiceError(c *gin.Context, err error) {
	if isDuplicateSemanticEntryKeyError(err) {
		writeServiceError(c, http.StatusConflict, "ErrConflict", i18n.KeyConflict, err)
		return
	}
	if session.IsServiceError(err, session.ErrCodeNotFound) {
		writeServiceError(c, http.StatusNotFound, "ErrNotFound", i18n.KeyNotFound, err)
		return
	}
	if session.IsServiceError(err, session.ErrCodeConflict) {
		writeServiceError(c, http.StatusConflict, "ErrConflict", i18n.KeyConflict, err)
		return
	}
	if session.IsServiceError(err, session.ErrCodeBadRequest) {
		writeServiceError(c, http.StatusBadRequest, "ErrParamInvalid", i18n.KeyParamInvalid, err)
		return
	}
	if moi.IsCode(err, common.ErrorCode_NOT_FOUND) {
		writeServiceError(c, http.StatusNotFound, "ErrNotFound", i18n.KeyNotFound, err)
		return
	}
	if moi.IsCode(err, common.ErrorCode_ALREADY_EXISTS) {
		writeServiceError(c, http.StatusConflict, "ErrConflict", i18n.KeyConflict, err)
		return
	}
	if moi.IsCode(err, common.ErrorCode_INVALID_ARGUMENT) {
		writeServiceError(c, http.StatusBadRequest, "ErrParamInvalid", i18n.KeyParamInvalid, err)
		return
	}
	if moi.IsCode(err, common.ErrorCode_UNAUTHENTICATED) {
		writeServiceError(c, http.StatusUnauthorized, "ErrUnauthorized", i18n.KeyUnauthorized, err)
		return
	}
	if moi.IsCode(err, common.ErrorCode_PERMISSION_DENIED) || moi.IsCode(err, common.ErrorCode_FORBIDDEN) {
		writeServiceError(c, http.StatusForbidden, "ErrForbidden", i18n.KeyPermissionDenied, err)
		return
	}
	writeServiceError(c, http.StatusInternalServerError, "ErrServer", i18n.KeyServer, err)
}

func writeServiceError(c *gin.Context, status int, code string, terminal i18n.Key, err error) {
	logger.Error("semantic model request failed",
		"status", status,
		"error_code", code,
		"request_id", middleware.RequestID(c),
		"trace_id", middleware.TraceID(c),
		"error", err,
	)
	message := i18n.PublicErrorMessage(c.Request.Context(), err, terminal)
	c.JSON(status, serviceErrorBody(code, message, err))
}

func isDuplicateSemanticEntryKeyError(err error) bool {
	coreErr, ok := i18n.ParseCoreError(err)
	return ok && coreErr.Domain == coreDomainSession && coreErr.Reason == coreReasonDuplicateEntryKey
}

// CreateModel POST /semantic-models
func (sc *SemanticModelController) CreateModel(c *gin.Context) {
	var params session.CreateSemanticModelRequest
	if err := c.ShouldBindJSON(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "ErrParamInvalid", "msg": i18n.RenderPublicMessage(c.Request.Context(), i18n.TranslateValidatorError(c.Request.Context(), err)), "data": nil})
		return
	}
	if strings.TrimSpace(params.Name) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": "ErrParamInvalid", "msg": i18n.T(c.Request.Context(), i18n.KeyNameRequired), "data": nil})
		return
	}
	model, err := sc.Svc.CreateModel(c.Request.Context(), params)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"code": "OK", "msg": "OK", "data": model})
}

// CreateEmptyModel POST /semantic-models/create-empty
func (sc *SemanticModelController) CreateEmptyModel(c *gin.Context) {
	var params session.CreateEmptySemanticModelRequest
	if err := c.ShouldBindJSON(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "ErrParamInvalid", "msg": i18n.RenderPublicMessage(c.Request.Context(), i18n.TranslateValidatorError(c.Request.Context(), err)), "data": nil})
		return
	}
	if strings.TrimSpace(params.Name) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": "ErrParamInvalid", "msg": i18n.T(c.Request.Context(), i18n.KeyNameRequired), "data": nil})
		return
	}
	ctx := withSemanticModelCoreHeaders(c.Request.Context(), c)
	response, err := sc.Svc.CreateEmptyModel(ctx, params)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"code": "OK", "msg": "OK", "data": response})
}

// withSemanticModelCoreHeaders copies request tracing into Core calls. Role
// identity is reconstructed only by coreclient.Execute from trusted context.
func withSemanticModelCoreHeaders(ctx context.Context, c *gin.Context) context.Context {
	if c == nil || c.Request == nil {
		return ctx
	}
	requestID := strings.TrimSpace(c.GetHeader("X-Request-ID"))
	traceID := strings.TrimSpace(c.GetHeader("X-Trace-ID"))
	if requestID == "" && traceID == "" {
		return ctx
	}
	headers := make(map[string]string, 3)
	if requestID != "" {
		headers["X-Request-ID"] = requestID
	}
	if traceID == "" {
		traceID = requestID
	}
	if traceID != "" {
		headers["X-Trace-ID"] = traceID
	}
	return moi.ContextWithHeaders(ctx, headers)
}

// CreateModelWithSources POST /semantic-models/create-with-sources
func (sc *SemanticModelController) CreateModelWithSources(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "ErrParamInvalid", "msg": i18n.T(c.Request.Context(), i18n.KeyParamInvalid), "data": nil})
		return
	}
	// Reject removed target_catalog_id before bind so old clients cannot silently
	// create into the shared KB catalog while believing they pinned a catalog.
	if requestBodyHasJSONField(body, "target_catalog_id") {
		c.JSON(http.StatusBadRequest, gin.H{"code": "ErrParamInvalid", "msg": i18n.T(c.Request.Context(), i18n.KeySessionSemanticModelTargetCatalogIDUnsupported), "data": nil})
		return
	}
	// Restore body and keep ShouldBindJSON so product-sdk contract-field-check
	// can resolve the request DTO (json.Unmarshal alone is invisible to the gate).
	c.Request.Body = io.NopCloser(bytes.NewReader(body))
	var params session.CreateSemanticModelWithSourcesRequest
	if err := c.ShouldBindJSON(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "ErrParamInvalid", "msg": i18n.RenderPublicMessage(c.Request.Context(), i18n.TranslateValidatorError(c.Request.Context(), err)), "data": nil})
		return
	}
	if strings.TrimSpace(params.Name) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": "ErrParamInvalid", "msg": i18n.T(c.Request.Context(), i18n.KeyNameRequired), "data": nil})
		return
	}
	ctx := withSemanticModelCoreHeaders(c.Request.Context(), c)
	result, err := sc.Svc.CreateModelWithSources(ctx, params)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"code": "OK", "msg": "OK", "data": result})
}

// requestBodyHasJSONField reports whether a top-level JSON object field is present.
func requestBodyHasJSONField(body []byte, field string) bool {
	if len(body) == 0 || strings.TrimSpace(field) == "" {
		return false
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return false
	}
	_, ok := raw[field]
	return ok
}

// UploadLocalFile POST /semantic-models/local-files/upload
// UploadLocalFile POST /semantic-models/:model_id/local-files/upload
// Multipart field name is "file". Returns catalog file_id for subsequent local_file source binding.
func (sc *SemanticModelController) UploadLocalFile(c *gin.Context) {
	if modelIDParam := c.Param("model_id"); modelIDParam != "" {
		if _, err := parseModelID(c); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": "ErrParamInvalid", "msg": i18n.PublicErrorMessage(c.Request.Context(), err, i18n.KeyParamInvalid), "data": nil})
			return
		}
	}
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "ErrParamInvalid", "msg": i18n.T(c.Request.Context(), i18n.KeyFileRequired), "data": nil})
		return
	}
	defer file.Close()

	fileName := header.Filename
	if strings.TrimSpace(fileName) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": "ErrParamInvalid", "msg": i18n.T(c.Request.Context(), i18n.KeyFileRequired), "data": nil})
		return
	}

	ctx := withSemanticModelCoreHeaders(c.Request.Context(), c)
	fileID, err := sc.Svc.UploadLocalFile(ctx, fileName, file)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": "OK", "msg": "OK", "data": gin.H{"file_id": fileID}})
}

// AppendSources POST /semantic-models/:model_id/sources
func (sc *SemanticModelController) AppendSources(c *gin.Context) {
	modelID, err := parseModelID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "ErrParamInvalid", "msg": i18n.PublicErrorMessage(c.Request.Context(), err, i18n.KeyParamInvalid), "data": nil})
		return
	}
	var params session.AppendSemanticModelSourcesRequest
	if err := c.ShouldBindJSON(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "ErrParamInvalid", "msg": i18n.RenderPublicMessage(c.Request.Context(), i18n.TranslateValidatorError(c.Request.Context(), err)), "data": nil})
		return
	}
	params.ModelID = modelID
	ctx := withSemanticModelCoreHeaders(c.Request.Context(), c)
	result, err := sc.Svc.AppendModelSources(ctx, params)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": "OK", "msg": "OK", "data": result})
}

// PreviewSourceSelectionCounts POST /semantic-models/source-selections/preview
func (sc *SemanticModelController) PreviewSourceSelectionCounts(c *gin.Context) {
	sc.previewSourceSelectionCounts(c, 0)
}

// PreviewModelSourceSelectionCounts POST /semantic-models/:model_id/source-selections/preview
func (sc *SemanticModelController) PreviewModelSourceSelectionCounts(c *gin.Context) {
	modelID, err := parseModelID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "ErrParamInvalid", "msg": i18n.PublicErrorMessage(c.Request.Context(), err, i18n.KeyParamInvalid), "data": nil})
		return
	}
	sc.previewSourceSelectionCounts(c, modelID)
}

func (sc *SemanticModelController) previewSourceSelectionCounts(c *gin.Context, modelID int) {
	var params session.PreviewSemanticModelSourceSelectionsRequest
	if err := c.ShouldBindJSON(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "ErrParamInvalid", "msg": i18n.RenderPublicMessage(c.Request.Context(), i18n.TranslateValidatorError(c.Request.Context(), err)), "data": nil})
		return
	}
	params.ModelID = modelID
	result, err := sc.Svc.PreviewSourceSelectionCounts(c.Request.Context(), params)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": "OK", "msg": "OK", "data": result})
}

func compactQueryValues(values []string) []string {
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

// ListModels GET /semantic-models
// Collection authorization is prepared by iampep.PrepareCoreRequestContext and
// applied by core before count/pagination. Tags are forwarded as list filters.
func (sc *SemanticModelController) ListModels(c *gin.Context) {
	pageSize := 20
	if ps := c.Query("page_size"); ps != "" {
		v, err := strconv.Atoi(ps)
		if err != nil || v <= 0 || v > 100 {
			c.JSON(http.StatusBadRequest, gin.H{"code": "ErrParamInvalid", "msg": i18n.T(c.Request.Context(), i18n.KeySessionPageSizeInvalid), "data": nil})
			return
		}
		pageSize = v
	}
	params := session.ListSemanticModelsRequest{
		PageSize:  pageSize,
		PageToken: c.Query("page_token"),
		Search:    c.Query("search"),
	}
	if tags := compactQueryValues(c.QueryArray("tags")); len(tags) > 0 {
		params.Tags = tags
	}
	result, err := sc.Svc.ListModels(c.Request.Context(), params)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": "OK", "msg": "OK", "data": result})
}

// ListModelTags GET /semantic-models/tags
// Collection authorization is prepared by iampep.PrepareCoreRequestContext and
// applied by core before aggregation.
func (sc *SemanticModelController) ListModelTags(c *gin.Context) {
	result, err := sc.Svc.ListModelTags(c.Request.Context(), session.ListSemanticModelsRequest{
		Search: c.Query("search"),
	})
	if err != nil {
		mapServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": "OK", "msg": "OK", "data": result})
}

// UpdateModel PUT /semantic-models/:model_id
func (sc *SemanticModelController) UpdateModel(c *gin.Context) {
	modelID, err := parseModelID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "ErrParamInvalid", "msg": i18n.PublicErrorMessage(c.Request.Context(), err, i18n.KeyParamInvalid), "data": nil})
		return
	}
	var params session.UpdateSemanticModelRequest
	if err := c.ShouldBindJSON(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "ErrParamInvalid", "msg": i18n.RenderPublicMessage(c.Request.Context(), i18n.TranslateValidatorError(c.Request.Context(), err)), "data": nil})
		return
	}
	params.ModelID = modelID
	if strings.TrimSpace(params.Name) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": "ErrParamInvalid", "msg": i18n.T(c.Request.Context(), i18n.KeyNameRequired), "data": nil})
		return
	}
	if err := sc.Svc.UpdateModel(c.Request.Context(), params); err != nil {
		mapServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": "OK", "msg": "OK", "data": gin.H{"updated": true}})
}

// DeleteModel DELETE /semantic-models/:model_id
func (sc *SemanticModelController) DeleteModel(c *gin.Context) {
	modelID, err := parseModelID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "ErrParamInvalid", "msg": i18n.PublicErrorMessage(c.Request.Context(), err, i18n.KeyParamInvalid), "data": nil})
		return
	}
	if err := sc.Svc.DeleteModel(c.Request.Context(), modelID); err != nil {
		mapServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": "OK", "msg": "OK", "data": gin.H{"deleted": true}})
}

// GetModel GET /semantic-models/:model_id
func (sc *SemanticModelController) GetModel(c *gin.Context) {
	modelID, err := parseModelID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "ErrParamInvalid", "msg": i18n.PublicErrorMessage(c.Request.Context(), err, i18n.KeyParamInvalid), "data": nil})
		return
	}
	model, err := sc.Svc.GetModel(c.Request.Context(), modelID)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": "OK", "msg": "OK", "data": model})
}

// ListSources GET /semantic-models/:model_id/sources
func (sc *SemanticModelController) ListSources(c *gin.Context) {
	modelID, err := parseModelID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "ErrParamInvalid", "msg": i18n.PublicErrorMessage(c.Request.Context(), err, i18n.KeyParamInvalid), "data": nil})
		return
	}
	params := session.ListSemanticModelSourcesParams{ModelID: modelID}
	if raw := strings.TrimSpace(c.Query("page")); raw != "" {
		page, parseErr := strconv.Atoi(raw)
		if parseErr != nil || page <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"code": "ErrParamInvalid", "msg": i18n.RenderPublicMessage(c.Request.Context(), i18n.NewError(i18n.KeyInvalidParam, map[string]any{"Name": "page"})), "data": nil})
			return
		}
		params.Page = page
	}
	if raw := strings.TrimSpace(c.Query("page_size")); raw != "" {
		pageSize, parseErr := strconv.Atoi(raw)
		if parseErr != nil || pageSize <= 0 || pageSize > 100 {
			c.JSON(http.StatusBadRequest, gin.H{"code": "ErrParamInvalid", "msg": i18n.RenderPublicMessage(c.Request.Context(), i18n.NewError(i18n.KeyInvalidParam, map[string]any{"Name": "page_size"})), "data": nil})
			return
		}
		params.PageSize = pageSize
	}
	result, err := sc.Svc.ListSources(c.Request.Context(), params)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": "OK", "msg": "OK", "data": result})
}

// CheckSourceExistence POST /semantic-models/:model_id/sources/existence
func (sc *SemanticModelController) CheckSourceExistence(c *gin.Context) {
	modelID, err := parseModelID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "ErrParamInvalid", "msg": i18n.PublicErrorMessage(c.Request.Context(), err, i18n.KeyParamInvalid), "data": nil})
		return
	}
	var params session.CheckSemanticModelSourceExistenceParams
	if err := c.ShouldBindJSON(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "ErrParamInvalid", "msg": i18n.RenderPublicMessage(c.Request.Context(), i18n.TranslateValidatorError(c.Request.Context(), err)), "data": nil})
		return
	}
	params.ModelID = modelID
	result, err := sc.Svc.CheckSourceExistence(c.Request.Context(), params)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": "OK", "msg": "OK", "data": result})
}

// BackfillLegacySources POST /semantic-models/:model_id/sources/backfill-legacy
func (sc *SemanticModelController) BackfillLegacySources(c *gin.Context) {
	modelID, err := parseModelID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "ErrParamInvalid", "msg": i18n.PublicErrorMessage(c.Request.Context(), err, i18n.KeyParamInvalid), "data": nil})
		return
	}
	if err := sc.Svc.BackfillLegacySources(c.Request.Context(), session.BackfillLegacyKnowledgeBaseSourcesParams{ModelID: int64(modelID)}); err != nil {
		mapServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": "OK", "msg": "OK", "data": session.MutationResponse{Updated: true}})
}

// GetSourceDocument GET /semantic-models/:model_id/sources/:source_row_id/document
func (sc *SemanticModelController) GetSourceDocument(c *gin.Context) {
	modelID, err := parseModelID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "ErrParamInvalid", "msg": i18n.PublicErrorMessage(c.Request.Context(), err, i18n.KeyParamInvalid), "data": nil})
		return
	}
	sourceID := c.Param("source_row_id")
	if strings.TrimSpace(sourceID) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": "ErrParamInvalid", "msg": i18n.T(c.Request.Context(), i18n.KeySessionInvalidSourceRowID), "data": nil})
		return
	}
	result, err := sc.Svc.GetSourceDocument(c.Request.Context(), session.GetSemanticModelSourceDocumentParams{ModelID: modelID, SourceID: sourceID, SegmentVersionID: c.Query("segment_version_id")})
	if err != nil {
		mapServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": "OK", "msg": "OK", "data": result})
}

// UpdateSourceGovernance PATCH /semantic-models/:model_id/sources/:source_row_id/governance
func (sc *SemanticModelController) UpdateSourceGovernance(c *gin.Context) {
	modelID, err := parseModelID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "ErrParamInvalid", "msg": i18n.PublicErrorMessage(c.Request.Context(), err, i18n.KeyParamInvalid), "data": nil})
		return
	}
	sourceID := c.Param("source_row_id")
	if sourceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": "ErrParamInvalid", "msg": i18n.T(c.Request.Context(), i18n.KeySessionInvalidSourceRowID), "data": nil})
		return
	}
	var params session.UpdateSemanticModelSourceGovernanceParams
	if err := c.ShouldBindJSON(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "ErrParamInvalid", "msg": i18n.RenderPublicMessage(c.Request.Context(), i18n.TranslateValidatorError(c.Request.Context(), err)), "data": nil})
		return
	}
	params.ModelID = modelID
	params.SourceID = sourceID
	result, err := sc.Svc.UpdateSourceGovernance(c.Request.Context(), params)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": "OK", "msg": "OK", "data": result})
}

// ImportInitialSegments POST /semantic-models/:model_id/sources/:source_row_id/segments/import-initial
func (sc *SemanticModelController) ImportInitialSegments(c *gin.Context) {
	modelID, sourceID, ok := parseSourceMutationPath(c)
	if !ok {
		return
	}
	var params session.ImportInitialSemanticModelSegmentsParams
	if err := c.ShouldBindJSON(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "ErrParamInvalid", "msg": i18n.RenderPublicMessage(c.Request.Context(), i18n.TranslateValidatorError(c.Request.Context(), err)), "data": nil})
		return
	}
	params.ModelID = modelID
	params.SourceID = sourceID
	result, err := sc.Svc.ImportInitialSegments(c.Request.Context(), params)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": "OK", "msg": "OK", "data": result})
}

// UpdateSegment PATCH /semantic-models/:model_id/sources/:source_row_id/segments/:segment_id
func (sc *SemanticModelController) UpdateSegment(c *gin.Context) {
	modelID, sourceID, ok := parseSourceMutationPath(c)
	if !ok {
		return
	}
	segmentID := c.Param("segment_id")
	if segmentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": "ErrParamInvalid", "msg": i18n.T(c.Request.Context(), i18n.KeySessionInvalidSegmentID), "data": nil})
		return
	}
	var params session.UpdateSemanticModelSegmentParams
	if err := c.ShouldBindJSON(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "ErrParamInvalid", "msg": i18n.RenderPublicMessage(c.Request.Context(), i18n.TranslateValidatorError(c.Request.Context(), err)), "data": nil})
		return
	}
	params.ModelID = modelID
	params.SourceID = sourceID
	params.SegmentID = segmentID
	result, err := sc.Svc.UpdateSegment(c.Request.Context(), params)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": "OK", "msg": "OK", "data": result})
}

// CreateSegment POST /semantic-models/:model_id/sources/:source_row_id/segments
func (sc *SemanticModelController) CreateSegment(c *gin.Context) {
	modelID, sourceID, ok := parseSourceMutationPath(c)
	if !ok {
		return
	}
	var params session.CreateSemanticModelSegmentParams
	if err := c.ShouldBindJSON(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "ErrParamInvalid", "msg": i18n.RenderPublicMessage(c.Request.Context(), i18n.TranslateValidatorError(c.Request.Context(), err)), "data": nil})
		return
	}
	params.ModelID = modelID
	params.SourceID = sourceID
	result, err := sc.Svc.CreateSegment(c.Request.Context(), params)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": "OK", "msg": "OK", "data": result})
}

// UpdateSegmentEnabled PATCH /semantic-models/:model_id/sources/:source_row_id/segments/:segment_id/enabled
func (sc *SemanticModelController) UpdateSegmentEnabled(c *gin.Context) {
	modelID, sourceID, ok := parseSourceMutationPath(c)
	if !ok {
		return
	}
	segmentID := c.Param("segment_id")
	if segmentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": "ErrParamInvalid", "msg": i18n.T(c.Request.Context(), i18n.KeySessionInvalidSegmentID), "data": nil})
		return
	}
	var params session.UpdateSemanticModelSegmentEnabledParams
	if err := c.ShouldBindJSON(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "ErrParamInvalid", "msg": i18n.RenderPublicMessage(c.Request.Context(), i18n.TranslateValidatorError(c.Request.Context(), err)), "data": nil})
		return
	}
	params.ModelID = modelID
	params.SourceID = sourceID
	params.SegmentID = segmentID
	result, err := sc.Svc.UpdateSegmentEnabled(c.Request.Context(), params)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": "OK", "msg": "OK", "data": result})
}

// DeleteSegment DELETE /semantic-models/:model_id/sources/:source_row_id/segments/:segment_id
func (sc *SemanticModelController) DeleteSegment(c *gin.Context) {
	modelID, sourceID, ok := parseSourceMutationPath(c)
	if !ok {
		return
	}
	segmentID := c.Param("segment_id")
	if segmentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": "ErrParamInvalid", "msg": i18n.T(c.Request.Context(), i18n.KeySessionInvalidSegmentID), "data": nil})
		return
	}
	var params session.DeleteSemanticModelSegmentParams
	if err := c.ShouldBindJSON(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "ErrParamInvalid", "msg": i18n.RenderPublicMessage(c.Request.Context(), i18n.TranslateValidatorError(c.Request.Context(), err)), "data": nil})
		return
	}
	params.ModelID = modelID
	params.SourceID = sourceID
	params.SegmentID = segmentID
	result, err := sc.Svc.DeleteSegment(c.Request.Context(), params)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": "OK", "msg": "OK", "data": result})
}

// ReembedSegments POST /semantic-models/:model_id/sources/:source_row_id/segments/re-embedding
func (sc *SemanticModelController) ReembedSegments(c *gin.Context) {
	modelID, sourceID, ok := parseSourceMutationPath(c)
	if !ok {
		return
	}
	var params session.ReembedSemanticModelSegmentsParams
	if err := c.ShouldBindJSON(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "ErrParamInvalid", "msg": i18n.RenderPublicMessage(c.Request.Context(), i18n.TranslateValidatorError(c.Request.Context(), err)), "data": nil})
		return
	}
	params.ModelID = modelID
	params.SourceID = sourceID
	result, err := sc.Svc.ReembedSegments(c.Request.Context(), params)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": "OK", "msg": "OK", "data": result})
}

// SetCurrentSegmentVersion PATCH /semantic-models/:model_id/sources/:source_row_id/segment-versions/:version_id/current
func (sc *SemanticModelController) SetCurrentSegmentVersion(c *gin.Context) {
	modelID, sourceID, ok := parseSourceMutationPath(c)
	if !ok {
		return
	}
	versionID := c.Param("version_id")
	if versionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": "ErrParamInvalid", "msg": i18n.T(c.Request.Context(), i18n.KeySessionInvalidSegmentVersionID), "data": nil})
		return
	}
	var params session.SetCurrentSemanticModelSegmentVersionParams
	if err := c.ShouldBindJSON(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "ErrParamInvalid", "msg": i18n.RenderPublicMessage(c.Request.Context(), i18n.TranslateValidatorError(c.Request.Context(), err)), "data": nil})
		return
	}
	params.ModelID = modelID
	params.SourceID = sourceID
	params.VersionID = versionID
	result, err := sc.Svc.SetCurrentSegmentVersion(c.Request.Context(), params)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": "OK", "msg": "OK", "data": result})
}

func parseSourceMutationPath(c *gin.Context) (int, string, bool) {
	modelID, err := parseModelID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "ErrParamInvalid", "msg": i18n.PublicErrorMessage(c.Request.Context(), err, i18n.KeyParamInvalid), "data": nil})
		return 0, "", false
	}
	sourceID := c.Param("source_row_id")
	if sourceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": "ErrParamInvalid", "msg": i18n.T(c.Request.Context(), i18n.KeySessionInvalidSourceRowID), "data": nil})
		return 0, "", false
	}
	return modelID, sourceID, true
}

// DeleteSource DELETE /semantic-models/:model_id/sources/:source_row_id
func (sc *SemanticModelController) DeleteSource(c *gin.Context) {
	modelID, err := parseModelID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "ErrParamInvalid", "msg": i18n.PublicErrorMessage(c.Request.Context(), err, i18n.KeyParamInvalid), "data": nil})
		return
	}
	sourceID := c.Param("source_row_id")
	if sourceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": "ErrParamInvalid", "msg": i18n.T(c.Request.Context(), i18n.KeySessionInvalidSourceRowID), "data": nil})
		return
	}
	if err := sc.Svc.DeleteSource(c.Request.Context(), session.DeleteSemanticModelSourceParams{ModelID: modelID, SourceID: sourceID}); err != nil {
		mapServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": "OK", "msg": "OK", "data": session.MutationResponse{Deleted: true}})
}

// ListSourceJobs GET /semantic-models/:model_id/source-jobs
func (sc *SemanticModelController) ListSourceJobs(c *gin.Context) {
	modelID, err := parseModelID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "ErrParamInvalid", "msg": i18n.PublicErrorMessage(c.Request.Context(), err, i18n.KeyParamInvalid), "data": nil})
		return
	}
	result, err := sc.Svc.ListSourceJobs(c.Request.Context(), session.ListSemanticModelSourceJobsParams{ModelID: modelID})
	if err != nil {
		mapServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": "OK", "msg": "OK", "data": result})
}

// ReconcileSourceJobs POST /semantic-models/:model_id/source-jobs/reconcile
func (sc *SemanticModelController) ReconcileSourceJobs(c *gin.Context) {
	modelID, err := parseModelID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "ErrParamInvalid", "msg": i18n.PublicErrorMessage(c.Request.Context(), err, i18n.KeyParamInvalid), "data": nil})
		return
	}
	ctx := withSemanticModelCoreHeaders(c.Request.Context(), c)
	if err := sc.Svc.ReconcileKnowledgeBaseSourceJobs(ctx, session.ReconcileKnowledgeBaseSourceJobsParams{ModelID: int64(modelID)}); err != nil {
		mapServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": "OK", "msg": "OK", "data": session.MutationResponse{Updated: true}})
}

// ListEntries GET /semantic-models/:model_id/entries
func (sc *SemanticModelController) ListEntries(c *gin.Context) {
	modelID, err := parseModelID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "ErrParamInvalid", "msg": i18n.PublicErrorMessage(c.Request.Context(), err, i18n.KeyParamInvalid), "data": nil})
		return
	}
	kind := strings.TrimSpace(c.Query("kind"))
	if kind != "" && !session.ValidSemanticKinds[kind] {
		c.JSON(http.StatusBadRequest, gin.H{"code": "ErrParamInvalid", "msg": i18n.RenderPublicMessage(c.Request.Context(), i18n.NewError(i18n.KeySessionInvalidKind, map[string]any{"Kind": kind})), "data": nil})
		return
	}
	pageSize := 20
	if ps := c.Query("page_size"); ps != "" {
		v, err := strconv.Atoi(ps)
		if err != nil || v <= 0 || v > 100 {
			c.JSON(http.StatusBadRequest, gin.H{"code": "ErrParamInvalid", "msg": i18n.T(c.Request.Context(), i18n.KeySessionPageSizeInvalid), "data": nil})
			return
		}
		pageSize = v
	}
	result, err := sc.Svc.ListEntries(c.Request.Context(), session.ListSemanticEntriesRequest{
		ModelID:   modelID,
		Kind:      kind,
		PageSize:  pageSize,
		PageToken: c.Query("page_token"),
	})
	if err != nil {
		mapServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": "OK", "msg": "OK", "data": result})
}

// CreateEntry POST /semantic-models/:model_id/entries
func (sc *SemanticModelController) CreateEntry(c *gin.Context) {
	modelID, err := parseModelID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "ErrParamInvalid", "msg": i18n.PublicErrorMessage(c.Request.Context(), err, i18n.KeyParamInvalid), "data": nil})
		return
	}
	var params session.CreateSemanticEntryRequest
	if err := c.ShouldBindJSON(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "ErrParamInvalid", "msg": i18n.RenderPublicMessage(c.Request.Context(), i18n.TranslateValidatorError(c.Request.Context(), err)), "data": nil})
		return
	}
	params.ModelID = modelID
	if err := session.ValidateCreateSemanticEntryRequest(params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "ErrParamInvalid", "msg": i18n.PublicErrorMessage(c.Request.Context(), err, i18n.KeyParamInvalid), "data": nil})
		return
	}
	entry, err := sc.Svc.CreateEntry(c.Request.Context(), params)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"code": "OK", "msg": "OK", "data": entry})
}

// UpdateEntry PUT /semantic-models/:model_id/entries/:entry_id
func (sc *SemanticModelController) UpdateEntry(c *gin.Context) {
	modelID, err := parseModelID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "ErrParamInvalid", "msg": i18n.PublicErrorMessage(c.Request.Context(), err, i18n.KeyParamInvalid), "data": nil})
		return
	}
	entryID, err := parseEntryID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "ErrParamInvalid", "msg": i18n.PublicErrorMessage(c.Request.Context(), err, i18n.KeyParamInvalid), "data": nil})
		return
	}
	var params session.UpdateSemanticEntryRequest
	if err := c.ShouldBindJSON(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "ErrParamInvalid", "msg": i18n.RenderPublicMessage(c.Request.Context(), i18n.TranslateValidatorError(c.Request.Context(), err)), "data": nil})
		return
	}
	params.ModelID = modelID
	params.EntryID = entryID
	if err := session.ValidateCreateSemanticEntryRequest(session.CreateSemanticEntryRequest{
		Kind: params.Kind, Key: params.Key, Tables: params.Tables, Spec: params.Spec,
	}); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "ErrParamInvalid", "msg": i18n.PublicErrorMessage(c.Request.Context(), err, i18n.KeyParamInvalid), "data": nil})
		return
	}
	if err := sc.Svc.UpdateEntry(c.Request.Context(), params); err != nil {
		mapServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": "OK", "msg": "OK", "data": gin.H{"updated": true}})
}

// DeleteEntry DELETE /semantic-models/:model_id/entries/:entry_id
func (sc *SemanticModelController) DeleteEntry(c *gin.Context) {
	modelID, err := parseModelID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "ErrParamInvalid", "msg": i18n.PublicErrorMessage(c.Request.Context(), err, i18n.KeyParamInvalid), "data": nil})
		return
	}
	entryID, err := parseEntryID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "ErrParamInvalid", "msg": i18n.PublicErrorMessage(c.Request.Context(), err, i18n.KeyParamInvalid), "data": nil})
		return
	}
	if err := sc.Svc.DeleteEntry(c.Request.Context(), session.DeleteSemanticEntryRequest{
		ModelID: modelID, EntryID: entryID,
	}); err != nil {
		mapServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": "OK", "msg": "OK", "data": gin.H{"deleted": true}})
}

// Import POST /semantic-models/:model_id/import
func (sc *SemanticModelController) Import(c *gin.Context) {
	modelID, err := parseModelID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "ErrParamInvalid", "msg": i18n.PublicErrorMessage(c.Request.Context(), err, i18n.KeyParamInvalid), "data": nil})
		return
	}
	var params session.ImportSemanticModelRequest
	if err := c.ShouldBindJSON(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "ErrParamInvalid", "msg": i18n.RenderPublicMessage(c.Request.Context(), i18n.TranslateValidatorError(c.Request.Context(), err)), "data": nil})
		return
	}
	params.ModelID = modelID
	if len(params.Entries) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": "ErrParamInvalid", "msg": i18n.T(c.Request.Context(), i18n.KeySessionEntriesEmpty), "data": nil})
		return
	}
	keys := make(map[string]bool, len(params.Entries))
	filteredEntries := make([]session.CreateSemanticEntryRequest, 0, len(params.Entries))
	for i, e := range params.Entries {
		if session.IsDisabledLegacySemanticEntryTables(e.Tables) {
			continue
		}
		if err := session.ValidateCreateSemanticEntryRequest(e); err != nil {
			detailErr := i18n.WrapError(i18n.KeySessionEntryDetail, err, map[string]any{
				"Index":  i,
				"Detail": i18n.PublicErrorMessage(c.Request.Context(), err, i18n.KeyParamInvalid),
			})
			c.JSON(http.StatusBadRequest, gin.H{"code": "ErrParamInvalid", "msg": i18n.RenderPublicMessage(c.Request.Context(), detailErr), "data": nil})
			return
		}
		if keys[e.Key] {
			c.JSON(http.StatusConflict, gin.H{"code": "ErrConflict", "msg": i18n.RenderPublicMessage(c.Request.Context(), i18n.NewError(i18n.KeySessionDuplicateEntryKey, map[string]any{"Key": e.Key})), "data": nil})
			return
		}
		keys[e.Key] = true
		filteredEntries = append(filteredEntries, e)
	}
	params.Entries = filteredEntries
	ctx := withSemanticModelCoreHeaders(c.Request.Context(), c)
	result, err := sc.Svc.Import(ctx, params)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": "OK", "msg": "OK", "data": result})
}

// Export GET /semantic-models/:model_id/export
func (sc *SemanticModelController) Export(c *gin.Context) {
	modelID, err := parseModelID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "ErrParamInvalid", "msg": i18n.PublicErrorMessage(c.Request.Context(), err, i18n.KeyParamInvalid), "data": nil})
		return
	}
	result, err := sc.Svc.Export(c.Request.Context(), modelID)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": "OK", "msg": "OK", "data": result})
}

// Validate POST /semantic-models/:model_id/validate
func (sc *SemanticModelController) Validate(c *gin.Context) {
	modelID, err := parseModelID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "ErrParamInvalid", "msg": i18n.PublicErrorMessage(c.Request.Context(), err, i18n.KeyParamInvalid), "data": nil})
		return
	}
	result, err := sc.Svc.Validate(c.Request.Context(), modelID)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": "OK", "msg": "OK", "data": result})
}
