//go:build integration
// +build integration

package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	ginctx "github.com/matrixflow/moi-core/catalog/pkg/api"
	"github.com/matrixflow/moi-core/model/auth"
	catalogpb "github.com/matrixflow/moi-core/model/catalog"
)

// TestNl2SqlKnowledgeIntegration 集成测试
// 需要运行 MatrixOne 数据库
// 运行方式: INTEGRATION_TEST=1 go test -tags=integration -v ./catalog/pkg/api/handlers/ -run TestNl2SqlKnowledgeIntegration
//
// TODO: 需要实现 setupIntegrationPool() 辅助函数来创建真实的 ConnectionPool + TenantStorage，
// 参考 catalog/pkg/api/server.go 中的初始化逻辑。
func TestNl2SqlKnowledgeIntegration(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") == "" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=1 to run.")
	}

	// TODO: 替换为真实的 ConnectionPool 和 Storage 初始化
	// pool, storage, cleanup := setupIntegrationPool(t)
	// defer cleanup()
	t.Skip("TODO: 需要实现集成测试基础设施 (ConnectionPool + WorkspaceResolver + DatabaseProvider)")

	// 以下为集成测试逻辑，待基础设施就绪后启用
	_ = setupIntegrationNl2SqlRouter
}

// setupIntegrationNl2SqlRouter creates a test router with auth middleware for integration tests.
func setupIntegrationNl2SqlRouter(handler *Nl2SqlKnowledgeHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		ginctx.SetAPIKeyInfo(c, &auth.APIKey{Uid: "integration-test-user"})
		c.Next()
	})
	api := router.Group("/api/v1/workspaces/:id")
	{
		api.POST("/nl2sql-knowledge", handler.Create)
		api.POST("/nl2sql-knowledge/list", handler.List)
		api.PUT("/nl2sql-knowledge/:knowledge_id", handler.Update)
		api.DELETE("/nl2sql-knowledge/:knowledge_id", handler.Delete)
	}
	return router
}

// integrationNl2SqlTests contains the actual integration test logic.
// Uncomment and use when setupIntegrationPool is implemented.
func integrationNl2SqlTests(t *testing.T, handler *Nl2SqlKnowledgeHandler) {
	router := setupIntegrationNl2SqlRouter(handler)
	workspaceID := "test-workspace-001"
	knowledgeBaseID := int64(1)

	// Test 1: Create NL2SQL Knowledge
	t.Run("Create", func(t *testing.T) {
		reqBody := catalogpb.CreateNl2SqlKnowledgeRequest{
			KnowledgeBaseId: knowledgeBaseID,
			KnowledgeType:   "glossary",
			KnowledgeKey:    "integration_test_key",
			Name:            "Integration Test Knowledge",
			KnowledgeValue:  []string{"test_value_1", "test_value_2"},
			AssociateTables: []string{"test_table"},
			ExplanationType: "synonym",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/workspaces/%s/nl2sql-knowledge", workspaceID), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		data, code, _ := parseAPIResponse(t, w.Body.Bytes())
		assert.Equal(t, int32(0), code)
		var response catalogpb.Nl2SqlKnowledge
		require.NoError(t, json.Unmarshal(data, &response))
		assert.Greater(t, response.Id, int64(0))
		assert.Equal(t, "glossary", response.KnowledgeType)
		assert.Equal(t, "integration_test_key", response.KnowledgeKey)
	})

	// Test 2: List NL2SQL Knowledge
	t.Run("List", func(t *testing.T) {
		reqBody := catalogpb.ListNl2SqlKnowledgeRequest{
			KnowledgeType:    "glossary",
			KnowledgeBaseIds: []int64{knowledgeBaseID},
			PageSize:         10,
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/workspaces/%s/nl2sql-knowledge/list", workspaceID), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		data, code, _ := parseAPIResponse(t, w.Body.Bytes())
		assert.Equal(t, int32(0), code)
		var response catalogpb.ListNl2SqlKnowledgeResponse
		require.NoError(t, json.Unmarshal(data, &response))
		assert.Greater(t, response.Total, int64(0))
		assert.NotEmpty(t, response.Items)
	})

	// Test 3: Update NL2SQL Knowledge
	t.Run("Update", func(t *testing.T) {
		// 先 list 获取一个 ID
		listReq := catalogpb.ListNl2SqlKnowledgeRequest{
			KnowledgeType:    "glossary",
			KnowledgeBaseIds: []int64{knowledgeBaseID},
			PageSize:         1,
		}
		listBody, _ := json.Marshal(listReq)
		listHttpReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/workspaces/%s/nl2sql-knowledge/list", workspaceID), bytes.NewReader(listBody))
		listHttpReq.Header.Set("Content-Type", "application/json")
		listW := httptest.NewRecorder()
		router.ServeHTTP(listW, listHttpReq)

		data, _, _ := parseAPIResponse(t, listW.Body.Bytes())
		var listResponse catalogpb.ListNl2SqlKnowledgeResponse
		require.NoError(t, json.Unmarshal(data, &listResponse))
		require.NotEmpty(t, listResponse.Items)

		knowledgeID := listResponse.Items[0].Id

		updateReq := catalogpb.UpdateNl2SqlKnowledgeRequest{
			KnowledgeBaseId: knowledgeBaseID,
			KnowledgeType:   "glossary",
			KnowledgeKey:    "updated_integration_test_key",
			Name:            "Updated Integration Test Knowledge",
			KnowledgeValue:  []string{"updated_value"},
		}

		updateBody, _ := json.Marshal(updateReq)
		updateHttpReq := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/workspaces/%s/nl2sql-knowledge/%d", workspaceID, knowledgeID), bytes.NewReader(updateBody))
		updateHttpReq.Header.Set("Content-Type", "application/json")
		updateW := httptest.NewRecorder()

		router.ServeHTTP(updateW, updateHttpReq)

		assert.Equal(t, http.StatusOK, updateW.Code)
	})

	// Test 4: Delete NL2SQL Knowledge
	t.Run("Delete", func(t *testing.T) {
		listReq := catalogpb.ListNl2SqlKnowledgeRequest{
			KnowledgeType:    "glossary",
			KnowledgeBaseIds: []int64{knowledgeBaseID},
			PageSize:         1,
		}
		listBody, _ := json.Marshal(listReq)
		listHttpReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/workspaces/%s/nl2sql-knowledge/list", workspaceID), bytes.NewReader(listBody))
		listHttpReq.Header.Set("Content-Type", "application/json")
		listW := httptest.NewRecorder()
		router.ServeHTTP(listW, listHttpReq)

		data, _, _ := parseAPIResponse(t, listW.Body.Bytes())
		var listResponse catalogpb.ListNl2SqlKnowledgeResponse
		require.NoError(t, json.Unmarshal(data, &listResponse))
		require.NotEmpty(t, listResponse.Items)

		knowledgeID := listResponse.Items[0].Id

		deleteHttpReq := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/workspaces/%s/nl2sql-knowledge/%d", workspaceID, knowledgeID), nil)
		deleteW := httptest.NewRecorder()

		router.ServeHTTP(deleteW, deleteHttpReq)

		assert.Equal(t, http.StatusOK, deleteW.Code)
	})

	// Test 5: Create with missing required fields
	t.Run("Create_MissingRequiredFields", func(t *testing.T) {
		reqBody := catalogpb.CreateNl2SqlKnowledgeRequest{
			KnowledgeBaseId: 0,
			KnowledgeType:   "",
			KnowledgeKey:    "",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/workspaces/%s/nl2sql-knowledge", workspaceID), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}
