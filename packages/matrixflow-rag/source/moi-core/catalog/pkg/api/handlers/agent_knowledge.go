package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/matrixflow/moi-core/catalog/pkg/agentresource"
	ginctx "github.com/matrixflow/moi-core/catalog/pkg/api"
	"github.com/matrixflow/moi-core/model/common"
)

type AgentKnowledgeBaseHandler struct {
	service *agentresource.KnowledgeBaseService
}

func NewAgentKnowledgeBaseHandler(service *agentresource.KnowledgeBaseService) *AgentKnowledgeBaseHandler {
	return &AgentKnowledgeBaseHandler{service: service}
}

// CreateKnowledgeBase creates an agent knowledge base.
// @Summary Create agent knowledge base
// @Router /api/v1/workspaces/{id}/knowledge-bases [post]
func (h *AgentKnowledgeBaseHandler) CreateKnowledgeBase(c *gin.Context) {
	if h.service == nil {
		ginctx.WriteError(c, http.StatusServiceUnavailable, common.ErrorCode_UNAVAILABLE, "knowledge base service is not available")
		return
	}
	workspaceID, userID, ok := agentResourceScope(c)
	if !ok {
		return
	}
	var req agentresource.KnowledgeBaseCreateInput
	if err := c.ShouldBindJSON(&req); err != nil {
		ginctx.WriteError(c, http.StatusBadRequest, common.ErrorCode_INVALID_ARGUMENT, "Invalid request body")
		return
	}
	req.WorkspaceID = workspaceID
	req.UserID = userID
	kb, err := h.service.CreateKnowledgeBase(c.Request.Context(), req)
	writeAgentKnowledgeBaseResponse(c, http.StatusCreated, kb, err)
}

// ListKnowledgeBases lists agent knowledge bases.
// @Summary List agent knowledge bases
// @Router /api/v1/workspaces/{id}/knowledge-bases [get]
func (h *AgentKnowledgeBaseHandler) ListKnowledgeBases(c *gin.Context) {
	if h.service == nil {
		ginctx.WriteError(c, http.StatusServiceUnavailable, common.ErrorCode_UNAVAILABLE, "knowledge base service is not available")
		return
	}
	workspaceID, _, ok := agentResourceScope(c)
	if !ok {
		return
	}
	ownerUserID, ok := querySafeID(c, "owner_user_id")
	if !ok {
		return
	}
	limit, offset := normalizeAgentResourcePagination(queryInt(c, "limit"), queryInt(c, "offset"))
	items, total, err := h.service.ListKnowledgeBases(c.Request.Context(), agentresource.KnowledgeBaseListFilter{
		WorkspaceID: workspaceID,
		Status:      c.Query("status"),
		SourceType:  c.Query("source_type"),
		Visibility:  c.Query("visibility"),
		IndexStatus: c.Query("index_status"),
		OwnerUserID: ownerUserID,
		Query:       c.Query("query"),
		Limit:       limit,
		Offset:      offset,
	})
	if err != nil {
		writeAgentKnowledgeBaseResponse(c, http.StatusOK, nil, err)
		return
	}
	ginctx.WriteSuccess(c, gin.H{
		"items":  items,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// GetKnowledgeBase gets an agent knowledge base.
// @Summary Get agent knowledge base
// @Router /api/v1/workspaces/{id}/knowledge-bases/{knowledge_base_id} [get]
func (h *AgentKnowledgeBaseHandler) GetKnowledgeBase(c *gin.Context) {
	if h.service == nil {
		ginctx.WriteError(c, http.StatusServiceUnavailable, common.ErrorCode_UNAVAILABLE, "knowledge base service is not available")
		return
	}
	workspaceID, _, ok := agentResourceScope(c)
	if !ok {
		return
	}
	knowledgeBaseID, ok := pathSafeID(c, "knowledge_base_id")
	if !ok {
		return
	}
	kb, err := h.service.GetKnowledgeBase(c.Request.Context(), workspaceID, knowledgeBaseID)
	writeAgentKnowledgeBaseResponse(c, http.StatusOK, kb, err)
}

// UpdateKnowledgeBase updates an agent knowledge base.
// @Summary Update agent knowledge base
// @Router /api/v1/workspaces/{id}/knowledge-bases/{knowledge_base_id} [patch]
func (h *AgentKnowledgeBaseHandler) UpdateKnowledgeBase(c *gin.Context) {
	if h.service == nil {
		ginctx.WriteError(c, http.StatusServiceUnavailable, common.ErrorCode_UNAVAILABLE, "knowledge base service is not available")
		return
	}
	workspaceID, userID, ok := agentResourceScope(c)
	if !ok {
		return
	}
	var req agentresource.KnowledgeBaseUpdateInput
	if err := c.ShouldBindJSON(&req); err != nil {
		ginctx.WriteError(c, http.StatusBadRequest, common.ErrorCode_INVALID_ARGUMENT, "Invalid request body")
		return
	}
	req.UserID = userID
	knowledgeBaseID, ok := pathSafeID(c, "knowledge_base_id")
	if !ok {
		return
	}
	kb, err := h.service.UpdateKnowledgeBase(c.Request.Context(), workspaceID, knowledgeBaseID, req)
	writeAgentKnowledgeBaseResponse(c, http.StatusOK, kb, err)
}

func writeAgentKnowledgeBaseResponse(c *gin.Context, status int, payload any, err error) {
	if err != nil {
		switch {
		case errors.Is(err, agentresource.ErrKnowledgeBaseNotFound):
			ginctx.WriteError(c, http.StatusNotFound, common.ErrorCode_NOT_FOUND, "knowledge base not found")
		case errors.Is(err, agentresource.ErrKnowledgeBaseAlreadyExists):
			ginctx.WriteError(c, http.StatusConflict, common.ErrorCode_ALREADY_EXISTS, "knowledge base already exists")
		case errors.Is(err, agentresource.ErrInvalidKnowledgeBase):
			ginctx.WriteError(c, http.StatusBadRequest, common.ErrorCode_INVALID_ARGUMENT, err.Error())
		default:
			ginctx.WriteError(c, http.StatusBadRequest, common.ErrorCode_INVALID_ARGUMENT, err.Error())
		}
		return
	}
	ginctx.WriteSuccessWithStatus(c, status, payload)
}
