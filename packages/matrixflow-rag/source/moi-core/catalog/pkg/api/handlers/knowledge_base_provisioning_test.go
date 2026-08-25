package handlers

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	ginctx "github.com/matrixflow/moi-core/catalog/pkg/api"
	"github.com/matrixflow/moi-core/catalog/pkg/iamcore"
	"github.com/matrixflow/moi-core/model/internalservice"
)

func TestKnowledgeBaseProvisioningDecisionRequiresSemanticModelCreateFact(t *testing.T) {
	tests := []struct {
		name             string
		facts            []internalservice.AuthorizedActionFact
		defaultCatalogID string
		want             bool
	}{
		{
			name:             "semantic model create",
			defaultCatalogID: "7",
			facts: []internalservice.AuthorizedActionFact{{
				ActionID: iamcore.IAMActionSemanticModelCreate, ResourceType: iamcore.IAMResourceWorkspace, ResourceID: "ws-1",
			}},
			want: true,
		},
		{
			name:             "database create is not a substitute",
			defaultCatalogID: "7",
			facts: []internalservice.AuthorizedActionFact{{
				ActionID: iamcore.IAMActionDatabaseCreate, ResourceType: iamcore.IAMResourceCatalog, ResourceID: "7",
			}},
		},
		{name: "missing default catalog id", facts: []internalservice.AuthorizedActionFact{{
			ActionID: iamcore.IAMActionSemanticModelCreate, ResourceType: iamcore.IAMResourceWorkspace, ResourceID: "ws-1",
		}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest("POST", "/", nil)
			c.Request.Header.Set(internalservice.HeaderKnowledgeBaseProvisioning, "true")
			if tt.defaultCatalogID != "" {
				c.Request.Header.Set(internalservice.HeaderKnowledgeBaseProvisioningCatalogID, tt.defaultCatalogID)
			}
			ginctx.SetAuthenticatedBackendExecution(c, ginctx.BackendExecutionContext{
				WorkspaceID: "ws-1", WorkspaceAccessVerified: true, BusinessActionAuthorized: true,
				VerifiedEffectiveRoleID: "role-1", AuthorizedActionFacts: tt.facts,
			})

			decision, catalogID, ok := knowledgeBaseProvisioningDecision(c, "ws-1")
			if ok != tt.want {
				t.Fatalf("ok = %v, want %v", ok, tt.want)
			}
			if tt.want && (decision.VerifiedEffectiveRoleID != "role-1" || !decision.Allow || catalogID != 7) {
				t.Fatalf("decision = %+v, want verified role and allow", decision)
			}
		})
	}
}

func TestKnowledgeBaseProvisioningDecisionRejectsWorkspaceMismatch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/", nil)
	c.Request.Header.Set(internalservice.HeaderKnowledgeBaseProvisioning, "true")
	c.Request.Header.Set(internalservice.HeaderKnowledgeBaseProvisioningCatalogID, "7")
	ginctx.SetAuthenticatedBackendExecution(c, ginctx.BackendExecutionContext{
		WorkspaceID: "ws-other", WorkspaceAccessVerified: true, BusinessActionAuthorized: true,
		VerifiedEffectiveRoleID: "role-1",
		AuthorizedActionFacts: []internalservice.AuthorizedActionFact{{
			ActionID: iamcore.IAMActionSemanticModelCreate, ResourceType: iamcore.IAMResourceWorkspace, ResourceID: "ws-other",
		}},
	})
	if _, _, ok := knowledgeBaseProvisioningDecision(c, "ws-1"); ok {
		t.Fatal("workspace mismatch unexpectedly allowed")
	}
}
