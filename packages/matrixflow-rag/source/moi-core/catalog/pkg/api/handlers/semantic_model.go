package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/matrixflow/moi-core/catalog/pkg/logging"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/matrixflow/moi-core/catalog/pkg/agentresource"
	ginctx "github.com/matrixflow/moi-core/catalog/pkg/api"
	"github.com/matrixflow/moi-core/catalog/pkg/iamcore"
	semanticservice "github.com/matrixflow/moi-core/catalog/pkg/service/semantic"
	"github.com/matrixflow/moi-core/catalog/pkg/service/storage/tenant"
	systemdisplay "github.com/matrixflow/moi-core/catalog/pkg/systemresourcedisplay"
	displayi18n "github.com/matrixflow/moi-core/catalog/pkg/systemresourcedisplay/i18n"
	catalogpb "github.com/matrixflow/moi-core/model/catalog"
	"github.com/matrixflow/moi-core/model/common"
	authzcore "github.com/matrixorigin/matrixflow/shared/authz/pkg/core"
)

const (
	semanticModelDisplayOwnerBackend      = "moi_backend"
	semanticModelDisplayKeyLiteralDefault = "moi_backend.resource.literal_default_text"

	semanticModelInternalDisplayCreateNotSupportedMessage = "knowledge_base_database_display_name is only supported when updating semantic models"
	semanticModelInternalDisplayDatabaseIDRequiredMessage = "knowledge base database display database_id is required"
	semanticModelInternalDisplayNameRequiredMessage       = "knowledge base database display display_name is required"
	semanticModelInternalDisplayDatabaseNotFoundMessage   = "Database not found"
	semanticModelBackendExecutionRequiredMessage          = "authenticated backend execution is required"
)

// SemanticModelHandler handles semantic model CRUD/import/export/validate APIs.
type SemanticModelHandler struct {
	pool            tenant.ConnectionPool
	storage         tenant.SemanticModelStorage
	creationService semanticModelCreationService
	logger          *zap.Logger
	// knowledgeDeleteHandler optionally cleans agent bindings/package versions in the same delete/rename TX.
	knowledgeDeleteHandler semanticKnowledgeBaseDeleteHandler
	iamAuthorizer          WorkflowIAMAuthorizer
	iamAccessFilter        WorkflowIAMAccessFilter
	iamRegistrar           semanticModelIAMRegistrar
}

type semanticModelCreationService interface {
	CreateModel(context.Context, semanticservice.ModelCreateCommand) (*tenant.SemanticModelRecord, error)
}

// semanticKnowledgeBaseDeleteHandler is the narrow agentresource owner surface used by delete/rename.
type semanticKnowledgeBaseDeleteHandler interface {
	HandleSemanticKnowledgeBaseDeleted(ctx context.Context, workspaceID string, modelID int64, modelName, userID string) (agentresource.SemanticKnowledgeBaseDeleteStats, error)
	HandleSemanticKnowledgeBaseRenamed(ctx context.Context, workspaceID string, modelID int64, oldName, newName string) (agentresource.SemanticKnowledgeBaseDeleteStats, error)
}

type semanticModelIAMRegistrar interface {
	Register(context.Context, iamcore.SemanticModelResourceRegisterRequest) (*tenant.IAMResourceOwnershipResult, error)
	RegisterAuthorized(context.Context, iamcore.SemanticModelResourceRegisterRequest, string) (*tenant.IAMResourceOwnershipResult, error)
	BeginDelete(context.Context, iamcore.SemanticModelResourceDeleteRequest) (*tenant.IAMResourceOwnershipResult, error)
	FinalizeDelete(context.Context, iamcore.SemanticModelResourceDeleteRequest, int64, string) error
}

type semanticEntryBatchStorage interface {
	CreateSemanticEntriesBatch(ctx context.Context, entries []*tenant.SemanticEntryRecord) error
}

type semanticModelDatabaseDisplayStorage interface {
	systemdisplay.MappingWriter
	GetDatabase(ctx context.Context, databaseID int64) (*catalogpb.Database, error)
}

// NewSemanticModelHandler creates semantic model HTTP handler.
func NewSemanticModelHandler(pool tenant.ConnectionPool, storage tenant.SemanticModelStorage, log *zap.Logger) *SemanticModelHandler {
	if log == nil {
		log = logging.NewDefaultLogger()
	}
	return &SemanticModelHandler{pool: pool, storage: storage, logger: log}
}

// WithSemanticKnowledgeBaseDeleteHandler injects the agentresource owner used by delete consistency.
func (h *SemanticModelHandler) WithSemanticKnowledgeBaseDeleteHandler(handler semanticKnowledgeBaseDeleteHandler) *SemanticModelHandler {
	if h == nil {
		return h
	}
	h.knowledgeDeleteHandler = handler
	return h
}

// WithCreationService injects the single semantic-model creation entry used by
// both the HTTP API and aggregate setup services.
func (h *SemanticModelHandler) WithCreationService(service semanticModelCreationService) *SemanticModelHandler {
	if h == nil {
		return h
	}
	h.creationService = service
	return h
}

func (h *SemanticModelHandler) WithIAM(authorizer WorkflowIAMAuthorizer, filter WorkflowIAMAccessFilter, registrar semanticModelIAMRegistrar) *SemanticModelHandler {
	h.iamAuthorizer, h.iamAccessFilter, h.iamRegistrar = authorizer, filter, registrar
	return h
}

func (h *SemanticModelHandler) iamContext(c *gin.Context, workspaceID string) (context.Context, bool) {
	if h == nil || h.pool == nil {
		writeCatalogAPIError(c, http.StatusServiceUnavailable, common.ErrorCode_UNAVAILABLE, "")
		return nil, false
	}
	tm, err := h.pool.GetTransactionManager(c.Request.Context(), workspaceID)
	if err != nil {
		writeCatalogAPIError(c, http.StatusServiceUnavailable, common.ErrorCode_UNAVAILABLE, "")
		return nil, false
	}
	return tenant.WithTransactionManager(c.Request.Context(), tm), true
}

func semanticModelCorrelation(c *gin.Context, operation string) (string, string, bool) {
	base := firstNonEmptyString(c.GetHeader("X-Request-ID"), ginctx.GetRequestID(c), ginctx.GetTraceID(c))
	traceID := firstNonEmptyString(c.GetHeader("X-Trace-ID"), ginctx.GetTraceID(c), base)
	if base == "" || traceID == "" {
		return "", "", false
	}
	return iamcoreRequestID(base, operation), traceID, true
}

func iamcoreRequestID(base, operation string) string {
	base, operation = strings.TrimSpace(base), strings.TrimSpace(operation)
	suffix := "." + operation
	if len(base)+len(suffix) <= 128 {
		return base + suffix
	}
	digest := sha256.Sum256([]byte(base + "\x00" + operation))
	return operation + "." + hex.EncodeToString(digest[:])
}

func resourceCreateLifecycleRequestID(base, resourceType string, stableParts ...string) string {
	parts := make([]string, 0, len(stableParts)+1)
	parts = append(parts, strings.TrimSpace(resourceType))
	for _, part := range stableParts {
		parts = append(parts, strings.TrimSpace(part))
	}
	digest := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return iamcoreRequestID(base, parts[0]+"-create-"+hex.EncodeToString(digest[:8]))
}

func (h *SemanticModelHandler) authorize(c *gin.Context, workspaceID, principalID, actionID string, modelID int64) bool {
	_, ok := h.authorizeDecision(c, workspaceID, principalID, actionID, modelID)
	return ok
}

func (h *SemanticModelHandler) authorizeDecision(c *gin.Context, workspaceID, principalID, actionID string, modelID int64) (authzcore.Decision, bool) {
	if h == nil || h.iamAuthorizer == nil {
		writeCatalogAPIError(c, http.StatusServiceUnavailable, common.ErrorCode_UNAVAILABLE, "")
		return authzcore.Decision{}, false
	}
	ctx, ok := h.iamContext(c, workspaceID)
	if !ok {
		return authzcore.Decision{}, false
	}
	requestID, traceID, ok := semanticModelCorrelation(c, "semantic-model-authorize")
	if !ok {
		writeCatalogAPIError(c, http.StatusInternalServerError, common.ErrorCode_INTERNAL, "")
		return authzcore.Decision{}, false
	}
	resourceType, resourceID := iamcore.IAMResourceSemanticModel, strconv.FormatInt(modelID, 10)
	if actionID == iamcore.IAMActionSemanticModelCreate {
		resourceType, resourceID = iamcore.IAMResourceWorkspace, workspaceID
	}
	decision, err := h.iamAuthorizer.AuthorizeUserResourceDecision(ctx, workspaceID, principalID, workflowExplicitRole(c), actionID, resourceType, resourceID, requestID, traceID)
	if err != nil {
		writeSemanticModelIAMError(c, err)
		return authzcore.Decision{}, false
	}
	return decision, true
}

func writeSemanticModelIAMError(c *gin.Context, err error) {
	if errors.Is(err, authzcore.ErrDenied) {
		writeCatalogAPIError(c, http.StatusForbidden, common.ErrorCode_PERMISSION_DENIED, "")
		return
	}
	writeCatalogAPIError(c, http.StatusServiceUnavailable, common.ErrorCode_UNAVAILABLE, "")
}

type semanticModelCreateRequest struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Tables      json.RawMessage `json:"tables"`
	Files       json.RawMessage `json:"files,omitempty"`
}

type semanticModelCreateInternalRequest struct {
	semanticModelCreateRequest
	KnowledgeBaseDatabaseDisplayName *semanticModelDatabaseDisplayNameInternalRequest `json:"knowledge_base_database_display_name,omitempty"`
}

type semanticModelUpdateRequest struct {
	Name                             string                                           `json:"name"`
	Description                      string                                           `json:"description"`
	Tables                           json.RawMessage                                  `json:"tables"`
	Files                            json.RawMessage                                  `json:"files,omitempty"`
	KnowledgeBaseDatabaseDisplayName *semanticModelDatabaseDisplayNameInternalRequest `json:"knowledge_base_database_display_name,omitempty"`
}

type semanticModelDatabaseDisplayNameInternalRequest struct {
	DatabaseID  int64  `json:"database_id"`
	DisplayName string `json:"display_name"`
}

type semanticEntryUpsertRequest struct {
	Kind   string          `json:"kind"`
	Key    string          `json:"key"`
	Tables []string        `json:"tables"`
	Spec   json.RawMessage `json:"spec"`
}

type semanticModelImportRequest struct {
	Name        string                       `json:"name"`
	Description string                       `json:"description"`
	Tables      json.RawMessage              `json:"tables"`
	Files       json.RawMessage              `json:"files,omitempty"`
	Entries     []semanticEntryUpsertRequest `json:"entries"`
}

type semanticModelImportBatchResponse struct {
	Items []*semanticModelResponse `json:"items"`
	Total int64                    `json:"total"`
}

type semanticModelResponse struct {
	ID           int64           `json:"id"`
	Name         string          `json:"name"`
	Description  string          `json:"description,omitempty"`
	Tables       json.RawMessage `json:"tables"`
	Files        json.RawMessage `json:"files,omitempty"`
	TableSetHash string          `json:"table_set_hash"`
	CreatedBy    string          `json:"created_by,omitempty"`
	UpdatedBy    string          `json:"updated_by,omitempty"`
	CreatedAt    int64           `json:"created_at"`
	UpdatedAt    int64           `json:"updated_at"`
}

type semanticEntryResponse struct {
	ID        int64           `json:"id"`
	ModelID   int64           `json:"model_id"`
	Kind      string          `json:"kind"`
	Key       string          `json:"key"`
	Tables    []string        `json:"tables,omitempty"`
	Spec      json.RawMessage `json:"spec"`
	CreatedBy string          `json:"created_by,omitempty"`
	UpdatedBy string          `json:"updated_by,omitempty"`
	CreatedAt int64           `json:"created_at"`
	UpdatedAt int64           `json:"updated_at"`
}

type semanticModelListResponse struct {
	Items         []*semanticModelResponse `json:"items"`
	Total         int64                    `json:"total"`
	NextPageToken string                   `json:"next_page_token,omitempty"`
}

type semanticModelTagStatResponse struct {
	Tag   string `json:"tag"`
	Count int64  `json:"count"`
}

type semanticModelTagListResponse struct {
	Items []semanticModelTagStatResponse `json:"items"`
}

type semanticEntryListResponse struct {
	Items         []*semanticEntryResponse `json:"items"`
	Total         int64                    `json:"total"`
	NextPageToken string                   `json:"next_page_token,omitempty"`
}

type semanticEntryInput struct {
	Kind   string
	Key    string
	Tables []string
	Spec   json.RawMessage
}

func (h *SemanticModelHandler) parseWorkspaceID(c *gin.Context) (string, error) {
	workspaceID := strings.TrimSpace(c.Param("id"))
	if workspaceID == "" {
		ginctx.WriteError(c, http.StatusBadRequest, common.ErrorCode_INVALID_ARGUMENT, "Workspace ID is required")
		return "", errors.New("workspace ID is required")
	}
	return workspaceID, nil
}

func (h *SemanticModelHandler) parseModelID(c *gin.Context) (int64, error) {
	modelIDStr := strings.TrimSpace(c.Param("model_id"))
	if modelIDStr == "" {
		ginctx.WriteError(c, http.StatusBadRequest, common.ErrorCode_INVALID_ARGUMENT, "Model ID is required")
		return 0, errors.New("model ID is required")
	}
	modelID, err := strconv.ParseInt(modelIDStr, 10, 64)
	if err != nil || modelID <= 0 {
		ginctx.WriteError(c, http.StatusBadRequest, common.ErrorCode_INVALID_ARGUMENT, "Invalid model ID format")
		return 0, errors.New("invalid model ID")
	}
	return modelID, nil
}

func (h *SemanticModelHandler) parseEntryID(c *gin.Context) (int64, error) {
	entryIDStr := strings.TrimSpace(c.Param("entry_id"))
	if entryIDStr == "" {
		ginctx.WriteError(c, http.StatusBadRequest, common.ErrorCode_INVALID_ARGUMENT, "Entry ID is required")
		return 0, errors.New("entry ID is required")
	}
	entryID, err := strconv.ParseInt(entryIDStr, 10, 64)
	if err != nil || entryID <= 0 {
		ginctx.WriteError(c, http.StatusBadRequest, common.ErrorCode_INVALID_ARGUMENT, "Invalid entry ID format")
		return 0, errors.New("invalid entry ID")
	}
	return entryID, nil
}

func parsePaginationOptions(c *gin.Context) ([]tenant.ListOption, int32, string) {
	pageSize := tenant.DefaultPageSize
	pageToken := strings.TrimSpace(c.Query("page_token"))
	if pageSizeStr := strings.TrimSpace(c.Query("page_size")); pageSizeStr != "" {
		if parsed, err := strconv.ParseInt(pageSizeStr, 10, 32); err == nil && parsed > 0 {
			if parsed > int64(tenant.MaxPageSize) {
				parsed = int64(tenant.MaxPageSize)
			}
			pageSize = int32(parsed)
		}
	}

	opts := []tenant.ListOption{tenant.WithPageSize(pageSize)}
	if pageToken != "" {
		opts = append(opts, tenant.WithPageToken(pageToken))
	}
	return opts, pageSize, pageToken
}

func nextPageTokenForList(current string, returned int, total int64) string {
	if returned <= 0 || total <= 0 {
		return ""
	}
	offset, _ := tenant.DecodePageToken(current)
	nextOffset := offset + int64(returned)
	if nextOffset >= total {
		return ""
	}
	return tenant.EncodePageToken(nextOffset)
}

func compactQueryValues(values []string) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{})
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

// semanticModelTableEntry represents a structured table entry in the new tables JSON format.
type semanticModelTableEntry struct {
	DBName     string   `json:"db_name"`
	TableNames []string `json:"table_names"`
	Parents    []string `json:"parents,omitempty"`
}

// extractTableNamesFromJSON extracts all table_names from the structured tables JSON.
// Expected format: [{"db_name": "...", "table_names": ["..."], "parents": ["..."]}]
func extractTableNamesFromJSON(tablesJSON json.RawMessage) ([]string, error) {
	if len(tablesJSON) == 0 {
		return nil, nil // tables is optional — empty means no tables configured yet
	}

	var entries []semanticModelTableEntry
	if err := json.Unmarshal(tablesJSON, &entries); err != nil {
		return nil, fmt.Errorf("invalid tables format: expected structured array [{db_name, table_names, parents}], got: %w", err)
	}
	if len(entries) == 0 {
		return nil, nil // empty tables array is allowed — no tables configured yet
	}

	// Reject legacy flat string array: if no entry has table_names, it's likely the old format.
	hasStructured := false
	for _, entry := range entries {
		if len(entry.TableNames) > 0 {
			hasStructured = true
			break
		}
	}
	if !hasStructured {
		return nil, fmt.Errorf("invalid tables format: each entry must have non-empty table_names; legacy flat string array is no longer supported")
	}

	out := make([]string, 0)
	seen := make(map[string]struct{})
	for _, entry := range entries {
		for _, name := range entry.TableNames {
			normalized := semanticservice.NormalizeModelTableNameForDB(entry.DBName, name)
			if normalized == "" {
				return nil, fmt.Errorf("table name must not be empty")
			}
			if _, exists := seen[normalized]; exists {
				return nil, fmt.Errorf("duplicate table %q", normalized)
			}
			seen[normalized] = struct{}{}
			out = append(out, normalized)
		}
	}
	sort.Strings(out)
	return out, nil
}

func normalizeEntryTables(raw []string) []string {
	if len(raw) == 0 {
		return nil
	}
	out := make([]string, 0, len(raw))
	seen := make(map[string]struct{}, len(raw))
	for _, item := range raw {
		name := normalizeSemanticTableName(item)
		if name == "" {
			continue
		}
		if _, exists := seen[name]; exists {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func normalizeSemanticTableName(raw string) string {
	name := strings.Trim(strings.TrimSpace(strings.ToLower(raw)), "`\"")
	if dot := strings.LastIndex(name, "."); dot >= 0 {
		name = name[dot+1:]
	}
	return name
}

func semanticTableSetHash(tableNames []string) string {
	sum := sha256.Sum256([]byte(strings.Join(tableNames, ",")))
	return fmt.Sprintf("%x", sum[:])
}

func toModelResponse(model *tenant.SemanticModelRecord) *semanticModelResponse {
	if model == nil {
		return nil
	}
	return &semanticModelResponse{
		ID:           model.ID,
		Name:         model.Name,
		Description:  model.Description,
		Tables:       model.Tables,
		Files:        model.Files,
		TableSetHash: model.TableSetHash,
		CreatedBy:    model.CreatedBy,
		UpdatedBy:    model.UpdatedBy,
		CreatedAt:    model.CreatedAt,
		UpdatedAt:    model.UpdatedAt,
	}
}

func toEntryResponse(entry *tenant.SemanticEntryRecord) *semanticEntryResponse {
	if entry == nil {
		return nil
	}
	return &semanticEntryResponse{
		ID:        entry.ID,
		ModelID:   entry.ModelID,
		Kind:      entry.Kind,
		Key:       entry.KeyName,
		Tables:    append([]string(nil), entry.Tables...),
		Spec:      append(json.RawMessage(nil), entry.Spec...),
		CreatedBy: entry.CreatedBy,
		UpdatedBy: entry.UpdatedBy,
		CreatedAt: entry.CreatedAt,
		UpdatedAt: entry.UpdatedAt,
	}
}

func validateSemanticModelSpec(tablesRaw json.RawMessage, entries []semanticEntryInput) error {
	modelTables, err := semanticservice.NormalizeModelTablesFromJSON(tablesRaw)
	if err != nil {
		return err
	}
	inputs := make([]semanticservice.EntryInput, 0, len(entries))
	for _, entry := range entries {
		inputs = append(inputs, semanticservice.EntryInput{
			Kind:   entry.Kind,
			Key:    entry.Key,
			Tables: semanticservice.NormalizeEntryTables(entry.Tables),
			Spec:   bytesTrimSpace(entry.Spec),
		})
	}
	return semanticservice.ValidateSpec(modelTables, inputs)
}

func bytesTrimSpace(raw []byte) []byte {
	if len(raw) == 0 {
		return nil
	}
	return []byte(strings.TrimSpace(string(raw)))
}

func mapSemanticStorageError(c *gin.Context, err error, internalMessage string, metadata ...map[string]string) {
	if err == nil {
		return
	}
	if errors.Is(err, tenant.ErrSemanticModelNotFound) {
		writeSemanticReasonError(c, http.StatusNotFound, common.ErrorCode_NOT_FOUND, err.Error(), "SESSION_SEMANTIC_MODEL_NOT_FOUND", nil)
		return
	}
	if errors.Is(err, tenant.ErrSemanticEntryNotFound) {
		entryMetadata := firstSemanticErrorMetadata(metadata)
		if entryMetadata["entry_id"] != "" {
			writeSemanticReasonError(c, http.StatusNotFound, common.ErrorCode_NOT_FOUND, err.Error(), "SESSION_ENTRY_NOT_FOUND", entryMetadata)
			return
		}
		ginctx.WriteError(c, http.StatusNotFound, common.ErrorCode_NOT_FOUND, err.Error())
		return
	}
	if errors.Is(err, tenant.ErrSemanticModelAlreadyExist) || errors.Is(err, tenant.ErrSemanticEntryAlreadyExist) {
		ginctx.WriteError(c, http.StatusConflict, common.ErrorCode_ALREADY_EXISTS, err.Error())
		return
	}
	ginctx.WriteError(c, http.StatusInternalServerError, common.ErrorCode_INTERNAL, internalMessage)
}

func writeSemanticReasonError(c *gin.Context, statusCode int, code common.ErrorCode, message string, reason string, metadata map[string]string) {
	ginctx.WriteErrorWithDetails(c, statusCode, code, message, common.NewErrorInfoDetails(reason, semanticservice.ErrorDomainSession, metadata))
}

func writeSemanticValidationError(c *gin.Context, err error) bool {
	reason, domain, metadata, ok := semanticservice.ValidationErrorInfo(err)
	if !ok {
		return false
	}
	ginctx.WriteErrorWithDetails(c, http.StatusBadRequest, common.ErrorCode_INVALID_ARGUMENT, err.Error(), common.NewErrorInfoDetails(reason, domain, metadata))
	return true
}

func firstSemanticErrorMetadata(values []map[string]string) map[string]string {
	if len(values) == 0 || values[0] == nil {
		return nil
	}
	return values[0]
}

func (h *SemanticModelHandler) loadAllSemanticEntries(ctx context.Context, modelID int64) ([]*tenant.SemanticEntryRecord, error) {
	pageToken := ""
	all := make([]*tenant.SemanticEntryRecord, 0)
	for {
		opts := []tenant.ListOption{tenant.WithPageSize(tenant.MaxPageSize)}
		if pageToken != "" {
			opts = append(opts, tenant.WithPageToken(pageToken))
		}
		items, total, err := h.storage.ListSemanticEntries(ctx, modelID, "", opts...)
		if err != nil {
			return nil, err
		}
		all = append(all, items...)
		next := nextPageTokenForList(pageToken, len(items), total)
		if next == "" {
			break
		}
		pageToken = next
	}
	return all, nil
}

// Create POST /api/v1/workspaces/:id/semantic-models
//
// @Summary 创建语义模型
// @Description 在指定 workspace 下创建语义模型
// @Tags Semantic Model 管理
// @Accept json
// @Produce json
// @Param id path string true "Workspace ID"
// @Param request body semanticModelCreateRequest true "语义模型信息"
// @Success 201 {object} semanticModelResponse "创建成功"
// @Failure 400 {object} ginctx.ErrorResponse "参数错误"
// @Failure 401 {object} ginctx.ErrorResponse "未认证"
// @Failure 409 {object} ginctx.ErrorResponse "模型已存在"
// @Failure 500 {object} ginctx.ErrorResponse "内部错误"
// @Security ApiKeyAuth
// @Router /api/v1/workspaces/{id}/semantic-models [post]
func (h *SemanticModelHandler) Create(c *gin.Context) {
	workspaceID, err := h.parseWorkspaceID(c)
	if err != nil {
		return
	}
	uid := ginctx.GetUserID(c)
	if uid == "" {
		ginctx.WriteError(c, http.StatusUnauthorized, common.ErrorCode_UNAUTHENTICATED, "User not authenticated")
		return
	}
	createDecision, ok := h.authorizeDecision(c, workspaceID, uid, iamcore.IAMActionSemanticModelCreate, 0)
	if !ok {
		return
	}

	var req semanticModelCreateInternalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ginctx.WriteError(c, http.StatusBadRequest, common.ErrorCode_INVALID_ARGUMENT, "Invalid request body")
		return
	}
	if req.KnowledgeBaseDatabaseDisplayName != nil {
		ginctx.WriteError(c, http.StatusBadRequest, common.ErrorCode_INVALID_ARGUMENT, semanticModelInternalDisplayCreateNotSupportedMessage)
		return
	}
	requestID, traceID, ok := semanticModelCorrelation(c, "semantic-model-create")
	if !ok || h.creationService == nil {
		writeCatalogAPIError(c, http.StatusServiceUnavailable, common.ErrorCode_UNAVAILABLE, "")
		return
	}
	created, err := h.creationService.CreateModel(c.Request.Context(), semanticservice.ModelCreateCommand{
		WorkspaceID:             workspaceID,
		PrincipalID:             uid,
		RoleCandidateID:         workflowExplicitRole(c),
		VerifiedEffectiveRoleID: createDecision.VerifiedEffectiveRoleID,
		RequestID:               requestID,
		TraceID:                 traceID,
		Name:                    req.Name,
		Description:             req.Description,
		Tables:                  req.Tables,
		Files:                   req.Files,
	})
	if err != nil {
		if writeSemanticValidationError(c, err) {
			return
		}
		mapSemanticStorageError(c, err, "Failed to create semantic model")
		return
	}
	ginctx.WriteSuccessWithStatus(c, http.StatusCreated, toModelResponse(created))
}

// List GET /api/v1/workspaces/:id/semantic-models
//
// @Summary 列出语义模型
// @Description 列出指定 workspace 下的语义模型
// @Tags Semantic Model 管理
// @Produce json
// @Param id path string true "Workspace ID"
// @Param page_size query int false "分页大小"
// @Param page_token query string false "分页游标"
// @Param search query string false "模糊搜索关键词（匹配名称或描述）"
// @Success 200 {object} semanticModelListResponse "查询成功"
// @Failure 400 {object} ginctx.ErrorResponse "参数错误"
// @Failure 401 {object} ginctx.ErrorResponse "未认证"
// @Failure 500 {object} ginctx.ErrorResponse "内部错误"
// @Security ApiKeyAuth
// @Router /api/v1/workspaces/{id}/semantic-models [get]
func (h *SemanticModelHandler) List(c *gin.Context) {
	ctx := c.Request.Context()
	workspaceID, err := h.parseWorkspaceID(c)
	if err != nil {
		return
	}
	uid := ginctx.GetUserID(c)
	if uid == "" {
		ginctx.WriteError(c, http.StatusUnauthorized, common.ErrorCode_UNAUTHENTICATED, "User not authenticated")
		return
	}
	if h.iamAccessFilter == nil {
		writeCatalogAPIError(c, http.StatusServiceUnavailable, common.ErrorCode_UNAVAILABLE, "")
		return
	}

	opts, _, pageToken := parsePaginationOptions(c)

	// Fuzzy search on name or description
	if search := strings.TrimSpace(c.Query("search")); search != "" {
		opts = append(opts, tenant.WithFilter("search", []string{search}, true))
	}
	if tags := compactQueryValues(c.QueryArray("tags")); len(tags) > 0 {
		opts = append(opts, tenant.WithFilter("tags", tags, false))
	}

	tm, err := h.pool.GetTransactionManager(ctx, workspaceID)
	if err != nil {
		h.logger.Error("List: GetTransactionManager failed",
			zap.String("workspace_id", workspaceID), zap.Error(err))
		writeCatalogAPIError(c, http.StatusInternalServerError, common.ErrorCode_INTERNAL, "")
		return
	}
	ctx = tenant.WithTransactionManager(ctx, tm)
	plan, err := h.iamAccessFilter.ResourceAccessFilter(ctx, iamcore.IAMResourceAccessFilterRequest{WorkspaceID: workspaceID, PrincipalID: uid, RoleID: workflowExplicitRole(c), ActorType: authzcore.ActorTypeUser, ResourceType: iamcore.IAMResourceSemanticModel, ActionID: iamcore.IAMActionSemanticModelRead})
	if err != nil {
		writeSemanticModelIAMError(c, err)
		return
	}
	if !plan.AllResources {
		opts = append(opts, tenant.WithFilter("ids", append([]string(nil), plan.ResourceIDs...), false))
	}

	var (
		items []*tenant.SemanticModelRecord
		total int64
	)
	err = tm.RunInTx(ctx, func(txCtx context.Context) error {
		items, total, err = h.storage.ListSemanticModels(txCtx, opts...)
		return err
	})
	if err != nil {
		mapSemanticStorageError(c, err, "Failed to list semantic models")
		return
	}

	respItems := make([]*semanticModelResponse, 0, len(items))
	for _, item := range items {
		respItems = append(respItems, toModelResponse(item))
	}
	ginctx.WriteSuccess(c, &semanticModelListResponse{
		Items:         respItems,
		Total:         total,
		NextPageToken: nextPageTokenForList(pageToken, len(items), total),
	})
}

// ListTags GET /api/v1/workspaces/:id/semantic-models/tags
//
// @Summary 列出语义模型标签聚合
// @Description 列出指定 workspace 下语义模型的 KB 级标签聚合，可按 search 缩小范围
// @Tags Semantic Model 管理
// @Produce json
// @Param id path string true "Workspace ID"
// @Param search query string false "模糊搜索关键词（匹配名称或描述）"
// @Success 200 {object} semanticModelTagListResponse "查询成功"
// @Failure 401 {object} ginctx.ErrorResponse "未认证"
// @Failure 500 {object} ginctx.ErrorResponse "内部错误"
// @Security ApiKeyAuth
// @Router /api/v1/workspaces/{id}/semantic-models/tags [get]
func (h *SemanticModelHandler) ListTags(c *gin.Context) {
	ctx := c.Request.Context()
	workspaceID, err := h.parseWorkspaceID(c)
	if err != nil {
		return
	}
	uid := ginctx.GetUserID(c)
	if uid == "" {
		ginctx.WriteError(c, http.StatusUnauthorized, common.ErrorCode_UNAUTHENTICATED, "User not authenticated") // i18n-allow: existing semantic model API error contract
		return
	}
	if h.iamAccessFilter == nil {
		writeCatalogAPIError(c, http.StatusServiceUnavailable, common.ErrorCode_UNAVAILABLE, "")
		return
	}

	var opts []tenant.ListOption
	if search := strings.TrimSpace(c.Query("search")); search != "" {
		opts = append(opts, tenant.WithFilter("search", []string{search}, true))
	}

	tm, err := h.pool.GetTransactionManager(ctx, workspaceID)
	if err != nil {
		h.logger.Error("ListTags: GetTransactionManager failed", zap.String("workspace_id", workspaceID), zap.Error(err))
		ginctx.WriteError(c, http.StatusInternalServerError, common.ErrorCode_INTERNAL, "Failed to list semantic model tags") // i18n-allow: existing semantic model API error contract
		return
	}
	ctx = tenant.WithTransactionManager(ctx, tm)
	plan, err := h.iamAccessFilter.ResourceAccessFilter(ctx, iamcore.IAMResourceAccessFilterRequest{WorkspaceID: workspaceID, PrincipalID: uid, RoleID: workflowExplicitRole(c), ActorType: authzcore.ActorTypeUser, ResourceType: iamcore.IAMResourceSemanticModel, ActionID: iamcore.IAMActionSemanticModelRead})
	if err != nil {
		writeSemanticModelIAMError(c, err)
		return
	}
	if !plan.AllResources {
		opts = append(opts, tenant.WithFilter("ids", append([]string(nil), plan.ResourceIDs...), false))
	}

	var stats []tenant.SemanticModelTagStat
	err = tm.RunInTx(ctx, func(txCtx context.Context) error {
		stats, err = h.storage.ListSemanticModelTags(txCtx, opts...)
		return err
	})
	if err != nil {
		mapSemanticStorageError(c, err, "Failed to list semantic model tags")
		return
	}

	items := make([]semanticModelTagStatResponse, 0, len(stats))
	for _, stat := range stats {
		items = append(items, semanticModelTagStatResponse{Tag: stat.Tag, Count: stat.Count})
	}
	ginctx.WriteSuccess(c, &semanticModelTagListResponse{Items: items})
}

// Get GET /api/v1/workspaces/:id/semantic-models/:model_id
//
// @Summary 获取语义模型
// @Description 获取指定语义模型详情
// @Tags Semantic Model 管理
// @Produce json
// @Param id path string true "Workspace ID"
// @Param model_id path int true "模型 ID"
// @Success 200 {object} semanticModelResponse "查询成功"
// @Failure 400 {object} ginctx.ErrorResponse "参数错误"
// @Failure 401 {object} ginctx.ErrorResponse "未认证"
// @Failure 404 {object} ginctx.ErrorResponse "模型不存在"
// @Failure 500 {object} ginctx.ErrorResponse "内部错误"
// @Security ApiKeyAuth
// @Router /api/v1/workspaces/{id}/semantic-models/{model_id} [get]
func (h *SemanticModelHandler) Get(c *gin.Context) {
	ctx := c.Request.Context()
	workspaceID, err := h.parseWorkspaceID(c)
	if err != nil {
		return
	}
	modelID, err := h.parseModelID(c)
	if err != nil {
		return
	}
	uid := ginctx.GetUserID(c)
	if uid == "" {
		writeCatalogAPIError(c, http.StatusUnauthorized, common.ErrorCode_UNAUTHENTICATED, "")
		return
	}
	if !h.authorize(c, workspaceID, uid, iamcore.IAMActionSemanticModelRead, modelID) {
		return
	}

	tm, err := h.pool.GetTransactionManager(ctx, workspaceID)
	if err != nil {
		h.logger.Error("Get: GetTransactionManager failed",
			zap.String("workspace_id", workspaceID), zap.Error(err))
		ginctx.WriteError(c, http.StatusInternalServerError, common.ErrorCode_INTERNAL, "Failed to get semantic model")
		return
	}
	ctx = tenant.WithTransactionManager(ctx, tm)

	var model *tenant.SemanticModelRecord
	err = tm.RunInTx(ctx, func(txCtx context.Context) error {
		model, err = h.storage.GetSemanticModel(txCtx, modelID)
		return err
	})
	if err != nil {
		mapSemanticStorageError(c, err, "Failed to get semantic model")
		return
	}
	ginctx.WriteSuccess(c, toModelResponse(model))
}

// Update PUT /api/v1/workspaces/:id/semantic-models/:model_id
//
// @Summary 更新语义模型
// @Description 更新指定语义模型。若名称变更，同一事务内将引用旧名称的非 disabled Agent Package 版本标记为 needs_configuration。
// @Tags Semantic Model 管理
// @Accept json
// @Produce json
// @Param id path string true "Workspace ID"
// @Param model_id path int true "模型 ID"
// @Param request body semanticModelUpdateRequest true "语义模型信息"
// @Success 200 {object} map[string]interface{} "更新成功"
// @Failure 400 {object} ginctx.ErrorResponse "参数错误"
// @Failure 401 {object} ginctx.ErrorResponse "未认证"
// @Failure 404 {object} ginctx.ErrorResponse "模型不存在"
// @Failure 500 {object} ginctx.ErrorResponse "内部错误"
// @Security ApiKeyAuth
// @Router /api/v1/workspaces/{id}/semantic-models/{model_id} [put]
func (h *SemanticModelHandler) Update(c *gin.Context) {
	ctx := c.Request.Context()
	workspaceID, err := h.parseWorkspaceID(c)
	if err != nil {
		return
	}
	modelID, err := h.parseModelID(c)
	if err != nil {
		return
	}
	uid := ginctx.GetUserID(c)
	if uid == "" {
		ginctx.WriteError(c, http.StatusUnauthorized, common.ErrorCode_UNAUTHENTICATED, "User not authenticated")
		return
	}
	if !h.authorize(c, workspaceID, uid, iamcore.IAMActionSemanticModelUpdate, modelID) {
		return
	}

	var req semanticModelUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ginctx.WriteError(c, http.StatusBadRequest, common.ErrorCode_INVALID_ARGUMENT, "Invalid request body")
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		ginctx.WriteError(c, http.StatusBadRequest, common.ErrorCode_INVALID_ARGUMENT, "name is required")
		return
	}
	if message := validateSemanticModelDatabaseDisplayNameRequest(req.KnowledgeBaseDatabaseDisplayName); message != "" {
		ginctx.WriteError(c, http.StatusBadRequest, common.ErrorCode_INVALID_ARGUMENT, message)
		return
	}
	if req.KnowledgeBaseDatabaseDisplayName != nil && !authorizeSemanticModelInternalDisplayUpdate(c) {
		return
	}
	tables, err := extractTableNamesFromJSON(req.Tables)
	if err != nil {
		ginctx.WriteError(c, http.StatusBadRequest, common.ErrorCode_INVALID_ARGUMENT, err.Error())
		return
	}

	tm, err := h.pool.GetTransactionManager(ctx, workspaceID)
	if err != nil {
		h.logger.Error("Update: GetTransactionManager failed",
			zap.String("workspace_id", workspaceID), zap.Error(err))
		ginctx.WriteError(c, http.StatusInternalServerError, common.ErrorCode_INTERNAL, "Failed to update semantic model")
		return
	}
	ctx = tenant.WithTransactionManager(ctx, tm)

	newName := strings.TrimSpace(req.Name)
	err = tm.RunInTx(ctx, func(txCtx context.Context) error {
		// Lock the model first so rename/delete cannot race on the same row.
		current, err := h.storage.GetSemanticModelForUpdate(txCtx, modelID)
		if err != nil {
			return err
		}
		if h.knowledgeDeleteHandler != nil {
			if _, err := h.knowledgeDeleteHandler.HandleSemanticKnowledgeBaseRenamed(txCtx, workspaceID, modelID, current.Name, newName); err != nil {
				return err
			}
		}
		if err := h.storage.UpdateSemanticModel(txCtx, &tenant.SemanticModelRecord{
			ID:           modelID,
			Name:         newName,
			Description:  strings.TrimSpace(req.Description),
			Tables:       req.Tables,
			Files:        req.Files,
			TableSetHash: semanticTableSetHash(tables),
			UpdatedBy:    uid,
		}); err != nil {
			return err
		}
		return h.ensureSemanticModelDatabaseDisplayName(txCtx, req.KnowledgeBaseDatabaseDisplayName)
	})
	if err != nil {
		if common.IsCode(err, common.ErrorCode_DATABASE_NOT_FOUND) {
			ginctx.WriteError(c, http.StatusNotFound, common.ErrorCode_DATABASE_NOT_FOUND, semanticModelInternalDisplayDatabaseNotFoundMessage)
			return
		}
		mapSemanticStorageError(c, err, "Failed to update semantic model")
		return
	}
	ginctx.WriteSuccess(c, gin.H{"updated": true})
}

func validateSemanticModelDatabaseDisplayNameRequest(req *semanticModelDatabaseDisplayNameInternalRequest) string {
	if req == nil {
		return ""
	}
	if req.DatabaseID <= 0 {
		return semanticModelInternalDisplayDatabaseIDRequiredMessage
	}
	if req.DisplayName == "" {
		return semanticModelInternalDisplayNameRequiredMessage
	}
	return ""
}

func authorizeSemanticModelInternalDisplayUpdate(c *gin.Context) bool {
	if _, ok := ginctx.GetAuthenticatedBackendExecution(c); !ok {
		ginctx.WriteError(c, http.StatusForbidden, common.ErrorCode_PERMISSION_DENIED, semanticModelBackendExecutionRequiredMessage)
		return false
	}
	return true
}

func (h *SemanticModelHandler) ensureSemanticModelDatabaseDisplayName(ctx context.Context, request *semanticModelDatabaseDisplayNameInternalRequest) error {
	if request == nil {
		return nil
	}
	storage, ok := h.storage.(semanticModelDatabaseDisplayStorage)
	if !ok {
		return fmt.Errorf("semantic model storage does not support system resource display mappings")
	}
	if _, err := storage.GetDatabase(ctx, request.DatabaseID); err != nil {
		return err
	}
	return systemdisplay.EnsureResourceDisplayMappings(ctx, storage, []systemdisplay.Binding{{
		ResourceType: displayi18n.ResourceTypeDatabase,
		ResourceID:   systemdisplay.BindingResourceID(request.DatabaseID),
		Field:        displayi18n.FieldName,
		DisplayOwner: semanticModelDisplayOwnerBackend,
		DisplayKey:   semanticModelDisplayKeyLiteralDefault,
		DefaultText:  request.DisplayName,
	}})
}

// Delete DELETE /api/v1/workspaces/:id/semantic-models/:model_id
//
// @Summary 删除语义模型
// @Description 删除指定语义模型。同一租户事务内同步清理当前工作区普通智能体绑定与系统/共享智能体覆盖绑定中的目标知识库引用，并为引用该知识库名称的非 disabled Agent Package 版本追加 knowledge_base_deleted 诊断并转为 needs_configuration。成功响应仍为 {deleted:true}。
// @Tags Semantic Model 管理
// @Produce json
// @Param id path string true "Workspace ID"
// @Param model_id path int true "模型 ID"
// @Success 200 {object} map[string]interface{} "删除成功"
// @Failure 400 {object} ginctx.ErrorResponse "参数错误"
// @Failure 401 {object} ginctx.ErrorResponse "未认证"
// @Failure 404 {object} ginctx.ErrorResponse "模型不存在"
// @Failure 500 {object} ginctx.ErrorResponse "内部错误"
// @Security ApiKeyAuth
// @Router /api/v1/workspaces/{id}/semantic-models/{model_id} [delete]
func (h *SemanticModelHandler) Delete(c *gin.Context) {
	ctx := c.Request.Context()
	workspaceID, err := h.parseWorkspaceID(c)
	if err != nil {
		return
	}
	modelID, err := h.parseModelID(c)
	if err != nil {
		return
	}
	uid := ginctx.GetUserID(c)
	if uid == "" {
		writeCatalogAPIError(c, http.StatusUnauthorized, common.ErrorCode_UNAUTHENTICATED, "")
		return
	}
	if h.iamRegistrar == nil {
		writeCatalogAPIError(c, http.StatusServiceUnavailable, common.ErrorCode_UNAVAILABLE, "")
		return
	}
	requestID, traceID, ok := semanticModelCorrelation(c, "semantic-model-delete")
	if !ok {
		writeCatalogAPIError(c, http.StatusInternalServerError, common.ErrorCode_INTERNAL, "")
		return
	}

	tm, err := h.pool.GetTransactionManager(ctx, workspaceID)
	if err != nil {
		h.logger.Error("Delete: GetTransactionManager failed",
			zap.String("workspace_id", workspaceID), zap.Error(err))
		writeCatalogAPIError(c, http.StatusInternalServerError, common.ErrorCode_INTERNAL, "")
		return
	}
	ctx = tenant.WithTransactionManager(ctx, tm)

	var deleteStats agentresource.SemanticKnowledgeBaseDeleteStats
	err = tm.RunInTx(ctx, func(txCtx context.Context) error {
		modelIDString := strconv.FormatInt(modelID, 10)
		digest := sha256.Sum256([]byte("semantic-model-delete\x00" + workspaceID + "\x00" + modelIDString + "\x00" + requestID))
		req := iamcore.SemanticModelResourceDeleteRequest{WorkspaceID: workspaceID, SemanticModelID: modelIDString, PrincipalID: uid, RoleID: workflowExplicitRole(c), OperationID: "op_" + hex.EncodeToString(digest[:16]), RequestID: requestID, TraceID: traceID}
		owner, err := h.iamRegistrar.BeginDelete(txCtx, req)
		if err != nil {
			return err
		}
		model, err := h.storage.GetSemanticModelForUpdate(txCtx, modelID)
		if err != nil {
			return err
		}
		if h.knowledgeDeleteHandler != nil {
			stats, err := h.knowledgeDeleteHandler.HandleSemanticKnowledgeBaseDeleted(txCtx, workspaceID, model.ID, model.Name, uid)
			if err != nil {
				return err
			}
			deleteStats = stats
		}
		if err := h.storage.DeleteSemanticModel(txCtx, modelID); err != nil {
			return err
		}
		return h.iamRegistrar.FinalizeDelete(txCtx, req, owner.OwnershipVersion, owner.AuthorizedRoleID)
	})
	if err != nil {
		if errors.Is(err, authzcore.ErrDenied) {
			writeSemanticModelIAMError(c, err)
			return
		}
		mapSemanticStorageError(c, err, "Failed to delete semantic model")
		return
	}
	h.logger.Info("semantic model deleted with agent reference cleanup",
		zap.String("workspace_id", workspaceID),
		zap.Int64("model_id", modelID),
		zap.Int("unbound_agents", deleteStats.UnboundAgents),
		zap.Int("unbound_agent_bindings", deleteStats.UnboundAgentBindings),
		zap.Int("needs_configuration_versions", deleteStats.NeedsConfigurationCount),
	)
	ginctx.WriteSuccess(c, gin.H{"deleted": true})
}

// CreateEntry POST /api/v1/workspaces/:id/semantic-models/:model_id/entries
//
// @Summary 创建语义条目
// @Description 在指定语义模型下创建语义条目
// @Tags Semantic Model 管理
// @Accept json
// @Produce json
// @Param id path string true "Workspace ID"
// @Param model_id path int true "模型 ID"
// @Param request body semanticEntryUpsertRequest true "语义条目信息"
// @Success 201 {object} semanticEntryResponse "创建成功"
// @Failure 400 {object} ginctx.ErrorResponse "参数错误"
// @Failure 401 {object} ginctx.ErrorResponse "未认证"
// @Failure 404 {object} ginctx.ErrorResponse "模型不存在"
// @Failure 409 {object} ginctx.ErrorResponse "条目已存在"
// @Failure 500 {object} ginctx.ErrorResponse "内部错误"
// @Security ApiKeyAuth
// @Router /api/v1/workspaces/{id}/semantic-models/{model_id}/entries [post]
func (h *SemanticModelHandler) CreateEntry(c *gin.Context) {
	ctx := c.Request.Context()
	workspaceID, err := h.parseWorkspaceID(c)
	if err != nil {
		return
	}
	modelID, err := h.parseModelID(c)
	if err != nil {
		return
	}
	uid := ginctx.GetUserID(c)
	if uid == "" {
		ginctx.WriteError(c, http.StatusUnauthorized, common.ErrorCode_UNAUTHENTICATED, "User not authenticated")
		return
	}
	if !h.authorize(c, workspaceID, uid, iamcore.IAMActionSemanticModelUpdate, modelID) {
		return
	}

	var req semanticEntryUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ginctx.WriteError(c, http.StatusBadRequest, common.ErrorCode_INVALID_ARGUMENT, "Invalid request body")
		return
	}
	entryTables := normalizeEntryTables(req.Tables)
	if semanticservice.IsDisabledLegacyEntryTables(entryTables) {
		writeSemanticValidationError(c, semanticservice.WrapValidationError(semanticservice.NewDisabledLegacyEntriesError()))
		return
	}

	tm, err := h.pool.GetTransactionManager(ctx, workspaceID)
	if err != nil {
		h.logger.Error("CreateEntry: GetTransactionManager failed",
			zap.String("workspace_id", workspaceID), zap.Error(err))
		ginctx.WriteError(c, http.StatusInternalServerError, common.ErrorCode_INTERNAL, "Failed to create semantic entry")
		return
	}
	ctx = tenant.WithTransactionManager(ctx, tm)

	var created *tenant.SemanticEntryRecord
	err = tm.RunInTx(ctx, func(txCtx context.Context) error {
		model, getErr := h.storage.GetSemanticModel(txCtx, modelID)
		if getErr != nil {
			return getErr
		}
		currentEntries, listErr := h.loadAllSemanticEntries(txCtx, modelID)
		if listErr != nil {
			return listErr
		}

		inputs := make([]semanticEntryInput, 0, len(currentEntries)+1)
		for _, item := range currentEntries {
			inputs = append(inputs, semanticEntryInput{
				Kind:   item.Kind,
				Key:    item.KeyName,
				Tables: normalizeEntryTables(item.Tables),
				Spec:   item.Spec,
			})
		}
		inputs = append(inputs, semanticEntryInput{
			Kind:   strings.ToLower(strings.TrimSpace(req.Kind)),
			Key:    strings.TrimSpace(req.Key),
			Tables: entryTables,
			Spec:   req.Spec,
		})
		if err := validateSemanticModelSpec(model.Tables, inputs); err != nil {
			return semanticservice.WrapValidationError(err)
		}

		created, err = h.storage.CreateSemanticEntry(txCtx, &tenant.SemanticEntryRecord{
			ModelID:   modelID,
			Kind:      strings.ToLower(strings.TrimSpace(req.Kind)),
			KeyName:   strings.TrimSpace(req.Key),
			Tables:    entryTables,
			Spec:      bytesTrimSpace(req.Spec),
			CreatedBy: uid,
			UpdatedBy: uid,
		})
		return err
	})
	if err != nil {
		if writeSemanticValidationError(c, err) {
			return
		}
		mapSemanticStorageError(c, err, "Failed to create semantic entry")
		return
	}
	ginctx.WriteSuccessWithStatus(c, http.StatusCreated, toEntryResponse(created))
}

// ListEntries GET /api/v1/workspaces/:id/semantic-models/:model_id/entries
//
// @Summary 列出语义条目
// @Description 列出指定语义模型下的语义条目
// @Tags Semantic Model 管理
// @Produce json
// @Param id path string true "Workspace ID"
// @Param model_id path int true "模型 ID"
// @Param kind query string false "条目类型过滤"
// @Param page_size query int false "分页大小"
// @Param page_token query string false "分页游标"
// @Success 200 {object} semanticEntryListResponse "查询成功"
// @Failure 400 {object} ginctx.ErrorResponse "参数错误"
// @Failure 401 {object} ginctx.ErrorResponse "未认证"
// @Failure 404 {object} ginctx.ErrorResponse "模型不存在"
// @Failure 500 {object} ginctx.ErrorResponse "内部错误"
// @Security ApiKeyAuth
// @Router /api/v1/workspaces/{id}/semantic-models/{model_id}/entries [get]
func (h *SemanticModelHandler) ListEntries(c *gin.Context) {
	ctx := c.Request.Context()
	workspaceID, err := h.parseWorkspaceID(c)
	if err != nil {
		return
	}
	modelID, err := h.parseModelID(c)
	if err != nil {
		return
	}
	uid := ginctx.GetUserID(c)
	if uid == "" {
		ginctx.WriteError(c, http.StatusUnauthorized, common.ErrorCode_UNAUTHENTICATED, "User not authenticated")
		return
	}
	if !h.authorize(c, workspaceID, uid, iamcore.IAMActionSemanticModelRead, modelID) {
		return
	}

	kind := strings.ToLower(strings.TrimSpace(c.Query("kind")))
	opts, _, pageToken := parsePaginationOptions(c)

	tm, err := h.pool.GetTransactionManager(ctx, workspaceID)
	if err != nil {
		h.logger.Error("ListEntries: GetTransactionManager failed",
			zap.String("workspace_id", workspaceID), zap.Error(err))
		ginctx.WriteError(c, http.StatusInternalServerError, common.ErrorCode_INTERNAL, "Failed to list semantic entries")
		return
	}
	ctx = tenant.WithTransactionManager(ctx, tm)

	var (
		entries []*tenant.SemanticEntryRecord
		total   int64
	)
	err = tm.RunInTx(ctx, func(txCtx context.Context) error {
		if _, getErr := h.storage.GetSemanticModel(txCtx, modelID); getErr != nil {
			return getErr
		}
		entries, total, err = h.storage.ListSemanticEntries(txCtx, modelID, kind, opts...)
		return err
	})
	if err != nil {
		mapSemanticStorageError(c, err, "Failed to list semantic entries")
		return
	}

	items := make([]*semanticEntryResponse, 0, len(entries))
	for _, entry := range entries {
		items = append(items, toEntryResponse(entry))
	}
	ginctx.WriteSuccess(c, &semanticEntryListResponse{
		Items:         items,
		Total:         total,
		NextPageToken: nextPageTokenForList(pageToken, len(entries), total),
	})
}

// UpdateEntry PUT /api/v1/workspaces/:id/semantic-models/:model_id/entries/:entry_id
//
// @Summary 更新语义条目
// @Description 更新指定语义模型下的语义条目
// @Tags Semantic Model 管理
// @Accept json
// @Produce json
// @Param id path string true "Workspace ID"
// @Param model_id path int true "模型 ID"
// @Param entry_id path int true "条目 ID"
// @Param request body semanticEntryUpsertRequest true "语义条目信息"
// @Success 200 {object} map[string]interface{} "更新成功"
// @Failure 400 {object} ginctx.ErrorResponse "参数错误"
// @Failure 401 {object} ginctx.ErrorResponse "未认证"
// @Failure 404 {object} ginctx.ErrorResponse "模型或条目不存在"
// @Failure 500 {object} ginctx.ErrorResponse "内部错误"
// @Security ApiKeyAuth
// @Router /api/v1/workspaces/{id}/semantic-models/{model_id}/entries/{entry_id} [put]
func (h *SemanticModelHandler) UpdateEntry(c *gin.Context) {
	ctx := c.Request.Context()
	workspaceID, err := h.parseWorkspaceID(c)
	if err != nil {
		return
	}
	modelID, err := h.parseModelID(c)
	if err != nil {
		return
	}
	entryID, err := h.parseEntryID(c)
	if err != nil {
		return
	}
	uid := ginctx.GetUserID(c)
	if uid == "" {
		ginctx.WriteError(c, http.StatusUnauthorized, common.ErrorCode_UNAUTHENTICATED, "User not authenticated")
		return
	}
	if !h.authorize(c, workspaceID, uid, iamcore.IAMActionSemanticModelUpdate, modelID) {
		return
	}

	var req semanticEntryUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ginctx.WriteError(c, http.StatusBadRequest, common.ErrorCode_INVALID_ARGUMENT, "Invalid request body")
		return
	}
	entryTables := normalizeEntryTables(req.Tables)
	if semanticservice.IsDisabledLegacyEntryTables(entryTables) {
		writeSemanticValidationError(c, semanticservice.WrapValidationError(semanticservice.NewDisabledLegacyEntriesError()))
		return
	}

	tm, err := h.pool.GetTransactionManager(ctx, workspaceID)
	if err != nil {
		h.logger.Error("UpdateEntry: GetTransactionManager failed",
			zap.String("workspace_id", workspaceID), zap.Error(err))
		ginctx.WriteError(c, http.StatusInternalServerError, common.ErrorCode_INTERNAL, "Failed to update semantic entry")
		return
	}
	ctx = tenant.WithTransactionManager(ctx, tm)

	err = tm.RunInTx(ctx, func(txCtx context.Context) error {
		model, getErr := h.storage.GetSemanticModel(txCtx, modelID)
		if getErr != nil {
			return getErr
		}
		existing, getEntryErr := h.storage.GetSemanticEntry(txCtx, modelID, entryID)
		if getEntryErr != nil {
			return getEntryErr
		}

		currentEntries, listErr := h.loadAllSemanticEntries(txCtx, modelID)
		if listErr != nil {
			return listErr
		}

		inputs := make([]semanticEntryInput, 0, len(currentEntries))
		replaced := false
		for _, item := range currentEntries {
			if item.ID == existing.ID {
				inputs = append(inputs, semanticEntryInput{
					Kind:   strings.ToLower(strings.TrimSpace(req.Kind)),
					Key:    strings.TrimSpace(req.Key),
					Tables: entryTables,
					Spec:   req.Spec,
				})
				replaced = true
				continue
			}
			inputs = append(inputs, semanticEntryInput{
				Kind:   item.Kind,
				Key:    item.KeyName,
				Tables: normalizeEntryTables(item.Tables),
				Spec:   item.Spec,
			})
		}
		if !replaced {
			return tenant.ErrSemanticEntryNotFound
		}
		if err := validateSemanticModelSpec(model.Tables, inputs); err != nil {
			return semanticservice.WrapValidationError(err)
		}

		return h.storage.UpdateSemanticEntry(txCtx, &tenant.SemanticEntryRecord{
			ID:        entryID,
			ModelID:   modelID,
			Kind:      strings.ToLower(strings.TrimSpace(req.Kind)),
			KeyName:   strings.TrimSpace(req.Key),
			Tables:    entryTables,
			Spec:      bytesTrimSpace(req.Spec),
			UpdatedBy: uid,
		})
	})
	if err != nil {
		if writeSemanticValidationError(c, err) {
			return
		}
		mapSemanticStorageError(c, err, "Failed to update semantic entry", map[string]string{"entry_id": strconv.FormatInt(entryID, 10)})
		return
	}
	ginctx.WriteSuccess(c, gin.H{"updated": true})
}

// DeleteEntry DELETE /api/v1/workspaces/:id/semantic-models/:model_id/entries/:entry_id
//
// @Summary 删除语义条目
// @Description 删除指定语义模型下的语义条目
// @Tags Semantic Model 管理
// @Produce json
// @Param id path string true "Workspace ID"
// @Param model_id path int true "模型 ID"
// @Param entry_id path int true "条目 ID"
// @Success 200 {object} map[string]interface{} "删除成功"
// @Failure 400 {object} ginctx.ErrorResponse "参数错误"
// @Failure 401 {object} ginctx.ErrorResponse "未认证"
// @Failure 404 {object} ginctx.ErrorResponse "模型或条目不存在"
// @Failure 500 {object} ginctx.ErrorResponse "内部错误"
// @Security ApiKeyAuth
// @Router /api/v1/workspaces/{id}/semantic-models/{model_id}/entries/{entry_id} [delete]
func (h *SemanticModelHandler) DeleteEntry(c *gin.Context) {
	ctx := c.Request.Context()
	workspaceID, err := h.parseWorkspaceID(c)
	if err != nil {
		return
	}
	modelID, err := h.parseModelID(c)
	if err != nil {
		return
	}
	entryID, err := h.parseEntryID(c)
	if err != nil {
		return
	}
	uid := ginctx.GetUserID(c)
	if uid == "" {
		ginctx.WriteError(c, http.StatusUnauthorized, common.ErrorCode_UNAUTHENTICATED, "User not authenticated")
		return
	}
	if !h.authorize(c, workspaceID, uid, iamcore.IAMActionSemanticModelDelete, modelID) {
		return
	}

	tm, err := h.pool.GetTransactionManager(ctx, workspaceID)
	if err != nil {
		h.logger.Error("DeleteEntry: GetTransactionManager failed",
			zap.String("workspace_id", workspaceID), zap.Error(err))
		ginctx.WriteError(c, http.StatusInternalServerError, common.ErrorCode_INTERNAL, "Failed to delete semantic entry")
		return
	}
	ctx = tenant.WithTransactionManager(ctx, tm)

	err = tm.RunInTx(ctx, func(txCtx context.Context) error {
		if _, getErr := h.storage.GetSemanticModel(txCtx, modelID); getErr != nil {
			return getErr
		}
		return h.storage.DeleteSemanticEntry(txCtx, modelID, entryID)
	})
	if err != nil {
		mapSemanticStorageError(c, err, "Failed to delete semantic entry", map[string]string{"entry_id": strconv.FormatInt(entryID, 10)})
		return
	}
	ginctx.WriteSuccess(c, gin.H{"deleted": true})
}

// Import POST /api/v1/workspaces/:id/semantic-models/import
//
// @Summary 导入语义模型
// @Description 导入语义模型，支持单个对象或对象数组（批量导入）
// @Tags Semantic Model 管理
// @Accept json
// @Produce json
// @Param id path string true "Workspace ID"
// @Param request body semanticModelImportRequest true "导入内容（单个对象或对象数组）"
// @Success 201 {object} map[string]interface{} "导入成功"
// @Failure 400 {object} ginctx.ErrorResponse "参数错误"
// @Failure 401 {object} ginctx.ErrorResponse "未认证"
// @Failure 409 {object} ginctx.ErrorResponse "模型或条目冲突"
// @Failure 500 {object} ginctx.ErrorResponse "内部错误"
// @Security ApiKeyAuth
// @Router /api/v1/workspaces/{id}/semantic-models/import [post]
func (h *SemanticModelHandler) Import(c *gin.Context) {
	ctx := c.Request.Context()
	requestStart := time.Now()
	workspaceID, err := h.parseWorkspaceID(c)
	if err != nil {
		return
	}
	uid := ginctx.GetUserID(c)
	if uid == "" {
		ginctx.WriteError(c, http.StatusUnauthorized, common.ErrorCode_UNAUTHENTICATED, "User not authenticated")
		return
	}
	createDecision, ok := h.authorizeDecision(c, workspaceID, uid, iamcore.IAMActionSemanticModelCreate, 0)
	if !ok {
		return
	}
	reqLog := logging.WithContext(ctx, h.logger).With(
		zap.String("workspace_id", workspaceID),
		zap.String("user_id", uid),
	)

	rawBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		reqLog.Warn("semantic model import failed: read request body",
			zap.Duration("elapsed", time.Since(requestStart)),
			zap.Error(err))
		ginctx.WriteError(c, http.StatusBadRequest, common.ErrorCode_INVALID_ARGUMENT, "Invalid request body")
		return
	}
	reqLog.Info("semantic model import started",
		zap.Int("payload_bytes", len(rawBody)))

	importReqs, batchMode, err := parseSemanticImportPayload(rawBody)
	if err != nil {
		reqLog.Warn("semantic model import failed: parse payload",
			zap.Duration("elapsed", time.Since(requestStart)),
			zap.Error(err))
		ginctx.WriteError(c, http.StatusBadRequest, common.ErrorCode_INVALID_ARGUMENT, "Invalid request body")
		return
	}
	reqLog.Debug("semantic model import payload parsed",
		zap.Bool("batch_mode", batchMode),
		zap.Int("model_count", len(importReqs)))

	type preparedImport struct {
		req        semanticModelImportRequest
		tableNames []string
		inputs     []semanticEntryInput
	}
	prepared := make([]preparedImport, 0, len(importReqs))
	totalEntries := 0
	skippedDisabledLegacyEntries := 0
	for _, req := range importReqs {
		if strings.TrimSpace(req.Name) == "" {
			reqLog.Warn("semantic model import failed: empty model name",
				zap.Duration("elapsed", time.Since(requestStart)))
			ginctx.WriteError(c, http.StatusBadRequest, common.ErrorCode_INVALID_ARGUMENT, "name is required")
			return
		}
		tableNames, err := extractTableNamesFromJSON(req.Tables)
		if err != nil {
			reqLog.Warn("semantic model import failed: invalid model tables",
				zap.String("model_name", strings.TrimSpace(req.Name)),
				zap.Duration("elapsed", time.Since(requestStart)),
				zap.Error(err))
			ginctx.WriteError(c, http.StatusBadRequest, common.ErrorCode_INVALID_ARGUMENT, err.Error())
			return
		}

		inputs := make([]semanticEntryInput, 0, len(req.Entries))
		for _, entry := range req.Entries {
			entryTables := normalizeEntryTables(entry.Tables)
			if semanticservice.IsDisabledLegacyEntryTables(entryTables) {
				skippedDisabledLegacyEntries++
				continue
			}
			inputs = append(inputs, semanticEntryInput{
				Kind:   strings.ToLower(strings.TrimSpace(entry.Kind)),
				Key:    strings.TrimSpace(entry.Key),
				Tables: entryTables,
				Spec:   bytesTrimSpace(entry.Spec),
			})
		}
		if err := validateSemanticModelSpec(req.Tables, inputs); err != nil {
			reqLog.Warn("semantic model import failed: invalid semantic spec",
				zap.String("model_name", strings.TrimSpace(req.Name)),
				zap.Duration("elapsed", time.Since(requestStart)),
				zap.Error(err))
			if !writeSemanticValidationError(c, semanticservice.WrapValidationError(err)) {
				ginctx.WriteError(c, http.StatusBadRequest, common.ErrorCode_INVALID_ARGUMENT, err.Error())
			}
			return
		}
		prepared = append(prepared, preparedImport{
			req:        req,
			tableNames: tableNames,
			inputs:     inputs,
		})
		totalEntries += len(inputs)
	}
	reqLog.Debug("semantic model import prepared",
		zap.Int("prepared_models", len(prepared)),
		zap.Int("prepared_entries", totalEntries),
		zap.Int("skipped_disabled_legacy_entries", skippedDisabledLegacyEntries))

	tmStart := time.Now()
	tm, err := h.pool.GetTransactionManager(ctx, workspaceID)
	if err != nil {
		reqLog.Error("semantic model import failed: get transaction manager",
			zap.Duration("tm_duration", time.Since(tmStart)),
			zap.Duration("elapsed", time.Since(requestStart)),
			zap.Error(err))
		writeCatalogAPIError(c, http.StatusInternalServerError, common.ErrorCode_INTERNAL, "")
		return
	}
	reqLog.Debug("semantic model import transaction manager ready",
		zap.Duration("tm_duration", time.Since(tmStart)))
	ctx = tenant.WithTransactionManager(ctx, tm)
	requestID, traceID, ok := semanticModelCorrelation(c, "semantic-model-import")
	if !ok || h.iamRegistrar == nil {
		writeCatalogAPIError(c, http.StatusServiceUnavailable, common.ErrorCode_UNAVAILABLE, "")
		return
	}

	createdModels := make([]*tenant.SemanticModelRecord, 0, len(prepared))
	entryBatchStorage, canBatchInsertEntries := h.storage.(semanticEntryBatchStorage)
	txStart := time.Now()
	err = tm.RunInTx(ctx, func(txCtx context.Context) error {
		for index, item := range prepared {
			created, err := h.storage.CreateSemanticModel(txCtx, &tenant.SemanticModelRecord{
				Name:         strings.TrimSpace(item.req.Name),
				Description:  strings.TrimSpace(item.req.Description),
				Tables:       item.req.Tables,
				Files:        item.req.Files,
				TableSetHash: semanticTableSetHash(item.tableNames),
				CreatedBy:    uid,
				UpdatedBy:    uid,
			})
			if err != nil {
				return err
			}
			modelID := strconv.FormatInt(created.ID, 10)
			modelRequestID := iamcoreRequestID(requestID, "model-"+strconv.Itoa(index)+"-"+modelID)
			digest := sha256.Sum256([]byte("semantic-model-import\x00" + workspaceID + "\x00" + modelID + "\x00" + modelRequestID))
			if _, err := h.iamRegistrar.RegisterAuthorized(txCtx, iamcore.SemanticModelResourceRegisterRequest{WorkspaceID: workspaceID, SemanticModelID: modelID, PrincipalID: uid, RoleID: workflowExplicitRole(c), OperationID: "op_" + hex.EncodeToString(digest[:16]), RequestID: modelRequestID, TraceID: traceID}, createDecision.VerifiedEffectiveRoleID); err != nil {
				return err
			}
			entries := make([]*tenant.SemanticEntryRecord, 0, len(item.inputs))
			for _, entry := range item.inputs {
				entries = append(entries, &tenant.SemanticEntryRecord{
					ModelID:   created.ID,
					Kind:      entry.Kind,
					KeyName:   entry.Key,
					Tables:    entry.Tables,
					Spec:      entry.Spec,
					CreatedBy: uid,
					UpdatedBy: uid,
				})
			}
			if canBatchInsertEntries {
				if err := entryBatchStorage.CreateSemanticEntriesBatch(txCtx, entries); err != nil {
					return err
				}
			} else {
				for _, entry := range entries {
					if _, err := h.storage.CreateSemanticEntry(txCtx, entry); err != nil {
						return err
					}
				}
			}
			createdModels = append(createdModels, created)
		}
		return nil
	})
	if err != nil {
		reqLog.Error("semantic model import failed: transaction execution",
			zap.Duration("tx_duration", time.Since(txStart)),
			zap.Duration("elapsed", time.Since(requestStart)),
			zap.Int("prepared_models", len(prepared)),
			zap.Int("prepared_entries", totalEntries),
			zap.Bool("entry_batch_insert", canBatchInsertEntries),
			zap.Error(err))
		mapSemanticStorageError(c, err, "Failed to import semantic model")
		return
	}
	reqLog.Info("semantic model import succeeded",
		zap.Bool("batch_mode", batchMode),
		zap.Int("created_models", len(createdModels)),
		zap.Int("created_entries", totalEntries),
		zap.Int("skipped_disabled_legacy_entries", skippedDisabledLegacyEntries),
		zap.Bool("entry_batch_insert", canBatchInsertEntries),
		zap.Duration("tx_duration", time.Since(txStart)),
		zap.Duration("elapsed", time.Since(requestStart)))
	if !batchMode && len(createdModels) == 1 {
		ginctx.WriteSuccessWithStatus(c, http.StatusCreated, toModelResponse(createdModels[0]))
		return
	}
	items := make([]*semanticModelResponse, 0, len(createdModels))
	for _, item := range createdModels {
		items = append(items, toModelResponse(item))
	}
	ginctx.WriteSuccessWithStatus(c, http.StatusCreated, &semanticModelImportBatchResponse{
		Items: items,
		Total: int64(len(items)),
	})
}

func parseSemanticImportPayload(raw []byte) ([]semanticModelImportRequest, bool, error) {
	trimmed := bytesTrimSpace(raw)
	if len(trimmed) == 0 {
		return nil, false, fmt.Errorf("empty payload")
	}
	if trimmed[0] == '[' {
		var items []semanticModelImportRequest
		if err := json.Unmarshal(trimmed, &items); err != nil {
			return nil, false, err
		}
		if len(items) == 0 {
			return nil, false, fmt.Errorf("empty import list")
		}
		return items, true, nil
	}
	var req semanticModelImportRequest
	if err := json.Unmarshal(trimmed, &req); err != nil {
		return nil, false, err
	}
	return []semanticModelImportRequest{req}, false, nil
}

// Export GET /api/v1/workspaces/:id/semantic-models/:model_id/export
//
// @Summary 导出语义模型
// @Description 导出语义模型及其全部语义条目
// @Tags Semantic Model 管理
// @Produce json
// @Param id path string true "Workspace ID"
// @Param model_id path int true "模型 ID"
// @Success 200 {object} map[string]interface{} "导出成功"
// @Failure 400 {object} ginctx.ErrorResponse "参数错误"
// @Failure 401 {object} ginctx.ErrorResponse "未认证"
// @Failure 404 {object} ginctx.ErrorResponse "模型不存在"
// @Failure 500 {object} ginctx.ErrorResponse "内部错误"
// @Security ApiKeyAuth
// @Router /api/v1/workspaces/{id}/semantic-models/{model_id}/export [get]
func (h *SemanticModelHandler) Export(c *gin.Context) {
	ctx := c.Request.Context()
	workspaceID, err := h.parseWorkspaceID(c)
	if err != nil {
		return
	}
	modelID, err := h.parseModelID(c)
	if err != nil {
		return
	}
	uid := ginctx.GetUserID(c)
	if uid == "" {
		ginctx.WriteError(c, http.StatusUnauthorized, common.ErrorCode_UNAUTHENTICATED, "User not authenticated")
		return
	}
	if !h.authorize(c, workspaceID, uid, iamcore.IAMActionSemanticModelRead, modelID) {
		return
	}

	tm, err := h.pool.GetTransactionManager(ctx, workspaceID)
	if err != nil {
		h.logger.Error("Export: GetTransactionManager failed",
			zap.String("workspace_id", workspaceID), zap.Error(err))
		ginctx.WriteError(c, http.StatusInternalServerError, common.ErrorCode_INTERNAL, "Failed to export semantic model")
		return
	}
	ctx = tenant.WithTransactionManager(ctx, tm)

	var (
		model   *tenant.SemanticModelRecord
		entries []*tenant.SemanticEntryRecord
	)
	err = tm.RunInTx(ctx, func(txCtx context.Context) error {
		model, err = h.storage.GetSemanticModel(txCtx, modelID)
		if err != nil {
			return err
		}
		entries, err = h.loadAllSemanticEntries(txCtx, modelID)
		return err
	})
	if err != nil {
		mapSemanticStorageError(c, err, "Failed to export semantic model")
		return
	}

	respEntries := make([]*semanticEntryResponse, 0, len(entries))
	for _, entry := range entries {
		respEntries = append(respEntries, toEntryResponse(entry))
	}
	ginctx.WriteSuccess(c, gin.H{
		"model":   toModelResponse(model),
		"entries": respEntries,
	})
}

// Validate POST /api/v1/workspaces/:id/semantic-models/:model_id/validate
//
// @Summary 校验语义模型
// @Description 对语义模型及全部语义条目执行一致性校验
// @Tags Semantic Model 管理
// @Produce json
// @Param id path string true "Workspace ID"
// @Param model_id path int true "模型 ID"
// @Success 200 {object} map[string]interface{} "校验通过"
// @Failure 400 {object} ginctx.ErrorResponse "参数错误或校验失败"
// @Failure 401 {object} ginctx.ErrorResponse "未认证"
// @Failure 404 {object} ginctx.ErrorResponse "模型不存在"
// @Failure 500 {object} ginctx.ErrorResponse "内部错误"
// @Security ApiKeyAuth
// @Router /api/v1/workspaces/{id}/semantic-models/{model_id}/validate [post]
func (h *SemanticModelHandler) Validate(c *gin.Context) {
	ctx := c.Request.Context()
	workspaceID, err := h.parseWorkspaceID(c)
	if err != nil {
		return
	}
	modelID, err := h.parseModelID(c)
	if err != nil {
		return
	}
	uid := ginctx.GetUserID(c)
	if uid == "" {
		ginctx.WriteError(c, http.StatusUnauthorized, common.ErrorCode_UNAUTHENTICATED, "User not authenticated")
		return
	}
	if !h.authorize(c, workspaceID, uid, iamcore.IAMActionSemanticModelUse, modelID) {
		return
	}

	tm, err := h.pool.GetTransactionManager(ctx, workspaceID)
	if err != nil {
		h.logger.Error("Validate: GetTransactionManager failed",
			zap.String("workspace_id", workspaceID), zap.Error(err))
		ginctx.WriteError(c, http.StatusInternalServerError, common.ErrorCode_INTERNAL, "Failed to validate semantic model")
		return
	}
	ctx = tenant.WithTransactionManager(ctx, tm)

	err = tm.RunInTx(ctx, func(txCtx context.Context) error {
		model, err := h.storage.GetSemanticModel(txCtx, modelID)
		if err != nil {
			return err
		}
		entries, err := h.loadAllSemanticEntries(txCtx, modelID)
		if err != nil {
			return err
		}
		inputs := make([]semanticEntryInput, 0, len(entries))
		for _, entry := range entries {
			inputs = append(inputs, semanticEntryInput{
				Kind:   entry.Kind,
				Key:    entry.KeyName,
				Tables: normalizeEntryTables(entry.Tables),
				Spec:   entry.Spec,
			})
		}
		if err := validateSemanticModelSpec(model.Tables, inputs); err != nil {
			return semanticservice.WrapValidationError(err)
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, tenant.ErrSemanticModelNotFound) {
			mapSemanticStorageError(c, err, "Failed to validate semantic model")
			return
		}
		if writeSemanticValidationError(c, err) {
			return
		}
		mapSemanticStorageError(c, err, "Failed to validate semantic model")
		return
	}
	ginctx.WriteSuccess(c, gin.H{"valid": true})
}
