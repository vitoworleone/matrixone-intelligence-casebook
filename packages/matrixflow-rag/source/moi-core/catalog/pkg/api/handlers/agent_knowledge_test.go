package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/matrixflow/moi-core/catalog/pkg/agentresource"
	ginctx "github.com/matrixflow/moi-core/catalog/pkg/api"
	"github.com/matrixflow/moi-core/model/auth"
)

func TestAgentKnowledgeBaseHandlerCRUD(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewAgentKnowledgeBaseHandler(agentresource.NewKnowledgeBaseService(agentresource.NewInMemoryAgentStore()))
	router := gin.New()
	router.Use(func(c *gin.Context) {
		ginctx.SetAPIKeyInfo(c, &auth.APIKey{Uid: "user_1"})
		c.Next()
	})
	router.POST("/api/v1/workspaces/:id/knowledge-bases", handler.CreateKnowledgeBase)
	router.GET("/api/v1/workspaces/:id/knowledge-bases", handler.ListKnowledgeBases)
	router.GET("/api/v1/workspaces/:id/knowledge-bases/:knowledge_base_id", handler.GetKnowledgeBase)
	router.PATCH("/api/v1/workspaces/:id/knowledge-bases/:knowledge_base_id", handler.UpdateKnowledgeBase)

	postJSON(t, router, "/api/v1/workspaces/ws_1/knowledge-bases", `{
		"id":"kb_1",
		"workspace_id":"attacker_ws",
		"name":"Supplier catalog",
		"description":"Approved supplier documents",
		"status":"active",
		"source_type":"catalog_resource",
		"catalog_asset_refs":[{"type":"catalog","id":"cat_1","role":"source"}],
		"tags":["supplier","bid"],
		"visibility":"workspace",
		"index_status":"ready"
	}`, http.StatusCreated)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/workspaces/ws_1/knowledge-bases?status=active&source_type=catalog_resource&visibility=workspace&index_status=ready&owner_user_id=user_1&query=supplier", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("list status = %d body = %s", rec.Code, rec.Body.String())
	}
	var listResp struct {
		Data struct {
			Items []agentresource.KnowledgeBase `json:"items"`
			Total int                           `json:"total"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if listResp.Data.Total != 1 || listResp.Data.Items[0].WorkspaceID != "ws_1" || listResp.Data.Items[0].CreatedBy != "user_1" {
		t.Fatalf("list response = %+v", listResp.Data)
	}

	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/workspaces/ws_1/knowledge-bases?owner_user_id=../user_1", nil))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid owner filter status = %d body = %s", rec.Code, rec.Body.String())
	}

	postJSON(t, router, "/api/v1/workspaces/ws_1/knowledge-bases/kb_1", `{"status":"disabled"}`, http.StatusOK, http.MethodPatch)
	postJSON(t, router, "/api/v1/workspaces/ws_1/knowledge-bases/kb_1", `{"owner_user_id":"../user_1"}`, http.StatusBadRequest, http.MethodPatch)

	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/workspaces/ws_1/knowledge-bases/kb_1", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("get status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"status":"disabled"`)) {
		t.Fatalf("get body = %s", rec.Body.String())
	}
}

func TestAgentKnowledgeBaseHandlerRejectsUnauthenticatedRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewAgentKnowledgeBaseHandler(agentresource.NewKnowledgeBaseService(agentresource.NewInMemoryAgentStore()))
	router := gin.New()
	router.POST("/api/v1/workspaces/:id/knowledge-bases", handler.CreateKnowledgeBase)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/ws_1/knowledge-bases", bytes.NewReader([]byte(`{"name":"KB"}`)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
}
